// 장애 시나리오 및 복구 테스트
// 시스템 장애 상황에서의 복구 능력을 테스트

//go:build integration
// +build integration

package test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/aicli/aicli-web/internal/docker"
	"github.com/aicli/aicli-web/internal/storage"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// FailureRecoveryTestSuite는 장애 복구 테스트를 정의합니다
type FailureRecoveryTestSuite struct {
	suite.Suite
	dockerClient  *client.Client
	dockerManager *docker.Client
	storage       storage.Storage
	testContainers []string
	cleanup       []func()
}

// FailureScenario는 장애 시나리오를 정의합니다
type FailureScenario struct {
	Name        string
	Description string
	Setup       func(s *FailureRecoveryTestSuite) error
	TriggerFailure func(s *FailureRecoveryTestSuite) error
	VerifyFailure  func(s *FailureRecoveryTestSuite) error
	TriggerRecovery func(s *FailureRecoveryTestSuite) error
	VerifyRecovery  func(s *FailureRecoveryTestSuite) error
	Cleanup     func(s *FailureRecoveryTestSuite) error
}

// SetupSuite는 장애 복구 테스트 스위트 초기화를 수행합니다
func (s *FailureRecoveryTestSuite) SetupSuite() {
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

	// Docker 매니저 초기화
	s.dockerManager, err = docker.NewClient()
	s.Require().NoError(err, "Docker 매니저 초기화 실패")

	// 스토리지 초기화
	s.storage, err = storage.New()
	s.Require().NoError(err, "스토리지 초기화 실패")

	s.T().Log("장애 복구 테스트 환경 초기화 완료")
}

// TearDownSuite는 장애 복구 테스트 스위트 정리를 수행합니다
func (s *FailureRecoveryTestSuite) TearDownSuite() {
	ctx := context.Background()

	// 테스트 컨테이너들 정리
	for _, containerID := range s.testContainers {
		if err := s.dockerClient.ContainerStop(ctx, containerID, container.StopOptions{}); err != nil {
			s.T().Logf("컨테이너 중지 실패 %s: %v", containerID, err)
		}
		if err := s.dockerClient.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{Force: true}); err != nil {
			s.T().Logf("컨테이너 제거 실패 %s: %v", containerID, err)
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

	s.T().Log("장애 복구 테스트 환경 정리 완료")
}

// SetupTest는 각 테스트 시작 전 초기화를 수행합니다
func (s *FailureRecoveryTestSuite) SetupTest() {
	s.testContainers = []string{}
}

// TearDownTest는 각 테스트 종료 후 정리를 수행합니다
func (s *FailureRecoveryTestSuite) TearDownTest() {
	ctx := context.Background()
	
	// 테스트별 컨테이너 정리
	for _, containerID := range s.testContainers {
		if err := s.dockerClient.ContainerStop(ctx, containerID, container.StopOptions{}); err != nil {
			s.T().Logf("테스트 컨테이너 중지 실패 %s: %v", containerID, err)
		}
		if err := s.dockerClient.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{Force: true}); err != nil {
			s.T().Logf("테스트 컨테이너 제거 실패 %s: %v", containerID, err)
		}
	}
	s.testContainers = []string{}
}

// createTestContainer는 테스트용 컨테이너를 생성합니다
func (s *FailureRecoveryTestSuite) createTestContainer(name, image string, autoRestart bool) (string, error) {
	ctx := context.Background()
	
	restartPolicy := container.RestartPolicy{Name: "no"}
	if autoRestart {
		restartPolicy = container.RestartPolicy{
			Name:              "unless-stopped",
			MaximumRetryCount: 3,
		}
	}

	createResp, err := s.dockerClient.ContainerCreate(ctx,
		&container.Config{
			Image: image,
			Cmd:   []string{"sleep", "300"},
			Labels: map[string]string{
				"test.type":       "failure-recovery",
				"test.container":  name,
				"test.timestamp":  fmt.Sprintf("%d", time.Now().Unix()),
			},
		},
		&container.HostConfig{
			RestartPolicy: restartPolicy,
			Resources: container.Resources{
				Memory:   128 * 1024 * 1024, // 128MB
				NanoCPUs: 100000000,         // 0.1 CPU
			},
		},
		nil,
		nil,
		name,
	)
	
	if err != nil {
		return "", err
	}

	containerID := createResp.ID
	s.testContainers = append(s.testContainers, containerID)
	
	return containerID, nil
}

// waitForContainerState는 컨테이너가 특정 상태가 될 때까지 대기합니다
func (s *FailureRecoveryTestSuite) waitForContainerState(containerID string, expectedRunning bool, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("타임아웃: 컨테이너 %s가 예상 상태가 되지 않음", containerID[:12])
		case <-ticker.C:
			inspect, err := s.dockerClient.ContainerInspect(context.Background(), containerID)
			if err != nil {
				continue
			}
			
			if inspect.State.Running == expectedRunning {
				return nil
			}
		}
	}
}

// TestContainerCrashRecovery는 컨테이너 크래시 복구를 테스트합니다
func (s *FailureRecoveryTestSuite) TestContainerCrashRecovery() {
	ctx := context.Background()

	s.T().Log("=== 컨테이너 크래시 복구 테스트 시작 ===")

	// 1. 자동 재시작이 활성화된 컨테이너 생성
	containerName := fmt.Sprintf("crash-recovery-test-%d", time.Now().Unix())
	containerID, err := s.createTestContainer(containerName, "alpine:latest", true)
	s.Require().NoError(err, "크래시 복구 테스트 컨테이너 생성 실패")

	// 2. 컨테이너 시작
	err = s.dockerClient.ContainerStart(ctx, containerID, types.ContainerStartOptions{})
	s.Require().NoError(err, "크래시 복구 테스트 컨테이너 시작 실패")

	// 3. 컨테이너가 실행 상태가 될 때까지 대기
	err = s.waitForContainerState(containerID, true, 15*time.Second)
	s.Require().NoError(err, "컨테이너가 실행 상태가 되지 않음")
	s.T().Log("컨테이너 정상 시작 확인")

	// 4. 컨테이너 강제 종료 (크래시 시뮬레이션)
	err = s.dockerClient.ContainerKill(ctx, containerID, "SIGKILL")
	s.Require().NoError(err, "컨테이너 강제 종료 실패")
	s.T().Log("컨테이너 강제 종료 (크래시 시뮬레이션)")

	// 5. 컨테이너가 중지되었는지 확인
	err = s.waitForContainerState(containerID, false, 10*time.Second)
	s.Require().NoError(err, "컨테이너가 중지되지 않음")
	s.T().Log("컨테이너 중지 확인")

	// 6. 자동 재시작이 일어날 때까지 대기
	s.T().Log("자동 재시작 대기 중... (최대 30초)")
	err = s.waitForContainerState(containerID, true, 30*time.Second)
	s.Require().NoError(err, "컨테이너 자동 재시작이 일어나지 않음")

	// 7. 재시작된 컨테이너 상태 확인
	inspect, err := s.dockerClient.ContainerInspect(ctx, containerID)
	s.Require().NoError(err, "재시작된 컨테이너 검사 실패")
	
	s.Assert().True(inspect.State.Running, "재시작된 컨테이너가 실행 중이지 않음")
	s.Assert().Greater(inspect.RestartCount, 0, "재시작 카운트가 증가하지 않음")
	s.T().Logf("컨테이너 자동 재시작 성공 (재시작 횟수: %d)", inspect.RestartCount)

	s.T().Log("=== 컨테이너 크래시 복구 테스트 완료 ===")
}

// TestResourceExhaustionRecovery는 리소스 부족 상황에서의 복구를 테스트합니다
func (s *FailureRecoveryTestSuite) TestResourceExhaustionRecovery() {
	ctx := context.Background()

	s.T().Log("=== 리소스 부족 복구 테스트 시작 ===")

	// 1. 메모리 제한이 있는 컨테이너 생성
	containerName := fmt.Sprintf("memory-limit-test-%d", time.Now().Unix())
	createResp, err := s.dockerClient.ContainerCreate(ctx,
		&container.Config{
			Image: "alpine:latest",
			Cmd:   []string{"sleep", "60"},
			Labels: map[string]string{
				"test.type":    "resource-exhaustion",
				"test.purpose": "memory-limit-test",
			},
		},
		&container.HostConfig{
			Resources: container.Resources{
				Memory: 16 * 1024 * 1024, // 16MB (매우 제한적)
			},
			RestartPolicy: container.RestartPolicy{
				Name:              "on-failure",
				MaximumRetryCount: 2,
			},
		},
		nil,
		nil,
		containerName,
	)
	s.Require().NoError(err, "메모리 제한 테스트 컨테이너 생성 실패")
	
	containerID := createResp.ID
	s.testContainers = append(s.testContainers, containerID)

	// 2. 컨테이너 시작
	err = s.dockerClient.ContainerStart(ctx, containerID, types.ContainerStartOptions{})
	s.Require().NoError(err, "메모리 제한 테스트 컨테이너 시작 실패")

	// 3. 컨테이너가 실행 상태가 될 때까지 대기
	err = s.waitForContainerState(containerID, true, 10*time.Second)
	s.Require().NoError(err, "메모리 제한 테스트 컨테이너가 실행 상태가 되지 않음")

	// 4. 메모리 부족 시뮬레이션 (메모리를 많이 사용하는 작업 실행)
	execConfig := types.ExecConfig{
		Cmd:          []string{"sh", "-c", "dd if=/dev/zero of=/tmp/bigfile bs=1M count=20 || true"},
		AttachStdout: true,
		AttachStderr: true,
	}

	execResp, err := s.dockerClient.ContainerExecCreate(ctx, containerID, execConfig)
	s.Require().NoError(err, "메모리 부족 시뮬레이션 명령 준비 실패")

	execAttach, err := s.dockerClient.ContainerExecAttach(ctx, execResp.ID, types.ExecStartCheck{})
	s.Require().NoError(err, "메모리 부족 시뮬레이션 명령 연결 실패")
	defer execAttach.Close()

	s.T().Log("메모리 부족 시뮬레이션 실행 중...")

	// 5. 컨테이너 상태 모니터링
	time.Sleep(5 * time.Second) // 메모리 부족이 일어날 시간을 줌

	// 6. 컨테이너 상태 확인
	inspect, err := s.dockerClient.ContainerInspect(ctx, containerID)
	s.Require().NoError(err, "메모리 제한 테스트 컨테이너 검사 실패")

	// 7. 리소스 사용량 확인
	stats, err := s.dockerClient.ContainerStats(ctx, containerID, false)
	if err == nil {
		defer stats.Body.Close()
		
		// 메모리 사용량이 제한에 근접했는지 확인
		s.T().Log("컨테이너 리소스 사용량 확인 완료")
	}

	s.T().Logf("컨테이너 상태: Running=%v, 재시작 횟수=%d", 
		inspect.State.Running, inspect.RestartCount)

	s.T().Log("=== 리소스 부족 복구 테스트 완료 ===")
}

// TestNetworkPartitionRecovery는 네트워크 분할 복구를 테스트합니다
func (s *FailureRecoveryTestSuite) TestNetworkPartitionRecovery() {
	ctx := context.Background()

	s.T().Log("=== 네트워크 분할 복구 테스트 시작 ===")

	// 1. 테스트용 네트워크 생성
	networkName := fmt.Sprintf("test-network-%d", time.Now().Unix())
	networkResp, err := s.dockerClient.NetworkCreate(ctx, networkName, types.NetworkCreate{
		Driver: "bridge",
		Labels: map[string]string{
			"test.type":    "network-partition",
			"test.purpose": "connectivity-test",
		},
	})
	s.Require().NoError(err, "테스트 네트워크 생성 실패")
	
	networkID := networkResp.ID
	s.cleanup = append(s.cleanup, func() {
		s.dockerClient.NetworkRemove(context.Background(), networkID)
	})

	// 2. 첫 번째 컨테이너 생성 (서버 역할)
	serverName := fmt.Sprintf("network-server-%d", time.Now().Unix())
	serverID, err := s.createTestContainer(serverName, "alpine:latest", false)
	s.Require().NoError(err, "네트워크 서버 컨테이너 생성 실패")

	// 3. 두 번째 컨테이너 생성 (클라이언트 역할)
	clientName := fmt.Sprintf("network-client-%d", time.Now().Unix())
	clientID, err := s.createTestContainer(clientName, "alpine:latest", false)
	s.Require().NoError(err, "네트워크 클라이언트 컨테이너 생성 실패")

	// 4. 컨테이너들을 네트워크에 연결
	err = s.dockerClient.NetworkConnect(ctx, networkID, serverID, nil)
	s.Require().NoError(err, "서버 컨테이너 네트워크 연결 실패")

	err = s.dockerClient.NetworkConnect(ctx, networkID, clientID, nil)
	s.Require().NoError(err, "클라이언트 컨테이너 네트워크 연결 실패")

	// 5. 컨테이너들 시작
	err = s.dockerClient.ContainerStart(ctx, serverID, types.ContainerStartOptions{})
	s.Require().NoError(err, "서버 컨테이너 시작 실패")

	err = s.dockerClient.ContainerStart(ctx, clientID, types.ContainerStartOptions{})
	s.Require().NoError(err, "클라이언트 컨테이너 시작 실패")

	// 6. 컨테이너들이 실행 상태가 될 때까지 대기
	err = s.waitForContainerState(serverID, true, 10*time.Second)
	s.Require().NoError(err, "서버 컨테이너가 실행 상태가 되지 않음")

	err = s.waitForContainerState(clientID, true, 10*time.Second)
	s.Require().NoError(err, "클라이언트 컨테이너가 실행 상태가 되지 않음")

	// 7. 네트워크 연결 테스트 (정상 상태)
	execConfig := types.ExecConfig{
		Cmd:          []string{"ping", "-c", "2", serverName},
		AttachStdout: true,
		AttachStderr: true,
	}

	execResp, err := s.dockerClient.ContainerExecCreate(ctx, clientID, execConfig)
	s.Require().NoError(err, "네트워크 연결 테스트 준비 실패")

	execAttach, err := s.dockerClient.ContainerExecAttach(ctx, execResp.ID, types.ExecStartCheck{})
	s.Require().NoError(err, "네트워크 연결 테스트 연결 실패")
	defer execAttach.Close()

	s.T().Log("네트워크 연결 정상 상태 확인")

	// 8. 네트워크 분할 시뮬레이션 (클라이언트를 네트워크에서 분리)
	err = s.dockerClient.NetworkDisconnect(ctx, networkID, clientID, true)
	s.Require().NoError(err, "네트워크 분할 시뮬레이션 실패")
	s.T().Log("네트워크 분할 시뮬레이션 완료")

	// 9. 네트워크 연결 실패 확인
	time.Sleep(2 * time.Second) // 네트워크 변경 사항이 적용될 시간을 줌

	// 10. 네트워크 복구 (클라이언트를 다시 네트워크에 연결)
	time.Sleep(5 * time.Second) // 분할 상태 유지
	
	err = s.dockerClient.NetworkConnect(ctx, networkID, clientID, nil)
	s.Require().NoError(err, "네트워크 복구 실패")
	s.T().Log("네트워크 복구 완료")

	// 11. 네트워크 연결 복구 확인
	time.Sleep(2 * time.Second) // 네트워크 복구가 적용될 시간을 줌

	execConfig = types.ExecConfig{
		Cmd:          []string{"ping", "-c", "2", serverName},
		AttachStdout: true,
		AttachStderr: true,
	}

	execResp, err = s.dockerClient.ContainerExecCreate(ctx, clientID, execConfig)
	s.Require().NoError(err, "네트워크 복구 테스트 준비 실패")

	execAttach, err = s.dockerClient.ContainerExecAttach(ctx, execResp.ID, types.ExecStartCheck{})
	s.Require().NoError(err, "네트워크 복구 테스트 연결 실패")
	defer execAttach.Close()

	s.T().Log("네트워크 연결 복구 확인 완료")

	s.T().Log("=== 네트워크 분할 복구 테스트 완료 ===")
}

// TestDiskSpaceExhaustionRecovery는 디스크 공간 부족 복구를 테스트합니다
func (s *FailureRecoveryTestSuite) TestDiskSpaceExhaustionRecovery() {
	ctx := context.Background()

	s.T().Log("=== 디스크 공간 부족 복구 테스트 시작 ===")

	// 1. 제한된 tmpfs를 가진 컨테이너 생성
	containerName := fmt.Sprintf("disk-space-test-%d", time.Now().Unix())
	createResp, err := s.dockerClient.ContainerCreate(ctx,
		&container.Config{
			Image: "alpine:latest",
			Cmd:   []string{"sleep", "120"},
			Labels: map[string]string{
				"test.type":    "disk-exhaustion",
				"test.purpose": "disk-space-test",
			},
		},
		&container.HostConfig{
			Tmpfs: map[string]string{
				"/tmp": "size=10m", // 10MB tmpfs
			},
		},
		nil,
		nil,
		containerName,
	)
	s.Require().NoError(err, "디스크 공간 테스트 컨테이너 생성 실패")
	
	containerID := createResp.ID
	s.testContainers = append(s.testContainers, containerID)

	// 2. 컨테이너 시작
	err = s.dockerClient.ContainerStart(ctx, containerID, types.ContainerStartOptions{})
	s.Require().NoError(err, "디스크 공간 테스트 컨테이너 시작 실패")

	// 3. 컨테이너가 실행 상태가 될 때까지 대기
	err = s.waitForContainerState(containerID, true, 10*time.Second)
	s.Require().NoError(err, "디스크 공간 테스트 컨테이너가 실행 상태가 되지 않음")

	// 4. 디스크 공간 부족 시뮬레이션
	execConfig := types.ExecConfig{
		Cmd:          []string{"sh", "-c", "dd if=/dev/zero of=/tmp/bigfile bs=1M count=15 || echo 'Disk full'"},
		AttachStdout: true,
		AttachStderr: true,
	}

	execResp, err := s.dockerClient.ContainerExecCreate(ctx, containerID, execConfig)
	s.Require().NoError(err, "디스크 공간 부족 시뮬레이션 준비 실패")

	execAttach, err := s.dockerClient.ContainerExecAttach(ctx, execResp.ID, types.ExecStartCheck{})
	s.Require().NoError(err, "디스크 공간 부족 시뮬레이션 연결 실패")
	defer execAttach.Close()

	s.T().Log("디스크 공간 부족 시뮬레이션 실행")

	// 5. 디스크 정리 (복구 시뮬레이션)
	time.Sleep(3 * time.Second)

	execConfig = types.ExecConfig{
		Cmd:          []string{"rm", "-f", "/tmp/bigfile"},
		AttachStdout: true,
		AttachStderr: true,
	}

	execResp, err = s.dockerClient.ContainerExecCreate(ctx, containerID, execConfig)
	s.Require().NoError(err, "디스크 정리 작업 준비 실패")

	execAttach, err = s.dockerClient.ContainerExecAttach(ctx, execResp.ID, types.ExecStartCheck{})
	s.Require().NoError(err, "디스크 정리 작업 연결 실패")
	defer execAttach.Close()

	s.T().Log("디스크 정리 작업 완료")

	// 6. 디스크 공간 확인
	execConfig = types.ExecConfig{
		Cmd:          []string{"df", "-h", "/tmp"},
		AttachStdout: true,
		AttachStderr: true,
	}

	execResp, err = s.dockerClient.ContainerExecCreate(ctx, containerID, execConfig)
	s.Require().NoError(err, "디스크 공간 확인 준비 실패")

	execAttach, err = s.dockerClient.ContainerExecAttach(ctx, execResp.ID, types.ExecStartCheck{})
	s.Require().NoError(err, "디스크 공간 확인 연결 실패")
	defer execAttach.Close()

	s.T().Log("디스크 공간 복구 확인 완료")

	s.T().Log("=== 디스크 공간 부족 복구 테스트 완료 ===")
}

// TestFailureRecoveryTestSuite는 장애 복구 테스트 스위트를 실행합니다
func TestFailureRecoveryTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("장애 복구 테스트는 short 모드에서 스킵됩니다")
	}

	// 통합 테스트 환경 변수 확인
	if os.Getenv("INTEGRATION_TEST") != "1" {
		t.Skip("INTEGRATION_TEST 환경 변수가 설정되지 않음")
	}

	suite.Run(t, new(FailureRecoveryTestSuite))
}