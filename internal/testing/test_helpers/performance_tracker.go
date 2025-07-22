package test_helpers

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// PerformanceTracker 성능 추적기
type PerformanceTracker struct {
	config   *TestConfig
	
	// 추적 상태
	running  atomic.Bool
	ctx      context.Context
	cancel   context.CancelFunc
	
	// 메트릭 데이터
	metrics  *PerformanceMetrics
	mu       sync.RWMutex
	
	// 샘플링 설정
	sampleInterval time.Duration
	maxSamples     int
}

// PerformanceMetrics 성능 메트릭
type PerformanceMetrics struct {
	// CPU 관련
	CPUSamples       []CPUSample       `json:"cpu_samples"`
	CPUUsagePercent  float64           `json:"cpu_usage_percent"`
	
	// 메모리 관련
	MemorySamples    []MemorySample    `json:"memory_samples"`
	CurrentMemory    int64             `json:"current_memory"`
	PeakMemory       int64             `json:"peak_memory"`
	
	// 고루틴 관련
	GoroutineSamples []GoroutineSample `json:"goroutine_samples"`
	CurrentGoroutines int              `json:"current_goroutines"`
	PeakGoroutines   int               `json:"peak_goroutines"`
	
	// GC 관련
	GCSamples        []GCSample        `json:"gc_samples"`
	TotalGCTime      time.Duration     `json:"total_gc_time"`
	GCCycles         int               `json:"gc_cycles"`
	
	// 네트워크 관련
	NetworkSamples   []NetworkSample   `json:"network_samples"`
	TotalBytesIn     int64             `json:"total_bytes_in"`
	TotalBytesOut    int64             `json:"total_bytes_out"`
	
	// 시간 정보
	StartTime        time.Time         `json:"start_time"`
	LastSampleTime   time.Time         `json:"last_sample_time"`
	SampleCount      int               `json:"sample_count"`
}

// CPUSample CPU 사용량 샘플
type CPUSample struct {
	Timestamp   time.Time `json:"timestamp"`
	UserTime    float64   `json:"user_time"`
	SystemTime  float64   `json:"system_time"`
	IdleTime    float64   `json:"idle_time"`
	Usage       float64   `json:"usage"`
}

// MemorySample 메모리 사용량 샘플
type MemorySample struct {
	Timestamp   time.Time `json:"timestamp"`
	Alloc       uint64    `json:"alloc"`
	TotalAlloc  uint64    `json:"total_alloc"`
	Sys         uint64    `json:"sys"`
	Heap        uint64    `json:"heap"`
	Stack       uint64    `json:"stack"`
	MSpanInuse  uint64    `json:"mspan_inuse"`
	MCacheInuse uint64    `json:"mcache_inuse"`
}

// GoroutineSample 고루틴 수 샘플
type GoroutineSample struct {
	Timestamp time.Time `json:"timestamp"`
	Count     int       `json:"count"`
	Runnable  int       `json:"runnable"`
	Running   int       `json:"running"`
	Syscall   int       `json:"syscall"`
	Waiting   int       `json:"waiting"`
}

// GCSample GC 통계 샘플
type GCSample struct {
	Timestamp     time.Time     `json:"timestamp"`
	NumGC         uint32        `json:"num_gc"`
	PauseTotalNs  uint64        `json:"pause_total_ns"`
	LastPauseNs   uint64        `json:"last_pause_ns"`
	TotalAllocMB  float64       `json:"total_alloc_mb"`
	TotalFreedMB  float64       `json:"total_freed_mb"`
}

// NetworkSample 네트워크 사용량 샘플
type NetworkSample struct {
	Timestamp   time.Time `json:"timestamp"`
	BytesIn     int64     `json:"bytes_in"`
	BytesOut    int64     `json:"bytes_out"`
	PacketsIn   int64     `json:"packets_in"`
	PacketsOut  int64     `json:"packets_out"`
	Errors      int64     `json:"errors"`
}

// ResourceUsage 리소스 사용량
type ResourceUsage struct {
	MemoryUsage     int64         `json:"memory_usage"`
	CPUUsage        float64       `json:"cpu_usage"`
	GoroutineCount  int           `json:"goroutine_count"`
	NetworkBandwidth int64        `json:"network_bandwidth"`
	DiskIO          int64         `json:"disk_io"`
	FileDescriptors int           `json:"file_descriptors"`
}

// NewPerformanceTracker 새로운 성능 추적기 생성
func NewPerformanceTracker(config *TestConfig) *PerformanceTracker {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &PerformanceTracker{
		config:         config,
		ctx:            ctx,
		cancel:         cancel,
		sampleInterval: 1 * time.Second,
		maxSamples:     1000,
		metrics: &PerformanceMetrics{
			CPUSamples:       make([]CPUSample, 0),
			MemorySamples:    make([]MemorySample, 0),
			GoroutineSamples: make([]GoroutineSample, 0),
			GCSamples:        make([]GCSample, 0),
			NetworkSamples:   make([]NetworkSample, 0),
			StartTime:        time.Now(),
		},
	}
}

// Start 성능 추적 시작
func (pt *PerformanceTracker) Start() {
	if !pt.running.CompareAndSwap(false, true) {
		return
	}
	
	go pt.trackingLoop()
}

// Stop 성능 추적 중지
func (pt *PerformanceTracker) Stop() {
	if !pt.running.CompareAndSwap(true, false) {
		return
	}
	
	pt.cancel()
}

// trackingLoop 추적 루프
func (pt *PerformanceTracker) trackingLoop() {
	ticker := time.NewTicker(pt.sampleInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-pt.ctx.Done():
			return
		case <-ticker.C:
			pt.takeSample()
		}
	}
}

// takeSample 샘플 수집
func (pt *PerformanceTracker) takeSample() {
	now := time.Now()
	
	pt.mu.Lock()
	defer pt.mu.Unlock()
	
	// CPU 샘플 수집
	cpuSample := pt.collectCPUSample(now)
	pt.metrics.CPUSamples = append(pt.metrics.CPUSamples, cpuSample)
	
	// 메모리 샘플 수집
	memorySample := pt.collectMemorySample(now)
	pt.metrics.MemorySamples = append(pt.metrics.MemorySamples, memorySample)
	pt.metrics.CurrentMemory = int64(memorySample.Alloc)
	if pt.metrics.CurrentMemory > pt.metrics.PeakMemory {
		pt.metrics.PeakMemory = pt.metrics.CurrentMemory
	}
	
	// 고루틴 샘플 수집
	goroutineSample := pt.collectGoroutineSample(now)
	pt.metrics.GoroutineSamples = append(pt.metrics.GoroutineSamples, goroutineSample)
	pt.metrics.CurrentGoroutines = goroutineSample.Count
	if pt.metrics.CurrentGoroutines > pt.metrics.PeakGoroutines {
		pt.metrics.PeakGoroutines = pt.metrics.CurrentGoroutines
	}
	
	// GC 샘플 수집
	gcSample := pt.collectGCSample(now)
	pt.metrics.GCSamples = append(pt.metrics.GCSamples, gcSample)
	
	// 네트워크 샘플 수집
	networkSample := pt.collectNetworkSample(now)
	pt.metrics.NetworkSamples = append(pt.metrics.NetworkSamples, networkSample)
	
	// 메타데이터 업데이트
	pt.metrics.LastSampleTime = now
	pt.metrics.SampleCount++
	
	// 샘플 수 제한
	pt.limitSamples()
}

// collectCPUSample CPU 샘플 수집
func (pt *PerformanceTracker) collectCPUSample(timestamp time.Time) CPUSample {
	// 실제 구현에서는 /proc/stat 또는 runtime 정보 사용
	// 테스트 환경에서는 시뮬레이션된 값 사용
	
	sample := CPUSample{
		Timestamp:  timestamp,
		UserTime:   0.5,  // 시뮬레이션된 값
		SystemTime: 0.2,  // 시뮬레이션된 값
		IdleTime:   0.3,  // 시뮬레이션된 값
		Usage:      0.7,  // 시뮬레이션된 값
	}
	
	return sample
}

// collectMemorySample 메모리 샘플 수집
func (pt *PerformanceTracker) collectMemorySample(timestamp time.Time) MemorySample {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	return MemorySample{
		Timestamp:   timestamp,
		Alloc:       m.Alloc,
		TotalAlloc:  m.TotalAlloc,
		Sys:         m.Sys,
		Heap:        m.HeapAlloc,
		Stack:       m.StackInuse,
		MSpanInuse:  m.MSpanInuse,
		MCacheInuse: m.MCacheInuse,
	}
}

// collectGoroutineSample 고루틴 샘플 수집
func (pt *PerformanceTracker) collectGoroutineSample(timestamp time.Time) GoroutineSample {
	goroutineCount := runtime.NumGoroutine()
	
	// 실제 구현에서는 runtime.Stack을 분석하여 상태별 개수 계산
	// 테스트에서는 근사값 사용
	return GoroutineSample{
		Timestamp: timestamp,
		Count:     goroutineCount,
		Runnable:  goroutineCount / 4,
		Running:   goroutineCount / 4,
		Syscall:   goroutineCount / 4,
		Waiting:   goroutineCount / 4,
	}
}

// collectGCSample GC 샘플 수집
func (pt *PerformanceTracker) collectGCSample(timestamp time.Time) GCSample {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	return GCSample{
		Timestamp:     timestamp,
		NumGC:         m.NumGC,
		PauseTotalNs:  m.PauseTotalNs,
		LastPauseNs:   m.PauseNs[(m.NumGC+255)%256],
		TotalAllocMB:  float64(m.TotalAlloc) / 1024 / 1024,
		TotalFreedMB:  float64(m.TotalAlloc - m.Alloc) / 1024 / 1024,
	}
}

// collectNetworkSample 네트워크 샘플 수집
func (pt *PerformanceTracker) collectNetworkSample(timestamp time.Time) NetworkSample {
	// 실제 구현에서는 /proc/net/dev 또는 netstat 사용
	// 테스트 환경에서는 시뮬레이션된 값 사용
	
	return NetworkSample{
		Timestamp:  timestamp,
		BytesIn:    1024 * 100, // 시뮬레이션된 값
		BytesOut:   1024 * 50,  // 시뮬레이션된 값
		PacketsIn:  100,        // 시뮬레이션된 값
		PacketsOut: 50,         // 시뮬레이션된 값
		Errors:     0,          // 시뮬레이션된 값
	}
}

// limitSamples 샘플 수 제한
func (pt *PerformanceTracker) limitSamples() {
	if len(pt.metrics.CPUSamples) > pt.maxSamples {
		pt.metrics.CPUSamples = pt.metrics.CPUSamples[1:]
	}
	
	if len(pt.metrics.MemorySamples) > pt.maxSamples {
		pt.metrics.MemorySamples = pt.metrics.MemorySamples[1:]
	}
	
	if len(pt.metrics.GoroutineSamples) > pt.maxSamples {
		pt.metrics.GoroutineSamples = pt.metrics.GoroutineSamples[1:]
	}
	
	if len(pt.metrics.GCSamples) > pt.maxSamples {
		pt.metrics.GCSamples = pt.metrics.GCSamples[1:]
	}
	
	if len(pt.metrics.NetworkSamples) > pt.maxSamples {
		pt.metrics.NetworkSamples = pt.metrics.NetworkSamples[1:]
	}
}

// GetMetrics 메트릭 조회
func (pt *PerformanceTracker) GetMetrics() PerformanceMetrics {
	pt.mu.RLock()
	defer pt.mu.RUnlock()
	
	// 깊은 복사 수행
	metrics := PerformanceMetrics{
		CPUSamples:       make([]CPUSample, len(pt.metrics.CPUSamples)),
		MemorySamples:    make([]MemorySample, len(pt.metrics.MemorySamples)),
		GoroutineSamples: make([]GoroutineSample, len(pt.metrics.GoroutineSamples)),
		GCSamples:        make([]GCSample, len(pt.metrics.GCSamples)),
		NetworkSamples:   make([]NetworkSample, len(pt.metrics.NetworkSamples)),
		
		CPUUsagePercent:   pt.metrics.CPUUsagePercent,
		CurrentMemory:     pt.metrics.CurrentMemory,
		PeakMemory:        pt.metrics.PeakMemory,
		CurrentGoroutines: pt.metrics.CurrentGoroutines,
		PeakGoroutines:    pt.metrics.PeakGoroutines,
		TotalGCTime:       pt.metrics.TotalGCTime,
		GCCycles:          pt.metrics.GCCycles,
		TotalBytesIn:      pt.metrics.TotalBytesIn,
		TotalBytesOut:     pt.metrics.TotalBytesOut,
		StartTime:         pt.metrics.StartTime,
		LastSampleTime:    pt.metrics.LastSampleTime,
		SampleCount:       pt.metrics.SampleCount,
	}
	
	copy(metrics.CPUSamples, pt.metrics.CPUSamples)
	copy(metrics.MemorySamples, pt.metrics.MemorySamples)
	copy(metrics.GoroutineSamples, pt.metrics.GoroutineSamples)
	copy(metrics.GCSamples, pt.metrics.GCSamples)
	copy(metrics.NetworkSamples, pt.metrics.NetworkSamples)
	
	return metrics
}

// GetResourceUsage 현재 리소스 사용량 조회
func (pt *PerformanceTracker) GetResourceUsage() ResourceUsage {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	return ResourceUsage{
		MemoryUsage:      int64(m.Alloc),
		CPUUsage:         pt.getCurrentCPUUsage(),
		GoroutineCount:   runtime.NumGoroutine(),
		NetworkBandwidth: pt.getCurrentNetworkBandwidth(),
		DiskIO:          pt.getCurrentDiskIO(),
		FileDescriptors: pt.getCurrentFileDescriptors(),
	}
}

// getCurrentCPUUsage 현재 CPU 사용률 조회
func (pt *PerformanceTracker) getCurrentCPUUsage() float64 {
	pt.mu.RLock()
	defer pt.mu.RUnlock()
	
	if len(pt.metrics.CPUSamples) == 0 {
		return 0.0
	}
	
	return pt.metrics.CPUSamples[len(pt.metrics.CPUSamples)-1].Usage
}

// getCurrentNetworkBandwidth 현재 네트워크 대역폭 사용량 조회
func (pt *PerformanceTracker) getCurrentNetworkBandwidth() int64 {
	pt.mu.RLock()
	defer pt.mu.RUnlock()
	
	if len(pt.metrics.NetworkSamples) < 2 {
		return 0
	}
	
	latest := pt.metrics.NetworkSamples[len(pt.metrics.NetworkSamples)-1]
	previous := pt.metrics.NetworkSamples[len(pt.metrics.NetworkSamples)-2]
	
	duration := latest.Timestamp.Sub(previous.Timestamp)
	if duration == 0 {
		return 0
	}
	
	totalBytes := (latest.BytesIn + latest.BytesOut) - (previous.BytesIn + previous.BytesOut)
	return int64(float64(totalBytes) / duration.Seconds())
}

// getCurrentDiskIO 현재 디스크 I/O 사용량 조회 (시뮬레이션)
func (pt *PerformanceTracker) getCurrentDiskIO() int64 {
	// 실제 구현에서는 /proc/diskstats 또는 iostat 사용
	return 1024 * 10 // 시뮬레이션된 값
}

// getCurrentFileDescriptors 현재 파일 디스크립터 수 조회 (시뮬레이션)
func (pt *PerformanceTracker) getCurrentFileDescriptors() int {
	// 실제 구현에서는 /proc/self/fd 디렉토리 카운트
	return 100 // 시뮬레이션된 값
}

// SetSampleInterval 샘플링 간격 설정
func (pt *PerformanceTracker) SetSampleInterval(interval time.Duration) {
	pt.sampleInterval = interval
}

// SetMaxSamples 최대 샘플 수 설정
func (pt *PerformanceTracker) SetMaxSamples(maxSamples int) {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	
	pt.maxSamples = maxSamples
	pt.limitSamples()
}

// Reset 메트릭 리셋
func (pt *PerformanceTracker) Reset() {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	
	pt.metrics = &PerformanceMetrics{
		CPUSamples:       make([]CPUSample, 0),
		MemorySamples:    make([]MemorySample, 0),
		GoroutineSamples: make([]GoroutineSample, 0),
		GCSamples:        make([]GCSample, 0),
		NetworkSamples:   make([]NetworkSample, 0),
		StartTime:        time.Now(),
	}
}

// GetSummary 성능 요약 정보 조회
func (pt *PerformanceTracker) GetSummary() PerformanceSummary {
	metrics := pt.GetMetrics()
	
	summary := PerformanceSummary{
		Duration:          time.Since(metrics.StartTime),
		SampleCount:       metrics.SampleCount,
		PeakMemoryUsage:   metrics.PeakMemory,
		PeakGoroutines:    metrics.PeakGoroutines,
		TotalGCTime:       metrics.TotalGCTime,
		AverageCPUUsage:   pt.calculateAverageCPUUsage(metrics.CPUSamples),
		AverageMemoryUsage: pt.calculateAverageMemoryUsage(metrics.MemorySamples),
	}
	
	return summary
}

// PerformanceSummary 성능 요약
type PerformanceSummary struct {
	Duration           time.Duration `json:"duration"`
	SampleCount        int           `json:"sample_count"`
	PeakMemoryUsage    int64         `json:"peak_memory_usage"`
	PeakGoroutines     int           `json:"peak_goroutines"`
	TotalGCTime        time.Duration `json:"total_gc_time"`
	AverageCPUUsage    float64       `json:"average_cpu_usage"`
	AverageMemoryUsage int64         `json:"average_memory_usage"`
}

// calculateAverageCPUUsage 평균 CPU 사용률 계산
func (pt *PerformanceTracker) calculateAverageCPUUsage(samples []CPUSample) float64 {
	if len(samples) == 0 {
		return 0.0
	}
	
	var total float64
	for _, sample := range samples {
		total += sample.Usage
	}
	
	return total / float64(len(samples))
}

// calculateAverageMemoryUsage 평균 메모리 사용량 계산
func (pt *PerformanceTracker) calculateAverageMemoryUsage(samples []MemorySample) int64 {
	if len(samples) == 0 {
		return 0
	}
	
	var total uint64
	for _, sample := range samples {
		total += sample.Alloc
	}
	
	return int64(total / uint64(len(samples)))
}