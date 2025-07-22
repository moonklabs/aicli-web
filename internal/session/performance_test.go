// +build performance

package session

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/aicli/aicli-web/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// PerformanceTestSuite는 세션 관리 시스템의 성능 테스트를 위한 테스트 스위트입니다.
type PerformanceTestSuite struct {
	store           *RedisStore
	limiter         *ConcurrentSessionLimiter
	deviceGenerator *DeviceFingerprintGenerator
	geoipService    *GeoIPService
	monitor         *SessionMonitor
}

func setupPerformanceTest(t *testing.T) *PerformanceTestSuite {
	// 실제 Redis 연결이 필요한 성능 테스트
	config := &RedisConfig{
		Addr:              "localhost:6379",
		Password:          "",
		DB:                2, // 성능 테스트용 DB
		KeyPrefix:         "perf_test:",
		DefaultExpiration: time.Hour,
	}

	store, err := NewRedisStore(config)
	require.NoError(t, err, "Redis 연결이 필요합니다")

	// 성능 테스트용 설정
	limiter := NewConcurrentSessionLimiter(store, 10) // 높은 한계
	deviceGenerator := NewDeviceFingerprintGeneratorWithoutGeoIP()
	geoipService := NewGeoIPServiceWithFallback(&GeoIPConfig{})
	monitor := NewSessionMonitor(store)

	return &PerformanceTestSuite{
		store:           store,
		limiter:         limiter,
		deviceGenerator: deviceGenerator,
		geoipService:    geoipService,
		monitor:         monitor,
	}
}

// BenchmarkSessionCreate는 세션 생성 성능을 측정합니다.
func BenchmarkSessionCreate(b *testing.B) {
	suite := setupPerformanceTest(b)
	defer suite.store.Close()
	
	ctx := context.Background()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			session := &models.AuthSession{
				ID:        fmt.Sprintf("bench_create_%d_%d", b.N, i),
				UserID:    fmt.Sprintf("user_%d", i%1000),
				IsActive:  true,
				CreatedAt: time.Now(),
				ExpiresAt: time.Now().Add(time.Hour),
				DeviceInfo: &models.DeviceFingerprint{
					Fingerprint: fmt.Sprintf("device_%d", i%100),
					IPAddress:   fmt.Sprintf("192.168.1.%d", i%255),
					Browser:     "Chrome",
					OS:          "Windows",
					Device:      "Desktop",
				},
			}
			
			err := suite.store.StoreSession(ctx, session)
			if err != nil {
				b.Fatalf("Session creation failed: %v", err)
			}
			i++
		}
	})
}

// BenchmarkSessionGet은 세션 조회 성능을 측정합니다.
func BenchmarkSessionGet(b *testing.B) {
	suite := setupPerformanceTest(b)
	defer suite.store.Close()
	
	ctx := context.Background()
	
	// 사전에 세션들을 생성
	const numSessions = 10000
	sessionIDs := make([]string, numSessions)
	
	for i := 0; i < numSessions; i++ {
		sessionID := fmt.Sprintf("bench_get_%d", i)
		sessionIDs[i] = sessionID
		
		session := &models.AuthSession{
			ID:        sessionID,
			UserID:    fmt.Sprintf("user_%d", i%1000),
			IsActive:  true,
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(time.Hour),
		}
		
		err := suite.store.StoreSession(ctx, session)
		require.NoError(b, err)
	}
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			sessionID := sessionIDs[i%numSessions]
			_, err := suite.store.GetSession(ctx, sessionID)
			if err != nil {
				b.Fatalf("Session get failed: %v", err)
			}
			i++
		}
	})
}

// BenchmarkConcurrentSessionLimiter는 동시 세션 제한기 성능을 측정합니다.
func BenchmarkConcurrentSessionLimiter(b *testing.B) {
	suite := setupPerformanceTest(b)
	defer suite.store.Close()
	
	ctx := context.Background()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			userID := fmt.Sprintf("limiter_user_%d", i%100)
			sessionID := fmt.Sprintf("limiter_session_%d_%d", b.N, i)
			
			allowed, err := suite.limiter.AllowNewSession(ctx, userID, sessionID)
			if err != nil {
				b.Fatalf("Limiter check failed: %v", err)
			}
			
			if allowed {
				// 세션이 허용된 경우 정리
				suite.limiter.RemoveSession(ctx, userID, sessionID)
			}
			i++
		}
	})
}

// BenchmarkDeviceFingerprinting는 디바이스 핑거프린팅 성능을 측정합니다.
func BenchmarkDeviceFingerprinting(b *testing.B) {
	suite := setupPerformanceTest(b)
	
	userAgents := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
		"Mozilla/5.0 (iPhone; CPU iPhone OS 14_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.1.1 Mobile/15E148 Safari/604.1",
		"Mozilla/5.0 (iPad; CPU OS 14_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.1.1 Mobile/15E148 Safari/604.1",
		"Mozilla/5.0 (Android 11; Mobile; rv:89.0) Gecko/89.0 Firefox/89.0",
	}
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			userAgent := userAgents[i%len(userAgents)]
			ipAddress := fmt.Sprintf("192.168.%d.%d", (i/255)%255, i%255)
			
			// 핑거프린트 생성
			fingerprint := suite.deviceGenerator.generateFingerprint(
				userAgent, ipAddress, "Chrome", "Windows", "Desktop")
			
			if fingerprint == "" {
				b.Fatal("Fingerprint generation failed")
			}
			i++
		}
	})
}

// TestHighConcurrencySessionManagement는 고동시성 세션 관리를 테스트합니다.
func TestHighConcurrencySessionManagement(t *testing.T) {
	if testing.Short() {
		t.Skip("고동시성 테스트는 -short 모드에서 건너뜁니다")
	}
	
	suite := setupPerformanceTest(t)
	defer suite.store.Close()
	
	ctx := context.Background()
	
	const (
		numGoroutines = 100
		sessionsPerGoroutine = 50
		totalSessions = numGoroutines * sessionsPerGoroutine
	)
	
	var wg sync.WaitGroup
	var successCount int64
	var errorCount int64
	var mu sync.Mutex
	
	start := time.Now()
	
	// 동시에 많은 세션 생성
	wg.Add(numGoroutines)
	for g := 0; g < numGoroutines; g++ {
		go func(goroutineID int) {
			defer wg.Done()
			
			for s := 0; s < sessionsPerGoroutine; s++ {
				sessionID := fmt.Sprintf("high_concurrency_%d_%d", goroutineID, s)
				session := &models.AuthSession{
					ID:        sessionID,
					UserID:    fmt.Sprintf("user_%d", goroutineID),
					IsActive:  true,
					CreatedAt: time.Now(),
					ExpiresAt: time.Now().Add(time.Hour),
					DeviceInfo: &models.DeviceFingerprint{
						Fingerprint: fmt.Sprintf("device_%d_%d", goroutineID, s),
						IPAddress:   fmt.Sprintf("10.0.%d.%d", goroutineID%255, s%255),
					},
				}
				
				err := suite.store.StoreSession(ctx, session)
				
				mu.Lock()
				if err != nil {
					errorCount++
				} else {
					successCount++
				}
				mu.Unlock()
			}
		}(g)
	}
	
	wg.Wait()
	duration := time.Since(start)
	
	t.Logf("고동시성 세션 생성 완료:")
	t.Logf("  - 총 고루틴: %d", numGoroutines)
	t.Logf("  - 고루틴당 세션: %d", sessionsPerGoroutine)
	t.Logf("  - 총 세션: %d", totalSessions)
	t.Logf("  - 성공: %d", successCount)
	t.Logf("  - 실패: %d", errorCount)
	t.Logf("  - 소요시간: %v", duration)
	t.Logf("  - 초당 세션 생성: %.2f", float64(successCount)/duration.Seconds())
	
	// 성공률 검증
	successRate := float64(successCount) / float64(totalSessions)
	assert.Greater(t, successRate, 0.95, "성공률은 95% 이상이어야 합니다")
	
	// 이제 동시에 세션들을 조회해봅시다
	start = time.Now()
	var readSuccessCount int64
	var readErrorCount int64
	
	wg.Add(numGoroutines)
	for g := 0; g < numGoroutines; g++ {
		go func(goroutineID int) {
			defer wg.Done()
			
			for s := 0; s < sessionsPerGoroutine; s++ {
				sessionID := fmt.Sprintf("high_concurrency_%d_%d", goroutineID, s)
				_, err := suite.store.GetSession(ctx, sessionID)
				
				mu.Lock()
				if err != nil {
					readErrorCount++
				} else {
					readSuccessCount++
				}
				mu.Unlock()
			}
		}(g)
	}
	
	wg.Wait()
	readDuration := time.Since(start)
	
	t.Logf("고동시성 세션 조회 완료:")
	t.Logf("  - 조회 성공: %d", readSuccessCount)
	t.Logf("  - 조회 실패: %d", readErrorCount)
	t.Logf("  - 소요시간: %v", readDuration)
	t.Logf("  - 초당 세션 조회: %.2f", float64(readSuccessCount)/readDuration.Seconds())
	
	// 조회 성공률 검증
	readSuccessRate := float64(readSuccessCount) / float64(successCount)
	assert.Greater(t, readSuccessRate, 0.95, "조회 성공률은 95% 이상이어야 합니다")
}

// TestMemoryLeakDetection는 메모리 누수를 감지합니다.
func TestMemoryLeakDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("메모리 누수 테스트는 -short 모드에서 건너뜁니다")
	}
	
	suite := setupPerformanceTest(t)
	defer suite.store.Close()
	
	ctx := context.Background()
	
	// 초기 메모리 상태 측정
	runtime.GC()
	var initialStats runtime.MemStats
	runtime.ReadMemStats(&initialStats)
	
	const iterations = 1000
	
	// 반복적인 세션 생성/삭제 사이클
	for i := 0; i < iterations; i++ {
		sessionID := fmt.Sprintf("memory_leak_test_%d", i)
		session := &models.AuthSession{
			ID:        sessionID,
			UserID:    fmt.Sprintf("user_%d", i%10),
			IsActive:  true,
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(time.Minute),
			DeviceInfo: &models.DeviceFingerprint{
				Fingerprint: fmt.Sprintf("device_%d", i),
				IPAddress:   "127.0.0.1",
				UserAgent:   "Test Agent",
			},
		}
		
		// 생성
		err := suite.store.StoreSession(ctx, session)
		require.NoError(t, err)
		
		// 조회
		_, err = suite.store.GetSession(ctx, sessionID)
		require.NoError(t, err)
		
		// 업데이트
		session.LastAccess = time.Now()
		err = suite.store.UpdateSession(ctx, session)
		require.NoError(t, err)
		
		// 삭제
		err = suite.store.DeleteSession(ctx, sessionID)
		require.NoError(t, err)
		
		// 중간에 GC 실행
		if i%100 == 0 {
			runtime.GC()
		}
	}
	
	// 최종 메모리 상태 측정
	runtime.GC()
	time.Sleep(100 * time.Millisecond) // GC 완료 대기
	runtime.GC()
	
	var finalStats runtime.MemStats
	runtime.ReadMemStats(&finalStats)
	
	// 메모리 사용량 분석
	initialHeap := initialStats.HeapInuse
	finalHeap := finalStats.HeapInuse
	heapGrowth := finalHeap - initialHeap
	
	t.Logf("메모리 누수 검사 결과:")
	t.Logf("  - 반복횟수: %d", iterations)
	t.Logf("  - 초기 힙 사용량: %d KB", initialHeap/1024)
	t.Logf("  - 최종 힙 사용량: %d KB", finalHeap/1024)
	t.Logf("  - 힙 증가량: %d KB", heapGrowth/1024)
	t.Logf("  - 총 할당량: %d KB", (finalStats.TotalAlloc-initialStats.TotalAlloc)/1024)
	t.Logf("  - GC 실행 횟수: %d", finalStats.NumGC-initialStats.NumGC)
	
	// 메모리 누수 임계값 검증 (1MB 증가까지 허용)
	maxAllowedGrowth := uint64(1024 * 1024) // 1MB
	assert.Less(t, heapGrowth, maxAllowedGrowth, 
		"힙 메모리 증가량이 임계값을 초과했습니다. 메모리 누수가 의심됩니다.")
}

// TestPerformanceUnderLoad는 부하 상황에서의 성능을 테스트합니다.
func TestPerformanceUnderLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("부하 테스트는 -short 모드에서 건너뜁니다")
	}
	
	suite := setupPerformanceTest(t)
	defer suite.store.Close()
	
	ctx := context.Background()
	
	// 부하 테스트 설정
	const (
		duration = 30 * time.Second
		numWorkers = 50
	)
	
	var (
		totalOperations int64
		successCount    int64
		errorCount      int64
		mu             sync.Mutex
	)
	
	// 부하 생성 워커들
	var wg sync.WaitGroup
	stopChan := make(chan struct{})
	
	start := time.Now()
	
	// 시간 제한 설정
	go func() {
		time.Sleep(duration)
		close(stopChan)
	}()
	
	// 워커 고루틴들 시작
	wg.Add(numWorkers)
	for w := 0; w < numWorkers; w++ {
		go func(workerID int) {
			defer wg.Done()
			
			operationCount := 0
			for {
				select {
				case <-stopChan:
					return
				default:
					sessionID := fmt.Sprintf("load_test_%d_%d", workerID, operationCount)
					
					// 세션 생성
					session := &models.AuthSession{
						ID:        sessionID,
						UserID:    fmt.Sprintf("load_user_%d", workerID),
						IsActive:  true,
						CreatedAt: time.Now(),
						ExpiresAt: time.Now().Add(time.Hour),
					}
					
					err := suite.store.StoreSession(ctx, session)
					if err == nil {
						// 성공한 경우 즉시 삭제하여 정리
						suite.store.DeleteSession(ctx, sessionID)
						
						mu.Lock()
						successCount++
						mu.Unlock()
					} else {
						mu.Lock()
						errorCount++
						mu.Unlock()
					}
					
					operationCount++
					
					mu.Lock()
					totalOperations++
					mu.Unlock()
					
					// 짧은 휴식으로 시스템 과부하 방지
					time.Sleep(time.Millisecond)
				}
			}
		}(w)
	}
	
	wg.Wait()
	actualDuration := time.Since(start)
	
	// 결과 분석
	operationsPerSecond := float64(totalOperations) / actualDuration.Seconds()
	successRate := float64(successCount) / float64(totalOperations)
	
	t.Logf("부하 테스트 결과:")
	t.Logf("  - 워커 수: %d", numWorkers)
	t.Logf("  - 실행 시간: %v", actualDuration)
	t.Logf("  - 총 작업 수: %d", totalOperations)
	t.Logf("  - 성공: %d", successCount)
	t.Logf("  - 실패: %d", errorCount)
	t.Logf("  - 초당 작업 수: %.2f", operationsPerSecond)
	t.Logf("  - 성공률: %.2f%%", successRate*100)
	
	// 성능 기준 검증
	assert.Greater(t, operationsPerSecond, 50.0, "초당 최소 50개 작업을 처리해야 합니다")
	assert.Greater(t, successRate, 0.9, "성공률은 90% 이상이어야 합니다")
}

// TestSessionMonitorPerformance는 세션 모니터링 성능을 테스트합니다.
func TestSessionMonitorPerformance(t *testing.T) {
	suite := setupPerformanceTest(t)
	defer suite.store.Close()
	
	ctx := context.Background()
	
	// 테스트용 세션들 생성
	const numTestSessions = 100
	for i := 0; i < numTestSessions; i++ {
		session := &models.AuthSession{
			ID:        fmt.Sprintf("monitor_perf_%d", i),
			UserID:    fmt.Sprintf("monitor_user_%d", i%10),
			IsActive:  i%2 == 0, // 절반은 활성, 절반은 비활성
			CreatedAt: time.Now().Add(time.Duration(-i) * time.Minute),
			ExpiresAt: time.Now().Add(time.Hour),
		}
		
		err := suite.store.StoreSession(ctx, session)
		require.NoError(t, err)
	}
	
	// 통계 조회 성능 측정
	start := time.Now()
	stats, err := suite.monitor.GetSessionStatistics(ctx)
	statsDuration := time.Since(start)
	
	require.NoError(t, err)
	require.NotNil(t, stats)
	
	t.Logf("세션 통계 조회 성능:")
	t.Logf("  - 소요시간: %v", statsDuration)
	t.Logf("  - 총 세션: %d", stats.TotalSessions)
	t.Logf("  - 활성 세션: %d", stats.ActiveSessions)
	
	// 통계 조회는 1초 이내여야 함
	assert.Less(t, statsDuration, time.Second, "통계 조회는 1초 이내에 완료되어야 합니다")
	
	// 사용자별 세션 조회 성능 측정
	start = time.Now()
	for i := 0; i < 10; i++ {
		userID := fmt.Sprintf("monitor_user_%d", i)
		sessions, err := suite.store.GetUserSessions(ctx, userID)
		require.NoError(t, err)
		assert.Greater(t, len(sessions), 0)
	}
	userQueryDuration := time.Since(start)
	
	t.Logf("사용자별 세션 조회 성능:")
	t.Logf("  - 10명 사용자 조회 시간: %v", userQueryDuration)
	t.Logf("  - 사용자당 평균 시간: %v", userQueryDuration/10)
	
	// 사용자별 조회는 평균 100ms 이내여야 함
	avgPerUser := userQueryDuration / 10
	assert.Less(t, avgPerUser, 100*time.Millisecond, "사용자당 평균 조회 시간이 100ms를 초과합니다")
}