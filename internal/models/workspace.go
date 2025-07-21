package models

import (
	"time"
)

// WorkspaceStatus 워크스페이스 상태
type WorkspaceStatus string

const (
	// WorkspaceStatusActive 활성 상태
	WorkspaceStatusActive WorkspaceStatus = "active"
	// WorkspaceStatusInactive 비활성 상태
	WorkspaceStatusInactive WorkspaceStatus = "inactive"
	// WorkspaceStatusArchived 아카이브 상태
	WorkspaceStatusArchived WorkspaceStatus = "archived"
)

// Workspace 워크스페이스 모델
// swagger:model Workspace
type Workspace struct {
	// 워크스페이스 ID
	// example: ws_123456789
	ID string `json:"id" binding:"-"`
	
	// 워크스페이스 이름
	// example: My Project
	Name string `json:"name" binding:"required,min=1,max=100"`
	
	// 프로젝트 경로
	// example: /home/user/projects/myproject
	ProjectPath string `json:"project_path" binding:"required,dir"`
	
	// 워크스페이스 상태
	// example: active
	Status WorkspaceStatus `json:"status" binding:"-"`
	
	// 소유자 ID
	// example: user_123456789
	OwnerID string `json:"owner_id" binding:"-"`
	
	// Claude API 키 (선택적, 응답에서는 마스킹)
	// example: sk-ant-...
	ClaudeKey string `json:"claude_key,omitempty" binding:"omitempty"`
	
	// 활성 태스크 수
	// example: 3
	ActiveTasks int `json:"active_tasks" binding:"-"`
	
	// 생성 시간
	// example: 2025-07-21T14:31:00Z
	CreatedAt time.Time `json:"created_at" binding:"-"`
	
	// 수정 시간
	// example: 2025-07-21T14:31:00Z
	UpdatedAt time.Time `json:"updated_at" binding:"-"`
	
	// 삭제 시간 (soft delete)
	DeletedAt *time.Time `json:"deleted_at,omitempty" binding:"-"`
}

// CreateWorkspaceRequest 워크스페이스 생성 요청
// swagger:model CreateWorkspaceRequest
type CreateWorkspaceRequest struct {
	// 워크스페이스 이름
	// example: My New Project
	Name string `json:"name" binding:"required,min=1,max=100"`
	
	// 프로젝트 경로
	// example: /home/user/projects/newproject
	ProjectPath string `json:"project_path" binding:"required"`
	
	// Claude API 키 (선택적)
	// example: sk-ant-api03-...
	ClaudeKey string `json:"claude_key,omitempty" binding:"omitempty"`
}

// UpdateWorkspaceRequest 워크스페이스 수정 요청
// swagger:model UpdateWorkspaceRequest
type UpdateWorkspaceRequest struct {
	// 워크스페이스 이름
	// example: Updated Project Name
	Name string `json:"name,omitempty" binding:"omitempty,min=1,max=100"`
	
	// 프로젝트 경로
	// example: /home/user/projects/updated-path
	ProjectPath string `json:"project_path,omitempty" binding:"omitempty"`
	
	// Claude API 키
	// example: sk-ant-api03-...
	ClaudeKey string `json:"claude_key,omitempty" binding:"omitempty"`
	
	// 워크스페이스 상태
	// example: inactive
	Status WorkspaceStatus `json:"status,omitempty" binding:"omitempty,oneof=active inactive archived"`
}

// WorkspaceListResponse 워크스페이스 목록 응답
// swagger:model WorkspaceListResponse
type WorkspaceListResponse struct {
	// 성공 여부
	// example: true
	Success bool `json:"success"`
	
	// 워크스페이스 목록
	Data []Workspace `json:"data"`
	
	// 페이지네이션 메타 정보
	Meta PaginationMeta `json:"meta"`
}

// WorkspaceResponse 단일 워크스페이스 응답
// swagger:model WorkspaceResponse
type WorkspaceResponse struct {
	// 성공 여부
	// example: true
	Success bool `json:"success"`
	
	// 워크스페이스 정보
	Data Workspace `json:"data"`
}

// IsValid 워크스페이스 상태 유효성 검사
func (s WorkspaceStatus) IsValid() bool {
	switch s {
	case WorkspaceStatusActive, WorkspaceStatusInactive, WorkspaceStatusArchived:
		return true
	default:
		return false
	}
}

// MaskClaudeKey Claude API 키 마스킹
func (w *Workspace) MaskClaudeKey() {
	if w.ClaudeKey != "" {
		if len(w.ClaudeKey) > 10 {
			w.ClaudeKey = w.ClaudeKey[:10] + "..."
		} else {
			w.ClaudeKey = "***"
		}
	}
}