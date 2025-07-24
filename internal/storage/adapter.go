package storage

import (
	"context"
	"time"
	
	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/storage/memory"
)

// memoryAdapter adapts memory.Storage to storage.Storage interface
type memoryAdapter struct {
	mem *memory.Storage
}

// NewMemoryAdapter creates a new memory adapter
func NewMemoryAdapter() Storage {
	return &memoryAdapter{
		mem: memory.New(),
	}
}

// Workspace returns workspace storage
func (m *memoryAdapter) Workspace() WorkspaceStorage {
	return &workspaceAdapter{ws: m.mem.Workspace()}
}

// Project returns project storage
func (m *memoryAdapter) Project() ProjectStorage {
	return &projectAdapter{ps: m.mem.Project()}
}

// Session returns session storage
func (m *memoryAdapter) Session() SessionStorage {
	return &sessionAdapter{ss: m.mem.Session()}
}

// Task returns task storage
func (m *memoryAdapter) Task() TaskStorage {
	return &taskAdapter{ts: m.mem.Task()}
}

// RBAC returns RBAC storage
func (m *memoryAdapter) RBAC() RBACStorage {
	return &rbacAdapter{rs: m.mem.RBAC()}
}

// Close closes the storage
func (m *memoryAdapter) Close() error {
	return m.mem.Close()
}

// workspaceAdapter adapts memory.WorkspaceStorage to storage.WorkspaceStorage
type workspaceAdapter struct {
	ws *memory.WorkspaceStorage
}

func (w *workspaceAdapter) Create(ctx context.Context, workspace *models.Workspace) error {
	return w.ws.Create(ctx, workspace)
}

func (w *workspaceAdapter) GetByID(ctx context.Context, id string) (*models.Workspace, error) {
	return w.ws.GetByID(ctx, id)
}

func (w *workspaceAdapter) GetByName(ctx context.Context, ownerID, name string) (*models.Workspace, error) {
	// GetByName은 memory.WorkspaceStorage에 없으므로 직접 구현
	workspaces, _, err := w.ws.GetByOwnerID(ctx, ownerID, &models.PaginationRequest{Page: 1, Limit: 1000})
	if err != nil {
		return nil, err
	}
	for _, ws := range workspaces {
		if ws.Name == name {
			return ws, nil
		}
	}
	return nil, ErrNotFound
}

func (w *workspaceAdapter) GetByOwnerID(ctx context.Context, ownerID string, pagination *models.PaginationRequest) ([]*models.Workspace, int, error) {
	return w.ws.GetByOwnerID(ctx, ownerID, pagination)
}

func (w *workspaceAdapter) CountByOwner(ctx context.Context, ownerID string) (int, error) {
	// 간단한 구현: GetByOwnerID를 호출하고 결과 개수 반환
	workspaces, count, err := w.ws.GetByOwnerID(ctx, ownerID, &models.PaginationRequest{Page: 1, Limit: 1000})
	if err != nil {
		return 0, err
	}
	_ = workspaces
	return count, nil
}

func (w *workspaceAdapter) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	return w.ws.Update(ctx, id, updates)
}

func (w *workspaceAdapter) Delete(ctx context.Context, id string) error {
	return w.ws.Delete(ctx, id)
}

func (w *workspaceAdapter) List(ctx context.Context, pagination *models.PaginationRequest) ([]*models.Workspace, int, error) {
	return w.ws.List(ctx, pagination)
}

func (w *workspaceAdapter) ExistsByName(ctx context.Context, ownerID, name string) (bool, error) {
	return w.ws.ExistsByName(ctx, ownerID, name)
}

// projectAdapter adapts memory.ProjectStorage to storage.ProjectStorage
type projectAdapter struct {
	ps *memory.ProjectStorage
}

func (p *projectAdapter) Create(ctx context.Context, project *models.Project) error {
	return p.ps.Create(ctx, project)
}

func (p *projectAdapter) GetByID(ctx context.Context, id string) (*models.Project, error) {
	return p.ps.GetByID(ctx, id)
}

func (p *projectAdapter) GetByWorkspaceID(ctx context.Context, workspaceID string, pagination *models.PaginationRequest) ([]*models.Project, int, error) {
	return p.ps.GetByWorkspaceID(ctx, workspaceID, pagination)
}

func (p *projectAdapter) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	return p.ps.Update(ctx, id, updates)
}

func (p *projectAdapter) Delete(ctx context.Context, id string) error {
	return p.ps.Delete(ctx, id)
}

func (p *projectAdapter) ExistsByName(ctx context.Context, workspaceID, name string) (bool, error) {
	return p.ps.ExistsByName(ctx, workspaceID, name)
}

func (p *projectAdapter) GetByPath(ctx context.Context, path string) (*models.Project, error) {
	return p.ps.GetByPath(ctx, path)
}

func (p *projectAdapter) GetByName(ctx context.Context, workspaceID, name string) (*models.Project, error) {
	// GetByName은 memory.ProjectStorage에 없으므로 직접 구현
	projects, _, err := p.ps.GetByWorkspaceID(ctx, workspaceID, &models.PaginationRequest{Page: 1, Limit: 1000})
	if err != nil {
		return nil, err
	}
	for _, proj := range projects {
		if proj.Name == name {
			return proj, nil
		}
	}
	return nil, ErrNotFound
}

func (p *projectAdapter) UpdateStatus(ctx context.Context, projectID string, status models.ProjectStatus) error {
	// UpdateStatus를 Update로 구현
	return p.ps.Update(ctx, projectID, map[string]interface{}{"status": status})
}

func (p *projectAdapter) CountByWorkspace(ctx context.Context, workspaceID string) (int, error) {
	// 간단한 구현: GetByWorkspaceID를 호출하고 결과 개수 반환
	projects, count, err := p.ps.GetByWorkspaceID(ctx, workspaceID, &models.PaginationRequest{Page: 1, Limit: 1000})
	if err != nil {
		return 0, err
	}
	_ = projects
	return count, nil
}

// sessionAdapter adapts memory.SessionStorage to storage.SessionStorage
type sessionAdapter struct {
	ss *memory.SessionStorage
}

func (s *sessionAdapter) Create(ctx context.Context, session *models.Session) error {
	return s.ss.Create(ctx, session)
}

func (s *sessionAdapter) GetByID(ctx context.Context, id string) (*models.Session, error) {
	return s.ss.GetByID(ctx, id)
}

func (s *sessionAdapter) List(ctx context.Context, filter *models.SessionFilter, paging *models.PaginationRequest) (*models.PaginationResponse, error) {
	return s.ss.List(ctx, filter, paging)
}

func (s *sessionAdapter) Update(ctx context.Context, session *models.Session) error {
	return s.ss.Update(ctx, session)
}

func (s *sessionAdapter) Delete(ctx context.Context, id string) error {
	return s.ss.Delete(ctx, id)
}

func (s *sessionAdapter) GetActiveCount(ctx context.Context, projectID string) (int64, error) {
	return s.ss.GetActiveCount(ctx, projectID)
}

// taskAdapter adapts memory.taskStorage to storage.TaskStorage
type taskAdapter struct {
	ts interface{} // Will be *memory.taskStorage but we use interface{} to avoid import cycle
}

func (t *taskAdapter) Create(ctx context.Context, task *models.Task) error {
	return t.ts.(interface {
		Create(context.Context, *models.Task) error
	}).Create(ctx, task)
}

func (t *taskAdapter) GetByID(ctx context.Context, id string) (*models.Task, error) {
	return t.ts.(interface {
		GetByID(context.Context, string) (*models.Task, error)
	}).GetByID(ctx, id)
}

func (t *taskAdapter) List(ctx context.Context, filter *models.TaskFilter, paging *models.PaginationRequest) ([]*models.Task, int, error) {
	return t.ts.(interface {
		List(context.Context, *models.TaskFilter, *models.PaginationRequest) ([]*models.Task, int, error)
	}).List(ctx, filter, paging)
}

func (t *taskAdapter) Update(ctx context.Context, task *models.Task) error {
	return t.ts.(interface {
		Update(context.Context, *models.Task) error
	}).Update(ctx, task)
}

func (t *taskAdapter) Delete(ctx context.Context, id string) error {
	return t.ts.(interface {
		Delete(context.Context, string) error
	}).Delete(ctx, id)
}

func (t *taskAdapter) GetBySessionID(ctx context.Context, sessionID string, paging *models.PaginationRequest) ([]*models.Task, int, error) {
	return t.ts.(interface {
		GetBySessionID(context.Context, string, *models.PaginationRequest) ([]*models.Task, int, error)
	}).GetBySessionID(ctx, sessionID, paging)
}

func (t *taskAdapter) GetActiveCount(ctx context.Context, sessionID string) (int64, error) {
	return t.ts.(interface {
		GetActiveCount(context.Context, string) (int64, error)
	}).GetActiveCount(ctx, sessionID)
}

// rbacAdapter adapts memory.RBACStorage to storage.RBACStorage
type rbacAdapter struct {
	rs *memory.RBACStorage
}

// Implement all RBACStorage methods by delegating to memory.RBACStorage
func (r *rbacAdapter) CreateRole(ctx context.Context, role *models.Role) error {
	return r.rs.CreateRole(ctx, role)
}

func (r *rbacAdapter) GetRoleByID(ctx context.Context, roleID string) (*models.Role, error) {
	return r.rs.GetRoleByID(ctx, roleID)
}

func (r *rbacAdapter) GetRoleByName(ctx context.Context, name string) (*models.Role, error) {
	return r.rs.GetRoleByName(ctx, name)
}

func (r *rbacAdapter) GetRolesByUserID(ctx context.Context, userID string) ([]models.Role, error) {
	return r.rs.GetRolesByUserID(ctx, userID)
}

func (r *rbacAdapter) GetRolesByGroupID(ctx context.Context, groupID string) ([]models.Role, error) {
	return r.rs.GetRolesByGroupID(ctx, groupID)
}

func (r *rbacAdapter) GetRoleHierarchy(ctx context.Context, roleID string) ([]models.Role, error) {
	return r.rs.GetRoleHierarchy(ctx, roleID)
}

func (r *rbacAdapter) GetAllRoles(ctx context.Context) ([]models.Role, error) {
	return r.rs.GetAllRoles(ctx)
}

func (r *rbacAdapter) GetChildRoles(ctx context.Context, parentID string) ([]models.Role, error) {
	return r.rs.GetChildRoles(ctx, parentID)
}

func (r *rbacAdapter) ListRoles(ctx context.Context, req models.ListRolesRequest) ([]models.Role, int64, error) {
	return r.rs.ListRoles(ctx, req)
}

func (r *rbacAdapter) UpdateRole(ctx context.Context, role *models.Role) error {
	return r.rs.UpdateRole(ctx, role)
}

func (r *rbacAdapter) DeleteRole(ctx context.Context, roleID string) error {
	return r.rs.DeleteRole(ctx, roleID)
}

func (r *rbacAdapter) GetUsersByRoleID(ctx context.Context, roleID string) ([]string, error) {
	return r.rs.GetUsersByRoleID(ctx, roleID)
}

func (r *rbacAdapter) CreatePermission(ctx context.Context, permission *models.Permission) error {
	return r.rs.CreatePermission(ctx, permission)
}

func (r *rbacAdapter) GetPermissionByID(ctx context.Context, permissionID string) (*models.Permission, error) {
	return r.rs.GetPermissionByID(ctx, permissionID)
}

func (r *rbacAdapter) GetPermissionByName(ctx context.Context, name string) (*models.Permission, error) {
	return r.rs.GetPermissionByName(ctx, name)
}

func (r *rbacAdapter) GetPermissionsByRoleID(ctx context.Context, roleID string) ([]models.Permission, error) {
	return r.rs.GetPermissionsByRoleID(ctx, roleID)
}

func (r *rbacAdapter) GetAllPermissions(ctx context.Context) ([]models.Permission, error) {
	return r.rs.GetAllPermissions(ctx)
}

func (r *rbacAdapter) GetPermissionsByResourceType(ctx context.Context, resourceType models.ResourceType) ([]models.Permission, error) {
	return r.rs.GetPermissionsByResourceType(ctx, resourceType)
}

func (r *rbacAdapter) GetPermissionsByAction(ctx context.Context, action models.ActionType) ([]models.Permission, error) {
	return r.rs.GetPermissionsByAction(ctx, action)
}

func (r *rbacAdapter) ListPermissions(ctx context.Context, req models.ListPermissionsRequest) ([]models.Permission, int64, error) {
	return r.rs.ListPermissions(ctx, req)
}

func (r *rbacAdapter) UpdatePermission(ctx context.Context, permission *models.Permission) error {
	return r.rs.UpdatePermission(ctx, permission)
}

func (r *rbacAdapter) DeletePermission(ctx context.Context, permissionID string) error {
	return r.rs.DeletePermission(ctx, permissionID)
}

func (r *rbacAdapter) CreateResource(ctx context.Context, resource *models.Resource) error {
	return r.rs.CreateResource(ctx, resource)
}

func (r *rbacAdapter) GetResourceByID(ctx context.Context, resourceID string) (*models.Resource, error) {
	return r.rs.GetResourceByID(ctx, resourceID)
}

func (r *rbacAdapter) GetResourceByIdentifier(ctx context.Context, resourceType models.ResourceType, identifier string) (*models.Resource, error) {
	return r.rs.GetResourceByIdentifier(ctx, resourceType, identifier)
}

func (r *rbacAdapter) GetResourceHierarchy(ctx context.Context, resourceID string) ([]models.Resource, error) {
	return r.rs.GetResourceHierarchy(ctx, resourceID)
}

func (r *rbacAdapter) GetResourcesByType(ctx context.Context, resourceType models.ResourceType) ([]models.Resource, error) {
	return r.rs.GetResourcesByType(ctx, resourceType)
}

func (r *rbacAdapter) GetChildResources(ctx context.Context, parentID string) ([]models.Resource, error) {
	return r.rs.GetChildResources(ctx, parentID)
}

func (r *rbacAdapter) UpdateResource(ctx context.Context, resourceID string, updates map[string]interface{}) error {
	return r.rs.UpdateResource(ctx, resourceID, updates)
}

func (r *rbacAdapter) DeleteResource(ctx context.Context, resourceID string) error {
	return r.rs.DeleteResource(ctx, resourceID)
}

func (r *rbacAdapter) CreateUserGroup(ctx context.Context, group *models.UserGroup) error {
	return r.rs.CreateUserGroup(ctx, group)
}

func (r *rbacAdapter) GetUserGroupByID(ctx context.Context, groupID string) (*models.UserGroup, error) {
	return r.rs.GetUserGroupByID(ctx, groupID)
}

func (r *rbacAdapter) GetUserGroupByName(ctx context.Context, name string) (*models.UserGroup, error) {
	return r.rs.GetUserGroupByName(ctx, name)
}

func (r *rbacAdapter) GetUserGroups(ctx context.Context, userID string) ([]models.UserGroup, error) {
	return r.rs.GetUserGroups(ctx, userID)
}

func (r *rbacAdapter) GetGroupsByType(ctx context.Context, groupType string) ([]models.UserGroup, error) {
	return r.rs.GetGroupsByType(ctx, groupType)
}

func (r *rbacAdapter) GetGroupHierarchy(ctx context.Context, groupID string) ([]models.UserGroup, error) {
	return r.rs.GetGroupHierarchy(ctx, groupID)
}

func (r *rbacAdapter) GetChildGroups(ctx context.Context, parentID string) ([]models.UserGroup, error) {
	return r.rs.GetChildGroups(ctx, parentID)
}

func (r *rbacAdapter) UpdateUserGroup(ctx context.Context, groupID string, updates map[string]interface{}) error {
	return r.rs.UpdateUserGroup(ctx, groupID, updates)
}

func (r *rbacAdapter) DeleteUserGroup(ctx context.Context, groupID string) error {
	return r.rs.DeleteUserGroup(ctx, groupID)
}

func (r *rbacAdapter) AssignPermissionToRole(ctx context.Context, roleID, permissionID string, effect models.PermissionEffect, conditions string) error {
	return r.rs.AssignPermissionToRole(ctx, roleID, permissionID, effect, conditions)
}

func (r *rbacAdapter) RevokePermissionFromRole(ctx context.Context, roleID, permissionID string) error {
	return r.rs.RevokePermissionFromRole(ctx, roleID, permissionID)
}

func (r *rbacAdapter) GetRolePermissions(ctx context.Context, roleID string) ([]models.RolePermission, error) {
	return r.rs.GetRolePermissions(ctx, roleID)
}

func (r *rbacAdapter) UpdateRolePermission(ctx context.Context, roleID, permissionID string, effect models.PermissionEffect, conditions string) error {
	return r.rs.UpdateRolePermission(ctx, roleID, permissionID, effect, conditions)
}

func (r *rbacAdapter) AssignRoleToUser(ctx context.Context, userRole *models.UserRole) error {
	return r.rs.AssignRoleToUser(ctx, userRole)
}

func (r *rbacAdapter) RevokeRoleFromUser(ctx context.Context, userID, roleID string, resourceID *string) error {
	return r.rs.RevokeRoleFromUser(ctx, userID, roleID, resourceID)
}

func (r *rbacAdapter) GetUserRoles(ctx context.Context, userID string, resourceID *string) ([]models.UserRole, error) {
	return r.rs.GetUserRoles(ctx, userID, resourceID)
}

func (r *rbacAdapter) GetUsersInRole(ctx context.Context, roleID string, resourceID *string) ([]models.UserRole, error) {
	return r.rs.GetUsersInRole(ctx, roleID, resourceID)
}

func (r *rbacAdapter) UpdateUserRole(ctx context.Context, userID, roleID string, resourceID *string, expiresAt *time.Time, isActive bool) error {
	return r.rs.UpdateUserRole(ctx, userID, roleID, resourceID, expiresAt, isActive)
}

func (r *rbacAdapter) AddUserToGroup(ctx context.Context, userID, groupID, role string) error {
	return r.rs.AddUserToGroup(ctx, userID, groupID, role)
}

func (r *rbacAdapter) RemoveUserFromGroup(ctx context.Context, userID, groupID string) error {
	return r.rs.RemoveUserFromGroup(ctx, userID, groupID)
}

func (r *rbacAdapter) GetGroupMembers(ctx context.Context, groupID string) ([]models.UserGroupMember, error) {
	return r.rs.GetGroupMembers(ctx, groupID)
}

func (r *rbacAdapter) GetUserGroupMemberships(ctx context.Context, userID string) ([]models.UserGroupMember, error) {
	return r.rs.GetUserGroupMemberships(ctx, userID)
}

func (r *rbacAdapter) UpdateGroupMembership(ctx context.Context, userID, groupID, role string, isActive bool) error {
	return r.rs.UpdateGroupMembership(ctx, userID, groupID, role, isActive)
}

func (r *rbacAdapter) AssignRoleToGroup(ctx context.Context, groupID, roleID, assignedBy string, resourceID *string, expiresAt *time.Time) error {
	return r.rs.AssignRoleToGroup(ctx, groupID, roleID, assignedBy, resourceID, expiresAt)
}

func (r *rbacAdapter) RevokeRoleFromGroup(ctx context.Context, groupID, roleID string, resourceID *string) error {
	return r.rs.RevokeRoleFromGroup(ctx, groupID, roleID, resourceID)
}

func (r *rbacAdapter) GetGroupRoles(ctx context.Context, groupID string, resourceID *string) ([]models.GroupRole, error) {
	return r.rs.GetGroupRoles(ctx, groupID, resourceID)
}

func (r *rbacAdapter) GetGroupsInRole(ctx context.Context, roleID string, resourceID *string) ([]models.GroupRole, error) {
	return r.rs.GetGroupsInRole(ctx, roleID, resourceID)
}

func (r *rbacAdapter) UpdateGroupRole(ctx context.Context, groupID, roleID string, resourceID *string, expiresAt *time.Time, isActive bool) error {
	return r.rs.UpdateGroupRole(ctx, groupID, roleID, resourceID, expiresAt, isActive)
}

func (r *rbacAdapter) CheckUserPermission(ctx context.Context, userID string, resourceType models.ResourceType, resourceID string, action models.ActionType) (bool, error) {
	return r.rs.CheckUserPermission(ctx, userID, resourceType, resourceID, action)
}

func (r *rbacAdapter) GetUserEffectivePermissions(ctx context.Context, userID string) ([]models.PermissionDecision, error) {
	return r.rs.GetUserEffectivePermissions(ctx, userID)
}

func (r *rbacAdapter) GetUserPermissionMatrix(ctx context.Context, userID string) (*models.UserPermissionMatrix, error) {
	return r.rs.GetUserPermissionMatrix(ctx, userID)
}

func (r *rbacAdapter) GetRoleUsageStats(ctx context.Context, roleID string) (map[string]int64, error) {
	return r.rs.GetRoleUsageStats(ctx, roleID)
}

func (r *rbacAdapter) GetPermissionUsageStats(ctx context.Context, permissionID string) (map[string]int64, error) {
	return r.rs.GetPermissionUsageStats(ctx, permissionID)
}

func (r *rbacAdapter) GetUserRoleHistory(ctx context.Context, userID string, limit int) ([]models.UserRole, error) {
	return r.rs.GetUserRoleHistory(ctx, userID, limit)
}

func (r *rbacAdapter) GetGroupRoleHistory(ctx context.Context, groupID string, limit int) ([]models.GroupRole, error) {
	return r.rs.GetGroupRoleHistory(ctx, groupID, limit)
}

// Additional methods required by the extended interface
func (r *rbacAdapter) CreateUser(ctx context.Context, user *models.User) error {
	return r.rs.CreateUser(ctx, user)
}

func (r *rbacAdapter) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	return r.rs.GetUserByID(ctx, id)
}

func (r *rbacAdapter) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	return r.rs.GetUserByEmail(ctx, email)
}

func (r *rbacAdapter) UpdateUser(ctx context.Context, id string, updates map[string]interface{}) error {
	return r.rs.UpdateUser(ctx, id, updates)
}

func (r *rbacAdapter) DeleteUser(ctx context.Context, id string) error {
	return r.rs.DeleteUser(ctx, id)
}

func (r *rbacAdapter) AssignRole(ctx context.Context, userID, roleID string) error {
	return r.rs.AssignRole(ctx, userID, roleID)
}

func (r *rbacAdapter) RemoveRole(ctx context.Context, userID, roleID string) error {
	return r.rs.RemoveRole(ctx, userID, roleID)
}

func (r *rbacAdapter) GetUserPermissions(ctx context.Context, userID string) ([]*models.Permission, error) {
	return r.rs.GetUserPermissions(ctx, userID)
}

func (r *rbacAdapter) HasPermission(ctx context.Context, userID, resource, action string) (bool, error) {
	return r.rs.HasPermission(ctx, userID, resource, action)
}