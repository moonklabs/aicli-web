package session

import (
	"context"
	"time"

	"github.com/aicli/aicli-web/internal/models"
)

// Store는 세션 저장소 인터페이스입니다.
type Store interface {
	// Create는 새로운 세션을 생성합니다.
	Create(ctx context.Context, session *models.AuthSession) error
	
	// Get은 세션 ID로 세션을 조회합니다.
	Get(ctx context.Context, sessionID string) (*models.AuthSession, error)
	
	// Update는 기존 세션을 업데이트합니다.
	Update(ctx context.Context, session *models.AuthSession) error
	
	// Delete는 세션을 삭제합니다.
	Delete(ctx context.Context, sessionID string) error
	
	// GetUserSessions는 사용자의 모든 활성 세션을 조회합니다.
	GetUserSessions(ctx context.Context, userID string) ([]*models.AuthSession, error)
	
	// GetDeviceSessions는 특정 디바이스의 모든 세션을 조회합니다.
	GetDeviceSessions(ctx context.Context, fingerprint string) ([]*models.AuthSession, error)
	
	// CleanupExpiredSessions는 만료된 세션들을 정리합니다.
	CleanupExpiredSessions(ctx context.Context) error
	
	// ExtendSession은 세션의 만료 시간을 연장합니다.
	ExtendSession(ctx context.Context, sessionID string, duration time.Duration) error
	
	// CountUserActiveSessions는 사용자의 활성 세션 수를 반환합니다.
	CountUserActiveSessions(ctx context.Context, userID string) (int, error)
}

// Monitor는 세션 모니터링 인터페이스입니다.
type Monitor interface {
	// GetActiveSessions는 현재 활성 세션 목록을 반환합니다.
	GetActiveSessions(ctx context.Context) ([]*models.AuthSession, error)
	
	// GetSessionMetrics는 세션 메트릭을 반환합니다.
	GetSessionMetrics(ctx context.Context) (*SessionMetrics, error)
	
	// GetSessionHistory는 세션 히스토리를 반환합니다.
	GetSessionHistory(ctx context.Context, userID string, limit int) ([]*SessionEvent, error)
	
	// RecordSessionEvent는 세션 이벤트를 기록합니다.
	RecordSessionEvent(ctx context.Context, event *SessionEvent) error
	
	// RecordSuspiciousActivity는 의심스러운 활동을 기록합니다.
	RecordSuspiciousActivity(ctx context.Context, event *SessionEvent) error
}

// SecurityChecker는 세션 보안 검사 인터페이스입니다.
type SecurityChecker interface {
	// CheckDeviceFingerprint는 디바이스 핑거프린트를 검증합니다.
	CheckDeviceFingerprint(ctx context.Context, userID string, deviceInfo *models.DeviceFingerprint) error
	
	// CheckLocationChange는 위치 변경을 감지합니다.
	CheckLocationChange(ctx context.Context, sessionID string, newLocation *models.LocationInfo) error
	
	// DetectSuspiciousActivity는 의심스러운 활동을 감지합니다.
	DetectSuspiciousActivity(ctx context.Context, session *models.AuthSession) (bool, string)
	
	// ValidateConcurrentSessions는 동시 세션 제한을 검증합니다.
	ValidateConcurrentSessions(ctx context.Context, userID string, maxSessions int) error
}

// SessionMetrics는 세션 메트릭 정보입니다.
type SessionMetrics struct {
	TotalActiveSessions    int                        `json:"total_active_sessions"`
	SessionsByUser         map[string]int             `json:"sessions_by_user"`
	SessionsByDevice       map[string]int             `json:"sessions_by_device"`
	SessionsByLocation     map[string]int             `json:"sessions_by_location"`
	AverageSessionDuration time.Duration              `json:"average_session_duration"`
	CreatedToday           int                        `json:"created_today"`
	ExpiredToday           int                        `json:"expired_today"`
	SuspiciousActivities   int                        `json:"suspicious_activities"`
	TopUserAgents          []UserAgentStat            `json:"top_user_agents"`
	TopLocations           []LocationStat             `json:"top_locations"`
}

// SessionEvent는 세션 이벤트입니다.
type SessionEvent struct {
	ID          string                 `json:"id"`
	SessionID   string                 `json:"session_id"`
	UserID      string                 `json:"user_id"`
	EventType   SessionEventType       `json:"event_type"`
	EventData   map[string]interface{} `json:"event_data"`
	IPAddress   string                 `json:"ip_address"`
	UserAgent   string                 `json:"user_agent"`
	Location    *models.LocationInfo   `json:"location,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Severity    EventSeverity          `json:"severity"`
	Description string                 `json:"description"`
}

// SessionEventType는 세션 이벤트 타입입니다.
type SessionEventType string

const (
	EventSessionCreated         SessionEventType = "session_created"
	EventSessionExtended        SessionEventType = "session_extended"
	EventSessionExpired         SessionEventType = "session_expired"
	EventSessionTerminated      SessionEventType = "session_terminated"
	EventSuspiciousActivity     SessionEventType = "suspicious_activity"
	EventLocationChanged        SessionEventType = "location_changed"
	EventDeviceChanged          SessionEventType = "device_changed"
	EventConcurrentLimitReached SessionEventType = "concurrent_limit_reached"
	EventLoginSuccess           SessionEventType = "login_success"
	EventLoginFailed            SessionEventType = "login_failed"
)

// EventSeverity는 이벤트 심각도입니다.
type EventSeverity string

const (
	SeverityInfo     EventSeverity = "info"
	SeverityWarning  EventSeverity = "warning"
	SeverityCritical EventSeverity = "critical"
)

// UserAgentStat는 사용자 에이전트 통계입니다.
type UserAgentStat struct {
	UserAgent string `json:"user_agent"`
	Count     int    `json:"count"`
}

// LocationStat는 위치 통계입니다.
type LocationStat struct {
	Country string `json:"country"`
	City    string `json:"city"`
	Count   int    `json:"count"`
}