package transaction

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aicli/aicli-web/internal/storage"
)

// ContextKey 컨텍스트 키들
type ContextKey string

const (
	// TransactionKey 트랜잭션 키
	TransactionKey ContextKey = "transaction"
	
	// TransactionDepthKey 트랜잭션 중첩 깊이 키
	TransactionDepthKey ContextKey = "transaction_depth"
	
	// TransactionIDKey 트랜잭션 ID 키
	TransactionIDKey ContextKey = "transaction_id"
	
	// TransactionStartTimeKey 트랜잭션 시작 시간 키
	TransactionStartTimeKey ContextKey = "transaction_start_time"
)

// TransactionContext 트랜잭션 컨텍스트 정보
type TransactionContext struct {
	// Transaction 트랜잭션 인스턴스
	Transaction storage.Transaction
	
	// ID 트랜잭션 고유 ID
	ID string
	
	// Depth 중첩 깊이 (0부터 시작)
	Depth int
	
	// StartTime 시작 시간
	StartTime time.Time
	
	// ReadOnly 읽기 전용 여부
	ReadOnly bool
	
	// ParentID 부모 트랜잭션 ID (중첩된 경우)
	ParentID string
}

// ContextHelper 컨텍스트 헬퍼
type ContextHelper struct {
	mutex   sync.RWMutex
	counter int64
}

// NewContextHelper 새 컨텍스트 헬퍼 생성
func NewContextHelper() *ContextHelper {
	return &ContextHelper{}
}

// WithTransaction 컨텍스트에 트랜잭션 추가
func (ch *ContextHelper) WithTransaction(ctx context.Context, tx storage.Transaction, opts *storage.TransactionOptions) context.Context {
	ch.mutex.Lock()
	ch.counter++
	txID := fmt.Sprintf("tx_%d_%d", time.Now().UnixNano(), ch.counter)
	ch.mutex.Unlock()
	
	// 기존 트랜잭션 컨텍스트 확인
	existingTxCtx := ch.GetTransactionContext(ctx)
	depth := 0
	parentID := ""
	
	if existingTxCtx != nil {
		depth = existingTxCtx.Depth + 1
		parentID = existingTxCtx.ID
	}
	
	// 트랜잭션 컨텍스트 생성
	txContext := &TransactionContext{
		Transaction: tx,
		ID:          txID,
		Depth:       depth,
		StartTime:   time.Now(),
		ReadOnly:    opts != nil && opts.ReadOnly,
		ParentID:    parentID,
	}
	
	// 컨텍스트에 값들 설정
	ctx = context.WithValue(ctx, TransactionKey, tx)
	ctx = context.WithValue(ctx, TransactionIDKey, txID)
	ctx = context.WithValue(ctx, TransactionDepthKey, depth)
	ctx = context.WithValue(ctx, TransactionStartTimeKey, txContext.StartTime)
	ctx = context.WithValue(ctx, "transaction_context", txContext)
	
	return ctx
}

// GetTransaction 컨텍스트에서 트랜잭션 가져오기
func (ch *ContextHelper) GetTransaction(ctx context.Context) (storage.Transaction, bool) {
	tx, ok := ctx.Value(TransactionKey).(storage.Transaction)
	return tx, ok
}

// GetTransactionContext 트랜잭션 컨텍스트 정보 가져오기
func (ch *ContextHelper) GetTransactionContext(ctx context.Context) *TransactionContext {
	txCtx, ok := ctx.Value("transaction_context").(*TransactionContext)
	if !ok {
		return nil
	}
	return txCtx
}

// GetTransactionID 트랜잭션 ID 가져오기
func (ch *ContextHelper) GetTransactionID(ctx context.Context) string {
	id, ok := ctx.Value(TransactionIDKey).(string)
	if !ok {
		return ""
	}
	return id
}

// GetTransactionDepth 트랜잭션 중첩 깊이 가져오기
func (ch *ContextHelper) GetTransactionDepth(ctx context.Context) int {
	depth, ok := ctx.Value(TransactionDepthKey).(int)
	if !ok {
		return -1
	}
	return depth
}

// GetTransactionStartTime 트랜잭션 시작 시간 가져오기
func (ch *ContextHelper) GetTransactionStartTime(ctx context.Context) (time.Time, bool) {
	startTime, ok := ctx.Value(TransactionStartTimeKey).(time.Time)
	return startTime, ok
}

// IsInTransaction 트랜잭션 내에 있는지 확인
func (ch *ContextHelper) IsInTransaction(ctx context.Context) bool {
	_, exists := ch.GetTransaction(ctx)
	return exists
}

// IsNestedTransaction 중첩 트랜잭션인지 확인
func (ch *ContextHelper) IsNestedTransaction(ctx context.Context) bool {
	return ch.GetTransactionDepth(ctx) > 0
}

// GetTransactionDuration 트랜잭션 실행 시간 계산
func (ch *ContextHelper) GetTransactionDuration(ctx context.Context) time.Duration {
	startTime, ok := ch.GetTransactionStartTime(ctx)
	if !ok {
		return 0
	}
	return time.Since(startTime)
}

// ValidateTransactionState 트랜잭션 상태 검증
func (ch *ContextHelper) ValidateTransactionState(ctx context.Context) error {
	tx, exists := ch.GetTransaction(ctx)
	if !exists {
		return fmt.Errorf("트랜잭션이 컨텍스트에 없습니다")
	}
	
	if tx.IsClosed() {
		return fmt.Errorf("트랜잭션이 이미 종료되었습니다")
	}
	
	return nil
}

// WithNestedTransaction 중첩 트랜잭션 생성
func (ch *ContextHelper) WithNestedTransaction(ctx context.Context, tx storage.Transaction, opts *storage.TransactionOptions) context.Context {
	// 기존 트랜잭션이 있는지 확인
	existingTx, exists := ch.GetTransaction(ctx)
	if !exists {
		// 기존 트랜잭션이 없으면 일반 트랜잭션으로 처리
		return ch.WithTransaction(ctx, tx, opts)
	}
	
	// 중첩 트랜잭션의 경우 같은 트랜잭션을 재사용하되 깊이만 증가
	ch.mutex.Lock()
	ch.counter++
	txID := fmt.Sprintf("nested_tx_%d_%d", time.Now().UnixNano(), ch.counter)
	ch.mutex.Unlock()
	
	existingTxCtx := ch.GetTransactionContext(ctx)
	depth := 0
	parentID := ""
	
	if existingTxCtx != nil {
		depth = existingTxCtx.Depth + 1
		parentID = existingTxCtx.ID
	}
	
	// 중첩 트랜잭션 컨텍스트 생성 (실제 트랜잭션은 재사용)
	nestedTxContext := &TransactionContext{
		Transaction: existingTx, // 기존 트랜잭션 재사용
		ID:          txID,
		Depth:       depth,
		StartTime:   time.Now(),
		ReadOnly:    opts != nil && opts.ReadOnly,
		ParentID:    parentID,
	}
	
	// 컨텍스트 값 업데이트
	ctx = context.WithValue(ctx, TransactionIDKey, txID)
	ctx = context.WithValue(ctx, TransactionDepthKey, depth)
	ctx = context.WithValue(ctx, TransactionStartTimeKey, nestedTxContext.StartTime)
	ctx = context.WithValue(ctx, "transaction_context", nestedTxContext)
	
	return ctx
}

// TransactionChain 트랜잭션 체인 정보
type TransactionChain struct {
	// Transactions 트랜잭션 체인
	Transactions []*TransactionContext
	
	// TotalDepth 총 깊이
	TotalDepth int
	
	// TotalDuration 총 실행 시간
	TotalDuration time.Duration
}

// GetTransactionChain 트랜잭션 체인 정보 가져오기
func (ch *ContextHelper) GetTransactionChain(ctx context.Context) *TransactionChain {
	chain := &TransactionChain{
		Transactions: []*TransactionContext{},
	}
	
	currentTxCtx := ch.GetTransactionContext(ctx)
	if currentTxCtx == nil {
		return chain
	}
	
	// 현재 트랜잭션부터 시작해서 체인을 구성
	chain.Transactions = append(chain.Transactions, currentTxCtx)
	chain.TotalDepth = currentTxCtx.Depth + 1
	
	// 실행 시간 계산
	if !currentTxCtx.StartTime.IsZero() {
		chain.TotalDuration = time.Since(currentTxCtx.StartTime)
	}
	
	return chain
}

// LogTransactionInfo 트랜잭션 정보 로깅용 맵 반환
func (ch *ContextHelper) LogTransactionInfo(ctx context.Context) map[string]interface{} {
	info := make(map[string]interface{})
	
	txCtx := ch.GetTransactionContext(ctx)
	if txCtx == nil {
		info["in_transaction"] = false
		return info
	}
	
	info["in_transaction"] = true
	info["transaction_id"] = txCtx.ID
	info["depth"] = txCtx.Depth
	info["read_only"] = txCtx.ReadOnly
	info["start_time"] = txCtx.StartTime
	info["duration"] = time.Since(txCtx.StartTime)
	info["is_nested"] = txCtx.Depth > 0
	info["parent_id"] = txCtx.ParentID
	
	if tx := txCtx.Transaction; tx != nil {
		info["transaction_closed"] = tx.IsClosed()
	}
	
	return info
}

// 전역 컨텍스트 헬퍼 인스턴스
var DefaultContextHelper = NewContextHelper()

// 편의 함수들 (전역 헬퍼 사용)

// WithTx 컨텍스트에 트랜잭션 추가 (편의 함수)
func WithTx(ctx context.Context, tx storage.Transaction, opts *storage.TransactionOptions) context.Context {
	return DefaultContextHelper.WithTransaction(ctx, tx, opts)
}

// GetTx 컨텍스트에서 트랜잭션 가져오기 (편의 함수)
func GetTx(ctx context.Context) (storage.Transaction, bool) {
	return DefaultContextHelper.GetTransaction(ctx)
}

// GetTxContext 트랜잭션 컨텍스트 정보 가져오기 (편의 함수)
func GetTxContext(ctx context.Context) *TransactionContext {
	return DefaultContextHelper.GetTransactionContext(ctx)
}

// IsInTx 트랜잭션 내에 있는지 확인 (편의 함수)
func IsInTx(ctx context.Context) bool {
	return DefaultContextHelper.IsInTransaction(ctx)
}

// GetTxID 트랜잭션 ID 가져오기 (편의 함수)
func GetTxID(ctx context.Context) string {
	return DefaultContextHelper.GetTransactionID(ctx)
}