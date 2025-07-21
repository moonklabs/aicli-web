//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"aicli-web/internal/claude"
	"aicli-web/test/helpers"
)

// TestClaudeWorkflowE2E는 Claude CLI 전체 워크플로우 E2E 테스트를 수행합니다
func TestClaudeWorkflowE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("E2E 테스트는 짧은 테스트 모드에서 생략")
	}
	
	env := helpers.NewTestEnvironment(t)
	client := NewTestAPIClient(env.APIServer.URL)
	
	t.Run("코드 생성 워크플로우", func(t *testing.T) {
		ctx := context.Background()
		
		// 1. 세션 생성
		sessionConfig := SessionCreateRequest{
			SystemPrompt: "You are a helpful code generator assistant",
			Tools:        []string{"Write", "Read", "Bash"},
			MaxTurns:     10,
		}
		
		session, err := client.CreateSession(sessionConfig)
		require.NoError(t, err)
		require.NotEmpty(t, session.ID)
		assert.Equal(t, "ready", session.Status)
		
		// 2. 코드 생성 요청 실행
		executeReq := ExecutionRequest{
			Prompt: "Create a simple Go HTTP server that responds 'Hello World' on port 8080",
		}
		
		execution, err := client.ExecutePrompt(session.ID, executeReq)
		require.NoError(t, err)
		require.NotEmpty(t, execution.ID)
		assert.NotEmpty(t, execution.WebSocketURL)
		
		// 3. WebSocket 연결 및 실시간 스트림 수신
		wsURL := convertToWebSocketURL(env.APIServer.URL, execution.WebSocketURL)
		conn, err := connectWebSocket(wsURL)
		require.NoError(t, err)
		defer conn.Close()
		
		// 4. 메시지 수집 및 검증
		messages := collectWebSocketMessages(t, conn, 30*time.Second)
		
		// 5. 결과 검증
		assert.Greater(t, len(messages), 5, "충분한 메시지를 받아야 함")
		
		// Write 도구 사용 확인
		writeToolUsed := false
		generatedCode := false
		
		for _, msg := range messages {
			if msg.Type == "claude_message" {
				if claudeMsg, ok := msg.Data.(map[string]interface{}); ok {
					if claudeMsg["type"] == "tool_use" && claudeMsg["tool_name"] == "Write" {
						writeToolUsed = true
						
						// 파일 내용 검증
						if input, ok := claudeMsg["input"].(map[string]interface{}); ok {
							if content, ok := input["content"].(string); ok {
								generatedCode = strings.Contains(content, "http.ListenAndServe") &&
									strings.Contains(content, "Hello World")
							}
						}
					}
				}
			}
		}
		
		assert.True(t, writeToolUsed, "Write 도구가 사용되어야 함")
		assert.True(t, generatedCode, "적절한 HTTP 서버 코드가 생성되어야 함")
		
		// 6. 세션 상태 확인
		finalSession, err := client.GetSession(session.ID)
		require.NoError(t, err)
		assert.Equal(t, "completed", finalSession.Status)
	})
	
	t.Run("다중 도구 사용 워크플로우", func(t *testing.T) {
		ctx := context.Background()
		
		// 1. 세션 생성
		sessionConfig := SessionCreateRequest{
			SystemPrompt: "You are a development assistant. Create files and run tests.",
			Tools:        []string{"Write", "Read", "Bash"},
			WorkingDir:   env.CreateWorkspace("multi-tool-test"),
		}
		
		session, err := client.CreateSession(sessionConfig)
		require.NoError(t, err)
		
		// 2. 파일 생성 및 테스트 실행 요청
		executeReq := ExecutionRequest{
			Prompt: "Create a simple calculator function in Go and write a test for it, then run the test",
		}
		
		execution, err := client.ExecutePrompt(session.ID, executeReq)
		require.NoError(t, err)
		
		// 3. WebSocket 연결
		wsURL := convertToWebSocketURL(env.APIServer.URL, execution.WebSocketURL)
		conn, err := connectWebSocket(wsURL)
		require.NoError(t, err)
		defer conn.Close()
		
		// 4. 메시지 수집 (더 긴 타임아웃 - 여러 도구 사용)
		messages := collectWebSocketMessages(t, conn, 60*time.Second)
		
		// 5. 도구 사용 검증
		toolsUsed := make(map[string]bool)
		for _, msg := range messages {
			if msg.Type == "claude_message" {
				if claudeMsg, ok := msg.Data.(map[string]interface{}); ok {
					if claudeMsg["type"] == "tool_use" {
						if toolName, ok := claudeMsg["tool_name"].(string); ok {
							toolsUsed[toolName] = true
						}
					}
				}
			}
		}
		
		assert.True(t, toolsUsed["Write"], "파일 작성을 위해 Write 도구가 사용되어야 함")
		assert.True(t, toolsUsed["Bash"], "테스트 실행을 위해 Bash 도구가 사용되어야 함")
		
		// 적어도 2개의 Write 호출이 있어야 함 (main 파일 + test 파일)
		writeCount := 0
		for _, msg := range messages {
			if msg.Type == "claude_message" {
				if claudeMsg, ok := msg.Data.(map[string]interface{}); ok {
					if claudeMsg["type"] == "tool_use" && claudeMsg["tool_name"] == "Write" {
						writeCount++
					}
				}
			}
		}
		assert.GreaterOrEqual(t, writeCount, 2, "최소 2개의 파일이 생성되어야 함")
	})
	
	t.Run("에러 복구 워크플로우", func(t *testing.T) {
		ctx := context.Background()
		
		// 1. 세션 생성
		sessionConfig := SessionCreateRequest{
			SystemPrompt: "You are a helpful assistant",
			Tools:        []string{"Write", "Bash"},
		}
		
		session, err := client.CreateSession(sessionConfig)
		require.NoError(t, err)
		
		// 2. 의도적으로 실패하는 명령어 실행 요청
		executeReq := ExecutionRequest{
			Prompt: "Run the command 'nonexistent-command-12345' and handle any errors",
		}
		
		execution, err := client.ExecutePrompt(session.ID, executeReq)
		require.NoError(t, err)
		
		// 3. WebSocket 연결
		wsURL := convertToWebSocketURL(env.APIServer.URL, execution.WebSocketURL)
		conn, err := connectWebSocket(wsURL)
		require.NoError(t, err)
		defer conn.Close()
		
		// 4. 메시지 수집
		messages := collectWebSocketMessages(t, conn, 30*time.Second)
		
		// 5. 에러 처리 확인
		errorHandled := false
		for _, msg := range messages {
			if msg.Type == "claude_message" {
				if claudeMsg, ok := msg.Data.(map[string]interface{}); ok {
					if content, ok := claudeMsg["content"].(string); ok {
						if strings.Contains(strings.ToLower(content), "error") ||
							strings.Contains(strings.ToLower(content), "failed") ||
							strings.Contains(strings.ToLower(content), "command not found") {
							errorHandled = true
						}
					}
				}
			}
		}
		
		assert.True(t, errorHandled, "에러가 적절히 처리되고 보고되어야 함")
		
		// 6. 세션이 여전히 활성 상태인지 확인 (복구됨)
		finalSession, err := client.GetSession(session.ID)
		require.NoError(t, err)
		assert.NotEqual(t, "error", finalSession.Status, "세션이 에러 상태가 아니어야 함")
	})
}

// TestConcurrentSessions는 동시 세션 처리 E2E 테스트를 수행합니다
func TestConcurrentSessions(t *testing.T) {
	if testing.Short() {
		t.Skip("E2E 테스트는 짧은 테스트 모드에서 생략")
	}
	
	env := helpers.NewTestEnvironment(t)
	client := NewTestAPIClient(env.APIServer.URL)
	
	const numSessions = 3
	sessions := make([]*SessionResponse, numSessions)
	
	// 1. 동시에 여러 세션 생성
	for i := 0; i < numSessions; i++ {
		sessionConfig := SessionCreateRequest{
			SystemPrompt: fmt.Sprintf("You are assistant #%d", i+1),
			Tools:        []string{"Write"},
			WorkingDir:   env.CreateWorkspace(fmt.Sprintf("concurrent-test-%d", i)),
		}
		
		session, err := client.CreateSession(sessionConfig)
		require.NoError(t, err)
		sessions[i] = session
	}
	
	// 2. 각 세션에서 동시에 작업 실행
	results := make(chan error, numSessions)
	
	for i, session := range sessions {
		go func(sessionID string, index int) {
			executeReq := ExecutionRequest{
				Prompt: fmt.Sprintf("Create a file named 'session_%d.txt' with content 'Hello from session %d'", index, index),
			}
			
			execution, err := client.ExecutePrompt(sessionID, executeReq)
			if err != nil {
				results <- err
				return
			}
			
			// WebSocket 연결 및 완료 대기
			wsURL := convertToWebSocketURL(env.APIServer.URL, execution.WebSocketURL)
			conn, err := connectWebSocket(wsURL)
			if err != nil {
				results <- err
				return
			}
			defer conn.Close()
			
			_ = collectWebSocketMessages(t, conn, 20*time.Second)
			results <- nil
		}(session.ID, i)
	}
	
	// 3. 모든 작업 완료 대기
	successCount := 0
	for i := 0; i < numSessions; i++ {
		err := <-results
		if err == nil {
			successCount++
		} else {
			t.Errorf("세션 %d 실행 실패: %v", i, err)
		}
	}
	
	assert.Equal(t, numSessions, successCount, "모든 동시 세션이 성공해야 함")
}

// TestAPIClient는 API 클라이언트 구조체입니다
type TestAPIClient struct {
	baseURL    string
	httpClient *http.Client
}

type SessionCreateRequest struct {
	SystemPrompt string   `json:"system_prompt"`
	Tools        []string `json:"tools"`
	WorkingDir   string   `json:"working_dir,omitempty"`
	MaxTurns     int      `json:"max_turns,omitempty"`
}

type SessionResponse struct {
	ID         string    `json:"id"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	SystemPrompt string  `json:"system_prompt"`
}

type ExecutionRequest struct {
	Prompt string `json:"prompt"`
}

type ExecutionResponse struct {
	ID           string `json:"id"`
	SessionID    string `json:"session_id"`
	Status       string `json:"status"`
	WebSocketURL string `json:"websocket_url"`
}

type WebSocketMessage struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}

// NewTestAPIClient는 새로운 테스트 API 클라이언트를 생성합니다
func NewTestAPIClient(baseURL string) *TestAPIClient {
	return &TestAPIClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CreateSession은 새로운 세션을 생성합니다
func (c *TestAPIClient) CreateSession(config SessionCreateRequest) (*SessionResponse, error) {
	reqBody, _ := json.Marshal(config)
	
	resp, err := http.Post(c.baseURL+"/api/v1/sessions", "application/json", strings.NewReader(string(reqBody)))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("세션 생성 실패: status %d", resp.StatusCode)
	}
	
	var session SessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		return nil, err
	}
	
	return &session, nil
}

// GetSession은 세션 정보를 조회합니다
func (c *TestAPIClient) GetSession(sessionID string) (*SessionResponse, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/api/v1/sessions/" + sessionID)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("세션 조회 실패: status %d", resp.StatusCode)
	}
	
	var session SessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		return nil, err
	}
	
	return &session, nil
}

// ExecutePrompt는 프롬프트를 실행합니다
func (c *TestAPIClient) ExecutePrompt(sessionID string, req ExecutionRequest) (*ExecutionResponse, error) {
	reqBody, _ := json.Marshal(req)
	
	url := fmt.Sprintf("%s/api/v1/sessions/%s/execute", c.baseURL, sessionID)
	resp, err := http.Post(url, "application/json", strings.NewReader(string(reqBody)))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("프롬프트 실행 실패: status %d", resp.StatusCode)
	}
	
	var execution ExecutionResponse
	if err := json.NewDecoder(resp.Body).Decode(&execution); err != nil {
		return nil, err
	}
	
	return &execution, nil
}

// convertToWebSocketURL은 HTTP URL을 WebSocket URL로 변환합니다
func convertToWebSocketURL(baseURL, wsPath string) string {
	u, _ := url.Parse(baseURL)
	
	scheme := "ws"
	if u.Scheme == "https" {
		scheme = "wss"
	}
	
	return fmt.Sprintf("%s://%s%s", scheme, u.Host, wsPath)
}

// connectWebSocket은 WebSocket에 연결합니다
func connectWebSocket(wsURL string) (*websocket.Conn, error) {
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}
	
	conn, _, err := dialer.Dial(wsURL, nil)
	return conn, err
}

// collectWebSocketMessages는 WebSocket 메시지를 수집합니다
func collectWebSocketMessages(t *testing.T, conn *websocket.Conn, timeout time.Duration) []WebSocketMessage {
	var messages []WebSocketMessage
	deadline := time.Now().Add(timeout)
	
	conn.SetReadDeadline(deadline)
	
	for time.Now().Before(deadline) {
		var msg WebSocketMessage
		err := conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				break // 정상 종료
			}
			t.Logf("WebSocket 읽기 에러: %v", err)
			break
		}
		
		messages = append(messages, msg)
		
		// 완료 메시지 수신 시 종료
		if msg.Type == "completion" {
			break
		}
	}
	
	return messages
}