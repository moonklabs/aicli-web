package docker

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// WebSocketManager WebSocket 연결 관리자
type WebSocketManager struct {
	connections     map[string]*WebSocketConnection
	ptyManager      PTYSessionManagement
	upgrader        websocket.Upgrader
	mu              sync.RWMutex
	ctx             context.Context
	cancel          context.CancelFunc
	cleanupInterval time.Duration
	cleanupTicker   *time.Ticker

	// 연결 제한
	maxConnections int
	
	// 모니터링
	totalConnections    int64
	activeConnections   int
	totalMessages       int64
	totalErrors         int64
}

// NewWebSocketManager 새로운 WebSocket 관리자 생성
func NewWebSocketManager(ptyManager PTYSessionManagement, maxConnections int) *WebSocketManager {
	ctx, cancel := context.WithCancel(context.Background())

	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			// 개발 환경에서는 모든 오리진 허용
			// 프로덕션에서는 적절한 오리진 검증 필요
			return true
		},
	}

	manager := &WebSocketManager{
		connections:     make(map[string]*WebSocketConnection),
		ptyManager:      ptyManager,
		upgrader:        upgrader,
		ctx:             ctx,
		cancel:          cancel,
		cleanupInterval: 30 * time.Second,
		maxConnections:  maxConnections,
	}

	// 정리 작업 시작
	manager.startCleanupRoutine()

	return manager
}

// HandleWebSocket WebSocket 연결 요청 처리
func (wsm *WebSocketManager) HandleWebSocket(w http.ResponseWriter, r *http.Request) error {
	// 연결 제한 확인
	wsm.mu.RLock()
	if len(wsm.connections) >= wsm.maxConnections {
		wsm.mu.RUnlock()
		http.Error(w, "Too many connections", http.StatusTooManyRequests)
		return fmt.Errorf("connection limit exceeded: %d", wsm.maxConnections)
	}
	wsm.mu.RUnlock()

	// WebSocket 업그레이드
	conn, err := wsm.upgrader.Upgrade(w, r, nil)
	if err != nil {
		wsm.totalErrors++
		return fmt.Errorf("websocket upgrade failed: %w", err)
	}

	// 연결 ID 생성
	connectionID := uuid.New().String()

	// WebSocket 연결 생성 및 등록
	wsConn := NewWebSocketConnection(connectionID, conn)
	
	wsm.mu.Lock()
	wsm.connections[connectionID] = wsConn
	wsm.totalConnections++
	wsm.activeConnections++
	wsm.mu.Unlock()

	// 메시지 펌프 시작
	wsConn.startMessagePump()

	// 연결 완료 로그
	fmt.Printf("WebSocket connected: %s (total: %d)\n", connectionID, wsm.activeConnections)

	// 연결 종료 대기
	go wsm.waitForDisconnection(connectionID)

	return nil
}

// AttachPTYToConnection PTY 세션을 WebSocket 연결에 연결
func (wsm *WebSocketManager) AttachPTYToConnection(connectionID, sessionID string) error {
	wsm.mu.RLock()
	conn, exists := wsm.connections[connectionID]
	wsm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("WebSocket connection %s not found", connectionID)
	}

	// PTY 세션 검증
	ptySession, err := wsm.ptyManager.GetSession(sessionID)
	if err != nil {
		return fmt.Errorf("PTY session %s not found: %w", sessionID, err)
	}

	// PTY 세션 연결
	conn.mu.Lock()
	conn.ptySession = ptySession
	conn.ptySessionID = sessionID
	conn.mu.Unlock()

	// PTY-WebSocket 브리지 시작
	go wsm.bridgePTYToWebSocket(conn)

	fmt.Printf("PTY session %s attached to WebSocket %s\n", sessionID, connectionID)
	return nil
}

// DetachPTYFromConnection PTY 세션을 WebSocket 연결에서 분리
func (wsm *WebSocketManager) DetachPTYFromConnection(connectionID string) error {
	wsm.mu.RLock()
	conn, exists := wsm.connections[connectionID]
	wsm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("WebSocket connection %s not found", connectionID)
	}

	return conn.DetachPTYSession()
}

// CloseConnection WebSocket 연결 종료
func (wsm *WebSocketManager) CloseConnection(connectionID string) error {
	wsm.mu.Lock()
	conn, exists := wsm.connections[connectionID]
	if exists {
		delete(wsm.connections, connectionID)
		wsm.activeConnections--
	}
	wsm.mu.Unlock()

	if !exists {
		return fmt.Errorf("WebSocket connection %s not found", connectionID)
	}

	return conn.Close()
}

// GetConnection 특정 연결 조회
func (wsm *WebSocketManager) GetConnection(connectionID string) (*WebSocketConnection, error) {
	wsm.mu.RLock()
	defer wsm.mu.RUnlock()

	conn, exists := wsm.connections[connectionID]
	if !exists {
		return nil, fmt.Errorf("WebSocket connection %s not found", connectionID)
	}

	return conn, nil
}

// ListConnections 모든 활성 연결 목록 조회
func (wsm *WebSocketManager) ListConnections() []*WebSocketConnection {
	wsm.mu.RLock()
	defer wsm.mu.RUnlock()

	connections := make([]*WebSocketConnection, 0, len(wsm.connections))
	for _, conn := range wsm.connections {
		connections = append(connections, conn)
	}

	return connections
}

// GetConnectionsByPTYSession PTY 세션으로 연결 목록 조회
func (wsm *WebSocketManager) GetConnectionsByPTYSession(sessionID string) []*WebSocketConnection {
	wsm.mu.RLock()
	defer wsm.mu.RUnlock()

	var connections []*WebSocketConnection
	for _, conn := range wsm.connections {
		if conn.GetPTYSessionID() == sessionID {
			connections = append(connections, conn)
		}
	}

	return connections
}

// BroadcastToPTYSessions PTY 세션 연결들에게 브로드캐스트
func (wsm *WebSocketManager) BroadcastToPTYSessions(sessionID string, msgType string, data []byte) error {
	connections := wsm.GetConnectionsByPTYSession(sessionID)
	
	if len(connections) == 0 {
		return fmt.Errorf("no WebSocket connections found for PTY session %s", sessionID)
	}

	var lastError error
	for _, conn := range connections {
		if err := conn.SendTypedMessage(msgType, data); err != nil {
			lastError = err
			fmt.Printf("Failed to send to connection %s: %v\n", conn.GetConnectionID(), err)
		}
	}

	return lastError
}

// bridgePTYToWebSocket PTY에서 WebSocket으로 데이터 브리지
func (wsm *WebSocketManager) bridgePTYToWebSocket(wsConn *WebSocketConnection) {
	if wsConn.ptySession == nil {
		return
	}

	buffer := make([]byte, 4096)
	for {
		if !wsConn.IsConnected() {
			break
		}

		// PTY에서 데이터 읽기
		n, err := wsConn.ptySession.Read(buffer)
		if err != nil {
			// PTY 읽기 오류
			errorMsg := fmt.Sprintf("PTY read error: %v", err)
			wsConn.SendTypedMessage(MessageTypeError, []byte(errorMsg))
			break
		}

		if n > 0 {
			// WebSocket으로 전송
			if err := wsConn.SendMessage(buffer[:n]); err != nil {
				fmt.Printf("Failed to send PTY data to WebSocket: %v\n", err)
				break
			}
			wsm.totalMessages++
		}
	}
}

// waitForDisconnection 연결 종료 대기
func (wsm *WebSocketManager) waitForDisconnection(connectionID string) {
	wsm.mu.RLock()
	conn, exists := wsm.connections[connectionID]
	wsm.mu.RUnlock()

	if !exists {
		return
	}

	// 연결 종료 대기
	select {
	case <-conn.closeChan:
	case <-conn.ctx.Done():
	case <-wsm.ctx.Done():
	}

	// 연결 정리
	wsm.mu.Lock()
	if _, exists := wsm.connections[connectionID]; exists {
		delete(wsm.connections, connectionID)
		wsm.activeConnections--
	}
	wsm.mu.Unlock()

	fmt.Printf("WebSocket disconnected: %s (remaining: %d)\n", connectionID, wsm.activeConnections)
}

// startCleanupRoutine 정리 루틴 시작
func (wsm *WebSocketManager) startCleanupRoutine() {
	wsm.cleanupTicker = time.NewTicker(wsm.cleanupInterval)

	go func() {
		defer wsm.cleanupTicker.Stop()

		for {
			select {
			case <-wsm.cleanupTicker.C:
				wsm.cleanupInactiveConnections()
			case <-wsm.ctx.Done():
				return
			}
		}
	}()
}

// cleanupInactiveConnections 비활성 연결 정리
func (wsm *WebSocketManager) cleanupInactiveConnections() {
	wsm.mu.Lock()
	defer wsm.mu.Unlock()

	var toRemove []string
	now := time.Now()

	for connectionID, conn := range wsm.connections {
		if !conn.IsConnected() {
			toRemove = append(toRemove, connectionID)
			continue
		}

		// 마지막 pong으로부터 너무 오래된 연결 확인
		if now.Sub(conn.lastPongTime) > 90*time.Second {
			toRemove = append(toRemove, connectionID)
			conn.Close()
		}
	}

	// 비활성 연결 제거
	for _, connectionID := range toRemove {
		delete(wsm.connections, connectionID)
		wsm.activeConnections--
	}

	if len(toRemove) > 0 {
		fmt.Printf("Cleaned up %d inactive WebSocket connections\n", len(toRemove))
	}
}

// GetStats 관리자 통계 조회
func (wsm *WebSocketManager) GetStats() *WebSocketManagerStats {
	wsm.mu.RLock()
	defer wsm.mu.RUnlock()

	connectionStats := make(map[string]*WebSocketConnectionStats)
	for id, conn := range wsm.connections {
		connectionStats[id] = conn.GetStats()
	}

	return &WebSocketManagerStats{
		TotalConnections:    wsm.totalConnections,
		ActiveConnections:   wsm.activeConnections,
		MaxConnections:      wsm.maxConnections,
		TotalMessages:       wsm.totalMessages,
		TotalErrors:         wsm.totalErrors,
		ConnectionStats:     connectionStats,
		CleanupInterval:     wsm.cleanupInterval,
	}
}

// Shutdown 관리자 종료
func (wsm *WebSocketManager) Shutdown() error {
	wsm.mu.Lock()
	defer wsm.mu.Unlock()

	// 컨텍스트 취소
	wsm.cancel()

	// 정리 타이머 중단
	if wsm.cleanupTicker != nil {
		wsm.cleanupTicker.Stop()
	}

	// 모든 연결 종료
	for connectionID, conn := range wsm.connections {
		conn.Close()
		delete(wsm.connections, connectionID)
	}

	wsm.activeConnections = 0

	fmt.Println("WebSocket manager shutdown completed")
	return nil
}

// WebSocketManagerStats WebSocket 관리자 통계
type WebSocketManagerStats struct {
	TotalConnections    int64                                    `json:"total_connections"`
	ActiveConnections   int                                      `json:"active_connections"`
	MaxConnections      int                                      `json:"max_connections"`
	TotalMessages       int64                                    `json:"total_messages"`
	TotalErrors         int64                                    `json:"total_errors"`
	ConnectionStats     map[string]*WebSocketConnectionStats    `json:"connection_stats"`
	CleanupInterval     time.Duration                            `json:"cleanup_interval"`
}

// WebSocketStreamingManager 통합 WebSocket 스트리밍 관리자
type WebSocketStreamingManager struct {
	wsManager       *WebSocketManager
	ptyManager      PTYSessionManagement
	containerPTY    ContainerPTYManagement
	mu              sync.RWMutex
}

// NewWebSocketStreamingManager 새로운 통합 스트리밍 관리자 생성
func NewWebSocketStreamingManager(ptyManager PTYSessionManagement, containerPTY ContainerPTYManagement, maxConnections int) *WebSocketStreamingManager {
	wsManager := NewWebSocketManager(ptyManager, maxConnections)

	return &WebSocketStreamingManager{
		wsManager:    wsManager,
		ptyManager:   ptyManager,
		containerPTY: containerPTY,
	}
}

// HandleWebSocketConnection WebSocket 연결 및 PTY 세션 연결 처리
func (wssm *WebSocketStreamingManager) HandleWebSocketConnection(w http.ResponseWriter, r *http.Request, sessionID string) error {
	// WebSocket 연결 처리
	if err := wssm.wsManager.HandleWebSocket(w, r); err != nil {
		return fmt.Errorf("failed to handle WebSocket: %w", err)
	}

	// 가장 최근 연결 찾기 (연결 ID는 UUID로 생성되므로 마지막 연결 찾기)
	connections := wssm.wsManager.ListConnections()
	if len(connections) == 0 {
		return fmt.Errorf("no WebSocket connections found")
	}

	latestConnection := connections[len(connections)-1]

	// PTY 세션 연결
	if sessionID != "" {
		if err := wssm.wsManager.AttachPTYToConnection(latestConnection.GetConnectionID(), sessionID); err != nil {
			return fmt.Errorf("failed to attach PTY session: %w", err)
		}
	}

	return nil
}

// GetManager WebSocket 관리자 반환
func (wssm *WebSocketStreamingManager) GetManager() *WebSocketManager {
	return wssm.wsManager
}

// Shutdown 통합 관리자 종료
func (wssm *WebSocketStreamingManager) Shutdown() error {
	return wssm.wsManager.Shutdown()
}