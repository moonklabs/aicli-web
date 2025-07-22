package session

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/aicli/aicli-web/internal/models"
)

// RedisMonitor는 Redis 기반 세션 모니터링 구현체입니다.
type RedisMonitor struct {
	client    redis.UniversalClient
	store     Store
	keyPrefix string
}

// NewRedisMonitor는 새로운 Redis 세션 모니터를 생성합니다.
func NewRedisMonitor(client redis.UniversalClient, store Store, keyPrefix string) *RedisMonitor {
	return &RedisMonitor{
		client:    client,
		store:     store,
		keyPrefix: keyPrefix,
	}
}

// eventKey는 이벤트 저장용 키를 생성합니다.
func (m *RedisMonitor) eventKey(userID string) string {
	return fmt.Sprintf("%s:events:%s", m.keyPrefix, userID)
}

// metricsKey는 메트릭 저장용 키를 생성합니다.
func (m *RedisMonitor) metricsKey() string {
	return fmt.Sprintf("%s:metrics", m.keyPrefix)
}

// GetActiveSessions는 현재 활성 세션 목록을 반환합니다.
func (m *RedisMonitor) GetActiveSessions(ctx context.Context) ([]*models.Session, error) {
	// 모든 세션 키를 스캔
	pattern := fmt.Sprintf("%s:session:*", m.keyPrefix)
	
	var cursor uint64
	var allSessions []*models.Session
	
	for {
		keys, nextCursor, err := m.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, fmt.Errorf("세션 키 스캔 실패: %w", err)
		}
		
		// 각 키에 대해 세션 조회
		for _, key := range keys {
			data, err := m.client.Get(ctx, key).Result()
			if err != nil {
				if err == redis.Nil {
					continue // 이미 만료된 세션
				}
				continue // 에러가 있는 세션은 건너뛰기
			}
			
			var session models.Session
			if err := json.Unmarshal([]byte(data), &session); err != nil {
				continue // 파싱 에러가 있는 세션은 건너뛰기
			}
			
			// 활성 세션만 포함
			if session.IsActive {
				allSessions = append(allSessions, &session)
			}
		}
		
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	
	return allSessions, nil
}

// GetSessionMetrics는 세션 메트릭을 반환합니다.
func (m *RedisMonitor) GetSessionMetrics(ctx context.Context) (*SessionMetrics, error) {
	activeSessions, err := m.GetActiveSessions(ctx)
	if err != nil {
		return nil, fmt.Errorf("활성 세션 조회 실패: %w", err)
	}
	
	metrics := &SessionMetrics{
		TotalActiveSessions: len(activeSessions),
		SessionsByUser:      make(map[string]int),
		SessionsByDevice:    make(map[string]int),
		SessionsByLocation:  make(map[string]int),
		TopUserAgents:       make([]UserAgentStat, 0),
		TopLocations:        make([]LocationStat, 0),
	}
	
	userAgentMap := make(map[string]int)
	locationMap := make(map[string]int)
	var totalDuration time.Duration
	
	// 세션 데이터 분석
	for _, session := range activeSessions {
		// 사용자별 세션 수
		metrics.SessionsByUser[session.UserID]++
		
		// 디바이스별 세션 수
		if session.DeviceInfo != nil {
			deviceKey := fmt.Sprintf("%s/%s", session.DeviceInfo.OS, session.DeviceInfo.Browser)
			metrics.SessionsByDevice[deviceKey]++
			
			// User-Agent 통계
			if session.DeviceInfo.UserAgent != "" {
				userAgentMap[session.DeviceInfo.UserAgent]++
			}
		}
		
		// 위치별 세션 수
		if session.LocationInfo != nil {
			locationKey := fmt.Sprintf("%s/%s", session.LocationInfo.Country, session.LocationInfo.City)
			metrics.SessionsByLocation[locationKey]++
			locationMap[locationKey]++
		}
		
		// 세션 지속 시간
		sessionDuration := time.Since(session.CreatedAt)
		totalDuration += sessionDuration
	}
	
	// 평균 세션 지속 시간 계산
	if len(activeSessions) > 0 {
		metrics.AverageSessionDuration = totalDuration / time.Duration(len(activeSessions))
	}
	
	// 상위 User-Agent 정렬
	for userAgent, count := range userAgentMap {
		metrics.TopUserAgents = append(metrics.TopUserAgents, UserAgentStat{
			UserAgent: userAgent,
			Count:     count,
		})
	}
	
	// 상위 위치 정렬  
	for location, count := range locationMap {
		parts := strings.Split(location, "/")
		if len(parts) >= 2 {
			metrics.TopLocations = append(metrics.TopLocations, LocationStat{
				Country: parts[0],
				City:    parts[1],
				Count:   count,
			})
		}
	}
	
	// 오늘의 생성/만료 세션 수 (Redis에서 별도 카운터로 관리)
	today := time.Now().Format("2006-01-02")
	createdKey := fmt.Sprintf("%s:daily:created:%s", m.keyPrefix, today)
	expiredKey := fmt.Sprintf("%s:daily:expired:%s", m.keyPrefix, today)
	suspiciousKey := fmt.Sprintf("%s:daily:suspicious:%s", m.keyPrefix, today)
	
	if created, err := m.client.Get(ctx, createdKey).Int(); err == nil {
		metrics.CreatedToday = created
	}
	
	if expired, err := m.client.Get(ctx, expiredKey).Int(); err == nil {
		metrics.ExpiredToday = expired
	}
	
	if suspicious, err := m.client.Get(ctx, suspiciousKey).Int(); err == nil {
		metrics.SuspiciousActivities = suspicious
	}
	
	return metrics, nil
}

// GetSessionHistory는 세션 히스토리를 반환합니다.
func (m *RedisMonitor) GetSessionHistory(ctx context.Context, userID string, limit int) ([]*SessionEvent, error) {
	eventKey := m.eventKey(userID)
	
	// Redis List에서 최신 이벤트들을 조회
	events, err := m.client.LRange(ctx, eventKey, 0, int64(limit-1)).Result()
	if err != nil {
		return nil, fmt.Errorf("세션 이벤트 조회 실패: %w", err)
	}
	
	var sessionEvents []*SessionEvent
	for _, eventData := range events {
		var event SessionEvent
		if err := json.Unmarshal([]byte(eventData), &event); err != nil {
			continue // 파싱 에러가 있는 이벤트는 건너뛰기
		}
		sessionEvents = append(sessionEvents, &event)
	}
	
	return sessionEvents, nil
}

// RecordSessionEvent는 세션 이벤트를 기록합니다.
func (m *RedisMonitor) RecordSessionEvent(ctx context.Context, event *SessionEvent) error {
	// 이벤트 ID 생성
	if event.ID == "" {
		event.ID = generateEventID()
	}
	
	// 타임스탬프 설정
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	
	// 이벤트 직렬화
	eventData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("이벤트 직렬화 실패: %w", err)
	}
	
	// Redis List에 이벤트 추가 (최신 이벤트가 앞에 오도록)
	eventKey := m.eventKey(event.UserID)
	err = m.client.LPush(ctx, eventKey, eventData).Err()
	if err != nil {
		return fmt.Errorf("이벤트 저장 실패: %w", err)
	}
	
	// 이벤트 목록 크기 제한 (최대 1000개)
	m.client.LTrim(ctx, eventKey, 0, 999)
	
	// 이벤트 키 TTL 설정 (30일)
	m.client.Expire(ctx, eventKey, time.Hour*24*30)
	
	// 일별 통계 업데이트
	m.updateDailyStats(ctx, event)
	
	return nil
}

// updateDailyStats는 일별 통계를 업데이트합니다.
func (m *RedisMonitor) updateDailyStats(ctx context.Context, event *SessionEvent) {
	today := time.Now().Format("2006-01-02")
	
	switch event.EventType {
	case EventSessionCreated:
		key := fmt.Sprintf("%s:daily:created:%s", m.keyPrefix, today)
		m.client.Incr(ctx, key)
		m.client.Expire(ctx, key, time.Hour*24*7) // 7일 보관
		
	case EventSessionExpired, EventSessionTerminated:
		key := fmt.Sprintf("%s:daily:expired:%s", m.keyPrefix, today)
		m.client.Incr(ctx, key)
		m.client.Expire(ctx, key, time.Hour*24*7)
		
	case EventSuspiciousActivity:
		key := fmt.Sprintf("%s:daily:suspicious:%s", m.keyPrefix, today)
		m.client.Incr(ctx, key)
		m.client.Expire(ctx, key, time.Hour*24*30) // 30일 보관
	}
}

// generateEventID는 고유한 이벤트 ID를 생성합니다.
func generateEventID() string {
	return fmt.Sprintf("evt_%d_%d", time.Now().UnixNano(), time.Now().UnixMilli()%1000)
}

// GetSessionsByTimeRange는 시간 범위별 세션을 조회합니다.
func (m *RedisMonitor) GetSessionsByTimeRange(ctx context.Context, startTime, endTime time.Time) ([]*models.Session, error) {
	activeSessions, err := m.GetActiveSessions(ctx)
	if err != nil {
		return nil, err
	}
	
	var filteredSessions []*models.Session
	for _, session := range activeSessions {
		if session.CreatedAt.After(startTime) && session.CreatedAt.Before(endTime) {
			filteredSessions = append(filteredSessions, session)
		}
	}
	
	return filteredSessions, nil
}

// GetSuspiciousActivities는 의심스러운 활동 목록을 반환합니다.
func (m *RedisMonitor) GetSuspiciousActivities(ctx context.Context, limit int) ([]*SessionEvent, error) {
	// 모든 사용자의 의심스러운 활동 이벤트를 조회
	// 실제 구현에서는 별도의 의심스러운 활동 저장소를 사용할 수 있음
	
	suspiciousKey := fmt.Sprintf("%s:suspicious_events", m.keyPrefix)
	
	events, err := m.client.LRange(ctx, suspiciousKey, 0, int64(limit-1)).Result()
	if err != nil {
		return nil, fmt.Errorf("의심스러운 활동 조회 실패: %w", err)
	}
	
	var suspiciousEvents []*SessionEvent
	for _, eventData := range events {
		var event SessionEvent
		if err := json.Unmarshal([]byte(eventData), &event); err != nil {
			continue
		}
		suspiciousEvents = append(suspiciousEvents, &event)
	}
	
	return suspiciousEvents, nil
}

// RecordSuspiciousActivity는 의심스러운 활동을 기록합니다.
func (m *RedisMonitor) RecordSuspiciousActivity(ctx context.Context, event *SessionEvent) error {
	// 일반 이벤트로도 기록
	if err := m.RecordSessionEvent(ctx, event); err != nil {
		return err
	}
	
	// 의심스러운 활동 전용 저장소에도 기록
	suspiciousKey := fmt.Sprintf("%s:suspicious_events", m.keyPrefix)
	
	eventData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("이벤트 직렬화 실패: %w", err)
	}
	
	err = m.client.LPush(ctx, suspiciousKey, eventData).Err()
	if err != nil {
		return fmt.Errorf("의심스러운 활동 저장 실패: %w", err)
	}
	
	// 목록 크기 제한
	m.client.LTrim(ctx, suspiciousKey, 0, 9999) // 최대 10,000개
	
	// TTL 설정 (90일)
	m.client.Expire(ctx, suspiciousKey, time.Hour*24*90)
	
	return nil
}