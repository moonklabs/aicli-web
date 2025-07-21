// +build integration

package claude

import (
	"bytes"
	"context"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/drumcap/aicli-web/internal/storage"
)

// BenchmarkSessionManagement는 세션 관리 성능을 벤치마크합니다.
func BenchmarkSessionManagement(b *testing.B) {
	store, err := storage.New()
	if err != nil {
		b.Fatal(err)
	}
	defer store.Close()

	sessionManager := NewSessionManager(store.Session())
	ctx := context.Background()

	b.Run("CreateSession", func(b *testing.B) {
		sessionIDs := make([]string, b.N)
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			sessionID, err := sessionManager.Create(ctx, &SessionConfig{
				WorkspaceID:  fmt.Sprintf("bench-workspace-%d", i),
				SystemPrompt: "Benchmark test assistant",
				MaxTurns:     10,
			})
			if err != nil {
				b.Fatal(err)
			}
			sessionIDs[i] = sessionID
		}
		b.StopTimer()

		// 정리
		for _, id := range sessionIDs {
			sessionManager.Close(ctx, id)
		}
	})

	b.Run("GetSession", func(b *testing.B) {
		// 벤치마크용 세션 생성
		sessionID, err := sessionManager.Create(ctx, &SessionConfig{
			WorkspaceID:  "bench-get-session",
			SystemPrompt: "Test",
			MaxTurns:     10,
		})
		if err != nil {
			b.Fatal(err)
		}
		defer sessionManager.Close(ctx, sessionID)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := sessionManager.Get(ctx, sessionID)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("ConcurrentAccess", func(b *testing.B) {
		// 벤치마크용 세션 생성
		sessionID, err := sessionManager.Create(ctx, &SessionConfig{
			WorkspaceID:  "bench-concurrent",
			SystemPrompt: "Test",
			MaxTurns:     10,
		})
		if err != nil {
			b.Fatal(err)
		}
		defer sessionManager.Close(ctx, sessionID)

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, err := sessionManager.Get(ctx, sessionID)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	})
}

// BenchmarkStreamProcessing은 스트림 처리 성능을 벤치마크합니다.
func BenchmarkStreamProcessing(b *testing.B) {
	b.Run("JSONParsing", func(b *testing.B) {
		// 1KB 메시지 생성
		messageData := strings.Repeat("x", 1000)
		streamData := fmt.Sprintf(`{"type":"text","content":"%s","id":"bench"}`, messageData)
		
		b.SetBytes(int64(len(streamData)))
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			reader := strings.NewReader(streamData)
			parser := NewJSONStreamParser(reader, nil)
			
			_, err := parser.ParseNext()
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("LargeStreamParsing", func(b *testing.B) {
		// 100개의 메시지가 있는 스트림 생성
		var streamBuilder strings.Builder
		for i := 0; i < 100; i++ {
			streamBuilder.WriteString(fmt.Sprintf(`{"type":"text","content":"Message %d","id":"msg%d"}`+"\n", i, i))
		}
		streamData := streamBuilder.String()
		
		b.SetBytes(int64(len(streamData)))
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			reader := strings.NewReader(streamData)
			parser := NewJSONStreamParser(reader, nil)
			
			messageCount := 0
			for {
				_, err := parser.ParseNext()
				if err != nil {
					if err.Error() == "EOF" {
						break
					}
					b.Fatal(err)
				}
				messageCount++
			}
			
			if messageCount != 100 {
				b.Fatalf("Expected 100 messages, got %d", messageCount)
			}
		}
	})

	b.Run("ConcurrentStreamProcessing", func(b *testing.B) {
		streamData := `{"type":"text","content":"Concurrent test","id":"concurrent"}`
		
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				reader := strings.NewReader(streamData)
				parser := NewJSONStreamParser(reader, nil)
				
				_, err := parser.ParseNext()
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	})
}

// BenchmarkBackpressureHandling은 백프레셔 처리 성능을 벤치마크합니다.
func BenchmarkBackpressureHandling(b *testing.B) {
	b.Run("DropOldest", func(b *testing.B) {
		handler := NewBackpressureHandler(1000, DropOldest)
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			msg := Message{
				Type:    "text",
				Content: fmt.Sprintf("Benchmark message %d", i),
				ID:      fmt.Sprintf("bench-%d", i),
			}
			handler.Submit(msg)
		}
	})

	b.Run("DropNewest", func(b *testing.B) {
		handler := NewBackpressureHandler(1000, DropNewest)
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			msg := Message{
				Type:    "text",
				Content: fmt.Sprintf("Benchmark message %d", i),
				ID:      fmt.Sprintf("bench-%d", i),
			}
			handler.Submit(msg)
		}
	})

	b.Run("BlockUntilReady", func(b *testing.B) {
		handler := NewBackpressureHandler(1000, BlockUntilReady)
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			msg := Message{
				Type:    "text",
				Content: fmt.Sprintf("Benchmark message %d", i),
				ID:      fmt.Sprintf("bench-%d", i),
			}
			handler.Submit(msg)
		}
	})
}

// BenchmarkMemoryUsage는 메모리 사용량을 측정합니다.
func BenchmarkMemoryUsage(b *testing.B) {
	b.Run("SessionCreationMemory", func(b *testing.B) {
		store, err := storage.New()
		if err != nil {
			b.Fatal(err)
		}
		defer store.Close()

		sessionManager := NewSessionManager(store.Session())
		ctx := context.Background()

		// 메모리 사용량 측정 시작
		var m1, m2 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m1)

		b.ResetTimer()
		sessionIDs := make([]string, b.N)
		for i := 0; i < b.N; i++ {
			sessionID, err := sessionManager.Create(ctx, &SessionConfig{
				WorkspaceID:  fmt.Sprintf("memory-test-%d", i),
				SystemPrompt: "Memory test",
				MaxTurns:     10,
			})
			if err != nil {
				b.Fatal(err)
			}
			sessionIDs[i] = sessionID
		}
		b.StopTimer()

		// 메모리 사용량 측정 종료
		runtime.GC()
		runtime.ReadMemStats(&m2)

		// 메모리 사용량 보고
		b.ReportMetric(float64(m2.Alloc-m1.Alloc)/float64(b.N), "bytes/session")

		// 정리
		for _, id := range sessionIDs {
			sessionManager.Close(ctx, id)
		}
	})

	b.Run("StreamParsingMemory", func(b *testing.B) {
		// 큰 메시지 생성 (10KB)
		largeContent := strings.Repeat("x", 10000)
		streamData := fmt.Sprintf(`{"type":"text","content":"%s","id":"large"}`, largeContent)

		var m1, m2 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m1)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			reader := strings.NewReader(streamData)
			parser := NewJSONStreamParser(reader, nil)
			_, err := parser.ParseNext()
			if err != nil {
				b.Fatal(err)
			}
		}
		b.StopTimer()

		runtime.GC()
		runtime.ReadMemStats(&m2)

		b.ReportMetric(float64(m2.Alloc-m1.Alloc)/float64(b.N), "bytes/parse")
	})
}

// TestStabilityUnderLoad는 부하 상황에서의 안정성을 테스트합니다.
func TestStabilityUnderLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stability tests in short mode")
	}

	t.Run("HighConcurrencyStability", func(t *testing.T) {
		testHighConcurrencyStability(t)
	})

	t.Run("LongRunningStability", func(t *testing.T) {
		testLongRunningStability(t)
	})

	t.Run("MemoryLeakTest", func(t *testing.T) {
		testMemoryLeaks(t)
	})
}

// testHighConcurrencyStability는 높은 동시성 상황에서의 안정성을 테스트합니다.
func testHighConcurrencyStability(t *testing.T) {
	store, err := storage.New()
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	sessionManager := NewSessionManager(store.Session())
	ctx := context.Background()

	const numWorkers = 100
	const operationsPerWorker = 10
	
	var wg sync.WaitGroup
	errors := make(chan error, numWorkers*operationsPerWorker)

	// 여러 고루틴에서 동시에 세션 생성/조회/종료
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			
			for j := 0; j < operationsPerWorker; j++ {
				// 세션 생성
				sessionID, err := sessionManager.Create(ctx, &SessionConfig{
					WorkspaceID:  fmt.Sprintf("stability-worker-%d-op-%d", workerID, j),
					SystemPrompt: "Stability test",
					MaxTurns:     5,
				})
				if err != nil {
					errors <- fmt.Errorf("worker %d op %d create: %w", workerID, j, err)
					continue
				}

				// 세션 조회
				_, err = sessionManager.Get(ctx, sessionID)
				if err != nil {
					errors <- fmt.Errorf("worker %d op %d get: %w", workerID, j, err)
					continue
				}

				// 세션 종료
				err = sessionManager.Close(ctx, sessionID)
				if err != nil {
					errors <- fmt.Errorf("worker %d op %d close: %w", workerID, j, err)
					continue
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// 에러 수집 및 확인
	var errorCount int
	for err := range errors {
		t.Logf("Error: %v", err)
		errorCount++
	}

	if errorCount > 0 {
		t.Errorf("Got %d errors during high concurrency test", errorCount)
	}
}

// testLongRunningStability는 장시간 실행 상황에서의 안정성을 테스트합니다.
func testLongRunningStability(t *testing.T) {
	store, err := storage.New()
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	sessionManager := NewSessionManager(store.Session())
	ctx := context.Background()

	// 30초 동안 지속적으로 세션 생성/종료
	duration := 30 * time.Second
	deadline := time.Now().Add(duration)
	
	var operationCount int
	var errorCount int

	for time.Now().Before(deadline) {
		sessionID, err := sessionManager.Create(ctx, &SessionConfig{
			WorkspaceID:  fmt.Sprintf("longrun-%d", operationCount),
			SystemPrompt: "Long running test",
			MaxTurns:     3,
		})
		operationCount++
		
		if err != nil {
			t.Logf("Create error: %v", err)
			errorCount++
			continue
		}

		_, err = sessionManager.Get(ctx, sessionID)
		if err != nil {
			t.Logf("Get error: %v", err)
			errorCount++
		}

		err = sessionManager.Close(ctx, sessionID)
		if err != nil {
			t.Logf("Close error: %v", err)
			errorCount++
		}

		// 약간의 지연
		time.Sleep(10 * time.Millisecond)
	}

	t.Logf("Long running test completed: %d operations, %d errors", operationCount, errorCount)
	
	errorRate := float64(errorCount) / float64(operationCount)
	if errorRate > 0.01 { // 1% 이상 에러가 발생하면 실패
		t.Errorf("Error rate too high: %.2f%% (%d/%d)", errorRate*100, errorCount, operationCount)
	}
}

// testMemoryLeaks는 메모리 누수를 테스트합니다.
func testMemoryLeaks(t *testing.T) {
	store, err := storage.New()
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	sessionManager := NewSessionManager(store.Session())
	ctx := context.Background()

	// 초기 메모리 상태 측정
	var m1, m2, m3 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	// 많은 세션 생성 및 종료
	const numSessions = 1000
	for i := 0; i < numSessions; i++ {
		sessionID, err := sessionManager.Create(ctx, &SessionConfig{
			WorkspaceID:  fmt.Sprintf("memleak-test-%d", i),
			SystemPrompt: "Memory leak test",
			MaxTurns:     5,
		})
		if err != nil {
			t.Fatal(err)
		}

		err = sessionManager.Close(ctx, sessionID)
		if err != nil {
			t.Fatal(err)
		}
	}

	// 중간 메모리 상태 측정
	runtime.GC()
	runtime.ReadMemStats(&m2)

	// 추가로 같은 작업 반복
	for i := 0; i < numSessions; i++ {
		sessionID, err := sessionManager.Create(ctx, &SessionConfig{
			WorkspaceID:  fmt.Sprintf("memleak-test2-%d", i),
			SystemPrompt: "Memory leak test 2",
			MaxTurns:     5,
		})
		if err != nil {
			t.Fatal(err)
		}

		err = sessionManager.Close(ctx, sessionID)
		if err != nil {
			t.Fatal(err)
		}
	}

	// 최종 메모리 상태 측정
	runtime.GC()
	runtime.ReadMemStats(&m3)

	// 메모리 사용량 분석
	firstRoundAlloc := m2.Alloc - m1.Alloc
	secondRoundAlloc := m3.Alloc - m2.Alloc

	t.Logf("First round memory increase: %d bytes", firstRoundAlloc)
	t.Logf("Second round memory increase: %d bytes", secondRoundAlloc)

	// 두 번째 라운드의 메모리 증가가 첫 번째보다 현저히 작아야 함 (메모리 누수가 없다면)
	if secondRoundAlloc > firstRoundAlloc*2 {
		t.Errorf("Potential memory leak detected: second round used %d bytes, first round used %d bytes", 
			secondRoundAlloc, firstRoundAlloc)
	}
}