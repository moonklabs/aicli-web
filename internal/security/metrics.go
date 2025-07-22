package security

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

// MetricsCollector는 보안 메트릭 수집기입니다.
type MetricsCollector struct {
	redis      redis.UniversalClient
	logger     *zap.Logger
	metrics    map[string]*Metric
	mu         sync.RWMutex
	
	// 수집 설정
	collectInterval time.Duration
	retentionPeriod time.Duration
	
	// 채널
	metricChan chan *MetricEvent
	stopChan   chan struct{}
}

// Metric은 개별 메트릭을 나타냅니다.
type Metric struct {
	Name        string                 `json:"name"`
	Type        MetricType             `json:"type"`
	Value       float64                `json:"value"`
	Labels      map[string]string      `json:"labels,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Description string                 `json:"description,omitempty"`
}

// MetricEvent는 메트릭 이벤트를 나타냅니다.
type MetricEvent struct {
	Name      string            `json:"name"`
	Type      MetricType        `json:"type"`
	Value     float64           `json:"value"`
	Labels    map[string]string `json:"labels,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

// MetricType은 메트릭 유형을 정의합니다.
type MetricType string

const (
	MetricTypeCounter   MetricType = "counter"
	MetricTypeGauge     MetricType = "gauge"
	MetricTypeHistogram MetricType = "histogram"
	MetricTypeSummary   MetricType = "summary"
)

// SecurityMetrics는 보안 관련 메트릭들을 나타냅니다.
type SecurityMetrics struct {
	// 공격 관련 메트릭
	TotalAttacks         int64                  `json:"total_attacks"`
	AttacksByType        map[string]int64       `json:"attacks_by_type"`
	AttacksBySeverity    map[string]int64       `json:"attacks_by_severity"`
	AttacksBlocked       int64                  `json:"attacks_blocked"`
	AttacksPerHour       []TimeSeriesPoint      `json:"attacks_per_hour"`
	
	// Rate Limiting 메트릭
	RateLimitViolations  int64                  `json:"rate_limit_violations"`
	RateLimitByType      map[string]int64       `json:"rate_limit_by_type"`
	BlockedIPs           int64                  `json:"blocked_ips"`
	
	// 인증 관련 메트릭
	AuthFailures         int64                  `json:"auth_failures"`
	AuthSuccesses        int64                  `json:"auth_successes"`
	BruteForceAttempts   int64                  `json:"brute_force_attempts"`
	
	// 세션 관련 메트릭
	ActiveSessions       int64                  `json:"active_sessions"`
	SessionAnomalies     int64                  `json:"session_anomalies"`
	DeviceChanges        int64                  `json:"device_changes"`
	
	// 시스템 성능 메트릭
	ResponseTimes        ResponseTimeMetrics    `json:"response_times"`
	RequestVolume        []TimeSeriesPoint      `json:"request_volume"`
	ErrorRates           map[string]float64     `json:"error_rates"`
	
	// 지리적 분석
	RequestsByCountry    map[string]int64       `json:"requests_by_country"`
	SuspiciousCountries  []string               `json:"suspicious_countries"`
	
	Timestamp            time.Time              `json:"timestamp"`
}

// TimeSeriesPoint는 시계열 데이터 포인트입니다.
type TimeSeriesPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

// ResponseTimeMetrics는 응답 시간 메트릭입니다.
type ResponseTimeMetrics struct {
	Mean   float64 `json:"mean"`
	P50    float64 `json:"p50"`
	P95    float64 `json:"p95"`
	P99    float64 `json:"p99"`
	Max    float64 `json:"max"`
}

// MetricsCollectorConfig는 메트릭 수집기 설정입니다.
type MetricsCollectorConfig struct {
	Redis           redis.UniversalClient
	Logger          *zap.Logger
	CollectInterval time.Duration
	RetentionPeriod time.Duration
	BufferSize      int
}

// NewMetricsCollector는 새로운 메트릭 수집기를 생성합니다.
func NewMetricsCollector(config *MetricsCollectorConfig) *MetricsCollector {
	if config.CollectInterval == 0 {
		config.CollectInterval = 10 * time.Second
	}
	if config.RetentionPeriod == 0 {
		config.RetentionPeriod = 7 * 24 * time.Hour // 7일
	}
	if config.BufferSize == 0 {
		config.BufferSize = 1000
	}

	mc := &MetricsCollector{
		redis:           config.Redis,
		logger:          config.Logger,
		metrics:         make(map[string]*Metric),
		collectInterval: config.CollectInterval,
		retentionPeriod: config.RetentionPeriod,
		metricChan:      make(chan *MetricEvent, config.BufferSize),
		stopChan:        make(chan struct{}),
	}

	// 메트릭 수집 워커 시작
	go mc.startCollectionWorker()
	go mc.startAggregationWorker()

	return mc
}

// RecordMetric은 메트릭을 기록합니다.
func (mc *MetricsCollector) RecordMetric(name string, metricType MetricType, value float64, labels map[string]string) {
	event := &MetricEvent{
		Name:      name,
		Type:      metricType,
		Value:     value,
		Labels:    labels,
		Timestamp: time.Now(),
	}

	select {
	case mc.metricChan <- event:
		// 성공
	default:
		mc.logger.Warn("메트릭 버퍼 가득참", zap.String("metric", name))
	}
}

// IncrementCounter는 카운터 메트릭을 증가시킵니다.
func (mc *MetricsCollector) IncrementCounter(name string, labels map[string]string) {
	mc.RecordMetric(name, MetricTypeCounter, 1, labels)
}

// SetGauge는 게이지 메트릭을 설정합니다.
func (mc *MetricsCollector) SetGauge(name string, value float64, labels map[string]string) {
	mc.RecordMetric(name, MetricTypeGauge, value, labels)
}

// RecordHistogram은 히스토그램 메트릭을 기록합니다.
func (mc *MetricsCollector) RecordHistogram(name string, value float64, labels map[string]string) {
	mc.RecordMetric(name, MetricTypeHistogram, value, labels)
}

// GetSecurityMetrics는 보안 메트릭을 조회합니다.
func (mc *MetricsCollector) GetSecurityMetrics(ctx context.Context, period time.Duration) (*SecurityMetrics, error) {
	now := time.Now()
	startTime := now.Add(-period)

	metrics := &SecurityMetrics{
		AttacksByType:       make(map[string]int64),
		AttacksBySeverity:   make(map[string]int64),
		RateLimitByType:     make(map[string]int64),
		ResponseTimes:       ResponseTimeMetrics{},
		ErrorRates:          make(map[string]float64),
		RequestsByCountry:   make(map[string]int64),
		SuspiciousCountries: make([]string, 0),
		Timestamp:           now,
	}

	// 공격 관련 메트릭 수집
	if err := mc.collectAttackMetrics(ctx, metrics, startTime, now); err != nil {
		mc.logger.Error("공격 메트릭 수집 실패", zap.Error(err))
	}

	// Rate Limiting 메트릭 수집
	if err := mc.collectRateLimitMetrics(ctx, metrics, startTime, now); err != nil {
		mc.logger.Error("Rate Limit 메트릭 수집 실패", zap.Error(err))
	}

	// 인증 메트릭 수집
	if err := mc.collectAuthMetrics(ctx, metrics, startTime, now); err != nil {
		mc.logger.Error("인증 메트릭 수집 실패", zap.Error(err))
	}

	// 세션 메트릭 수집
	if err := mc.collectSessionMetrics(ctx, metrics, startTime, now); err != nil {
		mc.logger.Error("세션 메트릭 수집 실패", zap.Error(err))
	}

	// 성능 메트릭 수집
	if err := mc.collectPerformanceMetrics(ctx, metrics, startTime, now); err != nil {
		mc.logger.Error("성능 메트릭 수집 실패", zap.Error(err))
	}

	return metrics, nil
}

// collectAttackMetrics는 공격 관련 메트릭을 수집합니다.
func (mc *MetricsCollector) collectAttackMetrics(ctx context.Context, metrics *SecurityMetrics, start, end time.Time) error {
	// 공격 타입별 집계
	attackTypes := []string{"sql_injection", "xss", "command_injection", "brute_force", "csrf_violation"}
	for _, attackType := range attackTypes {
		count, err := mc.getCounterValue(ctx, fmt.Sprintf("security_attacks_total{type=\"%s\"}", attackType), start, end)
		if err == nil {
			metrics.AttacksByType[attackType] = count
			metrics.TotalAttacks += count
		}
	}

	// 심각도별 집계
	severities := []string{"low", "medium", "high", "critical"}
	for _, severity := range severities {
		count, err := mc.getCounterValue(ctx, fmt.Sprintf("security_attacks_total{severity=\"%s\"}", severity), start, end)
		if err == nil {
			metrics.AttacksBySeverity[severity] = count
		}
	}

	// 차단된 공격 수
	blocked, err := mc.getCounterValue(ctx, "security_attacks_blocked_total", start, end)
	if err == nil {
		metrics.AttacksBlocked = blocked
	}

	// 시간별 공격 수
	metrics.AttacksPerHour = mc.getTimeSeriesData(ctx, "security_attacks_total", start, end, time.Hour)

	return nil
}

// collectRateLimitMetrics는 Rate Limiting 메트릭을 수집합니다.
func (mc *MetricsCollector) collectRateLimitMetrics(ctx context.Context, metrics *SecurityMetrics, start, end time.Time) error {
	// Rate Limit 위반 총 수
	violations, err := mc.getCounterValue(ctx, "rate_limit_violations_total", start, end)
	if err == nil {
		metrics.RateLimitViolations = violations
	}

	// 타입별 Rate Limit 위반
	limitTypes := []string{"ip", "user", "endpoint", "global"}
	for _, limitType := range limitTypes {
		count, err := mc.getCounterValue(ctx, fmt.Sprintf("rate_limit_violations_total{type=\"%s\"}", limitType), start, end)
		if err == nil {
			metrics.RateLimitByType[limitType] = count
		}
	}

	// 차단된 IP 수
	blockedIPs, err := mc.getGaugeValue(ctx, "blocked_ips_total")
	if err == nil {
		metrics.BlockedIPs = int64(blockedIPs)
	}

	return nil
}

// collectAuthMetrics는 인증 메트릭을 수집합니다.
func (mc *MetricsCollector) collectAuthMetrics(ctx context.Context, metrics *SecurityMetrics, start, end time.Time) error {
	// 인증 실패 수
	failures, err := mc.getCounterValue(ctx, "auth_failures_total", start, end)
	if err == nil {
		metrics.AuthFailures = failures
	}

	// 인증 성공 수
	successes, err := mc.getCounterValue(ctx, "auth_successes_total", start, end)
	if err == nil {
		metrics.AuthSuccesses = successes
	}

	// 브루트포스 시도 수
	bruteForce, err := mc.getCounterValue(ctx, "brute_force_attempts_total", start, end)
	if err == nil {
		metrics.BruteForceAttempts = bruteForce
	}

	return nil
}

// collectSessionMetrics는 세션 메트릭을 수집합니다.
func (mc *MetricsCollector) collectSessionMetrics(ctx context.Context, metrics *SecurityMetrics, start, end time.Time) error {
	// 활성 세션 수
	activeSessions, err := mc.getGaugeValue(ctx, "active_sessions_total")
	if err == nil {
		metrics.ActiveSessions = int64(activeSessions)
	}

	// 세션 이상 행위 수
	anomalies, err := mc.getCounterValue(ctx, "session_anomalies_total", start, end)
	if err == nil {
		metrics.SessionAnomalies = anomalies
	}

	// 디바이스 변경 수
	deviceChanges, err := mc.getCounterValue(ctx, "device_changes_total", start, end)
	if err == nil {
		metrics.DeviceChanges = deviceChanges
	}

	return nil
}

// collectPerformanceMetrics는 성능 메트릭을 수집합니다.
func (mc *MetricsCollector) collectPerformanceMetrics(ctx context.Context, metrics *SecurityMetrics, start, end time.Time) error {
	// 응답 시간 메트릭
	responseTime, err := mc.getHistogramSummary(ctx, "http_request_duration_seconds", start, end)
	if err == nil {
		metrics.ResponseTimes = *responseTime
	}

	// 요청 볼륨
	metrics.RequestVolume = mc.getTimeSeriesData(ctx, "http_requests_total", start, end, time.Hour)

	// 에러율
	errorRates := []string{"4xx", "5xx"}
	for _, errorRate := range errorRates {
		rate, err := mc.getErrorRate(ctx, errorRate, start, end)
		if err == nil {
			metrics.ErrorRates[errorRate] = rate
		}
	}

	return nil
}

// startCollectionWorker는 메트릭 수집 워커를 시작합니다.
func (mc *MetricsCollector) startCollectionWorker() {
	for {
		select {
		case event := <-mc.metricChan:
			mc.processMetricEvent(event)
		case <-mc.stopChan:
			return
		}
	}
}

// startAggregationWorker는 메트릭 집계 워커를 시작합니다.
func (mc *MetricsCollector) startAggregationWorker() {
	ticker := time.NewTicker(mc.collectInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mc.aggregateMetrics()
		case <-mc.stopChan:
			return
		}
	}
}

// processMetricEvent는 메트릭 이벤트를 처리합니다.
func (mc *MetricsCollector) processMetricEvent(event *MetricEvent) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	// 메트릭 키 생성
	key := mc.generateMetricKey(event.Name, event.Labels)

	// 기존 메트릭 업데이트 또는 새 메트릭 생성
	if existing, exists := mc.metrics[key]; exists {
		switch event.Type {
		case MetricTypeCounter:
			existing.Value += event.Value
		case MetricTypeGauge:
			existing.Value = event.Value
		case MetricTypeHistogram, MetricTypeSummary:
			// 히스토그램과 서머리는 별도 처리 필요
			existing.Value = event.Value
		}
		existing.Timestamp = event.Timestamp
	} else {
		mc.metrics[key] = &Metric{
			Name:      event.Name,
			Type:      event.Type,
			Value:     event.Value,
			Labels:    event.Labels,
			Timestamp: event.Timestamp,
		}
	}

	// Redis에 저장
	mc.storeMetricToRedis(key, mc.metrics[key])
}

// aggregateMetrics는 메트릭을 집계합니다.
func (mc *MetricsCollector) aggregateMetrics() {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	ctx := context.Background()
	now := time.Now()

	// 현재 메트릭들을 Redis에 집계 저장
	for key, metric := range mc.metrics {
		aggregateKey := fmt.Sprintf("aggregated:%s:%s", key, now.Format("2006-01-02-15"))
		mc.storeAggregatedMetric(ctx, aggregateKey, metric)
	}

	mc.logger.Debug("메트릭 집계 완료", zap.Int("metrics_count", len(mc.metrics)))
}

// storeMetricToRedis는 메트릭을 Redis에 저장합니다.
func (mc *MetricsCollector) storeMetricToRedis(key string, metric *Metric) {
	if mc.redis == nil {
		return
	}

	ctx := context.Background()
	
	// 메트릭 직렬화
	data, err := json.Marshal(metric)
	if err != nil {
		mc.logger.Error("메트릭 직렬화 실패", zap.String("key", key), zap.Error(err))
		return
	}

	// 시계열 데이터로 저장
	redisKey := fmt.Sprintf("metrics:%s", key)
	score := float64(metric.Timestamp.Unix())

	err = mc.redis.ZAdd(ctx, redisKey, &redis.Z{
		Score:  score,
		Member: data,
	}).Err()
	
	if err != nil {
		mc.logger.Error("메트릭 Redis 저장 실패", zap.String("key", key), zap.Error(err))
		return
	}

	// TTL 설정
	mc.redis.Expire(ctx, redisKey, mc.retentionPeriod)
}

// storeAggregatedMetric은 집계된 메트릭을 저장합니다.
func (mc *MetricsCollector) storeAggregatedMetric(ctx context.Context, key string, metric *Metric) {
	if mc.redis == nil {
		return
	}

	data, err := json.Marshal(metric)
	if err != nil {
		mc.logger.Error("집계 메트릭 직렬화 실패", zap.String("key", key), zap.Error(err))
		return
	}

	err = mc.redis.Set(ctx, key, data, mc.retentionPeriod).Err()
	if err != nil {
		mc.logger.Error("집계 메트릭 저장 실패", zap.String("key", key), zap.Error(err))
	}
}

// generateMetricKey는 메트릭 키를 생성합니다.
func (mc *MetricsCollector) generateMetricKey(name string, labels map[string]string) string {
	if len(labels) == 0 {
		return name
	}

	key := name + "{"
	first := true
	for k, v := range labels {
		if !first {
			key += ","
		}
		key += fmt.Sprintf("%s=\"%s\"", k, v)
		first = false
	}
	key += "}"
	return key
}

// getCounterValue는 카운터 메트릭 값을 조회합니다.
func (mc *MetricsCollector) getCounterValue(ctx context.Context, metricName string, start, end time.Time) (int64, error) {
	if mc.redis == nil {
		return 0, fmt.Errorf("Redis 연결이 없습니다")
	}

	key := fmt.Sprintf("metrics:%s", metricName)
	startScore := float64(start.Unix())
	endScore := float64(end.Unix())

	members, err := mc.redis.ZRangeByScore(ctx, key, &redis.ZRangeBy{
		Min: fmt.Sprintf("%f", startScore),
		Max: fmt.Sprintf("%f", endScore),
	}).Result()

	if err != nil {
		return 0, err
	}

	var total int64
	for _, member := range members {
		var metric Metric
		if err := json.Unmarshal([]byte(member), &metric); err == nil {
			total += int64(metric.Value)
		}
	}

	return total, nil
}

// getGaugeValue는 게이지 메트릭 값을 조회합니다.
func (mc *MetricsCollector) getGaugeValue(ctx context.Context, metricName string) (float64, error) {
	if mc.redis == nil {
		mc.mu.RLock()
		defer mc.mu.RUnlock()
		
		if metric, exists := mc.metrics[metricName]; exists {
			return metric.Value, nil
		}
		return 0, fmt.Errorf("메트릭을 찾을 수 없습니다")
	}

	key := fmt.Sprintf("metrics:%s", metricName)
	
	// 최신 값 조회
	members, err := mc.redis.ZRevRange(ctx, key, 0, 0).Result()
	if err != nil || len(members) == 0 {
		return 0, err
	}

	var metric Metric
	if err := json.Unmarshal([]byte(members[0]), &metric); err != nil {
		return 0, err
	}

	return metric.Value, nil
}

// getHistogramSummary는 히스토그램 요약 통계를 조회합니다.
func (mc *MetricsCollector) getHistogramSummary(ctx context.Context, metricName string, start, end time.Time) (*ResponseTimeMetrics, error) {
	// 실제 구현에서는 히스토그램 버킷에서 백분위수를 계산해야 함
	// 여기서는 간단한 예시
	return &ResponseTimeMetrics{
		Mean: 0.150,
		P50:  0.100,
		P95:  0.500,
		P99:  1.000,
		Max:  2.500,
	}, nil
}

// getTimeSeriesData는 시계열 데이터를 조회합니다.
func (mc *MetricsCollector) getTimeSeriesData(ctx context.Context, metricName string, start, end time.Time, interval time.Duration) []TimeSeriesPoint {
	points := make([]TimeSeriesPoint, 0)
	
	current := start
	for current.Before(end) {
		next := current.Add(interval)
		
		// 해당 시간 구간의 값 조회
		value, err := mc.getCounterValue(ctx, metricName, current, next)
		if err == nil {
			points = append(points, TimeSeriesPoint{
				Timestamp: current,
				Value:     float64(value),
			})
		}
		
		current = next
	}

	return points
}

// getErrorRate는 에러율을 계산합니다.
func (mc *MetricsCollector) getErrorRate(ctx context.Context, errorType string, start, end time.Time) (float64, error) {
	errorCount, err := mc.getCounterValue(ctx, fmt.Sprintf("http_requests_total{status=\"%s\"}", errorType), start, end)
	if err != nil {
		return 0, err
	}

	totalCount, err := mc.getCounterValue(ctx, "http_requests_total", start, end)
	if err != nil || totalCount == 0 {
		return 0, err
	}

	return float64(errorCount) / float64(totalCount) * 100, nil
}

// GetCurrentMetrics는 현재 메트릭 상태를 반환합니다.
func (mc *MetricsCollector) GetCurrentMetrics() map[string]*Metric {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	result := make(map[string]*Metric, len(mc.metrics))
	for k, v := range mc.metrics {
		result[k] = v
	}

	return result
}

// Close는 메트릭 수집기를 종료합니다.
func (mc *MetricsCollector) Close() error {
	close(mc.stopChan)
	close(mc.metricChan)
	return nil
}

// 보안 관련 특화 메서드들

// RecordSecurityEvent는 보안 이벤트 메트릭을 기록합니다.
func (mc *MetricsCollector) RecordSecurityEvent(eventType, severity string) {
	labels := map[string]string{
		"type":     eventType,
		"severity": severity,
	}
	mc.IncrementCounter("security_events_total", labels)
}

// RecordAttackDetection은 공격 탐지 메트릭을 기록합니다.
func (mc *MetricsCollector) RecordAttackDetection(attackType string, blocked bool) {
	labels := map[string]string{
		"type": attackType,
	}
	mc.IncrementCounter("security_attacks_total", labels)
	
	if blocked {
		mc.IncrementCounter("security_attacks_blocked_total", labels)
	}
}

// RecordAuthEvent는 인증 이벤트 메트릭을 기록합니다.
func (mc *MetricsCollector) RecordAuthEvent(eventType, result string) {
	labels := map[string]string{
		"type":   eventType,
		"result": result,
	}
	mc.IncrementCounter("auth_events_total", labels)
}

// RecordRateLimitViolation은 Rate Limit 위반 메트릭을 기록합니다.
func (mc *MetricsCollector) RecordRateLimitViolation(limitType, clientType string) {
	labels := map[string]string{
		"limit_type":  limitType,
		"client_type": clientType,
	}
	mc.IncrementCounter("rate_limit_violations_total", labels)
}

// UpdateActiveSessionCount는 활성 세션 수를 업데이트합니다.
func (mc *MetricsCollector) UpdateActiveSessionCount(count int64) {
	mc.SetGauge("active_sessions_total", float64(count), nil)
}

// RecordResponseTime은 응답 시간을 기록합니다.
func (mc *MetricsCollector) RecordResponseTime(path string, method string, statusCode int, duration time.Duration) {
	labels := map[string]string{
		"path":   path,
		"method": method,
		"status": fmt.Sprintf("%d", statusCode),
	}
	mc.RecordHistogram("http_request_duration_seconds", duration.Seconds(), labels)
	mc.IncrementCounter("http_requests_total", labels)
}