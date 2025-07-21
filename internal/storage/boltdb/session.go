package boltdb

import (
	"context"
	"fmt"
	"time"

	"go.etcd.io/bbolt"

	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/storage"
)

// sessionStorage BoltDB Session 스토리지 구현
type sessionStorage struct {
	db         *bbolt.DB
	serializer *Serializer
	indexer    *IndexManager
	querier    *QueryHelper
}

// newSessionStorage 세션 스토리지 생성자
func newSessionStorage(storage *Storage) *sessionStorage {
	return &sessionStorage{
		db:         storage.db,
		serializer: storage.serializer,
		indexer:    storage.indexer,
		querier:    storage.querier,
	}
}

// Create 새 세션 생성
func (ss *sessionStorage) Create(ctx context.Context, session *models.Session) error {
	return ss.db.Update(func(tx *bbolt.Tx) error {
		// 세션 버킷 가져오기
		sessionBucket := tx.Bucket([]byte(bucketSessions))
		if sessionBucket == nil {
			return fmt.Errorf("sessions bucket not found")
		}
		
		// ID 중복 체크
		if existing := sessionBucket.Get([]byte(session.ID)); existing != nil {
			return storage.ErrAlreadyExists
		}
		
		// 프로젝트 존재 확인
		projectBucket := tx.Bucket([]byte(bucketProjects))
		if projectBucket == nil {
			return fmt.Errorf("projects bucket not found")
		}
		
		projectData := projectBucket.Get([]byte(session.ProjectID))
		if projectData == nil {
			return storage.ErrNotFound
		}
		
		// 프로젝트가 삭제되지 않았는지 확인
		var project models.Project
		if err := ss.serializer.UnmarshalProject(projectData, &project); err != nil {
			return err
		}
		if project.DeletedAt != nil {
			return storage.ErrNotFound
		}
		
		// 타임스탬프 설정
		now := time.Now()
		if session.CreatedAt.IsZero() {
			session.CreatedAt = now
		}
		session.UpdatedAt = now
		if session.Version == 0 {
			session.Version = 1
		}
		
		// LastActive 기본값 설정
		if session.LastActive.IsZero() {
			session.LastActive = now
		}
		
		// 세션 직렬화 및 저장
		data, err := ss.serializer.MarshalSession(session)
		if err != nil {
			return err
		}
		
		if err := sessionBucket.Put([]byte(session.ID), data); err != nil {
			return err
		}
		
		// 인덱스 업데이트
		if err := ss.indexer.AddToIndex(tx, IndexProjectSessions, session.ProjectID, session.ID); err != nil {
			return err
		}
		
		if err := ss.indexer.AddToIndex(tx, IndexSessionStatus, string(session.Status), session.ID); err != nil {
			return err
		}
		
		if session.ProcessID > 0 {
			pidStr := fmt.Sprintf("%d", session.ProcessID)
			if err := ss.indexer.AddToIndex(tx, IndexSessionProcess, pidStr, session.ID); err != nil {
				return err
			}
		}
		
		return nil
	})
}

// GetByID ID로 세션 조회
func (ss *sessionStorage) GetByID(ctx context.Context, id string) (*models.Session, error) {
	var session models.Session
	
	err := ss.db.View(func(tx *bbolt.Tx) error {
		sessionBucket := tx.Bucket([]byte(bucketSessions))
		if sessionBucket == nil {
			return fmt.Errorf("sessions bucket not found")
		}
		
		data := sessionBucket.Get([]byte(id))
		if data == nil {
			return storage.ErrNotFound
		}
		
		if err := ss.serializer.UnmarshalSession(data, &session); err != nil {
			return err
		}
		
		return nil
	})
	
	if err != nil {
		return nil, err
	}
	
	return &session, nil
}

// GetByProjectID 프로젝트별 세션 목록 조회
func (ss *sessionStorage) GetByProjectID(ctx context.Context, projectID string, pagination *models.PaginationRequest) ([]*models.Session, int64, error) {
	var sessions []*models.Session
	var total int64
	
	err := ss.db.View(func(tx *bbolt.Tx) error {
		// 인덱스에서 세션 ID 목록 가져오기
		sessionIDs, err := ss.indexer.GetFromIndex(tx, IndexProjectSessions, projectID)
		if err != nil {
			return err
		}
		
		// 세션 데이터 조회 및 필터링
		var allSessions []*models.Session
		sessionBucket := tx.Bucket([]byte(bucketSessions))
		if sessionBucket == nil {
			return fmt.Errorf("sessions bucket not found")
		}
		
		for _, sessionID := range sessionIDs {
			data := sessionBucket.Get([]byte(sessionID))
			if data == nil {
				continue
			}
			
			var session models.Session
			if err := ss.serializer.UnmarshalSession(data, &session); err != nil {
				continue
			}
			
			allSessions = append(allSessions, &session)
		}
		
		// 생성 시간 역순 정렬
		ss.querier.SortSessions(allSessions, "created_at", false)
		
		total = int64(len(allSessions))
		
		// 페이지네이션 적용
		start, end := ss.querier.CalculatePagination(len(allSessions), pagination)
		sessions = allSessions[start:end]
		
		return nil
	})
	
	if err != nil {
		return nil, 0, err
	}
	
	return sessions, total, nil
}

// GetByStatus 상태별 세션 조회
func (ss *sessionStorage) GetByStatus(ctx context.Context, status models.SessionStatus, pagination *models.PaginationRequest) ([]*models.Session, int64, error) {
	var sessions []*models.Session
	var total int64
	
	err := ss.db.View(func(tx *bbolt.Tx) error {
		// 인덱스에서 세션 ID 목록 가져오기
		sessionIDs, err := ss.indexer.GetFromIndex(tx, IndexSessionStatus, string(status))
		if err != nil {
			return err
		}
		
		// 세션 데이터 조회
		var allSessions []*models.Session
		sessionBucket := tx.Bucket([]byte(bucketSessions))
		if sessionBucket == nil {
			return fmt.Errorf("sessions bucket not found")
		}
		
		for _, sessionID := range sessionIDs {
			data := sessionBucket.Get([]byte(sessionID))
			if data == nil {
				continue
			}
			
			var session models.Session
			if err := ss.serializer.UnmarshalSession(data, &session); err != nil {
				continue
			}
			
			allSessions = append(allSessions, &session)
		}
		
		// 마지막 활동 시간 역순 정렬
		ss.querier.SortSessions(allSessions, "last_active", false)
		
		total = int64(len(allSessions))
		
		// 페이지네이션 적용
		start, end := ss.querier.CalculatePagination(len(allSessions), pagination)
		sessions = allSessions[start:end]
		
		return nil
	})
	
	if err != nil {
		return nil, 0, err
	}
	
	return sessions, total, nil
}

// GetByProcessID 프로세스 ID로 세션 조회
func (ss *sessionStorage) GetByProcessID(ctx context.Context, processID int32) (*models.Session, error) {
	var session models.Session
	
	err := ss.db.View(func(tx *bbolt.Tx) error {
		// 인덱스에서 세션 ID 찾기
		pidStr := fmt.Sprintf("%d", processID)
		sessionIDs, err := ss.indexer.GetFromIndex(tx, IndexSessionProcess, pidStr)
		if err != nil {
			return err
		}
		
		if len(sessionIDs) == 0 {
			return storage.ErrNotFound
		}
		
		// 첫 번째 매치 (프로세스 ID는 고유해야 함)
		sessionBucket := tx.Bucket([]byte(bucketSessions))
		if sessionBucket == nil {
			return fmt.Errorf("sessions bucket not found")
		}
		
		data := sessionBucket.Get([]byte(sessionIDs[0]))
		if data == nil {
			return storage.ErrNotFound
		}
		
		if err := ss.serializer.UnmarshalSession(data, &session); err != nil {
			return err
		}
		
		return nil
	})
	
	if err != nil {
		return nil, err
	}
	
	return &session, nil
}

// GetActiveCount 프로젝트의 활성 세션 수 조회
func (ss *sessionStorage) GetActiveCount(ctx context.Context, projectID string) (int64, error) {
	var count int64
	
	err := ss.db.View(func(tx *bbolt.Tx) error {
		// 프로젝트의 모든 세션 조회
		sessionIDs, err := ss.indexer.GetFromIndex(tx, IndexProjectSessions, projectID)
		if err != nil {
			return err
		}
		
		sessionBucket := tx.Bucket([]byte(bucketSessions))
		if sessionBucket == nil {
			return fmt.Errorf("sessions bucket not found")
		}
		
		for _, sessionID := range sessionIDs {
			data := sessionBucket.Get([]byte(sessionID))
			if data == nil {
				continue
			}
			
			var session models.Session
			if err := ss.serializer.UnmarshalSession(data, &session); err != nil {
				continue
			}
			
			if session.Status == models.SessionStatusActive {
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

// Update 세션 정보 업데이트
func (ss *sessionStorage) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	return ss.db.Update(func(tx *bbolt.Tx) error {
		sessionBucket := tx.Bucket([]byte(bucketSessions))
		if sessionBucket == nil {
			return fmt.Errorf("sessions bucket not found")
		}
		
		// 기존 세션 조회
		data := sessionBucket.Get([]byte(id))
		if data == nil {
			return storage.ErrNotFound
		}
		
		var session models.Session
		if err := ss.serializer.UnmarshalSession(data, &session); err != nil {
			return err
		}
		
		// 인덱스 업데이트를 위한 이전 값 저장
		oldStatus := string(session.Status)
		oldProcessID := session.ProcessID
		
		// 업데이트 적용
		if status, ok := updates["status"]; ok {
			if statusVal, ok := status.(models.SessionStatus); ok {
				session.Status = statusVal
			} else if statusStr, ok := status.(string); ok {
				session.Status = models.SessionStatus(statusStr)
			}
		}
		
		if processID, ok := updates["process_id"]; ok {
			if pidVal, ok := processID.(int32); ok {
				session.ProcessID = pidVal
			}
		}
		
		if startedAt, ok := updates["started_at"]; ok {
			if startedTime, ok := startedAt.(*time.Time); ok {
				session.StartedAt = startedTime
			}
		}
		
		if endedAt, ok := updates["ended_at"]; ok {
			if endedTime, ok := endedAt.(*time.Time); ok {
				session.EndedAt = endedTime
			}
		}
		
		if lastActive, ok := updates["last_active"]; ok {
			if activeTime, ok := lastActive.(time.Time); ok {
				session.LastActive = activeTime
			}
		}
		
		if commandCount, ok := updates["command_count"]; ok {
			if countVal, ok := commandCount.(int64); ok {
				session.CommandCount = countVal
			}
		}
		
		if metadata, ok := updates["metadata"]; ok {
			if metaVal, ok := metadata.(map[string]interface{}); ok {
				session.Metadata = metaVal
			}
		}
		
		// 타임스탬프 및 버전 업데이트
		session.UpdatedAt = time.Now()
		session.Version++
		
		// 직렬화 및 저장
		updatedData, err := ss.serializer.MarshalSession(&session)
		if err != nil {
			return err
		}
		
		if err := sessionBucket.Put([]byte(id), updatedData); err != nil {
			return err
		}
		
		// 인덱스 업데이트
		if string(session.Status) != oldStatus {
			ss.indexer.RemoveFromIndex(tx, IndexSessionStatus, oldStatus, id)
			ss.indexer.AddToIndex(tx, IndexSessionStatus, string(session.Status), id)
		}
		
		if session.ProcessID != oldProcessID {
			if oldProcessID > 0 {
				oldPidStr := fmt.Sprintf("%d", oldProcessID)
				ss.indexer.RemoveFromIndex(tx, IndexSessionProcess, oldPidStr, id)
			}
			if session.ProcessID > 0 {
				pidStr := fmt.Sprintf("%d", session.ProcessID)
				ss.indexer.AddToIndex(tx, IndexSessionProcess, pidStr, id)
			}
		}
		
		return nil
	})
}

// Delete 세션 삭제
func (ss *sessionStorage) Delete(ctx context.Context, id string) error {
	return ss.db.Update(func(tx *bbolt.Tx) error {
		sessionBucket := tx.Bucket([]byte(bucketSessions))
		if sessionBucket == nil {
			return fmt.Errorf("sessions bucket not found")
		}
		
		// 기존 세션 조회
		data := sessionBucket.Get([]byte(id))
		if data == nil {
			return storage.ErrNotFound
		}
		
		var session models.Session
		if err := ss.serializer.UnmarshalSession(data, &session); err != nil {
			return err
		}
		
		// 관련 태스크가 있는지 확인
		taskIDs, err := ss.indexer.GetFromIndex(tx, IndexSessionTasks, id)
		if err != nil {
			return err
		}
		
		// 실행 중인 태스크가 있는지 확인
		if len(taskIDs) > 0 {
			taskBucket := tx.Bucket([]byte(bucketTasks))
			if taskBucket != nil {
				for _, taskID := range taskIDs {
					taskData := taskBucket.Get([]byte(taskID))
					if taskData != nil {
						var task models.Task
						if ss.serializer.UnmarshalTask(taskData, &task) == nil {
							if task.Status == models.TaskStatusRunning || task.Status == models.TaskStatusPending {
								return fmt.Errorf("cannot delete session with running or pending tasks")
							}
						}
					}
				}
			}
		}
		
		// 인덱스에서 제거
		ss.indexer.RemoveFromIndex(tx, IndexProjectSessions, session.ProjectID, id)
		ss.indexer.RemoveFromIndex(tx, IndexSessionStatus, string(session.Status), id)
		if session.ProcessID > 0 {
			pidStr := fmt.Sprintf("%d", session.ProcessID)
			ss.indexer.RemoveFromIndex(tx, IndexSessionProcess, pidStr, id)
		}
		
		// 세션 삭제
		return sessionBucket.Delete([]byte(id))
	})
}

// GetIdleSessions 유휴 세션 조회 (마지막 활동이 일정 시간 이전)
func (ss *sessionStorage) GetIdleSessions(ctx context.Context, threshold time.Duration) ([]*models.Session, error) {
	var sessions []*models.Session
	cutoff := time.Now().Add(-threshold)
	
	err := ss.db.View(func(tx *bbolt.Tx) error {
		sessionBucket := tx.Bucket([]byte(bucketSessions))
		if sessionBucket == nil {
			return fmt.Errorf("sessions bucket not found")
		}
		
		// 모든 세션 순회
		return sessionBucket.ForEach(func(k, v []byte) error {
			var session models.Session
			if err := ss.serializer.UnmarshalSession(v, &session); err != nil {
				return nil // 파싱 실패한 항목은 무시
			}
			
			// 활성 또는 유휴 상태이고 임계값보다 오래된 세션
			if (session.Status == models.SessionStatusActive || session.Status == models.SessionStatusIdle) &&
				session.LastActive.Before(cutoff) {
				sessions = append(sessions, &session)
			}
			
			return nil
		})
	})
	
	if err != nil {
		return nil, err
	}
	
	return sessions, nil
}

// UpdateLastActive 마지막 활동 시간 업데이트
func (ss *sessionStorage) UpdateLastActive(ctx context.Context, id string) error {
	updates := map[string]interface{}{
		"last_active": time.Now(),
	}
	return ss.Update(ctx, id, updates)
}

// GetSessionStats 세션 통계 조회
func (ss *sessionStorage) GetSessionStats(ctx context.Context, projectID string) (*models.SessionStats, error) {
	stats := &models.SessionStats{
		ProjectID: projectID,
	}
	
	err := ss.db.View(func(tx *bbolt.Tx) error {
		// 프로젝트의 모든 세션 조회
		sessionIDs, err := ss.indexer.GetFromIndex(tx, IndexProjectSessions, projectID)
		if err != nil {
			return err
		}
		
		sessionBucket := tx.Bucket([]byte(bucketSessions))
		if sessionBucket == nil {
			return fmt.Errorf("sessions bucket not found")
		}
		
		var totalCommands int64
		var totalDuration time.Duration
		
		for _, sessionID := range sessionIDs {
			data := sessionBucket.Get([]byte(sessionID))
			if data == nil {
				continue
			}
			
			var session models.Session
			if err := ss.serializer.UnmarshalSession(data, &session); err != nil {
				continue
			}
			
			stats.Total++
			totalCommands += session.CommandCount
			
			// 세션 상태별 카운트
			switch session.Status {
			case models.SessionStatusActive:
				stats.Active++
			case models.SessionStatusIdle:
				stats.Idle++
			case models.SessionStatusEnded:
				stats.Ended++
			}
			
			// 세션 지속 시간 계산
			if session.StartedAt != nil {
				endTime := time.Now()
				if session.EndedAt != nil {
					endTime = *session.EndedAt
				}
				duration := endTime.Sub(*session.StartedAt)
				totalDuration += duration
			}
		}
		
		stats.AverageCommands = float64(totalCommands) / float64(stats.Total)
		if stats.Total > 0 {
			stats.AverageDuration = totalDuration / time.Duration(stats.Total)
		}
		
		return nil
	})
	
	if err != nil {
		return nil, err
	}
	
	return stats, nil
}