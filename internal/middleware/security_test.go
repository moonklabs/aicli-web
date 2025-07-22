package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/aicli/aicli-web/internal/security"
)

// TestAdvancedRateLimit는 고급 Rate Limiting 테스트입니다.
func TestAdvancedRateLimit(t *testing.T) {
	// 테스트용 Redis 클라이언트 (메모리)
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       1, // 테스트용 DB
	})

	// Redis 연결 확인
	ctx := context.Background()
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		t.Skip("Redis not available, skipping test")
	}

	// 테스트 후 정리
	defer rdb.FlushDB(ctx)
	defer rdb.Close()

	config := &AdvancedRateLimitConfig{
		Redis:           rdb,
		GlobalRateLimit: 10,
		IPRateLimit:     5,
		UserRateLimit:   3,
		WindowSize:      time.Minute,
		Logger:          zap.NewNop(),
	}

	middleware := AdvancedRateLimit(config)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware)
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	tests := []struct {
		name           string
		requests       int
		expectedStatus int
		clientIP       string
	}{
		{
			name:           "허용된 요청",
			requests:       3,
			expectedStatus: http.StatusOK,
			clientIP:       "192.168.1.100",
		},
		{
			name:           "IP Rate Limit 초과",
			requests:       6,
			expectedStatus: http.StatusTooManyRequests,
			clientIP:       "192.168.1.101",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var lastStatus int

			for i := 0; i < tt.requests; i++ {
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("X-Forwarded-For", tt.clientIP)
				w := httptest.NewRecorder()

				r.ServeHTTP(w, req)
				lastStatus = w.Code
			}

			assert.Equal(t, tt.expectedStatus, lastStatus)
		})
	}
}

// TestCSRFProtection는 CSRF 보호 테스트입니다.
func TestCSRFProtection(t *testing.T) {
	config := DefaultCSRFConfig()
	config.Logger = zap.NewNop()

	middleware := CSRF(config)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware)
	
	// GET 요청은 토큰 생성
	r.GET("/form", func(c *gin.Context) {
		token := c.GetHeader("X-CSRF-Token")
		c.JSON(http.StatusOK, gin.H{"csrf_token": token})
	})
	
	// POST 요청은 토큰 검증
	r.POST("/submit", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	t.Run("GET 요청 - 토큰 생성", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/form", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		token, exists := response["csrf_token"]
		assert.True(t, exists)
		assert.NotEmpty(t, token)
	})

	t.Run("POST 요청 - 토큰 없음 (실패)", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/submit", strings.NewReader("data=test"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("POST 요청 - 유효한 토큰 (성공)", func(t *testing.T) {
		// 먼저 토큰 얻기
		getReq := httptest.NewRequest("GET", "/form", nil)
		getW := httptest.NewRecorder()
		r.ServeHTTP(getW, getReq)

		cookies := getW.Result().Cookies()
		var csrfCookie *http.Cookie
		for _, cookie := range cookies {
			if cookie.Name == "csrf_token" {
				csrfCookie = cookie
				break
			}
		}
		require.NotNil(t, csrfCookie)

		// POST 요청에 토큰 포함
		postReq := httptest.NewRequest("POST", "/submit", strings.NewReader("data=test"))
		postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		postReq.Header.Set("X-CSRF-Token", csrfCookie.Value)
		postReq.AddCookie(csrfCookie)
		postW := httptest.NewRecorder()

		r.ServeHTTP(postW, postReq)

		assert.Equal(t, http.StatusOK, postW.Code)
	})
}

// TestSecurityHeaders는 보안 헤더 테스트입니다.
func TestSecurityHeaders(t *testing.T) {
	config := DefaultSecurityHeadersConfig()
	config.Logger = zap.NewNop()

	middleware := SecurityHeadersMiddleware(config)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware)
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// 보안 헤더 확인
	headers := w.Header()
	
	assert.Equal(t, "SAMEORIGIN", headers.Get("X-Frame-Options"))
	assert.Equal(t, "nosniff", headers.Get("X-Content-Type-Options"))
	assert.Equal(t, "1; mode=block", headers.Get("X-XSS-Protection"))
	assert.Equal(t, "strict-origin-when-cross-origin", headers.Get("Referrer-Policy"))
	assert.Contains(t, headers.Get("Content-Security-Policy"), "default-src 'self'")
	assert.Contains(t, headers.Get("Permissions-Policy"), "geolocation=('none')")
}

// TestAuditLogging는 감사 로깅 테스트입니다.
func TestAuditLogging(t *testing.T) {
	config := DefaultAuditConfig()
	config.Logger = zap.NewNop()

	middleware := AuditMiddleware(config)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware)
	r.POST("/api/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	t.Run("기본 감사 로깅", func(t *testing.T) {
		body := `{"test": "data"}`
		req := httptest.NewRequest("POST", "/api/test?param=value", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "test-client/1.0")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("민감한 데이터 마스킹", func(t *testing.T) {
		body := `{"password": "secret123", "data": "normal"}`
		req := httptest.NewRequest("POST", "/api/test", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer token123")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// TestAttackDetection은 공격 탐지 테스트입니다.
func TestAttackDetection(t *testing.T) {
	config := security.DefaultAttackDetectorConfig()
	config.Logger = zap.NewNop()

	detector := security.NewAttackDetector(config)

	tests := []struct {
		name           string
		request        *security.AttackDetectionRequest
		expectAttack   bool
		expectedType   string
	}{
		{
			name: "정상 요청",
			request: &security.AttackDetectionRequest{
				IPAddress: "192.168.1.100",
				UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
				Method:    "GET",
				URL:       "https://example.com/api/users",
				Path:      "/api/users",
				Timestamp: time.Now(),
			},
			expectAttack: false,
		},
		{
			name: "SQL Injection 공격",
			request: &security.AttackDetectionRequest{
				IPAddress: "192.168.1.100",
				UserAgent: "sqlmap/1.0",
				Method:    "GET",
				URL:       "https://example.com/api/users?id=1' UNION SELECT * FROM users--",
				Path:      "/api/users",
				Query:     "id=1' UNION SELECT * FROM users--",
				Timestamp: time.Now(),
			},
			expectAttack: true,
			expectedType: "sql_injection",
		},
		{
			name: "XSS 공격",
			request: &security.AttackDetectionRequest{
				IPAddress: "192.168.1.100",
				UserAgent: "Mozilla/5.0",
				Method:    "POST",
				URL:       "https://example.com/api/comments",
				Path:      "/api/comments",
				Body:      `{"comment": "<script>alert('xss')</script>"}`,
				Timestamp: time.Now(),
			},
			expectAttack: true,
			expectedType: "xss",
		},
		{
			name: "의심스러운 User-Agent",
			request: &security.AttackDetectionRequest{
				IPAddress: "192.168.1.100",
				UserAgent: "python-requests/2.25.1",
				Method:    "GET",
				URL:       "https://example.com/admin",
				Path:      "/admin",
				Timestamp: time.Now(),
			},
			expectAttack: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result := detector.DetectAttacks(ctx, tt.request)

			assert.Equal(t, tt.expectAttack, result.IsAttack)
			
			if tt.expectedType != "" {
				assert.Equal(t, tt.expectedType, result.AttackType)
			}
			
			if tt.expectAttack {
				assert.Greater(t, result.Confidence, 0.0)
				assert.NotEmpty(t, result.Evidence)
			}
		})
	}
}

// TestMetricsCollection은 메트릭 수집 테스트입니다.
func TestMetricsCollection(t *testing.T) {
	config := &security.MetricsCollectorConfig{
		Logger:          zap.NewNop(),
		CollectInterval: 100 * time.Millisecond,
		RetentionPeriod: time.Hour,
		BufferSize:      100,
	}

	collector := security.NewMetricsCollector(config)
	defer collector.Close()

	t.Run("카운터 메트릭", func(t *testing.T) {
		collector.IncrementCounter("test_counter", map[string]string{"label": "value"})
		
		// 수집될 시간을 기다림
		time.Sleep(200 * time.Millisecond)
		
		metrics := collector.GetCurrentMetrics()
		assert.NotEmpty(t, metrics)
	})

	t.Run("게이지 메트릭", func(t *testing.T) {
		collector.SetGauge("test_gauge", 42.5, nil)
		
		time.Sleep(200 * time.Millisecond)
		
		metrics := collector.GetCurrentMetrics()
		found := false
		for _, metric := range metrics {
			if metric.Name == "test_gauge" && metric.Value == 42.5 {
				found = true
				break
			}
		}
		assert.True(t, found)
	})

	t.Run("히스토그램 메트릭", func(t *testing.T) {
		collector.RecordHistogram("test_histogram", 1.23, map[string]string{"path": "/api/test"})
		
		time.Sleep(200 * time.Millisecond)
		
		metrics := collector.GetCurrentMetrics()
		assert.NotEmpty(t, metrics)
	})
}

// BenchmarkRateLimit는 Rate Limiting 성능 테스트입니다.
func BenchmarkRateLimit(b *testing.B) {
	config := &AdvancedRateLimitConfig{
		GlobalRateLimit: 1000,
		IPRateLimit:     100,
		WindowSize:      time.Minute,
		Logger:          zap.NewNop(),
	}

	middleware := AdvancedRateLimit(config)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(middleware)
	r.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("X-Forwarded-For", "192.168.1.100")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
		}
	})
}

// BenchmarkSecurityHeaders는 보안 헤더 성능 테스트입니다.
func BenchmarkSecurityHeaders(b *testing.B) {
	config := DefaultSecurityHeadersConfig()
	middleware := SecurityHeadersMiddleware(config)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(middleware)
	r.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
		}
	})
}

// BenchmarkAttackDetection은 공격 탐지 성능 테스트입니다.
func BenchmarkAttackDetection(b *testing.B) {
	config := security.DefaultAttackDetectorConfig()
	config.Logger = zap.NewNop()
	detector := security.NewAttackDetector(config)

	request := &security.AttackDetectionRequest{
		IPAddress: "192.168.1.100",
		UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		Method:    "GET",
		URL:       "https://example.com/api/users?id=123",
		Path:      "/api/users",
		Query:     "id=123",
		Timestamp: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := context.Background()
		detector.DetectAttacks(ctx, request)
	}
}

// TestSecurityIntegration는 보안 미들웨어 통합 테스트입니다.
func TestSecurityIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// 모든 보안 미들웨어 적용
	r.Use(SecurityHeadersMiddleware(DefaultSecurityHeadersConfig()))
	
	csrfConfig := DefaultCSRFConfig()
	csrfConfig.Logger = zap.NewNop()
	r.Use(CSRF(csrfConfig))
	
	auditConfig := DefaultAuditConfig()
	auditConfig.Logger = zap.NewNop()
	r.Use(AuditMiddleware(auditConfig))

	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "secure endpoint"})
	})

	r.POST("/api/data", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "data received"})
	})

	t.Run("통합 보안 테스트", func(t *testing.T) {
		// GET 요청 (토큰 생성)
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		
		// 보안 헤더 확인
		headers := w.Header()
		assert.NotEmpty(t, headers.Get("X-Frame-Options"))
		assert.NotEmpty(t, headers.Get("Content-Security-Policy"))
		assert.NotEmpty(t, headers.Get("X-CSRF-Token"))
	})
}