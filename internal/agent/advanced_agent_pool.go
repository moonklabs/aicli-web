package agent

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aicli/aicli-web/internal/docker"
	"github.com/aicli/aicli-web/internal/models"
	"github.com/prometheus/client_golang/prometheus"
)

// AdvancedAgentPool 고급 에이전트 풀 관리자
type AdvancedAgentPool struct {
	// 풀 설정
	config AdvancedPoolConfig

	// 풀 상태
	availableAgents chan *PooledAgent
	runningAgents   sync.Map // string -> *PooledAgent
	totalCreated    int64
	totalDestroyed  int64

	// 성능 최적화
	warmUpCount     int32
	preCreateWorker chan struct{}
	recycleWorker   chan *PooledAgent

	// 생명주기 관리
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// 의존성
	dockerClient docker.Client
	containerPool *ContainerPool

	// 메트릭
	metrics *PoolMetrics

	// 동시성 제어
	mu sync.RWMutex
}

// AdvancedPoolConfig 고급 풀 설정
type AdvancedPoolConfig struct {
	// 기본 풀 설정
	MinSize    int `json:"min_size"`    // 최소 풀 크기
	MaxSize    int `json:"max_size"`    // 최대 풀 크기
	WarmUpSize int `json:"warmup_size"` // 사전 생성할 에이전트 수

	// 성능 설정
	CreationTimeout    time.Duration `json:"creation_timeout"`    // 생성 타임아웃
	IdleTimeout        time.Duration `json:"idle_timeout"`        // 유휴 타임아웃
	MaxIdleTime        time.Duration `json:"max_idle_time"`       // 최대 유휴 시간
	RecycleWorkers     int           `json:"recycle_workers"`     // 재활용 워커 수
	PreCreateWorkers   int           `json:"pre_create_workers"`  // 사전 생성 워커 수

	// 리소스 제한
	MaxMemoryPerAgent string `json:"max_memory_per_agent"` // 에이전트당 최대 메모리
	MaxCPUPerAgent    string `json:"max_cpu_per_agent"`    // 에이전트당 최대 CPU

	// 자동 스케일링
	EnableAutoScaling   bool          `json:"enable_auto_scaling"`   // 자동 스케일링 활성화
	ScaleUpThreshold    float64       `json:"scale_up_threshold"`    // 스케일 업 임계값
	ScaleDownThreshold  float64       `json:"scale_down_threshold"`  // 스케일 다운 임계값
	ScaleCheckInterval  time.Duration `json:"scale_check_interval"`  // 스케일 체크 간격
}

// PooledAgent 풀링된 에이전트
type PooledAgent struct {
	// 기본 정보
	ID        string    `json:"id"`
	Agent     *models.Agent `json:"agent"`
	Container *docker.WorkspaceContainer `json:"container"`

	// 상태 관리
	Status        PooledAgentStatus `json:"status"`
	CreatedAt     time.Time         `json:"created_at"`
	LastUsedAt    time.Time         `json:"last_used_at"`
	UsageCount    int64             `json:"usage_count"`

	// 성능 메트릭
	CreationTime  time.Duration `json:"creation_time"`
	TotalRunTime  time.Duration `json:"total_run_time"`
	AverageMemory float64       `json:"average_memory"`
	PeakMemory    float64       `json:"peak_memory"`

	// 동시성 제어
	mu sync.RWMutex
}

// PooledAgentStatus 풀링된 에이전트 상태
type PooledAgentStatus string

const (
	PooledAgentStatusCreating   PooledAgentStatus = "creating"
	PooledAgentStatusAvailable  PooledAgentStatus = "available"
	PooledAgentStatusInUse      PooledAgentStatus = "in_use"
	PooledAgentStatusIdle       PooledAgentStatus = "idle"
	PooledAgentStatusRecycling  PooledAgentStatus = "recycling"
	PooledAgentStatusDestroying PooledAgentStatus = "destroying"
)

// PoolMetrics 풀 메트릭
type PoolMetrics struct {
	// Prometheus 메트릭
	PoolSize         prometheus.Gauge
	AvailableAgents  prometheus.Gauge
	RunningAgents    prometheus.Gauge
	CreationTime     prometheus.Histogram
	AgentUsageTime   prometheus.Histogram
	MemoryUsage      prometheus.Gauge
	RecycleRate      prometheus.Counter
	PoolHitRate      prometheus.Gauge

	// 내부 통계
	TotalCreated    int64
	TotalDestroyed  int64
	TotalRecycled   int64
	PoolHits        int64
	PoolMisses      int64
}

// NewAdvancedAgentPool 새 고급 에이전트 풀 생성
func NewAdvancedAgentPool(config AdvancedPoolConfig, dockerClient docker.Client) *AdvancedAgentPool {
	ctx, cancel := context.WithCancel(context.Background())

	pool := &AdvancedAgentPool{
		config:          config,
		availableAgents: make(chan *PooledAgent, config.MaxSize),
		preCreateWorker: make(chan struct{}, config.PreCreateWorkers),
		recycleWorker:   make(chan *PooledAgent, config.MaxSize),
		ctx:             ctx,
		cancel:          cancel,
		dockerClient:    dockerClient,
		metrics:         NewPoolMetrics(),
	}

	return pool
}

// Start 풀 시작
func (p *AdvancedAgentPool) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 컨테이너 풀 생성
	p.containerPool = NewContainerPool(
		p.config.WarmUpSize,
		p.config.MaxSize,
		p.dockerClient,
	)

	if err := p.containerPool.Start(); err != nil {
		return fmt.Errorf("failed to start container pool: %w", err)
	}

	// 워커 고루틴들 시작
	p.startWorkers()

	// 초기 에이전트 사전 생성
	go p.warmUpPool()

	// 자동 스케일링 활성화
	if p.config.EnableAutoScaling {
		go p.autoScalingLoop()
	}

	// 정리 작업 시작
	go p.cleanupLoop()

	return nil
}

// Stop 풀 중지
func (p *AdvancedAgentPool) Stop() error {
	p.cancel()
	p.wg.Wait()

	// 모든 에이전트 정리
	p.destroyAllAgents()

	// 컨테이너 풀 중지
	if p.containerPool != nil {
		p.containerPool.Stop()
	}

	return nil
}

// AcquireAgent 에이전트 획득
func (p *AdvancedAgentPool) AcquireAgent(ctx context.Context, req AgentRequest) (*PooledAgent, error) {
	start := time.Now()

	// 사용 가능한 에이전트 대기
	select {
	case agent := <-p.availableAgents:
		// 풀에서 에이전트 획득 성공
		agent.mu.Lock()
		agent.Status = PooledAgentStatusInUse
		agent.LastUsedAt = time.Now()
		agent.UsageCount++
		agent.mu.Unlock()

		p.runningAgents.Store(agent.ID, agent)
		
		// 메트릭 업데이트
		p.metrics.PoolHits++
		p.metrics.CreationTime.Observe(time.Since(start).Seconds())
		p.updateMetrics()

		return agent, nil

	case <-ctx.Done():
		p.metrics.PoolMisses++
		return nil, ctx.Err()

	case <-time.After(p.config.CreationTimeout):
		// 타임아웃 시 새 에이전트 생성
		agent, err := p.createNewAgent(ctx, req)
		if err != nil {
			p.metrics.PoolMisses++
			return nil, err
		}

		p.metrics.CreationTime.Observe(time.Since(start).Seconds())
		return agent, nil
	}
}

// ReleaseAgent 에이전트 반환
func (p *AdvancedAgentPool) ReleaseAgent(agent *PooledAgent) error {
	if agent == nil {
		return fmt.Errorf("agent is nil")
	}

	agent.mu.Lock()
	defer agent.mu.Unlock()

	// 실행 중인 에이전트 목록에서 제거
	p.runningAgents.Delete(agent.ID)

	// 에이전트 상태 확인 및 재활용 결정
	if p.shouldRecycleAgent(agent) {
		agent.Status = PooledAgentStatusRecycling
		select {
		case p.recycleWorker <- agent:
			// 재활용 워커에 전달
		default:
			// 재활용 큐가 가득 찬 경우 에이전트 삭제
			go p.destroyAgent(agent)
		}
	} else {
		// 재활용 불가능한 경우 에이전트 삭제
		go p.destroyAgent(agent)
	}

	p.updateMetrics()
	return nil
}

// createNewAgent 새 에이전트 생성
func (p *AdvancedAgentPool) createNewAgent(ctx context.Context, req AgentRequest) (*PooledAgent, error) {
	start := time.Now()

	// 새 Agent 모델 생성
	agent := &models.Agent{
		ID:        generateAgentID(),
		ProjectID: req.ProjectID,
		Name:      req.Name,
		Type:      models.AgentTypeClaude,
		Status:    models.AgentStatusCreated,
		Config:    req.Config,
		CreatedAt: time.Now(),
	}

	// 컨테이너 생성
	prebuiltContainer, err := p.containerPool.AcquireContainer(agent.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire container: %w", err)
	}

	// PrebuiltContainer를 WorkspaceContainer로 변환
	container := &docker.WorkspaceContainer{
		ID:          prebuiltContainer.ID,
		Name:        fmt.Sprintf("agent-%s", agent.ID),
		WorkspaceID: agent.ID,
		Created:     prebuiltContainer.CreatedAt,
	}

	// PooledAgent 생성
	pooledAgent := &PooledAgent{
		ID:           agent.ID,
		Agent:        agent,
		Container:    container,
		Status:       PooledAgentStatusInUse,
		CreatedAt:    time.Now(),
		LastUsedAt:   time.Now(),
		CreationTime: time.Since(start),
		UsageCount:   1,
	}

	atomic.AddInt64(&p.totalCreated, 1)
	p.runningAgents.Store(pooledAgent.ID, pooledAgent)

	return pooledAgent, nil
}

// startWorkers 워커 고루틴들 시작
func (p *AdvancedAgentPool) startWorkers() {
	// 사전 생성 워커들
	for i := 0; i < p.config.PreCreateWorkers; i++ {
		p.wg.Add(1)
		go p.preCreateWorkerLoop()
	}

	// 재활용 워커들
	for i := 0; i < p.config.RecycleWorkers; i++ {
		p.wg.Add(1)
		go p.recycleWorkerLoop()
	}
}

// preCreateWorkerLoop 사전 생성 워커 루프
func (p *AdvancedAgentPool) preCreateWorkerLoop() {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			return

		case <-p.preCreateWorker:
			// 새 에이전트 사전 생성
			agent, err := p.createNewAgent(context.Background(), AgentRequest{
				ProjectID: "pool-warmup",
				Name:      "pool-agent",
				Config:    models.AgentConfig{},
			})

			if err != nil {
				continue
			}

			agent.mu.Lock()
			agent.Status = PooledAgentStatusAvailable
			agent.mu.Unlock()

			// 사용 가능한 풀에 추가
			select {
			case p.availableAgents <- agent:
			case <-p.ctx.Done():
				p.destroyAgent(agent)
				return
			}
		}
	}
}

// recycleWorkerLoop 재활용 워커 루프
func (p *AdvancedAgentPool) recycleWorkerLoop() {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			return

		case agent := <-p.recycleWorker:
			// 에이전트 재활용 처리
			if err := p.recycleAgent(agent); err != nil {
				p.destroyAgent(agent)
			} else {
				atomic.AddInt64(&p.metrics.TotalRecycled, 1)
				
				agent.mu.Lock()
				agent.Status = PooledAgentStatusAvailable
				agent.mu.Unlock()

				// 재활용된 에이전트를 풀에 추가
				select {
				case p.availableAgents <- agent:
				case <-p.ctx.Done():
					p.destroyAgent(agent)
					return
				}
			}
		}
	}
}

// warmUpPool 풀 워밍업
func (p *AdvancedAgentPool) warmUpPool() {
	for i := 0; i < p.config.WarmUpSize; i++ {
		select {
		case p.preCreateWorker <- struct{}{}:
		case <-p.ctx.Done():
			return
		}
	}
}

// shouldRecycleAgent 에이전트 재활용 가능 여부 확인
func (p *AdvancedAgentPool) shouldRecycleAgent(agent *PooledAgent) bool {
	// 사용 횟수가 너무 많은 경우
	if agent.UsageCount > 100 {
		return false
	}

	// 유휴 시간이 너무 긴 경우
	if time.Since(agent.LastUsedAt) > p.config.MaxIdleTime {
		return false
	}

	// 메모리 사용량이 너무 높은 경우
	if agent.PeakMemory > 512*1024*1024 { // 512MB
		return false
	}

	return true
}

// recycleAgent 에이전트 재활용
func (p *AdvancedAgentPool) recycleAgent(agent *PooledAgent) error {
	agent.mu.Lock()
	defer agent.mu.Unlock()

	// 컨테이너 재시작 및 정리
	if err := p.containerPool.ReleaseContainer(agent.Container.ID); err != nil {
		return fmt.Errorf("failed to recycle container: %w", err)
	}

	// 에이전트 상태 리셋
	agent.Agent.Status = models.AgentStatusCreated
	agent.Agent.ErrorMessage = ""
	agent.LastUsedAt = time.Now()

	return nil
}

// destroyAgent 에이전트 삭제
func (p *AdvancedAgentPool) destroyAgent(agent *PooledAgent) {
	if agent == nil {
		return
	}

	agent.mu.Lock()
	agent.Status = PooledAgentStatusDestroying
	agent.mu.Unlock()

	// 컨테이너 반환
	if agent.Container != nil {
		p.containerPool.ReleaseContainer(agent.Container.ID)
	}

	// 실행 중인 에이전트 목록에서 제거
	p.runningAgents.Delete(agent.ID)

	atomic.AddInt64(&p.totalDestroyed, 1)
}

// destroyAllAgents 모든 에이전트 삭제
func (p *AdvancedAgentPool) destroyAllAgents() {
	// 사용 가능한 에이전트들 정리
	for {
		select {
		case agent := <-p.availableAgents:
			p.destroyAgent(agent)
		default:
			goto cleanup_running
		}
	}

cleanup_running:
	// 실행 중인 에이전트들 정리
	p.runningAgents.Range(func(key, value interface{}) bool {
		if agent, ok := value.(*PooledAgent); ok {
			p.destroyAgent(agent)
		}
		return true
	})
}

// autoScalingLoop 자동 스케일링 루프
func (p *AdvancedAgentPool) autoScalingLoop() {
	ticker := time.NewTicker(p.config.ScaleCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return

		case <-ticker.C:
			p.checkAndScale()
		}
	}
}

// checkAndScale 스케일링 체크 및 실행
func (p *AdvancedAgentPool) checkAndScale() {
	availableCount := len(p.availableAgents)
	runningCount := 0
	p.runningAgents.Range(func(key, value interface{}) bool {
		runningCount++
		return true
	})

	totalCount := availableCount + runningCount
	utilization := float64(runningCount) / float64(totalCount)

	// 스케일 업 조건
	if utilization > p.config.ScaleUpThreshold && totalCount < p.config.MaxSize {
		scaleUpCount := min(p.config.WarmUpSize, p.config.MaxSize-totalCount)
		for i := 0; i < scaleUpCount; i++ {
			select {
			case p.preCreateWorker <- struct{}{}:
			default:
				break
			}
		}
	}

	// 스케일 다운 조건
	if utilization < p.config.ScaleDownThreshold && totalCount > p.config.MinSize {
		scaleDownCount := min(availableCount/2, totalCount-p.config.MinSize)
		for i := 0; i < scaleDownCount; i++ {
			select {
			case agent := <-p.availableAgents:
				go p.destroyAgent(agent)
			default:
				break
			}
		}
	}
}

// cleanupLoop 정리 작업 루프
func (p *AdvancedAgentPool) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return

		case <-ticker.C:
			p.cleanupIdleAgents()
			runtime.GC() // 강제 가비지 컬렉션
		}
	}
}

// cleanupIdleAgents 유휴 에이전트 정리
func (p *AdvancedAgentPool) cleanupIdleAgents() {
	p.runningAgents.Range(func(key, value interface{}) bool {
		if agent, ok := value.(*PooledAgent); ok {
			agent.mu.RLock()
			idle := time.Since(agent.LastUsedAt) > p.config.IdleTimeout
			agent.mu.RUnlock()

			if idle {
				p.destroyAgent(agent)
			}
		}
		return true
	})
}

// updateMetrics 메트릭 업데이트
func (p *AdvancedAgentPool) updateMetrics() {
	availableCount := len(p.availableAgents)
	runningCount := 0
	p.runningAgents.Range(func(key, value interface{}) bool {
		runningCount++
		return true
	})

	p.metrics.AvailableAgents.Set(float64(availableCount))
	p.metrics.RunningAgents.Set(float64(runningCount))
	p.metrics.PoolSize.Set(float64(availableCount + runningCount))

	if p.metrics.PoolHits+p.metrics.PoolMisses > 0 {
		hitRate := float64(p.metrics.PoolHits) / float64(p.metrics.PoolHits+p.metrics.PoolMisses)
		p.metrics.PoolHitRate.Set(hitRate)
	}
}

// GetStats 풀 통계 조회
func (p *AdvancedAgentPool) GetStats() PoolStats {
	availableCount := len(p.availableAgents)
	runningCount := 0
	p.runningAgents.Range(func(key, value interface{}) bool {
		runningCount++
		return true
	})

	return PoolStats{
		TotalSize:      availableCount + runningCount,
		AvailableCount: availableCount,
		RunningCount:   runningCount,
		TotalCreated:   atomic.LoadInt64(&p.totalCreated),
		TotalDestroyed: atomic.LoadInt64(&p.totalDestroyed),
		TotalRecycled:  atomic.LoadInt64(&p.metrics.TotalRecycled),
		PoolHits:       atomic.LoadInt64(&p.metrics.PoolHits),
		PoolMisses:     atomic.LoadInt64(&p.metrics.PoolMisses),
	}
}

// AgentRequest 에이전트 요청
type AgentRequest struct {
	ProjectID string              `json:"project_id"`
	Name      string              `json:"name"`
	Config    models.AgentConfig  `json:"config"`
}

// PoolStats 풀 통계
type PoolStats struct {
	TotalSize      int   `json:"total_size"`
	AvailableCount int   `json:"available_count"`
	RunningCount   int   `json:"running_count"`
	TotalCreated   int64 `json:"total_created"`
	TotalDestroyed int64 `json:"total_destroyed"`
	TotalRecycled  int64 `json:"total_recycled"`
	PoolHits       int64 `json:"pool_hits"`
	PoolMisses     int64 `json:"pool_misses"`
}

// 유틸리티 함수들
func generateAgentID() string {
	return fmt.Sprintf("agent-%d", time.Now().UnixNano())
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// NewPoolMetrics 새 풀 메트릭 생성
func NewPoolMetrics() *PoolMetrics {
	return &PoolMetrics{
		PoolSize: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "agent_pool_size_total",
			Help: "Total number of agents in the pool",
		}),
		AvailableAgents: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "agent_pool_available_total",
			Help: "Number of available agents in the pool",
		}),
		RunningAgents: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "agent_pool_running_total",
			Help: "Number of running agents",
		}),
		CreationTime: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "agent_pool_creation_duration_seconds",
			Help:    "Time taken to create/acquire an agent",
			Buckets: prometheus.DefBuckets,
		}),
		AgentUsageTime: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "agent_pool_usage_duration_seconds",
			Help:    "Time an agent was in use",
			Buckets: prometheus.DefBuckets,
		}),
		MemoryUsage: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "agent_pool_memory_usage_bytes",
			Help: "Memory usage of agent pool",
		}),
		RecycleRate: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "agent_pool_recycle_total",
			Help: "Total number of agent recycles",
		}),
		PoolHitRate: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "agent_pool_hit_rate",
			Help: "Pool hit rate (0-1)",
		}),
	}
}

// DefaultAdvancedPoolConfig 기본 고급 풀 설정 반환
func DefaultAdvancedPoolConfig() AdvancedPoolConfig {
	return AdvancedPoolConfig{
		MinSize:             5,
		MaxSize:             100,
		WarmUpSize:          10,
		CreationTimeout:     30 * time.Second,
		IdleTimeout:         10 * time.Minute,
		MaxIdleTime:         30 * time.Minute,
		RecycleWorkers:      3,
		PreCreateWorkers:    2,
		MaxMemoryPerAgent:   "512Mi",
		MaxCPUPerAgent:      "0.5",
		EnableAutoScaling:   true,
		ScaleUpThreshold:    0.8,
		ScaleDownThreshold:  0.3,
		ScaleCheckInterval:  30 * time.Second,
	}
}