package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/storage"
)

// sessionStorage 세션 스토리지 SQLite 구현
type sessionStorage struct {
	storage *Storage
}

// newSessionStorage 새 세션 스토리지 생성
func newSessionStorage(s *Storage) *sessionStorage {
	return &sessionStorage{storage: s}
}

// PagingRequest 기본 페이징 요청 구조 (임시)
type PagingRequest struct {
	Page  int    `json:"page,default=1"`
	Limit int    `json:"limit,default=20"`
	Sort  string `json:"sort,default=created_at"`
	Order string `json:"order,default=desc"`
}

// PagingResponse 페이징 응답 구조 (임시)
type PagingResponse[T any] struct {
	Data []T          `json:"data"`
	Meta PaginationMeta `json:"meta"`
}

// PaginationMeta 페이지네이션 메타데이터
type PaginationMeta struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// Normalize 페이징 요청 정규화
func (p *PagingRequest) Normalize() {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.Limit < 1 {
		p.Limit = 20
	}
	if p.Limit > 100 {
		p.Limit = 100
	}
	if p.Sort == "" {
		p.Sort = "created_at"
	}
	if p.Order != "asc" && p.Order != "desc" {
		p.Order = "desc"
	}
}

// GetOffset 오프셋 계산
func (p *PagingRequest) GetOffset() int {
	return (p.Page - 1) * p.Limit
}

// NewPaginationMeta 페이지네이션 메타데이터 생성
func NewPaginationMeta(page, limit, total int) PaginationMeta {
	totalPages := total / limit
	if total%limit > 0 {
		totalPages++
	}
	return PaginationMeta{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}
}

const (
	// 세션 조회 쿼리
	selectSessionQuery = `
		SELECT id, project_id, process_id, status, started_at, ended_at, last_active,
		       metadata, command_count, bytes_in, bytes_out, error_count, 
		       max_idle_time, max_lifetime, created_at, updated_at, version
		FROM sessions
	`
	
	// 세션 삽입 쿼리
	insertSessionQuery = `
		INSERT INTO sessions (id, project_id, process_id, status, started_at, ended_at, 
		                     last_active, metadata, command_count, bytes_in, bytes_out, 
		                     error_count, max_idle_time, max_lifetime, created_at, updated_at, version)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	
	// 세션 업데이트 쿼리
	updateSessionQuery = `
		UPDATE sessions 
		SET status = ?, started_at = ?, ended_at = ?, last_active = ?, metadata = ?,
		    command_count = ?, bytes_in = ?, bytes_out = ?, error_count = ?,
		    max_idle_time = ?, max_lifetime = ?, updated_at = ?, version = version + 1
		WHERE id = ? AND version = ?
	`
	
	// 세션 삭제 쿼리
	deleteSessionQuery = `DELETE FROM sessions WHERE id = ?`
	
	// 활성 세션 수 조회 쿼리
	countActiveSessionsQuery = `
		SELECT COUNT(*) FROM sessions 
		WHERE project_id = ? AND (status = 'active' OR status = 'idle')
	`
	
	// 카운트 쿼리
	countSessionsQuery = `SELECT COUNT(*) FROM sessions`
)

// Create 새 세션 생성
func (s *sessionStorage) Create(ctx context.Context, session *models.Session) error {
	now := time.Now()
	
	// 기본값 설정
	if session.Status == "" {
		session.Status = models.SessionPending
	}
	session.CreatedAt = now
	session.UpdatedAt = now
	session.LastActive = now
	if session.Version == 0 {
		session.Version = 1
	}
	
	// 기본 타임아웃 설정
	if session.MaxIdleTime == 0 {
		session.MaxIdleTime = 30 * time.Minute
	}
	if session.MaxLifetime == 0 {
		session.MaxLifetime = 4 * time.Hour
	}
	
	// 메타데이터 JSON 직렬화
	metadataJSON, err := s.marshalMetadata(session.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}
	
	// 세션 삽입
	_, err = s.storage.execContext(ctx, insertSessionQuery,
		session.ID,
		session.ProjectID,
		session.ProcessID,
		session.Status,
		session.StartedAt,
		session.EndedAt,
		session.LastActive,
		metadataJSON,
		session.CommandCount,
		session.BytesIn,
		session.BytesOut,
		session.ErrorCount,
		int64(session.MaxIdleTime),
		int64(session.MaxLifetime),
		session.CreatedAt,
		session.UpdatedAt,
		session.Version,
	)
	
	if err != nil {
		return storage.ConvertError(err, "create session", "sqlite")
	}
	
	return nil
}

// GetByID ID로 세션 조회
func (s *sessionStorage) GetByID(ctx context.Context, id string) (*models.Session, error) {
	query := selectSessionQuery + ` WHERE id = ?`
	
	row := s.storage.queryRowContext(ctx, query, id)
	
	session, err := s.scanSession(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrNotFound
		}
		return nil, storage.ConvertError(err, "get session by id", "sqlite")
	}
	
	return session, nil
}

// List 세션 목록 조회
func (s *sessionStorage) List(ctx context.Context, filter *models.SessionFilter, paging *models.PaginationRequest) (*models.PaginationResponse, error) {
	if paging == nil {
		paging = &models.PaginationRequest{Page: 1, Limit: 20, Sort: "created_at", Order: "desc"}
	}
	paging.Normalize()
	
	// WHERE 조건 생성
	whereConditions := []string{}
	args := []interface{}{}
	
	if filter != nil {
		if filter.ProjectID != "" {
			whereConditions = append(whereConditions, "project_id = ?")
			args = append(args, filter.ProjectID)
		}
		if filter.Status != "" {
			whereConditions = append(whereConditions, "status = ?")
			args = append(args, filter.Status)
		}
		if filter.Active != nil {
			if *filter.Active {
				whereConditions = append(whereConditions, "(status = 'active' OR status = 'idle')")
			} else {
				whereConditions = append(whereConditions, "(status = 'ended' OR status = 'error')")
			}
		}
	}
	
	whereClause := ""
	if len(whereConditions) > 0 {
		whereClause = " WHERE " + strings.Join(whereConditions, " AND ")
	}
	
	// 전체 카운트 조회
	countQuery := countSessionsQuery + whereClause
	var total int
	err := s.storage.queryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, storage.ConvertError(err, "count sessions", "sqlite")
	}
	
	// 데이터 조회
	query := selectSessionQuery + whereClause + `
		ORDER BY ` + paging.Sort + ` ` + paging.Order + `
		LIMIT ? OFFSET ?
	`
	queryArgs := append(args, paging.Limit, paging.GetOffset())
	
	rows, err := s.storage.queryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, storage.ConvertError(err, "list sessions", "sqlite")
	}
	defer rows.Close()
	
	sessions, err := s.scanSessions(rows)
	if err != nil {
		return nil, err
	}
	
	// 페이징 메타 생성
	totalPages := total / paging.Limit
	if total%paging.Limit > 0 {
		totalPages++
	}
	
	return &models.PaginationResponse{
		Data: sessions,
		Meta: models.PaginationMeta{
			CurrentPage: paging.Page,
			PerPage:     paging.Limit,
			Total:       total,
			TotalPages:  totalPages,
			HasNext:     paging.Page < totalPages,
			HasPrev:     paging.Page > 1,
		},
	}, nil
}

// Update 세션 업데이트
func (s *sessionStorage) Update(ctx context.Context, session *models.Session) error {
	// 메타데이터 JSON 직렬화
	metadataJSON, err := s.marshalMetadata(session.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}
	
	session.UpdatedAt = time.Now()
	
	result, err := s.storage.execContext(ctx, updateSessionQuery,
		session.Status,
		session.StartedAt,
		session.EndedAt,
		session.LastActive,
		metadataJSON,
		session.CommandCount,
		session.BytesIn,
		session.BytesOut,
		session.ErrorCount,
		int64(session.MaxIdleTime),
		int64(session.MaxLifetime),
		session.UpdatedAt,
		session.ID,
		session.Version,
	)
	
	if err != nil {
		return storage.ConvertError(err, "update session", "sqlite")
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
	session.Version++
	
	return nil
}

// Delete 세션 삭제
func (s *sessionStorage) Delete(ctx context.Context, id string) error {
	result, err := s.storage.execContext(ctx, deleteSessionQuery, id)
	if err != nil {
		return storage.ConvertError(err, "delete session", "sqlite")
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

// GetActiveCount 활성 세션 수 조회
func (s *sessionStorage) GetActiveCount(ctx context.Context, projectID string) (int64, error) {
	var count int64
	err := s.storage.queryRowContext(ctx, countActiveSessionsQuery, projectID).Scan(&count)
	
	if err != nil {
		return 0, storage.ConvertError(err, "get active session count", "sqlite")
	}
	
	return count, nil
}

// marshalMetadata 메타데이터를 JSON으로 직렬화
func (s *sessionStorage) marshalMetadata(metadata map[string]string) (string, error) {
	if metadata == nil || len(metadata) == 0 {
		return "{}", nil
	}
	
	data, err := json.Marshal(metadata)
	if err != nil {
		return "", err
	}
	
	return string(data), nil
}

// unmarshalMetadata JSON을 메타데이터로 역직렬화
func (s *sessionStorage) unmarshalMetadata(data string) (map[string]string, error) {
	var metadata map[string]string
	
	if data == "" || data == "null" || data == "{}" {
		return make(map[string]string), nil
	}
	
	err := json.Unmarshal([]byte(data), &metadata)
	if err != nil {
		return nil, err
	}
	
	if metadata == nil {
		metadata = make(map[string]string)
	}
	
	return metadata, nil
}

// scanSession 단일 세션 스캔
func (s *sessionStorage) scanSession(row *sql.Row) (*models.Session, error) {
	session := &models.Session{}
	var startedAt, endedAt sql.NullTime
	var metadataJSON string
	var maxIdleTimeNS, maxLifetimeNS int64
	
	err := row.Scan(
		&session.ID,
		&session.ProjectID,
		&session.ProcessID,
		&session.Status,
		&startedAt,
		&endedAt,
		&session.LastActive,
		&metadataJSON,
		&session.CommandCount,
		&session.BytesIn,
		&session.BytesOut,
		&session.ErrorCount,
		&maxIdleTimeNS,
		&maxLifetimeNS,
		&session.CreatedAt,
		&session.UpdatedAt,
		&session.Version,
	)
	
	if err != nil {
		return nil, err
	}
	
	// NULL 값 처리
	if startedAt.Valid {
		session.StartedAt = &startedAt.Time
	}
	if endedAt.Valid {
		session.EndedAt = &endedAt.Time
	}
	
	// Duration 변환
	session.MaxIdleTime = time.Duration(maxIdleTimeNS)
	session.MaxLifetime = time.Duration(maxLifetimeNS)
	
	// 메타데이터 역직렬화
	session.Metadata, err = s.unmarshalMetadata(metadataJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}
	
	return session, nil
}

// scanSessions 여러 세션 스캔
func (s *sessionStorage) scanSessions(rows *sql.Rows) ([]*models.Session, error) {
	var sessions []*models.Session
	
	for rows.Next() {
		session := &models.Session{}
		var startedAt, endedAt sql.NullTime
		var metadataJSON string
		var maxIdleTimeNS, maxLifetimeNS int64
		
		err := rows.Scan(
			&session.ID,
			&session.ProjectID,
			&session.ProcessID,
			&session.Status,
			&startedAt,
			&endedAt,
			&session.LastActive,
			&metadataJSON,
			&session.CommandCount,
			&session.BytesIn,
			&session.BytesOut,
			&session.ErrorCount,
			&maxIdleTimeNS,
			&maxLifetimeNS,
			&session.CreatedAt,
			&session.UpdatedAt,
			&session.Version,
		)
		
		if err != nil {
			return nil, storage.ConvertError(err, "scan session row", "sqlite")
		}
		
		// NULL 값 처리
		if startedAt.Valid {
			session.StartedAt = &startedAt.Time
		}
		if endedAt.Valid {
			session.EndedAt = &endedAt.Time
		}
		
		// Duration 변환
		session.MaxIdleTime = time.Duration(maxIdleTimeNS)
		session.MaxLifetime = time.Duration(maxLifetimeNS)
		
		// 메타데이터 역직렬화
		session.Metadata, err = s.unmarshalMetadata(metadataJSON)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
		
		sessions = append(sessions, session)
	}
	
	// rows 순회 중 에러 확인
	if err := rows.Err(); err != nil {
		return nil, storage.ConvertError(err, "scan session rows", "sqlite")
	}
	
	return sessions, nil
}