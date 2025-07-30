package claude

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aicli/aicli-web/internal/docker"
	"github.com/aicli/aicli-web/internal/models"
)

// PerformanceManager는 통합 성능 관리 시스템입니다
type PerformanceManager struct {
	// 코어 컴포넌트들
	profiler           *PerformanceProfiler
	agentPoolManager   *AgentPoolManager
	imageOptimizer     *docker.ImageOptimizer
	concurrencyController *ConcurrencyController

	// 통합 설정
	config             PerformanceManagerConfig

	// 통합 메트릭 및 통계
	metrics            *IntegratedMetrics
	metricsMu          sync.RWMutex

	// 성능 대시보드
	dashboard          *PerformanceDashboard

	// 자동 최적화 엔진
	optimizer          *AutoOptimizationEngine

	// 알림 및 모니터링
	alertManager       *AlertManager
	healthChecker      *SystemHealthChecker

	// 성능 목표 및 SLA
	slaManager         *SLAManager

	// 생명주기 관리
	running            atomic.Bool
	ctx                context.Context
	cancel             context.CancelFunc
	wg                 sync.WaitGroup
}

// PerformanceManagerConfig는 성능 관리자 설정입니다
type PerformanceManagerConfig struct {
	// 성능 목표
	TargetAgentCreationTime  time.Duration `json:"target_agent_creation_time"`
	TargetConcurrentAgents   int           `json:"target_concurrent_agents"`
	TargetThroughputRPS      float64       `json:"target_throughput_rps"`
	TargetLatencyP95         time.Duration `json:"target_latency_p95"`
	
	// 시스템 제한
	MaxCPUUsage              float64       `json:"max_cpu_usage"`
	MaxMemoryUsage           float64       `json:"max_memory_usage"`
	MaxDiskUsage             float64       `json:"max_disk_usage"`
	
	// 모니터링 설정
	MetricsCollectionInterval time.Duration `json:"metrics_collection_interval"`
	HealthCheckInterval       time.Duration `json:"health_check_interval"`
	AlertCheckInterval        time.Duration `json:"alert_check_interval"`
	
	// 최적화 설정
	EnableAutoOptimization    bool          `json:"enable_auto_optimization"`
	OptimizationInterval      time.Duration `json:"optimization_interval"`
	OptimizationAggression    int           `json:"optimization_aggression"` // 1-10
	
	// 보고 설정
	EnableReporting           bool          `json:"enable_reporting"`
	ReportingInterval         time.Duration `json:"reporting_interval"`
	ReportRetentionDays       int           `json:"report_retention_days"`
	
	// SLA 설정
	EnableSLAMonitoring       bool          `json:"enable_sla_monitoring"`
	SLAViolationThreshold     int           `json:"sla_violation_threshold"`
	
	// 대시보드 설정
	EnableDashboard           bool          `json:"enable_dashboard"`
	DashboardUpdateInterval   time.Duration `json:"dashboard_update_interval"`
}

// IntegratedMetrics는 통합 성능 메트릭입니다
type IntegratedMetrics struct {
	// 에이전트 성능
	AgentMetrics           AgentPerformanceMetrics `json:"agent_metrics"`
	
	// 시스템 성능
	SystemMetrics          SystemPerformanceMetrics `json:"system_metrics"`
	
	// Docker 성능
	DockerMetrics          DockerPerformanceMetrics `json:"docker_metrics"`
	
	// 동시성 성능
	ConcurrencyMetrics     ConcurrencyPerformanceMetrics `json:"concurrency_metrics"`
	
	// 전체 성능 지표
	OverallPerformance     OverallPerformanceMetrics `json:"overall_performance"`
	
	// 수집 시간
	CollectedAt            time.Time `json:"collected_at"`
	CollectionDuration     time.Duration `json:"collection_duration"`
}

// AgentPerformanceMetrics는 에이전트 성능 메트릭입니다
type AgentPerformanceMetrics struct {
	// 생성 성능
	AverageCreationTime    time.Duration `json:"average_creation_time"`
	CreationTimeP50        time.Duration `json:"creation_time_p50"`
	CreationTimeP95        time.Duration `json:"creation_time_p95"`
	CreationTimeP99        time.Duration `json:"creation_time_p99"`
	
	// 활용률
	PoolUtilization        float64       `json:"pool_utilization"`
	CacheHitRate           float64       `json:"cache_hit_rate"`
	
	// 처리량
	AgentsCreatedPerSecond float64       `json:"agents_created_per_second"`
	TasksProcessedPerSecond float64      `json:"tasks_processed_per_second"`
	
	// 에러율
	CreationErrorRate      float64       `json:"creation_error_rate"`
	TaskErrorRate          float64       `json:"task_error_rate"`
}

// SystemPerformanceMetrics는 시스템 성능 메트릭입니다
type SystemPerformanceMetrics struct {
	// CPU 메트릭
	CPUUsagePercent        float64       `json:"cpu_usage_percent"`
	CPULoadAverage         []float64     `json:"cpu_load_average"`
	
	// 메모리 메트릭
	MemoryUsagePercent     float64       `json:"memory_usage_percent"`
	MemoryUsageBytes       int64         `json:"memory_usage_bytes"`
	GCPauseTime            time.Duration `json:"gc_pause_time"`
	
	// I/O 메트릭
	DiskIOReadBPS          int64         `json:"disk_io_read_bps"`
	DiskIOWriteBPS         int64         `json:"disk_io_write_bps"`
	NetworkIOInBPS         int64         `json:"network_io_in_bps"`
	NetworkIOOutBPS        int64         `json:"network_io_out_bps"`
	
	// 고루틴 메트릭
	GoroutineCount         int           `json:"goroutine_count"`
	ThreadCount            int           `json:"thread_count"`
}

// DockerPerformanceMetrics는 Docker 성능 메트릭입니다
type DockerPerformanceMetrics struct {
	// 빌드 성능
	AverageBuildTime       time.Duration `json:"average_build_time"`
	BuildCacheHitRate      float64       `json:"build_cache_hit_rate"`
	ImageCacheHitRate      float64       `json:"image_cache_hit_rate"`
	
	// 최적화 성능
	AverageOptimizationRatio float64     `json:"average_optimization_ratio"`
	BytesSavedTotal        int64         `json:"bytes_saved_total"`
	TimeSavedTotal         time.Duration `json:"time_saved_total"`
	
	// 컨테이너 성능
	ContainerStartTime     time.Duration `json:"container_start_time"`
	ContainerStopTime      time.Duration `json:"container_stop_time"`
}

// ConcurrencyPerformanceMetrics는 동시성 성능 메트릭입니다
type ConcurrencyPerformanceMetrics struct {
	// 동시성 지표
	ActiveOperations       int           `json:"active_operations"`
	QueuedOperations       int           `json:"queued_operations"`
	MaxConcurrency         int           `json:"max_concurrency"`
	
	// 처리 성능
	ThroughputRPS          float64       `json:"throughput_rps"`
	AverageLatency         time.Duration `json:"average_latency"`
	LatencyP95             time.Duration `json:"latency_p95"`
	
	// 제한 및 제어
	RateLimitHitRate       float64       `json:"rate_limit_hit_rate"`
	CircuitBreakerTrips    int64         `json:"circuit_breaker_trips"`
	BackpressureEvents     int64         `json:"backpressure_events"`
}

// OverallPerformanceMetrics는 전체 성능 메트릭입니다
type OverallPerformanceMetrics struct {
	// 종합 점수
	PerformanceScore       float64       `json:"performance_score"` // 0-100
	EfficiencyScore        float64       `json:"efficiency_score"`  // 0-100
	ReliabilityScore       float64       `json:"reliability_score"` // 0-100
	
	// SLA 준수
	SLAComplianceRate      float64       `json:"sla_compliance_rate"`
	TargetAchievementRate  float64       `json:"target_achievement_rate"`
	
	// 개선 여지
	OptimizationPotential  float64       `json:"optimization_potential"`
	RecommendedActions     []string      `json:"recommended_actions"`
}

// PerformanceDashboard는 성능 대시보드입니다
type PerformanceDashboard struct {
	// 실시간 데이터
	RealTimeMetrics        *IntegratedMetrics `json:"real_time_metrics"`
	
	// 히스토리컬 데이터
	MetricHistory          []IntegratedMetrics `json:"metric_history"`
	
	// 차트 데이터
	ChartData              map[string][]ChartPoint `json:"chart_data"`
	
	// 알림 및 이벤트
	ActiveAlerts           []PerformanceAlert `json:"active_alerts"`
	RecentEvents           []PerformanceEvent `json:"recent_events"`
	
	// 상태 정보
	SystemStatus           SystemStatus `json:"system_status"`
	ComponentHealth        map[string]ComponentHealth `json:"component_health"`
	
	// 업데이트 정보
	LastUpdated            time.Time `json:"last_updated"`
	UpdateFrequency        time.Duration `json:"update_frequency"`
}

// ChartPoint는 차트 데이터 포인트입니다
type ChartPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
	Label     string    `json:"label,omitempty"`
}

// SystemStatus는 시스템 상태입니다
type SystemStatus string

const (
	SystemStatusHealthy    SystemStatus = "healthy"
	SystemStatusWarning    SystemStatus = "warning"
	SystemStatusCritical   SystemStatus = "critical"
	SystemStatusDegraded   SystemStatus = "degraded"
)

// ComponentHealth는 컴포넌트 건강 상태입니다
type ComponentHealth struct {
	Status        HealthStatus `json:"status"`
	LastCheck     time.Time    `json:"last_check"`
	Message       string       `json:"message"`
	ResponseTime  time.Duration `json:"response_time"`
	ErrorCount    int64        `json:"error_count"`
}

// HealthStatus는 건강 상태입니다
type HealthStatus string

const (
	HealthStatusHealthy    HealthStatus = "healthy"
	HealthStatusWarning    HealthStatus = "warning"
	HealthStatusUnhealthy  HealthStatus = "unhealthy"
	HealthStatusUnknown    HealthStatus = "unknown"
)

// AutoOptimizationEngine은 자동 최적화 엔진입니다
type AutoOptimizationEngine struct {
	manager           *PerformanceManager
	
	// 최적화 전략들
	strategies        []OptimizationStrategy
	
	// 최적화 이력
	optimizations     []OptimizationRecord
	optimizationsMu   sync.RWMutex
	
	// 학습 모델
	learningModel     *PerformanceLearningModel
	
	// 설정
	config            OptimizationEngineConfig
	
	// 생명주기
	running           atomic.Bool
	ctx               context.Context
	cancel            context.CancelFunc
}

// OptimizationEngineConfig는 최적화 엔진 설정입니다
type OptimizationEngineConfig struct {
	EnableLearning        bool          `json:"enable_learning"`
	LearningWindow        time.Duration `json:"learning_window"`
	MinDataPoints         int           `json:"min_data_points"`
	ConfidenceThreshold   float64       `json:"confidence_threshold"`
	MaxOptimizationsPerHour int         `json:"max_optimizations_per_hour"`
}

// OptimizationRecord는 최적화 기록입니다
type OptimizationRecord struct {
	ID                string                 `json:"id"`
	Timestamp         time.Time              `json:"timestamp"`
	Strategy          string                 `json:"strategy"`
	TargetComponent   string                 `json:"target_component"`
	BeforeMetrics     *IntegratedMetrics     `json:"before_metrics"`
	AfterMetrics      *IntegratedMetrics     `json:"after_metrics"`
	ImprovementScore  float64                `json:"improvement_score"`
	Success           bool                   `json:"success"`
	ErrorMessage      string                 `json:"error_message,omitempty"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
}

// PerformanceLearningModel은 성능 학습 모델입니다
type PerformanceLearningModel struct {
	// 패턴 인식
	patterns          []PerformancePattern
	
	// 예측 모델
	predictions       map[string][]PerformancePrediction
	
	// 학습 데이터
	trainingData      []TrainingDataPoint
	
	// 모델 정확도
	accuracy          float64
	lastTraining      time.Time
}

// PerformancePattern은 성능 패턴입니다
type PerformancePattern struct {
	Name              string    `json:"name"`
	Description       string    `json:"description"`
	Conditions        []string  `json:"conditions"`
	Frequency         int       `json:"frequency"`
	Confidence        float64   `json:"confidence"`
	LastSeen          time.Time `json:"last_seen"`
}

// TrainingDataPoint는 학습 데이터 포인트입니다
type TrainingDataPoint struct {
	Timestamp         time.Time          `json:"timestamp"`
	Input             map[string]float64 `json:"input"`
	Output            map[string]float64 `json:"output"`
	Context           string             `json:"context"`
}

// AlertManager는 알림 관리자입니다
type AlertManager struct {
	// 알림 규칙들
	rules             []AlertRule
	rulesMu           sync.RWMutex
	
	// 활성 알림들
	activeAlerts      map[string]*PerformanceAlert
	alertsMu          sync.RWMutex
	
	// 알림 이력
	alertHistory      []PerformanceAlert
	historyMu         sync.RWMutex
	
	// 알림 채널들
	channels          []AlertChannel
	
	// 설정
	config            AlertManagerConfig
}

// AlertRule은 알림 규칙입니다
type AlertRule struct {
	ID                string          `json:"id"`
	Name              string          `json:"name"`
	Description       string          `json:"description"`
	MetricName        string          `json:"metric_name"`
	Operator          string          `json:"operator"` // >, <, >=, <=, ==, !=
	Threshold         float64         `json:"threshold"`
	Duration          time.Duration   `json:"duration"`
	Severity          AlertSeverity   `json:"severity"`
	Enabled           bool            `json:"enabled"`
	LastTriggered     time.Time       `json:"last_triggered"`
	TriggerCount      int64           `json:"trigger_count"`
}

// AlertChannel은 알림 채널입니다
type AlertChannel interface {
	SendAlert(alert *PerformanceAlert) error
	GetType() string
	IsEnabled() bool
}

// AlertManagerConfig는 알림 관리자 설정입니다
type AlertManagerConfig struct {
	MaxActiveAlerts   int           `json:"max_active_alerts"`
	AlertRetention    time.Duration `json:"alert_retention"`
	EnableGrouping    bool          `json:"enable_grouping"`
	GroupingWindow    time.Duration `json:"grouping_window"`
	EnableSuppression bool          `json:"enable_suppression"`
	SuppressionWindow time.Duration `json:"suppression_window"`
}

// SystemHealthChecker는 시스템 건강 상태 검사기입니다
type SystemHealthChecker struct {
	manager           *PerformanceManager
	
	// 건강 상태 검사들
	healthChecks      []HealthCheck
	checksMu          sync.RWMutex
	
	// 건강 상태 이력
	healthHistory     []HealthSnapshot
	historyMu         sync.RWMutex
	
	// 설정
	config            HealthCheckerConfig
}

// HealthCheck는 건강 상태 검사입니다
type HealthCheck interface {
	Name() string
	Check(ctx context.Context) (HealthStatus, string, error)
	Timeout() time.Duration
	IsEnabled() bool
}

// HealthSnapshot은 건강 상태 스냅샷입니다
type HealthSnapshot struct {
	Timestamp         time.Time                    `json:"timestamp"`
	OverallStatus     HealthStatus                 `json:"overall_status"`
	ComponentStatuses map[string]ComponentHealth   `json:"component_statuses"`
	Issues            []HealthIssue                `json:"issues"`
}

// HealthIssue는 건강 상태 이슈입니다
type HealthIssue struct {
	Component         string       `json:"component"`
	Severity          AlertSeverity `json:"severity"`
	Message           string       `json:"message"`
	DetectedAt        time.Time    `json:"detected_at"`
	ResolvedAt        *time.Time   `json:"resolved_at,omitempty"`
}

// HealthCheckerConfig는 건강 상태 검사기 설정입니다
type HealthCheckerConfig struct {
	CheckInterval     time.Duration `json:"check_interval"`
	CheckTimeout      time.Duration `json:"check_timeout"`
	RetryAttempts     int           `json:"retry_attempts"`
	RetryDelay        time.Duration `json:"retry_delay"`
	HistoryRetention  time.Duration `json:"history_retention"`
}

// SLAManager는 SLA 관리자입니다
type SLAManager struct {
	// SLA 정의들
	slas              []SLADefinition
	slasMu            sync.RWMutex
	
	// SLA 상태 추적
	slaStatus         map[string]*SLAStatus
	statusMu          sync.RWMutex
	
	// 위반 이력
	violations        []SLAViolation
	violationsMu      sync.RWMutex
	
	// 설정
	config            SLAManagerConfig
}

// SLADefinition은 SLA 정의입니다
type SLADefinition struct {
	ID                string        `json:"id"`
	Name              string        `json:"name"`
	Description       string        `json:"description"`
	MetricName        string        `json:"metric_name"`
	Target            float64       `json:"target"`
	Operator          string        `json:"operator"`
	MeasurementWindow time.Duration `json:"measurement_window"`
	ComplianceTarget  float64       `json:"compliance_target"` // 99.9%
	Enabled           bool          `json:"enabled"`
}

// SLAStatus는 SLA 상태입니다
type SLAStatus struct {
	SLAID             string        `json:"sla_id"`
	CurrentValue      float64       `json:"current_value"`
	ComplianceRate    float64       `json:"compliance_rate"`
	LastMeasurement   time.Time     `json:"last_measurement"`
	Status            SLAStatusType `json:"status"`
	ViolationCount    int64         `json:"violation_count"`
}

// SLAStatusType은 SLA 상태 유형입니다
type SLAStatusType string

const (
	SLAStatusCompliant    SLAStatusType = "compliant"
	SLAStatusAtRisk       SLAStatusType = "at_risk"
	SLAStatusViolating    SLAStatusType = "violating"
	SLAStatusUnknown      SLAStatusType = "unknown"
)

// SLAViolation은 SLA 위반입니다
type SLAViolation struct {
	ID                string                 `json:"id"`
	SLAID             string                 `json:"sla_id"`
	Timestamp         time.Time              `json:"timestamp"`
	Duration          time.Duration          `json:"duration"`
	Severity          ViolationSeverity      `json:"severity"`
	ActualValue       float64                `json:"actual_value"`
	TargetValue       float64                `json:"target_value"`
	Impact            string                 `json:"impact"`
	RootCause         string                 `json:"root_cause,omitempty"`
	Resolution        string                 `json:"resolution,omitempty"`
	ResolvedAt        *time.Time             `json:"resolved_at,omitempty"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
}

// ViolationSeverity는 위반 심각도입니다
type ViolationSeverity string

const (
	ViolationSeverityMinor    ViolationSeverity = "minor"
	ViolationSeverityMajor    ViolationSeverity = "major"
	ViolationSeverityCritical ViolationSeverity = "critical"
)

// SLAManagerConfig는 SLA 관리자 설정입니다
type SLAManagerConfig struct {
	MeasurementInterval   time.Duration `json:"measurement_interval"`
	ViolationThreshold    int           `json:"violation_threshold"`
	AlertOnViolation      bool          `json:"alert_on_violation"`
	ViolationRetention    time.Duration `json:"violation_retention"`
}

// DefaultPerformanceManagerConfig는 기본 성능 관리자 설정을 반환합니다
func DefaultPerformanceManagerConfig() PerformanceManagerConfig {
	return PerformanceManagerConfig{
		TargetAgentCreationTime:   5 * time.Second,
		TargetConcurrentAgents:    100,
		TargetThroughputRPS:       100.0,
		TargetLatencyP95:          100 * time.Millisecond,
		MaxCPUUsage:               80.0,
		MaxMemoryUsage:            80.0,
		MaxDiskUsage:              80.0,
		MetricsCollectionInterval: 30 * time.Second,
		HealthCheckInterval:       1 * time.Minute,
		AlertCheckInterval:        30 * time.Second,
		EnableAutoOptimization:    true,
		OptimizationInterval:      5 * time.Minute,
		OptimizationAggression:    5,
		EnableReporting:           true,
		ReportingInterval:         1 * time.Hour,
		ReportRetentionDays:       30,
		EnableSLAMonitoring:       true,
		SLAViolationThreshold:     3,
		EnableDashboard:           true,
		DashboardUpdateInterval:   10 * time.Second,
	}
}

// NewPerformanceManager는 새로운 성능 관리자를 생성합니다
func NewPerformanceManager(config PerformanceManagerConfig) *PerformanceManager {
	ctx, cancel := context.WithCancel(context.Background())
	
	manager := &PerformanceManager{
		config:    config,
		metrics:   &IntegratedMetrics{},
		ctx:       ctx,
		cancel:    cancel,
	}
	
	// 컴포넌트 초기화
	manager.dashboard = NewPerformanceDashboard()
	manager.optimizer = NewAutoOptimizationEngine(manager)
	manager.alertManager = NewAlertManager()
	manager.healthChecker = NewSystemHealthChecker(manager)
	manager.slaManager = NewSLAManager()
	
	return manager
}

// Start는 성능 관리자를 시작합니다
func (pm *PerformanceManager) Start() error {
	if !pm.running.CompareAndSwap(false, true) {
		return fmt.Errorf("performance manager is already running")
	}
	
	// 코어 컴포넌트 시작
	if pm.profiler != nil {
		if err := pm.profiler.Start(); err != nil {
			return fmt.Errorf("failed to start profiler: %w", err)
		}
	}
	
	if pm.concurrencyController != nil {
		if err := pm.concurrencyController.Start(); err != nil {
			return fmt.Errorf("failed to start concurrency controller: %w", err)
		}
	}
	
	if pm.agentPoolManager != nil {
		if err := pm.agentPoolManager.Start(); err != nil {
			return fmt.Errorf("failed to start agent pool manager: %w", err)
		}
	}
	
	if pm.imageOptimizer != nil {
		if err := pm.imageOptimizer.Start(); err != nil {
			return fmt.Errorf("failed to start image optimizer: %w", err)
		}
	}
	
	// 서브 컴포넌트 시작
	if pm.config.EnableAutoOptimization {
		if err := pm.optimizer.Start(); err != nil {
			return fmt.Errorf("failed to start optimizer: %w", err)
		}
	}
	
	if err := pm.alertManager.Start(); err != nil {
		return fmt.Errorf("failed to start alert manager: %w", err)
	}
	
	if err := pm.healthChecker.Start(); err != nil {
		return fmt.Errorf("failed to start health checker: %w", err)
	}
	
	if pm.config.EnableSLAMonitoring {
		if err := pm.slaManager.Start(); err != nil {
			return fmt.Errorf("failed to start SLA manager: %w", err)
		}
	}
	
	// 백그라운드 작업 시작
	pm.wg.Add(4)
	go pm.metricsCollectionLoop()
	go pm.dashboardUpdateLoop()
	go pm.reportingLoop()
	go pm.maintenanceLoop()
	
	return nil
}

// Stop은 성능 관리자를 중지합니다
func (pm *PerformanceManager) Stop() error {
	if !pm.running.CompareAndSwap(true, false) {
		return nil
	}
	
	// 백그라운드 작업 중지
	pm.cancel()
	pm.wg.Wait()
	
	// 컴포넌트들 중지
	if pm.slaManager != nil {
		pm.slaManager.Stop()
	}
	if pm.healthChecker != nil {
		pm.healthChecker.Stop()
	}
	if pm.alertManager != nil {
		pm.alertManager.Stop()
	}
	if pm.optimizer != nil {
		pm.optimizer.Stop()
	}
	if pm.imageOptimizer != nil {
		pm.imageOptimizer.Stop()
	}
	if pm.agentPoolManager != nil {
		pm.agentPoolManager.Stop()
	}
	if pm.concurrencyController != nil {
		pm.concurrencyController.Stop()
	}
	if pm.profiler != nil {
		pm.profiler.Stop()
	}
	
	return nil
}

// GetIntegratedMetrics는 통합 메트릭을 반환합니다
func (pm *PerformanceManager) GetIntegratedMetrics() *IntegratedMetrics {
	pm.metricsMu.RLock()
	defer pm.metricsMu.RUnlock()
	
	// 메트릭 복사본 생성
	metrics := *pm.metrics
	return &metrics
}

// GetDashboard는 성능 대시보드를 반환합니다
func (pm *PerformanceManager) GetDashboard() *PerformanceDashboard {
	return pm.dashboard.GetCurrentState()
}

// TriggerOptimization은 즉시 최적화를 실행합니다
func (pm *PerformanceManager) TriggerOptimization() error {
	if pm.optimizer == nil {
		return fmt.Errorf("optimizer is not available")
	}
	
	return pm.optimizer.ExecuteOptimization()
}

// SetComponents는 성능 관리자에 컴포넌트들을 설정합니다
func (pm *PerformanceManager) SetComponents(
	profiler *PerformanceProfiler,
	agentPoolManager *AgentPoolManager,
	imageOptimizer *docker.ImageOptimizer,
	concurrencyController *ConcurrencyController) {
	
	pm.profiler = profiler
	pm.agentPoolManager = agentPoolManager
	pm.imageOptimizer = imageOptimizer
	pm.concurrencyController = concurrencyController
}

// 내부 메서드들

func (pm *PerformanceManager) metricsCollectionLoop() {
	defer pm.wg.Done()
	
	ticker := time.NewTicker(pm.config.MetricsCollectionInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-pm.ctx.Done():
			return
		case <-ticker.C:
			pm.collectIntegratedMetrics()
		}
	}
}

func (pm *PerformanceManager) dashboardUpdateLoop() {
	defer pm.wg.Done()
	
	if !pm.config.EnableDashboard {
		return
	}
	
	ticker := time.NewTicker(pm.config.DashboardUpdateInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-pm.ctx.Done():
			return
		case <-ticker.C:
			pm.updateDashboard()
		}
	}
}

func (pm *PerformanceManager) reportingLoop() {
	defer pm.wg.Done()
	
	if !pm.config.EnableReporting {
		return
	}
	
	ticker := time.NewTicker(pm.config.ReportingInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-pm.ctx.Done():
			return
		case <-ticker.C:
			pm.generatePerformanceReport()
		}
	}
}

func (pm *PerformanceManager) maintenanceLoop() {
	defer pm.wg.Done()
	
	ticker := time.NewTicker(10 * time.Minute)
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

func (pm *PerformanceManager) collectIntegratedMetrics() {
	startTime := time.Now()
	
	metrics := &IntegratedMetrics{
		CollectedAt: startTime,
	}
	
	// 각 컴포넌트에서 메트릭 수집
	if pm.agentPoolManager != nil {
		pm.collectAgentMetrics(metrics)
	}
	
	if pm.profiler != nil {
		pm.collectSystemMetrics(metrics)
	}
	
	if pm.imageOptimizer != nil {
		pm.collectDockerMetrics(metrics)
	}
	
	if pm.concurrencyController != nil {
		pm.collectConcurrencyMetrics(metrics)
	}
	
	// 전체 성능 점수 계산
	pm.calculateOverallPerformance(metrics)
	
	metrics.CollectionDuration = time.Since(startTime)
	
	// 메트릭 저장
	pm.metricsMu.Lock()
	pm.metrics = metrics
	pm.metricsMu.Unlock()
	
	// 알림 규칙 확인
	if pm.alertManager != nil {
		pm.alertManager.CheckRules(metrics)
	}
	
	// SLA 확인
	if pm.slaManager != nil {
		pm.slaManager.CheckCompliance(metrics)
	}
}

func (pm *PerformanceManager) collectAgentMetrics(metrics *IntegratedMetrics) {
	if pm.agentPoolManager == nil {
		return
	}
	
	poolStats := pm.agentPoolManager.GetPoolStats()
	
	// 풀 활용률 계산
	var totalUtilization float64
	for _, typeStats := range poolStats.TypeStats {
		if typeStats.PoolSize > 0 {
			utilization := float64(typeStats.InUse) / float64(typeStats.PoolSize)
			totalUtilization += utilization
		}
	}
	
	if len(poolStats.TypeStats) > 0 {
		totalUtilization /= float64(len(poolStats.TypeStats))
	}
	
	metrics.AgentMetrics = AgentPerformanceMetrics{
		AverageCreationTime:    poolStats.AverageCreationTime,
		PoolUtilization:        totalUtilization,
		CacheHitRate:           poolStats.PoolHitRate,
		AgentsCreatedPerSecond: 0, // 계산 필요
		CreationErrorRate:      0, // 계산 필요
	}
}

func (pm *PerformanceManager) collectSystemMetrics(metrics *IntegratedMetrics) {
	if pm.profiler == nil {
		return
	}
	
	profilerMetrics := pm.profiler.GetCurrentMetrics()
	
	metrics.SystemMetrics = SystemPerformanceMetrics{
		CPUUsagePercent:    profilerMetrics.CPUUsagePercent,
		MemoryUsagePercent: profilerMetrics.MemoryUsagePercent,
		GoroutineCount:     profilerMetrics.GoroutineCount,
		ThreadCount:        profilerMetrics.ThreadCount,
	}
}

func (pm *PerformanceManager) collectDockerMetrics(metrics *IntegratedMetrics) {
	if pm.imageOptimizer == nil {
		return
	}
	
	optimizerStats := pm.imageOptimizer.GetOptimizationStats()
	
	metrics.DockerMetrics = DockerPerformanceMetrics{
		AverageBuildTime:         optimizerStats.AverageBuildTime,
		BuildCacheHitRate:        optimizerStats.CacheHitRate,
		AverageOptimizationRatio: optimizerStats.OptimizationRatio,
		BytesSavedTotal:          optimizerStats.BytesSaved,
		TimeSavedTotal:           optimizerStats.TimeSaved,
	}
}

func (pm *PerformanceManager) collectConcurrencyMetrics(metrics *IntegratedMetrics) {
	if pm.concurrencyController == nil {
		return
	}
	
	concurrencyStats := pm.concurrencyController.GetStats()
	
	metrics.ConcurrencyMetrics = ConcurrencyPerformanceMetrics{
		ActiveOperations:    concurrencyStats.ActiveOperations,
		QueuedOperations:    concurrencyStats.QueuedOperations,
		ThroughputRPS:       concurrencyStats.ThroughputPerSecond,
		AverageLatency:      concurrencyStats.AverageExecutionTime,
		RateLimitHitRate:    float64(concurrencyStats.RateLimitHits) / float64(concurrencyStats.RateLimitHits+concurrencyStats.RateLimitPasses),
		CircuitBreakerTrips: concurrencyStats.CircuitBreakerTrips,
		BackpressureEvents:  concurrencyStats.BackpressureEvents,
	}
}

func (pm *PerformanceManager) calculateOverallPerformance(metrics *IntegratedMetrics) {
	// 성능 점수 계산 (0-100)
	var performanceScore float64 = 100
	
	// CPU 사용률에 따른 감점
	if metrics.SystemMetrics.CPUUsagePercent > pm.config.MaxCPUUsage {
		performanceScore -= (metrics.SystemMetrics.CPUUsagePercent - pm.config.MaxCPUUsage) * 2
	}
	
	// 메모리 사용률에 따른 감점
	if metrics.SystemMetrics.MemoryUsagePercent > pm.config.MaxMemoryUsage {
		performanceScore -= (metrics.SystemMetrics.MemoryUsagePercent - pm.config.MaxMemoryUsage) * 2
	}
	
	// 대기열 상태에 따른 감점
	if metrics.ConcurrencyMetrics.QueuedOperations > 100 {
		performanceScore -= float64(metrics.ConcurrencyMetrics.QueuedOperations-100) * 0.1
	}
	
	if performanceScore < 0 {
		performanceScore = 0
	}
	
	// 효율성 점수 (캐시 히트율 기반)
	efficiencyScore := (metrics.AgentMetrics.CacheHitRate + metrics.DockerMetrics.BuildCacheHitRate) * 50
	
	// 신뢰성 점수 (에러율 기반)
	reliabilityScore := (1.0 - metrics.AgentMetrics.CreationErrorRate) * 100
	if reliabilityScore < 0 {
		reliabilityScore = 0
	}
	
	metrics.OverallPerformance = OverallPerformanceMetrics{
		PerformanceScore:  performanceScore,
		EfficiencyScore:   efficiencyScore,
		ReliabilityScore:  reliabilityScore,
		RecommendedActions: pm.generateRecommendations(metrics),
	}
}

func (pm *PerformanceManager) generateRecommendations(metrics *IntegratedMetrics) []string {
	var recommendations []string
	
	// CPU 사용률이 높은 경우
	if metrics.SystemMetrics.CPUUsagePercent > pm.config.MaxCPUUsage {
		recommendations = append(recommendations, "CPU 사용률이 높습니다. 동시성 제한을 줄이거나 스케일아웃을 고려하세요.")
	}
	
	// 메모리 사용률이 높은 경우
	if metrics.SystemMetrics.MemoryUsagePercent > pm.config.MaxMemoryUsage {
		recommendations = append(recommendations, "메모리 사용률이 높습니다. 메모리 누수를 확인하거나 가비지 컬렉션을 조정하세요.")
	}
	
	// 캐시 히트율이 낮은 경우
	if metrics.AgentMetrics.CacheHitRate < 0.7 {
		recommendations = append(recommendations, "에이전트 풀 캐시 히트율이 낮습니다. 풀 크기를 늘리거나 워밍업을 개선하세요.")
	}
	
	// 대기열이 쌓이는 경우
	if metrics.ConcurrencyMetrics.QueuedOperations > 50 {
		recommendations = append(recommendations, "대기열이 쌓이고 있습니다. 처리 용량을 늘리거나 우선순위를 조정하세요.")
	}
	
	return recommendations
}

func (pm *PerformanceManager) updateDashboard() {
	if pm.dashboard == nil {
		return
	}
	
	// 대시보드 업데이트 로직
	pm.dashboard.Update(pm.GetIntegratedMetrics())
}

func (pm *PerformanceManager) generatePerformanceReport() {
	// 성능 보고서 생성 로직
}

func (pm *PerformanceManager) performMaintenance() {
	// 유지보수 작업 수행
}

// 스텁 구현들 (실제 구현에서 완성 필요)

func NewPerformanceDashboard() *PerformanceDashboard {
	return &PerformanceDashboard{
		ChartData:       make(map[string][]ChartPoint),
		ActiveAlerts:    make([]PerformanceAlert, 0),
		RecentEvents:    make([]PerformanceEvent, 0),
		ComponentHealth: make(map[string]ComponentHealth),
	}
}

func (pd *PerformanceDashboard) Update(metrics *IntegratedMetrics) {
	pd.RealTimeMetrics = metrics
	pd.LastUpdated = time.Now()
}

func (pd *PerformanceDashboard) GetCurrentState() *PerformanceDashboard {
	return pd
}

func NewAutoOptimizationEngine(manager *PerformanceManager) *AutoOptimizationEngine {
	return &AutoOptimizationEngine{
		manager:       manager,
		optimizations: make([]OptimizationRecord, 0),
		learningModel: &PerformanceLearningModel{},
	}
}

func (aoe *AutoOptimizationEngine) Start() error { return nil }
func (aoe *AutoOptimizationEngine) Stop() error  { return nil }
func (aoe *AutoOptimizationEngine) ExecuteOptimization() error { return nil }

func NewAlertManager() *AlertManager {
	return &AlertManager{
		rules:        make([]AlertRule, 0),
		activeAlerts: make(map[string]*PerformanceAlert),
		alertHistory: make([]PerformanceAlert, 0),
		channels:     make([]AlertChannel, 0),
	}
}

func (am *AlertManager) Start() error { return nil }
func (am *AlertManager) Stop() error  { return nil }
func (am *AlertManager) CheckRules(metrics *IntegratedMetrics) {}

func NewSystemHealthChecker(manager *PerformanceManager) *SystemHealthChecker {
	return &SystemHealthChecker{
		manager:       manager,
		healthChecks:  make([]HealthCheck, 0),
		healthHistory: make([]HealthSnapshot, 0),
	}
}

func (shc *SystemHealthChecker) Start() error { return nil }
func (shc *SystemHealthChecker) Stop() error  { return nil }

func NewSLAManager() *SLAManager {
	return &SLAManager{
		slas:       make([]SLADefinition, 0),
		slaStatus:  make(map[string]*SLAStatus),
		violations: make([]SLAViolation, 0),
	}
}

func (sm *SLAManager) Start() error { return nil }
func (sm *SLAManager) Stop() error  { return nil }
func (sm *SLAManager) CheckCompliance(metrics *IntegratedMetrics) {}