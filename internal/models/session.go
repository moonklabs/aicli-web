package models

import (
	"sync"
	"time"
)

// SessionStatus 세션 상태를 나타내는 타입
type SessionStatus string

const (
	// SessionPending 세션 생성 중
	SessionPending SessionStatus = "pending"
	// SessionActive 세션 활성화 상태
	SessionActive SessionStatus = "active"
	// SessionIdle 세션 유휴 상태
	SessionIdle SessionStatus = "idle"
	// SessionEnding 세션 종료 중
	SessionEnding SessionStatus = "ending"
	// SessionEnded 세션 종료됨
	SessionEnded SessionStatus = "ended"
	// SessionError 세션 오류 상태
	SessionError SessionStatus = "error"
)

// Session Claude CLI 세션을 나타내는 모델
type Session struct {
	BaseModel
	ProjectID  string            `json:"project_id" gorm:"index;not null" validate:"required,uuid"`
	ProcessID  int               `json:"process_id" validate:"omitempty,min=0"`
	Status     SessionStatus     `json:"status" gorm:"default:'pending'" validate:"omitempty,session_status"`
	StartedAt  *time.Time        `json:"started_at" validate:"-"`
	EndedAt    *time.Time        `json:"ended_at" validate:"-"`
	LastActive time.Time         `json:"last_active" gorm:"not null" validate:"-"`
	Metadata   map[string]string `json:"metadata" gorm:"serializer:json" validate:"-"`
	
	// 연관 관계
	Project    *Project          `json:"project,omitempty" gorm:"foreignKey:ProjectID" validate:"-"`
	
	// 세션 통계
	CommandCount int64           `json:"command_count" gorm:"default:0" validate:"min=0"`
	BytesIn      int64           `json:"bytes_in" gorm:"default:0" validate:"min=0"`
	BytesOut     int64           `json:"bytes_out" gorm:"default:0" validate:"min=0"`
	ErrorCount   int64           `json:"error_count" gorm:"default:0" validate:"min=0"`
	
	// 리소스 제한
	MaxIdleTime  time.Duration   `json:"max_idle_time" gorm:"default:1800000000000" validate:"min=0"` // 30분
	MaxLifetime  time.Duration   `json:"max_lifetime" gorm:"default:14400000000000" validate:"min=0"` // 4시간
	
	mu sync.RWMutex `json:"-" gorm:"-"` // 동시성 제어를 위한 뮤텍스
}

// IsActive 세션이 활성 상태인지 확인
func (s *Session) IsActive() bool {
	return s.Status == SessionActive || s.Status == SessionIdle
}

// IsTerminated 세션이 종료 상태인지 확인
func (s *Session) IsTerminated() bool {
	return s.Status == SessionEnded || s.Status == SessionError
}

// IsIdleTimeout 유휴 타임아웃 확인
func (s *Session) IsIdleTimeout() bool {
	if s.MaxIdleTime == 0 || !s.IsActive() {
		return false
	}
	return time.Since(s.LastActive) > s.MaxIdleTime
}

// IsLifetimeTimeout 생명주기 타임아웃 확인
func (s *Session) IsLifetimeTimeout() bool {
	if s.MaxLifetime == 0 || s.StartedAt == nil || !s.IsActive() {
		return false
	}
	return time.Since(*s.StartedAt) > s.MaxLifetime
}

// UpdateActivity 활동 시간 업데이트
func (s *Session) UpdateActivity() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.LastActive = time.Now()
}

// Clone Session의 깊은 복사본을 생성 (mutex는 제외)
func (s *Session) Clone() *Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	clone := &Session{
		BaseModel:    s.BaseModel,
		ProjectID:    s.ProjectID,
		ProcessID:    s.ProcessID,
		Status:       s.Status,
		StartedAt:    s.StartedAt,
		EndedAt:      s.EndedAt,
		LastActive:   s.LastActive,
		CommandCount: s.CommandCount,
		BytesIn:      s.BytesIn,
		BytesOut:     s.BytesOut,
		ErrorCount:   s.ErrorCount,
		MaxIdleTime:  s.MaxIdleTime,
		MaxLifetime:  s.MaxLifetime,
	}
	
	// Metadata 깊은 복사
	if s.Metadata != nil {
		clone.Metadata = make(map[string]string)
		for k, v := range s.Metadata {
			clone.Metadata[k] = v
		}
	}
	
	// Project는 포인터 복사만 (깊은 복사 불필요)
	clone.Project = s.Project
	
	return clone
}

// SessionFilter 세션 필터링 옵션
type SessionFilter struct {
	ProjectID string
	Status    SessionStatus
	Active    *bool // true: 활성 세션만, false: 비활성 세션만, nil: 전체
}

// SessionCreateRequest 세션 생성 요청
type SessionCreateRequest struct {
	ProjectID   string            `json:"project_id" validate:"required,uuid"`
	Metadata    map[string]string `json:"metadata" validate:"-"`
	MaxIdleTime *time.Duration    `json:"max_idle_time" validate:"omitempty,min=0"`
	MaxLifetime *time.Duration    `json:"max_lifetime" validate:"omitempty,min=0"`
}

// SessionResponse 세션 응답
type SessionResponse struct {
	*Session
	Project *ProjectResponse `json:"project,omitempty"`
}

// ToResponse Session을 SessionResponse로 변환
func (s *Session) ToResponse() *SessionResponse {
	resp := &SessionResponse{
		Session: s,
	}
	if s.Project != nil {
		resp.Project = s.Project.ToResponse()
	}
	return resp
}