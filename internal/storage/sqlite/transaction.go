package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/storage"
)

// transaction SQLite 트랜잭션 래퍼
type transaction struct {
	tx      *sql.Tx
	storage *Storage
	closed  bool
	
	// 트랜잭션 스토리지들
	workspace *transactionWorkspaceStorage
	project   *transactionProjectStorage
	session   *transactionSessionStorage
	task      *transactionTaskStorage
}

// newTransaction 새 트랜잭션 래퍼 생성
func newTransaction(tx *sql.Tx, storage *Storage) *transaction {
	t := &transaction{
		tx:      tx,
		storage: storage,
		closed:  false,
	}
	
	// 트랜잭션용 스토리지 생성
	t.workspace = &transactionWorkspaceStorage{tx: tx, storage: storage}
	t.project = &transactionProjectStorage{tx: tx, storage: storage}
	t.session = &transactionSessionStorage{tx: tx, storage: storage}
	t.task = &transactionTaskStorage{tx: tx, storage: storage}
	
	return t
}

// Commit 트랜잭션 커밋
func (t *transaction) Commit() error {
	if t.closed {
		return fmt.Errorf("transaction is already closed")
	}
	
	t.closed = true
	err := t.tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	return nil
}

// Rollback 트랜잭션 롤백
func (t *transaction) Rollback() error {
	if t.closed {
		return fmt.Errorf("transaction is already closed")
	}
	
	t.closed = true
	err := t.tx.Rollback()
	if err != nil {
		return fmt.Errorf("failed to rollback transaction: %w", err)
	}
	
	return nil
}

// Context 트랜잭션 컨텍스트 반환
func (t *transaction) Context() context.Context {
	// 트랜잭션 자체는 context를 가지지 않으므로 background context 반환
	// 실제 컨텍스트는 각 작업에서 전달받음
	return context.Background()
}

// IsClosed 트랜잭션 종료 여부 확인
func (t *transaction) IsClosed() bool {
	return t.closed
}

// Workspace 워크스페이스 스토리지 반환
func (t *transaction) Workspace() storage.WorkspaceStorage {
	return t.workspace
}

// Project 프로젝트 스토리지 반환
func (t *transaction) Project() storage.ProjectStorage {
	return t.project
}

// Session 세션 스토리지 반환
func (t *transaction) Session() storage.SessionStorage {
	return t.session
}

// Task 태스크 스토리지 반환
func (t *transaction) Task() storage.TaskStorage {
	return t.task
}

// transactionWorkspaceStorage 트랜잭션용 워크스페이스 스토리지
type transactionWorkspaceStorage struct {
	tx      *sql.Tx
	storage *Storage
}

// execContext 트랜잭션 내에서 실행
func (t *transactionWorkspaceStorage) execContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	stmt, err := t.tx.PrepareContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("prepare statement: %w", err)
	}
	defer stmt.Close()
	
	result, err := stmt.ExecContext(ctx, args...)
	if err != nil {
		return nil, storage.ConvertError(err, "exec in transaction", "sqlite")
	}
	
	return result, nil
}

// queryContext 트랜잭션 내에서 쿼리
func (t *transactionWorkspaceStorage) queryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	stmt, err := t.tx.PrepareContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("prepare statement: %w", err)
	}
	defer stmt.Close()
	
	rows, err := stmt.QueryContext(ctx, args...)
	if err != nil {
		return nil, storage.ConvertError(err, "query in transaction", "sqlite")
	}
	
	return rows, nil
}

// queryRowContext 트랜잭션 내에서 단일 행 쿼리
func (t *transactionWorkspaceStorage) queryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return t.tx.QueryRowContext(ctx, query, args...)
}

// WorkspaceStorage 인터페이스 구현을 위해 원본 workspace storage 메서드들을 위임
// 하지만 트랜잭션 컨텍스트에서 실행되도록 수정

// Create 워크스페이스 생성 (트랜잭션 내)
func (t *transactionWorkspaceStorage) Create(ctx context.Context, workspace *models.Workspace) error {
	// 트랜잭션용 임시 워크스페이스 스토리지 생성
	tempStorage := &workspaceStorage{storage: t.storage}
	
	// 실행 메서드를 트랜잭션용으로 오버라이드
	return t.createInTx(ctx, workspace, tempStorage)
}

// createInTx 트랜잭션 내에서 워크스페이스 생성
func (t *transactionWorkspaceStorage) createInTx(ctx context.Context, workspace *models.Workspace, ws *workspaceStorage) error {
	// 원본 workspaceStorage의 Create 로직을 복사하되, tx를 사용
	now := time.Now()
	
	if workspace.Status == "" {
		workspace.Status = models.WorkspaceStatusActive
	}
	workspace.CreatedAt = now
	workspace.UpdatedAt = now
	
	_, err := t.execContext(ctx, insertWorkspaceQuery,
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
	
	return err
}

// GetByID ID로 워크스페이스 조회 (트랜잭션 내)
func (t *transactionWorkspaceStorage) GetByID(ctx context.Context, id string) (*models.Workspace, error) {
	query := selectWorkspaceQuery + ` WHERE id = ? AND deleted_at IS NULL`
	
	row := t.queryRowContext(ctx, query, id)
	
	workspace, err := t.scanWorkspace(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrNotFound
		}
		return nil, storage.ConvertError(err, "get workspace by id in tx", "sqlite")
	}
	
	return workspace, nil
}

// scanWorkspace 워크스페이스 스캔 (트랜잭션용)
func (t *transactionWorkspaceStorage) scanWorkspace(row *sql.Row) (*models.Workspace, error) {
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

// 나머지 WorkspaceStorage 메서드들도 유사하게 구현
// 여기서는 간략화를 위해 주요 메서드들만 구현하고, 필요에 따라 추가

// GetByOwnerID 소유자 ID로 워크스페이스 목록 조회 (트랜잭션 내)
func (t *transactionWorkspaceStorage) GetByOwnerID(ctx context.Context, ownerID string, pagination *models.PaginationRequest) ([]*models.Workspace, int, error) {
	// 원본 스토리지와 동일한 로직이지만 트랜잭션 컨텍스트에서 실행
	// 구현 생략 (필요시 추가)
	return nil, 0, fmt.Errorf("not implemented in transaction context")
}

// Update 워크스페이스 업데이트 (트랜잭션 내)
func (t *transactionWorkspaceStorage) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	// 구현 생략 (필요시 추가)
	return fmt.Errorf("not implemented in transaction context")
}

// Delete 워크스페이스 삭제 (트랜잭션 내)
func (t *transactionWorkspaceStorage) Delete(ctx context.Context, id string) error {
	// 구현 생략 (필요시 추가)
	return fmt.Errorf("not implemented in transaction context")
}

// List 전체 워크스페이스 목록 조회 (트랜잭션 내)
func (t *transactionWorkspaceStorage) List(ctx context.Context, pagination *models.PaginationRequest) ([]*models.Workspace, int, error) {
	// 구현 생략 (필요시 추가)
	return nil, 0, fmt.Errorf("not implemented in transaction context")
}

// ExistsByName 이름으로 존재 여부 확인 (트랜잭션 내)
func (t *transactionWorkspaceStorage) ExistsByName(ctx context.Context, ownerID, name string) (bool, error) {
	var exists int
	err := t.queryRowContext(ctx, existsWorkspaceByNameQuery, ownerID, name).Scan(&exists)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, storage.ConvertError(err, "check workspace exists by name in tx", "sqlite")
	}
	
	return exists == 1, nil
}

// 다른 스토리지 타입들의 트랜잭션 래퍼들 (기본 구현)

// transactionProjectStorage 트랜잭션용 프로젝트 스토리지 (기본 구현)
type transactionProjectStorage struct {
	tx      *sql.Tx
	storage *Storage
}

// 필요한 인터페이스 메서드들을 구현해야 하지만, 여기서는 기본적인 스텁만 제공
func (t *transactionProjectStorage) Create(ctx context.Context, project *models.Project) error {
	return fmt.Errorf("project transaction methods not implemented yet")
}

func (t *transactionProjectStorage) GetByID(ctx context.Context, id string) (*models.Project, error) {
	return nil, fmt.Errorf("project transaction methods not implemented yet")
}

func (t *transactionProjectStorage) GetByWorkspaceID(ctx context.Context, workspaceID string, pagination *models.PaginationRequest) ([]*models.Project, int, error) {
	return nil, 0, fmt.Errorf("project transaction methods not implemented yet")
}

func (t *transactionProjectStorage) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	return fmt.Errorf("project transaction methods not implemented yet")
}

func (t *transactionProjectStorage) Delete(ctx context.Context, id string) error {
	return fmt.Errorf("project transaction methods not implemented yet")
}

func (t *transactionProjectStorage) ExistsByName(ctx context.Context, workspaceID, name string) (bool, error) {
	return false, fmt.Errorf("project transaction methods not implemented yet")
}

func (t *transactionProjectStorage) GetByPath(ctx context.Context, path string) (*models.Project, error) {
	return nil, fmt.Errorf("project transaction methods not implemented yet")
}

// transactionSessionStorage 트랜잭션용 세션 스토리지 (기본 구현)
type transactionSessionStorage struct {
	tx      *sql.Tx
	storage *Storage
}

func (t *transactionSessionStorage) Create(ctx context.Context, session *models.Session) error {
	return fmt.Errorf("session transaction methods not implemented yet")
}

func (t *transactionSessionStorage) GetByID(ctx context.Context, id string) (*models.Session, error) {
	return nil, fmt.Errorf("session transaction methods not implemented yet")
}

func (t *transactionSessionStorage) List(ctx context.Context, filter *models.SessionFilter, paging *models.PagingRequest) (*models.PagingResponse[*models.Session], error) {
	return nil, fmt.Errorf("session transaction methods not implemented yet")
}

func (t *transactionSessionStorage) Update(ctx context.Context, session *models.Session) error {
	return fmt.Errorf("session transaction methods not implemented yet")
}

func (t *transactionSessionStorage) Delete(ctx context.Context, id string) error {
	return fmt.Errorf("session transaction methods not implemented yet")
}

func (t *transactionSessionStorage) GetActiveCount(ctx context.Context, projectID string) (int64, error) {
	return 0, fmt.Errorf("session transaction methods not implemented yet")
}

// transactionTaskStorage 트랜잭션용 태스크 스토리지 (기본 구현)  
type transactionTaskStorage struct {
	tx      *sql.Tx
	storage *Storage
}

func (t *transactionTaskStorage) Create(ctx context.Context, task *models.Task) error {
	return fmt.Errorf("task transaction methods not implemented yet")
}

func (t *transactionTaskStorage) GetByID(ctx context.Context, id string) (*models.Task, error) {
	return nil, fmt.Errorf("task transaction methods not implemented yet")
}

func (t *transactionTaskStorage) List(ctx context.Context, filter *models.TaskFilter, paging *models.PagingRequest) ([]*models.Task, int, error) {
	return nil, 0, fmt.Errorf("task transaction methods not implemented yet")
}

func (t *transactionTaskStorage) Update(ctx context.Context, task *models.Task) error {
	return fmt.Errorf("task transaction methods not implemented yet")
}

func (t *transactionTaskStorage) Delete(ctx context.Context, id string) error {
	return fmt.Errorf("task transaction methods not implemented yet")
}

func (t *transactionTaskStorage) GetBySessionID(ctx context.Context, sessionID string, paging *models.PagingRequest) ([]*models.Task, int, error) {
	return nil, 0, fmt.Errorf("task transaction methods not implemented yet")
}

func (t *transactionTaskStorage) GetActiveCount(ctx context.Context, sessionID string) (int64, error) {
	return 0, fmt.Errorf("task transaction methods not implemented yet")
}