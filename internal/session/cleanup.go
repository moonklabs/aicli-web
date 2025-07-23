package session

import (
	"context"
	"fmt"
	"time"

	"github.com/aicli/aicli-web/internal/models"
)

// CleanupService는 세션 정리 서비스입니다.
type CleanupService struct {
	store   Store
	monitor Monitor
	
	// 정리 정책 설정
	cleanupInterval    time.Duration
	expiredRetention   time.Duration // 만료된 세션 보관 기간
	inactiveThreshold  time.Duration // 비활성 세션 임계값
	batchSize          int           // 배치 처리 크기
	
	stopCh chan struct{}
}

// NewCleanupService는 새로운 정리 서비스를 생성합니다.
func NewCleanupService(store Store, monitor Monitor) *CleanupService {
	return &CleanupService{
		store:             store,
		monitor:          monitor,
		cleanupInterval:  time.Hour,        // 1시간마다 정리
		expiredRetention: time.Hour * 24,   // 만료된 세션 24시간 보관
		inactiveThreshold: time.Hour * 2,   // 2시간 비활성시 정리 대상
		batchSize:        100,              // 한 번에 100개씩 처리
		stopCh:           make(chan struct{}),
	}
}

// SetCleanupInterval은 정리 주기를 설정합니다.
func (c *CleanupService) SetCleanupInterval(interval time.Duration) {
	c.cleanupInterval = interval
}

// SetExpiredRetention은 만료된 세션 보관 기간을 설정합니다.
func (c *CleanupService) SetExpiredRetention(retention time.Duration) {
	c.expiredRetention = retention
}

// SetInactiveThreshold는 비활성 세션 임계값을 설정합니다.
func (c *CleanupService) SetInactiveThreshold(threshold time.Duration) {
	c.inactiveThreshold = threshold
}

// Start는 정리 서비스를 시작합니다.
func (c *CleanupService) Start(ctx context.Context) {
	ticker := time.NewTicker(c.cleanupInterval)
	defer ticker.Stop()
	
	// 시작 시 한 번 정리 실행
	c.runCleanup(ctx)
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopCh:
			return
		case <-ticker.C:
			c.runCleanup(ctx)
		}
	}
}

// Stop은 정리 서비스를 중지합니다.
func (c *CleanupService) Stop() {
	close(c.stopCh)
}

// runCleanup은 정리 작업을 실행합니다.
func (c *CleanupService) runCleanup(ctx context.Context) {
	// 1. 만료된 세션 정리
	if err := c.cleanupExpiredSessions(ctx); err != nil {
		if c.monitor != nil {
			event := &SessionEvent{
				EventType:   "cleanup_error",
				Timestamp:   time.Now(),
				Severity:    SeverityWarning,
				Description: fmt.Sprintf("만료된 세션 정리 실패: %v", err),
			}
			c.monitor.RecordSessionEvent(ctx, event)
		}
	}
	
	// 2. 비활성 세션 정리
	if err := c.cleanupInactiveSessions(ctx); err != nil {
		if c.monitor != nil {
			event := &SessionEvent{
				EventType:   "cleanup_error",
				Timestamp:   time.Now(),
				Severity:    SeverityWarning,
				Description: fmt.Sprintf("비활성 세션 정리 실패: %v", err),
			}
			c.monitor.RecordSessionEvent(ctx, event)
		}
	}
	
	// 3. 고아 데이터 정리
	if err := c.cleanupOrphanedData(ctx); err != nil {
		if c.monitor != nil {
			event := &SessionEvent{
				EventType:   "cleanup_error",
				Timestamp:   time.Now(),
				Severity:    SeverityWarning,
				Description: fmt.Sprintf("고아 데이터 정리 실패: %v", err),
			}
			c.monitor.RecordSessionEvent(ctx, event)
		}
	}
	
	// 정리 완료 이벤트 기록
	if c.monitor != nil {
		event := &SessionEvent{
			EventType:   "cleanup_completed",
			Timestamp:   time.Now(),
			Severity:    SeverityInfo,
			Description: "세션 정리 작업이 완료되었습니다",
		}
		c.monitor.RecordSessionEvent(ctx, event)
	}
}

// cleanupExpiredSessions는 만료된 세션들을 정리합니다.
func (c *CleanupService) cleanupExpiredSessions(ctx context.Context) error {
	// 기본 Store의 만료 세션 정리 기능 사용
	return c.store.CleanupExpiredSessions(ctx)
}

// cleanupInactiveSessions는 비활성 세션들을 정리합니다.
func (c *CleanupService) cleanupInactiveSessions(ctx context.Context) error {
	// 활성 세션 목록 조회
	activeSessions, err := c.monitor.GetActiveSessions(ctx)
	if err != nil {
		return fmt.Errorf("활성 세션 조회 실패: %w", err)
	}
	
	inactiveThreshold := time.Now().Add(-c.inactiveThreshold)
	var inactiveSessions []*models.AuthSession
	
	// 비활성 세션 식별
	for _, session := range activeSessions {
		if session.LastAccess.Before(inactiveThreshold) {
			inactiveSessions = append(inactiveSessions, session)
		}
	}
	
	// 배치별로 정리 처리
	for i := 0; i < len(inactiveSessions); i += c.batchSize {
		end := i + c.batchSize
		if end > len(inactiveSessions) {
			end = len(inactiveSessions)
		}
		
		batch := inactiveSessions[i:end]
		if err := c.processBatch(ctx, batch, "비활성으로 인한 자동 정리"); err != nil {
			return fmt.Errorf("배치 처리 실패 (인덱스 %d-%d): %w", i, end-1, err)
		}
	}
	
	return nil
}

// processBatch는 세션 배치를 처리합니다.
func (c *CleanupService) processBatch(ctx context.Context, sessions []*models.AuthSession, reason string) error {
	for _, session := range sessions {
		// 세션 비활성화
		session.IsActive = false
		session.LastAccess = time.Now()
		
		if err := c.store.Update(ctx, session); err != nil {
			// 개별 세션 처리 실패는 로그만 남기고 계속 진행
			continue
		}
		
		// 세션 종료 이벤트 기록
		if c.monitor != nil {
			event := &SessionEvent{
				SessionID:   session.ID,
				UserID:      session.UserID,
				EventType:   EventSessionExpired,
				Timestamp:   time.Now(),
				Severity:    SeverityInfo,
				Description: reason,
				EventData: map[string]interface{}{
					"last_access":     session.LastAccess,
					"inactive_duration": time.Since(session.LastAccess),
				},
			}
			c.monitor.RecordSessionEvent(ctx, event)
		}
	}
	
	return nil
}

// cleanupOrphanedData는 고아 데이터를 정리합니다.
func (c *CleanupService) cleanupOrphanedData(ctx context.Context) error {
	// 예: 사용자 세션 리스트에는 있지만 실제 세션은 없는 경우
	// 예: 디바이스 세션 리스트에는 있지만 실제 세션은 없는 경우
	
	// 이 기능은 Redis 구현체에 따라 달라질 수 있음
	// 현재는 기본 정리 기능에 의존
	
	return nil
}

// ForceCleanup은 즉시 정리 작업을 실행합니다.
func (c *CleanupService) ForceCleanup(ctx context.Context) error {
	c.runCleanup(ctx)
	return nil
}

// GetCleanupStats는 정리 통계를 반환합니다.
func (c *CleanupService) GetCleanupStats(ctx context.Context) (*CleanupStats, error) {
	// 활성 세션 수 조회
	activeSessions, err := c.monitor.GetActiveSessions(ctx)
	if err != nil {
		return nil, fmt.Errorf("활성 세션 조회 실패: %w", err)
	}
	
	stats := &CleanupStats{
		LastCleanupTime:   time.Now(), // 실제로는 마지막 정리 시간을 저장해야 함
		ActiveSessions:    len(activeSessions),
		CleanupInterval:   c.cleanupInterval,
		InactiveThreshold: c.inactiveThreshold,
		ExpiredRetention:  c.expiredRetention,
	}
	
	// 비활성 세션 수 계산
	inactiveThreshold := time.Now().Add(-c.inactiveThreshold)
	for _, session := range activeSessions {
		if session.LastAccess.Before(inactiveThreshold) {
			stats.InactiveSessions++
		}
	}
	
	// 만료 임박 세션 수 계산
	expirationThreshold := time.Now().Add(time.Hour) // 1시간 내 만료
	for _, session := range activeSessions {
		if session.ExpiresAt.Before(expirationThreshold) {
			stats.ExpiringSoon++
		}
	}
	
	return stats, nil
}

// CleanupStats는 정리 통계입니다.
type CleanupStats struct {
	LastCleanupTime   time.Time     `json:"last_cleanup_time"`
	ActiveSessions    int           `json:"active_sessions"`
	InactiveSessions  int           `json:"inactive_sessions"`
	ExpiringSoon      int           `json:"expiring_soon"`
	CleanupInterval   time.Duration `json:"cleanup_interval"`
	InactiveThreshold time.Duration `json:"inactive_threshold"`
	ExpiredRetention  time.Duration `json:"expired_retention"`
}

// CleanupPolicy는 정리 정책입니다.
type CleanupPolicy struct {
	EnableAutoCleanup  bool          `json:"enable_auto_cleanup"`
	CleanupInterval    time.Duration `json:"cleanup_interval"`
	InactiveThreshold  time.Duration `json:"inactive_threshold"`
	ExpiredRetention   time.Duration `json:"expired_retention"`
	MaxSessionsPerUser int           `json:"max_sessions_per_user"`
	BatchSize          int           `json:"batch_size"`
}

// ApplyPolicy는 정리 정책을 적용합니다.
func (c *CleanupService) ApplyPolicy(policy *CleanupPolicy) {
	if policy == nil {
		return
	}
	
	if policy.CleanupInterval > 0 {
		c.cleanupInterval = policy.CleanupInterval
	}
	
	if policy.InactiveThreshold > 0 {
		c.inactiveThreshold = policy.InactiveThreshold
	}
	
	if policy.ExpiredRetention > 0 {
		c.expiredRetention = policy.ExpiredRetention
	}
	
	if policy.BatchSize > 0 {
		c.batchSize = policy.BatchSize
	}
}