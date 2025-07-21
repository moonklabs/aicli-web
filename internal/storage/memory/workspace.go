package memory

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"aicli-web/internal/models"
	"aicli-web/internal/storage"
)

// WorkspaceStorage 메모리 기반 워크스페이스 스토리지
type WorkspaceStorage struct {
	mu         sync.RWMutex
	workspaces map[string]*models.Workspace
	nameIndex  map[string]string // ownerID:name -> workspaceID
}

// NewWorkspaceStorage 새 워크스페이스 스토리지 생성
func NewWorkspaceStorage() *WorkspaceStorage {
	return &WorkspaceStorage{
		workspaces: make(map[string]*models.Workspace),
		nameIndex:  make(map[string]string),
	}
}

// Create 새 워크스페이스 생성
func (s *WorkspaceStorage) Create(ctx context.Context, workspace *models.Workspace) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// ID 생성
	if workspace.ID == "" {
		workspace.ID = "ws_" + uuid.New().String()
	}

	// 이름 중복 확인
	nameKey := fmt.Sprintf("%s:%s", workspace.OwnerID, workspace.Name)
	if _, exists := s.nameIndex[nameKey]; exists {
		return storage.ErrAlreadyExists
	}

	// 기본값 설정
	now := time.Now()
	workspace.CreatedAt = now
	workspace.UpdatedAt = now
	if workspace.Status == "" {
		workspace.Status = models.WorkspaceStatusActive
	}

	// 저장
	s.workspaces[workspace.ID] = workspace
	s.nameIndex[nameKey] = workspace.ID

	return nil
}

// GetByID ID로 워크스페이스 조회
func (s *WorkspaceStorage) GetByID(ctx context.Context, id string) (*models.Workspace, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	workspace, exists := s.workspaces[id]
	if !exists || workspace.DeletedAt != nil {
		return nil, storage.ErrNotFound
	}

	// 복사본 반환
	result := *workspace
	return &result, nil
}

// GetByOwnerID 소유자 ID로 워크스페이스 목록 조회
func (s *WorkspaceStorage) GetByOwnerID(ctx context.Context, ownerID string, pagination *models.PaginationRequest) ([]*models.Workspace, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 필터링
	var filtered []*models.Workspace
	for _, ws := range s.workspaces {
		if ws.OwnerID == ownerID && ws.DeletedAt == nil {
			filtered = append(filtered, ws)
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
		return []*models.Workspace{}, total, nil
	}

	end := offset + limit
	if end > total {
		end = total
	}

	// 복사본 반환
	result := make([]*models.Workspace, 0, end-offset)
	for i := offset; i < end; i++ {
		ws := *filtered[i]
		result = append(result, &ws)
	}

	return result, total, nil
}

// Update 워크스페이스 업데이트
func (s *WorkspaceStorage) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	workspace, exists := s.workspaces[id]
	if !exists || workspace.DeletedAt != nil {
		return storage.ErrNotFound
	}

	// 이름 변경 시 중복 확인
	if name, ok := updates["name"].(string); ok && name != workspace.Name {
		nameKey := fmt.Sprintf("%s:%s", workspace.OwnerID, name)
		if existingID, exists := s.nameIndex[nameKey]; exists && existingID != id {
			return storage.ErrAlreadyExists
		}
		
		// 기존 인덱스 삭제
		oldNameKey := fmt.Sprintf("%s:%s", workspace.OwnerID, workspace.Name)
		delete(s.nameIndex, oldNameKey)
		
		// 새 인덱스 추가
		s.nameIndex[nameKey] = id
		workspace.Name = name
	}

	// 다른 필드 업데이트
	if projectPath, ok := updates["project_path"].(string); ok {
		workspace.ProjectPath = projectPath
	}
	if claudeKey, ok := updates["claude_key"].(string); ok {
		workspace.ClaudeKey = claudeKey
	}
	if status, ok := updates["status"].(models.WorkspaceStatus); ok {
		workspace.Status = status
	}

	workspace.UpdatedAt = time.Now()

	return nil
}

// Delete 워크스페이스 삭제 (soft delete)
func (s *WorkspaceStorage) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	workspace, exists := s.workspaces[id]
	if !exists || workspace.DeletedAt != nil {
		return storage.ErrNotFound
	}

	// Soft delete
	now := time.Now()
	workspace.DeletedAt = &now
	workspace.UpdatedAt = now

	// 이름 인덱스에서 제거
	nameKey := fmt.Sprintf("%s:%s", workspace.OwnerID, workspace.Name)
	delete(s.nameIndex, nameKey)

	return nil
}

// List 전체 워크스페이스 목록 조회
func (s *WorkspaceStorage) List(ctx context.Context, pagination *models.PaginationRequest) ([]*models.Workspace, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 삭제되지 않은 워크스페이스만 필터링
	var filtered []*models.Workspace
	for _, ws := range s.workspaces {
		if ws.DeletedAt == nil {
			filtered = append(filtered, ws)
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
		return []*models.Workspace{}, total, nil
	}

	end := offset + limit
	if end > total {
		end = total
	}

	// 복사본 반환
	result := make([]*models.Workspace, 0, end-offset)
	for i := offset; i < end; i++ {
		ws := *filtered[i]
		result = append(result, &ws)
	}

	return result, total, nil
}

// ExistsByName 이름으로 존재 여부 확인
func (s *WorkspaceStorage) ExistsByName(ctx context.Context, ownerID, name string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	nameKey := fmt.Sprintf("%s:%s", ownerID, name)
	workspaceID, exists := s.nameIndex[nameKey]
	if !exists {
		return false, nil
	}

	// 삭제되지 않은 경우만 true
	workspace, ok := s.workspaces[workspaceID]
	return ok && workspace.DeletedAt == nil, nil
}