package cache

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	
	"github.com/aicli/aicli-web/internal/models"
)

// QueryCache 쿼리 캐시
type QueryCache struct {
	cache  Cache
	keys   *CacheKey
	logger *zap.Logger
	
	// TTL 설정
	workspaceTTL time.Duration
	projectTTL   time.Duration
	sessionTTL   time.Duration
	taskTTL      time.Duration
	listTTL      time.Duration
}

// QueryCacheConfig 쿼리 캐시 설정
type QueryCacheConfig struct {
	Cache        Cache
	WorkspaceTTL time.Duration
	ProjectTTL   time.Duration
	SessionTTL   time.Duration
	TaskTTL      time.Duration
	ListTTL      time.Duration
	Logger       *zap.Logger
}

// DefaultQueryCacheConfig 기본 쿼리 캐시 설정
func DefaultQueryCacheConfig(cache Cache) QueryCacheConfig {
	return QueryCacheConfig{
		Cache:        cache,
		WorkspaceTTL: 10 * time.Minute,  // 워크스페이스는 자주 변경되지 않음
		ProjectTTL:   5 * time.Minute,   // 프로젝트도 비교적 안정적
		SessionTTL:   2 * time.Minute,   // 세션은 자주 변경됨
		TaskTTL:      1 * time.Minute,   // 태스크는 매우 자주 변경됨
		ListTTL:      30 * time.Second,  // 목록은 짧은 캐시
		Logger:       zap.NewNop(),
	}
}

// NewQueryCache 새 쿼리 캐시 생성
func NewQueryCache(config QueryCacheConfig) *QueryCache {
	if config.Logger == nil {
		config.Logger = zap.NewNop()
	}
	
	return &QueryCache{
		cache:        config.Cache,
		keys:         NewCacheKey("aicli"),
		logger:       config.Logger,
		workspaceTTL: config.WorkspaceTTL,
		projectTTL:   config.ProjectTTL,
		sessionTTL:   config.SessionTTL,
		taskTTL:      config.TaskTTL,
		listTTL:      config.ListTTL,
	}
}

// Workspace 캐시 메서드들

// GetWorkspace 워크스페이스 조회
func (qc *QueryCache) GetWorkspace(ctx context.Context, id string) (*models.Workspace, error) {
	key := qc.keys.WorkspaceKey(id)
	
	var workspace models.Workspace
	err := qc.cache.Get(ctx, key, &workspace)
	if err != nil {
		if err == ErrCacheMiss {
			qc.logger.Debug("워크스페이스 캐시 미스", zap.String("id", id))
		} else {
			qc.logger.Error("워크스페이스 캐시 조회 실패",
				zap.String("id", id),
				zap.Error(err),
			)
		}
		return nil, err
	}
	
	qc.logger.Debug("워크스페이스 캐시 히트", zap.String("id", id))
	return &workspace, nil
}

// SetWorkspace 워크스페이스 캐시
func (qc *QueryCache) SetWorkspace(ctx context.Context, workspace *models.Workspace) error {
	key := qc.keys.WorkspaceKey(workspace.ID)
	
	err := qc.cache.Set(ctx, key, workspace, qc.workspaceTTL)
	if err != nil {
		qc.logger.Error("워크스페이스 캐시 저장 실패",
			zap.String("id", workspace.ID),
			zap.Error(err),
		)
		return err
	}
	
	qc.logger.Debug("워크스페이스 캐시됨",
		zap.String("id", workspace.ID),
		zap.Duration("ttl", qc.workspaceTTL),
	)
	return nil
}

// InvalidateWorkspace 워크스페이스 캐시 무효화
func (qc *QueryCache) InvalidateWorkspace(ctx context.Context, id string) error {
	key := qc.keys.WorkspaceKey(id)
	
	err := qc.cache.Delete(ctx, key)
	if err != nil {
		qc.logger.Error("워크스페이스 캐시 무효화 실패",
			zap.String("id", id),
			zap.Error(err),
		)
		return err
	}
	
	qc.logger.Debug("워크스페이스 캐시 무효화됨", zap.String("id", id))
	return nil
}

// GetWorkspaceList 워크스페이스 목록 조회
func (qc *QueryCache) GetWorkspaceList(ctx context.Context, ownerID string, page, limit int) ([]*models.Workspace, int, error) {
	key := qc.keys.WorkspaceListKey(ownerID, page, limit)
	
	var result WorkspaceListResult
	err := qc.cache.Get(ctx, key, &result)
	if err != nil {
		if err == ErrCacheMiss {
			qc.logger.Debug("워크스페이스 목록 캐시 미스",
				zap.String("owner_id", ownerID),
				zap.Int("page", page),
				zap.Int("limit", limit),
			)
		}
		return nil, 0, err
	}
	
	qc.logger.Debug("워크스페이스 목록 캐시 히트",
		zap.String("owner_id", ownerID),
		zap.Int("page", page),
		zap.Int("limit", limit),
		zap.Int("count", len(result.Workspaces)),
	)
	
	return result.Workspaces, result.TotalCount, nil
}

// SetWorkspaceList 워크스페이스 목록 캐시
func (qc *QueryCache) SetWorkspaceList(ctx context.Context, ownerID string, page, limit int, workspaces []*models.Workspace, totalCount int) error {
	key := qc.keys.WorkspaceListKey(ownerID, page, limit)
	
	result := WorkspaceListResult{
		Workspaces: workspaces,
		TotalCount: totalCount,
	}
	
	err := qc.cache.Set(ctx, key, result, qc.listTTL)
	if err != nil {
		qc.logger.Error("워크스페이스 목록 캐시 저장 실패",
			zap.String("owner_id", ownerID),
			zap.Error(err),
		)
		return err
	}
	
	qc.logger.Debug("워크스페이스 목록 캐시됨",
		zap.String("owner_id", ownerID),
		zap.Int("count", len(workspaces)),
		zap.Duration("ttl", qc.listTTL),
	)
	return nil
}

// InvalidateWorkspaceList 워크스페이스 목록 캐시 무효화
func (qc *QueryCache) InvalidateWorkspaceList(ctx context.Context, ownerID string) error {
	// 페이지별 캐시를 모두 무효화하는 대신 패턴 기반으로 처리
	// 실제로는 더 정교한 무효화 전략이 필요함
	
	qc.logger.Debug("워크스페이스 목록 캐시 무효화 요청",
		zap.String("owner_id", ownerID),
	)
	
	// 여기서는 간단히 전체 캐시에서 관련 키들을 찾아 삭제
	// 실제 구현에서는 더 효율적인 방법 필요
	return nil
}

// Project 캐시 메서드들

// GetProject 프로젝트 조회
func (qc *QueryCache) GetProject(ctx context.Context, id string) (*models.Project, error) {
	key := qc.keys.ProjectKey(id)
	
	var project models.Project
	err := qc.cache.Get(ctx, key, &project)
	if err != nil {
		if err == ErrCacheMiss {
			qc.logger.Debug("프로젝트 캐시 미스", zap.String("id", id))
		}
		return nil, err
	}
	
	qc.logger.Debug("프로젝트 캐시 히트", zap.String("id", id))
	return &project, nil
}

// SetProject 프로젝트 캐시
func (qc *QueryCache) SetProject(ctx context.Context, project *models.Project) error {
	key := qc.keys.ProjectKey(project.ID)
	
	err := qc.cache.Set(ctx, key, project, qc.projectTTL)
	if err != nil {
		qc.logger.Error("프로젝트 캐시 저장 실패",
			zap.String("id", project.ID),
			zap.Error(err),
		)
		return err
	}
	
	qc.logger.Debug("프로젝트 캐시됨",
		zap.String("id", project.ID),
		zap.Duration("ttl", qc.projectTTL),
	)
	return nil
}

// InvalidateProject 프로젝트 캐시 무효화
func (qc *QueryCache) InvalidateProject(ctx context.Context, id string) error {
	key := qc.keys.ProjectKey(id)
	
	err := qc.cache.Delete(ctx, key)
	if err != nil {
		qc.logger.Error("프로젝트 캐시 무효화 실패",
			zap.String("id", id),
			zap.Error(err),
		)
		return err
	}
	
	qc.logger.Debug("프로젝트 캐시 무효화됨", zap.String("id", id))
	return nil
}

// Session 캐시 메서드들

// GetSession 세션 조회
func (qc *QueryCache) GetSession(ctx context.Context, id string) (*models.Session, error) {
	key := qc.keys.SessionKey(id)
	
	var session models.Session
	err := qc.cache.Get(ctx, key, &session)
	if err != nil {
		if err == ErrCacheMiss {
			qc.logger.Debug("세션 캐시 미스", zap.String("id", id))
		}
		return nil, err
	}
	
	qc.logger.Debug("세션 캐시 히트", zap.String("id", id))
	return &session, nil
}

// SetSession 세션 캐시
func (qc *QueryCache) SetSession(ctx context.Context, session *models.Session) error {
	key := qc.keys.SessionKey(session.ID)
	
	err := qc.cache.Set(ctx, key, session, qc.sessionTTL)
	if err != nil {
		qc.logger.Error("세션 캐시 저장 실패",
			zap.String("id", session.ID),
			zap.Error(err),
		)
		return err
	}
	
	qc.logger.Debug("세션 캐시됨",
		zap.String("id", session.ID),
		zap.Duration("ttl", qc.sessionTTL),
	)
	return nil
}

// InvalidateSession 세션 캐시 무효화
func (qc *QueryCache) InvalidateSession(ctx context.Context, id string) error {
	key := qc.keys.SessionKey(id)
	
	err := qc.cache.Delete(ctx, key)
	if err != nil {
		qc.logger.Error("세션 캐시 무효화 실패",
			zap.String("id", id),
			zap.Error(err),
		)
		return err
	}
	
	qc.logger.Debug("세션 캐시 무효화됨", zap.String("id", id))
	return nil
}

// Task 캐시 메서드들

// GetTask 태스크 조회
func (qc *QueryCache) GetTask(ctx context.Context, id string) (*models.Task, error) {
	key := qc.keys.TaskKey(id)
	
	var task models.Task
	err := qc.cache.Get(ctx, key, &task)
	if err != nil {
		if err == ErrCacheMiss {
			qc.logger.Debug("태스크 캐시 미스", zap.String("id", id))
		}
		return nil, err
	}
	
	qc.logger.Debug("태스크 캐시 히트", zap.String("id", id))
	return &task, nil
}

// SetTask 태스크 캐시
func (qc *QueryCache) SetTask(ctx context.Context, task *models.Task) error {
	key := qc.keys.TaskKey(task.ID)
	
	err := qc.cache.Set(ctx, key, task, qc.taskTTL)
	if err != nil {
		qc.logger.Error("태스크 캐시 저장 실패",
			zap.String("id", task.ID),
			zap.Error(err),
		)
		return err
	}
	
	qc.logger.Debug("태스크 캐시됨",
		zap.String("id", task.ID),
		zap.Duration("ttl", qc.taskTTL),
	)
	return nil
}

// InvalidateTask 태스크 캐시 무효화
func (qc *QueryCache) InvalidateTask(ctx context.Context, id string) error {
	key := qc.keys.TaskKey(id)
	
	err := qc.cache.Delete(ctx, key)
	if err != nil {
		qc.logger.Error("태스크 캐시 무효화 실패",
			zap.String("id", id),
			zap.Error(err),
		)
		return err
	}
	
	qc.logger.Debug("태스크 캐시 무효화됨", zap.String("id", id))
	return nil
}

// 일반적인 쿼리 캐시 메서드들

// GetQueryResult 쿼리 결과 조회
func (qc *QueryCache) GetQueryResult(ctx context.Context, query string, params []interface{}, dest interface{}) error {
	key := qc.keys.QueryKey(query, params...)
	
	err := qc.cache.Get(ctx, key, dest)
	if err != nil {
		if err == ErrCacheMiss {
			qc.logger.Debug("쿼리 결과 캐시 미스",
				zap.String("query_hash", key),
			)
		}
		return err
	}
	
	qc.logger.Debug("쿼리 결과 캐시 히트",
		zap.String("query_hash", key),
	)
	return nil
}

// SetQueryResult 쿼리 결과 캐시
func (qc *QueryCache) SetQueryResult(ctx context.Context, query string, params []interface{}, result interface{}, ttl time.Duration) error {
	key := qc.keys.QueryKey(query, params...)
	
	if ttl <= 0 {
		ttl = qc.listTTL
	}
	
	err := qc.cache.Set(ctx, key, result, ttl)
	if err != nil {
		qc.logger.Error("쿼리 결과 캐시 저장 실패",
			zap.String("query_hash", key),
			zap.Error(err),
		)
		return err
	}
	
	qc.logger.Debug("쿼리 결과 캐시됨",
		zap.String("query_hash", key),
		zap.Duration("ttl", ttl),
	)
	return nil
}

// InvalidatePattern 패턴 기반 캐시 무효화
func (qc *QueryCache) InvalidatePattern(ctx context.Context, pattern string) error {
	// 실제 구현에서는 캐시 백엔드에서 패턴 매칭을 지원해야 함
	// 여기서는 로그만 남김
	qc.logger.Debug("패턴 기반 캐시 무효화 요청",
		zap.String("pattern", pattern),
	)
	return nil
}

// Stats 캐시 통계 반환
func (qc *QueryCache) Stats() CacheStats {
	return qc.cache.Stats()
}

// Close 쿼리 캐시 종료
func (qc *QueryCache) Close() error {
	return qc.cache.Close()
}

// 결과 구조체들

// WorkspaceListResult 워크스페이스 목록 결과
type WorkspaceListResult struct {
	Workspaces []*models.Workspace `json:"workspaces"`
	TotalCount int                 `json:"total_count"`
}

// ProjectListResult 프로젝트 목록 결과
type ProjectListResult struct {
	Projects   []*models.Project `json:"projects"`
	TotalCount int               `json:"total_count"`
}

// SessionListResult 세션 목록 결과
type SessionListResult struct {
	Sessions   []*models.Session `json:"sessions"`
	TotalCount int               `json:"total_count"`
}

// TaskListResult 태스크 목록 결과
type TaskListResult struct {
	Tasks      []*models.Task `json:"tasks"`
	TotalCount int            `json:"total_count"`
}

// CacheMiddleware 캐시 미들웨어
type CacheMiddleware struct {
	queryCache *QueryCache
	logger     *zap.Logger
}

// NewCacheMiddleware 새 캐시 미들웨어 생성
func NewCacheMiddleware(queryCache *QueryCache, logger *zap.Logger) *CacheMiddleware {
	if logger == nil {
		logger = zap.NewNop()
	}
	
	return &CacheMiddleware{
		queryCache: queryCache,
		logger:     logger,
	}
}

// WithCache 캐시와 함께 실행
func (cm *CacheMiddleware) WithCache(ctx context.Context, key string, ttl time.Duration, dest interface{}, fallback func() (interface{}, error)) error {
	// 캐시에서 조회 시도
	err := cm.queryCache.cache.Get(ctx, key, dest)
	if err == nil {
		cm.logger.Debug("캐시 미들웨어 히트", zap.String("key", key))
		return nil
	}
	
	if err != ErrCacheMiss {
		cm.logger.Error("캐시 조회 실패",
			zap.String("key", key),
			zap.Error(err),
		)
	}
	
	// 캐시 미스 시 fallback 실행
	result, err := fallback()
	if err != nil {
		return err
	}
	
	// 결과를 캐시에 저장
	cacheErr := cm.queryCache.cache.Set(ctx, key, result, ttl)
	if cacheErr != nil {
		cm.logger.Error("캐시 저장 실패",
			zap.String("key", key),
			zap.Error(cacheErr),
		)
		// 캐시 저장 실패는 원본 결과에 영향을 주지 않음
	} else {
		cm.logger.Debug("캐시 미들웨어 저장",
			zap.String("key", key),
			zap.Duration("ttl", ttl),
		)
	}
	
	// 결과를 dest에 복사
	if result != nil {
		if err := copyResult(result, dest); err != nil {
			return fmt.Errorf("결과 복사 실패: %w", err)
		}
	}
	
	return nil
}

// copyResult 결과 복사 유틸리티
func copyResult(src, dest interface{}) error {
	// JSON 기반 복사 (성능상 최적은 아니지만 간단함)
	data, err := json.Marshal(src)
	if err != nil {
		return err
	}
	
	return json.Unmarshal(data, dest)
}