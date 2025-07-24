package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/session"
)

// SessionManager는 고급 세션 관리자입니다.
type SessionManager struct {
	store          session.Store
	monitor        session.Monitor
	securityChecker session.SecurityChecker
	limiter        *session.ConcurrentSessionLimiter
	deviceGenerator *session.DeviceFingerprintGenerator
	
	// 설정
	defaultSessionTTL  time.Duration
	maxConcurrentSessions int
	enableSecurity     bool
}

// NewSessionManager는 새로운 세션 관리자를 생성합니다.
func NewSessionManager(redisClient redis.UniversalClient, config SessionConfig) *SessionManager {
	store := session.NewRedisStore(redisClient, config.KeyPrefix, config.DefaultTTL)
	monitor := session.NewRedisMonitor(redisClient, store, config.KeyPrefix)
	securityChecker := session.NewRedisSecurityChecker(store, monitor)
	limiter := session.NewConcurrentSessionLimiter(store, monitor, config.MaxConcurrentSessions)
	deviceGenerator := session.NewDeviceFingerprintGeneratorWithoutGeoIP()
	
	return &SessionManager{
		store:                 store,
		monitor:              monitor,
		securityChecker:      securityChecker,
		limiter:             limiter,
		deviceGenerator:     deviceGenerator,
		defaultSessionTTL:   config.DefaultTTL,
		maxConcurrentSessions: config.MaxConcurrentSessions,
		enableSecurity:      config.EnableSecurity,
	}
}

// SessionConfig는 세션 관리자 설정입니다.
type SessionConfig struct {
	KeyPrefix             string
	DefaultTTL            time.Duration
	MaxConcurrentSessions int
	EnableSecurity        bool
}

// CreateSession은 새로운 세션을 생성합니다.
func (sm *SessionManager) CreateSession(ctx context.Context, userID string, deviceInfo *models.DeviceFingerprint, locationInfo *models.LocationInfo) (*models.AuthSession, error) {
	// 보안 검사 (활성화된 경우)
	if sm.enableSecurity {
		// 디바이스 핑거프린트 검증
		if err := sm.securityChecker.CheckDeviceFingerprint(ctx, userID, deviceInfo); err != nil {
			// 새로운 디바이스지만 허용할 수 있는 경우 경고만 기록
			if err != session.ErrDeviceNotRecognized {
				return nil, err
			}
		}
		
		// 동시 세션 제한 검사
		if err := sm.limiter.CheckLimit(ctx, userID, "user"); err != nil {
			// 제한 초과 시 전략적으로 처리
			if err := sm.limiter.EnforceLimit(ctx, userID, "user", session.StrategyOldestFirst); err != nil {
				return nil, fmt.Errorf("동시 세션 제한 처리 실패: %w", err)
			}
		}
	}
	
	// 세션 생성
	sessionData := &models.AuthSession{
		ID:           generateSessionID(),
		UserID:       userID,
		DeviceInfo:   deviceInfo,
		LocationInfo: locationInfo,
		CreatedAt:    time.Now(),
		LastAccess:   time.Now(),
		ExpiresAt:    time.Now().Add(sm.defaultSessionTTL),
		IsActive:     true,
	}
	
	// 세션 저장
	if err := sm.store.Create(ctx, sessionData); err != nil {
		return nil, fmt.Errorf("세션 생성 실패: %w", err)
	}
	
	// 세션 생성 이벤트 기록
	event := &session.SessionEvent{
		SessionID:   sessionData.ID,
		UserID:      userID,
		EventType:   session.EventSessionCreated,
		Timestamp:   time.Now(),
		Severity:    session.SeverityInfo,
		Description: "새로운 세션이 생성되었습니다",
		IPAddress:   deviceInfo.IPAddress,
		UserAgent:   deviceInfo.UserAgent,
		Location:    locationInfo,
	}
	
	if sm.monitor != nil {
		sm.monitor.RecordSessionEvent(ctx, event)
	}
	
	return sessionData, nil
}

// ValidateSession은 세션을 검증하고 갱신합니다.
func (sm *SessionManager) ValidateSession(ctx context.Context, sessionID string) (*models.AuthSession, error) {
	// 세션 조회
	sessionData, err := sm.store.Get(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	
	// 세션 만료 체크
	if time.Now().After(sessionData.ExpiresAt) {
		sm.terminateSession(ctx, sessionData, "세션 만료")
		return nil, session.ErrSessionExpired
	}
	
	// 비활성 세션 체크
	if !sessionData.IsActive {
		return nil, session.ErrSessionInactive
	}
	
	// 보안 검사 (활성화된 경우)
	if sm.enableSecurity {
		if suspicious, reason := sm.securityChecker.DetectSuspiciousActivity(ctx, sessionData); suspicious {
			// 의심스러운 활동 이벤트 기록
			event := &session.SessionEvent{
				SessionID:   sessionID,
				UserID:      sessionData.UserID,
				EventType:   session.EventSuspiciousActivity,
				Timestamp:   time.Now(),
				Severity:    session.SeverityCritical,
				Description: fmt.Sprintf("의심스러운 활동 감지: %s", reason),
			}
			
			if sm.monitor != nil {
				sm.monitor.RecordSuspiciousActivity(ctx, event)
			}
			
			// 보안 정책에 따라 세션 종료 여부 결정
			// 현재는 경고만 하고 세션 유지
		}
	}
	
	// 마지막 접근 시간 업데이트
	sessionData.LastAccess = time.Now()
	if err := sm.store.Update(ctx, sessionData); err != nil {
		return nil, fmt.Errorf("세션 업데이트 실패: %w", err)
	}
	
	return sessionData, nil
}

// ExtendSession은 세션을 연장합니다.
func (sm *SessionManager) ExtendSession(ctx context.Context, sessionID string, duration time.Duration) error {
	// 세션 존재 및 유효성 확인
	sessionData, err := sm.ValidateSession(ctx, sessionID)
	if err != nil {
		return err
	}
	
	// 만료 시간 연장
	if err := sm.store.ExtendSession(ctx, sessionID, duration); err != nil {
		return fmt.Errorf("세션 연장 실패: %w", err)
	}
	
	// 세션 연장 이벤트 기록
	event := &session.SessionEvent{
		SessionID:   sessionID,
		UserID:      sessionData.UserID,
		EventType:   session.EventSessionExtended,
		Timestamp:   time.Now(),
		Severity:    session.SeverityInfo,
		Description: fmt.Sprintf("세션이 %.0f분 연장되었습니다", duration.Minutes()),
		EventData: map[string]interface{}{
			"extension_duration": duration.String(),
		},
	}
	
	if sm.monitor != nil {
		sm.monitor.RecordSessionEvent(ctx, event)
	}
	
	return nil
}

// TerminateSession은 세션을 강제 종료합니다.
func (sm *SessionManager) TerminateSession(ctx context.Context, sessionID string, reason string) error {
	sessionData, err := sm.store.Get(ctx, sessionID)
	if err != nil {
		// 세션이 없으면 이미 종료된 것으로 간주
		if err == session.ErrSessionNotFound {
			return nil
		}
		return err
	}
	
	return sm.terminateSession(ctx, sessionData, reason)
}

// terminateSession은 내부적으로 세션을 종료합니다.
func (sm *SessionManager) terminateSession(ctx context.Context, sessionData *models.AuthSession, reason string) error {
	// 세션 비활성화
	sessionData.IsActive = false
	sessionData.LastAccess = time.Now()
	
	if err := sm.store.Update(ctx, sessionData); err != nil {
		return fmt.Errorf("세션 종료 처리 실패: %w", err)
	}
	
	// 세션 종료 이벤트 기록
	event := &session.SessionEvent{
		SessionID:   sessionData.ID,
		UserID:      sessionData.UserID,
		EventType:   session.EventSessionTerminated,
		Timestamp:   time.Now(),
		Severity:    session.SeverityWarning,
		Description: fmt.Sprintf("세션이 종료되었습니다: %s", reason),
		EventData: map[string]interface{}{
			"termination_reason": reason,
		},
	}
	
	if sm.monitor != nil {
		sm.monitor.RecordSessionEvent(ctx, event)
	}
	
	return nil
}

// TerminateUserSessions는 특정 사용자의 모든 세션을 종료합니다.
func (sm *SessionManager) TerminateUserSessions(ctx context.Context, userID string, reason string) error {
	sessions, err := sm.store.GetUserSessions(ctx, userID)
	if err != nil {
		return fmt.Errorf("사용자 세션 조회 실패: %w", err)
	}
	
	var errors []string
	for _, sessionData := range sessions {
		if err := sm.terminateSession(ctx, sessionData, reason); err != nil {
			errors = append(errors, fmt.Sprintf("세션 %s 종료 실패: %v", sessionData.ID, err))
		}
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("일부 세션 종료 실패: %v", errors)
	}
	
	return nil
}

// GetUserSessions는 사용자의 활성 세션 목록을 반환합니다.
func (sm *SessionManager) GetUserSessions(ctx context.Context, userID string) ([]*models.AuthSession, error) {
	return sm.store.GetUserSessions(ctx, userID)
}

// GetSessionStats는 세션 통계를 반환합니다.
func (sm *SessionManager) GetSessionStats(ctx context.Context) (*session.SessionMetrics, error) {
	if sm.monitor == nil {
		return nil, fmt.Errorf("모니터링이 비활성화되어 있습니다")
	}
	
	return sm.monitor.GetSessionMetrics(ctx)
}

// CleanupExpiredSessions는 만료된 세션들을 정리합니다.
func (sm *SessionManager) CleanupExpiredSessions(ctx context.Context) error {
	return sm.store.CleanupExpiredSessions(ctx)
}

// generateSessionID는 고유한 세션 ID를 생성합니다.
func generateSessionID() string {
	// 실제 구현에서는 UUID 또는 secure random을 사용
	return fmt.Sprintf("sess_%d_%d", time.Now().UnixNano(), time.Now().UnixMilli()%1000)
}