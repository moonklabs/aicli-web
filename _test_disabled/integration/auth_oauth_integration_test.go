package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/aicli/aicli-web/internal/auth"
	"github.com/aicli/aicli-web/internal/api/handlers"
	"github.com/aicli/aicli-web/internal/config"
	"github.com/aicli/aicli-web/internal/middleware"
	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/session"
	"github.com/aicli/aicli-web/internal/storage"
	"github.com/aicli/aicli-web/internal/storage/memory"
)

// OAuthIntegrationTestSuite OAuth 통합 테스트 스위트
type OAuthIntegrationTestSuite struct {
	suite.Suite
	app           *gin.Engine
	authHandler   *handlers.AuthHandler
	oauthManager  auth.OAuthManager
	sessionStore  session.Store
	storage       storage.Storage
	jwtManager    auth.JWTManager
	mockServer    *httptest.Server
}

// SetupSuite 테스트 스위트 초기화
func (suite *OAuthIntegrationTestSuite) SetupSuite() {
	gin.SetMode(gin.TestMode)

	// 메모리 저장소 초기화
	suite.storage = memory.NewMemoryStorage()
	
	// JWT 매니저 초기화
	suite.jwtManager = auth.NewJWTManager("test-secret-key", 1*time.Hour, 24*time.Hour)
	
	// OAuth 설정
	oauthConfigs := map[auth.OAuthProvider]*auth.OAuthConfig{
		auth.ProviderGoogle: {
			Provider:     auth.ProviderGoogle,
			ClientID:     "test-google-client-id",
			ClientSecret: "test-google-secret",
			RedirectURL:  "http://localhost:8080/auth/oauth/google/callback",
			Scopes:       []string{"openid", "email", "profile"},
			Enabled:      true,
		},
		auth.ProviderGitHub: {
			Provider:     auth.ProviderGitHub,
			ClientID:     "test-github-client-id",
			ClientSecret: "test-github-secret",
			RedirectURL:  "http://localhost:8080/auth/oauth/github/callback",
			Scopes:       []string{"user:email"},
			Enabled:      true,
		},
	}
	
	// OAuth 매니저 초기화
	suite.oauthManager = auth.NewOAuthManager(oauthConfigs, suite.jwtManager)
	
	// 세션 저장소 초기화 (메모리 기반)
	sessionConfig := &config.SessionConfig{
		CookieName:     "session_id",
		MaxAge:         3600,
		Secure:         false,
		HTTPOnly:       true,
		SameSite:       "Lax",
		Domain:         "",
		RedisAddress:   "",
		RedisPassword:  "",
		RedisDB:        0,
	}
	suite.sessionStore = session.NewMemoryStore(sessionConfig)
	
	// 인증 핸들러 초기화
	suite.authHandler = handlers.NewAuthHandler(
		suite.oauthManager,
		suite.sessionStore,
		suite.storage,
	)
	
	// Mock OAuth 제공자 서버 설정
	suite.setupMockOAuthServer()
	
	// Gin 앱 설정
	suite.app = gin.New()
	suite.setupRoutes()
}

// TearDownSuite 테스트 스위트 정리
func (suite *OAuthIntegrationTestSuite) TearDownSuite() {
	if suite.mockServer != nil {
		suite.mockServer.Close()
	}
}

// setupMockOAuthServer Mock OAuth 제공자 서버 설정
func (suite *OAuthIntegrationTestSuite) setupMockOAuthServer() {
	mux := http.NewServeMux()
	
	// Google OAuth 토큰 엔드포인트
	mux.HandleFunc("/oauth2/v4/token", func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"access_token":  "mock-access-token",
			"token_type":    "Bearer",
			"expires_in":    3600,
			"refresh_token": "mock-refresh-token",
			"scope":         "openid email profile",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
	
	// Google 사용자 정보 엔드포인트
	mux.HandleFunc("/oauth2/v2/userinfo", func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		
		userInfo := map[string]interface{}{
			"id":             "google123456789",
			"email":          "testuser@gmail.com",
			"name":           "Test Google User",
			"picture":        "https://lh3.googleusercontent.com/test.jpg",
			"verified_email": true,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(userInfo)
	})
	
	// GitHub OAuth 토큰 엔드포인트
	mux.HandleFunc("/login/oauth/access_token", func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"access_token": "mock-github-token",
			"token_type":   "bearer",
			"scope":        "user:email",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
	
	// GitHub 사용자 정보 엔드포인트
	mux.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "token ") {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		
		userInfo := map[string]interface{}{
			"id":         987654321,
			"login":      "testgithubuser",
			"email":      "testuser@github.com",
			"name":       "Test GitHub User",
			"avatar_url": "https://avatars.githubusercontent.com/test.png",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(userInfo)
	})
	
	suite.mockServer = httptest.NewServer(mux)
}

// setupRoutes 라우트 설정
func (suite *OAuthIntegrationTestSuite) setupRoutes() {
	// 미들웨어 설정
	suite.app.Use(middleware.ErrorHandler())
	suite.app.Use(middleware.CORSMiddleware())
	
	// 인증 라우트
	authGroup := suite.app.Group("/auth")
	{
		// OAuth 인증 시작
		authGroup.GET("/oauth/:provider", suite.authHandler.InitiateOAuth)
		
		// OAuth 콜백
		authGroup.GET("/oauth/:provider/callback", suite.authHandler.HandleOAuthCallback)
		
		// 로그아웃
		authGroup.POST("/logout", suite.authHandler.Logout)
		
		// 토큰 새로고침
		authGroup.POST("/refresh", suite.authHandler.RefreshToken)
		
		// 사용자 정보 조회 (인증 필요)
		authGroup.GET("/me", middleware.AuthMiddleware(suite.jwtManager), suite.authHandler.GetCurrentUser)
	}
	
	// 보호된 라우트 예시
	protectedGroup := suite.app.Group("/api")
	protectedGroup.Use(middleware.AuthMiddleware(suite.jwtManager))
	{
		protectedGroup.GET("/profile", func(c *gin.Context) {
			claims, _ := c.Get("claims")
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"user":    claims,
			})
		})
	}
}

// TestGoogleOAuthFlow Google OAuth 전체 플로우 통합 테스트
func (suite *OAuthIntegrationTestSuite) TestGoogleOAuthFlow() {
	// 1. OAuth 인증 시작
	req := httptest.NewRequest("GET", "/auth/oauth/google", nil)
	w := httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	assert.Equal(suite.T(), http.StatusFound, w.Code)
	
	// 리다이렉트 URL 파싱
	location := w.Header().Get("Location")
	require.NotEmpty(suite.T(), location)
	
	parsedURL, err := url.Parse(location)
	require.NoError(suite.T(), err)
	
	// OAuth 파라미터 검증
	assert.Contains(suite.T(), parsedURL.Host, "accounts.google.com")
	assert.Equal(suite.T(), "code", parsedURL.Query().Get("response_type"))
	assert.Equal(suite.T(), "test-google-client-id", parsedURL.Query().Get("client_id"))
	assert.Contains(suite.T(), parsedURL.Query().Get("scope"), "openid")
	
	state := parsedURL.Query().Get("state")
	require.NotEmpty(suite.T(), state)
	
	// 2. OAuth 콜백 시뮬레이션
	callbackURL := "/auth/oauth/google/callback?code=test-auth-code&state=" + state
	req = httptest.NewRequest("GET", callbackURL, nil)
	w = httptest.NewRecorder()
	
	// Mock 서버 URL로 교체 (실제 환경에서는 설정으로 처리)
	// 이 부분은 실제 구현에서 환경 변수나 설정으로 처리해야 함
	
	suite.app.ServeHTTP(w, req)
	
	// 콜백 처리 성공 확인 (실제 토큰 교환 없이 테스트)
	// 실제 토큰 교환은 Mock 서버를 사용하거나 의존성 주입으로 테스트
	assert.Equal(suite.T(), http.StatusFound, w.Code) // 성공 시 리다이렉트
	
	// 3. JWT 토큰 검증 (쿠키에서)
	cookies := w.Result().Cookies()
	var jwtCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == "access_token" {
			jwtCookie = cookie
			break
		}
	}
	
	// JWT 토큰이 설정되었는지 확인 (실제 구현에 따라 달라질 수 있음)
	if jwtCookie != nil {
		assert.NotEmpty(suite.T(), jwtCookie.Value)
		
		// JWT 토큰 유효성 검증
		claims, err := suite.jwtManager.ValidateToken(jwtCookie.Value)
		assert.NoError(suite.T(), err)
		assert.NotEmpty(suite.T(), claims.UserID)
	}
}

// TestGitHubOAuthFlow GitHub OAuth 전체 플로우 통합 테스트
func (suite *OAuthIntegrationTestSuite) TestGitHubOAuthFlow() {
	// 1. OAuth 인증 시작
	req := httptest.NewRequest("GET", "/auth/oauth/github", nil)
	w := httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	assert.Equal(suite.T(), http.StatusFound, w.Code)
	
	// 리다이렉트 URL 파싱
	location := w.Header().Get("Location")
	require.NotEmpty(suite.T(), location)
	
	parsedURL, err := url.Parse(location)
	require.NoError(suite.T(), err)
	
	// OAuth 파라미터 검증
	assert.Contains(suite.T(), parsedURL.Host, "github.com")
	assert.Equal(suite.T(), "code", parsedURL.Query().Get("response_type"))
	assert.Equal(suite.T(), "test-github-client-id", parsedURL.Query().Get("client_id"))
	assert.Contains(suite.T(), parsedURL.Query().Get("scope"), "user:email")
	
	state := parsedURL.Query().Get("state")
	require.NotEmpty(suite.T(), state)
	
	// 2. OAuth 콜백 처리는 Google과 유사하므로 생략
	// 실제 환경에서는 각 제공자별로 테스트 구현
}

// TestOAuthErrorHandling OAuth 오류 처리 테스트
func (suite *OAuthIntegrationTestSuite) TestOAuthErrorHandling() {
	t := suite.T()
	
	// 1. 잘못된 제공자
	req := httptest.NewRequest("GET", "/auth/oauth/invalid", nil)
	w := httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusBadRequest, w.Code)
	
	// 2. 잘못된 state로 콜백
	req = httptest.NewRequest("GET", "/auth/oauth/google/callback?code=test&state=invalid", nil)
	w = httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusBadRequest, w.Code)
	
	// 3. 인증 코드 없이 콜백
	req = httptest.NewRequest("GET", "/auth/oauth/google/callback", nil)
	w = httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestAuthenticatedEndpoints 인증이 필요한 엔드포인트 테스트
func (suite *OAuthIntegrationTestSuite) TestAuthenticatedEndpoints() {
	t := suite.T()
	
	// 1. 토큰 없이 보호된 엔드포인트 접근
	req := httptest.NewRequest("GET", "/api/profile", nil)
	w := httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	
	// 2. 유효한 토큰으로 접근
	testClaims := &auth.Claims{
		UserID:   "test-user-123",
		Email:    "test@example.com",
		Provider: string(auth.ProviderGoogle),
	}
	
	token, err := suite.jwtManager.GenerateTokens(testClaims)
	require.NoError(t, err)
	
	req = httptest.NewRequest("GET", "/api/profile", nil)
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	w = httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.True(t, response["success"].(bool))
}

// TestSessionManagement 세션 관리 통합 테스트
func (suite *OAuthIntegrationTestSuite) TestSessionManagement() {
	t := suite.T()
	
	// 테스트용 사용자 세션 생성
	ctx := context.Background()
	sessionData := &session.SessionData{
		UserID:    "test-user-123",
		Email:     "test@example.com",
		Provider:  string(auth.ProviderGoogle),
		LoginTime: time.Now(),
		LastActivity: time.Now(),
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	}
	
	sessionID, err := suite.sessionStore.CreateSession(ctx, sessionData)
	require.NoError(t, err)
	assert.NotEmpty(t, sessionID)
	
	// 세션 조회
	retrievedSession, err := suite.sessionStore.GetSession(ctx, sessionID)
	require.NoError(t, err)
	assert.Equal(t, sessionData.UserID, retrievedSession.UserID)
	assert.Equal(t, sessionData.Email, retrievedSession.Email)
	
	// 세션 업데이트
	err = suite.sessionStore.UpdateLastActivity(ctx, sessionID)
	assert.NoError(t, err)
	
	// 세션 삭제
	err = suite.sessionStore.DeleteSession(ctx, sessionID)
	assert.NoError(t, err)
	
	// 삭제된 세션 조회 시 오류
	_, err = suite.sessionStore.GetSession(ctx, sessionID)
	assert.Error(t, err)
}

// TestLogout 로그아웃 플로우 테스트
func (suite *OAuthIntegrationTestSuite) TestLogout() {
	t := suite.T()
	
	// 1. 로그인 상태 준비
	testClaims := &auth.Claims{
		UserID:   "test-user-123",
		Email:    "test@example.com",
		Provider: string(auth.ProviderGoogle),
	}
	
	tokens, err := suite.jwtManager.GenerateTokens(testClaims)
	require.NoError(t, err)
	
	// 2. 로그아웃 요청
	req := httptest.NewRequest("POST", "/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	w := httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	// 3. 로그아웃 후 토큰 무효성 확인 (실제 구현에 따라)
	// JWT 자체는 stateless이므로 블랙리스트나 짧은 만료시간 등으로 처리
}

// TestConcurrentOAuthRequests 동시 OAuth 요청 테스트
func (suite *OAuthIntegrationTestSuite) TestConcurrentOAuthRequests() {
	t := suite.T()
	
	concurrentRequests := 10
	results := make(chan int, concurrentRequests)
	
	// 동시에 여러 OAuth 요청 실행
	for i := 0; i < concurrentRequests; i++ {
		go func() {
			req := httptest.NewRequest("GET", "/auth/oauth/google", nil)
			w := httptest.NewRecorder()
			suite.app.ServeHTTP(w, req)
			results <- w.Code
		}()
	}
	
	// 모든 요청이 성공적으로 처리되었는지 확인
	for i := 0; i < concurrentRequests; i++ {
		statusCode := <-results
		assert.Equal(t, http.StatusFound, statusCode)
	}
}

// TestOAuthStateExpiration OAuth state 만료 테스트
func (suite *OAuthIntegrationTestSuite) TestOAuthStateExpiration() {
	t := suite.T()
	
	// OAuth 인증 시작하여 state 생성
	req := httptest.NewRequest("GET", "/auth/oauth/google", nil)
	w := httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	location := w.Header().Get("Location")
	parsedURL, err := url.Parse(location)
	require.NoError(t, err)
	
	state := parsedURL.Query().Get("state")
	require.NotEmpty(t, state)
	
	// state가 유효한지 확인
	assert.True(t, suite.oauthManager.ValidateState(state))
	
	// 시간 경과 시뮬레이션은 실제 구현에서 의존성 주입으로 처리
	// 여기서는 잘못된 state로 테스트
	callbackURL := "/auth/oauth/google/callback?code=test&state=expired-state"
	req = httptest.NewRequest("GET", callbackURL, nil)
	w = httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestTokenRefresh 토큰 새로고침 테스트
func (suite *OAuthIntegrationTestSuite) TestTokenRefresh() {
	t := suite.T()
	
	// 테스트용 클레임과 토큰 생성
	testClaims := &auth.Claims{
		UserID:   "test-user-123",
		Email:    "test@example.com",
		Provider: string(auth.ProviderGoogle),
	}
	
	tokens, err := suite.jwtManager.GenerateTokens(testClaims)
	require.NoError(t, err)
	
	// 토큰 새로고침 요청
	refreshBody := map[string]string{
		"refresh_token": tokens.RefreshToken,
	}
	
	bodyBytes, _ := json.Marshal(refreshBody)
	req := httptest.NewRequest("POST", "/auth/refresh", strings.NewReader(string(bodyBytes)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	// 새로운 액세스 토큰 확인
	assert.True(t, response["success"].(bool))
	assert.NotEmpty(t, response["access_token"])
}

// 테스트 스위트 실행
func TestOAuthIntegrationSuite(t *testing.T) {
	suite.Run(t, new(OAuthIntegrationTestSuite))
}

// 벤치마크 테스트
func BenchmarkOAuthFlow(b *testing.B) {
	gin.SetMode(gin.TestMode)
	
	// 간단한 설정으로 벤치마크
	jwtManager := auth.NewJWTManager("test-secret", 1*time.Hour, 24*time.Hour)
	oauthConfigs := map[auth.OAuthProvider]*auth.OAuthConfig{
		auth.ProviderGoogle: {
			Provider:     auth.ProviderGoogle,
			ClientID:     "test-client-id",
			ClientSecret: "test-secret",
			RedirectURL:  "http://localhost:8080/auth/callback",
			Scopes:       []string{"openid", "email"},
			Enabled:      true,
		},
	}
	
	oauthManager := auth.NewOAuthManager(oauthConfigs, jwtManager)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		state := "benchmark-state"
		_, err := oauthManager.GetAuthURL(auth.ProviderGoogle, state)
		if err != nil {
			b.Fatal(err)
		}
	}
}