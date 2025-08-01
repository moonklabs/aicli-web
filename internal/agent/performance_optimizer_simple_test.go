package agent

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestDefaultPerformanceConfig는 기본 성능 설정이 올바르게 생성되는지 테스트합니다
func TestDefaultPerformanceConfig(t *testing.T) {
	config := DefaultPerformanceConfig()

	assert.NotNil(t, config)
	assert.True(t, config.ContainerPoolSize > 0)
	assert.True(t, config.ContainerPoolMaxSize > 0)
	assert.True(t, config.WorktreePoolSize > 0)
	assert.True(t, config.AutoScaling.MinAgents > 0)
	assert.True(t, config.AutoScaling.MaxAgents > 0)
	assert.True(t, config.TargetCreationTime > 0)
	assert.True(t, config.OptimizationInterval > 0)
	assert.True(t, config.MetricsInterval > 0)
}

// TestSystemStatus는 시스템 상태 구조체가 올바르게 생성되는지 테스트합니다
func TestSystemStatus(t *testing.T) {
	status := &SystemStatus{
		CPUUsage:       50.0,
		MemoryUsage:    1024 * 1024 * 1024, // 1GB
		DiskUsage:      0.75,
		ActiveAgents:   5,
		QueuedRequests: 10,
		LastUpdated:    time.Now(),
	}

	assert.NotNil(t, status)
	assert.Equal(t, 50.0, status.CPUUsage)
	assert.Equal(t, float64(1024*1024*1024), status.MemoryUsage)
	assert.Equal(t, 0.75, status.DiskUsage)
	assert.Equal(t, 5, status.ActiveAgents)
	assert.Equal(t, 10, status.QueuedRequests)
}

// TestPerformanceMetrics는 성능 메트릭 구조체가 올바르게 동작하는지 테스트합니다
func TestPerformanceMetrics(t *testing.T) {
	metrics := &PerformanceMetrics{
		AgentCreationTimes:  []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
		AverageCreationTime: 150 * time.Millisecond,
		P95CreationTime:     180 * time.Millisecond,
		P99CreationTime:     190 * time.Millisecond,
	}

	assert.NotNil(t, metrics)
	assert.Len(t, metrics.AgentCreationTimes, 2)
	assert.Equal(t, 150*time.Millisecond, metrics.AverageCreationTime)
	assert.Equal(t, 180*time.Millisecond, metrics.P95CreationTime)
	assert.Equal(t, 190*time.Millisecond, metrics.P99CreationTime)
}

// TestPoolContainerStatus는 컨테이너 상태 enum이 올바르게 정의되었는지 테스트합니다
func TestPoolContainerStatus(t *testing.T) {
	assert.Equal(t, PoolContainerStatus(0), PoolContainerStatusReady)
	assert.Equal(t, PoolContainerStatus(1), PoolContainerStatusInUse)
	assert.Equal(t, PoolContainerStatus(2), PoolContainerStatusWarming)
	assert.Equal(t, PoolContainerStatus(3), PoolContainerStatusRecycling)
	assert.Equal(t, PoolContainerStatus(4), PoolContainerStatusError)
}

// TestAutoScalingConfig는 자동 스케일링 설정이 올바르게 동작하는지 테스트합니다
func TestAutoScalingConfig(t *testing.T) {
	config := AutoScalingConfig{
		Enabled:            true,
		MinAgents:          2,
		MaxAgents:          20,
		ScaleUpThreshold:   80.0,
		ScaleDownThreshold: 30.0,
		ScaleUpCooldown:    5 * time.Minute,
		ScaleDownCooldown:  10 * time.Minute,
		PredictiveScaling:  false,
		TargetUtilization:  70.0,
	}

	assert.True(t, config.Enabled)
	assert.Equal(t, 2, config.MinAgents)
	assert.Equal(t, 20, config.MaxAgents)
	assert.Equal(t, 80.0, config.ScaleUpThreshold)
	assert.Equal(t, 30.0, config.ScaleDownThreshold)
	assert.Equal(t, 5*time.Minute, config.ScaleUpCooldown)
	assert.Equal(t, 10*time.Minute, config.ScaleDownCooldown)
	assert.False(t, config.PredictiveScaling)
	assert.Equal(t, 70.0, config.TargetUtilization)
}

// TestContainerPoolStats는 컨테이너 풀 통계가 올바르게 동작하는지 테스트합니다
func TestContainerPoolStats(t *testing.T) {
	stats := ContainerPoolStats{
		TotalContainers:     10,
		AvailableContainers: 7,
		InUseContainers:     3,
		MaxCapacity:         15,
		Utilization:         0.2, // 3/15
		LastOptimized:       time.Now(),
	}

	assert.Equal(t, 10, stats.TotalContainers)
	assert.Equal(t, 7, stats.AvailableContainers)
	assert.Equal(t, 3, stats.InUseContainers)
	assert.Equal(t, 15, stats.MaxCapacity)
	assert.Equal(t, 0.2, stats.Utilization)
}

// TestResourceUsage는 리소스 사용량 구조체가 올바르게 동작하는지 테스트합니다
func TestResourceUsage(t *testing.T) {
	usage := ResourceUsage{
		CPUUsage:    0.5,
		MemoryUsage: 512 * 1024 * 1024, // 512MB
		DiskUsage:   100 * 1024 * 1024, // 100MB
		NetworkRx:   1024,
		NetworkTx:   2048,
		LastUpdated: time.Now(),
	}

	assert.Equal(t, 0.5, usage.CPUUsage)
	assert.Equal(t, int64(512*1024*1024), usage.MemoryUsage)
	assert.Equal(t, int64(100*1024*1024), usage.DiskUsage)
	assert.Equal(t, int64(1024), usage.NetworkRx)
	assert.Equal(t, int64(2048), usage.NetworkTx)
}
