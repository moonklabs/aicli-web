package storage_test

import (
	"context"
	"testing"
	"time"
	
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	
	"github.com/aicli/aicli-web/internal/storage"
	"github.com/aicli/aicli-web/internal/storage/memory"
)

// TestDefaultTransactionOptions 기본 트랜잭션 옵션 테스트
func TestDefaultTransactionOptions(t *testing.T) {
	opts := storage.DefaultTransactionOptions()
	
	assert.False(t, opts.ReadOnly)
	assert.Empty(t, opts.IsolationLevel)
	assert.Nil(t, opts.Timeout)
}

// TestTxContextKey 트랜잭션 컨텍스트 키 테스트
func TestTxContextKey(t *testing.T) {
	ctx := context.Background()
	storage := memory.New()
	baseTx := NewBaseTx(ctx, storage)
	
	// 컨텍스트에 트랜잭션 추가
	ctxWithTx := WithTxContext(ctx, baseTx)
	
	// 컨텍스트에서 트랜잭션 추출
	extractedTx, ok := GetTxFromContext(ctxWithTx)
	assert.True(t, ok)
	assert.Equal(t, baseTx, extractedTx)
	
	// 트랜잭션이 없는 컨텍스트
	_, ok = GetTxFromContext(ctx)
	assert.False(t, ok)
}

// TestBaseTx 기본 트랜잭션 테스트
func TestBaseTx(t *testing.T) {
	ctx := context.Background()
	storage := memory.New()
	
	t.Run("기본 트랜잭션 생성", func(t *testing.T) {
		tx := NewBaseTx(ctx, storage)
		
		assert.NotNil(t, tx)
		assert.Equal(t, ctx, tx.Context())
		assert.False(t, tx.IsClosed())
		assert.NotNil(t, tx.Workspace())
		assert.NotNil(t, tx.Project())
		assert.NotNil(t, tx.Session())
		assert.NotNil(t, tx.Task())
	})
	
	t.Run("트랜잭션 커밋", func(t *testing.T) {
		tx := NewBaseTx(ctx, storage)
		
		err := tx.Commit()
		assert.NoError(t, err)
		assert.True(t, tx.IsClosed())
		assert.True(t, tx.committed)
		assert.False(t, tx.rolledBack)
		
		// 이미 커밋된 트랜잭션을 다시 커밋하면 에러
		err = tx.Commit()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "이미 완료되었습니다")
	})
	
	t.Run("트랜잭션 롤백", func(t *testing.T) {
		tx := NewBaseTx(ctx, storage)
		
		err := tx.Rollback()
		assert.NoError(t, err)
		assert.True(t, tx.IsClosed())
		assert.False(t, tx.committed)
		assert.True(t, tx.rolledBack)
		
		// 이미 롤백된 트랜잭션을 다시 롤백하면 에러
		err = tx.Rollback()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "이미 완료되었습니다")
	})
	
	t.Run("이미 커밋된 트랜잭션 롤백", func(t *testing.T) {
		tx := NewBaseTx(ctx, storage)
		
		err := tx.Commit()
		assert.NoError(t, err)
		
		// 커밋 후 롤백 시도
		err = tx.Rollback()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "이미 완료되었습니다")
	})
}

// MockTransactionalStorage 테스트용 트랜잭션 스토리지
type MockTransactionalStorage struct {
	storage.Storage
	beginTxCalled    int
	withTxCalled     int
	beginTxError     error
	withTxError      error
	transactions     []*storage.BaseTx
}

// BeginTx 새 트랜잭션 시작
func (m *MockTransactionalStorage) BeginTx(ctx context.Context) (storage.Transaction, error) {
	m.beginTxCalled++
	if m.beginTxError != nil {
		return nil, m.beginTxError
	}
	
	tx := storage.NewBaseTx(ctx, m.Storage)
	m.transactions = append(m.transactions, tx)
	return tx, nil
}

// WithTx 트랜잭션 내에서 작업 실행
func (m *MockTransactionalStorage) WithTx(ctx context.Context, fn func(tx storage.Transaction) error) error {
	m.withTxCalled++
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

// TestTxManager 트랜잭션 매니저 테스트
func TestTxManager(t *testing.T) {
	storage := memory.New()
	mockStorage := &MockTransactionalStorage{Storage: storage}
	
	t.Run("트랜잭션 매니저 생성", func(t *testing.T) {
		tm := NewTxManager(mockStorage)
		assert.NotNil(t, tm)
	})
	
	t.Run("성공적인 트랜잭션", func(t *testing.T) {
		tm := NewTxManager(mockStorage)
		
		executed := false
		err := tm.WithTransaction(context.Background(), func(tx Transaction) error {
			executed = true
			assert.NotNil(t, tx)
			assert.False(t, tx.IsClosed())
			return nil
		})
		
		assert.NoError(t, err)
		assert.True(t, executed)
		assert.Equal(t, 1, mockStorage.withTxCalled)
	})
	
	t.Run("실패한 트랜잭션", func(t *testing.T) {
		mockStorage := &MockTransactionalStorage{Storage: storage}
		tm := NewTxManager(mockStorage)
		
		testErr := assert.AnError
		err := tm.WithTransaction(context.Background(), func(tx Transaction) error {
			return testErr
		})
		
		assert.Error(t, err)
		assert.Equal(t, testErr, err)
		assert.Equal(t, 1, mockStorage.withTxCalled)
	})
	
	t.Run("여러 작업 실행", func(t *testing.T) {
		mockStorage := &MockTransactionalStorage{Storage: storage}
		tm := NewTxManager(mockStorage)
		
		executed1, executed2, executed3 := false, false, false
		
		err := tm.ExecuteInTx(context.Background(),
			func(tx Transaction) error {
				executed1 = true
				return nil
			},
			func(tx Transaction) error {
				executed2 = true
				return nil
			},
			func(tx Transaction) error {
				executed3 = true
				return nil
			},
		)
		
		assert.NoError(t, err)
		assert.True(t, executed1)
		assert.True(t, executed2)
		assert.True(t, executed3)
		assert.Equal(t, 1, mockStorage.withTxCalled)
	})
	
	t.Run("여러 작업 중 실패", func(t *testing.T) {
		mockStorage := &MockTransactionalStorage{Storage: storage}
		tm := NewTxManager(mockStorage)
		
		executed1, executed2, executed3 := false, false, false
		testErr := assert.AnError
		
		err := tm.ExecuteInTx(context.Background(),
			func(tx Transaction) error {
				executed1 = true
				return nil
			},
			func(tx Transaction) error {
				executed2 = true
				return testErr
			},
			func(tx Transaction) error {
				executed3 = true
				return nil
			},
		)
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "트랜잭션 작업 실행 실패")
		assert.True(t, executed1)
		assert.True(t, executed2)
		assert.False(t, executed3) // 실패 후 실행되지 않음
		assert.Equal(t, 1, mockStorage.withTxCalled)
	})
}

// TestRunInTx 트랜잭션 유틸리티 함수 테스트
func TestRunInTx(t *testing.T) {
	storage := memory.New()
	
	t.Run("성공적인 실행", func(t *testing.T) {
		mockStorage := &MockTransactionalStorage{Storage: storage}
		
		executed := false
		err := RunInTx(context.Background(), mockStorage, func(tx Transaction) error {
			executed = true
			return nil
		})
		
		assert.NoError(t, err)
		assert.True(t, executed)
		assert.Equal(t, 1, mockStorage.beginTxCalled)
	})
	
	t.Run("실패한 실행", func(t *testing.T) {
		mockStorage := &MockTransactionalStorage{Storage: storage}
		testErr := assert.AnError
		
		err := RunInTx(context.Background(), mockStorage, func(tx Transaction) error {
			return testErr
		})
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "트랜잭션 함수 실행 실패")
		assert.Equal(t, 1, mockStorage.beginTxCalled)
	})
	
	t.Run("트랜잭션 시작 실패", func(t *testing.T) {
		mockStorage := &MockTransactionalStorage{
			Storage:      storage,
			beginTxError: assert.AnError,
		}
		
		executed := false
		err := RunInTx(context.Background(), mockStorage, func(tx Transaction) error {
			executed = true
			return nil
		})
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "트랜잭션 시작 실패")
		assert.False(t, executed)
		assert.Equal(t, 1, mockStorage.beginTxCalled)
	})
}

// TestTransactionLifecycle 트랜잭션 생명주기 테스트
func TestTransactionLifecycle(t *testing.T) {
	ctx := context.Background()
	storage := memory.New()
	
	t.Run("정상적인 커밋 플로우", func(t *testing.T) {
		tx := NewBaseTx(ctx, storage)
		
		// 초기 상태 확인
		assert.False(t, tx.IsClosed())
		assert.Equal(t, ctx, tx.Context())
		
		// 스토리지 인터페이스 접근
		ws := tx.Workspace()
		assert.NotNil(t, ws)
		
		// 커밋
		err := tx.Commit()
		assert.NoError(t, err)
		assert.True(t, tx.IsClosed())
	})
	
	t.Run("정상적인 롤백 플로우", func(t *testing.T) {
		tx := NewBaseTx(ctx, storage)
		
		// 초기 상태 확인
		assert.False(t, tx.IsClosed())
		
		// 롤백
		err := tx.Rollback()
		assert.NoError(t, err)
		assert.True(t, tx.IsClosed())
	})
	
	t.Run("컨텍스트 취소", func(t *testing.T) {
		cancelCtx, cancel := context.WithCancel(ctx)
		tx := NewBaseTx(cancelCtx, storage)
		
		cancel() // 컨텍스트 취소
		
		// 트랜잭션 자체는 여전히 작동해야 함
		assert.False(t, tx.IsClosed())
		assert.NotNil(t, tx.Context())
		
		// 수동으로 커밋 가능
		err := tx.Commit()
		assert.NoError(t, err)
		assert.True(t, tx.IsClosed())
	})
	
	t.Run("타임아웃 컨텍스트", func(t *testing.T) {
		timeoutCtx, cancel := context.WithTimeout(ctx, time.Millisecond)
		defer cancel()
		
		tx := NewBaseTx(timeoutCtx, storage)
		
		time.Sleep(time.Millisecond * 2) // 타임아웃 발생
		
		// 트랜잭션은 여전히 작동
		assert.False(t, tx.IsClosed())
		
		err := tx.Commit()
		assert.NoError(t, err)
		assert.True(t, tx.IsClosed())
	})
}

// BenchmarkBaseTxOperations 기본 트랜잭션 작업 벤치마크
func BenchmarkBaseTxOperations(b *testing.B) {
	ctx := context.Background()
	storage := memory.New()
	
	b.Run("트랜잭션 생성", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			tx := NewBaseTx(ctx, storage)
			tx.Commit()
		}
	})
	
	b.Run("트랜잭션 커밋", func(b *testing.B) {
		txs := make([]*BaseTx, b.N)
		for i := 0; i < b.N; i++ {
			txs[i] = NewBaseTx(ctx, storage)
		}
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			txs[i].Commit()
		}
	})
	
	b.Run("컨텍스트 조작", func(b *testing.B) {
		tx := NewBaseTx(ctx, storage)
		defer tx.Commit()
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ctxWithTx := WithTxContext(ctx, tx)
			_, ok := GetTxFromContext(ctxWithTx)
			if !ok {
				b.Fatal("트랜잭션을 찾을 수 없습니다")
			}
		}
	})
}