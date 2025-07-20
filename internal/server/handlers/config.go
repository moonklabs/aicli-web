package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

// ConfigResponse는 설정 응답 구조체입니다.
type ConfigResponse struct {
	Port        string `json:"port"`
	Environment string `json:"environment"`
	LogLevel    string `json:"log_level"`
	Version     string `json:"version"`
}

// UpdateConfigRequest는 설정 업데이트 요청 구조체입니다.
type UpdateConfigRequest struct {
	LogLevel string `json:"log_level,omitempty"`
	// 다른 업데이트 가능한 설정들
}

// GetConfig는 현재 설정을 조회합니다.
func GetConfig(c *gin.Context) {
	config := ConfigResponse{
		Port:        viper.GetString("port"),
		Environment: viper.GetString("env"),
		LogLevel:    viper.GetString("log_level"),
		Version:     viper.GetString("version"),
	}

	// 민감한 정보는 제외하고 응답
	c.JSON(http.StatusOK, gin.H{
		"config": config,
		"note":   "민감한 설정 정보는 표시되지 않습니다",
	})
}

// UpdateConfig는 설정을 업데이트합니다.
func UpdateConfig(c *gin.Context) {
	var req UpdateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"message": err.Error(),
		})
		return
	}

	// 업데이트 가능한 설정들만 처리
	updated := make(map[string]string)

	if req.LogLevel != "" {
		// 유효한 로그 레벨인지 확인
		validLevels := []string{"debug", "info", "warn", "error"}
		isValid := false
		for _, level := range validLevels {
			if req.LogLevel == level {
				isValid = true
				break
			}
		}

		if !isValid {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid log level",
				"message": "로그 레벨은 debug, info, warn, error 중 하나여야 합니다",
			})
			return
		}

		viper.Set("log_level", req.LogLevel)
		updated["log_level"] = req.LogLevel
	}

	if len(updated) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "No updates provided",
			"message": "업데이트할 설정이 없습니다",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "설정이 업데이트되었습니다",
		"updated": updated,
		"note":    "일부 설정은 서버 재시작 후 적용됩니다",
	})
}