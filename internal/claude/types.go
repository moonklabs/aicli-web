package claude

import "time"

// FormattedMessage는 포맷터에서 사용하는 확장된 메시지 구조체입니다.
type FormattedMessage struct {
	Type      string                 `json:"type"`
	Content   string                 `json:"content"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Index     int                    `json:"index,omitempty"`
	Timestamp time.Time              `json:"timestamp,omitempty"`
}

// ExecutionSummary는 Claude 실행 완료 시의 요약 정보입니다.
type ExecutionSummary struct {
	Success      bool          `json:"success"`
	Duration     int64         `json:"duration_ms"`     // 밀리초
	InputTokens  int           `json:"input_tokens"`
	OutputTokens int           `json:"output_tokens"`
	ErrorMessage string        `json:"error_message,omitempty"`
	SessionID    string        `json:"session_id,omitempty"`
	StartedAt    time.Time     `json:"started_at"`
	CompletedAt  time.Time     `json:"completed_at"`
	TotalSteps   int           `json:"total_steps,omitempty"`
	FailedSteps  int           `json:"failed_steps,omitempty"`
}

// ProgressInfo는 실행 진행 상황을 나타냅니다.
type ProgressInfo struct {
	Current     int    `json:"current"`
	Total       int    `json:"total"`
	CurrentTask string `json:"current_task,omitempty"`
	Stage       string `json:"stage,omitempty"`
	Percentage  float64 `json:"percentage"`
}

// ClaudeError는 Claude CLI에서 발생하는 에러를 나타냅니다.
type ClaudeError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
	Hint    string `json:"hint,omitempty"`
}

func (e *ClaudeError) Error() string {
	return e.Message
}

// NewFormattedMessage는 기본 Message를 FormattedMessage로 변환합니다.
func NewFormattedMessage(msg *Message) *FormattedMessage {
	if msg == nil {
		return nil
	}
	
	return &FormattedMessage{
		Type:      msg.Type,
		Content:   msg.Content,
		Metadata:  msg.Meta,
		Timestamp: time.Now(),
	}
}

// NewFormattedResponse는 Response를 FormattedMessage로 변환합니다.
func NewFormattedResponse(resp *Response, index int) *FormattedMessage {
	if resp == nil {
		return nil
	}
	
	return &FormattedMessage{
		Type:      resp.Type,
		Content:   resp.Content,
		Metadata:  resp.Metadata,
		Index:     index,
		Timestamp: time.Now(),
	}
}

// NewProgressInfo는 현재/전체 개수로 ProgressInfo를 생성합니다.
func NewProgressInfo(current, total int, task string) *ProgressInfo {
	percentage := float64(0)
	if total > 0 {
		percentage = float64(current) / float64(total) * 100
	}
	
	return &ProgressInfo{
		Current:     current,
		Total:       total,
		CurrentTask: task,
		Percentage:  percentage,
	}
}

// NewExecutionSummary는 실행 결과 요약을 생성합니다.
func NewExecutionSummary(success bool, startTime time.Time) *ExecutionSummary {
	now := time.Now()
	duration := now.Sub(startTime)
	
	return &ExecutionSummary{
		Success:     success,
		Duration:    duration.Milliseconds(),
		StartedAt:   startTime,
		CompletedAt: now,
	}
}