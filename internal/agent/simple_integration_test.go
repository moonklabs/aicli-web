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

func TestSimpleIntegration(t *testing.T) {
	// Mock dependencies 설정
	mockStorage := memory.New()
	mockDockerAdapter := &MockDockerAdapter{}
	mockWorktreeManager := &MockWorktreeManager{}

	// Factory로 통합 시스템 생성
	factory := NewAgentServiceFactory(mockStorage, mockDockerAdapter, mockWorktreeManager)
	system := factory.CreateIntegratedAgentSystem()

	ctx := context.Background()

	t.Run("단순 이벤트 발행 테스트", func(t *testing.T) {
		fmt.Printf("=== 단순 이벤트 발행 테스트 시작 ===\n")
		
		// 이벤트 구독 설정
		fmt.Printf("1. 전역 이벤트 구독 시작\n")
		eventChan, err := system.EventBus.SubscribeGlobal(ctx)
		require.NoError(t, err)
		
		// 에이전트 생성 요청
		agentReq := CreateAgentRequest{
			ProjectID:   uuid.New().String(),
			Name:        "simple-test-agent",
			Type:        models.AgentTypeClaude,
			Description: "단순 테스트용 에이전트",
			Config: models.AgentConfig{
				Environment: map[string]string{"TEST": "true"},
				WorkingDir:  "/workspace",
				MemoryLimit: "512m",
				CPULimit:    "0.5",
			},
		}

		fmt.Printf("2. 에이전트 생성 요청: %+v\n", agentReq.Name)

		// 에이전트 생성
		agent, err := system.AgentService.CreateAgent(ctx, agentReq)
		require.NoError(t, err)
		fmt.Printf("3. 에이전트 생성 완료: %s\n", agent.ID)

		// 생성 이벤트 확인 (더 긴 타임아웃)
		fmt.Printf("4. 생성 이벤트 수신 대기...\n")
		select {
		case event := <-eventChan:
			fmt.Printf("5. 이벤트 수신 성공: Type=%s, AgentID=%s, Message=%s\n", 
				event.Type, event.AgentID, event.Message)
			require.Equal(t, AgentEventCreated, event.Type)
			require.Equal(t, agent.ID, event.AgentID)
		case <-time.After(3 * time.Second):
			fmt.Printf("5. 이벤트 수신 타임아웃!\n")
			
			// 디버그 정보
			counts := system.EventBus.(*basicEventBus).GetSubscriberCount()
			fmt.Printf("구독자 수: %+v\n", counts)
			
			// EventPublisher가 nil인지 확인
			fmt.Printf("EventPublisher nil? %v\n", system.EventPublisher == nil)
			
			t.Fatal("생성 이벤트를 받지 못했습니다")
		}
		
		fmt.Printf("=== 단순 이벤트 발행 테스트 완료 ===\n")
	})

	t.Run("직접 EventPublisher 테스트", func(t *testing.T) {
		fmt.Printf("=== 직접 EventPublisher 테스트 시작 ===\n")
		
		// 이벤트 구독 설정
		eventChan, err := system.EventBus.SubscribeGlobal(ctx)
		require.NoError(t, err)
		
		// 테스트 에이전트
		testAgent := &models.Agent{
			ID:        uuid.New().String(),
			ProjectID: uuid.New().String(),
			Name:      "direct-test-agent",
			Type:      models.AgentTypeClaude,
			Status:    models.AgentStatusCreated,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		fmt.Printf("1. 직접 이벤트 발행 시작: %s\n", testAgent.ID)

		// 직접 이벤트 발행
		err = system.EventPublisher.PublishAgentCreated(ctx, testAgent)
		require.NoError(t, err)
		fmt.Printf("2. 직접 이벤트 발행 완료\n")

		// 이벤트 수신 확인
		fmt.Printf("3. 직접 이벤트 수신 대기...\n")
		select {
		case event := <-eventChan:
			fmt.Printf("4. 직접 이벤트 수신 성공: Type=%s, AgentID=%s\n", 
				event.Type, event.AgentID)
			require.Equal(t, AgentEventCreated, event.Type)
			require.Equal(t, testAgent.ID, event.AgentID)
		case <-time.After(2 * time.Second):
			fmt.Printf("4. 직접 이벤트 수신 타임아웃!\n")
			t.Fatal("직접 이벤트를 받지 못했습니다")
		}
		
		fmt.Printf("=== 직접 EventPublisher 테스트 완료 ===\n")
	})
}