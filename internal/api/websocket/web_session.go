package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/aicli/aicli-web/internal/claude"
	"github.com/aicli/aicli-web/internal/auth"
	"github.com/aicli/aicli-web/internal/storage"
)

// WebSessionController는 웹 세션 관련 REST API를 처리합니다
type WebSessionController struct {
	sessionManager   claude.SessionManager
	streamHandler    *ClaudeStreamHandler
	connectionManager *ConnectionManager
	fileManager      *FileManager
	storage          storage.Storage
	authValidator    *auth.Validator
}

// CreateSessionRequest는 세션 생성 요청입니다
type CreateSessionRequest struct {
	Name         string                 `json:"name" binding:"required"`
	WorkspaceID  string                 `json:"workspace_id" binding:"required"`
	SystemPrompt string                 `json:"system_prompt,omitempty"`
	MaxTurns     int                    `json:"max_turns,omitempty"`
	Options      map[string]interface{} `json:"options,omitempty"`
	IsPrivate    bool                   `json:"is_private,omitempty"`
	Tags         []string               `json:"tags,omitempty"`
}

// SessionResponse는 세션 응답입니다
type SessionResponse struct {
	SessionID     string             `json:"session_id"`
	Name          string             `json:"name"`
	WorkspaceID   string             `json:"workspace_id"`
	Status        string             `json:"status"`
	CreatedAt     time.Time          `json:"created_at"`
	UpdatedAt     time.Time          `json:"updated_at"`
	IsPrivate     bool               `json:"is_private"`
	Tags          []string           `json:"tags"`
	Participants  []SessionUser      `json:"participants"`
	ConnectionURL string             `json:"connection_url"`
	ShareToken    string             `json:"share_token,omitempty"`
	Statistics    SessionStatistics  `json:"statistics"`
}

// SessionStatistics는 세션 통계입니다
type SessionStatistics struct {
	MessageCount      int           `json:"message_count"`
	ParticipantCount  int           `json:"participant_count"`
	ActiveConnections int           `json:"active_connections"`
	LastActivity      time.Time     `json:"last_activity"`
	TotalDuration     time.Duration `json:"total_duration"`
	AverageResponseTime time.Duration `json:"average_response_time"`
}

// UpdateSessionRequest는 세션 업데이트 요청입니다
type UpdateSessionRequest struct {
	Name         *string                `json:"name,omitempty"`
	SystemPrompt *string                `json:"system_prompt,omitempty"`
	MaxTurns     *int                   `json:"max_turns,omitempty"`
	Options      map[string]interface{} `json:"options,omitempty"`
	IsPrivate    *bool                  `json:"is_private,omitempty"`
	Tags         []string               `json:"tags,omitempty"`
}

// SessionListRequest는 세션 목록 요청입니다
type SessionListRequest struct {
	WorkspaceID string `form:"workspace_id"`
	Status      string `form:"status"`
	Tags        string `form:"tags"`
	IsPrivate   *bool  `form:"is_private"`
	Limit       int    `form:"limit"`
	Offset      int    `form:"offset"`
	SortBy      string `form:"sort_by"`
	SortOrder   string `form:"sort_order"`
}

// MessageSendRequest는 메시지 전송 요청입니다
type MessageSendRequest struct {
	Message string                 `json:"message" binding:"required"`
	Type    string                 `json:"type,omitempty"`
	Data    map[string]interface{} `json:"data,omitempty"`
}

// InviteUserRequest는 사용자 초대 요청입니다
type InviteUserRequest struct {
	UserID     string     `json:"user_id,omitempty"`
	Email      string     `json:"email,omitempty"`
	Permission Permission `json:"permission" binding:"required"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	Message    string     `json:"message,omitempty"`
}

// NewWebSessionController는 새로운 웹 세션 컨트롤러를 생성합니다
func NewWebSessionController(
	sessionManager claude.SessionManager,
	streamHandler *ClaudeStreamHandler,
	connectionManager *ConnectionManager,
	fileManager *FileManager,
	storage storage.Storage,
	authValidator *auth.Validator,
) *WebSessionController {
	return &WebSessionController{
		sessionManager:    sessionManager,
		streamHandler:     streamHandler,
		connectionManager: connectionManager,
		fileManager:       fileManager,
		storage:           storage,
		authValidator:     authValidator,
	}
}

// CreateSession은 새로운 세션을 생성합니다
func (c *WebSessionController) CreateSession(ctx *gin.Context) {
	var req CreateSessionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 사용자 인증 확인
	userInfo, err := c.authValidator.ValidateGinContext(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "인증이 필요합니다"})
		return
	}

	// 세션 설정 생성
	sessionConfig := claude.SessionConfig{
		Name:         req.Name,
		WorkspaceID:  req.WorkspaceID,
		SystemPrompt: req.SystemPrompt,
		MaxTurns:     req.MaxTurns,
		UserID:       userInfo.UserID,
		Options:      req.Options,
	}

	// 기본값 설정
	if sessionConfig.MaxTurns == 0 {
		sessionConfig.MaxTurns = 100
	}

	// 세션 생성
	session, err := c.sessionManager.CreateSession(ctx.Request.Context(), sessionConfig)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "세션 생성에 실패했습니다: " + err.Error()})
		return
	}

	// 웹 세션 메타데이터 저장
	webSessionData := map[string]interface{}{
		"session_id":     session.ID,
		"name":           req.Name,
		"workspace_id":   req.WorkspaceID,
		"creator_id":     userInfo.UserID,
		"is_private":     req.IsPrivate,
		"tags":           req.Tags,
		"created_at":     time.Now(),
		"updated_at":     time.Now(),
		"status":         "active",
	}

	// 스토리지에 저장 (실제 구현에서는 스토리지 스키마에 맞게 조정)
	// 여기서는 임시로 JSON 형태로 저장
	if err := c.saveWebSessionMetadata(session.ID, webSessionData); err != nil {
		// 세션 정리
		c.sessionManager.CloseSession(session.ID)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "세션 메타데이터 저장에 실패했습니다"})
		return
	}

	// 응답 생성
	response := &SessionResponse{
		SessionID:     session.ID,
		Name:          req.Name,
		WorkspaceID:   req.WorkspaceID,
		Status:        "active",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		IsPrivate:     req.IsPrivate,
		Tags:          req.Tags,
		Participants:  []SessionUser{},
		ConnectionURL: fmt.Sprintf("/ws/session/%s", session.ID),
		Statistics: SessionStatistics{
			MessageCount:        0,
			ParticipantCount:    0,
			ActiveConnections:   0,
			LastActivity:        time.Now(),
			TotalDuration:       0,
			AverageResponseTime: 0,
		},
	}

	// 비공개가 아닌 경우 공유 토큰 생성
	if !req.IsPrivate {
		shareToken := generateShareToken(session.ID)
		response.ShareToken = shareToken
	}

	ctx.JSON(http.StatusCreated, response)
}

// GetSession은 세션 정보를 조회합니다
func (c *WebSessionController) GetSession(ctx *gin.Context) {
	sessionID := ctx.Param("id")
	if sessionID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "session_id가 필요합니다"})
		return
	}

	// 사용자 인증 확인
	userInfo, err := c.authValidator.ValidateGinContext(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "인증이 필요합니다"})
		return
	}

	// 세션 조회
	session, err := c.sessionManager.GetSession(ctx.Request.Context(), sessionID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "세션을 찾을 수 없습니다"})
		return
	}

	// 권한 확인 (실제 구현에서는 더 정교한 권한 검사 필요)
	if !c.hasSessionAccess(userInfo.UserID, sessionID) {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "세션에 대한 접근 권한이 없습니다"})
		return
	}

	// 웹 세션 메타데이터 조회
	metadata, err := c.getWebSessionMetadata(sessionID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "세션 메타데이터 조회에 실패했습니다"})
		return
	}

	// 현재 참여자 목록 조회
	participants := c.streamHandler.GetSessionUsers(sessionID)

	// 연결 통계 조회
	connections := c.streamHandler.GetActiveConnections()
	activeConnections := 0
	if group, exists := connections[sessionID]; exists {
		activeConnections = len(group.Connections)
	}

	// 응답 생성
	response := &SessionResponse{
		SessionID:     session.ID,
		Name:          metadata["name"].(string),
		WorkspaceID:   session.Config.WorkspaceID,
		Status:        session.Status.String(),
		CreatedAt:     session.CreatedAt,
		UpdatedAt:     session.UpdatedAt,
		IsPrivate:     metadata["is_private"].(bool),
		Tags:          metadata["tags"].([]string),
		Participants:  participants,
		ConnectionURL: fmt.Sprintf("/ws/session/%s", session.ID),
		Statistics: SessionStatistics{
			MessageCount:        len(session.Messages),
			ParticipantCount:    len(participants),
			ActiveConnections:   activeConnections,
			LastActivity:        session.UpdatedAt,
			TotalDuration:       time.Since(session.CreatedAt),
			AverageResponseTime: calculateAverageResponseTime(session.Messages),
		},
	}

	ctx.JSON(http.StatusOK, response)
}

// ListSessions는 세션 목록을 조회합니다
func (c *WebSessionController) ListSessions(ctx *gin.Context) {
	var req SessionListRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 사용자 인증 확인
	userInfo, err := c.authValidator.ValidateGinContext(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "인증이 필요합니다"})
		return
	}

	// 기본값 설정
	if req.Limit == 0 {
		req.Limit = 20
	}
	if req.Limit > 100 {
		req.Limit = 100
	}

	// 세션 목록 조회
	sessions, err := c.sessionManager.ListSessions(ctx.Request.Context())
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "세션 목록 조회에 실패했습니다"})
		return
	}

	// 필터링 및 페이징 적용
	filteredSessions := c.filterSessions(sessions, req, userInfo.UserID)
	paginatedSessions := c.paginateSessions(filteredSessions, req.Offset, req.Limit)

	// 응답 변환
	var responses []SessionResponse
	for _, session := range paginatedSessions {
		metadata, _ := c.getWebSessionMetadata(session.ID)
		participants := c.streamHandler.GetSessionUsers(session.ID)
		
		var name string
		var isPrivate bool
		var tags []string
		
		if metadata != nil {
			if n, ok := metadata["name"].(string); ok {
				name = n
			}
			if p, ok := metadata["is_private"].(bool); ok {
				isPrivate = p
			}
			if t, ok := metadata["tags"].([]string); ok {
				tags = t
			}
		}

		responses = append(responses, SessionResponse{
			SessionID:     session.ID,
			Name:          name,
			WorkspaceID:   session.Config.WorkspaceID,
			Status:        session.Status.String(),
			CreatedAt:     session.CreatedAt,
			UpdatedAt:     session.UpdatedAt,
			IsPrivate:     isPrivate,
			Tags:          tags,
			Participants:  participants,
			ConnectionURL: fmt.Sprintf("/ws/session/%s", session.ID),
			Statistics: SessionStatistics{
				MessageCount:      len(session.Messages),
				ParticipantCount:  len(participants),
				LastActivity:      session.UpdatedAt,
				TotalDuration:     time.Since(session.CreatedAt),
			},
		})
	}

	ctx.JSON(http.StatusOK, gin.H{
		"sessions": responses,
		"total":    len(filteredSessions),
		"offset":   req.Offset,
		"limit":    req.Limit,
	})
}

// UpdateSession은 세션을 업데이트합니다
func (c *WebSessionController) UpdateSession(ctx *gin.Context) {
	sessionID := ctx.Param("id")
	if sessionID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "session_id가 필요합니다"})
		return
	}

	var req UpdateSessionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 사용자 인증 및 권한 확인
	userInfo, err := c.authValidator.ValidateGinContext(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "인증이 필요합니다"})
		return
	}

	if !c.hasSessionWriteAccess(userInfo.UserID, sessionID) {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "세션 수정 권한이 없습니다"})
		return
	}

	// 세션 업데이트 준비
	update := claude.SessionUpdate{}
	if req.SystemPrompt != nil {
		update.SystemPrompt = req.SystemPrompt
	}
	if req.MaxTurns != nil {
		update.MaxTurns = req.MaxTurns
	}
	if req.Options != nil {
		update.Options = req.Options
	}

	// 세션 업데이트
	if err := c.sessionManager.UpdateSession(sessionID, update); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "세션 업데이트에 실패했습니다"})
		return
	}

	// 웹 세션 메타데이터 업데이트
	metadata, _ := c.getWebSessionMetadata(sessionID)
	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	if req.Name != nil {
		metadata["name"] = *req.Name
	}
	if req.IsPrivate != nil {
		metadata["is_private"] = *req.IsPrivate
	}
	if req.Tags != nil {
		metadata["tags"] = req.Tags
	}
	metadata["updated_at"] = time.Now()

	if err := c.saveWebSessionMetadata(sessionID, metadata); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "세션 메타데이터 업데이트에 실패했습니다"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "세션이 업데이트되었습니다"})
}

// DeleteSession은 세션을 삭제합니다
func (c *WebSessionController) DeleteSession(ctx *gin.Context) {
	sessionID := ctx.Param("id")
	if sessionID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "session_id가 필요합니다"})
		return
	}

	// 사용자 인증 및 권한 확인
	userInfo, err := c.authValidator.ValidateGinContext(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "인증이 필요합니다"})
		return
	}

	if !c.hasSessionAdminAccess(userInfo.UserID, sessionID) {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "세션 삭제 권한이 없습니다"})
		return
	}

	// 활성 연결들 모두 종료
	connections := c.streamHandler.GetActiveConnections()
	if group, exists := connections[sessionID]; exists {
		for connID := range group.Connections {
			c.streamHandler.CloseConnection(sessionID, connID)
		}
	}

	// 세션 종료
	if err := c.sessionManager.CloseSession(sessionID); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "세션 종료에 실패했습니다"})
		return
	}

	// 웹 세션 메타데이터 삭제
	c.deleteWebSessionMetadata(sessionID)

	ctx.JSON(http.StatusOK, gin.H{"message": "세션이 삭제되었습니다"})
}

// SendMessage는 세션에 메시지를 전송합니다
func (c *WebSessionController) SendMessage(ctx *gin.Context) {
	sessionID := ctx.Param("id")
	if sessionID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "session_id가 필요합니다"})
		return
	}

	var req MessageSendRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 사용자 인증 및 권한 확인
	userInfo, err := c.authValidator.ValidateGinContext(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "인증이 필요합니다"})
		return
	}

	if !c.hasSessionWriteAccess(userInfo.UserID, sessionID) {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "메시지 전송 권한이 없습니다"})
		return
	}

	// Claude 세션에 메시지 전달
	if err := c.streamHandler.ForwardToSession(sessionID, userInfo.UserID, req.Message); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "메시지 전송에 실패했습니다"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "메시지가 전송되었습니다"})
}

// InviteUser는 사용자를 세션에 초대합니다
func (c *WebSessionController) InviteUser(ctx *gin.Context) {
	sessionID := ctx.Param("id")
	if sessionID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "session_id가 필요합니다"})
		return
	}

	var req InviteUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 사용자 인증 및 권한 확인
	userInfo, err := c.authValidator.ValidateGinContext(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "인증이 필요합니다"})
		return
	}

	if !c.hasSessionAdminAccess(userInfo.UserID, sessionID) {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "사용자 초대 권한이 없습니다"})
		return
	}

	// 초대 토큰 생성 (실제 구현에서는 더 정교한 초대 시스템 구축)
	inviteToken := generateInviteToken(sessionID, req.UserID, req.Permission)
	
	// 초대 정보 저장 (실제 구현 필요)
	inviteData := map[string]interface{}{
		"session_id":   sessionID,
		"inviter_id":   userInfo.UserID,
		"invitee_id":   req.UserID,
		"invitee_email": req.Email,
		"permission":   req.Permission,
		"token":        inviteToken,
		"created_at":   time.Now(),
		"expires_at":   req.ExpiresAt,
		"message":      req.Message,
		"status":       "pending",
	}

	if err := c.saveInviteData(inviteToken, inviteData); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "초대 정보 저장에 실패했습니다"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"invite_token": inviteToken,
		"invite_url":   fmt.Sprintf("/session/join?token=%s", inviteToken),
		"message":      "사용자가 초대되었습니다",
	})
}

// GetSessionStatistics는 세션 통계를 조회합니다
func (c *WebSessionController) GetSessionStatistics(ctx *gin.Context) {
	sessionID := ctx.Param("id")
	if sessionID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "session_id가 필요합니다"})
		return
	}

	// 사용자 인증 및 권한 확인
	userInfo, err := c.authValidator.ValidateGinContext(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "인증이 필요합니다"})
		return
	}

	if !c.hasSessionAccess(userInfo.UserID, sessionID) {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "세션 통계 조회 권한이 없습니다"})
		return
	}

	// 세션 조회
	session, err := c.sessionManager.GetSession(ctx.Request.Context(), sessionID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "세션을 찾을 수 없습니다"})
		return
	}

	// 상세 통계 생성
	participants := c.streamHandler.GetSessionUsers(sessionID)
	connections := c.streamHandler.GetActiveConnections()
	
	activeConnections := 0
	if group, exists := connections[sessionID]; exists {
		activeConnections = len(group.Connections)
	}

	// 메시지 분석
	messageStats := analyzeMessages(session.Messages)

	statistics := SessionStatistics{
		MessageCount:        len(session.Messages),
		ParticipantCount:    len(participants),
		ActiveConnections:   activeConnections,
		LastActivity:        session.UpdatedAt,
		TotalDuration:       time.Since(session.CreatedAt),
		AverageResponseTime: calculateAverageResponseTime(session.Messages),
	}

	ctx.JSON(http.StatusOK, gin.H{
		"session_id":  sessionID,
		"statistics":  statistics,
		"messages":    messageStats,
		"participants": participants,
	})
}

// 헬퍼 메서드들

func (c *WebSessionController) hasSessionAccess(userID, sessionID string) bool {
	// 실제 구현에서는 데이터베이스에서 권한 확인
	return true // 임시
}

func (c *WebSessionController) hasSessionWriteAccess(userID, sessionID string) bool {
	// 실제 구현에서는 데이터베이스에서 쓰기 권한 확인
	return true // 임시
}

func (c *WebSessionController) hasSessionAdminAccess(userID, sessionID string) bool {
	// 실제 구현에서는 데이터베이스에서 관리자 권한 확인
	return true // 임시
}

func (c *WebSessionController) saveWebSessionMetadata(sessionID string, data map[string]interface{}) error {
	// 실제 구현에서는 스토리지에 저장
	return nil // 임시
}

func (c *WebSessionController) getWebSessionMetadata(sessionID string) (map[string]interface{}, error) {
	// 실제 구현에서는 스토리지에서 조회
	return map[string]interface{}{
		"name":       "Sample Session",
		"is_private": false,
		"tags":       []string{"test"},
	}, nil // 임시
}

func (c *WebSessionController) deleteWebSessionMetadata(sessionID string) error {
	// 실제 구현에서는 스토리지에서 삭제
	return nil // 임시
}

func (c *WebSessionController) saveInviteData(token string, data map[string]interface{}) error {
	// 실제 구현에서는 스토리지에 저장
	return nil // 임시
}

func (c *WebSessionController) filterSessions(sessions []*claude.Session, req SessionListRequest, userID string) []*claude.Session {
	var filtered []*claude.Session
	
	for _, session := range sessions {
		// 워크스페이스 필터
		if req.WorkspaceID != "" && session.Config.WorkspaceID != req.WorkspaceID {
			continue
		}
		
		// 상태 필터
		if req.Status != "" && session.Status.String() != req.Status {
			continue
		}
		
		// 접근 권한 확인
		if !c.hasSessionAccess(userID, session.ID) {
			continue
		}
		
		filtered = append(filtered, session)
	}
	
	return filtered
}

func (c *WebSessionController) paginateSessions(sessions []*claude.Session, offset, limit int) []*claude.Session {
	if offset >= len(sessions) {
		return []*claude.Session{}
	}
	
	end := offset + limit
	if end > len(sessions) {
		end = len(sessions)
	}
	
	return sessions[offset:end]
}

func calculateAverageResponseTime(messages []claude.Message) time.Duration {
	if len(messages) < 2 {
		return 0
	}
	
	var totalTime time.Duration
	var responseCount int
	
	for i := 1; i < len(messages); i++ {
		if messages[i].Type == "assistant" && messages[i-1].Type == "user" {
			totalTime += messages[i].Timestamp.Sub(messages[i-1].Timestamp)
			responseCount++
		}
	}
	
	if responseCount == 0 {
		return 0
	}
	
	return totalTime / time.Duration(responseCount)
}

func analyzeMessages(messages []claude.Message) map[string]interface{} {
	stats := make(map[string]interface{})
	
	if len(messages) == 0 {
		return stats
	}
	
	userMessages := 0
	assistantMessages := 0
	totalLength := 0
	
	for _, msg := range messages {
		switch msg.Type {
		case "user":
			userMessages++
		case "assistant":
			assistantMessages++
		}
		totalLength += len(msg.Content)
	}
	
	stats["user_messages"] = userMessages
	stats["assistant_messages"] = assistantMessages
	stats["total_length"] = totalLength
	stats["average_length"] = totalLength / len(messages)
	
	return stats
}

func generateShareToken(sessionID string) string {
	// 실제 구현에서는 암호화된 토큰 생성
	return fmt.Sprintf("share_%s_%d", sessionID, time.Now().Unix())
}

func generateInviteToken(sessionID, userID string, permission Permission) string {
	// 실제 구현에서는 암호화된 토큰 생성
	return fmt.Sprintf("invite_%s_%s_%d_%d", sessionID, userID, int(permission), time.Now().Unix())
}