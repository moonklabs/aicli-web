package claude

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// SessionEventType은 세션 이벤트 타입을 정의합니다
type SessionEventType int

const (
	SessionEventCreated SessionEventType = iota
	SessionEventStarted
	SessionEventSuspended
	SessionEventResumed
	SessionEventClosed
	SessionEventError
	SessionEventStateChanged
	SessionEventConfigUpdated
	SessionEventMetadataUpdated
)

// String은 SessionEventType의 문자열 표현을 반환합니다
func (t SessionEventType) String() string {
	types := []string{
		"created",
		"started",
		"suspended",
		"resumed",
		"closed",
		"error",
		"state_changed",
		"config_updated",
		"metadata_updated",
	}
	if int(t) < len(types) {
		return types[t]
	}
	return "unknown"
}

// SessionEvent는 세션 이벤트를 나타냅니다
type SessionEvent struct {
	SessionID string           `json:"session_id"`
	Type      SessionEventType `json:"type"`
	Timestamp time.Time        `json:"timestamp"`
	Data      interface{}      `json:"data,omitempty"`
	Error     error            `json:"error,omitempty"`
}

// SessionEventListener는 세션 이벤트를 수신하는 인터페이스입니다
type SessionEventListener interface {
	OnSessionEvent(event SessionEvent)
}

// SessionEventHandler는 이벤트 핸들러 함수 타입입니다
type SessionEventHandler func(event SessionEvent)

// SessionEventBus는 세션 이벤트 버스를 관리합니다
type SessionEventBus struct {
	listeners map[string][]SessionEventListener
	handlers  map[SessionEventType][]SessionEventHandler
	buffer    chan SessionEvent
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	mu        sync.RWMutex
}

// NewSessionEventBus는 새로운 세션 이벤트 버스를 생성합니다
func NewSessionEventBus(bufferSize int) *SessionEventBus {
	ctx, cancel := context.WithCancel(context.Background())
	
	bus := &SessionEventBus{
		listeners: make(map[string][]SessionEventListener),
		handlers:  make(map[SessionEventType][]SessionEventHandler),
		buffer:    make(chan SessionEvent, bufferSize),
		ctx:       ctx,
		cancel:    cancel,
	}

	// 이벤트 처리 고루틴 시작
	bus.wg.Add(1)
	go bus.processEvents()

	return bus
}

// Subscribe는 특정 세션의 이벤트를 구독합니다
func (bus *SessionEventBus) Subscribe(sessionID string, listener SessionEventListener) {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	if _, exists := bus.listeners[sessionID]; !exists {
		bus.listeners[sessionID] = []SessionEventListener{}
	}
	bus.listeners[sessionID] = append(bus.listeners[sessionID], listener)
}

// Unsubscribe는 구독을 취소합니다
func (bus *SessionEventBus) Unsubscribe(sessionID string, listener SessionEventListener) {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	if listeners, exists := bus.listeners[sessionID]; exists {
		filtered := []SessionEventListener{}
		for _, l := range listeners {
			if l != listener {
				filtered = append(filtered, l)
			}
		}
		if len(filtered) == 0 {
			delete(bus.listeners, sessionID)
		} else {
			bus.listeners[sessionID] = filtered
		}
	}
}

// SubscribeToType는 특정 타입의 이벤트를 구독합니다
func (bus *SessionEventBus) SubscribeToType(eventType SessionEventType, handler SessionEventHandler) {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	if _, exists := bus.handlers[eventType]; !exists {
		bus.handlers[eventType] = []SessionEventHandler{}
	}
	bus.handlers[eventType] = append(bus.handlers[eventType], handler)
}

// Publish는 이벤트를 발행합니다
func (bus *SessionEventBus) Publish(event SessionEvent) {
	// 타임스탬프 설정
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// 비블로킹 전송
	select {
	case bus.buffer <- event:
	case <-bus.ctx.Done():
		return
	default:
		// 버퍼가 가득 찬 경우 로그
		fmt.Printf("Session event buffer full, dropping event: %+v\n", event)
	}
}

// processEvents는 이벤트를 처리하는 고루틴입니다
func (bus *SessionEventBus) processEvents() {
	defer bus.wg.Done()

	for {
		select {
		case <-bus.ctx.Done():
			return
		case event := <-bus.buffer:
			bus.handleEvent(event)
		}
	}
}

// handleEvent는 개별 이벤트를 처리합니다
func (bus *SessionEventBus) handleEvent(event SessionEvent) {
	bus.mu.RLock()
	defer bus.mu.RUnlock()

	// 세션별 리스너에 전달
	if listeners, exists := bus.listeners[event.SessionID]; exists {
		for _, listener := range listeners {
			// 패닉 방지를 위한 recover
			func(l SessionEventListener) {
				defer func() {
					if r := recover(); r != nil {
						fmt.Printf("Panic in session event listener: %v\n", r)
					}
				}()
				l.OnSessionEvent(event)
			}(listener)
		}
	}

	// 타입별 핸들러에 전달
	if handlers, exists := bus.handlers[event.Type]; exists {
		for _, handler := range handlers {
			// 패닉 방지를 위한 recover
			func(h SessionEventHandler) {
				defer func() {
					if r := recover(); r != nil {
						fmt.Printf("Panic in session event handler: %v\n", r)
					}
				}()
				h(event)
			}(handler)
		}
	}
}

// Shutdown은 이벤트 버스를 종료합니다
func (bus *SessionEventBus) Shutdown() {
	bus.cancel()
	bus.wg.Wait()
	close(bus.buffer)
}

// SessionEventRecorder는 세션 이벤트를 기록하는 리스너입니다
type SessionEventRecorder struct {
	events []SessionEvent
	mu     sync.RWMutex
}

// NewSessionEventRecorder는 새로운 이벤트 레코더를 생성합니다
func NewSessionEventRecorder() *SessionEventRecorder {
	return &SessionEventRecorder{
		events: []SessionEvent{},
	}
}

// OnSessionEvent는 이벤트를 기록합니다
func (r *SessionEventRecorder) OnSessionEvent(event SessionEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, event)
}

// GetEvents는 기록된 이벤트를 반환합니다
func (r *SessionEventRecorder) GetEvents() []SessionEvent {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	// 복사본 반환
	events := make([]SessionEvent, len(r.events))
	copy(events, r.events)
	return events
}

// GetEventsByType는 특정 타입의 이벤트만 반환합니다
func (r *SessionEventRecorder) GetEventsByType(eventType SessionEventType) []SessionEvent {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	filtered := []SessionEvent{}
	for _, event := range r.events {
		if event.Type == eventType {
			filtered = append(filtered, event)
		}
	}
	return filtered
}

// Clear는 기록된 이벤트를 초기화합니다
func (r *SessionEventRecorder) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = []SessionEvent{}
}

// SessionEventLogger는 이벤트를 로그로 출력하는 리스너입니다
type SessionEventLogger struct {
	logger func(format string, args ...interface{})
}

// NewSessionEventLogger는 새로운 이벤트 로거를 생성합니다
func NewSessionEventLogger(logger func(format string, args ...interface{})) *SessionEventLogger {
	if logger == nil {
		// fmt.Printf를 래핑하여 적절한 시그니처로 변환
		logger = func(format string, args ...interface{}) {
			fmt.Printf(format, args...)
		}
	}
	return &SessionEventLogger{logger: logger}
}

// OnSessionEvent는 이벤트를 로그로 출력합니다
func (l *SessionEventLogger) OnSessionEvent(event SessionEvent) {
	if event.Error != nil {
		l.logger("[%s] Session %s: %s (error: %v, data: %+v)\n",
			event.Timestamp.Format(time.RFC3339),
			event.SessionID,
			event.Type,
			event.Error,
			event.Data)
	} else {
		l.logger("[%s] Session %s: %s (data: %+v)\n",
			event.Timestamp.Format(time.RFC3339),
			event.SessionID,
			event.Type,
			event.Data)
	}
}

// EventData 타입들
type StateChangeData struct {
	OldState SessionState `json:"old_state"`
	NewState SessionState `json:"new_state"`
}

type ConfigUpdateData struct {
	Field    string      `json:"field"`
	OldValue interface{} `json:"old_value"`
	NewValue interface{} `json:"new_value"`
}

type ErrorData struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}