package mount

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/docker/docker/api/types/mount"
	"github.com/aicli/aicli-web/internal/models"
)

// Manager 마운트 관리를 담당합니다.
type Manager struct {
	validator *Validator
	syncer    *Syncer
}

// NewManager 새로운 마운트 매니저를 생성합니다.
func NewManager() *Manager {
	return &Manager{
		validator: NewValidator(),
		syncer:    NewSyncer(),
	}
}

// MountConfig 마운트 구성 정보를 담습니다.
type MountConfig struct {
	// 경로 설정
	SourcePath      string            `json:"source_path"`      // 로컬 프로젝트 경로
	TargetPath      string            `json:"target_path"`      // 컨테이너 내부 경로
	
	// 권한 설정
	ReadOnly        bool              `json:"read_only"`        // 읽기 전용 마운트
	UserID          int               `json:"user_id"`          // 마운트 소유자 UID
	GroupID         int               `json:"group_id"`         // 마운트 그룹 GID
	
	// 동기화 옵션
	SyncMode        SyncMode          `json:"sync_mode"`        // 동기화 모드
	ExcludePatterns []string          `json:"exclude_patterns"` // 제외 패턴
	IncludePatterns []string          `json:"include_patterns"` // 포함 패턴
	
	// 보안 설정
	NoExec          bool              `json:"no_exec"`          // 실행 권한 제거
	NoSuid          bool              `json:"no_suid"`          // SUID 비트 무시
	NoDev           bool              `json:"no_dev"`           // 디바이스 파일 접근 차단
	
	// 메타데이터
	WorkspaceID     string            `json:"workspace_id"`     // 연결된 워크스페이스 ID
	CreatedAt       time.Time         `json:"created_at"`       // 생성 시간
	LastChecked     time.Time         `json:"last_checked"`     // 마지막 검증 시간
}

// CreateWorkspaceMount 워크스페이스용 마운트 설정을 생성합니다.
func (m *Manager) CreateWorkspaceMount(workspace *models.Workspace) (*MountConfig, error) {
	// 경로 검증
	if err := m.validator.ValidateProjectPath(workspace.ProjectPath); err != nil {
		return nil, fmt.Errorf("invalid project path: %w", err)
	}
	
	// 절대 경로 변환
	absPath, err := filepath.Abs(workspace.ProjectPath)
	if err != nil {
		return nil, fmt.Errorf("resolve absolute path: %w", err)
	}
	
	// 타겟 경로 검증
	targetPath := "/workspace"
	if err := m.validator.ValidateMountPath(absPath, targetPath); err != nil {
		return nil, fmt.Errorf("mount path validation failed: %w", err)
	}
	
	// 동기화 모드 최적화 확인
	canOptimize, syncMode := m.validator.CanOptimizeMount(absPath)
	if !canOptimize {
		syncMode = SyncModeNative
	}
	
	config := &MountConfig{
		SourcePath:      absPath,
		TargetPath:      targetPath,
		ReadOnly:        false, // 기본적으로 쓰기 가능
		UserID:          1000,  // 비특권 사용자
		GroupID:         1000,  // 비특권 그룹
		SyncMode:        syncMode,
		NoExec:          false,
		NoSuid:          true,  // 보안상 SUID 비활성화
		NoDev:           true,  // 디바이스 파일 접근 차단
		ExcludePatterns: m.getDefaultExcludePatterns(),
		WorkspaceID:     workspace.ID,
		CreatedAt:       time.Now(),
		LastChecked:     time.Now(),
	}
	
	return config, nil
}

// CreateCustomMount 사용자 정의 마운트 설정을 생성합니다.
func (m *Manager) CreateCustomMount(req *CreateMountRequest) (*MountConfig, error) {
	// 경로 검증
	if err := m.validator.ValidateMountPath(req.SourcePath, req.TargetPath); err != nil {
		return nil, fmt.Errorf("mount path validation failed: %w", err)
	}
	
	// 절대 경로 변환
	absSourcePath, err := filepath.Abs(req.SourcePath)
	if err != nil {
		return nil, fmt.Errorf("resolve source absolute path: %w", err)
	}
	
	// 동기화 모드 확인
	syncMode := req.SyncMode
	if syncMode == "" {
		canOptimize, optimizedMode := m.validator.CanOptimizeMount(absSourcePath)
		if canOptimize {
			syncMode = optimizedMode
		} else {
			syncMode = SyncModeNative
		}
	}
	
	config := &MountConfig{
		SourcePath:      absSourcePath,
		TargetPath:      req.TargetPath,
		ReadOnly:        req.ReadOnly,
		UserID:          req.UserID,
		GroupID:         req.GroupID,
		SyncMode:        syncMode,
		NoExec:          req.NoExec,
		NoSuid:          req.NoSuid,
		NoDev:           req.NoDev,
		ExcludePatterns: req.ExcludePatterns,
		IncludePatterns: req.IncludePatterns,
		WorkspaceID:     req.WorkspaceID,
		CreatedAt:       time.Now(),
		LastChecked:     time.Now(),
	}
	
	// 기본 제외 패턴 추가
	if len(config.ExcludePatterns) == 0 {
		config.ExcludePatterns = m.getDefaultExcludePatterns()
	} else {
		// 사용자 패턴에 기본 패턴 추가
		defaultPatterns := m.getDefaultExcludePatterns()
		config.ExcludePatterns = append(config.ExcludePatterns, defaultPatterns...)
	}
	
	return config, nil
}

// getDefaultExcludePatterns 기본 제외 패턴을 반환합니다.
func (m *Manager) getDefaultExcludePatterns() []string {
	return []string{
		// VCS 디렉토리
		".git",
		".svn",
		".hg",
		".bzr",
		
		// IDE 파일
		".vscode",
		".idea",
		"*.swp",
		"*.swo",
		"*~",
		
		// 빌드 결과물
		"node_modules",
		"dist",
		"build",
		"target",
		"bin",
		"obj",
		"out",
		
		// 로그 및 임시 파일
		"*.log",
		"logs",
		"*.tmp",
		"*.temp",
		".DS_Store",
		"Thumbs.db",
		
		// 패키지 매니저 캐시
		".npm",
		".yarn",
		".pnpm-store",
		".cargo",
		".m2",
		
		// 언어별 캐시/빌드
		"__pycache__",
		"*.pyc",
		".pytest_cache",
		".coverage",
		"vendor",
		"Gemfile.lock",
		
		// OS 관련
		"desktop.ini",
		".directory",
		
		// Docker 관련
		".dockerignore",
		"Dockerfile.*",
		"docker-compose.*.yml",
	}
}

// ValidateMountConfig 마운트 설정의 유효성을 검사합니다.
func (m *Manager) ValidateMountConfig(config *MountConfig) error {
	if config == nil {
		return fmt.Errorf("mount config is required")
	}
	
	// 필수 필드 검증
	if config.SourcePath == "" {
		return fmt.Errorf("source path is required")
	}
	
	if config.TargetPath == "" {
		return fmt.Errorf("target path is required")
	}
	
	// 경로 검증
	if err := m.validator.ValidateMountPath(config.SourcePath, config.TargetPath); err != nil {
		return fmt.Errorf("mount path validation failed: %w", err)
	}
	
	// 권한 설정 검증
	if config.UserID < 0 {
		return fmt.Errorf("invalid user ID: %d", config.UserID)
	}
	
	if config.GroupID < 0 {
		return fmt.Errorf("invalid group ID: %d", config.GroupID)
	}
	
	// 동기화 모드 검증
	validSyncModes := []SyncMode{SyncModeNative, SyncModeOptimized, SyncModeCached, SyncModeDelegated}
	valid := false
	for _, mode := range validSyncModes {
		if config.SyncMode == mode {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid sync mode: %s", config.SyncMode)
	}
	
	return nil
}

// ToDockerMount Docker Mount 객체로 변환합니다.
func (m *Manager) ToDockerMount(config *MountConfig) (mount.Mount, error) {
	// 설정 검증
	if err := m.ValidateMountConfig(config); err != nil {
		return mount.Mount{}, fmt.Errorf("invalid mount config: %w", err)
	}
	
	mountObj := mount.Mount{
		Type:   mount.TypeBind,
		Source: config.SourcePath,
		Target: config.TargetPath,
		BindOptions: &mount.BindOptions{
			Propagation: mount.PropagationRPrivate,
		},
	}
	
	// 읽기 전용 설정
	if config.ReadOnly {
		mountObj.ReadOnly = true
	}
	
	// 동기화 모드 설정
	consistency := m.syncModeToConsistency(config.SyncMode)
	if consistency != "" {
		mountObj.Consistency = mount.Consistency(consistency)
	}
	
	return mountObj, nil
}

// syncModeToConsistency 동기화 모드를 Docker Consistency로 변환합니다.
func (m *Manager) syncModeToConsistency(mode SyncMode) string {
	switch mode {
	case SyncModeCached:
		return "cached"
	case SyncModeDelegated:
		return "delegated"
	default:
		return "" // native/optimized는 기본 일관성 사용
	}
}

// GetMountStatus 마운트 상태를 조회합니다.
func (m *Manager) GetMountStatus(ctx context.Context, config *MountConfig) (*MountStatus, error) {
	status := &MountStatus{
		SourcePath: config.SourcePath,
		TargetPath: config.TargetPath,
		CheckedAt:  time.Now(),
	}
	
	// 소스 경로 존재 확인
	if _, err := os.Stat(config.SourcePath); err != nil {
		status.Available = false
		status.Error = fmt.Sprintf("source path not accessible: %v", err)
		return status, nil
	}
	
	// 권한 확인
	if err := m.validator.checkAccess(config.SourcePath); err != nil {
		status.Available = false
		status.Error = fmt.Sprintf("access check failed: %v", err)
		return status, nil
	}
	
	// 디스크 사용량 조회
	if usage, err := m.validator.GetDiskUsage(config.SourcePath); err == nil {
		status.DiskUsage = usage
	}
	
	// 동기화 상태 확인 (향후 구현)
	if syncStatus, err := m.syncer.CheckMountStatus(ctx, "", config.SourcePath); err == nil {
		status.SyncStatus = syncStatus
	}
	
	status.Available = true
	return status, nil
}

// RefreshMountConfig 마운트 설정을 갱신합니다.
func (m *Manager) RefreshMountConfig(config *MountConfig) error {
	// 경로 재검증
	if err := m.validator.ValidateMountPath(config.SourcePath, config.TargetPath); err != nil {
		return fmt.Errorf("mount path revalidation failed: %w", err)
	}
	
	// 동기화 모드 재최적화
	canOptimize, syncMode := m.validator.CanOptimizeMount(config.SourcePath)
	if canOptimize && config.SyncMode == SyncModeNative {
		config.SyncMode = syncMode
	}
	
	config.LastChecked = time.Now()
	return nil
}

// CreateMountRequest 마운트 생성 요청 구조체입니다.
type CreateMountRequest struct {
	WorkspaceID     string            `json:"workspace_id"`
	SourcePath      string            `json:"source_path"`
	TargetPath      string            `json:"target_path"`
	ReadOnly        bool              `json:"read_only"`
	UserID          int               `json:"user_id"`
	GroupID         int               `json:"group_id"`
	SyncMode        SyncMode          `json:"sync_mode,omitempty"`
	ExcludePatterns []string          `json:"exclude_patterns,omitempty"`
	IncludePatterns []string          `json:"include_patterns,omitempty"`
	NoExec          bool              `json:"no_exec"`
	NoSuid          bool              `json:"no_suid"`
	NoDev           bool              `json:"no_dev"`
}

// StartFileWatcher 파일 변경 감시를 시작합니다.
func (m *Manager) StartFileWatcher(ctx context.Context, sourcePath string, excludePatterns []string, callback func([]string)) error {
	return m.syncer.WatchChanges(ctx, sourcePath, excludePatterns, callback)
}

// StopFileWatcher 파일 변경 감시를 중지합니다.
func (m *Manager) StopFileWatcher(sourcePath string) {
	m.syncer.StopWatch(sourcePath)
}

// GetFileStats 파일 통계를 조회합니다.
func (m *Manager) GetFileStats(ctx context.Context, sourcePath string, excludePatterns []string) (*FileStats, error) {
	return m.syncer.GetFileStats(ctx, sourcePath, excludePatterns)
}

// GetActiveWatchers 활성화된 파일 watcher 목록을 반환합니다.
func (m *Manager) GetActiveWatchers() []string {
	return m.syncer.GetActiveWatchers()
}

// StopAllWatchers 모든 파일 watcher를 중지합니다.
func (m *Manager) StopAllWatchers() {
	m.syncer.StopAll()
}

// MountStatus 마운트 상태 정보입니다.
type MountStatus struct {
	SourcePath  string        `json:"source_path"`
	TargetPath  string        `json:"target_path"`
	Available   bool          `json:"available"`
	Error       string        `json:"error,omitempty"`
	CheckedAt   time.Time     `json:"checked_at"`
	DiskUsage   *DiskUsage    `json:"disk_usage,omitempty"`
	SyncStatus  *SyncStatus   `json:"sync_status,omitempty"`
}