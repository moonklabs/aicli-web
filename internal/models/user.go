package models

import (
	"time"
)

// User 사용자 모델
type User struct {
	Base
	Username      string           `json:"username" db:"username"`
	Email         string           `json:"email" db:"email"`
	PasswordHash  *string          `json:"-" db:"password_hash"` // 포인터로 nullable 처리
	DisplayName   string           `json:"display_name" db:"display_name"`
	ProfilePicture string          `json:"profile_picture" db:"profile_picture"`
	Role          string           `json:"role" db:"role"`
	IsActive      bool             `json:"is_active" db:"is_active"`
	LastLoginAt   *time.Time       `json:"last_login_at" db:"last_login_at"`
	EmailVerified bool             `json:"email_verified" db:"email_verified"`
	
	// OAuth 관련 필드들
	OAuthAccounts []OAuthAccount   `json:"oauth_accounts,omitempty"`
}

// OAuthAccount OAuth 계정 연결 정보
type OAuthAccount struct {
	Base
	UserID       string    `json:"user_id" db:"user_id"`
	Provider     string    `json:"provider" db:"provider"`
	ProviderID   string    `json:"provider_id" db:"provider_id"`
	AccessToken  *string   `json:"-" db:"access_token"` // 보안상 JSON에서 제외
	RefreshToken *string   `json:"-" db:"refresh_token"`
	TokenExpiry  *time.Time `json:"-" db:"token_expiry"`
	ProfileData  string    `json:"profile_data" db:"profile_data"` // JSON 형태로 저장
	IsActive     bool      `json:"is_active" db:"is_active"`
	
	// 관계
	User *User `json:"user,omitempty"`
}

// CreateUserRequest 사용자 생성 요청
type CreateUserRequest struct {
	Username    string `json:"username" validate:"required,min=3,max=50"`
	Email       string `json:"email" validate:"required,email"`
	Password    string `json:"password" validate:"required,min=8"`
	DisplayName string `json:"display_name" validate:"max=100"`
	Role        string `json:"role" validate:"oneof=admin user"`
}

// UpdateUserRequest 사용자 수정 요청
type UpdateUserRequest struct {
	DisplayName    *string `json:"display_name,omitempty" validate:"omitempty,max=100"`
	ProfilePicture *string `json:"profile_picture,omitempty"`
	IsActive       *bool   `json:"is_active,omitempty"`
	Role           *string `json:"role,omitempty" validate:"omitempty,oneof=admin user"`
}

// UserResponse 사용자 응답 (비밀번호 제외)
type UserResponse struct {
	ID             string         `json:"id"`
	Username       string         `json:"username"`
	Email          string         `json:"email"`
	DisplayName    string         `json:"display_name"`
	ProfilePicture string         `json:"profile_picture"`
	Role           string         `json:"role"`
	IsActive       bool           `json:"is_active"`
	LastLoginAt    *time.Time     `json:"last_login_at"`
	EmailVerified  bool           `json:"email_verified"`
	OAuthProviders []string       `json:"oauth_providers,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

// ToResponse User 모델을 UserResponse로 변환
func (u *User) ToResponse() *UserResponse {
	var oauthProviders []string
	for _, account := range u.OAuthAccounts {
		if account.IsActive {
			oauthProviders = append(oauthProviders, account.Provider)
		}
	}

	return &UserResponse{
		ID:             u.ID,
		Username:       u.Username,
		Email:          u.Email,
		DisplayName:    u.DisplayName,
		ProfilePicture: u.ProfilePicture,
		Role:           u.Role,
		IsActive:       u.IsActive,
		LastLoginAt:    u.LastLoginAt,
		EmailVerified:  u.EmailVerified,
		OAuthProviders: oauthProviders,
		CreatedAt:      u.CreatedAt,
		UpdatedAt:      u.UpdatedAt,
	}
}

// HasOAuthProvider 특정 OAuth 제공자 연결 확인
func (u *User) HasOAuthProvider(provider string) bool {
	for _, account := range u.OAuthAccounts {
		if account.Provider == provider && account.IsActive {
			return true
		}
	}
	return false
}

// GetOAuthAccount 특정 OAuth 제공자 계정 조회
func (u *User) GetOAuthAccount(provider string) *OAuthAccount {
	for i := range u.OAuthAccounts {
		if u.OAuthAccounts[i].Provider == provider && u.OAuthAccounts[i].IsActive {
			return &u.OAuthAccounts[i]
		}
	}
	return nil
}

// UserFilter 사용자 필터링 조건
type UserFilter struct {
	Search     string   `json:"search,omitempty"`
	Role       string   `json:"role,omitempty"`
	IsActive   *bool    `json:"is_active,omitempty"`
	Provider   string   `json:"provider,omitempty"` // OAuth 제공자로 필터링
	Verified   *bool    `json:"verified,omitempty"` // 이메일 검증 상태
	Pagination
}

// UserStats 사용자 통계
type UserStats struct {
	TotalUsers      int64            `json:"total_users"`
	ActiveUsers     int64            `json:"active_users"`
	VerifiedUsers   int64            `json:"verified_users"`
	UsersByRole     map[string]int64 `json:"users_by_role"`
	UsersByProvider map[string]int64 `json:"users_by_provider"`
	RecentLogins    int64            `json:"recent_logins"` // 최근 30일
}