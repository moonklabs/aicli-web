package docker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketMessage WebSocket 메시지 구조체
type WebSocketMessage struct {
	Type      string    `json:"type"`      // data, resize, ping, pong, error
	Data      string    `json:"data,omitempty"`      // Base64 인코딩된 데이터
	Width     int       `json:"width,omitempty"`     // 터미널 너비 (resize용)
	Height    int       `json:"height,omitempty"`    // 터미널 높이 (resize용)
	Timestamp time.Time `json:"timestamp"`
	Error     string    `json:"error,omitempty"`     // 에러 메시지
}

// WebSocketMessageType 메시지 타입 상수
const (
	MessageTypeData   = "data"
	MessageTypeResize = "resize"
	MessageTypePing   = "ping"
	MessageTypePong   = "pong"
	MessageTypeError  = "error"
	MessageTypeClose  = "close"
)

// PTYWebSocketStreamer WebSocket 스트리밍 인터페이스
type PTYWebSocketStreamer interface {
	HandleWebSocket(w http.ResponseWriter, r *http.Request) error
	AttachPTYSession(sessionID string) error
	DetachPTYSession() error
	SendMessage(data []byte) error
	SendTypedMessage(msgType string, data []byte) error
	Close() error
	IsConnected() bool
	GetConnectionID() string
	GetPTYSessionID() string
}

// WebSocketConnection 개별 WebSocket 연결
type WebSocketConnection struct {
	id            string
	conn          *websocket.Conn
	ptySession    PTYSession
	ptySessionID  string
	sendChan      chan WebSocketMessage
	receiveChan   chan WebSocketMessage
	closeChan     chan struct{}
	isConnected   bool
	lastPingTime  time.Time
	lastPongTime  time.Time
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	
	// 플로우 컨트롤
	sendBuffer    []WebSocketMessage
	bufferSize    int
	maxBufferSize int
	
	// 백프레셜 제어
	backpressureThreshold int
	isBackpressure        bool
}

// NewWebSocketConnection 새로운 WebSocket 연결 생성
func NewWebSocketConnection(connectionID string, conn *websocket.Conn) *WebSocketConnection {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &WebSocketConnection{
		id:                    connectionID,
		conn:                  conn,
		sendChan:              make(chan WebSocketMessage, 100),
		receiveChan:           make(chan WebSocketMessage, 100),
		closeChan:             make(chan struct{}),
		isConnected:           true,
		lastPingTime:          time.Now(),
		lastPongTime:          time.Now(),
		ctx:                   ctx,
		cancel:                cancel,
		sendBuffer:            make([]WebSocketMessage, 0),
		maxBufferSize:         1000,
		backpressureThreshold: 500,
		isBackpressure:        false,
	}
}

// GetConnectionID 연결 ID 반환
func (wsc *WebSocketConnection) GetConnectionID() string {
	return wsc.id
}

// GetPTYSessionID PTY 세션 ID 반환
func (wsc *WebSocketConnection) GetPTYSessionID() string {
	wsc.mu.RLock()
	defer wsc.mu.RUnlock()
	return wsc.ptySessionID
}

// AttachPTYSession PTY 세션 연결
func (wsc *WebSocketConnection) AttachPTYSession(sessionID string) error {
	wsc.mu.Lock()
	defer wsc.mu.Unlock()

	if wsc.ptySession != nil {
		return fmt.Errorf("PTY session already attached to connection %s", wsc.id)
	}

	// PTY 세션 매니저에서 세션 조회 (실제 구현에서는 의존성 주입 필요)
	// 여기서는 세션 ID만 저장
	wsc.ptySessionID = sessionID

	// PTY 데이터 스트리밍 시작
	go wsc.streamPTYToPTY()
	go wsc.streamWebSocketToPTY()

	return nil
}

// DetachPTYSession PTY 세션 연결 해제
func (wsc *WebSocketConnection) DetachPTYSession() error {
	wsc.mu.Lock()
	defer wsc.mu.Unlock()

	if wsc.ptySession != nil {
		wsc.ptySession = nil
	}
	wsc.ptySessionID = ""

	return nil
}

// SendMessage 데이터 메시지 전송
func (wsc *WebSocketConnection) SendMessage(data []byte) error {
	return wsc.SendTypedMessage(MessageTypeData, data)
}

// SendTypedMessage 타입이 지정된 메시지 전송
func (wsc *WebSocketConnection) SendTypedMessage(msgType string, data []byte) error {
	wsc.mu.RLock()
	defer wsc.mu.RUnlock()

	if !wsc.isConnected {
		return fmt.Errorf("WebSocket connection %s is not connected", wsc.id)
	}

	msg := WebSocketMessage{
		Type:      msgType,
		Timestamp: time.Now(),
	}

	if data != nil {
		msg.Data = base64.StdEncoding.EncodeToString(data)
	}

	// 백프레셜 체크
	if wsc.isBackpressure && msgType == MessageTypeData {
		// 버퍼에 추가
		if len(wsc.sendBuffer) < wsc.maxBufferSize {
			wsc.sendBuffer = append(wsc.sendBuffer, msg)
			return nil
		} else {
			// 버퍼 오버플로우 - 가장 오래된 메시지 제거
			wsc.sendBuffer = wsc.sendBuffer[1:]
			wsc.sendBuffer = append(wsc.sendBuffer, msg)
			return nil
		}
	}

	select {
	case wsc.sendChan <- msg:
		return nil
	case <-wsc.ctx.Done():
		return fmt.Errorf("connection context cancelled")
	default:
		// 채널이 가득 찬 경우 백프레셜 활성화
		wsc.isBackpressure = true
		wsc.sendBuffer = append(wsc.sendBuffer, msg)
		return nil
	}
}

// IsConnected 연결 상태 확인
func (wsc *WebSocketConnection) IsConnected() bool {
	wsc.mu.RLock()
	defer wsc.mu.RUnlock()
	return wsc.isConnected
}

// Close 연결 종료
func (wsc *WebSocketConnection) Close() error {
	wsc.mu.Lock()
	defer wsc.mu.Unlock()

	if !wsc.isConnected {
		return nil
	}

	wsc.isConnected = false
	wsc.cancel()

	// 종료 메시지 전송 (시도)
	closeMsg := WebSocketMessage{
		Type:      MessageTypeClose,
		Timestamp: time.Now(),
	}

	select {
	case wsc.sendChan <- closeMsg:
	default:
		// 채널이 가득 찬 경우 무시
	}

	close(wsc.closeChan)

	// WebSocket 연결 종료
	if wsc.conn != nil {
		wsc.conn.Close()
	}

	// PTY 세션 분리
	wsc.DetachPTYSession()

	return nil
}

// HandleWebSocket WebSocket 연결 처리
func (wsc *WebSocketConnection) HandleWebSocket(w http.ResponseWriter, r *http.Request) error {
	return fmt.Errorf("HandleWebSocket should be called from WebSocketManager")
}

// startMessagePump 메시지 펌프 시작
func (wsc *WebSocketConnection) startMessagePump() {
	go wsc.writePump()
	go wsc.readPump()
	go wsc.pingPump()
}

// writePump WebSocket 쓰기 펌프
func (wsc *WebSocketConnection) writePump() {
	defer wsc.conn.Close()

	ticker := time.NewTicker(54 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case msg, ok := <-wsc.sendChan:
			wsc.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				wsc.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := wsc.conn.WriteJSON(msg); err != nil {
				log.Printf("WebSocket write error: %v", err)
				return
			}

			// 백프레셜 완화 체크
			if wsc.isBackpressure && len(wsc.sendChan) < wsc.backpressureThreshold/2 {
				wsc.flushBuffer()
			}

		case <-ticker.C:
			wsc.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := wsc.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}

		case <-wsc.closeChan:
			return

		case <-wsc.ctx.Done():
			return
		}
	}
}

// readPump WebSocket 읽기 펌프
func (wsc *WebSocketConnection) readPump() {
	defer wsc.conn.Close()

	wsc.conn.SetReadLimit(512)
	wsc.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	wsc.conn.SetPongHandler(func(string) error {
		wsc.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		wsc.lastPongTime = time.Now()
		return nil
	})

	for {
		var msg WebSocketMessage
		err := wsc.conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// 메시지 처리
		wsc.handleIncomingMessage(msg)
	}
}

// pingPump 핑 펌프
func (wsc *WebSocketConnection) pingPump() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if !wsc.isConnected {
				return
			}

			pingMsg := WebSocketMessage{
				Type:      MessageTypePing,
				Timestamp: time.Now(),
			}

			select {
			case wsc.sendChan <- pingMsg:
				wsc.lastPingTime = time.Now()
			default:
				// 채널이 가득 찬 경우 무시
			}

		case <-wsc.ctx.Done():
			return
		}
	}
}

// handleIncomingMessage 수신 메시지 처리
func (wsc *WebSocketConnection) handleIncomingMessage(msg WebSocketMessage) {
	switch msg.Type {
	case MessageTypeData:
		// PTY에 데이터 전송
		if wsc.ptySession != nil && msg.Data != "" {
			data, err := base64.StdEncoding.DecodeString(msg.Data)
			if err != nil {
				log.Printf("Failed to decode message data: %v", err)
				return
			}
			wsc.ptySession.Write(data)
		}

	case MessageTypeResize:
		// PTY 터미널 크기 조정
		if wsc.ptySession != nil && msg.Width > 0 && msg.Height > 0 {
			wsc.ptySession.Resize(msg.Width, msg.Height)
		}

	case MessageTypePing:
		// Pong 응답
		pongMsg := WebSocketMessage{
			Type:      MessageTypePong,
			Timestamp: time.Now(),
		}
		select {
		case wsc.sendChan <- pongMsg:
		default:
		}

	case MessageTypePong:
		// Pong 처리
		wsc.lastPongTime = time.Now()

	default:
		log.Printf("Unknown message type: %s", msg.Type)
	}
}

// streamPTYToPTY PTY에서 WebSocket으로 데이터 스트리밍
func (wsc *WebSocketConnection) streamPTYToPTY() {
	if wsc.ptySession == nil {
		return
	}

	buffer := make([]byte, 4096)
	for {
		select {
		case <-wsc.ctx.Done():
			return
		default:
			n, err := wsc.ptySession.Read(buffer)
			if err != nil {
				if wsc.isConnected {
					errorMsg := WebSocketMessage{
						Type:      MessageTypeError,
						Error:     fmt.Sprintf("PTY read error: %v", err),
						Timestamp: time.Now(),
					}
					select {
					case wsc.sendChan <- errorMsg:
					default:
					}
				}
				return
			}

			if n > 0 {
				wsc.SendMessage(buffer[:n])
			}
		}
	}
}

// streamWebSocketToPTY WebSocket에서 PTY로 데이터 스트리밍
func (wsc *WebSocketConnection) streamWebSocketToPTY() {
	for {
		select {
		case msg := <-wsc.receiveChan:
			wsc.handleIncomingMessage(msg)
		case <-wsc.ctx.Done():
			return
		}
	}
}

// flushBuffer 버퍼 플러시
func (wsc *WebSocketConnection) flushBuffer() {
	if len(wsc.sendBuffer) == 0 {
		return
	}

	for _, msg := range wsc.sendBuffer {
		select {
		case wsc.sendChan <- msg:
		default:
			// 여전히 가득 찬 경우 중단
			return
		}
	}

	wsc.sendBuffer = wsc.sendBuffer[:0]
	wsc.isBackpressure = false
}

// GetStats 연결 통계 조회
func (wsc *WebSocketConnection) GetStats() *WebSocketConnectionStats {
	wsc.mu.RLock()
	defer wsc.mu.RUnlock()

	return &WebSocketConnectionStats{
		ConnectionID:      wsc.id,
		PTYSessionID:      wsc.ptySessionID,
		IsConnected:       wsc.isConnected,
		LastPingTime:      wsc.lastPingTime,
		LastPongTime:      wsc.lastPongTime,
		SendChannelSize:   len(wsc.sendChan),
		ReceiveChannelSize: len(wsc.receiveChan),
		BufferSize:        len(wsc.sendBuffer),
		IsBackpressure:    wsc.isBackpressure,
	}
}

// WebSocketConnectionStats 연결 통계
type WebSocketConnectionStats struct {
	ConnectionID       string    `json:"connection_id"`
	PTYSessionID       string    `json:"pty_session_id"`
	IsConnected        bool      `json:"is_connected"`
	LastPingTime       time.Time `json:"last_ping_time"`
	LastPongTime       time.Time `json:"last_pong_time"`
	SendChannelSize    int       `json:"send_channel_size"`
	ReceiveChannelSize int       `json:"receive_channel_size"`
	BufferSize         int       `json:"buffer_size"`
	IsBackpressure     bool      `json:"is_backpressure"`
}