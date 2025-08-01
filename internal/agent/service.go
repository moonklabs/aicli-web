package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aicli/aicli-web/internal/git"
	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/storage"
	"github.com/google/uuid"
)

// agentService는 AgentService 인터페이스의 구현체입니다
type agentService struct {
	storage         storage.Storage
	dockerAdapter   DockerAdapter
	monitoring      MonitoringService
	eventPublisher  EventPublisher
	worktreeManager git.WorktreeManager
	recoveryManager *AgentRecoveryManager

	// 상태 관리를 위한 메모리 캐시
	agentStates map[string]*AgentStatusInfo
	stateMutex  sync.RWMutex

	// 동시성 제어
	operationMutex sync.Mutex
}

// NewAgentService는 새로운 에이전트 서비스를 생성합니다
func NewAgentService(
	storage storage.Storage,
	dockerAdapter DockerAdapter,
	monitoring MonitoringService,
	eventPublisher EventPublisher,
	worktreeManager git.WorktreeManager,
) AgentService {
	service := &agentService{
		storage:         storage,
		dockerAdapter:   dockerAdapter,
		monitoring:      monitoring,
		eventPublisher:  eventPublisher,
		worktreeManager: worktreeManager,
		recoveryManager: NewAgentRecoveryManager(dockerAdapter, worktreeManager, storage),
		agentStates:     make(map[string]*AgentStatusInfo),
	}

	// 백그라운드 정리 작업 시작
	go service.startMaintenanceRoutine()

	return service
}

// CreateAgent는 새로운 에이전트를 생성합니다
func (s *agentService) CreateAgent(ctx context.Context, req CreateAgentRequest) (*models.Agent, error) {
	// 요청 검증
	if req.ProjectID == "" {
		return nil, &ServiceError{
			Code:    ErrCodeInvalidState,
			Message: "프로젝트 ID가 필요합니다",
			Cause:   nil,
		}
	}

	if req.Name == "" {
		return nil, &ServiceError{
			Code:    ErrCodeInvalidState,
			Message: "에이전트 이름이 필요합니다",
			Cause:   nil,
		}
	}

	// UUID 생성
	agentID := uuid.New().String()

	// 에이전트 모델 생성
	agent := &models.Agent{
		ID:          agentID,
		ProjectID:   req.ProjectID,
		Name:        req.Name,
		Type:        req.Type,
		Status:      models.AgentStatusCreated,
		Config:      req.Config,
		Description: req.Description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Version:     1,
	}

	// 트랜잭션 내에서 에이전트 생성
	err := s.storage.Transaction(ctx, func(txCtx context.Context) error {
		// 데이터베이스에 저장
		if err := s.storage.Agent().Create(txCtx, agent); err != nil {
			return fmt.Errorf("에이전트 저장 실패: %w", err)
		}

		// 초기 상태 정보 설정
		s.setAgentState(agentID, &AgentStatusInfo{
			ID:           agentID,
			Status:       models.AgentStatusCreated,
			LastActivity: time.Now(),
			Health: HealthStatus{
				Status:    "unknown",
				LastCheck: time.Now(),
			},
		})

		return nil
	})

	if err != nil {
		return nil, &ServiceError{
			Code:    ErrCodeInternalError,
			Message: "에이전트 생성 실패",
			Cause:   err,
		}
	}

	// 이벤트 발행
	if s.eventPublisher != nil {
		if err := s.eventPublisher.PublishAgentCreated(ctx, agent); err != nil {
			// 이벤트 발행 실패는 로깅만 처리 (에이전트 생성은 성공)
			// TODO: 로깅 시스템 연동
		}
	}

	return agent, nil
}

// GetAgent는 특정 에이전트를 조회합니다
func (s *agentService) GetAgent(ctx context.Context, id string) (*models.Agent, error) {
	if id == "" {
		return nil, &ServiceError{
			Code:    ErrCodeInvalidState,
			Message: "에이전트 ID가 필요합니다",
			Cause:   nil,
		}
	}

	agent, err := s.storage.Agent().GetByID(ctx, id)
	if err != nil {
		if err == storage.ErrNotFound {
			return nil, &ServiceError{
				Code:    ErrCodeAgentNotFound,
				Message: "에이전트를 찾을 수 없습니다",
				Cause:   err,
			}
		}
		return nil, &ServiceError{
			Code:    ErrCodeInternalError,
			Message: "에이전트 조회 실패",
			Cause:   err,
		}
	}

	return agent, nil
}

// GetAgentByProjectID는 프로젝트별 에이전트 목록을 조회합니다
func (s *agentService) GetAgentByProjectID(ctx context.Context, projectID string) ([]*models.Agent, error) {
	if projectID == "" {
		return nil, &ServiceError{
			Code:    ErrCodeInvalidState,
			Message: "프로젝트 ID가 필요합니다",
			Cause:   nil,
		}
	}

	agents, err := s.storage.Agent().GetByProjectID(ctx, projectID)
	if err != nil {
		return nil, &ServiceError{
			Code:    ErrCodeInternalError,
			Message: "에이전트 목록 조회 실패",
			Cause:   err,
		}
	}

	return agents, nil
}

// UpdateAgent는 에이전트를 업데이트합니다
func (s *agentService) UpdateAgent(ctx context.Context, id string, req UpdateAgentRequest) (*models.Agent, error) {
	if id == "" {
		return nil, &ServiceError{
			Code:    ErrCodeInvalidState,
			Message: "에이전트 ID가 필요합니다",
			Cause:   nil,
		}
	}

	// 기존 에이전트 조회
	_, err := s.GetAgent(ctx, id)
	if err != nil {
		return nil, err
	}

	// 업데이트 필드 구성
	updates := make(map[string]interface{})
	updates["updated_at"] = time.Now()

	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Config != nil {
		updates["config"] = *req.Config
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}

	// 트랜잭션 내에서 업데이트
	err = s.storage.Transaction(ctx, func(txCtx context.Context) error {
		return s.storage.Agent().Update(txCtx, id, updates)
	})

	if err != nil {
		return nil, &ServiceError{
			Code:    ErrCodeInternalError,
			Message: "에이전트 업데이트 실패",
			Cause:   err,
		}
	}

	// 업데이트된 에이전트 조회
	updatedAgent, err := s.GetAgent(ctx, id)
	if err != nil {
		return nil, err
	}

	return updatedAgent, nil
}

// DeleteAgent는 에이전트를 삭제합니다
func (s *agentService) DeleteAgent(ctx context.Context, id string) error {
	if id == "" {
		return &ServiceError{
			Code:    ErrCodeInvalidState,
			Message: "에이전트 ID가 필요합니다",
			Cause:   nil,
		}
	}

	// 에이전트 조회
	agent, err := s.GetAgent(ctx, id)
	if err != nil {
		return err
	}

	// 실행 중인 에이전트는 먼저 중지
	if agent.Status == models.AgentStatusRunning {
		if err := s.StopAgent(ctx, id); err != nil {
			return fmt.Errorf("에이전트 중지 실패: %w", err)
		}
	}

	// 트랜잭션 내에서 삭제
	err = s.storage.Transaction(ctx, func(txCtx context.Context) error {
		// 컨테이너 정리
		if agent.ContainerID != "" {
			if err := s.dockerAdapter.RemoveContainer(txCtx, agent.ContainerID); err != nil {
				// 컨테이너 삭제 실패는 로깅만 처리
				// TODO: 로깅 시스템 연동
			}
		}

		// 워크트리 정리
		if agent.WorktreeID != "" {
			// TODO: 워크트리 정리 로직 구현
		}

		// 데이터베이스에서 삭제
		return s.storage.Agent().Delete(txCtx, id)
	})

	if err != nil {
		return &ServiceError{
			Code:    ErrCodeInternalError,
			Message: "에이전트 삭제 실패",
			Cause:   err,
		}
	}

	// 상태 캐시에서 제거
	s.removeAgentState(id)

	// 이벤트 발행
	if s.eventPublisher != nil {
		if err := s.eventPublisher.PublishAgentDeleted(ctx, id); err != nil {
			// 이벤트 발행 실패는 로깅만 처리
			// TODO: 로깅 시스템 연동
		}
	}

	return nil
}

// StartAgent는 에이전트를 시작합니다
func (s *agentService) StartAgent(ctx context.Context, id string) error {
	s.operationMutex.Lock()
	defer s.operationMutex.Unlock()

	// 에이전트 조회
	agent, err := s.GetAgent(ctx, id)
	if err != nil {
		return err
	}

	// 상태 검증
	if agent.Status == models.AgentStatusRunning {
		return &ServiceError{
			Code:    ErrCodeInvalidState,
			Message: "에이전트가 이미 실행 중입니다",
			Cause:   nil,
		}
	}

	// 상태 변경: Starting
	if err := s.updateAgentStatus(ctx, id, models.AgentStatusStarting, ""); err != nil {
		return err
	}

	// 이벤트 발행
	if s.eventPublisher != nil {
		s.eventPublisher.PublishAgentStarted(ctx, agent)
	}

	// 비동기로 시작 프로세스 실행 (복구 시스템과 함께)
	go s.startAgentProcessWithRecovery(context.Background(), agent)

	return nil
}

// StopAgent는 에이전트를 중지합니다
func (s *agentService) StopAgent(ctx context.Context, id string) error {
	s.operationMutex.Lock()
	defer s.operationMutex.Unlock()

	// 에이전트 조회
	agent, err := s.GetAgent(ctx, id)
	if err != nil {
		return err
	}

	// 상태 검증
	if agent.Status == models.AgentStatusStopped {
		return nil // 이미 중지됨
	}

	// 상태 변경: Stopping
	if err := s.updateAgentStatus(ctx, id, models.AgentStatusStopping, ""); err != nil {
		return err
	}

	// 컨테이너 중지
	if agent.ContainerID != "" {
		if err := s.dockerAdapter.StopContainer(ctx, agent.ContainerID); err != nil {
			return &ServiceError{
				Code:    ErrCodeContainerError,
				Message: "컨테이너 중지 실패",
				Cause:   err,
			}
		}
	}

	// 상태 변경: Stopped
	if err := s.updateAgentStatus(ctx, id, models.AgentStatusStopped, ""); err != nil {
		return err
	}

	// 이벤트 발행
	if s.eventPublisher != nil {
		s.eventPublisher.PublishAgentStopped(ctx, agent)
	}

	return nil
}

// RestartAgent는 에이전트를 재시작합니다
func (s *agentService) RestartAgent(ctx context.Context, id string) error {
	// 먼저 중지
	if err := s.StopAgent(ctx, id); err != nil {
		return fmt.Errorf("에이전트 중지 실패: %w", err)
	}

	// 잠시 대기 (컨테이너가 완전히 중지될 때까지)
	time.Sleep(2 * time.Second)

	// 다시 시작
	return s.StartAgent(ctx, id)
}

// GetAgentStatus는 에이전트 상태를 조회합니다
func (s *agentService) GetAgentStatus(ctx context.Context, id string) (AgentStatusInfo, error) {
	s.stateMutex.RLock()
	state, exists := s.agentStates[id]
	s.stateMutex.RUnlock()

	if !exists {
		// 캐시에 없으면 데이터베이스에서 조회
		agent, err := s.GetAgent(ctx, id)
		if err != nil {
			return AgentStatusInfo{}, err
		}

		state = &AgentStatusInfo{
			ID:           agent.ID,
			Status:       agent.Status,
			ContainerID:  agent.ContainerID,
			WorktreeID:   agent.WorktreeID,
			LastActivity: agent.LastActivity,
			ErrorMessage: agent.ErrorMessage,
			Health: HealthStatus{
				Status:    "unknown",
				LastCheck: time.Now(),
			},
		}

		// 캐시에 저장
		s.setAgentState(id, state)
	}

	return *state, nil
}

// StartMultipleAgents는 여러 에이전트를 동시에 시작합니다
func (s *agentService) StartMultipleAgents(ctx context.Context, ids []string) ([]AgentOperationResult, error) {
	results := make([]AgentOperationResult, len(ids))

	// 병렬 처리를 위한 고루틴
	var wg sync.WaitGroup
	for i, id := range ids {
		wg.Add(1)
		go func(index int, agentID string) {
			defer wg.Done()

			err := s.StartAgent(ctx, agentID)
			results[index] = AgentOperationResult{
				AgentID: agentID,
				Success: err == nil,
			}
			if err != nil {
				results[index].Error = err.Error()
			}
		}(i, id)
	}

	wg.Wait()
	return results, nil
}

// StopMultipleAgents는 여러 에이전트를 동시에 중지합니다
func (s *agentService) StopMultipleAgents(ctx context.Context, ids []string) ([]AgentOperationResult, error) {
	results := make([]AgentOperationResult, len(ids))

	// 병렬 처리를 위한 고루틴
	var wg sync.WaitGroup
	for i, id := range ids {
		wg.Add(1)
		go func(index int, agentID string) {
			defer wg.Done()

			err := s.StopAgent(ctx, agentID)
			results[index] = AgentOperationResult{
				AgentID: agentID,
				Success: err == nil,
			}
			if err != nil {
				results[index].Error = err.Error()
			}
		}(i, id)
	}

	wg.Wait()
	return results, nil
}

// ListActiveAgents는 활성 에이전트 목록을 조회합니다
func (s *agentService) ListActiveAgents(ctx context.Context) ([]*models.Agent, error) {
	agents, err := s.storage.Agent().GetByStatus(ctx, models.AgentStatusRunning)
	if err != nil {
		return nil, &ServiceError{
			Code:    ErrCodeInternalError,
			Message: "활성 에이전트 목록 조회 실패",
			Cause:   err,
		}
	}

	return agents, nil
}

// GetHealthStatus는 에이전트 헬스 상태를 조회합니다
func (s *agentService) GetHealthStatus(ctx context.Context, id string) (HealthStatus, error) {
	agent, err := s.GetAgent(ctx, id)
	if err != nil {
		return HealthStatus{}, err
	}

	if s.monitoring != nil {
		return s.monitoring.CheckAgentHealth(ctx, agent)
	}

	// 모니터링 서비스가 없으면 기본 헬스 상태 반환
	return HealthStatus{
		Status:    "unknown",
		LastCheck: time.Now(),
	}, nil
}

// GetAgentMetrics는 에이전트 메트릭을 조회합니다
func (s *agentService) GetAgentMetrics(ctx context.Context, id string) (AgentMetrics, error) {
	agent, err := s.GetAgent(ctx, id)
	if err != nil {
		return AgentMetrics{}, err
	}

	if s.monitoring != nil {
		return s.monitoring.CollectAgentMetrics(ctx, agent)
	}

	// 모니터링 서비스가 없으면 기본 메트릭 반환
	return AgentMetrics{
		AgentID:   id,
		Timestamp: time.Now(),
	}, nil
}

// CleanupStaleAgents는 오래된 에이전트를 정리합니다
func (s *agentService) CleanupStaleAgents(ctx context.Context) (int, error) {
	// 24시간 이상 비활성 상태인 에이전트 조회
	staleThreshold := time.Now().Add(-24 * time.Hour)

	// TODO: storage에 GetStaleAgents 메서드 추가 필요
	// agents, err := s.storage.Agent().GetStaleAgents(ctx, staleThreshold)

	// 임시로 모든 에이전트를 조회하여 필터링
	allAgents, err := s.storage.Agent().GetAll(ctx)
	if err != nil {
		return 0, &ServiceError{
			Code:    ErrCodeInternalError,
			Message: "에이전트 조회 실패",
			Cause:   err,
		}
	}

	cleanupCount := 0
	for _, agent := range allAgents {
		// 비활성 상태이고 오래된 에이전트만 정리
		if (agent.Status == models.AgentStatusStopped || agent.Status == models.AgentStatusError) &&
			agent.LastActivity.Before(staleThreshold) {

			if err := s.DeleteAgent(ctx, agent.ID); err != nil {
				// 개별 삭제 실패는 로깅만 처리
				continue
			}
			cleanupCount++
		}
	}

	return cleanupCount, nil
}

// PerformMaintenance는 정기 maintenance 작업을 수행합니다
func (s *agentService) PerformMaintenance(ctx context.Context) error {
	// 오래된 에이전트 정리
	_, err := s.CleanupStaleAgents(ctx)
	if err != nil {
		return err
	}

	// 상태 캐시 정리
	s.cleanupStateCache()

	return nil
}

// 내부 헬퍼 메서드들

// updateAgentStatus는 에이전트 상태를 업데이트합니다
func (s *agentService) updateAgentStatus(ctx context.Context, id string, status models.AgentStatus, errorMessage string) error {
	updates := map[string]interface{}{
		"status":        status,
		"last_activity": time.Now(),
		"updated_at":    time.Now(),
	}

	if errorMessage != "" {
		updates["error_message"] = errorMessage
	}

	err := s.storage.Agent().Update(ctx, id, updates)
	if err != nil {
		return &ServiceError{
			Code:    ErrCodeInternalError,
			Message: "에이전트 상태 업데이트 실패",
			Cause:   err,
		}
	}

	// 상태 캐시 업데이트
	s.stateMutex.Lock()
	if state, exists := s.agentStates[id]; exists {
		state.Status = status
		state.LastActivity = time.Now()
		state.ErrorMessage = errorMessage
	}
	s.stateMutex.Unlock()

	return nil
}

// setAgentState는 에이전트 상태를 캐시에 설정합니다
func (s *agentService) setAgentState(id string, state *AgentStatusInfo) {
	s.stateMutex.Lock()
	defer s.stateMutex.Unlock()
	s.agentStates[id] = state
}

// removeAgentState는 에이전트 상태를 캐시에서 제거합니다
func (s *agentService) removeAgentState(id string) {
	s.stateMutex.Lock()
	defer s.stateMutex.Unlock()
	delete(s.agentStates, id)
}

// cleanupStateCache는 오래된 상태 캐시를 정리합니다
func (s *agentService) cleanupStateCache() {
	s.stateMutex.Lock()
	defer s.stateMutex.Unlock()

	staleThreshold := time.Now().Add(-1 * time.Hour)
	for id, state := range s.agentStates {
		if state.LastActivity.Before(staleThreshold) {
			delete(s.agentStates, id)
		}
	}
}

// startAgentProcess는 에이전트 시작 프로세스를 비동기로 실행합니다
// startAgentProcessWithRecovery 복구 시스템과 함께 에이전트 프로세스 시작
func (s *agentService) startAgentProcessWithRecovery(ctx context.Context, agent *models.Agent) {
	// 복구와 함께 시작 프로세스 실행
	err := s.recoveryManager.RecoverWithRetry(ctx, agent, func() error {
		return s.startAgentProcessInternal(ctx, agent)
	})

	if err != nil {
		// 복구 실패 시 에러 상태로 설정
		s.updateAgentStatus(ctx, agent.ID, models.AgentStatusError, fmt.Sprintf("Failed to start after recovery attempts: %v", err))

		// 에러 이벤트 발행
		if s.eventPublisher != nil {
			s.eventPublisher.PublishAgentError(ctx, agent, err)
		}
	}
}

// startAgentProcessInternal 실제 에이전트 프로세스 시작 로직
func (s *agentService) startAgentProcessInternal(ctx context.Context, agent *models.Agent) error {

	// 1. Docker 컨테이너 생성
	containerConfig := ContainerConfig{
		Image:       getImageForAgentType(agent.Type),
		Environment: agent.Config.Environment,
		WorkingDir:  agent.Config.WorkingDir,
		MemoryLimit: agent.Config.MemoryLimit,
		CPULimit:    agent.Config.CPULimit,
		Labels: map[string]string{
			"agent.id":   agent.ID,
			"agent.name": agent.Name,
			"agent.type": string(agent.Type),
			"project.id": agent.ProjectID,
		},
	}

	containerInfo, err := s.dockerAdapter.CreateContainer(ctx, containerConfig)
	if err != nil {
		return fmt.Errorf("컨테이너 생성 실패: %w", err)
	}

	// 컨테이너 ID 업데이트
	updates := map[string]interface{}{
		"container_id": containerInfo.ID,
		"updated_at":   time.Now(),
	}
	if updateErr := s.storage.Agent().Update(ctx, agent.ID, updates); updateErr != nil {
		return fmt.Errorf("컨테이너 ID 업데이트 실패: %w", updateErr)
	}

	// 2. 컨테이너 시작
	if err = s.dockerAdapter.StartContainer(ctx, containerInfo.ID); err != nil {
		return fmt.Errorf("컨테이너 시작 실패: %w", err)
	}

	// 3. 헬스체크 대기
	if err = s.waitForAgentHealthy(ctx, agent.ID, containerInfo.ID, 30*time.Second); err != nil {
		return fmt.Errorf("헬스체크 실패: %w", err)
	}

	// 4. 상태 변경: Running
	if err = s.updateAgentStatus(ctx, agent.ID, models.AgentStatusRunning, ""); err != nil {
		return err
	}

	// 5. 모니터링 시작
	if s.monitoring != nil {
		if monErr := s.monitoring.StartHealthMonitoring(ctx, agent); monErr != nil {
			// 모니터링 시작 실패는 로깅만 처리
			// TODO: 로깅 시스템 연동
		}
	}

	return nil
}

// waitForAgentHealthy는 에이전트가 정상 상태가 될 때까지 대기합니다
func (s *agentService) waitForAgentHealthy(ctx context.Context, agentID, containerID string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// 컨테이너 헬스체크
		health, err := s.dockerAdapter.GetContainerHealth(ctx, containerID)
		if err != nil {
			time.Sleep(2 * time.Second)
			continue
		}

		if health.Status == "healthy" {
			return nil
		}

		time.Sleep(2 * time.Second)
	}

	return fmt.Errorf("헬스체크 타임아웃")
}

// startMaintenanceRoutine는 정기 maintenance 작업을 시작합니다
func (s *agentService) startMaintenanceRoutine() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		ctx := context.Background()
		if err := s.PerformMaintenance(ctx); err != nil {
			// maintenance 실패는 로깅만 처리
			// TODO: 로깅 시스템 연동
		}
	}
}

// getImageForAgentType는 에이전트 타입에 따른 Docker 이미지를 반환합니다
func getImageForAgentType(agentType models.AgentType) string {
	switch agentType {
	case models.AgentTypeClaude:
		return "aicli/claude-agent:latest"
	case models.AgentTypeGemini:
		return "aicli/gemini-agent:latest"
	case models.AgentTypeCustom:
		return "aicli/custom-agent:latest"
	default:
		return "aicli/claude-agent:latest"
	}
}
