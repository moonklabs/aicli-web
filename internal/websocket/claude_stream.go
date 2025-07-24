package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/aicli/aicli-web/internal/claude"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// 개발 환경에서는 모든 오리진 허용
		// 프로덕션에서는 적절한 오리진 체크 필요
		return true
	},
}

// ClaudeStreamHandler는 Claude 스트림 WebSocket 연결을 관리합니다.
type ClaudeStreamHandler struct {
	hub      *Hub
	sessions map[string]*StreamSession
	mu       sync.RWMutex
	claude   claude.Wrapper
}

// NewClaudeStreamHandler는 새로운 Claude 스트림 핸들러를 생성합니다.
func NewClaudeStreamHandler(hub *Hub, claudeWrapper claude.Wrapper) *ClaudeStreamHandler {
	return &ClaudeStreamHandler{
		hub:      hub,
		sessions: make(map[string]*StreamSession),
		claude:   claudeWrapper,
	}
}

// StreamSession은 WebSocket 스트림 세션을 나타냅니다.
type StreamSession struct {
	ID           string
	Conn         *websocket.Conn
	Send         chan []byte
	claudeStream chan claude.Message
	ctx          context.Context
	cancel       context.CancelFunc
}

// HandleConnection은 WebSocket 연결을 처리합니다.
func (h *ClaudeStreamHandler) HandleConnection(c *gin.Context) {
	executionID := c.Param("executionID")
	if executionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "execution_id parameter is required",
		})
		return
	}

	// WebSocket 업그레이드
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	// 컨텍스트 생성
	ctx, cancel := context.WithCancel(c.Request.Context())

	// 스트림 세션 생성
	session := &StreamSession{
		ID:           executionID,
		Conn:         conn,
		Send:         make(chan []byte, 256),
		claudeStream: make(chan claude.Message, 100),
		ctx:          ctx,
		cancel:       cancel,
	}

	h.registerSession(session)

	// 동시 고루틴 실행
	go session.writePump()
	go session.readPump()
	go session.streamClaude()

	// 연결 성공 메시지 전송
	welcomeMsg := WebSocketMessage{
		Type:      "connection_established",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"execution_id": executionID,
			"status":       "ready",
		},
	}

	data, _ := json.Marshal(welcomeMsg)
	select {
	case session.Send <- data:
	default:
		log.Printf("Failed to send welcome message for execution %s", executionID)
	}
}

// registerSession은 세션을 등록합니다.
func (h *ClaudeStreamHandler) registerSession(session *StreamSession) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.sessions[session.ID] = session
}

// unregisterSession은 세션을 해제합니다.
func (h *ClaudeStreamHandler) unregisterSession(sessionID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if session, exists := h.sessions[sessionID]; exists {
		session.cancel()
		close(session.Send)
		delete(h.sessions, sessionID)
	}
}

// GetSession은 등록된 세션을 가져옵니다.
func (h *ClaudeStreamHandler) GetSession(sessionID string) *StreamSession {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.sessions[sessionID]
}

// BroadcastToSession은 특정 세션에 메시지를 전송합니다.
func (h *ClaudeStreamHandler) BroadcastToSession(sessionID string, message interface{}) error {
	session := h.GetSession(sessionID)
	if session == nil {
		return fmt.Errorf("session %s not found", sessionID)
	}

	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	select {
	case session.Send <- data:
		return nil
	case <-session.ctx.Done():
		return fmt.Errorf("session %s is closed", sessionID)
	default:
		return fmt.Errorf("session %s send channel is full", sessionID)
	}
}

// writePump은 메시지를 클라이언트로 전송합니다.
func (s *StreamSession) writePump() {
	ticker := time.NewTicker(54 * time.Second) // WebSocket ping 주기
	defer func() {
		ticker.Stop()
		s.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-s.Send:
			s.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// 채널이 닫힘
				s.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := s.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("WebSocket write error: %v", err)
				return
			}

		case <-ticker.C:
			s.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := s.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("WebSocket ping error: %v", err)
				return
			}

		case <-s.ctx.Done():
			return
		}
	}
}

// readPump은 클라이언트로부터 메시지를 읽습니다.
func (s *StreamSession) readPump() {
	defer func() {
		s.cancel()
		s.Conn.Close()
	}()

	s.Conn.SetReadLimit(512)
	s.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	s.Conn.SetPongHandler(func(string) error {
		s.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			_, message, err := s.Conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket error: %v", err)
				}
				return
			}

			// 클라이언트 메시지 처리
			s.handleClientMessage(message)
		}
	}
}

// handleClientMessage는 클라이언트에서 받은 메시지를 처리합니다.
func (s *StreamSession) handleClientMessage(message []byte) {
	var clientMsg ClientMessage
	if err := json.Unmarshal(message, &clientMsg); err != nil {
		log.Printf("Invalid client message: %v", err)
		return
	}

	// 클라이언트 메시지 타입별 처리
	switch clientMsg.Type {
	case "ping":
		// Pong 응답
		pongMsg := WebSocketMessage{
			Type:      "pong",
			Timestamp: time.Now(),
			Data:      map[string]interface{}{"status": "ok"},
		}
		data, _ := json.Marshal(pongMsg)
		select {
		case s.Send <- data:
		default:
		}

	case "subscribe_logs":
		// 로그 구독 요청 처리
		// TODO: 로그 구독 로직 구현

	case "unsubscribe_logs":
		// 로그 구독 해제 요청 처리
		// TODO: 로그 구독 해제 로직 구현

	default:
		log.Printf("Unknown client message type: %s", clientMsg.Type)
	}
}

// streamClaude는 Claude 메시지를 WebSocket으로 스트리밍합니다.
func (s *StreamSession) streamClaude() {
	for {
		select {
		case msg, ok := <-s.claudeStream:
			if !ok {
				// 채널이 닫힘
				return
			}

			// Claude 메시지를 WebSocket 메시지로 변환
			wsMsg := WebSocketMessage{
				Type:      "claude_message",
				Timestamp: time.Now(),
				Data: map[string]interface{}{
					"execution_id": s.ID,
					"message_type": msg.Type,
					"content":      msg.Content,
					"metadata":     nil, // msg.Metadata 필드 없음
				},
			}

			data, err := json.Marshal(wsMsg)
			if err != nil {
				log.Printf("Failed to marshal Claude message: %v", err)
				continue
			}

			select {
			case s.Send <- data:
			case <-time.After(time.Second):
				// 전송 타임아웃
				log.Printf("Message send timeout for session %s", s.ID)
				s.Close()
				return
			case <-s.ctx.Done():
				return
			}

		case <-s.ctx.Done():
			return
		}
	}
}

// Close는 스트림 세션을 종료합니다.
func (s *StreamSession) Close() {
	s.cancel()
}

// ClientMessage는 클라이언트에서 받는 메시지 구조체입니다.
type ClientMessage struct {
	Type string                 `json:"type"`
	Data map[string]interface{} `json:"data"`
}

// WebSocketMessage는 WebSocket으로 전송되는 메시지 구조체입니다.
type WebSocketMessage struct {
	Type      string      `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}