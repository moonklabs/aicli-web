package memory

import (
	"context"
	"sync"
	"time"
	
	"github.com/aicli/aicli-web/internal/models"
)

// RBACStorage 메모리 기반 RBAC 스토리지
type RBACStorage struct {
	mu            sync.RWMutex
	users         map[string]*models.User
	roles         map[string]*models.Role
	permissions   map[string]*models.Permission
	userRoles     map[string][]string // userID -> roleIDs
}

// NewRBACStorage 새 RBAC 스토리지 생성
func NewRBACStorage() *RBACStorage {
	return &RBACStorage{
		users:         make(map[string]*models.User),
		roles:         make(map[string]*models.Role),
		permissions:   make(map[string]*models.Permission),
		userRoles:     make(map[string][]string),
	}
}

// CreateUser 사용자 생성
func (r *RBACStorage) CreateUser(ctx context.Context, user *models.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.users[user.ID] = user
	return nil
}

// GetUserByID ID로 사용자 조회
func (r *RBACStorage) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	user, exists := r.users[id]
	if !exists {
		return nil, ErrNotFound
	}
	return user, nil
}

// GetUserByEmail 이메일로 사용자 조회
func (r *RBACStorage) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	for _, user := range r.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, ErrNotFound
}

// UpdateUser 사용자 업데이트
func (r *RBACStorage) UpdateUser(ctx context.Context, id string, updates map[string]interface{}) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if _, exists := r.users[id]; !exists {
		return ErrNotFound
	}
	
	// 간단한 구현 - 실제로는 updates를 적용해야 함
	return nil
}

// DeleteUser 사용자 삭제
func (r *RBACStorage) DeleteUser(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	delete(r.users, id)
	delete(r.userRoles, id)
	return nil
}

// CreateRole 역할 생성
func (r *RBACStorage) CreateRole(ctx context.Context, role *models.Role) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.roles[role.ID] = role
	return nil
}

// GetRoleByID ID로 역할 조회
func (r *RBACStorage) GetRoleByID(ctx context.Context, id string) (*models.Role, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	role, exists := r.roles[id]
	if !exists {
		return nil, ErrNotFound
	}
	return role, nil
}

// GetRoleByName 이름으로 역할 조회
func (r *RBACStorage) GetRoleByName(ctx context.Context, name string) (*models.Role, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	for _, role := range r.roles {
		if role.Name == name {
			return role, nil
		}
	}
	return nil, ErrNotFound
}

// listAllRoles 모든 역할 조회 (내부 헬퍼 메서드)
func (r *RBACStorage) listAllRoles(ctx context.Context) ([]*models.Role, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	roles := make([]*models.Role, 0, len(r.roles))
	for _, role := range r.roles {
		roles = append(roles, role)
	}
	return roles, nil
}

// updateRoleByID 역할 업데이트 (내부 헬퍼 메서드)
func (r *RBACStorage) updateRoleByID(ctx context.Context, id string, updates map[string]interface{}) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if _, exists := r.roles[id]; !exists {
		return ErrNotFound
	}
	
	// 간단한 구현
	return nil
}

// DeleteRole 역할 삭제
func (r *RBACStorage) DeleteRole(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	delete(r.roles, id)
	return nil
}

// CreatePermission 권한 생성
func (r *RBACStorage) CreatePermission(ctx context.Context, permission *models.Permission) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.permissions[permission.ID] = permission
	return nil
}

// GetPermissionByID ID로 권한 조회
func (r *RBACStorage) GetPermissionByID(ctx context.Context, id string) (*models.Permission, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	permission, exists := r.permissions[id]
	if !exists {
		return nil, ErrNotFound
	}
	return permission, nil
}

// listAllPermissions 모든 권한 조회 (내부 헬퍼 메서드)
func (r *RBACStorage) listAllPermissions(ctx context.Context) ([]*models.Permission, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	permissions := make([]*models.Permission, 0, len(r.permissions))
	for _, permission := range r.permissions {
		permissions = append(permissions, permission)
	}
	return permissions, nil
}

// DeletePermission 권한 삭제
func (r *RBACStorage) DeletePermission(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	delete(r.permissions, id)
	return nil
}

// AssignRole 사용자에게 역할 할당
func (r *RBACStorage) AssignRole(ctx context.Context, userID, roleID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if r.userRoles[userID] == nil {
		r.userRoles[userID] = []string{}
	}
	
	// 중복 체크
	for _, rid := range r.userRoles[userID] {
		if rid == roleID {
			return nil
		}
	}
	
	r.userRoles[userID] = append(r.userRoles[userID], roleID)
	return nil
}

// RemoveRole 사용자로부터 역할 제거
func (r *RBACStorage) RemoveRole(ctx context.Context, userID, roleID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	roles := r.userRoles[userID]
	for i, rid := range roles {
		if rid == roleID {
			r.userRoles[userID] = append(roles[:i], roles[i+1:]...)
			break
		}
	}
	return nil
}

// getUserRolesAsPointers 사용자의 역할 목록 조회 (포인터 타입으로)
func (r *RBACStorage) getUserRolesAsPointers(ctx context.Context, userID string) ([]*models.Role, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	roleIDs, exists := r.userRoles[userID]
	if !exists {
		return []*models.Role{}, nil
	}
	
	roles := make([]*models.Role, 0, len(roleIDs))
	for _, roleID := range roleIDs {
		if role, exists := r.roles[roleID]; exists {
			roles = append(roles, role)
		}
	}
	return roles, nil
}

// getRolePermissionsAsPermissions 역할의 권한 목록 조회 (Permission 타입으로)
func (r *RBACStorage) getRolePermissionsAsPermissions(ctx context.Context, roleID string) ([]*models.Permission, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	role, exists := r.roles[roleID]
	if !exists {
		return nil, ErrNotFound
	}
	
	// 간단한 구하 - Role.Permissions 필드 사용
	permissions := make([]*models.Permission, 0, len(role.Permissions))
	for i := range role.Permissions {
		permissions = append(permissions, &role.Permissions[i])
	}
	return permissions, nil
}

// GetUserPermissions 사용자의 모든 권한 조회
func (r *RBACStorage) GetUserPermissions(ctx context.Context, userID string) ([]*models.Permission, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	permMap := make(map[string]*models.Permission)
	
	// 사용자의 모든 역할에서 권한 수집
	if roleIDs, exists := r.userRoles[userID]; exists {
		for _, roleID := range roleIDs {
			if role, exists := r.roles[roleID]; exists {
				// Role.Permissions 필드에서 권한 수집
				for i := range role.Permissions {
					perm := &role.Permissions[i]
					permMap[perm.ID] = perm
				}
			}
		}
	}
	
	// 맵을 슬라이스로 변환
	permissions := make([]*models.Permission, 0, len(permMap))
	for _, perm := range permMap {
		permissions = append(permissions, perm)
	}
	return permissions, nil
}

// HasPermission 사용자가 특정 권한을 가지고 있는지 확인
func (r *RBACStorage) HasPermission(ctx context.Context, userID, resource, action string) (bool, error) {
	permissions, err := r.GetUserPermissions(ctx, userID)
	if err != nil {
		return false, err
	}
	
	// ResourceType을 문자열로 변환하여 비교하고, ActionType으로 변환하여 비교
	resourceType := models.ResourceType(resource)
	actionType := models.ActionType(action)
	
	for _, perm := range permissions {
		if perm.ResourceType == resourceType && perm.Action == actionType {
			return true, nil
		}
	}
	return false, nil
}

// AddUserToGroup 사용자를 그룹에 추가
func (r *RBACStorage) AddUserToGroup(ctx context.Context, userID, groupID, role string) error {
	// 간단한 구현 - 실제로는 UserGroupMember를 관리해야 함
	return nil
}

// 나머지 필수 메서드들의 스텁 구현
// TODO: 실제 구현 필요

func (r *RBACStorage) GetRolesByUserID(ctx context.Context, userID string) ([]models.Role, error) {
	roles, err := r.getUserRolesAsPointers(ctx, userID)
	if err != nil {
		return nil, err
	}
	result := make([]models.Role, 0, len(roles))
	for _, role := range roles {
		result = append(result, *role)
	}
	return result, nil
}

func (r *RBACStorage) GetRolesByGroupID(ctx context.Context, groupID string) ([]models.Role, error) {
	return []models.Role{}, nil
}

func (r *RBACStorage) GetRoleHierarchy(ctx context.Context, roleID string) ([]models.Role, error) {
	return []models.Role{}, nil
}

func (r *RBACStorage) GetAllRoles(ctx context.Context) ([]models.Role, error) {
	roles, err := r.listAllRoles(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]models.Role, 0, len(roles))
	for _, role := range roles {
		result = append(result, *role)
	}
	return result, nil
}

func (r *RBACStorage) GetChildRoles(ctx context.Context, parentID string) ([]models.Role, error) {
	return []models.Role{}, nil
}

func (r *RBACStorage) ListRoles(ctx context.Context, req models.ListRolesRequest) ([]models.Role, int64, error) {
	roles, err := r.listAllRoles(ctx)
	if err != nil {
		return nil, 0, err
	}
	result := make([]models.Role, 0, len(roles))
	for _, role := range roles {
		result = append(result, *role)
	}
	return result, int64(len(result)), nil
}

func (r *RBACStorage) UpdateRole(ctx context.Context, role *models.Role) error {
	return r.updateRoleByID(ctx, role.ID, nil)
}

func (r *RBACStorage) GetUsersByRoleID(ctx context.Context, roleID string) ([]string, error) {
	return []string{}, nil
}

func (r *RBACStorage) GetPermissionByName(ctx context.Context, name string) (*models.Permission, error) {
	return nil, ErrNotFound
}

func (r *RBACStorage) GetPermissionsByRoleID(ctx context.Context, roleID string) ([]models.Permission, error) {
	// 간단한 구현 - GetRolePermissions와 다른 반환 타입
	rolePerms, err := r.GetRolePermissions(ctx, roleID)
	if err != nil {
		return nil, err
	}
	result := make([]models.Permission, 0, len(rolePerms))
	for _, rp := range rolePerms {
		// RolePermission에서 Permission으로 변환
		if rp.Permission != nil {
			result = append(result, *rp.Permission)
		}
	}
	return result, nil
}

func (r *RBACStorage) GetAllPermissions(ctx context.Context) ([]models.Permission, error) {
	perms, err := r.listAllPermissions(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]models.Permission, 0, len(perms))
	for _, perm := range perms {
		result = append(result, *perm)
	}
	return result, nil
}

func (r *RBACStorage) GetPermissionsByResourceType(ctx context.Context, resourceType models.ResourceType) ([]models.Permission, error) {
	return []models.Permission{}, nil
}

func (r *RBACStorage) GetPermissionsByAction(ctx context.Context, action models.ActionType) ([]models.Permission, error) {
	return []models.Permission{}, nil
}

func (r *RBACStorage) ListPermissions(ctx context.Context, req models.ListPermissionsRequest) ([]models.Permission, int64, error) {
	perms, err := r.listAllPermissions(ctx)
	if err != nil {
		return nil, 0, err
	}
	result := make([]models.Permission, 0, len(perms))
	for _, perm := range perms {
		result = append(result, *perm)
	}
	return result, int64(len(result)), nil
}

func (r *RBACStorage) UpdatePermission(ctx context.Context, permission *models.Permission) error {
	return nil
}

// Resource methods
func (r *RBACStorage) CreateResource(ctx context.Context, resource *models.Resource) error {
	return nil
}

func (r *RBACStorage) GetResourceByID(ctx context.Context, resourceID string) (*models.Resource, error) {
	return nil, ErrNotFound
}

func (r *RBACStorage) GetResourceByIdentifier(ctx context.Context, resourceType models.ResourceType, identifier string) (*models.Resource, error) {
	return nil, ErrNotFound
}

func (r *RBACStorage) GetResourceHierarchy(ctx context.Context, resourceID string) ([]models.Resource, error) {
	return []models.Resource{}, nil
}

func (r *RBACStorage) GetResourcesByType(ctx context.Context, resourceType models.ResourceType) ([]models.Resource, error) {
	return []models.Resource{}, nil
}

func (r *RBACStorage) GetChildResources(ctx context.Context, parentID string) ([]models.Resource, error) {
	return []models.Resource{}, nil
}

func (r *RBACStorage) UpdateResource(ctx context.Context, resourceID string, updates map[string]interface{}) error {
	return nil
}

func (r *RBACStorage) DeleteResource(ctx context.Context, resourceID string) error {
	return nil
}

// UserGroup methods
func (r *RBACStorage) CreateUserGroup(ctx context.Context, group *models.UserGroup) error {
	return nil
}

func (r *RBACStorage) GetUserGroupByID(ctx context.Context, groupID string) (*models.UserGroup, error) {
	return nil, ErrNotFound
}

func (r *RBACStorage) GetUserGroupByName(ctx context.Context, name string) (*models.UserGroup, error) {
	return nil, ErrNotFound
}

func (r *RBACStorage) GetUserGroups(ctx context.Context, userID string) ([]models.UserGroup, error) {
	return []models.UserGroup{}, nil
}

func (r *RBACStorage) GetGroupsByType(ctx context.Context, groupType string) ([]models.UserGroup, error) {
	return []models.UserGroup{}, nil
}

func (r *RBACStorage) GetGroupHierarchy(ctx context.Context, groupID string) ([]models.UserGroup, error) {
	return []models.UserGroup{}, nil
}

func (r *RBACStorage) GetChildGroups(ctx context.Context, parentID string) ([]models.UserGroup, error) {
	return []models.UserGroup{}, nil
}

func (r *RBACStorage) UpdateUserGroup(ctx context.Context, groupID string, updates map[string]interface{}) error {
	return nil
}

func (r *RBACStorage) DeleteUserGroup(ctx context.Context, groupID string) error {
	return nil
}

// RolePermission methods
func (r *RBACStorage) AssignPermissionToRole(ctx context.Context, roleID, permissionID string, effect models.PermissionEffect, conditions string) error {
	return nil
}

func (r *RBACStorage) RevokePermissionFromRole(ctx context.Context, roleID, permissionID string) error {
	return nil
}

func (r *RBACStorage) GetRolePermissions(ctx context.Context, roleID string) ([]models.RolePermission, error) {
	return []models.RolePermission{}, nil
}

func (r *RBACStorage) UpdateRolePermission(ctx context.Context, roleID, permissionID string, effect models.PermissionEffect, conditions string) error {
	return nil
}

// UserRole methods
func (r *RBACStorage) AssignRoleToUser(ctx context.Context, userRole *models.UserRole) error {
	return r.AssignRole(ctx, userRole.UserID, userRole.RoleID)
}

func (r *RBACStorage) RevokeRoleFromUser(ctx context.Context, userID, roleID string, resourceID *string) error {
	return r.RemoveRole(ctx, userID, roleID)
}

func (r *RBACStorage) GetUserRoles(ctx context.Context, userID string, resourceID *string) ([]models.UserRole, error) {
	return []models.UserRole{}, nil
}

func (r *RBACStorage) GetUsersInRole(ctx context.Context, roleID string, resourceID *string) ([]models.UserRole, error) {
	return []models.UserRole{}, nil
}

func (r *RBACStorage) UpdateUserRole(ctx context.Context, userID, roleID string, resourceID *string, expiresAt *time.Time, isActive bool) error {
	return nil
}

// UserGroupMember methods
func (r *RBACStorage) RemoveUserFromGroup(ctx context.Context, userID, groupID string) error {
	return nil
}

func (r *RBACStorage) GetGroupMembers(ctx context.Context, groupID string) ([]models.UserGroupMember, error) {
	return []models.UserGroupMember{}, nil
}

func (r *RBACStorage) GetUserGroupMemberships(ctx context.Context, userID string) ([]models.UserGroupMember, error) {
	return []models.UserGroupMember{}, nil
}

func (r *RBACStorage) UpdateGroupMembership(ctx context.Context, userID, groupID, role string, isActive bool) error {
	return nil
}

// GroupRole methods
func (r *RBACStorage) AssignRoleToGroup(ctx context.Context, groupID, roleID, assignedBy string, resourceID *string, expiresAt *time.Time) error {
	return nil
}

func (r *RBACStorage) RevokeRoleFromGroup(ctx context.Context, groupID, roleID string, resourceID *string) error {
	return nil
}

func (r *RBACStorage) GetGroupRoles(ctx context.Context, groupID string, resourceID *string) ([]models.GroupRole, error) {
	return []models.GroupRole{}, nil
}

func (r *RBACStorage) GetGroupsInRole(ctx context.Context, roleID string, resourceID *string) ([]models.GroupRole, error) {
	return []models.GroupRole{}, nil
}

func (r *RBACStorage) UpdateGroupRole(ctx context.Context, groupID, roleID string, resourceID *string, expiresAt *time.Time, isActive bool) error {
	return nil
}

// Permission checking methods
func (r *RBACStorage) CheckUserPermission(ctx context.Context, userID string, resourceType models.ResourceType, resourceID string, action models.ActionType) (bool, error) {
	return r.HasPermission(ctx, userID, string(resourceType), string(action))
}

func (r *RBACStorage) GetUserEffectivePermissions(ctx context.Context, userID string) ([]models.PermissionDecision, error) {
	return []models.PermissionDecision{}, nil
}

func (r *RBACStorage) GetUserPermissionMatrix(ctx context.Context, userID string) (*models.UserPermissionMatrix, error) {
	return nil, nil
}

// Statistics methods
func (r *RBACStorage) GetRoleUsageStats(ctx context.Context, roleID string) (map[string]int64, error) {
	return map[string]int64{}, nil
}

func (r *RBACStorage) GetPermissionUsageStats(ctx context.Context, permissionID string) (map[string]int64, error) {
	return map[string]int64{}, nil
}

func (r *RBACStorage) GetUserRoleHistory(ctx context.Context, userID string, limit int) ([]models.UserRole, error) {
	return []models.UserRole{}, nil
}

func (r *RBACStorage) GetGroupRoleHistory(ctx context.Context, groupID string, limit int) ([]models.GroupRole, error) {
	return []models.GroupRole{}, nil
}