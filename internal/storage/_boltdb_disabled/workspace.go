package boltdb

import (
	"context"
	"fmt"
	"time"

	"go.etcd.io/bbolt"

	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/storage"
)

// workspaceStorage 워크스페이스 스토리지 BoltDB 구현
type workspaceStorage struct {
	storage    *Storage
	serializer *WorkspaceSerializer
	indexMgr   *IndexManager
	queryMgr   *QueryHelper
}

// newWorkspaceStorage 새 워크스페이스 스토리지 생성
func newWorkspaceStorage(s *Storage) *workspaceStorage {
	indexMgr := newIndexManager(s)
	return &workspaceStorage{
		storage:    s,
		serializer: &WorkspaceSerializer{},
		indexMgr:   indexMgr,
		queryMgr:   newQueryHelper(s, indexMgr),
	}
}

// Create 새 워크스페이스 생성
func (w *workspaceStorage) Create(ctx context.Context, workspace *models.Workspace) error {
	// 검증
	if err := ValidateRequiredFields(workspace); err != nil {
		return err
	}
	
	// 타임스탬프 정규화
	NormalizeTimestamps(workspace)
	
	// 기본값 설정
	if workspace.Status == "" {
		workspace.Status = models.WorkspaceStatusActive
	}
	
	return w.storage.Update(func(tx *bbolt.Tx) error {
		// 중복 검사
		nameKey := fmt.Sprintf("%s:%s", workspace.OwnerID, workspace.Name)
		exists, err := w.indexMgr.ExistsInIndex(tx, IndexWorkspaceName, nameKey)
		if err != nil {
			return fmt.Errorf("중복 검사 실패: %w", err)
		}
		if exists {
			return storage.ErrAlreadyExists
		}
		
		// 직렬화
		data, err := w.serializer.Marshal(workspace)
		if err != nil {
			return fmt.Errorf("워크스페이스 직렬화 실패: %w", err)
		}
		
		// 메인 버킷에 저장
		bucket := tx.Bucket([]byte(BucketWorkspaces))
		if bucket == nil {
			return fmt.Errorf("워크스페이스 버킷이 존재하지 않습니다")
		}
		
		if err := bucket.Put([]byte(workspace.ID), data); err != nil {
			return fmt.Errorf("워크스페이스 저장 실패: %w", err)
		}
		
		// 인덱스 업데이트
		indexUpdates := []IndexUpdate{
			{
				Index:     IndexWorkspaceOwner,
				Operation: IndexOpAdd,
				Key:       workspace.OwnerID,
				Value:     workspace.ID,
			},
			{
				Index:     IndexWorkspaceName,
				Operation: IndexOpAdd,
				Key:       nameKey,
				Value:     workspace.ID,
			},
		}
		
		return w.indexMgr.BatchUpdate(tx, indexUpdates)
	})
}

// GetByID ID로 워크스페이스 조회
func (w *workspaceStorage) GetByID(ctx context.Context, id string) (*models.Workspace, error) {
	var workspace *models.Workspace
	
	err := w.storage.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(BucketWorkspaces))
		if bucket == nil {
			return fmt.Errorf("워크스페이스 버킷이 존재하지 않습니다")
		}
		
		data := bucket.Get([]byte(id))
		if data == nil {
			return storage.ErrNotFound
		}
		
		ws, err := w.serializer.Unmarshal(data)
		if err != nil {
			return fmt.Errorf("워크스페이스 역직렬화 실패: %w", err)
		}
		
		// Soft delete 체크
		if ws.DeletedAt != nil {
			return storage.ErrNotFound
		}
		
		workspace = ws
		return nil
	})
	
	return workspace, err
}

// GetByOwnerID 소유자 ID로 워크스페이스 목록 조회
func (w *workspaceStorage) GetByOwnerID(ctx context.Context, ownerID string, pagination *models.PaginationRequest) ([]*models.Workspace, int, error) {
	if pagination == nil {
		pagination = &models.PaginationRequest{Page: 1, Limit: 20, Sort: "created_at", Order: "desc"}
	}
	pagination.Normalize()
	
	var workspaces []*models.Workspace
	var total int
	
	err := w.storage.View(func(tx *bbolt.Tx) error {
		options := &QueryOptions{
			Page:  pagination.Page,
			Limit: pagination.Limit,
			Sort:  pagination.Sort,
			Order: SortOrder(pagination.Order),
			Filter: map[string]interface{}{
				"owner_id": ownerID,
			},
		}
		
		result, err := w.queryMgr.WorkspaceQuery(tx, options)
		if err != nil {
			return err
		}
		
		workspaces = result.Items
		total = result.TotalCount
		return nil
	})
	
	return workspaces, total, err
}

// Update 워크스페이스 업데이트
func (w *workspaceStorage) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return storage.ErrInvalidInput
	}
	
	return w.storage.Update(func(tx *bbolt.Tx) error {
		// 기존 워크스페이스 조회
		bucket := tx.Bucket([]byte(BucketWorkspaces))
		if bucket == nil {
			return fmt.Errorf("워크스페이스 버킷이 존재하지 않습니다")
		}
		
		data := bucket.Get([]byte(id))
		if data == nil {
			return storage.ErrNotFound
		}
		
		workspace, err := w.serializer.Unmarshal(data)
		if err != nil {
			return fmt.Errorf("워크스페이스 역직렬화 실패: %w", err)
		}
		
		// Soft delete 체크
		if workspace.DeletedAt != nil {
			return storage.ErrNotFound
		}
		
		// 기존 값들 저장 (인덱스 업데이트용)
		oldName := workspace.Name
		oldStatus := workspace.Status
		
		// 업데이트 적용
		for field, value := range updates {
			switch field {
			case "name":
				if name, ok := value.(string); ok {
					workspace.Name = name
				}
			case "project_path":
				if path, ok := value.(string); ok {
					workspace.ProjectPath = path
				}
			case "status":
				if status, ok := value.(models.WorkspaceStatus); ok {
					workspace.Status = status
				}
			case "claude_key":
				if key, ok := value.(string); ok {
					workspace.ClaudeKey = key
				}
			case "active_tasks":
				if count, ok := value.(int); ok {
					workspace.ActiveTasks = count
				}
			default:
				return fmt.Errorf("허용되지 않은 필드: %s", field)
			}
		}
		
		// 타임스탬프 업데이트
		workspace.UpdatedAt = time.Now()
		
		// 이름이 변경된 경우 중복 검사
		if workspace.Name != oldName {
			nameKey := fmt.Sprintf("%s:%s", workspace.OwnerID, workspace.Name)
			exists, err := w.indexMgr.ExistsInIndex(tx, IndexWorkspaceName, nameKey)
			if err != nil {
				return fmt.Errorf("중복 검사 실패: %w", err)
			}
			if exists {
				return storage.ErrAlreadyExists
			}
		}
		
		// 직렬화 및 저장
		newData, err := w.serializer.Marshal(workspace)
		if err != nil {
			return fmt.Errorf("워크스페이스 직렬화 실패: %w", err)
		}
		
		if err := bucket.Put([]byte(id), newData); err != nil {
			return fmt.Errorf("워크스페이스 업데이트 실패: %w", err)
		}
		
		// 인덱스 업데이트
		var indexUpdates []IndexUpdate
		
		// 이름이 변경된 경우 이름 인덱스 업데이트
		if workspace.Name != oldName {
			oldNameKey := fmt.Sprintf("%s:%s", workspace.OwnerID, oldName)
			newNameKey := fmt.Sprintf("%s:%s", workspace.OwnerID, workspace.Name)
			
			indexUpdates = append(indexUpdates, IndexUpdate{
				Index:     IndexWorkspaceName,
				Operation: IndexOpUpdate,
				Key:       oldNameKey,
				OldValue:  id,
				Value:     id,
			})
			
			// 실제로는 키 자체가 변경되므로 삭제 후 추가
			indexUpdates = append(indexUpdates, 
				IndexUpdate{
					Index:     IndexWorkspaceName,
					Operation: IndexOpRemove,
					Key:       oldNameKey,
					Value:     id,
				},
				IndexUpdate{
					Index:     IndexWorkspaceName,
					Operation: IndexOpAdd,
					Key:       newNameKey,
					Value:     id,
				},
			)
		}
		
		// 상태 인덱스 업데이트 (필요한 경우)
		if workspace.Status != oldStatus {
			statusKey := fmt.Sprintf("workspace:%s", oldStatus)
			newStatusKey := fmt.Sprintf("workspace:%s", workspace.Status)
			
			indexUpdates = append(indexUpdates,
				IndexUpdate{
					Index:     IndexEntityStatus,
					Operation: IndexOpRemove,
					Key:       statusKey,
					Value:     id,
				},
				IndexUpdate{
					Index:     IndexEntityStatus,
					Operation: IndexOpAdd,
					Key:       newStatusKey,
					Value:     id,
				},
			)
		}
		
		if len(indexUpdates) > 0 {
			return w.indexMgr.BatchUpdate(tx, indexUpdates)
		}
		
		return nil
	})
}

// Delete 워크스페이스 삭제 (soft delete)
func (w *workspaceStorage) Delete(ctx context.Context, id string) error {
	return w.storage.Update(func(tx *bbolt.Tx) error {
		// 기존 워크스페이스 조회
		bucket := tx.Bucket([]byte(BucketWorkspaces))
		if bucket == nil {
			return fmt.Errorf("워크스페이스 버킷이 존재하지 않습니다")
		}
		
		data := bucket.Get([]byte(id))
		if data == nil {
			return storage.ErrNotFound
		}
		
		workspace, err := w.serializer.Unmarshal(data)
		if err != nil {
			return fmt.Errorf("워크스페이스 역직렬화 실패: %w", err)
		}
		
		// 이미 삭제된 경우
		if workspace.DeletedAt != nil {
			return storage.ErrNotFound
		}
		
		// Soft delete
		now := time.Now()
		workspace.DeletedAt = &now
		workspace.UpdatedAt = now
		
		// 직렬화 및 저장
		newData, err := w.serializer.Marshal(workspace)
		if err != nil {
			return fmt.Errorf("워크스페이스 직렬화 실패: %w", err)
		}
		
		if err := bucket.Put([]byte(id), newData); err != nil {
			return fmt.Errorf("워크스페이스 삭제 실패: %w", err)
		}
		
		// 인덱스에서 제거
		nameKey := fmt.Sprintf("%s:%s", workspace.OwnerID, workspace.Name)
		statusKey := fmt.Sprintf("workspace:%s", workspace.Status)
		
		indexUpdates := []IndexUpdate{
			{
				Index:     IndexWorkspaceOwner,
				Operation: IndexOpRemove,
				Key:       workspace.OwnerID,
				Value:     id,
			},
			{
				Index:     IndexWorkspaceName,
				Operation: IndexOpRemove,
				Key:       nameKey,
				Value:     id,
			},
			{
				Index:     IndexEntityStatus,
				Operation: IndexOpRemove,
				Key:       statusKey,
				Value:     id,
			},
		}
		
		return w.indexMgr.BatchUpdate(tx, indexUpdates)
	})
}

// List 전체 워크스페이스 목록 조회 (관리자용)
func (w *workspaceStorage) List(ctx context.Context, pagination *models.PaginationRequest) ([]*models.Workspace, int, error) {
	if pagination == nil {
		pagination = &models.PaginationRequest{Page: 1, Limit: 20, Sort: "created_at", Order: "desc"}
	}
	pagination.Normalize()
	
	var workspaces []*models.Workspace
	var total int
	
	err := w.storage.View(func(tx *bbolt.Tx) error {
		options := &QueryOptions{
			Page:   pagination.Page,
			Limit:  pagination.Limit,
			Sort:   pagination.Sort,
			Order:  SortOrder(pagination.Order),
			Filter: map[string]interface{}{}, // 전체 조회
		}
		
		result, err := w.queryMgr.WorkspaceQuery(tx, options)
		if err != nil {
			return err
		}
		
		workspaces = result.Items
		total = result.TotalCount
		return nil
	})
	
	return workspaces, total, err
}

// ExistsByName 이름으로 존재 여부 확인
func (w *workspaceStorage) ExistsByName(ctx context.Context, ownerID, name string) (bool, error) {
	var exists bool
	
	err := w.storage.View(func(tx *bbolt.Tx) error {
		nameKey := fmt.Sprintf("%s:%s", ownerID, name)
		
		found, err := w.indexMgr.ExistsInIndex(tx, IndexWorkspaceName, nameKey)
		if err != nil {
			return err
		}
		
		exists = found
		return nil
	})
	
	return exists, err
}

// GetStats 워크스페이스 통계 조회
func (w *workspaceStorage) GetStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	
	err := w.storage.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(BucketWorkspaces))
		if bucket == nil {
			return fmt.Errorf("워크스페이스 버킷이 존재하지 않습니다")
		}
		
		var total, active, inactive, archived int
		
		cursor := bucket.Cursor()
		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			workspace, err := w.serializer.Unmarshal(v)
			if err != nil {
				continue
			}
			
			// Soft delete 체크
			if workspace.DeletedAt != nil {
				continue
			}
			
			total++
			
			switch workspace.Status {
			case models.WorkspaceStatusActive:
				active++
			case models.WorkspaceStatusInactive:
				inactive++
			case models.WorkspaceStatusArchived:
				archived++
			}
		}
		
		stats["total"] = total
		stats["active"] = active
		stats["inactive"] = inactive
		stats["archived"] = archived
		
		return nil
	})
	
	return stats, err
}

// Search 워크스페이스 검색
func (w *workspaceStorage) Search(ctx context.Context, query string, limit int) ([]*models.Workspace, error) {
	if limit <= 0 {
		limit = 50
	}
	
	var workspaces []*models.Workspace
	
	err := w.storage.View(func(tx *bbolt.Tx) error {
		keys, err := w.queryMgr.SearchByText(tx, BucketWorkspaces, query, limit)
		if err != nil {
			return err
		}
		
		bucket := tx.Bucket([]byte(BucketWorkspaces))
		
		for _, key := range keys {
			data := bucket.Get([]byte(key))
			if data == nil {
				continue
			}
			
			workspace, err := w.serializer.Unmarshal(data)
			if err != nil {
				continue
			}
			
			// Soft delete 체크
			if workspace.DeletedAt != nil {
				continue
			}
			
			workspaces = append(workspaces, workspace)
		}
		
		return nil
	})
	
	return workspaces, err
}

// GetByOwners 여러 소유자의 워크스페이스 조회
func (w *workspaceStorage) GetByOwners(ctx context.Context, ownerIDs []string) ([]*models.Workspace, error) {
	var workspaces []*models.Workspace
	
	err := w.storage.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(BucketWorkspaces))
		if bucket == nil {
			return fmt.Errorf("워크스페이스 버킷이 존재하지 않습니다")
		}
		
		for _, ownerID := range ownerIDs {
			workspaceIDs, err := w.indexMgr.GetFromIndex(tx, IndexWorkspaceOwner, ownerID)
			if err != nil {
				continue
			}
			
			for _, workspaceID := range workspaceIDs {
				data := bucket.Get([]byte(workspaceID))
				if data == nil {
					continue
				}
				
				workspace, err := w.serializer.Unmarshal(data)
				if err != nil {
					continue
				}
				
				// Soft delete 체크
				if workspace.DeletedAt != nil {
					continue
				}
				
				workspaces = append(workspaces, workspace)
			}
		}
		
		return nil
	})
	
	return workspaces, err
}

// CountByOwner 소유자별 워크스페이스 개수 조회
func (w *workspaceStorage) CountByOwner(ctx context.Context, ownerID string) (int, error) {
	var count int
	
	err := w.storage.View(func(tx *bbolt.Tx) error {
		workspaceIDs, err := w.indexMgr.GetFromIndex(tx, IndexWorkspaceOwner, ownerID)
		if err != nil {
			return err
		}
		
		bucket := tx.Bucket([]byte(BucketWorkspaces))
		if bucket == nil {
			return fmt.Errorf("워크스페이스 버킷이 존재하지 않습니다")
		}
		
		// 삭제되지 않은 워크스페이스만 카운트
		for _, workspaceID := range workspaceIDs {
			data := bucket.Get([]byte(workspaceID))
			if data == nil {
				continue
			}
			
			workspace, err := w.serializer.Unmarshal(data)
			if err != nil {
				continue
			}
			
			// Soft delete 체크
			if workspace.DeletedAt == nil {
				count++
			}
		}
		
		return nil
	})
	
	return count, err
}

// GetByName 이름으로 워크스페이스 조회
func (w *workspaceStorage) GetByName(ctx context.Context, ownerID, name string) (*models.Workspace, error) {
	var workspace *models.Workspace
	
	err := w.storage.View(func(tx *bbolt.Tx) error {
		nameKey := fmt.Sprintf("%s:%s", ownerID, name)
		
		// 인덱스에서 워크스페이스 ID 조회
		workspaceIDs, err := w.indexMgr.GetFromIndex(tx, IndexWorkspaceName, nameKey)
		if err != nil {
			return err
		}
		
		if len(workspaceIDs) == 0 {
			return storage.ErrNotFound
		}
		
		// 첫 번째 매치 (이름은 고유해야 함)
		bucket := tx.Bucket([]byte(BucketWorkspaces))
		if bucket == nil {
			return fmt.Errorf("워크스페이스 버킷이 존재하지 않습니다")
		}
		
		data := bucket.Get([]byte(workspaceIDs[0]))
		if data == nil {
			return storage.ErrNotFound
		}
		
		ws, err := w.serializer.Unmarshal(data)
		if err != nil {
			return fmt.Errorf("워크스페이스 역직렬화 실패: %w", err)
		}
		
		// Soft delete 체크
		if ws.DeletedAt != nil {
			return storage.ErrNotFound
		}
		
		workspace = ws
		return nil
	})
	
	if err != nil {
		return nil, err
	}
	
	return workspace, nil
}