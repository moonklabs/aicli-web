package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aicli/aicli-web/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockRBACManager RBAC 매니저 모킹
type MockRBACManager struct {
	mock.Mock
}

func (m *MockRBACManager) CheckPermission(ctx context.Context, req *models.CheckPermissionRequest) (*models.CheckPermissionResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*models.CheckPermissionResponse), args.Error(1)
}

func (m *MockRBACManager) ComputeUserPermissionMatrix(ctx context.Context, userID string) (*models.UserPermissionMatrix, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(*models.UserPermissionMatrix), args.Error(1)
}

func (m *MockRBACManager) InvalidateUserPermissions(userID string) error {
	args := m.Called(userID)
	return args.Error(0)
}

func (m *MockRBACManager) InvalidateRolePermissions(roleID string) error {
	args := m.Called(roleID)
	return args.Error(0)
}

func (m *MockRBACManager) InvalidateGroupPermissions(groupID string) error {
	args := m.Called(groupID)
	return args.Error(0)
}

func TestRequirePermission_Success(t *testing.T) {
	// Mock RBAC Manager
	mockRBACManager := &MockRBACManager{}
	
	// 권한 허용 응답 설정
	mockResponse := &models.CheckPermissionResponse{
		Allowed: true,
		Decision: models.PermissionDecision{
			ResourceType: models.ResourceTypeWorkspace,
			ResourceID:   "test-workspace",
			Action:       models.ActionCreate,
			Effect:       models.PermissionAllow,
			Source:       "role:admin",
			Reason:       "Admin role allows all actions",
		},
		Evaluation: []string{"User has admin role", "Admin role grants workspace:create permission"},
	}
	
	mockRBACManager.On("CheckPermission", mock.Anything, mock.AnythingOfType("*models.CheckPermissionRequest")).Return(mockResponse, nil)

	// Gin 설정
	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	// 미들웨어 적용
	router.Use(func(c *gin.Context) {
		// 테스트용 사용자 ID 설정
		c.Set("user_id", "test-user-123")
		c.Next()
	})
	
	router.POST("/test/:id", RequirePermission(mockRBACManager, models.ResourceTypeWorkspace, models.ActionCreate), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// 테스트 요청
	req, _ := http.NewRequest("POST", "/test/test-workspace", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 검증
	assert.Equal(t, http.StatusOK, w.Code)
	mockRBACManager.AssertExpectations(t)
}

func TestRequirePermission_Denied(t *testing.T) {
	// Mock RBAC Manager
	mockRBACManager := &MockRBACManager{}
	
	// 권한 거부 응답 설정
	mockResponse := &models.CheckPermissionResponse{
		Allowed: false,
		Decision: models.PermissionDecision{
			ResourceType: models.ResourceTypeWorkspace,
			ResourceID:   "test-workspace",
			Action:       models.ActionCreate,
			Effect:       models.PermissionDeny,
			Source:       "default",
			Reason:       "No explicit permission granted",
		},
		Evaluation: []string{"No matching permissions found"},
	}
	
	mockRBACManager.On("CheckPermission", mock.Anything, mock.AnythingOfType("*models.CheckPermissionRequest")).Return(mockResponse, nil)

	// Gin 설정
	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	// 미들웨어 적용
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "test-user-123")
		c.Next()
	})
	
	router.POST("/test/:id", RequirePermission(mockRBACManager, models.ResourceTypeWorkspace, models.ActionCreate), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// 테스트 요청
	req, _ := http.NewRequest("POST", "/test/test-workspace", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 검증
	assert.Equal(t, http.StatusForbidden, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.False(t, response["success"].(bool))
	
	errorObj := response["error"].(map[string]interface{})
	assert.Equal(t, "INSUFFICIENT_PERMISSIONS", errorObj["code"])
	
	mockRBACManager.AssertExpectations(t)
}

func TestRequirePermission_NoAuth(t *testing.T) {
	// Mock RBAC Manager
	mockRBACManager := &MockRBACManager{}

	// Gin 설정
	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	router.POST("/test/:id", RequirePermission(mockRBACManager, models.ResourceTypeWorkspace, models.ActionCreate), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// 테스트 요청 (인증 정보 없음)
	req, _ := http.NewRequest("POST", "/test/test-workspace", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 검증
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.False(t, response["success"].(bool))
	
	errorObj := response["error"].(map[string]interface{})
	assert.Equal(t, "AUTHENTICATION_REQUIRED", errorObj["code"])
}

// 벤치마크 테스트
func BenchmarkRequirePermission(b *testing.B) {
	// Mock RBAC Manager
	mockRBACManager := &MockRBACManager{}
	
	mockResponse := &models.CheckPermissionResponse{
		Allowed: true,
		Decision: models.PermissionDecision{
			Effect: models.PermissionAllow,
		},
	}
	
	mockRBACManager.On("CheckPermission", mock.Anything, mock.Anything).Return(mockResponse, nil)

	// Gin 설정
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "test-user-123")
		c.Next()
	})
	
	router.GET("/test", RequirePermission(mockRBACManager, models.ResourceTypeWorkspace, models.ActionRead), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// 벤치마크 실행
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}