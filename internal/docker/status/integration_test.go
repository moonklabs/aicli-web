package status

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/aicli/aicli-web/internal/docker"
	"github.com/aicli/aicli-web/internal/models"
)

// TestIntegration_TrackerAndMonitor 상태 추적자와 리소스 모니터의 통합 테스트
func TestIntegration_TrackerAndMonitor(t *testing.T) {
	mockService := NewMockWorkspaceService()
	mockContainer := NewMockContainerManager()
	mockFactory := NewMockDockerManager()

	// 테스트 데이터 설정
	workspace := &models.Workspace{
		ID:     "ws-integration-test",
		Name:   "Integration Test Workspace",
		Status: models.WorkspaceStatusActive,
	}
	mockService.AddWorkspace(workspace)

	container := &MockWorkspaceContainer{
		id:          "container-integration-test",
		name:        "integration-test-container",
		workspaceID: "ws-integration-test",
		state:       "running",
		createdAt:   time.Now(),
	}
	mockContainer.AddContainer("ws-integration-test", container)

	stats := &docker.ContainerStats{
		CPUPercent:   45.0,
		MemoryUsage:  512 * 1024 * 1024, // 512MB
		MemoryLimit:  2048 * 1024 * 1024, // 2GB
		NetworkRxMB:  120.0,
		NetworkTxMB:  80.0,
		Timestamp:    time.Now(),
	}
	mockFactory.statsCollector.AddStats("container-integration-test", stats)

	// Tracker와 Monitor 생성
	tracker := NewTracker(mockService, mockContainer, mockFactory)
	monitor := NewResourceMonitor(mockContainer, mockFactory)

	// 빠른 테스트를 위한 간격 설정
	tracker.SetSyncInterval(200 * time.Millisecond)
	monitor.SetCollectInterval(150 * time.Millisecond)

	// 상태 변경 이벤트 추적
	var eventReceived bool
	var receivedState *WorkspaceState
	var eventMutex sync.Mutex

	tracker.OnStateChange(func(workspaceID string, oldState, newState *WorkspaceState) {
		eventMutex.Lock()
		defer eventMutex.Unlock()
		if workspaceID == "ws-integration-test" {
			eventReceived = true
			receivedState = newState
		}
	})

	// 시스템 시작
	err := tracker.Start()
	if err != nil {
		t.Fatalf("Failed to start tracker: %v", err)
	}
	defer tracker.Stop()

	err = monitor.Start()
	if err != nil {
		t.Fatalf("Failed to start monitor: %v", err)
	}
	defer monitor.Stop()

	// 모니터링 시작
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	metricsChan, err := monitor.StartMonitoring(ctx, "ws-integration-test")
	if err != nil {
		t.Fatalf("Failed to start monitoring: %v", err)
	}

	// 이벤트와 메트릭 수신 대기
	var metricsReceived bool
	var receivedMetrics *WorkspaceMetrics

	select {
	case metrics := <-metricsChan:
		metricsReceived = true
		receivedMetrics = metrics
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for metrics")
	}

	// 이벤트 수신 대기 (비동기이므로 잠시 대기)
	time.Sleep(300 * time.Millisecond)

	// 검증
	eventMutex.Lock()
	defer eventMutex.Unlock()

	if !eventReceived {
		t.Fatal("State change event not received")
	}

	if !metricsReceived {
		t.Fatal("Metrics not received")
	}

	// 상태 검증
	if receivedState == nil {
		t.Fatal("Received state is nil")
	}

	if receivedState.WorkspaceID != "ws-integration-test" {
		t.Errorf("Expected workspace ID ws-integration-test, got %s", receivedState.WorkspaceID)
	}

	if receivedState.Status != models.WorkspaceStatusActive {
		t.Errorf("Expected status active, got %s", receivedState.Status)
	}

	if receivedState.ContainerID != "container-integration-test" {
		t.Errorf("Expected container ID container-integration-test, got %s", receivedState.ContainerID)
	}

	// 메트릭 검증
	if receivedMetrics == nil {
		t.Fatal("Received metrics is nil")
	}

	if receivedMetrics.CPUPercent != 45.0 {
		t.Errorf("Expected CPU percent 45.0, got %f", receivedMetrics.CPUPercent)
	}

	if receivedMetrics.MemoryUsage != 512*1024*1024 {
		t.Errorf("Expected memory usage 512MB, got %d", receivedMetrics.MemoryUsage)
	}

	// 트래커에서 상태 조회
	trackerState, exists := tracker.GetWorkspaceState("ws-integration-test")
	if !exists {
		t.Fatal("Workspace state not found in tracker")
	}

	if trackerState.Metrics != nil {
		if trackerState.Metrics.CPUPercent != 45.0 {
			t.Errorf("Expected tracker metrics CPU 45.0, got %f", trackerState.Metrics.CPUPercent)
		}
	}
}

// TestIntegration_MultipleWorkspaces 여러 워크스페이스 동시 관리 테스트
func TestIntegration_MultipleWorkspaces(t *testing.T) {
	mockService := NewMockWorkspaceService()
	mockContainer := NewMockContainerManager()
	mockFactory := NewMockDockerManager()

	tracker := NewTracker(mockService, mockContainer, mockFactory)
	monitor := NewResourceMonitor(mockContainer, mockFactory)

	tracker.SetSyncInterval(100 * time.Millisecond)
	monitor.SetCollectInterval(100 * time.Millisecond)

	// 여러 워크스페이스 설정
	numWorkspaces := 5
	workspaces := make([]*models.Workspace, numWorkspaces)
	containers := make([]*MockWorkspaceContainer, numWorkspaces)

	for i := 0; i < numWorkspaces; i++ {
		workspaceID := fmt.Sprintf("ws-multi-%d", i)
		containerID := fmt.Sprintf("container-multi-%d", i)

		workspaces[i] = &models.Workspace{
			ID:     workspaceID,
			Name:   fmt.Sprintf("Multi Workspace %d", i),
			Status: models.WorkspaceStatusActive,
		}
		mockService.AddWorkspace(workspaces[i])

		containers[i] = &MockWorkspaceContainer{
			id:          containerID,
			name:        fmt.Sprintf("multi-container-%d", i),
			workspaceID: workspaceID,
			state:       "running",
			createdAt:   time.Now(),
		}
		mockContainer.AddContainer(workspaceID, containers[i])

		stats := &docker.ContainerStats{
			CPUPercent:   float64(10 + i*10), // 10, 20, 30, 40, 50
			MemoryUsage:  int64((100 + i*100) * 1024 * 1024), // 100MB, 200MB, ...
			MemoryLimit:  1024 * 1024 * 1024, // 1GB
			NetworkRxMB:  float64(50 + i*10),
			NetworkTxMB:  float64(25 + i*5),
			Timestamp:    time.Now(),
		}
		mockFactory.statsCollector.AddStats(containerID, stats)
	}

	// 시스템 시작
	err := tracker.Start()
	if err != nil {
		t.Fatalf("Failed to start tracker: %v", err)
	}
	defer tracker.Stop()

	err = monitor.Start()
	if err != nil {
		t.Fatalf("Failed to start monitor: %v", err)
	}
	defer monitor.Stop()

	// 모든 워크스페이스 모니터링 시작
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	channels := make([]<-chan *WorkspaceMetrics, numWorkspaces)
	for i, workspace := range workspaces {
		metricsChan, err := monitor.StartMonitoring(ctx, workspace.ID)
		if err != nil {
			t.Fatalf("Failed to start monitoring workspace %s: %v", workspace.ID, err)
		}
		channels[i] = metricsChan
	}

	// 동기화 대기
	time.Sleep(300 * time.Millisecond)

	// 모든 워크스페이스 상태 확인
	allStates := tracker.GetAllWorkspaceStates()
	if len(allStates) != numWorkspaces {
		t.Errorf("Expected %d workspace states, got %d", numWorkspaces, len(allStates))
	}

	for i, workspace := range workspaces {
		state, exists := allStates[workspace.ID]
		if !exists {
			t.Errorf("Workspace %s state not found", workspace.ID)
			continue
		}

		if state.Status != models.WorkspaceStatusActive {
			t.Errorf("Workspace %s expected status active, got %s", workspace.ID, state.Status)
		}

		if state.ContainerID != containers[i].id {
			t.Errorf("Workspace %s expected container %s, got %s", 
				workspace.ID, containers[i].id, state.ContainerID)
		}
	}

	// 리소스 요약 확인
	summary, err := monitor.GetResourceSummary(ctx)
	if err != nil {
		t.Fatalf("Failed to get resource summary: %v", err)
	}

	if summary.ActiveContainers != numWorkspaces {
		t.Errorf("Expected %d active containers, got %d", numWorkspaces, summary.ActiveContainers)
	}

	expectedTotalCPU := 10.0 + 20.0 + 30.0 + 40.0 + 50.0 // 150.0
	if summary.TotalCPUUsage != expectedTotalCPU {
		t.Errorf("Expected total CPU usage %f, got %f", expectedTotalCPU, summary.TotalCPUUsage)
	}

	// 활성 모니터 확인
	activeMonitors := monitor.GetActiveMonitors()
	if len(activeMonitors) != numWorkspaces {
		t.Errorf("Expected %d active monitors, got %d", numWorkspaces, len(activeMonitors))
	}
}

// TestIntegration_StateTransitions 상태 전환 테스트
func TestIntegration_StateTransitions(t *testing.T) {
	mockService := NewMockWorkspaceService()
	mockContainer := NewMockContainerManager()
	mockFactory := NewMockDockerManager()

	tracker := NewTracker(mockService, mockContainer, mockFactory)
	tracker.SetSyncInterval(50 * time.Millisecond)

	// 초기 상태: inactive (컨테이너 없음)
	workspace := &models.Workspace{
		ID:     "ws-transition-test",
		Name:   "Transition Test",
		Status: models.WorkspaceStatusInactive,
	}
	mockService.AddWorkspace(workspace)

	var stateChanges []string
	var stateChangesMutex sync.Mutex

	tracker.OnStateChange(func(workspaceID string, oldState, newState *WorkspaceState) {
		stateChangesMutex.Lock()
		defer stateChangesMutex.Unlock()
		if workspaceID == "ws-transition-test" {
			transition := fmt.Sprintf("%s->%s", 
				func() string {
					if oldState != nil {
						return string(oldState.Status)
					}
					return "nil"
				}(), 
				newState.Status)
			stateChanges = append(stateChanges, transition)
		}
	})

	err := tracker.Start()
	if err != nil {
		t.Fatalf("Failed to start tracker: %v", err)
	}
	defer tracker.Stop()

	// 초기 동기화 대기
	time.Sleep(100 * time.Millisecond)

	// 상태 1: 컨테이너 추가 (running)
	container := &MockWorkspaceContainer{
		id:          "container-transition-test",
		workspaceID: "ws-transition-test",
		state:       "running",
		createdAt:   time.Now(),
	}
	mockContainer.AddContainer("ws-transition-test", container)

	// 수동 동기화 트리거
	tracker.ForceSync("ws-transition-test")
	time.Sleep(100 * time.Millisecond)

	// 상태 2: 컨테이너 중지 (exited)
	container.state = "exited"
	tracker.ForceSync("ws-transition-test")
	time.Sleep(100 * time.Millisecond)

	// 상태 3: 컨테이너 제거
	mockContainer.containers["ws-transition-test"] = []*MockWorkspaceContainer{}
	tracker.ForceSync("ws-transition-test")
	time.Sleep(100 * time.Millisecond)

	// 상태 변경 확인
	stateChangesMutex.Lock()
	defer stateChangesMutex.Unlock()

	if len(stateChanges) == 0 {
		t.Fatal("No state changes recorded")
	}

	t.Logf("State changes: %v", stateChanges)

	// 최종 상태 확인
	finalState, exists := tracker.GetWorkspaceState("ws-transition-test")
	if !exists {
		t.Fatal("Final workspace state not found")
	}

	if finalState.Status != models.WorkspaceStatusInactive {
		t.Errorf("Expected final status inactive, got %s", finalState.Status)
	}
}

// TestIntegration_ErrorRecovery 오류 복구 테스트
func TestIntegration_ErrorRecovery(t *testing.T) {
	mockService := NewMockWorkspaceService()
	mockContainer := NewMockContainerManager()
	mockFactory := NewMockDockerManager()

	tracker := NewTracker(mockService, mockContainer, mockFactory)
	tracker.SetSyncInterval(100 * time.Millisecond)
	tracker.SetMaxRetries(2)

	// 에러가 발생할 워크스페이스 (서비스에 없음)
	container := &MockWorkspaceContainer{
		id:          "error-container",
		workspaceID: "ws-error-test",
		state:       "running",
		createdAt:   time.Now(),
	}
	mockContainer.AddContainer("ws-error-test", container)

	var errorCount int
	var errorCountMutex sync.Mutex

	// 사용자 정의 로거로 에러 추적
	tracker.SetLogger(&testLogger{
		onError: func(msg string, err error, args ...interface{}) {
			errorCountMutex.Lock()
			defer errorCountMutex.Unlock()
			errorCount++
		},
	})

	err := tracker.Start()
	if err != nil {
		t.Fatalf("Failed to start tracker: %v", err)
	}
	defer tracker.Stop()

	// 에러 발생 대기
	time.Sleep(300 * time.Millisecond)

	// 오류가 기록되었는지 확인
	errorCountMutex.Lock()
	currentErrorCount := errorCount
	errorCountMutex.Unlock()

	if currentErrorCount == 0 {
		t.Error("Expected errors to be logged for missing workspace")
	}

	// 워크스페이스를 서비스에 추가하여 복구
	workspace := &models.Workspace{
		ID:     "ws-error-test",
		Name:   "Error Recovery Test",
		Status: models.WorkspaceStatusActive,
	}
	mockService.AddWorkspace(workspace)

	// 복구 대기
	time.Sleep(300 * time.Millisecond)

	// 상태가 정상적으로 동기화되었는지 확인
	state, exists := tracker.GetWorkspaceState("ws-error-test")
	if !exists {
		t.Fatal("Workspace state not found after recovery")
	}

	if state.Status != models.WorkspaceStatusActive {
		t.Errorf("Expected status active after recovery, got %s", state.Status)
	}

	if state.ContainerID != "error-container" {
		t.Errorf("Expected container ID error-container, got %s", state.ContainerID)
	}
}

// testLogger 테스트용 로거
type testLogger struct {
	onInfo  func(string, ...interface{})
	onError func(string, error, ...interface{})
	onDebug func(string, ...interface{})
	onWarn  func(string, ...interface{})
}

func (l *testLogger) Info(msg string, args ...interface{}) {
	if l.onInfo != nil {
		l.onInfo(msg, args...)
	}
}

func (l *testLogger) Error(msg string, err error, args ...interface{}) {
	if l.onError != nil {
		l.onError(msg, err, args...)
	}
}

func (l *testLogger) Debug(msg string, args ...interface{}) {
	if l.onDebug != nil {
		l.onDebug(msg, args...)
	}
}

func (l *testLogger) Warn(msg string, args ...interface{}) {
	if l.onWarn != nil {
		l.onWarn(msg, args...)
	}
}

// TestIntegration_Performance 성능 통합 테스트
func TestIntegration_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	mockService := NewMockWorkspaceService()
	mockContainer := NewMockContainerManager()
	mockFactory := NewMockDockerManager()

	// 대량의 워크스페이스 생성
	numWorkspaces := 50
	for i := 0; i < numWorkspaces; i++ {
		workspaceID := fmt.Sprintf("ws-perf-%d", i)
		containerID := fmt.Sprintf("container-perf-%d", i)

		workspace := &models.Workspace{
			ID:     workspaceID,
			Name:   fmt.Sprintf("Performance Test %d", i),
			Status: models.WorkspaceStatusActive,
		}
		mockService.AddWorkspace(workspace)

		container := &MockWorkspaceContainer{
			id:          containerID,
			workspaceID: workspaceID,
			state:       "running",
			createdAt:   time.Now(),
		}
		mockContainer.AddContainer(workspaceID, container)

		stats := &docker.ContainerStats{
			CPUPercent:   float64(i % 100),
			MemoryUsage:  int64(i * 10 * 1024 * 1024),
			NetworkRxMB:  float64(i),
			NetworkTxMB:  float64(i / 2),
			Timestamp:    time.Now(),
		}
		mockFactory.statsCollector.AddStats(containerID, stats)
	}

	tracker := NewTracker(mockService, mockContainer, mockFactory)
	monitor := NewResourceMonitor(mockContainer, mockFactory)

	tracker.SetSyncInterval(200 * time.Millisecond)
	monitor.SetCollectInterval(100 * time.Millisecond)

	start := time.Now()

	// 시스템 시작
	err := tracker.Start()
	if err != nil {
		t.Fatalf("Failed to start tracker: %v", err)
	}
	defer tracker.Stop()

	err = monitor.Start()
	if err != nil {
		t.Fatalf("Failed to start monitor: %v", err)
	}
	defer monitor.Stop()

	// 초기 동기화 대기
	time.Sleep(500 * time.Millisecond)

	syncTime := time.Since(start)

	// 상태 조회 성능 테스트
	start = time.Now()
	allStates := tracker.GetAllWorkspaceStates()
	stateQueryTime := time.Since(start)

	// 리소스 요약 성능 테스트
	ctx := context.Background()
	start = time.Now()
	summary, err := monitor.GetResourceSummary(ctx)
	if err != nil {
		t.Fatalf("Failed to get resource summary: %v", err)
	}
	summaryTime := time.Since(start)

	// 성능 검증
	if syncTime > 2*time.Second {
		t.Errorf("Initial sync took too long: %v", syncTime)
	}

	if stateQueryTime > 100*time.Millisecond {
		t.Errorf("State query took too long: %v", stateQueryTime)
	}

	if summaryTime > 200*time.Millisecond {
		t.Errorf("Summary generation took too long: %v", summaryTime)
	}

	// 결과 검증
	if len(allStates) != numWorkspaces {
		t.Errorf("Expected %d states, got %d", numWorkspaces, len(allStates))
	}

	if summary.ActiveContainers != numWorkspaces {
		t.Errorf("Expected %d active containers, got %d", numWorkspaces, summary.ActiveContainers)
	}

	t.Logf("Performance results:")
	t.Logf("  Initial sync: %v", syncTime)
	t.Logf("  State query: %v", stateQueryTime)
	t.Logf("  Summary generation: %v", summaryTime)
	t.Logf("  Workspaces: %d", numWorkspaces)
}