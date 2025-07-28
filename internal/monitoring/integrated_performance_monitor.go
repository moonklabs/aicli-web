package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/aicli/aicli-web/internal/profiling"
	"github.com/aicli/aicli-web/internal/testing"
)

// IntegratedPerformanceMonitor는 통합 성능 모니터링 시스템입니다
type IntegratedPerformanceMonitor struct {
	// 컴포넌트들
	errorTracker       *ErrorTracker
	profiler          *profiling.PerformanceProfiler
	testSuite         *testing.PerformanceTestSuite
	benchmarkSuite    *testing.BenchmarkTestSuite
	alertingSystem    *AlertingSystem
	
	// 설정
	config            MonitoringConfig
	
	// 상태
	running           bool
	mutex             sync.RWMutex
	
	// 생명주기
	ctx               context.Context
	cancel            context.CancelFunc
	wg                sync.WaitGroup
	
	// 메트릭 저장소
	metricsStore      *MetricsStore
	
	// 이벤트 채널
	alertChan         chan Alert
	metricChan        chan Metric
}

// MonitoringConfig는 모니터링 설정입니다
type MonitoringConfig struct {
	// 에러 추적 설정
	ErrorTracking     ErrorTrackerConfig        `json:"error_tracking"`
	
	// 프로파일링 설정
	Profiling         profiling.ProfilingConfig `json:"profiling"`
	
	// 성능 테스트 설정
	PerformanceTesting testing.TestConfig       `json:"performance_testing"`
	
	// 벤치마크 설정
	Benchmarking      testing.BenchmarkConfig   `json:"benchmarking"`
	
	// 알림 설정
	Alerting          AlertingConfig            `json:"alerting"`
	
	// 메트릭 수집 설정
	MetricsCollection MetricsConfig             `json:"metrics_collection"`
}

// MetricsConfig는 메트릭 수집 설정입니다
type MetricsConfig struct {
	CollectionInterval time.Duration `json:"collection_interval"`
	RetentionPeriod    time.Duration `json:"retention_period"`
	BatchSize          int           `json:"batch_size"`
	EnableRealTime     bool          `json:"enable_real_time"`
	
	// 메트릭 타입별 활성화
	EnableSystemMetrics     bool `json:"enable_system_metrics"`
	EnableApplicationMetrics bool `json:"enable_application_metrics"`
	EnableBusinessMetrics   bool `json:"enable_business_metrics"`
}

// Metric은 수집되는 메트릭입니다
type Metric struct {
	Name      string                 `json:"name"`
	Type      string                 `json:"type"` // "counter", "gauge", "histogram"
	Value     float64                `json:"value"`
	Labels    map[string]string      `json:"labels"`
	Timestamp time.Time              `json:"timestamp"`
	Source    string                 `json:"source"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// Alert 타입은 alerting_system.go에서 정의됨

// MetricsStore는 메트릭 저장소입니다
type MetricsStore struct {
	metrics []Metric
	mutex   sync.RWMutex
	config  MetricsConfig
}

// MonitoringReport는 통합 모니터링 보고서입니다
type MonitoringReport struct {
	// 기본 정보
	Timestamp time.Time `json:"timestamp"`
	Duration  time.Duration `json:"duration"`
	
	// 에러 추적 결과
	ErrorSummary *ErrorSummary `json:"error_summary"`
	
	// 프로파일링 결과
	ProfilingReport *profiling.ProfilingReport `json:"profiling_report"`
	
	// 성능 테스트 결과
	LoadTestReport *testing.LoadTestReport `json:"load_test_report"`
	
	// 벤치마크 결과
	BenchmarkReport *testing.BenchmarkReport `json:"benchmark_report"`
	
	// 메트릭 요약
	MetricsSummary *MetricsSummary `json:"metrics_summary"`
	
	// 경고 요약
	AlertsSummary *AlertsSummary `json:"alerts_summary"`
	
	// 전체 건강 점수
	HealthScore float64 `json:"health_score"`
	
	// 권장사항
	Recommendations []string `json:"recommendations"`
	
	// 이슈
	Issues []Issue `json:"issues"`
}

// MetricsSummary는 메트릭 요약입니다
type MetricsSummary struct {
	TotalMetrics    int                     `json:"total_metrics"`
	MetricsByType   map[string]int          `json:"metrics_by_type"`
	MetricsBySource map[string]int          `json:"metrics_by_source"`
	TimeRange       TimeRange               `json:"time_range"`
	TopMetrics      []Metric                `json:"top_metrics"`
	Trends          map[string]MetricTrend  `json:"trends"`
}

// AlertsSummary는 경고 요약입니다
type AlertsSummary struct {
	TotalAlerts      int                    `json:"total_alerts"`
	AlertsByLevel    map[AlertSeverity]int  `json:"alerts_by_level"`
	AlertsBySource   map[string]int         `json:"alerts_by_source"`
	RecentAlerts     []Alert                `json:"recent_alerts"`
	UnresolvedAlerts []Alert                `json:"unresolved_alerts"`
}

// TimeRange는 시간 범위입니다
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// MetricTrend는 메트릭 트렌드입니다
type MetricTrend struct {
	Direction string  `json:"direction"` // "up", "down", "stable"
	Change    float64 `json:"change"`    // 변화율 (%)
	Confidence float64 `json:"confidence"` // 신뢰도 (0-1)
}

// Issue는 발견된 이슈입니다
type Issue struct {
	ID          string     `json:"id"`
	Type        string     `json:"type"`
	Severity    string     `json:"severity"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Source      string     `json:"source"`
	FirstSeen   time.Time  `json:"first_seen"`
	LastSeen    time.Time  `json:"last_seen"`
	Count       int        `json:"count"`
	Resolved    bool       `json:"resolved"`
}

// DefaultMonitoringConfig는 기본 모니터링 설정을 반환합니다
func DefaultMonitoringConfig() MonitoringConfig {
	return MonitoringConfig{
		ErrorTracking:      DefaultErrorTrackerConfig(),
		Profiling:         profiling.DefaultProfilingConfig(),
		PerformanceTesting: testing.DefaultTestConfig(),
		Benchmarking:      testing.DefaultBenchmarkConfig(),
		Alerting:          DefaultAlertingConfig(),
		MetricsCollection: MetricsConfig{
			CollectionInterval: 30 * time.Second,
			RetentionPeriod:   24 * time.Hour,
			BatchSize:         100,
			EnableRealTime:    true,
			EnableSystemMetrics:     true,
			EnableApplicationMetrics: true,
			EnableBusinessMetrics:   false,
		},
	}
}

// NewIntegratedPerformanceMonitor는 새로운 통합 성능 모니터를 생성합니다
func NewIntegratedPerformanceMonitor(config MonitoringConfig) (*IntegratedPerformanceMonitor, error) {
	ctx, cancel := context.WithCancel(context.Background())
	
	// 컴포넌트들 초기화
	errorTracker := NewErrorTracker(config.ErrorTracking)
	
	profiler, err := profiling.NewPerformanceProfiler(config.Profiling)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create profiler: %w", err)
	}
	
	testSuite := testing.NewPerformanceTestSuite(config.PerformanceTesting)
	benchmarkSuite := testing.NewBenchmarkTestSuite(config.Benchmarking)
	alertingSystem := NewAlertingSystem(config.Alerting)
	
	metricsStore := &MetricsStore{
		metrics: make([]Metric, 0),
		config:  config.MetricsCollection,
	}
	
	monitor := &IntegratedPerformanceMonitor{
		errorTracker:    errorTracker,
		profiler:       profiler,
		testSuite:      testSuite,
		benchmarkSuite: benchmarkSuite,
		alertingSystem: alertingSystem,
		config:         config,
		ctx:           ctx,
		cancel:        cancel,
		metricsStore:  metricsStore,
		alertChan:     make(chan Alert, 100),
		metricChan:    make(chan Metric, 1000),
	}
	
	return monitor, nil
}

// Start는 모니터링을 시작합니다
func (ipm *IntegratedPerformanceMonitor) Start() error {
	ipm.mutex.Lock()
	defer ipm.mutex.Unlock()
	
	if ipm.running {
		return fmt.Errorf("monitoring is already running")
	}
	
	fmt.Println("통합 성능 모니터링 시작")
	
	// 프로파일러 시작
	if err := ipm.profiler.Start(); err != nil {
		return fmt.Errorf("failed to start profiler: %w", err)
	}
	
	// 알림 시스템은 별도 시작 없이 즉시 사용 가능
	
	// 백그라운드 작업들 시작
	ipm.wg.Add(3)
	go ipm.metricsCollector()
	go ipm.alertProcessor()
	go ipm.healthChecker()
	
	ipm.running = true
	return nil
}

// Stop은 모니터링을 중지합니다
func (ipm *IntegratedPerformanceMonitor) Stop() error {
	ipm.mutex.Lock()
	defer ipm.mutex.Unlock()
	
	if !ipm.running {
		return nil
	}
	
	fmt.Println("통합 성능 모니터링 중지")
	
	// 컨텍스트 취소
	ipm.cancel()
	
	// 백그라운드 작업 완료 대기
	ipm.wg.Wait()
	
	// 컴포넌트들 중지
	ipm.profiler.Stop()
	ipm.alertingSystem.Stop()
	ipm.errorTracker.Stop()
	
	ipm.running = false
	return nil
}

// CollectMetric은 메트릭을 수집합니다
func (ipm *IntegratedPerformanceMonitor) CollectMetric(metric Metric) {
	select {
	case ipm.metricChan <- metric:
	default:
		// 채널이 가득 찬 경우 로그만 남기고 메트릭 손실
		fmt.Printf("Metric channel full, dropping metric: %s\n", metric.Name)
	}
}

// SendAlert는 경고를 발송합니다
func (ipm *IntegratedPerformanceMonitor) SendAlert(alert Alert) {
	select {
	case ipm.alertChan <- alert:
	default:
		// 채널이 가득 찬 경우 중요한 경고는 즉시 처리
		if alert.Severity == SeverityCritical {
			go ipm.alertingSystem.AddAlert(alert)
		}
	}
}

// GenerateReport는 통합 모니터링 보고서를 생성합니다
func (ipm *IntegratedPerformanceMonitor) GenerateReport() (*MonitoringReport, error) {
	startTime := time.Now()
	
	report := &MonitoringReport{
		Timestamp: startTime,
	}
	
	// 각 컴포넌트에서 데이터 수집
	report.ErrorSummary = ipm.errorTracker.GetSummary()
	
	if profilingReport, err := ipm.profiler.Capture(); err == nil {
		report.ProfilingReport = profilingReport
	}
	
	report.BenchmarkReport = ipm.benchmarkSuite.GenerateReport()
	report.MetricsSummary = ipm.generateMetricsSummary()
	report.AlertsSummary = ipm.generateAlertsSummary()
	
	// 건강 점수 계산
	report.HealthScore = ipm.calculateHealthScore(report)
	
	// 권장사항 생성
	report.Recommendations = ipm.generateRecommendations(report)
	
	// 이슈 탐지
	report.Issues = ipm.detectIssues(report)
	
	report.Duration = time.Since(startTime)
	
	return report, nil
}

// RunPerformanceTest는 성능 테스트를 실행합니다
func (ipm *IntegratedPerformanceMonitor) RunPerformanceTest() (*testing.LoadTestReport, error) {
	fmt.Println("성능 테스트 실행 중...")
	return ipm.testSuite.RunLoadTest()
}

// RunBenchmark는 벤치마크를 실행합니다
func (ipm *IntegratedPerformanceMonitor) RunBenchmark(name string, fn testing.BenchmarkFunction) (*testing.BenchmarkResult, error) {
	fmt.Printf("벤치마크 실행 중: %s\n", name)
	return ipm.benchmarkSuite.RunBenchmark(name, fn)
}

// 내부 메서드들

func (ipm *IntegratedPerformanceMonitor) metricsCollector() {
	defer ipm.wg.Done()
	
	ticker := time.NewTicker(ipm.config.MetricsCollection.CollectionInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ipm.ctx.Done():
			return
		case metric := <-ipm.metricChan:
			ipm.metricsStore.AddMetric(metric)
		case <-ticker.C:
			ipm.collectSystemMetrics()
		}
	}
}

func (ipm *IntegratedPerformanceMonitor) alertProcessor() {
	defer ipm.wg.Done()
	
	for {
		select {
		case <-ipm.ctx.Done():
			return
		case alert := <-ipm.alertChan:
			ipm.alertingSystem.AddAlert(alert)
		}
	}
}

func (ipm *IntegratedPerformanceMonitor) healthChecker() {
	defer ipm.wg.Done()
	
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-ipm.ctx.Done():
			return
		case <-ticker.C:
			ipm.performHealthCheck()
		}
	}
}

func (ipm *IntegratedPerformanceMonitor) collectSystemMetrics() {
	now := time.Now()
	
	if ipm.config.MetricsCollection.EnableSystemMetrics {
		// 시스템 메트릭 수집
		stats := ipm.profiler.GetCurrentStats()
		
		for name, value := range stats {
			if metric, ok := ipm.convertToMetric(name, value, now); ok {
				ipm.metricsStore.AddMetric(metric)
			}
		}
	}
}

func (ipm *IntegratedPerformanceMonitor) convertToMetric(name string, value interface{}, timestamp time.Time) (Metric, bool) {
	var floatValue float64
	var ok bool
	
	switch v := value.(type) {
	case float64:
		floatValue = v
		ok = true
	case int:
		floatValue = float64(v)
		ok = true
	case int64:
		floatValue = float64(v)
		ok = true
	case uint64:
		floatValue = float64(v)
		ok = true
	}
	
	if !ok {
		return Metric{}, false
	}
	
	return Metric{
		Name:      name,
		Type:      "gauge",
		Value:     floatValue,
		Timestamp: timestamp,
		Source:    "system",
		Labels:    map[string]string{"component": "performance_monitor"},
	}, true
}

func (ipm *IntegratedPerformanceMonitor) performHealthCheck() {
	// 각 컴포넌트의 건강 상태 확인
	issues := make([]Issue, 0)
	
	// 에러율 확인
	errorSummary := ipm.errorTracker.GetSummary()
	if errorSummary.ErrorsToday > 100 {
		issues = append(issues, Issue{
			ID:          fmt.Sprintf("high_error_rate_%d", time.Now().Unix()),
			Type:        "error_rate",
			Severity:    "warning",
			Title:       "높은 에러율 감지",
			Description: fmt.Sprintf("오늘 %d개의 에러가 발생했습니다", errorSummary.ErrorsToday),
			Source:      "error_tracker",
			FirstSeen:   time.Now(),
			LastSeen:    time.Now(),
			Count:       1,
		})
	}
	
	// 이슈가 있으면 알림 발송
	for _, issue := range issues {
		alert := Alert{
			ID:        issue.ID,
			Name:      issue.Title,
			Message:   issue.Description,
			Severity:  SeverityError,
			Status:    StatusFiring,
			StartsAt:  time.Now(),
			Labels:    map[string]string{"source": issue.Source},
		}
		ipm.SendAlert(alert)
	}
}

func (ipm *IntegratedPerformanceMonitor) generateMetricsSummary() *MetricsSummary {
	metrics := ipm.metricsStore.GetMetrics()
	
	summary := &MetricsSummary{
		TotalMetrics:    len(metrics),
		MetricsByType:   make(map[string]int),
		MetricsBySource: make(map[string]int),
		Trends:          make(map[string]MetricTrend),
	}
	
	if len(metrics) == 0 {
		return summary
	}
	
	// 시간 범위 계산
	summary.TimeRange.Start = metrics[0].Timestamp
	summary.TimeRange.End = metrics[len(metrics)-1].Timestamp
	
	// 타입별, 소스별 분류
	for _, metric := range metrics {
		summary.MetricsByType[metric.Type]++
		summary.MetricsBySource[metric.Source]++
		
		if summary.TimeRange.Start.After(metric.Timestamp) {
			summary.TimeRange.Start = metric.Timestamp
		}
		if summary.TimeRange.End.Before(metric.Timestamp) {
			summary.TimeRange.End = metric.Timestamp
		}
	}
	
	// 상위 메트릭 선별 (최근 10개)
	if len(metrics) > 10 {
		summary.TopMetrics = metrics[len(metrics)-10:]
	} else {
		summary.TopMetrics = metrics
	}
	
	return summary
}

func (ipm *IntegratedPerformanceMonitor) generateAlertsSummary() *AlertsSummary {
	alerts := ipm.alertingSystem.GetAlerts()
	
	summary := &AlertsSummary{
		TotalAlerts:      len(alerts),
		AlertsByLevel:    make(map[AlertSeverity]int),
		AlertsBySource:   make(map[string]int),
		RecentAlerts:     make([]Alert, 0),
		UnresolvedAlerts: make([]Alert, 0),
	}
	
	cutoff := time.Now().Add(-24 * time.Hour)
	
	for _, alert := range alerts {
		summary.AlertsByLevel[alert.Severity]++
		// Alert 구조체에 Source 필드가 없으므로 Labels에서 가져옴
		if source, ok := alert.Labels["source"]; ok {
			summary.AlertsBySource[source]++
		}
		
		if alert.StartsAt.After(cutoff) {
			summary.RecentAlerts = append(summary.RecentAlerts, *alert)
		}
		
		if alert.Status != StatusResolved {
			summary.UnresolvedAlerts = append(summary.UnresolvedAlerts, *alert)
		}
	}
	
	return summary
}

func (ipm *IntegratedPerformanceMonitor) calculateHealthScore(report *MonitoringReport) float64 {
	score := 100.0
	
	// 에러율 반영
	if report.ErrorSummary != nil && report.ErrorSummary.TotalErrors > 0 {
		errorRate := float64(report.ErrorSummary.ErrorsToday) / float64(report.ErrorSummary.TotalErrors) * 100.0
		score -= errorRate * 0.5
	}
	
	// 알림 수준 반영
	if report.AlertsSummary != nil {
		criticalAlerts := report.AlertsSummary.AlertsByLevel[SeverityCritical]
		errorAlerts := report.AlertsSummary.AlertsByLevel[SeverityError]
		score -= float64(criticalAlerts)*10 + float64(errorAlerts)*5
	}
	
	// 성능 메트릭 반영
	if report.ProfilingReport != nil {
		score += report.ProfilingReport.Summary.PerformanceScore * 0.3
	}
	
	// 0-100 범위로 제한
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	
	return score
}

func (ipm *IntegratedPerformanceMonitor) generateRecommendations(report *MonitoringReport) []string {
	recommendations := make([]string, 0)
	
	// 건강 점수 기반 권장사항
	if report.HealthScore < 50 {
		recommendations = append(recommendations, "시스템 건강 상태가 매우 나쁩니다. 즉시 조치가 필요합니다.")
	} else if report.HealthScore < 70 {
		recommendations = append(recommendations, "시스템 성능 개선이 필요합니다.")
	}
	
	// 에러 기반 권장사항
	if report.ErrorSummary != nil && report.ErrorSummary.ErrorsToday > 50 {
		recommendations = append(recommendations, "높은 에러율을 보이고 있습니다. 로그를 확인하고 원인을 파악하세요.")
	}
	
	// 프로파일링 기반 권장사항
	if report.ProfilingReport != nil {
		recommendations = append(recommendations, report.ProfilingReport.Summary.Recommendations...)
	}
	
	return recommendations
}

func (ipm *IntegratedPerformanceMonitor) detectIssues(report *MonitoringReport) []Issue {
	issues := make([]Issue, 0)
	
	// 높은 에러율 이슈
	if report.ErrorSummary != nil && report.ErrorSummary.ErrorsToday > 100 {
		issues = append(issues, Issue{
			ID:          fmt.Sprintf("high_error_rate_%d", time.Now().Unix()),
			Type:        "error_rate",
			Severity:    "high",
			Title:       "높은 에러율",
			Description: fmt.Sprintf("오늘 %d개의 에러가 발생했습니다", report.ErrorSummary.ErrorsToday),
			Source:      "error_tracker",
			FirstSeen:   time.Now(),
			LastSeen:    time.Now(),
			Count:       1,
		})
	}
	
	// 낮은 건강 점수 이슈
	if report.HealthScore < 50 {
		issues = append(issues, Issue{
			ID:          fmt.Sprintf("low_health_score_%d", time.Now().Unix()),
			Type:        "health_score",
			Severity:    "critical",
			Title:       "낮은 시스템 건강 점수",
			Description: fmt.Sprintf("현재 건강 점수: %.1f", report.HealthScore),
			Source:      "health_monitor",
			FirstSeen:   time.Now(),
			LastSeen:    time.Now(),
			Count:       1,
		})
	}
	
	return issues
}

// MetricsStore 메서드들

func (ms *MetricsStore) AddMetric(metric Metric) {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	
	ms.metrics = append(ms.metrics, metric)
	
	// 보존 기간 초과 메트릭 제거
	cutoff := time.Now().Add(-ms.config.RetentionPeriod)
	newMetrics := make([]Metric, 0)
	
	for _, m := range ms.metrics {
		if m.Timestamp.After(cutoff) {
			newMetrics = append(newMetrics, m)
		}
	}
	
	ms.metrics = newMetrics
}

func (ms *MetricsStore) GetMetrics() []Metric {
	ms.mutex.RLock()
	defer ms.mutex.RUnlock()
	
	result := make([]Metric, len(ms.metrics))
	copy(result, ms.metrics)
	return result
}

func (ms *MetricsStore) GetMetricsByName(name string) []Metric {
	ms.mutex.RLock()
	defer ms.mutex.RUnlock()
	
	result := make([]Metric, 0)
	for _, metric := range ms.metrics {
		if metric.Name == name {
			result = append(result, metric)
		}
	}
	
	return result
}

func (ms *MetricsStore) GetMetricsByTimeRange(start, end time.Time) []Metric {
	ms.mutex.RLock()
	defer ms.mutex.RUnlock()
	
	result := make([]Metric, 0)
	for _, metric := range ms.metrics {
		if metric.Timestamp.After(start) && metric.Timestamp.Before(end) {
			result = append(result, metric)
		}
	}
	
	return result
}

// ExportMetrics는 메트릭을 JSON으로 내보냅니다
func (ms *MetricsStore) ExportMetrics() ([]byte, error) {
	ms.mutex.RLock()
	defer ms.mutex.RUnlock()
	
	return json.MarshalIndent(ms.metrics, "", "  ")
}