package agent

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

// PerformanceMetricsCollector 고급 성능 메트릭 수집기
type PerformanceMetricsCollector struct {
	// 메트릭 저장소
	agentMetrics     sync.Map // string -> *AgentPerformanceMetrics
	systemMetrics    *SystemPerformanceMetrics
	aggregatedStats  *AggregatedMetrics

	// Prometheus 메트릭
	prometheusMetrics *PrometheusMetrics

	// 설정
	config MetricsConfig

	// 수집 제어
	collecting    bool
	collectTicker *time.Ticker
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup

	// 의존성
	pool *AdvancedAgentPool

	// 동시성 제어
	mu sync.RWMutex
}

// MetricsConfig 메트릭 수집 설정
type MetricsConfig struct {
	CollectionInterval time.Duration `json:"collection_interval"` // 수집 간격
	RetentionPeriod    time.Duration `json:"retention_period"`    // 데이터 보관 기간
	MaxMetricsPerAgent int           `json:"max_metrics_per_agent"` // 에이전트당 최대 메트릭 수
	EnableSystemMetrics bool         `json:"enable_system_metrics"` // 시스템 메트릭 활성화
	EnablePrometheus   bool         `json:"enable_prometheus"`     // Prometheus 메트릭 활성화
}

// AgentPerformanceMetrics 에이전트 성능 메트릭
type AgentPerformanceMetrics struct {
	AgentID   string    `json:"agent_id"`
	Timestamp time.Time `json:"timestamp"`

	// CPU 메트릭
	CPUUsagePercent float64 `json:"cpu_usage_percent"`
	CPUCores        int     `json:"cpu_cores"`

	// 메모리 메트릭
	MemoryUsageBytes   uint64  `json:"memory_usage_bytes"`
	MemoryUsagePercent float64 `json:"memory_usage_percent"`
	MemoryLimitBytes   uint64  `json:"memory_limit_bytes"`

	// 컨테이너 메트릭
	ContainerID       string        `json:"container_id"`
	ContainerStatus   string        `json:"container_status"`
	ContainerUptime   time.Duration `json:"container_uptime"`

	// 작업 메트릭
	TasksCompleted    int64         `json:"tasks_completed"`
	AverageTaskTime   time.Duration `json:"average_task_time"`
	ErrorCount        int64         `json:"error_count"`
	SuccessRate       float64       `json:"success_rate"`

	// 네트워크 메트릭
	NetworkBytesIn  uint64 `json:"network_bytes_in"`
	NetworkBytesOut uint64 `json:"network_bytes_out"`
	NetworkPacketsIn uint64 `json:"network_packets_in"`
	NetworkPacketsOut uint64 `json:"network_packets_out"`

	// 디스크 메트릭
	DiskReadBytes    uint64 `json:"disk_read_bytes"`
	DiskWriteBytes   uint64 `json:"disk_write_bytes"`
	DiskUsageBytes   uint64 `json:"disk_usage_bytes"`
	DiskUsagePercent float64 `json:"disk_usage_percent"`
}

// SystemPerformanceMetrics 시스템 성능 메트릭
type SystemPerformanceMetrics struct {
	Timestamp time.Time `json:"timestamp"`

	// 전체 시스템 CPU
	SystemCPUUsagePercent float64 `json:"system_cpu_usage_percent"`
	CPUCoreCount          int     `json:"cpu_core_count"`
	LoadAverage1Min       float64 `json:"load_average_1min"`
	LoadAverage5Min       float64 `json:"load_average_5min"`
	LoadAverage15Min      float64 `json:"load_average_15min"`

	// 전체 시스템 메모리
	TotalMemoryBytes     uint64  `json:"total_memory_bytes"`
	UsedMemoryBytes      uint64  `json:"used_memory_bytes"`
	AvailableMemoryBytes uint64  `json:"available_memory_bytes"`
	MemoryUsagePercent   float64 `json:"memory_usage_percent"`

	// 디스크 메트릭
	TotalDiskBytes     uint64  `json:"total_disk_bytes"`
	UsedDiskBytes      uint64  `json:"used_disk_bytes"`
	AvailableDiskBytes uint64  `json:"available_disk_bytes"`
	DiskUsagePercent   float64 `json:"disk_usage_percent"`

	// 네트워크 메트릭
	NetworkConnectionsTotal uint64 `json:"network_connections_total"`
	NetworkBytesReceivedTotal uint64 `json:"network_bytes_received_total"`
	NetworkBytesSentTotal uint64 `json:"network_bytes_sent_total"`

	// 프로세스 메트릭
	TotalProcesses    int `json:"total_processes"`
	RunningProcesses  int `json:"running_processes"`
	SleepingProcesses int `json:"sleeping_processes"`

	// Go 런타임 메트릭
	GoRoutines       int     `json:"go_routines"`
	GoMemoryHeapUsed uint64  `json:"go_memory_heap_used"`
	GoMemoryHeapSys  uint64  `json:"go_memory_heap_sys"`
	GoGCPauseTotal   float64 `json:"go_gc_pause_total"`
}

// AggregatedMetrics 집계된 메트릭
type AggregatedMetrics struct {
	Timestamp time.Time `json:"timestamp"`

	// 에이전트 통계
	TotalActiveAgents   int     `json:"total_active_agents"`
	AverageCPUUsage     float64 `json:"average_cpu_usage"`
	AverageMemoryUsage  float64 `json:"average_memory_usage"`
	TotalMemoryUsage    uint64  `json:"total_memory_usage"`
	
	// 성능 통계
	AverageCreationTime time.Duration `json:"average_creation_time"`
	P95CreationTime     time.Duration `json:"p95_creation_time"`
	P99CreationTime     time.Duration `json:"p99_creation_time"`
	TotalTasksCompleted int64         `json:"total_tasks_completed"`
	OverallSuccessRate  float64       `json:"overall_success_rate"`
	
	// 풀 통계
	PoolHitRate         float64 `json:"pool_hit_rate"`
	PoolUtilization     float64 `json:"pool_utilization"`
	RecycleRate         float64 `json:"recycle_rate"`
}

// PrometheusMetrics Prometheus 메트릭 정의
type PrometheusMetrics struct {
	// 에이전트 메트릭
	AgentCPUUsage      prometheus.GaugeVec
	AgentMemoryUsage   prometheus.GaugeVec
	AgentTasksTotal    prometheus.CounterVec
	AgentErrorsTotal   prometheus.CounterVec
	AgentCreationTime  prometheus.HistogramVec

	// 시스템 메트릭
	SystemCPUUsage     prometheus.Gauge
	SystemMemoryUsage  prometheus.Gauge
	SystemDiskUsage    prometheus.Gauge
	SystemLoadAverage  prometheus.GaugeVec

	// 풀 메트릭
	PoolSize           prometheus.Gauge
	PoolUtilization    prometheus.Gauge
	PoolHitRate        prometheus.Gauge
	PoolRecycleRate    prometheus.Counter

	// Go 런타임 메트릭
	GoRoutines         prometheus.Gauge
	GoMemoryHeap       prometheus.Gauge
	GoGCDuration       prometheus.Histogram
}

// NewPerformanceMetricsCollector 새 성능 메트릭 수집기 생성
func NewPerformanceMetricsCollector(config MetricsConfig, pool *AdvancedAgentPool) *PerformanceMetricsCollector {
	ctx, cancel := context.WithCancel(context.Background())

	collector := &PerformanceMetricsCollector{
		systemMetrics:   &SystemPerformanceMetrics{},
		aggregatedStats: &AggregatedMetrics{},
		config:         config,
		ctx:            ctx,
		cancel:         cancel,
		pool:           pool,
	}

	if config.EnablePrometheus {
		collector.prometheusMetrics = NewPrometheusMetrics()
	}

	return collector
}

// Start 메트릭 수집 시작
func (c *PerformanceMetricsCollector) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.collecting {
		return fmt.Errorf("metrics collector already running")
	}

	c.collecting = true
	c.collectTicker = time.NewTicker(c.config.CollectionInterval)

	// 메트릭 수집 고루틴 시작
	c.wg.Add(1)
	go c.collectLoop()

	// 정리 작업 고루틴 시작
	c.wg.Add(1)
	go c.cleanupLoop()

	return nil
}

// Stop 메트릭 수집 중지
func (c *PerformanceMetricsCollector) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.collecting {
		return nil
	}

	c.collecting = false
	c.cancel()

	if c.collectTicker != nil {
		c.collectTicker.Stop()
	}

	c.wg.Wait()
	return nil
}

// collectLoop 메트릭 수집 루프
func (c *PerformanceMetricsCollector) collectLoop() {
	defer c.wg.Done()

	for {
		select {
		case <-c.ctx.Done():
			return

		case <-c.collectTicker.C:
			c.collectAllMetrics()
		}
	}
}

// collectAllMetrics 모든 메트릭 수집
func (c *PerformanceMetricsCollector) collectAllMetrics() {
	// 시스템 메트릭 수집
	if c.config.EnableSystemMetrics {
		c.collectSystemMetrics()
	}

	// 에이전트 메트릭 수집
	c.collectAgentMetrics()

	// 집계 메트릭 계산
	c.calculateAggregatedMetrics()

	// Prometheus 메트릭 업데이트
	if c.config.EnablePrometheus {
		c.updatePrometheusMetrics()
	}
}

// collectSystemMetrics 시스템 메트릭 수집
func (c *PerformanceMetricsCollector) collectSystemMetrics() {
	now := time.Now()
	c.systemMetrics.Timestamp = now

	// CPU 메트릭
	if cpuPercent, err := cpu.Percent(0, false); err == nil && len(cpuPercent) > 0 {
		c.systemMetrics.SystemCPUUsagePercent = cpuPercent[0]
	}

	if cpuCounts, err := cpu.Counts(true); err == nil {
		c.systemMetrics.CPUCoreCount = cpuCounts
	}

	// 메모리 메트릭
	if memInfo, err := mem.VirtualMemory(); err == nil {
		c.systemMetrics.TotalMemoryBytes = memInfo.Total
		c.systemMetrics.UsedMemoryBytes = memInfo.Used
		c.systemMetrics.AvailableMemoryBytes = memInfo.Available
		c.systemMetrics.MemoryUsagePercent = memInfo.UsedPercent
	}

	// Go 런타임 메트릭
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	c.systemMetrics.GoRoutines = runtime.NumGoroutine()
	c.systemMetrics.GoMemoryHeapUsed = memStats.HeapInuse
	c.systemMetrics.GoMemoryHeapSys = memStats.HeapSys
	c.systemMetrics.GoGCPauseTotal = float64(memStats.PauseTotalNs) / 1e9
}

// collectAgentMetrics 에이전트 메트릭 수집
func (c *PerformanceMetricsCollector) collectAgentMetrics() {
	// 풀에서 실행 중인 에이전트들 메트릭 수집
	c.pool.runningAgents.Range(func(key, value interface{}) bool {
		agentID := key.(string)
		pooledAgent := value.(*PooledAgent)

		metrics := c.collectAgentMetric(pooledAgent)
		if metrics != nil {
			c.storeAgentMetrics(agentID, metrics)
		}

		return true
	})
}

// collectAgentMetric 개별 에이전트 메트릭 수집
func (c *PerformanceMetricsCollector) collectAgentMetric(pooledAgent *PooledAgent) *AgentPerformanceMetrics {
	if pooledAgent == nil || pooledAgent.Agent == nil {
		return nil
	}

	metrics := &AgentPerformanceMetrics{
		AgentID:   pooledAgent.Agent.ID,
		Timestamp: time.Now(),
	}

	// 컨테이너 정보
	if pooledAgent.Container != nil {
		metrics.ContainerID = pooledAgent.Container.ID
		metrics.ContainerUptime = time.Since(pooledAgent.CreatedAt)
	}

	// 기본 성능 정보
	metrics.CPUCores = runtime.NumCPU()

	// 에이전트별 사용 통계
	pooledAgent.mu.RLock()
	metrics.TasksCompleted = pooledAgent.UsageCount
	metrics.AverageTaskTime = pooledAgent.TotalRunTime / time.Duration(max(1, int(pooledAgent.UsageCount)))
	pooledAgent.mu.RUnlock()

	// 성공률 계산 (임시로 90% 가정)
	metrics.SuccessRate = 0.9

	return metrics
}

// storeAgentMetrics 에이전트 메트릭 저장
func (c *PerformanceMetricsCollector) storeAgentMetrics(agentID string, metrics *AgentPerformanceMetrics) {
	var agentMetricsList []*AgentPerformanceMetrics

	if existing, ok := c.agentMetrics.Load(agentID); ok {
		agentMetricsList = existing.([]*AgentPerformanceMetrics)
	}

	// 새 메트릭 추가
	agentMetricsList = append(agentMetricsList, metrics)

	// 최대 개수 제한
	if len(agentMetricsList) > c.config.MaxMetricsPerAgent {
		agentMetricsList = agentMetricsList[len(agentMetricsList)-c.config.MaxMetricsPerAgent:]
	}

	c.agentMetrics.Store(agentID, agentMetricsList)
}

// calculateAggregatedMetrics 집계 메트릭 계산
func (c *PerformanceMetricsCollector) calculateAggregatedMetrics() {
	c.aggregatedStats.Timestamp = time.Now()

	var totalCPU, totalMemory float64
	var activeAgents int
	var totalTasks int64

	// 에이전트 메트릭 집계
	c.agentMetrics.Range(func(key, value interface{}) bool {
		agentMetricsList := value.([]*AgentPerformanceMetrics)
		if len(agentMetricsList) > 0 {
			latest := agentMetricsList[len(agentMetricsList)-1]
			totalCPU += latest.CPUUsagePercent
			totalMemory += latest.MemoryUsagePercent
			totalTasks += latest.TasksCompleted
			activeAgents++
		}
		return true
	})

	// 풀 통계 수집
	poolStats := c.pool.GetStats()
	
	if activeAgents > 0 {
		c.aggregatedStats.AverageCPUUsage = totalCPU / float64(activeAgents)
		c.aggregatedStats.AverageMemoryUsage = totalMemory / float64(activeAgents)
	}

	c.aggregatedStats.TotalActiveAgents = activeAgents
	c.aggregatedStats.TotalTasksCompleted = totalTasks
	c.aggregatedStats.OverallSuccessRate = 0.9 // 임시값

	// 풀 통계
	if poolStats.PoolHits+poolStats.PoolMisses > 0 {
		c.aggregatedStats.PoolHitRate = float64(poolStats.PoolHits) / float64(poolStats.PoolHits+poolStats.PoolMisses)
	}

	if poolStats.TotalSize > 0 {
		c.aggregatedStats.PoolUtilization = float64(poolStats.RunningCount) / float64(poolStats.TotalSize)
	}

	if poolStats.TotalCreated > 0 {
		c.aggregatedStats.RecycleRate = float64(poolStats.TotalRecycled) / float64(poolStats.TotalCreated)
	}
}

// updatePrometheusMetrics Prometheus 메트릭 업데이트
func (c *PerformanceMetricsCollector) updatePrometheusMetrics() {
	if c.prometheusMetrics == nil {
		return
	}

	// 시스템 메트릭
	c.prometheusMetrics.SystemCPUUsage.Set(c.systemMetrics.SystemCPUUsagePercent)
	c.prometheusMetrics.SystemMemoryUsage.Set(c.systemMetrics.MemoryUsagePercent)
	c.prometheusMetrics.GoRoutines.Set(float64(c.systemMetrics.GoRoutines))
	c.prometheusMetrics.GoMemoryHeap.Set(float64(c.systemMetrics.GoMemoryHeapUsed))

	// 집계 메트릭
	c.prometheusMetrics.PoolUtilization.Set(c.aggregatedStats.PoolUtilization)
	c.prometheusMetrics.PoolHitRate.Set(c.aggregatedStats.PoolHitRate)

	// 에이전트별 메트릭
	c.agentMetrics.Range(func(key, value interface{}) bool {
		agentID := key.(string)
		agentMetricsList := value.([]*AgentPerformanceMetrics)
		
		if len(agentMetricsList) > 0 {
			latest := agentMetricsList[len(agentMetricsList)-1]
			c.prometheusMetrics.AgentCPUUsage.WithLabelValues(agentID).Set(latest.CPUUsagePercent)
			c.prometheusMetrics.AgentMemoryUsage.WithLabelValues(agentID).Set(latest.MemoryUsagePercent)
			c.prometheusMetrics.AgentTasksTotal.WithLabelValues(agentID).Add(float64(latest.TasksCompleted))
		}
		
		return true
	})
}

// cleanupLoop 정리 작업 루프
func (c *PerformanceMetricsCollector) cleanupLoop() {
	defer c.wg.Done()

	ticker := time.NewTicker(time.Hour) // 1시간마다 정리
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return

		case <-ticker.C:
			c.cleanupOldMetrics()
		}
	}
}

// cleanupOldMetrics 오래된 메트릭 정리
func (c *PerformanceMetricsCollector) cleanupOldMetrics() {
	cutoff := time.Now().Add(-c.config.RetentionPeriod)

	c.agentMetrics.Range(func(key, value interface{}) bool {
		agentMetricsList := value.([]*AgentPerformanceMetrics)
		var filtered []*AgentPerformanceMetrics

		for _, metrics := range agentMetricsList {
			if metrics.Timestamp.After(cutoff) {
				filtered = append(filtered, metrics)
			}
		}

		if len(filtered) > 0 {
			c.agentMetrics.Store(key, filtered)
		} else {
			c.agentMetrics.Delete(key)
		}

		return true
	})
}

// GetAgentMetrics 에이전트 메트릭 조회
func (c *PerformanceMetricsCollector) GetAgentMetrics(agentID string) ([]*AgentPerformanceMetrics, bool) {
	if value, ok := c.agentMetrics.Load(agentID); ok {
		return value.([]*AgentPerformanceMetrics), true
	}
	return nil, false
}

// GetSystemMetrics 시스템 메트릭 조회
func (c *PerformanceMetricsCollector) GetSystemMetrics() *SystemPerformanceMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.systemMetrics
}

// GetAggregatedMetrics 집계 메트릭 조회
func (c *PerformanceMetricsCollector) GetAggregatedMetrics() *AggregatedMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.aggregatedStats
}

// NewPrometheusMetrics 새 Prometheus 메트릭 생성
func NewPrometheusMetrics() *PrometheusMetrics {
	return &PrometheusMetrics{
		AgentCPUUsage: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "agent_cpu_usage_percent",
				Help: "CPU usage percentage per agent",
			},
			[]string{"agent_id"},
		),
		AgentMemoryUsage: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "agent_memory_usage_percent",
				Help: "Memory usage percentage per agent",
			},
			[]string{"agent_id"},
		),
		AgentTasksTotal: *prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "agent_tasks_total",
				Help: "Total number of tasks completed by agent",
			},
			[]string{"agent_id"},
		),
		AgentErrorsTotal: *prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "agent_errors_total",
				Help: "Total number of errors per agent",
			},
			[]string{"agent_id"},
		),
		AgentCreationTime: *prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "agent_creation_duration_seconds",
				Help:    "Time taken to create an agent",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"agent_id"},
		),
		SystemCPUUsage: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "system_cpu_usage_percent",
				Help: "System CPU usage percentage",
			},
		),
		SystemMemoryUsage: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "system_memory_usage_percent",
				Help: "System memory usage percentage",
			},
		),
		SystemDiskUsage: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "system_disk_usage_percent",
				Help: "System disk usage percentage",
			},
		),
		SystemLoadAverage: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "system_load_average",
				Help: "System load average",
			},
			[]string{"period"},
		),
		PoolSize: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "agent_pool_size_total",
				Help: "Total size of agent pool",
			},
		),
		PoolUtilization: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "agent_pool_utilization",
				Help: "Agent pool utilization rate",
			},
		),
		PoolHitRate: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "agent_pool_hit_rate",
				Help: "Agent pool hit rate",
			},
		),
		PoolRecycleRate: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "agent_pool_recycle_total",
				Help: "Total number of agent pool recycles",
			},
		),
		GoRoutines: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "go_goroutines",
				Help: "Number of goroutines",
			},
		),
		GoMemoryHeap: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "go_memory_heap_bytes",
				Help: "Go heap memory usage in bytes",
			},
		),
		GoGCDuration: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "go_gc_duration_seconds",
				Help:    "Go garbage collection duration",
				Buckets: prometheus.DefBuckets,
			},
		),
	}
}

// DefaultMetricsConfig 기본 메트릭 설정 반환
func DefaultMetricsConfig() MetricsConfig {
	return MetricsConfig{
		CollectionInterval:  10 * time.Second,
		RetentionPeriod:     24 * time.Hour,
		MaxMetricsPerAgent:  1000,
		EnableSystemMetrics: true,
		EnablePrometheus:    true,
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}