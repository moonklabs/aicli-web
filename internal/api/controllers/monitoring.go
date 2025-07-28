package controllers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"your-project/internal/monitoring"
	"your-project/internal/middleware"
	"your-project/internal/testing"
)

// MonitoringController는 모니터링 관련 API를 처리합니다
type MonitoringController struct {
	performanceMiddleware *middleware.PerformanceMonitoringMiddleware
	errorTracker         *monitoring.ErrorTracker
	alertingSystem       *monitoring.AlertingSystem
	testSuite           *testing.PerformanceTestSuite
}

// NewMonitoringController는 새로운 모니터링 컨트롤러를 생성합니다
func NewMonitoringController(
	perfMiddleware *middleware.PerformanceMonitoringMiddleware,
	errorTracker *monitoring.ErrorTracker,
	alertingSystem *monitoring.AlertingSystem,
	testSuite *testing.PerformanceTestSuite,
) *MonitoringController {
	return &MonitoringController{
		performanceMiddleware: perfMiddleware,
		errorTracker:         errorTracker,
		alertingSystem:       alertingSystem,
		testSuite:           testSuite,
	}
}

// GetPerformanceMetrics는 성능 메트릭을 반환합니다
// @Summary Get performance metrics
// @Description 실시간 성능 메트릭을 반환합니다
// @Tags monitoring
// @Accept json
// @Produce json
// @Success 200 {object} middleware.PerformanceMetrics
// @Failure 500 {object} ErrorResponse
// @Router /api/monitoring/performance [get]
func (mc *MonitoringController) GetPerformanceMetrics(c *gin.Context) {
	metrics := mc.performanceMiddleware.GetMetrics()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    metrics,
	})
}

// GetPerformanceScore는 성능 점수를 반환합니다
// @Summary Get performance score
// @Description 전체 성능 점수를 계산하여 반환합니다
// @Tags monitoring
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/monitoring/performance/score [get]
func (mc *MonitoringController) GetPerformanceScore(c *gin.Context) {
	score := mc.performanceMiddleware.CalculatePerformanceScore()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"score": score,
			"grade": getPerformanceGrade(score),
			"timestamp": time.Now(),
		},
	})
}

// GetTopEndpoints는 가장 많이 사용되는 엔드포인트들을 반환합니다
// @Summary Get top endpoints
// @Description 요청 수가 많은 엔드포인트들을 반환합니다
// @Tags monitoring
// @Accept json
// @Produce json
// @Param limit query int false "Number of endpoints to return" default(10)
// @Success 200 {array} middleware.EndpointMetric
// @Router /api/monitoring/endpoints/top [get]
func (mc *MonitoringController) GetTopEndpoints(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 10
	}

	endpoints := mc.performanceMiddleware.GetTopEndpoints(limit)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    endpoints,
	})
}

// GetSlowEndpoints는 느린 엔드포인트들을 반환합니다
// @Summary Get slow endpoints
// @Description 응답 시간이 느린 엔드포인트들을 반환합니다
// @Tags monitoring
// @Accept json
// @Produce json
// @Param limit query int false "Number of endpoints to return" default(10)
// @Success 200 {array} middleware.EndpointMetric
// @Router /api/monitoring/endpoints/slow [get]
func (mc *MonitoringController) GetSlowEndpoints(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 10
	}

	endpoints := mc.performanceMiddleware.GetSlowEndpoints(limit)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    endpoints,
	})
}

// ResetMetrics는 성능 메트릭을 초기화합니다
// @Summary Reset performance metrics
// @Description 모든 성능 메트릭을 초기화합니다
// @Tags monitoring
// @Accept json
// @Produce json
// @Success 200 {object} SuccessResponse
// @Router /api/monitoring/performance/reset [post]
func (mc *MonitoringController) ResetMetrics(c *gin.Context) {
	mc.performanceMiddleware.Reset()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Performance metrics reset successfully",
	})
}

// Error Tracking APIs

// GetErrors는 에러 목록을 반환합니다
// @Summary Get all errors
// @Description 모든 에러들을 반환합니다
// @Tags monitoring
// @Accept json
// @Produce json
// @Success 200 {array} monitoring.ErrorInfo
// @Router /api/monitoring/errors [get]
func (mc *MonitoringController) GetErrors(c *gin.Context) {
	errors := mc.errorTracker.GetAllErrors()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    errors,
	})
}

// GetError는 특정 에러를 반환합니다
// @Summary Get specific error
// @Description 에러 ID로 특정 에러를 반환합니다
// @Tags monitoring
// @Accept json
// @Produce json
// @Param id path string true "Error ID"
// @Success 200 {object} monitoring.ErrorInfo
// @Failure 404 {object} ErrorResponse
// @Router /api/monitoring/errors/{id} [get]
func (mc *MonitoringController) GetError(c *gin.Context) {
	errorID := c.Param("id")
	errorInfo, exists := mc.errorTracker.GetError(errorID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Error not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    errorInfo,
	})
}

// GetTopErrors는 가장 빈번한 에러들을 반환합니다
// @Summary Get top errors
// @Description 발생 빈도가 높은 에러들을 반환합니다
// @Tags monitoring
// @Accept json
// @Produce json
// @Param limit query int false "Number of errors to return" default(10)
// @Success 200 {array} monitoring.ErrorInfo
// @Router /api/monitoring/errors/top [get]
func (mc *MonitoringController) GetTopErrors(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 10
	}

	errors := mc.errorTracker.GetTopErrors(limit)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    errors,
	})
}

// GetErrorSummary는 에러 요약을 반환합니다
// @Summary Get error summary
// @Description 에러 통계 요약을 반환합니다
// @Tags monitoring
// @Accept json
// @Produce json
// @Success 200 {object} monitoring.ErrorSummary
// @Router /api/monitoring/errors/summary [get]
func (mc *MonitoringController) GetErrorSummary(c *gin.Context) {
	summary := mc.errorTracker.GetSummary()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    summary,
	})
}

// ResolveError는 에러를 해결됨으로 표시합니다
// @Summary Resolve error
// @Description 에러를 해결됨으로 표시합니다
// @Tags monitoring
// @Accept json
// @Produce json
// @Param id path string true "Error ID"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Router /api/monitoring/errors/{id}/resolve [post]
func (mc *MonitoringController) ResolveError(c *gin.Context) {
	errorID := c.Param("id")
	err := mc.errorTracker.ResolveError(errorID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Error resolved successfully",
	})
}

// IgnoreError는 에러를 무시됨으로 표시합니다
// @Summary Ignore error
// @Description 에러를 무시됨으로 표시합니다
// @Tags monitoring
// @Accept json
// @Produce json
// @Param id path string true "Error ID"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Router /api/monitoring/errors/{id}/ignore [post]
func (mc *MonitoringController) IgnoreError(c *gin.Context) {
	errorID := c.Param("id")
	err := mc.errorTracker.IgnoreError(errorID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Error ignored successfully",
	})
}

// ClearErrors는 모든 에러를 지웁니다
// @Summary Clear all errors
// @Description 모든 에러를 지웁니다
// @Tags monitoring
// @Accept json
// @Produce json
// @Success 200 {object} SuccessResponse
// @Router /api/monitoring/errors/clear [post]
func (mc *MonitoringController) ClearErrors(c *gin.Context) {
	mc.errorTracker.ClearErrors()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "All errors cleared successfully",
	})
}

// Alerting APIs

// GetAlerts는 알림 목록을 반환합니다
// @Summary Get all alerts
// @Description 모든 알림들을 반환합니다
// @Tags monitoring
// @Accept json
// @Produce json
// @Param active query bool false "Only active alerts"
// @Success 200 {array} monitoring.Alert
// @Router /api/monitoring/alerts [get]
func (mc *MonitoringController) GetAlerts(c *gin.Context) {
	activeOnly := c.Query("active") == "true"

	var alerts []*monitoring.Alert
	if activeOnly {
		alerts = mc.alertingSystem.GetActiveAlerts()
	} else {
		alerts = mc.alertingSystem.GetAllAlerts()
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    alerts,
	})
}

// GetAlertingMetrics는 알림 시스템 메트릭을 반환합니다
// @Summary Get alerting metrics
// @Description 알림 시스템의 메트릭을 반환합니다
// @Tags monitoring
// @Accept json
// @Produce json
// @Success 200 {object} monitoring.AlertingMetrics
// @Router /api/monitoring/alerts/metrics [get]
func (mc *MonitoringController) GetAlertingMetrics(c *gin.Context) {
	metrics := mc.alertingSystem.GetMetrics()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    metrics,
	})
}

// SilenceAlert는 알림을 음소거합니다
// @Summary Silence alert
// @Description 알림을 음소거합니다
// @Tags monitoring
// @Accept json
// @Produce json
// @Param id path string true "Alert ID"
// @Success 200 {object} SuccessResponse
// @Router /api/monitoring/alerts/{id}/silence [post]
func (mc *MonitoringController) SilenceAlert(c *gin.Context) {
	alertID := c.Param("id")
	mc.alertingSystem.SilenceAlert(alertID)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Alert silenced successfully",
	})
}

// ResolveAlert는 알림을 해결됨으로 표시합니다
// @Summary Resolve alert
// @Description 알림을 해결됨으로 표시합니다
// @Tags monitoring
// @Accept json
// @Produce json
// @Param id path string true "Alert ID"
// @Success 200 {object} SuccessResponse
// @Router /api/monitoring/alerts/{id}/resolve [post]
func (mc *MonitoringController) ResolveAlert(c *gin.Context) {
	alertID := c.Param("id")
	mc.alertingSystem.ResolveAlert(alertID)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Alert resolved successfully",
	})
}

// Load Testing APIs

// RunLoadTest는 부하 테스트를 실행합니다
// @Summary Run load test
// @Description 부하 테스트를 실행합니다
// @Tags monitoring
// @Accept json
// @Produce json
// @Success 200 {object} testing.LoadTestReport
// @Failure 500 {object} ErrorResponse
// @Router /api/monitoring/load-test [post]
func (mc *MonitoringController) RunLoadTest(c *gin.Context) {
	report, err := mc.testSuite.RunLoadTest()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    report,
	})
}

// GetTestResults는 테스트 결과를 반환합니다
// @Summary Get test results
// @Description 최근 테스트 결과를 반환합니다
// @Tags monitoring
// @Accept json
// @Produce json
// @Success 200 {array} testing.TestResult
// @Router /api/monitoring/test-results [get]
func (mc *MonitoringController) GetTestResults(c *gin.Context) {
	results := mc.testSuite.GetResults()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    results,
	})
}

// ClearTestResults는 테스트 결과를 지웁니다
// @Summary Clear test results
// @Description 모든 테스트 결과를 지웁니다
// @Tags monitoring
// @Accept json
// @Produce json
// @Success 200 {object} SuccessResponse
// @Router /api/monitoring/test-results/clear [post]
func (mc *MonitoringController) ClearTestResults(c *gin.Context) {
	mc.testSuite.ClearResults()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Test results cleared successfully",
	})
}

// StopTest는 실행 중인 테스트를 중지합니다
// @Summary Stop running test
// @Description 실행 중인 테스트를 중지합니다
// @Tags monitoring
// @Accept json
// @Produce json
// @Success 200 {object} SuccessResponse
// @Router /api/monitoring/test/stop [post]
func (mc *MonitoringController) StopTest(c *gin.Context) {
	mc.testSuite.Stop()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Test stopped successfully",
	})
}

// 라우터 설정

// RegisterRoutes는 모니터링 라우트를 등록합니다
func (mc *MonitoringController) RegisterRoutes(r *gin.RouterGroup) {
	monitoring := r.Group("/monitoring")
	{
		// Performance monitoring
		perf := monitoring.Group("/performance")
		{
			perf.GET("", mc.GetPerformanceMetrics)
			perf.GET("/score", mc.GetPerformanceScore)
			perf.POST("/reset", mc.ResetMetrics)
		}

		// Endpoint monitoring
		endpoints := monitoring.Group("/endpoints")
		{
			endpoints.GET("/top", mc.GetTopEndpoints)
			endpoints.GET("/slow", mc.GetSlowEndpoints)
		}

		// Error tracking
		errors := monitoring.Group("/errors")
		{
			errors.GET("", mc.GetErrors)
			errors.GET("/top", mc.GetTopErrors)
			errors.GET("/summary", mc.GetErrorSummary)
			errors.GET("/:id", mc.GetError)
			errors.POST("/:id/resolve", mc.ResolveError)
			errors.POST("/:id/ignore", mc.IgnoreError)
			errors.POST("/clear", mc.ClearErrors)
		}

		// Alerting
		alerts := monitoring.Group("/alerts")
		{
			alerts.GET("", mc.GetAlerts)
			alerts.GET("/metrics", mc.GetAlertingMetrics)
			alerts.POST("/:id/silence", mc.SilenceAlert)
			alerts.POST("/:id/resolve", mc.ResolveAlert)
		}

		// Load testing
		testing := monitoring.Group("/test")
		{
			testing.POST("/load", mc.RunLoadTest)
			testing.GET("/results", mc.GetTestResults)
			testing.POST("/results/clear", mc.ClearTestResults)
			testing.POST("/stop", mc.StopTest)
		}
	}
}

// 도우미 함수들

func getPerformanceGrade(score float64) string {
	if score >= 90 {
		return "A"
	} else if score >= 80 {
		return "B"
	} else if score >= 70 {
		return "C"
	} else if score >= 60 {
		return "D"
	}
	return "F"
}

// 응답 타입들

type ErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

type SuccessResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}
