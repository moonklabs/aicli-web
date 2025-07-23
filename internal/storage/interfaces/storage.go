package interfaces

import (
	"context"
	
	"github.com/aicli/aicli-web/internal/models"
)

// WorkspaceStorage 워크스페이스 스토리지 인터페이스
type WorkspaceStorage interface {
	// Create 새 워크스페이스 생성
	Create(ctx context.Context, workspace *models.Workspace) error
	
	// GetByID ID로 워크스페이스 조회
	GetByID(ctx context.Context, id string) (*models.Workspace, error)
	
	// GetByOwnerID 소유자 ID로 워크스페이스 목록 조회
	GetByOwnerID(ctx context.Context, ownerID string, pagination *models.PaginationRequest) ([]*models.Workspace, int, error)
	
	// Update 워크스페이스 업데이트
	Update(ctx context.Context, id string, updates map[string]interface{}) error
	
	// Delete 워크스페이스 삭제 (soft delete)
	Delete(ctx context.Context, id string) error
	
	// List 전체 워크스페이스 목록 조회 (관리자용)
	List(ctx context.Context, pagination *models.PaginationRequest) ([]*models.Workspace, int, error)
	
	// ExistsByName 이름으로 존재 여부 확인
	ExistsByName(ctx context.Context, ownerID, name string) (bool, error)
}

// ProjectStorage 프로젝트 스토리지 인터페이스
type ProjectStorage interface {
	// Create 새 프로젝트 생성
	Create(ctx context.Context, project *models.Project) error
	
	// GetByID ID로 프로젝트 조회
	GetByID(ctx context.Context, id string) (*models.Project, error)
	
	// GetByWorkspaceID 워크스페이스 ID로 프로젝트 목록 조회
	GetByWorkspaceID(ctx context.Context, workspaceID string, pagination *models.PaginationRequest) ([]*models.Project, int, error)
	
	// Update 프로젝트 업데이트
	Update(ctx context.Context, id string, updates map[string]interface{}) error
	
	// Delete 프로젝트 삭제 (soft delete)
	Delete(ctx context.Context, id string) error
	
	// ExistsByName 워크스페이스 내 이름으로 존재 여부 확인
	ExistsByName(ctx context.Context, workspaceID, name string) (bool, error)
	
	// GetByPath 경로로 프로젝트 조회
	GetByPath(ctx context.Context, path string) (*models.Project, error)
}

// SessionStorage 세션 스토리지 인터페이스
type SessionStorage interface {
	// Create 새 세션 생성
	Create(ctx context.Context, session *models.Session) error
	
	// GetByID ID로 세션 조회
	GetByID(ctx context.Context, id string) (*models.Session, error)
	
	// List 세션 목록 조회
	List(ctx context.Context, filter *models.SessionFilter, paging *models.PaginationRequest) (*models.PaginationResponse, error)
	
	// Update 세션 업데이트
	Update(ctx context.Context, session *models.Session) error
	
	// Delete 세션 삭제
	Delete(ctx context.Context, id string) error
	
	// GetActiveCount 활성 세션 수 조회
	GetActiveCount(ctx context.Context, projectID string) (int64, error)
}

// TaskStorage 태스크 스토리지 인터페이스
type TaskStorage interface {
	// Create 새 태스크 생성
	Create(ctx context.Context, task *models.Task) error
	
	// GetByID ID로 태스크 조회
	GetByID(ctx context.Context, id string) (*models.Task, error)
	
	// List 태스크 목록 조회
	List(ctx context.Context, filter *models.TaskFilter, paging *models.PaginationRequest) ([]*models.Task, int, error)
	
	// Update 태스크 업데이트
	Update(ctx context.Context, task *models.Task) error
	
	// Delete 태스크 삭제
	Delete(ctx context.Context, id string) error
	
	// GetBySessionID 세션 ID로 태스크 목록 조회
	GetBySessionID(ctx context.Context, sessionID string, paging *models.PaginationRequest) ([]*models.Task, int, error)
	
	// GetActiveCount 활성 태스크 수 조회
	GetActiveCount(ctx context.Context, sessionID string) (int64, error)
}

// Storage 전체 스토리지 인터페이스
type Storage interface {
	// Workspace 워크스페이스 스토리지 반환
	Workspace() WorkspaceStorage
	
	// Project 프로젝트 스토리지 반환
	Project() ProjectStorage
	
	// Session 세션 스토리지 반환
	Session() SessionStorage
	
	// Task 태스크 스토리지 반환
	Task() TaskStorage
	
	// Close 스토리지 연결 종료
	Close() error
	
	// Generic methods for policy service compatibility
	GetByField(ctx context.Context, collection string, field string, value interface{}, result interface{}) error
	Create(ctx context.Context, collection string, data interface{}) error
	GetAll(ctx context.Context, collection string, result interface{}) error
	GetByID(ctx context.Context, collection string, id string, result interface{}) error
	Update(ctx context.Context, collection string, id string, updates interface{}) error
	Delete(ctx context.Context, collection string, id string) error
}