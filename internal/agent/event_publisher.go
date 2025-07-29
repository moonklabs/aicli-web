package agent

import (
	"context"
	"time"

	"github.com/aicli/aicli-web/internal/models"
)

// basicEventPublisher EventBus를 사용한 기본 이벤트 발행자 구현
type basicEventPublisher struct {
	eventBus EventBus
}

// NewBasicEventPublisher 새 기본 이벤트 발행자 생성
func NewBasicEventPublisher(eventBus EventBus) EventPublisher {
	return &basicEventPublisher{
		eventBus: eventBus,
	}
}

// PublishAgentCreated 에이전트 생성 이벤트 발행
func (p *basicEventPublisher) PublishAgentCreated(ctx context.Context, agent *models.Agent) error {
	if p.eventBus == nil {
		return nil // EventBus가 없으면 조용히 무시
	}

	event := AgentEvent{
		Type:      AgentEventCreated,
		AgentID:   agent.ID,
		Timestamp: time.Now(),
		Data:      agent,
		Message:   "에이전트가 생성되었습니다",
	}

	return p.eventBus.Publish(ctx, event)
}

// PublishAgentStarted 에이전트 시작 이벤트 발행
func (p *basicEventPublisher) PublishAgentStarted(ctx context.Context, agent *models.Agent) error {
	if p.eventBus == nil {
		return nil // EventBus가 없으면 조용히 무시
	}

	event := AgentEvent{
		Type:      AgentEventStarted,
		AgentID:   agent.ID,
		Timestamp: time.Now(),
		Data:      agent,
		Message:   "에이전트가 시작되었습니다",
	}

	return p.eventBus.Publish(ctx, event)
}

// PublishAgentStopped 에이전트 중지 이벤트 발행
func (p *basicEventPublisher) PublishAgentStopped(ctx context.Context, agent *models.Agent) error {
	if p.eventBus == nil {
		return nil // EventBus가 없으면 조용히 무시
	}

	event := AgentEvent{
		Type:      AgentEventStopped,
		AgentID:   agent.ID,
		Timestamp: time.Now(),
		Data:      agent,
		Message:   "에이전트가 중지되었습니다",
	}

	return p.eventBus.Publish(ctx, event)
}

// PublishAgentError 에이전트 에러 이벤트 발행
func (p *basicEventPublisher) PublishAgentError(ctx context.Context, agent *models.Agent, err error) error {
	if p.eventBus == nil {
		return nil // EventBus가 없으면 조용히 무시
	}

	event := AgentEvent{
		Type:      AgentEventError,
		AgentID:   agent.ID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"agent": agent,
			"error": err.Error(),
		},
		Message: "에이전트에서 오류가 발생했습니다: " + err.Error(),
	}

	return p.eventBus.Publish(ctx, event)
}

// PublishAgentDeleted 에이전트 삭제 이벤트 발행
func (p *basicEventPublisher) PublishAgentDeleted(ctx context.Context, agentID string) error {
	if p.eventBus == nil {
		return nil // EventBus가 없으면 조용히 무시
	}

	event := AgentEvent{
		Type:      AgentEventDeleted,
		AgentID:   agentID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"agent_id": agentID,
		},
		Message: "에이전트가 삭제되었습니다",
	}

	return p.eventBus.Publish(ctx, event)
}

// EventBusPublisher EventBus 인터페이스를 만족하는 발행자
type EventBusPublisher struct {
	eventBus EventBus
}

// NewEventBusPublisher EventBus를 직접 사용하는 발행자 생성
func NewEventBusPublisher(eventBus EventBus) *EventBusPublisher {
	return &EventBusPublisher{
		eventBus: eventBus,
	}
}

// PublishEvent 일반적인 이벤트 발행
func (p *EventBusPublisher) PublishEvent(ctx context.Context, event AgentEvent) error {
	if p.eventBus == nil {
		return nil
	}
	return p.eventBus.Publish(ctx, event)
}

// SubscribeToEvents 이벤트 구독
func (p *EventBusPublisher) SubscribeToEvents(ctx context.Context, agentID string) (<-chan AgentEvent, error) {
	if p.eventBus == nil {
		return nil, nil
	}
	return p.eventBus.Subscribe(ctx, agentID)
}

// SubscribeToAllEvents 모든 이벤트 구독
func (p *EventBusPublisher) SubscribeToAllEvents(ctx context.Context) (<-chan AgentEvent, error) {
	if p.eventBus == nil {
		return nil, nil
	}
	return p.eventBus.SubscribeGlobal(ctx)
}