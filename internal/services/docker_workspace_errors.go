package services

import (
	"context"
	"fmt"
	"strings"
	"time"
	
	"github.com/aicli/aicli-web/internal/docker"
	"github.com/aicli/aicli-web/internal/models"
)

// ErrorRecoveryStrategy 에러 복구 전략
type ErrorRecoveryStrategy struct {
	MaxRetries      int           `json:"max_retries"`
	BackoffDuration time.Duration `json:"backoff_duration"`
	FallbackAction  string        `json:"fallback_action"` // "stop", "remove", "ignore"
}

// WorkspaceErrorHandler 워크스페이스 에러 처리기
type WorkspaceErrorHandler struct {
	service  *DockerWorkspaceService
	strategy *ErrorRecoveryStrategy
}

// NewWorkspaceErrorHandler 새로운 워크스페이스 에러 처리기를 생성합니다
func NewWorkspaceErrorHandler(service *DockerWorkspaceService) *WorkspaceErrorHandler {
	return &WorkspaceErrorHandler{
		service: service,
		strategy: &ErrorRecoveryStrategy{
			MaxRetries:      3,
			BackoffDuration: 5 * time.Second,
			FallbackAction:  "stop",
		},
	}
}

// HandleWorkspaceError 워크스페이스 에러를 처리합니다
func (dws *DockerWorkspaceService) HandleWorkspaceError(workspaceID string, err error, strategy *ErrorRecoveryStrategy) error {
	if strategy == nil {
		strategy = &ErrorRecoveryStrategy{
			MaxRetries:      3,
			BackoffDuration: 5 * time.Second,
			FallbackAction:  "stop",
		}
	}
	
	workspace, getErr := dws.storage.Workspace().GetByID(context.Background(), workspaceID)
	if getErr != nil {
		return fmt.Errorf("get workspace for error handling: %w", getErr)
	}
	
	// 에러 로깅
	dws.logWorkspaceError(workspace, err)
	
	// 에러 유형별 처리
	switch {
	case isDockerError(err):
		return dws.handleDockerError(workspace, err, strategy)
	case isNetworkError(err):
		return dws.handleNetworkError(workspace, err, strategy)
	case isStorageError(err):
		return dws.handleStorageError(workspace, err, strategy)
	default:
		return dws.handleGenericError(workspace, err, strategy)
	}
}

// handleDockerError Docker 특이적 에러를 처리합니다
func (dws *DockerWorkspaceService) handleDockerError(workspace *models.Workspace, err error, strategy *ErrorRecoveryStrategy) error {
	ctx := context.Background()
	
	// Docker 컨테이너 상태 확인
	containers, listErr := dws.dockerManager.Container().ListWorkspaceContainers(ctx, workspace.ID)
	if listErr != nil {
		return fmt.Errorf("list containers for error handling: %w", listErr)
	}
	
	// 컨테이너 상태에 따른 복구 전략
	for _, container := range containers {
		switch container.State {
		case docker.ContainerStateExited:
			// 컨테이너가 종료된 경우 재시작 시도
			return dws.restartWorkspace(workspace.ID, strategy)
		case docker.ContainerStateDead:
			// 컨테이너가 죽은 경우 재생성
			return dws.recreateWorkspace(workspace.ID, strategy)
		// TODO: OOMKilled 상태 처리 추가
		// case docker.ContainerStateOOMKilled:
		// 	// 메모리 부족으로 종료된 경우 리소스 제한 조정 후 재시작
		// 	return dws.recreateWorkspaceWithMoreResources(workspace.ID, strategy)
		}
	}
	
	// 기본 복구 전략 적용
	return dws.applyFallbackAction(workspace.ID, strategy)
}

// handleNetworkError 네트워크 에러를 처리합니다
func (dws *DockerWorkspaceService) handleNetworkError(workspace *models.Workspace, err error, strategy *ErrorRecoveryStrategy) error {
	// 네트워크 연결 재시도
	return dws.retryNetworkOperation(workspace.ID, strategy)
}

// handleStorageError 스토리지 에러를 처리합니다
func (dws *DockerWorkspaceService) handleStorageError(workspace *models.Workspace, err error, strategy *ErrorRecoveryStrategy) error {
	// 스토리지 마운트 재시도
	return dws.retryStorageMount(workspace.ID, strategy)
}

// handleGenericError 일반적인 에러를 처리합니다
func (dws *DockerWorkspaceService) handleGenericError(workspace *models.Workspace, err error, strategy *ErrorRecoveryStrategy) error {
	// 워크스페이스 상태를 에러로 설정
	updates := map[string]interface{}{
		"status":     models.WorkspaceStatusInactive,
		"updated_at": time.Now(),
	}
	
	if updateErr := dws.storage.Workspace().Update(context.Background(), workspace.ID, updates); updateErr != nil {
		return fmt.Errorf("update workspace status after error: %w", updateErr)
	}
	
	return err
}

// restartWorkspace 워크스페이스를 재시작합니다
func (dws *DockerWorkspaceService) restartWorkspace(workspaceID string, strategy *ErrorRecoveryStrategy) error {
	task := &WorkspaceTask{
		Type:        TaskTypeRestart,
		WorkspaceID: workspaceID,
		Timeout:     30 * time.Second,
		Retries:     strategy.MaxRetries,
	}
	
	return dws.retryTask(task, strategy)
}

// recreateWorkspace 워크스페이스를 재생성합니다
func (dws *DockerWorkspaceService) recreateWorkspace(workspaceID string, strategy *ErrorRecoveryStrategy) error {
	// 기존 컨테이너 정리
	cleanupTask := &WorkspaceTask{
		Type:        TaskTypeDelete,
		WorkspaceID: workspaceID,
		Timeout:     30 * time.Second,
		Retries:     strategy.MaxRetries,
	}
	
	if err := dws.retryTask(cleanupTask, strategy); err != nil {
		return fmt.Errorf("cleanup containers: %w", err)
	}
	
	// 새 컨테이너 생성
	createTask := &WorkspaceTask{
		Type:        TaskTypeCreate,
		WorkspaceID: workspaceID,
		Timeout:     60 * time.Second,
		Retries:     strategy.MaxRetries,
	}
	
	return dws.retryTask(createTask, strategy)
}

// recreateWorkspaceWithMoreResources 더 많은 리소스로 워크스페이스를 재생성합니다
func (dws *DockerWorkspaceService) recreateWorkspaceWithMoreResources(workspaceID string, strategy *ErrorRecoveryStrategy) error {
	// TODO: 리소스 제한 증가 로직 구현
	// 현재는 일반적인 재생성과 동일하게 처리
	return dws.recreateWorkspace(workspaceID, strategy)
}

// retryNetworkOperation 네트워크 작업을 재시도합니다
func (dws *DockerWorkspaceService) retryNetworkOperation(workspaceID string, strategy *ErrorRecoveryStrategy) error {
	ctx := context.Background()
	
	for i := 0; i < strategy.MaxRetries; i++ {
		// 네트워크 상태 확인
		if healthy, _ := dws.dockerManager.GetFactory().IsHealthy(ctx); healthy {
			return nil
		}
		
		// 백오프 대기
		if i < strategy.MaxRetries-1 {
			time.Sleep(strategy.BackoffDuration * time.Duration(i+1))
		}
	}
	
	return fmt.Errorf("network operation retry failed after %d attempts", strategy.MaxRetries)
}

// retryStorageMount 스토리지 마운트를 재시도합니다
func (dws *DockerWorkspaceService) retryStorageMount(workspaceID string, strategy *ErrorRecoveryStrategy) error {
	// TODO: 스토리지 마운트 재시도 로직 구현
	return fmt.Errorf("storage mount retry not implemented")
}

// applyFallbackAction 폴백 액션을 적용합니다
func (dws *DockerWorkspaceService) applyFallbackAction(workspaceID string, strategy *ErrorRecoveryStrategy) error {
	switch strategy.FallbackAction {
	case "stop":
		task := &WorkspaceTask{
			Type:        TaskTypeStop,
			WorkspaceID: workspaceID,
			Timeout:     30 * time.Second,
		}
		return dws.executeTask(task)
		
	case "remove":
		task := &WorkspaceTask{
			Type:        TaskTypeDelete,
			WorkspaceID: workspaceID,
			Timeout:     30 * time.Second,
		}
		return dws.executeTask(task)
		
	case "ignore":
		return nil
		
	default:
		return fmt.Errorf("unknown fallback action: %s", strategy.FallbackAction)
	}
}

// retryTask 작업을 재시도합니다
func (dws *DockerWorkspaceService) retryTask(task *WorkspaceTask, strategy *ErrorRecoveryStrategy) error {
	var lastErr error
	
	for i := 0; i <= strategy.MaxRetries; i++ {
		lastErr = dws.executeTask(task)
		if lastErr == nil {
			return nil
		}
		
		// 마지막 시도가 아니면 백오프 대기
		if i < strategy.MaxRetries {
			time.Sleep(strategy.BackoffDuration * time.Duration(i+1))
		}
	}
	
	return fmt.Errorf("task retry failed after %d attempts: %w", strategy.MaxRetries+1, lastErr)
}

// logWorkspaceError 워크스페이스 에러를 로깅합니다
func (dws *DockerWorkspaceService) logWorkspaceError(workspace *models.Workspace, err error) {
	// TODO: 구조화된 로깅 시스템 사용
	fmt.Printf("[ERROR] Workspace %s (%s): %v\n", workspace.ID, workspace.Name, err)
}

// isDockerError Docker 에러인지 확인합니다
func isDockerError(err error) bool {
	if err == nil {
		return false
	}
	
	errStr := err.Error()
	dockerErrorKeywords := []string{
		"docker",
		"container",
		"image",
		"network",
		"volume",
		"daemon",
		"API error",
	}
	
	for _, keyword := range dockerErrorKeywords {
		if strings.Contains(strings.ToLower(errStr), strings.ToLower(keyword)) {
			return true
		}
	}
	
	return false
}

// isNetworkError 네트워크 에러인지 확인합니다
func isNetworkError(err error) bool {
	if err == nil {
		return false
	}
	
	errStr := err.Error()
	networkErrorKeywords := []string{
		"network",
		"connection",
		"timeout",
		"refused",
		"unreachable",
		"dns",
	}
	
	for _, keyword := range networkErrorKeywords {
		if strings.Contains(strings.ToLower(errStr), strings.ToLower(keyword)) {
			return true
		}
	}
	
	return false
}

// isStorageError 스토리지 에러인지 확인합니다
func isStorageError(err error) bool {
	if err == nil {
		return false
	}
	
	errStr := err.Error()
	storageErrorKeywords := []string{
		"storage",
		"mount",
		"volume",
		"disk",
		"filesystem",
		"permission",
		"no space left",
	}
	
	for _, keyword := range storageErrorKeywords {
		if strings.Contains(strings.ToLower(errStr), strings.ToLower(keyword)) {
			return true
		}
	}
	
	return false
}

// GetErrorRecoveryStrategy 워크스페이스 에러 복구 전략을 가져옵니다
func (dws *DockerWorkspaceService) GetErrorRecoveryStrategy(workspaceID string) (*ErrorRecoveryStrategy, error) {
	// TODO: 워크스페이스별 커스텀 복구 전략 지원
	return &ErrorRecoveryStrategy{
		MaxRetries:      3,
		BackoffDuration: 5 * time.Second,
		FallbackAction:  "stop",
	}, nil
}

// SetErrorRecoveryStrategy 워크스페이스 에러 복구 전략을 설정합니다
func (dws *DockerWorkspaceService) SetErrorRecoveryStrategy(workspaceID string, strategy *ErrorRecoveryStrategy) error {
	// TODO: 워크스페이스별 커스텀 복구 전략 저장
	return nil
}