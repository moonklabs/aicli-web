package claude

import "fmt"

// ProcessError 프로세스 관련 에러
type ProcessError struct {
	Type    string
	Message string
	Cause   error
	PID     int
	Status  ProcessStatus
}

// Error 에러 메시지를 반환합니다
func (e *ProcessError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s (PID: %d, 상태: %s): %s: %v",
			e.Type, e.PID, e.Status, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s (PID: %d, 상태: %s): %s",
		e.Type, e.PID, e.Status, e.Message)
}

// Unwrap 래핑된 에러를 반환합니다
func (e *ProcessError) Unwrap() error {
	return e.Cause
}

// NewProcessError 새로운 프로세스 에러를 생성합니다
func NewProcessError(errorType, message string, cause error, pid int, status ProcessStatus) *ProcessError {
	return &ProcessError{
		Type:    errorType,
		Message: message,
		Cause:   cause,
		PID:     pid,
		Status:  status,
	}
}

// 일반적인 프로세스 에러 타입
const (
	// ErrTypeStartFailed 프로세스 시작 실패
	ErrTypeStartFailed = "START_FAILED"
	// ErrTypeStopFailed 프로세스 중지 실패
	ErrTypeStopFailed = "STOP_FAILED"
	// ErrTypeKillFailed 프로세스 강제 종료 실패
	ErrTypeKillFailed = "KILL_FAILED"
	// ErrTypeHealthCheckFailed 헬스체크 실패
	ErrTypeHealthCheckFailed = "HEALTH_CHECK_FAILED"
	// ErrTypeUnexpectedExit 예기치 않은 종료
	ErrTypeUnexpectedExit = "UNEXPECTED_EXIT"
	// ErrTypeTimeout 타임아웃
	ErrTypeTimeout = "TIMEOUT"
	// ErrTypeInvalidState 잘못된 상태
	ErrTypeInvalidState = "INVALID_STATE"
)