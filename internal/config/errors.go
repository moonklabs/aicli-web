package config

import (
	"errors"
	"fmt"
)

// 설정 관련 에러 타입
var (
	// ErrConfigNotFound 는 설정 파일을 찾을 수 없을 때 반환됩니다
	ErrConfigNotFound = errors.New("설정 파일을 찾을 수 없습니다")
	
	// ErrInvalidConfig 는 설정이 유효하지 않을 때 반환됩니다
	ErrInvalidConfig = errors.New("유효하지 않은 설정입니다")
	
	// ErrPermissionDenied 는 파일 권한 문제가 있을 때 반환됩니다
	ErrPermissionDenied = errors.New("파일 권한이 거부되었습니다")
	
	// ErrBackupFailed 는 백업 생성에 실패했을 때 반환됩니다
	ErrBackupFailed = errors.New("백업 생성에 실패했습니다")
)

// ConfigError 는 설정 관련 에러를 포장하는 구조체입니다
type ConfigError struct {
	Op   string // 작업 이름 (예: "read", "write", "validate")
	Path string // 파일 경로
	Err  error  // 실제 에러
}

// Error 는 에러 메시지를 반환합니다
func (e *ConfigError) Error() string {
	if e.Path != "" {
		return fmt.Sprintf("config %s %s: %v", e.Op, e.Path, e.Err)
	}
	return fmt.Sprintf("config %s: %v", e.Op, e.Err)
}

// Unwrap 은 내부 에러를 반환합니다
func (e *ConfigError) Unwrap() error {
	return e.Err
}

// NewConfigError 는 새로운 ConfigError를 생성합니다
func NewConfigError(op, path string, err error) error {
	return &ConfigError{
		Op:   op,
		Path: path,
		Err:  err,
	}
}