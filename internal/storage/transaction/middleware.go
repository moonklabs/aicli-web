package transaction

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/aicli/aicli-web/internal/storage"
)

// MiddlewareConfig 미들웨어 설정
type MiddlewareConfig struct {
	// Manager 트랜잭션 매니저
	Manager Manager
	
	// Logger 로거
	Logger *log.Logger
	
	// TransactionMethods 트랜잭션을 사용할 HTTP 메서드들
	TransactionMethods []string
	
	// SkipPaths 트랜잭션을 스킵할 경로들
	SkipPaths []string
	
	// DefaultTimeout 기본 타임아웃
	DefaultTimeout time.Duration
	
	// MaxTimeout 최대 타임아웃
	MaxTimeout time.Duration
	
	// EnableLogging 로깅 활성화 여부
	EnableLogging bool
	
	// EnableMetrics 메트릭 수집 활성화 여부
	EnableMetrics bool
	
	// CustomErrorHandler 커스텀 에러 핸들러
	CustomErrorHandler func(http.ResponseWriter, *http.Request, error)
}

// DefaultMiddlewareConfig 기본 미들웨어 설정
func DefaultMiddlewareConfig(manager Manager) *MiddlewareConfig {
	return &MiddlewareConfig{
		Manager:            manager,
		TransactionMethods: []string{"POST", "PUT", "PATCH", "DELETE"},
		SkipPaths:          []string{"/health", "/metrics", "/favicon.ico"},
		DefaultTimeout:     30 * time.Second,
		MaxTimeout:         5 * time.Minute,
		EnableLogging:      true,
		EnableMetrics:      true,
	}
}

// TransactionMiddleware HTTP 트랜잭션 미들웨어
type TransactionMiddleware struct {
	config *MiddlewareConfig
	stats  *MiddlewareStats
}

// MiddlewareStats 미들웨어 통계
type MiddlewareStats struct {
	TotalRequests       int64
	TransactionRequests int64
	SuccessfulCommits   int64
	Rollbacks          int64
	Errors             int64
	AverageResponseTime time.Duration
	LastError          error
	LastErrorTime      time.Time
}

// NewTransactionMiddleware 새 트랜잭션 미들웨어 생성
func NewTransactionMiddleware(config *MiddlewareConfig) *TransactionMiddleware {
	if config == nil {
		panic("middleware config cannot be nil")
	}
	
	return &TransactionMiddleware{
		config: config,
		stats:  &MiddlewareStats{},
	}
}

// Handler HTTP 핸들러 래퍼
func (tm *TransactionMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		tm.stats.TotalRequests++
		
		// 트랜잭션이 필요한지 확인
		if !tm.shouldUseTransaction(r) {
			next.ServeHTTP(w, r)
			return
		}
		
		tm.stats.TransactionRequests++
		
		// 트랜잭션 옵션 생성
		opts := tm.createTransactionOptions(r)
		
		// 트랜잭션 내에서 요청 처리
		err := tm.config.Manager.RunInTx(r.Context(), func(ctx context.Context) error {
			// 트랜잭션 컨텍스트로 요청 업데이트
			r = r.WithContext(ctx)
			
			// 응답 래퍼 생성
			responseWrapper := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}
			
			// 다음 핸들러 실행
			next.ServeHTTP(responseWrapper, r)
			
			// 응답 상태 코드 확인
			if responseWrapper.statusCode >= 400 {
				return fmt.Errorf("HTTP 에러 응답: %d", responseWrapper.statusCode)
			}
			
			return nil
		}, opts)
		
		// 에러 처리
		duration := time.Since(startTime)
		if err != nil {
			tm.handleError(w, r, err, duration)
		} else {
			tm.handleSuccess(duration)
		}
		
		// 로깅
		if tm.config.EnableLogging && tm.config.Logger != nil {
			tm.logRequest(r, err, duration)
		}
	})
}

// shouldUseTransaction 트랜잭션 사용 여부 결정
func (tm *TransactionMiddleware) shouldUseTransaction(r *http.Request) bool {
	// 스킵할 경로 확인
	for _, skipPath := range tm.config.SkipPaths {
		if strings.HasPrefix(r.URL.Path, skipPath) {
			return false
		}
	}
	
	// HTTP 메서드 확인
	method := r.Method
	for _, txMethod := range tm.config.TransactionMethods {
		if method == txMethod {
			return true
		}
	}
	
	// 헤더로 강제 트랜잭션 활성화
	if r.Header.Get("X-Force-Transaction") == "true" {
		return true
	}
	
	return false
}

// createTransactionOptions 요청에서 트랜잭션 옵션 생성
func (tm *TransactionMiddleware) createTransactionOptions(r *http.Request) *storage.TransactionOptions {
	opts := storage.DefaultTransactionOptions()
	
	// 타임아웃 설정
	opts.Timeout = tm.config.DefaultTimeout
	
	// 헤더에서 옵션 읽기
	if timeoutHeader := r.Header.Get("X-Transaction-Timeout"); timeoutHeader != "" {
		if timeout, err := time.ParseDuration(timeoutHeader); err == nil {
			if timeout <= tm.config.MaxTimeout {
				opts.Timeout = timeout
			}
		}
	}
	
	// 읽기 전용 모드
	if r.Header.Get("X-Transaction-ReadOnly") == "true" || r.Method == "GET" {
		opts.ReadOnly = true
	}
	
	// 격리 수준
	if isolationHeader := r.Header.Get("X-Transaction-Isolation"); isolationHeader != "" {
		switch strings.ToUpper(isolationHeader) {
		case "READ_UNCOMMITTED":
			opts.IsolationLevel = storage.IsolationLevelReadUncommitted
		case "READ_COMMITTED":
			opts.IsolationLevel = storage.IsolationLevelReadCommitted
		case "REPEATABLE_READ":
			opts.IsolationLevel = storage.IsolationLevelRepeatableRead
		case "SERIALIZABLE":
			opts.IsolationLevel = storage.IsolationLevelSerializable
		}
	}
	
	// 재시도 설정
	if retryHeader := r.Header.Get("X-Transaction-Retry-Count"); retryHeader != "" {
		if retryCount := parseIntHeader(retryHeader, opts.RetryCount); retryCount >= 0 {
			opts.RetryCount = retryCount
		}
	}
	
	return &opts
}

// handleError 에러 처리
func (tm *TransactionMiddleware) handleError(w http.ResponseWriter, r *http.Request, err error, duration time.Duration) {
	tm.stats.Errors++
	tm.stats.Rollbacks++
	tm.stats.LastError = err
	tm.stats.LastErrorTime = time.Now()
	
	// 평균 응답 시간 업데이트
	tm.updateAverageResponseTime(duration)
	
	// 커스텀 에러 핸들러 사용
	if tm.config.CustomErrorHandler != nil {
		tm.config.CustomErrorHandler(w, r, err)
		return
	}
	
	// 기본 에러 응답
	w.Header().Set("Content-Type", "application/json")
	
	statusCode := http.StatusInternalServerError
	errorMessage := "Internal Server Error"
	
	// 에러 타입에 따른 상태 코드 결정
	if strings.Contains(err.Error(), "timeout") {
		statusCode = http.StatusRequestTimeout
		errorMessage = "Request Timeout"
	} else if strings.Contains(err.Error(), "deadlock") {
		statusCode = http.StatusConflict
		errorMessage = "Resource Conflict"
	} else if strings.Contains(err.Error(), "HTTP 에러 응답") {
		// 이미 핸들러에서 응답을 보냈으므로 추가 처리 불요
		return
	}
	
	w.WriteHeader(statusCode)
	
	errorResponse := map[string]interface{}{
		"error":     errorMessage,
		"message":   err.Error(),
		"timestamp": time.Now().Unix(),
	}
	
	json.NewEncoder(w).Encode(errorResponse)
}

// handleSuccess 성공 처리
func (tm *TransactionMiddleware) handleSuccess(duration time.Duration) {
	tm.stats.SuccessfulCommits++
	tm.updateAverageResponseTime(duration)
}

// updateAverageResponseTime 평균 응답 시간 업데이트
func (tm *TransactionMiddleware) updateAverageResponseTime(duration time.Duration) {
	if tm.stats.TotalRequests == 1 {
		tm.stats.AverageResponseTime = duration
	} else {
		tm.stats.AverageResponseTime = (tm.stats.AverageResponseTime + duration) / 2
	}
}

// logRequest 요청 로깅
func (tm *TransactionMiddleware) logRequest(r *http.Request, err error, duration time.Duration) {
	status := "SUCCESS"
	if err != nil {
		status = "ERROR"
	}
	
	tm.config.Logger.Printf(
		"Transaction Request: %s %s - Status: %s - Duration: %v - Error: %v",
		r.Method, r.URL.Path, status, duration, err,
	)
}

// GetStats 미들웨어 통계 조회
func (tm *TransactionMiddleware) GetStats() *MiddlewareStats {
	statsCopy := *tm.stats
	return &statsCopy
}

// ResetStats 통계 초기화
func (tm *TransactionMiddleware) ResetStats() {
	tm.stats = &MiddlewareStats{}
}

// responseWriter HTTP 응답 래퍼
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

// WriteHeader 상태 코드 기록
func (rw *responseWriter) WriteHeader(statusCode int) {
	if !rw.written {
		rw.statusCode = statusCode
		rw.written = true
		rw.ResponseWriter.WriteHeader(statusCode)
	}
}

// Write 응답 작성
func (rw *responseWriter) Write(data []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(data)
}

// TransactionInfoHandler 트랜잭션 정보 핸들러
func (tm *TransactionMiddleware) TransactionInfoHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		info := map[string]interface{}{
			"transaction_middleware_stats": tm.GetStats(),
			"manager_stats":               tm.config.Manager.GetStats(),
			"in_transaction":              tm.config.Manager.IsInTransaction(r.Context()),
		}
		
		// 현재 트랜잭션 정보
		if tx, exists := tm.config.Manager.Current(r.Context()); exists {
			info["current_transaction"] = map[string]interface{}{
				"closed": tx.IsClosed(),
			}
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(info)
	}
}

// parseIntHeader 헤더에서 정수 값 파싱
func parseIntHeader(value string, defaultValue int) int {
	if value == "" {
		return defaultValue
	}
	
	var result int
	if n, err := fmt.Sscanf(value, "%d", &result); err != nil || n != 1 {
		return defaultValue
	}
	
	return result
}

// GinMiddleware Gin 프레임워크용 미들웨어 (필요한 경우)
// 참고: 이 프로젝트에서 Gin을 사용하는지 확인 후 구현
/*
import "github.com/gin-gonic/gin"

func (tm *TransactionMiddleware) GinMiddleware() gin.HandlerFunc {
	return gin.WrapH(tm.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Gin 컨텍스트를 HTTP 핸들러로 변환
		c := w.(*gin.Context)
		c.Next()
	})))
}
*/

// EchoMiddleware Echo 프레임워크용 미들웨어 (필요한 경우)
// 참고: 이 프로젝트에서 Echo를 사용하는지 확인 후 구현
/*
import "github.com/labstack/echo/v4"

func (tm *TransactionMiddleware) EchoMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Echo 컨텍스트를 HTTP 핸들러로 변환하여 처리
			// 구현 생략
			return next(c)
		}
	}
}
*/