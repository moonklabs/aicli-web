package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/aicli/aicli-web/internal/errors"
	"github.com/aicli/aicli-web/internal/git"
	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/storage"
)

// AgentRecoveryStrategy 에이전트 복구 전략 인터페이스
type AgentRecoveryStrategy interface {
	// CanRecover 에이전트 에러가 복구 가능한지 확인
	CanRecover(agent *models.Agent, err error) bool

	// Recover 에이전트 에러 복구 수행
	Recover(ctx context.Context, agent *models.Agent, err error) error

	// Name 복구 전략 이름
	Name() string

	// Priority 복구 전략 우선순위 (낮을수록 먼저 시도)
	Priority() int
}

// ContainerRecoveryStrategy 컨테이너 관련 복구 전략
type ContainerRecoveryStrategy struct {
	dockerAdapter DockerAdapter
	maxRestarts   int
}

// NewContainerRecoveryStrategy 새 컨테이너 복구 전략 생성
func NewContainerRecoveryStrategy(dockerAdapter DockerAdapter) *ContainerRecoveryStrategy {
	return &ContainerRecoveryStrategy{
		dockerAdapter: dockerAdapter,
		maxRestarts:   3,
	}
}

func (s *ContainerRecoveryStrategy) CanRecover(agent *models.Agent, err error) bool {
	// 컨테이너가 있는 에이전트만 복구 가능
	if agent.ContainerID == "" {
		return false
	}

	// ServiceError 타입 확인
	if serviceErr, ok := err.(*ServiceError); ok {
		switch serviceErr.Code {
		case ErrCodeContainerError:
			return true
		case ErrCodeInternalError:
			return true // 내부 오류는 컨테이너 재시작으로 해결될 수 있음
		}
	}

	return false
}

func (s *ContainerRecoveryStrategy) Recover(ctx context.Context, agent *models.Agent, err error) error {
	if s.dockerAdapter == nil {
		return fmt.Errorf("docker adapter not available")
	}

	// 컨테이너 상태 확인
	status, statusErr := s.dockerAdapter.GetContainerStatus(ctx, agent.ContainerID)
	if statusErr != nil {
		return fmt.Errorf("failed to get container status: %w", statusErr)
	}

	// 컨테이너가 실행 중이 아닌 경우 재시작
	if status.Status != "running" {
		if err := s.dockerAdapter.StartContainer(ctx, agent.ContainerID); err != nil {
			// 시작 실패 시 컨테이너 제거 후 재생성 시도
			return s.recreateContainer(ctx, agent)
		}
		return nil
	}

	// 실행 중이지만 문제가 있는 경우 재시작
	if err := s.dockerAdapter.StopContainer(ctx, agent.ContainerID); err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}

	// 잠시 대기 후 재시작
	time.Sleep(2 * time.Second)

	if err := s.dockerAdapter.StartContainer(ctx, agent.ContainerID); err != nil {
		// 재시작 실패 시 재생성 시도
		return s.recreateContainer(ctx, agent)
	}

	return nil
}

func (s *ContainerRecoveryStrategy) recreateContainer(ctx context.Context, agent *models.Agent) error {
	// 기존 컨테이너 제거
	if err := s.dockerAdapter.RemoveContainer(ctx, agent.ContainerID); err != nil {
		// 제거 실패는 무시 (이미 없을 수 있음)
	}

	// 새 컨테이너 생성
	config := ContainerConfig{
		Image:       "claude-agent:latest", // TODO: 에이전트 타입별 이미지 설정
		Environment: make(map[string]string),
		WorkingDir:  "/workspace",
		MemoryLimit: "512m",
		CPULimit:    "0.5",
		Labels: map[string]string{
			"agent.id":         agent.ID,
			"agent.project_id": agent.ProjectID,
			"agent.type":       string(agent.Type),
		},
	}

	// 환경 변수 설정
	if agent.SessionID != "" {
		config.Environment["SESSION_ID"] = agent.SessionID
	}
	if agent.WorktreeID != "" {
		config.Environment["WORKTREE_ID"] = agent.WorktreeID
	}

	// 새 컨테이너 생성
	containerInfo, err := s.dockerAdapter.CreateContainer(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to recreate container: %w", err)
	}

	// 에이전트의 컨테이너 ID 업데이트 (호출자가 처리해야 함)
	agent.ContainerID = containerInfo.ID

	// 컨테이너 시작
	if err := s.dockerAdapter.StartContainer(ctx, containerInfo.ID); err != nil {
		return fmt.Errorf("failed to start recreated container: %w", err)
	}

	return nil
}

func (s *ContainerRecoveryStrategy) Name() string {
	return "ContainerRecovery"
}

func (s *ContainerRecoveryStrategy) Priority() int {
	return 1 // 높은 우선순위
}

// WorktreeRecoveryStrategy 워크트리 관련 복구 전략
type WorktreeRecoveryStrategy struct {
	worktreeManager git.WorktreeManager
}

func NewWorktreeRecoveryStrategy(worktreeManager git.WorktreeManager) *WorktreeRecoveryStrategy {
	return &WorktreeRecoveryStrategy{
		worktreeManager: worktreeManager,
	}
}

func (s *WorktreeRecoveryStrategy) CanRecover(agent *models.Agent, err error) bool {
	// 워크트리가 있는 에이전트만 복구 가능
	if agent.WorktreeID == "" {
		return false
	}

	if serviceErr, ok := err.(*ServiceError); ok {
		return serviceErr.Code == ErrCodeWorktreeError
	}

	return false
}

func (s *WorktreeRecoveryStrategy) Recover(ctx context.Context, agent *models.Agent, err error) error {
	if s.worktreeManager == nil {
		return fmt.Errorf("worktree manager not available")
	}

	// 워크트리 상태 확인 및 복구
	// TODO: 워크트리 매니저 인터페이스에 따라 구현
	return fmt.Errorf("worktree recovery not implemented yet")
}

func (s *WorktreeRecoveryStrategy) Name() string {
	return "WorktreeRecovery"
}

func (s *WorktreeRecoveryStrategy) Priority() int {
	return 2
}

// ResourceLimitRecoveryStrategy 리소스 제한 복구 전략
type ResourceLimitRecoveryStrategy struct {
	dockerAdapter DockerAdapter
}

func NewResourceLimitRecoveryStrategy(dockerAdapter DockerAdapter) *ResourceLimitRecoveryStrategy {
	return &ResourceLimitRecoveryStrategy{
		dockerAdapter: dockerAdapter,
	}
}

func (s *ResourceLimitRecoveryStrategy) CanRecover(agent *models.Agent, err error) bool {
	if serviceErr, ok := err.(*ServiceError); ok {
		return serviceErr.Code == ErrCodeResourceLimit
	}
	return false
}

func (s *ResourceLimitRecoveryStrategy) Recover(ctx context.Context, agent *models.Agent, err error) error {
	// 리소스 사용량 확인
	if agent.ContainerID != "" && s.dockerAdapter != nil {
		metrics, err := s.dockerAdapter.GetContainerMetrics(ctx, agent.ContainerID)
		if err != nil {
			return fmt.Errorf("failed to get container metrics: %w", err)
		}

		// 메모리 사용률이 높은 경우 컨테이너 재시작
		if metrics.Memory.UsagePercent > 90 {
			if err := s.dockerAdapter.StopContainer(ctx, agent.ContainerID); err != nil {
				return fmt.Errorf("failed to stop container: %w", err)
			}

			time.Sleep(2 * time.Second)

			if err := s.dockerAdapter.StartContainer(ctx, agent.ContainerID); err != nil {
				return fmt.Errorf("failed to restart container: %w", err)
			}
		}
	}

	return nil
}

func (s *ResourceLimitRecoveryStrategy) Name() string {
	return "ResourceLimitRecovery"
}

func (s *ResourceLimitRecoveryStrategy) Priority() int {
	return 3
}

// StateCorruptionRecoveryStrategy 상태 손상 복구 전략
type StateCorruptionRecoveryStrategy struct {
	storage storage.Storage
}

func NewStateCorruptionRecoveryStrategy(storage storage.Storage) *StateCorruptionRecoveryStrategy {
	return &StateCorruptionRecoveryStrategy{
		storage: storage,
	}
}

func (s *StateCorruptionRecoveryStrategy) CanRecover(agent *models.Agent, err error) bool {
	if serviceErr, ok := err.(*ServiceError); ok {
		switch serviceErr.Code {
		case ErrCodeInvalidState:
			return true
		case ErrCodeInternalError:
			return true
		}
	}
	return false
}

func (s *StateCorruptionRecoveryStrategy) Recover(ctx context.Context, agent *models.Agent, err error) error {
	// 에이전트 상태를 안전한 상태로 재설정
	updates := map[string]interface{}{
		"status":        models.AgentStatusStopped,
		"error_message": fmt.Sprintf("Recovered from: %v", err),
		"last_activity": time.Now(),
	}

	if err := s.storage.Agent().Update(ctx, agent.ID, updates); err != nil {
		return fmt.Errorf("failed to reset agent state: %w", err)
	}

	return nil
}

func (s *StateCorruptionRecoveryStrategy) Name() string {
	return "StateCorruptionRecovery"
}

func (s *StateCorruptionRecoveryStrategy) Priority() int {
	return 4
}

// AgentRecoveryManager 에이전트 복구 매니저
type AgentRecoveryManager struct {
	strategies  []AgentRecoveryStrategy
	retryPolicy *errors.RetryPolicy
}

// NewAgentRecoveryManager 새 에이전트 복구 매니저 생성
func NewAgentRecoveryManager(dockerAdapter DockerAdapter, worktreeManager git.WorktreeManager, storage storage.Storage) *AgentRecoveryManager {
	strategies := []AgentRecoveryStrategy{
		NewContainerRecoveryStrategy(dockerAdapter),
		NewWorktreeRecoveryStrategy(worktreeManager),
		NewResourceLimitRecoveryStrategy(dockerAdapter),
		NewStateCorruptionRecoveryStrategy(storage),
	}

	// 우선순위별로 정렬
	for i := 0; i < len(strategies)-1; i++ {
		for j := i + 1; j < len(strategies); j++ {
			if strategies[i].Priority() > strategies[j].Priority() {
				strategies[i], strategies[j] = strategies[j], strategies[i]
			}
		}
	}

	return &AgentRecoveryManager{
		strategies: strategies,
		retryPolicy: &errors.RetryPolicy{
			MaxAttempts: 3,
			BaseDelay:   2 * time.Second,
			MaxDelay:    30 * time.Second,
			Multiplier:  2.0,
			Jitter:      true,
			RetryableFunc: func(err error) bool {
				if serviceErr, ok := err.(*ServiceError); ok {
					switch serviceErr.Code {
					case ErrCodeContainerError, ErrCodeWorktreeError, ErrCodeResourceLimit, ErrCodeInternalError:
						return true
					}
				}
				return false
			},
		},
	}
}

// AddStrategy 복구 전략 추가
func (m *AgentRecoveryManager) AddStrategy(strategy AgentRecoveryStrategy) {
	m.strategies = append(m.strategies, strategy)

	// 우선순위별로 재정렬
	for i := 0; i < len(m.strategies)-1; i++ {
		for j := i + 1; j < len(m.strategies); j++ {
			if m.strategies[i].Priority() > m.strategies[j].Priority() {
				m.strategies[i], m.strategies[j] = m.strategies[j], m.strategies[i]
			}
		}
	}
}

// TryRecover 에이전트 복구 시도
func (m *AgentRecoveryManager) TryRecover(ctx context.Context, agent *models.Agent, err error) error {
	for _, strategy := range m.strategies {
		if strategy.CanRecover(agent, err) {
			// 복구 시도
			if recoveryErr := strategy.Recover(ctx, agent, err); recoveryErr == nil {
				return nil // 복구 성공
			}
			// 복구 실패 시 다음 전략 시도
		}
	}

	// 모든 전략 실패
	return fmt.Errorf("all recovery strategies failed for agent %s: %w", agent.ID, err)
}

// RecoverWithRetry 재시도와 함께 복구 수행
func (m *AgentRecoveryManager) RecoverWithRetry(ctx context.Context, agent *models.Agent, operation func() error) error {
	return errors.RetryWithPolicy(ctx, m.retryPolicy, func(ctx context.Context, attempt int) error {
		err := operation()
		if err == nil {
			return nil
		}

		// 첫 번째 시도에서만 복구 시도
		if attempt == 1 {
			if recoveryErr := m.TryRecover(ctx, agent, err); recoveryErr == nil {
				// 복구 성공 후 다시 시도
				return operation()
			}
		}

		return err
	})
}

// GetRecoveryStrategies 등록된 복구 전략 목록 반환
func (m *AgentRecoveryManager) GetRecoveryStrategies() []string {
	strategies := make([]string, len(m.strategies))
	for i, strategy := range m.strategies {
		strategies[i] = strategy.Name()
	}
	return strategies
}

// IsRecoverable 에러가 복구 가능한지 확인
func (m *AgentRecoveryManager) IsRecoverable(agent *models.Agent, err error) bool {
	for _, strategy := range m.strategies {
		if strategy.CanRecover(agent, err) {
			return true
		}
	}
	return false
}
