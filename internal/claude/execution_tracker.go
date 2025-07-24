package claude

import (
	"context"
	"sync"
	"time"
)

// Message는 브로드캐스트 메시지 구조체입니다.
type TrackerMessage struct {
	Type string                 `json:"type"`
	Data map[string]interface{} `json:"data"`
}

// MessageBroadcaster는 메시지 브로드캐스트 인터페이스입니다.
type MessageBroadcaster interface {
	Broadcast(message interface{})
	BroadcastToWorkspace(workspaceID string, message interface{})
}

// ExecutionTracker는 Claude 실행 상태를 추적합니다.
type ExecutionTracker struct {
	executions map[string]*ExecutionStatus
	mu         sync.RWMutex
	broadcaster MessageBroadcaster
}

// ExecutionStatus는 실행 상태를 나타냅니다.
type ExecutionStatus struct {
	ID           string                 `json:"id"`
	SessionID    string                 `json:"session_id"`
	WorkspaceID  string                 `json:"workspace_id"`
	Status       string                 `json:"status"` // pending, running, completed, failed, cancelled
	Progress     float64                `json:"progress"`
	StartTime    time.Time              `json:"start_time"`
	EndTime      *time.Time             `json:"end_time,omitempty"`
	Messages     int                    `json:"messages"`
	Errors       []ExecutionError       `json:"errors,omitempty"`
	Result       interface{}            `json:"result,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// ExecutionError는 실행 중 발생한 에러 정보입니다.
type ExecutionError struct {
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
	Code      string    `json:"code,omitempty"`
	Details   string    `json:"details,omitempty"`
}

// ProgressUpdate는 진행 상황 업데이트 정보입니다.
type ProgressUpdate struct {
	Progress     float64
	MessageCount int
	Status       string
	Message      string
	Error        error
	Result       interface{}
}

// NewExecutionTracker는 새로운 실행 추적기를 생성합니다.
func NewExecutionTracker(broadcaster MessageBroadcaster) *ExecutionTracker {
	return &ExecutionTracker{
		executions: make(map[string]*ExecutionStatus),
		broadcaster: broadcaster,
	}
}

// StartExecution은 새로운 실행을 시작합니다.
func (t *ExecutionTracker) StartExecution(executionID, sessionID, workspaceID string) *ExecutionStatus {
	t.mu.Lock()
	defer t.mu.Unlock()

	status := &ExecutionStatus{
		ID:          executionID,
		SessionID:   sessionID,
		WorkspaceID: workspaceID,
		Status:      "pending",
		Progress:    0.0,
		StartTime:   time.Now(),
		Messages:    0,
		Errors:      []ExecutionError{},
		Metadata:    make(map[string]interface{}),
	}

	t.executions[executionID] = status

	// WebSocket으로 시작 알림
	t.broadcastStatusUpdate(status)

	return status
}

// UpdateProgress는 실행 진행 상황을 업데이트합니다.
func (t *ExecutionTracker) UpdateProgress(executionID string, update ProgressUpdate) {
	t.mu.Lock()
	defer t.mu.Unlock()

	status, exists := t.executions[executionID]
	if !exists {
		return
	}

	// 진행 상황 업데이트
	if update.Progress >= 0 {
		status.Progress = update.Progress
	}
	if update.MessageCount > 0 {
		status.Messages = update.MessageCount
	}
	if update.Status != "" {
		status.Status = update.Status
	}
	if update.Result != nil {
		status.Result = update.Result
	}

	// 에러 추가
	if update.Error != nil {
		execError := ExecutionError{
			Timestamp: time.Now(),
			Message:   update.Error.Error(),
		}

		// Claude 에러인 경우 추가 정보 포함
		if claudeErr, ok := update.Error.(*ClaudeError); ok {
			execError.Code = claudeErr.Code
			if details, ok := claudeErr.Details["details"].(string); ok {
				execError.Details = details
			}
		}

		status.Errors = append(status.Errors, execError)
	}

	// 완료 상태인 경우 종료 시간 설정
	if status.Status == "completed" || status.Status == "failed" || status.Status == "cancelled" {
		now := time.Now()
		status.EndTime = &now
	}

	// WebSocket으로 진행 상황 전송
	t.broadcastProgressUpdate(executionID, status, update.Message)
}

// GetExecution은 실행 상태를 조회합니다.
func (t *ExecutionTracker) GetExecution(executionID string) (*ExecutionStatus, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	status, exists := t.executions[executionID]
	return status, exists
}

// ListExecutions는 실행 목록을 조회합니다.
func (t *ExecutionTracker) ListExecutions(workspaceID string) []*ExecutionStatus {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var results []*ExecutionStatus
	for _, status := range t.executions {
		if workspaceID == "" || status.WorkspaceID == workspaceID {
			results = append(results, status)
		}
	}

	return results
}

// CompleteExecution은 실행을 완료 상태로 설정합니다.
func (t *ExecutionTracker) CompleteExecution(executionID string, result interface{}) {
	update := ProgressUpdate{
		Progress: 1.0,
		Status:   "completed",
		Result:   result,
		Message:  "Execution completed successfully",
	}
	t.UpdateProgress(executionID, update)
}

// FailExecution은 실행을 실패 상태로 설정합니다.
func (t *ExecutionTracker) FailExecution(executionID string, err error) {
	update := ProgressUpdate{
		Status:  "failed",
		Error:   err,
		Message: "Execution failed",
	}
	t.UpdateProgress(executionID, update)
}

// CancelExecution은 실행을 취소합니다.
func (t *ExecutionTracker) CancelExecution(executionID string) {
	update := ProgressUpdate{
		Status:  "cancelled",
		Message: "Execution cancelled by user",
	}
	t.UpdateProgress(executionID, update)
}

// CleanupExpiredExecutions는 만료된 실행 기록을 정리합니다.
func (t *ExecutionTracker) CleanupExpiredExecutions(maxAge time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	for id, status := range t.executions {
		// 완료된 실행 중에서 오래된 것들 삭제
		if status.EndTime != nil && status.EndTime.Before(cutoff) {
			delete(t.executions, id)
		}
	}
}

// broadcastStatusUpdate는 상태 업데이트를 WebSocket으로 브로드캐스트합니다.
func (t *ExecutionTracker) broadcastStatusUpdate(status *ExecutionStatus) {
	if t.broadcaster == nil {
		return
	}

	message := TrackerMessage{
		Type: "execution_status",
		Data: map[string]interface{}{
			"execution_id": status.ID,
			"session_id":   status.SessionID,
			"workspace_id": status.WorkspaceID,
			"status":       status.Status,
			"progress":     status.Progress,
			"start_time":   status.StartTime,
			"messages":     status.Messages,
		},
	}

	t.broadcaster.Broadcast(message)
}

// broadcastProgressUpdate는 진행 상황 업데이트를 WebSocket으로 브로드캐스트합니다.
func (t *ExecutionTracker) broadcastProgressUpdate(executionID string, status *ExecutionStatus, message string) {
	if t.broadcaster == nil {
		return
	}

	wsMessage := TrackerMessage{
		Type: "execution_progress",
		Data: map[string]interface{}{
			"execution_id": executionID,
			"session_id":   status.SessionID,
			"workspace_id": status.WorkspaceID,
			"status":       status.Status,
			"progress":     status.Progress,
			"messages":     status.Messages,
			"message":      message,
			"timestamp":    time.Now(),
		},
	}

	// 에러가 있는 경우 포함
	if len(status.Errors) > 0 {
		wsMessage.Data["latest_error"] = status.Errors[len(status.Errors)-1]
	}

	// 결과가 있는 경우 포함
	if status.Result != nil {
		wsMessage.Data["result"] = status.Result
	}

	t.broadcaster.Broadcast(wsMessage)
}

// GetExecutionStats는 실행 통계를 반환합니다.
func (t *ExecutionTracker) GetExecutionStats(workspaceID string) ExecutionStats {
	t.mu.RLock()
	defer t.mu.RUnlock()

	stats := ExecutionStats{
		Total:      0,
		Running:    0,
		Completed:  0,
		Failed:     0,
		Cancelled:  0,
		AvgDuration: 0,
	}

	var totalDuration time.Duration
	var completedCount int

	for _, status := range t.executions {
		if workspaceID != "" && status.WorkspaceID != workspaceID {
			continue
		}

		stats.Total++

		switch status.Status {
		case "running", "pending":
			stats.Running++
		case "completed":
			stats.Completed++
			completedCount++
			if status.EndTime != nil {
				duration := status.EndTime.Sub(status.StartTime)
				totalDuration += duration
			}
		case "failed":
			stats.Failed++
		case "cancelled":
			stats.Cancelled++
		}
	}

	// 평균 실행 시간 계산
	if completedCount > 0 {
		stats.AvgDuration = totalDuration / time.Duration(completedCount)
	}

	return stats
}

// ExecutionStats는 실행 통계 정보입니다.
type ExecutionStats struct {
	Total       int           `json:"total"`
	Running     int           `json:"running"`
	Completed   int           `json:"completed"`
	Failed      int           `json:"failed"`
	Cancelled   int           `json:"cancelled"`
	AvgDuration time.Duration `json:"avg_duration"`
}

// StartPeriodicCleanup은 주기적인 만료된 실행 기록 정리를 시작합니다.
func (t *ExecutionTracker) StartPeriodicCleanup(ctx context.Context, interval, maxAge time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			t.CleanupExpiredExecutions(maxAge)
		case <-ctx.Done():
			return
		}
	}
}