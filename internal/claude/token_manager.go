package claude

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// TokenManager OAuth 토큰 관리 인터페이스
type TokenManager interface {
	// GetToken 유효한 토큰을 반환합니다
	GetToken(ctx context.Context) (string, error)
	// RefreshToken 토큰을 갱신합니다
	RefreshToken(ctx context.Context) error
	// ValidateToken 토큰의 유효성을 검증합니다
	ValidateToken(token string) error
	// SetToken 토큰을 설정합니다 (외부에서 갱신된 경우)
	SetToken(token string, expiresAt time.Time)
}

// tokenManager OAuth 토큰 관리자 구현
type tokenManager struct {
	mu          sync.RWMutex
	token       string
	apiKey      string // fallback용 API 키
	expiresAt   time.Time
	refreshFunc TokenRefreshFunc
}

// TokenRefreshFunc 토큰 갱신 함수 타입
type TokenRefreshFunc func(ctx context.Context) (token string, expiresAt time.Time, err error)

// NewTokenManager 새로운 토큰 관리자를 생성합니다
func NewTokenManager(token string, apiKey string, refreshFunc TokenRefreshFunc) TokenManager {
	return &tokenManager{
		token:       token,
		apiKey:      apiKey,
		refreshFunc: refreshFunc,
		expiresAt:   time.Now().Add(24 * time.Hour), // 기본 24시간 유효
	}
}

// GetToken 유효한 토큰을 반환합니다
func (tm *tokenManager) GetToken(ctx context.Context) (string, error) {
	tm.mu.RLock()
	
	// OAuth 토큰이 있고 유효한 경우
	if tm.token != "" && time.Now().Before(tm.expiresAt) {
		token := tm.token
		tm.mu.RUnlock()
		return token, nil
	}
	
	// API 키가 있는 경우 (fallback)
	if tm.apiKey != "" {
		apiKey := tm.apiKey
		tm.mu.RUnlock()
		return apiKey, nil
	}
	
	tm.mu.RUnlock()

	// 토큰 갱신 시도
	if err := tm.RefreshToken(ctx); err != nil {
		return "", fmt.Errorf("토큰 갱신 실패: %w", err)
	}

	tm.mu.RLock()
	token := tm.token
	tm.mu.RUnlock()

	if token == "" {
		return "", fmt.Errorf("유효한 인증 정보가 없습니다")
	}

	return token, nil
}

// RefreshToken 토큰을 갱신합니다
func (tm *tokenManager) RefreshToken(ctx context.Context) error {
	if tm.refreshFunc == nil {
		return fmt.Errorf("토큰 갱신 함수가 설정되지 않았습니다")
	}

	token, expiresAt, err := tm.refreshFunc(ctx)
	if err != nil {
		return fmt.Errorf("토큰 갱신 실패: %w", err)
	}

	tm.mu.Lock()
	tm.token = token
	tm.expiresAt = expiresAt
	tm.mu.Unlock()

	return nil
}

// ValidateToken 토큰의 유효성을 검증합니다
func (tm *tokenManager) ValidateToken(token string) error {
	if token == "" {
		return fmt.Errorf("토큰이 비어있습니다")
	}

	tm.mu.RLock()
	defer tm.mu.RUnlock()

	// OAuth 토큰 검증
	if token == tm.token {
		if time.Now().After(tm.expiresAt) {
			return fmt.Errorf("토큰이 만료되었습니다")
		}
		return nil
	}

	// API 키 검증
	if token == tm.apiKey {
		return nil
	}

	return fmt.Errorf("유효하지 않은 토큰입니다")
}

// SetToken 토큰을 설정합니다
func (tm *tokenManager) SetToken(token string, expiresAt time.Time) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	
	tm.token = token
	tm.expiresAt = expiresAt
}