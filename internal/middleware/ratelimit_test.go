package middleware

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/aicli/aicli-web/internal/ratelimit"
)

func TestRateLimit_Basic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	config := &RateLimitConfig{
		Enabled: true,
		DefaultConfig: &ratelimit.LimiterConfig{
			Rate:   60,
			Burst:  3,
			Window: 60,
		},
		AuthenticatedConfig: &ratelimit.LimiterConfig{
			Rate:   300,
			Burst:  10,
			Window: 60,
		},
		EndpointConfigs: make(map[string]*ratelimit.LimiterConfig),
		Whitelist:       []string{},
		KeyGenerator:    defaultKeyGenerator,
	}
	
	router := gin.New()
	router.Use(RateLimit(config))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})
	
	// 처음 3개 요청은 성공해야 함
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code, "Request %d should succeed", i+1)
		
		// Rate Limit 헤더 확인
		assert.Equal(t, "60", w.Header().Get("X-RateLimit-Limit"))
		remaining, _ := strconv.Atoi(w.Header().Get("X-RateLimit-Remaining"))
		assert.Equal(t, 2-i, remaining)
		assert.NotEmpty(t, w.Header().Get("X-RateLimit-Reset"))
	}
	
	// 4번째 요청은 Rate Limit에 걸려야 함
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.NotEmpty(t, w.Header().Get("Retry-After"))
}

func TestRateLimit_Disabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	config := &RateLimitConfig{
		Enabled: false, // Rate Limiting 비활성화
		DefaultConfig: &ratelimit.LimiterConfig{
			Rate:   1,
			Burst:  1,
			Window: 60,
		},
	}
	
	router := gin.New()
	router.Use(RateLimit(config))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})
	
	// Rate Limiting이 비활성화되어 있으므로 모든 요청이 성공해야 함
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code, "Request %d should succeed when rate limiting is disabled", i+1)
	}
}

func TestRateLimit_Whitelist(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	config := &RateLimitConfig{
		Enabled: true,
		DefaultConfig: &ratelimit.LimiterConfig{
			Rate:   60,
			Burst:  1,
			Window: 60,
		},
		AuthenticatedConfig: &ratelimit.LimiterConfig{
			Rate:   300,
			Burst:  10,
			Window: 60,
		},
		EndpointConfigs: make(map[string]*ratelimit.LimiterConfig),
		Whitelist:       []string{"127.0.0.1"}, // localhost 화이트리스트
		KeyGenerator:    defaultKeyGenerator,
	}
	
	router := gin.New()
	router.Use(RateLimit(config))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})
	
	// 화이트리스트 IP에서는 모든 요청이 성공해야 함
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Forwarded-For", "127.0.0.1")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code, "Request %d from whitelisted IP should succeed", i+1)
	}
}

func TestRateLimit_EndpointSpecific(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	config := &RateLimitConfig{
		Enabled: true,
		DefaultConfig: &ratelimit.LimiterConfig{
			Rate:   60,
			Burst:  5,
			Window: 60,
		},
		AuthenticatedConfig: &ratelimit.LimiterConfig{
			Rate:   300,
			Burst:  10,
			Window: 60,
		},
		EndpointConfigs: map[string]*ratelimit.LimiterConfig{
			"/api/v1/auth/login": {
				Rate:   5,
				Burst:  2,
				Window: 60,
			},
		},
		Whitelist:    []string{},
		KeyGenerator: defaultKeyGenerator,
	}
	
	router := gin.New()
	router.Use(RateLimit(config))
	router.POST("/api/v1/auth/login", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "login"})
	})
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "test"})
	})
	
	// 로그인 엔드포인트는 2번만 허용되어야 함
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code, "Login request %d should succeed", i+1)
		assert.Equal(t, "5", w.Header().Get("X-RateLimit-Limit"))
	}
	
	// 3번째 로그인 시도는 실패해야 함
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	
	// 하지만 일반 엔드포인트는 여전히 허용되어야 함
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "60", w.Header().Get("X-RateLimit-Limit"))
}

func TestRateLimit_AuthenticatedUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	config := &RateLimitConfig{
		Enabled: true,
		DefaultConfig: &ratelimit.LimiterConfig{
			Rate:   60,
			Burst:  3,
			Window: 60,
		},
		AuthenticatedConfig: &ratelimit.LimiterConfig{
			Rate:   300,
			Burst:  10,
			Window: 60,
		},
		EndpointConfigs: make(map[string]*ratelimit.LimiterConfig),
		Whitelist:       []string{},
		KeyGenerator:    defaultKeyGenerator,
	}
	
	router := gin.New()
	router.Use(RateLimit(config))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})
	
	// 인증된 사용자는 더 높은 한도를 가져야 함
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Authorization", "Bearer valid-jwt-token")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code, "Authenticated request %d should succeed", i+1)
		assert.Equal(t, "300", w.Header().Get("X-RateLimit-Limit"))
	}
}

func TestRateLimit_Headers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	config := &RateLimitConfig{
		Enabled: true,
		DefaultConfig: &ratelimit.LimiterConfig{
			Rate:   60,
			Burst:  5,
			Window: 60,
		},
		AuthenticatedConfig: &ratelimit.LimiterConfig{
			Rate:   300,
			Burst:  10,
			Window: 60,
		},
		EndpointConfigs: make(map[string]*ratelimit.LimiterConfig),
		Whitelist:       []string{},
		KeyGenerator:    defaultKeyGenerator,
	}
	
	router := gin.New()
	router.Use(RateLimit(config))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})
	
	// 첫 번째 요청
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	// Rate Limit 헤더 확인
	assert.Equal(t, "60", w.Header().Get("X-RateLimit-Limit"))
	assert.Equal(t, "4", w.Header().Get("X-RateLimit-Remaining"))
	assert.NotEmpty(t, w.Header().Get("X-RateLimit-Reset"))
	
	// Reset 헤더가 Unix timestamp 형식인지 확인
	resetHeader := w.Header().Get("X-RateLimit-Reset")
	resetTime, err := strconv.ParseInt(resetHeader, 10, 64)
	require.NoError(t, err)
	
	// Reset 시간이 미래여야 함
	assert.True(t, resetTime > time.Now().Unix())
}

func TestRateLimit_TooManyRequestsResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	config := &RateLimitConfig{
		Enabled: true,
		DefaultConfig: &ratelimit.LimiterConfig{
			Rate:   60,
			Burst:  1,
			Window: 60,
		},
		AuthenticatedConfig: &ratelimit.LimiterConfig{
			Rate:   300,
			Burst:  10,
			Window: 60,
		},
		EndpointConfigs: make(map[string]*ratelimit.LimiterConfig),
		Whitelist:       []string{},
		KeyGenerator:    defaultKeyGenerator,
	}
	
	router := gin.New()
	router.Use(RateLimit(config))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})
	
	// 첫 번째 요청은 성공
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	
	// 두 번째 요청은 Rate Limit에 걸림
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.NotEmpty(t, w.Header().Get("Retry-After"))
	
	// 응답 본문 확인
	assert.Contains(t, w.Body.String(), "Rate limit exceeded")
	assert.Contains(t, w.Body.String(), "RATE_LIMIT_EXCEEDED")
}

func TestDefaultRateLimitConfig(t *testing.T) {
	config := DefaultRateLimitConfig()
	
	assert.NotNil(t, config)
	assert.True(t, config.Enabled)
	assert.Equal(t, 60, config.DefaultConfig.Rate)
	assert.Equal(t, 10, config.DefaultConfig.Burst)
	assert.Equal(t, 300, config.AuthenticatedConfig.Rate)
	assert.Equal(t, 50, config.AuthenticatedConfig.Burst)
	
	// 엔드포인트별 설정 확인
	loginConfig, exists := config.EndpointConfigs["/api/v1/auth/login"]
	assert.True(t, exists)
	assert.Equal(t, 5, loginConfig.Rate)
	assert.Equal(t, 2, loginConfig.Burst)
	
	// 화이트리스트 확인
	assert.Contains(t, config.Whitelist, "127.0.0.1")
	assert.Contains(t, config.Whitelist, "::1")
}

func TestDefaultKeyGenerator(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
	c.Request.Header.Set("X-Forwarded-For", "192.168.1.100")
	
	// 미인증 사용자 - IP 기반 키
	key := defaultKeyGenerator(c)
	assert.Equal(t, "ip:192.168.1.100", key)
	
	// 인증된 사용자 - 사용자 ID 기반 키
	c.Set("user_id", "user123")
	key = defaultKeyGenerator(c)
	assert.Equal(t, "user:user123", key)
}

func TestRateLimitMiddleware_Stats(t *testing.T) {
	config := DefaultRateLimitConfig()
	middleware := NewRateLimitMiddleware(config)
	defer middleware.Close()
	
	stats := middleware.GetStats()
	assert.NotNil(t, stats)
	
	// 기본 통계 구조 확인
	assert.Contains(t, stats, "default")
	assert.Contains(t, stats, "authenticated")
	assert.Contains(t, stats, "endpoints")
	
	endpointStats, ok := stats["endpoints"].(map[string]interface{})
	assert.True(t, ok)
	assert.Contains(t, endpointStats, "/api/v1/auth/login")
}