package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
)

// OAuthProvider OAuth 제공자 타입
type OAuthProvider string

const (
	// ProviderGoogle Google OAuth 제공자
	ProviderGoogle OAuthProvider = "google"
	// ProviderGitHub GitHub OAuth 제공자
	ProviderGitHub OAuthProvider = "github"
	// ProviderMicrosoft Microsoft OAuth 제공자 (선택적)
	ProviderMicrosoft OAuthProvider = "microsoft"
)

// OAuthConfig OAuth 설정 구조체
type OAuthConfig struct {
	Provider     OAuthProvider `json:"provider"`
	ClientID     string        `json:"client_id"`
	ClientSecret string        `json:"client_secret"`
	RedirectURL  string        `json:"redirect_url"`
	Scopes       []string      `json:"scopes"`
	Enabled      bool          `json:"enabled"`
}

// OAuthManager OAuth 관리자 인터페이스
type OAuthManager interface {
	// GetAuthURL 인증 URL 생성 (PKCE 지원)
	GetAuthURL(provider OAuthProvider, state string) (string, error)
	
	// ExchangeCode 인증 코드를 액세스 토큰으로 교환
	ExchangeCode(provider OAuthProvider, code, state string) (*oauth2.Token, error)
	
	// GetUserInfo 사용자 정보 조회
	GetUserInfo(provider OAuthProvider, token *oauth2.Token) (*OAuthUserInfo, error)
	
	// RefreshToken 토큰 갱신
	RefreshToken(provider OAuthProvider, refreshToken string) (*oauth2.Token, error)
	
	// ValidateState state 파라미터 검증
	ValidateState(state string) bool
}

// OAuthUserInfo 외부 제공자에서 받은 사용자 정보
type OAuthUserInfo struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Picture  string `json:"picture,omitempty"`
	Verified bool   `json:"verified"`
	Provider string `json:"provider"`
}

// OAuthManagerImpl OAuth 관리자 구현체
type OAuthManagerImpl struct {
	configs    map[OAuthProvider]*OAuthConfig
	jwtManager *JWTManager
	stateStore map[string]time.Time // 실제 환경에서는 Redis나 DB 사용
}

// NewOAuthManager 새로운 OAuth 매니저 생성
func NewOAuthManager(configs map[OAuthProvider]*OAuthConfig, jwtManager *JWTManager) *OAuthManagerImpl {
	return &OAuthManagerImpl{
		configs:    configs,
		jwtManager: jwtManager,
		stateStore: make(map[string]time.Time),
	}
}

// GetAuthURL 인증 URL 생성
func (m *OAuthManagerImpl) GetAuthURL(provider OAuthProvider, state string) (string, error) {
	config, err := m.getOAuthConfig(provider)
	if err != nil {
		return "", err
	}
	
	// state 저장 (5분 유효)
	m.stateStore[state] = time.Now().Add(5 * time.Minute)
	
	// PKCE는 oauth2 라이브러리에서 자동으로 처리
	return config.AuthCodeURL(state, oauth2.AccessTypeOffline), nil
}

// ExchangeCode 인증 코드를 액세스 토큰으로 교환
func (m *OAuthManagerImpl) ExchangeCode(provider OAuthProvider, code, state string) (*oauth2.Token, error) {
	// state 검증
	if !m.ValidateState(state) {
		return nil, fmt.Errorf("invalid or expired state parameter")
	}
	
	config, err := m.getOAuthConfig(provider)
	if err != nil {
		return nil, err
	}
	
	// 토큰 교환
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	token, err := config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code for token: %w", err)
	}
	
	// state 정리
	delete(m.stateStore, state)
	
	return token, nil
}

// GetUserInfo 사용자 정보 조회
func (m *OAuthManagerImpl) GetUserInfo(provider OAuthProvider, token *oauth2.Token) (*OAuthUserInfo, error) {
	config, err := m.getOAuthConfig(provider)
	if err != nil {
		return nil, err
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	client := config.Client(ctx, token)
	
	var userInfoURL string
	switch provider {
	case ProviderGoogle:
		userInfoURL = "https://www.googleapis.com/oauth2/v2/userinfo"
	case ProviderGitHub:
		userInfoURL = "https://api.github.com/user"
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
	
	resp, err := client.Get(userInfoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get user info: status %d", resp.StatusCode)
	}
	
	var rawUserInfo map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&rawUserInfo); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}
	
	return m.mapUserInfo(provider, rawUserInfo)
}

// RefreshToken 토큰 갱신
func (m *OAuthManagerImpl) RefreshToken(provider OAuthProvider, refreshToken string) (*oauth2.Token, error) {
	config, err := m.getOAuthConfig(provider)
	if err != nil {
		return nil, err
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	token := &oauth2.Token{RefreshToken: refreshToken}
	tokenSource := config.TokenSource(ctx, token)
	
	newToken, err := tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}
	
	return newToken, nil
}

// ValidateState state 파라미터 검증
func (m *OAuthManagerImpl) ValidateState(state string) bool {
	expiry, exists := m.stateStore[state]
	if !exists {
		return false
	}
	
	if time.Now().After(expiry) {
		delete(m.stateStore, state)
		return false
	}
	
	return true
}

// getOAuthConfig 제공자별 OAuth 설정 조회
func (m *OAuthManagerImpl) getOAuthConfig(provider OAuthProvider) (*oauth2.Config, error) {
	configData, exists := m.configs[provider]
	if !exists {
		return nil, fmt.Errorf("provider %s not configured", provider)
	}
	
	if !configData.Enabled {
		return nil, fmt.Errorf("provider %s is disabled", provider)
	}
	
	var endpoint oauth2.Endpoint
	switch provider {
	case ProviderGoogle:
		endpoint = google.Endpoint
	case ProviderGitHub:
		endpoint = github.Endpoint
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
	
	return &oauth2.Config{
		ClientID:     configData.ClientID,
		ClientSecret: configData.ClientSecret,
		RedirectURL:  configData.RedirectURL,
		Scopes:       configData.Scopes,
		Endpoint:     endpoint,
	}, nil
}

// mapUserInfo 제공자별 사용자 정보 매핑
func (m *OAuthManagerImpl) mapUserInfo(provider OAuthProvider, rawInfo map[string]interface{}) (*OAuthUserInfo, error) {
	userInfo := &OAuthUserInfo{
		Provider: string(provider),
	}
	
	switch provider {
	case ProviderGoogle:
		userInfo.ID = getString(rawInfo, "id")
		userInfo.Email = getString(rawInfo, "email")
		userInfo.Name = getString(rawInfo, "name")
		userInfo.Picture = getString(rawInfo, "picture")
		userInfo.Verified = getBool(rawInfo, "verified_email")
		
	case ProviderGitHub:
		userInfo.ID = fmt.Sprintf("%v", rawInfo["id"]) // GitHub은 숫자 ID
		userInfo.Email = getString(rawInfo, "email")
		userInfo.Name = getString(rawInfo, "name")
		if userInfo.Name == "" {
			userInfo.Name = getString(rawInfo, "login") // fallback to username
		}
		userInfo.Picture = getString(rawInfo, "avatar_url")
		userInfo.Verified = true // GitHub 계정은 기본적으로 검증됨
	}
	
	if userInfo.ID == "" {
		return nil, fmt.Errorf("failed to get user ID from %s", provider)
	}
	
	return userInfo, nil
}

// 헬퍼 함수들
func getString(data map[string]interface{}, key string) string {
	if val, ok := data[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getBool(data map[string]interface{}, key string) bool {
	if val, ok := data[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return false
}