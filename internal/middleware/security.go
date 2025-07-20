package middleware

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

// Security는 보안 헤더를 설정하는 미들웨어입니다.
// OWASP 권장사항을 따라 기본적인 보안 헤더들을 설정합니다.
func Security() gin.HandlerFunc {
	return func(c *gin.Context) {
		// X-Content-Type-Options: MIME 타입 스니핑 방지
		c.Header("X-Content-Type-Options", "nosniff")
		
		// X-Frame-Options: 클릭재킹 공격 방지
		c.Header("X-Frame-Options", "DENY")
		
		// X-XSS-Protection: XSS 공격 방지 (구형 브라우저용)
		c.Header("X-XSS-Protection", "1; mode=block")
		
		// Referrer-Policy: 리퍼러 정보 제어
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		
		// Content-Security-Policy: XSS 및 데이터 인젝션 공격 방지
		csp := getContentSecurityPolicy()
		if csp != "" {
			c.Header("Content-Security-Policy", csp)
		}
		
		// Strict-Transport-Security: HTTPS 강제 (HTTPS 환경에서만)
		if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		
		// Permissions-Policy: 브라우저 기능 제어
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		
		// Server 헤더 제거 (정보 노출 방지)
		c.Header("Server", "")

		c.Next()
	}
}

// getContentSecurityPolicy는 환경에 따른 CSP 정책을 반환합니다.
func getContentSecurityPolicy() string {
	env := viper.GetString("env")
	
	switch env {
	case "development":
		// 개발 환경: 느슨한 정책
		return "default-src 'self' 'unsafe-inline' 'unsafe-eval'; connect-src 'self' ws: wss:; img-src 'self' data: blob:;"
	case "production":
		// 프로덕션 환경: 엄격한 정책
		return "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'; font-src 'self'; object-src 'none'; media-src 'self'; frame-src 'none';"
	default:
		// 기본 정책
		return "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:;"
	}
}

// SecurityWithConfig는 사용자 정의 보안 설정을 적용하는 미들웨어입니다.
func SecurityWithConfig(config SecurityConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		if config.ContentTypeNosniff {
			c.Header("X-Content-Type-Options", "nosniff")
		}
		
		if config.FrameOptions != "" {
			c.Header("X-Frame-Options", config.FrameOptions)
		}
		
		if config.XSSProtection != "" {
			c.Header("X-XSS-Protection", config.XSSProtection)
		}
		
		if config.ContentSecurityPolicy != "" {
			c.Header("Content-Security-Policy", config.ContentSecurityPolicy)
		}
		
		if config.ReferrerPolicy != "" {
			c.Header("Referrer-Policy", config.ReferrerPolicy)
		}
		
		if config.HSTSMaxAge > 0 && (c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https") {
			hstsHeader := fmt.Sprintf("max-age=%d", config.HSTSMaxAge)
			if config.HSTSIncludeSubdomains {
				hstsHeader += "; includeSubDomains"
			}
			if config.HSTSPreload {
				hstsHeader += "; preload"
			}
			c.Header("Strict-Transport-Security", hstsHeader)
		}
		
		if config.PermissionsPolicy != "" {
			c.Header("Permissions-Policy", config.PermissionsPolicy)
		}
		
		if config.HideServerHeader {
			c.Header("Server", "")
		}

		c.Next()
	}
}

// SecurityConfig는 보안 미들웨어 설정 구조체입니다.
type SecurityConfig struct {
	ContentTypeNosniff      bool
	FrameOptions           string
	XSSProtection          string
	ContentSecurityPolicy  string
	ReferrerPolicy         string
	HSTSMaxAge            int
	HSTSIncludeSubdomains  bool
	HSTSPreload           bool
	PermissionsPolicy      string
	HideServerHeader       bool
}