package services

import (
	"context"
	"testing"
	"time"

	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/storage/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTaskTest() (*TaskService, *SessionService, *models.Session) {
	storage := memory.New()
	projectService := NewProjectService(storage)
	sessionService := NewSessionService(storage, projectService, nil)
	
	// 태스크 서비스 설정
	config := &TaskServiceConfig{
		MaxWorkers:      2,
		MaxQueueSize:    10,
		TaskTimeout:     30 * time.Second,
		CleanupInterval: 1 * time.Minute,
		CleanupMaxAge:   5 * time.Minute,
	}
	taskService := NewTaskService(storage, sessionService, config)
	
	// 테스트용 워크스페이스와 프로젝트, 세션 생성
	workspace := &models.Workspace{
		BaseModel: models.BaseModel{ID: "ws-test"},
		Name:      "Test Workspace",
		OwnerID:   "user-test",
		Settings:  map[string]interface{}{},
	}
	_ = storage.Workspace().Create(context.Background(), workspace)
	
	project := &models.Project{
		BaseModel:   models.BaseModel{ID: "proj-test"},
		WorkspaceID: workspace.ID,
		Name:        "Test Project",
		Path:        "/tmp/test",
		Status:      models.ProjectActive,
		Settings:    map[string]interface{}{},
	}
	_ = storage.Project().Create(context.Background(), project)
	
	session, _ := sessionService.Create(context.Background(), &models.SessionCreateRequest{
		ProjectID: project.ID,
	})
	_ = sessionService.UpdateStatus(context.Background(), session.ID, models.SessionActive)
	
	// 태스크 서비스 시작
	_ = taskService.Start(context.Background())
	
	return taskService, sessionService, session
}

func TestTaskService_Create(t *testing.T) {
	taskService, _, session := setupTaskTest()
	defer taskService.Stop()
	
	tests := []struct {
		name        string
		request     *models.TaskCreateRequest
		wantError   bool
		errorString string
	}{
		{
			name: "Valid task creation",
			request: &models.TaskCreateRequest{
				SessionID: session.ID,
				Command:   "echo hello",
			},
			wantError: false,
		},
		{
			name: "Invalid session ID",
			request: &models.TaskCreateRequest{
				SessionID: "invalid-session",
				Command:   "echo hello",
			},
			wantError:   true,
			errorString: "세션을 찾을 수 없습니다",
		},
		{
			name: "Empty command",
			request: &models.TaskCreateRequest{
				SessionID: session.ID,
				Command:   "",
			},
			wantError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task, err := taskService.Create(context.Background(), tt.request)
			
			if tt.wantError {
				assert.Error(t, err)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
				assert.Nil(t, task)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, task)
				assert.Equal(t, tt.request.SessionID, task.SessionID)
				assert.Equal(t, tt.request.Command, task.Command)
				assert.Equal(t, models.TaskPending, task.Status)
			}
		})
	}
}

func TestTaskService_GetByID(t *testing.T) {
	taskService, _, session := setupTaskTest()
	defer taskService.Stop()
	
	// 테스트 태스크 생성
	task, err := taskService.Create(context.Background(), &models.TaskCreateRequest{
		SessionID: session.ID,
		Command:   "echo test",
	})
	require.NoError(t, err)
	
	tests := []struct {
		name      string
		taskID    string
		wantError bool
	}{
		{
			name:      "Valid task ID",
			taskID:    task.ID,
			wantError: false,
		},
		{
			name:      "Invalid task ID",
			taskID:    "invalid-task",
			wantError: true,
		},
		{
			name:      "Empty task ID",
			taskID:    "",
			wantError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := taskService.GetByID(context.Background(), tt.taskID)
			
			if tt.wantError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.taskID, result.ID)
			}
		})
	}
}

func TestTaskService_List(t *testing.T) {
	taskService, _, session := setupTaskTest()
	defer taskService.Stop()
	
	// 여러 테스트 태스크 생성
	tasks := make([]*models.Task, 5)
	for i := 0; i < 5; i++ {
		task, err := taskService.Create(context.Background(), &models.TaskCreateRequest{
			SessionID: session.ID,
			Command:   "echo test",
		})
		require.NoError(t, err)
		tasks[i] = task
	}
	
	tests := []struct {
		name      string
		filter    *models.TaskFilter
		paging    *models.PagingRequest
		wantCount int
	}{
		{
			name:   "All tasks",
			filter: nil,
			paging: &models.PagingRequest{
				Page:  1,
				Limit: 10,
			},
			wantCount: 5,
		},
		{
			name: "Filter by session",
			filter: &models.TaskFilter{
				SessionID: &session.ID,
			},
			paging: &models.PagingRequest{
				Page:  1,
				Limit: 10,
			},
			wantCount: 5,
		},
		{
			name: "Filter by status",
			filter: &models.TaskFilter{
				Status: func() *models.TaskStatus {
					status := models.TaskPending
					return &status
				}(),
			},
			paging: &models.PagingRequest{
				Page:  1,
				Limit: 10,
			},
			wantCount: 5,
		},
		{
			name:   "Pagination",
			filter: nil,
			paging: &models.PagingRequest{
				Page:  1,
				Limit: 2,
			},
			wantCount: 2,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := taskService.List(context.Background(), tt.filter, tt.paging)
			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Len(t, result.Items, tt.wantCount)
		})
	}
}

func TestTaskService_Cancel(t *testing.T) {
	taskService, _, session := setupTaskTest()
	defer taskService.Stop()
	
	// 테스트 태스크 생성
	task, err := taskService.Create(context.Background(), &models.TaskCreateRequest{
		SessionID: session.ID,
		Command:   "sleep 10", // 긴 실행 시간
	})
	require.NoError(t, err)
	
	tests := []struct {
		name      string
		taskID    string
		wantError bool
	}{
		{
			name:      "Valid task cancellation",
			taskID:    task.ID,
			wantError: false,
		},
		{
			name:      "Invalid task ID",
			taskID:    "invalid-task",
			wantError: true,
		},
		{
			name:      "Empty task ID",
			taskID:    "",
			wantError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := taskService.Cancel(context.Background(), tt.taskID)
			
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				
				// 취소된 태스크 상태 확인
				cancelledTask, getErr := taskService.GetByID(context.Background(), tt.taskID)
				assert.NoError(t, getErr)
				assert.Equal(t, models.TaskCancelled, cancelledTask.Status)
			}
		})
	}
}

func TestTaskService_GetActiveTasks(t *testing.T) {
	taskService, _, session := setupTaskTest()
	defer taskService.Stop()
	
	// 여러 태스크 생성
	for i := 0; i < 3; i++ {
		_, err := taskService.Create(context.Background(), &models.TaskCreateRequest{
			SessionID: session.ID,
			Command:   "echo test",
		})
		require.NoError(t, err)
	}
	
	// 활성 태스크 조회
	activeTasks, err := taskService.GetActiveTasks(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, activeTasks)
	
	// 모든 태스크가 활성 상태인지 확인
	for _, task := range activeTasks {
		assert.True(t, task.Status == models.TaskPending || task.Status == models.TaskRunning)
	}
}

func TestTaskService_GetStats(t *testing.T) {
	taskService, _, session := setupTaskTest()
	defer taskService.Stop()
	
	// 여러 태스크 생성
	for i := 0; i < 3; i++ {
		_, err := taskService.Create(context.Background(), &models.TaskCreateRequest{
			SessionID: session.ID,
			Command:   "echo test",
		})
		require.NoError(t, err)
	}
	
	// 통계 조회
	stats, err := taskService.GetStats(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, stats)
	
	// 기본 통계 항목 확인
	assert.Contains(t, stats, "total_tasks")
	assert.Contains(t, stats, "queue_size")
	assert.Contains(t, stats, "max_workers")
	assert.Contains(t, stats, "running")
}

func TestTaskService_ExecuteCommand(t *testing.T) {
	taskService, _, session := setupTaskTest()
	defer taskService.Stop()
	
	tests := []struct {
		name        string
		command     string
		wantError   bool
		expectedOut string
	}{
		{
			name:        "Echo command",
			command:     "echo hello",
			wantError:   false,
			expectedOut: "hello",
		},
		{
			name:      "Dangerous command",
			command:   "rm -rf /",
			wantError: true,
		},
		{
			name:      "Disallowed command",
			command:   "sudo ls",
			wantError: true,
		},
		{
			name:      "Empty command",
			command:   "",
			wantError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := taskService.executeCommand(context.Background(), tt.command, session)
			
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.expectedOut != "" {
					assert.Contains(t, output, tt.expectedOut)
				}
			}
		})
	}
}

func TestTaskService_ValidateCommand(t *testing.T) {
	taskService, _, _ := setupTaskTest()
	defer taskService.Stop()
	
	tests := []struct {
		name      string
		command   string
		wantError bool
	}{
		{
			name:      "Allowed command",
			command:   "echo",
			wantError: false,
		},
		{
			name:      "Dangerous command",
			command:   "rm",
			wantError: true,
		},
		{
			name:      "Disallowed command",
			command:   "unknown-command",
			wantError: true,
		},
		{
			name:      "Git command",
			command:   "git",
			wantError: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := taskService.validateCommand(tt.command)
			
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}