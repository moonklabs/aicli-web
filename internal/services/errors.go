package services

import "errors"

// 워크스페이스 관련 에러
var (
	// 일반적인 에러
	ErrWorkspaceNotFound    = errors.New("workspace not found")
	ErrInvalidWorkspaceName = errors.New("invalid workspace name")
	ErrInvalidProjectPath   = errors.New("invalid project path")
	ErrWorkspaceExists      = errors.New("workspace already exists")
	ErrUnauthorized         = errors.New("unauthorized access")
	ErrInvalidRequest       = errors.New("invalid request")
	
	// 상태 관련 에러
	ErrInvalidWorkspaceStatus = errors.New("invalid workspace status")
	ErrWorkspaceNotActive     = errors.New("workspace is not active")
	ErrWorkspaceArchived      = errors.New("workspace is archived")
	
	// 권한 관련 에러
	ErrInsufficientPermissions = errors.New("insufficient permissions")
	ErrOwnershipRequired       = errors.New("ownership required")
	
	// 리소스 관련 에러
	ErrMaxWorkspacesReached = errors.New("maximum number of workspaces reached")
	ErrResourceBusy         = errors.New("resource is busy")
	ErrDependencyExists     = errors.New("dependency exists")
)

// WorkspaceError는 워크스페이스 관련 에러를 나타냅니다
type WorkspaceError struct {
	Code    string
	Message string
	Err     error
}

func (e *WorkspaceError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *WorkspaceError) Unwrap() error {
	return e.Err
}

// NewWorkspaceError는 새로운 워크스페이스 에러를 생성합니다
func NewWorkspaceError(code, message string, err error) *WorkspaceError {
	return &WorkspaceError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// 일반적인 에러 코드 상수
const (
	ErrCodeNotFound         = "WORKSPACE_NOT_FOUND"
	ErrCodeInvalidName      = "INVALID_WORKSPACE_NAME"
	ErrCodeInvalidPath      = "INVALID_PROJECT_PATH"
	ErrCodeAlreadyExists    = "WORKSPACE_ALREADY_EXISTS"
	ErrCodeUnauthorized     = "UNAUTHORIZED_ACCESS"
	ErrCodeInvalidRequest   = "INVALID_REQUEST"
	ErrCodeInvalidStatus    = "INVALID_WORKSPACE_STATUS"
	ErrCodeNotActive        = "WORKSPACE_NOT_ACTIVE"
	ErrCodeArchived         = "WORKSPACE_ARCHIVED"
	ErrCodeInsufficientPerm = "INSUFFICIENT_PERMISSIONS"
	ErrCodeOwnershipRequired = "OWNERSHIP_REQUIRED"
	ErrCodeMaxWorkspaces    = "MAX_WORKSPACES_REACHED"
	ErrCodeResourceBusy     = "RESOURCE_BUSY"
	ErrCodeDependencyExists = "DEPENDENCY_EXISTS"
)