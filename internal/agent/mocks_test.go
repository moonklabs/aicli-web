package agent

import (
	"context"
	"io"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/system"
	"github.com/stretchr/testify/mock"

	"github.com/aicli/aicli-web/internal/docker"
)

// MockContainerManager Docker 컨테이너 매니저 Mock (통합 버전)
type MockContainerManager struct {
	mock.Mock
}

func (m *MockContainerManager) CreateWorkspaceContainer(ctx context.Context, req *docker.CreateContainerRequest) (*docker.WorkspaceContainer, error) {
	args := m.Called(ctx, req)
	if args.Get(0) != nil {
		return args.Get(0).(*docker.WorkspaceContainer), args.Error(1)
	}
	// 기본 Mock 동작 (mock.Mock이 없는 경우)
	return &docker.WorkspaceContainer{
		ID:          "test-container-id",
		Name:        req.Name,
		WorkspaceID: req.WorkspaceID,
		State:       docker.ContainerStateCreated,
		Created:     time.Now(),
	}, nil
}

func (m *MockContainerManager) InspectContainer(ctx context.Context, containerID string) (*docker.WorkspaceContainer, error) {
	args := m.Called(ctx, containerID)
	if args.Get(0) != nil {
		return args.Get(0).(*docker.WorkspaceContainer), args.Error(1)
	}
	// 기본 Mock 동작
	return &docker.WorkspaceContainer{
		ID:          containerID,
		Name:        "test-container",
		WorkspaceID: "test-workspace",
		State:       docker.ContainerStateRunning,
		Created:     time.Now(),
	}, nil
}

func (m *MockContainerManager) ListWorkspaceContainers(ctx context.Context, workspaceID string) ([]*docker.WorkspaceContainer, error) {
	args := m.Called(ctx, workspaceID)
	if args.Get(0) != nil {
		return args.Get(0).([]*docker.WorkspaceContainer), args.Error(1)
	}
	return []*docker.WorkspaceContainer{}, nil
}

func (m *MockContainerManager) ListContainers(ctx context.Context, labels map[string]string) ([]*docker.WorkspaceContainer, error) {
	args := m.Called(ctx, labels)
	if args.Get(0) != nil {
		return args.Get(0).([]*docker.WorkspaceContainer), args.Error(1)
	}
	return []*docker.WorkspaceContainer{}, nil
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

func (m *MockContainerManager) CleanupWorkspace(ctx context.Context, workspaceID string, force bool) error {
	args := m.Called(ctx, workspaceID, force)
	return args.Error(0)
}

// MockDockerClient Docker 클라이언트 Mock (통합 버전)
type MockDockerClient struct {
	mock.Mock
}

func (m *MockDockerClient) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockDockerClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockDockerClient) GetConfig() *docker.Config {
	args := m.Called()
	if args.Get(0) != nil {
		return args.Get(0).(*docker.Config)
	}
	return nil
}

func (m *MockDockerClient) GetNetworkID() string {
	args := m.Called()
	if args.Get(0) != nil {
		return args.String(0)
	}
	return "test-network"
}

// NetworkManagement 메서드들
func (m *MockDockerClient) CreateNetwork(ctx context.Context, req docker.CreateNetworkRequest) (*docker.NetworkInfo, error) {
	args := m.Called(ctx, req)
	if args.Get(0) != nil {
		return args.Get(0).(*docker.NetworkInfo), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockDockerClient) GetNetwork(ctx context.Context, networkID string) (*docker.NetworkInfo, error) {
	args := m.Called(ctx, networkID)
	if args.Get(0) != nil {
		return args.Get(0).(*docker.NetworkInfo), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockDockerClient) ListNetworks(ctx context.Context) ([]docker.NetworkInfo, error) {
	args := m.Called(ctx)
	if args.Get(0) != nil {
		return args.Get(0).([]docker.NetworkInfo), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockDockerClient) DeleteNetwork(ctx context.Context, networkID string) error {
	args := m.Called(ctx, networkID)
	return args.Error(0)
}

func (m *MockDockerClient) ConnectContainer(ctx context.Context, networkID, containerID string, config *network.EndpointSettings) error {
	args := m.Called(ctx, networkID, containerID, config)
	return args.Error(0)
}

func (m *MockDockerClient) DisconnectContainer(ctx context.Context, networkID, containerID string, force bool) error {
	args := m.Called(ctx, networkID, containerID, force)
	return args.Error(0)
}

func (m *MockDockerClient) EnsureNetwork(ctx context.Context, name string) (string, error) {
	args := m.Called(ctx, name)
	if args.Get(0) != nil {
		return args.String(0), args.Error(1)
	}
	return "test-network", args.Error(1)
}

func (m *MockDockerClient) CleanupNetworks(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// HealthMonitoring 메서드들
func (m *MockDockerClient) CheckDaemon(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockDockerClient) CheckContainer(ctx context.Context, containerID string) (bool, error) {
	args := m.Called(ctx, containerID)
	return args.Bool(0), args.Error(1)
}

func (m *MockDockerClient) GetSystemInfo(ctx context.Context) (*system.Info, error) {
	args := m.Called(ctx)
	if args.Get(0) != nil {
		return args.Get(0).(*system.Info), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockDockerClient) GetVersion(ctx context.Context) (types.Version, error) {
	args := m.Called(ctx)
	return args.Get(0).(types.Version), args.Error(1)
}

func (m *MockDockerClient) WaitHealthy(ctx context.Context, containerID string, timeout time.Duration) error {
	args := m.Called(ctx, containerID, timeout)
	return args.Error(0)
}

func (m *MockDockerClient) StartMonitoring(ctx context.Context, callback func(error)) {
	m.Called(ctx, callback)
}

// Docker API 메서드들
func (m *MockDockerClient) ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform interface{}, containerName string) (container.CreateResponse, error) {
	args := m.Called(ctx, config, hostConfig, networkingConfig, platform, containerName)
	return args.Get(0).(container.CreateResponse), args.Error(1)
}

func (m *MockDockerClient) ContainerStart(ctx context.Context, containerID string, options container.StartOptions) error {
	args := m.Called(ctx, containerID, options)
	return args.Error(0)
}

func (m *MockDockerClient) ContainerStop(ctx context.Context, containerID string, options container.StopOptions) error {
	args := m.Called(ctx, containerID, options)
	return args.Error(0)
}

func (m *MockDockerClient) ContainerRemove(ctx context.Context, containerID string, options container.RemoveOptions) error {
	args := m.Called(ctx, containerID, options)
	return args.Error(0)
}

func (m *MockDockerClient) ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error) {
	args := m.Called(ctx, containerID)
	return args.Get(0).(types.ContainerJSON), args.Error(1)
}

func (m *MockDockerClient) ContainerExecCreate(ctx context.Context, containerID string, config docker.ExecConfig) (types.IDResponse, error) {
	args := m.Called(ctx, containerID, config)
	return args.Get(0).(types.IDResponse), args.Error(1)
}

func (m *MockDockerClient) ContainerExecStart(ctx context.Context, execID string, config docker.ExecStartConfig) (types.HijackedResponse, error) {
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

func (m *MockDockerClient) ContainerLogs(ctx context.Context, containerID string, options types.ContainerLogsOptions) (io.ReadCloser, error) {
	args := m.Called(ctx, containerID, options)
	if args.Get(0) != nil {
		return args.Get(0).(io.ReadCloser), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockDockerClient) ContainerExecAttach(ctx context.Context, execID string, config types.ExecStartCheck) (types.HijackedResponse, error) {
	args := m.Called(ctx, execID, config)
	return args.Get(0).(types.HijackedResponse), args.Error(1)
}

func (m *MockDockerClient) VolumeRemove(ctx context.Context, volumeID string, force bool) error {
	args := m.Called(ctx, volumeID, force)
	return args.Error(0)
}

func (m *MockDockerClient) ContainerKill(ctx context.Context, containerID string, signal string) error {
	args := m.Called(ctx, containerID, signal)
	return args.Error(0)
}

// StatsCollection 메서드들
func (m *MockDockerClient) Collect(ctx context.Context, containerID string) (*docker.ContainerStats, error) {
	args := m.Called(ctx, containerID)
	if args.Get(0) != nil {
		return args.Get(0).(*docker.ContainerStats), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockDockerClient) CollectAll(ctx context.Context) (map[string]*docker.ContainerStats, error) {
	args := m.Called(ctx)
	if args.Get(0) != nil {
		return args.Get(0).(map[string]*docker.ContainerStats), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockDockerClient) GetSystemStats(ctx context.Context) (*docker.SystemStats, error) {
	args := m.Called(ctx)
	if args.Get(0) != nil {
		return args.Get(0).(*docker.SystemStats), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockDockerClient) GetAggregatedStats(ctx context.Context) (*docker.AggregatedStats, error) {
	args := m.Called(ctx)
	if args.Get(0) != nil {
		return args.Get(0).(*docker.AggregatedStats), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockDockerClient) Monitor(ctx context.Context, containerID string, interval time.Duration) (<-chan *docker.ContainerStats, error) {
	args := m.Called(ctx, containerID, interval)
	if args.Get(0) != nil {
		return args.Get(0).(<-chan *docker.ContainerStats), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockDockerClient) MonitorAll(ctx context.Context, interval time.Duration) (<-chan map[string]*docker.ContainerStats, error) {
	args := m.Called(ctx, interval)
	if args.Get(0) != nil {
		return args.Get(0).(<-chan map[string]*docker.ContainerStats), args.Error(1)
	}
	return nil, args.Error(1)
}

// MockStatsCollector 통계 수집기 모의 객체 (통합 버전)
type MockStatsCollector struct {
	mock.Mock
}

func (m *MockStatsCollector) Collect(ctx context.Context, containerID string) (*docker.ContainerStats, error) {
	args := m.Called(ctx, containerID)
	if args.Get(0) != nil {
		return args.Get(0).(*docker.ContainerStats), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockStatsCollector) CollectAll(ctx context.Context) (map[string]*docker.ContainerStats, error) {
	args := m.Called(ctx)
	if args.Get(0) != nil {
		return args.Get(0).(map[string]*docker.ContainerStats), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockStatsCollector) GetSystemStats(ctx context.Context) (*docker.SystemStats, error) {
	args := m.Called(ctx)
	if args.Get(0) != nil {
		return args.Get(0).(*docker.SystemStats), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockStatsCollector) GetAggregatedStats(ctx context.Context) (*docker.AggregatedStats, error) {
	args := m.Called(ctx)
	if args.Get(0) != nil {
		return args.Get(0).(*docker.AggregatedStats), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockStatsCollector) Monitor(ctx context.Context, containerID string, interval time.Duration) (<-chan *docker.ContainerStats, error) {
	args := m.Called(ctx, containerID, interval)
	if args.Get(0) != nil {
		return args.Get(0).(<-chan *docker.ContainerStats), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockStatsCollector) MonitorAll(ctx context.Context, interval time.Duration) (<-chan map[string]*docker.ContainerStats, error) {
	args := m.Called(ctx, interval)
	if args.Get(0) != nil {
		return args.Get(0).(<-chan map[string]*docker.ContainerStats), args.Error(1)
	}
	return nil, args.Error(1)
}
