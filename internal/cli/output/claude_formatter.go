package output

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aicli/aicli-web/internal/claude"
	"github.com/fatih/color"
)

// ClaudeFormatter는 Claude 메시지 출력을 위한 인터페이스입니다.
type ClaudeFormatter interface {
	FormatMessage(msg *claude.Message) string
	FormatError(err error) string
	FormatComplete(summary *claude.ExecutionSummary) string
	FormatProgress(progress *claude.ProgressInfo) string
}

// TextFormatter는 텍스트 형식의 Claude 포맷터입니다.
type TextFormatter struct {
	useColor     bool
	showMetadata bool
	showTime     bool
}

// JSONFormatter는 JSON 형식의 Claude 포맷터입니다.
type ClaudeJSONFormatter struct {
	pretty bool
}

// MarkdownFormatter는 Markdown 형식의 Claude 포맷터입니다.
type MarkdownFormatter struct {
	syntaxHighlight bool
}

// NewTextFormatter는 새로운 TextFormatter를 생성합니다.
func NewTextFormatter(useColor, showMetadata bool) *TextFormatter {
	return &TextFormatter{
		useColor:     useColor && detectColorSupport(),
		showMetadata: showMetadata,
		showTime:     true,
	}
}

// NewJSONFormatter는 새로운 ClaudeJSONFormatter를 생성합니다.
func NewJSONFormatter(pretty bool) *ClaudeJSONFormatter {
	return &ClaudeJSONFormatter{
		pretty: pretty,
	}
}

// NewMarkdownFormatter는 새로운 MarkdownFormatter를 생성합니다.
func NewMarkdownFormatter(syntaxHighlight bool) *MarkdownFormatter {
	return &MarkdownFormatter{
		syntaxHighlight: syntaxHighlight,
	}
}

// FormatMessage는 Claude 메시지를 텍스트 형식으로 포맷합니다.
func (tf *TextFormatter) FormatMessage(msg *claude.Message) string {
	if msg == nil {
		return ""
	}

	var result strings.Builder

	// 타임스탬프 (옵션)
	if tf.showTime {
		timestamp := time.Now().Format("15:04:05")
		if tf.useColor {
			result.WriteString(color.HiBlackString("[%s] ", timestamp))
		} else {
			result.WriteString(fmt.Sprintf("[%s] ", timestamp))
		}
	}

	// 메시지 타입에 따른 포맷팅
	switch msg.Type {
	case "text":
		result.WriteString(tf.formatTextMessageFromMsg(msg))
	case "tool_use":
		result.WriteString(tf.formatToolUseMessageFromMsg(msg))
	case "tool_result":
		result.WriteString(tf.formatToolResultMessageFromMsg(msg))
	case "error":
		result.WriteString(tf.formatErrorMessageFromMsg(msg))
	case "thinking":
		result.WriteString(tf.formatThinkingMessageFromMsg(msg))
	case "system":
		result.WriteString(tf.formatSystemMessageFromMsg(msg))
	default:
		result.WriteString(tf.formatGenericMessageFromMsg(msg))
	}

	// 메타데이터 (옵션)
	if tf.showMetadata && len(msg.Meta) > 0 {
		result.WriteString(tf.formatMetadata(msg.Meta))
	}

	return result.String()
}

// formatTextMessage는 텍스트 메시지를 포맷합니다.
func (tf *TextFormatter) formatTextMessage(msg *claude.FormattedMessage) string {
	content := msg.Content

	// 코드 블록 하이라이팅
	if tf.useColor {
		content = tf.highlightCodeBlocks(content)
	}

	return content
}

// formatTextMessageFromMsg는 Message 타입에서 텍스트 메시지를 포맷합니다.
func (tf *TextFormatter) formatTextMessageFromMsg(msg *claude.Message) string {
	content := msg.Content

	// 코드 블록 하이라이팅
	if tf.useColor {
		content = tf.highlightCodeBlocks(content)
	}

	return content
}

// formatToolUseMessage는 도구 사용 메시지를 포맷합니다.
func (tf *TextFormatter) formatToolUseMessage(msg *claude.FormattedMessage) string {
	var result strings.Builder

	toolName := "unknown"
	if name, ok := msg.Metadata["tool_name"].(string); ok {
		toolName = name
	}

	if tf.useColor {
		result.WriteString(color.YellowString("🔧 도구 사용: %s", toolName))
	} else {
		result.WriteString(fmt.Sprintf("🔧 도구 사용: %s", toolName))
	}

	// 도구 입력 표시 (간략화)
	if msg.Content != "" && len(msg.Content) < 200 {
		result.WriteString("\n")
		if tf.useColor {
			result.WriteString(color.HiBlackString("  입력: %s", msg.Content))
		} else {
			result.WriteString(fmt.Sprintf("  입력: %s", msg.Content))
		}
	}

	return result.String()
}

// formatToolUseMessageFromMsg는 Message 타입에서 도구 사용 메시지를 포맷합니다.
func (tf *TextFormatter) formatToolUseMessageFromMsg(msg *claude.Message) string {
	var result strings.Builder

	toolName := "unknown"
	if name, ok := msg.Meta["tool_name"].(string); ok {
		toolName = name
	}

	if tf.useColor {
		result.WriteString(color.YellowString("🔧 도구 사용: %s", toolName))
	} else {
		result.WriteString(fmt.Sprintf("🔧 도구 사용: %s", toolName))
	}

	// 도구 입력 표시 (간략화)
	if msg.Content != "" && len(msg.Content) < 200 {
		result.WriteString("\n")
		if tf.useColor {
			result.WriteString(color.HiBlackString("  입력: %s", msg.Content))
		} else {
			result.WriteString(fmt.Sprintf("  입력: %s", msg.Content))
		}
	}

	return result.String()
}

// formatToolResultMessage는 도구 결과 메시지를 포맷합니다.
func (tf *TextFormatter) formatToolResultMessage(msg *claude.FormattedMessage) string {
	var result strings.Builder

	success := true
	if s, ok := msg.Metadata["success"].(bool); ok {
		success = s
	}

	if success {
		if tf.useColor {
			result.WriteString(color.GreenString("✅ 도구 실행 완료"))
		} else {
			result.WriteString("✅ 도구 실행 완료")
		}
	} else {
		if tf.useColor {
			result.WriteString(color.RedString("❌ 도구 실행 실패"))
		} else {
			result.WriteString("❌ 도구 실행 실패")
		}
	}

	// 결과 내용 표시 (간략화)
	if msg.Content != "" {
		lines := strings.Split(msg.Content, "\n")
		if len(lines) > 5 {
			// 처음 3줄과 마지막 1줄만 표시
			content := strings.Join(lines[:3], "\n") + "\n... (" +
				fmt.Sprintf("%d lines omitted", len(lines)-4) + ")\n" + lines[len(lines)-1]
			result.WriteString("\n")
			if tf.useColor {
				result.WriteString(color.HiBlackString("  결과: %s", content))
			} else {
				result.WriteString(fmt.Sprintf("  결과: %s", content))
			}
		} else {
			result.WriteString("\n")
			if tf.useColor {
				result.WriteString(color.HiBlackString("  결과: %s", msg.Content))
			} else {
				result.WriteString(fmt.Sprintf("  결과: %s", msg.Content))
			}
		}
	}

	return result.String()
}

// formatToolResultMessageFromMsg는 Message 타입에서 도구 결과 메시지를 포맷합니다.
func (tf *TextFormatter) formatToolResultMessageFromMsg(msg *claude.Message) string {
	var result strings.Builder

	success := true
	if s, ok := msg.Meta["success"].(bool); ok {
		success = s
	}

	if success {
		if tf.useColor {
			result.WriteString(color.GreenString("✅ 도구 실행 완료"))
		} else {
			result.WriteString("✅ 도구 실행 완료")
		}
	} else {
		if tf.useColor {
			result.WriteString(color.RedString("❌ 도구 실행 실패"))
		} else {
			result.WriteString("❌ 도구 실행 실패")
		}
	}

	// 결과 내용 표시 (간략화)
	if msg.Content != "" {
		lines := strings.Split(msg.Content, "\n")
		if len(lines) > 5 {
			// 처음 3줄과 마지막 1줄만 표시
			content := strings.Join(lines[:3], "\n") + "\n... (" +
				fmt.Sprintf("%d lines omitted", len(lines)-4) + ")\n" + lines[len(lines)-1]
			result.WriteString("\n")
			if tf.useColor {
				result.WriteString(color.HiBlackString("  결과: %s", content))
			} else {
				result.WriteString(fmt.Sprintf("  결과: %s", content))
			}
		} else {
			result.WriteString("\n")
			if tf.useColor {
				result.WriteString(color.HiBlackString("  결과: %s", msg.Content))
			} else {
				result.WriteString(fmt.Sprintf("  결과: %s", msg.Content))
			}
		}
	}

	return result.String()
}

// formatErrorMessage는 에러 메시지를 포맷합니다.
func (tf *TextFormatter) formatErrorMessage(msg *claude.FormattedMessage) string {
	if tf.useColor {
		return color.RedString("❌ 오류: %s", msg.Content)
	}
	return fmt.Sprintf("❌ 오류: %s", msg.Content)
}

// formatErrorMessageFromMsg는 Message 타입에서 에러 메시지를 포맷합니다.
func (tf *TextFormatter) formatErrorMessageFromMsg(msg *claude.Message) string {
	if tf.useColor {
		return color.RedString("❌ 오류: %s", msg.Content)
	}
	return fmt.Sprintf("❌ 오류: %s", msg.Content)
}

// formatThinkingMessage는 thinking 메시지를 포맷합니다.
func (tf *TextFormatter) formatThinkingMessage(msg *claude.FormattedMessage) string {
	// thinking 메시지는 보통 내부적이므로 간략하게 표시
	if tf.useColor {
		return color.MagentaString("🤔 생각 중...")
	}
	return "🤔 생각 중..."
}

// formatThinkingMessageFromMsg는 Message 타입에서 thinking 메시지를 포맷합니다.
func (tf *TextFormatter) formatThinkingMessageFromMsg(msg *claude.Message) string {
	// thinking 메시지는 보통 내부적이므로 간략하게 표시
	if tf.useColor {
		return color.MagentaString("🤔 생각 중...")
	}
	return "🤔 생각 중..."
}

// formatSystemMessage는 시스템 메시지를 포맷합니다.
func (tf *TextFormatter) formatSystemMessage(msg *claude.FormattedMessage) string {
	if tf.useColor {
		return color.CyanString("ℹ️  %s", msg.Content)
	}
	return fmt.Sprintf("ℹ️  %s", msg.Content)
}

// formatSystemMessageFromMsg는 Message 타입에서 시스템 메시지를 포맷합니다.
func (tf *TextFormatter) formatSystemMessageFromMsg(msg *claude.Message) string {
	if tf.useColor {
		return color.CyanString("ℹ️  %s", msg.Content)
	}
	return fmt.Sprintf("ℹ️  %s", msg.Content)
}

// formatGenericMessage는 일반적인 메시지를 포맷합니다.
func (tf *TextFormatter) formatGenericMessage(msg *claude.FormattedMessage) string {
	return fmt.Sprintf("[%s] %s", msg.Type, msg.Content)
}

// formatGenericMessageFromMsg는 Message 타입에서 일반적인 메시지를 포맷합니다.
func (tf *TextFormatter) formatGenericMessageFromMsg(msg *claude.Message) string {
	return fmt.Sprintf("[%s] %s", msg.Type, msg.Content)
}

// formatMetadata는 메타데이터를 포맷합니다.
func (tf *TextFormatter) formatMetadata(metadata map[string]interface{}) string {
	if len(metadata) == 0 {
		return ""
	}

	var result strings.Builder
	result.WriteString("\n")

	if tf.useColor {
		result.WriteString(color.HiBlackString("  메타데이터: "))
	} else {
		result.WriteString("  메타데이터: ")
	}

	first := true
	for key, value := range metadata {
		if !first {
			result.WriteString(", ")
		}
		result.WriteString(fmt.Sprintf("%s=%v", key, value))
		first = false
	}

	return result.String()
}

// highlightCodeBlocks는 코드 블록에 하이라이팅을 적용합니다.
func (tf *TextFormatter) highlightCodeBlocks(content string) string {
	// 간단한 코드 블록 감지 및 하이라이팅
	lines := strings.Split(content, "\n")
	var result strings.Builder
	inCodeBlock := false

	for _, line := range lines {
		if strings.HasPrefix(line, "```") {
			inCodeBlock = !inCodeBlock
			if tf.useColor {
				result.WriteString(color.HiBlackString(line))
			} else {
				result.WriteString(line)
			}
		} else if inCodeBlock {
			if tf.useColor {
				result.WriteString(color.HiYellowString(line))
			} else {
				result.WriteString(line)
			}
		} else {
			result.WriteString(line)
		}
		result.WriteString("\n")
	}

	return strings.TrimSuffix(result.String(), "\n")
}

// FormatError는 에러를 텍스트 형식으로 포맷합니다.
func (tf *TextFormatter) FormatError(err error) string {
	if err == nil {
		return ""
	}

	if tf.useColor {
		return color.RedString("❌ 오류: %s", err.Error())
	}
	return fmt.Sprintf("❌ 오류: %s", err.Error())
}

// FormatComplete는 실행 완료 요약을 텍스트 형식으로 포맷합니다.
func (tf *TextFormatter) FormatComplete(summary *claude.ExecutionSummary) string {
	if summary == nil {
		return ""
	}

	var result strings.Builder

	if summary.Success {
		if tf.useColor {
			result.WriteString(color.GreenString("✅ 실행 완료"))
		} else {
			result.WriteString("✅ 실행 완료")
		}
	} else {
		if tf.useColor {
			result.WriteString(color.RedString("❌ 실행 실패"))
		} else {
			result.WriteString("❌ 실행 실패")
		}
	}

	// 실행 시간 표시
	if summary.Duration > 0 {
		duration := time.Duration(summary.Duration) * time.Millisecond
		if tf.useColor {
			result.WriteString(color.HiBlackString(" (%s)", duration.String()))
		} else {
			result.WriteString(fmt.Sprintf(" (%s)", duration.String()))
		}
	}

	// 토큰 사용량 표시
	if summary.InputTokens > 0 || summary.OutputTokens > 0 {
		result.WriteString("\n")
		if tf.useColor {
			result.WriteString(color.HiBlackString("  토큰 사용량: 입력 %d, 출력 %d",
				summary.InputTokens, summary.OutputTokens))
		} else {
			result.WriteString(fmt.Sprintf("  토큰 사용량: 입력 %d, 출력 %d",
				summary.InputTokens, summary.OutputTokens))
		}
	}

	return result.String()
}

// FormatProgress는 진행 상황을 텍스트 형식으로 포맷합니다.
func (tf *TextFormatter) FormatProgress(progress *claude.ProgressInfo) string {
	if progress == nil {
		return ""
	}

	var result strings.Builder

	// 진행률 표시
	if progress.Total > 0 {
		percentage := float64(progress.Current) / float64(progress.Total) * 100
		if tf.useColor {
			result.WriteString(color.CyanString("⏳ 진행률: %.1f%% (%d/%d)",
				percentage, progress.Current, progress.Total))
		} else {
			result.WriteString(fmt.Sprintf("⏳ 진행률: %.1f%% (%d/%d)",
				percentage, progress.Current, progress.Total))
		}
	} else {
		if tf.useColor {
			result.WriteString(color.CyanString("⏳ 처리 중... (%d)", progress.Current))
		} else {
			result.WriteString(fmt.Sprintf("⏳ 처리 중... (%d)", progress.Current))
		}
	}

	// 현재 작업 표시
	if progress.CurrentTask != "" {
		result.WriteString("\n")
		if tf.useColor {
			result.WriteString(color.HiBlackString("  현재 작업: %s", progress.CurrentTask))
		} else {
			result.WriteString(fmt.Sprintf("  현재 작업: %s", progress.CurrentTask))
		}
	}

	return result.String()
}

// JSON 포맷터 구현

// FormatMessage는 Claude 메시지를 JSON 형식으로 포맷합니다.
func (jf *ClaudeJSONFormatter) FormatMessage(msg *claude.Message) string {
	if msg == nil {
		return ""
	}

	// Message를 JSON 직렬화할 수 있는 구조로 변환
	jsonMsg := map[string]interface{}{
		"type":      msg.Type,
		"content":   msg.Content,
		"id":        msg.ID,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	if len(msg.Meta) > 0 {
		jsonMsg["meta"] = msg.Meta
	}

	var output strings.Builder
	encoder := json.NewEncoder(&output)

	if jf.pretty {
		encoder.SetIndent("", "  ")
	}

	encoder.Encode(jsonMsg)
	return output.String()
}

// FormatError는 에러를 JSON 형식으로 포맷합니다.
func (jf *ClaudeJSONFormatter) FormatError(err error) string {
	if err == nil {
		return ""
	}

	errorObj := map[string]interface{}{
		"type":      "error",
		"message":   err.Error(),
		"timestamp": time.Now().Format(time.RFC3339),
	}

	var output strings.Builder
	encoder := json.NewEncoder(&output)

	if jf.pretty {
		encoder.SetIndent("", "  ")
	}

	encoder.Encode(errorObj)
	return output.String()
}

// FormatComplete는 실행 완료 요약을 JSON 형식으로 포맷합니다.
func (jf *ClaudeJSONFormatter) FormatComplete(summary *claude.ExecutionSummary) string {
	if summary == nil {
		return ""
	}

	var output strings.Builder
	encoder := json.NewEncoder(&output)

	if jf.pretty {
		encoder.SetIndent("", "  ")
	}

	encoder.Encode(summary)
	return output.String()
}

// FormatProgress는 진행 상황을 JSON 형식으로 포맷합니다.
func (jf *ClaudeJSONFormatter) FormatProgress(progress *claude.ProgressInfo) string {
	if progress == nil {
		return ""
	}

	var output strings.Builder
	encoder := json.NewEncoder(&output)

	if jf.pretty {
		encoder.SetIndent("", "  ")
	}

	encoder.Encode(progress)
	return output.String()
}

// Markdown 포맷터 구현

// FormatMessage는 Claude 메시지를 Markdown 형식으로 포맷합니다.
func (mf *MarkdownFormatter) FormatMessage(msg *claude.Message) string {
	if msg == nil {
		return ""
	}

	var result strings.Builder

	// 메시지 타입에 따른 마크다운 포맷팅
	switch msg.Type {
	case "text":
		result.WriteString(msg.Content)
	case "tool_use":
		toolName := "unknown"
		if name, ok := msg.Meta["tool_name"].(string); ok {
			toolName = name
		}
		result.WriteString(fmt.Sprintf("### 🔧 도구 사용: %s\n\n", toolName))
		if msg.Content != "" {
			result.WriteString("**입력:**\n```\n")
			result.WriteString(msg.Content)
			result.WriteString("\n```\n")
		}
	case "tool_result":
		success := true
		if s, ok := msg.Meta["success"].(bool); ok {
			success = s
		}

		if success {
			result.WriteString("### ✅ 도구 실행 결과\n\n")
		} else {
			result.WriteString("### ❌ 도구 실행 실패\n\n")
		}

		if msg.Content != "" {
			result.WriteString("**결과:**\n```\n")
			result.WriteString(msg.Content)
			result.WriteString("\n```\n")
		}
	case "error":
		result.WriteString(fmt.Sprintf("### ❌ 오류\n\n```\n%s\n```\n", msg.Content))
	case "system":
		result.WriteString(fmt.Sprintf("> ℹ️ %s\n\n", msg.Content))
	default:
		result.WriteString(fmt.Sprintf("**[%s]** %s\n\n", msg.Type, msg.Content))
	}

	return result.String()
}

// FormatError는 에러를 Markdown 형식으로 포맷합니다.
func (mf *MarkdownFormatter) FormatError(err error) string {
	if err == nil {
		return ""
	}

	return fmt.Sprintf("### ❌ 오류\n\n```\n%s\n```\n", err.Error())
}

// FormatComplete는 실행 완료 요약을 Markdown 형식으로 포맷합니다.
func (mf *MarkdownFormatter) FormatComplete(summary *claude.ExecutionSummary) string {
	if summary == nil {
		return ""
	}

	var result strings.Builder

	if summary.Success {
		result.WriteString("## ✅ 실행 완료\n\n")
	} else {
		result.WriteString("## ❌ 실행 실패\n\n")
	}

	// 실행 시간과 토큰 사용량
	result.WriteString("**실행 정보:**\n")
	if summary.Duration > 0 {
		duration := time.Duration(summary.Duration) * time.Millisecond
		result.WriteString(fmt.Sprintf("- 실행 시간: %s\n", duration.String()))
	}
	if summary.InputTokens > 0 || summary.OutputTokens > 0 {
		result.WriteString(fmt.Sprintf("- 토큰 사용량: 입력 %d, 출력 %d\n",
			summary.InputTokens, summary.OutputTokens))
	}
	result.WriteString("\n")

	return result.String()
}

// FormatProgress는 진행 상황을 Markdown 형식으로 포맷합니다.
func (mf *MarkdownFormatter) FormatProgress(progress *claude.ProgressInfo) string {
	if progress == nil {
		return ""
	}

	var result strings.Builder

	if progress.Total > 0 {
		percentage := float64(progress.Current) / float64(progress.Total) * 100
		result.WriteString(fmt.Sprintf("⏳ **진행률:** %.1f%% (%d/%d)\n\n",
			percentage, progress.Current, progress.Total))
	} else {
		result.WriteString(fmt.Sprintf("⏳ **처리 중...** (%d)\n\n", progress.Current))
	}

	if progress.CurrentTask != "" {
		result.WriteString(fmt.Sprintf("**현재 작업:** %s\n\n", progress.CurrentTask))
	}

	return result.String()
}
