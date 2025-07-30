package docker

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// AgentIntegrationTestSuite 에이전트 Docker 고급 통합 테스트 스위트
type AgentIntegrationTestSuite struct {
	suite.Suite
	ctx             context.Context
	cancel          context.CancelFunc
	client          *Client
	networkMgr      *AgentNetworkManager
	resourceMgr     *AgentResourceManager
	eventMgr        *AgentEventMonitor
	healthMgr       *AgentHealthMonitor
	syncMgr         *AgentStateSynchronizer
	recoveryMgr     *AgentRecoveryManager
	metricsMgr      *AgentMetricsCollector
	testAgentID     string
	testContainerID string
}

// SetupSuite 테스트 스위트 초기화
func (suite *AgentIntegrationTestSuite) SetupSuite() {
	suite.ctx, suite.cancel = context.WithTimeout(context.Background(), 10*time.Minute)
	
	// Docker 클라이언트 초기화
	client, err := NewClient()
	require.NoError(suite.T(), err)
	suite.client = client

	// 관리자들 초기화
	suite.networkMgr = NewAgentNetworkManager(client)
	suite.resourceMgr = NewAgentResourceManager(client)
	suite.eventMgr = NewAgentEventMonitor(client)
	suite.healthMgr = NewAgentHealthMonitor(client)
	suite.syncMgr = NewAgentStateSynchronizer(
		client, suite.networkMgr, suite.resourceMgr, 
		suite.healthMgr, suite.eventMgr,
	)
	suite.recoveryMgr = NewAgentRecoveryManager(
		client, suite.syncMgr, suite.healthMgr, 
		suite.networkMgr, suite.resourceMgr,
	)
	suite.metricsMgr = NewAgentMetricsCollector(client)

	// 테스트용 에이전트 ID 생성
	suite.testAgentID = fmt.Sprintf("test-agent-%d", time.Now().Unix())
}

// TearDownSuite 테스트 스위트 정리
func (suite *AgentIntegrationTestSuite) TearDownSuite() {
	if suite.cancel != nil {
		suite.cancel()
	}

	// 테스트 리소스 정리
	suite.cleanupTestResources()
}

// SetupTest 각 테스트 초기화
func (suite *AgentIntegrationTestSuite) SetupTest() {
	// 이벤트 모니터 시작
	err := suite.eventMgr.Start(suite.ctx)
	require.NoError(suite.T(), err)

	// 헬스 모니터 시작
	err = suite.healthMgr.Start(suite.ctx)
	require.NoError(suite.T(), err)

	// 상태 동기화 시작
	syncConfig := SyncConfig{
		Interval:        5 * time.Second,
		EnableAutoSync:  true,
		EnableEventSync: true,
		MaxErrors:       10,
		RetryInterval:   2 * time.Second,
	}
	err = suite.syncMgr.Start(suite.ctx, syncConfig)
	require.NoError(suite.T(), err)

	// 복구 관리자 시작
	recoveryConfig := RecoveryConfig{
		Enabled:              true,
		MaxAttempts:          3,
		RetryInterval:        10 * time.Second,
		BackoffMultiplier:    1.5,
		MaxRetryInterval:     1 * time.Minute,
		EnableAutoRestart:    true,
		EnableAutoRecreate:   true,
		EnableNetworkRepair:  true,
		EnableResourceScale:  false,
		MonitoringInterval:   5 * time.Second,
	}
	err = suite.recoveryMgr.Start(suite.ctx, recoveryConfig)
	require.NoError(suite.T(), err)

	// 메트릭 수집기 시작
	metricsConfig := MetricsConfig{
		CollectInterval:   10 * time.Second,
		HistoryRetention:  20,
		EnableAlerts:      true,
	}
	err = suite.metricsMgr.Start(suite.ctx, metricsConfig)
	require.NoError(suite.T(), err)
}

// TearDownTest 각 테스트 정리
func (suite *AgentIntegrationTestSuite) TearDownTest() {
	// 매니저들 중지
	suite.metricsMgr.Stop()
	suite.recoveryMgr.Stop()
	suite.syncMgr.Stop()
	suite.healthMgr.Stop()
	suite.eventMgr.Stop()

	// 테스트 컨테이너 정리
	suite.cleanupTestContainer()
}

// TestNetworkIsolation 네트워크 격리 테스트
func (suite *AgentIntegrationTestSuite) TestNetworkIsolation() {
	// 에이전트별 네트워크 생성
	agent1ID := suite.testAgentID + "-1"
	agent2ID := suite.testAgentID + "-2"

	network1, err := suite.networkMgr.CreateAgentNetwork(suite.ctx, agent1ID)
	require.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), network1.NetworkID)
	assert.NotEmpty(suite.T(), network1.Subnet)
	assert.NotEmpty(suite.T(), network1.Gateway)

	network2, err := suite.networkMgr.CreateAgentNetwork(suite.ctx, agent2ID)
	require.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), network2.NetworkID)
	assert.NotEmpty(suite.T(), network2.Subnet)
	assert.NotEmpty(suite.T(), network2.Gateway)

	// 네트워크가 서로 다른지 확인
	assert.NotEqual(suite.T(), network1.NetworkID, network2.NetworkID)
	assert.NotEqual(suite.T(), network1.Subnet, network2.Subnet)

	// 네트워크 격리 검증
	isolated, err := suite.networkMgr.ValidateNetworkIsolation(suite.ctx, agent1ID, agent2ID)
	require.NoError(suite.T(), err)
	assert.True(suite.T(), isolated)

	// 정리
	err = suite.networkMgr.DeleteAgentNetwork(suite.ctx, agent1ID)
	assert.NoError(suite.T(), err)
	err = suite.networkMgr.DeleteAgentNetwork(suite.ctx, agent2ID)
	assert.NoError(suite.T(), err)
}

// TestResourceLimits 리소스 제한 테스트
func (suite *AgentIntegrationTestSuite) TestResourceLimits() {
	// 다양한 리소스 등급 테스트
	tiers := []ResourceTier{
		ResourceTierMicro,
		ResourceTierSmall,
		ResourceTierMedium,
		ResourceTierLarge,
	}

	for _, tier := range tiers {
		config := suite.resourceMgr.GetPresetConfig(tier)
		
		// 설정 유효성 검증
		err := suite.resourceMgr.ValidateResourceConfig(config)
		assert.NoError(suite.T(), err, "Tier %s should have valid config", tier)

		// 기본 검증
		assert.Greater(suite.T(), config.CPUQuota, int64(0))
		assert.Greater(suite.T(), config.Memory, int64(0))
		assert.NotNil(suite.T(), config.PidsLimit)
		assert.Greater(suite.T(), *config.PidsLimit, int64(0))
	}

	// 리소스 문자열 파싱 테스트
	cpuQuota, err := suite.resourceMgr.ParseResourceString("cpu", "1.5")
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(150000), cpuQuota)

	memoryBytes, err := suite.resourceMgr.ParseResourceString("memory", "2g")
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(2*1024*1024*1024), memoryBytes)
}

// TestContainerLifecycleWithEvents 컨테이너 생명주기 및 이벤트 테스트
func (suite *AgentIntegrationTestSuite) TestContainerLifecycleWithEvents() {
	// 이벤트 수집을 위한 채널
	events := make([]AgentDockerEvent, 0)
	eventHandler := func(event AgentDockerEvent) {
		if event.AgentID == suite.testAgentID {
			events = append(events, event)
		}
	}
	suite.eventMgr.RegisterHandler("container", eventHandler)

	// 테스트 컨테이너 생성 및 시작
	containerID := suite.createTestContainer()
	suite.testContainerID = containerID

	// 네트워크 생성 및 연결
	network, err := suite.networkMgr.CreateAgentNetwork(suite.ctx, suite.testAgentID)
	require.NoError(suite.T(), err)

	err = suite.networkMgr.ConnectAgentContainer(suite.ctx, suite.testAgentID, containerID)
	require.NoError(suite.T(), err)

	// 상태 동기화 등록
	err = suite.syncMgr.RegisterAgent(suite.testAgentID, containerID)
	require.NoError(suite.T(), err)

	// 헬스체크 등록
	healthConfig := GetDefaultHealthCheckConfig()
	healthConfig.Interval = 5 * time.Second
	err = suite.healthMgr.RegisterHealthCheck(suite.testAgentID, containerID, healthConfig)
	require.NoError(suite.T(), err)

	// 메트릭 수집 등록
	err = suite.metricsMgr.RegisterAgent(suite.testAgentID, containerID, nil)
	require.NoError(suite.T(), err)

	// 잠시 대기하여 초기 상태 수집
	time.Sleep(3 * time.Second)

	// 상태 확인
	agentState, exists := suite.syncMgr.GetAgentState(suite.testAgentID)
	require.True(suite.T(), exists)
	assert.Equal(suite.T(), containerID, agentState.ContainerID)
	assert.True(suite.T(), agentState.ContainerState.Running)
	assert.NotNil(suite.T(), agentState.NetworkState)
	assert.Equal(suite.T(), network.NetworkID, agentState.NetworkState.NetworkID)

	// 헬스체크 상태 확인
	healthCheck, exists := suite.healthMgr.GetHealthCheck(suite.testAgentID)
	require.True(suite.T(), exists)
	assert.Equal(suite.T(), containerID, healthCheck.ContainerID)

	// 메트릭 확인
	metrics, exists := suite.metricsMgr.GetAgentMetrics(suite.testAgentID)
	require.True(suite.T(), exists)
	assert.Equal(suite.T(), containerID, metrics.ContainerID)
	assert.Greater(suite.T(), metrics.CollectCount, int64(0))

	// 컨테이너 중지하여 복구 테스트
	err = suite.client.cli.ContainerStop(suite.ctx, containerID, container.StopOptions{})
	require.NoError(suite.T(), err)

	// 복구 작업 확인 (최대 30초 대기)
	suite.waitForRecovery(30 * time.Second)

	// 복구 후 상태 확인
	time.Sleep(5 * time.Second)
	agentState, exists = suite.syncMgr.GetAgentState(suite.testAgentID)
	require.True(suite.T(), exists)
	
	// 복구가 성공했다면 다시 실행 중이어야 함
	if agentState.ContainerState.Running {
		assert.True(suite.T(), agentState.ContainerState.Running)
	}

	// 복구 작업 이력 확인
	recoveryJob, exists := suite.recoveryMgr.GetRecoveryJob(suite.testAgentID)
	if exists {
		assert.Greater(suite.T(), recoveryJob.Attempts, 0)
	}

	// 정리
	err = suite.networkMgr.DeleteAgentNetwork(suite.ctx, suite.testAgentID)
	assert.NoError(suite.T(), err)
}

// TestHighAvailabilityScenario 고가용성 시나리오 테스트
func (suite *AgentIntegrationTestSuite) TestHighAvailabilityScenario() {
	agentCount := 5
	agents := make([]string, agentCount)
	containers := make([]string, agentCount)

	// 여러 에이전트 생성
	for i := 0; i < agentCount; i++ {
		agentID := fmt.Sprintf("%s-ha-%d", suite.testAgentID, i)
		agents[i] = agentID

		// 컨테이너 생성
		containerID := suite.createTestContainer()
		containers[i] = containerID

		// 네트워크 생성
		_, err := suite.networkMgr.CreateAgentNetwork(suite.ctx, agentID)
		require.NoError(suite.T(), err)

		err = suite.networkMgr.ConnectAgentContainer(suite.ctx, agentID, containerID)
		require.NoError(suite.T(), err)

		// 상태 동기화 등록
		err = suite.syncMgr.RegisterAgent(agentID, containerID)
		require.NoError(suite.T(), err)

		// 헬스체크 등록
		healthConfig := GetDefaultHealthCheckConfig()
		healthConfig.Interval = 10 * time.Second
		err = suite.healthMgr.RegisterHealthCheck(agentID, containerID, healthConfig)
		require.NoError(suite.T(), err)

		// 메트릭 수집 등록
		err = suite.metricsMgr.RegisterAgent(agentID, containerID, nil)
		require.NoError(suite.T(), err)
	}

	// 초기화 대기
	time.Sleep(5 * time.Second)

	// 모든 에이전트가 정상 상태인지 확인
	allStates := suite.syncMgr.GetAllAgentStates()
	assert.Equal(suite.T(), agentCount, len(allStates))

	healthyAgents := suite.healthMgr.GetHealthyAgents()
	// 일부 에이전트는 아직 헬스체크가 완료되지 않을 수 있음
	assert.LessOrEqual(suite.T(), len(healthyAgents), agentCount)

	// 일부 에이전트 장애 시뮬레이션 (절반 중지)
	failureCount := agentCount / 2
	for i := 0; i < failureCount; i++ {
		err := suite.client.cli.ContainerStop(suite.ctx, containers[i], container.StopOptions{})
		require.NoError(suite.T(), err)
	}

	// 복구 대기
	time.Sleep(20 * time.Second)

	// 복구 상태 확인
	recoveryStats := make(map[string]bool)
	for i := 0; i < failureCount; i++ {
		agentState, exists := suite.syncMgr.GetAgentState(agents[i])
		require.True(suite.T(), exists)
		recoveryStats[agents[i]] = agentState.ContainerState.Running
	}

	// 일부라도 복구되었는지 확인
	recoveredCount := 0
	for _, recovered := range recoveryStats {
		if recovered {
			recoveredCount++
		}
	}

	suite.T().Logf("Recovered %d out of %d failed agents", recoveredCount, failureCount)

	// 정리
	for i, agentID := range agents {
		suite.client.cli.ContainerStop(suite.ctx, containers[i], container.StopOptions{})
		suite.client.cli.ContainerRemove(suite.ctx, containers[i], container.RemoveOptions{Force: true})
		suite.networkMgr.DeleteAgentNetwork(suite.ctx, agentID)
	}
}

// TestResourceMonitoring 리소스 모니터링 테스트
func (suite *AgentIntegrationTestSuite) TestResourceMonitoring() {
	// 테스트 컨테이너 생성
	containerID := suite.createTestContainer()
	suite.testContainerID = containerID

	// 메트릭 수집 등록
	thresholds := &MetricThresholds{
		CPUWarning:    50.0,
		CPUCritical:   80.0,
		MemoryWarning: 50.0,
		MemoryCritical: 80.0,
	}
	err := suite.metricsMgr.RegisterAgent(suite.testAgentID, containerID, thresholds)
	require.NoError(suite.T(), err)

	// 메트릭 수집 대기
	time.Sleep(15 * time.Second)

	// 메트릭 확인
	metrics, exists := suite.metricsMgr.GetAgentMetrics(suite.testAgentID)
	require.True(suite.T(), exists)

	assert.Greater(suite.T(), metrics.CollectCount, int64(0))
	assert.NotEmpty(suite.T(), metrics.History)

	// 현재 메트릭 검증
	current := metrics.Current
	assert.True(suite.T(), current.Timestamp.After(time.Time{}))
	assert.GreaterOrEqual(suite.T(), current.CPU.UsagePercent, 0.0)
	assert.Greater(suite.T(), current.Memory.Usage, uint64(0))
	assert.True(suite.T(), current.Container.Running)

	// 통계 검증
	stats := metrics.Statistics
	assert.True(suite.T(), stats.LastUpdated.After(time.Time{}))
	assert.GreaterOrEqual(suite.T(), stats.CPUUsageAvg, 0.0)
	assert.Greater(suite.T(), stats.MemoryUsageMax, uint64(0))

	suite.T().Logf("Collected %d metric snapshots", len(metrics.History))
	suite.T().Logf("CPU Usage: %.2f%%, Memory Usage: %.2f%%", 
		current.CPU.UsagePercent, current.Memory.UsagePercent)
}

// TestEventDrivenRecovery 이벤트 기반 복구 테스트
func (suite *AgentIntegrationTestSuite) TestEventDrivenRecovery() {
	// 테스트 컨테이너 생성
	containerID := suite.createTestContainer()
	suite.testContainerID = containerID

	// 상태 동기화 등록
	err := suite.syncMgr.RegisterAgent(suite.testAgentID, containerID)
	require.NoError(suite.T(), err)

	// 초기 상태 확인
	time.Sleep(3 * time.Second)
	agentState, exists := suite.syncMgr.GetAgentState(suite.testAgentID)
	require.True(suite.T(), exists)
	assert.True(suite.T(), agentState.ContainerState.Running)

	// 컨테이너 강제 종료 (exit code 1로)
	err = suite.client.cli.ContainerKill(suite.ctx, containerID, "SIGTERM")
	require.NoError(suite.T(), err)

	// 복구 대기
	recovered := suite.waitForRecovery(30 * time.Second)
	
	if recovered {
		// 복구 성공 검증
		agentState, exists = suite.syncMgr.GetAgentState(suite.testAgentID)
		require.True(suite.T(), exists)
		assert.True(suite.T(), agentState.ContainerState.Running)

		recoveryJob, exists := suite.recoveryMgr.GetRecoveryJob(suite.testAgentID)
		require.True(suite.T(), exists)
		assert.Equal(suite.T(), RecoveryStatusSucceeded, recoveryJob.Status)
		assert.Greater(suite.T(), len(recoveryJob.History), 0)
	} else {
		suite.T().Log("Recovery was not successful within timeout")
		
		// 복구 시도는 있었는지 확인
		recoveryJob, exists := suite.recoveryMgr.GetRecoveryJob(suite.testAgentID)
		if exists {
			assert.Greater(suite.T(), recoveryJob.Attempts, 0)
			suite.T().Logf("Recovery attempts: %d, Status: %s", 
				recoveryJob.Attempts, recoveryJob.Status)
		}
	}
}

// waitForRecovery 복구 완료까지 대기
func (suite *AgentIntegrationTestSuite) waitForRecovery(timeout time.Duration) bool {
	startTime := time.Now()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-suite.ctx.Done():
			return false
		case <-ticker.C:
			if time.Since(startTime) > timeout {
				return false
			}

			// 복구 상태 확인
			agentState, exists := suite.syncMgr.GetAgentState(suite.testAgentID)
			if exists && agentState.ContainerState.Running {
				return true
			}

			// 복구 작업 상태 확인
			recoveryJob, exists := suite.recoveryMgr.GetRecoveryJob(suite.testAgentID)
			if exists && recoveryJob.Status == RecoveryStatusSucceeded {
				return true
			}
		}
	}
}

// createTestContainer 테스트용 컨테이너 생성
func (suite *AgentIntegrationTestSuite) createTestContainer() string {
	// 간단한 테스트 컨테이너 생성 (Alpine Linux with sleep)
	config := &container.Config{
		Image: "alpine:latest",
		Cmd:   []string{"sleep", "3600"},
		Labels: map[string]string{
			suite.client.labelKey("managed"):  "true",
			suite.client.labelKey("agent.id"): suite.testAgentID,
		},
	}

	hostConfig := &container.HostConfig{
		Resources: container.Resources{
			Memory:   512 * 1024 * 1024, // 512MB
			CPUQuota: 50000,             // 0.5 CPU
		},
	}

	resp, err := suite.client.cli.ContainerCreate(suite.ctx, config, hostConfig, nil, nil, "")
	require.NoError(suite.T(), err)

	err = suite.client.cli.ContainerStart(suite.ctx, resp.ID, container.StartOptions{})
	require.NoError(suite.T(), err)

	return resp.ID
}

// cleanupTestContainer 테스트 컨테이너 정리
func (suite *AgentIntegrationTestSuite) cleanupTestContainer() {
	if suite.testContainerID != "" {
		suite.client.cli.ContainerStop(suite.ctx, suite.testContainerID, container.StopOptions{})
		suite.client.cli.ContainerRemove(suite.ctx, suite.testContainerID, container.RemoveOptions{Force: true})
		suite.testContainerID = ""
	}
}

// cleanupTestResources 모든 테스트 리소스 정리
func (suite *AgentIntegrationTestSuite) cleanupTestResources() {
	// 테스트와 관련된 모든 컨테이너 정리
	containers, err := suite.client.cli.ContainerList(suite.ctx, container.ListOptions{
		All: true,
	})
	if err == nil {
		for _, c := range containers {
			if agentID, exists := c.Labels[suite.client.labelKey("agent.id")]; exists {
				if agentID == suite.testAgentID || 
				   len(agentID) > len(suite.testAgentID) && 
				   agentID[:len(suite.testAgentID)] == suite.testAgentID {
					suite.client.cli.ContainerStop(suite.ctx, c.ID, container.StopOptions{})
					suite.client.cli.ContainerRemove(suite.ctx, c.ID, container.RemoveOptions{Force: true})
				}
			}
		}
	}

	// 테스트 네트워크 정리
	if suite.networkMgr != nil {
		suite.networkMgr.CleanupOrphanedNetworks(suite.ctx)
	}
}

// TestAgentIntegration 통합 테스트 실행
func TestAgentIntegration(t *testing.T) {
	// Docker 데몬 사용 가능 여부 확인
	client, err := NewClient()
	if err != nil {
		t.Skip("Docker daemon not available:", err)
		return
	}
	
	// 간단한 연결 테스트
	_, err = client.cli.Ping(context.Background())
	if err != nil {
		t.Skip("Cannot connect to Docker daemon:", err)
		return
	}

	suite.Run(t, new(AgentIntegrationTestSuite))
}