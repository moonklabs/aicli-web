// Docker 통합 테스트
// Docker 컨테이너 관리, 네트워크, 볼륨, 보안 등의 통합 테스트

//go:build integration
// +build integration

package test

import (
	"context"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/aicli/aicli-web/internal/docker"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// DockerIntegrationTestSuite는 Docker 통합 테스트를 정의합니다
type DockerIntegrationTestSuite struct {
	suite.Suite
	dockerClient     *client.Client
	dockerManager    *docker.Client
	testNetworkID    string
	testVolumeNames  []string
	testContainerIDs []string
	cleanup          []func()
}

// SetupSuite는 Docker 테스트 스위트 초기화를 수행합니다
func (s *DockerIntegrationTestSuite) SetupSuite() {
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

	// 테스트용 네트워크 생성
	s.createTestNetwork()

	s.T().Log("Docker 통합 테스트 환경 초기화 완료")
}

// TearDownSuite는 Docker 테스트 스위트 정리를 수행합니다
func (s *DockerIntegrationTestSuite) TearDownSuite() {
	ctx := context.Background()

	// 테스트 컨테이너들 정리
	for _, containerID := range s.testContainerIDs {
		if err := s.dockerClient.ContainerStop(ctx, containerID, container.StopOptions{}); err != nil {
			s.T().Logf("컨테이너 중지 실패 %s: %v", containerID, err)
		}
		if err := s.dockerClient.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{Force: true}); err != nil {
			s.T().Logf("컨테이너 제거 실패 %s: %v", containerID, err)
		}
	}

	// 테스트 볼륨들 정리
	for _, volumeName := range s.testVolumeNames {
		if err := s.dockerClient.VolumeRemove(ctx, volumeName, true); err != nil {
			s.T().Logf("볼륨 제거 실패 %s: %v", volumeName, err)
		}
	}

	// 테스트 네트워크 정리
	if s.testNetworkID != "" {
		if err := s.dockerClient.NetworkRemove(ctx, s.testNetworkID); err != nil {
			s.T().Logf("네트워크 제거 실패 %s: %v", s.testNetworkID, err)
		}
	}

	// 추가 정리 함수들 실행
	for _, cleanup := range s.cleanup {
		cleanup()
	}

	// 리소스 정리
	if s.dockerClient != nil {
		s.dockerClient.Close()
	}

	s.T().Log("Docker 통합 테스트 환경 정리 완료")
}

// SetupTest는 각 테스트 시작 전 초기화를 수행합니다
func (s *DockerIntegrationTestSuite) SetupTest() {
	s.testContainerIDs = []string{}
	s.testVolumeNames = []string{}
}

// createTestNetwork는 테스트용 네트워크를 생성합니다
func (s *DockerIntegrationTestSuite) createTestNetwork() {
	ctx := context.Background()
	
	networkName := fmt.Sprintf("aicli-test-network-%d", time.Now().Unix())
	resp, err := s.dockerClient.NetworkCreate(ctx, networkName, types.NetworkCreate{
		Driver: "bridge",
		IPAM: &network.IPAM{
			Config: []network.IPAMConfig{
				{
					Subnet:  "172.30.0.0/16",
					Gateway: "172.30.0.1",
				},
			},
		},
		Options: map[string]string{
			"com.docker.network.bridge.name": networkName,
		},
		Labels: map[string]string{
			"test.type":    "integration",
			"test.purpose": "docker-network-test",
		},
	})
	s.Require().NoError(err, "테스트 네트워크 생성 실패")
	
	s.testNetworkID = resp.ID
	s.T().Logf("테스트 네트워크 생성 완료: %s", networkName)
}

// TestContainerLifecycle는 컨테이너 생명주기를 테스트합니다
func (s *DockerIntegrationTestSuite) TestContainerLifecycle() {
	ctx := context.Background()

	// 1. 컨테이너 생성
	containerName := fmt.Sprintf("test-lifecycle-%d", time.Now().Unix())
	createResp, err := s.dockerClient.ContainerCreate(ctx,
		&container.Config{
			Image: "alpine:latest",
			Cmd:   []string{"sleep", "30"},
			Labels: map[string]string{
				"test.type":    "integration",
				"test.purpose": "container-lifecycle",
			},
			WorkingDir: "/workspace",
			Env:        []string{"TEST_ENV=integration"},
		},
		&container.HostConfig{
			Resources: container.Resources{
				Memory:   64 * 1024 * 1024, // 64MB
				NanoCPUs: 50000000,         // 0.05 CPU
			},
			NetworkMode: container.NetworkMode(s.testNetworkID),
			LogConfig: container.LogConfig{
				Type: "json-file",
				Config: map[string]string{
					"max-size": "10m",
					"max-file": "3",
				},
			},
		},
		&network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{
				s.testNetworkID: {},
			},
		},
		nil,
		containerName,
	)
	s.Require().NoError(err, "컨테이너 생성 실패")
	
	containerID := createResp.ID
	s.testContainerIDs = append(s.testContainerIDs, containerID)
	s.T().Logf("컨테이너 생성 완료: %s", containerID[:12])

	// 2. 컨테이너 시작
	err = s.dockerClient.ContainerStart(ctx, containerID, types.ContainerStartOptions{})
	s.Require().NoError(err, "컨테이너 시작 실패")
	s.T().Log("컨테이너 시작 완료")

	// 3. 컨테이너 상태 확인
	s.Eventually(func() bool {
		inspect, err := s.dockerClient.ContainerInspect(ctx, containerID)
		return err == nil && inspect.State.Running
	}, 10*time.Second, 500*time.Millisecond, "컨테이너가 실행 상태가 되지 않음")

	// 4. 컨테이너 정보 검사
	inspect, err := s.dockerClient.ContainerInspect(ctx, containerID)
	s.Require().NoError(err, "컨테이너 검사 실패")
	
	s.Assert().True(inspect.State.Running, "컨테이너가 실행 중이지 않음")
	s.Assert().Equal("alpine:latest", inspect.Config.Image, "컨테이너 이미지가 다름")
	s.Assert().Contains(inspect.Config.Env, "TEST_ENV=integration", "환경 변수가 설정되지 않음")
	s.Assert().Equal("/workspace", inspect.Config.WorkingDir, "작업 디렉토리가 다름")

	// 5. 리소스 제한 확인
	s.Assert().Equal(int64(64*1024*1024), inspect.HostConfig.Memory, "메모리 제한이 다름")
	s.Assert().Equal(int64(50000000), inspect.HostConfig.NanoCPUs, "CPU 제한이 다름")

	// 6. 네트워크 설정 확인
	networkSettings := inspect.NetworkSettings
	s.Assert().Contains(networkSettings.Networks, s.testNetworkID, "컨테이너가 테스트 네트워크에 연결되지 않음")

	// 7. 컨테이너 일시 정지
	err = s.dockerClient.ContainerPause(ctx, containerID)
	s.Require().NoError(err, "컨테이너 일시 정지 실패")
	
	inspect, err = s.dockerClient.ContainerInspect(ctx, containerID)
	s.Require().NoError(err)
	s.Assert().True(inspect.State.Paused, "컨테이너가 일시 정지되지 않음")
	s.T().Log("컨테이너 일시 정지 완료")

	// 8. 컨테이너 재시작
	err = s.dockerClient.ContainerUnpause(ctx, containerID)
	s.Require().NoError(err, "컨테이너 재시작 실패")
	
	inspect, err = s.dockerClient.ContainerInspect(ctx, containerID)
	s.Require().NoError(err)
	s.Assert().False(inspect.State.Paused, "컨테이너가 아직 일시 정지 상태임")
	s.Assert().True(inspect.State.Running, "컨테이너가 실행 중이지 않음")
	s.T().Log("컨테이너 재시작 완료")

	// 9. 컨테이너 중지
	timeout := 5 * time.Second
	err = s.dockerClient.ContainerStop(ctx, containerID, &timeout)
	s.Require().NoError(err, "컨테이너 중지 실패")
	
	s.Eventually(func() bool {
		inspect, err := s.dockerClient.ContainerInspect(ctx, containerID)
		return err == nil && !inspect.State.Running
	}, 10*time.Second, 500*time.Millisecond, "컨테이너가 중지되지 않음")
	s.T().Log("컨테이너 중지 완료")
}

// TestContainerExecution는 컨테이너 내 명령 실행을 테스트합니다
func (s *DockerIntegrationTestSuite) TestContainerExecution() {
	ctx := context.Background()

	// 컨테이너 생성 및 시작
	containerName := fmt.Sprintf("test-exec-%d", time.Now().Unix())
	createResp, err := s.dockerClient.ContainerCreate(ctx,
		&container.Config{
			Image: "alpine:latest",
			Cmd:   []string{"sleep", "60"},
			Labels: map[string]string{
				"test.type":    "integration",
				"test.purpose": "container-execution",
			},
		},
		&container.HostConfig{
			Resources: container.Resources{
				Memory: 64 * 1024 * 1024, // 64MB
			},
		},
		nil,
		nil,
		containerName,
	)
	s.Require().NoError(err, "실행 테스트 컨테이너 생성 실패")
	
	containerID := createResp.ID
	s.testContainerIDs = append(s.testContainerIDs, containerID)

	err = s.dockerClient.ContainerStart(ctx, containerID, types.ContainerStartOptions{})
	s.Require().NoError(err, "실행 테스트 컨테이너 시작 실패")

	// 컨테이너가 실행 상태가 될 때까지 대기
	s.Eventually(func() bool {
		inspect, err := s.dockerClient.ContainerInspect(ctx, containerID)
		return err == nil && inspect.State.Running
	}, 10*time.Second, 500*time.Millisecond, "실행 테스트 컨테이너가 실행 상태가 되지 않음")

	// 1. 간단한 명령 실행
	execConfig := types.ExecConfig{
		Cmd:          []string{"echo", "Hello Docker Integration Test"},
		AttachStdout: true,
		AttachStderr: true,
	}

	execResp, err := s.dockerClient.ContainerExecCreate(ctx, containerID, execConfig)
	s.Require().NoError(err, "명령 실행 준비 실패")

	execAttach, err := s.dockerClient.ContainerExecAttach(ctx, execResp.ID, types.ExecStartCheck{})
	s.Require().NoError(err, "명령 실행 연결 실패")
	defer execAttach.Close()

	output, err := io.ReadAll(execAttach.Reader)
	s.Require().NoError(err, "명령 실행 결과 읽기 실패")
	
	outputStr := string(output)
	s.Assert().Contains(outputStr, "Hello Docker Integration Test", "명령 실행 결과가 예상과 다름")
	s.T().Logf("명령 실행 결과: %s", outputStr)

	// 2. 파일 시스템 작업 테스트
	execConfig = types.ExecConfig{
		Cmd:          []string{"sh", "-c", "echo 'test content' > /tmp/test.txt && cat /tmp/test.txt"},
		AttachStdout: true,
		AttachStderr: true,
	}

	execResp, err = s.dockerClient.ContainerExecCreate(ctx, containerID, execConfig)
	s.Require().NoError(err, "파일 시스템 작업 준비 실패")

	execAttach, err = s.dockerClient.ContainerExecAttach(ctx, execResp.ID, types.ExecStartCheck{})
	s.Require().NoError(err, "파일 시스템 작업 연결 실패")
	defer execAttach.Close()

	output, err = io.ReadAll(execAttach.Reader)
	s.Require().NoError(err, "파일 시스템 작업 결과 읽기 실패")
	
	outputStr = string(output)
	s.Assert().Contains(outputStr, "test content", "파일 시스템 작업 결과가 예상과 다름")
	s.T().Log("파일 시스템 작업 테스트 완료")

	// 3. 환경 변수 확인
	execConfig = types.ExecConfig{
		Cmd:          []string{"env"},
		AttachStdout: true,
		AttachStderr: true,
	}

	execResp, err = s.dockerClient.ContainerExecCreate(ctx, containerID, execConfig)
	s.Require().NoError(err, "환경 변수 확인 준비 실패")

	execAttach, err = s.dockerClient.ContainerExecAttach(ctx, execResp.ID, types.ExecStartCheck{})
	s.Require().NoError(err, "환경 변수 확인 연결 실패")
	defer execAttach.Close()

	output, err = io.ReadAll(execAttach.Reader)
	s.Require().NoError(err, "환경 변수 확인 결과 읽기 실패")
	
	outputStr = string(output)
	s.Assert().Contains(outputStr, "PATH=", "PATH 환경 변수가 없음")
	s.T().Log("환경 변수 확인 테스트 완료")
}

// TestVolumeManagement는 볼륨 관리를 테스트합니다
func (s *DockerIntegrationTestSuite) TestVolumeManagement() {
	ctx := context.Background()

	// 1. 볼륨 생성
	volumeName := fmt.Sprintf("test-volume-%d", time.Now().Unix())
	_, err := s.dockerClient.VolumeCreate(ctx, volume.CreateOptions{
		Name:   volumeName,
		Driver: "local",
		Labels: map[string]string{
			"test.type":    "integration",
			"test.purpose": "volume-management",
		},
	})
	s.Require().NoError(err, "볼륨 생성 실패")
	s.testVolumeNames = append(s.testVolumeNames, volumeName)
	s.T().Logf("볼륨 생성 완료: %s", volumeName)

	// 2. 볼륨 검사
	volumeInfo, err := s.dockerClient.VolumeInspect(ctx, volumeName)
	s.Require().NoError(err, "볼륨 검사 실패")
	s.Assert().Equal(volumeName, volumeInfo.Name, "볼륨 이름이 다름")
	s.Assert().Equal("local", volumeInfo.Driver, "볼륨 드라이버가 다름")

	// 3. 볼륨을 사용하는 컨테이너 생성
	containerName := fmt.Sprintf("test-volume-container-%d", time.Now().Unix())
	createResp, err := s.dockerClient.ContainerCreate(ctx,
		&container.Config{
			Image: "alpine:latest",
			Cmd:   []string{"sleep", "30"},
			Labels: map[string]string{
				"test.type":    "integration",
				"test.purpose": "volume-test",
			},
		},
		&container.HostConfig{
			Mounts: []mount.Mount{
				{
					Type:   mount.TypeVolume,
					Source: volumeName,
					Target: "/data",
				},
			},
		},
		nil,
		nil,
		containerName,
	)
	s.Require().NoError(err, "볼륨 테스트 컨테이너 생성 실패")
	
	containerID := createResp.ID
	s.testContainerIDs = append(s.testContainerIDs, containerID)

	err = s.dockerClient.ContainerStart(ctx, containerID, types.ContainerStartOptions{})
	s.Require().NoError(err, "볼륨 테스트 컨테이너 시작 실패")

	// 4. 볼륨에 데이터 쓰기
	s.Eventually(func() bool {
		inspect, err := s.dockerClient.ContainerInspect(ctx, containerID)
		return err == nil && inspect.State.Running
	}, 10*time.Second, 500*time.Millisecond, "볼륨 테스트 컨테이너가 실행 상태가 되지 않음")

	execConfig := types.ExecConfig{
		Cmd:          []string{"sh", "-c", "echo 'volume test data' > /data/test.txt"},
		AttachStdout: true,
		AttachStderr: true,
	}

	execResp, err := s.dockerClient.ContainerExecCreate(ctx, containerID, execConfig)
	s.Require().NoError(err, "볼륨 쓰기 작업 준비 실패")

	execAttach, err := s.dockerClient.ContainerExecAttach(ctx, execResp.ID, types.ExecStartCheck{})
	s.Require().NoError(err, "볼륨 쓰기 작업 연결 실패")
	defer execAttach.Close()

	// 5. 볼륨에서 데이터 읽기
	execConfig = types.ExecConfig{
		Cmd:          []string{"cat", "/data/test.txt"},
		AttachStdout: true,
		AttachStderr: true,
	}

	execResp, err = s.dockerClient.ContainerExecCreate(ctx, containerID, execConfig)
	s.Require().NoError(err, "볼륨 읽기 작업 준비 실패")

	execAttach, err = s.dockerClient.ContainerExecAttach(ctx, execResp.ID, types.ExecStartCheck{})
	s.Require().NoError(err, "볼륨 읽기 작업 연결 실패")
	defer execAttach.Close()

	output, err := io.ReadAll(execAttach.Reader)
	s.Require().NoError(err, "볼륨 읽기 결과 읽기 실패")
	
	outputStr := string(output)
	s.Assert().Contains(outputStr, "volume test data", "볼륨 데이터가 예상과 다름")
	s.T().Log("볼륨 데이터 읽기/쓰기 테스트 완료")
}

// TestNetworkConnectivity는 네트워크 연결을 테스트합니다
func (s *DockerIntegrationTestSuite) TestNetworkConnectivity() {
	ctx := context.Background()

	// 1. 첫 번째 컨테이너 생성 (서버 역할)
	serverName := fmt.Sprintf("test-server-%d", time.Now().Unix())
	serverResp, err := s.dockerClient.ContainerCreate(ctx,
		&container.Config{
			Image: "alpine:latest",
			Cmd:   []string{"sh", "-c", "while true; do echo 'Server running' && sleep 1; done"},
			Labels: map[string]string{
				"test.type":    "integration",
				"test.purpose": "network-server",
			},
		},
		&container.HostConfig{
			NetworkMode: container.NetworkMode(s.testNetworkID),
		},
		&network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{
				s.testNetworkID: {},
			},
		},
		nil,
		serverName,
	)
	s.Require().NoError(err, "서버 컨테이너 생성 실패")
	
	serverID := serverResp.ID
	s.testContainerIDs = append(s.testContainerIDs, serverID)

	err = s.dockerClient.ContainerStart(ctx, serverID, types.ContainerStartOptions{})
	s.Require().NoError(err, "서버 컨테이너 시작 실패")

	// 2. 두 번째 컨테이너 생성 (클라이언트 역할)
	clientName := fmt.Sprintf("test-client-%d", time.Now().Unix())
	clientResp, err := s.dockerClient.ContainerCreate(ctx,
		&container.Config{
			Image: "alpine:latest",
			Cmd:   []string{"sleep", "30"},
			Labels: map[string]string{
				"test.type":    "integration",
				"test.purpose": "network-client",
			},
		},
		&container.HostConfig{
			NetworkMode: container.NetworkMode(s.testNetworkID),
		},
		&network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{
				s.testNetworkID: {},
			},
		},
		nil,
		clientName,
	)
	s.Require().NoError(err, "클라이언트 컨테이너 생성 실패")
	
	clientID := clientResp.ID
	s.testContainerIDs = append(s.testContainerIDs, clientID)

	err = s.dockerClient.ContainerStart(ctx, clientID, types.ContainerStartOptions{})
	s.Require().NoError(err, "클라이언트 컨테이너 시작 실패")

	// 3. 컨테이너들이 실행 상태가 될 때까지 대기
	s.Eventually(func() bool {
		serverInspect, err1 := s.dockerClient.ContainerInspect(ctx, serverID)
		clientInspect, err2 := s.dockerClient.ContainerInspect(ctx, clientID)
		return err1 == nil && err2 == nil && 
			   serverInspect.State.Running && clientInspect.State.Running
	}, 15*time.Second, 1*time.Second, "네트워크 테스트 컨테이너들이 실행 상태가 되지 않음")

	// 4. 네트워크 연결 테스트 (ping)
	execConfig := types.ExecConfig{
		Cmd:          []string{"ping", "-c", "3", serverName},
		AttachStdout: true,
		AttachStderr: true,
	}

	execResp, err := s.dockerClient.ContainerExecCreate(ctx, clientID, execConfig)
	s.Require().NoError(err, "ping 테스트 준비 실패")

	execAttach, err := s.dockerClient.ContainerExecAttach(ctx, execResp.ID, types.ExecStartCheck{})
	s.Require().NoError(err, "ping 테스트 연결 실패")
	defer execAttach.Close()

	output, err := io.ReadAll(execAttach.Reader)
	s.Require().NoError(err, "ping 테스트 결과 읽기 실패")
	
	outputStr := string(output)
	s.Assert().Contains(outputStr, "3 packets transmitted", "ping 테스트 결과가 예상과 다름")
	s.T().Log("네트워크 연결 테스트 (ping) 완료")

	// 5. DNS 해상도 테스트
	execConfig = types.ExecConfig{
		Cmd:          []string{"nslookup", serverName},
		AttachStdout: true,
		AttachStderr: true,
	}

	execResp, err = s.dockerClient.ContainerExecCreate(ctx, clientID, execConfig)
	s.Require().NoError(err, "DNS 테스트 준비 실패")

	execAttach, err = s.dockerClient.ContainerExecAttach(ctx, execResp.ID, types.ExecStartCheck{})
	s.Require().NoError(err, "DNS 테스트 연결 실패")
	defer execAttach.Close()

	output, err = io.ReadAll(execAttach.Reader)
	s.Require().NoError(err, "DNS 테스트 결과 읽기 실패")
	
	outputStr = string(output)
	s.Assert().Contains(outputStr, serverName, "DNS 해상도 테스트 결과가 예상과 다름")
	s.T().Log("DNS 해상도 테스트 완료")
}

// TestDockerIntegrationTestSuite는 Docker 통합 테스트 스위트를 실행합니다
func TestDockerIntegrationTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Docker 통합 테스트는 short 모드에서 스킵됩니다")
	}

	// Docker 통합 테스트 환경 변수 확인
	if os.Getenv("INTEGRATION_TEST") != "1" {
		t.Skip("INTEGRATION_TEST 환경 변수가 설정되지 않음")
	}

	suite.Run(t, new(DockerIntegrationTestSuite))
}