package agent

import (
	"context"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"aicli-web/internal/docker"
	"aicli-web/internal/models"
)

// MockContainerManager 컨테이너 관리자 모의 객체
type MockContainerManager struct {
	mock.Mock
}

func (m *MockContainerManager) CreateWorkspaceContainer(ctx context.Context, req *docker.CreateContainerRequest) (*docker.WorkspaceContainer, error) {
	args := m.Called(ctx, req)
	if args.Get(0) != nil {
		return args.Get(0).(*docker.WorkspaceContainer), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockContainerManager) StartContainer(ctx context.Context, containerID string) error {
	args := m.Called(ctx, containerID)
	return args.Error(0)
}

func (m *MockContainerManager) StopContainer(ctx context.Context, containerID string, timeout time.Duration) error {
	args := m.Called(ctx, containerID, timeout)
	return args.Error(0)
}

func (m *MockContainerManager) RemoveContainer(ctx context.Context, containerID string, force bool) error {
	args := m.Called(ctx, containerID, force)
	return args.Error(0)
}

func (m *MockContainerManager) InspectContainer(ctx context.Context, containerID string) (*docker.WorkspaceContainer, error) {
	args := m.Called(ctx, containerID)
	if args.Get(0) != nil {
		return args.Get(0).(*docker.WorkspaceContainer), args.Error(1)
	}
	return nil, args.Error(1)
}

// MockStatsCollector 통계 수집기 모의 객체
type MockStatsCollector struct {
	mock.Mock
}

func (m *MockStatsCollector) Collect(ctx context.Context, containerID string) (interface{}, error) {
	args := m.Called(ctx, containerID)
	return args.Get(0), args.Error(1)
}

// MockClientInterface Docker 클라이언트 인터페이스 모의 객체
type MockClientInterface struct {
	mock.Mock
}

func (m *MockClientInterface) ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error) {
	args := m.Called(ctx, containerID)
	return args.Get(0).(types.ContainerJSON), args.Error(1)
}

func (m *MockClientInterface) ContainerExecCreate(ctx context.Context, containerID string, config docker.ExecConfig) (types.IDResponse, error) {
	args := m.Called(ctx, containerID, config)
	return args.Get(0).(types.IDResponse), args.Error(1)
}

func (m *MockClientInterface) ContainerExecStart(ctx context.Context, execID string, config docker.ExecStartConfig) (types.HijackedResponse, error) {
	args := m.Called(ctx, execID, config)
	return args.Get(0).(types.HijackedResponse), args.Error(1)
}

func (m *MockClientInterface) ContainerExecInspect(ctx context.Context, execID string) (types.ContainerExecInspect, error) {
	args := m.Called(ctx, execID)
	return args.Get(0).(types.ContainerExecInspect), args.Error(1)
}

func TestDockerAdapter_CreateAgentContainer(t *testing.T) {
	ctx := context.Background()
	
	// 모의 객체 생성
	mockContainerManager := &MockContainerManager{}
	mockClient := &MockClientInterface{}
	mockStatsCollector := &MockStatsCollector{}
	
	// Docker 어댑터 생성
	adapter := NewDockerAdapter(mockContainerManager, mockClient, mockStatsCollector)
	
	// Agent 전용 Docker 통합 설정
	mockDockerClient := &MockDockerClient{}
	agentIntegration := &DockerAgentIntegration{
		client:      mockDockerClient,
		imageName:   DefaultAgentImage,
		networkName: "aicli-agent-network",
	}
	adapter.(*dockerAdapter).SetAgentIntegration(agentIntegration)
	
	// 컨테이너 설정
	config := ContainerConfig{
		Image: DefaultAgentImage,
		Labels: map[string]string{
			"type":       "agent",
			"agent_id":   "test-agent-123",
			"project_id": "test-project-456",
			"agent_name": "test-agent",
			"agent_type": string(models.AgentTypeClaude),
		},
		Environment: map[string]string{
			"CLAUDE_API_KEY": "test-api-key",
			"GIT_USER_NAME":  "Test User",
			"GIT_USER_EMAIL": "test@example.com",
		},
		WorkingDir: "/tmp/test-worktree",
	}
	
	// 네트워크 확인 모의
	mockDockerClient.On("NetworkList", ctx, mock.Anything).Return([]types.NetworkResource{
		{Name: "aicli-agent-network"},
	}, nil)
	
	// 컨테이너 생성 모의
	mockDockerClient.On("ContainerCreate", ctx, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(container.CreateResponse{ID: "container-abc123"}, nil)
	
	// 컨테이너 생성 실행
	containerInfo, err := adapter.CreateContainer(ctx, config)
	
	// 검증
	assert.NoError(t, err)
	assert.NotNil(t, containerInfo)
	assert.Equal(t, "container-abc123", containerInfo.ID)
	assert.Equal(t, "aicli-agent-test-agent-123", containerInfo.Name)
	assert.Equal(t, "created", containerInfo.Status)
	mockDockerClient.AssertExpectations(t)
}

func TestDockerAdapter_CreateRegularContainer(t *testing.T) {
	ctx := context.Background()
	
	// 모의 객체 생성
	mockContainerManager := &MockContainerManager{}
	mockClient := &MockClientInterface{}
	mockStatsCollector := &MockStatsCollector{}
	
	// Docker 어댑터 생성
	adapter := NewDockerAdapter(mockContainerManager, mockClient, mockStatsCollector)
	
	// 일반 컨테이너 설정
	config := ContainerConfig{
		Image: "ubuntu:22.04",
		Labels: map[string]string{
			"app": "test",
		},
		Environment: map[string]string{
			"ENV_VAR": "value",
		},
		WorkingDir: "/app",
	}
	
	// 컨테이너 생성 모의
	mockContainerManager.On("CreateWorkspaceContainer", ctx, mock.Anything).
		Return(&docker.WorkspaceContainer{
			ID:      "container-xyz789",
			Name:    "test-container",
			State:   docker.ContainerStateCreated,
			Created: time.Now(),
		}, nil)
	
	// 컨테이너 생성 실행
	containerInfo, err := adapter.CreateContainer(ctx, config)
	
	// 검증
	assert.NoError(t, err)
	assert.NotNil(t, containerInfo)
	assert.Equal(t, "container-xyz789", containerInfo.ID)
	assert.Equal(t, "test-container", containerInfo.Name)
	assert.Equal(t, "created", containerInfo.Status)
	mockContainerManager.AssertExpectations(t)
}