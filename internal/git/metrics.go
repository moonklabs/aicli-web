package git

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// PrometheusMetrics Git worktree 관련 Prometheus 메트릭
type PrometheusMetrics struct {
	// 카운터 메트릭
	worktreesCreated   prometheus.Counter
	worktreesDeleted   prometheus.Counter
	cloneOperations    prometheus.Counter
	cacheHits          prometheus.Counter
	cacheMisses        prometheus.Counter
	gcRuns             prometheus.Counter
	gcDeletedWorktrees prometheus.Counter

	// 게이지 메트릭
	activeWorktrees prometheus.Gauge
	cachedWorktrees prometheus.Gauge
	diskUsageBytes  prometheus.Gauge
	concurrentOps   prometheus.Gauge

	// 히스토그램 메트릭
	cloneDuration  prometheus.Histogram
	createDuration prometheus.Histogram
	deleteDuration prometheus.Histogram
	gcDuration     prometheus.Histogram

	// 서머리 메트릭
	cacheHitRate prometheus.Summary
}

// NewPrometheusMetrics Prometheus 메트릭 생성
func NewPrometheusMetrics() *PrometheusMetrics {
	return &PrometheusMetrics{
		// 카운터 메트릭
		worktreesCreated: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "git_worktrees_created_total",
			Help: "Total number of worktrees created",
		}),
		worktreesDeleted: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "git_worktrees_deleted_total",
			Help: "Total number of worktrees deleted",
		}),
		cloneOperations: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "git_clone_operations_total",
			Help: "Total number of clone operations",
		}),
		cacheHits: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "git_cache_hits_total",
			Help: "Total number of cache hits",
		}),
		cacheMisses: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "git_cache_misses_total",
			Help: "Total number of cache misses",
		}),
		gcRuns: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "git_gc_runs_total",
			Help: "Total number of garbage collection runs",
		}),
		gcDeletedWorktrees: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "git_gc_deleted_worktrees_total",
			Help: "Total number of worktrees deleted by garbage collection",
		}),

		// 게이지 메트릭
		activeWorktrees: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "git_active_worktrees",
			Help: "Current number of active worktrees",
		}),
		cachedWorktrees: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "git_cached_worktrees",
			Help: "Current number of cached worktrees",
		}),
		diskUsageBytes: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "git_disk_usage_bytes",
			Help: "Total disk usage of all worktrees in bytes",
		}),
		concurrentOps: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "git_concurrent_operations",
			Help: "Current number of concurrent operations",
		}),

		// 히스토그램 메트릭
		cloneDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "git_clone_duration_seconds",
			Help:    "Duration of clone operations in seconds",
			Buckets: prometheus.ExponentialBuckets(0.1, 2, 10), // 0.1s부터 시작, 2배씩 증가, 10개 버킷
		}),
		createDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "git_worktree_create_duration_seconds",
			Help:    "Duration of worktree creation in seconds",
			Buckets: prometheus.ExponentialBuckets(0.01, 2, 8), // 0.01s부터 시작
		}),
		deleteDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "git_worktree_delete_duration_seconds",
			Help:    "Duration of worktree deletion in seconds",
			Buckets: prometheus.ExponentialBuckets(0.01, 2, 8),
		}),
		gcDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "git_gc_duration_seconds",
			Help:    "Duration of garbage collection runs in seconds",
			Buckets: prometheus.ExponentialBuckets(1, 2, 10), // 1s부터 시작
		}),

		// 서머리 메트릭
		cacheHitRate: prometheus.NewSummary(prometheus.SummaryOpts{
			Name:       "git_cache_hit_rate",
			Help:       "Cache hit rate over the last 10 minutes",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
			MaxAge:     10 * time.Minute,
		}),
	}
}

// RecordWorktreeCreated worktree 생성 기록
func (m *PrometheusMetrics) RecordWorktreeCreated() {
	m.worktreesCreated.Inc()
	m.activeWorktrees.Inc()
}

// RecordWorktreeDeleted worktree 삭제 기록
func (m *PrometheusMetrics) RecordWorktreeDeleted() {
	m.worktreesDeleted.Inc()
	m.activeWorktrees.Dec()
}

// RecordCloneOperation 클론 작업 기록
func (m *PrometheusMetrics) RecordCloneOperation(duration time.Duration) {
	m.cloneOperations.Inc()
	m.cloneDuration.Observe(duration.Seconds())
}

// RecordWorktreeCreate worktree 생성 시간 기록
func (m *PrometheusMetrics) RecordWorktreeCreate(duration time.Duration) {
	m.createDuration.Observe(duration.Seconds())
}

// RecordWorktreeDelete worktree 삭제 시간 기록
func (m *PrometheusMetrics) RecordWorktreeDelete(duration time.Duration) {
	m.deleteDuration.Observe(duration.Seconds())
}

// RecordCacheHit 캐시 히트 기록
func (m *PrometheusMetrics) RecordCacheHit() {
	m.cacheHits.Inc()
	m.cacheHitRate.Observe(1)
}

// RecordCacheMiss 캐시 미스 기록
func (m *PrometheusMetrics) RecordCacheMiss() {
	m.cacheMisses.Inc()
	m.cacheHitRate.Observe(0)
}

// RecordGCRun GC 실행 기록
func (m *PrometheusMetrics) RecordGCRun(duration time.Duration, deletedCount int) {
	m.gcRuns.Inc()
	m.gcDuration.Observe(duration.Seconds())
	m.gcDeletedWorktrees.Add(float64(deletedCount))
}

// SetActiveWorktrees 활성 worktree 수 설정
func (m *PrometheusMetrics) SetActiveWorktrees(count int) {
	m.activeWorktrees.Set(float64(count))
}

// SetCachedWorktrees 캐시된 worktree 수 설정
func (m *PrometheusMetrics) SetCachedWorktrees(count int) {
	m.cachedWorktrees.Set(float64(count))
}

// SetDiskUsage 디스크 사용량 설정
func (m *PrometheusMetrics) SetDiskUsage(bytes int64) {
	m.diskUsageBytes.Set(float64(bytes))
}

// SetConcurrentOps 동시 작업 수 설정
func (m *PrometheusMetrics) SetConcurrentOps(count int) {
	m.concurrentOps.Set(float64(count))
}

// MetricsCollector 메트릭 수집기
type MetricsCollector struct {
	manager  *AdvancedWorktreeManager
	metrics  *PrometheusMetrics
	stopCh   chan struct{}
	interval time.Duration
}

// NewMetricsCollector 새로운 메트릭 수집기 생성
func NewMetricsCollector(manager *AdvancedWorktreeManager, interval time.Duration) *MetricsCollector {
	if interval <= 0 {
		interval = 30 * time.Second
	}

	return &MetricsCollector{
		manager:  manager,
		metrics:  NewPrometheusMetrics(),
		stopCh:   make(chan struct{}),
		interval: interval,
	}
}

// Start 메트릭 수집 시작
func (c *MetricsCollector) Start() {
	go c.collectLoop()
}

// Stop 메트릭 수집 중지
func (c *MetricsCollector) Stop() {
	close(c.stopCh)
}

// collectLoop 메트릭 수집 루프
func (c *MetricsCollector) collectLoop() {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	// 초기 수집
	c.collect()

	for {
		select {
		case <-ticker.C:
			c.collect()
		case <-c.stopCh:
			return
		}
	}
}

// collect 메트릭 수집
func (c *MetricsCollector) collect() {
	metrics := c.manager.GetMetrics()

	// 게이지 메트릭 업데이트
	c.metrics.SetActiveWorktrees(metrics.ActiveWorktrees)
	c.metrics.SetCachedWorktrees(metrics.CachedWorktrees)
	c.metrics.SetDiskUsage(metrics.DiskUsageBytes)

	// TODO: 디스크 사용량 계산 구현
	// diskUsage := c.calculateDiskUsage()
	// c.metrics.SetDiskUsage(diskUsage)
}

// GetPrometheusMetrics Prometheus 메트릭 반환
func (c *MetricsCollector) GetPrometheusMetrics() *PrometheusMetrics {
	return c.metrics
}
