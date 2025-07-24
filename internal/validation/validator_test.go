package validation

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 테스트용 구조체
type TestStruct struct {
	ID          string `validate:"required,uuid"`
	Name        string `validate:"required,min=1,max=100,no_special_chars"`
	Email       string `validate:"required,email"`
	Age         int    `validate:"min=0,max=150"`
	Status      string `validate:"required,oneof=active inactive"`
	OptionalURL string `validate:"omitempty,url"`
}

func TestValidationManager_Validate(t *testing.T) {
	manager := NewValidationManager()

	tests := []struct {
		name      string
		input     TestStruct
		wantError bool
	}{
		{
			name: "유효한 데이터",
			input: TestStruct{
				ID:     "123e4567-e89b-12d3-a456-426614174000",
				Name:   "테스트 이름",
				Email:  "test@example.com",
				Age:    25,
				Status: "active",
			},
			wantError: false,
		},
		{
			name: "필수 필드 누락",
			input: TestStruct{
				Name:  "테스트 이름",
				Email: "test@example.com",
				Age:   25,
			},
			wantError: true,
		},
		{
			name: "잘못된 UUID",
			input: TestStruct{
				ID:     "invalid-uuid",
				Name:   "테스트 이름",
				Email:  "test@example.com",
				Age:    25,
				Status: "active",
			},
			wantError: true,
		},
		{
			name: "이름 길이 초과",
			input: TestStruct{
				ID:     "123e4567-e89b-12d3-a456-426614174000",
				Name:   string(make([]byte, 101)), // 101자
				Email:  "test@example.com",
				Age:    25,
				Status: "active",
			},
			wantError: true,
		},
		{
			name: "잘못된 이메일",
			input: TestStruct{
				ID:     "123e4567-e89b-12d3-a456-426614174000",
				Name:   "테스트 이름",
				Email:  "invalid-email",
				Age:    25,
				Status: "active",
			},
			wantError: true,
		},
		{
			name: "나이 범위 초과",
			input: TestStruct{
				ID:     "123e4567-e89b-12d3-a456-426614174000",
				Name:   "테스트 이름",
				Email:  "test@example.com",
				Age:    200,
				Status: "active",
			},
			wantError: true,
		},
		{
			name: "유효하지 않은 상태",
			input: TestStruct{
				ID:     "123e4567-e89b-12d3-a456-426614174000",
				Name:   "테스트 이름",
				Email:  "test@example.com",
				Age:    25,
				Status: "invalid",
			},
			wantError: true,
		},
		{
			name: "선택적 필드 - 유효한 URL",
			input: TestStruct{
				ID:          "123e4567-e89b-12d3-a456-426614174000",
				Name:        "테스트 이름",
				Email:       "test@example.com",
				Age:         25,
				Status:      "active",
				OptionalURL: "https://example.com",
			},
			wantError: false,
		},
		{
			name: "선택적 필드 - 잘못된 URL",
			input: TestStruct{
				ID:          "123e4567-e89b-12d3-a456-426614174000",
				Name:        "테스트 이름",
				Email:       "test@example.com",
				Age:         25,
				Status:      "active",
				OptionalURL: "invalid-url",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.Validate(tt.input)

			if tt.wantError {
				assert.Error(t, err)
				
				// ValidationErrors 타입 확인
				var validationErrors ValidationErrors
				assert.IsType(t, validationErrors, err)
				
				// 에러가 적절한 구조를 갖고 있는지 확인
				vErr := err.(ValidationErrors)
				assert.NotEmpty(t, vErr.Errors)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidationManager_ValidateVar(t *testing.T) {
	manager := NewValidationManager()

	tests := []struct {
		name      string
		value     interface{}
		tag       string
		wantError bool
	}{
		{
			name:      "유효한 UUID",
			value:     "123e4567-e89b-12d3-a456-426614174000",
			tag:       "uuid",
			wantError: false,
		},
		{
			name:      "잘못된 UUID",
			value:     "invalid-uuid",
			tag:       "uuid",
			wantError: true,
		},
		{
			name:      "유효한 이메일",
			value:     "test@example.com",
			tag:       "email",
			wantError: false,
		},
		{
			name:      "잘못된 이메일",
			value:     "invalid-email",
			tag:       "email",
			wantError: true,
		},
		{
			name:      "범위 내 숫자",
			value:     50,
			tag:       "min=1,max=100",
			wantError: false,
		},
		{
			name:      "범위 초과 숫자",
			value:     150,
			tag:       "min=1,max=100",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.ValidateVar(tt.value, tt.tag)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCustomValidators(t *testing.T) {
	manager := NewValidationManager()

	// 테스트용 구조체
	type CustomTestStruct struct {
		WorkspaceStatus string `validate:"workspace_status"`
		ProjectStatus   string `validate:"project_status"`
		SessionStatus   string `validate:"session_status"`
		TaskStatus      string `validate:"task_status"`
		ClaudeAPIKey    string `validate:"claude_api_key"`
		NoSpecialChars  string `validate:"no_special_chars"`
	}

	tests := []struct {
		name      string
		input     CustomTestStruct
		wantError bool
	}{
		{
			name: "유효한 상태들",
			input: CustomTestStruct{
				WorkspaceStatus: "active",
				ProjectStatus:   "active",
				SessionStatus:   "active",
				TaskStatus:      "running",
				ClaudeAPIKey:    "sk-ant-api03-" + string(make([]byte, 50)),
				NoSpecialChars:  "ValidName123",
			},
			wantError: false,
		},
		{
			name: "유효하지 않은 워크스페이스 상태",
			input: CustomTestStruct{
				WorkspaceStatus: "invalid",
			},
			wantError: true,
		},
		{
			name: "유효하지 않은 프로젝트 상태",
			input: CustomTestStruct{
				ProjectStatus: "invalid",
			},
			wantError: true,
		},
		{
			name: "유효하지 않은 세션 상태",
			input: CustomTestStruct{
				SessionStatus: "invalid",
			},
			wantError: true,
		},
		{
			name: "유효하지 않은 태스크 상태",
			input: CustomTestStruct{
				TaskStatus: "invalid",
			},
			wantError: true,
		},
		{
			name: "유효하지 않은 Claude API 키",
			input: CustomTestStruct{
				ClaudeAPIKey: "invalid-key",
			},
			wantError: true,
		},
		{
			name: "특수문자 포함",
			input: CustomTestStruct{
				NoSpecialChars: "Invalid<Name>",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.Validate(tt.input)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateUUID(t *testing.T) {
	tests := []struct {
		name     string
		uuid     string
		expected bool
	}{
		{
			name:     "유효한 UUID v4",
			uuid:     "123e4567-e89b-12d3-a456-426614174000",
			expected: true,
		},
		{
			name:     "다른 유효한 UUID v4",
			uuid:     "f47ac10b-58cc-4372-a567-0e02b2c3d479",
			expected: true,
		},
		{
			name:     "빈 문자열",
			uuid:     "",
			expected: true, // empty는 허용 (required 태그에서 처리)
		},
		{
			name:     "잘못된 형식",
			uuid:     "invalid-uuid",
			expected: false,
		},
		{
			name:     "하이픈 누락",
			uuid:     "123e4567e89b12d3a456426614174000",
			expected: false,
		},
		{
			name:     "UUID v1 형식",
			uuid:     "123e4567-e89b-12d3-8456-426614174000",
			expected: false, // v4가 아니므로 false
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateUUID(&mockFieldLevel{value: tt.uuid})
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateNoSpecialChars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "유효한 문자열",
			input:    "ValidName123",
			expected: true,
		},
		{
			name:     "한글 포함",
			input:    "테스트이름",
			expected: true,
		},
		{
			name:     "공백 포함",
			input:    "Valid Name",
			expected: true,
		},
		{
			name:     "빈 문자열",
			input:    "",
			expected: true, // empty는 허용
		},
		{
			name:     "특수문자 < 포함",
			input:    "Invalid<Name",
			expected: false,
		},
		{
			name:     "특수문자 > 포함",
			input:    "Invalid>Name",
			expected: false,
		},
		{
			name:     "특수문자 & 포함",
			input:    "Invalid&Name",
			expected: false,
		},
		{
			name:     "제어 문자 포함",
			input:    "Invalid\x00Name",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateNoSpecialChars(&mockFieldLevel{value: tt.input})
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateClaudeAPIKey(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		expected bool
	}{
		{
			name:     "유효한 API 키",
			key:      "sk-ant-api03-" + string(make([]byte, 50)),
			expected: true,
		},
		{
			name:     "빈 문자열",
			key:      "",
			expected: true, // empty는 허용 (omitempty에서 처리)
		},
		{
			name:     "잘못된 접두사",
			key:      "sk-invalid-prefix" + string(make([]byte, 50)),
			expected: false,
		},
		{
			name:     "너무 짧은 키",
			key:      "sk-ant-api03-short",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateClaudeAPIKey(&mockFieldLevel{value: tt.key})
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Mock 구조체들
type mockFieldLevel struct {
	value interface{}
}

func (m *mockFieldLevel) Top() interface{} { return nil }
func (m *mockFieldLevel) Parent() interface{} { return nil }
func (m *mockFieldLevel) Field() reflect.Value { 
	if str, ok := m.value.(string); ok {
		return reflect.ValueOf(str)
	}
	return reflect.Value{}
}
func (m *mockFieldLevel) FieldName() string { return "field" }
func (m *mockFieldLevel) StructFieldName() string { return "Field" }
func (m *mockFieldLevel) Param() string { return "" }
func (m *mockFieldLevel) GetTag() string { return "" }
func (m *mockFieldLevel) ExtractType(field reflect.Value) (reflect.Value, reflect.Kind, bool) { 
	return reflect.Value{}, reflect.Invalid, false 
}
func (m *mockFieldLevel) GetStructFieldOK() (reflect.Value, reflect.Kind, bool) { return reflect.Value{}, reflect.Invalid, false }
func (m *mockFieldLevel) GetStructFieldOKAdvanced(val interface{}, namespace string) (interface{}, string, bool) { return nil, "", false }

func TestTranslateValidatorError(t *testing.T) {
	// 실제 validator를 사용한 에러 번역 테스트
	manager := NewValidationManager()

	type TestModel struct {
		Name  string `validate:"required,min=3,max=10"`
		Email string `validate:"required,email"`
	}

	// 에러를 발생시키는 데이터
	invalidModel := TestModel{
		Name:  "ab", // 너무 짧음
		Email: "invalid", // 잘못된 이메일
	}

	err := manager.Validate(invalidModel)
	require.Error(t, err)

	validationErrors, ok := err.(ValidationErrors)
	require.True(t, ok)
	
	// 에러가 번역되었는지 확인
	assert.Equal(t, "TestModel", validationErrors.Model)
	assert.True(t, len(validationErrors.Errors) > 0)
	
	// 각 에러가 적절한 한국어 메시지를 가지고 있는지 확인
	for _, fieldErr := range validationErrors.Errors {
		assert.NotEmpty(t, fieldErr.Message)
		assert.NotEmpty(t, fieldErr.Field)
		assert.NotEmpty(t, fieldErr.Tag)
	}
}