package storage

import (
	"fmt"
	"strings"
)

// 추가 에러 정의
var (
	// ErrDuplicateKey 중복 키 에러
	ErrDuplicateKey = fmt.Errorf("중복된 키")
	
	// ErrConnectionFailed 연결 실패
	ErrConnectionFailed = fmt.Errorf("데이터베이스 연결 실패")
	
	// ErrTransactionFailed 트랜잭션 실패
	ErrTransactionFailed = fmt.Errorf("트랜잭션 실패")
	
	// ErrStorageTimeout 스토리지 타임아웃
	ErrStorageTimeout = fmt.Errorf("스토리지 작업 타임아웃")
	
	// ErrInvalidStorageType 지원하지 않는 스토리지 타입
	ErrInvalidStorageType = fmt.Errorf("지원하지 않는 스토리지 타입")
	
	// ErrStorageClosed 스토리지가 닫혔음
	ErrStorageClosed = fmt.Errorf("스토리지가 닫혔습니다")
	
	// ErrCorruptedData 손상된 데이터
	ErrCorruptedData = fmt.Errorf("데이터가 손상되었습니다")
)

// StorageError 스토리지 에러 래퍼
type StorageError struct {
	Operation string
	Type      string
	Cause     error
	Details   map[string]interface{}
}

// Error 에러 메시지 반환
func (e *StorageError) Error() string {
	var builder strings.Builder
	
	builder.WriteString(fmt.Sprintf("스토리지 에러 [%s:%s]", e.Type, e.Operation))
	
	if e.Cause != nil {
		builder.WriteString(fmt.Sprintf(": %s", e.Cause.Error()))
	}
	
	if len(e.Details) > 0 {
		builder.WriteString(" (")
		first := true
		for k, v := range e.Details {
			if !first {
				builder.WriteString(", ")
			}
			builder.WriteString(fmt.Sprintf("%s=%v", k, v))
			first = false
		}
		builder.WriteString(")")
	}
	
	return builder.String()
}

// Unwrap 원본 에러 반환
func (e *StorageError) Unwrap() error {
	return e.Cause
}

// Is 에러 타입 비교
func (e *StorageError) Is(target error) bool {
	if e.Cause == nil {
		return false
	}
	return e.Cause == target || fmt.Errorf(e.Error()) == target
}

// NewStorageError 새 스토리지 에러 생성
func NewStorageError(operation, storageType string, cause error) *StorageError {
	return &StorageError{
		Operation: operation,
		Type:      storageType,
		Cause:     cause,
		Details:   make(map[string]interface{}),
	}
}

// WithDetail 세부 정보 추가
func (e *StorageError) WithDetail(key string, value interface{}) *StorageError {
	e.Details[key] = value
	return e
}

// ConvertError 데이터베이스별 에러를 공통 에러로 변환
func ConvertError(err error, operation, storageType string) error {
	if err == nil {
		return nil
	}
	
	// 이미 StorageError인 경우
	if storageErr, ok := err.(*StorageError); ok {
		return storageErr
	}
	
	errMsg := strings.ToLower(err.Error())
	
	// SQLite 에러 변환
	if storageType == "sqlite" {
		return convertSQLiteError(err, operation, errMsg)
	}
	
	// BoltDB 에러 변환
	if storageType == "boltdb" {
		return convertBoltDBError(err, operation, errMsg)
	}
	
	// 기본 에러 처리
	return NewStorageError(operation, storageType, err)
}

// convertSQLiteError SQLite 에러 변환
func convertSQLiteError(err error, operation, errMsg string) error {
	storageErr := NewStorageError(operation, "sqlite", err)
	
	switch {
	case strings.Contains(errMsg, "unique constraint"):
		return storageErr.WithDetail("type", "unique_constraint")
	case strings.Contains(errMsg, "foreign key constraint"):
		return storageErr.WithDetail("type", "foreign_key_constraint")
	case strings.Contains(errMsg, "no such table"):
		return storageErr.WithDetail("type", "table_not_found")
	case strings.Contains(errMsg, "database is locked"):
		return storageErr.WithDetail("type", "database_locked")
	case strings.Contains(errMsg, "no such column"):
		return storageErr.WithDetail("type", "column_not_found")
	case strings.Contains(errMsg, "syntax error"):
		return storageErr.WithDetail("type", "syntax_error")
	default:
		return storageErr
	}
}

// convertBoltDBError BoltDB 에러 변환
func convertBoltDBError(err error, operation, errMsg string) error {
	storageErr := NewStorageError(operation, "boltdb", err)
	
	switch {
	case strings.Contains(errMsg, "bucket not found"):
		return storageErr.WithDetail("type", "bucket_not_found")
	case strings.Contains(errMsg, "key not found"):
		return storageErr.WithDetail("type", "key_not_found")
	case strings.Contains(errMsg, "database not open"):
		return storageErr.WithDetail("type", "database_not_open")
	case strings.Contains(errMsg, "read-only"):
		return storageErr.WithDetail("type", "read_only")
	case strings.Contains(errMsg, "timeout"):
		return storageErr.WithDetail("type", "timeout")
	default:
		return storageErr
	}
}

// IsNotFoundError 데이터를 찾을 수 없는 에러인지 확인
func IsNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	
	if err == ErrNotFound {
		return true
	}
	
	if storageErr, ok := err.(*StorageError); ok {
		if storageErr.Cause == ErrNotFound {
			return true
		}
		
		if detail, ok := storageErr.Details["type"].(string); ok {
			return detail == "key_not_found" || detail == "table_not_found"
		}
	}
	
	errMsg := strings.ToLower(err.Error())
	return strings.Contains(errMsg, "not found") || 
		   strings.Contains(errMsg, "no rows") ||
		   strings.Contains(errMsg, "key not found")
}

// IsAlreadyExistsError 이미 존재하는 에러인지 확인
func IsAlreadyExistsError(err error) bool {
	if err == nil {
		return false
	}
	
	if err == ErrAlreadyExists || err == ErrDuplicateKey {
		return true
	}
	
	if storageErr, ok := err.(*StorageError); ok {
		if storageErr.Cause == ErrAlreadyExists || storageErr.Cause == ErrDuplicateKey {
			return true
		}
		
		if detail, ok := storageErr.Details["type"].(string); ok {
			return detail == "unique_constraint" || detail == "duplicate_key"
		}
	}
	
	errMsg := strings.ToLower(err.Error())
	return strings.Contains(errMsg, "already exists") || 
		   strings.Contains(errMsg, "duplicate") ||
		   strings.Contains(errMsg, "unique constraint")
}

// IsConnectionError 연결 에러인지 확인
func IsConnectionError(err error) bool {
	if err == nil {
		return false
	}
	
	if err == ErrConnectionFailed {
		return true
	}
	
	if storageErr, ok := err.(*StorageError); ok {
		if storageErr.Cause == ErrConnectionFailed {
			return true
		}
		
		if detail, ok := storageErr.Details["type"].(string); ok {
			return detail == "database_locked" || detail == "database_not_open"
		}
	}
	
	errMsg := strings.ToLower(err.Error())
	return strings.Contains(errMsg, "connection") || 
		   strings.Contains(errMsg, "network") ||
		   strings.Contains(errMsg, "timeout") ||
		   strings.Contains(errMsg, "database is locked")
}

// IsTransactionError 트랜잭션 에러인지 확인
func IsTransactionError(err error) bool {
	if err == nil {
		return false
	}
	
	if err == ErrTransactionFailed {
		return true
	}
	
	if storageErr, ok := err.(*StorageError); ok {
		if storageErr.Cause == ErrTransactionFailed {
			return true
		}
		
		if detail, ok := storageErr.Details["type"].(string); ok {
			return detail == "read_only" || detail == "rollback"
		}
	}
	
	errMsg := strings.ToLower(err.Error())
	return strings.Contains(errMsg, "transaction") || 
		   strings.Contains(errMsg, "rollback") ||
		   strings.Contains(errMsg, "commit")
}

// WrapError 에러 래핑 유틸리티
func WrapError(err error, operation, storageType string, details ...map[string]interface{}) error {
	if err == nil {
		return nil
	}
	
	storageErr := ConvertError(err, operation, storageType)
	
	// 세부 정보 추가
	if len(details) > 0 && len(details[0]) > 0 {
		if se, ok := storageErr.(*StorageError); ok {
			for k, v := range details[0] {
				se.WithDetail(k, v)
			}
		}
	}
	
	return storageErr
}