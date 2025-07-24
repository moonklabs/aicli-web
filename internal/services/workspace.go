package services

import (
	"context"
	"time"
	
	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/storage"
	wsinterfaces "github.com/aicli/aicli-web/internal/interfaces"
)

// WorkspaceService 는 interfaces 패키지에서 사용
type WorkspaceService = wsinterfaces.WorkspaceService
type WorkspaceStats = wsinterfaces.WorkspaceStats

// workspaceService는 WorkspaceService 인터페이스의 구현체입니다
type workspaceService struct {
	storage   storage.Storage
	validator *WorkspaceValidator
}

// NewWorkspaceService는 새로운 워크스페이스 서비스를 생성합니다
func NewWorkspaceService(storage storage.Storage) WorkspaceService {
	return &workspaceService{
		storage:   storage,
		validator: NewWorkspaceValidator(),
	}
}

// CreateWorkspace는 새로운 워크스페이스를 생성합니다
func (s *workspaceService) CreateWorkspace(ctx context.Context, req *models.CreateWorkspaceRequest, ownerID string) (*models.Workspace, error) {
	if req == nil {
		return nil, NewWorkspaceError(ErrCodeInvalidRequest, "생성 요청이 nil입니다", ErrInvalidRequest)
	}
	
	if ownerID == "" {
		return nil, NewWorkspaceError(ErrCodeInvalidRequest, "소유자 ID가 필요합니다", ErrInvalidRequest)
	}
	
	// 요청 검증
	if err := s.validator.ValidateCreate(ctx, req); err != nil {
		return nil, err
	}
	
	// 사용자별 워크스페이스 수 제한 확인
	_, count, err := s.storage.Workspace().GetByOwnerID(ctx, ownerID, &models.PaginationRequest{Page: 1, Limit: 1})
	if err != nil {
		return nil, NewWorkspaceError(ErrCodeInvalidRequest, "워크스페이스 수 확인 실패", err)
	}
	
	if err := s.validator.CanCreateWorkspace(ctx, ownerID, count); err != nil {
		return nil, err
	}
	
	// 이름 중복 확인
	exists, err := s.storage.Workspace().ExistsByName(ctx, ownerID, req.Name)
	if err != nil {
		return nil, NewWorkspaceError(ErrCodeInvalidRequest, "이름 중복 확인 실패", err)
	}
	
	if exists {
		return nil, NewWorkspaceError(ErrCodeAlreadyExists, "이미 존재하는 워크스페이스 이름입니다", ErrWorkspaceExists)
	}
	
	// 워크스페이스 객체 생성
	workspace := &models.Workspace{
		Name:        req.Name,
		ProjectPath: req.ProjectPath,
		Status:      models.WorkspaceStatusActive,
		OwnerID:     ownerID,
		ClaudeKey:   req.ClaudeKey,
		ActiveTasks: 0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	
	// 워크스페이스 검증
	if err := s.validator.ValidateWorkspace(ctx, workspace); err != nil {
		return nil, err
	}
	
	// 데이터베이스에 저장
	if err := s.storage.Workspace().Create(ctx, workspace); err != nil {
		if err == storage.ErrAlreadyExists {
			return nil, NewWorkspaceError(ErrCodeAlreadyExists, "이미 존재하는 워크스페이스입니다", ErrWorkspaceExists)
		}
		return nil, NewWorkspaceError(ErrCodeInvalidRequest, "워크스페이스 생성 실패", err)
	}
	
	// API 키 마스킹
	workspace.MaskClaudeKey()
	
	return workspace, nil
}

// GetWorkspace는 특정 워크스페이스를 조회합니다
func (s *workspaceService) GetWorkspace(ctx context.Context, id string, ownerID string) (*models.Workspace, error) {
	if id == "" {
		return nil, NewWorkspaceError(ErrCodeInvalidRequest, "워크스페이스 ID가 필요합니다", ErrInvalidRequest)
	}
	
	if ownerID == "" {
		return nil, NewWorkspaceError(ErrCodeUnauthorized, "소유자 ID가 필요합니다", ErrUnauthorized)
	}
	
	// 워크스페이스 조회
	workspace, err := s.storage.Workspace().GetByID(ctx, id)
	if err != nil {
		if err == storage.ErrNotFound {
			return nil, NewWorkspaceError(ErrCodeNotFound, "워크스페이스를 찾을 수 없습니다", ErrWorkspaceNotFound)
		}
		return nil, NewWorkspaceError(ErrCodeInvalidRequest, "워크스페이스 조회 실패", err)
	}
	
	// 권한 확인
	if workspace.OwnerID != ownerID {
		return nil, NewWorkspaceError(ErrCodeUnauthorized, "워크스페이스에 접근할 권한이 없습니다", ErrUnauthorized)
	}
	
	// API 키 마스킹
	workspace.MaskClaudeKey()
	
	return workspace, nil
}

// UpdateWorkspace는 워크스페이스를 수정합니다
func (s *workspaceService) UpdateWorkspace(ctx context.Context, id string, req *models.UpdateWorkspaceRequest, ownerID string) (*models.Workspace, error) {
	if id == "" {
		return nil, NewWorkspaceError(ErrCodeInvalidRequest, "워크스페이스 ID가 필요합니다", ErrInvalidRequest)
	}
	
	if req == nil {
		return nil, NewWorkspaceError(ErrCodeInvalidRequest, "수정 요청이 nil입니다", ErrInvalidRequest)
	}
	
	if ownerID == "" {
		return nil, NewWorkspaceError(ErrCodeUnauthorized, "소유자 ID가 필요합니다", ErrUnauthorized)
	}
	
	// 요청 검증
	if err := s.validator.ValidateUpdate(ctx, req); err != nil {
		return nil, err
	}
	
	// 기존 워크스페이스 조회 및 권한 확인
	workspace, err := s.GetWorkspace(ctx, id, ownerID)
	if err != nil {
		return nil, err
	}
	
	// 이름 변경 시 중복 확인
	if req.Name != "" && req.Name != workspace.Name {
		exists, err := s.storage.Workspace().ExistsByName(ctx, ownerID, req.Name)
		if err != nil {
			return nil, NewWorkspaceError(ErrCodeInvalidRequest, "이름 중복 확인 실패", err)
		}
		
		if exists {
			return nil, NewWorkspaceError(ErrCodeAlreadyExists, "이미 존재하는 워크스페이스 이름입니다", ErrWorkspaceExists)
		}
	}
	
	// 상태 변경 검증
	if req.Status != "" && req.Status != workspace.Status {
		switch req.Status {
		case models.WorkspaceStatusActive:
			if err := s.validator.CanActivateWorkspace(ctx, workspace); err != nil {
				return nil, err
			}
		case models.WorkspaceStatusInactive:
			if err := s.validator.CanDeactivateWorkspace(ctx, workspace); err != nil {
				return nil, err
			}
		case models.WorkspaceStatusArchived:
			if err := s.validator.CanDeleteWorkspace(ctx, workspace); err != nil {
				return nil, err
			}
		}
	}
	
	// 업데이트할 필드 구성
	updates := make(map[string]interface{})
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.ProjectPath != "" {
		updates["project_path"] = req.ProjectPath
	}
	if req.ClaudeKey != "" {
		updates["claude_key"] = req.ClaudeKey
	}
	if req.Status != "" {
		updates["status"] = req.Status
	}
	updates["updated_at"] = time.Now()
	
	// 데이터베이스 업데이트
	if err := s.storage.Workspace().Update(ctx, id, updates); err != nil {
		if err == storage.ErrNotFound {
			return nil, NewWorkspaceError(ErrCodeNotFound, "워크스페이스를 찾을 수 없습니다", ErrWorkspaceNotFound)
		}
		if err == storage.ErrAlreadyExists {
			return nil, NewWorkspaceError(ErrCodeAlreadyExists, "이미 존재하는 워크스페이스 이름입니다", ErrWorkspaceExists)
		}
		return nil, NewWorkspaceError(ErrCodeInvalidRequest, "워크스페이스 수정 실패", err)
	}
	
	// 수정된 워크스페이스 조회
	updatedWorkspace, err := s.GetWorkspace(ctx, id, ownerID)
	if err != nil {
		return nil, err
	}
	
	return updatedWorkspace, nil
}

// DeleteWorkspace는 워크스페이스를 삭제합니다
func (s *workspaceService) DeleteWorkspace(ctx context.Context, id string, ownerID string) error {
	if id == "" {
		return NewWorkspaceError(ErrCodeInvalidRequest, "워크스페이스 ID가 필요합니다", ErrInvalidRequest)
	}
	
	if ownerID == "" {
		return NewWorkspaceError(ErrCodeUnauthorized, "소유자 ID가 필요합니다", ErrUnauthorized)
	}
	
	// 워크스페이스 존재 및 권한 확인
	workspace, err := s.GetWorkspace(ctx, id, ownerID)
	if err != nil {
		return err
	}
	
	// 삭제 가능 여부 확인
	if err := s.validator.CanDeleteWorkspace(ctx, workspace); err != nil {
		return err
	}
	
	// 데이터베이스에서 삭제 (Soft Delete)
	if err := s.storage.Workspace().Delete(ctx, id); err != nil {
		if err == storage.ErrNotFound {
			return NewWorkspaceError(ErrCodeNotFound, "워크스페이스를 찾을 수 없습니다", ErrWorkspaceNotFound)
		}
		return NewWorkspaceError(ErrCodeInvalidRequest, "워크스페이스 삭제 실패", err)
	}
	
	return nil
}

// ListWorkspaces는 워크스페이스 목록을 조회합니다
func (s *workspaceService) ListWorkspaces(ctx context.Context, ownerID string, req *models.PaginationRequest) (*models.WorkspaceListResponse, error) {
	if ownerID == "" {
		return nil, NewWorkspaceError(ErrCodeUnauthorized, "소유자 ID가 필요합니다", ErrUnauthorized)
	}
	
	if req == nil {
		req = &models.PaginationRequest{
			Page:  1,
			Limit: 10,
			Sort:  "created_at",
			Order: "desc",
		}
	}
	
	// 기본값 설정
	if req.Page < 1 {
		req.Page = 1
	}
	if req.Limit < 1 {
		req.Limit = 10
	}
	if req.Sort == "" {
		req.Sort = "created_at"
	}
	if req.Order == "" {
		req.Order = "desc"
	}
	
	// 워크스페이스 목록 조회
	workspaces, total, err := s.storage.Workspace().GetByOwnerID(ctx, ownerID, req)
	if err != nil {
		return nil, NewWorkspaceError(ErrCodeInvalidRequest, "워크스페이스 목록 조회 실패", err)
	}
	
	// API 키 마스킹
	for _, workspace := range workspaces {
		workspace.MaskClaudeKey()
	}
	
	// 응답 생성
	response := &models.WorkspaceListResponse{
		Success: true,
		Data:    make([]models.Workspace, len(workspaces)),
		Meta: models.PaginationMeta{
			CurrentPage: req.Page,
			PerPage:     req.Limit,
			Total:       total,
			TotalPages:  (total + req.Limit - 1) / req.Limit,
		},
	}
	
	// 포인터를 값으로 변환
	for i, workspace := range workspaces {
		response.Data[i] = *workspace
	}
	
	return response, nil
}

// ValidateWorkspace는 워크스페이스를 검증합니다
func (s *workspaceService) ValidateWorkspace(ctx context.Context, workspace *models.Workspace) error {
	return s.validator.ValidateWorkspace(ctx, workspace)
}

// ActivateWorkspace는 워크스페이스를 활성화합니다
func (s *workspaceService) ActivateWorkspace(ctx context.Context, id string, ownerID string) error {
	req := &models.UpdateWorkspaceRequest{
		Status: models.WorkspaceStatusActive,
	}
	
	_, err := s.UpdateWorkspace(ctx, id, req, ownerID)
	return err
}

// DeactivateWorkspace는 워크스페이스를 비활성화합니다
func (s *workspaceService) DeactivateWorkspace(ctx context.Context, id string, ownerID string) error {
	req := &models.UpdateWorkspaceRequest{
		Status: models.WorkspaceStatusInactive,
	}
	
	_, err := s.UpdateWorkspace(ctx, id, req, ownerID)
	return err
}

// ArchiveWorkspace는 워크스페이스를 아카이브합니다
func (s *workspaceService) ArchiveWorkspace(ctx context.Context, id string, ownerID string) error {
	req := &models.UpdateWorkspaceRequest{
		Status: models.WorkspaceStatusArchived,
	}
	
	_, err := s.UpdateWorkspace(ctx, id, req, ownerID)
	return err
}

// UpdateActiveTaskCount는 활성 태스크 수를 업데이트합니다
func (s *workspaceService) UpdateActiveTaskCount(ctx context.Context, id string, delta int) error {
	if id == "" {
		return NewWorkspaceError(ErrCodeInvalidRequest, "워크스페이스 ID가 필요합니다", ErrInvalidRequest)
	}
	
	// 현재 워크스페이스 조회
	workspace, err := s.storage.Workspace().GetByID(ctx, id)
	if err != nil {
		if err == storage.ErrNotFound {
			return NewWorkspaceError(ErrCodeNotFound, "워크스페이스를 찾을 수 없습니다", ErrWorkspaceNotFound)
		}
		return NewWorkspaceError(ErrCodeInvalidRequest, "워크스페이스 조회 실패", err)
	}
	
	// 새로운 활성 태스크 수 계산
	newCount := workspace.ActiveTasks + delta
	if newCount < 0 {
		newCount = 0
	}
	
	// 업데이트
	updates := map[string]interface{}{
		"active_tasks": newCount,
		"updated_at":   time.Now(),
	}
	
	if err := s.storage.Workspace().Update(ctx, id, updates); err != nil {
		return NewWorkspaceError(ErrCodeInvalidRequest, "활성 태스크 수 업데이트 실패", err)
	}
	
	return nil
}

// GetWorkspaceStats는 워크스페이스 통계를 조회합니다
func (s *workspaceService) GetWorkspaceStats(ctx context.Context, ownerID string) (*WorkspaceStats, error) {
	if ownerID == "" {
		return nil, NewWorkspaceError(ErrCodeUnauthorized, "소유자 ID가 필요합니다", ErrUnauthorized)
	}
	
	// 전체 워크스페이스 목록 조회
	workspaces, total, err := s.storage.Workspace().GetByOwnerID(ctx, ownerID, &models.PaginationRequest{
		Page:  1,
		Limit: 1000, // 충분히 큰 값으로 설정
	})
	if err != nil {
		return nil, NewWorkspaceError(ErrCodeInvalidRequest, "워크스페이스 통계 조회 실패", err)
	}
	
	stats := &WorkspaceStats{
		TotalWorkspaces: total,
	}
	
	// 상태별 카운트 및 활성 태스크 수 집계
	for _, workspace := range workspaces {
		switch workspace.Status {
		case models.WorkspaceStatusActive:
			stats.ActiveWorkspaces++
		case models.WorkspaceStatusArchived:
			stats.ArchivedWorkspaces++
		}
		stats.TotalActiveTasks += workspace.ActiveTasks
	}
	
	return stats, nil
}