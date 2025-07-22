package claude

import (
	"encoding/json"
	"sync"
	"sync/atomic"
	"time"
)

// ErrorStatisticsCollector는 에러 통계를 수집하고 분석합니다
type ErrorStatisticsCollector struct {
	// 전체 통계
	totalErrors       int64
	totalRecoveries   int64
	successfulRecoveries int64
	
	// 에러 유형별 통계
	errorStats        map[ErrorType]*ErrorTypeStats
	errorStatsMutex   sync.RWMutex
	
	// 시간대별 통계
	hourlyStats       map[int]*HourlyStats  // 0-23시간
	dailyStats        map[string]*DailyStats // YYYY-MM-DD
	monthlyStats      map[string]*MonthlyStats // YYYY-MM
	timeStatsMutex    sync.RWMutex
	
	// 복구 전략별 통계
	strategyStats     map[string]*StrategyStats
	strategyMutex     sync.RWMutex
	
	// 최근 에러 패턴
	recentPatterns    []ErrorPattern
	patternsMutex     sync.RWMutex
	maxPatterns       int
	
	// 실시간 메트릭
	realTimeMetrics   *RealTimeMetrics
	metricsMutex      sync.RWMutex
	
	// 통계 설정
	config            StatisticsConfig
	
	// 생명주기
	startTime         time.Time
	lastUpdate        time.Time
}

// ErrorTypeStats는 에러 유형별 통계입니다
type ErrorTypeStats struct {
	Type              ErrorType     `json:"type"`
	TotalCount        int64         `json:"total_count"`
	RecentCount       int64         `json:"recent_count"` // 최근 24시간
	FirstOccurrence   time.Time     `json:"first_occurrence"`
	LastOccurrence    time.Time     `json:"last_occurrence"`
	AverageFrequency  float64       `json:"average_frequency"` // 시간당 평균
	PeakFrequency     float64       `json:"peak_frequency"`
	PeakTime          time.Time     `json:"peak_time"`
	
	// 복구 통계
	RecoveryAttempts  int64         `json:"recovery_attempts"`
	SuccessfulRecoveries int64      `json:"successful_recoveries"`
	RecoverySuccessRate float64     `json:"recovery_success_rate"`
	AverageRecoveryTime time.Duration `json:"average_recovery_time"`
	
	// 심각도별 분포
	SeverityDistribution map[ErrorSeverity]int64 `json:"severity_distribution"`
	
	// 관련 컴포넌트
	AffectedComponents   map[string]int64 `json:"affected_components"`
	
	// 트렌드 정보
	Trend             TrendDirection `json:"trend"`
	TrendConfidence   float64        `json:"trend_confidence"`
}

// HourlyStats는 시간별 통계입니다
type HourlyStats struct {
	Hour              int           `json:"hour"` // 0-23
	ErrorCount        int64         `json:"error_count"`
	RecoveryCount     int64         `json:"recovery_count"`
	SuccessfulRecoveries int64      `json:"successful_recoveries"`
	AverageResponseTime time.Duration `json:"average_response_time"`
	ErrorsByType      map[ErrorType]int64 `json:"errors_by_type"`
	LastUpdated       time.Time     `json:"last_updated"`
}

// DailyStats는 일별 통계입니다
type DailyStats struct {
	Date              string        `json:"date"` // YYYY-MM-DD
	ErrorCount        int64         `json:"error_count"`
	RecoveryCount     int64         `json:"recovery_count"`
	SuccessfulRecoveries int64      `json:"successful_recoveries"`
	Uptime            time.Duration `json:"uptime"`
	Downtime          time.Duration `json:"downtime"`
	MTTR              time.Duration `json:"mttr"` // Mean Time To Recovery
	MTBF              time.Duration `json:"mtbf"` // Mean Time Between Failures
	ErrorsByType      map[ErrorType]int64 `json:"errors_by_type"`
	HourlyBreakdown   [24]int64     `json:"hourly_breakdown"`
	WorstHour         int           `json:"worst_hour"`
	BestHour          int           `json:"best_hour"`
}

// MonthlyStats는 월별 통계입니다
type MonthlyStats struct {
	Month             string        `json:"month"` // YYYY-MM
	ErrorCount        int64         `json:"error_count"`
	RecoveryCount     int64         `json:"recovery_count"`
	SuccessfulRecoveries int64      `json:"successful_recoveries"`
	TotalUptime       time.Duration `json:"total_uptime"`
	TotalDowntime     time.Duration `json:"total_downtime"`
	AvailabilityPercent float64     `json:"availability_percent"`
	ErrorsByType      map[ErrorType]int64 `json:"errors_by_type"`
	DailyBreakdown    map[string]int64 `json:"daily_breakdown"`
	WorstDay          string        `json:"worst_day"`
	BestDay           string        `json:"best_day"`
	Trends            map[ErrorType]TrendDirection `json:"trends"`
}

// StrategyStats는 복구 전략별 통계입니다
type StrategyStats struct {
	StrategyName      string        `json:"strategy_name"`
	TotalAttempts     int64         `json:"total_attempts"`
	SuccessfulAttempts int64        `json:"successful_attempts"`
	SuccessRate       float64       `json:"success_rate"`
	AverageTime       time.Duration `json:"average_time"`
	MinTime           time.Duration `json:"min_time"`
	MaxTime           time.Duration `json:"max_time"`
	RecentSuccessRate float64       `json:"recent_success_rate"` // 최근 24시간
	LastUsed          time.Time     `json:"last_used"`
	
	// 에러 유형별 성공률
	SuccessByErrorType map[ErrorType]float64 `json:"success_by_error_type"`
	
	// 시간대별 성공률
	SuccessByHour     [24]float64   `json:"success_by_hour"`
}

// RealTimeMetrics는 실시간 메트릭입니다
type RealTimeMetrics struct {
	CurrentErrorRate    float64       `json:"current_error_rate"`    // 에러/분
	CurrentRecoveryRate float64       `json:"current_recovery_rate"` // 복구/분
	ActiveRecoveries    int           `json:"active_recoveries"`
	SystemHealth        float64       `json:"system_health"`         // 0.0-1.0
	LastMinuteErrors    int64         `json:"last_minute_errors"`
	LastHourErrors      int64         `json:"last_hour_errors"`
	Last24HourErrors    int64         `json:"last_24hour_errors"`
	
	// 예측 메트릭
	PredictedErrorRate  float64       `json:"predicted_error_rate"`
	PredictionConfidence float64      `json:"prediction_confidence"`
	
	// 성능 메트릭
	AverageRecoveryTime time.Duration `json:"average_recovery_time"`
	P95RecoveryTime     time.Duration `json:"p95_recovery_time"`
	P99RecoveryTime     time.Duration `json:"p99_recovery_time"`
	
	// 시스템 부하
	CPUUsage            float64       `json:"cpu_usage"`
	MemoryUsage         float64       `json:"memory_usage"`
	DiskUsage           float64       `json:"disk_usage"`
	NetworkLatency      time.Duration `json:"network_latency"`
}

// TrendDirection은 트렌드 방향입니다
type TrendDirection int

const (
	TrendUnknown TrendDirection = iota
	TrendDecreasing
	TrendStable
	TrendIncreasing
	TrendSpike
	TrendDrop
)

// StatisticsConfig는 통계 수집 설정입니다
type StatisticsConfig struct {
	CollectionInterval  time.Duration `json:"collection_interval"`
	RetentionPeriod     time.Duration `json:"retention_period"`
	MaxPatterns         int           `json:"max_patterns"`
	TrendAnalysisWindow int           `json:"trend_analysis_window"` // 데이터 포인트 수
	PredictionEnabled   bool          `json:"prediction_enabled"`
	StatisticsAlertThresholds     StatisticsAlertThresholds `json:"alert_thresholds"`
}

// StatisticsAlertThresholds는 알림 임계값입니다 (통계용)
type StatisticsAlertThresholds struct {
	ErrorRateThreshold      float64 `json:"error_rate_threshold"`
	RecoveryFailureThreshold float64 `json:"recovery_failure_threshold"`
	DowntimeThreshold       time.Duration `json:"downtime_threshold"`
	ResponseTimeThreshold   time.Duration `json:"response_time_threshold"`
}

// ErrorMetricsSummary는 에러 메트릭 요약입니다
type ErrorMetricsSummary struct {
	// 기본 카운터
	TotalErrors         int64         `json:"total_errors"`
	TotalRecoveries     int64         `json:"total_recoveries"`
	SuccessfulRecoveries int64        `json:"successful_recoveries"`
	RecoverySuccessRate float64       `json:"recovery_success_rate"`
	
	// 시간 관련
	Uptime              time.Duration `json:"uptime"`
	MTTR                time.Duration `json:"mttr"`
	MTBF                time.Duration `json:"mtbf"`
	
	// 실시간 메트릭
	RealTime            RealTimeMetrics `json:"real_time"`
	
	// 통계 기간
	StartTime           time.Time     `json:"start_time"`
	LastUpdate          time.Time     `json:"last_update"`
	
	// 에러 유형별 상위 5개
	TopErrorTypes       []ErrorTypeStats `json:"top_error_types"`
	
	// 복구 전략별 상위 5개
	TopStrategies       []StrategyStats  `json:"top_strategies"`
	
	// 트렌드 정보
	OverallTrend        TrendDirection   `json:"overall_trend"`
	TrendConfidence     float64          `json:"trend_confidence"`
}

// NewErrorStatisticsCollector는 새로운 에러 통계 수집기를 생성합니다
func NewErrorStatisticsCollector(config StatisticsConfig) *ErrorStatisticsCollector {
	collector := &ErrorStatisticsCollector{
		errorStats:      make(map[ErrorType]*ErrorTypeStats),
		hourlyStats:     make(map[int]*HourlyStats),
		dailyStats:      make(map[string]*DailyStats),
		monthlyStats:    make(map[string]*MonthlyStats),
		strategyStats:   make(map[string]*StrategyStats),
		recentPatterns:  make([]ErrorPattern, 0),
		maxPatterns:     config.MaxPatterns,
		config:          config,
		startTime:       time.Now(),
		lastUpdate:      time.Now(),
		realTimeMetrics: &RealTimeMetrics{},
	}
	
	return collector
}

// RecordError는 에러 발생을 기록합니다
func (c *ErrorStatisticsCollector) RecordError(err error, errorClass ErrorClass, component string) {
	now := time.Now()
	
	// 전체 카운터 증가
	atomic.AddInt64(&c.totalErrors, 1)
	
	// 에러 유형별 통계 업데이트
	c.updateErrorTypeStats(errorClass.Type, errorClass.Severity, component, now)
	
	// 시간대별 통계 업데이트
	c.updateTimeStats(errorClass.Type, now)
	
	// 실시간 메트릭 업데이트
	c.updateRealTimeMetrics()
	
	// 마지막 업데이트 시간 갱신
	c.lastUpdate = now
}

// RecordRecovery는 복구 시도를 기록합니다
func (c *ErrorStatisticsCollector) RecordRecovery(strategyName string, errorType ErrorType, success bool, duration time.Duration) {
	now := time.Now()
	
	// 전체 카운터 증가
	atomic.AddInt64(&c.totalRecoveries, 1)
	if success {
		atomic.AddInt64(&c.successfulRecoveries, 1)
	}
	
	// 전략별 통계 업데이트
	c.updateStrategyStats(strategyName, errorType, success, duration, now)
	
	// 에러 유형별 복구 통계 업데이트
	c.updateErrorTypeRecoveryStats(errorType, success, duration)
	
	// 시간대별 복구 통계 업데이트
	c.updateTimeRecoveryStats(success, now)
	
	// 마지막 업데이트 시간 갱신
	c.lastUpdate = now
}

// GetSummary는 통계 요약을 반환합니다
func (c *ErrorStatisticsCollector) GetSummary() ErrorMetricsSummary {
	c.updateRealTimeMetrics()
	
	summary := ErrorMetricsSummary{
		TotalErrors:         atomic.LoadInt64(&c.totalErrors),
		TotalRecoveries:     atomic.LoadInt64(&c.totalRecoveries),
		SuccessfulRecoveries: atomic.LoadInt64(&c.successfulRecoveries),
		Uptime:              time.Since(c.startTime),
		RealTime:            *c.realTimeMetrics,
		StartTime:           c.startTime,
		LastUpdate:          c.lastUpdate,
	}
	
	// 복구 성공률 계산
	if summary.TotalRecoveries > 0 {
		summary.RecoverySuccessRate = float64(summary.SuccessfulRecoveries) / float64(summary.TotalRecoveries)
	}
	
	// MTTR, MTBF 계산
	summary.MTTR = c.calculateMTTR()
	summary.MTBF = c.calculateMTBF()
	
	// 상위 에러 유형
	summary.TopErrorTypes = c.getTopErrorTypes(5)
	
	// 상위 복구 전략
	summary.TopStrategies = c.getTopStrategies(5)
	
	// 전체 트렌드
	summary.OverallTrend, summary.TrendConfidence = c.analyzeOverallTrend()
	
	return summary
}

// GetErrorTypeStats는 특정 에러 유형의 상세 통계를 반환합니다
func (c *ErrorStatisticsCollector) GetErrorTypeStats(errorType ErrorType) *ErrorTypeStats {
	c.errorStatsMutex.RLock()
	defer c.errorStatsMutex.RUnlock()
	
	if stats, exists := c.errorStats[errorType]; exists {
		// 복사본 반환
		statsCopy := *stats
		statsCopy.SeverityDistribution = make(map[ErrorSeverity]int64)
		for k, v := range stats.SeverityDistribution {
			statsCopy.SeverityDistribution[k] = v
		}
		statsCopy.AffectedComponents = make(map[string]int64)
		for k, v := range stats.AffectedComponents {
			statsCopy.AffectedComponents[k] = v
		}
		return &statsCopy
	}
	
	return nil
}

// GetHourlyStats는 시간별 통계를 반환합니다
func (c *ErrorStatisticsCollector) GetHourlyStats() [24]*HourlyStats {
	c.timeStatsMutex.RLock()
	defer c.timeStatsMutex.RUnlock()
	
	var stats [24]*HourlyStats
	for i := 0; i < 24; i++ {
		if hourStats, exists := c.hourlyStats[i]; exists {
			// 복사본 생성
			statsCopy := *hourStats
			statsCopy.ErrorsByType = make(map[ErrorType]int64)
			for k, v := range hourStats.ErrorsByType {
				statsCopy.ErrorsByType[k] = v
			}
			stats[i] = &statsCopy
		}
	}
	
	return stats
}

// GetDailyStats는 일별 통계를 반환합니다
func (c *ErrorStatisticsCollector) GetDailyStats(days int) []DailyStats {
	c.timeStatsMutex.RLock()
	defer c.timeStatsMutex.RUnlock()
	
	var stats []DailyStats
	now := time.Now()
	
	for i := 0; i < days; i++ {
		date := now.AddDate(0, 0, -i).Format("2006-01-02")
		if dayStats, exists := c.dailyStats[date]; exists {
			// 복사본 생성
			statsCopy := *dayStats
			statsCopy.ErrorsByType = make(map[ErrorType]int64)
			for k, v := range dayStats.ErrorsByType {
				statsCopy.ErrorsByType[k] = v
			}
			stats = append(stats, statsCopy)
		}
	}
	
	return stats
}

// GetStrategyStats는 복구 전략 통계를 반환합니다
func (c *ErrorStatisticsCollector) GetStrategyStats() []StrategyStats {
	c.strategyMutex.RLock()
	defer c.strategyMutex.RUnlock()
	
	var stats []StrategyStats
	for _, strategyStats := range c.strategyStats {
		// 복사본 생성
		statsCopy := *strategyStats
		statsCopy.SuccessByErrorType = make(map[ErrorType]float64)
		for k, v := range strategyStats.SuccessByErrorType {
			statsCopy.SuccessByErrorType[k] = v
		}
		stats = append(stats, statsCopy)
	}
	
	return stats
}

// ExportJSON은 통계를 JSON으로 내보냅니다
func (c *ErrorStatisticsCollector) ExportJSON() ([]byte, error) {
	summary := c.GetSummary()
	return json.MarshalIndent(summary, "", "  ")
}

// 내부 메서드들

func (c *ErrorStatisticsCollector) updateErrorTypeStats(errorType ErrorType, severity ErrorSeverity, component string, timestamp time.Time) {
	c.errorStatsMutex.Lock()
	defer c.errorStatsMutex.Unlock()
	
	stats, exists := c.errorStats[errorType]
	if !exists {
		stats = &ErrorTypeStats{
			Type:                 errorType,
			FirstOccurrence:      timestamp,
			SeverityDistribution: make(map[ErrorSeverity]int64),
			AffectedComponents:   make(map[string]int64),
		}
		c.errorStats[errorType] = stats
	}
	
	// 카운터 업데이트
	stats.TotalCount++
	stats.LastOccurrence = timestamp
	
	// 심각도별 분포 업데이트
	stats.SeverityDistribution[severity]++
	
	// 영향받은 컴포넌트 업데이트
	if component != "" {
		stats.AffectedComponents[component]++
	}
	
	// 빈도 계산 (간단한 구현)
	elapsed := timestamp.Sub(stats.FirstOccurrence)
	if elapsed > 0 {
		stats.AverageFrequency = float64(stats.TotalCount) / elapsed.Hours()
	}
	
	// 최근 24시간 카운트 업데이트 (실제로는 더 정확한 구현 필요)
	if timestamp.Sub(stats.LastOccurrence) < 24*time.Hour {
		stats.RecentCount++
	}
}

func (c *ErrorStatisticsCollector) updateTimeStats(errorType ErrorType, timestamp time.Time) {
	c.timeStatsMutex.Lock()
	defer c.timeStatsMutex.Unlock()
	
	hour := timestamp.Hour()
	date := timestamp.Format("2006-01-02")
	month := timestamp.Format("2006-01")
	
	// 시간별 통계
	hourStats, exists := c.hourlyStats[hour]
	if !exists {
		hourStats = &HourlyStats{
			Hour:         hour,
			ErrorsByType: make(map[ErrorType]int64),
		}
		c.hourlyStats[hour] = hourStats
	}
	hourStats.ErrorCount++
	hourStats.ErrorsByType[errorType]++
	hourStats.LastUpdated = timestamp
	
	// 일별 통계
	dayStats, exists := c.dailyStats[date]
	if !exists {
		dayStats = &DailyStats{
			Date:         date,
			ErrorsByType: make(map[ErrorType]int64),
		}
		c.dailyStats[date] = dayStats
	}
	dayStats.ErrorCount++
	dayStats.ErrorsByType[errorType]++
	dayStats.HourlyBreakdown[hour]++
	
	// 월별 통계
	monthStats, exists := c.monthlyStats[month]
	if !exists {
		monthStats = &MonthlyStats{
			Month:          month,
			ErrorsByType:   make(map[ErrorType]int64),
			DailyBreakdown: make(map[string]int64),
			Trends:         make(map[ErrorType]TrendDirection),
		}
		c.monthlyStats[month] = monthStats
	}
	monthStats.ErrorCount++
	monthStats.ErrorsByType[errorType]++
	monthStats.DailyBreakdown[date]++
}

func (c *ErrorStatisticsCollector) updateStrategyStats(strategyName string, errorType ErrorType, success bool, duration time.Duration, timestamp time.Time) {
	c.strategyMutex.Lock()
	defer c.strategyMutex.Unlock()
	
	stats, exists := c.strategyStats[strategyName]
	if !exists {
		stats = &StrategyStats{
			StrategyName:       strategyName,
			SuccessByErrorType: make(map[ErrorType]float64),
			MinTime:            duration,
			MaxTime:            duration,
		}
		c.strategyStats[strategyName] = stats
	}
	
	// 기본 통계 업데이트
	stats.TotalAttempts++
	if success {
		stats.SuccessfulAttempts++
	}
	stats.LastUsed = timestamp
	
	// 성공률 계산
	if stats.TotalAttempts > 0 {
		stats.SuccessRate = float64(stats.SuccessfulAttempts) / float64(stats.TotalAttempts)
	}
	
	// 시간 통계 업데이트
	if duration < stats.MinTime {
		stats.MinTime = duration
	}
	if duration > stats.MaxTime {
		stats.MaxTime = duration
	}
	
	// 평균 시간 계산 (간단한 구현)
	stats.AverageTime = (stats.AverageTime*time.Duration(stats.TotalAttempts-1) + duration) / time.Duration(stats.TotalAttempts)
	
	// 에러 유형별 성공률 업데이트 (간단한 구현)
	stats.SuccessByErrorType[errorType] = stats.SuccessRate
	
	// 시간대별 성공률 업데이트
	hour := timestamp.Hour()
	if success {
		stats.SuccessByHour[hour] = (stats.SuccessByHour[hour] + 1.0) / 2.0
	} else {
		stats.SuccessByHour[hour] = stats.SuccessByHour[hour] / 2.0
	}
}

func (c *ErrorStatisticsCollector) updateErrorTypeRecoveryStats(errorType ErrorType, success bool, duration time.Duration) {
	c.errorStatsMutex.Lock()
	defer c.errorStatsMutex.Unlock()
	
	stats, exists := c.errorStats[errorType]
	if !exists {
		return
	}
	
	stats.RecoveryAttempts++
	if success {
		stats.SuccessfulRecoveries++
	}
	
	// 복구 성공률 계산
	if stats.RecoveryAttempts > 0 {
		stats.RecoverySuccessRate = float64(stats.SuccessfulRecoveries) / float64(stats.RecoveryAttempts)
	}
	
	// 평균 복구 시간 계산
	if success {
		if stats.AverageRecoveryTime == 0 {
			stats.AverageRecoveryTime = duration
		} else {
			stats.AverageRecoveryTime = (stats.AverageRecoveryTime + duration) / 2
		}
	}
}

func (c *ErrorStatisticsCollector) updateTimeRecoveryStats(success bool, timestamp time.Time) {
	c.timeStatsMutex.Lock()
	defer c.timeStatsMutex.Unlock()
	
	hour := timestamp.Hour()
	date := timestamp.Format("2006-01-02")
	month := timestamp.Format("2006-01")
	
	// 시간별 복구 통계
	if hourStats, exists := c.hourlyStats[hour]; exists {
		hourStats.RecoveryCount++
		if success {
			hourStats.SuccessfulRecoveries++
		}
	}
	
	// 일별 복구 통계
	if dayStats, exists := c.dailyStats[date]; exists {
		dayStats.RecoveryCount++
		if success {
			dayStats.SuccessfulRecoveries++
		}
	}
	
	// 월별 복구 통계
	if monthStats, exists := c.monthlyStats[month]; exists {
		monthStats.RecoveryCount++
		if success {
			monthStats.SuccessfulRecoveries++
		}
	}
}

func (c *ErrorStatisticsCollector) updateRealTimeMetrics() {
	c.metricsMutex.Lock()
	defer c.metricsMutex.Unlock()
	
	now := time.Now()
	
	// 시간 윈도우별 에러 수 계산
	c.realTimeMetrics.LastMinuteErrors = c.countErrorsInWindow(now, time.Minute)
	c.realTimeMetrics.LastHourErrors = c.countErrorsInWindow(now, time.Hour)
	c.realTimeMetrics.Last24HourErrors = c.countErrorsInWindow(now, 24*time.Hour)
	
	// 현재 에러율 계산 (에러/분)
	c.realTimeMetrics.CurrentErrorRate = float64(c.realTimeMetrics.LastMinuteErrors)
	
	// 복구율 계산
	c.realTimeMetrics.CurrentRecoveryRate = c.calculateRecoveryRate(now)
	
	// 시스템 건강도 계산 (간단한 구현)
	c.realTimeMetrics.SystemHealth = c.calculateSystemHealth()
	
	// 예측 메트릭 (간단한 구현)
	if c.config.PredictionEnabled {
		c.realTimeMetrics.PredictedErrorRate = c.predictErrorRate()
		c.realTimeMetrics.PredictionConfidence = 0.7 // 임시값
	}
}

func (c *ErrorStatisticsCollector) countErrorsInWindow(endTime time.Time, window time.Duration) int64 {
	// 실제 구현에서는 더 효율적인 방법 사용
	startTime := endTime.Add(-window)
	
	// 간단한 구현 - 실제로는 시계열 데이터베이스 활용
	var count int64
	c.errorStatsMutex.RLock()
	for _, stats := range c.errorStats {
		if stats.LastOccurrence.After(startTime) && stats.LastOccurrence.Before(endTime) {
			count += stats.RecentCount
		}
	}
	c.errorStatsMutex.RUnlock()
	
	return count
}

func (c *ErrorStatisticsCollector) calculateRecoveryRate(now time.Time) float64 {
	// 지난 1분간 복구 횟수
	minute := now.Minute()
	if minute < 0 || minute >= 60 {
		return 0.0
	}
	
	hour := now.Hour()
	c.timeStatsMutex.RLock()
	hourStats, exists := c.hourlyStats[hour]
	c.timeStatsMutex.RUnlock()
	
	if !exists {
		return 0.0
	}
	
	// 간단한 계산 (실제로는 더 정확한 구현 필요)
	return float64(hourStats.RecoveryCount) / 60.0
}

func (c *ErrorStatisticsCollector) calculateSystemHealth() float64 {
	totalErrors := atomic.LoadInt64(&c.totalErrors)
	totalRecoveries := atomic.LoadInt64(&c.totalRecoveries)
	successfulRecoveries := atomic.LoadInt64(&c.successfulRecoveries)
	
	if totalErrors == 0 {
		return 1.0
	}
	
	// 간단한 건강도 계산
	recoveryRate := float64(successfulRecoveries) / float64(totalErrors)
	
	// 0.0-1.0 범위로 정규화
	if recoveryRate > 1.0 {
		recoveryRate = 1.0
	}
	
	return recoveryRate
}

func (c *ErrorStatisticsCollector) predictErrorRate() float64 {
	// 간단한 선형 예측 (실제로는 더 복잡한 ML 모델 사용)
	recentRates := []float64{
		float64(c.realTimeMetrics.LastMinuteErrors),
		float64(c.realTimeMetrics.LastHourErrors) / 60.0,
		float64(c.realTimeMetrics.Last24HourErrors) / 1440.0,
	}
	
	// 가중 평균
	weights := []float64{0.5, 0.3, 0.2}
	var prediction float64
	
	for i, rate := range recentRates {
		prediction += rate * weights[i]
	}
	
	return prediction
}

func (c *ErrorStatisticsCollector) calculateMTTR() time.Duration {
	c.strategyMutex.RLock()
	defer c.strategyMutex.RUnlock()
	
	var totalTime time.Duration
	var totalAttempts int64
	
	for _, stats := range c.strategyStats {
		totalTime += stats.AverageTime * time.Duration(stats.SuccessfulAttempts)
		totalAttempts += stats.SuccessfulAttempts
	}
	
	if totalAttempts == 0 {
		return 0
	}
	
	return totalTime / time.Duration(totalAttempts)
}

func (c *ErrorStatisticsCollector) calculateMTBF() time.Duration {
	totalErrors := atomic.LoadInt64(&c.totalErrors)
	if totalErrors <= 1 {
		return 0
	}
	
	uptime := time.Since(c.startTime)
	return uptime / time.Duration(totalErrors-1)
}

func (c *ErrorStatisticsCollector) getTopErrorTypes(limit int) []ErrorTypeStats {
	c.errorStatsMutex.RLock()
	defer c.errorStatsMutex.RUnlock()
	
	var stats []ErrorTypeStats
	for _, errorStats := range c.errorStats {
		stats = append(stats, *errorStats)
	}
	
	// 총 에러 수로 정렬
	for i := 0; i < len(stats); i++ {
		for j := i + 1; j < len(stats); j++ {
			if stats[i].TotalCount < stats[j].TotalCount {
				stats[i], stats[j] = stats[j], stats[i]
			}
		}
	}
	
	if len(stats) > limit {
		stats = stats[:limit]
	}
	
	return stats
}

func (c *ErrorStatisticsCollector) getTopStrategies(limit int) []StrategyStats {
	c.strategyMutex.RLock()
	defer c.strategyMutex.RUnlock()
	
	var stats []StrategyStats
	for _, strategyStats := range c.strategyStats {
		stats = append(stats, *strategyStats)
	}
	
	// 성공률로 정렬
	for i := 0; i < len(stats); i++ {
		for j := i + 1; j < len(stats); j++ {
			if stats[i].SuccessRate < stats[j].SuccessRate {
				stats[i], stats[j] = stats[j], stats[i]
			}
		}
	}
	
	if len(stats) > limit {
		stats = stats[:limit]
	}
	
	return stats
}

func (c *ErrorStatisticsCollector) analyzeOverallTrend() (TrendDirection, float64) {
	// 간단한 트렌드 분석 (실제로는 더 정교한 분석 필요)
	recentErrors := c.realTimeMetrics.Last24HourErrors
	olderErrors := c.realTimeMetrics.LastHourErrors * 24 // 추정값
	
	if olderErrors == 0 {
		return TrendUnknown, 0.0
	}
	
	change := float64(recentErrors-olderErrors) / float64(olderErrors)
	
	confidence := 0.7 // 임시값
	
	switch {
	case change < -0.2:
		return TrendDecreasing, confidence
	case change > 0.2:
		return TrendIncreasing, confidence
	case change > 1.0:
		return TrendSpike, confidence
	case change < -0.8:
		return TrendDrop, confidence
	default:
		return TrendStable, confidence
	}
}