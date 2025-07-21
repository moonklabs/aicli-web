package controllers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"aicli-web/internal/auth"
	"aicli-web/internal/middleware"
	"aicli-web/internal/models"
	"aicli-web/internal/storage"
	"aicli-web/internal/utils"
)

// WorkspaceController는 워크스페이스 관련 API를 처리합니다.
type WorkspaceController struct {
	storage storage.Storage
}

// NewWorkspaceController는 새로운 워크스페이스 컨트롤러를 생성합니다.
func NewWorkspaceController(storage storage.Storage) *WorkspaceController {
	return &WorkspaceController{
		storage: storage,
	}
}


// ListWorkspaces는 워크스페이스 목록을 조회합니다.
// @Summary 워크스페이스 목록 조회
// @Description 사용자의 모든 워크스페이스 목록을 페이지네이션과 함께 조회합니다
// @Tags workspaces
// @Accept json
// @Produce json
// @Param page query int false "페이지 번호" default(1)
// @Param limit query int false "페이지당 항목 수" default(10)
// @Param sort query string false "정렬 기준 (name, created_at, updated_at)" default("created_at")
// @Param order query string false "정렬 순서 (asc, desc)" default("desc")
// @Security BearerAuth
// @Success 200 {object} models.PaginationResponse "워크스페이스 목록"
// @Failure 401 {object} models.ErrorResponse "인증 실패"
// @Router /workspaces [get]
func (wc *WorkspaceController) ListWorkspaces(c *gin.Context) {
	// 사용자 정보 가져오기
	claims, exists := c.Get("claims")
	if !exists {
		middleware.UnauthorizedError(c, "인증 정보를 찾을 수 없습니다")
		return
	}
	userClaims := claims.(*auth.Claims)
	
	// 페이지네이션 요청 처리
	var req models.PaginationRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		middleware.ValidationError(c, "잘못된 요청 파라미터", err.Error())
		return
	}
	
	// 기본값 설정
	if req.Page < 1 {
		req.Page = 1
	}
	if req.Limit < 1 {
		req.Limit = 10
	}
	if req.Sort == "" {
		req.Sort = "created_at"
	}
	if req.Order == "" {
		req.Order = "desc"
	}
	
	// 워크스페이스 목록 조회
	workspaces, total, err := wc.storage.Workspace().GetByOwnerID(c, userClaims.UserID, &req)
	if err != nil {
		middleware.InternalError(c, "워크스페이스 목록 조회 실패", err)
		return
	}
	
	// 응답 생성
	response := models.PaginationResponse{
		SuccessResponse: models.SuccessResponse{
			Success: true,
			Message: "워크스페이스 목록을 조회했습니다",
		},
		Data: workspaces,
		Meta: models.PaginationMeta{
			Page:       req.Page,
			Limit:      req.Limit,
			Total:      total,
			TotalPages: (total + req.Limit - 1) / req.Limit,
		},
	}
	
	c.JSON(http.StatusOK, response)
}

// CreateWorkspace는 새 워크스페이스를 생성합니다.
// @Summary 워크스페이스 생성
// @Description 새로운 워크스페이스를 생성합니다
// @Tags workspaces
// @Accept json
// @Produce json
// @Param body body models.Workspace true "워크스페이스 생성 요청"
// @Security BearerAuth
// @Success 201 {object} models.SuccessResponse "생성된 워크스페이스"
// @Failure 400 {object} models.ErrorResponse "잘못된 요청"
// @Failure 401 {object} models.ErrorResponse "인증 실패"
// @Failure 409 {object} models.ErrorResponse "이미 존재하는 워크스페이스"
// @Router /workspaces [post]
func (wc *WorkspaceController) CreateWorkspace(c *gin.Context) {
	// 사용자 정보 가져오기
	claims, exists := c.Get("claims")
	if !exists {
		middleware.UnauthorizedError(c, "인증 정보를 찾을 수 없습니다")
		return
	}
	userClaims := claims.(*auth.Claims)
	
	// 요청 데이터 파싱
	var workspace models.Workspace
	if err := c.ShouldBindJSON(&workspace); err != nil {
		middleware.ValidationError(c, "요청 데이터가 올바르지 않습니다", err.Error())
		return
	}
	
	// 프로젝트 경로 유효성 검사
	if err := utils.IsValidProjectPath(workspace.ProjectPath); err != nil {
		middleware.ValidationError(c, "프로젝트 경로가 유효하지 않습니다", err.Error())
		return
	}
	
	// 소유자 정보 설정
	workspace.OwnerID = userClaims.UserID
	
	// 중복 확인
	exists, err := wc.storage.Workspace().ExistsByName(c, workspace.OwnerID, workspace.Name)
	if err != nil {
		middleware.InternalError(c, "워크스페이스 중복 확인 실패", err)
		return
	}
	if exists {
		middleware.ConflictError(c, "이미 존재하는 워크스페이스 이름입니다")
		return
	}
	
	// 워크스페이스 생성
	if err := wc.storage.Workspace().Create(c, &workspace); err != nil {
		if err == storage.ErrAlreadyExists {
			middleware.ConflictError(c, "이미 존재하는 워크스페이스입니다")
			return
		}
		middleware.InternalError(c, "워크스페이스 생성 실패", err)
		return
	}
	
	// 성공 응답
	response := models.SuccessResponse{
		Success: true,
		Message: "워크스페이스가 생성되었습니다",
		Data:    workspace,
	}
	
	c.JSON(http.StatusCreated, response)
}

// GetWorkspace는 특정 워크스페이스를 조회합니다.
// @Summary 워크스페이스 상세 조회
// @Description ID로 특정 워크스페이스의 상세 정보를 조회합니다
// @Tags workspaces
// @Accept json
// @Produce json
// @Param id path string true "워크스페이스 ID"
// @Security BearerAuth
// @Success 200 {object} models.SuccessResponse "워크스페이스 정보"
// @Failure 400 {object} models.ErrorResponse "잘못된 요청"
// @Failure 401 {object} models.ErrorResponse "인증 실패"
// @Failure 404 {object} models.ErrorResponse "워크스페이스를 찾을 수 없음"
// @Router /workspaces/{id} [get]
func (wc *WorkspaceController) GetWorkspace(c *gin.Context) {
	// 사용자 정보 가져오기
	claims, exists := c.Get("claims")
	if !exists {
		middleware.UnauthorizedError(c, "인증 정보를 찾을 수 없습니다")
		return
	}
	userClaims := claims.(*auth.Claims)
	
	id := c.Param("id")
	if id == "" {
		middleware.ValidationError(c, "워크스페이스 ID가 필요합니다", nil)
		return
	}
	
	// 워크스페이스 조회
	workspace, err := wc.storage.Workspace().GetByID(c, id)
	if err != nil {
		if err == storage.ErrNotFound {
			middleware.NotFoundError(c, "워크스페이스를 찾을 수 없습니다")
			return
		}
		middleware.InternalError(c, "워크스페이스 조회 실패", err)
		return
	}
	
	// 권한 확인
	if workspace.OwnerID != userClaims.UserID {
		middleware.ForbiddenError(c, "워크스페이스에 접근할 권한이 없습니다")
		return
	}
	
	// 성공 응답
	response := models.SuccessResponse{
		Success: true,
		Data:    workspace,
	}
	
	c.JSON(http.StatusOK, response)
}

// UpdateWorkspace는 워크스페이스를 수정합니다.
// @Summary 워크스페이스 수정
// @Description 워크스페이스 정보를 수정합니다
// @Tags workspaces
// @Accept json
// @Produce json
// @Param id path string true "워크스페이스 ID"
// @Param body body map[string]interface{} true "워크스페이스 수정 요청"
// @Security BearerAuth
// @Success 200 {object} models.SuccessResponse "수정된 워크스페이스"
// @Failure 400 {object} models.ErrorResponse "잘못된 요청"
// @Failure 401 {object} models.ErrorResponse "인증 실패"
// @Failure 404 {object} models.ErrorResponse "워크스페이스를 찾을 수 없음"
// @Failure 409 {object} models.ErrorResponse "이미 존재하는 워크스페이스 이름"
// @Router /workspaces/{id} [put]
func (wc *WorkspaceController) UpdateWorkspace(c *gin.Context) {
	// 사용자 정보 가져오기
	claims, exists := c.Get("claims")
	if !exists {
		middleware.UnauthorizedError(c, "인증 정보를 찾을 수 없습니다")
		return
	}
	userClaims := claims.(*auth.Claims)
	
	id := c.Param("id")
	if id == "" {
		middleware.ValidationError(c, "워크스페이스 ID가 필요합니다", nil)
		return
	}
	
	// 요청 데이터 파싱
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		middleware.ValidationError(c, "요청 데이터가 올바르지 않습니다", err.Error())
		return
	}
	
	// 워크스페이스 존재 및 권한 확인
	workspace, err := wc.storage.Workspace().GetByID(c, id)
	if err != nil {
		if err == storage.ErrNotFound {
			middleware.NotFoundError(c, "워크스페이스를 찾을 수 없습니다")
			return
		}
		middleware.InternalError(c, "워크스페이스 조회 실패", err)
		return
	}
	
	if workspace.OwnerID != userClaims.UserID {
		middleware.ForbiddenError(c, "워크스페이스를 수정할 권한이 없습니다")
		return
	}
	
	// 프로젝트 경로 유효성 검사 (변경되는 경우)
	if projectPath, ok := updates["project_path"].(string); ok {
		if err := utils.IsValidProjectPath(projectPath); err != nil {
			middleware.ValidationError(c, "프로젝트 경로가 유효하지 않습니다", err.Error())
			return
		}
	}
	
	// 워크스페이스 업데이트
	if err := wc.storage.Workspace().Update(c, id, updates); err != nil {
		if err == storage.ErrAlreadyExists {
			middleware.ConflictError(c, "이미 존재하는 워크스페이스 이름입니다")
			return
		}
		if err == storage.ErrNotFound {
			middleware.NotFoundError(c, "워크스페이스를 찾을 수 없습니다")
			return
		}
		middleware.InternalError(c, "워크스페이스 수정 실패", err)
		return
	}
	
	// 수정된 워크스페이스 조회
	updatedWorkspace, err := wc.storage.Workspace().GetByID(c, id)
	if err != nil {
		middleware.InternalError(c, "수정된 워크스페이스 조회 실패", err)
		return
	}
	
	// 성공 응답
	response := models.SuccessResponse{
		Success: true,
		Message: "워크스페이스가 수정되었습니다",
		Data:    updatedWorkspace,
	}
	
	c.JSON(http.StatusOK, response)
}

// DeleteWorkspace는 워크스페이스를 삭제합니다.
// @Summary 워크스페이스 삭제
// @Description 워크스페이스와 관련된 모든 데이터를 삭제합니다
// @Tags workspaces
// @Accept json
// @Produce json
// @Param id path string true "워크스페이스 ID"
// @Security BearerAuth
// @Success 200 {object} models.SuccessResponse "삭제 성공"
// @Failure 400 {object} models.ErrorResponse "잘못된 요청"
// @Failure 401 {object} models.ErrorResponse "인증 실패"
// @Failure 404 {object} models.ErrorResponse "워크스페이스를 찾을 수 없음"
// @Router /workspaces/{id} [delete]
func (wc *WorkspaceController) DeleteWorkspace(c *gin.Context) {
	// 사용자 정보 가져오기
	claims, exists := c.Get("claims")
	if !exists {
		middleware.UnauthorizedError(c, "인증 정보를 찾을 수 없습니다")
		return
	}
	userClaims := claims.(*auth.Claims)
	
	id := c.Param("id")
	if id == "" {
		middleware.ValidationError(c, "워크스페이스 ID가 필요합니다", nil)
		return
	}
	
	// 워크스페이스 존재 및 권한 확인
	workspace, err := wc.storage.Workspace().GetByID(c, id)
	if err != nil {
		if err == storage.ErrNotFound {
			middleware.NotFoundError(c, "워크스페이스를 찾을 수 없습니다")
			return
		}
		middleware.InternalError(c, "워크스페이스 조회 실패", err)
		return
	}
	
	if workspace.OwnerID != userClaims.UserID {
		middleware.ForbiddenError(c, "워크스페이스를 삭제할 권한이 없습니다")
		return
	}
	
	// TODO: 실행 중인 태스크 확인 및 정리
	// TODO: Docker 컨테이너 제거
	
	// 워크스페이스 삭제 (Soft Delete)
	if err := wc.storage.Workspace().Delete(c, id); err != nil {
		if err == storage.ErrNotFound {
			middleware.NotFoundError(c, "워크스페이스를 찾을 수 없습니다")
			return
		}
		middleware.InternalError(c, "워크스페이스 삭제 실패", err)
		return
	}
	
	// 성공 응답
	response := models.SuccessResponse{
		Success: true,
		Message: "워크스페이스가 삭제되었습니다",
		Data: gin.H{
			"id": id,
		},
	}
	
	c.JSON(http.StatusOK, response)
}