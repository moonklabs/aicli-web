package claude

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// StreamEvent는 스트림에서 발생하는 이벤트를 나타냅니다.
type StreamEvent struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
	Source    string      `json:"source,omitempty"`
	ID        string      `json:"id,omitempty"`
}

// EventHandler는 이벤트를 처리하는 함수 타입입니다.
type EventHandler func(event *StreamEvent) error

// EventSubscription는 이벤트 구독 정보를 나타냅니다.
type EventSubscription struct {
	ID        string
	EventType string
	Handler   EventHandler
	CreatedAt time.Time
}

// EventBus는 이벤트 발행/구독을 관리하는 구조체입니다.
type EventBus struct {
	subscribers map[string][]*EventSubscription
	mutex       sync.RWMutex
	logger      *logrus.Logger
	metrics     *EventMetrics
	ctx         context.Context
	cancel      context.CancelFunc
}

// EventMetrics는 이벤트 버스의 메트릭을 추적합니다.
type EventMetrics struct {
	PublishedEvents   int64
	DeliveredEvents   int64
	FailedDeliveries  int64
	ActiveSubscribers int
	mutex             sync.RWMutex
}

// NewEventBus는 새로운 이벤트 버스를 생성합니다.
func NewEventBus(logger *logrus.Logger) *EventBus {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &EventBus{
		subscribers: make(map[string][]*EventSubscription),
		logger:      logger,
		metrics:     &EventMetrics{},
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Subscribe는 특정 이벤트 타입에 대한 핸들러를 등록합니다.
func (eb *EventBus) Subscribe(eventType string, handler EventHandler) (*EventSubscription, error) {
	if eventType == "" {
		return nil, fmt.Errorf("event type cannot be empty")
	}
	if handler == nil {
		return nil, fmt.Errorf("event handler cannot be nil")
	}

	eb.mutex.Lock()
	defer eb.mutex.Unlock()

	subscription := &EventSubscription{
		ID:        eb.generateSubscriptionID(),
		EventType: eventType,
		Handler:   handler,
		CreatedAt: time.Now(),
	}

	eb.subscribers[eventType] = append(eb.subscribers[eventType], subscription)
	
	eb.updateMetrics()
	
	eb.logger.WithFields(logrus.Fields{
		"event_type":      eventType,
		"subscription_id": subscription.ID,
	}).Debug("Event subscription created")

	return subscription, nil
}

// Unsubscribe는 구독을 취소합니다.
func (eb *EventBus) Unsubscribe(subscription *EventSubscription) error {
	if subscription == nil {
		return fmt.Errorf("subscription cannot be nil")
	}

	eb.mutex.Lock()
	defer eb.mutex.Unlock()

	subscribers, exists := eb.subscribers[subscription.EventType]
	if !exists {
		return fmt.Errorf("no subscribers for event type: %s", subscription.EventType)
	}

	// 구독자 제거
	for i, sub := range subscribers {
		if sub.ID == subscription.ID {
			eb.subscribers[subscription.EventType] = append(subscribers[:i], subscribers[i+1:]...)
			break
		}
	}

	// 구독자가 없으면 이벤트 타입 제거
	if len(eb.subscribers[subscription.EventType]) == 0 {
		delete(eb.subscribers, subscription.EventType)
	}

	eb.updateMetrics()

	eb.logger.WithFields(logrus.Fields{
		"event_type":      subscription.EventType,
		"subscription_id": subscription.ID,
	}).Debug("Event subscription removed")

	return nil
}

// Publish는 이벤트를 발행합니다.
func (eb *EventBus) Publish(event *StreamEvent) {
	if event == nil {
		eb.logger.Error("Cannot publish nil event")
		return
	}

	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	eb.mutex.RLock()
	handlers := eb.subscribers[event.Type]
	eb.mutex.RUnlock()

	eb.metrics.mutex.Lock()
	eb.metrics.PublishedEvents++
	eb.metrics.mutex.Unlock()

	if len(handlers) == 0 {
		eb.logger.WithField("event_type", event.Type).Debug("No subscribers for event")
		return
	}

	eb.logger.WithFields(logrus.Fields{
		"event_type":   event.Type,
		"subscribers":  len(handlers),
		"event_id":     event.ID,
	}).Debug("Publishing event")

	// 각 핸들러를 고루틴에서 실행
	for _, subscription := range handlers {
		go eb.handleEvent(subscription, event)
	}
}

// PublishSync는 이벤트를 동기적으로 발행합니다.
func (eb *EventBus) PublishSync(event *StreamEvent) []error {
	if event == nil {
		return []error{fmt.Errorf("cannot publish nil event")}
	}

	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	eb.mutex.RLock()
	handlers := eb.subscribers[event.Type]
	eb.mutex.RUnlock()

	eb.metrics.mutex.Lock()
	eb.metrics.PublishedEvents++
	eb.metrics.mutex.Unlock()

	if len(handlers) == 0 {
		eb.logger.WithField("event_type", event.Type).Debug("No subscribers for event")
		return nil
	}

	var errors []error
	for _, subscription := range handlers {
		if err := eb.callHandler(subscription.Handler, event); err != nil {
			errors = append(errors, err)
		}
	}

	return errors
}

// handleEvent는 개별 이벤트 핸들러를 실행합니다.
func (eb *EventBus) handleEvent(subscription *EventSubscription, event *StreamEvent) {
	defer func() {
		if r := recover(); r != nil {
			eb.logger.WithFields(logrus.Fields{
				"panic":           r,
				"subscription_id": subscription.ID,
				"event_type":      event.Type,
			}).Error("Event handler panicked")
			
			eb.metrics.mutex.Lock()
			eb.metrics.FailedDeliveries++
			eb.metrics.mutex.Unlock()
		}
	}()

	if err := eb.callHandler(subscription.Handler, event); err != nil {
		eb.logger.WithFields(logrus.Fields{
			"error":           err,
			"subscription_id": subscription.ID,
			"event_type":      event.Type,
		}).Error("Event handler error")
		
		eb.metrics.mutex.Lock()
		eb.metrics.FailedDeliveries++
		eb.metrics.mutex.Unlock()
	} else {
		eb.metrics.mutex.Lock()
		eb.metrics.DeliveredEvents++
		eb.metrics.mutex.Unlock()
	}
}

// callHandler는 핸들러를 호출하고 타임아웃을 적용합니다.
func (eb *EventBus) callHandler(handler EventHandler, event *StreamEvent) error {
	done := make(chan error, 1)
	
	go func() {
		done <- handler(event)
	}()

	select {
	case err := <-done:
		return err
	case <-time.After(5 * time.Second): // 핸들러 타임아웃
		return fmt.Errorf("event handler timeout")
	case <-eb.ctx.Done():
		return fmt.Errorf("event bus is shutting down")
	}
}

// GetSubscribers는 특정 이벤트 타입의 구독자 목록을 반환합니다.
func (eb *EventBus) GetSubscribers(eventType string) []*EventSubscription {
	eb.mutex.RLock()
	defer eb.mutex.RUnlock()

	subscribers := eb.subscribers[eventType]
	result := make([]*EventSubscription, len(subscribers))
	copy(result, subscribers)

	return result
}

// GetAllSubscribers는 모든 구독자 목록을 반환합니다.
func (eb *EventBus) GetAllSubscribers() map[string][]*EventSubscription {
	eb.mutex.RLock()
	defer eb.mutex.RUnlock()

	result := make(map[string][]*EventSubscription)
	for eventType, subscribers := range eb.subscribers {
		result[eventType] = make([]*EventSubscription, len(subscribers))
		copy(result[eventType], subscribers)
	}

	return result
}

// GetMetrics는 이벤트 버스의 메트릭을 반환합니다.
func (eb *EventBus) GetMetrics() map[string]interface{} {
	eb.metrics.mutex.RLock()
	defer eb.metrics.mutex.RUnlock()

	return map[string]interface{}{
		"published_events":    eb.metrics.PublishedEvents,
		"delivered_events":    eb.metrics.DeliveredEvents,
		"failed_deliveries":   eb.metrics.FailedDeliveries,
		"active_subscribers":  eb.metrics.ActiveSubscribers,
		"success_rate":        eb.calculateSuccessRate(),
	}
}

// Close는 이벤트 버스를 종료합니다.
func (eb *EventBus) Close() error {
	eb.cancel()
	
	eb.mutex.Lock()
	defer eb.mutex.Unlock()

	// 모든 구독자 정리
	for eventType := range eb.subscribers {
		delete(eb.subscribers, eventType)
	}

	eb.updateMetrics()
	eb.logger.Info("Event bus closed")

	return nil
}

// generateSubscriptionID는 고유한 구독 ID를 생성합니다.
func (eb *EventBus) generateSubscriptionID() string {
	return fmt.Sprintf("sub_%d", time.Now().UnixNano())
}

// updateMetrics는 메트릭을 업데이트합니다.
func (eb *EventBus) updateMetrics() {
	eb.metrics.mutex.Lock()
	defer eb.metrics.mutex.Unlock()

	count := 0
	for _, subscribers := range eb.subscribers {
		count += len(subscribers)
	}
	eb.metrics.ActiveSubscribers = count
}

// calculateSuccessRate는 이벤트 전달 성공률을 계산합니다.
func (eb *EventBus) calculateSuccessRate() float64 {
	if eb.metrics.PublishedEvents == 0 {
		return 0
	}
	return float64(eb.metrics.DeliveredEvents) / float64(eb.metrics.PublishedEvents) * 100
}