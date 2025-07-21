package memory

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/storage"
	"github.com/google/uuid"
)

// taskStorage 메모리 기반 태스크 스토리지
type taskStorage struct {
	tasks map[string]*models.Task
	mutex sync.RWMutex
}

// newTaskStorage 새 태스크 스토리지 생성
func newTaskStorage() *taskStorage {
	return &taskStorage{
		tasks: make(map[string]*models.Task),
	}
}

// Create 새 태스크 생성
func (ts *taskStorage) Create(ctx context.Context, task *models.Task) error {
	ts.mutex.Lock()
	defer ts.mutex.Unlock()
	
	// ID 생성
	if task.ID == "" {
		task.ID = uuid.New().String()
	}
	
	// 기본값 설정
	now := time.Now()
	if task.CreatedAt.IsZero() {
		task.CreatedAt = now
	}
	task.UpdatedAt = now
	
	// 상태 기본값
	if task.Status == "" {
		task.Status = models.TaskPending
	}
	
	// 중복 확인
	if _, exists := ts.tasks[task.ID]; exists {
		return storage.ErrAlreadyExists
	}
	
	// 복사본 저장
	taskCopy := *task
	ts.tasks[task.ID] = &taskCopy
	
	return nil
}

// GetByID ID로 태스크 조회
func (ts *taskStorage) GetByID(ctx context.Context, id string) (*models.Task, error) {
	ts.mutex.RLock()
	defer ts.mutex.RUnlock()
	
	task, exists := ts.tasks[id]
	if !exists {
		return nil, storage.ErrNotFound
	}
	
	// 복사본 반환
	taskCopy := *task
	return &taskCopy, nil
}

// List 태스크 목록 조회
func (ts *taskStorage) List(ctx context.Context, filter *models.TaskFilter, paging *models.PagingRequest) ([]*models.Task, int, error) {
	ts.mutex.RLock()
	defer ts.mutex.RUnlock()
	
	// 모든 태스크 수집
	var allTasks []*models.Task
	for _, task := range ts.tasks {
		// 삭제되지 않은 태스크만
		if !task.DeletedAt.Valid {
			allTasks = append(allTasks, task)
		}
	}
	
	// 필터링 적용
	if filter != nil {
		allTasks = ts.applyFilter(allTasks, filter)
	}
	
	// 정렬 (최신순)
	sort.Slice(allTasks, func(i, j int) bool {
		return allTasks[i].CreatedAt.After(allTasks[j].CreatedAt)
	})
	
	total := len(allTasks)
	
	// 페이징 적용
	if paging != nil {
		start := (paging.Page - 1) * paging.Limit
		end := start + paging.Limit
		
		if start >= total {
			return []*models.Task{}, total, nil
		}
		
		if end > total {
			end = total
		}
		
		allTasks = allTasks[start:end]
	}
	
	// 복사본 반환
	result := make([]*models.Task, len(allTasks))
	for i, task := range allTasks {
		taskCopy := *task
		result[i] = &taskCopy
	}
	
	return result, total, nil
}

// Update 태스크 업데이트
func (ts *taskStorage) Update(ctx context.Context, task *models.Task) error {
	ts.mutex.Lock()
	defer ts.mutex.Unlock()
	
	existing, exists := ts.tasks[task.ID]
	if !exists {
		return storage.ErrNotFound
	}
	
	// 삭제된 리소스 확인
	if existing.DeletedAt.Valid {
		return storage.ErrNotFound
	}
	
	// 업데이트 시간 설정
	task.UpdatedAt = time.Now()
	
	// 복사본 저장
	taskCopy := *task
	ts.tasks[task.ID] = &taskCopy
	
	return nil
}

// Delete 태스크 삭제
func (ts *taskStorage) Delete(ctx context.Context, id string) error {
	ts.mutex.Lock()
	defer ts.mutex.Unlock()
	
	task, exists := ts.tasks[id]
	if !exists {
		return storage.ErrNotFound
	}
	
	// 이미 삭제된 경우
	if task.DeletedAt.Valid {
		return storage.ErrNotFound
	}
	
	// Soft delete
	now := time.Now()
	task.DeletedAt.Time = now
	task.DeletedAt.Valid = true
	task.UpdatedAt = now
	
	return nil
}

// GetBySessionID 세션 ID로 태스크 목록 조회
func (ts *taskStorage) GetBySessionID(ctx context.Context, sessionID string, paging *models.PagingRequest) ([]*models.Task, int, error) {
	filter := &models.TaskFilter{
		SessionID: &sessionID,
	}
	
	return ts.List(ctx, filter, paging)
}

// GetActiveCount 활성 태스크 수 조회
func (ts *taskStorage) GetActiveCount(ctx context.Context, sessionID string) (int64, error) {
	ts.mutex.RLock()
	defer ts.mutex.RUnlock()
	
	count := int64(0)
	for _, task := range ts.tasks {
		if !task.DeletedAt.Valid && (sessionID == "" || task.SessionID == sessionID) && task.IsActive() {
			count++
		}
	}
	
	return count, nil
}

// applyFilter 필터 적용
func (ts *taskStorage) applyFilter(tasks []*models.Task, filter *models.TaskFilter) []*models.Task {
	var filtered []*models.Task
	
	for _, task := range tasks {
		// 세션 ID 필터
		if filter.SessionID != nil && task.SessionID != *filter.SessionID {
			continue
		}
		
		// 상태 필터
		if filter.Status != nil && task.Status != *filter.Status {
			continue
		}
		
		// 활성 태스크 필터
		if filter.Active != nil {
			if *filter.Active && !task.IsActive() {
				continue
			}
			if !*filter.Active && task.IsActive() {
				continue
			}
		}
		
		filtered = append(filtered, task)
	}
	
	return filtered
}

// searchTasks 태스크 검색 (확장용)
func (ts *taskStorage) searchTasks(keyword string) []*models.Task {
	ts.mutex.RLock()
	defer ts.mutex.RUnlock()
	
	var results []*models.Task
	keyword = strings.ToLower(keyword)
	
	for _, task := range ts.tasks {
		if !task.DeletedAt.Valid {
			// 명령어에서 검색
			if strings.Contains(strings.ToLower(task.Command), keyword) {
				results = append(results, task)
				continue
			}
			
			// 출력에서 검색
			if strings.Contains(strings.ToLower(task.Output), keyword) {
				results = append(results, task)
				continue
			}
			
			// 에러에서 검색
			if strings.Contains(strings.ToLower(task.Error), keyword) {
				results = append(results, task)
				continue
			}
		}
	}
	
	return results
}

// getTasksByTimeRange 시간 범위로 태스크 조회 (확장용)
func (ts *taskStorage) getTasksByTimeRange(start, end time.Time) []*models.Task {
	ts.mutex.RLock()
	defer ts.mutex.RUnlock()
	
	var results []*models.Task
	
	for _, task := range ts.tasks {
		if !task.DeletedAt.Valid && 
		   task.CreatedAt.After(start) && 
		   task.CreatedAt.Before(end) {
			results = append(results, task)
		}
	}
	
	return results
}

// getStats 태스크 통계 조회 (확장용)
func (ts *taskStorage) getStats() map[string]interface{} {
	ts.mutex.RLock()
	defer ts.mutex.RUnlock()
	
	stats := map[string]interface{}{
		"total": 0,
		"by_status": make(map[models.TaskStatus]int),
		"active_count": 0,
	}
	
	for _, task := range ts.tasks {
		if !task.DeletedAt.Valid {
			stats["total"] = stats["total"].(int) + 1
			
			// 상태별 카운트
			statusMap := stats["by_status"].(map[models.TaskStatus]int)
			statusMap[task.Status]++
			
			// 활성 태스크 카운트
			if task.IsActive() {
				stats["active_count"] = stats["active_count"].(int) + 1
			}
		}
	}
	
	return stats
}

// cleanup 완료된 태스크 정리 (확장용)
func (ts *taskStorage) cleanup(maxAge time.Duration) int {
	ts.mutex.Lock()
	defer ts.mutex.Unlock()
	
	cutoff := time.Now().Add(-maxAge)
	var toDelete []string
	
	for id, task := range ts.tasks {
		if task.IsTerminal() && 
		   task.CompletedAt != nil && 
		   task.CompletedAt.Before(cutoff) {
			toDelete = append(toDelete, id)
		}
	}
	
	for _, id := range toDelete {
		delete(ts.tasks, id)
	}
	
	return len(toDelete)
}