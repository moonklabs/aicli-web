package agent

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aicli/aicli-web/internal/git"
	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/storage"
	"github.com/aicli/aicli-web/internal/storage/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockDockerAdapter Docker 어댑터 Mock
type MockDockerAdapter struct {
	mock.Mock
}

func (m *MockDockerAdapter) CreateContainer(ctx context.Context, config ContainerConfig) (*ContainerInfo, error) {
	args := m.Called(ctx, config)
	return args.Get(0).(*ContainerInfo), args.Error(1)
}

func (m *MockDockerAdapter) StartContainer(ctx context.Context, containerID string) error {
	args := m.Called(ctx, containerID)
	return args.Error(0)
}

func (m *MockDockerAdapter) StopContainer(ctx context.Context, containerID string) error {
	args := m.Called(ctx, containerID)
	return args.Error(0)
}

func (m *MockDockerAdapter) RemoveContainer(ctx context.Context, containerID string) error {
	args := m.Called(ctx, containerID)
	return args.Error(0)
}

func (m *MockDockerAdapter) GetContainerStatus(ctx context.Context, containerID string) (ContainerStatus, error) {
	args := m.Called(ctx, containerID)
	return args.Get(0).(ContainerStatus), args.Error(1)
}

func (m *MockDockerAdapter) GetContainerHealth(ctx context.Context, containerID string) (HealthStatus, error) {
	args := m.Called(ctx, containerID)
	return args.Get(0).(HealthStatus), args.Error(1)
}

func (m *MockDockerAdapter) GetContainerMetrics(ctx context.Context, containerID string) (ContainerMetrics, error) {
	args := m.Called(ctx, containerID)
	return args.Get(0).(ContainerMetrics), args.Error(1)
}

func (m *MockDockerAdapter) GetContainerLogs(ctx context.Context, containerID string, opts LogOptions) (LogStream, error) {
	args := m.Called(ctx, containerID, opts)
	return args.Get(0).(LogStream), args.Error(1)
}

func (m *MockDockerAdapter) ExecuteCommand(ctx context.Context, containerID string, cmd []string) (ExecResult, error) {
	args := m.Called(ctx, containerID, cmd)
	return args.Get(0).(ExecResult), args.Error(1)
}

// MockMonitoringService 모니터링 서비스 Mock
type MockMonitoringService struct {
	mock.Mock
}

func (m *MockMonitoringService) CheckAgentHealth(ctx context.Context, agent *models.Agent) (HealthStatus, error) {
	args := m.Called(ctx, agent)
	return args.Get(0).(HealthStatus), args.Error(1)
}

func (m *MockMonitoringService) StartHealthMonitoring(ctx context.Context, agent *models.Agent) error {
	args := m.Called(ctx, agent)
	return args.Error(0)
}

func (m *MockMonitoringService) StopHealthMonitoring(ctx context.Context, agentID string) error {
	args := m.Called(ctx, agentID)
	return args.Error(0)
}

func (m *MockMonitoringService) CollectAgentMetrics(ctx context.Context, agent *models.Agent) (AgentMetrics, error) {
	args := m.Called(ctx, agent)
	return args.Get(0).(AgentMetrics), args.Error(1)
}

func (m *MockMonitoringService) GetMetricsHistory(ctx context.Context, agentID string, period time.Duration) ([]AgentMetrics, error) {
	args := m.Called(ctx, agentID, period)
	return args.Get(0).([]AgentMetrics), args.Error(1)
}

func (m *MockMonitoringService) PublishAgentEvent(ctx context.Context, event AgentEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockMonitoringService) SubscribeToAgentEvents(ctx context.Context, agentID string) (<-chan AgentEvent, error) {
	args := m.Called(ctx, agentID)
	return args.Get(0).(<-chan AgentEvent), args.Error(1)
}

// MockEventPublisher 이벤트 발행자 Mock
type MockEventPublisher struct {
	mock.Mock
}

func (m *MockEventPublisher) PublishAgentCreated(ctx context.Context, agent *models.Agent) error {
	args := m.Called(ctx, agent)
	return args.Error(0)
}

func (m *MockEventPublisher) PublishAgentStarted(ctx context.Context, agent *models.Agent) error {
	args := m.Called(ctx, agent)
	return args.Error(0)
}

func (m *MockEventPublisher) PublishAgentStopped(ctx context.Context, agent *models.Agent) error {
	args := m.Called(ctx, agent)
	return args.Error(0)
}

func (m *MockEventPublisher) PublishAgentError(ctx context.Context, agent *models.Agent, err error) error {
	args := m.Called(ctx, agent, err)
	return args.Error(0)
}

func (m *MockEventPublisher) PublishAgentDeleted(ctx context.Context, agentID string) error {
	args := m.Called(ctx, agentID)
	return args.Error(0)
}

// MockWorktreeManager Git Worktree 매니저 Mock
type MockWorktreeManager struct {
	mock.Mock
}

func (m *MockWorktreeManager) Clone(ctx context.Context, url, path string, opts git.CloneOptions) (*git.Repository, error) {
	args := m.Called(ctx, url, path, opts)
	return args.Get(0).(*git.Repository), args.Error(1)
}

func (m *MockWorktreeManager) CreateWorktree(ctx context.Context, repo *git.Repository, name string, opts git.WorktreeOptions) (*git.Worktree, error) {
	args := m.Called(ctx, repo, name, opts)
	return args.Get(0).(*git.Worktree), args.Error(1)
}

func (m *MockWorktreeManager) RemoveWorktree(ctx context.Context, repo *git.Repository, name string) error {
	args := m.Called(ctx, repo, name)
	return args.Error(0)
}

func (m *MockWorktreeManager) ListWorktrees(ctx context.Context, repo *git.Repository) ([]*git.Worktree, error) {
	args := m.Called(ctx, repo)
	return args.Get(0).([]*git.Worktree), args.Error(1)
}

func (m *MockWorktreeManager) CreateBranch(ctx context.Context, worktree *git.Worktree, branchName string, baseBranch string) error {
	args := m.Called(ctx, worktree, branchName, baseBranch)
	return args.Error(0)
}

func (m *MockWorktreeManager) ListBranches(ctx context.Context, repo *git.Repository) ([]git.Branch, error) {
	args := m.Called(ctx, repo)
	return args.Get(0).([]git.Branch), args.Error(1)
}

func (m *MockWorktreeManager) GetStatus(ctx context.Context, worktree *git.Worktree) (*git.Status, error) {
	args := m.Called(ctx, worktree)
	return args.Get(0).(*git.Status), args.Error(1)
}

func (m *MockWorktreeManager) Cleanup(ctx context.Context, repo *git.Repository) error {
	args := m.Called(ctx, repo)
	return args.Error(0)
}

// setupTestService 테스트용 서비스 설정
func setupTestService() (AgentService, storage.Storage, *MockDockerAdapter, *MockMonitoringService, *MockEventPublisher) {
	storage := memory.New()
	dockerAdapter := &MockDockerAdapter{}
	monitoring := &MockMonitoringService{}
	eventPublisher := &MockEventPublisher{}
	worktreeManager := &MockWorktreeManager{}

	service := NewAgentService(storage, dockerAdapter, monitoring, eventPublisher, worktreeManager)
	return service, storage, dockerAdapter, monitoring, eventPublisher
}

// TestCreateAgent 에이전트 생성 테스트
func TestCreateAgent(t *testing.T) {
	service, _, _, _, eventPublisher := setupTestService()
	ctx := context.Background()

	// 이벤트 발행 Mock 설정
	eventPublisher.On("PublishAgentCreated", mock.Anything, mock.Anything).Return(nil)

	// 테스트 케이스
	tests := []struct {
		name    string
		req     CreateAgentRequest
		wantErr bool
	}{
		{
			name: "성공_Claude_에이전트_생성",
			req: CreateAgentRequest{
				ProjectID:   "test-project-id",
				Name:        "test-agent",
				Type:        models.AgentTypeClaude,
				Description: "테스트 Claude 에이전트",
				Config: models.AgentConfig{
					Model:       "claude-3-sonnet",
					MaxTokens:   4000,
					Temperature: 0.7,
				},
			},
			wantErr: false,
		},
		{
			name: "실패_프로젝트ID_누락",
			req: CreateAgentRequest{
				Name: "test-agent",
				Type: models.AgentTypeClaude,
			},
			wantErr: true,
		},
		{
			name: "실패_이름_누락",
			req: CreateAgentRequest{
				ProjectID: "test-project-id",
				Type:      models.AgentTypeClaude,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent, err := service.CreateAgent(ctx, tt.req)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, agent)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, agent)
				assert.Equal(t, tt.req.Name, agent.Name)
				assert.Equal(t, tt.req.Type, agent.Type)
				assert.Equal(t, tt.req.ProjectID, agent.ProjectID)
				assert.Equal(t, models.AgentStatusCreated, agent.Status)
				assert.NotEmpty(t, agent.ID)
			}
		})
	}
}

// TestGetAgent 에이전트 조회 테스트
func TestGetAgent(t *testing.T) {
	service, _, _, _, eventPublisher := setupTestService()
	ctx := context.Background()

	// 이벤트 발행 Mock 설정
	eventPublisher.On("PublishAgentCreated", mock.Anything, mock.Anything).Return(nil)

	// 테스트 에이전트 생성
	createReq := CreateAgentRequest{
		ProjectID: "test-project-id",
		Name:      "test-agent",
		Type:      models.AgentTypeClaude,
	}
	agent, err := service.CreateAgent(ctx, createReq)
	assert.NoError(t, err)
	assert.NotNil(t, agent)

	// 테스트 케이스
	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{
			name:    "성공_에이전트_조회",
			id:      agent.ID,
			wantErr: false,
		},
		{
			name:    "실패_존재하지_않는_ID",
			id:      "non-existent-id",
			wantErr: true,
		},
		{
			name:    "실패_빈_ID",
			id:      "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.GetAgent(ctx, tt.id)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, agent.ID, result.ID)
				assert.Equal(t, agent.Name, result.Name)
			}
		})
	}
}

// TestUpdateAgent 에이전트 업데이트 테스트
func TestUpdateAgent(t *testing.T) {
	service, _, _, _, eventPublisher := setupTestService()
	ctx := context.Background()

	// 이벤트 발행 Mock 설정
	eventPublisher.On("PublishAgentCreated", mock.Anything, mock.Anything).Return(nil)

	// 테스트 에이전트 생성
	createReq := CreateAgentRequest{
		ProjectID: "test-project-id",
		Name:      "test-agent",
		Type:      models.AgentTypeClaude,
	}
	agent, err := service.CreateAgent(ctx, createReq)
	assert.NoError(t, err)
	assert.NotNil(t, agent)

	// 업데이트 테스트
	newName := "updated-agent"
	newDescription := "업데이트된 에이전트"
	updateReq := UpdateAgentRequest{
		Name:        &newName,
		Description: &newDescription,
	}

	updatedAgent, err := service.UpdateAgent(ctx, agent.ID, updateReq)
	assert.NoError(t, err)
	assert.NotNil(t, updatedAgent)
	assert.Equal(t, newName, updatedAgent.Name)
	assert.Equal(t, newDescription, updatedAgent.Description)
}

// TestDeleteAgent 에이전트 삭제 테스트
func TestDeleteAgent(t *testing.T) {
	service, _, _, _, eventPublisher := setupTestService()
	ctx := context.Background()

	// 이벤트 발행 Mock 설정
	eventPublisher.On("PublishAgentCreated", mock.Anything, mock.Anything).Return(nil)
	eventPublisher.On("PublishAgentDeleted", mock.Anything, mock.Anything).Return(nil)

	// 테스트 에이전트 생성
	createReq := CreateAgentRequest{
		ProjectID: "test-project-id",
		Name:      "test-agent",
		Type:      models.AgentTypeClaude,
	}
	agent, err := service.CreateAgent(ctx, createReq)
	assert.NoError(t, err)
	assert.NotNil(t, agent)

	// 삭제 테스트
	err = service.DeleteAgent(ctx, agent.ID)
	assert.NoError(t, err)

	// 삭제된 에이전트 조회 시 에러 확인
	_, err = service.GetAgent(ctx, agent.ID)
	assert.Error(t, err)
}

// TestGetAgentByProjectID 프로젝트별 에이전트 조회 테스트
func TestGetAgentByProjectID(t *testing.T) {
	service, _, _, _, eventPublisher := setupTestService()
	ctx := context.Background()

	// 이벤트 발행 Mock 설정
	eventPublisher.On("PublishAgentCreated", mock.Anything, mock.Anything).Return(nil)

	projectID := "test-project-id"

	// 테스트 에이전트들 생성
	agents := make([]*models.Agent, 3)
	for i := 0; i < 3; i++ {
		createReq := CreateAgentRequest{
			ProjectID: projectID,
			Name:      fmt.Sprintf("test-agent-%d", i),
			Type:      models.AgentTypeClaude,
		}
		agent, err := service.CreateAgent(ctx, createReq)
		assert.NoError(t, err)
		agents[i] = agent
	}

	// 프로젝트별 에이전트 조회
	result, err := service.GetAgentByProjectID(ctx, projectID)
	assert.NoError(t, err)
	assert.Len(t, result, 3)

	// 빈 프로젝트 ID 테스트
	_, err = service.GetAgentByProjectID(ctx, "")
	assert.Error(t, err)
}

// TestListActiveAgents 활성 에이전트 목록 조회 테스트
func TestListActiveAgents(t *testing.T) {
	service, storage, _, _, eventPublisher := setupTestService()
	ctx := context.Background()

	// 이벤트 발행 Mock 설정
	eventPublisher.On("PublishAgentCreated", mock.Anything, mock.Anything).Return(nil)

	// 테스트 에이전트 생성
	createReq := CreateAgentRequest{
		ProjectID: "test-project-id",
		Name:      "test-agent",
		Type:      models.AgentTypeClaude,
	}
	agent, err := service.CreateAgent(ctx, createReq)
	assert.NoError(t, err)

	// 에이전트 상태를 Running으로 변경
	err = storage.Agent().UpdateStatus(ctx, agent.ID, models.AgentStatusRunning, "")
	assert.NoError(t, err)

	// 활성 에이전트 조회
	activeAgents, err := service.ListActiveAgents(ctx)
	assert.NoError(t, err)
	assert.Len(t, activeAgents, 1)
	assert.Equal(t, agent.ID, activeAgents[0].ID)
}

// TestGetAgentStatus 에이전트 상태 조회 테스트
func TestGetAgentStatus(t *testing.T) {
	service, _, _, _, eventPublisher := setupTestService()
	ctx := context.Background()

	// 이벤트 발행 Mock 설정
	eventPublisher.On("PublishAgentCreated", mock.Anything, mock.Anything).Return(nil)

	// 테스트 에이전트 생성
	createReq := CreateAgentRequest{
		ProjectID: "test-project-id",
		Name:      "test-agent",
		Type:      models.AgentTypeClaude,
	}
	agent, err := service.CreateAgent(ctx, createReq)
	assert.NoError(t, err)

	// 에이전트 상태 조회
	status, err := service.GetAgentStatus(ctx, agent.ID)
	assert.NoError(t, err)
	assert.Equal(t, agent.ID, status.ID)
	assert.Equal(t, models.AgentStatusCreated, status.Status)
}

// TestStartMultipleAgents 여러 에이전트 시작 테스트
func TestStartMultipleAgents(t *testing.T) {
	service, _, dockerAdapter, monitoring, eventPublisher := setupTestService()
	ctx := context.Background()

	// Mock 설정
	eventPublisher.On("PublishAgentCreated", mock.Anything, mock.Anything).Return(nil)
	eventPublisher.On("PublishAgentStarted", mock.Anything, mock.Anything).Return(nil)
	dockerAdapter.On("CreateContainer", mock.Anything, mock.Anything).Return(&ContainerInfo{ID: "container-123"}, nil)
	dockerAdapter.On("StartContainer", mock.Anything, "container-123").Return(nil)
	dockerAdapter.On("GetContainerHealth", mock.Anything, "container-123").Return(HealthStatus{Status: "healthy"}, nil)
	monitoring.On("StartHealthMonitoring", mock.Anything, mock.Anything).Return(nil)

	// 테스트 에이전트들 생성
	var agentIDs []string
	for i := 0; i < 3; i++ {
		createReq := CreateAgentRequest{
			ProjectID: "test-project-id",
			Name:      fmt.Sprintf("test-agent-%d", i),
			Type:      models.AgentTypeClaude,
		}
		agent, err := service.CreateAgent(ctx, createReq)
		assert.NoError(t, err)
		agentIDs = append(agentIDs, agent.ID)
	}

	// 여러 에이전트 시작
	results, err := service.StartMultipleAgents(ctx, agentIDs)
	assert.NoError(t, err)
	assert.Len(t, results, 3)

	// 모든 결과가 성공인지 확인
	for _, result := range results {
		assert.True(t, result.Success)
		assert.Empty(t, result.Error)
	}
}

// TestCleanupStaleAgents 오래된 에이전트 정리 테스트
func TestCleanupStaleAgents(t *testing.T) {
	service, storage, _, _, eventPublisher := setupTestService()
	ctx := context.Background()

	// 이벤트 발행 Mock 설정
	eventPublisher.On("PublishAgentCreated", mock.Anything, mock.Anything).Return(nil)
	eventPublisher.On("PublishAgentDeleted", mock.Anything, mock.Anything).Return(nil)

	// 테스트 에이전트 생성
	createReq := CreateAgentRequest{
		ProjectID: "test-project-id",
		Name:      "test-agent",
		Type:      models.AgentTypeClaude,
	}
	agent, err := service.CreateAgent(ctx, createReq)
	assert.NoError(t, err)

	// 에이전트 상태를 오래된 Stopped로 변경 (25시간 전)
	oldTime := time.Now().Add(-25 * time.Hour)
	updates := map[string]interface{}{
		"status":        models.AgentStatusStopped,
		"last_activity": oldTime,
	}
	err = storage.Agent().Update(ctx, agent.ID, updates)
	assert.NoError(t, err)

	// 정리 작업 실행
	count, err := service.CleanupStaleAgents(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 1, count)

	// 에이전트가 삭제되었는지 확인
	_, err = service.GetAgent(ctx, agent.ID)
	assert.Error(t, err)
}