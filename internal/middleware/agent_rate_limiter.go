package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"

	"github.com/aicli/aicli-web/internal/security"
)

// AgentRateLimiterConfig는 에이전트 Rate Limiter 설정입니다.
type AgentRateLimiterConfig struct {
	Redis        redis.UniversalClient
	EventTracker *security.EventTracker

	// 기본 제한 설정
	DefaultRequestsPerMinute int           // 기본 분당 요청 수
	DefaultTokensPerHour     int           // 기본 시간당 토큰 수
	BurstAllowance          int           // 버스트 허용량
	BlockDuration           time.Duration // 차단 지속 시간

	// 동적 조정 설정
	DynamicAdjustmentEnabled bool    // 동적 조정 활성화
	AdjustmentFactor        float64 // 조정 계수 (0.1 ~ 2.0)
	BaselineWindow          time.Duration // 베이스라인 계산 윈도우

	// 사용자별 개별 제한
	UserLimits map[string]*UserRateLimit // 사용자별 개별 제한

	// 화이트리스트
	WhitelistedIPs    []string // 화이트리스트 IP
	WhitelistedUsers  []string // 화이트리스트 사용자

	Logger *zap.Logger
}

// UserRateLimit은 사용자별 개별 제한을 정의합니다.
type UserRateLimit struct {
	RequestsPerMinute int           // 분당 요청 수
	TokensPerHour     int           // 시간당 토큰 수
	BurstAllowance   int           // 버스트 허용량
	ValidUntil       *time.Time    // 제한 유효 기간
}

// AgentRateLimiter는 AI 에이전트 전용 Rate Limiter입니다.
type AgentRateLimiter struct {
	config       *AgentRateLimiterConfig
	redis        redis.UniversalClient
	logger       *zap.Logger
	eventTracker *security.EventTracker
}

// RateLimitResult는 Rate Limit 검사 결과입니다.
type RateLimitResult struct {
	Allowed           bool          `json:"allowed"`
	Reason            string        `json:"reason,omitempty"`
	RequestsRemaining int           `json:"requests_remaining"`
	TokensRemaining   int           `json:"tokens_remaining"`
	ResetTime         time.Time     `json:"reset_time"`
	RetryAfter        time.Duration `json:"retry_after,omitempty"`
}

// DefaultAgentRateLimiterConfig는 기본 설정을 반환합니다.
func DefaultAgentRateLimiterConfig() *AgentRateLimiterConfig {
	return &AgentRateLimiterConfig{
		DefaultRequestsPerMinute: 60,
		DefaultTokensPerHour:     5000,
		BurstAllowance:          10,
		BlockDuration:           15 * time.Minute,
		DynamicAdjustmentEnabled: true,
		AdjustmentFactor:        1.0,
		BaselineWindow:          24 * time.Hour,
		UserLimits:              make(map[string]*UserRateLimit),
		WhitelistedIPs:          make([]string, 0),
		WhitelistedUsers:        make([]string, 0),
	}
}

// NewAgentRateLimiter는 새로운 에이전트 Rate Limiter를 생성합니다.
func NewAgentRateLimiter(config *AgentRateLimiterConfig) *AgentRateLimiter {
	if config == nil {
		config = DefaultAgentRateLimiterConfig()
	}

	return &AgentRateLimiter{
		config:       config,
		redis:        config.Redis,
		logger:       config.Logger,
		eventTracker: config.EventTracker,
	}
}

// AgentRateLimit는 Gin 미들웨어로 사용되는 Rate Limiter입니다.
func (arl *AgentRateLimiter) AgentRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 기본 정보 추출
		userID := arl.extractUserID(c)
		sessionID := arl.extractSessionID(c)
		ipAddress := arl.getClientIP(c)
		tokenCount := arl.extractTokenCount(c)

		// 화이트리스트 검사
		if arl.isWhitelisted(userID, ipAddress) {
			c.Next()
			return
		}

		// 차단된 사용자/IP 확인
		if arl.isBlocked(c.Request.Context(), userID, ipAddress) {
			arl.handleBlocked(c, userID, ipAddress)
			return
		}

		// Rate Limit 검사
		result := arl.checkRateLimit(c.Request.Context(), userID, sessionID, ipAddress, tokenCount)
		
		if !result.Allowed {
			arl.handleRateLimited(c, userID, result)
			return
		}

		// 응답 헤더 설정
		arl.setRateLimitHeaders(c, result)

		c.Next()
	}
}

// checkRateLimit은 Rate Limit을 검사합니다.
func (arl *AgentRateLimiter) checkRateLimit(ctx context.Context, userID, sessionID, ipAddress string, tokenCount int) *RateLimitResult {
	now := time.Now()
	
	// 사용자별 제한 가져오기
	userLimit := arl.getUserLimit(userID)
	
	// 동적 조정 적용
	if arl.config.DynamicAdjustmentEnabled {
		userLimit = arl.applyDynamicAdjustment(ctx, userID, userLimit)
	}

	// 요청 빈도 검사
	requestResult := arl.checkRequestRate(ctx, userID, ipAddress, userLimit, now)
	if !requestResult.Allowed {
		return requestResult
	}

	// 토큰 사용량 검사
	tokenResult := arl.checkTokenRate(ctx, userID, tokenCount, userLimit, now)
	if !tokenResult.Allowed {
		return tokenResult
	}

	// 버스트 제한 검사
	burstResult := arl.checkBurstLimit(ctx, userID, userLimit, now)
	if !burstResult.Allowed {
		return burstResult
	}

	// 모든 검사 통과
	return &RateLimitResult{
		Allowed:           true,
		RequestsRemaining: requestResult.RequestsRemaining,
		TokensRemaining:   tokenResult.TokensRemaining,
		ResetTime:         now.Add(time.Minute).Truncate(time.Minute),
	}
}

// checkRequestRate는 요청 빈도를 검사합니다.
func (arl *AgentRateLimiter) checkRequestRate(ctx context.Context, userID, ipAddress string, limit *UserRateLimit, now time.Time) *RateLimitResult {
	// 사용자별 요청 수 확인
	userKey := fmt.Sprintf("rate:requests:user:%s:%d", userID, now.Unix()/60) // 분 단위
	userCount, err := arl.redis.Incr(ctx, userKey).Result()
	if err != nil {
		arl.logger.Error("Redis 요청 실패", zap.Error(err))
		return &RateLimitResult{Allowed: true} // Redis 오류 시 허용
	}

	if userCount == 1 {
		arl.redis.Expire(ctx, userKey, time.Minute)
	}

	if userCount > int64(limit.RequestsPerMinute) {
		return &RateLimitResult{
			Allowed:           false,
			Reason:            "분당 요청 수 초과",
			RequestsRemaining: 0,
			ResetTime:         now.Add(time.Minute).Truncate(time.Minute),
			RetryAfter:        time.Until(now.Add(time.Minute).Truncate(time.Minute)),
		}
	}

	// IP별 요청 수 확인 (보조 제한)
	ipKey := fmt.Sprintf("rate:requests:ip:%s:%d", ipAddress, now.Unix()/60)
	ipCount, err := arl.redis.Incr(ctx, ipKey).Result()
	if err == nil {
		if ipCount == 1 {
			arl.redis.Expire(ctx, ipKey, time.Minute)
		}
		
		// IP별 제한은 사용자별 제한의 2배
		if ipCount > int64(limit.RequestsPerMinute*2) {
			return &RateLimitResult{
				Allowed:    false,
				Reason:     "IP별 요청 수 초과",
				RetryAfter: time.Minute,
			}
		}
	}

	return &RateLimitResult{
		Allowed:           true,
		RequestsRemaining: int(int64(limit.RequestsPerMinute) - userCount),
	}
}

// checkTokenRate는 토큰 사용률을 검사합니다.
func (arl *AgentRateLimiter) checkTokenRate(ctx context.Context, userID string, tokenCount int, limit *UserRateLimit, now time.Time) *RateLimitResult {
	tokenKey := fmt.Sprintf("rate:tokens:user:%s:%d", userID, now.Unix()/3600) // 시간 단위
	
	currentTokens, err := arl.redis.IncrBy(ctx, tokenKey, int64(tokenCount)).Result()
	if err != nil {
		arl.logger.Error("Redis 토큰 추가 실패", zap.Error(err))
		return &RateLimitResult{Allowed: true}
	}

	if currentTokens == int64(tokenCount) {
		arl.redis.Expire(ctx, tokenKey, time.Hour)
	}

	if currentTokens > int64(limit.TokensPerHour) {
		return &RateLimitResult{
			Allowed:         false,
			Reason:          "시간당 토큰 사용량 초과",
			TokensRemaining: 0,
			ResetTime:       now.Add(time.Hour).Truncate(time.Hour),
			RetryAfter:      time.Until(now.Add(time.Hour).Truncate(time.Hour)),
		}
	}

	return &RateLimitResult{
		Allowed:         true,
		TokensRemaining: int(int64(limit.TokensPerHour) - currentTokens),
	}
}

// checkBurstLimit은 버스트 제한을 검사합니다.
func (arl *AgentRateLimiter) checkBurstLimit(ctx context.Context, userID string, limit *UserRateLimit, now time.Time) *RateLimitResult {
	// 최근 10초간의 요청 수 확인
	burstKey := fmt.Sprintf("rate:burst:user:%s:%d", userID, now.Unix()/10) // 10초 단위
	burstCount, err := arl.redis.Incr(ctx, burstKey).Result()
	if err != nil {
		return &RateLimitResult{Allowed: true}
	}

	if burstCount == 1 {
		arl.redis.Expire(ctx, burstKey, 10*time.Second)
	}

	if burstCount > int64(limit.BurstAllowance) {
		return &RateLimitResult{
			Allowed:    false,
			Reason:     "버스트 제한 초과",
			RetryAfter: 10 * time.Second,
		}
	}

	return &RateLimitResult{Allowed: true}
}

// applyDynamicAdjustment는 동적 조정을 적용합니다.
func (arl *AgentRateLimiter) applyDynamicAdjustment(ctx context.Context, userID string, baseLimit *UserRateLimit) *UserRateLimit {
	// 사용자의 과거 사용 패턴을 분석하여 동적으로 조정
	adjustedLimit := *baseLimit // 복사
	
	// 최근 24시간 사용 패턴 분석
	avgUsage := arl.getAverageUsage(ctx, userID, arl.config.BaselineWindow)
	
	// 평균 사용량 기반 조정
	if avgUsage > 0 {
		adjustment := arl.config.AdjustmentFactor
		
		// 사용량이 제한의 80% 이상이면 여유분 제공
		if avgUsage >= float64(baseLimit.RequestsPerMinute)*0.8 {
			adjustment = 1.2
		} else if avgUsage <= float64(baseLimit.RequestsPerMinute)*0.3 {
			// 사용량이 30% 이하면 제한 강화
			adjustment = 0.8
		}
		
		adjustedLimit.RequestsPerMinute = int(float64(baseLimit.RequestsPerMinute) * adjustment)
		adjustedLimit.TokensPerHour = int(float64(baseLimit.TokensPerHour) * adjustment)
		adjustedLimit.BurstAllowance = int(float64(baseLimit.BurstAllowance) * adjustment)
	}

	return &adjustedLimit
}

// getUserLimit은 사용자별 제한을 가져옵니다.
func (arl *AgentRateLimiter) getUserLimit(userID string) *UserRateLimit {
	// 사용자별 개별 제한 확인
	if customLimit, exists := arl.config.UserLimits[userID]; exists {
		// 유효 기간 확인
		if customLimit.ValidUntil == nil || time.Now().Before(*customLimit.ValidUntil) {
			return customLimit
		}
	}

	// 기본 제한 반환
	return &UserRateLimit{
		RequestsPerMinute: arl.config.DefaultRequestsPerMinute,
		TokensPerHour:     arl.config.DefaultTokensPerHour,
		BurstAllowance:   arl.config.BurstAllowance,
	}
}

// isWhitelisted는 화이트리스트 여부를 확인합니다.
func (arl *AgentRateLimiter) isWhitelisted(userID, ipAddress string) bool {
	// 사용자 화이트리스트 확인
	for _, whiteUser := range arl.config.WhitelistedUsers {
		if whiteUser == userID {
			return true
		}
	}

	// IP 화이트리스트 확인
	for _, whiteIP := range arl.config.WhitelistedIPs {
		if whiteIP == ipAddress {
			return true
		}
	}

	return false
}

// isBlocked는 차단 여부를 확인합니다.
func (arl *AgentRateLimiter) isBlocked(ctx context.Context, userID, ipAddress string) bool {
	// 사용자 차단 확인
	userBlockKey := fmt.Sprintf("blocked:user:%s", userID)
	if arl.redis.Exists(ctx, userBlockKey).Val() > 0 {
		return true
	}

	// IP 차단 확인
	ipBlockKey := fmt.Sprintf("blocked:ip:%s", ipAddress)
	if arl.redis.Exists(ctx, ipBlockKey).Val() > 0 {
		return true
	}

	return false
}

// BlockUser는 사용자를 차단합니다.
func (arl *AgentRateLimiter) BlockUser(ctx context.Context, userID string, duration time.Duration, reason string) error {
	blockKey := fmt.Sprintf("blocked:user:%s", userID)
	err := arl.redis.Set(ctx, blockKey, reason, duration).Err()
	if err != nil {
		return fmt.Errorf("사용자 차단 실패: %w", err)
	}

	// 차단 이벤트 기록
	if arl.eventTracker != nil {
		event := &security.SecurityEvent{
			Type:     security.EventTypeBlocked,
			Severity: security.SeverityHigh,
			Source:   userID,
			Target:   "Agent System",
			UserID:   userID,
			Details: map[string]interface{}{
				"block_type": "user_block",
				"duration":   duration.String(),
				"reason":     reason,
			},
		}
		arl.eventTracker.RecordEvent(ctx, event)
	}

	arl.logger.Warn("사용자 차단됨",
		zap.String("user_id", userID),
		zap.Duration("duration", duration),
		zap.String("reason", reason))

	return nil
}

// BlockIP는 IP를 차단합니다.
func (arl *AgentRateLimiter) BlockIP(ctx context.Context, ipAddress string, duration time.Duration, reason string) error {
	blockKey := fmt.Sprintf("blocked:ip:%s", ipAddress)
	err := arl.redis.Set(ctx, blockKey, reason, duration).Err()
	if err != nil {
		return fmt.Errorf("IP 차단 실패: %w", err)
	}

	// 차단 이벤트 기록
	if arl.eventTracker != nil {
		event := &security.SecurityEvent{
			Type:      security.EventTypeBlocked,
			Severity:  security.SeverityHigh,
			Source:    ipAddress,
			Target:    "Agent System",
			IPAddress: ipAddress,
			Details: map[string]interface{}{
				"block_type": "ip_block",
				"duration":   duration.String(),
				"reason":     reason,
			},
		}
		arl.eventTracker.RecordEvent(ctx, event)
	}

	arl.logger.Warn("IP 차단됨",
		zap.String("ip", ipAddress),
		zap.Duration("duration", duration),
		zap.String("reason", reason))

	return nil
}

// handleRateLimited는 Rate Limit 초과 시 처리합니다.
func (arl *AgentRateLimiter) handleRateLimited(c *gin.Context, userID string, result *RateLimitResult) {
	// Rate Limit 위반 이벤트 기록
	if arl.eventTracker != nil {
		event := &security.SecurityEvent{
			Type:      security.EventTypeRateLimit,
			Severity:  security.SeverityMedium,
			Source:    userID,
			Target:    "Agent System",
			UserID:    userID,
			IPAddress: arl.getClientIP(c),
			Details: map[string]interface{}{
				"reason":             result.Reason,
				"requests_remaining": result.RequestsRemaining,
				"tokens_remaining":   result.TokensRemaining,
				"retry_after":        result.RetryAfter.String(),
			},
		}
		arl.eventTracker.RecordEvent(c.Request.Context(), event)
	}

	// 응답 헤더 설정
	c.Header("X-RateLimit-Limit", strconv.Itoa(arl.config.DefaultRequestsPerMinute))
	c.Header("X-RateLimit-Remaining", strconv.Itoa(result.RequestsRemaining))
	c.Header("X-RateLimit-Reset", strconv.FormatInt(result.ResetTime.Unix(), 10))
	
	if result.RetryAfter > 0 {
		c.Header("Retry-After", strconv.Itoa(int(result.RetryAfter.Seconds())))
	}

	c.JSON(http.StatusTooManyRequests, gin.H{
		"error":   "Too Many Requests",
		"message": result.Reason,
		"retry_after": result.RetryAfter.Seconds(),
	})
	c.Abort()
}

// handleBlocked는 차단된 사용자/IP 처리합니다.
func (arl *AgentRateLimiter) handleBlocked(c *gin.Context, userID, ipAddress string) {
	c.JSON(http.StatusForbidden, gin.H{
		"error":   "Blocked",
		"message": "액세스가 일시적으로 차단되었습니다",
	})
	c.Abort()
}

// setRateLimitHeaders는 Rate Limit 관련 헤더를 설정합니다.
func (arl *AgentRateLimiter) setRateLimitHeaders(c *gin.Context, result *RateLimitResult) {
	c.Header("X-RateLimit-Requests-Remaining", strconv.Itoa(result.RequestsRemaining))
	c.Header("X-RateLimit-Tokens-Remaining", strconv.Itoa(result.TokensRemaining))
	c.Header("X-RateLimit-Reset", strconv.FormatInt(result.ResetTime.Unix(), 10))
}

// 헬퍼 메서드들

func (arl *AgentRateLimiter) extractUserID(c *gin.Context) string {
	if userID, exists := c.Get("user_id"); exists {
		if str, ok := userID.(string); ok {
			return str
		}
	}
	return "anonymous"
}

func (arl *AgentRateLimiter) extractSessionID(c *gin.Context) string {
	if sessionID, exists := c.Get("session_id"); exists {
		if str, ok := sessionID.(string); ok {
			return str
		}
	}
	return ""
}

func (arl *AgentRateLimiter) extractTokenCount(c *gin.Context) int {
	if tokenCount, exists := c.Get("token_count"); exists {
		if count, ok := tokenCount.(int); ok {
			return count
		}
	}
	// Content-Length 기반 토큰 수 추정 (대략적)
	contentLength := c.Request.ContentLength
	if contentLength > 0 {
		return int(contentLength / 4) // 대략 4바이트당 1토큰
	}
	return 100 // 기본값
}

func (arl *AgentRateLimiter) getClientIP(c *gin.Context) string {
	// X-Forwarded-For 헤더 확인
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		return xff
	}
	// X-Real-IP 헤더 확인
	if xri := c.GetHeader("X-Real-IP"); xri != "" {
		return xri
	}
	return c.ClientIP()
}

func (arl *AgentRateLimiter) getAverageUsage(ctx context.Context, userID string, window time.Duration) float64 {
	// 과거 사용 패턴 분석 로직
	// 실제 구현에서는 더 정교한 분석 필요
	return 0.0
}

// GetStats는 Rate Limiter 통계를 반환합니다.
func (arl *AgentRateLimiter) GetStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// 차단된 사용자 수
	blockedUsers := 0
	pattern := "blocked:user:*"
	if keys, err := arl.countKeysByPattern(ctx, pattern); err == nil {
		blockedUsers = keys
	}

	// 차단된 IP 수
	blockedIPs := 0
	pattern = "blocked:ip:*"
	if keys, err := arl.countKeysByPattern(ctx, pattern); err == nil {
		blockedIPs = keys
	}

	stats["blocked_users"] = blockedUsers
	stats["blocked_ips"] = blockedIPs
	stats["config"] = map[string]interface{}{
		"default_requests_per_minute": arl.config.DefaultRequestsPerMinute,
		"default_tokens_per_hour":     arl.config.DefaultTokensPerHour,
		"burst_allowance":            arl.config.BurstAllowance,
		"dynamic_adjustment_enabled": arl.config.DynamicAdjustmentEnabled,
	}

	return stats, nil
}

func (arl *AgentRateLimiter) countKeysByPattern(ctx context.Context, pattern string) (int, error) {
	var cursor uint64
	var count int

	for {
		keys, nextCursor, err := arl.redis.Scan(ctx, cursor, pattern, 100).Result()
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