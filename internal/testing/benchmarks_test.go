package testing

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"

	"github.com/aicli/aicli-web/internal/testing/test_helpers"
)

// BenchmarkSessionThroughput 세션 처리량 벤치마크
func BenchmarkSessionThroughput(b *testing.B) {
	env, err := test_helpers.NewAdvancedTestEnv(&testing.T{}, nil)
	require.NoError(b, err)
	defer env.Cleanup()
	
	b.ResetTimer()
	b.SetParallelism(10) // 10개 병렬 고루틴
	
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			session, err := env.CreateTestSession()
			if err != nil {
				b.Errorf("Failed to create session: %v", err)
				continue
			}
			
			// 세션 즉시 종료
			env.SessionManager.CloseSession(context.Background(), session.ID)
		}
	})
	
	// 성능 기준 검증
	if b.N > 0 {
		sessionPerSecond := float64(b.N) / b.Elapsed().Seconds()
		b.Logf("Sessions per second: %.2f", sessionPerSecond)
		
		// 목표: 초당 100개 이상의 세션 처리
		if sessionPerSecond < 100 {
			b.Errorf("Session throughput too low: %.2f sessions/sec (target: 100)", sessionPerSecond)
		}
	}
}

// BenchmarkMessageProcessing 메시지 처리 벤치마크
func BenchmarkMessageProcessing(b *testing.B) {
	env, err := test_helpers.NewAdvancedTestEnv(&testing.T{}, nil)
	require.NoError(b, err)
	defer env.Cleanup()
	
	// 고정 세션 생성
	session, err := env.CreateTestSession()
	require.NoError(b, err)
	defer env.SessionManager.CloseSession(context.Background(), session.ID)
	
	// WebSocket 연결
	conn, err := env.CreateWebSocketConnection(session.ID)
	require.NoError(b, err)
	defer env.CloseWebSocketConnection(session.ID)
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		message := "Benchmark test message"
		
		// 메시지 전송
		err := conn.WriteMessage(websocket.TextMessage, []byte(message))
		if err != nil {
			b.Errorf("Failed to send message: %v", err)
			continue
		}
		
		// 응답 수신
		_, _, err = conn.ReadMessage()
		if err != nil {
			b.Errorf("Failed to receive response: %v", err)
			continue
		}
	}
	
	if b.N > 0 {
		messagesPerSecond := float64(b.N) / b.Elapsed().Seconds()
		b.Logf("Messages per second: %.2f", messagesPerSecond)
		
		// 목표: 초당 1000개 이상의 메시지 처리
		if messagesPerSecond < 1000 {
			b.Errorf("Message throughput too low: %.2f messages/sec (target: 1000)", messagesPerSecond)
		}
	}
}

// BenchmarkResponseLatency 응답 지연시간 벤치마크
func BenchmarkResponseLatency(b *testing.B) {
	env, err := test_helpers.NewAdvancedTestEnv(&testing.T{}, nil)
	require.NoError(b, err)
	defer env.Cleanup()
	
	session, err := env.CreateTestSession()
	require.NoError(b, err)
	defer env.SessionManager.CloseSession(context.Background(), session.ID)
	
	conn, err := env.CreateWebSocketConnection(session.ID)
	require.NoError(b, err)
	defer env.CloseWebSocketConnection(session.ID)
	
	var totalLatency time.Duration
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		start := time.Now()
		
		// 메시지 전송
		err := conn.WriteMessage(websocket.TextMessage, []byte("latency test"))
		if err != nil {
			b.Errorf("Failed to send message: %v", err)
			continue
		}
		
		// 응답 수신
		_, _, err = conn.ReadMessage()
		if err != nil {
			b.Errorf("Failed to receive response: %v", err)
			continue
		}
		
		latency := time.Since(start)
		totalLatency += latency
	}
	
	if b.N > 0 {
		avgLatency := totalLatency / time.Duration(b.N)
		b.Logf("Average latency: %v", avgLatency)
		
		// 목표: 평균 100ms 이하
		if avgLatency > 100*time.Millisecond {
			b.Errorf("Average latency too high: %v (target: 100ms)", avgLatency)
		}
	}
}

// BenchmarkWebSocketLatency WebSocket 지연시간 벤치마크
func BenchmarkWebSocketLatency(b *testing.B) {
	env, err := test_helpers.NewAdvancedTestEnv(&testing.T{}, nil)
	require.NoError(b, err)
	defer env.Cleanup()
	
	// 여러 연결로 테스트
	connectionCount := 10
	connections := make([]*websocket.Conn, connectionCount)
	
	for i := 0; i < connectionCount; i++ {
		sessionID := fmt.Sprintf("benchmark-session-%d", i)
		conn, err := env.CreateWebSocketConnection(sessionID)
		require.NoError(b, err)
		connections[i] = conn
		defer env.CloseWebSocketConnection(sessionID)
	}
	
	b.ResetTimer()
	
	b.RunParallel(func(pb *testing.PB) {
		connIndex := 0
		for pb.Next() {
			conn := connections[connIndex%connectionCount]
			connIndex++
			
			start := time.Now()
			
			// 메시지 전송
			err := conn.WriteMessage(websocket.TextMessage, []byte("ws latency test"))
			if err != nil {
				b.Errorf("Failed to send WebSocket message: %v", err)
				continue
			}
			
			// 응답 수신
			_, _, err = conn.ReadMessage()
			if err != nil {
				b.Errorf("Failed to receive WebSocket response: %v", err)
				continue
			}
			
			latency := time.Since(start)
			
			// 목표: 50ms 이하
			if latency > 50*time.Millisecond {
				b.Errorf("WebSocket latency too high: %v (target: 50ms)", latency)
			}
		}
	})
}

// BenchmarkMemoryUsage 메모리 사용량 벤치마크
func BenchmarkMemoryUsage(b *testing.B) {
	env, err := test_helpers.NewAdvancedTestEnv(&testing.T{}, nil)
	require.NoError(b, err)
	defer env.Cleanup()
	
	// 메모리 모니터링 시작
	metricsStop := env.MetricsCollector.StartMemoryMonitoring(10 * time.Millisecond)
	defer close(metricsStop)
	
	// 초기 메모리 사용량
	env.MetricsCollector.TakeMemorySnapshot()
	initialMetrics := env.MetricsCollector.GetMetrics()
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		session, err := env.CreateTestSession()
		if err != nil {
			b.Errorf("Failed to create session: %v", err)
			continue
		}
		
		// 간단한 작업 수행
		conn, err := env.CreateWebSocketConnection(session.ID)
		if err != nil {
			b.Errorf("Failed to create WebSocket connection: %v", err)
			env.SessionManager.CloseSession(context.Background(), session.ID)
			continue
		}
		
		// 메시지 교환
		conn.WriteMessage(websocket.TextMessage, []byte("memory test"))
		conn.ReadMessage()
		
		// 정리
		env.CloseWebSocketConnection(session.ID)
		env.SessionManager.CloseSession(context.Background(), session.ID)
		
		// 주기적으로 메모리 스냅샷
		if i%100 == 0 {
			env.MetricsCollector.TakeMemorySnapshot()
		}
	}
	
	// 최종 메모리 사용량
	env.MetricsCollector.TakeMemorySnapshot()
	finalMetrics := env.MetricsCollector.GetMetrics()
	
	memoryIncrease := finalMetrics.AverageMemoryUsage - initialMetrics.AverageMemoryUsage
	memoryPerOperation := float64(memoryIncrease) / float64(b.N)
	
	b.Logf("Memory increase: %d bytes, Per operation: %.2f bytes", memoryIncrease, memoryPerOperation)
	
	// 목표: 이전 대비 30% 감소 (여기서는 절대값으로 확인)
	if memoryPerOperation > 1024 { // 1KB per operation
		b.Errorf("Memory usage per operation too high: %.2f bytes (target: <1024)", memoryPerOperation)
	}
}

// BenchmarkGoroutineCount 고루틴 수 벤치마크
func BenchmarkGoroutineCount(b *testing.B) {
	env, err := test_helpers.NewAdvancedTestEnv(&testing.T{}, nil)
	require.NoError(b, err)
	defer env.Cleanup()
	
	initialGoroutines := runtime.NumGoroutine()
	
	b.ResetTimer()
	
	var wg sync.WaitGroup
	
	for i := 0; i < b.N; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			session, err := env.CreateTestSession()
			if err != nil {
				return
			}
			
			// 작업 시뮬레이션
			time.Sleep(1 * time.Millisecond)
			
			env.SessionManager.CloseSession(context.Background(), session.ID)
		}()
	}
	
	peakGoroutines := runtime.NumGoroutine()
	wg.Wait()
	
	time.Sleep(100 * time.Millisecond) // 정리 대기
	finalGoroutines := runtime.NumGoroutine()
	
	goroutineIncrease := finalGoroutines - initialGoroutines
	
	b.Logf("Goroutines - Initial: %d, Peak: %d, Final: %d, Increase: %d", 
		initialGoroutines, peakGoroutines, finalGoroutines, goroutineIncrease)
	
	// 목표: 고루틴 수 증가가 제한적이어야 함
	if goroutineIncrease > 10 {
		b.Errorf("Too many goroutines leaked: %d (target: <10)", goroutineIncrease)
	}
}

// BenchmarkConcurrentSessions 동시 세션 벤치마크
func BenchmarkConcurrentSessions(b *testing.B) {
	config := test_helpers.DefaultTestConfig()
	config.MaxConcurrentSessions = 200
	
	env, err := test_helpers.NewAdvancedTestEnv(&testing.T{}, config)
	require.NoError(b, err)
	defer env.Cleanup()
	
	b.ResetTimer()
	
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var wg sync.WaitGroup
			sessionCount := 50 // 동시에 50개 세션
			
			for i := 0; i < sessionCount; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					
					session, err := env.CreateTestSession()
					if err != nil {
						return
					}
					
					// 짧은 작업
					time.Sleep(10 * time.Millisecond)
					
					env.SessionManager.CloseSession(context.Background(), session.ID)
				}()
			}
			
			wg.Wait()
		}
	})
	
	if b.N > 0 {
		totalSessions := float64(b.N * 50) // 각 반복마다 50개 세션
		sessionsPerSecond := totalSessions / b.Elapsed().Seconds()
		b.Logf("Concurrent sessions per second: %.2f", sessionsPerSecond)
	}
}

// BenchmarkSystemLoad 시스템 부하 벤치마크
func BenchmarkSystemLoad(b *testing.B) {
	config := test_helpers.DefaultTestConfig()
	config.MaxConcurrentSessions = 100
	config.MessageRate = 100
	
	env, err := test_helpers.NewAdvancedTestEnv(&testing.T{}, config)
	require.NoError(b, err)
	defer env.Cleanup()
	
	// 성능 추적 시작
	env.PerformanceTracker.Start()
	defer env.PerformanceTracker.Stop()
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		// 고부하 시뮬레이션 (작은 규모)
		err := env.SimulateHighLoad(10, 5) // 10개 세션, 각각 5개 메시지
		if err != nil {
			b.Errorf("High load simulation failed: %v", err)
		}
	}
	
	// 성능 검증
	err = env.ValidatePerformance()
	if err != nil {
		b.Errorf("Performance validation failed: %v", err)
	}
	
	// 리소스 사용량 검증
	err = env.ValidateResourceUsage()
	if err != nil {
		b.Errorf("Resource usage validation failed: %v", err)
	}
	
	// 성능 요약
	summary := env.PerformanceTracker.GetSummary()
	b.Logf("System Load Benchmark - Duration: %v, Peak Memory: %d, Peak Goroutines: %d", 
		summary.Duration, summary.PeakMemoryUsage, summary.PeakGoroutines)
}