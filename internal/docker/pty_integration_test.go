package docker

import (
	"context"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockDockerClient is a mock Docker client for testing
type MockDockerClient struct {
	mock.Mock
}

func TestNewDockerPTYIntegration(t *testing.T) {
	client, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		t.Skip("Docker not available")
	}

	config := &DockerPTYConfig{
		MaxSessions:         10,
		MonitorInterval:     1 * time.Second,
		ReconnectTimeout:    5 * time.Second,
		HealthCheckInterval: 1 * time.Second,
	}

	dpi := NewDockerPTYIntegration(client, config)
	assert.NotNil(t, dpi)
	assert.Equal(t, 10, dpi.config.MaxSessions)
	assert.NotNil(t, dpi.sessions)
	assert.NotNil(t, dpi.monitor)
}

func TestDockerPTYIntegration_ConnectContainer(t *testing.T) {
	// Docker가 설치되어 있지 않으면 테스트 건너뛰기
	client, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		t.Skip("Docker not available")
	}

	// 테스트 컨테이너 생성
	ctx := context.Background()
	resp, err := client.ContainerCreate(ctx, &container.Config{
		Image: "alpine:latest",
		Cmd:   []string{"sleep", "300"},
	}, nil, nil, nil, "")
	if err != nil {
		t.Skip("Failed to create test container:", err)
	}
	defer client.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{Force: true})

	// 컨테이너 시작
	if err := client.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		t.Fatal("Failed to start container:", err)
	}

	// PTY 통합 생성
	dpi := NewDockerPTYIntegration(client, nil)
	dpi.Start()
	defer dpi.Stop()

	// PTY 연결 테스트
	config := &PTYSessionConfig{
		Shell:        "/bin/sh",
		Term:         "xterm",
		Rows:         24,
		Cols:         80,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
	}

	session, err := dpi.ConnectContainer(ctx, resp.ID, config)
	require.NoError(t, err)
	assert.NotNil(t, session)
	assert.Equal(t, resp.ID, session.ContainerID)
	assert.Equal(t, PTYActive, session.Status)

	// 세션 정리
	err = dpi.DisconnectContainer(session.SessionID)
	assert.NoError(t, err)
}

func TestDockerPTYIntegration_ListActiveSessions(t *testing.T) {
	client, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		t.Skip("Docker not available")
	}

	dpi := NewDockerPTYIntegration(client, nil)

	// 초기 상태
	sessions := dpi.ListActiveSessions()
	assert.Empty(t, sessions)

	// 테스트 세션 추가
	testSession := &DockerPTYSession{
		SessionID:   "test-session-1",
		ContainerID: "test-container-1",
		Status:      PTYActive,
		CreatedAt:   time.Now(),
	}
	dpi.registerSession(testSession)

	sessions = dpi.ListActiveSessions()
	assert.Len(t, sessions, 1)
	assert.Equal(t, "test-session-1", sessions["test-session-1"].SessionID)
}

func TestDockerPTYIntegration_BuildEnvironment(t *testing.T) {
	dpi := &DockerPTYIntegration{}

	tests := []struct {
		name      string
		customEnv map[string]string
		expected  []string
	}{
		{
			name:      "Default environment",
			customEnv: nil,
			expected: []string{
				"TERM=xterm-256color",
				"LANG=en_US.UTF-8",
				"LC_ALL=en_US.UTF-8",
			},
		},
		{
			name: "Custom environment",
			customEnv: map[string]string{
				"NODE_ENV": "production",
				"API_KEY":  "secret",
			},
			expected: []string{
				"NODE_ENV=production",
				"API_KEY=secret",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := dpi.buildEnvironment(tt.customEnv)

			for _, expected := range tt.expected {
				assert.Contains(t, result, expected)
			}
		})
	}
}

func TestContainerMonitor_NewContainerMonitor(t *testing.T) {
	client, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		t.Skip("Docker not available")
	}

	monitor := NewContainerMonitor(client, 2*time.Second)
	assert.NotNil(t, monitor)
	assert.Equal(t, 2*time.Second, monitor.config.MonitorInterval)
	assert.NotNil(t, monitor.sessions)
	assert.NotNil(t, monitor.stats)
}

func TestImageOptimizer_GetProfileForImage(t *testing.T) {
	optimizer := NewImageOptimizer()

	tests := []struct {
		name      string
		imageName string
		expected  string
	}{
		{
			name:      "Node.js image",
			imageName: "node:14",
			expected:  "/bin/bash",
		},
		{
			name:      "Python image",
			imageName: "python:3.9",
			expected:  "/bin/bash",
		},
		{
			name:      "Alpine image",
			imageName: "alpine:3.14",
			expected:  "/bin/sh",
		},
		{
			name:      "Unknown image",
			imageName: "unknown:latest",
			expected:  "/bin/sh",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := optimizer.GetProfileForImage(tt.imageName)
			assert.Equal(t, tt.expected, profile.DefaultShell)
		})
	}
}

func TestReconnectionManager_QueueReconnection(t *testing.T) {
	client, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		t.Skip("Docker not available")
	}

	dpi := NewDockerPTYIntegration(client, nil)
	rm := NewReconnectionManager(dpi)

	// 테스트 세션
	session := &DockerPTYSession{
		SessionID:   "test-session",
		ContainerID: "test-container",
		Status:      PTYActive,
	}

	// 재연결 큐에 추가
	err = rm.QueueReconnection(session)
	assert.NoError(t, err)
	assert.Equal(t, 1, rm.GetQueueSize())
}

func TestResourceMonitor_NewResourceMonitor(t *testing.T) {
	client, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		t.Skip("Docker not available")
	}

	monitor := NewResourceMonitor(client)
	assert.NotNil(t, monitor)
	assert.NotNil(t, monitor.containerLimits)
	assert.NotNil(t, monitor.history)
	assert.NotNil(t, monitor.alerts)
}

func TestResourceMonitor_SetContainerLimits(t *testing.T) {
	client, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		t.Skip("Docker not available")
	}

	monitor := NewResourceMonitor(client)

	limits := &PTYResourceLimits{
		CPULimit:    2.0,
		MemoryLimit: 1024 * 1024 * 1024,
		PIDs:        100,
	}

	monitor.SetContainerLimits("test-container", limits)

	monitor.mutex.RLock()
	storedLimits := monitor.containerLimits["test-container"]
	monitor.mutex.RUnlock()

	assert.NotNil(t, storedLimits)
	assert.Equal(t, 2.0, storedLimits.CPULimit)
}

// Benchmark tests
func BenchmarkDockerPTYIntegration_BuildEnvironment(b *testing.B) {
	dpi := &DockerPTYIntegration{}
	customEnv := map[string]string{
		"VAR1": "value1",
		"VAR2": "value2",
		"VAR3": "value3",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = dpi.buildEnvironment(customEnv)
	}
}

func BenchmarkImageOptimizer_GetProfileForImage(b *testing.B) {
	optimizer := NewImageOptimizer()
	images := []string{
		"node:14",
		"python:3.9",
		"alpine:3.14",
		"ubuntu:20.04",
		"nginx:latest",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, img := range images {
			_ = optimizer.GetProfileForImage(img)
		}
	}
}