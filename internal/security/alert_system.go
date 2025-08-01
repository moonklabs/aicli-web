package security

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/smtp"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// AlertSystemConfig는 알림 시스템 설정입니다.
type AlertSystemConfig struct {
	// 기본 설정
	Enabled          bool     // 알림 시스템 활성화
	MinSeverity      Severity // 최소 알림 심각도
	MaxAlertsPerHour int      // 시간당 최대 알림 수

	// 이메일 설정
	EmailConfig *EmailConfig `json:"email_config,omitempty"`

	// 슬랙 설정
	SlackConfig *SlackConfig `json:"slack_config,omitempty"`

	// 웹훅 설정
	WebhookConfig *WebhookConfig `json:"webhook_config,omitempty"`

	// SMS 설정
	SMSConfig *SMSConfig `json:"sms_config,omitempty"`

	// 자동 대응 설정
	AutoResponseEnabled bool                // 자동 대응 활성화
	ResponseRules       []*AutoResponseRule // 자동 대응 규칙
	EscalationRules     []*EscalationRule   // 에스컬레이션 규칙

	Logger *zap.Logger
}

// EmailConfig는 이메일 알림 설정입니다.
type EmailConfig struct {
	Enabled    bool     `json:"enabled"`
	SMTPServer string   `json:"smtp_server"`
	SMTPPort   int      `json:"smtp_port"`
	Username   string   `json:"username"`
	Password   string   `json:"password"`
	FromEmail  string   `json:"from_email"`
	ToEmails   []string `json:"to_emails"`
	UseTLS     bool     `json:"use_tls"`
}

// SlackConfig는 슬랙 알림 설정입니다.
type SlackConfig struct {
	Enabled    bool   `json:"enabled"`
	WebhookURL string `json:"webhook_url"`
	Channel    string `json:"channel"`
	Username   string `json:"username"`
	IconEmoji  string `json:"icon_emoji"`
}

// WebhookConfig는 웹훅 알림 설정입니다.
type WebhookConfig struct {
	Enabled    bool              `json:"enabled"`
	URL        string            `json:"url"`
	Method     string            `json:"method"`
	Headers    map[string]string `json:"headers"`
	Timeout    time.Duration     `json:"timeout"`
	RetryCount int               `json:"retry_count"`
}

// SMSConfig는 SMS 알림 설정입니다.
type SMSConfig struct {
	Enabled    bool     `json:"enabled"`
	Provider   string   `json:"provider"` // "twilio", "aws_sns" 등
	APIKey     string   `json:"api_key"`
	APISecret  string   `json:"api_secret"`
	FromNumber string   `json:"from_number"`
	ToNumbers  []string `json:"to_numbers"`
}

// AutoResponseRule은 자동 대응 규칙입니다.
type AutoResponseRule struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Enabled       bool              `json:"enabled"`
	EventTypes    []EventType       `json:"event_types"`
	Severities    []Severity        `json:"severities"`
	Conditions    map[string]string `json:"conditions"`     // 추가 조건
	Actions       []*ResponseAction `json:"actions"`        // 수행할 액션들
	Cooldown      time.Duration     `json:"cooldown"`       // 재실행 방지 시간
	MaxExecutions int               `json:"max_executions"` // 최대 실행 횟수
}

// ResponseAction은 대응 액션입니다.
type ResponseAction struct {
	Type       string                 `json:"type"`       // "block_ip", "block_user", "rate_limit" 등
	Parameters map[string]interface{} `json:"parameters"` // 액션별 파라미터
	Delay      time.Duration          `json:"delay"`      // 실행 지연 시간
}

// EscalationRule은 에스컬레이션 규칙입니다.
type EscalationRule struct {
	ID                   string        `json:"id"`
	Name                 string        `json:"name"`
	Enabled              bool          `json:"enabled"`
	TriggerCondition     string        `json:"trigger_condition"`     // "event_count", "time_threshold" 등
	Threshold            int           `json:"threshold"`             // 임계값
	TimeWindow           time.Duration `json:"time_window"`           // 시간 윈도우
	EscalationDelay      time.Duration `json:"escalation_delay"`      // 에스컬레이션 지연
	NotificationChannels []string      `json:"notification_channels"` // 알림 채널
}

// AlertSystem은 보안 알림 시스템입니다.
type AlertSystem struct {
	config *AlertSystemConfig
	logger *zap.Logger

	// 알림 제한 관리
	alertCounts map[string]int // 시간당 알림 수 추적
	alertMutex  sync.RWMutex

	// 자동 대응 실행 추적
	executionCounts map[string]int       // 규칙별 실행 횟수
	lastExecutions  map[string]time.Time // 마지막 실행 시간
	responseMutex   sync.RWMutex

	// HTTP 클라이언트
	httpClient *http.Client
}

// AlertContext는 알림 컨텍스트입니다.
type AlertContext struct {
	Event     *SecurityEvent         `json:"event"`
	Timestamp time.Time              `json:"timestamp"`
	Severity  Severity               `json:"severity"`
	Summary   string                 `json:"summary"`
	Details   map[string]interface{} `json:"details"`
	Metadata  map[string]string      `json:"metadata"`
}

// NewAlertSystem은 새로운 알림 시스템을 생성합니다.
func NewAlertSystem(config *AlertSystemConfig) *AlertSystem {
	if config == nil {
		config = &AlertSystemConfig{
			Enabled:             true,
			MinSeverity:         SeverityMedium,
			MaxAlertsPerHour:    100,
			AutoResponseEnabled: true,
			ResponseRules:       make([]*AutoResponseRule, 0),
			EscalationRules:     make([]*EscalationRule, 0),
		}
	}

	return &AlertSystem{
		config:          config,
		logger:          config.Logger,
		alertCounts:     make(map[string]int),
		executionCounts: make(map[string]int),
		lastExecutions:  make(map[string]time.Time),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ProcessEvent는 보안 이벤트를 처리하고 필요시 알림을 발송합니다.
func (as *AlertSystem) ProcessEvent(ctx context.Context, event *SecurityEvent) error {
	if !as.config.Enabled {
		return nil
	}

	// 심각도 확인
	if !as.shouldAlert(event.Severity) {
		return nil
	}

	// 알림 제한 확인
	if !as.checkAlertLimit(event) {
		as.logger.Warn("알림 제한 초과",
			zap.String("event_id", event.ID),
			zap.String("severity", string(event.Severity)))
		return nil
	}

	// 알림 컨텍스트 생성
	alertCtx := as.createAlertContext(event)

	// 알림 발송
	as.sendAlert(ctx, alertCtx)

	// 자동 대응 처리
	if as.config.AutoResponseEnabled {
		as.executeAutoResponse(ctx, event)
	}

	// 에스컬레이션 검사
	as.checkEscalation(ctx, event)

	return nil
}

// shouldAlert는 알림을 발송할지 결정합니다.
func (as *AlertSystem) shouldAlert(severity Severity) bool {
	severityLevels := map[Severity]int{
		SeverityLow:      1,
		SeverityMedium:   2,
		SeverityHigh:     3,
		SeverityCritical: 4,
	}

	minLevel := severityLevels[as.config.MinSeverity]
	eventLevel := severityLevels[severity]

	return eventLevel >= minLevel
}

// checkAlertLimit는 알림 제한을 확인합니다.
func (as *AlertSystem) checkAlertLimit(event *SecurityEvent) bool {
	as.alertMutex.Lock()
	defer as.alertMutex.Unlock()

	hour := time.Now().Hour()
	key := fmt.Sprintf("%s_%d", string(event.Type), hour)

	count := as.alertCounts[key]
	if count >= as.config.MaxAlertsPerHour {
		return false
	}

	as.alertCounts[key] = count + 1
	return true
}

// createAlertContext는 알림 컨텍스트를 생성합니다.
func (as *AlertSystem) createAlertContext(event *SecurityEvent) *AlertContext {
	summary := as.generateSummary(event)

	return &AlertContext{
		Event:     event,
		Timestamp: time.Now(),
		Severity:  event.Severity,
		Summary:   summary,
		Details:   event.Details,
		Metadata: map[string]string{
			"event_id":   event.ID,
			"event_type": string(event.Type),
			"source":     event.Source,
			"target":     event.Target,
		},
	}
}

// generateSummary는 이벤트 요약을 생성합니다.
func (as *AlertSystem) generateSummary(event *SecurityEvent) string {
	switch event.Type {
	case EventTypeAIPromptInjection:
		return fmt.Sprintf("AI 프롬프트 인젝션 공격 탐지 - 사용자: %s", event.UserID)
	case EventTypeAIJailbreakAttempt:
		return fmt.Sprintf("AI 제한 우회 시도 탐지 - 사용자: %s", event.UserID)
	case EventTypeAIDataExtraction:
		return fmt.Sprintf("AI를 통한 데이터 추출 시도 - 사용자: %s", event.UserID)
	case EventTypeRateLimit:
		return fmt.Sprintf("Rate Limit 위반 - IP: %s", event.IPAddress)
	case EventTypeBlocked:
		return fmt.Sprintf("접근 차단됨 - 대상: %s", event.Source)
	case EventTypeAnomalous:
		return fmt.Sprintf("이상 행동 탐지 - 사용자: %s", event.UserID)
	default:
		return fmt.Sprintf("보안 이벤트 발생 - 타입: %s", event.Type)
	}
}

// sendAlert는 설정된 채널로 알림을 발송합니다.
func (as *AlertSystem) sendAlert(ctx context.Context, alertCtx *AlertContext) {
	// 이메일 알림
	if as.config.EmailConfig != nil && as.config.EmailConfig.Enabled {
		go as.sendEmailAlert(alertCtx)
	}

	// 슬랙 알림
	if as.config.SlackConfig != nil && as.config.SlackConfig.Enabled {
		go as.sendSlackAlert(alertCtx)
	}

	// 웹훅 알림
	if as.config.WebhookConfig != nil && as.config.WebhookConfig.Enabled {
		go as.sendWebhookAlert(alertCtx)
	}

	// SMS 알림 (Critical 등급만)
	if as.config.SMSConfig != nil && as.config.SMSConfig.Enabled && alertCtx.Severity == SeverityCritical {
		go as.sendSMSAlert(alertCtx)
	}
}

// sendEmailAlert는 이메일 알림을 발송합니다.
func (as *AlertSystem) sendEmailAlert(alertCtx *AlertContext) {
	config := as.config.EmailConfig

	subject := fmt.Sprintf("[보안알림] %s - %s", strings.ToUpper(string(alertCtx.Severity)), alertCtx.Summary)
	body := as.generateEmailBody(alertCtx)

	auth := smtp.PlainAuth("", config.Username, config.Password, config.SMTPServer)

	for _, toEmail := range config.ToEmails {
		message := fmt.Sprintf("From: %s\nTo: %s\nSubject: %s\n\n%s",
			config.FromEmail, toEmail, subject, body)

		addr := fmt.Sprintf("%s:%d", config.SMTPServer, config.SMTPPort)
		err := smtp.SendMail(addr, auth, config.FromEmail, []string{toEmail}, []byte(message))
		if err != nil {
			as.logger.Error("이메일 발송 실패",
				zap.String("to", toEmail),
				zap.Error(err))
		} else {
			as.logger.Info("이메일 알림 발송됨", zap.String("to", toEmail))
		}
	}
}

// sendSlackAlert는 슬랙 알림을 발송합니다.
func (as *AlertSystem) sendSlackAlert(alertCtx *AlertContext) {
	config := as.config.SlackConfig

	color := as.getSeverityColor(alertCtx.Severity)

	payload := map[string]interface{}{
		"channel":    config.Channel,
		"username":   config.Username,
		"icon_emoji": config.IconEmoji,
		"attachments": []map[string]interface{}{
			{
				"color":     color,
				"title":     alertCtx.Summary,
				"text":      as.generateSlackText(alertCtx),
				"timestamp": alertCtx.Timestamp.Unix(),
				"fields": []map[string]interface{}{
					{
						"title": "심각도",
						"value": strings.ToUpper(string(alertCtx.Severity)),
						"short": true,
					},
					{
						"title": "이벤트 ID",
						"value": alertCtx.Event.ID,
						"short": true,
					},
				},
			},
		},
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		as.logger.Error("슬랙 페이로드 생성 실패", zap.Error(err))
		return
	}

	resp, err := as.httpClient.Post(config.WebhookURL, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		as.logger.Error("슬랙 알림 발송 실패", zap.Error(err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		as.logger.Info("슬랙 알림 발송됨")
	} else {
		as.logger.Error("슬랙 알림 발송 실패", zap.Int("status_code", resp.StatusCode))
	}
}

// sendWebhookAlert는 웹훅 알림을 발송합니다.
func (as *AlertSystem) sendWebhookAlert(alertCtx *AlertContext) {
	config := as.config.WebhookConfig

	payload, err := json.Marshal(alertCtx)
	if err != nil {
		as.logger.Error("웹훅 페이로드 생성 실패", zap.Error(err))
		return
	}

	req, err := http.NewRequest(config.Method, config.URL, bytes.NewBuffer(payload))
	if err != nil {
		as.logger.Error("웹훅 요청 생성 실패", zap.Error(err))
		return
	}

	// 헤더 설정
	req.Header.Set("Content-Type", "application/json")
	for key, value := range config.Headers {
		req.Header.Set(key, value)
	}

	// 재시도 로직
	for i := 0; i <= config.RetryCount; i++ {
		resp, err := as.httpClient.Do(req)
		if err != nil {
			as.logger.Error("웹훅 요청 실패",
				zap.Int("attempt", i+1),
				zap.Error(err))

			if i < config.RetryCount {
				time.Sleep(time.Duration(i+1) * time.Second)
				continue
			}
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			as.logger.Info("웹훅 알림 발송됨", zap.Int("status_code", resp.StatusCode))
			return
		}

		as.logger.Error("웹훅 응답 오류",
			zap.Int("status_code", resp.StatusCode),
			zap.Int("attempt", i+1))

		if i < config.RetryCount {
			time.Sleep(time.Duration(i+1) * time.Second)
		}
	}
}

// sendSMSAlert는 SMS 알림을 발송합니다.
func (as *AlertSystem) sendSMSAlert(alertCtx *AlertContext) {
	// SMS 구현은 실제 서비스 제공업체에 따라 달라짐
	// 여기서는 로그만 남김
	as.logger.Info("SMS 알림 발송 요청",
		zap.String("severity", string(alertCtx.Severity)),
		zap.String("summary", alertCtx.Summary))
}

// executeAutoResponse는 자동 대응을 실행합니다.
func (as *AlertSystem) executeAutoResponse(ctx context.Context, event *SecurityEvent) {
	for _, rule := range as.config.ResponseRules {
		if as.shouldExecuteRule(rule, event) {
			as.executeRule(ctx, rule, event)
		}
	}
}

// shouldExecuteRule은 규칙을 실행할지 결정합니다.
func (as *AlertSystem) shouldExecuteRule(rule *AutoResponseRule, event *SecurityEvent) bool {
	if !rule.Enabled {
		return false
	}

	// 이벤트 타입 확인
	found := false
	for _, eventType := range rule.EventTypes {
		if eventType == event.Type {
			found = true
			break
		}
	}
	if !found {
		return false
	}

	// 심각도 확인
	found = false
	for _, severity := range rule.Severities {
		if severity == event.Severity {
			found = true
			break
		}
	}
	if !found {
		return false
	}

	// 쿨다운 확인
	as.responseMutex.RLock()
	lastExecution := as.lastExecutions[rule.ID]
	executionCount := as.executionCounts[rule.ID]
	as.responseMutex.RUnlock()

	if time.Since(lastExecution) < rule.Cooldown {
		return false
	}

	// 최대 실행 횟수 확인
	if rule.MaxExecutions > 0 && executionCount >= rule.MaxExecutions {
		return false
	}

	return true
}

// executeRule은 자동 대응 규칙을 실행합니다.
func (as *AlertSystem) executeRule(ctx context.Context, rule *AutoResponseRule, event *SecurityEvent) {
	as.responseMutex.Lock()
	as.lastExecutions[rule.ID] = time.Now()
	as.executionCounts[rule.ID]++
	as.responseMutex.Unlock()

	as.logger.Info("자동 대응 규칙 실행",
		zap.String("rule_id", rule.ID),
		zap.String("rule_name", rule.Name),
		zap.String("event_id", event.ID))

	for _, action := range rule.Actions {
		if action.Delay > 0 {
			time.Sleep(action.Delay)
		}

		err := as.executeAction(ctx, action, event)
		if err != nil {
			as.logger.Error("자동 대응 액션 실행 실패",
				zap.String("rule_id", rule.ID),
				zap.String("action_type", action.Type),
				zap.Error(err))
		}
	}
}

// executeAction은 대응 액션을 실행합니다.
func (as *AlertSystem) executeAction(ctx context.Context, action *ResponseAction, event *SecurityEvent) error {
	switch action.Type {
	case "block_ip":
		// IP 차단 로직 (실제 구현에서는 Rate Limiter와 연동)
		as.logger.Info("IP 차단 액션", zap.String("ip", event.IPAddress))
		return nil

	case "block_user":
		// 사용자 차단 로직
		as.logger.Info("사용자 차단 액션", zap.String("user_id", event.UserID))
		return nil

	case "rate_limit":
		// Rate Limit 강화 로직
		as.logger.Info("Rate Limit 강화 액션", zap.String("user_id", event.UserID))
		return nil

	case "session_invalidate":
		// 세션 무효화 로직
		as.logger.Info("세션 무효화 액션", zap.String("session_id", event.SessionID))
		return nil

	default:
		return fmt.Errorf("알 수 없는 액션 타입: %s", action.Type)
	}
}

// checkEscalation은 에스컬레이션을 확인합니다.
func (as *AlertSystem) checkEscalation(ctx context.Context, event *SecurityEvent) {
	// 에스컬레이션 로직 구현
	// 여기서는 간단한 로그만 남김
	as.logger.Debug("에스컬레이션 확인", zap.String("event_id", event.ID))
}

// 헬퍼 메서드들

func (as *AlertSystem) generateEmailBody(alertCtx *AlertContext) string {
	return fmt.Sprintf(`
보안 알림

시간: %s
심각도: %s
요약: %s

이벤트 정보:
- ID: %s
- 타입: %s
- 소스: %s
- 대상: %s
- 사용자 ID: %s
- IP 주소: %s

상세 정보:
%v

이 알림은 자동으로 생성되었습니다.
`,
		alertCtx.Timestamp.Format("2006-01-02 15:04:05"),
		strings.ToUpper(string(alertCtx.Severity)),
		alertCtx.Summary,
		alertCtx.Event.ID,
		string(alertCtx.Event.Type),
		alertCtx.Event.Source,
		alertCtx.Event.Target,
		alertCtx.Event.UserID,
		alertCtx.Event.IPAddress,
		alertCtx.Details,
	)
}

func (as *AlertSystem) generateSlackText(alertCtx *AlertContext) string {
	return fmt.Sprintf("시간: %s\n소스: %s\n대상: %s\n사용자: %s\nIP: %s",
		alertCtx.Timestamp.Format("2006-01-02 15:04:05"),
		alertCtx.Event.Source,
		alertCtx.Event.Target,
		alertCtx.Event.UserID,
		alertCtx.Event.IPAddress,
	)
}

func (as *AlertSystem) getSeverityColor(severity Severity) string {
	switch severity {
	case SeverityCritical:
		return "danger"
	case SeverityHigh:
		return "warning"
	case SeverityMedium:
		return "good"
	case SeverityLow:
		return "#439FE0"
	default:
		return "good"
	}
}

// GetStats는 알림 시스템 통계를 반환합니다.
func (as *AlertSystem) GetStats() map[string]interface{} {
	as.alertMutex.RLock()
	defer as.alertMutex.RUnlock()

	as.responseMutex.RLock()
	defer as.responseMutex.RUnlock()

	totalAlerts := 0
	for _, count := range as.alertCounts {
		totalAlerts += count
	}

	totalExecutions := 0
	for _, count := range as.executionCounts {
		totalExecutions += count
	}

	return map[string]interface{}{
		"enabled":               as.config.Enabled,
		"total_alerts_today":    totalAlerts,
		"auto_response_enabled": as.config.AutoResponseEnabled,
		"total_auto_responses":  totalExecutions,
		"active_rules":          len(as.config.ResponseRules),
		"escalation_rules":      len(as.config.EscalationRules),
	}
}
