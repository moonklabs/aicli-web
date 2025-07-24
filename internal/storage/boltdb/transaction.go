package boltdb

import (
	"context"
	"fmt"
	"time"

	"go.etcd.io/bbolt"

	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/storage"
)

// transaction BoltDB 트랜잭션 래퍼
type transaction struct {
	storage *Storage
	boltTx  *bbolt.Tx
	closed  bool
	
	// 트랜잭션용 스토리지들
	workspace *transactionWorkspaceStorage
	project   *transactionProjectStorage
	session   *transactionSessionStorage
	task      *transactionTaskStorage
}

// newTransaction 새 트랜잭션 래퍼 생성
func newTransaction(storage *Storage) *transaction {
	t := &transaction{
		storage: storage,
		closed:  false,
	}
	
	// 트랜잭션용 스토리지 생성
	t.workspace = &transactionWorkspaceStorage{tx: t}
	t.project = &transactionProjectStorage{tx: t}
	t.session = &transactionSessionStorage{tx: t}
	t.task = &transactionTaskStorage{tx: t}
	
	return t
}

// setBoltTx BoltDB 트랜잭션 설정
func (t *transaction) setBoltTx(boltTx *bbolt.Tx) {
	t.boltTx = boltTx
}

// reset 트랜잭션 초기화
func (t *transaction) reset() {
	t.boltTx = nil
}

// Commit 트랜잭션 커밋 (BoltDB는 자동 커밋)
func (t *transaction) Commit() error {
	if t.closed {
		return fmt.Errorf("transaction is already closed")
	}
	
	t.closed = true
	// BoltDB는 Update 함수 종료 시 자동으로 커밋됨
	return nil
}

// Rollback 트랜잭션 롤백 (BoltDB는 자동 롤백)
func (t *transaction) Rollback() error {
	if t.closed {
		return fmt.Errorf("transaction is already closed")
	}
	
	t.closed = true
	// BoltDB는 Update 함수에서 에러 발생 시 자동으로 롤백됨
	return nil
}

// Context 트랜잭션 컨텍스트 반환
func (t *transaction) Context() context.Context {
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

// getBoltTx BoltDB 트랜잭션 반환
func (t *transaction) getBoltTx() *bbolt.Tx {
	if t.boltTx == nil {
		panic("BoltDB transaction not set")
	}
	return t.boltTx
}

// 트랜잭션용 워크스페이스 스토리지

// transactionWorkspaceStorage 트랜잭션용 워크스페이스 스토리지
type transactionWorkspaceStorage struct {
	tx *transaction
}

// Create 워크스페이스 생성 (트랜잭션 내)
func (tws *transactionWorkspaceStorage) Create(ctx context.Context, workspace *models.Workspace) error {
	if tws.tx.closed {
		return fmt.Errorf("transaction is closed")
	}
	
	boltTx := tws.tx.getBoltTx()
	
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
	
	// 중복 검사
	exists, err := tws.ExistsByName(ctx, workspace.OwnerID, workspace.Name)
	if err != nil {
		return fmt.Errorf("중복 검사 실패: %w", err)
	}
	if exists {
		return storage.ErrAlreadyExists
	}
	
	// 직렬화
	serializer := &WorkspaceSerializer{}
	data, err := serializer.Marshal(workspace)
	if err != nil {
		return fmt.Errorf("워크스페이스 직렬화 실패: %w", err)
	}
	
	// 메인 버킷에 저장
	bucket := boltTx.Bucket([]byte(BucketWorkspaces))
	if bucket == nil {
		return fmt.Errorf("워크스페이스 버킷이 존재하지 않습니다")
	}
	
	if err := bucket.Put([]byte(workspace.ID), data); err != nil {
		return fmt.Errorf("워크스페이스 저장 실패: %w", err)
	}
	
	// 인덱스 업데이트
	indexMgr := newIndexManager(tws.tx.storage)
	
	// 소유자 인덱스
	if err := indexMgr.AddToIndex(boltTx, IndexWorkspaceOwner, workspace.OwnerID, workspace.ID); err != nil {
		return fmt.Errorf("소유자 인덱스 업데이트 실패: %w", err)
	}
	
	// 이름 인덱스 (유니크)
	nameKey := fmt.Sprintf("%s:%s", workspace.OwnerID, workspace.Name)
	if err := indexMgr.AddToIndex(boltTx, IndexWorkspaceName, nameKey, workspace.ID); err != nil {
		return fmt.Errorf("이름 인덱스 업데이트 실패: %w", err)
	}
	
	return nil
}

// GetByID ID로 워크스페이스 조회 (트랜잭션 내)
func (tws *transactionWorkspaceStorage) GetByID(ctx context.Context, id string) (*models.Workspace, error) {
	if tws.tx.closed {
		return nil, fmt.Errorf("transaction is closed")
	}
	
	boltTx := tws.tx.getBoltTx()
	
	bucket := boltTx.Bucket([]byte(BucketWorkspaces))
	if bucket == nil {
		return nil, fmt.Errorf("워크스페이스 버킷이 존재하지 않습니다")
	}
	
	data := bucket.Get([]byte(id))
	if data == nil {
		return nil, storage.ErrNotFound
	}
	
	serializer := &WorkspaceSerializer{}
	workspace, err := serializer.Unmarshal(data)
	if err != nil {
		return nil, fmt.Errorf("워크스페이스 역직렬화 실패: %w", err)
	}
	
	// Soft delete 체크
	if workspace.DeletedAt != nil {
		return nil, storage.ErrNotFound
	}
	
	return workspace, nil
}

// GetByOwnerID 소유자 ID로 워크스페이스 목록 조회 (트랜잭션 내)
func (tws *transactionWorkspaceStorage) GetByOwnerID(ctx context.Context, ownerID string, pagination *models.PaginationRequest) ([]*models.Workspace, int, error) {
	if tws.tx.closed {
		return nil, 0, fmt.Errorf("transaction is closed")
	}
	
	boltTx := tws.tx.getBoltTx()
	
	// 쿼리 헬퍼 사용
	queryHelper := newQueryHelper(tws.tx.storage, newIndexManager(tws.tx.storage))
	
	options := &QueryOptions{
		Page:  pagination.Page,
		Limit: pagination.Limit,
		Sort:  pagination.Sort,
		Order: SortOrder(pagination.Order),
		Filter: map[string]interface{}{
			"owner_id": ownerID,
		},
	}
	
	result, err := queryHelper.WorkspaceQuery(boltTx, options)
	if err != nil {
		return nil, 0, err
	}
	
	return result.Items, result.TotalCount, nil
}

// Update 워크스페이스 업데이트 (트랜잭션 내)
func (tws *transactionWorkspaceStorage) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	if tws.tx.closed {
		return fmt.Errorf("transaction is closed")
	}
	
	// 기존 워크스페이스 조회
	workspace, err := tws.GetByID(ctx, id)
	if err != nil {
		return err
	}
	
	oldName := workspace.Name
	
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
		}
	}
	
	// 타임스탬프 업데이트
	NormalizeTimestamps(workspace)
	
	// 이름이 변경된 경우 중복 검사
	if workspace.Name != oldName {
		exists, err := tws.ExistsByName(ctx, workspace.OwnerID, workspace.Name)
		if err != nil {
			return fmt.Errorf("중복 검사 실패: %w", err)
		}
		if exists {
			return storage.ErrAlreadyExists
		}
	}
	
	// 저장
	boltTx := tws.tx.getBoltTx()
	bucket := boltTx.Bucket([]byte(BucketWorkspaces))
	
	serializer := &WorkspaceSerializer{}
	data, err := serializer.Marshal(workspace)
	if err != nil {
		return fmt.Errorf("워크스페이스 직렬화 실패: %w", err)
	}
	
	if err := bucket.Put([]byte(id), data); err != nil {
		return fmt.Errorf("워크스페이스 업데이트 실패: %w", err)
	}
	
	// 이름 인덱스 업데이트
	if workspace.Name != oldName {
		indexMgr := newIndexManager(tws.tx.storage)
		
		oldNameKey := fmt.Sprintf("%s:%s", workspace.OwnerID, oldName)
		newNameKey := fmt.Sprintf("%s:%s", workspace.OwnerID, workspace.Name)
		
		// 기존 이름 인덱스 제거
		indexMgr.RemoveFromIndex(boltTx, IndexWorkspaceName, oldNameKey, id)
		
		// 새 이름 인덱스 추가
		if err := indexMgr.AddToIndex(boltTx, IndexWorkspaceName, newNameKey, id); err != nil {
			return fmt.Errorf("이름 인덱스 업데이트 실패: %w", err)
		}
	}
	
	return nil
}

// Delete 워크스페이스 삭제 (트랜잭션 내)
func (tws *transactionWorkspaceStorage) Delete(ctx context.Context, id string) error {
	if tws.tx.closed {
		return fmt.Errorf("transaction is closed")
	}
	
	// 기존 워크스페이스 조회
	workspace, err := tws.GetByID(ctx, id)
	if err != nil {
		return err
	}
	
	boltTx := tws.tx.getBoltTx()
	bucket := boltTx.Bucket([]byte(BucketWorkspaces))
	
	// Soft delete
	workspace.DeletedAt = &[]time.Time{time.Now()}[0]
	NormalizeTimestamps(workspace)
	
	serializer := &WorkspaceSerializer{}
	data, err := serializer.Marshal(workspace)
	if err != nil {
		return fmt.Errorf("워크스페이스 직렬화 실패: %w", err)
	}
	
	if err := bucket.Put([]byte(id), data); err != nil {
		return fmt.Errorf("워크스페이스 삭제 실패: %w", err)
	}
	
	// 인덱스 정리
	indexMgr := newIndexManager(tws.tx.storage)
	
	// 소유자 인덱스에서 제거
	indexMgr.RemoveFromIndex(boltTx, IndexWorkspaceOwner, workspace.OwnerID, id)
	
	// 이름 인덱스에서 제거
	nameKey := fmt.Sprintf("%s:%s", workspace.OwnerID, workspace.Name)
	indexMgr.RemoveFromIndex(boltTx, IndexWorkspaceName, nameKey, id)
	
	return nil
}

// List 전체 워크스페이스 목록 조회 (트랜잭션 내)
func (tws *transactionWorkspaceStorage) List(ctx context.Context, pagination *models.PaginationRequest) ([]*models.Workspace, int, error) {
	if tws.tx.closed {
		return nil, 0, fmt.Errorf("transaction is closed")
	}
	
	boltTx := tws.tx.getBoltTx()
	
	queryHelper := newQueryHelper(tws.tx.storage, newIndexManager(tws.tx.storage))
	
	options := &QueryOptions{
		Page:   pagination.Page,
		Limit:  pagination.Limit,
		Sort:   pagination.Sort,
		Order:  SortOrder(pagination.Order),
		Filter: map[string]interface{}{}, // 전체 조회
	}
	
	result, err := queryHelper.WorkspaceQuery(boltTx, options)
	if err != nil {
		return nil, 0, err
	}
	
	return result.Items, result.TotalCount, nil
}

// ExistsByName 이름으로 존재 여부 확인 (트랜잭션 내)
func (tws *transactionWorkspaceStorage) ExistsByName(ctx context.Context, ownerID, name string) (bool, error) {
	if tws.tx.closed {
		return false, fmt.Errorf("transaction is closed")
	}
	
	boltTx := tws.tx.getBoltTx()
	
	indexMgr := newIndexManager(tws.tx.storage)
	nameKey := fmt.Sprintf("%s:%s", ownerID, name)
	
	return indexMgr.ExistsInIndex(boltTx, IndexWorkspaceName, nameKey)
}

// 다른 스토리지 타입들의 트랜잭션 래퍼들 (기본 구현)

// transactionProjectStorage 트랜잭션용 프로젝트 스토리지 (기본 구현)
type transactionProjectStorage struct {
	tx *transaction
}

// 프로젝트 스토리지 인터페이스 구현 (스텁)
func (tps *transactionProjectStorage) Create(ctx context.Context, project *models.Project) error {
	return fmt.Errorf("project transaction methods not implemented yet")
}

func (tps *transactionProjectStorage) GetByID(ctx context.Context, id string) (*models.Project, error) {
	return nil, fmt.Errorf("project transaction methods not implemented yet")
}

func (tps *transactionProjectStorage) GetByWorkspaceID(ctx context.Context, workspaceID string, pagination *models.PaginationRequest) ([]*models.Project, int, error) {
	return nil, 0, fmt.Errorf("project transaction methods not implemented yet")
}

func (tps *transactionProjectStorage) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	return fmt.Errorf("project transaction methods not implemented yet")
}

func (tps *transactionProjectStorage) Delete(ctx context.Context, id string) error {
	return fmt.Errorf("project transaction methods not implemented yet")
}

func (tps *transactionProjectStorage) ExistsByName(ctx context.Context, workspaceID, name string) (bool, error) {
	return false, fmt.Errorf("project transaction methods not implemented yet")
}

func (tps *transactionProjectStorage) GetByPath(ctx context.Context, path string) (*models.Project, error) {
	return nil, fmt.Errorf("project transaction methods not implemented yet")
}

// transactionSessionStorage 트랜잭션용 세션 스토리지 (기본 구현)
type transactionSessionStorage struct {
	tx *transaction
}

// SessionStorage 인터페이스 구현 (스텁)
func (tss *transactionSessionStorage) Create(ctx context.Context, session *models.Session) error {
	return fmt.Errorf("session transaction methods not implemented yet")
}

func (tss *transactionSessionStorage) GetByID(ctx context.Context, id string) (*models.Session, error) {
	return nil, fmt.Errorf("session transaction methods not implemented yet")
}

func (tss *transactionSessionStorage) List(ctx context.Context, filter *models.SessionFilter, paging *models.PaginationRequest) (*models.PaginationResponse, error) {
	return nil, fmt.Errorf("session transaction methods not implemented yet")
}

func (tss *transactionSessionStorage) Update(ctx context.Context, session *models.Session) error {
	return fmt.Errorf("session transaction methods not implemented yet")
}

func (tss *transactionSessionStorage) Delete(ctx context.Context, id string) error {
	return fmt.Errorf("session transaction methods not implemented yet")
}

func (tss *transactionSessionStorage) GetActiveCount(ctx context.Context, projectID string) (int64, error) {
	return 0, fmt.Errorf("session transaction methods not implemented yet")
}

// transactionTaskStorage 트랜잭션용 태스크 스토리지 (기본 구현)
type transactionTaskStorage struct {
	tx *transaction
}

// TaskStorage 인터페이스 구현 (스텁)
func (tts *transactionTaskStorage) Create(ctx context.Context, task *models.Task) error {
	return fmt.Errorf("task transaction methods not implemented yet")
}

func (tts *transactionTaskStorage) GetByID(ctx context.Context, id string) (*models.Task, error) {
	return nil, fmt.Errorf("task transaction methods not implemented yet")
}

func (tts *transactionTaskStorage) List(ctx context.Context, filter *models.TaskFilter, paging *models.PaginationRequest) ([]*models.Task, int, error) {
	return nil, 0, fmt.Errorf("task transaction methods not implemented yet")
}

func (tts *transactionTaskStorage) Update(ctx context.Context, task *models.Task) error {
	return fmt.Errorf("task transaction methods not implemented yet")
}

func (tts *transactionTaskStorage) Delete(ctx context.Context, id string) error {
	return fmt.Errorf("task transaction methods not implemented yet")
}

func (tts *transactionTaskStorage) GetBySessionID(ctx context.Context, sessionID string, paging *models.PaginationRequest) ([]*models.Task, int, error) {
	return nil, 0, fmt.Errorf("task transaction methods not implemented yet")
}

func (tts *transactionTaskStorage) GetActiveCount(ctx context.Context, sessionID string) (int64, error) {
	return 0, fmt.Errorf("task transaction methods not implemented yet")
}