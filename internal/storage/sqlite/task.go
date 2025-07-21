package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/storage"
)

// taskStorage 태스크 스토리지 SQLite 구현
type taskStorage struct {
	storage *Storage
}

// newTaskStorage 새 태스크 스토리지 생성
func newTaskStorage(s *Storage) *taskStorage {
	return &taskStorage{storage: s}
}

const (
	// 태스크 조회 쿼리
	selectTaskQuery = `
		SELECT id, session_id, command, status, output, error, started_at, completed_at,
		       bytes_in, bytes_out, duration, created_at, updated_at, version
		FROM tasks
	`
	
	// 태스크 삽입 쿼리
	insertTaskQuery = `
		INSERT INTO tasks (id, session_id, command, status, output, error, started_at, 
		                  completed_at, bytes_in, bytes_out, duration, created_at, updated_at, version)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	
	// 태스크 업데이트 쿼리
	updateTaskQuery = `
		UPDATE tasks 
		SET command = ?, status = ?, output = ?, error = ?, started_at = ?, completed_at = ?,
		    bytes_in = ?, bytes_out = ?, duration = ?, updated_at = ?, version = version + 1
		WHERE id = ? AND version = ?
	`
	
	// 태스크 삭제 쿼리
	deleteTaskQuery = `DELETE FROM tasks WHERE id = ?`
	
	// 활성 태스크 수 조회 쿼리
	countActiveTasksQuery = `
		SELECT COUNT(*) FROM tasks 
		WHERE session_id = ? AND (status = 'pending' OR status = 'running')
	`
	
	// 카운트 쿼리
	countTasksQuery = `SELECT COUNT(*) FROM tasks`
)

// Create 새 태스크 생성
func (t *taskStorage) Create(ctx context.Context, task *models.Task) error {
	now := time.Now()
	
	// 기본값 설정
	if task.Status == "" {
		task.Status = models.TaskPending
	}
	task.CreatedAt = now
	task.UpdatedAt = now
	if task.Version == 0 {
		task.Version = 1
	}
	
	// 태스크 삽입
	_, err := t.storage.execContext(ctx, insertTaskQuery,
		task.ID,
		task.SessionID,
		task.Command,
		task.Status,
		task.Output,
		task.Error,
		task.StartedAt,
		task.CompletedAt,
		task.BytesIn,
		task.BytesOut,
		task.Duration,
		task.CreatedAt,
		task.UpdatedAt,
		task.Version,
	)
	
	if err != nil {
		return storage.ConvertError(err, "create task", "sqlite")
	}
	
	return nil
}

// GetByID ID로 태스크 조회
func (t *taskStorage) GetByID(ctx context.Context, id string) (*models.Task, error) {
	query := selectTaskQuery + ` WHERE id = ?`
	
	row := t.storage.queryRowContext(ctx, query, id)
	
	task, err := t.scanTask(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrNotFound
		}
		return nil, storage.ConvertError(err, "get task by id", "sqlite")
	}
	
	return task, nil
}

// List 태스크 목록 조회
func (t *taskStorage) List(ctx context.Context, filter *models.TaskFilter, paging *PagingRequest) ([]*models.Task, int, error) {
	if paging == nil {
		paging = &PagingRequest{Page: 1, Limit: 20, Sort: "created_at", Order: "desc"}
	}
	paging.Normalize()
	
	// WHERE 조건 생성
	whereConditions := []string{}
	args := []interface{}{}
	
	if filter != nil {
		if filter.SessionID != nil && *filter.SessionID != "" {
			whereConditions = append(whereConditions, "session_id = ?")
			args = append(args, *filter.SessionID)
		}
		if filter.Status != nil {
			whereConditions = append(whereConditions, "status = ?")
			args = append(args, *filter.Status)
		}
		if filter.Active != nil {
			if *filter.Active {
				whereConditions = append(whereConditions, "(status = 'pending' OR status = 'running')")
			} else {
				whereConditions = append(whereConditions, "(status = 'completed' OR status = 'failed' OR status = 'cancelled')")
			}
		}
	}
	
	whereClause := ""
	if len(whereConditions) > 0 {
		whereClause = " WHERE " + strings.Join(whereConditions, " AND ")
	}
	
	// 전체 카운트 조회
	countQuery := countTasksQuery + whereClause
	var total int
	err := t.storage.queryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, storage.ConvertError(err, "count tasks", "sqlite")
	}
	
	// 데이터 조회
	query := selectTaskQuery + whereClause + `
		ORDER BY ` + paging.Sort + ` ` + paging.Order + `
		LIMIT ? OFFSET ?
	`
	queryArgs := append(args, paging.Limit, paging.GetOffset())
	
	rows, err := t.storage.queryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, storage.ConvertError(err, "list tasks", "sqlite")
	}
	defer rows.Close()
	
	tasks, err := t.scanTasks(rows)
	if err != nil {
		return nil, 0, err
	}
	
	return tasks, total, nil
}

// Update 태스크 업데이트
func (t *taskStorage) Update(ctx context.Context, task *models.Task) error {
	task.UpdatedAt = time.Now()
	
	result, err := t.storage.execContext(ctx, updateTaskQuery,
		task.Command,
		task.Status,
		task.Output,
		task.Error,
		task.StartedAt,
		task.CompletedAt,
		task.BytesIn,
		task.BytesOut,
		task.Duration,
		task.UpdatedAt,
		task.ID,
		task.Version,
	)
	
	if err != nil {
		return storage.ConvertError(err, "update task", "sqlite")
	}
	
	// 업데이트된 행이 있는지 확인
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return storage.ConvertError(err, "check rows affected", "sqlite")
	}
	
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}
	
	// 버전 증가
	task.Version++
	
	return nil
}

// Delete 태스크 삭제
func (t *taskStorage) Delete(ctx context.Context, id string) error {
	result, err := t.storage.execContext(ctx, deleteTaskQuery, id)
	if err != nil {
		return storage.ConvertError(err, "delete task", "sqlite")
	}
	
	// 삭제된 행이 있는지 확인
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return storage.ConvertError(err, "check rows affected", "sqlite")
	}
	
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}
	
	return nil
}

// GetBySessionID 세션 ID로 태스크 목록 조회
func (t *taskStorage) GetBySessionID(ctx context.Context, sessionID string, paging *PagingRequest) ([]*models.Task, int, error) {
	if paging == nil {
		paging = &PagingRequest{Page: 1, Limit: 20, Sort: "created_at", Order: "desc"}
	}
	paging.Normalize()
	
	// 전체 카운트 조회
	countQuery := countTasksQuery + ` WHERE session_id = ?`
	var total int
	err := t.storage.queryRowContext(ctx, countQuery, sessionID).Scan(&total)
	if err != nil {
		return nil, 0, storage.ConvertError(err, "count tasks by session", "sqlite")
	}
	
	// 데이터 조회
	query := selectTaskQuery + `
		WHERE session_id = ?
		ORDER BY ` + paging.Sort + ` ` + paging.Order + `
		LIMIT ? OFFSET ?
	`
	
	rows, err := t.storage.queryContext(ctx, query, sessionID, paging.Limit, paging.GetOffset())
	if err != nil {
		return nil, 0, storage.ConvertError(err, "get tasks by session", "sqlite")
	}
	defer rows.Close()
	
	tasks, err := t.scanTasks(rows)
	if err != nil {
		return nil, 0, err
	}
	
	return tasks, total, nil
}

// GetActiveCount 활성 태스크 수 조회
func (t *taskStorage) GetActiveCount(ctx context.Context, sessionID string) (int64, error) {
	var count int64
	err := t.storage.queryRowContext(ctx, countActiveTasksQuery, sessionID).Scan(&count)
	
	if err != nil {
		return 0, storage.ConvertError(err, "get active task count", "sqlite")
	}
	
	return count, nil
}

// scanTask 단일 태스크 스캔
func (t *taskStorage) scanTask(row *sql.Row) (*models.Task, error) {
	task := &models.Task{}
	var output, error sql.NullString
	var startedAt, completedAt sql.NullTime
	
	err := row.Scan(
		&task.ID,
		&task.SessionID,
		&task.Command,
		&task.Status,
		&output,
		&error,
		&startedAt,
		&completedAt,
		&task.BytesIn,
		&task.BytesOut,
		&task.Duration,
		&task.CreatedAt,
		&task.UpdatedAt,
		&task.Version,
	)
	
	if err != nil {
		return nil, err
	}
	
	// NULL 값 처리
	if output.Valid {
		task.Output = output.String
	}
	if error.Valid {
		task.Error = error.String
	}
	if startedAt.Valid {
		task.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		task.CompletedAt = &completedAt.Time
	}
	
	return task, nil
}

// scanTasks 여러 태스크 스캔
func (t *taskStorage) scanTasks(rows *sql.Rows) ([]*models.Task, error) {
	var tasks []*models.Task
	
	for rows.Next() {
		task := &models.Task{}
		var output, error sql.NullString
		var startedAt, completedAt sql.NullTime
		
		err := rows.Scan(
			&task.ID,
			&task.SessionID,
			&task.Command,
			&task.Status,
			&output,
			&error,
			&startedAt,
			&completedAt,
			&task.BytesIn,
			&task.BytesOut,
			&task.Duration,
			&task.CreatedAt,
			&task.UpdatedAt,
			&task.Version,
		)
		
		if err != nil {
			return nil, storage.ConvertError(err, "scan task row", "sqlite")
		}
		
		// NULL 값 처리
		if output.Valid {
			task.Output = output.String
		}
		if error.Valid {
			task.Error = error.String
		}
		if startedAt.Valid {
			task.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			task.CompletedAt = &completedAt.Time
		}
		
		tasks = append(tasks, task)
	}
	
	// rows 순회 중 에러 확인
	if err := rows.Err(); err != nil {
		return nil, storage.ConvertError(err, "scan task rows", "sqlite")
	}
	
	return tasks, nil
}