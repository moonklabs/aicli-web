package agent

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/storage/memory"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestDirectIntegration(t *testing.T) {
	t.Run("Factory 없이 직접 컴포넌트 생성", func(t *testing.T) {
		fmt.Printf("=== Factory 없이 직접 컴포넌트 생성 테스트 시작 ===\n")
		
		// Mock dependencies
		mockStorage := memory.New()
		mockDockerAdapter := &MockDockerAdapter{}
		mockWorktreeManager := &MockWorktreeManager{}

		// 1. EventBus 직접 생성
		fmt.Printf("1. EventBus 직접 생성\n")
		eventBus := NewBasicEventBus(EventBusConfig{
			BufferSize:       100,
			MaxHistorySize:   1000,
			HistoryRetention: 24 * time.Hour,
			PublishTimeout:   5 * time.Second,
			EnableHistory:    true,
		})
		
		// 2. EventPublisher 직접 생성
		fmt.Printf("2. EventPublisher 직접 생성\n")
		eventPublisher := NewBasicEventPublisher(eventBus)
		
		// 3. AgentService 직접 생성 (MonitoringService 없이)
		fmt.Printf("3. AgentService 직접 생성\n")
		agentService := NewAgentService(
			mockStorage,
			mockDockerAdapter,
			nil, // monitoring 없음
			eventPublisher,
			mockWorktreeManager,
		)

		ctx := context.Background()

		// 4. 이벤트 구독
		fmt.Printf("4. 이벤트 구독 설정\n")
		eventChan, err := eventBus.SubscribeGlobal(ctx)
		require.NoError(t, err)
		
		// 구독자 수 확인
		counts := eventBus.(*basicEventBus).GetSubscriberCount()
		fmt.Printf("5. 구독자 수: %+v\n", counts)

		// 5. 에이전트 생성
		fmt.Printf("6. 에이전트 생성 시작\n")
		agentReq := CreateAgentRequest{
			ProjectID:   uuid.New().String(),
			Name:        "direct-test-agent",
			Type:        models.AgentTypeClaude,
			Description: "직접 생성 테스트",
			Config: models.AgentConfig{
				Environment: map[string]string{"TEST": "true"},
				WorkingDir:  "/workspace",
				MemoryLimit: "512m",
				CPULimit:    "0.5",
			},
		}

		agent, err := agentService.CreateAgent(ctx, agentReq)
		require.NoError(t, err)
		fmt.Printf("7. 에이전트 생성 완료: %s\n", agent.ID)

		// 6. 생성 이벤트 확인
		fmt.Printf("8. 생성 이벤트 수신 대기...\n")
		select {
		case event := <-eventChan:
			fmt.Printf("9. 이벤트 수신 성공: Type=%s, AgentID=%s\n", 
				event.Type, event.AgentID)
			require.Equal(t, AgentEventCreated, event.Type)
			require.Equal(t, agent.ID, event.AgentID)
		case <-time.After(3 * time.Second):
			fmt.Printf("9. 이벤트 수신 타임아웃!\n")
			
			// 추가 디버그
			counts = eventBus.(*basicEventBus).GetSubscriberCount()
			fmt.Printf("타임아웃 시 구독자 수: %+v\n", counts)
			
			t.Fatal("직접 생성에서도 이벤트를 받지 못했습니다")
		}
		
		fmt.Printf("=== Factory 없이 직접 컴포넌트 생성 테스트 완료 ===\n")
	})
	
	t.Run("EventPublisher만 단독 테스트", func(t *testing.T) {
		fmt.Printf("=== EventPublisher 단독 테스트 시작 ===\n")
		
		// EventBus와 EventPublisher만 생성
		eventBus := NewBasicEventBus(DefaultEventBusConfig())
		eventPublisher := NewBasicEventPublisher(eventBus)
		
		ctx := context.Background()
		
		// 구독 설정
		eventChan, err := eventBus.SubscribeGlobal(ctx)
		require.NoError(t, err)
		
		// 테스트 에이전트
		testAgent := &models.Agent{
			ID:        uuid.New().String(),
			ProjectID: uuid.New().String(),
			Name:      "publisher-only-test",
			Type:      models.AgentTypeClaude,
			Status:    models.AgentStatusCreated,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		fmt.Printf("1. EventPublisher로 이벤트 발행\n")
		err = eventPublisher.PublishAgentCreated(ctx, testAgent)
		require.NoError(t, err)

		// 이벤트 수신 확인
		fmt.Printf("2. 이벤트 수신 대기...\n")
		select {
		case event := <-eventChan:
			fmt.Printf("3. EventPublisher 단독 테스트 성공: %+v\n", event)
			require.Equal(t, AgentEventCreated, event.Type)
			require.Equal(t, testAgent.ID, event.AgentID)
		case <-time.After(2 * time.Second):
			fmt.Printf("3. EventPublisher 단독 테스트 실패\n")
			t.Fatal("EventPublisher 단독에서도 이벤트를 받지 못했습니다")
		}
		
		fmt.Printf("=== EventPublisher 단독 테스트 완료 ===\n")
	})
}