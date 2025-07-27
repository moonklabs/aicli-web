package errors

import (
	"fmt"
	"sort"
	"strings"
)

// Color 상수들 - 터미널 색상 코드
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorGray   = "\033[37m"
	ColorBold   = "\033[1m"
)

// ErrorFormatter는 에러 메시지 포맷팅 인터페이스입니다.
type ErrorFormatter interface {
	// Format은 기본 에러 메시지를 포맷팅합니다.
	Format(err *CLIError) string

	// FormatWithDetails는 상세 정보를 포함하여 에러 메시지를 포맷팅합니다.
	FormatWithDetails(err *CLIError, verbose bool) string
}

// HumanErrorFormatter는 사용자 친화적인 에러 메시지 포맷터입니다.
type HumanErrorFormatter struct {
	colorEnabled bool
	showIcon     bool
}

// NewHumanErrorFormatter는 새로운 HumanErrorFormatter를 생성합니다.
func NewHumanErrorFormatter(colorEnabled, showIcon bool) *HumanErrorFormatter {
	return &HumanErrorFormatter{
		colorEnabled: colorEnabled,
		showIcon:     showIcon,
	}
}

// colorize는 색상이 활성화된 경우 텍스트에 색상을 적용합니다.
func (f *HumanErrorFormatter) colorize(text, color string) string {
	if !f.colorEnabled {
		return text
	}
	return color + text + ColorReset
}

// getErrorIcon은 에러 타입에 따른 아이콘을 반환합니다.
func (f *HumanErrorFormatter) getErrorIcon(errorType ErrorType) string {
	if !f.showIcon {
		return ""
	}

	switch errorType {
	case ErrorTypeValidation:
		return "⚠️  "
	case ErrorTypeConfig:
		return "⚙️  "
	case ErrorTypeNetwork:
		return "🌐 "
	case ErrorTypeFileSystem:
		return "📁 "
	case ErrorTypeProcess:
		return "⚡ "
	case ErrorTypeAuthentication:
		return "🔐 "
	case ErrorTypePermission:
		return "🚫 "
	case ErrorTypeNotFound:
		return "🔍 "
	case ErrorTypeConflict:
		return "💥 "
	case ErrorTypeInternal:
		return "🐛 "
	default:
		return "❌ "
	}
}

// Format은 기본 에러 메시지를 포맷팅합니다.
func (f *HumanErrorFormatter) Format(err *CLIError) string {
	var buf strings.Builder

	// 에러 아이콘과 메시지
	icon := f.getErrorIcon(err.Type)
	buf.WriteString(f.colorize(icon+"오류: ", ColorRed))
	buf.WriteString(err.Message)
	buf.WriteString("\n")

	// 제안사항
	if len(err.Suggestions) > 0 {
		buf.WriteString("\n")
		buf.WriteString(f.colorize("해결 방법:\n", ColorYellow))
		for _, suggestion := range err.Suggestions {
			buf.WriteString("  • ")
			buf.WriteString(suggestion)
			buf.WriteString("\n")
		}
	}

	// 원본 에러 메시지 (존재하는 경우)
	if err.Cause != nil {
		buf.WriteString("\n")
		buf.WriteString(f.colorize("상세 오류: ", ColorGray))
		buf.WriteString(err.Cause.Error())
		buf.WriteString("\n")
	}

	return buf.String()
}

// FormatWithDetails는 상세 정보를 포함하여 에러 메시지를 포맷팅합니다.
func (f *HumanErrorFormatter) FormatWithDetails(err *CLIError, verbose bool) string {
	var buf strings.Builder

	// 기본 포맷부터 시작
	buf.WriteString(f.Format(err))

	// verbose 모드가 아니면 기본 포맷만 반환
	if !verbose {
		return buf.String()
	}

	// 에러 분류 정보
	buf.WriteString("\n")
	buf.WriteString(f.colorize("진단 정보:\n", ColorCyan))
	buf.WriteString(fmt.Sprintf("  에러 타입: %s\n", err.Type.String()))
	buf.WriteString(fmt.Sprintf("  종료 코드: %d\n", err.ExitCode))

	// 맥락 정보
	if len(err.Context) > 0 {
		buf.WriteString("\n")
		buf.WriteString(f.colorize("맥락 정보:\n", ColorBlue))

		// 키를 정렬하여 일관된 출력
		keys := make([]string, 0, len(err.Context))
		for k := range err.Context {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, key := range keys {
			value := err.Context[key]
			buf.WriteString(fmt.Sprintf("  %s: %v\n", key, value))
		}
	}

	// 디버그 정보
	if len(err.Debug) > 0 {
		buf.WriteString("\n")
		buf.WriteString(f.colorize("디버그 정보:\n", ColorPurple))

		// 키를 정렬하여 일관된 출력
		keys := make([]string, 0, len(err.Debug))
		for k := range err.Debug {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, key := range keys {
			value := err.Debug[key]
			buf.WriteString(fmt.Sprintf("  %s: %v\n", key, value))
		}
	}

	// 추가 도움말
	buf.WriteString("\n")
	buf.WriteString(f.colorize("추가 도움말:\n", ColorGreen))
	buf.WriteString("  • 더 많은 로그를 보려면 --verbose 플래그를 사용하세요\n")
	buf.WriteString("  • 도움말을 보려면 'aicli help [command]'를 사용하세요\n")
	buf.WriteString("  • 문제가 지속되면 GitHub에 이슈를 생성하세요\n")

	return buf.String()
}

// JSONErrorFormatter는 JSON 형식으로 에러를 포맷팅합니다.
type JSONErrorFormatter struct{}

// NewJSONErrorFormatter는 새로운 JSONErrorFormatter를 생성합니다.
func NewJSONErrorFormatter() *JSONErrorFormatter {
	return &JSONErrorFormatter{}
}

// Format은 기본 JSON 에러 메시지를 포맷팅합니다.
func (f *JSONErrorFormatter) Format(err *CLIError) string {
	result := map[string]interface{}{
		"error": map[string]interface{}{
			"type":     err.Type.String(),
			"message":  err.Message,
			"exitCode": err.ExitCode,
		},
	}

	if len(err.Suggestions) > 0 {
		result["suggestions"] = err.Suggestions
	}

	if err.Cause != nil {
		result["cause"] = err.Cause.Error()
	}

	// 간단한 JSON 직렬화 (외부 라이브러리 없이)
	return f.toJSON(result)
}

// FormatWithDetails는 상세 정보를 포함한 JSON 에러 메시지를 포맷팅합니다.
func (f *JSONErrorFormatter) FormatWithDetails(err *CLIError, verbose bool) string {
	result := map[string]interface{}{
		"error": map[string]interface{}{
			"type":     err.Type.String(),
			"message":  err.Message,
			"exitCode": err.ExitCode,
		},
	}

	if len(err.Suggestions) > 0 {
		result["suggestions"] = err.Suggestions
	}

	if err.Cause != nil {
		result["cause"] = err.Cause.Error()
	}

	if verbose {
		if len(err.Context) > 0 {
			result["context"] = err.Context
		}

		if len(err.Debug) > 0 {
			result["debug"] = err.Debug
		}
	}

	return f.toJSON(result)
}

// toJSON은 간단한 JSON 직렬화를 수행합니다.
func (f *JSONErrorFormatter) toJSON(data interface{}) string {
	switch v := data.(type) {
	case map[string]interface{}:
		var parts []string
		for key, value := range v {
			parts = append(parts, fmt.Sprintf("\"%s\":%s", key, f.toJSON(value)))
		}
		return "{" + strings.Join(parts, ",") + "}"
	case []string:
		var parts []string
		for _, item := range v {
			parts = append(parts, fmt.Sprintf("\"%s\"", item))
		}
		return "[" + strings.Join(parts, ",") + "]"
	case string:
		return fmt.Sprintf("\"%s\"", strings.ReplaceAll(v, "\"", "\\\""))
	case int:
		return fmt.Sprintf("%d", v)
	default:
		return fmt.Sprintf("\"%v\"", v)
	}
}

// PlainErrorFormatter는 색상 없이 단순한 텍스트로 에러를 포맷팅합니다.
type PlainErrorFormatter struct{}

// NewPlainErrorFormatter는 새로운 PlainErrorFormatter를 생성합니다.
func NewPlainErrorFormatter() *PlainErrorFormatter {
	return &PlainErrorFormatter{}
}

// Format은 기본 플레인 텍스트 에러 메시지를 포맷팅합니다.
func (f *PlainErrorFormatter) Format(err *CLIError) string {
	var buf strings.Builder

	buf.WriteString("Error: ")
	buf.WriteString(err.Message)
	buf.WriteString("\n")

	if len(err.Suggestions) > 0 {
		buf.WriteString("\nSuggestions:\n")
		for _, suggestion := range err.Suggestions {
			buf.WriteString("  • ")
			buf.WriteString(suggestion)
			buf.WriteString("\n")
		}
	}

	if err.Cause != nil {
		buf.WriteString("\nCause: ")
		buf.WriteString(err.Cause.Error())
		buf.WriteString("\n")
	}

	return buf.String()
}

// FormatWithDetails는 상세 정보를 포함한 플레인 텍스트 에러 메시지를 포맷팅합니다.
func (f *PlainErrorFormatter) FormatWithDetails(err *CLIError, verbose bool) string {
	var buf strings.Builder

	buf.WriteString(f.Format(err))

	if verbose {
		buf.WriteString("\nDiagnostics:\n")
		buf.WriteString(fmt.Sprintf("  Error Type: %s\n", err.Type.String()))
		buf.WriteString(fmt.Sprintf("  Exit Code: %d\n", err.ExitCode))

		if len(err.Context) > 0 {
			buf.WriteString("\nContext:\n")
			keys := make([]string, 0, len(err.Context))
			for k := range err.Context {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			for _, key := range keys {
				buf.WriteString(fmt.Sprintf("  %s: %v\n", key, err.Context[key]))
			}
		}

		if len(err.Debug) > 0 {
			buf.WriteString("\nDebug:\n")
			keys := make([]string, 0, len(err.Debug))
			for k := range err.Debug {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			for _, key := range keys {
				buf.WriteString(fmt.Sprintf("  %s: %v\n", key, err.Debug[key]))
			}
		}
	}

	return buf.String()
}

// GetFormatter는 지정된 형식에 맞는 포맷터를 반환합니다.
func GetFormatter(format string, colorEnabled bool) ErrorFormatter {
	switch strings.ToLower(format) {
	case "json":
		return NewJSONErrorFormatter()
	case "plain":
		return NewPlainErrorFormatter()
	case "human", "":
		return NewHumanErrorFormatter(colorEnabled, true)
	default:
		return NewHumanErrorFormatter(colorEnabled, true)
	}
}
