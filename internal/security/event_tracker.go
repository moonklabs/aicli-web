package security

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

// EventType은 보안 이벤트 타입을 정의합니다.
type EventType string

const (
	EventTypeRateLimitExceeded EventType = "rate_limit_exceeded"
	EventTypeCSRFViolation     EventType = "csrf_violation"
	EventTypeAuthFailure       EventType = "auth_failure"
	EventTypeSessionAnomaly    EventType = "session_anomaly"
	EventTypeDeviceChange      EventType = "device_change"
	EventTypeLocationChange    EventType = "location_change"
	EventTypeSuspiciousActivity EventType = "suspicious_activity"
	EventTypeIPBlocked         EventType = "ip_blocked"
	EventTypeAttackPattern     EventType = "attack_pattern"
	EventTypeBruteForce        EventType = "brute_force"
	EventTypePrivilegeEscalation EventType = "privilege_escalation"
	EventTypeMaliciousRequest  EventType = "malicious_request"
)

// Severity는 이벤트 심각도를 정의합니다.
type Severity string

const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

// SecurityEvent는 보안 이벤트를 나타냅니다.
type SecurityEvent struct {
	ID          string                 `json:"id" redis:"id"`
	Type        EventType              `json:"type" redis:"type"`
	Severity    Severity               `json:"severity" redis:"severity"`
	Source      string                 `json:"source" redis:"source"`
	Target      string                 `json:"target" redis:"target"`
	Details     map[string]interface{} `json:"details" redis:"details"`
	Timestamp   time.Time              `json:"timestamp" redis:"timestamp"`
	UserID      string                 `json:"user_id,omitempty" redis:"user_id"`
	SessionID   string                 `json:"session_id,omitempty" redis:"session_id"`
	IPAddress   string                 `json:"ip_address" redis:"ip_address"`
	UserAgent   string                 `json:"user_agent" redis:"user_agent"`
	RequestPath string                 `json:"request_path,omitempty" redis:"request_path"`
	Method      string                 `json:"method,omitempty" redis:"method"`
	StatusCode  int                    `json:"status_code,omitempty" redis:"status_code"`
	Resolved    bool                   `json:"resolved" redis:"resolved"`
	ResolvedAt  *time.Time             `json:"resolved_at,omitempty" redis:"resolved_at"`
}

// EventTrackerConfig는 이벤트 추적기 설정입니다.
type EventTrackerConfig struct {
	Redis           redis.UniversalClient
	RetentionPeriod time.Duration         // 이벤트 보관 기간
	MaxEvents       int                   // 최대 이벤트 수
	AlertThresholds map[EventType]int     // 알림 임계값
	EnableAlerts    bool                  // 알림 활성화
	Logger          *zap.Logger
}

// EventTracker는 보안 이벤트 추적기입니다.
type EventTracker struct {
	config *EventTrackerConfig
	redis  redis.UniversalClient
	logger *zap.Logger
}

// EventFilter는 이벤트 필터링을 위한 구조체입니다.
type EventFilter struct {
	Types      []EventType `json:"types,omitempty"`
	Severities []Severity  `json:"severities,omitempty"`
	UserID     string      `json:"user_id,omitempty"`
	IPAddress  string      `json:"ip_address,omitempty"`
	Source     string      `json:"source,omitempty"`
	Target     string      `json:"target,omitempty"`
	StartTime  *time.Time  `json:"start_time,omitempty"`
	EndTime    *time.Time  `json:"end_time,omitempty"`
	Resolved   *bool       `json:"resolved,omitempty"`
	Limit      int         `json:"limit,omitempty"`
	Offset     int         `json:"offset,omitempty"`
}

// EventStatistics는 이벤트 통계를 나타냅니다.
type EventStatistics struct {
	TotalEvents       int                    `json:"total_events"`
	EventsByType      map[EventType]int      `json:"events_by_type"`
	EventsBySeverity  map[Severity]int       `json:"events_by_severity"`
	EventsByHour      map[string]int         `json:"events_by_hour"`
	TopTargets        []TargetStat           `json:"top_targets"`
	TopSources        []SourceStat           `json:"top_sources"`
	RecentTrends      []TrendData            `json:"recent_trends"`
	UnresolvedEvents  int                    `json:"unresolved_events"`
}

// TargetStat는 타겟 통계입니다.
type TargetStat struct {
	Target string `json:"target"`
	Count  int    `json:"count"`
}

// SourceStat는 소스 통계입니다.
type SourceStat struct {
	Source string `json:"source"`
	Count  int    `json:"count"`
}

// TrendData는 트렌드 데이터입니다.
type TrendData struct {
	Period string `json:"period"`
	Count  int    `json:"count"`
}

// NewEventTracker는 새로운 이벤트 추적기를 생성합니다.
func NewEventTracker(config *EventTrackerConfig) *EventTracker {
	if config.RetentionPeriod == 0 {
		config.RetentionPeriod = 30 * 24 * time.Hour // 30일
	}
	if config.MaxEvents == 0 {
		config.MaxEvents = 100000
	}
	if config.AlertThresholds == nil {
		config.AlertThresholds = map[EventType]int{
			EventTypeRateLimitExceeded: 100,
			EventTypeAuthFailure:       50,
			EventTypeBruteForce:        10,
			EventTypeAttackPattern:     5,
		}
	}

	tracker := &EventTracker{
		config: config,
		redis:  config.Redis,
		logger: config.Logger,
	}

	// 정리 작업 시작
	go tracker.startCleanupWorker()

	return tracker
}

// RecordEvent는 보안 이벤트를 기록합니다.
func (et *EventTracker) RecordEvent(ctx context.Context, event *SecurityEvent) error {
	// 기본값 설정
	if event.ID == "" {
		event.ID = et.generateEventID()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// 이벤트 직렬화
	eventData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("이벤트 직렬화 실패: %w", err)
	}

	// Redis에 저장
	pipe := et.redis.Pipeline()

	// 개별 이벤트 저장
	eventKey := et.getEventKey(event.ID)
	pipe.Set(ctx, eventKey, eventData, et.config.RetentionPeriod)

	// 시계열 데이터에 추가
	timeSeriesKey := et.getTimeSeriesKey(event.Type)
	pipe.ZAdd(ctx, timeSeriesKey, &redis.Z{
		Score:  float64(event.Timestamp.Unix()),
		Member: event.ID,
	})
	pipe.Expire(ctx, timeSeriesKey, et.config.RetentionPeriod)

	// 심각도별 인덱스에 추가
	severityKey := et.getSeverityKey(event.Severity)
	pipe.ZAdd(ctx, severityKey, &redis.Z{
		Score:  float64(event.Timestamp.Unix()),
		Member: event.ID,
	})
	pipe.Expire(ctx, severityKey, et.config.RetentionPeriod)

	// 사용자별 인덱스에 추가 (사용자 ID가 있는 경우)
	if event.UserID != "" {
		userKey := et.getUserKey(event.UserID)
		pipe.ZAdd(ctx, userKey, &redis.Z{
			Score:  float64(event.Timestamp.Unix()),
			Member: event.ID,
		})
		pipe.Expire(ctx, userKey, et.config.RetentionPeriod)
	}

	// IP별 인덱스에 추가
	if event.IPAddress != "" {
		ipKey := et.getIPKey(event.IPAddress)
		pipe.ZAdd(ctx, ipKey, &redis.Z{
			Score:  float64(event.Timestamp.Unix()),
			Member: event.ID,
		})
		pipe.Expire(ctx, ipKey, et.config.RetentionPeriod)
	}

	// 파이프라인 실행
	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("이벤트 저장 실패: %w", err)
	}

	// 통계 업데이트
	et.updateStatistics(ctx, event)

	// 알림 확인
	if et.config.EnableAlerts {
		et.checkAlerts(ctx, event)
	}

	et.logger.Debug("보안 이벤트 기록됨",
		zap.String("event_id", event.ID),
		zap.String("type", string(event.Type)),
		zap.String("severity", string(event.Severity)),
		zap.String("source", event.Source))

	return nil
}

// GetEvent는 특정 이벤트를 조회합니다.
func (et *EventTracker) GetEvent(ctx context.Context, eventID string) (*SecurityEvent, error) {
	eventKey := et.getEventKey(eventID)
	
	data, err := et.redis.Get(ctx, eventKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("이벤트를 찾을 수 없음: %s", eventID)
		}
		return nil, fmt.Errorf("이벤트 조회 실패: %w", err)
	}

	var event SecurityEvent
	err = json.Unmarshal([]byte(data), &event)
	if err != nil {
		return nil, fmt.Errorf("이벤트 역직렬화 실패: %w", err)
	}

	return &event, nil
}

// QueryEvents는 필터 조건에 맞는 이벤트들을 조회합니다.
func (et *EventTracker) QueryEvents(ctx context.Context, filter *EventFilter) ([]*SecurityEvent, error) {
	if filter.Limit == 0 {
		filter.Limit = 100
	}

	var eventIDs []string
	var err error

	// 필터 조건에 따라 이벤트 ID 조회
	if len(filter.Types) == 1 {
		// 단일 타입 조회
		eventIDs, err = et.getEventIDsByType(ctx, filter.Types[0], filter)
	} else if len(filter.Severities) == 1 {
		// 단일 심각도 조회
		eventIDs, err = et.getEventIDsBySeverity(ctx, filter.Severities[0], filter)
	} else if filter.UserID != "" {
		// 사용자별 조회
		eventIDs, err = et.getEventIDsByUser(ctx, filter.UserID, filter)
	} else if filter.IPAddress != "" {
		// IP별 조회
		eventIDs, err = et.getEventIDsByIP(ctx, filter.IPAddress, filter)
	} else {
		// 전체 조회 (시간 범위 기반)
		eventIDs, err = et.getAllEventIDs(ctx, filter)
	}

	if err != nil {
		return nil, err
	}

	// 이벤트 데이터 조회
	events := make([]*SecurityEvent, 0, len(eventIDs))
	for _, eventID := range eventIDs {
		event, err := et.GetEvent(ctx, eventID)
		if err != nil {
			et.logger.Warn("이벤트 조회 실패", zap.String("event_id", eventID), zap.Error(err))
			continue
		}

		// 추가 필터링 적용
		if et.matchesFilter(event, filter) {
			events = append(events, event)
		}
	}

	return events, nil
}

// GetStatistics는 이벤트 통계를 반환합니다.
func (et *EventTracker) GetStatistics(ctx context.Context, period time.Duration) (*EventStatistics, error) {
	stats := &EventStatistics{
		EventsByType:     make(map[EventType]int),
		EventsBySeverity: make(map[Severity]int),
		EventsByHour:     make(map[string]int),
		TopTargets:       make([]TargetStat, 0),
		TopSources:       make([]SourceStat, 0),
		RecentTrends:     make([]TrendData, 0),
	}

	now := time.Now()
	startTime := now.Add(-period)

	// 타입별 통계
	for eventType := range et.config.AlertThresholds {
		count, err := et.getEventCountByType(ctx, eventType, startTime, now)
		if err == nil {
			stats.EventsByType[eventType] = count
			stats.TotalEvents += count
		}
	}

	// 심각도별 통계
	severities := []Severity{SeverityLow, SeverityMedium, SeverityHigh, SeverityCritical}
	for _, severity := range severities {
		count, err := et.getEventCountBySeverity(ctx, severity, startTime, now)
		if err == nil {
			stats.EventsBySeverity[severity] = count
		}
	}

	// 시간별 통계 (24시간)
	for i := 0; i < 24; i++ {
		hourStart := now.Add(time.Duration(-i) * time.Hour).Truncate(time.Hour)
		hourEnd := hourStart.Add(time.Hour)
		
		count, err := et.getEventCountInTimeRange(ctx, hourStart, hourEnd)
		if err == nil {
			stats.EventsByHour[hourStart.Format("2006-01-02 15:04")] = count
		}
	}

	return stats, nil
}

// ResolveEvent는 이벤트를 해결됨으로 표시합니다.
func (et *EventTracker) ResolveEvent(ctx context.Context, eventID string) error {
	event, err := et.GetEvent(ctx, eventID)
	if err != nil {
		return err
	}

	event.Resolved = true
	now := time.Now()
	event.ResolvedAt = &now

	// 업데이트된 이벤트 저장
	eventData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("이벤트 직렬화 실패: %w", err)
	}

	eventKey := et.getEventKey(eventID)
	err = et.redis.Set(ctx, eventKey, eventData, et.config.RetentionPeriod).Err()
	if err != nil {
		return fmt.Errorf("이벤트 업데이트 실패: %w", err)
	}

	et.logger.Info("보안 이벤트 해결됨",
		zap.String("event_id", eventID),
		zap.String("type", string(event.Type)))

	return nil
}

// 내부 헬퍼 메서드들

func (et *EventTracker) generateEventID() string {
	return fmt.Sprintf("evt_%d_%d", time.Now().UnixNano(), time.Now().Unix())
}

func (et *EventTracker) getEventKey(eventID string) string {
	return fmt.Sprintf("security:event:%s", eventID)
}

func (et *EventTracker) getTimeSeriesKey(eventType EventType) string {
	return fmt.Sprintf("security:events:type:%s", eventType)
}

func (et *EventTracker) getSeverityKey(severity Severity) string {
	return fmt.Sprintf("security:events:severity:%s", severity)
}

func (et *EventTracker) getUserKey(userID string) string {
	return fmt.Sprintf("security:events:user:%s", userID)
}

func (et *EventTracker) getIPKey(ipAddress string) string {
	return fmt.Sprintf("security:events:ip:%s", ipAddress)
}

func (et *EventTracker) getEventIDsByType(ctx context.Context, eventType EventType, filter *EventFilter) ([]string, error) {
	key := et.getTimeSeriesKey(eventType)
	return et.getEventIDsFromSortedSet(ctx, key, filter)
}

func (et *EventTracker) getEventIDsBySeverity(ctx context.Context, severity Severity, filter *EventFilter) ([]string, error) {
	key := et.getSeverityKey(severity)
	return et.getEventIDsFromSortedSet(ctx, key, filter)
}

func (et *EventTracker) getEventIDsByUser(ctx context.Context, userID string, filter *EventFilter) ([]string, error) {
	key := et.getUserKey(userID)
	return et.getEventIDsFromSortedSet(ctx, key, filter)
}

func (et *EventTracker) getEventIDsByIP(ctx context.Context, ipAddress string, filter *EventFilter) ([]string, error) {
	key := et.getIPKey(ipAddress)
	return et.getEventIDsFromSortedSet(ctx, key, filter)
}

func (et *EventTracker) getAllEventIDs(ctx context.Context, filter *EventFilter) ([]string, error) {
	// 전체 이벤트 조회는 메모리 사용량이 클 수 있으므로 제한적으로 구현
	return []string{}, nil
}

func (et *EventTracker) getEventIDsFromSortedSet(ctx context.Context, key string, filter *EventFilter) ([]string, error) {
	var minScore, maxScore string
	
	if filter.StartTime != nil {
		minScore = fmt.Sprintf("%d", filter.StartTime.Unix())
	} else {
		minScore = "-inf"
	}
	
	if filter.EndTime != nil {
		maxScore = fmt.Sprintf("%d", filter.EndTime.Unix())
	} else {
		maxScore = "+inf"
	}

	// ZRevRangeByScore로 최신순 조회
	eventIDs, err := et.redis.ZRevRangeByScore(ctx, key, &redis.ZRangeBy{
		Min:    minScore,
		Max:    maxScore,
		Offset: int64(filter.Offset),
		Count:  int64(filter.Limit),
	}).Result()

	return eventIDs, err
}

func (et *EventTracker) matchesFilter(event *SecurityEvent, filter *EventFilter) bool {
	// 타입 필터링
	if len(filter.Types) > 0 {
		matched := false
		for _, t := range filter.Types {
			if event.Type == t {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// 심각도 필터링
	if len(filter.Severities) > 0 {
		matched := false
		for _, s := range filter.Severities {
			if event.Severity == s {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// 해결 상태 필터링
	if filter.Resolved != nil {
		if event.Resolved != *filter.Resolved {
			return false
		}
	}

	return true
}

func (et *EventTracker) updateStatistics(ctx context.Context, event *SecurityEvent) {
	// 통계 카운터 업데이트
	pipe := et.redis.Pipeline()
	
	// 일별 통계
	dateKey := fmt.Sprintf("security:stats:daily:%s", time.Now().Format("2006-01-02"))
	pipe.HIncrBy(ctx, dateKey, fmt.Sprintf("type:%s", event.Type), 1)
	pipe.HIncrBy(ctx, dateKey, fmt.Sprintf("severity:%s", event.Severity), 1)
	pipe.Expire(ctx, dateKey, 32*24*time.Hour) // 32일 보관

	// 시간별 통계
	hourKey := fmt.Sprintf("security:stats:hourly:%s", time.Now().Format("2006-01-02:15"))
	pipe.HIncrBy(ctx, hourKey, fmt.Sprintf("type:%s", event.Type), 1)
	pipe.Expire(ctx, hourKey, 25*time.Hour) // 25시간 보관

	pipe.Exec(ctx)
}

func (et *EventTracker) checkAlerts(ctx context.Context, event *SecurityEvent) {
	threshold, exists := et.config.AlertThresholds[event.Type]
	if !exists {
		return
	}

	// 최근 1시간 내 같은 타입 이벤트 수 확인
	now := time.Now()
	oneHourAgo := now.Add(-time.Hour)
	
	count, err := et.getEventCountByType(ctx, event.Type, oneHourAgo, now)
	if err != nil {
		et.logger.Error("알림 확인 중 오류 발생", zap.Error(err))
		return
	}

	if count >= threshold {
		et.logger.Warn("보안 이벤트 임계값 초과",
			zap.String("event_type", string(event.Type)),
			zap.Int("count", count),
			zap.Int("threshold", threshold))
		
		// 여기에 실제 알림 로직 구현 (이메일, 슬랙 등)
		// TODO: 알림 발송 구현
	}
}

func (et *EventTracker) getEventCountByType(ctx context.Context, eventType EventType, start, end time.Time) (int, error) {
	key := et.getTimeSeriesKey(eventType)
	count, err := et.redis.ZCount(ctx, key, fmt.Sprintf("%d", start.Unix()), fmt.Sprintf("%d", end.Unix())).Result()
	return int(count), err
}

func (et *EventTracker) getEventCountBySeverity(ctx context.Context, severity Severity, start, end time.Time) (int, error) {
	key := et.getSeverityKey(severity)
	count, err := et.redis.ZCount(ctx, key, fmt.Sprintf("%d", start.Unix()), fmt.Sprintf("%d", end.Unix())).Result()
	return int(count), err
}

func (et *EventTracker) getEventCountInTimeRange(ctx context.Context, start, end time.Time) (int, error) {
	// 모든 이벤트 타입에 대해 집계
	totalCount := 0
	for eventType := range et.config.AlertThresholds {
		count, err := et.getEventCountByType(ctx, eventType, start, end)
		if err == nil {
			totalCount += count
		}
	}
	return totalCount, nil
}

func (et *EventTracker) startCleanupWorker() {
	ticker := time.NewTicker(6 * time.Hour) // 6시간마다 정리
	defer ticker.Stop()

	for range ticker.C {
		ctx := context.Background()
		et.cleanupExpiredEvents(ctx)
	}
}

func (et *EventTracker) cleanupExpiredEvents(ctx context.Context) {
	cutoffTime := time.Now().Add(-et.config.RetentionPeriod)
	cutoffScore := fmt.Sprintf("%d", cutoffTime.Unix())

	// 모든 시계열 키에서 만료된 이벤트 제거
	patterns := []string{
		"security:events:type:*",
		"security:events:severity:*",
		"security:events:user:*",
		"security:events:ip:*",
	}

	for _, pattern := range patterns {
		et.cleanupByPattern(ctx, pattern, cutoffScore)
	}

	et.logger.Debug("만료된 보안 이벤트 정리 완료",
		zap.Time("cutoff_time", cutoffTime))
}

func (et *EventTracker) cleanupByPattern(ctx context.Context, pattern, cutoffScore string) {
	var cursor uint64
	for {
		keys, nextCursor, err := et.redis.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			et.logger.Error("키 스캔 실패", zap.Error(err))
			break
		}

		for _, key := range keys {
			et.redis.ZRemRangeByScore(ctx, key, "-inf", cutoffScore)
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
}