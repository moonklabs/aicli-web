// Package errors는 AICode Manager CLI의 통합된 에러 처리 시스템을 제공합니다.
// 일관된 에러 분류, 사용자 친화적 메시지, 진단 정보, 해결책 제시 기능을 포함합니다.
package errors

import (
	"fmt"
	"strings"
)

// ErrorType은 에러의 종류를 나타냅니다.
type ErrorType int

const (
	// ErrorTypeUnknown은 분류되지 않은 에러를 나타냅니다.
	ErrorTypeUnknown ErrorType = iota
	
	// ErrorTypeValidation은 입력 검증 오류를 나타냅니다.
	ErrorTypeValidation
	
	// ErrorTypeConfig은 설정 관련 오류를 나타냅니다.
	ErrorTypeConfig
	
	// ErrorTypeNetwork은 네트워크 연결 오류를 나타냅니다.
	ErrorTypeNetwork
	
	// ErrorTypeFileSystem은 파일 시스템 오류를 나타냅니다.
	ErrorTypeFileSystem
	
	// ErrorTypeProcess은 프로세스 실행 오류를 나타냅니다.
	ErrorTypeProcess
	
	// ErrorTypeAuthentication은 인증 오류를 나타냅니다.
	ErrorTypeAuthentication
	
	// ErrorTypePermission은 권한 오류를 나타냅니다.
	ErrorTypePermission
	
	// ErrorTypeNotFound은 리소스 미발견 오류를 나타냅니다.
	ErrorTypeNotFound
	
	// ErrorTypeConflict은 충돌 상황 오류를 나타냅니다.
	ErrorTypeConflict
	
	// ErrorTypeInternal은 내부 시스템 오류를 나타냅니다.
	ErrorTypeInternal
)

// String은 ErrorType의 문자열 표현을 반환합니다.
func (e ErrorType) String() string {
	switch e {
	case ErrorTypeValidation:
		return "ValidationError"
	case ErrorTypeConfig:
		return "ConfigError"
	case ErrorTypeNetwork:
		return "NetworkError"
	case ErrorTypeFileSystem:
		return "FileSystemError"
	case ErrorTypeProcess:
		return "ProcessError"
	case ErrorTypeAuthentication:
		return "AuthenticationError"
	case ErrorTypePermission:
		return "PermissionError"
	case ErrorTypeNotFound:
		return "NotFoundError"
	case ErrorTypeConflict:
		return "ConflictError"
	case ErrorTypeInternal:
		return "InternalError"
	default:
		return "UnknownError"
	}
}

// ExitCode는 ErrorType에 대응하는 종료 코드를 반환합니다.
func (e ErrorType) ExitCode() int {
	switch e {
	case ErrorTypeValidation:
		return 1 // 일반적인 에러 (검증, 사용법)
	case ErrorTypeConfig:
		return 2 // 설정 에러
	case ErrorTypeNetwork:
		return 3 // 네트워크/연결 에러
	case ErrorTypeFileSystem:
		return 4 // 파일 시스템 에러
	case ErrorTypePermission:
		return 5 // 권한 에러
	case ErrorTypeAuthentication:
		return 6 // 인증 에러
	case ErrorTypeProcess:
		return 7 // 프로세스 실행 에러
	case ErrorTypeNotFound:
		return 8 // 리소스 미발견
	case ErrorTypeConflict:
		return 9 // 충돌 상황
	case ErrorTypeInternal:
		return 127 // 내부 시스템 에러
	default:
		return 1 // 기본값
	}
}

// CLIError는 CLI 전용 에러 타입으로 풍부한 메타데이터를 포함합니다.
type CLIError struct {
	// Type은 에러의 분류를 나타냅니다.
	Type ErrorType
	
	// Message는 사용자에게 표시할 주요 메시지입니다.
	Message string
	
	// Cause는 원본 에러를 나타냅니다 (체이닝용).
	Cause error
	
	// Suggestions는 문제 해결을 위한 제안사항들입니다.
	Suggestions []string
	
	// Context는 에러 발생 맥락에 대한 추가 정보입니다.
	Context map[string]interface{}
	
	// ExitCode는 프로세스 종료 코드입니다.
	ExitCode int
	
	// Debug는 디버깅용 상세 정보입니다.
	Debug map[string]interface{}
}

// Error는 error 인터페이스를 구현합니다.
func (e *CLIError) Error() string {
	return e.Message
}

// Unwrap은 래핑된 원본 에러를 반환합니다.
func (e *CLIError) Unwrap() error {
	return e.Cause
}

// AddSuggestion은 해결책 제안을 추가합니다.
func (e *CLIError) AddSuggestion(suggestion string) *CLIError {
	if e.Suggestions == nil {
		e.Suggestions = make([]string, 0, 1)
	}
	e.Suggestions = append(e.Suggestions, suggestion)
	return e
}

// AddContext는 맥락 정보를 추가합니다.
func (e *CLIError) AddContext(key string, value interface{}) *CLIError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// AddDebug는 디버깅 정보를 추가합니다.
func (e *CLIError) AddDebug(key string, value interface{}) *CLIError {
	if e.Debug == nil {
		e.Debug = make(map[string]interface{})
	}
	e.Debug[key] = value
	return e
}

// WithCause는 원본 에러를 설정합니다.
func (e *CLIError) WithCause(cause error) *CLIError {
	e.Cause = cause
	return e
}

// NewCLIError는 새로운 CLIError를 생성합니다.
func NewCLIError(errorType ErrorType, message string) *CLIError {
	return &CLIError{
		Type:     errorType,
		Message:  message,
		ExitCode: errorType.ExitCode(),
	}
}

// Validation 에러 생성 헬퍼 함수들

// NewValidationError는 입력 검증 오류를 생성합니다.
func NewValidationError(message string, suggestions ...string) *CLIError {
	err := NewCLIError(ErrorTypeValidation, message)
	for _, suggestion := range suggestions {
		err.AddSuggestion(suggestion)
	}
	return err
}

// NewRequiredFlagError는 필수 플래그 누락 오류를 생성합니다.
func NewRequiredFlagError(flagName, description string) *CLIError {
	message := fmt.Sprintf("필수 플래그가 누락되었습니다: --%s", flagName)
	err := NewValidationError(message)
	err.AddContext("flag_name", flagName)
	err.AddContext("description", description)
	err.AddSuggestion(fmt.Sprintf("플래그를 추가하세요: --%s [value]", flagName))
	err.AddSuggestion("도움말: aicli help [command]")
	return err
}

// NewInvalidValueError는 잘못된 값 입력 오류를 생성합니다.
func NewInvalidValueError(field, value string, validValues []string) *CLIError {
	message := fmt.Sprintf("잘못된 값입니다: %s = \"%s\"", field, value)
	err := NewValidationError(message)
	err.AddContext("field", field)
	err.AddContext("invalid_value", value)
	err.AddContext("valid_values", validValues)
	
	if len(validValues) > 0 {
		err.AddSuggestion(fmt.Sprintf("유효한 값 중 하나를 선택하세요: %s", strings.Join(validValues, ", ")))
	}
	err.AddSuggestion("도움말: aicli help [command]")
	return err
}

// Config 에러 생성 헬퍼 함수들

// NewConfigError는 설정 관련 오류를 생성합니다.
func NewConfigError(cause error, configPath string) *CLIError {
	message := fmt.Sprintf("설정 파일 오류: %s", configPath)
	err := NewCLIError(ErrorTypeConfig, message).WithCause(cause)
	err.AddContext("config_path", configPath)
	err.AddSuggestion("설정 파일 문법을 확인하세요")
	err.AddSuggestion("'aicli config validate'로 설정 검증하세요")
	err.AddSuggestion("'aicli config reset'으로 기본값으로 재설정하세요")
	return err
}

// NewConfigKeyError는 설정 키 관련 오류를 생성합니다.
func NewConfigKeyError(key string, cause error) *CLIError {
	message := fmt.Sprintf("설정 키 오류: %s", key)
	err := NewCLIError(ErrorTypeConfig, message).WithCause(cause)
	err.AddContext("config_key", key)
	err.AddSuggestion(fmt.Sprintf("'aicli config get %s'로 현재 값 확인하세요", key))
	err.AddSuggestion(fmt.Sprintf("'aicli config set %s [value]'로 올바른 값을 설정하세요", key))
	return err
}

// Network 에러 생성 헬퍼 함수들

// NewNetworkError는 네트워크 연결 오류를 생성합니다.
func NewNetworkError(service string, cause error) *CLIError {
	message := fmt.Sprintf("%s에 연결할 수 없습니다", service)
	err := NewCLIError(ErrorTypeNetwork, message).WithCause(cause)
	err.AddContext("service", service)
	err.AddSuggestion(fmt.Sprintf("%s 서비스가 실행 중인지 확인하세요", service))
	err.AddSuggestion("네트워크 연결 상태를 확인하세요")
	err.AddSuggestion("방화벽 설정을 확인하세요")
	return err
}

// FileSystem 에러 생성 헬퍼 함수들

// NewFileSystemError는 파일 시스템 오류를 생성합니다.
func NewFileSystemError(operation, path string, cause error) *CLIError {
	message := fmt.Sprintf("파일 시스템 오류: %s - %s", operation, path)
	err := NewCLIError(ErrorTypeFileSystem, message).WithCause(cause)
	err.AddContext("operation", operation)
	err.AddContext("path", path)
	err.AddSuggestion("파일/디렉토리 경로가 올바른지 확인하세요")
	err.AddSuggestion("파일/디렉토리 권한을 확인하세요")
	err.AddSuggestion("디스크 공간이 충분한지 확인하세요")
	return err
}

// Permission 에러 생성 헬퍼 함수들

// NewPermissionError는 권한 오류를 생성합니다.
func NewPermissionError(action, resource string) *CLIError {
	message := fmt.Sprintf("권한이 없습니다: %s - %s", action, resource)
	err := NewCLIError(ErrorTypePermission, message)
	err.AddContext("action", action)
	err.AddContext("resource", resource)
	err.AddSuggestion("파일/디렉토리 권한을 확인하세요")
	err.AddSuggestion("현재 사용자의 권한을 확인하세요")
	err.AddSuggestion("필요한 경우 sudo를 사용하여 재시도하세요")
	return err
}

// Process 에러 생성 헬퍼 함수들

// NewProcessError는 프로세스 실행 오류를 생성합니다.
func NewProcessError(command string, exitCode int, cause error) *CLIError {
	message := fmt.Sprintf("프로세스 실행 실패: %s (종료 코드: %d)", command, exitCode)
	err := NewCLIError(ErrorTypeProcess, message).WithCause(cause)
	err.AddContext("command", command)
	err.AddContext("exit_code", exitCode)
	err.AddSuggestion("명령어가 올바른지 확인하세요")
	err.AddSuggestion("필요한 의존성이 설치되어 있는지 확인하세요")
	err.AddSuggestion("--verbose 플래그로 상세 로그를 확인하세요")
	return err
}

// Authentication 에러 생성 헬퍼 함수들

// NewAuthenticationError는 인증 오류를 생성합니다.
func NewAuthenticationError(service string, cause error) *CLIError {
	message := fmt.Sprintf("인증 실패: %s", service)
	err := NewCLIError(ErrorTypeAuthentication, message).WithCause(cause)
	err.AddContext("service", service)
	err.AddSuggestion("API 키 또는 토큰이 올바른지 확인하세요")
	err.AddSuggestion("'aicli auth login'으로 다시 로그인하세요")
	err.AddSuggestion("인증 정보가 만료되지 않았는지 확인하세요")
	return err
}

// NotFound 에러 생성 헬퍼 함수들

// NewNotFoundError는 리소스 미발견 오류를 생성합니다.
func NewNotFoundError(resourceType, name string) *CLIError {
	message := fmt.Sprintf("%s를 찾을 수 없습니다: %s", resourceType, name)
	err := NewCLIError(ErrorTypeNotFound, message)
	err.AddContext("resource_type", resourceType)
	err.AddContext("resource_name", name)
	err.AddSuggestion("리소스 이름이 정확한지 확인하세요")
	err.AddSuggestion(fmt.Sprintf("'aicli %s list'로 존재하는 목록을 확인하세요", resourceType))
	err.AddSuggestion("리소스가 생성되었는지 확인하세요")
	return err
}

// Conflict 에러 생성 헬퍼 함수들

// NewConflictError는 충돌 상황 오류를 생성합니다.
func NewConflictError(resource, reason string) *CLIError {
	message := fmt.Sprintf("충돌이 발생했습니다: %s", resource)
	err := NewCLIError(ErrorTypeConflict, message)
	err.AddContext("resource", resource)
	err.AddContext("reason", reason)
	err.AddSuggestion("기존 리소스의 상태를 확인하세요")
	err.AddSuggestion("--force 플래그로 강제 실행을 고려하세요")
	err.AddSuggestion("다른 이름으로 시도해보세요")
	return err
}

// Internal 에러 생성 헬퍼 함수들

// NewInternalError는 내부 시스템 오류를 생성합니다.
func NewInternalError(component string, cause error) *CLIError {
	message := fmt.Sprintf("내부 오류가 발생했습니다: %s", component)
	err := NewCLIError(ErrorTypeInternal, message).WithCause(cause)
	err.AddContext("component", component)
	err.AddSuggestion("--verbose 플래그로 상세 로그를 확인하세요")
	err.AddSuggestion("문제가 지속되면 GitHub에 이슈를 생성하세요")
	err.AddSuggestion("'aicli version'으로 버전 정보를 확인하세요")
	return err
}

// IsType은 에러가 특정 타입인지 확인합니다.
func IsType(err error, errorType ErrorType) bool {
	if cliErr, ok := err.(*CLIError); ok {
		return cliErr.Type == errorType
	}
	return false
}

// GetExitCode는 에러의 종료 코드를 반환합니다.
func GetExitCode(err error) int {
	if cliErr, ok := err.(*CLIError); ok {
		return cliErr.ExitCode
	}
	return 1 // 기본 에러 코드
}