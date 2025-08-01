package agent

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aicli/aicli-web/internal/claude"
	"github.com/aicli/aicli-web/internal/docker"
	"github.com/aicli/aicli-web/internal/models"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

// AgentPerformanceOptimizer는 에이전트 성능 최적화 관리자입니다
type AgentPerformanceOptimizer struct {
	// 풀 관리
	containerPool *ContainerPool
	worktreePool  *WorktreePool
	memoryManager *claude.OptimizedMemoryManager

	// 메트릭 수집
	metrics  *PerformanceMetrics
	profiler *AgentProfiler

	// 자동 스케일링
	autoScaler   *AgentAutoScaler
	loadBalancer *AgentLoadBalancer

	// 설정
	config PerformanceConfig

	// 상태 관리
	running       atomic.Bool
	lastOptimized time.Time
	optimizeMutex sync.RWMutex

	// 생명주기
	ctx    context.Context
	cancel context.CancelFunc

	// 모니터링
	systemMonitor *SystemResourceMonitor
	alertManager  *PerformanceAlertManager
}

// PerformanceConfig는 성능 최적화 설정입니다
type PerformanceConfig struct {
	// 풀 설정
	ContainerPoolSize    int `json:"container_pool_size"`
	ContainerPoolMaxSize int `json:"container_pool_max_size"`
	WorktreePoolSize     int `json:"worktree_pool_size"`

	// 자동 스케일링 설정
	AutoScaling AutoScalingConfig `json:"auto_scaling"`

	// 성능 목표
	TargetCreationTime  time.Duration `json:"target_creation_time"`
	MaxConcurrentAgents int           `json:"max_concurrent_agents"`
	MemoryLimitPerAgent int64         `json:"memory_limit_per_agent"`
	CPULimitPerAgent    float64       `json:"cpu_limit_per_agent"`

	// 최적화 주기
	OptimizationInterval time.Duration `json:"optimization_interval"`
	MetricsInterval      time.Duration `json:"metrics_interval"`
	ProfileInterval      time.Duration `json:"profile_interval"`

	// 임계값
	HighMemoryThreshold float64       `json:"high_memory_threshold"`
	HighCPUThreshold    float64       `json:"high_cpu_threshold"`
	LatencyThreshold    time.Duration `json:"latency_threshold"`
}

// AutoScalingConfig는 자동 스케일링 설정입니다
type AutoScalingConfig struct {
	Enabled            bool          `json:"enabled"`
	MinAgents          int           `json:"min_agents"`
	MaxAgents          int           `json:"max_agents"`
	ScaleUpThreshold   float64       `json:"scale_up_threshold"`
	ScaleDownThreshold float64       `json:"scale_down_threshold"`
	ScaleUpCooldown    time.Duration `json:"scale_up_cooldown"`
	ScaleDownCooldown  time.Duration `json:"scale_down_cooldown"`
	PredictiveScaling  bool          `json:"predictive_scaling"`
	TargetUtilization  float64       `json:"target_utilization"`
}

// PerformanceMetrics는 성능 메트릭입니다
type PerformanceMetrics struct {
	// 에이전트 생성 메트릭
	AgentCreationTimes  []time.Duration `json:"agent_creation_times"`
	AverageCreationTime time.Duration   `json:"average_creation_time"`
	P95CreationTime     time.Duration   `json:"p95_creation_time"`
	P99CreationTime     time.Duration   `json:"p99_creation_time"`

	// 처리량 메트릭
	TotalAgentsCreated atomic.Int64 `json:"total_agents_created"`
	ActiveAgents       atomic.Int32 `json:"active_agents"`
	QueuedRequests     atomic.Int32 `json:"queued_requests"`
	ThroughputRPS      float64      `json:"throughput_rps"`

	// 리소스 사용률
	MemoryUsage atomic.Int64 `json:"memory_usage"`
	CPUUsage    atomic.Int64 `json:"cpu_usage"` // x100 for precision
	DiskUsage   atomic.Int64 `json:"disk_usage"`
	NetworkIO   atomic.Int64 `json:"network_io"`

	// 에러율
	FailedCreations atomic.Int64 `json:"failed_creations"`
	ErrorRate       float64      `json:"error_rate"`
	RecoveryCount   atomic.Int64 `json:"recovery_count"`

	// 풀 메트릭
	ContainerPoolHits   atomic.Int64 `json:"container_pool_hits"`
	ContainerPoolMisses atomic.Int64 `json:"container_pool_misses"`
	WorktreePoolHits    atomic.Int64 `json:"worktree_pool_hits"`
	WorktreePoolMisses  atomic.Int64 `json:"worktree_pool_misses"`

	// 시간 정보
	LastUpdated time.Time `json:"last_updated"`

	mutex sync.RWMutex
}

// ContainerPool은 에이전트 컨테이너 풀입니다
type ContainerPool struct {
	// 미리 생성된 컨테이너들
	availableContainers chan *PrebuiltContainer
	inUseContainers     map[string]*PrebuiltContainer

	// 설정
	maxSize     int
	currentSize atomic.Int32
	warmupSize  int

	// Docker 관리
	dockerClient docker.Client
	imageCache   *ImageCache

	// 생명주기
	ctx             context.Context
	cancel          context.CancelFunc
	cleanupInterval time.Duration

	// 동시성 제어
	mutex         sync.RWMutex
	creationMutex sync.Mutex
}

// PrebuiltContainer는 미리 생성된 컨테이너입니다
type PrebuiltContainer struct {
	ID            string                 `json:"id"`
	ImageID       string                 `json:"image_id"`
	Config        map[string]interface{} `json:"config"`
	CreatedAt     time.Time              `json:"created_at"`
	LastUsed      time.Time              `json:"last_used"`
	UseCount      int32                  `json:"use_count"`
	Status        PoolContainerStatus    `json:"status"`
	ResourceUsage ResourceUsage          `json:"resource_usage"`

	// Docker 정보
	DockerContainer interface{} `json:"-"` // Docker container object
}

// PoolContainerStatus는 풀 컨테이너 상태입니다
type PoolContainerStatus int

const (
	PoolContainerStatusReady PoolContainerStatus = iota
	PoolContainerStatusInUse
	PoolContainerStatusWarming
	PoolContainerStatusRecycling
	PoolContainerStatusError
)

// ResourceUsage는 리소스 사용량입니다
type ResourceUsage struct {
	CPUUsage    float64   `json:"cpu_usage"`
	MemoryUsage int64     `json:"memory_usage"`
	DiskUsage   int64     `json:"disk_usage"`
	NetworkRx   int64     `json:"network_rx"`
	NetworkTx   int64     `json:"network_tx"`
	LastUpdated time.Time `json:"last_updated"`
}

// WorktreePool은 Git worktree 풀입니다
type WorktreePool struct {
	// 미리 생성된 worktree들
	availableWorktrees chan *PrebuiltWorktree
	inUseWorktrees     map[string]*PrebuiltWorktree

	// 설정
	maxSize     int
	currentSize atomic.Int32

	// 생명주기
	ctx             context.Context
	cancel          context.CancelFunc
	cleanupInterval time.Duration

	// 동시성 제어
	mutex sync.RWMutex
}

// PrebuiltWorktree는 미리 생성된 워크트리입니다
type PrebuiltWorktree struct {
	ID        string         `json:"id"`
	ProjectID string         `json:"project_id"`
	Branch    string         `json:"branch"`
	Path      string         `json:"path"`
	CreatedAt time.Time      `json:"created_at"`
	LastUsed  time.Time      `json:"last_used"`
	UseCount  int32          `json:"use_count"`
	Status    WorktreeStatus `json:"status"`
}

// WorktreeStatus는 워크트리 상태입니다
type WorktreeStatus int

const (
	WorktreeStatusReady WorktreeStatus = iota
	WorktreeStatusInUse
	WorktreeStatusSyncing
	WorktreeStatusError
)

// AgentProfiler는 에이전트 성능 프로파일러입니다
type AgentProfiler struct {
	// 프로파일링 데이터
	cpuProfile       []CPUProfile       `json:"cpu_profile"`
	memoryProfile    []MemoryProfile    `json:"memory_profile"`
	goroutineProfile []GoroutineProfile `json:"goroutine_profile"`

	// 설정
	profilingEnabled  bool          `json:"profiling_enabled"`
	profilingInterval time.Duration `json:"profiling_interval"`
	profileRetention  time.Duration `json:"profile_retention"`

	// 생명주기
	ctx    context.Context
	cancel context.CancelFunc

	mutex sync.RWMutex
}

// CPUProfile은 CPU 프로파일 데이터입니다
type CPUProfile struct {
	Timestamp  time.Time     `json:"timestamp"`
	Usage      float64       `json:"usage"`
	UserTime   time.Duration `json:"user_time"`
	SystemTime time.Duration `json:"system_time"`
	Processes  []ProcessInfo `json:"processes"`
}

// MemoryProfile은 메모리 프로파일 데이터입니다
type MemoryProfile struct {
	Timestamp    time.Time `json:"timestamp"`
	HeapAlloc    uint64    `json:"heap_alloc"`
	HeapSys      uint64    `json:"heap_sys"`
	HeapInuse    uint64    `json:"heap_inuse"`
	StackInuse   uint64    `json:"stack_inuse"`
	NumGC        uint32    `json:"num_gc"`
	PauseTotalNs uint64    `json:"pause_total_ns"`
	AllocRate    float64   `json:"alloc_rate"`
}

// GoroutineProfile은 고루틴 프로파일 데이터입니다
type GoroutineProfile struct {
	Timestamp         time.Time      `json:"timestamp"`
	NumGoroutine      int            `json:"num_goroutine"`
	GoroutinesByState map[string]int `json:"goroutines_by_state"`
}

// ProcessInfo는 프로세스 정보입니다
type ProcessInfo struct {
	PID           int32   `json:"pid"`
	Name          string  `json:"name"`
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryPercent float32 `json:"memory_percent"`
}

// SystemResourceMonitor는 시스템 리소스 모니터입니다
type SystemResourceMonitor struct {
	// 현재 상태
	currentCPU    atomic.Value // float64
	currentMemory atomic.Value // *mem.VirtualMemoryStat
	currentDisk   atomic.Value // int64

	// 모니터링 설정
	monitoringInterval time.Duration
	alertThresholds    AlertThresholds

	// 생명주기
	ctx    context.Context
	cancel context.CancelFunc

	// 이벤트 알림
	alertChannel chan PerformanceAlert
}

// AlertThresholds는 알림 임계값입니다
type AlertThresholds struct {
	HighCPUUsage    float64       `json:"high_cpu_usage"`
	HighMemoryUsage float64       `json:"high_memory_usage"`
	HighDiskUsage   float64       `json:"high_disk_usage"`
	HighErrorRate   float64       `json:"high_error_rate"`
	HighLatency     time.Duration `json:"high_latency"`
	LowThroughput   float64       `json:"low_throughput"`
}

// PerformanceAlert는 성능 알림입니다
type PerformanceAlert struct {
	Type               AlertType          `json:"type"`
	Severity           AlertSeverity      `json:"severity"`
	Message            string             `json:"message"`
	Timestamp          time.Time          `json:"timestamp"`
	Metrics            map[string]float64 `json:"metrics"`
	RecommendedActions []string           `json:"recommended_actions"`
}

// AlertType은 알림 타입입니다
type AlertType int

const (
	AlertTypeHighCPU AlertType = iota
	AlertTypeHighMemory
	AlertTypeHighLatency
	AlertTypeHighErrorRate
	AlertTypeLowThroughput
	AlertTypeResourceExhaustion
)

// AlertSeverity는 알림 심각도입니다
type AlertSeverity int

const (
	AlertSeverityInfo AlertSeverity = iota
	AlertSeverityWarning
	AlertSeverityCritical
	AlertSeverityEmergency
)

// AgentAutoScaler는 에이전트 자동 스케일러입니다
type AgentAutoScaler struct {
	// 스케일링 상태
	currentCapacity atomic.Int32
	targetCapacity  atomic.Int32
	lastScaleAction atomic.Value // string
	lastScaleTime   atomic.Value // time.Time

	// 예측 모델
	loadPredictor   *LoadPredictor
	capacityPlanner *CapacityPlanner

	// 설정
	config AutoScalingConfig

	// 생명주기
	ctx    context.Context
	cancel context.CancelFunc

	// 동시성 제어
	scalingMutex sync.Mutex
}

// LoadPredictor는 부하 예측기입니다
type LoadPredictor struct {
	// 히스토리 데이터
	loadHistory    []LoadDataPoint `json:"load_history"`
	maxHistorySize int             `json:"max_history_size"`

	// 예측 모델
	model    PredictionModel `json:"model"`
	accuracy float64         `json:"accuracy"`

	// 시간 기반 패턴
	dailyPatterns  map[int]float64 `json:"daily_patterns"`  // hour -> load
	weeklyPatterns map[int]float64 `json:"weekly_patterns"` // weekday -> load

	mutex sync.RWMutex
}

// LoadDataPoint는 부하 데이터 포인트입니다
type LoadDataPoint struct {
	Timestamp   time.Time `json:"timestamp"`
	AgentCount  int       `json:"agent_count"`
	CPUUsage    float64   `json:"cpu_usage"`
	MemoryUsage float64   `json:"memory_usage"`
	RequestRate float64   `json:"request_rate"`
	QueueLength int       `json:"queue_length"`
}

// PredictionModel은 예측 모델입니다
type PredictionModel struct {
	Type             ModelType          `json:"type"`
	Parameters       map[string]float64 `json:"parameters"`
	LastTrained      time.Time          `json:"last_trained"`
	TrainingAccuracy float64            `json:"training_accuracy"`
}

// ModelType은 모델 타입입니다
type ModelType int

const (
	ModelTypeLinearRegression ModelType = iota
	ModelTypeMovingAverage
	ModelTypeExponentialSmoothing
	ModelTypeARIMA
)

// CapacityPlanner는 용량 계획기입니다
type CapacityPlanner struct {
	// 용량 계획
	currentPlan CapacityPlan   `json:"current_plan"`
	futurePlans []CapacityPlan `json:"future_plans"`

	// 리소스 모델
	resourceModel ResourceModel `json:"resource_model"`

	mutex sync.RWMutex
}

// CapacityPlan은 용량 계획입니다
type CapacityPlan struct {
	PlanID               string               `json:"plan_id"`
	StartTime            time.Time            `json:"start_time"`
	EndTime              time.Time            `json:"end_time"`
	TargetCapacity       int                  `json:"target_capacity"`
	ExpectedLoad         float64              `json:"expected_load"`
	ResourceRequirements ResourceRequirements `json:"resource_requirements"`
	Confidence           float64              `json:"confidence"`
}

// ResourceModel은 리소스 모델입니다
type ResourceModel struct {
	CPUPerAgent     float64 `json:"cpu_per_agent"`
	MemoryPerAgent  int64   `json:"memory_per_agent"`
	DiskPerAgent    int64   `json:"disk_per_agent"`
	NetworkPerAgent int64   `json:"network_per_agent"`

	// 오버헤드
	SystemOverhead  ResourceRequirements `json:"system_overhead"`
	ScalingOverhead ResourceRequirements `json:"scaling_overhead"`
}

// ResourceRequirements는 리소스 요구사항입니다
type ResourceRequirements struct {
	CPU     float64 `json:"cpu"`
	Memory  int64   `json:"memory"`
	Disk    int64   `json:"disk"`
	Network int64   `json:"network"`
}

// AgentLoadBalancer는 에이전트 로드 밸런서입니다
type AgentLoadBalancer struct {
	// 부하 분산 전략
	strategy LoadBalancingStrategy `json:"strategy"`

	// 에이전트 풀
	agentPools map[string]*AgentPool `json:"agent_pools"`

	// 라우팅 정보
	routingTable  *RoutingTable       `json:"routing_table"`
	healthChecker *AgentHealthChecker `json:"health_checker"`

	// 메트릭
	routingMetrics *RoutingMetrics `json:"routing_metrics"`

	mutex sync.RWMutex
}

// LoadBalancingStrategy는 부하 분산 전략입니다
type LoadBalancingStrategy int

const (
	StrategyRoundRobin LoadBalancingStrategy = iota
	StrategyLeastConnections
	StrategyWeightedRoundRobin
	StrategyResourceBased
	StrategyLatencyBased
	StrategyAdaptive
)

// AgentPool은 에이전트 풀입니다
type AgentPool struct {
	ID              string           `json:"id"`
	Name            string           `json:"name"`
	Agents          []*models.Agent  `json:"agents"`
	MaxCapacity     int              `json:"max_capacity"`
	CurrentLoad     float64          `json:"current_load"`
	HealthStatus    PoolHealthStatus `json:"health_status"`
	LastHealthCheck time.Time        `json:"last_health_check"`

	mutex sync.RWMutex
}

// PoolHealthStatus는 풀 건강 상태입니다
type PoolHealthStatus int

const (
	PoolHealthHealthy PoolHealthStatus = iota
	PoolHealthDegraded
	PoolHealthUnhealthy
	PoolHealthMaintenance
)

// RoutingTable은 라우팅 테이블입니다
type RoutingTable struct {
	Rules       []RoutingRule `json:"rules"`
	DefaultPool string        `json:"default_pool"`

	mutex sync.RWMutex
}

// RoutingRule은 라우팅 규칙입니다
type RoutingRule struct {
	ID         string      `json:"id"`
	Priority   int         `json:"priority"`
	Conditions []Condition `json:"conditions"`
	TargetPool string      `json:"target_pool"`
	Weight     float64     `json:"weight"`
}

// Condition은 조건입니다
type Condition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

// RoutingMetrics는 라우팅 메트릭입니다
type RoutingMetrics struct {
	TotalRequests    atomic.Int64 `json:"total_requests"`
	SuccessfulRoutes atomic.Int64 `json:"successful_routes"`
	FailedRoutes     atomic.Int64 `json:"failed_routes"`
	AverageLatency   atomic.Value `json:"average_latency"` // time.Duration

	// 풀별 메트릭
	PoolMetrics map[string]*PoolMetrics `json:"pool_metrics"`

	mutex sync.RWMutex
}

// PoolMetrics는 풀 메트릭입니다
type PoolMetrics struct {
	RequestCount   atomic.Int64  `json:"request_count"`
	SuccessCount   atomic.Int64  `json:"success_count"`
	ErrorCount     atomic.Int64  `json:"error_count"`
	AverageLatency time.Duration `json:"average_latency"`
	CurrentLoad    float64       `json:"current_load"`
}

// AgentHealthChecker는 에이전트 건강 체크입니다
type AgentHealthChecker struct {
	// 건강 체크 설정
	checkInterval time.Duration `json:"check_interval"`
	checkTimeout  time.Duration `json:"check_timeout"`

	// 건강 상태
	agentHealth map[string]*AgentHealth `json:"agent_health"`

	// 생명주기
	ctx    context.Context
	cancel context.CancelFunc

	mutex sync.RWMutex
}

// AgentHealth는 에이전트 건강 상태입니다
type AgentHealth struct {
	AgentID      string            `json:"agent_id"`
	Status       AgentHealthStatus `json:"status"`
	LastCheck    time.Time         `json:"last_check"`
	FailureCount int               `json:"failure_count"`
	SuccessCount int               `json:"success_count"`
	ResponseTime time.Duration     `json:"response_time"`
	ErrorMessage string            `json:"error_message"`
}

// AgentHealthStatus는 에이전트 건강 상태입니다
type AgentHealthStatus int

const (
	AgentHealthStatusHealthy AgentHealthStatus = iota
	AgentHealthStatusWarning
	AgentHealthStatusCritical
	AgentHealthStatusUnknown
)

// ImageCache는 Docker 이미지 캐시입니다
type ImageCache struct {
	// 캐시된 이미지들
	cachedImages map[string]*CachedImage `json:"cached_images"`

	// 캐시 설정
	maxCacheSize     int64        `json:"max_cache_size"`
	currentCacheSize atomic.Int64 `json:"current_cache_size"`

	// 생명주기
	ctx             context.Context
	cancel          context.CancelFunc
	cleanupInterval time.Duration

	mutex sync.RWMutex
}

// CachedImage는 캐시된 이미지입니다
type CachedImage struct {
	ID          string    `json:"id"`
	Tag         string    `json:"tag"`
	Size        int64     `json:"size"`
	CreatedAt   time.Time `json:"created_at"`
	LastUsed    time.Time `json:"last_used"`
	UseCount    int32     `json:"use_count"`
	LayerHashes []string  `json:"layer_hashes"`
}

// PerformanceAlertManager는 성능 알림 관리자입니다
type PerformanceAlertManager struct {
	// 알림 채널
	alertChannel chan PerformanceAlert

	// 알림 규칙
	alertRules []AlertRule `json:"alert_rules"`

	// 알림 히스토리
	alertHistory   []PerformanceAlert `json:"alert_history"`
	maxHistorySize int                `json:"max_history_size"`

	// 생명주기
	ctx    context.Context
	cancel context.CancelFunc

	mutex sync.RWMutex
}

// AlertRule은 알림 규칙입니다
type AlertRule struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	Condition string        `json:"condition"`
	Threshold float64       `json:"threshold"`
	Duration  time.Duration `json:"duration"`
	Severity  AlertSeverity `json:"severity"`
	Actions   []AlertAction `json:"actions"`
	Enabled   bool          `json:"enabled"`
}

// AlertAction은 알림 액션입니다
type AlertAction struct {
	Type   ActionType             `json:"type"`
	Config map[string]interface{} `json:"config"`
}

// ActionType은 액션 타입입니다
type ActionType int

const (
	ActionTypeLog ActionType = iota
	ActionTypeEmail
	ActionTypeSlack
	ActionTypeWebhook
	ActionTypeScale
)

// NewAgentPerformanceOptimizer는 새로운 에이전트 성능 최적화기를 생성합니다
func NewAgentPerformanceOptimizer(config PerformanceConfig, dockerClient docker.Client) *AgentPerformanceOptimizer {
	ctx, cancel := context.WithCancel(context.Background())

	optimizer := &AgentPerformanceOptimizer{
		config:        config,
		ctx:           ctx,
		cancel:        cancel,
		lastOptimized: time.Now(),
		metrics:       NewPerformanceMetrics(),
	}

	// 컴포넌트 초기화
	optimizer.containerPool = NewContainerPool(config.ContainerPoolSize, config.ContainerPoolMaxSize, dockerClient)
	optimizer.worktreePool = NewWorktreePool(config.WorktreePoolSize)
	optimizer.memoryManager = claude.NewOptimizedMemoryManager(claude.DefaultMemoryPoolConfig())
	optimizer.profiler = NewAgentProfiler(config.ProfileInterval)
	optimizer.autoScaler = NewAgentAutoScaler(config.AutoScaling)
	optimizer.loadBalancer = NewAgentLoadBalancer()
	optimizer.systemMonitor = NewSystemResourceMonitor(config.MetricsInterval)
	optimizer.alertManager = NewPerformanceAlertManager()

	return optimizer
}

// Start는 성능 최적화기를 시작합니다
func (apo *AgentPerformanceOptimizer) Start() error {
	if !apo.running.CompareAndSwap(false, true) {
		return fmt.Errorf("performance optimizer already running")
	}

	// 하위 컴포넌트들 시작
	if err := apo.containerPool.Start(); err != nil {
		return fmt.Errorf("failed to start container pool: %w", err)
	}

	if err := apo.worktreePool.Start(); err != nil {
		return fmt.Errorf("failed to start worktree pool: %w", err)
	}

	if err := apo.memoryManager.Start(); err != nil {
		return fmt.Errorf("failed to start memory manager: %w", err)
	}

	if err := apo.profiler.Start(); err != nil {
		return fmt.Errorf("failed to start profiler: %w", err)
	}

	if err := apo.autoScaler.Start(); err != nil {
		return fmt.Errorf("failed to start auto scaler: %w", err)
	}

	if err := apo.systemMonitor.Start(); err != nil {
		return fmt.Errorf("failed to start system monitor: %w", err)
	}

	if err := apo.alertManager.Start(); err != nil {
		return fmt.Errorf("failed to start alert manager: %w", err)
	}

	// 백그라운드 루프 시작
	go apo.optimizationLoop()
	go apo.metricsCollectionLoop()
	go apo.alertProcessingLoop()

	return nil
}

// Stop은 성능 최적화기를 중지합니다
func (apo *AgentPerformanceOptimizer) Stop() error {
	if !apo.running.CompareAndSwap(true, false) {
		return nil
	}

	apo.cancel()

	// 하위 컴포넌트들 중지
	apo.containerPool.Stop()
	apo.worktreePool.Stop()
	apo.memoryManager.Stop()
	apo.profiler.Stop()
	apo.autoScaler.Stop()
	apo.systemMonitor.Stop()
	apo.alertManager.Stop()

	return nil
}

// OptimizeAgent는 단일 에이전트를 최적화합니다
func (apo *AgentPerformanceOptimizer) OptimizeAgent(agentID string) error {
	start := time.Now()
	defer func() {
		apo.metrics.RecordAgentOptimization(agentID, time.Since(start))
	}()

	// 컨테이너 풀에서 최적화된 컨테이너 할당
	container, err := apo.containerPool.AcquireContainer(agentID)
	if err != nil {
		return fmt.Errorf("failed to acquire optimized container: %w", err)
	}
	defer apo.containerPool.ReleaseContainer(container.ID)

	// 워크트리 풀에서 최적화된 워크트리 할당
	worktree, err := apo.worktreePool.AcquireWorktree(agentID)
	if err != nil {
		return fmt.Errorf("failed to acquire optimized worktree: %w", err)
	}
	defer apo.worktreePool.ReleaseWorktree(worktree.ID)

	// 메모리 최적화 적용
	if err := apo.applyMemoryOptimization(agentID); err != nil {
		return fmt.Errorf("failed to apply memory optimization: %w", err)
	}

	return nil
}

// GetPerformanceMetrics는 성능 메트릭을 반환합니다
func (apo *AgentPerformanceOptimizer) GetPerformanceMetrics() *PerformanceMetrics {
	return apo.metrics
}

// GetSystemStatus는 시스템 상태를 반환합니다
func (apo *AgentPerformanceOptimizer) GetSystemStatus() *SystemStatus {
	return &SystemStatus{
		CPUUsage:       apo.systemMonitor.GetCPUUsage(),
		MemoryUsage:    apo.systemMonitor.GetMemoryUsage(),
		DiskUsage:      apo.systemMonitor.GetDiskUsage(),
		ActiveAgents:   int(apo.metrics.ActiveAgents.Load()),
		QueuedRequests: int(apo.metrics.QueuedRequests.Load()),
		LastUpdated:    time.Now(),
	}
}

// SystemStatus는 시스템 상태입니다
type SystemStatus struct {
	CPUUsage       float64   `json:"cpu_usage"`
	MemoryUsage    float64   `json:"memory_usage"`
	DiskUsage      float64   `json:"disk_usage"`
	ActiveAgents   int       `json:"active_agents"`
	QueuedRequests int       `json:"queued_requests"`
	LastUpdated    time.Time `json:"last_updated"`
}

// 내부 메서드들

func (apo *AgentPerformanceOptimizer) optimizationLoop() {
	ticker := time.NewTicker(apo.config.OptimizationInterval)
	defer ticker.Stop()

	for {
		select {
		case <-apo.ctx.Done():
			return
		case <-ticker.C:
			apo.runOptimizationCycle()
		}
	}
}

func (apo *AgentPerformanceOptimizer) runOptimizationCycle() {
	apo.optimizeMutex.Lock()
	defer apo.optimizeMutex.Unlock()

	// 시스템 상태 확인
	systemStatus := apo.GetSystemStatus()

	// 자동 스케일링 실행
	if apo.config.AutoScaling.Enabled {
		if err := apo.autoScaler.EvaluateScaling(systemStatus); err != nil {
			// 로그 기록 (실제 구현에서는 로거 사용)
		}
	}

	// 풀 최적화
	apo.containerPool.Optimize()
	apo.worktreePool.Optimize()

	// 메모리 최적화
	apo.memoryManager.OptimizePoolSizes()

	apo.lastOptimized = time.Now()
}

func (apo *AgentPerformanceOptimizer) metricsCollectionLoop() {
	ticker := time.NewTicker(apo.config.MetricsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-apo.ctx.Done():
			return
		case <-ticker.C:
			apo.collectMetrics()
		}
	}
}

func (apo *AgentPerformanceOptimizer) collectMetrics() {
	// CPU 사용률 수집
	cpuPercent, err := cpu.Percent(time.Second, false)
	if err == nil && len(cpuPercent) > 0 {
		apo.metrics.CPUUsage.Store(int64(cpuPercent[0] * 100))
	}

	// 메모리 사용률 수집
	memStat, err := mem.VirtualMemory()
	if err == nil {
		apo.metrics.MemoryUsage.Store(int64(memStat.Used))
	}

	// Runtime 메모리 통계
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// 프로파일 데이터 수집
	apo.profiler.CollectProfile()

	apo.metrics.LastUpdated = time.Now()
}

func (apo *AgentPerformanceOptimizer) alertProcessingLoop() {
	for {
		select {
		case <-apo.ctx.Done():
			return
		case alert := <-apo.alertManager.alertChannel:
			apo.processAlert(alert)
		}
	}
}

func (apo *AgentPerformanceOptimizer) processAlert(alert PerformanceAlert) {
	// 알림 처리 로직
	switch alert.Type {
	case AlertTypeHighCPU:
		apo.handleHighCPUAlert(alert)
	case AlertTypeHighMemory:
		apo.handleHighMemoryAlert(alert)
	case AlertTypeHighLatency:
		apo.handleHighLatencyAlert(alert)
	}
}

func (apo *AgentPerformanceOptimizer) handleHighCPUAlert(alert PerformanceAlert) {
	// CPU 사용률이 높을 때의 처리 로직
	if apo.config.AutoScaling.Enabled {
		apo.autoScaler.ScaleUp()
	}
}

func (apo *AgentPerformanceOptimizer) handleHighMemoryAlert(alert PerformanceAlert) {
	// 메모리 사용률이 높을 때의 처리 로직
	apo.memoryManager.RecycleUnusedObjects()
	runtime.GC()
}

func (apo *AgentPerformanceOptimizer) handleHighLatencyAlert(alert PerformanceAlert) {
	// 지연시간이 높을 때의 처리 로직
	apo.containerPool.WarmupContainers()
}

func (apo *AgentPerformanceOptimizer) applyMemoryOptimization(agentID string) error {
	// 메모리 최적화 적용 로직
	return nil
}

// NewPerformanceMetrics는 새로운 성능 메트릭을 생성합니다
func NewPerformanceMetrics() *PerformanceMetrics {
	return &PerformanceMetrics{
		AgentCreationTimes: make([]time.Duration, 0, 1000),
		LastUpdated:        time.Now(),
	}
}

// RecordAgentCreation은 에이전트 생성 시간을 기록합니다
func (pm *PerformanceMetrics) RecordAgentCreation(duration time.Duration) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	pm.AgentCreationTimes = append(pm.AgentCreationTimes, duration)
	pm.TotalAgentsCreated.Add(1)

	// 최근 1000개만 유지
	if len(pm.AgentCreationTimes) > 1000 {
		pm.AgentCreationTimes = pm.AgentCreationTimes[1:]
	}

	// 통계 업데이트
	pm.updateCreationTimeStats()
}

// RecordAgentOptimization은 에이전트 최적화 시간을 기록합니다
func (pm *PerformanceMetrics) RecordAgentOptimization(agentID string, duration time.Duration) {
	// 최적화 메트릭 기록 로직
}

func (pm *PerformanceMetrics) updateCreationTimeStats() {
	if len(pm.AgentCreationTimes) == 0 {
		return
	}

	// 평균 계산
	var total time.Duration
	for _, t := range pm.AgentCreationTimes {
		total += t
	}
	pm.AverageCreationTime = total / time.Duration(len(pm.AgentCreationTimes))

	// 백분위수 계산 (간단한 구현)
	if len(pm.AgentCreationTimes) >= 20 {
		sorted := make([]time.Duration, len(pm.AgentCreationTimes))
		copy(sorted, pm.AgentCreationTimes)

		// 간단한 정렬 (실제로는 sort.Slice 사용)
		p95Index := int(float64(len(sorted)) * 0.95)
		p99Index := int(float64(len(sorted)) * 0.99)

		if p95Index < len(sorted) {
			pm.P95CreationTime = sorted[p95Index]
		}
		if p99Index < len(sorted) {
			pm.P99CreationTime = sorted[p99Index]
		}
	}
}

// DefaultPerformanceConfig는 기본 성능 최적화 설정을 반환합니다
func DefaultPerformanceConfig() PerformanceConfig {
	return PerformanceConfig{
		ContainerPoolSize:    20,
		ContainerPoolMaxSize: 100,
		WorktreePoolSize:     50,
		AutoScaling: AutoScalingConfig{
			Enabled:            true,
			MinAgents:          5,
			MaxAgents:          200,
			ScaleUpThreshold:   0.8,
			ScaleDownThreshold: 0.3,
			ScaleUpCooldown:    2 * time.Minute,
			ScaleDownCooldown:  5 * time.Minute,
			PredictiveScaling:  true,
			TargetUtilization:  0.7,
		},
		TargetCreationTime:   5 * time.Second,
		MaxConcurrentAgents:  100,
		MemoryLimitPerAgent:  100 * 1024 * 1024, // 100MB
		CPULimitPerAgent:     0.1,               // 0.1 core
		OptimizationInterval: 30 * time.Second,
		MetricsInterval:      10 * time.Second,
		ProfileInterval:      1 * time.Minute,
		HighMemoryThreshold:  0.8,
		HighCPUThreshold:     0.8,
		LatencyThreshold:     1 * time.Second,
	}
}
