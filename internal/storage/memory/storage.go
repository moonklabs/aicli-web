package memory

import (
	"context"
)

// Storage 메모리 기반 스토리지 구현
type Storage struct {
	workspace *WorkspaceStorage
	project   *ProjectStorage
	session   *SessionStorage
	task      *taskStorage
	rbac      *RBACStorage
}

// New 새 메모리 스토리지 생성
func New() *Storage {
	return &Storage{
		workspace: NewWorkspaceStorage(),
		project:   NewProjectStorage(),
		session:   NewSessionStorage(),
		task:      newTaskStorage(),
		rbac:      NewRBACStorage(),
	}
}

// Workspace 워크스페이스 스토리지 반환
func (s *Storage) Workspace() *WorkspaceStorage {
	return s.workspace
}

// Project 프로젝트 스토리지 반환
func (s *Storage) Project() *ProjectStorage {
	return s.project
}

// Session 세션 스토리지 반환
func (s *Storage) Session() *SessionStorage {
	return s.session
}

// Task 태스크 스토리지 반화
func (s *Storage) Task() *taskStorage {
	return s.task
}

// RBAC RBAC 스토리지 반환
func (s *Storage) RBAC() *RBACStorage {
	return s.rbac
}

// Close 스토리지 연결 종료 (메모리 스토리지는 아무 작업 없음)
func (s *Storage) Close() error {
	return nil
}

// GetByField 필드로 데이터 조회
func (s *Storage) GetByField(ctx context.Context, collection string, field string, value interface{}, result interface{}) error {
	// 메모리 스토리지에서는 간단히 구현
	return nil
}

// Create 데이터 생성
func (s *Storage) Create(ctx context.Context, collection string, data interface{}) error {
	// 메모리 스토리지에서는 간단히 구현
	return nil
}

// GetAll 전체 데이터 조회
func (s *Storage) GetAll(ctx context.Context, collection string, result interface{}) error {
	// 메모리 스토리지에서는 간단히 구현
	return nil
}

// GetByID ID로 데이터 조회
func (s *Storage) GetByID(ctx context.Context, collection string, id string, result interface{}) error {
	// 메모리 스토리지에서는 간단히 구현
	return nil
}

// Update 데이터 업데이트
func (s *Storage) Update(ctx context.Context, collection string, id string, updates interface{}) error {
	// 메모리 스토리지에서는 간단히 구현
	return nil
}

// Delete 데이터 삭제
func (s *Storage) Delete(ctx context.Context, collection string, id string) error {
	// 메모리 스토리지에서는 간단히 구현
	return nil
}