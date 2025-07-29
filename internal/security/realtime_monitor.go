package security

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// RealtimeMonitorConfig는 실시간 모니터링 설정입니다.
type RealtimeMonitorConfig struct {
	EventTracker *EventTracker
	Logger       *zap.Logger

	// WebSocket 설정
	ReadBufferSize  int           // 읽기 버퍼 크기
	WriteBufferSize int           // 쓰기 버퍼 크기
	PingPeriod      time.Duration // Ping 주기
	PongWait        time.Duration // Pong 대기 시간
	WriteWait       time.Duration // 쓰기 대기 시간

	// 알림 설정
	AlertEnabled        bool        // 알림 활성화
	AlertThreshold      Severity    // 알림 최소 심각도
	MaxClientsPerUser   int         // 사용자당 최대 연결 수
	EventBufferSize     int         // 이벤트 버퍼 크기
	StatisticsInterval  time.Duration // 통계 전송 주기
}

// RealtimeMonitor는 실시간 보안 모니터링을 담당합니다.
type RealtimeMonitor struct {
	config       *RealtimeMonitorConfig
	eventTracker *EventTracker
	logger       *zap.Logger

	// WebSocket 관리
	upgrader websocket.Upgrader
	clients  map[string]*ClientConnection // userID -> connection
	mutex    sync.RWMutex

	// 이벤트 채널
	eventChan      chan *SecurityEvent
	statisticsChan chan *SecurityStatistics
	closeChan      chan struct{}
	wg             sync.WaitGroup
}

// ClientConnection은 클라이언트 WebSocket 연결을 관리합니다.
type ClientConnection struct {
	UserID     string          `json:"user_id"`
	Conn       *websocket.Conn `json:"-"`
	Send       chan []byte     `json:"-"`
	IsAdmin    bool            `json:"is_admin"`
	ConnectedAt time.Time      `json:"connected_at"`
	LastPing   time.Time       `json:"last_ping"`
}

// SecurityStatistics는 보안 통계 정보입니다.
type SecurityStatistics struct {
	Timestamp        time.Time         `json:"timestamp"`
	ActiveSessions   int               `json:"active_sessions"`
	TotalEvents      int               `json:"total_events"`
	EventsByType     map[string]int    `json:"events_by_type"`
	EventsBySeverity map[string]int    `json:"events_by_severity"`
	BlockedIPs       int               `json:"blocked_ips"`
	BlockedUsers     int               `json:"blocked_users"`
	AverageResponseTime time.Duration  `json:"average_response_time"`
	TopThreats       []ThreatSummary   `json:"top_threats"`
}

// ThreatSummary는 위협 요약 정보입니다.
type ThreatSummary struct {
	Type        string    `json:"type"`
	Count       int       `json:"count"`
	LastSeen    time.Time `json:"last_seen"`
	Severity    Severity  `json:"severity"`
	Description string    `json:"description"`
}

// RealtimeEvent는 실시간 전송되는 이벤트입니다.
type RealtimeEvent struct {
	Type      string      `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// DefaultRealtimeMonitorConfig는 기본 설정을 반환합니다.
func DefaultRealtimeMonitorConfig() *RealtimeMonitorConfig {
	return &RealtimeMonitorConfig{
		ReadBufferSize:      1024,
		WriteBufferSize:     1024,
		PingPeriod:          54 * time.Second,
		PongWait:            60 * time.Second,
		WriteWait:           10 * time.Second,
		AlertEnabled:        true,
		AlertThreshold:      SeverityHigh,
		MaxClientsPerUser:   3,
		EventBufferSize:     100,
		StatisticsInterval:  30 * time.Second,
	}
}

// NewRealtimeMonitor는 새로운 실시간 모니터를 생성합니다.
func NewRealtimeMonitor(config *RealtimeMonitorConfig) *RealtimeMonitor {
	if config == nil {
		config = DefaultRealtimeMonitorConfig()
	}

	monitor := &RealtimeMonitor{
		config:       config,
		eventTracker: config.EventTracker,
		logger:       config.Logger,
		clients:      make(map[string]*ClientConnection),
		eventChan:    make(chan *SecurityEvent, config.EventBufferSize),
		statisticsChan: make(chan *SecurityStatistics, 10),
		closeChan:    make(chan struct{}),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  config.ReadBufferSize,
			WriteBufferSize: config.WriteBufferSize,
			CheckOrigin: func(r *http.Request) bool {
				// 실제 환경에서는 더 엄격한 Origin 검사 필요
				return true
			},
		},
	}

	// 이벤트 처리 고루틴 시작
	monitor.wg.Add(1)
	go monitor.eventProcessor()

	// 통계 수집 고루틴 시작
	monitor.wg.Add(1)
	go monitor.statisticsCollector()

	return monitor
}

// HandleWebSocket은 WebSocket 연결을 처리합니다.
func (rm *RealtimeMonitor) HandleWebSocket() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := rm.extractUserID(c)
		isAdmin := rm.isAdmin(c)

		// 사용자 연결 수 제한 검사
		if rm.countUserConnections(userID) >= rm.config.MaxClientsPerUser {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "연결 수 제한 초과",
			})
			return
		}

		// WebSocket 연결 업그레이드
		conn, err := rm.upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			rm.logger.Error("WebSocket 업그레이드 실패", zap.Error(err))
			return
		}

		// 클라이언트 연결 생성
		client := &ClientConnection{
			UserID:      userID,
			Conn:        conn,
			Send:        make(chan []byte, 256),
			IsAdmin:     isAdmin,
			ConnectedAt: time.Now(),
			LastPing:    time.Now(),
		}

		// 클라이언트 등록
		rm.registerClient(client)

		// 연결 처리 고루틴 시작
		rm.wg.Add(2)
		go rm.handleClientRead(client)
		go rm.handleClientWrite(client)

		rm.logger.Info("WebSocket 연결 성공",
			zap.String("user_id", userID),
			zap.Bool("is_admin", isAdmin))
	}
}

// SendEvent는 보안 이벤트를 실시간으로 전송합니다.
func (rm *RealtimeMonitor) SendEvent(event *SecurityEvent) {
	select {
	case rm.eventChan <- event:
	default:
		rm.logger.Warn("이벤트 채널 가득참", zap.String("event_id", event.ID))
	}
}

// SendStatistics는 통계 정보를 전송합니다.
func (rm *RealtimeMonitor) SendStatistics(stats *SecurityStatistics) {
	select {
	case rm.statisticsChan <- stats:
	default:
		rm.logger.Warn("통계 채널 가득함")
	}
}

// eventProcessor는 이벤트를 처리하고 클라이언트에게 전송합니다.
func (rm *RealtimeMonitor) eventProcessor() {
	defer rm.wg.Done()

	for {
		select {
		case event := <-rm.eventChan:
			rm.broadcastEvent(event)

		case stats := <-rm.statisticsChan:
			rm.broadcastStatistics(stats)

		case <-rm.closeChan:
			return
		}
	}
}

// statisticsCollector는 주기적으로 통계를 수집합니다.
func (rm *RealtimeMonitor) statisticsCollector() {
	defer rm.wg.Done()

	ticker := time.NewTicker(rm.config.StatisticsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			stats := rm.collectStatistics()
			rm.SendStatistics(stats)

		case <-rm.closeChan:
			return
		}
	}
}

// broadcastEvent는 이벤트를 모든 적절한 클라이언트에게 전송합니다.
func (rm *RealtimeMonitor) broadcastEvent(event *SecurityEvent) {
	realtimeEvent := &RealtimeEvent{
		Type:      "security_event",
		Timestamp: time.Now(),
		Data:      event,
	}

	message, err := json.Marshal(realtimeEvent)
	if err != nil {
		rm.logger.Error("이벤트 직렬화 실패", zap.Error(err))
		return
	}

	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	for _, client := range rm.clients {
		// 권한 검사: 관리자는 모든 이벤트, 일반 사용자는 자신 관련 이벤트만
		if client.IsAdmin || event.UserID == client.UserID {
			select {
			case client.Send <- message:
			default:
				// 클라이언트 버퍼가 가득차면 연결 해제
				rm.unregisterClient(client)
			}
		}
	}
}

// broadcastStatistics는 통계를 모든 관리자 클라이언트에게 전송합니다.
func (rm *RealtimeMonitor) broadcastStatistics(stats *SecurityStatistics) {
	realtimeEvent := &RealtimeEvent{
		Type:      "security_statistics",
		Timestamp: time.Now(),
		Data:      stats,
	}

	message, err := json.Marshal(realtimeEvent)
	if err != nil {
		rm.logger.Error("통계 직렬화 실패", zap.Error(err))
		return
	}

	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	for _, client := range rm.clients {
		if client.IsAdmin {
			select {
			case client.Send <- message:
			default:
				rm.unregisterClient(client)
			}
		}
	}
}

// collectStatistics는 현재 보안 통계를 수집합니다.
func (rm *RealtimeMonitor) collectStatistics() *SecurityStatistics {
	ctx := context.Background()
	
	stats := &SecurityStatistics{
		Timestamp:           time.Now(),
		EventsByType:        make(map[string]int),
		EventsBySeverity:    make(map[string]int),
		TopThreats:          make([]ThreatSummary, 0),
	}

	// EventTracker가 있으면 통계 수집
	if rm.eventTracker != nil {
		if eventStats, err := rm.eventTracker.GetStatistics(ctx, 24*time.Hour); err == nil {
			stats.TotalEvents = eventStats.TotalEvents
			
			// EventType을 string으로 변환
			for eventType, count := range eventStats.EventsByType {
				stats.EventsByType[string(eventType)] = count
			}
			
			// Severity를 string으로 변환
			for severity, count := range eventStats.EventsBySeverity {
				stats.EventsBySeverity[string(severity)] = count
			}
		}
	}

	// 활성 세션 수
	rm.mutex.RLock()
	stats.ActiveSessions = len(rm.clients)
	rm.mutex.RUnlock()

	return stats
}

// handleClientRead는 클라이언트로부터 메시지를 읽습니다.
func (rm *RealtimeMonitor) handleClientRead(client *ClientConnection) {
	defer func() {
		rm.wg.Done()
		rm.unregisterClient(client)
		client.Conn.Close()
	}()

	client.Conn.SetReadDeadline(time.Now().Add(rm.config.PongWait))
	client.Conn.SetPongHandler(func(string) error {
		client.LastPing = time.Now()
		client.Conn.SetReadDeadline(time.Now().Add(rm.config.PongWait))
		return nil
	})

	for {
		var message map[string]interface{}
		err := client.Conn.ReadJSON(&message)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				rm.logger.Error("WebSocket 예상치 못한 종료", zap.Error(err))
			}
			break
		}

		// 클라이언트 메시지 처리
		rm.handleClientMessage(client, message)
	}
}

// handleClientWrite는 클라이언트에게 메시지를 보냅니다.
func (rm *RealtimeMonitor) handleClientWrite(client *ClientConnection) {
	ticker := time.NewTicker(rm.config.PingPeriod)
	defer func() {
		rm.wg.Done()
		ticker.Stop()
		client.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.Send:
			client.Conn.SetWriteDeadline(time.Now().Add(rm.config.WriteWait))
			if !ok {
				client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := client.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// 대기 중인 메시지가 있으면 함께 전송
			n := len(client.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-client.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			client.Conn.SetWriteDeadline(time.Now().Add(rm.config.WriteWait))
			if err := client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleClientMessage는 클라이언트 메시지를 처리합니다.
func (rm *RealtimeMonitor) handleClientMessage(client *ClientConnection, message map[string]interface{}) {
	msgType, ok := message["type"].(string)
	if !ok {
		return
	}

	switch msgType {
	case "subscribe":
		// 구독 처리 (필요시 구현)
	case "unsubscribe":
		// 구독 해제 처리 (필요시 구현)
	case "get_statistics":
		// 통계 요청 처리
		if client.IsAdmin {
			stats := rm.collectStatistics()
			rm.sendToClient(client, "security_statistics", stats)
		}
	}
}

// sendToClient는 특정 클라이언트에게 메시지를 전송합니다.
func (rm *RealtimeMonitor) sendToClient(client *ClientConnection, eventType string, data interface{}) {
	realtimeEvent := &RealtimeEvent{
		Type:      eventType,
		Timestamp: time.Now(),
		Data:      data,
	}

	message, err := json.Marshal(realtimeEvent)
	if err != nil {
		rm.logger.Error("메시지 직렬화 실패", zap.Error(err))
		return
	}

	select {
	case client.Send <- message:
	default:
		rm.unregisterClient(client)
	}
}

// registerClient는 클라이언트를 등록합니다.
func (rm *RealtimeMonitor) registerClient(client *ClientConnection) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	// 기존 연결이 있으면 해제
	if existingClient, exists := rm.clients[client.UserID]; exists {
		close(existingClient.Send)
		existingClient.Conn.Close()
	}

	rm.clients[client.UserID] = client

	rm.logger.Info("클라이언트 등록됨",
		zap.String("user_id", client.UserID),
		zap.Int("total_clients", len(rm.clients)))
}

// unregisterClient는 클라이언트를 해제합니다.
func (rm *RealtimeMonitor) unregisterClient(client *ClientConnection) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	if _, exists := rm.clients[client.UserID]; exists {
		delete(rm.clients, client.UserID)
		close(client.Send)

		rm.logger.Info("클라이언트 해제됨",
			zap.String("user_id", client.UserID),
			zap.Int("total_clients", len(rm.clients)))
	}
}

// GetConnectedClients는 연결된 클라이언트 목록을 반환합니다.
func (rm *RealtimeMonitor) GetConnectedClients() []*ClientConnection {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	clients := make([]*ClientConnection, 0, len(rm.clients))
	for _, client := range rm.clients {
		clients = append(clients, client)
	}

	return clients
}

// Close는 모니터를 종료합니다.
func (rm *RealtimeMonitor) Close() {
	close(rm.closeChan)

	// 모든 클라이언트 연결 해제
	rm.mutex.Lock()
	for _, client := range rm.clients {
		close(client.Send)
		client.Conn.Close()
	}
	rm.clients = make(map[string]*ClientConnection)
	rm.mutex.Unlock()

	// 고루틴 종료 대기
	rm.wg.Wait()

	rm.logger.Info("실시간 모니터 종료됨")
}

// 헬퍼 메서드들

func (rm *RealtimeMonitor) extractUserID(c *gin.Context) string {
	if userID, exists := c.Get("user_id"); exists {
		if str, ok := userID.(string); ok {
			return str
		}
	}
	return fmt.Sprintf("anonymous_%d", time.Now().UnixNano())
}

func (rm *RealtimeMonitor) isAdmin(c *gin.Context) bool {
	if role, exists := c.Get("user_role"); exists {
		if str, ok := role.(string); ok {
			return str == "admin" || str == "security_admin"
		}
	}
	return false
}

func (rm *RealtimeMonitor) countUserConnections(userID string) int {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	count := 0
	for _, client := range rm.clients {
		if client.UserID == userID {
			count++
		}
	}
	return count
}