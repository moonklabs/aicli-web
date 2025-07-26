package claude

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

// BenchmarkStreamHandler는 스트림 핸들러의 성능을 벤치마킹합니다.
func BenchmarkStreamHandler(b *testing.B) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	testCases := []struct {
		name        string
		messageSize int
		messageRate int // messages per second
	}{
		{"Small_100msg/s", 100, 100},
		{"Small_1000msg/s", 100, 1000},
		{"Medium_100msg/s", 1024, 100},
		{"Medium_1000msg/s", 1024, 1000},
		{"Large_100msg/s", 10240, 100},
		{"Large_1000msg/s", 10240, 1000},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			benchmarkStreamWithRate(b, tc.messageSize, tc.messageRate, logger)
		})
	}
}

func benchmarkStreamWithRate(b *testing.B, messageSize, messageRate int, logger *logrus.Logger) {
	handler := NewStreamHandler(logger)
	ctx := context.Background()

	// 테스트 메시지 생성
	content := strings.Repeat("a", messageSize)
	messages := make([]string, b.N)
	for i := 0; i < b.N; i++ {
		messages[i] = fmt.Sprintf(`{"type":"text","content":"%s","id":"msg-%d"}`, content, i)
	}

	// 스트림 리더 생성
	reader := newRateLimitedReader(messages, messageRate)

	b.ResetTimer()
	b.SetBytes(int64(messageSize))

	// 스트림 처리
	messageChan, err := handler.Stream(ctx, reader)
	if err != nil {
		b.Fatal(err)
	}

	// 메시지 소비
	count := 0
	for range messageChan {
		count++
		if count >= b.N {
			break
		}
	}

	b.StopTimer()

	// 메트릭 확인
	metrics := handler.GetMetrics()
	b.Logf("Messages: %d, Bytes: %d, Errors: %d, Backpressure: %d",
		metrics.MessagesReceived,
		metrics.BytesProcessed,
		metrics.ParseErrors,
		metrics.BackpressureEvents)
}

// BenchmarkBackpressure는 백프레셔 처리 성능을 벤치마킹합니다.
func BenchmarkBackpressure(b *testing.B) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	configs := []struct {
		name       string
		bufferSize int
		policy     DropPolicy
	}{
		{"DropOldest_100", 100, DropOldest},
		{"DropNewest_100", 100, DropNewest},
		{"BlockUntilReady_100", 100, BlockUntilReady},
		{"DropOldest_1000", 1000, DropOldest},
		{"DropNewest_1000", 1000, DropNewest},
		{"BlockUntilReady_1000", 1000, BlockUntilReady},
	}

	for _, cfg := range configs {
		b.Run(cfg.name, func(b *testing.B) {
			config := BackpressureConfig{
				MaxBufferSize:     cfg.bufferSize,
				DropPolicy:        cfg.policy,
				AdaptiveBuffering: false,
			}
			handler := NewBackpressureHandler(config, logger)

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				// 버퍼 증가
				handler.IncrementBuffer()

				// 드롭 체크
				if handler.ShouldDrop() {
					handler.HandleDrop([]interface{}{i})
				}

				// 버퍼 감소
				if i%2 == 0 {
					handler.DecrementBuffer()
				}
			}

			b.StopTimer()

			metrics := handler.GetMetrics()
			b.Logf("Dropped: %d, Events: %d, AvgUsage: %.2f",
				metrics.DroppedMessages,
				metrics.BackpressureEvents,
				metrics.AvgBufferUsage)
		})
	}
}

// BenchmarkStreamMessageRouter는 스트림 메시지 라우터 성능을 벤치마킹합니다.
func BenchmarkStreamMessageRouter(b *testing.B) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	testCases := []struct {
		name       string
		asyncMode  bool
		numHandlers int
	}{
		{"Sync_1Handler", false, 1},
		{"Sync_5Handlers", false, 5},
		{"Sync_10Handlers", false, 10},
		{"Async_1Handler", true, 1},
		{"Async_5Handlers", true, 5},
		{"Async_10Handlers", true, 10},
	}

	messageTypes := []MessageType{
		MessageTypeText,
		MessageTypeToolUse,
		MessageTypeError,
		MessageTypeSystem,
		MessageTypeMetadata,
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			config := RouterConfig{
				AsyncMode:      tc.asyncMode,
				MaxConcurrency: 10,
			}
			router := NewMessageRouter(config, logger)

			// 핸들러 등록
			for i := 0; i < tc.numHandlers; i++ {
				for _, msgType := range messageTypes {
					handler := &benchmarkHandler{
						name:     fmt.Sprintf("handler-%d", i),
						priority: i,
						delay:    time.Microsecond,
					}
					router.RegisterHandler(msgType, handler)
				}
			}

			ctx := context.Background()
			messages := generateTestMessages(b.N, messageTypes)

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				if err := router.Route(ctx, messages[i]); err != nil {
					b.Fatal(err)
				}
			}

			b.StopTimer()

			if tc.asyncMode {
				router.Stop()
			}

			metrics := router.GetMetrics()
			b.Logf("Total: %d, Errors: %d", 
				metrics["total_messages"], 
				metrics["total_errors"])
		})
	}
}

// BenchmarkJSONParser는 JSON 파서 성능을 벤치마킹합니다.
func BenchmarkJSONParser(b *testing.B) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	testCases := []struct {
		name      string
		jsonType  string
		multiline bool
		size      int
	}{
		{"SingleLine_Small", "simple", false, 100},
		{"SingleLine_Medium", "simple", false, 1024},
		{"SingleLine_Large", "simple", false, 10240},
		{"Multiline_Small", "complex", true, 100},
		{"Multiline_Medium", "complex", true, 1024},
		{"Multiline_Large", "complex", true, 10240},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			// 테스트 JSON 생성
			jsonData := generateJSONData(tc.jsonType, tc.multiline, tc.size)
			reader := strings.NewReader(strings.Repeat(jsonData+"\n", b.N))
			
			parser := NewJSONStreamParser(reader, logger)
			ctx := context.Background()

			b.ResetTimer()
			b.SetBytes(int64(len(jsonData)))

			responseChan, errorChan := parser.ParseStream(ctx)
			
			count := 0
			for {
				select {
				case resp := <-responseChan:
					if resp == nil {
						return
					}
					count++
					if count >= b.N {
						return
					}
				case err := <-errorChan:
					if err != nil && err != io.EOF {
						b.Fatal(err)
					}
					return
				}
			}
		})
	}
}

// 헬퍼 함수들

type rateLimitedReader struct {
	messages []string
	index    int
	rate     int
	lastRead time.Time
}

func newRateLimitedReader(messages []string, rate int) *rateLimitedReader {
	return &rateLimitedReader{
		messages: messages,
		rate:     rate,
		lastRead: time.Now(),
	}
}

func (r *rateLimitedReader) Read(p []byte) (n int, err error) {
	if r.index >= len(r.messages) {
		return 0, io.EOF
	}

	// Rate limiting
	elapsed := time.Since(r.lastRead)
	minInterval := time.Second / time.Duration(r.rate)
	if elapsed < minInterval {
		time.Sleep(minInterval - elapsed)
	}

	message := r.messages[r.index] + "\n"
	r.index++
	r.lastRead = time.Now()

	copy(p, []byte(message))
	return len(message), nil
}

type benchmarkHandler struct {
	name     string
	priority int
	delay    time.Duration
}

func (h *benchmarkHandler) Handle(ctx context.Context, msg StreamMessage) error {
	if h.delay > 0 {
		time.Sleep(h.delay)
	}
	return nil
}

func (h *benchmarkHandler) Priority() int {
	return h.priority
}

func (h *benchmarkHandler) Name() string {
	return h.name
}

func generateTestMessages(count int, types []MessageType) []StreamMessage {
	messages := make([]StreamMessage, count)
	for i := 0; i < count; i++ {
		msgType := types[i%len(types)]
		messages[i] = StreamMessage{
			Type:    string(msgType),
			Content: fmt.Sprintf("Test message %d", i),
			ID:      fmt.Sprintf("msg-%d", i),
			Meta: map[string]interface{}{
				"index": i,
				"timestamp": time.Now(),
			},
		}
	}
	return messages
}

func generateJSONData(jsonType string, multiline bool, size int) string {
	var builder bytes.Buffer
	
	if jsonType == "simple" {
		content := strings.Repeat("x", size)
		json := fmt.Sprintf(`{"type":"text","content":"%s","message_id":"test-123"}`, content)
		if multiline {
			// 멀티라인으로 변환
			json = strings.Replace(json, ",", ",\n  ", -1)
			json = strings.Replace(json, "{", "{\n  ", 1)
			json = strings.Replace(json, "}", "\n}", 1)
		}
		builder.WriteString(json)
	} else {
		// 복잡한 JSON
		builder.WriteString(`{`)
		if multiline {
			builder.WriteString("\n")
		}
		builder.WriteString(`"type":"complex",`)
		if multiline {
			builder.WriteString("\n")
		}
		builder.WriteString(`"metadata":{`)
		if multiline {
			builder.WriteString("\n")
		}
		
		// 메타데이터 추가
		for i := 0; i < 10; i++ {
			if i > 0 {
				builder.WriteString(",")
				if multiline {
					builder.WriteString("\n")
				}
			}
			builder.WriteString(fmt.Sprintf(`"field%d":"%s"`, i, strings.Repeat("v", size/20)))
		}
		
		if multiline {
			builder.WriteString("\n")
		}
		builder.WriteString("},")
		if multiline {
			builder.WriteString("\n")
		}
		builder.WriteString(`"content":"`)
		builder.WriteString(strings.Repeat("c", size/2))
		builder.WriteString(`"`)
		if multiline {
			builder.WriteString("\n")
		}
		builder.WriteString("}")
	}
	
	return builder.String()
}

// BenchmarkMemoryUsage는 메모리 사용량을 측정합니다.
func BenchmarkMemoryUsage(b *testing.B) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	b.Run("StreamHandler_1MB", func(b *testing.B) {
		benchmarkMemoryUsage(b, 1024*1024, logger)
	})

	b.Run("StreamHandler_10MB", func(b *testing.B) {
		benchmarkMemoryUsage(b, 10*1024*1024, logger)
	})

	b.Run("StreamHandler_100MB", func(b *testing.B) {
		benchmarkMemoryUsage(b, 100*1024*1024, logger)
	})
}

func benchmarkMemoryUsage(b *testing.B, totalBytes int, logger *logrus.Logger) {
	handler := NewStreamHandler(logger)
	ctx := context.Background()

	// 큰 메시지 생성
	messageSize := 1024 // 1KB per message
	numMessages := totalBytes / messageSize
	
	messages := make([]string, numMessages)
	content := strings.Repeat("x", messageSize-100) // JSON 오버헤드 고려
	for i := 0; i < numMessages; i++ {
		messages[i] = fmt.Sprintf(`{"type":"text","content":"%s","id":"msg-%d"}`, content, i)
	}

	reader := strings.NewReader(strings.Join(messages, "\n"))

	b.ResetTimer()
	b.ReportAllocs()

	messageChan, err := handler.Stream(ctx, reader)
	if err != nil {
		b.Fatal(err)
	}

	// 모든 메시지 소비
	count := 0
	for range messageChan {
		count++
		if count >= numMessages {
			break
		}
	}

	b.StopTimer()
	b.Logf("Processed %d messages", count)
}