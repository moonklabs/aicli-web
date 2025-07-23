package status

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aicli/aicli-web/internal/docker"
	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/interfaces"
)

// Tracker 워크스페이스 상태 추적자
type Tracker struct {
	// 의존성
	workspaceService interfaces.WorkspaceService
	containerManager docker.ContainerManagement
	factoryManager   docker.DockerManager

	// 내부 상태
	states         sync.Map // workspaceID -> *WorkspaceState
	eventCallbacks []EventCallback

	// 설정
	syncInterval  time.Duration
	retryInterval time.Duration
	maxRetries    int

	// 제어
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	logger Logger
}

// WorkspaceState 워크스페이스 상태 정보
type WorkspaceState struct {
	// 기본 정보
	WorkspaceID string                 `json:"workspace_id"`
	Name        string                 `json:"name"`
	Status      models.WorkspaceStatus `json:"status"`

	// 컨테이너 상태
	ContainerID    string               `json:"container_id,omitempty"`
	ContainerState docker.ContainerState `json:"container_state,omitempty"`

	// 시간 정보
	LastUpdated     time.Time `json:"last_updated"`
	LastSyncAttempt time.Time `json:"last_sync_attempt"`

	// 상태 메타데이터
	SyncAttempts int                `json:"sync_attempts"`
	LastError    string             `json:"last_error,omitempty"`
	Metrics      *WorkspaceMetrics  `json:"metrics,omitempty"`
}

// WorkspaceMetrics 워크스페이스 메트릭 정보
type WorkspaceMetrics struct {
	// 리소스 사용량
	CPUPercent  float64 `json:"cpu_percent"`
	MemoryUsage int64   `json:"memory_usage"`
	MemoryLimit int64   `json:"memory_limit"`
	NetworkRxMB float64 `json:"network_rx_mb"`
	NetworkTxMB float64 `json:"network_tx_mb"`

	// 타이밍 정보
	Uptime       string    `json:"uptime"`
	LastActivity time.Time `json:"last_activity"`

	// 오류 통계
	ErrorCount    int       `json:"error_count"`
	LastErrorTime time.Time `json:"last_error_time,omitempty"`
}

// EventCallback 상태 변경 이벤트 콜백
type EventCallback func(workspaceID string, oldState, newState *WorkspaceState)

// Logger 로거 인터페이스
type Logger interface {
	Info(msg string, args ...interface{})
	Error(msg string, err error, args ...interface{})
	Debug(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
}

// defaultLogger 기본 로거
type defaultLogger struct{}

func (l *defaultLogger) Info(msg string, args ...interface{}) {
	fmt.Printf("[INFO] "+msg+"\n", args...)
}

func (l *defaultLogger) Error(msg string, err error, args ...interface{}) {
	fmt.Printf("[ERROR] "+msg+": %v\n", append(args, err)...)
}

func (l *defaultLogger) Debug(msg string, args ...interface{}) {
	fmt.Printf("[DEBUG] "+msg+"\n", args...)
}

func (l *defaultLogger) Warn(msg string, args ...interface{}) {
	fmt.Printf("[WARN] "+msg+"\n", args...)
}

// NewTracker 새로운 상태 추적자 생성
func NewTracker(
	workspaceService interfaces.WorkspaceService,
	containerManager docker.ContainerManagement,
	factoryManager docker.DockerManager,
) *Tracker {
	ctx, cancel := context.WithCancel(context.Background())

	return &Tracker{
		workspaceService: workspaceService,
		containerManager: containerManager,
		factoryManager:   factoryManager,
		states:           sync.Map{},
		eventCallbacks:   make([]EventCallback, 0),
		syncInterval:     30 * time.Second,
		retryInterval:    5 * time.Second,
		maxRetries:       3,
		ctx:              ctx,
		cancel:           cancel,
		logger:           &defaultLogger{},
	}
}

// SetLogger 로거 설정
func (t *Tracker) SetLogger(logger Logger) {
	t.logger = logger
}

// SetSyncInterval 동기화 간격 설정
func (t *Tracker) SetSyncInterval(interval time.Duration) {
	t.syncInterval = interval
}

// SetMaxRetries 최대 재시도 횟수 설정
func (t *Tracker) SetMaxRetries(maxRetries int) {
	t.maxRetries = maxRetries
}

// Start 상태 추적 시작
func (t *Tracker) Start() error {
	t.logger.Info("상태 추적자 시작 - 동기화 간격: %v", t.syncInterval)
	
	t.wg.Add(1)
	go t.syncLoop()
	
	return nil
}

// Stop 상태 추적 중지
func (t *Tracker) Stop() error {
	t.logger.Info("상태 추적자 중지 중...")
	
	t.cancel()
	t.wg.Wait()
	
	t.logger.Info("상태 추적자 중지 완료")
	return nil
}

// syncLoop 동기화 메인 루프
func (t *Tracker) syncLoop() {
	defer t.wg.Done()

	ticker := time.NewTicker(t.syncInterval)
	defer ticker.Stop()

	t.logger.Debug("동기화 루프 시작")

	// 초기 동기화
	t.syncAllWorkspaces()

	for {
		select {
		case <-t.ctx.Done():
			t.logger.Debug("동기화 루프 종료")
			return
		case <-ticker.C:
			t.syncAllWorkspaces()
		}
	}
}

// syncAllWorkspaces 모든 워크스페이스 동기화
func (t *Tracker) syncAllWorkspaces() {
	t.logger.Debug("모든 워크스페이스 동기화 시작")

	// DB에서 모든 워크스페이스 조회
	workspaces, err := t.getAllWorkspaces()
	if err != nil {
		t.logger.Error("워크스페이스 조회 실패", err)
		return
	}

	t.logger.Debug("워크스페이스 %d개 동기화 중", len(workspaces))

	for _, workspace := range workspaces {
		t.syncWorkspaceState(workspace.ID)
	}

	// 삭제된 워크스페이스 정리
	t.cleanupDeletedWorkspaces(workspaces)

	t.logger.Debug("모든 워크스페이스 동기화 완료")
}

// syncWorkspaceState 개별 워크스페이스 상태 동기화
func (t *Tracker) syncWorkspaceState(workspaceID string) {
	// 현재 상태 조회
	currentState := t.getCurrentState(workspaceID)

	// DB에서 워크스페이스 정보 조회
	workspace, err := t.workspaceService.GetWorkspace(t.ctx, workspaceID, "")
	if err != nil {
		t.handleSyncError(workspaceID, err)
		return
	}

	// 컨테이너 상태 조회
	containers, err := t.containerManager.ListWorkspaceContainers(t.ctx, workspaceID)
	if err != nil {
		t.handleSyncError(workspaceID, err)
		return
	}

	// 새로운 상태 계산
	newState := t.calculateNewState(workspace, containers)

	// 상태 변경 감지 및 업데이트
	if t.hasStateChanged(currentState, newState) {
		t.updateState(workspaceID, currentState, newState)
	}
}

// getCurrentState 현재 상태 조회
func (t *Tracker) getCurrentState(workspaceID string) *WorkspaceState {
	if state, ok := t.states.Load(workspaceID); ok {
		return state.(*WorkspaceState)
	}
	return nil
}

// calculateNewState 새로운 상태 계산
func (t *Tracker) calculateNewState(workspace *models.Workspace, containers []*docker.WorkspaceContainer) *WorkspaceState {
	state := &WorkspaceState{
		WorkspaceID:     workspace.ID,
		Name:            workspace.Name,
		Status:          workspace.Status,
		LastUpdated:     time.Now(),
		LastSyncAttempt: time.Now(),
	}

	// 컨테이너 상태 반영
	if len(containers) > 0 {
		// 가장 최근 컨테이너 사용
		container := containers[0]
		state.ContainerID = container.GetID()
		state.ContainerState = docker.ContainerState(container.GetState())

		// 워크스페이스 상태를 컨테이너 상태에 따라 업데이트
		state.Status = t.deriveWorkspaceStatus(state.ContainerState)

		// 메트릭 수집
		if stats, err := t.factoryManager.Stats().Collect(t.ctx, container.GetID()); err == nil {
			state.Metrics = t.containerStatsToMetrics(stats, container)
		}
	} else {
		// 컨테이너가 없으면 inactive
		state.Status = models.WorkspaceStatusInactive
	}

	return state
}

// deriveWorkspaceStatus 컨테이너 상태로부터 워크스페이스 상태 도출
func (t *Tracker) deriveWorkspaceStatus(containerState docker.ContainerState) models.WorkspaceStatus {
	switch containerState {
	case docker.ContainerStateRunning:
		return models.WorkspaceStatusActive
	case docker.ContainerStateExited, docker.ContainerStateDead:
		return models.WorkspaceStatusInactive
	case docker.ContainerStatePaused:
		return models.WorkspaceStatusInactive // 일시 중지로 간주
	default:
		return models.WorkspaceStatusInactive
	}
}

// containerStatsToMetrics 컨테이너 통계를 메트릭으로 변환
func (t *Tracker) containerStatsToMetrics(stats *docker.ContainerStats, container docker.WorkspaceContainer) *WorkspaceMetrics {
	var uptime string
	if createdAt := container.GetCreatedAt(); !createdAt.IsZero() {
		uptime = time.Since(createdAt).String()
	}

	return &WorkspaceMetrics{
		CPUPercent:  stats.CPUPercent,
		MemoryUsage: stats.MemoryUsage,
		MemoryLimit: stats.MemoryLimit,
		NetworkRxMB: stats.NetworkRxMB,
		NetworkTxMB: stats.NetworkTxMB,
		Uptime:      uptime,
		LastActivity: time.Now(),
		ErrorCount:   0, // 에러 카운터는 따로 관리
	}
}

// hasStateChanged 상태 변경 여부 확인
func (t *Tracker) hasStateChanged(oldState, newState *WorkspaceState) bool {
	if oldState == nil {
		return true // 처음 생성된 상태
	}

	// 주요 상태 필드 비교
	return oldState.Status != newState.Status ||
		oldState.ContainerID != newState.ContainerID ||
		oldState.ContainerState != newState.ContainerState
}

// updateState 상태 업데이트
func (t *Tracker) updateState(workspaceID string, oldState, newState *WorkspaceState) {
	// 상태 저장
	t.states.Store(workspaceID, newState)

	t.logger.Info("워크스페이스 상태 변경: %s [%v -> %v]", 
		workspaceID, 
		func() string {
			if oldState != nil {
				return string(oldState.Status)
			}
			return "nil"
		}(),
		newState.Status,
	)

	// 이벤트 생성
	event := t.createStateChangeEvent(workspaceID, oldState, newState)
	t.emitEvent(event)

	// 콜백 실행
	for _, callback := range t.eventCallbacks {
		go func(cb EventCallback) {
			defer func() {
				if r := recover(); r != nil {
					t.logger.Error("콜백 실행 중 패닉 발생", fmt.Errorf("%v", r), "workspace_id", workspaceID)
				}
			}()
			cb(workspaceID, oldState, newState)
		}(callback)
	}

	// DB 상태 동기화
	t.syncToDatabase(workspaceID, newState)
}

// syncToDatabase 데이터베이스에 상태 동기화
func (t *Tracker) syncToDatabase(workspaceID string, state *WorkspaceState) {
	if state.Status == models.WorkspaceStatusInactive {
		// inactive 상태일 때만 DB 업데이트 (active는 컨테이너가 관리)
		req := &models.UpdateWorkspaceRequest{
			Status: state.Status,
		}
		if _, err := t.workspaceService.UpdateWorkspace(t.ctx, workspaceID, "", req); err != nil {
			t.logger.Error("DB 상태 동기화 실패", err, "workspace_id", workspaceID)
		}
	}
}

// getAllWorkspaces 모든 워크스페이스 조회
func (t *Tracker) getAllWorkspaces() ([]*models.Workspace, error) {
	workspaces, _, err := t.workspaceService.ListWorkspaces(t.ctx, "", &services.ListOptions{
		Page:    1,
		PerPage: 1000, // 충분히 큰 수
	})
	return workspaces, err
}

// cleanupDeletedWorkspaces 삭제된 워크스페이스 정리
func (t *Tracker) cleanupDeletedWorkspaces(activeWorkspaces []*models.Workspace) {
	// 활성 워크스페이스 ID 맵 생성
	activeIDs := make(map[string]bool)
	for _, workspace := range activeWorkspaces {
		activeIDs[workspace.ID] = true
	}

	// 메모리에서 삭제된 워크스페이스 제거
	t.states.Range(func(key, value interface{}) bool {
		workspaceID := key.(string)
		if !activeIDs[workspaceID] {
			t.states.Delete(workspaceID)
			t.logger.Debug("삭제된 워크스페이스 상태 정리: %s", workspaceID)
		}
		return true
	})
}

// handleSyncError 동기화 오류 처리
func (t *Tracker) handleSyncError(workspaceID string, err error) {
	t.logger.Error("워크스페이스 동기화 오류", err, "workspace_id", workspaceID)

	// 현재 상태 업데이트
	if state, ok := t.states.Load(workspaceID); ok {
		currentState := state.(*WorkspaceState)
		currentState.LastError = err.Error()
		currentState.SyncAttempts++
		currentState.LastSyncAttempt = time.Now()
		t.states.Store(workspaceID, currentState)
	}
}

// GetWorkspaceState 워크스페이스 상태 조회
func (t *Tracker) GetWorkspaceState(workspaceID string) (*WorkspaceState, bool) {
	if state, ok := t.states.Load(workspaceID); ok {
		return state.(*WorkspaceState), true
	}
	return nil, false
}

// GetAllWorkspaceStates 모든 워크스페이스 상태 조회
func (t *Tracker) GetAllWorkspaceStates() map[string]*WorkspaceState {
	result := make(map[string]*WorkspaceState)

	t.states.Range(func(key, value interface{}) bool {
		workspaceID := key.(string)
		state := value.(*WorkspaceState)
		result[workspaceID] = state
		return true
	})

	return result
}

// ForceSync 수동 동기화 트리거
func (t *Tracker) ForceSync(workspaceID string) error {
	if workspaceID == "" {
		// 모든 워크스페이스 동기화
		go t.syncAllWorkspaces()
	} else {
		// 특정 워크스페이스만 동기화
		go t.syncWorkspaceState(workspaceID)
	}
	return nil
}

// OnStateChange 상태 변경 이벤트 리스너 등록
func (t *Tracker) OnStateChange(callback EventCallback) {
	t.eventCallbacks = append(t.eventCallbacks, callback)
}

// GetStats 추적자 통계 조회
func (t *Tracker) GetStats() TrackerStats {
	var totalStates int
	t.states.Range(func(key, value interface{}) bool {
		totalStates++
		return true
	})

	return TrackerStats{
		TotalWorkspaces:     totalStates,
		SyncInterval:        t.syncInterval,
		ActiveCallbacks:     len(t.eventCallbacks),
		LastSyncTime:        time.Now(), // 실제로는 마지막 동기화 시간을 저장해야 함
	}
}

// TrackerStats 추적자 통계
type TrackerStats struct {
	TotalWorkspaces int           `json:"total_workspaces"`
	SyncInterval    time.Duration `json:"sync_interval"`
	ActiveCallbacks int           `json:"active_callbacks"`
	LastSyncTime    time.Time     `json:"last_sync_time"`
}