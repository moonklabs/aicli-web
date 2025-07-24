package memory

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/storage"
)

// ProjectStorage 메모리 기반 프로젝트 스토리지
type ProjectStorage struct {
	mu       sync.RWMutex
	projects map[string]*models.Project
	nameIndex map[string]string // workspaceID:name -> projectID
	pathIndex map[string]string // path -> projectID
}

// storage.ProjectStorage 인터페이스 구현 확인
var _ storage.ProjectStorage = (*ProjectStorage)(nil)

// NewProjectStorage 새 프로젝트 스토리지 생성
func NewProjectStorage() *ProjectStorage {
	return &ProjectStorage{
		projects:  make(map[string]*models.Project),
		nameIndex: make(map[string]string),
		pathIndex: make(map[string]string),
	}
}

// Create 새 프로젝트 생성
func (s *ProjectStorage) Create(ctx context.Context, project *models.Project) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// ID 생성
	if project.ID == "" {
		project.ID = "prj_" + uuid.New().String()
	}

	// 이름 중복 확인
	nameKey := fmt.Sprintf("%s:%s", project.WorkspaceID, project.Name)
	if _, exists := s.nameIndex[nameKey]; exists {
		return ErrAlreadyExists
	}

	// 경로 중복 확인
	if _, exists := s.pathIndex[project.Path]; exists {
		return ErrAlreadyExists
	}

	// 기본값 설정
	now := time.Now()
	project.CreatedAt = now
	project.UpdatedAt = now
	if project.Status == "" {
		project.Status = models.ProjectStatusActive
	}

	// 저장
	s.projects[project.ID] = project
	s.nameIndex[nameKey] = project.ID
	s.pathIndex[project.Path] = project.ID

	return nil
}

// GetByID ID로 프로젝트 조회
func (s *ProjectStorage) GetByID(ctx context.Context, id string) (*models.Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	project, exists := s.projects[id]
	if !exists || project.DeletedAt != nil {
		return nil, ErrNotFound
	}

	// 복사본 반환
	result := *project
	return &result, nil
}

// GetByWorkspaceID 워크스페이스 ID로 프로젝트 목록 조회
func (s *ProjectStorage) GetByWorkspaceID(ctx context.Context, workspaceID string, pagination *models.PaginationRequest) ([]*models.Project, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 필터링
	var filtered []*models.Project
	for _, prj := range s.projects {
		if prj.WorkspaceID == workspaceID && prj.DeletedAt == nil {
			filtered = append(filtered, prj)
		}
	}

	// 정렬
	sort.Slice(filtered, func(i, j int) bool {
		switch pagination.Sort {
		case "name":
			if pagination.Order == "asc" {
				return filtered[i].Name < filtered[j].Name
			}
			return filtered[i].Name > filtered[j].Name
		case "updated_at":
			if pagination.Order == "asc" {
				return filtered[i].UpdatedAt.Before(filtered[j].UpdatedAt)
			}
			return filtered[i].UpdatedAt.After(filtered[j].UpdatedAt)
		default: // created_at
			if pagination.Order == "asc" {
				return filtered[i].CreatedAt.Before(filtered[j].CreatedAt)
			}
			return filtered[i].CreatedAt.After(filtered[j].CreatedAt)
		}
	})

	// 페이지네이션
	total := len(filtered)
	offset := pagination.GetOffset()
	limit := pagination.Limit

	if offset >= total {
		return []*models.Project{}, total, nil
	}

	end := offset + limit
	if end > total {
		end = total
	}

	// 복사본 반환
	result := make([]*models.Project, 0, end-offset)
	for i := offset; i < end; i++ {
		prj := *filtered[i]
		result = append(result, &prj)
	}

	return result, total, nil
}

// Update 프로젝트 업데이트
func (s *ProjectStorage) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	project, exists := s.projects[id]
	if !exists || project.DeletedAt != nil {
		return ErrNotFound
	}

	// 이름 변경 시 중복 확인
	if name, ok := updates["name"].(string); ok && name != project.Name {
		nameKey := fmt.Sprintf("%s:%s", project.WorkspaceID, name)
		if existingID, exists := s.nameIndex[nameKey]; exists && existingID != id {
			return ErrAlreadyExists
		}
		
		// 기존 인덱스 삭제
		oldNameKey := fmt.Sprintf("%s:%s", project.WorkspaceID, project.Name)
		delete(s.nameIndex, oldNameKey)
		
		// 새 인덱스 추가
		s.nameIndex[nameKey] = id
		project.Name = name
	}

	// 경로 변경 시 중복 확인
	if path, ok := updates["path"].(string); ok && path != project.Path {
		if existingID, exists := s.pathIndex[path]; exists && existingID != id {
			return ErrAlreadyExists
		}
		
		// 기존 인덱스 삭제
		delete(s.pathIndex, project.Path)
		
		// 새 인덱스 추가
		s.pathIndex[path] = id
		project.Path = path
	}

	// 다른 필드 업데이트
	if description, ok := updates["description"].(string); ok {
		project.Description = description
	}
	if gitURL, ok := updates["git_url"].(string); ok {
		project.GitURL = gitURL
	}
	if gitBranch, ok := updates["git_branch"].(string); ok {
		project.GitBranch = gitBranch
	}
	if language, ok := updates["language"].(string); ok {
		project.Language = language
	}
	if status, ok := updates["status"].(models.ProjectStatus); ok {
		project.Status = status
	}
	if config, ok := updates["config"].(models.ProjectConfig); ok {
		project.Config = config
	}

	project.UpdatedAt = time.Now()

	return nil
}

// Delete 프로젝트 삭제 (soft delete)
func (s *ProjectStorage) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	project, exists := s.projects[id]
	if !exists || project.DeletedAt != nil {
		return ErrNotFound
	}

	// Soft delete
	now := time.Now()
	project.DeletedAt = &now
	project.UpdatedAt = now

	// 인덱스에서 제거
	nameKey := fmt.Sprintf("%s:%s", project.WorkspaceID, project.Name)
	delete(s.nameIndex, nameKey)
	delete(s.pathIndex, project.Path)

	return nil
}

// ExistsByName 워크스페이스 내 이름으로 존재 여부 확인
func (s *ProjectStorage) ExistsByName(ctx context.Context, workspaceID, name string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	nameKey := fmt.Sprintf("%s:%s", workspaceID, name)
	projectID, exists := s.nameIndex[nameKey]
	if !exists {
		return false, nil
	}

	// 삭제되지 않은 경우만 true
	project, ok := s.projects[projectID]
	return ok && project.DeletedAt == nil, nil
}

// GetByPath 경로로 프로젝트 조회
func (s *ProjectStorage) GetByPath(ctx context.Context, path string) (*models.Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	projectID, exists := s.pathIndex[path]
	if !exists {
		return nil, ErrNotFound
	}

	project, exists := s.projects[projectID]
	if !exists || project.DeletedAt != nil {
		return nil, ErrNotFound
	}

	// 복사본 반환
	result := *project
	return &result, nil
}