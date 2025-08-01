package docker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
)

// RecoveryPolicy 복구 정책
type RecoveryPolicy struct {
	MaxRetries       int           `json:"max_retries"`
	RetryInterval    time.Duration `json:"retry_interval"`
	HealthCheckDelay time.Duration `json:"health_check_delay"`
	EnableAutoRestart bool         `json:"enable_auto_restart"`
	EnableRecreate   bool          `json:"enable_recreate"`
}

// DefaultRecoveryPolicy 기본 복구 정책
func DefaultRecoveryPolicy() RecoveryPolicy {
	return RecoveryPolicy{
		MaxRetries:       3,
		RetryInterval:    30 * time.Second,
		HealthCheckDelay: 10 * time.Second,
		EnableAutoRestart: true,
		EnableRecreate:   false,
	}
}

// RecoveryAction 복구 액션 타입
type RecoveryAction string

const (
	RecoveryActionRestart  RecoveryAction = "restart"
	RecoveryActionRecreate RecoveryAction = "recreate"
	RecoveryActionStop     RecoveryAction = "stop"
	RecoveryActionNone     RecoveryAction = "none"
)

// RecoveryAttempt 복구 시도 정보
type RecoveryAttempt struct {
	AgentID     string         `json:"agent_id"`
	ContainerID string         `json:"container_id"`
	Action      RecoveryAction `json:"action"`
	Timestamp   time.Time      `json:"timestamp"`
	Success     bool           `json:"success"`
	Error       string         `json:"error,omitempty"`
	Duration    time.Duration  `json:"duration"`
}

// RecoveryState 복구 상태
type RecoveryState struct {
	AgentID           string            `json:"agent_id"`
	ContainerID       string            `json:"container_id"`
	Policy            RecoveryPolicy    `json:"policy"`
	FailureCount      int               `json:"failure_count"`
	LastFailure       time.Time         `json:"last_failure"`
	RecoveryAttempts  []RecoveryAttempt `json:"recovery_attempts"`
	IsRecovering      bool              `json:"is_recovering"`
	LastHealthCheck   time.Time         `json:"last_health_check"`
	ConsecutiveFailures int             `json:"consecutive_failures"`
}

// RecoveryCallback 복구 완료 콜백
type RecoveryCallback func(attempt RecoveryAttempt)

// AutoRecoveryManager 자동 복구 관리자
type AutoRecoveryManager struct {
	client      *Client
	lifecycle   *LifecycleManager
	health      *HealthChecker
	agentSync   *AgentDockerSync
	recoveryStates map[string]*RecoveryState
	callbacks   []RecoveryCallback
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewAutoRecoveryManager 새로운 자동 복구 관리자 생성
func NewAutoRecoveryManager(client *Client, lifecycle *LifecycleManager, health *HealthChecker, agentSync *AgentDockerSync) *AutoRecoveryManager {
	ctx, cancel := context.WithCancel(context.Background())

	arm := &AutoRecoveryManager{
		client:         client,
		lifecycle:      lifecycle,
		health:         health,
		agentSync:      agentSync,
		recoveryStates: make(map[string]*RecoveryState),
		callbacks:      make([]RecoveryCallback, 0),
		ctx:            ctx,
		cancel:         cancel,
	}

	// 컨테이너 이벤트 구독
	lifecycle.Subscribe("", arm.handleContainerEvent)

	// 주기적 헬스체크 시작
	go arm.startHealthCheckMonitor()

	return arm
}

// RegisterAgent 에이전트 등록
func (arm *AutoRecoveryManager) RegisterAgent(agentID, containerID string, policy RecoveryPolicy) {
	arm.mu.Lock()
	defer arm.mu.Unlock()

	arm.recoveryStates[agentID] = &RecoveryState{
		AgentID:             agentID,
		ContainerID:         containerID,
		Policy:              policy,
		FailureCount:        0,
		RecoveryAttempts:    make([]RecoveryAttempt, 0),
		IsRecovering:        false,
		LastHealthCheck:     time.Now(),
		ConsecutiveFailures: 0,
	}
}

// UnregisterAgent 에이전트 등록 해제
func (arm *AutoRecoveryManager) UnregisterAgent(agentID string) {
	arm.mu.Lock()
	defer arm.mu.Unlock()

	delete(arm.recoveryStates, agentID)
}

// AddRecoveryCallback 복구 콜백 추가
func (arm *AutoRecoveryManager) AddRecoveryCallback(callback RecoveryCallback) {
	arm.mu.Lock()
	defer arm.mu.Unlock()

	arm.callbacks = append(arm.callbacks, callback)
}

// GetRecoveryState 복구 상태 조회
func (arm *AutoRecoveryManager) GetRecoveryState(agentID string) (*RecoveryState, bool) {
	arm.mu.RLock()
	defer arm.mu.RUnlock()

	state, exists := arm.recoveryStates[agentID]
	if !exists {
		return nil, false
	}

	// 복사본 반환
	stateCopy := *state
	stateCopy.RecoveryAttempts = make([]RecoveryAttempt, len(state.RecoveryAttempts))
	copy(stateCopy.RecoveryAttempts, state.RecoveryAttempts)

	return &stateCopy, true
}

// GetAllRecoveryStates 모든 복구 상태 조회
func (arm *AutoRecoveryManager) GetAllRecoveryStates() map[string]*RecoveryState {
	arm.mu.RLock()
	defer arm.mu.RUnlock()

	states := make(map[string]*RecoveryState)
	for agentID, state := range arm.recoveryStates {
		stateCopy := *state
		stateCopy.RecoveryAttempts = make([]RecoveryAttempt, len(state.RecoveryAttempts))
		copy(stateCopy.RecoveryAttempts, state.RecoveryAttempts)
		states[agentID] = &stateCopy
	}

	return states
}

// Close 자동 복구 관리자 종료
func (arm *AutoRecoveryManager) Close() {
	arm.cancel()
}

// handleContainerEvent 컨테이너 이벤트 처리
func (arm *AutoRecoveryManager) handleContainerEvent(event ContainerEvent) {
	// 장애 이벤트만 처리
	if !arm.isFailureEvent(event) {
		return
	}

	// 해당 에이전트 찾기
	arm.mu.RLock()
	var targetAgentID string
	for agentID, state := range arm.recoveryStates {
		if state.ContainerID == event.ContainerID {
			targetAgentID = agentID
			break
		}
	}
	arm.mu.RUnlock()

	if targetAgentID == "" {
		return // 등록된 에이전트가 아님
	}

	// 복구 시도
	go arm.attemptRecovery(targetAgentID, event)
}

// isFailureEvent 장애 이벤트인지 확인
func (arm *AutoRecoveryManager) isFailureEvent(event ContainerEvent) bool {
	return event.Type == EventTypeDie || 
		   event.Type == EventTypeDestroy ||
		   event.Status == ContainerStateExited ||
		   event.Status == ContainerStateErrored
}

// attemptRecovery 복구 시도
func (arm *AutoRecoveryManager) attemptRecovery(agentID string, event ContainerEvent) {
	arm.mu.Lock()
	state, exists := arm.recoveryStates[agentID]
	if !exists || state.IsRecovering {
		arm.mu.Unlock()
		return
	}

	state.IsRecovering = true
	state.FailureCount++
	state.ConsecutiveFailures++
	state.LastFailure = time.Now()
	arm.mu.Unlock()

	defer func() {
		arm.mu.Lock()
		if state, exists := arm.recoveryStates[agentID]; exists {
			state.IsRecovering = false
		}
		arm.mu.Unlock()
	}()

	// 복구 정책에 따라 액션 결정
	action := arm.determineRecoveryAction(state)
	if action == RecoveryActionNone {
		return
	}

	// 복구 실행
	attempt := RecoveryAttempt{
		AgentID:     agentID,
		ContainerID: state.ContainerID,
		Action:      action,
		Timestamp:   time.Now(),
	}

	startTime := time.Now()
	err := arm.executeRecoveryAction(action, state)
	attempt.Duration = time.Since(startTime)
	attempt.Success = (err == nil)
	if err != nil {
		attempt.Error = err.Error()
	}

	// 복구 시도 기록
	arm.mu.Lock()
	if currentState, exists := arm.recoveryStates[agentID]; exists {
		currentState.RecoveryAttempts = append(currentState.RecoveryAttempts, attempt)
		
		// 성공 시 연속 실패 카운트 리셋
		if attempt.Success {
			currentState.ConsecutiveFailures = 0
		}
	}
	arm.mu.Unlock()

	// 콜백 실행
	for _, callback := range arm.callbacks {
		go callback(attempt)
	}
}

// determineRecoveryAction 복구 액션 결정
func (arm *AutoRecoveryManager) determineRecoveryAction(state *RecoveryState) RecoveryAction {
	// 최대 재시도 횟수 초과
	if state.ConsecutiveFailures >= state.Policy.MaxRetries {
		return RecoveryActionStop
	}

	// 재시작 정책 확인
	if state.Policy.EnableAutoRestart {
		return RecoveryActionRestart
	}

	// 재생성 정책 확인
	if state.Policy.EnableRecreate && state.ConsecutiveFailures >= 2 {
		return RecoveryActionRecreate
	}

	return RecoveryActionNone
}

// executeRecoveryAction 복구 액션 실행
func (arm *AutoRecoveryManager) executeRecoveryAction(action RecoveryAction, state *RecoveryState) error {
	ctx, cancel := context.WithTimeout(arm.ctx, 60*time.Second)
	defer cancel()

	switch action {
	case RecoveryActionRestart:
		return arm.restartContainer(ctx, state.ContainerID)
	case RecoveryActionRecreate:
		return arm.recreateContainer(ctx, state.AgentID, state.ContainerID)
	case RecoveryActionStop:
		return arm.stopContainer(ctx, state.ContainerID)
	default:
		return nil
	}
}

// restartContainer 컨테이너 재시작
func (arm *AutoRecoveryManager) restartContainer(ctx context.Context, containerID string) error {
	// 재시작 실행
	timeout := 30
	if err := arm.client.cli.ContainerRestart(ctx, containerID, container.StopOptions{
		Timeout: &timeout,
	}); err != nil {
		return fmt.Errorf("failed to restart container: %w", err)
	}

	// 헬스체크 대기
	time.Sleep(5 * time.Second)

	// 실행 상태 확인
	inspect, err := arm.client.cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return fmt.Errorf("failed to inspect container after restart: %w", err)
	}

	if !inspect.State.Running {
		return fmt.Errorf("container not running after restart")
	}

	return nil
}

// recreateContainer 컨테이너 재생성
func (arm *AutoRecoveryManager) recreateContainer(ctx context.Context, agentID, containerID string) error {
	// 현재 컨테이너 정보 백업
	inspect, err := arm.client.cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return fmt.Errorf("failed to inspect container: %w", err)
	}

	// 컨테이너 제거
	if err := arm.client.cli.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{
		Force: true,
	}); err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}

	// 새 컨테이너 생성
	config := inspect.Config
	hostConfig := inspect.HostConfig
	networkConfig := inspect.NetworkSettings

	resp, err := arm.client.cli.ContainerCreate(ctx, config, hostConfig, nil, nil, config.Hostname)
	if err != nil {
		return fmt.Errorf("failed to recreate container: %w", err)
	}

	newContainerID := resp.ID

	// 네트워크 연결
	for networkName := range networkConfig.Networks {
		if err := arm.client.cli.NetworkConnect(ctx, networkName, newContainerID, nil); err != nil {
			// 기본 네트워크 연결 실패는 무시
			continue
		}
	}

	// 컨테이너 시작
	if err := arm.client.cli.ContainerStart(ctx, newContainerID, types.ContainerStartOptions{}); err != nil {
		return fmt.Errorf("failed to start recreated container: %w", err)
	}

	// 복구 상태에서 컨테이너 ID 업데이트
	arm.mu.Lock()
	if state, exists := arm.recoveryStates[agentID]; exists {
		state.ContainerID = newContainerID
	}
	arm.mu.Unlock()

	return nil
}

// stopContainer 컨테이너 중지
func (arm *AutoRecoveryManager) stopContainer(ctx context.Context, containerID string) error {
	timeout := 30
	return arm.client.cli.ContainerStop(ctx, containerID, container.StopOptions{
		Timeout: &timeout,
	})
}

// startHealthCheckMonitor 헬스체크 모니터링 시작
func (arm *AutoRecoveryManager) startHealthCheckMonitor() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-arm.ctx.Done():
			return
		case <-ticker.C:
			arm.performHealthChecks()
		}
	}
}

// performHealthChecks 헬스체크 수행
func (arm *AutoRecoveryManager) performHealthChecks() {
	arm.mu.RLock()
	containerIDs := make([]string, 0, len(arm.recoveryStates))
	for _, state := range arm.recoveryStates {
		if !state.IsRecovering {
			containerIDs = append(containerIDs, state.ContainerID)
		}
	}
	arm.mu.RUnlock()

	// 배치 헬스체크 실행
	healthResults, err := arm.health.CheckMultipleContainers(arm.ctx, containerIDs)
	if err != nil {
		return
	}

	// 결과 처리
	for _, result := range healthResults {
		if !result.Healthy && result.Error != "" {
			// 헬스체크 실패 이벤트 생성
			event := ContainerEvent{
				ContainerID: result.ContainerID,
				Type:        EventTypeHealthcheck,
				Status:      ContainerStateErrored,
				Message:     "Health check failed: " + result.Error,
				Timestamp:   time.Now(),
			}

			go arm.handleContainerEvent(event)
		}
	}

	// 마지막 헬스체크 시간 업데이트
	arm.mu.Lock()
	for _, state := range arm.recoveryStates {
		state.LastHealthCheck = time.Now()
	}
	arm.mu.Unlock()
}

// GetRecoveryMetrics 복구 메트릭 조회
func (arm *AutoRecoveryManager) GetRecoveryMetrics() map[string]interface{} {
	arm.mu.RLock()
	defer arm.mu.RUnlock()

	totalAgents := len(arm.recoveryStates)
	recoveringAgents := 0
	totalAttempts := 0
	successfulAttempts := 0
	avgFailureCount := 0.0

	for _, state := range arm.recoveryStates {
		if state.IsRecovering {
			recoveringAgents++
		}
		
		totalAttempts += len(state.RecoveryAttempts)
		for _, attempt := range state.RecoveryAttempts {
			if attempt.Success {
				successfulAttempts++
			}
		}
		
		avgFailureCount += float64(state.FailureCount)
	}

	if totalAgents > 0 {
		avgFailureCount /= float64(totalAgents)
	}

	successRate := 0.0
	if totalAttempts > 0 {
		successRate = float64(successfulAttempts) / float64(totalAttempts) * 100
	}

	return map[string]interface{}{
		"total_agents":        totalAgents,
		"recovering_agents":   recoveringAgents,
		"total_attempts":      totalAttempts,
		"successful_attempts": successfulAttempts,
		"success_rate":        successRate,
		"avg_failure_count":   avgFailureCount,
	}
}