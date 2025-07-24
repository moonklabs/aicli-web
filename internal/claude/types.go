package claude

import (
	"context"
	"time"
)

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

// Message represents a message in the Claude CLI communication
type Message struct {
	ID      string                 `json:"id,omitempty"`
	Type    string                 `json:"type"`
	Content string                 `json:"content"`
	Meta    map[string]interface{} `json:"metadata,omitempty"`
}

// NewFormattedMessage는 StreamMessage를 FormattedMessage로 변환합니다.
func NewFormattedMessage(msg *StreamMessage) *FormattedMessage {
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

// === 공통 타입 정의 (중복 제거) ===

// RecoveryStrategy는 복구 전략 인터페이스입니다
type RecoveryStrategy interface {
	// 복구 가능 여부 확인
	CanRecover(ctx context.Context, err error) bool
	
	// 복구 실행
	Execute(ctx context.Context, target RecoveryTarget) error
	
	// 예상 소요 시간
	GetEstimatedTime() time.Duration
	
	// 성공률
	GetSuccessRate() float64
	
	// 전략명
	GetName() string
}

// RecoveryTarget은 복구 대상입니다
type RecoveryTarget interface {
	GetType() string
	GetIdentifier() string
	GetContext() map[string]interface{}
}

// AlertManager는 알림 관리 인터페이스입니다
type AlertManager interface {
	SendAlert(level AlertLevel, message string, context map[string]interface{}) error
}

// AlertLevel은 알림 레벨입니다
type AlertLevel int

const (
	AlertLevelInfo AlertLevel = iota
	AlertLevelWarning
	AlertLevelError
	AlertLevelCritical
)

// PoolStats는 풀 통계 정보입니다
type PoolStats struct {
	Active      int     `json:"active"`
	Idle        int     `json:"idle"`
	Total       int     `json:"total"`
	MaxCapacity int     `json:"max_capacity"`
	Utilization float64 `json:"utilization"`
}

// StateTransition은 상태 전환을 나타냅니다
type StateTransition struct {
	FromState string    `json:"from_state"`
	ToState   string    `json:"to_state"`
	Trigger   string    `json:"trigger"`
	Timestamp time.Time `json:"timestamp"`
}