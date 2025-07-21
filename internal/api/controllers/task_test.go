package controllers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/services"
	"github.com/aicli/aicli-web/internal/storage/memory"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTaskControllerTest() (*gin.Engine, *services.TaskService, *models.Session) {
	gin.SetMode(gin.TestMode)
	
	storage := memory.New()
	projectService := services.NewProjectService(storage)
	sessionService := services.NewSessionService(storage, projectService, nil)
	taskService := services.NewTaskService(storage, sessionService, nil)
	taskController := NewTaskController(taskService)
	
	// 태스크 서비스 시작
	_ = taskService.Start(nil)
	
	// 테스트용 워크스페이스와 프로젝트, 세션 생성
	workspace := &models.Workspace{
		BaseModel: models.BaseModel{ID: "ws-123"},
		Name:      "Test Workspace",
		OwnerID:   "user-123",
		Settings:  map[string]interface{}{},
	}
	_ = storage.Workspace().Create(nil, workspace)
	
	project := &models.Project{
		BaseModel:   models.BaseModel{ID: "proj-123"},
		WorkspaceID: workspace.ID,
		Name:        "Test Project",
		Path:        "/tmp/test",
		Status:      models.ProjectActive,
		Settings:    map[string]interface{}{},
	}
	_ = storage.Project().Create(nil, project)
	
	session, _ := sessionService.Create(nil, &models.SessionCreateRequest{
		ProjectID: project.ID,
	})
	_ = sessionService.UpdateStatus(nil, session.ID, models.SessionActive)
	
	router := gin.New()
	
	// 라우트 설정
	router.POST("/sessions/:sessionId/tasks", taskController.Create)
	router.GET("/tasks", taskController.List)
	router.GET("/tasks/active", taskController.GetActiveTasks)
	router.GET("/tasks/stats", taskController.GetStats)
	router.GET("/tasks/:id", taskController.GetByID)
	router.DELETE("/tasks/:id", taskController.Cancel)
	
	return router, taskService, session
}

func TestTaskController_Create(t *testing.T) {
	router, taskService, session := setupTaskControllerTest()
	defer taskService.Stop()
	
	tests := []struct {
		name       string
		sessionID  string
		body       interface{}
		wantStatus int
	}{
		{
			name:      "Valid request",
			sessionID: session.ID,
			body: models.TaskCreateRequest{
				Command: "echo hello",
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "Missing session ID",
			sessionID:  "",
			body:       models.TaskCreateRequest{Command: "echo hello"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "Invalid session ID",
			sessionID:  "invalid-session",
			body:       models.TaskCreateRequest{Command: "echo hello"},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:      "Invalid body",
			sessionID: session.ID,
			body:      "invalid",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:      "Empty command",
			sessionID: session.ID,
			body: models.TaskCreateRequest{
				Command: "",
			},
			wantStatus: http.StatusBadRequest,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest("POST", fmt.Sprintf("/sessions/%s/tasks", tt.sessionID), bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			assert.Equal(t, tt.wantStatus, w.Code)
			
			if tt.wantStatus == http.StatusCreated {
				var resp models.TaskResponse
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, session.ID, resp.SessionID)
				assert.Equal(t, models.TaskPending, resp.Status)
			}
		})
	}
}

func TestTaskController_List(t *testing.T) {
	router, taskService, session := setupTaskControllerTest()
	defer taskService.Stop()
	
	// 테스트 태스크 생성
	tasks := make([]*models.Task, 5)
	for i := 0; i < 5; i++ {
		task, err := taskService.Create(nil, &models.TaskCreateRequest{
			SessionID: session.ID,
			Command:   fmt.Sprintf("echo test%d", i),
		})
		require.NoError(t, err)
		tasks[i] = task
	}
	
	tests := []struct {
		name        string
		query       string
		wantStatus  int
		wantCount   int
	}{
		{
			name:       "All tasks",
			query:      "",
			wantStatus: http.StatusOK,
			wantCount:  5,
		},
		{
			name:       "Filter by session",
			query:      fmt.Sprintf("?session_id=%s", session.ID),
			wantStatus: http.StatusOK,
			wantCount:  5,
		},
		{
			name:       "Filter by status",
			query:      "?status=pending",
			wantStatus: http.StatusOK,
			wantCount:  5,
		},
		{
			name:       "Filter active tasks",
			query:      "?active=true",
			wantStatus: http.StatusOK,
			wantCount:  5,
		},
		{
			name:       "Pagination",
			query:      "?page=1&limit=2",
			wantStatus: http.StatusOK,
			wantCount:  2,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/tasks"+tt.query, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			assert.Equal(t, tt.wantStatus, w.Code)
			
			if tt.wantStatus == http.StatusOK {
				var resp models.PagingResponse[*models.TaskResponse]
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Len(t, resp.Items, tt.wantCount)
			}
		})
	}
}

func TestTaskController_GetByID(t *testing.T) {
	router, taskService, session := setupTaskControllerTest()
	defer taskService.Stop()
	
	// 테스트 태스크 생성
	task, err := taskService.Create(nil, &models.TaskCreateRequest{
		SessionID: session.ID,
		Command:   "echo test",
	})
	require.NoError(t, err)
	
	tests := []struct {
		name       string
		taskID     string
		wantStatus int
	}{
		{
			name:       "Valid task ID",
			taskID:     task.ID,
			wantStatus: http.StatusOK,
		},
		{
			name:       "Invalid task ID",
			taskID:     "invalid-task",
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "Empty task ID",
			taskID:     "",
			wantStatus: http.StatusBadRequest,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/tasks/%s", tt.taskID), nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			assert.Equal(t, tt.wantStatus, w.Code)
			
			if tt.wantStatus == http.StatusOK {
				var resp models.TaskResponse
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, tt.taskID, resp.ID)
			}
		})
	}
}

func TestTaskController_Cancel(t *testing.T) {
	router, taskService, session := setupTaskControllerTest()
	defer taskService.Stop()
	
	// 테스트 태스크 생성
	task, err := taskService.Create(nil, &models.TaskCreateRequest{
		SessionID: session.ID,
		Command:   "sleep 10",
	})
	require.NoError(t, err)
	
	tests := []struct {
		name       string
		taskID     string
		wantStatus int
	}{
		{
			name:       "Valid task cancellation",
			taskID:     task.ID,
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "Invalid task ID",
			taskID:     "invalid-task",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "Empty task ID",
			taskID:     "",
			wantStatus: http.StatusBadRequest,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("DELETE", fmt.Sprintf("/tasks/%s", tt.taskID), nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

func TestTaskController_GetActiveTasks(t *testing.T) {
	router, taskService, session := setupTaskControllerTest()
	defer taskService.Stop()
	
	// 테스트 태스크 생성
	activeTasks := make([]*models.Task, 3)
	for i := 0; i < 3; i++ {
		task, err := taskService.Create(nil, &models.TaskCreateRequest{
			SessionID: session.ID,
			Command:   fmt.Sprintf("echo test%d", i),
		})
		require.NoError(t, err)
		activeTasks[i] = task
	}
	
	// 활성 태스크 조회
	req := httptest.NewRequest("GET", "/tasks/active", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var resp []*models.TaskResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	
	// 모든 응답이 활성 태스크인지 확인
	for _, task := range resp {
		assert.True(t, task.Status == models.TaskPending || task.Status == models.TaskRunning)
	}
}

func TestTaskController_GetStats(t *testing.T) {
	router, taskService, session := setupTaskControllerTest()
	defer taskService.Stop()
	
	// 테스트 태스크 생성
	for i := 0; i < 3; i++ {
		_, err := taskService.Create(nil, &models.TaskCreateRequest{
			SessionID: session.ID,
			Command:   fmt.Sprintf("echo test%d", i),
		})
		require.NoError(t, err)
	}
	
	// 통계 조회
	req := httptest.NewRequest("GET", "/tasks/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	
	assert.Contains(t, resp, "stats")
	stats := resp["stats"].(map[string]interface{})
	assert.Contains(t, stats, "total_tasks")
	assert.Contains(t, stats, "max_workers")
	assert.Contains(t, stats, "running")
}