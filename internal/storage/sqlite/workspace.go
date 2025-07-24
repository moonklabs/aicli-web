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

// workspaceStorage 워크스페이스 스토리지 SQLite 구현
type workspaceStorage struct {
	storage *Storage
}

// newWorkspaceStorage 새 워크스페이스 스토리지 생성
func newWorkspaceStorage(s *Storage) *workspaceStorage {
	return &workspaceStorage{storage: s}
}

const (
	// 워크스페이스 조회 쿼리
	selectWorkspaceQuery = `
		SELECT id, name, project_path, status, owner_id, claude_key, 
		       active_tasks, created_at, updated_at, deleted_at
		FROM workspaces
	`
	
	// 워크스페이스 삽입 쿼리
	insertWorkspaceQuery = `
		INSERT INTO workspaces (id, name, project_path, status, owner_id, claude_key, 
		                       active_tasks, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	
	// 워크스페이스 업데이트 쿼리 베이스
	updateWorkspaceQueryBase = `UPDATE workspaces SET updated_at = ? `
	
	// 워크스페이스 삭제 쿼리 (soft delete)
	deleteWorkspaceQuery = `
		UPDATE workspaces 
		SET deleted_at = ?, updated_at = ?
		WHERE id = ? AND deleted_at IS NULL
	`
	
	// 워크스페이스 존재 여부 확인 쿼리
	existsWorkspaceByNameQuery = `
		SELECT 1 FROM workspaces 
		WHERE owner_id = ? AND name = ? AND deleted_at IS NULL
	`
	
	// 카운트 쿼리
	countWorkspacesQuery = `SELECT COUNT(*) FROM workspaces`
)

// Create 새 워크스페이스 생성
func (w *workspaceStorage) Create(ctx context.Context, workspace *models.Workspace) error {
	now := time.Now()
	
	// 기본값 설정
	if workspace.Status == "" {
		workspace.Status = models.WorkspaceStatusActive
	}
	workspace.CreatedAt = now
	workspace.UpdatedAt = now
	
	// 워크스페이스 삽입
	_, err := w.storage.execContext(ctx, insertWorkspaceQuery,
		workspace.ID,
		workspace.Name,
		workspace.ProjectPath,
		workspace.Status,
		workspace.OwnerID,
		workspace.ClaudeKey,
		workspace.ActiveTasks,
		workspace.CreatedAt,
		workspace.UpdatedAt,
	)
	
	if err != nil {
		return storage.ConvertError(err, "create workspace", "sqlite")
	}
	
	return nil
}

// GetByID ID로 워크스페이스 조회
func (w *workspaceStorage) GetByID(ctx context.Context, id string) (*models.Workspace, error) {
	query := selectWorkspaceQuery + ` WHERE id = ? AND deleted_at IS NULL`
	
	row := w.storage.queryRowContext(ctx, query, id)
	
	workspace, err := w.scanWorkspace(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrNotFound
		}
		return nil, storage.ConvertError(err, "get workspace by id", "sqlite")
	}
	
	return workspace, nil
}

// GetByName 이름으로 워크스페이스 조회
func (w *workspaceStorage) GetByName(ctx context.Context, ownerID, name string) (*models.Workspace, error) {
	query := selectWorkspaceQuery + ` WHERE owner_id = ? AND name = ? AND deleted_at IS NULL`
	
	row := w.storage.queryRowContext(ctx, query, ownerID, name)
	
	workspace, err := w.scanWorkspace(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrNotFound
		}
		return nil, storage.ConvertError(err, "get workspace by name", "sqlite")
	}
	
	return workspace, nil
}

// GetByOwnerID 소유자 ID로 워크스페이스 목록 조회
func (w *workspaceStorage) GetByOwnerID(ctx context.Context, ownerID string, pagination *models.PaginationRequest) ([]*models.Workspace, int, error) {
	if pagination == nil {
		pagination = &models.PaginationRequest{Page: 1, Limit: 20, Sort: "created_at", Order: "desc"}
	}
	pagination.Normalize()
	
	// 전체 카운트 조회
	countQuery := countWorkspacesQuery + ` WHERE owner_id = ? AND deleted_at IS NULL`
	var total int
	err := w.storage.queryRowContext(ctx, countQuery, ownerID).Scan(&total)
	if err != nil {
		return nil, 0, storage.ConvertError(err, "count workspaces by owner", "sqlite")
	}
	
	// 데이터 조회
	query := selectWorkspaceQuery + `
		WHERE owner_id = ? AND deleted_at IS NULL 
		ORDER BY ` + pagination.Sort + ` ` + pagination.Order + `
		LIMIT ? OFFSET ?
	`
	
	rows, err := w.storage.queryContext(ctx, query, 
		ownerID, pagination.Limit, pagination.GetOffset())
	if err != nil {
		return nil, 0, storage.ConvertError(err, "get workspaces by owner", "sqlite")
	}
	defer rows.Close()
	
	workspaces, err := w.scanWorkspaces(rows)
	if err != nil {
		return nil, 0, err
	}
	
	return workspaces, total, nil
}

// Update 워크스페이스 업데이트
func (w *workspaceStorage) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return storage.ErrInvalidInput
	}
	
	// 허용된 업데이트 필드들
	allowedFields := map[string]bool{
		"name":         true,
		"project_path": true,
		"status":       true,
		"claude_key":   true,
		"active_tasks": true,
	}
	
	// 동적 UPDATE 쿼리 생성
	var setParts []string
	var args []interface{}
	
	// updated_at은 항상 업데이트
	setParts = append(setParts, "updated_at = ?")
	args = append(args, time.Now())
	
	// 업데이트할 필드들 처리
	for field, value := range updates {
		if !allowedFields[field] {
			return fmt.Errorf("field '%s' is not allowed for update", field)
		}
		
		setParts = append(setParts, field+" = ?")
		args = append(args, value)
	}
	
	// 최종 쿼리 생성
	query := updateWorkspaceQueryBase + strings.Join(setParts, ", ") + 
		" WHERE id = ? AND deleted_at IS NULL"
	args = append(args, id)
	
	result, err := w.storage.execContext(ctx, query, args...)
	if err != nil {
		return storage.ConvertError(err, "update workspace", "sqlite")
	}
	
	// 업데이트된 행이 있는지 확인
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return storage.ConvertError(err, "check rows affected", "sqlite")
	}
	
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}
	
	return nil
}

// Delete 워크스페이스 삭제 (soft delete)
func (w *workspaceStorage) Delete(ctx context.Context, id string) error {
	now := time.Now()
	
	result, err := w.storage.execContext(ctx, deleteWorkspaceQuery, now, now, id)
	if err != nil {
		return storage.ConvertError(err, "delete workspace", "sqlite")
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

// List 전체 워크스페이스 목록 조회 (관리자용)
func (w *workspaceStorage) List(ctx context.Context, pagination *models.PaginationRequest) ([]*models.Workspace, int, error) {
	if pagination == nil {
		pagination = &models.PaginationRequest{Page: 1, Limit: 20, Sort: "created_at", Order: "desc"}
	}
	pagination.Normalize()
	
	// 전체 카운트 조회
	countQuery := countWorkspacesQuery + ` WHERE deleted_at IS NULL`
	var total int
	err := w.storage.queryRowContext(ctx, countQuery).Scan(&total)
	if err != nil {
		return nil, 0, storage.ConvertError(err, "count all workspaces", "sqlite")
	}
	
	// 데이터 조회
	query := selectWorkspaceQuery + `
		WHERE deleted_at IS NULL 
		ORDER BY ` + pagination.Sort + ` ` + pagination.Order + `
		LIMIT ? OFFSET ?
	`
	
	rows, err := w.storage.queryContext(ctx, query, pagination.Limit, pagination.GetOffset())
	if err != nil {
		return nil, 0, storage.ConvertError(err, "list all workspaces", "sqlite")
	}
	defer rows.Close()
	
	workspaces, err := w.scanWorkspaces(rows)
	if err != nil {
		return nil, 0, err
	}
	
	return workspaces, total, nil
}

// ExistsByName 이름으로 존재 여부 확인
func (w *workspaceStorage) ExistsByName(ctx context.Context, ownerID, name string) (bool, error) {
	var exists int
	err := w.storage.queryRowContext(ctx, existsWorkspaceByNameQuery, ownerID, name).Scan(&exists)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, storage.ConvertError(err, "check workspace exists by name", "sqlite")
	}
	
	return exists == 1, nil
}

// CountByOwner 소유자별 워크스페이스 개수 조회
func (w *workspaceStorage) CountByOwner(ctx context.Context, ownerID string) (int, error) {
	query := countWorkspacesQuery + ` WHERE owner_id = ? AND deleted_at IS NULL`
	
	var count int
	err := w.storage.queryRowContext(ctx, query, ownerID).Scan(&count)
	if err != nil {
		return 0, storage.ConvertError(err, "count workspaces by owner", "sqlite")
	}
	
	return count, nil
}

// scanWorkspace 단일 워크스페이스 스캔
func (w *workspaceStorage) scanWorkspace(row *sql.Row) (*models.Workspace, error) {
	workspace := &models.Workspace{}
	var deletedAt sql.NullTime
	var claudeKey sql.NullString
	
	err := row.Scan(
		&workspace.ID,
		&workspace.Name,
		&workspace.ProjectPath,
		&workspace.Status,
		&workspace.OwnerID,
		&claudeKey,
		&workspace.ActiveTasks,
		&workspace.CreatedAt,
		&workspace.UpdatedAt,
		&deletedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	// NULL 값 처리
	if claudeKey.Valid {
		workspace.ClaudeKey = claudeKey.String
	}
	if deletedAt.Valid {
		workspace.DeletedAt = &deletedAt.Time
	}
	
	return workspace, nil
}

// scanWorkspaces 여러 워크스페이스 스캔
func (w *workspaceStorage) scanWorkspaces(rows *sql.Rows) ([]*models.Workspace, error) {
	var workspaces []*models.Workspace
	
	for rows.Next() {
		workspace := &models.Workspace{}
		var deletedAt sql.NullTime
		var claudeKey sql.NullString
		
		err := rows.Scan(
			&workspace.ID,
			&workspace.Name,
			&workspace.ProjectPath,
			&workspace.Status,
			&workspace.OwnerID,
			&claudeKey,
			&workspace.ActiveTasks,
			&workspace.CreatedAt,
			&workspace.UpdatedAt,
			&deletedAt,
		)
		
		if err != nil {
			return nil, storage.ConvertError(err, "scan workspace row", "sqlite")
		}
		
		// NULL 값 처리
		if claudeKey.Valid {
			workspace.ClaudeKey = claudeKey.String
		}
		if deletedAt.Valid {
			workspace.DeletedAt = &deletedAt.Time
		}
		
		workspaces = append(workspaces, workspace)
	}
	
	// rows 순회 중 에러 확인
	if err := rows.Err(); err != nil {
		return nil, storage.ConvertError(err, "scan workspace rows", "sqlite")
	}
	
	return workspaces, nil
}