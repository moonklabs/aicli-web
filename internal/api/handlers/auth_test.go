package handlers

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
	"github.com/aicli/aicli-web/internal/auth"
	"github.com/aicli/aicli-web/internal/config"
)

func setupTestRouter() (*gin.Engine, *AuthHandler) {
	gin.SetMode(gin.TestMode)
	
	// JWT 매니저와 블랙리스트 생성
	jwtManager := auth.NewJWTManager(
		"test-secret-key-for-testing-only",
		15*time.Minute,
		7*24*time.Hour,
	)
	blacklist := auth.NewBlacklist()
	
	// 테스트용 OAuth 설정 (빈 설정)
	// OAuth 관련 코드는 필요시 추가
	
	// 핸들러 생성
	authHandler := NewAuthHandler(jwtManager, blacklist)
	
	// 라우터 설정
	router := gin.New()
	v1 := router.Group("/api/v1")
	authGroup := v1.Group("/auth")
	{
		authGroup.POST("/login", authHandler.Login)
		authGroup.POST("/refresh", authHandler.Refresh)
		authGroup.POST("/logout", authHandler.Logout)
	}
	
	return router, authHandler
}

func TestLogin(t *testing.T) {
	router, _ := setupTestRouter()
	
	t.Run("성공적인 로그인", func(t *testing.T) {
		loginReq := LoginRequest{
			Username: "admin",
			Password: "admin123",
		}
		
		body, _ := json.Marshal(loginReq)
		req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.True(t, response["success"].(bool))
		data := response["data"].(map[string]interface{})
		assert.NotEmpty(t, data["access_token"])
		assert.NotEmpty(t, data["refresh_token"])
		assert.Equal(t, "Bearer", data["token_type"])
		assert.Equal(t, float64(config.DefaultAccessTokenExpiry.Seconds()), data["expires_in"])
	})
	
	t.Run("잘못된 자격증명", func(t *testing.T) {
		loginReq := LoginRequest{
			Username: "admin",
			Password: "wrongpassword",
		}
		
		body, _ := json.Marshal(loginReq)
		req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.False(t, response["success"].(bool))
		errorData := response["error"].(map[string]interface{})
		assert.Equal(t, "INVALID_CREDENTIALS", errorData["code"])
	})
	
	t.Run("잘못된 요청 본문", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBufferString("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusBadRequest, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.False(t, response["success"].(bool))
		errorData := response["error"].(map[string]interface{})
		assert.Equal(t, "INVALID_REQUEST", errorData["code"])
	})
	
	t.Run("필수 필드 누락", func(t *testing.T) {
		loginReq := LoginRequest{
			Username: "admin",
			// Password 누락
		}
		
		body, _ := json.Marshal(loginReq)
		req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestRefresh(t *testing.T) {
	router, authHandler := setupTestRouter()
	
	// 테스트용 리프레시 토큰 생성
	refreshToken, err := authHandler.jwtManager.GenerateToken("user123", "testuser", "user", auth.RefreshToken)
	require.NoError(t, err)
	
	t.Run("성공적인 토큰 갱신", func(t *testing.T) {
		refreshReq := RefreshRequest{
			RefreshToken: refreshToken,
		}
		
		body, _ := json.Marshal(refreshReq)
		req := httptest.NewRequest("POST", "/api/v1/auth/refresh", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.True(t, response["success"].(bool))
		data := response["data"].(map[string]interface{})
		assert.NotEmpty(t, data["access_token"])
		assert.Equal(t, "Bearer", data["token_type"])
	})
	
	t.Run("잘못된 리프레시 토큰", func(t *testing.T) {
		refreshReq := RefreshRequest{
			RefreshToken: "invalid-refresh-token",
		}
		
		body, _ := json.Marshal(refreshReq)
		req := httptest.NewRequest("POST", "/api/v1/auth/refresh", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.False(t, response["success"].(bool))
		errorData := response["error"].(map[string]interface{})
		assert.Equal(t, "INVALID_REFRESH_TOKEN", errorData["code"])
	})
	
	t.Run("블랙리스트된 리프레시 토큰", func(t *testing.T) {
		// 토큰을 블랙리스트에 추가
		authHandler.blacklist.Add(refreshToken, time.Now().Add(1*time.Hour))
		
		refreshReq := RefreshRequest{
			RefreshToken: refreshToken,
		}
		
		body, _ := json.Marshal(refreshReq)
		req := httptest.NewRequest("POST", "/api/v1/auth/refresh", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.False(t, response["success"].(bool))
		errorData := response["error"].(map[string]interface{})
		assert.Equal(t, "TOKEN_BLACKLISTED", errorData["code"])
		
		// 블랙리스트에서 제거 (다른 테스트에 영향 주지 않도록)
		authHandler.blacklist.Remove(refreshToken)
	})
}

func TestLogout(t *testing.T) {
	router, authHandler := setupTestRouter()
	
	// 테스트용 액세스 토큰 생성
	accessToken, err := authHandler.jwtManager.GenerateToken("user123", "testuser", "user", auth.AccessToken)
	require.NoError(t, err)
	
	t.Run("성공적인 로그아웃", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/auth/logout", nil)
		req.Header.Set("Authorization", "Bearer "+accessToken)
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.True(t, response["success"].(bool))
		assert.Equal(t, "Logged out successfully", response["message"])
		
		// 토큰이 블랙리스트에 추가되었는지 확인
		assert.True(t, authHandler.blacklist.IsBlacklisted(accessToken))
	})
	
	t.Run("Authorization 헤더 없음", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/auth/logout", nil)
		// Authorization 헤더 설정하지 않음
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusBadRequest, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.False(t, response["success"].(bool))
		errorData := response["error"].(map[string]interface{})
		assert.Equal(t, "INVALID_TOKEN", errorData["code"])
	})
	
	t.Run("잘못된 토큰", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/auth/logout", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.False(t, response["success"].(bool))
		errorData := response["error"].(map[string]interface{})
		assert.Equal(t, "INVALID_TOKEN", errorData["code"])
	})
}

func TestValidateUser(t *testing.T) {
	_, authHandler := setupTestRouter()
	
	tests := []struct {
		name     string
		username string
		password string
		expected bool
	}{
		{"유효한 관리자 자격증명", "admin", "admin123", true},
		{"유효한 사용자 자격증명", "user", "user123", true},
		{"유효한 테스트 자격증명", "test", "test123", true},
		{"잘못된 비밀번호", "admin", "wrongpass", false},
		{"존재하지 않는 사용자", "nonexistent", "password", false},
		{"빈 사용자명", "", "password", false},
		{"빈 비밀번호", "admin", "", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := authHandler.validateUser(tt.username, tt.password)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUserRoles(t *testing.T) {
	router, _ := setupTestRouter()
	
	t.Run("관리자 역할", func(t *testing.T) {
		loginReq := LoginRequest{
			Username: "admin",
			Password: "admin123",
		}
		
		body, _ := json.Marshal(loginReq)
		req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		// 토큰에서 클레임 확인
		data := response["data"].(map[string]interface{})
		accessToken := data["access_token"].(string)
		
		// JWT 매니저로 토큰 검증
		jwtManager := auth.NewJWTManager(
			"test-secret-key-for-testing-only",
			15*time.Minute,
			7*24*time.Hour,
		)
		claims, err := jwtManager.VerifyToken(accessToken)
		require.NoError(t, err)
		
		assert.Equal(t, "admin", claims.Role)
	})
	
	t.Run("일반 사용자 역할", func(t *testing.T) {
		loginReq := LoginRequest{
			Username: "user",
			Password: "user123",
		}
		
		body, _ := json.Marshal(loginReq)
		req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		// 토큰에서 클레임 확인
		data := response["data"].(map[string]interface{})
		accessToken := data["access_token"].(string)
		
		// JWT 매니저로 토큰 검증
		jwtManager := auth.NewJWTManager(
			"test-secret-key-for-testing-only",
			15*time.Minute,
			7*24*time.Hour,
		)
		claims, err := jwtManager.VerifyToken(accessToken)
		require.NoError(t, err)
		
		assert.Equal(t, "user", claims.Role)
	})
}