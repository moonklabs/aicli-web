package errors

import (
	"fmt"
)

// 표준화된 에러 메시지 생성 함수들

// RequiredFlagError는 필수 플래그가 누락되었을 때 사용합니다.
func RequiredFlagError(flagName, description string) error {
	return fmt.Errorf(`필수 플래그가 누락되었습니다: --%s

설명: %s

사용 예시:
  aicli [command] --%s [value]

도움말을 보려면 'aicli help [command]'를 사용하세요.`, flagName, description, flagName)
}

// InvalidValueError는 잘못된 값이 입력되었을 때 사용합니다.
func InvalidValueError(field, value string, validValues []string) error {
	validStr := ""
	if len(validValues) > 0 {
		validStr = fmt.Sprintf("\n\n유효한 값: %v", validValues)
	}
	
	return fmt.Errorf(`잘못된 값입니다: %s = "%s"%s

도움말을 보려면 'aicli help [command]'를 사용하세요.`, field, value, validStr)
}

// NotFoundError는 리소스를 찾을 수 없을 때 사용합니다.
func NotFoundError(resourceType, name string) error {
	return fmt.Errorf(`%s를 찾을 수 없습니다: %s

다음을 확인해주세요:
  • 이름이 정확한지 확인
  • 'aicli %s list'로 존재하는 목록 확인
  • 리소스가 생성되었는지 확인`, resourceType, name, resourceType)
}

// ConnectionError는 연결 문제가 발생했을 때 사용합니다.
func ConnectionError(service string, err error) error {
	return fmt.Errorf(`%s에 연결할 수 없습니다.

문제: %v

해결 방법:
  • %s 서비스가 실행 중인지 확인
  • 네트워크 연결 상태 확인
  • 방화벽 설정 확인
  • 'aicli config list'로 설정 확인`, service, err, service)
}

// PermissionError는 권한 문제가 발생했을 때 사용합니다.
func PermissionError(action, resource string) error {
	return fmt.Errorf(`권한이 없습니다: %s - %s

해결 방법:
  • 파일/디렉토리 권한 확인
  • 현재 사용자의 권한 확인
  • sudo를 사용하여 재시도 (필요한 경우)`, action, resource)
}

// ConfigError는 설정 관련 문제가 발생했을 때 사용합니다.
func ConfigError(key string, err error) error {
	return fmt.Errorf(`설정 오류: %s

문제: %v

해결 방법:
  • 'aicli config get %s'로 현재 값 확인
  • 'aicli config set %s [value]'로 올바른 값 설정
  • ~/.aicli.yaml 파일 직접 편집`, key, err, key, key)
}

// ValidationError는 입력값 검증 실패 시 사용합니다.
func ValidationError(field, rule string) error {
	return fmt.Errorf(`검증 실패: %s

규칙: %s

도움말을 보려면 'aicli help [command]'를 사용하세요.`, field, rule)
}