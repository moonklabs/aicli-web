package middleware

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// PerformanceMetrics는 성능 메트릭을 저장합니다
type PerformanceMetrics struct {
	// HTTP 메트릭
	RequestCount     int64             `json:"request_count"`
	ErrorCount       int64             `json:"error_count"`
	RequestDuration  time.Duration     `json:"request_duration"`
	ActiveRequests   int64             `json:"active_requests"`
	ResponseCodes    map[string]int64  `json:"response_codes"`
	EndpointMetrics  map[string]*EndpointMetric `json:"endpoint_metrics"`

	// 시스템 메트릭
	CPUUsage         float64           `json:"cpu_usage"`
	MemoryUsage      uint64            `json:"memory_usage"`
	GoroutineCount   int               `json:"goroutine_count"`
	HeapSize         uint64            `json:"heap_size"`
	GCCount          uint32            `json:"gc_count"`

	// 메타데이터
	Timestamp        time.Time         `json:"timestamp"`
	Uptime           time.Duration     `json:"uptime"`
}

// EndpointMetric은 개별 엔드포인트 메트릭입니다
type EndpointMetric struct {
	Method           string            `json:"method"`
	Path             string            `json:"path"`
	Count            int64             `json:"count"`
	ErrorCount       int64             `json:"error_count"`
	TotalDuration    time.Duration     `json:"total_duration"`
	AverageDuration  time.Duration     `json:"average_duration"`
	MinDuration      time.Duration     `json:"min_duration"`
	MaxDuration      time.Duration     `json:"max_duration"`
	LastAccess       time.Time         `json:"last_access"`
	StatusCodes      map[int]int64     `json:"status_codes"`
}

// PerformanceMonitoringMiddleware는 성능 모니터링 미들웨어입니다
type PerformanceMonitoringMiddleware struct {
	metrics         *PerformanceMetrics
	mutex           sync.RWMutex
	startTime       time.Time
	collectionTicker *time.Ticker
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup

	// 설정
	config          MonitoringConfig

	// 이벤트 리스너들
	listeners       []PerformanceListener
	listenersMutex  sync.RWMutex
}

// MonitoringConfig는 모니터링 설정입니다
type MonitoringConfig struct {
	CollectionInterval time.Duration `json:"collection_interval"`
	RetentionPeriod    time.Duration `json:"retention_period"`
	EnableSystemMetrics bool         `json:"enable_system_metrics"`
	EnableEndpointMetrics bool       `json:"enable_endpoint_metrics"`
	MaxEndpoints       int           `json:"max_endpoints"`
	SlowRequestThreshold time.Duration `json:"slow_request_threshold"`
}

// PerformanceListener는 성능 이벤트 리스너입니다
type PerformanceListener interface {
	OnMetricsUpdate(metrics *PerformanceMetrics)
	OnSlowRequest(method, path string, duration time.Duration)
	OnError(method, path string, statusCode int, err error)
}

// DefaultMonitoringConfig는 기본 모니터링 설정을 반환합니다
func DefaultMonitoringConfig() MonitoringConfig {
	return MonitoringConfig{
		CollectionInterval:     10 * time.Second,
		RetentionPeriod:        24 * time.Hour,
		EnableSystemMetrics:    true,
		EnableEndpointMetrics:  true,
		MaxEndpoints:           100,
		SlowRequestThreshold:   2 * time.Second,
	}
}

// NewPerformanceMonitoringMiddleware는 새로운 성능 모니터링 미들웨어를 생성합니다
func NewPerformanceMonitoringMiddleware(config MonitoringConfig) *PerformanceMonitoringMiddleware {
	ctx, cancel := context.WithCancel(context.Background())

	middleware := &PerformanceMonitoringMiddleware{
		metrics: &PerformanceMetrics{
			ResponseCodes:   make(map[string]int64),
			EndpointMetrics: make(map[string]*EndpointMetric),
			Timestamp:       time.Now(),
		},
		startTime:        time.Now(),
		config:          config,
		ctx:             ctx,
		cancel:          cancel,
		listeners:       make([]PerformanceListener, 0),
	}

	// 시스템 메트릭 수집 시작
	if config.EnableSystemMetrics {
		middleware.startSystemMetricsCollection()
	}

	return middleware
}

// Handler는 Gin 미들웨어 핸들러를 반환합니다
func (pmm *PerformanceMonitoringMiddleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// 활성 요청 수 증가
		pmm.incrementActiveRequests()
		defer pmm.decrementActiveRequests()

		// 요청 처리
		c.Next()

		// 메트릭 업데이트
		duration := time.Since(start)
		method := c.Request.Method
		path := c.FullPath()
		statusCode := c.Writer.Status()

		pmm.updateMetrics(method, path, duration, statusCode)

		// 느린 요청 감지
		if duration > pmm.config.SlowRequestThreshold {
			pmm.notifySlowRequest(method, path, duration)
		}

		// 에러 감지
		if statusCode >= 400 {
			if err := c.Errors.Last(); err != nil {
				pmm.notifyError(method, path, statusCode, err)
			}
		}
	}
}

// GetMetrics는 현재 메트릭을 반환합니다
func (pmm *PerformanceMonitoringMiddleware) GetMetrics() *PerformanceMetrics {
	pmm.mutex.RLock()
	defer pmm.mutex.RUnlock()

	// 복사본 생성
	metrics := &PerformanceMetrics{
		RequestCount:    pmm.metrics.RequestCount,
		ErrorCount:      pmm.metrics.ErrorCount,
		RequestDuration: pmm.metrics.RequestDuration,
		ActiveRequests:  pmm.metrics.ActiveRequests,
		ResponseCodes:   make(map[string]int64),
		EndpointMetrics: make(map[string]*EndpointMetric),
		CPUUsage:        pmm.metrics.CPUUsage,
		MemoryUsage:     pmm.metrics.MemoryUsage,
		GoroutineCount:  pmm.metrics.GoroutineCount,
		HeapSize:        pmm.metrics.HeapSize,
		GCCount:         pmm.metrics.GCCount,
		Timestamp:       time.Now(),
		Uptime:          time.Since(pmm.startTime),
	}

	// ResponseCodes 복사
	for k, v := range pmm.metrics.ResponseCodes {
		metrics.ResponseCodes[k] = v
	}

	// EndpointMetrics 복사
	for k, v := range pmm.metrics.EndpointMetrics {
		metrics.EndpointMetrics[k] = &EndpointMetric{
			Method:          v.Method,
			Path:            v.Path,
			Count:           v.Count,
			ErrorCount:      v.ErrorCount,
			TotalDuration:   v.TotalDuration,
			AverageDuration: v.AverageDuration,
			MinDuration:     v.MinDuration,
			MaxDuration:     v.MaxDuration,
			LastAccess:      v.LastAccess,
			StatusCodes:     make(map[int]int64),
		}
		for code, count := range v.StatusCodes {
			metrics.EndpointMetrics[k].StatusCodes[code] = count
		}
	}

	return metrics
}

// AddListener는 성능 이벤트 리스너를 추가합니다
func (pmm *PerformanceMonitoringMiddleware) AddListener(listener PerformanceListener) {
	pmm.listenersMutex.Lock()
	defer pmm.listenersMutex.Unlock()

	pmm.listeners = append(pmm.listeners, listener)
}

// RemoveListener는 성능 이벤트 리스너를 제거합니다
func (pmm *PerformanceMonitoringMiddleware) RemoveListener(listener PerformanceListener) {
	pmm.listenersMutex.Lock()
	defer pmm.listenersMutex.Unlock()

	for i, l := range pmm.listeners {
		if l == listener {
			pmm.listeners = append(pmm.listeners[:i], pmm.listeners[i+1:]...)
			break
		}
	}
}

// Stop은 모니터링을 중지합니다
func (pmm *PerformanceMonitoringMiddleware) Stop() {
	pmm.cancel()
	if pmm.collectionTicker != nil {
		pmm.collectionTicker.Stop()
	}
	pmm.wg.Wait()
}

// 내부 메서드들

func (pmm *PerformanceMonitoringMiddleware) incrementActiveRequests() {
	pmm.mutex.Lock()
	defer pmm.mutex.Unlock()
	pmm.metrics.ActiveRequests++
}

func (pmm *PerformanceMonitoringMiddleware) decrementActiveRequests() {
	pmm.mutex.Lock()
	defer pmm.mutex.Unlock()
	pmm.metrics.ActiveRequests--
}

func (pmm *PerformanceMonitoringMiddleware) updateMetrics(method, path string, duration time.Duration, statusCode int) {
	pmm.mutex.Lock()
	defer pmm.mutex.Unlock()

	// 전역 메트릭 업데이트
	pmm.metrics.RequestCount++
	pmm.metrics.RequestDuration += duration

	if statusCode >= 400 {
		pmm.metrics.ErrorCount++
	}

	// 응답 코드 통계
	statusCodeStr := strconv.Itoa(statusCode)
	pmm.metrics.ResponseCodes[statusCodeStr]++

	// 엔드포인트별 메트릭 업데이트
	if pmm.config.EnableEndpointMetrics {
		pmm.updateEndpointMetrics(method, path, duration, statusCode)
	}
}

func (pmm *PerformanceMonitoringMiddleware) updateEndpointMetrics(method, path string, duration time.Duration, statusCode int) {
	key := fmt.Sprintf("%s %s", method, path)

	metric, exists := pmm.metrics.EndpointMetrics[key]
	if !exists {
		// 최대 엔드포인트 수 확인
		if len(pmm.metrics.EndpointMetrics) >= pmm.config.MaxEndpoints {
			return
		}

		metric = &EndpointMetric{
			Method:       method,
			Path:         path,
			MinDuration:  duration,
			MaxDuration:  duration,
			StatusCodes:  make(map[int]int64),
		}
		pmm.metrics.EndpointMetrics[key] = metric
	}

	// 메트릭 업데이트
	metric.Count++
	metric.TotalDuration += duration
	metric.AverageDuration = time.Duration(int64(metric.TotalDuration) / metric.Count)
	metric.LastAccess = time.Now()

	if duration < metric.MinDuration {
		metric.MinDuration = duration
	}
	if duration > metric.MaxDuration {
		metric.MaxDuration = duration
	}

	if statusCode >= 400 {
		metric.ErrorCount++
	}

	metric.StatusCodes[statusCode]++
}

func (pmm *PerformanceMonitoringMiddleware) startSystemMetricsCollection() {
	pmm.collectionTicker = time.NewTicker(pmm.config.CollectionInterval)

	pmm.wg.Add(1)
	go func() {
		defer pmm.wg.Done()

		for {
			select {
			case <-pmm.ctx.Done():
				return
			case <-pmm.collectionTicker.C:
				pmm.collectSystemMetrics()
				pmm.notifyMetricsUpdate()
			}
		}
	}()
}

func (pmm *PerformanceMonitoringMiddleware) collectSystemMetrics() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	pmm.mutex.Lock()
	defer pmm.mutex.Unlock()

	pmm.metrics.MemoryUsage = memStats.Alloc
	pmm.metrics.GoroutineCount = runtime.NumGoroutine()
	pmm.metrics.HeapSize = memStats.HeapAlloc
	pmm.metrics.GCCount = memStats.NumGC

	// CPU 사용률은 간단한 추정치 사용 (실제로는 외부 라이브러리 필요)
	// 여기서는 고루틴 수를 기반으로 추정
	pmm.metrics.CPUUsage = float64(pmm.metrics.GoroutineCount) / 100.0
	if pmm.metrics.CPUUsage > 100.0 {
		pmm.metrics.CPUUsage = 100.0
	}
}

func (pmm *PerformanceMonitoringMiddleware) notifyMetricsUpdate() {
	pmm.listenersMutex.RLock()
	listeners := make([]PerformanceListener, len(pmm.listeners))
	copy(listeners, pmm.listeners)
	pmm.listenersMutex.RUnlock()

	metrics := pmm.GetMetrics()

	for _, listener := range listeners {
		go func(l PerformanceListener) {
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("Performance listener panic: %v\n", r)
				}
			}()
			l.OnMetricsUpdate(metrics)
		}(listener)
	}
}

func (pmm *PerformanceMonitoringMiddleware) notifySlowRequest(method, path string, duration time.Duration) {
	pmm.listenersMutex.RLock()
	listeners := make([]PerformanceListener, len(pmm.listeners))
	copy(listeners, pmm.listeners)
	pmm.listenersMutex.RUnlock()

	for _, listener := range listeners {
		go func(l PerformanceListener) {
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("Performance listener panic: %v\n", r)
				}
			}()
			l.OnSlowRequest(method, path, duration)
		}(listener)
	}
}

func (pmm *PerformanceMonitoringMiddleware) notifyError(method, path string, statusCode int, err error) {
	pmm.listenersMutex.RLock()
	listeners := make([]PerformanceListener, len(pmm.listeners))
	copy(listeners, pmm.listeners)
	pmm.listenersMutex.RUnlock()

	for _, listener := range listeners {
		go func(l PerformanceListener) {
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("Performance listener panic: %v\n", r)
				}
			}()
			l.OnError(method, path, statusCode, err)
		}(listener)
	}
}

// 유틸리티 함수들

// CalculatePerformanceScore는 성능 점수를 계산합니다
func (pmm *PerformanceMonitoringMiddleware) CalculatePerformanceScore() float64 {
	metrics := pmm.GetMetrics()

	if metrics.RequestCount == 0 {
		return 100.0
	}

	score := 100.0

	// 에러율 감점 (최대 40점)
	errorRate := float64(metrics.ErrorCount) / float64(metrics.RequestCount)
	score -= errorRate * 40.0

	// 평균 응답 시간 감점 (최대 30점)
	avgDuration := time.Duration(int64(metrics.RequestDuration) / metrics.RequestCount)
	if avgDuration > time.Second {
		score -= 30.0
	} else if avgDuration > 500*time.Millisecond {
		score -= 15.0
	} else if avgDuration > 200*time.Millisecond {
		score -= 5.0
	}

	// 메모리 사용률 감점 (최대 20점)
	if metrics.MemoryUsage > 100*1024*1024 { // 100MB
		score -= 20.0
	} else if metrics.MemoryUsage > 50*1024*1024 { // 50MB
		score -= 10.0
	}

	// 고루틴 수 감점 (최대 10점)
	if metrics.GoroutineCount > 1000 {
		score -= 10.0
	} else if metrics.GoroutineCount > 500 {
		score -= 5.0
	}

	if score < 0 {
		score = 0
	}

	return score
}

// GetTopEndpoints는 가장 많이 사용되는 엔드포인트들을 반환합니다
func (pmm *PerformanceMonitoringMiddleware) GetTopEndpoints(limit int) []*EndpointMetric {
	pmm.mutex.RLock()
	defer pmm.mutex.RUnlock()

	endpoints := make([]*EndpointMetric, 0, len(pmm.metrics.EndpointMetrics))
	for _, metric := range pmm.metrics.EndpointMetrics {
		endpoints = append(endpoints, metric)
	}

	// 요청 수로 정렬
	for i := 0; i < len(endpoints)-1; i++ {
		for j := i + 1; j < len(endpoints); j++ {
			if endpoints[i].Count < endpoints[j].Count {
				endpoints[i], endpoints[j] = endpoints[j], endpoints[i]
			}
		}
	}

	if limit > 0 && len(endpoints) > limit {
		endpoints = endpoints[:limit]
	}

	return endpoints
}

// GetSlowEndpoints는 가장 느린 엔드포인트들을 반환합니다
func (pmm *PerformanceMonitoringMiddleware) GetSlowEndpoints(limit int) []*EndpointMetric {
	pmm.mutex.RLock()
	defer pmm.mutex.RUnlock()

	endpoints := make([]*EndpointMetric, 0, len(pmm.metrics.EndpointMetrics))
	for _, metric := range pmm.metrics.EndpointMetrics {
		endpoints = append(endpoints, metric)
	}

	// 평균 응답 시간으로 정렬
	for i := 0; i < len(endpoints)-1; i++ {
		for j := i + 1; j < len(endpoints); j++ {
			if endpoints[i].AverageDuration < endpoints[j].AverageDuration {
				endpoints[i], endpoints[j] = endpoints[j], endpoints[i]
			}
		}
	}

	if limit > 0 && len(endpoints) > limit {
		endpoints = endpoints[:limit]
	}

	return endpoints
}

// Reset은 모든 메트릭을 초기화합니다
func (pmm *PerformanceMonitoringMiddleware) Reset() {
	pmm.mutex.Lock()
	defer pmm.mutex.Unlock()

	pmm.metrics = &PerformanceMetrics{
		ResponseCodes:   make(map[string]int64),
		EndpointMetrics: make(map[string]*EndpointMetric),
		Timestamp:       time.Now(),
	}
}
