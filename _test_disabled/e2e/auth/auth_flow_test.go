// Package auth provides end-to-end tests for authentication flows
package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aicli/aicli-web/internal/auth"
	"github.com/aicli/aicli-web/internal/config"
)


// TestBasicAuthenticationFlow 기본 인증 플로우 E2E 테스트
func TestBasicAuthenticationFlow(t *testing.T) {
	suite := &AuthE2ETestSuite{}
	suite.setupAuthE2ETest(t)
	defer suite.teardownAuthE2ETest(t)
	
	t.Run("Complete Login Logout Flow", suite.testCompleteLoginLogoutFlow)
	t.Run("Invalid Credentials", suite.testInvalidCredentials)
	t.Run("Multiple Login Sessions", suite.testMultipleLoginSessions)
}

// TestJWTTokenManagement JWT 토큰 관리 E2E 테스트
func TestJWTTokenManagement(t *testing.T) {
	suite := &AuthE2ETestSuite{}
	suite.setupAuthE2ETest(t)
	defer suite.teardownAuthE2ETest(t)
	
	t.Run("Token Refresh Flow", suite.testTokenRefreshFlow)
	t.Run("Expired Token Handling", suite.testExpiredTokenHandling)
	t.Run("Invalid Token Handling", suite.testInvalidTokenHandling)
}

// TestSessionManagement 세션 관리 E2E 테스트
func TestSessionManagement(t *testing.T) {
	suite := &AuthE2ETestSuite{}
	suite.setupAuthE2ETest(t)
	defer suite.teardownAuthE2ETest(t)
	
	t.Run("Session Timeout", suite.testSessionTimeout)
	t.Run("Token Blacklist", suite.testTokenBlacklist)
	t.Run("Concurrent Sessions", suite.testConcurrentSessions)
}

// setupAuthE2ETest E2E 테스트 환경 설정
func (suite *AuthE2ETestSuite) setupAuthE2ETest(t *testing.T) {
	t.Log("Setting up Auth E2E test environment...")
	
	// JWT Manager 초기화
	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret:               "test-secret-key-for-e2e-testing",
			AccessTokenExpiry:    15 * time.Minute,
			RefreshTokenExpiry:   24 * time.Hour,
		},
	}
	
	var err error
	suite.jwtManager, err = auth.NewJWTManagerFromConfig(cfg)
	require.NoError(t, err)
	
	// Blacklist 초기화
	suite.blacklist = auth.NewBlacklist()
	
	// 테스트 서버 시작
	gin.SetMode(gin.TestMode)
	suite.server = suite.startAuthTestServer(t)
	suite.client = &http.Client{Timeout: 30 * time.Second}
	suite.baseURL = suite.server.URL
	
	// 테스트 사용자 데이터 초기화
	suite.testUsers = map[string]string{
		"admin": "admin123",
		"user":  "user123",
		"test":  "test123",
	}
	suite.validTokens = make(map[string]string)
	
	t.Log("Auth E2E test environment setup completed")
}

// teardownAuthE2ETest E2E 테스트 환경 정리
func (suite *AuthE2ETestSuite) teardownAuthE2ETest(t *testing.T) {
	t.Log("Cleaning up Auth E2E test environment...")
	
	if suite.server != nil {
		suite.server.Close()
	}
	
	t.Log("Auth E2E test environment cleanup completed")
}

// startAuthTestServer 인증 테스트 서버 시작
func (suite *AuthE2ETestSuite) startAuthTestServer(t *testing.T) *httptest.Server {
	router := gin.New()
	router.Use(gin.Recovery())
	
	// 인증 API 라우트 설정
	authGroup := router.Group("/api/auth")
	{
		authGroup.POST("/login", suite.mockLoginHandler)
		authGroup.POST("/refresh", suite.mockRefreshHandler)
		authGroup.POST("/logout", suite.mockLogoutHandler)
	}
	
	// 보호된 리소스 라우트 (인증 테스트용)
	protectedGroup := router.Group("/api/protected")
	protectedGroup.Use(suite.mockAuthMiddleware)
	{
		protectedGroup.GET("/profile", suite.mockGetProfileHandler)
		protectedGroup.GET("/admin", suite.mockAdminOnlyHandler)
	}
	
	return httptest.NewServer(router)
}

// testCompleteLoginLogoutFlow 완전한 로그인-로그아웃 플로우 테스트
func (suite *AuthE2ETestSuite) testCompleteLoginLogoutFlow(t *testing.T) {
	// Phase 1: 로그인
	loginResp := suite.performLogin(t, "admin", "admin123")
	
	assert.True(t, loginResp.Success)
	assert.NotEmpty(t, loginResp.Data.AccessToken)
	assert.NotEmpty(t, loginResp.Data.RefreshToken)
	assert.Equal(t, "Bearer", loginResp.Data.TokenType)
	assert.Greater(t, loginResp.Data.ExpiresIn, 0)
	
	accessToken := loginResp.Data.AccessToken
	
	// Phase 2: 인증된 리소스 접근
	profile := suite.getProfile(t, accessToken)
	assert.Equal(t, "admin", profile.Username)
	assert.Equal(t, "admin@example.com", profile.Email)
	
	// Phase 3: 로그아웃
	suite.performLogout(t, accessToken)
	
	// Phase 4: 로그아웃 후 리소스 접근 시도 (401 예상)
	resp := suite.tryGetProfile(t, accessToken)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// testInvalidCredentials 잘못된 인증 정보 테스트
func (suite *AuthE2ETestSuite) testInvalidCredentials(t *testing.T) {
	// 잘못된 사용자명
	resp := suite.tryLogin(t, "invalid_user", "admin123")
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	
	// 잘못된 비밀번호
	resp = suite.tryLogin(t, "admin", "wrong_password")
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	
	// 빈 인증 정보
	resp = suite.tryLogin(t, "", "")
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// testMultipleLoginSessions 다중 로그인 세션 테스트
func (suite *AuthE2ETestSuite) testMultipleLoginSessions(t *testing.T) {
	// 첫 번째 로그인
	login1 := suite.performLogin(t, "admin", "admin123")
	assert.True(t, login1.Success)
	
	// 두 번째 로그인 (같은 사용자)
	login2 := suite.performLogin(t, "admin", "admin123")
	assert.True(t, login2.Success)
	
	// 두 토큰 모두 유효해야 함
	profile1 := suite.getProfile(t, login1.Data.AccessToken)
	assert.Equal(t, "admin", profile1.Username)
	
	profile2 := suite.getProfile(t, login2.Data.AccessToken)
	assert.Equal(t, "admin", profile2.Username)
	
	// 첫 번째 세션 로그아웃
	suite.performLogout(t, login1.Data.AccessToken)
	
	// 첫 번째 토큰은 무효, 두 번째는 여전히 유효
	resp1 := suite.tryGetProfile(t, login1.Data.AccessToken)
	assert.Equal(t, http.StatusUnauthorized, resp1.StatusCode)
	
	profile2Again := suite.getProfile(t, login2.Data.AccessToken)
	assert.Equal(t, "admin", profile2Again.Username)
}

// testTokenRefreshFlow 토큰 갱신 플로우 테스트
func (suite *AuthE2ETestSuite) testTokenRefreshFlow(t *testing.T) {
	// 로그인
	loginResp := suite.performLogin(t, "user", "user123")
	refreshToken := loginResp.Data.RefreshToken
	
	// 토큰 갱신
	refreshResp := suite.performTokenRefresh(t, refreshToken)
	
	assert.True(t, refreshResp.Success)
	assert.NotEmpty(t, refreshResp.Data.AccessToken)
	assert.NotEqual(t, loginResp.Data.AccessToken, refreshResp.Data.AccessToken)
	
	// 새 토큰으로 리소스 접근
	profile := suite.getProfile(t, refreshResp.Data.AccessToken)
	assert.Equal(t, "user", profile.Username)
}

// testExpiredTokenHandling 만료된 토큰 처리 테스트
func (suite *AuthE2ETestSuite) testExpiredTokenHandling(t *testing.T) {
	// 만료된 토큰 생성 (테스트용)
	expiredToken, err := suite.jwtManager.GenerateTokenWithExpiry(
		"user-test", "test", "test@example.com", "user", 
		auth.AccessToken, -1*time.Hour, // 1시간 전 만료
	)
	require.NoError(t, err)
	
	// 만료된 토큰으로 접근 시도
	resp := suite.tryGetProfile(t, expiredToken)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// testInvalidTokenHandling 잘못된 토큰 처리 테스트
func (suite *AuthE2ETestSuite) testInvalidTokenHandling(t *testing.T) {
	// 잘못된 형식의 토큰
	resp := suite.tryGetProfile(t, "invalid.token.format")
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	
	// 빈 토큰
	resp = suite.tryGetProfileWithHeader(t, "")
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	
	// Bearer 없는 토큰
	resp = suite.tryGetProfileWithHeader(t, "some-token")
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// testSessionTimeout 세션 타임아웃 테스트
func (suite *AuthE2ETestSuite) testSessionTimeout(t *testing.T) {
	// 짧은 만료 시간으로 토큰 생성
	shortLivedToken, err := suite.jwtManager.GenerateTokenWithExpiry(
		"user-timeout", "timeout", "timeout@example.com", "user",
		auth.AccessToken, 2*time.Second, // 2초 후 만료
	)
	require.NoError(t, err)
	
	// 즉시 접근 (성공 예상)
	profile := suite.getProfile(t, shortLivedToken)
	assert.Equal(t, "timeout", profile.Username)
	
	// 3초 대기 후 접근 (실패 예상)
	time.Sleep(3 * time.Second)
	resp := suite.tryGetProfile(t, shortLivedToken)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// testTokenBlacklist 토큰 블랙리스트 테스트
func (suite *AuthE2ETestSuite) testTokenBlacklist(t *testing.T) {
	// 로그인
	loginResp := suite.performLogin(t, "test", "test123")
	accessToken := loginResp.Data.AccessToken
	
	// 정상 접근
	profile := suite.getProfile(t, accessToken)
	assert.Equal(t, "test", profile.Username)
	
	// 로그아웃 (토큰이 블랙리스트에 추가됨)
	suite.performLogout(t, accessToken)
	
	// 블랙리스트된 토큰으로 접근 시도
	resp := suite.tryGetProfile(t, accessToken)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// testConcurrentSessions 동시 세션 테스트
func (suite *AuthE2ETestSuite) testConcurrentSessions(t *testing.T) {
	// 여러 사용자의 동시 로그인
	adminLogin := suite.performLogin(t, "admin", "admin123")
	userLogin := suite.performLogin(t, "user", "user123")
	testLogin := suite.performLogin(t, "test", "test123")
	
	// 모든 세션이 독립적으로 작동해야 함
	adminProfile := suite.getProfile(t, adminLogin.Data.AccessToken)
	assert.Equal(t, "admin", adminProfile.Username)
	
	userProfile := suite.getProfile(t, userLogin.Data.AccessToken)
	assert.Equal(t, "user", userProfile.Username)
	
	testProfile := suite.getProfile(t, testLogin.Data.AccessToken)
	assert.Equal(t, "test", testProfile.Username)
	
	// 한 사용자 로그아웃이 다른 사용자에게 영향 없어야 함
	suite.performLogout(t, adminLogin.Data.AccessToken)
	
	// admin은 접근 불가
	resp := suite.tryGetProfile(t, adminLogin.Data.AccessToken)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	
	// 다른 사용자들은 계속 접근 가능
	userProfileAgain := suite.getProfile(t, userLogin.Data.AccessToken)
	assert.Equal(t, "user", userProfileAgain.Username)
}

// 헬퍼 메서드들은 auth_helpers.go 파일에서 계속...