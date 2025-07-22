// +build standalone

package status

import (
	"context"
	"testing"
	"time"

	"github.com/aicli/aicli-web/internal/models"
)

// 독립적인 상태 추적 시스템 테스트
// 다른 패키지에 의존하지 않는 순수한 유닛 테스트

// TestStandalone_TrackerBasics 기본 추적자 기능 테스트
func TestStandalone_TrackerBasics(t *testing.T) {
	// Mock 구현체 생성
	mockService := NewMockWorkspaceService()
	mockContainer := NewMockContainerManager()
	mockFactory := NewMockDockerManager()

	// 추적자 생성
	tracker := NewTracker(mockService, mockContainer, mockFactory)

	// 기본 설정 확인
	if tracker.syncInterval != 30*time.Second {
		t.Errorf("Expected sync interval 30s, got %v", tracker.syncInterval)
	}

	if tracker.maxRetries != 3 {
		t.Errorf("Expected max retries 3, got %d", tracker.maxRetries)
	}

	// 시작/중지 테스트
	err := tracker.Start()
	if err != nil {
		t.Fatalf("Failed to start tracker: %v", err)
	}

	// 잠시 실행
	time.Sleep(50 * time.Millisecond)

	err = tracker.Stop()
	if err != nil {
		t.Fatalf("Failed to stop tracker: %v", err)
	}
}

// TestStandalone_WorkspaceState 워크스페이스 상태 구조체 테스트
func TestStandalone_WorkspaceState(t *testing.T) {
	state := &WorkspaceState{
		WorkspaceID: "ws-test-123",
		Name:        "Test Workspace",
		Status:      models.WorkspaceStatusActive,
		Metrics: &WorkspaceMetrics{
			CPUPercent:   25.5,
			MemoryUsage:  512 * 1024 * 1024,
			NetworkRxMB:  100.0,
			NetworkTxMB:  50.0,
			LastActivity: time.Now(),
		},
	}

	// 상태 검증
	if state.WorkspaceID != "ws-test-123" {
		t.Errorf("Expected workspace ID ws-test-123, got %s", state.WorkspaceID)
	}

	if state.Status != models.WorkspaceStatusActive {
		t.Errorf("Expected status active, got %s", state.Status)
	}

	// 메트릭 검증
	if state.Metrics.CPUPercent != 25.5 {
		t.Errorf("Expected CPU percent 25.5, got %f", state.Metrics.CPUPercent)
	}

	if state.Metrics.MemoryUsage != 512*1024*1024 {
		t.Errorf("Expected memory usage 512MB, got %d", state.Metrics.MemoryUsage)
	}
}

// TestStandalone_EventCreation 이벤트 생성 테스트
func TestStandalone_EventCreation(t *testing.T) {
	tracker := NewTracker(nil, nil, nil)

	oldState := &WorkspaceState{
		WorkspaceID: "ws-event-test",
		Status:      models.WorkspaceStatusInactive,
	}

	newState := &WorkspaceState{
		WorkspaceID: "ws-event-test",
		Status:      models.WorkspaceStatusActive,
		LastUpdated: time.Now(),
	}

	event := tracker.createStateChangeEvent("ws-event-test", oldState, newState)

	// 이벤트 검증
	if event.Type != EventTypeStatusChanged {
		t.Errorf("Expected event type %s, got %s", EventTypeStatusChanged, event.Type)
	}

	if event.WorkspaceID != "ws-event-test" {
		t.Errorf("Expected workspace ID ws-event-test, got %s", event.WorkspaceID)
	}

	// 이벤트 데이터 검증
	if statusData, ok := event.Data.(StatusChangeEvent); ok {
		if statusData.OldStatus != models.WorkspaceStatusInactive {
			t.Errorf("Expected old status inactive, got %s", statusData.OldStatus)
		}

		if statusData.NewStatus != models.WorkspaceStatusActive {
			t.Errorf("Expected new status active, got %s", statusData.NewStatus)
		}
	} else {
		t.Error("Event data is not StatusChangeEvent")
	}
}

// TestStandalone_ResourceMonitorBasics 리소스 모니터 기본 기능 테스트
func TestStandalone_ResourceMonitorBasics(t *testing.T) {
	mockContainer := NewMockContainerManager()
	mockFactory := NewMockDockerManager()

	monitor := NewResourceMonitor(mockContainer, mockFactory)

	// 기본 설정 확인
	if monitor.collectInterval != 10*time.Second {
		t.Errorf("Expected collect interval 10s, got %v", monitor.collectInterval)
	}

	if monitor.cacheExpiry != 30*time.Second {
		t.Errorf("Expected cache expiry 30s, got %v", monitor.cacheExpiry)
	}

	// 시작/중지 테스트
	err := monitor.Start()
	if err != nil {
		t.Fatalf("Failed to start monitor: %v", err)
	}

	err = monitor.Stop()
	if err != nil {
		t.Fatalf("Failed to stop monitor: %v", err)
	}
}

// TestStandalone_MetricsStructure 메트릭 구조체 테스트
func TestStandalone_MetricsStructure(t *testing.T) {
	metrics := &WorkspaceMetrics{
		CPUPercent:   45.0,
		MemoryUsage:  1024 * 1024 * 1024, // 1GB
		MemoryLimit:  2048 * 1024 * 1024, // 2GB
		NetworkRxMB:  150.5,
		NetworkTxMB:  75.2,
		Uptime:       "2h30m15s",
		LastActivity: time.Now(),
		ErrorCount:   0,
	}

	// 메트릭 검증
	if metrics.CPUPercent != 45.0 {
		t.Errorf("Expected CPU percent 45.0, got %f", metrics.CPUPercent)
	}

	if metrics.MemoryUsage != 1024*1024*1024 {
		t.Errorf("Expected memory usage 1GB, got %d", metrics.MemoryUsage)
	}

	if metrics.NetworkRxMB != 150.5 {
		t.Errorf("Expected network RX 150.5MB, got %f", metrics.NetworkRxMB)
	}

	if metrics.Uptime != "2h30m15s" {
		t.Errorf("Expected uptime 2h30m15s, got %s", metrics.Uptime)
	}
}

// TestStandalone_TrackerStats 추적자 통계 테스트
func TestStandalone_TrackerStats(t *testing.T) {
	tracker := NewTracker(nil, nil, nil)

	stats := tracker.GetStats()

	if stats.SyncInterval != 30*time.Second {
		t.Errorf("Expected sync interval 30s, got %v", stats.SyncInterval)
	}

	if stats.ActiveCallbacks != 0 {
		t.Errorf("Expected 0 active callbacks, got %d", stats.ActiveCallbacks)
	}

	if stats.TotalWorkspaces != 0 {
		t.Errorf("Expected 0 total workspaces, got %d", stats.TotalWorkspaces)
	}
}

// TestStandalone_EventTypes 이벤트 타입 테스트
func TestStandalone_EventTypes(t *testing.T) {
	expectedTypes := []EventType{
		EventTypeStatusChanged,
		EventTypeContainerUpdate,
		EventTypeError,
		EventTypeRecovery,
		EventTypeMetricsUpdate,
		EventTypeSyncStart,
		EventTypeSyncComplete,
	}

	for _, eventType := range expectedTypes {
		if string(eventType) == "" {
			t.Errorf("Event type %v should not be empty", eventType)
		}
	}

	// 각 이벤트 타입이 고유한지 확인
	typeMap := make(map[EventType]bool)
	for _, eventType := range expectedTypes {
		if typeMap[eventType] {
			t.Errorf("Duplicate event type: %s", eventType)
		}
		typeMap[eventType] = true
	}
}

// TestStandalone_ErrorHandling 에러 처리 테스트
func TestStandalone_ErrorHandling(t *testing.T) {
	// 에러 코드 생성 테스트
	testCases := []struct {
		error    string
		expected string
	}{
		{"connection failed", "CONNECTION_ERROR"},
		{"timeout occurred", "TIMEOUT_ERROR"},
		{"not found", "NOT_FOUND_ERROR"},
		{"permission denied", "PERMISSION_ERROR"},
		{"container error", "CONTAINER_ERROR"},
		{"network issue", "NETWORK_ERROR"},
		{"unknown problem", "UNKNOWN_ERROR"},
	}

	for _, tc := range testCases {
		code := getErrorCode(&TestError{msg: tc.error})
		if code != tc.expected {
			t.Errorf("Expected error code %s for error '%s', got %s", 
				tc.expected, tc.error, code)
		}
	}
}

// TestError 테스트용 에러 구조체
type TestError struct {
	msg string
}

func (e *TestError) Error() string {
	return e.msg
}

// TestStandalone_EventID 이벤트 ID 생성 테스트
func TestStandalone_EventID(t *testing.T) {
	// 여러 개의 이벤트 ID 생성
	ids := make([]string, 10)
	for i := 0; i < 10; i++ {
		ids[i] = generateEventID()
		time.Sleep(1 * time.Millisecond) // ID 중복 방지
	}

	// 모든 ID가 고유한지 확인
	idMap := make(map[string]bool)
	for _, id := range ids {
		if id == "" {
			t.Error("Event ID should not be empty")
		}

		if idMap[id] {
			t.Errorf("Duplicate event ID: %s", id)
		}
		idMap[id] = true

		// ID 형식 확인 (evt_ 프리픽스)
		if len(id) < 5 || id[:4] != "evt_" {
			t.Errorf("Event ID should start with 'evt_', got: %s", id)
		}
	}
}