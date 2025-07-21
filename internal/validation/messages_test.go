package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultMessageTranslator_Translate(t *testing.T) {
	translator := NewDefaultMessageTranslator()

	tests := []struct {
		name     string
		key      MessageKey
		lang     Language
		params   []interface{}
		expected string
	}{
		{
			name:     "한국어 필수 필드 메시지",
			key:      MsgFieldRequired,
			lang:     LanguageKorean,
			params:   []interface{}{"이름"},
			expected: "이름 필드는 필수입니다",
		},
		{
			name:     "영어 필수 필드 메시지",
			key:      MsgFieldRequired,
			lang:     LanguageEnglish,
			params:   []interface{}{"name"},
			expected: "The name field is required",
		},
		{
			name:     "한국어 길이 제한 메시지",
			key:      MsgFieldTooLong,
			lang:     LanguageKorean,
			params:   []interface{}{"이름", "100"},
			expected: "이름 필드는 최대 100자 이하여야 합니다",
		},
		{
			name:     "영어 길이 제한 메시지",
			key:      MsgFieldTooLong,
			lang:     LanguageEnglish,
			params:   []interface{}{"name", "100"},
			expected: "The name field must not exceed 100 characters",
		},
		{
			name:     "파라미터 없는 메시지",
			key:      MsgInvalidClaudeAPIKey,
			lang:     LanguageKorean,
			params:   nil,
			expected: "Claude API 키 형식이 올바르지 않습니다 (sk-ant-api03-으로 시작해야 함)",
		},
		{
			name:     "존재하지 않는 키",
			key:      "nonexistent.key",
			lang:     LanguageKorean,
			params:   nil,
			expected: "Validation failed for key: nonexistent.key",
		},
		{
			name:     "지원하지 않는 언어 (fallback to Korean)",
			key:      MsgFieldRequired,
			lang:     "unsupported",
			params:   []interface{}{"이름"},
			expected: "이름 필드는 필수입니다",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := translator.Translate(tt.key, tt.lang, tt.params...)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetFieldDisplayName(t *testing.T) {
	tests := []struct {
		name      string
		fieldName string
		lang      Language
		expected  string
	}{
		{
			name:      "한국어 이름 필드",
			fieldName: "name",
			lang:      LanguageKorean,
			expected:  "이름",
		},
		{
			name:      "영어 이름 필드",
			fieldName: "name",
			lang:      LanguageEnglish,
			expected:  "name",
		},
		{
			name:      "한국어 프로젝트 경로",
			fieldName: "project_path",
			lang:      LanguageKorean,
			expected:  "프로젝트 경로",
		},
		{
			name:      "영어 프로젝트 경로",
			fieldName: "project_path",
			lang:      LanguageEnglish,
			expected:  "project path",
		},
		{
			name:      "존재하지 않는 필드명",
			fieldName: "unknown_field",
			lang:      LanguageKorean,
			expected:  "unknown_field", // 원본 반환
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetFieldDisplayName(tt.fieldName, tt.lang)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUpdateTranslatedFieldError(t *testing.T) {
	tests := []struct {
		name     string
		input    ValidationError
		lang     Language
		expected ValidationError
	}{
		{
			name: "required 에러 한국어 번역",
			input: ValidationError{
				Field: "name",
				Tag:   "required",
			},
			lang: LanguageKorean,
			expected: ValidationError{
				Field:   "이름",
				Tag:     "required",
				Message: "이름 필드는 필수입니다",
			},
		},
		{
			name: "min 에러 한국어 번역",
			input: ValidationError{
				Field: "name",
				Tag:   "min",
				Param: "3",
			},
			lang: LanguageKorean,
			expected: ValidationError{
				Field:   "이름",
				Tag:     "min",
				Param:   "3",
				Message: "이름 필드는 최소 3자 이상이어야 합니다",
			},
		},
		{
			name: "email 에러 영어 번역",
			input: ValidationError{
				Field: "email",
				Tag:   "email",
			},
			lang: LanguageEnglish,
			expected: ValidationError{
				Field:   "email",
				Tag:     "email",
				Message: "The email field must be a valid email address",
			},
		},
		{
			name: "workspace_status 에러 한국어 번역",
			input: ValidationError{
				Field: "status",
				Tag:   "workspace_status",
			},
			lang: LanguageKorean,
			expected: ValidationError{
				Field:   "상태",
				Tag:     "workspace_status",
				Message: "상태 필드는 유효한 워크스페이스 상태여야 합니다 (active, inactive, archived)",
			},
		},
		{
			name: "알 수 없는 태그",
			input: ValidationError{
				Field:   "name",
				Tag:     "unknown_tag",
				Message: "original message",
			},
			lang: LanguageKorean,
			expected: ValidationError{
				Field:   "이름",
				Tag:     "unknown_tag",
				Message: "original message", // 원본 메시지 유지
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := UpdateTranslatedFieldError(tt.input, tt.lang)
			assert.Equal(t, tt.expected.Field, result.Field)
			assert.Equal(t, tt.expected.Tag, result.Tag)
			assert.Equal(t, tt.expected.Message, result.Message)
			if tt.input.Param != "" {
				assert.Equal(t, tt.expected.Param, result.Param)
			}
		})
	}
}

func TestTranslateValidationErrors(t *testing.T) {
	input := ValidationErrors{
		Model: "TestModel",
		Errors: []ValidationError{
			{
				Field: "name",
				Tag:   "required",
			},
			{
				Field: "email",
				Tag:   "email",
			},
		},
	}

	result := TranslateValidationErrors(input, LanguageKorean)

	assert.Equal(t, "TestModel", result.Model)
	assert.Len(t, result.Errors, 2)

	// 첫 번째 에러 확인
	assert.Equal(t, "이름", result.Errors[0].Field)
	assert.Equal(t, "이름 필드는 필수입니다", result.Errors[0].Message)

	// 두 번째 에러 확인
	assert.Equal(t, "email", result.Errors[1].Field)
	assert.Equal(t, "email 필드는 유효한 이메일 주소여야 합니다", result.Errors[1].Message)
}

func TestGetLanguageFromContext(t *testing.T) {
	tests := []struct {
		name           string
		acceptLanguage string
		expected       Language
	}{
		{
			name:           "빈 Accept-Language",
			acceptLanguage: "",
			expected:       LanguageKorean,
		},
		{
			name:           "영어 선호",
			acceptLanguage: "en-US,en;q=0.9",
			expected:       LanguageEnglish,
		},
		{
			name:           "한국어 선호",
			acceptLanguage: "ko-KR,ko;q=0.9",
			expected:       LanguageKorean,
		},
		{
			name:           "영어 포함",
			acceptLanguage: "fr-FR,fr;q=0.9,en;q=0.8",
			expected:       LanguageEnglish,
		},
		{
			name:           "한국어만",
			acceptLanguage: "ko-KR",
			expected:       LanguageKorean,
		},
		{
			name:           "지원하지 않는 언어",
			acceptLanguage: "ja-JP,ja;q=0.9",
			expected:       LanguageKorean, // 기본값
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetLanguageFromContext(tt.acceptLanguage)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTranslateBusinessError(t *testing.T) {
	tests := []struct {
		name     string
		input    BusinessValidationError
		lang     Language
		expected BusinessValidationError
	}{
		{
			name: "중복 이름 에러 한국어 번역",
			input: BusinessValidationError{
				Code:  ErrCodeDuplicateName,
				Field: "name",
			},
			lang: LanguageKorean,
			expected: BusinessValidationError{
				Code:    ErrCodeDuplicateName,
				Field:   "이름",
				Message: "중복된 이름입니다",
			},
		},
		{
			name: "리소스 찾을 수 없음 에러 영어 번역",
			input: BusinessValidationError{
				Code: ErrCodeResourceNotFound,
			},
			lang: LanguageEnglish,
			expected: BusinessValidationError{
				Code:    ErrCodeResourceNotFound,
				Message: "Resource not found",
			},
		},
		{
			name: "권한 거부 에러 한국어 번역",
			input: BusinessValidationError{
				Code:  ErrCodePermissionDenied,
				Field: "project_path",
			},
			lang: LanguageKorean,
			expected: BusinessValidationError{
				Code:    ErrCodePermissionDenied,
				Field:   "프로젝트 경로",
				Message: "권한이 없습니다",
			},
		},
		{
			name: "번역되지 않은 에러 코드",
			input: BusinessValidationError{
				Code:    "UNKNOWN_ERROR",
				Message: "Original message",
				Field:   "name",
			},
			lang: LanguageKorean,
			expected: BusinessValidationError{
				Code:    "UNKNOWN_ERROR",
				Message: "Original message", // 원본 메시지 유지
				Field:   "이름",              // 필드명은 번역
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TranslateBusinessError(tt.input, tt.lang)
			assert.Equal(t, tt.expected.Code, result.Code)
			assert.Equal(t, tt.expected.Message, result.Message)
			assert.Equal(t, tt.expected.Field, result.Field)
		})
	}
}

func TestGlobalTranslatorFunctions(t *testing.T) {
	// 글로벌 번역기 초기화
	InitializeTranslator()

	t.Run("T 함수 테스트", func(t *testing.T) {
		result := T(MsgFieldRequired, "이름")
		assert.Equal(t, "이름 필드는 필수입니다", result)
	})

	t.Run("TL 함수 테스트", func(t *testing.T) {
		result := TL(MsgFieldRequired, LanguageEnglish, "name")
		assert.Equal(t, "The name field is required", result)
	})

	t.Run("언어 설정 테스트", func(t *testing.T) {
		GlobalTranslator.SetLanguage(LanguageEnglish)
		assert.Equal(t, LanguageEnglish, GlobalTranslator.GetLanguage())

		result := T(MsgFieldRequired, "name")
		assert.Equal(t, "The name field is required", result)

		// 원래 언어로 복원
		GlobalTranslator.SetLanguage(LanguageKorean)
	})
}

func TestMessageKeyConstant(t *testing.T) {
	// 메시지 키가 올바르게 정의되어 있는지 확인
	tests := []MessageKey{
		MsgFieldRequired,
		MsgFieldTooShort,
		MsgFieldTooLong,
		MsgFieldInvalidEmail,
		MsgFieldInvalidUUID,
		MsgPathNotExists,
		MsgPathNotDirectory,
		MsgInvalidWorkspaceStatus,
		MsgDuplicateName,
		MsgResourceNotFound,
		MsgInvalidClaudeAPIKey,
		MsgDangerousCommand,
	}

	translator := NewDefaultMessageTranslator()

	for _, key := range tests {
		t.Run(string(key), func(t *testing.T) {
			// 한국어 메시지 존재 확인
			koResult := translator.Translate(key, LanguageKorean)
			assert.NotEmpty(t, koResult)
			assert.NotContains(t, koResult, "Validation failed for key:")

			// 영어 메시지 존재 확인
			enResult := translator.Translate(key, LanguageEnglish)
			assert.NotEmpty(t, enResult)
			assert.NotContains(t, enResult, "Validation failed for key:")

			// 두 언어 메시지가 다른지 확인 (같으면 번역이 안된 것일 수 있음)
			if key != MsgInvalidClaudeAPIKey { // 일부 기술적 메시지는 비슷할 수 있음
				assert.NotEqual(t, koResult, enResult)
			}
		})
	}
}