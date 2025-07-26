package models

// CreateProjectRequest 프로젝트 생성 요청
type CreateProjectRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=100" validate:"required,min=1,max=100,no_special_chars"`
	Path        string `json:"path" binding:"required,dir" validate:"required,dir,safepath"`
	Description string `json:"description" binding:"max=500" validate:"max=500"`
	GitURL      string `json:"git_url,omitempty" validate:"omitempty,url"`
	GitBranch   string `json:"git_branch,omitempty" validate:"omitempty,min=1,max=100"`
	Language    string `json:"language,omitempty" validate:"omitempty,min=1,max=50"`
}

// UpdateProjectRequest 프로젝트 수정 요청
type UpdateProjectRequest struct {
	Name        *string `json:"name,omitempty" binding:"omitempty,min=1,max=100" validate:"omitempty,min=1,max=100,no_special_chars"`
	Path        *string `json:"path,omitempty" binding:"omitempty,dir" validate:"omitempty,dir,safepath"`
	Description *string `json:"description,omitempty" binding:"omitempty,max=500" validate:"omitempty,max=500"`
	GitURL      *string `json:"git_url,omitempty" validate:"omitempty,url"`
	GitBranch   *string `json:"git_branch,omitempty" validate:"omitempty,min=1,max=100"`
	Language    *string `json:"language,omitempty" validate:"omitempty,min=1,max=50"`
}