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
	"github.com/stretchr/testify/mock"
	"aicli-web/internal/auth"
	"aicli-web/internal/models"
	"aicli-web/internal/storage"
	"aicli-web/internal/storage/memory"
	"aicli-web/internal/utils"
)

// setupTest는 테스트를 위한 설정을 수행합니다
func setupTest() (*gin.Engine, *WorkspaceController, *auth.JWTManager) {
	gin.SetMode(gin.TestMode)
	utils.RegisterCustomValidators()
	
	// JWT 매니저 생성
	jwtManager := auth.NewJWTManager("test-secret", 3600, 86400)
	
	// 메모리 스토리지 생성
	storage := memory.New()
	
	// 컨트롤러 생성
	controller := NewWorkspaceController(storage)
	
	// 라우터 설정
	router := gin.New()
	
	return router, controller, jwtManager
}

// getAuthToken는 테스트용 JWT 토큰을 생성합니다
func getAuthToken(jwtManager *auth.JWTManager, userID, userName, role string) string {
	token, _ := jwtManager.GenerateAccessToken(userID, userName, role)
	return token
}

func TestListWorkspaces(t *testing.T) {
	router, controller, jwtManager := setupTest()
	
	// 테스트 사용자
	userID := "test-user-id"
	token := getAuthToken(jwtManager, userID, "testuser", "user")
	
	// 라우트 설정
	router.GET("/workspaces", func(c *gin.Context) {
		c.Set("claims", &auth.Claims{
			UserID:   userID,
			UserName: "testuser",
			Role:     "user",
		})
		controller.ListWorkspaces(c)
	})
	
	// 요청 생성
	req, _ := http.NewRequest("GET", "/workspaces?page=1&limit=10", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	
	// 응답 기록
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// 검증
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response models.PaginationResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.Success)
	assert.NotNil(t, response.Data)
	assert.Equal(t, 1, response.Meta.Page)
	assert.Equal(t, 10, response.Meta.Limit)
}

func TestCreateWorkspace(t *testing.T) {
	router, controller, jwtManager := setupTest()
	
	// 테스트 사용자
	userID := "test-user-id"
	token := getAuthToken(jwtManager, userID, "testuser", "user")
	
	// 라우트 설정
	router.POST("/workspaces", func(c *gin.Context) {
		c.Set("claims", &auth.Claims{
			UserID:   userID,
			UserName: "testuser",
			Role:     "user",
		})
		controller.CreateWorkspace(c)
	})
	
	// 요청 데이터
	workspace := models.Workspace{
		Name:        "test-workspace",
		ProjectPath: "/tmp/test-project",
		ClaudeKey:   "test-key",
	}
	body, _ := json.Marshal(workspace)
	
	// 요청 생성
	req, _ := http.NewRequest("POST", "/workspaces", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	
	// 응답 기록
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// 검증
	assert.Equal(t, http.StatusCreated, w.Code)
	
	var response models.SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.Success)
	assert.Contains(t, response.Message, "생성")
}

func TestGetWorkspace(t *testing.T) {
	router, controller, jwtManager := setupTest()
	
	// 테스트 사용자
	userID := "test-user-id"
	token := getAuthToken(jwtManager, userID, "testuser", "user")
	
	// 테스트 워크스페이스 생성
	workspace := &models.Workspace{
		Name:        "test-workspace",
		ProjectPath: "/tmp/test-project",
		OwnerID:     userID,
		ClaudeKey:   "test-key",
	}
	err := controller.storage.Workspace().Create(context.Background(), workspace)
	assert.NoError(t, err)
	
	// 라우트 설정
	router.GET("/workspaces/:id", func(c *gin.Context) {
		c.Set("claims", &auth.Claims{
			UserID:   userID,
			UserName: "testuser",
			Role:     "user",
		})
		controller.GetWorkspace(c)
	})
	
	// 요청 생성
	req, _ := http.NewRequest("GET", "/workspaces/"+workspace.ID, nil)
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

func TestUpdateWorkspace(t *testing.T) {
	router, controller, jwtManager := setupTest()
	
	// 테스트 사용자
	userID := "test-user-id"
	token := getAuthToken(jwtManager, userID, "testuser", "user")
	
	// 테스트 워크스페이스 생성
	workspace := &models.Workspace{
		Name:        "test-workspace",
		ProjectPath: "/tmp/test-project",
		OwnerID:     userID,
		ClaudeKey:   "test-key",
	}
	err := controller.storage.Workspace().Create(context.Background(), workspace)
	assert.NoError(t, err)
	
	// 라우트 설정
	router.PUT("/workspaces/:id", func(c *gin.Context) {
		c.Set("claims", &auth.Claims{
			UserID:   userID,
			UserName: "testuser",
			Role:     "user",
		})
		controller.UpdateWorkspace(c)
	})
	
	// 업데이트 데이터
	updates := map[string]interface{}{
		"name": "updated-workspace",
	}
	body, _ := json.Marshal(updates)
	
	// 요청 생성
	req, _ := http.NewRequest("PUT", "/workspaces/"+workspace.ID, bytes.NewBuffer(body))
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

func TestDeleteWorkspace(t *testing.T) {
	router, controller, jwtManager := setupTest()
	
	// 테스트 사용자
	userID := "test-user-id"
	token := getAuthToken(jwtManager, userID, "testuser", "user")
	
	// 테스트 워크스페이스 생성
	workspace := &models.Workspace{
		Name:        "test-workspace",
		ProjectPath: "/tmp/test-project",
		OwnerID:     userID,
		ClaudeKey:   "test-key",
	}
	err := controller.storage.Workspace().Create(context.Background(), workspace)
	assert.NoError(t, err)
	
	// 라우트 설정
	router.DELETE("/workspaces/:id", func(c *gin.Context) {
		c.Set("claims", &auth.Claims{
			UserID:   userID,
			UserName: "testuser",
			Role:     "user",
		})
		controller.DeleteWorkspace(c)
	})
	
	// 요청 생성
	req, _ := http.NewRequest("DELETE", "/workspaces/"+workspace.ID, nil)
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

func TestWorkspacePermissions(t *testing.T) {
	router, controller, jwtManager := setupTest()
	
	// 두 명의 사용자
	ownerID := "owner-id"
	otherUserID := "other-user-id"
	
	// 소유자가 워크스페이스 생성
	workspace := &models.Workspace{
		Name:        "owner-workspace",
		ProjectPath: "/tmp/owner-project",
		OwnerID:     ownerID,
		ClaudeKey:   "test-key",
	}
	err := controller.storage.Workspace().Create(context.Background(), workspace)
	assert.NoError(t, err)
	
	// 다른 사용자의 토큰
	otherUserToken := getAuthToken(jwtManager, otherUserID, "otheruser", "user")
	
	// 라우트 설정
	router.GET("/workspaces/:id", func(c *gin.Context) {
		c.Set("claims", &auth.Claims{
			UserID:   otherUserID,
			UserName: "otheruser",
			Role:     "user",
		})
		controller.GetWorkspace(c)
	})
	
	// 다른 사용자가 소유자의 워크스페이스에 접근 시도
	req, _ := http.NewRequest("GET", "/workspaces/"+workspace.ID, nil)
	req.Header.Set("Authorization", "Bearer "+otherUserToken)
	
	// 응답 기록
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// 검증 - 권한 없음 에러
	assert.Equal(t, http.StatusForbidden, w.Code)
}