package boltdb

import (
	"context"
	"fmt"
	"sort"
	"time"

	"go.etcd.io/bbolt"

	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/storage/interfaces"
)

// taskStorage BoltDB Task 스토리지 구현
type taskStorage struct {
	db         *bbolt.DB
	serializer *Serializer
	indexer    *IndexManager
	querier    *QueryHelper
}

// newTaskStorage 태스크 스토리지 생성자
func newTaskStorage(storage *Storage) *taskStorage {
	return &taskStorage{
		db:         storage.db,
		serializer: storage.serializer,
		indexer:    storage.indexer,
		querier:    storage.querier,
	}
}

// Create 새 태스크 생성
func (ts *taskStorage) Create(ctx context.Context, task *models.Task) error {
	return ts.db.Update(func(tx *bbolt.Tx) error {
		// 태스크 버킷 가져오기
		taskBucket := tx.Bucket([]byte(bucketTasks))
		if taskBucket == nil {
			return fmt.Errorf("tasks bucket not found")
		}
		
		// ID 중복 체크
		if existing := taskBucket.Get([]byte(task.ID)); existing != nil {
			return storage.ErrAlreadyExists
		}
		
		// 세션 존재 확인
		sessionBucket := tx.Bucket([]byte(bucketSessions))
		if sessionBucket == nil {
			return fmt.Errorf("sessions bucket not found")
		}
		
		sessionData := sessionBucket.Get([]byte(task.SessionID))
		if sessionData == nil {
			return storage.ErrNotFound
		}
		
		// 세션이 활성 상태인지 확인
		var session models.Session
		if err := ts.serializer.UnmarshalSession(sessionData, &session); err != nil {
			return err
		}
		if session.Status == models.SessionStatusEnded {
			return fmt.Errorf("cannot create task in ended session")
		}
		
		// 타임스탬프 설정
		now := time.Now()
		if task.CreatedAt.IsZero() {
			task.CreatedAt = now
		}
		task.UpdatedAt = now
		if task.Version == 0 {
			task.Version = 1
		}
		
		// 태스크 직렬화 및 저장
		data, err := ts.serializer.MarshalTask(task)
		if err != nil {
			return err
		}
		
		if err := taskBucket.Put([]byte(task.ID), data); err != nil {
			return err
		}
		
		// 인덱스 업데이트
		if err := ts.indexer.AddToIndex(tx, IndexSessionTasks, task.SessionID, task.ID); err != nil {
			return err
		}
		
		if err := ts.indexer.AddToIndex(tx, IndexTaskStatus, string(task.Status), task.ID); err != nil {
			return err
		}
		
		// 명령어 인덱스 (부분 검색을 위해)
		if task.Command != "" {
			commandWords := ts.querier.TokenizeCommand(task.Command)
			for _, word := range commandWords {
				if err := ts.indexer.AddToIndex(tx, IndexTaskCommand, word, task.ID); err != nil {
					return err
				}
			}
		}
		
		return nil
	})
}

// GetByID ID로 태스크 조회
func (ts *taskStorage) GetByID(ctx context.Context, id string) (*models.Task, error) {
	var task models.Task
	
	err := ts.db.View(func(tx *bbolt.Tx) error {
		taskBucket := tx.Bucket([]byte(bucketTasks))
		if taskBucket == nil {
			return fmt.Errorf("tasks bucket not found")
		}
		
		data := taskBucket.Get([]byte(id))
		if data == nil {
			return storage.ErrNotFound
		}
		
		if err := ts.serializer.UnmarshalTask(data, &task); err != nil {
			return err
		}
		
		return nil
	})
	
	if err != nil {
		return nil, err
	}
	
	return &task, nil
}

// GetBySessionID 세션별 태스크 목록 조회
func (ts *taskStorage) GetBySessionID(ctx context.Context, sessionID string, pagination *models.PaginationRequest) ([]*models.Task, int64, error) {
	var tasks []*models.Task
	var total int64
	
	err := ts.db.View(func(tx *bbolt.Tx) error {
		// 인덱스에서 태스크 ID 목록 가져오기
		taskIDs, err := ts.indexer.GetFromIndex(tx, IndexSessionTasks, sessionID)
		if err != nil {
			return err
		}
		
		// 태스크 데이터 조회
		var allTasks []*models.Task
		taskBucket := tx.Bucket([]byte(bucketTasks))
		if taskBucket == nil {
			return fmt.Errorf("tasks bucket not found")
		}
		
		for _, taskID := range taskIDs {
			data := taskBucket.Get([]byte(taskID))
			if data == nil {
				continue
			}
			
			var task models.Task
			if err := ts.serializer.UnmarshalTask(data, &task); err != nil {
				continue
			}
			
			allTasks = append(allTasks, &task)
		}
		
		// 생성 시간 역순 정렬
		ts.querier.SortTasks(allTasks, "created_at", false)
		
		total = int64(len(allTasks))
		
		// 페이지네이션 적용
		start, end := ts.querier.CalculatePagination(len(allTasks), pagination)
		tasks = allTasks[start:end]
		
		return nil
	})
	
	if err != nil {
		return nil, 0, err
	}
	
	return tasks, total, nil
}

// GetByStatus 상태별 태스크 조회
func (ts *taskStorage) GetByStatus(ctx context.Context, status models.TaskStatus, pagination *models.PaginationRequest) ([]*models.Task, int64, error) {
	var tasks []*models.Task
	var total int64
	
	err := ts.db.View(func(tx *bbolt.Tx) error {
		// 인덱스에서 태스크 ID 목록 가져오기
		taskIDs, err := ts.indexer.GetFromIndex(tx, IndexTaskStatus, string(status))
		if err != nil {
			return err
		}
		
		// 태스크 데이터 조회
		var allTasks []*models.Task
		taskBucket := tx.Bucket([]byte(bucketTasks))
		if taskBucket == nil {
			return fmt.Errorf("tasks bucket not found")
		}
		
		for _, taskID := range taskIDs {
			data := taskBucket.Get([]byte(taskID))
			if data == nil {
				continue
			}
			
			var task models.Task
			if err := ts.serializer.UnmarshalTask(data, &task); err != nil {
				continue
			}
			
			allTasks = append(allTasks, &task)
		}
		
		// 시작 시간 역순 정렬 (최신 태스크가 먼저)
		ts.querier.SortTasks(allTasks, "started_at", false)
		
		total = int64(len(allTasks))
		
		// 페이지네이션 적용
		start, end := ts.querier.CalculatePagination(len(allTasks), pagination)
		tasks = allTasks[start:end]
		
		return nil
	})
	
	if err != nil {
		return nil, 0, err
	}
	
	return tasks, total, nil
}

// GetActiveCount 세션의 활성 태스크 수 조회
func (ts *taskStorage) GetActiveCount(ctx context.Context, sessionID string) (int64, error) {
	var count int64
	
	err := ts.db.View(func(tx *bbolt.Tx) error {
		// 세션의 모든 태스크 조회
		taskIDs, err := ts.indexer.GetFromIndex(tx, IndexSessionTasks, sessionID)
		if err != nil {
			return err
		}
		
		taskBucket := tx.Bucket([]byte(bucketTasks))
		if taskBucket == nil {
			return fmt.Errorf("tasks bucket not found")
		}
		
		for _, taskID := range taskIDs {
			data := taskBucket.Get([]byte(taskID))
			if data == nil {
				continue
			}
			
			var task models.Task
			if err := ts.serializer.UnmarshalTask(data, &task); err != nil {
				continue
			}
			
			if task.Status == models.TaskStatusRunning || task.Status == models.TaskStatusPending {
				count++
			}
		}
		
		return nil
	})
	
	if err != nil {
		return 0, err
	}
	
	return count, nil
}

// SearchByCommand 명령어로 태스크 검색
func (ts *taskStorage) SearchByCommand(ctx context.Context, query string, pagination *models.PaginationRequest) ([]*models.Task, int64, error) {
	var tasks []*models.Task
	var total int64
	
	err := ts.db.View(func(tx *bbolt.Tx) error {
		// 쿼리 토큰화
		queryWords := ts.querier.TokenizeCommand(query)
		if len(queryWords) == 0 {
			return nil
		}
		
		// 각 단어별로 태스크 ID 수집
		taskIDSets := make([]map[string]bool, len(queryWords))
		for i, word := range queryWords {
			taskIDs, err := ts.indexer.GetFromIndex(tx, IndexTaskCommand, word)
			if err != nil {
				return err
			}
			
			taskIDSets[i] = make(map[string]bool)
			for _, taskID := range taskIDs {
				taskIDSets[i][taskID] = true
			}
		}
		
		// 교집합 계산 (모든 단어를 포함하는 태스크)
		var matchingTaskIDs []string
		if len(taskIDSets) > 0 {
			for taskID := range taskIDSets[0] {
				found := true
				for i := 1; i < len(taskIDSets); i++ {
					if !taskIDSets[i][taskID] {
						found = false
						break
					}
				}
				if found {
					matchingTaskIDs = append(matchingTaskIDs, taskID)
				}
			}
		}
		
		// 매칭된 태스크 데이터 조회
		var allTasks []*models.Task
		taskBucket := tx.Bucket([]byte(bucketTasks))
		if taskBucket == nil {
			return fmt.Errorf("tasks bucket not found")
		}
		
		for _, taskID := range matchingTaskIDs {
			data := taskBucket.Get([]byte(taskID))
			if data == nil {
				continue
			}
			
			var task models.Task
			if err := ts.serializer.UnmarshalTask(data, &task); err != nil {
				continue
			}
			
			allTasks = append(allTasks, &task)
		}
		
		// 관련성 순으로 정렬 (정확한 매치가 우선)
		ts.sortTasksByRelevance(allTasks, query)
		
		total = int64(len(allTasks))
		
		// 페이지네이션 적용
		start, end := ts.querier.CalculatePagination(len(allTasks), pagination)
		tasks = allTasks[start:end]
		
		return nil
	})
	
	if err != nil {
		return nil, 0, err
	}
	
	return tasks, total, nil
}

// Update 태스크 정보 업데이트
func (ts *taskStorage) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	return ts.db.Update(func(tx *bbolt.Tx) error {
		taskBucket := tx.Bucket([]byte(bucketTasks))
		if taskBucket == nil {
			return fmt.Errorf("tasks bucket not found")
		}
		
		// 기존 태스크 조회
		data := taskBucket.Get([]byte(id))
		if data == nil {
			return storage.ErrNotFound
		}
		
		var task models.Task
		if err := ts.serializer.UnmarshalTask(data, &task); err != nil {
			return err
		}
		
		// 인덱스 업데이트를 위한 이전 값 저장
		oldStatus := string(task.Status)
		oldCommand := task.Command
		
		// 업데이트 적용
		if status, ok := updates["status"]; ok {
			if statusVal, ok := status.(models.TaskStatus); ok {
				task.Status = statusVal
			} else if statusStr, ok := status.(string); ok {
				task.Status = models.TaskStatus(statusStr)
			}
		}
		
		if command, ok := updates["command"]; ok {
			if cmdStr, ok := command.(string); ok {
				task.Command = cmdStr
			}
		}
		
		if startedAt, ok := updates["started_at"]; ok {
			if startedTime, ok := startedAt.(*time.Time); ok {
				task.StartedAt = startedTime
			}
		}
		
		if completedAt, ok := updates["completed_at"]; ok {
			if completedTime, ok := completedAt.(*time.Time); ok {
				task.CompletedAt = completedTime
			}
		}
		
		if duration, ok := updates["duration"]; ok {
			if durationVal, ok := duration.(int64); ok {
				task.Duration = durationVal
			}
		}
		
		if output, ok := updates["output"]; ok {
			if outputStr, ok := output.(string); ok {
				task.Output = outputStr
			}
		}
		
		if errorMsg, ok := updates["error"]; ok {
			if errorStr, ok := errorMsg.(string); ok {
				task.Error = errorStr
			}
		}
		
		// 타임스탬프 및 버전 업데이트
		task.UpdatedAt = time.Now()
		task.Version++
		
		// 직렬화 및 저장
		updatedData, err := ts.serializer.MarshalTask(&task)
		if err != nil {
			return err
		}
		
		if err := taskBucket.Put([]byte(id), updatedData); err != nil {
			return err
		}
		
		// 인덱스 업데이트
		if string(task.Status) != oldStatus {
			ts.indexer.RemoveFromIndex(tx, IndexTaskStatus, oldStatus, id)
			ts.indexer.AddToIndex(tx, IndexTaskStatus, string(task.Status), id)
		}
		
		if task.Command != oldCommand {
			// 이전 명령어 인덱스 제거
			if oldCommand != "" {
				oldWords := ts.querier.TokenizeCommand(oldCommand)
				for _, word := range oldWords {
					ts.indexer.RemoveFromIndex(tx, IndexTaskCommand, word, id)
				}
			}
			
			// 새 명령어 인덱스 추가
			if task.Command != "" {
				newWords := ts.querier.TokenizeCommand(task.Command)
				for _, word := range newWords {
					ts.indexer.AddToIndex(tx, IndexTaskCommand, word, id)
				}
			}
		}
		
		return nil
	})
}

// Delete 태스크 삭제
func (ts *taskStorage) Delete(ctx context.Context, id string) error {
	return ts.db.Update(func(tx *bbolt.Tx) error {
		taskBucket := tx.Bucket([]byte(bucketTasks))
		if taskBucket == nil {
			return fmt.Errorf("tasks bucket not found")
		}
		
		// 기존 태스크 조회
		data := taskBucket.Get([]byte(id))
		if data == nil {
			return storage.ErrNotFound
		}
		
		var task models.Task
		if err := ts.serializer.UnmarshalTask(data, &task); err != nil {
			return err
		}
		
		// 실행 중인 태스크는 삭제 불가
		if task.Status == models.TaskStatusRunning {
			return fmt.Errorf("cannot delete running task")
		}
		
		// 인덱스에서 제거
		ts.indexer.RemoveFromIndex(tx, IndexSessionTasks, task.SessionID, id)
		ts.indexer.RemoveFromIndex(tx, IndexTaskStatus, string(task.Status), id)
		
		if task.Command != "" {
			commandWords := ts.querier.TokenizeCommand(task.Command)
			for _, word := range commandWords {
				ts.indexer.RemoveFromIndex(tx, IndexTaskCommand, word, id)
			}
		}
		
		// 태스크 삭제
		return taskBucket.Delete([]byte(id))
	})
}

// GetLongRunningTasks 장시간 실행 중인 태스크 조회
func (ts *taskStorage) GetLongRunningTasks(ctx context.Context, threshold time.Duration) ([]*models.Task, error) {
	var tasks []*models.Task
	cutoff := time.Now().Add(-threshold)
	
	err := ts.db.View(func(tx *bbolt.Tx) error {
		// 실행 중인 태스크 조회
		taskIDs, err := ts.indexer.GetFromIndex(tx, IndexTaskStatus, string(models.TaskStatusRunning))
		if err != nil {
			return err
		}
		
		taskBucket := tx.Bucket([]byte(bucketTasks))
		if taskBucket == nil {
			return fmt.Errorf("tasks bucket not found")
		}
		
		for _, taskID := range taskIDs {
			data := taskBucket.Get([]byte(taskID))
			if data == nil {
				continue
			}
			
			var task models.Task
			if err := ts.serializer.UnmarshalTask(data, &task); err != nil {
				continue
			}
			
			// 시작 시간이 임계값보다 오래된 태스크
			if task.StartedAt != nil && task.StartedAt.Before(cutoff) {
				tasks = append(tasks, &task)
			}
		}
		
		return nil
	})
	
	if err != nil {
		return nil, err
	}
	
	// 시작 시간 순으로 정렬 (오래 실행된 것부터)
	sort.Slice(tasks, func(i, j int) bool {
		if tasks[i].StartedAt == nil || tasks[j].StartedAt == nil {
			return false
		}
		return tasks[i].StartedAt.Before(*tasks[j].StartedAt)
	})
	
	return tasks, nil
}

// GetTaskStats 태스크 통계 조회
func (ts *taskStorage) GetTaskStats(ctx context.Context, sessionID string) (*models.TaskStats, error) {
	stats := &models.TaskStats{
		SessionID: sessionID,
	}
	
	err := ts.db.View(func(tx *bbolt.Tx) error {
		// 세션의 모든 태스크 조회
		taskIDs, err := ts.indexer.GetFromIndex(tx, IndexSessionTasks, sessionID)
		if err != nil {
			return err
		}
		
		taskBucket := tx.Bucket([]byte(bucketTasks))
		if taskBucket == nil {
			return fmt.Errorf("tasks bucket not found")
		}
		
		var totalDuration int64
		var completedTasks int64
		
		for _, taskID := range taskIDs {
			data := taskBucket.Get([]byte(taskID))
			if data == nil {
				continue
			}
			
			var task models.Task
			if err := ts.serializer.UnmarshalTask(data, &task); err != nil {
				continue
			}
			
			stats.Total++
			totalDuration += task.Duration
			
			// 태스크 상태별 카운트
			switch task.Status {
			case models.TaskStatusPending:
				stats.Pending++
			case models.TaskStatusRunning:
				stats.Running++
			case models.TaskStatusCompleted:
				stats.Completed++
				completedTasks++
			case models.TaskStatusFailed:
				stats.Failed++
			case models.TaskStatusCancelled:
				stats.Cancelled++
			}
		}
		
		if completedTasks > 0 {
			stats.AverageDuration = time.Duration(totalDuration / completedTasks)
		}
		
		return nil
	})
	
	if err != nil {
		return nil, err
	}
	
	return stats, nil
}

// GetRecentTasks 최근 태스크 조회
func (ts *taskStorage) GetRecentTasks(ctx context.Context, limit int) ([]*models.Task, error) {
	var tasks []*models.Task
	
	err := ts.db.View(func(tx *bbolt.Tx) error {
		taskBucket := tx.Bucket([]byte(bucketTasks))
		if taskBucket == nil {
			return fmt.Errorf("tasks bucket not found")
		}
		
		// 모든 태스크를 수집
		var allTasks []*models.Task
		
		taskBucket.ForEach(func(k, v []byte) error {
			var task models.Task
			if err := ts.serializer.UnmarshalTask(v, &task); err != nil {
				return nil // 파싱 실패한 항목은 무시
			}
			
			allTasks = append(allTasks, &task)
			return nil
		})
		
		// 생성 시간 역순 정렬
		ts.querier.SortTasks(allTasks, "created_at", false)
		
		// 제한된 수만큼 반환
		if len(allTasks) > limit {
			tasks = allTasks[:limit]
		} else {
			tasks = allTasks
		}
		
		return nil
	})
	
	if err != nil {
		return nil, err
	}
	
	return tasks, nil
}

// 헬퍼 메서드

// sortTasksByRelevance 관련성에 따라 태스크 정렬
func (ts *taskStorage) sortTasksByRelevance(tasks []*models.Task, query string) {
	sort.Slice(tasks, func(i, j int) bool {
		scoreI := ts.calculateRelevanceScore(tasks[i].Command, query)
		scoreJ := ts.calculateRelevanceScore(tasks[j].Command, query)
		
		if scoreI != scoreJ {
			return scoreI > scoreJ
		}
		
		// 점수가 같으면 최신 태스크가 우선
		return tasks[i].CreatedAt.After(tasks[j].CreatedAt)
	})
}

// calculateRelevanceScore 관련성 점수 계산
func (ts *taskStorage) calculateRelevanceScore(command, query string) float64 {
	if command == "" || query == "" {
		return 0
	}
	
	// 정확한 매치
	if command == query {
		return 100
	}
	
	// 포함 매치
	if strings.Contains(strings.ToLower(command), strings.ToLower(query)) {
		return 75
	}
	
	// 단어 매치 점수
	commandWords := ts.querier.TokenizeCommand(command)
	queryWords := ts.querier.TokenizeCommand(query)
	
	if len(queryWords) == 0 {
		return 0
	}
	
	matches := 0
	for _, queryWord := range queryWords {
		for _, commandWord := range commandWords {
			if strings.EqualFold(queryWord, commandWord) {
				matches++
				break
			}
		}
	}
	
	return float64(matches) / float64(len(queryWords)) * 50
}