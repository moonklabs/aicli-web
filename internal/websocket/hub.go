package websocket

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Hub WebSocket 연결 허브
type Hub struct {
	// 클라이언트 관리
	clients    map[string]*Client
	clientsMu  sync.RWMutex
	
	// 채널 구독 관리
	channels    map[string]map[string]*Client // channel -> clientID -> client
	channelsMu  sync.RWMutex
	
	// 메시지 채널
	register   chan *Client
	unregister chan *Client
	broadcast  chan *BroadcastMessage
	
	// 설정
	config *HubConfig
	
	// 상태 관리
	running bool
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	
	// 통계
	stats *HubStats
}

// HubConfig 허브 설정
type HubConfig struct {
	MaxClients        int           // 최대 클라이언트 수
	CleanupInterval   time.Duration // 정리 주기
	StatsInterval     time.Duration // 통계 업데이트 주기
	BroadcastBuffer   int           // 브로드캐스트 버퍼 크기
	HeartbeatInterval time.Duration // 하트비트 간격
}

// DefaultHubConfig 기본 허브 설정
func DefaultHubConfig() *HubConfig {
	return &HubConfig{
		MaxClients:        1000,
		CleanupInterval:   30 * time.Second,
		StatsInterval:     10 * time.Second,
		BroadcastBuffer:   1000,
		HeartbeatInterval: 30 * time.Second,
	}
}

// HubStats 허브 통계
type HubStats struct {
	mu                   sync.RWMutex
	ConnectedClients     int                            `json:"connected_clients"`
	AuthenticatedClients int                            `json:"authenticated_clients"`
	TotalConnections     int64                          `json:"total_connections"`
	TotalDisconnections  int64                          `json:"total_disconnections"`
	MessagesSent         int64                          `json:"messages_sent"`
	MessagesReceived     int64                          `json:"messages_received"`
	ChannelSubscriptions map[string]int                 `json:"channel_subscriptions"`
	ClientsByUser        map[string]int                 `json:"clients_by_user"`
	StartTime            time.Time                      `json:"start_time"`
	LastUpdate           time.Time                      `json:"last_update"`
}

// BroadcastMessage 브로드캐스트 메시지
type BroadcastMessage struct {
	Message  *Message
	Channels []string
	UserIDs  []string
	Exclude  []string // 제외할 클라이언트 ID
}

// NewHub 새 허브 생성
func NewHub(config *HubConfig) *Hub {
	if config == nil {
		config = DefaultHubConfig()
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	return &Hub{
		clients:    make(map[string]*Client),
		channels:   make(map[string]map[string]*Client),
		register:   make(chan *Client, config.BroadcastBuffer),
		unregister: make(chan *Client, config.BroadcastBuffer),
		broadcast:  make(chan *BroadcastMessage, config.BroadcastBuffer),
		config:     config,
		running:    false,
		ctx:        ctx,
		cancel:     cancel,
		stats: &HubStats{
			ChannelSubscriptions: make(map[string]int),
			ClientsByUser:        make(map[string]int),
			StartTime:            time.Now(),
		},
	}
}

// Start 허브 시작
func (h *Hub) Start() error {
	if h.running {
		return nil
	}
	
	h.running = true
	h.stats.StartTime = time.Now()
	
	// 메인 루프 시작
	h.wg.Add(1)
	go h.run()
	
	// 정리 루틴 시작
	h.wg.Add(1)
	go h.cleanupRoutine()
	
	// 통계 업데이트 루틴 시작
	h.wg.Add(1)
	go h.statsRoutine()
	
	// 하트비트 루틴 시작
	h.wg.Add(1)
	go h.heartbeatRoutine()
	
	log.Println("WebSocket 허브 시작됨")
	return nil
}

// Stop 허브 중지
func (h *Hub) Stop() {
	if !h.running {
		return
	}
	
	h.running = false
	h.cancel()
	
	// 모든 클라이언트 연결 해제
	h.clientsMu.RLock()
	for _, client := range h.clients {
		client.Stop()
	}
	h.clientsMu.RUnlock()
	
	// 고루틴 종료 대기
	h.wg.Wait()
	
	log.Println("WebSocket 허브 중지됨")
}

// Register 클라이언트 등록
func (h *Hub) Register(client *Client) error {
	if !h.running {
		return nil
	}
	
	// 최대 클라이언트 수 확인
	h.clientsMu.RLock()
	currentCount := len(h.clients)
	h.clientsMu.RUnlock()
	
	if currentCount >= h.config.MaxClients {
		return &HubError{
			Code:    "MAX_CLIENTS_EXCEEDED",
			Message: "최대 클라이언트 수 초과",
		}
	}
	
	select {
	case h.register <- client:
		return nil
	case <-h.ctx.Done():
		return &HubError{
			Code:    "HUB_STOPPED",
			Message: "허브가 중지됨",
		}
	}
}

// Unregister 클라이언트 등록 해제
func (h *Hub) Unregister(client *Client) {
	if !h.running {
		return
	}
	
	select {
	case h.unregister <- client:
	case <-h.ctx.Done():
	}
}

// Broadcast 메시지 브로드캐스트
func (h *Hub) Broadcast(message *Message, channels ...string) {
	if !h.running {
		return
	}
	
	broadcastMsg := &BroadcastMessage{
		Message:  message,
		Channels: channels,
	}
	
	select {
	case h.broadcast <- broadcastMsg:
	default:
		log.Println("브로드캐스트 버퍼 가득참, 메시지 버림")
	}
}

// BroadcastToUsers 특정 사용자들에게 메시지 브로드캐스트
func (h *Hub) BroadcastToUsers(message *Message, userIDs ...string) {
	if !h.running {
		return
	}
	
	broadcastMsg := &BroadcastMessage{
		Message: message,
		UserIDs: userIDs,
	}
	
	select {
	case h.broadcast <- broadcastMsg:
	default:
		log.Println("브로드캐스트 버퍼 가득함, 메시지 버림")
	}
}

// GetClient 클라이언트 조회
func (h *Hub) GetClient(clientID string) (*Client, bool) {
	h.clientsMu.RLock()
	defer h.clientsMu.RUnlock()
	
	client, exists := h.clients[clientID]
	return client, exists
}

// GetClientsByUser 사용자별 클라이언트 조회
func (h *Hub) GetClientsByUser(userID string) []*Client {
	h.clientsMu.RLock()
	defer h.clientsMu.RUnlock()
	
	var clients []*Client
	for _, client := range h.clients {
		if client.UserID == userID {
			clients = append(clients, client)
		}
	}
	return clients
}

// GetStats 허브 통계 반환
func (h *Hub) GetStats() *HubStats {
	h.stats.mu.RLock()
	defer h.stats.mu.RUnlock()
	
	// 복사본 반환
	statsCopy := *h.stats
	statsCopy.ChannelSubscriptions = make(map[string]int)
	statsCopy.ClientsByUser = make(map[string]int)
	
	for k, v := range h.stats.ChannelSubscriptions {
		statsCopy.ChannelSubscriptions[k] = v
	}
	for k, v := range h.stats.ClientsByUser {
		statsCopy.ClientsByUser[k] = v
	}
	
	return &statsCopy
}

// run 메인 루프
func (h *Hub) run() {
	defer h.wg.Done()
	
	for {
		select {
		case client := <-h.register:
			h.handleRegister(client)
			
		case client := <-h.unregister:
			h.handleUnregister(client)
			
		case broadcastMsg := <-h.broadcast:
			h.handleBroadcast(broadcastMsg)
			
		case <-h.ctx.Done():
			return
		}
	}
}

// handleRegister 클라이언트 등록 처리
func (h *Hub) handleRegister(client *Client) {
	h.clientsMu.Lock()
	h.clients[client.ID] = client
	h.clientsMu.Unlock()
	
	// 통계 업데이트
	h.stats.mu.Lock()
	h.stats.ConnectedClients = len(h.clients)
	h.stats.TotalConnections++
	h.stats.ClientsByUser[client.UserID]++
	h.stats.mu.Unlock()
	
	log.Printf("클라이언트 등록됨: %s (사용자: %s)", client.ID, client.UserID)
	
	// 클라이언트 시작
	client.Start()
}

// handleUnregister 클라이언트 등록 해제 처리
func (h *Hub) handleUnregister(client *Client) {
	h.clientsMu.Lock()
	if _, exists := h.clients[client.ID]; exists {
		delete(h.clients, client.ID)
		close(client.send)
	}
	h.clientsMu.Unlock()
	
	// 채널에서 제거
	h.removeClientFromChannels(client)
	
	// 통계 업데이트
	h.stats.mu.Lock()
	h.stats.ConnectedClients = len(h.clients)
	h.stats.TotalDisconnections++
	h.stats.ClientsByUser[client.UserID]--
	if h.stats.ClientsByUser[client.UserID] <= 0 {
		delete(h.stats.ClientsByUser, client.UserID)
	}
	h.stats.mu.Unlock()
	
	log.Printf("클라이언트 등록 해제됨: %s (사용자: %s)", client.ID, client.UserID)
}

// handleBroadcast 브로드캐스트 처리
func (h *Hub) handleBroadcast(broadcastMsg *BroadcastMessage) {
	messageData, err := broadcastMsg.Message.ToJSON()
	if err != nil {
		log.Printf("브로드캐스트 메시지 JSON 변환 실패: %v", err)
		return
	}
	
	sentCount := 0
	
	// 채널별 브로드캐스트
	if len(broadcastMsg.Channels) > 0 {
		sentCount += h.broadcastToChannels(messageData, broadcastMsg.Channels, broadcastMsg.Exclude)
	}
	
	// 사용자별 브로드캐스트
	if len(broadcastMsg.UserIDs) > 0 {
		sentCount += h.broadcastToUsers(messageData, broadcastMsg.UserIDs, broadcastMsg.Exclude)
	}
	
	// 통계 업데이트
	h.stats.mu.Lock()
	h.stats.MessagesSent += int64(sentCount)
	h.stats.mu.Unlock()
}

// broadcastToChannels 채널별 브로드캐스트
func (h *Hub) broadcastToChannels(messageData []byte, channels []string, exclude []string) int {
	h.channelsMu.RLock()
	defer h.channelsMu.RUnlock()
	
	excludeMap := make(map[string]bool)
	for _, id := range exclude {
		excludeMap[id] = true
	}
	
	sentCount := 0
	sent := make(map[string]bool) // 중복 전송 방지
	
	for _, channel := range channels {
		if clients, exists := h.channels[channel]; exists {
			for clientID, client := range clients {
				if !excludeMap[clientID] && !sent[clientID] && client.Send(messageData) {
					sent[clientID] = true
					sentCount++
				}
			}
		}
	}
	
	return sentCount
}

// broadcastToUsers 사용자별 브로드캐스트
func (h *Hub) broadcastToUsers(messageData []byte, userIDs []string, exclude []string) int {
	h.clientsMu.RLock()
	defer h.clientsMu.RUnlock()
	
	excludeMap := make(map[string]bool)
	for _, id := range exclude {
		excludeMap[id] = true
	}
	
	sentCount := 0
	
	for _, userID := range userIDs {
		for _, client := range h.clients {
			if client.UserID == userID && !excludeMap[client.ID] && client.Send(messageData) {
				sentCount++
			}
		}
	}
	
	return sentCount
}

// removeClientFromChannels 채널에서 클라이언트 제거
func (h *Hub) removeClientFromChannels(client *Client) {
	h.channelsMu.Lock()
	defer h.channelsMu.Unlock()
	
	for channel := range client.channels {
		if clients, exists := h.channels[channel]; exists {
			delete(clients, client.ID)
			if len(clients) == 0 {
				delete(h.channels, channel)
			}
		}
	}
}

// subscribeClientToChannel 클라이언트를 채널에 구독
func (h *Hub) subscribeClientToChannel(client *Client, channel string) {
	h.channelsMu.Lock()
	defer h.channelsMu.Unlock()
	
	if h.channels[channel] == nil {
		h.channels[channel] = make(map[string]*Client)
	}
	h.channels[channel][client.ID] = client
}

// cleanupRoutine 정리 루틴
func (h *Hub) cleanupRoutine() {
	defer h.wg.Done()
	
	ticker := time.NewTicker(h.config.CleanupInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			h.cleanup()
		case <-h.ctx.Done():
			return
		}
	}
}

// cleanup 정리 작업
func (h *Hub) cleanup() {
	h.clientsMu.Lock()
	var disconnectedClients []*Client
	
	for _, client := range h.clients {
		if !client.IsConnected() {
			disconnectedClients = append(disconnectedClients, client)
		}
	}
	h.clientsMu.Unlock()
	
	// 연결 해제된 클라이언트 제거
	for _, client := range disconnectedClients {
		h.Unregister(client)
	}
	
	if len(disconnectedClients) > 0 {
		log.Printf("정리됨: %d개 클라이언트", len(disconnectedClients))
	}
}

// statsRoutine 통계 업데이트 루틴
func (h *Hub) statsRoutine() {
	defer h.wg.Done()
	
	ticker := time.NewTicker(h.config.StatsInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			h.updateStats()
		case <-h.ctx.Done():
			return
		}
	}
}

// updateStats 통계 업데이트
func (h *Hub) updateStats() {
	h.stats.mu.Lock()
	defer h.stats.mu.Unlock()
	
	// 채널 구독 통계
	h.channelsMu.RLock()
	h.stats.ChannelSubscriptions = make(map[string]int)
	for channel, clients := range h.channels {
		h.stats.ChannelSubscriptions[channel] = len(clients)
	}
	h.channelsMu.RUnlock()
	
	// 인증된 클라이언트 수
	h.clientsMu.RLock()
	authenticatedCount := 0
	for _, client := range h.clients {
		if client.IsAuthenticated() {
			authenticatedCount++
		}
	}
	h.stats.AuthenticatedClients = authenticatedCount
	h.clientsMu.RUnlock()
	
	h.stats.LastUpdate = time.Now()
}

// heartbeatRoutine 하트비트 루틴
func (h *Hub) heartbeatRoutine() {
	defer h.wg.Done()
	
	ticker := time.NewTicker(h.config.HeartbeatInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			h.sendHeartbeat()
		case <-h.ctx.Done():
			return
		}
	}
}

// sendHeartbeat 하트비트 전송
func (h *Hub) sendHeartbeat() {
	heartbeatMsg := NewMessage(MessageTypePing, map[string]interface{}{
		"timestamp": time.Now(),
		"server":    "hub",
	})
	
	h.Broadcast(heartbeatMsg, ChannelBroadcast)
}

// HubError 허브 에러
type HubError struct {
	Code    string
	Message string
}

func (e *HubError) Error() string {
	return e.Message
}

// GenerateClientID 클라이언트 ID 생성
func GenerateClientID() string {
	return uuid.New().String()
}