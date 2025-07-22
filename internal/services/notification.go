package services

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"

	"gopkg.in/mail.v2"
)

// NotificationService는 알림 발송을 위한 서비스 인터페이스입니다
type NotificationService interface {
	// 이메일 발송
	SendEmail(ctx context.Context, to, subject, body string) error
	SendPasswordResetEmail(ctx context.Context, to, resetToken string) error
	SendEmailVerificationEmail(ctx context.Context, to, verificationToken string) error
	Send2FAEnabledEmail(ctx context.Context, to string) error
	
	// 푸시 알림 (향후 확장)
	SendPushNotification(ctx context.Context, userID, title, message string) error
}

// notificationService는 NotificationService 인터페이스의 구현체입니다
type notificationService struct {
	smtpHost     string
	smtpPort     int
	smtpUsername string
	smtpPassword string
	fromEmail   string
	fromName    string
}

// NotificationConfig는 알림 서비스 설정입니다
type NotificationConfig struct {
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	FromEmail   string
	FromName    string
}

// NewNotificationService는 새로운 알림 서비스를 생성합니다
func NewNotificationService(config *NotificationConfig) NotificationService {
	return &notificationService{
		smtpHost:     config.SMTPHost,
		smtpPort:     config.SMTPPort,
		smtpUsername: config.SMTPUsername,
		smtpPassword: config.SMTPPassword,
		fromEmail:   config.FromEmail,
		fromName:    config.FromName,
	}
}

// SendEmail은 이메일을 발송합니다
func (s *notificationService) SendEmail(ctx context.Context, to, subject, body string) error {
	m := mail.NewMessage()
	m.SetHeader("From", s.fromEmail)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)

	d := mail.NewDialer(s.smtpHost, s.smtpPort, s.smtpUsername, s.smtpPassword)
	
	if err := d.DialAndSend(m); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}
	
	return nil
}

// SendPasswordResetEmail은 비밀번호 재설정 이메일을 발송합니다
func (s *notificationService) SendPasswordResetEmail(ctx context.Context, to, resetToken string) error {
	subject := "비밀번호 재설정 요청"
	
	// HTML 템플릿 생성
	body := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>비밀번호 재설정</title>
    <style>
        .container { max-width: 600px; margin: 0 auto; font-family: Arial, sans-serif; }
        .header { background-color: #4CAF50; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; background-color: #f9f9f9; }
        .button { 
            display: inline-block; 
            background-color: #4CAF50; 
            color: white; 
            padding: 12px 24px; 
            text-decoration: none; 
            border-radius: 4px; 
            margin: 20px 0;
        }
        .footer { padding: 20px; font-size: 12px; color: #666; text-align: center; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>AICode Manager</h1>
        </div>
        <div class="content">
            <h2>비밀번호 재설정 요청</h2>
            <p>안녕하세요,</p>
            <p>귀하의 계정에 대한 비밀번호 재설정 요청이 접수되었습니다.</p>
            <p>아래 버튼을 클릭하여 비밀번호를 재설정하세요:</p>
            
            <a href="http://localhost:3000/auth/reset-password?token=%s" class="button">비밀번호 재설정</a>
            
            <p>이 링크는 24시간 후에 만료됩니다.</p>
            <p>만약 비밀번호 재설정을 요청하지 않으셨다면, 이 이메일을 무시하세요.</p>
        </div>
        <div class="footer">
            <p>이 이메일은 자동으로 생성되었습니다. 답장하지 마세요.</p>
        </div>
    </div>
</body>
</html>
	`, resetToken)

	return s.SendEmail(ctx, to, subject, body)
}

// SendEmailVerificationEmail은 이메일 인증 메일을 발송합니다
func (s *notificationService) SendEmailVerificationEmail(ctx context.Context, to, verificationToken string) error {
	subject := "이메일 주소 인증"
	
	body := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>이메일 인증</title>
    <style>
        .container { max-width: 600px; margin: 0 auto; font-family: Arial, sans-serif; }
        .header { background-color: #2196F3; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; background-color: #f9f9f9; }
        .button { 
            display: inline-block; 
            background-color: #2196F3; 
            color: white; 
            padding: 12px 24px; 
            text-decoration: none; 
            border-radius: 4px; 
            margin: 20px 0;
        }
        .footer { padding: 20px; font-size: 12px; color: #666; text-align: center; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>AICode Manager</h1>
        </div>
        <div class="content">
            <h2>이메일 주소 인증</h2>
            <p>안녕하세요,</p>
            <p>AICode Manager에 가입해 주셔서 감사합니다!</p>
            <p>아래 버튼을 클릭하여 이메일 주소를 인증해 주세요:</p>
            
            <a href="http://localhost:3000/auth/verify-email?token=%s" class="button">이메일 인증</a>
            
            <p>이 링크는 24시간 후에 만료됩니다.</p>
        </div>
        <div class="footer">
            <p>이 이메일은 자동으로 생성되었습니다. 답장하지 마세요.</p>
        </div>
    </div>
</body>
</html>
	`, verificationToken)

	return s.SendEmail(ctx, to, subject, body)
}

// Send2FAEnabledEmail은 2FA 활성화 알림 이메일을 발송합니다
func (s *notificationService) Send2FAEnabledEmail(ctx context.Context, to string) error {
	subject := "2단계 인증이 활성화되었습니다"
	
	body := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>2FA 활성화</title>
    <style>
        .container { max-width: 600px; margin: 0 auto; font-family: Arial, sans-serif; }
        .header { background-color: #FF9800; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; background-color: #f9f9f9; }
        .alert { 
            background-color: #fff3cd; 
            border: 1px solid #ffeaa7; 
            color: #856404; 
            padding: 15px; 
            border-radius: 4px; 
            margin: 20px 0;
        }
        .footer { padding: 20px; font-size: 12px; color: #666; text-align: center; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>AICode Manager</h1>
        </div>
        <div class="content">
            <h2>2단계 인증 활성화</h2>
            <p>안녕하세요,</p>
            <p>귀하의 계정에 2단계 인증(2FA)이 성공적으로 활성화되었습니다.</p>
            
            <div class="alert">
                <strong>보안 알림:</strong> 이제 로그인할 때 인증 앱에서 생성된 6자리 코드가 필요합니다.
            </div>
            
            <p>2단계 인증을 활성화하지 않으셨다면, 즉시 계정의 보안을 확인하고 비밀번호를 변경하세요.</p>
            
            <p><strong>백업 코드를 안전한 곳에 보관하는 것을 잊지 마세요!</strong></p>
        </div>
        <div class="footer">
            <p>이 이메일은 자동으로 생성되었습니다. 답장하지 마세요.</p>
        </div>
    </div>
</body>
</html>
	`

	return s.SendEmail(ctx, to, subject, body)
}

// SendPushNotification은 푸시 알림을 발송합니다 (향후 구현)
func (s *notificationService) SendPushNotification(ctx context.Context, userID, title, message string) error {
	// TODO: Firebase Cloud Messaging 또는 다른 푸시 알림 서비스 연동
	return fmt.Errorf("push notification not implemented yet")
}

// ===== 헬퍼 함수들 =====

// validateEmailFormat은 이메일 형식을 검증합니다
func validateEmailFormat(email string) bool {
	return strings.Contains(email, "@") && strings.Contains(email, ".")
}

// MockNotificationService는 테스트용 모의 알림 서비스입니다
type MockNotificationService struct {
	SentEmails []EmailRecord
}

type EmailRecord struct {
	To      string
	Subject string
	Body    string
}

func NewMockNotificationService() *MockNotificationService {
	return &MockNotificationService{
		SentEmails: make([]EmailRecord, 0),
	}
}

func (m *MockNotificationService) SendEmail(ctx context.Context, to, subject, body string) error {
	m.SentEmails = append(m.SentEmails, EmailRecord{
		To:      to,
		Subject: subject,
		Body:    body,
	})
	return nil
}

func (m *MockNotificationService) SendPasswordResetEmail(ctx context.Context, to, resetToken string) error {
	return m.SendEmail(ctx, to, "비밀번호 재설정 요청", "Reset token: "+resetToken)
}

func (m *MockNotificationService) SendEmailVerificationEmail(ctx context.Context, to, verificationToken string) error {
	return m.SendEmail(ctx, to, "이메일 주소 인증", "Verification token: "+verificationToken)
}

func (m *MockNotificationService) Send2FAEnabledEmail(ctx context.Context, to string) error {
	return m.SendEmail(ctx, to, "2단계 인증이 활성화되었습니다", "2FA enabled")
}

func (m *MockNotificationService) SendPushNotification(ctx context.Context, userID, title, message string) error {
	return nil // 테스트에서는 무시
}