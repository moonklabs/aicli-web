package docker

import (
	"errors"
	"strings"
)

// Docker 관련 에러들
var (
	ErrDockerNotAvailable = errors.New("docker daemon not available")
	ErrNetworkNotFound    = errors.New("docker network not found")
	ErrImageNotFound      = errors.New("docker image not found")
	ErrContainerNotFound  = errors.New("container not found")
	ErrInvalidConfig      = errors.New("invalid docker configuration")
)

// IsDockerError Docker 관련 에러인지 확인합니다.
func IsDockerError(err error) bool {
	if err == nil {
		return false
	}

	// Docker SDK에서 발생하는 에러 타입 확인
	return strings.Contains(err.Error(), "docker") ||
		strings.Contains(err.Error(), "container") ||
		strings.Contains(err.Error(), "daemon")
}

// ErrorType 에러 타입을 나타내는 열거형
type ErrorType int

const (
	ErrorTypeConnection ErrorType = iota
	ErrorTypeContainer
	ErrorTypeImage
	ErrorTypeNetwork
	ErrorTypePermission
	ErrorTypeTimeout
	ErrorTypeUnknown
)

// DockerError Docker 관련 상세 에러 정보
type DockerError struct {
	Type    ErrorType `json:"type"`
	Code    string    `json:"code"`
	Message string    `json:"message"`
	Cause   error     `json:"cause,omitempty"`
}

// Error Error 인터페이스 구현
func (de *DockerError) Error() string {
	if de.Cause != nil {
		return de.Message + ": " + de.Cause.Error()
	}
	return de.Message
}

// Unwrap 원본 에러를 반환합니다.
func (de *DockerError) Unwrap() error {
	return de.Cause
}

// NewDockerError 새로운 Docker 에러를 생성합니다.
func NewDockerError(errType ErrorType, code, message string, cause error) *DockerError {
	return &DockerError{
		Type:    errType,
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// ClassifyError 에러를 분류하여 Docker 에러로 변환합니다.
func ClassifyError(err error) *DockerError {
	if err == nil {
		return nil
	}

	errStr := err.Error()

	// 연결 관련 에러
	if strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "no such host") ||
		strings.Contains(errStr, "timeout") {
		return NewDockerError(ErrorTypeConnection, "CONNECTION_FAILED", "Failed to connect to Docker daemon", err)
	}

	// 컨테이너 관련 에러
	if strings.Contains(errStr, "no such container") ||
		strings.Contains(errStr, "container not found") {
		return NewDockerError(ErrorTypeContainer, "CONTAINER_NOT_FOUND", "Container not found", err)
	}

	// 이미지 관련 에러
	if strings.Contains(errStr, "no such image") ||
		strings.Contains(errStr, "image not found") ||
		strings.Contains(errStr, "pull access denied") {
		return NewDockerError(ErrorTypeImage, "IMAGE_NOT_FOUND", "Image not found or access denied", err)
	}

	// 네트워크 관련 에러
	if strings.Contains(errStr, "network") {
		return NewDockerError(ErrorTypeNetwork, "NETWORK_ERROR", "Network operation failed", err)
	}

	// 권한 관련 에러
	if strings.Contains(errStr, "permission denied") ||
		strings.Contains(errStr, "unauthorized") {
		return NewDockerError(ErrorTypePermission, "PERMISSION_DENIED", "Permission denied", err)
	}

	// 타임아웃 에러
	if strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "context deadline exceeded") {
		return NewDockerError(ErrorTypeTimeout, "TIMEOUT", "Operation timed out", err)
	}

	// 기타 에러
	return NewDockerError(ErrorTypeUnknown, "UNKNOWN", "Unknown Docker error", err)
}

// IsRetryableError 재시도 가능한 에러인지 확인합니다.
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	dockerErr, ok := err.(*DockerError)
	if !ok {
		dockerErr = ClassifyError(err)
	}

	// 재시도 가능한 에러 타입들
	switch dockerErr.Type {
	case ErrorTypeConnection, ErrorTypeTimeout:
		return true
	case ErrorTypeNetwork:
		// 일부 네트워크 에러는 재시도 가능
		return strings.Contains(dockerErr.Message, "temporary")
	default:
		return false
	}
}

// GetErrorMessage 사용자 친화적인 에러 메시지를 반환합니다.
func GetErrorMessage(err error) string {
	if err == nil {
		return ""
	}

	dockerErr, ok := err.(*DockerError)
	if !ok {
		dockerErr = ClassifyError(err)
	}

	switch dockerErr.Type {
	case ErrorTypeConnection:
		return "Docker daemon에 연결할 수 없습니다. Docker가 실행 중인지 확인해주세요."
	case ErrorTypeContainer:
		return "지정된 컨테이너를 찾을 수 없습니다."
	case ErrorTypeImage:
		return "지정된 이미지를 찾을 수 없거나 접근 권한이 없습니다."
	case ErrorTypeNetwork:
		return "네트워크 작업 중 오류가 발생했습니다."
	case ErrorTypePermission:
		return "Docker 작업에 필요한 권한이 없습니다."
	case ErrorTypeTimeout:
		return "작업이 시간 초과되었습니다."
	default:
		return dockerErr.Message
	}
}