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
		
		if err := ss.serializer.Unmarshal(data, &session); err != nil {
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
			if err := ss.serializer.Unmarshal(data, &session); err != nil {
				continue
			}
			
			allSessions = append(allSessions, &session)
		}
		
		// 생성 시간 역순 정렬
		ss.querier.sortSessions(allSessions, "created_at", SortOrderDesc)
		
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
			if err := ss.serializer.Unmarshal(data, &session); err != nil {
				continue
			}
			
			allSessions = append(allSessions, &session)
		}
		
		// 마지막 활동 시간 역순 정렬
		ss.querier.sortSessions(allSessions, "last_active", SortOrderDesc)
		
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
		
		if err := ss.serializer.Unmarshal(data, &session); err != nil {
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
			if err := ss.serializer.Unmarshal(data, &session); err != nil {
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
func (ss *sessionStorage) Update(ctx context.Context, session *models.Session) error {
	return ss.db.Update(func(tx *bbolt.Tx) error {
		sessionBucket := tx.Bucket([]byte(bucketSessions))
		if sessionBucket == nil {
			return fmt.Errorf("sessions bucket not found")
		}
		
		// 기존 세션 조회
		data := sessionBucket.Get([]byte(session.ID))
		if data == nil {
			return storage.ErrNotFound
		}
		
		var oldSession models.Session
		if err := ss.serializer.Unmarshal(data, &oldSession); err != nil {
			return err
		}
		
		// 인덱스 업데이트를 위한 이전 값 저장
		oldStatus := string(oldSession.Status)
		oldProcessID := oldSession.ProcessID
		
		// 타임스탬프 및 버전 업데이트
		session.UpdatedAt = time.Now()
		session.Version = oldSession.Version + 1
		
		// 직렬화 및 저장
		updatedData, err := ss.serializer.MarshalSession(session)
		if err != nil {
			return err
		}
		
		if err := sessionBucket.Put([]byte(session.ID), updatedData); err != nil {
			return err
		}
		
		// 인덱스 업데이트
		if string(session.Status) != oldStatus {
			ss.indexer.RemoveFromIndex(tx, IndexSessionStatus, oldStatus, session.ID)
			ss.indexer.AddToIndex(tx, IndexSessionStatus, string(session.Status), session.ID)
		}
		
		if session.ProcessID != oldProcessID {
			if oldProcessID > 0 {
				oldPidStr := fmt.Sprintf("%d", oldProcessID)
				ss.indexer.RemoveFromIndex(tx, IndexSessionProcess, oldPidStr, session.ID)
			}
			if session.ProcessID > 0 {
				pidStr := fmt.Sprintf("%d", session.ProcessID)
				ss.indexer.AddToIndex(tx, IndexSessionProcess, pidStr, session.ID)
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
		if err := ss.serializer.Unmarshal(data, &session); err != nil {
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
						if ss.serializer.Unmarshal(taskData, &task) == nil {
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
			if err := ss.serializer.Unmarshal(v, &session); err != nil {
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
	// 기존 세션 조회
	session, err := ss.GetByID(ctx, id)
	if err != nil {
		return err
	}
	
	// LastActive 시간 업데이트
	session.LastActive = time.Now()
	
	return ss.Update(ctx, session)
}

// GetSessionStats 세션 통계 조회
func (ss *sessionStorage) GetSessionStats(ctx context.Context, projectID string) (*models.SessionStats, error) {
	stats := &models.SessionStats{}
	
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
			if err := ss.serializer.Unmarshal(data, &session); err != nil {
				continue
			}
			
			stats.TotalCount++
			totalCommands += session.CommandCount
			
			// 세션 상태별 카운트
			switch session.Status {
			case models.SessionStatusActive:
				stats.ActiveCount++
			case models.SessionStatusIdle:
				stats.IdleCount++
			case models.SessionStatusEnded:
				stats.ErrorCount++ // Ended 대신 ErrorCount 사용
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
		
		// AverageCommands 필드가 없음 - 생략
		if stats.TotalCount > 0 {
			stats.AverageLifetime = totalDuration / time.Duration(stats.TotalCount)
		}
		
		return nil
	})
	
	if err != nil {
		return nil, err
	}
	
	return stats, nil
}

// List 모든 세션 목록 조회 (관리자용)
func (ss *sessionStorage) List(ctx context.Context, filter *models.SessionFilter, pagination *models.PaginationRequest) (*models.PaginationResponse, error) {
	if pagination == nil {
		pagination = &models.PaginationRequest{Limit: 20, Sort: "created_at", Order: "desc"}
	}
	
	var sessions []*models.Session
	var total int
	
	err := ss.db.View(func(tx *bbolt.Tx) error {
		sessionBucket := tx.Bucket([]byte(bucketSessions))
		if sessionBucket == nil {
			return fmt.Errorf("sessions bucket not found")
		}
		
		// 모든 세션 조회
		var allSessions []*models.Session
		cursor := sessionBucket.Cursor()
		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			var session models.Session
			if err := ss.serializer.Unmarshal(v, &session); err != nil {
				continue
			}
			
			// 필터 적용
			if filter != nil {
				if filter.ProjectID != "" && session.ProjectID != filter.ProjectID {
					continue
				}
				if filter.Status != "" && session.Status != models.SessionStatus(filter.Status) {
					continue
				}
			}
			
			allSessions = append(allSessions, &session)
		}
		
		// 정렬
		ss.querier.sortSessions(allSessions, pagination.Sort, SortOrder(pagination.Order))
		
		total = len(allSessions)
		
		// 페이지네이션 적용
		start, end := ss.querier.CalculatePagination(len(allSessions), pagination)
		sessions = allSessions[start:end]
		
		return nil
	})
	
	if err != nil {
		return nil, err
	}
	
	// PaginationResponse 생성
	meta := models.NewPaginationMeta(pagination.Page, pagination.Limit, total)
	return &models.PaginationResponse{
		Data: sessions,
		Meta: meta,
	}, nil
}