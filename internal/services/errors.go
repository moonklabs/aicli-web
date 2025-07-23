package services

import (
	"github.com/aicli/aicli-web/internal/errors"
)

// 에러 변수들을 errors 패키지로 재사용
var (
	// 일반적인 에러
	ErrWorkspaceNotFound           = errors.ErrWorkspaceNotFound
	ErrInvalidWorkspaceName        = errors.ErrInvalidWorkspaceName
	ErrInvalidProjectPath          = errors.ErrInvalidProjectPath
	ErrWorkspaceExists             = errors.ErrWorkspaceExists
	ErrUnauthorized                = errors.ErrUnauthorized
	ErrInvalidRequest              = errors.ErrInvalidRequest
	
	// 상태 관련 에러
	ErrInvalidWorkspaceStatus      = errors.ErrInvalidWorkspaceStatus
	ErrWorkspaceNotActive          = errors.ErrWorkspaceNotActive
	ErrWorkspaceArchived           = errors.ErrWorkspaceArchived
	
	// 권한 관련 에러
	ErrInsufficientPermissions     = errors.ErrInsufficientPermissions
	ErrOwnershipRequired           = errors.ErrOwnershipRequired
	
	// 리소스 관련 에러
	ErrMaxWorkspacesReached        = errors.ErrMaxWorkspacesReached
	ErrResourceBusy                = errors.ErrResourceBusy
	ErrDependencyExists            = errors.ErrDependencyExists
)

// WorkspaceError 는 errors 패키지에서 사용
type WorkspaceError = errors.WorkspaceError

// NewWorkspaceError 는 errors 패키지에서 사용
var NewWorkspaceError = errors.NewWorkspaceError
var NewWorkspaceErrorWithContext = errors.NewWorkspaceErrorWithContext

// 에러 코드들을 errors 패키지에서 사용
const (
	ErrCodeNotFound          = errors.ErrCodeNotFound
	ErrCodeInvalidName       = errors.ErrCodeInvalidName
	ErrCodeInvalidPath       = errors.ErrCodeInvalidPath
	ErrCodeAlreadyExists     = errors.ErrCodeAlreadyExists
	ErrCodeUnauthorized      = errors.ErrCodeUnauthorized
	ErrCodeInvalidRequest    = errors.ErrCodeInvalidRequest
	ErrCodeInvalidStatus     = errors.ErrCodeInvalidStatus
	ErrCodeNotActive         = errors.ErrCodeNotActive
	ErrCodeArchived          = errors.ErrCodeArchived
	ErrCodeInsufficientPerm  = errors.ErrCodeInsufficientPerm
	ErrCodeOwnershipRequired = errors.ErrCodeOwnershipRequired
	ErrCodeMaxWorkspaces     = errors.ErrCodeMaxWorkspaces
	ErrCodeResourceBusy      = errors.ErrCodeResourceBusy
	ErrCodeDependencyExists  = errors.ErrCodeDependencyExists
	ErrCodeInternal          = errors.ErrCodeInternal
)