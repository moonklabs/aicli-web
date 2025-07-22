package controllers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/aicli/aicli-web/internal/auth"
	"github.com/aicli/aicli-web/internal/session"
)

// AdvancedSessionController는 고급 세션 관리 컨트롤러입니다.
type AdvancedSessionController struct {
	sessionManager *auth.SessionManager
	auditLogger    *session.AuditLogger
	cleanupService *session.CleanupService
}

// NewAdvancedSessionController는 새로운 고급 세션 컨트롤러를 생성합니다.
func NewAdvancedSessionController(sessionManager *auth.SessionManager, auditLogger *session.AuditLogger, cleanupService *session.CleanupService) *AdvancedSessionController {
	return &AdvancedSessionController{
		sessionManager: sessionManager,
		auditLogger:    auditLogger,
		cleanupService: cleanupService,
	}
}

// GetUserSessions는 사용자의 모든 활성 세션을 조회합니다.
// @Summary 사용자 세션 목록 조회
// @Tags Session Management
// @Accept json
// @Produce json
// @Param user_id path string true "User ID"
// @Success 200 {object} SessionListResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/sessions/users/{user_id} [get]
func (c *AdvancedSessionController) GetUserSessions(ctx *gin.Context) {
	userID := ctx.Param("user_id")
	if userID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "사용자 ID가 필요합니다"})
		return
	}
	
	sessions, err := c.sessionManager.GetUserSessions(ctx, userID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "세션 조회 실패: " + err.Error()})
		return
	}
	
	response := SessionListResponse{
		UserID:      userID,
		Sessions:    sessions,
		TotalCount:  len(sessions),
		RetrievedAt: time.Now(),
	}
	
	ctx.JSON(http.StatusOK, response)
}

// TerminateSession은 특정 세션을 강제 종료합니다.
// @Summary 세션 강제 종료
// @Tags Session Management
// @Accept json
// @Produce json
// @Param session_id path string true "Session ID"
// @Param request body TerminateSessionRequest true "종료 사유"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/sessions/{session_id}/terminate [post]
func (c *AdvancedSessionController) TerminateSession(ctx *gin.Context) {
	sessionID := ctx.Param("session_id")
	if sessionID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "세션 ID가 필요합니다"})
		return
	}
	
	var req TerminateSessionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "잘못된 요청 형식: " + err.Error()})
		return
	}
	
	// 세션 종료
	if err := c.sessionManager.TerminateSession(ctx, sessionID, req.Reason); err != nil {
		if err == session.ErrSessionNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "세션을 찾을 수 없습니다"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "세션 종료 실패: " + err.Error()})
		return
	}
	
	// 관리자 작업으로 감사 로그 기록
	if c.auditLogger != nil {
		adminID := ctx.GetString("user_id") // JWT에서 추출
		c.auditLogger.LogAdminAction(ctx, adminID, "terminate_session", "", 
			"관리자에 의한 세션 강제 종료", map[string]interface{}{
				"session_id": sessionID,
				"reason": req.Reason,
			})
	}
	
	ctx.JSON(http.StatusOK, gin.H{
		"message": "세션이 성공적으로 종료되었습니다",
		"session_id": sessionID,
	})
}

// TerminateUserSessions는 특정 사용자의 모든 세션을 종료합니다.
// @Summary 사용자 모든 세션 종료
// @Tags Session Management
// @Accept json
// @Produce json
// @Param user_id path string true "User ID"
// @Param request body TerminateSessionRequest true "종료 사유"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/sessions/users/{user_id}/terminate-all [post]
func (c *AdvancedSessionController) TerminateUserSessions(ctx *gin.Context) {
	userID := ctx.Param("user_id")
	if userID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "사용자 ID가 필요합니다"})
		return
	}
	
	var req TerminateSessionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "잘못된 요청 형식: " + err.Error()})
		return
	}
	
	// 사용자의 모든 세션 종료
	if err := c.sessionManager.TerminateUserSessions(ctx, userID, req.Reason); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "세션 종료 실패: " + err.Error()})
		return
	}
	
	// 관리자 작업으로 감사 로그 기록
	if c.auditLogger != nil {
		adminID := ctx.GetString("user_id")
		c.auditLogger.LogAdminAction(ctx, adminID, "terminate_user_sessions", userID,
			"관리자에 의한 사용자 모든 세션 종료", map[string]interface{}{
				"target_user_id": userID,
				"reason": req.Reason,
			})
	}
	
	ctx.JSON(http.StatusOK, gin.H{
		"message": "사용자의 모든 세션이 종료되었습니다",
		"user_id": userID,
	})
}

// GetSessionMetrics는 세션 메트릭을 반환합니다.
// @Summary 세션 통계 조회
// @Tags Session Management
// @Accept json
// @Produce json
// @Success 200 {object} session.SessionMetrics
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/sessions/metrics [get]
func (c *AdvancedSessionController) GetSessionMetrics(ctx *gin.Context) {
	metrics, err := c.sessionManager.GetSessionStats(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "세션 통계 조회 실패: " + err.Error()})
		return
	}
	
	ctx.JSON(http.StatusOK, metrics)
}

// GetAuditLogs는 감사 로그를 조회합니다.
// @Summary 감사 로그 조회
// @Tags Session Management
// @Accept json
// @Produce json
// @Param user_id query string false "User ID"
// @Param category query string false "Category (session, security, admin)"
// @Param action query string false "Action"
// @Param severity query string false "Severity (info, warning, critical)"
// @Param start_date query string false "Start Date (YYYY-MM-DD)"
// @Param end_date query string false "End Date (YYYY-MM-DD)"
// @Param limit query int false "Limit" default(100)
// @Success 200 {object} AuditLogListResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/sessions/audit-logs [get]
func (c *AdvancedSessionController) GetAuditLogs(ctx *gin.Context) {
	filter := &session.AuditFilter{
		UserID:    ctx.Query("user_id"),
		Category:  ctx.Query("category"),
		Action:    ctx.Query("action"),
		Severity:  ctx.Query("severity"),
		StartDate: ctx.Query("start_date"),
		EndDate:   ctx.Query("end_date"),
		Limit:     100, // 기본값
	}
	
	// Limit 파라미터 파싱
	if limitStr := ctx.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			filter.Limit = limit
		}
	}
	
	logs, err := c.auditLogger.GetAuditLogs(ctx, filter)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "감사 로그 조회 실패: " + err.Error()})
		return
	}
	
	response := AuditLogListResponse{
		Logs:        logs,
		TotalCount:  len(logs),
		Filter:      filter,
		RetrievedAt: time.Now(),
	}
	
	ctx.JSON(http.StatusOK, response)
}

// GetCleanupStats는 정리 통계를 반환합니다.
// @Summary 세션 정리 통계 조회
// @Tags Session Management
// @Accept json
// @Produce json
// @Success 200 {object} session.CleanupStats
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/sessions/cleanup/stats [get]
func (c *AdvancedSessionController) GetCleanupStats(ctx *gin.Context) {
	if c.cleanupService == nil {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{"error": "정리 서비스가 비활성화되어 있습니다"})
		return
	}
	
	stats, err := c.cleanupService.GetCleanupStats(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "정리 통계 조회 실패: " + err.Error()})
		return
	}
	
	ctx.JSON(http.StatusOK, stats)
}

// ForceCleanup은 즉시 세션 정리를 실행합니다.
// @Summary 즉시 세션 정리 실행
// @Tags Session Management
// @Accept json
// @Produce json
// @Success 200 {object} SuccessResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/sessions/cleanup/force [post]
func (c *AdvancedSessionController) ForceCleanup(ctx *gin.Context) {
	if c.cleanupService == nil {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{"error": "정리 서비스가 비활성화되어 있습니다"})
		return
	}
	
	if err := c.cleanupService.ForceCleanup(ctx); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "정리 실행 실패: " + err.Error()})
		return
	}
	
	// 관리자 작업으로 감사 로그 기록
	if c.auditLogger != nil {
		adminID := ctx.GetString("user_id")
		c.auditLogger.LogAdminAction(ctx, adminID, "force_cleanup", "",
			"관리자에 의한 즉시 세션 정리 실행", nil)
	}
	
	ctx.JSON(http.StatusOK, gin.H{
		"message": "세션 정리가 실행되었습니다",
		"timestamp": time.Now(),
	})
}

// 요청/응답 구조체들

type TerminateSessionRequest struct {
	Reason string `json:"reason" binding:"required"`
}

type SessionListResponse struct {
	UserID      string      `json:"user_id"`
	Sessions    interface{} `json:"sessions"`
	TotalCount  int         `json:"total_count"`
	RetrievedAt time.Time   `json:"retrieved_at"`
}

type AuditLogListResponse struct {
	Logs        interface{} `json:"logs"`
	TotalCount  int         `json:"total_count"`
	Filter      interface{} `json:"filter"`
	RetrievedAt time.Time   `json:"retrieved_at"`
}

type SuccessResponse struct {
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

type ErrorResponse struct {
	Error     string    `json:"error"`
	Timestamp time.Time `json:"timestamp"`
}