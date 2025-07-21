package controllers

import (
	"net/http"
	"strconv"

	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/services"
	"github.com/gin-gonic/gin"
)

// TaskController 태스크 컨트롤러
type TaskController struct {
	taskService *services.TaskService
}

// NewTaskController 새 태스크 컨트롤러 생성
func NewTaskController(taskService *services.TaskService) *TaskController {
	return &TaskController{
		taskService: taskService,
	}
}

// Create 태스크 생성
// @Summary 새 태스크 생성
// @Description 세션에서 새 태스크를 생성합니다
// @Tags tasks
// @Accept json
// @Produce json
// @Param sessionId path string true "세션 ID"
// @Param task body models.TaskCreateRequest true "태스크 생성 요청"
// @Success 201 {object} models.TaskResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /sessions/{sessionId}/tasks [post]
// @Security BearerAuth
func (tc *TaskController) Create(c *gin.Context) {
	sessionID := c.Param("sessionId")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Message: "세션 ID가 필요합니다",
		})
		return
	}

	var req models.TaskCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Message: "잘못된 요청 형식입니다",
			Details: err.Error(),
		})
		return
	}

	// 세션 ID 설정
	req.SessionID = sessionID

	task, err := tc.taskService.Create(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Message: "태스크 생성에 실패했습니다",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, task.ToResponse())
}

// List 태스크 목록 조회
// @Summary 태스크 목록 조회
// @Description 태스크 목록을 조회합니다 (필터링 및 페이징 지원)
// @Tags tasks
// @Accept json
// @Produce json
// @Param session_id query string false "세션 ID"
// @Param status query string false "태스크 상태" Enums(pending, running, completed, failed, cancelled)
// @Param active query boolean false "활성 태스크만 조회"
// @Param page query int false "페이지 번호" default(1)
// @Param limit query int false "페이지 크기" default(10)
// @Success 200 {object} models.PagingResponse[models.TaskResponse]
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /tasks [get]
// @Security BearerAuth
func (tc *TaskController) List(c *gin.Context) {
	// 필터 파라미터 파싱
	var filter models.TaskFilter
	if sessionID := c.Query("session_id"); sessionID != "" {
		filter.SessionID = &sessionID
	}
	if status := c.Query("status"); status != "" {
		taskStatus := models.TaskStatus(status)
		filter.Status = &taskStatus
	}
	if activeStr := c.Query("active"); activeStr != "" {
		if active, err := strconv.ParseBool(activeStr); err == nil {
			filter.Active = &active
		}
	}

	// 페이징 파라미터 파싱
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	paging := &models.PagingRequest{
		Page:  page,
		Limit: limit,
	}

	result, err := tc.taskService.List(c.Request.Context(), &filter, paging)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Message: "태스크 목록 조회에 실패했습니다",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetByID 태스크 상세 조회
// @Summary 태스크 상세 조회
// @Description ID로 태스크 상세 정보를 조회합니다
// @Tags tasks
// @Accept json
// @Produce json
// @Param id path string true "태스크 ID"
// @Success 200 {object} models.TaskResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /tasks/{id} [get]
// @Security BearerAuth
func (tc *TaskController) GetByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Message: "태스크 ID가 필요합니다",
		})
		return
	}

	task, err := tc.taskService.GetByID(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "태스크 조회 실패: resource not found" {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Message: "태스크를 찾을 수 없습니다",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Message: "태스크 조회에 실패했습니다",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, task.ToResponse())
}

// Cancel 태스크 취소
// @Summary 태스크 취소
// @Description 실행 중인 태스크를 취소합니다
// @Tags tasks
// @Accept json
// @Produce json
// @Param id path string true "태스크 ID"
// @Success 204 "No Content"
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 409 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /tasks/{id} [delete]
// @Security BearerAuth
func (tc *TaskController) Cancel(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Message: "태스크 ID가 필요합니다",
		})
		return
	}

	err := tc.taskService.Cancel(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "태스크를 찾을 수 없습니다: "+id {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Message: "태스크를 찾을 수 없습니다",
			})
			return
		}
		if contains(err.Error(), "태스크를 취소할 수 없습니다") {
			c.JSON(http.StatusConflict, models.ErrorResponse{
				Message: "태스크를 취소할 수 없습니다",
				Details: err.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Message: "태스크 취소에 실패했습니다",
			Details: err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetActiveTasks 활성 태스크 목록 조회
// @Summary 활성 태스크 목록 조회
// @Description 현재 실행 중인 활성 태스크 목록을 조회합니다
// @Tags tasks
// @Accept json
// @Produce json
// @Success 200 {array} models.TaskResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /tasks/active [get]
// @Security BearerAuth
func (tc *TaskController) GetActiveTasks(c *gin.Context) {
	tasks, err := tc.taskService.GetActiveTasks(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Message: "활성 태스크 조회에 실패했습니다",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, tasks)
}

// GetStats 태스크 통계 조회
// @Summary 태스크 통계 조회
// @Description 태스크 큐 및 실행 통계를 조회합니다
// @Tags tasks
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} models.ErrorResponse
// @Router /tasks/stats [get]
// @Security BearerAuth
func (tc *TaskController) GetStats(c *gin.Context) {
	stats, err := tc.taskService.GetStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Message: "태스크 통계 조회에 실패했습니다",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"stats": stats,
	})
}

// contains 문자열 포함 여부 확인 (헬퍼 함수)
func contains(str, substr string) bool {
	return len(str) >= len(substr) && (str == substr || (len(str) > len(substr) && 
		(str[:len(substr)] == substr || str[len(str)-len(substr):] == substr ||
		 findIndex(str, substr) >= 0)))
}

// findIndex 문자열 인덱스 찾기 (헬퍼 함수)
func findIndex(str, substr string) int {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}