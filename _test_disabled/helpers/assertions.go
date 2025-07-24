package helpers

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/aicli/aicli-web/internal/claude"
)

// WebSocketMessage는 WebSocket 메시지 구조체입니다
type WebSocketMessage struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}

// AssertMessageTypes는 메시지 타입 순서를 검증합니다
func AssertMessageTypes(t *testing.T, messages []claude.Message, expected []string) {
	t.Helper()
	
	if !assert.Len(t, messages, len(expected), "메시지 수가 예상과 다름") {
		return
	}
	
	for i, msg := range messages {
		assert.Equal(t, expected[i], msg.Type, "메시지 타입이 예상과 다름: 인덱스 %d", i)
	}
}

// AssertContainsToolUse는 특정 도구 사용이 포함되어 있는지 검증합니다
func AssertContainsToolUse(t *testing.T, messages []claude.Message, toolName string) {
	t.Helper()
	
	found := false
	for _, msg := range messages {
		if msg.Type == "tool_use" && msg.ToolName == toolName {
			found = true
			break
		}
	}
	
	assert.True(t, found, "Expected tool use '%s' not found", toolName)
}

// AssertToolUseCount는 특정 도구 사용 횟수를 검증합니다
func AssertToolUseCount(t *testing.T, messages []claude.Message, toolName string, expectedCount int) {
	t.Helper()
	
	count := 0
	for _, msg := range messages {
		if msg.Type == "tool_use" && msg.ToolName == toolName {
			count++
		}
	}
	
	assert.Equal(t, expectedCount, count, "Tool '%s' usage count mismatch", toolName)
}

// AssertGeneratedCode는 생성된 코드가 특정 내용을 포함하는지 검증합니다
func AssertGeneratedCode(t *testing.T, messages []claude.Message, expectedContent string) {
	t.Helper()
	
	found := false
	for _, msg := range messages {
		if msg.Type == "tool_use" && msg.ToolName == "Write" {
			if input, ok := msg.Input.(map[string]interface{}); ok {
				if content, ok := input["content"].(string); ok {
					if strings.Contains(content, expectedContent) {
						found = true
						break
					}
				}
			}
		}
	}
	
	assert.True(t, found, "Expected code content '%s' not found in generated code", expectedContent)
}

// AssertWebSocketMessageTypes는 WebSocket 메시지 타입을 검증합니다
func AssertWebSocketMessageTypes(t *testing.T, messages []WebSocketMessage, expectedTypes []string) {
	t.Helper()
	
	if len(messages) < len(expectedTypes) {
		t.Errorf("Not enough messages: got %d, expected at least %d", len(messages), len(expectedTypes))
		return
	}
	
	for i, expectedType := range expectedTypes {
		if i < len(messages) {
			assert.Equal(t, expectedType, messages[i].Type, "Message type mismatch at index %d", i)
		}
	}
}

// AssertContainsWebSocketToolUse는 WebSocket 메시지에서 특정 도구 사용을 검증합니다
func AssertContainsWebSocketToolUse(t *testing.T, messages []WebSocketMessage, toolName string) {
	t.Helper()
	
	found := false
	for _, msg := range messages {
		if msg.Type == "claude_message" {
			if claudeMsg, ok := msg.Data.(map[string]interface{}); ok {
				if claudeMsg["type"] == "tool_use" && claudeMsg["tool_name"] == toolName {
					found = true
					break
				}
			}
		}
	}
	
	assert.True(t, found, "Expected tool use '%s' not found in WebSocket messages", toolName)
}

// AssertSessionState는 세션 상태를 검증합니다
func AssertSessionState(t *testing.T, session *claude.Session, expectedState string) {
	t.Helper()
	
	assert.NotNil(t, session, "Session should not be nil")
	if session != nil {
		assert.Equal(t, expectedState, session.State.String(), "Session state mismatch")
	}
}

// AssertProcessState는 프로세스 상태를 검증합니다
func AssertProcessState(t *testing.T, pm claude.ProcessManager, expectedState claude.ProcessState) {
	t.Helper()
	
	actualState := pm.GetStatus()
	assert.Equal(t, expectedState, actualState, "Process state mismatch")
}

// AssertStreamMetrics는 스트림 메트릭을 검증합니다
func AssertStreamMetrics(t *testing.T, metrics *claude.StreamMetrics, minProcessed int64) {
	t.Helper()
	
	assert.NotNil(t, metrics, "Stream metrics should not be nil")
	if metrics != nil {
		assert.GreaterOrEqual(t, metrics.ProcessedCount, minProcessed, "Processed count too low")
		assert.GreaterOrEqual(t, metrics.ProcessedCount, metrics.ErrorCount, "Error count cannot exceed processed count")
	}
}

// AssertErrorType는 에러 타입을 검증합니다
func AssertErrorType(t *testing.T, err error, expectedType claude.ErrorType) {
	t.Helper()
	
	assert.Error(t, err, "Expected an error")
	
	if processErr, ok := err.(*claude.ProcessError); ok {
		assert.Equal(t, expectedType, processErr.Type, "Error type mismatch")
	} else {
		t.Errorf("Expected ProcessError, got %T", err)
	}
}

// AssertPerformanceThreshold는 성능 임계값을 검증합니다
func AssertPerformanceThreshold(t *testing.T, actual, threshold float64, metric string) {
	t.Helper()
	
	assert.LessOrEqual(t, actual, threshold, "%s performance threshold exceeded: %.2f > %.2f", metric, actual, threshold)
}

// AssertMemoryUsage는 메모리 사용량을 검증합니다
func AssertMemoryUsage(t *testing.T, actualMB, thresholdMB float64) {
	t.Helper()
	
	assert.LessOrEqual(t, actualMB, thresholdMB, "Memory usage exceeded threshold: %.2f MB > %.2f MB", actualMB, thresholdMB)
}

// AssertConcurrentExecution은 동시 실행 결과를 검증합니다
func AssertConcurrentExecution(t *testing.T, results []error, expectedSuccesses int) {
	t.Helper()
	
	successCount := 0
	errorMessages := make([]string, 0)
	
	for i, err := range results {
		if err == nil {
			successCount++
		} else {
			errorMessages = append(errorMessages, fmt.Sprintf("Task %d: %v", i, err))
		}
	}
	
	assert.Equal(t, expectedSuccesses, successCount, "Unexpected number of successes")
	
	if len(errorMessages) > 0 {
		t.Logf("Errors encountered: %s", strings.Join(errorMessages, "; "))
	}
}

// AssertResponseTime은 응답 시간을 검증합니다
func AssertResponseTime(t *testing.T, duration time.Duration, threshold time.Duration, operation string) {
	t.Helper()
	
	assert.LessOrEqual(t, duration, threshold, "%s took too long: %v > %v", operation, duration, threshold)
}

// AssertThroughput은 처리량을 검증합니다
func AssertThroughput(t *testing.T, processed int, duration time.Duration, minThroughput float64) {
	t.Helper()
	
	actualThroughput := float64(processed) / duration.Seconds()
	assert.GreaterOrEqual(t, actualThroughput, minThroughput, "Throughput too low: %.2f < %.2f items/sec", actualThroughput, minThroughput)
}