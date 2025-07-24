package status

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aicli/aicli-web/internal/docker"
)

func TestNewResourceMonitor(t *testing.T) {
	mockContainer := NewMockContainerManager()
	mockFactory := NewMockDockerManager()

	monitor := NewResourceMonitor(mockContainer, mockFactory)

	if monitor == nil {
		t.Fatal("NewResourceMonitor should not return nil")
	}

	if monitor.collectInterval != 10*time.Second {
		t.Errorf("Expected collect interval 10s, got %v", monitor.collectInterval)
	}

	if monitor.cacheExpiry != 30*time.Second {
		t.Errorf("Expected cache expiry 30s, got %v", monitor.cacheExpiry)
	}
}

func TestResourceMonitor_StartStop(t *testing.T) {
	mockContainer := NewMockContainerManager()
	mockFactory := NewMockDockerManager()

	monitor := NewResourceMonitor(mockContainer, mockFactory)

	// 시작 테스트
	err := monitor.Start()
	if err != nil {
		t.Fatalf("Failed to start monitor: %v", err)
	}

	// 잠시 실행
	time.Sleep(50 * time.Millisecond)

	// 중지 테스트
	err = monitor.Stop()
	if err != nil {
		t.Fatalf("Failed to stop monitor: %v", err)
	}
}

func TestResourceMonitor_StartMonitoring(t *testing.T) {
	mockContainer := NewMockContainerManager()
	mockFactory := NewMockDockerManager()

	// 테스트 컨테이너 설정
	container := &MockWorkspaceContainer{
		id:          "monitor-test-container",
		name:        "test-container",
		workspaceID: "ws-monitor-test",
		state:       "running",
		createdAt:   time.Now().Add(-1 * time.Hour),
	}
	mockContainer.AddContainer("ws-monitor-test", container)

	// 통계 데이터 설정
	stats := &docker.ContainerStats{
		CPUPercent:   15.0,
		MemoryUsage:  256 * 1024 * 1024, // 256MB
		MemoryLimit:  1024 * 1024 * 1024, // 1GB
		NetworkRxMB:  75.5,
		NetworkTxMB:  32.1,
		Timestamp:    time.Now(),
	}
	mockFactory.statsCollector.AddStats("monitor-test-container", stats)

	monitor := NewResourceMonitor(mockContainer, mockFactory)
	monitor.SetCollectInterval(100 * time.Millisecond) // 빠른 테스트

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// 모니터링 시작
	metricsChan, err := monitor.StartMonitoring(ctx, "ws-monitor-test")
	if err != nil {
		t.Fatalf("Failed to start monitoring: %v", err)
	}

	// 첫 번째 메트릭 수신 대기
	select {
	case metrics := <-metricsChan:
		if metrics == nil {
			t.Fatal("Received nil metrics")
		}

		if metrics.CPUPercent != 15.0 {
			t.Errorf("Expected CPU percent 15.0, got %f", metrics.CPUPercent)
		}

		if metrics.MemoryUsage != 256*1024*1024 {
			t.Errorf("Expected memory usage 256MB, got %d", metrics.MemoryUsage)
		}

		if metrics.NetworkRxMB != 75.5 {
			t.Errorf("Expected network RX 75.5MB, got %f", metrics.NetworkRxMB)
		}

		if metrics.Uptime == "" {
			t.Error("Uptime should not be empty")
		}

	case <-ctx.Done():
		t.Fatal("Timeout waiting for metrics")
	}

	// 모니터링 중지
	monitor.StopMonitoring("ws-monitor-test")

	// 활성 모니터 확인
	activeMonitors := monitor.GetActiveMonitors()
	for _, workspaceID := range activeMonitors {
		if workspaceID == "ws-monitor-test" {
			t.Error("Monitor should have been stopped")
		}
	}
}

func TestResourceMonitor_GetResourceSummary(t *testing.T) {
	mockContainer := NewMockContainerManager()
	mockFactory := NewMockDockerManager()

	// 테스트 시스템 통계 설정
	systemStats := &docker.SystemStats{
		ContainersRunning: 3,
		ContainersPaused:  0,
		ContainersStopped: 1,
		Images:           5,
	}

	// 여러 컨테이너 통계 설정
	allStats := map[string]*docker.ContainerStats{
		"container-1": {
			CPUPercent:  10.0,
			MemoryUsage: 100 * 1024 * 1024,
			NetworkRxMB: 10.0,
			NetworkTxMB: 5.0,
			Timestamp:   time.Now(),
		},
		"container-2": {
			CPUPercent:  20.0,
			MemoryUsage: 200 * 1024 * 1024,
			NetworkRxMB: 15.0,
			NetworkTxMB: 8.0,
			Timestamp:   time.Now(),
		},
		"container-3": {
			CPUPercent:  30.0,
			MemoryUsage: 300 * 1024 * 1024,
			NetworkRxMB: 20.0,
			NetworkTxMB: 10.0,
			Timestamp:   time.Now(),
		},
	}

	for containerID, stats := range allStats {
		mockFactory.statsCollector.AddStats(containerID, stats)
	}

	monitor := NewResourceMonitor(mockContainer, mockFactory)

	ctx := context.Background()
	summary, err := monitor.GetResourceSummary(ctx)
	if err != nil {
		t.Fatalf("Failed to get resource summary: %v", err)
	}

	if summary == nil {
		t.Fatal("Resource summary should not be nil")
	}

	if summary.TotalWorkspaces != 3 { // 컨테이너 수
		t.Errorf("Expected 3 total workspaces, got %d", summary.TotalWorkspaces)
	}

	if summary.ActiveContainers != 3 {
		t.Errorf("Expected 3 active containers, got %d", summary.ActiveContainers)
	}

	expectedTotalCPU := 10.0 + 20.0 + 30.0 // 60.0
	if summary.TotalCPUUsage != expectedTotalCPU {
		t.Errorf("Expected total CPU usage %f, got %f", expectedTotalCPU, summary.TotalCPUUsage)
	}

	expectedTotalMemory := int64(100+200+300) * 1024 * 1024 // 600MB
	if summary.TotalMemoryUsage != expectedTotalMemory {
		t.Errorf("Expected total memory usage %d, got %d", expectedTotalMemory, summary.TotalMemoryUsage)
	}

	expectedTotalNetwork := 10.0 + 5.0 + 15.0 + 8.0 + 20.0 + 10.0 // 68.0
	if summary.TotalNetworkIO != expectedTotalNetwork {
		t.Errorf("Expected total network IO %f, got %f", expectedTotalNetwork, summary.TotalNetworkIO)
	}
}

func TestResourceMonitor_CollectMetrics(t *testing.T) {
	mockContainer := NewMockContainerManager()
	mockFactory := NewMockDockerManager()

	// 테스트 컨테이너 설정
	container := &MockWorkspaceContainer{
		id:          "metrics-test-container",
		name:        "test-container",
		workspaceID: "ws-metrics-test",
		state:       "running",
		createdAt:   time.Now().Add(-2 * time.Hour), // 2시간 전 생성
	}
	mockContainer.AddContainer("ws-metrics-test", container)

	// 통계 데이터 설정
	stats := &docker.ContainerStats{
		CPUPercent:   85.5, // 높은 CPU 사용률 (경고 테스트)
		MemoryUsage:  900 * 1024 * 1024,  // 900MB
		MemoryLimit:  1024 * 1024 * 1024, // 1GB (87.5% 사용률, 경고 테스트)
		NetworkRxMB:  200.0,
		NetworkTxMB:  150.0,
		Timestamp:    time.Now(),
	}
	mockFactory.statsCollector.AddStats("metrics-test-container", stats)

	monitor := NewResourceMonitor(mockContainer, mockFactory)

	ctx := context.Background()
	metrics := monitor.collectMetrics(ctx, "ws-metrics-test")

	if metrics == nil {
		t.Fatal("Metrics should not be nil")
	}

	if metrics.CPUPercent != 85.5 {
		t.Errorf("Expected CPU percent 85.5, got %f", metrics.CPUPercent)
	}

	if metrics.MemoryUsage != 900*1024*1024 {
		t.Errorf("Expected memory usage 900MB, got %d", metrics.MemoryUsage)
	}

	if metrics.MemoryLimit != 1024*1024*1024 {
		t.Errorf("Expected memory limit 1GB, got %d", metrics.MemoryLimit)
	}

	if metrics.NetworkRxMB != 200.0 {
		t.Errorf("Expected network RX 200.0MB, got %f", metrics.NetworkRxMB)
	}

	if metrics.NetworkTxMB != 150.0 {
		t.Errorf("Expected network TX 150.0MB, got %f", metrics.NetworkTxMB)
	}

	if metrics.Uptime == "" {
		t.Error("Uptime should not be empty")
	}

	// Uptime이 대략 2시간 정도인지 확인
	if !contains(metrics.Uptime, "h") {
		t.Errorf("Uptime should contain hours, got %s", metrics.Uptime)
	}
}

func TestResourceMonitor_GetActiveMonitors(t *testing.T) {
	mockContainer := NewMockContainerManager()
	mockFactory := NewMockDockerManager()

	monitor := NewResourceMonitor(mockContainer, mockFactory)

	// 초기에는 활성 모니터가 없어야 함
	activeMonitors := monitor.GetActiveMonitors()
	if len(activeMonitors) != 0 {
		t.Errorf("Expected 0 active monitors, got %d", len(activeMonitors))
	}

	// 모니터링 시작
	ctx1, cancel1 := context.WithCancel(context.Background())
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel1()
	defer cancel2()

	_, err1 := monitor.StartMonitoring(ctx1, "ws-1")
	if err1 != nil {
		t.Fatalf("Failed to start monitoring ws-1: %v", err1)
	}

	_, err2 := monitor.StartMonitoring(ctx2, "ws-2")
	if err2 != nil {
		t.Fatalf("Failed to start monitoring ws-2: %v", err2)
	}

	// 활성 모니터 확인
	activeMonitors = monitor.GetActiveMonitors()
	if len(activeMonitors) != 2 {
		t.Errorf("Expected 2 active monitors, got %d", len(activeMonitors))
	}

	// 예상되는 워크스페이스 ID들이 있는지 확인
	expectedIDs := map[string]bool{"ws-1": true, "ws-2": true}
	for _, workspaceID := range activeMonitors {
		if !expectedIDs[workspaceID] {
			t.Errorf("Unexpected workspace ID in active monitors: %s", workspaceID)
		}
		delete(expectedIDs, workspaceID)
	}

	if len(expectedIDs) > 0 {
		t.Errorf("Some expected workspace IDs not found in active monitors: %v", expectedIDs)
	}

	// 하나 중지
	monitor.StopMonitoring("ws-1")

	activeMonitors = monitor.GetActiveMonitors()
	if len(activeMonitors) != 1 {
		t.Errorf("Expected 1 active monitor after stopping one, got %d", len(activeMonitors))
	}

	if activeMonitors[0] != "ws-2" {
		t.Errorf("Expected remaining monitor to be ws-2, got %s", activeMonitors[0])
	}
}

func TestResourceMonitor_CacheStats(t *testing.T) {
	mockContainer := NewMockContainerManager()
	mockFactory := NewMockDockerManager()

	monitor := NewResourceMonitor(mockContainer, mockFactory)

	cacheStats := monitor.GetCacheStats()

	if cacheStats.CachedContainers != 0 {
		t.Errorf("Expected 0 cached containers initially, got %d", cacheStats.CachedContainers)
	}

	if cacheStats.CacheExpiry != 30*time.Second {
		t.Errorf("Expected cache expiry 30s, got %v", cacheStats.CacheExpiry)
	}
}

func TestResourceMonitor_GetMonitorStats(t *testing.T) {
	mockContainer := NewMockContainerManager()
	mockFactory := NewMockDockerManager()

	monitor := NewResourceMonitor(mockContainer, mockFactory)

	stats := monitor.GetMonitorStats()

	if stats.ActiveMonitors != 0 {
		t.Errorf("Expected 0 active monitors initially, got %d", stats.ActiveMonitors)
	}

	if stats.CollectInterval != 10*time.Second {
		t.Errorf("Expected collect interval 10s, got %v", stats.CollectInterval)
	}

	// 모니터링 시작 후 통계 확인
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	monitor.StartMonitoring(ctx, "ws-stats-test")

	stats = monitor.GetMonitorStats()
	if stats.ActiveMonitors != 1 {
		t.Errorf("Expected 1 active monitor after starting, got %d", stats.ActiveMonitors)
	}
}

// 성능 테스트
func TestResourceMonitor_Performance(t *testing.T) {
	mockContainer := NewMockContainerManager()
	mockFactory := NewMockDockerManager()

	// 여러 워크스페이스 설정
	for i := 0; i < 100; i++ {
		workspaceID := fmt.Sprintf("ws-perf-%d", i)
		containerID := fmt.Sprintf("container-perf-%d", i)

		container := &MockWorkspaceContainer{
			id:          containerID,
			workspaceID: workspaceID,
			state:       "running",
			createdAt:   time.Now(),
		}
		mockContainer.AddContainer(workspaceID, container)

		stats := &docker.ContainerStats{
			CPUPercent:  float64(i % 100),
			MemoryUsage: int64(i * 1024 * 1024),
			NetworkRxMB: float64(i),
			NetworkTxMB: float64(i / 2),
			Timestamp:   time.Now(),
		}
		mockFactory.statsCollector.AddStats(containerID, stats)
	}

	monitor := NewResourceMonitor(mockContainer, mockFactory)

	ctx := context.Background()
	start := time.Now()

	// 리소스 요약 조회 성능 테스트
	summary, err := monitor.GetResourceSummary(ctx)
	if err != nil {
		t.Fatalf("Failed to get resource summary: %v", err)
	}

	elapsed := time.Since(start)

	if elapsed > 1*time.Second {
		t.Errorf("Resource summary took too long: %v", elapsed)
	}

	if summary.TotalWorkspaces != 100 {
		t.Errorf("Expected 100 total workspaces, got %d", summary.TotalWorkspaces)
	}

	t.Logf("Resource summary for 100 workspaces completed in %v", elapsed)
}

// 헬퍼 함수들 - contains와 containsInner는 events.go에 정의되어 있음