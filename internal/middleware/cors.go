package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

// CORS는 Cross-Origin Resource Sharing 헤더를 설정하는 미들웨어입니다.
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		
		// 환경별 CORS 정책 적용
		env := viper.GetString("env")
		
		switch env {
		case "development":
			// 개발 환경: 모든 오리진 허용
			c.Header("Access-Control-Allow-Origin", "*")
		case "production":
			// 프로덕션 환경: 허용된 도메인만
			allowedOrigins := viper.GetStringSlice("cors.allowed_origins")
			if len(allowedOrigins) == 0 {
				// 기본값 설정
				allowedOrigins = []string{"https://aicli.example.com"}
			}
			
			for _, allowedOrigin := range allowedOrigins {
				if origin == allowedOrigin {
					c.Header("Access-Control-Allow-Origin", origin)
					break
				}
			}
		default:
			// 기본값: localhost만 허용
			if origin == "http://localhost:3000" || origin == "http://localhost:8080" {
				c.Header("Access-Control-Allow-Origin", origin)
			}
		}

		// 허용된 메소드
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		
		// 허용된 헤더
		c.Header("Access-Control-Allow-Headers", 
			"Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-Request-ID, X-API-Key")
		
		// 자격 증명 포함 허용
		c.Header("Access-Control-Allow-Credentials", "true")
		
		// 노출할 헤더
		c.Header("Access-Control-Expose-Headers", "X-Request-ID, X-Total-Count")
		
		// Preflight 요청 처리
		if c.Request.Method == "OPTIONS" {
			c.Header("Access-Control-Max-Age", "86400") // 24시간 캐시
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// CORSWithConfig는 사용자 정의 설정으로 CORS 미들웨어를 생성합니다.
func CORSWithConfig(config CORSConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		
		// 허용된 오리진 체크
		allowed := false
		for _, allowedOrigin := range config.AllowedOrigins {
			if allowedOrigin == "*" || allowedOrigin == origin {
				c.Header("Access-Control-Allow-Origin", allowedOrigin)
				allowed = true
				break
			}
		}
		
		if !allowed && len(config.AllowedOrigins) > 0 {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		// 기타 헤더 설정
		if len(config.AllowedMethods) > 0 {
			c.Header("Access-Control-Allow-Methods", joinStrings(config.AllowedMethods, ", "))
		}
		
		if len(config.AllowedHeaders) > 0 {
			c.Header("Access-Control-Allow-Headers", joinStrings(config.AllowedHeaders, ", "))
		}
		
		if config.AllowCredentials {
			c.Header("Access-Control-Allow-Credentials", "true")
		}
		
		if config.MaxAge > 0 {
			c.Header("Access-Control-Max-Age", string(rune(config.MaxAge)))
		}

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// CORSConfig는 CORS 설정 구조체입니다.
type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAge           int
}

// joinStrings는 문자열 슬라이스를 구분자로 연결합니다.
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	if len(strs) == 1 {
		return strs[0]
	}
	
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}