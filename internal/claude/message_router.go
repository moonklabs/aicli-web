package claude

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

// MessageType은 메시지 타입을 정의합니다.
type MessageType string

const (
	// 메시지 타입 상수
	MessageTypeText     MessageType = "text"
	MessageTypeToolUse  MessageType = "tool_use"
	MessageTypeError    MessageType = "error"
	MessageTypeSystem   MessageType = "system"
	MessageTypeMetadata MessageType = "metadata"
	MessageTypeStatus   MessageType = "status"
	MessageTypeProgress MessageType = "progress"
	MessageTypeComplete MessageType = "complete"
)

// MessageHandler는 메시지를 처리하는 핸들러 인터페이스입니다.
type MessageHandler interface {
	Handle(ctx context.Context, msg StreamMessage) error
	Priority() int
	Name() string
}

// BaseMessageHandler는 메시지 핸들러의 기본 구현체입니다.
type BaseMessageHandler struct {
	name     string
	priority int
	logger   *logrus.Logger
}

// Priority는 핸들러의 우선순위를 반환합니다.
func (h *BaseMessageHandler) Priority() int {
	return h.priority
}

// Name은 핸들러의 이름을 반환합니다.
func (h *BaseMessageHandler) Name() string {
	return h.name
}

// MessageRouter는 메시지 타입별로 핸들러를 라우팅하는 구조체입니다.
type MessageRouter struct {
	handlers       map[MessageType][]MessageHandler
	defaultHandler MessageHandler
	mu             sync.RWMutex
	logger         *logrus.Logger
	metrics        *RouterMetrics
	errorHandler   func(error, StreamMessage)
	
	// 비동기 처리를 위한 워커 풀
	workerPool     *WorkerPool
	asyncMode      bool
	maxConcurrency int
}

// RouterMetrics는 라우터 메트릭을 추적합니다.
type RouterMetrics struct {
	MessagesRouted   map[MessageType]int64
	HandleErrors     map[MessageType]int64
	AvgHandleTime    map[MessageType]time.Duration
	TotalMessages    int64
	TotalErrors      int64
	mu               sync.RWMutex
}

// RouterConfig는 메시지 라우터 설정을 정의합니다.
type RouterConfig struct {
	AsyncMode      bool
	MaxConcurrency int
	ErrorHandler   func(error, StreamMessage)
}

// NewMessageRouter는 새로운 메시지 라우터를 생성합니다.
func NewMessageRouter(config RouterConfig, logger *logrus.Logger) *MessageRouter {
	if config.MaxConcurrency <= 0 {
		config.MaxConcurrency = 10
	}

	router := &MessageRouter{
		handlers:       make(map[MessageType][]MessageHandler),
		logger:         logger,
		metrics:        &RouterMetrics{
			MessagesRouted: make(map[MessageType]int64),
			HandleErrors:   make(map[MessageType]int64),
			AvgHandleTime:  make(map[MessageType]time.Duration),
		},
		errorHandler:   config.ErrorHandler,
		asyncMode:      config.AsyncMode,
		maxConcurrency: config.MaxConcurrency,
	}

	if config.AsyncMode {
		router.workerPool = NewWorkerPool(config.MaxConcurrency, logger)
		router.workerPool.Start()
	}

	return router
}

// RegisterHandler는 특정 메시지 타입에 대한 핸들러를 등록합니다.
func (r *MessageRouter) RegisterHandler(msgType MessageType, handler MessageHandler) error {
	if handler == nil {
		return fmt.Errorf("handler cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.handlers[msgType] = append(r.handlers[msgType], handler)
	
	// 우선순위 순으로 정렬 (높은 우선순위가 먼저)
	sort.Slice(r.handlers[msgType], func(i, j int) bool {
		return r.handlers[msgType][i].Priority() > r.handlers[msgType][j].Priority()
	})

	r.logger.WithFields(logrus.Fields{
		"message_type": msgType,
		"handler":      handler.Name(),
		"priority":     handler.Priority(),
	}).Debug("Handler registered")

	return nil
}

// SetDefaultHandler는 기본 핸들러를 설정합니다.
func (r *MessageRouter) SetDefaultHandler(handler MessageHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.defaultHandler = handler
}

// Route는 메시지를 적절한 핸들러로 라우팅합니다.
func (r *MessageRouter) Route(ctx context.Context, msg StreamMessage) error {
	msgType := MessageType(msg.Type)
	
	// 메트릭 업데이트
	atomic.AddInt64(&r.metrics.TotalMessages, 1)
	r.incrementMessageCount(msgType)

	r.mu.RLock()
	handlers := r.handlers[msgType]
	defaultHandler := r.defaultHandler
	r.mu.RUnlock()

	// 핸들러가 없으면 기본 핸들러 사용
	if len(handlers) == 0 && defaultHandler != nil {
		handlers = []MessageHandler{defaultHandler}
	}

	if len(handlers) == 0 {
		r.logger.WithField("message_type", msgType).Warn("No handlers registered for message type")
		return fmt.Errorf("no handlers registered for message type: %s", msgType)
	}

	// 비동기 모드
	if r.asyncMode {
		return r.routeAsync(ctx, msg, handlers, msgType)
	}

	// 동기 모드
	return r.routeSync(ctx, msg, handlers, msgType)
}

// routeSync는 메시지를 동기적으로 처리합니다.
func (r *MessageRouter) routeSync(ctx context.Context, msg StreamMessage, handlers []MessageHandler, msgType MessageType) error {
	start := time.Now()
	var lastErr error

	for _, handler := range handlers {
		if err := r.executeHandler(ctx, handler, msg, msgType); err != nil {
			lastErr = err
			r.logger.WithError(err).WithFields(logrus.Fields{
				"handler":      handler.Name(),
				"message_type": msgType,
			}).Error("Handler failed")
			
			// 에러 핸들러 호출
			if r.errorHandler != nil {
				r.errorHandler(err, msg)
			}
		}
	}

	r.updateHandleTime(msgType, time.Since(start))
	return lastErr
}

// routeAsync는 메시지를 비동기적으로 처리합니다.
func (r *MessageRouter) routeAsync(ctx context.Context, msg StreamMessage, handlers []MessageHandler, msgType MessageType) error {
	if r.workerPool == nil {
		return fmt.Errorf("worker pool not initialized")
	}

	task := &RouterTask{
		ctx:      ctx,
		msg:      msg,
		handlers: handlers,
		msgType:  msgType,
		router:   r,
	}

	return r.workerPool.Submit(task)
}

// executeHandler는 단일 핸들러를 실행합니다.
func (r *MessageRouter) executeHandler(ctx context.Context, handler MessageHandler, msg StreamMessage, msgType MessageType) error {
	// 타임아웃 설정
	handlerCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- handler.Handle(handlerCtx, msg)
	}()

	select {
	case err := <-done:
		if err != nil {
			r.incrementErrorCount(msgType)
			return err
		}
		return nil
	case <-handlerCtx.Done():
		r.incrementErrorCount(msgType)
		return fmt.Errorf("handler timeout: %s", handler.Name())
	}
}

// UnregisterHandler는 특정 핸들러를 등록 해제합니다.
func (r *MessageRouter) UnregisterHandler(msgType MessageType, handlerName string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	handlers, exists := r.handlers[msgType]
	if !exists {
		return fmt.Errorf("no handlers registered for message type: %s", msgType)
	}

	newHandlers := make([]MessageHandler, 0, len(handlers))
	removed := false

	for _, h := range handlers {
		if h.Name() != handlerName {
			newHandlers = append(newHandlers, h)
		} else {
			removed = true
		}
	}

	if !removed {
		return fmt.Errorf("handler not found: %s", handlerName)
	}

	r.handlers[msgType] = newHandlers
	return nil
}

// GetHandlers는 특정 메시지 타입의 핸들러 목록을 반환합니다.
func (r *MessageRouter) GetHandlers(msgType MessageType) []MessageHandler {
	r.mu.RLock()
	defer r.mu.RUnlock()

	handlers := make([]MessageHandler, len(r.handlers[msgType]))
	copy(handlers, r.handlers[msgType])
	return handlers
}

// incrementMessageCount는 메시지 카운트를 증가시킵니다.
func (r *MessageRouter) incrementMessageCount(msgType MessageType) {
	r.metrics.mu.Lock()
	defer r.metrics.mu.Unlock()
	
	if r.metrics.MessagesRouted == nil {
		r.metrics.MessagesRouted = make(map[MessageType]int64)
	}
	r.metrics.MessagesRouted[msgType]++
}

// incrementErrorCount는 에러 카운트를 증가시킵니다.
func (r *MessageRouter) incrementErrorCount(msgType MessageType) {
	r.metrics.mu.Lock()
	defer r.metrics.mu.Unlock()
	
	if r.metrics.HandleErrors == nil {
		r.metrics.HandleErrors = make(map[MessageType]int64)
	}
	r.metrics.HandleErrors[msgType]++
	atomic.AddInt64(&r.metrics.TotalErrors, 1)
}

// updateHandleTime는 처리 시간을 업데이트합니다.
func (r *MessageRouter) updateHandleTime(msgType MessageType, duration time.Duration) {
	r.metrics.mu.Lock()
	defer r.metrics.mu.Unlock()
	
	if r.metrics.AvgHandleTime == nil {
		r.metrics.AvgHandleTime = make(map[MessageType]time.Duration)
	}
	
	// 이동 평균 계산
	alpha := 0.1
	current := r.metrics.AvgHandleTime[msgType]
	r.metrics.AvgHandleTime[msgType] = time.Duration(
		alpha*float64(duration) + (1-alpha)*float64(current),
	)
}

// GetMetrics는 라우터 메트릭을 반환합니다.
func (r *MessageRouter) GetMetrics() map[string]interface{} {
	r.metrics.mu.RLock()
	defer r.metrics.mu.RUnlock()

	metrics := make(map[string]interface{})
	metrics["total_messages"] = atomic.LoadInt64(&r.metrics.TotalMessages)
	metrics["total_errors"] = atomic.LoadInt64(&r.metrics.TotalErrors)
	
	// 메시지 타입별 통계
	messageStats := make(map[string]map[string]interface{})
	for msgType, count := range r.metrics.MessagesRouted {
		stats := make(map[string]interface{})
		stats["count"] = count
		stats["errors"] = r.metrics.HandleErrors[msgType]
		stats["avg_handle_time_ms"] = r.metrics.AvgHandleTime[msgType].Milliseconds()
		messageStats[string(msgType)] = stats
	}
	metrics["message_stats"] = messageStats

	// 워커 풀 통계 (비동기 모드인 경우)
	if r.workerPool != nil {
		metrics["worker_pool"] = r.workerPool.GetStats()
	}

	return metrics
}

// Stop은 라우터를 정지합니다.
func (r *MessageRouter) Stop() {
	if r.workerPool != nil {
		r.workerPool.Stop()
	}
}

// RouterTask는 비동기 처리를 위한 태스크입니다.
type RouterTask struct {
	ctx      context.Context
	msg      StreamMessage
	handlers []MessageHandler
	msgType  MessageType
	router   *MessageRouter
}

// Execute는 라우터 태스크를 실행합니다.
func (t *RouterTask) Execute() error {
	start := time.Now()
	var lastErr error

	for _, handler := range t.handlers {
		if err := t.router.executeHandler(t.ctx, handler, t.msg, t.msgType); err != nil {
			lastErr = err
			t.router.logger.WithError(err).WithFields(logrus.Fields{
				"handler":      handler.Name(),
				"message_type": t.msgType,
			}).Error("Handler failed in async mode")
			
			// 에러 핸들러 호출
			if t.router.errorHandler != nil {
				t.router.errorHandler(err, t.msg)
			}
		}
	}

	t.router.updateHandleTime(t.msgType, time.Since(start))
	return lastErr
}

// WorkerPool은 비동기 태스크 처리를 위한 워커 풀입니다.
type WorkerPool struct {
	workers    int
	taskQueue  chan MessageRouterTask
	wg         sync.WaitGroup
	stopCh     chan struct{}
	logger     *logrus.Logger
	stats      *WorkerPoolStats
}

// MessageRouterTask는 워커 풀에서 실행할 태스크 인터페이스입니다.
type MessageRouterTask interface {
	Execute() error
}

// WorkerPoolStats는 워커 풀 통계를 추적합니다.
type WorkerPoolStats struct {
	TasksSubmitted int64
	TasksCompleted int64
	TasksFailed    int64
	QueueSize      int64
}

// NewWorkerPool은 새로운 워커 풀을 생성합니다.
func NewWorkerPool(workers int, logger *logrus.Logger) *WorkerPool {
	return &WorkerPool{
		workers:   workers,
		taskQueue: make(chan MessageRouterTask, workers*10),
		stopCh:    make(chan struct{}),
		logger:    logger,
		stats:     &WorkerPoolStats{},
	}
}

// Start는 워커 풀을 시작합니다.
func (wp *WorkerPool) Start() {
	for i := 0; i < wp.workers; i++ {
		wp.wg.Add(1)
		go wp.worker(i)
	}
	wp.logger.WithField("workers", wp.workers).Info("Worker pool started")
}

// worker는 워커 고루틴입니다.
func (wp *WorkerPool) worker(id int) {
	defer wp.wg.Done()

	for {
		select {
		case task := <-wp.taskQueue:
			if task != nil {
				if err := task.Execute(); err != nil {
					atomic.AddInt64(&wp.stats.TasksFailed, 1)
				} else {
					atomic.AddInt64(&wp.stats.TasksCompleted, 1)
				}
				atomic.AddInt64(&wp.stats.QueueSize, -1)
			}
		case <-wp.stopCh:
			return
		}
	}
}

// Submit은 태스크를 워커 풀에 제출합니다.
func (wp *WorkerPool) Submit(task MessageRouterTask) error {
	select {
	case wp.taskQueue <- task:
		atomic.AddInt64(&wp.stats.TasksSubmitted, 1)
		atomic.AddInt64(&wp.stats.QueueSize, 1)
		return nil
	default:
		return fmt.Errorf("worker pool queue is full")
	}
}

// Stop은 워커 풀을 정지합니다.
func (wp *WorkerPool) Stop() {
	close(wp.stopCh)
	wp.wg.Wait()
	close(wp.taskQueue)
	wp.logger.Info("Worker pool stopped")
}

// GetStats는 워커 풀 통계를 반환합니다.
func (wp *WorkerPool) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"tasks_submitted": atomic.LoadInt64(&wp.stats.TasksSubmitted),
		"tasks_completed": atomic.LoadInt64(&wp.stats.TasksCompleted),
		"tasks_failed":    atomic.LoadInt64(&wp.stats.TasksFailed),
		"queue_size":      atomic.LoadInt64(&wp.stats.QueueSize),
		"workers":         wp.workers,
	}
}