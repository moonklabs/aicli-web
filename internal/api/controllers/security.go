package controllers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/aicli/aicli-web/internal/middleware"
	"github.com/aicli/aicli-web/internal/security"
)

// SecurityController는 보안 관리 API 컨트롤러입니다.
type SecurityController struct {
	eventTracker   *security.EventTracker
	attackDetector *security.AttackDetector
	rateLimiter    *middleware.AdvancedRateLimiter
	logger         *zap.Logger
}

// SecurityEventResponse는 보안 이벤트 응답입니다.
type SecurityEventResponse struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Severity    string                 `json:"severity"`
	Source      string                 `json:"source"`
	Target      string                 `json:"target"`
	Details     map[string]interface{} `json:"details"`
	Timestamp   time.Time              `json:"timestamp"`
	UserID      string                 `json:"user_id,omitempty"`
	SessionID   string                 `json:"session_id,omitempty"`
	IPAddress   string                 `json:"ip_address"`
	UserAgent   string                 `json:"user_agent"`
	RequestPath string                 `json:"request_path,omitempty"`
	Method      string                 `json:"method,omitempty"`
	StatusCode  int                    `json:"status_code,omitempty"`
	Resolved    bool                   `json:"resolved"`
	ResolvedAt  *time.Time             `json:"resolved_at,omitempty"`
}

// SecurityPolicyRequest는 보안 정책 요청입니다.
type SecurityPolicyRequest struct {
	Name     string                 `json:"name" binding:"required"`
	Type     string                 `json:"type" binding:"required"`
	Rules    map[string]interface{} `json:"rules" binding:"required"`
	IsActive bool                   `json:"is_active"`
}

// SecurityPolicyResponse는 보안 정책 응답입니다.
type SecurityPolicyResponse struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Type      string                 `json:"type"`
	Rules     map[string]interface{} `json:"rules"`
	IsActive  bool                   `json:"is_active"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// AttackPatternRequest는 공격 패턴 요청입니다.
type AttackPatternRequest struct {
	Name        string  `json:"name" binding:"required"`
	Type        string  `json:"type" binding:"required"`
	Pattern     string  `json:"pattern" binding:"required"`
	Severity    string  `json:"severity" binding:"required"`
	Confidence  float64 `json:"confidence" binding:"required,min=0,max=1"`
	Description string  `json:"description"`
	IsActive    bool    `json:"is_active"`
}

// AttackPatternResponse는 공격 패턴 응답입니다.
type AttackPatternResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Pattern     string    `json:"pattern"`
	Severity    string    `json:"severity"`
	Confidence  float64   `json:"confidence"`
	Description string    `json:"description"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// AttackDetectionRequest는 공격 탐지 요청입니다.
type AttackDetectionRequest struct {
	UserID    string            `json:"user_id,omitempty"`
	SessionID string            `json:"session_id,omitempty"`
	IPAddress string            `json:"ip_address" binding:"required"`
	UserAgent string            `json:"user_agent"`
	Method    string            `json:"method" binding:"required"`
	URL       string            `json:"url" binding:"required"`
	Path      string            `json:"path" binding:"required"`
	Query     string            `json:"query,omitempty"`
	Headers   map[string]string `json:"headers,omitempty"`
	Body      string            `json:"body,omitempty"`
}

// NewSecurityController는 새로운 보안 컨트롤러를 생성합니다.
func NewSecurityController(
	eventTracker *security.EventTracker,
	attackDetector *security.AttackDetector,
	rateLimiter *middleware.AdvancedRateLimiter,
	logger *zap.Logger,
) *SecurityController {
	return &SecurityController{
		eventTracker:   eventTracker,
		attackDetector: attackDetector,
		rateLimiter:    rateLimiter,
		logger:         logger,
	}
}

// GetSecurityEvents는 보안 이벤트 목록을 조회합니다.
// @Summary 보안 이벤트 조회
// @Description 보안 이벤트 목록을 필터 조건에 따라 조회합니다
// @Tags security
// @Accept json
// @Produce json
// @Param types query string false "이벤트 타입 (쉼표로 구분)"
// @Param severities query string false "심각도 (쉼표로 구분)"
// @Param user_id query string false "사용자 ID"
// @Param ip_address query string false "IP 주소"
// @Param start_time query string false "시작 시간 (RFC3339)"
// @Param end_time query string false "종료 시간 (RFC3339)"
// @Param resolved query bool false "해결 상태"
// @Param limit query int false "제한 수" default(100)
// @Param offset query int false "오프셋" default(0)
// @Success 200 {object} map[string]interface{} "보안 이벤트 목록"
// @Failure 400 {object} map[string]interface{} "잘못된 요청"
// @Failure 500 {object} map[string]interface{} "서버 오류"
// @Router /api/v1/security/events [get]
func (sc *SecurityController) GetSecurityEvents(c *gin.Context) {
	filter := &security.EventFilter{}

	// 쿼리 파라미터 파싱
	if types := c.Query("types"); types != "" {
		filter.Types = parseEventTypes(types)
	}
	if severities := c.Query("severities"); severities != "" {
		filter.Severities = parseSeverities(severities)
	}
	filter.UserID = c.Query("user_id")
	filter.IPAddress = c.Query("ip_address")
	filter.Source = c.Query("source")
	filter.Target = c.Query("target")

	// 시간 필터
	if startTime := c.Query("start_time"); startTime != "" {
		if t, err := time.Parse(time.RFC3339, startTime); err == nil {
			filter.StartTime = &t
		}
	}
	if endTime := c.Query("end_time"); endTime != "" {
		if t, err := time.Parse(time.RFC3339, endTime); err == nil {
			filter.EndTime = &t
		}
	}

	// 해결 상태 필터
	if resolved := c.Query("resolved"); resolved != "" {
		if r, err := strconv.ParseBool(resolved); err == nil {
			filter.Resolved = &r
		}
	}

	// 페이지네이션
	if limit := c.Query("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil && l > 0 {
			filter.Limit = l
		} else {
			filter.Limit = 100
		}
	} else {
		filter.Limit = 100
	}

	if offset := c.Query("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil && o >= 0 {
			filter.Offset = o
		}
	}

	// 이벤트 조회
	events, err := sc.eventTracker.QueryEvents(c.Request.Context(), filter)
	if err != nil {
		sc.logger.Error("보안 이벤트 조회 실패", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "보안 이벤트 조회에 실패했습니다",
		})
		return
	}

	// 응답 변환
	response := make([]*SecurityEventResponse, len(events))
	for i, event := range events {
		response[i] = convertToSecurityEventResponse(event)
	}

	c.JSON(http.StatusOK, gin.H{
		"events": response,
		"total":  len(response),
		"limit":  filter.Limit,
		"offset": filter.Offset,
	})
}

// GetSecurityEvent는 특정 보안 이벤트를 조회합니다.
// @Summary 보안 이벤트 상세 조회
// @Description 특정 보안 이벤트의 상세 정보를 조회합니다
// @Tags security
// @Accept json
// @Produce json
// @Param id path string true "이벤트 ID"
// @Success 200 {object} SecurityEventResponse "보안 이벤트 정보"
// @Failure 404 {object} map[string]interface{} "이벤트를 찾을 수 없음"
// @Failure 500 {object} map[string]interface{} "서버 오류"
// @Router /api/v1/security/events/{id} [get]
func (sc *SecurityController) GetSecurityEvent(c *gin.Context) {
	eventID := c.Param("id")

	event, err := sc.eventTracker.GetEvent(c.Request.Context(), eventID)
	if err != nil {
		sc.logger.Error("보안 이벤트 조회 실패", zap.String("event_id", eventID), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Not Found",
			"message": "보안 이벤트를 찾을 수 없습니다",
		})
		return
	}

	response := convertToSecurityEventResponse(event)
	c.JSON(http.StatusOK, response)
}

// ResolveSecurityEvent는 보안 이벤트를 해결됨으로 표시합니다.
// @Summary 보안 이벤트 해결
// @Description 보안 이벤트를 해결됨으로 표시합니다
// @Tags security
// @Accept json
// @Produce json
// @Param id path string true "이벤트 ID"
// @Success 200 {object} map[string]interface{} "성공 메시지"
// @Failure 404 {object} map[string]interface{} "이벤트를 찾을 수 없음"
// @Failure 500 {object} map[string]interface{} "서버 오류"
// @Router /api/v1/security/events/{id}/resolve [post]
func (sc *SecurityController) ResolveSecurityEvent(c *gin.Context) {
	eventID := c.Param("id")

	err := sc.eventTracker.ResolveEvent(c.Request.Context(), eventID)
	if err != nil {
		sc.logger.Error("보안 이벤트 해결 실패", zap.String("event_id", eventID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "보안 이벤트 해결에 실패했습니다",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "보안 이벤트가 해결되었습니다",
	})
}

// GetSecurityStatistics는 보안 통계를 조회합니다.
// @Summary 보안 통계 조회
// @Description 보안 이벤트 및 공격 탐지 통계를 조회합니다
// @Tags security
// @Accept json
// @Produce json
// @Param period query string false "기간 (예: 24h, 7d, 30d)" default(24h)
// @Success 200 {object} map[string]interface{} "보안 통계"
// @Failure 400 {object} map[string]interface{} "잘못된 요청"
// @Failure 500 {object} map[string]interface{} "서버 오류"
// @Router /api/v1/security/statistics [get]
func (sc *SecurityController) GetSecurityStatistics(c *gin.Context) {
	periodStr := c.DefaultQuery("period", "24h")
	period, err := time.ParseDuration(periodStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "유효하지 않은 기간 형식입니다",
		})
		return
	}

	// 이벤트 통계
	eventStats, err := sc.eventTracker.GetStatistics(c.Request.Context(), period)
	if err != nil {
		sc.logger.Error("보안 이벤트 통계 조회 실패", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "보안 통계 조회에 실패했습니다",
		})
		return
	}

	// 공격 탐지 통계
	attackStats, err := sc.attackDetector.GetStatistics(c.Request.Context())
	if err != nil {
		sc.logger.Error("공격 탐지 통계 조회 실패", zap.Error(err))
		attackStats = make(map[string]interface{})
	}

	// Rate Limit 통계
	var rateLimitStats map[string]interface{}
	if sc.rateLimiter != nil {
		rlStats, err := sc.rateLimiter.GetStats(c.Request.Context())
		if err == nil {
			rateLimitStats = rlStats
		} else {
			rateLimitStats = make(map[string]interface{})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"period":      periodStr,
		"events":      eventStats,
		"attacks":     attackStats,
		"rate_limits": rateLimitStats,
		"timestamp":   time.Now(),
	})
}

// DetectAttack은 요청에 대해 공격 패턴을 탐지합니다.
// @Summary 공격 패턴 탐지
// @Description 주어진 요청에 대해 공격 패턴을 탐지합니다
// @Tags security
// @Accept json
// @Produce json
// @Param request body AttackDetectionRequest true "탐지 요청"
// @Success 200 {object} map[string]interface{} "탐지 결과"
// @Failure 400 {object} map[string]interface{} "잘못된 요청"
// @Failure 500 {object} map[string]interface{} "서버 오류"
// @Router /api/v1/security/detect-attack [post]
func (sc *SecurityController) DetectAttack(c *gin.Context) {
	var req AttackDetectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "잘못된 요청 형식입니다",
			"details": err.Error(),
		})
		return
	}

	// 탐지 요청 변환
	detectionReq := &security.AttackDetectionRequest{
		UserID:    req.UserID,
		SessionID: req.SessionID,
		IPAddress: req.IPAddress,
		UserAgent: req.UserAgent,
		Method:    req.Method,
		URL:       req.URL,
		Path:      req.Path,
		Query:     req.Query,
		Headers:   req.Headers,
		Body:      req.Body,
		Timestamp: time.Now(),
	}

	// 공격 탐지 수행
	result := sc.attackDetector.DetectAttacks(c.Request.Context(), detectionReq)

	c.JSON(http.StatusOK, gin.H{
		"is_attack":       result.IsAttack,
		"attack_type":     result.AttackType,
		"confidence":      result.Confidence,
		"risk":           result.Risk,
		"patterns":       len(result.Patterns),
		"evidence":       result.Evidence,
		"recommendations": result.Recommendations,
	})
}

// GetAttackPatterns는 공격 패턴 목록을 조회합니다.
// @Summary 공격 패턴 목록 조회
// @Description 현재 설정된 공격 패턴 목록을 조회합니다
// @Tags security
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "공격 패턴 목록"
// @Failure 500 {object} map[string]interface{} "서버 오류"
// @Router /api/v1/security/attack-patterns [get]
func (sc *SecurityController) GetAttackPatterns(c *gin.Context) {
	patterns := sc.attackDetector.GetPatterns()
	
	response := make([]*AttackPatternResponse, len(patterns))
	for i, pattern := range patterns {
		response[i] = convertToAttackPatternResponse(pattern)
	}

	c.JSON(http.StatusOK, gin.H{
		"patterns": response,
		"total":    len(response),
	})
}

// CreateAttackPattern은 새로운 공격 패턴을 생성합니다.
// @Summary 공격 패턴 생성
// @Description 새로운 공격 패턴을 생성합니다
// @Tags security
// @Accept json
// @Produce json
// @Param pattern body AttackPatternRequest true "공격 패턴 정보"
// @Success 201 {object} AttackPatternResponse "생성된 공격 패턴"
// @Failure 400 {object} map[string]interface{} "잘못된 요청"
// @Failure 500 {object} map[string]interface{} "서버 오류"
// @Router /api/v1/security/attack-patterns [post]
func (sc *SecurityController) CreateAttackPattern(c *gin.Context) {
	var req AttackPatternRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "잘못된 요청 형식입니다",
			"details": err.Error(),
		})
		return
	}

	// 공격 패턴 생성
	pattern := &security.AttackPattern{
		ID:          generateID("pattern"),
		Name:        req.Name,
		Type:        req.Type,
		Pattern:     req.Pattern,
		Severity:    security.Severity(req.Severity),
		Confidence:  req.Confidence,
		Description: req.Description,
		IsActive:    req.IsActive,
	}

	err := sc.attackDetector.AddPattern(pattern)
	if err != nil {
		sc.logger.Error("공격 패턴 생성 실패", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "공격 패턴 생성에 실패했습니다",
		})
		return
	}

	response := convertToAttackPatternResponse(pattern)
	c.JSON(http.StatusCreated, response)
}

// UpdateAttackPattern은 공격 패턴을 업데이트합니다.
// @Summary 공격 패턴 업데이트
// @Description 기존 공격 패턴을 업데이트합니다
// @Tags security
// @Accept json
// @Produce json
// @Param id path string true "패턴 ID"
// @Param pattern body AttackPatternRequest true "업데이트할 패턴 정보"
// @Success 200 {object} map[string]interface{} "성공 메시지"
// @Failure 400 {object} map[string]interface{} "잘못된 요청"
// @Failure 404 {object} map[string]interface{} "패턴을 찾을 수 없음"
// @Failure 500 {object} map[string]interface{} "서버 오류"
// @Router /api/v1/security/attack-patterns/{id} [put]
func (sc *SecurityController) UpdateAttackPattern(c *gin.Context) {
	patternID := c.Param("id")
	
	var req AttackPatternRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "잘못된 요청 형식입니다",
			"details": err.Error(),
		})
		return
	}

	// 업데이트할 패턴 정보
	updates := &security.AttackPattern{
		Name:        req.Name,
		Pattern:     req.Pattern,
		Severity:    security.Severity(req.Severity),
		Confidence:  req.Confidence,
		Description: req.Description,
	}

	err := sc.attackDetector.UpdatePattern(patternID, updates)
	if err != nil {
		sc.logger.Error("공격 패턴 업데이트 실패", 
			zap.String("pattern_id", patternID), 
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "공격 패턴 업데이트에 실패했습니다",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "공격 패턴이 업데이트되었습니다",
	})
}

// DeleteAttackPattern은 공격 패턴을 삭제합니다.
// @Summary 공격 패턴 삭제
// @Description 공격 패턴을 삭제합니다
// @Tags security
// @Accept json
// @Produce json
// @Param id path string true "패턴 ID"
// @Success 200 {object} map[string]interface{} "성공 메시지"
// @Failure 404 {object} map[string]interface{} "패턴을 찾을 수 없음"
// @Failure 500 {object} map[string]interface{} "서버 오류"
// @Router /api/v1/security/attack-patterns/{id} [delete]
func (sc *SecurityController) DeleteAttackPattern(c *gin.Context) {
	patternID := c.Param("id")

	err := sc.attackDetector.RemovePattern(patternID)
	if err != nil {
		sc.logger.Error("공격 패턴 삭제 실패", 
			zap.String("pattern_id", patternID), 
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "공격 패턴 삭제에 실패했습니다",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "공격 패턴이 삭제되었습니다",
	})
}

// 헬퍼 함수들

func parseEventTypes(types string) []security.EventType {
	var result []security.EventType
	for _, t := range split(types) {
		result = append(result, security.EventType(t))
	}
	return result
}

func parseSeverities(severities string) []security.Severity {
	var result []security.Severity
	for _, s := range split(severities) {
		result = append(result, security.Severity(s))
	}
	return result
}

func split(s string) []string {
	var result []string
	for _, item := range strings.Split(s, ",") {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func convertToSecurityEventResponse(event *security.SecurityEvent) *SecurityEventResponse {
	return &SecurityEventResponse{
		ID:          event.ID,
		Type:        string(event.Type),
		Severity:    string(event.Severity),
		Source:      event.Source,
		Target:      event.Target,
		Details:     event.Details,
		Timestamp:   event.Timestamp,
		UserID:      event.UserID,
		SessionID:   event.SessionID,
		IPAddress:   event.IPAddress,
		UserAgent:   event.UserAgent,
		RequestPath: event.RequestPath,
		Method:      event.Method,
		StatusCode:  event.StatusCode,
		Resolved:    event.Resolved,
		ResolvedAt:  event.ResolvedAt,
	}
}

func convertToAttackPatternResponse(pattern *security.AttackPattern) *AttackPatternResponse {
	return &AttackPatternResponse{
		ID:          pattern.ID,
		Name:        pattern.Name,
		Type:        pattern.Type,
		Pattern:     pattern.Pattern,
		Severity:    string(pattern.Severity),
		Confidence:  pattern.Confidence,
		Description: pattern.Description,
		IsActive:    pattern.IsActive,
		CreatedAt:   pattern.CreatedAt,
		UpdatedAt:   pattern.UpdatedAt,
	}
}

func generateID(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}