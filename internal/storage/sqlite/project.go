package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/storage"
)

// projectStorage 프로젝트 스토리지 SQLite 구현
type projectStorage struct {
	storage *Storage
}

// newProjectStorage 새 프로젝트 스토리지 생성
func newProjectStorage(s *Storage) *projectStorage {
	return &projectStorage{storage: s}
}

const (
	// 프로젝트 조회 쿼리
	selectProjectQuery = `
		SELECT id, workspace_id, name, path, description, git_url, git_branch,
		       language, status, config, git_info, created_at, updated_at, deleted_at
		FROM projects
	`
	
	// 프로젝트 삽입 쿼리
	insertProjectQuery = `
		INSERT INTO projects (id, workspace_id, name, path, description, git_url, git_branch,
		                     language, status, config, git_info, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	
	// 프로젝트 업데이트 쿼리 베이스
	updateProjectQueryBase = `UPDATE projects SET updated_at = ? `
	
	// 프로젝트 삭제 쿼리 (soft delete)
	deleteProjectQuery = `
		UPDATE projects 
		SET deleted_at = ?, updated_at = ?
		WHERE id = ? AND deleted_at IS NULL
	`
	
	// 프로젝트 존재 여부 확인 쿼리
	existsProjectByNameQuery = `
		SELECT 1 FROM projects 
		WHERE workspace_id = ? AND name = ? AND deleted_at IS NULL
	`
	
	// 경로로 프로젝트 조회 쿼리
	selectProjectByPathQuery = selectProjectQuery + ` WHERE path = ? AND deleted_at IS NULL`
	
	// 카운트 쿼리
	countProjectsQuery = `SELECT COUNT(*) FROM projects`
)

// Create 새 프로젝트 생성
func (p *projectStorage) Create(ctx context.Context, project *models.Project) error {
	now := time.Now()
	
	// 기본값 설정
	if project.Status == "" {
		project.Status = models.ProjectStatusActive
	}
	project.CreatedAt = now
	project.UpdatedAt = now
	
	// JSON 필드들 직렬화
	configJSON, err := p.marshalConfig(project.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	gitInfoJSON, err := p.marshalGitInfo(project.GitInfo)
	if err != nil {
		return fmt.Errorf("failed to marshal git info: %w", err)
	}
	
	// 프로젝트 삽입
	_, err = p.storage.execContext(ctx, insertProjectQuery,
		project.ID,
		project.WorkspaceID,
		project.Name,
		project.Path,
		project.Description,
		project.GitURL,
		project.GitBranch,
		project.Language,
		project.Status,
		configJSON,
		gitInfoJSON,
		project.CreatedAt,
		project.UpdatedAt,
	)
	
	if err != nil {
		return storage.ConvertError(err, "create project", "sqlite")
	}
	
	return nil
}

// GetByID ID로 프로젝트 조회
func (p *projectStorage) GetByID(ctx context.Context, id string) (*models.Project, error) {
	query := selectProjectQuery + ` WHERE id = ? AND deleted_at IS NULL`
	
	row := p.storage.queryRowContext(ctx, query, id)
	
	project, err := p.scanProject(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrNotFound
		}
		return nil, storage.ConvertError(err, "get project by id", "sqlite")
	}
	
	return project, nil
}

// GetByWorkspaceID 워크스페이스 ID로 프로젝트 목록 조회
func (p *projectStorage) GetByWorkspaceID(ctx context.Context, workspaceID string, pagination *models.PaginationRequest) ([]*models.Project, int, error) {
	if pagination == nil {
		pagination = &models.PaginationRequest{Page: 1, Limit: 20, Sort: "created_at", Order: "desc"}
	}
	pagination.Normalize()
	
	// 전체 카운트 조회
	countQuery := countProjectsQuery + ` WHERE workspace_id = ? AND deleted_at IS NULL`
	var total int
	err := p.storage.queryRowContext(ctx, countQuery, workspaceID).Scan(&total)
	if err != nil {
		return nil, 0, storage.ConvertError(err, "count projects by workspace", "sqlite")
	}
	
	// 데이터 조회
	query := selectProjectQuery + `
		WHERE workspace_id = ? AND deleted_at IS NULL 
		ORDER BY ` + pagination.Sort + ` ` + pagination.Order + `
		LIMIT ? OFFSET ?
	`
	
	rows, err := p.storage.queryContext(ctx, query, 
		workspaceID, pagination.Limit, pagination.GetOffset())
	if err != nil {
		return nil, 0, storage.ConvertError(err, "get projects by workspace", "sqlite")
	}
	defer rows.Close()
	
	projects, err := p.scanProjects(rows)
	if err != nil {
		return nil, 0, err
	}
	
	return projects, total, nil
}

// Update 프로젝트 업데이트
func (p *projectStorage) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return storage.ErrInvalidInput
	}
	
	// 허용된 업데이트 필드들
	allowedFields := map[string]bool{
		"name":         true,
		"path":         true,
		"description":  true,
		"git_url":      true,
		"git_branch":   true,
		"language":     true,
		"status":       true,
		"config":       true,
		"git_info":     true,
	}
	
	// 동적 UPDATE 쿼리 생성
	var setParts []string
	var args []interface{}
	
	// updated_at은 항상 업데이트
	setParts = append(setParts, "updated_at = ?")
	args = append(args, time.Now())
	
	// 업데이트할 필드들 처리
	for field, value := range updates {
		if !allowedFields[field] {
			return fmt.Errorf("field '%s' is not allowed for update", field)
		}
		
		// JSON 필드들은 직렬화 필요
		var processedValue interface{}
		var err error
		
		switch field {
		case "config":
			if config, ok := value.(models.ProjectConfig); ok {
				processedValue, err = p.marshalConfig(config)
				if err != nil {
					return fmt.Errorf("failed to marshal config: %w", err)
				}
			} else {
				processedValue = value
			}
		case "git_info":
			if gitInfo, ok := value.(*models.GitInfo); ok {
				processedValue, err = p.marshalGitInfo(gitInfo)
				if err != nil {
					return fmt.Errorf("failed to marshal git info: %w", err)
				}
			} else {
				processedValue = value
			}
		default:
			processedValue = value
		}
		
		setParts = append(setParts, field+" = ?")
		args = append(args, processedValue)
	}
	
	// 최종 쿼리 생성
	query := updateProjectQueryBase + strings.Join(setParts, ", ") + 
		" WHERE id = ? AND deleted_at IS NULL"
	args = append(args, id)
	
	result, err := p.storage.execContext(ctx, query, args...)
	if err != nil {
		return storage.ConvertError(err, "update project", "sqlite")
	}
	
	// 업데이트된 행이 있는지 확인
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return storage.ConvertError(err, "check rows affected", "sqlite")
	}
	
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}
	
	return nil
}

// Delete 프로젝트 삭제 (soft delete)
func (p *projectStorage) Delete(ctx context.Context, id string) error {
	now := time.Now()
	
	result, err := p.storage.execContext(ctx, deleteProjectQuery, now, now, id)
	if err != nil {
		return storage.ConvertError(err, "delete project", "sqlite")
	}
	
	// 삭제된 행이 있는지 확인
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return storage.ConvertError(err, "check rows affected", "sqlite")
	}
	
	if rowsAffected == 0 {
		return storage.ErrNotFound
	}
	
	return nil
}

// ExistsByName 워크스페이스 내 이름으로 존재 여부 확인
func (p *projectStorage) ExistsByName(ctx context.Context, workspaceID, name string) (bool, error) {
	var exists int
	err := p.storage.queryRowContext(ctx, existsProjectByNameQuery, workspaceID, name).Scan(&exists)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, storage.ConvertError(err, "check project exists by name", "sqlite")
	}
	
	return exists == 1, nil
}

// GetByPath 경로로 프로젝트 조회
func (p *projectStorage) GetByPath(ctx context.Context, path string) (*models.Project, error) {
	row := p.storage.queryRowContext(ctx, selectProjectByPathQuery, path)
	
	project, err := p.scanProject(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrNotFound
		}
		return nil, storage.ConvertError(err, "get project by path", "sqlite")
	}
	
	return project, nil
}

// marshalConfig ProjectConfig를 JSON으로 직렬화
func (p *projectStorage) marshalConfig(config models.ProjectConfig) (string, error) {
	// 빈 설정인지 확인
	if config.ClaudeAPIKey == "" && 
		config.EncryptedAPIKey == "" && 
		len(config.Environment) == 0 && 
		len(config.BuildCommands) == 0 && 
		len(config.TestCommands) == 0 &&
		config.ClaudeOptions.Model == "" &&
		config.ClaudeOptions.MaxTokens == 0 &&
		config.ClaudeOptions.Temperature == 0 &&
		config.ClaudeOptions.SystemPrompt == "" &&
		len(config.ClaudeOptions.ExcludePaths) == 0 &&
		len(config.ClaudeOptions.IncludePaths) == 0 {
		return "{}", nil
	}
	
	data, err := json.Marshal(config)
	if err != nil {
		return "", err
	}
	
	return string(data), nil
}

// unmarshalConfig JSON을 ProjectConfig로 역직렬화
func (p *projectStorage) unmarshalConfig(data string) (models.ProjectConfig, error) {
	var config models.ProjectConfig
	
	if data == "" || data == "null" {
		return config, nil
	}
	
	err := json.Unmarshal([]byte(data), &config)
	return config, err
}

// marshalGitInfo GitInfo를 JSON으로 직렬화
func (p *projectStorage) marshalGitInfo(gitInfo *models.GitInfo) (string, error) {
	if gitInfo == nil {
		return "null", nil
	}
	
	data, err := json.Marshal(gitInfo)
	if err != nil {
		return "", err
	}
	
	return string(data), nil
}

// unmarshalGitInfo JSON을 GitInfo로 역직렬화
func (p *projectStorage) unmarshalGitInfo(data string) (*models.GitInfo, error) {
	if data == "" || data == "null" {
		return nil, nil
	}
	
	var gitInfo models.GitInfo
	err := json.Unmarshal([]byte(data), &gitInfo)
	if err != nil {
		return nil, err
	}
	
	return &gitInfo, nil
}

// scanProject 단일 프로젝트 스캔
func (p *projectStorage) scanProject(row *sql.Row) (*models.Project, error) {
	project := &models.Project{}
	var deletedAt sql.NullTime
	var description, gitURL, gitBranch, language sql.NullString
	var configJSON, gitInfoJSON string
	
	err := row.Scan(
		&project.ID,
		&project.WorkspaceID,
		&project.Name,
		&project.Path,
		&description,
		&gitURL,
		&gitBranch,
		&language,
		&project.Status,
		&configJSON,
		&gitInfoJSON,
		&project.CreatedAt,
		&project.UpdatedAt,
		&deletedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	// NULL 값 처리
	if description.Valid {
		project.Description = description.String
	}
	if gitURL.Valid {
		project.GitURL = gitURL.String
	}
	if gitBranch.Valid {
		project.GitBranch = gitBranch.String
	}
	if language.Valid {
		project.Language = language.String
	}
	if deletedAt.Valid {
		project.DeletedAt = &deletedAt.Time
	}
	
	// JSON 필드들 역직렬화
	project.Config, err = p.unmarshalConfig(configJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	
	project.GitInfo, err = p.unmarshalGitInfo(gitInfoJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal git info: %w", err)
	}
	
	return project, nil
}

// scanProjects 여러 프로젝트 스캔
func (p *projectStorage) scanProjects(rows *sql.Rows) ([]*models.Project, error) {
	var projects []*models.Project
	
	for rows.Next() {
		project := &models.Project{}
		var deletedAt sql.NullTime
		var description, gitURL, gitBranch, language sql.NullString
		var configJSON, gitInfoJSON string
		
		err := rows.Scan(
			&project.ID,
			&project.WorkspaceID,
			&project.Name,
			&project.Path,
			&description,
			&gitURL,
			&gitBranch,
			&language,
			&project.Status,
			&configJSON,
			&gitInfoJSON,
			&project.CreatedAt,
			&project.UpdatedAt,
			&deletedAt,
		)
		
		if err != nil {
			return nil, storage.ConvertError(err, "scan project row", "sqlite")
		}
		
		// NULL 값 처리
		if description.Valid {
			project.Description = description.String
		}
		if gitURL.Valid {
			project.GitURL = gitURL.String
		}
		if gitBranch.Valid {
			project.GitBranch = gitBranch.String
		}
		if language.Valid {
			project.Language = language.String
		}
		if deletedAt.Valid {
			project.DeletedAt = &deletedAt.Time
		}
		
		// JSON 필드들 역직렬화
		project.Config, err = p.unmarshalConfig(configJSON)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal config: %w", err)
		}
		
		project.GitInfo, err = p.unmarshalGitInfo(gitInfoJSON)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal git info: %w", err)
		}
		
		projects = append(projects, project)
	}
	
	// rows 순회 중 에러 확인
	if err := rows.Err(); err != nil {
		return nil, storage.ConvertError(err, "scan project rows", "sqlite")
	}
	
	return projects, nil
}