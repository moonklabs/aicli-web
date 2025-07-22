package middleware

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"golang.org/x/time/rate"
	"go.uber.org/zap"

	"github.com/aicli/aicli-web/internal/models"
)

// AdvancedRateLimitConfig는 고급 Rate Limiting 설정입니다.
type AdvancedRateLimitConfig struct {
	// Redis 클라이언트
	Redis redis.UniversalClient
	
	// 기본 설정
	GlobalRateLimit    int           // 전역 초당 요청 제한
	UserRateLimit      int           // 사용자별 초당 요청 제한
	IPRateLimit        int           // IP별 초당 요청 제한
	EndpointRateLimit  map[string]int // 엔드포인트별 초당 요청 제한
	
	// 시간 윈도우 설정
	WindowSize         time.Duration // 슬라이딩 윈도우 크기
	CleanupInterval    time.Duration // 정리 작업 간격
	
	// 화이트리스트
	WhitelistedIPs     []string      // 제외할 IP 목록
	WhitelistedUsers   []string      // 제외할 사용자 목록
	
	// 지능형 설정
	BurstMultiplier    float64       // 버스트 허용 배수
	AdaptiveScaling    bool          // 적응형 스케일링 활성화
	GeoBasedLimits     bool          // 지역 기반 제한 활성화
	
	// 보안 설정
	SuspiciousThreshold int          // 의심스러운 활동 임계값
	AutoBlockDuration   time.Duration // 자동 차단 지속 시간
	
	Logger *zap.Logger
}

// AdvancedRateLimiter는 Redis 기반 고급 Rate Limiter입니다.
type AdvancedRateLimiter struct {
	config     *AdvancedRateLimitConfig
	redis      redis.UniversalClient
	logger     *zap.Logger
	
	// 로컬 rate limiter (fallback)
	localLimiter *rate.Limiter
}

// LimitKey는 Rate Limiting 키 타입을 정의합니다.
type LimitKey struct {
	Type     string // "global", "user", "ip", "endpoint"
	Value    string // 실제 키 값
	Endpoint string // 엔드포인트 (필요한 경우)
}

// NewAdvancedRateLimiter는 새로운 고급 Rate Limiter를 생성합니다.
func NewAdvancedRateLimiter(config *AdvancedRateLimitConfig) *AdvancedRateLimiter {
	if config.WindowSize == 0 {
		config.WindowSize = time.Minute
	}
	if config.CleanupInterval == 0 {
		config.CleanupInterval = 5 * time.Minute
	}
	if config.BurstMultiplier == 0 {
		config.BurstMultiplier = 2.0
	}
	if config.AutoBlockDuration == 0 {
		config.AutoBlockDuration = 15 * time.Minute
	}
	if config.SuspiciousThreshold == 0 {
		config.SuspiciousThreshold = 100
	}

	limiter := &AdvancedRateLimiter{
		config:       config,
		redis:        config.Redis,
		logger:       config.Logger,
		localLimiter: rate.NewLimiter(rate.Limit(config.GlobalRateLimit), int(float64(config.GlobalRateLimit)*config.BurstMultiplier)),
	}

	// 정기적인 정리 작업 시작
	go limiter.cleanupExpiredEntries()

	return limiter
}

// AdvancedRateLimit은 고급 Rate Limiting 미들웨어를 생성합니다.
func AdvancedRateLimit(config *AdvancedRateLimitConfig) gin.HandlerFunc {
	limiter := NewAdvancedRateLimiter(config)
	return limiter.Handler()
}

// Handler는 미들웨어 핸들러를 반환합니다.
func (arl *AdvancedRateLimiter) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Redis 연결 확인
		if arl.redis == nil {
			// Fallback to local limiter
			if !arl.localLimiter.Allow() {
				arl.handleRateLimitExceeded(c, "global", 0)
				return
			}
			c.Next()
			return
		}

		// IP 및 사용자 정보 추출
		clientIP := c.ClientIP()
		userID := arl.extractUserID(c)
		endpoint := c.Request.URL.Path

		// 화이트리스트 확인
		if arl.isWhitelisted(clientIP, userID) {
			c.Next()
			return
		}

		// 차단된 IP 확인
		if arl.isBlocked(c.Request.Context(), clientIP) {
			arl.handleBlocked(c, clientIP)
			return
		}

		// 다층 Rate Limit 체크
		limitChecks := []LimitKey{
			{Type: "global", Value: "global", Endpoint: ""},
			{Type: "ip", Value: clientIP, Endpoint: ""},
			{Type: "endpoint", Value: endpoint, Endpoint: endpoint},
		}

		if userID != "" {
			limitChecks = append(limitChecks, LimitKey{
				Type: "user", Value: userID, Endpoint: "",
			})
		}

		// 각 레이어별 Rate Limit 검사
		for _, key := range limitChecks {
			allowed, remaining, resetTime := arl.checkRateLimit(c.Request.Context(), key)
			
			// Rate Limit 헤더 설정
			arl.setRateLimitHeaders(c, key.Type, allowed, remaining, resetTime)
			
			if !allowed {
				// 의심스러운 활동 기록
				arl.recordSuspiciousActivity(c.Request.Context(), clientIP, userID, key.Type)
				
				arl.handleRateLimitExceeded(c, key.Type, resetTime)
				return
			}
		}

		// 요청 허용
		c.Next()
		
		// 응답 상태에 따른 후처리
		arl.handlePostResponse(c.Request.Context(), c, clientIP, userID, endpoint)
	}
}

// checkRateLimit은 특정 키에 대한 Rate Limit을 검사합니다.
func (arl *AdvancedRateLimiter) checkRateLimit(ctx context.Context, key LimitKey) (allowed bool, remaining int, resetTime time.Time) {
	redisKey := arl.generateRedisKey(key)
	limit := arl.getLimitForKey(key)
	
	if limit <= 0 {
		return true, 999, time.Now().Add(arl.config.WindowSize)
	}

	// 슬라이딩 윈도우 알고리즘 사용
	now := time.Now()
	windowStart := now.Add(-arl.config.WindowSize)
	
	pipe := arl.redis.Pipeline()
	
	// 이전 요청들을 윈도우 시작 전 것들 제거
	pipe.ZRemRangeByScore(ctx, redisKey, "0", strconv.FormatInt(windowStart.UnixNano(), 10))
	
	// 현재 윈도우의 요청 수 조회
	countCmd := pipe.ZCard(ctx, redisKey)
	
	// 현재 요청 추가
	pipe.ZAdd(ctx, redisKey, &redis.Z{
		Score:  float64(now.UnixNano()),
		Member: fmt.Sprintf("%d:%s", now.UnixNano(), arl.generateRequestID()),
	})
	
	// TTL 설정
	pipe.Expire(ctx, redisKey, arl.config.WindowSize*2)
	
	_, err := pipe.Exec(ctx)
	if err != nil {
		arl.logger.Error("Redis pipeline 실행 실패", zap.Error(err))
		// Fallback to local limiter
		return arl.localLimiter.Allow(), 0, now.Add(arl.config.WindowSize)
	}

	currentCount := int(countCmd.Val()) + 1 // +1 for current request
	allowed = currentCount <= limit
	remaining = limit - currentCount
	if remaining < 0 {
		remaining = 0
	}
	
	resetTime = now.Add(arl.config.WindowSize)
	
	return allowed, remaining, resetTime
}

// generateRedisKey는 Redis 키를 생성합니다.
func (arl *AdvancedRateLimiter) generateRedisKey(key LimitKey) string {
	switch key.Type {
	case "global":
		return "rate_limit:global"
	case "user":
		return fmt.Sprintf("rate_limit:user:%s", key.Value)
	case "ip":
		return fmt.Sprintf("rate_limit:ip:%s", key.Value)
	case "endpoint":
		return fmt.Sprintf("rate_limit:endpoint:%s", strings.ReplaceAll(key.Value, "/", "_"))
	default:
		return fmt.Sprintf("rate_limit:custom:%s:%s", key.Type, key.Value)
	}
}

// getLimitForKey는 키 타입에 따른 제한값을 반환합니다.
func (arl *AdvancedRateLimiter) getLimitForKey(key LimitKey) int {
	switch key.Type {
	case "global":
		return arl.config.GlobalRateLimit
	case "user":
		return arl.config.UserRateLimit
	case "ip":
		return arl.config.IPRateLimit
	case "endpoint":
		if limit, exists := arl.config.EndpointRateLimit[key.Endpoint]; exists {
			return limit
		}
		return 0 // 제한 없음
	default:
		return arl.config.GlobalRateLimit
	}
}

// extractUserID는 컨텍스트에서 사용자 ID를 추출합니다.
func (arl *AdvancedRateLimiter) extractUserID(c *gin.Context) string {
	if userID, exists := c.Get("user_id"); exists {
		if uid, ok := userID.(string); ok {
			return uid
		}
	}
	
	// JWT 토큰에서 사용자 ID 추출 (간단한 구현)
	authHeader := c.GetHeader("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		// 실제 구현에서는 JWT 파싱 필요
		// 여기서는 간단히 처리
		return ""
	}
	
	return ""
}

// isWhitelisted는 IP나 사용자가 화이트리스트에 있는지 확인합니다.
func (arl *AdvancedRateLimiter) isWhitelisted(clientIP, userID string) bool {
	// IP 화이트리스트 확인
	for _, whiteIP := range arl.config.WhitelistedIPs {
		if arl.matchesIP(clientIP, whiteIP) {
			return true
		}
	}
	
	// 사용자 화이트리스트 확인
	if userID != "" {
		for _, whiteUser := range arl.config.WhitelistedUsers {
			if userID == whiteUser {
				return true
			}
		}
	}
	
	return false
}

// matchesIP는 IP 매칭을 수행합니다 (CIDR 지원).
func (arl *AdvancedRateLimiter) matchesIP(clientIP, pattern string) bool {
	// 정확한 매치
	if clientIP == pattern {
		return true
	}
	
	// CIDR 매치
	_, cidr, err := net.ParseCIDR(pattern)
	if err == nil {
		ip := net.ParseIP(clientIP)
		if ip != nil {
			return cidr.Contains(ip)
		}
	}
	
	return false
}

// isBlocked는 IP가 차단되었는지 확인합니다.
func (arl *AdvancedRateLimiter) isBlocked(ctx context.Context, clientIP string) bool {
	blockKey := fmt.Sprintf("blocked:ip:%s", clientIP)
	exists, err := arl.redis.Exists(ctx, blockKey).Result()
	if err != nil {
		arl.logger.Error("차단 상태 확인 실패", zap.Error(err))
		return false
	}
	return exists > 0
}

// recordSuspiciousActivity는 의심스러운 활동을 기록합니다.
func (arl *AdvancedRateLimiter) recordSuspiciousActivity(ctx context.Context, clientIP, userID, limitType string) {
	// 의심스러운 활동 카운터 증가
	suspiciousKey := fmt.Sprintf("suspicious:ip:%s", clientIP)
	count, err := arl.redis.Incr(ctx, suspiciousKey).Result()
	if err != nil {
		arl.logger.Error("의심스러운 활동 기록 실패", zap.Error(err))
		return
	}
	
	// TTL 설정 (24시간)
	arl.redis.Expire(ctx, suspiciousKey, 24*time.Hour)
	
	// 임계값 초과 시 자동 차단
	if count >= int64(arl.config.SuspiciousThreshold) {
		arl.blockIP(ctx, clientIP, arl.config.AutoBlockDuration)
		
		arl.logger.Warn("IP 자동 차단",
			zap.String("ip", clientIP),
			zap.String("user_id", userID),
			zap.String("limit_type", limitType),
			zap.Int64("violation_count", count))
	}
}

// blockIP는 IP를 차단합니다.
func (arl *AdvancedRateLimiter) blockIP(ctx context.Context, clientIP string, duration time.Duration) {
	blockKey := fmt.Sprintf("blocked:ip:%s", clientIP)
	err := arl.redis.Set(ctx, blockKey, time.Now().Unix(), duration).Err()
	if err != nil {
		arl.logger.Error("IP 차단 실패", zap.Error(err))
	}
}

// setRateLimitHeaders는 Rate Limit 헤더를 설정합니다.
func (arl *AdvancedRateLimiter) setRateLimitHeaders(c *gin.Context, limitType string, allowed bool, remaining int, resetTime time.Time) {
	prefix := fmt.Sprintf("X-RateLimit-%s-", strings.Title(limitType))
	
	c.Header(prefix+"Limit", strconv.Itoa(arl.getLimitForKey(LimitKey{Type: limitType})))
	c.Header(prefix+"Remaining", strconv.Itoa(remaining))
	c.Header(prefix+"Reset", strconv.FormatInt(resetTime.Unix(), 10))
}

// handleRateLimitExceeded는 Rate Limit 초과 시 처리합니다.
func (arl *AdvancedRateLimiter) handleRateLimitExceeded(c *gin.Context, limitType string, resetTime time.Time) {
	retryAfter := int(time.Until(resetTime).Seconds())
	if retryAfter < 1 {
		retryAfter = 1
	}
	
	c.Header("Retry-After", strconv.Itoa(retryAfter))
	
	c.JSON(http.StatusTooManyRequests, gin.H{
		"error":     "Too Many Requests",
		"message":   fmt.Sprintf("Rate limit exceeded for %s", limitType),
		"code":      "RATE_LIMIT_EXCEEDED",
		"type":      limitType,
		"retry_after": retryAfter,
	})
	
	c.Abort()
}

// handleBlocked는 차단된 IP 처리합니다.
func (arl *AdvancedRateLimiter) handleBlocked(c *gin.Context, clientIP string) {
	c.JSON(http.StatusForbidden, gin.H{
		"error":   "Forbidden",
		"message": "Your IP address has been temporarily blocked due to suspicious activity",
		"code":    "IP_BLOCKED",
	})
	
	c.Abort()
}

// handlePostResponse는 응답 후 처리를 수행합니다.
func (arl *AdvancedRateLimiter) handlePostResponse(ctx context.Context, c *gin.Context, clientIP, userID, endpoint string) {
	statusCode := c.Writer.Status()
	
	// 실패한 요청에 대한 추가 처리
	if statusCode >= 400 {
		// 실패 횟수 기록
		failKey := fmt.Sprintf("failures:ip:%s", clientIP)
		arl.redis.Incr(ctx, failKey)
		arl.redis.Expire(ctx, failKey, time.Hour)
	}
}

// cleanupExpiredEntries는 만료된 항목들을 정리합니다.
func (arl *AdvancedRateLimiter) cleanupExpiredEntries() {
	ticker := time.NewTicker(arl.config.CleanupInterval)
	defer ticker.Stop()
	
	for range ticker.C {
		ctx := context.Background()
		now := time.Now()
		
		// 만료된 Rate Limit 항목 정리
		pattern := "rate_limit:*"
		arl.cleanupByPattern(ctx, pattern, now.Add(-arl.config.WindowSize*2))
		
		// 만료된 의심스러운 활동 기록 정리
		pattern = "suspicious:*"
		arl.cleanupByPattern(ctx, pattern, now.Add(-24*time.Hour))
	}
}

// cleanupByPattern은 패턴에 맞는 키들을 정리합니다.
func (arl *AdvancedRateLimiter) cleanupByPattern(ctx context.Context, pattern string, before time.Time) {
	var cursor uint64
	for {
		keys, nextCursor, err := arl.redis.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			arl.logger.Error("Redis scan 실패", zap.Error(err))
			break
		}
		
		for _, key := range keys {
			// ZSet의 경우 시간 기반 정리
			if strings.Contains(key, "rate_limit:") {
				arl.redis.ZRemRangeByScore(ctx, key, "0", strconv.FormatInt(before.UnixNano(), 10))
			}
		}
		
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
}

// generateRequestID는 고유한 요청 ID를 생성합니다.
func (arl *AdvancedRateLimiter) generateRequestID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// GetStats는 Rate Limiting 통계를 반환합니다.
func (arl *AdvancedRateLimiter) GetStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	
	// 활성 IP 수
	activeIPsPattern := "rate_limit:ip:*"
	activeIPs, err := arl.getKeyCount(ctx, activeIPsPattern)
	if err == nil {
		stats["active_ips"] = activeIPs
	}
	
	// 활성 사용자 수
	activeUsersPattern := "rate_limit:user:*"
	activeUsers, err := arl.getKeyCount(ctx, activeUsersPattern)
	if err == nil {
		stats["active_users"] = activeUsers
	}
	
	// 차단된 IP 수
	blockedIPsPattern := "blocked:ip:*"
	blockedIPs, err := arl.getKeyCount(ctx, blockedIPsPattern)
	if err == nil {
		stats["blocked_ips"] = blockedIPs
	}
	
	// 의심스러운 활동 수
	suspiciousPattern := "suspicious:ip:*"
	suspiciousCount, err := arl.getKeyCount(ctx, suspiciousPattern)
	if err == nil {
		stats["suspicious_activities"] = suspiciousCount
	}
	
	return stats, nil
}

// getKeyCount는 패턴에 맞는 키의 수를 반환합니다.
func (arl *AdvancedRateLimiter) getKeyCount(ctx context.Context, pattern string) (int, error) {
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

// Close는 리소스를 정리합니다.
func (arl *AdvancedRateLimiter) Close() error {
	// 정리 작업이 필요한 경우 구현
	return nil
}