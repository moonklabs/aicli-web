package agent

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventBusFix(t *testing.T) {
	t.Run("전역 구독 문제 디버그", func(t *testing.T) {
		// 새로운 EventBus 생성
		config := EventBusConfig{
			BufferSize:       10,
			MaxHistorySize:   100,
			HistoryRetention: time.Hour,
			PublishTimeout:   5 * time.Second, // 더 긴 타임아웃
			EnableHistory:    true,
		}

		eventBus := NewBasicEventBus(config)
		ctx := context.Background()

		fmt.Printf("=== 전역 구독 문제 디버그 시작 ===\n")

		// 구독 전 상태 확인
		basicBus := eventBus.(*basicEventBus)
		fmt.Printf("1. 구독 전 전역 구독자 수: %d\n", len(basicBus.globalSubscribers))

		// 전역 구독 설정
		fmt.Printf("2. 전역 구독 설정 중...\n")
		eventChan, err := eventBus.SubscribeGlobal(ctx)
		require.NoError(t, err)
		assert.NotNil(t, eventChan)

		// 구독 후 상태 확인
		fmt.Printf("3. 구독 후 전역 구독자 수: %d\n", len(basicBus.globalSubscribers))
		counts := basicBus.GetSubscriberCount()
		fmt.Printf("4. 구독자 수 상세: %+v\n", counts)

		// 테스트 이벤트 생성
		testEvent := AgentEvent{
			Type:      AgentEventCreated,
			AgentID:   "fix-test-agent",
			Timestamp: time.Now(),
			Data:      nil,
			Message:   "수정 테스트 이벤트",
		}

		fmt.Printf("5. 이벤트 발행 시작: %+v\n", testEvent)

		// 이벤트 발행
		err = eventBus.Publish(ctx, testEvent)
		require.NoError(t, err)
		fmt.Printf("6. 이벤트 발행 완료\n")

		// 이벤트 수신 확인
		fmt.Printf("7. 이벤트 수신 대기 (최대 3초)...\n")

		timeout := time.After(3 * time.Second)
		select {
		case receivedEvent := <-eventChan:
			fmt.Printf("8. 이벤트 수신 성공: %+v\n", receivedEvent)
			assert.Equal(t, testEvent.Type, receivedEvent.Type)
			assert.Equal(t, testEvent.AgentID, receivedEvent.AgentID)
			assert.Equal(t, testEvent.Message, receivedEvent.Message)
		case <-timeout:
			fmt.Printf("8. 이벤트 수신 타임아웃!\n")

			// 추가 디버그 정보
			fmt.Printf("디버그: 채널 상태 확인\n")
			fmt.Printf("- 채널 nil? %v\n", eventChan == nil)

			t.Fatal("EventBus에서 전역 이벤트를 받지 못했습니다")
		}

		fmt.Printf("=== 전역 구독 문제 디버그 완료 ===\n")
	})

	t.Run("sendEvent 함수 직접 테스트", func(t *testing.T) {
		eventBus := NewBasicEventBus(DefaultEventBusConfig()).(*basicEventBus)
		ctx := context.Background()

		// 테스트 채널 생성
		testChan := make(chan AgentEvent, 10)

		// 테스트 이벤트 생성
		testEvent := AgentEvent{
			Type:    AgentEventCreated,
			AgentID: "direct-send-test",
			Message: "직접 전송 테스트",
		}

		fmt.Printf("=== sendEvent 직접 테스트 시작 ===\n")
		fmt.Printf("1. sendEvent 함수 직접 호출\n")

		// sendEvent 직접 호출
		eventBus.sendEvent(ctx, testChan, testEvent)

		// 수신 확인
		select {
		case receivedEvent := <-testChan:
			fmt.Printf("2. sendEvent 직접 수신 성공: %+v\n", receivedEvent)
			assert.Equal(t, testEvent.Type, receivedEvent.Type)
			assert.Equal(t, testEvent.AgentID, receivedEvent.AgentID)
		case <-time.After(time.Second):
			fmt.Printf("2. sendEvent 직접 수신 실패\n")
			t.Fatal("sendEvent 함수에 문제가 있음")
		}

		fmt.Printf("=== sendEvent 직접 테스트 완료 ===\n")
	})
}
