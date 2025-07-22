package auth

import (
	"net/http"
	"strings"
	
	"github.com/gin-gonic/gin"
	"github.com/aicli/aicli-web/internal/models"
)

// RBACMiddleware RBAC 미들웨어
type RBACMiddleware struct {
	rbacManager *RBACManager
}

// NewRBACMiddleware RBAC 미들웨어 생성자
func NewRBACMiddleware(rbacManager *RBACManager) *RBACMiddleware {
	return &RBACMiddleware{
		rbacManager: rbacManager,
	}
}

// RequirePermission 특정 권한을 요구하는 미들웨어
func (rm *RBACMiddleware) RequirePermission(resourceType models.ResourceType, action models.ActionType) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 사용자 ID 추출
		userID := rm.extractUserID(c)
		if userID == "" {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Success: false,
				Error: models.ErrorDetail{
					Code:    "UNAUTHORIZED",
					Message: "사용자 인증이 필요합니다",
				},
			})
			c.Abort()
			return
		}
		
		// 2. 리소스 ID 추출 (경로 매개변수 또는 쿼리 파라미터에서)
		resourceID := rm.extractResourceID(c)
		if resourceID == "" {
			resourceID = "*" // 일반적인 권한 확인
		}
		
		// 3. 권한 확인
		req := &models.CheckPermissionRequest{
			UserID:       userID,
			ResourceType: resourceType,
			ResourceID:   resourceID,
			Action:       action,
			Attributes:   rm.extractRequestAttributes(c),
		}
		
		resp, err := rm.rbacManager.CheckPermission(c.Request.Context(), req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Success: false,
				Error: models.ErrorDetail{
					Code:    "PERMISSION_CHECK_ERROR",
					Message: "권한 확인 중 오류가 발생했습니다",
					Details: err.Error(),
				},
			})
			c.Abort()
			return
		}
		
		// 4. 권한이 없는 경우
		if !resp.Allowed {
			c.JSON(http.StatusForbidden, models.ErrorResponse{
				Success: false,
				Error: models.ErrorDetail{
					Code:    "PERMISSION_DENIED",
					Message: "이 작업을 수행할 권한이 없습니다",
					Details: resp.Decision.Reason,
				},
			})
			c.Abort()
			return
		}
		
		// 5. 권한 정보를 컨텍스트에 저장
		c.Set("permission_decision", resp.Decision)
		c.Set("permission_evaluation", resp.Evaluation)
		
		c.Next()
	}
}

// RequireRole 특정 역할을 요구하는 미들웨어
func (rm *RBACMiddleware) RequireRole(roleNames ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := rm.extractUserID(c)
		if userID == "" {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Success: false,
				Error: models.ErrorDetail{
					Code:    "UNAUTHORIZED",
					Message: "사용자 인증이 필요합니다",
				},
			})
			c.Abort()
			return
		}
		
		// 사용자의 권한 매트릭스 조회
		matrix, err := rm.rbacManager.ComputeUserPermissionMatrix(c.Request.Context(), userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Success: false,
				Error: models.ErrorDetail{
					Code:    "ROLE_CHECK_ERROR",
					Message: "역할 확인 중 오류가 발생했습니다",
					Details: err.Error(),
				},
			})
			c.Abort()
			return
		}
		
		// 모든 역할 수집
		allUserRoles := append(matrix.DirectRoles, matrix.InheritedRoles...)
		allUserRoles = append(allUserRoles, matrix.GroupRoles...)
		
		// 필요한 역할 중 하나라도 가지고 있는지 확인
		hasRequiredRole := false
		for _, roleName := range roleNames {
			if rm.containsRole(allUserRoles, roleName) {
				hasRequiredRole = true
				break
			}
		}
		
		if !hasRequiredRole {
			c.JSON(http.StatusForbidden, models.ErrorResponse{
				Success: false,
				Error: models.ErrorDetail{
					Code:    "ROLE_REQUIRED",
					Message: "이 작업을 수행하려면 특정 역할이 필요합니다",
					Details: "Required roles: " + strings.Join(roleNames, ", "),
				},
			})
			c.Abort()
			return
		}
		
		c.Set("user_roles", allUserRoles)
		c.Next()
	}
}

// RequireOwnership 리소스 소유권을 확인하는 미들웨어
func (rm *RBACMiddleware) RequireOwnership(resourceType models.ResourceType) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := rm.extractUserID(c)
		if userID == "" {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Success: false,
				Error: models.ErrorDetail{
					Code:    "UNAUTHORIZED",
					Message: "사용자 인증이 필요합니다",
				},
			})
			c.Abort()
			return
		}
		
		resourceID := rm.extractResourceID(c)
		if resourceID == "" {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Success: false,
				Error: models.ErrorDetail{
					Code:    "RESOURCE_ID_REQUIRED",
					Message: "리소스 ID가 필요합니다",
				},
			})
			c.Abort()
			return
		}
		
		// 소유권 확인을 위한 권한 검사
		req := &models.CheckPermissionRequest{
			UserID:       userID,
			ResourceType: resourceType,
			ResourceID:   resourceID,
			Action:       models.ActionManage,
			Attributes: map[string]string{
				"ownership_check": "true",
			},
		}
		
		resp, err := rm.rbacManager.CheckPermission(c.Request.Context(), req)
		if err != nil || !resp.Allowed {
			c.JSON(http.StatusForbidden, models.ErrorResponse{
				Success: false,
				Error: models.ErrorDetail{
					Code:    "OWNERSHIP_REQUIRED",
					Message: "이 리소스에 대한 소유권이 필요합니다",
				},
			})
			c.Abort()
			return
		}
		
		c.Next()
	}
}

// OptionalPermission 선택적 권한 확인 미들웨어 (권한이 없어도 진행)
func (rm *RBACMiddleware) OptionalPermission(resourceType models.ResourceType, action models.ActionType) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := rm.extractUserID(c)
		if userID == "" {
			c.Next()
			return
		}
		
		resourceID := rm.extractResourceID(c)
		if resourceID == "" {
			resourceID = "*"
		}
		
		req := &models.CheckPermissionRequest{
			UserID:       userID,
			ResourceType: resourceType,
			ResourceID:   resourceID,
			Action:       action,
			Attributes:   rm.extractRequestAttributes(c),
		}
		
		resp, err := rm.rbacManager.CheckPermission(c.Request.Context(), req)
		if err == nil && resp.Allowed {
			c.Set("has_permission", true)
			c.Set("permission_decision", resp.Decision)
		} else {
			c.Set("has_permission", false)
		}
		
		c.Next()
	}
}

// extractUserID 컨텍스트에서 사용자 ID 추출
func (rm *RBACMiddleware) extractUserID(c *gin.Context) string {
	// JWT 클레임에서 사용자 ID 추출
	if claims, exists := c.Get("claims"); exists {
		if jwtClaims, ok := claims.(*Claims); ok {
			return jwtClaims.UserID
		}
	}
	
	// 헤더에서 사용자 ID 추출 (대안)
	if userID := c.GetHeader("X-User-ID"); userID != "" {
		return userID
	}
	
	return ""
}

// extractResourceID 경로나 쿼리에서 리소스 ID 추출
func (rm *RBACMiddleware) extractResourceID(c *gin.Context) string {
	// URL 경로에서 ID 파라미터 추출
	if id := c.Param("id"); id != "" {
		return id
	}
	
	// 리소스별 특정 파라미터 확인
	resourceParams := []string{
		"workspace_id", "workspaceId",
		"project_id", "projectId",
		"session_id", "sessionId",
		"task_id", "taskId",
		"user_id", "userId",
		"group_id", "groupId",
		"role_id", "roleId",
	}
	
	for _, param := range resourceParams {
		if id := c.Param(param); id != "" {
			return id
		}
		if id := c.Query(param); id != "" {
			return id
		}
	}
	
	return ""
}

// extractRequestAttributes 요청에서 추가 속성 추출
func (rm *RBACMiddleware) extractRequestAttributes(c *gin.Context) map[string]string {
	attributes := make(map[string]string)
	
	// HTTP 메서드
	attributes["http_method"] = c.Request.Method
	
	// 요청 경로
	attributes["request_path"] = c.Request.URL.Path
	
	// IP 주소
	attributes["client_ip"] = c.ClientIP()
	
	// User-Agent
	attributes["user_agent"] = c.GetHeader("User-Agent")
	
	// 시간대
	attributes["request_time"] = strings.Split(c.GetString("request_time"), "T")[0] // 날짜만
	
	// 추가 커스텀 헤더들
	if workspace := c.GetHeader("X-Workspace-ID"); workspace != "" {
		attributes["workspace_id"] = workspace
	}
	
	if project := c.GetHeader("X-Project-ID"); project != "" {
		attributes["project_id"] = project
	}
	
	return attributes
}

// containsRole 역할 목록에서 특정 역할이 포함되어 있는지 확인
func (rm *RBACMiddleware) containsRole(roles []string, targetRole string) bool {
	for _, role := range roles {
		if role == targetRole {
			return true
		}
	}
	return false
}

// PermissionResponse 권한 확인 결과 응답
type PermissionResponse struct {
	HasPermission bool                      `json:"has_permission"`
	Decision      *models.PermissionDecision `json:"decision,omitempty"`
	Evaluation    []string                  `json:"evaluation,omitempty"`
}

// GetCurrentUserPermissions 현재 사용자의 권한 정보 조회 핸들러
func (rm *RBACMiddleware) GetCurrentUserPermissions() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := rm.extractUserID(c)
		if userID == "" {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Success: false,
				Error: models.ErrorDetail{
					Code:    "UNAUTHORIZED",
					Message: "사용자 인증이 필요합니다",
				},
			})
			return
		}
		
		matrix, err := rm.rbacManager.ComputeUserPermissionMatrix(c.Request.Context(), userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Success: false,
				Error: models.ErrorDetail{
					Code:    "PERMISSION_QUERY_ERROR",
					Message: "권한 정보 조회 중 오류가 발생했습니다",
					Details: err.Error(),
				},
			})
			return
		}
		
		c.JSON(http.StatusOK, models.SuccessResponse{
			Success: true,
			Data:    matrix,
			Message: "권한 정보를 성공적으로 조회했습니다",
		})
	}
}

// CheckSpecificPermission 특정 권한 확인 핸들러
func (rm *RBACMiddleware) CheckSpecificPermission() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req models.CheckPermissionRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Success: false,
				Error: models.ErrorDetail{
					Code:    "INVALID_REQUEST",
					Message: "요청 데이터가 올바르지 않습니다",
					Details: err.Error(),
				},
			})
			return
		}
		
		// 사용자 ID가 없으면 현재 사용자 ID 사용
		if req.UserID == "" {
			req.UserID = rm.extractUserID(c)
		}
		
		if req.UserID == "" {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Success: false,
				Error: models.ErrorDetail{
					Code:    "UNAUTHORIZED",
					Message: "사용자 인증이 필요합니다",
				},
			})
			return
		}
		
		resp, err := rm.rbacManager.CheckPermission(c.Request.Context(), &req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Success: false,
				Error: models.ErrorDetail{
					Code:    "PERMISSION_CHECK_ERROR",
					Message: "권한 확인 중 오류가 발생했습니다",
					Details: err.Error(),
				},
			})
			return
		}
		
		c.JSON(http.StatusOK, models.SuccessResponse{
			Success: true,
			Data:    resp,
			Message: "권한 확인이 완료되었습니다",
		})
	}
}