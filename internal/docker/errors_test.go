package docker

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsDockerError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "docker error",
			err:      errors.New("docker daemon not available"),
			expected: true,
		},
		{
			name:     "container error",
			err:      errors.New("container not found"),
			expected: true,
		},
		{
			name:     "daemon error",
			err:      errors.New("daemon connection failed"),
			expected: true,
		},
		{
			name:     "non-docker error",
			err:      errors.New("file not found"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsDockerError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDockerError(t *testing.T) {
	cause := errors.New("original error")
	dockerErr := NewDockerError(ErrorTypeConnection, "CONNECTION_FAILED", "Failed to connect", cause)

	assert.Equal(t, ErrorTypeConnection, dockerErr.Type)
	assert.Equal(t, "CONNECTION_FAILED", dockerErr.Code)
	assert.Equal(t, "Failed to connect", dockerErr.Message)
	assert.Equal(t, cause, dockerErr.Cause)

	// Error() method test
	expected := "Failed to connect: original error"
	assert.Equal(t, expected, dockerErr.Error())

	// Unwrap() method test
	assert.Equal(t, cause, dockerErr.Unwrap())
}

func TestDockerError_WithoutCause(t *testing.T) {
	dockerErr := NewDockerError(ErrorTypeContainer, "CONTAINER_NOT_FOUND", "Container not found", nil)

	assert.Equal(t, ErrorTypeContainer, dockerErr.Type)
	assert.Equal(t, "CONTAINER_NOT_FOUND", dockerErr.Code)
	assert.Equal(t, "Container not found", dockerErr.Message)
	assert.Nil(t, dockerErr.Cause)

	// Error() method should return just the message
	assert.Equal(t, "Container not found", dockerErr.Error())

	// Unwrap() should return nil
	assert.Nil(t, dockerErr.Unwrap())
}

func TestClassifyError(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedType ErrorType
		expectedCode string
	}{
		{
			name:         "nil error",
			err:          nil,
			expectedType: ErrorType(0), // This will be nil, not compared
		},
		{
			name:         "connection refused",
			err:          errors.New("connection refused"),
			expectedType: ErrorTypeConnection,
			expectedCode: "CONNECTION_FAILED",
		},
		{
			name:         "timeout error",
			err:          errors.New("timeout occurred"),
			expectedType: ErrorTypeTimeout,
			expectedCode: "TIMEOUT",
		},
		{
			name:         "container not found",
			err:          errors.New("no such container"),
			expectedType: ErrorTypeContainer,
			expectedCode: "CONTAINER_NOT_FOUND",
		},
		{
			name:         "image not found",
			err:          errors.New("no such image"),
			expectedType: ErrorTypeImage,
			expectedCode: "IMAGE_NOT_FOUND",
		},
		{
			name:         "image pull access denied",
			err:          errors.New("pull access denied"),
			expectedType: ErrorTypeImage,
			expectedCode: "IMAGE_NOT_FOUND",
		},
		{
			name:         "network error",
			err:          errors.New("network operation failed"),
			expectedType: ErrorTypeNetwork,
			expectedCode: "NETWORK_ERROR",
		},
		{
			name:         "permission denied",
			err:          errors.New("permission denied"),
			expectedType: ErrorTypePermission,
			expectedCode: "PERMISSION_DENIED",
		},
		{
			name:         "context deadline exceeded",
			err:          errors.New("context deadline exceeded"),
			expectedType: ErrorTypeTimeout,
			expectedCode: "TIMEOUT",
		},
		{
			name:         "unknown error",
			err:          errors.New("some random error"),
			expectedType: ErrorTypeUnknown,
			expectedCode: "UNKNOWN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyError(tt.err)
			
			if tt.err == nil {
				assert.Nil(t, result)
				return
			}
			
			assert.NotNil(t, result)
			assert.Equal(t, tt.expectedType, result.Type)
			assert.Equal(t, tt.expectedCode, result.Code)
			assert.Equal(t, tt.err, result.Cause)
		})
	}
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		retryable bool
	}{
		{
			name:      "nil error",
			err:       nil,
			retryable: false,
		},
		{
			name:      "connection error",
			err:       NewDockerError(ErrorTypeConnection, "CONNECTION_FAILED", "connection failed", nil),
			retryable: true,
		},
		{
			name:      "timeout error",
			err:       NewDockerError(ErrorTypeTimeout, "TIMEOUT", "timeout", nil),
			retryable: true,
		},
		{
			name:      "temporary network error",
			err:       NewDockerError(ErrorTypeNetwork, "NETWORK_ERROR", "temporary failure", nil),
			retryable: true,
		},
		{
			name:      "permanent network error",
			err:       NewDockerError(ErrorTypeNetwork, "NETWORK_ERROR", "permanent failure", nil),
			retryable: false,
		},
		{
			name:      "container error",
			err:       NewDockerError(ErrorTypeContainer, "CONTAINER_NOT_FOUND", "container not found", nil),
			retryable: false,
		},
		{
			name:      "image error",
			err:       NewDockerError(ErrorTypeImage, "IMAGE_NOT_FOUND", "image not found", nil),
			retryable: false,
		},
		{
			name:      "permission error",
			err:       NewDockerError(ErrorTypePermission, "PERMISSION_DENIED", "permission denied", nil),
			retryable: false,
		},
		{
			name:      "raw connection error",
			err:       errors.New("connection refused"),
			retryable: true,
		},
		{
			name:      "raw container error",
			err:       errors.New("no such container"),
			retryable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetryableError(tt.err)
			assert.Equal(t, tt.retryable, result)
		})
	}
}

func TestGetErrorMessage(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: "",
		},
		{
			name:     "connection error",
			err:      NewDockerError(ErrorTypeConnection, "CONNECTION_FAILED", "connection failed", nil),
			expected: "Docker daemon에 연결할 수 없습니다. Docker가 실행 중인지 확인해주세요.",
		},
		{
			name:     "container error",
			err:      NewDockerError(ErrorTypeContainer, "CONTAINER_NOT_FOUND", "container not found", nil),
			expected: "지정된 컨테이너를 찾을 수 없습니다.",
		},
		{
			name:     "image error",
			err:      NewDockerError(ErrorTypeImage, "IMAGE_NOT_FOUND", "image not found", nil),
			expected: "지정된 이미지를 찾을 수 없거나 접근 권한이 없습니다.",
		},
		{
			name:     "network error",
			err:      NewDockerError(ErrorTypeNetwork, "NETWORK_ERROR", "network failed", nil),
			expected: "네트워크 작업 중 오류가 발생했습니다.",
		},
		{
			name:     "permission error",
			err:      NewDockerError(ErrorTypePermission, "PERMISSION_DENIED", "permission denied", nil),
			expected: "Docker 작업에 필요한 권한이 없습니다.",
		},
		{
			name:     "timeout error",
			err:      NewDockerError(ErrorTypeTimeout, "TIMEOUT", "timeout", nil),
			expected: "작업이 시간 초과되었습니다.",
		},
		{
			name:     "unknown error",
			err:      NewDockerError(ErrorTypeUnknown, "UNKNOWN", "unknown error", nil),
			expected: "unknown error",
		},
		{
			name:     "raw error gets classified",
			err:      errors.New("connection refused"),
			expected: "Docker daemon에 연결할 수 없습니다. Docker가 실행 중인지 확인해주세요.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetErrorMessage(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPredefinedErrors(t *testing.T) {
	// 미리 정의된 에러들이 올바른 메시지를 가지고 있는지 확인
	assert.Equal(t, "docker daemon not available", ErrDockerNotAvailable.Error())
	assert.Equal(t, "docker network not found", ErrNetworkNotFound.Error())
	assert.Equal(t, "docker image not found", ErrImageNotFound.Error())
	assert.Equal(t, "container not found", ErrContainerNotFound.Error())
	assert.Equal(t, "invalid docker configuration", ErrInvalidConfig.Error())
}

func BenchmarkClassifyError(b *testing.B) {
	testError := errors.New("connection refused by docker daemon")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ClassifyError(testError)
	}
}

func BenchmarkIsRetryableError(b *testing.B) {
	testError := NewDockerError(ErrorTypeConnection, "CONNECTION_FAILED", "connection failed", nil)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = IsRetryableError(testError)
	}
}