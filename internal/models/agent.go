package models

import (
	"sync"
	"time"
)

// AgentType 에이전트 타입
type AgentType string

const (
	AgentTypeClaude AgentType = "claude"
	AgentTypeGemini AgentType = "gemini"
	AgentTypeCustom AgentType = "custom"
)

// AgentStatus 에이전트 상태
type AgentStatus string

const (
	AgentStatusCreated    AgentStatus = "created"
	AgentStatusStarting   AgentStatus = "starting"
	AgentStatusRunning    AgentStatus = "running"
	AgentStatusStopping   AgentStatus = "stopping"
	AgentStatusStopped    AgentStatus = "stopped"
	AgentStatusError      AgentStatus = "error"
	AgentStatusTerminated AgentStatus = "terminated"
)

// AgentConfig 에이전트 설정
type AgentConfig struct {
	// AI 모델 설정
	Model        string  `json:"model,omitempty" validate:"omitempty,min=1,max=100"`
	MaxTokens    int     `json:"max_tokens,omitempty" validate:"omitempty,min=1,max=200000"`
	Temperature  float32 `json:"temperature,omitempty" validate:"omitempty,min=0,max=1"`
	SystemPrompt string  `json:"system_prompt,omitempty" validate:"omitempty,max=10000"`

	// 에이전트별 환경변수
	Environment map[string]string `json:"environment,omitempty" validate:"-"`

	// 작업 디렉토리 설정
	WorkingDir   string   `json:"working_dir,omitempty" validate:"omitempty,dir"`
	ExcludePaths []string `json:"exclude_paths,omitempty" validate:"dive,min=1"`
	IncludePaths []string `json:"include_paths,omitempty" validate:"dive,min=1"`

	// 리소스 제한
	MemoryLimit string `json:"memory_limit,omitempty" validate:"omitempty,memory_size"`
	CPULimit    string `json:"cpu_limit,omitempty" validate:"omitempty,cpu_limit"`

	// 자동 저장 설정
	AutoSave         bool `json:"auto_save,omitempty" validate:"-"`
	SaveIntervalSecs int  `json:"save_interval_secs,omitempty" validate:"omitempty,min=1,max=3600"`
}

// Agent 에이전트 모델
type Agent struct {
	// 기본 식별자
	ID        string    `json:"id" gorm:"primaryKey;type:char(36)" validate:"omitempty,uuid"`
	ProjectID string    `json:"project_id" gorm:"type:char(36);not null;index" validate:"required,uuid"`
	Name      string    `json:"name" gorm:"type:varchar(100);not null" validate:"required,min=1,max=100,no_special_chars"`
	Type      AgentType `json:"type" gorm:"type:varchar(20);not null" validate:"required,agent_type"`

	// 에이전트 상태
	Status AgentStatus `json:"status" gorm:"type:varchar(20);not null;default:'created'" validate:"omitempty,agent_status"`

	// 외부 연동 ID
	WorktreeID  string `json:"worktree_id,omitempty" gorm:"type:varchar(100);index" validate:"omitempty,min=1,max=100"`
	ContainerID string `json:"container_id,omitempty" gorm:"type:varchar(100);index" validate:"omitempty,min=1,max=100"`
	SessionID   string `json:"session_id,omitempty" gorm:"type:varchar(100);index" validate:"omitempty,min=1,max=100"`

	// 설정 및 메타데이터
	Config      AgentConfig `json:"config" gorm:"type:text" validate:"-"`
	Description string      `json:"description,omitempty" gorm:"type:varchar(500)" validate:"omitempty,max=500"`

	// 상태 정보
	LastActivity time.Time  `json:"last_activity,omitempty" gorm:"index" validate:"-"`
	ErrorMessage string     `json:"error_message,omitempty" gorm:"type:text" validate:"omitempty,max=2000"`
	Version      int        `json:"version" gorm:"default:1" validate:"min=1"`

	// 타임스탬프
	CreatedAt time.Time  `json:"created_at" gorm:"not null" validate:"-"`
	UpdatedAt time.Time  `json:"updated_at" gorm:"not null" validate:"-"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" gorm:"index" validate:"-"`

	// 관계 (GORM)
	Project *Project `json:"project,omitempty" gorm:"foreignKey:ProjectID" validate:"-"`

	// 동시성 제어 (JSON 직렬화 제외)
	mu sync.RWMutex `json:"-" gorm:"-"`
}

// AgentResponse 에이전트 응답 모델
type AgentResponse struct {
	ID           string       `json:"id"`
	ProjectID    string       `json:"project_id"`
	Name         string       `json:"name"`
	Type         AgentType    `json:"type"`
	Status       AgentStatus  `json:"status"`
	WorktreeID   string       `json:"worktree_id,omitempty"`
	ContainerID  string       `json:"container_id,omitempty"`
	SessionID    string       `json:"session_id,omitempty"`
	Config       AgentConfig  `json:"config"`
	Description  string       `json:"description,omitempty"`
	LastActivity time.Time    `json:"last_activity,omitempty"`
	ErrorMessage string       `json:"error_message,omitempty"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
	Project      *Project     `json:"project,omitempty"`
}

// ToResponse Agent를 AgentResponse로 변환
func (a *Agent) ToResponse() *AgentResponse {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return &AgentResponse{
		ID:           a.ID,
		ProjectID:    a.ProjectID,
		Name:         a.Name,
		Type:         a.Type,
		Status:       a.Status,
		WorktreeID:   a.WorktreeID,
		ContainerID:  a.ContainerID,
		SessionID:    a.SessionID,
		Config:       a.Config,
		Description:  a.Description,
		LastActivity: a.LastActivity,
		ErrorMessage: a.ErrorMessage,
		CreatedAt:    a.CreatedAt,
		UpdatedAt:    a.UpdatedAt,
		Project:      a.Project,
	}
}

// Clone 에이전트의 안전한 복사본 생성
func (a *Agent) Clone() *Agent {
	a.mu.RLock()
	defer a.mu.RUnlock()

	clone := &Agent{
		ID:           a.ID,
		ProjectID:    a.ProjectID,
		Name:         a.Name,
		Type:         a.Type,
		Status:       a.Status,
		WorktreeID:   a.WorktreeID,
		ContainerID:  a.ContainerID,
		SessionID:    a.SessionID,
		Config:       a.Config,
		Description:  a.Description,
		LastActivity: a.LastActivity,
		ErrorMessage: a.ErrorMessage,
		Version:      a.Version,
		CreatedAt:    a.CreatedAt,
		UpdatedAt:    a.UpdatedAt,
		DeletedAt:    a.DeletedAt,
		Project:      a.Project,
	}

	return clone
}

// UpdateStatus 상태 업데이트 (동시성 안전)
func (a *Agent) UpdateStatus(status AgentStatus, errorMessage string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.Status = status
	a.LastActivity = time.Now()
	a.ErrorMessage = errorMessage
	a.UpdatedAt = time.Now()
}

// UpdateActivity 활동 시간 업데이트
func (a *Agent) UpdateActivity() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.LastActivity = time.Now()
	a.UpdatedAt = time.Now()
}

// IsActive 에이전트 활성 상태 확인
func (a *Agent) IsActive() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.Status == AgentStatusRunning && a.DeletedAt == nil
}

// IsTerminal 종료 상태인지 확인
func (a *Agent) IsTerminal() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.Status == AgentStatusStopped ||
		a.Status == AgentStatusError ||
		a.Status == AgentStatusTerminated ||
		a.DeletedAt != nil
}

// GetDisplayStatus 표시용 상태 반환
func (a *Agent) GetDisplayStatus() string {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.DeletedAt != nil {
		return "deleted"
	}
	return string(a.Status)
}

// GetResourceUsage 리소스 사용량 정보 반환
func (a *Agent) GetResourceUsage() map[string]interface{} {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return map[string]interface{}{
		"memory_limit": a.Config.MemoryLimit,
		"cpu_limit":    a.Config.CPULimit,
		"container_id": a.ContainerID,
		"status":       a.Status,
		"last_activity": a.LastActivity,
	}
}