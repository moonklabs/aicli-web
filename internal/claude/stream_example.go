package claude

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/sirupsen/logrus"
)

// StreamExample은 Claude CLI 스트림 처리 시스템의 사용 예제입니다.
type StreamExample struct {
	handler StreamHandler
	logger  *logrus.Logger
}

// NewStreamExample은 새로운 스트림 예제를 생성합니다.
func NewStreamExample() *StreamExample {
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	return &StreamExample{
		handler: NewStreamHandler(logger),
		logger:  logger,
	}
}

// BasicUsage는 기본적인 스트림 핸들러 사용법을 보여줍니다.
func (se *StreamExample) BasicUsage() error {
	// Claude CLI 프로세스 시뮬레이션 (실제로는 외부 프로세스)
	cmd := exec.Command("echo", `{"type":"response","content":"Hello from Claude!","message_id":"test-123"}`)
	
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// 프로세스 시작
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start process: %w", err)
	}

	// 스트림 핸들러 시작
	if err := se.handler.Start(stdin, stdout, stderr); err != nil {
		return fmt.Errorf("failed to start stream handler: %w", err)
	}
	defer se.handler.Close()

	// 메시지 전송
	message := &Message{
		Type:    "query",
		Content: "Hello, Claude! How are you today?",
		Meta:    map[string]interface{}{"temperature": 0.7},
	}

	if err := se.handler.SendMessage(message); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	se.logger.Info("Message sent successfully")

	// 응답 수신
	response, err := se.handler.ReceiveMessage(5 * time.Second)
	if err != nil {
		return fmt.Errorf("failed to receive response: %w", err)
	}

	se.logger.WithFields(logrus.Fields{
		"type":    response.Type,
		"content": response.Content,
	}).Info("Response received")

	// 프로세스 종료 대기
	cmd.Wait()

	return nil
}

// EventHandlingExample는 이벤트 처리 예제를 보여줍니다.
func (se *StreamExample) EventHandlingExample() error {
	// 다양한 이벤트 구독
	eventSubscriptions := []struct {
		eventType string
		handler   EventHandler
	}{
		{
			eventType: "message_sent",
			handler: func(event *StreamEvent) error {
				se.logger.WithField("event", event.Type).Info("Message sent event received")
				return nil
			},
		},
		{
			eventType: "response_received",
			handler: func(event *StreamEvent) error {
				se.logger.WithField("event", event.Type).Info("Response received event")
				return nil
			},
		},
		{
			eventType: "stream_error",
			handler: func(event *StreamEvent) error {
				se.logger.WithField("event", event.Type).Error("Stream error event")
				return nil
			},
		},
		{
			eventType: "stderr_data",
			handler: func(event *StreamEvent) error {
				data, ok := event.Data.(map[string]interface{})
				if ok {
					se.logger.WithField("stderr", data["data"]).Warn("Error output received")
				}
				return nil
			},
		},
	}

	// 이벤트 구독 등록
	subscriptions := make([]*EventSubscription, 0, len(eventSubscriptions))
	for _, sub := range eventSubscriptions {
		subscription, err := se.handler.Subscribe(sub.eventType, sub.handler)
		if err != nil {
			return fmt.Errorf("failed to subscribe to %s: %w", sub.eventType, err)
		}
		subscriptions = append(subscriptions, subscription)
	}

	// 시뮬레이션을 위한 mock 스트림 생성
	stdinReader, stdinWriter := io.Pipe()
	stdoutReader, stdoutWriter := io.Pipe()
	stderrReader, stderrWriter := io.Pipe()

	// 스트림 핸들러 시작
	if err := se.handler.Start(stdinWriter, stdoutReader, stderrReader); err != nil {
		return fmt.Errorf("failed to start stream handler: %w", err)
	}
	defer se.handler.Close()

	// 백그라운드에서 응답 시뮬레이션
	go func() {
		defer stdoutWriter.Close()
		time.Sleep(100 * time.Millisecond)
		
		response := `{"type":"response","content":"Event handling example response","message_id":"event-test"}`
		stdoutWriter.Write([]byte(response))
	}()

	// 백그라운드에서 에러 시뮬레이션
	go func() {
		defer stderrWriter.Close()
		time.Sleep(200 * time.Millisecond)
		
		stderrWriter.Write([]byte("Warning: This is a simulated error message"))
	}()

	// 메시지 전송
	message := &Message{
		Type:    "test",
		Content: "Event handling test message",
	}

	if err := se.handler.SendMessage(message); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	// 응답 대기
	response, err := se.handler.ReceiveMessage(2 * time.Second)
	if err != nil {
		return fmt.Errorf("failed to receive response: %w", err)
	}

	se.logger.WithField("response", response.Content).Info("Event handling example completed")

	// 정리
	stdinReader.Close()

	return nil
}

// HighThroughputExample는 고처리량 시나리오 예제를 보여줍니다.
func (se *StreamExample) HighThroughputExample() error {
	// 시뮬레이션을 위한 mock 스트림 생성
	stdinReader, stdinWriter := io.Pipe()
	stdoutReader, stdoutWriter := io.Pipe()
	stderrReader, stderrWriter := io.Pipe()

	// 스트림 핸들러 시작
	if err := se.handler.Start(stdinWriter, stdoutReader, stderrReader); err != nil {
		return fmt.Errorf("failed to start stream handler: %w", err)
	}
	defer se.handler.Close()

	// 통계 추적을 위한 이벤트 구독
	messageCount := 0
	responseCount := 0

	se.handler.Subscribe("message_sent", func(event *StreamEvent) error {
		messageCount++
		if messageCount%10 == 0 {
			se.logger.WithField("count", messageCount).Info("Messages sent")
		}
		return nil
	})

	se.handler.Subscribe("response_received", func(event *StreamEvent) error {
		responseCount++
		if responseCount%10 == 0 {
			se.logger.WithField("count", responseCount).Info("Responses received")
		}
		return nil
	})

	// 백그라운드에서 응답 시뮬레이션
	go func() {
		defer stdoutWriter.Close()
		
		for i := 0; i < 50; i++ {
			response := fmt.Sprintf(`{"type":"response","content":"Response %d","message_id":"batch-%d"}`, i, i)
			stdoutWriter.Write([]byte(response))
			time.Sleep(50 * time.Millisecond)
		}
	}()

	// 대량 메시지 전송
	start := time.Now()
	for i := 0; i < 50; i++ {
		message := &Message{
			Type:    "batch_query",
			Content: fmt.Sprintf("Batch message %d", i),
			Meta:    map[string]interface{}{"batch_id": i},
		}

		if err := se.handler.SendMessage(message); err != nil {
			return fmt.Errorf("failed to send message %d: %w", i, err)
		}

		// 약간의 지연으로 버스트 방지
		time.Sleep(10 * time.Millisecond)
	}

	duration := time.Since(start)
	se.logger.WithFields(logrus.Fields{
		"messages": 50,
		"duration": duration,
		"rate":     float64(50) / duration.Seconds(),
	}).Info("Batch message sending completed")

	// 모든 응답 수신 대기
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	receivedResponses := 0
	for receivedResponses < 50 {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for all responses, received: %d/50", receivedResponses)
		default:
			response, err := se.handler.ReceiveMessage(1 * time.Second)
			if err != nil {
				se.logger.WithError(err).Warn("Failed to receive response")
				continue
			}
			receivedResponses++
			
			if receivedResponses%10 == 0 {
				se.logger.WithField("received", receivedResponses).Info("Progress update")
			}
			
			_ = response // 응답 처리
		}
	}

	se.logger.WithField("total_received", receivedResponses).Info("High throughput example completed")

	// 정리
	stdinReader.Close()
	stderrWriter.Close()

	return nil
}

// StreamStatsExample는 스트림 통계 모니터링 예제입니다.
func (se *StreamExample) StreamStatsExample() error {
	// 시뮬레이션을 위한 mock 스트림 생성
	stdinReader, stdinWriter := io.Pipe()
	stdoutReader, stdoutWriter := io.Pipe()
	stderrReader, stderrWriter := io.Pipe()

	// 스트림 핸들러 시작
	if err := se.handler.Start(stdinWriter, stdoutReader, stderrReader); err != nil {
		return fmt.Errorf("failed to start stream handler: %w", err)
	}
	defer se.handler.Close()

	// 주기적으로 통계 출력
	statsTicker := time.NewTicker(2 * time.Second)
	defer statsTicker.Stop()

	go func() {
		for range statsTicker.C {
			stats := se.handler.GetStats()
			se.logger.WithFields(logrus.Fields{
				"running":           stats["is_running"],
				"messages_sent":     stats["messages_sent"],
				"messages_received": stats["messages_received"],
				"errors":           stats["errors"],
				"uptime":           stats["uptime_seconds"],
			}).Info("Stream handler statistics")

			// 버퍼 통계
			if bufferStats, ok := stats["buffer"].(map[string]interface{}); ok {
				se.logger.WithFields(logrus.Fields{
					"buffer_size":  bufferStats["size"],
					"buffer_usage": bufferStats["usage_ratio"],
					"overflow":     bufferStats["overflow"],
				}).Debug("Buffer statistics")
			}

			// 이벤트 통계
			if eventStats, ok := stats["events"].(map[string]interface{}); ok {
				se.logger.WithFields(logrus.Fields{
					"published_events":   eventStats["published_events"],
					"delivered_events":   eventStats["delivered_events"],
					"active_subscribers": eventStats["active_subscribers"],
					"success_rate":       eventStats["success_rate"],
				}).Debug("Event bus statistics")
			}
		}
	}()

	// 백그라운드 작업 시뮬레이션
	go func() {
		defer stdoutWriter.Close()
		
		for i := 0; i < 5; i++ {
			time.Sleep(1 * time.Second)
			response := fmt.Sprintf(`{"type":"response","content":"Stats example response %d","message_id":"stats-%d"}`, i, i)
			stdoutWriter.Write([]byte(response))
		}
	}()

	// 메시지 전송과 응답 수신
	for i := 0; i < 5; i++ {
		message := &Message{
			Type:    "stats_test",
			Content: fmt.Sprintf("Stats test message %d", i),
		}

		if err := se.handler.SendMessage(message); err != nil {
			return fmt.Errorf("failed to send message: %w", err)
		}

		response, err := se.handler.ReceiveMessage(3 * time.Second)
		if err != nil {
			se.logger.WithError(err).Warn("Failed to receive response")
			continue
		}

		se.logger.WithField("response_type", response.Type).Info("Response received")
		time.Sleep(1 * time.Second)
	}

	// 정리
	stdinReader.Close()
	stderrWriter.Close()

	return nil
}

// RunAllExamples는 모든 예제를 실행합니다.
func (se *StreamExample) RunAllExamples() {
	examples := []struct {
		name string
		fn   func() error
	}{
		{"Basic Usage", se.BasicUsage},
		{"Event Handling", se.EventHandlingExample},
		{"High Throughput", se.HighThroughputExample},
		{"Stream Statistics", se.StreamStatsExample},
	}

	for _, example := range examples {
		se.logger.WithField("example", example.name).Info("Running example")
		
		if err := example.fn(); err != nil {
			se.logger.WithError(err).WithField("example", example.name).Error("Example failed")
		} else {
			se.logger.WithField("example", example.name).Info("Example completed successfully")
		}
		
		time.Sleep(1 * time.Second) // 예제 간 간격
	}
}

// main 함수는 예제 실행을 위한 것입니다.
func main() {
	example := NewStreamExample()
	
	// 특정 예제만 실행하려면:
	// example.BasicUsage()
	
	// 모든 예제 실행:
	example.RunAllExamples()
}