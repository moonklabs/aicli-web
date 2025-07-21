package errors

import (
	"fmt"
	"strings"
	"testing"
)

func TestHumanErrorFormatter_Format(t *testing.T) {
	formatter := NewHumanErrorFormatter(false, false) // 색상과 아이콘 비활성화
	
	err := NewValidationError("테스트 에러 메시지")
	err.AddSuggestion("첫 번째 제안")
	err.AddSuggestion("두 번째 제안")
	
	result := formatter.Format(err)
	
	// 기본 검증
	if !strings.Contains(result, "오류: 테스트 에러 메시지") {
		t.Errorf("포맷된 메시지에 에러 메시지가 포함되지 않음: %s", result)
	}
	
	if !strings.Contains(result, "해결 방법:") {
		t.Errorf("포맷된 메시지에 해결 방법 섹션이 포함되지 않음: %s", result)
	}
	
	if !strings.Contains(result, "첫 번째 제안") {
		t.Errorf("포맷된 메시지에 첫 번째 제안이 포함되지 않음: %s", result)
	}
	
	if !strings.Contains(result, "두 번째 제안") {
		t.Errorf("포맷된 메시지에 두 번째 제안이 포함되지 않음: %s", result)
	}
}

func TestHumanErrorFormatter_FormatWithDetails(t *testing.T) {
	formatter := NewHumanErrorFormatter(false, false)
	
	err := NewValidationError("테스트 에러 메시지")
	err.AddSuggestion("제안사항")
	err.AddContext("field", "test_field")
	err.AddContext("value", "test_value")
	err.AddDebug("debug_info", "debug_value")
	
	// Verbose 모드
	resultVerbose := formatter.FormatWithDetails(err, true)
	
	if !strings.Contains(resultVerbose, "진단 정보:") {
		t.Errorf("Verbose 모드에서 진단 정보가 포함되지 않음: %s", resultVerbose)
	}
	
	if !strings.Contains(resultVerbose, "맥락 정보:") {
		t.Errorf("Verbose 모드에서 맥락 정보가 포함되지 않음: %s", resultVerbose)
	}
	
	if !strings.Contains(resultVerbose, "디버그 정보:") {
		t.Errorf("Verbose 모드에서 디버그 정보가 포함되지 않음: %s", resultVerbose)
	}
	
	if !strings.Contains(resultVerbose, "test_field") {
		t.Errorf("Verbose 모드에서 맥락 정보 내용이 포함되지 않음: %s", resultVerbose)
	}
	
	// Non-verbose 모드
	resultNonVerbose := formatter.FormatWithDetails(err, false)
	
	if strings.Contains(resultNonVerbose, "진단 정보:") {
		t.Errorf("Non-verbose 모드에서 진단 정보가 포함됨: %s", resultNonVerbose)
	}
}

func TestHumanErrorFormatter_ColorEnabled(t *testing.T) {
	formatter := NewHumanErrorFormatter(true, false) // 색상 활성화
	
	err := NewValidationError("테스트 에러 메시지")
	result := formatter.Format(err)
	
	// ANSI 색상 코드가 포함되어야 함
	if !strings.Contains(result, "\033[") {
		t.Errorf("색상 활성화 시 ANSI 색상 코드가 포함되지 않음: %s", result)
	}
}

func TestHumanErrorFormatter_IconEnabled(t *testing.T) {
	formatter := NewHumanErrorFormatter(false, true) // 아이콘 활성화
	
	err := NewValidationError("테스트 에러 메시지")
	result := formatter.Format(err)
	
	// 아이콘이 포함되어야 함 (ValidationError는 ⚠️)
	if !strings.Contains(result, "⚠️") {
		t.Errorf("아이콘 활성화 시 ValidationError 아이콘이 포함되지 않음: %s", result)
	}
}

func TestJSONErrorFormatter_Format(t *testing.T) {
	formatter := NewJSONErrorFormatter()
	
	err := NewValidationError("테스트 에러 메시지")
	err.AddSuggestion("제안사항")
	
	result := formatter.Format(err)
	
	// JSON 형식 기본 검증
	if !strings.Contains(result, `"error"`) {
		t.Errorf("JSON 포맷에 error 필드가 포함되지 않음: %s", result)
	}
	
	if !strings.Contains(result, `"type":"ValidationError"`) {
		t.Errorf("JSON 포맷에 type 필드가 포함되지 않음: %s", result)
	}
	
	if !strings.Contains(result, `"message":"테스트 에러 메시지"`) {
		t.Errorf("JSON 포맷에 message 필드가 포함되지 않음: %s", result)
	}
	
	if !strings.Contains(result, `"suggestions"`) {
		t.Errorf("JSON 포맷에 suggestions 필드가 포함되지 않음: %s", result)
	}
}

func TestJSONErrorFormatter_FormatWithDetails(t *testing.T) {
	formatter := NewJSONErrorFormatter()
	
	err := NewValidationError("테스트 에러 메시지")
	err.AddContext("field", "test_field")
	err.AddDebug("debug_info", "debug_value")
	
	// Verbose 모드
	resultVerbose := formatter.FormatWithDetails(err, true)
	
	if !strings.Contains(resultVerbose, `"context"`) {
		t.Errorf("JSON Verbose 모드에서 context 필드가 포함되지 않음: %s", resultVerbose)
	}
	
	if !strings.Contains(resultVerbose, `"debug"`) {
		t.Errorf("JSON Verbose 모드에서 debug 필드가 포함되지 않음: %s", resultVerbose)
	}
	
	// Non-verbose 모드
	resultNonVerbose := formatter.FormatWithDetails(err, false)
	
	if strings.Contains(resultNonVerbose, `"context"`) {
		t.Errorf("JSON Non-verbose 모드에서 context 필드가 포함됨: %s", resultNonVerbose)
	}
	
	if strings.Contains(resultNonVerbose, `"debug"`) {
		t.Errorf("JSON Non-verbose 모드에서 debug 필드가 포함됨: %s", resultNonVerbose)
	}
}

func TestPlainErrorFormatter_Format(t *testing.T) {
	formatter := NewPlainErrorFormatter()
	
	err := NewValidationError("테스트 에러 메시지")
	err.AddSuggestion("제안사항")
	
	result := formatter.Format(err)
	
	if !strings.Contains(result, "Error: 테스트 에러 메시지") {
		t.Errorf("플레인 포맷에 에러 메시지가 포함되지 않음: %s", result)
	}
	
	if !strings.Contains(result, "Suggestions:") {
		t.Errorf("플레인 포맷에 제안사항 섹션이 포함되지 않음: %s", result)
	}
	
	if !strings.Contains(result, "제안사항") {
		t.Errorf("플레인 포맷에 제안사항 내용이 포함되지 않음: %s", result)
	}
	
	// 색상 코드가 포함되지 않아야 함
	if strings.Contains(result, "\033[") {
		t.Errorf("플레인 포맷에 ANSI 색상 코드가 포함됨: %s", result)
	}
}

func TestGetFormatter(t *testing.T) {
	tests := []struct {
		format   string
		expected string
	}{
		{"json", "*errors.JSONErrorFormatter"},
		{"JSON", "*errors.JSONErrorFormatter"},
		{"plain", "*errors.PlainErrorFormatter"},
		{"human", "*errors.HumanErrorFormatter"},
		{"", "*errors.HumanErrorFormatter"},
		{"unknown", "*errors.HumanErrorFormatter"},
	}
	
	for _, test := range tests {
		t.Run(test.format, func(t *testing.T) {
			formatter := GetFormatter(test.format, false)
			
			// 타입 이름을 문자열로 확인
			typeName := strings.Split(strings.Split(fmt.Sprintf("%T", formatter), ".")[1], "{")[0]
			expectedType := strings.Split(test.expected, ".")[1]
			
			if typeName != expectedType {
				t.Errorf("GetFormatter(%q) returned %T, want %s", test.format, formatter, test.expected)
			}
		})
	}
}

func TestErrorFormatterWithCause(t *testing.T) {
	formatter := NewHumanErrorFormatter(false, false)
	
	originalErr := NewInternalError("원본 에러", nil)
	err := NewValidationError("래핑된 에러").WithCause(originalErr)
	
	result := formatter.Format(err)
	
	if !strings.Contains(result, "상세 오류:") {
		t.Errorf("원본 에러가 있을 때 상세 오류 섹션이 포함되지 않음: %s", result)
	}
	
	if !strings.Contains(result, "원본 에러") {
		t.Errorf("원본 에러 메시지가 포함되지 않음: %s", result)
	}
}

