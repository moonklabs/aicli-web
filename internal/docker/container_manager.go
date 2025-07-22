package docker

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
)

// ContainerManager 컨테이너 생명주기를 관리합니다.
type ContainerManager struct {
	client *Client
}

// NewContainerManager 새로운 컨테이너 매니저를 생성합니다.
func NewContainerManager(client *Client) *ContainerManager {
	return &ContainerManager{
		client: client,
	}
}

// WorkspaceContainer 워크스페이스 컨테이너 정보를 담는 구조체입니다.
type WorkspaceContainer struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	WorkspaceID string                 `json:"workspace_id"`
	State       ContainerState         `json:"state"`
	Created     time.Time              `json:"created_at"`
	Started     *time.Time             `json:"started_at,omitempty"`
	Finished    *time.Time             `json:"finished_at,omitempty"`
	ExitCode    *int                   `json:"exit_code,omitempty"`
	Stats       *ContainerStats        `json:"stats,omitempty"`
	Ports       map[string]string      `json:"ports,omitempty"`
	Mounts      []ContainerMount       `json:"mounts,omitempty"`
}

// ContainerState 컨테이너 상태를 나타냅니다.
type ContainerState string

const (
	ContainerStateCreated    ContainerState = "created"
	ContainerStateRunning    ContainerState = "running"
	ContainerStateExited     ContainerState = "exited"
	ContainerStatePaused     ContainerState = "paused"
	ContainerStateRestarting ContainerState = "restarting"
	ContainerStateRemoving   ContainerState = "removing"
	ContainerStateDead       ContainerState = "dead"
)


// ContainerMount 컨테이너 마운트 정보입니다.
type ContainerMount struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Mode        string `json:"mode"`
	ReadOnly    bool   `json:"read_only"`
}

// CreateContainerRequest 컨테이너 생성 요청 구조체입니다.
type CreateContainerRequest struct {
	WorkspaceID   string            `json:"workspace_id"`
	Name          string            `json:"name"`
	ProjectPath   string            `json:"project_path"`
	Image         string            `json:"image,omitempty"`
	Command       []string          `json:"command,omitempty"`
	Environment   map[string]string `json:"environment,omitempty"`
	WorkingDir    string            `json:"working_dir,omitempty"`
	
	// 리소스 제한
	CPULimit      float64           `json:"cpu_limit,omitempty"`
	MemoryLimit   int64             `json:"memory_limit,omitempty"`
	
	// 네트워크 설정
	Ports         map[string]string `json:"ports,omitempty"`
	
	// 보안 설정
	Privileged    bool              `json:"privileged,omitempty"`
	ReadOnly      bool              `json:"read_only,omitempty"`
}

// CreateWorkspaceContainerRequest Docker 통합 서비스용 컨테이너 생성 요청
type CreateWorkspaceContainerRequest struct {
	WorkspaceID string                     `json:"workspace_id"`
	Name        string                     `json:"name"`
	Image       string                     `json:"image"`
	ProjectPath string                     `json:"project_path"`
	Environment map[string]string          `json:"environment,omitempty"`
	Isolation   interface{}                `json:"isolation,omitempty"` // security.WorkspaceIsolation 타입이지만 순환 참조 방지
	MemoryLimit   int64             `json:"memory_limit,omitempty"`
	
	// 네트워크 설정
	Ports         map[string]string `json:"ports,omitempty"`
	
	// 보안 설정
	Privileged    bool              `json:"privileged,omitempty"`
	ReadOnly      bool              `json:"read_only,omitempty"`
}

// CreateWorkspaceContainer 워크스페이스용 컨테이너를 생성합니다.
func (cm *ContainerManager) CreateWorkspaceContainer(ctx context.Context, req *CreateContainerRequest) (*WorkspaceContainer, error) {
	// 기본값 설정
	if req.Image == "" {
		req.Image = cm.client.config.DefaultImage
	}
	if req.WorkingDir == "" {
		req.WorkingDir = "/workspace"
	}
	
	containerName := cm.client.GenerateContainerName(req.WorkspaceID)
	
	// 기존 컨테이너 정리
	if err := cm.cleanupExistingContainer(ctx, containerName); err != nil {
		return nil, fmt.Errorf("cleanup existing container: %w", err)
	}
	
	// 컨테이너 설정
	config := &container.Config{
		Image:        req.Image,
		Cmd:          req.Command,
		WorkingDir:   req.WorkingDir,
		Hostname:     containerName,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		OpenStdin:    true,
		StdinOnce:    false,
		Tty:          true,
		Labels:       cm.client.WorkspaceLabels(req.WorkspaceID, req.Name),
	}
	
	// 환경 변수 설정
	config.Env = cm.buildEnvironment(req)
	
	// 호스트 설정
	hostConfig := &container.HostConfig{
		// 프로젝트 디렉토리 마운트
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: req.ProjectPath,
				Target: "/workspace",
				BindOptions: &mount.BindOptions{
					Propagation: mount.PropagationRPrivate,
				},
			},
		},
		
		// 리소스 제한
		Resources: cm.buildResourceLimits(req),
		
		// 보안 설정
		Privileged:     req.Privileged,
		ReadonlyRootfs: req.ReadOnly || cm.client.config.ReadOnly,
		SecurityOpt:    cm.client.config.SecurityOpts,
		CapDrop:        []string{"ALL"},
		CapAdd:         []string{"CHOWN", "SETUID", "SETGID", "DAC_OVERRIDE"},
		
		// 재시작 정책
		RestartPolicy: container.RestartPolicy{
			Name: "unless-stopped",
		},
		
		// 포트 매핑
		PortBindings: cm.buildPortBindings(req.Ports),
	}
	
	// 네트워크 설정
	networkConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			cm.client.config.NetworkName: {
				NetworkID: cm.client.GetNetworkID(),
			},
		},
	}
	
	// 컨테이너 생성
	resp, err := cm.client.cli.ContainerCreate(
		ctx, config, hostConfig, networkConfig, nil, containerName,
	)
	if err != nil {
		return nil, fmt.Errorf("create container: %w", err)
	}
	
	return &WorkspaceContainer{
		ID:          resp.ID,
		Name:        containerName,
		WorkspaceID: req.WorkspaceID,
		State:       ContainerStateCreated,
		Created:     time.Now(),
	}, nil
}

// buildEnvironment 환경 변수를 구성합니다.
func (cm *ContainerManager) buildEnvironment(req *CreateContainerRequest) []string {
	env := []string{
		fmt.Sprintf("WORKSPACE_ID=%s", req.WorkspaceID),
		fmt.Sprintf("WORKSPACE_NAME=%s", req.Name),
		fmt.Sprintf("AICLI_MANAGED=true"),
	}
	
	// 사용자 지정 환경 변수 추가
	for key, value := range req.Environment {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}
	
	return env
}

// buildResourceLimits 리소스 제한을 구성합니다.
func (cm *ContainerManager) buildResourceLimits(req *CreateContainerRequest) container.Resources {
	cpuLimit := req.CPULimit
	if cpuLimit == 0 {
		cpuLimit = cm.client.config.CPULimit
	}
	
	memLimit := req.MemoryLimit
	if memLimit == 0 {
		memLimit = cm.client.config.MemoryLimit
	}
	
	return container.Resources{
		CPUQuota:   int64(cpuLimit * 100000), // 1.0 = 100%
		CPUPeriod:  100000,
		Memory:     memLimit,
		MemorySwap: memLimit, // Swap 비활성화
		
		// PID 제한
		PidsLimit: func() *int64 { 
			limit := int64(100) // 최대 100개 프로세스
			return &limit
		}(),
	}
}

// buildPortBindings 포트 매핑을 구성합니다.
func (cm *ContainerManager) buildPortBindings(ports map[string]string) nat.PortMap {
	bindings := make(nat.PortMap)
	
	for containerPort, hostPort := range ports {
		port, err := nat.NewPort("tcp", containerPort)
		if err != nil {
			continue // 잘못된 포트는 무시
		}
		
		bindings[port] = []nat.PortBinding{
			{
				HostPort: hostPort,
			},
		}
	}
	
	return bindings
}

// StartContainer 컨테이너를 시작합니다.
func (cm *ContainerManager) StartContainer(ctx context.Context, containerID string) error {
	if err := cm.client.cli.ContainerStart(ctx, containerID, types.ContainerStartOptions{}); err != nil {
		return fmt.Errorf("start container: %w", err)
	}
	return nil
}

// StopContainer 컨테이너를 중지합니다.
func (cm *ContainerManager) StopContainer(ctx context.Context, containerID string, timeout time.Duration) error {
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	
	timeoutSeconds := int(timeout.Seconds())
	if err := cm.client.cli.ContainerStop(ctx, containerID, container.StopOptions{
		Timeout: &timeoutSeconds,
	}); err != nil {
		return fmt.Errorf("stop container: %w", err)
	}
	return nil
}

// RestartContainer 컨테이너를 재시작합니다.
func (cm *ContainerManager) RestartContainer(ctx context.Context, containerID string, timeout time.Duration) error {
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	
	timeoutSeconds := int(timeout.Seconds())
	if err := cm.client.cli.ContainerRestart(ctx, containerID, container.StopOptions{
		Timeout: &timeoutSeconds,
	}); err != nil {
		return fmt.Errorf("restart container: %w", err)
	}
	return nil
}

// RemoveContainer 컨테이너를 삭제합니다.
func (cm *ContainerManager) RemoveContainer(ctx context.Context, containerID string, force bool) error {
	options := types.ContainerRemoveOptions{
		RemoveVolumes: true,
		RemoveLinks:   true,
		Force:         force,
	}
	
	if err := cm.client.cli.ContainerRemove(ctx, containerID, options); err != nil {
		return fmt.Errorf("remove container: %w", err)
	}
	return nil
}

// InspectContainer 컨테이너 상태를 조회합니다.
func (cm *ContainerManager) InspectContainer(ctx context.Context, containerID string) (*WorkspaceContainer, error) {
	inspect, err := cm.client.cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("inspect container: %w", err)
	}
	
	return cm.inspectResultToWorkspaceContainer(&inspect), nil
}

// inspectResultToWorkspaceContainer 검사 결과를 워크스페이스 컨테이너로 변환합니다.
func (cm *ContainerManager) inspectResultToWorkspaceContainer(inspect *types.ContainerJSON) *WorkspaceContainer {
	// 생성 시간 파싱
	var createdTime time.Time
	if inspect.Created != "" {
		if parsedTime, err := time.Parse(time.RFC3339Nano, inspect.Created); err == nil {
			createdTime = parsedTime
		}
	}
	
	wc := &WorkspaceContainer{
		ID:          inspect.ID,
		Name:        inspect.Name,
		WorkspaceID: inspect.Config.Labels[cm.client.labelKey("workspace.id")],
		State:       ContainerState(inspect.State.Status),
		Created:     createdTime,
	}
	
	if inspect.State.StartedAt != "" {
		if startTime, err := time.Parse(time.RFC3339Nano, inspect.State.StartedAt); err == nil {
			wc.Started = &startTime
		}
	}
	
	if inspect.State.FinishedAt != "" {
		if finishTime, err := time.Parse(time.RFC3339Nano, inspect.State.FinishedAt); err == nil {
			wc.Finished = &finishTime
		}
	}
	
	if inspect.State.ExitCode != 0 {
		wc.ExitCode = &inspect.State.ExitCode
	}
	
	// 마운트 정보
	for _, mount := range inspect.Mounts {
		wc.Mounts = append(wc.Mounts, ContainerMount{
			Source:      mount.Source,
			Destination: mount.Destination,
			Mode:        string(mount.Mode),
			ReadOnly:    !mount.RW,
		})
	}
	
	return wc
}

// ListWorkspaceContainers 워크스페이스별 컨테이너 목록을 조회합니다.
func (cm *ContainerManager) ListWorkspaceContainers(ctx context.Context, workspaceID string) ([]*WorkspaceContainer, error) {
	filters := filters.NewArgs()
	filters.Add("label", fmt.Sprintf("%s.workspace.id=%s", cm.client.labelPrefix, workspaceID))
	
	containers, err := cm.client.cli.ContainerList(ctx, types.ContainerListOptions{
		All:     true,
		Filters: filters,
	})
	if err != nil {
		return nil, fmt.Errorf("list containers: %w", err)
	}
	
	result := make([]*WorkspaceContainer, 0, len(containers))
	for _, container := range containers {
		wc := &WorkspaceContainer{
			ID:          container.ID,
			Name:        container.Names[0],
			WorkspaceID: workspaceID,
			State:       ContainerState(container.State),
			Created:     time.Unix(container.Created, 0),
		}
		result = append(result, wc)
	}
	
	return result, nil
}

// CleanupWorkspace 워크스페이스의 모든 컨테이너를 정리합니다.
func (cm *ContainerManager) CleanupWorkspace(ctx context.Context, workspaceID string, force bool) error {
	containers, err := cm.ListWorkspaceContainers(ctx, workspaceID)
	if err != nil {
		return fmt.Errorf("list workspace containers: %w", err)
	}
	
	for _, container := range containers {
		// 실행 중이면 중지
		if container.State == ContainerStateRunning {
			if err := cm.StopContainer(ctx, container.ID, 10*time.Second); err != nil {
				if !force {
					return fmt.Errorf("stop container %s: %w", container.ID, err)
				}
			}
		}
		
		// 컨테이너 삭제
		if err := cm.RemoveContainer(ctx, container.ID, force); err != nil {
			if !force {
				return fmt.Errorf("remove container %s: %w", container.ID, err)
			}
		}
	}
	
	return nil
}

// cleanupExistingContainer 기존 컨테이너를 정리합니다.
func (cm *ContainerManager) cleanupExistingContainer(ctx context.Context, containerName string) error {
	// 이름으로 컨테이너 찾기
	containers, err := cm.client.cli.ContainerList(ctx, types.ContainerListOptions{
		All: true,
		Filters: filters.NewArgs(
			filters.Arg("name", containerName),
		),
	})
	
	if err != nil {
		return err
	}
	
	// 기존 컨테이너 삭제
	for _, container := range containers {
		if err := cm.RemoveContainer(ctx, container.ID, true); err != nil {
			return fmt.Errorf("remove existing container %s: %w", container.ID, err)
		}
	}
	
	return nil
}