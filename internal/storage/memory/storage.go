package memory

import (
	"aicli-web/internal/storage"
)

// Storage 메모리 기반 스토리지 구현
type Storage struct {
	workspace *WorkspaceStorage
	project   *ProjectStorage
	session   *SessionStorage
}

// New 새 메모리 스토리지 생성
func New() *Storage {
	return &Storage{
		workspace: NewWorkspaceStorage(),
		project:   NewProjectStorage(),
		session:   NewSessionStorage(),
	}
}

// Workspace 워크스페이스 스토리지 반환
func (s *Storage) Workspace() storage.WorkspaceStorage {
	return s.workspace
}

// Project 프로젝트 스토리지 반환
func (s *Storage) Project() storage.ProjectStorage {
	return s.project
}

// Session 세션 스토리지 반환
func (s *Storage) Session() storage.SessionStorage {
	return s.session
}

// Close 스토리지 연결 종료 (메모리 스토리지는 아무 작업 없음)
func (s *Storage) Close() error {
	return nil
}