package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/aicli/aicli-web/internal/claude"
	"github.com/aicli/aicli-web/internal/auth"
)

// ClaudeStreamHandler는 Claude 세션과 WebSocket 간의 실시간 스트림을 처리합니다
type ClaudeStreamHandler struct {
	sessionManager   claude.SessionManager
	connections      map[string]*ConnectionGroup
	connectionsMutex sync.RWMutex
	messageRouter    *MessageRouter
	authValidator    *auth.Validator
	upgrader         websocket.Upgrader
	
	// 설정
	config ClaudeStreamConfig
}

// ConnectionGroup은 하나의 세션에 연결된 WebSocket 연결들을 관리합니다
type ConnectionGroup struct {
	SessionID     string                    `json:"session_id"`
	Connections   map[string]*ClientConnection `json:"connections"`
	Permissions   map[string]Permission     `json:"permissions"`
	CreatedAt     time.Time                 `json:"created_at"`
	LastActivity  time.Time                 `json:"last_activity"`
	mutex         sync.RWMutex
}

// ClientConnection은 개별 클라이언트 연결을 나타냅니다
type ClientConnection struct {
	ID          string          `json:"id"`
	UserID      string          `json:"user_id"`
	UserName    string          `json:"user_name"`
	Conn        *websocket.Conn `json:"-"`
	Permission  Permission      `json:"permission"`
	ConnectedAt time.Time       `json:"connected_at"`
	LastSeen    time.Time       `json:"last_seen"`
	IsActive    bool            `json:"is_active"`
	
	// 채널들
	sendChan    chan []byte
	closeChan   chan struct{}
	ctx         context.Context
	cancel      context.CancelFunc
}

// ClaudeStreamConfig는 스트림 핸들러 설정입니다
type ClaudeStreamConfig struct {
	MaxConnections      int           `json:"max_connections"`
	MaxConnectionsPerUser int         `json:"max_connections_per_user"`
	MessageBufferSize   int           `json:"message_buffer_size"`
	PingInterval        time.Duration `json:"ping_interval"`
	PongTimeout         time.Duration `json:"pong_timeout"`
	ReadTimeout         time.Duration `json:"read_timeout"`
	WriteTimeout        time.Duration `json:"write_timeout"`
	MaxMessageSize      int64         `json:"max_message_size"`
	EnableCompression   bool          `json:"enable_compression"`
}

// Permission은 사용자 권한을 나타냅니다
type Permission int

const (
	PermissionNone Permission = iota
	PermissionRead
	PermissionWrite
	PermissionAdmin
)

// WebSocketMessage는 WebSocket 메시지 구조입니다
type WebSocketMessage struct {
	Type      string                 `json:"type"`
	SessionID string                 `json:"session_id,omitempty"`
	UserID    string                 `json:"user_id,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	MessageID string                 `json:"message_id,omitempty"`
}

// SessionEvent는 세션 이벤트를 나타냅니다
type SessionEvent struct {
	Type        string    `json:"type"`
	SessionID   string    `json:"session_id"`
	UserID      string    `json:"user_id"`
	UserName    string    `json:"user_name"`
	Timestamp   time.Time `json:"timestamp"`
	Data        interface{} `json:"data,omitempty"`
}

// DefaultClaudeStreamConfig는 기본 설정을 반환합니다
func DefaultClaudeStreamConfig() ClaudeStreamConfig {
	return ClaudeStreamConfig{
		MaxConnections:        1000,
		MaxConnectionsPerUser: 5,
		MessageBufferSize:     256,
		PingInterval:          30 * time.Second,
		PongTimeout:           10 * time.Second,
		ReadTimeout:           60 * time.Second,
		WriteTimeout:          10 * time.Second,
		MaxMessageSize:        32 * 1024, // 32KB
		EnableCompression:     true,
	}
}

// NewClaudeStreamHandler는 새로운 Claude 스트림 핸들러를 생성합니다
func NewClaudeStreamHandler(sessionManager claude.SessionManager, authValidator *auth.Validator, config ClaudeStreamConfig) *ClaudeStreamHandler {
	upgrader := websocket.Upgrader{
		ReadBufferSize:    4096,
		WriteBufferSize:   4096,
		EnableCompression: config.EnableCompression,
		CheckOrigin: func(r *http.Request) bool {
			// 실제 환경에서는 더 엄격한 검증 필요
			return true
		},
	}

	handler := &ClaudeStreamHandler{
		sessionManager: sessionManager,
		connections:    make(map[string]*ConnectionGroup),
		authValidator:  authValidator,
		upgrader:       upgrader,
		config:         config,
	}

	handler.messageRouter = NewMessageRouter(handler)
	
	return handler
}

// HandleWebSocket는 WebSocket 연결을 처리합니다
func (h *ClaudeStreamHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// 인증 확인
	token := r.Header.Get("Authorization")
	if token == "" {
		http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
		return
	}
	
	// Bearer 토큰 추출
	tokenStr, err := auth.ExtractTokenFromHeader(token)
	if err != nil {
		http.Error(w, "Invalid Authorization header", http.StatusUnauthorized)
		return
	}
	
	userInfo, err := (*h.authValidator).ValidateToken(r.Context(), tokenStr)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// WebSocket 업그레이드
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	// 클라이언트 연결 생성
	clientConn := h.createClientConnection(conn, userInfo)
	defer h.closeClientConnection(clientConn)

	// 연결 처리 시작
	h.handleConnection(clientConn)
}

// ConnectSession은 세션을 WebSocket 연결 그룹에 연결합니다
func (h *ClaudeStreamHandler) ConnectSession(sessionID string, clientConn *ClientConnection) error {
	h.connectionsMutex.Lock()
	defer h.connectionsMutex.Unlock()

	// 연결 그룹 찾기 또는 생성
	group, exists := h.connections[sessionID]
	if !exists {
		group = &ConnectionGroup{
			SessionID:    sessionID,
			Connections:  make(map[string]*ClientConnection),
			Permissions:  make(map[string]Permission),
			CreatedAt:    time.Now(),
			LastActivity: time.Now(),
		}
		h.connections[sessionID] = group
	}

	// 연결 추가
	group.mutex.Lock()
	group.Connections[clientConn.ID] = clientConn
	group.Permissions[clientConn.UserID] = clientConn.Permission
	group.LastActivity = time.Now()
	group.mutex.Unlock()

	// 세션 참여 이벤트 전송
	h.broadcastSessionEvent(sessionID, SessionEvent{
		Type:      "user_joined",
		SessionID: sessionID,
		UserID:    clientConn.UserID,
		UserName:  clientConn.UserName,
		Timestamp: time.Now(),
	})

	return nil
}

// StreamToWebSocket은 Claude 세션의 메시지를 WebSocket으로 전달합니다
func (h *ClaudeStreamHandler) StreamToWebSocket(sessionID string, messages <-chan claude.Message) error {
	go func() {
		for message := range messages {
			h.broadcastToSession(sessionID, WebSocketMessage{
				Type:      "claude_message",
				SessionID: sessionID,
				Data: map[string]interface{}{
					"message_type": message.Type,
					"content":      message.Content,
					"message_id":   message.ID,
					"meta":         message.Meta,
				},
				Timestamp: time.Now(),
			})
		}
	}()

	return nil
}

// ForwardToSession은 WebSocket 입력을 Claude 세션으로 전달합니다
func (h *ClaudeStreamHandler) ForwardToSession(sessionID string, userID string, input string) error {
	// 권한 확인
	if !h.hasWritePermission(sessionID, userID) {
		return fmt.Errorf("insufficient permissions")
	}

	// Claude 세션에 메시지 전송 (실제 구현 필요)
	// 여기서는 시뮬레이션
	h.broadcastToSession(sessionID, WebSocketMessage{
		Type:      "user_input",
		SessionID: sessionID,
		UserID:    userID,
		Data: map[string]interface{}{
			"input": input,
		},
		Timestamp: time.Now(),
	})

	return nil
}

// GetActiveConnections는 활성 연결 정보를 반환합니다
func (h *ClaudeStreamHandler) GetActiveConnections() map[string]*ConnectionGroup {
	h.connectionsMutex.RLock()
	defer h.connectionsMutex.RUnlock()

	result := make(map[string]*ConnectionGroup)
	for sessionID, group := range h.connections {
		// 복사본 생성
		groupCopy := &ConnectionGroup{
			SessionID:    group.SessionID,
			Connections:  make(map[string]*ClientConnection),
			Permissions:  make(map[string]Permission),
			CreatedAt:    group.CreatedAt,
			LastActivity: group.LastActivity,
		}

		group.mutex.RLock()
		for connID, conn := range group.Connections {
			if conn.IsActive {
				groupCopy.Connections[connID] = conn
			}
		}
		for userID, perm := range group.Permissions {
			groupCopy.Permissions[userID] = perm
		}
		group.mutex.RUnlock()

		if len(groupCopy.Connections) > 0 {
			result[sessionID] = groupCopy
		}
	}

	return result
}

// CloseConnection은 특정 연결을 종료합니다
func (h *ClaudeStreamHandler) CloseConnection(sessionID string, connectionID string) error {
	h.connectionsMutex.RLock()
	group, exists := h.connections[sessionID]
	h.connectionsMutex.RUnlock()

	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	group.mutex.Lock()
	defer group.mutex.Unlock()

	conn, exists := group.Connections[connectionID]
	if !exists {
		return fmt.Errorf("connection not found: %s", connectionID)
	}

	h.closeClientConnection(conn)
	delete(group.Connections, connectionID)

	return nil
}

// 내부 메서드들

func (h *ClaudeStreamHandler) createClientConnection(conn *websocket.Conn, userInfo *auth.UserInfo) *ClientConnection {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &ClientConnection{
		ID:          fmt.Sprintf("conn_%d", time.Now().UnixNano()),
		UserID:      userInfo.ID,
		UserName:    userInfo.Username,
		Conn:        conn,
		Permission:  h.determinePermission(userInfo),
		ConnectedAt: time.Now(),
		LastSeen:    time.Now(),
		IsActive:    true,
		sendChan:    make(chan []byte, h.config.MessageBufferSize),
		closeChan:   make(chan struct{}),
		ctx:         ctx,
		cancel:      cancel,
	}
}

func (h *ClaudeStreamHandler) handleConnection(clientConn *ClientConnection) {
	// 메시지 전송 고루틴 시작
	go h.handleSendMessages(clientConn)

	// Ping/Pong 처리 설정
	clientConn.Conn.SetReadLimit(h.config.MaxMessageSize)
	clientConn.Conn.SetReadDeadline(time.Now().Add(h.config.ReadTimeout))
	clientConn.Conn.SetPongHandler(func(string) error {
		clientConn.LastSeen = time.Now()
		clientConn.Conn.SetReadDeadline(time.Now().Add(h.config.ReadTimeout))
		return nil
	})

	// 메시지 읽기 루프
	for {
		select {
		case <-clientConn.ctx.Done():
			return
		default:
			_, message, err := clientConn.Conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket error: %v", err)
				}
				return
			}

			clientConn.LastSeen = time.Now()
			h.handleIncomingMessage(clientConn, message)
		}
	}
}

func (h *ClaudeStreamHandler) handleSendMessages(clientConn *ClientConnection) {
	ticker := time.NewTicker(h.config.PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-clientConn.ctx.Done():
			return
		case <-clientConn.closeChan:
			return
		case message := <-clientConn.sendChan:
			clientConn.Conn.SetWriteDeadline(time.Now().Add(h.config.WriteTimeout))
			if err := clientConn.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("Failed to send message: %v", err)
				return
			}
		case <-ticker.C:
			clientConn.Conn.SetWriteDeadline(time.Now().Add(h.config.WriteTimeout))
			if err := clientConn.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (h *ClaudeStreamHandler) handleIncomingMessage(clientConn *ClientConnection, message []byte) {
	var wsMessage WebSocketMessage
	if err := json.Unmarshal(message, &wsMessage); err != nil {
		log.Printf("Invalid message format: %v", err)
		return
	}

	wsMessage.UserID = clientConn.UserID
	wsMessage.Timestamp = time.Now()

	// 메시지 라우터로 전달
	h.messageRouter.RouteMessage(clientConn, &wsMessage)
}

func (h *ClaudeStreamHandler) broadcastToSession(sessionID string, message WebSocketMessage) {
	h.connectionsMutex.RLock()
	group, exists := h.connections[sessionID]
	h.connectionsMutex.RUnlock()

	if !exists {
		return
	}

	messageBytes, err := json.Marshal(message)
	if err != nil {
		log.Printf("Failed to marshal message: %v", err)
		return
	}

	group.mutex.RLock()
	defer group.mutex.RUnlock()

	for _, conn := range group.Connections {
		if conn.IsActive {
			select {
			case conn.sendChan <- messageBytes:
			default:
				// 버퍼가 가득 찬 경우 연결 종료
				h.closeClientConnection(conn)
			}
		}
	}
}

func (h *ClaudeStreamHandler) broadcastSessionEvent(sessionID string, event SessionEvent) {
	message := WebSocketMessage{
		Type:      "session_event",
		SessionID: sessionID,
		Data: map[string]interface{}{
			"event": event,
		},
		Timestamp: time.Now(),
	}

	h.broadcastToSession(sessionID, message)
}

func (h *ClaudeStreamHandler) closeClientConnection(clientConn *ClientConnection) {
	if !clientConn.IsActive {
		return
	}

	clientConn.IsActive = false
	clientConn.cancel()
	close(clientConn.closeChan)
	clientConn.Conn.Close()

	// 세션에서 연결 제거
	h.removeConnectionFromSessions(clientConn)
}

func (h *ClaudeStreamHandler) removeConnectionFromSessions(clientConn *ClientConnection) {
	h.connectionsMutex.Lock()
	defer h.connectionsMutex.Unlock()

	for sessionID, group := range h.connections {
		group.mutex.Lock()
		if _, exists := group.Connections[clientConn.ID]; exists {
			delete(group.Connections, clientConn.ID)
			
			// 세션에서 사용자 퇴장 이벤트 전송
			h.broadcastSessionEvent(sessionID, SessionEvent{
				Type:      "user_left",
				SessionID: sessionID,
				UserID:    clientConn.UserID,
				UserName:  clientConn.UserName,
				Timestamp: time.Now(),
			})

			// 연결이 없으면 그룹 제거
			if len(group.Connections) == 0 {
				delete(h.connections, sessionID)
			}
		}
		group.mutex.Unlock()
	}
}

func (h *ClaudeStreamHandler) determinePermission(userInfo *auth.UserInfo) Permission {
	// 실제 구현에서는 사용자 역할에 따라 권한 결정
	switch userInfo.Role {
	case "admin":
		return PermissionAdmin
	case "editor":
		return PermissionWrite
	case "viewer":
		return PermissionRead
	default:
		return PermissionRead
	}
}

func (h *ClaudeStreamHandler) hasWritePermission(sessionID string, userID string) bool {
	h.connectionsMutex.RLock()
	group, exists := h.connections[sessionID]
	h.connectionsMutex.RUnlock()

	if !exists {
		return false
	}

	group.mutex.RLock()
	defer group.mutex.RUnlock()

	permission, exists := group.Permissions[userID]
	if !exists {
		return false
	}

	return permission >= PermissionWrite
}

// GetSessionUsers는 세션의 참여 사용자 목록을 반환합니다
func (h *ClaudeStreamHandler) GetSessionUsers(sessionID string) []SessionUser {
	h.connectionsMutex.RLock()
	group, exists := h.connections[sessionID]
	h.connectionsMutex.RUnlock()

	if !exists {
		return []SessionUser{}
	}

	group.mutex.RLock()
	defer group.mutex.RUnlock()

	userMap := make(map[string]SessionUser)
	for _, conn := range group.Connections {
		if conn.IsActive {
			userMap[conn.UserID] = SessionUser{
				UserID:      conn.UserID,
				UserName:    conn.UserName,
				Permission:  conn.Permission,
				ConnectedAt: conn.ConnectedAt,
				LastSeen:    conn.LastSeen,
				IsActive:    conn.IsActive,
			}
		}
	}

	users := make([]SessionUser, 0, len(userMap))
	for _, user := range userMap {
		users = append(users, user)
	}

	return users
}

// SessionUser는 세션 참여 사용자 정보입니다
type SessionUser struct {
	UserID      string     `json:"user_id"`
	UserName    string     `json:"user_name"`
	Permission  Permission `json:"permission"`
	ConnectedAt time.Time  `json:"connected_at"`
	LastSeen    time.Time  `json:"last_seen"`
	IsActive    bool       `json:"is_active"`
}