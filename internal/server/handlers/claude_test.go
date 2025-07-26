package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/aicli/aicli-web/internal/claude"
	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/websocket"
)

// MockClaudeWrapper는 테스트용 Claude 래퍼 mock입니다.
type MockClaudeWrapper struct {
	mock.Mock
}

func (m *MockClaudeWrapper) CreateSession(config *claude.SessionConfig) (*claude.Session, error) {
	args := m.Called(config)
	return args.Get(0).(*claude.Session), args.Error(1)
}

func (m *MockClaudeWrapper) GetSession(sessionID string) (*claude.Session, error) {
	args := m.Called(sessionID)
	return args.Get(0).(*claude.Session), args.Error(1)
}

func (m *MockClaudeWrapper) CloseSession(sessionID string) error {
	args := m.Called(sessionID)
	return args.Error(0)
}

func (m *MockClaudeWrapper) Execute(sessionID, prompt string) (interface{}, error) {
	args := m.Called(sessionID, prompt)
	return args.Get(0), args.Error(1)
}

func (m *MockClaudeWrapper) ListSessions(filter claude.SessionFilter) ([]*claude.Session, error) {
	args := m.Called(filter)
	return args.Get(0).([]*claude.Session), args.Error(1)
}

// MockSessionRepository는 테스트용 세션 저장소 mock입니다.
type MockSessionRepository struct {
	mock.Mock
}

func (m *MockSessionRepository) Create(ctx context.Context, session *models.Session) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *MockSessionRepository) GetByID(ctx context.Context, id string) (*models.Session, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.Session), args.Error(1)
}

func (m *MockSessionRepository) Update(ctx context.Context, id string, update *models.SessionUpdate) (*models.Session, error) {
	args := m.Called(ctx, id, update)
	return args.Get(0).(*models.Session), args.Error(1)
}

func (m *MockSessionRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockSessionRepository) List(ctx context.Context, filter *models.SessionFilter, paging *models.PaginationRequest) (*models.PaginationResponse, error) {
	args := m.Called(ctx, filter, paging)
	return args.Get(0).(*models.PaginationResponse), args.Error(1)
}

func TestClaudeHandler_Execute(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		request        ExecuteRequest
		expectedStatus int
		setupMocks     func(*MockClaudeWrapper, *MockSessionRepository)
	}{
		{
			name: "성공적인 실행 요청",
			request: ExecuteRequest{
				WorkspaceID: "workspace-1",
				Prompt:      "Hello, Claude!",
				Stream:      true,
			},
			expectedStatus: http.StatusAccepted,
			setupMocks: func(wrapper *MockClaudeWrapper, repo *MockSessionRepository) {
				// 기존 세션이 없는 경우
				repo.On("FindByWorkspace", "workspace-1").Return([]*claude.Session{}, nil)
				
				// 새 세션 생성
				session := &claude.Session{
					ID:          "session-1",
					WorkspaceID: "workspace-1",
					UserID:      "user-1",
					State:       claude.SessionState{Status: "idle"},
					Created:     time.Now(),
					LastActive:  time.Now(),
				}
				wrapper.On("CreateSession", mock.AnythingOfType("*claude.SessionConfig")).Return(session, nil)
				
				// Execute는 비동기로 실행되므로 mock 설정하지 않음
			},
		},
		{
			name: "잘못된 요청 - WorkspaceID 없음",
			request: ExecuteRequest{
				Prompt: "Hello, Claude!",
			},
			expectedStatus: http.StatusBadRequest,
			setupMocks:     func(wrapper *MockClaudeWrapper, repo *MockSessionRepository) {},
		},
		{
			name: "잘못된 요청 - Prompt 없음",
			request: ExecuteRequest{
				WorkspaceID: "workspace-1",
			},
			expectedStatus: http.StatusBadRequest,
			setupMocks:     func(wrapper *MockClaudeWrapper, repo *MockSessionRepository) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock 객체 생성
			mockWrapper := new(MockClaudeWrapper)
			mockRepo := new(MockSessionRepository)
			mockHub := websocket.NewHub(nil)

			// Mock 설정
			tt.setupMocks(mockWrapper, mockRepo)

			// 핸들러 생성
			handler := NewClaudeHandler(mockWrapper, mockRepo, mockHub)

			// 요청 데이터 준비
			body, _ := json.Marshal(tt.request)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/claude/execute", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			// 응답 레코더 생성
			w := httptest.NewRecorder()

			// Gin 컨텍스트 생성
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			// 핸들러 실행
			handler.Execute(c)

			// 결과 검증
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusAccepted {
				var response ExecuteResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.NotEmpty(t, response.ExecutionID)
				assert.NotEmpty(t, response.SessionID)
				assert.Equal(t, "started", response.Status)
				if tt.request.Stream {
					assert.NotEmpty(t, response.WebSocketURL)
				}
			}

			// Mock 검증
			mockWrapper.AssertExpectations(t)
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestClaudeHandler_ListSessions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		workspaceID    string
		expectedStatus int
		setupMocks     func(*MockClaudeWrapper, *MockSessionRepository)
	}{
		{
			name:           "성공적인 세션 목록 조회",
			workspaceID:    "workspace-1",
			expectedStatus: http.StatusOK,
			setupMocks: func(wrapper *MockClaudeWrapper, repo *MockSessionRepository) {
				sessions := []*claude.Session{
					{
						ID:          "session-1",
						WorkspaceID: "workspace-1",
						UserID:      "user-1",
						State:       claude.SessionState{Status: "idle"},
						Created:     time.Now(),
						LastActive:  time.Now(),
					},
					{
						ID:          "session-2",
						WorkspaceID: "workspace-1",
						UserID:      "user-1",
						State:       claude.SessionState{Status: "running"},
						Created:     time.Now(),
						LastActive:  time.Now(),
					},
				}
				repo.On("FindByWorkspace", "workspace-1").Return(sessions, nil)
			},
		},
		{
			name:           "WorkspaceID 없음",
			workspaceID:    "",
			expectedStatus: http.StatusBadRequest,
			setupMocks:     func(wrapper *MockClaudeWrapper, repo *MockSessionRepository) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock 객체 생성
			mockWrapper := new(MockClaudeWrapper)
			mockRepo := new(MockSessionRepository)
			mockHub := websocket.NewHub(nil)

			// Mock 설정
			tt.setupMocks(mockWrapper, mockRepo)

			// 핸들러 생성
			handler := NewClaudeHandler(mockWrapper, mockRepo, mockHub)

			// 요청 준비
			req := httptest.NewRequest(http.MethodGet, "/api/v1/claude/sessions", nil)
			if tt.workspaceID != "" {
				q := req.URL.Query()
				q.Add("workspace_id", tt.workspaceID)
				req.URL.RawQuery = q.Encode()
			}

			// 응답 레코더 생성
			w := httptest.NewRecorder()

			// Gin 컨텍스트 생성
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			// 핸들러 실행
			handler.ListSessions(c)

			// 결과 검증
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, response, "sessions")
				assert.Contains(t, response, "total")
			}

			// Mock 검증
			mockWrapper.AssertExpectations(t)
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestClaudeHandler_GetSession(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		sessionID      string
		expectedStatus int
		setupMocks     func(*MockClaudeWrapper, *MockSessionRepository)
	}{
		{
			name:           "성공적인 세션 조회",
			sessionID:      "session-1",
			expectedStatus: http.StatusOK,
			setupMocks: func(wrapper *MockClaudeWrapper, repo *MockSessionRepository) {
				session := &claude.Session{
					ID:          "session-1",
					WorkspaceID: "workspace-1",
					UserID:      "user-1",
					State:       claude.SessionState{Status: "idle"},
					Created:     time.Now(),
					LastActive:  time.Now(),
				}
				wrapper.On("GetSession", "session-1").Return(session, nil)
			},
		},
		{
			name:           "세션이 존재하지 않음",
			sessionID:      "nonexistent",
			expectedStatus: http.StatusNotFound,
			setupMocks: func(wrapper *MockClaudeWrapper, repo *MockSessionRepository) {
				wrapper.On("GetSession", "nonexistent").Return((*claude.Session)(nil), &claude.ClaudeError{
					Code:    "SESSION_NOT_FOUND",
					Message: "Session not found",
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock 객체 생성
			mockWrapper := new(MockClaudeWrapper)
			mockRepo := new(MockSessionRepository)
			mockHub := websocket.NewHub(nil)

			// Mock 설정
			tt.setupMocks(mockWrapper, mockRepo)

			// 핸들러 생성
			handler := NewClaudeHandler(mockWrapper, mockRepo, mockHub)

			// 요청 준비
			req := httptest.NewRequest(http.MethodGet, "/api/v1/claude/sessions/"+tt.sessionID, nil)

			// 응답 레코더 생성
			w := httptest.NewRecorder()

			// Gin 컨텍스트 생성
			c, _ := gin.CreateTestContext(w)
			c.Request = req
			c.Params = gin.Params{
				{Key: "id", Value: tt.sessionID},
			}

			// 핸들러 실행
			handler.GetSession(c)

			// 결과 검증
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var response SessionResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.sessionID, response.ID)
			}

			// Mock 검증
			mockWrapper.AssertExpectations(t)
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestClaudeHandler_CloseSession(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		sessionID      string
		expectedStatus int
		setupMocks     func(*MockClaudeWrapper, *MockSessionRepository)
	}{
		{
			name:           "성공적인 세션 종료",
			sessionID:      "session-1",
			expectedStatus: http.StatusOK,
			setupMocks: func(wrapper *MockClaudeWrapper, repo *MockSessionRepository) {
				wrapper.On("CloseSession", "session-1").Return(nil)
			},
		},
		{
			name:           "세션 종료 실패",
			sessionID:      "session-1",
			expectedStatus: http.StatusInternalServerError,
			setupMocks: func(wrapper *MockClaudeWrapper, repo *MockSessionRepository) {
				wrapper.On("CloseSession", "session-1").Return(&claude.ClaudeError{
					Code:    "PROCESS_FAILED",
					Message: "Failed to close session",
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock 객체 생성
			mockWrapper := new(MockClaudeWrapper)
			mockRepo := new(MockSessionRepository)
			mockHub := websocket.NewHub(nil)

			// Mock 설정
			tt.setupMocks(mockWrapper, mockRepo)

			// 핸들러 생성
			handler := NewClaudeHandler(mockWrapper, mockRepo, mockHub)

			// 요청 준비
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/claude/sessions/"+tt.sessionID, nil)

			// 응답 레코더 생성
			w := httptest.NewRecorder()

			// Gin 컨텍스트 생성
			c, _ := gin.CreateTestContext(w)
			c.Request = req
			c.Params = gin.Params{
				{Key: "id", Value: tt.sessionID},
			}

			// 핸들러 실행
			handler.CloseSession(c)

			// 결과 검증
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, "Session closed", response["message"])
				assert.Equal(t, tt.sessionID, response["session_id"])
			}

			// Mock 검증
			mockWrapper.AssertExpectations(t)
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestClaudeHandler_executeAsync(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Mock 객체 생성
	mockWrapper := new(MockClaudeWrapper)
	mockRepo := new(MockSessionRepository)
	mockHub := websocket.NewHub(nil)

	// 핸들러 생성
	handler := NewClaudeHandler(mockWrapper, mockRepo, mockHub)

	// 테스트 데이터
	session := &claude.Session{
		ID:          "session-1",
		WorkspaceID: "workspace-1",
		UserID:      "user-1",
		State:       claude.SessionState{Status: "idle"},
		Created:     time.Now(),
		LastActive:  time.Now(),
	}

	req := ExecuteRequest{
		WorkspaceID: "workspace-1",
		Prompt:      "Hello, Claude!",
		Stream:      true,
	}

	executionID := "execution-1"

	// Mock 설정
	mockWrapper.On("Execute", "session-1", "Hello, Claude!").Return("Hello, human!", nil)

	// 비동기 실행 테스트
	ctx := context.Background()
	handler.executeAsync(ctx, session, req, executionID)

	// Mock 검증 (약간의 지연 후)
	time.Sleep(100 * time.Millisecond)
	mockWrapper.AssertExpectations(t)
}