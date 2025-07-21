package models

import (
	"time"
)

// ProjectStatus 프로젝트 상태
type ProjectStatus string

const (
	ProjectStatusActive   ProjectStatus = "active"
	ProjectStatusInactive ProjectStatus = "inactive"
	ProjectStatusArchived ProjectStatus = "archived"
)

// Project 프로젝트 모델
type Project struct {
	ID           string         `json:"id" validate:"omitempty,uuid"`
	WorkspaceID  string         `json:"workspace_id" binding:"required" validate:"required,uuid"`
	Name         string         `json:"name" binding:"required,min=1,max=100" validate:"required,min=1,max=100,no_special_chars"`
	Path         string         `json:"path" binding:"required,dir" validate:"required,dir,safepath"` // 커스텀 validator 사용
	Description  string         `json:"description" binding:"max=500" validate:"max=500"`
	GitURL       string         `json:"git_url,omitempty" validate:"omitempty,url"`
	GitBranch    string         `json:"git_branch,omitempty" validate:"omitempty,min=1,max=100"`
	Language     string         `json:"language,omitempty" validate:"omitempty,min=1,max=50"`
	Status       ProjectStatus  `json:"status" validate:"omitempty,project_status"`
	Config       ProjectConfig  `json:"config" validate:"-"`
	GitInfo      *GitInfo       `json:"git_info,omitempty" validate:"-"`
	CreatedAt    time.Time      `json:"created_at" validate:"-"`
	UpdatedAt    time.Time      `json:"updated_at" validate:"-"`
	DeletedAt    *time.Time     `json:"deleted_at,omitempty" validate:"-"`
}

// ProjectConfig 프로젝트 설정
type ProjectConfig struct {
	ClaudeAPIKey    string            `json:"-" validate:"omitempty,claude_api_key"` // 보안상 JSON 직렬화에서 제외
	EncryptedAPIKey string            `json:"encrypted_api_key,omitempty" validate:"omitempty"`
	Environment     map[string]string `json:"environment,omitempty" validate:"-"`
	ClaudeOptions   ClaudeOptions     `json:"claude_options" validate:"-"`
	BuildCommands   []string          `json:"build_commands,omitempty" validate:"dive,min=1"`
	TestCommands    []string          `json:"test_commands,omitempty" validate:"dive,min=1"`
}

// ClaudeOptions Claude CLI 옵션
type ClaudeOptions struct {
	Model           string   `json:"model,omitempty" validate:"omitempty,min=1,max=100"`
	MaxTokens       int      `json:"max_tokens,omitempty" validate:"omitempty,min=1,max=200000"`
	Temperature     float32  `json:"temperature,omitempty" validate:"omitempty,min=0,max=1"`
	SystemPrompt    string   `json:"system_prompt,omitempty" validate:"omitempty,max=10000"`
	ExcludePaths    []string `json:"exclude_paths,omitempty" validate:"dive,min=1"`
	IncludePaths    []string `json:"include_paths,omitempty" validate:"dive,min=1"`
}

// GitInfo Git 리포지토리 정보
type GitInfo struct {
	RemoteURL     string       `json:"remote_url"`
	CurrentBranch string       `json:"current_branch"`
	IsClean       bool         `json:"is_clean"`
	LastCommit    *CommitInfo  `json:"last_commit,omitempty"`
	Status        *GitStatus   `json:"status,omitempty"`
}

// CommitInfo 커밋 정보
type CommitInfo struct {
	Hash      string    `json:"hash"`
	Author    string    `json:"author"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// GitStatus Git 상태 정보
type GitStatus struct {
	Modified   []string `json:"modified,omitempty"`
	Added      []string `json:"added,omitempty"`
	Deleted    []string `json:"deleted,omitempty"`
	Untracked  []string `json:"untracked,omitempty"`
	HasChanges bool     `json:"has_changes"`
}

// IsValid 프로젝트 유효성 검사
func (p *Project) IsValid() bool {
	return p.Status == ProjectStatusActive && p.DeletedAt == nil
}

// GetDisplayStatus 표시용 상태 반환
func (p *Project) GetDisplayStatus() string {
	if p.DeletedAt != nil {
		return "deleted"
	}
	return string(p.Status)
}