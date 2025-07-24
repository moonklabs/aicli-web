package auth

import (
	"context"
	"fmt"
	"time"
)

// Validator는 인증 검증을 위한 인터페이스입니다.
type Validator interface {
	// ValidateToken은 토큰을 검증하고 사용자 정보를 반환합니다.
	ValidateToken(ctx context.Context, token string) (*UserInfo, error)
	
	// ValidateSession은 세션을 검증합니다.
	ValidateSession(ctx context.Context, sessionID string) (*UserInfo, error)
	
	// IsTokenBlacklisted는 토큰이 블랙리스트에 있는지 확인합니다.
	IsTokenBlacklisted(ctx context.Context, token string) bool
}

// UserInfo는 사용자 정보를 담는 구조체입니다.
type UserInfo struct {
	ID       string    `json:"id"`
	Username string    `json:"username"`
	Email    string    `json:"email,omitempty"`
	Role     string    `json:"role"`
	IsActive bool      `json:"is_active"`
	LoginAt  time.Time `json:"login_at"`
}

// JWTValidator는 JWT 기반 인증 검증을 구현합니다.
type JWTValidator struct {
	jwtManager *JWTManager
	blacklist  *Blacklist
}

// NewJWTValidator는 새로운 JWT 검증자를 생성합니다.
func NewJWTValidator(jwtManager *JWTManager, blacklist *Blacklist) *JWTValidator {
	return &JWTValidator{
		jwtManager: jwtManager,
		blacklist:  blacklist,
	}
}

// ValidateToken은 JWT 토큰을 검증하고 사용자 정보를 반환합니다.
func (v *JWTValidator) ValidateToken(ctx context.Context, token string) (*UserInfo, error) {
	// 블랙리스트 확인
	if v.IsTokenBlacklisted(ctx, token) {
		return nil, fmt.Errorf("토큰이 블랙리스트에 등록되어 있습니다")
	}

	// JWT 토큰 검증
	claims, err := v.jwtManager.VerifyToken(token)
	if err != nil {
		return nil, fmt.Errorf("토큰 검증 실패: %w", err)
	}

	// 토큰 만료 확인
	if claims.ExpiresAt != nil && time.Now().After(claims.ExpiresAt.Time) {
		return nil, fmt.Errorf("토큰이 만료되었습니다")
	}

	// 사용자 정보 생성
	userInfo := &UserInfo{
		ID:       claims.UserID,
		Username: claims.UserName,
		Email:    claims.Email,
		Role:     claims.Role,
		IsActive: true,
		LoginAt:  claims.IssuedAt.Time,
	}

	return userInfo, nil
}

// ValidateSession은 세션을 검증합니다.
func (v *JWTValidator) ValidateSession(ctx context.Context, sessionID string) (*UserInfo, error) {
	// 세션 기반 검증은 현재 JWT만 지원하므로 에러 반환
	return nil, fmt.Errorf("세션 기반 검증은 지원되지 않습니다")
}

// IsTokenBlacklisted는 토큰이 블랙리스트에 있는지 확인합니다.
func (v *JWTValidator) IsTokenBlacklisted(ctx context.Context, token string) bool {
	if v.blacklist == nil {
		return false
	}
	
	return v.blacklist.IsBlacklisted(token)
}

// MockValidator는 테스트용 목 검증자입니다.
type MockValidator struct {
	ValidTokens map[string]*UserInfo
}

// NewMockValidator는 새로운 목 검증자를 생성합니다.
func NewMockValidator() *MockValidator {
	return &MockValidator{
		ValidTokens: make(map[string]*UserInfo),
	}
}

// ValidateToken은 목 토큰 검증을 수행합니다.
func (m *MockValidator) ValidateToken(ctx context.Context, token string) (*UserInfo, error) {
	if userInfo, exists := m.ValidTokens[token]; exists {
		return userInfo, nil
	}
	return nil, fmt.Errorf("유효하지 않은 토큰입니다")
}

// ValidateSession은 목 세션 검증을 수행합니다.
func (m *MockValidator) ValidateSession(ctx context.Context, sessionID string) (*UserInfo, error) {
	// 기본 사용자 정보 반환
	return &UserInfo{
		ID:       "test-user",
		Username: "testuser",
		Email:    "test@example.com",
		Role:     "user",
		IsActive: true,
		LoginAt:  time.Now(),
	}, nil
}

// IsTokenBlacklisted는 목 블랙리스트 확인을 수행합니다.
func (m *MockValidator) IsTokenBlacklisted(ctx context.Context, token string) bool {
	return false
}

// AddValidToken은 유효한 토큰을 추가합니다.
func (m *MockValidator) AddValidToken(token string, userInfo *UserInfo) {
	m.ValidTokens[token] = userInfo
}