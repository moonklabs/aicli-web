package docker

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/aicli/aicli-web/internal/models"
	mountpkg "github.com/aicli/aicli-web/internal/docker/mount"
)

// CreateWorkspaceContainerWithMounts 마운트 관리자를 사용하여 워크스페이스 컨테이너를 생성합니다.
func (cm *ContainerManager) CreateWorkspaceContainerWithMounts(
	ctx context.Context,
	workspace *models.Workspace,
	req *CreateContainerRequest,
	mountManager *MountManager,
	additionalMounts []*mountpkg.CreateMountRequest,
) (*WorkspaceContainer, error) {
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
	
	// 마운트 생성
	mounts, err := mountManager.CreateMountsForContainer(workspace, additionalMounts)
	if err != nil {
		return nil, fmt.Errorf("create mounts for container: %w", err)
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
	
	// 호스트 설정 (개선된 마운트 사용)
	hostConfig := &container.HostConfig{
		// 마운트 매니저에서 생성된 마운트 사용
		Mounts: mounts,
		
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
	networkingConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			cm.client.config.NetworkName: {
				NetworkID: cm.client.networkID,
			},
		},
	}
	
	// 컨테이너 생성
	resp, err := cm.client.cli.ContainerCreate(
		ctx,
		config,
		hostConfig,
		networkingConfig,
		nil,
		containerName,
	)
	if err != nil {
		return nil, fmt.Errorf("create container: %w", err)
	}
	
	// 생성된 컨테이너 정보 조회
	containerInfo, err := cm.client.cli.ContainerInspect(ctx, resp.ID)
	if err != nil {
		// 컨테이너 생성은 성공했지만 정보 조회 실패 시 정리
		_ = cm.client.cli.ContainerRemove(ctx, resp.ID, container.RemoveOptions{Force: true})
		return nil, fmt.Errorf("inspect created container: %w", err)
	}
	
	// 시간 문자열 파싱
	createdTime, _ := time.Parse(time.RFC3339Nano, containerInfo.Created)
	
	// WorkspaceContainer 생성
	workspaceContainer := &WorkspaceContainer{
		ID:          containerInfo.ID,
		Name:        containerInfo.Name,
		WorkspaceID: req.WorkspaceID,
		State:       ContainerState(containerInfo.State.Status),
		Created:     createdTime,
	}
	
	// 상태별 추가 정보 설정
	if containerInfo.State.Running {
		if startedTime, err := time.Parse(time.RFC3339Nano, containerInfo.State.StartedAt); err == nil {
			workspaceContainer.Started = &startedTime
		}
	}
	if containerInfo.State.FinishedAt != "" {
		if finishedTime, err := time.Parse(time.RFC3339Nano, containerInfo.State.FinishedAt); err == nil {
			workspaceContainer.Finished = &finishedTime
		}
	}
	if containerInfo.State.ExitCode != 0 {
		workspaceContainer.ExitCode = &containerInfo.State.ExitCode
	}
	
	// 포트 정보 설정
	workspaceContainer.Ports = extractPortBindings(containerInfo.NetworkSettings)
	
	// 마운트 정보 설정
	workspaceContainer.Mounts = extractMountInfo(containerInfo.Mounts)
	
	return workspaceContainer, nil
}

// extractPortBindings는 네트워크 설정에서 포트 바인딩 정보를 추출합니다.
func extractPortBindings(settings *network.NetworkSettings) map[string]string {
	if settings == nil || settings.Ports == nil {
		return nil
	}
	
	ports := make(map[string]string)
	for containerPort, bindings := range settings.Ports {
		if len(bindings) > 0 && bindings[0].HostPort != "" {
			ports[string(containerPort)] = bindings[0].HostPort
		}
	}
	return ports
}

// extractMountInfo는 마운트 정보를 추출합니다.
func extractMountInfo(mounts []types.MountPoint) []ContainerMount {
	var containerMounts []ContainerMount
	for _, mount := range mounts {
		containerMounts = append(containerMounts, ContainerMount{
			Source:      mount.Source,
			Destination: mount.Destination,
			Mode:        mount.Mode,
			RW:          mount.RW,
			Type:        string(mount.Type),
		})
	}
	return containerMounts
}

// ValidateWorkspaceMounts 워크스페이스 마운트 설정을 검증합니다.
func (cm *ContainerManager) ValidateWorkspaceMounts(
	workspace *models.Workspace,
	mountManager *MountManager,
	additionalMounts []*mountpkg.CreateMountRequest,
) error {
	// 워크스페이스 기본 마운트 검증
	workspaceMount, err := mountManager.CreateWorkspaceMount(workspace)
	if err != nil {
		return fmt.Errorf("validate workspace mount: %w", err)
	}
	
	if err := mountManager.ValidateMountConfig(workspaceMount); err != nil {
		return fmt.Errorf("workspace mount config invalid: %w", err)
	}
	
	// 추가 마운트 검증
	for i, req := range additionalMounts {
		customMount, err := mountManager.CreateCustomMount(req)
		if err != nil {
			return fmt.Errorf("validate additional mount %d: %w", i, err)
		}
		
		if err := mountManager.ValidateMountConfig(customMount); err != nil {
			return fmt.Errorf("additional mount %d config invalid: %w", i, err)
		}
	}
	
	return nil
}

// GetContainerMountStatus 컨테이너의 마운트 상태를 조회합니다.
func (cm *ContainerManager) GetContainerMountStatus(
	ctx context.Context,
	containerID string,
	mountManager *MountManager,
) ([]*mountpkg.MountStatus, error) {
	// 컨테이너 정보 조회
	containerInfo, err := cm.client.cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("inspect container: %w", err)
	}
	
	var mountStatuses []*mountpkg.MountStatus
	
	// 각 마운트 포인트의 상태 확인
	for _, mountPoint := range containerInfo.Mounts {
		if mountPoint.Type != mount.TypeBind {
			continue // bind mount만 처리
		}
		
		// MountConfig 재구성 (실제로는 저장된 설정을 사용해야 함)
		config := &mountpkg.MountConfig{
			SourcePath: mountPoint.Source,
			TargetPath: mountPoint.Destination,
			ReadOnly:   !mountPoint.RW,
		}
		
		// 마운트 상태 조회
		status, err := mountManager.GetMountStatus(ctx, config)
		if err != nil {
			// 에러가 있어도 다른 마운트는 계속 처리
			status = &mountpkg.MountStatus{
				SourcePath: mountPoint.Source,
				TargetPath: mountPoint.Destination,
				Available:  false,
				Error:      err.Error(),
			}
		}
		
		mountStatuses = append(mountStatuses, status)
	}
	
	return mountStatuses, nil
}

// extractMountInfo 컨테이너의 마운트 정보를 추출합니다.
func (cm *ContainerManager) extractMountInfo(mounts []mount.Mount) []ContainerMount {
	var containerMounts []ContainerMount
	
	for _, m := range mounts {
		containerMount := ContainerMount{
			Source:      m.Source,
			Destination: m.Target,
			ReadOnly:    m.ReadOnly,
		}
		
		// 마운트 모드 설정
		if m.BindOptions != nil {
			containerMount.Mode = string(m.BindOptions.Propagation)
		}
		
		containerMounts = append(containerMounts, containerMount)
	}
	
	return containerMounts
}