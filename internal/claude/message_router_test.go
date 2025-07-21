package claude

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockHandler는 테스트용 메시지 핸들러입니다.
type mockHandler struct {
	name      string
	priority  int
	handled   int32
	failAfter int
	delay     time.Duration
	mu        sync.Mutex
	messages  []Message
}

func newMockHandler(name string, priority int) *mockHandler {
	return &mockHandler{
		name:     name,
		priority: priority,
		messages: make([]Message, 0),
	}
}

func (h *mockHandler) Handle(ctx context.Context, msg Message) error {
	atomic.AddInt32(&h.handled, 1)
	
	h.mu.Lock()
	h.messages = append(h.messages, msg)
	h.mu.Unlock()

	if h.delay > 0 {
		time.Sleep(h.delay)
	}

	if h.failAfter > 0 && int(atomic.LoadInt32(&h.handled)) > h.failAfter {
		return fmt.Errorf("handler %s failed after %d messages", h.name, h.failAfter)
	}

	return nil
}

func (h *mockHandler) Priority() int {
	return h.priority
}

func (h *mockHandler) Name() string {
	return h.name
}

func (h *mockHandler) GetHandledCount() int32 {
	return atomic.LoadInt32(&h.handled)
}

func (h *mockHandler) GetMessages() []Message {
	h.mu.Lock()
	defer h.mu.Unlock()
	result := make([]Message, len(h.messages))
	copy(result, h.messages)
	return result
}

func TestMessageRouter(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	t.Run("Basic Routing", func(t *testing.T) {
		config := RouterConfig{
			AsyncMode: false,
		}
		router := NewMessageRouter(config, logger)

		handler := newMockHandler("test", 100)
		err := router.RegisterHandler(MessageTypeText, handler)
		require.NoError(t, err)

		ctx := context.Background()
		msg := Message{
			Type:    string(MessageTypeText),
			Content: "Hello, World!",
			ID:      "test-123",
		}

		err = router.Route(ctx, msg)
		assert.NoError(t, err)
		assert.Equal(t, int32(1), handler.GetHandledCount())
	})

	t.Run("Multiple Handlers Priority", func(t *testing.T) {
		config := RouterConfig{
			AsyncMode: false,
		}
		router := NewMessageRouter(config, logger)

		handlers := []*mockHandler{
			newMockHandler("low", 10),
			newMockHandler("high", 100),
			newMockHandler("medium", 50),
		}

		for _, h := range handlers {
			router.RegisterHandler(MessageTypeText, h)
		}

		ctx := context.Background()
		msg := Message{
			Type:    string(MessageTypeText),
			Content: "Priority test",
			ID:      "priority-123",
		}

		err := router.Route(ctx, msg)
		assert.NoError(t, err)

		// 모든 핸들러가 호출되어야 함
		for _, h := range handlers {
			assert.Equal(t, int32(1), h.GetHandledCount())
		}
	})

	t.Run("Default Handler", func(t *testing.T) {
		config := RouterConfig{
			AsyncMode: false,
		}
		router := NewMessageRouter(config, logger)

		defaultHandler := newMockHandler("default", 0)
		router.SetDefaultHandler(defaultHandler)

		ctx := context.Background()
		msg := Message{
			Type:    "unknown_type",
			Content: "Unknown message",
			ID:      "unknown-123",
		}

		err := router.Route(ctx, msg)
		assert.NoError(t, err)
		assert.Equal(t, int32(1), defaultHandler.GetHandledCount())
	})

	t.Run("Async Mode", func(t *testing.T) {
		config := RouterConfig{
			AsyncMode:      true,
			MaxConcurrency: 5,
		}
		router := NewMessageRouter(config, logger)
		defer router.Stop()

		handler := newMockHandler("async", 100)
		handler.delay = 10 * time.Millisecond
		router.RegisterHandler(MessageTypeText, handler)

		ctx := context.Background()
		numMessages := 10

		start := time.Now()
		for i := 0; i < numMessages; i++ {
			msg := Message{
				Type:    string(MessageTypeText),
				Content: fmt.Sprintf("Async message %d", i),
				ID:      fmt.Sprintf("async-%d", i),
			}
			err := router.Route(ctx, msg)
			assert.NoError(t, err)
		}

		// 비동기 모드에서는 즉시 반환
		elapsed := time.Since(start)
		assert.Less(t, elapsed, time.Duration(numMessages)*handler.delay)

		// 모든 메시지가 처리될 때까지 대기
		time.Sleep(200 * time.Millisecond)
		assert.Equal(t, int32(numMessages), handler.GetHandledCount())
	})

	t.Run("Error Handling", func(t *testing.T) {
		var errorCalled bool
		var capturedError error
		var capturedMessage Message

		config := RouterConfig{
			AsyncMode: false,
			ErrorHandler: func(err error, msg Message) {
				errorCalled = true
				capturedError = err
				capturedMessage = msg
			},
		}
		router := NewMessageRouter(config, logger)

		handler := newMockHandler("error", 100)
		handler.failAfter = 0 // 항상 실패
		router.RegisterHandler(MessageTypeError, handler)

		ctx := context.Background()
		msg := Message{
			Type:    string(MessageTypeError),
			Content: "Error message",
			ID:      "error-123",
		}

		err := router.Route(ctx, msg)
		assert.Error(t, err)
		assert.True(t, errorCalled)
		assert.NotNil(t, capturedError)
		assert.Equal(t, msg.ID, capturedMessage.ID)
	})

	t.Run("Unregister Handler", func(t *testing.T) {
		config := RouterConfig{
			AsyncMode: false,
		}
		router := NewMessageRouter(config, logger)

		handler1 := newMockHandler("handler1", 100)
		handler2 := newMockHandler("handler2", 50)

		router.RegisterHandler(MessageTypeText, handler1)
		router.RegisterHandler(MessageTypeText, handler2)

		// handler1 제거
		err := router.UnregisterHandler(MessageTypeText, "handler1")
		assert.NoError(t, err)

		ctx := context.Background()
		msg := Message{
			Type:    string(MessageTypeText),
			Content: "After unregister",
			ID:      "unregister-123",
		}

		err = router.Route(ctx, msg)
		assert.NoError(t, err)

		// handler1은 호출되지 않고 handler2만 호출
		assert.Equal(t, int32(0), handler1.GetHandledCount())
		assert.Equal(t, int32(1), handler2.GetHandledCount())
	})

	t.Run("Handler Timeout", func(t *testing.T) {
		config := RouterConfig{
			AsyncMode: false,
		}
		router := NewMessageRouter(config, logger)

		handler := newMockHandler("slow", 100)
		handler.delay = 10 * time.Second // 타임아웃보다 긴 지연
		router.RegisterHandler(MessageTypeText, handler)

		ctx := context.Background()
		msg := Message{
			Type:    string(MessageTypeText),
			Content: "Timeout test",
			ID:      "timeout-123",
		}

		start := time.Now()
		err := router.Route(ctx, msg)
		elapsed := time.Since(start)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "timeout")
		assert.Less(t, elapsed, 10*time.Second)
		assert.Greater(t, elapsed, 4*time.Second)
	})

	t.Run("Concurrent Routing", func(t *testing.T) {
		config := RouterConfig{
			AsyncMode:      true,
			MaxConcurrency: 10,
		}
		router := NewMessageRouter(config, logger)
		defer router.Stop()

		handler := newMockHandler("concurrent", 100)
		router.RegisterHandler(MessageTypeText, handler)

		ctx := context.Background()
		numGoroutines := 10
		messagesPerGoroutine := 100

		var wg sync.WaitGroup
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < messagesPerGoroutine; j++ {
					msg := Message{
						Type:    string(MessageTypeText),
						Content: fmt.Sprintf("Concurrent %d-%d", id, j),
						ID:      fmt.Sprintf("concurrent-%d-%d", id, j),
					}
					err := router.Route(ctx, msg)
					assert.NoError(t, err)
				}
			}(i)
		}

		wg.Wait()
		time.Sleep(100 * time.Millisecond) // 비동기 처리 대기

		expectedTotal := int32(numGoroutines * messagesPerGoroutine)
		assert.Equal(t, expectedTotal, handler.GetHandledCount())
	})

	t.Run("Metrics Collection", func(t *testing.T) {
		config := RouterConfig{
			AsyncMode: false,
		}
		router := NewMessageRouter(config, logger)

		handler := newMockHandler("metrics", 100)
		router.RegisterHandler(MessageTypeText, handler)
		
		errorHandler := newMockHandler("error", 100)
		errorHandler.failAfter = 2
		router.RegisterHandler(MessageTypeError, errorHandler)

		ctx := context.Background()

		// 성공 메시지
		for i := 0; i < 5; i++ {
			msg := Message{
				Type:    string(MessageTypeText),
				Content: fmt.Sprintf("Success %d", i),
				ID:      fmt.Sprintf("success-%d", i),
			}
			router.Route(ctx, msg)
		}

		// 에러 메시지
		for i := 0; i < 3; i++ {
			msg := Message{
				Type:    string(MessageTypeError),
				Content: fmt.Sprintf("Error %d", i),
				ID:      fmt.Sprintf("error-%d", i),
			}
			router.Route(ctx, msg)
		}

		metrics := router.GetMetrics()
		assert.Equal(t, int64(8), metrics["total_messages"])
		assert.Equal(t, int64(1), metrics["total_errors"]) // failAfter = 2

		messageStats := metrics["message_stats"].(map[string]map[string]interface{})
		textStats := messageStats[string(MessageTypeText)]
		assert.Equal(t, int64(5), textStats["count"])
		assert.Equal(t, int64(0), textStats["errors"])

		errorStats := messageStats[string(MessageTypeError)]
		assert.Equal(t, int64(3), errorStats["count"])
		assert.Equal(t, int64(1), errorStats["errors"])
	})
}

func TestWorkerPool(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	t.Run("Basic Task Execution", func(t *testing.T) {
		pool := NewWorkerPool(5, logger)
		pool.Start()
		defer pool.Stop()

		var counter int32
		numTasks := 50

		for i := 0; i < numTasks; i++ {
			task := &testTask{
				id: i,
				fn: func() error {
					atomic.AddInt32(&counter, 1)
					return nil
				},
			}
			err := pool.Submit(task)
			assert.NoError(t, err)
		}

		time.Sleep(100 * time.Millisecond)
		assert.Equal(t, int32(numTasks), atomic.LoadInt32(&counter))
	})

	t.Run("Queue Full", func(t *testing.T) {
		pool := NewWorkerPool(1, logger)
		pool.Start()
		defer pool.Stop()

		// 큐를 가득 채우기
		blockingTask := &testTask{
			fn: func() error {
				time.Sleep(1 * time.Second)
				return nil
			},
		}

		// 워커 수 * 10 만큼의 큐 크기
		for i := 0; i < 10; i++ {
			pool.Submit(blockingTask)
		}

		// 큐가 가득 찬 상태에서 제출
		err := pool.Submit(blockingTask)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "queue is full")
	})
}

type testTask struct {
	id int
	fn func() error
}

func (t *testTask) Execute() error {
	return t.fn()
}

func TestMessageTypeHandlers(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	t.Run("TextMessageHandler", func(t *testing.T) {
		var captured string
		handler := NewTextMessageHandler(func(text string) error {
			captured = text
			return nil
		}, logger)

		ctx := context.Background()
		msg := Message{
			Type:    string(MessageTypeText),
			Content: "Test text message",
			ID:      "text-123",
		}

		err := handler.Handle(ctx, msg)
		assert.NoError(t, err)
		assert.Equal(t, "Test text message", captured)
		assert.Equal(t, "text_handler", handler.Name())
		assert.Equal(t, 100, handler.Priority())
	})

	t.Run("ToolUseHandler", func(t *testing.T) {
		var capturedTool string
		var capturedParams map[string]interface{}

		handler := NewToolUseHandler(func(tool string, params map[string]interface{}) (interface{}, error) {
			capturedTool = tool
			capturedParams = params
			return "result", nil
		}, logger)

		ctx := context.Background()
		msg := Message{
			Type:    string(MessageTypeToolUse),
			Content: "Tool execution",
			ID:      "tool-123",
			Meta: map[string]interface{}{
				"tool": "calculator",
				"params": map[string]interface{}{
					"operation": "add",
					"a":         1,
					"b":         2,
				},
			},
		}

		err := handler.Handle(ctx, msg)
		assert.NoError(t, err)
		assert.Equal(t, "calculator", capturedTool)
		assert.Equal(t, "add", capturedParams["operation"])
	})

	t.Run("ErrorMessageHandler", func(t *testing.T) {
		var capturedError error
		var capturedMeta map[string]interface{}

		handler := NewErrorMessageHandler(func(err error, meta map[string]interface{}) {
			capturedError = err
			capturedMeta = meta
		}, logger)

		ctx := context.Background()
		msg := Message{
			Type:    string(MessageTypeError),
			Content: "Something went wrong",
			ID:      "error-123",
			Meta: map[string]interface{}{
				"error_type": "validation",
				"error_code": 400,
			},
		}

		err := handler.Handle(ctx, msg)
		assert.Error(t, err)
		assert.NotNil(t, capturedError)
		assert.Contains(t, capturedError.Error(), "validation")
		assert.Equal(t, 400, capturedMeta["error_code"])
	})

	t.Run("SystemMessageHandler", func(t *testing.T) {
		var capturedEvent string
		var capturedData map[string]interface{}

		handler := NewSystemMessageHandler(func(event string, data map[string]interface{}) {
			capturedEvent = event
			capturedData = data
		}, logger)

		ctx := context.Background()
		msg := Message{
			Type:    string(MessageTypeSystem),
			Content: "System event",
			ID:      "system-123",
			Meta: map[string]interface{}{
				"event": "startup",
				"version": "1.0.0",
			},
		}

		err := handler.Handle(ctx, msg)
		assert.NoError(t, err)
		assert.Equal(t, "startup", capturedEvent)
		assert.Equal(t, "1.0.0", capturedData["version"])
	})

	t.Run("ProgressMessageHandler", func(t *testing.T) {
		var capturedTaskID string
		var capturedProgress *TaskProgress

		handler := NewProgressMessageHandler(func(taskID string, progress *TaskProgress) {
			capturedTaskID = taskID
			capturedProgress = progress
		}, logger)

		ctx := context.Background()
		msg := Message{
			Type:    string(MessageTypeProgress),
			Content: "Processing files",
			ID:      "progress-123",
			Meta: map[string]interface{}{
				"task_id": "task-456",
				"current": float64(50),
				"total":   float64(100),
				"status":  "in_progress",
			},
		}

		err := handler.Handle(ctx, msg)
		assert.NoError(t, err)
		assert.Equal(t, "task-456", capturedTaskID)
		assert.NotNil(t, capturedProgress)
		assert.Equal(t, float64(50), capturedProgress.Percentage)
	})

	t.Run("CompleteMessageHandler", func(t *testing.T) {
		var capturedResult map[string]interface{}

		handler := NewCompleteMessageHandler(func(result map[string]interface{}) {
			capturedResult = result
		}, logger)

		ctx := context.Background()
		msg := Message{
			Type:    string(MessageTypeComplete),
			Content: "Task completed successfully",
			ID:      "complete-123",
			Meta: map[string]interface{}{
				"duration": "5s",
				"status":   "success",
			},
		}

		err := handler.Handle(ctx, msg)
		assert.NoError(t, err)
		assert.NotNil(t, capturedResult)
		assert.Equal(t, "5s", capturedResult["duration"])
		assert.Equal(t, "success", capturedResult["status"])
	})

	t.Run("ChainHandler", func(t *testing.T) {
		var order []string
		
		handler1 := newMockHandler("first", 100)
		handler1.delay = 10 * time.Millisecond
		
		handler2 := newMockHandler("second", 50)
		
		chainHandler := NewChainHandler("chain", 200, []MessageHandler{handler1, handler2}, logger)

		ctx := context.Background()
		msg := Message{
			Type:    string(MessageTypeText),
			Content: "Chain test",
			ID:      "chain-123",
		}

		err := chainHandler.Handle(ctx, msg)
		assert.NoError(t, err)
		assert.Equal(t, int32(1), handler1.GetHandledCount())
		assert.Equal(t, int32(1), handler2.GetHandledCount())
	})
}

func BenchmarkMessageRouter(b *testing.B) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	b.Run("SyncRouting", func(b *testing.B) {
		config := RouterConfig{
			AsyncMode: false,
		}
		router := NewMessageRouter(config, logger)
		
		handler := newMockHandler("bench", 100)
		router.RegisterHandler(MessageTypeText, handler)

		ctx := context.Background()
		msg := Message{
			Type:    string(MessageTypeText),
			Content: "Benchmark message",
			ID:      "bench-123",
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			router.Route(ctx, msg)
		}
	})

	b.Run("AsyncRouting", func(b *testing.B) {
		config := RouterConfig{
			AsyncMode:      true,
			MaxConcurrency: 10,
		}
		router := NewMessageRouter(config, logger)
		defer router.Stop()
		
		handler := newMockHandler("bench", 100)
		router.RegisterHandler(MessageTypeText, handler)

		ctx := context.Background()
		msg := Message{
			Type:    string(MessageTypeText),
			Content: "Benchmark message",
			ID:      "bench-123",
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			router.Route(ctx, msg)
		}
	})
}