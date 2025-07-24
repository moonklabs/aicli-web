package status

import (
	"context"
	"testing"
	"time"

	"github.com/aicli/aicli-web/internal/docker"
	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/interfaces"
	"github.com/aicli/aicli-web/internal/errors"
)

// Mock implementations for testing

// MockWorkspaceService Mock 워크스페이스 서비스
type MockWorkspaceService struct {
	workspaces map[string]*models.Workspace
}

func NewMockWorkspaceService() *MockWorkspaceService {
	return &MockWorkspaceService{
		workspaces: make(map[string]*models.Workspace),
	}
}

func (m *MockWorkspaceService) CreateWorkspace(ctx context.Context, req *models.CreateWorkspaceRequest, ownerID string) (*models.Workspace, error) {
	return nil, nil
}

func (m *MockWorkspaceService) GetWorkspace(ctx context.Context, workspaceID, ownerID string) (*models.Workspace, error) {
	if workspace, exists := m.workspaces[workspaceID]; exists {
		return workspace, nil
	}
	return nil, errors.NewWorkspaceError(errors.ErrCodeNotFound, "workspace not found", nil)
}

func (m *MockWorkspaceService) ListWorkspaces(ctx context.Context, ownerID string, req *models.PaginationRequest) (*models.WorkspaceListResponse, error) {
	var result []*models.Workspace
	for _, workspace := range m.workspaces {
		result = append(result, workspace)
	}
	
	return &models.WorkspaceListResponse{
		Success: true,
		Data: func() []models.Workspace {
			workspaces := make([]models.Workspace, len(result))
			for i, ws := range result {
				workspaces[i] = *ws
			}
			return workspaces
		}(),
		Meta: models.PaginationMeta{
			CurrentPage: 1,
			PerPage:     100,
			Total:       len(result),
			TotalPages:  1,
		},
	}, nil
}

func (m *MockWorkspaceService) UpdateWorkspace(ctx context.Context, workspaceID string, req *models.UpdateWorkspaceRequest, ownerID string) (*models.Workspace, error) {
	if workspace, exists := m.workspaces[workspaceID]; exists {
		if req.Status != "" {
			workspace.Status = req.Status
		}
		return workspace, nil
	}
	return nil, errors.NewWorkspaceError(errors.ErrCodeNotFound, "workspace not found", nil)
}

func (m *MockWorkspaceService) DeleteWorkspace(ctx context.Context, workspaceID, ownerID string) error {
	delete(m.workspaces, workspaceID)
	return nil
}

// 누락된 인터페이스 메서드들 추가
func (m *MockWorkspaceService) ValidateWorkspace(ctx context.Context, workspace *models.Workspace) error {
	return nil
}

func (m *MockWorkspaceService) ActivateWorkspace(ctx context.Context, id string, ownerID string) error {
	return nil
}

func (m *MockWorkspaceService) DeactivateWorkspace(ctx context.Context, id string, ownerID string) error {
	return nil
}

func (m *MockWorkspaceService) ArchiveWorkspace(ctx context.Context, id string, ownerID string) error {
	return nil
}

func (m *MockWorkspaceService) UpdateActiveTaskCount(ctx context.Context, id string, delta int) error {
	return nil
}

func (m *MockWorkspaceService) GetWorkspaceStats(ctx context.Context, ownerID string) (*interfaces.WorkspaceStats, error) {
	return &interfaces.WorkspaceStats{
		TotalWorkspaces:  len(m.workspaces),
		ActiveWorkspaces: len(m.workspaces),
		ArchivedWorkspaces: 0,
		TotalActiveTasks: 0,
	}, nil
}

func (m *MockWorkspaceService) AddWorkspace(workspace *models.Workspace) {
	m.workspaces[workspace.ID] = workspace
}

// MockContainerManager Mock 컨테이너 관리자
type MockContainerManager struct {
	containers map[string][]*MockWorkspaceContainer
}

func NewMockContainerManager() *MockContainerManager {
	return &MockContainerManager{
		containers: make(map[string][]*MockWorkspaceContainer),
	}
}

func (m *MockContainerManager) CreateWorkspaceContainer(ctx context.Context, req *docker.CreateContainerRequest) (*docker.WorkspaceContainer, error) {
	return nil, nil
}

func (m *MockContainerManager) InspectContainer(ctx context.Context, containerID string) (*docker.WorkspaceContainer, error) {
	return nil, nil
}

func (m *MockContainerManager) ListWorkspaceContainers(ctx context.Context, workspaceID string) ([]*docker.WorkspaceContainer, error) {
	if containers, exists := m.containers[workspaceID]; exists {
		result := make([]*docker.WorkspaceContainer, len(containers))
		for i, container := range containers {
			var wc docker.WorkspaceContainer = container
			result[i] = &wc
		}
		return result, nil
	}
	return []*docker.WorkspaceContainer{}, nil
}

func (m *MockContainerManager) StartContainer(ctx context.Context, containerID string) error {
	return nil
}

func (m *MockContainerManager) StopContainer(ctx context.Context, containerID string, timeout time.Duration) error {
	return nil
}

func (m *MockContainerManager) RestartContainer(ctx context.Context, containerID string, timeout time.Duration) error {
	return nil
}

func (m *MockContainerManager) RemoveContainer(ctx context.Context, containerID string, force bool) error {
	return nil
}

func (m *MockContainerManager) CleanupWorkspace(ctx context.Context, workspaceID string, force bool) error {
	return nil
}

func (m *MockContainerManager) ListContainers(ctx context.Context) ([]*docker.WorkspaceContainer, error) {
	var allContainers []*docker.WorkspaceContainer
	for _, containers := range m.containers {
		for _, container := range containers {
			var wc docker.WorkspaceContainer = container
			allContainers = append(allContainers, &wc)
		}
	}
	return allContainers, nil
}

func (m *MockContainerManager) AddContainer(workspaceID string, container *MockWorkspaceContainer) {
	if m.containers[workspaceID] == nil {
		m.containers[workspaceID] = make([]*MockWorkspaceContainer, 0)
	}
	m.containers[workspaceID] = append(m.containers[workspaceID], container)
}

// MockWorkspaceContainer Mock 워크스페이스 컨테이너
type MockWorkspaceContainer struct {
	id          string
	name        string
	workspaceID string
	state       string
	createdAt   time.Time
}

func (m *MockWorkspaceContainer) GetID() string {
	return m.id
}

func (m *MockWorkspaceContainer) GetName() string {
	return m.name
}

func (m *MockWorkspaceContainer) GetWorkspaceID() string {
	return m.workspaceID
}

func (m *MockWorkspaceContainer) GetState() string {
	return m.state
}

func (m *MockWorkspaceContainer) GetCreatedAt() time.Time {
	return m.createdAt
}

// MockDockerManager Mock Docker 매니저
type MockDockerManager struct {
	statsCollector *MockStatsCollector
}

func NewMockDockerManager() *MockDockerManager {
	return &MockDockerManager{
		statsCollector: NewMockStatsCollector(),
	}
}

func (m *MockDockerManager) GetFactory() docker.DockerFactory {
	return nil
}

func (m *MockDockerManager) Client() docker.DockerClient {
	return nil
}

func (m *MockDockerManager) Network() docker.NetworkManagement {
	return nil
}

func (m *MockDockerManager) Stats() docker.StatsCollection {
	return m.statsCollector
}

func (m *MockDockerManager) Health() docker.HealthMonitoring {
	return nil
}

func (m *MockDockerManager) Mount() docker.MountManagement {
	return nil
}

func (m *MockDockerManager) Config() *docker.Config {
	return nil
}

func (m *MockDockerManager) Context() context.Context {
	return context.Background()
}

func (m *MockDockerManager) GetSystemStatus(ctx context.Context) (*docker.SystemStatus, error) {
	return nil, nil
}

func (m *MockDockerManager) Cleanup(ctx context.Context) error {
	return nil
}

func (m *MockDockerManager) Shutdown() error {
	return nil
}

// MockStatsCollector Mock 통계 수집기
type MockStatsCollector struct {
	stats map[string]*docker.ContainerStats
}

func NewMockStatsCollector() *MockStatsCollector {
	return &MockStatsCollector{
		stats: make(map[string]*docker.ContainerStats),
	}
}

func (m *MockStatsCollector) Collect(ctx context.Context, containerID string) (*docker.ContainerStats, error) {
	if stats, exists := m.stats[containerID]; exists {
		return stats, nil
	}
	return &docker.ContainerStats{
		CPUPercent:   10.5,
		MemoryUsage:  100 * 1024 * 1024, // 100MB
		MemoryLimit:  1024 * 1024 * 1024, // 1GB
		NetworkRxMB:  50.0,
		NetworkTxMB:  25.0,
		Timestamp:    time.Now(),
	}, nil
}

func (m *MockStatsCollector) CollectAll(ctx context.Context) (map[string]*docker.ContainerStats, error) {
	return m.stats, nil
}

func (m *MockStatsCollector) GetSystemStats(ctx context.Context) (*docker.SystemStats, error) {
	return nil, nil
}

func (m *MockStatsCollector) GetAggregatedStats(ctx context.Context) (*docker.AggregatedStats, error) {
	return nil, nil
}

func (m *MockStatsCollector) Monitor(ctx context.Context, containerID string, interval time.Duration) (<-chan *docker.ContainerStats, error) {
	return nil, nil
}

func (m *MockStatsCollector) MonitorAll(ctx context.Context, interval time.Duration) (<-chan map[string]*docker.ContainerStats, error) {
	return nil, nil
}

func (m *MockStatsCollector) AddStats(containerID string, stats *docker.ContainerStats) {
	m.stats[containerID] = stats
}

// Test functions

func TestNewTracker(t *testing.T) {
	mockService := NewMockWorkspaceService()
	mockContainer := NewMockContainerManager()
	mockFactory := NewMockDockerManager()

	tracker := NewTracker(mockService, mockContainer, mockFactory)

	if tracker == nil {
		t.Fatal("NewTracker should not return nil")
	}

	if tracker.syncInterval != 30*time.Second {
		t.Errorf("Expected sync interval 30s, got %v", tracker.syncInterval)
	}

	if tracker.maxRetries != 3 {
		t.Errorf("Expected max retries 3, got %d", tracker.maxRetries)
	}
}

func TestTracker_StartStop(t *testing.T) {
	mockService := NewMockWorkspaceService()
	mockContainer := NewMockContainerManager()
	mockFactory := NewMockDockerManager()

	tracker := NewTracker(mockService, mockContainer, mockFactory)
	tracker.SetSyncInterval(100 * time.Millisecond) // 빠른 테스트

	// 시작 테스트
	err := tracker.Start()
	if err != nil {
		t.Fatalf("Failed to start tracker: %v", err)
	}

	// 잠시 실행
	time.Sleep(150 * time.Millisecond)

	// 중지 테스트
	err = tracker.Stop()
	if err != nil {
		t.Fatalf("Failed to stop tracker: %v", err)
	}
}

func TestTracker_SyncWorkspaceState(t *testing.T) {
	mockService := NewMockWorkspaceService()
	mockContainer := NewMockContainerManager()
	mockFactory := NewMockDockerManager()

	// 테스트 데이터 설정
	workspace := &models.Workspace{
		ID:     "ws-test-123",
		Name:   "Test Workspace",
		Status: models.WorkspaceStatusActive,
	}
	mockService.AddWorkspace(workspace)

	container := &MockWorkspaceContainer{
		id:          "container-123",
		name:        "test-container",
		workspaceID: "ws-test-123",
		state:       "running",
		createdAt:   time.Now(),
	}
	mockContainer.AddContainer("ws-test-123", container)

	tracker := NewTracker(mockService, mockContainer, mockFactory)

	// 상태 동기화 실행
	tracker.syncWorkspaceState("ws-test-123")

	// 상태 확인
	state, exists := tracker.GetWorkspaceState("ws-test-123")
	if !exists {
		t.Fatal("Workspace state should exist after sync")
	}

	if state.WorkspaceID != "ws-test-123" {
		t.Errorf("Expected workspace ID ws-test-123, got %s", state.WorkspaceID)
	}

	if state.Status != models.WorkspaceStatusActive {
		t.Errorf("Expected status active, got %s", state.Status)
	}

	if state.ContainerID != "container-123" {
		t.Errorf("Expected container ID container-123, got %s", state.ContainerID)
	}
}

func TestTracker_StateChange(t *testing.T) {
	mockService := NewMockWorkspaceService()
	mockContainer := NewMockContainerManager()
	mockFactory := NewMockDockerManager()

	tracker := NewTracker(mockService, mockContainer, mockFactory)

	// 콜백 등록
	var callbackCalled bool
	var receivedWorkspaceID string
	var receivedOldState *WorkspaceState
	var receivedNewState *WorkspaceState

	tracker.OnStateChange(func(workspaceID string, oldState, newState *WorkspaceState) {
		callbackCalled = true
		receivedWorkspaceID = workspaceID
		receivedOldState = oldState
		receivedNewState = newState
	})

	// 테스트 워크스페이스 설정
	workspace := &models.Workspace{
		ID:     "ws-callback-test",
		Name:   "Callback Test",
		Status: models.WorkspaceStatusActive,
	}
	mockService.AddWorkspace(workspace)

	// 상태 변경 트리거
	tracker.syncWorkspaceState("ws-callback-test")

	// 콜백이 호출될 때까지 대기
	time.Sleep(50 * time.Millisecond)

	// 콜백 검증
	if !callbackCalled {
		t.Fatal("State change callback should have been called")
	}

	if receivedWorkspaceID != "ws-callback-test" {
		t.Errorf("Expected workspace ID ws-callback-test, got %s", receivedWorkspaceID)
	}

	if receivedOldState != nil {
		t.Error("Old state should be nil for new workspace")
	}

	if receivedNewState == nil {
		t.Fatal("New state should not be nil")
	}

	if receivedNewState.WorkspaceID != "ws-callback-test" {
		t.Errorf("Expected new state workspace ID ws-callback-test, got %s", receivedNewState.WorkspaceID)
	}
}

func TestTracker_ForceSync(t *testing.T) {
	mockService := NewMockWorkspaceService()
	mockContainer := NewMockContainerManager()
	mockFactory := NewMockDockerManager()

	tracker := NewTracker(mockService, mockContainer, mockFactory)

	// 개별 동기화 테스트
	err := tracker.ForceSync("ws-test-123")
	if err != nil {
		t.Errorf("ForceSync should not return error, got %v", err)
	}

	// 전체 동기화 테스트
	err = tracker.ForceSync("")
	if err != nil {
		t.Errorf("ForceSync all should not return error, got %v", err)
	}
}

func TestTracker_GetAllWorkspaceStates(t *testing.T) {
	mockService := NewMockWorkspaceService()
	mockContainer := NewMockContainerManager()
	mockFactory := NewMockDockerManager()

	tracker := NewTracker(mockService, mockContainer, mockFactory)

	// 테스트 워크스페이스 여러 개 설정
	workspaces := []*models.Workspace{
		{ID: "ws-1", Name: "Workspace 1", Status: models.WorkspaceStatusActive},
		{ID: "ws-2", Name: "Workspace 2", Status: models.WorkspaceStatusInactive},
	}

	for _, ws := range workspaces {
		mockService.AddWorkspace(ws)
		tracker.syncWorkspaceState(ws.ID)
	}

	// 모든 상태 조회
	allStates := tracker.GetAllWorkspaceStates()

	if len(allStates) != 2 {
		t.Errorf("Expected 2 workspace states, got %d", len(allStates))
	}

	for _, ws := range workspaces {
		if _, exists := allStates[ws.ID]; !exists {
			t.Errorf("Workspace %s state not found", ws.ID)
		}
	}
}

func TestTracker_GetStats(t *testing.T) {
	mockService := NewMockWorkspaceService()
	mockContainer := NewMockContainerManager()
	mockFactory := NewMockDockerManager()

	tracker := NewTracker(mockService, mockContainer, mockFactory)

	stats := tracker.GetStats()

	if stats.SyncInterval != 30*time.Second {
		t.Errorf("Expected sync interval 30s, got %v", stats.SyncInterval)
	}

	if stats.ActiveCallbacks != 0 {
		t.Errorf("Expected 0 active callbacks, got %d", stats.ActiveCallbacks)
	}
}

func TestWorkspaceMetrics(t *testing.T) {
	// 메트릭 구조체 테스트
	metrics := &WorkspaceMetrics{
		CPUPercent:   25.5,
		MemoryUsage:  512 * 1024 * 1024, // 512MB
		MemoryLimit:  2048 * 1024 * 1024, // 2GB
		NetworkRxMB:  100.5,
		NetworkTxMB:  50.2,
		Uptime:       "1h30m45s",
		LastActivity: time.Now(),
		ErrorCount:   0,
	}

	if metrics.CPUPercent != 25.5 {
		t.Errorf("Expected CPU percent 25.5, got %f", metrics.CPUPercent)
	}

	if metrics.MemoryUsage != 512*1024*1024 {
		t.Errorf("Expected memory usage 512MB, got %d", metrics.MemoryUsage)
	}

	if metrics.NetworkRxMB != 100.5 {
		t.Errorf("Expected network RX 100.5MB, got %f", metrics.NetworkRxMB)
	}
}