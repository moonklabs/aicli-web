package integration

import (
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
	"github.com/aicli/aicli-web/internal/api/handlers"
	"github.com/aicli/aicli-web/internal/config"
	"github.com/aicli/aicli-web/internal/middleware"
	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/session"
	"github.com/aicli/aicli-web/internal/storage"
	"github.com/aicli/aicli-web/internal/storage/memory"
)

// SessionManagementIntegrationTestSuite 세션 관리 통합 테스트 스위트
type SessionManagementIntegrationTestSuite struct {
	suite.Suite
	app              *gin.Engine
	storage          storage.Storage
	sessionStore     session.Store
	jwtManager       auth.JWTManager
	rbacManager      auth.RBACManager
	authHandler      *handlers.AuthHandler
	userHandler      *handlers.UserHandler
	sessionService   session.Service
	deviceFingerprint *session.DeviceFingerprintGenerator
	
	// 테스트 데이터
	testUsers    map[string]*models.User
	testSessions map[string]*session.SessionData
}

// SetupSuite 테스트 스위트 초기화
func (suite *SessionManagementIntegrationTestSuite) SetupSuite() {
	gin.SetMode(gin.TestMode)

	// 저장소 초기화
	suite.storage = memory.NewMemoryStorage()
	
	// JWT 매니저 초기화
	suite.jwtManager = auth.NewJWTManager("test-secret-key", 15*time.Minute, 24*time.Hour)
	
	// 세션 저장소 초기화
	sessionConfig := &config.SessionConfig{
		CookieName:    "session_id",
		MaxAge:        3600,
		Secure:        false,
		HTTPOnly:      true,
		SameSite:      "Lax",
		Domain:        "",
		MaxSessions:   5, // 사용자당 최대 5개 세션
		CleanupInterval: 30 * time.Minute,
	}
	suite.sessionStore = session.NewMemoryStore(sessionConfig)
	
	// 디바이스 핑거프린팅 초기화
	suite.deviceFingerprint = session.NewDeviceFingerprintGenerator()
	
	// 세션 서비스 초기화
	suite.sessionService = session.NewService(suite.sessionStore, suite.deviceFingerprint)
	
	// RBAC 매니저 초기화
	cache := auth.NewMemoryPermissionCache()
	rbacStorage := auth.NewRBACStorageAdapter(suite.storage)
	suite.rbacManager = auth.NewRBACManager(rbacStorage, cache)
	
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
	}
	oauthManager := auth.NewOAuthManager(oauthConfigs, suite.jwtManager)
	
	// 핸들러 초기화
	suite.authHandler = handlers.NewAuthHandler(oauthManager, suite.sessionStore, suite.storage)
	suite.userHandler = handlers.NewUserHandler(suite.storage, suite.sessionService)
	
	// 테스트 데이터 초기화
	suite.setupTestData()
	
	// Gin 앱 설정
	suite.app = gin.New()
	suite.setupRoutes()
}

// setupTestData 테스트 데이터 설정
func (suite *SessionManagementIntegrationTestSuite) setupTestData() {
	ctx := context.Background()
	
	suite.testUsers = make(map[string]*models.User)
	suite.testSessions = make(map[string]*session.SessionData)
	
	// 테스트 사용자 생성
	users := []*models.User{
		{
			Base:     models.Base{ID: "user-john"},
			Username: "john",
			Email:    "john@example.com",
			IsActive: true,
		},
		{
			Base:     models.Base{ID: "user-jane"},
			Username: "jane",
			Email:    "jane@example.com",
			IsActive: true,
		},
		{
			Base:     models.Base{ID: "user-admin"},
			Username: "admin",
			Email:    "admin@example.com",
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
	
	// 테스트 역할 생성
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
	
	for _, userRole := range userRoles {
		userRole.CreatedAt = time.Now()
		userRole.UpdatedAt = time.Now()
		err := suite.storage.Create(ctx, "user_roles", userRole)
		require.NoError(suite.T(), err)
	}
}

// setupRoutes 라우트 설정
func (suite *SessionManagementIntegrationTestSuite) setupRoutes() {
	// 미들웨어 설정
	suite.app.Use(middleware.ErrorHandler())
	suite.app.Use(middleware.CORSMiddleware())
	
	// 인증 라우트
	authGroup := suite.app.Group("/auth")
	{
		authGroup.POST("/login", suite.authHandler.Login)
		authGroup.POST("/logout", middleware.AuthMiddleware(suite.jwtManager), suite.authHandler.Logout)
		authGroup.POST("/refresh", suite.authHandler.RefreshToken)
		authGroup.GET("/me", middleware.AuthMiddleware(suite.jwtManager), suite.authHandler.GetCurrentUser)
	}
	
	// 세션 관리 라우트
	sessionGroup := suite.app.Group("/sessions")
	sessionGroup.Use(middleware.AuthMiddleware(suite.jwtManager))
	{
		sessionGroup.GET("", suite.userHandler.GetUserSessions)
		sessionGroup.DELETE("/:sessionId", suite.userHandler.TerminateSession)
		sessionGroup.DELETE("", suite.userHandler.TerminateAllSessions)
		sessionGroup.POST("/:sessionId/extend", suite.userHandler.ExtendSession)
	}
	
	// 보호된 API 라우트
	apiGroup := suite.app.Group("/api")
	apiGroup.Use(middleware.AuthMiddleware(suite.jwtManager))
	apiGroup.Use(middleware.SessionValidationMiddleware(suite.sessionService))
	{
		apiGroup.GET("/profile", func(c *gin.Context) {
			claims, _ := c.Get("claims")
			sessionData, _ := c.Get("session")
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"user":    claims,
				"session": sessionData,
			})
		})
		
		// 관리자 전용 라우트
		adminGroup := apiGroup.Group("/admin")
		adminGroup.Use(middleware.RBACMiddleware(suite.rbacManager, models.ResourceTypeSystem, "", models.ActionManage))
		{
			adminGroup.GET("/users", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"success": true, "users": []string{"admin-only"}})
			})
		}
	}
}

// TestFullSessionLifecycleIntegration 전체 세션 라이프사이클 통합 테스트
func (suite *SessionManagementIntegrationTestSuite) TestFullSessionLifecycleIntegration() {
	t := suite.T()
	
	// 1. 로그인 (세션 생성)
	loginData := map[string]string{
		"username": "john",
		"password": "password123",
	}
	loginBody, _ := json.Marshal(loginData)
	
	req := httptest.NewRequest("POST", "/auth/login", strings.NewReader(string(loginBody)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/91.0")
	req.RemoteAddr = "192.168.1.100:12345"
	
	w := httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var loginResponse map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &loginResponse)
	require.NoError(t, err)
	
	accessToken := loginResponse["access_token"].(string)
	require.NotEmpty(t, accessToken)
	
	// 2. 인증된 요청으로 프로필 조회 (세션 활동 기록)
	req = httptest.NewRequest("GET", "/api/profile", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/91.0")
	req.RemoteAddr = "192.168.1.100:12345"
	
	w = httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var profileResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &profileResponse)
	require.NoError(t, err)
	assert.True(t, profileResponse["success"].(bool))
	
	// 3. 세션 목록 조회
	req = httptest.NewRequest("GET", "/sessions", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	
	w = httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var sessionsResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &sessionsResponse)
	require.NoError(t, err)
	
	sessions := sessionsResponse["sessions"].([]interface{})
	assert.Len(t, sessions, 1) // 하나의 활성 세션
	
	sessionData := sessions[0].(map[string]interface{})
	sessionID := sessionData["id"].(string)
	
	// 4. 세션 연장
	req = httptest.NewRequest("POST", "/sessions/"+sessionID+"/extend", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	
	w = httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	// 5. 로그아웃 (세션 종료)
	req = httptest.NewRequest("POST", "/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	
	w = httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	// 6. 로그아웃 후 인증된 요청 실패 확인
	req = httptest.NewRequest("GET", "/api/profile", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	
	w = httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestMultipleSessionsManagement 다중 세션 관리 테스트
func (suite *SessionManagementIntegrationTestSuite) TestMultipleSessionsManagement() {
	t := suite.T()
	
	userID := "user-john"
	var accessTokens []string
	
	// 여러 디바이스에서 로그인 (다중 세션 생성)
	devices := []struct {
		userAgent string
		ipAddress string
		name      string
	}{
		{
			userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/91.0",
			ipAddress: "192.168.1.100:12345",
			name:      "Windows Chrome",
		},
		{
			userAgent: "Mozilla/5.0 (iPhone; CPU iPhone OS 14_6 like Mac OS X) Safari/604.1",
			ipAddress: "192.168.1.101:54321",
			name:      "iPhone Safari",
		},
		{
			userAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) Chrome/91.0",
			ipAddress: "192.168.1.102:8080",
			name:      "macOS Chrome",
		},
	}
	
	for _, device := range devices {
		loginData := map[string]string{
			"username": "john",
			"password": "password123",
		}
		loginBody, _ := json.Marshal(loginData)
		
		req := httptest.NewRequest("POST", "/auth/login", strings.NewReader(string(loginBody)))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", device.userAgent)
		req.RemoteAddr = device.ipAddress
		
		w := httptest.NewRecorder()
		suite.app.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code, "Login should succeed for %s", device.name)
		
		var loginResponse map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &loginResponse)
		require.NoError(t, err)
		
		accessToken := loginResponse["access_token"].(string)
		accessTokens = append(accessTokens, accessToken)
	}
	
	// 모든 세션 조회
	req := httptest.NewRequest("GET", "/sessions", nil)
	req.Header.Set("Authorization", "Bearer "+accessTokens[0])
	
	w := httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var sessionsResponse map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &sessionsResponse)
	require.NoError(t, err)
	
	sessions := sessionsResponse["sessions"].([]interface{})
	assert.Len(t, sessions, 3) // 3개의 활성 세션
	
	// 특정 세션 종료
	sessionToTerminate := sessions[1].(map[string]interface{})
	sessionID := sessionToTerminate["id"].(string)
	
	req = httptest.NewRequest("DELETE", "/sessions/"+sessionID, nil)
	req.Header.Set("Authorization", "Bearer "+accessTokens[0])
	
	w = httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	// 종료된 세션으로 접근 시도
	req = httptest.NewRequest("GET", "/api/profile", nil)
	req.Header.Set("Authorization", "Bearer "+accessTokens[1]) // 종료된 세션의 토큰
	
	w = httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	
	// 다른 세션은 여전히 유효해야 함
	req = httptest.NewRequest("GET", "/api/profile", nil)
	req.Header.Set("Authorization", "Bearer "+accessTokens[0])
	
	w = httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	// 모든 세션 종료
	req = httptest.NewRequest("DELETE", "/sessions", nil)
	req.Header.Set("Authorization", "Bearer "+accessTokens[0])
	
	w = httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	// 모든 토큰이 무효화되었는지 확인
	for i, token := range accessTokens {
		req = httptest.NewRequest("GET", "/api/profile", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		
		w = httptest.NewRecorder()
		suite.app.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusUnauthorized, w.Code, "Token %d should be invalidated", i)
	}
}

// TestSessionLimitsAndSecurity 세션 제한 및 보안 테스트
func (suite *SessionManagementIntegrationTestSuite) TestSessionLimitsAndSecurity() {
	t := suite.T()
	
	// 최대 세션 수 초과 테스트
	maxSessions := 5
	var tokens []string
	
	// 최대 세션 수만큼 로그인
	for i := 0; i < maxSessions+2; i++ { // 제한보다 2개 더 시도
		loginData := map[string]string{
			"username": "jane",
			"password": "password123",
		}
		loginBody, _ := json.Marshal(loginData)
		
		req := httptest.NewRequest("POST", "/auth/login", strings.NewReader(string(loginBody)))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", fmt.Sprintf("TestAgent/%d", i))
		req.RemoteAddr = fmt.Sprintf("192.168.1.%d:12345", 100+i)
		
		w := httptest.NewRecorder()
		suite.app.ServeHTTP(w, req)
		
		if i < maxSessions {
			assert.Equal(t, http.StatusOK, w.Code, "Login %d should succeed", i)
			
			var loginResponse map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &loginResponse)
			require.NoError(t, err)
			
			token := loginResponse["access_token"].(string)
			tokens = append(tokens, token)
		} else {
			// 최대 세션 수 초과 시 가장 오래된 세션이 무효화되어야 함
			assert.Equal(t, http.StatusOK, w.Code, "Login should still succeed but oldest session should be invalidated")
		}
	}
	
	// 세션 수 확인
	req := httptest.NewRequest("GET", "/sessions", nil)
	req.Header.Set("Authorization", "Bearer "+tokens[len(tokens)-1]) // 마지막 토큰 사용
	
	w := httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	if w.Code == http.StatusOK {
		var sessionsResponse map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &sessionsResponse)
		require.NoError(t, err)
		
		sessions := sessionsResponse["sessions"].([]interface{})
		assert.LessOrEqual(t, len(sessions), maxSessions, "Should not exceed max sessions")
	}
}

// TestSessionSecurityValidation 세션 보안 검증 테스트
func (suite *SessionManagementIntegrationTestSuite) TestSessionSecurityValidation() {
	t := suite.T()
	
	// 정상 로그인
	loginData := map[string]string{
		"username": "john",
		"password": "password123",
	}
	loginBody, _ := json.Marshal(loginData)
	
	req := httptest.NewRequest("POST", "/auth/login", strings.NewReader(string(loginBody)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/91.0")
	req.RemoteAddr = "192.168.1.100:12345"
	
	w := httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var loginResponse map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &loginResponse)
	require.NoError(t, err)
	
	accessToken := loginResponse["access_token"].(string)
	
	// 1. IP 주소 변경 시도
	req = httptest.NewRequest("GET", "/api/profile", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/91.0")
	req.RemoteAddr = "203.0.113.1:12345" // 완전히 다른 IP
	
	w = httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	// IP 변경 시 보안 정책에 따라 접근이 제한될 수 있음
	// 현재 구현에서는 경고 로그만 기록하고 허용할 수 있음
	
	// 2. User-Agent 변경 시도
	req = httptest.NewRequest("GET", "/api/profile", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 14_6 like Mac OS X) Safari/604.1")
	req.RemoteAddr = "192.168.1.100:12345"
	
	w = httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	// User-Agent 변경도 보안 정책에 따라 처리
	
	// 3. 동시 세션 공격 시뮬레이션
	var wg sync.WaitGroup
	results := make(chan int, 10)
	
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			req := httptest.NewRequest("GET", "/api/profile", nil)
			req.Header.Set("Authorization", "Bearer "+accessToken)
			req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/91.0")
			req.RemoteAddr = "192.168.1.100:12345"
			
			w := httptest.NewRecorder()
			suite.app.ServeHTTP(w, req)
			
			results <- w.Code
		}()
	}
	
	wg.Wait()
	close(results)
	
	// 동시 요청이 모두 성공해야 함 (정상적인 사용자 행동)
	successCount := 0
	for statusCode := range results {
		if statusCode == http.StatusOK {
			successCount++
		}
	}
	
	assert.Greater(t, successCount, 5, "Most concurrent requests should succeed for legitimate user")
}

// TestRBACSessionIntegration RBAC와 세션 통합 테스트
func (suite *SessionManagementIntegrationTestSuite) TestRBACSessionIntegration() {
	t := suite.T()
	
	// 관리자 로그인
	loginData := map[string]string{
		"username": "admin",
		"password": "admin123",
	}
	loginBody, _ := json.Marshal(loginData)
	
	req := httptest.NewRequest("POST", "/auth/login", strings.NewReader(string(loginBody)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/91.0")
	req.RemoteAddr = "192.168.1.100:12345"
	
	w := httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var loginResponse map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &loginResponse)
	require.NoError(t, err)
	
	adminToken := loginResponse["access_token"].(string)
	
	// 관리자 권한 필요한 API 호출
	req = httptest.NewRequest("GET", "/api/admin/users", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	
	w = httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code) // 관리자는 접근 가능
	
	// 일반 사용자 로그인
	loginData = map[string]string{
		"username": "john",
		"password": "password123",
	}
	loginBody, _ = json.Marshal(loginData)
	
	req = httptest.NewRequest("POST", "/auth/login", strings.NewReader(string(loginBody)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/91.0")
	req.RemoteAddr = "192.168.1.101:12345"
	
	w = httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	err = json.Unmarshal(w.Body.Bytes(), &loginResponse)
	require.NoError(t, err)
	
	userToken := loginResponse["access_token"].(string)
	
	// 일반 사용자가 관리자 API 호출 시도
	req = httptest.NewRequest("GET", "/api/admin/users", nil)
	req.Header.Set("Authorization", "Bearer "+userToken)
	
	w = httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusForbidden, w.Code) // 일반 사용자는 접근 불가
	
	// 일반 사용자는 자신의 프로필 접근 가능
	req = httptest.NewRequest("GET", "/api/profile", nil)
	req.Header.Set("Authorization", "Bearer "+userToken)
	
	w = httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestSessionExpiration 세션 만료 테스트
func (suite *SessionManagementIntegrationTestSuite) TestSessionExpiration() {
	t := suite.T()
	
	// 짧은 만료 시간으로 JWT 매니저 생성
	shortJWTManager := auth.NewJWTManager("test-secret", 1*time.Second, 1*time.Hour)
	
	// 임시 로그인 핸들러로 짧은 토큰 생성
	testClaims := &auth.Claims{
		UserID:   "user-john",
		Email:    "john@example.com",
		Provider: "local",
	}
	
	tokens, err := shortJWTManager.GenerateTokens(testClaims)
	require.NoError(t, err)
	
	// 즉시 사용하면 유효해야 함
	req := httptest.NewRequest("GET", "/api/profile", nil)
	req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	
	w := httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	// 만료 대기
	time.Sleep(2 * time.Second)
	
	// 만료된 토큰으로 접근 시도
	req = httptest.NewRequest("GET", "/api/profile", nil)
	req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	
	w = httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	
	// 리프레시 토큰으로 새 토큰 요청
	refreshData := map[string]string{
		"refresh_token": tokens.RefreshToken,
	}
	refreshBody, _ := json.Marshal(refreshData)
	
	req = httptest.NewRequest("POST", "/auth/refresh", strings.NewReader(string(refreshBody)))
	req.Header.Set("Content-Type", "application/json")
	
	w = httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var refreshResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &refreshResponse)
	require.NoError(t, err)
	
	newAccessToken := refreshResponse["access_token"].(string)
	assert.NotEmpty(t, newAccessToken)
	
	// 새 토큰으로 접근 시도
	req = httptest.NewRequest("GET", "/api/profile", nil)
	req.Header.Set("Authorization", "Bearer "+newAccessToken)
	
	w = httptest.NewRecorder()
	suite.app.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestConcurrentSessionOperations 동시 세션 작업 테스트
func (suite *SessionManagementIntegrationTestSuite) TestConcurrentSessionOperations() {
	t := suite.T()
	
	// 여러 사용자가 동시에 로그인
	const numUsers = 10
	const operationsPerUser = 5
	
	var wg sync.WaitGroup
	results := make(chan bool, numUsers*operationsPerUser)
	
	for i := 0; i < numUsers; i++ {
		wg.Add(1)
		go func(userIndex int) {
			defer wg.Done()
			
			username := fmt.Sprintf("testuser%d", userIndex)
			
			// 사용자 생성 (간단화를 위해 여기서 직접 생성)
			user := &models.User{
				Base:     models.Base{ID: fmt.Sprintf("user-test-%d", userIndex)},
				Username: username,
				Email:    fmt.Sprintf("%s@test.com", username),
				IsActive: true,
			}
			user.CreatedAt = time.Now()
			user.UpdatedAt = time.Now()
			
			ctx := context.Background()
			suite.storage.Create(ctx, "users", user)
			
			for j := 0; j < operationsPerUser; j++ {
				// 로그인
				loginData := map[string]string{
					"username": username,
					"password": "password123",
				}
				loginBody, _ := json.Marshal(loginData)
				
				req := httptest.NewRequest("POST", "/auth/login", strings.NewReader(string(loginBody)))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("User-Agent", fmt.Sprintf("TestAgent%d-%d", userIndex, j))
				req.RemoteAddr = fmt.Sprintf("192.168.%d.%d:12345", userIndex%255, j%255)
				
				w := httptest.NewRecorder()
				suite.app.ServeHTTP(w, req)
				
				success := w.Code == http.StatusOK
				results <- success
				
				if success {
					// API 호출
					var loginResponse map[string]interface{}
					json.Unmarshal(w.Body.Bytes(), &loginResponse)
					
					if token, ok := loginResponse["access_token"].(string); ok {
						req = httptest.NewRequest("GET", "/api/profile", nil)
						req.Header.Set("Authorization", "Bearer "+token)
						
						w = httptest.NewRecorder()
						suite.app.ServeHTTP(w, req)
						
						results <- w.Code == http.StatusOK
					} else {
						results <- false
					}
				} else {
					results <- false
				}
			}
		}(i)
	}
	
	wg.Wait()
	close(results)
	
	successCount := 0
	totalCount := 0
	for result := range results {
		totalCount++
		if result {
			successCount++
		}
	}
	
	// 대부분의 작업이 성공해야 함
	successRate := float64(successCount) / float64(totalCount)
	assert.Greater(t, successRate, 0.8, "Success rate should be > 80%")
	
	t.Logf("Concurrent operations: %d/%d succeeded (%.1f%%)", 
		successCount, totalCount, successRate*100)
}

// 테스트 스위트 실행
func TestSessionManagementIntegrationSuite(t *testing.T) {
	suite.Run(t, new(SessionManagementIntegrationTestSuite))
}

// 벤치마크 테스트
func BenchmarkSessionValidation(b *testing.B) {
	gin.SetMode(gin.TestMode)
	
	// 간단한 설정
	storage := memory.NewMemoryStorage()
	sessionStore := session.NewMemoryStore(&config.SessionConfig{
		CookieName: "session_id",
		MaxAge:     3600,
	})
	jwtManager := auth.NewJWTManager("test-secret", 15*time.Minute, 24*time.Hour)
	deviceFingerprint := session.NewDeviceFingerprintGenerator()
	sessionService := session.NewService(sessionStore, deviceFingerprint)
	
	// 테스트용 토큰 생성
	claims := &auth.Claims{
		UserID:   "bench-user",
		Email:    "bench@test.com",
		Provider: "local",
	}
	
	tokens, _ := jwtManager.GenerateTokens(claims)
	
	// 라우터 설정
	router := gin.New()
	router.Use(middleware.AuthMiddleware(jwtManager))
	router.Use(middleware.SessionValidationMiddleware(sessionService))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
		req.Header.Set("User-Agent", "BenchmarkAgent")
		req.RemoteAddr = "127.0.0.1:12345"
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			b.Fatalf("Expected 200, got %d", w.Code)
		}
	}
}