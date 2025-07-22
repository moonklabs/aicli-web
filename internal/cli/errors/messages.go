// Package errors는 CLI 명령어에서 사용하는 레거시 에러 메시지 함수들을 제공합니다.
// 새로운 코드에서는 internal/errors 패키지의 CLIError 시스템을 사용하는 것을 권장합니다.
package errors

import (
	"fmt"

	cliErrors "github.com/aicli/aicli-web/internal/errors"
)

// RequiredFlagError는 필수 플래그가 누락되었을 때 사용합니다.
// 호환성을 위해 유지되지만, 새로운 코드에서는 cliErrors.NewRequiredFlagError를 사용하세요.
func RequiredFlagError(flagName, description string) error {
	return cliErrors.NewRequiredFlagError(flagName, description)
}

// InvalidValueError는 잘못된 값이 입력되었을 때 사용합니다.
// 호환성을 위해 유지되지만, 새로운 코드에서는 cliErrors.NewInvalidValueError를 사용하세요.
func InvalidValueError(field, value string, validValues []string) error {
	return cliErrors.NewInvalidValueError(field, value, validValues)
}

// NotFoundError는 리소스를 찾을 수 없을 때 사용합니다.
// 호환성을 위해 유지되지만, 새로운 코드에서는 cliErrors.NewNotFoundError를 사용하세요.
func NotFoundError(resourceType, name string) error {
	return cliErrors.NewNotFoundError(resourceType, name)
}

// ConnectionError는 연결 문제가 발생했을 때 사용합니다.
// 호환성을 위해 유지되지만, 새로운 코드에서는 cliErrors.NewNetworkError를 사용하세요.
func ConnectionError(service string, err error) error {
	return cliErrors.NewNetworkError(service, err)
}

// PermissionError는 권한 문제가 발생했을 때 사용합니다.
// 호환성을 위해 유지되지만, 새로운 코드에서는 cliErrors.NewPermissionError를 사용하세요.
func PermissionError(action, resource string) error {
	return cliErrors.NewPermissionError(action, resource)
}

// ConfigError는 설정 관련 문제가 발생했을 때 사용합니다.
// 호환성을 위해 유지되지만, 새로운 코드에서는 cliErrors.NewConfigKeyError를 사용하세요.
func ConfigError(key string, err error) error {
	return cliErrors.NewConfigKeyError(key, err)
}

// ValidationError는 입력값 검증 실패 시 사용합니다.
// 호환성을 위해 유지되지만, 새로운 코드에서는 cliErrors.NewValidationError를 사용하세요.
func ValidationError(field, rule string) error {
	message := fmt.Sprintf("검증 실패: %s", field)
	suggestion := fmt.Sprintf("규칙: %s", rule)
	return cliErrors.NewValidationError(message, suggestion, "도움말을 보려면 'aicli help [command]'를 사용하세요")
}