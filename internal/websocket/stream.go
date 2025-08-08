package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

// StreamManager WebSocket 스트림 관리자
type StreamManager struct {
	connections map[string]*StreamConnection
	sessions    map[string]*StreamSession
	mutex       sync.RWMutex
	config      *StreamConfig
	
	// 메트릭
	totalConnections uint64
	totalMessages    uint64
	totalBytes       uint64
}

// StreamConnection WebSocket 연결
type StreamConnection struct {
	ID           string
	Conn         *websocket.Conn
	SessionID    string
	SendChan     chan []byte
	CloseChan    chan struct{}
	Context      context.Context
	Cancel       context.CancelFunc
	LastActivity time.Time
	
	// 버퍼링
	writeBuffer  []byte
	readBuffer   []byte
	
	// 통계
	messagesSent uint64
	bytesOut     uint64
	bytesIn      uint64
}

// StreamSession 스트리밍 세션
type StreamSession struct {
	ID            string
	PTYSessionID  string
	Connections   map[string]*StreamConnection
	InputChan     chan []byte
	OutputChan    chan []byte
	ErrorChan     chan error
	Context       context.Context
	Cancel        context.CancelFunc
	CreatedAt     time.Time
	LastActive    time.Time
	
	// 플로우 제어
	backpressure  *BackpressureController
	rateLimiter   *RateLimiter
	
	// 상태
	mutex         sync.RWMutex
	active        bool
	paused        bool
}

// StreamConfig 스트림 설정
type StreamConfig struct {
	MaxConnections      int
	MaxSessionsPerConn  int
	BufferSize          int
	WriteTimeout        time.Duration
	ReadTimeout         time.Duration
	PingInterval        time.Duration
	MaxMessageSize      int64
	EnableCompression   bool
	CompressionLevel    int
	BackpressureLimit   int
	RateLimitPerSecond  int
}

// DefaultStreamConfig 기본 스트림 설정
func DefaultStreamConfig() *StreamConfig {
	return &StreamConfig{
		MaxConnections:     1000,
		MaxSessionsPerConn: 10,
		BufferSize:         4096,
		WriteTimeout:       10 * time.Second,
		ReadTimeout:        60 * time.Second,
		PingInterval:       30 * time.Second,
		MaxMessageSize:     512 * 1024, // 512KB
		EnableCompression:  true,
		CompressionLevel:   1,
		BackpressureLimit:  100,
		RateLimitPerSecond: 1000,
	}
}

// NewStreamManager 새 스트림 관리자 생성
func NewStreamManager(config *StreamConfig) *StreamManager {
	if config == nil {
		config = DefaultStreamConfig()
	}

	return &StreamManager{
		connections: make(map[string]*StreamConnection),
		sessions:    make(map[string]*StreamSession),
		config:      config,
	}
}

// CreateConnection 새 WebSocket 연결 생성
func (sm *StreamManager) CreateConnection(ws *websocket.Conn) (*StreamConnection, error) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if len(sm.connections) >= sm.config.MaxConnections {
		return nil, fmt.Errorf("max connections reached: %d", sm.config.MaxConnections)
	}

	ctx, cancel := context.WithCancel(context.Background())
	
	conn := &StreamConnection{
		ID:           generateConnectionID(),
		Conn:         ws,
		SendChan:     make(chan []byte, sm.config.BufferSize),
		CloseChan:    make(chan struct{}),
		Context:      ctx,
		Cancel:       cancel,
		LastActivity: time.Now(),
		writeBuffer:  make([]byte, 0, sm.config.BufferSize),
		readBuffer:   make([]byte, sm.config.BufferSize),
	}

	// WebSocket 설정
	ws.SetReadLimit(sm.config.MaxMessageSize)
	ws.SetReadDeadline(time.Now().Add(sm.config.ReadTimeout))
	ws.SetPongHandler(func(string) error {
		ws.SetReadDeadline(time.Now().Add(sm.config.ReadTimeout))
		conn.LastActivity = time.Now()
		return nil
	})

	// 압축 설정
	if sm.config.EnableCompression {
		ws.EnableWriteCompression(true)
		ws.SetCompressionLevel(sm.config.CompressionLevel)
	}

	sm.connections[conn.ID] = conn
	sm.totalConnections++

	// 읽기/쓰기 고루틴 시작
	go sm.connectionWriter(conn)
	go sm.connectionReader(conn)
	go sm.connectionPinger(conn)

	log.Infof("WebSocket connection created: %s", conn.ID)
	return conn, nil
}

// CreateSession 새 스트리밍 세션 생성
func (sm *StreamManager) CreateSession(ptySessionID string) (*StreamSession, error) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	
	session := &StreamSession{
		ID:           generateSessionID(),
		PTYSessionID: ptySessionID,
		Connections:  make(map[string]*StreamConnection),
		InputChan:    make(chan []byte, sm.config.BufferSize),
		OutputChan:   make(chan []byte, sm.config.BufferSize),
		ErrorChan:    make(chan error, 10),
		Context:      ctx,
		Cancel:       cancel,
		CreatedAt:    time.Now(),
		LastActive:   time.Now(),
		backpressure: NewBackpressureController(sm.config.BackpressureLimit),
		rateLimiter:  NewRateLimiter(sm.config.RateLimitPerSecond),
		active:       true,
	}

	sm.sessions[session.ID] = session

	// 세션 처리 고루틴 시작
	go sm.sessionProcessor(session)

	log.Infof("Streaming session created: %s for PTY: %s", session.ID, ptySessionID)
	return session, nil
}

// AttachConnection 연결을 세션에 연결
func (sm *StreamManager) AttachConnection(connID, sessionID string) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	conn, exists := sm.connections[connID]
	if !exists {
		return fmt.Errorf("connection %s not found", connID)
	}

	session, exists := sm.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	if conn.SessionID != "" {
		return fmt.Errorf("connection already attached to session %s", conn.SessionID)
	}

	// 연결 추가
	conn.SessionID = sessionID
	session.mutex.Lock()
	session.Connections[connID] = conn
	session.mutex.Unlock()

	log.Infof("Connection %s attached to session %s", connID, sessionID)
	return nil
}

// connectionWriter 연결 쓰기 처리
func (sm *StreamManager) connectionWriter(conn *StreamConnection) {
	ticker := time.NewTicker(sm.config.PingInterval)
	defer func() {
		ticker.Stop()
		conn.Cancel()
		sm.removeConnection(conn.ID)
	}()

	for {
		select {
		case message, ok := <-conn.SendChan:
			if !ok {
				conn.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			conn.Conn.SetWriteDeadline(time.Now().Add(sm.config.WriteTimeout))
			
			if err := conn.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Errorf("Write error on connection %s: %v", conn.ID, err)
				return
			}

			conn.messagesSent++
			conn.bytesOut += uint64(len(message))
			sm.totalMessages++
			sm.totalBytes += uint64(len(message))

		case <-ticker.C:
			conn.Conn.SetWriteDeadline(time.Now().Add(sm.config.WriteTimeout))
			if err := conn.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Errorf("Ping error on connection %s: %v", conn.ID, err)
				return
			}

		case <-conn.Context.Done():
			return
		}
	}
}

// connectionReader 연결 읽기 처리
func (sm *StreamManager) connectionReader(conn *StreamConnection) {
	defer conn.Cancel()

	for {
		messageType, message, err := conn.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Errorf("Read error on connection %s: %v", conn.ID, err)
			}
			return
		}

		if messageType == websocket.TextMessage || messageType == websocket.BinaryMessage {
			conn.bytesIn += uint64(len(message))
			conn.LastActivity = time.Now()

			// 세션으로 메시지 전달
			if conn.SessionID != "" {
				sm.mutex.RLock()
				session, exists := sm.sessions[conn.SessionID]
				sm.mutex.RUnlock()

				if exists && session.active {
					select {
					case session.InputChan <- message:
					case <-time.After(100 * time.Millisecond):
						log.Warnf("Input channel full for session %s", session.ID)
					}
				}
			}
		}
	}
}

// connectionPinger 연결 핑 처리
func (sm *StreamManager) connectionPinger(conn *StreamConnection) {
	ticker := time.NewTicker(sm.config.PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if time.Since(conn.LastActivity) > 2*sm.config.PingInterval {
				log.Warnf("Connection %s inactive, closing", conn.ID)
				conn.Cancel()
				return
			}
		case <-conn.Context.Done():
			return
		}
	}
}

// sessionProcessor 세션 처리
func (sm *StreamManager) sessionProcessor(session *StreamSession) {
	defer session.Cancel()

	for {
		select {
		case data := <-session.OutputChan:
			// 백프레셔 확인
			if session.backpressure.ShouldThrottle() {
				time.Sleep(10 * time.Millisecond)
			}

			// 레이트 리미팅
			session.rateLimiter.Wait()

			// 모든 연결로 브로드캐스트
			session.mutex.RLock()
			connections := make([]*StreamConnection, 0, len(session.Connections))
			for _, conn := range session.Connections {
				connections = append(connections, conn)
			}
			session.mutex.RUnlock()

			for _, conn := range connections {
				select {
				case conn.SendChan <- data:
					session.backpressure.MessageSent()
				default:
					session.backpressure.MessageDropped()
					log.Warnf("Send channel full for connection %s", conn.ID)
				}
			}

			session.LastActive = time.Now()

		case err := <-session.ErrorChan:
			log.Errorf("Session %s error: %v", session.ID, err)
			
			// 에러 메시지를 연결로 전송
			errorMsg := map[string]interface{}{
				"type":  "error",
				"error": err.Error(),
			}
			
			if data, err := json.Marshal(errorMsg); err == nil {
				session.OutputChan <- data
			}

		case <-session.Context.Done():
			log.Infof("Session %s terminated", session.ID)
			return
		}
	}
}

// SendToSession 세션으로 데이터 전송
func (sm *StreamManager) SendToSession(sessionID string, data []byte) error {
	sm.mutex.RLock()
	session, exists := sm.sessions[sessionID]
	sm.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	if !session.active {
		return fmt.Errorf("session %s is not active", sessionID)
	}

	select {
	case session.OutputChan <- data:
		return nil
	case <-time.After(100 * time.Millisecond):
		return fmt.Errorf("timeout sending to session %s", sessionID)
	}
}

// BroadcastToAll 모든 연결로 브로드캐스트
func (sm *StreamManager) BroadcastToAll(data []byte) {
	sm.mutex.RLock()
	connections := make([]*StreamConnection, 0, len(sm.connections))
	for _, conn := range sm.connections {
		connections = append(connections, conn)
	}
	sm.mutex.RUnlock()

	for _, conn := range connections {
		select {
		case conn.SendChan <- data:
		default:
			log.Warnf("Failed to broadcast to connection %s", conn.ID)
		}
	}
}

// CloseConnection 연결 종료
func (sm *StreamManager) CloseConnection(connID string) error {
	sm.mutex.Lock()
	conn, exists := sm.connections[connID]
	sm.mutex.Unlock()

	if !exists {
		return fmt.Errorf("connection %s not found", connID)
	}

	conn.Cancel()
	conn.Conn.Close()

	return nil
}

// CloseSession 세션 종료
func (sm *StreamManager) CloseSession(sessionID string) error {
	sm.mutex.Lock()
	session, exists := sm.sessions[sessionID]
	sm.mutex.Unlock()

	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	session.active = false
	session.Cancel()

	// 모든 연결 종료
	session.mutex.RLock()
	for connID := range session.Connections {
		sm.CloseConnection(connID)
	}
	session.mutex.RUnlock()

	sm.mutex.Lock()
	delete(sm.sessions, sessionID)
	sm.mutex.Unlock()

	log.Infof("Session %s closed", sessionID)
	return nil
}

// removeConnection 연결 제거
func (sm *StreamManager) removeConnection(connID string) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	conn, exists := sm.connections[connID]
	if !exists {
		return
	}

	// 세션에서 제거
	if conn.SessionID != "" {
		if session, exists := sm.sessions[conn.SessionID]; exists {
			session.mutex.Lock()
			delete(session.Connections, connID)
			session.mutex.Unlock()
		}
	}

	delete(sm.connections, connID)
	log.Infof("Connection %s removed", connID)
}

// GetStats 통계 조회
func (sm *StreamManager) GetStats() map[string]interface{} {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	return map[string]interface{}{
		"active_connections": len(sm.connections),
		"active_sessions":    len(sm.sessions),
		"total_connections":  sm.totalConnections,
		"total_messages":     sm.totalMessages,
		"total_bytes":        sm.totalBytes,
	}
}

// generateConnectionID 연결 ID 생성
func generateConnectionID() string {
	return fmt.Sprintf("conn-%d-%d", time.Now().UnixNano(), randInt())
}

// generateSessionID 세션 ID 생성
func generateSessionID() string {
	return fmt.Sprintf("sess-%d-%d", time.Now().UnixNano(), randInt())
}

// randInt 랜덤 정수 생성
func randInt() int {
	return int(time.Now().UnixNano() % 100000)
}