package controllers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/aicli/aicli-web/internal/auth"
	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/storage"
	"github.com/aicli/aicli-web/internal/middleware"
)

// RBACController RBAC 관리 컨트롤러
type RBACController struct {
	rbacManager *auth.RBACManager
	storage     storage.Storage
}

// NewRBACController RBAC 컨트롤러 생성자
func NewRBACController(rbacManager *auth.RBACManager, storage storage.Storage) *RBACController {
	return &RBACController{
		rbacManager: rbacManager,
		storage:     storage,
	}
}

// 역할 관리 API

// CreateRole godoc
// @Summary 새 역할 생성
// @Description 새로운 역할을 생성합니다
// @Tags RBAC
// @Accept json
// @Produce json
// @Param role body models.CreateRoleRequest true "역할 생성 요청"
// @Success 201 {object} models.Role
// @Failure 400 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 403 {object} gin.H
// @Failure 500 {object} gin.H
// @Security BearerAuth
// @Router /api/v1/rbac/roles [post]
func (rc *RBACController) CreateRole(c *gin.Context) {
	var req models.CreateRoleRequest
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

	// 역할 생성
	role := &models.Role{
		Name:        req.Name,
		Description: req.Description,
		ParentID:    req.ParentID,
		IsSystem:    false, // 사용자 생성 역할
		IsActive:    true,
	}

	// 부모 역할이 있는 경우 레벨 계산
	if req.ParentID != nil {
		ctx := context.Background()
		parentRole, err := rc.storage.RBAC().GetRoleByID(ctx, *req.ParentID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INVALID_PARENT_ROLE",
					"message": "Parent role not found",
					"details": err.Error(),
				},
			})
			return
		}
		role.Level = parentRole.Level + 1
	}

	ctx := context.Background()
	if err := rc.storage.RBAC().CreateRole(ctx, role); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "ROLE_CREATION_FAILED",
				"message": "Failed to create role",
				"details": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    role,
	})
}

// GetRole godoc
// @Summary 역할 상세 조회
// @Description 역할 ID로 역할 상세 정보를 조회합니다
// @Tags RBAC
// @Accept json
// @Produce json
// @Param id path string true "역할 ID"
// @Success 200 {object} models.Role
// @Failure 404 {object} gin.H
// @Failure 500 {object} gin.H
// @Security BearerAuth
// @Router /api/v1/rbac/roles/{id} [get]
func (rc *RBACController) GetRole(c *gin.Context) {
	roleID := c.Param("id")
	if roleID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "MISSING_ROLE_ID",
				"message": "Role ID is required",
			},
		})
		return
	}

	ctx := context.Background()
	role, err := rc.storage.RBAC().GetRoleByID(ctx, roleID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "ROLE_NOT_FOUND",
				"message": "Role not found",
				"details": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    role,
	})
}

// ListRoles godoc
// @Summary 역할 목록 조회
// @Description 페이지네이션이 적용된 역할 목록을 조회합니다
// @Tags RBAC
// @Accept json
// @Produce json
// @Param page query int false "페이지 번호" default(1)
// @Param limit query int false "페이지당 항목 수" default(20)
// @Param search query string false "검색 키워드"
// @Param active query bool false "활성 상태 필터"
// @Success 200 {object} gin.H
// @Failure 400 {object} gin.H
// @Failure 500 {object} gin.H
// @Security BearerAuth
// @Router /api/v1/rbac/roles [get]
func (rc *RBACController) ListRoles(c *gin.Context) {
	// 페이지네이션 파라미터
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	// 필터 파라미터
	search := c.Query("search")
	activeStr := c.Query("active")
	var active *bool
	if activeStr != "" {
		activeBool, _ := strconv.ParseBool(activeStr)
		active = &activeBool
	}

	ctx := context.Background()
	roles, total, err := rc.storage.RBAC().ListRoles(ctx, models.ListRolesRequest{
		Page:   page,
		Limit:  limit,
		Search: search,
		Active: active,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "ROLES_LIST_FAILED",
				"message": "Failed to list roles",
				"details": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"roles": roles,
			"pagination": gin.H{
				"page":       page,
				"limit":      limit,
				"total":      total,
				"total_pages": (total + int64(limit) - 1) / int64(limit),
			},
		},
	})
}

// UpdateRole godoc
// @Summary 역할 수정
// @Description 역할 정보를 수정합니다
// @Tags RBAC
// @Accept json
// @Produce json
// @Param id path string true "역할 ID"
// @Param role body models.UpdateRoleRequest true "역할 수정 요청"
// @Success 200 {object} models.Role
// @Failure 400 {object} gin.H
// @Failure 404 {object} gin.H
// @Failure 500 {object} gin.H
// @Security BearerAuth
// @Router /api/v1/rbac/roles/{id} [put]
func (rc *RBACController) UpdateRole(c *gin.Context) {
	roleID := c.Param("id")
	if roleID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "MISSING_ROLE_ID",
				"message": "Role ID is required",
			},
		})
		return
	}

	var req models.UpdateRoleRequest
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

	ctx := context.Background()
	
	// 기존 역할 조회
	role, err := rc.storage.RBAC().GetRoleByID(ctx, roleID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "ROLE_NOT_FOUND",
				"message": "Role not found",
				"details": err.Error(),
			},
		})
		return
	}

	// 시스템 역할은 수정 제한
	if role.IsSystem {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "SYSTEM_ROLE_IMMUTABLE",
				"message": "System roles cannot be modified",
			},
		})
		return
	}

	// 요청된 필드만 업데이트
	if req.Name != nil {
		role.Name = *req.Name
	}
	if req.Description != nil {
		role.Description = *req.Description
	}
	if req.ParentID != nil {
		role.ParentID = req.ParentID
		// 부모 역할 변경 시 레벨 재계산
		if *req.ParentID != "" {
			parentRole, err := rc.storage.RBAC().GetRoleByID(ctx, *req.ParentID)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "INVALID_PARENT_ROLE",
						"message": "Parent role not found",
						"details": err.Error(),
					},
				})
				return
			}
			role.Level = parentRole.Level + 1
		} else {
			role.Level = 0
		}
	}
	if req.IsActive != nil {
		role.IsActive = *req.IsActive
	}

	if err := rc.storage.RBAC().UpdateRole(ctx, role); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "ROLE_UPDATE_FAILED",
				"message": "Failed to update role",
				"details": err.Error(),
			},
		})
		return
	}

	// 역할 변경 시 캐시 무효화
	rc.rbacManager.InvalidateRolePermissions(roleID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    role,
	})
}

// DeleteRole godoc
// @Summary 역할 삭제
// @Description 역할을 삭제합니다
// @Tags RBAC
// @Accept json
// @Produce json
// @Param id path string true "역할 ID"
// @Success 204 "역할 삭제 성공"
// @Failure 400 {object} gin.H
// @Failure 404 {object} gin.H
// @Failure 500 {object} gin.H
// @Security BearerAuth
// @Router /api/v1/rbac/roles/{id} [delete]
func (rc *RBACController) DeleteRole(c *gin.Context) {
	roleID := c.Param("id")
	if roleID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "MISSING_ROLE_ID",
				"message": "Role ID is required",
			},
		})
		return
	}

	ctx := context.Background()
	
	// 기존 역할 조회
	role, err := rc.storage.RBAC().GetRoleByID(ctx, roleID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "ROLE_NOT_FOUND",
				"message": "Role not found",
				"details": err.Error(),
			},
		})
		return
	}

	// 시스템 역할은 삭제 금지
	if role.IsSystem {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "SYSTEM_ROLE_UNDELETABLE",
				"message": "System roles cannot be deleted",
			},
		})
		return
	}

	// 역할 사용 중인지 확인
	users, err := rc.storage.RBAC().GetUsersByRoleID(ctx, roleID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "ROLE_USAGE_CHECK_FAILED",
				"message": "Failed to check role usage",
				"details": err.Error(),
			},
		})
		return
	}

	if len(users) > 0 {
		c.JSON(http.StatusConflict, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "ROLE_IN_USE",
				"message": "Cannot delete role that is assigned to users",
				"details": gin.H{
					"assigned_users": len(users),
				},
			},
		})
		return
	}

	if err := rc.storage.RBAC().DeleteRole(ctx, roleID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "ROLE_DELETION_FAILED",
				"message": "Failed to delete role",
				"details": err.Error(),
			},
		})
		return
	}

	// 역할 삭제 시 캐시 무효화
	rc.rbacManager.InvalidateRolePermissions(roleID)

	c.Status(http.StatusNoContent)
}

// 권한 관리 API

// CreatePermission godoc
// @Summary 새 권한 생성
// @Description 새로운 권한을 생성합니다
// @Tags RBAC
// @Accept json
// @Produce json
// @Param permission body models.CreatePermissionRequest true "권한 생성 요청"
// @Success 201 {object} models.Permission
// @Failure 400 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 403 {object} gin.H
// @Failure 500 {object} gin.H
// @Security BearerAuth
// @Router /api/v1/rbac/permissions [post]
func (rc *RBACController) CreatePermission(c *gin.Context) {
	var req models.CreatePermissionRequest
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

	permission := &models.Permission{
		Name:         req.Name,
		Description:  req.Description,
		ResourceType: req.ResourceType,
		Action:       req.Action,
		Effect:       req.Effect,
		Conditions:   req.Conditions,
		IsActive:     true,
	}

	ctx := context.Background()
	if err := rc.storage.RBAC().CreatePermission(ctx, permission); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "PERMISSION_CREATION_FAILED",
				"message": "Failed to create permission",
				"details": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    permission,
	})
}

// ListPermissions godoc
// @Summary 권한 목록 조회
// @Description 페이지네이션이 적용된 권한 목록을 조회합니다
// @Tags RBAC
// @Accept json
// @Produce json
// @Param page query int false "페이지 번호" default(1)
// @Param limit query int false "페이지당 항목 수" default(20)
// @Param resource_type query string false "리소스 타입 필터"
// @Param action query string false "액션 필터"
// @Param effect query string false "효과 필터"
// @Success 200 {object} gin.H
// @Failure 400 {object} gin.H
// @Failure 500 {object} gin.H
// @Security BearerAuth
// @Router /api/v1/rbac/permissions [get]
func (rc *RBACController) ListPermissions(c *gin.Context) {
	// 페이지네이션 파라미터
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	// 필터 파라미터
	resourceType := c.Query("resource_type")
	action := c.Query("action")
	effect := c.Query("effect")

	ctx := context.Background()
	permissions, total, err := rc.storage.RBAC().ListPermissions(ctx, models.ListPermissionsRequest{
		Page:         page,
		Limit:        limit,
		ResourceType: resourceType,
		Action:       action,
		Effect:       effect,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "PERMISSIONS_LIST_FAILED",
				"message": "Failed to list permissions",
				"details": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"permissions": permissions,
			"pagination": gin.H{
				"page":       page,
				"limit":      limit,
				"total":      total,
				"total_pages": (total + int64(limit) - 1) / int64(limit),
			},
		},
	})
}

// AssignRoleToUser godoc
// @Summary 사용자에게 역할 할당
// @Description 사용자에게 역할을 할당합니다
// @Tags RBAC
// @Accept json
// @Produce json
// @Param assignment body models.AssignRoleRequest true "역할 할당 요청"
// @Success 201 {object} gin.H
// @Failure 400 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 403 {object} gin.H
// @Failure 500 {object} gin.H
// @Security BearerAuth
// @Router /api/v1/rbac/user-roles [post]
func (rc *RBACController) AssignRoleToUser(c *gin.Context) {
	var req models.AssignRoleRequest
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

	// 할당하는 사용자 ID 추출
	assignedBy, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "AUTHENTICATION_REQUIRED",
				"message": "Authentication required",
			},
		})
		return
	}

	userRole := &models.UserRole{
		UserID:     req.UserID,
		RoleID:     req.RoleID,
		AssignedBy: assignedBy,
		ResourceID: req.ResourceID,
		ExpiresAt:  req.ExpiresAt,
		IsActive:   true,
	}

	ctx := context.Background()
	if err := rc.storage.RBAC().AssignRoleToUser(ctx, userRole); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "ROLE_ASSIGNMENT_FAILED",
				"message": "Failed to assign role to user",
				"details": err.Error(),
			},
		})
		return
	}

	// 사용자 권한 캐시 무효화
	rc.rbacManager.InvalidateUserPermissions(req.UserID)

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Role assigned to user successfully",
		"data":    userRole,
	})
}

// CheckPermission godoc
// @Summary 권한 확인
// @Description 사용자의 특정 리소스에 대한 권한을 확인합니다
// @Tags RBAC
// @Accept json
// @Produce json
// @Param request body models.CheckPermissionRequest true "권한 확인 요청"
// @Success 200 {object} models.CheckPermissionResponse
// @Failure 400 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 500 {object} gin.H
// @Security BearerAuth
// @Router /api/v1/rbac/check-permission [post]
func (rc *RBACController) CheckPermission(c *gin.Context) {
	var req models.CheckPermissionRequest
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

	ctx := context.Background()
	response, err := rc.rbacManager.CheckPermission(ctx, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "PERMISSION_CHECK_FAILED",
				"message": "Failed to check permission",
				"details": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
	})
}

// GetUserPermissions godoc
// @Summary 사용자 권한 조회
// @Description 사용자의 모든 권한을 조회합니다
// @Tags RBAC
// @Accept json
// @Produce json
// @Param user_id path string true "사용자 ID"
// @Success 200 {object} gin.H
// @Failure 400 {object} gin.H
// @Failure 404 {object} gin.H
// @Failure 500 {object} gin.H
// @Security BearerAuth
// @Router /api/v1/rbac/users/{user_id}/permissions [get]
func (rc *RBACController) GetUserPermissions(c *gin.Context) {
	userID := c.Param("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "MISSING_USER_ID",
				"message": "User ID is required",
			},
		})
		return
	}

	ctx := context.Background()
	matrix, err := rc.rbacManager.ComputeUserPermissionMatrix(ctx, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "PERMISSION_MATRIX_FAILED",
				"message": "Failed to compute user permission matrix",
				"details": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    matrix,
	})
}

// InvalidateCache godoc
// @Summary 캐시 무효화
// @Description 특정 타입의 권한 캐시를 무효화합니다
// @Tags RBAC
// @Accept json
// @Produce json
// @Param type query string true "무효화 타입" Enums(user,role,group,all)
// @Param id query string false "대상 ID (type이 all이 아닌 경우 필수)"
// @Success 200 {object} gin.H
// @Failure 400 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 403 {object} gin.H
// @Failure 500 {object} gin.H
// @Security BearerAuth
// @Router /api/v1/rbac/cache/invalidate [post]
func (rc *RBACController) InvalidateCache(c *gin.Context) {
	cacheType := c.Query("type")
	targetID := c.Query("id")

	if cacheType == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "MISSING_CACHE_TYPE",
				"message": "Cache type is required",
			},
		})
		return
	}

	var err error
	switch cacheType {
	case "user":
		if targetID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "MISSING_USER_ID",
					"message": "User ID is required for user cache invalidation",
				},
			})
			return
		}
		err = rc.rbacManager.InvalidateUserPermissions(targetID)
	case "role":
		if targetID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "MISSING_ROLE_ID",
					"message": "Role ID is required for role cache invalidation",
				},
			})
			return
		}
		err = rc.rbacManager.InvalidateRolePermissions(targetID)
	case "group":
		if targetID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "MISSING_GROUP_ID",
					"message": "Group ID is required for group cache invalidation",
				},
			})
			return
		}
		err = rc.rbacManager.InvalidateGroupPermissions(targetID)
	case "all":
		// TODO: 모든 캐시 무효화 구현
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "All cache invalidation not implemented yet",
		})
		return
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_CACHE_TYPE",
				"message": "Invalid cache type. Must be one of: user, role, group, all",
			},
		})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "CACHE_INVALIDATION_FAILED",
				"message": "Failed to invalidate cache",
				"details": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Cache invalidated successfully",
	})
}