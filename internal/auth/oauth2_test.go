package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockOAuthServer OAuth 테스트를 위한 목업 서버
func NewMockOAuthServer() *httptest.Server {
	mux := http.NewServeMux()
	
	// Google 사용자 정보 엔드포인트 목업
	mux.HandleFunc("/oauth2/v2/userinfo", func(w http.ResponseWriter, r *http.Request) {
		userInfo := map[string]interface{}{
			"id":             "123456789",
			"email":          "test@example.com",
			"name":           "Test User",
			"picture":        "https://example.com/avatar.jpg",
			"verified_email": true,
		}
		json.NewEncoder(w).Encode(userInfo)
	})
	
	// GitHub 사용자 정보 엔드포인트 목업
	mux.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
		userInfo := map[string]interface{}{
			"id":         123456789,
			"email":      "test@github.com",
			"name":       "GitHub User",
			"login":      "testuser",
			"avatar_url": "https://github.com/avatar.jpg",
		}
		json.NewEncoder(w).Encode(userInfo)
	})
	
	return httptest.NewServer(mux)
}

func TestOAuthManagerImpl_GetAuthURL(t *testing.T) {
	// 테스트 설정
	configs := map[OAuthProvider]*OAuthConfig{
		ProviderGoogle: {
			Provider:     ProviderGoogle,
			ClientID:     "test-google-client-id",
			ClientSecret: "test-google-secret",
			RedirectURL:  "http://localhost:8080/auth/oauth/google/callback",
			Scopes:       []string{"openid", "email", "profile"},
			Enabled:      true,
		},
	}
	
	jwtManager := NewJWTManager("test-secret", 1*time.Hour, 24*time.Hour)
	oauthManager := NewOAuthManager(configs, jwtManager)
	
	t.Run("성공적인 인증 URL 생성", func(t *testing.T) {
		state := "test-state-123"
		authURL, err := oauthManager.GetAuthURL(ProviderGoogle, state)
		
		require.NoError(t, err)
		assert.Contains(t, authURL, "accounts.google.com")
		assert.Contains(t, authURL, "client_id=test-google-client-id")
		assert.Contains(t, authURL, "state=test-state-123")
		assert.Contains(t, authURL, "scope=openid+email+profile")
		
		// state가 저장되었는지 확인
		assert.True(t, oauthManager.ValidateState(state))
	})
	
	t.Run("비활성화된 제공자", func(t *testing.T) {
		configs[ProviderGoogle].Enabled = false
		defer func() { configs[ProviderGoogle].Enabled = true }()
		
		_, err := oauthManager.GetAuthURL(ProviderGoogle, "test-state")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "disabled")
	})
	
	t.Run("지원하지 않는 제공자", func(t *testing.T) {
		_, err := oauthManager.GetAuthURL(ProviderMicrosoft, "test-state")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not configured")
	})
}

func TestOAuthManagerImpl_ValidateState(t *testing.T) {
	jwtManager := NewJWTManager("test-secret", 1*time.Hour, 24*time.Hour)
	oauthManager := NewOAuthManager(make(map[OAuthProvider]*OAuthConfig), jwtManager)
	
	t.Run("유효한 state", func(t *testing.T) {
		state := "valid-state"
		oauthManager.stateStore[state] = time.Now().Add(5 * time.Minute)
		
		assert.True(t, oauthManager.ValidateState(state))
	})
	
	t.Run("존재하지 않는 state", func(t *testing.T) {
		assert.False(t, oauthManager.ValidateState("nonexistent-state"))
	})
	
	t.Run("만료된 state", func(t *testing.T) {
		state := "expired-state"
		oauthManager.stateStore[state] = time.Now().Add(-1 * time.Minute)
		
		assert.False(t, oauthManager.ValidateState(state))
		// 만료된 state는 자동으로 삭제되어야 함
		_, exists := oauthManager.stateStore[state]
		assert.False(t, exists)
	})
}

func TestOAuthManagerImpl_mapUserInfo(t *testing.T) {
	jwtManager := NewJWTManager("test-secret", 1*time.Hour, 24*time.Hour)
	oauthManager := NewOAuthManager(make(map[OAuthProvider]*OAuthConfig), jwtManager)
	
	t.Run("Google 사용자 정보 매핑", func(t *testing.T) {
		rawInfo := map[string]interface{}{
			"id":             "123456789",
			"email":          "test@gmail.com",
			"name":           "Google User",
			"picture":        "https://example.com/avatar.jpg",
			"verified_email": true,
		}
		
		userInfo, err := oauthManager.mapUserInfo(ProviderGoogle, rawInfo)
		require.NoError(t, err)
		
		assert.Equal(t, "123456789", userInfo.ID)
		assert.Equal(t, "test@gmail.com", userInfo.Email)
		assert.Equal(t, "Google User", userInfo.Name)
		assert.Equal(t, "https://example.com/avatar.jpg", userInfo.Picture)
		assert.True(t, userInfo.Verified)
		assert.Equal(t, string(ProviderGoogle), userInfo.Provider)
	})
	
	t.Run("GitHub 사용자 정보 매핑", func(t *testing.T) {
		rawInfo := map[string]interface{}{
			"id":         123456789, // GitHub은 숫자 ID
			"email":      "test@github.com",
			"name":       "GitHub User",
			"login":      "testuser",
			"avatar_url": "https://github.com/avatar.jpg",
		}
		
		userInfo, err := oauthManager.mapUserInfo(ProviderGitHub, rawInfo)
		require.NoError(t, err)
		
		assert.Equal(t, "123456789", userInfo.ID)
		assert.Equal(t, "test@github.com", userInfo.Email)
		assert.Equal(t, "GitHub User", userInfo.Name)
		assert.Equal(t, "https://github.com/avatar.jpg", userInfo.Picture)
		assert.True(t, userInfo.Verified) // GitHub은 기본적으로 검증됨
		assert.Equal(t, string(ProviderGitHub), userInfo.Provider)
	})
	
	t.Run("이름이 없는 GitHub 사용자 (username fallback)", func(t *testing.T) {
		rawInfo := map[string]interface{}{
			"id":         123456789,
			"email":      "test@github.com",
			"login":      "testuser",
			"avatar_url": "https://github.com/avatar.jpg",
		}
		
		userInfo, err := oauthManager.mapUserInfo(ProviderGitHub, rawInfo)
		require.NoError(t, err)
		
		assert.Equal(t, "testuser", userInfo.Name) // login을 name으로 사용
	})
	
	t.Run("ID가 없는 경우 에러", func(t *testing.T) {
		rawInfo := map[string]interface{}{
			"email": "test@example.com",
			"name":  "Test User",
		}
		
		_, err := oauthManager.mapUserInfo(ProviderGoogle, rawInfo)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get user ID")
	})
}

func TestOAuthManagerImpl_GetUserInfo(t *testing.T) {
	// Mock 서버 시작
	mockServer := NewMockOAuthServer()
	defer mockServer.Close()
	
	// 테스트 설정
	// configs := map[OAuthProvider]*OAuthConfig{
	// 	ProviderGoogle: {
	// 		Provider: ProviderGoogle,
	// 		Enabled:  true,
	// 	},
	// 	ProviderGitHub: {
	// 		Provider: ProviderGitHub,
	// 		Enabled:  true,
	// 	},
	// }
	
	// jwtManager := NewJWTManager("test-secret", 1*time.Hour, 24*time.Hour)
	// oauthManager := NewOAuthManager(configs, jwtManager)
	
	// 테스트용 토큰
	// token := &oauth2.Token{
	// 	AccessToken: "test-access-token",
	// 	TokenType:   "Bearer",
	// }
	
	t.Run("Google 사용자 정보 조회 성공", func(t *testing.T) {
		// Google의 userinfo URL을 mock 서버로 변경
		originalURL := "https://www.googleapis.com/oauth2/v2/userinfo"
		mockURL := mockServer.URL + "/oauth2/v2/userinfo"
		
		// 이 테스트는 실제로는 더 복잡한 설정이 필요하므로
		// 단위 테스트에서는 mapUserInfo 함수만 테스트하고
		// 통합 테스트에서 전체 플로우를 테스트하는 것이 좋습니다.
		t.Skip("이 테스트는 통합 테스트에서 수행합니다")
		
		_ = originalURL
		_ = mockURL
	})
}

func TestHelperFunctions(t *testing.T) {
	t.Run("getString 함수", func(t *testing.T) {
		data := map[string]interface{}{
			"string_field": "test_value",
			"number_field": 123,
			"bool_field":   true,
		}
		
		assert.Equal(t, "test_value", getString(data, "string_field"))
		assert.Equal(t, "", getString(data, "number_field")) // 숫자는 빈 문자열
		assert.Equal(t, "", getString(data, "nonexistent"))  // 존재하지 않는 필드
	})
	
	t.Run("getBool 함수", func(t *testing.T) {
		data := map[string]interface{}{
			"bool_field":   true,
			"string_field": "test",
			"false_field":  false,
		}
		
		assert.True(t, getBool(data, "bool_field"))
		assert.False(t, getBool(data, "string_field")) // 문자열은 false
		assert.False(t, getBool(data, "false_field"))
		assert.False(t, getBool(data, "nonexistent")) // 존재하지 않는 필드
	})
}

// 벤치마크 테스트
func BenchmarkOAuthManagerImpl_ValidateState(b *testing.B) {
	jwtManager := NewJWTManager("test-secret", 1*time.Hour, 24*time.Hour)
	oauthManager := NewOAuthManager(make(map[OAuthProvider]*OAuthConfig), jwtManager)
	
	// 테스트용 state 추가
	for i := 0; i < 1000; i++ {
		state := fmt.Sprintf("state-%d", i)
		oauthManager.stateStore[state] = time.Now().Add(5 * time.Minute)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		state := fmt.Sprintf("state-%d", i%1000)
		oauthManager.ValidateState(state)
	}
}

func BenchmarkOAuthManagerImpl_mapUserInfo(b *testing.B) {
	jwtManager := NewJWTManager("test-secret", 1*time.Hour, 24*time.Hour)
	oauthManager := NewOAuthManager(make(map[OAuthProvider]*OAuthConfig), jwtManager)
	
	rawInfo := map[string]interface{}{
		"id":             "123456789",
		"email":          "test@example.com",
		"name":           "Test User",
		"picture":        "https://example.com/avatar.jpg",
		"verified_email": true,
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := oauthManager.mapUserInfo(ProviderGoogle, rawInfo)
		if err != nil {
			b.Fatal(err)
		}
	}
}