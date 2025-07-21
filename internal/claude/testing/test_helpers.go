// Package testing provides test utilities for Claude integration tests.
package testing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/drumcap/aicli-web/internal/claude"
)

// TestEnvironment는 통합 테스트를 위한 환경을 제공합니다.
type TestEnvironment struct {
	TempDir      string
	MockClaude   *MockClaudeServer
	RealClaude   bool // 실제 Claude CLI 사용 여부
	TestData     *TestDataProvider
	cleanup      []func()
}

// NewTestEnvironment는 새로운 테스트 환경을 생성합니다.
func NewTestEnvironment(t *testing.T) *TestEnvironment {
	env := &TestEnvironment{
		TempDir:    t.TempDir(),
		RealClaude: os.Getenv("TEST_REAL_CLAUDE") == "true",
		TestData:   NewTestDataProvider(),
	}

	if !env.RealClaude {
		// Mock Claude 서버 시작
		env.MockClaude = NewMockClaudeServer()
		env.addCleanup(env.MockClaude.Stop)
	}

	// 테스트 종료 시 정리
	t.Cleanup(env.Cleanup)

	return env
}

// addCleanup은 정리 함수를 추가합니다.
func (env *TestEnvironment) addCleanup(fn func()) {
	env.cleanup = append(env.cleanup, fn)
}

// Cleanup은 테스트 환경을 정리합니다.
func (env *TestEnvironment) Cleanup() {
	for i := len(env.cleanup) - 1; i >= 0; i-- {
		env.cleanup[i]()
	}
}

// MockClaudeServer는 Claude CLI를 모방하는 Mock 서버입니다.
type MockClaudeServer struct {
	server    *httptest.Server
	responses map[string][]byte
	requests  []MockRequest
	mu        sync.Mutex
}

// MockRequest는 Mock 서버에 들어온 요청을 기록합니다.
type MockRequest struct {
	Method    string
	Path      string
	Body      string
	Headers   map[string]string
	Timestamp time.Time
}

// NewMockClaudeServer는 새로운 Mock Claude 서버를 생성합니다.
func NewMockClaudeServer() *MockClaudeServer {
	mock := &MockClaudeServer{
		responses: make(map[string][]byte),
		requests:  make([]MockRequest, 0),
	}

	// HTTP 서버 설정
	mock.server = httptest.NewServer(http.HandlerFunc(mock.handleRequest))

	// 기본 응답 설정
	mock.SetResponse("POST /chat", mock.createChatResponse("Hello! How can I help you?"))
	mock.SetResponse("GET /status", mock.createStatusResponse())

	return mock
}

// handleRequest는 HTTP 요청을 처리합니다.
func (m *MockClaudeServer) handleRequest(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 요청 기록
	body, _ := io.ReadAll(r.Body)
	r.Body.Close()

	headers := make(map[string]string)
	for key, values := range r.Header {
		headers[key] = strings.Join(values, ", ")
	}

	request := MockRequest{
		Method:    r.Method,
		Path:      r.URL.Path,
		Body:      string(body),
		Headers:   headers,
		Timestamp: time.Now(),
	}
	m.requests = append(m.requests, request)

	// 응답 찾기
	key := fmt.Sprintf("%s %s", r.Method, r.URL.Path)
	if response, exists := m.responses[key]; exists {
		w.Header().Set("Content-Type", "application/json")
		w.Write(response)
	} else {
		// 기본 응답
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"Not found"}`))
	}
}

// SetResponse는 특정 패턴에 대한 응답을 설정합니다.
func (m *MockClaudeServer) SetResponse(pattern string, response []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responses[pattern] = response
}

// GetRequests는 서버에 들어온 모든 요청을 반환합니다.
func (m *MockClaudeServer) GetRequests() []MockRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	requests := make([]MockRequest, len(m.requests))
	copy(requests, m.requests)
	return requests
}

// ClearRequests는 요청 기록을 지웁니다.
func (m *MockClaudeServer) ClearRequests() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.requests = m.requests[:0]
}

// Stop은 Mock 서버를 중지합니다.
func (m *MockClaudeServer) Stop() {
	if m.server != nil {
		m.server.Close()
	}
}

// URL은 Mock 서버의 URL을 반환합니다.
func (m *MockClaudeServer) URL() string {
	if m.server != nil {
		return m.server.URL
	}
	return ""
}

// createChatResponse는 채팅 응답을 생성합니다.
func (m *MockClaudeServer) createChatResponse(content string) []byte {
	response := map[string]interface{}{
		"type":    "text",
		"content": content,
		"id":      fmt.Sprintf("mock-%d", time.Now().UnixNano()),
		"meta": map[string]interface{}{
			"timestamp": time.Now().Format(time.RFC3339),
			"model":     "claude-3-sonnet-mock",
		},
	}
	
	data, _ := json.Marshal(response)
	return data
}

// createStatusResponse는 상태 응답을 생성합니다.
func (m *MockClaudeServer) createStatusResponse() []byte {
	response := map[string]interface{}{
		"status":  "healthy",
		"version": "mock-1.0.0",
		"uptime":  "1h30m",
		"sessions": map[string]interface{}{
			"active": 0,
			"total":  0,
		},
	}
	
	data, _ := json.Marshal(response)
	return data
}

// TestDataProvider는 테스트 데이터를 제공합니다.
type TestDataProvider struct {
	streamData map[string][]byte
}

// NewTestDataProvider는 새로운 테스트 데이터 제공자를 생성합니다.
func NewTestDataProvider() *TestDataProvider {
	provider := &TestDataProvider{
		streamData: make(map[string][]byte),
	}
	
	// 미리 정의된 테스트 데이터 로드
	provider.loadStreamData()
	
	return provider
}

// loadStreamData는 스트림 테스트 데이터를 로드합니다.
func (tdp *TestDataProvider) loadStreamData() {
	// 복잡한 응답 시나리오
	complexResponse := `{"type":"text","content":"I'll help you with that task.","id":"msg1"}
{"type":"tool_use","content":"Reading file...","id":"msg2","meta":{"tool":"Read","file":"test.go"}}
{"type":"tool_result","content":"File content here","id":"msg3","meta":{"success":true}}
{"type":"text","content":"Based on the file content, I can see...","id":"msg4"}
{"type":"text","content":"Here's my analysis...","id":"msg5"}
{"type":"system","content":"Task completed successfully","id":"msg6"}`

	tdp.streamData["complex_response.jsonl"] = []byte(complexResponse)
	
	// 에러 시나리오
	errorResponse := `{"type":"text","content":"Starting task...","id":"msg1"}
{"type":"error","content":"Permission denied","id":"msg2","meta":{"error_code":403}}
{"type":"system","content":"Task failed","id":"msg3"}`

	tdp.streamData["error_response.jsonl"] = []byte(errorResponse)
	
	// 대용량 응답
	var largeBuilder strings.Builder
	for i := 0; i < 100; i++ {
		largeBuilder.WriteString(fmt.Sprintf(`{"type":"text","content":"Line %d of large response","id":"msg%d"}`+"\n", i+1, i+1))
	}
	tdp.streamData["large_response.jsonl"] = []byte(largeBuilder.String())
}

// LoadStreamData는 지정된 이름의 스트림 데이터를 반환합니다.
func (tdp *TestDataProvider) LoadStreamData(name string) []byte {
	if data, exists := tdp.streamData[name]; exists {
		return data
	}
	return nil
}

// CreateTestMessage는 테스트용 메시지를 생성합니다.
func (tdp *TestDataProvider) CreateTestMessage(msgType, content string) claude.Message {
	return claude.Message{
		Type:    msgType,
		Content: content,
		ID:      fmt.Sprintf("test-%d", time.Now().UnixNano()),
		Meta: map[string]interface{}{
			"timestamp": time.Now(),
			"test":      true,
		},
	}
}

// AssertionHelpers는 테스트 검증을 위한 헬퍼 함수들을 제공합니다.
type AssertionHelpers struct {
	t *testing.T
}

// NewAssertionHelpers는 새로운 검증 헬퍼를 생성합니다.
func NewAssertionHelpers(t *testing.T) *AssertionHelpers {
	return &AssertionHelpers{t: t}
}

// AssertMessageTypes는 메시지 타입 순서를 검증합니다.
func (ah *AssertionHelpers) AssertMessageTypes(messages []claude.Message, expected []string) {
	if len(messages) != len(expected) {
		ah.t.Errorf("Expected %d messages, got %d", len(expected), len(messages))
		return
	}
	
	for i, msg := range messages {
		if msg.Type != expected[i] {
			ah.t.Errorf("Message %d: expected type %s, got %s", i, expected[i], msg.Type)
		}
	}
}

// AssertContainsToolUse는 메시지 목록에 특정 도구 사용이 포함되어 있는지 확인합니다.
func (ah *AssertionHelpers) AssertContainsToolUse(messages []claude.Message, toolName string) {
	for _, msg := range messages {
		if msg.Type == "tool_use" {
			if tool, ok := msg.Meta["tool"].(string); ok && tool == toolName {
				return
			}
		}
	}
	ah.t.Errorf("Expected tool use '%s' not found in messages", toolName)
}

// AssertGeneratedCode는 메시지에 특정 코드가 포함되어 있는지 확인합니다.
func (ah *AssertionHelpers) AssertGeneratedCode(messages []claude.Message, expectedCode string) {
	for _, msg := range messages {
		if strings.Contains(msg.Content, expectedCode) {
			return
		}
	}
	ah.t.Errorf("Expected code '%s' not found in any message", expectedCode)
}

// WaitForResponse는 응답을 기다리는 헬퍼 함수입니다.
func WaitForResponse(t *testing.T, ctx context.Context, messageChan <-chan claude.Message) string {
	var response strings.Builder
	
	for {
		select {
		case <-ctx.Done():
			t.Fatal("Timeout waiting for response")
			return ""
		case msg, ok := <-messageChan:
			if !ok {
				return response.String()
			}
			if msg.Type == "text" {
				response.WriteString(msg.Content)
			}
		}
	}
}

// CreateTestProcess는 테스트용 프로세스를 생성합니다.
func CreateTestProcess(t *testing.T, env *TestEnvironment) *TestProcess {
	return &TestProcess{
		t:        t,
		env:      env,
		messages: make(chan claude.Message, 100),
		closed:   make(chan struct{}),
	}
}

// TestProcess는 테스트용 프로세스를 나타냅니다.
type TestProcess struct {
	t        *testing.T
	env      *TestEnvironment
	messages chan claude.Message
	closed   chan struct{}
	state    string
}

// SendPrompt는 프롬프트를 전송합니다 (시뮬레이션).
func (tp *TestProcess) SendPrompt(prompt string) error {
	tp.state = "processing"
	
	// 시뮬레이션 응답 생성
	go func() {
		time.Sleep(100 * time.Millisecond)
		
		// 응답 메시지들 전송
		responses := []claude.Message{
			{Type: "text", Content: "I understand your request.", ID: "sim1"},
			{Type: "text", Content: "Processing...", ID: "sim2"},
			{Type: "text", Content: "Task completed.", ID: "sim3"},
		}
		
		for _, msg := range responses {
			select {
			case tp.messages <- msg:
			case <-tp.closed:
				return
			}
			time.Sleep(50 * time.Millisecond)
		}
		
		close(tp.messages)
		tp.state = "completed"
	}()
	
	return nil
}

// StreamOutput는 출력 스트림을 반환합니다.
func (tp *TestProcess) StreamOutput(ctx context.Context) <-chan claude.Message {
	return tp.messages
}

// State는 현재 상태를 반환합니다.
func (tp *TestProcess) State() string {
	return tp.state
}

// Close는 프로세스를 종료합니다.
func (tp *TestProcess) Close() error {
	select {
	case <-tp.closed:
		return nil // already closed
	default:
		close(tp.closed)
		tp.state = "closed"
		return nil
	}
}

// CreateMockPipe는 테스트용 파이프를 생성합니다.
func CreateMockPipe() *MockPipe {
	return &MockPipe{
		buffer: &bytes.Buffer{},
		closed: make(chan struct{}),
	}
}

// MockPipe는 테스트용 파이프입니다.
type MockPipe struct {
	buffer *bytes.Buffer
	closed chan struct{}
	mu     sync.Mutex
}

// Read는 파이프에서 데이터를 읽습니다.
func (mp *MockPipe) Read(p []byte) (n int, err error) {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	
	select {
	case <-mp.closed:
		return 0, io.EOF
	default:
		return mp.buffer.Read(p)
	}
}

// Write는 파이프에 데이터를 씁니다.
func (mp *MockPipe) Write(p []byte) (n int, err error) {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	
	select {
	case <-mp.closed:
		return 0, io.ErrClosedPipe
	default:
		return mp.buffer.Write(p)
	}
}

// Close는 파이프를 닫습니다.
func (mp *MockPipe) Close() error {
	select {
	case <-mp.closed:
		return nil
	default:
		close(mp.closed)
		return nil
	}
}

// WriteString은 문자열을 파이프에 씁니다.
func (mp *MockPipe) WriteString(s string) (n int, err error) {
	return mp.Write([]byte(s))
}