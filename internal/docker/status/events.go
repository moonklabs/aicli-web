package status

import (
	"fmt"
	"sync"
	"time"

	"github.com/aicli/aicli-web/internal/models"
)

// EventType 이벤트 타입
type EventType string

const (
	// EventTypeStatusChanged 상태 변경 이벤트
	EventTypeStatusChanged EventType = "status_changed"
	// EventTypeContainerUpdate 컨테이너 업데이트 이벤트
	EventTypeContainerUpdate EventType = "container_update"
	// EventTypeError 오류 이벤트
	EventTypeError EventType = "error"
	// EventTypeRecovery 복구 이벤트
	EventTypeRecovery EventType = "recovery"
	// EventTypeMetricsUpdate 메트릭 업데이트 이벤트
	EventTypeMetricsUpdate EventType = "metrics_update"
	// EventTypeSyncStart 동기화 시작 이벤트
	EventTypeSyncStart EventType = "sync_start"
	// EventTypeSyncComplete 동기화 완료 이벤트
	EventTypeSyncComplete EventType = "sync_complete"
)

// Event 이벤트 구조체
type Event struct {
	// 이벤트 타입
	Type EventType `json:"type"`
	// 워크스페이스 ID
	WorkspaceID string `json:"workspace_id"`
	// 타임스탬프
	Timestamp time.Time `json:"timestamp"`
	// 이벤트 데이터
	Data interface{} `json:"data"`
	// 메시지
	Message string `json:"message,omitempty"`
	// 이벤트 ID
	ID string `json:"id"`
}

// StatusChangeEvent 상태 변경 이벤트 데이터
type StatusChangeEvent struct {
	// 이전 상태
	OldStatus models.WorkspaceStatus `json:"old_status"`
	// 새로운 상태
	NewStatus models.WorkspaceStatus `json:"new_status"`
	// 변경 이유
	Reason string `json:"reason,omitempty"`
	// 컨테이너 ID
	ContainerID string `json:"container_id,omitempty"`
	// 변경 시간
	ChangedAt time.Time `json:"changed_at"`
}

// ContainerUpdateEvent 컨테이너 업데이트 이벤트 데이터
type ContainerUpdateEvent struct {
	// 컨테이너 ID
	ContainerID string `json:"container_id"`
	// 이전 상태
	OldState string `json:"old_state"`
	// 새로운 상태
	NewState string `json:"new_state"`
	// 업데이트 시간
	UpdatedAt time.Time `json:"updated_at"`
	// 메트릭 정보
	Metrics *WorkspaceMetrics `json:"metrics,omitempty"`
}

// ErrorEvent 오류 이벤트 데이터
type ErrorEvent struct {
	// 오류 메시지
	Error string `json:"error"`
	// 오류 코드
	Code string `json:"code,omitempty"`
	// 재시도 횟수
	RetryCount int `json:"retry_count"`
	// 발생 시간
	OccurredAt time.Time `json:"occurred_at"`
	// 컨텍스트 정보
	Context map[string]interface{} `json:"context,omitempty"`
}

// RecoveryEvent 복구 이벤트 데이터
type RecoveryEvent struct {
	// 복구된 오류
	ResolvedError string `json:"resolved_error"`
	// 복구 방법
	RecoveryMethod string `json:"recovery_method"`
	// 복구 시간
	RecoveredAt time.Time `json:"recovered_at"`
	// 다운타임
	Downtime time.Duration `json:"downtime"`
}

// MetricsUpdateEvent 메트릭 업데이트 이벤트 데이터
type MetricsUpdateEvent struct {
	// 이전 메트릭
	OldMetrics *WorkspaceMetrics `json:"old_metrics"`
	// 새로운 메트릭
	NewMetrics *WorkspaceMetrics `json:"new_metrics"`
	// 업데이트 시간
	UpdatedAt time.Time `json:"updated_at"`
	// 변화량
	Delta map[string]float64 `json:"delta,omitempty"`
}

// SyncEvent 동기화 이벤트 데이터
type SyncEvent struct {
	// 동기화된 워크스페이스 수
	WorkspaceCount int `json:"workspace_count"`
	// 소요 시간
	Duration time.Duration `json:"duration"`
	// 성공 수
	SuccessCount int `json:"success_count"`
	// 실패 수
	ErrorCount int `json:"error_count"`
	// 시작/완료 시간
	Timestamp time.Time `json:"timestamp"`
}

// createStateChangeEvent 상태 변경 이벤트 생성
func (t *Tracker) createStateChangeEvent(workspaceID string, oldState, newState *WorkspaceState) *Event {
	var reason string
	var oldStatus models.WorkspaceStatus

	if oldState == nil {
		reason = "workspace_initialized"
		oldStatus = ""
	} else if oldState.Status != newState.Status {
		reason = fmt.Sprintf("status_changed_%s_to_%s", oldState.Status, newState.Status)
		oldStatus = oldState.Status
	} else if oldState.ContainerState != newState.ContainerState {
		reason = fmt.Sprintf("container_state_changed_%s_to_%s", oldState.ContainerState, newState.ContainerState)
		oldStatus = oldState.Status
	} else {
		reason = "metrics_updated"
		oldStatus = oldState.Status
	}

	eventData := StatusChangeEvent{
		OldStatus:   oldStatus,
		NewStatus:   newState.Status,
		Reason:      reason,
		ContainerID: newState.ContainerID,
		ChangedAt:   newState.LastUpdated,
	}

	return &Event{
		ID:          generateEventID(),
		Type:        EventTypeStatusChanged,
		WorkspaceID: workspaceID,
		Timestamp:   time.Now(),
		Data:        eventData,
		Message:     fmt.Sprintf("워크스페이스 %s 상태 변경: %s", workspaceID, reason),
	}
}

// createContainerUpdateEvent 컨테이너 업데이트 이벤트 생성
func (t *Tracker) createContainerUpdateEvent(workspaceID, containerID, oldState, newState string, metrics *WorkspaceMetrics) *Event {
	eventData := ContainerUpdateEvent{
		ContainerID: containerID,
		OldState:    oldState,
		NewState:    newState,
		UpdatedAt:   time.Now(),
		Metrics:     metrics,
	}

	return &Event{
		ID:          generateEventID(),
		Type:        EventTypeContainerUpdate,
		WorkspaceID: workspaceID,
		Timestamp:   time.Now(),
		Data:        eventData,
		Message:     fmt.Sprintf("컨테이너 %s 상태 변경: %s -> %s", containerID, oldState, newState),
	}
}

// createErrorEvent 오류 이벤트 생성
func (t *Tracker) createErrorEvent(workspaceID string, err error, retryCount int, context map[string]interface{}) *Event {
	eventData := ErrorEvent{
		Error:      err.Error(),
		Code:       getErrorCode(err),
		RetryCount: retryCount,
		OccurredAt: time.Now(),
		Context:    context,
	}

	return &Event{
		ID:          generateEventID(),
		Type:        EventTypeError,
		WorkspaceID: workspaceID,
		Timestamp:   time.Now(),
		Data:        eventData,
		Message:     fmt.Sprintf("워크스페이스 %s 오류 발생: %s", workspaceID, err.Error()),
	}
}

// createRecoveryEvent 복구 이벤트 생성
func (t *Tracker) createRecoveryEvent(workspaceID, resolvedError, recoveryMethod string, downtime time.Duration) *Event {
	eventData := RecoveryEvent{
		ResolvedError:  resolvedError,
		RecoveryMethod: recoveryMethod,
		RecoveredAt:    time.Now(),
		Downtime:       downtime,
	}

	return &Event{
		ID:          generateEventID(),
		Type:        EventTypeRecovery,
		WorkspaceID: workspaceID,
		Timestamp:   time.Now(),
		Data:        eventData,
		Message:     fmt.Sprintf("워크스페이스 %s 복구 완료: %s (다운타임: %v)", workspaceID, recoveryMethod, downtime),
	}
}

// createMetricsUpdateEvent 메트릭 업데이트 이벤트 생성
func (t *Tracker) createMetricsUpdateEvent(workspaceID string, oldMetrics, newMetrics *WorkspaceMetrics) *Event {
	var delta map[string]float64
	if oldMetrics != nil && newMetrics != nil {
		delta = map[string]float64{
			"cpu_percent_delta":  newMetrics.CPUPercent - oldMetrics.CPUPercent,
			"memory_usage_delta": float64(newMetrics.MemoryUsage - oldMetrics.MemoryUsage),
			"network_rx_delta":   newMetrics.NetworkRxMB - oldMetrics.NetworkRxMB,
			"network_tx_delta":   newMetrics.NetworkTxMB - oldMetrics.NetworkTxMB,
		}
	}

	eventData := MetricsUpdateEvent{
		OldMetrics: oldMetrics,
		NewMetrics: newMetrics,
		UpdatedAt:  time.Now(),
		Delta:      delta,
	}

	return &Event{
		ID:          generateEventID(),
		Type:        EventTypeMetricsUpdate,
		WorkspaceID: workspaceID,
		Timestamp:   time.Now(),
		Data:        eventData,
		Message:     fmt.Sprintf("워크스페이스 %s 메트릭 업데이트", workspaceID),
	}
}

// createSyncEvent 동기화 이벤트 생성
func (t *Tracker) createSyncEvent(eventType EventType, workspaceCount, successCount, errorCount int, duration time.Duration) *Event {
	eventData := SyncEvent{
		WorkspaceCount: workspaceCount,
		Duration:       duration,
		SuccessCount:   successCount,
		ErrorCount:     errorCount,
		Timestamp:      time.Now(),
	}

	message := ""
	switch eventType {
	case EventTypeSyncStart:
		message = fmt.Sprintf("동기화 시작: %d개 워크스페이스", workspaceCount)
	case EventTypeSyncComplete:
		message = fmt.Sprintf("동기화 완료: %d개 워크스페이스 (성공: %d, 실패: %d, 소요시간: %v)", 
			workspaceCount, successCount, errorCount, duration)
	}

	return &Event{
		ID:          generateEventID(),
		Type:        eventType,
		WorkspaceID: "", // 전체 시스템 이벤트
		Timestamp:   time.Now(),
		Data:        eventData,
		Message:     message,
	}
}

// emitEvent 이벤트 발송
func (t *Tracker) emitEvent(event *Event) {
	// 이벤트 로깅
	t.logger.Debug("이벤트 발송: [%s] %s - %s", event.Type, event.WorkspaceID, event.Message)
	
	// 향후 확장 가능한 부분:
	// - 이벤트 버스로 전송
	// - WebSocket으로 실시간 전송
	// - 이벤트 저장소에 저장
	// - 외부 시스템 알림
	
	// 현재는 로깅만 수행
	if t.logger != nil {
		switch event.Type {
		case EventTypeError:
			if errorData, ok := event.Data.(ErrorEvent); ok {
				t.logger.Error("이벤트", fmt.Errorf(errorData.Error), 
					"event_id", event.ID,
					"workspace_id", event.WorkspaceID,
					"retry_count", errorData.RetryCount)
			}
		case EventTypeStatusChanged:
			if statusData, ok := event.Data.(StatusChangeEvent); ok {
				t.logger.Info("상태 변경 이벤트: %s [%s -> %s]", 
					event.WorkspaceID, 
					statusData.OldStatus, 
					statusData.NewStatus)
			}
		default:
			t.logger.Debug("이벤트 처리: %s", event.Message)
		}
	}
}

// generateEventID 이벤트 ID 생성
func generateEventID() string {
	return fmt.Sprintf("evt_%d", time.Now().UnixNano())
}

// getErrorCode 오류 코드 추출
func getErrorCode(err error) string {
	// 오류 타입에 따른 코드 분류
	if err == nil {
		return ""
	}

	errStr := err.Error()
	switch {
	case contains(errStr, "connection"):
		return "CONNECTION_ERROR"
	case contains(errStr, "timeout"):
		return "TIMEOUT_ERROR"
	case contains(errStr, "not found"):
		return "NOT_FOUND_ERROR"
	case contains(errStr, "permission"):
		return "PERMISSION_ERROR"
	case contains(errStr, "container"):
		return "CONTAINER_ERROR"
	case contains(errStr, "network"):
		return "NETWORK_ERROR"
	default:
		return "UNKNOWN_ERROR"
	}
}

// contains 문자열 포함 여부 확인 (대소문자 무시)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || 
		len(s) > len(substr) && (
			s[:len(substr)] == substr || 
			s[len(s)-len(substr):] == substr ||
			containsInner(s, substr)))
}

// containsInner 내부 문자열 포함 확인
func containsInner(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// EventHandler 이벤트 핸들러 인터페이스
type EventHandler interface {
	HandleEvent(event *Event) error
}

// EventBus 이벤트 버스 (향후 확장)
type EventBus struct {
	handlers []EventHandler
	mu       sync.RWMutex
}

// NewEventBus 새로운 이벤트 버스 생성
func NewEventBus() *EventBus {
	return &EventBus{
		handlers: make([]EventHandler, 0),
	}
}

// Subscribe 핸들러 구독
func (eb *EventBus) Subscribe(handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.handlers = append(eb.handlers, handler)
}

// Publish 이벤트 발송
func (eb *EventBus) Publish(event *Event) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	
	for _, handler := range eb.handlers {
		go func(h EventHandler) {
			defer func() {
				if r := recover(); r != nil {
					// 핸들러 패닉 복구
				}
			}()
			h.HandleEvent(event)
		}(handler)
	}
}