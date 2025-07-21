package controllers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/drumcap/aicli-web/internal/middleware"
)

// WorkspaceController는 워크스페이스 관련 API를 처리합니다.
type WorkspaceController struct {
	// TODO: 실제 서비스 계층 의존성 추가
}

// NewWorkspaceController는 새로운 워크스페이스 컨트롤러를 생성합니다.
func NewWorkspaceController() *WorkspaceController {
	return &WorkspaceController{}
}

// Workspace는 워크스페이스 모델입니다.
type Workspace struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Path        string `json:"path"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
	ActiveTasks int    `json:"active_tasks"`
}

// CreateWorkspaceRequest는 워크스페이스 생성 요청입니다.
type CreateWorkspaceRequest struct {
	Name      string `json:"name" binding:"required"`
	Path      string `json:"path" binding:"required"`
	ClaudeKey string `json:"claude_key,omitempty"`
}

// UpdateWorkspaceRequest는 워크스페이스 수정 요청입니다.
type UpdateWorkspaceRequest struct {
	Name      string `json:"name,omitempty"`
	Path      string `json:"path,omitempty"`
	ClaudeKey string `json:"claude_key,omitempty"`
}

// ListWorkspaces는 워크스페이스 목록을 조회합니다.
// @Summary 워크스페이스 목록 조회
// @Description 사용자의 모든 워크스페이스 목록을 페이지네이션과 함께 조회합니다
// @Tags workspaces
// @Accept json
// @Produce json
// @Param page query int false "페이지 번호" default(1)
// @Param limit query int false "페이지당 항목 수" default(10)
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "워크스페이스 목록"
// @Failure 401 {object} map[string]interface{} "인증 실패"
// @Router /workspaces [get]
func (wc *WorkspaceController) ListWorkspaces(c *gin.Context) {
	// 쿼리 파라미터 처리
	page := c.DefaultQuery("page", "1")
	limit := c.DefaultQuery("limit", "10")
	status := c.Query("status")
	
	// TODO: 실제 데이터베이스에서 조회
	// 현재는 스텁 데이터 반환
	workspaces := []Workspace{
		{
			ID:          1,
			Name:        "example-project",
			Path:        "/projects/example",
			Status:      "active",
			CreatedAt:   "2025-01-20T10:00:00Z",
			UpdatedAt:   "2025-01-20T10:00:00Z",
			ActiveTasks: 2,
		},
		{
			ID:          2,
			Name:        "test-project",
			Path:        "/projects/test",
			Status:      "inactive",
			CreatedAt:   "2025-01-20T11:00:00Z",
			UpdatedAt:   "2025-01-20T11:00:00Z",
			ActiveTasks: 0,
		},
	}
	
	// 상태 필터링 (스텁)
	if status != "" {
		filtered := make([]Workspace, 0)
		for _, ws := range workspaces {
			if ws.Status == status {
				filtered = append(filtered, ws)
			}
		}
		workspaces = filtered
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    workspaces,
		"meta": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       len(workspaces),
			"total_pages": 1,
		},
	})
}

// CreateWorkspace는 새 워크스페이스를 생성합니다.
// @Summary 워크스페이스 생성
// @Description 새로운 워크스페이스를 생성합니다
// @Tags workspaces
// @Accept json
// @Produce json
// @Param body body CreateWorkspaceRequest true "워크스페이스 생성 요청"
// @Security BearerAuth
// @Success 201 {object} map[string]interface{} "생성된 워크스페이스"
// @Failure 400 {object} map[string]interface{} "잘못된 요청"
// @Failure 401 {object} map[string]interface{} "인증 실패"
// @Router /workspaces [post]
func (wc *WorkspaceController) CreateWorkspace(c *gin.Context) {
	var req CreateWorkspaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.ValidationError(c, "요청 데이터가 올바르지 않습니다", err.Error())
		return
	}
	
	// TODO: 실제 워크스페이스 생성 로직
	// 1. 경로 유효성 검사
	// 2. 이름 중복 검사
	// 3. Docker 컨테이너 준비
	// 4. 데이터베이스 저장
	
	workspace := Workspace{
		ID:          3,
		Name:        req.Name,
		Path:        req.Path,
		Status:      "creating",
		CreatedAt:   "2025-01-20T12:00:00Z",
		UpdatedAt:   "2025-01-20T12:00:00Z",
		ActiveTasks: 0,
	}
	
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    workspace,
		"message": "워크스페이스가 생성되었습니다",
	})
}

// GetWorkspace는 특정 워크스페이스를 조회합니다.
// @Summary 워크스페이스 상세 조회
// @Description ID로 특정 워크스페이스의 상세 정보를 조회합니다
// @Tags workspaces
// @Accept json
// @Produce json
// @Param id path int true "워크스페이스 ID"
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "워크스페이스 정보"
// @Failure 400 {object} map[string]interface{} "잘못된 요청"
// @Failure 401 {object} map[string]interface{} "인증 실패"
// @Failure 404 {object} map[string]interface{} "워크스페이스를 찾을 수 없음"
// @Router /workspaces/{id} [get]
func (wc *WorkspaceController) GetWorkspace(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		middleware.ValidationError(c, "워크스페이스 ID는 숫자여야 합니다", nil)
		return
	}
	
	// TODO: 실제 데이터베이스에서 조회
	if id == 1 {
		workspace := Workspace{
			ID:          1,
			Name:        "example-project",
			Path:        "/projects/example",
			Status:      "active",
			CreatedAt:   "2025-01-20T10:00:00Z",
			UpdatedAt:   "2025-01-20T10:00:00Z",
			ActiveTasks: 2,
		}
		
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    workspace,
		})
	} else {
		middleware.NotFoundError(c, "워크스페이스를 찾을 수 없습니다")
	}
}

// UpdateWorkspace는 워크스페이스를 수정합니다.
// @Summary 워크스페이스 수정
// @Description 워크스페이스 정보를 수정합니다
// @Tags workspaces
// @Accept json
// @Produce json
// @Param id path int true "워크스페이스 ID"
// @Param body body UpdateWorkspaceRequest true "워크스페이스 수정 요청"
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "수정된 워크스페이스"
// @Failure 400 {object} map[string]interface{} "잘못된 요청"
// @Failure 401 {object} map[string]interface{} "인증 실패"
// @Failure 404 {object} map[string]interface{} "워크스페이스를 찾을 수 없음"
// @Router /workspaces/{id} [put]
func (wc *WorkspaceController) UpdateWorkspace(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		middleware.ValidationError(c, "워크스페이스 ID는 숫자여야 합니다", nil)
		return
	}
	
	var req UpdateWorkspaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.ValidationError(c, "요청 데이터가 올바르지 않습니다", err.Error())
		return
	}
	
	// TODO: 실제 워크스페이스 수정 로직
	// 1. 워크스페이스 존재 확인
	// 2. 수정 권한 확인
	// 3. 변경사항 적용
	// 4. 데이터베이스 업데이트
	
	workspace := Workspace{
		ID:          id,
		Name:        req.Name,
		Path:        req.Path,
		Status:      "active",
		CreatedAt:   "2025-01-20T10:00:00Z",
		UpdatedAt:   "2025-01-20T12:30:00Z",
		ActiveTasks: 2,
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    workspace,
		"message": "워크스페이스가 수정되었습니다",
	})
}

// DeleteWorkspace는 워크스페이스를 삭제합니다.
// @Summary 워크스페이스 삭제
// @Description 워크스페이스와 관련된 모든 데이터를 삭제합니다
// @Tags workspaces
// @Accept json
// @Produce json
// @Param id path int true "워크스페이스 ID"
// @Security BearerAuth
// @Success 204 "삭제 성공"
// @Failure 400 {object} map[string]interface{} "잘못된 요청"
// @Failure 401 {object} map[string]interface{} "인증 실패"
// @Failure 404 {object} map[string]interface{} "워크스페이스를 찾을 수 없음"
// @Router /workspaces/{id} [delete]
func (wc *WorkspaceController) DeleteWorkspace(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		middleware.ValidationError(c, "워크스페이스 ID는 숫자여야 합니다", nil)
		return
	}
	
	// TODO: 실제 워크스페이스 삭제 로직
	// 1. 워크스페이스 존재 확인
	// 2. 실행 중인 태스크 확인
	// 3. 태스크 정리 또는 강제 종료
	// 4. Docker 컨테이너 제거
	// 5. 데이터베이스에서 삭제
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "워크스페이스가 삭제되었습니다",
		"data": gin.H{
			"id": id,
		},
	})
}