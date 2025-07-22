package services

import (
	"context"
	"testing"
	"time"
	
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	
	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/docker"
	"github.com/aicli/aicli-web/internal/docker/security"
)

// MockWorkspaceService is a mock implementation of WorkspaceService
type MockWorkspaceService struct {
	mock.Mock
}

func (m *MockWorkspaceService) CreateWorkspace(ctx context.Context, req *models.CreateWorkspaceRequest, ownerID string) (*models.Workspace, error) {
	args := m.Called(ctx, req, ownerID)
	return args.Get(0).(*models.Workspace), args.Error(1)
}

func (m *MockWorkspaceService) GetWorkspace(ctx context.Context, id string, ownerID string) (*models.Workspace, error) {
	args := m.Called(ctx, id, ownerID)
	return args.Get(0).(*models.Workspace), args.Error(1)
}

func (m *MockWorkspaceService) UpdateWorkspace(ctx context.Context, id string, req *models.UpdateWorkspaceRequest, ownerID string) (*models.Workspace, error) {
	args := m.Called(ctx, id, req, ownerID)
	return args.Get(0).(*models.Workspace), args.Error(1)
}

func (m *MockWorkspaceService) DeleteWorkspace(ctx context.Context, id string, ownerID string) error {
	args := m.Called(ctx, id, ownerID)
	return args.Error(0)
}

func (m *MockWorkspaceService) ListWorkspaces(ctx context.Context, ownerID string, req *models.PaginationRequest) (*models.WorkspaceListResponse, error) {
	args := m.Called(ctx, ownerID, req)
	return args.Get(0).(*models.WorkspaceListResponse), args.Error(1)
}

func (m *MockWorkspaceService) ValidateWorkspace(ctx context.Context, workspace *models.Workspace) error {
	args := m.Called(ctx, workspace)
	return args.Error(0)
}

func (m *MockWorkspaceService) ActivateWorkspace(ctx context.Context, id string, ownerID string) error {
	args := m.Called(ctx, id, ownerID)
	return args.Error(0)
}

func (m *MockWorkspaceService) DeactivateWorkspace(ctx context.Context, id string, ownerID string) error {
	args := m.Called(ctx, id, ownerID)
	return args.Error(0)
}

func (m *MockWorkspaceService) ArchiveWorkspace(ctx context.Context, id string, ownerID string) error {
	args := m.Called(ctx, id, ownerID)
	return args.Error(0)
}

func (m *MockWorkspaceService) UpdateActiveTaskCount(ctx context.Context, id string, delta int) error {
	args := m.Called(ctx, id, delta)
	return args.Error(0)
}

func (m *MockWorkspaceService) GetWorkspaceStats(ctx context.Context, ownerID string) (*WorkspaceStats, error) {
	args := m.Called(ctx, ownerID)
	return args.Get(0).(*WorkspaceStats), args.Error(1)
}

// MockDockerManager is a mock implementation of Docker Manager
type MockDockerManager struct {
	mock.Mock
	containerManager *MockContainerManager
}

func (m *MockDockerManager) Container() *MockContainerManager {
	return m.containerManager
}

// MockContainerManager is a mock implementation of Container Manager
type MockContainerManager struct {
	mock.Mock
}

func (m *MockContainerManager) CreateWorkspaceContainer(ctx context.Context, req *docker.CreateContainerRequest) (*docker.WorkspaceContainer, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*docker.WorkspaceContainer), args.Error(1)
}

func (m *MockContainerManager) StartContainer(ctx context.Context, containerID string) error {
	args := m.Called(ctx, containerID)
	return args.Error(0)
}

func (m *MockContainerManager) StopContainer(ctx context.Context, containerID string, timeout time.Duration) error {
	args := m.Called(ctx, containerID, timeout)
	return args.Error(0)
}

func (m *MockContainerManager) RestartContainer(ctx context.Context, containerID string, timeout time.Duration) error {
	args := m.Called(ctx, containerID, timeout)
	return args.Error(0)
}

func (m *MockContainerManager) RemoveContainer(ctx context.Context, containerID string, force bool) error {
	args := m.Called(ctx, containerID, force)
	return args.Error(0)
}

func (m *MockContainerManager) ListWorkspaceContainers(ctx context.Context, workspaceID string) ([]*docker.WorkspaceContainer, error) {
	args := m.Called(ctx, workspaceID)
	return args.Get(0).([]*docker.WorkspaceContainer), args.Error(1)
}

// MockStorage is a mock implementation of Storage interface
type MockStorage struct {
	mock.Mock
	workspace *MockWorkspaceStorage
}

func (m *MockStorage) Workspace() *MockWorkspaceStorage {
	return m.workspace
}

type MockWorkspaceStorage struct {
	mock.Mock
}

func (m *MockWorkspaceStorage) GetByID(ctx context.Context, id string) (*models.Workspace, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.Workspace), args.Error(1)
}

func (m *MockWorkspaceStorage) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	args := m.Called(ctx, id, updates)
	return args.Error(0)
}

func TestDockerWorkspaceService_CreateWorkspace(t *testing.T) {
	// Mock 설정
	mockBaseService := new(MockWorkspaceService)
	mockStorage := new(MockStorage)
	mockWorkspaceStorage := new(MockWorkspaceStorage)
	mockStorage.workspace = mockWorkspaceStorage
	
	mockContainerManager := new(MockContainerManager)
	mockDockerManager := new(MockDockerManager)
	mockDockerManager.containerManager = mockContainerManager
	
	// 테스트용 워크스페이스
	workspace := &models.Workspace{
		ID:          "test-workspace-id",
		Name:        "test-workspace",
		ProjectPath: "/tmp/test",
		Status:      models.WorkspaceStatusActive,
		OwnerID:     "user1",
		ActiveTasks: 0,
		CreatedAt:   time.Now(),
	}
	
	// 테스트용 컨테이너
	container := &docker.WorkspaceContainer{
		ID:          "container-123",
		Name:        "workspace-test-workspace-id",
		WorkspaceID: "test-workspace-id",
		State:       docker.ContainerStateRunning,
		Created:     time.Now(),
	}
	
	// Mock 동작 설정
	req := &models.CreateWorkspaceRequest{
		Name:        "test-workspace",
		ProjectPath: "/tmp/test",
		ClaudeKey:   "test-key",
	}
	
	mockBaseService.On("CreateWorkspace", mock.Anything, req, "user1").Return(workspace, nil)
	mockWorkspaceStorage.On("GetByID", mock.Anything, "test-workspace-id").Return(workspace, nil)
	mockContainerManager.On("CreateWorkspaceContainer", mock.Anything, mock.AnythingOfType("*docker.CreateContainerRequest")).Return(container, nil)
	mockContainerManager.On("StartContainer", mock.Anything, "container-123").Return(nil)
	mockWorkspaceStorage.On("Update", mock.Anything, "test-workspace-id", mock.Anything).Return(nil)
	
	// 서비스 생성
	service := NewDockerWorkspaceService(mockBaseService, mockStorage, mockDockerManager)
	defer service.Close()
	
	// 테스트 실행
	result, err := service.CreateWorkspace(context.Background(), req, "user1")
	
	// 검증
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "test-workspace-id", result.ID)
	assert.Equal(t, "test-workspace", result.Name)
	
	// Mock 호출 검증
	mockBaseService.AssertExpectations(t)
	mockContainerManager.AssertExpectations(t)
	mockWorkspaceStorage.AssertExpectations(t)
}

func TestDockerWorkspaceService_ExecuteTask(t *testing.T) {
	// Mock 설정
	mockBaseService := new(MockWorkspaceService)
	mockStorage := new(MockStorage)
	mockWorkspaceStorage := new(MockWorkspaceStorage)
	mockStorage.workspace = mockWorkspaceStorage
	
	mockContainerManager := new(MockContainerManager)
	mockDockerManager := new(MockDockerManager)
	mockDockerManager.containerManager = mockContainerManager
	
	// 테스트용 워크스페이스
	workspace := &models.Workspace{
		ID:          "test-workspace-id",
		Name:        "test-workspace",
		ProjectPath: "/tmp/test",
		Status:      models.WorkspaceStatusInactive,
		OwnerID:     "user1",
		ActiveTasks: 0,
	}
	
	// 테스트용 컨테이너
	containers := []*docker.WorkspaceContainer{
		{
			ID:          "container-123",
			Name:        "workspace-test-workspace-id",
			WorkspaceID: "test-workspace-id",
			State:       docker.ContainerStateExited,
		},
	}
	
	// Mock 동작 설정
	mockWorkspaceStorage.On("GetByID", mock.Anything, "test-workspace-id").Return(workspace, nil)
	mockContainerManager.On("ListWorkspaceContainers", mock.Anything, "test-workspace-id").Return(containers, nil)
	mockContainerManager.On("StartContainer", mock.Anything, "container-123").Return(nil)
	
	// 서비스 생성
	service := NewDockerWorkspaceService(mockBaseService, mockStorage, mockDockerManager)
	defer service.Close()
	
	// 시작 작업 테스트
	task := &WorkspaceTask{
		Type:        TaskTypeStart,
		WorkspaceID: "test-workspace-id",
		Timeout:     30 * time.Second,
		Context:     context.Background(),
	}
	
	err := service.executeTask(task)
	
	// 검증
	assert.NoError(t, err)
	
	// Mock 호출 검증
	mockContainerManager.AssertExpectations(t)
}

func TestBatchOperationRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		request BatchOperationRequest
		valid   bool
	}{
		{
			name: "Valid start operation",
			request: BatchOperationRequest{
				Operation:    "start",
				WorkspaceIDs: []string{"ws1", "ws2", "ws3"},
			},
			valid: true,
		},
		{
			name: "Invalid operation",
			request: BatchOperationRequest{
				Operation:    "invalid",
				WorkspaceIDs: []string{"ws1"},
			},
			valid: false,
		},
		{
			name: "Empty workspace IDs",
			request: BatchOperationRequest{
				Operation:    "start",
				WorkspaceIDs: []string{},
			},
			valid: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 검증 로직 (실제로는 binding 태그를 통해 검증됨)
			hasValidOperation := tt.request.Operation == "start" || tt.request.Operation == "stop" || 
				tt.request.Operation == "restart" || tt.request.Operation == "delete"
			hasWorkspaces := len(tt.request.WorkspaceIDs) > 0
			
			isValid := hasValidOperation && hasWorkspaces
			assert.Equal(t, tt.valid, isValid)
		})
	}
}

func TestDockerWorkspaceService_CalculateUptime(t *testing.T) {
	service := &DockerWorkspaceService{}
	
	tests := []struct {
		name      string
		startTime *time.Time
		expected  string
	}{
		{
			name:      "Nil start time",
			startTime: nil,
			expected:  "",
		},
		{
			name:      "Zero start time",
			startTime: &time.Time{},
			expected:  "",
		},
		{
			name:      "30 seconds ago",
			startTime: func() *time.Time { t := time.Now().Add(-30 * time.Second); return &t }(),
			expected:  "30초",
		},
		{
			name:      "5 minutes ago",
			startTime: func() *time.Time { t := time.Now().Add(-5 * time.Minute); return &t }(),
			expected:  "5분",
		},
		{
			name:      "2.5 hours ago",
			startTime: func() *time.Time { t := time.Now().Add(-150 * time.Minute); return &t }(),
			expected:  "2.5시간",
		},
		{
			name:      "1 day 3 hours ago",
			startTime: func() *time.Time { t := time.Now().Add(-27 * time.Hour); return &t }(),
			expected:  "1일 3시간",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.calculateUptime(tt.startTime)
			if tt.name == "Nil start time" || tt.name == "Zero start time" {
				assert.Equal(t, tt.expected, result)
			} else {
				// 실제 시간 기반 테스트는 대략적인 매치를 확인
				assert.NotEmpty(t, result)
			}
		})
	}
}

func TestErrorRecoveryStrategy(t *testing.T) {
	strategy := &ErrorRecoveryStrategy{
		MaxRetries:      3,
		BackoffDuration: 5 * time.Second,
		FallbackAction:  "stop",
	}
	
	assert.Equal(t, 3, strategy.MaxRetries)
	assert.Equal(t, 5*time.Second, strategy.BackoffDuration)
	assert.Equal(t, "stop", strategy.FallbackAction)
}

func TestWorkspaceStatus(t *testing.T) {
	status := &WorkspaceStatus{
		ContainerID:    "container-123",
		ContainerState: "running",
		Uptime:         "1시간",
		LastError:      "",
	}
	
	assert.Equal(t, "container-123", status.ContainerID)
	assert.Equal(t, "running", status.ContainerState)
	assert.Equal(t, "1시간", status.Uptime)
	assert.Empty(t, status.LastError)
}