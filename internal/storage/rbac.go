package storage

import (
	"context"
	"fmt"
	"time"
	
	"github.com/aicli/aicli-web/internal/models"
)

// RBACStorage RBAC 관련 저장소 인터페이스 정의
type RBACStorage interface {
	// Role 관련 메서드
	CreateRole(ctx context.Context, role *models.Role) error
	GetRoleByID(ctx context.Context, roleID string) (*models.Role, error)
	GetRoleByName(ctx context.Context, name string) (*models.Role, error)
	GetRolesByUserID(ctx context.Context, userID string) ([]models.Role, error)
	GetRolesByGroupID(ctx context.Context, groupID string) ([]models.Role, error)
	GetRoleHierarchy(ctx context.Context, roleID string) ([]models.Role, error)
	GetAllRoles(ctx context.Context) ([]models.Role, error)
	GetChildRoles(ctx context.Context, parentID string) ([]models.Role, error)
	UpdateRole(ctx context.Context, roleID string, updates map[string]interface{}) error
	DeleteRole(ctx context.Context, roleID string) error
	
	// Permission 관련 메서드
	CreatePermission(ctx context.Context, permission *models.Permission) error
	GetPermissionByID(ctx context.Context, permissionID string) (*models.Permission, error)
	GetPermissionByName(ctx context.Context, name string) (*models.Permission, error)
	GetPermissionsByRoleID(ctx context.Context, roleID string) ([]models.Permission, error)
	GetAllPermissions(ctx context.Context) ([]models.Permission, error)
	GetPermissionsByResourceType(ctx context.Context, resourceType models.ResourceType) ([]models.Permission, error)
	GetPermissionsByAction(ctx context.Context, action models.ActionType) ([]models.Permission, error)
	UpdatePermission(ctx context.Context, permissionID string, updates map[string]interface{}) error
	DeletePermission(ctx context.Context, permissionID string) error
	
	// Resource 관련 메서드
	CreateResource(ctx context.Context, resource *models.Resource) error
	GetResourceByID(ctx context.Context, resourceID string) (*models.Resource, error)
	GetResourceByIdentifier(ctx context.Context, resourceType models.ResourceType, identifier string) (*models.Resource, error)
	GetResourceHierarchy(ctx context.Context, resourceID string) ([]models.Resource, error)
	GetResourcesByType(ctx context.Context, resourceType models.ResourceType) ([]models.Resource, error)
	GetChildResources(ctx context.Context, parentID string) ([]models.Resource, error)
	UpdateResource(ctx context.Context, resourceID string, updates map[string]interface{}) error
	DeleteResource(ctx context.Context, resourceID string) error
	
	// UserGroup 관련 메서드
	CreateUserGroup(ctx context.Context, group *models.UserGroup) error
	GetUserGroupByID(ctx context.Context, groupID string) (*models.UserGroup, error)
	GetUserGroupByName(ctx context.Context, name string) (*models.UserGroup, error)
	GetUserGroups(ctx context.Context, userID string) ([]models.UserGroup, error)
	GetGroupsByType(ctx context.Context, groupType string) ([]models.UserGroup, error)
	GetGroupHierarchy(ctx context.Context, groupID string) ([]models.UserGroup, error)
	GetChildGroups(ctx context.Context, parentID string) ([]models.UserGroup, error)
	UpdateUserGroup(ctx context.Context, groupID string, updates map[string]interface{}) error
	DeleteUserGroup(ctx context.Context, groupID string) error
	
	// RolePermission 관련 메서드
	AssignPermissionToRole(ctx context.Context, roleID, permissionID string, effect models.PermissionEffect, conditions string) error
	RevokePermissionFromRole(ctx context.Context, roleID, permissionID string) error
	GetRolePermissions(ctx context.Context, roleID string) ([]models.RolePermission, error)
	UpdateRolePermission(ctx context.Context, roleID, permissionID string, effect models.PermissionEffect, conditions string) error
	
	// UserRole 관련 메서드
	AssignRoleToUser(ctx context.Context, userID, roleID, assignedBy string, resourceID *string, expiresAt *time.Time) error
	RevokeRoleFromUser(ctx context.Context, userID, roleID string, resourceID *string) error
	GetUserRoles(ctx context.Context, userID string, resourceID *string) ([]models.UserRole, error)
	GetUsersInRole(ctx context.Context, roleID string, resourceID *string) ([]models.UserRole, error)
	UpdateUserRole(ctx context.Context, userID, roleID string, resourceID *string, expiresAt *time.Time, isActive bool) error
	
	// UserGroupMember 관련 메서드
	AddUserToGroup(ctx context.Context, userID, groupID, role string) error
	RemoveUserFromGroup(ctx context.Context, userID, groupID string) error
	GetGroupMembers(ctx context.Context, groupID string) ([]models.UserGroupMember, error)
	GetUserGroupMemberships(ctx context.Context, userID string) ([]models.UserGroupMember, error)
	UpdateGroupMembership(ctx context.Context, userID, groupID, role string, isActive bool) error
	
	// GroupRole 관련 메서드
	AssignRoleToGroup(ctx context.Context, groupID, roleID, assignedBy string, resourceID *string, expiresAt *time.Time) error
	RevokeRoleFromGroup(ctx context.Context, groupID, roleID string, resourceID *string) error
	GetGroupRoles(ctx context.Context, groupID string, resourceID *string) ([]models.GroupRole, error)
	GetGroupsInRole(ctx context.Context, roleID string, resourceID *string) ([]models.GroupRole, error)
	UpdateGroupRole(ctx context.Context, groupID, roleID string, resourceID *string, expiresAt *time.Time, isActive bool) error
	
	// 권한 검사 및 집계 메서드
	CheckUserPermission(ctx context.Context, userID string, resourceType models.ResourceType, resourceID string, action models.ActionType) (bool, error)
	GetUserEffectivePermissions(ctx context.Context, userID string) ([]models.PermissionDecision, error)
	GetUserPermissionMatrix(ctx context.Context, userID string) (*models.UserPermissionMatrix, error)
	
	// 통계 및 분석 메서드
	GetRoleUsageStats(ctx context.Context, roleID string) (map[string]int64, error)
	GetPermissionUsageStats(ctx context.Context, permissionID string) (map[string]int64, error)
	GetUserRoleHistory(ctx context.Context, userID string, limit int) ([]models.UserRole, error)
	GetGroupRoleHistory(ctx context.Context, groupID string, limit int) ([]models.GroupRole, error)
}

// RBACExtendedStorage 확장된 RBAC 저장소 인터페이스 (고급 기능)
type RBACExtendedStorage interface {
	RBACStorage
	
	// 배치 작업
	BatchAssignRolesToUser(ctx context.Context, userID string, roleAssignments []models.AssignRoleRequest) error
	BatchAssignRolesToGroup(ctx context.Context, groupID string, roleAssignments []models.AssignRoleRequest) error
	BatchRevokeRolesFromUser(ctx context.Context, userID string, roleIDs []string) error
	BatchRevokeRolesFromGroup(ctx context.Context, groupID string, roleIDs []string) error
	
	// 조건부 권한 조회
	GetConditionalPermissions(ctx context.Context, userID string, conditions map[string]interface{}) ([]models.PermissionDecision, error)
	GetResourceBasedPermissions(ctx context.Context, userID string, resourceType models.ResourceType, resourceID string) ([]models.PermissionDecision, error)
	
	// 권한 추적 및 감사
	LogPermissionCheck(ctx context.Context, userID string, resourceType models.ResourceType, resourceID string, action models.ActionType, allowed bool, reason string) error
	GetPermissionAuditLog(ctx context.Context, userID string, startTime, endTime time.Time) ([]models.PermissionAuditEntry, error)
	
	// 권한 분석
	AnalyzePermissionConflicts(ctx context.Context, userID string) ([]models.PermissionConflict, error)
	FindUnusedRoles(ctx context.Context, cutoffDate time.Time) ([]models.Role, error)
	FindUnusedPermissions(ctx context.Context, cutoffDate time.Time) ([]models.Permission, error)
	
	// 권한 최적화
	OptimizeUserRoles(ctx context.Context, userID string) ([]models.RoleOptimizationSuggestion, error)
	SuggestRoleConsolidation(ctx context.Context, threshold int) ([]models.RoleConsolidationSuggestion, error)
	
	// 임시 권한 관리
	GrantTemporaryPermission(ctx context.Context, userID string, permission models.Permission, duration time.Duration, reason string) error
	RevokeTemporaryPermission(ctx context.Context, userID, permissionID string) error
	GetActiveTemporaryPermissions(ctx context.Context, userID string) ([]models.TemporaryPermission, error)
	CleanupExpiredPermissions(ctx context.Context) (int64, error)
}

// 추가 모델 정의 (확장 기능용)

// PermissionAuditEntry 권한 감사 로그 엔트리
type PermissionAuditEntry struct {
	ID           string                `json:"id"`
	UserID       string                `json:"user_id"`
	ResourceType models.ResourceType   `json:"resource_type"`
	ResourceID   string                `json:"resource_id"`
	Action       models.ActionType     `json:"action"`
	Allowed      bool                  `json:"allowed"`
	Reason       string                `json:"reason"`
	IPAddress    string                `json:"ip_address,omitempty"`
	UserAgent    string                `json:"user_agent,omitempty"`
	Timestamp    time.Time             `json:"timestamp"`
	Context      map[string]interface{} `json:"context,omitempty"`
}

// PermissionConflict 권한 충돌 정보
type PermissionConflict struct {
	UserID           string                `json:"user_id"`
	ResourceType     models.ResourceType   `json:"resource_type"`
	ResourceID       string                `json:"resource_id"`
	Action           models.ActionType     `json:"action"`
	ConflictingRoles []string              `json:"conflicting_roles"`
	AllowSources     []string              `json:"allow_sources"`
	DenySources      []string              `json:"deny_sources"`
	Resolution       models.PermissionEffect `json:"resolution"`
	Reason           string                `json:"reason"`
}

// RoleOptimizationSuggestion 역할 최적화 제안
type RoleOptimizationSuggestion struct {
	UserID          string   `json:"user_id"`
	RedundantRoles  []string `json:"redundant_roles"`
	SuggestedRoles  []string `json:"suggested_roles"`
	Reason          string   `json:"reason"`
	PotentialSavings string   `json:"potential_savings"`
}

// RoleConsolidationSuggestion 역할 통합 제안  
type RoleConsolidationSuggestion struct {
	RolesToConsolidate []string `json:"roles_to_consolidate"`
	ConsolidatedRole   string   `json:"consolidated_role"`
	AffectedUsers      []string `json:"affected_users"`
	Reason             string   `json:"reason"`
	Impact             string   `json:"impact"`
}

// TemporaryPermission 임시 권한
type TemporaryPermission struct {
	ID           string               `json:"id"`
	UserID       string               `json:"user_id"`
	Permission   models.Permission    `json:"permission"`
	GrantedAt    time.Time            `json:"granted_at"`
	ExpiresAt    time.Time            `json:"expires_at"`
	GrantedBy    string               `json:"granted_by"`
	Reason       string               `json:"reason"`
	IsActive     bool                 `json:"is_active"`
	RevokedAt    *time.Time           `json:"revoked_at,omitempty"`
	RevokedBy    *string              `json:"revoked_by,omitempty"`
}

// RBACFilter RBAC 조회 필터
type RBACFilter struct {
	// Role 필터
	RoleName     string    `json:"role_name,omitempty"`
	RoleLevel    *int      `json:"role_level,omitempty"`
	IsSystemRole *bool     `json:"is_system_role,omitempty"`
	IsActive     *bool     `json:"is_active,omitempty"`
	ParentID     string    `json:"parent_id,omitempty"`
	
	// Permission 필터
	PermissionName string                 `json:"permission_name,omitempty"`
	ResourceType   models.ResourceType    `json:"resource_type,omitempty"`
	Action         models.ActionType      `json:"action,omitempty"`
	Effect         models.PermissionEffect `json:"effect,omitempty"`
	
	// Resource 필터
	ResourceName string                `json:"resource_name,omitempty"`
	Identifier   string                `json:"identifier,omitempty"`
	
	// Group 필터
	GroupName string `json:"group_name,omitempty"`
	GroupType string `json:"group_type,omitempty"`
	
	// 일반 필터
	CreatedAfter  *time.Time `json:"created_after,omitempty"`
	CreatedBefore *time.Time `json:"created_before,omitempty"`
	UpdatedAfter  *time.Time `json:"updated_after,omitempty"`
	UpdatedBefore *time.Time `json:"updated_before,omitempty"`
	
	// 페이지네이션
	Offset int `json:"offset,omitempty"`
	Limit  int `json:"limit,omitempty"`
	SortBy string `json:"sort_by,omitempty"`
	Order  string `json:"order,omitempty"`
}

// RBACQueryOptions RBAC 쿼리 옵션
type RBACQueryOptions struct {
	IncludeInactive  bool `json:"include_inactive,omitempty"`
	IncludeSystem    bool `json:"include_system,omitempty"`
	IncludeExpired   bool `json:"include_expired,omitempty"`
	LoadRelations    bool `json:"load_relations,omitempty"`
	ComputeHierarchy bool `json:"compute_hierarchy,omitempty"`
}

// RBACTransactionContext RBAC 트랜잭션 컨텍스트
type RBACTransactionContext struct {
	Storage    RBACStorage
	TxID       string
	UserID     string
	Timestamp  time.Time
	Operations []string
}

// RBACValidationError RBAC 검증 오류
type RBACValidationError struct {
	Field   string `json:"field"`
	Value   string `json:"value"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

// Error 오류 메시지 반환
func (e *RBACValidationError) Error() string {
	return fmt.Sprintf("RBAC validation error in field '%s': %s", e.Field, e.Message)
}

// RBACConstraint RBAC 제약 조건
type RBACConstraint struct {
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Conditions  map[string]interface{} `json:"conditions"`
	Severity    string                 `json:"severity"`
}

// RBACMetrics RBAC 메트릭
type RBACMetrics struct {
	TotalRoles           int64            `json:"total_roles"`
	ActiveRoles          int64            `json:"active_roles"`
	TotalPermissions     int64            `json:"total_permissions"`
	ActivePermissions    int64            `json:"active_permissions"`
	TotalUsers           int64            `json:"total_users"`
	UsersWithRoles       int64            `json:"users_with_roles"`
	TotalGroups          int64            `json:"total_groups"`
	ActiveGroups         int64            `json:"active_groups"`
	RoleAssignments      int64            `json:"role_assignments"`
	PermissionChecks     int64            `json:"permission_checks"`
	AverageRolesPerUser  float64          `json:"average_roles_per_user"`
	TopPermissions       []string         `json:"top_permissions"`
	MostAssignedRoles    []string         `json:"most_assigned_roles"`
	UnusedRoles          []string         `json:"unused_roles"`
	ConflictingPermissions int64          `json:"conflicting_permissions"`
}

// NewRBACFilter RBAC 필터 생성자
func NewRBACFilter() *RBACFilter {
	return &RBACFilter{
		Limit:  50,
		Offset: 0,
		SortBy: "created_at",
		Order:  "desc",
	}
}

// NewRBACQueryOptions RBAC 쿼리 옵션 생성자
func NewRBACQueryOptions() *RBACQueryOptions {
	return &RBACQueryOptions{
		IncludeInactive:  false,
		IncludeSystem:    false,
		IncludeExpired:   false,
		LoadRelations:    false,
		ComputeHierarchy: false,
	}
}