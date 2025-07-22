package integration

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/aicli/aicli-web/internal/auth"
	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/storage"
	"github.com/aicli/aicli-web/internal/storage/memory"
)

// RBACScenarioTestSuite RBAC 시나리오 기반 통합 테스트 스위트
type RBACScenarioTestSuite struct {
	suite.Suite
	storage       storage.Storage
	rbacManager   auth.RBACManager
	cache         auth.PermissionCache
	testUsers     map[string]*models.User
	testRoles     map[string]*models.Role
	testGroups    map[string]*models.UserGroup
	testResources map[string]*models.Resource
	testPermissions map[string]*models.Permission
}

// SetupSuite 테스트 스위트 초기화
func (suite *RBACScenarioTestSuite) SetupSuite() {
	// 메모리 저장소 초기화
	suite.storage = memory.NewMemoryStorage()
	
	// 캐시 초기화 (메모리 기반)
	suite.cache = auth.NewMemoryPermissionCache()
	
	// RBAC 매니저 초기화
	rbacStorage := auth.NewRBACStorageAdapter(suite.storage)
	suite.rbacManager = auth.NewRBACManager(rbacStorage, suite.cache)
	
	// 테스트 데이터 초기화
	suite.setupTestData()
}

// setupTestData 테스트용 데이터 설정
func (suite *RBACScenarioTestSuite) setupTestData() {
	ctx := context.Background()
	
	// 테스트 데이터 맵 초기화
	suite.testUsers = make(map[string]*models.User)
	suite.testRoles = make(map[string]*models.Role)
	suite.testGroups = make(map[string]*models.UserGroup)
	suite.testResources = make(map[string]*models.Resource)
	suite.testPermissions = make(map[string]*models.Permission)
	
	// 역할 계층 생성 (시스템 관리자 -> 조직 관리자 -> 팀 리더 -> 일반 사용자)
	suite.createRoleHierarchy(ctx)
	
	// 사용자 그룹 생성 (조직 -> 부서 -> 팀)
	suite.createUserGroups(ctx)
	
	// 리소스 계층 생성 (조직 -> 프로젝트 -> 워크스페이스)
	suite.createResourceHierarchy(ctx)
	
	// 권한 정의
	suite.createPermissions(ctx)
	
	// 테스트 사용자 생성
	suite.createTestUsers(ctx)
	
	// 역할-권한 연결
	suite.assignPermissionsToRoles(ctx)
	
	// 사용자-역할 및 그룹 연결
	suite.assignUsersToRolesAndGroups(ctx)
}

// createRoleHierarchy 역할 계층 구조 생성
func (suite *RBACScenarioTestSuite) createRoleHierarchy(ctx context.Context) {
	roles := []*models.Role{
		{
			Base:        models.Base{ID: "role-system-admin"},
			Name:        "System Administrator",
			Description: "시스템 최고 관리자",
			Level:       1,
			ParentID:    nil,
			IsSystem:    true,
			IsActive:    true,
		},
		{
			Base:        models.Base{ID: "role-org-admin"},
			Name:        "Organization Administrator", 
			Description: "조직 관리자",
			Level:       2,
			ParentID:    stringPtr("role-system-admin"),
			IsSystem:    false,
			IsActive:    true,
		},
		{
			Base:        models.Base{ID: "role-team-lead"},
			Name:        "Team Leader",
			Description: "팀 리더",
			Level:       3,
			ParentID:    stringPtr("role-org-admin"),
			IsSystem:    false,
			IsActive:    true,
		},
		{
			Base:        models.Base{ID: "role-user"},
			Name:        "User",
			Description: "일반 사용자",
			Level:       4,
			ParentID:    stringPtr("role-team-lead"),
			IsSystem:    false,
			IsActive:    true,
		},
		{
			Base:        models.Base{ID: "role-guest"},
			Name:        "Guest",
			Description: "게스트 사용자",
			Level:       5,
			ParentID:    nil,
			IsSystem:    false,
			IsActive:    true,
		},
	}
	
	for _, role := range roles {
		role.CreatedAt = time.Now()
		role.UpdatedAt = time.Now()
		err := suite.storage.Create(ctx, "roles", role)
		require.NoError(suite.T(), err)
		suite.testRoles[role.ID] = role
	}
}

// createUserGroups 사용자 그룹 생성
func (suite *RBACScenarioTestSuite) createUserGroups(ctx context.Context) {
	groups := []*models.UserGroup{
		{
			Base:        models.Base{ID: "group-engineering"},
			Name:        "Engineering",
			Description: "엔지니어링 조직",
			Type:        "organization",
			ParentID:    nil,
			IsActive:    true,
		},
		{
			Base:        models.Base{ID: "group-backend-team"},
			Name:        "Backend Team",
			Description: "백엔드 개발팀",
			Type:        "team",
			ParentID:    stringPtr("group-engineering"),
			IsActive:    true,
		},
		{
			Base:        models.Base{ID: "group-frontend-team"},
			Name:        "Frontend Team", 
			Description: "프론트엔드 개발팀",
			Type:        "team",
			ParentID:    stringPtr("group-engineering"),
			IsActive:    true,
		},
		{
			Base:        models.Base{ID: "group-devops-team"},
			Name:        "DevOps Team",
			Description: "데브옵스팀",
			Type:        "team",
			ParentID:    stringPtr("group-engineering"),
			IsActive:    true,
		},
	}
	
	for _, group := range groups {
		group.CreatedAt = time.Now()
		group.UpdatedAt = time.Now()
		err := suite.storage.Create(ctx, "user_groups", group)
		require.NoError(suite.T(), err)
		suite.testGroups[group.ID] = group
	}
}

// createResourceHierarchy 리소스 계층 구조 생성
func (suite *RBACScenarioTestSuite) createResourceHierarchy(ctx context.Context) {
	resources := []*models.Resource{
		{
			Base:         models.Base{ID: "org-acme"},
			Name:         "ACME Corporation",
			Type:         models.ResourceTypeOrganization,
			ParentID:     nil,
			IsActive:     true,
		},
		{
			Base:         models.Base{ID: "project-web-app"},
			Name:         "Web Application",
			Type:         models.ResourceTypeProject,
			ParentID:     stringPtr("org-acme"),
			IsActive:     true,
		},
		{
			Base:         models.Base{ID: "project-mobile-app"},
			Name:         "Mobile Application",
			Type:         models.ResourceTypeProject,
			ParentID:     stringPtr("org-acme"),
			IsActive:     true,
		},
		{
			Base:         models.Base{ID: "workspace-backend"},
			Name:         "Backend Workspace",
			Type:         models.ResourceTypeWorkspace,
			ParentID:     stringPtr("project-web-app"),
			IsActive:     true,
		},
		{
			Base:         models.Base{ID: "workspace-frontend"},
			Name:         "Frontend Workspace",
			Type:         models.ResourceTypeWorkspace,
			ParentID:     stringPtr("project-web-app"),
			IsActive:     true,
		},
	}
	
	for _, resource := range resources {
		resource.CreatedAt = time.Now()
		resource.UpdatedAt = time.Now()
		err := suite.storage.Create(ctx, "resources", resource)
		require.NoError(suite.T(), err)
		suite.testResources[resource.ID] = resource
	}
}

// createPermissions 권한 정의 생성
func (suite *RBACScenarioTestSuite) createPermissions(ctx context.Context) {
	permissions := []*models.Permission{
		// 시스템 관리 권한
		{
			Base:         models.Base{ID: "perm-system-admin"},
			Name:         "System Administration",
			ResourceType: models.ResourceTypeSystem,
			Action:       models.ActionManage,
			Effect:       models.PermissionAllow,
			IsActive:     true,
		},
		// 조직 관리 권한
		{
			Base:         models.Base{ID: "perm-org-manage"},
			Name:         "Organization Management",
			ResourceType: models.ResourceTypeOrganization,
			Action:       models.ActionManage,
			Effect:       models.PermissionAllow,
			IsActive:     true,
		},
		// 프로젝트 관리 권한
		{
			Base:         models.Base{ID: "perm-project-manage"},
			Name:         "Project Management",
			ResourceType: models.ResourceTypeProject,
			Action:       models.ActionManage,
			Effect:       models.PermissionAllow,
			IsActive:     true,
		},
		// 워크스페이스 권한들
		{
			Base:         models.Base{ID: "perm-workspace-read"},
			Name:         "Workspace Read",
			ResourceType: models.ResourceTypeWorkspace,
			Action:       models.ActionRead,
			Effect:       models.PermissionAllow,
			IsActive:     true,
		},
		{
			Base:         models.Base{ID: "perm-workspace-write"},
			Name:         "Workspace Write", 
			ResourceType: models.ResourceTypeWorkspace,
			Action:       models.ActionWrite,
			Effect:       models.PermissionAllow,
			IsActive:     true,
		},
		{
			Base:         models.Base{ID: "perm-workspace-delete"},
			Name:         "Workspace Delete",
			ResourceType: models.ResourceTypeWorkspace,
			Action:       models.ActionDelete,
			Effect:       models.PermissionAllow,
			IsActive:     true,
		},
		// 사용자 관리 권한
		{
			Base:         models.Base{ID: "perm-user-manage"},
			Name:         "User Management",
			ResourceType: models.ResourceTypeUser,
			Action:       models.ActionManage,
			Effect:       models.PermissionAllow,
			IsActive:     true,
		},
		// 명시적 거부 권한 (테스트용)
		{
			Base:         models.Base{ID: "perm-workspace-delete-deny"},
			Name:         "Workspace Delete Deny",
			ResourceType: models.ResourceTypeWorkspace,
			Action:       models.ActionDelete,
			Effect:       models.PermissionDeny,
			IsActive:     true,
		},
	}
	
	for _, permission := range permissions {
		permission.CreatedAt = time.Now()
		permission.UpdatedAt = time.Now()
		err := suite.storage.Create(ctx, "permissions", permission)
		require.NoError(suite.T(), err)
		suite.testPermissions[permission.ID] = permission
	}
}

// createTestUsers 테스트 사용자 생성
func (suite *RBACScenarioTestSuite) createTestUsers(ctx context.Context) {
	users := []*models.User{
		{
			Base:     models.Base{ID: "user-admin"},
			Username: "admin",
			Email:    "admin@acme.com",
			IsActive: true,
		},
		{
			Base:     models.Base{ID: "user-john"},
			Username: "john",
			Email:    "john@acme.com",
			IsActive: true,
		},
		{
			Base:     models.Base{ID: "user-jane"},
			Username: "jane",
			Email:    "jane@acme.com",
			IsActive: true,
		},
		{
			Base:     models.Base{ID: "user-bob"},
			Username: "bob",
			Email:    "bob@acme.com",
			IsActive: true,
		},
		{
			Base:     models.Base{ID: "user-alice"},
			Username: "alice",
			Email:    "alice@acme.com",
			IsActive: true,
		},
	}
	
	for _, user := range users {
		user.CreatedAt = time.Now()
		user.UpdatedAt = time.Now()
		err := suite.storage.Create(ctx, "users", user)
		require.NoError(suite.T(), err)
		suite.testUsers[user.ID] = user
	}
}

// assignPermissionsToRoles 역할-권한 연결
func (suite *RBACScenarioTestSuite) assignPermissionsToRoles(ctx context.Context) {
	// 시스템 관리자: 모든 권한
	systemAdminPerms := []string{
		"perm-system-admin", "perm-org-manage", "perm-project-manage", 
		"perm-workspace-read", "perm-workspace-write", "perm-workspace-delete", "perm-user-manage",
	}
	suite.assignPermissionsToRole(ctx, "role-system-admin", systemAdminPerms)
	
	// 조직 관리자: 조직 및 하위 권한
	orgAdminPerms := []string{
		"perm-org-manage", "perm-project-manage", 
		"perm-workspace-read", "perm-workspace-write", "perm-user-manage",
	}
	suite.assignPermissionsToRole(ctx, "role-org-admin", orgAdminPerms)
	
	// 팀 리더: 프로젝트 관리 및 워크스페이스 권한
	teamLeadPerms := []string{
		"perm-project-manage", "perm-workspace-read", "perm-workspace-write",
	}
	suite.assignPermissionsToRole(ctx, "role-team-lead", teamLeadPerms)
	
	// 일반 사용자: 읽기/쓰기 권한
	userPerms := []string{
		"perm-workspace-read", "perm-workspace-write",
	}
	suite.assignPermissionsToRole(ctx, "role-user", userPerms)
	
	// 게스트: 읽기 권한만
	guestPerms := []string{
		"perm-workspace-read",
	}
	suite.assignPermissionsToRole(ctx, "role-guest", guestPerms)
}

// assignPermissionsToRole 특정 역할에 권한 할당
func (suite *RBACScenarioTestSuite) assignPermissionsToRole(ctx context.Context, roleID string, permissionIDs []string) {
	for _, permID := range permissionIDs {
		rolePermission := &models.RolePermission{
			Base:         models.Base{ID: fmt.Sprintf("rp-%s-%s", roleID, permID)},
			RoleID:       roleID,
			PermissionID: permID,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		
		err := suite.storage.Create(ctx, "role_permissions", rolePermission)
		require.NoError(suite.T(), err)
	}
}

// assignUsersToRolesAndGroups 사용자-역할/그룹 할당
func (suite *RBACScenarioTestSuite) assignUsersToRolesAndGroups(ctx context.Context) {
	// Admin: 시스템 관리자 역할
	suite.assignUserToRole(ctx, "user-admin", "role-system-admin", nil)
	
	// John: 조직 관리자, 엔지니어링 그룹
	suite.assignUserToRole(ctx, "user-john", "role-org-admin", nil)
	suite.assignUserToGroup(ctx, "user-john", "group-engineering")
	
	// Jane: 팀 리더, 백엔드 팀
	suite.assignUserToRole(ctx, "user-jane", "role-team-lead", stringPtr("project-web-app"))
	suite.assignUserToGroup(ctx, "user-jane", "group-backend-team")
	
	// Bob: 일반 사용자, 프론트엔드 팀
	suite.assignUserToRole(ctx, "user-bob", "role-user", stringPtr("workspace-frontend"))
	suite.assignUserToGroup(ctx, "user-bob", "group-frontend-team")
	
	// Alice: 게스트, 그룹 없음
	suite.assignUserToRole(ctx, "user-alice", "role-guest", nil)
}

// assignUserToRole 사용자에게 역할 할당
func (suite *RBACScenarioTestSuite) assignUserToRole(ctx context.Context, userID, roleID string, resourceID *string) {
	userRole := &models.UserRole{
		Base:       models.Base{ID: fmt.Sprintf("ur-%s-%s", userID, roleID)},
		UserID:     userID,
		RoleID:     roleID,
		ResourceID: resourceID,
		GrantedAt:  time.Now(),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	
	err := suite.storage.Create(ctx, "user_roles", userRole)
	require.NoError(suite.T(), err)
}

// assignUserToGroup 사용자를 그룹에 할당
func (suite *RBACScenarioTestSuite) assignUserToGroup(ctx context.Context, userID, groupID string) {
	userGroup := &models.UserGroupMembership{
		Base:      models.Base{ID: fmt.Sprintf("ugm-%s-%s", userID, groupID)},
		UserID:    userID,
		GroupID:   groupID,
		JoinedAt:  time.Now(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	err := suite.storage.Create(ctx, "user_group_memberships", userGroup)
	require.NoError(suite.T(), err)
}

// TestComplexRoleHierarchyScenario 복잡한 역할 계층 시나리오 테스트
func (suite *RBACScenarioTestSuite) TestComplexRoleHierarchyScenario() {
	ctx := context.Background()
	
	// 시나리오: Jane(팀 리더)이 시스템 관리 권한을 상속받는지 확인
	req := &models.CheckPermissionRequest{
		UserID:       "user-jane",
		ResourceType: models.ResourceTypeSystem,
		ResourceID:   "system",
		Action:       models.ActionManage,
	}
	
	resp, err := suite.rbacManager.CheckPermission(ctx, req)
	assert.NoError(suite.T(), err)
	
	// Jane은 팀 리더이므로 시스템 관리 권한이 없어야 함
	assert.False(suite.T(), resp.Allowed)
	assert.Equal(suite.T(), models.PermissionDeny, resp.Decision.Effect)
	
	// Admin은 시스템 관리 권한이 있어야 함
	req.UserID = "user-admin"
	resp, err = suite.rbacManager.CheckPermission(ctx, req)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), resp.Allowed)
}

// TestResourceScopedPermissions 리소스별 권한 범위 테스트
func (suite *RBACScenarioTestSuite) TestResourceScopedPermissions() {
	ctx := context.Background()
	
	// 시나리오: Bob은 frontend 워크스페이스에만 권한이 있음
	tests := []struct {
		name       string
		userID     string
		resourceID string
		action     models.ActionType
		expected   bool
	}{
		{
			name:       "Bob - Frontend workspace read (allowed)",
			userID:     "user-bob",
			resourceID: "workspace-frontend",
			action:     models.ActionRead,
			expected:   true,
		},
		{
			name:       "Bob - Frontend workspace write (allowed)",
			userID:     "user-bob", 
			resourceID: "workspace-frontend",
			action:     models.ActionWrite,
			expected:   true,
		},
		{
			name:       "Bob - Backend workspace read (denied - no permission)",
			userID:     "user-bob",
			resourceID: "workspace-backend",
			action:     models.ActionRead,
			expected:   false,
		},
		{
			name:       "Bob - Frontend workspace delete (denied - no permission)",
			userID:     "user-bob",
			resourceID: "workspace-frontend",
			action:     models.ActionDelete,
			expected:   false,
		},
	}
	
	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			req := &models.CheckPermissionRequest{
				UserID:       tt.userID,
				ResourceType: models.ResourceTypeWorkspace,
				ResourceID:   tt.resourceID,
				Action:       tt.action,
			}
			
			resp, err := suite.rbacManager.CheckPermission(ctx, req)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, resp.Allowed, "Test: %s", tt.name)
		})
	}
}

// TestGroupBasedPermissions 그룹 기반 권한 테스트
func (suite *RBACScenarioTestSuite) TestGroupBasedPermissions() {
	ctx := context.Background()
	
	// 그룹에 역할 할당 (백엔드 팀에 추가 권한 부여)
	groupRole := &models.GroupRole{
		Base:       models.Base{ID: "gr-backend-team-developer"},
		GroupID:    "group-backend-team",
		RoleID:     "role-user",  // 추가 사용자 권한
		ResourceID: stringPtr("workspace-backend"),
		GrantedAt:  time.Now(),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	
	err := suite.storage.Create(ctx, "group_roles", groupRole)
	require.NoError(suite.T(), err)
	
	// Jane은 백엔드 팀 멤버이므로 그룹을 통해 백엔드 워크스페이스 권한을 가져야 함
	req := &models.CheckPermissionRequest{
		UserID:       "user-jane",
		ResourceType: models.ResourceTypeWorkspace,
		ResourceID:   "workspace-backend", 
		Action:       models.ActionRead,
	}
	
	resp, err := suite.rbacManager.CheckPermission(ctx, req)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), resp.Allowed)
	
	// 그룹이 아닌 사용자는 해당 권한이 없어야 함
	req.UserID = "user-alice"  // 그룹 없는 사용자
	resp, err = suite.rbacManager.CheckPermission(ctx, req)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), resp.Allowed)
}

// TestPermissionConflictResolution 권한 충돌 해결 테스트
func (suite *RBACScenarioTestSuite) TestPermissionConflictResolution() {
	ctx := context.Background()
	
	// 충돌 시나리오: 사용자에게 허용과 거부 권한을 동시에 부여
	
	// 1. Bob에게 삭제 허용 역할 추가
	deleteAllowRole := &models.Role{
		Base:        models.Base{ID: "role-delete-allow"},
		Name:        "Delete Allow Role",
		Description: "삭제 허용 역할",
		Level:       4,
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	err := suite.storage.Create(ctx, "roles", deleteAllowRole)
	require.NoError(suite.T(), err)
	
	// 2. 삭제 허용 권한을 역할에 할당
	rolePermission := &models.RolePermission{
		Base:         models.Base{ID: "rp-delete-allow-workspace-delete"},
		RoleID:       deleteAllowRole.ID,
		PermissionID: "perm-workspace-delete",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	err = suite.storage.Create(ctx, "role_permissions", rolePermission)
	require.NoError(suite.T(), err)
	
	// 3. Bob에게 허용 역할 할당
	userRole := &models.UserRole{
		Base:       models.Base{ID: "ur-bob-delete-allow"},
		UserID:     "user-bob",
		RoleID:     deleteAllowRole.ID,
		ResourceID: stringPtr("workspace-frontend"),
		GrantedAt:  time.Now(),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	err = suite.storage.Create(ctx, "user_roles", userRole)
	require.NoError(suite.T(), err)
	
	// 4. 거부 역할 및 권한 생성
	denyRole := &models.Role{
		Base:        models.Base{ID: "role-delete-deny"},
		Name:        "Delete Deny Role",
		Description: "삭제 거부 역할",
		Level:       4,
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	err = suite.storage.Create(ctx, "roles", denyRole)
	require.NoError(suite.T(), err)
	
	// 5. 거부 권한을 역할에 할당
	denyRolePermission := &models.RolePermission{
		Base:         models.Base{ID: "rp-delete-deny-workspace-delete"},
		RoleID:       denyRole.ID,
		PermissionID: "perm-workspace-delete-deny",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	err = suite.storage.Create(ctx, "role_permissions", denyRolePermission)
	require.NoError(suite.T(), err)
	
	// 6. Bob에게 거부 역할도 할당
	denyUserRole := &models.UserRole{
		Base:       models.Base{ID: "ur-bob-delete-deny"},
		UserID:     "user-bob",
		RoleID:     denyRole.ID,
		ResourceID: stringPtr("workspace-frontend"),
		GrantedAt:  time.Now(),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	err = suite.storage.Create(ctx, "user_roles", denyUserRole)
	require.NoError(suite.T(), err)
	
	// 캐시 무효화
	suite.cache.InvalidateUser("user-bob")
	
	// 7. 권한 검사 - 거부가 우선해야 함
	req := &models.CheckPermissionRequest{
		UserID:       "user-bob",
		ResourceType: models.ResourceTypeWorkspace,
		ResourceID:   "workspace-frontend",
		Action:       models.ActionDelete,
	}
	
	resp, err := suite.rbacManager.CheckPermission(ctx, req)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), resp.Allowed)
	assert.Equal(suite.T(), models.PermissionDeny, resp.Decision.Effect)
}

// TestConcurrentPermissionChecks 동시 권한 검사 테스트
func (suite *RBACScenarioTestSuite) TestConcurrentPermissionChecks() {
	ctx := context.Background()
	
	const numGoroutines = 50
	const checksPerGoroutine = 10
	
	var wg sync.WaitGroup
	results := make(chan bool, numGoroutines*checksPerGoroutine)
	
	// 동시에 권한 검사 실행
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			
			for j := 0; j < checksPerGoroutine; j++ {
				req := &models.CheckPermissionRequest{
					UserID:       "user-jane",
					ResourceType: models.ResourceTypeProject,
					ResourceID:   "project-web-app",
					Action:       models.ActionManage,
				}
				
				resp, err := suite.rbacManager.CheckPermission(ctx, req)
				if err != nil {
					results <- false
					continue
				}
				
				results <- resp.Allowed
			}
		}(i)
	}
	
	wg.Wait()
	close(results)
	
	// 모든 결과가 일관되어야 함
	successCount := 0
	totalCount := 0
	for result := range results {
		totalCount++
		if result {
			successCount++
		}
	}
	
	expectedCount := numGoroutines * checksPerGoroutine
	assert.Equal(suite.T(), expectedCount, totalCount)
	
	// Jane은 팀 리더이므로 프로젝트 관리 권한이 있어야 함
	assert.Equal(suite.T(), expectedCount, successCount)
}

// TestPermissionCacheEffectiveness 권한 캐시 효과성 테스트
func (suite *RBACScenarioTestSuite) TestPermissionCacheEffectiveness() {
	ctx := context.Background()
	
	userID := "user-john"
	
	// 첫 번째 요청 (캐시 미스)
	start := time.Now()
	req := &models.CheckPermissionRequest{
		UserID:       userID,
		ResourceType: models.ResourceTypeOrganization,
		ResourceID:   "org-acme",
		Action:       models.ActionManage,
	}
	
	resp1, err := suite.rbacManager.CheckPermission(ctx, req)
	firstCallDuration := time.Since(start)
	
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), resp1.Allowed)
	
	// 두 번째 요청 (캐시 히트)
	start = time.Now()
	resp2, err := suite.rbacManager.CheckPermission(ctx, req)
	secondCallDuration := time.Since(start)
	
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), resp2.Allowed)
	
	// 캐시 히트가 훨씬 빨라야 함
	suite.T().Logf("First call: %v, Second call: %v", firstCallDuration, secondCallDuration)
	
	// 일반적으로 캐시 히트는 10배 이상 빨라야 함
	// 하지만 메모리 저장소에서는 차이가 크지 않을 수 있으므로 단순히 캐시 히트가 더 빠른지만 확인
	assert.True(suite.T(), secondCallDuration <= firstCallDuration)
}

// TestPermissionMatrixComputation 권한 매트릭스 계산 테스트
func (suite *RBACScenarioTestSuite) TestPermissionMatrixComputation() {
	ctx := context.Background()
	
	// John의 권한 매트릭스 계산
	matrix, err := suite.rbacManager.ComputeUserPermissionMatrix(ctx, "user-john")
	assert.NoError(suite.T(), err)
	
	// John은 조직 관리자 역할을 가져야 함
	assert.Contains(suite.T(), matrix.DirectRoles, "role-org-admin")
	
	// 상속된 역할들도 포함되어야 함
	assert.Contains(suite.T(), matrix.InheritedRoles, "role-system-admin")
	
	// 그룹 역할도 있어야 함 (그룹을 통한 역할이 있다면)
	// assert.NotEmpty(suite.T(), matrix.GroupRoles)
	
	// 최종 권한들이 올바르게 계산되었는지 확인
	assert.NotEmpty(suite.T(), matrix.FinalPermissions)
	
	// 조직 관리 권한이 있어야 함
	orgManageKey := "organization:*:manage"
	assert.Contains(suite.T(), matrix.FinalPermissions, orgManageKey)
	assert.Equal(suite.T(), models.PermissionAllow, matrix.FinalPermissions[orgManageKey].Effect)
}

// TestResourceHierarchyPermissions 리소스 계층 권한 테스트
func (suite *RBACScenarioTestSuite) TestResourceHierarchyPermissions() {
	ctx := context.Background()
	
	// Jane은 프로젝트 레벨에서 팀 리더 권한을 가짐
	// 하위 워크스페이스에서도 권한이 상속되어야 함
	
	tests := []struct {
		name       string
		resourceID string
		action     models.ActionType
		expected   bool
	}{
		{
			name:       "Project level - manage (allowed)",
			resourceID: "project-web-app",
			action:     models.ActionManage,
			expected:   true,
		},
		{
			name:       "Workspace level - read (inherited)",
			resourceID: "workspace-backend",
			action:     models.ActionRead,
			expected:   true,
		},
		{
			name:       "Workspace level - write (inherited)",
			resourceID: "workspace-frontend", 
			action:     models.ActionWrite,
			expected:   true,
		},
		{
			name:       "Different project - no permission",
			resourceID: "project-mobile-app",
			action:     models.ActionRead,
			expected:   false, // Jane은 mobile app에 권한 없음
		},
	}
	
	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			var resourceType models.ResourceType
			if strings.Contains(tt.resourceID, "project") {
				resourceType = models.ResourceTypeProject
			} else {
				resourceType = models.ResourceTypeWorkspace
			}
			
			req := &models.CheckPermissionRequest{
				UserID:       "user-jane",
				ResourceType: resourceType,
				ResourceID:   tt.resourceID,
				Action:       tt.action,
			}
			
			resp, err := suite.rbacManager.CheckPermission(ctx, req)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, resp.Allowed, "Test: %s", tt.name)
		})
	}
}

// TestDynamicRoleAssignment 동적 역할 할당 테스트
func (suite *RBACScenarioTestSuite) TestDynamicRoleAssignment() {
	ctx := context.Background()
	
	// Alice는 처음에 게스트 권한만 가짐
	req := &models.CheckPermissionRequest{
		UserID:       "user-alice",
		ResourceType: models.ResourceTypeWorkspace,
		ResourceID:   "workspace-frontend",
		Action:       models.ActionWrite,
	}
	
	resp, err := suite.rbacManager.CheckPermission(ctx, req)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), resp.Allowed) // 쓰기 권한 없음
	
	// Alice에게 동적으로 사용자 역할 할당
	newUserRole := &models.UserRole{
		Base:       models.Base{ID: "ur-alice-user"},
		UserID:     "user-alice",
		RoleID:     "role-user",
		ResourceID: stringPtr("workspace-frontend"),
		GrantedAt:  time.Now(),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	
	err = suite.storage.Create(ctx, "user_roles", newUserRole)
	require.NoError(suite.T(), err)
	
	// 캐시 무효화
	err = suite.cache.InvalidateUser("user-alice")
	assert.NoError(suite.T(), err)
	
	// 이제 쓰기 권한이 있어야 함
	resp, err = suite.rbacManager.CheckPermission(ctx, req)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), resp.Allowed)
}

// 테스트 스위트 실행
func TestRBACScenarioSuite(t *testing.T) {
	suite.Run(t, new(RBACScenarioTestSuite))
}

// 헬퍼 함수들

// stringPtr 문자열 포인터 생성
func stringPtr(s string) *string {
	return &s
}

// BenchmarkRBACPermissionCheck RBAC 권한 검사 벤치마크
func BenchmarkRBACPermissionCheck(b *testing.B) {
	// 간단한 벤치마크 설정
	storage := memory.NewMemoryStorage()
	cache := auth.NewMemoryPermissionCache()
	rbacStorage := auth.NewRBACStorageAdapter(storage)
	rbacManager := auth.NewRBACManager(rbacStorage, cache)
	
	ctx := context.Background()
	
	// 기본 테스트 데이터 설정 (간단화)
	role := &models.Role{
		Base:     models.Base{ID: "role-test"},
		Name:     "Test Role",
		Level:    1,
		IsActive: true,
	}
	storage.Create(ctx, "roles", role)
	
	permission := &models.Permission{
		Base:         models.Base{ID: "perm-test"},
		Name:         "Test Permission",
		ResourceType: models.ResourceTypeWorkspace,
		Action:       models.ActionRead,
		Effect:       models.PermissionAllow,
		IsActive:     true,
	}
	storage.Create(ctx, "permissions", permission)
	
	rolePermission := &models.RolePermission{
		Base:         models.Base{ID: "rp-test"},
		RoleID:       role.ID,
		PermissionID: permission.ID,
	}
	storage.Create(ctx, "role_permissions", rolePermission)
	
	userRole := &models.UserRole{
		Base:   models.Base{ID: "ur-test"},
		UserID: "user-test",
		RoleID: role.ID,
	}
	storage.Create(ctx, "user_roles", userRole)
	
	req := &models.CheckPermissionRequest{
		UserID:       "user-test",
		ResourceType: models.ResourceTypeWorkspace,
		ResourceID:   "workspace-test",
		Action:       models.ActionRead,
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := rbacManager.CheckPermission(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}