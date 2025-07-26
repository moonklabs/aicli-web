package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/aicli/aicli-web/internal/auth"
	"github.com/aicli/aicli-web/internal/models"
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

// IsOAuthUser OAuth를 통해 로그인한 사용자인지 확인 헬퍼 함수
func IsOAuthUser(c *gin.Context) bool {
	userID, exists := GetUserID(c)
	if !exists {
		return false
	}
	// OAuth 사용자 ID는 "oauth_" 접두사로 시작
	return strings.HasPrefix(userID, "oauth_")
}

// GetOAuthProvider OAuth 사용자의 제공자 정보 추출
func GetOAuthProvider(c *gin.Context) (string, bool) {
	userID, exists := GetUserID(c)
	if !exists || !IsOAuthUser(c) {
		return "", false
	}
	// oauth_google_123456 → google
	parts := strings.Split(userID, "_")
	if len(parts) >= 2 {
		return parts[1], true
	}
	return "", false
}

// RBAC 관련 미들웨어들

// RBACManagerInterface RBAC 매니저 인터페이스
type RBACManagerInterface interface {
	CheckPermission(ctx context.Context, req *models.CheckPermissionRequest) (*models.CheckPermissionResponse, error)
	ComputeUserPermissionMatrix(ctx context.Context, userID string) (*models.UserPermissionMatrix, error)
}

// RequirePermission RBAC 기반 권한 확인 미들웨어
func RequirePermission(rbacManager RBACManagerInterface, resourceType models.ResourceType, action models.ActionType) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 사용자 ID 추출
		userID, exists := GetUserID(c)
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "AUTHENTICATION_REQUIRED",
					"message": "Authentication required for this resource",
				},
			})
			return
		}

		// 리소스 ID 추출 (URL 파라미터에서)
		resourceID := c.Param("id")
		if resourceID == "" {
			resourceID = "*" // 전체 리소스 타입에 대한 권한
		}

		// 권한 확인 요청 생성
		req := &models.CheckPermissionRequest{
			UserID:       userID,
			ResourceType: resourceType,
			ResourceID:   resourceID,
			Action:       action,
			Attributes:   extractRequestAttributes(c),
		}

		// 권한 확인
		ctx := context.Background()
		response, err := rbacManager.CheckPermission(ctx, req)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "PERMISSION_CHECK_FAILED",
					"message": "Failed to check permissions",
					"details": err.Error(),
				},
			})
			return
		}

		// 권한 확인 결과
		if !response.Allowed {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INSUFFICIENT_PERMISSIONS",
					"message": "You don't have permission to perform this action",
					"details": gin.H{
						"required_permission": gin.H{
							"resource_type": resourceType,
							"resource_id":   resourceID,
							"action":        action,
						},
						"decision": response.Decision,
						"evaluation": response.Evaluation,
					},
				},
			})
			return
		}

		// 권한 확인 성공 - 컨텍스트에 권한 정보 저장
		c.Set("permission_decision", response.Decision)
		c.Set("permission_evaluation", response.Evaluation)

		c.Next()
	}
}

// RequireResourceOwnership 리소스 소유권 확인 미들웨어
func RequireResourceOwnership(rbacManager RBACManagerInterface, resourceType models.ResourceType) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := GetUserID(c)
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "AUTHENTICATION_REQUIRED",
					"message": "Authentication required for ownership verification",
				},
			})
			return
		}

		resourceID := c.Param("id")
		if resourceID == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "MISSING_RESOURCE_ID",
					"message": "Resource ID is required for ownership verification",
				},
			})
			return
		}

		// 소유권 확인을 위한 특별한 권한 체크
		req := &models.CheckPermissionRequest{
			UserID:       userID,
			ResourceType: resourceType,
			ResourceID:   resourceID,
			Action:       models.ActionManage, // 관리 권한으로 소유권 확인
			Attributes: map[string]string{
				"ownership_check": "true",
				"owner_id":        userID,
			},
		}

		ctx := context.Background()
		response, err := rbacManager.CheckPermission(ctx, req)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "OWNERSHIP_CHECK_FAILED",
					"message": "Failed to verify resource ownership",
					"details": err.Error(),
				},
			})
			return
		}

		if !response.Allowed {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "NOT_RESOURCE_OWNER",
					"message": "You are not the owner of this resource",
					"details": gin.H{
						"resource_type": resourceType,
						"resource_id":   resourceID,
						"user_id":       userID,
					},
				},
			})
			return
		}

		c.Set("is_resource_owner", true)
		c.Set("verified_resource_id", resourceID)

		c.Next()
	}
}

// RequireAnyPermission 여러 권한 중 하나라도 만족하면 통과하는 미들웨어
func RequireAnyPermission(rbacManager RBACManagerInterface, permissions ...PermissionRequirement) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := GetUserID(c)
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "AUTHENTICATION_REQUIRED",
					"message": "Authentication required",
				},
			})
			return
		}

		ctx := context.Background()
		attributes := extractRequestAttributes(c)
		
		// 모든 권한을 시도해보고 하나라도 통과하면 성공
		for _, perm := range permissions {
			resourceID := c.Param("id")
			if resourceID == "" {
				resourceID = "*"
			}

			req := &models.CheckPermissionRequest{
				UserID:       userID,
				ResourceType: perm.ResourceType,
				ResourceID:   resourceID,
				Action:       perm.Action,
				Attributes:   attributes,
			}

			response, err := rbacManager.CheckPermission(ctx, req)
			if err != nil {
				continue // 에러 발생한 권한은 건너뛰기
			}

			if response.Allowed {
				// 권한 확인 성공
				c.Set("permission_decision", response.Decision)
				c.Set("permission_evaluation", response.Evaluation)
				c.Set("matched_permission", perm)
				c.Next()
				return
			}
		}

		// 모든 권한 확인 실패
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INSUFFICIENT_PERMISSIONS",
				"message": "You don't have any of the required permissions",
				"details": gin.H{
					"required_permissions": permissions,
				},
			},
		})
	}
}

// RequireRoleAdvanced RBAC 기반 고급 역할 확인 미들웨어
func RequireRoleAdvanced(rbacManager RBACManagerInterface, roleNames ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := GetUserID(c)
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "AUTHENTICATION_REQUIRED",
					"message": "Authentication required",
				},
			})
			return
		}

		ctx := context.Background()
		
		// 사용자의 권한 매트릭스 계산
		matrix, err := rbacManager.ComputeUserPermissionMatrix(ctx, userID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "ROLE_CHECK_FAILED",
					"message": "Failed to check user roles",
					"details": err.Error(),
				},
			})
			return
		}

		// 요구되는 역할 중 하나를 사용자가 가지고 있는지 확인
		hasRequiredRole := false
		allUserRoles := append(matrix.DirectRoles, matrix.InheritedRoles...)
		allUserRoles = append(allUserRoles, matrix.GroupRoles...)

		// TODO: 역할 이름으로 역할 ID를 조회하는 기능 필요
		// 현재는 간단히 역할 이름 비교로 처리
		for _, requiredRole := range roleNames {
			for _, userRole := range allUserRoles {
				if userRole == requiredRole {
					hasRequiredRole = true
					break
				}
			}
			if hasRequiredRole {
				break
			}
		}

		if !hasRequiredRole {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INSUFFICIENT_ROLE",
					"message": "You don't have any of the required roles",
					"details": gin.H{
						"required_roles": roleNames,
						"user_roles":     allUserRoles,
					},
				},
			})
			return
		}

		c.Set("user_roles", allUserRoles)
		c.Set("permission_matrix", matrix)

		c.Next()
	}
}

// PermissionRequirement 권한 요구사항 구조체
type PermissionRequirement struct {
	ResourceType models.ResourceType `json:"resource_type"`
	Action       models.ActionType   `json:"action"`
}

// extractRequestAttributes 요청에서 속성 추출
func extractRequestAttributes(c *gin.Context) map[string]string {
	attributes := make(map[string]string)
	
	// HTTP 메서드
	attributes["http_method"] = c.Request.Method
	
	// 경로
	attributes["path"] = c.Request.URL.Path
	
	// IP 주소
	attributes["client_ip"] = c.ClientIP()
	
	// 사용자 에이전트
	attributes["user_agent"] = c.GetHeader("User-Agent")
	
	// 요청 시간
	attributes["request_time"] = c.GetHeader("X-Request-Time")
	
	// 추가 컨텍스트 정보
	if userID, exists := GetUserID(c); exists {
		attributes["user_id"] = userID
	}
	
	if username, exists := GetUsername(c); exists {
		attributes["username"] = username
	}
	
	return attributes
}

// GetPermissionDecision 컨텍스트에서 권한 결정 정보 추출
func GetPermissionDecision(c *gin.Context) (*models.PermissionDecision, bool) {
	decision, exists := c.Get("permission_decision")
	if !exists {
		return nil, false
	}
	
	permDecision, ok := decision.(models.PermissionDecision)
	if !ok {
		return nil, false
	}
	
	return &permDecision, true
}

// GetPermissionEvaluation 컨텍스트에서 권한 평가 과정 추출
func GetPermissionEvaluation(c *gin.Context) ([]string, bool) {
	evaluation, exists := c.Get("permission_evaluation")
	if !exists {
		return nil, false
	}
	
	evalSlice, ok := evaluation.([]string)
	return evalSlice, ok
}

// IsResourceOwner 컨텍스트에서 리소스 소유 여부 확인
func IsResourceOwner(c *gin.Context) bool {
	isOwner, exists := c.Get("is_resource_owner")
	if !exists {
		return false
	}
	
	owner, ok := isOwner.(bool)
	return ok && owner
}