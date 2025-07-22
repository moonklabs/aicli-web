package services

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/storage"
	"github.com/aicli/aicli-web/internal/storage/interfaces"
)

// MockWorkspaceStorage는 테스트용 워크스페이스 스토리지 모크입니다
type MockWorkspaceStorage struct {
	mock.Mock
}

func (m *MockWorkspaceStorage) Create(ctx context.Context, workspace *models.Workspace) error {
	args := m.Called(ctx, workspace)
	return args.Error(0)
}

func (m *MockWorkspaceStorage) GetByID(ctx context.Context, id string) (*models.Workspace, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Workspace), args.Error(1)
}

func (m *MockWorkspaceStorage) GetByOwnerID(ctx context.Context, ownerID string, pagination *models.PaginationRequest) ([]*models.Workspace, int, error) {
	args := m.Called(ctx, ownerID, pagination)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*models.Workspace), args.Int(1), args.Error(2)
}

func (m *MockWorkspaceStorage) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	args := m.Called(ctx, id, updates)
	return args.Error(0)
}

func (m *MockWorkspaceStorage) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockWorkspaceStorage) List(ctx context.Context, pagination *models.PaginationRequest) ([]*models.Workspace, int, error) {
	args := m.Called(ctx, pagination)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*models.Workspace), args.Int(1), args.Error(2)
}

func (m *MockWorkspaceStorage) ExistsByName(ctx context.Context, ownerID, name string) (bool, error) {
	args := m.Called(ctx, ownerID, name)
	return args.Bool(0), args.Error(1)
}

// MockStorage는 테스트용 스토리지 모크입니다
type MockStorage struct {
	mock.Mock
	workspaceStorage *MockWorkspaceStorage
}

func NewMockStorage() *MockStorage {
	return &MockStorage{
		workspaceStorage: &MockWorkspaceStorage{},
	}
}

func (m *MockStorage) Workspace() interfaces.WorkspaceStorage {
	return m.workspaceStorage
}

func (m *MockStorage) Project() interfaces.ProjectStorage {
	args := m.Called()
	return args.Get(0).(interfaces.ProjectStorage)
}

func (m *MockStorage) Session() interfaces.SessionStorage {
	args := m.Called()
	return args.Get(0).(interfaces.SessionStorage)
}

func (m *MockStorage) Task() interfaces.TaskStorage {
	args := m.Called()
	return args.Get(0).(interfaces.TaskStorage)
}

func (m *MockStorage) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestWorkspaceService_CreateWorkspace(t *testing.T) {
	mockStorage := NewMockStorage()
	service := NewWorkspaceService(mockStorage)
	ctx := context.Background()

	tests := []struct {
		name      string
		req       *models.CreateWorkspaceRequest
		ownerID   string
		setupMock func()
		wantErr   bool
		errType   error
	}{
		{
			name: "성공적인 워크스페이스 생성",
			req: &models.CreateWorkspaceRequest{
				Name:        "test-workspace",
				ProjectPath: "/tmp/test",
				ClaudeKey:   "sk-ant-test123456789012345678901234567890123456789012",
			},
			ownerID: "user123",
			setupMock: func() {
				// 워크스페이스 수 확인
				mockStorage.workspaceStorage.On("GetByOwnerID", ctx, "user123", mock.AnythingOfType("*models.PaginationRequest")).Return([]*models.Workspace{}, 0, nil)
				// 이름 중복 확인
				mockStorage.workspaceStorage.On("ExistsByName", ctx, "user123", "test-workspace").Return(false, nil)
				// 워크스페이스 생성
				mockStorage.workspaceStorage.On("Create", ctx, mock.AnythingOfType("*models.Workspace")).Return(nil)
			},
			wantErr: false,
		},
		{
			name:    "nil 요청",
			req:     nil,
			ownerID: "user123",
			setupMock: func() {
				// 모크 설정 없음
			},
			wantErr: true,
			errType: ErrInvalidRequest,
		},
		{
			name: "빈 소유자 ID",
			req: &models.CreateWorkspaceRequest{
				Name:        "test-workspace",
				ProjectPath: "/tmp/test",
			},
			ownerID: "",
			setupMock: func() {
				// 모크 설정 없음
			},
			wantErr: true,
			errType: ErrInvalidRequest,
		},
		{
			name: "중복된 워크스페이스 이름",
			req: &models.CreateWorkspaceRequest{
				Name:        "test-workspace",
				ProjectPath: "/tmp/test",
			},
			ownerID: "user123",
			setupMock: func() {
				// 워크스페이스 수 확인
				mockStorage.workspaceStorage.On("GetByOwnerID", ctx, "user123", mock.AnythingOfType("*models.PaginationRequest")).Return([]*models.Workspace{}, 0, nil)
				// 이름 중복 확인 - 이미 존재
				mockStorage.workspaceStorage.On("ExistsByName", ctx, "user123", "test-workspace").Return(true, nil)
			},
			wantErr: true,
			errType: ErrWorkspaceExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 모크 초기화
			mockStorage.workspaceStorage.Mock = mock.Mock{}
			
			// 테스트별 모크 설정
			tt.setupMock()

			// 테스트 실행
			workspace, err := service.CreateWorkspace(ctx, tt.req, tt.ownerID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, workspace)
				if tt.errType != nil {
					assert.ErrorIs(t, err, tt.errType)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, workspace)
				assert.Equal(t, tt.req.Name, workspace.Name)
				assert.Equal(t, tt.req.ProjectPath, workspace.ProjectPath)
				assert.Equal(t, tt.ownerID, workspace.OwnerID)
				assert.Equal(t, models.WorkspaceStatusActive, workspace.Status)
				assert.Zero(t, workspace.ActiveTasks)
				assert.NotZero(t, workspace.CreatedAt)
				assert.NotZero(t, workspace.UpdatedAt)
			}

			// 모든 모크 호출이 예상대로 이루어졌는지 확인
			mockStorage.workspaceStorage.AssertExpectations(t)
		})
	}
}

func TestWorkspaceService_GetWorkspace(t *testing.T) {
	mockStorage := NewMockStorage()
	service := NewWorkspaceService(mockStorage)
	ctx := context.Background()

	testWorkspace := &models.Workspace{
		ID:          "ws123",
		Name:        "test-workspace",
		ProjectPath: "/tmp/test",
		Status:      models.WorkspaceStatusActive,
		OwnerID:     "user123",
		ClaudeKey:   "sk-ant-test123456789012345678901234567890123456789012",
		ActiveTasks: 0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	tests := []struct {
		name      string
		id        string
		ownerID   string
		setupMock func()
		wantErr   bool
		errType   error
	}{
		{
			name:    "성공적인 워크스페이스 조회",
			id:      "ws123",
			ownerID: "user123",
			setupMock: func() {
				mockStorage.workspaceStorage.On("GetByID", ctx, "ws123").Return(testWorkspace, nil)
			},
			wantErr: false,
		},
		{
			name:    "빈 워크스페이스 ID",
			id:      "",
			ownerID: "user123",
			setupMock: func() {
				// 모크 설정 없음
			},
			wantErr: true,
			errType: ErrInvalidRequest,
		},
		{
			name:    "빈 소유자 ID",
			id:      "ws123",
			ownerID: "",
			setupMock: func() {
				// 모크 설정 없음
			},
			wantErr: true,
			errType: ErrUnauthorized,
		},
		{
			name:    "존재하지 않는 워크스페이스",
			id:      "ws999",
			ownerID: "user123",
			setupMock: func() {
				mockStorage.workspaceStorage.On("GetByID", ctx, "ws999").Return(nil, storage.ErrNotFound)
			},
			wantErr: true,
			errType: ErrWorkspaceNotFound,
		},
		{
			name:    "권한 없는 접근",
			id:      "ws123",
			ownerID: "user999",
			setupMock: func() {
				mockStorage.workspaceStorage.On("GetByID", ctx, "ws123").Return(testWorkspace, nil)
			},
			wantErr: true,
			errType: ErrUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 모크 초기화
			mockStorage.workspaceStorage.Mock = mock.Mock{}
			
			// 테스트별 모크 설정
			tt.setupMock()

			// 테스트 실행
			workspace, err := service.GetWorkspace(ctx, tt.id, tt.ownerID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, workspace)
				if tt.errType != nil {
					assert.ErrorIs(t, err, tt.errType)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, workspace)
				assert.Equal(t, tt.id, workspace.ID)
				assert.Equal(t, tt.ownerID, workspace.OwnerID)
				// Claude 키가 마스킹되었는지 확인
				if testWorkspace.ClaudeKey != "" {
					assert.NotEqual(t, testWorkspace.ClaudeKey, workspace.ClaudeKey)
					assert.Contains(t, workspace.ClaudeKey, "...")
				}
			}

			// 모든 모크 호출이 예상대로 이루어졌는지 확인
			mockStorage.workspaceStorage.AssertExpectations(t)
		})
	}
}

func TestWorkspaceService_UpdateWorkspace(t *testing.T) {
	mockStorage := NewMockStorage()
	service := NewWorkspaceService(mockStorage)
	ctx := context.Background()

	existingWorkspace := &models.Workspace{
		ID:          "ws123",
		Name:        "original-workspace",
		ProjectPath: "/tmp/original",
		Status:      models.WorkspaceStatusActive,
		OwnerID:     "user123",
		ClaudeKey:   "sk-ant-original123456789012345678901234567890123456789012",
		ActiveTasks: 0,
		CreatedAt:   time.Now().Add(-time.Hour),
		UpdatedAt:   time.Now().Add(-time.Hour),
	}

	updatedWorkspace := &models.Workspace{
		ID:          "ws123",
		Name:        "updated-workspace",
		ProjectPath: "/tmp/updated",
		Status:      models.WorkspaceStatusActive,
		OwnerID:     "user123",
		ClaudeKey:   "sk-ant-updated123456789012345678901234567890123456789012",
		ActiveTasks: 0,
		CreatedAt:   existingWorkspace.CreatedAt,
		UpdatedAt:   time.Now(),
	}

	tests := []struct {
		name      string
		id        string
		req       *models.UpdateWorkspaceRequest
		ownerID   string
		setupMock func()
		wantErr   bool
		errType   error
	}{
		{
			name: "성공적인 워크스페이스 업데이트",
			id:   "ws123",
			req: &models.UpdateWorkspaceRequest{
				Name:        "updated-workspace",
				ProjectPath: "/tmp/updated",
			},
			ownerID: "user123",
			setupMock: func() {
				// 기존 워크스페이스 조회
				mockStorage.workspaceStorage.On("GetByID", ctx, "ws123").Return(existingWorkspace, nil).Times(2)
				// 이름 중복 확인
				mockStorage.workspaceStorage.On("ExistsByName", ctx, "user123", "updated-workspace").Return(false, nil)
				// 업데이트 실행
				mockStorage.workspaceStorage.On("Update", ctx, "ws123", mock.AnythingOfType("map[string]interface {}")).Return(nil)
			},
			wantErr: false,
		},
		{
			name:    "빈 워크스페이스 ID",
			id:      "",
			req:     &models.UpdateWorkspaceRequest{},
			ownerID: "user123",
			setupMock: func() {
				// 모크 설정 없음
			},
			wantErr: true,
			errType: ErrInvalidRequest,
		},
		{
			name: "nil 요청",
			id:   "ws123",
			req:  nil,
			ownerID: "user123",
			setupMock: func() {
				// 모크 설정 없음
			},
			wantErr: true,
			errType: ErrInvalidRequest,
		},
		{
			name: "존재하지 않는 워크스페이스",
			id:   "ws999",
			req: &models.UpdateWorkspaceRequest{
				Name: "updated-workspace",
			},
			ownerID: "user123",
			setupMock: func() {
				mockStorage.workspaceStorage.On("GetByID", ctx, "ws999").Return(nil, storage.ErrNotFound)
			},
			wantErr: true,
			errType: ErrWorkspaceNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 모크 초기화
			mockStorage.workspaceStorage.Mock = mock.Mock{}
			
			// 테스트별 모크 설정
			tt.setupMock()
			
			// 성공 케이스의 경우 업데이트된 워크스페이스 반환 설정
			if !tt.wantErr && tt.name == "성공적인 워크스페이스 업데이트" {
				// 업데이트 후 조회를 위한 추가 모크 (GetWorkspace 내부에서 호출)
				mockStorage.workspaceStorage.On("GetByID", ctx, tt.id).Return(updatedWorkspace, nil).Once()
			}

			// 테스트 실행
			workspace, err := service.UpdateWorkspace(ctx, tt.id, tt.req, tt.ownerID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, workspace)
				if tt.errType != nil {
					assert.ErrorIs(t, err, tt.errType)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, workspace)
				assert.Equal(t, tt.id, workspace.ID)
				if tt.req.Name != "" {
					assert.Equal(t, tt.req.Name, workspace.Name)
				}
				if tt.req.ProjectPath != "" {
					assert.Equal(t, tt.req.ProjectPath, workspace.ProjectPath)
				}
			}

			// 모든 모크 호출이 예상대로 이루어졌는지 확인
			mockStorage.workspaceStorage.AssertExpectations(t)
		})
	}
}

func TestWorkspaceService_DeleteWorkspace(t *testing.T) {
	mockStorage := NewMockStorage()
	service := NewWorkspaceService(mockStorage)
	ctx := context.Background()

	testWorkspace := &models.Workspace{
		ID:          "ws123",
		Name:        "test-workspace",
		ProjectPath: "/tmp/test",
		Status:      models.WorkspaceStatusActive,
		OwnerID:     "user123",
		ActiveTasks: 0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	busyWorkspace := &models.Workspace{
		ID:          "ws124",
		Name:        "busy-workspace",
		ProjectPath: "/tmp/busy",
		Status:      models.WorkspaceStatusActive,
		OwnerID:     "user123",
		ActiveTasks: 3, // 활성 태스크가 있음
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	tests := []struct {
		name      string
		id        string
		ownerID   string
		setupMock func()
		wantErr   bool
		errType   error
	}{
		{
			name:    "성공적인 워크스페이스 삭제",
			id:      "ws123",
			ownerID: "user123",
			setupMock: func() {
				// 워크스페이스 조회 (GetWorkspace 호출)
				mockStorage.workspaceStorage.On("GetByID", ctx, "ws123").Return(testWorkspace, nil)
				// 삭제 실행
				mockStorage.workspaceStorage.On("Delete", ctx, "ws123").Return(nil)
			},
			wantErr: false,
		},
		{
			name:    "빈 워크스페이스 ID",
			id:      "",
			ownerID: "user123",
			setupMock: func() {
				// 모크 설정 없음
			},
			wantErr: true,
			errType: ErrInvalidRequest,
		},
		{
			name:    "빈 소유자 ID",
			id:      "ws123",
			ownerID: "",
			setupMock: func() {
				// 모크 설정 없음
			},
			wantErr: true,
			errType: ErrUnauthorized,
		},
		{
			name:    "존재하지 않는 워크스페이스",
			id:      "ws999",
			ownerID: "user123",
			setupMock: func() {
				mockStorage.workspaceStorage.On("GetByID", ctx, "ws999").Return(nil, storage.ErrNotFound)
			},
			wantErr: true,
			errType: ErrWorkspaceNotFound,
		},
		{
			name:    "활성 태스크가 있는 워크스페이스 삭제 시도",
			id:      "ws124",
			ownerID: "user123",
			setupMock: func() {
				mockStorage.workspaceStorage.On("GetByID", ctx, "ws124").Return(busyWorkspace, nil)
			},
			wantErr: true,
			errType: ErrResourceBusy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 모크 초기화
			mockStorage.workspaceStorage.Mock = mock.Mock{}
			
			// 테스트별 모크 설정
			tt.setupMock()

			// 테스트 실행
			err := service.DeleteWorkspace(ctx, tt.id, tt.ownerID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errType != nil {
					assert.ErrorIs(t, err, tt.errType)
				}
			} else {
				assert.NoError(t, err)
			}

			// 모든 모크 호출이 예상대로 이루어졌는지 확인
			mockStorage.workspaceStorage.AssertExpectations(t)
		})
	}
}

func TestWorkspaceService_ListWorkspaces(t *testing.T) {
	mockStorage := NewMockStorage()
	service := NewWorkspaceService(mockStorage)
	ctx := context.Background()

	testWorkspaces := []*models.Workspace{
		{
			ID:          "ws123",
			Name:        "workspace-1",
			ProjectPath: "/tmp/test1",
			Status:      models.WorkspaceStatusActive,
			OwnerID:     "user123",
			ActiveTasks: 1,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          "ws124",
			Name:        "workspace-2",
			ProjectPath: "/tmp/test2",
			Status:      models.WorkspaceStatusInactive,
			OwnerID:     "user123",
			ActiveTasks: 0,
			CreatedAt:   time.Now().Add(-time.Hour),
			UpdatedAt:   time.Now().Add(-time.Hour),
		},
	}

	tests := []struct {
		name      string
		ownerID   string
		req       *models.PaginationRequest
		setupMock func()
		wantErr   bool
		errType   error
		wantCount int
	}{
		{
			name:    "성공적인 워크스페이스 목록 조회",
			ownerID: "user123",
			req: &models.PaginationRequest{
				Page:  1,
				Limit: 10,
				Sort:  "created_at",
				Order: "desc",
			},
			setupMock: func() {
				mockStorage.workspaceStorage.On("GetByOwnerID", ctx, "user123", mock.AnythingOfType("*models.PaginationRequest")).Return(testWorkspaces, 2, nil)
			},
			wantErr:   false,
			wantCount: 2,
		},
		{
			name:    "빈 소유자 ID",
			ownerID: "",
			req: &models.PaginationRequest{
				Page:  1,
				Limit: 10,
			},
			setupMock: func() {
				// 모크 설정 없음
			},
			wantErr: true,
			errType: ErrUnauthorized,
		},
		{
			name:    "nil 페이지네이션 요청 (기본값 사용)",
			ownerID: "user123",
			req:     nil,
			setupMock: func() {
				mockStorage.workspaceStorage.On("GetByOwnerID", ctx, "user123", mock.AnythingOfType("*models.PaginationRequest")).Return([]*models.Workspace{}, 0, nil)
			},
			wantErr:   false,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 모크 초기화
			mockStorage.workspaceStorage.Mock = mock.Mock{}
			
			// 테스트별 모크 설정
			tt.setupMock()

			// 테스트 실행
			response, err := service.ListWorkspaces(ctx, tt.ownerID, tt.req)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, response)
				if tt.errType != nil {
					assert.ErrorIs(t, err, tt.errType)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, response)
				assert.Equal(t, tt.wantCount, len(response.Data))
				assert.Equal(t, tt.wantCount, response.Meta.Total)
				
				// 페이지네이션 메타 검증
				if tt.req != nil {
					assert.Equal(t, tt.req.Page, response.Meta.Page)
					assert.Equal(t, tt.req.Limit, response.Meta.Limit)
				} else {
					// 기본값 검증
					assert.Equal(t, 1, response.Meta.Page)
					assert.Equal(t, 10, response.Meta.Limit)
				}
			}

			// 모든 모크 호출이 예상대로 이루어졌는지 확인
			mockStorage.workspaceStorage.AssertExpectations(t)
		})
	}
}

func TestWorkspaceService_UpdateActiveTaskCount(t *testing.T) {
	mockStorage := NewMockStorage()
	service := NewWorkspaceService(mockStorage)
	ctx := context.Background()

	testWorkspace := &models.Workspace{
		ID:          "ws123",
		Name:        "test-workspace",
		ProjectPath: "/tmp/test",
		Status:      models.WorkspaceStatusActive,
		OwnerID:     "user123",
		ActiveTasks: 5,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	tests := []struct {
		name      string
		id        string
		delta     int
		setupMock func()
		wantErr   bool
		errType   error
	}{
		{
			name:  "활성 태스크 수 증가",
			id:    "ws123",
			delta: 2,
			setupMock: func() {
				mockStorage.workspaceStorage.On("GetByID", ctx, "ws123").Return(testWorkspace, nil)
				mockStorage.workspaceStorage.On("Update", ctx, "ws123", mock.MatchedBy(func(updates map[string]interface{}) bool {
					return updates["active_tasks"] == 7 // 5 + 2
				})).Return(nil)
			},
			wantErr: false,
		},
		{
			name:  "활성 태스크 수 감소",
			id:    "ws123",
			delta: -3,
			setupMock: func() {
				mockStorage.workspaceStorage.On("GetByID", ctx, "ws123").Return(testWorkspace, nil)
				mockStorage.workspaceStorage.On("Update", ctx, "ws123", mock.MatchedBy(func(updates map[string]interface{}) bool {
					return updates["active_tasks"] == 2 // 5 - 3
				})).Return(nil)
			},
			wantErr: false,
		},
		{
			name:  "활성 태스크 수가 음수가 되는 경우 (0으로 제한)",
			id:    "ws123",
			delta: -10,
			setupMock: func() {
				mockStorage.workspaceStorage.On("GetByID", ctx, "ws123").Return(testWorkspace, nil)
				mockStorage.workspaceStorage.On("Update", ctx, "ws123", mock.MatchedBy(func(updates map[string]interface{}) bool {
					return updates["active_tasks"] == 0 // 음수가 되면 0으로 제한
				})).Return(nil)
			},
			wantErr: false,
		},
		{
			name:  "빈 워크스페이스 ID",
			id:    "",
			delta: 1,
			setupMock: func() {
				// 모크 설정 없음
			},
			wantErr: true,
			errType: ErrInvalidRequest,
		},
		{
			name:  "존재하지 않는 워크스페이스",
			id:    "ws999",
			delta: 1,
			setupMock: func() {
				mockStorage.workspaceStorage.On("GetByID", ctx, "ws999").Return(nil, storage.ErrNotFound)
			},
			wantErr: true,
			errType: ErrWorkspaceNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 모크 초기화
			mockStorage.workspaceStorage.Mock = mock.Mock{}
			
			// 테스트별 모크 설정
			tt.setupMock()

			// 테스트 실행
			err := service.UpdateActiveTaskCount(ctx, tt.id, tt.delta)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errType != nil {
					assert.ErrorIs(t, err, tt.errType)
				}
			} else {
				assert.NoError(t, err)
			}

			// 모든 모크 호출이 예상대로 이루어졌는지 확인
			mockStorage.workspaceStorage.AssertExpectations(t)
		})
	}
}

func TestWorkspaceService_GetWorkspaceStats(t *testing.T) {
	mockStorage := NewMockStorage()
	service := NewWorkspaceService(mockStorage)
	ctx := context.Background()

	testWorkspaces := []*models.Workspace{
		{
			ID:          "ws1",
			Name:        "active-1",
			Status:      models.WorkspaceStatusActive,
			OwnerID:     "user123",
			ActiveTasks: 2,
		},
		{
			ID:          "ws2",
			Name:        "active-2",
			Status:      models.WorkspaceStatusActive,
			OwnerID:     "user123",
			ActiveTasks: 3,
		},
		{
			ID:          "ws3",
			Name:        "inactive-1",
			Status:      models.WorkspaceStatusInactive,
			OwnerID:     "user123",
			ActiveTasks: 0,
		},
		{
			ID:          "ws4",
			Name:        "archived-1",
			Status:      models.WorkspaceStatusArchived,
			OwnerID:     "user123",
			ActiveTasks: 0,
		},
	}

	tests := []struct {
		name         string
		ownerID      string
		setupMock    func()
		wantErr      bool
		errType      error
		expectedStats *WorkspaceStats
	}{
		{
			name:    "성공적인 워크스페이스 통계 조회",
			ownerID: "user123",
			setupMock: func() {
				mockStorage.workspaceStorage.On("GetByOwnerID", ctx, "user123", mock.AnythingOfType("*models.PaginationRequest")).Return(testWorkspaces, 4, nil)
			},
			wantErr: false,
			expectedStats: &WorkspaceStats{
				TotalWorkspaces:    4,
				ActiveWorkspaces:   2,
				ArchivedWorkspaces: 1,
				TotalActiveTasks:   5, // 2 + 3 + 0 + 0
			},
		},
		{
			name:    "빈 소유자 ID",
			ownerID: "",
			setupMock: func() {
				// 모크 설정 없음
			},
			wantErr: true,
			errType: ErrUnauthorized,
		},
		{
			name:    "워크스페이스가 없는 사용자",
			ownerID: "user999",
			setupMock: func() {
				mockStorage.workspaceStorage.On("GetByOwnerID", ctx, "user999", mock.AnythingOfType("*models.PaginationRequest")).Return([]*models.Workspace{}, 0, nil)
			},
			wantErr: false,
			expectedStats: &WorkspaceStats{
				TotalWorkspaces:    0,
				ActiveWorkspaces:   0,
				ArchivedWorkspaces: 0,
				TotalActiveTasks:   0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 모크 초기화
			mockStorage.workspaceStorage.Mock = mock.Mock{}
			
			// 테스트별 모크 설정
			tt.setupMock()

			// 테스트 실행
			stats, err := service.GetWorkspaceStats(ctx, tt.ownerID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, stats)
				if tt.errType != nil {
					assert.ErrorIs(t, err, tt.errType)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, stats)
				assert.Equal(t, tt.expectedStats.TotalWorkspaces, stats.TotalWorkspaces)
				assert.Equal(t, tt.expectedStats.ActiveWorkspaces, stats.ActiveWorkspaces)
				assert.Equal(t, tt.expectedStats.ArchivedWorkspaces, stats.ArchivedWorkspaces)
				assert.Equal(t, tt.expectedStats.TotalActiveTasks, stats.TotalActiveTasks)
			}

			// 모든 모크 호출이 예상대로 이루어졌는지 확인
			mockStorage.workspaceStorage.AssertExpectations(t)
		})
	}
}