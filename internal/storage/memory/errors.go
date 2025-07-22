package memory

import "errors"

// 메모리 스토리지 공통 에러 정의
var (
	// ErrNotFound 항목을 찾을 수 없음
	ErrNotFound = errors.New("not found")
	
	// ErrAlreadyExists 이미 존재함
	ErrAlreadyExists = errors.New("already exists")
)