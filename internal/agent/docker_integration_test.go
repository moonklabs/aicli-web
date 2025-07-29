package agent

import (
	"context"
	"io"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"aicli-web/internal/models"
)

// MockDockerClient Docker 클라이언트 모의 객체
type MockDockerClient struct {
	mock.Mock
}

func (m *MockDockerClient) ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform *types.Platform, containerName string) (container.CreateResponse, error) {
	args := m.Called(ctx, config, hostConfig, networkingConfig, platform, containerName)
	return args.Get(0).(container.CreateResponse), args.Error(1)
}

func (m *MockDockerClient) ContainerStart(ctx context.Context, containerID string, options types.ContainerStartOptions) error {
	args := m.Called(ctx, containerID, options)
	return args.Error(0)
}

func (m *MockDockerClient) ContainerStop(ctx context.Context, containerID string, options container.StopOptions) error {
	args := m.Called(ctx, containerID, options)
	return args.Error(0)
}

func (m *MockDockerClient) ContainerRemove(ctx context.Context, containerID string, options types.ContainerRemoveOptions) error {
	args := m.Called(ctx, containerID, options)
	return args.Error(0)
}

func (m *MockDockerClient) ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error) {
	args := m.Called(ctx, containerID)
	return args.Get(0).(types.ContainerJSON), args.Error(1)
}

func (m *MockDockerClient) ContainerLogs(ctx context.Context, containerID string, options types.ContainerLogsOptions) (types.LogReader, error) {
	args := m.Called(ctx, containerID, options)
	if args.Get(0) != nil {
		return args.Get(0).(types.LogReader), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockDockerClient) ContainerExecCreate(ctx context.Context, containerID string, config types.ExecConfig) (types.IDResponse, error) {
	args := m.Called(ctx, containerID, config)
	return args.Get(0).(types.IDResponse), args.Error(1)
}

func (m *MockDockerClient) ContainerExecAttach(ctx context.Context, execID string, config types.ExecStartCheck) (types.HijackedResponse, error) {
	args := m.Called(ctx, execID, config)
	return args.Get(0).(types.HijackedResponse), args.Error(1)
}

func (m *MockDockerClient) ContainerExecInspect(ctx context.Context, execID string) (types.ContainerExecInspect, error) {
	args := m.Called(ctx, execID)
	return args.Get(0).(types.ContainerExecInspect), args.Error(1)
}

func (m *MockDockerClient) NetworkList(ctx context.Context, options types.NetworkListOptions) ([]types.NetworkResource, error) {
	args := m.Called(ctx, options)
	return args.Get(0).([]types.NetworkResource), args.Error(1)
}

func (m *MockDockerClient) NetworkCreate(ctx context.Context, name string, options types.NetworkCreate) (types.NetworkCreateResponse, error) {
	args := m.Called(ctx, name, options)
	return args.Get(0).(types.NetworkCreateResponse), args.Error(1)
}

func (m *MockDockerClient) VolumeRemove(ctx context.Context, volumeID string, force bool) error {
	args := m.Called(ctx, volumeID, force)
	return args.Error(0)
}

// 테스트를 위한 실제 client.Client 타입으로 래핑
type testDockerClient struct {
	*client.Client
	mock *MockDockerClient
}

func TestDockerAgentIntegration_CreateAgentContainer(t *testing.T) {
	ctx := context.Background()
	
	// 모의 Docker 클라이언트 생성
	mockClient := &MockDockerClient{}
	
	// 에이전트 모델 준비
	agent := &models.Agent{
		ID:        "test-agent-123",
		ProjectID: "test-project-456",
		Name:      "test-agent",
		Type:      models.AgentTypeClaude,
		Config: map[string]interface{}{
			"claude_api_key": "test-api-key",
			"git_user_name":  "Test User",
			"git_user_email": "test@example.com",
		},
	}
	
	worktreePath := "/tmp/test-worktree"
	
	// 예상 컨테이너 설정
	expectedContainerName := "aicli-agent-test-agent-123"
	expectedContainerID := "container-abc123"
	
	// 네트워크 확인 모의
	mockClient.On("NetworkList", ctx, mock.Anything).Return([]types.NetworkResource{
		{Name: "aicli-agent-network"},
	}, nil)
	
	// 컨테이너 생성 모의
	mockClient.On("ContainerCreate", ctx, mock.Anything, mock.Anything, mock.Anything, mock.Anything, expectedContainerName).
		Run(func(args mock.Arguments) {
			// 컨테이너 설정 검증
			config := args.Get(1).(*container.Config)
			assert.Equal(t, DefaultAgentImage, config.Image)
			assert.Contains(t, config.Env, "AGENT_ID=test-agent-123")
			assert.Contains(t, config.Env, "PROJECT_ID=test-project-456")
			assert.Contains(t, config.Env, "CLAUDE_API_KEY=test-api-key")
			assert.Equal(t, AgentWorkDir, config.WorkingDir)
			
			// 호스트 설정 검증
			hostConfig := args.Get(2).(*container.HostConfig)
			assert.Len(t, hostConfig.Mounts, 2) // worktree + claude data volume
			
			// 첫 번째 마운트는 worktree
			worktreeMount := hostConfig.Mounts[0]
			assert.Equal(t, mount.TypeBind, worktreeMount.Type)
			assert.Equal(t, worktreePath, worktreeMount.Source)
			assert.Equal(t, AgentWorkDir, worktreeMount.Target)
			
			// 두 번째 마운트는 claude data volume
			dataMount := hostConfig.Mounts[1]
			assert.Equal(t, mount.TypeVolume, dataMount.Type)
			assert.Equal(t, "aicli-agent-test-agent-123-claude", dataMount.Source)
		}).
		Return(container.CreateResponse{ID: expectedContainerID}, nil)
	
	// 테스트를 위한 커스텀 생성자 사용
	integration := &DockerAgentIntegration{
		client:      mockClient,
		imageName:   DefaultAgentImage,
		networkName: "aicli-agent-network",
	}
	
	// 컨테이너 생성 실행
	containerID, err := integration.CreateAgentContainer(ctx, agent, worktreePath)
	
	// 검증
	assert.NoError(t, err)
	assert.Equal(t, expectedContainerID, containerID)
	mockClient.AssertExpectations(t)
}

func TestDockerAgentIntegration_StartAgentContainer(t *testing.T) {
	ctx := context.Background()
	mockClient := &MockDockerClient{}
	
	containerID := "container-abc123"
	
	// 컨테이너 시작 모의
	mockClient.On("ContainerStart", ctx, containerID, types.ContainerStartOptions{}).Return(nil)
	
	// 컨테이너 상태 확인 모의
	mockClient.On("ContainerInspect", ctx, containerID).Return(types.ContainerJSON{
		State: &types.ContainerState{
			Running: true,
		},
	}, nil)
	
	integration := &DockerAgentIntegration{
		client:      mockClient,
		imageName:   DefaultAgentImage,
		networkName: "aicli-agent-network",
	}
	
	// 컨테이너 시작 실행
	err := integration.StartAgentContainer(ctx, containerID)
	
	// 검증
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestDockerAgentIntegration_StopAgentContainer(t *testing.T) {
	ctx := context.Background()
	mockClient := &MockDockerClient{}
	
	containerID := "container-abc123"
	timeout := 30
	
	// 컨테이너 중지 모의
	mockClient.On("ContainerStop", ctx, containerID, container.StopOptions{
		Timeout: &timeout,
	}).Return(nil)
	
	integration := &DockerAgentIntegration{
		client:      mockClient,
		imageName:   DefaultAgentImage,
		networkName: "aicli-agent-network",
	}
	
	// 컨테이너 중지 실행
	err := integration.StopAgentContainer(ctx, containerID)
	
	// 검증
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestDockerAgentIntegration_RemoveAgentContainer(t *testing.T) {
	ctx := context.Background()
	mockClient := &MockDockerClient{}
	
	containerID := "container-abc123"
	
	// 컨테이너 제거 모의
	mockClient.On("ContainerRemove", ctx, containerID, types.ContainerRemoveOptions{
		RemoveVolumes: false,
		Force:         true,
	}).Return(nil)
	
	integration := &DockerAgentIntegration{
		client:      mockClient,
		imageName:   DefaultAgentImage,
		networkName: "aicli-agent-network",
	}
	
	// 컨테이너 제거 실행
	err := integration.RemoveAgentContainer(ctx, containerID)
	
	// 검증
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestDockerAgentIntegration_EnsureAgentNetwork(t *testing.T) {
	ctx := context.Background()
	
	t.Run("네트워크가 이미 존재하는 경우", func(t *testing.T) {
		mockClient := &MockDockerClient{}
		
		// 네트워크 목록에 이미 존재
		mockClient.On("NetworkList", ctx, types.NetworkListOptions{}).Return([]types.NetworkResource{
			{Name: "aicli-agent-network"},
		}, nil)
		
		integration := &DockerAgentIntegration{
			client:      mockClient,
			imageName:   DefaultAgentImage,
			networkName: "aicli-agent-network",
		}
		
		err := integration.EnsureAgentNetwork(ctx)
		assert.NoError(t, err)
		mockClient.AssertExpectations(t)
	})
	
	t.Run("네트워크를 새로 생성해야 하는 경우", func(t *testing.T) {
		mockClient := &MockDockerClient{}
		
		// 네트워크 목록에 없음
		mockClient.On("NetworkList", ctx, types.NetworkListOptions{}).Return([]types.NetworkResource{}, nil)
		
		// 네트워크 생성 모의
		mockClient.On("NetworkCreate", ctx, "aicli-agent-network", types.NetworkCreate{
			Driver: "bridge",
			Labels: map[string]string{
				"aicli.network.type": "agent",
			},
			Options: map[string]string{
				"com.docker.network.bridge.name": "aicli-agent-br",
			},
		}).Return(types.NetworkCreateResponse{ID: "network-123"}, nil)
		
		integration := &DockerAgentIntegration{
			client:      mockClient,
			imageName:   DefaultAgentImage,
			networkName: "aicli-agent-network",
		}
		
		err := integration.EnsureAgentNetwork(ctx)
		assert.NoError(t, err)
		mockClient.AssertExpectations(t)
	})
}

func TestDockerAgentIntegration_ExecInAgentContainer(t *testing.T) {
	ctx := context.Background()
	mockClient := &MockDockerClient{}
	
	containerID := "container-abc123"
	cmd := []string{"claude", "--version"}
	execID := "exec-xyz789"
	
	// Exec 생성 모의
	mockClient.On("ContainerExecCreate", ctx, containerID, mock.MatchedBy(func(config types.ExecConfig) bool {
		return config.Cmd[0] == "claude" && config.Cmd[1] == "--version" &&
			config.WorkingDir == AgentWorkDir && config.User == "agent"
	})).Return(types.IDResponse{ID: execID}, nil)
	
	// Exec 실행 모의
	mockClient.On("ContainerExecAttach", ctx, execID, types.ExecStartCheck{}).Return(types.HijackedResponse{
		Reader: &mockReader{data: []byte("claude version 0.1.0\n")},
	}, nil)
	
	// Exec 상태 확인 모의
	mockClient.On("ContainerExecInspect", ctx, execID).Return(types.ContainerExecInspect{
		ExitCode: 0,
	}, nil)
	
	integration := &DockerAgentIntegration{
		client:      mockClient,
		imageName:   DefaultAgentImage,
		networkName: "aicli-agent-network",
	}
	
	// 명령 실행
	output, err := integration.ExecInAgentContainer(ctx, containerID, cmd)
	
	// 검증
	assert.NoError(t, err)
	assert.Equal(t, "claude version 0.1.0\n", output)
	mockClient.AssertExpectations(t)
}

// mockReader io.Reader 모의 구현
type mockReader struct {
	data []byte
	pos  int
}

func (r *mockReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}