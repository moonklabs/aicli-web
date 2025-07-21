package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenType 토큰 타입
type TokenType string

const (
	// AccessToken 액세스 토큰
	AccessToken TokenType = "access"
	// RefreshToken 리프레시 토큰
	RefreshToken TokenType = "refresh"
)

// JWTManager JWT 토큰 관리자
type JWTManager struct {
	secretKey          string
	accessTokenExpiry  time.Duration
	refreshTokenExpiry time.Duration
}

// NewJWTManager 새로운 JWT 매니저 생성
func NewJWTManager(secretKey string, accessExpiry, refreshExpiry time.Duration) *JWTManager {
	return &JWTManager{
		secretKey:          secretKey,
		accessTokenExpiry:  accessExpiry,
		refreshTokenExpiry: refreshExpiry,
	}
}

// GenerateToken 토큰 생성
func (m *JWTManager) GenerateToken(userID, userName, role string, tokenType TokenType) (string, error) {
	var expirationTime time.Time
	
	switch tokenType {
	case AccessToken:
		expirationTime = time.Now().Add(m.accessTokenExpiry)
	case RefreshToken:
		expirationTime = time.Now().Add(m.refreshTokenExpiry)
	default:
		return "", fmt.Errorf("invalid token type: %s", tokenType)
	}

	// 클레임 생성
	claims := NewClaims(userID, userName, role, expirationTime)
	
	// 토큰 생성
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	
	// 토큰 서명
	tokenString, err := token.SignedString([]byte(m.secretKey))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}
	
	return tokenString, nil
}

// VerifyToken 토큰 검증 및 클레임 추출
func (m *JWTManager) VerifyToken(tokenString string) (*Claims, error) {
	claims := &Claims{}
	
	// 토큰 파싱 및 검증
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// 서명 방법 확인
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(m.secretKey), nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}
	
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	
	return claims, nil
}

// RefreshAccessToken 리프레시 토큰으로 새로운 액세스 토큰 발급
func (m *JWTManager) RefreshAccessToken(refreshTokenString string) (string, error) {
	// 리프레시 토큰 검증
	claims, err := m.VerifyToken(refreshTokenString)
	if err != nil {
		return "", fmt.Errorf("invalid refresh token: %w", err)
	}
	
	// 새로운 액세스 토큰 생성
	newAccessToken, err := m.GenerateToken(claims.UserID, claims.UserName, claims.Role, AccessToken)
	if err != nil {
		return "", fmt.Errorf("failed to generate new access token: %w", err)
	}
	
	return newAccessToken, nil
}

// ExtractTokenFromHeader Authorization 헤더에서 토큰 추출
func ExtractTokenFromHeader(authHeader string) (string, error) {
	if authHeader == "" {
		return "", fmt.Errorf("authorization header is empty")
	}
	
	// Bearer 토큰 형식 확인
	const bearerPrefix = "Bearer "
	if len(authHeader) < len(bearerPrefix) {
		return "", fmt.Errorf("invalid authorization header format")
	}
	
	if authHeader[:len(bearerPrefix)] != bearerPrefix {
		return "", fmt.Errorf("authorization header must start with 'Bearer '")
	}
	
	return authHeader[len(bearerPrefix):], nil
}