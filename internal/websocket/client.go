package websocket

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Client WebSocket 클라이언트 연결을 나타냄
type Client struct {
	// 연결 정보
	ID     string          `json:"id"`
	UserID string          `json:"user_id"`
	Conn   *websocket.Conn `json:"-"`
	
	// 채널 관리
	channels   map[string]bool `json:"-"`
	channelsMu sync.RWMutex    `json:"-"`
	
	// 메시지 전송
	send chan []byte
	
	// 상태 관리
	isAuthenticated bool
	lastPing        time.Time
	lastPong        time.Time
	
	// 제어
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
	
	// 허브 참조
	hub *Hub
	
	// 통계
	messagesReceived int64
	messagesSent     int64
	connectedAt      time.Time
}

// ClientConfig 클라이언트 설정
type ClientConfig struct {
	// 메시지 버퍼 크기
	SendBufferSize int
	
	// 타임아웃 설정
	WriteTimeout time.Duration
	ReadTimeout  time.Duration
	PingInterval time.Duration
	PongTimeout  time.Duration
	
	// 메시지 크기 제한
	MaxMessageSize int64
}

// DefaultClientConfig 기본 클라이언트 설정
func DefaultClientConfig() *ClientConfig {
	return &ClientConfig{
		SendBufferSize: 256,
		WriteTimeout:   10 * time.Second,
		ReadTimeout:    60 * time.Second,
		PingInterval:   30 * time.Second,
		PongTimeout:    10 * time.Second,
		MaxMessageSize: 1024 * 1024, // 1MB
	}
}

// NewClient 새 클라이언트 생성
func NewClient(id, userID string, conn *websocket.Conn, hub *Hub, config *ClientConfig) *Client {
	if config == nil {
		config = DefaultClientConfig()
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	client := &Client{
		ID:              id,
		UserID:          userID,
		Conn:            conn,
		channels:        make(map[string]bool),
		send:            make(chan []byte, config.SendBufferSize),
		isAuthenticated: false,
		lastPing:        time.Now(),
		lastPong:        time.Now(),
		ctx:             ctx,
		cancel:          cancel,
		done:            make(chan struct{}),
		hub:             hub,
		connectedAt:     time.Now(),
	}
	
	// WebSocket 설정
	conn.SetReadLimit(config.MaxMessageSize)
	conn.SetReadDeadline(time.Now().Add(config.ReadTimeout))
	conn.SetPongHandler(func(string) error {
		client.lastPong = time.Now()
		conn.SetReadDeadline(time.Now().Add(config.ReadTimeout))
		return nil
	})
	
	return client
}

// Start 클라이언트 시작 (읽기/쓰기 고루틴 실행)
func (c *Client) Start() {
	go c.readPump()
	go c.writePump()
	go c.pingPump()
}

// Stop 클라이언트 중지
func (c *Client) Stop() {
	if c.cancel != nil {
		c.cancel()
	}
}

// IsConnected 연결 상태 확인
func (c *Client) IsConnected() bool {
	select {
	case <-c.done:
		return false
	default:
		return true
	}
}

// IsAuthenticated 인증 상태 확인
func (c *Client) IsAuthenticated() bool {
	return c.isAuthenticated
}

// SetAuthenticated 인증 상태 설정
func (c *Client) SetAuthenticated(authenticated bool) {
	c.isAuthenticated = authenticated
}

// Subscribe 채널 구독
func (c *Client) Subscribe(channels ...string) {
	c.channelsMu.Lock()
	defer c.channelsMu.Unlock()
	
	for _, channel := range channels {
		c.channels[channel] = true
	}
}

// Unsubscribe 채널 구독 취소
func (c *Client) Unsubscribe(channels ...string) {
	c.channelsMu.Lock()
	defer c.channelsMu.Unlock()
	
	for _, channel := range channels {
		delete(c.channels, channel)
	}
}

// IsSubscribed 채널 구독 여부 확인
func (c *Client) IsSubscribed(channel string) bool {
	c.channelsMu.RLock()
	defer c.channelsMu.RUnlock()
	
	return c.channels[channel]
}

// GetChannels 구독 중인 채널 목록 반환
func (c *Client) GetChannels() []string {
	c.channelsMu.RLock()
	defer c.channelsMu.RUnlock()
	
	channels := make([]string, 0, len(c.channels))
	for channel := range c.channels {
		channels = append(channels, channel)
	}
	return channels
}

// Send 메시지 전송
func (c *Client) Send(message []byte) bool {
	if !c.IsConnected() {
		return false
	}
	
	select {
	case c.send <- message:
		return true
	default:
		// 버퍼가 가득 찬 경우 연결 해제
		log.Printf("클라이언트 %s 전송 버퍼 가득참, 연결 해제", c.ID)
		c.Stop()
		return false
	}
}

// SendMessage 메시지 객체 전송
func (c *Client) SendMessage(msg *Message) bool {
	data, err := msg.ToJSON()
	if err != nil {
		log.Printf("메시지 JSON 변환 실패: %v", err)
		return false
	}
	return c.Send(data)
}

// SendError 에러 메시지 전송
func (c *Client) SendError(code, message, details string) {
	errorMsg := NewErrorMessage(code, message, details)
	c.SendMessage(errorMsg)
}

// SendSuccess 성공 메시지 전송
func (c *Client) SendSuccess(message string, data interface{}) {
	successMsg := NewSuccessMessage(message, data)
	c.SendMessage(successMsg)
}

// readPump 메시지 읽기 펌프
func (c *Client) readPump() {
	defer func() {
		c.hub.Unregister(c)
		c.Conn.Close()
		close(c.done)
	}()
	
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			_, message, err := c.Conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket 읽기 에러: %v", err)
				}
				return
			}
			
			c.messagesReceived++
			
			// 메시지 처리
			if err := c.handleMessage(message); err != nil {
				log.Printf("메시지 처리 에러 (클라이언트 %s): %v", c.ID, err)
				c.SendError("MESSAGE_ERROR", "메시지 처리 실패", err.Error())
			}
		}
	}
}

// writePump 메시지 쓰기 펌프
func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()
	
	for {
		select {
		case message, ok := <-c.send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			
			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("WebSocket 쓰기 에러: %v", err)
				return
			}
			
			c.messagesSent++
			
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
			c.lastPing = time.Now()
			
		case <-c.ctx.Done():
			return
		}
	}
}

// pingPump 핑 펌프 (연결 상태 모니터링)
func (c *Client) pingPump() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			// 마지막 퐁으로부터 너무 오래 지났으면 연결 해제
			if time.Since(c.lastPong) > 60*time.Second {
				log.Printf("클라이언트 %s 응답 없음, 연결 해제", c.ID)
				c.Stop()
				return
			}
			
		case <-c.ctx.Done():
			return
		}
	}
}

// handleMessage 수신된 메시지 처리
func (c *Client) handleMessage(data []byte) error {
	msg, err := ParseMessage(data)
	if err != nil {
		return err
	}
	
	// 메시지 타입별 처리
	switch msg.Type {
	case MessageTypeAuth:
		return c.handleAuthMessage(msg)
	case MessageTypePing:
		return c.handlePingMessage(msg)
	case MessageTypeSubscribe:
		return c.handleSubscribeMessage(msg)
	case MessageTypeUnsubscribe:
		return c.handleUnsubscribeMessage(msg)
	case MessageTypeCommand:
		return c.handleCommandMessage(msg)
	default:
		// 비즈니스 메시지는 허브로 전달
		if c.isAuthenticated && msg.IsBusinessMessage() {
			c.hub.Broadcast(msg, c.GetChannels()...)
		} else if !c.isAuthenticated {
			c.SendError("NOT_AUTHENTICATED", "인증이 필요합니다", "")
		}
	}
	
	return nil
}

// handleAuthMessage 인증 메시지 처리
func (c *Client) handleAuthMessage(msg *Message) error {
	auth, err := msg.ParseAuthMessage()
	if err != nil {
		return err
	}
	
	// TODO: JWT 토큰 검증 로직 구현
	// 현재는 토큰이 있으면 인증된 것으로 처리
	if auth.Token != "" {
		c.SetAuthenticated(true)
		c.SendSuccess("인증 성공", map[string]interface{}{
			"client_id": c.ID,
			"user_id":   c.UserID,
		})
		
		// 사용자별 채널 자동 구독
		c.Subscribe(GetUserChannel(c.UserID))
	} else {
		c.SendError("INVALID_TOKEN", "유효하지 않은 토큰입니다", "")
	}
	
	return nil
}

// handlePingMessage 핑 메시지 처리
func (c *Client) handlePingMessage(msg *Message) error {
	pongMsg := NewMessage(MessageTypePong, map[string]interface{}{
		"timestamp": time.Now(),
	})
	c.SendMessage(pongMsg)
	return nil
}

// handleSubscribeMessage 구독 메시지 처리
func (c *Client) handleSubscribeMessage(msg *Message) error {
	sub, err := msg.ParseSubscribeMessage()
	if err != nil {
		return err
	}
	
	if !c.isAuthenticated {
		c.SendError("NOT_AUTHENTICATED", "인증이 필요합니다", "")
		return nil
	}
	
	// TODO: 채널별 권한 확인 로직 추가
	c.Subscribe(sub.Channels...)
	c.SendSuccess("채널 구독 완료", map[string]interface{}{
		"channels": sub.Channels,
	})
	
	return nil
}

// handleUnsubscribeMessage 구독 취소 메시지 처리
func (c *Client) handleUnsubscribeMessage(msg *Message) error {
	unsub, err := msg.ParseUnsubscribeMessage()
	if err != nil {
		return err
	}
	
	c.Unsubscribe(unsub.Channels...)
	c.SendSuccess("채널 구독 취소 완료", map[string]interface{}{
		"channels": unsub.Channels,
	})
	
	return nil
}

// handleCommandMessage 명령 메시지 처리
func (c *Client) handleCommandMessage(msg *Message) error {
	if !c.isAuthenticated {
		c.SendError("NOT_AUTHENTICATED", "인증이 필요합니다", "")
		return nil
	}
	
	cmd, err := msg.ParseCommandMessage()
	if err != nil {
		return err
	}
	
	// TODO: 명령 처리 로직 구현
	log.Printf("명령 수신 (클라이언트 %s): %s", c.ID, cmd.Command)
	
	// 현재는 단순히 성공 응답 전송
	c.SendSuccess("명령 수신됨", map[string]interface{}{
		"command":    cmd.Command,
		"session_id": cmd.SessionID,
	})
	
	return nil
}

// GetStats 클라이언트 통계 반환
func (c *Client) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"id":                 c.ID,
		"user_id":            c.UserID,
		"connected_at":       c.connectedAt,
		"is_authenticated":   c.isAuthenticated,
		"channels":           c.GetChannels(),
		"messages_received":  c.messagesReceived,
		"messages_sent":      c.messagesSent,
		"last_ping":          c.lastPing,
		"last_pong":          c.lastPong,
		"uptime":             time.Since(c.connectedAt).String(),
	}
}