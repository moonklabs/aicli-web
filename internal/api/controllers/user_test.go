package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/aicli/aicli-web/internal/auth"
	"github.com/aicli/aicli-web/internal/models"
)

// MockUserService는 테스트용 UserService 모의 객체입니다
type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) GetProfile(ctx context.Context, userID string) (*models.UserResponse, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(*models.UserResponse), args.Error(1)
}

func (m *MockUserService) UpdateProfile(ctx context.Context, userID string, req *models.UpdateProfileRequest) (*models.UserResponse, error) {
	args := m.Called(ctx, userID, req)
	return args.Get(0).(*models.UserResponse), args.Error(1)
}

func (m *MockUserService) ChangePassword(ctx context.Context, userID string, req *models.ChangePasswordRequest) error {
	args := m.Called(ctx, userID, req)
	return args.Error(0)
}

func (m *MockUserService) ChangeEmail(ctx context.Context, userID string, req *models.ChangeEmailRequest) error {
	args := m.Called(ctx, userID, req)
	return args.Error(0)
}

func (m *MockUserService) Enable2FA(ctx context.Context, userID string, req *models.Enable2FARequest) (*models.TwoFactorSecret, error) {
	args := m.Called(ctx, userID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.TwoFactorSecret), args.Error(1)
}

func (m *MockUserService) Disable2FA(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockUserService) Verify2FA(ctx context.Context, userID string, req *models.Verify2FARequest) error {
	args := m.Called(ctx, userID, req)
	return args.Error(0)
}

func (m *MockUserService) Generate2FASecret(ctx context.Context, userID string) (*models.TwoFactorSecret, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(*models.TwoFactorSecret), args.Error(1)
}

func (m *MockUserService) RequestPasswordReset(ctx context.Context, req *models.ResetPasswordRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *MockUserService) ConfirmPasswordReset(ctx context.Context, req *models.ConfirmResetPasswordRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *MockUserService) LogActivity(ctx context.Context, userID, action, resource, details, ipAddress, userAgent string) error {
	args := m.Called(ctx, userID, action, resource, details, ipAddress, userAgent)
	return args.Error(0)
}

func (m *MockUserService) GetUserActivities(ctx context.Context, userID string, pagination *models.PaginationRequest) (*models.PaginatedResponse, error) {
	args := m.Called(ctx, userID, pagination)
	return args.Get(0).(*models.PaginatedResponse), args.Error(1)
}

func (m *MockUserService) GetUserStats(ctx context.Context) (*models.UserStats, error) {
	args := m.Called(ctx)
	return args.Get(0).(*models.UserStats), args.Error(1)
}

func (m *MockUserService) ListUsers(ctx context.Context, filter *models.UserFilter) (*models.PaginatedResponse, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(*models.PaginatedResponse), args.Error(1)
}

func (m *MockUserService) CreateUser(ctx context.Context, req *models.CreateUserRequest) (*models.UserResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*models.UserResponse), args.Error(1)
}

func (m *MockUserService) GetUser(ctx context.Context, userID string) (*models.UserResponse, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(*models.UserResponse), args.Error(1)
}

func (m *MockUserService) UpdateUser(ctx context.Context, userID string, req *models.UpdateUserRequest) (*models.UserResponse, error) {
	args := m.Called(ctx, userID, req)
	return args.Get(0).(*models.UserResponse), args.Error(1)
}

func (m *MockUserService) DeleteUser(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

// 테스트 설정 함수들
func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	return router
}

func setupUserController() (*UserController, *MockUserService) {
	mockService := new(MockUserService)
	controller := NewUserController(mockService)
	return controller, mockService
}

func addAuthContext(c *gin.Context, userID string) {
	claims := &auth.Claims{
		UserID: userID,
	}
	c.Set("claims", claims)
}

// 테스트 케이스들
func TestGetProfile(t *testing.T) {
	router := setupTestRouter()
	controller, mockService := setupUserController()

	// 테스트 데이터
	expectedProfile := &models.UserResponse{
		ID:          "user123",
		Username:    "testuser",
		Email:       "test@example.com",
		DisplayName: "Test User",
	}

	// Mock 설정
	mockService.On("GetProfile", mock.Anything, "user123").Return(expectedProfile, nil)

	// 라우트 설정
	router.GET("/users/me", func(c *gin.Context) {
		addAuthContext(c, "user123")
		controller.GetProfile(c)
	})

	// 요청 실행
	req, _ := http.NewRequest("GET", "/users/me", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 결과 검증
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response["success"].(bool))
	
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "user123", data["id"])
	assert.Equal(t, "testuser", data["username"])

	mockService.AssertExpectations(t)
}

func TestUpdateProfile(t *testing.T) {
	router := setupTestRouter()
	controller, mockService := setupUserController()

	// 테스트 데이터
	updateReq := &models.UpdateProfileRequest{
		DisplayName: stringPtr("Updated Name"),
		Bio:         stringPtr("Updated bio"),
	}

	expectedProfile := &models.UserResponse{
		ID:          "user123",
		Username:    "testuser",
		DisplayName: "Updated Name",
		Bio:         stringPtr("Updated bio"),
	}

	// Mock 설정
	mockService.On("UpdateProfile", mock.Anything, "user123", mock.MatchedBy(func(req *models.UpdateProfileRequest) bool {
		return req.DisplayName != nil && *req.DisplayName == "Updated Name"
	})).Return(expectedProfile, nil)

	mockService.On("LogActivity", mock.Anything, "user123", "profile_update", "user", "프로파일 업데이트", mock.Anything, mock.Anything).Return(nil)

	// 라우트 설정
	router.PUT("/users/me", func(c *gin.Context) {
		addAuthContext(c, "user123")
		controller.UpdateProfile(c)
	})

	// 요청 데이터 준비
	reqBody, _ := json.Marshal(updateReq)
	
	// 요청 실행
	req, _ := http.NewRequest("PUT", "/users/me", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 결과 검증
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response["success"].(bool))
	assert.Equal(t, "프로파일이 성공적으로 업데이트되었습니다", response["message"])

	mockService.AssertExpectations(t)
}

func TestChangePassword(t *testing.T) {
	router := setupTestRouter()
	controller, mockService := setupUserController()

	// 테스트 데이터
	changePasswordReq := &models.ChangePasswordRequest{
		CurrentPassword: "oldpassword",
		NewPassword:     "newpassword123",
		ConfirmPassword: "newpassword123",
	}

	// Mock 설정
	mockService.On("ChangePassword", mock.Anything, "user123", mock.MatchedBy(func(req *models.ChangePasswordRequest) bool {
		return req.CurrentPassword == "oldpassword" && req.NewPassword == "newpassword123"
	})).Return(nil)

	mockService.On("LogActivity", mock.Anything, "user123", "password_change", "user", "비밀번호 변경", mock.Anything, mock.Anything).Return(nil)

	// 라우트 설정
	router.PUT("/users/me/password", func(c *gin.Context) {
		addAuthContext(c, "user123")
		controller.ChangePassword(c)
	})

	// 요청 데이터 준비
	reqBody, _ := json.Marshal(changePasswordReq)
	
	// 요청 실행
	req, _ := http.NewRequest("PUT", "/users/me/password", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 결과 검증
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response["success"].(bool))
	assert.Equal(t, "비밀번호가 성공적으로 변경되었습니다", response["message"])

	mockService.AssertExpectations(t)
}

func TestGenerate2FASecret(t *testing.T) {
	router := setupTestRouter()
	controller, mockService := setupUserController()

	// 테스트 데이터
	expectedSecret := &models.TwoFactorSecret{
		Base: models.Base{ID: "secret123"},
		UserID:      "user123",
		Secret:      "ABCDEFGHIJKLMNOP",
		BackupCodes: []string{"12345678", "87654321"},
		IsActive:    false,
	}

	// Mock 설정
	mockService.On("Generate2FASecret", mock.Anything, "user123").Return(expectedSecret, nil)

	// 라우트 설정
	router.POST("/users/me/2fa/generate", func(c *gin.Context) {
		addAuthContext(c, "user123")
		controller.Generate2FASecret(c)
	})

	// 요청 실행
	req, _ := http.NewRequest("POST", "/users/me/2fa/generate", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 결과 검증
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response["success"].(bool))
	assert.Equal(t, "2FA 비밀키가 생성되었습니다", response["message"])

	data := response["data"].(map[string]interface{})
	assert.Equal(t, "secret123", data["id"])
	assert.Equal(t, "user123", data["user_id"])

	mockService.AssertExpectations(t)
}

func TestListUsers_Admin(t *testing.T) {
	router := setupTestRouter()
	controller, mockService := setupUserController()

	// 테스트 데이터
	expectedUsers := &models.PaginatedResponse{
		Data: []models.UserResponse{
			{ID: "user1", Username: "user1", Email: "user1@example.com"},
			{ID: "user2", Username: "user2", Email: "user2@example.com"},
		},
		Total:      2,
		Page:       1,
		Limit:      10,
		TotalPages: 1,
	}

	// Mock 설정
	mockService.On("ListUsers", mock.Anything, mock.MatchedBy(func(filter *models.UserFilter) bool {
		return filter.Page == 1 && filter.Limit == 10
	})).Return(expectedUsers, nil)

	// 라우트 설정
	router.GET("/admin/users", controller.ListUsers)

	// 요청 실행
	req, _ := http.NewRequest("GET", "/admin/users?page=1&limit=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 결과 검증
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response["success"].(bool))

	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(2), data["total"])
	assert.Equal(t, float64(1), data["page"])

	mockService.AssertExpectations(t)
}

func TestRequestPasswordReset(t *testing.T) {
	router := setupTestRouter()
	controller, mockService := setupUserController()

	// 테스트 데이터
	resetReq := &models.ResetPasswordRequest{
		Email: "test@example.com",
	}

	// Mock 설정
	mockService.On("RequestPasswordReset", mock.Anything, mock.MatchedBy(func(req *models.ResetPasswordRequest) bool {
		return req.Email == "test@example.com"
	})).Return(nil)

	// 라우트 설정
	router.POST("/auth/password-reset", controller.RequestPasswordReset)

	// 요청 데이터 준비
	reqBody, _ := json.Marshal(resetReq)
	
	// 요청 실행
	req, _ := http.NewRequest("POST", "/auth/password-reset", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 결과 검증
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response["success"].(bool))
	assert.Equal(t, "비밀번호 재설정 이메일이 발송되었습니다", response["message"])

	mockService.AssertExpectations(t)
}

// 헬퍼 함수들
func stringPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}