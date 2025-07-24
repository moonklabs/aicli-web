package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aicli/aicli-web/internal/auth"
	"github.com/gorilla/websocket"
)

// ConnectionManager는 WebSocket 연결 생명주기를 관리합니다
type ConnectionManager struct {
	// 연결 관리
	connections      map[string]*ManagedConnection
	connectionsMutex sync.RWMutex
	
	// 사용자별 연결 추적
	userConnections  map[string]map[string]*ManagedConnection
	userMutex        sync.RWMutex
	
	// 세션별 연결 추적
	sessionConnections map[string]map[string]*ManagedConnection
	sessionMutex       sync.RWMutex
	
	// 통계
	stats            ConnectionStats
	statsUpdatedAt   time.Time
	
	// 설정
	config ConnectionManagerConfig
	
	// 이벤트 채널
	eventChan chan ConnectionEvent
	
	// 생명주기
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// ManagedConnection은 관리되는 연결입니다
type ManagedConnection struct {
	*ClientConnection
	
	// 연결 관리
	manager     *ConnectionManager
	healthCheck *ConnectionHealthCheck
	
	// 메트릭
	metrics ConnectionMetrics
	
	// 상태
	state       ConnectionState
	stateMutex  sync.RWMutex
	
	// 타이머
	heartbeatTicker *time.Ticker
	timeoutTimer    *time.Timer
	
	// 구독
	subscriptions map[string]bool
	subMutex      sync.RWMutex
}

// ConnectionHealthCheck는 연결 상태 검사입니다
type ConnectionHealthCheck struct {
	LastPing       time.Time     `json:"last_ping"`
	LastPong       time.Time     `json:"last_pong"`
	PingCount      int64         `json:"ping_count"`
	PongCount      int64         `json:"pong_count"`
	MissedPongs    int           `json:"missed_pongs"`
	Latency        time.Duration `json:"latency"`
	IsHealthy      bool          `json:"is_healthy"`
	LastHealthy    time.Time     `json:"last_healthy"`
}

// ConnectionMetrics는 연결 메트릭입니다
type ConnectionMetrics struct {
	MessagesReceived int64         `json:"messages_received"`
	MessagesSent     int64         `json:"messages_sent"`
	BytesReceived    int64         `json:"bytes_received"`
	BytesSent        int64         `json:"bytes_sent"`
	ErrorCount       int64         `json:"error_count"`
	ReconnectCount   int64         `json:"reconnect_count"`
	AverageLatency   time.Duration `json:"average_latency"`
	LastMessage      time.Time     `json:"last_message"`
}

// ConnectionState는 연결 상태입니다
type ConnectionState int

const (
	StateConnecting ConnectionState = iota
	StateConnected
	StateAuthenticated
	StateIdle
	StateReconnecting
	StateDisconnecting
	StateDisconnected
	StateError
)

// ConnectionStats는 전체 연결 통계입니다
type ConnectionStats struct {
	TotalConnections    int64 `json:"total_connections"`
	ActiveConnections   int64 `json:"active_connections"`
	AuthenticatedConns  int64 `json:"authenticated_connections"`
	TotalSessions       int64 `json:"total_sessions"`
	ActiveSessions      int64 `json:"active_sessions"`
	TotalUsers          int64 `json:"total_users"`
	ActiveUsers         int64 `json:"active_users"`
	TotalMessages       int64 `json:"total_messages"`
	MessagesPerSecond   float64 `json:"messages_per_second"`
	AverageConnTime     time.Duration `json:"average_connection_time"`
	PeakConnections     int64 `json:"peak_connections"`
	PeakConnectionsTime time.Time `json:"peak_connections_time"`
}

// ConnectionManagerConfig는 연결 매니저 설정입니다
type ConnectionManagerConfig struct {
	MaxConnections       int           `json:"max_connections"`
	MaxConnectionsPerUser int          `json:"max_connections_per_user"`
	HeartbeatInterval    time.Duration `json:"heartbeat_interval"`
	ConnectionTimeout    time.Duration `json:"connection_timeout"`
	IdleTimeout          time.Duration `json:"idle_timeout"`
	MaxMissedPongs       int           `json:"max_missed_pongs"`
	EnableReconnect      bool          `json:"enable_reconnect"`
	ReconnectAttempts    int           `json:"reconnect_attempts"`
	ReconnectDelay       time.Duration `json:"reconnect_delay"`
	StatsUpdateInterval  time.Duration `json:"stats_update_interval"`
	CleanupInterval      time.Duration `json:"cleanup_interval"`
}

// ConnectionEvent는 연결 이벤트입니다
type ConnectionEvent struct {
	Type       string                 `json:"type"`
	ConnectionID string               `json:"connection_id"`
	UserID     string                 `json:"user_id"`
	SessionID  string                 `json:"session_id,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
	Data       map[string]interface{} `json:"data,omitempty"`
}

// HeartbeatMessage는 하트비트 메시지입니다
type HeartbeatMessage struct {
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Sequence  int64     `json:"sequence"`
}

// ConnectionInfo는 연결 정보입니다
type ConnectionInfo struct {
	ConnectionID  string            `json:"connection_id"`
	UserID        string            `json:"user_id"`
	UserName      string            `json:"user_name"`
	SessionID     string            `json:"session_id,omitempty"`
	State         ConnectionState   `json:"state"`
	ConnectedAt   time.Time         `json:"connected_at"`
	LastActivity  time.Time         `json:"last_activity"`
	RemoteAddr    string            `json:"remote_addr"`
	UserAgent     string            `json:"user_agent"`
	Permission    Permission        `json:"permission"`
	Metrics       ConnectionMetrics `json:"metrics"`
	HealthCheck   ConnectionHealthCheck `json:"health_check"`
	Subscriptions []string          `json:"subscriptions"`
}

// DefaultConnectionManagerConfig는 기본 설정을 반환합니다
func DefaultConnectionManagerConfig() ConnectionManagerConfig {
	return ConnectionManagerConfig{
		MaxConnections:       1000,
		MaxConnectionsPerUser: 10,
		HeartbeatInterval:    30 * time.Second,
		ConnectionTimeout:    60 * time.Second,
		IdleTimeout:          5 * time.Minute,
		MaxMissedPongs:       3,
		EnableReconnect:      true,
		ReconnectAttempts:    5,
		ReconnectDelay:       time.Second,
		StatsUpdateInterval:  10 * time.Second,
		CleanupInterval:      time.Minute,
	}
}

// NewConnectionManager는 새로운 연결 매니저를 생성합니다
func NewConnectionManager(config ConnectionManagerConfig) *ConnectionManager {
	ctx, cancel := context.WithCancel(context.Background())
	
	cm := &ConnectionManager{
		connections:        make(map[string]*ManagedConnection),
		userConnections:    make(map[string]map[string]*ManagedConnection),
		sessionConnections: make(map[string]map[string]*ManagedConnection),
		config:            config,
		eventChan:         make(chan ConnectionEvent, 1000),
		ctx:               ctx,
		cancel:            cancel,
		statsUpdatedAt:    time.Now(),
	}
	
	// 백그라운드 작업 시작
	cm.wg.Add(3)
	go cm.statsUpdater()
	go cm.cleanupWorker()
	go cm.eventProcessor()
	
	return cm
}

// RegisterConnection은 새로운 연결을 등록합니다
func (cm *ConnectionManager) RegisterConnection(conn *websocket.Conn, userInfo *auth.UserInfo) (*ManagedConnection, error) {
	// 전체 연결 수 제한 확인
	cm.connectionsMutex.RLock()
	totalConnections := len(cm.connections)
	cm.connectionsMutex.RUnlock()
	
	if totalConnections >= cm.config.MaxConnections {
		return nil, fmt.Errorf("maximum connections exceeded")
	}
	
	// 사용자별 연결 수 제한 확인
	cm.userMutex.RLock()
	userConns := cm.userConnections[userInfo.ID]
	userConnCount := len(userConns)
	cm.userMutex.RUnlock()
	
	if userConnCount >= cm.config.MaxConnectionsPerUser {
		return nil, fmt.Errorf("maximum connections per user exceeded")
	}
	
	// 관리 연결 생성
	managed := cm.createManagedConnection(conn, userInfo)
	
	// 연결 등록
	cm.connectionsMutex.Lock()
	cm.connections[managed.ID] = managed
	cm.connectionsMutex.Unlock()
	
	// 사용자별 연결 등록
	cm.userMutex.Lock()
	if cm.userConnections[userInfo.ID] == nil {
		cm.userConnections[userInfo.ID] = make(map[string]*ManagedConnection)
	}
	cm.userConnections[userInfo.ID][managed.ID] = managed
	cm.userMutex.Unlock()
	
	// 통계 업데이트
	atomic.AddInt64(&cm.stats.TotalConnections, 1)
	atomic.AddInt64(&cm.stats.ActiveConnections, 1)
	
	// 이벤트 발생
	cm.publishEvent(ConnectionEvent{
		Type:         "connection_registered",
		ConnectionID: managed.ID,
		UserID:       userInfo.ID,
		Timestamp:    time.Now(),
		Data: map[string]interface{}{
			"remote_addr": conn.RemoteAddr().String(),
		},
	})
	
	// 연결 관리 시작
	managed.start()
	
	return managed, nil
}

// UnregisterConnection은 연결을 해제합니다
func (cm *ConnectionManager) UnregisterConnection(connectionID string) error {
	cm.connectionsMutex.Lock()
	managed, exists := cm.connections[connectionID]
	if exists {
		delete(cm.connections, connectionID)
	}
	cm.connectionsMutex.Unlock()
	
	if !exists {
		return fmt.Errorf("connection not found: %s", connectionID)
	}
	
	// 사용자별 연결에서 제거
	cm.userMutex.Lock()
	if userConns := cm.userConnections[managed.UserID]; userConns != nil {
		delete(userConns, connectionID)
		if len(userConns) == 0 {
			delete(cm.userConnections, managed.UserID)
		}
	}
	cm.userMutex.Unlock()
	
	// 세션별 연결에서 제거
	cm.removeFromAllSessions(connectionID)
	
	// 연결 정리
	managed.stop()
	
	// 통계 업데이트
	atomic.AddInt64(&cm.stats.ActiveConnections, -1)
	
	// 이벤트 발생
	cm.publishEvent(ConnectionEvent{
		Type:         "connection_unregistered",
		ConnectionID: connectionID,
		UserID:       managed.UserID,
		Timestamp:    time.Now(),
	})
	
	return nil
}

// AssignToSession은 연결을 세션에 할당합니다
func (cm *ConnectionManager) AssignToSession(connectionID, sessionID string) error {
	cm.connectionsMutex.RLock()
	managed, exists := cm.connections[connectionID]
	cm.connectionsMutex.RUnlock()
	
	if !exists {
		return fmt.Errorf("connection not found: %s", connectionID)
	}
	
	// 세션별 연결에 추가
	cm.sessionMutex.Lock()
	if cm.sessionConnections[sessionID] == nil {
		cm.sessionConnections[sessionID] = make(map[string]*ManagedConnection)
	}
	cm.sessionConnections[sessionID][connectionID] = managed
	cm.sessionMutex.Unlock()
	
	// 연결 상태 업데이트
	managed.stateMutex.Lock()
	if managed.state == StateConnected {
		managed.state = StateAuthenticated
	}
	managed.stateMutex.Unlock()
	
	// 이벤트 발생
	cm.publishEvent(ConnectionEvent{
		Type:         "connection_assigned_to_session",
		ConnectionID: connectionID,
		UserID:       managed.UserID,
		SessionID:    sessionID,
		Timestamp:    time.Now(),
	})
	
	return nil
}

// RemoveFromSession은 연결을 세션에서 제거합니다
func (cm *ConnectionManager) RemoveFromSession(connectionID, sessionID string) error {
	cm.sessionMutex.Lock()
	if sessionConns := cm.sessionConnections[sessionID]; sessionConns != nil {
		delete(sessionConns, connectionID)
		if len(sessionConns) == 0 {
			delete(cm.sessionConnections, sessionID)
		}
	}
	cm.sessionMutex.Unlock()
	
	// 이벤트 발생
	cm.publishEvent(ConnectionEvent{
		Type:         "connection_removed_from_session",
		ConnectionID: connectionID,
		SessionID:    sessionID,
		Timestamp:    time.Now(),
	})
	
	return nil
}

// GetConnection은 연결을 조회합니다
func (cm *ConnectionManager) GetConnection(connectionID string) (*ManagedConnection, error) {
	cm.connectionsMutex.RLock()
	defer cm.connectionsMutex.RUnlock()
	
	managed, exists := cm.connections[connectionID]
	if !exists {
		return nil, fmt.Errorf("connection not found: %s", connectionID)
	}
	
	return managed, nil
}

// GetUserConnections는 사용자의 모든 연결을 조회합니다
func (cm *ConnectionManager) GetUserConnections(userID string) []*ManagedConnection {
	cm.userMutex.RLock()
	defer cm.userMutex.RUnlock()
	
	userConns := cm.userConnections[userID]
	if userConns == nil {
		return []*ManagedConnection{}
	}
	
	connections := make([]*ManagedConnection, 0, len(userConns))
	for _, conn := range userConns {
		connections = append(connections, conn)
	}
	
	return connections
}

// GetSessionConnections는 세션의 모든 연결을 조회합니다
func (cm *ConnectionManager) GetSessionConnections(sessionID string) []*ManagedConnection {
	cm.sessionMutex.RLock()
	defer cm.sessionMutex.RUnlock()
	
	sessionConns := cm.sessionConnections[sessionID]
	if sessionConns == nil {
		return []*ManagedConnection{}
	}
	
	connections := make([]*ManagedConnection, 0, len(sessionConns))
	for _, conn := range sessionConns {
		connections = append(connections, conn)
	}
	
	return connections
}

// GetAllConnections는 모든 연결 정보를 조회합니다
func (cm *ConnectionManager) GetAllConnections() []ConnectionInfo {
	cm.connectionsMutex.RLock()
	defer cm.connectionsMutex.RUnlock()
	
	connections := make([]ConnectionInfo, 0, len(cm.connections))
	for _, managed := range cm.connections {
		info := cm.buildConnectionInfo(managed)
		connections = append(connections, info)
	}
	
	return connections
}

// GetStats는 연결 통계를 조회합니다
func (cm *ConnectionManager) GetStats() ConnectionStats {
	return cm.stats
}

// BroadcastToAll은 모든 연결에 메시지를 브로드캐스트합니다
func (cm *ConnectionManager) BroadcastToAll(message WebSocketMessage) {
	cm.connectionsMutex.RLock()
	defer cm.connectionsMutex.RUnlock()
	
	messageBytes, err := json.Marshal(message)
	if err != nil {
		return
	}
	
	for _, managed := range cm.connections {
		if managed.IsActive {
			select {
			case managed.sendChan <- messageBytes:
			default:
				// 버퍼가 가득 찬 경우 연결 종료
				managed.forceClose()
			}
		}
	}
}

// BroadcastToSession은 세션의 모든 연결에 메시지를 브로드캐스트합니다
func (cm *ConnectionManager) BroadcastToSession(sessionID string, message WebSocketMessage) {
	connections := cm.GetSessionConnections(sessionID)
	
	messageBytes, err := json.Marshal(message)
	if err != nil {
		return
	}
	
	for _, managed := range connections {
		if managed.IsActive {
			select {
			case managed.sendChan <- messageBytes:
			default:
				// 버퍼가 가득 찬 경우 연결 종료
				managed.forceClose()
			}
		}
	}
}

// BroadcastToUser는 사용자의 모든 연결에 메시지를 브로드캐스트합니다
func (cm *ConnectionManager) BroadcastToUser(userID string, message WebSocketMessage) {
	connections := cm.GetUserConnections(userID)
	
	messageBytes, err := json.Marshal(message)
	if err != nil {
		return
	}
	
	for _, managed := range connections {
		if managed.IsActive {
			select {
			case managed.sendChan <- messageBytes:
			default:
				// 버퍼가 가득 찬 경우 연결 종료
				managed.forceClose()
			}
		}
	}
}

// Shutdown은 연결 매니저를 종료합니다
func (cm *ConnectionManager) Shutdown() {
	cm.cancel()
	
	// 모든 연결 종료
	cm.connectionsMutex.RLock()
	connections := make([]*ManagedConnection, 0, len(cm.connections))
	for _, managed := range cm.connections {
		connections = append(connections, managed)
	}
	cm.connectionsMutex.RUnlock()
	
	for _, managed := range connections {
		managed.forceClose()
	}
	
	cm.wg.Wait()
	close(cm.eventChan)
}

// 내부 메서드들

func (cm *ConnectionManager) createManagedConnection(conn *websocket.Conn, userInfo *auth.UserInfo) *ManagedConnection {
	clientConn := &ClientConnection{
		ID:          fmt.Sprintf("conn_%d", time.Now().UnixNano()),
		UserID:      userInfo.ID,
		UserName:    userInfo.Username,
		Conn:        conn,
		Permission:  PermissionRead, // 기본 권한
		ConnectedAt: time.Now(),
		LastSeen:    time.Now(),
		IsActive:    true,
		sendChan:    make(chan []byte, 256),
		closeChan:   make(chan struct{}),
	}
	
	ctx, cancel := context.WithCancel(cm.ctx)
	clientConn.ctx = ctx
	clientConn.cancel = cancel
	
	managed := &ManagedConnection{
		ClientConnection: clientConn,
		manager:         cm,
		healthCheck: &ConnectionHealthCheck{
			IsHealthy:   true,
			LastHealthy: time.Now(),
		},
		state:         StateConnected,
		subscriptions: make(map[string]bool),
		heartbeatTicker: time.NewTicker(cm.config.HeartbeatInterval),
	}
	
	return managed
}

func (cm *ConnectionManager) buildConnectionInfo(managed *ManagedConnection) ConnectionInfo {
	managed.stateMutex.RLock()
	state := managed.state
	managed.stateMutex.RUnlock()
	
	managed.subMutex.RLock()
	subscriptions := make([]string, 0, len(managed.subscriptions))
	for sub := range managed.subscriptions {
		subscriptions = append(subscriptions, sub)
	}
	managed.subMutex.RUnlock()
	
	return ConnectionInfo{
		ConnectionID:  managed.ID,
		UserID:        managed.UserID,
		UserName:      managed.UserName,
		State:         state,
		ConnectedAt:   managed.ConnectedAt,
		LastActivity:  managed.LastSeen,
		RemoteAddr:    managed.Conn.RemoteAddr().String(),
		Permission:    managed.Permission,
		Metrics:       managed.metrics,
		HealthCheck:   *managed.healthCheck,
		Subscriptions: subscriptions,
	}
}

func (cm *ConnectionManager) removeFromAllSessions(connectionID string) {
	cm.sessionMutex.Lock()
	defer cm.sessionMutex.Unlock()
	
	for sessionID, sessionConns := range cm.sessionConnections {
		if _, exists := sessionConns[connectionID]; exists {
			delete(sessionConns, connectionID)
			if len(sessionConns) == 0 {
				delete(cm.sessionConnections, sessionID)
			}
		}
	}
}

func (cm *ConnectionManager) publishEvent(event ConnectionEvent) {
	select {
	case cm.eventChan <- event:
	default:
		// 이벤트 채널이 가득 찬 경우 이벤트 손실
	}
}

func (cm *ConnectionManager) statsUpdater() {
	defer cm.wg.Done()
	
	ticker := time.NewTicker(cm.config.StatsUpdateInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-cm.ctx.Done():
			return
		case <-ticker.C:
			cm.updateStats()
		}
	}
}

func (cm *ConnectionManager) updateStats() {
	now := time.Now()
	
	cm.connectionsMutex.RLock()
	activeConnections := int64(len(cm.connections))
	cm.connectionsMutex.RUnlock()
	
	cm.userMutex.RLock()
	activeUsers := int64(len(cm.userConnections))
	cm.userMutex.RUnlock()
	
	cm.sessionMutex.RLock()
	activeSessions := int64(len(cm.sessionConnections))
	cm.sessionMutex.RUnlock()
	
	// 피크 연결 수 업데이트
	if activeConnections > cm.stats.PeakConnections {
		cm.stats.PeakConnections = activeConnections
		cm.stats.PeakConnectionsTime = now
	}
	
	// 통계 업데이트
	cm.stats.ActiveConnections = activeConnections
	cm.stats.ActiveUsers = activeUsers
	cm.stats.ActiveSessions = activeSessions
	cm.statsUpdatedAt = now
}

func (cm *ConnectionManager) cleanupWorker() {
	defer cm.wg.Done()
	
	ticker := time.NewTicker(cm.config.CleanupInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-cm.ctx.Done():
			return
		case <-ticker.C:
			cm.cleanupInactiveConnections()
		}
	}
}

func (cm *ConnectionManager) cleanupInactiveConnections() {
	now := time.Now()
	idleTimeout := cm.config.IdleTimeout
	
	cm.connectionsMutex.RLock()
	var inactiveConnections []string
	for id, managed := range cm.connections {
		if !managed.IsActive || now.Sub(managed.LastSeen) > idleTimeout {
			inactiveConnections = append(inactiveConnections, id)
		}
	}
	cm.connectionsMutex.RUnlock()
	
	// 비활성 연결 정리
	for _, id := range inactiveConnections {
		cm.UnregisterConnection(id)
	}
}

func (cm *ConnectionManager) eventProcessor() {
	defer cm.wg.Done()
	
	for {
		select {
		case <-cm.ctx.Done():
			return
		case event := <-cm.eventChan:
			cm.processEvent(event)
		}
	}
}

func (cm *ConnectionManager) processEvent(event ConnectionEvent) {
	// 이벤트 로깅 및 처리
	// 실제 구현에서는 이벤트 기반 로직 추가
}

// ManagedConnection 메서드들

func (mc *ManagedConnection) start() {
	go mc.heartbeatLoop()
	go mc.readLoop()
	go mc.writeLoop()
}

func (mc *ManagedConnection) stop() {
	mc.heartbeatTicker.Stop()
	if mc.timeoutTimer != nil {
		mc.timeoutTimer.Stop()
	}
	mc.cancel()
}

func (mc *ManagedConnection) forceClose() {
	mc.IsActive = false
	mc.stop()
	mc.Conn.Close()
}

func (mc *ManagedConnection) heartbeatLoop() {
	for {
		select {
		case <-mc.ctx.Done():
			return
		case <-mc.heartbeatTicker.C:
			mc.sendHeartbeat()
		}
	}
}

func (mc *ManagedConnection) sendHeartbeat() {
	heartbeat := HeartbeatMessage{
		Type:      "ping",
		Timestamp: time.Now(),
		Sequence:  atomic.AddInt64(&mc.healthCheck.PingCount, 1),
	}
	
	heartbeatBytes, err := json.Marshal(heartbeat)
	if err != nil {
		return
	}
	
	mc.healthCheck.LastPing = time.Now()
	
	select {
	case mc.sendChan <- heartbeatBytes:
	default:
		// 버퍼가 가득 찬 경우 연결 문제
		mc.healthCheck.MissedPongs++
		if mc.healthCheck.MissedPongs >= mc.manager.config.MaxMissedPongs {
			mc.forceClose()
		}
	}
}

func (mc *ManagedConnection) readLoop() {
	for {
		select {
		case <-mc.ctx.Done():
			return
		default:
			_, message, err := mc.Conn.ReadMessage()
			if err != nil {
				mc.forceClose()
				return
			}
			
			mc.LastSeen = time.Now()
			mc.metrics.MessagesReceived++
			mc.metrics.BytesReceived += int64(len(message))
			mc.metrics.LastMessage = time.Now()
			
			// 메시지 처리 (실제 구현에서는 메시지 라우터로 전달)
		}
	}
}

func (mc *ManagedConnection) writeLoop() {
	for {
		select {
		case <-mc.ctx.Done():
			return
		case <-mc.closeChan:
			return
		case message := <-mc.sendChan:
			mc.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := mc.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				mc.forceClose()
				return
			}
			
			mc.metrics.MessagesSent++
			mc.metrics.BytesSent += int64(len(message))
		}
	}
}

// Subscribe는 토픽을 구독합니다
func (mc *ManagedConnection) Subscribe(topic string) {
	mc.subMutex.Lock()
	mc.subscriptions[topic] = true
	mc.subMutex.Unlock()
}

// Unsubscribe는 토픽 구독을 해제합니다
func (mc *ManagedConnection) Unsubscribe(topic string) {
	mc.subMutex.Lock()
	delete(mc.subscriptions, topic)
	mc.subMutex.Unlock()
}

// IsSubscribed는 토픽 구독 여부를 확인합니다
func (mc *ManagedConnection) IsSubscribed(topic string) bool {
	mc.subMutex.RLock()
	defer mc.subMutex.RUnlock()
	return mc.subscriptions[topic]
}