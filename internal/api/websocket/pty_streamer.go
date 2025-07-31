package websocket

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/aicli/aicli-web/internal/docker"
	"github.com/gorilla/websocket"
)

// PTYWebSocketStreamer PTY 세션을 위한 WebSocket 스트리밍 인터페이스
type PTYWebSocketStreamer interface {
	HandleWebSocket(w http.ResponseWriter, r *http.Request) error
	AttachPTYSession(sessionID string) error
	DetachPTYSession() error
	SendMessage(data []byte) error
	Close() error
	IsConnected() bool
	GetMetrics() *PTYStreamingMetrics
}

// PTYStreamingMessage WebSocket 메시지 타입 정의
type PTYStreamingMessage struct {
	Type      string    `json:"type"`                // "data", "resize", "ping", "pong", "error"
	Data      string    `json:"data,omitempty"`      // Base64 인코딩된 데이터
	Width     int       `json:"width,omitempty"`     // 터미널 너비
	Height    int       `json:"height,omitempty"`    // 터미널 높이
	Timestamp time.Time `json:"timestamp"`           // 메시지 타임스탬프
	Error     string    `json:"error,omitempty"`     // 에러 메시지
}

// PTYStreamingMetrics PTY 스트리밍 메트릭
type PTYStreamingMetrics struct {
	ConnectionDuration time.Duration `json:"connection_duration"`
	MessagesReceived   int64         `json:"messages_received"`
	MessagesSent      int64         `json:"messages_sent"`
	BytesReceived     int64         `json:"bytes_received"`
	BytesSent         int64         `json:"bytes_sent"`
	ErrorCount        int64         `json:"error_count"`
	LastActivity      time.Time     `json:"last_activity"`
	AverageLatency    time.Duration `json:"average_latency"`
	PTYSessionID      string        `json:"pty_session_id"`
	IsStreaming       bool          `json:"is_streaming"`
}

// ptyWebSocketStreamer PTY WebSocket 스트리밍 구현
type ptyWebSocketStreamer struct {
	// WebSocket 연결
	conn        *websocket.Conn
	isConnected bool
	connMu      sync.RWMutex

	// PTY 세션
	ptyManager docker.PTYSessionManagement
	ptySession docker.PTYSession
	sessionMu  sync.RWMutex

	// 데이터 채널
	sendChan    chan []byte
	receiveChan chan []byte
	closeChan   chan struct{}

	// 컨텍스트 및 제어
	ctx    context.Context
	cancel context.CancelFunc

	// 메트릭
	metrics *PTYStreamingMetrics
	startTime time.Time

	// 설정
	config PTYStreamerConfig
}

// PTYStreamerConfig PTY 스트리머 설정
type PTYStreamerConfig struct {
	ReadBufferSize  int           `json:"read_buffer_size"`  // 읽기 버퍼 크기
	WriteBufferSize int           `json:"write_buffer_size"` // 쓰기 버퍼 크기
	PingInterval    time.Duration `json:"ping_interval"`     // Ping 간격
	PongTimeout     time.Duration `json:"pong_timeout"`      // Pong 대기 시간
	MaxMessageSize  int64         `json:"max_message_size"`  // 최대 메시지 크기
	ChannelBuffer   int           `json:"channel_buffer"`    // 채널 버퍼 크기
}

// DefaultPTYStreamerConfig 기본 설정
func DefaultPTYStreamerConfig() PTYStreamerConfig {
	return PTYStreamerConfig{
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
		PingInterval:    30 * time.Second,
		PongTimeout:     10 * time.Second,
		MaxMessageSize:  32768, // 32KB
		ChannelBuffer:   100,
	}
}

// NewPTYWebSocketStreamer 새로운 PTY WebSocket 스트리머 생성
func NewPTYWebSocketStreamer(ptyManager docker.PTYSessionManagement, config PTYStreamerConfig) PTYWebSocketStreamer {
	ctx, cancel := context.WithCancel(context.Background())

	return &ptyWebSocketStreamer{
		ptyManager:  ptyManager,
		sendChan:    make(chan []byte, config.ChannelBuffer),
		receiveChan: make(chan []byte, config.ChannelBuffer),
		closeChan:   make(chan struct{}),
		ctx:         ctx,
		cancel:      cancel,
		config:      config,
		metrics: &PTYStreamingMetrics{
			LastActivity: time.Now(),
		},
		startTime: time.Now(),
	}
}

// WebSocket 업그레이더
var ptyUpgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		// TODO: 프로덕션에서는 적절한 Origin 검증 구현
		return true
	},
}

// HandleWebSocket WebSocket 연결 처리
func (s *ptyWebSocketStreamer) HandleWebSocket(w http.ResponseWriter, r *http.Request) error {
	// WebSocket 업그레이드
	conn, err := ptyUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return fmt.Errorf("websocket upgrade failed: %w", err)
	}

	s.connMu.Lock()
	s.conn = conn
	s.isConnected = true
	s.connMu.Unlock()

	// 연결 설정
	s.setupConnection()

	// 메시지 처리 고루틴 시작
	go s.handleIncomingMessages()
	go s.handleOutgoingMessages()
	go s.handlePTYData()
	go s.heartbeat()

	// 연결 완료 메시지 전송
	s.sendSystemMessage("connected", "PTY WebSocket connection established")

	// 연결 대기
	<-s.closeChan

	return nil
}

// setupConnection 연결 초기 설정
func (s *ptyWebSocketStreamer) setupConnection() {
	s.conn.SetReadLimit(s.config.MaxMessageSize)
	s.conn.SetReadDeadline(time.Now().Add(s.config.PongTimeout))
	
	// Pong 핸들러 설정
	s.conn.SetPongHandler(func(data string) error {
		s.metrics.LastActivity = time.Now()
		s.conn.SetReadDeadline(time.Now().Add(s.config.PongTimeout))
		return nil
	})

	// Close 핸들러 설정
	s.conn.SetCloseHandler(func(code int, text string) error {
		s.Close()
		return nil
	})
}

// AttachPTYSession PTY 세션 연결
func (s *ptyWebSocketStreamer) AttachPTYSession(sessionID string) error {
	s.sessionMu.Lock()
	defer s.sessionMu.Unlock()

	// 기존 세션 분리
	if s.ptySession != nil {
		s.ptySession = nil
	}

	// 새 세션 연결
	session, err := s.ptyManager.GetSession(sessionID)
	if err != nil {
		return fmt.Errorf("failed to get PTY session %s: %w", sessionID, err)
	}

	s.ptySession = session
	s.metrics.PTYSessionID = sessionID
	s.metrics.IsStreaming = true

	// 연결 성공 메시지 전송
	s.sendSystemMessage("attached", fmt.Sprintf("Attached to PTY session %s", sessionID))

	return nil
}

// DetachPTYSession PTY 세션 분리
func (s *ptyWebSocketStreamer) DetachPTYSession() error {
	s.sessionMu.Lock()
	defer s.sessionMu.Unlock()

	if s.ptySession != nil {
		sessionID := s.ptySession.ID()
		s.ptySession = nil
		s.metrics.PTYSessionID = ""
		s.metrics.IsStreaming = false

		// 분리 메시지 전송
		s.sendSystemMessage("detached", fmt.Sprintf("Detached from PTY session %s", sessionID))
	}

	return nil
}

// handleIncomingMessages 수신 메시지 처리
func (s *ptyWebSocketStreamer) handleIncomingMessages() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Panic in handleIncomingMessages: %v\n", r)
		}
	}()

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			_, messageData, err := s.conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					fmt.Printf("WebSocket read error: %v\n", err)
				}
				s.Close()
				return
			}

			s.metrics.MessagesReceived++
			s.metrics.BytesReceived += int64(len(messageData))
			s.metrics.LastActivity = time.Now()

			// 메시지 파싱
			var msg PTYStreamingMessage
			if err := json.Unmarshal(messageData, &msg); err != nil {
				s.sendErrorMessage(fmt.Sprintf("Invalid message format: %v", err))
				continue
			}

			// 메시지 타입별 처리
			switch msg.Type {
			case "data":
				s.handleDataMessage(msg)
			case "resize":
				s.handleResizeMessage(msg)
			case "ping":
				s.handlePingMessage(msg)
			default:
				s.sendErrorMessage(fmt.Sprintf("Unknown message type: %s", msg.Type))
			}
		}
	}
}

// handleDataMessage 데이터 메시지 처리
func (s *ptyWebSocketStreamer) handleDataMessage(msg PTYStreamingMessage) {
	s.sessionMu.RLock()
	session := s.ptySession
	s.sessionMu.RUnlock()

	if session == nil {
		s.sendErrorMessage("No PTY session attached")
		return
	}

	// Base64 디코딩
	data, err := base64.StdEncoding.DecodeString(msg.Data)
	if err != nil {
		s.sendErrorMessage(fmt.Sprintf("Failed to decode data: %v", err))
		return
	}

	// PTY 세션에 데이터 전송
	_, err = session.Write(data)
	if err != nil {
		s.sendErrorMessage(fmt.Sprintf("Failed to write to PTY session: %v", err))
		s.metrics.ErrorCount++
		return
	}
}

// handleResizeMessage 터미널 크기 변경 메시지 처리
func (s *ptyWebSocketStreamer) handleResizeMessage(msg PTYStreamingMessage) {
	s.sessionMu.RLock()
	session := s.ptySession
	s.sessionMu.RUnlock()

	if session == nil {
		s.sendErrorMessage("No PTY session attached")
		return
	}

	// PTY 세션 크기 조정
	err := session.Resize(msg.Width, msg.Height)
	if err != nil {
		s.sendErrorMessage(fmt.Sprintf("Failed to resize PTY session: %v", err))
		s.metrics.ErrorCount++
		return
	}
}

// handlePingMessage Ping 메시지 처리
func (s *ptyWebSocketStreamer) handlePingMessage(msg PTYStreamingMessage) {
	// Pong 응답 전송
	pongMsg := PTYStreamingMessage{
		Type:      "pong",
		Timestamp: time.Now(),
	}

	s.sendMessage(pongMsg)
}

// handleOutgoingMessages 송신 메시지 처리
func (s *ptyWebSocketStreamer) handleOutgoingMessages() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Panic in handleOutgoingMessages: %v\n", r)
		}
	}()

	for {
		select {
		case <-s.ctx.Done():
			return
		case data := <-s.sendChan:
			// Base64 인코딩
			encodedData := base64.StdEncoding.EncodeToString(data)

			msg := PTYStreamingMessage{
				Type:      "data",
				Data:      encodedData,
				Timestamp: time.Now(),
			}

			if err := s.sendMessage(msg); err != nil {
				fmt.Printf("Failed to send message: %v\n", err)
				s.metrics.ErrorCount++
			}
		}
	}
}

// handlePTYData PTY 세션 데이터 처리
func (s *ptyWebSocketStreamer) handlePTYData() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Panic in handlePTYData: %v\n", r)
		}
	}()

	buffer := make([]byte, s.config.ReadBufferSize)

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			s.sessionMu.RLock()
			session := s.ptySession
			s.sessionMu.RUnlock()

			if session == nil || !session.IsAlive() {
				time.Sleep(100 * time.Millisecond)
				continue
			}

			// PTY 세션에서 데이터 읽기
			n, err := session.Read(buffer)
			if err != nil {
				if err == io.EOF {
					// 세션 종료
					s.sendSystemMessage("session_closed", "PTY session closed")
					s.DetachPTYSession()
					continue
				}
				fmt.Printf("Failed to read from PTY session: %v\n", err)
				s.metrics.ErrorCount++
				continue
			}

			if n > 0 {
				// WebSocket으로 데이터 전송
				data := make([]byte, n)
				copy(data, buffer[:n])

				select {
				case s.sendChan <- data:
					// 성공적으로 전송
				case <-time.After(1 * time.Second):
					// 전송 타임아웃
					fmt.Printf("Send channel timeout, dropping data\n")
					s.metrics.ErrorCount++
				}
			}
		}
	}
}

// heartbeat 하트비트 처리
func (s *ptyWebSocketStreamer) heartbeat() {
	ticker := time.NewTicker(s.config.PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			if !s.IsConnected() {
				return
			}

			pingMsg := PTYStreamingMessage{
				Type:      "ping",
				Timestamp: time.Now(),
			}

			if err := s.sendMessage(pingMsg); err != nil {
				fmt.Printf("Failed to send ping: %v\n", err)
				s.Close()
				return
			}
		}
	}
}

// sendMessage 메시지 전송
func (s *ptyWebSocketStreamer) sendMessage(msg PTYStreamingMessage) error {
	s.connMu.RLock()
	conn := s.conn
	connected := s.isConnected
	s.connMu.RUnlock()

	if !connected || conn == nil {
		return fmt.Errorf("websocket not connected")
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	err = conn.WriteMessage(websocket.TextMessage, data)
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	s.metrics.MessagesSent++
	s.metrics.BytesSent += int64(len(data))
	s.metrics.LastActivity = time.Now()

	return nil
}

// sendSystemMessage 시스템 메시지 전송
func (s *ptyWebSocketStreamer) sendSystemMessage(msgType, message string) {
	msg := PTYStreamingMessage{
		Type:      msgType,
		Data:      base64.StdEncoding.EncodeToString([]byte(message)),
		Timestamp: time.Now(),
	}
	s.sendMessage(msg)
}

// sendErrorMessage 에러 메시지 전송
func (s *ptyWebSocketStreamer) sendErrorMessage(errorMsg string) {
	msg := PTYStreamingMessage{
		Type:      "error",
		Error:     errorMsg,
		Timestamp: time.Now(),
	}
	s.sendMessage(msg)
	s.metrics.ErrorCount++
}

// SendMessage 외부에서 메시지 전송
func (s *ptyWebSocketStreamer) SendMessage(data []byte) error {
	if !s.IsConnected() {
		return fmt.Errorf("websocket not connected")
	}

	select {
	case s.sendChan <- data:
		return nil
	case <-time.After(1 * time.Second):
		return fmt.Errorf("send timeout")
	}
}

// IsConnected 연결 상태 확인
func (s *ptyWebSocketStreamer) IsConnected() bool {
	s.connMu.RLock()
	defer s.connMu.RUnlock()
	return s.isConnected
}

// GetMetrics 메트릭 조회
func (s *ptyWebSocketStreamer) GetMetrics() *PTYStreamingMetrics {
	s.metrics.ConnectionDuration = time.Since(s.startTime)
	return s.metrics
}

// Close 연결 종료
func (s *ptyWebSocketStreamer) Close() error {
	s.connMu.Lock()
	defer s.connMu.Unlock()

	if !s.isConnected {
		return nil
	}

	s.isConnected = false

	// PTY 세션 분리
	s.DetachPTYSession()

	// 컨텍스트 취소
	s.cancel()

	// WebSocket 연결 종료
	if s.conn != nil {
		s.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		s.conn.Close()
	}

	// 채널 정리
	close(s.closeChan)

	return nil
}

// PTYWebSocketManager 여러 PTY WebSocket 연결을 관리하는 매니저
type PTYWebSocketManager struct {
	ptyManager  docker.PTYSessionManagement
	streamers   map[string]PTYWebSocketStreamer
	config      PTYStreamerConfig
	mu          sync.RWMutex
}

// NewPTYWebSocketManager 새로운 PTY WebSocket 매니저 생성
func NewPTYWebSocketManager(ptyManager docker.PTYSessionManagement) *PTYWebSocketManager {
	return &PTYWebSocketManager{
		ptyManager: ptyManager,
		streamers:  make(map[string]PTYWebSocketStreamer),
		config:     DefaultPTYStreamerConfig(),
	}
}

// CreateStreamer 새로운 스트리머 생성
func (m *PTYWebSocketManager) CreateStreamer(connectionID string) PTYWebSocketStreamer {
	m.mu.Lock()
	defer m.mu.Unlock()

	streamer := NewPTYWebSocketStreamer(m.ptyManager, m.config)
	m.streamers[connectionID] = streamer

	return streamer
}

// GetStreamer 스트리머 조회
func (m *PTYWebSocketManager) GetStreamer(connectionID string) (PTYWebSocketStreamer, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	streamer, exists := m.streamers[connectionID]
	return streamer, exists
}

// RemoveStreamer 스트리머 제거
func (m *PTYWebSocketManager) RemoveStreamer(connectionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if streamer, exists := m.streamers[connectionID]; exists {
		streamer.Close()
		delete(m.streamers, connectionID)
	}
}

// GetAllStreamers 모든 스트리머 조회
func (m *PTYWebSocketManager) GetAllStreamers() map[string]*PTYStreamingMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*PTYStreamingMetrics)
	for id, streamer := range m.streamers {
		result[id] = streamer.GetMetrics()
	}

	return result
}

// CleanupInactiveStreamers 비활성 스트리머 정리
func (m *PTYWebSocketManager) CleanupInactiveStreamers() {
	m.mu.Lock()
	defer m.mu.Unlock()

	inactiveThreshold := 5 * time.Minute
	now := time.Now()

	var toRemove []string
	for id, streamer := range m.streamers {
		metrics := streamer.GetMetrics()
		if now.Sub(metrics.LastActivity) > inactiveThreshold || !streamer.IsConnected() {
			toRemove = append(toRemove, id)
		}
	}

	for _, id := range toRemove {
		if streamer := m.streamers[id]; streamer != nil {
			streamer.Close()
			delete(m.streamers, id)
		}
	}

	if len(toRemove) > 0 {
		fmt.Printf("Cleaned up %d inactive PTY streamers\n", len(toRemove))
	}
}

// Shutdown 매니저 종료
func (m *PTYWebSocketManager) Shutdown() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, streamer := range m.streamers {
		streamer.Close()
		delete(m.streamers, id)
	}

	return nil
}