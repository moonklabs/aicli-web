package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/aicli/aicli-web/internal/agent"
	"github.com/aicli/aicli-web/internal/auth"
	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/utils"
)

// MockAgentService는 테스트용 에이전트 서비스 모의 객체입니다
type MockAgentService struct {
	mock.Mock
}

func (m *MockAgentService) CreateAgent(ctx context.Context, req agent.CreateAgentRequest) (*models.Agent, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Agent), args.Error(1)
}

func (m *MockAgentService) GetAgent(ctx context.Context, id string) (*models.Agent, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Agent), args.Error(1)
}

func (m *MockAgentService) GetAgentByProjectID(ctx context.Context, projectID string) ([]*models.Agent, error) {
	args := m.Called(ctx, projectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Agent), args.Error(1)
}

func (m *MockAgentService) UpdateAgent(ctx context.Context, id string, req agent.UpdateAgentRequest) (*models.Agent, error) {
	args := m.Called(ctx, id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Agent), args.Error(1)
}

func (m *MockAgentService) DeleteAgent(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAgentService) StartAgent(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAgentService) StopAgent(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAgentService) RestartAgent(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAgentService) GetAgentStatus(ctx context.Context, id string) (agent.AgentStatusInfo, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(agent.AgentStatusInfo), args.Error(1)
}

func (m *MockAgentService) StartMultipleAgents(ctx context.Context, ids []string) ([]agent.AgentOperationResult, error) {
	args := m.Called(ctx, ids)
	return args.Get(0).([]agent.AgentOperationResult), args.Error(1)
}

func (m *MockAgentService) StopMultipleAgents(ctx context.Context, ids []string) ([]agent.AgentOperationResult, error) {
	args := m.Called(ctx, ids)
	return args.Get(0).([]agent.AgentOperationResult), args.Error(1)
}

func (m *MockAgentService) ListActiveAgents(ctx context.Context) ([]*models.Agent, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Agent), args.Error(1)
}

func (m *MockAgentService) GetHealthStatus(ctx context.Context, id string) (agent.HealthStatus, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(agent.HealthStatus), args.Error(1)
}

func (m *MockAgentService) GetAgentMetrics(ctx context.Context, id string) (agent.AgentMetrics, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(agent.AgentMetrics), args.Error(1)
}

func (m *MockAgentService) CleanupStaleAgents(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

func (m *MockAgentService) PerformMaintenance(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// setupAgentTest는 에이전트 API 테스트를 위한 설정을 수행합니다
func setupAgentTest() (*gin.Engine, *AgentController, *MockAgentService, *auth.JWTManager) {
	gin.SetMode(gin.TestMode)
	utils.RegisterCustomValidators()

	// JWT 매니저 생성
	jwtManager := auth.NewJWTManager("test-secret", 3600, 86400)

	// Mock 에이전트 서비스 생성
	mockAgentService := &MockAgentService{}

	// 컨트롤러 생성
	controller := NewAgentController(mockAgentService)

	// 라우터 설정
	router := gin.New()

	return router, controller, mockAgentService, jwtManager
}

// createTestAgent는 테스트용 에이전트를 생성합니다
func createTestAgent() *models.Agent {
	return &models.Agent{
		ID:        uuid.New().String(),
		ProjectID: uuid.New().String(),
		Name:      "test-agent",
		Type:      models.AgentTypeClaude,
		Status:    models.AgentStatusCreated,
		Config: models.AgentConfig{
			Model:       "claude-3-sonnet",
			MaxTokens:   4000,
			Temperature: 0.7,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func TestListAgents(t *testing.T) {
	_, controller, mockService, _ := setupAgentTest()

	// 테스트 데이터 준비
	testAgents := []*models.Agent{
		createTestAgent(),
		createTestAgent(),
	}

	tests := []struct {
		name           string
		projectID      string
		setupMock      func()
		expectedStatus int
		expectedCount  int
	}{
		{
			name:      "모든 에이전트 목록 조회 성공",
			projectID: "",
			setupMock: func() {
				mockService.On("ListActiveAgents", mock.Anything).Return(testAgents, nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectedCount:  2,
		},
		{
			name:      "프로젝트별 에이전트 목록 조회 성공",
			projectID: testAgents[0].ProjectID,
			setupMock: func() {
				projectAgents := []*models.Agent{testAgents[0]}
				mockService.On("GetAgentByProjectID", mock.Anything, testAgents[0].ProjectID).Return(projectAgents, nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
		{
			name:      "서비스 오류",
			projectID: "",
			setupMock: func() {
				mockService.On("ListActiveAgents", mock.Anything).Return(nil, fmt.Errorf("service error")).Once()
			},
			expectedStatus: http.StatusInternalServerError,
			expectedCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock 설정
			tt.setupMock()

			// 새로운 라우터 생성 (라우트 충돌 방지)
			testRouter := gin.New()
			testRouter.GET("/api/v1/agents", controller.ListAgents)

			// 요청 준비
			url := "/api/v1/agents"
			if tt.projectID != "" {
				url += "?project_id=" + tt.projectID
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			// 요청 실행
			testRouter.ServeHTTP(w, req)

			// 응답 검증
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, float64(tt.expectedCount), response["count"])
			}

			// Mock 호출 검증
			mockService.AssertExpectations(t)
		})
	}
}

func TestCreateAgent(t *testing.T) {
	// 테스트 데이터 준비
	testAgent := createTestAgent()

	tests := []struct {
		name           string
		requestBody    interface{}
		setupMock      func(*MockAgentService)
		expectedStatus int
	}{
		{
			name: "에이전트 생성 성공",
			requestBody: agent.CreateAgentRequest{
				ProjectID: testAgent.ProjectID,
				Name:      testAgent.Name,
				Type:      testAgent.Type,
				Config:    testAgent.Config,
			},
			setupMock: func(mockService *MockAgentService) {
				mockService.On("CreateAgent", mock.Anything, mock.AnythingOfType("agent.CreateAgentRequest")).Return(testAgent, nil).Once()
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:        "잘못된 요청 데이터",
			requestBody: "invalid json",
			setupMock: func(mockService *MockAgentService) {
				// JSON 파싱 실패 시 서비스 호출되지 않음
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "서비스 오류",
			requestBody: agent.CreateAgentRequest{
				ProjectID: testAgent.ProjectID,
				Name:      testAgent.Name,
				Type:      testAgent.Type,
				Config:    testAgent.Config,
			},
			setupMock: func(mockService *MockAgentService) {
				mockService.On("CreateAgent", mock.Anything, mock.AnythingOfType("agent.CreateAgentRequest")).Return(nil, fmt.Errorf("service error")).Once()
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 각 테스트마다 새로운 Mock 서비스 생성
			_, controller, mockService, _ := setupAgentTest()

			// Mock 설정
			tt.setupMock(mockService)

			// 새로운 라우터 생성
			testRouter := gin.New()
			testRouter.POST("/api/v1/agents", controller.CreateAgent)

			// 요청 데이터 준비
			jsonData, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewBuffer(jsonData))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// 요청 실행
			testRouter.ServeHTTP(w, req)

			// 응답 검증
			if w.Code != tt.expectedStatus {
				t.Logf("Expected status %d, got %d. Response body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusCreated {
				var response models.Agent
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, testAgent.ID, response.ID)
				assert.Equal(t, testAgent.Name, response.Name)
			}

			// Mock 호출 검증
			mockService.AssertExpectations(t)
		})
	}
}

func TestGetAgent(t *testing.T) {
	_, controller, mockService, _ := setupAgentTest()

	// 테스트 데이터 준비
	testAgent := createTestAgent()

	tests := []struct {
		name           string
		agentID        string
		setupMock      func()
		expectedStatus int
	}{
		{
			name:    "에이전트 조회 성공",
			agentID: testAgent.ID,
			setupMock: func() {
				mockService.On("GetAgent", mock.Anything, testAgent.ID).Return(testAgent, nil).Once()
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:    "에이전트 없음",
			agentID: "nonexistent-id",
			setupMock: func() {
				mockService.On("GetAgent", mock.Anything, "nonexistent-id").Return(nil, fmt.Errorf("agent not found")).Once()
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock 설정
			tt.setupMock()

			// 새로운 라우터 생성
			testRouter := gin.New()
			testRouter.GET("/api/v1/agents/:id", controller.GetAgent)

			// 요청 준비
			req := httptest.NewRequest(http.MethodGet, "/api/v1/agents/"+tt.agentID, nil)
			w := httptest.NewRecorder()

			// 요청 실행
			testRouter.ServeHTTP(w, req)

			// 응답 검증
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var response models.Agent
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, testAgent.ID, response.ID)
			}

			// Mock 호출 검증
			mockService.AssertExpectations(t)
		})
	}
}

func TestStartAgent(t *testing.T) {
	_, controller, mockService, _ := setupAgentTest()

	// 테스트 데이터 준비
	testAgent := createTestAgent()

	tests := []struct {
		name           string
		agentID        string
		setupMock      func()
		expectedStatus int
	}{
		{
			name:    "에이전트 시작 성공",
			agentID: testAgent.ID,
			setupMock: func() {
				mockService.On("StartAgent", mock.Anything, testAgent.ID).Return(nil).Once()
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:    "에이전트 시작 실패",
			agentID: testAgent.ID,
			setupMock: func() {
				mockService.On("StartAgent", mock.Anything, testAgent.ID).Return(fmt.Errorf("start failed")).Once()
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock 설정
			tt.setupMock()

			// 새로운 라우터 생성
			testRouter := gin.New()
			testRouter.POST("/api/v1/agents/:id/start", controller.StartAgent)

			// 요청 준비
			req := httptest.NewRequest(http.MethodPost, "/api/v1/agents/"+tt.agentID+"/start", nil)
			w := httptest.NewRecorder()

			// 요청 실행
			testRouter.ServeHTTP(w, req)

			// 응답 검증
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Mock 호출 검증
			mockService.AssertExpectations(t)
		})
	}
}

func TestStopAgent(t *testing.T) {
	_, controller, mockService, _ := setupAgentTest()

	// 테스트 데이터 준비
	testAgent := createTestAgent()

	tests := []struct {
		name           string
		agentID        string
		setupMock      func()
		expectedStatus int
	}{
		{
			name:    "에이전트 중지 성공",
			agentID: testAgent.ID,
			setupMock: func() {
				mockService.On("StopAgent", mock.Anything, testAgent.ID).Return(nil).Once()
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:    "에이전트 중지 실패",
			agentID: testAgent.ID,
			setupMock: func() {
				mockService.On("StopAgent", mock.Anything, testAgent.ID).Return(fmt.Errorf("stop failed")).Once()
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock 설정
			tt.setupMock()

			// 새로운 라우터 생성
			testRouter := gin.New()
			testRouter.POST("/api/v1/agents/:id/stop", controller.StopAgent)

			// 요청 준비
			req := httptest.NewRequest(http.MethodPost, "/api/v1/agents/"+tt.agentID+"/stop", nil)
			w := httptest.NewRecorder()

			// 요청 실행
			testRouter.ServeHTTP(w, req)

			// 응답 검증
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Mock 호출 검증
			mockService.AssertExpectations(t)
		})
	}
}

func TestDeleteAgent(t *testing.T) {
	_, controller, mockService, _ := setupAgentTest()

	// 테스트 데이터 준비
	testAgent := createTestAgent()

	tests := []struct {
		name           string
		agentID        string
		setupMock      func()
		expectedStatus int
	}{
		{
			name:    "에이전트 삭제 성공",
			agentID: testAgent.ID,
			setupMock: func() {
				mockService.On("DeleteAgent", mock.Anything, testAgent.ID).Return(nil).Once()
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name:    "에이전트 삭제 실패",
			agentID: testAgent.ID,
			setupMock: func() {
				mockService.On("DeleteAgent", mock.Anything, testAgent.ID).Return(fmt.Errorf("delete failed")).Once()
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock 설정
			tt.setupMock()

			// 새로운 라우터 생성
			testRouter := gin.New()
			testRouter.DELETE("/api/v1/agents/:id", controller.DeleteAgent)

			// 요청 준비
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/agents/"+tt.agentID, nil)
			w := httptest.NewRecorder()

			// 요청 실행
			testRouter.ServeHTTP(w, req)

			// 응답 검증
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Mock 호출 검증
			mockService.AssertExpectations(t)
		})
	}
}
