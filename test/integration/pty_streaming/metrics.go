package pty_streaming

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	docker_client "github.com/docker/docker/client"
)

// TestMetrics collects test metrics
type TestMetrics struct {
	TotalSessions    int64
	ActiveSessions   int64
	TotalMessages    int64
	ErrorCount       int64
	AverageLatency   time.Duration
	MemoryUsage      int64
	CPUUsage         float64
	startTime        time.Time
	latencies        []time.Duration
	mutex            sync.RWMutex
	ticker           *time.Ticker
	stopCh           chan struct{}
}

// NewTestMetrics creates a new test metrics collector
func NewTestMetrics() *TestMetrics {
	return &TestMetrics{
		startTime: time.Now(),
		latencies: make([]time.Duration, 0, 1000),
		stopCh:    make(chan struct{}),
	}
}

// Start starts metrics collection
func (tm *TestMetrics) Start() {
	tm.ticker = time.NewTicker(1 * time.Second)

	go func() {
		for {
			select {
			case <-tm.ticker.C:
				tm.updateSystemMetrics()
			case <-tm.stopCh:
				return
			}
		}
	}()
}

// Stop stops metrics collection
func (tm *TestMetrics) Stop() {
	if tm.ticker != nil {
		tm.ticker.Stop()
	}
	close(tm.stopCh)
}

// IncrementSessions increments session counters
func (tm *TestMetrics) IncrementSessions() {
	atomic.AddInt64(&tm.TotalSessions, 1)
	atomic.AddInt64(&tm.ActiveSessions, 1)
}

// DecrementActiveSessions decrements active session counter
func (tm *TestMetrics) DecrementActiveSessions() {
	atomic.AddInt64(&tm.ActiveSessions, -1)
}

// RecordMessage records a message with latency
func (tm *TestMetrics) RecordMessage(latency time.Duration) {
	atomic.AddInt64(&tm.TotalMessages, 1)

	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	tm.latencies = append(tm.latencies, latency)

	// 최대 1000개의 지연 시간 유지
	if len(tm.latencies) > 1000 {
		tm.latencies = tm.latencies[len(tm.latencies)-1000:]
	}

	// 이동 평균 계산
	tm.calculateAverageLatency()
}

// RecordError records an error
func (tm *TestMetrics) RecordError() {
	atomic.AddInt64(&tm.ErrorCount, 1)
}

// GetMemoryUsage returns current memory usage
func (tm *TestMetrics) GetMemoryUsage() int64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return int64(m.Alloc)
}

// GetReport generates a test report
func (tm *TestMetrics) GetReport() *TestReport {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	duration := time.Since(tm.startTime)
	throughput := float64(tm.TotalMessages) / duration.Seconds()

	errorRate := float64(0)
	if tm.TotalMessages > 0 {
		errorRate = float64(tm.ErrorCount) / float64(tm.TotalMessages)
	}

	return &TestReport{
		Duration:       duration,
		TotalSessions:  tm.TotalSessions,
		ActiveSessions: tm.ActiveSessions,
		TotalMessages:  tm.TotalMessages,
		ErrorCount:     tm.ErrorCount,
		ErrorRate:      errorRate,
		AverageLatency: tm.AverageLatency,
		Throughput:     throughput,
		MemoryUsage:    tm.MemoryUsage,
		CPUUsage:       tm.CPUUsage,
	}
}

// calculateAverageLatency calculates average latency
func (tm *TestMetrics) calculateAverageLatency() {
	if len(tm.latencies) == 0 {
		tm.AverageLatency = 0
		return
	}

	var total time.Duration
	for _, latency := range tm.latencies {
		total += latency
	}

	tm.AverageLatency = total / time.Duration(len(tm.latencies))
}

// updateSystemMetrics updates system metrics
func (tm *TestMetrics) updateSystemMetrics() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	tm.MemoryUsage = int64(m.Alloc)
	// CPU 사용률은 실제 구현에서는 더 정교한 방법이 필요
	tm.CPUUsage = 0
}

// TestReport represents a test execution report
type TestReport struct {
	Duration       time.Duration
	TotalSessions  int64
	ActiveSessions int64
	TotalMessages  int64
	ErrorCount     int64
	ErrorRate      float64
	AverageLatency time.Duration
	Throughput     float64
	MemoryUsage    int64
	CPUUsage       float64
}

// String returns a formatted report string
func (tr *TestReport) String() string {
	return fmt.Sprintf(`
Test Report:
============
Duration: %v
Total Sessions: %d
Active Sessions: %d
Total Messages: %d
Error Count: %d
Error Rate: %.2f%%
Average Latency: %v
Throughput: %.2f msg/sec
Memory Usage: %d MB
CPU Usage: %.2f%%
`,
		tr.Duration,
		tr.TotalSessions,
		tr.ActiveSessions,
		tr.TotalMessages,
		tr.ErrorCount,
		tr.ErrorRate*100,
		tr.AverageLatency,
		tr.Throughput,
		tr.MemoryUsage/(1024*1024),
		tr.CPUUsage,
	)
}

// ResourceMonitor monitors container resources
type ResourceMonitor struct {
	client      *docker_client.Client
	containerID string
	stats       []*ContainerResourceStats
	ticker      *time.Ticker
	stopCh      chan struct{}
	mutex       sync.RWMutex
}

// ContainerResourceStats represents container resource statistics
type ContainerResourceStats struct {
	CPUUsage    float64
	MemoryUsage int64
	Timestamp   time.Time
}

// NewResourceMonitor creates a new resource monitor
func NewResourceMonitor(client *docker_client.Client, containerID string) *ResourceMonitor {
	return &ResourceMonitor{
		client:      client,
		containerID: containerID,
		stats:       make([]*ContainerResourceStats, 0, 100),
		stopCh:      make(chan struct{}),
	}
}

// Start starts resource monitoring
func (rm *ResourceMonitor) Start() {
	rm.ticker = time.NewTicker(1 * time.Second)

	go func() {
		for {
			select {
			case <-rm.ticker.C:
				rm.collectStats()
			case <-rm.stopCh:
				return
			}
		}
	}()
}

// Stop stops resource monitoring
func (rm *ResourceMonitor) Stop() {
	if rm.ticker != nil {
		rm.ticker.Stop()
	}
	close(rm.stopCh)
}

// GetLatestStats returns the latest statistics
func (rm *ResourceMonitor) GetLatestStats() *ContainerResourceStats {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	if len(rm.stats) == 0 {
		return &ContainerResourceStats{}
	}

	return rm.stats[len(rm.stats)-1]
}

// collectStats collects container statistics
func (rm *ResourceMonitor) collectStats() {
	// 실제 구현에서는 Docker API를 통해 통계 수집
	stats := &ContainerResourceStats{
		CPUUsage:    0,
		MemoryUsage: 0,
		Timestamp:   time.Now(),
	}

	rm.mutex.Lock()
	rm.stats = append(rm.stats, stats)

	// 최대 100개의 통계 유지
	if len(rm.stats) > 100 {
		rm.stats = rm.stats[len(rm.stats)-100:]
	}
	rm.mutex.Unlock()
}