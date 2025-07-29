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
	"github.com/google/uuid"
)

// agentStorage 에이전트 스토리지 SQLite 구현
type agentStorage struct {
	storage *Storage
}

// newAgentStorage 새 에이전트 스토리지 생성
func newAgentStorage(s *Storage) *agentStorage {
	return &agentStorage{storage: s}
}

const (
	// 에이전트 조회 쿼리
	selectAgentQuery = `
		SELECT id, project_id, name, type, status, worktree_id, container_id, session_id,
		       config, description, last_activity, error_message, version,
		       created_at, updated_at, deleted_at
		FROM agents
	`

	// 에이전트 삽입 쿼리
	insertAgentQuery = `
		INSERT INTO agents (id, project_id, name, type, status, worktree_id, container_id, session_id,
		                   config, description, last_activity, error_message, version,
		                   created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	// 에이전트 업데이트 쿼리 베이스
	updateAgentQueryBase = `UPDATE agents SET `

	// 에이전트 삭제 쿼리 (soft delete)
	deleteAgentQuery = `
		UPDATE agents 
		SET deleted_at = ?, updated_at = ?
		WHERE id = ? AND deleted_at IS NULL
	`

	// 에이전트 존재 여부 확인 쿼리
	existsAgentByNameQuery = `
		SELECT EXISTS(
			SELECT 1 FROM agents 
			WHERE project_id = ? AND name = ? AND deleted_at IS NULL
		)
	`

	// 프로젝트별 에이전트 개수 조회 쿼리
	countAgentsByProjectQuery = `
		SELECT COUNT(*) FROM agents 
		WHERE project_id = ? AND deleted_at IS NULL
	`

	// 활성 에이전트 개수 조회 쿼리
	countActiveAgentsQuery = `
		SELECT COUNT(*) FROM agents 
		WHERE project_id = ? AND status = ? AND deleted_at IS NULL
	`

	// 에이전트 상태 업데이트 쿼리
	updateAgentStatusQuery = `
		UPDATE agents 
		SET status = ?, error_message = ?, last_activity = ?, updated_at = ?
		WHERE id = ? AND deleted_at IS NULL
	`

	// 에이전트 활동 시간 업데이트 쿼리
	updateAgentActivityQuery = `
		UPDATE agents 
		SET last_activity = ?, updated_at = ?
		WHERE id = ? AND deleted_at IS NULL
	`
)

// Create 새 에이전트 생성
func (a *agentStorage) Create(ctx context.Context, agent *models.Agent) error {
	if agent.ID == "" {
		agent.ID = uuid.New().String()
	}

	now := time.Now()
	agent.CreatedAt = now
	agent.UpdatedAt = now
	agent.LastActivity = now

	// AgentConfig JSON 직렬화
	configJSON, err := json.Marshal(agent.Config)
	if err != nil {
		return fmt.Errorf("config 직렬화 실패: %w", err)
	}

	_, err = a.storage.db.ExecContext(ctx, insertAgentQuery,
		agent.ID, agent.ProjectID, agent.Name, string(agent.Type), string(agent.Status),
		agent.WorktreeID, agent.ContainerID, agent.SessionID,
		string(configJSON), agent.Description, agent.LastActivity, agent.ErrorMessage, agent.Version,
		agent.CreatedAt, agent.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("에이전트 생성 실패: %w", err)
	}

	return nil
}

// GetByID ID로 에이전트 조회
func (a *agentStorage) GetByID(ctx context.Context, id string) (*models.Agent, error) {
	query := selectAgentQuery + " WHERE id = ? AND deleted_at IS NULL"
	
	agent, err := a.scanAgent(ctx, query, id)
	if err == sql.ErrNoRows {
		return nil, storage.ErrNotFound
	}
	
	return agent, err
}

// GetByProjectID 프로젝트 ID로 에이전트 목록 조회 (페이지네이션 없음)
func (a *agentStorage) GetByProjectID(ctx context.Context, projectID string) ([]*models.Agent, error) {
	query := selectAgentQuery + " WHERE project_id = ? AND deleted_at IS NULL ORDER BY created_at DESC"
	return a.scanAgents(ctx, query, projectID)
}

// GetByProjectIDWithPagination 프로젝트 ID로 에이전트 목록 조회 (페이지네이션 포함)
func (a *agentStorage) GetByProjectIDWithPagination(ctx context.Context, projectID string, pagination *models.PaginationRequest) ([]*models.Agent, int, error) {
	// 전체 개수 조회
	var totalCount int
	err := a.storage.db.QueryRowContext(ctx, countAgentsByProjectQuery, projectID).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("에이전트 개수 조회 실패: %w", err)
	}

	// 페이지네이션 적용
	query := selectAgentQuery + " WHERE project_id = ? AND deleted_at IS NULL ORDER BY created_at DESC"
	if pagination != nil {
		query += fmt.Sprintf(" LIMIT %d OFFSET %d", pagination.Limit, pagination.GetOffset())
	}

	agents, err := a.scanAgents(ctx, query, projectID)
	if err != nil {
		return nil, 0, err
	}

	return agents, totalCount, nil
}

// Update 에이전트 업데이트
func (a *agentStorage) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}

	// updated_at 자동 추가
	updates["updated_at"] = time.Now()

	// config가 있으면 JSON 직렬화
	if config, exists := updates["config"]; exists {
		configJSON, err := json.Marshal(config)
		if err != nil {
			return fmt.Errorf("config 직렬화 실패: %w", err)
		}
		updates["config"] = string(configJSON)
	}

	setParts := make([]string, 0, len(updates))
	args := make([]interface{}, 0, len(updates)+1)

	for key, value := range updates {
		setParts = append(setParts, key+" = ?")
		args = append(args, value)
	}
	args = append(args, id)

	query := updateAgentQueryBase + strings.Join(setParts, ", ") + " WHERE id = ? AND deleted_at IS NULL"

	result, err := a.storage.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("에이전트 업데이트 실패: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("영향받은 행 수 확인 실패: %w", err)
	}

	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// Delete 에이전트 삭제 (soft delete)
func (a *agentStorage) Delete(ctx context.Context, id string) error {
	now := time.Now()
	result, err := a.storage.db.ExecContext(ctx, deleteAgentQuery, now, now, id)
	if err != nil {
		return fmt.Errorf("에이전트 삭제 실패: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("영향받은 행 수 확인 실패: %w", err)
	}

	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// ExistsByName 프로젝트 내 이름으로 존재 여부 확인
func (a *agentStorage) ExistsByName(ctx context.Context, projectID, name string) (bool, error) {
	var exists bool
	err := a.storage.db.QueryRowContext(ctx, existsAgentByNameQuery, projectID, name).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("에이전트 존재 여부 확인 실패: %w", err)
	}
	return exists, nil
}

// GetByStatus 상태별 에이전트 목록 조회 (페이지네이션 없음)
func (a *agentStorage) GetByStatus(ctx context.Context, status models.AgentStatus) ([]*models.Agent, error) {
	query := selectAgentQuery + " WHERE status = ? AND deleted_at IS NULL ORDER BY last_activity DESC"
	return a.scanAgents(ctx, query, string(status))
}

// GetByStatusWithPagination 상태별 에이전트 목록 조회 (페이지네이션 포함)
func (a *agentStorage) GetByStatusWithPagination(ctx context.Context, status models.AgentStatus, pagination *models.PaginationRequest) ([]*models.Agent, int, error) {
	// 전체 개수 조회
	countQuery := `SELECT COUNT(*) FROM agents WHERE status = ? AND deleted_at IS NULL`
	var totalCount int
	err := a.storage.db.QueryRowContext(ctx, countQuery, string(status)).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("상태별 에이전트 개수 조회 실패: %w", err)
	}

	// 페이지네이션 적용
	query := selectAgentQuery + " WHERE status = ? AND deleted_at IS NULL ORDER BY last_activity DESC"
	if pagination != nil {
		query += fmt.Sprintf(" LIMIT %d OFFSET %d", pagination.Limit, pagination.GetOffset())
	}

	agents, err := a.scanAgents(ctx, query, string(status))
	if err != nil {
		return nil, 0, err
	}

	return agents, totalCount, nil
}

// GetActiveCount 활성 에이전트 수 조회
func (a *agentStorage) GetActiveCount(ctx context.Context, projectID string) (int64, error) {
	var count int64
	err := a.storage.db.QueryRowContext(ctx, countActiveAgentsQuery, projectID, string(models.AgentStatusRunning)).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("활성 에이전트 개수 조회 실패: %w", err)
	}
	return count, nil
}

// UpdateStatus 에이전트 상태 업데이트
func (a *agentStorage) UpdateStatus(ctx context.Context, id string, status models.AgentStatus, errorMessage string) error {
	now := time.Now()
	result, err := a.storage.db.ExecContext(ctx, updateAgentStatusQuery, string(status), errorMessage, now, now, id)
	if err != nil {
		return fmt.Errorf("에이전트 상태 업데이트 실패: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("영향받은 행 수 확인 실패: %w", err)
	}

	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// UpdateActivity 마지막 활동 시간 업데이트
func (a *agentStorage) UpdateActivity(ctx context.Context, id string) error {
	now := time.Now()
	result, err := a.storage.db.ExecContext(ctx, updateAgentActivityQuery, now, now, id)
	if err != nil {
		return fmt.Errorf("에이전트 활동 시간 업데이트 실패: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("영향받은 행 수 확인 실패: %w", err)
	}

	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// GetByContainerID 컨테이너 ID로 에이전트 조회
func (a *agentStorage) GetByContainerID(ctx context.Context, containerID string) (*models.Agent, error) {
	query := selectAgentQuery + " WHERE container_id = ? AND deleted_at IS NULL"
	
	agent, err := a.scanAgent(ctx, query, containerID)
	if err == sql.ErrNoRows {
		return nil, storage.ErrNotFound
	}
	
	return agent, err
}

// GetAll 모든 에이전트 목록 조회
func (a *agentStorage) GetAll(ctx context.Context) ([]*models.Agent, error) {
	query := selectAgentQuery + " WHERE deleted_at IS NULL ORDER BY created_at DESC"
	return a.scanAgents(ctx, query)
}

// scanAgent 단일 에이전트 스캔
func (a *agentStorage) scanAgent(ctx context.Context, query string, args ...interface{}) (*models.Agent, error) {
	row := a.storage.db.QueryRowContext(ctx, query, args...)
	return a.scanAgentRow(row)
}

// scanAgents 복수 에이전트 스캔
func (a *agentStorage) scanAgents(ctx context.Context, query string, args ...interface{}) ([]*models.Agent, error) {
	rows, err := a.storage.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("에이전트 조회 실패: %w", err)
	}
	defer rows.Close()

	var agents []*models.Agent
	for rows.Next() {
		agent, err := a.scanAgentRow(rows)
		if err != nil {
			return nil, err
		}
		agents = append(agents, agent)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("에이전트 행 처리 실패: %w", err)
	}

	return agents, nil
}

// scanAgentRow 에이전트 행 스캔
func (a *agentStorage) scanAgentRow(scanner interface{}) (*models.Agent, error) {
	var agent models.Agent
	var configJSON sql.NullString
	var deletedAt sql.NullTime

	var err error
	switch s := scanner.(type) {
	case *sql.Row:
		err = s.Scan(
			&agent.ID, &agent.ProjectID, &agent.Name, &agent.Type, &agent.Status,
			&agent.WorktreeID, &agent.ContainerID, &agent.SessionID,
			&configJSON, &agent.Description, &agent.LastActivity, &agent.ErrorMessage, &agent.Version,
			&agent.CreatedAt, &agent.UpdatedAt, &deletedAt,
		)
	case *sql.Rows:
		err = s.Scan(
			&agent.ID, &agent.ProjectID, &agent.Name, &agent.Type, &agent.Status,
			&agent.WorktreeID, &agent.ContainerID, &agent.SessionID,
			&configJSON, &agent.Description, &agent.LastActivity, &agent.ErrorMessage, &agent.Version,
			&agent.CreatedAt, &agent.UpdatedAt, &deletedAt,
		)
	default:
		return nil, fmt.Errorf("지원되지 않는 스캐너 타입")
	}

	if err != nil {
		return nil, fmt.Errorf("에이전트 스캔 실패: %w", err)
	}

	// AgentConfig JSON 역직렬화
	if configJSON.Valid && configJSON.String != "" {
		if err := json.Unmarshal([]byte(configJSON.String), &agent.Config); err != nil {
			return nil, fmt.Errorf("config 역직렬화 실패: %w", err)
		}
	}

	// DeletedAt 처리
	if deletedAt.Valid {
		agent.DeletedAt = &deletedAt.Time
	}

	return &agent, nil
}