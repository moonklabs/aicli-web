package claude

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// TextMessageHandler는 텍스트 메시지를 처리하는 핸들러입니다.
type TextMessageHandler struct {
	BaseMessageHandler
	outputHandler func(string) error
}

// NewTextMessageHandler는 새로운 텍스트 메시지 핸들러를 생성합니다.
func NewTextMessageHandler(outputHandler func(string) error, logger *logrus.Logger) *TextMessageHandler {
	return &TextMessageHandler{
		BaseMessageHandler: BaseMessageHandler{
			name:     "text_handler",
			priority: 100,
			logger:   logger,
		},
		outputHandler: outputHandler,
	}
}

// Handle은 텍스트 메시지를 처리합니다.
func (h *TextMessageHandler) Handle(ctx context.Context, msg StreamMessage) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	h.logger.WithFields(logrus.Fields{
		"message_id": msg.ID,
		"content_len": len(msg.Content),
	}).Debug("Handling text message")

	if h.outputHandler != nil {
		return h.outputHandler(msg.Content)
	}

	return nil
}

// ToolUseHandler는 도구 사용 메시지를 처리하는 핸들러입니다.
type ToolUseHandler struct {
	BaseMessageHandler
	toolExecutor func(toolName string, params map[string]interface{}) (interface{}, error)
}

// NewToolUseHandler는 새로운 도구 사용 핸들러를 생성합니다.
func NewToolUseHandler(toolExecutor func(string, map[string]interface{}) (interface{}, error), logger *logrus.Logger) *ToolUseHandler {
	return &ToolUseHandler{
		BaseMessageHandler: BaseMessageHandler{
			name:     "tool_use_handler",
			priority: 90,
			logger:   logger,
		},
		toolExecutor: toolExecutor,
	}
}

// Handle은 도구 사용 메시지를 처리합니다.
func (h *ToolUseHandler) Handle(ctx context.Context, msg StreamMessage) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// 메타데이터에서 도구 정보 추출
	toolName, ok := msg.Meta["tool"].(string)
	if !ok {
		return fmt.Errorf("tool name not found in message metadata")
	}

	params, ok := msg.Meta["params"].(map[string]interface{})
	if !ok {
		params = make(map[string]interface{})
	}

	h.logger.WithFields(logrus.Fields{
		"message_id": msg.ID,
		"tool":       toolName,
		"params":     params,
	}).Debug("Handling tool use message")

	if h.toolExecutor != nil {
		result, err := h.toolExecutor(toolName, params)
		if err != nil {
			return fmt.Errorf("tool execution failed: %w", err)
		}

		h.logger.WithField("result", result).Debug("Tool execution completed")
	}

	return nil
}

// ErrorMessageHandler는 에러 메시지를 처리하는 핸들러입니다.
type ErrorMessageHandler struct {
	BaseMessageHandler
	errorReporter func(error, map[string]interface{})
}

// NewErrorMessageHandler는 새로운 에러 메시지 핸들러를 생성합니다.
func NewErrorMessageHandler(errorReporter func(error, map[string]interface{}), logger *logrus.Logger) *ErrorMessageHandler {
	return &ErrorMessageHandler{
		BaseMessageHandler: BaseMessageHandler{
			name:     "error_handler",
			priority: 110, // 에러는 높은 우선순위
			logger:   logger,
		},
		errorReporter: errorReporter,
	}
}

// Handle은 에러 메시지를 처리합니다.
func (h *ErrorMessageHandler) Handle(ctx context.Context, msg StreamMessage) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// 에러 정보 추출
	errorType, _ := msg.Meta["error_type"].(string)
	errorCode, _ := msg.Meta["error_code"].(int)
	
	err := fmt.Errorf("claude error [%s:%d]: %s", errorType, errorCode, msg.Content)
	
	h.logger.WithFields(logrus.Fields{
		"message_id":  msg.ID,
		"error_type":  errorType,
		"error_code":  errorCode,
		"error_msg":   msg.Content,
	}).Error("Handling error message")

	if h.errorReporter != nil {
		h.errorReporter(err, msg.Meta)
	}

	return err
}

// SystemMessageHandler는 시스템 메시지를 처리하는 핸들러입니다.
type SystemMessageHandler struct {
	BaseMessageHandler
	systemEventHandler func(event string, data map[string]interface{})
}

// NewSystemMessageHandler는 새로운 시스템 메시지 핸들러를 생성합니다.
func NewSystemMessageHandler(systemEventHandler func(string, map[string]interface{}), logger *logrus.Logger) *SystemMessageHandler {
	return &SystemMessageHandler{
		BaseMessageHandler: BaseMessageHandler{
			name:     "system_handler",
			priority: 80,
			logger:   logger,
		},
		systemEventHandler: systemEventHandler,
	}
}

// Handle은 시스템 메시지를 처리합니다.
func (h *SystemMessageHandler) Handle(ctx context.Context, msg StreamMessage) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	event, _ := msg.Meta["event"].(string)
	
	h.logger.WithFields(logrus.Fields{
		"message_id": msg.ID,
		"event":      event,
		"data":       msg.Meta,
	}).Debug("Handling system message")

	if h.systemEventHandler != nil {
		h.systemEventHandler(event, msg.Meta)
	}

	return nil
}

// MetadataHandler는 메타데이터 메시지를 처리하는 핸들러입니다.
type MetadataHandler struct {
	BaseMessageHandler
	metadataStore map[string]interface{}
	mu            sync.RWMutex
}

// NewMetadataHandler는 새로운 메타데이터 핸들러를 생성합니다.
func NewMetadataHandler(logger *logrus.Logger) *MetadataHandler {
	return &MetadataHandler{
		BaseMessageHandler: BaseMessageHandler{
			name:     "metadata_handler",
			priority: 70,
			logger:   logger,
		},
		metadataStore: make(map[string]interface{}),
	}
}

// Handle은 메타데이터 메시지를 처리합니다.
func (h *MetadataHandler) Handle(ctx context.Context, msg StreamMessage) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	// 메타데이터 저장 또는 업데이트
	for key, value := range msg.Meta {
		h.metadataStore[key] = value
	}

	h.logger.WithFields(logrus.Fields{
		"message_id": msg.ID,
		"keys":       len(msg.Meta),
	}).Debug("Handling metadata message")

	return nil
}

// GetMetadata는 저장된 메타데이터를 반환합니다.
func (h *MetadataHandler) GetMetadata(key string) (interface{}, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	value, exists := h.metadataStore[key]
	return value, exists
}

// GetAllMetadata는 모든 메타데이터를 반환합니다.
func (h *MetadataHandler) GetAllMetadata() map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	result := make(map[string]interface{})
	for k, v := range h.metadataStore {
		result[k] = v
	}
	return result
}

// StatusMessageHandler는 상태 메시지를 처리하는 핸들러입니다.
type StatusMessageHandler struct {
	BaseMessageHandler
	statusCallback func(status string, progress float64)
}

// NewStatusMessageHandler는 새로운 상태 메시지 핸들러를 생성합니다.
func NewStatusMessageHandler(statusCallback func(string, float64), logger *logrus.Logger) *StatusMessageHandler {
	return &StatusMessageHandler{
		BaseMessageHandler: BaseMessageHandler{
			name:     "status_handler",
			priority: 75,
			logger:   logger,
		},
		statusCallback: statusCallback,
	}
}

// Handle은 상태 메시지를 처리합니다.
func (h *StatusMessageHandler) Handle(ctx context.Context, msg StreamMessage) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	status := msg.Content
	progress, _ := msg.Meta["progress"].(float64)

	h.logger.WithFields(logrus.Fields{
		"message_id": msg.ID,
		"status":     status,
		"progress":   progress,
	}).Debug("Handling status message")

	if h.statusCallback != nil {
		h.statusCallback(status, progress)
	}

	return nil
}

// ProgressMessageHandler는 진행률 메시지를 처리하는 핸들러입니다.
type ProgressMessageHandler struct {
	BaseMessageHandler
	progressTracker *ProgressTracker
}

// ProgressTracker는 진행률을 추적하는 구조체입니다.
type ProgressTracker struct {
	tasks      map[string]*TaskProgress
	mu         sync.RWMutex
	logger     *logrus.Logger
	onUpdate   func(taskID string, progress *TaskProgress)
}

// TaskProgress는 태스크 진행률 정보입니다.
type TaskProgress struct {
	TaskID      string
	Description string
	Current     int64
	Total       int64
	Percentage  float64
	StartTime   time.Time
	UpdateTime  time.Time
	Status      string
}

// NewProgressMessageHandler는 새로운 진행률 메시지 핸들러를 생성합니다.
func NewProgressMessageHandler(onUpdate func(string, *TaskProgress), logger *logrus.Logger) *ProgressMessageHandler {
	return &ProgressMessageHandler{
		BaseMessageHandler: BaseMessageHandler{
			name:     "progress_handler",
			priority: 85,
			logger:   logger,
		},
		progressTracker: &ProgressTracker{
			tasks:    make(map[string]*TaskProgress),
			logger:   logger,
			onUpdate: onUpdate,
		},
	}
}

// Handle은 진행률 메시지를 처리합니다.
func (h *ProgressMessageHandler) Handle(ctx context.Context, msg StreamMessage) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	taskID, _ := msg.Meta["task_id"].(string)
	if taskID == "" {
		taskID = msg.ID
	}

	current, _ := msg.Meta["current"].(float64)
	total, _ := msg.Meta["total"].(float64)
	status, _ := msg.Meta["status"].(string)

	progress := h.progressTracker.UpdateProgress(taskID, msg.Content, int64(current), int64(total), status)

	h.logger.WithFields(logrus.Fields{
		"message_id": msg.ID,
		"task_id":    taskID,
		"percentage": progress.Percentage,
		"status":     status,
	}).Debug("Handling progress message")

	return nil
}

// UpdateProgress는 진행률을 업데이트합니다.
func (pt *ProgressTracker) UpdateProgress(taskID, description string, current, total int64, status string) *TaskProgress {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	progress, exists := pt.tasks[taskID]
	if !exists {
		progress = &TaskProgress{
			TaskID:      taskID,
			StartTime:   time.Now(),
			Description: description,
		}
		pt.tasks[taskID] = progress
	}

	progress.Current = current
	progress.Total = total
	progress.Status = status
	progress.UpdateTime = time.Now()
	
	if total > 0 {
		progress.Percentage = float64(current) / float64(total) * 100
	}

	if pt.onUpdate != nil {
		pt.onUpdate(taskID, progress)
	}

	return progress
}

// GetProgress는 특정 태스크의 진행률을 반환합니다.
func (pt *ProgressTracker) GetProgress(taskID string) (*TaskProgress, bool) {
	pt.mu.RLock()
	defer pt.mu.RUnlock()
	progress, exists := pt.tasks[taskID]
	return progress, exists
}

// CompleteMessageHandler는 완료 메시지를 처리하는 핸들러입니다.
type CompleteMessageHandler struct {
	BaseMessageHandler
	onComplete func(result map[string]interface{})
}

// NewCompleteMessageHandler는 새로운 완료 메시지 핸들러를 생성합니다.
func NewCompleteMessageHandler(onComplete func(map[string]interface{}), logger *logrus.Logger) *CompleteMessageHandler {
	return &CompleteMessageHandler{
		BaseMessageHandler: BaseMessageHandler{
			name:     "complete_handler",
			priority: 95,
			logger:   logger,
		},
		onComplete: onComplete,
	}
}

// Handle은 완료 메시지를 처리합니다.
func (h *CompleteMessageHandler) Handle(ctx context.Context, msg StreamMessage) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	result := make(map[string]interface{})
	result["message_id"] = msg.ID
	result["content"] = msg.Content
	result["timestamp"] = time.Now()
	
	// 메타데이터 병합
	for k, v := range msg.Meta {
		result[k] = v
	}

	h.logger.WithFields(logrus.Fields{
		"message_id": msg.ID,
		"result":     result,
	}).Info("Handling complete message")

	if h.onComplete != nil {
		h.onComplete(result)
	}

	return nil
}

// ChainHandler는 여러 핸들러를 체인으로 연결하는 핸들러입니다.
type ChainHandler struct {
	BaseMessageHandler
	handlers []MessageHandler
}

// NewChainHandler는 새로운 체인 핸들러를 생성합니다.
func NewChainHandler(name string, priority int, handlers []MessageHandler, logger *logrus.Logger) *ChainHandler {
	return &ChainHandler{
		BaseMessageHandler: BaseMessageHandler{
			name:     name,
			priority: priority,
			logger:   logger,
		},
		handlers: handlers,
	}
}

// Handle은 체인의 모든 핸들러를 순차적으로 실행합니다.
func (h *ChainHandler) Handle(ctx context.Context, msg StreamMessage) error {
	for _, handler := range h.handlers {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := handler.Handle(ctx, msg); err != nil {
				h.logger.WithError(err).WithField("handler", handler.Name()).Error("Chain handler failed")
				return err
			}
		}
	}
	return nil
}