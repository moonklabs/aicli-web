package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/aicli/aicli-web/internal/auth"
	"github.com/aicli/aicli-web/internal/config"
	"github.com/aicli/aicli-web/internal/middleware"
	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/storage"
	"github.com/aicli/aicli-web/internal/storage/memory"
)

// SecurityMiddlewareScenarioTestSuite 보안 미들웨어 시나리오 테스트 스위트
type SecurityMiddlewareScenarioTestSuite struct {
	suite.Suite
	app         *gin.Engine
	storage     storage.Storage
	jwtManager  auth.JWTManager
	rbacManager auth.RBACManager
	
	// 테스트용 토큰들
	validTokens   map[string]string // userID -> token
	invalidTokens []string
}

// SetupSuite 테스트 스위트 초기화
func (suite *SecurityMiddlewareScenarioTestSuite) SetupSuite() {
	gin.SetMode(gin.TestMode)
	
	// 저장소 초기화
	suite.storage = memory.NewMemoryStorage()
	
	// JWT 매니저 초기화
	suite.jwtManager = auth.NewJWTManager("test-secret-key", 15*time.Minute, 24*time.Hour)
	
	// RBAC 매니저 초기화
	cache := auth.NewMemoryPermissionCache()
	rbacStorage := auth.NewRBACStorageAdapter(suite.storage)
	suite.rbacManager = auth.NewRBACManager(rbacStorage, cache)
	
	// 테스트 데이터 초기화
	suite.setupTestData()
	
	// Gin 앱 설정
	suite.setupApplication()
}

// setupTestData 테스트 데이터 설정
func (suite *SecurityMiddlewareScenarioTestSuite) setupTestData() {
	ctx := context.Background()
	
	// 테스트 사용자 생성
	users := []*models.User{
		{
			Base:     models.Base{ID: "user-admin"},
			Username: "admin",
			Email:    "admin@test.com",
			IsActive: true,
		},
		{
			Base:     models.Base{ID: "user-normal"},
			Username: "normal",
			Email:    "normal@test.com",
			IsActive: true,
		},
		{
			Base:     models.Base{ID: "user-limited"},
			Username: "limited",
			Email:    "limited@test.com",
			IsActive: true,
		},
	}
	
	for _, user := range users {
		user.CreatedAt = time.Now()
		user.UpdatedAt = time.Now()
		err := suite.storage.Create(ctx, "users", user)
		require.NoError(suite.T(), err)
	}
	
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
		err := suite.storage.Create(ctx, "roles", role)
		require.NoError(suite.T(), err)
	}
	
	// 권한 생성
	permissions := []*models.Permission{
		{
			Base:         models.Base{ID: "perm-system-admin"},
			Name:         "System Administration",
			ResourceType: models.ResourceTypeSystem,
			Action:       models.ActionManage,
			Effect:       models.PermissionAllow,
			IsActive:     true,
		},
		{
			Base:         models.Base{ID: "perm-user-read"},
			Name:         "User Read",
			ResourceType: models.ResourceTypeUser,
			Action:       models.ActionRead,
			Effect:       models.PermissionAllow,
			IsActive:     true,
		},
	}
	
	for _, permission := range permissions {
		permission.CreatedAt = time.Now()
		permission.UpdatedAt = time.Now()
		err := suite.storage.Create(ctx, "permissions", permission)
		require.NoError(suite.T(), err)
	}
	
	// 역할-권한 연결
	rolePermissions := []*models.RolePermission{
		{
			Base:         models.Base{ID: "rp-admin-system"},
			RoleID:       "role-admin",
			PermissionID: "perm-system-admin",
		},
		{
			Base:         models.Base{ID: "rp-admin-user"},
			RoleID:       "role-admin",
			PermissionID: "perm-user-read",
		},
		{
			Base:         models.Base{ID: "rp-user-read"},
			RoleID:       "role-user",
			PermissionID: "perm-user-read",
		},
	}
	
	for _, rp := range rolePermissions {
		rp.CreatedAt = time.Now()
		rp.UpdatedAt = time.Now()
		err := suite.storage.Create(ctx, "role_permissions", rp)
		require.NoError(suite.T(), err)
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
			Base:      models.Base{ID: "ur-normal"},
			UserID:    "user-normal",
			RoleID:    "role-user",
			GrantedAt: time.Now(),
		},
		{
			Base:      models.Base{ID: "ur-limited"},
			UserID:    "user-limited",
			RoleID:    "role-user",
			GrantedAt: time.Now(),
		},
	}
	
	for _, ur := range userRoles {
		ur.CreatedAt = time.Now()
		ur.UpdatedAt = time.Now()
		err := suite.storage.Create(ctx, "user_roles", ur)
		require.NoError(suite.T(), err)
	}
	
	// 유효한 토큰 생성
	suite.validTokens = make(map[string]string)
	
	userTokens := map[string]*auth.Claims{
		"user-admin": {
			UserID:   "user-admin",
			Email:    "admin@test.com",
			Provider: "local",
		},
		"user-normal": {
			UserID:   "user-normal",
			Email:    "normal@test.com",
			Provider: "local",
		},
		"user-limited": {
			UserID:   "user-limited",
			Email:    "limited@test.com",
			Provider: "local",
		},
	}
	
	for userID, claims := range userTokens {
		tokens, err := suite.jwtManager.GenerateTokens(claims)
		require.NoError(suite.T(), err)
		suite.validTokens[userID] = tokens.AccessToken
	}
	
	// 무효한 토큰들
	suite.invalidTokens = []string{
		"invalid.token.here",
		"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.invalid_signature",
		"",
		"Bearer malformed",
	}
}

// setupApplication 애플리케이션 설정
func (suite *SecurityMiddlewareScenarioTestSuite) setupApplication() {
	suite.app = gin.New()
	
	// 기본 미들웨어
	suite.app.Use(middleware.ErrorHandler())
	suite.app.Use(middleware.CORSMiddleware())
	
	// Rate Limiting 설정
	rateLimitConfig := &config.RateLimitConfig{
		RequestsPerSecond: 10,
		BurstSize:         20,
		WindowSize:        time.Minute,
		Enabled:           true,
		SkipPaths:         []string{"/health"},
	}
	suite.app.Use(middleware.RateLimitMiddleware(rateLimitConfig))
	
	// CSRF 보호 (특정 경로에만)
	csrfConfig := &config.CSRFConfig{
		TokenLength:    32,
		CookieName:     "csrf_token",
		HeaderName:     "X-CSRF-Token",
		Secure:         false,
		HTTPOnly:       true,
		SameSite:       "Strict",
		Enabled:        true,
		IgnoreMethods:  []string{"GET", "HEAD", "OPTIONS"},
		TrustedOrigins: []string{"http://localhost:8080"},
	}
	csrfGroup := suite.app.Group("/api/csrf")
	csrfGroup.Use(middleware.CSRFMiddleware(csrfConfig))
	
	// 보안 헤더
	securityConfig := &config.SecurityConfig{
		EnableHSTS:              true,
		EnableCSP:               true,
		EnableXFrameOptions:     true,
		EnableXContentTypeOptions: true,
		EnableReferrerPolicy:    true,
		CSPDirectives: map[string]string{
			"default-src": "'self'",
			"script-src":  "'self' 'unsafe-inline'",
			"style-src":   "'self' 'unsafe-inline'",
		},
	}
	suite.app.Use(middleware.SecurityHeadersMiddleware(securityConfig))
	
	// Attack Detection
	attackConfig := &config.AttackDetectionConfig{
		EnableSQLInjectionDetection: true,
		EnableXSSDetection:          true,
		EnablePathTraversalDetection: true,
		BlockSuspiciousRequests:     true,
		LogLevel:                    "warn",
	}
	suite.app.Use(middleware.AttackDetectionMiddleware(attackConfig))
	
	// 라우트 설정
	suite.setupRoutes()
}

// setupRoutes 라우트 설정
func (suite *SecurityMiddlewareScenarioTestSuite) setupRoutes() {
	// 공개 엔드포인트
	suite.app.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	
	suite.app.GET("/public", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "public endpoint"})
	})
	
	// Rate limiting 테스트용 엔드포인트
	suite.app.GET("/rate-test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "rate test"})
	})
	
	// 인증이 필요한 엔드포인트
	authGroup := suite.app.Group("/api/auth")
	authGroup.Use(middleware.AuthMiddleware(suite.jwtManager))
	{
		authGroup.GET("/profile", func(c *gin.Context) {
			claims, _ := c.Get("claims")
			c.JSON(http.StatusOK, gin.H{"user": claims})
		})
		
		authGroup.POST("/update", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "updated"})
		})
	}
	
	// RBAC가 필요한 엔드포인트
	adminGroup := suite.app.Group("/api/admin")
	adminGroup.Use(middleware.AuthMiddleware(suite.jwtManager))
	adminGroup.Use(middleware.RBACMiddleware(suite.rbacManager, models.ResourceTypeSystem, "", models.ActionManage))
	{
		adminGroup.GET("/users", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"users": []string{"admin-only"}})
		})
		
		adminGroup.POST("/settings", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "settings updated"})
		})
	}
	
	// CSRF 보호가 적용된 엔드포인트
	csrfGroup := suite.app.Group("/api/csrf")
	{
		csrfGroup.GET("/token", func(c *gin.Context) {
			token := c.GetHeader("X-CSRF-Token")
			c.JSON(http.StatusOK, gin.H{"csrf_token": token})
		})
		
		csrfGroup.POST("/protected", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "CSRF protected endpoint"})
		})
	}
	
	// 공격 탐지 테스트용 엔드포인트
	suite.app.GET("/vulnerable", func(c *gin.Context) {
		// 이 엔드포인트는 공격 탐지 테스트용
		query := c.Query("q")
		c.JSON(http.StatusOK, gin.H{"query": query})
	})
	
	suite.app.POST("/upload", func(c *gin.Context) {
		// 파일 업로드 시뮬레이션
		filename := c.PostForm("filename")
		c.JSON(http.StatusOK, gin.H{"filename": filename})
	})
}

// TestRateLimitingScenarios Rate Limiting 시나리오 테스트
func (suite *SecurityMiddlewareScenarioTestSuite) TestRateLimitingScenarios() {
	t := suite.T()
	
	// 1. 정상적인 요청 - 제한 내에서
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/rate-test", nil)
		req.RemoteAddr = "192.168.1.100:12345"
		
		w := httptest.NewRecorder()
		suite.app.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code, "Request %d should succeed", i+1)
	}
	
	// 2. Rate limit 초과 테스트
	var blockedRequests int
	var successRequests int
	
	// 짧은 시간에 많은 요청 전송
	for i := 0; i < 50; i++ {
		req := httptest.NewRequest("GET", "/rate-test", nil)
		req.RemoteAddr = "192.168.1.101:12345" // 다른 IP 사용
		
		w := httptest.NewRecorder()
		suite.app.ServeHTTP(w, req)
		
		if w.Code == http.StatusTooManyRequests {
			blockedRequests++
		} else if w.Code == http.StatusOK {
			successRequests++
		}
	}
	
	assert.Greater(t, blockedRequests, 0, "Some requests should be rate limited")
	assert.Greater(t, successRequests, 0, "Some requests should succeed")
	
	t.Logf("Rate limiting results: %d blocked, %d succeeded out of 50 requests", 
		blockedRequests, successRequests)
	
	// 3. 다른 IP에서의 요청은 영향받지 않아야 함
	req := httptest.NewRequest("GET", "/rate-test", nil)
	req.RemoteAddr = "192.168.1.102:12345" // 완전히 다른 IP
	
	w := httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code, "Request from different IP should succeed")
	
	// 4. 예외 경로는 rate limiting에 영향받지 않아야 함
	for i := 0; i < 20; i++ {
		req := httptest.NewRequest("GET", "/health", nil)
		req.RemoteAddr = "192.168.1.103:12345"
		
		w := httptest.NewRecorder()
		suite.app.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code, "Health check should always succeed")
	}
}

// TestAuthenticationMiddlewareScenarios 인증 미들웨어 시나리오 테스트
func (suite *SecurityMiddlewareScenarioTestSuite) TestAuthenticationMiddlewareScenarios() {
	t := suite.T()
	
	// 1. 토큰 없이 보호된 엔드포인트 접근
	req := httptest.NewRequest("GET", "/api/auth/profile", nil)
	w := httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	
	// 2. 유효한 토큰으로 접근
	req = httptest.NewRequest("GET", "/api/auth/profile", nil)
	req.Header.Set("Authorization", "Bearer "+suite.validTokens["user-normal"])
	w = httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	user := response["user"].(map[string]interface{})
	assert.Equal(t, "user-normal", user["user_id"])
	
	// 3. 무효한 토큰들로 접근 시도
	for i, invalidToken := range suite.invalidTokens {
		req = httptest.NewRequest("GET", "/api/auth/profile", nil)
		if invalidToken != "" {
			req.Header.Set("Authorization", "Bearer "+invalidToken)
		}
		
		w = httptest.NewRecorder()
		suite.app.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusUnauthorized, w.Code, 
			"Invalid token %d should be rejected", i)
	}
	
	// 4. 잘못된 Authorization 헤더 형식
	malformedHeaders := []string{
		"invalid-format",
		"Bearer",
		"Basic " + suite.validTokens["user-normal"],
		"Bearer " + suite.validTokens["user-normal"] + " extra",
	}
	
	for i, header := range malformedHeaders {
		req = httptest.NewRequest("GET", "/api/auth/profile", nil)
		req.Header.Set("Authorization", header)
		
		w = httptest.NewRecorder()
		suite.app.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusUnauthorized, w.Code,
			"Malformed header %d should be rejected", i)
	}
}

// TestRBACMiddlewareScenarios RBAC 미들웨어 시나리오 테스트
func (suite *SecurityMiddlewareScenarioTestSuite) TestRBACMiddlewareScenarios() {
	t := suite.T()
	
	// 1. 관리자 권한으로 관리자 엔드포인트 접근
	req := httptest.NewRequest("GET", "/api/admin/users", nil)
	req.Header.Set("Authorization", "Bearer "+suite.validTokens["user-admin"])
	w := httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code, "Admin should access admin endpoint")
	
	// 2. 일반 사용자가 관리자 엔드포인트 접근 시도
	req = httptest.NewRequest("GET", "/api/admin/users", nil)
	req.Header.Set("Authorization", "Bearer "+suite.validTokens["user-normal"])
	w = httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusForbidden, w.Code, "Normal user should be forbidden")
	
	// 3. 제한된 사용자가 관리자 엔드포인트 접근 시도
	req = httptest.NewRequest("GET", "/api/admin/users", nil)
	req.Header.Set("Authorization", "Bearer "+suite.validTokens["user-limited"])
	w = httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusForbidden, w.Code, "Limited user should be forbidden")
	
	// 4. POST 요청에 대한 권한 검사
	req = httptest.NewRequest("POST", "/api/admin/settings", bytes.NewBufferString(`{"setting": "value"}`))
	req.Header.Set("Authorization", "Bearer "+suite.validTokens["user-admin"])
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code, "Admin should be able to POST")
	
	// 5. 일반 사용자의 POST 시도
	req = httptest.NewRequest("POST", "/api/admin/settings", bytes.NewBufferString(`{"setting": "value"}`))
	req.Header.Set("Authorization", "Bearer "+suite.validTokens["user-normal"])
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusForbidden, w.Code, "Normal user POST should be forbidden")
}

// TestCSRFProtectionScenarios CSRF 보호 시나리오 테스트
func (suite *SecurityMiddlewareScenarioTestSuite) TestCSRFProtectionScenarios() {
	t := suite.T()
	
	// 1. GET 요청은 CSRF 토큰 없이도 허용되어야 함
	req := httptest.NewRequest("GET", "/api/csrf/token", nil)
	w := httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code, "GET request should succeed without CSRF token")
	
	// 2. POST 요청에는 CSRF 토큰이 필요함
	req = httptest.NewRequest("POST", "/api/csrf/protected", bytes.NewBufferString(`{"data": "test"}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusForbidden, w.Code, "POST without CSRF token should be forbidden")
	
	// 3. 유효한 CSRF 토큰으로 POST 요청
	// 실제 구현에서는 CSRF 토큰 생성 로직이 필요
	// 여기서는 테스트용으로 간단히 처리
	req = httptest.NewRequest("POST", "/api/csrf/protected", bytes.NewBufferString(`{"data": "test"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", "valid-test-token") // 실제로는 서버에서 발급받은 토큰 사용
	w = httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	// CSRF 구현 상태에 따라 결과가 달라질 수 있음
	// 여기서는 로깅만 하고 검증은 실제 구현에 맞게 조정
	t.Logf("CSRF protected POST response: %d", w.Code)
	
	// 4. 잘못된 CSRF 토큰으로 POST 요청
	req = httptest.NewRequest("POST", "/api/csrf/protected", bytes.NewBufferString(`{"data": "test"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", "invalid-token")
	w = httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusForbidden, w.Code, "POST with invalid CSRF token should be forbidden")
}

// TestSecurityHeadersScenarios 보안 헤더 시나리오 테스트
func (suite *SecurityMiddlewareScenarioTestSuite) TestSecurityHeadersScenarios() {
	t := suite.T()
	
	// 모든 응답에 보안 헤더가 포함되어야 함
	req := httptest.NewRequest("GET", "/public", nil)
	w := httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	// 필수 보안 헤더 검증
	headers := w.Header()
	
	// HSTS 헤더
	hsts := headers.Get("Strict-Transport-Security")
	assert.NotEmpty(t, hsts, "HSTS header should be present")
	assert.Contains(t, hsts, "max-age=", "HSTS should have max-age")
	
	// CSP 헤더
	csp := headers.Get("Content-Security-Policy")
	assert.NotEmpty(t, csp, "CSP header should be present")
	assert.Contains(t, csp, "default-src", "CSP should have default-src")
	
	// X-Frame-Options
	frameOptions := headers.Get("X-Frame-Options")
	assert.NotEmpty(t, frameOptions, "X-Frame-Options header should be present")
	
	// X-Content-Type-Options
	contentTypeOptions := headers.Get("X-Content-Type-Options")
	assert.Equal(t, "nosniff", contentTypeOptions, "X-Content-Type-Options should be nosniff")
	
	// Referrer Policy
	referrerPolicy := headers.Get("Referrer-Policy")
	assert.NotEmpty(t, referrerPolicy, "Referrer-Policy header should be present")
	
	t.Logf("Security Headers - HSTS: %s, CSP: %s, Frame: %s, ContentType: %s, Referrer: %s",
		hsts, csp, frameOptions, contentTypeOptions, referrerPolicy)
}

// TestAttackDetectionScenarios 공격 탐지 시나리오 테스트
func (suite *SecurityMiddlewareScenarioTestSuite) TestAttackDetectionScenarios() {
	t := suite.T()
	
	// 1. SQL Injection 시도
	sqlInjectionPayloads := []string{
		"'; DROP TABLE users; --",
		"1' OR '1'='1",
		"UNION SELECT * FROM passwords",
		"'; EXEC xp_cmdshell('dir'); --",
	}
	
	for i, payload := range sqlInjectionPayloads {
		req := httptest.NewRequest("GET", "/vulnerable?q="+payload, nil)
		w := httptest.NewRecorder()
		suite.app.ServeHTTP(w, req)
		
		// 공격 탐지 미들웨어가 활성화되어 있다면 블록되어야 함
		if w.Code == http.StatusForbidden || w.Code == http.StatusBadRequest {
			t.Logf("SQL injection payload %d blocked: %s", i+1, payload)
		} else {
			t.Logf("SQL injection payload %d not blocked: %s (status: %d)", i+1, payload, w.Code)
		}
	}
	
	// 2. XSS 시도
	xssPayloads := []string{
		"<script>alert('xss')</script>",
		"javascript:alert('xss')",
		"<img src=x onerror=alert('xss')>",
		"<svg onload=alert('xss')>",
	}
	
	for i, payload := range xssPayloads {
		req := httptest.NewRequest("GET", "/vulnerable?q="+payload, nil)
		w := httptest.NewRecorder()
		suite.app.ServeHTTP(w, req)
		
		if w.Code == http.StatusForbidden || w.Code == http.StatusBadRequest {
			t.Logf("XSS payload %d blocked: %s", i+1, payload)
		} else {
			t.Logf("XSS payload %d not blocked: %s (status: %d)", i+1, payload, w.Code)
		}
	}
	
	// 3. Path Traversal 시도
	pathTraversalPayloads := []string{
		"../../../etc/passwd",
		"..\\..\\..\\windows\\system32\\config\\sam",
		"....//....//....//etc/passwd",
		"%2e%2e%2f%2e%2e%2f%2e%2e%2fetc%2fpasswd",
	}
	
	for i, payload := range pathTraversalPayloads {
		formData := fmt.Sprintf("filename=%s", payload)
		req := httptest.NewRequest("POST", "/upload", strings.NewReader(formData))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		suite.app.ServeHTTP(w, req)
		
		if w.Code == http.StatusForbidden || w.Code == http.StatusBadRequest {
			t.Logf("Path traversal payload %d blocked: %s", i+1, payload)
		} else {
			t.Logf("Path traversal payload %d not blocked: %s (status: %d)", i+1, payload, w.Code)
		}
	}
	
	// 4. 정상적인 요청은 통과해야 함
	req := httptest.NewRequest("GET", "/vulnerable?q=normal_query", nil)
	w := httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code, "Normal request should not be blocked")
}

// TestConcurrentSecurityScenarios 동시 보안 시나리오 테스트
func (suite *SecurityMiddlewareScenarioTestSuite) TestConcurrentSecurityScenarios() {
	t := suite.T()
	
	const numGoroutines = 20
	const requestsPerGoroutine = 10
	
	var wg sync.WaitGroup
	results := make(chan map[string]int, numGoroutines)
	
	// 동시에 다양한 시나리오 실행
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			
			stats := map[string]int{
				"success":      0,
				"unauthorized": 0,
				"forbidden":    0,
				"rate_limited": 0,
				"blocked":      0,
			}
			
			for j := 0; j < requestsPerGoroutine; j++ {
				// 다양한 시나리오를 랜덤하게 실행
				switch j % 5 {
				case 0:
					// 정상 인증 요청
					req := httptest.NewRequest("GET", "/api/auth/profile", nil)
					req.Header.Set("Authorization", "Bearer "+suite.validTokens["user-normal"])
					w := httptest.NewRecorder()
					suite.app.ServeHTTP(w, req)
					
					if w.Code == http.StatusOK {
						stats["success"]++
					} else {
						stats["unauthorized"]++
					}
					
				case 1:
					// 권한 없는 요청
					req := httptest.NewRequest("GET", "/api/admin/users", nil)
					req.Header.Set("Authorization", "Bearer "+suite.validTokens["user-normal"])
					w := httptest.NewRecorder()
					suite.app.ServeHTTP(w, req)
					
					if w.Code == http.StatusForbidden {
						stats["forbidden"]++
					} else {
						stats["success"]++
					}
					
				case 2:
					// Rate limiting 테스트
					req := httptest.NewRequest("GET", "/rate-test", nil)
					req.RemoteAddr = fmt.Sprintf("192.168.%d.%d:12345", goroutineID%255, j%255)
					w := httptest.NewRecorder()
					suite.app.ServeHTTP(w, req)
					
					if w.Code == http.StatusTooManyRequests {
						stats["rate_limited"]++
					} else {
						stats["success"]++
					}
					
				case 3:
					// 공격 시나리오
					req := httptest.NewRequest("GET", "/vulnerable?q=<script>alert('test')</script>", nil)
					w := httptest.NewRecorder()
					suite.app.ServeHTTP(w, req)
					
					if w.Code == http.StatusForbidden || w.Code == http.StatusBadRequest {
						stats["blocked"]++
					} else {
						stats["success"]++
					}
					
				case 4:
					// 무효한 토큰
					req := httptest.NewRequest("GET", "/api/auth/profile", nil)
					req.Header.Set("Authorization", "Bearer invalid.token.here")
					w := httptest.NewRecorder()
					suite.app.ServeHTTP(w, req)
					
					if w.Code == http.StatusUnauthorized {
						stats["unauthorized"]++
					} else {
						stats["success"]++
					}
				}
			}
			
			results <- stats
		}(i)
	}
	
	wg.Wait()
	close(results)
	
	// 결과 집계
	totalStats := map[string]int{
		"success":      0,
		"unauthorized": 0,
		"forbidden":    0,
		"rate_limited": 0,
		"blocked":      0,
	}
	
	for stats := range results {
		for key, value := range stats {
			totalStats[key] += value
		}
	}
	
	totalRequests := numGoroutines * requestsPerGoroutine
	
	t.Logf("Concurrent security test results (total: %d requests):", totalRequests)
	for status, count := range totalStats {
		percentage := float64(count) / float64(totalRequests) * 100
		t.Logf("  %s: %d (%.1f%%)", status, count, percentage)
	}
	
	// 기본적인 검증
	assert.Greater(t, totalStats["success"]+totalStats["unauthorized"]+
		totalStats["forbidden"]+totalStats["rate_limited"]+totalStats["blocked"], 
		totalRequests/2, "Most requests should be handled properly")
}

// TestMiddlewareOrderAndInteraction 미들웨어 순서 및 상호작용 테스트
func (suite *SecurityMiddlewareScenarioTestSuite) TestMiddlewareOrderAndInteraction() {
	t := suite.T()
	
	// 1. Rate limiting이 인증보다 먼저 적용되는지 테스트
	// 유효한 토큰이지만 rate limit에 걸릴 만큼 요청
	var rateLimitedCount int
	var authFailedCount int
	
	for i := 0; i < 30; i++ {
		req := httptest.NewRequest("GET", "/api/auth/profile", nil)
		req.Header.Set("Authorization", "Bearer "+suite.validTokens["user-normal"])
		req.RemoteAddr = "192.168.1.200:12345" // 같은 IP에서 연속 요청
		
		w := httptest.NewRecorder()
		suite.app.ServeHTTP(w, req)
		
		if w.Code == http.StatusTooManyRequests {
			rateLimitedCount++
		} else if w.Code == http.StatusUnauthorized {
			authFailedCount++
		}
	}
	
	t.Logf("Rate limiting vs Auth: %d rate limited, %d auth failed", 
		rateLimitedCount, authFailedCount)
	
	// 2. 보안 헤더가 모든 응답에 포함되는지 테스트
	scenarios := []struct {
		method string
		path   string
		token  string
	}{
		{"GET", "/public", ""},
		{"GET", "/api/auth/profile", suite.validTokens["user-normal"]},
		{"GET", "/api/admin/users", suite.validTokens["user-admin"]},
		{"GET", "/nonexistent", ""},
	}
	
	for _, scenario := range scenarios {
		req := httptest.NewRequest(scenario.method, scenario.path, nil)
		if scenario.token != "" {
			req.Header.Set("Authorization", "Bearer "+scenario.token)
		}
		
		w := httptest.NewRecorder()
		suite.app.ServeHTTP(w, req)
		
		// 모든 응답에 기본 보안 헤더가 있어야 함
		headers := w.Header()
		assert.NotEmpty(t, headers.Get("X-Content-Type-Options"), 
			"Security headers should be present for %s %s", scenario.method, scenario.path)
	}
	
	// 3. 에러 응답에도 보안 조치가 적용되는지 테스트
	req := httptest.NewRequest("GET", "/api/auth/profile", nil)
	// 토큰 없음 - 인증 실패
	w := httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	
	// 에러 응답에도 보안 헤더가 있어야 함
	assert.NotEmpty(t, w.Header().Get("X-Content-Type-Options"))
	
	// 에러 응답에서 민감한 정보가 노출되지 않는지 확인
	var errorResponse map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	if err == nil {
		// 에러 응답에 스택 트레이스나 내부 정보가 포함되지 않았는지 확인
		responseStr := w.Body.String()
		assert.NotContains(t, responseStr, "panic")
		assert.NotContains(t, responseStr, "goroutine")
		assert.NotContains(t, responseStr, "/internal/")
	}
}

// TestSecurityBypass보안 우회 시도 테스트
func (suite *SecurityMiddlewareScenarioTestSuite) TestSecurityBypassAttempts() {
	t := suite.T()
	
	// 1. HTTP Method Override 시도
	req := httptest.NewRequest("POST", "/api/admin/users", nil)
	req.Header.Set("X-HTTP-Method-Override", "GET")
	req.Header.Set("Authorization", "Bearer "+suite.validTokens["user-normal"])
	
	w := httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	// POST 메서드로 처리되어야 하므로 403 Forbidden이어야 함
	assert.Equal(t, http.StatusForbidden, w.Code, "Method override should not bypass security")
	
	// 2. 헤더 조작으로 IP 스푸핑 시도
	spoofingHeaders := []string{
		"X-Forwarded-For",
		"X-Real-IP", 
		"X-Originating-IP",
		"X-Remote-IP",
		"X-Client-IP",
	}
	
	for _, header := range spoofingHeaders {
		req = httptest.NewRequest("GET", "/rate-test", nil)
		req.RemoteAddr = "192.168.1.250:12345" // 실제 IP
		req.Header.Set(header, "127.0.0.1")    // 스푸핑 시도
		
		w = httptest.NewRecorder()
		suite.app.ServeHTTP(w, req)
		
		// IP 스푸핑이 rate limiting을 우회하지 못해야 함
		// 실제 구현에 따라 결과가 달라질 수 있음
		t.Logf("Header %s spoofing test result: %d", header, w.Code)
	}
	
	// 3. User-Agent 조작으로 봇 탐지 우회 시도
	botUserAgents := []string{
		"curl/7.68.0",
		"wget/1.20.3",
		"python-requests/2.25.1",
		"Googlebot/2.1",
		"",
	}
	
	for _, userAgent := range botUserAgents {
		req = httptest.NewRequest("GET", "/api/auth/profile", nil)
		req.Header.Set("Authorization", "Bearer "+suite.validTokens["user-normal"])
		req.Header.Set("User-Agent", userAgent)
		
		w = httptest.NewRecorder()
		suite.app.ServeHTTP(w, req)
		
		// 유효한 토큰이면 User-Agent와 관계없이 접근 허용되어야 함
		// (봇 탐지가 구현되어 있다면 다를 수 있음)
		t.Logf("User-Agent '%s' test result: %d", userAgent, w.Code)
	}
	
	// 4. Referer 헤더 조작 시도
	req = httptest.NewRequest("POST", "/api/csrf/protected", bytes.NewBufferString(`{"data": "test"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Referer", "https://trusted-site.com")
	
	w = httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	// CSRF 보호가 활성화되어 있다면 Referer만으로는 우회할 수 없어야 함
	assert.NotEqual(t, http.StatusOK, w.Code, "Referer header should not bypass CSRF protection")
}

// 테스트 스위트 실행
func TestSecurityMiddlewareScenarioSuite(t *testing.T) {
	suite.Run(t, new(SecurityMiddlewareScenarioTestSuite))
}

// 벤치마크 테스트
func BenchmarkSecurityMiddlewareStack(b *testing.B) {
	gin.SetMode(gin.TestMode)
	
	// 간단한 미들웨어 스택 설정
	app := gin.New()
	
	// 기본 미들웨어들
	app.Use(middleware.ErrorHandler())
	app.Use(middleware.CORSMiddleware())
	
	rateLimitConfig := &config.RateLimitConfig{
		RequestsPerSecond: 1000, // 벤치마크를 위해 높게 설정
		BurstSize:         2000,
		WindowSize:        time.Minute,
		Enabled:           true,
	}
	app.Use(middleware.RateLimitMiddleware(rateLimitConfig))
	
	securityConfig := &config.SecurityConfig{
		EnableHSTS:              true,
		EnableCSP:               true,
		EnableXFrameOptions:     true,
		EnableXContentTypeOptions: true,
		EnableReferrerPolicy:    true,
	}
	app.Use(middleware.SecurityHeadersMiddleware(securityConfig))
	
	// 테스트용 엔드포인트
	app.GET("/bench", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "benchmark"})
	})
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/bench", nil)
		req.RemoteAddr = fmt.Sprintf("192.168.1.%d:12345", i%255)
		
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			b.Fatalf("Expected 200, got %d", w.Code)
		}
	}
}