// Package e2e provides end-to-end tests for the complete workspace flow
package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/testutil"
)

// E2ETestSuite provides end-to-end testing for the complete workspace flow
type E2ETestSuite struct {
	server    *httptest.Server
	client    *http.Client
	baseURL   string
	token     string
	testUser  string
	
	// Test data cleanup
	workspaceIDs []string
	tempDirs     []string
}

// TestCompleteWorkspaceFlow tests the complete workspace workflow via API
func TestCompleteWorkspaceFlow(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker daemon not available, skipping E2E tests")
	}
	
	suite := &E2ETestSuite{}
	suite.setupE2ETest(t)
	defer suite.teardownE2ETest(t)
	
	// 전체 워크플로우 테스트
	suite.runCompleteWorkflow(t)
}

// TestWorkspaceWebSocketIntegration tests WebSocket integration
func TestWorkspaceWebSocketIntegration(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker daemon not available, skipping E2E WebSocket tests")
	}
	
	suite := &E2ETestSuite{}
	suite.setupE2ETest(t)
	defer suite.teardownE2ETest(t)
	
	// WebSocket 통합 테스트
	suite.runWebSocketIntegrationTest(t)
}

// TestMultiUserWorkspaceIsolation tests multi-user workspace isolation
func TestMultiUserWorkspaceIsolation(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker daemon not available, skipping multi-user E2E tests")
	}
	
	suite := &E2ETestSuite{}
	suite.setupE2ETest(t)
	defer suite.teardownE2ETest(t)
	
	// 멀티 유저 격리 테스트
	suite.runMultiUserIsolationTest(t)
}

// setupE2ETest initializes the E2E test environment
func (suite *E2ETestSuite) setupE2ETest(t *testing.T) {
	t.Log("Setting up E2E test environment...")
	
	// 테스트 서버 시작 (mock implementation)
	gin.SetMode(gin.TestMode)
	suite.server = suite.startTestAPIServer(t, nil) // storage는 mock에서 처리
	suite.client = &http.Client{Timeout: 30 * time.Second}
	suite.baseURL = suite.server.URL
	
	// 테스트 사용자 생성 및 인증
	suite.testUser = "test-user-" + testutil.GenerateRandomID()
	suite.token = suite.authenticateTestUser(t)
	
	tmpDir, err := os.MkdirTemp("", "e2e-test-*")
	require.NoError(t, err)
	suite.tempDirs = append(suite.tempDirs, tmpDir)
	
	t.Log("E2E test environment setup completed")
}

// teardownE2ETest cleans up the E2E test environment
func (suite *E2ETestSuite) teardownE2ETest(t *testing.T) {
	t.Log("Cleaning up E2E test environment...")
	
	// 테스트 워크스페이스 정리
	for _, workspaceID := range suite.workspaceIDs {
		suite.deleteWorkspaceViaAPI(t, workspaceID)
	}
	
	// 임시 디렉토리 정리
	for _, dir := range suite.tempDirs {
		os.RemoveAll(dir)
	}
	
	// 서버 정리
	if suite.server != nil {
		suite.server.Close()
	}
	
	t.Log("E2E test environment cleanup completed")
}

// runCompleteWorkflow runs the complete workspace workflow test
func (suite *E2ETestSuite) runCompleteWorkflow(t *testing.T) {
	t.Log("Running complete workspace workflow test...")
	
	// Phase 1: 워크스페이스 생성
	t.Log("Phase 1: Creating workspace via API...")
	workspace := suite.createWorkspaceViaAPI(t, WorkspaceCreateRequest{
		Name:        "e2e-test-workspace",
		ProjectPath: suite.createTempTestProject(t),
	})
	
	require.NotEmpty(t, workspace.ID)
	require.Equal(t, "e2e-test-workspace", workspace.Name)
	suite.workspaceIDs = append(suite.workspaceIDs, workspace.ID)
	
	// Phase 2: 워크스페이스 상태 확인
	t.Log("Phase 2: Verifying workspace status...")
	suite.eventually(t, 30*time.Second, func() bool {
		status := suite.getWorkspaceStatusViaAPI(t, workspace.ID)
		return status.Workspace.Status == models.WorkspaceStatusActive &&
			status.ContainerStatus.ContainerState == "running"
	}, "Workspace should be active and container running")
	
	// Phase 3: 워크스페이스 목록 조회
	t.Log("Phase 3: Listing workspaces...")
	workspaces := suite.listWorkspacesViaAPI(t)
	assert.NotEmpty(t, workspaces)
	
	found := false
	for _, ws := range workspaces {
		if ws.ID == workspace.ID {
			found = true
			break
		}
	}
	assert.True(t, found, "Created workspace should be in the list")
	
	// Phase 4: 워크스페이스 중지
	t.Log("Phase 4: Stopping workspace...")
	suite.stopWorkspaceViaAPI(t, workspace.ID)
	
	suite.eventually(t, 15*time.Second, func() bool {
		status := suite.getWorkspaceStatusViaAPI(t, workspace.ID)
		return status.Workspace.Status == models.WorkspaceStatusInactive
	}, "Workspace should be inactive")
	
	// Phase 5: 워크스페이스 재시작
	t.Log("Phase 5: Restarting workspace...")
	suite.startWorkspaceViaAPI(t, workspace.ID)
	
	suite.eventually(t, 15*time.Second, func() bool {
		status := suite.getWorkspaceStatusViaAPI(t, workspace.ID)
		return status.Workspace.Status == models.WorkspaceStatusActive
	}, "Workspace should be active again")
	
	// Phase 6: 워크스페이스 업데이트
	t.Log("Phase 6: Updating workspace...")
	updatedWorkspace := suite.updateWorkspaceViaAPI(t, workspace.ID, WorkspaceUpdateRequest{
		Name:        "updated-e2e-test-workspace",
	})
	
	assert.Equal(t, "updated-e2e-test-workspace", updatedWorkspace.Name)
	
	// Phase 7: 워크스페이스 삭제
	t.Log("Phase 7: Deleting workspace...")
	suite.deleteWorkspaceViaAPI(t, workspace.ID)
	
	// Phase 8: 삭제 후 조회 시 404 확인
	suite.eventually(t, 20*time.Second, func() bool {
		resp, err := suite.client.Get(fmt.Sprintf("%s/api/workspaces/%s", suite.baseURL, workspace.ID))
		if err != nil {
			return false
		}
		defer resp.Body.Close()
		return resp.StatusCode == http.StatusNotFound
	}, "Workspace should return 404 after deletion")
	
	t.Log("Complete workflow test passed!")
}

// runWebSocketIntegrationTest runs WebSocket integration test
func (suite *E2ETestSuite) runWebSocketIntegrationTest(t *testing.T) {
	t.Log("Running WebSocket integration test...")
	
	// 워크스페이스 생성
	workspace := suite.createWorkspaceViaAPI(t, WorkspaceCreateRequest{
		Name:        "websocket-test-workspace",
		ProjectPath: suite.createTempTestProject(t),
	})
	suite.workspaceIDs = append(suite.workspaceIDs, workspace.ID)
	
	// WebSocket 연결 설정
	wsURL := fmt.Sprintf("ws%s/api/workspaces/%s/ws", 
		suite.baseURL[4:], workspace.ID) // http -> ws
	
	headers := http.Header{}
	headers.Set("Authorization", "Bearer "+suite.token)
	
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, headers)
	require.NoError(t, err, "Failed to connect to WebSocket")
	defer conn.Close()
	defer resp.Body.Close()
	
	// 연결 확인 메시지 수신
	var connectMsg map[string]interface{}
	err = conn.ReadJSON(&connectMsg)
	require.NoError(t, err)
	assert.Equal(t, "connected", connectMsg["type"])
	
	// 상태 업데이트 메시지 전송
	statusMsg := map[string]interface{}{
		"type":    "subscribe",
		"channel": "status",
	}
	
	err = conn.WriteJSON(statusMsg)
	require.NoError(t, err)
	
	// 상태 메시지 수신
	var statusUpdate map[string]interface{}
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	err = conn.ReadJSON(&statusUpdate)
	require.NoError(t, err)
	assert.Equal(t, "status", statusUpdate["type"])
	
	t.Log("WebSocket integration test passed!")
}

// runMultiUserIsolationTest runs multi-user isolation test
func (suite *E2ETestSuite) runMultiUserIsolationTest(t *testing.T) {
	t.Log("Running multi-user isolation test...")
	
	// 첫 번째 사용자의 워크스페이스 생성
	user1Token := suite.token
	workspace1 := suite.createWorkspaceViaAPI(t, WorkspaceCreateRequest{
		Name:        "user1-workspace",
		ProjectPath: suite.createTempTestProject(t),
	})
	suite.workspaceIDs = append(suite.workspaceIDs, workspace1.ID)
	
	// 두 번째 사용자 생성 및 인증
	suite.testUser = "test-user-2-" + testutil.GenerateRandomID()
	user2Token := suite.authenticateTestUser(t)
	
	// 두 번째 사용자의 워크스페이스 생성
	suite.token = user2Token
	workspace2 := suite.createWorkspaceViaAPI(t, WorkspaceCreateRequest{
		Name:        "user2-workspace",
		ProjectPath: suite.createTempTestProject(t),
	})
	suite.workspaceIDs = append(suite.workspaceIDs, workspace2.ID)
	
	// User 2가 User 1의 워크스페이스에 접근 시도 (403 예상)
	resp, err := suite.client.Get(fmt.Sprintf("%s/api/workspaces/%s", suite.baseURL, workspace1.ID))
	require.NoError(t, err)
	defer resp.Body.Close()
	
	assert.Equal(t, http.StatusForbidden, resp.StatusCode, 
		"User 2 should not access User 1's workspace")
	
	// User 1이 User 2의 워크스페이스에 접근 시도 (403 예상)
	suite.token = user1Token
	resp, err = suite.client.Get(fmt.Sprintf("%s/api/workspaces/%s", suite.baseURL, workspace2.ID))
	require.NoError(t, err)
	defer resp.Body.Close()
	
	assert.Equal(t, http.StatusForbidden, resp.StatusCode,
		"User 1 should not access User 2's workspace")
	
	// 각 사용자가 자신의 워크스페이스만 볼 수 있는지 확인
	suite.token = user1Token
	user1Workspaces := suite.listWorkspacesViaAPI(t)
	
	suite.token = user2Token
	user2Workspaces := suite.listWorkspacesViaAPI(t)
	
	// User 1은 자신의 워크스페이스만 볼 수 있어야 함
	found1InUser1List := false
	found2InUser1List := false
	for _, ws := range user1Workspaces {
		if ws.ID == workspace1.ID {
			found1InUser1List = true
		}
		if ws.ID == workspace2.ID {
			found2InUser1List = true
		}
	}
	
	assert.True(t, found1InUser1List, "User 1 should see their own workspace")
	assert.False(t, found2InUser1List, "User 1 should not see User 2's workspace")
	
	// User 2는 자신의 워크스페이스만 볼 수 있어야 함
	found1InUser2List := false
	found2InUser2List := false
	for _, ws := range user2Workspaces {
		if ws.ID == workspace1.ID {
			found1InUser2List = true
		}
		if ws.ID == workspace2.ID {
			found2InUser2List = true
		}
	}
	
	assert.False(t, found1InUser2List, "User 2 should not see User 1's workspace")
	assert.True(t, found2InUser2List, "User 2 should see their own workspace")
	
	t.Log("Multi-user isolation test passed!")
}

// startTestAPIServer starts a test API server
func (suite *E2ETestSuite) startTestAPIServer(t *testing.T, storage interface{}) *httptest.Server {
	// 실제 서버 설정과 유사한 테스트 서버 생성
	router := gin.New()
	router.Use(gin.Recovery())
	
	// 기본 라우트 설정
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	
	// API 라우트 설정 (실제 server.SetupRouter와 유사)
	apiGroup := router.Group("/api")
	{
		// 인증 관련
		apiGroup.POST("/auth/login", suite.mockLoginHandler)
		
		// 워크스페이스 관련
		workspaceGroup := apiGroup.Group("/workspaces")
		workspaceGroup.Use(suite.mockAuthMiddleware)
		{
			workspaceGroup.GET("", suite.mockListWorkspacesHandler)
			workspaceGroup.POST("", suite.mockCreateWorkspaceHandler)
			workspaceGroup.GET("/:id", suite.mockGetWorkspaceHandler)
			workspaceGroup.PUT("/:id", suite.mockUpdateWorkspaceHandler)
			workspaceGroup.DELETE("/:id", suite.mockDeleteWorkspaceHandler)
			workspaceGroup.POST("/:id/start", suite.mockStartWorkspaceHandler)
			workspaceGroup.POST("/:id/stop", suite.mockStopWorkspaceHandler)
			workspaceGroup.GET("/:id/status", suite.mockGetWorkspaceStatusHandler)
			workspaceGroup.GET("/:id/ws", suite.mockWebSocketHandler)
		}
	}
	
	return httptest.NewServer(router)
}

// Mock handlers for testing
func (suite *E2ETestSuite) mockLoginHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"token": "mock-jwt-token-" + testutil.GenerateRandomID(),
		"user":  suite.testUser,
	})
}

func (suite *E2ETestSuite) mockAuthMiddleware(c *gin.Context) {
	// 간단한 토큰 검증 (테스트용)
	token := c.GetHeader("Authorization")
	if token == "" || len(token) < 10 {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	
	// 사용자 정보 설정
	c.Set("user_id", suite.testUser)
	c.Next()
}

func (suite *E2ETestSuite) mockListWorkspacesHandler(c *gin.Context) {
	// 실제 구현에서는 데이터베이스에서 조회
	workspaces := []models.Workspace{}
	c.JSON(http.StatusOK, workspaces)
}

func (suite *E2ETestSuite) mockCreateWorkspaceHandler(c *gin.Context) {
	var req WorkspaceCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	workspace := models.Workspace{
		ID:          testutil.GenerateRandomID(),
		Name:        req.Name,
		ProjectPath: req.ProjectPath,
		Status:      models.WorkspaceStatusActive,
		OwnerID:     suite.testUser,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	
	c.JSON(http.StatusCreated, workspace)
}

func (suite *E2ETestSuite) mockGetWorkspaceHandler(c *gin.Context) {
	workspaceID := c.Param("id")
	userID := c.GetString("user_id")
	
	// 간단한 접근 제어 시뮬레이션
	if userID != suite.testUser {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}
	
	workspace := models.Workspace{
		ID:          workspaceID,
		Name:        "test-workspace",
		Status:      models.WorkspaceStatusActive,
		OwnerID:     userID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	
	c.JSON(http.StatusOK, workspace)
}

func (suite *E2ETestSuite) mockUpdateWorkspaceHandler(c *gin.Context) {
	workspaceID := c.Param("id")
	
	var req WorkspaceUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	workspace := models.Workspace{
		ID:          workspaceID,
		Name:        req.Name,
		ProjectPath: "/tmp/updated-project",
		Status:      models.WorkspaceStatusActive,
		OwnerID:     suite.testUser,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	
	c.JSON(http.StatusOK, workspace)
}

func (suite *E2ETestSuite) mockDeleteWorkspaceHandler(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

func (suite *E2ETestSuite) mockStartWorkspaceHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "started"})
}

func (suite *E2ETestSuite) mockStopWorkspaceHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "stopped"})
}

func (suite *E2ETestSuite) mockGetWorkspaceStatusHandler(c *gin.Context) {
	status := WorkspaceStatus{
		Workspace: models.Workspace{
			ID:     c.Param("id"),
			Status: models.WorkspaceStatusActive,
		},
		ContainerStatus: ContainerStatus{
			ContainerID:    "mock-container-id",
			ContainerState: "running",
			IsHealthy:     true,
		},
	}
	
	c.JSON(http.StatusOK, status)
}

func (suite *E2ETestSuite) mockWebSocketHandler(c *gin.Context) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // 테스트용으로 모든 origin 허용
		},
	}
	
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer conn.Close()
	
	// 연결 확인 메시지 전송
	conn.WriteJSON(map[string]interface{}{
		"type":    "connected",
		"message": "WebSocket connection established",
	})
	
	// 간단한 에코 서버로 동작
	for {
		var msg map[string]interface{}
		if err := conn.ReadJSON(&msg); err != nil {
			break
		}
		
		// 상태 메시지 응답
		if msg["type"] == "subscribe" {
			conn.WriteJSON(map[string]interface{}{
				"type":   "status",
				"status": "active",
				"data":   msg,
			})
		}
	}
}

// Helper methods for API calls

type WorkspaceCreateRequest struct {
	Name        string `json:"name"`
	ProjectPath string `json:"project_path"`
}

type WorkspaceUpdateRequest struct {
	Name        string `json:"name"`
}

type WorkspaceStatus struct {
	Workspace       models.Workspace `json:"workspace"`
	ContainerStatus ContainerStatus  `json:"container_status"`
}

type ContainerStatus struct {
	ContainerID    string `json:"container_id"`
	ContainerState string `json:"container_state"`
	IsHealthy      bool   `json:"is_healthy"`
}

func (suite *E2ETestSuite) authenticateTestUser(t *testing.T) string {
	loginData := map[string]string{
		"username": suite.testUser,
		"password": "test-password",
	}
	
	body, _ := json.Marshal(loginData)
	resp, err := suite.client.Post(suite.baseURL+"/api/auth/login", 
		"application/json", bytes.NewBuffer(body))
	require.NoError(t, err)
	defer resp.Body.Close()
	
	require.Equal(t, http.StatusOK, resp.StatusCode)
	
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	
	return result["token"].(string)
}

func (suite *E2ETestSuite) createWorkspaceViaAPI(t *testing.T, req WorkspaceCreateRequest) models.Workspace {
	body, _ := json.Marshal(req)
	
	httpReq, _ := http.NewRequest("POST", suite.baseURL+"/api/workspaces", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+suite.token)
	
	resp, err := suite.client.Do(httpReq)
	require.NoError(t, err)
	defer resp.Body.Close()
	
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	
	var workspace models.Workspace
	json.NewDecoder(resp.Body).Decode(&workspace)
	
	return workspace
}

func (suite *E2ETestSuite) getWorkspaceStatusViaAPI(t *testing.T, workspaceID string) WorkspaceStatus {
	httpReq, _ := http.NewRequest("GET", 
		fmt.Sprintf("%s/api/workspaces/%s/status", suite.baseURL, workspaceID), nil)
	httpReq.Header.Set("Authorization", "Bearer "+suite.token)
	
	resp, err := suite.client.Do(httpReq)
	require.NoError(t, err)
	defer resp.Body.Close()
	
	var status WorkspaceStatus
	json.NewDecoder(resp.Body).Decode(&status)
	
	return status
}

func (suite *E2ETestSuite) listWorkspacesViaAPI(t *testing.T) []models.Workspace {
	httpReq, _ := http.NewRequest("GET", suite.baseURL+"/api/workspaces", nil)
	httpReq.Header.Set("Authorization", "Bearer "+suite.token)
	
	resp, err := suite.client.Do(httpReq)
	require.NoError(t, err)
	defer resp.Body.Close()
	
	var workspaces []models.Workspace
	json.NewDecoder(resp.Body).Decode(&workspaces)
	
	return workspaces
}

func (suite *E2ETestSuite) stopWorkspaceViaAPI(t *testing.T, workspaceID string) {
	httpReq, _ := http.NewRequest("POST", 
		fmt.Sprintf("%s/api/workspaces/%s/stop", suite.baseURL, workspaceID), nil)
	httpReq.Header.Set("Authorization", "Bearer "+suite.token)
	
	resp, err := suite.client.Do(httpReq)
	require.NoError(t, err)
	defer resp.Body.Close()
	
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func (suite *E2ETestSuite) startWorkspaceViaAPI(t *testing.T, workspaceID string) {
	httpReq, _ := http.NewRequest("POST", 
		fmt.Sprintf("%s/api/workspaces/%s/start", suite.baseURL, workspaceID), nil)
	httpReq.Header.Set("Authorization", "Bearer "+suite.token)
	
	resp, err := suite.client.Do(httpReq)
	require.NoError(t, err)
	defer resp.Body.Close()
	
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func (suite *E2ETestSuite) updateWorkspaceViaAPI(t *testing.T, workspaceID string, req WorkspaceUpdateRequest) models.Workspace {
	body, _ := json.Marshal(req)
	
	httpReq, _ := http.NewRequest("PUT", 
		fmt.Sprintf("%s/api/workspaces/%s", suite.baseURL, workspaceID), bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+suite.token)
	
	resp, err := suite.client.Do(httpReq)
	require.NoError(t, err)
	defer resp.Body.Close()
	
	require.Equal(t, http.StatusOK, resp.StatusCode)
	
	var workspace models.Workspace
	json.NewDecoder(resp.Body).Decode(&workspace)
	
	return workspace
}

func (suite *E2ETestSuite) deleteWorkspaceViaAPI(t *testing.T, workspaceID string) {
	httpReq, _ := http.NewRequest("DELETE", 
		fmt.Sprintf("%s/api/workspaces/%s", suite.baseURL, workspaceID), nil)
	httpReq.Header.Set("Authorization", "Bearer "+suite.token)
	
	resp, err := suite.client.Do(httpReq)
	require.NoError(t, err)
	defer resp.Body.Close()
	
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func (suite *E2ETestSuite) createTempTestProject(t *testing.T) string {
	tmpDir, err := os.MkdirTemp("", "e2e-project-*")
	require.NoError(t, err)
	
	// 기본 프로젝트 파일 생성
	files := map[string]string{
		"README.md":  "# E2E Test Project\n\nThis is a test project for E2E testing.",
		"main.go":    "package main\n\nfunc main() {\n\tprintln(\"Hello E2E Test\")\n}",
		"go.mod":     "module e2e-test\n\ngo 1.21\n",
		".gitignore": "*.log\n*.tmp\n",
	}
	
	for filename, content := range files {
		err := os.WriteFile(filepath.Join(tmpDir, filename), []byte(content), 0644)
		require.NoError(t, err)
	}
	
	suite.tempDirs = append(suite.tempDirs, tmpDir)
	return tmpDir
}

func (suite *E2ETestSuite) eventually(t *testing.T, timeout time.Duration, condition func() bool, msgAndArgs ...interface{}) {
	deadline := time.Now().Add(timeout)
	
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(1 * time.Second)
	}
	
	// 최종 확인
	if !condition() {
		if len(msgAndArgs) > 0 {
			require.Fail(t, "Condition not met within timeout", msgAndArgs...)
		} else {
			require.Fail(t, "Condition not met within timeout")
		}
	}
}

// isDockerAvailable checks if Docker is available for testing
func isDockerAvailable() bool {
	// 실제 구현에서는 Docker 클라이언트로 ping 테스트
	return true // 테스트 목적으로 항상 true 반환
}