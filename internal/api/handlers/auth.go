package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"aicli-web/internal/auth"
	"aicli-web/internal/config"
)

// LoginRequest 로그인 요청 구조체
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse 로그인 응답 구조체
type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

// RefreshRequest 토큰 갱신 요청 구조체
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// RefreshResponse 토큰 갱신 응답 구조체
type RefreshResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// AuthHandler 인증 핸들러
type AuthHandler struct {
	jwtManager *auth.JWTManager
	blacklist  *auth.Blacklist
}

// NewAuthHandler 새로운 인증 핸들러 생성
func NewAuthHandler(jwtManager *auth.JWTManager, blacklist *auth.Blacklist) *AuthHandler {
	return &AuthHandler{
		jwtManager: jwtManager,
		blacklist:  blacklist,
	}
}

// Login 로그인 처리
// @Summary 사용자 로그인
// @Description 사용자 자격증명으로 로그인하여 JWT 토큰을 받습니다
// @Tags auth
// @Accept json
// @Produce json
// @Param body body LoginRequest true "로그인 요청"
// @Success 200 {object} map[string]interface{} "로그인 성공"
// @Failure 400 {object} map[string]interface{} "잘못된 요청"
// @Failure 401 {object} map[string]interface{} "인증 실패"
// @Router /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
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

	// TODO: 실제 환경에서는 데이터베이스에서 사용자 검증 필요
	// 임시로 하드코딩된 사용자 정보 사용
	if !h.validateUser(req.Username, req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_CREDENTIALS",
				"message": "Invalid username or password",
			},
		})
		return
	}

	// 사용자 정보 설정 (임시)
	userID := "user-" + req.Username
	role := "user"
	if req.Username == "admin" {
		role = "admin"
	}

	// 액세스 토큰 생성
	accessToken, err := h.jwtManager.GenerateToken(userID, req.Username, role, auth.AccessToken)
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

	// 리프레시 토큰 생성
	refreshToken, err := h.jwtManager.GenerateToken(userID, req.Username, role, auth.RefreshToken)
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

	// 응답
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": LoginResponse{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			TokenType:    "Bearer",
			ExpiresIn:    int(config.DefaultAccessTokenExpiry.Seconds()),
		},
	})
}

// Refresh 토큰 갱신
// @Summary 액세스 토큰 갱신
// @Description 리프레시 토큰을 사용하여 새 액세스 토큰을 받습니다
// @Tags auth
// @Accept json
// @Produce json
// @Param body body RefreshRequest true "토큰 갱신 요청"
// @Success 200 {object} map[string]interface{} "토큰 갱신 성공"
// @Failure 400 {object} map[string]interface{} "잘못된 요청"
// @Failure 401 {object} map[string]interface{} "인증 실패"
// @Router /auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req RefreshRequest
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
	if h.blacklist.IsBlacklisted(req.RefreshToken) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "TOKEN_BLACKLISTED",
				"message": "Refresh token has been revoked",
			},
		})
		return
	}

	// 새 액세스 토큰 생성
	newAccessToken, err := h.jwtManager.RefreshAccessToken(req.RefreshToken)
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

	// 응답
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": RefreshResponse{
			AccessToken: newAccessToken,
			TokenType:   "Bearer",
			ExpiresIn:   int(config.DefaultAccessTokenExpiry.Seconds()),
		},
	})
}

// Logout 로그아웃 처리
// @Summary 사용자 로그아웃
// @Description 현재 액세스 토큰을 무효화합니다
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "로그아웃 성공"
// @Failure 400 {object} map[string]interface{} "잘못된 요청"
// @Failure 401 {object} map[string]interface{} "인증 실패"
// @Router /auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	// Authorization 헤더에서 토큰 추출
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
	claims, err := h.jwtManager.VerifyToken(token)
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
	h.blacklist.Add(token, claims.ExpiresAt.Time)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Logged out successfully",
	})
}

// validateUser 사용자 검증 (임시 구현)
func (h *AuthHandler) validateUser(username, password string) bool {
	// TODO: 실제 환경에서는 데이터베이스에서 사용자 정보 조회
	// 임시로 하드코딩된 사용자 정보 사용
	validUsers := map[string]string{
		"admin": "admin123",
		"user":  "user123",
		"test":  "test123",
	}

	validPassword, exists := validUsers[username]
	return exists && validPassword == password
}