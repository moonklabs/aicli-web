package services

import (
	"context"
	"time"

	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/storage/interfaces"
	"github.com/stretchr/testify/mock"
)

// MockWorkspaceStorage 워크스페이스 스토리지 Mock
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

func (m *MockWorkspaceStorage) GetByName(ctx context.Context, ownerID, name string) (*models.Workspace, error) {
	args := m.Called(ctx, ownerID, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Workspace), args.Error(1)
}

func (m *MockWorkspaceStorage) GetByOwnerID(ctx context.Context, ownerID string, pagination *models.PaginationRequest) ([]*models.Workspace, int, error) {
	args := m.Called(ctx, ownerID, pagination)
	return args.Get(0).([]*models.Workspace), args.Int(1), args.Error(2)
}

func (m *MockWorkspaceStorage) CountByOwner(ctx context.Context, ownerID string) (int, error) {
	args := m.Called(ctx, ownerID)
	return args.Int(0), args.Error(1)
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
	return args.Get(0).([]*models.Workspace), args.Int(1), args.Error(2)
}

func (m *MockWorkspaceStorage) ExistsByName(ctx context.Context, ownerID, name string) (bool, error) {
	args := m.Called(ctx, ownerID, name)
	return args.Bool(0), args.Error(1)
}

// MockProjectStorage 프로젝트 스토리지 Mock
type MockProjectStorage struct {
	mock.Mock
}

func (m *MockProjectStorage) Create(ctx context.Context, project *models.Project) error {
	args := m.Called(ctx, project)
	return args.Error(0)
}

func (m *MockProjectStorage) GetByID(ctx context.Context, id string) (*models.Project, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Project), args.Error(1)
}

func (m *MockProjectStorage) GetByWorkspaceID(ctx context.Context, workspaceID string, pagination *models.PaginationRequest) ([]*models.Project, int, error) {
	args := m.Called(ctx, workspaceID, pagination)
	return args.Get(0).([]*models.Project), args.Int(1), args.Error(2)
}

func (m *MockProjectStorage) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	args := m.Called(ctx, id, updates)
	return args.Error(0)
}

func (m *MockProjectStorage) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockProjectStorage) ExistsByName(ctx context.Context, workspaceID, name string) (bool, error) {
	args := m.Called(ctx, workspaceID, name)
	return args.Bool(0), args.Error(1)
}

func (m *MockProjectStorage) GetByPath(ctx context.Context, path string) (*models.Project, error) {
	args := m.Called(ctx, path)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Project), args.Error(1)
}

// MockSessionStorage 세션 스토리지 Mock
type MockSessionStorage struct {
	mock.Mock
}

func (m *MockSessionStorage) Create(ctx context.Context, session *models.Session) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *MockSessionStorage) GetByID(ctx context.Context, id string) (*models.Session, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Session), args.Error(1)
}

func (m *MockSessionStorage) List(ctx context.Context, filter *models.SessionFilter, paging *models.PaginationRequest) (*models.PaginationResponse, error) {
	args := m.Called(ctx, filter, paging)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.PaginationResponse), args.Error(1)
}

func (m *MockSessionStorage) Update(ctx context.Context, session *models.Session) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *MockSessionStorage) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockSessionStorage) GetActiveCount(ctx context.Context, projectID string) (int64, error) {
	args := m.Called(ctx, projectID)
	return args.Get(0).(int64), args.Error(1)
}

// MockTaskStorage 태스크 스토리지 Mock
type MockTaskStorage struct {
	mock.Mock
}

func (m *MockTaskStorage) Create(ctx context.Context, task *models.Task) error {
	args := m.Called(ctx, task)
	return args.Error(0)
}

func (m *MockTaskStorage) GetByID(ctx context.Context, id string) (*models.Task, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Task), args.Error(1)
}

func (m *MockTaskStorage) List(ctx context.Context, filter *models.TaskFilter, paging *models.PaginationRequest) ([]*models.Task, int, error) {
	args := m.Called(ctx, filter, paging)
	return args.Get(0).([]*models.Task), args.Int(1), args.Error(2)
}

func (m *MockTaskStorage) Update(ctx context.Context, task *models.Task) error {
	args := m.Called(ctx, task)
	return args.Error(0)
}

func (m *MockTaskStorage) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockTaskStorage) GetBySessionID(ctx context.Context, sessionID string, paging *models.PaginationRequest) ([]*models.Task, int, error) {
	args := m.Called(ctx, sessionID, paging)
	return args.Get(0).([]*models.Task), args.Int(1), args.Error(2)
}

func (m *MockTaskStorage) GetActiveCount(ctx context.Context, sessionID string) (int64, error) {
	args := m.Called(ctx, sessionID)
	return args.Get(0).(int64), args.Error(1)
}

// MockStorage 전체 스토리지 Mock
type MockStorage struct {
	mock.Mock
	workspaceStorage *MockWorkspaceStorage
	projectStorage   *MockProjectStorage
	sessionStorage   *MockSessionStorage
	taskStorage      *MockTaskStorage
}

func NewMockStorage() *MockStorage {
	return &MockStorage{
		workspaceStorage: new(MockWorkspaceStorage),
		projectStorage:   new(MockProjectStorage),
		sessionStorage:   new(MockSessionStorage),
		taskStorage:      new(MockTaskStorage),
	}
}

func (m *MockStorage) Workspace() interfaces.WorkspaceStorage {
	return m.workspaceStorage
}

func (m *MockStorage) Project() interfaces.ProjectStorage {
	return m.projectStorage
}

func (m *MockStorage) Session() interfaces.SessionStorage {
	return m.sessionStorage
}

func (m *MockStorage) Task() interfaces.TaskStorage {
	return m.taskStorage
}

func (m *MockStorage) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockStorage) GetByField(ctx context.Context, collection string, field string, value interface{}, result interface{}) error {
	args := m.Called(ctx, collection, field, value, result)
	return args.Error(0)
}

func (m *MockStorage) Create(ctx context.Context, collection string, data interface{}) error {
	args := m.Called(ctx, collection, data)
	return args.Error(0)
}

func (m *MockStorage) GetAll(ctx context.Context, collection string, result interface{}) error {
	args := m.Called(ctx, collection, result)
	return args.Error(0)
}

func (m *MockStorage) GetByID(ctx context.Context, collection string, id string, result interface{}) error {
	args := m.Called(ctx, collection, id, result)
	return args.Error(0)
}

func (m *MockStorage) Update(ctx context.Context, collection string, id string, updates interface{}) error {
	args := m.Called(ctx, collection, id, updates)
	return args.Error(0)
}

func (m *MockStorage) Delete(ctx context.Context, collection string, id string) error {
	args := m.Called(ctx, collection, id)
	return args.Error(0)
}

// MockDockerManager Docker Manager Mock
type MockDockerManager struct {
	mock.Mock
}

func (m *MockDockerManager) CreateContainer(ctx context.Context, config interface{}) (string, error) {
	args := m.Called(ctx, config)
	return args.String(0), args.Error(1)
}

func (m *MockDockerManager) StartContainer(ctx context.Context, containerID string) error {
	args := m.Called(ctx, containerID)
	return args.Error(0)
}

func (m *MockDockerManager) StopContainer(ctx context.Context, containerID string, timeout *time.Duration) error {
	args := m.Called(ctx, containerID, timeout)
	return args.Error(0)
}

func (m *MockDockerManager) RemoveContainer(ctx context.Context, containerID string) error {
	args := m.Called(ctx, containerID)
	return args.Error(0)
}

func (m *MockDockerManager) GetContainerInfo(ctx context.Context, containerID string) (interface{}, error) {
	args := m.Called(ctx, containerID)
	return args.Get(0), args.Error(1)
}

func (m *MockDockerManager) ListContainers(ctx context.Context, options interface{}) ([]interface{}, error) {
	args := m.Called(ctx, options)
	return args.Get(0).([]interface{}), args.Error(1)
}

func (m *MockDockerManager) ExecCommand(ctx context.Context, containerID string, cmd []string) (string, error) {
	args := m.Called(ctx, containerID, cmd)
	return args.String(0), args.Error(1)
}

func (m *MockDockerManager) StreamLogs(ctx context.Context, containerID string, options interface{}) (interface{}, error) {
	args := m.Called(ctx, containerID, options)
	return args.Get(0), args.Error(1)
}

func (m *MockDockerManager) CopyToContainer(ctx context.Context, containerID, path string, content interface{}) error {
	args := m.Called(ctx, containerID, path, content)
	return args.Error(0)
}

func (m *MockDockerManager) CopyFromContainer(ctx context.Context, containerID, path string) (interface{}, error) {
	args := m.Called(ctx, containerID, path)
	return args.Get(0), args.Error(1)
}