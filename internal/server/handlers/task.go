package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// TaskResponse는 태스크 정보 응답 구조체입니다.
type TaskResponse struct {
	ID          string `json:"id"`
	WorkspaceID int    `json:"workspace_id"`
	Command     string `json:"command"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
	StartedAt   string `json:"started_at,omitempty"`
	CompletedAt string `json:"completed_at,omitempty"`
	Output      string `json:"output,omitempty"`
}

// CreateTaskRequest는 태스크 생성 요청 구조체입니다.
type CreateTaskRequest struct {
	WorkspaceID int    `json:"workspace_id" binding:"required"`
	Command     string `json:"command" binding:"required"`
	Interactive bool   `json:"interactive"`
	Detach      bool   `json:"detach"`
}

// ListTasks는 모든 태스크를 조회합니다.
func ListTasks(c *gin.Context) {
	// 쿼리 파라미터 처리
	workspaceID := c.Query("workspace_id")
	status := c.Query("status")

	// TODO: 실제 데이터베이스에서 태스크 목록 조회
	tasks := []TaskResponse{
		{
			ID:          "task-12345",
			WorkspaceID: 1,
			Command:     "aicli workspace list",
			Status:      "completed",
			CreatedAt:   "2025-01-20T10:00:00Z",
			UpdatedAt:   "2025-01-20T10:05:00Z",
			StartedAt:   "2025-01-20T10:00:01Z",
			CompletedAt: "2025-01-20T10:05:00Z",
		},
	}

	// 필터 적용 (임시)
	if workspaceID != "" {
		// workspace_id로 필터링
	}
	if status != "" {
		// status로 필터링
	}

	c.JSON(http.StatusOK, gin.H{
		"tasks": tasks,
		"total": len(tasks),
		"filters": gin.H{
			"workspace_id": workspaceID,
			"status":       status,
		},
	})
}

// CreateTask는 새 태스크를 생성합니다.
func CreateTask(c *gin.Context) {
	var req CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"message": err.Error(),
		})
		return
	}

	// TODO: 워크스페이스 존재 확인
	// TODO: 실제 Claude CLI 태스크 생성 로직

	task := TaskResponse{
		ID:          "task-67890",
		WorkspaceID: req.WorkspaceID,
		Command:     req.Command,
		Status:      "running",
		CreatedAt:   "2025-01-20T10:00:00Z",
		UpdatedAt:   "2025-01-20T10:00:00Z",
		StartedAt:   "2025-01-20T10:00:01Z",
	}

	if req.Detach {
		// 백그라운드 실행
		c.JSON(http.StatusAccepted, task)
	} else {
		// 동기 실행 (완료까지 대기)
		task.Status = "completed"
		task.CompletedAt = "2025-01-20T10:05:00Z"
		task.Output = "Task completed successfully"
		c.JSON(http.StatusOK, task)
	}
}

// GetTask는 특정 태스크 정보를 조회합니다.
func GetTask(c *gin.Context) {
	taskID := c.Param("id")

	// TODO: 실제 데이터베이스에서 태스크 조회
	if taskID == "task-12345" {
		task := TaskResponse{
			ID:          "task-12345",
			WorkspaceID: 1,
			Command:     "aicli workspace list",
			Status:      "completed",
			CreatedAt:   "2025-01-20T10:00:00Z",
			UpdatedAt:   "2025-01-20T10:05:00Z",
			StartedAt:   "2025-01-20T10:00:01Z",
			CompletedAt: "2025-01-20T10:05:00Z",
			Output:      "Available workspaces:\n- example-project",
		}
		c.JSON(http.StatusOK, task)
	} else {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Task not found",
			"message": "해당 태스크를 찾을 수 없습니다",
		})
	}
}

// CancelTask는 실행 중인 태스크를 취소합니다.
func CancelTask(c *gin.Context) {
	taskID := c.Param("id")

	// TODO: 실제 태스크 취소 로직
	// Docker 컨테이너 종료 등

	c.JSON(http.StatusOK, gin.H{
		"message": "태스크가 취소되었습니다",
		"task_id": taskID,
		"status":  "cancelled",
	})
}