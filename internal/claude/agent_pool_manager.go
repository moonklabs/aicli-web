package claude

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aicli/aicli-web/internal/models"
)

// AgentPoolManager는 에이전트 풀을 관리합니다
type AgentPoolManager struct {
	// 풀 설정
	config PoolManagerConfig

	// 에이전트 풀들 (타입별)
	pools    map[models.AgentType]*TypedAgentPool
	poolsMu  sync.RWMutex

	// 전역 통계
	stats    *PoolManagerStats
	statsMu  sync.RWMutex

	// 스케일링 및 최적화
	scaler    *PoolScaler
	optimizer *PoolOptimizer
	
	// 워밍업 관리
	warmer   *PoolWarmer
	warmerMu sync.RWMutex

	// 생명주기 관리
	running atomic.Bool
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// PoolManagerConfig는 풀 관리자 설정입니다
type PoolManagerConfig struct {
	// 풀 크기 설정
	DefaultPoolSize    int                              `json:"default_pool_size"`
	MaxPoolSize        int                              `json:"max_pool_size"`
	MinPoolSize        int                              `json:"min_pool_size"`
	TypeSpecificSizes  map[models.AgentType]PoolSizes  `json:"type_specific_sizes"`

	// 생성 설정
	PrewarmAgents      bool          `json:"prewarm_agents"`
	PrewarmCount       int           `json:"prewarm_count"`
	CreationTimeout    time.Duration `json:"creation_timeout"`
	MaxConcurrentCreation int        `json:"max_concurrent_creation"`

	// 유지 관리 설정
	IdleTimeout        time.Duration `json:"idle_timeout"`
	MaxIdleTime        time.Duration `json:"max_idle_time"`
	HealthCheckInterval time.Duration `json:"health_check_interval"`
	CleanupInterval    time.Duration `json:"cleanup_interval"`

	// 성능 설정
	EnableBackgroundCreation bool  `json:"enable_background_creation"`
	EnablePredictiveScaling  bool  `json:"enable_predictive_scaling"`
	EnableMetrics           bool  `json:"enable_metrics"`
	OptimizationInterval    time.Duration `json:"optimization_interval"`
}

// PoolSizes는 풀 크기 설정입니다
type PoolSizes struct {
	Min     int `json:"min"`
	Max     int `json:"max"`
	Initial int `json:"initial"`
}

// TypedAgentPool은 특정 타입의 에이전트 풀입니다
type TypedAgentPool struct {
	agentType models.AgentType
	
	// 풀 상태
	available   []*PooledAgent
	inUse       map[string]*PooledAgent
	creating    int32
	availableMu sync.RWMutex
	inUseMu     sync.RWMutex

	// 설정
	config     PoolSizes
	creationCh chan struct{} // 동시 생성 제한
	
	// 통계
	stats      *TypedPoolStats
	statsMu    sync.RWMutex
	
	// 팩토리
	factory AgentFactory
	
	// 생명주기
	ctx    context.Context
	cancel context.CancelFunc
}

// PooledAgent는 풀링된 에이전트입니다
type PooledAgent struct {
	Agent        *models.Agent
	CreatedAt    time.Time
	LastUsed     time.Time
	UsageCount   int32
	IsHealthy    bool
	
	// 풀 메타데이터
	PoolType     models.AgentType
	IsPrewarmed  bool
	
	// 동기화
	mu sync.RWMutex
}

// AgentFactory는 에이전트 생성 팩토리 인터페이스입니다
type AgentFactory interface {
	CreateAgent(ctx context.Context, agentType models.AgentType, config *models.AgentConfig) (*models.Agent, error)
	PrepareAgent(ctx context.Context, agent *models.Agent) error
	ValidateAgent(ctx context.Context, agent *models.Agent) error
	CleanupAgent(ctx context.Context, agent *models.Agent) error
}

// PoolManagerStats는 풀 관리자 통계입니다
type PoolManagerStats struct {
	TotalPools           int                             `json:"total_pools"`
	TotalAgents          int                             `json:"total_agents"`
	TotalAvailable       int                             `json:"total_available"`
	TotalInUse           int                             `json:"total_in_use"`
	TotalCreating        int                             `json:"total_creating"`
	
	// 성능 메트릭
	AverageCreationTime  time.Duration                   `json:"average_creation_time"`
	AverageAcquisitionTime time.Duration                 `json:"average_acquisition_time"`
	PoolHitRate          float64                         `json:"pool_hit_rate"`
	
	// 타입별 통계
	TypeStats            map[models.AgentType]*TypedPoolStats `json:"type_stats"`
	
	// 시간 정보
	LastUpdate           time.Time                       `json:"last_update"`
	UptimeSeconds        int64                          `json:"uptime_seconds"`
}

// TypedPoolStats는 타입별 풀 통계입니다
type TypedPoolStats struct {
	AgentType            models.AgentType `json:"agent_type"`
	PoolSize             int              `json:"pool_size"`
	Available            int              `json:"available"`
	InUse                int              `json:"in_use"`
	Creating             int              `json:"creating"`
	
	// 성능 메트릭
	CreationTime         MovingAverage    `json:"creation_time"`
	AcquisitionTime      MovingAverage    `json:"acquisition_time"`
	HitRate              float64          `json:"hit_rate"`
	
	// 카운터
	TotalCreated         int64            `json:"total_created"`
	TotalAcquired        int64            `json:"total_acquired"`
	TotalReleased        int64            `json:"total_released"`
	TotalDestroyed       int64            `json:"total_destroyed"`
	
	// 에러 통계
	CreationErrors       int64            `json:"creation_errors"`
	AcquisitionErrors    int64            `json:"acquisition_errors"`
	HealthCheckFailures  int64            `json:"health_check_failures"`
	
	// 시간 정보
	LastCreated          time.Time        `json:"last_created"`
	LastAcquired         time.Time        `json:"last_acquired"`
}

// MovingAverage는 이동 평균 계산기입니다
type MovingAverage struct {
	values []time.Duration
	sum    time.Duration
	index  int
	size   int
	full   bool
	mu     sync.RWMutex
}

// PoolScaler는 풀 자동 스케일링을 담당합니다
type PoolScaler struct {
	manager      *AgentPoolManager
	config       ScalerConfig
	
	// 예측 모델
	predictor    *UsagePredictor
	
	// 스케일링 이력
	history      []ScalingEvent
	historyMu    sync.RWMutex
	
	// 생명주기
	running      atomic.Bool
	ctx          context.Context
	cancel       context.CancelFunc
}

// ScalerConfig는 스케일러 설정입니다
type ScalerConfig struct {
	Enabled              bool          `json:"enabled"`
	EvaluationInterval   time.Duration `json:"evaluation_interval"`
	ScaleUpThreshold     float64       `json:"scale_up_threshold"`
	ScaleDownThreshold   float64       `json:"scale_down_threshold"`
	CooldownPeriod       time.Duration `json:"cooldown_period"`
	MaxScaleUp           int           `json:"max_scale_up"`
	MaxScaleDown         int           `json:"max_scale_down"`
	PredictiveEnabled    bool          `json:"predictive_enabled"`
}

// ScalingEvent는 스케일링 이벤트입니다
type ScalingEvent struct {
	Timestamp    time.Time        `json:"timestamp"`
	AgentType    models.AgentType `json:"agent_type"`
	Action       ScalingAction    `json:"action"`
	FromSize     int              `json:"from_size"`
	ToSize       int              `json:"to_size"`
	Reason       string           `json:"reason"`
	Utilization  float64          `json:"utilization"`
}

// ScalingAction은 스케일링 액션입니다
type ScalingAction string

const (
	ScalingActionScaleUp   ScalingAction = "scale_up"
	ScalingActionScaleDown ScalingAction = "scale_down"
	ScalingActionNoAction  ScalingAction = "no_action"
)

// UsagePredictor는 사용량 예측기입니다
type UsagePredictor struct {
	// 이력 데이터
	usageHistory []UsageDataPoint
	historyMu    sync.RWMutex
	
	// 예측 모델 설정
	windowSize   int
	predictionWindow time.Duration
	
	// 통계
	accuracy     float64
	lastPrediction time.Time
}

// UsageDataPoint는 사용량 데이터 포인트입니다
type UsageDataPoint struct {
	Timestamp   time.Time `json:"timestamp"`
	AgentType   models.AgentType `json:"agent_type"`
	InUse       int       `json:"in_use"`
	Available   int       `json:"available"`
	Utilization float64   `json:"utilization"`
}

// PoolOptimizer는 풀 최적화를 담당합니다
type PoolOptimizer struct {
	manager     *AgentPoolManager
	
	// 최적화 전략
	strategies  []OptimizationStrategy
	
	// 최적화 이력
	optimizations []OptimizationEvent
	optimizeMu    sync.RWMutex
	
	// 생명주기
	running     atomic.Bool
	ctx         context.Context
	cancel      context.CancelFunc
}

// OptimizationStrategy는 최적화 전략 인터페이스입니다
type OptimizationStrategy interface {
	Name() string
	Apply(ctx context.Context, pool *TypedAgentPool) (*OptimizationResult, error)
	CanApply(pool *TypedAgentPool) bool
	Priority() int
}

// OptimizationResult는 최적화 결과입니다
type OptimizationResult struct {
	Strategy      string                 `json:"strategy"`
	Applied       bool                   `json:"applied"`
	Description   string                 `json:"description"`
	ImpactScore   float64                `json:"impact_score"`
	Changes       []string               `json:"changes"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// OptimizationEvent는 최적화 이벤트입니다
type OptimizationEvent struct {
	Timestamp   time.Time           `json:"timestamp"`
	AgentType   models.AgentType    `json:"agent_type"`
	Strategy    string              `json:"strategy"`
	Result      *OptimizationResult `json:"result"`
}

// PoolWarmer는 풀 워밍업을 담당합니다
type PoolWarmer struct {
	manager     *AgentPoolManager
	
	// 워밍업 설정
	config      WarmupConfig
	
	// 워밍업 상태
	warmupTasks map[models.AgentType]*WarmupTask
	tasksMu     sync.RWMutex
	
	// 생명주기
	running     atomic.Bool
	ctx         context.Context
	cancel      context.CancelFunc
}

// WarmupConfig는 워밍업 설정입니다
type WarmupConfig struct {
	Enabled            bool                              `json:"enabled"`
	InitialWarmupCount map[models.AgentType]int         `json:"initial_warmup_count"`
	BackgroundWarmup   bool                              `json:"background_warmup"`
	WarmupInterval     time.Duration                     `json:"warmup_interval"`
	MaxConcurrentWarmup int                              `json:"max_concurrent_warmup"`
}

// WarmupTask는 워밍업 태스크입니다
type WarmupTask struct {
	AgentType     models.AgentType `json:"agent_type"`
	TargetCount   int              `json:"target_count"`
	CurrentCount  int              `json:"current_count"`
	InProgress    int              `json:"in_progress"`
	StartedAt     time.Time        `json:"started_at"`
	LastUpdate    time.Time        `json:"last_update"`
	Completed     bool             `json:"completed"`
}

// DefaultPoolManagerConfig는 기본 풀 관리자 설정을 반환합니다
func DefaultPoolManagerConfig() PoolManagerConfig {
	return PoolManagerConfig{
		DefaultPoolSize:       10,
		MaxPoolSize:          100,
		MinPoolSize:          2,
		TypeSpecificSizes: map[models.AgentType]PoolSizes{
			models.AgentTypeClaude: {Min: 5, Max: 50, Initial: 10},
			models.AgentTypeGit:    {Min: 2, Max: 20, Initial: 5},
			models.AgentTypeDocker: {Min: 3, Max: 30, Initial: 8},
		},
		PrewarmAgents:         true,
		PrewarmCount:          5,
		CreationTimeout:       30 * time.Second,
		MaxConcurrentCreation: 5,
		IdleTimeout:           10 * time.Minute,
		MaxIdleTime:           30 * time.Minute,
		HealthCheckInterval:   2 * time.Minute,
		CleanupInterval:       5 * time.Minute,
		EnableBackgroundCreation: true,
		EnablePredictiveScaling:  true,
		EnableMetrics:           true,
		OptimizationInterval:    10 * time.Minute,
	}
}

// NewAgentPoolManager는 새로운 에이전트 풀 관리자를 생성합니다
func NewAgentPoolManager(config PoolManagerConfig, factory AgentFactory) *AgentPoolManager {
	ctx, cancel := context.WithCancel(context.Background())
	
	manager := &AgentPoolManager{
		config:  config,
		pools:   make(map[models.AgentType]*TypedAgentPool),
		stats:   &PoolManagerStats{TypeStats: make(map[models.AgentType]*TypedPoolStats)},
		ctx:     ctx,
		cancel:  cancel,
	}
	
	// 컴포넌트 초기화
	manager.scaler = NewPoolScaler(manager, DefaultScalerConfig())
	manager.optimizer = NewPoolOptimizer(manager)
	manager.warmer = NewPoolWarmer(manager, DefaultWarmupConfig())
	
	// 타입별 풀 초기화
	for agentType, sizes := range config.TypeSpecificSizes {
		pool := manager.createTypedPool(agentType, sizes, factory)
		manager.pools[agentType] = pool
		manager.stats.TypeStats[agentType] = &TypedPoolStats{AgentType: agentType}
	}
	
	return manager
}

// Start는 풀 관리자를 시작합니다
func (pm *AgentPoolManager) Start() error {
	if !pm.running.CompareAndSwap(false, true) {
		return fmt.Errorf("pool manager is already running")
	}
	
	// 워밍업 시작
	if pm.config.PrewarmAgents {
		if err := pm.warmer.Start(); err != nil {
			return fmt.Errorf("failed to start warmer: %w", err)
		}
	}
	
	// 스케일러 시작
	if err := pm.scaler.Start(); err != nil {
		return fmt.Errorf("failed to start scaler: %w", err)
	}
	
	// 옵티마이저 시작
	if err := pm.optimizer.Start(); err != nil {
		return fmt.Errorf("failed to start optimizer: %w", err)
	}
	
	// 백그라운드 작업 시작
	pm.wg.Add(3)
	go pm.maintenanceLoop()
	go pm.metricsLoop()
	go pm.healthCheckLoop()
	
	return nil
}

// Stop은 풀 관리자를 중지합니다
func (pm *AgentPoolManager) Stop() error {
	if !pm.running.CompareAndSwap(true, false) {
		return nil
	}
	
	// 컴포넌트 중지
	pm.warmer.Stop()
	pm.scaler.Stop()
	pm.optimizer.Stop()
	
	// 백그라운드 작업 중지
	pm.cancel()
	pm.wg.Wait()
	
	// 모든 풀 정리
	pm.poolsMu.Lock()
	for _, pool := range pm.pools {
		pool.cleanup()
	}
	pm.poolsMu.Unlock()
	
	return nil
}

// AcquireAgent는 풀에서 에이전트를 획득합니다
func (pm *AgentPoolManager) AcquireAgent(ctx context.Context, agentType models.AgentType, config *models.AgentConfig) (*PooledAgent, error) {
	startTime := time.Now()
	
	pool, exists := pm.getPool(agentType)
	if !exists {
		return nil, fmt.Errorf("no pool available for agent type: %s", agentType)
	}
	
	// 풀에서 에이전트 획득 시도
	agent, err := pool.acquireAgent(ctx, config)
	if err != nil {
		// 통계 업데이트
		pool.updateAcquisitionError()
		return nil, fmt.Errorf("failed to acquire agent: %w", err)
	}
	
	// 통계 업데이트
	acquisitionTime := time.Since(startTime)
	pool.updateAcquisitionSuccess(acquisitionTime)
	
	return agent, nil
}

// ReleaseAgent는 에이전트를 풀에 반환합니다
func (pm *AgentPoolManager) ReleaseAgent(agent *PooledAgent) error {
	if agent == nil {
		return fmt.Errorf("cannot release nil agent")
	}
	
	pool, exists := pm.getPool(agent.PoolType)
	if !exists {
		return fmt.Errorf("no pool available for agent type: %s", agent.PoolType)
	}
	
	return pool.releaseAgent(agent)
}

// GetPoolStats는 풀 통계를 반환합니다
func (pm *AgentPoolManager) GetPoolStats() *PoolManagerStats {
	pm.statsMu.RLock()
	defer pm.statsMu.RUnlock()
	
	// 통계 복사본 생성
	stats := &PoolManagerStats{
		TotalPools:     len(pm.pools),
		TypeStats:     make(map[models.AgentType]*TypedPoolStats),
		LastUpdate:    time.Now(),
	}
	
	// 타입별 통계 수집
	for agentType, pool := range pm.pools {
		typeStats := pool.getStats()
		stats.TypeStats[agentType] = typeStats
		
		stats.TotalAgents += typeStats.PoolSize
		stats.TotalAvailable += typeStats.Available
		stats.TotalInUse += typeStats.InUse
		stats.TotalCreating += typeStats.Creating
	}
	
	// 전역 메트릭 계산
	pm.calculateGlobalMetrics(stats)
	
	return stats
}

// WarmupPool은 특정 타입의 풀을 워밍업합니다
func (pm *AgentPoolManager) WarmupPool(agentType models.AgentType, count int) error {
	pool, exists := pm.getPool(agentType)
	if !exists {
		return fmt.Errorf("no pool available for agent type: %s", agentType)
	}
	
	return pool.warmup(pm.ctx, count)
}

// 내부 메서드들

func (pm *AgentPoolManager) createTypedPool(agentType models.AgentType, sizes PoolSizes, factory AgentFactory) *TypedAgentPool {
	ctx, cancel := context.WithCancel(pm.ctx)
	
	pool := &TypedAgentPool{
		agentType:   agentType,
		available:   make([]*PooledAgent, 0, sizes.Max),
		inUse:       make(map[string]*PooledAgent),
		config:      sizes,
		creationCh:  make(chan struct{}, pm.config.MaxConcurrentCreation),
		stats:       &TypedPoolStats{AgentType: agentType},
		factory:     factory,
		ctx:         ctx,
		cancel:      cancel,
	}
	
	// 초기 에이전트 생성
	if sizes.Initial > 0 {
		go pool.warmup(ctx, sizes.Initial)
	}
	
	return pool
}

func (pm *AgentPoolManager) getPool(agentType models.AgentType) (*TypedAgentPool, bool) {
	pm.poolsMu.RLock()
	defer pm.poolsMu.RUnlock()
	
	pool, exists := pm.pools[agentType]
	return pool, exists
}

func (pm *AgentPoolManager) maintenanceLoop() {
	defer pm.wg.Done()
	
	ticker := time.NewTicker(pm.config.CleanupInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-pm.ctx.Done():
			return
		case <-ticker.C:
			pm.performMaintenance()
		}
	}
}

func (pm *AgentPoolManager) metricsLoop() {
	defer pm.wg.Done()
	
	ticker := time.NewTicker(30 * time.Second) // 30초마다 메트릭 업데이트
	defer ticker.Stop()
	
	for {
		select {
		case <-pm.ctx.Done():
			return
		case <-ticker.C:
			pm.updateMetrics()
		}
	}
}

func (pm *AgentPoolManager) healthCheckLoop() {
	defer pm.wg.Done()
	
	ticker := time.NewTicker(pm.config.HealthCheckInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-pm.ctx.Done():
			return
		case <-ticker.C:
			pm.performHealthChecks()
		}
	}
}

func (pm *AgentPoolManager) performMaintenance() {
	pm.poolsMu.RLock()
	pools := make([]*TypedAgentPool, 0, len(pm.pools))
	for _, pool := range pm.pools {
		pools = append(pools, pool)
	}
	pm.poolsMu.RUnlock()
	
	for _, pool := range pools {
		pool.cleanup()
	}
}

func (pm *AgentPoolManager) updateMetrics() {
	// 메트릭 업데이트 로직
	stats := pm.GetPoolStats()
	
	pm.statsMu.Lock()
	pm.stats = stats
	pm.statsMu.Unlock()
}

func (pm *AgentPoolManager) performHealthChecks() {
	pm.poolsMu.RLock()
	pools := make([]*TypedAgentPool, 0, len(pm.pools))
	for _, pool := range pm.pools {
		pools = append(pools, pool)
	}
	pm.poolsMu.RUnlock()
	
	for _, pool := range pools {
		go pool.performHealthCheck(pm.ctx)
	}
}

func (pm *AgentPoolManager) calculateGlobalMetrics(stats *PoolManagerStats) {
	if len(stats.TypeStats) == 0 {
		return
	}
	
	// 전역 히트율 계산
	var totalAcquired, totalHits int64
	var totalCreationTime, totalAcquisitionTime time.Duration
	
	for _, typeStats := range stats.TypeStats {
		totalAcquired += typeStats.TotalAcquired
		totalHits += int64(float64(typeStats.TotalAcquired) * typeStats.HitRate)
		totalCreationTime += typeStats.CreationTime.Average()
		totalAcquisitionTime += typeStats.AcquisitionTime.Average()
	}
	
	if totalAcquired > 0 {
		stats.PoolHitRate = float64(totalHits) / float64(totalAcquired)
	}
	
	if len(stats.TypeStats) > 0 {
		stats.AverageCreationTime = totalCreationTime / time.Duration(len(stats.TypeStats))
		stats.AverageAcquisitionTime = totalAcquisitionTime / time.Duration(len(stats.TypeStats))
	}
}

// TypedAgentPool 메서드들

func (tap *TypedAgentPool) acquireAgent(ctx context.Context, config *models.AgentConfig) (*PooledAgent, error) {
	// 사용 가능한 에이전트 확인
	if agent := tap.getAvailableAgent(); agent != nil {
		tap.markInUse(agent)
		atomic.AddInt64(&tap.stats.TotalAcquired, 1)
		tap.stats.LastAcquired = time.Now()
		return agent, nil
	}
	
	// 새 에이전트 생성
	return tap.createAgent(ctx, config)
}

func (tap *TypedAgentPool) getAvailableAgent() *PooledAgent {
	tap.availableMu.Lock()
	defer tap.availableMu.Unlock()
	
	if len(tap.available) == 0 {
		return nil
	}
	
	// 가장 최근에 사용된 에이전트 반환 (LIFO)
	agent := tap.available[len(tap.available)-1]
	tap.available = tap.available[:len(tap.available)-1]
	
	return agent
}

func (tap *TypedAgentPool) createAgent(ctx context.Context, config *models.AgentConfig) (*PooledAgent, error) {
	startTime := time.Now()
	
	// 동시 생성 제한
	select {
	case tap.creationCh <- struct{}{}:
		defer func() { <-tap.creationCh }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	
	atomic.AddInt32(&tap.creating, 1)
	defer atomic.AddInt32(&tap.creating, -1)
	
	// 에이전트 생성
	agent, err := tap.factory.CreateAgent(ctx, tap.agentType, config)
	if err != nil {
		atomic.AddInt64(&tap.stats.CreationErrors, 1)
		return nil, fmt.Errorf("failed to create agent: %w", err)
	}
	
	// 풀링된 에이전트 래핑
	pooledAgent := &PooledAgent{
		Agent:       agent,
		CreatedAt:   time.Now(),
		LastUsed:    time.Now(),
		UsageCount:  0,
		IsHealthy:   true,
		PoolType:    tap.agentType,
		IsPrewarmed: false,
	}
	
	// 에이전트 준비
	if err := tap.factory.PrepareAgent(ctx, agent); err != nil {
		tap.factory.CleanupAgent(ctx, agent)
		atomic.AddInt64(&tap.stats.CreationErrors, 1)
		return nil, fmt.Errorf("failed to prepare agent: %w", err)
	}
	
	// 통계 업데이트
	creationTime := time.Since(startTime)
	tap.stats.CreationTime.Add(creationTime)
	atomic.AddInt64(&tap.stats.TotalCreated, 1)
	tap.stats.LastCreated = time.Now()
	
	tap.markInUse(pooledAgent)
	
	return pooledAgent, nil
}

func (tap *TypedAgentPool) markInUse(agent *PooledAgent) {
	tap.inUseMu.Lock()
	defer tap.inUseMu.Unlock()
	
	agent.mu.Lock()
	agent.LastUsed = time.Now()
	atomic.AddInt32(&agent.UsageCount, 1)
	agent.mu.Unlock()
	
	tap.inUse[agent.Agent.ID] = agent
}

func (tap *TypedAgentPool) releaseAgent(agent *PooledAgent) error {
	// 사용 중 목록에서 제거
	tap.inUseMu.Lock()
	delete(tap.inUse, agent.Agent.ID)
	tap.inUseMu.Unlock()
	
	// 에이전트 상태 검증
	if !agent.IsHealthy {
		// 건강하지 않은 에이전트는 파괴
		return tap.destroyAgent(agent)
	}
	
	// 유휴 시간 확인
	if time.Since(agent.LastUsed) > 30*time.Minute {
		return tap.destroyAgent(agent)
	}
	
	// 사용 가능 목록에 추가
	tap.availableMu.Lock()
	tap.available = append(tap.available, agent)
	tap.availableMu.Unlock()
	
	atomic.AddInt64(&tap.stats.TotalReleased, 1)
	
	return nil
}

func (tap *TypedAgentPool) destroyAgent(agent *PooledAgent) error {
	if err := tap.factory.CleanupAgent(tap.ctx, agent.Agent); err != nil {
		// 정리 실패 로그만 하고 계속 진행
	}
	
	atomic.AddInt64(&tap.stats.TotalDestroyed, 1)
	
	return nil
}

func (tap *TypedAgentPool) warmup(ctx context.Context, count int) error {
	var wg sync.WaitGroup
	errCh := make(chan error, count)
	
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			agent, err := tap.createPrewarmedAgent(ctx)
			if err != nil {
				errCh <- err
				return
			}
			
			// 미리 생성된 에이전트를 사용 가능 목록에 추가
			tap.availableMu.Lock()
			tap.available = append(tap.available, agent)
			tap.availableMu.Unlock()
		}()
	}
	
	wg.Wait()
	close(errCh)
	
	// 에러 수집
	var errors []error
	for err := range errCh {
		errors = append(errors, err)
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("warmup completed with %d errors", len(errors))
	}
	
	return nil
}

func (tap *TypedAgentPool) createPrewarmedAgent(ctx context.Context) (*PooledAgent, error) {
	// 기본 설정으로 에이전트 생성
	config := &models.AgentConfig{
		Type: tap.agentType,
		// 기타 기본 설정
	}
	
	agent, err := tap.factory.CreateAgent(ctx, tap.agentType, config)
	if err != nil {
		return nil, err
	}
	
	pooledAgent := &PooledAgent{
		Agent:       agent,
		CreatedAt:   time.Now(),
		LastUsed:    time.Time{}, // 아직 사용되지 않음
		UsageCount:  0,
		IsHealthy:   true,
		PoolType:    tap.agentType,
		IsPrewarmed: true,
	}
	
	// 에이전트 준비
	if err := tap.factory.PrepareAgent(ctx, agent); err != nil {
		tap.factory.CleanupAgent(ctx, agent)
		return nil, err
	}
	
	return pooledAgent, nil
}

func (tap *TypedAgentPool) cleanup() {
	now := time.Now()
	
	// 사용 가능한 에이전트 정리
	tap.availableMu.Lock()
	var validAgents []*PooledAgent
	for _, agent := range tap.available {
		// 최대 유휴 시간 확인
		if agent.LastUsed.IsZero() || now.Sub(agent.LastUsed) < 30*time.Minute {
			validAgents = append(validAgents, agent)
		} else {
			// 오래된 에이전트 제거
			go tap.destroyAgent(agent)
		}
	}
	tap.available = validAgents
	tap.availableMu.Unlock()
}

func (tap *TypedAgentPool) performHealthCheck(ctx context.Context) {
	// 사용 가능한 에이전트들 건강 상태 확인
	tap.availableMu.RLock()
	agents := make([]*PooledAgent, len(tap.available))
	copy(agents, tap.available)
	tap.availableMu.RUnlock()
	
	for _, agent := range agents {
		if err := tap.factory.ValidateAgent(ctx, agent.Agent); err != nil {
			// 건강하지 않은 에이전트 표시
			agent.mu.Lock()
			agent.IsHealthy = false
			agent.mu.Unlock()
			
			atomic.AddInt64(&tap.stats.HealthCheckFailures, 1)
		}
	}
}

func (tap *TypedAgentPool) getStats() *TypedPoolStats {
	tap.statsMu.RLock()
	defer tap.statsMu.RUnlock()
	
	stats := *tap.stats
	
	// 현재 상태 업데이트
	tap.availableMu.RLock()
	stats.Available = len(tap.available)
	tap.availableMu.RUnlock()
	
	tap.inUseMu.RLock()
	stats.InUse = len(tap.inUse)
	tap.inUseMu.RUnlock()
	
	stats.Creating = int(atomic.LoadInt32(&tap.creating))
	stats.PoolSize = stats.Available + stats.InUse + stats.Creating
	
	// 히트율 계산
	if stats.TotalAcquired > 0 {
		hits := stats.TotalAcquired - stats.CreationErrors
		stats.HitRate = float64(hits) / float64(stats.TotalAcquired)
	}
	
	return &stats
}

func (tap *TypedAgentPool) updateAcquisitionSuccess(duration time.Duration) {
	tap.statsMu.Lock()
	defer tap.statsMu.Unlock()
	
	tap.stats.AcquisitionTime.Add(duration)
}

func (tap *TypedAgentPool) updateAcquisitionError() {
	atomic.AddInt64(&tap.stats.AcquisitionErrors, 1)
}

// MovingAverage 메서드들

func NewMovingAverage(size int) *MovingAverage {
	return &MovingAverage{
		values: make([]time.Duration, size),
		size:   size,
	}
}

func (ma *MovingAverage) Add(value time.Duration) {
	ma.mu.Lock()
	defer ma.mu.Unlock()
	
	if ma.full {
		ma.sum -= ma.values[ma.index]
	}
	
	ma.values[ma.index] = value
	ma.sum += value
	ma.index = (ma.index + 1) % ma.size
	
	if !ma.full && ma.index == 0 {
		ma.full = true
	}
}

func (ma *MovingAverage) Average() time.Duration {
	ma.mu.RLock()
	defer ma.mu.RUnlock()
	
	if !ma.full && ma.index == 0 {
		return 0
	}
	
	count := ma.size
	if !ma.full {
		count = ma.index
	}
	
	return ma.sum / time.Duration(count)
}

// 기본 설정들

func DefaultScalerConfig() ScalerConfig {
	return ScalerConfig{
		Enabled:            true,
		EvaluationInterval: 1 * time.Minute,
		ScaleUpThreshold:   0.8,
		ScaleDownThreshold: 0.3,
		CooldownPeriod:     2 * time.Minute,
		MaxScaleUp:         5,
		MaxScaleDown:       3,
		PredictiveEnabled:  true,
	}
}

func DefaultWarmupConfig() WarmupConfig {
	return WarmupConfig{
		Enabled: true,
		InitialWarmupCount: map[models.AgentType]int{
			models.AgentTypeClaude: 5,
			models.AgentTypeGit:    2,
			models.AgentTypeDocker: 3,
		},
		BackgroundWarmup:    true,
		WarmupInterval:      5 * time.Minute,
		MaxConcurrentWarmup: 3,
	}
}

// 스텁 구현들 (실제 구현에서 완성 필요)

func NewPoolScaler(manager *AgentPoolManager, config ScalerConfig) *PoolScaler {
	return &PoolScaler{manager: manager, config: config}
}

func (ps *PoolScaler) Start() error { return nil }
func (ps *PoolScaler) Stop() error  { return nil }

func NewPoolOptimizer(manager *AgentPoolManager) *PoolOptimizer {
	return &PoolOptimizer{manager: manager}
}

func (po *PoolOptimizer) Start() error { return nil }
func (po *PoolOptimizer) Stop() error  { return nil }

func NewPoolWarmer(manager *AgentPoolManager, config WarmupConfig) *PoolWarmer {
	return &PoolWarmer{manager: manager, config: config}
}

func (pw *PoolWarmer) Start() error { return nil }
func (pw *PoolWarmer) Stop() error  { return nil }