package auth

import (
	"context"
	"fmt"
	"sync"
	"time"
	
	"github.com/aicli/aicli-web/internal/models"
)

// RBACEventType RBAC 이벤트 타입
type RBACEventType string

const (
	// 역할 관련 이벤트
	EventRoleCreated   RBACEventType = "role.created"
	EventRoleUpdated   RBACEventType = "role.updated"
	EventRoleDeleted   RBACEventType = "role.deleted"
	EventRoleAssigned  RBACEventType = "role.assigned"
	EventRoleRevoked   RBACEventType = "role.revoked"
	
	// 권한 관련 이벤트
	EventPermissionCreated   RBACEventType = "permission.created"
	EventPermissionUpdated   RBACEventType = "permission.updated"
	EventPermissionDeleted   RBACEventType = "permission.deleted"
	EventPermissionGranted   RBACEventType = "permission.granted"
	EventPermissionDenied    RBACEventType = "permission.denied"
	
	// 그룹 관련 이벤트
	EventGroupCreated      RBACEventType = "group.created"
	EventGroupUpdated      RBACEventType = "group.updated"
	EventGroupDeleted      RBACEventType = "group.deleted"
	EventGroupMemberAdded  RBACEventType = "group.member.added"
	EventGroupMemberRemoved RBACEventType = "group.member.removed"
	
	// 캐시 관련 이벤트
	EventCacheInvalidated RBACEventType = "cache.invalidated"
)

// RBACEvent RBAC 이벤트 구조체
type RBACEvent struct {
	ID         string                 `json:"id"`
	Type       RBACEventType         `json:"type"`
	Timestamp  time.Time             `json:"timestamp"`
	UserID     string                `json:"user_id,omitempty"`     // 이벤트를 발생시킨 사용자
	TargetID   string                `json:"target_id,omitempty"`   // 대상 ID (역할, 권한, 그룹 등)
	TargetType string                `json:"target_type,omitempty"` // 대상 타입
	ResourceID string                `json:"resource_id,omitempty"` // 관련 리소스 ID
	Metadata   map[string]interface{} `json:"metadata,omitempty"`    // 추가 메타데이터
	Context    *RBACEventContext     `json:"context,omitempty"`     // 이벤트 컨텍스트
}

// RBACEventContext 이벤트 컨텍스트
type RBACEventContext struct {
	IPAddress   string            `json:"ip_address,omitempty"`
	UserAgent   string            `json:"user_agent,omitempty"`
	RequestID   string            `json:"request_id,omitempty"`
	SessionID   string            `json:"session_id,omitempty"`
	Attributes  map[string]string `json:"attributes,omitempty"`
}

// RBACEventHandler 이벤트 핸들러 인터페이스
type RBACEventHandler interface {
	HandleEvent(ctx context.Context, event *RBACEvent) error
}

// RBACEventBus RBAC 이벤트 버스
type RBACEventBus struct {
	handlers map[RBACEventType][]RBACEventHandler
	mu       sync.RWMutex
	logger   EventLogger
}

// EventLogger 이벤트 로거 인터페이스
type EventLogger interface {
	LogEvent(event *RBACEvent) error
}

// NewRBACEventBus 이벤트 버스 생성자
func NewRBACEventBus(logger EventLogger) *RBACEventBus {
	return &RBACEventBus{
		handlers: make(map[RBACEventType][]RBACEventHandler),
		logger:   logger,
	}
}

// RegisterHandler 이벤트 핸들러 등록
func (bus *RBACEventBus) RegisterHandler(eventType RBACEventType, handler RBACEventHandler) {
	bus.mu.Lock()
	defer bus.mu.Unlock()
	
	if _, exists := bus.handlers[eventType]; !exists {
		bus.handlers[eventType] = make([]RBACEventHandler, 0)
	}
	bus.handlers[eventType] = append(bus.handlers[eventType], handler)
}

// PublishEvent 이벤트 발행
func (bus *RBACEventBus) PublishEvent(ctx context.Context, event *RBACEvent) error {
	// 이벤트 ID 생성
	if event.ID == "" {
		event.ID = generateEventID()
	}
	
	// 타임스탬프 설정
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	
	// 이벤트 로깅
	if bus.logger != nil {
		if err := bus.logger.LogEvent(event); err != nil {
			fmt.Printf("Failed to log event: %v\n", err)
		}
	}
	
	// 핸들러 호출
	bus.mu.RLock()
	handlers, exists := bus.handlers[event.Type]
	bus.mu.RUnlock()
	
	if !exists || len(handlers) == 0 {
		return nil // 핸들러가 없어도 오류가 아님
	}
	
	// 각 핸들러를 병렬로 실행
	var wg sync.WaitGroup
	errChan := make(chan error, len(handlers))
	
	for _, handler := range handlers {
		wg.Add(1)
		go func(h RBACEventHandler) {
			defer wg.Done()
			if err := h.HandleEvent(ctx, event); err != nil {
				errChan <- fmt.Errorf("handler error: %w", err)
			}
		}(handler)
	}
	
	wg.Wait()
	close(errChan)
	
	// 에러 수집
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("event handling errors: %v", errors)
	}
	
	return nil
}

// CacheInvalidationHandler 캐시 무효화 핸들러
type CacheInvalidationHandler struct {
	rbacManager *RBACManager
}

// NewCacheInvalidationHandler 캐시 무효화 핸들러 생성자
func NewCacheInvalidationHandler(rbacManager *RBACManager) *CacheInvalidationHandler {
	return &CacheInvalidationHandler{
		rbacManager: rbacManager,
	}
}

// HandleEvent 이벤트 처리
func (h *CacheInvalidationHandler) HandleEvent(ctx context.Context, event *RBACEvent) error {
	switch event.Type {
	case EventRoleCreated, EventRoleUpdated, EventRoleDeleted:
		return h.rbacManager.InvalidateRolePermissions(event.TargetID)
		
	case EventRoleAssigned, EventRoleRevoked:
		// 사용자 캐시 무효화
		if userID, ok := event.Metadata["user_id"].(string); ok {
			if err := h.rbacManager.InvalidateUserPermissions(userID); err != nil {
				return fmt.Errorf("failed to invalidate user cache: %w", err)
			}
		}
		// 역할 캐시도 무효화
		return h.rbacManager.InvalidateRolePermissions(event.TargetID)
		
	case EventPermissionCreated, EventPermissionUpdated, EventPermissionDeleted:
		// 권한 변경은 전체적인 영향을 미칠 수 있으므로 관련 역할들의 캐시 무효화
		if roleIDs, ok := event.Metadata["affected_roles"].([]string); ok {
			for _, roleID := range roleIDs {
				if err := h.rbacManager.InvalidateRolePermissions(roleID); err != nil {
					fmt.Printf("Failed to invalidate role %s cache: %v\n", roleID, err)
				}
			}
		}
		
	case EventGroupCreated, EventGroupUpdated, EventGroupDeleted, EventGroupMemberAdded, EventGroupMemberRemoved:
		return h.rbacManager.InvalidateGroupPermissions(event.TargetID)
	}
	
	return nil
}

// AuditLogHandler 감사 로그 핸들러
type AuditLogHandler struct {
	logger AuditLogger
}

// AuditLogger 감사 로거 인터페이스
type AuditLogger interface {
	LogAuditEvent(event *RBACEvent) error
}

// NewAuditLogHandler 감사 로그 핸들러 생성자
func NewAuditLogHandler(logger AuditLogger) *AuditLogHandler {
	return &AuditLogHandler{
		logger: logger,
	}
}

// HandleEvent 이벤트 처리
func (h *AuditLogHandler) HandleEvent(ctx context.Context, event *RBACEvent) error {
	return h.logger.LogAuditEvent(event)
}

// MetricsHandler 메트릭 수집 핸들러
type MetricsHandler struct {
	collector MetricsCollector
}

// MetricsCollector 메트릭 수집기 인터페이스
type MetricsCollector interface {
	IncrementCounter(name string, tags map[string]string)
	RecordGauge(name string, value float64, tags map[string]string)
	RecordHistogram(name string, value float64, tags map[string]string)
}

// NewMetricsHandler 메트릭 핸들러 생성자
func NewMetricsHandler(collector MetricsCollector) *MetricsHandler {
	return &MetricsHandler{
		collector: collector,
	}
}

// HandleEvent 이벤트 처리
func (h *MetricsHandler) HandleEvent(ctx context.Context, event *RBACEvent) error {
	tags := map[string]string{
		"event_type":   string(event.Type),
		"target_type":  event.TargetType,
	}
	
	// 이벤트 발생 카운터 증가
	h.collector.IncrementCounter("rbac.events.total", tags)
	
	// 이벤트 타입별 특별 메트릭
	switch event.Type {
	case EventPermissionDenied:
		h.collector.IncrementCounter("rbac.permission.denied.total", tags)
	case EventPermissionGranted:
		h.collector.IncrementCounter("rbac.permission.granted.total", tags)
	case EventCacheInvalidated:
		h.collector.IncrementCounter("rbac.cache.invalidations.total", tags)
	}
	
	return nil
}

// NotificationHandler 알림 핸들러
type NotificationHandler struct {
	notifier EventNotifier
}

// EventNotifier 이벤트 알리미 인터페이스
type EventNotifier interface {
	SendNotification(event *RBACEvent) error
}

// NewNotificationHandler 알림 핸들러 생성자
func NewNotificationHandler(notifier EventNotifier) *NotificationHandler {
	return &NotificationHandler{
		notifier: notifier,
	}
}

// HandleEvent 이벤트 처리
func (h *NotificationHandler) HandleEvent(ctx context.Context, event *RBACEvent) error {
	// 중요한 이벤트만 알림
	importantEvents := map[RBACEventType]bool{
		EventRoleAssigned:       true,
		EventRoleRevoked:        true,
		EventPermissionDenied:   true,
		EventGroupMemberAdded:   true,
		EventGroupMemberRemoved: true,
	}
	
	if importantEvents[event.Type] {
		return h.notifier.SendNotification(event)
	}
	
	return nil
}

// RBACEventManager RBAC 이벤트 관리자
type RBACEventManager struct {
	eventBus *RBACEventBus
	ctx      context.Context
}

// NewRBACEventManager 이벤트 관리자 생성자
func NewRBACEventManager(eventBus *RBACEventBus) *RBACEventManager {
	return &RBACEventManager{
		eventBus: eventBus,
		ctx:      context.Background(),
	}
}

// EmitRoleEvent 역할 관련 이벤트 발행
func (em *RBACEventManager) EmitRoleEvent(eventType RBACEventType, roleID, userID string, metadata map[string]interface{}) error {
	event := &RBACEvent{
		Type:       eventType,
		UserID:     userID,
		TargetID:   roleID,
		TargetType: "role",
		Metadata:   metadata,
	}
	
	return em.eventBus.PublishEvent(em.ctx, event)
}

// EmitPermissionEvent 권한 관련 이벤트 발행
func (em *RBACEventManager) EmitPermissionEvent(eventType RBACEventType, permissionID, userID string, metadata map[string]interface{}) error {
	event := &RBACEvent{
		Type:       eventType,
		UserID:     userID,
		TargetID:   permissionID,
		TargetType: "permission",
		Metadata:   metadata,
	}
	
	return em.eventBus.PublishEvent(em.ctx, event)
}

// EmitGroupEvent 그룹 관련 이벤트 발행
func (em *RBACEventManager) EmitGroupEvent(eventType RBACEventType, groupID, userID string, metadata map[string]interface{}) error {
	event := &RBACEvent{
		Type:       eventType,
		UserID:     userID,
		TargetID:   groupID,
		TargetType: "group",
		Metadata:   metadata,
	}
	
	return em.eventBus.PublishEvent(em.ctx, event)
}

// EmitCacheEvent 캐시 관련 이벤트 발행
func (em *RBACEventManager) EmitCacheEvent(cacheType, targetID, userID string) error {
	event := &RBACEvent{
		Type:       EventCacheInvalidated,
		UserID:     userID,
		TargetID:   targetID,
		TargetType: cacheType,
		Metadata: map[string]interface{}{
			"cache_type": cacheType,
		},
	}
	
	return em.eventBus.PublishEvent(em.ctx, event)
}

// 헬퍼 함수들

// generateEventID 이벤트 ID 생성
func generateEventID() string {
	return fmt.Sprintf("rbac_event_%d", time.Now().UnixNano())
}

// SimpleEventLogger 간단한 이벤트 로거 구현
type SimpleEventLogger struct{}

// LogEvent 이벤트 로깅
func (l *SimpleEventLogger) LogEvent(event *RBACEvent) error {
	fmt.Printf("[RBAC Event] %s: %s (User: %s, Target: %s)\n", 
		event.Timestamp.Format("2006-01-02 15:04:05"), 
		event.Type, 
		event.UserID, 
		event.TargetID)
	return nil
}

// SimpleAuditLogger 간단한 감사 로거 구현
type SimpleAuditLogger struct{}

// LogAuditEvent 감사 이벤트 로깅
func (l *SimpleAuditLogger) LogAuditEvent(event *RBACEvent) error {
	fmt.Printf("[RBAC Audit] %s: %s by user %s on %s %s\n",
		event.Timestamp.Format("2006-01-02 15:04:05"),
		event.Type,
		event.UserID,
		event.TargetType,
		event.TargetID)
	return nil
}

// SimpleMetricsCollector 간단한 메트릭 수집기 구현
type SimpleMetricsCollector struct{}

// IncrementCounter 카운터 증가
func (c *SimpleMetricsCollector) IncrementCounter(name string, tags map[string]string) {
	fmt.Printf("[Metrics] Counter %s incremented with tags: %v\n", name, tags)
}

// RecordGauge 게이지 기록
func (c *SimpleMetricsCollector) RecordGauge(name string, value float64, tags map[string]string) {
	fmt.Printf("[Metrics] Gauge %s recorded value %.2f with tags: %v\n", name, value, tags)
}

// RecordHistogram 히스토그램 기록
func (c *SimpleMetricsCollector) RecordHistogram(name string, value float64, tags map[string]string) {
	fmt.Printf("[Metrics] Histogram %s recorded value %.2f with tags: %v\n", name, value, tags)
}

// SimpleEventNotifier 간단한 이벤트 알리미 구현
type SimpleEventNotifier struct{}

// SendNotification 알림 전송
func (n *SimpleEventNotifier) SendNotification(event *RBACEvent) error {
	fmt.Printf("[Notification] Important RBAC event: %s (User: %s, Target: %s)\n",
		event.Type, event.UserID, event.TargetID)
	return nil
}