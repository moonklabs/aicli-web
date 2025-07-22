package controllers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/aicli/aicli-web/internal/auth"
	"github.com/aicli/aicli-web/internal/middleware"
	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/security"
)

// PolicyController는 보안 정책 관련 API를 처리합니다.
type PolicyController struct {
	policyService security.PolicyService
}

// NewPolicyController는 새로운 정책 컨트롤러를 생성합니다.
func NewPolicyController(policyService security.PolicyService) *PolicyController {
	return &PolicyController{
		policyService: policyService,
	}
}

// CreatePolicy는 새로운 보안 정책을 생성합니다.
// @Summary 보안 정책 생성
// @Description 새로운 보안 정책을 생성합니다
// @Tags security,policies
// @Accept json
// @Produce json
// @Param request body security.CreatePolicyRequest true "정책 생성 요청"
// @Security BearerAuth
// @Success 201 {object} security.SecurityPolicyResponse "생성된 정책"
// @Failure 400 {object} models.ErrorResponse "잘못된 요청"
// @Failure 401 {object} models.ErrorResponse "인증 실패"
// @Failure 403 {object} models.ErrorResponse "권한 없음"
// @Router /admin/policies [post]
func (pc *PolicyController) CreatePolicy(c *gin.Context) {
	claims, exists := c.Get("claims")
	if !exists {
		middleware.UnauthorizedError(c, "인증 정보를 찾을 수 없습니다")
		return
	}
	userClaims := claims.(*auth.Claims)

	var req security.CreatePolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.BadRequestError(c, "잘못된 요청 형식", err)
		return
	}

	policy, err := pc.policyService.CreatePolicy(c.Request.Context(), &req)
	if err != nil {
		middleware.BadRequestError(c, "정책 생성 실패", err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    policy.ToResponse(),
		"message": "보안 정책이 성공적으로 생성되었습니다",
	})
}

// GetPolicy는 특정 보안 정책을 조회합니다.
// @Summary 보안 정책 조회
// @Description 특정 보안 정책을 조회합니다
// @Tags security,policies
// @Accept json
// @Produce json
// @Param id path string true "정책 ID"
// @Security BearerAuth
// @Success 200 {object} security.SecurityPolicyResponse "정책 정보"
// @Failure 401 {object} models.ErrorResponse "인증 실패"
// @Failure 403 {object} models.ErrorResponse "권한 없음"
// @Failure 404 {object} models.ErrorResponse "정책 없음"
// @Router /admin/policies/{id} [get]
func (pc *PolicyController) GetPolicy(c *gin.Context) {
	policyID := c.Param("id")
	if policyID == "" {
		middleware.BadRequestError(c, "정책 ID가 필요합니다", nil)
		return
	}

	policy, err := pc.policyService.GetPolicy(c.Request.Context(), policyID)
	if err != nil {
		middleware.NotFoundError(c, "정책을 찾을 수 없습니다", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    policy.ToResponse(),
	})
}

// UpdatePolicy는 보안 정책을 업데이트합니다.
// @Summary 보안 정책 업데이트
// @Description 보안 정책을 업데이트합니다
// @Tags security,policies
// @Accept json
// @Produce json
// @Param id path string true "정책 ID"
// @Param request body security.UpdatePolicyRequest true "정책 업데이트 요청"
// @Security BearerAuth
// @Success 200 {object} security.SecurityPolicyResponse "업데이트된 정책"
// @Failure 400 {object} models.ErrorResponse "잘못된 요청"
// @Failure 401 {object} models.ErrorResponse "인증 실패"
// @Failure 403 {object} models.ErrorResponse "권한 없음"
// @Failure 404 {object} models.ErrorResponse "정책 없음"
// @Router /admin/policies/{id} [put]
func (pc *PolicyController) UpdatePolicy(c *gin.Context) {
	policyID := c.Param("id")
	if policyID == "" {
		middleware.BadRequestError(c, "정책 ID가 필요합니다", nil)
		return
	}

	var req security.UpdatePolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.BadRequestError(c, "잘못된 요청 형식", err)
		return
	}

	policy, err := pc.policyService.UpdatePolicy(c.Request.Context(), policyID, &req)
	if err != nil {
		middleware.BadRequestError(c, "정책 업데이트 실패", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    policy.ToResponse(),
		"message": "보안 정책이 성공적으로 업데이트되었습니다",
	})
}

// DeletePolicy는 보안 정책을 삭제합니다.
// @Summary 보안 정책 삭제
// @Description 보안 정책을 삭제합니다
// @Tags security,policies
// @Accept json
// @Produce json
// @Param id path string true "정책 ID"
// @Security BearerAuth
// @Success 200 {object} gin.H "성공 메시지"
// @Failure 400 {object} models.ErrorResponse "잘못된 요청"
// @Failure 401 {object} models.ErrorResponse "인증 실패"
// @Failure 403 {object} models.ErrorResponse "권한 없음"
// @Failure 404 {object} models.ErrorResponse "정책 없음"
// @Router /admin/policies/{id} [delete]
func (pc *PolicyController) DeletePolicy(c *gin.Context) {
	policyID := c.Param("id")
	if policyID == "" {
		middleware.BadRequestError(c, "정책 ID가 필요합니다", nil)
		return
	}

	err := pc.policyService.DeletePolicy(c.Request.Context(), policyID)
	if err != nil {
		middleware.BadRequestError(c, "정책 삭제 실패", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "보안 정책이 성공적으로 삭제되었습니다",
	})
}

// ListPolicies는 보안 정책 목록을 조회합니다.
// @Summary 보안 정책 목록 조회
// @Description 보안 정책 목록을 조회합니다
// @Tags security,policies
// @Accept json
// @Produce json
// @Param page query int false "페이지 번호" default(1)
// @Param limit query int false "페이지당 항목 수" default(20)
// @Param category query string false "카테고리 필터"
// @Param is_active query bool false "활성 상태 필터"
// @Param search query string false "검색어"
// @Security BearerAuth
// @Success 200 {object} models.PaginatedResponse "정책 목록"
// @Failure 401 {object} models.ErrorResponse "인증 실패"
// @Failure 403 {object} models.ErrorResponse "권한 없음"
// @Router /admin/policies [get]
func (pc *PolicyController) ListPolicies(c *gin.Context) {
	// 페이지네이션 파라미터 파싱
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	category := c.Query("category")
	search := c.Query("search")
	
	var isActive *bool
	if activeStr := c.Query("is_active"); activeStr != "" {
		if activeVal, err := strconv.ParseBool(activeStr); err == nil {
			isActive = &activeVal
		}
	}

	filter := &security.PolicyFilter{
		Category: category,
		IsActive: isActive,
		Search:   search,
		PaginationRequest: models.PaginationRequest{
			Page:  page,
			Limit: limit,
		},
	}

	policies, err := pc.policyService.ListPolicies(c.Request.Context(), filter)
	if err != nil {
		middleware.InternalServerError(c, "정책 목록 조회 실패", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    policies,
	})
}

// ApplyPolicy는 정책을 적용합니다.
// @Summary 정책 적용
// @Description 정책을 시스템에 적용합니다
// @Tags security,policies
// @Accept json
// @Produce json
// @Param id path string true "정책 ID"
// @Security BearerAuth
// @Success 200 {object} gin.H "성공 메시지"
// @Failure 400 {object} models.ErrorResponse "잘못된 요청"
// @Failure 401 {object} models.ErrorResponse "인증 실패"
// @Failure 403 {object} models.ErrorResponse "권한 없음"
// @Router /admin/policies/{id}/apply [post]
func (pc *PolicyController) ApplyPolicy(c *gin.Context) {
	policyID := c.Param("id")
	if policyID == "" {
		middleware.BadRequestError(c, "정책 ID가 필요합니다", nil)
		return
	}

	err := pc.policyService.ApplyPolicy(c.Request.Context(), policyID)
	if err != nil {
		middleware.BadRequestError(c, "정책 적용 실패", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "정책이 성공적으로 적용되었습니다",
	})
}

// DeactivatePolicy는 정책을 비활성화합니다.
// @Summary 정책 비활성화
// @Description 정책을 비활성화합니다
// @Tags security,policies
// @Accept json
// @Produce json
// @Param id path string true "정책 ID"
// @Security BearerAuth
// @Success 200 {object} gin.H "성공 메시지"
// @Failure 400 {object} models.ErrorResponse "잘못된 요청"
// @Failure 401 {object} models.ErrorResponse "인증 실패"
// @Failure 403 {object} models.ErrorResponse "권한 없음"
// @Router /admin/policies/{id}/deactivate [post]
func (pc *PolicyController) DeactivatePolicy(c *gin.Context) {
	policyID := c.Param("id")
	if policyID == "" {
		middleware.BadRequestError(c, "정책 ID가 필요합니다", nil)
		return
	}

	err := pc.policyService.DeactivatePolicy(c.Request.Context(), policyID)
	if err != nil {
		middleware.BadRequestError(c, "정책 비활성화 실패", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "정책이 성공적으로 비활성화되었습니다",
	})
}

// RollbackPolicy는 정책을 이전 버전으로 롤백합니다.
// @Summary 정책 롤백
// @Description 정책을 이전 버전으로 롤백합니다
// @Tags security,policies
// @Accept json
// @Produce json
// @Param id path string true "정책 ID"
// @Param version query string true "롤백할 버전"
// @Security BearerAuth
// @Success 200 {object} gin.H "성공 메시지"
// @Failure 400 {object} models.ErrorResponse "잘못된 요청"
// @Failure 401 {object} models.ErrorResponse "인증 실패"
// @Failure 403 {object} models.ErrorResponse "권한 없음"
// @Router /admin/policies/{id}/rollback [post]
func (pc *PolicyController) RollbackPolicy(c *gin.Context) {
	policyID := c.Param("id")
	version := c.Query("version")
	
	if policyID == "" {
		middleware.BadRequestError(c, "정책 ID가 필요합니다", nil)
		return
	}
	
	if version == "" {
		middleware.BadRequestError(c, "롤백할 버전이 필요합니다", nil)
		return
	}

	err := pc.policyService.RollbackPolicy(c.Request.Context(), policyID, version)
	if err != nil {
		middleware.BadRequestError(c, "정책 롤백 실패", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "정책이 성공적으로 롤백되었습니다",
	})
}

// GetActivePolicies는 활성 정책 목록을 조회합니다.
// @Summary 활성 정책 목록 조회
// @Description 현재 활성화된 정책 목록을 조회합니다
// @Tags security,policies
// @Accept json
// @Produce json
// @Param category query string false "카테고리 필터"
// @Security BearerAuth
// @Success 200 {object} []security.SecurityPolicyResponse "활성 정책 목록"
// @Failure 401 {object} models.ErrorResponse "인증 실패"
// @Failure 403 {object} models.ErrorResponse "권한 없음"
// @Router /admin/policies/active [get]
func (pc *PolicyController) GetActivePolicies(c *gin.Context) {
	category := c.Query("category")

	policies, err := pc.policyService.GetActivePolicies(c.Request.Context(), category)
	if err != nil {
		middleware.InternalServerError(c, "활성 정책 조회 실패", err)
		return
	}

	// 응답 형태로 변환
	responses := make([]*security.SecurityPolicyResponse, len(policies))
	for i, policy := range policies {
		responses[i] = policy.ToResponse()
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    responses,
	})
}

// ValidatePolicy는 정책을 검증합니다.
// @Summary 정책 검증
// @Description 정책의 유효성을 검증합니다
// @Tags security,policies
// @Accept json
// @Produce json
// @Param id path string true "정책 ID"
// @Security BearerAuth
// @Success 200 {object} security.ValidationResult "검증 결과"
// @Failure 400 {object} models.ErrorResponse "잘못된 요청"
// @Failure 401 {object} models.ErrorResponse "인증 실패"
// @Failure 403 {object} models.ErrorResponse "권한 없음"
// @Router /admin/policies/{id}/validate [post]
func (pc *PolicyController) ValidatePolicy(c *gin.Context) {
	policyID := c.Param("id")
	if policyID == "" {
		middleware.BadRequestError(c, "정책 ID가 필요합니다", nil)
		return
	}

	// 정책 조회
	policy, err := pc.policyService.GetPolicy(c.Request.Context(), policyID)
	if err != nil {
		middleware.NotFoundError(c, "정책을 찾을 수 없습니다", err)
		return
	}

	// 검증 실행
	result, err := pc.policyService.ValidatePolicy(c.Request.Context(), policy)
	if err != nil {
		middleware.InternalServerError(c, "정책 검증 실패", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

// TestPolicy는 정책을 테스트합니다.
// @Summary 정책 테스트
// @Description 정책을 테스트 데이터로 테스트합니다
// @Tags security,policies
// @Accept json
// @Produce json
// @Param id path string true "정책 ID"
// @Param test_data body interface{} true "테스트 데이터"
// @Security BearerAuth
// @Success 200 {object} security.TestResult "테스트 결과"
// @Failure 400 {object} models.ErrorResponse "잘못된 요청"
// @Failure 401 {object} models.ErrorResponse "인증 실패"
// @Failure 403 {object} models.ErrorResponse "권한 없음"
// @Router /admin/policies/{id}/test [post]
func (pc *PolicyController) TestPolicy(c *gin.Context) {
	policyID := c.Param("id")
	if policyID == "" {
		middleware.BadRequestError(c, "정책 ID가 필요합니다", nil)
		return
	}

	var testData interface{}
	if err := c.ShouldBindJSON(&testData); err != nil {
		middleware.BadRequestError(c, "잘못된 테스트 데이터 형식", err)
		return
	}

	result, err := pc.policyService.TestPolicy(c.Request.Context(), policyID, testData)
	if err != nil {
		middleware.InternalServerError(c, "정책 테스트 실패", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

// GetPolicyHistory는 정책 히스토리를 조회합니다.
// @Summary 정책 히스토리 조회
// @Description 정책의 변경 히스토리를 조회합니다
// @Tags security,policies
// @Accept json
// @Produce json
// @Param id path string true "정책 ID"
// @Security BearerAuth
// @Success 200 {object} []security.PolicyAuditEntry "정책 히스토리"
// @Failure 401 {object} models.ErrorResponse "인증 실패"
// @Failure 403 {object} models.ErrorResponse "권한 없음"
// @Failure 404 {object} models.ErrorResponse "정책 없음"
// @Router /admin/policies/{id}/history [get]
func (pc *PolicyController) GetPolicyHistory(c *gin.Context) {
	policyID := c.Param("id")
	if policyID == "" {
		middleware.BadRequestError(c, "정책 ID가 필요합니다", nil)
		return
	}

	history, err := pc.policyService.GetPolicyHistory(c.Request.Context(), policyID)
	if err != nil {
		middleware.InternalServerError(c, "정책 히스토리 조회 실패", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    history,
	})
}

// GetPolicyAuditLog는 정책 감사 로그를 조회합니다.
// @Summary 정책 감사 로그 조회
// @Description 정책 관련 감사 로그를 조회합니다
// @Tags security,policies
// @Accept json
// @Produce json
// @Param page query int false "페이지 번호" default(1)
// @Param limit query int false "페이지당 항목 수" default(20)
// @Param policy_id query string false "정책 ID 필터"
// @Param action query string false "액션 필터"
// @Param user_id query string false "사용자 ID 필터"
// @Security BearerAuth
// @Success 200 {object} models.PaginatedResponse "감사 로그"
// @Failure 401 {object} models.ErrorResponse "인증 실패"
// @Failure 403 {object} models.ErrorResponse "권한 없음"
// @Router /admin/policies/audit [get]
func (pc *PolicyController) GetPolicyAuditLog(c *gin.Context) {
	// 페이지네이션 파라미터 파싱
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	policyID := c.Query("policy_id")
	action := c.Query("action")
	userID := c.Query("user_id")

	filter := &security.AuditFilter{
		PolicyID: policyID,
		Action:   action,
		UserID:   userID,
		PaginationRequest: models.PaginationRequest{
			Page:  page,
			Limit: limit,
		},
	}

	auditLog, err := pc.policyService.GetPolicyAuditLog(c.Request.Context(), filter)
	if err != nil {
		middleware.InternalServerError(c, "감사 로그 조회 실패", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    auditLog,
	})
}

// ===== 정책 템플릿 관련 API =====

// CreateTemplate은 정책 템플릿을 생성합니다.
// @Summary 정책 템플릿 생성
// @Description 새로운 정책 템플릿을 생성합니다
// @Tags security,policy-templates
// @Accept json
// @Produce json
// @Param request body security.CreateTemplateRequest true "템플릿 생성 요청"
// @Security BearerAuth
// @Success 201 {object} security.PolicyTemplate "생성된 템플릿"
// @Failure 400 {object} models.ErrorResponse "잘못된 요청"
// @Failure 401 {object} models.ErrorResponse "인증 실패"
// @Failure 403 {object} models.ErrorResponse "권한 없음"
// @Router /admin/policy-templates [post]
func (pc *PolicyController) CreateTemplate(c *gin.Context) {
	var req security.CreateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.BadRequestError(c, "잘못된 요청 형식", err)
		return
	}

	template, err := pc.policyService.CreateTemplate(c.Request.Context(), &req)
	if err != nil {
		middleware.BadRequestError(c, "템플릿 생성 실패", err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    template,
		"message": "정책 템플릿이 성공적으로 생성되었습니다",
	})
}

// GetTemplate은 정책 템플릿을 조회합니다.
// @Summary 정책 템플릿 조회
// @Description 특정 정책 템플릿을 조회합니다
// @Tags security,policy-templates
// @Accept json
// @Produce json
// @Param id path string true "템플릿 ID"
// @Security BearerAuth
// @Success 200 {object} security.PolicyTemplate "템플릿 정보"
// @Failure 401 {object} models.ErrorResponse "인증 실패"
// @Failure 403 {object} models.ErrorResponse "권한 없음"
// @Failure 404 {object} models.ErrorResponse "템플릿 없음"
// @Router /admin/policy-templates/{id} [get]
func (pc *PolicyController) GetTemplate(c *gin.Context) {
	templateID := c.Param("id")
	if templateID == "" {
		middleware.BadRequestError(c, "템플릿 ID가 필요합니다", nil)
		return
	}

	template, err := pc.policyService.GetTemplate(c.Request.Context(), templateID)
	if err != nil {
		middleware.NotFoundError(c, "템플릿을 찾을 수 없습니다", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    template,
	})
}

// ListTemplates는 정책 템플릿 목록을 조회합니다.
// @Summary 정책 템플릿 목록 조회
// @Description 정책 템플릿 목록을 조회합니다
// @Tags security,policy-templates
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} []security.PolicyTemplate "템플릿 목록"
// @Failure 401 {object} models.ErrorResponse "인증 실패"
// @Failure 403 {object} models.ErrorResponse "권한 없음"
// @Router /admin/policy-templates [get]
func (pc *PolicyController) ListTemplates(c *gin.Context) {
	templates, err := pc.policyService.ListTemplates(c.Request.Context())
	if err != nil {
		middleware.InternalServerError(c, "템플릿 목록 조회 실패", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    templates,
	})
}

// CreatePolicyFromTemplate은 템플릿으로부터 정책을 생성합니다.
// @Summary 템플릿으로부터 정책 생성
// @Description 기존 템플릿을 사용하여 새로운 정책을 생성합니다
// @Tags security,policy-templates
// @Accept json
// @Produce json
// @Param id path string true "템플릿 ID"
// @Param request body security.CreateFromTemplateRequest true "템플릿으로부터 생성 요청"
// @Security BearerAuth
// @Success 201 {object} security.SecurityPolicyResponse "생성된 정책"
// @Failure 400 {object} models.ErrorResponse "잘못된 요청"
// @Failure 401 {object} models.ErrorResponse "인증 실패"
// @Failure 403 {object} models.ErrorResponse "권한 없음"
// @Router /admin/policy-templates/{id}/create-policy [post]
func (pc *PolicyController) CreatePolicyFromTemplate(c *gin.Context) {
	templateID := c.Param("id")
	if templateID == "" {
		middleware.BadRequestError(c, "템플릿 ID가 필요합니다", nil)
		return
	}

	var req security.CreateFromTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.BadRequestError(c, "잘못된 요청 형식", err)
		return
	}

	policy, err := pc.policyService.CreatePolicyFromTemplate(c.Request.Context(), templateID, &req)
	if err != nil {
		middleware.BadRequestError(c, "템플릿으로부터 정책 생성 실패", err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    policy.ToResponse(),
		"message": "템플릿으로부터 정책이 성공적으로 생성되었습니다",
	})
}