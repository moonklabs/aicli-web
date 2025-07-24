package websocket

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/aicli/aicli-web/internal/auth"
)

// Authenticator WebSocket 인증 인터페이스
type Authenticator interface {
	// AuthenticateConnection 연결 시 인증 (핸드셰이크)
	AuthenticateConnection(r *http.Request) (*AuthInfo, error)
	
	// AuthenticateMessage 메시지 기반 인증
	AuthenticateMessage(token string) (*AuthInfo, error)
	
	// ValidateChannelAccess 채널 접근 권한 확인
	ValidateChannelAccess(userID, channel string) error
}

// AuthInfo 인증 정보
type AuthInfo struct {
	UserID   string            `json:"user_id"`
	Username string            `json:"username"`
	Role     string            `json:"role"`
	Claims   map[string]string `json:"claims"`
}

// JWTAuthenticator JWT 기반 인증
type JWTAuthenticator struct {
	jwtManager *auth.JWTManager
	blacklist  *auth.Blacklist
}

// NewJWTAuthenticator 새 JWT 인증기 생성
func NewJWTAuthenticator(jwtManager *auth.JWTManager, blacklist *auth.Blacklist) *JWTAuthenticator {
	return &JWTAuthenticator{
		jwtManager: jwtManager,
		blacklist:  blacklist,
	}
}

// AuthenticateConnection 연결 시 인증 (핸드셰이크)
func (ja *JWTAuthenticator) AuthenticateConnection(r *http.Request) (*AuthInfo, error) {
	// Authorization 헤더에서 토큰 추출
	token := ja.extractTokenFromHeader(r)
	if token == "" {
		// 쿼리 파라미터에서 토큰 추출
		token = r.URL.Query().Get("token")
	}
	
	if token == "" {
		return nil, &AuthError{
			Code:    "NO_TOKEN",
			Message: "인증 토큰이 필요합니다",
		}
	}
	
	return ja.AuthenticateMessage(token)
}

// AuthenticateMessage 메시지 기반 인증
func (ja *JWTAuthenticator) AuthenticateMessage(token string) (*AuthInfo, error) {
	// 블랙리스트 확인
	if ja.blacklist != nil && ja.blacklist.IsBlacklisted(token) {
		return nil, &AuthError{
			Code:    "TOKEN_BLACKLISTED",
			Message: "차단된 토큰입니다",
		}
	}
	
	// JWT 토큰 검증 (스텁 - 메서드 미구현)
	// claims, err := ja.jwtManager.ValidateToken(token)
	// if err != nil {
	//	return nil, &AuthError{
	//		Code:    "INVALID_TOKEN",
	//		Message: "유효하지 않은 토큰입니다",
	//		Details: err.Error(),
	//	}
	// }
	
	// 스텁 구현: 기본 클레임 생성
	claims := &auth.Claims{
		UserID: "stub_user",
		Role:   "user",
	}
	
	// 토큰 만료 확인 (스텁)
	// if !claims.Valid() {
	//	return nil, &AuthError{
	//		Code:    "TOKEN_EXPIRED",
	//		Message: "토큰이 만료되었습니다",
	//	}
	// }
	
	return &AuthInfo{
		UserID:   claims.UserID,
		Username: claims.UserName,
		Role:     claims.Role,
		Claims: map[string]string{
			"user_id":  claims.UserID,
			"username": claims.UserName,
			"role":     claims.Role,
		},
	}, nil
}

// ValidateChannelAccess 채널 접근 권한 확인
func (ja *JWTAuthenticator) ValidateChannelAccess(userID, channel string) error {
	// 채널 접근 권한 로직
	switch {
	// 사용자 개인 채널
	case strings.HasPrefix(channel, "user:"):
		channelUserID := strings.TrimPrefix(channel, "user:")
		if channelUserID != userID {
			return &AuthError{
				Code:    "ACCESS_DENIED",
				Message: "개인 채널 접근 권한이 없습니다",
			}
		}
	
	// 시스템 채널
	case channel == ChannelSystem:
		// TODO: 관리자 권한 확인
		return nil
	
	// 브로드캐스트 채널 (모든 사용자 접근 가능)
	case channel == ChannelBroadcast:
		return nil
	
	// 워크스페이스 채널
	case strings.HasPrefix(channel, ChannelWorkspace+":"):
		workspaceID := strings.TrimPrefix(channel, ChannelWorkspace+":")
		return ja.validateWorkspaceAccess(userID, workspaceID)
	
	// 세션 채널
	case strings.HasPrefix(channel, ChannelSession+":"):
		sessionID := strings.TrimPrefix(channel, ChannelSession+":")
		return ja.validateSessionAccess(userID, sessionID)
	
	// 태스크 채널
	case strings.HasPrefix(channel, ChannelTask+":"):
		taskID := strings.TrimPrefix(channel, ChannelTask+":")
		return ja.validateTaskAccess(userID, taskID)
	
	// 기타 채널은 거부
	default:
		return &AuthError{
			Code:    "INVALID_CHANNEL",
			Message: "유효하지 않은 채널입니다",
		}
	}
	
	return nil
}

// extractTokenFromHeader Authorization 헤더에서 토큰 추출
func (ja *JWTAuthenticator) extractTokenFromHeader(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}
	
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ""
	}
	
	return parts[1]
}

// validateWorkspaceAccess 워크스페이스 접근 권한 확인
func (ja *JWTAuthenticator) validateWorkspaceAccess(userID, workspaceID string) error {
	// TODO: 워크스페이스 멤버십 확인 로직 구현
	// 현재는 모든 사용자에게 접근 허용
	return nil
}

// validateSessionAccess 세션 접근 권한 확인
func (ja *JWTAuthenticator) validateSessionAccess(userID, sessionID string) error {
	// TODO: 세션 소유권 확인 로직 구현
	// 현재는 모든 사용자에게 접근 허용
	return nil
}

// validateTaskAccess 태스크 접근 권한 확인
func (ja *JWTAuthenticator) validateTaskAccess(userID, taskID string) error {
	// TODO: 태스크 소유권 확인 로직 구현
	// 현재는 모든 사용자에게 접근 허용
	return nil
}

// MockAuthenticator 테스트용 모의 인증기
type MockAuthenticator struct {
	ValidTokens map[string]*AuthInfo
}

// NewMockAuthenticator 새 모의 인증기 생성
func NewMockAuthenticator() *MockAuthenticator {
	return &MockAuthenticator{
		ValidTokens: make(map[string]*AuthInfo),
	}
}

// AddValidToken 유효한 토큰 추가
func (ma *MockAuthenticator) AddValidToken(token string, authInfo *AuthInfo) {
	ma.ValidTokens[token] = authInfo
}

// AuthenticateConnection 연결 시 인증 (모의)
func (ma *MockAuthenticator) AuthenticateConnection(r *http.Request) (*AuthInfo, error) {
	token := r.URL.Query().Get("token")
	if token == "" {
		return nil, &AuthError{
			Code:    "NO_TOKEN",
			Message: "인증 토큰이 필요합니다",
		}
	}
	
	return ma.AuthenticateMessage(token)
}

// AuthenticateMessage 메시지 기반 인증 (모의)
func (ma *MockAuthenticator) AuthenticateMessage(token string) (*AuthInfo, error) {
	if authInfo, exists := ma.ValidTokens[token]; exists {
		return authInfo, nil
	}
	
	return nil, &AuthError{
		Code:    "INVALID_TOKEN",
		Message: "유효하지 않은 토큰입니다",
	}
}

// ValidateChannelAccess 채널 접근 권한 확인 (모의)
func (ma *MockAuthenticator) ValidateChannelAccess(userID, channel string) error {
	// 모의 인증기는 모든 채널 접근 허용
	return nil
}

// RoleBasedAuthenticator 역할 기반 인증기
type RoleBasedAuthenticator struct {
	*JWTAuthenticator
	rolePermissions map[string][]string // role -> channels
}

// NewRoleBasedAuthenticator 새 역할 기반 인증기 생성
func NewRoleBasedAuthenticator(jwtManager *auth.JWTManager, blacklist *auth.Blacklist) *RoleBasedAuthenticator {
	return &RoleBasedAuthenticator{
		JWTAuthenticator: NewJWTAuthenticator(jwtManager, blacklist),
		rolePermissions: map[string][]string{
			"admin": {"*"}, // 관리자는 모든 채널 접근 가능
			"user":  {ChannelBroadcast, "user:*", ChannelWorkspace + ":*"}, // 사용자는 제한된 채널만
		},
	}
}

// ValidateChannelAccess 역할 기반 채널 접근 권한 확인
func (rba *RoleBasedAuthenticator) ValidateChannelAccess(userID, channel string) error {
	// TODO: 사용자 역할 조회 로직 필요
	userRole := "user" // 임시로 user 역할 할당
	
	permissions, exists := rba.rolePermissions[userRole]
	if !exists {
		return &AuthError{
			Code:    "INVALID_ROLE",
			Message: "유효하지 않은 역할입니다",
		}
	}
	
	// 권한 확인
	for _, permission := range permissions {
		if permission == "*" || permission == channel {
			return nil
		}
		
		// 와일드카드 패턴 확인
		if strings.HasSuffix(permission, "*") {
			prefix := strings.TrimSuffix(permission, "*")
			if strings.HasPrefix(channel, prefix) {
				return nil
			}
		}
	}
	
	return &AuthError{
		Code:    "ACCESS_DENIED",
		Message: fmt.Sprintf("채널 '%s'에 대한 접근 권한이 없습니다", channel),
	}
}

// AuthError 인증 에러
type AuthError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

func (e *AuthError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// AuthMiddleware WebSocket 인증 미들웨어
type AuthMiddleware struct {
	authenticator Authenticator
}

// NewAuthMiddleware 새 인증 미들웨어 생성
func NewAuthMiddleware(authenticator Authenticator) *AuthMiddleware {
	return &AuthMiddleware{
		authenticator: authenticator,
	}
}

// ValidateConnection 연결 유효성 검사
func (am *AuthMiddleware) ValidateConnection(r *http.Request) (*AuthInfo, error) {
	return am.authenticator.AuthenticateConnection(r)
}

// ValidateMessage 메시지 유효성 검사
func (am *AuthMiddleware) ValidateMessage(token string) (*AuthInfo, error) {
	return am.authenticator.AuthenticateMessage(token)
}

// ValidateChannelAccess 채널 접근 권한 검사
func (am *AuthMiddleware) ValidateChannelAccess(userID, channel string) error {
	return am.authenticator.ValidateChannelAccess(userID, channel)
}

// ConnectionInfo 연결 정보
type ConnectionInfo struct {
	UserID    string
	Username  string
	Role      string
	IPAddress string
	UserAgent string
	ConnectAt time.Time
}

// NewConnectionInfo 새 연결 정보 생성
func NewConnectionInfo(r *http.Request, authInfo *AuthInfo) *ConnectionInfo {
	return &ConnectionInfo{
		UserID:    authInfo.UserID,
		Username:  authInfo.Username,
		Role:      authInfo.Role,
		IPAddress: getClientIP(r),
		UserAgent: r.UserAgent(),
		ConnectAt: time.Now(),
	}
}

// getClientIP 클라이언트 IP 주소 추출
func getClientIP(r *http.Request) string {
	// X-Forwarded-For 헤더 확인
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	
	// X-Real-IP 헤더 확인
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	
	// RemoteAddr 사용
	parts := strings.Split(r.RemoteAddr, ":")
	if len(parts) >= 1 {
		return parts[0]
	}
	
	return r.RemoteAddr
}