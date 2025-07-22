package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims JWT 토큰에 포함되는 클레임 구조체
type Claims struct {
	jwt.RegisteredClaims
	UserID   string `json:"user_id"`
	UserName string `json:"username"`
	Role     string `json:"role"`
}

// NewClaims 새로운 JWT 클레임 생성
func NewClaims(userID, userName, role string, expirationTime time.Time) *Claims {
	return &Claims{
		UserID:   userID,
		UserName: userName,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "aicli-web",
			Subject:   userID,
		},
	}
}

// Valid 클레임 유효성 검증
func (c *Claims) Valid() error {
	// 추가 검증 로직 (필요시)
	if c.UserID == "" {
		return jwt.ErrTokenInvalidClaims
	}
	
	return nil
}