package testing

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aicli/aicli-web/internal/claude"
	"github.com/aicli/aicli-web/internal/testing/test_helpers"
)

// TestAdvancedIntegrationSuite 고급 통합 테스트 스위트
func TestAdvancedIntegrationSuite(t *testing.T) {
	t.Run("SessionPoolIntegration", TestSessionPoolIntegration)
	t.Run("WebSocketIntegration", TestWebSocketIntegration)
	t.Run("ErrorRecoveryIntegration", TestErrorRecoveryIntegration)
	t.Run("PerformanceOptimization", TestPerformanceOptimization)
	t.Run("E2EWorkflow", TestE2EWorkflow)
	t.Run("HighLoadScenario", TestHighLoadScenario)
	t.Run("ChaosEngineering", TestChaosEngineering)
}

// TestSessionPoolIntegration 세션 풀 통합 테스트
func TestSessionPoolIntegration(t *testing.T) {
	config := test_helpers.DefaultTestConfig()
	config.MaxConcurrentSessions = 50
	
	env, err := test_helpers.NewAdvancedTestEnv(t, config)
	require.NoError(t, err)
	defer env.Cleanup()
	
	t.Run("DynamicScaling", func(t *testing.T) {
		testSessionPoolDynamicScaling(t, env)
	})
	
	t.Run("LoadBalancing", func(t *testing.T) {
		testSessionPoolLoadBalancing(t, env)
	})
	
	t.Run("SessionReuse", func(t *testing.T) {
		testSessionPoolSessionReuse(t, env)
	})
	
	t.Run("ResourceLimits", func(t *testing.T) {
		testSessionPoolResourceLimits(t, env)
	})
}

// testSessionPoolDynamicScaling 세션 풀 동적 스케일링 테스트
func testSessionPoolDynamicScaling(t *testing.T, env *test_helpers.AdvancedTestEnvironment) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	// 메트릭 수집 시작
	metricsStop := env.MetricsCollector.StartMemoryMonitoring(100 * time.Millisecond)
	defer close(metricsStop)
	
	// 초기 세션 수 확인
	initialStats := env.SessionPool.GetStatistics()
	t.Logf("Initial pool size: %d", initialStats.ActiveSessions)
	
	// 부하 증가 시뮬레이션
	var wg sync.WaitGroup
	sessionCount := 20
	
	for i := 0; i < sessionCount; i++ {
		wg.Add(1)
		go func(sessionIndex int) {
			defer wg.Done()
			
			session, err := env.CreateTestSession()
			if err != nil {
				t.Errorf("Failed to create session %d: %v", sessionIndex, err)
				return
			}
			
			// 세션 사용 시뮬레이션
			select {
			case <-ctx.Done():
				return
			case <-time.After(1 * time.Second):
				// 세션 종료
				env.SessionManager.CloseSession(ctx, session.ID)
			}
		}(i)
	}
	
	wg.Wait()
	
	// 최종 통계 확인
	finalStats := env.SessionPool.GetStatistics()
	t.Logf("Final pool size: %d", finalStats.ActiveSessions)
	
	// 검증: 풀이 동적으로 조정되었는지 확인
	assert.True(t, finalStats.PeakSessions >= sessionCount/2, "Pool should have scaled up during load")
	
	// 성능 검증
	metrics := env.MetricsCollector.GetMetrics()
	assert.Less(t, metrics.AverageLatency, 200*time.Millisecond, "Average latency should be acceptable")
}

// testSessionPoolLoadBalancing 세션 풀 로드 밸런싱 테스트
func testSessionPoolLoadBalancing(t *testing.T, env *test_helpers.AdvancedTestEnvironment) {
	// 여러 로드 밸런싱 전략 테스트
	strategies := []claude.LoadBalancingStrategy{
		claude.LoadBalancingRoundRobin,
		claude.LoadBalancingLeastConnections,
		claude.LoadBalancingWeightedRoundRobin,
	}
	
	for _, strategy := range strategies {
		t.Run(string(strategy), func(t *testing.T) {
			// 로드 밸런싱 전략 설정
			config := env.SessionPool.GetConfiguration()
			config.LoadBalancingStrategy = strategy
			err := env.SessionPool.UpdateConfiguration(config)
			require.NoError(t, err)
			
			// 동시 세션 생성으로 로드 밸런싱 테스트
			sessionCount := 10
			var wg sync.WaitGroup
			
			for i := 0; i < sessionCount; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					
					session, err := env.CreateTestSession()
					assert.NoError(t, err)
					if session != nil {
						time.Sleep(100 * time.Millisecond)
						env.SessionManager.CloseSession(context.Background(), session.ID)
					}
				}()
			}
			
			wg.Wait()
			
			// 로드 분산 확인
			stats := env.SessionPool.GetStatistics()
			t.Logf("Strategy: %s, Sessions created: %d", strategy, stats.TotalSessions)
			assert.True(t, stats.TotalSessions >= sessionCount, "All sessions should be created")
		})
	}
}

// testSessionPoolSessionReuse 세션 재사용 테스트
func testSessionPoolSessionReuse(t *testing.T, env *test_helpers.AdvancedTestEnvironment) {
	// 세션 재사용 설정 활성화
	config := env.SessionPool.GetConfiguration()
	config.EnableSessionReuse = true
	config.SessionIdleTimeout = 5 * time.Second
	err := env.SessionPool.UpdateConfiguration(config)
	require.NoError(t, err)
	
	// 첫 번째 세션 생성 및 사용
	session1, err := env.CreateTestSession()
	require.NoError(t, err)
	
	originalID := session1.ID
	
	// 세션 종료
	err = env.SessionManager.CloseSession(context.Background(), session1.ID)
	require.NoError(t, err)
	
	// 동일한 프로젝트로 새 세션 요청 - 재사용되어야 함
	session2, err := env.CreateTestSession()
	require.NoError(t, err)
	
	// 재사용 확인 (실제 구현에서는 내부 식별자로 확인)
	stats := env.SessionPool.GetStatistics()
	assert.True(t, stats.ReuseRate > 0, "Session reuse should occur")
	
	t.Logf("Original session: %s, Reused session: %s", originalID, session2.ID)
}

// testSessionPoolResourceLimits 세션 풀 리소스 제한 테스트
func testSessionPoolResourceLimits(t *testing.T, env *test_helpers.AdvancedTestEnvironment) {
	// 리소스 제한 설정
	config := env.SessionPool.GetConfiguration()
	config.MaxSessions = 5
	config.MaxMemoryUsage = 50 * 1024 * 1024 // 50MB
	err := env.SessionPool.UpdateConfiguration(config)
	require.NoError(t, err)
	
	// 제한을 초과하는 세션 생성 시도
	sessions := make([]*claude.Session, 0)
	
	for i := 0; i < 10; i++ {
		session, err := env.CreateTestSession()
		if err != nil {
			t.Logf("Session creation failed at %d: %v", i, err)
			break
		}
		sessions = append(sessions, session)
	}
	
	// 제한이 적용되었는지 확인
	assert.LessOrEqual(t, len(sessions), 5, "Should not exceed max session limit")
	
	// 리소스 사용량 확인
	err = env.ValidateResourceUsage()
	assert.NoError(t, err, "Resource usage should be within limits")
	
	// 세션 정리
	for _, session := range sessions {
		env.SessionManager.CloseSession(context.Background(), session.ID)
	}
}

// TestWebSocketIntegration WebSocket 통합 테스트
func TestWebSocketIntegration(t *testing.T) {
	env, err := test_helpers.NewAdvancedTestEnv(t, nil)
	require.NoError(t, err)
	defer env.Cleanup()
	
	t.Run("RealTimeMessaging", func(t *testing.T) {
		testWebSocketRealTimeMessaging(t, env)
	})
	
	t.Run("MultiUserCollaboration", func(t *testing.T) {
		testWebSocketMultiUserCollaboration(t, env)
	})
	
	t.Run("ConnectionResilience", func(t *testing.T) {
		testWebSocketConnectionResilience(t, env)
	})
}

// testWebSocketRealTimeMessaging WebSocket 실시간 메시징 테스트
func testWebSocketRealTimeMessaging(t *testing.T, env *test_helpers.AdvancedTestEnvironment) {
	sessionID := "test-realtime-session"
	
	// WebSocket 연결 생성
	conn, err := env.CreateWebSocketConnection(sessionID)
	require.NoError(t, err)
	defer env.CloseWebSocketConnection(sessionID)
	
	// 메시지 전송 및 응답 확인
	testMessage := "Hello, Claude!"
	
	err = conn.WriteMessage(websocket.TextMessage, []byte(testMessage))
	require.NoError(t, err)
	
	// 응답 대기
	_, response, err := conn.ReadMessage()
	require.NoError(t, err)
	
	assert.NotEmpty(t, response, "Should receive response")
	t.Logf("Sent: %s, Received: %s", testMessage, string(response))
	
	// 응답 시간 측정
	start := time.Now()
	err = conn.WriteMessage(websocket.TextMessage, []byte("Quick response test"))
	require.NoError(t, err)
	
	_, _, err = conn.ReadMessage()
	require.NoError(t, err)
	
	latency := time.Since(start)
	assert.Less(t, latency, 100*time.Millisecond, "Response should be fast")
}

// testWebSocketMultiUserCollaboration 다중 사용자 협업 테스트
func testWebSocketMultiUserCollaboration(t *testing.T, env *test_helpers.AdvancedTestEnvironment) {
	userCount := 3
	sessionID := "collaborative-session"
	
	// 여러 사용자 연결 생성
	connections := make([]*websocket.Conn, userCount)
	for i := 0; i < userCount; i++ {
		userSessionID := fmt.Sprintf("%s-user-%d", sessionID, i)
		conn, err := env.CreateWebSocketConnection(userSessionID)
		require.NoError(t, err)
		connections[i] = conn
		defer env.CloseWebSocketConnection(userSessionID)
	}
	
	// 메시지 브로드캐스트 테스트
	broadcastMessage := "Broadcast to all users"
	
	// 첫 번째 사용자가 메시지 전송
	err := connections[0].WriteMessage(websocket.TextMessage, []byte(broadcastMessage))
	require.NoError(t, err)
	
	// 모든 사용자가 메시지 수신하는지 확인
	for i := 1; i < userCount; i++ {
		_, response, err := connections[i].ReadMessage()
		require.NoError(t, err)
		
		// 브로드캐스트 메시지가 포함되어 있는지 확인
		assert.Contains(t, string(response), "broadcast", "Should receive broadcast message")
	}
}

// testWebSocketConnectionResilience WebSocket 연결 복구 테스트
func testWebSocketConnectionResilience(t *testing.T, env *test_helpers.AdvancedTestEnvironment) {
	sessionID := "resilience-test-session"
	
	// 초기 연결 생성
	conn, err := env.CreateWebSocketConnection(sessionID)
	require.NoError(t, err)
	
	// 정상 메시지 전송 확인
	err = conn.WriteMessage(websocket.TextMessage, []byte("Test before disconnect"))
	require.NoError(t, err)
	
	_, _, err = conn.ReadMessage()
	require.NoError(t, err)
	
	// 연결 강제 종료
	conn.Close()
	
	// 재연결 시뮬레이션
	time.Sleep(1 * time.Second)
	
	newConn, err := env.CreateWebSocketConnection(sessionID)
	require.NoError(t, err)
	defer env.CloseWebSocketConnection(sessionID)
	
	// 재연결 후 메시지 전송 확인
	err = newConn.WriteMessage(websocket.TextMessage, []byte("Test after reconnect"))
	require.NoError(t, err)
	
	_, response, err := newConn.ReadMessage()
	require.NoError(t, err)
	
	assert.NotEmpty(t, response, "Should work after reconnection")
}

// TestErrorRecoveryIntegration 에러 복구 통합 테스트
func TestErrorRecoveryIntegration(t *testing.T) {
	env, err := test_helpers.NewAdvancedTestEnv(t, nil)
	require.NoError(t, err)
	defer env.Cleanup()
	
	t.Run("CircuitBreakerIntegration", func(t *testing.T) {
		testCircuitBreakerIntegration(t, env)
	})
	
	t.Run("AutomaticRecovery", func(t *testing.T) {
		testAutomaticRecovery(t, env)
	})
	
	t.Run("GracefulDegradation", func(t *testing.T) {
		testGracefulDegradation(t, env)
	})
}

// testCircuitBreakerIntegration Circuit Breaker 통합 테스트
func testCircuitBreakerIntegration(t *testing.T, env *test_helpers.AdvancedTestEnvironment) {
	// Mock 서버에 오류 주입
	env.MockClaude.SetErrorConfig(&test_helpers.ErrorConfig{
		ErrorRate:  0.8, // 80% 오류율
		ErrorTypes: []string{"internal_server_error"},
	})
	
	// 여러 요청으로 Circuit Breaker 트리거
	failureCount := 0
	for i := 0; i < 10; i++ {
		session, err := env.CreateTestSession()
		if err != nil {
			failureCount++
		} else if session != nil {
			env.SessionManager.CloseSession(context.Background(), session.ID)
		}
		time.Sleep(100 * time.Millisecond)
	}
	
	assert.Greater(t, failureCount, 5, "Should have multiple failures to trigger circuit breaker")
	
	// Circuit Breaker가 열린 후 빠른 실패 확인
	start := time.Now()
	_, err = env.CreateTestSession()
	duration := time.Since(start)
	
	// Circuit Breaker가 작동하면 빠르게 실패해야 함
	assert.Less(t, duration, 50*time.Millisecond, "Circuit breaker should fail fast")
	
	// 오류율 정상화
	env.MockClaude.SetErrorConfig(&test_helpers.ErrorConfig{
		ErrorRate: 0.1, // 10% 오류율로 복구
	})
	
	// 복구 대기
	time.Sleep(5 * time.Second)
	
	// 정상 동작 확인
	session, err := env.CreateTestSession()
	assert.NoError(t, err, "Should recover after error rate decreases")
	if session != nil {
		env.SessionManager.CloseSession(context.Background(), session.ID)
	}
}

// TestPerformanceOptimization 성능 최적화 테스트
func TestPerformanceOptimization(t *testing.T) {
	env, err := test_helpers.NewAdvancedTestEnv(t, nil)
	require.NoError(t, err)
	defer env.Cleanup()
	
	t.Run("MemoryPoolEfficiency", func(t *testing.T) {
		testMemoryPoolEfficiency(t, env)
	})
	
	t.Run("GoroutineManagement", func(t *testing.T) {
		testGoroutineManagement(t, env)
	})
	
	t.Run("CachePerformance", func(t *testing.T) {
		testCachePerformance(t, env)
	})
}

// testMemoryPoolEfficiency 메모리 풀 효율성 테스트
func testMemoryPoolEfficiency(t *testing.T, env *test_helpers.AdvancedTestEnvironment) {
	// 성능 추적 시작
	env.PerformanceTracker.Start()
	defer env.PerformanceTracker.Stop()
	
	// 메모리 모니터링 시작
	metricsStop := env.MetricsCollector.StartMemoryMonitoring(50 * time.Millisecond)
	defer close(metricsStop)
	
	// 초기 메모리 상태 기록
	initialUsage := env.PerformanceTracker.GetResourceUsage()
	
	// 대량 세션 생성 및 해제로 메모리 풀 테스트
	sessionCount := 100
	for i := 0; i < sessionCount; i++ {
		session, err := env.CreateTestSession()
		require.NoError(t, err)
		
		// 즉시 해제하여 풀 재사용 유도
		env.SessionManager.CloseSession(context.Background(), session.ID)
		
		if i%10 == 0 {
			// 중간중간 메모리 상태 체크
			env.MetricsCollector.TakeMemorySnapshot()
		}
	}
	
	// 최종 메모리 상태 확인
	finalUsage := env.PerformanceTracker.GetResourceUsage()
	
	// 메모리 증가가 제한적이어야 함 (풀 재사용으로 인해)
	memoryIncrease := finalUsage.MemoryUsage - initialUsage.MemoryUsage
	t.Logf("Memory increase: %d bytes", memoryIncrease)
	
	// 메모리 풀이 효과적이라면 큰 증가가 없어야 함
	assert.Less(t, memoryIncrease, int64(10*1024*1024), "Memory increase should be limited due to pooling")
	
	// GC 강제 실행 후 메모리 감소 확인
	runtime.GC()
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	
	afterGCUsage := env.PerformanceTracker.GetResourceUsage()
	assert.Less(t, afterGCUsage.MemoryUsage, finalUsage.MemoryUsage, "Memory should decrease after GC")
}

// testGoroutineManagement 고루틴 관리 테스트
func testGoroutineManagement(t *testing.T, env *test_helpers.AdvancedTestEnvironment) {
	// 초기 고루틴 수 확인
	initialGoroutines := runtime.NumGoroutine()
	
	// 대량 동시 작업 실행
	var wg sync.WaitGroup
	taskCount := 50
	
	for i := 0; i < taskCount; i++ {
		wg.Add(1)
		go func(taskIndex int) {
			defer wg.Done()
			
			session, err := env.CreateTestSession()
			if err != nil {
				t.Logf("Task %d failed: %v", taskIndex, err)
				return
			}
			
			// 작업 시뮬레이션
			time.Sleep(100 * time.Millisecond)
			
			env.SessionManager.CloseSession(context.Background(), session.ID)
		}(i)
	}
	
	// 최대 고루틴 수 추적
	peakGoroutines := runtime.NumGoroutine()
	
	wg.Wait()
	
	// 작업 완료 후 고루틴 수 확인
	time.Sleep(1 * time.Second) // 고루틴 정리 대기
	finalGoroutines := runtime.NumGoroutine()
	
	t.Logf("Goroutines - Initial: %d, Peak: %d, Final: %d", 
		initialGoroutines, peakGoroutines, finalGoroutines)
	
	// 고루틴 수가 합리적인 범위 내에 있는지 확인
	assert.Less(t, peakGoroutines, initialGoroutines+taskCount+10, "Peak goroutines should be reasonable")
	assert.Less(t, finalGoroutines, initialGoroutines+10, "Should not leak goroutines")
}

// TestE2EWorkflow E2E 워크플로우 테스트
func TestE2EWorkflow(t *testing.T) {
	env, err := test_helpers.NewAdvancedTestEnv(t, nil)
	require.NoError(t, err)
	defer env.Cleanup()
	
	t.Run("CompleteUserWorkflow", func(t *testing.T) {
		testCompleteUserWorkflow(t, env)
	})
}

// testCompleteUserWorkflow 완전한 사용자 워크플로우 테스트
func testCompleteUserWorkflow(t *testing.T, env *test_helpers.AdvancedTestEnvironment) {
	// 1. 웹 세션 생성
	session, err := env.CreateTestSession()
	require.NoError(t, err)
	defer env.SessionManager.CloseSession(context.Background(), session.ID)
	
	// 2. WebSocket 연결
	conn, err := env.CreateWebSocketConnection(session.ID)
	require.NoError(t, err)
	defer env.CloseWebSocketConnection(session.ID)
	
	// 3. Claude와 대화 시작
	greeting := "안녕하세요, Claude!"
	err = conn.WriteMessage(websocket.TextMessage, []byte(greeting))
	require.NoError(t, err)
	
	_, response, err := conn.ReadMessage()
	require.NoError(t, err)
	assert.NotEmpty(t, response, "Should receive greeting response")
	
	// 4. 코드 분석 요청
	codeRequest := "다음 Go 코드를 분석해주세요: func main() { fmt.Println(\"Hello\") }"
	err = conn.WriteMessage(websocket.TextMessage, []byte(codeRequest))
	require.NoError(t, err)
	
	_, codeResponse, err := conn.ReadMessage()
	require.NoError(t, err)
	assert.Contains(t, string(codeResponse), "Go", "Should analyze Go code")
	
	// 5. 세션 상태 확인
	sessionInfo := env.SessionManager.GetSession(session.ID)
	assert.NotNil(t, sessionInfo, "Session should exist")
	assert.True(t, sessionInfo.IsActive, "Session should be active")
	
	// 6. 성능 메트릭 확인
	metrics := env.MetricsCollector.GetMetrics()
	assert.Greater(t, metrics.MessagesSent, int64(0), "Should have sent messages")
	assert.Greater(t, metrics.MessagesReceived, int64(0), "Should have received messages")
	assert.Greater(t, metrics.SuccessRate, 0.8, "Success rate should be high")
}

// TestHighLoadScenario 고부하 시나리오 테스트
func TestHighLoadScenario(t *testing.T) {
	// 고부하 테스트를 위한 설정
	config := test_helpers.DefaultTestConfig()
	config.MaxConcurrentSessions = 100
	config.TestDuration = 30 * time.Second
	config.MessageRate = 50 // 초당 50개 메시지
	
	env, err := test_helpers.NewAdvancedTestEnv(t, config)
	require.NoError(t, err)
	defer env.Cleanup()
	
	// 성능 추적 시작
	env.PerformanceTracker.Start()
	defer env.PerformanceTracker.Stop()
	
	// 고부하 시뮬레이션
	err = env.SimulateHighLoad(50, 10) // 50개 세션, 각각 10개 메시지
	require.NoError(t, err)
	
	// 성능 검증
	err = env.ValidatePerformance()
	assert.NoError(t, err, "Performance should meet requirements under high load")
	
	// 리소스 사용량 검증
	err = env.ValidateResourceUsage()
	assert.NoError(t, err, "Resource usage should be within limits")
	
	// 메트릭 리포트 생성
	report := env.MetricsCollector.GenerateReport()
	t.Logf("High Load Test Report: Throughput=%.2f, Latency=%v, Success Rate=%.2f", 
		report.Summary.Throughput, 
		report.Performance.AverageLatency, 
		report.Summary.SuccessRate)
}

// TestChaosEngineering 카오스 엔지니어링 테스트
func TestChaosEngineering(t *testing.T) {
	config := test_helpers.DefaultTestConfig()
	config.EnableChaosTest = true
	config.FailureRate = 0.3
	config.RecoveryTimeout = 10 * time.Second
	
	env, err := test_helpers.NewAdvancedTestEnv(t, config)
	require.NoError(t, err)
	defer env.Cleanup()
	
	// 정상 상태 확인
	session, err := env.CreateTestSession()
	require.NoError(t, err)
	defer env.SessionManager.CloseSession(context.Background(), session.ID)
	
	// 카오스 주입
	chaosTypes := []test_helpers.ChaosType{
		test_helpers.ChaosTypeNetworkLatency,
		test_helpers.ChaosTypeMemoryPressure,
	}
	
	for _, chaosType := range chaosTypes {
		t.Run(string(chaosType), func(t *testing.T) {
			// 카오스 주입
			err := env.InjectChaos(chaosType)
			if err != nil {
				t.Logf("Chaos injection failed: %v", err)
			}
			
			// 시스템 복구 대기
			time.Sleep(config.RecoveryTimeout)
			
			// 복구 확인
			recoverySession, err := env.CreateTestSession()
			if err != nil {
				t.Logf("System not recovered from %s: %v", chaosType, err)
			} else {
				env.SessionManager.CloseSession(context.Background(), recoverySession.ID)
				t.Logf("System recovered from %s", chaosType)
			}
		})
	}
}