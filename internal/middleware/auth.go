package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"aicli-web/internal/auth"
)

// JWTAuth JWT 인증 미들웨어
func JWTAuth(jwtManager *auth.JWTManager, blacklist *auth.Blacklist) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Authorization 헤더 추출
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "MISSING_AUTH_HEADER",
					"message": "Authorization header is required",
				},
			})
			return
		}

		// Bearer 토큰 추출
		token, err := auth.ExtractTokenFromHeader(authHeader)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INVALID_AUTH_HEADER",
					"message": "Invalid authorization header format",
					"details": err.Error(),
				},
			})
			return
		}

		// 블랙리스트 확인
		if blacklist.IsBlacklisted(token) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "TOKEN_BLACKLISTED",
					"message": "Token has been revoked",
				},
			})
			return
		}

		// 토큰 검증
		claims, err := jwtManager.VerifyToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INVALID_TOKEN",
					"message": "Invalid or expired token",
					"details": err.Error(),
				},
			})
			return
		}

		// 클레임을 컨텍스트에 저장
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.UserName)
		c.Set("role", claims.Role)
		c.Set("claims", claims)

		c.Next()
	}
}

// RequireAuth 특정 라우트에 인증을 요구하는 헬퍼 함수
func RequireAuth(jwtManager *auth.JWTManager, blacklist *auth.Blacklist) gin.HandlerFunc {
	return JWTAuth(jwtManager, blacklist)
}

// RequireRole 특정 역할을 요구하는 미들웨어
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 컨텍스트에서 사용자 역할 확인
		userRole, exists := c.Get("role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "NO_ROLE_FOUND",
					"message": "User role not found in context",
				},
			})
			return
		}

		// 역할 확인
		userRoleStr, ok := userRole.(string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INVALID_ROLE_TYPE",
					"message": "Invalid role type in context",
				},
			})
			return
		}

		// 허용된 역할인지 확인
		allowed := false
		for _, role := range roles {
			if userRoleStr == role {
				allowed = true
				break
			}
		}

		if !allowed {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INSUFFICIENT_PERMISSIONS",
					"message": "You don't have permission to access this resource",
					"details": gin.H{
						"required_roles": roles,
						"user_role":      userRoleStr,
					},
				},
			})
			return
		}

		c.Next()
	}
}

// OptionalAuth 선택적 인증 미들웨어 (인증이 있으면 사용자 정보 설정, 없어도 통과)
func OptionalAuth(jwtManager *auth.JWTManager, blacklist *auth.Blacklist) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Authorization 헤더 추출
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// 인증 헤더가 없어도 통과
			c.Next()
			return
		}

		// Bearer 토큰 추출
		token, err := auth.ExtractTokenFromHeader(authHeader)
		if err != nil {
			// 잘못된 형식이어도 통과 (선택적 인증)
			c.Next()
			return
		}

		// 블랙리스트 확인
		if blacklist.IsBlacklisted(token) {
			// 블랙리스트에 있어도 통과 (선택적 인증)
			c.Next()
			return
		}

		// 토큰 검증
		claims, err := jwtManager.VerifyToken(token)
		if err != nil {
			// 검증 실패해도 통과 (선택적 인증)
			c.Next()
			return
		}

		// 유효한 토큰인 경우 클레임을 컨텍스트에 저장
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.UserName)
		c.Set("role", claims.Role)
		c.Set("claims", claims)
		c.Set("authenticated", true)

		c.Next()
	}
}

// GetUserID 컨텍스트에서 사용자 ID 추출 헬퍼 함수
func GetUserID(c *gin.Context) (string, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return "", false
	}
	userIDStr, ok := userID.(string)
	return userIDStr, ok
}

// GetUsername 컨텍스트에서 사용자명 추출 헬퍼 함수
func GetUsername(c *gin.Context) (string, bool) {
	username, exists := c.Get("username")
	if !exists {
		return "", false
	}
	usernameStr, ok := username.(string)
	return usernameStr, ok
}

// GetUserRole 컨텍스트에서 사용자 역할 추출 헬퍼 함수
func GetUserRole(c *gin.Context) (string, bool) {
	role, exists := c.Get("role")
	if !exists {
		return "", false
	}
	roleStr, ok := role.(string)
	return roleStr, ok
}

// IsAuthenticated 인증 여부 확인 헬퍼 함수
func IsAuthenticated(c *gin.Context) bool {
	authenticated, exists := c.Get("authenticated")
	if !exists {
		// authenticated 플래그가 없으면 user_id로 확인
		_, hasUserID := c.Get("user_id")
		return hasUserID
	}
	isAuth, ok := authenticated.(bool)
	return ok && isAuth
}