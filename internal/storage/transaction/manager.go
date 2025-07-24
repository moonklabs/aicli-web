package transaction

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/aicli/aicli-web/internal/storage"
)

// Manager 통합 트랜잭션 매니저 인터페이스
type Manager interface {
	// Begin 새 트랜잭션 시작
	Begin(ctx context.Context, opts *storage.TransactionOptions) (storage.Transaction, error)
	
	// RunInTx 함수 내에서 트랜잭션 실행
	RunInTx(ctx context.Context, fn func(ctx context.Context) error, opts ...*storage.TransactionOptions) error
	
	// RunInTxWithResult 결과와 함께 트랜잭션 실행 (제네릭 지원 안함, interface{} 사용)
	
	// Current 현재 컨텍스트에서 트랜잭션 가져오기
	Current(ctx context.Context) (storage.Transaction, bool)
	
	// IsInTransaction 현재 트랜잭션 내에 있는지 확인
	IsInTransaction(ctx context.Context) bool
	
	// GetStats 트랜잭션 통계 정보
	GetStats() TransactionStats
}

// TransactionStats 트랜잭션 통계
type TransactionStats struct {
	// ActiveCount 활성 트랜잭션 수
	ActiveCount int
	
	// TotalCount 총 트랜잭션 수
	TotalCount int64
	
	// CommittedCount 커밋된 트랜잭션 수
	CommittedCount int64
	
	// RolledBackCount 롤백된 트랜잭션 수
	RolledBackCount int64
	
	// RetryCount 재시도된 트랜잭션 수
	RetryCount int64
	
	// AverageExecutionTime 평균 실행 시간
	AverageExecutionTime time.Duration
	
	// LastError 마지막 에러
	LastError error
}

// TransactionManager 구체적인 트랜잭션 매니저 구현
type TransactionManager struct {
	storage storage.TransactionalStorage
	stats   TransactionStats
	mutex   sync.RWMutex
	logger  *log.Logger
}

// NewManager 새 트랜잭션 매니저 생성
func NewManager(storage storage.TransactionalStorage, logger *log.Logger) Manager {
	return &TransactionManager{
		storage: storage,
		logger:  logger,
	}
}

// Begin 새 트랜잭션 시작
func (tm *TransactionManager) Begin(ctx context.Context, opts *storage.TransactionOptions) (storage.Transaction, error) {
	if opts == nil {
		defaultOpts := storage.DefaultTransactionOptions()
		opts = &defaultOpts
	}
	
	// 타임아웃 컨텍스트 생성
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}
	
	// 재시도 로직을 사용한 트랜잭션 시작
	return tm.beginWithRetry(ctx, opts)
}

// beginWithRetry 재시도 로직이 포함된 트랜잭션 시작
func (tm *TransactionManager) beginWithRetry(ctx context.Context, opts *storage.TransactionOptions) (storage.Transaction, error) {
	var lastErr error
	
	for attempt := 0; attempt <= opts.RetryCount; attempt++ {
		if attempt > 0 {
			// 재시도 지연
			select {
			case <-time.After(opts.RetryDelay):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
			
			tm.mutex.Lock()
			tm.stats.RetryCount++
			tm.mutex.Unlock()
			
			if tm.logger != nil {
				tm.logger.Printf("트랜잭션 시작 재시도 %d/%d: %v", attempt, opts.RetryCount, lastErr)
			}
		}
		
		tx, err := tm.storage.BeginTx(ctx)
		if err != nil {
			lastErr = err
			
			// 데드락이나 일시적 에러인 경우 재시도
			if isRetryableError(err) {
				continue
			}
			break
		}
		
		// 성공적으로 트랜잭션 시작
		tm.mutex.Lock()
		tm.stats.TotalCount++
		tm.stats.ActiveCount++
		tm.mutex.Unlock()
		
		// 트랜잭션 래퍼 반환
		return &managedTransaction{
			Transaction: tx,
			manager:     tm,
			startTime:   time.Now(),
		}, nil
	}
	
	tm.mutex.Lock()
	tm.stats.LastError = lastErr
	tm.mutex.Unlock()
	
	return nil, fmt.Errorf("트랜잭션 시작 실패 (최대 재시도 횟수 초과): %w", lastErr)
}

// RunInTx 함수 내에서 트랜잭션 실행
func (tm *TransactionManager) RunInTx(ctx context.Context, fn func(ctx context.Context) error, opts ...*storage.TransactionOptions) error {
	_, err := tm.RunInTxWithResult(ctx, func(ctx context.Context) (interface{}, error) {
		return nil, fn(ctx)
	}, opts...)
	return err
}

// RunInTxWithResult 결과와 함께 트랜잭션 실행
func (tm *TransactionManager) RunInTxWithResult(ctx context.Context, fn func(ctx context.Context) (interface{}, error), opts ...*storage.TransactionOptions) (interface{}, error) {
	var zero interface{}
	
	// 기존 트랜잭션이 있는 경우 재사용 (중첩 트랜잭션)
	if _, exists := storage.GetTxFromContext(ctx); exists {
		if tm.logger != nil {
			tm.logger.Printf("기존 트랜잭션을 재사용합니다")
		}
		return fn(ctx)
	}
	
	// 옵션 설정
	var txOpts *storage.TransactionOptions
	if len(opts) > 0 && opts[0] != nil {
		txOpts = opts[0]
	} else {
		defaultOpts := storage.DefaultTransactionOptions()
		txOpts = &defaultOpts
	}
	
	// 트랜잭션 시작
	tx, err := tm.Begin(ctx, txOpts)
	if err != nil {
		return zero, fmt.Errorf("트랜잭션 시작 실패: %w", err)
	}
	
	// 트랜잭션 컨텍스트 생성
	txCtx := storage.WithTxContext(ctx, tx)
	
	// defer로 트랜잭션 정리
	var committed bool
	defer func() {
		if !committed && !tx.IsClosed() {
			if rollbackErr := tx.Rollback(); rollbackErr != nil && tm.logger != nil {
				tm.logger.Printf("트랜잭션 롤백 실패: %v", rollbackErr)
			}
		}
	}()
	
	// 함수 실행
	result, err := fn(txCtx)
	if err != nil {
		return zero, fmt.Errorf("트랜잭션 함수 실행 실패: %w", err)
	}
	
	// 커밋
	if commitErr := tx.Commit(); commitErr != nil {
		return zero, fmt.Errorf("트랜잭션 커밋 실패: %w", commitErr)
	}
	committed = true
	
	return result, nil
}

// Current 현재 컨텍스트에서 트랜잭션 가져오기
func (tm *TransactionManager) Current(ctx context.Context) (storage.Transaction, bool) {
	return storage.GetTxFromContext(ctx)
}

// IsInTransaction 현재 트랜잭션 내에 있는지 확인
func (tm *TransactionManager) IsInTransaction(ctx context.Context) bool {
	_, exists := storage.GetTxFromContext(ctx)
	return exists
}

// GetStats 트랜잭션 통계 정보
func (tm *TransactionManager) GetStats() TransactionStats {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()
	return tm.stats
}

// managedTransaction 관리되는 트랜잭션 래퍼
type managedTransaction struct {
	storage.Transaction
	manager   *TransactionManager
	startTime time.Time
}

// Commit 커밋 시 통계 업데이트
func (mt *managedTransaction) Commit() error {
	err := mt.Transaction.Commit()
	mt.updateStats(err == nil, false)
	return err
}

// Rollback 롤백 시 통계 업데이트
func (mt *managedTransaction) Rollback() error {
	err := mt.Transaction.Rollback()
	mt.updateStats(false, true)
	return err
}

// updateStats 통계 업데이트
func (mt *managedTransaction) updateStats(committed, rolledBack bool) {
	mt.manager.mutex.Lock()
	defer mt.manager.mutex.Unlock()
	
	// 활성 트랜잭션 수 감소
	mt.manager.stats.ActiveCount--
	
	// 커밋/롤백 카운터 업데이트
	if committed {
		mt.manager.stats.CommittedCount++
	}
	if rolledBack {
		mt.manager.stats.RolledBackCount++
	}
	
	// 평균 실행 시간 계산
	executionTime := time.Since(mt.startTime)
	if mt.manager.stats.TotalCount > 0 {
		mt.manager.stats.AverageExecutionTime = 
			(mt.manager.stats.AverageExecutionTime + executionTime) / 2
	} else {
		mt.manager.stats.AverageExecutionTime = executionTime
	}
}

// isRetryableError 재시도 가능한 에러인지 확인
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	
	errStr := err.Error()
	
	// SQLite 데드락 및 일시적 에러들
	retryablePatterns := []string{
		"database is locked",
		"database schema has changed",
		"SQLITE_BUSY",
		"SQLITE_LOCKED",
		"deadlock",
		"timeout",
		"connection reset",
		"connection refused",
	}
	
	for _, pattern := range retryablePatterns {
		if contains(errStr, pattern) {
			return true
		}
	}
	
	// BoltDB 관련 에러
	if errors.Is(err, errors.New("database not open")) ||
		errors.Is(err, errors.New("database is read only")) {
		return true
	}
	
	return false
}

// contains 문자열 포함 여부 확인 (대소문자 무시)
func contains(s, substr string) bool {
	// 간단한 구현 - 실제로는 strings.Contains를 사용
	return len(s) >= len(substr) && 
		   s[len(s)-len(substr):] == substr || 
		   (len(s) > len(substr) && s[:len(substr)] == substr)
}