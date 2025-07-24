package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClaims(t *testing.T) {
	userID := "user123"
	userName := "testuser"
	role := "admin"
	expirationTime := time.Now().Add(15 * time.Minute)
	
	claims := NewClaims(userID, userName, "", role, expirationTime)
	
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, userName, claims.UserName)
	assert.Equal(t, role, claims.Role)
	assert.Equal(t, "aicli-web", claims.Issuer)
	assert.Equal(t, userID, claims.Subject)
}

func TestClaimsValid(t *testing.T) {
	t.Run("유효한 클레임", func(t *testing.T) {
		claims := NewClaims("user123", "testuser", "", "admin", time.Now().Add(15*time.Minute))
		err := claims.Valid()
		assert.NoError(t, err)
	})
	
	t.Run("만료된 클레임", func(t *testing.T) {
		claims := NewClaims("user123", "testuser", "", "admin", time.Now().Add(-15*time.Minute))
		err := claims.Valid()
		assert.Error(t, err)
	})
	
	t.Run("UserID가 없는 클레임", func(t *testing.T) {
		claims := NewClaims("", "testuser", "", "admin", time.Now().Add(15*time.Minute))
		err := claims.Valid()
		assert.Error(t, err)
	})
}

func TestJWTManager(t *testing.T) {
	secretKey := "test-secret-key-12345"
	manager := NewJWTManager(secretKey, 15*time.Minute, 7*24*time.Hour)
	
	t.Run("액세스 토큰 생성 및 검증", func(t *testing.T) {
		// 토큰 생성
		token, err := manager.GenerateToken("user123", "testuser", "", "admin", AccessToken)
		require.NoError(t, err)
		assert.NotEmpty(t, token)
		
		// 토큰 검증
		claims, err := manager.VerifyToken(token)
		require.NoError(t, err)
		assert.Equal(t, "user123", claims.UserID)
		assert.Equal(t, "testuser", claims.UserName)
		assert.Equal(t, "admin", claims.Role)
	})
	
	t.Run("리프레시 토큰 생성 및 검증", func(t *testing.T) {
		// 토큰 생성
		token, err := manager.GenerateToken("user123", "testuser", "", "admin", RefreshToken)
		require.NoError(t, err)
		assert.NotEmpty(t, token)
		
		// 토큰 검증
		claims, err := manager.VerifyToken(token)
		require.NoError(t, err)
		assert.Equal(t, "user123", claims.UserID)
	})
	
	t.Run("잘못된 토큰 타입", func(t *testing.T) {
		_, err := manager.GenerateToken("user123", "testuser", "", "admin", TokenType("invalid"))
		assert.Error(t, err)
	})
	
	t.Run("잘못된 토큰 검증", func(t *testing.T) {
		_, err := manager.VerifyToken("invalid-token")
		assert.Error(t, err)
	})
	
	t.Run("다른 시크릿 키로 서명된 토큰", func(t *testing.T) {
		// 다른 시크릿 키로 토큰 생성
		otherManager := NewJWTManager("other-secret-key", 15*time.Minute, 7*24*time.Hour)
		token, err := otherManager.GenerateToken("user123", "testuser", "", "admin", AccessToken)
		require.NoError(t, err)
		
		// 원래 매니저로 검증 시도
		_, err = manager.VerifyToken(token)
		assert.Error(t, err)
	})
}

func TestRefreshAccessToken(t *testing.T) {
	secretKey := "test-secret-key-12345"
	manager := NewJWTManager(secretKey, 15*time.Minute, 7*24*time.Hour)
	
	// 리프레시 토큰 생성
	refreshToken, err := manager.GenerateToken("user123", "testuser", "", "admin", RefreshToken)
	require.NoError(t, err)
	
	// 액세스 토큰 갱신
	newAccessToken, err := manager.RefreshAccessToken(refreshToken)
	require.NoError(t, err)
	assert.NotEmpty(t, newAccessToken)
	
	// 새 액세스 토큰 검증
	claims, err := manager.VerifyToken(newAccessToken)
	require.NoError(t, err)
	assert.Equal(t, "user123", claims.UserID)
	assert.Equal(t, "testuser", claims.UserName)
	assert.Equal(t, "admin", claims.Role)
}

func TestExtractTokenFromHeader(t *testing.T) {
	t.Run("유효한 Bearer 토큰", func(t *testing.T) {
		header := "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
		token, err := ExtractTokenFromHeader(header)
		require.NoError(t, err)
		assert.Equal(t, "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c", token)
	})
	
	t.Run("빈 헤더", func(t *testing.T) {
		_, err := ExtractTokenFromHeader("")
		assert.Error(t, err)
	})
	
	t.Run("Bearer 프리픽스 없음", func(t *testing.T) {
		_, err := ExtractTokenFromHeader("InvalidToken")
		assert.Error(t, err)
	})
	
	t.Run("잘못된 형식", func(t *testing.T) {
		_, err := ExtractTokenFromHeader("Bearer")
		assert.Error(t, err)
	})
}

func TestBlacklist(t *testing.T) {
	blacklist := NewBlacklist()
	
	t.Run("토큰 추가 및 확인", func(t *testing.T) {
		token := "test-token-123"
		expiresAt := time.Now().Add(1 * time.Hour)
		
		// 블랙리스트에 추가
		blacklist.Add(token, expiresAt)
		
		// 블랙리스트에 있는지 확인
		assert.True(t, blacklist.IsBlacklisted(token))
		assert.Equal(t, 1, blacklist.Size())
	})
	
	t.Run("블랙리스트에 없는 토큰", func(t *testing.T) {
		assert.False(t, blacklist.IsBlacklisted("non-existent-token"))
	})
	
	t.Run("만료된 토큰", func(t *testing.T) {
		token := "expired-token"
		expiresAt := time.Now().Add(-1 * time.Hour) // 이미 만료됨
		
		blacklist.Add(token, expiresAt)
		assert.False(t, blacklist.IsBlacklisted(token))
	})
	
	t.Run("토큰 제거", func(t *testing.T) {
		token := "remove-test-token"
		expiresAt := time.Now().Add(1 * time.Hour)
		
		blacklist.Add(token, expiresAt)
		assert.True(t, blacklist.IsBlacklisted(token))
		
		blacklist.Remove(token)
		assert.False(t, blacklist.IsBlacklisted(token))
	})
}

func TestJWTSigningMethod(t *testing.T) {
	// 잘못된 서명 방법을 사용한 토큰 테스트
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"user_id": "test",
	})
	
	// RSA 키 없이 서명 시도 (실패해야 함)
	tokenString, err := token.SignedString([]byte("secret"))
	assert.Error(t, err)
	
	// 만약 어떻게든 토큰이 생성되었다면, 검증은 실패해야 함
	if tokenString != "" {
		manager := NewJWTManager("secret", 15*time.Minute, 7*24*time.Hour)
		_, err = manager.VerifyToken(tokenString)
		assert.Error(t, err)
	}
}