package agent

import (
	"context"
	"testing"
	"time"

	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/storage/memory"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestIntegratedAgentSystem 통합된 에이전트 시스템 테스트
func TestIntegratedAgentSystem(t *testing.T) {
	// Mock dependencies 설정
	mockStorage := memory.New()
	mockDockerAdapter := &MockDockerAdapter{}
	mockWorktreeManager := &MockWorktreeManager{}

	// Docker adapter 설정
	mockDockerAdapter.On("CreateContainer", mock.Anything, mock.Anything).Return(&ContainerInfo{
		ID:      "container-123",
		Name:    "test-container",
		Status:  "created",
		Created: time.Now(),
	}, nil)
	mockDockerAdapter.On("StartContainer", mock.Anything, "container-123").Return(nil)
	mockDockerAdapter.On("StopContainer", mock.Anything, "container-123").Return(nil)
	mockDockerAdapter.On("RemoveContainer", mock.Anything, "container-123").Return(nil)
	mockDockerAdapter.On("GetContainerHealth", mock.Anything, "container-123").Return(HealthStatus{
		Status:    "healthy",
		LastCheck: time.Now(),
	}, nil)
	mockDockerAdapter.On("GetContainerMetrics", mock.Anything, "container-123").Return(ContainerMetrics{
		ContainerID: "container-123",
		Timestamp:   time.Now(),
		CPU:         CPUMetrics{UsagePercent: 10.5},
		Memory:      MemoryMetrics{UsageBytes: 1024 * 1024, LimitBytes: 10 * 1024 * 1024, UsagePercent: 10.0},
	}, nil)

	// Factory로 통합 시스템 생성
	factory := NewAgentServiceFactory(mockStorage, mockDockerAdapter, mockWorktreeManager)
	system := factory.CreateIntegratedAgentSystem()

	t.Run("이벤트 시스템 통합 테스트", func(t *testing.T) {
		ctx := context.Background()

		// 이벤트 구독 설정
		eventChan, err := system.EventBus.SubscribeGlobal(ctx)
		require.NoError(t, err)

		// 에이전트 생성 요청
		agentReq := CreateAgentRequest{
			ProjectID:   uuid.New().String(),
			Name:        "test-agent",
			Type:        models.AgentTypeClaude,
			Description: "통합 테스트용 에이전트",
			Config: models.AgentConfig{
				Environment: map[string]string{"TEST": "true"},
				WorkingDir:  "/workspace",
				MemoryLimit: "512m",
				CPULimit:    "0.5",
			},
		}

		// 에이전트 생성
		agent, err := system.AgentService.CreateAgent(ctx, agentReq)
		require.NoError(t, err)
		assert.NotEmpty(t, agent.ID)

		// 생성 이벤트 확인
		select {
		case event := <-eventChan:
			assert.Equal(t, AgentEventCreated, event.Type)
			assert.Equal(t, agent.ID, event.AgentID)
			assert.NotZero(t, event.Timestamp)
		case <-time.After(2 * time.Second):
			t.Fatal("생성 이벤트를 받지 못했습니다")
		}

		// 에이전트 시작
		err = system.AgentService.StartAgent(ctx, agent.ID)
		require.NoError(t, err)

		// 시작 이벤트 확인
		select {
		case event := <-eventChan:
			assert.Equal(t, AgentEventStarted, event.Type)
			assert.Equal(t, agent.ID, event.AgentID)
		case <-time.After(5 * time.Second):
			t.Fatal("시작 이벤트를 받지 못했습니다")
		}

		// 메트릭 수집 테스트
		time.Sleep(100 * time.Millisecond) // 메트릭 수집을 위한 잠시 대기
		metrics, err := system.AgentService.GetAgentMetrics(ctx, agent.ID)
		require.NoError(t, err)
		assert.Equal(t, agent.ID, metrics.AgentID)
		assert.NotZero(t, metrics.Timestamp)

		// 헬스 체크 테스트
		health, err := system.AgentService.GetHealthStatus(ctx, agent.ID)
		require.NoError(t, err)
		assert.Equal(t, "healthy", health.Status)

		// 에이전트 중지
		err = system.AgentService.StopAgent(ctx, agent.ID)
		require.NoError(t, err)

		// 중지 이벤트 확인
		select {
		case event := <-eventChan:
			assert.Equal(t, AgentEventStopped, event.Type)
			assert.Equal(t, agent.ID, event.AgentID)
		case <-time.After(2 * time.Second):
			t.Fatal("중지 이벤트를 받지 못했습니다")
		}

		// 에이전트 삭제
		err = system.AgentService.DeleteAgent(ctx, agent.ID)
		require.NoError(t, err)

		// 삭제 이벤트 확인
		select {
		case event := <-eventChan:
			assert.Equal(t, AgentEventDeleted, event.Type)
			assert.Equal(t, agent.ID, event.AgentID)
		case <-time.After(2 * time.Second):
			t.Fatal("삭제 이벤트를 받지 못했습니다")
		}
	})

	t.Run("메트릭 수집 및 저장 테스트", func(t *testing.T) {
		ctx := context.Background()

		// 테스트용 에이전트 생성
		agentReq := CreateAgentRequest{
			ProjectID: uuid.New().String(),
			Name:      "metrics-test-agent",
			Type:      models.AgentTypeClaude,
			Config:    models.AgentConfig{},
		}

		agent, err := system.AgentService.CreateAgent(ctx, agentReq)
		require.NoError(t, err)

		// 메트릭 수집
		metrics, err := system.MetricsCollector.CollectAgentMetrics(ctx, agent)
		require.NoError(t, err)
		assert.Equal(t, agent.ID, metrics.AgentID)

		// 메트릭 저장
		err = system.MetricsCollector.StoreMetrics(ctx, metrics)
		require.NoError(t, err)

		// 메트릭 히스토리 조회
		history, err := system.MetricsCollector.GetMetricsHistory(ctx, agent.ID, time.Hour)
		require.NoError(t, err)
		assert.Len(t, history, 1)
		assert.Equal(t, agent.ID, history[0].AgentID)
	})

	t.Run("이벤트 히스토리 테스트", func(t *testing.T) {
		ctx := context.Background()

		// 테스트용 에이전트 생성
		agentReq := CreateAgentRequest{
			ProjectID: uuid.New().String(),
			Name:      "history-test-agent",
			Type:      models.AgentTypeClaude,
			Config:    models.AgentConfig{},
		}

		agent, err := system.AgentService.CreateAgent(ctx, agentReq)
		require.NoError(t, err)

		// 잠시 대기 후 이벤트 히스토리 조회
		time.Sleep(100 * time.Millisecond)
		
		history, err := system.EventBus.GetEventHistory(agent.ID, time.Now().Add(-time.Minute))
		require.NoError(t, err)
		assert.NotEmpty(t, history)

		// 생성 이벤트가 포함되어 있는지 확인
		found := false
		for _, event := range history {
			if event.Type == AgentEventCreated && event.AgentID == agent.ID {
				found = true
				break
			}
		}
		assert.True(t, found, "생성 이벤트가 히스토리에 포함되어 있어야 합니다")
	})

	t.Run("복구 시스템 통합 테스트", func(t *testing.T) {
		ctx := context.Background()

		// 컨테이너 시작 실패 시나리오 설정
		mockDockerAdapter.On("StartContainer", mock.Anything, "failing-container").Return(&ServiceError{
			Code:    ErrCodeContainerError,
			Message: "컨테이너 시작 실패",
		}).Once()

		// 복구 후 성공 시나리오
		mockDockerAdapter.On("GetContainerStatus", mock.Anything, "failing-container").Return(ContainerStatus{
			Status: "stopped",
		}, nil)
		mockDockerAdapter.On("StartContainer", mock.Anything, "failing-container").Return(nil).Once()

		// 실패하는 컨테이너로 에이전트 생성
		mockDockerAdapter.On("CreateContainer", mock.Anything, mock.Anything).Return(&ContainerInfo{
			ID:      "failing-container",
			Name:    "failing-test-container",
			Status:  "created",
			Created: time.Now(),
		}, nil).Once()

		agentReq := CreateAgentRequest{
			ProjectID: uuid.New().String(),
			Name:      "recovery-test-agent",
			Type:      models.AgentTypeClaude,
			Config:    models.AgentConfig{},
		}

		agent, err := system.AgentService.CreateAgent(ctx, agentReq)
		require.NoError(t, err)

		// 에이전트 시작 (복구 시스템이 작동해야 함)
		err = system.AgentService.StartAgent(ctx, agent.ID)
		require.NoError(t, err)

		// 복구가 성공했는지 확인 (시간이 필요할 수 있음)
		time.Sleep(500 * time.Millisecond)
		status, err := system.AgentService.GetAgentStatus(ctx, agent.ID)
		require.NoError(t, err)
		
		// 최종적으로 실행 상태가 되어야 함 (복구 완료)
		assert.Contains(t, []models.AgentStatus{
			models.AgentStatusRunning,
			models.AgentStatusStarting,
		}, status.Status)
	})
}

// TestEventPublisherIntegration EventPublisher 통합 테스트
func TestEventPublisherIntegration(t *testing.T) {
	// EventBus 생성
	eventBus := NewBasicEventBus(DefaultEventBusConfig())
	
	// EventPublisher 생성
	publisher := NewBasicEventPublisher(eventBus)

	ctx := context.Background()

	// 이벤트 구독
	eventChan, err := eventBus.SubscribeGlobal(ctx)
	require.NoError(t, err)

	// 테스트 에이전트
	agent := &models.Agent{
		ID:        uuid.New().String(),
		ProjectID: uuid.New().String(),
		Name:      "test-agent",
		Type:      models.AgentTypeClaude,
		Status:    models.AgentStatusCreated,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	t.Run("모든 이벤트 발행 테스트", func(t *testing.T) {
		// 생성 이벤트
		err := publisher.PublishAgentCreated(ctx, agent)
		require.NoError(t, err)

		event := <-eventChan
		assert.Equal(t, AgentEventCreated, event.Type)
		assert.Equal(t, agent.ID, event.AgentID)

		// 시작 이벤트
		err = publisher.PublishAgentStarted(ctx, agent)
		require.NoError(t, err)

		event = <-eventChan
		assert.Equal(t, AgentEventStarted, event.Type)

		// 중지 이벤트
		err = publisher.PublishAgentStopped(ctx, agent)
		require.NoError(t, err)

		event = <-eventChan
		assert.Equal(t, AgentEventStopped, event.Type)

		// 에러 이벤트
		testErr := &ServiceError{Code: ErrCodeInternalError, Message: "테스트 에러"}
		err = publisher.PublishAgentError(ctx, agent, testErr)
		require.NoError(t, err)

		event = <-eventChan
		assert.Equal(t, AgentEventError, event.Type)
		assert.Contains(t, event.Message, "테스트 에러")

		// 삭제 이벤트
		err = publisher.PublishAgentDeleted(ctx, agent.ID)
		require.NoError(t, err)

		event = <-eventChan
		assert.Equal(t, AgentEventDeleted, event.Type)
		assert.Equal(t, agent.ID, event.AgentID)
	})
}

// MockWorktreeManager는 service_test.go에 정의됨