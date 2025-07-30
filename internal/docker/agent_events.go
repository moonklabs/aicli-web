package docker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
)

// AgentEventMonitor 에이전트 Docker 이벤트 모니터링 시스템
type AgentEventMonitor struct {
	client        *Client
	mu            sync.RWMutex
	eventHandlers map[string][]EventHandler
	running       bool
	cancel        context.CancelFunc
	eventChan     chan AgentDockerEvent
}

// EventHandler 이벤트 처리 함수 타입
type EventHandler func(event AgentDockerEvent)

// AgentDockerEvent 에이전트 Docker 이벤트
type AgentDockerEvent struct {
	// Docker 기본 이벤트 정보
	Type   string    `json:"type"`   // container, network, volume, etc.
	Action string    `json:"action"` // create, start, stop, die, etc.
	Time   time.Time `json:"time"`
	TimeNano int64   `json:"time_nano"`

	// 이벤트 대상 정보
	Actor EventActor `json:"actor"`

	// 에이전트 관련 정보
	AgentID     string `json:"agent_id,omitempty"`
	ContainerID string `json:"container_id,omitempty"`
	NetworkID   string `json:"network_id,omitempty"`

	// 추가 속성
	Attributes map[string]string `json:"attributes"`
	Scope      string            `json:"scope"`
}

// EventActor 이벤트 액터 정보
type EventActor struct {
	ID         string            `json:"id"`
	Attributes map[string]string `json:"attributes"`
}

// EventFilter 이벤트 필터
type EventFilter struct {
	AgentID     string   `json:"agent_id,omitempty"`
	Types       []string `json:"types,omitempty"`       // container, network, volume
	Actions     []string `json:"actions,omitempty"`     // create, start, stop, die
	Since       string   `json:"since,omitempty"`       // 시작 시간
	Until       string   `json:"until,omitempty"`       // 종료 시간
	Labels      []string `json:"labels,omitempty"`      // 라벨 필터
}

// AgentEventStats 에이전트 이벤트 통계
type AgentEventStats struct {
	AgentID       string                    `json:"agent_id"`
	TotalEvents   int64                     `json:"total_events"`
	EventsByType  map[string]int64          `json:"events_by_type"`
	EventsByAction map[string]int64         `json:"events_by_action"`
	LastEvent     *AgentDockerEvent         `json:"last_event,omitempty"`
	LastEventTime time.Time                 `json:"last_event_time"`
	Uptime        time.Duration             `json:"uptime"`
	Errors        []string                  `json:"errors,omitempty"`
}

// NewAgentEventMonitor 새로운 에이전트 이벤트 모니터 생성
func NewAgentEventMonitor(client *Client) *AgentEventMonitor {
	return &AgentEventMonitor{
		client:        client,
		eventHandlers: make(map[string][]EventHandler),
		eventChan:     make(chan AgentDockerEvent, 1000), // 버퍼링된 채널
	}
}

// Start 이벤트 모니터링 시작
func (aem *AgentEventMonitor) Start(ctx context.Context) error {
	aem.mu.Lock()
	defer aem.mu.Unlock()

	if aem.running {
		return fmt.Errorf("event monitor already running")
	}

	// 컨텍스트 생성
	monitorCtx, cancel := context.WithCancel(ctx)
	aem.cancel = cancel

	// 에이전트 관련 이벤트만 필터링
	eventFilters := filters.NewArgs()
	eventFilters.Add("label", fmt.Sprintf("%s.managed=true", aem.client.labelPrefix))

	// Docker 이벤트 스트림 시작
	eventReader, err := aem.client.cli.Events(monitorCtx, types.EventsOptions{
		Filters: eventFilters,
	})
	if err != nil {
		cancel()
		return fmt.Errorf("start event stream: %w", err)
	}

	aem.running = true

	// 이벤트 처리 고루틴 시작
	go aem.processEvents(monitorCtx, eventReader)
	go aem.handleEvents(monitorCtx)

	return nil
}

// Stop 이벤트 모니터링 중지
func (aem *AgentEventMonitor) Stop() error {
	aem.mu.Lock()
	defer aem.mu.Unlock()

	if !aem.running {
		return nil
	}

	if aem.cancel != nil {
		aem.cancel()
	}

	aem.running = false
	close(aem.eventChan)

	return nil
}

// RegisterHandler 이벤트 핸들러 등록
func (aem *AgentEventMonitor) RegisterHandler(eventType string, handler EventHandler) {
	aem.mu.Lock()
	defer aem.mu.Unlock()

	if handlers, exists := aem.eventHandlers[eventType]; exists {
		aem.eventHandlers[eventType] = append(handlers, handler)
	} else {
		aem.eventHandlers[eventType] = []EventHandler{handler}
	}
}

// UnregisterHandler 이벤트 핸들러 해제
func (aem *AgentEventMonitor) UnregisterHandler(eventType string, handler EventHandler) {
	aem.mu.Lock()
	defer aem.mu.Unlock()

	_, exists := aem.eventHandlers[eventType]
	if !exists {
		return
	}

	// 핸들러 제거 (함수 포인터 비교는 복잡하므로 전체 목록 초기화)
	// 실제 구현에서는 핸들러 ID 시스템을 사용하는 것이 좋음
	aem.eventHandlers[eventType] = make([]EventHandler, 0)
}

// processEvents Docker 이벤트 스트림 처리
func (aem *AgentEventMonitor) processEvents(ctx context.Context, eventReader <-chan events.Message) {
	// 채널 기반 이벤트 처리

	for {
		select {
		case <-ctx.Done():
			return
		case dockerEvent, ok := <-eventReader:
			if !ok {
				return
			}

			// Docker 이벤트를 AgentDockerEvent로 변환
			agentEvent := aem.convertDockerEvent(dockerEvent)
			if agentEvent != nil {
				select {
				case aem.eventChan <- *agentEvent:
				case <-ctx.Done():
					return
				default:
					// 채널이 가득 찬 경우 이벤트 드롭 (백프레셔 방지)
				}
			}
		}
	}
}

// handleEvents 이벤트 핸들러 실행
func (aem *AgentEventMonitor) handleEvents(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-aem.eventChan:
			if !ok {
				return
			}

			// 이벤트 타입별 핸들러 실행
			aem.mu.RLock()
			if handlers, exists := aem.eventHandlers[event.Type]; exists {
				for _, handler := range handlers {
					go func(h EventHandler, e AgentDockerEvent) {
						defer func() {
							if r := recover(); r != nil {
								// 핸들러에서 패닉이 발생해도 모니터는 계속 실행
							}
						}()
						h(e)
					}(handler, event)
				}
			}

			// 모든 이벤트에 대한 핸들러 실행
			if handlers, exists := aem.eventHandlers["*"]; exists {
				for _, handler := range handlers {
					go func(h EventHandler, e AgentDockerEvent) {
						defer func() {
							if r := recover(); r != nil {
								// 핸들러에서 패닉이 발생해도 모니터는 계속 실행
							}
						}()
						h(e)
					}(handler, event)
				}
			}
			aem.mu.RUnlock()
		}
	}
}

// convertDockerEvent Docker 이벤트를 AgentDockerEvent로 변환
func (aem *AgentEventMonitor) convertDockerEvent(dockerEvent events.Message) *AgentDockerEvent {
	// 에이전트 관련 이벤트인지 확인
	agentID := aem.extractAgentID(dockerEvent.Actor.Attributes)
	if agentID == "" {
		return nil // 에이전트 관련 이벤트가 아님
	}

	event := &AgentDockerEvent{
		Type:     string(dockerEvent.Type),
		Action:   string(dockerEvent.Action),
		Time:     time.Unix(dockerEvent.Time, 0),
		TimeNano: dockerEvent.TimeNano,
		Actor: EventActor{
			ID:         dockerEvent.Actor.ID,
			Attributes: dockerEvent.Actor.Attributes,
		},
		AgentID:    agentID,
		Attributes: dockerEvent.Actor.Attributes,
		Scope:      dockerEvent.Scope,
	}

	// 이벤트 타입별 추가 정보 설정
	switch dockerEvent.Type {
	case "container":
		event.ContainerID = dockerEvent.Actor.ID
	case "network":
		event.NetworkID = dockerEvent.Actor.ID
	}

	return event
}

// extractAgentID 이벤트 속성에서 에이전트 ID 추출
func (aem *AgentEventMonitor) extractAgentID(attributes map[string]string) string {
	// 라벨에서 에이전트 ID 추출
	agentIDKey := fmt.Sprintf("%s.agent.id", aem.client.labelPrefix)
	if agentID, exists := attributes[agentIDKey]; exists {
		return agentID
	}

	// 컨테이너 이름에서 에이전트 ID 추출 (fallback)
	if name, exists := attributes["name"]; exists {
		if len(name) > 6 && name[:6] == "agent-" {
			return name[6:] // "agent-" 제거
		}
	}

	return ""
}

// GetEventHistory 이벤트 히스토리 조회
func (aem *AgentEventMonitor) GetEventHistory(ctx context.Context, filter EventFilter) ([]AgentDockerEvent, error) {
	// Docker 이벤트 필터 생성
	eventFilters := filters.NewArgs()
	
	// 에이전트 관련 이벤트만
	eventFilters.Add("label", fmt.Sprintf("%s.managed=true", aem.client.labelPrefix))
	
	if filter.AgentID != "" {
		eventFilters.Add("label", fmt.Sprintf("%s.agent.id=%s", aem.client.labelPrefix, filter.AgentID))
	}

	for _, eventType := range filter.Types {
		eventFilters.Add("type", eventType)
	}

	for _, action := range filter.Actions {
		eventFilters.Add("event", action)
	}

	for _, label := range filter.Labels {
		eventFilters.Add("label", label)
	}

	// 이벤트 조회 옵션
	options := types.EventsOptions{
		Filters: eventFilters,
	}

	if filter.Since != "" {
		options.Since = filter.Since
	}

	if filter.Until != "" {
		options.Until = filter.Until
	}

	// Docker 이벤트 조회
	_, err := aem.client.cli.Events(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("get events: %w", err)
	}
	// TODO: Docker 이벤트 파싱 로직 구현 필요
	// 현재는 빈 이벤트 리스트 반환
	return []AgentDockerEvent{}, nil
}

// GetAgentEventStats 에이전트 이벤트 통계 조회
func (aem *AgentEventMonitor) GetAgentEventStats(ctx context.Context, agentID string, since time.Time) (*AgentEventStats, error) {
	// 에이전트 이벤트 히스토리 조회
	filter := EventFilter{
		AgentID: agentID,
		Since:   fmt.Sprintf("%d", since.Unix()),
	}

	events, err := aem.GetEventHistory(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("get event history: %w", err)
	}

	// 통계 계산
	stats := &AgentEventStats{
		AgentID:        agentID,
		TotalEvents:    int64(len(events)),
		EventsByType:   make(map[string]int64),
		EventsByAction: make(map[string]int64),
		Uptime:         time.Since(since),
	}

	if len(events) > 0 {
		stats.LastEvent = &events[len(events)-1]
		stats.LastEventTime = stats.LastEvent.Time

		// 타입별, 액션별 통계 계산
		for _, event := range events {
			stats.EventsByType[event.Type]++
			stats.EventsByAction[event.Action]++
		}
	}

	return stats, nil
}

// IsRunning 모니터링 실행 상태 확인
func (aem *AgentEventMonitor) IsRunning() bool {
	aem.mu.RLock()
	defer aem.mu.RUnlock()
	return aem.running
}

// GetEventChannelSize 이벤트 채널 크기 조회
func (aem *AgentEventMonitor) GetEventChannelSize() int {
	return len(aem.eventChan)
}

// FlushEvents 대기 중인 모든 이벤트 처리
func (aem *AgentEventMonitor) FlushEvents() {
	for len(aem.eventChan) > 0 {
		time.Sleep(time.Millisecond * 10)
	}
}

// 사전 정의된 이벤트 핸들러들

// ContainerLifecycleHandler 컨테이너 생명주기 이벤트 핸들러
func ContainerLifecycleHandler(onStart, onStop, onDie func(agentID, containerID string)) EventHandler {
	return func(event AgentDockerEvent) {
		if event.Type != "container" {
			return
		}

		switch event.Action {
		case "start":
			if onStart != nil {
				onStart(event.AgentID, event.ContainerID)
			}
		case "stop":
			if onStop != nil {
				onStop(event.AgentID, event.ContainerID)
			}
		case "die":
			if onDie != nil {
				onDie(event.AgentID, event.ContainerID)
			}
		}
	}
}

// NetworkEventHandler 네트워크 이벤트 핸들러
func NetworkEventHandler(onCreate, onConnect, onDisconnect, onDestroy func(agentID, networkID string)) EventHandler {
	return func(event AgentDockerEvent) {
		if event.Type != "network" {
			return
		}

		switch event.Action {
		case "create":
			if onCreate != nil {
				onCreate(event.AgentID, event.NetworkID)
			}
		case "connect":
			if onConnect != nil {
				onConnect(event.AgentID, event.NetworkID)
			}
		case "disconnect":
			if onDisconnect != nil {
				onDisconnect(event.AgentID, event.NetworkID)
			}
		case "destroy":
			if onDestroy != nil {
				onDestroy(event.AgentID, event.NetworkID)
			}
		}
	}
}

// ErrorEventHandler 에러 이벤트 핸들러
func ErrorEventHandler(onError func(agentID string, errorMsg string)) EventHandler {
	return func(event AgentDockerEvent) {
		// 에러를 나타내는 액션들
		errorActions := map[string]bool{
			"die":    true,
			"kill":   true,
			"oom":    true,
			"destroy": true,
		}

		if errorActions[event.Action] && onError != nil {
			errorMsg := fmt.Sprintf("Container %s action: %s", event.ContainerID, event.Action)
			if exitCode, exists := event.Attributes["exitCode"]; exists && exitCode != "0" {
				errorMsg += fmt.Sprintf(" (exit code: %s)", exitCode)
			}
			onError(event.AgentID, errorMsg)
		}
	}
}