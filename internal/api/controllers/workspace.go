package controllers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/aicli/aicli-web/internal/auth"
	"github.com/aicli/aicli-web/internal/middleware"
	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/services"
)

// WorkspaceController는 워크스페이스 관련 API를 처리합니다.
type WorkspaceController struct {
	service       services.WorkspaceService
	dockerService *services.DockerWorkspaceService // Docker 통합 서비스 추가
}

// NewWorkspaceController는 새로운 워크스페이스 컨트롤러를 생성합니다.
func NewWorkspaceController(service services.WorkspaceService, dockerService *services.DockerWorkspaceService) *WorkspaceController {
	return &WorkspaceController{
		service:       service,
		dockerService: dockerService,
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
// @Success 200 {object} models.WorkspaceListResponse "워크스페이스 목록"
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
	
	// 워크스페이스 목록 조회 (서비스 계층 사용)
	response, err := wc.service.ListWorkspaces(c, userClaims.UserID, &req)
	if err != nil {
		middleware.HandleServiceError(c, err)
		return
	}
	
	// 성공 응답에 메시지 추가
	response.Success = true
	if len(response.Data) == 0 {
		c.JSON(http.StatusOK, models.SuccessResponse{
			Success: true,
			Message: "워크스페이스가 없습니다",
			Data: gin.H{
				"workspaces": []models.Workspace{},
				"meta": response.Meta,
			},
		})
		return
	}
	
	c.JSON(http.StatusOK, models.SuccessResponse{
		Success: true,
		Message: "워크스페이스 목록을 조회했습니다",
		Data: gin.H{
			"workspaces": response.Data,
			"meta": response.Meta,
		},
	})
}

// CreateWorkspace는 새 워크스페이스를 생성합니다.
// @Summary 워크스페이스 생성
// @Description 새로운 워크스페이스를 생성합니다 (Docker 컨테이너 포함)
// @Tags workspaces
// @Accept json
// @Produce json
// @Param body body models.CreateWorkspaceRequest true "워크스페이스 생성 요청"
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
	var req models.CreateWorkspaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.ValidationError(c, "요청 데이터가 올바르지 않습니다", err.Error())
		return
	}
	
	// Docker 통합 서비스를 사용하여 워크스페이스 생성
	var workspace *models.Workspace
	var err error
	
	if wc.dockerService != nil {
		// Docker 통합 워크스페이스 생성
		workspace, err = wc.dockerService.CreateWorkspace(c, &req, userClaims.UserID)
	} else {
		// 기본 서비스 사용 (Docker 비활성화된 경우)
		workspace, err = wc.service.CreateWorkspace(c, &req, userClaims.UserID)
	}
	
	if err != nil {
		middleware.HandleServiceError(c, err)
		return
	}
	
	// 성공 응답
	message := "워크스페이스가 생성되었습니다"
	if wc.dockerService != nil {
		message = "워크스페이스가 생성되었습니다 (컨테이너 설정 중)"
	}
	
	c.JSON(http.StatusCreated, models.SuccessResponse{
		Success: true,
		Message: message,
		Data:    workspace,
	})
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
	
	// 워크스페이스 조회 (서비스 계층 사용)
	workspace, err := wc.service.GetWorkspace(c, id, userClaims.UserID)
	if err != nil {
		middleware.HandleServiceError(c, err)
		return
	}
	
	// 성공 응답
	c.JSON(http.StatusOK, models.SuccessResponse{
		Success: true,
		Message: "워크스페이스 정보를 조회했습니다",
		Data:    workspace,
	})
}

// UpdateWorkspace는 워크스페이스를 수정합니다.
// @Summary 워크스페이스 수정
// @Description 워크스페이스 정보를 수정합니다
// @Tags workspaces
// @Accept json
// @Produce json
// @Param id path string true "워크스페이스 ID"
// @Param body body models.UpdateWorkspaceRequest true "워크스페이스 수정 요청"
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
	var req models.UpdateWorkspaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.ValidationError(c, "요청 데이터가 올바르지 않습니다", err.Error())
		return
	}
	
	// 워크스페이스 업데이트 (서비스 계층 사용)
	updatedWorkspace, err := wc.service.UpdateWorkspace(c, id, &req, userClaims.UserID)
	if err != nil {
		middleware.HandleServiceError(c, err)
		return
	}
	
	// 성공 응답
	c.JSON(http.StatusOK, models.SuccessResponse{
		Success: true,
		Message: "워크스페이스가 수정되었습니다",
		Data:    updatedWorkspace,
	})
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
	
	// 워크스페이스 삭제 (서비스 계층 사용)
	err := wc.service.DeleteWorkspace(c, id, userClaims.UserID)
	if err != nil {
		middleware.HandleServiceError(c, err)
		return
	}
	
	// 성공 응답
	c.JSON(http.StatusOK, models.SuccessResponse{
		Success: true,
		Message: "워크스페이스가 삭제되었습니다",
		Data: gin.H{
			"id": id,
		},
	})
}

// GetWorkspaceStatus는 워크스페이스의 상세 상태를 조회합니다.
// @Summary 워크스페이스 상태 조회
// @Description 워크스페이스와 연관된 Docker 컨테이너 상태를 포함한 상세 정보를 조회합니다
// @Tags workspaces
// @Accept json
// @Produce json
// @Param id path string true "워크스페이스 ID"
// @Security BearerAuth
// @Success 200 {object} models.SuccessResponse "워크스페이스 상태 정보"
// @Failure 400 {object} models.ErrorResponse "잘못된 요청"
// @Failure 401 {object} models.ErrorResponse "인증 실패"
// @Failure 404 {object} models.ErrorResponse "워크스페이스를 찾을 수 없음"
// @Router /workspaces/{id}/status [get]
func (wc *WorkspaceController) GetWorkspaceStatus(c *gin.Context) {
	claims := c.MustGet("claims").(*auth.Claims)
	workspaceID := c.Param("id")
	
	if workspaceID == "" {
		middleware.ValidationError(c, "워크스페이스 ID가 필요합니다", nil)
		return
	}
	
	// 기본 워크스페이스 정보 조회
	workspace, err := wc.service.GetWorkspace(c, workspaceID, claims.UserID)
	if err != nil {
		middleware.HandleServiceError(c, err)
		return
	}
	
	// Docker 컨테이너 상태 조회 (선택적)
	var containerStatus *services.WorkspaceStatus
	if wc.dockerService != nil {
		status, err := wc.dockerService.GetWorkspaceStatus(c, workspaceID)
		if err != nil {
			// Docker 상태 조회 실패는 경고만 출력
			containerStatus = &services.WorkspaceStatus{
				ContainerState: "unknown",
				LastError:      err.Error(),
			}
		} else {
			containerStatus = status
		}
	}
	
	response := gin.H{
		"workspace":        workspace,
		"container_status": containerStatus,
		"last_updated":     time.Now(),
	}
	
	c.JSON(http.StatusOK, models.SuccessResponse{
		Success: true,
		Message: "워크스페이스 상태를 조회했습니다",
		Data:    response,
	})
}

// BatchWorkspaceOperation은 대량 워크스페이스 작업을 처리합니다.
// @Summary 배치 워크스페이스 작업
// @Description 여러 워크스페이스에 대해 일괄 작업을 수행합니다
// @Tags workspaces
// @Accept json
// @Produce json
// @Param body body services.BatchOperationRequest true "배치 작업 요청"
// @Security BearerAuth
// @Success 202 {object} models.SuccessResponse "배치 작업 시작됨"
// @Failure 400 {object} models.ErrorResponse "잘못된 요청"
// @Failure 401 {object} models.ErrorResponse "인증 실패"
// @Router /workspaces/batch [post]
func (wc *WorkspaceController) BatchWorkspaceOperation(c *gin.Context) {
	claims := c.MustGet("claims").(*auth.Claims)
	
	var req services.BatchOperationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.ValidationError(c, "요청 데이터가 올바르지 않습니다", err.Error())
		return
	}
	
	// Docker 서비스가 없으면 에러 반환
	if wc.dockerService == nil {
		middleware.ValidationError(c, "배치 작업은 Docker 서비스가 활성화된 경우에만 사용할 수 있습니다", nil)
		return
	}
	
	// 비동기 배치 작업 시작
	batchID, err := wc.dockerService.StartBatchOperation(c, &req, claims.UserID)
	if err != nil {
		middleware.HandleServiceError(c, err)
		return
	}
	
	c.JSON(http.StatusAccepted, models.SuccessResponse{
		Success: true,
		Message: "배치 작업이 시작되었습니다",
		Data: gin.H{
			"batch_id":        batchID,
			"operation_type":  req.Operation,
			"workspace_count": len(req.WorkspaceIDs),
			"status_url":      "/api/workspaces/batch/" + batchID + "/status",
		},
	})
}

// GetBatchOperationStatus는 배치 작업 상태를 조회합니다.
// @Summary 배치 작업 상태 조회
// @Description 배치 작업의 진행 상황과 결과를 조회합니다
// @Tags workspaces
// @Accept json
// @Produce json
// @Param batch_id path string true "배치 작업 ID"
// @Security BearerAuth
// @Success 200 {object} models.SuccessResponse "배치 작업 상태"
// @Failure 400 {object} models.ErrorResponse "잘못된 요청"
// @Failure 401 {object} models.ErrorResponse "인증 실패"
// @Failure 404 {object} models.ErrorResponse "배치 작업을 찾을 수 없음"
// @Router /workspaces/batch/{batch_id}/status [get]
func (wc *WorkspaceController) GetBatchOperationStatus(c *gin.Context) {
	batchID := c.Param("batch_id")
	
	if batchID == "" {
		middleware.ValidationError(c, "배치 작업 ID가 필요합니다", nil)
		return
	}
	
	if wc.dockerService == nil {
		middleware.ValidationError(c, "배치 작업은 Docker 서비스가 활성화된 경우에만 사용할 수 있습니다", nil)
		return
	}
	
	status, err := wc.dockerService.GetBatchOperationStatus(c, batchID)
	if err != nil {
		middleware.HandleServiceError(c, err)
		return
	}
	
	c.JSON(http.StatusOK, models.SuccessResponse{
		Success: true,
		Message: "배치 작업 상태를 조회했습니다",
		Data:    status,
	})
}

// CancelBatchOperation은 배치 작업을 취소합니다.
// @Summary 배치 작업 취소
// @Description 진행 중인 배치 작업을 취소합니다
// @Tags workspaces
// @Accept json
// @Produce json
// @Param batch_id path string true "배치 작업 ID"
// @Security BearerAuth
// @Success 200 {object} models.SuccessResponse "배치 작업 취소됨"
// @Failure 400 {object} models.ErrorResponse "잘못된 요청"
// @Failure 401 {object} models.ErrorResponse "인증 실패"
// @Failure 404 {object} models.ErrorResponse "배치 작업을 찾을 수 없음"
// @Router /workspaces/batch/{batch_id}/cancel [post]
func (wc *WorkspaceController) CancelBatchOperation(c *gin.Context) {
	batchID := c.Param("batch_id")
	
	if batchID == "" {
		middleware.ValidationError(c, "배치 작업 ID가 필요합니다", nil)
		return
	}
	
	if wc.dockerService == nil {
		middleware.ValidationError(c, "배치 작업은 Docker 서비스가 활성화된 경우에만 사용할 수 있습니다", nil)
		return
	}
	
	err := wc.dockerService.CancelBatchOperation(c, batchID)
	if err != nil {
		middleware.HandleServiceError(c, err)
		return
	}
	
	c.JSON(http.StatusOK, models.SuccessResponse{
		Success: true,
		Message: "배치 작업이 취소되었습니다",
		Data: gin.H{
			"batch_id": batchID,
			"status":   "cancelled",
		},
	})
}