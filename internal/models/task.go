package models

import (
	"time"
)

// TaskStatus 태스크 상태 열거형
type TaskStatus string

const (
	TaskPending   TaskStatus = "pending"   // 대기 중
	TaskRunning   TaskStatus = "running"   // 실행 중
	TaskCompleted TaskStatus = "completed" // 완료
	TaskFailed    TaskStatus = "failed"    // 실패
	TaskCancelled TaskStatus = "cancelled" // 취소됨
)

// Task 태스크 모델
type Task struct {
	BaseModel
	SessionID   string     `json:"session_id" gorm:"not null;index"`
	Command     string     `json:"command" binding:"required" gorm:"not null"`
	Status      TaskStatus `json:"status" gorm:"default:'pending';index"`
	Output      string     `json:"output,omitempty" gorm:"type:text"`
	Error       string     `json:"error,omitempty" gorm:"type:text"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	
	// 통계 정보
	BytesIn  int64 `json:"bytes_in" gorm:"default:0"`
	BytesOut int64 `json:"bytes_out" gorm:"default:0"`
	Duration int64 `json:"duration" gorm:"default:0"` // 실행 시간 (밀리초)
}

// TaskCreateRequest 태스크 생성 요청
type TaskCreateRequest struct {
	SessionID string            `json:"session_id" binding:"required"`
	Command   string            `json:"command" binding:"required,min=1,max=10000"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// TaskResponse 태스크 응답
type TaskResponse struct {
	ID          string     `json:"id"`
	SessionID   string     `json:"session_id"`
	Command     string     `json:"command"`
	Status      TaskStatus `json:"status"`
	Output      string     `json:"output,omitempty"`
	Error       string     `json:"error,omitempty"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	BytesIn     int64      `json:"bytes_in"`
	BytesOut    int64      `json:"bytes_out"`
	Duration    int64      `json:"duration"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// TaskFilter 태스크 필터링
type TaskFilter struct {
	SessionID *string     `form:"session_id"`
	Status    *TaskStatus `form:"status"`
	Active    *bool       `form:"active"` // 실행 중인 태스크만 조회
}

// TaskUpdateRequest 태스크 업데이트 요청
type TaskUpdateRequest struct {
	Status TaskStatus `json:"status" binding:"required"`
	Output string     `json:"output,omitempty"`
	Error  string     `json:"error,omitempty"`
}

// IsActive 태스크가 활성 상태인지 확인
func (t *Task) IsActive() bool {
	return t.Status == TaskPending || t.Status == TaskRunning
}

// IsTerminal 태스크가 종료 상태인지 확인
func (t *Task) IsTerminal() bool {
	return t.Status == TaskCompleted || t.Status == TaskFailed || t.Status == TaskCancelled
}

// CanCancel 태스크를 취소할 수 있는지 확인
func (t *Task) CanCancel() bool {
	return t.Status == TaskPending || t.Status == TaskRunning
}

// SetRunning 태스크를 실행 중 상태로 변경
func (t *Task) SetRunning() {
	if t.Status == TaskPending {
		t.Status = TaskRunning
		now := time.Now()
		t.StartedAt = &now
	}
}

// SetCompleted 태스크를 완료 상태로 변경
func (t *Task) SetCompleted(output string) {
	if t.Status == TaskRunning {
		t.Status = TaskCompleted
		now := time.Now()
		t.CompletedAt = &now
		t.Output = output
		
		if t.StartedAt != nil {
			t.Duration = now.Sub(*t.StartedAt).Milliseconds()
		}
	}
}

// SetFailed 태스크를 실패 상태로 변경
func (t *Task) SetFailed(errorMsg string) {
	if t.Status == TaskRunning {
		t.Status = TaskFailed
		now := time.Now()
		t.CompletedAt = &now
		t.Error = errorMsg
		
		if t.StartedAt != nil {
			t.Duration = now.Sub(*t.StartedAt).Milliseconds()
		}
	}
}

// SetCancelled 태스크를 취소 상태로 변경
func (t *Task) SetCancelled() {
	if t.CanCancel() {
		t.Status = TaskCancelled
		now := time.Now()
		t.CompletedAt = &now
		
		if t.StartedAt != nil {
			t.Duration = now.Sub(*t.StartedAt).Milliseconds()
		}
	}
}

// ToResponse 태스크를 응답 모델로 변환
func (t *Task) ToResponse() *TaskResponse {
	return &TaskResponse{
		ID:          t.ID,
		SessionID:   t.SessionID,
		Command:     t.Command,
		Status:      t.Status,
		Output:      t.Output,
		Error:       t.Error,
		StartedAt:   t.StartedAt,
		CompletedAt: t.CompletedAt,
		BytesIn:     t.BytesIn,
		BytesOut:    t.BytesOut,
		Duration:    t.Duration,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
}