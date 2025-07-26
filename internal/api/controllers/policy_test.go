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

	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/security"
)

// MockPolicyService는 테스트용 PolicyService 모의 객체입니다
type MockPolicyService struct {
	mock.Mock
}

func (m *MockPolicyService) CreatePolicy(ctx context.Context, req *security.CreatePolicyRequest) (*security.SecurityPolicy, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*security.SecurityPolicy), args.Error(1)
}

func (m *MockPolicyService) GetPolicy(ctx context.Context, id string) (*security.SecurityPolicy, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*security.SecurityPolicy), args.Error(1)
}

func (m *MockPolicyService) UpdatePolicy(ctx context.Context, id string, req *security.UpdatePolicyRequest) (*security.SecurityPolicy, error) {
	args := m.Called(ctx, id, req)
	return args.Get(0).(*security.SecurityPolicy), args.Error(1)
}

func (m *MockPolicyService) DeletePolicy(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockPolicyService) ListPolicies(ctx context.Context, filter *security.PolicyFilter) (*models.PaginatedResponse[*security.SecurityPolicy], error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(*models.PaginatedResponse[*security.SecurityPolicy]), args.Error(1)
}

func (m *MockPolicyService) ApplyPolicy(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockPolicyService) DeactivatePolicy(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockPolicyService) RollbackPolicy(ctx context.Context, id string, toVersion string) error {
	args := m.Called(ctx, id, toVersion)
	return args.Error(0)
}

func (m *MockPolicyService) GetActivePolicies(ctx context.Context, category string) ([]*security.SecurityPolicy, error) {
	args := m.Called(ctx, category)
	return args.Get(0).([]*security.SecurityPolicy), args.Error(1)
}

func (m *MockPolicyService) CreateTemplate(ctx context.Context, req *security.CreateTemplateRequest) (*security.PolicyTemplate, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*security.PolicyTemplate), args.Error(1)
}

func (m *MockPolicyService) GetTemplate(ctx context.Context, id string) (*security.PolicyTemplate, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*security.PolicyTemplate), args.Error(1)
}

func (m *MockPolicyService) ListTemplates(ctx context.Context) ([]*security.PolicyTemplate, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*security.PolicyTemplate), args.Error(1)
}

func (m *MockPolicyService) CreatePolicyFromTemplate(ctx context.Context, templateID string, req *security.CreateFromTemplateRequest) (*security.SecurityPolicy, error) {
	args := m.Called(ctx, templateID, req)
	return args.Get(0).(*security.SecurityPolicy), args.Error(1)
}

func (m *MockPolicyService) ValidatePolicy(ctx context.Context, policy *security.SecurityPolicy) (*security.ValidationResult, error) {
	args := m.Called(ctx, policy)
	return args.Get(0).(*security.ValidationResult), args.Error(1)
}

func (m *MockPolicyService) TestPolicy(ctx context.Context, id string, testData interface{}) (*security.TestResult, error) {
	args := m.Called(ctx, id, testData)
	return args.Get(0).(*security.TestResult), args.Error(1)
}

func (m *MockPolicyService) GetPolicyHistory(ctx context.Context, id string) ([]*security.PolicyAuditEntry, error) {
	args := m.Called(ctx, id)
	return args.Get(0).([]*security.PolicyAuditEntry), args.Error(1)
}

func (m *MockPolicyService) GetPolicyAuditLog(ctx context.Context, filter *security.AuditFilter) (*models.PaginatedResponse[*security.PolicyAuditEntry], error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(*models.PaginatedResponse[*security.PolicyAuditEntry]), args.Error(1)
}

// 테스트 설정 함수들
func setupPolicyController() (*PolicyController, *MockPolicyService) {
	mockService := new(MockPolicyService)
	controller := NewPolicyController(mockService)
	return controller, mockService
}

// 테스트 케이스들
func TestCreatePolicy(t *testing.T) {
	router := setupTestRouter()
	controller, mockService := setupPolicyController()

	// 테스트 데이터
	createReq := &security.CreatePolicyRequest{
		Name:        "Test Rate Limiting Policy",
		Description: "Test policy for rate limiting",
		Category:    "rate_limiting",
		ConfigData: map[string]interface{}{
			"requests_per_second": 100,
			"burst_size":          10,
		},
		Priority: 5,
	}

	expectedPolicy := &security.SecurityPolicy{
		Base: models.Base{ID: "pol_123"},
		Name:        "Test Rate Limiting Policy",
		Description: "Test policy for rate limiting",
		Category:    "rate_limiting",
		Priority:    5,
		Version:     "1.0.0",
		IsActive:    false,
	}

	// Mock 설정
	mockService.On("CreatePolicy", mock.Anything, mock.MatchedBy(func(req *security.CreatePolicyRequest) bool {
		return req.Name == "Test Rate Limiting Policy" && req.Category == "rate_limiting"
	})).Return(expectedPolicy, nil)

	// 라우트 설정
	router.POST("/admin/policies", func(c *gin.Context) {
		addAuthContext(c, "admin123")
		controller.CreatePolicy(c)
	})

	// 요청 데이터 준비
	reqBody, _ := json.Marshal(createReq)
	
	// 요청 실행
	req, _ := http.NewRequest("POST", "/admin/policies", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 결과 검증
	assert.Equal(t, http.StatusCreated, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response["success"].(bool))
	assert.Equal(t, "보안 정책이 성공적으로 생성되었습니다", response["message"])

	data := response["data"].(map[string]interface{})
	assert.Equal(t, "pol_123", data["id"])
	assert.Equal(t, "Test Rate Limiting Policy", data["name"])

	mockService.AssertExpectations(t)
}

func TestGetPolicy(t *testing.T) {
	router := setupTestRouter()
	controller, mockService := setupPolicyController()

	// 테스트 데이터
	expectedPolicy := &security.SecurityPolicy{
		Base: models.Base{ID: "pol_123"},
		Name:        "Test Policy",
		Description: "Test policy description",
		Category:    "rate_limiting",
		Priority:    5,
		Version:     "1.0.0",
		IsActive:    true,
	}

	// Mock 설정
	mockService.On("GetPolicy", mock.Anything, "pol_123").Return(expectedPolicy, nil)

	// 라우트 설정
	router.GET("/admin/policies/:id", controller.GetPolicy)

	// 요청 실행
	req, _ := http.NewRequest("GET", "/admin/policies/pol_123", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 결과 검증
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response["success"].(bool))

	data := response["data"].(map[string]interface{})
	assert.Equal(t, "pol_123", data["id"])
	assert.Equal(t, "Test Policy", data["name"])

	mockService.AssertExpectations(t)
}

func TestApplyPolicy(t *testing.T) {
	router := setupTestRouter()
	controller, mockService := setupPolicyController()

	// Mock 설정
	mockService.On("ApplyPolicy", mock.Anything, "pol_123").Return(nil)

	// 라우트 설정
	router.POST("/admin/policies/:id/apply", controller.ApplyPolicy)

	// 요청 실행
	req, _ := http.NewRequest("POST", "/admin/policies/pol_123/apply", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 결과 검증
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response["success"].(bool))
	assert.Equal(t, "정책이 성공적으로 적용되었습니다", response["message"])

	mockService.AssertExpectations(t)
}

func TestListPolicies(t *testing.T) {
	router := setupTestRouter()
	controller, mockService := setupPolicyController()

	// 테스트 데이터
	policy1 := &security.SecurityPolicy{
		Base:     models.Base{ID: "pol_1"},
		Name:     "Policy 1",
		Category: "rate_limiting",
	}
	policy2 := &security.SecurityPolicy{
		Base:     models.Base{ID: "pol_2"},
		Name:     "Policy 2", 
		Category: "authentication",
	}
	
	expectedPolicies := &models.PaginatedResponse[*security.SecurityPolicy]{
		Data: []*security.SecurityPolicy{policy1, policy2},
		Pagination: models.NewPaginationMeta(1, 10, 2),
	}

	// Mock 설정
	mockService.On("ListPolicies", mock.Anything, mock.MatchedBy(func(filter *security.PolicyFilter) bool {
		return filter.Page == 1 && filter.Limit == 10
	})).Return(expectedPolicies, nil)

	// 라우트 설정
	router.GET("/admin/policies", controller.ListPolicies)

	// 요청 실행
	req, _ := http.NewRequest("GET", "/admin/policies?page=1&limit=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 결과 검증
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response["success"].(bool))

	data := response["data"].(map[string]interface{})
	pagination := data["pagination"].(map[string]interface{})
	assert.Equal(t, float64(2), pagination["total"])

	mockService.AssertExpectations(t)
}

func TestCreateTemplate(t *testing.T) {
	router := setupTestRouter()
	controller, mockService := setupPolicyController()

	// 테스트 데이터
	createReq := &security.CreateTemplateRequest{
		Name:        "Test Template",
		Description: "Test template description",
		Category:    "rate_limiting",
		ConfigSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"requests_per_second": map[string]interface{}{
					"type": "number",
					"minimum": 1,
				},
			},
		},
	}

	expectedTemplate := &security.PolicyTemplate{
		Base: models.Base{ID: "tpl_123"},
		Name:        "Test Template",
		Description: "Test template description",
		Category:    "rate_limiting",
		IsBuiltIn:   false,
	}

	// Mock 설정
	mockService.On("CreateTemplate", mock.Anything, mock.MatchedBy(func(req *security.CreateTemplateRequest) bool {
		return req.Name == "Test Template"
	})).Return(expectedTemplate, nil)

	// 라우트 설정
	router.POST("/admin/policy-templates", controller.CreateTemplate)

	// 요청 데이터 준비
	reqBody, _ := json.Marshal(createReq)
	
	// 요청 실행
	req, _ := http.NewRequest("POST", "/admin/policy-templates", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 결과 검증
	assert.Equal(t, http.StatusCreated, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response["success"].(bool))
	assert.Equal(t, "정책 템플릿이 성공적으로 생성되었습니다", response["message"])

	mockService.AssertExpectations(t)
}

func TestGetActivePolicies(t *testing.T) {
	router := setupTestRouter()
	controller, mockService := setupPolicyController()

	// 테스트 데이터
	activePolicies := []*security.SecurityPolicy{
		{
			Base: models.Base{ID: "pol_1"},
			Name:     "Active Policy 1",
			Category: "rate_limiting",
			IsActive: true,
		},
		{
			Base: models.Base{ID: "pol_2"},
			Name:     "Active Policy 2",
			Category: "rate_limiting",
			IsActive: true,
		},
	}

	// Mock 설정
	mockService.On("GetActivePolicies", mock.Anything, "rate_limiting").Return(activePolicies, nil)

	// 라우트 설정
	router.GET("/admin/policies/active", controller.GetActivePolicies)

	// 요청 실행
	req, _ := http.NewRequest("GET", "/admin/policies/active?category=rate_limiting", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 결과 검증
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response["success"].(bool))

	data := response["data"].([]interface{})
	assert.Len(t, data, 2)
	
	policy1 := data[0].(map[string]interface{})
	assert.Equal(t, "pol_1", policy1["id"])
	assert.Equal(t, "Active Policy 1", policy1["name"])

	mockService.AssertExpectations(t)
}