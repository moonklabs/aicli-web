package handlers

import (
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/drumcap/aicli-web/pkg/version"
)

// SystemInfo는 시스템 정보 응답 구조체입니다.
type SystemInfo struct {
	Service     string            `json:"service"`
	Version     string            `json:"version"`
	BuildTime   string            `json:"build_time"`
	GitCommit   string            `json:"git_commit"`
	GoVersion   string            `json:"go_version"`
	Platform    string            `json:"platform"`
	Runtime     RuntimeInfo       `json:"runtime"`
	Uptime      string            `json:"uptime"`
	Environment string            `json:"environment"`
	Features    map[string]bool   `json:"features"`
}

// RuntimeInfo는 런타임 정보 구조체입니다.
type RuntimeInfo struct {
	NumGoroutines int     `json:"num_goroutines"`
	NumCPU        int     `json:"num_cpu"`
	AllocMB       float64 `json:"alloc_mb"`
	TotalAllocMB  float64 `json:"total_alloc_mb"`
	SysMB         float64 `json:"sys_mb"`
	NumGC         uint32  `json:"num_gc"`
}

var (
	startTime = time.Now()
)

// GetSystemInfo는 시스템 정보를 반환합니다.
// @Summary 시스템 정보 조회
// @Description API 서버의 상세 시스템 정보를 조회합니다
// @Tags system
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "시스템 정보"
// @Router /system/info [get]
func GetSystemInfo(c *gin.Context) {
	versionInfo := version.Get()
	
	// 메모리 통계 수집
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	// 런타임 정보 구성
	runtimeInfo := RuntimeInfo{
		NumGoroutines: runtime.NumGoroutine(),
		NumCPU:        runtime.NumCPU(),
		AllocMB:       bToMb(memStats.Alloc),
		TotalAllocMB:  bToMb(memStats.TotalAlloc),
		SysMB:         bToMb(memStats.Sys),
		NumGC:         memStats.NumGC,
	}
	
	// 업타임 계산
	uptime := time.Since(startTime)
	
	// 활성화된 기능 목록
	features := map[string]bool{
		"middleware_logging":    true,
		"middleware_cors":       true,
		"middleware_security":   true,
		"middleware_recovery":   true,
		"error_handling":        true,
		"request_id_tracking":   true,
		"graceful_shutdown":     true,
		"structured_logging":    true,
		"version_info":          true,
		"health_check":          true,
	}
	
	systemInfo := SystemInfo{
		Service:     "AICode Manager API",
		Version:     versionInfo.Version,
		BuildTime:   versionInfo.BuildTime,
		GitCommit:   versionInfo.GitCommit,
		GoVersion:   versionInfo.GoVersion,
		Platform:    versionInfo.Platform,
		Runtime:     runtimeInfo,
		Uptime:      uptime.String(),
		Environment: gin.Mode(),
		Features:    features,
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    systemInfo,
	})
}

// bToMb는 바이트를 메가바이트로 변환합니다.
func bToMb(b uint64) float64 {
	return float64(b) / 1024 / 1024
}

// GetSystemStatus는 간단한 시스템 상태를 반환합니다.
// @Summary 시스템 상태 확인
// @Description API 서버의 간단한 상태 정보를 조회합니다
// @Tags system
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "시스템 상태"
// @Router /system/status [get]
func GetSystemStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"status":    "running",
			"timestamp": time.Now().Format(time.RFC3339),
			"uptime":    time.Since(startTime).String(),
		},
	})
}