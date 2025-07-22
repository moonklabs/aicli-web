package docker

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	
	assert.NotNil(t, config)
	assert.NotEmpty(t, config.Host)
	assert.NotEmpty(t, config.Version)
	assert.NotZero(t, config.Timeout)
	assert.NotEmpty(t, config.DefaultImage)
	assert.NotEmpty(t, config.DefaultShell)
	assert.NotEmpty(t, config.NetworkName)
	assert.Greater(t, config.CPULimit, 0.0)
	assert.Greater(t, config.MemoryLimit, int64(0))
	assert.False(t, config.Privileged)
	assert.True(t, config.ReadOnly)
	assert.NotEmpty(t, config.SecurityOpts)
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
	}{
		{
			name:        "nil config should use defaults",
			config:      nil,
			expectError: false,
		},
		{
			name:        "default config should be valid",
			config:      DefaultConfig(),
			expectError: false,
		},
		{
			name: "custom valid config",
			config: &Config{
				Host:         "unix:///var/run/docker.sock",
				Version:      "1.41",
				Timeout:      10 * time.Second,
				DefaultImage: "alpine:latest",
				DefaultShell: []string{"/bin/sh"},
				NetworkName:  "test-network",
				CPULimit:     2.0,
				MemoryLimit:  1024 * 1024 * 1024,
				Privileged:   false,
				ReadOnly:     true,
				SecurityOpts: []string{"no-new-privileges:true"},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := tt.config
			if config == nil {
				config = DefaultConfig()
			}
			
			// 설정 검증 로직 (실제 Docker 연결은 건너뜀)
			assert.NotEmpty(t, config.Host)
			assert.NotEmpty(t, config.Version)
			assert.Greater(t, config.Timeout, time.Duration(0))
		})
	}
}

func TestClient_LabelKey(t *testing.T) {
	config := DefaultConfig()
	client := &Client{
		config:      config,
		labelPrefix: "aicli",
	}

	tests := []struct {
		name     string
		key      string
		expected string
	}{
		{
			name:     "simple key",
			key:      "managed",
			expected: "aicli.managed",
		},
		{
			name:     "workspace id",
			key:      "workspace.id",
			expected: "aicli.workspace.id",
		},
		{
			name:     "empty key",
			key:      "",
			expected: "aicli.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := client.labelKey(tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestClient_WorkspaceLabels(t *testing.T) {
	client := &Client{
		labelPrefix: "aicli",
	}

	workspaceID := "ws-12345"
	name := "test-workspace"
	
	labels := client.WorkspaceLabels(workspaceID, name)
	
	assert.NotNil(t, labels)
	assert.Equal(t, "true", labels["aicli.managed"])
	assert.Equal(t, workspaceID, labels["aicli.workspace.id"])
	assert.Equal(t, name, labels["aicli.workspace.name"])
	assert.NotEmpty(t, labels["aicli.created"])
	
	// 시간 형식 검증
	_, err := time.Parse(time.RFC3339, labels["aicli.created"])
	assert.NoError(t, err)
}

func TestClient_GenerateName(t *testing.T) {
	client := &Client{
		config: &Config{
			NetworkName: "aicli-network",
		},
	}

	workspaceID := "ws-12345"

	// 이미지 태그 생성
	imageTag := client.GenerateImageTag(workspaceID)
	expected := "aicli-workspace:ws-12345"
	assert.Equal(t, expected, imageTag)

	// 컨테이너 이름 생성
	containerName := client.GenerateContainerName(workspaceID)
	expected = "workspace_ws-12345"
	assert.Equal(t, expected, containerName)

	// 네트워크 이름 생성
	networkName := client.GenerateNetworkName("")
	assert.Equal(t, "aicli-network", networkName)

	networkName = client.GenerateNetworkName("test")
	assert.Equal(t, "aicli-network_test", networkName)
}

// Docker daemon이 실행 중인 환경에서만 실행되는 통합 테스트
func TestClient_Integration(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker daemon not available")
	}

	config := DefaultConfig()
	client, err := NewClient(config)
	require.NoError(t, err)
	require.NotNil(t, client)
	defer client.Close()

	ctx := context.Background()

	// Ping 테스트
	err = client.Ping(ctx)
	assert.NoError(t, err)

	// 네트워크 ID 확인
	assert.NotEmpty(t, client.GetNetworkID())

	// 설정 확인
	receivedConfig := client.GetConfig()
	assert.Equal(t, config, receivedConfig)
}

func TestClient_NetworkSetup(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker daemon not available")
	}

	config := DefaultConfig()
	config.NetworkName = "test-aicli-network"

	client, err := NewClient(config)
	require.NoError(t, err)
	require.NotNil(t, client)
	defer func() {
		// 테스트 네트워크 정리
		if client.networkID != "" {
			ctx := context.Background()
			client.cli.NetworkRemove(ctx, client.networkID)
		}
		client.Close()
	}()

	// 네트워크가 생성되었는지 확인
	assert.NotEmpty(t, client.GetNetworkID())

	// 네트워크 정보 확인
	ctx := context.Background()
	inspect, err := client.cli.NetworkInspect(ctx, client.networkID, map[string]bool{})
	require.NoError(t, err)
	
	assert.Equal(t, config.NetworkName, inspect.Name)
	assert.Equal(t, "bridge", inspect.Driver)
	assert.True(t, inspect.Attachable)
	assert.False(t, inspect.Internal)
	
	// 레이블 확인
	assert.Equal(t, "true", inspect.Labels[client.labelKey("managed")])
	assert.NotEmpty(t, inspect.Labels[client.labelKey("created")])
}

// isDockerAvailable Docker daemon이 사용 가능한지 확인합니다.
func isDockerAvailable() bool {
	config := DefaultConfig()
	client, err := NewClient(config)
	if err != nil {
		return false
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = client.Ping(ctx)
	return err == nil
}

func BenchmarkClient_LabelKey(b *testing.B) {
	client := &Client{
		labelPrefix: "aicli",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = client.labelKey("workspace.id")
	}
}

func BenchmarkClient_WorkspaceLabels(b *testing.B) {
	client := &Client{
		labelPrefix: "aicli",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = client.WorkspaceLabels("ws-12345", "test-workspace")
	}
}