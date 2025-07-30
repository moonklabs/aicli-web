package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aicli/aicli-web/internal/agent"
	"github.com/aicli/aicli-web/internal/api/controllers"
	"github.com/aicli/aicli-web/internal/docker"
	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/server"
	"github.com/aicli/aicli-web/internal/storage"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// APIIntegrationTestSuite API 통합 테스트 스위트
// 실제 HTTP API 엔드포인트들을 테스트합니다.
type APIIntegrationTestSuite struct {
	suite.Suite
	ctx               context.Context
	cancel            context.CancelFunc
	server           *httptest.Server
	client           *http.Client
	storage          storage.Storage
	agentService     *agent.Service
	testAgents       []string
	cleanupFunctions []func()
}

// SetupSuite 테스트 스위트 초기화
func (suite *APIIntegrationTestSuite) SetupSuite() {
	suite.ctx, suite.cancel = context.WithTimeout(context.Background(), 10*time.Minute)
	
	// 테스트 모드로 Gin 설정
	gin.SetMode(gin.TestMode)
	
	// 스토리지 초기화
	store, err := storage.New()
	require.NoError(suite.T(), err)
	suite.storage = store
	suite.addCleanupFunction(func() { store.Close() })

	// Docker 클라이언트 초기화
	dockerClient, err := docker.NewClient()
	require.NoError(suite.T(), err)

	// Agent 서비스 초기화
	suite.agentService = agent.NewService(
		suite.storage.Agent(),
		dockerClient,
		nil, // Git manager는 이 테스트에서 사용하지 않음
	)

	// 테스트 서버 설정
	suite.setupTestServer()
	
	// HTTP 클라이언트 초기화
	suite.client = &http.Client{
		Timeout: 30 * time.Second,
	}

	suite.T().Logf("API 통합 테스트 환경 초기화 완료 - 서버: %s", suite.server.URL)
}

// TearDownSuite 테스트 스위트 정리
func (suite *APIIntegrationTestSuite) TearDownSuite() {
	// 모든 테스트 에이전트 정리
	suite.cleanupTestAgents()
	
	// 서버 종료
	if suite.server != nil {
		suite.server.Close()
	}
	
	// 정리 함수들 실행
	for i := len(suite.cleanupFunctions) - 1; i >= 0; i-- {
		suite.cleanupFunctions[i]()
	}
	
	if suite.cancel != nil {
		suite.cancel()
	}
}

// SetupTest 각 테스트 초기화
func (suite *APIIntegrationTestSuite) SetupTest() {
	suite.testAgents = make([]string, 0)
}

// TearDownTest 각 테스트 정리
func (suite *APIIntegrationTestSuite) TearDownTest() {
	suite.cleanupTestAgents()
}

// TestHealthEndpoint 헬스 체크 엔드포인트 테스트
func (suite *APIIntegrationTestSuite) TestHealthEndpoint() {
	suite.T().Log("🔄 헬스 체크 엔드포인트 테스트 시작")

	// GET /health
	resp, err := suite.client.Get(suite.server.URL + "/health")
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
	
	var healthResponse map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&healthResponse)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), "healthy", healthResponse["status"])
	assert.NotEmpty(suite.T(), healthResponse["timestamp"])

	suite.T().Log("   ✅ 헬스 체크 엔드포인트 테스트 성공")
}

// TestSystemInfoEndpoint 시스템 정보 엔드포인트 테스트
func (suite *APIIntegrationTestSuite) TestSystemInfoEndpoint() {
	suite.T().Log("🔄 시스템 정보 엔드포인트 테스트 시작")

	// GET /api/v1/system/info
	resp, err := suite.client.Get(suite.server.URL + "/api/v1/system/info")
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
	
	var systemInfo map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&systemInfo)
	require.NoError(suite.T(), err)

	assert.NotEmpty(suite.T(), systemInfo["version"])
	assert.NotEmpty(suite.T(), systemInfo["build_time"])
	assert.NotEmpty(suite.T(), systemInfo["go_version"])

	suite.T().Log("   ✅ 시스템 정보 엔드포인트 테스트 성공")
}

// TestAgentCRUDEndpoints 에이전트 CRUD 엔드포인트 테스트
func (suite *APIIntegrationTestSuite) TestAgentCRUDEndpoints() {
	suite.T().Log("🔄 에이전트 CRUD 엔드포인트 테스트 시작")

	// 1. 에이전트 생성 (POST /api/v1/agents)
	suite.T().Log("   📝 에이전트 생성 테스트")
	createRequest := map[string]interface{}{
		"name":        fmt.Sprintf("api-test-agent-%d", time.Now().Unix()),
		"project_id":  "test-project",
		"agent_type":  "standard",
		"description": "API integration test agent",
		"config": map[string]interface{}{
			"resources": map[string]interface{}{
				"cpu":    "1.0",
				"memory": "1Gi",
			},
		},
	}

	createBody, err := json.Marshal(createRequest)
	require.NoError(suite.T(), err)

	resp, err := suite.client.Post(
		suite.server.URL+"/api/v1/agents",
		"application/json",
		bytes.NewBuffer(createBody),
	)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)

	var createResponse map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&createResponse)
	require.NoError(suite.T(), err)

	agentID := createResponse["id"].(string)
	assert.NotEmpty(suite.T(), agentID)
	assert.Equal(suite.T(), createRequest["name"], createResponse["name"])
	suite.testAgents = append(suite.testAgents, agentID)

	suite.T().Logf("   ✅ 에이전트 생성 성공 - ID: %s", agentID)

	// 2. 에이전트 조회 (GET /api/v1/agents/:id)
	suite.T().Log("   📖 에이전트 조회 테스트")
	resp, err = suite.client.Get(suite.server.URL + "/api/v1/agents/" + agentID)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var getResponse map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&getResponse)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), agentID, getResponse["id"])
	assert.Equal(suite.T(), createRequest["name"], getResponse["name"])

	suite.T().Log("   ✅ 에이전트 조회 성공")

	// 3. 에이전트 목록 조회 (GET /api/v1/agents)
	suite.T().Log("   📋 에이전트 목록 조회 테스트")
	resp, err = suite.client.Get(suite.server.URL + "/api/v1/agents")
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var listResponse map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&listResponse)
	require.NoError(suite.T(), err)

	agents := listResponse["agents"].([]interface{})
	assert.GreaterOrEqual(suite.T(), len(agents), 1)

	// 생성한 에이전트가 목록에 있는지 확인
	found := false
	for _, a := range agents {
		agent := a.(map[string]interface{})
		if agent["id"] == agentID {
			found = true
			break
		}
	}
	assert.True(suite.T(), found, "생성한 에이전트가 목록에 있어야 함")

	suite.T().Log("   ✅ 에이전트 목록 조회 성공")

	// 4. 에이전트 수정 (PUT /api/v1/agents/:id)
	suite.T().Log("   ✏️  에이전트 수정 테스트")
	updateRequest := map[string]interface{}{
		"description": "Updated API integration test agent",
	}

	updateBody, err := json.Marshal(updateRequest)
	require.NoError(suite.T(), err)

	req, err := http.NewRequest("PUT", suite.server.URL+"/api/v1/agents/"+agentID, bytes.NewBuffer(updateBody))
	require.NoError(suite.T(), err)
	req.Header.Set("Content-Type", "application/json")

	resp, err = suite.client.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var updateResponse map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&updateResponse)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), updateRequest["description"], updateResponse["description"])

	suite.T().Log("   ✅ 에이전트 수정 성공")

	// 5. 에이전트 삭제 (DELETE /api/v1/agents/:id)
	suite.T().Log("   🗑️  에이전트 삭제 테스트")
	req, err = http.NewRequest("DELETE", suite.server.URL+"/api/v1/agents/"+agentID, nil)
	require.NoError(suite.T(), err)

	resp, err = suite.client.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusNoContent, resp.StatusCode)

	// 삭제 확인 - 조회 시 404 반환
	resp, err = suite.client.Get(suite.server.URL + "/api/v1/agents/" + agentID)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusNotFound, resp.StatusCode)

	suite.T().Log("   ✅ 에이전트 삭제 성공")
	
	// 테스트 에이전트 목록에서 제거 (이미 삭제됨)
	suite.testAgents = suite.testAgents[:0]

	suite.T().Log("🎉 에이전트 CRUD 엔드포인트 테스트 성공")
}

// TestAgentControlEndpoints 에이전트 제어 엔드포인트 테스트
func (suite *APIIntegrationTestSuite) TestAgentControlEndpoints() {
	suite.T().Log("🔄 에이전트 제어 엔드포인트 테스트 시작")

	// 테스트용 에이전트 생성
	agentID := suite.createTestAgent("control-test-agent")

	// 1. 에이전트 시작 (POST /api/v1/agents/:id/start)
	suite.T().Log("   🚀 에이전트 시작 테스트")
	resp, err := suite.client.Post(suite.server.URL+"/api/v1/agents/"+agentID+"/start", "application/json", nil)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	// 시작 요청이 접수되었는지 확인 (실제 컨테이너 시작은 시간이 걸릴 수 있음)
	assert.True(suite.T(), resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusAccepted)

	suite.T().Log("   ✅ 에이전트 시작 요청 성공")

	// 2. 에이전트 상태 확인 (GET /api/v1/agents/:id/status)
	suite.T().Log("   📊 에이전트 상태 확인 테스트")
	resp, err = suite.client.Get(suite.server.URL + "/api/v1/agents/" + agentID + "/status")
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var statusResponse map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&statusResponse)
	require.NoError(suite.T(), err)

	assert.NotEmpty(suite.T(), statusResponse["status"])
	assert.NotEmpty(suite.T(), statusResponse["agent_id"])

	suite.T().Logf("   📋 현재 상태: %s", statusResponse["status"])

	// 3. 에이전트 중지 (POST /api/v1/agents/:id/stop)
	suite.T().Log("   ⏹️  에이전트 중지 테스트")
	resp, err = suite.client.Post(suite.server.URL+"/api/v1/agents/"+agentID+"/stop", "application/json", nil)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.True(suite.T(), resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusAccepted)

	suite.T().Log("   ✅ 에이전트 중지 요청 성공")

	// 4. 에이전트 재시작 (POST /api/v1/agents/:id/restart)
	suite.T().Log("   🔄 에이전트 재시작 테스트")
	resp, err = suite.client.Post(suite.server.URL+"/api/v1/agents/"+agentID+"/restart", "application/json", nil)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.True(suite.T(), resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusAccepted)

	suite.T().Log("   ✅ 에이전트 재시작 요청 성공")

	suite.T().Log("🎉 에이전트 제어 엔드포인트 테스트 성공")
}

// TestAgentMetricsEndpoint 에이전트 메트릭 엔드포인트 테스트
func (suite *APIIntegrationTestSuite) TestAgentMetricsEndpoint() {
	suite.T().Log("🔄 에이전트 메트릭 엔드포인트 테스트 시작")

	// 테스트용 에이전트 생성
	agentID := suite.createTestAgent("metrics-test-agent")

	// 에이전트 메트릭 조회 (GET /api/v1/agents/:id/metrics)
	suite.T().Log("   📈 에이전트 메트릭 조회 테스트")
	resp, err := suite.client.Get(suite.server.URL + "/api/v1/agents/" + agentID + "/metrics")
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var metricsResponse map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&metricsResponse)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), agentID, metricsResponse["agent_id"])
	assert.NotNil(suite.T(), metricsResponse["uptime_seconds"])
	assert.NotNil(suite.T(), metricsResponse["created_at"])

	suite.T().Log("   ✅ 에이전트 메트릭 조회 성공")
	suite.T().Log("🎉 에이전트 메트릭 엔드포인트 테스트 성공")
}

// TestErrorHandling API 에러 처리 테스트
func (suite *APIIntegrationTestSuite) TestErrorHandling() {
	suite.T().Log("🔄 API 에러 처리 테스트 시작")

	// 1. 존재하지 않는 에이전트 조회 (404)
	suite.T().Log("   🔍 존재하지 않는 에이전트 조회 테스트")
	resp, err := suite.client.Get(suite.server.URL + "/api/v1/agents/nonexistent-id")
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusNotFound, resp.StatusCode)

	var errorResponse map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&errorResponse)
	require.NoError(suite.T(), err)

	assert.Contains(suite.T(), strings.ToLower(errorResponse["error"].(string)), "not found")

	// 2. 잘못된 JSON 형식 (400)
	suite.T().Log("   📝 잘못된 JSON 형식 테스트")
	invalidJSON := bytes.NewBufferString(`{"invalid": json}`)
	resp, err = suite.client.Post(suite.server.URL+"/api/v1/agents", "application/json", invalidJSON)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusBadRequest, resp.StatusCode)

	// 3. 빈 요청 필드 (422)
	suite.T().Log("   🚫 필수 필드 누락 테스트")
	emptyRequest := map[string]interface{}{
		"name": "", // 빈 이름
	}
	emptyBody, err := json.Marshal(emptyRequest)
	require.NoError(suite.T(), err)

	resp, err = suite.client.Post(
		suite.server.URL+"/api/v1/agents",
		"application/json",
		bytes.NewBuffer(emptyBody),
	)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.True(suite.T(), resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusUnprocessableEntity)

	suite.T().Log("🎉 API 에러 처리 테스트 성공")
}

// TestConcurrentAPIRequests 동시 API 요청 테스트
func (suite *APIIntegrationTestSuite) TestConcurrentAPIRequests() {
	suite.T().Log("🔄 동시 API 요청 테스트 시작")

	requestCount := 10
	results := make(chan error, requestCount)

	// 동시에 여러 에이전트 생성 요청
	suite.T().Logf("   📝 %d개 동시 에이전트 생성 요청", requestCount)
	for i := 0; i < requestCount; i++ {
		go func(index int) {
			agentID := suite.createTestAgentAsync(fmt.Sprintf("concurrent-test-%d", index))
			if agentID != "" {
				suite.testAgents = append(suite.testAgents, agentID)
				results <- nil
			} else {
				results <- fmt.Errorf("에이전트 생성 실패: %d", index)
			}
		}(i)
	}

	// 결과 수집
	successCount := 0
	for i := 0; i < requestCount; i++ {
		err := <-results
		if err == nil {
			successCount++
		}
	}

	suite.T().Logf("   📊 성공률: %d/%d", successCount, requestCount)
	assert.Greater(suite.T(), successCount, requestCount/2, "최소 50% 이상의 요청이 성공해야 함")

	suite.T().Log("🎉 동시 API 요청 테스트 성공")
}

// 헬퍼 메서드들

// setupTestServer 테스트 서버 설정
func (suite *APIIntegrationTestSuite) setupTestServer() {
	// Router 설정
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	// 헬스 체크 엔드포인트
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"timestamp": time.Now().Format(time.RFC3339),
		})
	})

	// 시스템 정보 엔드포인트
	router.GET("/api/v1/system/info", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"version":    "test-version",
			"build_time": "test-build-time",
			"go_version": "test-go-version",
		})
	})

	// Agent API 컨트롤러 설정
	agentController := controllers.NewAgentController(suite.agentService)
	
	v1 := router.Group("/api/v1")
	{
		agents := v1.Group("/agents")
		{
			agents.GET("", agentController.ListAgents)
			agents.POST("", agentController.CreateAgent)
			agents.GET("/:id", agentController.GetAgent)
			agents.PUT("/:id", agentController.UpdateAgent)
			agents.DELETE("/:id", agentController.DeleteAgent)
			agents.POST("/:id/start", agentController.StartAgent)
			agents.POST("/:id/stop", agentController.StopAgent)
			agents.POST("/:id/restart", agentController.RestartAgent)
			agents.GET("/:id/status", agentController.GetAgentStatus)
			agents.GET("/:id/metrics", agentController.GetAgentMetrics)
		}
	}

	// 테스트 서버 시작
	suite.server = httptest.NewServer(router)
}

// createTestAgent 테스트용 에이전트 생성
func (suite *APIIntegrationTestSuite) createTestAgent(name string) string {
	createRequest := map[string]interface{}{
		"name":        name,
		"project_id":  "test-project",
		"agent_type":  "standard",
		"description": "Test agent for API integration",
		"config": map[string]interface{}{
			"resources": map[string]interface{}{
				"cpu":    "0.5",
				"memory": "512Mi",
			},
		},
	}

	body, err := json.Marshal(createRequest)
	require.NoError(suite.T(), err)

	resp, err := suite.client.Post(
		suite.server.URL+"/api/v1/agents",
		"application/json",
		bytes.NewBuffer(body),
	)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	require.Equal(suite.T(), http.StatusCreated, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(suite.T(), err)

	agentID := response["id"].(string)
	suite.testAgents = append(suite.testAgents, agentID)
	return agentID
}

// createTestAgentAsync 비동기 테스트용 에이전트 생성
func (suite *APIIntegrationTestSuite) createTestAgentAsync(name string) string {
	createRequest := map[string]interface{}{
		"name":        name,
		"project_id":  "test-project",
		"agent_type":  "standard",
		"description": "Async test agent",
		"config": map[string]interface{}{
			"resources": map[string]interface{}{
				"cpu":    "0.2",
				"memory": "256Mi",
			},
		},
	}

	body, err := json.Marshal(createRequest)
	if err != nil {
		return ""
	}

	resp, err := suite.client.Post(
		suite.server.URL+"/api/v1/agents",
		"application/json",
		bytes.NewBuffer(body),
	)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return ""
	}

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return ""
	}

	return response["id"].(string)
}

// cleanupTestAgents 테스트 에이전트들 정리
func (suite *APIIntegrationTestSuite) cleanupTestAgents() {
	for _, agentID := range suite.testAgents {
		// 에이전트 삭제 요청
		req, err := http.NewRequest("DELETE", suite.server.URL+"/api/v1/agents/"+agentID, nil)
		if err != nil {
			continue
		}

		resp, err := suite.client.Do(req)
		if err != nil {
			continue
		}
		resp.Body.Close()
	}
	suite.testAgents = suite.testAgents[:0]
}

// addCleanupFunction 정리 함수 추가
func (suite *APIIntegrationTestSuite) addCleanupFunction(fn func()) {
	suite.cleanupFunctions = append(suite.cleanupFunctions, fn)
}

// TestAPIIntegrationSuite API 통합 테스트 실행
func TestAPIIntegrationSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("통합 테스트는 short 모드에서 제외됩니다")
	}

	suite.Run(t, new(APIIntegrationTestSuite))
}