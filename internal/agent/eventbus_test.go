package agent

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBasicEventBus(t *testing.T) {
	config := EventBusConfig{
		BufferSize:       10,
		MaxHistorySize:   100,
		HistoryRetention: time.Hour,
		PublishTimeout:   time.Second,
		EnableHistory:    true,
	}

	eventBus := NewBasicEventBus(config)
	ctx := context.Background()

	t.Run("기본 이벤트 발행/구독 테스트", func(t *testing.T) {
		// 구독 설정
		eventChan, err := eventBus.SubscribeGlobal(ctx)
		require.NoError(t, err)
		assert.NotNil(t, eventChan)

		// 테스트 이벤트 생성
		testEvent := AgentEvent{
			Type:      AgentEventCreated,
			AgentID:   "test-agent-123",
			Timestamp: time.Now(),
			Message:   "테스트 이벤트",
		}

		// 이벤트 발행
		err = eventBus.Publish(ctx, testEvent)
		require.NoError(t, err)

		// 이벤트 수신 확인
		select {
		case receivedEvent := <-eventChan:
			assert.Equal(t, testEvent.Type, receivedEvent.Type)
			assert.Equal(t, testEvent.AgentID, receivedEvent.AgentID)
			assert.Equal(t, testEvent.Message, receivedEvent.Message)
		case <-time.After(time.Second):
			t.Fatal("이벤트를 받지 못했습니다")
		}
	})

	t.Run("에이전트별 구독 테스트", func(t *testing.T) {
		agentID := "test-agent-456"

		// 에이전트별 구독
		eventChan, err := eventBus.Subscribe(ctx, agentID)
		require.NoError(t, err)

		// 다른 에이전트의 이벤트 발행 (수신되지 않아야 함)
		otherEvent := AgentEvent{
			Type:    AgentEventStarted,
			AgentID: "other-agent",
			Message: "다른 에이전트 이벤트",
		}
		err = eventBus.Publish(ctx, otherEvent)
		require.NoError(t, err)

		// 대상 에이전트의 이벤트 발행
		targetEvent := AgentEvent{
			Type:    AgentEventStarted,
			AgentID: agentID,
			Message: "대상 에이전트 이벤트",
		}
		err = eventBus.Publish(ctx, targetEvent)
		require.NoError(t, err)

		// 대상 에이전트의 이벤트만 수신되어야 함
		select {
		case receivedEvent := <-eventChan:
			assert.Equal(t, targetEvent.AgentID, receivedEvent.AgentID)
			assert.Equal(t, targetEvent.Message, receivedEvent.Message)
		case <-time.After(time.Second):
			t.Fatal("대상 에이전트 이벤트를 받지 못했습니다")
		}

		// 다른 이벤트는 수신되지 않아야 함
		select {
		case unexpectedEvent := <-eventChan:
			t.Fatalf("예상치 못한 이벤트를 받았습니다: %+v", unexpectedEvent)
		case <-time.After(100 * time.Millisecond):
			// 정상 - 다른 이벤트는 수신되지 않음
		}
	})
}

func TestEventPublisherBasic(t *testing.T) {
	// EventBus 생성
	eventBus := NewBasicEventBus(DefaultEventBusConfig())

	ctx := context.Background()

	t.Run("단순 이벤트 발행 테스트", func(t *testing.T) {
		// 구독 설정
		eventChan, err := eventBus.SubscribeGlobal(ctx)
		require.NoError(t, err)

		// 직접 이벤트 발행
		testEvent := AgentEvent{
			Type:      AgentEventCreated,
			AgentID:   "direct-test",
			Timestamp: time.Now(),
			Message:   "직접 발행 테스트",
		}

		err = eventBus.Publish(ctx, testEvent)
		require.NoError(t, err)

		// 이벤트 수신 확인
		select {
		case receivedEvent := <-eventChan:
			assert.Equal(t, testEvent.Type, receivedEvent.Type)
			assert.Equal(t, testEvent.AgentID, receivedEvent.AgentID)
			t.Logf("직접 발행 테스트 성공: %+v", receivedEvent)
		case <-time.After(time.Second):
			t.Fatal("직접 발행 이벤트를 받지 못했습니다")
		}
	})
}
