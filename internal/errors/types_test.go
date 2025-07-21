package errors

import (
	"errors"
	"testing"
)

func TestErrorType_String(t *testing.T) {
	tests := []struct {
		errorType ErrorType
		expected  string
	}{
		{ErrorTypeValidation, "ValidationError"},
		{ErrorTypeConfig, "ConfigError"},
		{ErrorTypeNetwork, "NetworkError"},
		{ErrorTypeFileSystem, "FileSystemError"},
		{ErrorTypeProcess, "ProcessError"},
		{ErrorTypeAuthentication, "AuthenticationError"},
		{ErrorTypePermission, "PermissionError"},
		{ErrorTypeNotFound, "NotFoundError"},
		{ErrorTypeConflict, "ConflictError"},
		{ErrorTypeInternal, "InternalError"},
		{ErrorTypeUnknown, "UnknownError"},
	}

	for _, test := range tests {
		t.Run(test.expected, func(t *testing.T) {
			result := test.errorType.String()
			if result != test.expected {
				t.Errorf("ErrorType.String() = %v, want %v", result, test.expected)
			}
		})
	}
}

func TestErrorType_ExitCode(t *testing.T) {
	tests := []struct {
		errorType ErrorType
		expected  int
	}{
		{ErrorTypeValidation, 1},
		{ErrorTypeConfig, 2},
		{ErrorTypeNetwork, 3},
		{ErrorTypeFileSystem, 4},
		{ErrorTypePermission, 5},
		{ErrorTypeAuthentication, 6},
		{ErrorTypeProcess, 7},
		{ErrorTypeNotFound, 8},
		{ErrorTypeConflict, 9},
		{ErrorTypeInternal, 127},
		{ErrorTypeUnknown, 1},
	}

	for _, test := range tests {
		t.Run(test.errorType.String(), func(t *testing.T) {
			result := test.errorType.ExitCode()
			if result != test.expected {
				t.Errorf("ErrorType.ExitCode() = %v, want %v", result, test.expected)
			}
		})
	}
}

func TestNewCLIError(t *testing.T) {
	message := "테스트 에러 메시지"
	err := NewCLIError(ErrorTypeValidation, message)

	if err.Type != ErrorTypeValidation {
		t.Errorf("CLIError.Type = %v, want %v", err.Type, ErrorTypeValidation)
	}

	if err.Message != message {
		t.Errorf("CLIError.Message = %v, want %v", err.Message, message)
	}

	if err.ExitCode != ErrorTypeValidation.ExitCode() {
		t.Errorf("CLIError.ExitCode = %v, want %v", err.ExitCode, ErrorTypeValidation.ExitCode())
	}
}

func TestCLIError_Error(t *testing.T) {
	message := "테스트 에러 메시지"
	err := NewCLIError(ErrorTypeValidation, message)

	if err.Error() != message {
		t.Errorf("CLIError.Error() = %v, want %v", err.Error(), message)
	}
}

func TestCLIError_Unwrap(t *testing.T) {
	originalErr := errors.New("원본 에러")
	err := NewCLIError(ErrorTypeInternal, "래핑된 에러").WithCause(originalErr)

	unwrapped := err.Unwrap()
	if unwrapped != originalErr {
		t.Errorf("CLIError.Unwrap() = %v, want %v", unwrapped, originalErr)
	}
}

func TestCLIError_AddSuggestion(t *testing.T) {
	err := NewCLIError(ErrorTypeValidation, "테스트 에러")
	suggestion := "이렇게 해보세요"

	err.AddSuggestion(suggestion)

	if len(err.Suggestions) != 1 {
		t.Errorf("len(CLIError.Suggestions) = %v, want %v", len(err.Suggestions), 1)
	}

	if err.Suggestions[0] != suggestion {
		t.Errorf("CLIError.Suggestions[0] = %v, want %v", err.Suggestions[0], suggestion)
	}
}

func TestCLIError_AddContext(t *testing.T) {
	err := NewCLIError(ErrorTypeValidation, "테스트 에러")
	key := "test_key"
	value := "test_value"

	err.AddContext(key, value)

	if err.Context[key] != value {
		t.Errorf("CLIError.Context[%v] = %v, want %v", key, err.Context[key], value)
	}
}

func TestCLIError_AddDebug(t *testing.T) {
	err := NewCLIError(ErrorTypeValidation, "테스트 에러")
	key := "debug_key"
	value := "debug_value"

	err.AddDebug(key, value)

	if err.Debug[key] != value {
		t.Errorf("CLIError.Debug[%v] = %v, want %v", key, err.Debug[key], value)
	}
}

func TestNewValidationError(t *testing.T) {
	message := "검증 실패"
	suggestions := []string{"첫 번째 제안", "두 번째 제안"}

	err := NewValidationError(message, suggestions...)

	if err.Type != ErrorTypeValidation {
		t.Errorf("ValidationError.Type = %v, want %v", err.Type, ErrorTypeValidation)
	}

	if err.Message != message {
		t.Errorf("ValidationError.Message = %v, want %v", err.Message, message)
	}

	if len(err.Suggestions) != len(suggestions) {
		t.Errorf("len(ValidationError.Suggestions) = %v, want %v", len(err.Suggestions), len(suggestions))
	}

	for i, suggestion := range suggestions {
		if err.Suggestions[i] != suggestion {
			t.Errorf("ValidationError.Suggestions[%d] = %v, want %v", i, err.Suggestions[i], suggestion)
		}
	}
}

func TestNewRequiredFlagError(t *testing.T) {
	flagName := "test-flag"
	description := "테스트 플래그"

	err := NewRequiredFlagError(flagName, description)

	if err.Type != ErrorTypeValidation {
		t.Errorf("RequiredFlagError.Type = %v, want %v", err.Type, ErrorTypeValidation)
	}

	if err.Context["flag_name"] != flagName {
		t.Errorf("RequiredFlagError.Context[\"flag_name\"] = %v, want %v", err.Context["flag_name"], flagName)
	}

	if err.Context["description"] != description {
		t.Errorf("RequiredFlagError.Context[\"description\"] = %v, want %v", err.Context["description"], description)
	}

	if len(err.Suggestions) == 0 {
		t.Error("RequiredFlagError should have suggestions")
	}
}

func TestNewInvalidValueError(t *testing.T) {
	field := "format"
	value := "invalid"
	validValues := []string{"json", "yaml", "table"}

	err := NewInvalidValueError(field, value, validValues)

	if err.Type != ErrorTypeValidation {
		t.Errorf("InvalidValueError.Type = %v, want %v", err.Type, ErrorTypeValidation)
	}

	if err.Context["field"] != field {
		t.Errorf("InvalidValueError.Context[\"field\"] = %v, want %v", err.Context["field"], field)
	}

	if err.Context["invalid_value"] != value {
		t.Errorf("InvalidValueError.Context[\"invalid_value\"] = %v, want %v", err.Context["invalid_value"], value)
	}
}

func TestNewConfigError(t *testing.T) {
	originalErr := errors.New("설정 파일을 찾을 수 없습니다")
	configPath := "/path/to/config.yaml"

	err := NewConfigError(originalErr, configPath)

	if err.Type != ErrorTypeConfig {
		t.Errorf("ConfigError.Type = %v, want %v", err.Type, ErrorTypeConfig)
	}

	if err.Cause != originalErr {
		t.Errorf("ConfigError.Cause = %v, want %v", err.Cause, originalErr)
	}

	if err.Context["config_path"] != configPath {
		t.Errorf("ConfigError.Context[\"config_path\"] = %v, want %v", err.Context["config_path"], configPath)
	}
}

func TestNewNetworkError(t *testing.T) {
	originalErr := errors.New("connection refused")
	service := "Claude API"

	err := NewNetworkError(service, originalErr)

	if err.Type != ErrorTypeNetwork {
		t.Errorf("NetworkError.Type = %v, want %v", err.Type, ErrorTypeNetwork)
	}

	if err.Cause != originalErr {
		t.Errorf("NetworkError.Cause = %v, want %v", err.Cause, originalErr)
	}

	if err.Context["service"] != service {
		t.Errorf("NetworkError.Context[\"service\"] = %v, want %v", err.Context["service"], service)
	}
}

func TestIsType(t *testing.T) {
	validationErr := NewValidationError("검증 에러")
	configErr := NewConfigError(errors.New("설정 에러"), "/path/to/config")
	standardErr := errors.New("표준 에러")

	tests := []struct {
		name      string
		err       error
		errorType ErrorType
		expected  bool
	}{
		{"ValidationError matches", validationErr, ErrorTypeValidation, true},
		{"ValidationError doesn't match Config", validationErr, ErrorTypeConfig, false},
		{"ConfigError matches", configErr, ErrorTypeConfig, true},
		{"ConfigError doesn't match Validation", configErr, ErrorTypeValidation, false},
		{"Standard error doesn't match any", standardErr, ErrorTypeValidation, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := IsType(test.err, test.errorType)
			if result != test.expected {
				t.Errorf("IsType(%v, %v) = %v, want %v", test.err, test.errorType, result, test.expected)
			}
		})
	}
}

func TestGetExitCode(t *testing.T) {
	validationErr := NewValidationError("검증 에러")
	configErr := NewConfigError(errors.New("설정 에러"), "/path/to/config")
	standardErr := errors.New("표준 에러")

	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{"ValidationError exit code", validationErr, 1},
		{"ConfigError exit code", configErr, 2},
		{"Standard error default exit code", standardErr, 1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := GetExitCode(test.err)
			if result != test.expected {
				t.Errorf("GetExitCode(%v) = %v, want %v", test.err, result, test.expected)
			}
		})
	}
}