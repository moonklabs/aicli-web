//go:build benchmark
// +build benchmark

package benchmark

import (
	"bytes"
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/aicli/aicli-web/internal/claude"
	"aicli-web/test/helpers"
)

// BenchmarkStreamProcessing는 스트림 처리 성능을 벤치마크합니다
func BenchmarkStreamProcessing(b *testing.B) {
	env := helpers.NewTestEnvironment(&testing.T{})
	defer env.Cleanup()
	
	// 다양한 크기의 데이터로 벤치마크
	sizes := []int{
		1024,      // 1KB
		10240,     // 10KB
		102400,    // 100KB
		1048576,   // 1MB
	}
	
	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size_%dB", size), func(b *testing.B) {
			data := env.TestData.GenerateLargeStreamData(size)
			b.SetBytes(int64(size))
			b.ResetTimer()
			
			for i := 0; i < b.N; i++ {
				reader := bytes.NewReader(data)
				handler := claude.NewStreamHandler()
				
				messages, err := handler.Stream(context.Background(), reader)
				require.NoError(b, err)
				
				// 메시지 소비
				for range messages {
					// 처리
				}
			}
		})
	}
}

// BenchmarkConcurrentStreamProcessing는 동시 스트림 처리 성능을 벤치마크합니다
func BenchmarkConcurrentStreamProcessing(b *testing.B) {
	env := helpers.NewTestEnvironment(&testing.T{})
	defer env.Cleanup()
	
	data := env.TestData.GenerateLargeStreamData(10240) // 10KB
	
	concurrencies := []int{1, 2, 4, 8, 16}
	
	for _, concurrency := range concurrencies {
		b.Run(fmt.Sprintf("Concurrency_%d", concurrency), func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			b.ResetTimer()
			
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					reader := bytes.NewReader(data)
					handler := claude.NewStreamHandler()
					
					messages, err := handler.Stream(context.Background(), reader)
					require.NoError(b, err)
					
					for range messages {
						// 처리
					}
				}
			})
		})
	}
}

// BenchmarkSessionManagement는 세션 관리 성능을 벤치마크합니다
func BenchmarkSessionManagement(b *testing.B) {
	env := helpers.NewTestEnvironment(&testing.T{})
	defer env.Cleanup()
	
	manager := claude.NewSessionManager(env.GetTestLogger())
	
	b.Run("세션생성", func(b *testing.B) {
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			config := &claude.SessionConfig{
				SystemPrompt: "Test assistant",
				MaxTurns:     5,
				Tools:        []string{"Write"},
			}
			
			session, err := manager.CreateSession(context.Background(), config)
			require.NoError(b, err)
			
			// 세션 정리
			manager.CloseSession(session.ID)
		}
	})
	
	b.Run("세션조회", func(b *testing.B) {
		// 테스트용 세션 미리 생성
		sessions := make([]*claude.Session, 100)
		for i := 0; i < 100; i++ {
			config := &claude.SessionConfig{
				SystemPrompt: fmt.Sprintf("Test assistant %d", i),
				MaxTurns:     5,
			}
			session, _ := manager.CreateSession(context.Background(), config)
			sessions[i] = session
		}
		
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			sessionID := sessions[i%len(sessions)].ID
			_, err := manager.GetSession(sessionID)
			require.NoError(b, err)
		}
		
		// 정리
		for _, session := range sessions {
			manager.CloseSession(session.ID)
		}
	})
	
	b.Run("동시세션관리", func(b *testing.B) {
		numSessions := 50
		sessions := make([]*claude.Session, numSessions)
		
		// 세션 풀 생성
		for i := 0; i < numSessions; i++ {
			config := &claude.SessionConfig{
				SystemPrompt: fmt.Sprintf("Pool session %d", i),
				MaxTurns:     10,
			}
			session, _ := manager.CreateSession(context.Background(), config)
			sessions[i] = session
		}
		
		b.ResetTimer()
		
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				// 랜덤 세션 선택 (단순한 방식)
				sessionIdx := runtime.NumGoroutine() % numSessions
				sessionID := sessions[sessionIdx].ID
				
				// 세션 상태 조회
				session, err := manager.GetSession(sessionID)
				require.NoError(b, err)
				require.NotNil(b, session)
			}
		})
		
		// 정리
		for _, session := range sessions {
			manager.CloseSession(session.ID)
		}
	})
}

// BenchmarkProcessManager는 프로세스 관리자 성능을 벤치마크합니다
func BenchmarkProcessManager(b *testing.B) {
	env := helpers.NewTestEnvironment(&testing.T{})
	defer env.Cleanup()
	
	b.Run("프로세스시작종료", func(b *testing.B) {
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			pm := claude.NewProcessManager(env.GetTestLogger())
			
			config := &claude.ProcessConfig{
				Command: "echo",
				Args:    []string{fmt.Sprintf("test %d", i)},
			}
			
			// 시작
			err := pm.Start(context.Background(), config)
			require.NoError(b, err)
			
			// 완료 대기
			err = pm.Wait()
			require.NoError(b, err)
		}
	})
	
	b.Run("다중프로세스", func(b *testing.B) {
		concurrencies := []int{2, 4, 8}
		
		for _, concurrency := range concurrencies {
			b.Run(fmt.Sprintf("Processes_%d", concurrency), func(b *testing.B) {
				b.ResetTimer()
				
				for i := 0; i < b.N; i++ {
					var wg sync.WaitGroup
					
					for j := 0; j < concurrency; j++ {
						wg.Add(1)
						go func(id int) {
							defer wg.Done()
							
							pm := claude.NewProcessManager(env.GetTestLogger())
							config := &claude.ProcessConfig{
								Command: "echo",
								Args:    []string{fmt.Sprintf("process %d", id)},
							}
							
							err := pm.Start(context.Background(), config)
							require.NoError(b, err)
							
							err = pm.Wait()
							require.NoError(b, err)
						}(j)
					}
					
					wg.Wait()
				}
			})
		}
	})
}

// BenchmarkMemoryUsage는 메모리 사용량을 벤치마크합니다
func BenchmarkMemoryUsage(b *testing.B) {
	env := helpers.NewTestEnvironment(&testing.T{})
	defer env.Cleanup()
	
	b.Run("스트림핸들러메모리", func(b *testing.B) {
		var m1, m2 runtime.MemStats
		
		// 시작 메모리 측정
		runtime.GC()
		runtime.ReadMemStats(&m1)
		
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			handler := claude.NewStreamHandler()
			data := env.TestData.GenerateLargeStreamData(1024 * 1024) // 1MB
			reader := bytes.NewReader(data)
			
			messages, err := handler.Stream(context.Background(), reader)
			require.NoError(b, err)
			
			for range messages {
				// 처리
			}
		}
		
		b.StopTimer()
		
		// 종료 메모리 측정
		runtime.GC()
		runtime.ReadMemStats(&m2)
		
		allocatedMB := float64(m2.TotalAlloc-m1.TotalAlloc) / 1024 / 1024
		b.Logf("총 할당된 메모리: %.2f MB", allocatedMB)
		b.Logf("작업당 메모리: %.2f KB", allocatedMB*1024/float64(b.N))
	})
	
	b.Run("세션관리자메모리", func(b *testing.B) {
		var m1, m2 runtime.MemStats
		
		manager := claude.NewSessionManager(env.GetTestLogger())
		
		// 시작 메모리 측정
		runtime.GC()
		runtime.ReadMemStats(&m1)
		
		b.ResetTimer()
		
		sessions := make([]*claude.Session, b.N)
		for i := 0; i < b.N; i++ {
			config := &claude.SessionConfig{
				SystemPrompt: fmt.Sprintf("Memory test session %d", i),
				MaxTurns:     5,
			}
			
			session, err := manager.CreateSession(context.Background(), config)
			require.NoError(b, err)
			sessions[i] = session
		}
		
		b.StopTimer()
		
		// 종료 메모리 측정
		runtime.GC()
		runtime.ReadMemStats(&m2)
		
		allocatedMB := float64(m2.TotalAlloc-m1.TotalAlloc) / 1024 / 1024
		b.Logf("총 할당된 메모리: %.2f MB", allocatedMB)
		b.Logf("세션당 메모리: %.2f KB", allocatedMB*1024/float64(b.N))
		
		// 정리
		for _, session := range sessions {
			manager.CloseSession(session.ID)
		}
	})
}

// BenchmarkNetworkLatency는 네트워크 지연 시뮬레이션 벤치마크입니다
func BenchmarkNetworkLatency(b *testing.B) {
	env := helpers.NewTestEnvironment(&testing.T{})
	defer env.Cleanup()
	
	// 다양한 지연 시간 시뮬레이션
	latencies := []time.Duration{
		0 * time.Millisecond,    // 로컬
		10 * time.Millisecond,   // LAN
		50 * time.Millisecond,   // 같은 지역
		100 * time.Millisecond,  // 다른 지역
		200 * time.Millisecond,  // 대륙간
	}
	
	for _, latency := range latencies {
		b.Run(fmt.Sprintf("Latency_%dms", latency.Milliseconds()), func(b *testing.B) {
			handler := claude.NewStreamHandler()
			
			b.ResetTimer()
			
			for i := 0; i < b.N; i++ {
				// 네트워크 지연 시뮬레이션
				time.Sleep(latency)
				
				// 작은 메시지 처리
				data := env.TestData.GenerateLargeStreamData(1024)
				reader := bytes.NewReader(data)
				
				messages, err := handler.Stream(context.Background(), reader)
				require.NoError(b, err)
				
				for range messages {
					// 처리
				}
			}
		})
	}
}

// BenchmarkResourceContention는 리소스 경합 상황을 벤치마크합니다
func BenchmarkResourceContention(b *testing.B) {
	env := helpers.NewTestEnvironment(&testing.T{})
	defer env.Cleanup()
	
	b.Run("동시스트림경합", func(b *testing.B) {
		numWorkers := runtime.NumCPU() * 2
		data := env.TestData.GenerateLargeStreamData(10240) // 10KB
		
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			var wg sync.WaitGroup
			results := make(chan int, numWorkers)
			
			for w := 0; w < numWorkers; w++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					
					handler := claude.NewStreamHandler()
					reader := bytes.NewReader(data)
					
					messages, err := handler.Stream(context.Background(), reader)
					require.NoError(b, err)
					
					count := 0
					for range messages {
						count++
					}
					results <- count
				}()
			}
			
			wg.Wait()
			close(results)
			
			// 결과 수집
			totalMessages := 0
			for count := range results {
				totalMessages += count
			}
			
			// 예상 메시지 수와 비교
			expectedTotal := numWorkers * (len(data) / 50) // 대략적 추정
			if totalMessages < expectedTotal/2 {
				b.Errorf("처리된 메시지가 너무 적음: %d < %d", totalMessages, expectedTotal/2)
			}
		}
	})
	
	b.Run("세션풀경합", func(b *testing.B) {
		manager := claude.NewSessionManager(env.GetTestLogger())
		poolSize := 10
		
		// 세션 풀 생성
		sessions := make([]*claude.Session, poolSize)
		for i := 0; i < poolSize; i++ {
			config := &claude.SessionConfig{
				SystemPrompt: fmt.Sprintf("Pool session %d", i),
				MaxTurns:     5,
			}
			session, _ := manager.CreateSession(context.Background(), config)
			sessions[i] = session
		}
		
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			numWorkers := 20 // 세션보다 많은 워커
			var wg sync.WaitGroup
			
			for w := 0; w < numWorkers; w++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					
					// 세션 획득 시도 (단순한 방식)
					sessionIdx := runtime.NumGoroutine() % poolSize
					sessionID := sessions[sessionIdx].ID
					session, err := manager.GetSession(sessionID)
					require.NoError(b, err)
					require.NotNil(b, session)
					
					// 짧은 작업 시뮬레이션
					time.Sleep(time.Microsecond * 100)
				}()
			}
			
			wg.Wait()
		}
		
		// 정리
		for _, session := range sessions {
			manager.CloseSession(session.ID)
		}
	})
}

// StressTest는 스트레스 테스트를 수행합니다
func TestStressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("스트레스 테스트는 짧은 테스트 모드에서 생략")
	}
	
	env := helpers.NewTestEnvironment(t)
	
	t.Run("장시간실행", func(t *testing.T) {
		duration := 5 * time.Minute
		ctx, cancel := context.WithTimeout(context.Background(), duration)
		defer cancel()
		
		handler := claude.NewStreamHandler()
		processed := 0
		errors := 0
		
		start := time.Now()
		for time.Now().Sub(start) < duration {
			select {
			case <-ctx.Done():
				goto done
			default:
			}
			
			data := env.TestData.GenerateLargeStreamData(1024)
			reader := bytes.NewReader(data)
			
			messages, err := handler.Stream(context.Background(), reader)
			if err != nil {
				errors++
				continue
			}
			
			for range messages {
				processed++
			}
			
			// CPU 부하 조절
			if processed%1000 == 0 {
				time.Sleep(time.Millisecond)
			}
		}
	done:
		
		elapsed := time.Since(start)
		throughput := float64(processed) / elapsed.Seconds()
		
		t.Logf("스트레스 테스트 결과:")
		t.Logf("- 실행 시간: %v", elapsed)
		t.Logf("- 처리된 메시지: %d", processed)
		t.Logf("- 에러 수: %d", errors)
		t.Logf("- 처리량: %.2f 메시지/초", throughput)
		
		// 성능 기준 검증
		require.Greater(t, processed, 10000, "충분한 메시지를 처리해야 함")
		require.Less(t, float64(errors)/float64(processed), 0.01, "에러율이 1% 미만이어야 함")
	})
	
	t.Run("메모리압박", func(t *testing.T) {
		var m1, m2 runtime.MemStats
		
		// 시작 메모리
		runtime.GC()
		runtime.ReadMemStats(&m1)
		
		// 대량 데이터 처리
		for i := 0; i < 1000; i++ {
			handler := claude.NewStreamHandler()
			data := env.TestData.GenerateLargeStreamData(1024 * 1024) // 1MB
			reader := bytes.NewReader(data)
			
			messages, err := handler.Stream(context.Background(), reader)
			require.NoError(t, err)
			
			for range messages {
				// 처리
			}
			
			// 주기적으로 GC 실행
			if i%100 == 0 {
				runtime.GC()
			}
		}
		
		// 종료 메모리
		runtime.GC()
		runtime.ReadMemStats(&m2)
		
		allocatedMB := float64(m2.TotalAlloc-m1.TotalAlloc) / 1024 / 1024
		currentMB := float64(m2.Alloc) / 1024 / 1024
		
		t.Logf("총 할당 메모리: %.2f MB", allocatedMB)
		t.Logf("현재 사용 메모리: %.2f MB", currentMB)
		
		// 메모리 누수 확인
		require.Less(t, currentMB, 100.0, "메모리 사용량이 100MB를 넘지 않아야 함")
	})
}