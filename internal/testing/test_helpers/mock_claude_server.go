package test_helpers

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// MockClaudeServer Mock Claude 서버
type MockClaudeServer struct {
	server       *httptest.Server
	config       *MockServerConfig
	
	// 상태 관리
	running      atomic.Bool
	requestCount int64
	errorCount   int64
	
	// 응답 시뮬레이션
	responses    map[string]string
	responsesMu  sync.RWMutex
	
	// 지연 시뮬레이션
	latencyConfig *LatencyConfig
	
	// 오류 주입
	errorConfig  *ErrorConfig
	
	// 세션 관리
	sessions     map[string]*MockSession
	sessionsMu   sync.RWMutex
}

// MockServerConfig Mock 서버 설정
type MockServerConfig struct {
	// 기본 응답 설정
	DefaultDelay        time.Duration `json:"default_delay"`
	DefaultErrorRate    float64       `json:"default_error_rate"`
	
	// 리소스 제한
	MaxConcurrentRequests int         `json:"max_concurrent_requests"`
	MaxRequestSize       int64        `json:"max_request_size"`
	
	// 기능 설정
	EnableStreaming     bool          `json:"enable_streaming"`
	EnableFileUpload    bool          `json:"enable_file_upload"`
	EnableMultiSession  bool          `json:"enable_multi_session"`
	
	// 시뮬레이션 설정
	SimulateThinking    bool          `json:"simulate_thinking"`
	ThinkingTime        time.Duration `json:"thinking_time"`
}

// LatencyConfig 지연 시간 설정
type LatencyConfig struct {
	MinLatency    time.Duration `json:"min_latency"`
	MaxLatency    time.Duration `json:"max_latency"`
	Distribution  string        `json:"distribution"` // "uniform", "normal", "exponential"
	Jitter        float64       `json:"jitter"`       // 0.0 ~ 1.0
}

// ErrorConfig 오류 주입 설정
type ErrorConfig struct {
	ErrorRate      float64           `json:"error_rate"`
	ErrorTypes     []string          `json:"error_types"`
	ErrorMessages  map[string]string `json:"error_messages"`
	TransientErrors bool             `json:"transient_errors"`
}

// MockSession Mock 세션
type MockSession struct {
	ID           string                 `json:"id"`
	CreatedAt    time.Time              `json:"created_at"`
	LastActivity time.Time              `json:"last_activity"`
	MessageCount int                    `json:"message_count"`
	Context      map[string]interface{} `json:"context"`
	IsActive     bool                   `json:"is_active"`
}

// DefaultMockServerConfig 기본 Mock 서버 설정
func DefaultMockServerConfig() *MockServerConfig {
	return &MockServerConfig{
		DefaultDelay:          10 * time.Millisecond,
		DefaultErrorRate:      0.05, // 5% 오류율
		MaxConcurrentRequests: 100,
		MaxRequestSize:        10 * 1024 * 1024, // 10MB
		EnableStreaming:       true,
		EnableFileUpload:      true,
		EnableMultiSession:    true,
		SimulateThinking:      true,
		ThinkingTime:          50 * time.Millisecond,
	}
}

// DefaultLatencyConfig 기본 지연 시간 설정
func DefaultLatencyConfig() *LatencyConfig {
	return &LatencyConfig{
		MinLatency:   10 * time.Millisecond,
		MaxLatency:   100 * time.Millisecond,
		Distribution: "normal",
		Jitter:       0.2,
	}
}

// DefaultErrorConfig 기본 오류 설정
func DefaultErrorConfig() *ErrorConfig {
	return &ErrorConfig{
		ErrorRate: 0.05,
		ErrorTypes: []string{
			"rate_limit_exceeded",
			"internal_server_error",
			"timeout",
			"invalid_request",
		},
		ErrorMessages: map[string]string{
			"rate_limit_exceeded":   "Rate limit exceeded. Please try again later.",
			"internal_server_error": "Internal server error occurred.",
			"timeout":              "Request timeout occurred.",
			"invalid_request":      "Invalid request format.",
		},
		TransientErrors: true,
	}
}

// NewMockClaudeServer 새로운 Mock Claude 서버 생성
func NewMockClaudeServer() *MockClaudeServer {
	config := DefaultMockServerConfig()
	latencyConfig := DefaultLatencyConfig()
	errorConfig := DefaultErrorConfig()
	
	mock := &MockClaudeServer{
		config:        config,
		latencyConfig: latencyConfig,
		errorConfig:   errorConfig,
		responses:     make(map[string]string),
		sessions:      make(map[string]*MockSession),
	}
	
	// 기본 응답 설정
	mock.setupDefaultResponses()
	
	// HTTP 서버 시작
	mock.startServer()
	
	return mock
}

// startServer HTTP 서버 시작
func (ms *MockClaudeServer) startServer() {
	mux := http.NewServeMux()
	
	// Claude API 엔드포인트
	mux.HandleFunc("/v1/chat/completions", ms.handleChatCompletions)
	mux.HandleFunc("/v1/sessions", ms.handleSessions)
	mux.HandleFunc("/v1/sessions/", ms.handleSessionOperations)
	mux.HandleFunc("/v1/files/upload", ms.handleFileUpload)
	mux.HandleFunc("/v1/health", ms.handleHealth)
	
	// 테스트용 관리 엔드포인트
	mux.HandleFunc("/admin/config", ms.handleAdminConfig)
	mux.HandleFunc("/admin/responses", ms.handleAdminResponses)
	mux.HandleFunc("/admin/metrics", ms.handleAdminMetrics)
	mux.HandleFunc("/admin/reset", ms.handleAdminReset)
	
	ms.server = httptest.NewServer(mux)
	ms.running.Store(true)
}

// Stop 서버 중지
func (ms *MockClaudeServer) Stop() error {
	if !ms.running.CompareAndSwap(true, false) {
		return nil
	}
	
	if ms.server != nil {
		ms.server.Close()
	}
	
	return nil
}

// GetURL 서버 URL 조회
func (ms *MockClaudeServer) GetURL() string {
	if ms.server == nil {
		return ""
	}
	return ms.server.URL
}

// setupDefaultResponses 기본 응답 설정
func (ms *MockClaudeServer) setupDefaultResponses() {
	ms.responses["hello"] = "안녕하세요! Claude입니다. 무엇을 도와드릴까요?"
	ms.responses["how are you"] = "저는 잘 지내고 있습니다. 감사합니다!"
	ms.responses["test"] = "테스트 응답입니다."
	ms.responses["error"] = "ERROR: 테스트 오류입니다."
	ms.responses["long"] = strings.Repeat("이것은 긴 응답입니다. ", 100)
	ms.responses["code"] = `다음은 간단한 Go 코드 예제입니다:

package main

import "fmt"

func main() {
    fmt.Println("Hello, World!")
}`
}

// SetResponse 사용자 정의 응답 설정
func (ms *MockClaudeServer) SetResponse(key, response string) {
	ms.responsesMu.Lock()
	defer ms.responsesMu.Unlock()
	
	ms.responses[key] = response
}

// SetLatencyConfig 지연 시간 설정
func (ms *MockClaudeServer) SetLatencyConfig(config *LatencyConfig) {
	ms.latencyConfig = config
}

// SetErrorConfig 오류 설정
func (ms *MockClaudeServer) SetErrorConfig(config *ErrorConfig) {
	ms.errorConfig = config
}

// InjectError 오류 주입
func (ms *MockClaudeServer) InjectError(sessionID string, errorType string) {
	ms.sessionsMu.Lock()
	defer ms.sessionsMu.Unlock()
	
	if session, exists := ms.sessions[sessionID]; exists {
		if session.Context == nil {
			session.Context = make(map[string]interface{})
		}
		session.Context["inject_error"] = errorType
	}
}

// SimulateOverload 과부하 시뮬레이션
func (ms *MockClaudeServer) SimulateOverload(duration time.Duration) {
	originalDelay := ms.config.DefaultDelay
	originalErrorRate := ms.config.DefaultErrorRate
	
	// 응답 시간 증가 및 오류율 증가
	ms.config.DefaultDelay = 5 * time.Second
	ms.config.DefaultErrorRate = 0.5
	
	// 지정된 시간 후 원래 설정으로 복원
	time.AfterFunc(duration, func() {
		ms.config.DefaultDelay = originalDelay
		ms.config.DefaultErrorRate = originalErrorRate
	})
}

// HTTP 핸들러들

// handleChatCompletions 채팅 완료 핸들러
func (ms *MockClaudeServer) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	atomic.AddInt64(&ms.requestCount, 1)
	
	// 동시 요청 수 제한
	if ms.getCurrentRequestCount() > ms.config.MaxConcurrentRequests {
		http.Error(w, "Too many concurrent requests", http.StatusTooManyRequests)
		atomic.AddInt64(&ms.errorCount, 1)
		return
	}
	
	// 지연 시뮬레이션
	ms.simulateLatency()
	
	// 오류 주입
	if ms.shouldInjectError() {
		ms.handleInjectedError(w, r)
		atomic.AddInt64(&ms.errorCount, 1)
		return
	}
	
	// 요청 처리
	switch r.Method {
	case "POST":
		ms.handleChatMessage(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		atomic.AddInt64(&ms.errorCount, 1)
	}
}

// handleChatMessage 채팅 메시지 처리
func (ms *MockClaudeServer) handleChatMessage(w http.ResponseWriter, r *http.Request) {
	// 요청 본문 읽기
	body := make([]byte, r.ContentLength)
	r.Body.Read(body)
	defer r.Body.Close()
	
	// 간단한 키워드 매칭으로 응답 생성
	message := string(body)
	response := ms.generateResponse(message)
	
	// 스트리밍 응답 시뮬레이션
	if ms.config.EnableStreaming && strings.Contains(r.Header.Get("Accept"), "text/stream") {
		ms.handleStreamingResponse(w, response)
	} else {
		ms.handleRegularResponse(w, response)
	}
}

// generateResponse 응답 생성
func (ms *MockClaudeServer) generateResponse(message string) string {
	ms.responsesMu.RLock()
	defer ms.responsesMu.RUnlock()
	
	// 키워드 매칭
	messageLower := strings.ToLower(message)
	for keyword, response := range ms.responses {
		if strings.Contains(messageLower, keyword) {
			return response
		}
	}
	
	// 기본 응답
	return fmt.Sprintf("메시지를 받았습니다: %s", message)
}

// handleStreamingResponse 스트리밍 응답 처리
func (ms *MockClaudeServer) handleStreamingResponse(w http.ResponseWriter, response string) {
	w.Header().Set("Content-Type", "text/stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}
	
	// 응답을 청크로 나누어 전송
	words := strings.Fields(response)
	for i, word := range words {
		if i > 0 {
			fmt.Fprint(w, " ")
		}
		fmt.Fprint(w, word)
		flusher.Flush()
		
		// 타이핑 효과 시뮬레이션
		time.Sleep(10 * time.Millisecond)
	}
}

// handleRegularResponse 일반 응답 처리
func (ms *MockClaudeServer) handleRegularResponse(w http.ResponseWriter, response string) {
	w.Header().Set("Content-Type", "application/json")
	
	// 사고 시간 시뮬레이션
	if ms.config.SimulateThinking {
		time.Sleep(ms.config.ThinkingTime)
	}
	
	jsonResponse := fmt.Sprintf(`{
		"id": "msg_%d",
		"type": "message",
		"role": "assistant",
		"content": [
			{
				"type": "text",
				"text": %q
			}
		],
		"model": "claude-3-sonnet-20240229",
		"stop_reason": "end_turn",
		"stop_sequence": null,
		"usage": {
			"input_tokens": 10,
			"output_tokens": %d
		}
	}`, time.Now().UnixNano(), response, len(strings.Fields(response)))
	
	fmt.Fprint(w, jsonResponse)
}

// 세션 관리 핸들러들

// handleSessions 세션 목록 핸들러
func (ms *MockClaudeServer) handleSessions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		ms.listSessions(w, r)
	case "POST":
		ms.createSession(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// createSession 세션 생성
func (ms *MockClaudeServer) createSession(w http.ResponseWriter, r *http.Request) {
	sessionID := fmt.Sprintf("session_%d", time.Now().UnixNano())
	
	session := &MockSession{
		ID:           sessionID,
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
		MessageCount: 0,
		Context:      make(map[string]interface{}),
		IsActive:     true,
	}
	
	ms.sessionsMu.Lock()
	ms.sessions[sessionID] = session
	ms.sessionsMu.Unlock()
	
	w.Header().Set("Content-Type", "application/json")
	jsonResponse := fmt.Sprintf(`{
		"id": %q,
		"created_at": %q,
		"status": "active"
	}`, sessionID, session.CreatedAt.Format(time.RFC3339))
	
	fmt.Fprint(w, jsonResponse)
}

// listSessions 세션 목록 조회
func (ms *MockClaudeServer) listSessions(w http.ResponseWriter, r *http.Request) {
	ms.sessionsMu.RLock()
	sessions := make([]*MockSession, 0, len(ms.sessions))
	for _, session := range ms.sessions {
		sessions = append(sessions, session)
	}
	ms.sessionsMu.RUnlock()
	
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, `{"sessions": [`)
	
	for i, session := range sessions {
		if i > 0 {
			fmt.Fprint(w, ",")
		}
		fmt.Fprintf(w, `{
			"id": %q,
			"created_at": %q,
			"last_activity": %q,
			"message_count": %d,
			"is_active": %t
		}`, session.ID, session.CreatedAt.Format(time.RFC3339),
			session.LastActivity.Format(time.RFC3339),
			session.MessageCount, session.IsActive)
	}
	
	fmt.Fprint(w, `]}`)
}

// 관리용 핸들러들

// handleAdminMetrics 메트릭 조회
func (ms *MockClaudeServer) handleAdminMetrics(w http.ResponseWriter, r *http.Request) {
	requestCount := atomic.LoadInt64(&ms.requestCount)
	errorCount := atomic.LoadInt64(&ms.errorCount)
	
	ms.sessionsMu.RLock()
	sessionCount := len(ms.sessions)
	ms.sessionsMu.RUnlock()
	
	w.Header().Set("Content-Type", "application/json")
	jsonResponse := fmt.Sprintf(`{
		"request_count": %d,
		"error_count": %d,
		"session_count": %d,
		"error_rate": %.2f,
		"uptime_seconds": %.0f
	}`, requestCount, errorCount, sessionCount,
		float64(errorCount)/float64(requestCount),
		time.Since(time.Now()).Seconds())
	
	fmt.Fprint(w, jsonResponse)
}

// handleAdminReset 상태 리셋
func (ms *MockClaudeServer) handleAdminReset(w http.ResponseWriter, r *http.Request) {
	atomic.StoreInt64(&ms.requestCount, 0)
	atomic.StoreInt64(&ms.errorCount, 0)
	
	ms.sessionsMu.Lock()
	ms.sessions = make(map[string]*MockSession)
	ms.sessionsMu.Unlock()
	
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"status": "reset"}`)
}

// 유틸리티 메서드들

// simulateLatency 지연 시뮬레이션
func (ms *MockClaudeServer) simulateLatency() {
	if ms.latencyConfig == nil {
		time.Sleep(ms.config.DefaultDelay)
		return
	}
	
	var delay time.Duration
	
	switch ms.latencyConfig.Distribution {
	case "uniform":
		delay = ms.latencyConfig.MinLatency + 
			time.Duration(rand.Int63n(int64(ms.latencyConfig.MaxLatency-ms.latencyConfig.MinLatency)))
	case "normal":
		// 정규분포 근사
		mean := float64(ms.latencyConfig.MinLatency+ms.latencyConfig.MaxLatency) / 2
		delay = time.Duration(mean + rand.NormFloat64()*mean*ms.latencyConfig.Jitter)
	case "exponential":
		// 지수분포
		rate := 1.0 / float64(ms.latencyConfig.MaxLatency)
		delay = time.Duration(rand.ExpFloat64() / rate)
	default:
		delay = ms.config.DefaultDelay
	}
	
	// 지터 적용
	if ms.latencyConfig.Jitter > 0 {
		jitter := time.Duration(float64(delay) * ms.latencyConfig.Jitter * (rand.Float64() - 0.5))
		delay += jitter
	}
	
	// 최소/최대 제한
	if delay < ms.latencyConfig.MinLatency {
		delay = ms.latencyConfig.MinLatency
	}
	if delay > ms.latencyConfig.MaxLatency {
		delay = ms.latencyConfig.MaxLatency
	}
	
	time.Sleep(delay)
}

// shouldInjectError 오류 주입 여부 확인
func (ms *MockClaudeServer) shouldInjectError() bool {
	if ms.errorConfig == nil {
		return rand.Float64() < ms.config.DefaultErrorRate
	}
	
	return rand.Float64() < ms.errorConfig.ErrorRate
}

// handleInjectedError 주입된 오류 처리
func (ms *MockClaudeServer) handleInjectedError(w http.ResponseWriter, r *http.Request) {
	if ms.errorConfig == nil || len(ms.errorConfig.ErrorTypes) == 0 {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	
	// 랜덤 오류 타입 선택
	errorType := ms.errorConfig.ErrorTypes[rand.Intn(len(ms.errorConfig.ErrorTypes))]
	errorMessage := ms.errorConfig.ErrorMessages[errorType]
	
	if errorMessage == "" {
		errorMessage = "Unknown error occurred"
	}
	
	// 오류 타입에 따른 HTTP 상태 코드 설정
	var statusCode int
	switch errorType {
	case "rate_limit_exceeded":
		statusCode = http.StatusTooManyRequests
	case "timeout":
		statusCode = http.StatusRequestTimeout
	case "invalid_request":
		statusCode = http.StatusBadRequest
	default:
		statusCode = http.StatusInternalServerError
	}
	
	http.Error(w, errorMessage, statusCode)
}

// getCurrentRequestCount 현재 요청 수 조회 (근사값)
func (ms *MockClaudeServer) getCurrentRequestCount() int {
	// 실제 구현에서는 요청 시작/종료 시점을 추적해야 함
	// 여기서는 간단히 최근 요청 수로 근사
	return int(atomic.LoadInt64(&ms.requestCount) % int64(ms.config.MaxConcurrentRequests))
}