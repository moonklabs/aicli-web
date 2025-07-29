package agent

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"

	"github.com/aicli/aicli-web/internal/docker"
)

// MockContainerManager Docker 컨테이너 매니저 Mock
type MockContainerManager struct{}

func (m *MockContainerManager) CreateWorkspaceContainer(ctx context.Context, req *docker.CreateContainerRequest) (*docker.WorkspaceContainer, error) {
	return &docker.WorkspaceContainer{
		ID:          "test-container-id",
		Name:        req.Name,
		WorkspaceID: req.WorkspaceID,
		State:       docker.ContainerStateCreated,
		Created:     time.Now(),
	}, nil
}

func (m *MockContainerManager) InspectContainer(ctx context.Context, containerID string) (*docker.WorkspaceContainer, error) {
	return &docker.WorkspaceContainer{
		ID:          containerID,
		Name:        "test-container",
		WorkspaceID: "test-workspace",
		State:       docker.ContainerStateRunning,
		Created:     time.Now(),
	}, nil
}

func (m *MockContainerManager) ListWorkspaceContainers(ctx context.Context, workspaceID string) ([]*docker.WorkspaceContainer, error) {
	return []*docker.WorkspaceContainer{}, nil
}

func (m *MockContainerManager) ListContainers(ctx context.Context, labels map[string]string) ([]*docker.WorkspaceContainer, error) {
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

// MockDockerClient Docker 클라이언트 Mock
type MockDockerClient struct{}

func (m *MockDockerClient) Ping(ctx context.Context) error                   { return nil }
func (m *MockDockerClient) Close() error                                     { return nil }
func (m *MockDockerClient) GetConfig() *docker.Config                        { return nil }
func (m *MockDockerClient) GetNetworkID() string                             { return "test-network" }

// NetworkManagement 메서드들
func (m *MockDockerClient) CreateNetwork(ctx context.Context, req docker.CreateNetworkRequest) (*docker.NetworkInfo, error) {
	return nil, nil
}
func (m *MockDockerClient) GetNetwork(ctx context.Context, networkID string) (*docker.NetworkInfo, error) {
	return nil, nil
}
func (m *MockDockerClient) ListNetworks(ctx context.Context) ([]docker.NetworkInfo, error) {
	return nil, nil
}
func (m *MockDockerClient) DeleteNetwork(ctx context.Context, networkID string) error {
	return nil
}
func (m *MockDockerClient) ConnectContainer(ctx context.Context, networkID, containerID string, config *network.EndpointSettings) error {
	return nil
}
func (m *MockDockerClient) DisconnectContainer(ctx context.Context, networkID, containerID string, force bool) error {
	return nil
}
func (m *MockDockerClient) EnsureNetwork(ctx context.Context, name string) (string, error) {
	return "test-network", nil
}
func (m *MockDockerClient) CleanupNetworks(ctx context.Context) error {
	return nil
}

// HealthMonitoring 메서드들
func (m *MockDockerClient) CheckDaemon(ctx context.Context) error {
	return nil
}
func (m *MockDockerClient) CheckContainer(ctx context.Context, containerID string) (bool, error) {
	return true, nil
}
func (m *MockDockerClient) GetSystemInfo(ctx context.Context) (*types.Info, error) {
	return nil, nil
}
func (m *MockDockerClient) GetVersion(ctx context.Context) (types.Version, error) {
	return types.Version{}, nil
}
func (m *MockDockerClient) WaitHealthy(ctx context.Context, containerID string, timeout time.Duration) error {
	return nil
}
func (m *MockDockerClient) StartMonitoring(ctx context.Context, callback func(error)) {
}

// StatsCollection 메서드들
func (m *MockDockerClient) Collect(ctx context.Context, containerID string) (*docker.ContainerStats, error) {
	return nil, nil
}
func (m *MockDockerClient) CollectAll(ctx context.Context) (map[string]*docker.ContainerStats, error) {
	return nil, nil
}
func (m *MockDockerClient) GetSystemStats(ctx context.Context) (*docker.SystemStats, error) {
	return nil, nil
}
func (m *MockDockerClient) GetAggregatedStats(ctx context.Context) (*docker.AggregatedStats, error) {
	return nil, nil
}
func (m *MockDockerClient) Monitor(ctx context.Context, containerID string, interval time.Duration) (<-chan *docker.ContainerStats, error) {
	return nil, nil
}
func (m *MockDockerClient) MonitorAll(ctx context.Context, interval time.Duration) (<-chan map[string]*docker.ContainerStats, error) {
	return nil, nil
}

// Docker API 메서드들
func (m *MockDockerClient) ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error) {
	return types.ContainerJSON{
		ContainerJSONBase: &types.ContainerJSONBase{
			ID:    containerID,
			Name:  "/test-container",
			State: &types.ContainerState{
				Running: true,
				Status:  "running",
			},
		},
	}, nil
}
func (m *MockDockerClient) ContainerKill(ctx context.Context, containerID string, signal string) error {
	return nil
}
func (m *MockDockerClient) ContainerRemove(ctx context.Context, containerID string, options types.ContainerRemoveOptions) error {
	return nil
}
func (m *MockDockerClient) ContainerExecCreate(ctx context.Context, containerID string, config docker.ExecConfig) (types.IDResponse, error) {
	return types.IDResponse{ID: "exec-id"}, nil
}
func (m *MockDockerClient) ContainerExecStart(ctx context.Context, execID string, config docker.ExecStartConfig) (types.HijackedResponse, error) {
	// Mock HijackedResponse를 안전하게 생성
	return types.HijackedResponse{
		Conn:   nil, // 테스트에서는 nil로 설정
		Reader: nil, // 테스트에서는 nil로 설정
	}, nil
}
func (m *MockDockerClient) ContainerExecInspect(ctx context.Context, execID string) (types.ContainerExecInspect, error) {
	return types.ContainerExecInspect{ExitCode: 0}, nil
}

// MockStatsCollector 통계 수집기 Mock
type MockStatsCollector struct{}

func (m *MockStatsCollector) Collect(ctx context.Context, containerID string) (*docker.ContainerStats, error) {
	return &docker.ContainerStats{
		ContainerID: containerID,
		Timestamp:   time.Now(),
	}, nil
}

func (m *MockStatsCollector) CollectAll(ctx context.Context) (map[string]*docker.ContainerStats, error) {
	return make(map[string]*docker.ContainerStats), nil
}

func (m *MockStatsCollector) GetSystemStats(ctx context.Context) (*docker.SystemStats, error) {
	return &docker.SystemStats{}, nil
}

func (m *MockStatsCollector) GetAggregatedStats(ctx context.Context) (*docker.AggregatedStats, error) {
	return &docker.AggregatedStats{}, nil
}

func (m *MockStatsCollector) Monitor(ctx context.Context, containerID string, interval time.Duration) (<-chan *docker.ContainerStats, error) {
	ch := make(chan *docker.ContainerStats, 1)
	close(ch)
	return ch, nil
}

func (m *MockStatsCollector) MonitorAll(ctx context.Context, interval time.Duration) (<-chan map[string]*docker.ContainerStats, error) {
	ch := make(chan map[string]*docker.ContainerStats, 1)
	close(ch)
	return ch, nil
}

func TestDockerAdapter_CreateContainer(t *testing.T) {
	// Arrange
	containerManager := &MockContainerManager{}
	client := &MockDockerClient{}
	statsCollector := &MockStatsCollector{}
	adapter := NewDockerAdapter(containerManager, client, statsCollector)

	config := ContainerConfig{
		Image:       "test:latest",
		Environment: map[string]string{"ENV": "test"},
		WorkingDir:  "/app",
		Labels:      map[string]string{"app": "test"},
	}

	// Act
	result, err := adapter.CreateContainer(context.Background(), config)

	// Assert
	if err != nil {
		t.Fatalf("CreateContainer 실패: %v", err)
	}

	if result.ID != "test-container-id" {
		t.Errorf("예상 ID: test-container-id, 실제: %s", result.ID)
	}
}

func TestDockerAdapter_StartContainer(t *testing.T) {
	// Arrange
	containerManager := &MockContainerManager{}
	client := &MockDockerClient{}
	statsCollector := &MockStatsCollector{}
	adapter := NewDockerAdapter(containerManager, client, statsCollector)

	// Act
	err := adapter.StartContainer(context.Background(), "test-container-id")

	// Assert
	if err != nil {
		t.Fatalf("StartContainer 실패: %v", err)
	}
}

func TestDockerAdapter_StopContainer(t *testing.T) {
	// Arrange
	containerManager := &MockContainerManager{}
	client := &MockDockerClient{}
	statsCollector := &MockStatsCollector{}
	adapter := NewDockerAdapter(containerManager, client, statsCollector)

	// Act
	err := adapter.StopContainer(context.Background(), "test-container-id")

	// Assert
	if err != nil {
		t.Fatalf("StopContainer 실패: %v", err)
	}
}

func TestDockerAdapter_GetContainerStatus(t *testing.T) {
	// Arrange
	containerManager := &MockContainerManager{}
	client := &MockDockerClient{}
	statsCollector := &MockStatsCollector{}
	adapter := NewDockerAdapter(containerManager, client, statsCollector)

	// Act
	status, err := adapter.GetContainerStatus(context.Background(), "test-container-id")

	// Assert
	if err != nil {
		t.Fatalf("GetContainerStatus 실패: %v", err)
	}

	if status.ID != "test-container-id" {
		t.Errorf("예상 ID: test-container-id, 실제: %s", status.ID)
	}

	if status.Status != "running" {
		t.Errorf("예상 상태: running, 실제: %s", status.Status)
	}
}

func TestDockerAdapter_GetContainerHealth(t *testing.T) {
	// Arrange
	containerManager := &MockContainerManager{}
	client := &MockDockerClient{}
	statsCollector := &MockStatsCollector{}
	adapter := NewDockerAdapter(containerManager, client, statsCollector)

	// Act
	health, err := adapter.GetContainerHealth(context.Background(), "test-container-id")

	// Assert
	if err != nil {
		t.Fatalf("GetContainerHealth 실패: %v", err)
	}

	if health.Status != "healthy" {
		t.Errorf("예상 상태: healthy, 실제: %s", health.Status)
	}
}

func TestDockerAdapter_GetContainerMetrics(t *testing.T) {
	// Arrange
	containerManager := &MockContainerManager{}
	client := &MockDockerClient{}
	statsCollector := &MockStatsCollector{}
	adapter := NewDockerAdapter(containerManager, client, statsCollector)

	// Act
	metrics, err := adapter.GetContainerMetrics(context.Background(), "test-container-id")

	// Assert
	if err != nil {
		t.Fatalf("GetContainerMetrics 실패: %v", err)
	}

	if metrics.ContainerID != "test-container-id" {
		t.Errorf("예상 컨테이너 ID: test-container-id, 실제: %s", metrics.ContainerID)
	}
}

func TestDockerAdapter_GetContainerLogs(t *testing.T) {
	// Arrange
	containerManager := &MockContainerManager{}
	client := &MockDockerClient{}
	statsCollector := &MockStatsCollector{}
	adapter := NewDockerAdapter(containerManager, client, statsCollector)

	opts := LogOptions{
		Follow:     false,
		Timestamps: true,
		Tail:       100,
	}

	// Act
	logStream, err := adapter.GetContainerLogs(context.Background(), "test-container-id", opts)

	// Assert
	if err != nil {
		t.Fatalf("GetContainerLogs 실패: %v", err)
	}

	if logStream == nil {
		t.Fatal("로그 스트림이 nil입니다")
	}

	// 로그 읽기 테스트
	data, err := logStream.Read()
	if err != nil && err != io.EOF {
		t.Fatalf("로그 읽기 실패: %v", err)
	}

	if len(data) == 0 {
		t.Error("로그 데이터가 비어있습니다")
	}

	// 스트림 종료
	err = logStream.Close()
	if err != nil {
		t.Fatalf("로그 스트림 종료 실패: %v", err)
	}
}

func TestDockerAdapter_ExecuteCommand(t *testing.T) {
	// Arrange
	containerManager := &MockContainerManager{}
	client := &MockDockerClient{}
	statsCollector := &MockStatsCollector{}
	adapter := NewDockerAdapter(containerManager, client, statsCollector)

	// Act
	result, err := adapter.ExecuteCommand(context.Background(), "test-container-id", []string{"echo", "test"})

	// Assert
	if err != nil {
		t.Fatalf("ExecuteCommand 실패: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("예상 종료 코드: 0, 실제: %d", result.ExitCode)
	}
}