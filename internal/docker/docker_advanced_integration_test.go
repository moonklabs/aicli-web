package docker

import (
	"context"
	"testing"
	"time"

	"github.com/aicli/aicli-web/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDockerAdvancedIntegration(t *testing.T) {
	// Skip if Docker is not available
	client, err := NewClient(DefaultConfig())
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}

	// Create test components
	lifecycle := NewLifecycleManager(client)
	defer lifecycle.Close()

	health := NewHealthChecker(client, 5*time.Second)
	agentSync := NewAgentDockerSync(client, lifecycle)
	defer agentSync.Close()

	resourceMonitor := NewAdvancedResourceMonitor(client)
	defer resourceMonitor.Close()

	autoRecovery := NewAutoRecoveryManager(client, lifecycle, health, agentSync)
	defer autoRecovery.Close()

	t.Run("AgentDockerSync", func(t *testing.T) {
		agentID := "test-agent-1"
		containerID := "test-container-1"
		initialStatus := models.AgentStatusIdle

		// Register agent
		err := agentSync.RegisterAgent(agentID, containerID, initialStatus)
		require.NoError(t, err)

		// Get sync state
		state, exists := agentSync.GetSyncState(agentID)
		assert.True(t, exists)
		assert.Equal(t, agentID, state.AgentID)
		assert.Equal(t, containerID, state.ContainerID)
		assert.Equal(t, initialStatus, state.AgentStatus)

		// Update agent status
		err = agentSync.UpdateAgentStatus(agentID, models.AgentStatusRunning)
		assert.NoError(t, err)

		// Verify metrics
		metrics := agentSync.GetSyncMetrics()
		assert.NotNil(t, metrics)
		assert.Equal(t, 1, metrics["total_agents"])

		// Unregister agent
		agentSync.UnregisterAgent(agentID)
		_, exists = agentSync.GetSyncState(agentID)
		assert.False(t, exists)
	})

	t.Run("AutoRecoveryManager", func(t *testing.T) {
		agentID := "test-agent-2"
		containerID := "test-container-2"
		policy := DefaultRecoveryPolicy()

		// Register agent
		autoRecovery.RegisterAgent(agentID, containerID, policy)

		// Get recovery state
		state, exists := autoRecovery.GetRecoveryState(agentID)
		assert.True(t, exists)
		assert.Equal(t, agentID, state.AgentID)
		assert.Equal(t, containerID, state.ContainerID)
		assert.Equal(t, policy, state.Policy)

		// Verify metrics
		metrics := autoRecovery.GetRecoveryMetrics()
		assert.NotNil(t, metrics)
		assert.Equal(t, 1, metrics["total_agents"])

		// Unregister agent
		autoRecovery.UnregisterAgent(agentID)
		_, exists = autoRecovery.GetRecoveryState(agentID)
		assert.False(t, exists)
	})

	t.Run("AdvancedResourceMonitor", func(t *testing.T) {
		agentID := "test-agent-3"
		containerID := "test-container-3"
		config := DefaultMonitoringConfig()

		// Register container
		resourceMonitor.RegisterContainer(agentID, containerID, config)

		// Get container state
		state, exists := resourceMonitor.GetContainerState(agentID)
		assert.True(t, exists)
		assert.Equal(t, agentID, state.AgentID)
		assert.Equal(t, containerID, state.ContainerID)
		assert.Equal(t, config, state.Config)

		// Verify metrics
		metrics := resourceMonitor.GetMonitoringMetrics()
		assert.NotNil(t, metrics)
		assert.Equal(t, 1, metrics["total_containers"])

		// Unregister container
		resourceMonitor.UnregisterContainer(agentID)
		_, exists = resourceMonitor.GetContainerState(agentID)
		assert.False(t, exists)
	})
}

func TestResourceThreshold(t *testing.T) {
	threshold := DefaultResourceThreshold()
	
	assert.Equal(t, 70.0, threshold.CPUWarning)
	assert.Equal(t, 90.0, threshold.CPUCritical)
	assert.Equal(t, 80.0, threshold.MemoryWarning)
	assert.Equal(t, 95.0, threshold.MemoryCritical)
}

func TestRecoveryPolicy(t *testing.T) {
	policy := DefaultRecoveryPolicy()
	
	assert.Equal(t, 3, policy.MaxRetries)
	assert.Equal(t, 30*time.Second, policy.RetryInterval)
	assert.Equal(t, 10*time.Second, policy.HealthCheckDelay)
	assert.True(t, policy.EnableAutoRestart)
	assert.False(t, policy.EnableRecreate)
}

func TestMonitoringConfig(t *testing.T) {
	config := DefaultMonitoringConfig()
	
	assert.Equal(t, 10*time.Second, config.Interval)
	assert.Equal(t, 100, config.HistorySize)
	assert.True(t, config.EnableAlerting)
	assert.False(t, config.EnableAutoScale)
}

func TestUtilizationCalculation(t *testing.T) {
	metrics := &ResourceMetrics{
		CPUUsage: CPUMetrics{
			UsagePercent: 75.5,
		},
		MemoryUsage: MemoryMetrics{
			UsagePercent: 85.2,
		},
		NetworkIO: NetworkMetrics{
			RxRate: 50 * 1024 * 1024, // 50MB/s
			TxRate: 30 * 1024 * 1024, // 30MB/s
		},
		DiskIO: DiskMetrics{
			ReadRate:  10 * 1024 * 1024, // 10MB/s
			WriteRate: 5 * 1024 * 1024,  // 5MB/s
		},
	}

	arm := &AdvancedResourceMonitor{}
	arm.calculateUtilization(metrics)

	assert.Equal(t, 75.5, metrics.Utilization.CPUUtilization)
	assert.Equal(t, 85.2, metrics.Utilization.MemoryUtilization)
	assert.True(t, metrics.Utilization.NetworkUtilization > 0)
	assert.True(t, metrics.Utilization.DiskUtilization > 0)
	assert.True(t, metrics.Utilization.OverallScore > 0)
}

func TestAlertGeneration(t *testing.T) {
	alerts := make([]ResourceAlert, 0)
	
	handler := func(alert ResourceAlert) {
		alerts = append(alerts, alert)
	}

	// Test critical CPU alert
	alert := ResourceAlert{
		AgentID:     "test-agent",
		ContainerID: "test-container",
		Level:       AlertLevelCritical,
		Resource:    "cpu",
		Value:       95.0,
		Threshold:   90.0,
		Message:     "CPU usage critical: 95.00%",
	}

	handler(alert)
	
	assert.Len(t, alerts, 1)
	assert.Equal(t, AlertLevelCritical, alerts[0].Level)
	assert.Equal(t, "cpu", alerts[0].Resource)
	assert.Equal(t, 95.0, alerts[0].Value)
}