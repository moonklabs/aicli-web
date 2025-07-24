package controllers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/aicli/aicli-web/internal/auth"
	"github.com/aicli/aicli-web/internal/middleware"
	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/services"
)

// UserController는 사용자 관리 관련 API를 처리합니다.
type UserController struct {
	userService services.UserService
}

// NewUserController는 새로운 사용자 컨트롤러를 생성합니다.
func NewUserController(userService services.UserService) *UserController {
	return &UserController{
		userService: userService,
	}
}

// GetProfile은 현재 사용자의 프로파일을 조회합니다.
// @Summary 내 프로파일 조회
// @Description 현재 로그인한 사용자의 프로파일 정보를 조회합니다
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.UserResponse "사용자 프로파일"
// @Failure 401 {object} models.ErrorResponse "인증 실패"
// @Failure 404 {object} models.ErrorResponse "사용자 없음"
// @Router /users/me [get]
func (uc *UserController) GetProfile(c *gin.Context) {
	claims, exists := c.Get("claims")
	if !exists {
		middleware.UnauthorizedError(c, "인증 정보를 찾을 수 없습니다")
		return
	}
	userClaims := claims.(*auth.Claims)

	profile, err := uc.userService.GetProfile(c.Request.Context(), userClaims.UserID)
	if err != nil {
		middleware.InternalServerError(c, "프로파일 조회 실패")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    profile,
	})
}

// UpdateProfile은 현재 사용자의 프로파일을 업데이트합니다.
// @Summary 내 프로파일 업데이트
// @Description 현재 로그인한 사용자의 프로파일 정보를 업데이트합니다
// @Tags users
// @Accept json
// @Produce json
// @Param request body models.UpdateProfileRequest true "프로파일 업데이트 요청"
// @Security BearerAuth
// @Success 200 {object} models.UserResponse "업데이트된 사용자 프로파일"
// @Failure 400 {object} models.ErrorResponse "잘못된 요청"
// @Failure 401 {object} models.ErrorResponse "인증 실패"
// @Router /users/me [put]
func (uc *UserController) UpdateProfile(c *gin.Context) {
	claims, exists := c.Get("claims")
	if !exists {
		middleware.UnauthorizedError(c, "인증 정보를 찾을 수 없습니다")
		return
	}
	userClaims := claims.(*auth.Claims)

	var req models.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.BadRequestError(c, "잘못된 요청 형식")
		return
	}

	profile, err := uc.userService.UpdateProfile(c.Request.Context(), userClaims.UserID, &req)
	if err != nil {
		middleware.InternalServerError(c, "프로파일 업데이트 실패")
		return
	}

	// 활동 로그 기록
	uc.userService.LogActivity(c.Request.Context(), userClaims.UserID, "profile_update", "user", 
		"프로파일 업데이트", c.ClientIP(), c.GetHeader("User-Agent"))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    profile,
		"message": "프로파일이 성공적으로 업데이트되었습니다",
	})
}

// ChangePassword는 현재 사용자의 비밀번호를 변경합니다.
// @Summary 비밀번호 변경
// @Description 현재 로그인한 사용자의 비밀번호를 변경합니다
// @Tags users
// @Accept json
// @Produce json
// @Param request body models.ChangePasswordRequest true "비밀번호 변경 요청"
// @Security BearerAuth
// @Success 200 {object} gin.H "성공 메시지"
// @Failure 400 {object} models.ErrorResponse "잘못된 요청"
// @Failure 401 {object} models.ErrorResponse "인증 실패"
// @Router /users/me/password [put]
func (uc *UserController) ChangePassword(c *gin.Context) {
	claims, exists := c.Get("claims")
	if !exists {
		middleware.UnauthorizedError(c, "인증 정보를 찾을 수 없습니다")
		return
	}
	userClaims := claims.(*auth.Claims)

	var req models.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.BadRequestError(c, "잘못된 요청 형식")
		return
	}

	err := uc.userService.ChangePassword(c.Request.Context(), userClaims.UserID, &req)
	if err != nil {
		middleware.BadRequestError(c, "비밀번호 변경 실패")
		return
	}

	// 활동 로그 기록
	uc.userService.LogActivity(c.Request.Context(), userClaims.UserID, "password_change", "user", 
		"비밀번호 변경", c.ClientIP(), c.GetHeader("User-Agent"))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "비밀번호가 성공적으로 변경되었습니다",
	})
}

// ChangeEmail은 현재 사용자의 이메일을 변경합니다.
// @Summary 이메일 변경
// @Description 현재 로그인한 사용자의 이메일을 변경합니다
// @Tags users
// @Accept json
// @Produce json
// @Param request body models.ChangeEmailRequest true "이메일 변경 요청"
// @Security BearerAuth
// @Success 200 {object} gin.H "성공 메시지"
// @Failure 400 {object} models.ErrorResponse "잘못된 요청"
// @Failure 401 {object} models.ErrorResponse "인증 실패"
// @Router /users/me/email [put]
func (uc *UserController) ChangeEmail(c *gin.Context) {
	claims, exists := c.Get("claims")
	if !exists {
		middleware.UnauthorizedError(c, "인증 정보를 찾을 수 없습니다")
		return
	}
	userClaims := claims.(*auth.Claims)

	var req models.ChangeEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.BadRequestError(c, "잘못된 요청 형식")
		return
	}

	err := uc.userService.ChangeEmail(c.Request.Context(), userClaims.UserID, &req)
	if err != nil {
		middleware.BadRequestError(c, "이메일 변경 실패")
		return
	}

	// 활동 로그 기록
	uc.userService.LogActivity(c.Request.Context(), userClaims.UserID, "email_change", "user", 
		"이메일 변경", c.ClientIP(), c.GetHeader("User-Agent"))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "이메일이 성공적으로 변경되었습니다. 새 이메일로 인증을 진행해주세요.",
	})
}

// Generate2FASecret은 2FA 비밀키를 생성합니다.
// @Summary 2FA 비밀키 생성
// @Description 2단계 인증을 위한 비밀키를 생성합니다
// @Tags users,security
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.TwoFactorSecret "2FA 비밀키"
// @Failure 401 {object} models.ErrorResponse "인증 실패"
// @Router /users/me/2fa/generate [post]
func (uc *UserController) Generate2FASecret(c *gin.Context) {
	claims, exists := c.Get("claims")
	if !exists {
		middleware.UnauthorizedError(c, "인증 정보를 찾을 수 없습니다")
		return
	}
	userClaims := claims.(*auth.Claims)

	secret, err := uc.userService.Generate2FASecret(c.Request.Context(), userClaims.UserID)
	if err != nil {
		middleware.InternalServerError(c, "2FA 비밀키 생성 실패")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    secret,
		"message": "2FA 비밀키가 생성되었습니다",
	})
}

// Enable2FA는 2FA를 활성화합니다.
// @Summary 2FA 활성화
// @Description 2단계 인증을 활성화합니다
// @Tags users,security
// @Accept json
// @Produce json
// @Param request body models.Enable2FARequest true "2FA 활성화 요청"
// @Security BearerAuth
// @Success 200 {object} gin.H "성공 메시지"
// @Failure 400 {object} models.ErrorResponse "잘못된 요청"
// @Failure 401 {object} models.ErrorResponse "인증 실패"
// @Router /users/me/2fa/enable [post]
func (uc *UserController) Enable2FA(c *gin.Context) {
	claims, exists := c.Get("claims")
	if !exists {
		middleware.UnauthorizedError(c, "인증 정보를 찾을 수 없습니다")
		return
	}
	userClaims := claims.(*auth.Claims)

	var req models.Enable2FARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.BadRequestError(c, "잘못된 요청 형식")
		return
	}

	_, err := uc.userService.Enable2FA(c.Request.Context(), userClaims.UserID, &req)
	if err != nil {
		middleware.BadRequestError(c, "2FA 활성화 실패")
		return
	}

	// 활동 로그 기록
	uc.userService.LogActivity(c.Request.Context(), userClaims.UserID, "2fa_enable", "user", 
		"2FA 활성화", c.ClientIP(), c.GetHeader("User-Agent"))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "2FA가 성공적으로 활성화되었습니다",
	})
}

// Disable2FA는 2FA를 비활성화합니다.
// @Summary 2FA 비활성화
// @Description 2단계 인증을 비활성화합니다
// @Tags users,security
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} gin.H "성공 메시지"
// @Failure 401 {object} models.ErrorResponse "인증 실패"
// @Router /users/me/2fa/disable [post]
func (uc *UserController) Disable2FA(c *gin.Context) {
	claims, exists := c.Get("claims")
	if !exists {
		middleware.UnauthorizedError(c, "인증 정보를 찾을 수 없습니다")
		return
	}
	userClaims := claims.(*auth.Claims)

	err := uc.userService.Disable2FA(c.Request.Context(), userClaims.UserID)
	if err != nil {
		middleware.InternalServerError(c, "2FA 비활성화 실패")
		return
	}

	// 활동 로그 기록
	uc.userService.LogActivity(c.Request.Context(), userClaims.UserID, "2fa_disable", "user", 
		"2FA 비활성화", c.ClientIP(), c.GetHeader("User-Agent"))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "2FA가 성공적으로 비활성화되었습니다",
	})
}

// GetActivities는 현재 사용자의 활동 로그를 조회합니다.
// @Summary 내 활동 로그 조회
// @Description 현재 로그인한 사용자의 활동 로그를 조회합니다
// @Tags users
// @Accept json
// @Produce json
// @Param page query int false "페이지 번호" default(1)
// @Param limit query int false "페이지당 항목 수" default(20)
// @Security BearerAuth
// @Success 200 {object} models.PaginatedResponse "활동 로그 목록"
// @Failure 401 {object} models.ErrorResponse "인증 실패"
// @Router /users/me/activities [get]
func (uc *UserController) GetActivities(c *gin.Context) {
	claims, exists := c.Get("claims")
	if !exists {
		middleware.UnauthorizedError(c, "인증 정보를 찾을 수 없습니다")
		return
	}
	userClaims := claims.(*auth.Claims)

	// 페이지네이션 파라미터 파싱
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	pagination := &models.PaginationRequest{
		Page:  page,
		Limit: limit,
	}

	activities, err := uc.userService.GetUserActivities(c.Request.Context(), userClaims.UserID, pagination)
	if err != nil {
		middleware.InternalServerError(c, "활동 로그 조회 실패")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    activities,
	})
}

// ===== 관리자 전용 API =====

// ListUsers는 사용자 목록을 조회합니다 (관리자 전용).
// @Summary 사용자 목록 조회 (관리자)
// @Description 관리자가 모든 사용자 목록을 조회합니다
// @Tags admin,users
// @Accept json
// @Produce json
// @Param page query int false "페이지 번호" default(1)
// @Param limit query int false "페이지당 항목 수" default(20)
// @Param search query string false "검색어"
// @Param role query string false "역할 필터"
// @Param is_active query bool false "활성 상태 필터"
// @Security BearerAuth
// @Success 200 {object} models.PaginatedResponse "사용자 목록"
// @Failure 401 {object} models.ErrorResponse "인증 실패"
// @Failure 403 {object} models.ErrorResponse "권한 없음"
// @Router /admin/users [get]
func (uc *UserController) ListUsers(c *gin.Context) {
	// 관리자 권한 확인은 미들웨어에서 처리됨

	// 필터 및 페이지네이션 파라미터 파싱
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	search := c.Query("search")
	role := c.Query("role")
	
	var isActive *bool
	if activeStr := c.Query("is_active"); activeStr != "" {
		if activeVal, err := strconv.ParseBool(activeStr); err == nil {
			isActive = &activeVal
		}
	}

	filter := &models.UserFilter{
		Search:   search,
		Role:     role,
		IsActive: isActive,
		PaginationRequest: models.PaginationRequest{
			Page:  page,
			Limit: limit,
		},
	}

	users, err := uc.userService.ListUsers(c.Request.Context(), filter)
	if err != nil {
		middleware.InternalServerError(c, "사용자 목록 조회 실패")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    users,
	})
}

// CreateUser는 새 사용자를 생성합니다 (관리자 전용).
// @Summary 사용자 생성 (관리자)
// @Description 관리자가 새 사용자를 생성합니다
// @Tags admin,users
// @Accept json
// @Produce json
// @Param request body models.CreateUserRequest true "사용자 생성 요청"
// @Security BearerAuth
// @Success 201 {object} models.UserResponse "생성된 사용자"
// @Failure 400 {object} models.ErrorResponse "잘못된 요청"
// @Failure 401 {object} models.ErrorResponse "인증 실패"
// @Failure 403 {object} models.ErrorResponse "권한 없음"
// @Router /admin/users [post]
func (uc *UserController) CreateUser(c *gin.Context) {
	var req models.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.BadRequestError(c, "잘못된 요청 형식")
		return
	}

	user, err := uc.userService.CreateUser(c.Request.Context(), &req)
	if err != nil {
		middleware.BadRequestError(c, "사용자 생성 실패")
		return
	}

	// 활동 로그 기록 (관리자)
	claims, _ := c.Get("claims")
	if userClaims, ok := claims.(*auth.Claims); ok {
		uc.userService.LogActivity(c.Request.Context(), userClaims.UserID, "user_create", "admin", 
			"사용자 생성: "+req.Username, c.ClientIP(), c.GetHeader("User-Agent"))
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    user,
		"message": "사용자가 성공적으로 생성되었습니다",
	})
}

// GetUser는 특정 사용자를 조회합니다 (관리자 전용).
// @Summary 사용자 조회 (관리자)
// @Description 관리자가 특정 사용자를 조회합니다
// @Tags admin,users
// @Accept json
// @Produce json
// @Param id path string true "사용자 ID"
// @Security BearerAuth
// @Success 200 {object} models.UserResponse "사용자 정보"
// @Failure 401 {object} models.ErrorResponse "인증 실패"
// @Failure 403 {object} models.ErrorResponse "권한 없음"
// @Failure 404 {object} models.ErrorResponse "사용자 없음"
// @Router /admin/users/{id} [get]
func (uc *UserController) GetUser(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		middleware.BadRequestError(c, "사용자 ID가 필요합니다")
		return
	}

	user, err := uc.userService.GetUser(c.Request.Context(), userID)
	if err != nil {
		middleware.NotFoundError(c, "사용자를 찾을 수 없습니다")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    user,
	})
}

// UpdateUser는 사용자를 업데이트합니다 (관리자 전용).
// @Summary 사용자 업데이트 (관리자)
// @Description 관리자가 사용자 정보를 업데이트합니다
// @Tags admin,users
// @Accept json
// @Produce json
// @Param id path string true "사용자 ID"
// @Param request body models.UpdateUserRequest true "사용자 업데이트 요청"
// @Security BearerAuth
// @Success 200 {object} models.UserResponse "업데이트된 사용자"
// @Failure 400 {object} models.ErrorResponse "잘못된 요청"
// @Failure 401 {object} models.ErrorResponse "인증 실패"
// @Failure 403 {object} models.ErrorResponse "권한 없음"
// @Failure 404 {object} models.ErrorResponse "사용자 없음"
// @Router /admin/users/{id} [put]
func (uc *UserController) UpdateUser(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		middleware.BadRequestError(c, "사용자 ID가 필요합니다")
		return
	}

	var req models.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.BadRequestError(c, "잘못된 요청 형식")
		return
	}

	user, err := uc.userService.UpdateUser(c.Request.Context(), userID, &req)
	if err != nil {
		middleware.BadRequestError(c, "사용자 업데이트 실패")
		return
	}

	// 활동 로그 기록 (관리자)
	claims, _ := c.Get("claims")
	if userClaims, ok := claims.(*auth.Claims); ok {
		uc.userService.LogActivity(c.Request.Context(), userClaims.UserID, "user_update", "admin", 
			"사용자 업데이트: "+userID, c.ClientIP(), c.GetHeader("User-Agent"))
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    user,
		"message": "사용자가 성공적으로 업데이트되었습니다",
	})
}

// DeleteUser는 사용자를 삭제합니다 (관리자 전용).
// @Summary 사용자 삭제 (관리자)
// @Description 관리자가 사용자를 삭제합니다
// @Tags admin,users
// @Accept json
// @Produce json
// @Param id path string true "사용자 ID"
// @Security BearerAuth
// @Success 200 {object} gin.H "성공 메시지"
// @Failure 400 {object} models.ErrorResponse "잘못된 요청"
// @Failure 401 {object} models.ErrorResponse "인증 실패"
// @Failure 403 {object} models.ErrorResponse "권한 없음"
// @Failure 404 {object} models.ErrorResponse "사용자 없음"
// @Router /admin/users/{id} [delete]
func (uc *UserController) DeleteUser(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		middleware.BadRequestError(c, "사용자 ID가 필요합니다")
		return
	}

	err := uc.userService.DeleteUser(c.Request.Context(), userID)
	if err != nil {
		middleware.BadRequestError(c, "사용자 삭제 실패")
		return
	}

	// 활동 로그 기록 (관리자)
	claims, _ := c.Get("claims")
	if userClaims, ok := claims.(*auth.Claims); ok {
		uc.userService.LogActivity(c.Request.Context(), userClaims.UserID, "user_delete", "admin", 
			"사용자 삭제: "+userID, c.ClientIP(), c.GetHeader("User-Agent"))
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "사용자가 성공적으로 삭제되었습니다",
	})
}

// GetUserStats는 사용자 통계를 조회합니다 (관리자 전용).
// @Summary 사용자 통계 조회 (관리자)
// @Description 관리자가 사용자 통계를 조회합니다
// @Tags admin,users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.UserStats "사용자 통계"
// @Failure 401 {object} models.ErrorResponse "인증 실패"
// @Failure 403 {object} models.ErrorResponse "권한 없음"
// @Router /admin/users/stats [get]
func (uc *UserController) GetUserStats(c *gin.Context) {
	stats, err := uc.userService.GetUserStats(c.Request.Context())
	if err != nil {
		middleware.InternalServerError(c, "사용자 통계 조회 실패")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// ===== 비밀번호 재설정 API (인증 불필요) =====

// RequestPasswordReset은 비밀번호 재설정을 요청합니다.
// @Summary 비밀번호 재설정 요청
// @Description 이메일을 통한 비밀번호 재설정을 요청합니다
// @Tags auth
// @Accept json
// @Produce json
// @Param request body models.ResetPasswordRequest true "비밀번호 재설정 요청"
// @Success 200 {object} gin.H "성공 메시지"
// @Failure 400 {object} models.ErrorResponse "잘못된 요청"
// @Router /auth/password-reset [post]
func (uc *UserController) RequestPasswordReset(c *gin.Context) {
	var req models.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.BadRequestError(c, "잘못된 요청 형식")
		return
	}

	err := uc.userService.RequestPasswordReset(c.Request.Context(), &req)
	if err != nil {
		middleware.InternalServerError(c, "비밀번호 재설정 요청 실패")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "비밀번호 재설정 이메일이 발송되었습니다",
	})
}

// ConfirmPasswordReset은 비밀번호 재설정을 확인합니다.
// @Summary 비밀번호 재설정 확인
// @Description 토큰을 통한 비밀번호 재설정을 확인합니다
// @Tags auth
// @Accept json
// @Produce json
// @Param request body models.ConfirmResetPasswordRequest true "비밀번호 재설정 확인 요청"
// @Success 200 {object} gin.H "성공 메시지"
// @Failure 400 {object} models.ErrorResponse "잘못된 요청"
// @Router /auth/password-reset/confirm [post]
func (uc *UserController) ConfirmPasswordReset(c *gin.Context) {
	var req models.ConfirmResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.BadRequestError(c, "잘못된 요청 형식")
		return
	}

	err := uc.userService.ConfirmPasswordReset(c.Request.Context(), &req)
	if err != nil {
		middleware.BadRequestError(c, "비밀번호 재설정 실패")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "비밀번호가 성공적으로 재설정되었습니다",
	})
}