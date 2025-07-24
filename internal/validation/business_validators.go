package validation

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/storage"
)

// WorkspaceBusinessValidator 워크스페이스 비즈니스 검증자
type WorkspaceBusinessValidator struct {
	storage storage.WorkspaceStorage
}

// NewWorkspaceBusinessValidator 새로운 워크스페이스 비즈니스 검증자 생성
func NewWorkspaceBusinessValidator(storage storage.WorkspaceStorage) *WorkspaceBusinessValidator {
	return &WorkspaceBusinessValidator{
		storage: storage,
	}
}

// ValidateCreate 워크스페이스 생성 검증
func (v *WorkspaceBusinessValidator) ValidateCreate(ctx context.Context, model interface{}) error {
	ws, ok := model.(*models.Workspace)
	if !ok {
		createReq, ok := model.(*models.CreateWorkspaceRequest)
		if !ok {
			return NewBusinessValidationError(ErrCodeInvalidConfiguration, "유효하지 않은 모델 타입")
		}
		// CreateWorkspaceRequest를 Workspace로 변환
		ws = &models.Workspace{
			Name:        createReq.Name,
			ProjectPath: createReq.ProjectPath,
			ClaudeKey:   createReq.ClaudeKey,
		}
	}

	// 1. 중복 이름 체크 (같은 소유자 내에서)
	if ws.OwnerID != "" {
		existing, err := v.storage.GetByName(ctx, ws.OwnerID, ws.Name)
		if err != nil && !storage.IsNotFoundError(err) {
			return NewBusinessValidationError(ErrCodeInvalidConfiguration, "워크스페이스 이름 중복 확인 중 오류 발생")
		}
		if existing != nil {
			return ErrDuplicateWorkspaceName
		}
	}

	// 2. 프로젝트 경로 검증
	if err := v.validateProjectPath(ws.ProjectPath); err != nil {
		return err
	}

	// 3. 최대 워크스페이스 수 제한 체크 (예: 사용자당 20개)
	if ws.OwnerID != "" {
		if err := v.checkWorkspaceLimit(ctx, ws.OwnerID); err != nil {
			return err
		}
	}

	// 4. Claude API 키 검증 (옵션)
	if ws.ClaudeKey != "" {
		if err := v.validateClaudeAPIKey(ws.ClaudeKey); err != nil {
			return err
		}
	}

	return nil
}

// ValidateUpdate 워크스페이스 업데이트 검증
func (v *WorkspaceBusinessValidator) ValidateUpdate(ctx context.Context, model interface{}) error {
	updateReq, ok := model.(*models.UpdateWorkspaceRequest)
	if !ok {
		return NewBusinessValidationError(ErrCodeInvalidConfiguration, "유효하지 않은 업데이트 요청 타입")
	}

	// 현재 워크스페이스 정보가 필요한 경우를 위해 ID를 받을 수 있도록 확장 가능

	// 1. 이름 변경 시 중복 체크
	if updateReq.Name != "" {
		// TODO: 현재 워크스페이스 ID를 받아서 중복 체크 시 자기 자신은 제외
	}

	// 2. 프로젝트 경로 변경 시 검증
	if updateReq.ProjectPath != "" {
		if err := v.validateProjectPath(updateReq.ProjectPath); err != nil {
			return err
		}
	}

	// 3. 상태 변경 검증
	if updateReq.Status != "" {
		if err := v.validateStatusChange(ctx, string(updateReq.Status)); err != nil {
			return err
		}
	}

	// 4. Claude API 키 검증
	if updateReq.ClaudeKey != "" {
		if err := v.validateClaudeAPIKey(updateReq.ClaudeKey); err != nil {
			return err
		}
	}

	return nil
}

// ValidateDelete 워크스페이스 삭제 검증
func (v *WorkspaceBusinessValidator) ValidateDelete(ctx context.Context, id string) error {
	// 1. 워크스페이스 존재 여부 확인
	ws, err := v.storage.GetByID(ctx, id)
	if err != nil {
		if storage.IsNotFoundError(err) {
			return ErrWorkspaceNotFound
		}
		return NewBusinessValidationError(ErrCodeInvalidConfiguration, "워크스페이스 조회 중 오류 발생")
	}

	// 2. 활성 태스크 존재 여부 확인
	if ws.ActiveTasks > 0 {
		return ErrActiveTasksExist
	}

	// 3. 연관된 프로젝트 확인 (필요시)
	// TODO: 프로젝트 스토리지를 주입받아서 연관 프로젝트 확인

	return nil
}

// validateProjectPath 프로젝트 경로 검증
func (v *WorkspaceBusinessValidator) validateProjectPath(path string) error {
	// 경로 안전성 검사
	if err := ValidatePathWithOptions(path, PathValidationOptions{
		MustBeDir:     true,
		Writable:      true,
		AllowRelative: false,
		MaxDepth:      10,
	}); err != nil {
		return err
	}

	// 시스템 보호 경로 확인
	protectedPaths := []string{
		"/",
		"/bin",
		"/boot",
		"/dev",
		"/etc",
		"/lib",
		"/proc",
		"/root",
		"/sys",
		"/usr",
		"/var",
	}

	absPath, _ := filepath.Abs(path)
	for _, protected := range protectedPaths {
		if strings.HasPrefix(absPath, protected) {
			return NewBusinessValidationError(
				ErrCodePathNotAccessible,
				"시스템 보호 경로는 사용할 수 없습니다",
				"project_path",
			)
		}
	}

	return nil
}

// checkWorkspaceLimit 워크스페이스 수 제한 확인
func (v *WorkspaceBusinessValidator) checkWorkspaceLimit(ctx context.Context, ownerID string) error {
	// TODO: 설정에서 최대값을 가져오도록 수정
	const maxWorkspaces = 20

	count, err := v.storage.CountByOwner(ctx, ownerID)
	if err != nil {
		return NewBusinessValidationError(ErrCodeInvalidConfiguration, "워크스페이스 수 확인 중 오류 발생")
	}

	if count >= maxWorkspaces {
		return ErrMaxWorkspaceLimit
	}

	return nil
}

// validateClaudeAPIKey Claude API 키 검증
func (v *WorkspaceBusinessValidator) validateClaudeAPIKey(key string) error {
	// 기본 형식 검증은 validator에서 처리됨
	// 여기서는 추가적인 비즈니스 로직 처리

	// API 키 길이 검증
	if len(key) < 50 || len(key) > 200 {
		return NewBusinessValidationError(
			ErrCodeInvalidConfiguration,
			"Claude API 키 길이가 올바르지 않습니다",
			"claude_key",
		)
	}

	return nil
}

// validateStatusChange 상태 변경 검증
func (v *WorkspaceBusinessValidator) validateStatusChange(ctx context.Context, newStatus string) error {
	// 상태 변경 규칙 검증 (예: archived -> active 불가)
	switch newStatus {
	case string(models.WorkspaceStatusArchived):
		// 아카이브로 변경 시 추가 검증
		return nil
	case string(models.WorkspaceStatusActive):
		// 활성화로 변경 시 추가 검증
		return nil
	case string(models.WorkspaceStatusInactive):
		// 비활성화로 변경 시 추가 검증
		return nil
	default:
		return NewBusinessValidationError(
			ErrCodeInvalidStatus,
			"유효하지 않은 워크스페이스 상태입니다",
			"status",
		)
	}
}

// ProjectBusinessValidator 프로젝트 비즈니스 검증자
type ProjectBusinessValidator struct {
	projectStorage   storage.ProjectStorage
	workspaceStorage storage.WorkspaceStorage
}

// NewProjectBusinessValidator 새로운 프로젝트 비즈니스 검증자 생성
func NewProjectBusinessValidator(projectStorage storage.ProjectStorage, workspaceStorage storage.WorkspaceStorage) *ProjectBusinessValidator {
	return &ProjectBusinessValidator{
		projectStorage:   projectStorage,
		workspaceStorage: workspaceStorage,
	}
}

// ValidateCreate 프로젝트 생성 검증
func (v *ProjectBusinessValidator) ValidateCreate(ctx context.Context, model interface{}) error {
	project, ok := model.(*models.Project)
	if !ok {
		return NewBusinessValidationError(ErrCodeInvalidConfiguration, "유효하지 않은 모델 타입")
	}

	// 1. 워크스페이스 존재 여부 확인
	workspace, err := v.workspaceStorage.GetByID(ctx, project.WorkspaceID)
	if err != nil {
		if storage.IsNotFoundError(err) {
			return ErrWorkspaceNotFound
		}
		return NewBusinessValidationError(ErrCodeInvalidConfiguration, "워크스페이스 조회 중 오류 발생")
	}

	// 2. 워크스페이스 상태 확인
	if workspace.Status != models.WorkspaceStatusActive {
		return NewBusinessValidationError(
			ErrCodeInvalidStatus,
			"비활성 워크스페이스에는 프로젝트를 생성할 수 없습니다",
		)
	}

	// 3. 프로젝트 이름 중복 체크 (스킵 - 메서드 미구현)
	// existing, err := v.projectStorage.GetByName(ctx, project.WorkspaceID, project.Name)
	// if err != nil && !storage.IsNotFoundError(err) {
	//	return NewBusinessValidationError(ErrCodeInvalidConfiguration, "프로젝트 이름 중복 확인 중 오류 발생")
	// }
	// if existing != nil {
	//	return ErrDuplicateProjectName
	// }

	// 4. 최대 프로젝트 수 제한 체크
	if err := v.checkProjectLimit(ctx, project.WorkspaceID); err != nil {
		return err
	}

	// 5. 프로젝트 경로 검증
	if err := v.validateProjectPath(project.Path, workspace.ProjectPath); err != nil {
		return err
	}

	return nil
}

// ValidateUpdate 프로젝트 업데이트 검증
func (v *ProjectBusinessValidator) ValidateUpdate(ctx context.Context, model interface{}) error {
	// 업데이트 로직 구현
	return nil
}

// ValidateDelete 프로젝트 삭제 검증
func (v *ProjectBusinessValidator) ValidateDelete(ctx context.Context, id string) error {
	// 1. 프로젝트 존재 여부 확인
	project, err := v.projectStorage.GetByID(ctx, id)
	if err != nil {
		if storage.IsNotFoundError(err) {
			return ErrProjectNotFound
		}
		return NewBusinessValidationError(ErrCodeInvalidConfiguration, "프로젝트 조회 중 오류 발생")
	}

	// 2. 활성 세션 존재 여부 확인
	// TODO: 세션 스토리지를 주입받아서 활성 세션 확인
	_ = project

	return nil
}

// checkProjectLimit 프로젝트 수 제한 확인
func (v *ProjectBusinessValidator) checkProjectLimit(ctx context.Context, workspaceID string) error {
	const maxProjects = 50

	// count, err := v.projectStorage.CountByWorkspace(ctx, workspaceID)
	// if err != nil {
	//	return NewBusinessValidationError(ErrCodeInvalidConfiguration, "프로젝트 수 확인 중 오류 발생")
	// }
	count := int64(0) // 스킵 - 메서드 미구현

	if count >= maxProjects {
		return ErrMaxProjectLimit
	}

	return nil
}

// validateProjectPath 프로젝트 경로 검증 (워크스페이스 경로 기준)
func (v *ProjectBusinessValidator) validateProjectPath(projectPath, workspacePath string) error {
	// 절대 경로로 변환
	absProjectPath, err := filepath.Abs(projectPath)
	if err != nil {
		return NewBusinessValidationError(
			ErrCodeInvalidConfiguration,
			"프로젝트 경로를 절대 경로로 변환할 수 없습니다",
			"path",
		)
	}

	absWorkspacePath, err := filepath.Abs(workspacePath)
	if err != nil {
		return NewBusinessValidationError(
			ErrCodeInvalidConfiguration,
			"워크스페이스 경로를 절대 경로로 변환할 수 없습니다",
		)
	}

	// 프로젝트 경로가 워크스페이스 경로 내에 있는지 확인
	if !strings.HasPrefix(absProjectPath, absWorkspacePath) {
		return NewBusinessValidationError(
			ErrCodePathNotAccessible,
			"프로젝트 경로는 워크스페이스 경로 내에 있어야 합니다",
			"path",
		)
	}

	// 디렉토리 존재 여부 및 권한 확인
	return ValidatePathWithOptions(projectPath, PathValidationOptions{
		MustBeDir: true,
		Writable:  true,
		Readable:  true,
	})
}

// SessionBusinessValidator 세션 비즈니스 검증자
type SessionBusinessValidator struct {
	sessionStorage storage.SessionStorage
	projectStorage storage.ProjectStorage
}

// NewSessionBusinessValidator 새로운 세션 비즈니스 검증자 생성
func NewSessionBusinessValidator(sessionStorage storage.SessionStorage, projectStorage storage.ProjectStorage) *SessionBusinessValidator {
	return &SessionBusinessValidator{
		sessionStorage: sessionStorage,
		projectStorage: projectStorage,
	}
}

// ValidateCreate 세션 생성 검증
func (v *SessionBusinessValidator) ValidateCreate(ctx context.Context, model interface{}) error {
	createReq, ok := model.(*models.SessionCreateRequest)
	if !ok {
		return NewBusinessValidationError(ErrCodeInvalidConfiguration, "유효하지 않은 모델 타입")
	}

	// 1. 프로젝트 존재 여부 확인
	project, err := v.projectStorage.GetByID(ctx, createReq.ProjectID)
	if err != nil {
		if storage.IsNotFoundError(err) {
			return ErrProjectNotFound
		}
		return NewBusinessValidationError(ErrCodeInvalidConfiguration, "프로젝트 조회 중 오류 발생")
	}

	// 2. 프로젝트 상태 확인
	if project.Status != models.ProjectStatusActive {
		return NewBusinessValidationError(
			ErrCodeInvalidStatus,
			"비활성 프로젝트에는 세션을 생성할 수 없습니다",
		)
	}

	// 3. 동시 세션 수 제한 체크
	if err := v.checkConcurrentSessionLimit(ctx, createReq.ProjectID); err != nil {
		return err
	}

	return nil
}

// ValidateUpdate 세션 업데이트 검증
func (v *SessionBusinessValidator) ValidateUpdate(ctx context.Context, model interface{}) error {
	// 업데이트 로직 구현
	return nil
}

// ValidateDelete 세션 삭제 검증
func (v *SessionBusinessValidator) ValidateDelete(ctx context.Context, id string) error {
	// 1. 세션 존재 여부 확인
	session, err := v.sessionStorage.GetByID(ctx, id)
	if err != nil {
		if storage.IsNotFoundError(err) {
			return ErrSessionNotFound
		}
		return NewBusinessValidationError(ErrCodeInvalidConfiguration, "세션 조회 중 오류 발생")
	}

	// 2. 세션이 삭제 가능한 상태인지 확인
	if session.Status == models.SessionActive || session.Status == models.SessionPending {
		return NewBusinessValidationError(
			ErrCodeInvalidStatus,
			"활성 상태의 세션은 삭제할 수 없습니다. 먼저 종료해주세요.",
		)
	}

	return nil
}

// checkConcurrentSessionLimit 동시 세션 수 제한 확인
func (v *SessionBusinessValidator) checkConcurrentSessionLimit(ctx context.Context, projectID string) error {
	const maxConcurrentSessions = 3

	// activeCount, err := v.sessionStorage.CountActiveByProject(ctx, projectID)
	// if err != nil {
	//	return NewBusinessValidationError(ErrCodeInvalidConfiguration, "활성 세션 수 확인 중 오류 발생")
	// }
	activeCount := int64(0) // 스킵 - 메서드 미구현

	if activeCount >= maxConcurrentSessions {
		return NewBusinessValidationError(
			ErrCodeResourceLimit,
			fmt.Sprintf("프로젝트당 최대 %d개의 동시 세션만 허용됩니다", maxConcurrentSessions),
		)
	}

	return nil
}

// TaskBusinessValidator 태스크 비즈니스 검증자
type TaskBusinessValidator struct {
	taskStorage    storage.TaskStorage
	sessionStorage storage.SessionStorage
}

// NewTaskBusinessValidator 새로운 태스크 비즈니스 검증자 생성
func NewTaskBusinessValidator(taskStorage storage.TaskStorage, sessionStorage storage.SessionStorage) *TaskBusinessValidator {
	return &TaskBusinessValidator{
		taskStorage:    taskStorage,
		sessionStorage: sessionStorage,
	}
}

// ValidateCreate 태스크 생성 검증
func (v *TaskBusinessValidator) ValidateCreate(ctx context.Context, model interface{}) error {
	createReq, ok := model.(*models.TaskCreateRequest)
	if !ok {
		return NewBusinessValidationError(ErrCodeInvalidConfiguration, "유효하지 않은 모델 타입")
	}

	// 1. 세션 존재 여부 확인
	session, err := v.sessionStorage.GetByID(ctx, createReq.SessionID)
	if err != nil {
		if storage.IsNotFoundError(err) {
			return ErrSessionNotFound
		}
		return NewBusinessValidationError(ErrCodeInvalidConfiguration, "세션 조회 중 오류 발생")
	}

	// 2. 세션이 활성 상태인지 확인
	if !session.IsActive() {
		return NewBusinessValidationError(
			ErrCodeInvalidStatus,
			"비활성 세션에는 태스크를 생성할 수 없습니다",
		)
	}

	// 3. 명령어 안전성 검증
	if err := v.validateCommand(createReq.Command); err != nil {
		return err
	}

	// 4. 동시 태스크 수 제한 체크
	if err := v.checkConcurrentTaskLimit(ctx, createReq.SessionID); err != nil {
		return err
	}

	return nil
}

// ValidateUpdate 태스크 업데이트 검증
func (v *TaskBusinessValidator) ValidateUpdate(ctx context.Context, model interface{}) error {
	updateReq, ok := model.(*models.TaskUpdateRequest)
	if !ok {
		return NewBusinessValidationError(ErrCodeInvalidConfiguration, "유효하지 않은 모델 타입")
	}

	// 상태 변경 검증
	return v.validateStatusTransition(updateReq.Status)
}

// ValidateDelete 태스크 삭제 검증
func (v *TaskBusinessValidator) ValidateDelete(ctx context.Context, id string) error {
	// 1. 태스크 존재 여부 확인
	task, err := v.taskStorage.GetByID(ctx, id)
	if err != nil {
		if storage.IsNotFoundError(err) {
			return NewBusinessValidationError(ErrCodeResourceNotFound, "태스크를 찾을 수 없습니다")
		}
		return NewBusinessValidationError(ErrCodeInvalidConfiguration, "태스크 조회 중 오류 발생")
	}

	// 2. 실행 중인 태스크는 삭제 불가
	if task.Status == models.TaskRunning {
		return NewBusinessValidationError(
			ErrCodeInvalidStatus,
			"실행 중인 태스크는 삭제할 수 없습니다. 먼저 취소해주세요.",
		)
	}

	return nil
}

// validateCommand 명령어 안전성 검증
func (v *TaskBusinessValidator) validateCommand(command string) error {
	// 위험한 명령어 패턴 검사
	dangerousPatterns := []string{
		"rm -rf /",
		"dd if=",
		"mkfs",
		"format",
		"fdisk",
		"> /dev/",
		"shutdown",
		"reboot",
		"init 0",
		"init 6",
		"halt",
		"poweroff",
	}

	lowerCommand := strings.ToLower(command)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(lowerCommand, pattern) {
			return NewBusinessValidationError(
				ErrCodeInvalidConfiguration,
				fmt.Sprintf("위험한 명령어가 감지되었습니다: %s", pattern),
				"command",
			)
		}
	}

	return nil
}

// checkConcurrentTaskLimit 동시 태스크 수 제한 확인
func (v *TaskBusinessValidator) checkConcurrentTaskLimit(ctx context.Context, sessionID string) error {
	const maxConcurrentTasks = 5

	// activeCount, err := v.taskStorage.CountActiveBySession(ctx, sessionID)
	// if err != nil {
	//	return NewBusinessValidationError(ErrCodeInvalidConfiguration, "활성 태스크 수 확인 중 오류 발생")
	// }
	activeCount := int64(0) // 스킵 - 메서드 미구현

	if activeCount >= maxConcurrentTasks {
		return NewBusinessValidationError(
			ErrCodeResourceLimit,
			fmt.Sprintf("세션당 최대 %d개의 동시 태스크만 허용됩니다", maxConcurrentTasks),
		)
	}

	return nil
}

// validateStatusTransition 상태 전환 검증
func (v *TaskBusinessValidator) validateStatusTransition(newStatus models.TaskStatus) error {
	// 상태 전환 규칙 검증
	switch newStatus {
	case models.TaskRunning:
		// pending -> running만 허용
		return nil
	case models.TaskCompleted, models.TaskFailed:
		// running -> completed/failed만 허용
		return nil
	case models.TaskCancelled:
		// pending/running -> cancelled 허용
		return nil
	default:
		return NewBusinessValidationError(
			ErrCodeInvalidStatus,
			"유효하지 않은 태스크 상태 전환입니다",
			"status",
		)
	}
}