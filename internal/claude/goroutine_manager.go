package claude

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// GoroutineManager는 고루틴 생명주기 관리 인터페이스입니다
type GoroutineManager interface {
	// 고루틴 풀 관리
	SpawnWorker(task Task) error
	SpawnBoundedWorker(task Task, timeout time.Duration) error
	
	// 리소스 추적
	GetActiveGoroutines() int
	GetGoroutineStats() GoroutineStats
	
	// 정리 및 최적화
	CleanupIdleWorkers() error
	SetMaxGoroutines(max int) error
	
	// 생명주기
	Start() error
	Stop() error
	
	// 고급 기능
	GetGoroutinePoolStatus() PoolStatus
	EnableGoroutineTracking(enabled bool)
	GetGoroutineLeaks() []GoroutineLeak
	ForceGarbageCollection() error
}

// GoroutineLifecycleManager 고루틴 생명주기 전용 관리자
type GoroutineLifecycleManager struct {
	// 기본 설정
	config         *GoroutineLifecycleConfig
	
	// 고루틴 추적
	activeGoroutines sync.Map // map[int64]*GoroutineInfo
	nextGoroutineID  int64
	
	// 메트릭 수집
	metrics          *GoroutineMetrics
	metricsCollector *MetricsCollector
	
	// 생명주기 관리
	lifecycleHooks   []LifecycleHook
	shutdownTimeout  time.Duration
	
	// 모니터링
	tracker          *GoroutineTracker
	leakDetector     *LeakDetector
	
	// 동기화
	mu               sync.RWMutex
	running          atomic.Bool
	ctx              context.Context
	cancel           context.CancelFunc
}

// GoroutineLifecycleConfig 고루틴 생명주기 설정
type GoroutineLifecycleConfig struct {
	// 기본 설정
	MaxGoroutines           int           `yaml:"max_goroutines"`
	DefaultTimeout          time.Duration `yaml:"default_timeout"`
	ShutdownTimeout         time.Duration `yaml:"shutdown_timeout"`
	
	// 추적 설정
	EnableTracking          bool          `yaml:"enable_tracking"`
	TrackingInterval        time.Duration `yaml:"tracking_interval"`
	StackTraceDepth         int           `yaml:"stack_trace_depth"`
	
	// 정리 설정
	CleanupInterval         time.Duration `yaml:"cleanup_interval"`
	IdleThreshold           time.Duration `yaml:"idle_threshold"`
	ForceCleanupAfter       time.Duration `yaml:"force_cleanup_after"`
	
	// 모니터링
	EnableMetrics           bool          `yaml:"enable_metrics"`
	MetricsInterval         time.Duration `yaml:"metrics_interval"`
	EnableLeakDetection     bool          `yaml:"enable_leak_detection"`
	LeakDetectionThreshold  time.Duration `yaml:"leak_detection_threshold"`
	
	// 성능 최적화
	EnableBatching          bool          `yaml:"enable_batching"`
	BatchSize               int           `yaml:"batch_size"`
	PreallocateWorkers      int           `yaml:"preallocate_workers"`
}

// GoroutineInfo 고루틴 정보
type GoroutineInfo struct {
	ID              int64                 `json:"id"`
	StartTime       time.Time             `json:"start_time"`
	LastActive      time.Time             `json:"last_active"`
	State           GoroutineState        `json:"state"`
	Task            Task                  `json:"task,omitempty"`
	StackTrace      []string              `json:"stack_trace,omitempty"`
	CPU             time.Duration         `json:"cpu_time"`
	Memory          int64                 `json:"memory_bytes"`
	Context         context.Context       `json:"-"`
	Cancel          context.CancelFunc    `json:"-"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// GoroutineState 고루틴 상태
type GoroutineState int

const (
	GoroutineStateCreated GoroutineState = iota
	GoroutineStateRunning
	GoroutineStateWaiting
	GoroutineStateCompleted
	GoroutineStateFailed
	GoroutineStateTerminated
)

// GoroutineMetrics 고루틴 메트릭
type GoroutineMetrics struct {
	// 카운터
	TotalCreated    int64 `json:"total_created"`
	TotalCompleted  int64 `json:"total_completed"`
	TotalFailed     int64 `json:"total_failed"`
	TotalTerminated int64 `json:"total_terminated"`
	
	// 현재 상태
	CurrentActive   int64 `json:"current_active"`
	CurrentIdle     int64 `json:"current_idle"`
	PeakActive      int64 `json:"peak_active"`
	
	// 성능 지표
	AvgLifetime     time.Duration `json:"avg_lifetime"`
	AvgCPUTime      time.Duration `json:"avg_cpu_time"`
	AvgMemoryUsage  int64         `json:"avg_memory_usage"`
	
	// 리소스 사용량
	TotalCPUTime    time.Duration `json:"total_cpu_time"`
	TotalMemory     int64         `json:"total_memory"`
	
	// 오류 통계
	PanicCount      int64         `json:"panic_count"`
	TimeoutCount    int64         `json:"timeout_count"`
	LeakCount       int64         `json:"leak_count"`
	
	// 시간 정보
	LastUpdate      time.Time     `json:"last_update"`
	CollectionStart time.Time     `json:"collection_start"`
}

// PoolStatus 풀 상태
type PoolStatus struct {
	Active       int               `json:"active"`
	Idle         int               `json:"idle"`
	Total        int               `json:"total"`
	MaxCapacity  int               `json:"max_capacity"`
	Utilization  float64           `json:"utilization"`
	Health       PoolHealth        `json:"health"`
	LastActivity time.Time         `json:"last_activity"`
}

// PoolHealth 풀 건강 상태
type PoolHealth string

const (
	PoolHealthy     PoolHealth = "healthy"
	PoolHealthWarning PoolHealth = "warning"
	PoolHealthCritical PoolHealth = "critical"
)

// GoroutineLeak 고루틴 누수 정보
type GoroutineLeak struct {
	GoroutineID   int64         `json:"goroutine_id"`
	StartTime     time.Time     `json:"start_time"`
	Duration      time.Duration `json:"duration"`
	LastActivity  time.Time     `json:"last_activity"`
	StackTrace    []string      `json:"stack_trace"`
	TaskInfo      interface{}   `json:"task_info,omitempty"`
	MemoryUsage   int64         `json:"memory_usage"`
	Severity      LeakSeverity  `json:"severity"`
}

// LeakSeverity 누수 심각도
type LeakSeverity string

const (
	LeakSeverityLow    LeakSeverity = "low"
	LeakSeverityMedium LeakSeverity = "medium"
	LeakSeverityHigh   LeakSeverity = "high"
	LeakSeverityCritical LeakSeverity = "critical"
)

// LifecycleHook 생명주기 훅
type LifecycleHook interface {
	OnGoroutineStart(info *GoroutineInfo) error
	OnGoroutineEnd(info *GoroutineInfo) error
	OnGoroutinePanic(info *GoroutineInfo, recovered interface{}) error
}

// WorkerPoolManager는 고루틴 풀 관리자입니다
type WorkerPoolManager struct {
	// 워커 풀
	workers      map[int]*Worker
	workersMutex sync.RWMutex
	nextWorkerID int32
	
	// 태스크 큐
	taskQueue    chan TaskWrapper
	priorityQueue chan TaskWrapper
	
	// 설정
	config WorkerPoolConfig
	
	// 스케일러
	scaler   *WorkerScaler
	monitor  *GoroutineMonitor
	
	// 통계
	stats    GoroutineStats
	statsMutex sync.RWMutex
	
	// 생명주기
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	running  atomic.Bool
}

// Task는 실행할 작업입니다
type Task interface {
	Execute(ctx context.Context) error
	GetPriority() TaskPriority
	GetEstimatedDuration() time.Duration
	GetDescription() string
}

// TaskWrapper는 태스크 래퍼입니다
type TaskWrapper struct {
	Task      Task
	StartTime time.Time
	Timeout   time.Duration
	ResultCh  chan TaskResult
	Ctx       context.Context
	Cancel    context.CancelFunc
}

// TaskResult는 태스크 실행 결과입니다
type TaskResult struct {
	Success   bool          `json:"success"`
	Error     error         `json:"error"`
	Duration  time.Duration `json:"duration"`
	StartTime time.Time     `json:"start_time"`
	EndTime   time.Time     `json:"end_time"`
}

// TaskPriority는 태스크 우선순위입니다
type TaskPriority int

const (
	TaskPriorityLow TaskPriority = iota
	TaskPriorityNormal
	TaskPriorityHigh
	TaskPriorityCritical
)

// Worker는 작업 실행자입니다
type Worker struct {
	ID        int           `json:"id"`
	State     WorkerState   `json:"state"`
	TaskQueue chan TaskWrapper `json:"-"`
	QuitChan  chan bool     `json:"-"`
	Pool      *WorkerPoolManager `json:"-"`
	
	// 통계
	TasksProcessed int64     `json:"tasks_processed"`
	TasksCompleted int64     `json:"tasks_completed"`
	TasksFailed    int64     `json:"tasks_failed"`
	LastTaskTime   time.Time `json:"last_task_time"`
	StartTime      time.Time `json:"start_time"`
	IdleTime       time.Duration `json:"idle_time"`
	
	// 현재 태스크
	CurrentTask    *TaskWrapper `json:"current_task,omitempty"`
	currentMutex   sync.RWMutex
	
	// 생명주기
	ctx    context.Context
	cancel context.CancelFunc
}

// WorkerState는 워커 상태입니다
type WorkerState int

const (
	WorkerStateIdle WorkerState = iota
	WorkerStateBusy
	WorkerStateStopping
	WorkerStateStopped
)

// GoroutineStats는 고루틴 통계입니다
type GoroutineStats struct {
	Active       int `json:"active"`
	Idle         int `json:"idle"`
	Busy         int `json:"busy"`
	Total        int `json:"total"`
	Completed    int64 `json:"completed"`
	Failed       int64 `json:"failed"`
	
	// 성능 지표
	AvgLifetime     time.Duration `json:"avg_lifetime"`
	AvgTaskDuration time.Duration `json:"avg_task_duration"`
	ThroughputPerSec float64      `json:"throughput_per_sec"`
	
	// 큐 상태
	QueuedTasks      int `json:"queued_tasks"`
	PriorityTasks    int `json:"priority_tasks"`
	
	// 리소스 사용량
	MemoryUsage      int64   `json:"memory_usage"`
	CPUUsage         float64 `json:"cpu_usage"`
	GoroutineCount   int     `json:"goroutine_count"`
	
	// 시간 정보
	LastUpdate       time.Time `json:"last_update"`
	UptimeSeconds    int64     `json:"uptime_seconds"`
}

// WorkerPoolConfig는 워커 풀 설정입니다
type WorkerPoolConfig struct {
	MinWorkers         int           `json:"min_workers"`
	MaxWorkers         int           `json:"max_workers"`
	IdleTimeout        time.Duration `json:"idle_timeout"`
	TaskTimeout        time.Duration `json:"task_timeout"`
	QueueSize          int           `json:"queue_size"`
	PriorityQueueSize  int           `json:"priority_queue_size"`
	
	// 스케일링 설정
	ScaleUpThreshold   float64       `json:"scale_up_threshold"`
	ScaleDownThreshold float64       `json:"scale_down_threshold"`
	ScaleInterval      time.Duration `json:"scale_interval"`
	
	// 모니터링
	StatsInterval      time.Duration `json:"stats_interval"`
	HealthCheckInterval time.Duration `json:"health_check_interval"`
	
	// 성능 최적화
	EnableProfiling    bool `json:"enable_profiling"`
	GCThreshold        int  `json:"gc_threshold"`
}

// WorkerScaler는 워커 자동 스케일링을 담당합니다
type WorkerScaler struct {
	pool       *WorkerPoolManager
	config     ScalerConfig
	
	// 메트릭
	metrics    ScalerMetrics
	metricsMutex sync.RWMutex
	
	// 결정 이력
	decisions  []ScalingDecision
	decisionMutex sync.RWMutex
	maxDecisions int
}

// ScalerConfig는 스케일러 설정입니다
type ScalerConfig struct {
	Enabled            bool          `json:"enabled"`
	MinWorkers         int           `json:"min_workers"`
	MaxWorkers         int           `json:"max_workers"`
	ScaleUpFactor      float64       `json:"scale_up_factor"`
	ScaleDownFactor    float64       `json:"scale_down_factor"`
	CooldownPeriod     time.Duration `json:"cooldown_period"`
	EvaluationWindow   time.Duration `json:"evaluation_window"`
}

// ScalerMetrics는 스케일러 메트릭입니다
type ScalerMetrics struct {
	LastScaleUp        time.Time     `json:"last_scale_up"`
	LastScaleDown      time.Time     `json:"last_scale_down"`
	ScaleUpCount       int64         `json:"scale_up_count"`
	ScaleDownCount     int64         `json:"scale_down_count"`
	CurrentUtilization float64       `json:"current_utilization"`
	TargetUtilization  float64       `json:"target_utilization"`
	EfficiencyScore    float64       `json:"efficiency_score"`
}

// ScalingDecision는 스케일링 결정입니다
type ScalingDecision struct {
	Timestamp   time.Time `json:"timestamp"`
	Action      string    `json:"action"`
	FromWorkers int       `json:"from_workers"`
	ToWorkers   int       `json:"to_workers"`
	Reason      string    `json:"reason"`
	Utilization float64   `json:"utilization"`
}

// GoroutineMonitor는 고루틴 모니터링을 담당합니다
type GoroutineMonitor struct {
	pool         *WorkerPoolManager
	
	// 모니터링 데이터
	snapshots    []MonitoringSnapshot
	snapshotMutex sync.RWMutex
	maxSnapshots int
	
	// 알림
	alertManager AlertManager
	thresholds   MonitoringThresholds
}

// MonitoringSnapshot은 모니터링 스냅샷입니다
type MonitoringSnapshot struct {
	Timestamp      time.Time `json:"timestamp"`
	WorkerCount    int       `json:"worker_count"`
	ActiveWorkers  int       `json:"active_workers"`
	QueueLength    int       `json:"queue_length"`
	MemoryUsage    int64     `json:"memory_usage"`
	GoroutineCount int       `json:"goroutine_count"`
	CPUUsage       float64   `json:"cpu_usage"`
	ThroughputRate float64   `json:"throughput_rate"`
}

// MonitoringThresholds는 모니터링 임계값입니다
type MonitoringThresholds struct {
	MaxMemoryUsage    int64   `json:"max_memory_usage"`
	MaxGoroutineCount int     `json:"max_goroutine_count"`
	MaxCPUUsage       float64 `json:"max_cpu_usage"`
	MinThroughput     float64 `json:"min_throughput"`
}

// DefaultWorkerPoolConfig는 기본 워커 풀 설정을 반환합니다
func DefaultWorkerPoolConfig() WorkerPoolConfig {
	return WorkerPoolConfig{
		MinWorkers:          2,
		MaxWorkers:          runtime.NumCPU() * 2,
		IdleTimeout:         5 * time.Minute,
		TaskTimeout:         30 * time.Second,
		QueueSize:           1000,
		PriorityQueueSize:   100,
		ScaleUpThreshold:    0.8,
		ScaleDownThreshold:  0.3,
		ScaleInterval:       30 * time.Second,
		StatsInterval:       10 * time.Second,
		HealthCheckInterval: 30 * time.Second,
		EnableProfiling:     false,
		GCThreshold:         1000,
	}
}

// DefaultGoroutineLifecycleConfig 기본 고루틴 생명주기 설정
func DefaultGoroutineLifecycleConfig() *GoroutineLifecycleConfig {
	return &GoroutineLifecycleConfig{
		MaxGoroutines:           1000,
		DefaultTimeout:          30 * time.Second,
		ShutdownTimeout:         60 * time.Second,
		EnableTracking:          true,
		TrackingInterval:        10 * time.Second,
		StackTraceDepth:         10,
		CleanupInterval:         30 * time.Second,
		IdleThreshold:           5 * time.Minute,
		ForceCleanupAfter:       10 * time.Minute,
		EnableMetrics:           true,
		MetricsInterval:         15 * time.Second,
		EnableLeakDetection:     true,
		LeakDetectionThreshold:  15 * time.Minute,
		EnableBatching:          true,
		BatchSize:              50,
		PreallocateWorkers:      runtime.NumCPU(),
	}
}

// NewGoroutineLifecycleManager 새로운 고루틴 생명주기 관리자 생성
func NewGoroutineLifecycleManager(config *GoroutineLifecycleConfig) *GoroutineLifecycleManager {
	if config == nil {
		config = DefaultGoroutineLifecycleConfig()
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	manager := &GoroutineLifecycleManager{
		config:          config,
		nextGoroutineID: 0,
		metrics: &GoroutineMetrics{
			CollectionStart: time.Now(),
			LastUpdate:      time.Now(),
		},
		shutdownTimeout: config.ShutdownTimeout,
		ctx:             ctx,
		cancel:          cancel,
	}
	
	// 컴포넌트 초기화
	if config.EnableMetrics {
		manager.metricsCollector = NewMetricsCollector(manager)
	}
	
	if config.EnableTracking {
		manager.tracker = NewGoroutineTracker(manager)
	}
	
	if config.EnableLeakDetection {
		manager.leakDetector = NewLeakDetector(manager)
	}
	
	return manager
}

// SpawnGoroutine 새로운 고루틴 생성
func (glm *GoroutineLifecycleManager) SpawnGoroutine(task Task) (*GoroutineInfo, error) {
	return glm.SpawnGoroutineWithTimeout(task, glm.config.DefaultTimeout)
}

// SpawnGoroutineWithTimeout 타임아웃과 함께 고루틴 생성
func (glm *GoroutineLifecycleManager) SpawnGoroutineWithTimeout(task Task, timeout time.Duration) (*GoroutineInfo, error) {
	if !glm.running.Load() {
		return nil, fmt.Errorf("goroutine lifecycle manager is not running")
	}
	
	// 최대 고루틴 수 확인
	currentCount := glm.getCurrentGoroutineCount()
	if currentCount >= glm.config.MaxGoroutines {
		return nil, fmt.Errorf("maximum goroutines limit reached: %d", glm.config.MaxGoroutines)
	}
	
	// 고루틴 정보 생성
	goroutineID := atomic.AddInt64(&glm.nextGoroutineID, 1)
	ctx, cancel := context.WithTimeout(glm.ctx, timeout)
	
	info := &GoroutineInfo{
		ID:         goroutineID,
		StartTime:  time.Now(),
		LastActive: time.Now(),
		State:      GoroutineStateCreated,
		Task:       task,
		Context:    ctx,
		Cancel:     cancel,
		Metadata:   make(map[string]interface{}),
	}
	
	// 스택 트레이스 수집 (활성화된 경우)
	if glm.config.EnableTracking {
		info.StackTrace = glm.captureStackTrace()
	}
	
	// 고루틴 등록
	glm.activeGoroutines.Store(goroutineID, info)
	
	// 생명주기 훅 실행
	for _, hook := range glm.lifecycleHooks {
		if err := hook.OnGoroutineStart(info); err != nil {
			glm.activeGoroutines.Delete(goroutineID)
			cancel()
			return nil, fmt.Errorf("lifecycle hook failed: %w", err)
		}
	}
	
	// 메트릭 업데이트
	atomic.AddInt64(&glm.metrics.TotalCreated, 1)
	atomic.AddInt64(&glm.metrics.CurrentActive, 1)
	glm.updatePeakActive()
	
	// 고루틴 시작
	go glm.executeGoroutine(info)
	
	return info, nil
}

// executeGoroutine 고루틴 실행
func (glm *GoroutineLifecycleManager) executeGoroutine(info *GoroutineInfo) {
	defer func() {
		if r := recover(); r != nil {
			// 패닉 처리
			atomic.AddInt64(&glm.metrics.PanicCount, 1)
			info.State = GoroutineStateFailed
			
			// 생명주기 훅 실행
			for _, hook := range glm.lifecycleHooks {
				hook.OnGoroutinePanic(info, r)
			}
		}
		
		// 정리 작업
		glm.finalizeGoroutine(info)
	}()
	
	// 상태 업데이트
	info.State = GoroutineStateRunning
	info.LastActive = time.Now()
	
	// 태스크 실행
	err := info.Task.Execute(info.Context)
	
	// 결과에 따른 상태 설정
	if err != nil {
		info.State = GoroutineStateFailed
		atomic.AddInt64(&glm.metrics.TotalFailed, 1)
	} else {
		info.State = GoroutineStateCompleted
		atomic.AddInt64(&glm.metrics.TotalCompleted, 1)
	}
}

// finalizeGoroutine 고루틴 종료 처리
func (glm *GoroutineLifecycleManager) finalizeGoroutine(info *GoroutineInfo) {
	// 컨텍스트 정리
	info.Cancel()
	
	// 생존 시간 계산
	lifetime := time.Since(info.StartTime)
	
	// 생명주기 훅 실행
	for _, hook := range glm.lifecycleHooks {
		hook.OnGoroutineEnd(info)
	}
	
	// 메트릭 업데이트
	atomic.AddInt64(&glm.metrics.CurrentActive, -1)
	glm.updateAverageLifetime(lifetime)
	
	// 고루틴 제거
	glm.activeGoroutines.Delete(info.ID)
}

// GetGoroutinePoolStatus 풀 상태 조회
func (glm *GoroutineLifecycleManager) GetGoroutinePoolStatus() PoolStatus {
	active := int(atomic.LoadInt64(&glm.metrics.CurrentActive))
	total := glm.getCurrentGoroutineCount()
	idle := total - active
	utilization := float64(active) / float64(glm.config.MaxGoroutines)
	
	// 건강 상태 판단
	var health PoolHealth
	switch {
	case utilization > 0.9:
		health = PoolHealthCritical
	case utilization > 0.7:
		health = PoolHealthWarning
	default:
		health = PoolHealthy
	}
	
	return PoolStatus{
		Active:       active,
		Idle:         idle,
		Total:        total,
		MaxCapacity:  glm.config.MaxGoroutines,
		Utilization:  utilization,
		Health:       health,
		LastActivity: time.Now(),
	}
}

// EnableGoroutineTracking 고루틴 추적 활성화/비활성화
func (glm *GoroutineLifecycleManager) EnableGoroutineTracking(enabled bool) {
	glm.mu.Lock()
	defer glm.mu.Unlock()
	
	glm.config.EnableTracking = enabled
	
	if enabled && glm.tracker == nil {
		glm.tracker = NewGoroutineTracker(glm)
		if glm.running.Load() {
			glm.tracker.Start()
		}
	} else if !enabled && glm.tracker != nil {
		glm.tracker.Stop()
		glm.tracker = nil
	}
}

// GetGoroutineLeaks 고루틴 누수 조회
func (glm *GoroutineLifecycleManager) GetGoroutineLeaks() []GoroutineLeak {
	if glm.leakDetector == nil {
		return []GoroutineLeak{}
	}
	
	return glm.leakDetector.DetectLeaks()
}

// ForceGarbageCollection 강제 가비지 컬렉션
func (glm *GoroutineLifecycleManager) ForceGarbageCollection() error {
	runtime.GC()
	runtime.GC() // 두 번 실행하여 확실히 정리
	
	// 통계 업데이트
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	glm.mu.Lock()
	glm.metrics.TotalMemory = int64(m.Alloc)
	glm.metrics.LastUpdate = time.Now()
	glm.mu.Unlock()
	
	return nil
}

// Start 생명주기 관리자 시작
func (glm *GoroutineLifecycleManager) Start() error {
	if !glm.running.CompareAndSwap(false, true) {
		return fmt.Errorf("goroutine lifecycle manager already running")
	}
	
	// 메트릭 수집기 시작
	if glm.metricsCollector != nil {
		glm.metricsCollector.Start()
	}
	
	// 추적기 시작
	if glm.tracker != nil {
		glm.tracker.Start()
	}
	
	// 누수 감지기 시작
	if glm.leakDetector != nil {
		glm.leakDetector.Start()
	}
	
	// 정리 루틴 시작
	go glm.cleanupRoutine()
	
	return nil
}

// Stop 생명주기 관리자 중지
func (glm *GoroutineLifecycleManager) Stop() error {
	if !glm.running.CompareAndSwap(true, false) {
		return nil
	}
	
	// 컨텍스트 취소
	glm.cancel()
	
	// 모든 활성 고루틴 종료 대기
	timeout := time.After(glm.shutdownTimeout)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-timeout:
			// 타임아웃 - 강제 종료
			glm.forceTerminateAll()
			return fmt.Errorf("shutdown timeout, some goroutines may have been forcibly terminated")
		case <-ticker.C:
			if glm.getCurrentGoroutineCount() == 0 {
				// 모든 고루틴이 정상적으로 종료됨
				glm.stopComponents()
				return nil
			}
		}
	}
}

// 내부 유틸리티 메서드들

func (glm *GoroutineLifecycleManager) getCurrentGoroutineCount() int {
	count := 0
	glm.activeGoroutines.Range(func(_, _ interface{}) bool {
		count++
		return true
	})
	return count
}

func (glm *GoroutineLifecycleManager) updatePeakActive() {
	current := atomic.LoadInt64(&glm.metrics.CurrentActive)
	for {
		peak := atomic.LoadInt64(&glm.metrics.PeakActive)
		if current <= peak {
			break
		}
		if atomic.CompareAndSwapInt64(&glm.metrics.PeakActive, peak, current) {
			break
		}
	}
}

func (glm *GoroutineLifecycleManager) updateAverageLifetime(lifetime time.Duration) {
	// 이동 평균 계산 (간단한 구현)
	glm.mu.Lock()
	defer glm.mu.Unlock()
	
	if glm.metrics.AvgLifetime == 0 {
		glm.metrics.AvgLifetime = lifetime
	} else {
		glm.metrics.AvgLifetime = (glm.metrics.AvgLifetime + lifetime) / 2
	}
}

func (glm *GoroutineLifecycleManager) captureStackTrace() []string {
	// 스택 트레이스 캡처 구현
	// runtime.Stack 사용하여 현재 스택 정보 수집
	buf := make([]byte, 1024*glm.config.StackTraceDepth)
	n := runtime.Stack(buf, false)
	return []string{string(buf[:n])}
}

func (glm *GoroutineLifecycleManager) cleanupRoutine() {
	ticker := time.NewTicker(glm.config.CleanupInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-glm.ctx.Done():
			return
		case <-ticker.C:
			glm.performCleanup()
		}
	}
}

func (glm *GoroutineLifecycleManager) performCleanup() {
	now := time.Now()
	var toTerminate []int64
	
	glm.activeGoroutines.Range(func(key, value interface{}) bool {
		info := value.(*GoroutineInfo)
		
		// 유휴 시간 확인
		if info.State == GoroutineStateWaiting {
			idleTime := now.Sub(info.LastActive)
			if idleTime > glm.config.IdleThreshold {
				toTerminate = append(toTerminate, info.ID)
			}
		}
		
		// 강제 정리 시간 확인
		if now.Sub(info.StartTime) > glm.config.ForceCleanupAfter {
			toTerminate = append(toTerminate, info.ID)
		}
		
		return true
	})
	
	// 정리 대상 고루틴들 종료
	for _, id := range toTerminate {
		if value, ok := glm.activeGoroutines.Load(id); ok {
			info := value.(*GoroutineInfo)
			info.Cancel()
			info.State = GoroutineStateTerminated
			atomic.AddInt64(&glm.metrics.TotalTerminated, 1)
		}
	}
}

func (glm *GoroutineLifecycleManager) forceTerminateAll() {
	glm.activeGoroutines.Range(func(key, value interface{}) bool {
		info := value.(*GoroutineInfo)
		info.Cancel()
		info.State = GoroutineStateTerminated
		atomic.AddInt64(&glm.metrics.TotalTerminated, 1)
		return true
	})
}

func (glm *GoroutineLifecycleManager) stopComponents() {
	if glm.metricsCollector != nil {
		glm.metricsCollector.Stop()
	}
	
	if glm.tracker != nil {
		glm.tracker.Stop()
	}
	
	if glm.leakDetector != nil {
		glm.leakDetector.Stop()
	}
}

// NewWorkerPoolManager는 새로운 워커 풀 관리자를 생성합니다
func NewWorkerPoolManager(config WorkerPoolConfig) *WorkerPoolManager {
	ctx, cancel := context.WithCancel(context.Background())
	
	manager := &WorkerPoolManager{
		workers:       make(map[int]*Worker),
		taskQueue:     make(chan TaskWrapper, config.QueueSize),
		priorityQueue: make(chan TaskWrapper, config.PriorityQueueSize),
		config:        config,
		ctx:           ctx,
		cancel:        cancel,
		stats: GoroutineStats{
			LastUpdate: time.Now(),
		},
	}
	
	// 스케일러 초기화
	scalerConfig := ScalerConfig{
		Enabled:          true,
		MinWorkers:       config.MinWorkers,
		MaxWorkers:       config.MaxWorkers,
		ScaleUpFactor:    1.5,
		ScaleDownFactor:  0.7,
		CooldownPeriod:   2 * time.Minute,
		EvaluationWindow: 5 * time.Minute,
	}
	manager.scaler = NewWorkerScaler(manager, scalerConfig)
	
	// 모니터 초기화
	manager.monitor = NewGoroutineMonitor(manager)
	
	return manager
}

// Start는 워커 풀을 시작합니다
func (wpm *WorkerPoolManager) Start() error {
	if !wpm.running.CompareAndSwap(false, true) {
		return fmt.Errorf("worker pool is already running")
	}
	
	// 초기 워커들 생성
	for i := 0; i < wpm.config.MinWorkers; i++ {
		if err := wpm.createWorker(); err != nil {
			return fmt.Errorf("failed to create initial worker: %w", err)
		}
	}
	
	// 백그라운드 작업들 시작
	wpm.wg.Add(3)
	go wpm.taskDispatcher()
	go wpm.statisticsCollector()
	go wpm.healthChecker()
	
	// 스케일러 시작
	if wpm.scaler != nil {
		wpm.scaler.Start()
	}
	
	// 모니터 시작
	if wpm.monitor != nil {
		wpm.monitor.Start()
	}
	
	return nil
}

// Stop은 워커 풀을 중지합니다
func (wpm *WorkerPoolManager) Stop() error {
	if !wpm.running.CompareAndSwap(true, false) {
		return nil // 이미 중지됨
	}
	
	// 스케일러 중지
	if wpm.scaler != nil {
		wpm.scaler.Stop()
	}
	
	// 모니터 중지
	if wpm.monitor != nil {
		wpm.monitor.Stop()
	}
	
	// 모든 워커들 중지
	wpm.workersMutex.Lock()
	for _, worker := range wpm.workers {
		worker.Stop()
	}
	wpm.workersMutex.Unlock()
	
	// 컨텍스트 취소
	wpm.cancel()
	
	// 백그라운드 작업들 완료 대기
	wpm.wg.Wait()
	
	// 큐 정리
	close(wpm.taskQueue)
	close(wpm.priorityQueue)
	
	return nil
}

// SpawnWorker는 새로운 작업을 생성합니다
func (wpm *WorkerPoolManager) SpawnWorker(task Task) error {
	return wpm.SpawnBoundedWorker(task, wpm.config.TaskTimeout)
}

// SpawnBoundedWorker는 타임아웃이 있는 작업을 생성합니다
func (wpm *WorkerPoolManager) SpawnBoundedWorker(task Task, timeout time.Duration) error {
	if !wpm.running.Load() {
		return fmt.Errorf("worker pool is not running")
	}
	
	ctx, cancel := context.WithTimeout(wpm.ctx, timeout)
	
	wrapper := TaskWrapper{
		Task:      task,
		StartTime: time.Now(),
		Timeout:   timeout,
		ResultCh:  make(chan TaskResult, 1),
		Ctx:       ctx,
		Cancel:    cancel,
	}
	
	// 우선순위에 따라 큐 선택
	if task.GetPriority() >= TaskPriorityHigh {
		select {
		case wpm.priorityQueue <- wrapper:
			return nil
		default:
			cancel()
			return fmt.Errorf("priority queue is full")
		}
	} else {
		select {
		case wpm.taskQueue <- wrapper:
			return nil
		default:
			cancel()
			return fmt.Errorf("task queue is full")
		}
	}
}

// GetActiveGoroutines는 활성 고루틴 수를 반환합니다
func (wpm *WorkerPoolManager) GetActiveGoroutines() int {
	wpm.workersMutex.RLock()
	defer wpm.workersMutex.RUnlock()
	
	active := 0
	for _, worker := range wpm.workers {
		if worker.State == WorkerStateBusy {
			active++
		}
	}
	
	return active
}

// GetGoroutineStats는 고루틴 통계를 반환합니다
func (wpm *WorkerPoolManager) GetGoroutineStats() GoroutineStats {
	wpm.statsMutex.RLock()
	defer wpm.statsMutex.RUnlock()
	
	stats := wpm.stats
	stats.LastUpdate = time.Now()
	
	return stats
}

// CleanupIdleWorkers는 유휴 워커들을 정리합니다
func (wpm *WorkerPoolManager) CleanupIdleWorkers() error {
	wpm.workersMutex.Lock()
	defer wpm.workersMutex.Unlock()
	
	now := time.Now()
	var toRemove []int
	
	for id, worker := range wpm.workers {
		if worker.State == WorkerStateIdle {
			idleTime := now.Sub(worker.LastTaskTime)
			if idleTime > wpm.config.IdleTimeout {
				toRemove = append(toRemove, id)
			}
		}
	}
	
	// 최소 워커 수 유지
	currentWorkers := len(wpm.workers)
	maxRemovable := currentWorkers - wpm.config.MinWorkers
	if maxRemovable < 0 {
		maxRemovable = 0
	}
	
	if len(toRemove) > maxRemovable {
		toRemove = toRemove[:maxRemovable]
	}
	
	// 워커들 제거
	for _, id := range toRemove {
		if worker, exists := wpm.workers[id]; exists {
			worker.Stop()
			delete(wpm.workers, id)
		}
	}
	
	return nil
}

// SetMaxGoroutines는 최대 고루틴 수를 설정합니다
func (wpm *WorkerPoolManager) SetMaxGoroutines(max int) error {
	if max < wpm.config.MinWorkers {
		return fmt.Errorf("max goroutines cannot be less than min workers")
	}
	
	wpm.config.MaxWorkers = max
	
	// 현재 워커 수가 새로운 최대값보다 많으면 조정
	wpm.workersMutex.Lock()
	currentWorkers := len(wpm.workers)
	wpm.workersMutex.Unlock()
	
	if currentWorkers > max {
		excess := currentWorkers - max
		for i := 0; i < excess; i++ {
			wpm.removeWorker()
		}
	}
	
	return nil
}

// 내부 메서드들

func (wpm *WorkerPoolManager) createWorker() error {
	wpm.workersMutex.Lock()
	defer wpm.workersMutex.Unlock()
	
	if len(wpm.workers) >= wpm.config.MaxWorkers {
		return fmt.Errorf("maximum worker limit reached")
	}
	
	workerID := int(atomic.AddInt32(&wpm.nextWorkerID, 1))
	ctx, cancel := context.WithCancel(wpm.ctx)
	
	worker := &Worker{
		ID:        workerID,
		State:     WorkerStateIdle,
		TaskQueue: make(chan TaskWrapper, 1),
		QuitChan:  make(chan bool, 1),
		Pool:      wpm,
		StartTime: time.Now(),
		ctx:       ctx,
		cancel:    cancel,
	}
	
	wpm.workers[workerID] = worker
	
	// 워커 고루틴 시작
	go worker.Run()
	
	return nil
}

func (wpm *WorkerPoolManager) removeWorker() error {
	wpm.workersMutex.Lock()
	defer wpm.workersMutex.Unlock()
	
	// 유휴 상태인 워커 찾기
	for id, worker := range wpm.workers {
		if worker.State == WorkerStateIdle {
			worker.Stop()
			delete(wpm.workers, id)
			return nil
		}
	}
	
	return fmt.Errorf("no idle worker found to remove")
}

func (wpm *WorkerPoolManager) taskDispatcher() {
	defer wpm.wg.Done()
	
	for {
		select {
		case <-wpm.ctx.Done():
			return
		case task := <-wpm.priorityQueue:
			wpm.assignTask(task)
		case task := <-wpm.taskQueue:
			wpm.assignTask(task)
		}
	}
}

func (wpm *WorkerPoolManager) assignTask(task TaskWrapper) {
	wpm.workersMutex.RLock()
	
	// 유휴 워커 찾기
	var idleWorker *Worker
	for _, worker := range wpm.workers {
		if worker.State == WorkerStateIdle {
			idleWorker = worker
			break
		}
	}
	wpm.workersMutex.RUnlock()
	
	if idleWorker == nil {
		// 유휴 워커가 없으면 새 워커 생성 시도
		if err := wpm.createWorker(); err == nil {
			wpm.workersMutex.RLock()
			for _, worker := range wpm.workers {
				if worker.State == WorkerStateIdle {
					idleWorker = worker
					break
				}
			}
			wpm.workersMutex.RUnlock()
		}
	}
	
	if idleWorker != nil {
		select {
		case idleWorker.TaskQueue <- task:
			// 태스크 할당 성공
		default:
			// 워커가 바쁘면 다시 큐에 추가
			go func() {
				if task.Task.GetPriority() >= TaskPriorityHigh {
					wpm.priorityQueue <- task
				} else {
					wpm.taskQueue <- task
				}
			}()
		}
	} else {
		// 모든 워커가 바쁘면 다시 큐에 추가
		go func() {
			if task.Task.GetPriority() >= TaskPriorityHigh {
				wpm.priorityQueue <- task
			} else {
				wpm.taskQueue <- task
			}
		}()
	}
}

func (wpm *WorkerPoolManager) statisticsCollector() {
	defer wpm.wg.Done()
	
	ticker := time.NewTicker(wpm.config.StatsInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-wpm.ctx.Done():
			return
		case <-ticker.C:
			wpm.updateStatistics()
		}
	}
}

func (wpm *WorkerPoolManager) updateStatistics() {
	wpm.workersMutex.RLock()
	workerCount := len(wpm.workers)
	activeWorkers := 0
	idleWorkers := 0
	var totalCompleted, totalFailed int64
	var totalLifetime time.Duration
	
	for _, worker := range wpm.workers {
		switch worker.State {
		case WorkerStateBusy:
			activeWorkers++
		case WorkerStateIdle:
			idleWorkers++
		}
		
		totalCompleted += worker.TasksCompleted
		totalFailed += worker.TasksFailed
		totalLifetime += time.Since(worker.StartTime)
	}
	wpm.workersMutex.RUnlock()
	
	wpm.statsMutex.Lock()
	wpm.stats.Active = activeWorkers
	wpm.stats.Idle = idleWorkers
	wpm.stats.Busy = activeWorkers
	wpm.stats.Total = workerCount
	wpm.stats.Completed = totalCompleted
	wpm.stats.Failed = totalFailed
	wpm.stats.QueuedTasks = len(wpm.taskQueue)
	wpm.stats.PriorityTasks = len(wpm.priorityQueue)
	wpm.stats.GoroutineCount = runtime.NumGoroutine()
	wpm.stats.LastUpdate = time.Now()
	
	if workerCount > 0 {
		wpm.stats.AvgLifetime = totalLifetime / time.Duration(workerCount)
	}
	wpm.statsMutex.Unlock()
}

func (wpm *WorkerPoolManager) healthChecker() {
	defer wpm.wg.Done()
	
	ticker := time.NewTicker(wpm.config.HealthCheckInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-wpm.ctx.Done():
			return
		case <-ticker.C:
			wpm.performHealthCheck()
		}
	}
}

func (wpm *WorkerPoolManager) performHealthCheck() {
	// 메모리 사용량 확인
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	wpm.statsMutex.Lock()
	wpm.stats.MemoryUsage = int64(m.Alloc)
	wpm.statsMutex.Unlock()
	
	// GC 임계값 확인
	if wpm.config.EnableProfiling && wpm.stats.Completed%int64(wpm.config.GCThreshold) == 0 {
		runtime.GC()
	}
	
	// 유휴 워커 정리
	wpm.CleanupIdleWorkers()
}

// Worker 메서드들

// Run은 워커를 실행합니다
func (w *Worker) Run() {
	defer func() {
		w.State = WorkerStateStopped
		if r := recover(); r != nil {
			// 패닉 복구
			fmt.Printf("Worker %d panicked: %v\n", w.ID, r)
		}
	}()
	
	for {
		w.State = WorkerStateIdle
		w.LastTaskTime = time.Now()
		
		select {
		case <-w.ctx.Done():
			return
		case <-w.QuitChan:
			return
		case task := <-w.TaskQueue:
			w.executeTask(task)
		}
	}
}

func (w *Worker) executeTask(wrapper TaskWrapper) {
	w.State = WorkerStateBusy
	w.TasksProcessed++
	
	w.currentMutex.Lock()
	w.CurrentTask = &wrapper
	w.currentMutex.Unlock()
	
	startTime := time.Now()
	err := wrapper.Task.Execute(wrapper.Ctx)
	endTime := time.Now()
	duration := endTime.Sub(startTime)
	
	result := TaskResult{
		Success:   err == nil,
		Error:     err,
		Duration:  duration,
		StartTime: startTime,
		EndTime:   endTime,
	}
	
	if err == nil {
		w.TasksCompleted++
	} else {
		w.TasksFailed++
	}
	
	// 결과 전송
	select {
	case wrapper.ResultCh <- result:
	default:
		// 채널이 막혀있으면 무시
	}
	
	// 컨텍스트 정리
	wrapper.Cancel()
	
	w.currentMutex.Lock()
	w.CurrentTask = nil
	w.currentMutex.Unlock()
	
	w.LastTaskTime = time.Now()
}

// Stop은 워커를 중지합니다
func (w *Worker) Stop() {
	if w.State != WorkerStateStopped {
		w.State = WorkerStateStopping
		
		// 현재 실행 중인 태스크 취소
		w.currentMutex.RLock()
		if w.CurrentTask != nil {
			w.CurrentTask.Cancel()
		}
		w.currentMutex.RUnlock()
		
		w.cancel()
		close(w.QuitChan)
	}
}

// NewWorkerScaler는 새로운 워커 스케일러를 생성합니다
func NewWorkerScaler(pool *WorkerPoolManager, config ScalerConfig) *WorkerScaler {
	return &WorkerScaler{
		pool:         pool,
		config:       config,
		decisions:    make([]ScalingDecision, 0),
		maxDecisions: 100,
	}
}

// Start는 스케일러를 시작합니다
func (ws *WorkerScaler) Start() {
	// 스케일러 구현은 간단화
}

// Stop은 스케일러를 중지합니다
func (ws *WorkerScaler) Stop() {
	// 스케일러 중지 로직
}

// NewGoroutineMonitor는 새로운 고루틴 모니터를 생성합니다
func NewGoroutineMonitor(pool *WorkerPoolManager) *GoroutineMonitor {
	return &GoroutineMonitor{
		pool:         pool,
		snapshots:    make([]MonitoringSnapshot, 0),
		maxSnapshots: 1000,
	}
}

// Start는 모니터를 시작합니다
func (gm *GoroutineMonitor) Start() {
	// 모니터 구현은 간단화
}

// Stop은 모니터를 중지합니다
func (gm *GoroutineMonitor) Stop() {
	// 모니터 중지 로직
}

// 고루틴 생명주기 관리 전용 컴포넌트들

// MetricsCollector 메트릭 수집기
type MetricsCollector struct {
	manager   *GoroutineLifecycleManager
	interval  time.Duration
	running   atomic.Bool
	ctx       context.Context
	cancel    context.CancelFunc
	
	// Prometheus 메트릭스
	promGoroutinesActive    prometheus.Gauge
	promGoroutinesTotal     prometheus.Counter
	promGoroutinesCompleted prometheus.Counter
	promGoroutinesFailed    prometheus.Counter
	promGoroutinesLifetime  prometheus.Histogram
	promGoroutinesMemory    prometheus.Gauge
}

// NewMetricsCollector 새로운 메트릭 수집기 생성
func NewMetricsCollector(manager *GoroutineLifecycleManager) *MetricsCollector {
	ctx, cancel := context.WithCancel(context.Background())
	
	collector := &MetricsCollector{
		manager:  manager,
		interval: manager.config.MetricsInterval,
		ctx:      ctx,
		cancel:   cancel,
	}
	
	// Prometheus 메트릭스 초기화
	collector.initPrometheusMetrics()
	
	return collector
}

func (mc *MetricsCollector) initPrometheusMetrics() {
	mc.promGoroutinesActive = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "claude_goroutines_active",
		Help: "현재 활성 고루틴 수",
	})
	
	mc.promGoroutinesTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "claude_goroutines_created_total",
		Help: "생성된 총 고루틴 수",
	})
	
	mc.promGoroutinesCompleted = promauto.NewCounter(prometheus.CounterOpts{
		Name: "claude_goroutines_completed_total",
		Help: "완료된 총 고루틴 수",
	})
	
	mc.promGoroutinesFailed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "claude_goroutines_failed_total",
		Help: "실패한 총 고루틴 수",
	})
	
	mc.promGoroutinesLifetime = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "claude_goroutines_lifetime_seconds",
		Help:    "고루틴 생존 시간",
		Buckets: prometheus.ExponentialBuckets(0.001, 2, 15),
	})
	
	mc.promGoroutinesMemory = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "claude_goroutines_memory_bytes",
		Help: "고루틴 메모리 사용량",
	})
}

func (mc *MetricsCollector) Start() {
	if !mc.running.CompareAndSwap(false, true) {
		return
	}
	
	go mc.collectLoop()
}

func (mc *MetricsCollector) Stop() {
	if !mc.running.CompareAndSwap(true, false) {
		return
	}
	
	mc.cancel()
}

func (mc *MetricsCollector) collectLoop() {
	ticker := time.NewTicker(mc.interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-mc.ctx.Done():
			return
		case <-ticker.C:
			mc.collectMetrics()
		}
	}
}

func (mc *MetricsCollector) collectMetrics() {
	metrics := mc.manager.metrics
	
	// Prometheus 메트릭스 업데이트
	mc.promGoroutinesActive.Set(float64(atomic.LoadInt64(&metrics.CurrentActive)))
	mc.promGoroutinesTotal.Add(float64(atomic.LoadInt64(&metrics.TotalCreated)))
	mc.promGoroutinesCompleted.Add(float64(atomic.LoadInt64(&metrics.TotalCompleted)))
	mc.promGoroutinesFailed.Add(float64(atomic.LoadInt64(&metrics.TotalFailed)))
	mc.promGoroutinesMemory.Set(float64(metrics.TotalMemory))
}

// GoroutineTracker 고루틴 추적기
type GoroutineTracker struct {
	manager   *GoroutineLifecycleManager
	interval  time.Duration
	running   atomic.Bool
	ctx       context.Context
	cancel    context.CancelFunc
	
	// 추적 데이터
	snapshots []GoroutineSnapshot
	mu        sync.RWMutex
}

// GoroutineSnapshot 고루틴 스냅샷
type GoroutineSnapshot struct {
	Timestamp       time.Time                 `json:"timestamp"`
	ActiveCount     int                       `json:"active_count"`
	TotalCount      int                       `json:"total_count"`
	GoroutineStates map[GoroutineState]int    `json:"goroutine_states"`
	MemoryUsage     int64                     `json:"memory_usage"`
	Goroutines      []*GoroutineInfo          `json:"goroutines,omitempty"`
}

// NewGoroutineTracker 새로운 고루틴 추적기 생성
func NewGoroutineTracker(manager *GoroutineLifecycleManager) *GoroutineTracker {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &GoroutineTracker{
		manager:   manager,
		interval:  manager.config.TrackingInterval,
		ctx:       ctx,
		cancel:    cancel,
		snapshots: make([]GoroutineSnapshot, 0),
	}
}

func (gt *GoroutineTracker) Start() {
	if !gt.running.CompareAndSwap(false, true) {
		return
	}
	
	go gt.trackLoop()
}

func (gt *GoroutineTracker) Stop() {
	if !gt.running.CompareAndSwap(true, false) {
		return
	}
	
	gt.cancel()
}

func (gt *GoroutineTracker) trackLoop() {
	ticker := time.NewTicker(gt.interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-gt.ctx.Done():
			return
		case <-ticker.C:
			gt.takeSnapshot()
		}
	}
}

func (gt *GoroutineTracker) takeSnapshot() {
	snapshot := GoroutineSnapshot{
		Timestamp:       time.Now(),
		GoroutineStates: make(map[GoroutineState]int),
	}
	
	var goroutines []*GoroutineInfo
	
	gt.manager.activeGoroutines.Range(func(key, value interface{}) bool {
		info := value.(*GoroutineInfo)
		snapshot.GoroutineStates[info.State]++
		snapshot.TotalCount++
		
		if info.State == GoroutineStateRunning {
			snapshot.ActiveCount++
		}
		
		// 세부 정보 포함 (옵션)
		if gt.manager.config.EnableTracking {
			goroutines = append(goroutines, info)
		}
		
		return true
	})
	
	snapshot.Goroutines = goroutines
	
	// 메모리 사용량 수집
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	snapshot.MemoryUsage = int64(m.Alloc)
	
	// 스냅샷 저장
	gt.mu.Lock()
	gt.snapshots = append(gt.snapshots, snapshot)
	
	// 최대 1000개 스냅샷 유지
	if len(gt.snapshots) > 1000 {
		gt.snapshots = gt.snapshots[1:]
	}
	gt.mu.Unlock()
}

func (gt *GoroutineTracker) GetSnapshots() []GoroutineSnapshot {
	gt.mu.RLock()
	defer gt.mu.RUnlock()
	
	snapshots := make([]GoroutineSnapshot, len(gt.snapshots))
	copy(snapshots, gt.snapshots)
	return snapshots
}

// LeakDetector 고루틴 누수 감지기
type LeakDetector struct {
	manager   *GoroutineLifecycleManager
	threshold time.Duration
	running   atomic.Bool
	ctx       context.Context
	cancel    context.CancelFunc
	
	// 감지된 누수
	leaks     []GoroutineLeak
	leaksMu   sync.RWMutex
}

// NewLeakDetector 새로운 누수 감지기 생성
func NewLeakDetector(manager *GoroutineLifecycleManager) *LeakDetector {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &LeakDetector{
		manager:   manager,
		threshold: manager.config.LeakDetectionThreshold,
		ctx:       ctx,
		cancel:    cancel,
		leaks:     make([]GoroutineLeak, 0),
	}
}

func (ld *LeakDetector) Start() {
	if !ld.running.CompareAndSwap(false, true) {
		return
	}
	
	go ld.detectLoop()
}

func (ld *LeakDetector) Stop() {
	if !ld.running.CompareAndSwap(true, false) {
		return
	}
	
	ld.cancel()
}

func (ld *LeakDetector) detectLoop() {
	ticker := time.NewTicker(ld.threshold / 2) // 임계값의 절반마다 검사
	defer ticker.Stop()
	
	for {
		select {
		case <-ld.ctx.Done():
			return
		case <-ticker.C:
			ld.detectLeaks()
		}
	}
}

func (ld *LeakDetector) detectLeaks() {
	now := time.Now()
	var detectedLeaks []GoroutineLeak
	
	ld.manager.activeGoroutines.Range(func(key, value interface{}) bool {
		info := value.(*GoroutineInfo)
		
		// 오래 실행되고 있는 고루틴 확인
		lifetime := now.Sub(info.StartTime)
		if lifetime > ld.threshold {
			severity := ld.calculateSeverity(lifetime, info)
			
			leak := GoroutineLeak{
				GoroutineID:  info.ID,
				StartTime:    info.StartTime,
				Duration:     lifetime,
				LastActivity: info.LastActive,
				StackTrace:   info.StackTrace,
				TaskInfo:     info.Task,
				MemoryUsage:  info.Memory,
				Severity:     severity,
			}
			
			detectedLeaks = append(detectedLeaks, leak)
		}
		
		return true
	})
	
	// 누수 목록 업데이트
	ld.leaksMu.Lock()
	ld.leaks = detectedLeaks
	ld.leaksMu.Unlock()
	
	// 메트릭 업데이트
	atomic.StoreInt64(&ld.manager.metrics.LeakCount, int64(len(detectedLeaks)))
}

func (ld *LeakDetector) calculateSeverity(lifetime time.Duration, info *GoroutineInfo) LeakSeverity {
	ratio := float64(lifetime) / float64(ld.threshold)
	
	switch {
	case ratio > 5.0:
		return LeakSeverityCritical
	case ratio > 3.0:
		return LeakSeverityHigh
	case ratio > 2.0:
		return LeakSeverityMedium
	default:
		return LeakSeverityLow
	}
}

func (ld *LeakDetector) DetectLeaks() []GoroutineLeak {
	ld.leaksMu.RLock()
	defer ld.leaksMu.RUnlock()
	
	leaks := make([]GoroutineLeak, len(ld.leaks))
	copy(leaks, ld.leaks)
	return leaks
}

// AlertManager 알림 관리자 (인터페이스)
type AlertManager interface {
	SendAlert(alert Alert) error
}

// Alert 알림
type Alert struct {
	Level       AlertLevel    `json:"level"`
	Message     string        `json:"message"`
	Source      string        `json:"source"`
	Timestamp   time.Time     `json:"timestamp"`
	Metadata    interface{}   `json:"metadata,omitempty"`
}

// AlertLevel 알림 레벨
type AlertLevel string

const (
	AlertLevelInfo     AlertLevel = "info"
	AlertLevelWarning  AlertLevel = "warning"
	AlertLevelError    AlertLevel = "error"
	AlertLevelCritical AlertLevel = "critical"
)