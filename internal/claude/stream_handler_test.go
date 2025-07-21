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

// mockPipe는 테스트를 위한 파이프 구조체입니다.
type mockPipe struct {
	*bytes.Buffer
}

func (m *mockPipe) Close() error {
	return nil
}

func newMockPipe() *mockPipe {
	return &mockPipe{Buffer: &bytes.Buffer{}}
}

func TestStreamHandler_Basic(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	handler := NewStreamHandler(logger)
	assert.NotNil(t, handler)
	assert.False(t, handler.IsRunning())
}

func TestStreamHandler_StartAndStop(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	handler := NewStreamHandler(logger)

	stdin := newMockPipe()
	stdout := newMockPipe()
	stderr := newMockPipe()

	// 스트림 핸들러 시작
	err := handler.Start(stdin, stdout, stderr)
	assert.NoError(t, err)
	assert.True(t, handler.IsRunning())

	// 중복 시작 시도
	err = handler.Start(stdin, stdout, stderr)
	assert.Error(t, err)

	// 종료
	err = handler.Close()
	assert.NoError(t, err)
	assert.False(t, handler.IsRunning())
}

func TestStreamHandler_SendMessage(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	handler := NewStreamHandler(logger)
	stdin := newMockPipe()
	stdout := newMockPipe()
	stderr := newMockPipe()

	err := handler.Start(stdin, stdout, stderr)
	require.NoError(t, err)
	defer handler.Close()

	// 메시지 전송
	msg := &Message{
		Type:    "test",
		Content: "Hello, Claude!",
		ID:      "test-123",
	}

	err = handler.SendMessage(msg)
	assert.NoError(t, err)

	// stdin에 데이터가 쓰여졌는지 확인
	data := stdin.String()
	assert.Contains(t, data, "test")
	assert.Contains(t, data, "Hello, Claude!")
	assert.Contains(t, data, "test-123")

	// JSON 형식 검증
	var parsed Message
	lines := strings.Split(strings.TrimSpace(data), "\n")
	err = json.Unmarshal([]byte(lines[0]), &parsed)
	assert.NoError(t, err)
	assert.Equal(t, msg.Type, parsed.Type)
	assert.Equal(t, msg.Content, parsed.Content)
	assert.Equal(t, msg.ID, parsed.ID)
}

func TestStreamHandler_ReceiveMessage(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	handler := NewStreamHandler(logger)
	stdin := newMockPipe()
	stdout := newMockPipe()
	stderr := newMockPipe()

	// 응답 데이터 준비
	response := &Response{
		Type:      "response",
		Content:   "Hello from Claude!",
		MessageID: "test-123",
	}
	responseJSON, _ := json.Marshal(response)
	stdout.Write(responseJSON)

	err := handler.Start(stdin, stdout, stderr)
	require.NoError(t, err)
	defer handler.Close()

	// 짧은 대기 후 메시지 수신 시도
	time.Sleep(100 * time.Millisecond)
	
	received, err := handler.ReceiveMessage(1 * time.Second)
	assert.NoError(t, err)
	assert.NotNil(t, received)
	assert.Equal(t, response.Type, received.Type)
	assert.Equal(t, response.Content, received.Content)
	assert.Equal(t, response.MessageID, received.MessageID)
}

func TestStreamHandler_ReceiveTimeout(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	handler := NewStreamHandler(logger)
	stdin := newMockPipe()
	stdout := newMockPipe()
	stderr := newMockPipe()

	err := handler.Start(stdin, stdout, stderr)
	require.NoError(t, err)
	defer handler.Close()

	// 타임아웃 테스트
	received, err := handler.ReceiveMessage(100 * time.Millisecond)
	assert.Error(t, err)
	assert.Nil(t, received)
	assert.Contains(t, err.Error(), "timeout")
}

func TestStreamHandler_Events(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	handler := NewStreamHandler(logger)
	stdin := newMockPipe()
	stdout := newMockPipe()
	stderr := newMockPipe()

	// 이벤트 구독
	events := make(chan *StreamEvent, 10)
	subscription, err := handler.Subscribe("message_sent", func(event *StreamEvent) error {
		events <- event
		return nil
	})
	require.NoError(t, err)
	assert.NotNil(t, subscription)

	err = handler.Start(stdin, stdout, stderr)
	require.NoError(t, err)
	defer handler.Close()

	// 메시지 전송
	msg := &Message{
		Type:    "test",
		Content: "Hello!",
		ID:      "test-123",
	}

	err = handler.SendMessage(msg)
	require.NoError(t, err)

	// 이벤트 수신 확인
	select {
	case event := <-events:
		assert.Equal(t, "message_sent", event.Type)
		assert.NotNil(t, event.Data)
		data, ok := event.Data.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "test", data["type"])
		assert.Equal(t, "test-123", data["id"])
	case <-time.After(1 * time.Second):
		t.Fatal("Event not received within timeout")
	}
}

func TestStreamHandler_Stats(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	handler := NewStreamHandler(logger)
	stdin := newMockPipe()
	stdout := newMockPipe()
	stderr := newMockPipe()

	err := handler.Start(stdin, stdout, stderr)
	require.NoError(t, err)
	defer handler.Close()

	// 초기 통계
	stats := handler.GetStats()
	assert.NotNil(t, stats)
	assert.True(t, stats["is_running"].(bool))
	assert.Equal(t, int64(0), stats["messages_sent"].(int64))
	assert.Equal(t, int64(0), stats["messages_received"].(int64))

	// 메시지 전송 후 통계
	msg := &Message{Type: "test", Content: "Hello!"}
	err = handler.SendMessage(msg)
	require.NoError(t, err)

	stats = handler.GetStats()
	assert.Equal(t, int64(1), stats["messages_received"].(int64))
}

func TestStreamBuffer_Basic(t *testing.T) {
	buffer := NewStreamBuffer(100)
	assert.NotNil(t, buffer)
	assert.Equal(t, 0, buffer.Len())
	assert.Equal(t, 100, buffer.Cap())
	assert.True(t, buffer.IsEmpty())
	assert.False(t, buffer.IsFull())
	assert.False(t, buffer.HasOverflow())
}

func TestStreamBuffer_WriteRead(t *testing.T) {
	buffer := NewStreamBuffer(100)

	// 쓰기
	data := []byte("Hello, World!")
	n, err := buffer.Write(data)
	assert.NoError(t, err)
	assert.Equal(t, len(data), n)
	assert.Equal(t, len(data), buffer.Len())
	assert.False(t, buffer.IsEmpty())

	// 읽기
	readData := make([]byte, len(data))
	n, err = buffer.Read(readData)
	assert.NoError(t, err)
	assert.Equal(t, len(data), n)
	assert.Equal(t, data, readData)
	assert.True(t, buffer.IsEmpty())
}

func TestStreamBuffer_Overflow(t *testing.T) {
	buffer := NewStreamBuffer(10)

	// 버퍼 크기보다 큰 데이터 쓰기
	data := []byte("This is a very long string that exceeds buffer size")
	n, err := buffer.Write(data)
	assert.NoError(t, err)
	assert.Equal(t, len(data), n)
	assert.True(t, buffer.HasOverflow())
	assert.Equal(t, 10, buffer.Len()) // 최대 크기로 제한
}

func TestStreamBuffer_Stats(t *testing.T) {
	buffer := NewStreamBuffer(100)
	
	data := []byte("Hello!")
	buffer.Write(data)

	stats := buffer.GetStats()
	assert.NotNil(t, stats)
	assert.Equal(t, len(data), stats["size"].(int))
	assert.Equal(t, 100, stats["max_size"].(int))
	assert.Equal(t, false, stats["overflow"].(bool))
	assert.Equal(t, int64(len(data)), stats["written"].(int64))
	assert.Equal(t, int64(0), stats["read"].(int64))
}

func TestEventBus_Basic(t *testing.T) {
	logger := logrus.New()
	eventBus := NewEventBus(logger)
	defer eventBus.Close()

	assert.NotNil(t, eventBus)

	// 구독자 없는 상태에서 이벤트 발행
	event := &StreamEvent{
		Type: "test",
		Data: "test data",
	}
	eventBus.Publish(event) // 에러 없이 완료되어야 함
}

func TestEventBus_SubscribeUnsubscribe(t *testing.T) {
	logger := logrus.New()
	eventBus := NewEventBus(logger)
	defer eventBus.Close()

	// 구독
	events := make(chan *StreamEvent, 1)
	subscription, err := eventBus.Subscribe("test", func(event *StreamEvent) error {
		events <- event
		return nil
	})
	assert.NoError(t, err)
	assert.NotNil(t, subscription)

	// 이벤트 발행
	testEvent := &StreamEvent{
		Type: "test",
		Data: "test data",
	}
	eventBus.Publish(testEvent)

	// 이벤트 수신 확인
	select {
	case receivedEvent := <-events:
		assert.Equal(t, "test", receivedEvent.Type)
		assert.Equal(t, "test data", receivedEvent.Data)
	case <-time.After(1 * time.Second):
		t.Fatal("Event not received")
	}

	// 구독 취소
	err = eventBus.Unsubscribe(subscription)
	assert.NoError(t, err)

	// 구독 취소 후 이벤트 발행
	eventBus.Publish(testEvent)

	// 이벤트가 수신되지 않아야 함
	select {
	case <-events:
		t.Fatal("Event received after unsubscribe")
	case <-time.After(100 * time.Millisecond):
		// 예상된 동작
	}
}

func TestEventBus_Metrics(t *testing.T) {
	logger := logrus.New()
	eventBus := NewEventBus(logger)
	defer eventBus.Close()

	// 구독
	subscription, err := eventBus.Subscribe("test", func(event *StreamEvent) error {
		return nil
	})
	require.NoError(t, err)

	// 이벤트 발행
	event := &StreamEvent{Type: "test", Data: "test"}
	eventBus.Publish(event)

	// 잠시 대기
	time.Sleep(100 * time.Millisecond)

	// 메트릭 확인
	metrics := eventBus.GetMetrics()
	assert.NotNil(t, metrics)
	assert.Equal(t, int64(1), metrics["published_events"].(int64))
	assert.Equal(t, 1, metrics["active_subscribers"].(int))

	// 구독 취소 후 메트릭 확인
	eventBus.Unsubscribe(subscription)
	metrics = eventBus.GetMetrics()
	assert.Equal(t, 0, metrics["active_subscribers"].(int))
}

func TestJSONStreamParser_ParseNext(t *testing.T) {
	logger := logrus.New()
	
	response := &Response{
		Type:      "test",
		Content:   "Hello!",
		MessageID: "123",
	}
	responseJSON, _ := json.Marshal(response)
	
	reader := bytes.NewReader(responseJSON)
	parser := NewJSONStreamParser(reader, logger)

	parsed, err := parser.ParseNext()
	assert.NoError(t, err)
	assert.NotNil(t, parsed)
	assert.Equal(t, response.Type, parsed.Type)
	assert.Equal(t, response.Content, parsed.Content)
	assert.Equal(t, response.MessageID, parsed.MessageID)
}

func TestJSONStreamParser_ParseStream(t *testing.T) {
	logger := logrus.New()
	
	responses := []*Response{
		{Type: "test1", Content: "Hello 1", MessageID: "1"},
		{Type: "test2", Content: "Hello 2", MessageID: "2"},
	}

	var jsonData []byte
	for _, resp := range responses {
		data, _ := json.Marshal(resp)
		jsonData = append(jsonData, data...)
	}

	reader := bytes.NewReader(jsonData)
	parser := NewJSONStreamParser(reader, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	responseChan, errorChan := parser.ParseStream(ctx)

	receivedCount := 0
	for {
		select {
		case response := <-responseChan:
			if response == nil {
				goto done
			}
			receivedCount++
			assert.Contains(t, []string{"test1", "test2"}, response.Type)
			
		case err := <-errorChan:
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			
		case <-ctx.Done():
			goto done
		}
	}

done:
	assert.Equal(t, len(responses), receivedCount)
}