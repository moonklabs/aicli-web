package test_helpers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/claude-ai/aicli-web/internal/claude"
	"github.com/claude-ai/aicli-web/internal/api/websocket"
	"github.com/claude-ai/aicli-web/internal/storage"
)

// AdvancedTestEnvironment 고급 테스트 환경
type AdvancedTestEnvironment struct {
	// 세션 관리
	SessionPool       *claude.AdvancedSessionPool
	SessionManager    claude.SessionManager
	
	// 웹 서버
	WebServer         *httptest.Server
	WSHandler         *websocket.ClaudeStreamHandler
	WSConnections     map[string]*websocket.Conn
	wsConnMutex       sync.RWMutex
	
	// Mock 서비스
	MockClaude        *MockClaudeServer
	MockStorage       storage.Storage
	
	// 메트릭 및 모니터링
	MetricsCollector  *MetricsCollector
	PerformanceTracker *PerformanceTracker
	
	// 테스트 설정
	Config            *TestConfig
	
	// 생명주기 관리
	ctx               context.Context
	cancel            context.CancelFunc
	cleanup           []func() error
	t                 *testing.T
}

// TestConfig 테스트 설정
type TestConfig struct {
	// 세션 풀 설정
	SessionPoolConfig *claude.AdvancedSessionPoolConfig
	
	// 성능 테스트 설정
	MaxConcurrentSessions  int           `json:"max_concurrent_sessions"`
	TestDuration          time.Duration `json:"test_duration"`
	MessageRate           int           `json:"message_rate"` // 초당 메시지 수
	
	// 카오스 테스트 설정
	EnableChaosTest       bool          `json:"enable_chaos_test"`
	FailureRate          float64       `json:"failure_rate"`
	RecoveryTimeout      time.Duration `json:"recovery_timeout"`
	
	// 리소스 제한
	MaxMemoryUsage       int64         `json:"max_memory_usage"`
	MaxGoroutineCount    int           `json:"max_goroutine_count"`
	
	// 검증 기준
	ExpectedThroughput   float64       `json:"expected_throughput"`
	MaxLatency          time.Duration `json:"max_latency"`
	MinSuccessRate      float64       `json:"min_success_rate"`
}

// DefaultTestConfig 기본 테스트 설정
func DefaultTestConfig() *TestConfig {
	return &TestConfig{
		MaxConcurrentSessions: 100,
		TestDuration:         5 * time.Minute,
		MessageRate:          10,
		EnableChaosTest:      false,
		FailureRate:         0.1,
		RecoveryTimeout:     30 * time.Second,
		MaxMemoryUsage:      100 * 1024 * 1024, // 100MB
		MaxGoroutineCount:   1000,
		ExpectedThroughput:  100.0, // 초당 처리량
		MaxLatency:         100 * time.Millisecond,
		MinSuccessRate:     0.95,
	}
}

// NewAdvancedTestEnv 새로운 고급 테스트 환경 생성
func NewAdvancedTestEnv(t *testing.T, config *TestConfig) (*AdvancedTestEnvironment, error) {
	if config == nil {
		config = DefaultTestConfig()
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	env := &AdvancedTestEnvironment{
		WSConnections:    make(map[string]*websocket.Conn),
		Config:          config,
		ctx:             ctx,
		cancel:          cancel,
		cleanup:         make([]func() error, 0),
		t:               t,
	}
	
	// 컴포넌트 초기화
	if err := env.initializeComponents(); err != nil {
		env.Cleanup()
		return nil, fmt.Errorf("failed to initialize test environment: %w", err)
	}
	
	return env, nil
}

// initializeComponents 컴포넌트 초기화
func (env *AdvancedTestEnvironment) initializeComponents() error {
	// Mock Claude 서버 초기화
	env.MockClaude = NewMockClaudeServer()
	env.addCleanup(env.MockClaude.Stop)
	
	// Mock 스토리지 초기화
	env.MockStorage = storage.NewMemoryStorage()
	
	// 세션 풀 초기화
	if env.Config.SessionPoolConfig == nil {
		env.Config.SessionPoolConfig = claude.DefaultAdvancedSessionPoolConfig()
	}
	
	sessionPool, err := claude.NewAdvancedSessionPool(env.Config.SessionPoolConfig)
	if err != nil {
		return fmt.Errorf("failed to create session pool: %w", err)
	}
	env.SessionPool = sessionPool
	env.addCleanup(func() error {
		return env.SessionPool.Shutdown(context.Background())
	})
	
	// 세션 매니저 초기화
	sessionManager := claude.NewSessionManager(env.MockStorage)
	env.SessionManager = sessionManager
	
	// WebSocket 핸들러 초기화
	wsHandler := websocket.NewClaudeStreamHandler(
		env.SessionManager,
		nil, // auth validator는 테스트에서 생략
	)
	env.WSHandler = wsHandler
	
	// 웹 서버 초기화
	env.initializeWebServer()
	
	// 메트릭 수집기 초기화
	env.MetricsCollector = NewMetricsCollector()
	env.PerformanceTracker = NewPerformanceTracker(env.Config)
	
	return nil
}

// initializeWebServer 웹 서버 초기화
func (env *AdvancedTestEnvironment) initializeWebServer() {
	mux := http.NewServeMux()
	
	// WebSocket 엔드포인트
	mux.HandleFunc("/ws", env.WSHandler.HandleWebSocket)
	
	// 테스트용 REST API 엔드포인트
	mux.HandleFunc("/api/sessions", env.handleSessions)
	mux.HandleFunc("/api/health", env.handleHealth)
	mux.HandleFunc("/api/metrics", env.handleMetrics)
	
	env.WebServer = httptest.NewServer(mux)
	env.addCleanup(func() error {
		env.WebServer.Close()
		return nil
	})
}

// CreateWebSocketConnection WebSocket 연결 생성
func (env *AdvancedTestEnvironment) CreateWebSocketConnection(sessionID string) (*websocket.Conn, error) {
	wsURL := "ws" + env.WebServer.URL[4:] + "/ws"
	
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to WebSocket: %w", err)
	}
	
	env.wsConnMutex.Lock()
	env.WSConnections[sessionID] = conn
	env.wsConnMutex.Unlock()
	
	return conn, nil
}

// CloseWebSocketConnection WebSocket 연결 종료
func (env *AdvancedTestEnvironment) CloseWebSocketConnection(sessionID string) error {
	env.wsConnMutex.Lock()
	defer env.wsConnMutex.Unlock()
	
	if conn, exists := env.WSConnections[sessionID]; exists {
		err := conn.Close()
		delete(env.WSConnections, sessionID)
		return err
	}
	
	return nil
}

// CreateTestSession 테스트 세션 생성
func (env *AdvancedTestEnvironment) CreateTestSession() (*claude.Session, error) {
	sessionID := fmt.Sprintf("test-session-%d", time.Now().UnixNano())
	
	session, err := env.SessionManager.CreateSession(env.ctx, &claude.CreateSessionRequest{
		SessionID:   sessionID,
		ProjectPath: "/tmp/test-project",
		Config: claude.SessionConfig{
			Model:        "claude-3-sonnet",
			MaxTokens:    4096,
			Temperature:  0.7,
		},
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to create test session: %w", err)
	}
	
	return session, nil
}

// SimulateHighLoad 고부하 시뮬레이션
func (env *AdvancedTestEnvironment) SimulateHighLoad(sessionCount int, messageCount int) error {
	var wg sync.WaitGroup
	errorChan := make(chan error, sessionCount)
	
	for i := 0; i < sessionCount; i++ {
		wg.Add(1)
		go func(sessionIndex int) {
			defer wg.Done()
			
			sessionID := fmt.Sprintf("load-test-%d", sessionIndex)
			
			// 세션 생성
			session, err := env.CreateTestSession()
			if err != nil {
				errorChan <- fmt.Errorf("session %d creation failed: %w", sessionIndex, err)
				return
			}
			
			// WebSocket 연결
			conn, err := env.CreateWebSocketConnection(sessionID)
			if err != nil {
				errorChan <- fmt.Errorf("session %d websocket connection failed: %w", sessionIndex, err)
				return
			}
			defer env.CloseWebSocketConnection(sessionID)
			
			// 메시지 전송
			for j := 0; j < messageCount; j++ {
				message := fmt.Sprintf("Test message %d from session %d", j, sessionIndex)
				
				err := conn.WriteMessage(websocket.TextMessage, []byte(message))
				if err != nil {
					errorChan <- fmt.Errorf("session %d message %d send failed: %w", sessionIndex, j, err)
					return
				}
				
				// 응답 대기
				_, response, err := conn.ReadMessage()
				if err != nil {
					errorChan <- fmt.Errorf("session %d message %d response failed: %w", sessionIndex, j, err)
					return
				}
				
				// 메트릭 기록
				env.MetricsCollector.RecordMessageExchange(len(message), len(response))
			}
		}(i)
	}
	
	wg.Wait()
	close(errorChan)
	
	// 에러 수집
	var errors []error
	for err := range errorChan {
		errors = append(errors, err)
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("load test failed with %d errors: %v", len(errors), errors[0])
	}
	
	return nil
}

// InjectChaos 카오스 주입
func (env *AdvancedTestEnvironment) InjectChaos(chaosType ChaosType) error {
	switch chaosType {
	case ChaosTypeNetworkLatency:
		return env.simulateNetworkLatency()
	case ChaosTypeMemoryPressure:
		return env.simulateMemoryPressure()
	case ChaosTypeProcessKill:
		return env.simulateProcessKill()
	case ChaosTypeDiskFull:
		return env.simulateDiskFull()
	default:
		return fmt.Errorf("unknown chaos type: %v", chaosType)
	}
}

// ValidatePerformance 성능 검증
func (env *AdvancedTestEnvironment) ValidatePerformance() error {
	metrics := env.MetricsCollector.GetMetrics()
	
	// 처리량 검증
	if metrics.Throughput < env.Config.ExpectedThroughput {
		return fmt.Errorf("throughput too low: expected %.2f, got %.2f", 
			env.Config.ExpectedThroughput, metrics.Throughput)
	}
	
	// 지연시간 검증
	if metrics.AverageLatency > env.Config.MaxLatency {
		return fmt.Errorf("latency too high: expected %v, got %v", 
			env.Config.MaxLatency, metrics.AverageLatency)
	}
	
	// 성공률 검증
	if metrics.SuccessRate < env.Config.MinSuccessRate {
		return fmt.Errorf("success rate too low: expected %.2f, got %.2f", 
			env.Config.MinSuccessRate, metrics.SuccessRate)
	}
	
	return nil
}

// ValidateResourceUsage 리소스 사용량 검증
func (env *AdvancedTestEnvironment) ValidateResourceUsage() error {
	usage := env.PerformanceTracker.GetResourceUsage()
	
	// 메모리 사용량 검증
	if usage.MemoryUsage > env.Config.MaxMemoryUsage {
		return fmt.Errorf("memory usage too high: expected %d, got %d", 
			env.Config.MaxMemoryUsage, usage.MemoryUsage)
	}
	
	// 고루틴 수 검증
	if usage.GoroutineCount > env.Config.MaxGoroutineCount {
		return fmt.Errorf("goroutine count too high: expected %d, got %d", 
			env.Config.MaxGoroutineCount, usage.GoroutineCount)
	}
	
	return nil
}

// Cleanup 테스트 환경 정리
func (env *AdvancedTestEnvironment) Cleanup() {
	env.cancel()
	
	// WebSocket 연결 정리
	env.wsConnMutex.Lock()
	for sessionID, conn := range env.WSConnections {
		conn.Close()
		delete(env.WSConnections, sessionID)
	}
	env.wsConnMutex.Unlock()
	
	// 등록된 정리 함수들 실행
	for i := len(env.cleanup) - 1; i >= 0; i-- {
		if err := env.cleanup[i](); err != nil {
			env.t.Logf("Cleanup error: %v", err)
		}
	}
}

// 내부 유틸리티 메서드들

func (env *AdvancedTestEnvironment) addCleanup(fn func() error) {
	env.cleanup = append(env.cleanup, fn)
}

func (env *AdvancedTestEnvironment) handleSessions(w http.ResponseWriter, r *http.Request) {
	// 세션 관리 REST API 구현
	switch r.Method {
	case "GET":
		// 세션 목록 조회
		sessions := env.SessionManager.GetActiveSessions()
		json.NewEncoder(w).Encode(sessions)
	case "POST":
		// 새 세션 생성
		// 구현 생략
	case "DELETE":
		// 세션 삭제
		// 구현 생략
	}
}

func (env *AdvancedTestEnvironment) handleHealth(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status": "ok",
		"timestamp": time.Now(),
		"sessions": env.SessionManager.GetSessionCount(),
		"connections": len(env.WSConnections),
	}
	json.NewEncoder(w).Encode(health)
}

func (env *AdvancedTestEnvironment) handleMetrics(w http.ResponseWriter, r *http.Request) {
	metrics := env.MetricsCollector.GetMetrics()
	json.NewEncoder(w).Encode(metrics)
}

// 카오스 엔지니어링 시뮬레이션 메서드들

func (env *AdvancedTestEnvironment) simulateNetworkLatency() error {
	// 네트워크 지연 시뮬레이션
	// 실제 구현에서는 네트워크 인터페이스 조작 또는 프록시 사용
	time.Sleep(100 * time.Millisecond)
	return nil
}

func (env *AdvancedTestEnvironment) simulateMemoryPressure() error {
	// 메모리 압박 시뮬레이션
	// 대량의 메모리 할당으로 GC 압박 생성
	_ = make([]byte, 10*1024*1024) // 10MB 할당
	return nil
}

func (env *AdvancedTestEnvironment) simulateProcessKill() error {
	// 프로세스 종료 시뮬레이션
	// 테스트 환경에서는 고루틴 취소로 대체
	env.cancel()
	return nil
}

func (env *AdvancedTestEnvironment) simulateDiskFull() error {
	// 디스크 공간 부족 시뮬레이션
	// 테스트 환경에서는 임시 파일 생성으로 대체
	return fmt.Errorf("simulated disk full error")
}

// ChaosType 카오스 타입
type ChaosType int

const (
	ChaosTypeNetworkLatency ChaosType = iota
	ChaosTypeMemoryPressure
	ChaosTypeProcessKill
	ChaosTypeDiskFull
)