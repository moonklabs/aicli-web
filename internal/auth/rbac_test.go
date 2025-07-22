package auth

import (
	"context"
	"testing"
	"time"
	
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/aicli/aicli-web/internal/models"
)

// MockRBACStorage RBAC 저장소 모킹
type MockRBACStorage struct {
	mock.Mock
}

func (m *MockRBACStorage) GetRoleByID(ctx context.Context, roleID string) (*models.Role, error) {
	args := m.Called(ctx, roleID)
	return args.Get(0).(*models.Role), args.Error(1)
}

func (m *MockRBACStorage) GetRolesByUserID(ctx context.Context, userID string) ([]models.Role, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]models.Role), args.Error(1)
}

func (m *MockRBACStorage) GetRolesByGroupID(ctx context.Context, groupID string) ([]models.Role, error) {
	args := m.Called(ctx, groupID)
	return args.Get(0).([]models.Role), args.Error(1)
}

func (m *MockRBACStorage) GetRoleHierarchy(ctx context.Context, roleID string) ([]models.Role, error) {
	args := m.Called(ctx, roleID)
	return args.Get(0).([]models.Role), args.Error(1)
}

func (m *MockRBACStorage) GetPermissionsByRoleID(ctx context.Context, roleID string) ([]models.Permission, error) {
	args := m.Called(ctx, roleID)
	return args.Get(0).([]models.Permission), args.Error(1)
}

func (m *MockRBACStorage) GetAllPermissions(ctx context.Context) ([]models.Permission, error) {
	args := m.Called(ctx)
	return args.Get(0).([]models.Permission), args.Error(1)
}

func (m *MockRBACStorage) GetUserRoles(ctx context.Context, userID string, resourceID *string) ([]models.UserRole, error) {
	args := m.Called(ctx, userID, resourceID)
	return args.Get(0).([]models.UserRole), args.Error(1)
}

func (m *MockRBACStorage) GetUserGroups(ctx context.Context, userID string) ([]models.UserGroup, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]models.UserGroup), args.Error(1)
}

func (m *MockRBACStorage) GetGroupRoles(ctx context.Context, groupID string, resourceID *string) ([]models.GroupRole, error) {
	args := m.Called(ctx, groupID, resourceID)
	return args.Get(0).([]models.GroupRole), args.Error(1)
}

func (m *MockRBACStorage) GetResourceByID(ctx context.Context, resourceID string) (*models.Resource, error) {
	args := m.Called(ctx, resourceID)
	return args.Get(0).(*models.Resource), args.Error(1)
}

func (m *MockRBACStorage) GetResourceHierarchy(ctx context.Context, resourceID string) ([]models.Resource, error) {
	args := m.Called(ctx, resourceID)
	return args.Get(0).([]models.Resource), args.Error(1)
}

// MockPermissionCache 권한 캐시 모킹
type MockPermissionCache struct {
	mock.Mock
}

func (m *MockPermissionCache) GetUserPermissionMatrix(userID string) (*models.UserPermissionMatrix, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserPermissionMatrix), args.Error(1)
}

func (m *MockPermissionCache) SetUserPermissionMatrix(userID string, matrix *models.UserPermissionMatrix, ttl time.Duration) error {
	args := m.Called(userID, matrix, ttl)
	return args.Error(0)
}

func (m *MockPermissionCache) InvalidateUser(userID string) error {
	args := m.Called(userID)
	return args.Error(0)
}

func (m *MockPermissionCache) InvalidateRole(roleID string) error {
	args := m.Called(roleID)
	return args.Error(0)
}

func (m *MockPermissionCache) InvalidateGroup(groupID string) error {
	args := m.Called(groupID)
	return args.Error(0)
}

// 테스트 유틸리티 함수들

// createTestRole 테스트용 역할 생성
func createTestRole(id, name, description string, level int, parentID *string) *models.Role {
	return &models.Role{
		Base: models.Base{
			ID:        id,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Name:        name,
		Description: description,
		Level:       level,
		ParentID:    parentID,
		IsSystem:    false,
		IsActive:    true,
	}
}

// createTestPermission 테스트용 권한 생성
func createTestPermission(id, name string, resourceType models.ResourceType, action models.ActionType, effect models.PermissionEffect) *models.Permission {
	return &models.Permission{
		Base: models.Base{
			ID:        id,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Name:         name,
		ResourceType: resourceType,
		Action:       action,
		Effect:       effect,
		IsActive:     true,
	}
}

// createTestUserGroup 테스트용 사용자 그룹 생성
func createTestUserGroup(id, name, description, groupType string, parentID *string) *models.UserGroup {
	return &models.UserGroup{
		Base: models.Base{
			ID:        id,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Name:        name,
		Description: description,
		Type:        groupType,
		ParentID:    parentID,
		IsActive:    true,
	}
}

// 실제 테스트 케이스들

func TestRBACManager_CheckPermission_Allow(t *testing.T) {
	// Given
	ctx := context.Background()
	mockStorage := new(MockRBACStorage)
	mockCache := new(MockPermissionCache)
	rbacManager := NewRBACManager(mockStorage, mockCache)
	
	userID := "user-123"
	resourceType := models.ResourceTypeWorkspace
	resourceID := "workspace-456"
	action := models.ActionRead
	
	// Mock 설정: 캐시 미스
	mockCache.On("GetUserPermissionMatrix", userID).Return(nil, nil)
	
	// Mock 설정: 사용자 역할 조회
	testRole := createTestRole("role-user", "user", "일반 사용자", 2, nil)
	mockStorage.On("GetRolesByUserID", ctx, userID).Return([]models.Role{*testRole}, nil)
	
	// Mock 설정: 사용자 그룹 조회
	mockStorage.On("GetUserGroups", ctx, userID).Return([]models.UserGroup{}, nil)
	
	// Mock 설정: 역할 권한 조회
	testPermission := createTestPermission("perm-workspace-read", "workspace_read", models.ResourceTypeWorkspace, models.ActionRead, models.PermissionAllow)
	mockStorage.On("GetPermissionsByRoleID", ctx, testRole.ID).Return([]models.Permission{*testPermission}, nil)
	
	// Mock 설정: 역할 상속 조회
	mockStorage.On("GetRoleHierarchy", ctx, testRole.ID).Return([]models.Role{}, nil)
	
	// Mock 설정: 캐시 저장
	mockCache.On("SetUserPermissionMatrix", userID, mock.AnythingOfType("*models.UserPermissionMatrix"), 30*time.Minute).Return(nil)
	
	req := &models.CheckPermissionRequest{
		UserID:       userID,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Action:       action,
	}
	
	// When
	resp, err := rbacManager.CheckPermission(ctx, req)
	
	// Then
	assert.NoError(t, err)
	assert.True(t, resp.Allowed)
	assert.Equal(t, models.PermissionAllow, resp.Decision.Effect)
	assert.Contains(t, resp.Decision.Source, "role:user")
	
	mockStorage.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestRBACManager_CheckPermission_Deny(t *testing.T) {
	// Given
	ctx := context.Background()
	mockStorage := new(MockRBACStorage)
	mockCache := new(MockPermissionCache)
	rbacManager := NewRBACManager(mockStorage, mockCache)
	
	userID := "user-123"
	resourceType := models.ResourceTypeWorkspace
	resourceID := "workspace-456"
	action := models.ActionDelete
	
	// Mock 설정: 캐시 미스
	mockCache.On("GetUserPermissionMatrix", userID).Return(nil, nil)
	
	// Mock 설정: 사용자에게 역할이 없음
	mockStorage.On("GetRolesByUserID", ctx, userID).Return([]models.Role{}, nil)
	mockStorage.On("GetUserGroups", ctx, userID).Return([]models.UserGroup{}, nil)
	
	req := &models.CheckPermissionRequest{
		UserID:       userID,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Action:       action,
	}
	
	// When
	resp, err := rbacManager.CheckPermission(ctx, req)
	
	// Then
	assert.NoError(t, err)
	assert.False(t, resp.Allowed)
	assert.Equal(t, models.PermissionDeny, resp.Decision.Effect)
	assert.Equal(t, "default", resp.Decision.Source)
	
	mockStorage.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestRBACManager_CheckPermission_CacheHit(t *testing.T) {
	// Given
	ctx := context.Background()
	mockStorage := new(MockRBACStorage)
	mockCache := new(MockPermissionCache)
	rbacManager := NewRBACManager(mockStorage, mockCache)
	
	userID := "user-123"
	resourceType := models.ResourceTypeWorkspace
	resourceID := "workspace-456"
	action := models.ActionRead
	
	// 캐시된 권한 매트릭스 생성
	matrix := &models.UserPermissionMatrix{
		UserID:         userID,
		DirectRoles:    []string{"role-user"},
		InheritedRoles: []string{},
		GroupRoles:     []string{},
		FinalPermissions: map[string]models.PermissionDecision{
			"workspace:*:read": {
				ResourceType: models.ResourceTypeWorkspace,
				ResourceID:   "*",
				Action:       models.ActionRead,
				Effect:       models.PermissionAllow,
				Source:       "role:user",
				Reason:       "역할 'user'을 통해 부여됨",
			},
		},
		ComputedAt: time.Now(),
	}
	
	// Mock 설정: 캐시 히트
	mockCache.On("GetUserPermissionMatrix", userID).Return(matrix, nil)
	
	req := &models.CheckPermissionRequest{
		UserID:       userID,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Action:       action,
	}
	
	// When
	resp, err := rbacManager.CheckPermission(ctx, req)
	
	// Then
	assert.NoError(t, err)
	assert.True(t, resp.Allowed)
	assert.Equal(t, models.PermissionAllow, resp.Decision.Effect)
	
	// 스토리지 호출이 없어야 함 (캐시 히트)
	mockStorage.AssertNotCalled(t, "GetRolesByUserID")
	mockCache.AssertExpectations(t)
}

func TestRBACManager_ComputeUserPermissionMatrix_WithGroupRoles(t *testing.T) {
	// Given
	ctx := context.Background()
	mockStorage := new(MockRBACStorage)
	mockCache := new(MockPermissionCache)
	rbacManager := NewRBACManager(mockStorage, mockCache)
	
	userID := "user-123"
	
	// 직접 역할
	directRole := createTestRole("role-user", "user", "일반 사용자", 2, nil)
	mockStorage.On("GetRolesByUserID", ctx, userID).Return([]models.Role{*directRole}, nil)
	
	// 그룹을 통한 역할
	testGroup := createTestUserGroup("group-dev", "developers", "개발자 그룹", "team", nil)
	groupRole := createTestRole("role-developer", "developer", "개발자", 3, nil)
	mockStorage.On("GetUserGroups", ctx, userID).Return([]models.UserGroup{*testGroup}, nil)
	mockStorage.On("GetRolesByGroupID", ctx, testGroup.ID).Return([]models.Role{*groupRole}, nil)
	mockStorage.On("GetRoleByID", ctx, groupRole.ID).Return(groupRole, nil)
	
	// 권한 조회
	userPermission := createTestPermission("perm-workspace-read", "workspace_read", models.ResourceTypeWorkspace, models.ActionRead, models.PermissionAllow)
	devPermission := createTestPermission("perm-project-create", "project_create", models.ResourceTypeProject, models.ActionCreate, models.PermissionAllow)
	
	mockStorage.On("GetPermissionsByRoleID", ctx, directRole.ID).Return([]models.Permission{*userPermission}, nil)
	mockStorage.On("GetPermissionsByRoleID", ctx, groupRole.ID).Return([]models.Permission{*devPermission}, nil)
	
	// 상속 역할 없음
	mockStorage.On("GetRoleHierarchy", ctx, directRole.ID).Return([]models.Role{}, nil)
	mockStorage.On("GetRoleHierarchy", ctx, groupRole.ID).Return([]models.Role{}, nil)
	
	// When
	matrix, err := rbacManager.ComputeUserPermissionMatrix(ctx, userID)
	
	// Then
	assert.NoError(t, err)
	assert.Equal(t, userID, matrix.UserID)
	assert.Contains(t, matrix.DirectRoles, directRole.ID)
	assert.Contains(t, matrix.GroupRoles, groupRole.ID)
	assert.Empty(t, matrix.InheritedRoles)
	
	// 권한이 올바르게 계산되었는지 확인
	workspaceReadKey := "workspace:*:read"
	projectCreateKey := "project:*:create"
	
	assert.Contains(t, matrix.FinalPermissions, workspaceReadKey)
	assert.Contains(t, matrix.FinalPermissions, projectCreateKey)
	
	assert.Equal(t, models.PermissionAllow, matrix.FinalPermissions[workspaceReadKey].Effect)
	assert.Equal(t, models.PermissionAllow, matrix.FinalPermissions[projectCreateKey].Effect)
	
	mockStorage.AssertExpectations(t)
}

func TestRBACManager_ComputeUserPermissionMatrix_WithRoleHierarchy(t *testing.T) {
	// Given
	ctx := context.Background()
	mockStorage := new(MockRBACStorage)
	mockCache := new(MockPermissionCache)
	rbacManager := NewRBACManager(mockStorage, mockCache)
	
	userID := "user-123"
	
	// 역할 계층: admin(1) -> user(2)
	adminRoleID := "role-admin"
	userRole := createTestRole("role-user", "user", "일반 사용자", 2, &adminRoleID)
	adminRole := createTestRole(adminRoleID, "admin", "관리자", 1, nil)
	
	mockStorage.On("GetRolesByUserID", ctx, userID).Return([]models.Role{*userRole}, nil)
	mockStorage.On("GetUserGroups", ctx, userID).Return([]models.UserGroup{}, nil)
	
	// 상속 역할 조회
	mockStorage.On("GetRoleHierarchy", ctx, userRole.ID).Return([]models.Role{*adminRole}, nil)
	
	// 권한 조회
	userPermission := createTestPermission("perm-workspace-read", "workspace_read", models.ResourceTypeWorkspace, models.ActionRead, models.PermissionAllow)
	adminPermission := createTestPermission("perm-user-manage", "user_manage", models.ResourceTypeUser, models.ActionManage, models.PermissionAllow)
	
	mockStorage.On("GetPermissionsByRoleID", ctx, userRole.ID).Return([]models.Permission{*userPermission}, nil)
	mockStorage.On("GetPermissionsByRoleID", ctx, adminRole.ID).Return([]models.Permission{*adminPermission}, nil)
	
	// When
	matrix, err := rbacManager.ComputeUserPermissionMatrix(ctx, userID)
	
	// Then
	assert.NoError(t, err)
	assert.Contains(t, matrix.DirectRoles, userRole.ID)
	assert.Contains(t, matrix.InheritedRoles, adminRole.ID)
	
	// 두 권한 모두 있어야 함
	assert.Contains(t, matrix.FinalPermissions, "workspace:*:read")
	assert.Contains(t, matrix.FinalPermissions, "user:*:manage")
	
	mockStorage.AssertExpectations(t)
}

func TestRBACManager_PermissionConflictResolution(t *testing.T) {
	// Given
	ctx := context.Background()
	mockStorage := new(MockRBACStorage)
	mockCache := new(MockPermissionCache)
	rbacManager := NewRBACManager(mockStorage, mockCache)
	
	userID := "user-123"
	
	// 두 개의 역할: 하나는 허용, 하나는 거부
	allowRole := createTestRole("role-allow", "allow", "허용 역할", 2, nil)
	denyRole := createTestRole("role-deny", "deny", "거부 역할", 2, nil)
	
	mockStorage.On("GetRolesByUserID", ctx, userID).Return([]models.Role{*allowRole, *denyRole}, nil)
	mockStorage.On("GetUserGroups", ctx, userID).Return([]models.UserGroup{}, nil)
	
	// 충돌하는 권한 설정
	allowPermission := createTestPermission("perm-workspace-delete-allow", "workspace_delete", models.ResourceTypeWorkspace, models.ActionDelete, models.PermissionAllow)
	denyPermission := createTestPermission("perm-workspace-delete-deny", "workspace_delete", models.ResourceTypeWorkspace, models.ActionDelete, models.PermissionDeny)
	
	mockStorage.On("GetPermissionsByRoleID", ctx, allowRole.ID).Return([]models.Permission{*allowPermission}, nil)
	mockStorage.On("GetPermissionsByRoleID", ctx, denyRole.ID).Return([]models.Permission{*denyPermission}, nil)
	
	mockStorage.On("GetRoleHierarchy", ctx, allowRole.ID).Return([]models.Role{}, nil)
	mockStorage.On("GetRoleHierarchy", ctx, denyRole.ID).Return([]models.Role{}, nil)
	
	// When
	matrix, err := rbacManager.ComputeUserPermissionMatrix(ctx, userID)
	
	// Then
	assert.NoError(t, err)
	
	deleteKey := "workspace:*:delete"
	assert.Contains(t, matrix.FinalPermissions, deleteKey)
	
	// 거부 권한이 우선해야 함
	decision := matrix.FinalPermissions[deleteKey]
	assert.Equal(t, models.PermissionDeny, decision.Effect)
	assert.Contains(t, decision.Reason, "거부 권한이 허용 권한을 오버라이드함")
	
	mockStorage.AssertExpectations(t)
}

func TestJSONConditionEvaluator_EvaluateConditions(t *testing.T) {
	evaluator := &JSONConditionEvaluator{}
	
	tests := []struct {
		name       string
		conditions string
		context    map[string]interface{}
		expected   bool
		shouldErr  bool
	}{
		{
			name:       "No conditions",
			conditions: "",
			context:    map[string]interface{}{},
			expected:   true,
			shouldErr:  false,
		},
		{
			name:       "Simple match",
			conditions: `{"department": "engineering"}`,
			context:    map[string]interface{}{"department": "engineering"},
			expected:   true,
			shouldErr:  false,
		},
		{
			name:       "Simple mismatch",
			conditions: `{"department": "engineering"}`,
			context:    map[string]interface{}{"department": "sales"},
			expected:   false,
			shouldErr:  false,
		},
		{
			name:       "Missing context",
			conditions: `{"department": "engineering"}`,
			context:    map[string]interface{}{},
			expected:   false,
			shouldErr:  false,
		},
		{
			name:       "Multiple conditions match",
			conditions: `{"department": "engineering", "level": "senior"}`,
			context:    map[string]interface{}{"department": "engineering", "level": "senior"},
			expected:   true,
			shouldErr:  false,
		},
		{
			name:       "Multiple conditions partial match",
			conditions: `{"department": "engineering", "level": "senior"}`,
			context:    map[string]interface{}{"department": "engineering", "level": "junior"},
			expected:   false,
			shouldErr:  false,
		},
		{
			name:       "Invalid JSON",
			conditions: `{"department": "engineering"`,
			context:    map[string]interface{}{},
			expected:   false,
			shouldErr:  true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evaluator.EvaluateConditions(tt.conditions, tt.context)
			
			if tt.shouldErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestRBACManager_removeDuplicates(t *testing.T) {
	rbacManager := &RBACManager{}
	
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "No duplicates",
			input:    []string{"a", "b", "c"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "With duplicates",
			input:    []string{"a", "b", "a", "c", "b"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "Empty slice",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "Single element",
			input:    []string{"a"},
			expected: []string{"a"},
		},
		{
			name:     "All duplicates",
			input:    []string{"a", "a", "a"},
			expected: []string{"a"},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rbacManager.removeDuplicates(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}