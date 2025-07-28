// Package auth provides helper functions for authentication E2E tests
package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/aicli/aicli-web/internal/auth"
	"github.com/aicli/aicli-web/internal/testutil"
)

// Mock Handlers

// mockLoginHandler 로그인 Mock 핸들러
func (suite *AuthE2ETestSuite) mockLoginHandler(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "Invalid request body",
				"details": err.Error(),
			},
		})
		return
	}
	
	// 사용자 검증
	validPassword, exists := suite.testUsers[req.Username]
	if !exists || validPassword != req.Password {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_CREDENTIALS",
				"message": "Invalid username or password",
			},
		})
		return
	}
	
	// 사용자 정보 설정
	userID := "user-" + req.Username
	email := req.Username + "@example.com"
	role := "user"
	if req.Username == "admin" {
		role = "admin"
	}
	
	// 토큰 생성
	accessToken, err := suite.jwtManager.GenerateToken(userID, req.Username, email, role, auth.AccessToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "TOKEN_GENERATION_ERROR",
				"message": "Failed to generate access token",
			},
		})
		return
	}
	
	refreshToken, err := suite.jwtManager.GenerateToken(userID, req.Username, email, role, auth.RefreshToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "TOKEN_GENERATION_ERROR",
				"message": "Failed to generate refresh token",
			},
		})
		return
	}
	
	// 토큰 저장 (테스트용)
	suite.validTokens[req.Username] = accessToken
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"access_token":  accessToken,
			"refresh_token": refreshToken,
			"token_type":    "Bearer",
			"expires_in":    int(time.Hour.Seconds()), // 1시간
		},
	})
}

// mockRefreshHandler 토큰 갱신 Mock 핸들러
func (suite *AuthE2ETestSuite) mockRefreshHandler(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "Invalid request body",
				"details": err.Error(),
			},
		})
		return
	}
	
	// 블랙리스트 확인
	if suite.blacklist.IsBlacklisted(req.RefreshToken) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "TOKEN_BLACKLISTED",
				"message": "Refresh token has been revoked",
			},
		})
		return
	}
	
	// 리프레시 토큰 검증 및 새 액세스 토큰 생성
	claims, err := suite.jwtManager.VerifyToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_REFRESH_TOKEN",
				"message": "Invalid or expired refresh token",
				"details": err.Error(),
			},
		})
		return
	}

	// 새 액세스 토큰 생성
	newAccessToken, err := suite.jwtManager.GenerateToken(claims.UserID, claims.UserName, claims.Email, claims.Role, auth.AccessToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "TOKEN_GENERATION_ERROR",
				"message": "Failed to generate new access token",
			},
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"access_token": newAccessToken,
			"token_type":   "Bearer",
			"expires_in":   int(time.Hour.Seconds()),
		},
	})
}

// mockLogoutHandler 로그아웃 Mock 핸들러
func (suite *AuthE2ETestSuite) mockLogoutHandler(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	token, err := auth.ExtractTokenFromHeader(authHeader)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_TOKEN",
				"message": "Invalid authorization header",
				"details": err.Error(),
			},
		})
		return
	}
	
	// 토큰 검증
	claims, err := suite.jwtManager.VerifyToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_TOKEN",
				"message": "Invalid or expired token",
			},
		})
		return
	}
	
	// 토큰을 블랙리스트에 추가
	suite.blacklist.Add(token, claims.ExpiresAt.Time)
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Logged out successfully",
	})
}

// mockAuthMiddleware 인증 미들웨어 Mock
func (suite *AuthE2ETestSuite) mockAuthMiddleware(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	token, err := auth.ExtractTokenFromHeader(authHeader)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid authorization header",
		})
		c.Abort()
		return
	}
	
	// 블랙리스트 확인
	if suite.blacklist.IsBlacklisted(token) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Token has been revoked",
		})
		c.Abort()
		return
	}
	
	// 토큰 검증
	claims, err := suite.jwtManager.VerifyToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid or expired token",
		})
		c.Abort()
		return
	}
	
	// 사용자 정보를 컨텍스트에 저장
	c.Set("user_id", claims.UserID)
	c.Set("username", claims.UserName)
	c.Set("email", claims.Email)
	c.Set("role", claims.Role)
	
	c.Next()
}

// mockGetProfileHandler 프로필 조회 Mock 핸들러
func (suite *AuthE2ETestSuite) mockGetProfileHandler(c *gin.Context) {
	username := c.GetString("username")
	email := c.GetString("email")
	role := c.GetString("role")
	
	c.JSON(http.StatusOK, ProfileResponse{
		Username: username,
		Email:    email,
		Role:     role,
	})
}

// mockAdminOnlyHandler 관리자 전용 Mock 핸들러
func (suite *AuthE2ETestSuite) mockAdminOnlyHandler(c *gin.Context) {
	role := c.GetString("role")
	if role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Admin access required",
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": "Admin access granted",
		"data":    "sensitive admin data",
	})
}

// Helper Methods for API Calls

// performLogin 로그인 수행
func (suite *AuthE2ETestSuite) performLogin(t *testing.T, username, password string) LoginResponse {
	resp := suite.tryLogin(t, username, password)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	
	var loginResp LoginResponse
	err := json.NewDecoder(resp.Body).Decode(&loginResp)
	require.NoError(t, err)
	resp.Body.Close()
	
	return loginResp
}

// tryLogin 로그인 시도 (응답 검증 없음)
func (suite *AuthE2ETestSuite) tryLogin(t *testing.T, username, password string) *http.Response {
	loginData := map[string]string{
		"username": username,
		"password": password,
	}
	
	body, _ := json.Marshal(loginData)
	resp, err := suite.client.Post(
		suite.baseURL+"/api/auth/login",
		"application/json",
		bytes.NewBuffer(body),
	)
	require.NoError(t, err)
	
	return resp
}

// performTokenRefresh 토큰 갱신 수행
func (suite *AuthE2ETestSuite) performTokenRefresh(t *testing.T, refreshToken string) RefreshResponse {
	refreshData := map[string]string{
		"refresh_token": refreshToken,
	}
	
	body, _ := json.Marshal(refreshData)
	resp, err := suite.client.Post(
		suite.baseURL+"/api/auth/refresh",
		"application/json",
		bytes.NewBuffer(body),
	)
	require.NoError(t, err)
	defer resp.Body.Close()
	
	require.Equal(t, http.StatusOK, resp.StatusCode)
	
	var refreshResp RefreshResponse
	err = json.NewDecoder(resp.Body).Decode(&refreshResp)
	require.NoError(t, err)
	
	return refreshResp
}

// performLogout 로그아웃 수행
func (suite *AuthE2ETestSuite) performLogout(t *testing.T, accessToken string) {
	req, err := http.NewRequest("POST", suite.baseURL+"/api/auth/logout", nil)
	require.NoError(t, err)
	
	req.Header.Set("Authorization", "Bearer "+accessToken)
	
	resp, err := suite.client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

// getProfile 프로필 조회 (성공 예상)
func (suite *AuthE2ETestSuite) getProfile(t *testing.T, accessToken string) ProfileResponse {
	resp := suite.tryGetProfile(t, accessToken)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	
	var profile ProfileResponse
	err := json.NewDecoder(resp.Body).Decode(&profile)
	require.NoError(t, err)
	resp.Body.Close()
	
	return profile
}

// tryGetProfile 프로필 조회 시도 (응답 검증 없음)
func (suite *AuthE2ETestSuite) tryGetProfile(t *testing.T, accessToken string) *http.Response {
	return suite.tryGetProfileWithHeader(t, "Bearer "+accessToken)
}

// tryGetProfileWithHeader 헤더로 프로필 조회 시도
func (suite *AuthE2ETestSuite) tryGetProfileWithHeader(t *testing.T, authHeader string) *http.Response {
	req, err := http.NewRequest("GET", suite.baseURL+"/api/protected/profile", nil)
	require.NoError(t, err)
	
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	
	resp, err := suite.client.Do(req)
	require.NoError(t, err)
	
	return resp
}

// eventually 조건이 만족될 때까지 대기 (테스트 유틸리티)
func (suite *AuthE2ETestSuite) eventually(t *testing.T, timeout time.Duration, condition func() bool, msgAndArgs ...interface{}) {
	deadline := time.Now().Add(timeout)
	
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	
	// 최종 확인
	if !condition() {
		if len(msgAndArgs) > 0 {
			require.Fail(t, "Condition not met within timeout", msgAndArgs...)
		} else {
			require.Fail(t, "Condition not met within timeout")
		}
	}
}

// generateTestUserID 테스트 사용자 ID 생성
func generateTestUserID() string {
	return "test-user-" + testutil.GenerateRandomID()
}

// assertValidJWT JWT 토큰 유효성 검증
func (suite *AuthE2ETestSuite) assertValidJWT(t *testing.T, token string) *auth.Claims {
	require.NotEmpty(t, token)
	
	// JWT 형식 확인 (3개 부분으로 구성)
	parts := strings.Split(token, ".")
	require.Len(t, parts, 3, "JWT should have 3 parts")
	
	// 토큰 검증
	claims, err := suite.jwtManager.VerifyToken(token)
	require.NoError(t, err)
	require.NotNil(t, claims)
	
	// 기본 클레임 확인
	require.NotEmpty(t, claims.UserID)
	require.NotEmpty(t, claims.UserName)
	require.NotEmpty(t, claims.Email)
	require.NotEmpty(t, claims.Role)
	require.True(t, claims.ExpiresAt.After(time.Now()))
	
	return claims
}

// assertErrorResponse 에러 응답 검증
func assertErrorResponse(t *testing.T, resp *http.Response, expectedCode string) {
	var errorResp map[string]interface{}
	err := json.NewDecoder(resp.Body).Decode(&errorResp)
	require.NoError(t, err)
	
	require.False(t, errorResp["success"].(bool))
	require.NotNil(t, errorResp["error"])
	
	errorData := errorResp["error"].(map[string]interface{})
	require.Equal(t, expectedCode, errorData["code"])
	require.NotEmpty(t, errorData["message"])
}