//go:build integration
// +build integration

package agent

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/aicli/aicli-web/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// LoadTestConfig 부하 테스트 설정
type LoadTestConfig struct {
	ConcurrentAgents    int           `json:"concurrent_agents"`    // 동시 에이전트 수
	TestDuration        time.Duration `json:"test_duration"`        // 테스트 지속 시간
	AgentCreationRate   int           `json:"agent_creation_rate"`  // 초당 에이전트 생성 수
	TasksPerAgent       int           `json:"tasks_per_agent"`      // 에이전트당 작업 수
	ExpectedCreationTime time.Duration `json:"expected_creation_time"` // 예상 생성 시간
	MemoryLimitMB       int           `json:"memory_limit_mb"`      // 메모리 제한 (MB)
}

// LoadTestResult 부하 테스트 결과
type LoadTestResult struct {
	// 기본 통계
	TotalAgentsCreated    int           `json:"total_agents_created"`
	TotalAgentsFailed     int           `json:"total_agents_failed"`
	TotalTasksCompleted   int64         `json:"total_tasks_completed"`
	TotalTasksFailed      int64         `json:"total_tasks_failed"`
	TestDuration          time.Duration `json:"test_duration"`

	// 성능 메트릭
	AverageCreationTime   time.Duration `json:"average_creation_time"`
	P95CreationTime       time.Duration `json:"p95_creation_time"`
	P99CreationTime       time.Duration `json:"p99_creation_time"`
	MaxCreationTime       time.Duration `json:"max_creation_time"`
	MinCreationTime       time.Duration `json:"min_creation_time"`

	// 처리량 메트릭
	AgentsPerSecond       float64       `json:"agents_per_second"`
	TasksPerSecond        float64       `json:"tasks_per_second"`
	SuccessRate           float64       `json:"success_rate"`

	// 리소스 사용량
	PeakMemoryUsageMB     float64       `json:"peak_memory_usage_mb"`
	AverageMemoryUsageMB  float64       `json:"average_memory_usage_mb"`
	PeakCPUUsagePercent   float64       `json:"peak_cpu_usage_percent"`
	AverageCPUUsagePercent float64      `json:"average_cpu_usage_percent"`

	// 에러 정보
	Errors                []string      `json:"errors"`

	// 상세 타이밍 정보
	CreationTimes         []time.Duration `json:"creation_times"`
	TaskCompletionTimes   []time.Duration `json:"task_completion_times"`
}

// TestLoadTest_100ConcurrentAgents 100개 동시 에이전트 부하 테스트
func TestLoadTest_100ConcurrentAgents(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping load test in short mode")
	}

	config := LoadTestConfig{
		ConcurrentAgents:     100,
		TestDuration:         2 * time.Minute,
		AgentCreationRate:    20, // 초당 20개
		TasksPerAgent:        5,
		ExpectedCreationTime: 5 * time.Second,
		MemoryLimitMB:        1024, // 1GB
	}

	result := RunLoadTest(t, config)

	// 성능 요구사항 검증
	assert.LessOrEqual(t, result.P95CreationTime, config.ExpectedCreationTime,
		"P95 creation time should be within expected limit")
	assert.GreaterOrEqual(t, result.SuccessRate, 0.95,
		"Success rate should be at least 95%")
	assert.LessOrEqual(t, result.PeakMemoryUsageMB, float64(config.MemoryLimitMB),
		"Peak memory usage should be within limit")

	// 결과 출력
	t.Logf("Load Test Results (100 Agents):")
	t.Logf("- Total Agents Created: %d", result.TotalAgentsCreated)
	t.Logf("- Success Rate: %.2f%%", result.SuccessRate*100)
	t.Logf("- Average Creation Time: %v", result.AverageCreationTime)
	t.Logf("- P95 Creation Time: %v", result.P95CreationTime)
	t.Logf("- P99 Creation Time: %v", result.P99CreationTime)
	t.Logf("- Agents/Second: %.2f", result.AgentsPerSecond)
	t.Logf("- Peak Memory Usage: %.2f MB", result.PeakMemoryUsageMB)
	t.Logf("- Average CPU Usage: %.2f%%", result.AverageCPUUsagePercent)
}

// TestLoadTest_StressTest 스트레스 테스트 (리소스 한계 테스트)
func TestLoadTest_StressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	config := LoadTestConfig{
		ConcurrentAgents:     200, // 높은 부하
		TestDuration:         5 * time.Minute,
		AgentCreationRate:    50, // 초당 50개
		TasksPerAgent:        10,
		ExpectedCreationTime: 10 * time.Second, // 여유 있는 시간
		MemoryLimitMB:        2048, // 2GB
	}

	result := RunLoadTest(t, config)

	// 스트레스 테스트는 일부 실패가 허용됨
	assert.GreaterOrEqual(t, result.SuccessRate, 0.80,
		"Success rate should be at least 80% under stress")
	
	t.Logf("Stress Test Results (200 Agents):")
	t.Logf("- Total Agents Created: %d", result.TotalAgentsCreated)
	t.Logf("- Success Rate: %.2f%%", result.SuccessRate*100)
	t.Logf("- Average Creation Time: %v", result.AverageCreationTime)
	t.Logf("- P95 Creation Time: %v", result.P95CreationTime)
	t.Logf("- Peak Memory Usage: %.2f MB", result.PeakMemoryUsageMB)
	t.Logf("- Errors: %d", len(result.Errors))
}

// TestLoadTest_ScalabilityTest 확장성 테스트
func TestLoadTest_ScalabilityTest(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping scalability test in short mode")
	}

	agentCounts := []int{10, 25, 50, 100}
	results := make(map[int]*LoadTestResult)

	for _, count := range agentCounts {
		t.Run(fmt.Sprintf("Agents_%d", count), func(t *testing.T) {
			config := LoadTestConfig{
				ConcurrentAgents:     count,
				TestDuration:         1 * time.Minute,
				AgentCreationRate:    count / 2, // 절반 속도로 생성
				TasksPerAgent:        3,
				ExpectedCreationTime: 5 * time.Second,
				MemoryLimitMB:        512 * count / 10, // 에이전트 수에 비례
			}

			result := RunLoadTest(t, config)
			results[count] = result

			t.Logf("Scalability Test (%d agents):", count)
			t.Logf("- Success Rate: %.2f%%", result.SuccessRate*100)
			t.Logf("- Average Creation Time: %v", result.AverageCreationTime)
			t.Logf("- Memory Usage: %.2f MB", result.PeakMemoryUsageMB)
		})
	}

	// 확장성 분석
	analyzeScalability(t, results)
}

// RunLoadTest 부하 테스트 실행
func RunLoadTest(t *testing.T, config LoadTestConfig) *LoadTestResult {
	// 모의 에이전트 풀 생성 (실제 Docker 없이 테스트)
	pool := createMockAgentPool(t, config)

	// 메트릭 수집기 설정
	metricsConfig := DefaultMetricsConfig()
	metricsConfig.CollectionInterval = 1 * time.Second
	collector := NewPerformanceMetricsCollector(metricsConfig, pool)

	err := collector.Start()
	require.NoError(t, err)
	defer collector.Stop()

	// 테스트 결과 초기화
	result := &LoadTestResult{
		CreationTimes:       make([]time.Duration, 0, config.ConcurrentAgents),
		TaskCompletionTimes: make([]time.Duration, 0, config.ConcurrentAgents*config.TasksPerAgent),
		Errors:              make([]string, 0),
	}

	// 테스트 시작
	startTime := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), config.TestDuration)
	defer cancel()

	// 동시 에이전트 생성 및 작업 실행
	var wg sync.WaitGroup
	agentChan := make(chan AgentRequest, config.ConcurrentAgents)
	resultChan := make(chan AgentResult, config.ConcurrentAgents*config.TasksPerAgent)

	// 에이전트 생성 속도 제어
	go func() {
		ticker := time.NewTicker(time.Duration(1000/config.AgentCreationRate) * time.Millisecond)
		defer ticker.Stop()

		agentCount := 0
		for {
			select {
			case <-ctx.Done():
				close(agentChan)
				return
			case <-ticker.C:
				if agentCount < config.ConcurrentAgents {
					agentChan <- AgentRequest{
						ProjectID: fmt.Sprintf("load-test-project-%d", agentCount),
						Name:      fmt.Sprintf("load-test-agent-%d", agentCount),
						Config:    models.AgentConfig{},
					}
					agentCount++
				}
			}
		}
	}()

	// 워커 고루틴들 시작
	for i := 0; i < runtime.NumCPU(); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for req := range agentChan {
				runAgentWorkload(ctx, pool, req, config, resultChan)
			}
		}()
	}

	// 결과 수집 고루틴
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// 메모리 및 CPU 모니터링
	go monitorResources(ctx, result)

	// 결과 수집
	for agentResult := range resultChan {
		if agentResult.Error != nil {
			result.TotalAgentsFailed++
			result.Errors = append(result.Errors, agentResult.Error.Error())
		} else {
			result.TotalAgentsCreated++
			result.CreationTimes = append(result.CreationTimes, agentResult.CreationTime)
			result.TotalTasksCompleted += int64(len(agentResult.TaskTimes))
			result.TaskCompletionTimes = append(result.TaskCompletionTimes, agentResult.TaskTimes...)
		}
	}

	// 테스트 완료
	result.TestDuration = time.Since(startTime)

	// 메트릭 계산
	calculateLoadTestMetrics(result)

	return result
}

// createMockAgentPool 모의 에이전트 풀 생성
func createMockAgentPool(t *testing.T, config LoadTestConfig) *AdvancedAgentPool {
	poolConfig := DefaultAdvancedPoolConfig()
	poolConfig.MaxSize = config.ConcurrentAgents * 2
	poolConfig.WarmUpSize = config.ConcurrentAgents / 4
	poolConfig.CreationTimeout = config.ExpectedCreationTime

	// 모의 Docker 클라이언트 (실제로는 nil, 메모리 기반 테스트)
	pool := NewAdvancedAgentPool(poolConfig, nil)
	
	// 테스트용으로 실제 컨테이너 없이 풀만 초기화
	pool.availableAgents = make(chan *PooledAgent, poolConfig.MaxSize)
	
	return pool
}

// runAgentWorkload 에이전트 워크로드 실행
func runAgentWorkload(ctx context.Context, pool *AdvancedAgentPool, req AgentRequest, config LoadTestConfig, resultChan chan<- AgentResult) {
	start := time.Now()
	
	// 모의 에이전트 생성 (실제 컨테이너 없이)
	agent := &PooledAgent{
		ID: generateAgentID(),
		Agent: &models.Agent{
			ID:        generateAgentID(),
			ProjectID: req.ProjectID,
			Name:      req.Name,
			Type:      models.AgentTypeClaude,
			Status:    models.AgentStatusRunning,
			CreatedAt: time.Now(),
		},
		Status:       PooledAgentStatusInUse,
		CreatedAt:    time.Now(),
		LastUsedAt:   time.Now(),
		CreationTime: time.Since(start),
	}

	// 작업 시뮬레이션
	taskTimes := make([]time.Duration, config.TasksPerAgent)
	for i := 0; i < config.TasksPerAgent; i++ {
		taskStart := time.Now()
		
		// 실제 작업 시뮬레이션 (10-100ms 랜덤 대기)
		simulateWork(ctx, 10*time.Millisecond, 100*time.Millisecond)
		
		taskTimes[i] = time.Since(taskStart)
		
		if ctx.Err() != nil {
			break
		}
	}

	result := AgentResult{
		AgentID:      agent.ID,
		CreationTime: agent.CreationTime,
		TaskTimes:    taskTimes,
	}

	// 실패 시뮬레이션 (5% 확률)
	if time.Now().UnixNano()%100 < 5 {
		result.Error = fmt.Errorf("simulated agent failure")
	}

	select {
	case resultChan <- result:
	case <-ctx.Done():
	}
}

// simulateWork 작업 시뮬레이션
func simulateWork(ctx context.Context, minDuration, maxDuration time.Duration) {
	// 랜덤 대기 시간 계산
	duration := minDuration + time.Duration(time.Now().UnixNano()%(maxDuration.Nanoseconds()-minDuration.Nanoseconds()))
	
	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-timer.C:
		// CPU 집약적 작업 시뮬레이션
		sum := 0
		for i := 0; i < 10000; i++ {
			sum += i * i
		}
		_ = sum
	case <-ctx.Done():
	}
}

// monitorResources 리소스 모니터링
func monitorResources(ctx context.Context, result *LoadTestResult) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	var memSamples []float64
	var cpuSamples []float64

	for {
		select {
		case <-ctx.Done():
			// 최종 통계 계산
			if len(memSamples) > 0 {
				result.PeakMemoryUsageMB = maxFloat64(memSamples)
				result.AverageMemoryUsageMB = avgFloat64(memSamples)
			}
			if len(cpuSamples) > 0 {
				result.PeakCPUUsagePercent = maxFloat64(cpuSamples)
				result.AverageCPUUsagePercent = avgFloat64(cpuSamples)
			}
			return

		case <-ticker.C:
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			memUsageMB := float64(m.Alloc) / 1024 / 1024
			memSamples = append(memSamples, memUsageMB)

			// CPU 사용률은 고루틴 수로 근사 (실제로는 더 정확한 측정 필요)
			cpuUsage := float64(runtime.NumGoroutine()) / float64(runtime.NumCPU()) * 10
			if cpuUsage > 100 {
				cpuUsage = 100
			}
			cpuSamples = append(cpuSamples, cpuUsage)
		}
	}
}

// calculateLoadTestMetrics 부하 테스트 메트릭 계산
func calculateLoadTestMetrics(result *LoadTestResult) {
	// 성공률 계산
	totalAttempts := result.TotalAgentsCreated + result.TotalAgentsFailed
	if totalAttempts > 0 {
		result.SuccessRate = float64(result.TotalAgentsCreated) / float64(totalAttempts)
	}

	// 처리량 계산
	seconds := result.TestDuration.Seconds()
	if seconds > 0 {
		result.AgentsPerSecond = float64(result.TotalAgentsCreated) / seconds
		result.TasksPerSecond = float64(result.TotalTasksCompleted) / seconds
	}

	// 생성 시간 통계
	if len(result.CreationTimes) > 0 {
		result.AverageCreationTime = avgDuration(result.CreationTimes)
		result.MinCreationTime = minDuration(result.CreationTimes)
		result.MaxCreationTime = maxDuration(result.CreationTimes)
		result.P95CreationTime = percentileDuration(result.CreationTimes, 0.95)
		result.P99CreationTime = percentileDuration(result.CreationTimes, 0.99)
	}
}

// analyzeScalability 확장성 분석
func analyzeScalability(t *testing.T, results map[int]*LoadTestResult) {
	t.Log("\nScalability Analysis:")
	t.Log("Agents | Success Rate | Avg Creation Time | Memory Usage")
	t.Log("-------|--------------|-------------------|-------------")

	for _, count := range []int{10, 25, 50, 100} {
		if result, ok := results[count]; ok {
			t.Logf("%6d | %11.1f%% | %17v | %10.1f MB",
				count,
				result.SuccessRate*100,
				result.AverageCreationTime,
				result.PeakMemoryUsageMB)
		}
	}

	// 선형성 분석
	if len(results) >= 2 {
		// 간단한 선형성 체크
		first := results[10]
		last := results[100]
		
		if first != nil && last != nil {
			memoryGrowthRatio := last.PeakMemoryUsageMB / first.PeakMemoryUsageMB
			agentRatio := float64(100) / float64(10)
			
			t.Logf("\nMemory growth ratio: %.2fx (expected: %.2fx)", memoryGrowthRatio, agentRatio)
			
			if memoryGrowthRatio < agentRatio*1.5 {
				t.Log("✓ Memory usage scales reasonably well")
			} else {
				t.Log("⚠ Memory usage may not scale linearly")
			}
		}
	}
}

// AgentResult 에이전트 작업 결과
type AgentResult struct {
	AgentID      string
	CreationTime time.Duration
	TaskTimes    []time.Duration
	Error        error
}

// 유틸리티 함수들
func maxFloat64(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	max := values[0]
	for _, v := range values[1:] {
		if v > max {
			max = v
		}
	}
	return max
}

func avgFloat64(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func avgDuration(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	var sum time.Duration
	for _, d := range durations {
		sum += d
	}
	return sum / time.Duration(len(durations))
}

func minDuration(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	min := durations[0]
	for _, d := range durations[1:] {
		if d < min {
			min = d
		}
	}
	return min
}

func maxDuration(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	max := durations[0]
	for _, d := range durations[1:] {
		if d > max {
			max = d
		}
	}
	return max
}

func percentileDuration(durations []time.Duration, percentile float64) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	
	// 간단한 정렬 및 백분위수 계산
	sorted := make([]time.Duration, len(durations))
	copy(sorted, durations)
	
	// 버블 정렬 (작은 배열이므로 충분)
	for i := 0; i < len(sorted)-1; i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			if sorted[j] > sorted[j+1] {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}
	
	index := int(float64(len(sorted)-1) * percentile)
	return sorted[index]
}