package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/storage/interfaces"
)

// UserService는 사용자 관리 비즈니스 로직을 처리하는 서비스 인터페이스입니다
type UserService interface {
	// 프로파일 관리
	GetProfile(ctx context.Context, userID string) (*models.UserResponse, error)
	UpdateProfile(ctx context.Context, userID string, req *models.UpdateProfileRequest) (*models.UserResponse, error)
	
	// 계정 설정 관리
	ChangePassword(ctx context.Context, userID string, req *models.ChangePasswordRequest) error
	ChangeEmail(ctx context.Context, userID string, req *models.ChangeEmailRequest) error
	
	// 보안 설정
	Enable2FA(ctx context.Context, userID string, req *models.Enable2FARequest) (*models.TwoFactorSecret, error)
	Disable2FA(ctx context.Context, userID string) error
	Verify2FA(ctx context.Context, userID string, req *models.Verify2FARequest) error
	Generate2FASecret(ctx context.Context, userID string) (*models.TwoFactorSecret, error)
	
	// 비밀번호 재설정
	RequestPasswordReset(ctx context.Context, req *models.ResetPasswordRequest) error
	ConfirmPasswordReset(ctx context.Context, req *models.ConfirmResetPasswordRequest) error
	
	// 관리자 기능
	ListUsers(ctx context.Context, filter *models.UserFilter) (*models.PaginatedResponse, error)
	CreateUser(ctx context.Context, req *models.CreateUserRequest) (*models.UserResponse, error)
	GetUser(ctx context.Context, userID string) (*models.UserResponse, error)
	UpdateUser(ctx context.Context, userID string, req *models.UpdateUserRequest) (*models.UserResponse, error)
	DeleteUser(ctx context.Context, userID string) error
	
	// 활동 로그
	LogActivity(ctx context.Context, userID, action, resource, details, ipAddress, userAgent string) error
	GetUserActivities(ctx context.Context, userID string, pagination *models.PaginationRequest) (*models.PaginatedResponse, error)
	
	// 통계 및 메트릭
	GetUserStats(ctx context.Context) (*models.UserStats, error)
}

// userService는 UserService 인터페이스의 구현체입니다
type userService struct {
	storage interfaces.Storage
}

// NewUserService는 새로운 사용자 서비스를 생성합니다
func NewUserService(storage interfaces.Storage) UserService {
	return &userService{
		storage: storage,
	}
}

// GetProfile은 사용자 프로파일을 조회합니다
func (s *userService) GetProfile(ctx context.Context, userID string) (*models.UserResponse, error) {
	user := &models.User{}
	err := s.storage.GetByID(ctx, "users", userID, user)
	if err != nil {
		return nil, fmt.Errorf("failed to get user profile: %w", err)
	}
	
	return user.ToResponse(), nil
}

// UpdateProfile은 사용자 프로파일을 업데이트합니다
func (s *userService) UpdateProfile(ctx context.Context, userID string, req *models.UpdateProfileRequest) (*models.UserResponse, error) {
	// 기존 사용자 조회
	user := &models.User{}
	err := s.storage.GetByID(ctx, "users", userID, user)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	
	// 업데이트할 필드 적용
	if req.DisplayName != nil {
		user.DisplayName = *req.DisplayName
	}
	if req.FirstName != nil {
		user.FirstName = req.FirstName
	}
	if req.LastName != nil {
		user.LastName = req.LastName
	}
	if req.Bio != nil {
		user.Bio = req.Bio
	}
	if req.Location != nil {
		user.Location = req.Location
	}
	if req.Website != nil {
		user.Website = req.Website
	}
	if req.ProfilePicture != nil {
		user.ProfilePicture = *req.ProfilePicture
	}
	if req.Avatar != nil {
		user.Avatar = req.Avatar
	}
	
	user.UpdatedAt = time.Now()
	
	// 데이터베이스 업데이트
	err = s.storage.Update(ctx, "users", userID, user)
	if err != nil {
		return nil, fmt.Errorf("failed to update user profile: %w", err)
	}
	
	// 활동 로그 기록
	s.LogActivity(ctx, userID, "profile_update", "user", "프로파일 업데이트", "", "")
	
	return user.ToResponse(), nil
}

// ChangePassword는 사용자 비밀번호를 변경합니다
func (s *userService) ChangePassword(ctx context.Context, userID string, req *models.ChangePasswordRequest) error {
	// 비밀번호 확인 검증
	if req.NewPassword != req.ConfirmPassword {
		return fmt.Errorf("새 비밀번호와 확인 비밀번호가 일치하지 않습니다")
	}
	
	// 기존 사용자 조회
	user := &models.User{}
	err := s.storage.GetByID(ctx, "users", userID, user)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	
	// 현재 비밀번호 검증
	if user.PasswordHash != nil {
		err = bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(req.CurrentPassword))
		if err != nil {
			return fmt.Errorf("현재 비밀번호가 올바르지 않습니다")
		}
	}
	
	// 새 비밀번호 해시화
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}
	
	hashedPasswordStr := string(hashedPassword)
	user.PasswordHash = &hashedPasswordStr
	user.UpdatedAt = time.Now()
	
	// 데이터베이스 업데이트
	err = s.storage.Update(ctx, "users", userID, user)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}
	
	// 활동 로그 기록
	s.LogActivity(ctx, userID, "password_change", "user", "비밀번호 변경", "", "")
	
	return nil
}

// ChangeEmail은 사용자 이메일을 변경합니다
func (s *userService) ChangeEmail(ctx context.Context, userID string, req *models.ChangeEmailRequest) error {
	// 기존 사용자 조회
	user := &models.User{}
	err := s.storage.GetByID(ctx, "users", userID, user)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	
	// 비밀번호 검증
	if user.PasswordHash != nil {
		err = bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(req.Password))
		if err != nil {
			return fmt.Errorf("비밀번호가 올바르지 않습니다")
		}
	}
	
	// 이메일 중복 확인
	existingUser := &models.User{}
	err = s.storage.GetByField(ctx, "users", "email", req.NewEmail, existingUser)
	if err == nil {
		return fmt.Errorf("이미 사용 중인 이메일입니다")
	}
	
	// 이메일 업데이트
	user.Email = req.NewEmail
	user.EmailVerified = false // 새 이메일은 재인증 필요
	user.UpdatedAt = time.Now()
	
	err = s.storage.Update(ctx, "users", userID, user)
	if err != nil {
		return fmt.Errorf("failed to update email: %w", err)
	}
	
	// 활동 로그 기록
	s.LogActivity(ctx, userID, "email_change", "user", fmt.Sprintf("이메일 변경: %s", req.NewEmail), "", "")
	
	return nil
}

// Generate2FASecret은 2FA 비밀키를 생성합니다
func (s *userService) Generate2FASecret(ctx context.Context, userID string) (*models.TwoFactorSecret, error) {
	// 32바이트 랜덤 비밀키 생성
	secret := make([]byte, 32)
	_, err := rand.Read(secret)
	if err != nil {
		return nil, fmt.Errorf("failed to generate secret: %w", err)
	}
	
	secretStr := hex.EncodeToString(secret)
	
	// 백업 코드 생성 (8개의 8자리 코드)
	backupCodes := make([]string, 8)
	for i := range backupCodes {
		code := make([]byte, 4)
		rand.Read(code)
		backupCodes[i] = hex.EncodeToString(code)
	}
	
	twoFactorSecret := &models.TwoFactorSecret{
		Base: models.Base{
			ID:        generateID(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		UserID:      userID,
		Secret:      secretStr,
		BackupCodes: backupCodes,
		IsActive:    false, // 아직 활성화되지 않음
	}
	
	// 기존 2FA 설정이 있다면 비활성화
	existingSecret := &models.TwoFactorSecret{}
	err = s.storage.GetByField(ctx, "two_factor_secrets", "user_id", userID, existingSecret)
	if err == nil {
		s.storage.Delete(ctx, "two_factor_secrets", existingSecret.ID)
	}
	
	// 새 비밀키 저장
	err = s.storage.Create(ctx, "two_factor_secrets", twoFactorSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to save 2FA secret: %w", err)
	}
	
	return twoFactorSecret, nil
}

// Enable2FA는 2FA를 활성화합니다
func (s *userService) Enable2FA(ctx context.Context, userID string, req *models.Enable2FARequest) (*models.TwoFactorSecret, error) {
	// 2FA 비밀키 조회
	secret := &models.TwoFactorSecret{}
	err := s.storage.GetByField(ctx, "two_factor_secrets", "user_id", userID, secret)
	if err != nil {
		return nil, fmt.Errorf("2FA 비밀키를 찾을 수 없습니다")
	}
	
	// 제공된 비밀키와 일치하는지 확인
	if secret.Secret != req.Secret {
		return nil, fmt.Errorf("비밀키가 일치하지 않습니다")
	}
	
	// TODO: TOTP 코드 검증 로직 추가 필요
	// 실제 구현에서는 pquerna/otp 라이브러리 등을 사용하여 검증
	
	// 2FA 활성화
	secret.IsActive = true
	secret.UpdatedAt = time.Now()
	
	err = s.storage.Update(ctx, "two_factor_secrets", secret.ID, secret)
	if err != nil {
		return nil, fmt.Errorf("failed to enable 2FA: %w", err)
	}
	
	// 사용자 모델에도 2FA 활성화 표시
	user := &models.User{}
	err = s.storage.GetByID(ctx, "users", userID, user)
	if err == nil {
		user.TwoFactorEnabled = true
		user.UpdatedAt = time.Now()
		s.storage.Update(ctx, "users", userID, user)
	}
	
	// 활동 로그 기록
	s.LogActivity(ctx, userID, "2fa_enable", "user", "2FA 활성화", "", "")
	
	return secret, nil
}

// Disable2FA는 2FA를 비활성화합니다
func (s *userService) Disable2FA(ctx context.Context, userID string) error {
	// 2FA 비밀키 조회 및 삭제
	secret := &models.TwoFactorSecret{}
	err := s.storage.GetByField(ctx, "two_factor_secrets", "user_id", userID, secret)
	if err == nil {
		s.storage.Delete(ctx, "two_factor_secrets", secret.ID)
	}
	
	// 사용자 모델에서 2FA 비활성화
	user := &models.User{}
	err = s.storage.GetByID(ctx, "users", userID, user)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	
	user.TwoFactorEnabled = false
	user.UpdatedAt = time.Now()
	
	err = s.storage.Update(ctx, "users", userID, user)
	if err != nil {
		return fmt.Errorf("failed to disable 2FA: %w", err)
	}
	
	// 활동 로그 기록
	s.LogActivity(ctx, userID, "2fa_disable", "user", "2FA 비활성화", "", "")
	
	return nil
}

// Verify2FA는 2FA 코드를 검증합니다
func (s *userService) Verify2FA(ctx context.Context, userID string, req *models.Verify2FARequest) error {
	// TODO: TOTP 코드 검증 로직 구현
	// 실제 구현에서는 pquerna/otp 라이브러리 등을 사용
	return nil
}

// RequestPasswordReset은 비밀번호 재설정을 요청합니다
func (s *userService) RequestPasswordReset(ctx context.Context, req *models.ResetPasswordRequest) error {
	// 사용자 조회
	user := &models.User{}
	err := s.storage.GetByField(ctx, "users", "email", req.Email, user)
	if err != nil {
		// 보안상 사용자가 존재하지 않아도 성공으로 응답
		return nil
	}
	
	// 기존 토큰 삭제
	existingToken := &models.PasswordResetToken{}
	err = s.storage.GetByField(ctx, "password_reset_tokens", "user_id", user.ID, existingToken)
	if err == nil {
		s.storage.Delete(ctx, "password_reset_tokens", existingToken.ID)
	}
	
	// 새 토큰 생성
	tokenBytes := make([]byte, 32)
	rand.Read(tokenBytes)
	tokenStr := hex.EncodeToString(tokenBytes)
	
	resetToken := &models.PasswordResetToken{
		Base: models.Base{
			ID:        generateID(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Token:     tokenStr,
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(24 * time.Hour), // 24시간 후 만료
	}
	
	err = s.storage.Create(ctx, "password_reset_tokens", resetToken)
	if err != nil {
		return fmt.Errorf("failed to create reset token: %w", err)
	}
	
	// TODO: 이메일 발송 로직 추가
	// NotificationService를 통해 이메일 발송
	
	// 활동 로그 기록
	s.LogActivity(ctx, user.ID, "password_reset_request", "user", "비밀번호 재설정 요청", "", "")
	
	return nil
}

// ConfirmPasswordReset은 비밀번호 재설정을 확인합니다
func (s *userService) ConfirmPasswordReset(ctx context.Context, req *models.ConfirmResetPasswordRequest) error {
	// 비밀번호 확인 검증
	if req.NewPassword != req.ConfirmPassword {
		return fmt.Errorf("새 비밀번호와 확인 비밀번호가 일치하지 않습니다")
	}
	
	// 토큰 조회
	resetToken := &models.PasswordResetToken{}
	err := s.storage.GetByField(ctx, "password_reset_tokens", "token", req.Token, resetToken)
	if err != nil {
		return fmt.Errorf("유효하지 않은 토큰입니다")
	}
	
	// 토큰 만료 확인
	if time.Now().After(resetToken.ExpiresAt) {
		s.storage.Delete(ctx, "password_reset_tokens", resetToken.ID)
		return fmt.Errorf("토큰이 만료되었습니다")
	}
	
	// 토큰 사용 여부 확인
	if resetToken.UsedAt != nil {
		return fmt.Errorf("이미 사용된 토큰입니다")
	}
	
	// 사용자 조회
	user := &models.User{}
	err = s.storage.GetByID(ctx, "users", resetToken.UserID, user)
	if err != nil {
		return fmt.Errorf("사용자를 찾을 수 없습니다")
	}
	
	// 새 비밀번호 해시화
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}
	
	hashedPasswordStr := string(hashedPassword)
	user.PasswordHash = &hashedPasswordStr
	user.UpdatedAt = time.Now()
	
	// 사용자 업데이트
	err = s.storage.Update(ctx, "users", user.ID, user)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}
	
	// 토큰을 사용됨으로 표시
	now := time.Now()
	resetToken.UsedAt = &now
	resetToken.UpdatedAt = now
	s.storage.Update(ctx, "password_reset_tokens", resetToken.ID, resetToken)
	
	// 활동 로그 기록
	s.LogActivity(ctx, user.ID, "password_reset_confirm", "user", "비밀번호 재설정 완료", "", "")
	
	return nil
}

// LogActivity는 사용자 활동을 로그에 기록합니다
func (s *userService) LogActivity(ctx context.Context, userID, action, resource, details, ipAddress, userAgent string) error {
	activity := &models.UserActivity{
		Base: models.Base{
			ID:        generateID(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		UserID:    userID,
		Action:    action,
		Resource:  resource,
		Details:   details,
		IPAddress: ipAddress,
		UserAgent: userAgent,
	}
	
	return s.storage.Create(ctx, "user_activities", activity)
}

// GetUserActivities는 사용자 활동 로그를 조회합니다
func (s *userService) GetUserActivities(ctx context.Context, userID string, pagination *models.PaginationRequest) (*models.PaginatedResponse, error) {
	// TODO: 실제 구현에서는 pagination과 필터링 로직 추가
	activities := []models.UserActivity{}
	err := s.storage.GetByField(ctx, "user_activities", "user_id", userID, &activities)
	if err != nil {
		return nil, fmt.Errorf("failed to get user activities: %w", err)
	}
	
	return &models.PaginatedResponse{
		Data:        activities,
		Total:       int64(len(activities)),
		Page:        pagination.Page,
		Limit:       pagination.Limit,
		TotalPages:  1,
	}, nil
}

// GetUserStats는 사용자 통계를 조회합니다
func (s *userService) GetUserStats(ctx context.Context) (*models.UserStats, error) {
	// TODO: 실제 통계 계산 로직 구현
	stats := &models.UserStats{
		TotalUsers:      0,
		ActiveUsers:     0,
		VerifiedUsers:   0,
		UsersByRole:     make(map[string]int64),
		UsersByProvider: make(map[string]int64),
		RecentLogins:    0,
	}
	
	return stats, nil
}

// ListUsers는 사용자 목록을 조회합니다 (관리자용)
func (s *userService) ListUsers(ctx context.Context, filter *models.UserFilter) (*models.PaginatedResponse, error) {
	// TODO: 필터링 및 검색 로직 구현
	users := []models.User{}
	err := s.storage.GetAll(ctx, "users", &users)
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}
	
	// User 모델을 UserResponse로 변환
	userResponses := make([]models.UserResponse, len(users))
	for i, user := range users {
		userResponses[i] = *user.ToResponse()
	}
	
	return &models.PaginatedResponse{
		Data:       userResponses,
		Total:      int64(len(users)),
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: 1,
	}, nil
}

// CreateUser는 새 사용자를 생성합니다 (관리자용)
func (s *userService) CreateUser(ctx context.Context, req *models.CreateUserRequest) (*models.UserResponse, error) {
	// 사용자명/이메일 중복 확인
	existingUser := &models.User{}
	err := s.storage.GetByField(ctx, "users", "username", req.Username, existingUser)
	if err == nil {
		return nil, fmt.Errorf("이미 사용 중인 사용자명입니다")
	}
	
	err = s.storage.GetByField(ctx, "users", "email", req.Email, existingUser)
	if err == nil {
		return nil, fmt.Errorf("이미 사용 중인 이메일입니다")
	}
	
	// 비밀번호 해시화
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}
	
	hashedPasswordStr := string(hashedPassword)
	user := &models.User{
		Base: models.Base{
			ID:        generateID(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Username:      req.Username,
		Email:         req.Email,
		PasswordHash:  &hashedPasswordStr,
		DisplayName:   req.DisplayName,
		Role:          req.Role,
		IsActive:      true,
		EmailVerified: false,
	}
	
	err = s.storage.Create(ctx, "users", user)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	
	return user.ToResponse(), nil
}

// GetUser는 특정 사용자를 조회합니다 (관리자용)
func (s *userService) GetUser(ctx context.Context, userID string) (*models.UserResponse, error) {
	user := &models.User{}
	err := s.storage.GetByID(ctx, "users", userID, user)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	
	return user.ToResponse(), nil
}

// UpdateUser는 사용자를 업데이트합니다 (관리자용)
func (s *userService) UpdateUser(ctx context.Context, userID string, req *models.UpdateUserRequest) (*models.UserResponse, error) {
	user := &models.User{}
	err := s.storage.GetByID(ctx, "users", userID, user)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	
	// 업데이트할 필드 적용
	if req.DisplayName != nil {
		user.DisplayName = *req.DisplayName
	}
	if req.ProfilePicture != nil {
		user.ProfilePicture = *req.ProfilePicture
	}
	if req.IsActive != nil {
		user.IsActive = *req.IsActive
	}
	if req.Role != nil {
		user.Role = *req.Role
	}
	
	user.UpdatedAt = time.Now()
	
	err = s.storage.Update(ctx, "users", userID, user)
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}
	
	return user.ToResponse(), nil
}

// DeleteUser는 사용자를 삭제합니다 (관리자용)
func (s *userService) DeleteUser(ctx context.Context, userID string) error {
	// 관련 데이터도 함께 삭제
	s.storage.DeleteByField(ctx, "user_activities", "user_id", userID)
	s.storage.DeleteByField(ctx, "password_reset_tokens", "user_id", userID)
	s.storage.DeleteByField(ctx, "two_factor_secrets", "user_id", userID)
	
	return s.storage.Delete(ctx, "users", userID)
}

// generateID는 새로운 ID를 생성합니다
func generateID() string {
	// 간단한 UUID 생성 (실제로는 google/uuid 등을 사용 권장)
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}