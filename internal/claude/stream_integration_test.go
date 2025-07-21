package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStreamHandler_Integration는 스트림 핸들러의 통합 테스트입니다.
func TestStreamHandler_Integration(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// 테스트용 스트림 생성
	stdin := newMockPipe()
	stdout := newMockPipe()
	stderr := newMockPipe()

	// 스트림 핸들러 생성 및 시작
	handler := NewStreamHandler(logger)
	err := handler.Start(stdin, stdout, stderr)
	require.NoError(t, err)
	defer handler.Close()

	// 이벤트 구독
	events := make(chan *StreamEvent, 10)
	_, err = handler.Subscribe("message_sent", func(event *StreamEvent) error {
		events <- event
		return nil
	})
	require.NoError(t, err)

	_, err = handler.Subscribe("response_received", func(event *StreamEvent) error {
		events <- event
		return nil
	})
	require.NoError(t, err)

	// 1. 메시지 전송
	msg := &Message{
		Type:    "query",
		Content: "Hello, Claude! How are you?",
		Meta:    map[string]interface{}{"source": "test"},
	}

	err = handler.SendMessage(msg)
	require.NoError(t, err)

	// 2. 전송 이벤트 확인
	select {
	case event := <-events:
		assert.Equal(t, "message_sent", event.Type)
		data, ok := event.Data.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "query", data["type"])
	case <-time.After(1 * time.Second):
		t.Fatal("Message sent event not received")
	}

	// 3. stdout에 응답 데이터 시뮬레이션
	response := &Response{
		Type:      "response",
		Content:   "Hello! I'm doing well, thank you for asking.",
		MessageID: msg.ID,
		Metadata:  map[string]interface{}{"tokens": 12},
	}
	responseJSON, _ := json.Marshal(response)
	stdout.Write(responseJSON)

	// 4. 응답 수신
	received, err := handler.ReceiveMessage(2 * time.Second)
	require.NoError(t, err)
	assert.Equal(t, response.Type, received.Type)
	assert.Equal(t, response.Content, received.Content)
	assert.Equal(t, response.MessageID, received.MessageID)

	// 5. 응답 수신 이벤트 확인
	select {
	case event := <-events:
		assert.Equal(t, "response_received", event.Type)
	case <-time.After(1 * time.Second):
		t.Fatal("Response received event not received")
	}

	// 6. 통계 확인
	stats := handler.GetStats()
	assert.True(t, stats["is_running"].(bool))
	assert.Equal(t, int64(1), stats["messages_received"].(int64))
	assert.Equal(t, int64(1), stats["messages_sent"].(int64))
}

// TestStreamHandler_ErrorHandling은 에러 처리 테스트입니다.
func TestStreamHandler_ErrorHandling(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	stdin := newMockPipe()
	stdout := newMockPipe()
	stderr := newMockPipe()

	handler := NewStreamHandler(logger)
	err := handler.Start(stdin, stdout, stderr)
	require.NoError(t, err)
	defer handler.Close()

	// stderr에 에러 데이터 쓰기
	errorData := "Error: Claude CLI failed to process request"
	stderr.WriteString(errorData)

	// 에러 이벤트 구독
	errorEvents := make(chan *StreamEvent, 1)
	_, err = handler.Subscribe("stderr_data", func(event *StreamEvent) error {
		errorEvents <- event
		return nil
	})
	require.NoError(t, err)

	// 짧은 대기 후 에러 이벤트 확인
	time.Sleep(100 * time.Millisecond)

	select {
	case event := <-errorEvents:
		assert.Equal(t, "stderr_data", event.Type)
		data, ok := event.Data.(map[string]interface{})
		require.True(t, ok)
		assert.Contains(t, data["data"].(string), "Error:")
	case <-time.After(1 * time.Second):
		t.Fatal("Error event not received")
	}
}

// TestStreamHandler_MultipleMessages는 다중 메시지 처리 테스트입니다.
func TestStreamHandler_MultipleMessages(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	stdin := newMockPipe()
	stdout := newMockPipe()
	stderr := newMockPipe()

	handler := NewStreamHandler(logger)
	err := handler.Start(stdin, stdout, stderr)
	require.NoError(t, err)
	defer handler.Close()

	// 여러 메시지 전송
	messages := []*Message{
		{Type: "query", Content: "First question"},
		{Type: "query", Content: "Second question"},
		{Type: "command", Content: "Execute task"},
	}

	for _, msg := range messages {
		err = handler.SendMessage(msg)
		require.NoError(t, err)
	}

	// stdin 내용 확인
	sentData := stdin.String()
	lines := strings.Split(strings.TrimSpace(sentData), "\n")
	assert.Equal(t, len(messages), len(lines))

	for i, line := range lines {
		var parsed Message
		err = json.Unmarshal([]byte(line), &parsed)
		require.NoError(t, err)
		assert.Equal(t, messages[i].Type, parsed.Type)
		assert.Equal(t, messages[i].Content, parsed.Content)
	}

	// 통계 확인
	stats := handler.GetStats()
	assert.Equal(t, int64(len(messages)), stats["messages_received"].(int64))
}

// TestStreamBuffer_LargeData는 대용량 데이터 처리 테스트입니다.
func TestStreamBuffer_LargeData(t *testing.T) {
	buffer := NewStreamBuffer(1024) // 1KB 버퍼

	// 10KB 데이터 생성
	largeData := bytes.Repeat([]byte("A"), 10*1024)

	// 데이터 쓰기
	n, err := buffer.Write(largeData)
	assert.NoError(t, err)
	assert.Equal(t, len(largeData), n)

	// 오버플로우 확인
	assert.True(t, buffer.HasOverflow())
	assert.Equal(t, 1024, buffer.Len()) // 버퍼 크기로 제한됨

	// 통계 확인
	stats := buffer.GetStats()
	assert.True(t, stats["overflow"].(bool))
	assert.Equal(t, 1.0, stats["usage_ratio"].(float64))
}

// TestEventBus_HighLoad는 이벤트 버스의 고부하 테스트입니다.
func TestEventBus_HighLoad(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel) // 로그 줄이기

	eventBus := NewEventBus(logger)
	defer eventBus.Close()

	// 여러 구독자 등록
	eventCount := 0
	subscribers := 10
	eventsPerSubscriber := make([]chan *StreamEvent, subscribers)

	for i := 0; i < subscribers; i++ {
		events := make(chan *StreamEvent, 100)
		eventsPerSubscriber[i] = events

		_, err := eventBus.Subscribe("test", func(event *StreamEvent) error {
			events <- event
			return nil
		})
		require.NoError(t, err)
	}

	// 100개 이벤트 발행
	eventCount = 100
	for i := 0; i < eventCount; i++ {
		event := &StreamEvent{
			Type: "test",
			Data: map[string]interface{}{"index": i},
		}
		eventBus.Publish(event)
	}

	// 모든 구독자가 모든 이벤트를 수신했는지 확인
	for i := 0; i < subscribers; i++ {
		received := 0
		timeout := time.After(2 * time.Second)

		for received < eventCount {
			select {
			case <-eventsPerSubscriber[i]:
				received++
			case <-timeout:
				t.Fatalf("Subscriber %d received only %d/%d events", i, received, eventCount)
			}
		}
	}

	// 메트릭 확인
	metrics := eventBus.GetMetrics()
	assert.Equal(t, int64(eventCount), metrics["published_events"].(int64))
	assert.Equal(t, subscribers, metrics["active_subscribers"].(int))
}

// TestJSONStreamParser_RealTimeStreaming은 실시간 스트리밍 파싱 테스트입니다.
func TestJSONStreamParser_RealTimeStreaming(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// 파이프 생성
	reader, writer := io.Pipe()
	defer reader.Close()
	defer writer.Close()

	parser := NewJSONStreamParser(reader, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	responseChan, errorChan := parser.ParseStream(ctx)

	// 응답 수집을 위한 채널
	responses := make(chan *Response, 10)
	go func() {
		for {
			select {
			case response := <-responseChan:
				if response != nil {
					responses <- response
				}
			case err := <-errorChan:
				if err != nil {
					t.Logf("Parser error: %v", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// 실시간으로 JSON 객체 전송
	testResponses := []*Response{
		{Type: "start", Content: "Starting conversation"},
		{Type: "thinking", Content: "Processing request"},
		{Type: "response", Content: "Here is the answer"},
		{Type: "end", Content: "Conversation completed"},
	}

	go func() {
		defer writer.Close()
		for i, resp := range testResponses {
			data, _ := json.Marshal(resp)
			writer.Write(data)
			
			// 스트리밍 시뮬레이션을 위한 지연
			if i < len(testResponses)-1 {
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()

	// 응답 수신 확인
	receivedCount := 0
	timeout := time.After(3 * time.Second)

	for receivedCount < len(testResponses) {
		select {
		case response := <-responses:
			assert.Equal(t, testResponses[receivedCount].Type, response.Type)
			assert.Equal(t, testResponses[receivedCount].Content, response.Content)
			receivedCount++
		case <-timeout:
			t.Fatalf("Received only %d/%d responses", receivedCount, len(testResponses))
		}
	}
}