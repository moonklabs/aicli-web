package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-playground/validator/v10"
)

// 사용자 정의 검증 태그
const (
	tagDir         = "dir"
	tagFile        = "file"
	tagHostnamePort = "hostname_port"
)

// ConfigValidator는 설정 검증을 담당합니다
type ConfigValidator struct {
	validator *validator.Validate
}

// NewConfigValidator는 새로운 ConfigValidator를 생성합니다
func NewConfigValidator() *ConfigValidator {
	v := validator.New()
	
	// 사용자 정의 검증 함수 등록
	v.RegisterValidation(tagDir, validateDirectory)
	v.RegisterValidation(tagFile, validateFile)
	v.RegisterValidation(tagHostnamePort, validateHostnamePort)
	
	return &ConfigValidator{
		validator: v,
	}
}

// Validate는 설정을 검증합니다
func (cv *ConfigValidator) Validate(cfg *Config) error {
	if err := cv.validator.Struct(cfg); err != nil {
		return fmt.Errorf("설정 검증 실패: %w", err)
	}
	
	// 추가 비즈니스 규칙 검증
	if err := cv.validateBusinessRules(cfg); err != nil {
		return err
	}
	
	return nil
}

// validateBusinessRules는 비즈니스 규칙을 검증합니다
func (cv *ConfigValidator) validateBusinessRules(cfg *Config) error {
	// Claude API 키 검증
	if cfg.Claude.APIKey == "" {
		return fmt.Errorf("Claude API 키가 설정되지 않았습니다")
	}
	
	// JWT Secret 검증 (API 서버가 활성화된 경우)
	if cfg.API.TLSEnabled || cfg.API.Address != "" {
		if len(cfg.API.JWTSecret) < 32 {
			return fmt.Errorf("JWT Secret은 최소 32자 이상이어야 합니다")
		}
	}
	
	// TLS 설정 검증
	if cfg.API.TLSEnabled {
		if cfg.API.TLSCertPath == "" || cfg.API.TLSKeyPath == "" {
			return fmt.Errorf("TLS가 활성화되었지만 인증서 경로가 설정되지 않았습니다")
		}
		if _, err := os.Stat(cfg.API.TLSCertPath); err != nil {
			return fmt.Errorf("TLS 인증서 파일을 찾을 수 없습니다: %s", cfg.API.TLSCertPath)
		}
		if _, err := os.Stat(cfg.API.TLSKeyPath); err != nil {
			return fmt.Errorf("TLS 키 파일을 찾을 수 없습니다: %s", cfg.API.TLSKeyPath)
		}
	}
	
	// Docker 설정 검증 (격리 모드가 docker인 경우)
	if cfg.Workspace.IsolationMode == "docker" {
		if _, err := os.Stat(cfg.Docker.SocketPath); err != nil {
			return fmt.Errorf("Docker 소켓을 찾을 수 없습니다: %s", cfg.Docker.SocketPath)
		}
	}
	
	// 메모리 제한 검증
	if cfg.Docker.MemoryLimit < 128 {
		return fmt.Errorf("Docker 메모리 제한은 최소 128MB 이상이어야 합니다")
	}
	
	// CPU 제한 검증
	if cfg.Docker.CPULimit < 0.1 {
		return fmt.Errorf("Docker CPU 제한은 최소 0.1 이상이어야 합니다")
	}
	
	return nil
}

// validateDirectory는 디렉토리 존재 여부를 검증합니다
func validateDirectory(fl validator.FieldLevel) bool {
	path := fl.Field().String()
	if path == "" {
		return true // 빈 문자열은 허용 (required 태그로 별도 처리)
	}
	
	info, err := os.Stat(path)
	if err != nil {
		// 디렉토리가 존재하지 않으면 생성 시도
		if os.IsNotExist(err) {
			// 설정 검증 단계에서는 디렉토리를 생성하지 않고 경로만 검증
			return true
		}
		return false
	}
	
	return info.IsDir()
}

// validateFile은 파일 존재 여부를 검증합니다
func validateFile(fl validator.FieldLevel) bool {
	path := fl.Field().String()
	if path == "" {
		return true // 빈 문자열은 허용
	}
	
	info, err := os.Stat(path)
	if err != nil {
		return os.IsNotExist(err) // 파일이 존재하지 않아도 OK (나중에 생성될 수 있음)
	}
	
	return !info.IsDir()
}

// validateHostnamePort는 hostname:port 형식을 검증합니다
func validateHostnamePort(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return true
	}
	
	parts := strings.Split(value, ":")
	if len(parts) != 2 {
		return false
	}
	
	// 포트 번호 검증은 validator의 port 태그가 처리
	return true
}

// ValidationError는 검증 오류를 나타냅니다
type ValidationError struct {
	Field   string
	Message string
}

// FormatValidationErrors는 검증 오류를 사용자 친화적인 형식으로 변환합니다
func FormatValidationErrors(err error) []ValidationError {
	var errors []ValidationError
	
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, e := range validationErrors {
			var message string
			
			switch e.Tag() {
			case "required":
				message = "필수 항목입니다"
			case "min":
				message = fmt.Sprintf("최소값은 %s입니다", e.Param())
			case "max":
				message = fmt.Sprintf("최대값은 %s입니다", e.Param())
			case "oneof":
				message = fmt.Sprintf("다음 중 하나여야 합니다: %s", e.Param())
			case tagDir:
				message = "유효한 디렉토리 경로가 아닙니다"
			case tagFile:
				message = "유효한 파일 경로가 아닙니다"
			case tagHostnamePort:
				message = "hostname:port 형식이어야 합니다"
			default:
				message = fmt.Sprintf("검증 실패: %s", e.Tag())
			}
			
			errors = append(errors, ValidationError{
				Field:   e.Namespace(),
				Message: message,
			})
		}
	}
	
	return errors
}