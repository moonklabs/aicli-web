// Package auth provides types for authentication E2E tests
package auth

import (
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/aicli/aicli-web/internal/auth"
)

// AuthE2ETestSuite 인증 E2E 테스트 스위트
type AuthE2ETestSuite struct {
	server      *httptest.Server
	client      *http.Client
	baseURL     string
	jwtManager  *auth.JWTManager
	blacklist   *auth.Blacklist
	
	// 테스트 데이터
	testUsers   map[string]string // username -> password
	validTokens map[string]string // username -> token
}

// SecurityE2ETestSuite 보안 E2E 테스트 스위트
type SecurityE2ETestSuite struct {
	*AuthE2ETestSuite
	rateLimiter map[string][]time.Time // IP별 요청 시간 추적
}

// API 응답 구조체들

// LoginResponse 로그인 응답
type LoginResponse struct {
	Success bool                `json:"success"`
	Data    LoginResponseData   `json:"data"`
	Error   *ErrorResponse      `json:"error,omitempty"`
}

// LoginResponseData 로그인 응답 데이터
type LoginResponseData struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

// RefreshResponse 토큰 갱신 응답
type RefreshResponse struct {
	Success bool                 `json:"success"`
	Data    RefreshResponseData  `json:"data"`
	Error   *ErrorResponse       `json:"error,omitempty"`
}

// RefreshResponseData 토큰 갱신 응답 데이터
type RefreshResponseData struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// ProfileResponse 프로필 조회 응답
type ProfileResponse struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

// ErrorResponse 에러 응답
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}