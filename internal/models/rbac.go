package models

import (
	"time"
)

// ResourceType 리소스 타입
type ResourceType string

const (
	ResourceTypeWorkspace ResourceType = "workspace"
	ResourceTypeProject   ResourceType = "project"  
	ResourceTypeSession   ResourceType = "session"
	ResourceTypeTask      ResourceType = "task"
	ResourceTypeUser      ResourceType = "user"
	ResourceTypeSystem    ResourceType = "system"
)

// ActionType 액션 타입
type ActionType string

const (
	ActionCreate ActionType = "create"
	ActionRead   ActionType = "read"
	ActionUpdate ActionType = "update"
	ActionDelete ActionType = "delete"
	ActionExecute ActionType = "execute"
	ActionManage ActionType = "manage"
)

// PermissionEffect 권한 효과
type PermissionEffect string

const (
	PermissionAllow PermissionEffect = "allow"
	PermissionDeny  PermissionEffect = "deny"
)

// Role 역할 모델 - 권한 그룹핑 및 계층 구조 지원
type Role struct {
	BaseModel
	Name        string  `json:"name" db:"name" validate:"required,min=1,max=50"`
	Description string  `json:"description" db:"description" validate:"max=200"`
	ParentID    *string `json:"parent_id,omitempty" db:"parent_id" validate:"omitempty,uuid"`
	Level       int     `json:"level" db:"level" validate:"min=0,max=10"` // 역할 계층 레벨
	IsSystem    bool    `json:"is_system" db:"is_system"`                 // 시스템 기본 역할 여부
	IsActive    bool    `json:"is_active" db:"is_active"`
	
	// 관계
	Parent      *Role        `json:"parent,omitempty"`
	Children    []Role       `json:"children,omitempty"`
	Permissions []Permission `json:"permissions,omitempty"`
}

// Permission 권한 모델 - 세분화된 권한 정의
type Permission struct {
	BaseModel
	Name         string           `json:"name" db:"name" validate:"required,min=1,max=100"`
	Description  string           `json:"description" db:"description" validate:"max=200"`
	ResourceType ResourceType     `json:"resource_type" db:"resource_type" validate:"required"`
	Action       ActionType       `json:"action" db:"action" validate:"required"`
	Effect       PermissionEffect `json:"effect" db:"effect" validate:"required"`
	Conditions   string           `json:"conditions,omitempty" db:"conditions"` // JSON 형태 조건
	IsActive     bool             `json:"is_active" db:"is_active"`
}

// Resource 리소스 모델 - 권한 대상 리소스 정의
type Resource struct {
	BaseModel
	Name         string       `json:"name" db:"name" validate:"required,min=1,max=100"`
	Type         ResourceType `json:"type" db:"type" validate:"required"`
	Identifier   string       `json:"identifier" db:"identifier" validate:"required"` // 리소스 고유 식별자
	ParentID     *string      `json:"parent_id,omitempty" db:"parent_id" validate:"omitempty,uuid"`
	Path         string       `json:"path" db:"path"`         // 리소스 경로 (계층 구조)
	Attributes   string       `json:"attributes" db:"attributes"` // JSON 형태 추가 속성
	IsActive     bool         `json:"is_active" db:"is_active"`
	
	// 관계
	Parent   *Resource `json:"parent,omitempty"`
	Children []Resource `json:"children,omitempty"`
}

// UserGroup 사용자 그룹 모델 - 그룹 기반 권한 관리
type UserGroup struct {
	BaseModel
	Name        string  `json:"name" db:"name" validate:"required,min=1,max=50"`
	Description string  `json:"description" db:"description" validate:"max=200"`
	ParentID    *string `json:"parent_id,omitempty" db:"parent_id" validate:"omitempty,uuid"`
	Type        string  `json:"type" db:"type" validate:"required,oneof=organization department team project"`
	IsActive    bool    `json:"is_active" db:"is_active"`
	
	// 관계
	Parent   *UserGroup `json:"parent,omitempty"`
	Children []UserGroup `json:"children,omitempty"`
	Members  []User     `json:"members,omitempty"`
	Roles    []Role     `json:"roles,omitempty"`
}

// RolePermission 역할-권한 연결 모델 (다대다 관계)
type RolePermission struct {
	RoleID       string           `json:"role_id" db:"role_id" validate:"required,uuid"`
	PermissionID string           `json:"permission_id" db:"permission_id" validate:"required,uuid"`
	Effect       PermissionEffect `json:"effect" db:"effect" validate:"required"` // 역할별 권한 오버라이드
	Conditions   string           `json:"conditions,omitempty" db:"conditions"`   // 역할별 추가 조건
	CreatedAt    time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at" db:"updated_at"`
	
	// 관계
	Role       *Role       `json:"role,omitempty"`
	Permission *Permission `json:"permission,omitempty"`
}

// UserRole 사용자-역할 연결 모델 (다대다 관계)
type UserRole struct {
	UserID       string     `json:"user_id" db:"user_id" validate:"required,uuid"`
	RoleID       string     `json:"role_id" db:"role_id" validate:"required,uuid"`
	AssignedBy   string     `json:"assigned_by" db:"assigned_by" validate:"required,uuid"`
	ResourceID   *string    `json:"resource_id,omitempty" db:"resource_id" validate:"omitempty,uuid"` // 특정 리소스에 대한 역할
	ExpiresAt    *time.Time `json:"expires_at,omitempty" db:"expires_at"`                            // 역할 만료 시간
	IsActive     bool       `json:"is_active" db:"is_active"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
	
	// 관계
	User     *User     `json:"user,omitempty"`
	Role     *Role     `json:"role,omitempty"`
	Resource *Resource `json:"resource,omitempty"`
}

// UserGroupMember 사용자-그룹 연결 모델 (다대다 관계)
type UserGroupMember struct {
	UserID    string    `json:"user_id" db:"user_id" validate:"required,uuid"`
	GroupID   string    `json:"group_id" db:"group_id" validate:"required,uuid"`
	Role      string    `json:"role" db:"role" validate:"required,oneof=member admin owner"`
	JoinedAt  time.Time `json:"joined_at" db:"joined_at"`
	IsActive  bool      `json:"is_active" db:"is_active"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
	
	// 관계
	User  *User      `json:"user,omitempty"`
	Group *UserGroup `json:"group,omitempty"`
}

// GroupRole 그룹-역할 연결 모델 (다대다 관계) 
type GroupRole struct {
	GroupID    string     `json:"group_id" db:"group_id" validate:"required,uuid"`
	RoleID     string     `json:"role_id" db:"role_id" validate:"required,uuid"`
	ResourceID *string    `json:"resource_id,omitempty" db:"resource_id" validate:"omitempty,uuid"`
	AssignedBy string     `json:"assigned_by" db:"assigned_by" validate:"required,uuid"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty" db:"expires_at"`
	IsActive   bool       `json:"is_active" db:"is_active"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at" db:"updated_at"`
	
	// 관계
	Group    *UserGroup `json:"group,omitempty"`
	Role     *Role      `json:"role,omitempty"`
	Resource *Resource  `json:"resource,omitempty"`
}

// UserPermissionMatrix 사용자 권한 매트릭스 - 계산된 최종 권한
type UserPermissionMatrix struct {
	UserID           string                        `json:"user_id"`
	DirectRoles      []string                      `json:"direct_roles"`      // 직접 할당된 역할
	InheritedRoles   []string                      `json:"inherited_roles"`   // 그룹을 통해 상속된 역할
	GroupRoles       []string                      `json:"group_roles"`       // 그룹 역할
	FinalPermissions map[string]PermissionDecision `json:"final_permissions"` // 최종 권한 결정
	ComputedAt       time.Time                     `json:"computed_at"`
}

// PermissionDecision 권한 결정 결과
type PermissionDecision struct {
	ResourceType ResourceType     `json:"resource_type"`
	ResourceID   string           `json:"resource_id"`
	Action       ActionType       `json:"action"`
	Effect       PermissionEffect `json:"effect"`
	Source       string           `json:"source"`     // 권한 출처 (role, group, direct)
	Reason       string           `json:"reason"`     // 권한 결정 이유
	Conditions   string           `json:"conditions"` // 적용 조건
}

// PermissionContext 권한 검증 컨텍스트
type PermissionContext struct {
	UserID       string            `json:"user_id"`
	ResourceType ResourceType      `json:"resource_type"`
	ResourceID   string            `json:"resource_id"`
	Action       ActionType        `json:"action"`
	Attributes   map[string]string `json:"attributes,omitempty"` // 추가 컨텍스트 정보
}

// 권한 요청/응답 모델들

// CreateRoleRequest 역할 생성 요청
type CreateRoleRequest struct {
	Name        string  `json:"name" validate:"required,min=1,max=50"`
	Description string  `json:"description" validate:"max=200"`
	ParentID    *string `json:"parent_id,omitempty" validate:"omitempty,uuid"`
}

// UpdateRoleRequest 역할 수정 요청  
type UpdateRoleRequest struct {
	Name        *string `json:"name,omitempty" validate:"omitempty,min=1,max=50"`
	Description *string `json:"description,omitempty" validate:"omitempty,max=200"`
	ParentID    *string `json:"parent_id,omitempty" validate:"omitempty,uuid"`
	IsActive    *bool   `json:"is_active,omitempty"`
}

// CreatePermissionRequest 권한 생성 요청
type CreatePermissionRequest struct {
	Name         string           `json:"name" validate:"required,min=1,max=100"`
	Description  string           `json:"description" validate:"max=200"`
	ResourceType ResourceType     `json:"resource_type" validate:"required"`
	Action       ActionType       `json:"action" validate:"required"`
	Effect       PermissionEffect `json:"effect" validate:"required"`
	Conditions   string           `json:"conditions,omitempty"`
}

// AssignRoleRequest 역할 할당 요청
type AssignRoleRequest struct {
	UserID     string     `json:"user_id" validate:"required,uuid"`
	RoleID     string     `json:"role_id" validate:"required,uuid"`
	ResourceID *string    `json:"resource_id,omitempty" validate:"omitempty,uuid"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
}

// CheckPermissionRequest 권한 확인 요청
type CheckPermissionRequest struct {
	UserID       string            `json:"user_id" validate:"required,uuid"`
	ResourceType ResourceType      `json:"resource_type" validate:"required"`
	ResourceID   string            `json:"resource_id" validate:"required"`
	Action       ActionType        `json:"action" validate:"required"`
	Attributes   map[string]string `json:"attributes,omitempty"`
}

// CheckPermissionResponse 권한 확인 응답
type CheckPermissionResponse struct {
	Allowed    bool               `json:"allowed"`
	Decision   PermissionDecision `json:"decision"`
	Evaluation []string           `json:"evaluation"` // 권한 평가 과정
}

// 유효성 검사 메서드들

// IsValid ResourceType 유효성 검사
func (rt ResourceType) IsValid() bool {
	switch rt {
	case ResourceTypeWorkspace, ResourceTypeProject, ResourceTypeSession, 
		 ResourceTypeTask, ResourceTypeUser, ResourceTypeSystem:
		return true
	default:
		return false
	}
}

// IsValid ActionType 유효성 검사
func (at ActionType) IsValid() bool {
	switch at {
	case ActionCreate, ActionRead, ActionUpdate, ActionDelete, ActionExecute, ActionManage:
		return true
	default:
		return false
	}
}

// IsValid PermissionEffect 유효성 검사
func (pe PermissionEffect) IsValid() bool {
	switch pe {
	case PermissionAllow, PermissionDeny:
		return true
	default:
		return false
	}
}

// IsValid Role 유효성 검사
func (r *Role) IsValid() bool {
	if r.Name == "" || r.Level < 0 || r.Level > 10 {
		return false
	}
	return true
}

// IsValid Permission 유효성 검사
func (p *Permission) IsValid() bool {
	if p.Name == "" || !p.ResourceType.IsValid() || !p.Action.IsValid() || !p.Effect.IsValid() {
		return false
	}
	return true
}

// IsValid Resource 유효성 검사
func (r *Resource) IsValid() bool {
	if r.Name == "" || r.Identifier == "" || !r.Type.IsValid() {
		return false
	}
	return true
}

// IsValid UserGroup 유효성 검사
func (ug *UserGroup) IsValid() bool {
	if ug.Name == "" || ug.Type == "" {
		return false
	}
	validTypes := []string{"organization", "department", "team", "project"}
	for _, vt := range validTypes {
		if ug.Type == vt {
			return true
		}
	}
	return false
}

// 목록 조회 요청 모델들

// ListRolesRequest 역할 목록 조회 요청
type ListRolesRequest struct {
	Page   int     `json:"page" validate:"min=1"`
	Limit  int     `json:"limit" validate:"min=1,max=100"`
	Search string  `json:"search,omitempty"`
	Active *bool   `json:"active,omitempty"`
}

// ListPermissionsRequest 권한 목록 조회 요청
type ListPermissionsRequest struct {
	Page         int    `json:"page" validate:"min=1"`
	Limit        int    `json:"limit" validate:"min=1,max=100"`
	ResourceType string `json:"resource_type,omitempty"`
	Action       string `json:"action,omitempty"`
	Effect       string `json:"effect,omitempty"`
}

// PermissionAuditEntry 권한 감사 엔트리
type PermissionAuditEntry struct {
	BaseModel
	UserID     string    `json:"user_id" db:"user_id"`
	ResourceID string    `json:"resource_id" db:"resource_id"`
	Action     string    `json:"action" db:"action"`
	Result     string    `json:"result" db:"result"`
	Timestamp  time.Time `json:"timestamp" db:"timestamp"`
}

// PermissionConflict 권한 충돌
type PermissionConflict struct {
	BaseModel
	ResourceID   string `json:"resource_id" db:"resource_id"`
	ConflictType string `json:"conflict_type" db:"conflict_type"`
	Description  string `json:"description" db:"description"`
}

// RoleOptimizationSuggestion 역할 최적화 제안
type RoleOptimizationSuggestion struct {
	BaseModel
	RoleID      string `json:"role_id" db:"role_id"`
	Type        string `json:"type" db:"type"`
	Description string `json:"description" db:"description"`
	Priority    int    `json:"priority" db:"priority"`
}

// RoleConsolidationSuggestion 역할 통합 제안
type RoleConsolidationSuggestion struct {
	BaseModel
	SourceRoleID string `json:"source_role_id" db:"source_role_id"`
	TargetRoleID string `json:"target_role_id" db:"target_role_id"`
	Reason       string `json:"reason" db:"reason"`
}

// TemporaryPermission 임시 권한
type TemporaryPermission struct {
	BaseModel
	UserID       string     `json:"user_id" db:"user_id"`
	ResourceID   string     `json:"resource_id" db:"resource_id"`
	Action       string     `json:"action" db:"action"`
	ExpiresAt    time.Time  `json:"expires_at" db:"expires_at"`
	GrantedBy    string     `json:"granted_by" db:"granted_by"`
}