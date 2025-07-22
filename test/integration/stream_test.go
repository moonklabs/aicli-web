//go:build integration
// +build integration

package integration

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/aicli/aicli-web/internal/claude"
	"aicli-web/test/helpers"
)

// TestStreamProcessingIntegration은 스트림 처리 통합 테스트를 수행합니다
func TestStreamProcessingIntegration(t *testing.T) {
	env := helpers.NewTestEnvironment(t)
	
	t.Run("JSON 스트림 파싱", func(t *testing.T) {
		// 테스트 데이터 준비
		testStream := env.TestData.LoadStreamData("complex_response.jsonl")
		reader := bytes.NewReader(testStream)
		
		// 스트림 핸들러 생성
		handler := claude.NewStreamHandler()
		ctx := context.Background()
		
		// 스트림 처리
		messages, err := handler.Stream(ctx, reader)
		require.NoError(t, err)
		
		// 메시지 수집
		var collected []claude.Message
		timeout := time.After(10 * time.Second)
		
		for {
			select {
			case msg, ok := <-messages:
				if !ok {
					goto done // 채널 닫힘
				}
				collected = append(collected, msg)
			case <-timeout:
				t.Fatal("스트림 처리 타임아웃")
			}
		}
	done:
		
		// 검증
		assert.Greater(t, len(collected), 0, "메시지가 처리되어야 함")
		
		// 메시지 타입 순서 확인
		expectedTypes := []string{"text", "tool_use", "text", "tool_use", "text", "completion"}
		require.Len(t, collected, len(expectedTypes), "예상된 메시지 수와 일치해야 함")
		
		for i, msg := range collected {
			assert.Equal(t, expectedTypes[i], msg.Type, "메시지 타입이 예상과 다름: 인덱스 %d", i)
		}
		
		// tool_use 메시지 확인
		toolUseCount := 0
		for _, msg := range collected {
			if msg.Type == "tool_use" {
				toolUseCount++
				assert.NotEmpty(t, msg.ToolName, "tool_use 메시지는 도구 이름을 가져야 함")
			}
		}
		assert.Equal(t, 2, toolUseCount, "2개의 tool_use 메시지가 있어야 함")
	})
	
	t.Run("대용량 스트림 처리", func(t *testing.T) {
		// 1MB 크기의 테스트 데이터 생성
		largeData := env.TestData.GenerateLargeStreamData(1024 * 1024)
		reader := bytes.NewReader(largeData)
		
		handler := claude.NewStreamHandler()
		ctx := context.Background()
		
		start := time.Now()
		messages, err := handler.Stream(ctx, reader)
		require.NoError(t, err)
		
		// 메시지 카운팅
		messageCount := 0
		for range messages {
			messageCount++
		}
		
		duration := time.Since(start)
		
		// 성능 검증
		assert.Greater(t, messageCount, 1000, "충분한 메시지가 처리되어야 함")
		assert.Less(t, duration, 5*time.Second, "대용량 스트림 처리가 5초 이내에 완료되어야 함")
		
		// 처리량 로깅
		throughput := float64(len(largeData)) / duration.Seconds() / 1024 / 1024 // MB/s
		t.Logf("스트림 처리 성능: %.2f MB/s", throughput)
	})
	
	t.Run("스트림 에러 처리", func(t *testing.T) {
		// 잘못된 JSON 데이터
		invalidJSON := `{"type":"text","content":"valid"}
{"type":"invalid_json"content":"missing comma"}
{"type":"text","content":"another valid"}`
		
		reader := bytes.NewReader([]byte(invalidJSON))
		handler := claude.NewStreamHandler()
		ctx := context.Background()
		
		messages, err := handler.Stream(ctx, reader)
		require.NoError(t, err)
		
		// 유효한 메시지만 수집
		var validMessages []claude.Message
		for msg := range messages {
			validMessages = append(validMessages, msg)
		}
		
		// 유효한 메시지만 처리되었는지 확인
		assert.Len(t, validMessages, 2, "유효한 메시지만 처리되어야 함")
		
		// 에러 메트릭 확인
		metrics := handler.GetMetrics()
		assert.Greater(t, metrics.ErrorCount, int64(0), "에러가 기록되어야 함")
	})
}

// TestStreamBackpressure는 백프레셔 처리 테스트를 수행합니다
func TestStreamBackpressure(t *testing.T) {
	env := helpers.NewTestEnvironment(t)
	
	t.Run("백프레셔 처리", func(t *testing.T) {
		// 빠른 생산자, 느린 소비자 시뮬레이션
		producer := make(chan []byte, 1000)
		
		// 대량 데이터 생성
		go func() {
			defer close(producer)
			for i := 0; i < 10000; i++ {
				msg := fmt.Sprintf(`{"type":"text","content":"Message %d"}`, i)
				producer <- []byte(msg)
			}
		}()
		
		// 스트림 처리 (느린 소비)
		handler := claude.NewStreamHandler()
		handler.SetBufferSize(100) // 작은 버퍼
		
		processed := 0
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		messages, err := handler.StreamFromChannel(ctx, producer)
		require.NoError(t, err)
		
		for range messages {
			processed++
			time.Sleep(time.Millisecond) // 인위적 지연
			
			// 타임아웃 체크
			select {
			case <-ctx.Done():
				goto done
			default:
			}
		}
	done:
		
		// 백프레셔로 인한 드롭 확인
		assert.Less(t, processed, 10000, "백프레셔로 인해 모든 메시지가 처리되지 않아야 함")
		
		metrics := handler.GetMetrics()
		assert.Greater(t, metrics.BackpressureEvents, int64(0), "백프레셔 이벤트가 발생해야 함")
		
		t.Logf("처리된 메시지: %d/10000, 백프레셔 이벤트: %d", processed, metrics.BackpressureEvents)
	})
	
	t.Run("버퍼 크기별 성능", func(t *testing.T) {
		bufferSizes := []int{50, 100, 500, 1000}
		
		for _, bufferSize := range bufferSizes {
			t.Run(fmt.Sprintf("버퍼크기_%d", bufferSize), func(t *testing.T) {
				// 테스트 데이터 준비
				testData := env.TestData.GenerateLargeStreamData(100 * 1024) // 100KB
				reader := bytes.NewReader(testData)
				
				// 스트림 핸들러 설정
				handler := claude.NewStreamHandler()
				handler.SetBufferSize(bufferSize)
				
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				
				start := time.Now()
				messages, err := handler.Stream(ctx, reader)
				require.NoError(t, err)
				
				// 메시지 소비
				messageCount := 0
				for range messages {
					messageCount++
				}
				
				duration := time.Since(start)
				metrics := handler.GetMetrics()
				
				t.Logf("버퍼크기 %d: %d 메시지, %v 소요, 백프레셔 %d회", 
					bufferSize, messageCount, duration, metrics.BackpressureEvents)
				
				// 기본 검증
				assert.Greater(t, messageCount, 0, "메시지가 처리되어야 함")
				assert.Less(t, duration, 5*time.Second, "처리 시간이 합리적이어야 함")
			})
		}
	})
}

// TestStreamConcurrency는 스트림 동시성 테스트를 수행합니다
func TestStreamConcurrency(t *testing.T) {
	env := helpers.NewTestEnvironment(t)
	
	t.Run("동시 스트림 처리", func(t *testing.T) {
		const numStreams = 10
		var wg sync.WaitGroup
		results := make(chan int, numStreams)
		
		// 테스트 데이터
		testData := env.TestData.GenerateLargeStreamData(50 * 1024) // 50KB per stream
		
		ctx := context.Background()
		
		// 여러 스트림 동시 처리
		for i := 0; i < numStreams; i++ {
			wg.Add(1)
			go func(streamID int) {
				defer wg.Done()
				
				reader := bytes.NewReader(testData)
				handler := claude.NewStreamHandler()
				
				messages, err := handler.Stream(ctx, reader)
				if err != nil {
					t.Errorf("스트림 %d 처리 실패: %v", streamID, err)
					results <- 0
					return
				}
				
				count := 0
				for range messages {
					count++
				}
				
				results <- count
			}(i)
		}
		
		wg.Wait()
		close(results)
		
		// 결과 검증
		totalMessages := 0
		streamCount := 0
		for count := range results {
			totalMessages += count
			streamCount++
		}
		
		assert.Equal(t, numStreams, streamCount, "모든 스트림이 처리되어야 함")
		assert.Greater(t, totalMessages, numStreams*10, "충분한 메시지가 처리되어야 함")
		
		t.Logf("동시 스트림 처리: %d개 스트림, 총 %d개 메시지", streamCount, totalMessages)
	})
	
	t.Run("스트림 취소 처리", func(t *testing.T) {
		// 장시간 실행되는 스트림 생성
		producer := make(chan []byte)
		
		go func() {
			defer close(producer)
			for i := 0; i < 1000000; i++ { // 매우 많은 메시지
				msg := fmt.Sprintf(`{"type":"text","content":"Message %d"}`, i)
				producer <- []byte(msg)
				time.Sleep(time.Microsecond) // 느린 생성
			}
		}()
		
		handler := claude.NewStreamHandler()
		
		// 짧은 타임아웃으로 컨텍스트 생성
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		
		messages, err := handler.StreamFromChannel(ctx, producer)
		require.NoError(t, err)
		
		start := time.Now()
		processed := 0
		
		for range messages {
			processed++
			
			// 컨텍스트 취소 확인
			if ctx.Err() != nil {
				break
			}
		}
		
		duration := time.Since(start)
		
		// 취소가 적절한 시간 내에 처리되었는지 확인
		assert.Less(t, duration, 2*time.Second, "취소가 빠르게 처리되어야 함")
		assert.Greater(t, processed, 0, "일부 메시지는 처리되어야 함")
		assert.Less(t, processed, 1000000, "모든 메시지가 처리되지 않아야 함")
		
		t.Logf("취소된 스트림: %d개 메시지 처리, %v 소요", processed, duration)
	})
}

// TestStreamResilience는 스트림 복원력 테스트를 수행합니다
func TestStreamResilience(t *testing.T) {
	env := helpers.NewTestEnvironment(t)
	
	t.Run("부분적 실패 복구", func(t *testing.T) {
		// 중간에 오류가 있는 스트림 데이터
		mixedData := `{"type":"text","content":"Message 1"}
{"type":"text","content":"Message 2"}
invalid json line
{"type":"text","content":"Message 3"}
{"invalid":"json","missing":"fields"}
{"type":"text","content":"Message 4"}
{"type":"completion","final":true}`
		
		reader := bytes.NewReader([]byte(mixedData))
		handler := claude.NewStreamHandler()
		
		// 에러 허용 모드 설정
		handler.SetErrorTolerance(true)
		
		ctx := context.Background()
		messages, err := handler.Stream(ctx, reader)
		require.NoError(t, err)
		
		// 유효한 메시지 수집
		var validMessages []claude.Message
		for msg := range messages {
			validMessages = append(validMessages, msg)
		}
		
		// 유효한 메시지만 처리되었는지 확인
		expectedValid := 5 // 4개 text + 1개 completion
		assert.Len(t, validMessages, expectedValid, "유효한 메시지만 처리되어야 함")
		
		// 에러 통계 확인
		metrics := handler.GetMetrics()
		assert.Greater(t, metrics.ErrorCount, int64(0), "에러가 기록되어야 함")
		assert.Greater(t, metrics.ProcessedCount, int64(0), "처리된 메시지가 기록되어야 함")
	})
	
	t.Run("메모리 제한 처리", func(t *testing.T) {
		if testing.Short() {
			t.Skip("짧은 테스트 모드에서 메모리 테스트 생략")
		}
		
		// 매우 큰 메시지들 생성
		var largeMessages []string
		largeContent := string(make([]byte, 1024*1024)) // 1MB per message
		
		for i := 0; i < 100; i++ {
			msg := fmt.Sprintf(`{"type":"text","content":"%s"}`, largeContent)
			largeMessages = append(largeMessages, msg)
		}
		
		data := []byte(fmt.Sprintf("%s\n", largeMessages[0:10])) // 처음 10개만 테스트
		reader := bytes.NewReader(data)
		
		handler := claude.NewStreamHandler()
		handler.SetMaxMessageSize(2 * 1024 * 1024) // 2MB 제한
		
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		
		messages, err := handler.Stream(ctx, reader)
		require.NoError(t, err)
		
		processed := 0
		for range messages {
			processed++
		}
		
		// 메모리 제한 하에서도 처리가 완료되어야 함
		assert.Greater(t, processed, 0, "메시지가 처리되어야 함")
		
		metrics := handler.GetMetrics()
		t.Logf("대용량 메시지 처리: %d개 처리, 에러 %d개", processed, metrics.ErrorCount)
	})
}