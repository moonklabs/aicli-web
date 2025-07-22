package session

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/aicli/aicli-web/internal/models"
)

// ConcurrentSessionLimiter는 동시 세션 제한을 관리합니다.
type ConcurrentSessionLimiter struct {
	store        Store
	monitor      Monitor
	maxSessions  map[string]int // 사용자 역할별 최대 세션 수
	defaultMax   int            // 기본 최대 세션 수
}

// NewConcurrentSessionLimiter는 새로운 동시 세션 제한기를 생성합니다.
func NewConcurrentSessionLimiter(store Store, monitor Monitor, defaultMax int) *ConcurrentSessionLimiter {
	return &ConcurrentSessionLimiter{
		store:       store,
		monitor:     monitor,
		maxSessions: make(map[string]int),
		defaultMax:  defaultMax,
	}
}

// SetRoleLimit은 특정 역할의 최대 세션 수를 설정합니다.
func (l *ConcurrentSessionLimiter) SetRoleLimit(role string, maxSessions int) {
	l.maxSessions[role] = maxSessions
}

// GetMaxSessions는 사용자의 최대 세션 수를 반환합니다.
func (l *ConcurrentSessionLimiter) GetMaxSessions(userRole string) int {
	if max, exists := l.maxSessions[userRole]; exists {
		return max
	}
	return l.defaultMax
}

// CheckLimit는 새로운 세션을 생성하기 전에 제한을 확인합니다.
func (l *ConcurrentSessionLimiter) CheckLimit(ctx context.Context, userID string, userRole string) error {
	maxSessions := l.GetMaxSessions(userRole)
	
	// 현재 활성 세션 수 조회
	activeSessions, err := l.store.GetUserSessions(ctx, userID)
	if err != nil {
		return fmt.Errorf("사용자 세션 조회 실패: %w", err)
	}
	
	// 활성 세션 수가 제한을 초과하는지 확인
	if len(activeSessions) >= maxSessions {
		return ErrConcurrentSessionLimitExceeded
	}
	
	return nil
}

// EnforceLimit는 세션 제한을 강제 적용합니다.
// 새 세션을 위해 기존 세션을 종료할 수 있습니다.
func (l *ConcurrentSessionLimiter) EnforceLimit(ctx context.Context, userID string, userRole string, strategy LimitStrategy) error {
	maxSessions := l.GetMaxSessions(userRole)
	
	activeSessions, err := l.store.GetUserSessions(ctx, userID)
	if err != nil {
		return fmt.Errorf("사용자 세션 조회 실패: %w", err)
	}
	
	// 제한을 초과하지 않으면 그대로 유지
	if len(activeSessions) < maxSessions {
		return nil
	}
	
	// 제한 초과 시 전략에 따라 처리
	sessionsToTerminate := len(activeSessions) - maxSessions + 1 // +1은 새 세션 공간 확보
	
	if sessionsToTerminate <= 0 {
		return nil
	}
	
	// 종료할 세션 선택
	toTerminate := l.selectSessionsToTerminate(activeSessions, sessionsToTerminate, strategy)
	
	// 선택된 세션들 종료
	for _, session := range toTerminate {
		err := l.terminateSession(ctx, session)
		if err != nil {
			return fmt.Errorf("세션 종료 실패 (%s): %w", session.ID, err)
		}
		
		// 이벤트 기록
		if l.monitor != nil {
			event := &SessionEvent{
				SessionID:   session.ID,
				UserID:      userID,
				EventType:   EventConcurrentLimitReached,
				Timestamp:   time.Now(),
				Severity:    SeverityWarning,
				Description: fmt.Sprintf("동시 세션 제한 초과로 인한 세션 종료 (전략: %s)", strategy),
				EventData: map[string]interface{}{
					"terminated_session_id": session.ID,
					"strategy":             strategy,
					"max_sessions":         maxSessions,
					"active_sessions":      len(activeSessions),
				},
			}
			l.monitor.RecordSessionEvent(ctx, event)
		}
	}
	
	return nil
}

// LimitStrategy는 세션 제한 전략을 정의합니다.
type LimitStrategy string

const (
	// StrategyOldestFirst는 가장 오래된 세션부터 종료합니다.
	StrategyOldestFirst LimitStrategy = "oldest_first"
	
	// StrategyInactiveFirst는 비활성 세션부터 종료합니다.
	StrategyInactiveFirst LimitStrategy = "inactive_first"
	
	// StrategyLeastUsedFirst는 가장 적게 사용된 세션부터 종료합니다.
	StrategyLeastUsedFirst LimitStrategy = "least_used_first"
	
	// StrategyRejectNew는 새로운 세션을 거부합니다.
	StrategyRejectNew LimitStrategy = "reject_new"
)

// selectSessionsToTerminate는 종료할 세션을 선택합니다.
func (l *ConcurrentSessionLimiter) selectSessionsToTerminate(sessions []*models.Session, count int, strategy LimitStrategy) []*models.Session {
	if count <= 0 || count >= len(sessions) {
		return sessions
	}
	
	// 세션 복사본 생성
	sessionsCopy := make([]*models.Session, len(sessions))
	copy(sessionsCopy, sessions)
	
	switch strategy {
	case StrategyOldestFirst:
		// 생성 시간 기준 정렬 (오래된 순)
		sort.Slice(sessionsCopy, func(i, j int) bool {
			return sessionsCopy[i].CreatedAt.Before(sessionsCopy[j].CreatedAt)
		})
		
	case StrategyInactiveFirst:
		// 마지막 접근 시간 기준 정렬 (비활성 순)
		sort.Slice(sessionsCopy, func(i, j int) bool {
			return sessionsCopy[i].LastAccess.Before(sessionsCopy[j].LastAccess)
		})
		
	case StrategyLeastUsedFirst:
		// 접근 횟수가 적은 순으로 정렬 (추후 확장 가능)
		// 현재는 마지막 접근 시간으로 대체
		sort.Slice(sessionsCopy, func(i, j int) bool {
			return sessionsCopy[i].LastAccess.Before(sessionsCopy[j].LastAccess)
		})
		
	default:
		// 기본값: 가장 오래된 세션부터
		sort.Slice(sessionsCopy, func(i, j int) bool {
			return sessionsCopy[i].CreatedAt.Before(sessionsCopy[j].CreatedAt)
		})
	}
	
	return sessionsCopy[:count]
}

// terminateSession은 세션을 종료합니다.
func (l *ConcurrentSessionLimiter) terminateSession(ctx context.Context, session *models.Session) error {
	// 세션 비활성화
	session.IsActive = false
	session.LastAccess = time.Now()
	
	// 세션 업데이트
	err := l.store.Update(ctx, session)
	if err != nil {
		return fmt.Errorf("세션 비활성화 실패: %w", err)
	}
	
	// 세션 삭제 (선택적)
	// 히스토리 보존을 위해 바로 삭제하지 않고 TTL로 관리
	return nil
}

// GetSessionUsageStats는 사용자의 세션 사용 통계를 반환합니다.
func (l *ConcurrentSessionLimiter) GetSessionUsageStats(ctx context.Context, userID string) (*SessionUsageStats, error) {
	activeSessions, err := l.store.GetUserSessions(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("사용자 세션 조회 실패: %w", err)
	}
	
	stats := &SessionUsageStats{
		UserID:         userID,
		ActiveSessions: len(activeSessions),
		MaxSessions:    l.defaultMax, // 역할별 최대값은 별도 조회 필요
		Sessions:       make([]SessionStats, 0, len(activeSessions)),
	}
	
	for _, session := range activeSessions {
		sessionStats := SessionStats{
			SessionID:    session.ID,
			CreatedAt:    session.CreatedAt,
			LastAccess:   session.LastAccess,
			Duration:     time.Since(session.CreatedAt),
			DeviceInfo:   session.DeviceInfo,
			LocationInfo: session.LocationInfo,
			IsActive:     session.IsActive,
		}
		stats.Sessions = append(stats.Sessions, sessionStats)
	}
	
	// 세션들을 생성 시간 기준으로 정렬
	sort.Slice(stats.Sessions, func(i, j int) bool {
		return stats.Sessions[i].CreatedAt.After(stats.Sessions[j].CreatedAt)
	})
	
	return stats, nil
}

// SessionUsageStats는 세션 사용 통계입니다.
type SessionUsageStats struct {
	UserID         string         `json:"user_id"`
	ActiveSessions int            `json:"active_sessions"`
	MaxSessions    int            `json:"max_sessions"`
	Sessions       []SessionStats `json:"sessions"`
}

// SessionStats는 개별 세션 통계입니다.
type SessionStats struct {
	SessionID    string                      `json:"session_id"`
	CreatedAt    time.Time                   `json:"created_at"`
	LastAccess   time.Time                   `json:"last_access"`
	Duration     time.Duration               `json:"duration"`
	DeviceInfo   *models.DeviceFingerprint   `json:"device_info,omitempty"`
	LocationInfo *models.LocationInfo        `json:"location_info,omitempty"`
	IsActive     bool                        `json:"is_active"`
}

// NotifySessionLimit는 세션 제한에 대한 알림을 전송합니다.
func (l *ConcurrentSessionLimiter) NotifySessionLimit(ctx context.Context, userID string, message string) error {
	// 실제 구현에서는 이메일, SMS, 푸시 알림 등을 통해 사용자에게 알림
	// 현재는 로그 이벤트로만 처리
	
	if l.monitor != nil {
		event := &SessionEvent{
			UserID:      userID,
			EventType:   EventConcurrentLimitReached,
			Timestamp:   time.Now(),
			Severity:    SeverityWarning,
			Description: message,
		}
		return l.monitor.RecordSessionEvent(ctx, event)
	}
	
	return nil
}