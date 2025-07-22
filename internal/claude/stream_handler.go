package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// StreamHandler는 Claude CLI와의 입출력 스트림을 처리하는 인터페이스입니다.
type StreamHandler interface {
	// 기본 스트림 처리
	Start(stdin io.WriteCloser, stdout, stderr io.ReadCloser) error
	SendMessage(msg *StreamMessage) error
	ReceiveMessage(timeout time.Duration) (*Response, error)
	Subscribe(eventType string, handler EventHandler) (*EventSubscription, error)
	Close() error
	IsRunning() bool
	GetStats() map[string]interface{}
	
	// 새로운 스트림 처리 메서드
	Stream(ctx context.Context, reader io.Reader) (<-chan StreamMessage, error)
	StreamWithCallback(ctx context.Context, reader io.Reader, callback MessageCallback) error
	SetBufferSize(size int)
	GetMetrics() StreamMetrics
}

// MessageCallback은 메시지 처리 콜백 함수 타입입니다.
type MessageCallback func(msg StreamMessage) error

// StreamMetrics는 스트림 처리 메트릭을 정의합니다.
type StreamMetrics struct {
	MessagesReceived   int64
	BytesProcessed     int64
	ParseErrors        int64
	BackpressureEvents int64
	AvgProcessingTime  time.Duration
}

// claudeStreamHandler는 StreamHandler 인터페이스의 구현체입니다.
type claudeStreamHandler struct {
	stdin        io.WriteCloser
	stdout       io.ReadCloser
	stderr       io.ReadCloser
	parser       *JSONStreamParser
	eventBus     *EventBus
	buffer       *StreamBuffer
	isRunning    bool
	mutex        sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	logger       *logrus.Logger
	
	// 채널들
	responseChan chan *Response
	errorChan    chan error
	
	// 메트릭
	messagesSent     int64
	messagesReceived int64
	errors           int64
	startTime        time.Time
	bytesProcessed   int64
	parseErrors      int64
	
	// 백프레셔 및 라우팅
	backpressure    *BackpressureHandler
	messageRouter   *MessageRouter
	bufferSize      int
	metrics         *StreamMetrics
}

// NewStreamHandler는 새로운 스트림 핸들러를 생성합니다.
func NewStreamHandler(logger *logrus.Logger) StreamHandler {
	// 백프레셔 설정
	backpressureConfig := BackpressureConfig{
		MaxBufferSize:     1000,
		DropPolicy:        BlockUntilReady,
		AdaptiveBuffering: true,
		MinBufferSize:     100,
		BufferGrowthRate:  1.5,
		BufferShrinkRate:  0.8,
	}
	
	// 메시지 라우터 설정
	routerConfig := RouterConfig{
		AsyncMode:      true,
		MaxConcurrency: 10,
	}
	
	handler := &claudeStreamHandler{
		eventBus:       NewEventBus(logger),
		buffer:         NewStreamBuffer(1024 * 1024), // 1MB 버퍼
		logger:         logger,
		responseChan:   make(chan *Response, 100),
		errorChan:      make(chan error, 10),
		backpressure:   NewBackpressureHandler(backpressureConfig, logger),
		messageRouter:  NewMessageRouter(routerConfig, logger),
		bufferSize:     100,
		metrics:        &StreamMetrics{},
	}
	
	// 기본 메시지 핸들러 등록
	handler.registerDefaultHandlers()
	
	return handler
}

// Start는 스트림 핸들러를 시작합니다.
func (sh *claudeStreamHandler) Start(stdin io.WriteCloser, stdout, stderr io.ReadCloser) error {
	sh.mutex.Lock()
	defer sh.mutex.Unlock()

	if sh.isRunning {
		return fmt.Errorf("stream handler is already running")
	}

	sh.stdin = stdin
	sh.stdout = stdout
	sh.stderr = stderr
	sh.parser = NewJSONStreamParser(stdout, sh.logger)
	sh.ctx, sh.cancel = context.WithCancel(context.Background())
	sh.isRunning = true
	sh.startTime = time.Now()

	// 스트림 처리 고루틴 시작
	go sh.processOutputStream()
	go sh.processErrorStream()

	// 시작 이벤트 발행
	sh.eventBus.Publish(&StreamEvent{
		Type:      "stream_started",
		Data:      map[string]interface{}{"timestamp": time.Now()},
		Timestamp: time.Now(),
		Source:    "stream_handler",
	})

	sh.logger.Info("Stream handler started")
	return nil
}

// processOutputStream은 stdout 스트림을 처리합니다.
func (sh *claudeStreamHandler) processOutputStream() {
	responseChan, errorChan := sh.parser.ParseStream(sh.ctx)

	for {
		select {
		case response := <-responseChan:
			if response == nil {
				return
			}
			sh.handleResponse(response)

		case err := <-errorChan:
			if err != nil {
				sh.handleStreamError(err)
				return
			}

		case <-sh.ctx.Done():
			return
		}
	}
}

// processErrorStream은 stderr 스트림을 처리합니다.
func (sh *claudeStreamHandler) processErrorStream() {
	buffer := make([]byte, 4096)
	
	for {
		select {
		case <-sh.ctx.Done():
			return
		default:
			n, err := sh.stderr.Read(buffer)
			if err != nil {
				if err != io.EOF {
					sh.logger.WithError(err).Error("Error reading stderr")
				}
				return
			}

			if n > 0 {
				errorData := buffer[:n]
				sh.buffer.Write(errorData)

				// 에러 이벤트 발행
				sh.eventBus.Publish(&StreamEvent{
					Type: "stderr_data",
					Data: map[string]interface{}{
						"data": string(errorData),
						"size": n,
					},
					Timestamp: time.Now(),
					Source:    "stream_handler",
				})
			}
		}
	}
}

// handleResponse는 응답을 처리합니다.
func (sh *claudeStreamHandler) handleResponse(response *Response) {
	sh.messagesSent++

	// 응답을 채널로 전송
	select {
	case sh.responseChan <- response:
	default:
		sh.logger.Warn("Response channel is full, dropping response")
	}

	// 응답 이벤트 발행
	sh.eventBus.Publish(&StreamEvent{
		Type: "response_received",
		Data: map[string]interface{}{
			"type":       response.Type,
			"message_id": response.MessageID,
			"has_error":  response.Error != nil,
		},
		Timestamp: time.Now(),
		Source:    "stream_handler",
		ID:        response.MessageID,
	})
}

// handleStreamError는 스트림 에러를 처리합니다.
func (sh *claudeStreamHandler) handleStreamError(err error) {
	sh.errors++

	// 에러를 채널로 전송
	select {
	case sh.errorChan <- err:
	default:
		sh.logger.Warn("Error channel is full, dropping error")
	}

	// 에러 이벤트 발행
	sh.eventBus.Publish(&StreamEvent{
		Type: "stream_error",
		Data: map[string]interface{}{
			"error":     err.Error(),
			"timestamp": time.Now(),
		},
		Timestamp: time.Now(),
		Source:    "stream_handler",
	})
}

// SendMessage는 메시지를 전송합니다.
func (sh *claudeStreamHandler) SendMessage(msg *StreamMessage) error {
	sh.mutex.RLock()
	defer sh.mutex.RUnlock()

	if !sh.isRunning {
		return fmt.Errorf("stream handler is not running")
	}

	// 메시지 ID 생성
	if msg.ID == "" {
		msg.ID = generateMessageID()
	}

	// JSON 인코딩
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// 스트림에 쓰기
	if _, err := sh.stdin.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write to stdin: %w", err)
	}

	sh.messagesReceived++

	// 메시지 전송 이벤트 발행
	sh.eventBus.Publish(&StreamEvent{
		Type: "message_sent",
		Data: map[string]interface{}{
			"type": msg.Type,
			"id":   msg.ID,
			"size": len(data),
		},
		Timestamp: time.Now(),
		Source:    "stream_handler",
		ID:        msg.ID,
	})

	sh.logger.WithFields(logrus.Fields{
		"type": msg.Type,
		"id":   msg.ID,
	}).Debug("Message sent")

	return nil
}

// ReceiveMessage는 메시지를 수신합니다.
func (sh *claudeStreamHandler) ReceiveMessage(timeout time.Duration) (*Response, error) {
	ctx, cancel := context.WithTimeout(sh.ctx, timeout)
	defer cancel()

	select {
	case response := <-sh.responseChan:
		return response, nil
	case err := <-sh.errorChan:
		return nil, err
	case <-ctx.Done():
		return nil, fmt.Errorf("receive timeout after %v", timeout)
	case <-sh.ctx.Done():
		return nil, fmt.Errorf("stream handler is shutting down")
	}
}

// Subscribe는 이벤트 구독을 등록합니다.
func (sh *claudeStreamHandler) Subscribe(eventType string, handler EventHandler) (*EventSubscription, error) {
	return sh.eventBus.Subscribe(eventType, handler)
}

// Close는 스트림 핸들러를 종료합니다.
func (sh *claudeStreamHandler) Close() error {
	sh.mutex.Lock()
	defer sh.mutex.Unlock()

	if !sh.isRunning {
		return fmt.Errorf("stream handler is not running")
	}

	// 컨텍스트 취소
	sh.cancel()

	// 스트림 닫기
	if sh.stdin != nil {
		sh.stdin.Close()
	}
	if sh.stdout != nil {
		sh.stdout.Close()
	}
	if sh.stderr != nil {
		sh.stderr.Close()
	}

	// 이벤트 버스 종료
	sh.eventBus.Close()

	// 채널 닫기
	close(sh.responseChan)
	close(sh.errorChan)

	sh.isRunning = false

	// 종료 이벤트 발행 (이벤트 버스 종료 전에)
	sh.eventBus.Publish(&StreamEvent{
		Type: "stream_closed",
		Data: map[string]interface{}{
			"uptime_seconds": time.Since(sh.startTime).Seconds(),
		},
		Timestamp: time.Now(),
		Source:    "stream_handler",
	})

	sh.logger.Info("Stream handler closed")
	return nil
}

// IsRunning은 스트림 핸들러가 실행 중인지 확인합니다.
func (sh *claudeStreamHandler) IsRunning() bool {
	sh.mutex.RLock()
	defer sh.mutex.RUnlock()

	return sh.isRunning
}

// GetStats는 스트림 핸들러의 통계 정보를 반환합니다.
func (sh *claudeStreamHandler) GetStats() map[string]interface{} {
	sh.mutex.RLock()
	defer sh.mutex.RUnlock()

	stats := map[string]interface{}{
		"is_running":         sh.isRunning,
		"messages_sent":      sh.messagesSent,
		"messages_received":  sh.messagesReceived,
		"errors":            sh.errors,
		"uptime_seconds":    0,
	}

	if !sh.startTime.IsZero() {
		stats["uptime_seconds"] = time.Since(sh.startTime).Seconds()
	}

	// 버퍼 통계 추가
	if sh.buffer != nil {
		bufferStats := sh.buffer.GetStats()
		stats["buffer"] = bufferStats
	}

	// 이벤트 버스 통계 추가
	if sh.eventBus != nil {
		eventStats := sh.eventBus.GetMetrics()
		stats["events"] = eventStats
	}

	return stats
}

// generateMessageID는 고유한 메시지 ID를 생성합니다.
func generateMessageID() string {
	return uuid.New().String()
}

// Flush는 버퍼된 데이터를 강제로 플러시합니다.
func (sh *claudeStreamHandler) Flush() error {
	sh.mutex.RLock()
	defer sh.mutex.RUnlock()

	if !sh.isRunning {
		return fmt.Errorf("stream handler is not running")
	}

	// stdin 플러시 (WriteCloser가 Flusher 인터페이스를 구현하는 경우)
	if flusher, ok := sh.stdin.(interface{ Flush() error }); ok {
		return flusher.Flush()
	}

	return nil
}

// GetBuffer는 내부 버퍼에 접근합니다 (디버깅/테스트용).
func (sh *claudeStreamHandler) GetBuffer() *StreamBuffer {
	sh.mutex.RLock()
	defer sh.mutex.RUnlock()

	return sh.buffer
}

// Stream은 리더에서 메시지를 스트리밍합니다.
func (sh *claudeStreamHandler) Stream(ctx context.Context, reader io.Reader) (<-chan StreamMessage, error) {
	messageChan := make(chan StreamMessage, sh.bufferSize)
	
	go func() {
		defer close(messageChan)
		
		parser := NewJSONStreamParser(reader, sh.logger)
		responseChan, errorChan := parser.ParseStream(ctx)
		
		for {
			select {
			case <-ctx.Done():
				return
			case response := <-responseChan:
				if response == nil {
					return
				}
				
				// Response를 Message로 변환
				msg := Message{
					Type:    response.Type,
					Content: response.Content,
					Meta:    response.Metadata,
					ID:      response.MessageID,
				}
				
				// 백프레셔 처리
				if sh.backpressure.ShouldDrop() {
					if err := sh.backpressure.WaitForSpace(ctx); err != nil {
						sh.logger.WithError(err).Error("Backpressure wait failed")
						return
					}
				}
				
				sh.backpressure.IncrementBuffer()
				atomic.AddInt64(&sh.messagesReceived, 1)
				atomic.AddInt64(&sh.bytesProcessed, int64(len(response.Content)))
				
				select {
				case messageChan <- msg:
				case <-ctx.Done():
					return
				}
				
			case err := <-errorChan:
				if err != nil {
					atomic.AddInt64(&sh.parseErrors, 1)
					sh.logger.WithError(err).Error("Parse error in stream")
				}
				return
			}
		}
	}()
	
	// 백프레셔 모니터링 시작
	go sh.backpressure.MonitorSlowConsumers(ctx)
	
	return messageChan, nil
}

// StreamWithCallback은 콜백을 사용하여 스트림을 처리합니다.
func (sh *claudeStreamHandler) StreamWithCallback(ctx context.Context, reader io.Reader, callback MessageCallback) error {
	messageChan, err := sh.Stream(ctx, reader)
	if err != nil {
		return err
	}
	
	for msg := range messageChan {
		start := time.Now()
		
		// 메시지 라우터로 전달
		if err := sh.messageRouter.Route(ctx, msg); err != nil {
			sh.logger.WithError(err).Error("Message routing failed")
		}
		
		// 콜백 실행
		if callback != nil {
			if err := callback(msg); err != nil {
				sh.logger.WithError(err).Error("Message callback failed")
			}
		}
		
		// 메트릭 업데이트
		processingTime := time.Since(start)
		sh.updateProcessingTime(processingTime)
		
		sh.backpressure.DecrementBuffer()
	}
	
	return nil
}

// SetBufferSize는 버퍼 크기를 설정합니다.
func (sh *claudeStreamHandler) SetBufferSize(size int) {
	sh.mutex.Lock()
	defer sh.mutex.Unlock()
	
	if size <= 0 {
		size = 100
	}
	sh.bufferSize = size
	
	// 백프레셔 핸들러 버퍼 크기도 조정
	if sh.backpressure != nil {
		sh.backpressure.mu.Lock()
		sh.backpressure.maxBufferSize = size
		sh.backpressure.mu.Unlock()
	}
}

// GetMetrics는 스트림 메트릭을 반환합니다.
func (sh *claudeStreamHandler) GetMetrics() StreamMetrics {
	sh.mutex.RLock()
	defer sh.mutex.RUnlock()
	
	backpressureMetrics := sh.backpressure.GetMetrics()
	
	return StreamMetrics{
		MessagesReceived:   atomic.LoadInt64(&sh.messagesReceived),
		BytesProcessed:     atomic.LoadInt64(&sh.bytesProcessed),
		ParseErrors:        atomic.LoadInt64(&sh.parseErrors),
		BackpressureEvents: backpressureMetrics.BackpressureEvents,
		AvgProcessingTime:  sh.metrics.AvgProcessingTime,
	}
}

// registerDefaultHandlers는 기본 메시지 핸들러들을 등록합니다.
func (sh *claudeStreamHandler) registerDefaultHandlers() {
	// 텍스트 메시지 핸들러
	textHandler := NewTextMessageHandler(func(text string) error {
		sh.logger.Debug("Text message: ", text)
		return nil
	}, sh.logger)
	sh.messageRouter.RegisterHandler(MessageTypeText, textHandler)
	
	// 에러 메시지 핸들러
	errorHandler := NewErrorMessageHandler(func(err error, meta map[string]interface{}) {
		sh.logger.WithError(err).Error("Claude error")
		atomic.AddInt64(&sh.errors, 1)
	}, sh.logger)
	sh.messageRouter.RegisterHandler(MessageTypeError, errorHandler)
	
	// 시스템 메시지 핸들러
	systemHandler := NewSystemMessageHandler(func(event string, data map[string]interface{}) {
		sh.eventBus.Publish(&StreamEvent{
			Type:      event,
			Data:      data,
			Timestamp: time.Now(),
			Source:    "claude",
		})
	}, sh.logger)
	sh.messageRouter.RegisterHandler(MessageTypeSystem, systemHandler)
	
	// 진행률 메시지 핸들러
	progressHandler := NewProgressMessageHandler(func(taskID string, progress *TaskProgress) {
		sh.eventBus.Publish(&StreamEvent{
			Type: "progress_update",
			Data: map[string]interface{}{
				"task_id":    taskID,
				"percentage": progress.Percentage,
				"status":     progress.Status,
			},
			Timestamp: time.Now(),
			Source:    "claude",
		})
	}, sh.logger)
	sh.messageRouter.RegisterHandler(MessageTypeProgress, progressHandler)
	
	// 완료 메시지 핸들러
	completeHandler := NewCompleteMessageHandler(func(result map[string]interface{}) {
		sh.eventBus.Publish(&StreamEvent{
			Type:      "task_complete",
			Data:      result,
			Timestamp: time.Now(),
			Source:    "claude",
		})
	}, sh.logger)
	sh.messageRouter.RegisterHandler(MessageTypeComplete, completeHandler)
}

// updateProcessingTime은 처리 시간을 업데이트합니다.
func (sh *claudeStreamHandler) updateProcessingTime(duration time.Duration) {
	sh.mutex.Lock()
	defer sh.mutex.Unlock()
	
	// 이동 평균 계산
	alpha := 0.1
	current := sh.metrics.AvgProcessingTime
	sh.metrics.AvgProcessingTime = time.Duration(
		alpha*float64(duration) + (1-alpha)*float64(current),
	)
}