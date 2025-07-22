package models

import (
	"time"
)

// User 사용자 모델
type User struct {
	Base
	Username         string           `json:"username" db:"username"`
	Email            string           `json:"email" db:"email"`
	PasswordHash     *string          `json:"-" db:"password_hash"` // 포인터로 nullable 처리
	DisplayName      string           `json:"display_name" db:"display_name"`
	FirstName        *string          `json:"first_name" db:"first_name"`
	LastName         *string          `json:"last_name" db:"last_name"`
	ProfilePicture   string           `json:"profile_picture" db:"profile_picture"`
	Avatar           *string          `json:"avatar" db:"avatar"` // 별도 아바타 URL
	Bio              *string          `json:"bio" db:"bio"`
	Location         *string          `json:"location" db:"location"`
	Website          *string          `json:"website" db:"website"`
	Role             string           `json:"role" db:"role"`
	IsActive         bool             `json:"is_active" db:"is_active"`
	LastLoginAt      *time.Time       `json:"last_login_at" db:"last_login_at"`
	EmailVerified    bool             `json:"email_verified" db:"email_verified"`
	TwoFactorEnabled bool             `json:"two_factor_enabled" db:"two_factor_enabled"`
	
	// OAuth 관련 필드들
	OAuthAccounts    []OAuthAccount   `json:"oauth_accounts,omitempty"`
	
	// RBAC 관련 필드들
	UserRoles        []UserRole       `json:"user_roles,omitempty"`
	Groups           []UserGroup      `json:"groups,omitempty"`
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

// UpdateUserRequest 사용자 수정 요청 (관리자용)
type UpdateUserRequest struct {
	DisplayName    *string `json:"display_name,omitempty" validate:"omitempty,max=100"`
	ProfilePicture *string `json:"profile_picture,omitempty"`
	IsActive       *bool   `json:"is_active,omitempty"`
	Role           *string `json:"role,omitempty" validate:"omitempty,oneof=admin user"`
}

// UpdateProfileRequest 프로파일 업데이트 요청 (일반 사용자용)
type UpdateProfileRequest struct {
	DisplayName    *string `json:"display_name,omitempty" validate:"omitempty,max=100"`
	FirstName      *string `json:"first_name,omitempty" validate:"omitempty,max=50"`
	LastName       *string `json:"last_name,omitempty" validate:"omitempty,max=50"`
	Bio            *string `json:"bio,omitempty" validate:"omitempty,max=500"`
	Location       *string `json:"location,omitempty" validate:"omitempty,max=100"`
	Website        *string `json:"website,omitempty" validate:"omitempty,url"`
	ProfilePicture *string `json:"profile_picture,omitempty"`
	Avatar         *string `json:"avatar,omitempty"`
}

// ChangePasswordRequest 비밀번호 변경 요청
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=8"`
	ConfirmPassword string `json:"confirm_password" validate:"required,min=8"`
}

// ChangeEmailRequest 이메일 변경 요청
type ChangeEmailRequest struct {
	NewEmail string `json:"new_email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// Enable2FARequest 2FA 활성화 요청
type Enable2FARequest struct {
	Secret string `json:"secret" validate:"required"`
	Code   string `json:"code" validate:"required,len=6"`
}

// Verify2FARequest 2FA 코드 검증 요청
type Verify2FARequest struct {
	Code string `json:"code" validate:"required,len=6"`
}

// ResetPasswordRequest 비밀번호 재설정 요청
type ResetPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// ConfirmResetPasswordRequest 비밀번호 재설정 확인 요청
type ConfirmResetPasswordRequest struct {
	Token           string `json:"token" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=8"`
	ConfirmPassword string `json:"confirm_password" validate:"required,min=8"`
}

// UserResponse 사용자 응답 (비밀번호 제외)
type UserResponse struct {
	ID               string         `json:"id"`
	Username         string         `json:"username"`
	Email            string         `json:"email"`
	DisplayName      string         `json:"display_name"`
	FirstName        *string        `json:"first_name"`
	LastName         *string        `json:"last_name"`
	ProfilePicture   string         `json:"profile_picture"`
	Avatar           *string        `json:"avatar"`
	Bio              *string        `json:"bio"`
	Location         *string        `json:"location"`
	Website          *string        `json:"website"`
	Role             string         `json:"role"`
	IsActive         bool           `json:"is_active"`
	LastLoginAt      *time.Time     `json:"last_login_at"`
	EmailVerified    bool           `json:"email_verified"`
	TwoFactorEnabled bool           `json:"two_factor_enabled"`
	OAuthProviders   []string       `json:"oauth_providers,omitempty"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
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
		ID:               u.ID,
		Username:         u.Username,
		Email:            u.Email,
		DisplayName:      u.DisplayName,
		FirstName:        u.FirstName,
		LastName:         u.LastName,
		ProfilePicture:   u.ProfilePicture,
		Avatar:           u.Avatar,
		Bio:              u.Bio,
		Location:         u.Location,
		Website:          u.Website,
		Role:             u.Role,
		IsActive:         u.IsActive,
		LastLoginAt:      u.LastLoginAt,
		EmailVerified:    u.EmailVerified,
		TwoFactorEnabled: u.TwoFactorEnabled,
		OAuthProviders:   oauthProviders,
		CreatedAt:        u.CreatedAt,
		UpdatedAt:        u.UpdatedAt,
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
	PaginationRequest
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

// UserActivity 사용자 활동 로그
type UserActivity struct {
	Base
	UserID    string    `json:"user_id" db:"user_id"`
	Action    string    `json:"action" db:"action"`       // 액션 타입 (login, logout, profile_update 등)
	Resource  string    `json:"resource" db:"resource"`   // 대상 리소스
	Details   string    `json:"details" db:"details"`     // 세부 정보 (JSON 형태)
	IPAddress string    `json:"ip_address" db:"ip_address"`
	UserAgent string    `json:"user_agent" db:"user_agent"`
	
	// 관계
	User *User `json:"user,omitempty"`
}

// PasswordResetToken 비밀번호 재설정 토큰
type PasswordResetToken struct {
	Base
	Token     string     `json:"token" db:"token"`
	UserID    string     `json:"user_id" db:"user_id"`
	ExpiresAt time.Time  `json:"expires_at" db:"expires_at"`
	UsedAt    *time.Time `json:"used_at" db:"used_at"`
	
	// 관계
	User *User `json:"user,omitempty"`
}

// TwoFactorSecret 2FA 비밀키 정보
type TwoFactorSecret struct {
	Base
	UserID    string    `json:"user_id" db:"user_id"`
	Secret    string    `json:"-" db:"secret"` // 보안상 JSON 응답에서 제외
	BackupCodes []string `json:"-" db:"backup_codes"` // JSON 배열로 저장
	IsActive  bool      `json:"is_active" db:"is_active"`
	
	// 관계
	User *User `json:"user,omitempty"`
}