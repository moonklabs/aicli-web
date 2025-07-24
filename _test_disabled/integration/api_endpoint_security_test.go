package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/aicli/aicli-web/internal/api/controllers"
	"github.com/aicli/aicli-web/internal/auth"
	"github.com/aicli/aicli-web/internal/config"
	"github.com/aicli/aicli-web/internal/middleware"
	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/services"
	"github.com/aicli/aicli-web/internal/storage"
	"github.com/aicli/aicli-web/internal/storage/memory"
)

// APIEndpointSecurityTestSuite API 엔드포인트 보안 테스트 스위트
type APIEndpointSecurityTestSuite struct {
	suite.Suite
	app           *gin.Engine
	storage       storage.Storage
	jwtManager    auth.JWTManager
	rbacManager   auth.RBACManager
	userService   services.UserService
	projectService services.ProjectService
	
	// 테스트용 토큰들
	adminToken  string
	userToken   string
	limitedToken string
	
	// 테스트 데이터
	testUsers    map[string]*models.User
	testProjects map[string]*models.Project
}

// SetupSuite 테스트 스위트 초기화
func (suite *APIEndpointSecurityTestSuite) SetupSuite() {
	gin.SetMode(gin.TestMode)
	
	// 저장소 초기화
	suite.storage = memory.NewMemoryStorage()
	
	// JWT 매니저 초기화
	suite.jwtManager = auth.NewJWTManager("test-secret-key", 15*time.Minute, 24*time.Hour)
	
	// RBAC 매니저 초기화
	cache := auth.NewMemoryPermissionCache()
	rbacStorage := auth.NewRBACStorageAdapter(suite.storage)
	suite.rbacManager = auth.NewRBACManager(rbacStorage, cache)
	
	// 서비스 초기화
	suite.userService = services.NewUserService(suite.storage)
	suite.projectService = services.NewProjectService(suite.storage)
	
	// 테스트 데이터 설정
	suite.setupTestData()
	
	// 애플리케이션 설정
	suite.setupApplication()
}

// setupTestData 테스트 데이터 설정
func (suite *APIEndpointSecurityTestSuite) setupTestData() {
	ctx := context.Background()
	suite.testUsers = make(map[string]*models.User)
	suite.testProjects = make(map[string]*models.Project)
	
	// 테스트 사용자 생성
	users := []*models.User{
		{
			Base:     models.Base{ID: "user-admin"},
			Username: "admin",
			Email:    "admin@test.com",
			IsActive: true,
		},
		{
			Base:     models.Base{ID: "user-john"},
			Username: "john",
			Email:    "john@test.com",
			IsActive: true,
		},
		{
			Base:     models.Base{ID: "user-jane"},
			Username: "jane",
			Email:    "jane@test.com",
			IsActive: true,
		},
		{
			Base:     models.Base{ID: "user-inactive"},
			Username: "inactive",
			Email:    "inactive@test.com",
			IsActive: false,
		},
	}
	
	for _, user := range users {
		user.CreatedAt = time.Now()
		user.UpdatedAt = time.Now()
		err := suite.storage.Create(ctx, "users", user)
		require.NoError(suite.T(), err)
		suite.testUsers[user.ID] = user
	}
	
	// 테스트 프로젝트 생성
	projects := []*models.Project{
		{
			Base:        models.Base{ID: "project-public"},
			Name:        "Public Project",
			Description: "공개 프로젝트",
			OwnerID:     "user-admin",
			IsPublic:    true,
			IsActive:    true,
		},
		{
			Base:        models.Base{ID: "project-private"},
			Name:        "Private Project",
			Description: "비공개 프로젝트",
			OwnerID:     "user-john",
			IsPublic:    false,
			IsActive:    true,
		},
		{
			Base:        models.Base{ID: "project-sensitive"},
			Name:        "Sensitive Project",
			Description: "민감한 프로젝트",
			OwnerID:     "user-admin",
			IsPublic:    false,
			IsActive:    true,
		},
	}
	
	for _, project := range projects {
		project.CreatedAt = time.Now()
		project.UpdatedAt = time.Now()
		err := suite.storage.Create(ctx, "projects", project)
		require.NoError(suite.T(), err)
		suite.testProjects[project.ID] = project
	}
	
	// 역할 및 권한 설정 (간소화)
	suite.setupRolesAndPermissions(ctx)
	
	// 토큰 생성
	suite.generateTestTokens()
}

// setupRolesAndPermissions 역할 및 권한 설정
func (suite *APIEndpointSecurityTestSuite) setupRolesAndPermissions(ctx context.Context) {
	// 역할 생성
	roles := []*models.Role{
		{
			Base:        models.Base{ID: "role-admin"},
			Name:        "Administrator",
			Description: "시스템 관리자",
			Level:       1,
			IsActive:    true,
		},
		{
			Base:        models.Base{ID: "role-user"},
			Name:        "User",
			Description: "일반 사용자",
			Level:       2,
			IsActive:    true,
		},
	}
	
	for _, role := range roles {
		role.CreatedAt = time.Now()
		role.UpdatedAt = time.Now()
		suite.storage.Create(ctx, "roles", role)
	}
	
	// 사용자-역할 할당
	userRoles := []*models.UserRole{
		{
			Base:      models.Base{ID: "ur-admin"},
			UserID:    "user-admin",
			RoleID:    "role-admin",
			GrantedAt: time.Now(),
		},
		{
			Base:      models.Base{ID: "ur-john"},
			UserID:    "user-john",
			RoleID:    "role-user",
			GrantedAt: time.Now(),
		},
		{
			Base:      models.Base{ID: "ur-jane"},
			UserID:    "user-jane",
			RoleID:    "role-user",
			GrantedAt: time.Now(),
		},
	}
	
	for _, ur := range userRoles {
		ur.CreatedAt = time.Now()
		ur.UpdatedAt = time.Now()
		suite.storage.Create(ctx, "user_roles", ur)
	}
}

// generateTestTokens 테스트용 토큰 생성
func (suite *APIEndpointSecurityTestSuite) generateTestTokens() {
	// 관리자 토큰
	adminClaims := &auth.Claims{
		UserID:   "user-admin",
		Email:    "admin@test.com",
		Provider: "local",
	}
	adminTokens, _ := suite.jwtManager.GenerateTokens(adminClaims)
	suite.adminToken = adminTokens.AccessToken
	
	// 일반 사용자 토큰
	userClaims := &auth.Claims{
		UserID:   "user-john",
		Email:    "john@test.com", 
		Provider: "local",
	}
	userTokens, _ := suite.jwtManager.GenerateTokens(userClaims)
	suite.userToken = userTokens.AccessToken
	
	// 제한된 사용자 토큰
	limitedClaims := &auth.Claims{
		UserID:   "user-jane",
		Email:    "jane@test.com",
		Provider: "local",
	}
	limitedTokens, _ := suite.jwtManager.GenerateTokens(limitedClaims)
	suite.limitedToken = limitedTokens.AccessToken
}

// setupApplication 애플리케이션 설정
func (suite *APIEndpointSecurityTestSuite) setupApplication() {
	suite.app = gin.New()
	
	// 미들웨어 설정
	suite.app.Use(middleware.ErrorHandler())
	suite.app.Use(middleware.CORSMiddleware())
	
	// Rate Limiting
	rateLimitConfig := &config.RateLimitConfig{
		RequestsPerSecond: 20,
		BurstSize:         40,
		WindowSize:        time.Minute,
		Enabled:           true,
	}
	suite.app.Use(middleware.RateLimitMiddleware(rateLimitConfig))
	
	// 보안 헤더
	securityConfig := &config.SecurityConfig{
		EnableHSTS:                true,
		EnableCSP:                 true,
		EnableXFrameOptions:       true,
		EnableXContentTypeOptions: true,
		EnableReferrerPolicy:      true,
	}
	suite.app.Use(middleware.SecurityHeadersMiddleware(securityConfig))
	
	// API 라우트 설정
	suite.setupAPIRoutes()
}

// setupAPIRoutes API 라우트 설정
func (suite *APIEndpointSecurityTestSuite) setupAPIRoutes() {
	// 컨트롤러 초기화
	userController := controllers.NewUserController(suite.userService)
	projectController := controllers.NewProjectController(suite.projectService)
	
	api := suite.app.Group("/api/v1")
	
	// 공개 엔드포인트
	api.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	
	api.POST("/users/register", userController.Register)
	api.POST("/auth/login", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"token": "dummy-token"})
	})
	
	// 인증이 필요한 엔드포인트
	protected := api.Group("")
	protected.Use(middleware.AuthMiddleware(suite.jwtManager))
	{
		// 사용자 관련 API
		users := protected.Group("/users")
		{
			users.GET("/me", userController.GetCurrentUser)
			users.PUT("/me", userController.UpdateCurrentUser)
			users.DELETE("/me", userController.DeleteCurrentUser)
			users.GET("/:id", userController.GetUser)
			users.PUT("/:id", userController.UpdateUser)
			users.DELETE("/:id", userController.DeleteUser)
			users.GET("", userController.ListUsers)
		}
		
		// 프로젝트 관련 API  
		projects := protected.Group("/projects")
		{
			projects.POST("", projectController.CreateProject)
			projects.GET("/:id", projectController.GetProject)
			projects.PUT("/:id", projectController.UpdateProject)
			projects.DELETE("/:id", projectController.DeleteProject)
			projects.GET("", projectController.ListProjects)
			
			// 프로젝트 내 리소스
			projects.GET("/:id/workspaces", projectController.GetProjectWorkspaces)
			projects.POST("/:id/workspaces", projectController.CreateWorkspace)
		}
		
		// 관리자 전용 API
		admin := protected.Group("/admin")
		admin.Use(middleware.RBACMiddleware(suite.rbacManager, models.ResourceTypeSystem, "", models.ActionManage))
		{
			admin.GET("/users", userController.AdminListUsers)
			admin.PUT("/users/:id/status", userController.AdminUpdateUserStatus)
			admin.DELETE("/users/:id", userController.AdminDeleteUser)
			admin.GET("/audit-logs", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"logs": []string{"admin-only"}})
			})
		}
		
		// 파일 업로드 API
		protected.POST("/upload", func(c *gin.Context) {
			file, header, err := c.Request.FormFile("file")
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "파일 업로드 실패"})
				return
			}
			defer file.Close()
			
			c.JSON(http.StatusOK, gin.H{
				"filename": header.Filename,
				"size":     header.Size,
			})
		})
	}
}

// TestInputValidationSecurity 입력 검증 보안 테스트
func (suite *APIEndpointSecurityTestSuite) TestInputValidationSecurity() {
	t := suite.T()
	
	// 1. JSON 구문 오류 테스트
	malformedJSON := `{"name": "test", "email": invalid-json}`
	req := httptest.NewRequest("PUT", "/api/v1/users/me", strings.NewReader(malformedJSON))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+suite.userToken)
	
	w := httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusBadRequest, w.Code, "Malformed JSON should be rejected")
	
	// 2. 과도하게 큰 페이로드 테스트
	largePayload := strings.Repeat("A", 10*1024*1024) // 10MB
	req = httptest.NewRequest("PUT", "/api/v1/users/me", strings.NewReader(largePayload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+suite.userToken)
	
	w = httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	// 대용량 페이로드는 거부되어야 함
	assert.NotEqual(t, http.StatusOK, w.Code, "Large payload should be rejected")
	
	// 3. SQL Injection 시도 (쿼리 파라미터)
	sqlInjectionPayloads := []string{
		"'; DROP TABLE users; --",
		"1' OR '1'='1",
		"UNION SELECT * FROM passwords",
	}
	
	for _, payload := range sqlInjectionPayloads {
		req = httptest.NewRequest("GET", "/api/v1/users?search="+payload, nil)
		req.Header.Set("Authorization", "Bearer "+suite.userToken)
		
		w = httptest.NewRecorder()
		suite.app.ServeHTTP(w, req)
		
		// SQL Injection 공격은 차단되거나 안전하게 처리되어야 함
		t.Logf("SQL injection test with payload '%s': status %d", payload, w.Code)
	}
	
	// 4. XSS 시도 (JSON 데이터)
	xssPayloads := []string{
		"<script>alert('xss')</script>",
		"javascript:alert('xss')",
		"<img src=x onerror=alert('xss')>",
	}
	
	for _, payload := range xssPayloads {
		userData := map[string]string{
			"name":  payload,
			"email": "test@example.com",
		}
		userJSON, _ := json.Marshal(userData)
		
		req = httptest.NewRequest("PUT", "/api/v1/users/me", bytes.NewReader(userJSON))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+suite.userToken)
		
		w = httptest.NewRecorder()
		suite.app.ServeHTTP(w, req)
		
		// XSS 공격은 차단되거나 안전하게 이스케이프되어야 함
		t.Logf("XSS test with payload '%s': status %d", payload, w.Code)
		
		if w.Code == http.StatusOK {
			// 응답에 원본 스크립트가 그대로 포함되지 않았는지 확인
			assert.NotContains(t, w.Body.String(), "<script>")
		}
	}
	
	// 5. 파라미터 경계값 테스트
	boundaryTests := []struct {
		name   string
		params string
		valid  bool
	}{
		{"negative page", "page=-1&limit=10", false},
		{"zero page", "page=0&limit=10", false},
		{"huge page", "page=999999&limit=10", false},
		{"negative limit", "page=1&limit=-1", false},
		{"huge limit", "page=1&limit=10000", false},
		{"valid params", "page=1&limit=20", true},
	}
	
	for _, test := range boundaryTests {
		req = httptest.NewRequest("GET", "/api/v1/users?"+test.params, nil)
		req.Header.Set("Authorization", "Bearer "+suite.userToken)
		
		w = httptest.NewRecorder()
		suite.app.ServeHTTP(w, req)
		
		if test.valid {
			assert.Equal(t, http.StatusOK, w.Code, "Valid params should succeed: %s", test.name)
		} else {
			assert.NotEqual(t, http.StatusOK, w.Code, "Invalid params should fail: %s", test.name)
		}
	}
}

// TestDataLeakagePrevention 데이터 누출 방지 테스트
func (suite *APIEndpointSecurityTestSuite) TestDataLeakagePrevention() {
	t := suite.T()
	
	// 1. 다른 사용자의 정보 접근 시도
	req := httptest.NewRequest("GET", "/api/v1/users/user-admin", nil)
	req.Header.Set("Authorization", "Bearer "+suite.userToken) // john이 admin 정보 조회 시도
	
	w := httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	if w.Code == http.StatusOK {
		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		
		// 민감한 정보가 포함되지 않았는지 확인
		responseStr := w.Body.String()
		assert.NotContains(t, responseStr, "password")
		assert.NotContains(t, responseStr, "token")
		assert.NotContains(t, responseStr, "secret")
		assert.NotContains(t, responseStr, "private_key")
	}
	
	// 2. 존재하지 않는 리소스 접근
	req = httptest.NewRequest("GET", "/api/v1/users/nonexistent", nil)
	req.Header.Set("Authorization", "Bearer "+suite.userToken)
	
	w = httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusNotFound, w.Code, "Nonexistent resource should return 404")
	
	// 에러 메시지에서 시스템 정보 노출 확인
	responseStr := w.Body.String()
	assert.NotContains(t, responseStr, "database")
	assert.NotContains(t, responseStr, "sql")
	assert.NotContains(t, responseStr, "panic")
	assert.NotContains(t, responseStr, "/internal/")
	
	// 3. 비공개 프로젝트 접근 테스트
	req = httptest.NewRequest("GET", "/api/v1/projects/project-private", nil)
	req.Header.Set("Authorization", "Bearer "+suite.limitedToken) // jane이 john의 프로젝트 접근
	
	w = httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	// 권한이 없으면 403 또는 404를 반환해야 함
	assert.True(t, w.Code == http.StatusForbidden || w.Code == http.StatusNotFound,
		"Private project should not be accessible")
	
	// 4. 관리자 전용 정보 접근 시도
	req = httptest.NewRequest("GET", "/api/v1/admin/audit-logs", nil)
	req.Header.Set("Authorization", "Bearer "+suite.userToken) // 일반 사용자가 관리자 API 접근
	
	w = httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusForbidden, w.Code, "Admin API should be forbidden for normal users")
	
	// 5. 비활성 사용자의 토큰 확인
	inactiveClaims := &auth.Claims{
		UserID:   "user-inactive",
		Email:    "inactive@test.com",
		Provider: "local",
	}
	inactiveTokens, _ := suite.jwtManager.GenerateTokens(inactiveClaims)
	
	req = httptest.NewRequest("GET", "/api/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+inactiveTokens.AccessToken)
	
	w = httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	// 비활성 사용자는 접근이 거부되어야 함
	assert.NotEqual(t, http.StatusOK, w.Code, "Inactive user should not have access")
}

// TestAPIRateLimiting API Rate Limiting 테스트
func (suite *APIEndpointSecurityTestSuite) TestAPIRateLimiting() {
	t := suite.T()
	
	// 인증된 사용자의 rate limiting 테스트
	var successCount, limitedCount int
	
	for i := 0; i < 50; i++ {
		req := httptest.NewRequest("GET", "/api/v1/users/me", nil)
		req.Header.Set("Authorization", "Bearer "+suite.userToken)
		req.RemoteAddr = "192.168.1.100:12345" // 같은 IP에서 연속 요청
		
		w := httptest.NewRecorder()
		suite.app.ServeHTTP(w, req)
		
		if w.Code == http.StatusOK {
			successCount++
		} else if w.Code == http.StatusTooManyRequests {
			limitedCount++
		}
		
		// 짧은 간격으로 요청
		time.Sleep(10 * time.Millisecond)
	}
	
	t.Logf("Rate limiting results: %d success, %d limited out of 50 requests", 
		successCount, limitedCount)
	
	assert.Greater(t, limitedCount, 0, "Some requests should be rate limited")
	assert.Greater(t, successCount, 0, "Some requests should succeed")
	
	// 다른 IP에서의 요청은 영향받지 않아야 함
	req := httptest.NewRequest("GET", "/api/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+suite.limitedToken)
	req.RemoteAddr = "192.168.1.200:12345" // 다른 IP
	
	w := httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code, "Request from different IP should succeed")
}

// TestFileUploadSecurity 파일 업로드 보안 테스트
func (suite *APIEndpointSecurityTestSuite) TestFileUploadSecurity() {
	t := suite.T()
	
	// 1. 정상적인 파일 업로드
	validContent := "valid file content"
	req := suite.createFileUploadRequest("/api/v1/upload", "test.txt", "text/plain", validContent)
	req.Header.Set("Authorization", "Bearer "+suite.userToken)
	
	w := httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code, "Valid file upload should succeed")
	
	// 2. 악성 파일명 테스트
	maliciousFilenames := []string{
		"../../../etc/passwd",
		"..\\..\\..\\windows\\system32\\config\\sam",
		"shell.php",
		"script.js",
		"<script>alert('xss')</script>.txt",
		"file\x00.txt", // Null byte injection
	}
	
	for _, filename := range maliciousFilenames {
		req = suite.createFileUploadRequest("/api/v1/upload", filename, "text/plain", "content")
		req.Header.Set("Authorization", "Bearer "+suite.userToken)
		
		w = httptest.NewRecorder()
		suite.app.ServeHTTP(w, req)
		
		// 악성 파일명은 차단되거나 안전하게 처리되어야 함
		t.Logf("Malicious filename '%s' test result: %d", filename, w.Code)
		
		if w.Code == http.StatusOK {
			// 응답에서 원본 파일명이 그대로 노출되지 않는지 확인
			responseStr := w.Body.String()
			assert.NotContains(t, responseStr, "../")
			assert.NotContains(t, responseStr, "<script>")
		}
	}
	
	// 3. 과도하게 큰 파일 업로드 테스트
	largeContent := strings.Repeat("A", 50*1024*1024) // 50MB
	req = suite.createFileUploadRequest("/api/v1/upload", "large.txt", "text/plain", largeContent)
	req.Header.Set("Authorization", "Bearer "+suite.userToken)
	
	w = httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	// 대용량 파일은 거부되어야 함
	assert.NotEqual(t, http.StatusOK, w.Code, "Large file should be rejected")
	
	// 4. 실행 가능한 파일 업로드 테스트
	executableContent := "#!/bin/bash\necho 'malicious script'"
	req = suite.createFileUploadRequest("/api/v1/upload", "malicious.sh", "application/x-sh", executableContent)
	req.Header.Set("Authorization", "Bearer "+suite.userToken)
	
	w = httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	// 실행 가능한 파일은 차단되어야 함
	t.Logf("Executable file upload result: %d", w.Code)
}

// TestAPIAuthenticationScenarios API 인증 시나리오 테스트
func (suite *APIEndpointSecurityTestSuite) TestAPIAuthenticationScenarios() {
	t := suite.T()
	
	// 1. 토큰 없이 보호된 API 접근
	req := httptest.NewRequest("GET", "/api/v1/users/me", nil)
	w := httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusUnauthorized, w.Code, "Protected API should require authentication")
	
	// 2. 만료된 토큰으로 접근 (짧은 만료 시간으로 테스트)
	shortJWTManager := auth.NewJWTManager("test-secret", 1*time.Nanosecond, 1*time.Hour)
	expiredClaims := &auth.Claims{
		UserID:   "user-john",
		Email:    "john@test.com",
		Provider: "local",
	}
	expiredTokens, _ := shortJWTManager.GenerateTokens(expiredClaims)
	
	time.Sleep(10 * time.Millisecond) // 토큰 만료 대기
	
	req = httptest.NewRequest("GET", "/api/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+expiredTokens.AccessToken)
	w = httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusUnauthorized, w.Code, "Expired token should be rejected")
	
	// 3. 잘못된 서명의 토큰으로 접근
	wrongJWTManager := auth.NewJWTManager("wrong-secret", 15*time.Minute, 24*time.Hour)
	wrongClaims := &auth.Claims{
		UserID:   "user-john",
		Email:    "john@test.com",
		Provider: "local",
	}
	wrongTokens, _ := wrongJWTManager.GenerateTokens(wrongClaims)
	
	req = httptest.NewRequest("GET", "/api/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+wrongTokens.AccessToken)
	w = httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusUnauthorized, w.Code, "Token with wrong signature should be rejected")
	
	// 4. 토큰 재사용 공격 시뮬레이션
	// 동일한 토큰으로 여러 번 빠른 요청
	for i := 0; i < 5; i++ {
		req = httptest.NewRequest("GET", "/api/v1/users/me", nil)
		req.Header.Set("Authorization", "Bearer "+suite.userToken)
		
		w = httptest.NewRecorder()
		suite.app.ServeHTTP(w, req)
		
		// 정상적인 토큰은 재사용 가능해야 함 (세션 기반이 아닌 경우)
		assert.Equal(t, http.StatusOK, w.Code, "Valid token should be reusable")
	}
}

// TestAPIErrorHandling API 에러 처리 테스트
func (suite *APIEndpointSecurityTestSuite) TestAPIErrorHandling() {
	t := suite.T()
	
	// 1. 존재하지 않는 엔드포인트
	req := httptest.NewRequest("GET", "/api/v1/nonexistent", nil)
	req.Header.Set("Authorization", "Bearer "+suite.userToken)
	
	w := httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusNotFound, w.Code, "Nonexistent endpoint should return 404")
	
	// 응답에서 시스템 정보 노출 확인
	responseStr := w.Body.String()
	assert.NotContains(t, responseStr, "internal")
	assert.NotContains(t, responseStr, "panic")
	assert.NotContains(t, responseStr, "goroutine")
	
	// 2. 잘못된 HTTP 메서드
	req = httptest.NewRequest("PATCH", "/api/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+suite.userToken)
	
	w = httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusMethodNotAllowed, w.Code, "Wrong HTTP method should return 405")
	
	// 3. 잘못된 Content-Type
	req = httptest.NewRequest("PUT", "/api/v1/users/me", strings.NewReader(`{"name": "test"}`))
	req.Header.Set("Authorization", "Bearer "+suite.userToken)
	req.Header.Set("Content-Type", "text/plain") // JSON이 아닌 Content-Type
	
	w = httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusBadRequest, w.Code, "Wrong Content-Type should return 400")
	
	// 4. 서버 에러 시뮬레이션 (잘못된 리소스 ID)
	req = httptest.NewRequest("GET", "/api/v1/projects/invalid-id-format", nil)
	req.Header.Set("Authorization", "Bearer "+suite.userToken)
	
	w = httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	// 잘못된 ID 형식은 400 또는 404를 반환해야 함
	assert.True(t, w.Code == http.StatusBadRequest || w.Code == http.StatusNotFound,
		"Invalid ID format should return 400 or 404")
	
	// 에러 응답 형식 확인
	var errorResponse map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	if err == nil {
		// 표준화된 에러 응답 형식인지 확인
		assert.Contains(t, errorResponse, "error", "Error response should have error field")
		
		// 민감한 정보가 노출되지 않는지 확인
		errorStr := fmt.Sprintf("%v", errorResponse)
		assert.NotContains(t, errorStr, "database")
		assert.NotContains(t, errorStr, "sql")
		assert.NotContains(t, errorStr, "password")
	}
}

// TestConcurrentAPIAccess 동시 API 접근 테스트
func (suite *APIEndpointSecurityTestSuite) TestConcurrentAPIAccess() {
	t := suite.T()
	
	const numRequests = 50
	results := make(chan int, numRequests)
	
	// 동시에 같은 리소스에 접근
	for i := 0; i < numRequests; i++ {
		go func() {
			req := httptest.NewRequest("GET", "/api/v1/users/me", nil)
			req.Header.Set("Authorization", "Bearer "+suite.userToken)
			
			w := httptest.NewRecorder()
			suite.app.ServeHTTP(w, req)
			
			results <- w.Code
		}()
	}
	
	// 결과 수집
	var successCount, errorCount int
	for i := 0; i < numRequests; i++ {
		statusCode := <-results
		if statusCode == http.StatusOK {
			successCount++
		} else {
			errorCount++
		}
	}
	
	t.Logf("Concurrent access results: %d success, %d errors", successCount, errorCount)
	
	// 대부분의 요청이 성공해야 함
	assert.Greater(t, successCount, numRequests/2, "Most concurrent requests should succeed")
}

// createFileUploadRequest 파일 업로드 요청 생성 헬퍼
func (suite *APIEndpointSecurityTestSuite) createFileUploadRequest(url, filename, contentType, content string) *http.Request {
	body := &bytes.Buffer{}
	body.WriteString("--boundary\r\n")
	body.WriteString(fmt.Sprintf("Content-Disposition: form-data; name=\"file\"; filename=\"%s\"\r\n", filename))
	body.WriteString(fmt.Sprintf("Content-Type: %s\r\n", contentType))
	body.WriteString("\r\n")
	body.WriteString(content)
	body.WriteString("\r\n--boundary--\r\n")
	
	req := httptest.NewRequest("POST", url, body)
	req.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")
	
	return req
}

// 테스트 스위트 실행
func TestAPIEndpointSecuritySuite(t *testing.T) {
	suite.Run(t, new(APIEndpointSecurityTestSuite))
}

// 벤치마크 테스트
func BenchmarkAPIEndpointAccess(b *testing.B) {
	gin.SetMode(gin.TestMode)
	
	// 간단한 API 설정
	app := gin.New()
	app.Use(middleware.ErrorHandler())
	
	jwtManager := auth.NewJWTManager("test-secret", 15*time.Minute, 24*time.Hour)
	app.Use(middleware.AuthMiddleware(jwtManager))
	
	app.GET("/api/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "test"})
	})
	
	// 테스트 토큰 생성
	claims := &auth.Claims{
		UserID:   "bench-user",
		Email:    "bench@test.com",
		Provider: "local",
	}
	tokens, _ := jwtManager.GenerateTokens(claims)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/test", nil)
		req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
		
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			b.Fatalf("Expected 200, got %d", w.Code)
		}
	}
}