package monitoring

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Monitor 통합 모니터링 시스템
type Monitor struct {
	queryMonitor *QueryMonitor
	analyzer     *QueryAnalyzer
	logger       *zap.Logger
	mu           sync.RWMutex
	enabled      bool
}

// IntegratedMonitorConfig 통합 모니터링 설정
type IntegratedMonitorConfig struct {
	QueryMonitorConfig MonitorConfig
	AnalyzerEnabled    bool
	Logger            *zap.Logger
}

// DefaultIntegratedMonitorConfig 기본 통합 모니터링 설정
func DefaultIntegratedMonitorConfig() IntegratedMonitorConfig {
	return IntegratedMonitorConfig{
		QueryMonitorConfig: DefaultMonitorConfig(),
		AnalyzerEnabled:    true,
		Logger:            zap.NewNop(),
	}
}

// NewMonitor 새 모니터링 시스템 생성
func NewMonitor(config IntegratedMonitorConfig) *Monitor {
	return &Monitor{
		queryMonitor: NewQueryMonitor(config.QueryMonitorConfig),
		analyzer:     NewQueryAnalyzer(config.Logger),
		logger:       config.Logger,
		enabled:      config.QueryMonitorConfig.EnableLogging,
	}
}

// IsEnabled 모니터링 활성화 여부 반환
func (m *Monitor) IsEnabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.enabled
}

// Enable 모니터링 활성화
func (m *Monitor) Enable() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enabled = true
	
	m.logger.Info("모니터링 시스템이 활성화되었습니다")
}

// Disable 모니터링 비활성화
func (m *Monitor) Disable() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enabled = false
	
	m.logger.Info("모니터링 시스템이 비활성화되었습니다")
}

// WrapQuery 쿼리 실행 모니터링
func (m *Monitor) WrapQuery(ctx context.Context, opts WrapOptions, fn func() error) error {
	if !m.IsEnabled() || m.queryMonitor == nil {
		return fn()
	}
	
	return m.queryMonitor.Wrap(ctx, opts, fn)
}

// AnalyzeQuery 쿼리 분석
func (m *Monitor) AnalyzeQuery(ctx context.Context, db *sql.DB, query string) (*QueryAnalysis, error) {
	if !m.IsEnabled() || m.analyzer == nil {
		return nil, ErrMonitoringDisabled
	}
	
	return m.analyzer.AnalyzeSQLiteQuery(ctx, db, query)
}

// BenchmarkQuery 쿼리 벤치마크
func (m *Monitor) BenchmarkQuery(ctx context.Context, db *sql.DB, query string, iterations int) (*BenchmarkResult, error) {
	if !m.IsEnabled() || m.analyzer == nil {
		return nil, ErrMonitoringDisabled
	}
	
	return m.analyzer.BenchmarkQuery(ctx, db, query, iterations)
}

// GetSlowQueries 느린 쿼리 목록 반환
func (m *Monitor) GetSlowQueries() ([]SlowQuery, error) {
	if !m.IsEnabled() || m.queryMonitor == nil {
		return nil, ErrMonitoringDisabled
	}
	
	return m.queryMonitor.GetSlowQueries(), nil
}

// ClearSlowQueries 느린 쿼리 목록 초기화
func (m *Monitor) ClearSlowQueries() error {
	if !m.IsEnabled() || m.queryMonitor == nil {
		return ErrMonitoringDisabled
	}
	
	m.queryMonitor.ClearSlowQueries()
	return nil
}

// GetStats 통계 정보 반환
func (m *Monitor) GetStats() (Stats, error) {
	if !m.IsEnabled() || m.queryMonitor == nil {
		return Stats{}, ErrMonitoringDisabled
	}
	
	queryStats := m.queryMonitor.GetStats()
	return Stats{
		TotalQueries:   int64(queryStats.TotalQueries),
		SlowQueries:    int64(queryStats.SlowQueries),
		FailedQueries:  int64(queryStats.ErrorQueries),
		AverageDuration: queryStats.AverageDuration,
	}, nil
}

// UpdateSlowThreshold 느린 쿼리 임계값 업데이트
func (m *Monitor) UpdateSlowThreshold(threshold time.Duration) error {
	if !m.IsEnabled() || m.queryMonitor == nil {
		return ErrMonitoringDisabled
	}
	
	m.queryMonitor.UpdateSlowThreshold(threshold)
	return nil
}

// MonitoredExecutor 모니터링이 포함된 실행기
type MonitoredExecutor struct {
	monitor *Monitor
	db      *sql.DB
	storage string // sqlite, boltdb
}

// NewMonitoredExecutor 모니터링 실행기 생성
func NewMonitoredExecutor(monitor *Monitor, db *sql.DB, storageType string) *MonitoredExecutor {
	return &MonitoredExecutor{
		monitor: monitor,
		db:      db,
		storage: storageType,
	}
}

// ExecContext 모니터링과 함께 쿼리 실행
func (e *MonitoredExecutor) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	opts := WrapOptions{
		Query:       query,
		Table:       getQueryType(query),
		StorageType: e.storage,
		Operation:   "exec",
		Context:     QueryContext{StorageType: e.storage},
	}
	
	var result sql.Result
	var err error
	
	execErr := e.monitor.WrapQuery(ctx, opts, func() error {
		result, err = e.db.ExecContext(ctx, query, args...)
		return err
	})
	
	if execErr != nil {
		return nil, execErr
	}
	
	return result, err
}

// QueryContext 모니터링과 함께 쿼리 실행
func (e *MonitoredExecutor) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	opts := WrapOptions{
		Query:       query,
		Table:       getQueryType(query),
		StorageType: e.storage,
		Operation:   "query",
		Context:     QueryContext{StorageType: e.storage},
	}
	
	var rows *sql.Rows
	var err error
	
	execErr := e.monitor.WrapQuery(ctx, opts, func() error {
		rows, err = e.db.QueryContext(ctx, query, args...)
		return err
	})
	
	if execErr != nil {
		return nil, execErr
	}
	
	return rows, err
}

// QueryRowContext 모니터링과 함께 단일 행 쿼리 실행
func (e *MonitoredExecutor) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	opts := WrapOptions{
		Query:       query,
		Table:       getQueryType(query),
		StorageType: e.storage,
		Operation:   "query_row",
		Context:     QueryContext{StorageType: e.storage},
	}
	
	var row *sql.Row
	
	// QueryRowContext는 항상 row를 반환하므로 에러를 무시
	_ = e.monitor.WrapQuery(ctx, opts, func() error {
		row = e.db.QueryRowContext(ctx, query, args...)
		return nil
	})
	
	return row
}

// 유틸리티 함수들

const (
	// 쿼리 타입 상수
	QueryTypeSelect = "SELECT"
	QueryTypeInsert = "INSERT"
	QueryTypeUpdate = "UPDATE"
	QueryTypeDelete = "DELETE"
)

// getQueryType 쿼리 타입 추출
func getQueryType(query string) string {
	query = strings.TrimSpace(strings.ToLower(query))
	
	switch {
	case strings.HasPrefix(query, "select"):
		return QueryTypeSelect
	case strings.HasPrefix(query, "insert"):
		return QueryTypeInsert
	case strings.HasPrefix(query, "update"):
		return QueryTypeUpdate
	case strings.HasPrefix(query, "delete"):
		return QueryTypeDelete
	default:
		return "other"
	}
}

// ErrMonitoringDisabled 모니터링 비활성화 에러
var ErrMonitoringDisabled = fmt.Errorf("모니터링이 비활성화되어 있습니다")