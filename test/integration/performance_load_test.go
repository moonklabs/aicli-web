package integration

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/aicli/aicli-web/internal/agent"
	"github.com/aicli/aicli-web/internal/docker"
	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// PerformanceTestSuite 성능 및 부하 테스트 스위트
// T06_S01에서 설정한 성능 목표를 검증합니다.
type PerformanceTestSuite struct {
	suite.Suite
	ctx               context.Context
	cancel            context.CancelFunc
	storage          storage.Storage
	dockerClient     *docker.Client
	agentService     *agent.Service
	testAgents       []*models.Agent
	cleanupFunctions []func()
}

// PerformanceResult 성능 테스트 결과
type PerformanceResult struct {
	AgentID      string
	CreationTime time.Duration
	StartTime    time.Duration
	Success      bool
	Error        error
}

// LoadTestMetrics 부하 테스트 메트릭
type LoadTestMetrics struct {
	TotalAgents        int
	SuccessfulCreates  int
	SuccessfulStarts   int
	FailedCreates      int
	FailedStarts       int
	AvgCreationTime    time.Duration
	MaxCreationTime    time.Duration
	MinCreationTime    time.Duration
	P95CreationTime    time.Duration
	ConcurrentRunning  int
	TotalTestDuration  time.Duration
}

// SetupSuite 테스트 스위트 초기화
func (suite *PerformanceTestSuite) SetupSuite() {
	suite.ctx, suite.cancel = context.WithTimeout(context.Background(), 30*time.Minute)
	
	// 스토리지 초기화
	store, err := storage.New()
	require.NoError(suite.T(), err)
	suite.storage = store
	suite.addCleanupFunction(func() { store.Close() })

	// Docker 클라이언트 초기화
	dockerClient, err := docker.NewClient()
	require.NoError(suite.T(), err)
	suite.dockerClient = dockerClient

	// Agent 서비스 초기화
	suite.agentService = agent.NewService(
		suite.storage.Agent(),
		dockerClient,
		nil, // Git manager는 이 테스트에서 사용하지 않음
	)

	suite.T().Log("성능 테스트 환경 초기화 완료")
}

// TearDownSuite 테스트 스위트 정리
func (suite *PerformanceTestSuite) TearDownSuite() {
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
func (suite *PerformanceTestSuite) SetupTest() {
	suite.testAgents = make([]*models.Agent, 0)
}

// TearDownTest 각 테스트 정리
func (suite *PerformanceTestSuite) TearDownTest() {
	suite.cleanupTestAgents()
}

// TestAgentCreationPerformance 에이전트 생성 성능 테스트
// 목표: 에이전트 생성 시간 5초 이내 (P95 기준)
func (suite *PerformanceTestSuite) TestAgentCreationPerformance() {
	suite.T().Log("🚀 에이전트 생성 성능 테스트 시작")

	agentCount := 50
	results := make(chan PerformanceResult, agentCount)
	var wg sync.WaitGroup

	startTime := time.Now()

	// 동시에 여러 에이전트 생성
	suite.T().Logf("   📝 %d개 에이전트 동시 생성 성능 측정", agentCount)
	for i := 0; i < agentCount; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			
			result := PerformanceResult{
				AgentID: fmt.Sprintf("perf-test-%d", index),
			}

			agentSpec := &models.Agent{
				Name:        fmt.Sprintf("perf-test-agent-%d-%d", index, time.Now().Unix()),
				ProjectID:   "performance-test",
				AgentType:   models.AgentTypeStandard,
				Description: fmt.Sprintf("Performance test agent %d", index),
				Config: models.AgentConfig{
					Resources: models.ResourceConfig{
						CPU:    "0.5",
						Memory: "512Mi",
					},
				},
			}

			// 에이전트 생성 시간 측정
			createStart := time.Now()
			agent, err := suite.agentService.Create(suite.ctx, agentSpec)
			result.CreationTime = time.Since(createStart)

			if err != nil {
				result.Success = false
				result.Error = err
			} else {
				result.Success = true
				result.AgentID = agent.ID
				suite.testAgents = append(suite.testAgents, agent)
			}

			results <- result
		}(i)
	}

	wg.Wait()
	close(results)

	totalDuration := time.Since(startTime)

	// 결과 분석
	var creationTimes []time.Duration
	successCount := 0
	var totalCreationTime time.Duration

	for result := range results {
		creationTimes = append(creationTimes, result.CreationTime)
		totalCreationTime += result.CreationTime
		
		if result.Success {
			successCount++
		} else {
			suite.T().Logf("   ❌ 에이전트 생성 실패: %v", result.Error)
		}
	}

	// 성능 통계 계산
	avgCreationTime := totalCreationTime / time.Duration(len(creationTimes))
	p95CreationTime := suite.calculatePercentile(creationTimes, 95)
	maxCreationTime := suite.calculatePercentile(creationTimes, 100)
	minCreationTime := suite.calculatePercentile(creationTimes, 0)

	// 결과 출력
	suite.T().Logf("   📊 성능 테스트 결과:")
	suite.T().Logf("      성공률: %d/%d (%.1f%%)", successCount, agentCount, float64(successCount)/float64(agentCount)*100)
	suite.T().Logf("      평균 생성 시간: %v", avgCreationTime)
	suite.T().Logf("      P95 생성 시간: %v", p95CreationTime)
	suite.T().Logf("      최대 생성 시간: %v", maxCreationTime)
	suite.T().Logf("      최소 생성 시간: %v", minCreationTime)
	suite.T().Logf("      전체 테스트 시간: %v", totalDuration)

	// 성능 목표 검증
	assert.GreaterOrEqual(suite.T(), float64(successCount)/float64(agentCount), 0.8, "성공률이 80% 이상이어야 함")
	assert.Less(suite.T(), p95CreationTime, 5*time.Second, "P95 에이전트 생성 시간이 5초 이내여야 함")
	assert.Less(suite.T(), avgCreationTime, 3*time.Second, "평균 에이전트 생성 시간이 3초 이내여야 함")

	suite.T().Log("🎉 에이전트 생성 성능 테스트 완료")
}

// TestConcurrentAgentCapacity 동시 에이전트 지원 능력 테스트
// 목표: 100개 이상의 동시 에이전트 지원
func (suite *PerformanceTestSuite) TestConcurrentAgentCapacity() {
	suite.T().Log("🚀 동시 에이전트 지원 능력 테스트 시작")

	targetAgentCount := 100
	batchSize := 10 // 배치별로 생성하여 시스템 부하 조절
	totalBatches := targetAgentCount / batchSize

	var allAgents []*models.Agent
	var creationTimes []time.Duration
	successCount := 0

	suite.T().Logf("   📝 %d개 에이전트를 %d개 배치로 나누어 생성", targetAgentCount, totalBatches)

	startTime := time.Now()

	// 배치별로 에이전트 생성
	for batch := 0; batch < totalBatches; batch++ {
		suite.T().Logf("   🔄 배치 %d/%d 처리 중...", batch+1, totalBatches)
		
		var wg sync.WaitGroup
		batchResults := make(chan PerformanceResult, batchSize)
		
		// 배치 내 에이전트들 동시 생성
		for i := 0; i < batchSize; i++ {
			wg.Add(1)
			go func(batchIdx, agentIdx int) {
				defer wg.Done()
				
				agentSpec := &models.Agent{
					Name:        fmt.Sprintf("capacity-test-agent-%d-%d-%d", batchIdx, agentIdx, time.Now().Unix()),
					ProjectID:   "capacity-test",
					AgentType:   models.AgentTypeStandard,
					Description: fmt.Sprintf("Capacity test agent batch %d index %d", batchIdx, agentIdx),
					Config: models.AgentConfig{
						Resources: models.ResourceConfig{
							CPU:    "0.2", // 작은 리소스로 더 많은 에이전트 생성
							Memory: "256Mi",
						},
					},
				}

				createStart := time.Now()
				agent, err := suite.agentService.Create(suite.ctx, agentSpec)
				creationTime := time.Since(createStart)

				result := PerformanceResult{
					CreationTime: creationTime,
					Success:      err == nil,
					Error:        err,
				}

				if agent != nil {
					result.AgentID = agent.ID
					allAgents = append(allAgents, agent)
				}

				batchResults <- result
			}(batch, i)
		}

		wg.Wait()
		close(batchResults)

		// 배치 결과 처리
		batchSuccessCount := 0
		for result := range batchResults {
			creationTimes = append(creationTimes, result.CreationTime)
			if result.Success {
				batchSuccessCount++
				successCount++
			}
		}

		suite.T().Logf("   📊 배치 %d 결과: %d/%d 성공", batch+1, batchSuccessCount, batchSize)
		
		// 배치 간 잠시 대기 (시스템 안정화)
		time.Sleep(2 * time.Second)
	}

	totalTestDuration := time.Since(startTime)
	suite.testAgents = allAgents

	// 실제로 실행 중인 에이전트 수 확인 (시작 시도)
	suite.T().Log("   🚀 생성된 에이전트들 시작 시도...")
	
	runningCount := 0
	startFailures := 0
	
	// 처음 50개만 시작해보기 (리소스 절약)
	agentsToStart := 50
	if len(allAgents) < agentsToStart {
		agentsToStart = len(allAgents)
	}

	for i := 0; i < agentsToStart; i++ {
		agent := allAgents[i]
		err := suite.agentService.Start(suite.ctx, agent.ID)
		if err != nil {
			startFailures++
			continue
		}
		
		// 시작 대기 (최대 10초)
		started := suite.waitForAgentStart(agent.ID, 10*time.Second)
		if started {
			runningCount++
		}
	}

	// 성능 통계 계산
	avgCreationTime := time.Duration(0)
	if len(creationTimes) > 0 {
		var total time.Duration
		for _, t := range creationTimes {
			total += t
		}
		avgCreationTime = total / time.Duration(len(creationTimes))
	}

	p95CreationTime := suite.calculatePercentile(creationTimes, 95)

	// 결과 출력
	suite.T().Logf("   📊 동시 에이전트 지원 능력 테스트 결과:")
	suite.T().Logf("      총 생성 시도: %d개", targetAgentCount)
	suite.T().Logf("      성공적으로 생성: %d개", successCount)
	suite.T().Logf("      생성 성공률: %.1f%%", float64(successCount)/float64(targetAgentCount)*100)
	suite.T().Logf("      시작 시도: %d개", agentsToStart)
	suite.T().Logf("      실행 중인 에이전트: %d개", runningCount)
	suite.T().Logf("      시작 실패: %d개", startFailures)
	suite.T().Logf("      평균 생성 시간: %v", avgCreationTime)
	suite.T().Logf("      P95 생성 시간: %v", p95CreationTime)
	suite.T().Logf("      전체 테스트 시간: %v", totalTestDuration)

	// 목표 검증
	assert.GreaterOrEqual(suite.T(), successCount, 80, "최소 80개 이상의 에이전트가 생성되어야 함")
	assert.GreaterOrEqual(suite.T(), float64(successCount)/float64(targetAgentCount), 0.8, "생성 성공률이 80% 이상이어야 함")
	assert.GreaterOrEqual(suite.T(), runningCount, 30, "최소 30개 이상의 에이전트가 실행되어야 함")
	assert.Less(suite.T(), p95CreationTime, 5*time.Second, "P95 생성 시간이 5초 이내여야 함")

	suite.T().Log("🎉 동시 에이전트 지원 능력 테스트 완료")
}

// TestResourceUtilizationUnderLoad 부하 상황에서 리소스 사용률 테스트
func (suite *PerformanceTestSuite) TestResourceUtilizationUnderLoad() {
	suite.T().Log("🚀 부하 상황에서 리소스 사용률 테스트 시작")

	agentCount := 20
	var wg sync.WaitGroup
	results := make(chan PerformanceResult, agentCount)

	suite.T().Logf("   📊 %d개 에이전트 생성하여 리소스 사용률 모니터링", agentCount)

	// 에이전트들 생성 및 시작
	for i := 0; i < agentCount; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			agentSpec := &models.Agent{
				Name:        fmt.Sprintf("resource-test-agent-%d-%d", index, time.Now().Unix()),
				ProjectID:   "resource-test",
				AgentType:   models.AgentTypeStandard,
				Description: fmt.Sprintf("Resource test agent %d", index),
				Config: models.AgentConfig{
					Resources: models.ResourceConfig{
						CPU:    "0.3",
						Memory: "384Mi",
					},
				},
			}

			// 에이전트 생성
			createStart := time.Now()
			agent, err := suite.agentService.Create(suite.ctx, agentSpec)
			creationTime := time.Since(createStart)

			result := PerformanceResult{
				CreationTime: creationTime,
				Success:      err == nil,
				Error:        err,
			}

			if agent != nil {
				result.AgentID = agent.ID
				suite.testAgents = append(suite.testAgents, agent)

				// 에이전트 시작
				startTime := time.Now()
				err = suite.agentService.Start(suite.ctx, agent.ID)
				result.StartTime = time.Since(startTime)
				
				if err != nil {
					result.Success = false
					result.Error = err
				}
			}

			results <- result
		}(i)
	}

	wg.Wait()
	close(results)

	// 결과 분석
	successfulStarts := 0
	var totalCreationTime, totalStartTime time.Duration
	
	for result := range results {
		totalCreationTime += result.CreationTime
		totalStartTime += result.StartTime
		
		if result.Success {
			successfulStarts++
		}
	}

	// 일정 시간 대기 후 시스템 리소스 상태 확인
	suite.T().Log("   ⏳ 시스템 안정화 대기 (30초)...")
	time.Sleep(30 * time.Second)

	// 실행 중인 에이전트 수 확인
	runningAgents := 0
	for _, agent := range suite.testAgents {
		currentAgent, err := suite.agentService.Get(suite.ctx, agent.ID)
		if err == nil && currentAgent.Status == models.AgentStatusRunning {
			runningAgents++
		}
	}

	// 메트릭 수집 (가능한 경우)
	suite.T().Log("   📈 시스템 메트릭 수집...")
	
	// Docker 시스템 정보 확인
	dockerInfo, err := suite.dockerClient.GetSystemInfo(suite.ctx)
	var containerCount int
	if err == nil && dockerInfo != nil {
		containerCount = dockerInfo.ContainersRunning
	}

	avgCreationTime := totalCreationTime / time.Duration(agentCount)
	avgStartTime := totalStartTime / time.Duration(agentCount)

	// 결과 출력
	suite.T().Logf("   📊 리소스 사용률 테스트 결과:")
	suite.T().Logf("      에이전트 생성 성공: %d/%d", successfulStarts, agentCount)
	suite.T().Logf("      현재 실행 중인 에이전트: %d개", runningAgents)
	suite.T().Logf("      총 실행 중인 컨테이너: %d개", containerCount)
	suite.T().Logf("      평균 생성 시간: %v", avgCreationTime)
	suite.T().Logf("      평균 시작 시간: %v", avgStartTime)

	// 리소스 효율성 검증
	assert.GreaterOrEqual(suite.T(), successfulStarts, agentCount*8/10, "80% 이상의 에이전트가 성공적으로 시작되어야 함")
	assert.GreaterOrEqual(suite.T(), runningAgents, agentCount*6/10, "60% 이상의 에이전트가 실행 상태를 유지해야 함")
	assert.Less(suite.T(), avgCreationTime, 4*time.Second, "평균 생성 시간이 4초 이내여야 함")
	assert.Less(suite.T(), avgStartTime, 10*time.Second, "평균 시작 시간이 10초 이내여야 함")

	suite.T().Log("🎉 부하 상황에서 리소스 사용률 테스트 완료")
}

// TestSystemStabilityUnderLoad 부하 상황에서 시스템 안정성 테스트
func (suite *PerformanceTestSuite) TestSystemStabilityUnderLoad() {
	suite.T().Log("🚀 부하 상황에서 시스템 안정성 테스트 시작")

	testDuration := 5 * time.Minute
	agentBatchSize := 5
	batchInterval := 30 * time.Second

	suite.T().Logf("   ⏱️  %v 동안 %v 간격으로 %d개씩 에이전트 생성/삭제 반복", 
		testDuration, batchInterval, agentBatchSize)

	startTime := time.Now()
	totalCreated := 0
	totalDeleted := 0
	errors := 0

	for time.Since(startTime) < testDuration {
		batchStart := time.Now()
		
		// 에이전트 배치 생성
		createdInBatch := 0
		for i := 0; i < agentBatchSize; i++ {
			agentSpec := &models.Agent{
				Name:        fmt.Sprintf("stability-test-%d-%d", totalCreated+i, time.Now().Unix()),
				ProjectID:   "stability-test",
				AgentType:   models.AgentTypeStandard,
				Description: "Stability test agent",
				Config: models.AgentConfig{
					Resources: models.ResourceConfig{
						CPU:    "0.1",
						Memory: "128Mi",
					},
				},
			}

			agent, err := suite.agentService.Create(suite.ctx, agentSpec)
			if err != nil {
				errors++
				continue
			}

			suite.testAgents = append(suite.testAgents, agent)
			createdInBatch++
		}

		totalCreated += createdInBatch

		// 일부 이전 에이전트들 삭제 (메모리 절약)
		if len(suite.testAgents) > 15 {
			agentsToDelete := suite.testAgents[:5]
			suite.testAgents = suite.testAgents[5:]

			for _, agent := range agentsToDelete {
				err := suite.agentService.Delete(suite.ctx, agent.ID)
				if err == nil {
					totalDeleted++
				} else {
					errors++
				}
			}
		}

		batchDuration := time.Since(batchStart)
		
		// 현재 에이전트 수 확인
		activeAgents := len(suite.testAgents)
		
		suite.T().Logf("   📊 배치 완료 - 생성: %d, 삭제: %d, 활성: %d, 소요시간: %v", 
			createdInBatch, 5, activeAgents, batchDuration)

		// 다음 배치까지 대기
		time.Sleep(batchInterval)
	}

	totalTestDuration := time.Since(startTime)

	// 최종 정리
	remainingAgents := len(suite.testAgents)
	for _, agent := range suite.testAgents {
		err := suite.agentService.Delete(suite.ctx, agent.ID)
		if err == nil {
			totalDeleted++
		}
	}
	suite.testAgents = suite.testAgents[:0]

	// 결과 출력
	suite.T().Logf("   📊 시스템 안정성 테스트 결과:")
	suite.T().Logf("      테스트 기간: %v", totalTestDuration)
	suite.T().Logf("      총 생성된 에이전트: %d개", totalCreated)
	suite.T().Logf("      총 삭제된 에이전트: %d개", totalDeleted)
	suite.T().Logf("      마지막 활성 에이전트: %d개", remainingAgents)
	suite.T().Logf("      총 에러 수: %d개", errors)
	suite.T().Logf("      에러율: %.2f%%", float64(errors)/float64(totalCreated+totalDeleted)*100)

	// 안정성 검증
	errorRate := float64(errors) / float64(totalCreated+totalDeleted)
	assert.Less(suite.T(), errorRate, 0.1, "에러율이 10% 미만이어야 함")
	assert.Greater(suite.T(), totalCreated, 20, "최소 20개 이상의 에이전트가 생성되어야 함")

	suite.T().Log("🎉 부하 상황에서 시스템 안정성 테스트 완료")
}

// 헬퍼 메서드들

// calculatePercentile 백분위수 계산
func (suite *PerformanceTestSuite) calculatePercentile(durations []time.Duration, percentile int) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	// 정렬
	for i := 0; i < len(durations)-1; i++ {
		for j := i + 1; j < len(durations); j++ {
			if durations[i] > durations[j] {
				durations[i], durations[j] = durations[j], durations[i]
			}
		}
	}

	if percentile == 0 {
		return durations[0]
	}
	if percentile == 100 {
		return durations[len(durations)-1]
	}

	index := int(float64(len(durations)) * float64(percentile) / 100.0)
	if index >= len(durations) {
		index = len(durations) - 1
	}

	return durations[index]
}

// waitForAgentStart 에이전트 시작까지 대기
func (suite *PerformanceTestSuite) waitForAgentStart(agentID string, timeout time.Duration) bool {
	startTime := time.Now()
	ticker := time.NewTicker(1 * time.Second)
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

			if agent.Status == models.AgentStatusRunning {
				return true
			}
		}
	}
}

// cleanupTestAgents 테스트 에이전트들 정리
func (suite *PerformanceTestSuite) cleanupTestAgents() {
	for _, agent := range suite.testAgents {
		if agent.Status == models.AgentStatusRunning {
			suite.agentService.Stop(suite.ctx, agent.ID)
		}
		suite.agentService.Delete(suite.ctx, agent.ID)
	}
	suite.testAgents = suite.testAgents[:0]
}

// addCleanupFunction 정리 함수 추가
func (suite *PerformanceTestSuite) addCleanupFunction(fn func()) {
	suite.cleanupFunctions = append(suite.cleanupFunctions, fn)
}

// TestPerformanceSuite 성능 테스트 실행
func TestPerformanceSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("성능 테스트는 short 모드에서 제외됩니다")
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

	t.Log("⚡ 성능 테스트 스위트 시작 - 시간이 오래 걸릴 수 있습니다")
	suite.Run(t, new(PerformanceTestSuite))
}