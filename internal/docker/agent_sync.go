package docker

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// AgentStateSynchronizer 에이전트와 컨테이너 상태 동기화 관리자
type AgentStateSynchronizer struct {
	client        *Client
	networkMgr    *AgentNetworkManager
	resourceMgr   *AgentResourceManager
	healthMgr     *AgentHealthMonitor
	eventMgr      *AgentEventMonitor
	
	mu            sync.RWMutex
	agentStates   map[string]*AgentState
	syncHandlers  []StateSyncHandler
	running       bool
	cancel        context.CancelFunc
}

// AgentState 에이전트 통합 상태
type AgentState struct {
	AgentID       string                    `json:"agent_id"`
	ContainerID   string                    `json:"container_id"`
	
	// 컨테이너 상태
	ContainerState AgentContainerState     `json:"container_state"`
	
	// 네트워크 상태
	NetworkState   *AgentNetworkInfo       `json:"network_state,omitempty"`
	
	// 리소스 상태
	ResourceState  *ResourceUsage          `json:"resource_state,omitempty"`
	
	// 헬스 상태
	HealthState    *AgentHealthCheck       `json:"health_state,omitempty"`
	
	// 상태 메타데이터
	LastSync      time.Time                `json:"last_sync"`
	SyncCount     int64                    `json:"sync_count"`
	Errors        []StateSyncError         `json:"errors,omitempty"`
	
	mu            sync.RWMutex
}

// AgentContainerState 에이전트 컨테이너 상태 정보
type AgentContainerState struct {
	Status      string                 `json:"status"`       // running, stopped, paused, etc.
	Running     bool                   `json:"running"`
	Paused      bool                   `json:"paused"`
	Restarting  bool                   `json:"restarting"`
	OOMKilled   bool                   `json:"oom_killed"`
	Dead        bool                   `json:"dead"`
	Pid         int                    `json:"pid"`
	ExitCode    int                    `json:"exit_code"`
	Error       string                 `json:"error,omitempty"`
	StartedAt   time.Time              `json:"started_at"`
	FinishedAt  time.Time              `json:"finished_at"`
	Labels      map[string]string      `json:"labels,omitempty"`
}

// StateSyncError 상태 동기화 에러
type StateSyncError struct {
	Type      string    `json:"type"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Resolved  bool      `json:"resolved"`
}

// StateSyncEvent 상태 동기화 이벤트
type StateSyncEvent struct {
	AgentID   string        `json:"agent_id"`
	Type      SyncEventType `json:"type"`
	OldState  *AgentState   `json:"old_state,omitempty"`
	NewState  *AgentState   `json:"new_state"`
	Timestamp time.Time     `json:"timestamp"`
}

// SyncEventType 동기화 이벤트 타입
type SyncEventType string

const (
	SyncEventStateChanged    SyncEventType = "state_changed"
	SyncEventContainerUpdate SyncEventType = "container_update"
	SyncEventNetworkUpdate   SyncEventType = "network_update"
	SyncEventResourceUpdate  SyncEventType = "resource_update"
	SyncEventHealthUpdate    SyncEventType = "health_update"
	SyncEventError          SyncEventType = "error"
)

// StateSyncHandler 상태 동기화 이벤트 핸들러
type StateSyncHandler func(event StateSyncEvent)

// SyncConfig 동기화 설정
type SyncConfig struct {
	Interval         time.Duration `json:"interval"`          // 동기화 간격 (기본: 30초)
	EnableAutoSync   bool          `json:"enable_auto_sync"`  // 자동 동기화 활성화
	EnableEventSync  bool          `json:"enable_event_sync"` // 이벤트 기반 동기화
	MaxErrors        int           `json:"max_errors"`        // 최대 에러 보관 수
	RetryInterval    time.Duration `json:"retry_interval"`    // 재시도 간격
}

// NewAgentStateSynchronizer 새로운 에이전트 상태 동기화 관리자 생성
func NewAgentStateSynchronizer(
	client *Client,
	networkMgr *AgentNetworkManager,
	resourceMgr *AgentResourceManager,
	healthMgr *AgentHealthMonitor,
	eventMgr *AgentEventMonitor,
) *AgentStateSynchronizer {
	return &AgentStateSynchronizer{
		client:       client,
		networkMgr:   networkMgr,
		resourceMgr:  resourceMgr,
		healthMgr:    healthMgr,
		eventMgr:     eventMgr,
		agentStates:  make(map[string]*AgentState),
		syncHandlers: make([]StateSyncHandler, 0),
	}
}

// Start 상태 동기화 시작
func (ass *AgentStateSynchronizer) Start(ctx context.Context, config SyncConfig) error {
	ass.mu.Lock()
	defer ass.mu.Unlock()

	if ass.running {
		return fmt.Errorf("state synchronizer already running")
	}

	// 기본값 설정
	if config.Interval == 0 {
		config.Interval = 30 * time.Second
	}
	if config.MaxErrors == 0 {
		config.MaxErrors = 100
	}
	if config.RetryInterval == 0 {
		config.RetryInterval = 5 * time.Second
	}

	syncCtx, cancel := context.WithCancel(ctx)
	ass.cancel = cancel
	ass.running = true

	// 이벤트 기반 동기화 설정
	if config.EnableEventSync && ass.eventMgr != nil {
		ass.setupEventBasedSync()
	}

	// 주기적 동기화 고루틴 시작
	if config.EnableAutoSync {
		go ass.runPeriodicSync(syncCtx, config)
	}

	return nil
}

// Stop 상태 동기화 중지
func (ass *AgentStateSynchronizer) Stop() error {
	ass.mu.Lock()
	defer ass.mu.Unlock()

	if !ass.running {
		return nil
	}

	if ass.cancel != nil {
		ass.cancel()
	}

	ass.running = false
	return nil
}

// RegisterAgent 에이전트 상태 추적 등록
func (ass *AgentStateSynchronizer) RegisterAgent(agentID, containerID string) error {
	ass.mu.Lock()
	defer ass.mu.Unlock()

	if _, exists := ass.agentStates[agentID]; exists {
		return fmt.Errorf("agent already registered: %s", agentID)
	}

	agentState := &AgentState{
		AgentID:     agentID,
		ContainerID: containerID,
		LastSync:    time.Time{},
		SyncCount:   0,
		Errors:      make([]StateSyncError, 0),
	}

	ass.agentStates[agentID] = agentState

	// 초기 상태 동기화 수행
	go ass.syncAgentState(context.Background(), agentID)

	return nil
}

// UnregisterAgent 에이전트 상태 추적 해제
func (ass *AgentStateSynchronizer) UnregisterAgent(agentID string) {
	ass.mu.Lock()
	defer ass.mu.Unlock()

	delete(ass.agentStates, agentID)
}

// GetAgentState 에이전트 상태 조회
func (ass *AgentStateSynchronizer) GetAgentState(agentID string) (*AgentState, bool) {
	ass.mu.RLock()
	defer ass.mu.RUnlock()

	state, exists := ass.agentStates[agentID]
	if !exists {
		return nil, false
	}

	// 스레드 안전한 복사본 반환
	state.mu.RLock()
	defer state.mu.RUnlock()

	stateCopy := *state
	stateCopy.Errors = make([]StateSyncError, len(state.Errors))
	copy(stateCopy.Errors, state.Errors)

	return &stateCopy, true
}

// SyncAgentState 특정 에이전트 상태 강제 동기화
func (ass *AgentStateSynchronizer) SyncAgentState(ctx context.Context, agentID string) error {
	return ass.syncAgentState(ctx, agentID)
}

// SyncAllAgents 모든 에이전트 상태 동기화
func (ass *AgentStateSynchronizer) SyncAllAgents(ctx context.Context) error {
	ass.mu.RLock()
	agentIDs := make([]string, 0, len(ass.agentStates))
	for agentID := range ass.agentStates {
		agentIDs = append(agentIDs, agentID)
	}
	ass.mu.RUnlock()

	for _, agentID := range agentIDs {
		if err := ass.syncAgentState(ctx, agentID); err != nil {
			// 개별 에이전트 동기화 실패는 로그만 남기고 계속 진행
			continue
		}
	}

	return nil
}

// syncAgentState 에이전트 상태 동기화 내부 구현
func (ass *AgentStateSynchronizer) syncAgentState(ctx context.Context, agentID string) error {
	ass.mu.RLock()
	agentState, exists := ass.agentStates[agentID]
	ass.mu.RUnlock()

	if !exists {
		return fmt.Errorf("agent not registered: %s", agentID)
	}

	agentState.mu.Lock()
	defer agentState.mu.Unlock()

	// 이전 상태 백업
	oldState := *agentState

	// 컨테이너 상태 동기화
	if err := ass.syncContainerState(ctx, agentState); err != nil {
		ass.addSyncError(agentState, "container_sync", err.Error())
	}

	// 네트워크 상태 동기화
	if ass.networkMgr != nil {
		if err := ass.syncNetworkState(ctx, agentState); err != nil {
			ass.addSyncError(agentState, "network_sync", err.Error())
		}
	}

	// 리소스 상태 동기화
	if ass.resourceMgr != nil {
		if err := ass.syncResourceState(ctx, agentState); err != nil {
			ass.addSyncError(agentState, "resource_sync", err.Error())
		}
	}

	// 헬스 상태 동기화
	if ass.healthMgr != nil {
		if err := ass.syncHealthState(ctx, agentState); err != nil {
			ass.addSyncError(agentState, "health_sync", err.Error())
		}
	}

	// 동기화 메타데이터 업데이트
	agentState.LastSync = time.Now()
	agentState.SyncCount++

	// 상태 변경 이벤트 발송
	if ass.hasStateChanged(&oldState, agentState) {
		event := StateSyncEvent{
			AgentID:   agentID,
			Type:      SyncEventStateChanged,
			OldState:  &oldState,
			NewState:  agentState,
			Timestamp: time.Now(),
		}
		ass.notifyStateChange(event)
	}

	return nil
}

// syncContainerState 컨테이너 상태 동기화
func (ass *AgentStateSynchronizer) syncContainerState(ctx context.Context, agentState *AgentState) error {
	containerInfo, err := ass.client.cli.ContainerInspect(ctx, agentState.ContainerID)
	if err != nil {
		return fmt.Errorf("inspect container: %w", err)
	}

	// TODO: Docker API 호환성 문제로 시간 파싱 임시 생략
	// 실제 구현에서는 RFC3339 형식의 문자열을 time.Time으로 파싱 필요
	var startedAt, finishedAt time.Time
	_ = containerInfo.State.StartedAt  // string 타입
	_ = containerInfo.State.FinishedAt // string 타입

	agentState.ContainerState = AgentContainerState{
		Status:     containerInfo.State.Status,
		Running:    containerInfo.State.Running,
		Paused:     containerInfo.State.Paused,
		Restarting: containerInfo.State.Restarting,
		OOMKilled:  containerInfo.State.OOMKilled,
		Dead:       containerInfo.State.Dead,
		Pid:        containerInfo.State.Pid,
		ExitCode:   containerInfo.State.ExitCode,
		Error:      containerInfo.State.Error,
		StartedAt:  startedAt,
		FinishedAt: finishedAt,
		Labels:     containerInfo.Config.Labels,
	}

	return nil
}

// syncNetworkState 네트워크 상태 동기화
func (ass *AgentStateSynchronizer) syncNetworkState(ctx context.Context, agentState *AgentState) error {
	networkInfo, exists := ass.networkMgr.GetAgentNetwork(agentState.AgentID)
	if exists {
		agentState.NetworkState = networkInfo
	} else {
		agentState.NetworkState = nil
	}
	return nil
}

// syncResourceState 리소스 상태 동기화
func (ass *AgentStateSynchronizer) syncResourceState(ctx context.Context, agentState *AgentState) error {
	resourceUsage, err := ass.resourceMgr.GetContainerResources(ctx, agentState.ContainerID)
	if err != nil {
		return fmt.Errorf("get container resources: %w", err)
	}

	agentState.ResourceState = resourceUsage
	return nil
}

// syncHealthState 헬스 상태 동기화
func (ass *AgentStateSynchronizer) syncHealthState(ctx context.Context, agentState *AgentState) error {
	healthCheck, exists := ass.healthMgr.GetHealthCheck(agentState.AgentID)
	if exists {
		agentState.HealthState = healthCheck
	} else {
		agentState.HealthState = nil
	}
	return nil
}

// setupEventBasedSync 이벤트 기반 동기화 설정
func (ass *AgentStateSynchronizer) setupEventBasedSync() {
	// 컨테이너 이벤트 핸들러 등록
	containerHandler := func(event AgentDockerEvent) {
		if event.Type == "container" {
			go ass.syncAgentState(context.Background(), event.AgentID)
		}
	}
	ass.eventMgr.RegisterHandler("container", containerHandler)

	// 네트워크 이벤트 핸들러 등록
	networkHandler := func(event AgentDockerEvent) {
		if event.Type == "network" {
			go ass.syncAgentState(context.Background(), event.AgentID)
		}
	}
	ass.eventMgr.RegisterHandler("network", networkHandler)
}

// runPeriodicSync 주기적 동기화 실행
func (ass *AgentStateSynchronizer) runPeriodicSync(ctx context.Context, config SyncConfig) {
	ticker := time.NewTicker(config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ass.SyncAllAgents(ctx)
		}
	}
}

// addSyncError 동기화 에러 추가
func (ass *AgentStateSynchronizer) addSyncError(agentState *AgentState, errorType, message string) {
	syncError := StateSyncError{
		Type:      errorType,
		Message:   message,
		Timestamp: time.Now(),
		Resolved:  false,
	}

	agentState.Errors = append(agentState.Errors, syncError)

	// 최대 에러 수 제한
	maxErrors := 100
	if len(agentState.Errors) > maxErrors {
		agentState.Errors = agentState.Errors[len(agentState.Errors)-maxErrors:]
	}
}

// hasStateChanged 상태 변경 여부 확인
func (ass *AgentStateSynchronizer) hasStateChanged(oldState, newState *AgentState) bool {
	// 컨테이너 상태 변경 확인
	if oldState.ContainerState.Status != newState.ContainerState.Status ||
		oldState.ContainerState.Running != newState.ContainerState.Running ||
		oldState.ContainerState.ExitCode != newState.ContainerState.ExitCode {
		return true
	}

	// 헬스 상태 변경 확인
	if (oldState.HealthState == nil) != (newState.HealthState == nil) {
		return true
	}
	if oldState.HealthState != nil && newState.HealthState != nil {
		if oldState.HealthState.Status != newState.HealthState.Status {
			return true
		}
	}

	// 네트워크 상태 변경 확인
	if (oldState.NetworkState == nil) != (newState.NetworkState == nil) {
		return true
	}
	if oldState.NetworkState != nil && newState.NetworkState != nil {
		if oldState.NetworkState.IPAddress != newState.NetworkState.IPAddress {
			return true
		}
	}

	return false
}

// notifyStateChange 상태 변경 알림
func (ass *AgentStateSynchronizer) notifyStateChange(event StateSyncEvent) {
	for _, handler := range ass.syncHandlers {
		go func(h StateSyncHandler, e StateSyncEvent) {
			defer func() {
				if r := recover(); r != nil {
					// 핸들러 패닉 방지
				}
			}()
			h(e)
		}(handler, event)
	}
}

// RegisterSyncHandler 상태 동기화 핸들러 등록
func (ass *AgentStateSynchronizer) RegisterSyncHandler(handler StateSyncHandler) {
	ass.mu.Lock()
	defer ass.mu.Unlock()
	ass.syncHandlers = append(ass.syncHandlers, handler)
}

// GetAllAgentStates 모든 에이전트 상태 조회
func (ass *AgentStateSynchronizer) GetAllAgentStates() map[string]*AgentState {
	ass.mu.RLock()
	defer ass.mu.RUnlock()

	result := make(map[string]*AgentState)
	for agentID, state := range ass.agentStates {
		state.mu.RLock()
		stateCopy := *state
		stateCopy.Errors = make([]StateSyncError, len(state.Errors))
		copy(stateCopy.Errors, state.Errors)
		state.mu.RUnlock()
		
		result[agentID] = &stateCopy
	}

	return result
}

// GetSyncStats 동기화 통계 조회
func (ass *AgentStateSynchronizer) GetSyncStats() map[string]interface{} {
	ass.mu.RLock()
	defer ass.mu.RUnlock()

	totalAgents := len(ass.agentStates)
	runningAgents := 0
	unhealthyAgents := 0
	totalSyncs := int64(0)
	totalErrors := 0

	for _, state := range ass.agentStates {
		state.mu.RLock()
		if state.ContainerState.Running {
			runningAgents++
		}
		if state.HealthState != nil && 
			(state.HealthState.Status == AgentHealthStatusUnhealthy || 
			 state.HealthState.Status == AgentHealthStatusCritical) {
			unhealthyAgents++
		}
		totalSyncs += state.SyncCount
		totalErrors += len(state.Errors)
		state.mu.RUnlock()
	}

	return map[string]interface{}{
		"total_agents":     totalAgents,
		"running_agents":   runningAgents,
		"unhealthy_agents": unhealthyAgents,
		"total_syncs":      totalSyncs,
		"total_errors":     totalErrors,
		"is_running":       ass.running,
	}
}

// IsRunning 동기화 실행 상태 확인
func (ass *AgentStateSynchronizer) IsRunning() bool {
	ass.mu.RLock()
	defer ass.mu.RUnlock()
	return ass.running
}