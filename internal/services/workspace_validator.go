package services

import (
	"context"
	"strings"
	
	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/utils"
)

// WorkspaceValidator는 워크스페이스 검증 로직을 담당합니다
type WorkspaceValidator struct {
	maxNameLength      int
	minNameLength      int
	maxWorkspaceCount  int
	restrictedNames    []string
}

// NewWorkspaceValidator는 새로운 워크스페이스 검증기를 생성합니다
func NewWorkspaceValidator() *WorkspaceValidator {
	return &WorkspaceValidator{
		maxNameLength:     100,
		minNameLength:     1,
		maxWorkspaceCount: 50, // 사용자당 최대 워크스페이스 수
		restrictedNames: []string{
			"admin", "system", "root", "api", "test", "dev", "prod", "staging",
			"www", "mail", "ftp", "sql", "database", "config", "tmp", "temp",
			"bin", "sbin", "usr", "var", "opt", "etc", "home", "lib", "proc",
			"sys", "boot", "srv", "mnt", "media", "run", "dev", "null",
		},
	}
}

// ValidateCreate는 워크스페이스 생성 요청을 검증합니다
func (v *WorkspaceValidator) ValidateCreate(ctx context.Context, req *models.CreateWorkspaceRequest) error {
	if err := v.validateName(req.Name); err != nil {
		return err
	}
	
	if err := v.validateProjectPath(req.ProjectPath); err != nil {
		return err
	}
	
	if err := v.validateClaudeKey(req.ClaudeKey); err != nil {
		return err
	}
	
	return nil
}

// ValidateUpdate는 워크스페이스 수정 요청을 검증합니다
func (v *WorkspaceValidator) ValidateUpdate(ctx context.Context, req *models.UpdateWorkspaceRequest) error {
	if req.Name != "" {
		if err := v.validateName(req.Name); err != nil {
			return err
		}
	}
	
	if req.ProjectPath != "" {
		if err := v.validateProjectPath(req.ProjectPath); err != nil {
			return err
		}
	}
	
	if req.ClaudeKey != "" {
		if err := v.validateClaudeKey(req.ClaudeKey); err != nil {
			return err
		}
	}
	
	if req.Status != "" {
		if err := v.validateStatus(req.Status); err != nil {
			return err
		}
	}
	
	return nil
}

// ValidateWorkspace는 워크스페이스 전체를 검증합니다
func (v *WorkspaceValidator) ValidateWorkspace(ctx context.Context, workspace *models.Workspace) error {
	if workspace == nil {
		return NewWorkspaceError(ErrCodeInvalidRequest, "워크스페이스가 nil입니다", ErrInvalidRequest)
	}
	
	if err := v.validateName(workspace.Name); err != nil {
		return err
	}
	
	if err := v.validateProjectPath(workspace.ProjectPath); err != nil {
		return err
	}
	
	if err := v.validateStatus(workspace.Status); err != nil {
		return err
	}
	
	if workspace.OwnerID == "" {
		return NewWorkspaceError(ErrCodeInvalidRequest, "소유자 ID가 필요합니다", ErrInvalidRequest)
	}
	
	return nil
}

// validateName은 워크스페이스 이름을 검증합니다
func (v *WorkspaceValidator) validateName(name string) error {
	if name == "" {
		return NewWorkspaceError(ErrCodeInvalidName, "워크스페이스 이름이 필요합니다", ErrInvalidWorkspaceName)
	}
	
	if len(name) < v.minNameLength {
		return NewWorkspaceError(ErrCodeInvalidName, "워크스페이스 이름이 너무 짧습니다", ErrInvalidWorkspaceName)
	}
	
	if len(name) > v.maxNameLength {
		return NewWorkspaceError(ErrCodeInvalidName, "워크스페이스 이름이 너무 깁니다", ErrInvalidWorkspaceName)
	}
	
	// 공백 및 특수문자 검사
	trimmed := strings.TrimSpace(name)
	if trimmed != name {
		return NewWorkspaceError(ErrCodeInvalidName, "워크스페이스 이름에 앞뒤 공백이 포함될 수 없습니다", ErrInvalidWorkspaceName)
	}
	
	if strings.Contains(name, "  ") {
		return NewWorkspaceError(ErrCodeInvalidName, "워크스페이스 이름에 연속된 공백이 포함될 수 없습니다", ErrInvalidWorkspaceName)
	}
	
	// 금지된 문자 검사
	forbiddenChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|", "\n", "\r", "\t"}
	for _, char := range forbiddenChars {
		if strings.Contains(name, char) {
			return NewWorkspaceError(ErrCodeInvalidName, "워크스페이스 이름에 허용되지 않는 문자가 포함되어 있습니다", ErrInvalidWorkspaceName)
		}
	}
	
	// 예약어 검사
	lowerName := strings.ToLower(name)
	for _, restricted := range v.restrictedNames {
		if lowerName == restricted {
			return NewWorkspaceError(ErrCodeInvalidName, "사용할 수 없는 워크스페이스 이름입니다", ErrInvalidWorkspaceName)
		}
	}
	
	return nil
}

// validateProjectPath는 프로젝트 경로를 검증합니다
func (v *WorkspaceValidator) validateProjectPath(path string) error {
	if path == "" {
		return NewWorkspaceError(ErrCodeInvalidPath, "프로젝트 경로가 필요합니다", ErrInvalidProjectPath)
	}
	
	// utils 패키지의 검증 로직 사용
	if err := utils.IsValidProjectPath(path); err != nil {
		return NewWorkspaceError(ErrCodeInvalidPath, "프로젝트 경로가 유효하지 않습니다", err)
	}
	
	return nil
}

// validateClaudeKey는 Claude API 키를 검증합니다
func (v *WorkspaceValidator) validateClaudeKey(key string) error {
	if key == "" {
		return nil // 선택적 필드
	}
	
	// Claude API 키 형식 검증 (sk-ant-로 시작하는 형태)
	if !strings.HasPrefix(key, "sk-ant-") {
		return NewWorkspaceError(ErrCodeInvalidRequest, "올바르지 않은 Claude API 키 형식입니다", ErrInvalidRequest)
	}
	
	// 최소 길이 검증
	if len(key) < 50 {
		return NewWorkspaceError(ErrCodeInvalidRequest, "Claude API 키 길이가 너무 짧습니다", ErrInvalidRequest)
	}
	
	return nil
}

// validateStatus는 워크스페이스 상태를 검증합니다
func (v *WorkspaceValidator) validateStatus(status models.WorkspaceStatus) error {
	if !status.IsValid() {
		return NewWorkspaceError(ErrCodeInvalidStatus, "유효하지 않은 워크스페이스 상태입니다", ErrInvalidWorkspaceStatus)
	}
	
	return nil
}

// CanCreateWorkspace는 사용자가 새 워크스페이스를 생성할 수 있는지 확인합니다
func (v *WorkspaceValidator) CanCreateWorkspace(ctx context.Context, userID string, currentCount int) error {
	if currentCount >= v.maxWorkspaceCount {
		return NewWorkspaceError(ErrCodeMaxWorkspaces, "최대 워크스페이스 수에 도달했습니다", ErrMaxWorkspacesReached)
	}
	
	return nil
}

// CanActivateWorkspace는 워크스페이스를 활성화할 수 있는지 확인합니다
func (v *WorkspaceValidator) CanActivateWorkspace(ctx context.Context, workspace *models.Workspace) error {
	if workspace.Status == models.WorkspaceStatusArchived {
		return NewWorkspaceError(ErrCodeArchived, "아카이브된 워크스페이스는 활성화할 수 없습니다", ErrWorkspaceArchived)
	}
	
	return nil
}

// CanDeactivateWorkspace는 워크스페이스를 비활성화할 수 있는지 확인합니다
func (v *WorkspaceValidator) CanDeactivateWorkspace(ctx context.Context, workspace *models.Workspace) error {
	if workspace.ActiveTasks > 0 {
		return NewWorkspaceError(ErrCodeResourceBusy, "활성 태스크가 있는 워크스페이스는 비활성화할 수 없습니다", ErrResourceBusy)
	}
	
	return nil
}

// CanDeleteWorkspace는 워크스페이스를 삭제할 수 있는지 확인합니다
func (v *WorkspaceValidator) CanDeleteWorkspace(ctx context.Context, workspace *models.Workspace) error {
	if workspace.ActiveTasks > 0 {
		return NewWorkspaceError(ErrCodeResourceBusy, "활성 태스크가 있는 워크스페이스는 삭제할 수 없습니다", ErrResourceBusy)
	}
	
	// 추가적인 의존성 검사는 여기서 수행
	// 예: 연결된 프로젝트, 세션 등
	
	return nil
}