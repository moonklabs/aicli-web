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

// Storage 전체 스토리지 인터페이스
type Storage interface {
	// Workspace 워크스페이스 스토리지 반환
	Workspace() WorkspaceStorage
	
	// Close 스토리지 연결 종료
	Close() error
}