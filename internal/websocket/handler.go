package websocket

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/aicli/aicli-web/internal/auth"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// WebSocketHandler WebSocket 핸들러
type WebSocketHandler struct {
	hub           *Hub
	authenticator Authenticator
	upgrader      websocket.Upgrader
	config        *HandlerConfig
}

// HandlerConfig 핸들러 설정
type HandlerConfig struct {
	// CORS 설정
	CheckOrigin func(r *http.Request) bool
	
	// WebSocket 설정
	ReadBufferSize  int
	WriteBufferSize int
	
	// 타임아웃 설정
	HandshakeTimeout time.Duration
	
	// 압축 설정
	EnableCompression bool
	
	// 서브프로토콜
	Subprotocols []string
}

// DefaultHandlerConfig 기본 핸들러 설정
func DefaultHandlerConfig() *HandlerConfig {
	return &HandlerConfig{
		CheckOrigin: func(r *http.Request) bool {
			// 개발 환경에서는 모든 오리진 허용
			// 프로덕션에서는 적절한 검증 필요
			return true
		},
		ReadBufferSize:    4096,
		WriteBufferSize:   4096,
		HandshakeTimeout:  10 * time.Second,
		EnableCompression: false,
		Subprotocols:      []string{"aicli-ws-v1"},
	}
}

// NewWebSocketHandler 새 WebSocket 핸들러 생성
func NewWebSocketHandler(hub *Hub, jwtManager *auth.JWTManager, blacklist *auth.Blacklist, config *HandlerConfig) *WebSocketHandler {
	if config == nil {
		config = DefaultHandlerConfig()
	}
	
	// JWT 인증기 생성
	authenticator := NewJWTAuthenticator(jwtManager, blacklist)
	
	// WebSocket 업그레이더 설정
	upgrader := websocket.Upgrader{
		ReadBufferSize:    config.ReadBufferSize,
		WriteBufferSize:   config.WriteBufferSize,
		HandshakeTimeout:  config.HandshakeTimeout,
		EnableCompression: config.EnableCompression,
		Subprotocols:      config.Subprotocols,
		CheckOrigin:       config.CheckOrigin,
	}
	
	return &WebSocketHandler{
		hub:           hub,
		authenticator: authenticator,
		upgrader:      upgrader,
		config:        config,
	}
}

// HandleConnection WebSocket 연결 처리 (Gin 핸들러)
func (wsh *WebSocketHandler) HandleConnection(c *gin.Context) {
	// WebSocket으로 업그레이드
	conn, err := wsh.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket 업그레이드 실패: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "WebSocket upgrade failed",
			"message": err.Error(),
		})
		return
	}
	
	// 연결 처리
	wsh.handleNewConnection(c.Request, conn)
}

// HandleConnectionHTTP WebSocket 연결 처리 (표준 HTTP 핸들러)
func (wsh *WebSocketHandler) HandleConnectionHTTP(w http.ResponseWriter, r *http.Request) {
	// WebSocket으로 업그레이드
	conn, err := wsh.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket 업그레이드 실패: %v", err)
		http.Error(w, "WebSocket upgrade failed: "+err.Error(), http.StatusBadRequest)
		return
	}
	
	// 연결 처리
	wsh.handleNewConnection(r, conn)
}

// handleNewConnection 새 연결 처리
func (wsh *WebSocketHandler) handleNewConnection(r *http.Request, conn *websocket.Conn) {
	// 클라이언트 ID 생성
	clientID := GenerateClientID()
	
	// 연결 시점에서 사용자 추출 시도 (선택적)
	userID := "anonymous"
	if authInfo, err := wsh.authenticator.AuthenticateConnection(r); err == nil {
		userID = authInfo.UserID
		log.Printf("사전 인증된 WebSocket 연결: 사용자 %s", userID)
	} else {
		log.Printf("WebSocket 연결 (인증 대기): %s", clientID)
	}
	
	// 클라이언트 생성
	client := NewClient(clientID, userID, conn, wsh.hub, nil)
	
	// 허브에 등록
	if err := wsh.hub.Register(client); err != nil {
		log.Printf("클라이언트 등록 실패: %v", err)
		client.SendError("REGISTRATION_FAILED", "클라이언트 등록 실패", err.Error())
		conn.Close()
		return
	}
	
	log.Printf("새 WebSocket 연결: %s (사용자: %s, IP: %s)", 
		clientID, userID, getClientIP(r))
}

// BroadcastManager 브로드캐스트 관리자
type BroadcastManager struct {
	hub           *Hub
	authenticator Authenticator
}

// NewBroadcastManager 새 브로드캐스트 관리자 생성
func NewBroadcastManager(hub *Hub, authenticator Authenticator) *BroadcastManager {
	return &BroadcastManager{
		hub:           hub,
		authenticator: authenticator,
	}
}

// BroadcastToWorkspace 워크스페이스에 메시지 브로드캐스트
func (bm *BroadcastManager) BroadcastToWorkspace(workspaceID string, message *Message) {
	channel := GetWorkspaceChannel(workspaceID)
	bm.hub.Broadcast(message, channel)
}

// BroadcastToSession 세션에 메시지 브로드캐스트
func (bm *BroadcastManager) BroadcastToSession(sessionID string, message *Message) {
	channel := GetSessionChannel(sessionID)
	bm.hub.Broadcast(message, channel)
}

// BroadcastToTask 태스크에 메시지 브로드캐스트
func (bm *BroadcastManager) BroadcastToTask(taskID string, message *Message) {
	channel := GetTaskChannel(taskID)
	bm.hub.Broadcast(message, channel)
}

// BroadcastToUser 사용자에게 메시지 브로드캐스트
func (bm *BroadcastManager) BroadcastToUser(userID string, message *Message) {
	bm.hub.BroadcastToUsers(message, userID)
}

// BroadcastToAll 모든 사용자에게 메시지 브로드캐스트
func (bm *BroadcastManager) BroadcastToAll(message *Message) {
	bm.hub.Broadcast(message, ChannelBroadcast)
}

// BroadcastSystemMessage 시스템 메시지 브로드캐스트
func (bm *BroadcastManager) BroadcastSystemMessage(level, message, source string) {
	logMsg := NewLogMessage(level, message, source, "", "")
	bm.hub.Broadcast(logMsg, ChannelSystem)
}

// EventHandler 이벤트 핸들러 인터페이스
type EventHandler interface {
	// HandleTaskUpdate 태스크 업데이트 이벤트 처리
	HandleTaskUpdate(taskID, sessionID, status, output, error string)
	
	// HandleSessionUpdate 세션 업데이트 이벤트 처리
	HandleSessionUpdate(sessionID, projectID, status string, data map[string]interface{})
	
	// HandleLogStream 로그 스트림 이벤트 처리
	HandleLogStream(level, message, source, sessionID, taskID string)
	
	// HandleSystemEvent 시스템 이벤트 처리
	HandleSystemEvent(eventType, source string, data map[string]interface{})
}

// DefaultEventHandler 기본 이벤트 핸들러
type DefaultEventHandler struct {
	broadcastManager *BroadcastManager
}

// NewDefaultEventHandler 새 기본 이벤트 핸들러 생성
func NewDefaultEventHandler(broadcastManager *BroadcastManager) *DefaultEventHandler {
	return &DefaultEventHandler{
		broadcastManager: broadcastManager,
	}
}

// HandleTaskUpdate 태스크 업데이트 이벤트 처리
func (eh *DefaultEventHandler) HandleTaskUpdate(taskID, sessionID, status, output, error string) {
	// 태스크 업데이트 메시지 생성
	taskMsg := NewTaskMessage(taskID, sessionID, status, output, error, nil)
	
	// 세션 채널에 브로드캐스트
	eh.broadcastManager.BroadcastToSession(sessionID, taskMsg)
	
	// 태스크 채널에 브로드캐스트
	eh.broadcastManager.BroadcastToTask(taskID, taskMsg)
	
	log.Printf("태스크 업데이트 브로드캐스트: %s (상태: %s)", taskID, status)
}

// HandleSessionUpdate 세션 업데이트 이벤트 처리
func (eh *DefaultEventHandler) HandleSessionUpdate(sessionID, projectID, status string, data map[string]interface{}) {
	// 세션 업데이트 메시지 생성
	sessionMsg := NewSessionMessage(sessionID, projectID, status, data)
	
	// 세션 채널에 브로드캐스트
	eh.broadcastManager.BroadcastToSession(sessionID, sessionMsg)
	
	log.Printf("세션 업데이트 브로드캐스트: %s (상태: %s)", sessionID, status)
}

// HandleLogStream 로그 스트림 이벤트 처리
func (eh *DefaultEventHandler) HandleLogStream(level, message, source, sessionID, taskID string) {
	// 로그 메시지 생성
	logMsg := NewLogMessage(level, message, source, sessionID, taskID)
	
	// 관련 채널들에 브로드캐스트
	if sessionID != "" {
		eh.broadcastManager.BroadcastToSession(sessionID, logMsg)
	}
	
	if taskID != "" {
		eh.broadcastManager.BroadcastToTask(taskID, logMsg)
	}
	
	// 시스템 로그는 시스템 채널에도 브로드캐스트
	if source == "system" {
		eh.broadcastManager.hub.Broadcast(logMsg, ChannelSystem)
	}
}

// HandleSystemEvent 시스템 이벤트 처리
func (eh *DefaultEventHandler) HandleSystemEvent(eventType, source string, data map[string]interface{}) {
	// 시스템 이벤트 메시지 생성
	eventMsg := NewMessage(MessageTypeEvent, EventMessage{
		Type:   eventType,
		Source: source,
		Data:   data,
	})
	
	// 시스템 채널에 브로드캐스트
	eh.broadcastManager.hub.Broadcast(eventMsg, ChannelSystem)
	
	log.Printf("시스템 이벤트 브로드캐스트: %s (출처: %s)", eventType, source)
}

// MetricsCollector 메트릭 수집기
type MetricsCollector struct {
	hub *Hub
}

// NewMetricsCollector 새 메트릭 수집기 생성
func NewMetricsCollector(hub *Hub) *MetricsCollector {
	return &MetricsCollector{
		hub: hub,
	}
}

// GetConnectionMetrics 연결 메트릭 조회
func (mc *MetricsCollector) GetConnectionMetrics() map[string]interface{} {
	stats := mc.hub.GetStats()
	
	return map[string]interface{}{
		"connected_clients":      stats.ConnectedClients,
		"authenticated_clients":  stats.AuthenticatedClients,
		"total_connections":      stats.TotalConnections,
		"total_disconnections":   stats.TotalDisconnections,
		"messages_sent":          stats.MessagesSent,
		"messages_received":      stats.MessagesReceived,
		"channel_subscriptions":  stats.ChannelSubscriptions,
		"clients_by_user":        stats.ClientsByUser,
		"uptime":                 time.Since(stats.StartTime).String(),
		"last_update":            stats.LastUpdate,
	}
}

// GetClientMetrics 클라이언트 메트릭 조회
func (mc *MetricsCollector) GetClientMetrics(clientID string) (map[string]interface{}, error) {
	client, exists := mc.hub.GetClient(clientID)
	if !exists {
		return nil, &HubError{
			Code:    "CLIENT_NOT_FOUND",
			Message: "클라이언트를 찾을 수 없습니다",
		}
	}
	
	return client.GetStats(), nil
}

// HealthChecker 헬스 체커
type HealthChecker struct {
	hub           *Hub
	maxClients    int
	maxMemoryMB   int64
}

// NewHealthChecker 새 헬스 체커 생성
func NewHealthChecker(hub *Hub, maxClients int, maxMemoryMB int64) *HealthChecker {
	return &HealthChecker{
		hub:         hub,
		maxClients:  maxClients,
		maxMemoryMB: maxMemoryMB,
	}
}

// CheckHealth 헬스 체크
func (hc *HealthChecker) CheckHealth() map[string]interface{} {
	stats := hc.hub.GetStats()
	
	status := "healthy"
	issues := []string{}
	
	// 클라이언트 수 확인
	if stats.ConnectedClients > hc.maxClients {
		status = "warning"
		issues = append(issues, "너무 많은 클라이언트 연결됨")
	}
	
	// TODO: 메모리 사용량 확인
	
	return map[string]interface{}{
		"status":            status,
		"issues":            issues,
		"connected_clients": stats.ConnectedClients,
		"max_clients":       hc.maxClients,
		"uptime":            time.Since(stats.StartTime).String(),
		"timestamp":         time.Now(),
	}
}

// ConnectionLimiter 연결 제한기
type ConnectionLimiter struct {
	maxConnections     int
	maxPerUser         int
	connectionCounts   map[string]int // userID -> count
	connectionCountsMu sync.RWMutex
}

// NewConnectionLimiter 새 연결 제한기 생성
func NewConnectionLimiter(maxConnections, maxPerUser int) *ConnectionLimiter {
	return &ConnectionLimiter{
		maxConnections:   maxConnections,
		maxPerUser:       maxPerUser,
		connectionCounts: make(map[string]int),
	}
}

// CanConnect 연결 가능 여부 확인
func (cl *ConnectionLimiter) CanConnect(userID string, totalConnections int) error {
	cl.connectionCountsMu.RLock()
	defer cl.connectionCountsMu.RUnlock()
	
	// 전체 연결 수 확인
	if totalConnections >= cl.maxConnections {
		return &HubError{
			Code:    "MAX_CONNECTIONS_EXCEEDED",
			Message: "최대 연결 수 초과",
		}
	}
	
	// 사용자별 연결 수 확인
	userConnections := cl.connectionCounts[userID]
	if userConnections >= cl.maxPerUser {
		return &HubError{
			Code:    "MAX_USER_CONNECTIONS_EXCEEDED",
			Message: "사용자별 최대 연결 수 초과",
		}
	}
	
	return nil
}

// AddConnection 연결 추가
func (cl *ConnectionLimiter) AddConnection(userID string) {
	cl.connectionCountsMu.Lock()
	defer cl.connectionCountsMu.Unlock()
	
	cl.connectionCounts[userID]++
}

// RemoveConnection 연결 제거
func (cl *ConnectionLimiter) RemoveConnection(userID string) {
	cl.connectionCountsMu.Lock()
	defer cl.connectionCountsMu.Unlock()
	
	if cl.connectionCounts[userID] > 0 {
		cl.connectionCounts[userID]--
		if cl.connectionCounts[userID] == 0 {
			delete(cl.connectionCounts, userID)
		}
	}
}

// GetConnectionCounts 연결 수 조회
func (cl *ConnectionLimiter) GetConnectionCounts() map[string]int {
	cl.connectionCountsMu.RLock()
	defer cl.connectionCountsMu.RUnlock()
	
	counts := make(map[string]int)
	for userID, count := range cl.connectionCounts {
		counts[userID] = count
	}
	return counts
}