package claude

import (
	"math"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// PoolMetrics는 풀 메트릭을 수집하고 관리합니다
type PoolMetrics struct {
	// 카운터들
	totalRequests    atomic.Int64
	successRequests  atomic.Int64
	failedRequests   atomic.Int64
	totalSessions    atomic.Int64
	activeSessions   atomic.Int64
	
	// 지연시간 추적
	latencyTracker *LatencyTracker
	
	// 처리량 추적
	throughputTracker *ThroughputTracker
	
	// 에러율 추적
	errorRateTracker *ErrorRateTracker
	
	// 액션 추적
	actionCounter map[string]*atomic.Int64
	actionMutex   sync.RWMutex
	
	// 시계열 메트릭
	timeSeriesMetrics *TimeSeriesMetrics
	
	// 상태 관리
	startTime time.Time
	running   atomic.Bool
	
	// 생명주기 관리
	ticker *time.Ticker
	done   chan struct{}
}

// LatencyTracker는 지연시간을 추적합니다
type LatencyTracker struct {
	samples    []time.Duration
	mutex      sync.RWMutex
	maxSamples int
}

// ThroughputTracker는 처리량을 추적합니다
type ThroughputTracker struct {
	windows    []ThroughputWindow
	mutex      sync.RWMutex
	windowSize time.Duration
	maxWindows int
}

// ThroughputWindow은 처리량 윈도우입니다
type ThroughputWindow struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Requests  int64     `json:"requests"`
	RPS       float64   `json:"rps"`
}

// ErrorRateTracker는 에러율을 추적합니다
type ErrorRateTracker struct {
	windows    []ErrorWindow
	mutex      sync.RWMutex
	windowSize time.Duration
	maxWindows int
}

// ErrorWindow은 에러 윈도우입니다
type ErrorWindow struct {
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	TotalRequests int64    `json:"total_requests"`
	ErrorRequests int64    `json:"error_requests"`
	ErrorRate    float64   `json:"error_rate"`
}

// TimeSeriesMetrics는 시계열 메트릭입니다
type TimeSeriesMetrics struct {
	dataPoints []MetricDataPoint
	mutex      sync.RWMutex
	maxPoints  int
}

// MetricDataPoint는 메트릭 데이터 포인트입니다
type MetricDataPoint struct {
	Timestamp       time.Time     `json:"timestamp"`
	ActiveSessions  int64         `json:"active_sessions"`
	TotalRequests   int64         `json:"total_requests"`
	RequestsPerSec  float64       `json:"requests_per_sec"`
	AverageLatency  time.Duration `json:"average_latency"`
	ErrorRate       float64       `json:"error_rate"`
	MemoryUsage     int64         `json:"memory_usage"`
	CPUUsage        float64       `json:"cpu_usage"`
}

// MetricsSummary는 메트릭 요약입니다
type MetricsSummary struct {
	// 기본 통계
	TotalRequests   int64         `json:"total_requests"`
	SuccessRequests int64         `json:"success_requests"`
	FailedRequests  int64         `json:"failed_requests"`
	SuccessRate     float64       `json:"success_rate"`
	ErrorRate       float64       `json:"error_rate"`
	
	// 지연시간 통계
	AverageLatency  time.Duration `json:"average_latency"`
	MedianLatency   time.Duration `json:"median_latency"`
	P95Latency      time.Duration `json:"p95_latency"`
	P99Latency      time.Duration `json:"p99_latency"`
	MinLatency      time.Duration `json:"min_latency"`
	MaxLatency      time.Duration `json:"max_latency"`
	
	// 처리량 통계
	CurrentRPS      float64       `json:"current_rps"`
	AverageRPS      float64       `json:"average_rps"`
	PeakRPS         float64       `json:"peak_rps"`
	
	// 세션 통계
	TotalSessions   int64         `json:"total_sessions"`
	ActiveSessions  int64         `json:"active_sessions"`
	SessionUtilization float64    `json:"session_utilization"`
	
	// 시간 정보
	StartTime       time.Time     `json:"start_time"`
	Uptime          time.Duration `json:"uptime"`
	LastUpdate      time.Time     `json:"last_update"`
	
	// 액션별 통계
	ActionCounts    map[string]int64 `json:"action_counts"`
}

// NewPoolMetrics는 새로운 풀 메트릭을 생성합니다
func NewPoolMetrics() *PoolMetrics {
	pm := &PoolMetrics{
		latencyTracker:     NewLatencyTracker(1000),
		throughputTracker:  NewThroughputTracker(time.Minute, 60),
		errorRateTracker:   NewErrorRateTracker(time.Minute, 60),
		timeSeriesMetrics:  NewTimeSeriesMetrics(1440), // 24시간 (1분 간격)
		actionCounter:      make(map[string]*atomic.Int64),
		startTime:          time.Now(),
		ticker:             time.NewTicker(time.Minute),
		done:               make(chan struct{}),
	}
	
	return pm
}

// NewLatencyTracker는 새로운 지연시간 추적기를 생성합니다
func NewLatencyTracker(maxSamples int) *LatencyTracker {
	return &LatencyTracker{
		samples:    make([]time.Duration, 0, maxSamples),
		maxSamples: maxSamples,
	}
}

// NewThroughputTracker는 새로운 처리량 추적기를 생성합니다
func NewThroughputTracker(windowSize time.Duration, maxWindows int) *ThroughputTracker {
	return &ThroughputTracker{
		windows:    make([]ThroughputWindow, 0, maxWindows),
		windowSize: windowSize,
		maxWindows: maxWindows,
	}
}

// NewErrorRateTracker는 새로운 에러율 추적기를 생성합니다
func NewErrorRateTracker(windowSize time.Duration, maxWindows int) *ErrorRateTracker {
	return &ErrorRateTracker{
		windows:    make([]ErrorWindow, 0, maxWindows),
		windowSize: windowSize,
		maxWindows: maxWindows,
	}
}

// NewTimeSeriesMetrics는 새로운 시계열 메트릭을 생성합니다
func NewTimeSeriesMetrics(maxPoints int) *TimeSeriesMetrics {
	return &TimeSeriesMetrics{
		dataPoints: make([]MetricDataPoint, 0, maxPoints),
		maxPoints:  maxPoints,
	}
}

// Start는 메트릭 수집을 시작합니다
func (pm *PoolMetrics) Start() {
	if !pm.running.CompareAndSwap(false, true) {
		return // 이미 실행 중
	}
	
	go pm.metricsLoop()
}

// Stop은 메트릭 수집을 중지합니다
func (pm *PoolMetrics) Stop() {
	if !pm.running.CompareAndSwap(true, false) {
		return // 이미 중지됨
	}
	
	close(pm.done)
	pm.ticker.Stop()
}

// RecordAction은 액션을 기록합니다
func (pm *PoolMetrics) RecordAction(action string, session *PooledSession) {
	// 액션 카운터 증가
	pm.incrementActionCounter(action)
	
	// 요청 카운터 업데이트
	switch action {
	case "acquired", "affinity_hit", "load_balanced":
		pm.totalRequests.Add(1)
		pm.successRequests.Add(1)
	case "acquisition_failed", "release_failed":
		pm.totalRequests.Add(1)
		pm.failedRequests.Add(1)
	case "released":
		// 세션 반환은 별도 처리
	}
	
	// 처리량 추적기에 기록
	pm.throughputTracker.RecordRequest()
}

// RecordLatency는 지연시간을 기록합니다
func (pm *PoolMetrics) RecordLatency(latency time.Duration) {
	pm.latencyTracker.AddSample(latency)
}

// RecordError는 에러를 기록합니다
func (pm *PoolMetrics) RecordError() {
	pm.failedRequests.Add(1)
	pm.errorRateTracker.RecordError()
}

// UpdateSessionCount는 세션 수를 업데이트합니다
func (pm *PoolMetrics) UpdateSessionCount(total, active int64) {
	pm.totalSessions.Store(total)
	pm.activeSessions.Store(active)
}

// GetThroughputRPS는 현재 처리량(RPS)을 반환합니다
func (pm *PoolMetrics) GetThroughputRPS() float64 {
	return pm.throughputTracker.GetCurrentRPS()
}

// GetAverageLatency는 평균 지연시간을 반환합니다
func (pm *PoolMetrics) GetAverageLatency() time.Duration {
	return pm.latencyTracker.GetAverage()
}

// GetErrorRate는 현재 에러율을 반환합니다
func (pm *PoolMetrics) GetErrorRate() float64 {
	return pm.errorRateTracker.GetCurrentErrorRate()
}

// GetMetricsSummary는 메트릭 요약을 반환합니다
func (pm *PoolMetrics) GetMetricsSummary() MetricsSummary {
	totalReq := pm.totalRequests.Load()
	successReq := pm.successRequests.Load()
	failedReq := pm.failedRequests.Load()
	
	var successRate, errorRate float64
	if totalReq > 0 {
		successRate = float64(successReq) / float64(totalReq)
		errorRate = float64(failedReq) / float64(totalReq)
	}
	
	// 액션 카운트 복사
	pm.actionMutex.RLock()
	actionCounts := make(map[string]int64)
	for action, counter := range pm.actionCounter {
		actionCounts[action] = counter.Load()
	}
	pm.actionMutex.RUnlock()
	
	return MetricsSummary{
		TotalRequests:      totalReq,
		SuccessRequests:    successReq,
		FailedRequests:     failedReq,
		SuccessRate:        successRate,
		ErrorRate:          errorRate,
		AverageLatency:     pm.latencyTracker.GetAverage(),
		MedianLatency:      pm.latencyTracker.GetPercentile(50),
		P95Latency:         pm.latencyTracker.GetPercentile(95),
		P99Latency:         pm.latencyTracker.GetPercentile(99),
		MinLatency:         pm.latencyTracker.GetMin(),
		MaxLatency:         pm.latencyTracker.GetMax(),
		CurrentRPS:         pm.throughputTracker.GetCurrentRPS(),
		AverageRPS:         pm.throughputTracker.GetAverageRPS(),
		PeakRPS:            pm.throughputTracker.GetPeakRPS(),
		TotalSessions:      pm.totalSessions.Load(),
		ActiveSessions:     pm.activeSessions.Load(),
		SessionUtilization: pm.calculateSessionUtilization(),
		StartTime:          pm.startTime,
		Uptime:             time.Since(pm.startTime),
		LastUpdate:         time.Now(),
		ActionCounts:       actionCounts,
	}
}

// GetTimeSeriesData는 시계열 데이터를 반환합니다
func (pm *PoolMetrics) GetTimeSeriesData() []MetricDataPoint {
	return pm.timeSeriesMetrics.GetDataPoints()
}

// 내부 메서드들

func (pm *PoolMetrics) metricsLoop() {
	for {
		select {
		case <-pm.done:
			return
		case <-pm.ticker.C:
			pm.collectTimeSeriesData()
		}
	}
}

func (pm *PoolMetrics) collectTimeSeriesData() {
	dataPoint := MetricDataPoint{
		Timestamp:      time.Now(),
		ActiveSessions: pm.activeSessions.Load(),
		TotalRequests:  pm.totalRequests.Load(),
		RequestsPerSec: pm.throughputTracker.GetCurrentRPS(),
		AverageLatency: pm.latencyTracker.GetAverage(),
		ErrorRate:      pm.errorRateTracker.GetCurrentErrorRate(),
		// MemoryUsage와 CPUUsage는 외부에서 설정해야 함
	}
	
	pm.timeSeriesMetrics.AddDataPoint(dataPoint)
}

func (pm *PoolMetrics) incrementActionCounter(action string) {
	pm.actionMutex.RLock()
	counter, exists := pm.actionCounter[action]
	pm.actionMutex.RUnlock()
	
	if !exists {
		pm.actionMutex.Lock()
		counter, exists = pm.actionCounter[action]
		if !exists {
			counter = &atomic.Int64{}
			pm.actionCounter[action] = counter
		}
		pm.actionMutex.Unlock()
	}
	
	counter.Add(1)
}

func (pm *PoolMetrics) calculateSessionUtilization() float64 {
	total := pm.totalSessions.Load()
	active := pm.activeSessions.Load()
	
	if total == 0 {
		return 0.0
	}
	
	return float64(active) / float64(total)
}

// LatencyTracker 메서드들

func (lt *LatencyTracker) AddSample(latency time.Duration) {
	lt.mutex.Lock()
	defer lt.mutex.Unlock()
	
	lt.samples = append(lt.samples, latency)
	
	// 최대 샘플 수 제한
	if len(lt.samples) > lt.maxSamples {
		lt.samples = lt.samples[1:]
	}
}

func (lt *LatencyTracker) GetAverage() time.Duration {
	lt.mutex.RLock()
	defer lt.mutex.RUnlock()
	
	if len(lt.samples) == 0 {
		return 0
	}
	
	var total time.Duration
	for _, sample := range lt.samples {
		total += sample
	}
	
	return total / time.Duration(len(lt.samples))
}

func (lt *LatencyTracker) GetPercentile(percentile float64) time.Duration {
	lt.mutex.RLock()
	defer lt.mutex.RUnlock()
	
	if len(lt.samples) == 0 {
		return 0
	}
	
	// 복사본 생성 및 정렬
	samples := make([]time.Duration, len(lt.samples))
	copy(samples, lt.samples)
	sort.Slice(samples, func(i, j int) bool {
		return samples[i] < samples[j]
	})
	
	index := int(math.Ceil(percentile/100.0*float64(len(samples)))) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(samples) {
		index = len(samples) - 1
	}
	
	return samples[index]
}

func (lt *LatencyTracker) GetMin() time.Duration {
	lt.mutex.RLock()
	defer lt.mutex.RUnlock()
	
	if len(lt.samples) == 0 {
		return 0
	}
	
	min := lt.samples[0]
	for _, sample := range lt.samples[1:] {
		if sample < min {
			min = sample
		}
	}
	
	return min
}

func (lt *LatencyTracker) GetMax() time.Duration {
	lt.mutex.RLock()
	defer lt.mutex.RUnlock()
	
	if len(lt.samples) == 0 {
		return 0
	}
	
	max := lt.samples[0]
	for _, sample := range lt.samples[1:] {
		if sample > max {
			max = sample
		}
	}
	
	return max
}

// ThroughputTracker 메서드들

func (tt *ThroughputTracker) RecordRequest() {
	tt.mutex.Lock()
	defer tt.mutex.Unlock()
	
	now := time.Now()
	
	// 현재 윈도우 찾기 또는 생성
	var currentWindow *ThroughputWindow
	if len(tt.windows) > 0 {
		lastWindow := &tt.windows[len(tt.windows)-1]
		if now.Sub(lastWindow.StartTime) < tt.windowSize {
			currentWindow = lastWindow
		}
	}
	
	if currentWindow == nil {
		// 새 윈도우 생성
		tt.windows = append(tt.windows, ThroughputWindow{
			StartTime: now,
			EndTime:   now.Add(tt.windowSize),
			Requests:  0,
		})
		currentWindow = &tt.windows[len(tt.windows)-1]
		
		// 최대 윈도우 수 제한
		if len(tt.windows) > tt.maxWindows {
			tt.windows = tt.windows[1:]
		}
	}
	
	currentWindow.Requests++
	
	// RPS 계산
	elapsed := time.Since(currentWindow.StartTime)
	if elapsed > 0 {
		currentWindow.RPS = float64(currentWindow.Requests) / elapsed.Seconds()
	}
}

func (tt *ThroughputTracker) GetCurrentRPS() float64 {
	tt.mutex.RLock()
	defer tt.mutex.RUnlock()
	
	if len(tt.windows) == 0 {
		return 0.0
	}
	
	return tt.windows[len(tt.windows)-1].RPS
}

func (tt *ThroughputTracker) GetAverageRPS() float64 {
	tt.mutex.RLock()
	defer tt.mutex.RUnlock()
	
	if len(tt.windows) == 0 {
		return 0.0
	}
	
	var totalRPS float64
	for _, window := range tt.windows {
		totalRPS += window.RPS
	}
	
	return totalRPS / float64(len(tt.windows))
}

func (tt *ThroughputTracker) GetPeakRPS() float64 {
	tt.mutex.RLock()
	defer tt.mutex.RUnlock()
	
	if len(tt.windows) == 0 {
		return 0.0
	}
	
	peak := tt.windows[0].RPS
	for _, window := range tt.windows[1:] {
		if window.RPS > peak {
			peak = window.RPS
		}
	}
	
	return peak
}

// ErrorRateTracker 메서드들

func (et *ErrorRateTracker) RecordError() {
	et.recordRequest(true)
}

func (et *ErrorRateTracker) RecordSuccess() {
	et.recordRequest(false)
}

func (et *ErrorRateTracker) recordRequest(isError bool) {
	et.mutex.Lock()
	defer et.mutex.Unlock()
	
	now := time.Now()
	
	// 현재 윈도우 찾기 또는 생성
	var currentWindow *ErrorWindow
	if len(et.windows) > 0 {
		lastWindow := &et.windows[len(et.windows)-1]
		if now.Sub(lastWindow.StartTime) < et.windowSize {
			currentWindow = lastWindow
		}
	}
	
	if currentWindow == nil {
		// 새 윈도우 생성
		et.windows = append(et.windows, ErrorWindow{
			StartTime: now,
			EndTime:   now.Add(et.windowSize),
		})
		currentWindow = &et.windows[len(et.windows)-1]
		
		// 최대 윈도우 수 제한
		if len(et.windows) > et.maxWindows {
			et.windows = et.windows[1:]
		}
	}
	
	currentWindow.TotalRequests++
	if isError {
		currentWindow.ErrorRequests++
	}
	
	// 에러율 계산
	if currentWindow.TotalRequests > 0 {
		currentWindow.ErrorRate = float64(currentWindow.ErrorRequests) / float64(currentWindow.TotalRequests)
	}
}

func (et *ErrorRateTracker) GetCurrentErrorRate() float64 {
	et.mutex.RLock()
	defer et.mutex.RUnlock()
	
	if len(et.windows) == 0 {
		return 0.0
	}
	
	return et.windows[len(et.windows)-1].ErrorRate
}

// TimeSeriesMetrics 메서드들

func (tsm *TimeSeriesMetrics) AddDataPoint(point MetricDataPoint) {
	tsm.mutex.Lock()
	defer tsm.mutex.Unlock()
	
	tsm.dataPoints = append(tsm.dataPoints, point)
	
	// 최대 포인트 수 제한
	if len(tsm.dataPoints) > tsm.maxPoints {
		tsm.dataPoints = tsm.dataPoints[1:]
	}
}

func (tsm *TimeSeriesMetrics) GetDataPoints() []MetricDataPoint {
	tsm.mutex.RLock()
	defer tsm.mutex.RUnlock()
	
	// 복사본 반환
	points := make([]MetricDataPoint, len(tsm.dataPoints))
	copy(points, tsm.dataPoints)
	
	return points
}

func (tsm *TimeSeriesMetrics) GetDataPointsSince(since time.Time) []MetricDataPoint {
	tsm.mutex.RLock()
	defer tsm.mutex.RUnlock()
	
	var result []MetricDataPoint
	for _, point := range tsm.dataPoints {
		if point.Timestamp.After(since) {
			result = append(result, point)
		}
	}
	
	return result
}