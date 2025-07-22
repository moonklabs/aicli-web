package memory

import (
	"github.com/aicli/aicli-web/internal/storage/interfaces"
)

// Storage 메모리 기반 스토리지 구현
type Storage struct {
	workspace *WorkspaceStorage
	project   *ProjectStorage
	session   *SessionStorage
	task      *taskStorage
}

// New 새 메모리 스토리지 생성
func New() *Storage {
	return &Storage{
		workspace: NewWorkspaceStorage(),
		project:   NewProjectStorage(),
		session:   NewSessionStorage(),
		task:      newTaskStorage(),
	}
}

// Workspace 워크스페이스 스토리지 반환
func (s *Storage) Workspace() interfaces.WorkspaceStorage {
	return s.workspace
}

// Project 프로젝트 스토리지 반환
func (s *Storage) Project() interfaces.ProjectStorage {
	return s.project
}

// Session 세션 스토리지 반환
func (s *Storage) Session() interfaces.SessionStorage {
	return s.session
}

// Task 태스크 스토리지 반환
func (s *Storage) Task() interfaces.TaskStorage {
	return s.task
}

// Close 스토리지 연결 종료 (메모리 스토리지는 아무 작업 없음)
func (s *Storage) Close() error {
	return nil
}