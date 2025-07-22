package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"

	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/security"
)

// AuditConfig는 감사 로깅 설정입니다.
type AuditConfig struct {
	// Redis 클라이언트
	Redis redis.UniversalClient
	
	// 로깅 설정
	EnableRequestLogging  bool          // 요청 로깅 활성화
	EnableResponseLogging bool          // 응답 로깅 활성화
	EnableBodyLogging     bool          // 요청/응답 바디 로깅
	MaxBodySize          int64         // 최대 바디 크기 (바이트)
	
	// 필터 설정
	SkipPaths            []string      // 로깅에서 제외할 경로
	SkipMethods          []string      // 로깅에서 제외할 HTTP 메서드
	LoggedPaths          []string      // 특별히 로깅할 경로 (비어있으면 모두)
	SensitiveHeaders     []string      // 마스킹할 헤더
	SensitiveFields      []string      // 마스킹할 필드
	
	// 보안 설정
	MaskSensitiveData    bool          // 민감한 데이터 마스킹
	IncludeSystemEvents  bool          // 시스템 이벤트 포함
	
	// 저장 설정
	RetentionPeriod      time.Duration // 로그 보관 기간
	BatchSize           int           // 배치 저장 크기
	FlushInterval       time.Duration // 배치 플러시 간격
	
	// 이벤트 추적기
	EventTracker *security.EventTracker
	
	Logger *zap.Logger
}

// AuditLog는 감사 로그를 나타냅니다.
type AuditLog struct {
	ID            string                 `json:"id"`
	Timestamp     time.Time              `json:"timestamp"`
	RequestID     string                 `json:"request_id,omitempty"`
	UserID        string                 `json:"user_id,omitempty"`
	SessionID     string                 `json:"session_id,omitempty"`
	IPAddress     string                 `json:"ip_address"`
	UserAgent     string                 `json:"user_agent"`
	Method        string                 `json:"method"`
	URL           string                 `json:"url"`
	Path          string                 `json:"path"`
	Query         string                 `json:"query,omitempty"`
	Headers       map[string]string      `json:"headers,omitempty"`
	RequestBody   string                 `json:"request_body,omitempty"`
	ResponseCode  int                    `json:"response_code"`
	ResponseSize  int64                  `json:"response_size"`
	ResponseBody  string                 `json:"response_body,omitempty"`
	Duration      time.Duration          `json:"duration"`
	Error         string                 `json:"error,omitempty"`
	Tags          []string               `json:"tags,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// responseWriter는 응답을 캡처하기 위한 래퍼입니다.
type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
	size int64
}

// AuditLogger는 감사 로깅 미들웨어입니다.
type AuditLogger struct {
	config      *AuditConfig
	redis       redis.UniversalClient
	logger      *zap.Logger
	logBuffer   chan *AuditLog
	stopChan    chan struct{}
}

// DefaultAuditConfig는 기본 감사 로깅 설정을 반환합니다.
func DefaultAuditConfig() *AuditConfig {
	return &AuditConfig{
		EnableRequestLogging:  true,
		EnableResponseLogging: true,
		EnableBodyLogging:     false,
		MaxBodySize:          1024 * 1024, // 1MB
		
		SkipPaths: []string{
			"/health",
			"/metrics",
			"/favicon.ico",
			"/robots.txt",
		},
		SkipMethods: []string{"OPTIONS"},
		
		SensitiveHeaders: []string{
			"Authorization",
			"Cookie",
			"X-CSRF-Token",
			"X-API-Key",
		},
		SensitiveFields: []string{
			"password",
			"token",
			"secret",
			"key",
			"credential",
		},
		
		MaskSensitiveData:   true,
		IncludeSystemEvents: false,
		RetentionPeriod:     30 * 24 * time.Hour, // 30일
		BatchSize:           100,
		FlushInterval:       10 * time.Second,
	}
}

// NewAuditLogger는 새로운 감사 로거를 생성합니다.
func NewAuditLogger(config *AuditConfig) *AuditLogger {
	if config == nil {
		config = DefaultAuditConfig()
	}

	al := &AuditLogger{
		config:    config,
		redis:     config.Redis,
		logger:    config.Logger,
		logBuffer: make(chan *AuditLog, config.BatchSize*2),
		stopChan:  make(chan struct{}),
	}

	// 배치 처리 워커 시작
	go al.startBatchWorker()

	return al
}

// AuditMiddleware는 감사 로깅 미들웨어를 생성합니다.
func AuditMiddleware(config *AuditConfig) gin.HandlerFunc {
	al := NewAuditLogger(config)
	return al.Handler()
}

// Handler는 감사 로깅 미들웨어 핸들러를 반환합니다.
func (al *AuditLogger) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 스킵 조건 확인
		if al.shouldSkip(c) {
			c.Next()
			return
		}

		startTime := time.Now()
		requestID := al.getOrGenerateRequestID(c)

		// 요청 정보 수집
		auditLog := &AuditLog{
			ID:        al.generateLogID(),
			Timestamp: startTime,
			RequestID: requestID,
			UserID:    al.getUserID(c),
			SessionID: al.getSessionID(c),
			IPAddress: c.ClientIP(),
			UserAgent: c.GetHeader("User-Agent"),
			Method:    c.Request.Method,
			URL:       c.Request.URL.String(),
			Path:      c.Request.URL.Path,
			Query:     c.Request.URL.RawQuery,
			Metadata:  make(map[string]interface{}),
		}

		// 헤더 수집
		if al.config.EnableRequestLogging {
			auditLog.Headers = al.collectHeaders(c)
		}

		// 요청 바디 수집
		if al.config.EnableBodyLogging {
			auditLog.RequestBody = al.collectRequestBody(c)
		}

		// 응답 캡처를 위한 래퍼 설정
		var respWriter *responseWriter
		if al.config.EnableResponseLogging {
			respWriter = &responseWriter{
				ResponseWriter: c.Writer,
				body:          bytes.NewBuffer([]byte{}),
			}
			c.Writer = respWriter
		}

		// 다음 핸들러 실행
		c.Next()

		// 응답 정보 수집
		auditLog.Duration = time.Since(startTime)
		auditLog.ResponseCode = c.Writer.Status()

		if respWriter != nil {
			auditLog.ResponseSize = respWriter.size
			if al.config.EnableBodyLogging {
				auditLog.ResponseBody = al.maskSensitiveData(respWriter.body.String())
			}
		}

		// 에러 정보 수집
		if len(c.Errors) > 0 {
			auditLog.Error = c.Errors.String()
		}

		// 태그 추가
		auditLog.Tags = al.generateTags(c, auditLog)

		// 보안 이벤트 감지 및 기록
		al.detectSecurityEvents(c, auditLog)

		// 비동기 로깅
		select {
		case al.logBuffer <- auditLog:
			// 성공
		default:
			// 버퍼가 가득 찬 경우 즉시 로깅
			al.logImmediately(auditLog)
		}
	}
}

// Write는 responseWriter의 Write 메서드를 구현합니다.
func (rw *responseWriter) Write(b []byte) (int, error) {
	rw.body.Write(b)
	rw.size += int64(len(b))
	return rw.ResponseWriter.Write(b)
}

// WriteHeader는 responseWriter의 WriteHeader 메서드를 구현합니다.
func (rw *responseWriter) WriteHeader(code int) {
	rw.ResponseWriter.WriteHeader(code)
}

// shouldSkip는 로깅을 건너뛸지 확인합니다.
func (al *AuditLogger) shouldSkip(c *gin.Context) bool {
	path := c.Request.URL.Path
	method := c.Request.Method

	// 제외할 경로 확인
	for _, skipPath := range al.config.SkipPaths {
		if strings.HasPrefix(path, skipPath) {
			return true
		}
	}

	// 제외할 메서드 확인
	for _, skipMethod := range al.config.SkipMethods {
		if method == skipMethod {
			return true
		}
	}

	// 특정 경로만 로깅하는 경우
	if len(al.config.LoggedPaths) > 0 {
		for _, loggedPath := range al.config.LoggedPaths {
			if strings.HasPrefix(path, loggedPath) {
				return false
			}
		}
		return true
	}

	return false
}

// getOrGenerateRequestID는 요청 ID를 가져오거나 생성합니다.
func (al *AuditLogger) getOrGenerateRequestID(c *gin.Context) string {
	// 기존 Request ID 확인
	if requestID := c.GetHeader("X-Request-ID"); requestID != "" {
		return requestID
	}
	
	// 컨텍스트에서 확인
	if requestID, exists := c.Get("request_id"); exists {
		if id, ok := requestID.(string); ok {
			return id
		}
	}

	// 새로 생성
	return al.generateRequestID()
}

// collectHeaders는 요청 헤더를 수집합니다.
func (al *AuditLogger) collectHeaders(c *gin.Context) map[string]string {
	headers := make(map[string]string)
	
	for name, values := range c.Request.Header {
		if al.isSensitiveHeader(name) && al.config.MaskSensitiveData {
			headers[name] = "[MASKED]"
		} else {
			headers[name] = strings.Join(values, ", ")
		}
	}

	return headers
}

// collectRequestBody는 요청 바디를 수집합니다.
func (al *AuditLogger) collectRequestBody(c *gin.Context) string {
	if c.Request.Body == nil {
		return ""
	}

	// 바디 크기 제한 확인
	if c.Request.ContentLength > al.config.MaxBodySize {
		return fmt.Sprintf("[BODY_TOO_LARGE: %d bytes]", c.Request.ContentLength)
	}

	// 바디 읽기
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return fmt.Sprintf("[ERROR_READING_BODY: %v]", err)
	}

	// 바디 복원 (다른 미들웨어에서 사용할 수 있도록)
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	// Content-Type 확인
	contentType := c.GetHeader("Content-Type")
	if strings.Contains(contentType, "multipart/form-data") {
		return "[MULTIPART_DATA]"
	}

	body := string(bodyBytes)
	if al.config.MaskSensitiveData {
		body = al.maskSensitiveData(body)
	}

	return body
}

// isSensitiveHeader는 민감한 헤더인지 확인합니다.
func (al *AuditLogger) isSensitiveHeader(headerName string) bool {
	headerLower := strings.ToLower(headerName)
	for _, sensitive := range al.config.SensitiveHeaders {
		if strings.ToLower(sensitive) == headerLower {
			return true
		}
	}
	return false
}

// maskSensitiveData는 민감한 데이터를 마스킹합니다.
func (al *AuditLogger) maskSensitiveData(data string) string {
	if !al.config.MaskSensitiveData {
		return data
	}

	// JSON 파싱 시도
	var jsonData interface{}
	if err := json.Unmarshal([]byte(data), &jsonData); err == nil {
		masked := al.maskJSONFields(jsonData)
		if maskedBytes, err := json.Marshal(masked); err == nil {
			return string(maskedBytes)
		}
	}

	// JSON이 아닌 경우 문자열 대체
	maskedData := data
	for _, field := range al.config.SensitiveFields {
		patterns := []string{
			fmt.Sprintf(`"%s"\s*:\s*"[^"]*"`, field),
			fmt.Sprintf(`%s=\w+`, field),
		}
		
		for _, pattern := range patterns {
			maskedData = strings.ReplaceAll(maskedData, pattern, fmt.Sprintf(`"%s":"[MASKED]"`, field))
		}
	}

	return maskedData
}

// maskJSONFields는 JSON 객체의 민감한 필드를 마스킹합니다.
func (al *AuditLogger) maskJSONFields(data interface{}) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, value := range v {
			if al.isSensitiveField(key) {
				result[key] = "[MASKED]"
			} else {
				result[key] = al.maskJSONFields(value)
			}
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = al.maskJSONFields(item)
		}
		return result
	default:
		return v
	}
}

// isSensitiveField는 민감한 필드인지 확인합니다.
func (al *AuditLogger) isSensitiveField(fieldName string) bool {
	fieldLower := strings.ToLower(fieldName)
	for _, sensitive := range al.config.SensitiveFields {
		if strings.Contains(fieldLower, strings.ToLower(sensitive)) {
			return true
		}
	}
	return false
}

// generateTags는 로그에 태그를 생성합니다.
func (al *AuditLogger) generateTags(c *gin.Context, log *AuditLog) []string {
	tags := make([]string, 0)

	// HTTP 상태 코드 기반 태그
	if log.ResponseCode >= 400 {
		tags = append(tags, "error")
		if log.ResponseCode >= 500 {
			tags = append(tags, "server_error")
		} else {
			tags = append(tags, "client_error")
		}
	}

	// 인증 상태 태그
	if log.UserID != "" {
		tags = append(tags, "authenticated")
	} else {
		tags = append(tags, "anonymous")
	}

	// 요청 타입 태그
	if strings.Contains(log.Path, "/api/") {
		tags = append(tags, "api")
	}

	// 관리자 경로 태그
	if strings.Contains(log.Path, "/admin/") {
		tags = append(tags, "admin")
	}

	// 보안 관련 경로 태그
	if strings.Contains(log.Path, "/auth/") {
		tags = append(tags, "auth")
	}

	// 긴 응답 시간 태그
	if log.Duration > 5*time.Second {
		tags = append(tags, "slow_response")
	}

	return tags
}

// detectSecurityEvents는 보안 이벤트를 감지하고 기록합니다.
func (al *AuditLogger) detectSecurityEvents(c *gin.Context, log *AuditLog) {
	if al.config.EventTracker == nil {
		return
	}

	ctx := c.Request.Context()

	// 인증 실패 감지
	if strings.Contains(log.Path, "/auth/") && log.ResponseCode == 401 {
		al.recordSecurityEvent(ctx, &security.SecurityEvent{
			Type:        security.EventTypeAuthFailure,
			Severity:    security.SeverityMedium,
			Source:      log.IPAddress,
			Target:      log.Path,
			UserID:      log.UserID,
			SessionID:   log.SessionID,
			IPAddress:   log.IPAddress,
			UserAgent:   log.UserAgent,
			RequestPath: log.Path,
			Method:      log.Method,
			StatusCode:  log.ResponseCode,
			Details: map[string]interface{}{
				"request_id": log.RequestID,
				"duration":   log.Duration,
			},
		})
	}

	// 권한 에러 감지
	if log.ResponseCode == 403 {
		al.recordSecurityEvent(ctx, &security.SecurityEvent{
			Type:        security.EventTypePrivilegeEscalation,
			Severity:    security.SeverityHigh,
			Source:      log.IPAddress,
			Target:      log.Path,
			UserID:      log.UserID,
			SessionID:   log.SessionID,
			IPAddress:   log.IPAddress,
			UserAgent:   log.UserAgent,
			RequestPath: log.Path,
			Method:      log.Method,
			StatusCode:  log.ResponseCode,
			Details: map[string]interface{}{
				"request_id": log.RequestID,
			},
		})
	}

	// 의심스러운 사용자 에이전트 감지
	if al.isSuspiciousUserAgent(log.UserAgent) {
		al.recordSecurityEvent(ctx, &security.SecurityEvent{
			Type:        security.EventTypeSuspiciousActivity,
			Severity:    security.SeverityMedium,
			Source:      log.IPAddress,
			Target:      "user_agent",
			UserID:      log.UserID,
			IPAddress:   log.IPAddress,
			UserAgent:   log.UserAgent,
			RequestPath: log.Path,
			Details: map[string]interface{}{
				"suspicious_ua": log.UserAgent,
				"request_id":    log.RequestID,
			},
		})
	}

	// SQL Injection 패턴 감지
	if al.containsSQLInjectionPattern(log.URL + log.RequestBody) {
		al.recordSecurityEvent(ctx, &security.SecurityEvent{
			Type:        security.EventTypeMaliciousRequest,
			Severity:    security.SeverityHigh,
			Source:      log.IPAddress,
			Target:      "sql_injection",
			UserID:      log.UserID,
			IPAddress:   log.IPAddress,
			UserAgent:   log.UserAgent,
			RequestPath: log.Path,
			Method:      log.Method,
			Details: map[string]interface{}{
				"request_id": log.RequestID,
				"pattern":    "sql_injection",
			},
		})
	}
}

// recordSecurityEvent는 보안 이벤트를 기록합니다.
func (al *AuditLogger) recordSecurityEvent(ctx context.Context, event *security.SecurityEvent) {
	if err := al.config.EventTracker.RecordEvent(ctx, event); err != nil {
		al.logger.Error("보안 이벤트 기록 실패", zap.Error(err))
	}
}

// isSuspiciousUserAgent는 의심스러운 사용자 에이전트인지 확인합니다.
func (al *AuditLogger) isSuspiciousUserAgent(userAgent string) bool {
	suspiciousPatterns := []string{
		"sqlmap",
		"nikto",
		"nmap",
		"masscan",
		"python-requests",
		"curl/7",
		"wget",
		"bot",
		"crawler",
		"spider",
		"scanner",
	}

	userAgentLower := strings.ToLower(userAgent)
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(userAgentLower, pattern) {
			return true
		}
	}

	return false
}

// containsSQLInjectionPattern은 SQL Injection 패턴을 확인합니다.
func (al *AuditLogger) containsSQLInjectionPattern(content string) bool {
	patterns := []string{
		"UNION SELECT",
		"DROP TABLE",
		"DELETE FROM",
		"INSERT INTO",
		"UPDATE SET",
		"' OR '1'='1",
		"' OR 1=1",
		"'; DROP",
		"/*",
		"*/",
		"xp_cmdshell",
		"sp_executesql",
	}

	contentUpper := strings.ToUpper(content)
	for _, pattern := range patterns {
		if strings.Contains(contentUpper, pattern) {
			return true
		}
	}

	return false
}

// getUserID는 사용자 ID를 추출합니다.
func (al *AuditLogger) getUserID(c *gin.Context) string {
	if userID, exists := c.Get("user_id"); exists {
		if uid, ok := userID.(string); ok {
			return uid
		}
	}
	return ""
}

// getSessionID는 세션 ID를 추출합니다.
func (al *AuditLogger) getSessionID(c *gin.Context) string {
	if sessionID, exists := c.Get("session_id"); exists {
		if sid, ok := sessionID.(string); ok {
			return sid
		}
	}
	return ""
}

// generateLogID는 로그 ID를 생성합니다.
func (al *AuditLogger) generateLogID() string {
	return fmt.Sprintf("audit_%d_%d", time.Now().UnixNano(), time.Now().Unix())
}

// generateRequestID는 요청 ID를 생성합니다.
func (al *AuditLogger) generateRequestID() string {
	return fmt.Sprintf("req_%d", time.Now().UnixNano())
}

// logImmediately는 즉시 로그를 저장합니다.
func (al *AuditLogger) logImmediately(log *AuditLog) {
	if err := al.storeLog(log); err != nil {
		al.logger.Error("감사 로그 저장 실패", zap.Error(err), zap.String("log_id", log.ID))
	}
}

// storeLog는 로그를 Redis에 저장합니다.
func (al *AuditLogger) storeLog(log *AuditLog) error {
	if al.redis == nil {
		// Redis가 없는 경우 파일 로그로 대체
		al.logger.Info("audit_log", zap.Any("log", log))
		return nil
	}

	ctx := context.Background()
	
	// 로그 직렬화
	logData, err := json.Marshal(log)
	if err != nil {
		return fmt.Errorf("로그 직렬화 실패: %w", err)
	}

	// Redis에 저장
	pipe := al.redis.Pipeline()

	// 개별 로그 저장
	logKey := fmt.Sprintf("audit:log:%s", log.ID)
	pipe.Set(ctx, logKey, logData, al.config.RetentionPeriod)

	// 시계열 인덱스에 추가
	timeSeriesKey := fmt.Sprintf("audit:timeline:%s", log.Timestamp.Format("2006-01-02"))
	pipe.ZAdd(ctx, timeSeriesKey, &redis.Z{
		Score:  float64(log.Timestamp.Unix()),
		Member: log.ID,
	})
	pipe.Expire(ctx, timeSeriesKey, al.config.RetentionPeriod)

	// 사용자별 인덱스에 추가
	if log.UserID != "" {
		userKey := fmt.Sprintf("audit:user:%s", log.UserID)
		pipe.ZAdd(ctx, userKey, &redis.Z{
			Score:  float64(log.Timestamp.Unix()),
			Member: log.ID,
		})
		pipe.Expire(ctx, userKey, al.config.RetentionPeriod)
	}

	// 태그별 인덱스에 추가
	for _, tag := range log.Tags {
		tagKey := fmt.Sprintf("audit:tag:%s", tag)
		pipe.ZAdd(ctx, tagKey, &redis.Z{
			Score:  float64(log.Timestamp.Unix()),
			Member: log.ID,
		})
		pipe.Expire(ctx, tagKey, al.config.RetentionPeriod)
	}

	_, err = pipe.Exec(ctx)
	return err
}

// startBatchWorker는 배치 처리 워커를 시작합니다.
func (al *AuditLogger) startBatchWorker() {
	ticker := time.NewTicker(al.config.FlushInterval)
	defer ticker.Stop()

	batch := make([]*AuditLog, 0, al.config.BatchSize)

	for {
		select {
		case log := <-al.logBuffer:
			batch = append(batch, log)
			
			// 배치가 가득 찬 경우 즉시 처리
			if len(batch) >= al.config.BatchSize {
				al.processBatch(batch)
				batch = batch[:0] // 슬라이스 리셋
			}

		case <-ticker.C:
			// 주기적으로 배치 처리
			if len(batch) > 0 {
				al.processBatch(batch)
				batch = batch[:0]
			}

		case <-al.stopChan:
			// 종료 시 남은 배치 처리
			if len(batch) > 0 {
				al.processBatch(batch)
			}
			return
		}
	}
}

// processBatch는 배치를 처리합니다.
func (al *AuditLogger) processBatch(batch []*AuditLog) {
	for _, log := range batch {
		if err := al.storeLog(log); err != nil {
			al.logger.Error("감사 로그 배치 저장 실패", zap.Error(err), zap.String("log_id", log.ID))
		}
	}
	
	al.logger.Debug("감사 로그 배치 처리 완료", zap.Int("count", len(batch)))
}

// Close는 리소스를 정리합니다.
func (al *AuditLogger) Close() error {
	close(al.stopChan)
	return nil
}