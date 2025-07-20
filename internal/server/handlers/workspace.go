package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// WorkspaceResponse는 워크스페이스 정보 응답 구조체입니다.
type WorkspaceResponse struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Path        string `json:"path"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
	ActiveTasks int    `json:"active_tasks"`
}

// CreateWorkspaceRequest는 워크스페이스 생성 요청 구조체입니다.
type CreateWorkspaceRequest struct {
	Name      string `json:"name" binding:"required"`
	Path      string `json:"path" binding:"required"`
	ClaudeKey string `json:"claude_key,omitempty"`
}

// ListWorkspaces는 모든 워크스페이스를 조회합니다.
func ListWorkspaces(c *gin.Context) {
	// TODO: 실제 데이터베이스에서 워크스페이스 목록 조회
	workspaces := []WorkspaceResponse{
		{
			ID:          1,
			Name:        "example-project",
			Path:        "/projects/example",
			Status:      "active",
			CreatedAt:   "2025-01-20T10:00:00Z",
			UpdatedAt:   "2025-01-20T10:00:00Z",
			ActiveTasks: 0,
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"workspaces": workspaces,
		"total":      len(workspaces),
	})
}

// CreateWorkspace는 새 워크스페이스를 생성합니다.
func CreateWorkspace(c *gin.Context) {
	var req CreateWorkspaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"message": err.Error(),
		})
		return
	}

	// TODO: 실제 워크스페이스 생성 로직
	workspace := WorkspaceResponse{
		ID:          2,
		Name:        req.Name,
		Path:        req.Path,
		Status:      "creating",
		CreatedAt:   "2025-01-20T10:00:00Z",
		UpdatedAt:   "2025-01-20T10:00:00Z",
		ActiveTasks: 0,
	}

	c.JSON(http.StatusCreated, workspace)
}

// GetWorkspace는 특정 워크스페이스 정보를 조회합니다.
func GetWorkspace(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid workspace ID",
			"message": "워크스페이스 ID는 숫자여야 합니다",
		})
		return
	}

	// TODO: 실제 데이터베이스에서 워크스페이스 조회
	if id == 1 {
		workspace := WorkspaceResponse{
			ID:          1,
			Name:        "example-project",
			Path:        "/projects/example",
			Status:      "active",
			CreatedAt:   "2025-01-20T10:00:00Z",
			UpdatedAt:   "2025-01-20T10:00:00Z",
			ActiveTasks: 0,
		}
		c.JSON(http.StatusOK, workspace)
	} else {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Workspace not found",
			"message": "해당 워크스페이스를 찾을 수 없습니다",
		})
	}
}

// DeleteWorkspace는 워크스페이스를 삭제합니다.
func DeleteWorkspace(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid workspace ID",
			"message": "워크스페이스 ID는 숫자여야 합니다",
		})
		return
	}

	// TODO: 실제 워크스페이스 삭제 로직
	// 실행 중인 태스크가 있는지 확인하고 삭제

	c.JSON(http.StatusOK, gin.H{
		"message": "워크스페이스가 삭제되었습니다",
		"id":      id,
	})
}