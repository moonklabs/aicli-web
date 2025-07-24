package transaction

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/aicli/aicli-web/internal/storage"
)

// TimeoutManager 타임아웃 매니저
type TimeoutManager struct {
	manager Manager
	logger  *log.Logger
	
	// 활성 트랜잭션 추적
	activeTransactions map[string]*ActiveTransaction
	mutex              sync.RWMutex
	
	// 타임아웃 감지 설정
	checkInterval    time.Duration
	defaultTimeout   time.Duration
	maxTimeout       time.Duration
	stopChan         chan struct{}
	wg               sync.WaitGroup
}

// ActiveTransaction 활성 트랜잭션 정보
type ActiveTransaction struct {
	ID        string
	StartTime time.Time
	Timeout   time.Duration
	Context   context.Context
	Cancel    context.CancelFunc
	
	// 데드락 감지용
	WaitingFor []string
	HeldLocks  []string
	
	// 재시도 정보
	RetryCount    int
	MaxRetries    int
	RetryDelay    time.Duration
	LastRetryTime time.Time
}

// NewTimeoutManager 새 타임아웃 매니저 생성
func NewTimeoutManager(manager Manager, logger *log.Logger) *TimeoutManager {
	tm := &TimeoutManager{
		manager:            manager,
		logger:             logger,
		activeTransactions: make(map[string]*ActiveTransaction),
		checkInterval:      time.Second * 5,
		defaultTimeout:     time.Second * 30,
		maxTimeout:         time.Minute * 10,
		stopChan:          make(chan struct{}),
	}
	
	// 백그라운드 타임아웃 검사 시작
	tm.startTimeoutChecker()
	
	return tm
}

// Close 타임아웃 매니저 종료
func (tm *TimeoutManager) Close() {
	close(tm.stopChan)
	tm.wg.Wait()
}

// BeginWithTimeout 타임아웃이 있는 트랜잭션 시작
func (tm *TimeoutManager) BeginWithTimeout(ctx context.Context, timeout time.Duration, opts *storage.TransactionOptions) (storage.Transaction, error) {
	if timeout <= 0 {
		timeout = tm.defaultTimeout
	}
	
	if timeout > tm.maxTimeout {
		timeout = tm.maxTimeout
	}
	
	// 타임아웃 컨텍스트 생성
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	
	// 트랜잭션 ID 생성
	txID := fmt.Sprintf("tx_%d", time.Now().UnixNano())
	
	// 활성 트랜잭션 등록
	activeTx := &ActiveTransaction{
		ID:         txID,
		StartTime:  time.Now(),
		Timeout:    timeout,
		Context:    timeoutCtx,
		Cancel:     cancel,
		RetryCount: 0,
		MaxRetries: 3,
		RetryDelay: time.Millisecond * 100,
	}
	
	if opts != nil {
		activeTx.MaxRetries = opts.RetryCount
		activeTx.RetryDelay = opts.RetryDelay
	}
	
	tm.mutex.Lock()
	tm.activeTransactions[txID] = activeTx
	tm.mutex.Unlock()
	
	// 실제 트랜잭션 시작
	tx, err := tm.manager.Begin(timeoutCtx, opts)
	if err != nil {
		// 실패 시 정리
		tm.removeActiveTransaction(txID)
		cancel()
		return nil, fmt.Errorf("트랜잭션 시작 실패: %w", err)
	}
	
	// 타임아웃 트랜잭션 래퍼 반환
	return &timeoutTransaction{
		Transaction: tx,
		manager:     tm,
		activeTx:    activeTx,
	}, nil
}

// RunWithTimeout 타임아웃이 있는 트랜잭션 실행
func (tm *TimeoutManager) RunWithTimeout(ctx context.Context, timeout time.Duration, fn func(ctx context.Context) error, opts ...*storage.TransactionOptions) error {
	_, err := tm.RunWithTimeoutAndResult(ctx, timeout, func(ctx context.Context) (interface{}, error) {
		return nil, fn(ctx)
	}, opts...)
	return err
}

// RunWithTimeoutAndResult 결과와 함께 타임아웃 트랜잭션 실행
func (tm *TimeoutManager) RunWithTimeoutAndResult(ctx context.Context, timeout time.Duration, fn func(ctx context.Context) (interface{}, error), opts ...*storage.TransactionOptions) (interface{}, error) {
	var zero interface{}
	
	tx, err := tm.BeginWithTimeout(ctx, timeout, getTransactionOptions(opts...))
	if err != nil {
		return zero, err
	}
	
	defer func() {
		if !tx.IsClosed() {
			tx.Rollback()
		}
	}()
	
	// 트랜잭션 컨텍스트로 함수 실행
	txCtx := storage.WithTxContext(ctx, tx)
	result, execErr := fn(txCtx)
	
	if execErr != nil {
		return zero, execErr
	}
	
	// 커밋
	if commitErr := tx.Commit(); commitErr != nil {
		return zero, fmt.Errorf("트랜잭션 커밋 실패: %w", commitErr)
	}
	
	return result, nil
}

// startTimeoutChecker 백그라운드 타임아웃 검사 시작
func (tm *TimeoutManager) startTimeoutChecker() {
	tm.wg.Add(1)
	go func() {
		defer tm.wg.Done()
		ticker := time.NewTicker(tm.checkInterval)
		defer ticker.Stop()
		
		for {
			select {
			case <-ticker.C:
				tm.checkTimeouts()
			case <-tm.stopChan:
				return
			}
		}
	}()
}

// checkTimeouts 타임아웃된 트랜잭션 검사 및 처리
func (tm *TimeoutManager) checkTimeouts() {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()
	
	now := time.Now()
	timedOutTxs := make([]*ActiveTransaction, 0)
	
	for id, activeTx := range tm.activeTransactions {
		elapsed := now.Sub(activeTx.StartTime)
		if elapsed > activeTx.Timeout {
			timedOutTxs = append(timedOutTxs, activeTx)
			delete(tm.activeTransactions, id)
		}
	}
	
	// 타임아웃된 트랜잭션들 처리
	for _, activeTx := range timedOutTxs {
		tm.handleTimeout(activeTx)
	}
}

// handleTimeout 타임아웃된 트랜잭션 처리
func (tm *TimeoutManager) handleTimeout(activeTx *ActiveTransaction) {
	if tm.logger != nil {
		tm.logger.Printf("트랜잭션 타임아웃 감지: %s (경과시간: %v)", 
			activeTx.ID, time.Since(activeTx.StartTime))
	}
	
	// 컨텍스트 취소
	if activeTx.Cancel != nil {
		activeTx.Cancel()
	}
}

// removeActiveTransaction 활성 트랜잭션 제거
func (tm *TimeoutManager) removeActiveTransaction(txID string) {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()
	delete(tm.activeTransactions, txID)
}

// GetActiveTransactionInfo 활성 트랜잭션 정보 조회
func (tm *TimeoutManager) GetActiveTransactionInfo() []*ActiveTransaction {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()
	
	result := make([]*ActiveTransaction, 0, len(tm.activeTransactions))
	for _, activeTx := range tm.activeTransactions {
		// 복사본 생성
		copy := *activeTx
		result = append(result, &copy)
	}
	return result
}

// timeoutTransaction 타임아웃 트랜잭션 래퍼
type timeoutTransaction struct {
	storage.Transaction
	manager  *TimeoutManager
	activeTx *ActiveTransaction
}

// Commit 커밋 시 활성 트랜잭션 정리
func (tt *timeoutTransaction) Commit() error {
	err := tt.Transaction.Commit()
	tt.manager.removeActiveTransaction(tt.activeTx.ID)
	if tt.activeTx.Cancel != nil {
		tt.activeTx.Cancel()
	}
	return err
}

// Rollback 롤백 시 활성 트랜잭션 정리
func (tt *timeoutTransaction) Rollback() error {
	err := tt.Transaction.Rollback()
	tt.manager.removeActiveTransaction(tt.activeTx.ID)
	if tt.activeTx.Cancel != nil {
		tt.activeTx.Cancel()
	}
	return err
}

// DeadlockDetector 데드락 감지기
type DeadlockDetector struct {
	mutex        sync.RWMutex
	transactions map[string]*ActiveTransaction
	logger       *log.Logger
	
	// 데드락 감지 설정
	checkInterval time.Duration
	stopChan      chan struct{}
	wg            sync.WaitGroup
}

// NewDeadlockDetector 새 데드락 감지기 생성
func NewDeadlockDetector(logger *log.Logger) *DeadlockDetector {
	dd := &DeadlockDetector{
		transactions:  make(map[string]*ActiveTransaction),
		logger:        logger,
		checkInterval: time.Second * 2,
		stopChan:      make(chan struct{}),
	}
	
	dd.startDeadlockDetection()
	return dd
}

// Close 데드락 감지기 종료
func (dd *DeadlockDetector) Close() {
	close(dd.stopChan)
	dd.wg.Wait()
}

// RegisterTransaction 트랜잭션 등록
func (dd *DeadlockDetector) RegisterTransaction(activeTx *ActiveTransaction) {
	dd.mutex.Lock()
	defer dd.mutex.Unlock()
	dd.transactions[activeTx.ID] = activeTx
}

// UnregisterTransaction 트랜잭션 등록 해제
func (dd *DeadlockDetector) UnregisterTransaction(txID string) {
	dd.mutex.Lock()
	defer dd.mutex.Unlock()
	delete(dd.transactions, txID)
}

// UpdateWaitingFor 대기 중인 리소스 업데이트
func (dd *DeadlockDetector) UpdateWaitingFor(txID string, waitingFor []string) {
	dd.mutex.Lock()
	defer dd.mutex.Unlock()
	
	if activeTx, exists := dd.transactions[txID]; exists {
		activeTx.WaitingFor = waitingFor
	}
}

// UpdateHeldLocks 보유 중인 락 업데이트
func (dd *DeadlockDetector) UpdateHeldLocks(txID string, heldLocks []string) {
	dd.mutex.Lock()
	defer dd.mutex.Unlock()
	
	if activeTx, exists := dd.transactions[txID]; exists {
		activeTx.HeldLocks = heldLocks
	}
}

// startDeadlockDetection 데드락 감지 시작
func (dd *DeadlockDetector) startDeadlockDetection() {
	dd.wg.Add(1)
	go func() {
		defer dd.wg.Done()
		ticker := time.NewTicker(dd.checkInterval)
		defer ticker.Stop()
		
		for {
			select {
			case <-ticker.C:
				dd.detectDeadlocks()
			case <-dd.stopChan:
				return
			}
		}
	}()
}

// detectDeadlocks 데드락 감지
func (dd *DeadlockDetector) detectDeadlocks() {
	dd.mutex.RLock()
	defer dd.mutex.RUnlock()
	
	// 간단한 데드락 감지 알고리즘
	// 실제로는 더 복잡한 그래프 순환 감지 알고리즘이 필요
	
	for txID, activeTx := range dd.transactions {
		for _, waitingRes := range activeTx.WaitingFor {
			// 이 리소스를 보유한 다른 트랜잭션 찾기
			for otherTxID, otherActiveTx := range dd.transactions {
				if txID == otherTxID {
					continue
				}
				
				// 상호 대기 상황 체크
				if dd.hasResource(otherActiveTx.HeldLocks, waitingRes) &&
				   dd.hasWaitingCycle(txID, otherTxID, dd.transactions) {
					
					if dd.logger != nil {
						dd.logger.Printf("데드락 감지: %s <-> %s", txID, otherTxID)
					}
					
					// 데드락 해결 (더 오래된 트랜잭션을 선택적으로 중단)
					dd.resolveDeadlock(activeTx, otherActiveTx)
					return
				}
			}
		}
	}
}

// hasResource 리소스 보유 여부 확인
func (dd *DeadlockDetector) hasResource(heldLocks []string, resource string) bool {
	for _, lock := range heldLocks {
		if lock == resource {
			return true
		}
	}
	return false
}

// hasWaitingCycle 대기 순환 확인 (간단한 구현)
func (dd *DeadlockDetector) hasWaitingCycle(tx1, tx2 string, transactions map[string]*ActiveTransaction) bool {
	// 실제로는 DFS나 유사한 알고리즘으로 순환 감지
	// 여기서는 간단하게 상호 대기만 확인
	
	tx1Info := transactions[tx1]
	tx2Info := transactions[tx2]
	
	if tx1Info == nil || tx2Info == nil {
		return false
	}
	
	// tx2가 tx1의 리소스를 대기하는지 확인
	for _, waiting := range tx2Info.WaitingFor {
		if dd.hasResource(tx1Info.HeldLocks, waiting) {
			return true
		}
	}
	
	return false
}

// resolveDeadlock 데드락 해결
func (dd *DeadlockDetector) resolveDeadlock(tx1, tx2 *ActiveTransaction) {
	// 더 오래된 트랜잭션을 종료 (단순한 정책)
	var victimTx *ActiveTransaction
	
	if tx1.StartTime.Before(tx2.StartTime) {
		victimTx = tx2
	} else {
		victimTx = tx1
	}
	
	if dd.logger != nil {
		dd.logger.Printf("데드락 해결: 트랜잭션 %s 종료", victimTx.ID)
	}
	
	// 희생 트랜잭션 취소
	if victimTx.Cancel != nil {
		victimTx.Cancel()
	}
}

// getTransactionOptions 옵션 헬퍼 함수
func getTransactionOptions(opts ...*storage.TransactionOptions) *storage.TransactionOptions {
	if len(opts) > 0 && opts[0] != nil {
		return opts[0]
	}
	defaultOpts := storage.DefaultTransactionOptions()
	return &defaultOpts
}