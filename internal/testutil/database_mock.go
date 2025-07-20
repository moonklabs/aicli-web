package testutil

import (
	"database/sql"
	"sync"
)

// MockDB 테스트용 데이터베이스 모킹
type MockDB struct {
	mu      sync.RWMutex
	data    map[string]interface{}
	queries []string
}

// NewMockDB 새로운 목 데이터베이스 생성
func NewMockDB() *MockDB {
	return &MockDB{
		data:    make(map[string]interface{}),
		queries: make([]string, 0),
	}
}

// Get 데이터 조회
func (m *MockDB) Get(key string) (interface{}, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	val, ok := m.data[key]
	return val, ok
}

// Set 데이터 저장
func (m *MockDB) Set(key string, value interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = value
}

// Delete 데이터 삭제
func (m *MockDB) Delete(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
}

// Clear 모든 데이터 삭제
func (m *MockDB) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data = make(map[string]interface{})
	m.queries = make([]string, 0)
}

// RecordQuery 쿼리 기록
func (m *MockDB) RecordQuery(query string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.queries = append(m.queries, query)
}

// GetQueries 기록된 쿼리 반환
func (m *MockDB) GetQueries() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]string, len(m.queries))
	copy(result, m.queries)
	return result
}

// MockSQLDB SQL 데이터베이스 모킹
type MockSQLDB struct {
	*sql.DB
	MockPing    func() error
	MockClose   func() error
	MockPrepare func(query string) (*sql.Stmt, error)
	MockExec    func(query string, args ...interface{}) (sql.Result, error)
	MockQuery   func(query string, args ...interface{}) (*sql.Rows, error)
}

// Ping 연결 테스트
func (m *MockSQLDB) Ping() error {
	if m.MockPing != nil {
		return m.MockPing()
	}
	return nil
}

// Close 연결 종료
func (m *MockSQLDB) Close() error {
	if m.MockClose != nil {
		return m.MockClose()
	}
	return nil
}

// MockResult SQL 실행 결과 모킹
type MockResult struct {
	lastInsertID int64
	rowsAffected int64
}

func (r *MockResult) LastInsertId() (int64, error) {
	return r.lastInsertID, nil
}

func (r *MockResult) RowsAffected() (int64, error) {
	return r.rowsAffected, nil
}

// NewMockResult 새로운 목 결과 생성
func NewMockResult(lastID, affected int64) sql.Result {
	return &MockResult{
		lastInsertID: lastID,
		rowsAffected: affected,
	}
}

// MockTransaction 트랜잭션 모킹
type MockTransaction struct {
	committed bool
	rolled    bool
	execFunc  func(query string, args ...interface{}) (sql.Result, error)
	queryFunc func(query string, args ...interface{}) (*sql.Rows, error)
}

// Exec 쿼리 실행
func (tx *MockTransaction) Exec(query string, args ...interface{}) (sql.Result, error) {
	if tx.execFunc != nil {
		return tx.execFunc(query, args...)
	}
	return NewMockResult(1, 1), nil
}

// Query 쿼리 실행 및 결과 반환
func (tx *MockTransaction) Query(query string, args ...interface{}) (*sql.Rows, error) {
	if tx.queryFunc != nil {
		return tx.queryFunc(query, args...)
	}
	return nil, nil
}

// Commit 트랜잭션 커밋
func (tx *MockTransaction) Commit() error {
	tx.committed = true
	return nil
}

// Rollback 트랜잭션 롤백
func (tx *MockTransaction) Rollback() error {
	tx.rolled = true
	return nil
}

// IsCommitted 커밋 여부 확인
func (tx *MockTransaction) IsCommitted() bool {
	return tx.committed
}

// IsRolledBack 롤백 여부 확인
func (tx *MockTransaction) IsRolledBack() bool {
	return tx.rolled
}