package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/aicli/aicli-web/internal/config"
)

// NewJWTManagerFromConfig 설정에서 JWT 매니저 생성
func NewJWTManagerFromConfig(cfg *config.Config) (*JWTManager, error) {
	if cfg.JWT.Secret == "" {
		return nil, fmt.Errorf("JWT secret key is required")
	}

	return NewJWTManager(
		cfg.JWT.Secret,
		cfg.JWT.AccessTokenExpiry,
		cfg.JWT.RefreshTokenExpiry,
	), nil
}

// GenerateTokenWithExpiry 사용자 정의 만료 시간으로 토큰 생성 (테스트용)
func (m *JWTManager) GenerateTokenWithExpiry(userID, userName, email, role string, tokenType TokenType, customExpiry time.Duration) (string, error) {
	expirationTime := time.Now().Add(customExpiry)

	// 클레임 생성
	claims := NewClaims(userID, userName, email, role, expirationTime)

	// 토큰 생성
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// 토큰 서명
	tokenString, err := token.SignedString([]byte(m.secretKey))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}


// GetTokenClaims 토큰에서 클레임 추출 (검증 없이, 테스트용)
func (m *JWTManager) GetTokenClaims(tokenString string) (*Claims, error) {
	claims := &Claims{}

	// 검증 없이 클레임만 추출
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, claims)
	if err != nil {
		return nil, fmt.Errorf("failed to parse token claims: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token format")
	}

	return claims, nil
}

// IsTokenExpired 토큰 만료 확인 (검증 없이)
func (m *JWTManager) IsTokenExpired(tokenString string) (bool, error) {
	claims, err := m.GetTokenClaims(tokenString)
	if err != nil {
		return true, err
	}

	return claims.ExpiresAt.Before(time.Now()), nil
}

// GetTokenRemainingTime 토큰의 남은 유효 시간 반환
func (m *JWTManager) GetTokenRemainingTime(tokenString string) (time.Duration, error) {
	claims, err := m.GetTokenClaims(tokenString)
	if err != nil {
		return 0, err
	}

	remaining := time.Until(claims.ExpiresAt.Time)
	if remaining < 0 {
		return 0, nil // 이미 만료됨
	}

	return remaining, nil
}