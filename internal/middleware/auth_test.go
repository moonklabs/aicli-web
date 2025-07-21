package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"aicli-web/internal/auth"
)

func setupTestMiddleware() (*gin.Engine, *auth.JWTManager, *auth.Blacklist) {
	gin.SetMode(gin.TestMode)
	
	jwtManager := auth.NewJWTManager(
		"test-secret-key",
		15*time.Minute,
		7*24*time.Hour,
	)
	blacklist := auth.NewBlacklist()
	
	router := gin.New()
	
	return router, jwtManager, blacklist
}

func TestJWTAuth(t *testing.T) {
	router, jwtManager, blacklist := setupTestMiddleware()
	
	// 보호된 엔드포인트 설정
	router.GET("/protected", JWTAuth(jwtManager, blacklist), func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		username, _ := c.Get("username")
		role, _ := c.Get("role")
		
		c.JSON(http.StatusOK, gin.H{
			"user_id":  userID,
			"username": username,
			"role":     role,
		})
	})
	
	t.Run("유효한 토큰", func(t *testing.T) {
		// 토큰 생성
		token, err := jwtManager.GenerateToken("user123", "testuser", "admin", auth.AccessToken)
		require.NoError(t, err)
		
		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		
		var response map[string]interface{}
		err = w.Result().Header.Get("Content-Type")
		assert.Contains(t, w.Result().Header.Get("Content-Type"), "application/json")
	})
	
	t.Run("Authorization 헤더 없음", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		// Authorization 헤더 설정하지 않음
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
	
	t.Run("잘못된 헤더 형식", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "InvalidFormat token")
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
	
	t.Run("만료된 토큰", func(t *testing.T) {
		// 이미 만료된 토큰 생성을 위한 임시 매니저
		tempManager := auth.NewJWTManager(
			"test-secret-key",
			-1*time.Hour, // 음수로 설정하여 즉시 만료
			7*24*time.Hour,
		)
		
		token, err := tempManager.GenerateToken("user123", "testuser", "admin", auth.AccessToken)
		require.NoError(t, err)
		
		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
	
	t.Run("블랙리스트된 토큰", func(t *testing.T) {
		// 토큰 생성
		token, err := jwtManager.GenerateToken("user123", "testuser", "admin", auth.AccessToken)
		require.NoError(t, err)
		
		// 블랙리스트에 추가
		blacklist.Add(token, time.Now().Add(1*time.Hour))
		
		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		
		// 정리
		blacklist.Remove(token)
	})
}

func TestRequireRole(t *testing.T) {
	router, jwtManager, blacklist := setupTestMiddleware()
	
	// 역할별 엔드포인트 설정
	adminGroup := router.Group("/admin")
	adminGroup.Use(JWTAuth(jwtManager, blacklist))
	adminGroup.Use(RequireRole("admin"))
	adminGroup.GET("/dashboard", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Admin dashboard"})
	})
	
	userGroup := router.Group("/user")
	userGroup.Use(JWTAuth(jwtManager, blacklist))
	userGroup.Use(RequireRole("user", "admin"))
	userGroup.GET("/profile", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "User profile"})
	})
	
	t.Run("관리자 역할로 관리자 엔드포인트 접근", func(t *testing.T) {
		token, err := jwtManager.GenerateToken("admin123", "admin", "admin", auth.AccessToken)
		require.NoError(t, err)
		
		req := httptest.NewRequest("GET", "/admin/dashboard", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
	})
	
	t.Run("사용자 역할로 관리자 엔드포인트 접근", func(t *testing.T) {
		token, err := jwtManager.GenerateToken("user123", "user", "user", auth.AccessToken)
		require.NoError(t, err)
		
		req := httptest.NewRequest("GET", "/admin/dashboard", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusForbidden, w.Code)
	})
	
	t.Run("사용자 역할로 사용자 엔드포인트 접근", func(t *testing.T) {
		token, err := jwtManager.GenerateToken("user123", "user", "user", auth.AccessToken)
		require.NoError(t, err)
		
		req := httptest.NewRequest("GET", "/user/profile", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
	})
	
	t.Run("관리자 역할로 사용자 엔드포인트 접근", func(t *testing.T) {
		token, err := jwtManager.GenerateToken("admin123", "admin", "admin", auth.AccessToken)
		require.NoError(t, err)
		
		req := httptest.NewRequest("GET", "/user/profile", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestOptionalAuth(t *testing.T) {
	router, jwtManager, blacklist := setupTestMiddleware()
	
	// 선택적 인증 엔드포인트
	router.GET("/optional", OptionalAuth(jwtManager, blacklist), func(c *gin.Context) {
		authenticated := IsAuthenticated(c)
		userID, _ := GetUserID(c)
		username, _ := GetUsername(c)
		
		c.JSON(http.StatusOK, gin.H{
			"authenticated": authenticated,
			"user_id":       userID,
			"username":      username,
		})
	})
	
	t.Run("토큰 없이 접근", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/optional", nil)
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		
		// Response 확인은 실제 JSON 파싱으로
	})
	
	t.Run("유효한 토큰으로 접근", func(t *testing.T) {
		token, err := jwtManager.GenerateToken("user123", "testuser", "user", auth.AccessToken)
		require.NoError(t, err)
		
		req := httptest.NewRequest("GET", "/optional", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
	})
	
	t.Run("잘못된 토큰으로 접근", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/optional", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code) // 선택적 인증이므로 통과
	})
}

func TestHelperFunctions(t *testing.T) {
	router, jwtManager, blacklist := setupTestMiddleware()
	
	router.GET("/test", JWTAuth(jwtManager, blacklist), func(c *gin.Context) {
		userID, hasUserID := GetUserID(c)
		username, hasUsername := GetUsername(c)
		role, hasRole := GetUserRole(c)
		isAuth := IsAuthenticated(c)
		
		c.JSON(http.StatusOK, gin.H{
			"user_id":       userID,
			"has_user_id":   hasUserID,
			"username":      username,
			"has_username":  hasUsername,
			"role":          role,
			"has_role":      hasRole,
			"authenticated": isAuth,
		})
	})
	
	t.Run("헬퍼 함수 테스트", func(t *testing.T) {
		token, err := jwtManager.GenerateToken("user123", "testuser", "admin", auth.AccessToken)
		require.NoError(t, err)
		
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
	})
}