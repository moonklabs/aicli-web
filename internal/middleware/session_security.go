package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/aicli/aicli-web/internal/auth"
	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/session"
)

// SessionSecurityMiddleware는 세션 보안 검사 미들웨어입니다.
type SessionSecurityMiddleware struct {
	sessionManager   *auth.SessionManager
	securityChecker  session.SecurityChecker
	auditLogger      *session.AuditLogger
	deviceGenerator  *session.DeviceFingerprintGenerator
	
	// 설정
	enableDeviceCheck    bool
	enableLocationCheck  bool
	enableSuspiciousCheck bool
	skipPaths           []string
}

// NewSessionSecurityMiddleware는 새로운 세션 보안 미들웨어를 생성합니다.
func NewSessionSecurityMiddleware(
	sessionManager *auth.SessionManager,
	securityChecker session.SecurityChecker,
	auditLogger *session.AuditLogger,
) *SessionSecurityMiddleware {
	return &SessionSecurityMiddleware{
		sessionManager:       sessionManager,
		securityChecker:     securityChecker,
		auditLogger:         auditLogger,
		deviceGenerator:     session.NewDeviceFingerprintGenerator(),
		enableDeviceCheck:   true,
		enableLocationCheck: true,
		enableSuspiciousCheck: true,
		skipPaths: []string{
			"/api/v1/auth/login",
			"/api/v1/auth/register", 
			"/api/v1/health",
			"/api/v1/version",
		},
	}
}

// SetSkipPaths는 보안 검사를 건너뛸 경로를 설정합니다.
func (m *SessionSecurityMiddleware) SetSkipPaths(paths []string) {
	m.skipPaths = paths
}

// EnableDeviceCheck는 디바이스 검사를 활성화/비활성화합니다.
func (m *SessionSecurityMiddleware) EnableDeviceCheck(enable bool) {
	m.enableDeviceCheck = enable
}

// EnableLocationCheck는 위치 검사를 활성화/비활성화합니다.
func (m *SessionSecurityMiddleware) EnableLocationCheck(enable bool) {
	m.enableLocationCheck = enable
}

// EnableSuspiciousCheck는 의심스러운 활동 검사를 활성화/비활성화합니다.
func (m *SessionSecurityMiddleware) EnableSuspiciousCheck(enable bool) {
	m.enableSuspiciousCheck = enable
}

// Handler는 세션 보안 검사 미들웨어 핸들러입니다.
func (m *SessionSecurityMiddleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 건너뛸 경로 확인
		if m.shouldSkipPath(c.Request.URL.Path) {
			c.Next()
			return
		}
		
		// Authorization 헤더에서 토큰 추출
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.Next() // 토큰이 없으면 다른 미들웨어가 처리하도록 함
			return
		}
		
		// 세션 ID 추출 (JWT 토큰에서 또는 별도 헤더에서)
		sessionID := c.GetHeader("X-Session-ID")
		if sessionID == "" {
			// JWT에서 세션 ID를 추출할 수도 있음
			sessionID = m.extractSessionIDFromToken(authHeader)
		}
		
		if sessionID == "" {
			c.Next() // 세션 ID가 없으면 건너뛰기
			return
		}
		
		// 세션 유효성 검사
		sessionData, err := m.sessionManager.ValidateSession(c, sessionID)
		if err != nil {
			m.handleSessionError(c, sessionID, err)
			return
		}
		
		// 보안 검사 수행
		if err := m.performSecurityChecks(c, sessionData); err != nil {
			m.handleSecurityViolation(c, sessionData, err)
			return
		}
		
		// 컨텍스트에 세션 정보 설정
		c.Set("session_id", sessionID)
		c.Set("session_data", sessionData)
		
		c.Next()
	}
}

// shouldSkipPath는 경로를 건너뛸지 확인합니다.
func (m *SessionSecurityMiddleware) shouldSkipPath(path string) bool {
	for _, skipPath := range m.skipPaths {
		if strings.HasPrefix(path, skipPath) {
			return true
		}
	}
	return false
}

// extractSessionIDFromToken는 토큰에서 세션 ID를 추출합니다.
func (m *SessionSecurityMiddleware) extractSessionIDFromToken(authHeader string) string {
	// JWT 토큰 파싱하여 세션 ID 추출
	// 실제 구현에서는 JWT 라이브러리를 사용
	// 현재는 간단한 구현만 제공
	return ""
}

// performSecurityChecks는 보안 검사를 수행합니다.
func (m *SessionSecurityMiddleware) performSecurityChecks(c *gin.Context, sessionData *models.Session) error {
	// 1. 디바이스 핑거프린트 검사
	if m.enableDeviceCheck {
		currentDevice := m.deviceGenerator.GenerateFromRequest(c.Request)
		if err := m.securityChecker.CheckDeviceFingerprint(c, sessionData.UserID, currentDevice); err != nil {
			// 새로운 디바이스 감지 시 로그만 기록하고 계속 진행
			if err == session.ErrDeviceNotRecognized && m.auditLogger != nil {
				m.auditLogger.LogSecurityEvent(c, sessionData.UserID, "device_change_detected", 
					"새로운 디바이스에서 접근 감지", map[string]interface{}{
						"session_id": sessionData.ID,
						"old_device": sessionData.DeviceInfo,
						"new_device": currentDevice,
					})
			}
		}
	}
	
	// 2. 위치 변경 검사
	if m.enableLocationCheck && m.securityChecker != nil {
		// 현재 위치 정보 추출 (실제로는 GeoIP 서비스 사용)
		currentLocation, _ := m.extractLocationFromRequest(c.Request)
		if currentLocation != nil {
			if err := m.securityChecker.CheckLocationChange(c, sessionData.ID, currentLocation); err != nil {
				if err == session.ErrLocationChanged {
					// 위치 변경 감지 시 경고 로그 기록
					if m.auditLogger != nil {
						m.auditLogger.LogSecurityEvent(c, sessionData.UserID, "location_change_detected",
							"비정상적인 위치 변경 감지", map[string]interface{}{
								"session_id": sessionData.ID,
								"old_location": sessionData.LocationInfo,
								"new_location": currentLocation,
							})
					}
				}
			}
		}
	}
	
	// 3. 의심스러운 활동 검사
	if m.enableSuspiciousCheck {
		if suspicious, reason := m.securityChecker.DetectSuspiciousActivity(c, sessionData); suspicious {
			if m.auditLogger != nil {
				m.auditLogger.LogSecurityEvent(c, sessionData.UserID, "suspicious_activity_detected",
					reason, map[string]interface{}{
						"session_id": sessionData.ID,
						"reason": reason,
						"ip_address": c.ClientIP(),
						"user_agent": c.Request.UserAgent(),
					})
			}
			
			// 보안 정책에 따라 세션을 차단할 수 있음
			// 현재는 로그만 기록하고 계속 진행
		}
	}
	
	return nil
}

// extractLocationFromRequest는 요청에서 위치 정보를 추출합니다.
func (m *SessionSecurityMiddleware) extractLocationFromRequest(req *http.Request) (*models.LocationInfo, error) {
	// 실제 구현에서는 GeoIP 서비스 사용
	// 현재는 간단한 구현만 제공
	return nil, nil
}

// handleSessionError는 세션 에러를 처리합니다.
func (m *SessionSecurityMiddleware) handleSessionError(c *gin.Context, sessionID string, err error) {
	var statusCode int
	var message string
	
	switch err {
	case session.ErrSessionNotFound:
		statusCode = http.StatusUnauthorized
		message = "세션을 찾을 수 없습니다"
	case session.ErrSessionExpired:
		statusCode = http.StatusUnauthorized  
		message = "세션이 만료되었습니다"
	case session.ErrSessionInactive:
		statusCode = http.StatusUnauthorized
		message = "세션이 비활성 상태입니다"
	default:
		statusCode = http.StatusInternalServerError
		message = "세션 검증 실패"
	}
	
	// 에러 로그 기록
	if m.auditLogger != nil {
		m.auditLogger.LogSecurityEvent(c, "", "session_validation_failed",
			message, map[string]interface{}{
				"session_id": sessionID,
				"error": err.Error(),
				"ip_address": c.ClientIP(),
				"user_agent": c.Request.UserAgent(),
			})
	}
	
	c.JSON(statusCode, gin.H{
		"error": message,
		"code": "SESSION_ERROR",
		"timestamp": time.Now(),
	})
	c.Abort()
}

// handleSecurityViolation는 보안 위반을 처리합니다.
func (m *SessionSecurityMiddleware) handleSecurityViolation(c *gin.Context, sessionData *models.Session, err error) {
	var statusCode int
	var message string
	var code string
	
	switch err {
	case session.ErrDeviceNotRecognized:
		statusCode = http.StatusForbidden
		message = "인식되지 않은 디바이스입니다"
		code = "DEVICE_NOT_RECOGNIZED"
	case session.ErrLocationChanged:
		statusCode = http.StatusForbidden
		message = "비정상적인 위치 변경이 감지되었습니다"
		code = "SUSPICIOUS_LOCATION"
	case session.ErrSuspiciousActivity:
		statusCode = http.StatusForbidden
		message = "의심스러운 활동이 감지되었습니다"
		code = "SUSPICIOUS_ACTIVITY"
	default:
		statusCode = http.StatusForbidden
		message = "보안 검사 실패"
		code = "SECURITY_VIOLATION"
	}
	
	// 보안 위반 로그 기록
	if m.auditLogger != nil {
		m.auditLogger.LogSecurityEvent(c, sessionData.UserID, "security_violation",
			message, map[string]interface{}{
				"session_id": sessionData.ID,
				"violation_type": code,
				"error": err.Error(),
				"ip_address": c.ClientIP(),
				"user_agent": c.Request.UserAgent(),
			})
	}
	
	c.JSON(statusCode, gin.H{
		"error": message,
		"code": code,
		"timestamp": time.Now(),
		"session_id": sessionData.ID,
	})
	c.Abort()
}

// DeviceTrackingMiddleware는 디바이스 추적 전용 미들웨어입니다.
func DeviceTrackingMiddleware(deviceGenerator *session.DeviceFingerprintGenerator) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 요청에서 디바이스 정보 생성
		deviceInfo := deviceGenerator.GenerateFromRequest(c.Request)
		
		// 컨텍스트에 디바이스 정보 설정
		c.Set("device_info", deviceInfo)
		
		c.Next()
	}
}