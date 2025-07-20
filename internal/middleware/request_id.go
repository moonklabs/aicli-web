package middleware

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/gin-gonic/gin"
)

// RequestID는 각 요청에 고유한 ID를 부여하는 미들웨어입니다.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 요청 헤더에서 기존 ID 확인
		requestID := c.GetHeader("X-Request-ID")
		
		// 없으면 새로 생성
		if requestID == "" {
			requestID = generateRequestID()
		}
		
		// 컨텍스트에 저장
		c.Set("request_id", requestID)
		
		// 응답 헤더에 추가
		c.Header("X-Request-ID", requestID)
		
		c.Next()
	}
}

// GetRequestID는 컨텍스트에서 요청 ID를 가져옵니다.
func GetRequestID(c *gin.Context) string {
	requestID, exists := c.Get("request_id")
	if !exists {
		return "unknown"
	}
	
	if id, ok := requestID.(string); ok {
		return id
	}
	
	return "unknown"
}

// generateRequestID는 랜덤한 요청 ID를 생성합니다.
func generateRequestID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		// 에러 발생 시 현재 시간 기반으로 생성
		return "fallback-id"
	}
	return hex.EncodeToString(bytes)
}