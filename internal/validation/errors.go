package validation

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

// ValidationError 단일 검증 에러
type ValidationError struct {
	Field   string      `json:"field"`
	Value   interface{} `json:"value,omitempty"`
	Tag     string      `json:"tag"`
	Message string      `json:"message"`
	Param   string      `json:"param,omitempty"`
}

// ValidationErrors 검증 에러 집합
type ValidationErrors struct {
	Errors []ValidationError `json:"errors"`
	Model  string            `json:"model,omitempty"`
}

// Error error 인터페이스 구현
func (e ValidationErrors) Error() string {
	if len(e.Errors) == 0 {
		return "validation failed"
	}

	var messages []string
	for _, err := range e.Errors {
		messages = append(messages, err.Message)
	}

	if e.Model != "" {
		return fmt.Sprintf("%s validation failed: %s", e.Model, strings.Join(messages, "; "))
	}
	return fmt.Sprintf("validation failed: %s", strings.Join(messages, "; "))
}

// HasField 특정 필드의 에러가 있는지 확인
func (e ValidationErrors) HasField(field string) bool {
	for _, err := range e.Errors {
		if err.Field == field {
			return true
		}
	}
	return false
}

// GetFieldErrors 특정 필드의 에러들 반환
func (e ValidationErrors) GetFieldErrors(field string) []ValidationError {
	var errors []ValidationError
	for _, err := range e.Errors {
		if err.Field == field {
			errors = append(errors, err)
		}
	}
	return errors
}

// BusinessValidationError 비즈니스 로직 검증 에러
type BusinessValidationError struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Field   string      `json:"field,omitempty"`
	Value   interface{} `json:"value,omitempty"`
	Details string      `json:"details,omitempty"`
}

// Error error 인터페이스 구현
func (e BusinessValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("%s: %s (field: %s)", e.Code, e.Message, e.Field)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// 사전 정의된 비즈니스 에러 코드
const (
	ErrCodeDuplicateName        = "DUPLICATE_NAME"
	ErrCodeInvalidStatus        = "INVALID_STATUS"
	ErrCodeResourceNotFound     = "RESOURCE_NOT_FOUND"
	ErrCodePermissionDenied     = "PERMISSION_DENIED"
	ErrCodeResourceLimit        = "RESOURCE_LIMIT"
	ErrCodeDependencyExists     = "DEPENDENCY_EXISTS"
	ErrCodeInvalidConfiguration = "INVALID_CONFIGURATION"
	ErrCodePathNotAccessible    = "PATH_NOT_ACCESSIBLE"
)

// 사전 정의된 비즈니스 검증 에러들
var (
	ErrDuplicateWorkspaceName = BusinessValidationError{
		Code:    ErrCodeDuplicateName,
		Message: "워크스페이스 이름이 이미 존재합니다",
		Field:   "name",
	}

	ErrDuplicateProjectName = BusinessValidationError{
		Code:    ErrCodeDuplicateName,
		Message: "프로젝트 이름이 이미 존재합니다",
		Field:   "name",
	}

	ErrWorkspaceNotFound = BusinessValidationError{
		Code:    ErrCodeResourceNotFound,
		Message: "워크스페이스를 찾을 수 없습니다",
	}

	ErrProjectNotFound = BusinessValidationError{
		Code:    ErrCodeResourceNotFound,
		Message: "프로젝트를 찾을 수 없습니다",
	}

	ErrSessionNotFound = BusinessValidationError{
		Code:    ErrCodeResourceNotFound,
		Message: "세션을 찾을 수 없습니다",
	}

	ErrMaxWorkspaceLimit = BusinessValidationError{
		Code:    ErrCodeResourceLimit,
		Message: "최대 워크스페이스 수를 초과했습니다",
	}

	ErrMaxProjectLimit = BusinessValidationError{
		Code:    ErrCodeResourceLimit,
		Message: "최대 프로젝트 수를 초과했습니다",
	}

	ErrActiveTasksExist = BusinessValidationError{
		Code:    ErrCodeDependencyExists,
		Message: "활성 태스크가 존재하여 삭제할 수 없습니다",
	}

	ErrPathNotAccessible = BusinessValidationError{
		Code:    ErrCodePathNotAccessible,
		Message: "지정된 경로에 접근할 수 없습니다",
		Field:   "path",
	}
)

// NewBusinessValidationError 새로운 비즈니스 검증 에러 생성
func NewBusinessValidationError(code, message string, field ...string) BusinessValidationError {
	err := BusinessValidationError{
		Code:    code,
		Message: message,
	}
	if len(field) > 0 {
		err.Field = field[0]
	}
	return err
}

// TranslateValidatorError validator.ValidationErrors를 ValidationErrors로 변환
func TranslateValidatorError(err error, model string) ValidationErrors {
	var validationErrors ValidationErrors
	validationErrors.Model = model

	if ve, ok := err.(validator.ValidationErrors); ok {
		for _, fieldErr := range ve {
			validationError := ValidationError{
				Field: getJSONFieldName(fieldErr),
				Value: fieldErr.Value(),
				Tag:   fieldErr.Tag(),
				Param: fieldErr.Param(),
			}

			// 메시지 번역
			validationError.Message = translateFieldError(fieldErr)
			validationErrors.Errors = append(validationErrors.Errors, validationError)
		}
	} else {
		// 다른 타입의 에러인 경우 일반 메시지로 변환
		validationErrors.Errors = append(validationErrors.Errors, ValidationError{
			Message: err.Error(),
			Tag:     "unknown",
		})
	}

	return validationErrors
}

// getJSONFieldName 구조체 필드의 JSON 태그명 반환
func getJSONFieldName(fe validator.FieldError) string {
	// validator에서 제공하는 네임스페이스를 파싱
	fieldName := fe.Field()
	
	// JSON 태그명으로 변환하는 로직 (간단한 변환)
	// 실제로는 리플렉션을 사용해서 더 정확하게 할 수 있음
	return strings.ToLower(strings.ReplaceAll(fieldName, ".", "_"))
}

// translateFieldError 필드 에러를 한국어 메시지로 번역
func translateFieldError(fe validator.FieldError) string {
	field := fe.Field()
	tag := fe.Tag()
	param := fe.Param()

	switch tag {
	case "required":
		return fmt.Sprintf("%s 필드는 필수입니다", field)
	case "min":
		return fmt.Sprintf("%s 필드는 최소 %s자 이상이어야 합니다", field, param)
	case "max":
		return fmt.Sprintf("%s 필드는 최대 %s자 이하여야 합니다", field, param)
	case "email":
		return fmt.Sprintf("%s 필드는 유효한 이메일 주소여야 합니다", field)
	case "uuid":
		return fmt.Sprintf("%s 필드는 유효한 UUID여야 합니다", field)
	case "dir":
		return fmt.Sprintf("%s 경로는 존재하는 디렉토리여야 합니다", field)
	case "safepath":
		return fmt.Sprintf("%s 경로에 위험한 문자가 포함되어 있습니다", field)
	case "oneof":
		return fmt.Sprintf("%s 필드는 다음 값 중 하나여야 합니다: %s", field, param)
	case "workspace_status":
		return fmt.Sprintf("%s 필드는 유효한 워크스페이스 상태여야 합니다", field)
	case "project_status":
		return fmt.Sprintf("%s 필드는 유효한 프로젝트 상태여야 합니다", field)
	case "session_status":
		return fmt.Sprintf("%s 필드는 유효한 세션 상태여야 합니다", field)
	case "task_status":
		return fmt.Sprintf("%s 필드는 유효한 태스크 상태여야 합니다", field)
	default:
		return fmt.Sprintf("%s 필드가 유효하지 않습니다 (규칙: %s)", field, tag)
	}
}

// ToJSON ValidationErrors를 JSON으로 직렬화
func (e ValidationErrors) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

// FromJSON JSON을 ValidationErrors로 역직렬화
func FromJSON(data []byte) (ValidationErrors, error) {
	var errors ValidationErrors
	err := json.Unmarshal(data, &errors)
	return errors, err
}