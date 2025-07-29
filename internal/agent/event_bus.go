package agent

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// basicEventBus 기본 이벤트 버스 구현 (메모리 기반)
type basicEventBus struct {
	// 구독자 관리
	subscribers map[string][]chan AgentEvent
	subsMutex   sync.RWMutex

	// 전역 구독자 (모든 이벤트 수신)
	globalSubscribers []chan AgentEvent
	globalMutex       sync.RWMutex

	// 이벤트 히스토리 (선택사항)
	eventHistory []AgentEvent
	historyMutex sync.RWMutex

	// 설정
	config EventBusConfig
}

// EventBusConfig 이벤트 버스 설정
type EventBusConfig struct {
	BufferSize        int           // 채널 버퍼 크기
	MaxHistorySize    int           // 최대 히스토리 보관 개수
	HistoryRetention  time.Duration // 히스토리 보관 기간
	PublishTimeout    time.Duration // 발행 타임아웃
	EnableHistory     bool          // 히스토리 보관 여부
}

// NewBasicEventBus 새 기본 이벤트 버스 생성
func NewBasicEventBus(config EventBusConfig) EventBus {
	if config.BufferSize == 0 {
		config.BufferSize = 100
	}
	if config.MaxHistorySize == 0 {
		config.MaxHistorySize = 1000
	}
	if config.HistoryRetention == 0 {
		config.HistoryRetention = 24 * time.Hour
	}
	if config.PublishTimeout == 0 {
		config.PublishTimeout = 5 * time.Second
	}

	return &basicEventBus{
		subscribers:       make(map[string][]chan AgentEvent),
		globalSubscribers: make([]chan AgentEvent, 0),
		eventHistory:      make([]AgentEvent, 0),
		config:            config,
	}
}

// Publish 이벤트 발행
func (b *basicEventBus) Publish(ctx context.Context, event AgentEvent) error {
	// 이벤트 타임스탬프 설정 (없는 경우)
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// 히스토리에 추가 (설정된 경우)
	if b.config.EnableHistory {
		b.addToHistory(event)
	}

	// 타임아웃 컨텍스트 생성
	publishCtx, cancel := context.WithTimeout(ctx, b.config.PublishTimeout)
	defer cancel()

	// 에이전트별 구독자에게 발송
	b.publishToAgentSubscribers(publishCtx, event)

	// 전역 구독자에게 발송
	b.publishToGlobalSubscribers(publishCtx, event)

	return nil
}

// Subscribe 에이전트별 이벤트 구독
func (b *basicEventBus) Subscribe(ctx context.Context, agentID string) (<-chan AgentEvent, error) {
	b.subsMutex.Lock()
	defer b.subsMutex.Unlock()

	// 새 채널 생성
	eventChan := make(chan AgentEvent, b.config.BufferSize)

	// 구독자 목록에 추가
	b.subscribers[agentID] = append(b.subscribers[agentID], eventChan)

	return eventChan, nil
}

// Unsubscribe 에이전트별 이벤트 구독 해제
func (b *basicEventBus) Unsubscribe(ctx context.Context, agentID string) error {
	b.subsMutex.Lock()
	defer b.subsMutex.Unlock()

	channels := b.subscribers[agentID]
	if len(channels) == 0 {
		return fmt.Errorf("no subscriptions found for agent %s", agentID)
	}

	// 모든 채널 닫기
	for _, ch := range channels {
		close(ch)
	}

	// 구독자 목록에서 제거
	delete(b.subscribers, agentID)

	return nil
}

// SubscribeGlobal 모든 이벤트 구독 (전역)
func (b *basicEventBus) SubscribeGlobal(ctx context.Context) (<-chan AgentEvent, error) {
	b.globalMutex.Lock()
	defer b.globalMutex.Unlock()

	// 새 채널 생성
	eventChan := make(chan AgentEvent, b.config.BufferSize)

	// 전역 구독자 목록에 추가
	b.globalSubscribers = append(b.globalSubscribers, eventChan)

	return eventChan, nil
}

// UnsubscribeGlobal 전역 구독 해제
func (b *basicEventBus) UnsubscribeGlobal(ctx context.Context, eventChan <-chan AgentEvent) error {
	b.globalMutex.Lock()
	defer b.globalMutex.Unlock()

	// 채널을 찾아서 제거
	for i, ch := range b.globalSubscribers {
		if ch == eventChan {
			// 채널 닫기
			close(ch)
			// 슬라이스에서 제거
			b.globalSubscribers = append(b.globalSubscribers[:i], b.globalSubscribers[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("subscription not found")
}

// GetEventHistory 이벤트 히스토리 조회
func (b *basicEventBus) GetEventHistory(agentID string, since time.Time) ([]AgentEvent, error) {
	if !b.config.EnableHistory {
		return nil, fmt.Errorf("event history is disabled")
	}

	b.historyMutex.RLock()
	defer b.historyMutex.RUnlock()

	var filtered []AgentEvent
	for _, event := range b.eventHistory {
		if (agentID == "" || event.AgentID == agentID) && event.Timestamp.After(since) {
			filtered = append(filtered, event)
		}
	}

	return filtered, nil
}

// publishToAgentSubscribers 에이전트별 구독자에게 이벤트 발송
func (b *basicEventBus) publishToAgentSubscribers(ctx context.Context, event AgentEvent) {
	b.subsMutex.RLock()
	channels := b.subscribers[event.AgentID]
	b.subsMutex.RUnlock()

	if len(channels) == 0 {
		return
	}

	// 각 구독자에게 이벤트 발송 (비동기)
	for _, ch := range channels {
		go b.sendEvent(ctx, ch, event)
	}
}

// publishToGlobalSubscribers 전역 구독자에게 이벤트 발송
func (b *basicEventBus) publishToGlobalSubscribers(ctx context.Context, event AgentEvent) {
	b.globalMutex.RLock()
	channels := make([]chan AgentEvent, len(b.globalSubscribers))
	copy(channels, b.globalSubscribers)
	b.globalMutex.RUnlock()

	if len(channels) == 0 {
		return
	}

	// 각 전역 구독자에게 이벤트 발송 (비동기)
	for _, ch := range channels {
		go b.sendEvent(ctx, ch, event)
	}
}

// sendEvent 채널로 이벤트 발송 (논블로킹)
func (b *basicEventBus) sendEvent(ctx context.Context, ch chan AgentEvent, event AgentEvent) {
	select {
	case ch <- event:
		// 성공적으로 발송
	case <-ctx.Done():
		// 타임아웃 또는 취소
	default:
		// 채널이 가득 참 - 이벤트 드롭
		// TODO: 로깅 또는 메트릭 수집
	}
}

// addToHistory 이벤트를 히스토리에 추가
func (b *basicEventBus) addToHistory(event AgentEvent) {
	b.historyMutex.Lock()
	defer b.historyMutex.Unlock()

	// 히스토리에 추가
	b.eventHistory = append(b.eventHistory, event)

	// 보관 정책 적용
	b.eventHistory = b.applyHistoryRetentionPolicy(b.eventHistory)
}

// applyHistoryRetentionPolicy 히스토리 보관 정책 적용
func (b *basicEventBus) applyHistoryRetentionPolicy(history []AgentEvent) []AgentEvent {
	if len(history) == 0 {
		return history
	}

	// 시간 기반 필터링
	cutoff := time.Now().Add(-b.config.HistoryRetention)
	var validEvents []AgentEvent

	for _, event := range history {
		if event.Timestamp.After(cutoff) {
			validEvents = append(validEvents, event)
		}
	}

	// 개수 기반 제한
	if len(validEvents) > b.config.MaxHistorySize {
		startIndex := len(validEvents) - b.config.MaxHistorySize
		validEvents = validEvents[startIndex:]
	}

	return validEvents
}

// CleanupSubscribers 끊어진 구독자 정리
func (b *basicEventBus) CleanupSubscribers() {
	b.subsMutex.Lock()
	defer b.subsMutex.Unlock()

	// 에이전트별 구독자 정리
	for agentID, channels := range b.subscribers {
		var activeChannels []chan AgentEvent
		for _, ch := range channels {
			// 채널이 닫혔는지 확인하는 방법은 제한적이므로
			// 실제 구현에서는 다른 방법을 사용해야 할 수 있음
			activeChannels = append(activeChannels, ch)
		}
		
		if len(activeChannels) == 0 {
			delete(b.subscribers, agentID)
		} else {
			b.subscribers[agentID] = activeChannels
		}
	}

	b.globalMutex.Lock()
	defer b.globalMutex.Unlock()

	// 전역 구독자 정리도 마찬가지
	// 실제 구현에서는 더 정교한 방법이 필요
}

// GetSubscriberCount 구독자 수 반환
func (b *basicEventBus) GetSubscriberCount() map[string]int {
	b.subsMutex.RLock()
	defer b.subsMutex.RUnlock()

	counts := make(map[string]int)
	for agentID, channels := range b.subscribers {
		counts[agentID] = len(channels)
	}

	b.globalMutex.RLock()
	counts["global"] = len(b.globalSubscribers)
	b.globalMutex.RUnlock()

	return counts
}

// Close 이벤트 버스 종료
func (b *basicEventBus) Close() error {
	// 모든 구독자 채널 닫기
	b.subsMutex.Lock()
	for agentID := range b.subscribers {
		channels := b.subscribers[agentID]
		for _, ch := range channels {
			close(ch)
		}
		delete(b.subscribers, agentID)
	}
	b.subsMutex.Unlock()

	// 전역 구독자 채널 닫기
	b.globalMutex.Lock()
	for _, ch := range b.globalSubscribers {
		close(ch)
	}
	b.globalSubscribers = nil
	b.globalMutex.Unlock()

	return nil
}