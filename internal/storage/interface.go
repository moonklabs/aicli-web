package storage

import (
	"context"
	"errors"
	
	"aicli-web/internal/models"
)

// 에러 정의
var (
	// ErrNotFound 리소스를 찾을 수 없음
	ErrNotFound = errors.New("resource not found")
	
	// ErrAlreadyExists 리소스가 이미 존재함
	ErrAlreadyExists = errors.New("resource already exists")
	
	// ErrInvalidInput 잘못된 입력
	ErrInvalidInput = errors.New("invalid input")
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

// Storage 전체 스토리지 인터페이스
type Storage interface {
	// Workspace 워크스페이스 스토리지 반환
	Workspace() WorkspaceStorage
	
	// Project 프로젝트 스토리지 반환
	Project() ProjectStorage
	
	// Close 스토리지 연결 종료
	Close() error
}