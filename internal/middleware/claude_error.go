package middleware

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/aicli/aicli-web/internal/claude"
)

// APIError는 표준화된 API 에러 응답 구조체입니다.
type APIError struct {
	Code      string                 `json:"code"`
	Message   string                 `json:"message"`
	Details   map[string]interface{} `json:"details,omitempty"`
	TraceID   string                 `json:"trace_id"`
	Timestamp time.Time              `json:"timestamp"`
}

// ClaudeErrorHandler는 Claude 관련 에러를 처리하는 미들웨어입니다.
func ClaudeErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// 에러가 있는지 확인
		if len(c.Errors) > 0 {
			err := c.Errors.Last()

			// 이미 응답이 작성된 경우 처리하지 않음
			if c.Writer.Written() {
				return
			}

			// Claude 에러 변환
			if claudeErr, ok := err.Err.(*claude.ClaudeError); ok {
				handleClaudeError(c, claudeErr)
				return
			}

			// 일반 에러 처리
			handleGenericError(c, err.Err)
		}
	}
}

// handleClaudeError는 Claude 에러를 처리합니다.
func handleClaudeError(c *gin.Context, claudeErr *claude.ClaudeError) {
	httpStatus := mapClaudeErrorToHTTPStatus(claudeErr.Code)

	apiErr := &APIError{
		Code:      claudeErr.Code,
		Message:   claudeErr.Message,
		Details:   claudeErr.Details,
		TraceID:   getTraceID(c),
		Timestamp: time.Now(),
	}

	// 에러 로깅
	log.Printf("Claude error: %s - %s (trace: %s)", claudeErr.Code, claudeErr.Message, apiErr.TraceID)

	c.JSON(httpStatus, apiErr)
}

// handleGenericError는 일반 에러를 처리합니다.
func handleGenericError(c *gin.Context, err error) {
	apiErr := &APIError{
		Code:      "INTERNAL_ERROR",
		Message:   "An internal error occurred",
		TraceID:   getTraceID(c),
		Timestamp: time.Now(),
	}

	// 개발 환경에서는 상세 에러 정보 제공
	if gin.Mode() == gin.DebugMode {
		apiErr.Details = map[string]interface{}{
			"internal_error": err.Error(),
		}
	}

	// 에러 로깅
	log.Printf("Internal error: %v (trace: %s)", err, apiErr.TraceID)

	c.JSON(http.StatusInternalServerError, apiErr)
}

// mapClaudeErrorToHTTPStatus는 Claude 에러 코드를 HTTP 상태 코드로 매핑합니다.
func mapClaudeErrorToHTTPStatus(errorCode string) int {
	switch errorCode {
	case "INVALID_REQUEST":
		return http.StatusBadRequest
	case "AUTHENTICATION_FAILED":
		return http.StatusUnauthorized
	case "PERMISSION_DENIED":
		return http.StatusForbidden
	case "SESSION_NOT_FOUND":
		return http.StatusNotFound
	case "PROCESS_NOT_FOUND":
		return http.StatusNotFound
	case "RESOURCE_EXHAUSTED":
		return http.StatusTooManyRequests
	case "DEADLINE_EXCEEDED":
		return http.StatusRequestTimeout
	case "PROCESS_FAILED":
		return http.StatusInternalServerError
	case "SESSION_EXPIRED":
		return http.StatusGone
	case "RATE_LIMIT_EXCEEDED":
		return http.StatusTooManyRequests
	case "WORKSPACE_NOT_FOUND":
		return http.StatusNotFound
	case "INVALID_CONFIGURATION":
		return http.StatusBadRequest
	case "STORAGE_ERROR":
		return http.StatusInternalServerError
	case "NETWORK_ERROR":
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

// getTraceID는 요청의 추적 ID를 가져옵니다.
func getTraceID(c *gin.Context) string {
	if traceID, exists := c.Get("trace_id"); exists {
		if id, ok := traceID.(string); ok {
			return id
		}
	}

	// trace_id가 없는 경우 request_id 사용
	if requestID, exists := c.Get("request_id"); exists {
		if id, ok := requestID.(string); ok {
			return id
		}
	}

	return "unknown"
}

// ClaudeErrorResponder는 Claude 에러 응답을 위한 헬퍼 함수들을 제공합니다.
type ClaudeErrorResponder struct{}

// NewClaudeErrorResponder는 새로운 에러 응답기를 생성합니다.
func NewClaudeErrorResponder() *ClaudeErrorResponder {
	return &ClaudeErrorResponder{}
}

// BadRequest는 400 Bad Request 에러를 응답합니다.
func (r *ClaudeErrorResponder) BadRequest(c *gin.Context, message string, details map[string]interface{}) {
	apiErr := &APIError{
		Code:      "BAD_REQUEST",
		Message:   message,
		Details:   details,
		TraceID:   getTraceID(c),
		Timestamp: time.Now(),
	}
	c.JSON(http.StatusBadRequest, apiErr)
}

// NotFound는 404 Not Found 에러를 응답합니다.
func (r *ClaudeErrorResponder) NotFound(c *gin.Context, resource string) {
	apiErr := &APIError{
		Code:      "NOT_FOUND",
		Message:   fmt.Sprintf("%s not found", resource),
		TraceID:   getTraceID(c),
		Timestamp: time.Now(),
	}
	c.JSON(http.StatusNotFound, apiErr)
}

// InternalError는 500 Internal Server Error를 응답합니다.
func (r *ClaudeErrorResponder) InternalError(c *gin.Context, message string) {
	apiErr := &APIError{
		Code:      "INTERNAL_ERROR",
		Message:   message,
		TraceID:   getTraceID(c),
		Timestamp: time.Now(),
	}
	c.JSON(http.StatusInternalServerError, apiErr)
}

// ValidationError는 검증 에러를 응답합니다.
func (r *ClaudeErrorResponder) ValidationError(c *gin.Context, errors map[string]string) {
	apiErr := &APIError{
		Code:    "VALIDATION_ERROR",
		Message: "Request validation failed",
		Details: map[string]interface{}{
			"validation_errors": errors,
		},
		TraceID:   getTraceID(c),
		Timestamp: time.Now(),
	}
	c.JSON(http.StatusBadRequest, apiErr)
}

// RateLimitExceeded는 429 Too Many Requests 에러를 응답합니다.
func (r *ClaudeErrorResponder) RateLimitExceeded(c *gin.Context, retryAfter int) {
	apiErr := &APIError{
		Code:    "RATE_LIMIT_EXCEEDED",
		Message: "Rate limit exceeded",
		Details: map[string]interface{}{
			"retry_after_seconds": retryAfter,
		},
		TraceID:   getTraceID(c),
		Timestamp: time.Now(),
	}
	
	// Retry-After 헤더 설정
	c.Header("Retry-After", fmt.Sprintf("%d", retryAfter))
	c.JSON(http.StatusTooManyRequests, apiErr)
}