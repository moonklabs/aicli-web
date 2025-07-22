package services

import (
	"context"
	"fmt"

	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/storage"
)

// ProjectService 프로젝트 비즈니스 로직 처리
type ProjectService struct {
	storage    storage.Storage
	gitService *GitService
}

// NewProjectService 새 프로젝트 서비스 생성
func NewProjectService(storage storage.Storage) *ProjectService {
	return &ProjectService{
		storage:    storage,
		gitService: NewGitService(),
	}
}

// CreateProject 새 프로젝트 생성
func (s *ProjectService) CreateProject(ctx context.Context, project *models.Project) error {
	// 워크스페이스 존재 확인
	workspace, err := s.storage.Workspace().GetByID(ctx, project.WorkspaceID)
	if err != nil {
		if err == storage.ErrNotFound {
			return fmt.Errorf("workspace not found")
		}
		return err
	}

	// 워크스페이스가 활성 상태인지 확인
	if !workspace.IsValid() {
		return fmt.Errorf("workspace is not active")
	}

	// 프로젝트 이름 중복 확인
	exists, err := s.storage.Project().ExistsByName(ctx, project.WorkspaceID, project.Name)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("project name already exists in workspace")
	}

	// 경로 중복 확인
	existingProject, _ := s.storage.Project().GetByPath(ctx, project.Path)
	if existingProject != nil {
		return fmt.Errorf("project path already in use")
	}

	// Git 정보 수집 (있는 경우)
	gitInfo, err := s.gitService.GetGitInfo(project.Path)
	if err == nil && gitInfo != nil {
		project.GitInfo = gitInfo
		if gitInfo.RemoteURL != "" {
			project.GitURL = gitInfo.RemoteURL
		}
		if gitInfo.CurrentBranch != "" {
			project.GitBranch = gitInfo.CurrentBranch
		}
	}

	// 프로젝트 생성
	return s.storage.Project().Create(ctx, project)
}

// GetProject ID로 프로젝트 조회
func (s *ProjectService) GetProject(ctx context.Context, id string) (*models.Project, error) {
	project, err := s.storage.Project().GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Git 정보 업데이트
	gitInfo, err := s.gitService.GetGitInfo(project.Path)
	if err == nil && gitInfo != nil {
		project.GitInfo = gitInfo
	}

	return project, nil
}

// GetProjectsByWorkspace 워크스페이스의 프로젝트 목록 조회
func (s *ProjectService) GetProjectsByWorkspace(ctx context.Context, workspaceID string, pagination *models.PaginationRequest) ([]*models.Project, int, error) {
	// 워크스페이스 존재 확인
	_, err := s.storage.Workspace().GetByID(ctx, workspaceID)
	if err != nil {
		if err == storage.ErrNotFound {
			return nil, 0, fmt.Errorf("workspace not found")
		}
		return nil, 0, err
	}

	// 프로젝트 목록 조회
	projects, total, err := s.storage.Project().GetByWorkspaceID(ctx, workspaceID, pagination)
	if err != nil {
		return nil, 0, err
	}

	// 각 프로젝트의 Git 정보 업데이트 (간단한 정보만)
	for _, project := range projects {
		gitInfo, err := s.gitService.GetGitInfo(project.Path)
		if err == nil && gitInfo != nil {
			// 목록에서는 기본 정보만 포함
			project.GitInfo = &models.GitInfo{
				CurrentBranch: gitInfo.CurrentBranch,
				IsClean:       gitInfo.IsClean,
			}
		}
	}

	return projects, total, nil
}

// UpdateProject 프로젝트 업데이트
func (s *ProjectService) UpdateProject(ctx context.Context, id string, updates map[string]interface{}) error {
	// 프로젝트 존재 확인
	project, err := s.storage.Project().GetByID(ctx, id)
	if err != nil {
		return err
	}

	// 이름 변경 시 중복 확인
	if name, ok := updates["name"].(string); ok && name != project.Name {
		exists, err := s.storage.Project().ExistsByName(ctx, project.WorkspaceID, name)
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("project name already exists in workspace")
		}
	}

	// 경로 변경 시 중복 확인
	if path, ok := updates["path"].(string); ok && path != project.Path {
		existingProject, _ := s.storage.Project().GetByPath(ctx, path)
		if existingProject != nil && existingProject.ID != id {
			return fmt.Errorf("project path already in use")
		}
	}

	// 프로젝트 업데이트
	return s.storage.Project().Update(ctx, id, updates)
}

// DeleteProject 프로젝트 삭제
func (s *ProjectService) DeleteProject(ctx context.Context, id string) error {
	// 프로젝트 존재 확인
	project, err := s.storage.Project().GetByID(ctx, id)
	if err != nil {
		return err
	}

	// TODO: 실행 중인 태스크 확인 및 정리

	// 프로젝트가 활성 상태인 경우 경고
	if project.Status == models.ProjectStatusActive {
		// 상태를 inactive로 변경
		updates := map[string]interface{}{
			"status": models.ProjectStatusInactive,
		}
		s.storage.Project().Update(ctx, id, updates)
	}

	// 프로젝트 삭제 (soft delete)
	return s.storage.Project().Delete(ctx, id)
}

// GetProjectConfig 프로젝트 설정 조회
func (s *ProjectService) GetProjectConfig(ctx context.Context, projectID string) (*models.ProjectConfig, error) {
	project, err := s.storage.Project().GetByID(ctx, projectID)
	if err != nil {
		return nil, err
	}

	// 보안상 API 키는 마스킹 처리
	config := project.Config
	if config.EncryptedAPIKey != "" {
		config.ClaudeAPIKey = "****" // 마스킹
	}

	return &config, nil
}

// UpdateProjectConfig 프로젝트 설정 업데이트
func (s *ProjectService) UpdateProjectConfig(ctx context.Context, projectID string, config models.ProjectConfig) error {
	// 프로젝트 존재 확인
	_, err := s.storage.Project().GetByID(ctx, projectID)
	if err != nil {
		return err
	}

	// API 키가 제공된 경우 암호화 (실제 구현에서는 암호화 필요)
	if config.ClaudeAPIKey != "" && config.ClaudeAPIKey != "****" {
		// TODO: 실제 암호화 구현
		config.EncryptedAPIKey = "encrypted_" + config.ClaudeAPIKey
		config.ClaudeAPIKey = "" // 평문은 저장하지 않음
	}

	// 설정 업데이트
	updates := map[string]interface{}{
		"config": config,
	}

	return s.storage.Project().Update(ctx, projectID, updates)
}

// GetByID 프로젝트 ID로 조회 (별칭)
func (s *ProjectService) GetByID(ctx context.Context, id string) (*models.Project, error) {
	return s.GetProject(ctx, id)
}