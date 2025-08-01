package monitoring

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// AlertingSystem은 실시간 알림 및 경고 시스템입니다
type AlertingSystem struct {
	alerts         map[string]*Alert
	rules          []AlertRule
	channels       map[string]AlertChannel
	mutex          sync.RWMutex
	config         AlertingConfig
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
	lastEvaluation time.Time
}

// AlertingConfig는 알림 시스템 설정입니다
type AlertingConfig struct {
	EvaluationInterval  time.Duration `json:"evaluation_interval"`
	DefaultChannel      string        `json:"default_channel"`
	EnableEmailAlerts   bool          `json:"enable_email_alerts"`
	EnableSlackAlerts   bool          `json:"enable_slack_alerts"`
	EnableWebhookAlerts bool          `json:"enable_webhook_alerts"`
	RetentionPeriod     time.Duration `json:"retention_period"`
	MaxAlerts           int           `json:"max_alerts"`
}

// Alert는 개별 알림입니다
type Alert struct {
	ID          string             `json:"id"`
	RuleID      string             `json:"rule_id"`
	Name        string             `json:"name"`
	Message     string             `json:"message"`
	Severity    AlertSeverity      `json:"severity"`
	Status      AlertStatus        `json:"status"`
	Labels      map[string]string  `json:"labels"`
	Annotations map[string]string  `json:"annotations"`
	StartsAt    time.Time          `json:"starts_at"`
	EndsAt      *time.Time         `json:"ends_at,omitempty"`
	LastSentAt  *time.Time         `json:"last_sent_at,omitempty"`
	SentCount   int                `json:"sent_count"`
	Metrics     map[string]float64 `json:"metrics,omitempty"`
}

// AlertRule은 알림 발생 규칙입니다
type AlertRule struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Description     string            `json:"description"`
	Query           string            `json:"query"`
	Condition       AlertCondition    `json:"condition"`
	Severity        AlertSeverity     `json:"severity"`
	For             time.Duration     `json:"for"` // 알림 발생 전 대기 시간
	Labels          map[string]string `json:"labels"`
	Annotations     map[string]string `json:"annotations"`
	Channels        []string          `json:"channels"`
	Enabled         bool              `json:"enabled"`
	LastEvaluation  time.Time         `json:"last_evaluation"`
	EvaluationCount int64             `json:"evaluation_count"`
}

// AlertCondition은 알림 발생 조건입니다
type AlertCondition struct {
	Operator  ConditionOperator `json:"operator"`
	Threshold float64           `json:"threshold"`
	Metric    string            `json:"metric"`
	Window    time.Duration     `json:"window"`
}

// ConditionOperator는 조건 연산자입니다
type ConditionOperator string

const (
	OperatorGreaterThan    ConditionOperator = "gt"
	OperatorLessThan       ConditionOperator = "lt"
	OperatorEquals         ConditionOperator = "eq"
	OperatorNotEquals      ConditionOperator = "ne"
	OperatorGreaterOrEqual ConditionOperator = "gte"
	OperatorLessOrEqual    ConditionOperator = "lte"
)

// AlertSeverity는 알림 심각도입니다
type AlertSeverity string

const (
	SeverityInfo     AlertSeverity = "info"
	SeverityWarning  AlertSeverity = "warning"
	SeverityError    AlertSeverity = "error"
	SeverityCritical AlertSeverity = "critical"
)

// AlertStatus는 알림 상태입니다
type AlertStatus string

const (
	StatusFiring   AlertStatus = "firing"
	StatusResolved AlertStatus = "resolved"
	StatusSilenced AlertStatus = "silenced"
)

// AlertChannel은 알림 채널 인터페이스입니다
type AlertChannel interface {
	Send(alert *Alert) error
	GetType() string
	IsEnabled() bool
}

// EmailChannel은 이메일 알림 채널입니다
type EmailChannel struct {
	SMTPHost  string   `json:"smtp_host"`
	SMTPPort  int      `json:"smtp_port"`
	Username  string   `json:"username"`
	Password  string   `json:"password"`
	FromEmail string   `json:"from_email"`
	ToEmails  []string `json:"to_emails"`
	Enabled   bool     `json:"enabled"`
}

// SlackChannel은 Slack 알림 채널입니다
type SlackChannel struct {
	WebhookURL string `json:"webhook_url"`
	Channel    string `json:"channel"`
	Username   string `json:"username"`
	Enabled    bool   `json:"enabled"`
}

// WebhookChannel은 웹훅 알림 채널입니다
type WebhookChannel struct {
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Method  string            `json:"method"`
	Enabled bool              `json:"enabled"`
}

// LogChannel은 로그 알림 채널입니다
type LogChannel struct {
	Enabled bool `json:"enabled"`
}

// AlertingMetrics는 알림 시스템 메트릭입니다
type AlertingMetrics struct {
	ActiveAlerts       int           `json:"active_alerts"`
	TotalAlerts        int64         `json:"total_alerts"`
	ResolvedAlerts     int64         `json:"resolved_alerts"`
	RuleCount          int           `json:"rule_count"`
	ChannelCount       int           `json:"channel_count"`
	LastEvaluation     time.Time     `json:"last_evaluation"`
	EvaluationDuration time.Duration `json:"evaluation_duration"`
}

// DefaultAlertingConfig는 기본 알림 설정을 반환합니다
func DefaultAlertingConfig() AlertingConfig {
	return AlertingConfig{
		EvaluationInterval:  30 * time.Second,
		DefaultChannel:      "log",
		EnableEmailAlerts:   false,
		EnableSlackAlerts:   false,
		EnableWebhookAlerts: false,
		RetentionPeriod:     24 * time.Hour,
		MaxAlerts:           1000,
	}
}

// NewAlertingSystem은 새로운 알림 시스템을 생성합니다
func NewAlertingSystem(config AlertingConfig) *AlertingSystem {
	ctx, cancel := context.WithCancel(context.Background())

	system := &AlertingSystem{
		alerts:   make(map[string]*Alert),
		rules:    make([]AlertRule, 0),
		channels: make(map[string]AlertChannel),
		config:   config,
		ctx:      ctx,
		cancel:   cancel,
	}

	// 기본 로그 채널 추가
	system.AddChannel("log", &LogChannel{Enabled: true})

	// 평가 루프 시작
	system.startEvaluationLoop()

	return system
}

// AddRule은 알림 발생 규칙을 추가합니다
func (as *AlertingSystem) AddRule(rule AlertRule) {
	as.mutex.Lock()
	defer as.mutex.Unlock()

	// 기본값 설정
	if rule.ID == "" {
		rule.ID = fmt.Sprintf("rule_%d", time.Now().Unix())
	}
	if len(rule.Channels) == 0 {
		rule.Channels = []string{as.config.DefaultChannel}
	}
	if rule.Labels == nil {
		rule.Labels = make(map[string]string)
	}
	if rule.Annotations == nil {
		rule.Annotations = make(map[string]string)
	}

	as.rules = append(as.rules, rule)
}

// RemoveRule은 알림 발생 규칙을 제거합니다
func (as *AlertingSystem) RemoveRule(ruleID string) {
	as.mutex.Lock()
	defer as.mutex.Unlock()

	for i, rule := range as.rules {
		if rule.ID == ruleID {
			as.rules = append(as.rules[:i], as.rules[i+1:]...)
			break
		}
	}
}

// AddChannel은 알림 채널을 추가합니다
func (as *AlertingSystem) AddChannel(name string, channel AlertChannel) {
	as.mutex.Lock()
	defer as.mutex.Unlock()

	as.channels[name] = channel
}

// RemoveChannel은 알림 채널을 제거합니다
func (as *AlertingSystem) RemoveChannel(name string) {
	as.mutex.Lock()
	defer as.mutex.Unlock()

	delete(as.channels, name)
}

// FireAlert는 알림을 발생시킵니다
func (as *AlertingSystem) FireAlert(ruleID string, message string, labels map[string]string, metrics map[string]float64) {
	as.mutex.Lock()
	defer as.mutex.Unlock()

	// 규칙 찾기
	var rule *AlertRule
	for i := range as.rules {
		if as.rules[i].ID == ruleID {
			rule = &as.rules[i]
			break
		}
	}

	if rule == nil {
		log.Printf("Alert rule not found: %s", ruleID)
		return
	}

	// 알림 ID 생성
	alertID := fmt.Sprintf("%s_%d", ruleID, time.Now().Unix())

	// 알림 생성
	alert := &Alert{
		ID:          alertID,
		RuleID:      ruleID,
		Name:        rule.Name,
		Message:     message,
		Severity:    rule.Severity,
		Status:      StatusFiring,
		Labels:      mergeLabels(rule.Labels, labels),
		Annotations: rule.Annotations,
		StartsAt:    time.Now(),
		Metrics:     metrics,
	}

	as.alerts[alertID] = alert

	// 알림 전송
	as.sendAlert(alert, rule.Channels)
}

// AddAlert는 알림을 직접 추가합니다
func (as *AlertingSystem) AddAlert(alert Alert) {
	as.mutex.Lock()
	defer as.mutex.Unlock()

	// ID가 없으면 생성
	if alert.ID == "" {
		alert.ID = fmt.Sprintf("alert-%d", time.Now().UnixNano())
	}

	as.alerts[alert.ID] = &alert
}

// GetAlerts는 모든 알림을 반환합니다
func (as *AlertingSystem) GetAlerts() []*Alert {
	return as.GetAllAlerts()
}

// ResolveAlert는 알림을 해결됨으로 표시합니다
func (as *AlertingSystem) ResolveAlert(alertID string) {
	as.mutex.Lock()
	defer as.mutex.Unlock()

	alert, exists := as.alerts[alertID]
	if !exists {
		return
	}

	now := time.Now()
	alert.Status = StatusResolved
	alert.EndsAt = &now

	// 해결 알림 전송
	var rule *AlertRule
	for i := range as.rules {
		if as.rules[i].ID == alert.RuleID {
			rule = &as.rules[i]
			break
		}
	}

	if rule != nil {
		as.sendAlert(alert, rule.Channels)
	}
}

// SilenceAlert는 알림을 음소거합니다
func (as *AlertingSystem) SilenceAlert(alertID string) {
	as.mutex.Lock()
	defer as.mutex.Unlock()

	alert, exists := as.alerts[alertID]
	if !exists {
		return
	}

	alert.Status = StatusSilenced
}

// GetActiveAlerts는 활성 알림들을 반환합니다
func (as *AlertingSystem) GetActiveAlerts() []*Alert {
	as.mutex.RLock()
	defer as.mutex.RUnlock()

	alerts := make([]*Alert, 0)
	for _, alert := range as.alerts {
		if alert.Status == StatusFiring {
			copy := *alert
			alerts = append(alerts, &copy)
		}
	}

	return alerts
}

// GetAllAlerts는 모든 알림들을 반환합니다
func (as *AlertingSystem) GetAllAlerts() []*Alert {
	as.mutex.RLock()
	defer as.mutex.RUnlock()

	alerts := make([]*Alert, 0, len(as.alerts))
	for _, alert := range as.alerts {
		copy := *alert
		alerts = append(alerts, &copy)
	}

	// 시작 시간순으로 정렬 (최신순)
	for i := 0; i < len(alerts)-1; i++ {
		for j := i + 1; j < len(alerts); j++ {
			if alerts[i].StartsAt.Before(alerts[j].StartsAt) {
				alerts[i], alerts[j] = alerts[j], alerts[i]
			}
		}
	}

	return alerts
}

// GetRules는 알림 발생 규칙들을 반환합니다
func (as *AlertingSystem) GetRules() []AlertRule {
	as.mutex.RLock()
	defer as.mutex.RUnlock()

	rules := make([]AlertRule, len(as.rules))
	copy(rules, as.rules)
	return rules
}

// GetMetrics는 알림 시스템 메트릭을 반환합니다
func (as *AlertingSystem) GetMetrics() *AlertingMetrics {
	as.mutex.RLock()
	defer as.mutex.RUnlock()

	activeCount := 0
	totalCount := int64(len(as.alerts))
	resolvedCount := int64(0)

	for _, alert := range as.alerts {
		if alert.Status == StatusFiring {
			activeCount++
		} else if alert.Status == StatusResolved {
			resolvedCount++
		}
	}

	return &AlertingMetrics{
		ActiveAlerts:   activeCount,
		TotalAlerts:    totalCount,
		ResolvedAlerts: resolvedCount,
		RuleCount:      len(as.rules),
		ChannelCount:   len(as.channels),
		LastEvaluation: as.lastEvaluation,
	}
}

// Stop은 알림 시스템을 중지합니다
func (as *AlertingSystem) Stop() {
	as.cancel()
	as.wg.Wait()
}

// 내부 메서드들

func (as *AlertingSystem) startEvaluationLoop() {
	as.wg.Add(1)
	go func() {
		defer as.wg.Done()

		ticker := time.NewTicker(as.config.EvaluationInterval)
		defer ticker.Stop()

		for {
			select {
			case <-as.ctx.Done():
				return
			case <-ticker.C:
				as.evaluateRules()
				as.cleanupOldAlerts()
			}
		}
	}()
}

func (as *AlertingSystem) evaluateRules() {
	start := time.Now()
	as.lastEvaluation = start

	as.mutex.RLock()
	rules := make([]AlertRule, len(as.rules))
	copy(rules, as.rules)
	as.mutex.RUnlock()

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		// 규칙 평가 (간단한 예시)
		// 실제 구현에서는 메트릭 수집 시스템과 연동
		if as.shouldFire(rule) {
			as.FireAlert(rule.ID, rule.Description, rule.Labels, nil)
		}

		// 규칙 평가 통계 업데이트
		as.mutex.Lock()
		for i := range as.rules {
			if as.rules[i].ID == rule.ID {
				as.rules[i].LastEvaluation = start
				as.rules[i].EvaluationCount++
				break
			}
		}
		as.mutex.Unlock()
	}
}

func (as *AlertingSystem) shouldFire(rule AlertRule) bool {
	// 실제 구현에서는 메트릭 수집 시스템에서 데이터를 가져와서
	// 규칙의 조건과 비교해야 함
	// 여기서는 더미 구현
	return false
}

func (as *AlertingSystem) sendAlert(alert *Alert, channels []string) {
	as.mutex.RLock()
	availableChannels := make(map[string]AlertChannel)
	for name, channel := range as.channels {
		availableChannels[name] = channel
	}
	as.mutex.RUnlock()

	for _, channelName := range channels {
		channel, exists := availableChannels[channelName]
		if !exists || !channel.IsEnabled() {
			continue
		}

		go func(ch AlertChannel, a *Alert) {
			if err := ch.Send(a); err != nil {
				log.Printf("Failed to send alert via %s: %v", ch.GetType(), err)
			} else {
				// 전송 성공 통계 업데이트
				as.mutex.Lock()
				if alertInMap, exists := as.alerts[a.ID]; exists {
					now := time.Now()
					alertInMap.LastSentAt = &now
					alertInMap.SentCount++
				}
				as.mutex.Unlock()
			}
		}(channel, alert)
	}
}

func (as *AlertingSystem) cleanupOldAlerts() {
	as.mutex.Lock()
	defer as.mutex.Unlock()

	cutoff := time.Now().Add(-as.config.RetentionPeriod)
	alertsToRemove := make([]string, 0)

	for id, alert := range as.alerts {
		if alert.Status == StatusResolved && alert.EndsAt != nil && alert.EndsAt.Before(cutoff) {
			alertsToRemove = append(alertsToRemove, id)
		}
	}

	for _, id := range alertsToRemove {
		delete(as.alerts, id)
	}

	// 최대 알림 수 제한
	if len(as.alerts) > as.config.MaxAlerts {
		// 가장 오래된 알림들 제거
		allAlerts := make([]*Alert, 0, len(as.alerts))
		for _, alert := range as.alerts {
			allAlerts = append(allAlerts, alert)
		}

		// 시작 시간순으로 정렬
		for i := 0; i < len(allAlerts)-1; i++ {
			for j := i + 1; j < len(allAlerts); j++ {
				if allAlerts[i].StartsAt.After(allAlerts[j].StartsAt) {
					allAlerts[i], allAlerts[j] = allAlerts[j], allAlerts[i]
				}
			}
		}

		// 초과한 알림들 제거
		toRemove := len(allAlerts) - as.config.MaxAlerts
		for i := 0; i < toRemove; i++ {
			delete(as.alerts, allAlerts[i].ID)
		}
	}
}

func mergeLabels(base, additional map[string]string) map[string]string {
	result := make(map[string]string)

	// 기본 레이블 복사
	for k, v := range base {
		result[k] = v
	}

	// 추가 레이블 복사 (덮어쓰기)
	for k, v := range additional {
		result[k] = v
	}

	return result
}

// 채널 구현들

// Send implements AlertChannel for EmailChannel
func (ec *EmailChannel) Send(alert *Alert) error {
	if !ec.Enabled {
		return fmt.Errorf("email channel is disabled")
	}

	// 실제 이메일 전송 구현
	// SMTP 라이브러리 사용
	log.Printf("[EMAIL] Alert: %s - %s", alert.Name, alert.Message)
	return nil
}

func (ec *EmailChannel) GetType() string {
	return "email"
}

func (ec *EmailChannel) IsEnabled() bool {
	return ec.Enabled
}

// Send implements AlertChannel for SlackChannel
func (sc *SlackChannel) Send(alert *Alert) error {
	if !sc.Enabled {
		return fmt.Errorf("slack channel is disabled")
	}

	// 실제 Slack 웹훅 전송 구현
	// HTTP 클라이언트로 Slack API 호출
	log.Printf("[SLACK] Alert: %s - %s", alert.Name, alert.Message)
	return nil
}

func (sc *SlackChannel) GetType() string {
	return "slack"
}

func (sc *SlackChannel) IsEnabled() bool {
	return sc.Enabled
}

// Send implements AlertChannel for WebhookChannel
func (wc *WebhookChannel) Send(alert *Alert) error {
	if !wc.Enabled {
		return fmt.Errorf("webhook channel is disabled")
	}

	// 실제 웹훅 전송 구현
	// HTTP 요청으로 알림 데이터 전송
	log.Printf("[WEBHOOK] Alert: %s - %s", alert.Name, alert.Message)
	return nil
}

func (wc *WebhookChannel) GetType() string {
	return "webhook"
}

func (wc *WebhookChannel) IsEnabled() bool {
	return wc.Enabled
}

// Send implements AlertChannel for LogChannel
func (lc *LogChannel) Send(alert *Alert) error {
	if !lc.Enabled {
		return fmt.Errorf("log channel is disabled")
	}

	// 로그로 알림 출력
	severityPrefix := ""
	switch alert.Severity {
	case SeverityCritical:
		severityPrefix = "[CRITICAL]"
	case SeverityError:
		severityPrefix = "[ERROR]"
	case SeverityWarning:
		severityPrefix = "[WARNING]"
	case SeverityInfo:
		severityPrefix = "[INFO]"
	}

	log.Printf("%s %s: %s (Status: %s, Started: %s)",
		severityPrefix, alert.Name, alert.Message, alert.Status, alert.StartsAt.Format("2006-01-02 15:04:05"))

	return nil
}

func (lc *LogChannel) GetType() string {
	return "log"
}

func (lc *LogChannel) IsEnabled() bool {
	return lc.Enabled
}
