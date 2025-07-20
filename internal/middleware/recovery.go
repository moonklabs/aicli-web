package middleware

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

// Recovery는 패닉을 복구하고 에러 응답을 반환하는 미들웨어입니다.
func Recovery() gin.HandlerFunc {
	return RecoveryWithWriter(gin.DefaultWriter)
}

// RecoveryWithWriter는 지정된 Writer로 패닉 로그를 출력하는 복구 미들웨어입니다.
func RecoveryWithWriter(writer io.Writer) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// 요청 ID 가져오기
				requestID := GetRequestID(c)
				
				// 스택 트레이스 가져오기
				stack := debug.Stack()
				
				// 에러 정보 로깅
				log.Printf(`{"level":"error","request_id":"%s","event":"panic_recovered","error":"%v","method":"%s","path":"%s","client_ip":"%s","user_agent":"%s","stack_trace":"%s"}`,
					requestID,
					err,
					c.Request.Method,
					c.Request.URL.Path,
					c.ClientIP(),
					c.Request.UserAgent(),
					string(stack),
				)

				// 에러 응답 반환
				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"error": gin.H{
						"code":       "ERR_INTERNAL",
						"message":    "서버 내부 오류가 발생했습니다",
						"request_id": requestID,
					},
				})
				
				c.Abort()
			}
		}()
		
		c.Next()
	}
}

// CustomRecovery는 사용자 정의 복구 핸들러를 사용하는 미들웨어입니다.
func CustomRecovery(handler RecoveryHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				handler(c, err)
			}
		}()
		
		c.Next()
	}
}

// RecoveryHandler는 패닉 복구 핸들러 함수 타입입니다.
type RecoveryHandler func(c *gin.Context, err interface{})

// DefaultRecoveryHandler는 기본 복구 핸들러입니다.
func DefaultRecoveryHandler(c *gin.Context, err interface{}) {
	requestID := GetRequestID(c)
	stack := debug.Stack()
	
	// 에러 유형에 따른 처리
	var errorMessage string
	var statusCode int
	
	switch e := err.(type) {
	case string:
		errorMessage = e
		statusCode = http.StatusInternalServerError
	case error:
		errorMessage = e.Error()
		statusCode = http.StatusInternalServerError
	default:
		errorMessage = fmt.Sprintf("알 수 없는 오류: %v", err)
		statusCode = http.StatusInternalServerError
	}
	
	// 에러 로깅
	log.Printf(`{"level":"error","request_id":"%s","event":"panic_recovered","error":"%s","stack":"%s"}`,
		requestID,
		errorMessage,
		string(stack),
	)
	
	// 클라이언트에게 안전한 에러 메시지 반환
	c.JSON(statusCode, gin.H{
		"success": false,
		"error": gin.H{
			"code":       "ERR_INTERNAL",
			"message":    "서버에서 예상치 못한 오류가 발생했습니다",
			"request_id": requestID,
		},
	})
	
	c.Abort()
}

// GracefulRecovery는 더 우아한 에러 처리를 위한 복구 미들웨어입니다.
func GracefulRecovery() gin.HandlerFunc {
	return CustomRecovery(func(c *gin.Context, err interface{}) {
		requestID := GetRequestID(c)
		
		// 상세한 에러 정보 수집
		errorInfo := map[string]interface{}{
			"panic_value": err,
			"request_id":  requestID,
			"method":      c.Request.Method,
			"path":        c.Request.URL.Path,
			"query":       c.Request.URL.RawQuery,
			"client_ip":   c.ClientIP(),
			"user_agent":  c.Request.UserAgent(),
			"headers":     c.Request.Header,
		}
		
		// 스택 트레이스
		stack := debug.Stack()
		
		// 구조화된 에러 로깅
		log.Printf(`{"level":"error","event":"panic_recovered","request_id":"%s","error_info":%v,"stack_trace":"%s"}`,
			requestID,
			errorInfo,
			string(stack),
		)
		
		// 개발 환경에서는 더 자세한 정보 제공
		response := gin.H{
			"success": false,
			"error": gin.H{
				"code":       "ERR_INTERNAL",
				"message":    "서버 내부 오류가 발생했습니다",
				"request_id": requestID,
			},
		}
		
		// 개발 환경에서 디버그 정보 추가
		if gin.Mode() == gin.DebugMode {
			response["debug"] = gin.H{
				"panic_value": fmt.Sprintf("%v", err),
				"stack_trace": string(stack),
			}
		}
		
		c.JSON(http.StatusInternalServerError, response)
		c.Abort()
	})
}