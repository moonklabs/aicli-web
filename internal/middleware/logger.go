package middleware

import (
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// Logger는 요청/응답 로깅 미들웨어를 반환합니다.
func Logger() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		// 구조화된 로그 포맷
		log.Printf(`{"time":"%s","method":"%s","path":"%s","protocol":"%s","status":%d,"latency":"%s","client_ip":"%s","user_agent":"%s","error":"%s"}`,
			param.TimeStamp.Format(time.RFC3339),
			param.Method,
			param.Path,
			param.Request.Proto,
			param.StatusCode,
			param.Latency.String(),
			param.ClientIP,
			param.Request.UserAgent(),
			param.ErrorMessage,
		)
		return ""
	})
}

// RequestLogger는 더 상세한 요청 로깅을 위한 미들웨어입니다.
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 요청 시작 시간 기록
		start := time.Now()
		
		// 요청 ID 가져오기 (request_id 미들웨어가 먼저 실행되어야 함)
		requestID := c.GetString("request_id")
		if requestID == "" {
			requestID = "unknown"
		}

		// 요청 정보 로깅
		log.Printf(`{"level":"info","request_id":"%s","event":"request_start","method":"%s","path":"%s","query":"%s","client_ip":"%s","user_agent":"%s"}`,
			requestID,
			c.Request.Method,
			c.Request.URL.Path,
			c.Request.URL.RawQuery,
			c.ClientIP(),
			c.Request.UserAgent(),
		)

		// 다음 핸들러 실행
		c.Next()

		// 응답 정보 로깅
		duration := time.Since(start)
		log.Printf(`{"level":"info","request_id":"%s","event":"request_end","method":"%s","path":"%s","status":%d,"duration":"%s","response_size":%d}`,
			requestID,
			c.Request.Method,
			c.Request.URL.Path,
			c.Writer.Status(),
			duration.String(),
			c.Writer.Size(),
		)

		// 에러가 있으면 별도 로깅
		if len(c.Errors) > 0 {
			for _, err := range c.Errors {
				log.Printf(`{"level":"error","request_id":"%s","event":"request_error","error":"%s","type":"%s"}`,
					requestID,
					err.Error(),
					fmt.Sprint(err.Type),
				)
			}
		}
	}
}