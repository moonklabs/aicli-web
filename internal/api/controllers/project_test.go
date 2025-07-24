package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/aicli/aicli-web/internal/auth"
	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/services"
	"github.com/aicli/aicli-web/internal/storage/memory"
	"github.com/aicli/aicli-web/internal/utils"
)

// setupProjectTest는 프로젝트 테스트를 위한 설정을 수행합니다
func setupProjectTest() (*gin.Engine, *ProjectController, *WorkspaceController, *auth.JWTManager) {
	gin.SetMode(gin.TestMode)
	utils.RegisterCustomValidators()
	
	// JWT 매니저 생성
	jwtManager := auth.NewJWTManager("test-secret", 3600, 86400)
	
	// 메모리 스토리지 생성
	storage := memory.New()
	
	// 서비스 생성
	workspaceService := services.NewWorkspaceService(storage)
	dockerWorkspaceService := services.NewDockerWorkspaceService(nil, storage, nil)
	
	// 컨트롤러 생성
	projectController := NewProjectController(storage)
	workspaceController := NewWorkspaceController(workspaceService, dockerWorkspaceService)
	
	// 라우터 설정
	router := gin.New()
	
	return router, projectController, workspaceController, jwtManager
}

func TestCreateProject(t *testing.T) {
	router, projectController, workspaceController, jwtManager := setupProjectTest()
	
	// 테스트 사용자
	userID := "test-user-id"
	token := getAuthToken(jwtManager, userID, "testuser", "user")
	
	// 워크스페이스 생성
	workspace := &models.Workspace{
		Name:        "test-workspace",
		ProjectPath: "/tmp/test-workspace",
		OwnerID:     userID,
		ClaudeKey:   "test-key",
	}
	err := workspaceController.storage.Workspace().Create(context.Background(), workspace)
	assert.NoError(t, err)
	
	// 라우트 설정
	router.POST("/workspaces/:workspace_id/projects", func(c *gin.Context) {
		c.Set("claims", &auth.Claims{
			UserID:   userID,
			UserName: "testuser",
			Role:     "user",
		})
		projectController.CreateProject(c)
	})
	
	// 요청 데이터
	project := models.Project{
		Name:        "test-project",
		Path:        "/tmp/test-project",
		Description: "Test project",
		Language:    "go",
	}
	body, _ := json.Marshal(project)
	
	// 요청 생성
	req, _ := http.NewRequest("POST", "/workspaces/"+workspace.ID+"/projects", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	
	// 응답 기록
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// 검증
	assert.Equal(t, http.StatusCreated, w.Code)
	
	var response models.SuccessResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.Success)
	assert.Contains(t, response.Message, "생성")
}

func TestListProjects(t *testing.T) {
	router, projectController, workspaceController, jwtManager := setupProjectTest()
	
	// 테스트 사용자
	userID := "test-user-id"
	token := getAuthToken(jwtManager, userID, "testuser", "user")
	
	// 워크스페이스 생성
	workspace := &models.Workspace{
		Name:        "test-workspace",
		ProjectPath: "/tmp/test-workspace",
		OwnerID:     userID,
		ClaudeKey:   "test-key",
	}
	err := workspaceController.storage.Workspace().Create(context.Background(), workspace)
	assert.NoError(t, err)
	
	// 프로젝트 생성
	project1 := &models.Project{
		WorkspaceID: workspace.ID,
		Name:        "project-1",
		Path:        "/tmp/project-1",
	}
	project2 := &models.Project{
		WorkspaceID: workspace.ID,
		Name:        "project-2",
		Path:        "/tmp/project-2",
	}
	err = projectController.storage.Project().Create(context.Background(), project1)
	assert.NoError(t, err)
	err = projectController.storage.Project().Create(context.Background(), project2)
	assert.NoError(t, err)
	
	// 라우트 설정
	router.GET("/workspaces/:workspace_id/projects", func(c *gin.Context) {
		c.Set("claims", &auth.Claims{
			UserID:   userID,
			UserName: "testuser",
			Role:     "user",
		})
		projectController.ListProjects(c)
	})
	
	// 요청 생성
	req, _ := http.NewRequest("GET", "/workspaces/"+workspace.ID+"/projects?page=1&limit=10", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	
	// 응답 기록
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// 검증
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response models.PaginationResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.Success)
	assert.Equal(t, 2, response.Meta.Total)
}

func TestGetProject(t *testing.T) {
	router, projectController, workspaceController, jwtManager := setupProjectTest()
	
	// 테스트 사용자
	userID := "test-user-id"
	token := getAuthToken(jwtManager, userID, "testuser", "user")
	
	// 워크스페이스 생성
	workspace := &models.Workspace{
		Name:        "test-workspace",
		ProjectPath: "/tmp/test-workspace",
		OwnerID:     userID,
		ClaudeKey:   "test-key",
	}
	err := workspaceController.storage.Workspace().Create(context.Background(), workspace)
	assert.NoError(t, err)
	
	// 프로젝트 생성
	project := &models.Project{
		WorkspaceID: workspace.ID,
		Name:        "test-project",
		Path:        "/tmp/test-project",
		Description: "Test project",
	}
	err = projectController.storage.Project().Create(context.Background(), project)
	assert.NoError(t, err)
	
	// 라우트 설정
	router.GET("/projects/:id", func(c *gin.Context) {
		c.Set("claims", &auth.Claims{
			UserID:   userID,
			UserName: "testuser",
			Role:     "user",
		})
		projectController.GetProject(c)
	})
	
	// 요청 생성
	req, _ := http.NewRequest("GET", "/projects/"+project.ID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	
	// 응답 기록
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// 검증
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response models.SuccessResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.Success)
}

func TestUpdateProject(t *testing.T) {
	router, projectController, workspaceController, jwtManager := setupProjectTest()
	
	// 테스트 사용자
	userID := "test-user-id"
	token := getAuthToken(jwtManager, userID, "testuser", "user")
	
	// 워크스페이스 생성
	workspace := &models.Workspace{
		Name:        "test-workspace",
		ProjectPath: "/tmp/test-workspace",
		OwnerID:     userID,
		ClaudeKey:   "test-key",
	}
	err := workspaceController.storage.Workspace().Create(context.Background(), workspace)
	assert.NoError(t, err)
	
	// 프로젝트 생성
	project := &models.Project{
		WorkspaceID: workspace.ID,
		Name:        "test-project",
		Path:        "/tmp/test-project",
		Description: "Test project",
	}
	err = projectController.storage.Project().Create(context.Background(), project)
	assert.NoError(t, err)
	
	// 라우트 설정
	router.PUT("/projects/:id", func(c *gin.Context) {
		c.Set("claims", &auth.Claims{
			UserID:   userID,
			UserName: "testuser",
			Role:     "user",
		})
		projectController.UpdateProject(c)
	})
	
	// 업데이트 데이터
	updates := map[string]interface{}{
		"name":        "updated-project",
		"description": "Updated description",
	}
	body, _ := json.Marshal(updates)
	
	// 요청 생성
	req, _ := http.NewRequest("PUT", "/projects/"+project.ID, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	
	// 응답 기록
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// 검증
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response models.SuccessResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.Success)
	assert.Contains(t, response.Message, "수정")
}

func TestDeleteProject(t *testing.T) {
	router, projectController, workspaceController, jwtManager := setupProjectTest()
	
	// 테스트 사용자
	userID := "test-user-id"
	token := getAuthToken(jwtManager, userID, "testuser", "user")
	
	// 워크스페이스 생성
	workspace := &models.Workspace{
		Name:        "test-workspace",
		ProjectPath: "/tmp/test-workspace",
		OwnerID:     userID,
		ClaudeKey:   "test-key",
	}
	err := workspaceController.storage.Workspace().Create(context.Background(), workspace)
	assert.NoError(t, err)
	
	// 프로젝트 생성
	project := &models.Project{
		WorkspaceID: workspace.ID,
		Name:        "test-project",
		Path:        "/tmp/test-project",
		Description: "Test project",
	}
	err = projectController.storage.Project().Create(context.Background(), project)
	assert.NoError(t, err)
	
	// 라우트 설정
	router.DELETE("/projects/:id", func(c *gin.Context) {
		c.Set("claims", &auth.Claims{
			UserID:   userID,
			UserName: "testuser",
			Role:     "user",
		})
		projectController.DeleteProject(c)
	})
	
	// 요청 생성
	req, _ := http.NewRequest("DELETE", "/projects/"+project.ID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	
	// 응답 기록
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// 검증
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response models.SuccessResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.Success)
	assert.Contains(t, response.Message, "삭제")
}

func TestProjectPermissions(t *testing.T) {
	router, projectController, workspaceController, jwtManager := setupProjectTest()
	
	// 두 명의 사용자
	ownerID := "owner-id"
	otherUserID := "other-user-id"
	
	// 소유자가 워크스페이스 생성
	workspace := &models.Workspace{
		Name:        "owner-workspace",
		ProjectPath: "/tmp/owner-workspace",
		OwnerID:     ownerID,
		ClaudeKey:   "test-key",
	}
	err := workspaceController.storage.Workspace().Create(context.Background(), workspace)
	assert.NoError(t, err)
	
	// 소유자가 프로젝트 생성
	project := &models.Project{
		WorkspaceID: workspace.ID,
		Name:        "owner-project",
		Path:        "/tmp/owner-project",
	}
	err = projectController.storage.Project().Create(context.Background(), project)
	assert.NoError(t, err)
	
	// 다른 사용자의 토큰
	otherUserToken := getAuthToken(jwtManager, otherUserID, "otheruser", "user")
	
	// 라우트 설정
	router.GET("/projects/:id", func(c *gin.Context) {
		c.Set("claims", &auth.Claims{
			UserID:   otherUserID,
			UserName: "otheruser",
			Role:     "user",
		})
		projectController.GetProject(c)
	})
	
	// 다른 사용자가 소유자의 프로젝트에 접근 시도
	req, _ := http.NewRequest("GET", "/projects/"+project.ID, nil)
	req.Header.Set("Authorization", "Bearer "+otherUserToken)
	
	// 응답 기록
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// 검증 - 권한 없음 에러
	assert.Equal(t, http.StatusForbidden, w.Code)
}