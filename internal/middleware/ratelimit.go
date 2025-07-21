package middleware

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/aicli/aicli-web/internal/ratelimit"
	"github.com/aicli/aicli-web/internal/auth"
)

// RateLimitConfig는 Rate Limit 미들웨어 설정입니다.
type RateLimitConfig struct {
	// Enabled는 Rate Limiting 활성화 여부입니다.
	Enabled bool
	
	// DefaultConfig는 기본 Rate Limit 설정입니다.
	DefaultConfig *ratelimit.LimiterConfig
	
	// AuthenticatedConfig는 인증된 사용자에 대한 설정입니다.
	AuthenticatedConfig *ratelimit.LimiterConfig
	
	// EndpointConfigs는 엔드포인트별 설정입니다.
	EndpointConfigs map[string]*ratelimit.LimiterConfig
	
	// Whitelist는 Rate Limit에서 제외할 IP 목록입니다.
	Whitelist []string
	
	// KeyGenerator는 Rate Limit 키 생성 함수입니다.
	KeyGenerator func(*gin.Context) string
	
	// SkipSuccessfulRequests는 성공적인 요청을 Rate Limit에서 제외할지 여부입니다.
	SkipSuccessfulRequests bool
	
	// SkipFailedRequests는 실패한 요청을 Rate Limit에서 제외할지 여부입니다.
	SkipFailedRequests bool
}

// RateLimitMiddleware는 Rate Limit 미들웨어 구조체입니다.
type RateLimitMiddleware struct {
	config      *RateLimitConfig
	defaultLimiter *ratelimit.MemoryLimiter
	authLimiter    *ratelimit.MemoryLimiter
	endpointLimiters map[string]*ratelimit.MemoryLimiter
}

// DefaultRateLimitConfig는 기본 Rate Limit 설정을 반환합니다.
func DefaultRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		Enabled: true,
		DefaultConfig: &ratelimit.LimiterConfig{
			Rate:   60,  // 60 requests per minute
			Burst:  10,  // burst of 10 requests
			Window: 60,  // 1 minute window
		},
		AuthenticatedConfig: &ratelimit.LimiterConfig{
			Rate:   300, // 300 requests per minute for authenticated users
			Burst:  50,  // burst of 50 requests
			Window: 60,  // 1 minute window
		},
		EndpointConfigs: map[string]*ratelimit.LimiterConfig{
			"/api/v1/auth/login": {
				Rate:   5,  // 5 login attempts per minute
				Burst:  2,  // burst of 2 attempts
				Window: 60, // 1 minute window
			},
		},
		Whitelist: []string{
			"127.0.0.1",
			"::1",
		},
		KeyGenerator: defaultKeyGenerator,
		SkipSuccessfulRequests: false,
		SkipFailedRequests: false,
	}
}

// NewRateLimitMiddleware는 새로운 Rate Limit 미들웨어를 생성합니다.
func NewRateLimitMiddleware(config *RateLimitConfig) *RateLimitMiddleware {
	if config == nil {
		config = DefaultRateLimitConfig()
	}
	
	rlm := &RateLimitMiddleware{
		config: config,
		defaultLimiter: ratelimit.NewMemoryLimiter(config.DefaultConfig),
		authLimiter: ratelimit.NewMemoryLimiter(config.AuthenticatedConfig),
		endpointLimiters: make(map[string]*ratelimit.MemoryLimiter),
	}
	
	// 엔드포인트별 limiter 생성
	for endpoint, endpointConfig := range config.EndpointConfigs {
		rlm.endpointLimiters[endpoint] = ratelimit.NewMemoryLimiter(endpointConfig)
	}
	
	return rlm
}

// RateLimit은 Rate Limit 미들웨어 함수를 반환합니다.
func RateLimit(config *RateLimitConfig) gin.HandlerFunc {
	middleware := NewRateLimitMiddleware(config)
	return middleware.Handler()
}

// Handler는 실제 미들웨어 핸들러를 반환합니다.
func (rlm *RateLimitMiddleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Rate Limiting이 비활성화된 경우
		if !rlm.config.Enabled {
			c.Next()
			return
		}
		
		// 화이트리스트 IP 확인
		if rlm.isWhitelisted(c) {
			c.Next()
			return
		}
		
		// Rate Limit 키 생성
		key := rlm.config.KeyGenerator(c)
		
		// 적절한 limiter 선택
		limiter := rlm.selectLimiter(c)
		
		// Rate Limit 체크
		if !limiter.Allow(key) {
			// Rate Limit 초과 시 헤더 설정
			rlm.setRateLimitHeaders(c, limiter, key)
			rlm.handleRateLimitExceeded(c)
			return
		}
		
		// 성공 시 헤더 설정
		rlm.setRateLimitHeaders(c, limiter, key)
		
		// 다음 핸들러 실행
		c.Next()
		
		// 응답 후 처리 (skip 조건 확인)
		rlm.handlePostResponse(c, limiter, key)
	}
}

// isWhitelisted는 IP가 화이트리스트에 있는지 확인합니다.
func (rlm *RateLimitMiddleware) isWhitelisted(c *gin.Context) bool {
	clientIP := c.ClientIP()
	
	for _, whiteIP := range rlm.config.Whitelist {
		if clientIP == whiteIP {
			return true
		}
		// CIDR 범위 체크도 추가할 수 있음
	}
	
	return false
}

// selectLimiter는 요청에 적합한 limiter를 선택합니다.
func (rlm *RateLimitMiddleware) selectLimiter(c *gin.Context) ratelimit.RateLimiter {
	// 엔드포인트별 설정 확인
	path := c.Request.URL.Path
	if endpointLimiter, exists := rlm.endpointLimiters[path]; exists {
		return endpointLimiter
	}
	
	// 인증된 사용자 확인
	if rlm.isAuthenticated(c) {
		return rlm.authLimiter
	}
	
	// 기본 limiter
	return rlm.defaultLimiter
}

// isAuthenticated는 사용자가 인증되었는지 확인합니다.
func (rlm *RateLimitMiddleware) isAuthenticated(c *gin.Context) bool {
	// JWT 토큰 확인
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return false
	}
	
	// Bearer 토큰 형식 확인
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return false
	}
	
	// 간단히 토큰 존재 여부만 확인 (실제로는 JWT 검증 필요)
	return len(authHeader) > 7
}

// setRateLimitHeaders는 Rate Limit 관련 헤더를 설정합니다.
func (rlm *RateLimitMiddleware) setRateLimitHeaders(c *gin.Context, limiter ratelimit.RateLimiter, key string) {
	// X-RateLimit-Limit: 제한 수
	c.Header("X-RateLimit-Limit", strconv.Itoa(limiter.Limit(key)))
	
	// X-RateLimit-Remaining: 남은 요청 수
	c.Header("X-RateLimit-Remaining", strconv.Itoa(limiter.Remaining(key)))
	
	// X-RateLimit-Reset: 리셋 시간 (Unix timestamp)
	resetTime := limiter.ResetTime(key)
	c.Header("X-RateLimit-Reset", strconv.FormatInt(resetTime.Unix(), 10))
}

// handleRateLimitExceeded는 Rate Limit 초과 시 응답을 처리합니다.
func (rlm *RateLimitMiddleware) handleRateLimitExceeded(c *gin.Context) {
	// Retry-After 헤더 설정
	limiter := rlm.selectLimiter(c)
	key := rlm.config.KeyGenerator(c)
	resetTime := limiter.ResetTime(key)
	retryAfter := int(time.Until(resetTime).Seconds())
	if retryAfter < 1 {
		retryAfter = 1
	}
	
	c.Header("Retry-After", strconv.Itoa(retryAfter))
	
	// 429 응답
	c.JSON(http.StatusTooManyRequests, gin.H{
		"error": "Too Many Requests",
		"message": "Rate limit exceeded. Please try again later.",
		"code": "RATE_LIMIT_EXCEEDED",
		"retry_after": retryAfter,
	})
	
	c.Abort()
}

// handlePostResponse는 응답 후 처리를 수행합니다.
func (rlm *RateLimitMiddleware) handlePostResponse(c *gin.Context, limiter ratelimit.RateLimiter, key string) {
	// Skip 조건 확인
	statusCode := c.Writer.Status()
	
	if rlm.config.SkipSuccessfulRequests && statusCode >= 200 && statusCode < 400 {
		// 성공 응답 시 토큰 복원 (구현 필요)
		return
	}
	
	if rlm.config.SkipFailedRequests && statusCode >= 400 {
		// 실패 응답 시 토큰 복원 (구현 필요)
		return
	}
}

// defaultKeyGenerator는 기본 키 생성 함수입니다.
func defaultKeyGenerator(c *gin.Context) string {
	// 인증된 사용자의 경우 사용자 ID 사용
	if userID, exists := c.Get("user_id"); exists {
		if uid, ok := userID.(string); ok {
			return "user:" + uid
		}
	}
	
	// 미인증 사용자의 경우 IP 주소 사용
	return "ip:" + c.ClientIP()
}

// GetStats는 Rate Limit 통계를 반환합니다.
func (rlm *RateLimitMiddleware) GetStats() map[string]interface{} {
	stats := make(map[string]interface{})
	
	// 기본 limiter 통계
	if memLimiter, ok := rlm.defaultLimiter.(*ratelimit.MemoryLimiter); ok {
		stats["default"] = memLimiter.GetStats()
	}
	
	// 인증된 사용자 limiter 통계
	if memLimiter, ok := rlm.authLimiter.(*ratelimit.MemoryLimiter); ok {
		stats["authenticated"] = memLimiter.GetStats()
	}
	
	// 엔드포인트별 limiter 통계
	endpointStats := make(map[string]interface{})
	for endpoint, limiter := range rlm.endpointLimiters {
		if memLimiter, ok := limiter.(*ratelimit.MemoryLimiter); ok {
			endpointStats[endpoint] = memLimiter.GetStats()
		}
	}
	stats["endpoints"] = endpointStats
	
	return stats
}

// Close는 미들웨어 리소스를 정리합니다.
func (rlm *RateLimitMiddleware) Close() error {
	rlm.defaultLimiter.Close()
	rlm.authLimiter.Close()
	
	for _, limiter := range rlm.endpointLimiters {
		limiter.Close()
	}
	
	return nil
}