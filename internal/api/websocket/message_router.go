package websocket

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// MessageRouter는 WebSocket 메시지를 라우팅합니다
type MessageRouter struct {
	handler     *ClaudeStreamHandler
	routes      map[string][]RouteHandler
	middleware  []MiddlewareFunc
	errorHandler ErrorHandler
}

// RouteHandler는 메시지 처리 함수입니다
type RouteHandler func(ctx *MessageContext) error

// MiddlewareFunc는 미들웨어 함수입니다
type MiddlewareFunc func(next RouteHandler) RouteHandler

// ErrorHandler는 에러 처리 함수입니다
type ErrorHandler func(ctx *MessageContext, err error)

// MessageContext는 메시지 처리 컨텍스트입니다
type MessageContext struct {
	Client     *ClientConnection   `json:"client"`
	Message    *WebSocketMessage   `json:"message"`
	SessionID  string              `json:"session_id"`
	UserID     string              `json:"user_id"`
	Permission Permission          `json:"permission"`
	Data       map[string]interface{} `json:"data"`
	Response   *WebSocketMessage   `json:"response"`
	router     *MessageRouter
}

// MessageType 상수들
const (
	MessageTypeConnect        = "session.connect"
	MessageTypeDisconnect     = "session.disconnect"
	MessageTypeMessage        = "session.message"
	MessageTypeExecute        = "session.execute"
	MessageTypeStatus         = "session.status"
	MessageTypeUsers          = "session.users"
	MessageTypeShare          = "session.share"
	MessageTypeJoin           = "session.join"
	MessageTypeFileUpload     = "file.upload"
	MessageTypeFileDownload   = "file.download"
	MessageTypeFileList       = "file.list"
	MessageTypePing           = "ping"
	MessageTypeError          = "error"
)

// NewMessageRouter는 새로운 메시지 라우터를 생성합니다
func NewMessageRouter(handler *ClaudeStreamHandler) *MessageRouter {
	router := &MessageRouter{
		handler: handler,
		routes:  make(map[string][]RouteHandler),
		errorHandler: func(ctx *MessageContext, err error) {
			log.Printf("Message handling error: %v", err)
			ctx.SendError(fmt.Sprintf("처리 중 오류가 발생했습니다: %v", err))
		},
	}

	// 기본 라우트 등록
	router.setupDefaultRoutes()
	
	return router
}

// RouteMessage는 메시지를 라우팅합니다
func (r *MessageRouter) RouteMessage(client *ClientConnection, message *WebSocketMessage) {
	ctx := &MessageContext{
		Client:     client,
		Message:    message,
		SessionID:  message.SessionID,
		UserID:     message.UserID,
		Permission: client.Permission,
		Data:       message.Data,
		router:     r,
	}

	// 미들웨어 체인 구성
	handler := r.buildHandlerChain(message.Type)
	
	// 에러 처리
	if err := handler(ctx); err != nil {
		r.errorHandler(ctx, err)
	}
}

// AddRoute는 라우트를 추가합니다
func (r *MessageRouter) AddRoute(messageType string, handler RouteHandler) {
	if r.routes[messageType] == nil {
		r.routes[messageType] = make([]RouteHandler, 0)
	}
	r.routes[messageType] = append(r.routes[messageType], handler)
}

// AddMiddleware는 미들웨어를 추가합니다
func (r *MessageRouter) AddMiddleware(middleware MiddlewareFunc) {
	r.middleware = append(r.middleware, middleware)
}

// SetErrorHandler는 에러 핸들러를 설정합니다
func (r *MessageRouter) SetErrorHandler(handler ErrorHandler) {
	r.errorHandler = handler
}

// setupDefaultRoutes는 기본 라우트를 설정합니다
func (r *MessageRouter) setupDefaultRoutes() {
	// 기본 미들웨어 추가
	r.AddMiddleware(r.loggingMiddleware)
	r.AddMiddleware(r.authMiddleware)
	r.AddMiddleware(r.rateLimitMiddleware)

	// 세션 관련 라우트
	r.AddRoute(MessageTypeConnect, r.handleSessionConnect)
	r.AddRoute(MessageTypeDisconnect, r.handleSessionDisconnect)
	r.AddRoute(MessageTypeMessage, r.handleSessionMessage)
	r.AddRoute(MessageTypeExecute, r.handleSessionExecute)
	r.AddRoute(MessageTypeStatus, r.handleSessionStatus)
	r.AddRoute(MessageTypeUsers, r.handleSessionUsers)
	r.AddRoute(MessageTypeShare, r.handleSessionShare)
	r.AddRoute(MessageTypeJoin, r.handleSessionJoin)

	// 파일 관련 라우트
	r.AddRoute(MessageTypeFileUpload, r.handleFileUpload)
	r.AddRoute(MessageTypeFileDownload, r.handleFileDownload)
	r.AddRoute(MessageTypeFileList, r.handleFileList)

	// 기타 라우트
	r.AddRoute(MessageTypePing, r.handlePing)
}

// 미들웨어들

func (r *MessageRouter) loggingMiddleware(next RouteHandler) RouteHandler {
	return func(ctx *MessageContext) error {
		start := time.Now()
		err := next(ctx)
		duration := time.Since(start)
		
		log.Printf("WebSocket message handled: type=%s, user=%s, session=%s, duration=%v, error=%v",
			ctx.Message.Type, ctx.UserID, ctx.SessionID, duration, err)
		
		return err
	}
}

func (r *MessageRouter) authMiddleware(next RouteHandler) RouteHandler {
	return func(ctx *MessageContext) error {
		// 일부 메시지는 인증 체크 생략
		if ctx.Message.Type == MessageTypePing {
			return next(ctx)
		}

		// 사용자 권한 확인
		if ctx.Permission == PermissionNone {
			return fmt.Errorf("insufficient permissions")
		}

		return next(ctx)
	}
}

func (r *MessageRouter) rateLimitMiddleware(next RouteHandler) RouteHandler {
	return func(ctx *MessageContext) error {
		// 간단한 레이트 제한 (실제 구현에서는 더 정교한 로직 필요)
		// 여기서는 시뮬레이션
		return next(ctx)
	}
}

// 라우트 핸들러들

func (r *MessageRouter) handleSessionConnect(ctx *MessageContext) error {
	sessionID, ok := ctx.Data["session_id"].(string)
	if !ok {
		return fmt.Errorf("session_id is required")
	}

	// 세션 연결
	err := r.handler.ConnectSession(sessionID, ctx.Client)
	if err != nil {
		return fmt.Errorf("failed to connect to session: %w", err)
	}

	// 성공 응답
	ctx.SendSuccess(map[string]interface{}{
		"session_id": sessionID,
		"connected":  true,
		"users":      r.handler.GetSessionUsers(sessionID),
	})

	return nil
}

func (r *MessageRouter) handleSessionDisconnect(ctx *MessageContext) error {
	sessionID := ctx.SessionID
	if sessionID == "" {
		return fmt.Errorf("session_id is required")
	}

	// 세션 연결 해제
	err := r.handler.CloseConnection(sessionID, ctx.Client.ID)
	if err != nil {
		return fmt.Errorf("failed to disconnect from session: %w", err)
	}

	ctx.SendSuccess(map[string]interface{}{
		"session_id":   sessionID,
		"disconnected": true,
	})

	return nil
}

func (r *MessageRouter) handleSessionMessage(ctx *MessageContext) error {
	if ctx.Permission < PermissionWrite {
		return fmt.Errorf("write permission required")
	}

	message, ok := ctx.Data["message"].(string)
	if !ok {
		return fmt.Errorf("message is required")
	}

	// Claude 세션에 메시지 전달
	err := r.handler.ForwardToSession(ctx.SessionID, ctx.UserID, message)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

func (r *MessageRouter) handleSessionExecute(ctx *MessageContext) error {
	if ctx.Permission < PermissionWrite {
		return fmt.Errorf("write permission required")
	}

	command, ok := ctx.Data["command"].(string)
	if !ok {
		return fmt.Errorf("command is required")
	}

	// 명령어 실행 (실제 구현 필요)
	result := fmt.Sprintf("Executed command: %s", command)
	
	ctx.SendSuccess(map[string]interface{}{
		"command": command,
		"result":  result,
	})

	return nil
}

func (r *MessageRouter) handleSessionStatus(ctx *MessageContext) error {
	sessionID := ctx.SessionID
	if sessionID == "" {
		return fmt.Errorf("session_id is required")
	}

	// 세션 상태 조회 (실제 구현 필요)
	status := map[string]interface{}{
		"session_id": sessionID,
		"status":     "active",
		"users":      r.handler.GetSessionUsers(sessionID),
		"timestamp":  time.Now(),
	}

	ctx.SendSuccess(status)
	return nil
}

func (r *MessageRouter) handleSessionUsers(ctx *MessageContext) error {
	sessionID := ctx.SessionID
	if sessionID == "" {
		return fmt.Errorf("session_id is required")
	}

	users := r.handler.GetSessionUsers(sessionID)
	
	ctx.SendSuccess(map[string]interface{}{
		"session_id": sessionID,
		"users":      users,
	})

	return nil
}

func (r *MessageRouter) handleSessionShare(ctx *MessageContext) error {
	if ctx.Permission < PermissionAdmin {
		return fmt.Errorf("admin permission required")
	}

	sessionID := ctx.SessionID
	if sessionID == "" {
		return fmt.Errorf("session_id is required")
	}

	// 공유 링크 생성 (실제 구현 필요)
	shareToken := fmt.Sprintf("share_%s_%d", sessionID, time.Now().Unix())
	shareURL := fmt.Sprintf("/session/join?token=%s", shareToken)

	ctx.SendSuccess(map[string]interface{}{
		"session_id":  sessionID,
		"share_token": shareToken,
		"share_url":   shareURL,
		"expires_at":  time.Now().Add(24 * time.Hour),
	})

	return nil
}

func (r *MessageRouter) handleSessionJoin(ctx *MessageContext) error {
	shareToken, ok := ctx.Data["share_token"].(string)
	if !ok {
		return fmt.Errorf("share_token is required")
	}

	// 토큰 검증 및 세션 ID 추출 (실제 구현 필요)
	// 여기서는 시뮬레이션
	sessionID := ctx.SessionID
	if sessionID == "" || shareToken == "" {
		return fmt.Errorf("invalid share token: %s", shareToken)
	}

	// 세션 참여
	err := r.handler.ConnectSession(sessionID, ctx.Client)
	if err != nil {
		return fmt.Errorf("failed to join session: %w", err)
	}

	ctx.SendSuccess(map[string]interface{}{
		"session_id": sessionID,
		"joined":     true,
		"users":      r.handler.GetSessionUsers(sessionID),
	})

	return nil
}

func (r *MessageRouter) handleFileUpload(ctx *MessageContext) error {
	if ctx.Permission < PermissionWrite {
		return fmt.Errorf("write permission required")
	}

	fileName, ok := ctx.Data["filename"].(string)
	if !ok {
		return fmt.Errorf("filename is required")
	}

	fileData, ok := ctx.Data["data"].(string)
	if !ok {
		return fmt.Errorf("file data is required")
	}

	// 파일 업로드 처리 (실제 구현 필요)
	fileID := fmt.Sprintf("file_%d", time.Now().UnixNano())
	
	ctx.SendSuccess(map[string]interface{}{
		"file_id":   fileID,
		"filename":  fileName,
		"size":      len(fileData),
		"uploaded":  true,
		"timestamp": time.Now(),
	})

	return nil
}

func (r *MessageRouter) handleFileDownload(ctx *MessageContext) error {
	fileID, ok := ctx.Data["file_id"].(string)
	if !ok {
		return fmt.Errorf("file_id is required")
	}

	// 파일 다운로드 처리 (실제 구현 필요)
	fileData := "dummy file content"
	fileName := "downloaded_file.txt"

	ctx.SendSuccess(map[string]interface{}{
		"file_id":  fileID,
		"filename": fileName,
		"data":     fileData,
		"size":     len(fileData),
	})

	return nil
}

func (r *MessageRouter) handleFileList(ctx *MessageContext) error {
	sessionID := ctx.SessionID
	if sessionID == "" {
		return fmt.Errorf("session_id is required")
	}

	// 파일 목록 조회 (실제 구현 필요)
	files := []map[string]interface{}{
		{
			"file_id":    "file_1",
			"filename":   "example.txt",
			"size":       1024,
			"uploaded_by": "user1",
			"uploaded_at": time.Now().Add(-time.Hour),
		},
		{
			"file_id":    "file_2",
			"filename":   "data.json",
			"size":       2048,
			"uploaded_by": "user2",
			"uploaded_at": time.Now().Add(-30 * time.Minute),
		},
	}

	ctx.SendSuccess(map[string]interface{}{
		"session_id": sessionID,
		"files":      files,
	})

	return nil
}

func (r *MessageRouter) handlePing(ctx *MessageContext) error {
	ctx.Send(&WebSocketMessage{
		Type: "pong",
		Data: map[string]interface{}{
			"timestamp": time.Now(),
		},
		Timestamp: time.Now(),
	})
	return nil
}

// 빌더 메서드들

func (r *MessageRouter) buildHandlerChain(messageType string) RouteHandler {
	handlers := r.routes[messageType]
	if len(handlers) == 0 {
		return func(ctx *MessageContext) error {
			return fmt.Errorf("unknown message type: %s", messageType)
		}
	}

	// 핸들러 체인 구성
	var chain RouteHandler = handlers[0]
	for i := len(handlers) - 1; i > 0; i-- {
		nextHandler := handlers[i]
		currentHandler := chain
		chain = func(ctx *MessageContext) error {
			if err := currentHandler(ctx); err != nil {
				return err
			}
			return nextHandler(ctx)
		}
	}

	// 미들웨어 체인 구성
	for i := len(r.middleware) - 1; i >= 0; i-- {
		chain = r.middleware[i](chain)
	}

	return chain
}

// MessageContext 메서드들

// Send는 메시지를 클라이언트에게 전송합니다
func (ctx *MessageContext) Send(message *WebSocketMessage) error {
	messageBytes, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	select {
	case ctx.Client.sendChan <- messageBytes:
		return nil
	default:
		return fmt.Errorf("client send buffer is full")
	}
}

// SendSuccess는 성공 응답을 전송합니다
func (ctx *MessageContext) SendSuccess(data interface{}) error {
	return ctx.Send(&WebSocketMessage{
		Type: "response",
		Data: map[string]interface{}{
			"status": "success",
			"data":   data,
		},
		Timestamp: time.Now(),
	})
}

// SendError는 에러 응답을 전송합니다
func (ctx *MessageContext) SendError(message string) error {
	return ctx.Send(&WebSocketMessage{
		Type: "error",
		Data: map[string]interface{}{
			"status":  "error",
			"message": message,
		},
		Timestamp: time.Now(),
	})
}

// Broadcast는 세션의 모든 사용자에게 메시지를 브로드캐스트합니다
func (ctx *MessageContext) Broadcast(message *WebSocketMessage) {
	if ctx.SessionID != "" {
		ctx.router.handler.broadcastToSession(ctx.SessionID, *message)
	}
}