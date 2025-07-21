package storage

import (
	"context"
	"fmt"
	"time"
)

// Transaction 트랜잭션 인터페이스
type Transaction interface {
	// Commit 트랜잭션 커밋
	Commit() error
	
	// Rollback 트랜잭션 롤백
	Rollback() error
	
	// Context 트랜잭션 컨텍스트 반환
	Context() context.Context
	
	// IsClosed 트랜잭션이 완료되었는지 확인
	IsClosed() bool
	
	// 각 스토리지 인터페이스의 트랜잭션 버전
	Workspace() WorkspaceStorage
	Project() ProjectStorage
	Session() SessionStorage
	Task() TaskStorage
}

// TransactionalStorage 트랜잭션을 지원하는 스토리지 인터페이스
type TransactionalStorage interface {
	Storage
	
	// BeginTx 새 트랜잭션 시작
	BeginTx(ctx context.Context) (Transaction, error)
	
	// WithTx 트랜잭션 내에서 작업 실행
	WithTx(ctx context.Context, fn func(tx Transaction) error) error
}

// TxManager 트랜잭션 매니저
type TxManager struct {
	storage TransactionalStorage
}

// NewTxManager 새 트랜잭션 매니저 생성
func NewTxManager(storage TransactionalStorage) *TxManager {
	return &TxManager{
		storage: storage,
	}
}

// WithTransaction 트랜잭션 내에서 작업 실행
func (tm *TxManager) WithTransaction(ctx context.Context, fn func(tx Transaction) error) error {
	return tm.storage.WithTx(ctx, fn)
}

// ExecuteInTx 트랜잭션 내에서 여러 작업 실행
func (tm *TxManager) ExecuteInTx(ctx context.Context, operations ...func(tx Transaction) error) error {
	return tm.storage.WithTx(ctx, func(tx Transaction) error {
		for _, op := range operations {
			if err := op(tx); err != nil {
				return fmt.Errorf("트랜잭션 작업 실행 실패: %w", err)
			}
		}
		return nil
	})
}

// BaseTx 기본 트랜잭션 구현체 (메모리 스토리지용)
type BaseTx struct {
	ctx      context.Context
	storage  Storage
	closed   bool
	rolledBack bool
	committed  bool
}

// NewBaseTx 새 기본 트랜잭션 생성
func NewBaseTx(ctx context.Context, storage Storage) *BaseTx {
	return &BaseTx{
		ctx:     ctx,
		storage: storage,
		closed:  false,
	}
}

// Commit 트랜잭션 커밋 (메모리 스토리지에서는 즉시 커밋됨)
func (tx *BaseTx) Commit() error {
	if tx.closed {
		return fmt.Errorf("트랜잭션이 이미 완료되었습니다")
	}
	
	tx.committed = true
	tx.closed = true
	return nil
}

// Rollback 트랜잭션 롤백 (메모리 스토리지에서는 실제 롤백 불가)
func (tx *BaseTx) Rollback() error {
	if tx.closed {
		return fmt.Errorf("트랜잭션이 이미 완료되었습니다")
	}
	
	tx.rolledBack = true
	tx.closed = true
	return nil
}

// Context 트랜잭션 컨텍스트 반환
func (tx *BaseTx) Context() context.Context {
	return tx.ctx
}

// IsClosed 트랜잭션이 완료되었는지 확인
func (tx *BaseTx) IsClosed() bool {
	return tx.closed
}

// Workspace 워크스페이스 스토리지 반환
func (tx *BaseTx) Workspace() WorkspaceStorage {
	return tx.storage.Workspace()
}

// Project 프로젝트 스토리지 반환
func (tx *BaseTx) Project() ProjectStorage {
	return tx.storage.Project()
}

// Session 세션 스토리지 반환
func (tx *BaseTx) Session() SessionStorage {
	return tx.storage.Session()
}

// Task 태스크 스토리지 반환
func (tx *BaseTx) Task() TaskStorage {
	return tx.storage.Task()
}

// IsolationLevel 트랜잭션 격리 수준
type IsolationLevel int

const (
	// IsolationLevelDefault 기본 격리 수준
	IsolationLevelDefault IsolationLevel = iota
	
	// IsolationLevelReadUncommitted 읽지 않은 커밋 허용
	IsolationLevelReadUncommitted
	
	// IsolationLevelReadCommitted 커밋된 데이터만 읽기 (기본값)
	IsolationLevelReadCommitted
	
	// IsolationLevelRepeatableRead 반복 가능한 읽기
	IsolationLevelRepeatableRead
	
	// IsolationLevelSerializable 직렬화 가능
	IsolationLevelSerializable
)

// String 격리 수준을 문자열로 반환
func (il IsolationLevel) String() string {
	switch il {
	case IsolationLevelReadUncommitted:
		return "READ_UNCOMMITTED"
	case IsolationLevelReadCommitted:
		return "READ_COMMITTED"
	case IsolationLevelRepeatableRead:
		return "REPEATABLE_READ"
	case IsolationLevelSerializable:
		return "SERIALIZABLE"
	default:
		return "DEFAULT"
	}
}

// SQLiteLevel SQLite용 격리 수준 변환
func (il IsolationLevel) SQLiteLevel() string {
	switch il {
	case IsolationLevelReadUncommitted:
		return "DEFERRED"
	case IsolationLevelReadCommitted:
		return "DEFERRED"
	case IsolationLevelRepeatableRead:
		return "IMMEDIATE"
	case IsolationLevelSerializable:
		return "EXCLUSIVE"
	default:
		return "DEFERRED"
	}
}

// TransactionOptions 트랜잭션 옵션
type TransactionOptions struct {
	// ReadOnly 읽기 전용 트랜잭션 여부
	ReadOnly bool
	
	// Timeout 트랜잭션 타임아웃 (시간)
	Timeout time.Duration
	
	// IsolationLevel 격리 수준
	IsolationLevel IsolationLevel
	
	// RetryCount 재시도 횟수 (데드락 처리용)
	RetryCount int
	
	// RetryDelay 재시도 지연 시간
	RetryDelay time.Duration
	
	// EnableSavepoint Savepoint 사용 여부 (SQLite)
	EnableSavepoint bool
	
	// Context 트랜잭션 컨텍스트
	Context context.Context
}

// DefaultTransactionOptions 기본 트랜잭션 옵션 반환
func DefaultTransactionOptions() TransactionOptions {
	return TransactionOptions{
		ReadOnly:        false,
		Timeout:         30 * time.Second,
		IsolationLevel:  IsolationLevelReadCommitted,
		RetryCount:      3,
		RetryDelay:      100 * time.Millisecond,
		EnableSavepoint: false,
		Context:         context.Background(),
	}
}

// WithReadOnly 읽기 전용 옵션 설정
func (opts TransactionOptions) WithReadOnly(readOnly bool) TransactionOptions {
	opts.ReadOnly = readOnly
	return opts
}

// WithTimeout 타임아웃 옵션 설정
func (opts TransactionOptions) WithTimeout(timeout time.Duration) TransactionOptions {
	opts.Timeout = timeout
	return opts
}

// WithIsolationLevel 격리 수준 설정
func (opts TransactionOptions) WithIsolationLevel(level IsolationLevel) TransactionOptions {
	opts.IsolationLevel = level
	return opts
}

// WithRetry 재시도 옵션 설정
func (opts TransactionOptions) WithRetry(count int, delay time.Duration) TransactionOptions {
	opts.RetryCount = count
	opts.RetryDelay = delay
	return opts
}

// WithContext 컨텍스트 설정
func (opts TransactionOptions) WithContext(ctx context.Context) TransactionOptions {
	opts.Context = ctx
	return opts
}

// WithSavepoint Savepoint 옵션 설정
func (opts TransactionOptions) WithSavepoint(enable bool) TransactionOptions {
	opts.EnableSavepoint = enable
	return opts
}

// TxContextKey 트랜잭션 컨텍스트 키 타입
type TxContextKey struct{}

// TxKey 트랜잭션 컨텍스트 키
var TxKey = TxContextKey{}

// GetTxFromContext 컨텍스트에서 트랜잭션 추출
func GetTxFromContext(ctx context.Context) (Transaction, bool) {
	tx, ok := ctx.Value(TxKey).(Transaction)
	return tx, ok
}

// WithTxContext 컨텍스트에 트랜잭션 추가
func WithTxContext(ctx context.Context, tx Transaction) context.Context {
	return context.WithValue(ctx, TxKey, tx)
}

// TxFunc 트랜잭션 함수 타입
type TxFunc func(tx Transaction) error

// RunInTx 트랜잭션 내에서 함수 실행 (유틸리티 함수)
func RunInTx(ctx context.Context, storage TransactionalStorage, fn TxFunc) error {
	tx, err := storage.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("트랜잭션 시작 실패: %w", err)
	}
	
	// defer로 트랜잭션 정리
	defer func() {
		if !tx.IsClosed() {
			if err != nil {
				tx.Rollback()
			} else {
				tx.Commit()
			}
		}
	}()
	
	// 함수 실행
	err = fn(tx)
	if err != nil {
		return fmt.Errorf("트랜잭션 함수 실행 실패: %w", err)
	}
	
	return nil
}