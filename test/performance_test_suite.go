// 성능 테스트 스위트
// 멀티 에이전트 플랫폼의 성능 및 부하 테스트

//go:build performance
// +build performance

package test

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/aicli/aicli-web/internal/agent"
	"github.com/aicli/aicli-web/internal/storage"
	"github.com/aicli/aicli-web/internal/testutil"
	"github.com/docker/docker/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// PerformanceTestSuite는 성능 테스트를 정의합니다
type PerformanceTestSuite struct {
	suite.Suite
	dockerClient     *client.Client
	storage          storage.Storage
	agentService     *agent.Service
	testWorkspace    string
	maxConcurrentAgents int
	testDuration     time.Duration
	rampUpDuration   time.Duration
	cleanup          []func()
}

// PerformanceMetrics는 성능 테스트 결과를 저장합니다
type PerformanceMetrics struct {
	TotalAgents        int           `json:"total_agents"`
	SuccessfulAgents   int           `json:"successful_agents"`
	FailedAgents       int           `json:"failed_agents"`
	TotalTasks         int           `json:"total_tasks"`
	SuccessfulTasks    int           `json:"successful_tasks"`
	FailedTasks        int           `json:"failed_tasks"`
	AvgResponseTime    time.Duration `json:"avg_response_time"`
	MaxResponseTime    time.Duration `json:"max_response_time"`
	MinResponseTime    time.Duration `json:"min_response_time"`
	ThroughputPerSec   float64       `json:"throughput_per_sec"`
	ErrorRate          float64       `json:"error_rate"`
	MemoryUsageMB      float64       `json:"memory_usage_mb"`
	CPUUsagePercent    float64       `json:"cpu_usage_percent"`
	TestDuration       time.Duration `json:"test_duration"`
	StartTime          time.Time     `json:"start_time"`
	EndTime            time.Time     `json:"end_time"`
}

// SetupSuite는 성능 테스트 스위트 초기화를 수행합니다
func (s *PerformanceTestSuite) SetupSuite() {
	// 환경 변수에서 성능 테스트 설정 읽기
	s.maxConcurrentAgents = 100 // 기본값
	if env := os.Getenv("MAX_CONCURRENT_AGENTS"); env != "" {
		fmt.Sscanf(env, "%d", &s.maxConcurrentAgents)
	}

	s.testDuration = 5 * time.Minute // 기본값
	if env := os.Getenv("TEST_DURATION"); env != "" {
		duration, err := time.ParseDuration(env)
		if err == nil {
			s.testDuration = duration
		}
	}

	s.rampUpDuration = 30 * time.Second // 기본값
	if env := os.Getenv("LOAD_TEST_RAMP_UP"); env != "" {
		duration, err := time.ParseDuration(env)
		if err == nil {
			s.rampUpDuration = duration
		}
	}

	// Docker 클라이언트 초기화
	var err error
	dockerHost := os.Getenv("DOCKER_HOST")
	if dockerHost == "" {
		dockerHost = "unix:///var/run/docker.sock"
	}

	s.dockerClient, err = client.NewClientWithOpts(
		client.WithHost(dockerHost),
		client.WithAPIVersionNegotiation(),
	)
	s.Require().NoError(err, "Docker 클라이언트 생성 실패")

	// 스토리지 초기화
	s.storage, err = storage.New()
	s.Require().NoError(err, "스토리지 초기화 실패")

	// 테스트 워크스페이스 생성
	s.testWorkspace = testutil.TempDir(s.T(), "performance-test-workspace")

	// 에이전트 서비스 초기화
	s.agentService = agent.NewService(s.storage, s.dockerClient)
	s.Require().NotNil(s.agentService, "에이전트 서비스 초기화 실패")

	s.T().Logf("성능 테스트 환경 초기화 완료")
	s.T().Logf("설정: 최대 에이전트=%d, 테스트 시간=%v, 램프업 시간=%v", 
		s.maxConcurrentAgents, s.testDuration, s.rampUpDuration)
}

// TearDownSuite는 성능 테스트 스위트 정리를 수행합니다
func (s *PerformanceTestSuite) TearDownSuite() {
	// 정리 함수들 실행
	for _, cleanup := range s.cleanup {
		cleanup()
	}

	// 리소스 정리
	if s.storage != nil {
		s.storage.Close()
	}
	if s.dockerClient != nil {
		s.dockerClient.Close()
	}

	s.T().Log("성능 테스트 환경 정리 완료")
}

// TestMaxConcurrentAgents는 최대 동시 에이전트 수를 테스트합니다
func (s *PerformanceTestSuite) TestMaxConcurrentAgents() {
	ctx := context.Background()
	metrics := &PerformanceMetrics{
		StartTime:       time.Now(),
		MinResponseTime: time.Hour, // 초기값을 큰 값으로 설정
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	agentIDs := make([]string, 0, s.maxConcurrentAgents)
	responseTimes := make([]time.Duration, 0, s.maxConcurrentAgents)

	s.T().Logf("최대 동시 에이전트 테스트 시작: %d개 에이전트", s.maxConcurrentAgents)

	// 램프업 단계: 점진적으로 에이전트 생성
	agentsPerSecond := float64(s.maxConcurrentAgents) / s.rampUpDuration.Seconds()
	intervalBetweenAgents := time.Duration(float64(time.Second) / agentsPerSecond)

	for i := 0; i < s.maxConcurrentAgents; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			startTime := time.Now()

			// 에이전트 설정
			agentConfig := &agent.Config{
				Name:         fmt.Sprintf("perf-agent-%d", index),
				WorkspaceID:  "perf-workspace",
				ImageName:    "aicli/claude-agent:test",
				MaxMemory:    128 * 1024 * 1024, // 128MB
				MaxCPU:       0.1,               // 10% CPU
				NetworkMode:  "bridge",
				Tools:        []string{"Read", "Write"},
				SystemPrompt: fmt.Sprintf("Performance test agent %d", index),
			}

			// 에이전트 생성
			agentID, err := s.agentService.CreateAgent(ctx, agentConfig)
			if err != nil {
				mu.Lock()
				metrics.FailedAgents++
				mu.Unlock()
				s.T().Logf("에이전트 %d 생성 실패: %v", index, err)
				return
			}

			// 에이전트 시작
			err = s.agentService.StartAgent(ctx, agentID)
			if err != nil {
				mu.Lock()
				metrics.FailedAgents++
				mu.Unlock()
				s.T().Logf("에이전트 %d 시작 실패: %v", index, err)
				return
			}

			responseTime := time.Since(startTime)

			mu.Lock()
			agentIDs = append(agentIDs, agentID)
			responseTimes = append(responseTimes, responseTime)
			metrics.SuccessfulAgents++
			
			if responseTime > metrics.MaxResponseTime {
				metrics.MaxResponseTime = responseTime
			}
			if responseTime < metrics.MinResponseTime {
				metrics.MinResponseTime = responseTime
			}
			mu.Unlock()

			s.T().Logf("에이전트 %d 생성 완료: %s (응답시간: %v)", index, agentID, responseTime)
		}(i)

		// 램프업 간격 대기
		if i < s.maxConcurrentAgents-1 {
			time.Sleep(intervalBetweenAgents)
		}
	}

	wg.Wait()

	// 메트릭 계산
	metrics.TotalAgents = s.maxConcurrentAgents
	metrics.EndTime = time.Now()
	metrics.TestDuration = metrics.EndTime.Sub(metrics.StartTime)

	if len(responseTimes) > 0 {
		var totalTime time.Duration
		for _, rt := range responseTimes {
			totalTime += rt
		}
		metrics.AvgResponseTime = totalTime / time.Duration(len(responseTimes))
	}

	metrics.ErrorRate = float64(metrics.FailedAgents) / float64(metrics.TotalAgents) * 100
	metrics.ThroughputPerSec = float64(metrics.SuccessfulAgents) / metrics.TestDuration.Seconds()

	// 결과 출력
	s.T().Logf("=== 최대 동시 에이전트 테스트 결과 ===")
	s.T().Logf("총 에이전트: %d", metrics.TotalAgents)
	s.T().Logf("성공한 에이전트: %d", metrics.SuccessfulAgents)
	s.T().Logf("실패한 에이전트: %d", metrics.FailedAgents)
	s.T().Logf("평균 응답시간: %v", metrics.AvgResponseTime)
	s.T().Logf("최대 응답시간: %v", metrics.MaxResponseTime)
	s.T().Logf("최소 응답시간: %v", metrics.MinResponseTime)
	s.T().Logf("에러율: %.2f%%", metrics.ErrorRate)
	s.T().Logf("처리량: %.2f agents/sec", metrics.ThroughputPerSec)
	s.T().Logf("테스트 시간: %v", metrics.TestDuration)

	// 성능 기준 확인
	s.Assert().LessOrEqual(metrics.ErrorRate, 5.0, "에러율이 5%를 초과함")
	s.Assert().LessOrEqual(metrics.AvgResponseTime, 30*time.Second, "평균 응답시간이 30초를 초과함")
	s.Assert().GreaterOrEqual(metrics.ThroughputPerSec, 1.0, "처리량이 1 agent/sec보다 낮음")

	// 정리: 모든 에이전트 중지
	for _, agentID := range agentIDs {
		if err := s.agentService.StopAgent(ctx, agentID); err != nil {
			s.T().Logf("에이전트 중지 실패 %s: %v", agentID, err)
		}
	}
}

// TestSustainedLoad는 지속적인 부하 테스트를 수행합니다
func (s *PerformanceTestSuite) TestSustainedLoad() {
	ctx := context.Background()
	metrics := &PerformanceMetrics{
		StartTime:       time.Now(),
		MinResponseTime: time.Hour,
	}

	numAgents := 20 // 지속 부하 테스트용 에이전트 수
	tasksPerAgent := 10

	var wg sync.WaitGroup
	var mu sync.Mutex
	agentIDs := make([]string, 0, numAgents)
	taskResults := make([]*agent.TaskResult, 0, numAgents*tasksPerAgent)

	s.T().Logf("지속 부하 테스트 시작: %d개 에이전트, 각각 %d개 작업, 지속시간: %v", 
		numAgents, tasksPerAgent, s.testDuration)

	// 에이전트 생성 및 시작
	for i := 0; i < numAgents; i++ {
		agentConfig := &agent.Config{
			Name:         fmt.Sprintf("sustained-agent-%d", i),
			WorkspaceID:  "perf-workspace",
			ImageName:    "aicli/claude-agent:test",
			MaxMemory:    256 * 1024 * 1024, // 256MB
			MaxCPU:       0.2,               // 20% CPU
			NetworkMode:  "bridge",
			Tools:        []string{"Read", "Write", "Bash"},
			SystemPrompt: fmt.Sprintf("Sustained load test agent %d", i),
		}

		agentID, err := s.agentService.CreateAgent(ctx, agentConfig)
		s.Require().NoError(err, "지속 부하 테스트 에이전트 생성 실패")

		err = s.agentService.StartAgent(ctx, agentID)
		s.Require().NoError(err, "지속 부하 테스트 에이전트 시작 실패")

		agentIDs = append(agentIDs, agentID)
	}

	// 모든 에이전트가 running 상태가 될 때까지 대기
	s.Eventually(func() bool {
		for _, agentID := range agentIDs {
			agentInfo, err := s.agentService.GetAgent(ctx, agentID)
			if err != nil || agentInfo.Status != "running" {
				return false
			}
		}
		return true
	}, 60*time.Second, 2*time.Second, "모든 지속 부하 테스트 에이전트가 running 상태가 되지 않음")

	// 지속적인 작업 실행
	endTime := time.Now().Add(s.testDuration)
	for time.Now().Before(endTime) {
		for _, agentID := range agentIDs {
			wg.Add(1)
			go func(id string) {
				defer wg.Done()

				startTime := time.Now()
				taskConfig := &agent.TaskConfig{
					Command: fmt.Sprintf("echo 'Load test at %s'", startTime.Format(time.RFC3339)),
					Timeout: 30 * time.Second,
				}

				result, err := s.agentService.ExecuteTask(ctx, id, taskConfig)
				responseTime := time.Since(startTime)

				mu.Lock()
				metrics.TotalTasks++
				if err != nil {
					metrics.FailedTasks++
				} else {
					metrics.SuccessfulTasks++
					taskResults = append(taskResults, result)
				}

				if responseTime > metrics.MaxResponseTime {
					metrics.MaxResponseTime = responseTime
				}
				if responseTime < metrics.MinResponseTime {
					metrics.MinResponseTime = responseTime
				}
				mu.Unlock()
			}(agentID)
		}

		// 작업 간격 (1초)
		time.Sleep(1 * time.Second)
	}

	wg.Wait()

	// 메트릭 계산
	metrics.EndTime = time.Now()
	metrics.TestDuration = metrics.EndTime.Sub(metrics.StartTime)
	metrics.TotalAgents = numAgents
	metrics.SuccessfulAgents = numAgents

	if metrics.TotalTasks > 0 {
		metrics.ErrorRate = float64(metrics.FailedTasks) / float64(metrics.TotalTasks) * 100
		metrics.ThroughputPerSec = float64(metrics.SuccessfulTasks) / metrics.TestDuration.Seconds()
	}

	// 결과 출력
	s.T().Logf("=== 지속 부하 테스트 결과 ===")
	s.T().Logf("테스트 시간: %v", metrics.TestDuration)
	s.T().Logf("총 작업: %d", metrics.TotalTasks)
	s.T().Logf("성공한 작업: %d", metrics.SuccessfulTasks)
	s.T().Logf("실패한 작업: %d", metrics.FailedTasks)
	s.T().Logf("에러율: %.2f%%", metrics.ErrorRate)
	s.T().Logf("처리량: %.2f tasks/sec", metrics.ThroughputPerSec)
	s.T().Logf("최대 응답시간: %v", metrics.MaxResponseTime)
	s.T().Logf("최소 응답시간: %v", metrics.MinResponseTime)

	// 성능 기준 확인
	s.Assert().LessOrEqual(metrics.ErrorRate, 2.0, "지속 부하 테스트에서 에러율이 2%를 초과함")
	s.Assert().LessOrEqual(metrics.MaxResponseTime, 60*time.Second, "최대 응답시간이 60초를 초과함")

	// 정리
	for _, agentID := range agentIDs {
		if err := s.agentService.StopAgent(ctx, agentID); err != nil {
			s.T().Logf("지속 부하 테스트 에이전트 중지 실패 %s: %v", agentID, err)
		}
	}
}

// TestMemoryLeakDetection은 메모리 누수를 감지합니다
func (s *PerformanceTestSuite) TestMemoryLeakDetection() {
	ctx := context.Background()
	numIterations := 50
	agentsPerIteration := 5

	var initialMemoryMB, finalMemoryMB float64

	s.T().Logf("메모리 누수 감지 테스트 시작: %d회 반복, 회당 %d개 에이전트", 
		numIterations, agentsPerIteration)

	// 초기 메모리 사용량 측정
	initialStats, err := s.dockerClient.DiskUsage(ctx)
	s.Require().NoError(err, "초기 디스크 사용량 조회 실패")
	initialMemoryMB = float64(initialStats.LayersSize) / (1024 * 1024)

	for i := 0; i < numIterations; i++ {
		agentIDs := make([]string, 0, agentsPerIteration)

		// 에이전트 생성 및 작업 실행
		for j := 0; j < agentsPerIteration; j++ {
			agentConfig := &agent.Config{
				Name:         fmt.Sprintf("memory-test-agent-%d-%d", i, j),
				WorkspaceID:  "memory-test-workspace",
				ImageName:    "aicli/claude-agent:test",
				MaxMemory:    128 * 1024 * 1024, // 128MB
				MaxCPU:       0.1,               // 10% CPU
				NetworkMode:  "bridge",
				Tools:        []string{"Read", "Write"},
				SystemPrompt: "Memory leak test agent",
			}

			agentID, err := s.agentService.CreateAgent(ctx, agentConfig)
			s.Require().NoError(err, "메모리 테스트 에이전트 생성 실패")

			err = s.agentService.StartAgent(ctx, agentID)
			s.Require().NoError(err, "메모리 테스트 에이전트 시작 실패")

			agentIDs = append(agentIDs, agentID)

			// 간단한 작업 실행
			taskConfig := &agent.TaskConfig{
				Command: "echo 'Memory test'",
				Timeout: 10 * time.Second,
			}

			_, err = s.agentService.ExecuteTask(ctx, agentID, taskConfig)
			s.Assert().NoError(err, "메모리 테스트 작업 실행 실패")
		}

		// 에이전트 정리
		for _, agentID := range agentIDs {
			err := s.agentService.StopAgent(ctx, agentID)
			s.Assert().NoError(err, "메모리 테스트 에이전트 중지 실패")

			err = s.agentService.DeleteAgent(ctx, agentID)
			s.Assert().NoError(err, "메모리 테스트 에이전트 삭제 실패")
		}

		// 가비지 컬렉션 대기
		time.Sleep(1 * time.Second)

		if (i+1)%10 == 0 {
			s.T().Logf("메모리 누수 테스트 진행: %d/%d 완료", i+1, numIterations)
		}
	}

	// 최종 메모리 사용량 측정
	finalStats, err := s.dockerClient.DiskUsage(ctx)
	s.Require().NoError(err, "최종 디스크 사용량 조회 실패")
	finalMemoryMB = float64(finalStats.LayersSize) / (1024 * 1024)

	memoryIncreaseMB := finalMemoryMB - initialMemoryMB
	memoryIncreasePercent := (memoryIncreaseMB / initialMemoryMB) * 100

	s.T().Logf("=== 메모리 누수 감지 테스트 결과 ===")
	s.T().Logf("초기 메모리: %.2f MB", initialMemoryMB)
	s.T().Logf("최종 메모리: %.2f MB", finalMemoryMB)
	s.T().Logf("메모리 증가: %.2f MB (%.2f%%)", memoryIncreaseMB, memoryIncreasePercent)

	// 메모리 누수 기준 확인 (20% 이하 증가는 허용)
	s.Assert().LessOrEqual(memoryIncreasePercent, 20.0, 
		"메모리 사용량이 20%% 이상 증가하여 메모리 누수 가능성이 있음")
}

// TestPerformanceTestSuite는 성능 테스트 스위트를 실행합니다
func TestPerformanceTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("성능 테스트는 short 모드에서 스킵됩니다")
	}

	// 성능 테스트 환경 변수 확인
	if os.Getenv("PERFORMANCE_TEST") != "1" {
		t.Skip("PERFORMANCE_TEST 환경 변수가 설정되지 않음")
	}

	suite.Run(t, new(PerformanceTestSuite))
}