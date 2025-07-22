package security

import (
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aicli/aicli-web/internal/models"
)

func TestNewIsolationManager(t *testing.T) {
	manager := NewIsolationManager()
	
	assert.NotNil(t, manager)
	assert.NotNil(t, manager.config)
	assert.True(t, manager.config.EnableNetworkIsolation)
	assert.True(t, manager.config.DisablePrivileged)
}

func TestNewIsolationManagerWithConfig(t *testing.T) {
	config := &IsolationConfig{
		EnableNetworkIsolation: false,
		DefaultCPULimit:        2.0,
		DefaultMemoryLimit:     1024 * 1024 * 1024,
	}
	
	manager := NewIsolationManagerWithConfig(config)
	
	assert.NotNil(t, manager)
	assert.Equal(t, config, manager.config)
	assert.False(t, manager.config.EnableNetworkIsolation)
	assert.Equal(t, 2.0, manager.config.DefaultCPULimit)
}

func TestDefaultIsolationConfig(t *testing.T) {
	config := DefaultIsolationConfig()
	
	assert.NotNil(t, config)
	assert.True(t, config.EnableNetworkIsolation)
	assert.True(t, config.EnableSeccomp)
	assert.True(t, config.EnableAppArmor)
	assert.True(t, config.DisablePrivileged)
	assert.True(t, config.NoNewPrivileges)
	assert.False(t, config.ReadOnlyRootFS) // 워크스페이스는 쓰기 가능
	assert.Equal(t, 1.0, config.DefaultCPULimit)
	assert.Equal(t, int64(512*1024*1024), config.DefaultMemoryLimit)
	assert.Contains(t, config.BlockedPorts, 22)
	assert.Contains(t, config.BlockedPorts, 80)
}

func TestCreateWorkspaceIsolation(t *testing.T) {
	manager := NewIsolationManager()
	workspace := &models.Workspace{
		ID:   "test-workspace-123",
		Name: "Test Workspace",
	}
	
	isolation, err := manager.CreateWorkspaceIsolation(workspace)
	
	require.NoError(t, err)
	assert.NotNil(t, isolation)
	assert.Equal(t, workspace.ID, isolation.WorkspaceID)
	assert.Equal(t, "custom", isolation.NetworkMode)
	assert.Equal(t, "aicli-workspace-test-workspace-123", isolation.NetworkName)
	assert.Equal(t, IsolationLevelStandard, isolation.IsolationLevel)
	assert.NotNil(t, isolation.ResourceLimits)
	assert.NotNil(t, isolation.SecurityOptions)
	assert.NotNil(t, isolation.MonitoringConfig)
	assert.WithinDuration(t, time.Now(), isolation.CreatedAt, time.Second)
}

func TestCreateWorkspaceIsolation_NilWorkspace(t *testing.T) {
	manager := NewIsolationManager()
	
	isolation, err := manager.CreateWorkspaceIsolation(nil)
	
	assert.Error(t, err)
	assert.Nil(t, isolation)
	assert.Contains(t, err.Error(), "workspace cannot be nil")
}

func TestCreateResourceLimits(t *testing.T) {
	manager := NewIsolationManager()
	
	limits := manager.createResourceLimits()
	
	assert.NotNil(t, limits)
	assert.Equal(t, int64(1024), limits.CPUShares)
	assert.Equal(t, int64(100000), limits.CPUQuota) // 1.0 CPU * 100000
	assert.Equal(t, int64(100000), limits.CPUPeriod)
	assert.Equal(t, int64(512*1024*1024), limits.Memory)
	assert.Equal(t, int64(512*1024*1024), limits.MemorySwap)
	assert.Equal(t, int64(100), limits.PidsLimit)
	assert.Equal(t, "100m", limits.IOMaxBandwidth)
	assert.Equal(t, int64(1000), limits.IOMaxIOps)
}

func TestCreateSecurityOptions(t *testing.T) {
	manager := NewIsolationManager()
	
	opts := manager.createSecurityOptions()
	
	assert.NotNil(t, opts)
	assert.True(t, opts.NoNewPrivileges)
	assert.False(t, opts.ReadOnlyRootFS) // 워크스페이스는 쓰기 가능
	assert.Equal(t, "default", opts.SeccompProfile)
	assert.Equal(t, "docker-default", opts.AppArmorProfile)
	assert.NotNil(t, opts.Capabilities)
	assert.Contains(t, opts.Capabilities.Drop, "ALL")
	assert.Contains(t, opts.Capabilities.Add, "CHOWN")
	assert.Contains(t, opts.Capabilities.Add, "SETUID")
}

func TestCreateMonitoringConfig(t *testing.T) {
	manager := NewIsolationManager()
	
	config := manager.createMonitoringConfig()
	
	assert.NotNil(t, config)
	assert.True(t, config.EnableResourceMonitoring)
	assert.True(t, config.EnableNetworkMonitoring)
	assert.True(t, config.EnableFileSystemAudit)
	assert.Equal(t, "info", config.LogLevel)
	assert.NotNil(t, config.AlertThresholds)
	assert.Equal(t, 85.0, config.AlertThresholds.CPUThreshold)
	assert.Equal(t, 90.0, config.AlertThresholds.MemoryThreshold)
}

func TestApplyToContainer(t *testing.T) {
	manager := NewIsolationManager()
	workspace := &models.Workspace{ID: "test-workspace", Name: "Test"}
	isolation, err := manager.CreateWorkspaceIsolation(workspace)
	require.NoError(t, err)
	
	config := &container.Config{}
	hostConfig := &container.HostConfig{}
	
	err = manager.ApplyToContainer(isolation, config, hostConfig)
	
	require.NoError(t, err)
	assert.Equal(t, container.NetworkMode(isolation.NetworkName), hostConfig.NetworkMode)
	assert.Equal(t, int64(1024), hostConfig.Resources.CPUShares)
	assert.Equal(t, int64(100000), hostConfig.Resources.CPUQuota)
	assert.Equal(t, int64(512*1024*1024), hostConfig.Resources.Memory)
	assert.Contains(t, hostConfig.SecurityOpt, "no-new-privileges:true")
	assert.Contains(t, hostConfig.SecurityOpt, "seccomp=default")
	assert.Contains(t, hostConfig.SecurityOpt, "apparmor=docker-default")
	assert.Contains(t, hostConfig.CapDrop, "ALL")
	assert.Contains(t, hostConfig.CapAdd, "CHOWN")
}

func TestApplyToContainer_NilParams(t *testing.T) {
	manager := NewIsolationManager()
	
	tests := []struct {
		name         string
		isolation    *WorkspaceIsolation
		config       *container.Config
		hostConfig   *container.HostConfig
		expectError  string
	}{
		{
			name:        "nil isolation",
			isolation:   nil,
			config:      &container.Config{},
			hostConfig:  &container.HostConfig{},
			expectError: "isolation config cannot be nil",
		},
		{
			name:        "nil container config",
			isolation:   &WorkspaceIsolation{},
			config:      nil,
			hostConfig:  &container.HostConfig{},
			expectError: "container config cannot be nil",
		},
		{
			name:        "nil host config",
			isolation:   &WorkspaceIsolation{},
			config:      &container.Config{},
			hostConfig:  nil,
			expectError: "host config cannot be nil",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.ApplyToContainer(tt.isolation, tt.config, tt.hostConfig)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectError)
		})
	}
}

func TestValidateIsolation(t *testing.T) {
	manager := NewIsolationManager()
	
	tests := []struct {
		name        string
		isolation   *WorkspaceIsolation
		expectError bool
		errorMsg    string
	}{
		{
			name:        "nil isolation",
			isolation:   nil,
			expectError: true,
			errorMsg:    "isolation config cannot be nil",
		},
		{
			name: "empty workspace ID",
			isolation: &WorkspaceIsolation{
				WorkspaceID: "",
			},
			expectError: true,
			errorMsg:    "workspace ID cannot be empty",
		},
		{
			name: "empty network mode",
			isolation: &WorkspaceIsolation{
				WorkspaceID: "test",
				NetworkMode: "",
			},
			expectError: true,
			errorMsg:    "network mode cannot be empty",
		},
		{
			name: "nil resource limits",
			isolation: &WorkspaceIsolation{
				WorkspaceID:    "test",
				NetworkMode:    "custom",
				ResourceLimits: nil,
			},
			expectError: true,
			errorMsg:    "resource limits cannot be nil",
		},
		{
			name: "invalid memory limit",
			isolation: &WorkspaceIsolation{
				WorkspaceID: "test",
				NetworkMode: "custom",
				ResourceLimits: &ResourceLimits{
					Memory: -1,
				},
			},
			expectError: true,
			errorMsg:    "memory limit must be positive",
		},
		{
			name: "nil security options",
			isolation: &WorkspaceIsolation{
				WorkspaceID: "test",
				NetworkMode: "custom",
				ResourceLimits: &ResourceLimits{
					Memory: 512 * 1024 * 1024,
				},
				SecurityOptions: nil,
			},
			expectError: true,
			errorMsg:    "security options cannot be nil",
		},
		{
			name: "valid isolation",
			isolation: &WorkspaceIsolation{
				WorkspaceID: "test",
				NetworkMode: "custom",
				ResourceLimits: &ResourceLimits{
					Memory: 512 * 1024 * 1024,
				},
				SecurityOptions: &SecurityOptions{},
			},
			expectError: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.ValidateIsolation(tt.isolation)
			
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestHashWorkspaceID(t *testing.T) {
	manager := NewIsolationManager()
	
	// 같은 ID는 항상 같은 해시를 생성
	id1 := "test-workspace-123"
	hash1a := manager.hashWorkspaceID(id1)
	hash1b := manager.hashWorkspaceID(id1)
	assert.Equal(t, hash1a, hash1b)
	
	// 다른 ID는 다른 해시를 생성
	id2 := "test-workspace-456"
	hash2 := manager.hashWorkspaceID(id2)
	assert.NotEqual(t, hash1a, hash2)
	
	// 빈 문자열도 처리 가능
	emptyHash := manager.hashWorkspaceID("")
	assert.NotEqual(t, uint32(0), emptyHash)
}

func TestUpdateConfig(t *testing.T) {
	manager := NewIsolationManager()
	originalConfig := manager.config
	
	newConfig := &IsolationConfig{
		EnableNetworkIsolation: false,
		DefaultCPULimit:        2.0,
		DefaultMemoryLimit:     1024 * 1024 * 1024,
	}
	
	err := manager.UpdateConfig(newConfig)
	
	require.NoError(t, err)
	assert.NotEqual(t, originalConfig, manager.config)
	assert.Equal(t, newConfig, manager.config)
	assert.False(t, manager.config.EnableNetworkIsolation)
	assert.Equal(t, 2.0, manager.config.DefaultCPULimit)
}

func TestUpdateConfig_Nil(t *testing.T) {
	manager := NewIsolationManager()
	
	err := manager.UpdateConfig(nil)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config cannot be nil")
}

func TestGetConfig(t *testing.T) {
	manager := NewIsolationManager()
	
	config := manager.GetConfig()
	
	assert.NotNil(t, config)
	assert.Equal(t, manager.config, config)
}

func TestApplyResourceLimits(t *testing.T) {
	manager := NewIsolationManager()
	limits := &ResourceLimits{
		CPUShares:  2048,
		CPUQuota:   200000,
		CPUPeriod:  100000,
		Memory:     1024 * 1024 * 1024,
		MemorySwap: 1024 * 1024 * 1024,
		PidsLimit:  200,
	}
	hostConfig := &container.HostConfig{}
	
	err := manager.applyResourceLimits(limits, hostConfig)
	
	require.NoError(t, err)
	assert.Equal(t, limits.CPUShares, hostConfig.Resources.CPUShares)
	assert.Equal(t, limits.CPUQuota, hostConfig.Resources.CPUQuota)
	assert.Equal(t, limits.CPUPeriod, hostConfig.Resources.CPUPeriod)
	assert.Equal(t, limits.Memory, hostConfig.Resources.Memory)
	assert.Equal(t, limits.MemorySwap, hostConfig.Resources.MemorySwap)
	assert.Equal(t, limits.PidsLimit, *hostConfig.Resources.PidsLimit)
	assert.Equal(t, uint16(500), hostConfig.Resources.BlkioWeight)
}

func TestApplyResourceLimits_NilLimits(t *testing.T) {
	manager := NewIsolationManager()
	hostConfig := &container.HostConfig{}
	
	err := manager.applyResourceLimits(nil, hostConfig)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "resource limits cannot be nil")
}

func TestApplySecurityOptions(t *testing.T) {
	manager := NewIsolationManager()
	opts := &SecurityOptions{
		NoNewPrivileges: true,
		ReadOnlyRootFS:  true,
		SeccompProfile:  "custom-seccomp",
		AppArmorProfile: "custom-apparmor",
		Capabilities: &CapabilityConfig{
			Drop: []string{"ALL", "NET_RAW"},
			Add:  []string{"CHOWN", "DAC_OVERRIDE"},
		},
	}
	config := &container.Config{}
	hostConfig := &container.HostConfig{}
	
	err := manager.applySecurityOptions(opts, config, hostConfig)
	
	require.NoError(t, err)
	assert.True(t, hostConfig.ReadonlyRootfs)
	assert.False(t, hostConfig.Privileged) // 기본적으로 비활성화
	assert.Contains(t, hostConfig.SecurityOpt, "no-new-privileges:true")
	assert.Contains(t, hostConfig.SecurityOpt, "seccomp=custom-seccomp")
	assert.Contains(t, hostConfig.SecurityOpt, "apparmor=custom-apparmor")
	assert.Contains(t, hostConfig.CapDrop, "ALL")
	assert.Contains(t, hostConfig.CapDrop, "NET_RAW")
	assert.Contains(t, hostConfig.CapAdd, "CHOWN")
	assert.Contains(t, hostConfig.CapAdd, "DAC_OVERRIDE")
}

func TestApplySecurityOptions_NilOptions(t *testing.T) {
	manager := NewIsolationManager()
	config := &container.Config{}
	hostConfig := &container.HostConfig{}
	
	err := manager.applySecurityOptions(nil, config, hostConfig)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "security options cannot be nil")
}

// 벤치마크 테스트
func BenchmarkCreateWorkspaceIsolation(b *testing.B) {
	manager := NewIsolationManager()
	workspace := &models.Workspace{
		ID:   "benchmark-workspace",
		Name: "Benchmark Workspace",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := manager.CreateWorkspaceIsolation(workspace)
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
	}
}

func BenchmarkApplyToContainer(b *testing.B) {
	manager := NewIsolationManager()
	workspace := &models.Workspace{ID: "benchmark", Name: "Benchmark"}
	isolation, err := manager.CreateWorkspaceIsolation(workspace)
	if err != nil {
		b.Fatalf("Failed to create isolation: %v", err)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		config := &container.Config{}
		hostConfig := &container.HostConfig{}
		
		err := manager.ApplyToContainer(isolation, config, hostConfig)
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
	}
}

func BenchmarkHashWorkspaceID(b *testing.B) {
	manager := NewIsolationManager()
	workspaceID := "test-workspace-for-benchmark-123456789"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.hashWorkspaceID(workspaceID)
	}
}