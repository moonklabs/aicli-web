package middleware

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/gin-gonic/gin"
)

const (
	// RequestIDHeader는 요청 ID 헤더 이름입니다.
	RequestIDHeader = "X-Request-ID"
	// RequestIDKey는 컨텍스트에서 요청 ID를 저장하는 키입니다.
	RequestIDKey = "request_id"
)

// RequestID는 각 요청에 고유한 ID를 부여하는 미들웨어입니다.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 기존 요청 ID가 있는지 확인
		requestID := c.GetHeader(RequestIDHeader)
		
		// 없으면 새로 생성
		if requestID == "" {
			requestID = generateRequestID()
		}

		// 컨텍스트에 저장
		c.Set(RequestIDKey, requestID)
		
		// 응답 헤더에 추가
		c.Header(RequestIDHeader, requestID)

		c.Next()
	}
}

// generateRequestID는 랜덤한 요청 ID를 생성합니다.
func generateRequestID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		// 랜덤 생성 실패 시 fallback
		return "req-unknown"
	}
	return "req-" + hex.EncodeToString(bytes)
}

// GetRequestID는 컨텍스트에서 요청 ID를 가져옵니다.
func GetRequestID(c *gin.Context) string {
	if requestID, exists := c.Get(RequestIDKey); exists {
		if id, ok := requestID.(string); ok {
			return id
		}
	}
	return ""
}