package controllers

import (
	"net/http"

	"github.com/aicli/aicli-web/internal/auth"
	"github.com/aicli/aicli-web/internal/middleware"
	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/services"
	"github.com/aicli/aicli-web/internal/storage"
	"github.com/aicli/aicli-web/internal/utils"
	"github.com/gin-gonic/gin"
)

// ProjectController는 프로젝트 관련 API를 처리합니다.
type ProjectController struct {
	projectService *services.ProjectService
	storage        storage.Storage
}

// NewProjectController는 새로운 프로젝트 컨트롤러를 생성합니다.
func NewProjectController(storage storage.Storage) *ProjectController {
	return &ProjectController{
		projectService: services.NewProjectService(storage),
		storage:        storage,
	}
}

// CreateProject는 새 프로젝트를 생성합니다.
// @Summary 프로젝트 생성
// @Description 워크스페이스 내에 새로운 프로젝트를 생성합니다
// @Tags projects
// @Accept json
// @Produce json
// @Param id path string true "워크스페이스 ID"
// @Param body body models.Project true "프로젝트 생성 요청"
// @Security BearerAuth
// @Success 201 {object} models.SuccessResponse "생성된 프로젝트"
// @Failure 400 {object} models.ErrorResponse "잘못된 요청"
// @Failure 401 {object} models.ErrorResponse "인증 실패"
// @Failure 404 {object} models.ErrorResponse "워크스페이스를 찾을 수 없음"
// @Failure 409 {object} models.ErrorResponse "이미 존재하는 프로젝트"
// @Router /workspaces/{id}/projects [post]
func (pc *ProjectController) CreateProject(c *gin.Context) {
	// 사용자 정보 가져오기
	claims, exists := c.Get("claims")
	if !exists {
		middleware.UnauthorizedError(c, "인증 정보를 찾을 수 없습니다")
		return
	}
	userClaims := claims.(*auth.Claims)
	
	// 워크스페이스 ID 가져오기
	workspaceID := c.Param("id")
	if workspaceID == "" {
		middleware.ValidationError(c, "워크스페이스 ID가 필요합니다", nil)
		return
	}
	
	// 워크스페이스 소유권 확인
	workspace, err := pc.storage.Workspace().GetByID(c, workspaceID)
	if err != nil {
		if err == storage.ErrNotFound {
			middleware.NotFoundError(c, "워크스페이스를 찾을 수 없습니다")
			return
		}
		middleware.InternalError(c, "워크스페이스 조회 실패", err)
		return
	}
	
	if workspace.OwnerID != userClaims.UserID {
		middleware.ForbiddenError(c, "워크스페이스에 프로젝트를 생성할 권한이 없습니다")
		return
	}
	
	// 요청 데이터 파싱
	var req models.CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.ValidationError(c, "요청 데이터가 올바르지 않습니다", err.Error())
		return
	}
	
	// 프로젝트 모델 생성
	project := models.Project{
		WorkspaceID: workspaceID,
		Name:        req.Name,
		Path:        req.Path,
		Description: req.Description,
		GitURL:      req.GitURL,
		GitBranch:   req.GitBranch,
		Language:    req.Language,
		Status:      models.ProjectStatusActive,
	}
	
	// 프로젝트 경로 유효성 검사
	if err := utils.IsValidProjectPath(project.Path); err != nil {
		middleware.ValidationError(c, "프로젝트 경로가 유효하지 않습니다", err.Error())
		return
	}
	
	// 프로젝트 생성
	if err := pc.projectService.CreateProject(c, &project); err != nil {
		if err.Error() == "project name already exists in workspace" {
			middleware.ConflictError(c, "이미 존재하는 프로젝트 이름입니다")
			return
		}
		if err.Error() == "project path already in use" {
			middleware.ConflictError(c, "이미 사용 중인 프로젝트 경로입니다")
			return
		}
		middleware.InternalError(c, "프로젝트 생성 실패", err)
		return
	}
	
	// 성공 응답
	response := models.SuccessResponse{
		Success: true,
		Message: "프로젝트가 생성되었습니다",
		Data:    project,
	}
	
	c.JSON(http.StatusCreated, response)
}

// ListProjects는 워크스페이스의 프로젝트 목록을 조회합니다.
// @Summary 프로젝트 목록 조회
// @Description 워크스페이스의 모든 프로젝트 목록을 페이지네이션과 함께 조회합니다
// @Tags projects
// @Accept json
// @Produce json
// @Param id path string true "워크스페이스 ID"
// @Param page query int false "페이지 번호" default(1)
// @Param limit query int false "페이지당 항목 수" default(10)
// @Param sort query string false "정렬 기준 (name, created_at, updated_at)" default("created_at")
// @Param order query string false "정렬 순서 (asc, desc)" default("desc")
// @Security BearerAuth
// @Success 200 {object} models.PaginationResponse "프로젝트 목록"
// @Failure 401 {object} models.ErrorResponse "인증 실패"
// @Failure 404 {object} models.ErrorResponse "워크스페이스를 찾을 수 없음"
// @Router /workspaces/{id}/projects [get]
func (pc *ProjectController) ListProjects(c *gin.Context) {
	// 사용자 정보 가져오기
	claims, exists := c.Get("claims")
	if !exists {
		middleware.UnauthorizedError(c, "인증 정보를 찾을 수 없습니다")
		return
	}
	userClaims := claims.(*auth.Claims)
	
	// 워크스페이스 ID 가져오기
	workspaceID := c.Param("id")
	if workspaceID == "" {
		middleware.ValidationError(c, "워크스페이스 ID가 필요합니다", nil)
		return
	}
	
	// 워크스페이스 소유권 확인
	workspace, err := pc.storage.Workspace().GetByID(c, workspaceID)
	if err != nil {
		if err == storage.ErrNotFound {
			middleware.NotFoundError(c, "워크스페이스를 찾을 수 없습니다")
			return
		}
		middleware.InternalError(c, "워크스페이스 조회 실패", err)
		return
	}
	
	if workspace.OwnerID != userClaims.UserID {
		middleware.ForbiddenError(c, "워크스페이스의 프로젝트를 조회할 권한이 없습니다")
		return
	}
	
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
	
	// 프로젝트 목록 조회
	projects, total, err := pc.projectService.GetProjectsByWorkspace(c, workspaceID, &req)
	if err != nil {
		middleware.InternalError(c, "프로젝트 목록 조회 실패", err)
		return
	}
	
	// 응답 생성
	response := models.PaginationResponse{
		Data: projects,
		Meta: models.PaginationMeta{
			CurrentPage: req.Page,
			PerPage:     req.Limit,
			Total:       total,
			TotalPages:  (total + req.Limit - 1) / req.Limit,
			HasNext:     req.Page < (total+req.Limit-1)/req.Limit,
			HasPrev:     req.Page > 1,
		},
	}
	
	c.JSON(http.StatusOK, response)
}

// GetProject는 특정 프로젝트를 조회합니다.
// @Summary 프로젝트 상세 조회
// @Description ID로 특정 프로젝트의 상세 정보를 조회합니다
// @Tags projects
// @Accept json
// @Produce json
// @Param id path string true "프로젝트 ID"
// @Security BearerAuth
// @Success 200 {object} models.SuccessResponse "프로젝트 정보"
// @Failure 400 {object} models.ErrorResponse "잘못된 요청"
// @Failure 401 {object} models.ErrorResponse "인증 실패"
// @Failure 404 {object} models.ErrorResponse "프로젝트를 찾을 수 없음"
// @Router /projects/{id} [get]
func (pc *ProjectController) GetProject(c *gin.Context) {
	// 사용자 정보 가져오기
	claims, exists := c.Get("claims")
	if !exists {
		middleware.UnauthorizedError(c, "인증 정보를 찾을 수 없습니다")
		return
	}
	userClaims := claims.(*auth.Claims)
	
	id := c.Param("id")
	if id == "" {
		middleware.ValidationError(c, "프로젝트 ID가 필요합니다", nil)
		return
	}
	
	// 프로젝트 조회
	project, err := pc.projectService.GetProject(c, id)
	if err != nil {
		if err == storage.ErrNotFound {
			middleware.NotFoundError(c, "프로젝트를 찾을 수 없습니다")
			return
		}
		middleware.InternalError(c, "프로젝트 조회 실패", err)
		return
	}
	
	// 워크스페이스 소유권 확인
	workspace, err := pc.storage.Workspace().GetByID(c, project.WorkspaceID)
	if err != nil {
		middleware.InternalError(c, "워크스페이스 조회 실패", err)
		return
	}
	
	if workspace.OwnerID != userClaims.UserID {
		middleware.ForbiddenError(c, "프로젝트에 접근할 권한이 없습니다")
		return
	}
	
	// 성공 응답
	response := models.SuccessResponse{
		Success: true,
		Data:    project,
	}
	
	c.JSON(http.StatusOK, response)
}

// UpdateProject는 프로젝트를 수정합니다.
// @Summary 프로젝트 수정
// @Description 프로젝트 정보를 수정합니다
// @Tags projects
// @Accept json
// @Produce json
// @Param id path string true "프로젝트 ID"
// @Param body body map[string]interface{} true "프로젝트 수정 요청"
// @Security BearerAuth
// @Success 200 {object} models.SuccessResponse "수정된 프로젝트"
// @Failure 400 {object} models.ErrorResponse "잘못된 요청"
// @Failure 401 {object} models.ErrorResponse "인증 실패"
// @Failure 404 {object} models.ErrorResponse "프로젝트를 찾을 수 없음"
// @Failure 409 {object} models.ErrorResponse "이미 존재하는 프로젝트 이름"
// @Router /projects/{id} [put]
func (pc *ProjectController) UpdateProject(c *gin.Context) {
	// 사용자 정보 가져오기
	claims, exists := c.Get("claims")
	if !exists {
		middleware.UnauthorizedError(c, "인증 정보를 찾을 수 없습니다")
		return
	}
	userClaims := claims.(*auth.Claims)
	
	id := c.Param("id")
	if id == "" {
		middleware.ValidationError(c, "프로젝트 ID가 필요합니다", nil)
		return
	}
	
	// 프로젝트 존재 및 권한 확인
	project, err := pc.storage.Project().GetByID(c, id)
	if err != nil {
		if err == storage.ErrNotFound {
			middleware.NotFoundError(c, "프로젝트를 찾을 수 없습니다")
			return
		}
		middleware.InternalError(c, "프로젝트 조회 실패", err)
		return
	}
	
	// 워크스페이스 소유권 확인
	workspace, err := pc.storage.Workspace().GetByID(c, project.WorkspaceID)
	if err != nil {
		middleware.InternalError(c, "워크스페이스 조회 실패", err)
		return
	}
	
	if workspace.OwnerID != userClaims.UserID {
		middleware.ForbiddenError(c, "프로젝트를 수정할 권한이 없습니다")
		return
	}
	
	// 요청 데이터 파싱
	var req models.UpdateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.ValidationError(c, "요청 데이터가 올바르지 않습니다", err.Error())
		return
	}
	
	// 프로젝트 경로 유효성 검사 (변경되는 경우)
	if req.Path != nil {
		if err := utils.IsValidProjectPath(*req.Path); err != nil {
			middleware.ValidationError(c, "프로젝트 경로가 유효하지 않습니다", err.Error())
			return
		}
	}
	
	// map[string]interface{}로 변환
	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Path != nil {
		updates["path"] = *req.Path
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.GitURL != nil {
		updates["git_url"] = *req.GitURL
	}
	if req.GitBranch != nil {
		updates["git_branch"] = *req.GitBranch
	}
	if req.Language != nil {
		updates["language"] = *req.Language
	}
	
	// 프로젝트 업데이트
	if err := pc.projectService.UpdateProject(c, id, updates); err != nil {
		if err.Error() == "project name already exists in workspace" {
			middleware.ConflictError(c, "이미 존재하는 프로젝트 이름입니다")
			return
		}
		if err.Error() == "project path already in use" {
			middleware.ConflictError(c, "이미 사용 중인 프로젝트 경로입니다")
			return
		}
		middleware.InternalError(c, "프로젝트 수정 실패", err)
		return
	}
	
	// 수정된 프로젝트 조회
	updatedProject, err := pc.projectService.GetProject(c, id)
	if err != nil {
		middleware.InternalError(c, "수정된 프로젝트 조회 실패", err)
		return
	}
	
	// 성공 응답
	response := models.SuccessResponse{
		Success: true,
		Message: "프로젝트가 수정되었습니다",
		Data:    updatedProject,
	}
	
	c.JSON(http.StatusOK, response)
}

// DeleteProject는 프로젝트를 삭제합니다.
// @Summary 프로젝트 삭제
// @Description 프로젝트와 관련된 모든 데이터를 삭제합니다
// @Tags projects
// @Accept json
// @Produce json
// @Param id path string true "프로젝트 ID"
// @Security BearerAuth
// @Success 200 {object} models.SuccessResponse "삭제 성공"
// @Failure 400 {object} models.ErrorResponse "잘못된 요청"
// @Failure 401 {object} models.ErrorResponse "인증 실패"
// @Failure 404 {object} models.ErrorResponse "프로젝트를 찾을 수 없음"
// @Router /projects/{id} [delete]
func (pc *ProjectController) DeleteProject(c *gin.Context) {
	// 사용자 정보 가져오기
	claims, exists := c.Get("claims")
	if !exists {
		middleware.UnauthorizedError(c, "인증 정보를 찾을 수 없습니다")
		return
	}
	userClaims := claims.(*auth.Claims)
	
	id := c.Param("id")
	if id == "" {
		middleware.ValidationError(c, "프로젝트 ID가 필요합니다", nil)
		return
	}
	
	// 프로젝트 존재 및 권한 확인
	project, err := pc.storage.Project().GetByID(c, id)
	if err != nil {
		if err == storage.ErrNotFound {
			middleware.NotFoundError(c, "프로젝트를 찾을 수 없습니다")
			return
		}
		middleware.InternalError(c, "프로젝트 조회 실패", err)
		return
	}
	
	// 워크스페이스 소유권 확인
	workspace, err := pc.storage.Workspace().GetByID(c, project.WorkspaceID)
	if err != nil {
		middleware.InternalError(c, "워크스페이스 조회 실패", err)
		return
	}
	
	if workspace.OwnerID != userClaims.UserID {
		middleware.ForbiddenError(c, "프로젝트를 삭제할 권한이 없습니다")
		return
	}
	
	// 프로젝트 삭제
	if err := pc.projectService.DeleteProject(c, id); err != nil {
		if err == storage.ErrNotFound {
			middleware.NotFoundError(c, "프로젝트를 찾을 수 없습니다")
			return
		}
		middleware.InternalError(c, "프로젝트 삭제 실패", err)
		return
	}
	
	// 성공 응답
	response := models.SuccessResponse{
		Success: true,
		Message: "프로젝트가 삭제되었습니다",
		Data: gin.H{
			"id": id,
		},
	}
	
	c.JSON(http.StatusOK, response)
}