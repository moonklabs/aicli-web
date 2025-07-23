package interfaces

import (
	"context"
	"time"

	"github.com/aicli/aicli-web/internal/models"
)

// WorkspaceService는 워크스페이스 비즈니스 로직을 처리하는 서비스 인터페이스입니다
type WorkspaceService interface {
	// CRUD 기본 오퍼레이션
	CreateWorkspace(ctx context.Context, req *models.CreateWorkspaceRequest, ownerID string) (*models.Workspace, error)
	GetWorkspace(ctx context.Context, id string, ownerID string) (*models.Workspace, error)
	UpdateWorkspace(ctx context.Context, id string, req *models.UpdateWorkspaceRequest, ownerID string) (*models.Workspace, error)
	DeleteWorkspace(ctx context.Context, id string, ownerID string) error
	
	// 목록 및 검색
	ListWorkspaces(ctx context.Context, ownerID string, req *models.PaginationRequest) (*models.WorkspaceListResponse, error)
	
	// 비즈니스 로직
	ValidateWorkspace(ctx context.Context, workspace *models.Workspace) error
	ActivateWorkspace(ctx context.Context, id string, ownerID string) error
	DeactivateWorkspace(ctx context.Context, id string, ownerID string) error
	ArchiveWorkspace(ctx context.Context, id string, ownerID string) error
	
	// 상태 관리
	UpdateActiveTaskCount(ctx context.Context, id string, delta int) error
	GetWorkspaceStats(ctx context.Context, ownerID string) (*WorkspaceStats, error)
}

// WorkspaceStats는 워크스페이스 통계 정보를 나타냅니다
type WorkspaceStats struct {
	TotalWorkspaces  int `json:"total_workspaces"`
	ActiveWorkspaces int `json:"active_workspaces"`
	ArchivedWorkspaces int `json:"archived_workspaces"`
	TotalActiveTasks int `json:"total_active_tasks"`
}

// ListOptions는 목록 조회를 위한 옵션입니다
type ListOptions struct {
	// 페이지네이션
	Page     int `json:"page" form:"page"`
	PageSize int `json:"page_size" form:"page_size"`
	
	// 정렬
	SortBy    string `json:"sort_by" form:"sort_by"`
	SortOrder string `json:"sort_order" form:"sort_order"`
	
	// 필터링
	Status         models.WorkspaceStatus `json:"status" form:"status"`
	Name           string                 `json:"name" form:"name"`
	ProjectPath    string                 `json:"project_path" form:"project_path"`
	CreatedAfter   *time.Time             `json:"created_after" form:"created_after"`
	CreatedBefore  *time.Time             `json:"created_before" form:"created_before"`
	UpdatedAfter   *time.Time             `json:"updated_after" form:"updated_after"`
	UpdatedBefore  *time.Time             `json:"updated_before" form:"updated_before"`
	AccessedAfter  *time.Time             `json:"accessed_after" form:"accessed_after"`
	AccessedBefore *time.Time             `json:"accessed_before" form:"accessed_before"`
	
	// 검색
	Search string `json:"search" form:"search"`
	
	// 카테고리/태그
	Tags       []string `json:"tags" form:"tags"`
	Category   string   `json:"category" form:"category"`
	
	// 포함 여부
	IncludeArchived bool `json:"include_archived" form:"include_archived"`
	
	// 소유자 필터
	OwnerID string `json:"owner_id" form:"owner_id"`
}