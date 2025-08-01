package security

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

// AgentRequestPattern은 AI 에이전트 요청 패턴을 정의합니다.
type AgentRequestPattern struct {
	UserID       string            `json:"user_id"`
	SessionID    string            `json:"session_id"`
	RequestType  string            `json:"request_type"`
	Timestamp    time.Time         `json:"timestamp"`
	TokenCount   int               `json:"token_count"`
	PromptLength int               `json:"prompt_length"`
	ResponseTime time.Duration     `json:"response_time"`
	Metadata     map[string]string `json:"metadata"`
}

// AgentAnomalyResult는 AI 에이전트 이상 탐지 결과를 나타냅니다.
type AgentAnomalyResult struct {
	IsAnomalous     bool                 `json:"is_anomalous"`
	AnomalyType     string               `json:"anomaly_type"`
	Score           float64              `json:"score"` // 0.0 ~ 1.0
	Severity        Severity             `json:"severity"`
	Evidence        []string             `json:"evidence"`
	Pattern         *AgentRequestPattern `json:"pattern"`
	Recommendations []string             `json:"recommendations"`
}

// AgentAnalyzerConfig는 에이전트 분석기 설정입니다.
type AgentAnalyzerConfig struct {
	Redis        redis.UniversalClient
	EventTracker *EventTracker

	// 임계값 설정
	MaxRequestsPerMinute   int           // 분당 최대 요청 수
	MaxTokensPerHour       int           // 시간당 최대 토큰 수
	MaxPromptLength        int           // 최대 프롬프트 길이
	SuspiciousResponseTime time.Duration // 의심스러운 응답 시간
	AnomalyThreshold       float64       // 이상 탐지 임계값

	// 시간 윈도우 설정
	AnalysisWindow time.Duration // 분석 시간 윈도우
	PatternWindow  time.Duration // 패턴 분석 윈도우
	BaselineWindow time.Duration // 베이스라인 계산 윈도우

	Logger *zap.Logger
}

// AgentAnalyzer는 AI 에이전트 요청 패턴 분석기입니다.
type AgentAnalyzer struct {
	config             *AgentAnalyzerConfig
	redis              redis.UniversalClient
	logger             *zap.Logger
	eventTracker       *EventTracker
	suspiciousPatterns []*regexp.Regexp
}

// DefaultAgentAnalyzerConfig는 기본 에이전트 분석기 설정을 반환합니다.
func DefaultAgentAnalyzerConfig() *AgentAnalyzerConfig {
	return &AgentAnalyzerConfig{
		MaxRequestsPerMinute:   100,
		MaxTokensPerHour:       10000,
		MaxPromptLength:        8192,
		SuspiciousResponseTime: 30 * time.Second,
		AnomalyThreshold:       0.7,
		AnalysisWindow:         15 * time.Minute,
		PatternWindow:          1 * time.Hour,
		BaselineWindow:         24 * time.Hour,
	}
}

// NewAgentAnalyzer는 새로운 에이전트 분석기를 생성합니다.
func NewAgentAnalyzer(config *AgentAnalyzerConfig) *AgentAnalyzer {
	if config == nil {
		config = DefaultAgentAnalyzerConfig()
	}

	analyzer := &AgentAnalyzer{
		config:       config,
		redis:        config.Redis,
		logger:       config.Logger,
		eventTracker: config.EventTracker,
	}

	// 의심스러운 패턴 정규표현식 컴파일
	analyzer.compileSuspiciousPatterns()

	return analyzer
}

// compileSuspiciousPatterns는 의심스러운 패턴들을 컴파일합니다.
func (aa *AgentAnalyzer) compileSuspiciousPatterns() {
	patterns := []string{
		// 반복적인 시스템 프롬프트 조작 시도
		`(?i)(system|assistant|user):\s*[^\n]{200,}`,
		// Base64 인코딩된 악성 페이로드
		`[A-Za-z0-9+/]{100,}={0,2}`,
		// 의심스러운 파일 경로 접근
		`(?i)(\\.\\.[\\/]|%2e%2e%2f|%252e%252e%252f)`,
		// 코드 실행 관련 키워드 패턴
		`(?i)(eval|exec|system|shell|cmd|powershell|bash)\\s*\\(`,
		// 민감한 환경 변수 요청
		`(?i)(\\$\\{|%)(PATH|HOME|USER|PASSWORD|SECRET|KEY|TOKEN)`,
	}

	aa.suspiciousPatterns = make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		if compiled, err := regexp.Compile(pattern); err == nil {
			aa.suspiciousPatterns = append(aa.suspiciousPatterns, compiled)
		} else {
			aa.logger.Error("의심스러운 패턴 컴파일 실패",
				zap.String("pattern", pattern),
				zap.Error(err))
		}
	}
}

// AnalyzeRequest는 AI 에이전트 요청을 분석합니다.
func (aa *AgentAnalyzer) AnalyzeRequest(ctx context.Context, pattern *AgentRequestPattern) *AgentAnomalyResult {
	result := &AgentAnomalyResult{
		IsAnomalous:     false,
		Score:           0.0,
		Severity:        SeverityLow,
		Evidence:        make([]string, 0),
		Pattern:         pattern,
		Recommendations: make([]string, 0),
	}

	// 1. 요청 빈도 분석
	aa.analyzeRequestFrequency(ctx, pattern, result)

	// 2. 토큰 사용량 분석
	aa.analyzeTokenUsage(ctx, pattern, result)

	// 3. 프롬프트 패턴 분석
	aa.analyzePromptPattern(ctx, pattern, result)

	// 4. 응답 시간 분석
	aa.analyzeResponseTime(ctx, pattern, result)

	// 5. 행동 패턴 분석
	aa.analyzeBehaviorPattern(ctx, pattern, result)

	// 6. 시간적 이상 탐지
	aa.analyzeTemporalAnomaly(ctx, pattern, result)

	// 최종 결과 계산
	aa.calculateFinalResult(result)

	// 이상 탐지 시 이벤트 기록
	if result.IsAnomalous {
		aa.handleAnomalousRequest(ctx, pattern, result)
	}

	// 패턴 데이터 저장 (분석용)
	aa.storePatternData(ctx, pattern)

	return result
}

// analyzeRequestFrequency는 요청 빈도를 분석합니다.
func (aa *AgentAnalyzer) analyzeRequestFrequency(ctx context.Context, pattern *AgentRequestPattern, result *AgentAnomalyResult) {
	userKey := fmt.Sprintf("agent:freq:user:%s", pattern.UserID)
	sessionKey := fmt.Sprintf("agent:freq:session:%s", pattern.SessionID)

	// 사용자별 요청 빈도 확인
	userCount, err := aa.redis.Incr(ctx, userKey).Result()
	if err == nil {
		if userCount == 1 {
			aa.redis.Expire(ctx, userKey, time.Minute)
		}
		if userCount > int64(aa.config.MaxRequestsPerMinute) {
			result.Score += 0.8
			result.Evidence = append(result.Evidence,
				fmt.Sprintf("사용자 요청 빈도 초과: %d/분", userCount))
			result.AnomalyType = "high_frequency"
		}
	}

	// 세션별 요청 빈도 확인
	sessionCount, err := aa.redis.Incr(ctx, sessionKey).Result()
	if err == nil {
		if sessionCount == 1 {
			aa.redis.Expire(ctx, sessionKey, time.Minute)
		}
		if sessionCount > int64(aa.config.MaxRequestsPerMinute/2) {
			result.Score += 0.6
			result.Evidence = append(result.Evidence,
				fmt.Sprintf("세션 요청 빈도 초과: %d/분", sessionCount))
		}
	}
}

// analyzeTokenUsage는 토큰 사용량을 분석합니다.
func (aa *AgentAnalyzer) analyzeTokenUsage(ctx context.Context, pattern *AgentRequestPattern, result *AgentAnomalyResult) {
	tokenKey := fmt.Sprintf("agent:tokens:user:%s", pattern.UserID)

	// 시간당 토큰 사용량 누적
	totalTokens, err := aa.redis.IncrBy(ctx, tokenKey, int64(pattern.TokenCount)).Result()
	if err == nil {
		if totalTokens == int64(pattern.TokenCount) {
			aa.redis.Expire(ctx, tokenKey, time.Hour)
		}

		if totalTokens > int64(aa.config.MaxTokensPerHour) {
			result.Score += 0.7
			result.Evidence = append(result.Evidence,
				fmt.Sprintf("시간당 토큰 사용량 초과: %d", totalTokens))
			result.AnomalyType = "excessive_token_usage"
		}
	}

	// 단일 요청의 과도한 토큰 사용
	if pattern.TokenCount > aa.config.MaxTokensPerHour/10 {
		result.Score += 0.5
		result.Evidence = append(result.Evidence,
			fmt.Sprintf("단일 요청 과도한 토큰 사용: %d", pattern.TokenCount))
	}
}

// analyzePromptPattern은 프롬프트 패턴을 분석합니다.
func (aa *AgentAnalyzer) analyzePromptPattern(ctx context.Context, pattern *AgentRequestPattern, result *AgentAnomalyResult) {
	// 프롬프트 길이 확인
	if pattern.PromptLength > aa.config.MaxPromptLength {
		result.Score += 0.6
		result.Evidence = append(result.Evidence,
			fmt.Sprintf("과도한 프롬프트 길이: %d", pattern.PromptLength))
		result.AnomalyType = "oversized_prompt"
	}

	// 메타데이터에서 프롬프트 내용 확인
	if promptContent, exists := pattern.Metadata["prompt"]; exists {
		// 의심스러운 패턴 검사
		for _, regex := range aa.suspiciousPatterns {
			if regex.MatchString(promptContent) {
				result.Score += 0.8
				result.Evidence = append(result.Evidence, "의심스러운 프롬프트 패턴 탐지")
				result.AnomalyType = "malicious_prompt"
				break
			}
		}

		// 반복적인 유사한 프롬프트 확인
		if aa.isRepeatedPrompt(ctx, pattern.UserID, promptContent) {
			result.Score += 0.4
			result.Evidence = append(result.Evidence, "반복적인 유사한 프롬프트")
		}
	}
}

// analyzeResponseTime은 응답 시간을 분석합니다.
func (aa *AgentAnalyzer) analyzeResponseTime(ctx context.Context, pattern *AgentRequestPattern, result *AgentAnomalyResult) {
	// 비정상적으로 빠른 응답 (봇 의심)
	if pattern.ResponseTime < 100*time.Millisecond {
		result.Score += 0.3
		result.Evidence = append(result.Evidence,
			fmt.Sprintf("비정상적으로 빠른 응답: %v", pattern.ResponseTime))
	}

	// 비정상적으로 느린 응답 (DoS 공격 의심)
	if pattern.ResponseTime > aa.config.SuspiciousResponseTime {
		result.Score += 0.5
		result.Evidence = append(result.Evidence,
			fmt.Sprintf("비정상적으로 느린 응답: %v", pattern.ResponseTime))
	}
}

// analyzeBehaviorPattern은 행동 패턴을 분석합니다.
func (aa *AgentAnalyzer) analyzeBehaviorPattern(ctx context.Context, pattern *AgentRequestPattern, result *AgentAnomalyResult) {
	// 사용자의 최근 요청 패턴 조회
	baselineKey := fmt.Sprintf("agent:baseline:user:%s", pattern.UserID)

	// 베이스라인과 비교한 이상 탐지
	if aa.isDeviationFromBaseline(ctx, baselineKey, pattern) {
		result.Score += 0.6
		result.Evidence = append(result.Evidence, "베이스라인으로부터의 행동 편차")
		result.AnomalyType = "behavioral_anomaly"
	}

	// 요청 타입 패턴 분석
	if aa.isSuspiciousRequestType(pattern.RequestType) {
		result.Score += 0.4
		result.Evidence = append(result.Evidence,
			fmt.Sprintf("의심스러운 요청 타입: %s", pattern.RequestType))
	}
}

// analyzeTemporalAnomaly는 시간적 이상을 분석합니다.
func (aa *AgentAnalyzer) analyzeTemporalAnomaly(ctx context.Context, pattern *AgentRequestPattern, result *AgentAnomalyResult) {
	hour := pattern.Timestamp.Hour()

	// 비정상적인 시간대 활동 (새벽 2-6시)
	if hour >= 2 && hour <= 6 {
		result.Score += 0.3
		result.Evidence = append(result.Evidence,
			fmt.Sprintf("비정상적인 시간대 활동: %d시", hour))
	}

	// 주말 또는 공휴일 비정상 활동 패턴
	weekday := pattern.Timestamp.Weekday()
	if weekday == time.Saturday || weekday == time.Sunday {
		// 주말 활동량이 평일의 2배 이상인 경우
		if aa.isExcessiveWeekendActivity(ctx, pattern.UserID) {
			result.Score += 0.2
			result.Evidence = append(result.Evidence, "주말 과도한 활동")
		}
	}
}

// calculateFinalResult는 최종 결과를 계산합니다.
func (aa *AgentAnalyzer) calculateFinalResult(result *AgentAnomalyResult) {
	// 점수를 0-1 범위로 정규화
	result.Score = math.Min(result.Score, 1.0)

	// 이상 여부 판단
	result.IsAnomalous = result.Score >= aa.config.AnomalyThreshold

	// 심각도 결정
	if result.Score >= 0.9 {
		result.Severity = SeverityCritical
	} else if result.Score >= 0.7 {
		result.Severity = SeverityHigh
	} else if result.Score >= 0.5 {
		result.Severity = SeverityMedium
	} else {
		result.Severity = SeverityLow
	}

	// 권장사항 생성
	result.Recommendations = aa.generateRecommendations(result)
}

// handleAnomalousRequest는 이상 요청을 처리합니다.
func (aa *AgentAnalyzer) handleAnomalousRequest(ctx context.Context, pattern *AgentRequestPattern, result *AgentAnomalyResult) {
	// 보안 이벤트 기록
	if aa.eventTracker != nil {
		event := &SecurityEvent{
			Type:      EventTypeAnomalous,
			Severity:  result.Severity,
			Source:    pattern.UserID,
			Target:    "AI Agent",
			UserID:    pattern.UserID,
			SessionID: pattern.SessionID,
			Details: map[string]interface{}{
				"anomaly_type":  result.AnomalyType,
				"score":         result.Score,
				"token_count":   pattern.TokenCount,
				"prompt_length": pattern.PromptLength,
				"response_time": pattern.ResponseTime.String(),
				"evidence":      result.Evidence,
			},
		}

		if err := aa.eventTracker.RecordEvent(ctx, event); err != nil {
			aa.logger.Error("이상 요청 이벤트 기록 실패", zap.Error(err))
		}
	}

	aa.logger.Warn("AI 에이전트 이상 요청 탐지됨",
		zap.String("user_id", pattern.UserID),
		zap.String("session_id", pattern.SessionID),
		zap.String("anomaly_type", result.AnomalyType),
		zap.Float64("score", result.Score),
		zap.String("severity", string(result.Severity)))
}

// 헬퍼 메서드들

func (aa *AgentAnalyzer) isRepeatedPrompt(ctx context.Context, userID, prompt string) bool {
	// 프롬프트 해시 계산 (간단한 예시)
	promptHash := fmt.Sprintf("%x", prompt[:min(len(prompt), 100)])
	key := fmt.Sprintf("agent:prompts:user:%s", userID)

	// 최근 프롬프트 해시들과 비교
	count, err := aa.redis.SAdd(ctx, key, promptHash).Result()
	if err != nil {
		return false
	}

	aa.redis.Expire(ctx, key, time.Hour)

	// 새로운 프롬프트가 아니면 반복으로 판단
	return count == 0
}

func (aa *AgentAnalyzer) isDeviationFromBaseline(ctx context.Context, baselineKey string, pattern *AgentRequestPattern) bool {
	// 베이스라인 데이터 조회 및 비교 로직
	// 여기서는 간단한 예시만 구현
	return false
}

func (aa *AgentAnalyzer) isSuspiciousRequestType(requestType string) bool {
	suspiciousTypes := []string{
		"file_access", "system_command", "network_request",
		"code_execution", "data_extraction",
	}

	for _, suspicious := range suspiciousTypes {
		if strings.Contains(strings.ToLower(requestType), suspicious) {
			return true
		}
	}

	return false
}

func (aa *AgentAnalyzer) isExcessiveWeekendActivity(ctx context.Context, userID string) bool {
	// 주말 활동량과 평일 활동량 비교 로직
	// 여기서는 간단한 예시만 구현
	return false
}

func (aa *AgentAnalyzer) generateRecommendations(result *AgentAnomalyResult) []string {
	recommendations := make([]string, 0)

	if result.Severity >= SeverityHigh {
		recommendations = append(recommendations, "사용자 계정 임시 제한 고려")
		recommendations = append(recommendations, "추가 인증 요구")
	}

	if result.Severity >= SeverityMedium {
		recommendations = append(recommendations, "추가 모니터링 적용")
		recommendations = append(recommendations, "요청 빈도 제한 강화")
	}

	if result.AnomalyType == "malicious_prompt" {
		recommendations = append(recommendations, "프롬프트 필터링 강화")
		recommendations = append(recommendations, "사용자 교육 필요")
	}

	if result.AnomalyType == "excessive_token_usage" {
		recommendations = append(recommendations, "토큰 사용량 제한 적용")
		recommendations = append(recommendations, "비용 모니터링 강화")
	}

	return recommendations
}

func (aa *AgentAnalyzer) storePatternData(ctx context.Context, pattern *AgentRequestPattern) {
	// 분석용 패턴 데이터 저장
	key := fmt.Sprintf("agent:patterns:user:%s:%d", pattern.UserID, pattern.Timestamp.Unix())
	data := map[string]interface{}{
		"request_type":  pattern.RequestType,
		"token_count":   pattern.TokenCount,
		"prompt_length": pattern.PromptLength,
		"response_time": pattern.ResponseTime.Nanoseconds(),
	}

	aa.redis.HMSet(ctx, key, data)
	aa.redis.Expire(ctx, key, aa.config.BaselineWindow)
}

// GetAnalytics는 에이전트 분석 통계를 반환합니다.
func (aa *AgentAnalyzer) GetAnalytics(ctx context.Context, userID string, period time.Duration) (map[string]interface{}, error) {
	analytics := make(map[string]interface{})

	// 사용자별 통계 수집
	if userID != "" {
		userStats, err := aa.getUserAnalytics(ctx, userID, period)
		if err == nil {
			analytics["user_stats"] = userStats
		}
	}

	// 전체 시스템 통계
	systemStats, err := aa.getSystemAnalytics(ctx, period)
	if err == nil {
		analytics["system_stats"] = systemStats
	}

	return analytics, nil
}

func (aa *AgentAnalyzer) getUserAnalytics(ctx context.Context, userID string, period time.Duration) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// 요청 빈도 통계
	freqKey := fmt.Sprintf("agent:freq:user:%s", userID)
	if count, err := aa.redis.Get(ctx, freqKey).Int(); err == nil {
		stats["requests_per_minute"] = count
	}

	// 토큰 사용량 통계
	tokenKey := fmt.Sprintf("agent:tokens:user:%s", userID)
	if tokens, err := aa.redis.Get(ctx, tokenKey).Int(); err == nil {
		stats["tokens_per_hour"] = tokens
	}

	return stats, nil
}

func (aa *AgentAnalyzer) getSystemAnalytics(ctx context.Context, period time.Duration) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// 전체 시스템 통계 수집 로직
	stats["total_users"] = 0
	stats["total_requests"] = 0
	stats["anomaly_rate"] = 0.0

	return stats, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
