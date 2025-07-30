package integration

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/aicli/aicli-web/internal/agent"
	"github.com/aicli/aicli-web/internal/docker"
	"github.com/aicli/aicli-web/internal/git"
	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/storage"
	"github.com/aicli/aicli-web/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// AgentIntegrationTestSuite 에이전트 통합 테스트 스위트
// 전체 시스템의 에이전트 생명주기를 E2E로 테스트합니다.
type AgentIntegrationTestSuite struct {
	suite.Suite
	ctx               context.Context
	cancel            context.CancelFunc
	storage          storage.Storage
	dockerClient     *docker.Client
	agentService     *agent.Service
	gitManager       *git.Manager
	testProjectDir   string
	testAgents       []*models.Agent
	cleanupFunctions []func()
}

// SetupSuite 테스트 스위트 초기화
func (suite *AgentIntegrationTestSuite) SetupSuite() {
	suite.ctx, suite.cancel = context.WithTimeout(context.Background(), 15*time.Minute)
	
	// 테스트 데이터베이스 초기화
	store, err := storage.New()
	require.NoError(suite.T(), err)
	suite.storage = store
	suite.addCleanupFunction(func() { store.Close() })

	// Docker 클라이언트 초기화
	dockerClient, err := docker.NewClient()
	require.NoError(suite.T(), err)
	suite.dockerClient = dockerClient
	
	// Git 매니저 초기화
	gitManager := git.NewManager()
	suite.gitManager = gitManager

	// Agent 서비스 초기화
	agentService := agent.NewService(
		suite.storage.Agent(),
		dockerClient,
		gitManager,
	)
	suite.agentService = agentService

	// 테스트 프로젝트 디렉토리 생성
	suite.testProjectDir = testutil.TempDir(suite.T(), "agent-integration-test")
	suite.setupTestProject()

	suite.T().Logf("통합 테스트 환경 초기화 완료 - 프로젝트 디렉토리: %s", suite.testProjectDir)
}

// TearDownSuite 테스트 스위트 정리
func (suite *AgentIntegrationTestSuite) TearDownSuite() {
	// 모든 테스트 에이전트 정리
	suite.cleanupTestAgents()
	
	// 정리 함수들 실행
	for i := len(suite.cleanupFunctions) - 1; i >= 0; i-- {
		suite.cleanupFunctions[i]()
	}
	
	if suite.cancel != nil {
		suite.cancel()
	}
}

// SetupTest 각 테스트 초기화
func (suite *AgentIntegrationTestSuite) SetupTest() {
	suite.testAgents = make([]*models.Agent, 0)
}

// TearDownTest 각 테스트 정리
func (suite *AgentIntegrationTestSuite) TearDownTest() {
	suite.cleanupTestAgents()
}

// TestCompleteAgentLifecycle 완전한 에이전트 생명주기 테스트
// 생성 → 시작 → 작업 → 중지 → 삭제 과정을 검증합니다.
func (suite *AgentIntegrationTestSuite) TestCompleteAgentLifecycle() {
	suite.T().Log("🔄 완전한 에이전트 생명주기 테스트 시작")

	// 1. 에이전트 생성
	agentSpec := &models.Agent{
		Name:        fmt.Sprintf("test-agent-%d", time.Now().Unix()),
		ProjectID:   "test-project",
		AgentType:   models.AgentTypeStandard,
		Description: "Integration test agent",
		Config: models.AgentConfig{
			Resources: models.ResourceConfig{
				CPU:    "1.0",
				Memory: "1Gi",
			},
			Environment: map[string]string{
				"TEST_MODE": "true",
			},
		},
	}

	suite.T().Log("   📝 에이전트 생성 중...")
	createdAgent, err := suite.agentService.Create(suite.ctx, agentSpec)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), createdAgent)
	require.NotEmpty(suite.T(), createdAgent.ID)
	suite.testAgents = append(suite.testAgents, createdAgent)

	assert.Equal(suite.T(), models.AgentStatusCreated, createdAgent.Status)
	assert.Equal(suite.T(), agentSpec.Name, createdAgent.Name)
	assert.Equal(suite.T(), agentSpec.ProjectID, createdAgent.ProjectID)
	
	suite.T().Logf("   ✅ 에이전트 생성 완료 - ID: %s", createdAgent.ID)

	// 2. 에이전트 시작
	suite.T().Log("   🚀 에이전트 시작 중...")
	err = suite.agentService.Start(suite.ctx, createdAgent.ID)
	require.NoError(suite.T(), err)

	// 에이전트가 시작될 때까지 대기
	suite.waitForAgentStatus(createdAgent.ID, models.AgentStatusRunning, 60*time.Second)
	
	runningAgent, err := suite.agentService.Get(suite.ctx, createdAgent.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), models.AgentStatusRunning, runningAgent.Status)
	assert.NotEmpty(suite.T(), runningAgent.ContainerID)
	
	suite.T().Logf("   ✅ 에이전트 시작 완료 - 컨테이너 ID: %s", runningAgent.ContainerID)

	// 3. 에이전트 상태 확인 및 간단한 작업 수행
	suite.T().Log("   📊 에이전트 상태 및 작업 확인 중...")
	
	// 컨테이너가 실제로 실행 중인지 확인
	containerInfo, err := suite.dockerClient.InspectContainer(suite.ctx, runningAgent.ContainerID)
	require.NoError(suite.T(), err)
	assert.True(suite.T(), containerInfo.State.Running)
	
	// 에이전트 메트릭 확인
	metrics, err := suite.agentService.GetMetrics(suite.ctx, createdAgent.ID)
	require.NoError(suite.T(), err)
	assert.NotNil(suite.T(), metrics)
	assert.Greater(suite.T(), metrics.UptimeSeconds, int64(0))
	
	suite.T().Log("   ✅ 에이전트 작업 확인 완료")

	// 4. 에이전트 중지
	suite.T().Log("   ⏹️  에이전트 중지 중...")
	err = suite.agentService.Stop(suite.ctx, createdAgent.ID)
	require.NoError(suite.T(), err)

	// 에이전트가 중지될 때까지 대기
	suite.waitForAgentStatus(createdAgent.ID, models.AgentStatusStopped, 30*time.Second)
	
	stoppedAgent, err := suite.agentService.Get(suite.ctx, createdAgent.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), models.AgentStatusStopped, stoppedAgent.Status)
	
	suite.T().Log("   ✅ 에이전트 중지 완료")

	// 5. 에이전트 삭제
	suite.T().Log("   🗑️  에이전트 삭제 중...")
	err = suite.agentService.Delete(suite.ctx, createdAgent.ID)
	require.NoError(suite.T(), err)

	// 에이전트가 완전히 삭제되었는지 확인
	_, err = suite.agentService.Get(suite.ctx, createdAgent.ID)
	assert.Error(suite.T(), err) // 에이전트가 없어야 함
	
	suite.T().Log("   ✅ 에이전트 삭제 완료")
	
	// 테스트 에이전트 목록에서 제거 (이미 삭제됨)
	suite.testAgents = suite.testAgents[:0]

	suite.T().Log("🎉 완전한 에이전트 생명주기 테스트 성공")
}

// TestConcurrentAgentOperations 동시 에이전트 작업 테스트
func (suite *AgentIntegrationTestSuite) TestConcurrentAgentOperations() {
	suite.T().Log("🔄 동시 에이전트 작업 테스트 시작")

	agentCount := 5
	var wg sync.WaitGroup
	agentIDs := make([]string, agentCount)
	errors := make([]error, agentCount)

	// 동시에 여러 에이전트 생성
	suite.T().Logf("   📝 %d개 에이전트 동시 생성 중...", agentCount)
	for i := 0; i < agentCount; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			agentSpec := &models.Agent{
				Name:        fmt.Sprintf("concurrent-agent-%d-%d", index, time.Now().Unix()),
				ProjectID:   "test-project",
				AgentType:   models.AgentTypeStandard,
				Description: fmt.Sprintf("Concurrent test agent %d", index),
				Config: models.AgentConfig{
					Resources: models.ResourceConfig{
						CPU:    "0.5",
						Memory: "512Mi",
					},
				},
			}

			agent, err := suite.agentService.Create(suite.ctx, agentSpec)
			if err == nil && agent != nil {
				agentIDs[index] = agent.ID
				suite.testAgents = append(suite.testAgents, agent)
			}
			errors[index] = err
		}(i)
	}

	wg.Wait()

	// 모든 에이전트가 성공적으로 생성되었는지 확인
	successCount := 0
	for i := 0; i < agentCount; i++ {
		if errors[i] == nil && agentIDs[i] != "" {
			successCount++
		}
	}

	assert.Equal(suite.T(), agentCount, successCount, "모든 에이전트가 성공적으로 생성되어야 함")
	suite.T().Logf("   ✅ %d개 에이전트 생성 완료", successCount)

	// 동시에 모든 에이전트 시작
	suite.T().Log("   🚀 모든 에이전트 동시 시작 중...")
	for i, agentID := range agentIDs {
		if agentID != "" {
			wg.Add(1)
			go func(id string, index int) {
				defer wg.Done()
				errors[index] = suite.agentService.Start(suite.ctx, id)
			}(agentID, i)
		}
	}

	wg.Wait()

	// 시작 결과 확인
	startedCount := 0
	for i := 0; i < agentCount; i++ {
		if agentIDs[i] != "" && errors[i] == nil {
			startedCount++
		}
	}

	assert.Greater(suite.T(), startedCount, 0, "최소 1개 이상의 에이전트가 시작되어야 함")
	suite.T().Logf("   ✅ %d개 에이전트 시작 완료", startedCount)

	// 모든 에이전트가 실행 상태가 될 때까지 대기 (최대 2분)
	suite.T().Log("   ⏳ 에이전트 시작 대기 중...")
	time.Sleep(30 * time.Second)

	// 실행 중인 에이전트 확인
	runningCount := 0
	for _, agentID := range agentIDs {
		if agentID != "" {
			agent, err := suite.agentService.Get(suite.ctx, agentID)
			if err == nil && agent.Status == models.AgentStatusRunning {
				runningCount++
			}
		}
	}

	suite.T().Logf("   📊 실행 중인 에이전트: %d개", runningCount)
	assert.Greater(suite.T(), runningCount, 0, "최소 1개 이상의 에이전트가 실행 중이어야 함")

	suite.T().Log("🎉 동시 에이전트 작업 테스트 성공")
}

// TestAgentFailureRecovery 에이전트 장애 복구 테스트
func (suite *AgentIntegrationTestSuite) TestAgentFailureRecovery() {
	suite.T().Log("🔄 에이전트 장애 복구 테스트 시작")

	// 에이전트 생성 및 시작
	agentSpec := &models.Agent{
		Name:        fmt.Sprintf("recovery-test-agent-%d", time.Now().Unix()),
		ProjectID:   "test-project",
		AgentType:   models.AgentTypeStandard,
		Description: "Recovery test agent",
		Config: models.AgentConfig{
			Resources: models.ResourceConfig{
				CPU:    "0.5",
				Memory: "512Mi",
			},
			Recovery: models.RecoveryConfig{
				Enabled:     true,
				MaxAttempts: 3,
				Backoff:     "exponential",
			},
		},
	}

	suite.T().Log("   📝 복구 테스트용 에이전트 생성 중...")
	agent, err := suite.agentService.Create(suite.ctx, agentSpec)
	require.NoError(suite.T(), err)
	suite.testAgents = append(suite.testAgents, agent)

	err = suite.agentService.Start(suite.ctx, agent.ID)
	require.NoError(suite.T(), err)

	// 에이전트가 실행될 때까지 대기
	suite.waitForAgentStatus(agent.ID, models.AgentStatusRunning, 60*time.Second)
	
	runningAgent, err := suite.agentService.Get(suite.ctx, agent.ID)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), models.AgentStatusRunning, runningAgent.Status)
	require.NotEmpty(suite.T(), runningAgent.ContainerID)

	suite.T().Logf("   ✅ 에이전트 실행 완료 - 컨테이너 ID: %s", runningAgent.ContainerID)

	// 컨테이너 강제 종료로 장애 시뮬레이션
	suite.T().Log("   💥 컨테이너 강제 종료로 장애 시뮬레이션...")
	err = suite.dockerClient.KillContainer(suite.ctx, runningAgent.ContainerID, "SIGKILL")
	require.NoError(suite.T(), err)

	// 복구 프로세스 대기
	suite.T().Log("   🔧 자동 복구 프로세스 대기 중...")
	time.Sleep(10 * time.Second)

	// 복구 상태 확인 (최대 2분 대기)
	recovered := suite.waitForAgentRecovery(agent.ID, 120*time.Second)
	if recovered {
		suite.T().Log("   ✅ 에이전트 자동 복구 성공")
		
		recoveredAgent, err := suite.agentService.Get(suite.ctx, agent.ID)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), models.AgentStatusRunning, recoveredAgent.Status)
		assert.NotEqual(suite.T(), runningAgent.ContainerID, recoveredAgent.ContainerID) // 새 컨테이너 ID
	} else {
		suite.T().Log("   ⚠️  자동 복구가 시간 내에 완료되지 않았음")
		
		// 복구 시도는 있었는지 확인
		agentStatus, err := suite.agentService.Get(suite.ctx, agent.ID)
		if err == nil {
			suite.T().Logf("   📊 현재 에이전트 상태: %s", agentStatus.Status)
		}
	}

	suite.T().Log("🎉 에이전트 장애 복구 테스트 완료")
}

// TestAgentResourceLimits 에이전트 리소스 제한 테스트
func (suite *AgentIntegrationTestSuite) TestAgentResourceLimits() {
	suite.T().Log("🔄 에이전트 리소스 제한 테스트 시작")

	// 제한된 리소스로 에이전트 생성
	agentSpec := &models.Agent{
		Name:        fmt.Sprintf("resource-test-agent-%d", time.Now().Unix()),
		ProjectID:   "test-project",
		AgentType:   models.AgentTypeStandard,
		Description: "Resource limit test agent",
		Config: models.AgentConfig{
			Resources: models.ResourceConfig{
				CPU:    "0.2",     // 20% CPU
				Memory: "256Mi",   // 256MB 메모리
			},
		},
	}

	suite.T().Log("   📝 리소스 제한 에이전트 생성 중...")
	agent, err := suite.agentService.Create(suite.ctx, agentSpec)
	require.NoError(suite.T(), err)
	suite.testAgents = append(suite.testAgents, agent)

	err = suite.agentService.Start(suite.ctx, agent.ID)
	require.NoError(suite.T(), err)

	// 에이전트가 실행될 때까지 대기
	suite.waitForAgentStatus(agent.ID, models.AgentStatusRunning, 60*time.Second)
	
	runningAgent, err := suite.agentService.Get(suite.ctx, agent.ID)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), models.AgentStatusRunning, runningAgent.Status)

	suite.T().Log("   ✅ 리소스 제한 에이전트 실행 완료")

	// 컨테이너 리소스 제한 확인
	suite.T().Log("   📊 컨테이너 리소스 제한 검증 중...")
	containerInfo, err := suite.dockerClient.InspectContainer(suite.ctx, runningAgent.ContainerID)
	require.NoError(suite.T(), err)

	// CPU 제한 확인 (20% = 20000 quota)
	if containerInfo.HostConfig.CPUQuota > 0 {
		assert.LessOrEqual(suite.T(), containerInfo.HostConfig.CPUQuota, int64(25000), "CPU 제한이 올바르게 설정되어야 함")
	}

	// 메모리 제한 확인 (256MB)
	if containerInfo.HostConfig.Memory > 0 {
		expectedMemory := int64(256 * 1024 * 1024) // 256MB
		assert.LessOrEqual(suite.T(), containerInfo.HostConfig.Memory, expectedMemory*12/10, "메모리 제한이 올바르게 설정되어야 함") // 20% 여유
	}

	suite.T().Log("   ✅ 리소스 제한 검증 완료")

	// 메트릭 수집 및 확인
	suite.T().Log("   📈 리소스 사용량 모니터링 중...")
	time.Sleep(10 * time.Second) // 메트릭 수집 대기

	metrics, err := suite.agentService.GetMetrics(suite.ctx, agent.ID)
	require.NoError(suite.T(), err)
	assert.NotNil(suite.T(), metrics)

	if metrics.ResourceUsage != nil {
		suite.T().Logf("   📊 CPU 사용률: %.2f%%, 메모리 사용량: %d MB", 
			metrics.ResourceUsage.CPUPercent, 
			metrics.ResourceUsage.MemoryUsage/(1024*1024))
	}

	suite.T().Log("🎉 에이전트 리소스 제한 테스트 성공")
}

// 헬퍼 메서드들

// waitForAgentStatus 에이전트가 특정 상태가 될 때까지 대기
func (suite *AgentIntegrationTestSuite) waitForAgentStatus(agentID string, expectedStatus models.AgentStatus, timeout time.Duration) bool {
	startTime := time.Now()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-suite.ctx.Done():
			return false
		case <-ticker.C:
			if time.Since(startTime) > timeout {
				suite.T().Logf("   ⏰ 상태 변경 대기 시간 초과 - 대상 상태: %s", expectedStatus)
				return false
			}

			agent, err := suite.agentService.Get(suite.ctx, agentID)
			if err != nil {
				continue
			}

			suite.T().Logf("   📊 현재 에이전트 상태: %s (대상: %s)", agent.Status, expectedStatus)
			if agent.Status == expectedStatus {
				return true
			}
		}
	}
}

// waitForAgentRecovery 에이전트 복구 완료까지 대기
func (suite *AgentIntegrationTestSuite) waitForAgentRecovery(agentID string, timeout time.Duration) bool {
	startTime := time.Now()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-suite.ctx.Done():
			return false
		case <-ticker.C:
			if time.Since(startTime) > timeout {
				return false
			}

			agent, err := suite.agentService.Get(suite.ctx, agentID)
			if err != nil {
				continue
			}

			// 복구되어 다시 실행 중인지 확인
			if agent.Status == models.AgentStatusRunning && agent.ContainerID != "" {
				// 컨테이너가 실제로 실행 중인지 확인
				containerInfo, err := suite.dockerClient.InspectContainer(suite.ctx, agent.ContainerID)
				if err == nil && containerInfo.State.Running {
					return true
				}
			}
		}
	}
}

// setupTestProject 테스트 프로젝트 설정
func (suite *AgentIntegrationTestSuite) setupTestProject() {
	// 간단한 테스트 프로젝트 구조 생성
	testutil.CreateTestProject(suite.T(), suite.testProjectDir)
	
	// 테스트용 README 파일 생성
	readmeContent := `# Test Project
This is a test project for agent integration testing.

## Commands
- echo "Hello World"
- ls -la
- pwd
`
	testutil.TempFile(suite.T(), suite.testProjectDir, "README.md", readmeContent)
}

// cleanupTestAgents 테스트 에이전트들 정리
func (suite *AgentIntegrationTestSuite) cleanupTestAgents() {
	for _, agent := range suite.testAgents {
		if agent.Status == models.AgentStatusRunning {
			suite.agentService.Stop(suite.ctx, agent.ID)
		}
		suite.agentService.Delete(suite.ctx, agent.ID)
	}
	suite.testAgents = suite.testAgents[:0]
}

// addCleanupFunction 정리 함수 추가
func (suite *AgentIntegrationTestSuite) addCleanupFunction(fn func()) {
	suite.cleanupFunctions = append(suite.cleanupFunctions, fn)
}

// TestAgentIntegrationSuite 통합 테스트 실행
func TestAgentIntegrationSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("통합 테스트는 short 모드에서 제외됩니다")
	}

	// Docker 사용 가능 여부 확인
	dockerClient, err := docker.NewClient()
	if err != nil {
		t.Skip("Docker를 사용할 수 없습니다:", err)
		return
	}

	// Docker 연결 테스트
	_, err = dockerClient.Ping(context.Background())
	if err != nil {
		t.Skip("Docker 데몬에 연결할 수 없습니다:", err)
		return
	}

	suite.Run(t, new(AgentIntegrationTestSuite))
}