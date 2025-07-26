package transaction

import (
	"context"
	"errors"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/aicli/aicli-web/internal/storage"
	"github.com/aicli/aicli-web/internal/storage/memory"
)

// MockTransactionalStorage 테스트용 트랜잭션 스토리지
type MockTransactionalStorage struct {
	storage.Storage
	beginTxCalled    int
	withTxCalled     int
	beginTxError     error
	withTxError      error
	transactions     []*storage.BaseTx
	mutex            sync.Mutex
	simulateDeadlock bool
	simulateTimeout  bool
}

// BeginTx 새 트랜잭션 시작
func (m *MockTransactionalStorage) BeginTx(ctx context.Context) (storage.Transaction, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	m.beginTxCalled++
	
	if m.beginTxError != nil {
		return nil, m.beginTxError
	}
	
	if m.simulateDeadlock && m.beginTxCalled > 1 {
		return nil, errors.New("database is locked")
	}
	
	if m.simulateTimeout {
		select {
		case <-time.After(100 * time.Millisecond):
			return nil, errors.New("timeout")
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	
	tx := storage.NewBaseTx(ctx, m.Storage)
	m.transactions = append(m.transactions, tx)
	return tx, nil
}

// WithTx 트랜잭션 내에서 작업 실행
func (m *MockTransactionalStorage) WithTx(ctx context.Context, fn func(tx storage.Transaction) error) error {
	m.mutex.Lock()
	m.withTxCalled++
	m.mutex.Unlock()
	
	if m.withTxError != nil {
		return m.withTxError
	}
	
	tx, err := m.BeginTx(ctx)
	if err != nil {
		return err
	}
	
	defer func() {
		if !tx.IsClosed() {
			if err != nil {
				tx.Rollback()
			} else {
				tx.Commit()
			}
		}
	}()
	
	err = fn(tx)
	return err
}

// TestTransactionManager 트랜잭션 매니저 테스트
func TestTransactionManager(t *testing.T) {
	memStore := memory.New()
	mockStorage := &MockTransactionalStorage{Storage: memStore}
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	
	t.Run("매니저 생성", func(t *testing.T) {
		manager := NewManager(mockStorage, logger)
		assert.NotNil(t, manager)
	})
	
	t.Run("기본 트랜잭션 시작", func(t *testing.T) {
		manager := NewManager(mockStorage, logger)
		
		tx, err := manager.Begin(context.Background(), nil)
		assert.NoError(t, err)
		assert.NotNil(t, tx)
		assert.False(t, tx.IsClosed())
		
		err = tx.Commit()
		assert.NoError(t, err)
		assert.True(t, tx.IsClosed())
	})
	
	t.Run("옵션을 사용한 트랜잭션 시작", func(t *testing.T) {
		manager := NewManager(mockStorage, logger)
		
		opts := storage.DefaultTransactionOptions().
			WithTimeout(5 * time.Second).
			WithReadOnly(true).
			WithIsolationLevel(storage.IsolationLevelSerializable)
		
		tx, err := manager.Begin(context.Background(), &opts)
		assert.NoError(t, err)
		assert.NotNil(t, tx)
		
		tx.Rollback()
	})
	
	t.Run("RunInTx 성공", func(t *testing.T) {
		manager := NewManager(mockStorage, logger)
		
		executed := false
		err := manager.RunInTx(context.Background(), func(ctx context.Context) error {
			executed = true
			
			// 컨텍스트에서 트랜잭션 확인
			tx, exists := storage.GetTxFromContext(ctx)
			assert.True(t, exists)
			assert.NotNil(t, tx)
			assert.False(t, tx.IsClosed())
			
			return nil
		})
		
		assert.NoError(t, err)
		assert.True(t, executed)
	})
	
	t.Run("RunInTx 실패 시 롤백", func(t *testing.T) {
		manager := NewManager(mockStorage, logger)
		
		testErr := errors.New("test error")
		err := manager.RunInTx(context.Background(), func(ctx context.Context) error {
			return testErr
		})
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "test error")
	})
	
	t.Run("RunInTxWithResult", func(t *testing.T) {
		manager := NewManager(mockStorage, logger)
		
		result, err := manager.RunInTxWithResult(context.Background(), func(ctx context.Context) (interface{}, error) {
			return 42, nil
		})
		
		assert.NoError(t, err)
		assert.Equal(t, 42, result)
	})
	
	t.Run("Current 트랜잭션 조회", func(t *testing.T) {
		manager := NewManager(mockStorage, logger)
		
		// 트랜잭션 밖에서는 없어야 함
		tx, exists := manager.Current(context.Background())
		assert.False(t, exists)
		assert.Nil(t, tx)
		
		// 트랜잭션 내에서는 있어야 함
		manager.RunInTx(context.Background(), func(ctx context.Context) error {
			tx, exists := manager.Current(ctx)
			assert.True(t, exists)
			assert.NotNil(t, tx)
			return nil
		})
	})
	
	t.Run("IsInTransaction", func(t *testing.T) {
		manager := NewManager(mockStorage, logger)
		
		// 트랜잭션 밖에서는 false
		assert.False(t, manager.IsInTransaction(context.Background()))
		
		// 트랜잭션 내에서는 true
		manager.RunInTx(context.Background(), func(ctx context.Context) error {
			assert.True(t, manager.IsInTransaction(ctx))
			return nil
		})
	})
	
	t.Run("중첩 트랜잭션 (기존 트랜잭션 재사용)", func(t *testing.T) {
		manager := NewManager(mockStorage, logger)
		
		outerExecuted := false
		innerExecuted := false
		
		err := manager.RunInTx(context.Background(), func(outerCtx context.Context) error {
			outerExecuted = true
			outerTx, exists := storage.GetTxFromContext(outerCtx)
			assert.True(t, exists)
			
			// 중첩 트랜잭션 실행
			return manager.RunInTx(outerCtx, func(innerCtx context.Context) error {
				innerExecuted = true
				innerTx, exists := storage.GetTxFromContext(innerCtx)
				assert.True(t, exists)
				
				// 같은 트랜잭션을 재사용해야 함
				assert.Equal(t, outerTx, innerTx)
				
				return nil
			})
		})
		
		assert.NoError(t, err)
		assert.True(t, outerExecuted)
		assert.True(t, innerExecuted)
	})
}

// TestTransactionManagerRetry 재시도 로직 테스트
func TestTransactionManagerRetry(t *testing.T) {
	storage := memory.New()
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	
	t.Run("데드락 시 재시도", func(t *testing.T) {
		mockStorage := &MockTransactionalStorage{
			Storage:          storage,
			simulateDeadlock: true,
		}
		manager := NewManager(mockStorage, logger)
		
		opts := storage.DefaultTransactionOptions().
			WithRetry(2, 10*time.Millisecond)
		
		_, err := manager.Begin(context.Background(), &opts)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "최대 재시도 횟수 초과")
		
		// 재시도 확인
		mockStorage.mutex.Lock()
		assert.True(t, mockStorage.beginTxCalled > 1)
		mockStorage.mutex.Unlock()
	})
	
	t.Run("타임아웃", func(t *testing.T) {
		mockStorage := &MockTransactionalStorage{
			Storage:         storage,
			simulateTimeout: true,
		}
		manager := NewManager(mockStorage, logger)
		
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		
		_, err := manager.Begin(ctx, nil)
		assert.Error(t, err)
	})
}

// TestTransactionManagerConcurrency 동시성 테스트
func TestTransactionManagerConcurrency(t *testing.T) {
	memStore := memory.New()
	mockStorage := &MockTransactionalStorage{Storage: memStore}
	manager := NewManager(mockStorage, log.New(os.Stdout, "[TEST] ", log.LstdFlags))
	
	t.Run("동시 트랜잭션 실행", func(t *testing.T) {
		const numGoroutines = 10
		var wg sync.WaitGroup
		errors := make(chan error, numGoroutines)
		
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				
				err := manager.RunInTx(context.Background(), func(ctx context.Context) error {
					// 약간의 작업 시뮬레이션
					time.Sleep(time.Millisecond * 10)
					return nil
				})
				
				if err != nil {
					errors <- err
				}
			}(i)
		}
		
		wg.Wait()
		close(errors)
		
		// 에러가 없어야 함
		for err := range errors {
			t.Errorf("예상치 못한 에러: %v", err)
		}
	})
}

// TestTransactionManagerStats 통계 테스트
func TestTransactionManagerStats(t *testing.T) {
	memStore := memory.New()
	mockStorage := &MockTransactionalStorage{Storage: memStore}
	manager := NewManager(mockStorage, log.New(os.Stdout, "[TEST] ", log.LstdFlags))
	
	t.Run("통계 수집", func(t *testing.T) {
		// 성공적인 트랜잭션
		err := manager.RunInTx(context.Background(), func(ctx context.Context) error {
			return nil
		})
		assert.NoError(t, err)
		
		// 실패한 트랜잭션
		err = manager.RunInTx(context.Background(), func(ctx context.Context) error {
			return errors.New("test error")
		})
		assert.Error(t, err)
		
		// 통계 확인
		stats := manager.GetStats()
		assert.True(t, stats.TotalCount > 0)
		assert.True(t, stats.CommittedCount > 0)
		assert.True(t, stats.RolledBackCount > 0)
		assert.NotNil(t, stats.LastError)
	})
}

// TestTransactionOptions 트랜잭션 옵션 테스트
func TestTransactionOptions(t *testing.T) {
	t.Run("기본 옵션", func(t *testing.T) {
		opts := storage.DefaultTransactionOptions()
		
		assert.False(t, opts.ReadOnly)
		assert.Equal(t, 30*time.Second, opts.Timeout)
		assert.Equal(t, storage.IsolationLevelReadCommitted, opts.IsolationLevel)
		assert.Equal(t, 3, opts.RetryCount)
		assert.Equal(t, 100*time.Millisecond, opts.RetryDelay)
		assert.False(t, opts.EnableSavepoint)
	})
	
	t.Run("옵션 빌더 패턴", func(t *testing.T) {
		opts := storage.DefaultTransactionOptions().
			WithReadOnly(true).
			WithTimeout(10 * time.Second).
			WithIsolationLevel(storage.IsolationLevelSerializable).
			WithRetry(5, 200*time.Millisecond).
			WithSavepoint(true)
		
		assert.True(t, opts.ReadOnly)
		assert.Equal(t, 10*time.Second, opts.Timeout)
		assert.Equal(t, storage.IsolationLevelSerializable, opts.IsolationLevel)
		assert.Equal(t, 5, opts.RetryCount)
		assert.Equal(t, 200*time.Millisecond, opts.RetryDelay)
		assert.True(t, opts.EnableSavepoint)
	})
	
	t.Run("격리 수준 문자열 변환", func(t *testing.T) {
		assert.Equal(t, "DEFAULT", storage.IsolationLevelDefault.String())
		assert.Equal(t, "READ_COMMITTED", storage.IsolationLevelReadCommitted.String())
		assert.Equal(t, "SERIALIZABLE", storage.IsolationLevelSerializable.String())
		
		assert.Equal(t, "DEFERRED", storage.IsolationLevelDefault.SQLiteLevel())
		assert.Equal(t, "EXCLUSIVE", storage.IsolationLevelSerializable.SQLiteLevel())
	})
}

// BenchmarkTransactionManager 벤치마크 테스트
func BenchmarkTransactionManager(b *testing.B) {
	memStore := memory.New()
	mockStorage := &MockTransactionalStorage{Storage: memStore}
	manager := NewManager(mockStorage, nil)
	
	b.Run("트랜잭션 시작/커밋", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			tx, err := manager.Begin(context.Background(), nil)
			if err != nil {
				b.Fatal(err)
			}
			tx.Commit()
		}
	})
	
	b.Run("RunInTx", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			err := manager.RunInTx(context.Background(), func(ctx context.Context) error {
				return nil
			})
			if err != nil {
				b.Fatal(err)
			}
		}
	})
	
	b.Run("동시 트랜잭션", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				err := manager.RunInTx(context.Background(), func(ctx context.Context) error {
					return nil
				})
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	})
}