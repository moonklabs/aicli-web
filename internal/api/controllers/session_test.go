package controllers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/services"
	"github.com/aicli/aicli-web/internal/storage/memory"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupSessionTest() (*gin.Engine, *services.SessionService, *models.Project) {
	gin.SetMode(gin.TestMode)
	
	storage := memory.New()
	projectService := services.NewProjectService(storage)
	sessionService := services.NewSessionService(storage, projectService, nil)
	sessionController := NewSessionController(sessionService)
	
	// 테스트용 워크스페이스와 프로젝트 생성
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
		Path:        "/test/path",
		Status:      models.ProjectActive,
		Settings:    map[string]interface{}{},
	}
	_ = storage.Project().Create(nil, project)
	
	router := gin.New()
	
	// 라우트 설정
	router.POST("/projects/:id/sessions", sessionController.Create)
	router.GET("/sessions", sessionController.List)
	router.GET("/sessions/active", sessionController.GetActiveSessions)
	router.GET("/sessions/:id", sessionController.GetByID)
	router.DELETE("/sessions/:id", sessionController.Terminate)
	router.PUT("/sessions/:id/activity", sessionController.UpdateActivity)
	
	return router, sessionService, project
}

func TestSessionController_Create(t *testing.T) {
	router, _, project := setupSessionTest()
	
	tests := []struct {
		name       string
		projectID  string
		body       interface{}
		wantStatus int
	}{
		{
			name:      "Valid request",
			projectID: project.ID,
			body: models.SessionCreateRequest{
				Metadata: map[string]string{
					"key": "value",
				},
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "Missing project ID",
			projectID:  "",
			body:       models.SessionCreateRequest{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "Invalid project ID",
			projectID:  "invalid-project",
			body:       models.SessionCreateRequest{},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "Invalid body",
			projectID:  project.ID,
			body:       "invalid",
			wantStatus: http.StatusBadRequest,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest("POST", fmt.Sprintf("/projects/%s/sessions", tt.projectID), bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			assert.Equal(t, tt.wantStatus, w.Code)
			
			if tt.wantStatus == http.StatusCreated {
				var resp models.SessionResponse
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, project.ID, resp.ProjectID)
				assert.Equal(t, models.SessionPending, resp.Status)
			}
		})
	}
}

func TestSessionController_List(t *testing.T) {
	router, sessionService, project := setupSessionTest()
	
	// 테스트 세션 생성
	sessions := make([]*models.Session, 5)
	for i := 0; i < 5; i++ {
		session, err := sessionService.Create(nil, &models.SessionCreateRequest{
			ProjectID: project.ID,
		})
		require.NoError(t, err)
		
		if i < 3 {
			_ = sessionService.UpdateStatus(nil, session.ID, models.SessionActive)
		}
		sessions[i] = session
	}
	
	tests := []struct {
		name        string
		query       string
		wantStatus  int
		wantCount   int
	}{
		{
			name:       "All sessions",
			query:      "",
			wantStatus: http.StatusOK,
			wantCount:  5,
		},
		{
			name:       "Filter by project",
			query:      fmt.Sprintf("?project_id=%s", project.ID),
			wantStatus: http.StatusOK,
			wantCount:  5,
		},
		{
			name:       "Filter by status",
			query:      "?status=active",
			wantStatus: http.StatusOK,
			wantCount:  3,
		},
		{
			name:       "Filter active only",
			query:      "?active=true",
			wantStatus: http.StatusOK,
			wantCount:  3,
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
			req := httptest.NewRequest("GET", "/sessions"+tt.query, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			assert.Equal(t, tt.wantStatus, w.Code)
			
			if tt.wantStatus == http.StatusOK {
				var resp models.PagingResponse[*models.SessionResponse]
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Len(t, resp.Items, tt.wantCount)
			}
		})
	}
}

func TestSessionController_GetByID(t *testing.T) {
	router, sessionService, project := setupSessionTest()
	
	// 테스트 세션 생성
	session, err := sessionService.Create(nil, &models.SessionCreateRequest{
		ProjectID: project.ID,
	})
	require.NoError(t, err)
	
	tests := []struct {
		name       string
		sessionID  string
		wantStatus int
	}{
		{
			name:       "Valid session ID",
			sessionID:  session.ID,
			wantStatus: http.StatusOK,
		},
		{
			name:       "Invalid session ID",
			sessionID:  "invalid-session",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "Empty session ID",
			sessionID:  "",
			wantStatus: http.StatusBadRequest,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/sessions/%s", tt.sessionID), nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			assert.Equal(t, tt.wantStatus, w.Code)
			
			if tt.wantStatus == http.StatusOK {
				var resp models.SessionResponse
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, session.ID, resp.ID)
			}
		})
	}
}

func TestSessionController_Terminate(t *testing.T) {
	router, sessionService, project := setupSessionTest()
	
	// 테스트 세션 생성
	session, err := sessionService.Create(nil, &models.SessionCreateRequest{
		ProjectID: project.ID,
	})
	require.NoError(t, err)
	
	tests := []struct {
		name       string
		sessionID  string
		wantStatus int
	}{
		{
			name:       "Valid session ID",
			sessionID:  session.ID,
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "Invalid session ID",
			sessionID:  "invalid-session",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "Empty session ID",
			sessionID:  "",
			wantStatus: http.StatusBadRequest,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("DELETE", fmt.Sprintf("/sessions/%s", tt.sessionID), nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

func TestSessionController_UpdateActivity(t *testing.T) {
	router, sessionService, project := setupSessionTest()
	
	// 테스트 세션 생성
	session, err := sessionService.Create(nil, &models.SessionCreateRequest{
		ProjectID: project.ID,
	})
	require.NoError(t, err)
	
	// 세션을 활성화
	err = sessionService.UpdateStatus(nil, session.ID, models.SessionActive)
	require.NoError(t, err)
	
	// 기존 LastActive 시간 저장
	oldSession, err := sessionService.GetByID(nil, session.ID)
	require.NoError(t, err)
	oldLastActive := oldSession.LastActive
	
	// 약간 대기
	time.Sleep(10 * time.Millisecond)
	
	// 활동 업데이트 요청
	req := httptest.NewRequest("PUT", fmt.Sprintf("/sessions/%s/activity", session.ID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusNoContent, w.Code)
	
	// 업데이트 확인
	updatedSession, err := sessionService.GetByID(nil, session.ID)
	require.NoError(t, err)
	assert.True(t, updatedSession.LastActive.After(oldLastActive))
}

func TestSessionController_GetActiveSessions(t *testing.T) {
	router, sessionService, project := setupSessionTest()
	
	// 테스트 세션 생성
	activeSessions := make([]*models.Session, 3)
	for i := 0; i < 3; i++ {
		session, err := sessionService.Create(nil, &models.SessionCreateRequest{
			ProjectID: project.ID,
		})
		require.NoError(t, err)
		
		err = sessionService.UpdateStatus(nil, session.ID, models.SessionActive)
		require.NoError(t, err)
		
		activeSessions[i] = session
	}
	
	// 종료된 세션도 생성
	endedSession, err := sessionService.Create(nil, &models.SessionCreateRequest{
		ProjectID: project.ID,
	})
	require.NoError(t, err)
	err = sessionService.UpdateStatus(nil, endedSession.ID, models.SessionActive)
	require.NoError(t, err)
	err = sessionService.UpdateStatus(nil, endedSession.ID, models.SessionEnding)
	require.NoError(t, err)
	err = sessionService.UpdateStatus(nil, endedSession.ID, models.SessionEnded)
	require.NoError(t, err)
	
	// 활성 세션 조회
	req := httptest.NewRequest("GET", "/sessions/active", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var resp []*models.SessionResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Len(t, resp, 3)
	
	// 모든 응답이 활성 세션인지 확인
	for _, session := range resp {
		assert.True(t, session.Status == models.SessionActive || session.Status == models.SessionIdle)
	}
}