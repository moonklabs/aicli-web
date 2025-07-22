package test_helpers

import (
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// MetricsCollector 테스트 메트릭 수집기
type MetricsCollector struct {
	// 성능 메트릭
	messagesSent      int64
	messagesReceived  int64
	totalLatency      int64 // 나노초
	latencyCount      int64
	errorCount        int64
	successCount      int64
	
	// 처리량 메트릭
	startTime         time.Time
	lastMetricTime    time.Time
	
	// 세부 메트릭
	latencies         []time.Duration
	latenciesMutex    sync.RWMutex
	
	// 리소스 메트릭
	memorySnapshots   []MemorySnapshot
	memoryMutex       sync.RWMutex
	
	// 커스텀 메트릭
	customMetrics     map[string]float64
	customMutex       sync.RWMutex
}

// TestMetrics 테스트 메트릭 집계
type TestMetrics struct {
	// 처리량 지표
	Throughput        float64       `json:"throughput"`         // 초당 처리량
	MessagesSent      int64         `json:"messages_sent"`
	MessagesReceived  int64         `json:"messages_received"`
	
	// 지연시간 지표
	AverageLatency    time.Duration `json:"average_latency"`
	MedianLatency     time.Duration `json:"median_latency"`
	P95Latency        time.Duration `json:"p95_latency"`
	P99Latency        time.Duration `json:"p99_latency"`
	MinLatency        time.Duration `json:"min_latency"`
	MaxLatency        time.Duration `json:"max_latency"`
	
	// 성공률 지표
	SuccessRate       float64       `json:"success_rate"`
	ErrorRate         float64       `json:"error_rate"`
	TotalRequests     int64         `json:"total_requests"`
	SuccessfulRequests int64        `json:"successful_requests"`
	FailedRequests    int64         `json:"failed_requests"`
	
	// 리소스 사용량
	PeakMemoryUsage   int64         `json:"peak_memory_usage"`
	AverageMemoryUsage int64        `json:"average_memory_usage"`
	GoroutineCount    int           `json:"goroutine_count"`
	
	// 시간 정보
	TestDuration      time.Duration `json:"test_duration"`
	StartTime         time.Time     `json:"start_time"`
	EndTime           time.Time     `json:"end_time"`
	
	// 커스텀 메트릭
	CustomMetrics     map[string]float64 `json:"custom_metrics"`
}

// MemorySnapshot 메모리 스냅샷
type MemorySnapshot struct {
	Timestamp    time.Time `json:"timestamp"`
	Alloc        uint64    `json:"alloc"`         // 할당된 메모리
	TotalAlloc   uint64    `json:"total_alloc"`   // 총 할당량
	Sys          uint64    `json:"sys"`           // 시스템에서 얻은 메모리
	Lookups      uint64    `json:"lookups"`       // 포인터 룩업 수
	Mallocs      uint64    `json:"mallocs"`       // 할당 횟수
	Frees        uint64    `json:"frees"`         // 해제 횟수
	GCCycles     uint32    `json:"gc_cycles"`     // GC 사이클 수
	NumGoroutine int       `json:"num_goroutine"` // 고루틴 수
}

// NewMetricsCollector 새로운 메트릭 수집기 생성
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		startTime:     time.Now(),
		lastMetricTime: time.Now(),
		latencies:     make([]time.Duration, 0),
		memorySnapshots: make([]MemorySnapshot, 0),
		customMetrics: make(map[string]float64),
	}
}

// RecordMessageSent 메시지 전송 기록
func (mc *MetricsCollector) RecordMessageSent() {
	atomic.AddInt64(&mc.messagesSent, 1)
}

// RecordMessageReceived 메시지 수신 기록
func (mc *MetricsCollector) RecordMessageReceived() {
	atomic.AddInt64(&mc.messagesReceived, 1)
}

// RecordLatency 지연시간 기록
func (mc *MetricsCollector) RecordLatency(latency time.Duration) {
	atomic.AddInt64(&mc.totalLatency, int64(latency))
	atomic.AddInt64(&mc.latencyCount, 1)
	
	mc.latenciesMutex.Lock()
	mc.latencies = append(mc.latencies, latency)
	mc.latenciesMutex.Unlock()
}

// RecordSuccess 성공 기록
func (mc *MetricsCollector) RecordSuccess() {
	atomic.AddInt64(&mc.successCount, 1)
}

// RecordError 오류 기록
func (mc *MetricsCollector) RecordError() {
	atomic.AddInt64(&mc.errorCount, 1)
}

// RecordMessageExchange 메시지 교환 기록 (편의 메서드)
func (mc *MetricsCollector) RecordMessageExchange(sentSize, receivedSize int) {
	start := time.Now()
	
	mc.RecordMessageSent()
	
	// 실제로는 비동기적으로 처리되지만 테스트에서는 동기적으로 처리
	time.Sleep(time.Millisecond) // 최소 처리 시간 시뮬레이션
	
	mc.RecordMessageReceived()
	mc.RecordLatency(time.Since(start))
	mc.RecordSuccess()
}

// RecordCustomMetric 커스텀 메트릭 기록
func (mc *MetricsCollector) RecordCustomMetric(name string, value float64) {
	mc.customMutex.Lock()
	mc.customMetrics[name] = value
	mc.customMutex.Unlock()
}

// TakeMemorySnapshot 메모리 스냅샷 수집
func (mc *MetricsCollector) TakeMemorySnapshot() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	snapshot := MemorySnapshot{
		Timestamp:    time.Now(),
		Alloc:        m.Alloc,
		TotalAlloc:   m.TotalAlloc,
		Sys:          m.Sys,
		Lookups:      m.Lookups,
		Mallocs:      m.Mallocs,
		Frees:        m.Frees,
		GCCycles:     m.NumGC,
		NumGoroutine: runtime.NumGoroutine(),
	}
	
	mc.memoryMutex.Lock()
	mc.memorySnapshots = append(mc.memorySnapshots, snapshot)
	
	// 최대 1000개 스냅샷 유지
	if len(mc.memorySnapshots) > 1000 {
		mc.memorySnapshots = mc.memorySnapshots[1:]
	}
	mc.memoryMutex.Unlock()
}

// GetMetrics 메트릭 집계 조회
func (mc *MetricsCollector) GetMetrics() TestMetrics {
	now := time.Now()
	duration := now.Sub(mc.startTime)
	
	// 기본 카운터 조회
	messagesSent := atomic.LoadInt64(&mc.messagesSent)
	messagesReceived := atomic.LoadInt64(&mc.messagesReceived)
	successCount := atomic.LoadInt64(&mc.successCount)
	errorCount := atomic.LoadInt64(&mc.errorCount)
	totalRequests := successCount + errorCount
	
	// 처리량 계산
	throughput := float64(successCount) / duration.Seconds()
	
	// 성공률 계산
	var successRate float64
	if totalRequests > 0 {
		successRate = float64(successCount) / float64(totalRequests)
	}
	
	// 지연시간 통계 계산
	latencyStats := mc.calculateLatencyStats()
	
	// 메모리 사용량 통계 계산
	memoryStats := mc.calculateMemoryStats()
	
	// 커스텀 메트릭 복사
	mc.customMutex.RLock()
	customMetrics := make(map[string]float64)
	for k, v := range mc.customMetrics {
		customMetrics[k] = v
	}
	mc.customMutex.RUnlock()
	
	return TestMetrics{
		Throughput:         throughput,
		MessagesSent:       messagesSent,
		MessagesReceived:   messagesReceived,
		AverageLatency:     latencyStats.Average,
		MedianLatency:      latencyStats.Median,
		P95Latency:         latencyStats.P95,
		P99Latency:         latencyStats.P99,
		MinLatency:         latencyStats.Min,
		MaxLatency:         latencyStats.Max,
		SuccessRate:        successRate,
		ErrorRate:          1.0 - successRate,
		TotalRequests:      totalRequests,
		SuccessfulRequests: successCount,
		FailedRequests:     errorCount,
		PeakMemoryUsage:    memoryStats.Peak,
		AverageMemoryUsage: memoryStats.Average,
		GoroutineCount:     runtime.NumGoroutine(),
		TestDuration:       duration,
		StartTime:          mc.startTime,
		EndTime:            now,
		CustomMetrics:      customMetrics,
	}
}

// LatencyStats 지연시간 통계
type LatencyStats struct {
	Average time.Duration
	Median  time.Duration
	P95     time.Duration
	P99     time.Duration
	Min     time.Duration
	Max     time.Duration
}

// calculateLatencyStats 지연시간 통계 계산
func (mc *MetricsCollector) calculateLatencyStats() LatencyStats {
	mc.latenciesMutex.RLock()
	latencies := make([]time.Duration, len(mc.latencies))
	copy(latencies, mc.latencies)
	mc.latenciesMutex.RUnlock()
	
	if len(latencies) == 0 {
		return LatencyStats{}
	}
	
	// 정렬
	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i] < latencies[j]
	})
	
	// 통계 계산
	var total time.Duration
	for _, latency := range latencies {
		total += latency
	}
	
	n := len(latencies)
	average := total / time.Duration(n)
	median := latencies[n/2]
	p95 := latencies[int(float64(n)*0.95)]
	p99 := latencies[int(float64(n)*0.99)]
	min := latencies[0]
	max := latencies[n-1]
	
	return LatencyStats{
		Average: average,
		Median:  median,
		P95:     p95,
		P99:     p99,
		Min:     min,
		Max:     max,
	}
}

// MemoryStats 메모리 통계
type MemoryStats struct {
	Peak    int64
	Average int64
	Current int64
}

// calculateMemoryStats 메모리 통계 계산
func (mc *MetricsCollector) calculateMemoryStats() MemoryStats {
	mc.memoryMutex.RLock()
	snapshots := make([]MemorySnapshot, len(mc.memorySnapshots))
	copy(snapshots, mc.memorySnapshots)
	mc.memoryMutex.RUnlock()
	
	if len(snapshots) == 0 {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		current := int64(m.Alloc)
		return MemoryStats{
			Peak:    current,
			Average: current,
			Current: current,
		}
	}
	
	var total uint64
	var peak uint64
	
	for _, snapshot := range snapshots {
		total += snapshot.Alloc
		if snapshot.Alloc > peak {
			peak = snapshot.Alloc
		}
	}
	
	average := total / uint64(len(snapshots))
	current := snapshots[len(snapshots)-1].Alloc
	
	return MemoryStats{
		Peak:    int64(peak),
		Average: int64(average),
		Current: int64(current),
	}
}

// Reset 메트릭 리셋
func (mc *MetricsCollector) Reset() {
	atomic.StoreInt64(&mc.messagesSent, 0)
	atomic.StoreInt64(&mc.messagesReceived, 0)
	atomic.StoreInt64(&mc.totalLatency, 0)
	atomic.StoreInt64(&mc.latencyCount, 0)
	atomic.StoreInt64(&mc.errorCount, 0)
	atomic.StoreInt64(&mc.successCount, 0)
	
	mc.startTime = time.Now()
	mc.lastMetricTime = time.Now()
	
	mc.latenciesMutex.Lock()
	mc.latencies = mc.latencies[:0]
	mc.latenciesMutex.Unlock()
	
	mc.memoryMutex.Lock()
	mc.memorySnapshots = mc.memorySnapshots[:0]
	mc.memoryMutex.Unlock()
	
	mc.customMutex.Lock()
	for k := range mc.customMetrics {
		delete(mc.customMetrics, k)
	}
	mc.customMutex.Unlock()
}

// StartMemoryMonitoring 메모리 모니터링 시작
func (mc *MetricsCollector) StartMemoryMonitoring(interval time.Duration) chan struct{} {
	done := make(chan struct{})
	
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		
		for {
			select {
			case <-ticker.C:
				mc.TakeMemorySnapshot()
			case <-done:
				return
			}
		}
	}()
	
	return done
}

// GenerateReport 테스트 리포트 생성
func (mc *MetricsCollector) GenerateReport() TestReport {
	metrics := mc.GetMetrics()
	
	report := TestReport{
		Summary: ReportSummary{
			TestDuration:    metrics.TestDuration,
			TotalRequests:   metrics.TotalRequests,
			SuccessfulRequests: metrics.SuccessfulRequests,
			FailedRequests:  metrics.FailedRequests,
			SuccessRate:     metrics.SuccessRate,
			Throughput:      metrics.Throughput,
		},
		Performance: PerformanceReport{
			AverageLatency: metrics.AverageLatency,
			MedianLatency:  metrics.MedianLatency,
			P95Latency:     metrics.P95Latency,
			P99Latency:     metrics.P99Latency,
			MinLatency:     metrics.MinLatency,
			MaxLatency:     metrics.MaxLatency,
		},
		Resources: ResourceReport{
			PeakMemoryUsage:    metrics.PeakMemoryUsage,
			AverageMemoryUsage: metrics.AverageMemoryUsage,
			GoroutineCount:     metrics.GoroutineCount,
		},
		CustomMetrics: metrics.CustomMetrics,
		Timestamp:     time.Now(),
	}
	
	return report
}

// TestReport 테스트 리포트
type TestReport struct {
	Summary       ReportSummary     `json:"summary"`
	Performance   PerformanceReport `json:"performance"`
	Resources     ResourceReport    `json:"resources"`
	CustomMetrics map[string]float64 `json:"custom_metrics"`
	Timestamp     time.Time         `json:"timestamp"`
}

// ReportSummary 리포트 요약
type ReportSummary struct {
	TestDuration       time.Duration `json:"test_duration"`
	TotalRequests      int64         `json:"total_requests"`
	SuccessfulRequests int64         `json:"successful_requests"`
	FailedRequests     int64         `json:"failed_requests"`
	SuccessRate        float64       `json:"success_rate"`
	Throughput         float64       `json:"throughput"`
}

// PerformanceReport 성능 리포트
type PerformanceReport struct {
	AverageLatency time.Duration `json:"average_latency"`
	MedianLatency  time.Duration `json:"median_latency"`
	P95Latency     time.Duration `json:"p95_latency"`
	P99Latency     time.Duration `json:"p99_latency"`
	MinLatency     time.Duration `json:"min_latency"`
	MaxLatency     time.Duration `json:"max_latency"`
}

// ResourceReport 리소스 리포트
type ResourceReport struct {
	PeakMemoryUsage    int64 `json:"peak_memory_usage"`
	AverageMemoryUsage int64 `json:"average_memory_usage"`
	GoroutineCount     int   `json:"goroutine_count"`
}