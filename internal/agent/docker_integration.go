package agent

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"

	"github.com/aicli/aicli-web/internal/models"
)

const (
	// Docker 이미지 설정
	DefaultAgentImage = "aicli-agent:latest"
	AgentWorkDir      = "/workspace"
	AgentHomeDir      = "/home/agent"

	// 컨테이너 라벨
	LabelAgentID   = "aicli.agent.id"
	LabelProjectID = "aicli.project.id"
	LabelAgentType = "aicli.agent.type"
	LabelCreatedAt = "aicli.created.at"

	// 리소스 제한 기본값
	DefaultCPULimit      = 2.0 // 2 CPU cores
	DefaultMemoryLimit   = 4   // 4GB
	DefaultCPUReserve    = 0.5 // 0.5 CPU cores
	DefaultMemoryReserve = 512 // 512MB
)

// DockerAgentIntegration Docker 에이전트 통합 구현
type DockerAgentIntegration struct {
	client      *client.Client
	imageName   string
	networkName string
}

// NewDockerAgentIntegration 새로운 Docker 에이전트 통합 인스턴스 생성
func NewDockerAgentIntegration(dockerClient *client.Client) (*DockerAgentIntegration, error) {
	if dockerClient == nil {
		return nil, fmt.Errorf("docker client is required")
	}

	return &DockerAgentIntegration{
		client:      dockerClient,
		imageName:   DefaultAgentImage,
		networkName: "aicli-agent-network",
	}, nil
}

// CreateAgentContainer 에이전트 컨테이너 생성
func (d *DockerAgentIntegration) CreateAgentContainer(ctx context.Context, agent *models.Agent, worktreePath string) (string, error) {
	// 컨테이너 이름 생성
	containerName := fmt.Sprintf("aicli-agent-%s", agent.ID)

	// 환경 변수 설정
	env := []string{
		fmt.Sprintf("AGENT_ID=%s", agent.ID),
		fmt.Sprintf("PROJECT_ID=%s", agent.ProjectID),
		fmt.Sprintf("AGENT_NAME=%s", agent.Name),
		"TZ=Asia/Seoul",
	}

	// 환경 변수에서 설정 추가
	if agent.Config.Environment != nil {
		for k, v := range agent.Config.Environment {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	// 마운트 설정
	mounts := []mount.Mount{
		{
			Type:   mount.TypeBind,
			Source: worktreePath,
			Target: AgentWorkDir,
			BindOptions: &mount.BindOptions{
				Propagation: mount.PropagationRPrivate,
			},
		},
	}

	// 볼륨 마운트 (영구 데이터)
	claudeDataVolume := fmt.Sprintf("aicli-agent-%s-claude", agent.ID)
	mounts = append(mounts, mount.Mount{
		Type:   mount.TypeVolume,
		Source: claudeDataVolume,
		Target: filepath.Join(AgentHomeDir, ".claude"),
	})

	// 컨테이너 설정
	config := &container.Config{
		Image:      d.imageName,
		Hostname:   containerName,
		Env:        env,
		WorkingDir: AgentWorkDir,
		Labels: map[string]string{
			LabelAgentID:   agent.ID,
			LabelProjectID: agent.ProjectID,
			LabelAgentType: string(agent.Type),
			LabelCreatedAt: time.Now().Format(time.RFC3339),
		},
		Tty:          true,
		OpenStdin:    true,
		StdinOnce:    false,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          []string{"sleep"}, // 컨테이너를 실행 상태로 유지
	}

	// 호스트 설정
	hostConfig := &container.HostConfig{
		Mounts:      mounts,
		NetworkMode: container.NetworkMode(d.networkName),
		RestartPolicy: container.RestartPolicy{
			Name: "unless-stopped",
		},
		Resources: container.Resources{
			CPUQuota:          int64(DefaultCPULimit * 100000), // CPU 제한
			CPUPeriod:         100000,
			Memory:            int64(DefaultMemoryLimit) << 30,   // 메모리 제한 (GB to bytes)
			MemoryReservation: int64(DefaultMemoryReserve) << 20, // 메모리 예약 (MB to bytes)
		},
		AutoRemove: false,
	}

	// 네트워크 설정
	networkConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			d.networkName: {
				Aliases: []string{containerName},
			},
		},
	}

	// 컨테이너 생성
	resp, err := d.client.ContainerCreate(ctx, config, hostConfig, networkConfig, nil, containerName)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	return resp.ID, nil
}

// StartAgentContainer 에이전트 컨테이너 시작
func (d *DockerAgentIntegration) StartAgentContainer(ctx context.Context, containerID string) error {
	// 컨테이너 시작
	if err := d.client.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	// 컨테이너가 실행 중인지 확인 (최대 10초 대기)
	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for container to start")
		case <-ticker.C:
			inspect, err := d.client.ContainerInspect(ctx, containerID)
			if err != nil {
				return fmt.Errorf("failed to inspect container: %w", err)
			}

			if inspect.State.Running {
				return nil
			}

			if inspect.State.ExitCode != 0 {
				return fmt.Errorf("container exited with code %d: %s", inspect.State.ExitCode, inspect.State.Error)
			}
		}
	}
}

// StopAgentContainer 에이전트 컨테이너 중지
func (d *DockerAgentIntegration) StopAgentContainer(ctx context.Context, containerID string) error {
	// 정상 종료 시간 (30초)
	timeout := 30

	// 컨테이너 중지
	if err := d.client.ContainerStop(ctx, containerID, container.StopOptions{
		Timeout: &timeout,
	}); err != nil {
		// 이미 중지된 경우는 무시
		if !strings.Contains(err.Error(), "not running") {
			return fmt.Errorf("failed to stop container: %w", err)
		}
	}

	return nil
}

// RemoveAgentContainer 에이전트 컨테이너 제거
func (d *DockerAgentIntegration) RemoveAgentContainer(ctx context.Context, containerID string) error {
	// 컨테이너 제거
	if err := d.client.ContainerRemove(ctx, containerID, container.RemoveOptions{
		RemoveVolumes: false, // 볼륨은 보존
		Force:         true,  // 실행 중이어도 강제 제거
	}); err != nil {
		// 이미 제거된 경우는 무시
		if !strings.Contains(err.Error(), "No such container") {
			return fmt.Errorf("failed to remove container: %w", err)
		}
	}

	return nil
}

// GetAgentContainerLogs 에이전트 컨테이너 로그 가져오기
func (d *DockerAgentIntegration) GetAgentContainerLogs(ctx context.Context, containerID string, options types.ContainerLogsOptions) (io.ReadCloser, error) {
	// 로그 옵션 설정
	if !options.ShowStdout {
		options.ShowStdout = true
	}
	if !options.ShowStderr {
		options.ShowStderr = true
	}

	// 로그 스트림 가져오기
	logs, err := d.client.ContainerLogs(ctx, containerID, options)
	if err != nil {
		return nil, fmt.Errorf("failed to get container logs: %w", err)
	}

	return logs, nil
}

// ExecInAgentContainer 에이전트 컨테이너에서 명령 실행
func (d *DockerAgentIntegration) ExecInAgentContainer(ctx context.Context, containerID string, cmd []string) (string, error) {
	// Exec 구성
	execConfig := types.ExecConfig{
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          cmd,
		WorkingDir:   AgentWorkDir,
		User:         "agent",
	}

	// Exec 생성
	execID, err := d.client.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return "", fmt.Errorf("failed to create exec: %w", err)
	}

	// Exec 시작
	resp, err := d.client.ContainerExecAttach(ctx, execID.ID, types.ExecStartCheck{})
	if err != nil {
		return "", fmt.Errorf("failed to attach to exec: %w", err)
	}
	defer resp.Close()

	// 출력 읽기
	output, err := io.ReadAll(resp.Reader)
	if err != nil {
		return "", fmt.Errorf("failed to read exec output: %w", err)
	}

	// Exec 상태 확인
	execInspect, err := d.client.ContainerExecInspect(ctx, execID.ID)
	if err != nil {
		return "", fmt.Errorf("failed to inspect exec: %w", err)
	}

	if execInspect.ExitCode != 0 {
		return string(output), fmt.Errorf("command exited with code %d", execInspect.ExitCode)
	}

	return string(output), nil
}

// EnsureAgentNetwork 에이전트 네트워크 확인 및 생성
func (d *DockerAgentIntegration) EnsureAgentNetwork(ctx context.Context) error {
	// 네트워크 존재 확인
	networks, err := d.client.NetworkList(ctx, types.NetworkListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list networks: %w", err)
	}

	for _, net := range networks {
		if net.Name == d.networkName {
			return nil // 이미 존재
		}
	}

	// 네트워크 생성
	_, err = d.client.NetworkCreate(ctx, d.networkName, types.NetworkCreate{
		Driver: "bridge",
		Labels: map[string]string{
			"aicli.network.type": "agent",
		},
		Options: map[string]string{
			"com.docker.network.bridge.name": "aicli-agent-br",
		},
	})

	if err != nil {
		return fmt.Errorf("failed to create network: %w", err)
	}

	return nil
}

// CleanupAgentVolumes 에이전트 볼륨 정리
func (d *DockerAgentIntegration) CleanupAgentVolumes(ctx context.Context, agentID string) error {
	volumeName := fmt.Sprintf("aicli-agent-%s-claude", agentID)

	// 볼륨 제거
	if err := d.client.VolumeRemove(ctx, volumeName, true); err != nil {
		// 볼륨이 없는 경우는 무시
		if !strings.Contains(err.Error(), "no such volume") {
			return fmt.Errorf("failed to remove volume: %w", err)
		}
	}

	return nil
}
