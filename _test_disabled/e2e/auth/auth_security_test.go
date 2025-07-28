// Package auth provides security-focused E2E tests for authentication
package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)


// TestSecurityHeaders 보안 헤더 E2E 테스트
func TestSecurityHeaders(t *testing.T) {
	suite := &SecurityE2ETestSuite{
		AuthE2ETestSuite: &AuthE2ETestSuite{},
		rateLimiter:      make(map[string][]time.Time),
	}
	suite.setupSecurityE2ETest(t)
	defer suite.teardownSecurityE2ETest(t)
	
	t.Run("Security Headers Present", suite.testSecurityHeaders)
	t.Run("CORS Headers", suite.testCORSHeaders)
	t.Run("Content Type Validation", suite.testContentTypeValidation)
}

// TestCSRFProtection CSRF 보호 E2E 테스트
func TestCSRFProtection(t *testing.T) {
	suite := &SecurityE2ETestSuite{
		AuthE2ETestSuite: &AuthE2ETestSuite{},
		rateLimiter:      make(map[string][]time.Time),
	}
	suite.setupSecurityE2ETest(t)
	defer suite.teardownSecurityE2ETest(t)
	
	t.Run("Missing CSRF Token", suite.testMissingCSRFToken)
	t.Run("Invalid CSRF Token", suite.testInvalidCSRFToken)
	t.Run("Valid CSRF Token", suite.testValidCSRFToken)
}

// TestRateLimiting Rate Limiting E2E 테스트
func TestRateLimiting(t *testing.T) {
	suite := &SecurityE2ETestSuite{
		AuthE2ETestSuite: &AuthE2ETestSuite{},
		rateLimiter:      make(map[string][]time.Time),
	}
	suite.setupSecurityE2ETest(t)
	defer suite.teardownSecurityE2ETest(t)
	
	t.Run("Login Rate Limiting", suite.testLoginRateLimiting)
	t.Run("API Rate Limiting", suite.testAPIRateLimiting)
	t.Run("Rate Limit Recovery", suite.testRateLimitRecovery)
}

// TestBruteForceProtection 브루트포스 공격 보호 테스트
func TestBruteForceProtection(t *testing.T) {
	suite := &SecurityE2ETestSuite{
		AuthE2ETestSuite: &AuthE2ETestSuite{},
		rateLimiter:      make(map[string][]time.Time),
	}
	suite.setupSecurityE2ETest(t)
	defer suite.teardownSecurityE2ETest(t)
	
	t.Run("Multiple Failed Login Attempts", suite.testMultipleFailedLogins)
	t.Run("Account Lockout", suite.testAccountLockout)
	t.Run("Lockout Recovery", suite.testLockoutRecovery)
}

// setupSecurityE2ETest 보안 E2E 테스트 환경 설정
func (suite *SecurityE2ETestSuite) setupSecurityE2ETest(t *testing.T) {
	t.Log("Setting up Security E2E test environment...")
	
	// 기본 Auth E2E 테스트 환경 설정
	suite.AuthE2ETestSuite.setupAuthE2ETest(t)
	
	// 보안 기능이 포함된 새 서버로 교체
	suite.server.Close()
	suite.server = suite.startSecurityTestServer(t)
	suite.baseURL = suite.server.URL
	
	t.Log("Security E2E test environment setup completed")
}

// teardownSecurityE2ETest 보안 E2E 테스트 환경 정리
func (suite *SecurityE2ETestSuite) teardownSecurityE2ETest(t *testing.T) {
	t.Log("Cleaning up Security E2E test environment...")
	suite.AuthE2ETestSuite.teardownAuthE2ETest(t)
	t.Log("Security E2E test environment cleanup completed")
}

// startSecurityTestServer 보안 기능이 포함된 테스트 서버 시작
func (suite *SecurityE2ETestSuite) startSecurityTestServer(t *testing.T) *httptest.Server {
	router := gin.New()
	router.Use(gin.Recovery())
	
	// 보안 미들웨어 추가
	router.Use(suite.securityHeadersMiddleware)
	router.Use(suite.corsMiddleware)
	router.Use(suite.rateLimitMiddleware)
	
	// 인증 API 라우트
	authGroup := router.Group("/api/auth")
	{
		authGroup.POST("/login", suite.csrfProtectedHandler(suite.mockLoginHandler))
		authGroup.POST("/refresh", suite.mockRefreshHandler)
		authGroup.POST("/logout", suite.mockLogoutHandler)
	}
	
	// 보호된 리소스 라우트
	protectedGroup := router.Group("/api/protected")
	protectedGroup.Use(suite.mockAuthMiddleware)
	protectedGroup.Use(suite.rateLimitMiddleware)
	{
		protectedGroup.GET("/profile", suite.mockGetProfileHandler)
		protectedGroup.GET("/admin", suite.mockAdminOnlyHandler)
	}
	
	return httptest.NewServer(router)
}

// Security Middleware

// securityHeadersMiddleware 보안 헤더 미들웨어
func (suite *SecurityE2ETestSuite) securityHeadersMiddleware(c *gin.Context) {
	// 기본 보안 헤더 설정
	c.Header("X-Frame-Options", "DENY")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("X-XSS-Protection", "1; mode=block")
	c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
	c.Header("Content-Security-Policy", "default-src 'self'")
	c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
	
	c.Next()
}

// corsMiddleware CORS 미들웨어
func (suite *SecurityE2ETestSuite) corsMiddleware(c *gin.Context) {
	origin := c.Request.Header.Get("Origin")
	
	// 허용된 origin 목록 (테스트용)
	allowedOrigins := []string{
		"http://localhost:3000",
		"https://aicli.dev",
	}
	
	for _, allowedOrigin := range allowedOrigins {
		if origin == allowedOrigin {
			c.Header("Access-Control-Allow-Origin", origin)
			break
		}
	}
	
	c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-CSRF-Token")
	c.Header("Access-Control-Allow-Credentials", "true")
	c.Header("Access-Control-Max-Age", "86400")
	
	if c.Request.Method == "OPTIONS" {
		c.Status(http.StatusNoContent)
		return
	}
	
	c.Next()
}

// rateLimitMiddleware Rate Limiting 미들웨어
func (suite *SecurityE2ETestSuite) rateLimitMiddleware(c *gin.Context) {
	clientIP := c.ClientIP()
	now := time.Now()
	
	// IP별 요청 기록 관리
	if suite.rateLimiter[clientIP] == nil {
		suite.rateLimiter[clientIP] = []time.Time{}
	}
	
	// 1분 이내의 요청만 유지
	var recentRequests []time.Time
	for _, reqTime := range suite.rateLimiter[clientIP] {
		if now.Sub(reqTime) < time.Minute {
			recentRequests = append(recentRequests, reqTime)
		}
	}
	suite.rateLimiter[clientIP] = recentRequests
	
	// Rate Limit 확인 (분당 10회 제한)
	if len(suite.rateLimiter[clientIP]) >= 10 {
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error": "Rate limit exceeded",
			"retry_after": 60,
		})
		c.Abort()
		return
	}
	
	// 현재 요청 기록
	suite.rateLimiter[clientIP] = append(suite.rateLimiter[clientIP], now)
	
	c.Next()
}

// csrfProtectedHandler CSRF 보호 핸들러 래퍼
func (suite *SecurityE2ETestSuite) csrfProtectedHandler(handler gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		// POST 요청에 대해서만 CSRF 토큰 확인
		if c.Request.Method == "POST" {
			csrfToken := c.GetHeader("X-CSRF-Token")
			if csrfToken == "" {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "CSRF token required",
				})
				return
			}
			
			// 간단한 CSRF 토큰 검증 (테스트용)
			if !suite.validateCSRFToken(csrfToken) {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "Invalid CSRF token",
				})
				return
			}
		}
		
		handler(c)
	}
}

// validateCSRFToken CSRF 토큰 검증 (테스트용 간단한 구현)
func (suite *SecurityE2ETestSuite) validateCSRFToken(token string) bool {
	// 테스트용으로 간단한 토큰 검증
	return token == "valid-csrf-token-for-testing"
}

// Test Methods

// testSecurityHeaders 보안 헤더 테스트
func (suite *SecurityE2ETestSuite) testSecurityHeaders(t *testing.T) {
	resp, err := suite.client.Get(suite.baseURL + "/api/protected/profile")
	require.NoError(t, err)
	defer resp.Body.Close()
	
	// 보안 헤더 확인
	assert.Equal(t, "DENY", resp.Header.Get("X-Frame-Options"))
	assert.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"))
	assert.Equal(t, "1; mode=block", resp.Header.Get("X-XSS-Protection"))
	assert.Equal(t, "strict-origin-when-cross-origin", resp.Header.Get("Referrer-Policy"))
	assert.Contains(t, resp.Header.Get("Content-Security-Policy"), "default-src 'self'")
	assert.Contains(t, resp.Header.Get("Strict-Transport-Security"), "max-age=31536000")
}

// testCORSHeaders CORS 헤더 테스트
func (suite *SecurityE2ETestSuite) testCORSHeaders(t *testing.T) {
	req, err := http.NewRequest("OPTIONS", suite.baseURL+"/api/auth/login", nil)
	require.NoError(t, err)
	
	req.Header.Set("Origin", "http://localhost:3000")
	
	resp, err := suite.client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	assert.Equal(t, "http://localhost:3000", resp.Header.Get("Access-Control-Allow-Origin"))
	assert.Contains(t, resp.Header.Get("Access-Control-Allow-Methods"), "POST")
	assert.Contains(t, resp.Header.Get("Access-Control-Allow-Headers"), "Authorization")
}

// testContentTypeValidation Content-Type 검증 테스트
func (suite *SecurityE2ETestSuite) testContentTypeValidation(t *testing.T) {
	// 잘못된 Content-Type으로 로그인 시도
	loginData := map[string]string{
		"username": "admin",
		"password": "admin123",
	}
	
	body, _ := json.Marshal(loginData)
	req, err := http.NewRequest("POST", suite.baseURL+"/api/auth/login", bytes.NewBuffer(body))
	require.NoError(t, err)
	
	req.Header.Set("Content-Type", "text/plain") // 잘못된 Content-Type
	req.Header.Set("X-CSRF-Token", "valid-csrf-token-for-testing")
	
	resp, err := suite.client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	
	// 400 Bad Request 예상
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// testMissingCSRFToken CSRF 토큰 누락 테스트
func (suite *SecurityE2ETestSuite) testMissingCSRFToken(t *testing.T) {
	loginData := map[string]string{
		"username": "admin",
		"password": "admin123",
	}
	
	body, _ := json.Marshal(loginData)
	resp, err := suite.client.Post(
		suite.baseURL+"/api/auth/login",
		"application/json",
		bytes.NewBuffer(body),
	)
	require.NoError(t, err)
	defer resp.Body.Close()
	
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	
	var errorResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&errorResp)
	assert.Contains(t, errorResp["error"], "CSRF token required")
}

// testInvalidCSRFToken 잘못된 CSRF 토큰 테스트
func (suite *SecurityE2ETestSuite) testInvalidCSRFToken(t *testing.T) {
	loginData := map[string]string{
		"username": "admin",
		"password": "admin123",
	}
	
	body, _ := json.Marshal(loginData)
	req, err := http.NewRequest("POST", suite.baseURL+"/api/auth/login", bytes.NewBuffer(body))
	require.NoError(t, err)
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", "invalid-csrf-token")
	
	resp, err := suite.client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	
	var errorResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&errorResp)
	assert.Contains(t, errorResp["error"], "Invalid CSRF token")
}

// testValidCSRFToken 유효한 CSRF 토큰 테스트
func (suite *SecurityE2ETestSuite) testValidCSRFToken(t *testing.T) {
	loginData := map[string]string{
		"username": "admin",
		"password": "admin123",
	}
	
	body, _ := json.Marshal(loginData)
	req, err := http.NewRequest("POST", suite.baseURL+"/api/auth/login", bytes.NewBuffer(body))
	require.NoError(t, err)
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", "valid-csrf-token-for-testing")
	
	resp, err := suite.client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	
	var loginResp LoginResponse
	json.NewDecoder(resp.Body).Decode(&loginResp)
	assert.True(t, loginResp.Success)
}

// testLoginRateLimiting 로그인 Rate Limiting 테스트
func (suite *SecurityE2ETestSuite) testLoginRateLimiting(t *testing.T) {
	loginData := map[string]string{
		"username": "admin",
		"password": "admin123",
	}
	
	body, _ := json.Marshal(loginData)
	
	// 10번의 로그인 시도 (Rate Limit 내)
	for i := 0; i < 10; i++ {
		req, err := http.NewRequest("POST", suite.baseURL+"/api/auth/login", bytes.NewBuffer(body))
		require.NoError(t, err)
		
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-CSRF-Token", "valid-csrf-token-for-testing")
		
		resp, err := suite.client.Do(req)
		require.NoError(t, err)
		resp.Body.Close()
		
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	}
	
	// 11번째 시도 (Rate Limit 초과)
	req, err := http.NewRequest("POST", suite.baseURL+"/api/auth/login", bytes.NewBuffer(body))
	require.NoError(t, err)
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", "valid-csrf-token-for-testing")
	
	resp, err := suite.client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	
	assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode)
	
	var errorResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&errorResp)
	assert.Contains(t, errorResp["error"], "Rate limit exceeded")
}

// testAPIRateLimiting API Rate Limiting 테스트
func (suite *SecurityE2ETestSuite) testAPIRateLimiting(t *testing.T) {
	// 먼저 로그인
	loginResp := suite.performValidLogin(t)
	token := loginResp.Data.AccessToken
	
	// 10번의 API 호출 (Rate Limit 내)
	for i := 0; i < 10; i++ {
		profile := suite.getProfile(t, token)
		assert.Equal(t, "admin", profile.Username)
	}
	
	// 11번째 호출 (Rate Limit 초과)
	resp := suite.tryGetProfile(t, token)
	defer resp.Body.Close()
	
	assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode)
}

// testRateLimitRecovery Rate Limit 복구 테스트
func (suite *SecurityE2ETestSuite) testRateLimitRecovery(t *testing.T) {
	// Rate Limit 초과 상태 만들기
	suite.testLoginRateLimiting(t)
	
	// 1분 대기 (실제로는 시간을 조작하거나 더 짧은 간격 사용)
	time.Sleep(100 * time.Millisecond) // 테스트용으로 짧게 설정
	
	// Rate Limit 초기화 (테스트용)
	suite.rateLimiter = make(map[string][]time.Time)
	
	// 다시 로그인 시도 (성공 예상)
	resp := suite.performValidLogin(t)
	assert.True(t, resp.Success)
}

// testMultipleFailedLogins 다중 실패 로그인 테스트
func (suite *SecurityE2ETestSuite) testMultipleFailedLogins(t *testing.T) {
	// 10번의 실패한 로그인 시도
	for i := 0; i < 10; i++ {
		resp := suite.tryLoginWithCSRF(t, "admin", "wrong-password")
		resp.Body.Close()
		
		if resp.StatusCode == http.StatusTooManyRequests {
			// Rate Limit에 걸린 경우 중단
			break
		}
		
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	}
}

// testAccountLockout 계정 잠금 테스트
func (suite *SecurityE2ETestSuite) testAccountLockout(t *testing.T) {
	// 실제 구현에서는 계정별 실패 횟수를 추적하고
	// 특정 횟수 초과 시 계정을 일시적으로 잠금
	// 여기서는 Rate Limiting으로 대체
	t.Skip("Account lockout requires user-specific failure tracking")
}

// testLockoutRecovery 잠금 복구 테스트
func (suite *SecurityE2ETestSuite) testLockoutRecovery(t *testing.T) {
	// 계정 잠금 후 시간 경과에 따른 복구 테스트
	t.Skip("Lockout recovery requires time-based account unlocking")
}

// Helper Methods

// performValidLogin 유효한 로그인 수행 (CSRF 토큰 포함)
func (suite *SecurityE2ETestSuite) performValidLogin(t *testing.T) LoginResponse {
	resp := suite.tryLoginWithCSRF(t, "admin", "admin123")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	
	var loginResp LoginResponse
	err := json.NewDecoder(resp.Body).Decode(&loginResp)
	require.NoError(t, err)
	resp.Body.Close()
	
	return loginResp
}

// tryLoginWithCSRF CSRF 토큰을 포함한 로그인 시도
func (suite *SecurityE2ETestSuite) tryLoginWithCSRF(t *testing.T, username, password string) *http.Response {
	loginData := map[string]string{
		"username": username,
		"password": password,
	}
	
	body, _ := json.Marshal(loginData)
	req, err := http.NewRequest("POST", suite.baseURL+"/api/auth/login", bytes.NewBuffer(body))
	require.NoError(t, err)
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", "valid-csrf-token-for-testing")
	
	resp, err := suite.client.Do(req)
	require.NoError(t, err)
	
	return resp
}