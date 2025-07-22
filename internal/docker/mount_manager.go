package docker

import (
	"context"
	"fmt"

	dockermount "github.com/docker/docker/api/types/mount"
	"github.com/aicli/aicli-web/internal/docker/mount"
	"github.com/aicli/aicli-web/internal/models"
)

// MountManager Docker 마운트 관리를 담당합니다.
type MountManager struct {
	manager *mount.Manager
}

// NewMountManager 새로운 마운트 매니저를 생성합니다.
func NewMountManager() *MountManager {
	return &MountManager{
		manager: mount.NewManager(),
	}
}

// CreateWorkspaceMount 워크스페이스용 마운트를 생성합니다.
func (mm *MountManager) CreateWorkspaceMount(workspace *models.Workspace) (*mount.MountConfig, error) {
	return mm.manager.CreateWorkspaceMount(workspace)
}

// CreateCustomMount 사용자 정의 마운트를 생성합니다.
func (mm *MountManager) CreateCustomMount(req *mount.CreateMountRequest) (*mount.MountConfig, error) {
	return mm.manager.CreateCustomMount(req)
}

// ValidateMountConfig 마운트 설정을 검증합니다.
func (mm *MountManager) ValidateMountConfig(config *mount.MountConfig) error {
	return mm.manager.ValidateMountConfig(config)
}

// ToDockerMount Docker Mount 객체로 변환합니다.
func (mm *MountManager) ToDockerMount(config *mount.MountConfig) (dockermount.Mount, error) {
	return mm.manager.ToDockerMount(config)
}

// GetMountStatus 마운트 상태를 조회합니다.
func (mm *MountManager) GetMountStatus(ctx context.Context, config *mount.MountConfig) (*mount.MountStatus, error) {
	return mm.manager.GetMountStatus(ctx, config)
}

// RefreshMountConfig 마운트 설정을 갱신합니다.
func (mm *MountManager) RefreshMountConfig(config *mount.MountConfig) error {
	return mm.manager.RefreshMountConfig(config)
}

// StartFileWatcher 파일 변경 감시를 시작합니다.
func (mm *MountManager) StartFileWatcher(ctx context.Context, sourcePath string, excludePatterns []string, callback func([]string)) error {
	return mm.manager.StartFileWatcher(ctx, sourcePath, excludePatterns, callback)
}

// StopFileWatcher 파일 변경 감시를 중지합니다.
func (mm *MountManager) StopFileWatcher(sourcePath string) {
	mm.manager.StopFileWatcher(sourcePath)
}

// GetFileStats 파일 통계를 조회합니다.
func (mm *MountManager) GetFileStats(ctx context.Context, sourcePath string, excludePatterns []string) (*mount.FileStats, error) {
	return mm.manager.GetFileStats(ctx, sourcePath, excludePatterns)
}

// GetActiveWatchers 활성화된 파일 watcher 목록을 반환합니다.
func (mm *MountManager) GetActiveWatchers() []string {
	return mm.manager.GetActiveWatchers()
}

// StopAllWatchers 모든 파일 watcher를 중지합니다.
func (mm *MountManager) StopAllWatchers() {
	mm.manager.StopAllWatchers()
}

// CreateMountsForContainer 컨테이너를 위한 마운트 배열을 생성합니다.
func (mm *MountManager) CreateMountsForContainer(workspace *models.Workspace, additionalMounts []*mount.CreateMountRequest) ([]dockermount.Mount, error) {
	var mounts []dockermount.Mount
	
	// 워크스페이스 기본 마운트 생성
	workspaceMount, err := mm.CreateWorkspaceMount(workspace)
	if err != nil {
		return nil, fmt.Errorf("create workspace mount: %w", err)
	}
	
	dockerMount, err := mm.ToDockerMount(workspaceMount)
	if err != nil {
		return nil, fmt.Errorf("convert to docker mount: %w", err)
	}
	
	mounts = append(mounts, dockerMount)
	
	// 추가 마운트 생성
	for _, req := range additionalMounts {
		customMount, err := mm.CreateCustomMount(req)
		if err != nil {
			return nil, fmt.Errorf("create custom mount for %s: %w", req.SourcePath, err)
		}
		
		dockerMount, err := mm.ToDockerMount(customMount)
		if err != nil {
			return nil, fmt.Errorf("convert custom mount to docker mount: %w", err)
		}
		
		mounts = append(mounts, dockerMount)
	}
	
	return mounts, nil
}