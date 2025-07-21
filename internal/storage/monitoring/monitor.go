package monitoring

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
)

// QueryMonitor 쿼리 모니터링 시스템
type QueryMonitor struct {
	slowQueryThreshold time.Duration
	logger            *logrus.Logger
	mu                sync.RWMutex
	metrics           *QueryMetrics
	queryLog          []QueryLog
	maxLogSize        int
	enabled           bool
}

// QueryMetrics Prometheus 메트릭
type QueryMetrics struct {
	QueryDuration *prometheus.HistogramVec
	QueryCount    *prometheus.CounterVec
	SlowQueries   *prometheus.CounterVec
	ActiveQueries *prometheus.GaugeVec
}

// QueryLog 쿼리 로그 엔트리
type QueryLog struct {
	ID          string        `json:"id"`
	Operation   string        `json:"operation"`
	Table       string        `json:"table"`
	Query       string        `json:"query"`
	Duration    time.Duration `json:"duration"`
	Error       string        `json:"error,omitempty"`
	Timestamp   time.Time     `json:"timestamp"`
	Context     QueryContext  `json:"context"`
	IsSlow      bool          `json:"is_slow"`
}

// QueryContext 쿼리 실행 컨텍스트
type QueryContext struct {
	UserID      string            `json:"user_id,omitempty"`
	SessionID   string            `json:"session_id,omitempty"`
	RequestID   string            `json:"request_id,omitempty"`
	StorageType string            `json:"storage_type"`
	Parameters  map[string]string `json:"parameters,omitempty"`
}

// MonitorConfig 모니터링 설정
type MonitorConfig struct {
	SlowQueryThreshold time.Duration
	MaxLogSize         int
	EnablePrometheus   bool
	EnableLogging      bool
	LogLevel           logrus.Level
}

// DefaultMonitorConfig 기본 모니터링 설정
func DefaultMonitorConfig() MonitorConfig {
	return MonitorConfig{
		SlowQueryThreshold: 100 * time.Millisecond,
		MaxLogSize:         1000,
		EnablePrometheus:   true,
		EnableLogging:      true,
		LogLevel:           logrus.InfoLevel,
	}
}

// NewQueryMonitor 새 쿼리 모니터 생성
func NewQueryMonitor(config MonitorConfig) *QueryMonitor {
	logger := logrus.New()
	logger.SetLevel(config.LogLevel)

	var metrics *QueryMetrics
	if config.EnablePrometheus {
		metrics = &QueryMetrics{
			QueryDuration: promauto.NewHistogramVec(
				prometheus.HistogramOpts{
					Name: "storage_query_duration_seconds",
					Help: "The duration of storage queries in seconds",
					Buckets: []float64{
						0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0,
					},
				},
				[]string{"operation", "table", "storage_type", "status"},
			),
			QueryCount: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Name: "storage_query_total",
					Help: "The total number of storage queries",
				},
				[]string{"operation", "table", "storage_type", "status"},
			),
			SlowQueries: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Name: "storage_slow_queries_total",
					Help: "The total number of slow storage queries",
				},
				[]string{"operation", "table", "storage_type"},
			),
			ActiveQueries: promauto.NewGaugeVec(
				prometheus.GaugeOpts{
					Name: "storage_active_queries",
					Help: "The number of currently active storage queries",
				},
				[]string{"storage_type"},
			),
		}
	}

	return &QueryMonitor{
		slowQueryThreshold: config.SlowQueryThreshold,
		logger:            logger,
		metrics:           metrics,
		queryLog:          make([]QueryLog, 0, config.MaxLogSize),
		maxLogSize:        config.MaxLogSize,
		enabled:           config.EnableLogging,
	}
}

// QueryTracker 쿼리 추적기
type QueryTracker struct {
	monitor   *QueryMonitor
	startTime time.Time
	queryLog  QueryLog
}

// StartQuery 쿼리 추적 시작
func (qm *QueryMonitor) StartQuery(ctx context.Context, operation, table, query string, queryCtx QueryContext) *QueryTracker {
	if !qm.enabled {
		return nil
	}

	tracker := &QueryTracker{
		monitor:   qm,
		startTime: time.Now(),
		queryLog: QueryLog{
			ID:        generateQueryID(),
			Operation: operation,
			Table:     table,
			Query:     query,
			Timestamp: time.Now(),
			Context:   queryCtx,
		},
	}

	// 활성 쿼리 수 증가
	if qm.metrics != nil {
		qm.metrics.ActiveQueries.WithLabelValues(queryCtx.StorageType).Inc()
	}

	return tracker
}

// Finish 쿼리 추적 완료
func (qt *QueryTracker) Finish(err error) {
	if qt == nil || qt.monitor == nil {
		return
	}

	duration := time.Since(qt.startTime)
	qt.queryLog.Duration = duration

	status := "success"
	if err != nil {
		status = "error"
		qt.queryLog.Error = err.Error()
	}

	// Slow 쿼리 체크
	if duration >= qt.monitor.slowQueryThreshold {
		qt.queryLog.IsSlow = true
	}

	// 메트릭 업데이트
	if qt.monitor.metrics != nil {
		storageType := qt.queryLog.Context.StorageType
		qt.monitor.metrics.QueryDuration.WithLabelValues(
			qt.queryLog.Operation,
			qt.queryLog.Table,
			storageType,
			status,
		).Observe(duration.Seconds())

		qt.monitor.metrics.QueryCount.WithLabelValues(
			qt.queryLog.Operation,
			qt.queryLog.Table,
			storageType,
			status,
		).Inc()

		if qt.queryLog.IsSlow {
			qt.monitor.metrics.SlowQueries.WithLabelValues(
				qt.queryLog.Operation,
				qt.queryLog.Table,
				storageType,
			).Inc()
		}

		qt.monitor.metrics.ActiveQueries.WithLabelValues(storageType).Dec()
	}

	// 로그 기록
	qt.monitor.logQuery(qt.queryLog)
}

// logQuery 쿼리 로그 기록
func (qm *QueryMonitor) logQuery(queryLog QueryLog) {
	// 슬로우 쿼리 특별 처리
	if queryLog.IsSlow {
		qm.logger.WithFields(logrus.Fields{
			"query_id":     queryLog.ID,
			"operation":    queryLog.Operation,
			"table":        queryLog.Table,
			"duration_ms":  queryLog.Duration.Milliseconds(),
			"storage_type": queryLog.Context.StorageType,
			"user_id":      queryLog.Context.UserID,
			"session_id":   queryLog.Context.SessionID,
		}).Warn("Slow query detected")

		// 슬로우 쿼리 알림 (필요시 확장)
		qm.alertSlowQuery(queryLog)
	}

	// 에러 쿼리 로깅
	if queryLog.Error != "" {
		qm.logger.WithFields(logrus.Fields{
			"query_id":     queryLog.ID,
			"operation":    queryLog.Operation,
			"table":        queryLog.Table,
			"error":        queryLog.Error,
			"storage_type": queryLog.Context.StorageType,
		}).Error("Query failed")
	}

	// 일반 쿼리 로깅 (Debug 레벨)
	qm.logger.WithFields(logrus.Fields{
		"query_id":     queryLog.ID,
		"operation":    queryLog.Operation,
		"table":        queryLog.Table,
		"duration_ms":  queryLog.Duration.Milliseconds(),
		"storage_type": queryLog.Context.StorageType,
	}).Debug("Query executed")

	// 메모리에 로그 저장 (순환 버퍼)
	qm.mu.Lock()
	if len(qm.queryLog) >= qm.maxLogSize {
		// 가장 오래된 로그 제거
		qm.queryLog = qm.queryLog[1:]
	}
	qm.queryLog = append(qm.queryLog, queryLog)
	qm.mu.Unlock()
}

// alertSlowQuery 슬로우 쿼리 알림
func (qm *QueryMonitor) alertSlowQuery(queryLog QueryLog) {
	// 여기에 슬로우 쿼리 알림 로직 추가 (이메일, Slack, PagerDuty 등)
	// 현재는 로그만 남김
	log.Printf("SLOW QUERY ALERT: %s on %s took %v", 
		queryLog.Operation, queryLog.Table, queryLog.Duration)
}

// GetStats 통계 조회
func (qm *QueryMonitor) GetStats() QueryStats {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	stats := QueryStats{
		TotalQueries: len(qm.queryLog),
		SlowQueries:  0,
		ErrorQueries: 0,
	}

	var totalDuration time.Duration
	for _, log := range qm.queryLog {
		if log.IsSlow {
			stats.SlowQueries++
		}
		if log.Error != "" {
			stats.ErrorQueries++
		}
		totalDuration += log.Duration
	}

	if stats.TotalQueries > 0 {
		stats.AverageDuration = totalDuration / time.Duration(stats.TotalQueries)
	}

	return stats
}

// GetQueryLogs 쿼리 로그 조회
func (qm *QueryMonitor) GetQueryLogs(limit int, onlySlow bool) []QueryLog {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	var result []QueryLog
	count := 0

	// 최신 로그부터 역순 조회
	for i := len(qm.queryLog) - 1; i >= 0 && count < limit; i-- {
		log := qm.queryLog[i]
		if onlySlow && !log.IsSlow {
			continue
		}
		result = append(result, log)
		count++
	}

	return result
}

// QueryStats 쿼리 통계
type QueryStats struct {
	TotalQueries    int           `json:"total_queries"`
	SlowQueries     int           `json:"slow_queries"`
	ErrorQueries    int           `json:"error_queries"`
	AverageDuration time.Duration `json:"average_duration"`
}

// generateQueryID 쿼리 ID 생성
func generateQueryID() string {
	return fmt.Sprintf("q_%d_%d", 
		time.Now().Unix(), 
		time.Now().Nanosecond()%1000000)
}

// SetSlowQueryThreshold Slow 쿼리 임계값 설정
func (qm *QueryMonitor) SetSlowQueryThreshold(threshold time.Duration) {
	qm.mu.Lock()
	defer qm.mu.Unlock()
	qm.slowQueryThreshold = threshold
}

// Enable/Disable 모니터링 활성화/비활성화
func (qm *QueryMonitor) Enable() {
	qm.mu.Lock()
	defer qm.mu.Unlock()
	qm.enabled = true
}

func (qm *QueryMonitor) Disable() {
	qm.mu.Lock()
	defer qm.mu.Unlock()
	qm.enabled = false
}

// IsEnabled 모니터링 활성화 상태 확인
func (qm *QueryMonitor) IsEnabled() bool {
	qm.mu.RLock()
	defer qm.mu.RUnlock()
	return qm.enabled
}