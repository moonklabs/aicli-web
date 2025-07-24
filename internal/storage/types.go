package storage

import (
	"encoding/json"
	"fmt"
)

// Serializer 데이터 직렬화 인터페이스
type Serializer interface {
	// Marshal 객체를 바이트 슬라이스로 직렬화
	Marshal(v interface{}) ([]byte, error)
	
	// Unmarshal 바이트 슬라이스를 객체로 역직렬화
	Unmarshal(data []byte, v interface{}) error
}

// JSONSerializer JSON 형식 직렬화 구현
type JSONSerializer struct{}

// Marshal JSON으로 직렬화
func (s JSONSerializer) Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// Unmarshal JSON에서 역직렬화
func (s JSONSerializer) Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// PagingRequest 페이징 요청 (PaginationRequest의 별칭)
type PagingRequest struct {
	Page     int    `json:"page" form:"page" validate:"min=1"`
	PageSize int    `json:"page_size" form:"page_size" validate:"min=1,max=100"`
	SortBy   string `json:"sort_by" form:"sort_by"`
	SortDir  string `json:"sort_dir" form:"sort_dir" validate:"omitempty,oneof=asc desc"`
}

// PagingResponse 페이징 응답
type PagingResponse struct {
	Data       interface{} `json:"data"`
	Pagination *PaginationMeta `json:"pagination"`
}

// PaginationMeta 페이지네이션 메타데이터
type PaginationMeta struct {
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	TotalItems int `json:"total_items"`
	TotalPages int `json:"total_pages"`
}

// QueryMonitor 쿼리 모니터링 인터페이스
type QueryMonitor interface {
	// RecordQuery 쿼리 실행 기록
	RecordQuery(query string, duration int64, error error)
	
	// GetStats 쿼리 통계 반환
	GetStats() map[string]interface{}
}

// DefaultQueryMonitor 기본 쿼리 모니터 구현
type DefaultQueryMonitor struct {
	queries []QueryRecord
}

// QueryRecord 쿼리 실행 기록
type QueryRecord struct {
	Query    string
	Duration int64
	Error    error
	Time     int64
}

// RecordQuery 쿼리 기록
func (m *DefaultQueryMonitor) RecordQuery(query string, duration int64, err error) {
	// 구현 필요
}

// GetStats 통계 반환
func (m *DefaultQueryMonitor) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"total_queries": len(m.queries),
	}
}

// TransactionFunc 트랜잭션 함수 타입
type TransactionFunc func() error

// Transactioner 트랜잭션 인터페이스
type Transactioner interface {
	// Begin 트랜잭션 시작
	Begin() error
	
	// Commit 트랜잭션 커밋
	Commit() error
	
	// Rollback 트랜잭션 롤백
	Rollback() error
	
	// RunInTransaction 트랜잭션 내에서 함수 실행
	RunInTransaction(fn TransactionFunc) error
}

// NotFoundError 리소스를 찾을 수 없을 때 발생하는 오류
type NotFoundError struct {
	Resource string
	ID       string
}

// Error 오류 메시지 반환
func (e NotFoundError) Error() string {
	return fmt.Sprintf("%s not found: %s", e.Resource, e.ID)
}

