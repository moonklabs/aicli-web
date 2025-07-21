package validation

import (
	"context"
	"reflect"
	"regexp"
	"strings"
	"unicode"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// Validator 통합 검증 인터페이스
type Validator interface {
	// Validate 구조체 검증
	Validate(model interface{}) error
	// ValidateVar 단일 변수 검증
	ValidateVar(field interface{}, tag string) error
	// RegisterValidation 커스텀 검증 함수 등록
	RegisterValidation(tag string, fn validator.Func) error
}

// BusinessValidator 비즈니스 로직 검증 인터페이스
type BusinessValidator interface {
	ValidateCreate(ctx context.Context, model interface{}) error
	ValidateUpdate(ctx context.Context, model interface{}) error
	ValidateDelete(ctx context.Context, id string) error
}

// ValidationManager 검증 관리자
type ValidationManager struct {
	validator *validator.Validate
	business  map[reflect.Type]BusinessValidator
}

// NewValidationManager 새로운 검증 관리자 생성
func NewValidationManager() *ValidationManager {
	v := validator.New()
	
	// JSON 태그를 필드명으로 사용
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	manager := &ValidationManager{
		validator: v,
		business:  make(map[reflect.Type]BusinessValidator),
	}

	// 기본 커스텀 검증 함수들 등록
	manager.registerDefaultValidators()

	return manager
}

// GetGinValidator Gin의 validator를 반환하고 커스텀 검증자 등록
func GetGinValidator() *ValidationManager {
	manager := NewValidationManager()
	
	// Gin의 기본 validator에 커스텀 함수들 등록
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		// 기존 utils의 함수들과 통합
		v.RegisterValidation("dir", validateDirectory)
		v.RegisterValidation("safepath", validateSafePath)
		
		// 새로운 검증 함수들 추가
		v.RegisterValidation("workspace_status", validateWorkspaceStatus)
		v.RegisterValidation("project_status", validateProjectStatus)
		v.RegisterValidation("session_status", validateSessionStatus)
		v.RegisterValidation("task_status", validateTaskStatus)
		v.RegisterValidation("no_special_chars", validateNoSpecialChars)
		v.RegisterValidation("uuid", validateUUID)
		v.RegisterValidation("claude_api_key", validateClaudeAPIKey)
	}

	return manager
}

// Validate 구조체 검증
func (vm *ValidationManager) Validate(model interface{}) error {
	err := vm.validator.Struct(model)
	if err != nil {
		modelType := reflect.TypeOf(model)
		if modelType.Kind() == reflect.Ptr {
			modelType = modelType.Elem()
		}
		return TranslateValidatorError(err, modelType.Name())
	}
	return nil
}

// ValidateVar 단일 변수 검증
func (vm *ValidationManager) ValidateVar(field interface{}, tag string) error {
	err := vm.validator.Var(field, tag)
	if err != nil {
		return TranslateValidatorError(err, "field")
	}
	return nil
}

// RegisterValidation 커스텀 검증 함수 등록
func (vm *ValidationManager) RegisterValidation(tag string, fn validator.Func) error {
	return vm.validator.RegisterValidation(tag, fn)
}

// RegisterBusinessValidator 비즈니스 검증자 등록
func (vm *ValidationManager) RegisterBusinessValidator(model interface{}, validator BusinessValidator) {
	modelType := reflect.TypeOf(model)
	if modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}
	vm.business[modelType] = validator
}

// ValidateBusinessCreate 생성 시 비즈니스 로직 검증
func (vm *ValidationManager) ValidateBusinessCreate(ctx context.Context, model interface{}) error {
	modelType := reflect.TypeOf(model)
	if modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}
	
	if businessValidator, exists := vm.business[modelType]; exists {
		return businessValidator.ValidateCreate(ctx, model)
	}
	
	return nil
}

// ValidateBusinessUpdate 업데이트 시 비즈니스 로직 검증
func (vm *ValidationManager) ValidateBusinessUpdate(ctx context.Context, model interface{}) error {
	modelType := reflect.TypeOf(model)
	if modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}
	
	if businessValidator, exists := vm.business[modelType]; exists {
		return businessValidator.ValidateUpdate(ctx, model)
	}
	
	return nil
}

// ValidateBusinessDelete 삭제 시 비즈니스 로직 검증
func (vm *ValidationManager) ValidateBusinessDelete(ctx context.Context, model interface{}, id string) error {
	modelType := reflect.TypeOf(model)
	if modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}
	
	if businessValidator, exists := vm.business[modelType]; exists {
		return businessValidator.ValidateDelete(ctx, id)
	}
	
	return nil
}

// registerDefaultValidators 기본 커스텀 검증 함수들 등록
func (vm *ValidationManager) registerDefaultValidators() {
	vm.validator.RegisterValidation("workspace_status", validateWorkspaceStatus)
	vm.validator.RegisterValidation("project_status", validateProjectStatus)
	vm.validator.RegisterValidation("session_status", validateSessionStatus)
	vm.validator.RegisterValidation("task_status", validateTaskStatus)
	vm.validator.RegisterValidation("no_special_chars", validateNoSpecialChars)
	vm.validator.RegisterValidation("uuid", validateUUID)
	vm.validator.RegisterValidation("claude_api_key", validateClaudeAPIKey)
}

// validateWorkspaceStatus 워크스페이스 상태 검증
func validateWorkspaceStatus(fl validator.FieldLevel) bool {
	status := fl.Field().String()
	validStatuses := []string{"active", "inactive", "archived"}
	for _, valid := range validStatuses {
		if status == valid {
			return true
		}
	}
	return false
}

// validateProjectStatus 프로젝트 상태 검증
func validateProjectStatus(fl validator.FieldLevel) bool {
	status := fl.Field().String()
	validStatuses := []string{"active", "inactive", "archived"}
	for _, valid := range validStatuses {
		if status == valid {
			return true
		}
	}
	return false
}

// validateSessionStatus 세션 상태 검증
func validateSessionStatus(fl validator.FieldLevel) bool {
	status := fl.Field().String()
	validStatuses := []string{"pending", "active", "idle", "ending", "ended", "error"}
	for _, valid := range validStatuses {
		if status == valid {
			return true
		}
	}
	return false
}

// validateTaskStatus 태스크 상태 검증
func validateTaskStatus(fl validator.FieldLevel) bool {
	status := fl.Field().String()
	validStatuses := []string{"pending", "running", "completed", "failed", "cancelled"}
	for _, valid := range validStatuses {
		if status == valid {
			return true
		}
	}
	return false
}

// validateNoSpecialChars 특수문자 제외 검증
func validateNoSpecialChars(fl validator.FieldLevel) bool {
	str := fl.Field().String()
	if str == "" {
		return true // empty는 required 태그에서 처리
	}

	// 허용되지 않는 특수문자들
	dangerousChars := []rune{'<', '>', '&', '"', '\'', '/', '\\', '|', '?', '*', ':', ';'}
	
	for _, char := range str {
		// 제어 문자 검사
		if unicode.IsControl(char) {
			return false
		}
		
		// 위험한 특수문자 검사
		for _, dangerous := range dangerousChars {
			if char == dangerous {
				return false
			}
		}
	}
	
	return true
}

// validateUUID UUID v4 형식 검증
func validateUUID(fl validator.FieldLevel) bool {
	str := fl.Field().String()
	if str == "" {
		return true // empty는 required 태그에서 처리
	}

	// UUID v4 정규식
	uuidRegex := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	return uuidRegex.MatchString(strings.ToLower(str))
}

// validateClaudeAPIKey Claude API 키 형식 검증
func validateClaudeAPIKey(fl validator.FieldLevel) bool {
	key := fl.Field().String()
	if key == "" {
		return true // empty는 omitempty나 required 태그에서 처리
	}

	// Claude API 키는 'sk-ant-api03-' 으로 시작
	if !strings.HasPrefix(key, "sk-ant-api03-") {
		return false
	}

	// 최소 길이 확인 (실제 키는 더 길지만 기본적인 확인)
	if len(key) < 50 {
		return false
	}

	return true
}

// Global validator instance
var DefaultManager *ValidationManager

// Initialize 기본 검증 관리자 초기화
func Initialize() {
	DefaultManager = GetGinValidator()
}

// 편의 함수들
func Validate(model interface{}) error {
	if DefaultManager == nil {
		Initialize()
	}
	return DefaultManager.Validate(model)
}

func ValidateVar(field interface{}, tag string) error {
	if DefaultManager == nil {
		Initialize()
	}
	return DefaultManager.ValidateVar(field, tag)
}

func RegisterValidation(tag string, fn validator.Func) error {
	if DefaultManager == nil {
		Initialize()
	}
	return DefaultManager.RegisterValidation(tag, fn)
}

func RegisterBusinessValidator(model interface{}, validator BusinessValidator) {
	if DefaultManager == nil {
		Initialize()
	}
	DefaultManager.RegisterBusinessValidator(model, validator)
}

func ValidateBusinessCreate(ctx context.Context, model interface{}) error {
	if DefaultManager == nil {
		Initialize()
	}
	return DefaultManager.ValidateBusinessCreate(ctx, model)
}

func ValidateBusinessUpdate(ctx context.Context, model interface{}) error {
	if DefaultManager == nil {
		Initialize()
	}
	return DefaultManager.ValidateBusinessUpdate(ctx, model)
}

func ValidateBusinessDelete(ctx context.Context, model interface{}, id string) error {
	if DefaultManager == nil {
		Initialize()
	}
	return DefaultManager.ValidateBusinessDelete(ctx, model, id)
}