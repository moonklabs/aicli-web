package memory

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/storage"
	"github.com/google/uuid"
)

// AgentStorage 메모리 기반 에이전트 스토리지
type AgentStorage struct {
	agents map[string]*models.Agent
	mu     sync.RWMutex
}

// NewAgentStorage 새 에이전트 스토리지 생성
func NewAgentStorage() *AgentStorage {
	return &AgentStorage{
		agents: make(map[string]*models.Agent),
	}
}

// Create 새 에이전트 생성
func (a *AgentStorage) Create(ctx context.Context, agent *models.Agent) error {
	if agent.ID == "" {
		agent.ID = uuid.New().String()
	}

	now := time.Now()
	agent.CreatedAt = now
	agent.UpdatedAt = now
	agent.LastActivity = now

	a.mu.Lock()
	defer a.mu.Unlock()

	// 중복 이름 확인
	for _, existing := range a.agents {
		if existing.ProjectID == agent.ProjectID &&
			existing.Name == agent.Name &&
			existing.DeletedAt == nil {
			return storage.ErrDuplicateKey
		}
	}

	a.agents[agent.ID] = agent.Clone()
	return nil
}

// GetByID ID로 에이전트 조회
func (a *AgentStorage) GetByID(ctx context.Context, id string) (*models.Agent, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	agent, exists := a.agents[id]
	if !exists || agent.DeletedAt != nil {
		return nil, storage.ErrNotFound
	}

	return agent.Clone(), nil
}

// GetByProjectIDWithPagination 프로젝트 ID로 에이전트 목록 조회 (페이지네이션 포함)
func (a *AgentStorage) GetByProjectIDWithPagination(ctx context.Context, projectID string, pagination *models.PaginationRequest) ([]*models.Agent, int, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var filteredAgents []*models.Agent
	for _, agent := range a.agents {
		if agent.ProjectID == projectID && agent.DeletedAt == nil {
			filteredAgents = append(filteredAgents, agent.Clone())
		}
	}

	// 생성 시간 기준 정렬 (최신순)
	sort.Slice(filteredAgents, func(i, j int) bool {
		return filteredAgents[i].CreatedAt.After(filteredAgents[j].CreatedAt)
	})

	totalCount := len(filteredAgents)

	// 페이지네이션 적용
	if pagination != nil {
		start := pagination.GetOffset()
		end := start + pagination.Limit

		if start > len(filteredAgents) {
			return []*models.Agent{}, totalCount, nil
		}
		if end > len(filteredAgents) {
			end = len(filteredAgents)
		}

		filteredAgents = filteredAgents[start:end]
	}

	return filteredAgents, totalCount, nil
}

// Update 에이전트 업데이트
func (a *AgentStorage) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	agent, exists := a.agents[id]
	if !exists || agent.DeletedAt != nil {
		return storage.ErrNotFound
	}

	// 업데이트 적용
	if name, ok := updates["name"].(string); ok {
		// 이름 중복 확인
		for _, existing := range a.agents {
			if existing.ID != id &&
				existing.ProjectID == agent.ProjectID &&
				existing.Name == name &&
				existing.DeletedAt == nil {
				return storage.ErrDuplicateKey
			}
		}
		agent.Name = name
	}

	if status, ok := updates["status"].(models.AgentStatus); ok {
		agent.Status = status
	}

	if description, ok := updates["description"].(string); ok {
		agent.Description = description
	}

	if config, ok := updates["config"].(models.AgentConfig); ok {
		agent.Config = config
	}

	if worktreeID, ok := updates["worktree_id"].(string); ok {
		agent.WorktreeID = worktreeID
	}

	if containerID, ok := updates["container_id"].(string); ok {
		agent.ContainerID = containerID
	}

	if sessionID, ok := updates["session_id"].(string); ok {
		agent.SessionID = sessionID
	}

	if errorMessage, ok := updates["error_message"].(string); ok {
		agent.ErrorMessage = errorMessage
	}

	if lastActivity, ok := updates["last_activity"].(time.Time); ok {
		agent.LastActivity = lastActivity
	}

	agent.UpdatedAt = time.Now()
	return nil
}

// Delete 에이전트 삭제 (soft delete)
func (a *AgentStorage) Delete(ctx context.Context, id string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	agent, exists := a.agents[id]
	if !exists || agent.DeletedAt != nil {
		return storage.ErrNotFound
	}

	now := time.Now()
	agent.DeletedAt = &now
	agent.UpdatedAt = now

	return nil
}

// ExistsByName 프로젝트 내 이름으로 존재 여부 확인
func (a *AgentStorage) ExistsByName(ctx context.Context, projectID, name string) (bool, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	for _, agent := range a.agents {
		if agent.ProjectID == projectID &&
			agent.Name == name &&
			agent.DeletedAt == nil {
			return true, nil
		}
	}

	return false, nil
}

// GetByStatusWithPagination 상태별 에이전트 목록 조회 (페이지네이션 포함)
func (a *AgentStorage) GetByStatusWithPagination(ctx context.Context, status models.AgentStatus, pagination *models.PaginationRequest) ([]*models.Agent, int, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var filteredAgents []*models.Agent
	for _, agent := range a.agents {
		if agent.Status == status && agent.DeletedAt == nil {
			filteredAgents = append(filteredAgents, agent.Clone())
		}
	}

	// 마지막 활동 시간 기준 정렬 (최근순)
	sort.Slice(filteredAgents, func(i, j int) bool {
		return filteredAgents[i].LastActivity.After(filteredAgents[j].LastActivity)
	})

	totalCount := len(filteredAgents)

	// 페이지네이션 적용
	if pagination != nil {
		start := pagination.GetOffset()
		end := start + pagination.Limit

		if start > len(filteredAgents) {
			return []*models.Agent{}, totalCount, nil
		}
		if end > len(filteredAgents) {
			end = len(filteredAgents)
		}

		filteredAgents = filteredAgents[start:end]
	}

	return filteredAgents, totalCount, nil
}

// GetActiveCount 활성 에이전트 수 조회
func (a *AgentStorage) GetActiveCount(ctx context.Context, projectID string) (int64, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var count int64
	for _, agent := range a.agents {
		if agent.ProjectID == projectID &&
			agent.Status == models.AgentStatusRunning &&
			agent.DeletedAt == nil {
			count++
		}
	}

	return count, nil
}

// UpdateStatus 에이전트 상태 업데이트
func (a *AgentStorage) UpdateStatus(ctx context.Context, id string, status models.AgentStatus, errorMessage string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	agent, exists := a.agents[id]
	if !exists || agent.DeletedAt != nil {
		return storage.ErrNotFound
	}

	agent.Status = status
	agent.ErrorMessage = errorMessage
	agent.LastActivity = time.Now()
	agent.UpdatedAt = time.Now()

	return nil
}

// UpdateActivity 마지막 활동 시간 업데이트
func (a *AgentStorage) UpdateActivity(ctx context.Context, id string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	agent, exists := a.agents[id]
	if !exists || agent.DeletedAt != nil {
		return storage.ErrNotFound
	}

	agent.LastActivity = time.Now()
	agent.UpdatedAt = time.Now()

	return nil
}

// GetByContainerID 컨테이너 ID로 에이전트 조회
func (a *AgentStorage) GetByContainerID(ctx context.Context, containerID string) (*models.Agent, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	for _, agent := range a.agents {
		if agent.ContainerID == containerID && agent.DeletedAt == nil {
			return agent.Clone(), nil
		}
	}

	return nil, storage.ErrNotFound
}

// GetByProjectID 프로젝트 ID로 에이전트 목록 조회 (페이지네이션 없이)
func (a *AgentStorage) GetByProjectID(ctx context.Context, projectID string) ([]*models.Agent, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var filteredAgents []*models.Agent
	for _, agent := range a.agents {
		if agent.ProjectID == projectID && agent.DeletedAt == nil {
			filteredAgents = append(filteredAgents, agent.Clone())
		}
	}

	// 생성 시간 기준 정렬 (최신순)
	sort.Slice(filteredAgents, func(i, j int) bool {
		return filteredAgents[i].CreatedAt.After(filteredAgents[j].CreatedAt)
	})

	return filteredAgents, nil
}

// GetByStatus 상태별 에이전트 목록 조회 (페이지네이션 없이)
func (a *AgentStorage) GetByStatus(ctx context.Context, status models.AgentStatus) ([]*models.Agent, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var filteredAgents []*models.Agent
	for _, agent := range a.agents {
		if agent.Status == status && agent.DeletedAt == nil {
			filteredAgents = append(filteredAgents, agent.Clone())
		}
	}

	// 마지막 활동 시간 기준 정렬 (최근순)
	sort.Slice(filteredAgents, func(i, j int) bool {
		return filteredAgents[i].LastActivity.After(filteredAgents[j].LastActivity)
	})

	return filteredAgents, nil
}

// GetAll 모든 에이전트 목록 조회
func (a *AgentStorage) GetAll(ctx context.Context) ([]*models.Agent, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var allAgents []*models.Agent
	for _, agent := range a.agents {
		if agent.DeletedAt == nil {
			allAgents = append(allAgents, agent.Clone())
		}
	}

	// 생성 시간 기준 정렬 (최신순)
	sort.Slice(allAgents, func(i, j int) bool {
		return allAgents[i].CreatedAt.After(allAgents[j].CreatedAt)
	})

	return allAgents, nil
}
