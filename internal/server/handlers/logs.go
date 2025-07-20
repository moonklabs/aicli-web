package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// LogEntry는 로그 엔트리 구조체입니다.
type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Message   string `json:"message"`
	Source    string `json:"source"`
}

// LogsResponse는 로그 응답 구조체입니다.
type LogsResponse struct {
	Logs  []LogEntry `json:"logs"`
	Total int        `json:"total"`
	Since string     `json:"since,omitempty"`
	Until string     `json:"until,omitempty"`
}

// GetWorkspaceLogs는 특정 워크스페이스의 로그를 조회합니다.
func GetWorkspaceLogs(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid workspace ID",
			"message": "워크스페이스 ID는 숫자여야 합니다",
		})
		return
	}

	// 쿼리 파라미터 처리
	since := c.Query("since")
	tail := c.Query("tail")
	follow := c.Query("follow") == "true"

	// TODO: 실제 로그 데이터베이스에서 조회
	logs := []LogEntry{
		{
			Timestamp: "2025-01-20T10:00:00Z",
			Level:     "info",
			Message:   "워크스페이스 초기화됨",
			Source:    "workspace",
		},
		{
			Timestamp: "2025-01-20T10:00:05Z",
			Level:     "debug",
			Message:   "Claude CLI 컨테이너 준비 중",
			Source:    "docker",
		},
	}

	if follow {
		// TODO: WebSocket으로 실시간 로그 스트리밍 구현
		c.JSON(http.StatusNotImplemented, gin.H{
			"error":   "Not implemented",
			"message": "실시간 로그 스트리밍은 아직 구현되지 않았습니다",
		})
		return
	}

	response := LogsResponse{
		Logs:  logs,
		Total: len(logs),
		Since: since,
	}

	// tail 파라미터 처리
	if tail != "" {
		if tailNum, err := strconv.Atoi(tail); err == nil && tailNum > 0 {
			if tailNum < len(logs) {
				response.Logs = logs[len(logs)-tailNum:]
			}
		}
	}

	c.JSON(http.StatusOK, response)
}

// GetTaskLogs는 특정 태스크의 로그를 조회합니다.
func GetTaskLogs(c *gin.Context) {
	taskID := c.Param("id")

	// 쿼리 파라미터 처리
	since := c.Query("since")
	tail := c.Query("tail")
	follow := c.Query("follow") == "true"

	// TODO: 실제 로그 데이터베이스에서 조회
	logs := []LogEntry{
		{
			Timestamp: "2025-01-20T10:00:01Z",
			Level:     "info",
			Message:   "태스크 시작: " + taskID,
			Source:    "task",
		},
		{
			Timestamp: "2025-01-20T10:00:02Z",
			Level:     "debug",
			Message:   "Claude CLI 명령어 실행 중",
			Source:    "claude",
		},
		{
			Timestamp: "2025-01-20T10:05:00Z",
			Level:     "info",
			Message:   "태스크 완료",
			Source:    "task",
		},
	}

	if follow {
		// TODO: WebSocket으로 실시간 로그 스트리밍 구현
		c.JSON(http.StatusNotImplemented, gin.H{
			"error":   "Not implemented",
			"message": "실시간 로그 스트리밍은 아직 구현되지 않았습니다",
		})
		return
	}

	response := LogsResponse{
		Logs:  logs,
		Total: len(logs),
		Since: since,
	}

	// tail 파라미터 처리
	if tail != "" {
		if tailNum, err := strconv.Atoi(tail); err == nil && tailNum > 0 {
			if tailNum < len(logs) {
				response.Logs = logs[len(logs)-tailNum:]
			}
		}
	}

	c.JSON(http.StatusOK, response)
}