package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/aicli/aicli-web/internal/claude"
	"github.com/aicli/aicli-web/internal/storage"
	"github.com/aicli/aicli-web/internal/websocket"
)

// ClaudeHandler는 Claude 관련 API 요청을 처리합니다.
type ClaudeHandler struct {
	claudeWrapper claude.Wrapper
	sessionStore  storage.SessionRepository
	wsHub         *websocket.Hub
}

// NewClaudeHandler는 새로운 Claude 핸들러를 생성합니다.
func NewClaudeHandler(wrapper claude.Wrapper, store storage.SessionRepository, hub *websocket.Hub) *ClaudeHandler {
	return &ClaudeHandler{
		claudeWrapper: wrapper,
		sessionStore:  store,
		wsHub:         hub,
	}
}

// ExecuteRequest는 Claude 실행 요청 구조체입니다.
type ExecuteRequest struct {
	WorkspaceID  string                 `json:"workspace_id" binding:"required"`
	Prompt       string                 `json:"prompt" binding:"required"`
	SystemPrompt string                 `json:"system_prompt,omitempty"`
	MaxTurns     int                    `json:"max_turns,omitempty"`
	Tools        []string               `json:"tools,omitempty"`
	Stream       bool                   `json:"stream"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// ExecuteResponse는 Claude 실행 응답 구조체입니다.
type ExecuteResponse struct {
	ExecutionID  string `json:"execution_id"`
	SessionID    string `json:"session_id"`
	Status       string `json:"status"`
	WebSocketURL string `json:"websocket_url,omitempty"`
}

// Execute는 Claude 실행 요청을 처리합니다.
func (h *ClaudeHandler) Execute(c *gin.Context) {
	var req ExecuteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	// 세션 생성 또는 재사용
	session, err := h.getOrCreateSession(c, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create or get session",
			"details": err.Error(),
		})
		return
	}

	// 실행 ID 생성
	executionID := uuid.New().String()

	// 비동기 실행 시작
	go h.executeAsync(c.Request.Context(), session, req, executionID)

	// 즉시 응답 반환
	response := ExecuteResponse{
		ExecutionID: executionID,
		SessionID:   session.ID,
		Status:      "started",
	}

	// 스트림 모드인 경우 WebSocket URL 추가
	if req.Stream {
		response.WebSocketURL = fmt.Sprintf("/ws/executions/%s", executionID)
	}

	c.JSON(http.StatusAccepted, response)
}

// getOrCreateSession은 세션을 생성하거나 기존 세션을 가져옵니다.
func (h *ClaudeHandler) getOrCreateSession(c *gin.Context, req ExecuteRequest) (*claude.Session, error) {
	// 기존 활성 세션 검색
	sessions, err := h.sessionStore.FindByWorkspace(req.WorkspaceID)
	if err != nil {
		return nil, err
	}

	// 활성 세션 중에서 재사용 가능한 세션 찾기
	for _, session := range sessions {
		if session.Status == "idle" && session.WorkspaceID == req.WorkspaceID {
			// 세션 설정 업데이트
			if req.SystemPrompt != "" {
				session.SystemPrompt = req.SystemPrompt
			}
			if req.MaxTurns > 0 {
				session.MaxTurns = req.MaxTurns
			}
			return session, nil
		}
	}

	// 새 세션 생성
	config := &claude.SessionConfig{
		WorkingDir:   "/tmp", // 기본 작업 디렉토리
		SystemPrompt: req.SystemPrompt,
		MaxTurns:     req.MaxTurns,
		AllowedTools: req.Tools,
		Temperature:  0.7, // 기본값
	}

	if config.MaxTurns == 0 {
		config.MaxTurns = 10 // 기본값
	}

	session, err := h.claudeWrapper.CreateSession(config)
	if err != nil {
		return nil, err
	}

	return session, nil
}

// executeAsync는 Claude 명령을 비동기적으로 실행합니다.
func (h *ClaudeHandler) executeAsync(ctx context.Context, session *claude.Session, req ExecuteRequest, executionID string) {
	defer func() {
		if r := recover(); r != nil {
			// 패닉 복구 - WebSocket을 통해 에러 전송
			if h.wsHub != nil {
				errorMsg := websocket.Message{
					Type: "execution_error",
					Data: map[string]interface{}{
						"execution_id": executionID,
						"error":        fmt.Sprintf("Execution panic: %v", r),
						"timestamp":    time.Now(),
					},
				}
				h.wsHub.Broadcast <- errorMsg
			}
		}
	}()

	// Claude 실행
	result, err := h.claudeWrapper.Execute(session.ID, req.Prompt)
	if err != nil {
		// 에러 메시지를 WebSocket으로 전송
		if h.wsHub != nil && req.Stream {
			errorMsg := websocket.Message{
				Type: "execution_error",
				Data: map[string]interface{}{
					"execution_id": executionID,
					"session_id":   session.ID,
					"error":        err.Error(),
					"timestamp":    time.Now(),
				},
			}
			h.wsHub.Broadcast <- errorMsg
		}
		return
	}

	// 성공 결과를 WebSocket으로 전송
	if h.wsHub != nil && req.Stream {
		successMsg := websocket.Message{
			Type: "execution_complete",
			Data: map[string]interface{}{
				"execution_id": executionID,
				"session_id":   session.ID,
				"result":       result,
				"timestamp":    time.Now(),
			},
		}
		h.wsHub.Broadcast <- successMsg
	}
}

// ListSessions는 세션 목록을 조회합니다.
func (h *ClaudeHandler) ListSessions(c *gin.Context) {
	workspaceID := c.Query("workspace_id")
	if workspaceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "workspace_id parameter is required",
		})
		return
	}

	// SessionManager를 통해 세션 조회
	filter := claude.SessionFilter{
		WorkspaceID: workspaceID,
	}
	
	sessions, err := h.claudeWrapper.ListSessions(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve sessions",
			"details": err.Error(),
		})
		return
	}

	// 응답 변환
	response := make([]SessionResponse, len(sessions))
	for i, session := range sessions {
		response[i] = toSessionResponse(session)
	}

	c.JSON(http.StatusOK, gin.H{
		"sessions": response,
		"total":    len(response),
	})
}

// GetSession은 특정 세션을 조회합니다.
func (h *ClaudeHandler) GetSession(c *gin.Context) {
	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "session id is required",
		})
		return
	}

	session, err := h.claudeWrapper.GetSession(sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Session not found",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, toSessionResponse(session))
}

// CloseSession은 세션을 종료합니다.
func (h *ClaudeHandler) CloseSession(c *gin.Context) {
	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "session id is required",
		})
		return
	}

	if err := h.claudeWrapper.CloseSession(sessionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to close session",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Session closed",
		"session_id": sessionID,
	})
}

// GetSessionLogs는 세션 로그를 조회합니다.
func (h *ClaudeHandler) GetSessionLogs(c *gin.Context) {
	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "session id is required",
		})
		return
	}

	limit := c.DefaultQuery("limit", "100")
	limitInt, err := strconv.Atoi(limit)
	if err != nil || limitInt < 1 || limitInt > 1000 {
		limitInt = 100
	}

	// 로그 조회
	logs, err := h.getSessionLogs(sessionID, limitInt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve logs",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"session_id": sessionID,
		"logs":       logs,
		"count":      len(logs),
	})
}

// getSessionLogs는 세션 로그를 가져옵니다.
func (h *ClaudeHandler) getSessionLogs(sessionID string, limit int) ([]LogEntry, error) {
	// TODO: 실제 로그 조회 로직 구현
	// 현재는 더미 데이터 반환
	logs := []LogEntry{
		{
			Timestamp: time.Now().Add(-time.Hour),
			Level:     "info",
			Message:   "Session started",
			SessionID: sessionID,
		},
		{
			Timestamp: time.Now().Add(-time.Minute * 30),
			Level:     "info",
			Message:   "Command executed successfully",
			SessionID: sessionID,
		},
	}

	if len(logs) > limit {
		logs = logs[:limit]
	}

	return logs, nil
}

// SessionResponse는 세션 응답 구조체입니다.
type SessionResponse struct {
	ID           string                 `json:"id"`
	WorkspaceID  string                 `json:"workspace_id"`
	Status       string                 `json:"status"`
	SystemPrompt string                 `json:"system_prompt,omitempty"`
	MaxTurns     int                    `json:"max_turns"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// LogEntry는 로그 엔트리 구조체입니다.
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	SessionID string    `json:"session_id"`
	Details   string    `json:"details,omitempty"`
}

// toSessionResponse는 Claude 세션을 응답 구조체로 변환합니다.
func toSessionResponse(session *claude.Session) SessionResponse {
	return SessionResponse{
		ID:           session.ID,
		WorkspaceID:  session.WorkspaceID,
		Status:       session.State.Status,
		SystemPrompt: session.Config.SystemPrompt,
		MaxTurns:     session.Config.MaxTurns,
		CreatedAt:    session.Created,
		UpdatedAt:    session.LastActive,
		Metadata:     session.Metadata,
	}
}