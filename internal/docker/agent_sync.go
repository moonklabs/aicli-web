package docker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aicli/aicli-web/internal/models"
)

// AgentSyncState 에이전트 동기화 상태
type AgentSyncState struct {
	AgentID       string                 `json:"agent_id"`
	ContainerID   string                 `json:"container_id"`
	AgentStatus   models.AgentStatus     `json:"agent_status"`
	ContainerStatus ContainerState      `json:"container_status"`
	LastSync      time.Time             `json:"last_sync"`
	SyncFailures  int                   `json:"sync_failures"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// SyncHandler 상태 동기화 핸들러
type SyncHandler func(ctx context.Context, state *AgentSyncState) error

// AgentDockerSync 에이전트와 Docker 컨테이너 상태 동기화 관리자
type AgentDockerSync struct {
	client       *Client
	lifecycle    *LifecycleManager
	syncStates   map[string]*AgentSyncState
	syncHandlers map[string][]SyncHandler
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	syncInterval time.Duration
	maxFailures  int
}

// NewAgentDockerSync 새로운 에이전트-Docker 동기화 관리자 생성
func NewAgentDockerSync(client *Client, lifecycle *LifecycleManager) *AgentDockerSync {
	ctx, cancel := context.WithCancel(context.Background())

	ads := &AgentDockerSync{
		client:       client,
		lifecycle:    lifecycle,
		syncStates:   make(map[string]*AgentSyncState),
		syncHandlers: make(map[string][]SyncHandler),
		ctx:          ctx,
		cancel:       cancel,
		syncInterval: 30 * time.Second,
		maxFailures:  3,
	}

	// 컨테이너 이벤트 구독
	ads.lifecycle.Subscribe("", ads.handleContainerEvent)
	
	// 주기적 동기화 시작
	go ads.startPeriodicSync()

	return ads
}

// RegisterAgent 에이전트 등록 및 동기화 시작
func (ads *AgentDockerSync) RegisterAgent(agentID, containerID string, initialStatus models.AgentStatus) error {
	ads.mu.Lock()
	defer ads.mu.Unlock()

	// 컨테이너 상태 확인
	containerStatus, err := ads.getContainerStatus(ads.ctx, containerID)
	if err != nil {
		return fmt.Errorf("failed to get container status: %w", err)
	}

	ads.syncStates[agentID] = &AgentSyncState{
		AgentID:         agentID,
		ContainerID:     containerID,
		AgentStatus:     initialStatus,
		ContainerStatus: containerStatus,
		LastSync:        time.Now(),
		SyncFailures:    0,
		Metadata:        make(map[string]interface{}),
	}

	return nil
}

// UnregisterAgent 에이전트 등록 해제
func (ads *AgentDockerSync) UnregisterAgent(agentID string) {
	ads.mu.Lock()
	defer ads.mu.Unlock()

	delete(ads.syncStates, agentID)
	delete(ads.syncHandlers, agentID)
}

// UpdateAgentStatus 에이전트 상태 업데이트
func (ads *AgentDockerSync) UpdateAgentStatus(agentID string, status models.AgentStatus) error {
	ads.mu.Lock()
	defer ads.mu.Unlock()

	state, exists := ads.syncStates[agentID]
	if !exists {
		return fmt.Errorf("agent %s not registered", agentID)
	}

	state.AgentStatus = status
	state.LastSync = time.Now()

	// 상태 동기화 실행
	go ads.syncAgentState(agentID)

	return nil
}

// AddSyncHandler 동기화 핸들러 추가
func (ads *AgentDockerSync) AddSyncHandler(agentID string, handler SyncHandler) {
	ads.mu.Lock()
	defer ads.mu.Unlock()

	ads.syncHandlers[agentID] = append(ads.syncHandlers[agentID], handler)
}

// GetSyncState 동기화 상태 조회
func (ads *AgentDockerSync) GetSyncState(agentID string) (*AgentSyncState, bool) {
	ads.mu.RLock()
	defer ads.mu.RUnlock()

	state, exists := ads.syncStates[agentID]
	if !exists {
		return nil, false
	}

	// 복사본 반환
	stateCopy := *state
	return &stateCopy, true
}

// GetAllSyncStates 모든 동기화 상태 조회
func (ads *AgentDockerSync) GetAllSyncStates() map[string]*AgentSyncState {
	ads.mu.RLock()
	defer ads.mu.RUnlock()

	states := make(map[string]*AgentSyncState)
	for agentID, state := range ads.syncStates {
		stateCopy := *state
		states[agentID] = &stateCopy
	}

	return states
}

// Close 동기화 관리자 종료
func (ads *AgentDockerSync) Close() {
	ads.cancel()
}

// handleContainerEvent 컨테이너 이벤트 처리
func (ads *AgentDockerSync) handleContainerEvent(event ContainerEvent) {
	// 에이전트 ID로 상태 찾기
	ads.mu.RLock()
	var targetAgentID string
	for agentID, state := range ads.syncStates {
		if state.ContainerID == event.ContainerID {
			targetAgentID = agentID
			break
		}
	}
	ads.mu.RUnlock()

	if targetAgentID == "" {
		return // 등록된 에이전트가 아님
	}

	// 컨테이너 상태 업데이트
	ads.mu.Lock()
	if state, exists := ads.syncStates[targetAgentID]; exists {
		state.ContainerStatus = event.Status
		state.LastSync = time.Now()
	}
	ads.mu.Unlock()

	// 상태 동기화 실행
	go ads.syncAgentState(targetAgentID)
}

// syncAgentState 특정 에이전트의 상태 동기화
func (ads *AgentDockerSync) syncAgentState(agentID string) {
	ads.mu.RLock()
	state, exists := ads.syncStates[agentID]
	if !exists {
		ads.mu.RUnlock()
		return
	}
	
	// 상태 복사
	stateCopy := *state
	handlers := make([]SyncHandler, len(ads.syncHandlers[agentID]))
	copy(handlers, ads.syncHandlers[agentID])
	ads.mu.RUnlock()

	// 동기화 실행
	err := ads.performSync(&stateCopy)
	
	// 결과 업데이트
	ads.mu.Lock()
	if currentState, exists := ads.syncStates[agentID]; exists {
		if err != nil {
			currentState.SyncFailures++
		} else {
			currentState.SyncFailures = 0
		}
		currentState.LastSync = time.Now()
	}
	ads.mu.Unlock()

	// 핸들러 실행
	for _, handler := range handlers {
		if handler != nil {
			go func(h SyncHandler) {
				defer func() {
					if r := recover(); r != nil {
						// 핸들러 패닉 처리
					}
				}()
				h(ads.ctx, &stateCopy)
			}(handler)
		}
	}
}

// performSync 실제 상태 동기화 수행
func (ads *AgentDockerSync) performSync(state *AgentSyncState) error {
	// 컨테이너 상태 확인
	containerStatus, err := ads.getContainerStatus(ads.ctx, state.ContainerID)
	if err != nil {
		return fmt.Errorf("failed to get container status: %w", err)
	}

	state.ContainerStatus = containerStatus

	// 에이전트와 컨테이너 상태 불일치 해결
	return ads.resolveStatusMismatch(state)
}

// resolveStatusMismatch 상태 불일치 해결
func (ads *AgentDockerSync) resolveStatusMismatch(state *AgentSyncState) error {
	// 에이전트가 실행 중인데 컨테이너가 중지된 경우
	if state.AgentStatus == models.AgentStatusRunning && 
	   (state.ContainerStatus == ContainerStateExited || state.ContainerStatus == ContainerStateStopped) {
		
		// 에이전트 상태를 중지로 변경
		state.AgentStatus = models.AgentStatusStopped
		return nil
	}

	// 에이전트가 중지된 상태인데 컨테이너가 실행 중인 경우
	if state.AgentStatus == models.AgentStatusStopped && 
	   state.ContainerStatus == ContainerStateRunning {
		
		// 에이전트 상태를 실행 중으로 변경
		state.AgentStatus = models.AgentStatusRunning
		return nil
	}

	// 컨테이너가 오류 상태인 경우
	if state.ContainerStatus == ContainerStateErrored {
		state.AgentStatus = models.AgentStatusError
		return nil
	}

	return nil
}

// getContainerStatus 컨테이너 상태 조회
func (ads *AgentDockerSync) getContainerStatus(ctx context.Context, containerID string) (ContainerState, error) {
	inspect, err := ads.client.cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return ContainerStateErrored, err
	}

	switch inspect.State.Status {
	case "running":
		return ContainerStateRunning, nil
	case "exited":
		return ContainerStateExited, nil
	case "paused":
		return ContainerStatePaused, nil
	case "restarting":
		return ContainerStateRestarting, nil
	case "removing":
		return ContainerStateRemoving, nil
	case "dead":
		return ContainerStateErrored, nil
	default:
		return ContainerState(inspect.State.Status), nil
	}
}

// startPeriodicSync 주기적 동기화 시작
func (ads *AgentDockerSync) startPeriodicSync() {
	ticker := time.NewTicker(ads.syncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ads.ctx.Done():
			return
		case <-ticker.C:
			ads.performPeriodicSync()
		}
	}
}

// performPeriodicSync 주기적 동기화 수행
func (ads *AgentDockerSync) performPeriodicSync() {
	ads.mu.RLock()
	agentIDs := make([]string, 0, len(ads.syncStates))
	for agentID := range ads.syncStates {
		agentIDs = append(agentIDs, agentID)
	}
	ads.mu.RUnlock()

	// 각 에이전트에 대해 동기화 실행
	for _, agentID := range agentIDs {
		go ads.syncAgentState(agentID)
	}
}

// GetSyncMetrics 동기화 메트릭 조회
func (ads *AgentDockerSync) GetSyncMetrics() map[string]interface{} {
	ads.mu.RLock()
	defer ads.mu.RUnlock()

	totalAgents := len(ads.syncStates)
	failedAgents := 0
	avgSyncFailures := 0.0

	for _, state := range ads.syncStates {
		if state.SyncFailures >= ads.maxFailures {
			failedAgents++
		}
		avgSyncFailures += float64(state.SyncFailures)
	}

	if totalAgents > 0 {
		avgSyncFailures /= float64(totalAgents)
	}

	return map[string]interface{}{
		"total_agents":       totalAgents,
		"failed_agents":      failedAgents,
		"avg_sync_failures":  avgSyncFailures,
		"success_rate":       float64(totalAgents-failedAgents) / float64(totalAgents) * 100,
		"sync_interval_sec":  ads.syncInterval.Seconds(),
	}
}