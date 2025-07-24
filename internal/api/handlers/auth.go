package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/aicli/aicli-web/internal/auth"
	"github.com/aicli/aicli-web/internal/config"
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
	email := req.Username + "@example.com" // 임시 이메일
	accessToken, err := h.jwtManager.GenerateToken(userID, req.Username, email, role, auth.AccessToken)
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
	refreshToken, err := h.jwtManager.GenerateToken(userID, req.Username, email, role, auth.RefreshToken)
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

// OAuth 관련 구조체들

// OAuthLoginRequest OAuth 로그인 시작 요청
type OAuthLoginRequest struct {
	Provider auth.OAuthProvider `uri:"provider" binding:"required"`
}

// OAuthCallbackRequest OAuth 콜백 요청
type OAuthCallbackRequest struct {
	Provider auth.OAuthProvider `uri:"provider" binding:"required"`
	Code     string             `form:"code" binding:"required"`
	State    string             `form:"state" binding:"required"`
}

// OAuthLoginResponse OAuth 로그인 응답
type OAuthLoginResponse struct {
	AuthURL string `json:"auth_url"`
	State   string `json:"state"`
}

// OAuth 핸들러들

// OAuthLogin OAuth 로그인 시작
// @Summary OAuth 로그인 시작
// @Description OAuth 제공자를 통한 로그인을 시작합니다
// @Tags oauth
// @Produce json
// @Param provider path string true "OAuth 제공자" Enums(google,github)
// @Success 200 {object} map[string]interface{} "로그인 URL 생성 성공"
// @Failure 400 {object} map[string]interface{} "잘못된 요청"
// @Router /auth/oauth/{provider} [get]
func (h *AuthHandler) OAuthLogin(c *gin.Context, oauthManager auth.OAuthManager) {
	var req OAuthLoginRequest
	if err := c.ShouldBindUri(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_PROVIDER",
				"message": "Invalid OAuth provider",
				"details": err.Error(),
			},
		})
		return
	}

	// state 생성 (CSRF 보호)
	stateBytes := make([]byte, 32)
	_, err := rand.Read(stateBytes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate state"})
		return
	}
	state := base64.URLEncoding.EncodeToString(stateBytes)

	// OAuth 인증 URL 생성
	authURL, err := oauthManager.GetAuthURL(req.Provider, state)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": OAuthLoginResponse{
			AuthURL: authURL,
			State:   state,
		},
	})
}

// OAuthCallback OAuth 콜백 처리
// @Summary OAuth 콜백 처리
// @Description OAuth 제공자로부터의 콜백을 처리하여 JWT 토큰을 발급합니다
// @Tags oauth
// @Produce json
// @Param provider path string true "OAuth 제공자" Enums(google,github)
// @Param code query string true "인증 코드"
// @Param state query string true "상태 파라미터"
// @Success 200 {object} map[string]interface{} "로그인 성공"
// @Failure 400 {object} map[string]interface{} "잘못된 요청"
// @Failure 401 {object} map[string]interface{} "인증 실패"
// @Router /auth/oauth/{provider}/callback [get]
func (h *AuthHandler) OAuthCallback(c *gin.Context, oauthManager auth.OAuthManager) {
	var req OAuthCallbackRequest
	if err := c.ShouldBindUri(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_PROVIDER",
				"message": "Invalid OAuth provider",
				"details": err.Error(),
			},
		})
		return
	}

	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_CALLBACK",
				"message": "Invalid OAuth callback parameters",
				"details": err.Error(),
			},
		})
		return
	}

	// 인증 코드를 액세스 토큰으로 교환
	token, err := oauthManager.ExchangeCode(req.Provider, req.Code, req.State)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// 사용자 정보 가져오기
	userInfo, err := oauthManager.GetUserInfo(req.Provider, token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// TODO: 실제 환경에서는 DB에서 사용자 매핑 또는 생성 필요
	// 현재는 OAuth 정보로 내부 사용자 생성
	userID := fmt.Sprintf("oauth_%s_%s", userInfo.Provider, userInfo.ID)
	userName := userInfo.Name
	if userName == "" {
		userName = userInfo.Email
	}
	role := "user" // OAuth 사용자는 기본적으로 user 역할

	// JWT 토큰 생성 (기존 시스템과 동일)
	accessToken, err := h.jwtManager.GenerateToken(userID, userName, userInfo.Email, role, auth.AccessToken)
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

	refreshToken, err := h.jwtManager.GenerateToken(userID, userName, userInfo.Email, role, auth.RefreshToken)
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
		"user": gin.H{
			"id":       userID,
			"name":     userName,
			"email":    userInfo.Email,
			"picture":  userInfo.Picture,
			"provider": userInfo.Provider,
		},
	})
}

// generateSecureState 보안 state 파라미터 생성
func (h *AuthHandler) generateSecureState() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}