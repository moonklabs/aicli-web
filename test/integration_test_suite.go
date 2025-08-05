// 통합 테스트 스위트
// 멀티 에이전트 플랫폼의 전체 시스템 통합 테스트

//go:build integration
// +build integration

package test

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/aicli/aicli-web/internal/agent"
	"github.com/aicli/aicli-web/internal/docker"
	"github.com/aicli/aicli-web/internal/storage"
	"github.com/aicli/aicli-web/internal/testutil"
	"github.com/docker/docker/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// IntegrationTestSuite는 전체 시스템 통합 테스트를 정의합니다
type IntegrationTestSuite struct {
	suite.Suite
	dockerClient  *client.Client
	storage       storage.Storage
	agentService  *agent.Service
	testWorkspace string
	testAgents    []string
	cleanup       []func()
}

// SetupSuite는 테스트 스위트 시작 전 초기화를 수행합니다
func (s *IntegrationTestSuite) SetupSuite() {
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

	// Docker 연결 확인
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err = s.dockerClient.Info(ctx)
	s.Require().NoError(err, "Docker 데몬 연결 실패")

	// 스토리지 초기화
	s.storage, err = storage.New()
	s.Require().NoError(err, "스토리지 초기화 실패")

	// 테스트 워크스페이스 생성
	s.testWorkspace = testutil.TempDir(s.T(), "integration-test-workspace")

	// 에이전트 서비스 초기화
	s.agentService = agent.NewService(s.storage, s.dockerClient)
	s.Require().NotNil(s.agentService, "에이전트 서비스 초기화 실패")

	s.T().Logf("통합 테스트 환경 초기화 완료 - Workspace: %s", s.testWorkspace)
}

// TearDownSuite는 테스트 스위트 종료 후 정리를 수행합니다
func (s *IntegrationTestSuite) TearDownSuite() {
	// 생성된 에이전트들 정리
	ctx := context.Background()
	for _, agentID := range s.testAgents {
		if err := s.agentService.DeleteAgent(ctx, agentID); err != nil {
			s.T().Logf("에이전트 정리 실패 %s: %v", agentID, err)
		}
	}

	// 추가 정리 함수들 실행
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

	s.T().Log("통합 테스트 환경 정리 완료")
}

// SetupTest는 각 테스트 시작 전 초기화를 수행합니다
func (s *IntegrationTestSuite) SetupTest() {
	s.testAgents = []string{} // 각 테스트마다 에이전트 목록 초기화
}

// TearDownTest는 각 테스트 종료 후 정리를 수행합니다
func (s *IntegrationTestSuite) TearDownTest() {
	// 테스트별 에이전트 정리
	ctx := context.Background()
	for _, agentID := range s.testAgents {
		if err := s.agentService.DeleteAgent(ctx, agentID); err != nil {
			s.T().Logf("테스트 에이전트 정리 실패 %s: %v", agentID, err)
		}
	}
	s.testAgents = []string{}
}

// TestCompleteAgentLifecycle는 에이전트의 전체 생명주기를 테스트합니다
func (s *IntegrationTestSuite) TestCompleteAgentLifecycle() {
	ctx := context.Background()

	// 1. 에이전트 생성
	agentConfig := &agent.Config{
		Name:         "test-lifecycle-agent",
		WorkspaceID:  "test-workspace-1",
		ImageName:    "aicli/claude-agent:test",
		MaxMemory:    512 * 1024 * 1024, // 512MB
		MaxCPU:       0.5,               // 50% CPU
		NetworkMode:  "bridge",
		Tools:        []string{"Read", "Write", "Bash"},
		SystemPrompt: "You are a test assistant for integration testing.",
	}

	agentID, err := s.agentService.CreateAgent(ctx, agentConfig)
	s.Require().NoError(err, "에이전트 생성 실패")
	s.Require().NotEmpty(agentID, "에이전트 ID가 비어있음")
	s.testAgents = append(s.testAgents, agentID)

	s.T().Logf("에이전트 생성 완료: %s", agentID)

	// 2. 에이전트 시작
	err = s.agentService.StartAgent(ctx, agentID)
	s.Require().NoError(err, "에이전트 시작 실패")

	// 3. 에이전트 상태 확인 (시작 대기)
	s.Eventually(func() bool {
		agentInfo, err := s.agentService.GetAgent(ctx, agentID)
		if err != nil {
			return false
		}
		return agentInfo.Status == "running"
	}, 30*time.Second, 1*time.Second, "에이전트가 running 상태가 되지 않음")

	// 4. 작업 실행
	taskConfig := &agent.TaskConfig{
		Command: "echo 'Hello from integration test'",
		Timeout: 30 * time.Second,
	}

	result, err := s.agentService.ExecuteTask(ctx, agentID, taskConfig)
	s.Require().NoError(err, "작업 실행 실패")
	s.Assert().Contains(result.Output, "Hello from integration test", "작업 출력이 예상과 다름")

	// 5. 로그 확인
	logs, err := s.agentService.GetAgentLogs(ctx, agentID, &agent.LogOptions{
		Lines: 100,
		Since: time.Now().Add(-5 * time.Minute),
	})
	s.Require().NoError(err, "로그 조회 실패")
	s.Assert().NotEmpty(logs, "로그가 비어있음")

	// 6. 에이전트 중지
	err = s.agentService.StopAgent(ctx, agentID)
	s.Require().NoError(err, "에이전트 중지 실패")

	// 7. 에이전트 상태 확인 (중지 대기)
	s.Eventually(func() bool {
		agentInfo, err := s.agentService.GetAgent(ctx, agentID)
		if err != nil {
			return false
		}
		return agentInfo.Status == "stopped"
	}, 15*time.Second, 1*time.Second, "에이전트가 stopped 상태가 되지 않음")

	s.T().Log("에이전트 생명주기 테스트 완료")
}

// TestConcurrentAgentOperations는 동시 에이전트 작업을 테스트합니다
func (s *IntegrationTestSuite) TestConcurrentAgentOperations() {
	ctx := context.Background()
	numAgents := 5

	var wg sync.WaitGroup
	agentIDs := make([]string, numAgents)
	errors := make([]error, numAgents)

	// 동시에 여러 에이전트 생성
	for i := 0; i < numAgents; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			agentConfig := &agent.Config{
				Name:         fmt.Sprintf("concurrent-agent-%d", index),
				WorkspaceID:  "test-workspace-1",
				ImageName:    "aicli/claude-agent:test",
				MaxMemory:    256 * 1024 * 1024, // 256MB
				MaxCPU:       0.25,              // 25% CPU
				NetworkMode:  "bridge",
				Tools:        []string{"Read", "Write"},
				SystemPrompt: fmt.Sprintf("You are test agent number %d", index),
			}

			agentID, err := s.agentService.CreateAgent(ctx, agentConfig)
			agentIDs[index] = agentID
			errors[index] = err
		}(i)
	}

	wg.Wait()

	// 모든 에이전트가 성공적으로 생성되었는지 확인
	for i, err := range errors {
		s.Require().NoError(err, "에이전트 %d 생성 실패", i)
		s.Require().NotEmpty(agentIDs[i], "에이전트 %d ID가 비어있음", i)
		s.testAgents = append(s.testAgents, agentIDs[i])
	}

	// 동시에 모든 에이전트 시작
	for i, agentID := range agentIDs {
		wg.Add(1)
		go func(index int, id string) {
			defer wg.Done()
			err := s.agentService.StartAgent(ctx, id)
			errors[index] = err
		}(i, agentID)
	}

	wg.Wait()

	// 모든 에이전트가 성공적으로 시작되었는지 확인
	for i, err := range errors {
		s.Require().NoError(err, "에이전트 %d 시작 실패", i)
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
	}, 60*time.Second, 2*time.Second, "모든 에이전트가 running 상태가 되지 않음")

	// 동시에 모든 에이전트에서 작업 실행
	taskResults := make([]*agent.TaskResult, numAgents)
	for i, agentID := range agentIDs {
		wg.Add(1)
		go func(index int, id string) {
			defer wg.Done()

			taskConfig := &agent.TaskConfig{
				Command: fmt.Sprintf("echo 'Task from agent %d'", index),
				Timeout: 30 * time.Second,
			}

			result, err := s.agentService.ExecuteTask(ctx, id, taskConfig)
			if err != nil {
				errors[index] = err
				return
			}
			taskResults[index] = result
			errors[index] = nil
		}(i, agentID)
	}

	wg.Wait()

	// 모든 작업이 성공적으로 실행되었는지 확인
	for i, err := range errors {
		s.Require().NoError(err, "에이전트 %d 작업 실행 실패", i)
		s.Assert().Contains(taskResults[i].Output, fmt.Sprintf("Task from agent %d", i), 
			"에이전트 %d 작업 출력이 예상과 다름", i)
	}

	s.T().Logf("동시 에이전트 작업 테스트 완료 - %d개 에이전트", numAgents)
}

// TestAgentFailureRecovery는 에이전트 장애 복구를 테스트합니다
func (s *IntegrationTestSuite) TestAgentFailureRecovery() {
	ctx := context.Background()

	// 에이전트 생성 및 시작
	agentConfig := &agent.Config{
		Name:         "failure-test-agent",
		WorkspaceID:  "test-workspace-1",
		ImageName:    "aicli/claude-agent:test",
		MaxMemory:    256 * 1024 * 1024,
		MaxCPU:       0.25,
		NetworkMode:  "bridge",
		Tools:        []string{"Read", "Write", "Bash"},
		SystemPrompt: "You are a test assistant for failure recovery testing.",
		AutoRestart:  true, // 자동 재시작 활성화
	}

	agentID, err := s.agentService.CreateAgent(ctx, agentConfig)
	s.Require().NoError(err, "에이전트 생성 실패")
	s.testAgents = append(s.testAgents, agentID)

	err = s.agentService.StartAgent(ctx, agentID)
	s.Require().NoError(err, "에이전트 시작 실패")

	// 에이전트가 running 상태가 될 때까지 대기
	s.Eventually(func() bool {
		agentInfo, err := s.agentService.GetAgent(ctx, agentID)
		return err == nil && agentInfo.Status == "running"
	}, 30*time.Second, 1*time.Second, "에이전트가 running 상태가 되지 않음")

	// 컨테이너 강제 종료 (장애 시뮬레이션)
	agentInfo, err := s.agentService.GetAgent(ctx, agentID)
	s.Require().NoError(err, "에이전트 정보 조회 실패")
	s.Require().NotEmpty(agentInfo.ContainerID, "컨테이너 ID가 비어있음")

	err = s.dockerClient.ContainerKill(ctx, agentInfo.ContainerID, "SIGKILL")
	s.Require().NoError(err, "컨테이너 강제 종료 실패")

	s.T().Log("에이전트 컨테이너를 강제 종료했습니다")

	// 자동 복구가 일어날 때까지 대기 (최대 60초)
	s.Eventually(func() bool {
		agentInfo, err := s.agentService.GetAgent(ctx, agentID)
		if err != nil {
			return false
		}
		// 새로운 컨테이너 ID로 복구되었는지 확인
		return agentInfo.Status == "running" && agentInfo.ContainerID != agentInfo.ContainerID
	}, 60*time.Second, 2*time.Second, "에이전트 자동 복구가 실행되지 않음")

	// 복구 후 작업 실행 확인
	taskConfig := &agent.TaskConfig{
		Command: "echo 'Recovery test successful'",
		Timeout: 30 * time.Second,
	}

	result, err := s.agentService.ExecuteTask(ctx, agentID, taskConfig)
	s.Require().NoError(err, "복구 후 작업 실행 실패")
	s.Assert().Contains(result.Output, "Recovery test successful", "복구 후 작업 출력이 예상과 다름")

	s.T().Log("에이전트 장애 복구 테스트 완료")
}

// TestResourceLimits는 리소스 제한을 테스트합니다
func (s *IntegrationTestSuite) TestResourceLimits() {
	ctx := context.Background()

	// 메모리 제한이 있는 에이전트 생성
	agentConfig := &agent.Config{
		Name:        "resource-limit-agent",
		WorkspaceID: "test-workspace-1",
		ImageName:   "aicli/claude-agent:test",
		MaxMemory:   64 * 1024 * 1024, // 64MB (매우 제한적)
		MaxCPU:      0.1,              // 10% CPU
		NetworkMode: "bridge",
		Tools:       []string{"Bash"},
		SystemPrompt: "You are a test assistant with limited resources.",
	}

	agentID, err := s.agentService.CreateAgent(ctx, agentConfig)
	s.Require().NoError(err, "리소스 제한 에이전트 생성 실패")
	s.testAgents = append(s.testAgents, agentID)

	err = s.agentService.StartAgent(ctx, agentID)
	s.Require().NoError(err, "리소스 제한 에이전트 시작 실패")

	// 에이전트가 running 상태가 될 때까지 대기
	s.Eventually(func() bool {
		agentInfo, err := s.agentService.GetAgent(ctx, agentID)
		return err == nil && agentInfo.Status == "running"
	}, 30*time.Second, 1*time.Second, "리소스 제한 에이전트가 running 상태가 되지 않음")

	// 리소스 사용량 확인
	metrics, err := s.agentService.GetAgentMetrics(ctx, agentID)
	s.Require().NoError(err, "에이전트 메트릭 조회 실패")

	// 메모리 제한이 적용되었는지 확인
	s.Assert().LessOrEqual(metrics.MemoryLimit, int64(64*1024*1024), 
		"메모리 제한이 올바르게 적용되지 않음")

	// CPU 제한이 적용되었는지 확인
	s.Assert().LessOrEqual(metrics.CPULimit, 0.1, 
		"CPU 제한이 올바르게 적용되지 않음")

	s.T().Log("리소스 제한 테스트 완료")
}

// TestIntegrationTestSuite는 통합 테스트 스위트를 실행합니다
func TestIntegrationTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("통합 테스트는 short 모드에서 스킵됩니다")
	}

	// 통합 테스트 환경 변수 확인
	if os.Getenv("INTEGRATION_TEST") != "1" {
		t.Skip("INTEGRATION_TEST 환경 변수가 설정되지 않음")
	}

	suite.Run(t, new(IntegrationTestSuite))
}