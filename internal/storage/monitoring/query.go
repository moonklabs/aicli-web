package monitoring

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// QueryType 쿼리 유형 상수
const (
	QueryTypeSelect = "select"
	QueryTypeInsert = "insert"
	QueryTypeUpdate = "update"
	QueryTypeDelete = "delete"
)

// StorageType 스토리지 유형 상수
const (
	StorageTypeSQLite = "sqlite"
	StorageTypeBoltDB = "boltdb"
	StorageTypeMemory = "memory"
)

// QueryMonitor 쿼리 모니터링 시스템
type QueryMonitor struct {
	logger           *zap.Logger
	slowThreshold    time.Duration
	metricsCollector *MetricsCollector
	mu               sync.RWMutex
	slowQueries      []SlowQuery
	maxSlowQueries   int
}

// SlowQuery 느린 쿼리 정보
type SlowQuery struct {
	Query       string        `json:"query"`
	QueryType   string        `json:"query_type"`
	StorageType string        `json:"storage_type"`
	Duration    time.Duration `json:"duration"`
	Timestamp   time.Time     `json:"timestamp"`
	Context     string        `json:"context"`
	Error       string        `json:"error,omitempty"`
}

// MetricsCollector Prometheus 메트릭 수집기
type MetricsCollector struct {
	queryDuration  *prometheus.HistogramVec
	queryCount     *prometheus.CounterVec
	slowQueryCount *prometheus.CounterVec
	errorCount     *prometheus.CounterVec
}

// Config 쿼리 모니터 설정
type Config struct {
	SlowThreshold    time.Duration
	MaxSlowQueries   int
	MetricsEnabled   bool
	Logger           *zap.Logger
}

// DefaultConfig 기본 설정 반환
func DefaultConfig() Config {
	return Config{
		SlowThreshold:  100 * time.Millisecond,
		MaxSlowQueries: 1000,
		MetricsEnabled: true,
		Logger:         zap.NewNop(),
	}
}

// NewQueryMonitor 새 쿼리 모니터 생성
func NewQueryMonitor(config Config) *QueryMonitor {
	monitor := &QueryMonitor{
		logger:         config.Logger,
		slowThreshold:  config.SlowThreshold,
		slowQueries:    make([]SlowQuery, 0, config.MaxSlowQueries),
		maxSlowQueries: config.MaxSlowQueries,
	}

	if config.MetricsEnabled {
		monitor.metricsCollector = NewMetricsCollector()
	}

	return monitor
}

// NewMetricsCollector Prometheus 메트릭 수집기 생성
func NewMetricsCollector() *MetricsCollector {
	collector := &MetricsCollector{
		queryDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "aicli_storage_query_duration_seconds",
				Help: "Storage query execution duration in seconds",
				Buckets: []float64{
					0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0,
				},
			},
			[]string{"query_type", "storage_type", "operation"},
		),
		queryCount: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "aicli_storage_query_total",
				Help: "Total number of storage queries",
			},
			[]string{"query_type", "storage_type", "operation", "status"},
		),
		slowQueryCount: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "aicli_storage_slow_query_total",
				Help: "Total number of slow storage queries",
			},
			[]string{"query_type", "storage_type", "operation"},
		),
		errorCount: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "aicli_storage_query_error_total",
				Help: "Total number of storage query errors",
			},
			[]string{"query_type", "storage_type", "operation", "error_type"},
		),
	}

	// 메트릭 등록
	prometheus.MustRegister(
		collector.queryDuration,
		collector.queryCount,
		collector.slowQueryCount,
		collector.errorCount,
	)

	return collector
}

// Wrap 쿼리 실행을 모니터링으로 래핑
func (m *QueryMonitor) Wrap(ctx context.Context, opts WrapOptions, fn func() error) error {
	start := time.Now()
	
	// 쿼리 실행
	err := fn()
	
	duration := time.Since(start)
	
	// 메트릭 수집
	if m.metricsCollector != nil {
		m.collectMetrics(opts, duration, err)
	}
	
	// 느린 쿼리 감지
	if duration > m.slowThreshold {
		m.recordSlowQuery(SlowQuery{
			Query:       opts.Query,
			QueryType:   opts.QueryType,
			StorageType: opts.StorageType,
			Duration:    duration,
			Timestamp:   start,
			Context:     opts.Context,
			Error:       errorToString(err),
		})
		
		// 경고 로그
		m.logger.Warn("느린 쿼리 감지됨",
			zap.String("query", opts.Query),
			zap.String("query_type", opts.QueryType),
			zap.String("storage_type", opts.StorageType),
			zap.Duration("duration", duration),
			zap.Duration("threshold", m.slowThreshold),
			zap.String("context", opts.Context),
		)
	}
	
	// 일반 로그 (디버그 레벨)
	m.logger.Debug("쿼리 실행 완료",
		zap.String("query", opts.Query),
		zap.String("query_type", opts.QueryType),
		zap.String("storage_type", opts.StorageType),
		zap.Duration("duration", duration),
		zap.String("context", opts.Context),
		zap.Error(err),
	)
	
	return err
}

// WrapOptions 래핑 옵션
type WrapOptions struct {
	Query       string
	QueryType   string
	StorageType string
	Operation   string // workspace.create, project.list 등
	Context     string
}

// collectMetrics 메트릭 수집
func (m *QueryMonitor) collectMetrics(opts WrapOptions, duration time.Duration, err error) {
	labels := []string{opts.QueryType, opts.StorageType, opts.Operation}
	
	// 실행 시간 기록
	m.metricsCollector.queryDuration.WithLabelValues(labels...).Observe(duration.Seconds())
	
	// 쿼리 수 증가
	status := "success"
	if err != nil {
		status = "error"
	}
	statusLabels := append(labels, status)
	m.metricsCollector.queryCount.WithLabelValues(statusLabels...).Inc()
	
	// 느린 쿼리 수 증가
	if duration > m.slowThreshold {
		m.metricsCollector.slowQueryCount.WithLabelValues(labels...).Inc()
	}
	
	// 에러 수 증가
	if err != nil {
		errorType := getErrorType(err)
		errorLabels := append(labels, errorType)
		m.metricsCollector.errorCount.WithLabelValues(errorLabels...).Inc()
	}
}

// recordSlowQuery 느린 쿼리 기록
func (m *QueryMonitor) recordSlowQuery(query SlowQuery) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// 최대 개수 초과 시 오래된 것 제거
	if len(m.slowQueries) >= m.maxSlowQueries {
		m.slowQueries = m.slowQueries[1:]
	}
	
	m.slowQueries = append(m.slowQueries, query)
}

// GetSlowQueries 느린 쿼리 목록 반환
func (m *QueryMonitor) GetSlowQueries() []SlowQuery {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// 복사본 반환
	queries := make([]SlowQuery, len(m.slowQueries))
	copy(queries, m.slowQueries)
	
	return queries
}

// ClearSlowQueries 느린 쿼리 목록 초기화
func (m *QueryMonitor) ClearSlowQueries() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.slowQueries = make([]SlowQuery, 0, m.maxSlowQueries)
}

// GetStats 통계 정보 반환
func (m *QueryMonitor) GetStats() Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return Stats{
		SlowQueryCount:   len(m.slowQueries),
		MaxSlowQueries:   m.maxSlowQueries,
		SlowThreshold:    m.slowThreshold,
		MetricsEnabled:   m.metricsCollector != nil,
	}
}

// Stats 통계 정보
type Stats struct {
	SlowQueryCount int           `json:"slow_query_count"`
	MaxSlowQueries int           `json:"max_slow_queries"`
	SlowThreshold  time.Duration `json:"slow_threshold"`
	MetricsEnabled bool          `json:"metrics_enabled"`
}

// UpdateSlowThreshold 느린 쿼리 임계값 업데이트
func (m *QueryMonitor) UpdateSlowThreshold(threshold time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.slowThreshold = threshold
	
	m.logger.Info("느린 쿼리 임계값 업데이트됨",
		zap.Duration("new_threshold", threshold),
	)
}

// 유틸리티 함수들
func errorToString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func getErrorType(err error) string {
	if err == nil {
		return "none"
	}
	
	// 에러 타입별 분류
	errStr := err.Error()
	
	switch {
	case contains(errStr, "timeout"):
		return "timeout"
	case contains(errStr, "connection"):
		return "connection"
	case contains(errStr, "syntax"):
		return "syntax"
	case contains(errStr, "constraint"):
		return "constraint"
	case contains(errStr, "not found"):
		return "not_found"
	default:
		return "unknown"
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || 
		(len(s) > len(substr) && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}