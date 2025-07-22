package models

import (
	"time"
)

// AuthSession Redis 기반 사용자 인증 세션을 나타내는 모델
type AuthSession struct {
	ID           string                 `json:"id" redis:"id"`
	UserID       string                 `json:"user_id" redis:"user_id"`
	DeviceInfo   *DeviceFingerprint     `json:"device_info" redis:"device_info"`
	LocationInfo *LocationInfo          `json:"location_info" redis:"location_info"`
	CreatedAt    time.Time              `json:"created_at" redis:"created_at"`
	LastAccess   time.Time              `json:"last_access" redis:"last_access"`
	ExpiresAt    time.Time              `json:"expires_at" redis:"expires_at"`
	IsActive     bool                   `json:"is_active" redis:"is_active"`
	Metadata     map[string]interface{} `json:"metadata,omitempty" redis:"metadata"`
}

// DeviceFingerprint 디바이스 핑거프린트 정보
type DeviceFingerprint struct {
	UserAgent   string `json:"user_agent"`
	IPAddress   string `json:"ip_address"`
	Browser     string `json:"browser"`
	OS          string `json:"os"`
	Device      string `json:"device"`
	Fingerprint string `json:"fingerprint"`
}

// LocationInfo 지리적 위치 정보
type LocationInfo struct {
	Country     string  `json:"country"`
	CountryCode string  `json:"country_code"`
	City        string  `json:"city"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	TimeZone    string  `json:"timezone"`
}

// IsExpired 세션이 만료되었는지 확인
func (s *AuthSession) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// UpdateLastAccess 마지막 접근 시간 업데이트
func (s *AuthSession) UpdateLastAccess() {
	s.LastAccess = time.Now()
}

// SessionAuditLog 세션 감사 로그
type SessionAuditLog struct {
	ID          string    `json:"id"`
	SessionID   string    `json:"session_id"`
	UserID      string    `json:"user_id"`
	EventType   string    `json:"event_type"`
	EventData   string    `json:"event_data"`
	IPAddress   string    `json:"ip_address"`
	UserAgent   string    `json:"user_agent"`
	Location    string    `json:"location"`
	Timestamp   time.Time `json:"timestamp"`
	Severity    string    `json:"severity"`
	Description string    `json:"description"`
}

// SessionSecurityEvent 세션 보안 이벤트
type SessionSecurityEvent struct {
	ID          string                 `json:"id"`
	SessionID   string                 `json:"session_id"`
	UserID      string                 `json:"user_id"`
	EventType   string                 `json:"event_type"`
	RiskLevel   int                    `json:"risk_level"`
	Details     map[string]interface{} `json:"details"`
	Timestamp   time.Time              `json:"timestamp"`
	ActionTaken string                 `json:"action_taken"`
}

// SessionStatistics 세션 통계
type SessionStatistics struct {
	TotalSessions     int                    `json:"total_sessions"`
	ActiveSessions    int                    `json:"active_sessions"`
	ExpiredSessions   int                    `json:"expired_sessions"`
	DeviceBreakdown   map[string]int         `json:"device_breakdown"`
	LocationBreakdown map[string]int         `json:"location_breakdown"`
	BrowserBreakdown  map[string]int         `json:"browser_breakdown"`
	HourlyDistribution map[int]int           `json:"hourly_distribution"`
	SecurityEvents    []SessionSecurityEvent `json:"security_events"`
}

// SessionCreateRequest 세션 생성 요청
type SessionCreateRequest struct {
	UserID       string                 `json:"user_id" binding:"required"`
	DeviceInfo   *DeviceFingerprint     `json:"device_info,omitempty"`
	LocationInfo *LocationInfo          `json:"location_info,omitempty"`
	ExpiresIn    time.Duration          `json:"expires_in,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// SessionUpdateRequest 세션 업데이트 요청
type SessionUpdateRequest struct {
	IsActive     *bool                  `json:"is_active,omitempty"`
	LocationInfo *LocationInfo          `json:"location_info,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// SessionFilter 세션 필터
type SessionFilter struct {
	UserID      string     `json:"user_id,omitempty"`
	IsActive    *bool      `json:"is_active,omitempty"`
	DeviceType  string     `json:"device_type,omitempty"`
	Country     string     `json:"country,omitempty"`
	CreatedAfter *time.Time `json:"created_after,omitempty"`
	CreatedBefore *time.Time `json:"created_before,omitempty"`
}

// SessionListResponse 세션 목록 응답
type SessionListResponse struct {
	Sessions   []AuthSession      `json:"sessions"`
	Total      int                `json:"total"`
	Statistics *SessionStatistics `json:"statistics,omitempty"`
	Pagination *PaginationMeta    `json:"pagination,omitempty"`
}

// PaginationMeta 페이지네이션 메타데이터
type PaginationMeta struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
	HasNext    bool `json:"has_next"`
	HasPrev    bool `json:"has_prev"`
}