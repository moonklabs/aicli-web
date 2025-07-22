package security

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"

	"github.com/aicli/aicli-web/internal/models"
)

// AttackPattern은 공격 패턴을 정의합니다.
type AttackPattern struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Pattern     string    `json:"pattern"`
	Regex       *regexp.Regexp `json:"-"`
	Severity    Severity  `json:"severity"`
	Confidence  float64   `json:"confidence"` // 0.0 ~ 1.0
	Description string    `json:"description"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// AttackDetectionResult는 공격 탐지 결과를 나타냅니다.
type AttackDetectionResult struct {
	IsAttack      bool               `json:"is_attack"`
	AttackType    string             `json:"attack_type,omitempty"`
	Patterns      []*AttackPattern   `json:"patterns,omitempty"`
	Confidence    float64            `json:"confidence"`
	Risk          Severity           `json:"risk"`
	Evidence      []string           `json:"evidence,omitempty"`
	Recommendations []string         `json:"recommendations,omitempty"`
}

// AttackDetectorConfig는 공격 탐지기 설정입니다.
type AttackDetectorConfig struct {
	Redis               redis.UniversalClient
	EventTracker        *EventTracker
	
	// 탐지 설정
	MinConfidence       float64       // 최소 신뢰도
	BruteForceThreshold int           // 무차별 공격 임계값
	BruteForceWindow    time.Duration // 무차별 공격 시간 윈도우
	AnomalyThreshold    float64       // 이상 행동 임계값
	
	// 자동 차단 설정
	AutoBlockEnabled    bool          // 자동 차단 활성화
	BlockDuration       time.Duration // 차단 지속 시간
	
	// 알림 설정
	AlertEnabled        bool          // 알림 활성화
	AlertThreshold      Severity      // 알림 최소 심각도
	
	Logger *zap.Logger
}

// AttackDetector는 공격 패턴 탐지기입니다.
type AttackDetector struct {
	config       *AttackDetectorConfig
	redis        redis.UniversalClient
	logger       *zap.Logger
	patterns     []*AttackPattern
	eventTracker *EventTracker
}

// DefaultAttackDetectorConfig는 기본 공격 탐지기 설정을 반환합니다.
func DefaultAttackDetectorConfig() *AttackDetectorConfig {
	return &AttackDetectorConfig{
		MinConfidence:       0.7,
		BruteForceThreshold: 10,
		BruteForceWindow:    5 * time.Minute,
		AnomalyThreshold:    0.8,
		AutoBlockEnabled:    true,
		BlockDuration:       30 * time.Minute,
		AlertEnabled:        true,
		AlertThreshold:      SeverityHigh,
	}
}

// NewAttackDetector는 새로운 공격 탐지기를 생성합니다.
func NewAttackDetector(config *AttackDetectorConfig) *AttackDetector {
	if config == nil {
		config = DefaultAttackDetectorConfig()
	}

	detector := &AttackDetector{
		config:       config,
		redis:        config.Redis,
		logger:       config.Logger,
		eventTracker: config.EventTracker,
		patterns:     make([]*AttackPattern, 0),
	}

	// 기본 공격 패턴 로드
	detector.loadDefaultPatterns()

	return detector
}

// loadDefaultPatterns는 기본 공격 패턴을 로드합니다.
func (ad *AttackDetector) loadDefaultPatterns() {
	defaultPatterns := []*AttackPattern{
		// SQL Injection 패턴들
		{
			ID:          "sqli_001",
			Name:        "SQL Injection - UNION",
			Type:        "sql_injection",
			Pattern:     `(?i)(union\s+(all\s+)?select)`,
			Severity:    SeverityHigh,
			Confidence:  0.9,
			Description: "UNION SELECT를 이용한 SQL Injection 공격",
			IsActive:    true,
		},
		{
			ID:          "sqli_002",
			Name:        "SQL Injection - OR 1=1",
			Type:        "sql_injection",
			Pattern:     `(?i)(\'\s*or\s*\d+\s*=\s*\d+|\'\s*or\s*\'\d+\'\s*=\s*\'\d+)`,
			Severity:    SeverityHigh,
			Confidence:  0.8,
			Description: "OR 1=1 조건을 이용한 SQL Injection",
			IsActive:    true,
		},
		{
			ID:          "sqli_003",
			Name:        "SQL Injection - Comment",
			Type:        "sql_injection",
			Pattern:     `(?i)(\/\*.*?\*\/|--[\s\S]*$|\#[\s\S]*$)`,
			Severity:    SeverityMedium,
			Confidence:  0.6,
			Description: "주석을 이용한 SQL Injection",
			IsActive:    true,
		},
		
		// XSS 패턴들
		{
			ID:          "xss_001",
			Name:        "XSS - Script Tag",
			Type:        "xss",
			Pattern:     `(?i)<script[^>]*>.*?</script>`,
			Severity:    SeverityHigh,
			Confidence:  0.9,
			Description: "Script 태그를 이용한 XSS 공격",
			IsActive:    true,
		},
		{
			ID:          "xss_002",
			Name:        "XSS - Event Handler",
			Type:        "xss",
			Pattern:     `(?i)on\w+\s*=\s*['\"].*?['\"]`,
			Severity:    SeverityHigh,
			Confidence:  0.8,
			Description: "이벤트 핸들러를 이용한 XSS 공격",
			IsActive:    true,
		},
		{
			ID:          "xss_003",
			Name:        "XSS - JavaScript URI",
			Type:        "xss",
			Pattern:     `(?i)javascript\s*:`,
			Severity:    SeverityMedium,
			Confidence:  0.7,
			Description: "JavaScript URI를 이용한 XSS 공격",
			IsActive:    true,
		},
		
		// Command Injection 패턴들
		{
			ID:          "cmdi_001",
			Name:        "Command Injection - Pipe",
			Type:        "command_injection",
			Pattern:     `(?i)[\|;&\$\(\)\{\}><]`,
			Severity:    SeverityHigh,
			Confidence:  0.6,
			Description: "시스템 명령어 실행을 위한 특수문자",
			IsActive:    true,
		},
		{
			ID:          "cmdi_002",
			Name:        "Command Injection - Common Commands",
			Type:        "command_injection",
			Pattern:     `(?i)(cat|ls|pwd|id|whoami|uname|ps|netstat|ifconfig|ping|wget|curl|nc|telnet)\s`,
			Severity:    SeverityHigh,
			Confidence:  0.8,
			Description: "일반적인 시스템 명령어",
			IsActive:    true,
		},
		
		// Directory Traversal 패턴들
		{
			ID:          "dt_001",
			Name:        "Directory Traversal - Dot Dot Slash",
			Type:        "directory_traversal",
			Pattern:     `(?i)(\.\./|\.\.\\|%2e%2e%2f|%252e%252e%252f)`,
			Severity:    SeverityMedium,
			Confidence:  0.8,
			Description: "디렉토리 순회 공격",
			IsActive:    true,
		},
		
		// File Upload 공격 패턴들
		{
			ID:          "fu_001",
			Name:        "Malicious File Upload - Executable",
			Type:        "file_upload",
			Pattern:     `(?i)\.(php|jsp|asp|aspx|exe|bat|sh|py|pl|rb)$`,
			Severity:    SeverityHigh,
			Confidence:  0.7,
			Description: "악성 실행파일 업로드",
			IsActive:    true,
		},
		
		// LDAP Injection 패턴들
		{
			ID:          "ldapi_001",
			Name:        "LDAP Injection",
			Type:        "ldap_injection",
			Pattern:     `(?i)[\(\)\*\&\|!]`,
			Severity:    SeverityMedium,
			Confidence:  0.6,
			Description: "LDAP Injection 공격",
			IsActive:    true,
		},
		
		// XXE (XML External Entity) 패턴들
		{
			ID:          "xxe_001",
			Name:        "XXE Attack",
			Type:        "xxe",
			Pattern:     `(?i)<!entity.*?>|<!doctype.*?\[.*?<!entity`,
			Severity:    SeverityHigh,
			Confidence:  0.8,
			Description: "XXE (XML External Entity) 공격",
			IsActive:    true,
		},
		
		// Server-Side Template Injection 패턴들
		{
			ID:          "ssti_001",
			Name:        "SSTI - Template Expressions",
			Type:        "ssti",
			Pattern:     `(?i)(\{\{.*?\}\}|\$\{.*?\}|<%.*?%>)`,
			Severity:    SeverityHigh,
			Confidence:  0.7,
			Description: "서버사이드 템플릿 인젝션",
			IsActive:    true,
		},
	}

	// 정규표현식 컴파일
	for _, pattern := range defaultPatterns {
		compiled, err := regexp.Compile(pattern.Pattern)
		if err != nil {
			ad.logger.Error("공격 패턴 정규표현식 컴파일 실패",
				zap.String("pattern_id", pattern.ID),
				zap.Error(err))
			continue
		}
		pattern.Regex = compiled
		pattern.CreatedAt = time.Now()
		pattern.UpdatedAt = time.Now()
	}

	ad.patterns = defaultPatterns
}

// DetectAttacks는 요청에서 공격 패턴을 탐지합니다.
func (ad *AttackDetector) DetectAttacks(ctx context.Context, request *AttackDetectionRequest) *AttackDetectionResult {
	result := &AttackDetectionResult{
		IsAttack:        false,
		Patterns:        make([]*AttackPattern, 0),
		Confidence:      0.0,
		Risk:           SeverityLow,
		Evidence:        make([]string, 0),
		Recommendations: make([]string, 0),
	}

	// 패턴 기반 탐지
	ad.detectPatternBasedAttacks(request, result)

	// 행동 기반 탐지
	ad.detectBehaviorBasedAttacks(ctx, request, result)

	// 빈도 기반 탐지
	ad.detectFrequencyBasedAttacks(ctx, request, result)

	// 이상 행동 탐지
	ad.detectAnomalyBasedAttacks(ctx, request, result)

	// 최종 결과 계산
	ad.calculateFinalResult(result)

	// 공격 탐지 시 이벤트 기록 및 조치
	if result.IsAttack {
		ad.handleDetectedAttack(ctx, request, result)
	}

	return result
}

// AttackDetectionRequest는 공격 탐지 요청을 나타냅니다.
type AttackDetectionRequest struct {
	UserID      string            `json:"user_id,omitempty"`
	SessionID   string            `json:"session_id,omitempty"`
	IPAddress   string            `json:"ip_address"`
	UserAgent   string            `json:"user_agent"`
	Method      string            `json:"method"`
	URL         string            `json:"url"`
	Path        string            `json:"path"`
	Query       string            `json:"query,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Body        string            `json:"body,omitempty"`
	Timestamp   time.Time         `json:"timestamp"`
}

// detectPatternBasedAttacks는 패턴 기반 공격을 탐지합니다.
func (ad *AttackDetector) detectPatternBasedAttacks(request *AttackDetectionRequest, result *AttackDetectionResult) {
	// 검사할 대상 문자열 구성
	targets := []string{
		request.URL,
		request.Query,
		request.Body,
	}
	
	// 헤더도 검사 대상에 추가
	for _, value := range request.Headers {
		targets = append(targets, value)
	}

	// 각 패턴에 대해 검사
	for _, pattern := range ad.patterns {
		if !pattern.IsActive {
			continue
		}

		for _, target := range targets {
			if target == "" {
				continue
			}

			if pattern.Regex.MatchString(target) {
				result.Patterns = append(result.Patterns, pattern)
				result.Evidence = append(result.Evidence, 
					fmt.Sprintf("Pattern %s matched: %s", pattern.Name, ad.truncateString(target, 100)))
				
				// 신뢰도 누적
				result.Confidence = ad.combineConfidence(result.Confidence, pattern.Confidence)
				
				ad.logger.Debug("공격 패턴 탐지됨",
					zap.String("pattern_id", pattern.ID),
					zap.String("pattern_name", pattern.Name),
					zap.String("ip", request.IPAddress))
			}
		}
	}
}

// detectBehaviorBasedAttacks는 행동 기반 공격을 탐지합니다.
func (ad *AttackDetector) detectBehaviorBasedAttacks(ctx context.Context, request *AttackDetectionRequest, result *AttackDetectionResult) {
	// 1. 비정상적인 요청 빈도 확인
	if ad.isHighFrequencyRequest(ctx, request.IPAddress) {
		result.Evidence = append(result.Evidence, "High frequency requests detected")
		result.Confidence = ad.combineConfidence(result.Confidence, 0.6)
	}

	// 2. 비정상적인 User-Agent 확인
	if ad.isSuspiciousUserAgent(request.UserAgent) {
		result.Evidence = append(result.Evidence, fmt.Sprintf("Suspicious User-Agent: %s", request.UserAgent))
		result.Confidence = ad.combineConfidence(result.Confidence, 0.5)
	}

	// 3. 비정상적인 요청 크기 확인
	if ad.isAbnormalRequestSize(request) {
		result.Evidence = append(result.Evidence, "Abnormal request size detected")
		result.Confidence = ad.combineConfidence(result.Confidence, 0.4)
	}

	// 4. 비정상적인 경로 패턴 확인
	if ad.isSuspiciousPath(request.Path) {
		result.Evidence = append(result.Evidence, fmt.Sprintf("Suspicious path pattern: %s", request.Path))
		result.Confidence = ad.combineConfidence(result.Confidence, 0.6)
	}
}

// detectFrequencyBasedAttacks는 빈도 기반 공격을 탐지합니다.
func (ad *AttackDetector) detectFrequencyBasedAttacks(ctx context.Context, request *AttackDetectionRequest, result *AttackDetectionResult) {
	now := time.Now()
	windowStart := now.Add(-ad.config.BruteForceWindow)

	// 브루트포스 공격 탐지
	if ad.isBruteForceAttack(ctx, request, windowStart, now) {
		result.Evidence = append(result.Evidence, "Brute force attack pattern detected")
		result.Confidence = ad.combineConfidence(result.Confidence, 0.9)
		result.AttackType = "brute_force"
	}

	// 스캔 공격 탐지
	if ad.isScanningAttack(ctx, request, windowStart, now) {
		result.Evidence = append(result.Evidence, "Port/Path scanning detected")
		result.Confidence = ad.combineConfidence(result.Confidence, 0.8)
		result.AttackType = "scanning"
	}
}

// detectAnomalyBasedAttacks는 이상 행동 기반 공격을 탐지합니다.
func (ad *AttackDetector) detectAnomalyBasedAttacks(ctx context.Context, request *AttackDetectionRequest, result *AttackDetectionResult) {
	// 시간 기반 이상 탐지
	if ad.isAbnormalTimeAccess(request.Timestamp) {
		result.Evidence = append(result.Evidence, "Access during abnormal hours")
		result.Confidence = ad.combineConfidence(result.Confidence, 0.3)
	}

	// 지리적 이상 탐지
	if ad.isGeographicAnomaly(ctx, request.IPAddress, request.UserID) {
		result.Evidence = append(result.Evidence, "Geographic anomaly detected")
		result.Confidence = ad.combineConfidence(result.Confidence, 0.6)
	}

	// 세션 이상 탐지
	if ad.isSessionAnomaly(ctx, request) {
		result.Evidence = append(result.Evidence, "Session anomaly detected")
		result.Confidence = ad.combineConfidence(result.Confidence, 0.5)
	}
}

// calculateFinalResult는 최종 결과를 계산합니다.
func (ad *AttackDetector) calculateFinalResult(result *AttackDetectionResult) {
	// 최종 공격 여부 판단
	result.IsAttack = result.Confidence >= ad.config.MinConfidence

	// 위험도 계산
	if result.Confidence >= 0.9 {
		result.Risk = SeverityCritical
	} else if result.Confidence >= 0.7 {
		result.Risk = SeverityHigh
	} else if result.Confidence >= 0.5 {
		result.Risk = SeverityMedium
	} else {
		result.Risk = SeverityLow
	}

	// 권장사항 생성
	result.Recommendations = ad.generateRecommendations(result)

	// 공격 타입 결정 (아직 설정되지 않은 경우)
	if result.AttackType == "" && len(result.Patterns) > 0 {
		// 가장 높은 신뢰도의 패턴 타입 사용
		var maxConfidence float64
		for _, pattern := range result.Patterns {
			if pattern.Confidence > maxConfidence {
				maxConfidence = pattern.Confidence
				result.AttackType = pattern.Type
			}
		}
	}
}

// handleDetectedAttack은 탐지된 공격을 처리합니다.
func (ad *AttackDetector) handleDetectedAttack(ctx context.Context, request *AttackDetectionRequest, result *AttackDetectionResult) {
	// 보안 이벤트 기록
	if ad.eventTracker != nil {
		event := &SecurityEvent{
			Type:        EventTypeAttackPattern,
			Severity:    result.Risk,
			Source:      request.IPAddress,
			Target:      request.Path,
			UserID:      request.UserID,
			SessionID:   request.SessionID,
			IPAddress:   request.IPAddress,
			UserAgent:   request.UserAgent,
			RequestPath: request.Path,
			Method:      request.Method,
			Details: map[string]interface{}{
				"attack_type":   result.AttackType,
				"confidence":    result.Confidence,
				"patterns":      len(result.Patterns),
				"evidence":      result.Evidence,
			},
		}

		if err := ad.eventTracker.RecordEvent(ctx, event); err != nil {
			ad.logger.Error("공격 이벤트 기록 실패", zap.Error(err))
		}
	}

	// 자동 차단
	if ad.config.AutoBlockEnabled && result.Risk >= SeverityHigh {
		ad.blockIP(ctx, request.IPAddress, ad.config.BlockDuration)
	}

	// 알림 발송
	if ad.config.AlertEnabled && result.Risk >= ad.config.AlertThreshold {
		ad.sendAlert(ctx, request, result)
	}

	ad.logger.Warn("공격 패턴 탐지됨",
		zap.String("ip", request.IPAddress),
		zap.String("user_id", request.UserID),
		zap.String("attack_type", result.AttackType),
		zap.Float64("confidence", result.Confidence),
		zap.String("risk", string(result.Risk)))
}

// 헬퍼 메서드들

func (ad *AttackDetector) truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func (ad *AttackDetector) combineConfidence(current, new float64) float64 {
	// 신뢰도 결합: 독립적인 증거들을 결합
	return current + new*(1-current)
}

func (ad *AttackDetector) isHighFrequencyRequest(ctx context.Context, ipAddress string) bool {
	if ad.redis == nil {
		return false
	}

	key := fmt.Sprintf("attack:freq:%s", ipAddress)
	count, err := ad.redis.Incr(ctx, key).Result()
	if err != nil {
		return false
	}

	if count == 1 {
		ad.redis.Expire(ctx, key, time.Minute)
	}

	return count > 60 // 분당 60회 이상
}

func (ad *AttackDetector) isSuspiciousUserAgent(userAgent string) bool {
	suspiciousPatterns := []string{
		"sqlmap", "nikto", "nmap", "masscan", "dirb", "gobuster",
		"burp", "owasp zap", "w3af", "acunetix", "netsparker",
		"python-requests", "curl/7", "wget", "bot", "crawler",
		"scanner", "fuzzer", "exploit",
	}

	userAgentLower := strings.ToLower(userAgent)
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(userAgentLower, pattern) {
			return true
		}
	}

	return false
}

func (ad *AttackDetector) isAbnormalRequestSize(request *AttackDetectionRequest) bool {
	// URL 길이 확인
	if len(request.URL) > 2048 {
		return true
	}

	// 바디 크기 확인
	if len(request.Body) > 1024*1024 { // 1MB
		return true
	}

	// 헤더 수 확인
	if len(request.Headers) > 50 {
		return true
	}

	return false
}

func (ad *AttackDetector) isSuspiciousPath(path string) bool {
	suspiciousPaths := []string{
		"/admin", "/config", "/backup", "/test", "/debug",
		"/phpinfo", "/phpmyadmin", "/wp-admin", "/manager",
		"/.env", "/.git", "/robots.txt", "/sitemap.xml",
	}

	pathLower := strings.ToLower(path)
	for _, suspicious := range suspiciousPaths {
		if strings.Contains(pathLower, suspicious) {
			return true
		}
	}

	return false
}

func (ad *AttackDetector) isBruteForceAttack(ctx context.Context, request *AttackDetectionRequest, start, end time.Time) bool {
	if ad.redis == nil {
		return false
	}

	// 로그인 실패 패턴 확인
	if !strings.Contains(request.Path, "login") && !strings.Contains(request.Path, "auth") {
		return false
	}

	key := fmt.Sprintf("attack:bruteforce:%s", request.IPAddress)
	count, err := ad.redis.Incr(ctx, key).Result()
	if err != nil {
		return false
	}

	if count == 1 {
		ad.redis.Expire(ctx, key, ad.config.BruteForceWindow)
	}

	return count > int64(ad.config.BruteForceThreshold)
}

func (ad *AttackDetector) isScanningAttack(ctx context.Context, request *AttackDetectionRequest, start, end time.Time) bool {
	if ad.redis == nil {
		return false
	}

	// 다양한 경로 접근 패턴 확인
	key := fmt.Sprintf("attack:scan:%s", request.IPAddress)
	pathKey := fmt.Sprintf("%s:paths", key)

	// 경로를 집합에 추가
	ad.redis.SAdd(ctx, pathKey, request.Path)
	ad.redis.Expire(ctx, pathKey, 10*time.Minute)

	// 고유 경로 수 확인
	pathCount, err := ad.redis.SCard(ctx, pathKey).Result()
	if err != nil {
		return false
	}

	return pathCount > 20 // 10분 내 20개 이상의 서로 다른 경로 접근
}

func (ad *AttackDetector) isAbnormalTimeAccess(timestamp time.Time) bool {
	hour := timestamp.Hour()
	// 새벽 2시~6시 접근을 비정상으로 판단
	return hour >= 2 && hour <= 6
}

func (ad *AttackDetector) isGeographicAnomaly(ctx context.Context, ipAddress, userID string) bool {
	// IP 지리적 위치 확인 로직 (GeoIP 데이터베이스 필요)
	// 여기서는 간단히 private IP 체크
	ip := net.ParseIP(ipAddress)
	if ip == nil {
		return false
	}

	// Private IP 범위는 정상으로 간주
	return !ip.IsPrivate()
}

func (ad *AttackDetector) isSessionAnomaly(ctx context.Context, request *AttackDetectionRequest) bool {
	if request.SessionID == "" {
		return false
	}

	// 세션 기반 이상 행동 탐지 로직
	// 예: 짧은 시간 내 다수의 IP에서 같은 세션 사용
	return false
}

func (ad *AttackDetector) generateRecommendations(result *AttackDetectionResult) []string {
	recommendations := make([]string, 0)

	if result.Risk >= SeverityHigh {
		recommendations = append(recommendations, "즉시 IP 차단 고려")
		recommendations = append(recommendations, "보안 팀에 즉시 알림")
	}

	if result.Risk >= SeverityMedium {
		recommendations = append(recommendations, "추가 모니터링 적용")
		recommendations = append(recommendations, "세션 무효화 고려")
	}

	if len(result.Patterns) > 0 {
		recommendations = append(recommendations, "WAF 규칙 업데이트 검토")
		recommendations = append(recommendations, "애플리케이션 보안 패치 확인")
	}

	return recommendations
}

func (ad *AttackDetector) blockIP(ctx context.Context, ipAddress string, duration time.Duration) {
	if ad.redis == nil {
		return
	}

	blockKey := fmt.Sprintf("blocked:ip:%s", ipAddress)
	err := ad.redis.Set(ctx, blockKey, time.Now().Unix(), duration).Err()
	if err != nil {
		ad.logger.Error("IP 차단 실패", zap.String("ip", ipAddress), zap.Error(err))
		return
	}

	ad.logger.Warn("IP 자동 차단됨", 
		zap.String("ip", ipAddress),
		zap.Duration("duration", duration))
}

func (ad *AttackDetector) sendAlert(ctx context.Context, request *AttackDetectionRequest, result *AttackDetectionResult) {
	// 여기에 실제 알림 발송 로직 구현 (이메일, 슬랙, 웹훅 등)
	ad.logger.Error("보안 공격 알림",
		zap.String("attack_type", result.AttackType),
		zap.String("ip", request.IPAddress),
		zap.Float64("confidence", result.Confidence),
		zap.Strings("evidence", result.Evidence))
}

// GetPatterns는 현재 로드된 공격 패턴 목록을 반환합니다.
func (ad *AttackDetector) GetPatterns() []*AttackPattern {
	return ad.patterns
}

// AddPattern은 새로운 공격 패턴을 추가합니다.
func (ad *AttackDetector) AddPattern(pattern *AttackPattern) error {
	// 정규표현식 컴파일
	compiled, err := regexp.Compile(pattern.Pattern)
	if err != nil {
		return fmt.Errorf("정규표현식 컴파일 실패: %w", err)
	}

	pattern.Regex = compiled
	pattern.CreatedAt = time.Now()
	pattern.UpdatedAt = time.Now()

	ad.patterns = append(ad.patterns, pattern)
	return nil
}

// UpdatePattern은 기존 공격 패턴을 업데이트합니다.
func (ad *AttackDetector) UpdatePattern(patternID string, updates *AttackPattern) error {
	for i, pattern := range ad.patterns {
		if pattern.ID == patternID {
			// 정규표현식 재컴파일 (패턴이 변경된 경우)
			if updates.Pattern != "" && updates.Pattern != pattern.Pattern {
				compiled, err := regexp.Compile(updates.Pattern)
				if err != nil {
					return fmt.Errorf("정규표현식 컴파일 실패: %w", err)
				}
				updates.Regex = compiled
			}

			// 필드 업데이트
			if updates.Name != "" {
				ad.patterns[i].Name = updates.Name
			}
			if updates.Pattern != "" {
				ad.patterns[i].Pattern = updates.Pattern
				ad.patterns[i].Regex = updates.Regex
			}
			if updates.Severity != "" {
				ad.patterns[i].Severity = updates.Severity
			}
			if updates.Confidence > 0 {
				ad.patterns[i].Confidence = updates.Confidence
			}
			if updates.Description != "" {
				ad.patterns[i].Description = updates.Description
			}

			ad.patterns[i].UpdatedAt = time.Now()
			return nil
		}
	}

	return fmt.Errorf("패턴을 찾을 수 없음: %s", patternID)
}

// RemovePattern은 공격 패턴을 제거합니다.
func (ad *AttackDetector) RemovePattern(patternID string) error {
	for i, pattern := range ad.patterns {
		if pattern.ID == patternID {
			ad.patterns = append(ad.patterns[:i], ad.patterns[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("패턴을 찾을 수 없음: %s", patternID)
}

// GetStatistics는 공격 탐지 통계를 반환합니다.
func (ad *AttackDetector) GetStatistics(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	if ad.redis == nil {
		return stats, nil
	}

	// 차단된 IP 수
	pattern := "blocked:ip:*"
	blockedIPs, err := ad.countKeysByPattern(ctx, pattern)
	if err == nil {
		stats["blocked_ips"] = blockedIPs
	}

	// 최근 24시간 공격 탐지 수
	// (이 부분은 실제 이벤트 추적기와 연동하여 구현)

	stats["active_patterns"] = len(ad.patterns)
	stats["detector_config"] = map[string]interface{}{
		"min_confidence":        ad.config.MinConfidence,
		"brute_force_threshold": ad.config.BruteForceThreshold,
		"auto_block_enabled":    ad.config.AutoBlockEnabled,
		"alert_enabled":         ad.config.AlertEnabled,
	}

	return stats, nil
}

func (ad *AttackDetector) countKeysByPattern(ctx context.Context, pattern string) (int, error) {
	var cursor uint64
	var count int

	for {
		keys, nextCursor, err := ad.redis.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return 0, err
		}

		count += len(keys)
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return count, nil
}