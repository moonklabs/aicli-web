package agent

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventBusDebug(t *testing.T) {
	config := EventBusConfig{
		BufferSize:       10,
		MaxHistorySize:   100,
		HistoryRetention: time.Hour,
		PublishTimeout:   time.Second,
		EnableHistory:    true,
	}

	eventBus := NewBasicEventBus(config)
	ctx := context.Background()
	agentID := "debug-agent-123"

	t.Run("에이전트별 구독 디버그", func(t *testing.T) {
		fmt.Printf("=== 에이전트별 구독 디버그 시작 ===\n")

		// 구독 설정
		fmt.Printf("1. 에이전트 %s에 대한 구독 시작\n", agentID)
		eventChan, err := eventBus.Subscribe(ctx, agentID)
		require.NoError(t, err)
		assert.NotNil(t, eventChan)

		// 구독자 수 확인
		counts := eventBus.(*basicEventBus).GetSubscriberCount()
		fmt.Printf("2. 구독자 수 확인: %+v\n", counts)

		// 테스트 이벤트 생성
		testEvent := AgentEvent{
			Type:      AgentEventCreated,
			AgentID:   agentID,
			Timestamp: time.Now(),
			Message:   "디버그 테스트 이벤트",
		}

		fmt.Printf("3. 이벤트 발행 시작: %+v\n", testEvent)

		// 이벤트 발행
		err = eventBus.Publish(ctx, testEvent)
		require.NoError(t, err)
		fmt.Printf("4. 이벤트 발행 완료\n")

		// 이벤트 수신 확인 (더 긴 타임아웃)
		fmt.Printf("5. 이벤트 수신 대기...\n")
		select {
		case receivedEvent := <-eventChan:
			fmt.Printf("6. 이벤트 수신 성공: %+v\n", receivedEvent)
			assert.Equal(t, testEvent.Type, receivedEvent.Type)
			assert.Equal(t, testEvent.AgentID, receivedEvent.AgentID)
			assert.Equal(t, testEvent.Message, receivedEvent.Message)
		case <-time.After(2 * time.Second):
			fmt.Printf("6. 이벤트 수신 타임아웃!\n")

			// 추가 디버그 정보
			counts = eventBus.(*basicEventBus).GetSubscriberCount()
			fmt.Printf("7. 타임아웃 시점 구독자 수: %+v\n", counts)

			t.Fatal("에이전트별 이벤트를 받지 못했습니다")
		}

		fmt.Printf("=== 에이전트별 구독 디버그 완료 ===\n")
	})

	t.Run("전역 구독 비교 테스트", func(t *testing.T) {
		fmt.Printf("=== 전역 구독 비교 테스트 시작 ===\n")

		// 전역 구독 설정
		globalChan, err := eventBus.SubscribeGlobal(ctx)
		require.NoError(t, err)

		// 테스트 이벤트 생성
		testEvent := AgentEvent{
			Type:      AgentEventStarted,
			AgentID:   "global-test-agent",
			Timestamp: time.Now(),
			Message:   "전역 테스트 이벤트",
		}

		fmt.Printf("1. 전역 이벤트 발행: %+v\n", testEvent)

		// 이벤트 발행
		err = eventBus.Publish(ctx, testEvent)
		require.NoError(t, err)

		// 전역 이벤트 수신 확인
		select {
		case receivedEvent := <-globalChan:
			fmt.Printf("2. 전역 이벤트 수신 성공: %+v\n", receivedEvent)
			assert.Equal(t, testEvent.Type, receivedEvent.Type)
		case <-time.After(time.Second):
			fmt.Printf("2. 전역 이벤트 수신 타임아웃!\n")
			t.Fatal("전역 이벤트를 받지 못했습니다")
		}

		fmt.Printf("=== 전역 구독 비교 테스트 완료 ===\n")
	})
}
