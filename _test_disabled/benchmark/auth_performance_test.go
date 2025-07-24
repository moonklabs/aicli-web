package benchmark

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/aicli/aicli-web/internal/auth"
	"github.com/aicli/aicli-web/internal/config"
	"github.com/aicli/aicli-web/internal/middleware"
	"github.com/aicli/aicli-web/internal/storage/memory"
)

// AuthPerformanceBenchmark 인증 시스템 성능 벤치마크
type AuthPerformanceBenchmark struct {
	app           *gin.Engine
	storage       *memory.MemoryStorage
	jwtManager    auth.JWTManager
	rbacManager   auth.RBACManager
	sessionStore  auth.SessionStore
	testTokens    []string
	benchResults  *BenchmarkResults
}

// BenchmarkResults 벤치마크 결과
type BenchmarkResults struct {
	JWTGeneration    BenchResult `json:"jwt_generation"`
	JWTValidation    BenchResult `json:"jwt_validation"`
	RBACPermCheck    BenchResult `json:"rbac_permission_check"`
	SessionCreation  BenchResult `json:"session_creation"`
	SessionRetrieval BenchResult `json:"session_retrieval"`
	FullAuthFlow     BenchResult `json:"full_auth_flow"`
	ConcurrentAuth   BenchResult `json:"concurrent_auth"`
	MemoryUsage      MemoryResult `json:"memory_usage"`
}

// BenchResult 개별 벤치마크 결과
type BenchResult struct {
	Name           string        `json:"name"`
	Operations     int           `json:"operations"`
	Duration       time.Duration `json:"duration"`
	AvgDuration    time.Duration `json:"avg_duration"`
	OpsPerSecond   float64       `json:"ops_per_second"`
	MinDuration    time.Duration `json:"min_duration"`
	MaxDuration    time.Duration `json:"max_duration"`
	P95Duration    time.Duration `json:"p95_duration"`
	Status         string        `json:"status"`
	ErrorRate      float64       `json:"error_rate"`
}

// MemoryResult 메모리 사용량 결과
type MemoryResult struct {
	InitialAlloc   uint64 `json:"initial_alloc_mb"`
	FinalAlloc     uint64 `json:"final_alloc_mb"`
	TotalAlloc     uint64 `json:"total_alloc_mb"`
	MaxAlloc       uint64 `json:"max_alloc_mb"`
	HeapObjects    uint64 `json:"heap_objects"`
	GCRuns         uint32 `json:"gc_runs"`
	MemoryGrowth   uint64 `json:"memory_growth_mb"`
}

// NewAuthPerformanceBenchmark 벤치마크 스위트 생성
func NewAuthPerformanceBenchmark() *AuthPerformanceBenchmark {
	gin.SetMode(gin.TestMode)
	
	storage := memory.NewMemoryStorage()
	jwtManager := auth.NewJWTManager("test-secret", 15*time.Minute, 24*time.Hour)
	rbacManager := auth.NewRBACManager(storage)
	sessionStore := auth.NewRedisSessionStore(&config.RedisConfig{
		Addr: "localhost:6379",
		DB:   1,
	})

	benchmark := &AuthPerformanceBenchmark{
		storage:      storage,
		jwtManager:   jwtManager,
		rbacManager:  rbacManager,
		sessionStore: sessionStore,
		benchResults: &BenchmarkResults{},
	}

	benchmark.setupApplication()
	benchmark.generateTestTokens(1000) // 1000개 테스트 토큰 미리 생성

	return benchmark
}

// setupApplication 테스트용 애플리케이션 설정
func (b *AuthPerformanceBenchmark) setupApplication() {
	b.app = gin.New()
	
	// 성능 측정을 위한 미들웨어 최소화
	b.app.Use(middleware.AuthMiddleware(b.jwtManager))
	
	// 테스트 엔드포인트
	b.app.GET("/api/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	
	b.app.POST("/api/auth", func(c *gin.Context) {
		claims := &auth.Claims{
			UserID:   "test-user",
			Email:    "test@example.com",
			Provider: "local",
		}
		tokens, _ := b.jwtManager.GenerateTokens(claims)
		c.JSON(http.StatusOK, gin.H{"token": tokens.AccessToken})
	})
}

// generateTestTokens 테스트용 토큰들 미리 생성
func (b *AuthPerformanceBenchmark) generateTestTokens(count int) {
	b.testTokens = make([]string, count)
	
	for i := 0; i < count; i++ {
		claims := &auth.Claims{
			UserID:   fmt.Sprintf("user-%d", i),
			Email:    fmt.Sprintf("user%d@example.com", i),
			Provider: "local",
		}
		tokens, _ := b.jwtManager.GenerateTokens(claims)
		b.testTokens[i] = tokens.AccessToken
	}
}

// RunAllBenchmarks 모든 벤치마크 실행
func (b *AuthPerformanceBenchmark) RunAllBenchmarks(t *testing.T) {
	fmt.Println("🚀 인증 시스템 성능 벤치마크 시작...")
	
	// 메모리 사용량 추적 시작
	runtime.GC()
	var initialMem runtime.MemStats
	runtime.ReadMemStats(&initialMem)
	
	// 개별 벤치마크 실행
	fmt.Println("1️⃣ JWT 토큰 생성 성능 테스트...")
	b.benchResults.JWTGeneration = b.benchmarkJWTGeneration(t)
	
	fmt.Println("2️⃣ JWT 토큰 검증 성능 테스트...")
	b.benchResults.JWTValidation = b.benchmarkJWTValidation(t)
	
	fmt.Println("3️⃣ RBAC 권한 검사 성능 테스트...")
	b.benchResults.RBACPermCheck = b.benchmarkRBACPermissionCheck(t)
	
	fmt.Println("4️⃣ 세션 생성 성능 테스트...")
	b.benchResults.SessionCreation = b.benchmarkSessionCreation(t)
	
	fmt.Println("5️⃣ 세션 조회 성능 테스트...")
	b.benchResults.SessionRetrieval = b.benchmarkSessionRetrieval(t)
	
	fmt.Println("6️⃣ 전체 인증 플로우 성능 테스트...")
	b.benchResults.FullAuthFlow = b.benchmarkFullAuthFlow(t)
	
	fmt.Println("7️⃣ 동시성 인증 성능 테스트...")
	b.benchResults.ConcurrentAuth = b.benchmarkConcurrentAuth(t)
	
	// 메모리 사용량 계산
	runtime.GC()
	var finalMem runtime.MemStats
	runtime.ReadMemStats(&finalMem)
	
	b.benchResults.MemoryUsage = MemoryResult{
		InitialAlloc: bToMb(initialMem.Alloc),
		FinalAlloc:   bToMb(finalMem.Alloc),
		TotalAlloc:   bToMb(finalMem.TotalAlloc),
		HeapObjects:  finalMem.HeapObjects,
		GCRuns:       finalMem.NumGC - initialMem.NumGC,
		MemoryGrowth: bToMb(finalMem.Alloc - initialMem.Alloc),
	}
	
	fmt.Println("✅ 모든 벤치마크 완료!")
	b.printResults()
}

// benchmarkJWTGeneration JWT 토큰 생성 벤치마크
func (b *AuthPerformanceBenchmark) benchmarkJWTGeneration(t *testing.T) BenchResult {
	const iterations = 10000
	claims := &auth.Claims{
		UserID:   "benchmark-user",
		Email:    "benchmark@example.com",
		Provider: "local",
	}
	
	durations := make([]time.Duration, iterations)
	errors := 0
	
	start := time.Now()
	for i := 0; i < iterations; i++ {
		iterStart := time.Now()
		_, err := b.jwtManager.GenerateTokens(claims)
		durations[i] = time.Since(iterStart)
		
		if err != nil {
			errors++
		}
	}
	totalDuration := time.Since(start)
	
	return b.calculateBenchResult("JWT Generation", iterations, totalDuration, durations, errors)
}

// benchmarkJWTValidation JWT 토큰 검증 벤치마크
func (b *AuthPerformanceBenchmark) benchmarkJWTValidation(t *testing.T) BenchResult {
	const iterations = 10000
	durations := make([]time.Duration, iterations)
	errors := 0
	
	// 토큰 인덱스를 순환하여 사용
	tokenIndex := 0
	
	start := time.Now()
	for i := 0; i < iterations; i++ {
		iterStart := time.Now()
		token := b.testTokens[tokenIndex%len(b.testTokens)]
		_, err := b.jwtManager.ValidateToken(token)
		durations[i] = time.Since(iterStart)
		
		if err != nil {
			errors++
		}
		tokenIndex++
	}
	totalDuration := time.Since(start)
	
	return b.calculateBenchResult("JWT Validation", iterations, totalDuration, durations, errors)
}

// benchmarkRBACPermissionCheck RBAC 권한 검사 벤치마크
func (b *AuthPerformanceBenchmark) benchmarkRBACPermissionCheck(t *testing.T) BenchResult {
	const iterations = 5000
	durations := make([]time.Duration, iterations)
	errors := 0
	
	// 테스트 사용자와 권한 설정
	userID := "benchmark-user"
	resource := "project"
	action := "read"
	
	start := time.Now()
	for i := 0; i < iterations; i++ {
		iterStart := time.Now()
		hasPermission, err := b.rbacManager.CheckPermission(context.Background(), userID, resource, action, nil)
		durations[i] = time.Since(iterStart)
		
		if err != nil {
			errors++
		}
		_ = hasPermission
	}
	totalDuration := time.Since(start)
	
	return b.calculateBenchResult("RBAC Permission Check", iterations, totalDuration, durations, errors)
}

// benchmarkSessionCreation 세션 생성 벤치마크
func (b *AuthPerformanceBenchmark) benchmarkSessionCreation(t *testing.T) BenchResult {
	const iterations = 3000
	durations := make([]time.Duration, iterations)
	errors := 0
	
	start := time.Now()
	for i := 0; i < iterations; i++ {
		iterStart := time.Now()
		sessionID := fmt.Sprintf("session-%d", i)
		sessionData := &auth.SessionData{
			UserID:    fmt.Sprintf("user-%d", i),
			ExpiresAt: time.Now().Add(time.Hour),
		}
		err := b.sessionStore.Create(context.Background(), sessionID, sessionData)
		durations[i] = time.Since(iterStart)
		
		if err != nil {
			errors++
		}
	}
	totalDuration := time.Since(start)
	
	return b.calculateBenchResult("Session Creation", iterations, totalDuration, durations, errors)
}

// benchmarkSessionRetrieval 세션 조회 벤치마크
func (b *AuthPerformanceBenchmark) benchmarkSessionRetrieval(t *testing.T) BenchResult {
	const iterations = 5000
	durations := make([]time.Duration, iterations)
	errors := 0
	
	// 미리 세션을 생성해둠
	sessionIDs := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		sessionID := fmt.Sprintf("bench-session-%d", i)
		sessionData := &auth.SessionData{
			UserID:    fmt.Sprintf("bench-user-%d", i),
			ExpiresAt: time.Now().Add(time.Hour),
		}
		b.sessionStore.Create(context.Background(), sessionID, sessionData)
		sessionIDs[i] = sessionID
	}
	
	start := time.Now()
	for i := 0; i < iterations; i++ {
		iterStart := time.Now()
		sessionID := sessionIDs[i%len(sessionIDs)]
		_, err := b.sessionStore.Get(context.Background(), sessionID)
		durations[i] = time.Since(iterStart)
		
		if err != nil {
			errors++
		}
	}
	totalDuration := time.Since(start)
	
	return b.calculateBenchResult("Session Retrieval", iterations, totalDuration, durations, errors)
}

// benchmarkFullAuthFlow 전체 인증 플로우 벤치마크
func (b *AuthPerformanceBenchmark) benchmarkFullAuthFlow(t *testing.T) BenchResult {
	const iterations = 1000
	durations := make([]time.Duration, iterations)
	errors := 0
	
	start := time.Now()
	for i := 0; i < iterations; i++ {
		iterStart := time.Now()
		
		// 1. 토큰 생성
		claims := &auth.Claims{
			UserID:   fmt.Sprintf("flow-user-%d", i),
			Email:    fmt.Sprintf("flow%d@example.com", i),
			Provider: "local",
		}
		tokens, err := b.jwtManager.GenerateTokens(claims)
		if err != nil {
			errors++
			continue
		}
		
		// 2. API 호출 (토큰 검증 포함)
		req := httptest.NewRequest("GET", "/api/test", nil)
		req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
		w := httptest.NewRecorder()
		b.app.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			errors++
		}
		
		durations[i] = time.Since(iterStart)
	}
	totalDuration := time.Since(start)
	
	return b.calculateBenchResult("Full Auth Flow", iterations, totalDuration, durations, errors)
}

// benchmarkConcurrentAuth 동시성 인증 벤치마크
func (b *AuthPerformanceBenchmark) benchmarkConcurrentAuth(t *testing.T) BenchResult {
	const totalOps = 5000
	const concurrency = 100
	const opsPerGoroutine = totalOps / concurrency
	
	durations := make([]time.Duration, totalOps)
	errors := int64(0)
	var wg sync.WaitGroup
	var mu sync.Mutex
	opIndex := 0
	
	start := time.Now()
	
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			
			for j := 0; j < opsPerGoroutine; j++ {
				iterStart := time.Now()
				
				// 토큰 검증 수행
				tokenIndex := (goroutineID*opsPerGoroutine + j) % len(b.testTokens)
				token := b.testTokens[tokenIndex]
				_, err := b.jwtManager.ValidateToken(token)
				
				duration := time.Since(iterStart)
				
				mu.Lock()
				if err != nil {
					errors++
				}
				durations[opIndex] = duration
				opIndex++
				mu.Unlock()
			}
		}(i)
	}
	
	wg.Wait()
	totalDuration := time.Since(start)
	
	return b.calculateBenchResult("Concurrent Auth", totalOps, totalDuration, durations[:opIndex], int(errors))
}

// calculateBenchResult 벤치마크 결과 계산
func (b *AuthPerformanceBenchmark) calculateBenchResult(name string, ops int, totalDuration time.Duration, durations []time.Duration, errors int) BenchResult {
	// 통계 계산
	var minDur, maxDur time.Duration = time.Hour, 0
	var totalDur time.Duration = 0
	
	for _, d := range durations {
		totalDur += d
		if d < minDur {
			minDur = d
		}
		if d > maxDur {
			maxDur = d
		}
	}
	
	avgDur := totalDur / time.Duration(len(durations))
	opsPerSec := float64(ops) / totalDuration.Seconds()
	errorRate := float64(errors) / float64(ops) * 100
	
	// P95 계산 (간단한 근사)
	p95Index := int(float64(len(durations)) * 0.95)
	p95Dur := durations[p95Index]
	
	// 성능 기준 검사
	status := "passed"
	if avgDur > 10*time.Millisecond {
		status = "warning"
	}
	if avgDur > 50*time.Millisecond || errorRate > 1.0 {
		status = "failed"
	}
	
	return BenchResult{
		Name:         name,
		Operations:   ops,
		Duration:     totalDuration,
		AvgDuration:  avgDur,
		OpsPerSecond: opsPerSec,
		MinDuration:  minDur,
		MaxDuration:  maxDur,
		P95Duration:  p95Dur,
		Status:       status,
		ErrorRate:    errorRate,
	}
}

// printResults 결과 출력
func (b *AuthPerformanceBenchmark) printResults() {
	fmt.Println("\n📊 인증 시스템 성능 벤치마크 결과:")
	fmt.Println("=" * 80)
	
	results := []BenchResult{
		b.benchResults.JWTGeneration,
		b.benchResults.JWTValidation,
		b.benchResults.RBACPermCheck,
		b.benchResults.SessionCreation,
		b.benchResults.SessionRetrieval,
		b.benchResults.FullAuthFlow,
		b.benchResults.ConcurrentAuth,
	}
	
	for _, result := range results {
		statusIcon := "✅"
		if result.Status == "warning" {
			statusIcon = "⚠️"
		} else if result.Status == "failed" {
			statusIcon = "❌"
		}
		
		fmt.Printf("%s %s:\n", statusIcon, result.Name)
		fmt.Printf("   Operations: %d | Avg: %v | Ops/sec: %.1f\n", 
			result.Operations, result.AvgDuration, result.OpsPerSecond)
		fmt.Printf("   Min: %v | Max: %v | P95: %v | Errors: %.1f%%\n\n",
			result.MinDuration, result.MaxDuration, result.P95Duration, result.ErrorRate)
	}
	
	fmt.Printf("💾 메모리 사용량:\n")
	fmt.Printf("   시작: %d MB | 종료: %d MB | 증가: %d MB\n",
		b.benchResults.MemoryUsage.InitialAlloc,
		b.benchResults.MemoryUsage.FinalAlloc,
		b.benchResults.MemoryUsage.MemoryGrowth)
	fmt.Printf("   GC 실행: %d회 | Heap Objects: %d\n",
		b.benchResults.MemoryUsage.GCRuns,
		b.benchResults.MemoryUsage.HeapObjects)
}

// bToMb 바이트를 MB로 변환
func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}

// TestAuthPerformanceBenchmark 성능 벤치마크 테스트
func TestAuthPerformanceBenchmark(t *testing.T) {
	if testing.Short() {
		t.Skip("성능 벤치마크는 short 모드에서 건너뜁니다")
	}
	
	benchmark := NewAuthPerformanceBenchmark()
	benchmark.RunAllBenchmarks(t)
	
	// 성능 기준 검증
	require.Greater(t, benchmark.benchResults.JWTGeneration.OpsPerSecond, 1000.0, 
		"JWT 생성 성능이 기준보다 낮습니다")
	require.Greater(t, benchmark.benchResults.JWTValidation.OpsPerSecond, 5000.0, 
		"JWT 검증 성능이 기준보다 낮습니다")
	require.Less(t, benchmark.benchResults.FullAuthFlow.AvgDuration, 10*time.Millisecond, 
		"전체 인증 플로우 응답시간이 기준보다 높습니다")
}

// BenchmarkJWTGeneration JWT 생성 벤치마크
func BenchmarkJWTGeneration(b *testing.B) {
	benchmark := NewAuthPerformanceBenchmark()
	claims := &auth.Claims{
		UserID:   "benchmark-user",
		Email:    "benchmark@example.com",
		Provider: "local",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchmark.jwtManager.GenerateTokens(claims)
	}
}

// BenchmarkJWTValidation JWT 검증 벤치마크
func BenchmarkJWTValidation(b *testing.B) {
	benchmark := NewAuthPerformanceBenchmark()
	token := benchmark.testTokens[0]
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchmark.jwtManager.ValidateToken(token)
	}
}

// BenchmarkFullAuthFlow 전체 인증 플로우 벤치마크
func BenchmarkFullAuthFlow(b *testing.B) {
	benchmark := NewAuthPerformanceBenchmark()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 토큰 생성
		claims := &auth.Claims{
			UserID:   fmt.Sprintf("user-%d", i),
			Email:    fmt.Sprintf("user%d@example.com", i),
			Provider: "local",
		}
		tokens, _ := benchmark.jwtManager.GenerateTokens(claims)
		
		// API 호출
		req := httptest.NewRequest("GET", "/api/test", nil)
		req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
		w := httptest.NewRecorder()
		benchmark.app.ServeHTTP(w, req)
	}
}