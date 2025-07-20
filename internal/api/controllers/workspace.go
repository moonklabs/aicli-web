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
// GET /api/v1/workspaces
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
// POST /api/v1/workspaces
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
// GET /api/v1/workspaces/:id
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
// PUT /api/v1/workspaces/:id
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
// DELETE /api/v1/workspaces/:id
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