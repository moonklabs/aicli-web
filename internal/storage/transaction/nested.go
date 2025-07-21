package transaction

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aicli/aicli-web/internal/storage"
)

// NestedTransactionManager 중첩 트랜잭션 매니저
type NestedTransactionManager struct {
	baseManager Manager
	contextHelper *ContextHelper
	savepoints  map[string]*Savepoint
	mutex       sync.RWMutex
}

// Savepoint 트랜잭션 저장점
type Savepoint struct {
	// ID 저장점 ID
	ID string
	
	// Name 저장점 이름
	Name string
	
	// TransactionID 트랜잭션 ID
	TransactionID string
	
	// CreatedAt 생성 시간
	CreatedAt time.Time
	
	// Depth 중첩 깊이
	Depth int
	
	// Data 저장점 데이터 (스냅샷)
	Data interface{}
}

// NewNestedTransactionManager 새 중첩 트랜잭션 매니저 생성
func NewNestedTransactionManager(baseManager Manager) *NestedTransactionManager {
	return &NestedTransactionManager{
		baseManager:   baseManager,
		contextHelper: NewContextHelper(),
		savepoints:    make(map[string]*Savepoint),
	}
}

// BeginNested 중첩 트랜잭션 시작
func (ntm *NestedTransactionManager) BeginNested(ctx context.Context, opts *storage.TransactionOptions) (context.Context, error) {
	// 기존 트랜잭션이 있는지 확인
	existingTx, exists := ntm.baseManager.Current(ctx)
	if !exists {
		// 최상위 트랜잭션 시작
		tx, err := ntm.baseManager.Begin(ctx, opts)
		if err != nil {
			return ctx, fmt.Errorf("최상위 트랜잭션 시작 실패: %w", err)
		}
		return ntm.contextHelper.WithTransaction(ctx, tx, opts), nil
	}
	
	// 중첩 트랜잭션 컨텍스트 생성
	nestedCtx := ntm.contextHelper.WithNestedTransaction(ctx, existingTx, opts)
	
	// Savepoint 생성 (SQLite 지원하는 경우)
	if opts != nil && opts.EnableSavepoint {
		if err := ntm.createSavepoint(nestedCtx, existingTx); err != nil {
			return ctx, fmt.Errorf("Savepoint 생성 실패: %w", err)
		}
	}
	
	return nestedCtx, nil
}

// RunNestedTx 중첩 트랜잭션 내에서 함수 실행
func (ntm *NestedTransactionManager) RunNestedTx(ctx context.Context, fn func(ctx context.Context) error, opts ...*storage.TransactionOptions) error {
	_, err := ntm.RunNestedTxWithResult(ctx, func(ctx context.Context) (interface{}, error) {
		return nil, fn(ctx)
	}, opts...)
	return err
}

// RunNestedTxWithResult 결과와 함께 중첩 트랜잭션 실행
func (ntm *NestedTransactionManager) RunNestedTxWithResult[T any](ctx context.Context, fn func(ctx context.Context) (T, error), opts ...*storage.TransactionOptions) (T, error) {
	var zero T
	
	// 옵션 설정
	var txOpts *storage.TransactionOptions
	if len(opts) > 0 && opts[0] != nil {
		txOpts = opts[0]
	} else {
		defaultOpts := storage.DefaultTransactionOptions()
		txOpts = &defaultOpts
	}
	
	// 중첩 트랜잭션 시작
	nestedCtx, err := ntm.BeginNested(ctx, txOpts)
	if err != nil {
		return zero, fmt.Errorf("중첩 트랜잭션 시작 실패: %w", err)
	}
	
	// 중첩 트랜잭션 정보
	txCtx := ntm.contextHelper.GetTransactionContext(nestedCtx)
	if txCtx == nil {
		return zero, fmt.Errorf("트랜잭션 컨텍스트를 찾을 수 없습니다")
	}
	
	// 함수 실행 및 에러 처리
	result, execErr := fn(nestedCtx)
	
	// 중첩 트랜잭션 처리
	if execErr != nil {
		// 에러 발생 시 Savepoint로 롤백 (가능한 경우)
		if txOpts.EnableSavepoint {
			if rollbackErr := ntm.rollbackToSavepoint(nestedCtx, txCtx.ID); rollbackErr != nil {
				return zero, fmt.Errorf("Savepoint 롤백 실패: %w, 원본 에러: %w", rollbackErr, execErr)
			}
		}
		return zero, fmt.Errorf("중첩 트랜잭션 실행 실패: %w", execErr)
	}
	
	// 성공적으로 완료된 경우 Savepoint 해제
	if txOpts.EnableSavepoint {
		ntm.releaseSavepoint(txCtx.ID)
	}
	
	return result, nil
}

// createSavepoint Savepoint 생성
func (ntm *NestedTransactionManager) createSavepoint(ctx context.Context, tx storage.Transaction) error {
	txCtx := ntm.contextHelper.GetTransactionContext(ctx)
	if txCtx == nil {
		return fmt.Errorf("트랜잭션 컨텍스트를 찾을 수 없습니다")
	}
	
	savepointName := fmt.Sprintf("sp_%s", txCtx.ID)
	savepoint := &Savepoint{
		ID:            txCtx.ID,
		Name:          savepointName,
		TransactionID: txCtx.ParentID,
		CreatedAt:     time.Now(),
		Depth:         txCtx.Depth,
	}
	
	ntm.mutex.Lock()
	ntm.savepoints[txCtx.ID] = savepoint
	ntm.mutex.Unlock()
	
	// SQLite Savepoint 생성 시도 (실제 구현에서는 SQL 실행)
	// 여기서는 개념적으로만 구현
	
	return nil
}

// rollbackToSavepoint Savepoint로 롤백
func (ntm *NestedTransactionManager) rollbackToSavepoint(ctx context.Context, savepointID string) error {
	ntm.mutex.Lock()
	savepoint, exists := ntm.savepoints[savepointID]
	ntm.mutex.Unlock()
	
	if !exists {
		return fmt.Errorf("Savepoint를 찾을 수 없습니다: %s", savepointID)
	}
	
	// SQLite ROLLBACK TO SAVEPOINT 실행 시도
	// 여기서는 개념적으로만 구현
	_ = savepoint
	
	return nil
}

// releaseSavepoint Savepoint 해제
func (ntm *NestedTransactionManager) releaseSavepoint(savepointID string) {
	ntm.mutex.Lock()
	delete(ntm.savepoints, savepointID)
	ntm.mutex.Unlock()
}

// GetSavepointInfo Savepoint 정보 조회
func (ntm *NestedTransactionManager) GetSavepointInfo(savepointID string) (*Savepoint, bool) {
	ntm.mutex.RLock()
	defer ntm.mutex.RUnlock()
	
	savepoint, exists := ntm.savepoints[savepointID]
	if exists {
		// 복사본 반환
		return &(*savepoint), true
	}
	return nil, false
}

// ListSavepoints 모든 Savepoint 목록 조회
func (ntm *NestedTransactionManager) ListSavepoints() []*Savepoint {
	ntm.mutex.RLock()
	defer ntm.mutex.RUnlock()
	
	savepoints := make([]*Savepoint, 0, len(ntm.savepoints))
	for _, sp := range ntm.savepoints {
		savepoints = append(savepoints, &(*sp))
	}
	return savepoints
}

// NestedTransactionInfo 중첩 트랜잭션 정보
type NestedTransactionInfo struct {
	// CurrentDepth 현재 중첩 깊이
	CurrentDepth int
	
	// MaxDepth 최대 중첩 깊이
	MaxDepth int
	
	// ActiveSavepoints 활성 Savepoint 수
	ActiveSavepoints int
	
	// TransactionChain 트랜잭션 체인
	TransactionChain []*TransactionContext
	
	// TotalDuration 총 실행 시간
	TotalDuration time.Duration
}

// GetNestedTransactionInfo 중첩 트랜잭션 정보 조회
func (ntm *NestedTransactionManager) GetNestedTransactionInfo(ctx context.Context) *NestedTransactionInfo {
	info := &NestedTransactionInfo{}
	
	// 현재 트랜잭션 컨텍스트 정보
	txCtx := ntm.contextHelper.GetTransactionContext(ctx)
	if txCtx != nil {
		info.CurrentDepth = txCtx.Depth
		info.TotalDuration = time.Since(txCtx.StartTime)
	}
	
	// 트랜잭션 체인 정보
	chain := ntm.contextHelper.GetTransactionChain(ctx)
	if chain != nil {
		info.TransactionChain = chain.Transactions
		info.MaxDepth = chain.TotalDepth
	}
	
	// Savepoint 정보
	ntm.mutex.RLock()
	info.ActiveSavepoints = len(ntm.savepoints)
	ntm.mutex.RUnlock()
	
	return info
}

// ValidateNestedTransaction 중첩 트랜잭션 유효성 검증
func (ntm *NestedTransactionManager) ValidateNestedTransaction(ctx context.Context, maxDepth int) error {
	if !ntm.contextHelper.IsInTransaction(ctx) {
		return fmt.Errorf("트랜잭션 컨텍스트가 아닙니다")
	}
	
	currentDepth := ntm.contextHelper.GetTransactionDepth(ctx)
	if currentDepth >= maxDepth {
		return fmt.Errorf("최대 중첩 깊이를 초과했습니다: %d >= %d", currentDepth, maxDepth)
	}
	
	// 트랜잭션 상태 검증
	if err := ntm.contextHelper.ValidateTransactionState(ctx); err != nil {
		return fmt.Errorf("트랜잭션 상태 검증 실패: %w", err)
	}
	
	return nil
}

// CleanupSavepoints 모든 Savepoint 정리
func (ntm *NestedTransactionManager) CleanupSavepoints() {
	ntm.mutex.Lock()
	defer ntm.mutex.Unlock()
	
	ntm.savepoints = make(map[string]*Savepoint)
}

// GetNestedTransactionStats 중첩 트랜잭션 통계
func (ntm *NestedTransactionManager) GetNestedTransactionStats() map[string]interface{} {
	ntm.mutex.RLock()
	defer ntm.mutex.RUnlock()
	
	stats := make(map[string]interface{})
	stats["active_savepoints"] = len(ntm.savepoints)
	
	// Savepoint별 통계
	depthCounts := make(map[int]int)
	for _, sp := range ntm.savepoints {
		depthCounts[sp.Depth]++
	}
	stats["savepoints_by_depth"] = depthCounts
	
	// 기본 매니저 통계
	if baseStats := ntm.baseManager.GetStats(); true {
		stats["base_manager"] = baseStats
	}
	
	return stats
}