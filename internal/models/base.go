package models

import (
	"time"
)

// BaseModel 모든 모델의 공통 필드를 포함하는 기본 모델
type BaseModel struct {
	// 고유 식별자 (UUID v4)
	ID string `json:"id" gorm:"primaryKey;type:char(36)" validate:"omitempty,uuid"`
	
	// 레코드 생성 시간
	CreatedAt time.Time `json:"created_at" gorm:"not null" validate:"-"`
	
	// 레코드 업데이트 시간
	UpdatedAt time.Time `json:"updated_at" gorm:"not null" validate:"-"`
	
	// 버전 번호 (낙관적 잠금용)
	Version int `json:"version" gorm:"default:1" validate:"min=1"`
}

// ProjectResponse 프로젝트 응답 모델
type ProjectResponse struct {
	ID          string        `json:"id"`
	WorkspaceID string        `json:"workspace_id"`
	Name        string        `json:"name"`
	Path        string        `json:"path"`
	Description string        `json:"description"`
	GitURL      string        `json:"git_url,omitempty"`
	GitBranch   string        `json:"git_branch,omitempty"`
	Language    string        `json:"language,omitempty"`
	Status      ProjectStatus `json:"status"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// ToResponse Project를 ProjectResponse로 변환
func (p *Project) ToResponse() *ProjectResponse {
	return &ProjectResponse{
		ID:          p.ID,
		WorkspaceID: p.WorkspaceID,
		Name:        p.Name,
		Path:        p.Path,
		Description: p.Description,
		GitURL:      p.GitURL,
		GitBranch:   p.GitBranch,
		Language:    p.Language,
		Status:      p.Status,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}