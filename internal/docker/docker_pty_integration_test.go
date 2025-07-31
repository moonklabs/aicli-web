package docker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockClientInterface Docker 클라이언트 모의 객체
type MockClientInterface struct {
	mock.Mock
}

func (m *MockClientInterface) ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error) {
	args := m.Called(ctx, containerID)
	return args.Get(0).(types.ContainerJSON), args.Error(1)
}

func (m *MockClientInterface) ContainerExecCreate(ctx context.Context, containerID string, config ExecConfig) (types.IDResponse, error) {
	args := m.Called(ctx, containerID, config)
	return args.Get(0).(types.IDResponse), args.Error(1)
}

func (m *MockClientInterface) ContainerExecStart(ctx context.Context, execID string, config ExecStartConfig) (types.HijackedResponse, error) {
	args := m.Called(ctx, execID, config)
	return args.Get(0).(types.HijackedResponse), args.Error(1)
}

func (m *MockClientInterface) ContainerExecInspect(ctx context.Context, execID string) (types.ContainerExecInspect, error) {
	args := m.Called(ctx, execID)
	return args.Get(0).(types.ContainerExecInspect), args.Error(1)
}

func (m *MockClientInterface) Events(ctx context.Context, options types.EventsOptions) (<-chan events.Message, <-chan error) {
	args := m.Called(ctx, options)
	return args.Get(0).(<-chan events.Message), args.Get(1).(<-chan error)
}

func (m *MockClientInterface) ContainerKill(ctx context.Context, containerID string, signal string) error {
	args := m.Called(ctx, containerID, signal)
	return args.Error(0)
}

func (m *MockClientInterface) ContainerRemove(ctx context.Context, containerID string, options types.ContainerRemoveOptions) error {
	args := m.Called(ctx, containerID, options)
	return args.Error(0)
}

func (m *MockClientInterface) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockClientInterface) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockClientInterface) GetConfig() *Config {
	args := m.Called()
	return args.Get(0).(*Config)
}

func (m *MockClientInterface) GetNetworkID() string {
	args := m.Called()
	return args.String(0)
}

// MockPTYSessionManagement PTY 세션 관리 모의 객체
type MockPTYSessionManagement struct {
	mock.Mock
}

func (m *MockPTYSessionManagement) CreateSession(ctx context.Context, containerID string) (PTYSession, error) {
	args := m.Called(ctx, containerID)
	return args.Get(0).(PTYSession), args.Error(1)
}

func (m *MockPTYSessionManagement) GetSession(sessionID string) (PTYSession, error) {
	args := m.Called(sessionID)
	return args.Get(0).(PTYSession), args.Error(1)
}

func (m *MockPTYSessionManagement) ListSessions() []PTYSession {
	args := m.Called()
	return args.Get(0).([]PTYSession)
}

func (m *MockPTYSessionManagement) RemoveSession(sessionID string) error {
	args := m.Called(sessionID)
	return args.Error(0)
}

func (m *MockPTYSessionManagement) GetSessionCount() int {
	args := m.Called()
	return args.Int(0)
}

func (m *MockPTYSessionManagement) GetSessionsByContainer(containerID string) []PTYSession {
	args := m.Called(containerID)
	return args.Get(0).([]PTYSession)
}

func (m *MockPTYSessionManagement) GetStats() *PTYSessionStats {
	args := m.Called()
	return args.Get(0).(*PTYSessionStats)
}

func (m *MockPTYSessionManagement) Shutdown() error {
	args := m.Called()
	return args.Error(0)
}

// TestNewContainerPTYManager 컨테이너 PTY 관리자 생성 테스트
func TestNewContainerPTYManager(t *testing.T) {
	mockClient := &MockClientInterface{}
	mockPTYManager := &MockPTYSessionManagement{}

	manager := NewContainerPTYManager(mockClient, mockPTYManager)

	assert.NotNil(t, manager)
	assert.Equal(t, mockClient, manager.dockerClient)
	assert.Equal(t, mockPTYManager, manager.ptyManager)
	assert.NotNil(t, manager.containerSessions)
	assert.NotNil(t, manager.execResults)
	assert.NotNil(t, manager.monitors)
}

// TestValidateContainer 컨테이너 유효성 검증 테스트
func TestValidateContainer(t *testing.T) {
	mockClient := &MockClientInterface{}
	mockPTYManager := &MockPTYSessionManagement{}
	manager := NewContainerPTYManager(mockClient, mockPTYManager)

	ctx := context.Background()
	containerID := "test-container"

	// 성공 케이스
	runningContainer := types.ContainerJSON{
		ContainerJSONBase: &types.ContainerJSONBase{
			State: &types.ContainerState{
				Running: true,
				Status:  "running",
			},
		},
	}

	mockClient.On("ContainerInspect", ctx, containerID).Return(runningContainer, nil)

	err := manager.ValidateContainer(ctx, containerID)
	assert.NoError(t, err)

	// 컨테이너가 없는 경우
	mockClient.On("ContainerInspect", ctx, "nonexistent").Return(types.ContainerJSON{}, errors.New("container not found"))

	err = manager.ValidateContainer(ctx, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "container not found")

	// 컨테이너가 실행 중이 아닌 경우
	stoppedContainer := types.ContainerJSON{
		ContainerJSONBase: &types.ContainerJSONBase{
			State: &types.ContainerState{
				Running: false,
				Status:  "exited",
			},
		},
	}

	mockClient.On("ContainerInspect", ctx, "stopped-container").Return(stoppedContainer, nil)

	err = manager.ValidateContainer(ctx, "stopped-container")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not running")

	mockClient.AssertExpectations(t)
}

// TestDetectShell 쉘 감지 테스트
func TestDetectShell(t *testing.T) {
	mockClient := &MockClientInterface{}
	mockPTYManager := &MockPTYSessionManagement{}
	manager := NewContainerPTYManager(mockClient, mockPTYManager)

	ctx := context.Background()
	containerID := "test-container"

	// bash가 있는 경우
	mockClient.On("ContainerExecCreate", ctx, containerID, mock.MatchedBy(func(config ExecConfig) bool {
		return len(config.Cmd) == 3 && config.Cmd[0] == "test" && config.Cmd[1] == "-x" && config.Cmd[2] == "/bin/bash"
	})).Return(types.IDResponse{ID: "exec-bash"}, nil)

	mockClient.On("ContainerExecStart", ctx, "exec-bash", mock.AnythingOfType("ExecStartConfig")).Return(types.HijackedResponse{}, nil)

	mockClient.On("ContainerExecInspect", ctx, "exec-bash").Return(types.ContainerExecInspect{ExitCode: 0}, nil)

	shell, err := manager.DetectShell(ctx, containerID)
	assert.NoError(t, err)
	assert.Equal(t, "/bin/bash", shell)

	mockClient.AssertExpectations(t)
}

// TestExecuteCommand 명령 실행 테스트
func TestExecuteCommand(t *testing.T) {
	mockClient := &MockClientInterface{}
	mockPTYManager := &MockPTYSessionManagement{}
	manager := NewContainerPTYManager(mockClient, mockPTYManager)

	ctx := context.Background()
	containerID := "test-container"
	cmd := []string{"echo", "hello"}

	// 성공 케이스
	mockClient.On("ContainerExecCreate", ctx, containerID, mock.MatchedBy(func(config ExecConfig) bool {
		return len(config.Cmd) == 2 && config.Cmd[0] == "echo" && config.Cmd[1] == "hello"
	})).Return(types.IDResponse{ID: "exec-echo"}, nil)

	mockHijacked := &MockHijackedResponse{
		data: []byte("hello\n"),
	}
	mockClient.On("ContainerExecStart", ctx, "exec-echo", mock.AnythingOfType("ExecStartConfig")).Return(types.HijackedResponse{
		Reader: mockHijacked,
	}, nil)

	mockClient.On("ContainerExecInspect", ctx, "exec-echo").Return(types.ContainerExecInspect{ExitCode: 0}, nil)

	result, err := manager.ExecuteCommand(ctx, containerID, cmd)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 0, result.ExitCode)
	assert.Equal(t, "hello\n", result.Output)
	assert.Greater(t, result.Duration, time.Duration(0))

	mockClient.AssertExpectations(t)
}

// TestMonitorContainer 컨테이너 모니터링 테스트
func TestMonitorContainer(t *testing.T) {
	mockClient := &MockClientInterface{}
	mockPTYManager := &MockPTYSessionManagement{}
	manager := NewContainerPTYManager(mockClient, mockPTYManager)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	containerID := "test-container"

	// 이벤트 채널 생성
	eventChan := make(chan events.Message, 1)
	errChan := make(chan error, 1)

	mockClient.On("Events", ctx, mock.MatchedBy(func(options types.EventsOptions) bool {
		return len(options.Filters["container"]) == 1 && options.Filters["container"][0] == containerID
	})).Return((<-chan events.Message)(eventChan), (<-chan error)(errChan))

	// 모니터링 시작
	resultChan, err := manager.MonitorContainer(ctx, containerID)
	assert.NoError(t, err)
	assert.NotNil(t, resultChan)

	// 테스트 이벤트 전송
	go func() {
		eventChan <- events.Message{
			Type:   "container",
			Action: "start",
			Actor: events.Actor{
				ID: containerID,
				Attributes: map[string]string{
					"name": "test-container",
				},
			},
			Time: time.Now().Unix(),
		}
		close(eventChan)
		close(errChan)
	}()

	// 이벤트 수신 확인
	select {
	case event := <-resultChan:
		assert.Equal(t, "start", event.Type)
		assert.Equal(t, containerID, event.ContainerID)
		assert.Equal(t, "test-container", event.Attributes["name"])
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for event")
	}

	mockClient.AssertExpectations(t)
}

// TestCleanupInactiveSessions 비활성 세션 정리 테스트
func TestCleanupInactiveSessions(t *testing.T) {
	mockClient := &MockClientInterface{}
	mockPTYManager := &MockPTYSessionManagement{}
	manager := NewContainerPTYManager(mockClient, mockPTYManager)

	containerID := "test-container"

	// 모의 세션 생성
	mockActiveSession := &MockPTYSession{alive: true}
	mockInactiveSession := &MockPTYSession{alive: false}

	// 세션 추가
	manager.containerSessions[containerID] = []PTYSession{
		mockActiveSession,
		mockInactiveSession,
	}

	// 정리 실행
	manager.CleanupInactiveSessions()

	// 활성 세션만 남아있는지 확인
	sessions := manager.containerSessions[containerID]
	assert.Len(t, sessions, 1)
	assert.Equal(t, mockActiveSession, sessions[0])

	// 비활성 세션의 Stop이 호출되었는지 확인
	assert.True(t, mockInactiveSession.stopCalled)
}

// TestGetStats 통계 조회 테스트
func TestGetStats(t *testing.T) {
	mockClient := &MockClientInterface{}
	mockPTYManager := &MockPTYSessionManagement{}
	manager := NewContainerPTYManager(mockClient, mockPTYManager)

	container1 := "container1"
	container2 := "container2"

	// 모의 세션 추가
	manager.containerSessions[container1] = []PTYSession{
		&MockPTYSession{alive: true},
		&MockPTYSession{alive: false},
	}
	manager.containerSessions[container2] = []PTYSession{
		&MockPTYSession{alive: true},
	}

	// 모니터링 컨테이너 추가
	manager.monitors[container1] = make(chan ContainerEvent)

	stats := manager.GetStats()

	assert.Equal(t, 2, stats.TotalContainers)
	assert.Equal(t, 3, stats.TotalSessions)
	assert.Equal(t, 2, stats.ActiveSessions)
	assert.Equal(t, 1, stats.ContainerCounts[container1])
	assert.Equal(t, 1, stats.ContainerCounts[container2])
	assert.Equal(t, 1, stats.MonitoredContainers)
}

// MockHijackedResponse HijackedResponse 모의 객체
type MockHijackedResponse struct {
	data []byte
	pos  int
}

func (m *MockHijackedResponse) Read(p []byte) (n int, error) {
	if m.pos >= len(m.data) {
		return 0, nil // EOF
	}

	n = copy(p, m.data[m.pos:])
	m.pos += n
	return n, nil
}

func (m *MockHijackedResponse) Close() error {
	return nil
}

// MockPTYSession PTY 세션 모의 객체
type MockPTYSession struct {
	alive      bool
	stopCalled bool
}

func (m *MockPTYSession) ID() string                    { return "mock-session" }
func (m *MockPTYSession) ContainerID() string           { return "mock-container" }
func (m *MockPTYSession) Start(ctx context.Context) error { return nil }
func (m *MockPTYSession) Write(data []byte) (int, error) { return len(data), nil }
func (m *MockPTYSession) Read(data []byte) (int, error)  { return 0, nil }
func (m *MockPTYSession) Resize(width, height int) error { return nil }
func (m *MockPTYSession) IsAlive() bool                  { return m.alive }
func (m *MockPTYSession) GetCreatedAt() time.Time        { return time.Now() }
func (m *MockPTYSession) GetLastActivity() time.Time     { return time.Now() }

func (m *MockPTYSession) Stop() error {
	m.stopCalled = true
	m.alive = false
	return nil
}

// BenchmarkCreateContainerPTY PTY 생성 성능 테스트
func BenchmarkCreateContainerPTY(b *testing.B) {
	mockClient := &MockClientInterface{}
	mockPTYManager := &MockPTYSessionManagement{}
	manager := NewContainerPTYManager(mockClient, mockPTYManager)

	ctx := context.Background()
	containerID := "benchmark-container"
	config := PTYConfig{
		Shell:      "/bin/bash",
		WorkingDir: "/workspace",
		Size:       PTYSize{Width: 80, Height: 24},
		User:       "root",
		Tty:        true,
	}

	// 모의 객체 설정
	runningContainer := types.ContainerJSON{
		ContainerJSONBase: &types.ContainerJSONBase{
			State: &types.ContainerState{
				Running: true,
				Status:  "running",
			},
		},
	}

	mockClient.On("ContainerInspect", ctx, containerID).Return(runningContainer, nil)

	mockSession := &MockPTYSession{alive: true}
	mockPTYManager.On("CreateSession", ctx, containerID).Return(mockSession, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := manager.CreateContainerPTY(ctx, containerID, config)
		if err != nil {
			b.Fatal(err)
		}
	}
}