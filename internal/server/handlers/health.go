package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/aicli/aicli-web/pkg/version"
)

// HealthResponse는 헬스체크 응답 구조체입니다.
type HealthResponse struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Version   string            `json:"version"`
	Uptime    string            `json:"uptime"`
	Checks    map[string]string `json:"checks"`
}

var startTime = time.Now()

// HealthCheck는 서버의 상태를 확인하는 엔드포인트입니다.
func HealthCheck(c *gin.Context) {
	uptime := time.Since(startTime)

	// 의존성 상태 체크 (추후 확장)
	checks := map[string]string{
		"api":    "healthy",
		"memory": "healthy",
		// TODO: 데이터베이스, 외부 서비스 체크 추가
	}

	// 모든 체크가 통과했는지 확인
	status := "healthy"
	for _, check := range checks {
		if check != "healthy" {
			status = "unhealthy"
			break
		}
	}

	response := HealthResponse{
		Status:    status,
		Timestamp: time.Now(),
		Version:   version.Version,
		Uptime:    uptime.String(),
		Checks:    checks,
	}

	// 상태에 따른 HTTP 상태 코드 설정
	if status == "healthy" {
		c.JSON(http.StatusOK, response)
	} else {
		c.JSON(http.StatusServiceUnavailable, response)
	}
}