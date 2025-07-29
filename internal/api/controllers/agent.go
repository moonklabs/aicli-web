package controllers

import (
	"context"
	"net/http"
	"time"

	"github.com/aicli/aicli-web/internal/agent"
	"github.com/aicli/aicli-web/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// AgentController는 에이전트 관련 API를 처리합니다.
type AgentController struct {
	agentService agent.AgentService
}

// NewAgentController는 새로운 에이전트 컨트롤러를 생성합니다.
func NewAgentController(agentService agent.AgentService) *AgentController {
	return &AgentController{
		agentService: agentService,
	}
}

// ListAgents는 에이전트 목록을 조회합니다.
// @Summary 에이전트 목록 조회
// @Description 활성 에이전트 목록을 조회합니다
// @Tags agents
// @Accept json
// @Produce json
// @Param project_id query string false "프로젝트 ID로 필터링"
// @Security BearerAuth
// @Success 200 {object} models.AgentListResponse "에이전트 목록"
// @Failure 400 {object} models.ErrorResponse "잘못된 요청"
// @Failure 401 {object} models.ErrorResponse "인증 실패"
// @Failure 500 {object} models.ErrorResponse "서버 오류"
// @Router /api/v1/agents [get]
func (ac *AgentController) ListAgents(c *gin.Context) {
	projectID := c.Query("project_id")
	
	var agents []*models.Agent
	var err error
	
	if projectID != "" {
		agents, err = ac.agentService.GetAgentByProjectID(c.Request.Context(), projectID)
	} else {
		agents, err = ac.agentService.ListActiveAgents(c.Request.Context())
	}
	
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "에이전트 목록 조회 실패",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"agents": agents,
		"count":  len(agents),
	})
}

// CreateAgent는 새로운 에이전트를 생성합니다.
// @Summary 에이전트 생성
// @Description 새로운 에이전트를 생성합니다
// @Tags agents
// @Accept json
// @Produce json
// @Param agent body agent.CreateAgentRequest true "에이전트 생성 요청"
// @Security BearerAuth
// @Success 201 {object} models.Agent "생성된 에이전트"
// @Failure 400 {object} models.ErrorResponse "잘못된 요청"
// @Failure 401 {object} models.ErrorResponse "인증 실패"
// @Failure 500 {object} models.ErrorResponse "서버 오류"
// @Router /api/v1/agents [post]
func (ac *AgentController) CreateAgent(c *gin.Context) {
	var req agent.CreateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "잘못된 요청 데이터",
			"details": err.Error(),
		})
		return
	}
	
	// 사용자 ID는 미들웨어에서 처리되므로 별도 설정 불필요
	
	newAgent, err := ac.agentService.CreateAgent(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "에이전트 생성 실패",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusCreated, newAgent)
}

// GetAgent는 특정 에이전트를 조회합니다.
// @Summary 에이전트 조회
// @Description ID로 특정 에이전트를 조회합니다
// @Tags agents
// @Accept json
// @Produce json
// @Param id path string true "에이전트 ID"
// @Security BearerAuth
// @Success 200 {object} models.Agent "에이전트 정보"
// @Failure 400 {object} models.ErrorResponse "잘못된 요청"
// @Failure 401 {object} models.ErrorResponse "인증 실패"
// @Failure 404 {object} models.ErrorResponse "에이전트 없음"
// @Failure 500 {object} models.ErrorResponse "서버 오류"
// @Router /api/v1/agents/{id} [get]
func (ac *AgentController) GetAgent(c *gin.Context) {
	agentID := c.Param("id")
	
	agentData, err := ac.agentService.GetAgent(c.Request.Context(), agentID)
	if err != nil {
		if err.Error() == "agent not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "에이전트를 찾을 수 없습니다",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "에이전트 조회 실패",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, agentData)
}

// UpdateAgent는 에이전트를 수정합니다.
// @Summary 에이전트 수정
// @Description 특정 에이전트의 정보를 수정합니다
// @Tags agents
// @Accept json
// @Produce json
// @Param id path string true "에이전트 ID"
// @Param agent body agent.UpdateAgentRequest true "에이전트 수정 요청"
// @Security BearerAuth
// @Success 200 {object} models.Agent "수정된 에이전트"
// @Failure 400 {object} models.ErrorResponse "잘못된 요청"
// @Failure 401 {object} models.ErrorResponse "인증 실패"
// @Failure 404 {object} models.ErrorResponse "에이전트 없음"
// @Failure 500 {object} models.ErrorResponse "서버 오류"
// @Router /api/v1/agents/{id} [put]
func (ac *AgentController) UpdateAgent(c *gin.Context) {
	agentID := c.Param("id")
	
	var req agent.UpdateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "잘못된 요청 데이터",
			"details": err.Error(),
		})
		return
	}
	
	updatedAgent, err := ac.agentService.UpdateAgent(c.Request.Context(), agentID, req)
	if err != nil {
		if err.Error() == "agent not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "에이전트를 찾을 수 없습니다",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "에이전트 수정 실패",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, updatedAgent)
}

// DeleteAgent는 에이전트를 삭제합니다.
// @Summary 에이전트 삭제
// @Description 특정 에이전트를 삭제합니다
// @Tags agents
// @Accept json
// @Produce json
// @Param id path string true "에이전트 ID"
// @Security BearerAuth
// @Success 204 "삭제 완료"
// @Failure 401 {object} models.ErrorResponse "인증 실패"
// @Failure 404 {object} models.ErrorResponse "에이전트 없음"
// @Failure 500 {object} models.ErrorResponse "서버 오류"
// @Router /api/v1/agents/{id} [delete]
func (ac *AgentController) DeleteAgent(c *gin.Context) {
	agentID := c.Param("id")
	
	err := ac.agentService.DeleteAgent(c.Request.Context(), agentID)
	if err != nil {
		if err.Error() == "agent not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "에이전트를 찾을 수 없습니다",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "에이전트 삭제 실패",
			"details": err.Error(),
		})
		return
	}
	
	c.Status(http.StatusNoContent)
}

// StartAgent는 에이전트를 시작합니다.
// @Summary 에이전트 시작
// @Description 특정 에이전트를 시작합니다
// @Tags agents
// @Accept json
// @Produce json
// @Param id path string true "에이전트 ID"
// @Security BearerAuth
// @Success 200 {object} map[string]string "시작 결과"
// @Failure 401 {object} models.ErrorResponse "인증 실패"
// @Failure 404 {object} models.ErrorResponse "에이전트 없음"
// @Failure 500 {object} models.ErrorResponse "서버 오류"
// @Router /api/v1/agents/{id}/start [post]
func (ac *AgentController) StartAgent(c *gin.Context) {
	agentID := c.Param("id")
	
	err := ac.agentService.StartAgent(c.Request.Context(), agentID)
	if err != nil {
		if err.Error() == "agent not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "에이전트를 찾을 수 없습니다",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "에이전트 시작 실패",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message":  "에이전트가 시작되었습니다",
		"agent_id": agentID,
	})
}

// StopAgent는 에이전트를 중지합니다.
// @Summary 에이전트 중지
// @Description 특정 에이전트를 중지합니다
// @Tags agents
// @Accept json
// @Produce json
// @Param id path string true "에이전트 ID"
// @Security BearerAuth
// @Success 200 {object} map[string]string "중지 결과"
// @Failure 401 {object} models.ErrorResponse "인증 실패"
// @Failure 404 {object} models.ErrorResponse "에이전트 없음"
// @Failure 500 {object} models.ErrorResponse "서버 오류"
// @Router /api/v1/agents/{id}/stop [post]
func (ac *AgentController) StopAgent(c *gin.Context) {
	agentID := c.Param("id")
	
	err := ac.agentService.StopAgent(c.Request.Context(), agentID)
	if err != nil {
		if err.Error() == "agent not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "에이전트를 찾을 수 없습니다",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "에이전트 중지 실패",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message":  "에이전트가 중지되었습니다",
		"agent_id": agentID,
	})
}

// RestartAgent는 에이전트를 재시작합니다.
// @Summary 에이전트 재시작
// @Description 특정 에이전트를 재시작합니다
// @Tags agents
// @Accept json
// @Produce json
// @Param id path string true "에이전트 ID"
// @Security BearerAuth
// @Success 200 {object} map[string]string "재시작 결과"
// @Failure 401 {object} models.ErrorResponse "인증 실패"
// @Failure 404 {object} models.ErrorResponse "에이전트 없음"
// @Failure 500 {object} models.ErrorResponse "서버 오류"
// @Router /api/v1/agents/{id}/restart [post]
func (ac *AgentController) RestartAgent(c *gin.Context) {
	agentID := c.Param("id")
	
	err := ac.agentService.RestartAgent(c.Request.Context(), agentID)
	if err != nil {
		if err.Error() == "agent not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "에이전트를 찾을 수 없습니다",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "에이전트 재시작 실패",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message":  "에이전트가 재시작되었습니다",
		"agent_id": agentID,
	})
}

// GetAgentStatus는 에이전트 상태를 조회합니다.
// @Summary 에이전트 상태 조회
// @Description 특정 에이전트의 현재 상태를 조회합니다
// @Tags agents
// @Accept json
// @Produce json
// @Param id path string true "에이전트 ID"
// @Security BearerAuth
// @Success 200 {object} agent.AgentStatusInfo "에이전트 상태"
// @Failure 401 {object} models.ErrorResponse "인증 실패"
// @Failure 404 {object} models.ErrorResponse "에이전트 없음"
// @Failure 500 {object} models.ErrorResponse "서버 오류"
// @Router /api/v1/agents/{id}/status [get]
func (ac *AgentController) GetAgentStatus(c *gin.Context) {
	agentID := c.Param("id")
	
	status, err := ac.agentService.GetAgentStatus(c.Request.Context(), agentID)
	if err != nil {
		if err.Error() == "agent not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "에이전트를 찾을 수 없습니다",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "에이전트 상태 조회 실패",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, status)
}

// GetAgentHealth는 에이전트 헬스 상태를 조회합니다.
// @Summary 에이전트 헬스체크
// @Description 특정 에이전트의 헬스 상태를 조회합니다
// @Tags agents
// @Accept json
// @Produce json
// @Param id path string true "에이전트 ID"
// @Security BearerAuth
// @Success 200 {object} agent.HealthStatus "헬스 상태"
// @Failure 401 {object} models.ErrorResponse "인증 실패"
// @Failure 404 {object} models.ErrorResponse "에이전트 없음"
// @Failure 500 {object} models.ErrorResponse "서버 오류"
// @Router /api/v1/agents/{id}/health [get]
func (ac *AgentController) GetAgentHealth(c *gin.Context) {
	agentID := c.Param("id")
	
	health, err := ac.agentService.GetHealthStatus(c.Request.Context(), agentID)
	if err != nil {
		if err.Error() == "agent not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "에이전트를 찾을 수 없습니다",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "헬스 상태 조회 실패",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, health)
}

// GetAgentMetrics는 에이전트 메트릭을 조회합니다.
// @Summary 에이전트 메트릭 조회
// @Description 특정 에이전트의 메트릭을 조회합니다
// @Tags agents
// @Accept json
// @Produce json
// @Param id path string true "에이전트 ID"
// @Security BearerAuth
// @Success 200 {object} agent.AgentMetrics "에이전트 메트릭"
// @Failure 401 {object} models.ErrorResponse "인증 실패"
// @Failure 404 {object} models.ErrorResponse "에이전트 없음"
// @Failure 500 {object} models.ErrorResponse "서버 오류"
// @Router /api/v1/agents/{id}/metrics [get]
func (ac *AgentController) GetAgentMetrics(c *gin.Context) {
	agentID := c.Param("id")
	
	metrics, err := ac.agentService.GetAgentMetrics(c.Request.Context(), agentID)
	if err != nil {
		if err.Error() == "agent not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "에이전트를 찾을 수 없습니다",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "메트릭 조회 실패",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, metrics)
}

// BatchStartAgents는 여러 에이전트를 일괄 시작합니다.
// @Summary 에이전트 일괄 시작
// @Description 여러 에이전트를 한 번에 시작합니다
// @Tags agents
// @Accept json
// @Produce json
// @Param request body map[string][]string true "에이전트 ID 목록"
// @Security BearerAuth
// @Success 200 {object} []agent.AgentOperationResult "일괄 작업 결과"
// @Failure 400 {object} models.ErrorResponse "잘못된 요청"
// @Failure 401 {object} models.ErrorResponse "인증 실패"
// @Failure 500 {object} models.ErrorResponse "서버 오류"
// @Router /api/v1/agents/batch/start [post]
func (ac *AgentController) BatchStartAgents(c *gin.Context) {
	var req struct {
		AgentIDs []string `json:"agent_ids" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "잘못된 요청 데이터",
			"details": err.Error(),
		})
		return
	}
	
	results, err := ac.agentService.StartMultipleAgents(c.Request.Context(), req.AgentIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "에이전트 일괄 시작 실패",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"results": results,
		"count":   len(results),
	})
}

// BatchStopAgents는 여러 에이전트를 일괄 중지합니다.
// @Summary 에이전트 일괄 중지
// @Description 여러 에이전트를 한 번에 중지합니다
// @Tags agents
// @Accept json
// @Produce json
// @Param request body map[string][]string true "에이전트 ID 목록"
// @Security BearerAuth
// @Success 200 {object} []agent.AgentOperationResult "일괄 작업 결과"
// @Failure 400 {object} models.ErrorResponse "잘못된 요청"
// @Failure 401 {object} models.ErrorResponse "인증 실패"
// @Failure 500 {object} models.ErrorResponse "서버 오류"
// @Router /api/v1/agents/batch/stop [post]
func (ac *AgentController) BatchStopAgents(c *gin.Context) {
	var req struct {
		AgentIDs []string `json:"agent_ids" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "잘못된 요청 데이터",
			"details": err.Error(),
		})
		return
	}
	
	results, err := ac.agentService.StopMultipleAgents(c.Request.Context(), req.AgentIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "에이전트 일괄 중지 실패",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"results": results,
		"count":   len(results),
	})
}

// WebSocket upgrader
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// 실제 프로덕션에서는 더 엄격한 origin 검사가 필요
		return true
	},
}

// StreamAgentLogs는 에이전트 로그를 실시간 스트리밍합니다.
// @Summary 에이전트 로그 스트리밍
// @Description WebSocket을 통해 특정 에이전트의 로그를 실시간으로 스트리밍합니다
// @Tags agents
// @Param id path string true "에이전트 ID"
// @Param follow query bool false "실시간 팔로우 여부" default(true)
// @Param tail query int false "마지막 N개 라인" default(100)
// @Security BearerAuth
// @Router /api/v1/agents/{id}/logs/stream [get]
func (ac *AgentController) StreamAgentLogs(c *gin.Context) {
	agentID := c.Param("id")
	
	// 에이전트 존재 확인
	_, err := ac.agentService.GetAgent(c.Request.Context(), agentID)
	if err != nil {
		if err.Error() == "agent not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "에이전트를 찾을 수 없습니다",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "에이전트 조회 실패",
			"details": err.Error(),
		})
		return
	}
	
	// WebSocket 연결 업그레이드
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "WebSocket 업그레이드 실패",
			"details": err.Error(),
		})
		return
	}
	defer conn.Close()
	
	// 로그 스트리밍 시작
	ac.startLogStreaming(c.Request.Context(), conn, agentID)
}

// StreamAgentEvents는 에이전트 이벤트를 실시간 스트리밍합니다.
// @Summary 에이전트 이벤트 스트리밍
// @Description WebSocket을 통해 특정 에이전트의 이벤트를 실시간으로 스트리밍합니다
// @Tags agents
// @Param id path string true "에이전트 ID"
// @Security BearerAuth
// @Router /api/v1/agents/{id}/events/stream [get]
func (ac *AgentController) StreamAgentEvents(c *gin.Context) {
	agentID := c.Param("id")
	
	// 에이전트 존재 확인
	_, err := ac.agentService.GetAgent(c.Request.Context(), agentID)
	if err != nil {
		if err.Error() == "agent not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "에이전트를 찾을 수 없습니다",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "에이전트 조회 실패",
			"details": err.Error(),
		})
		return
	}
	
	// WebSocket 연결 업그레이드
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "WebSocket 업그레이드 실패",
			"details": err.Error(),
		})
		return
	}
	defer conn.Close()
	
	// 이벤트 스트리밍 시작
	ac.startEventStreaming(c.Request.Context(), conn, agentID)
}

// StreamAllEvents는 모든 에이전트의 이벤트를 실시간 스트리밍합니다.
// @Summary 모든 에이전트 이벤트 스트리밍
// @Description WebSocket을 통해 모든 에이전트의 이벤트를 실시간으로 스트리밍합니다
// @Tags agents
// @Security BearerAuth
// @Router /api/v1/agents/events/stream [get]
func (ac *AgentController) StreamAllEvents(c *gin.Context) {
	// WebSocket 연결 업그레이드
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "WebSocket 업그레이드 실패",
			"details": err.Error(),
		})
		return
	}
	defer conn.Close()
	
	// 전역 이벤트 스트리밍 시작
	ac.startGlobalEventStreaming(c.Request.Context(), conn)
}

// 로그 스트리밍 헬퍼 함수
func (ac *AgentController) startLogStreaming(ctx context.Context, conn *websocket.Conn, agentID string) {
	// TODO: 실제 로그 스트리밍 구현
	// 현재는 mock 구현으로 데모 목적
	
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Mock 로그 메시지 전송
			logMessage := map[string]interface{}{
				"type": "log",
				"data": map[string]interface{}{
					"agent_id":  agentID,
					"timestamp": time.Now(),
					"level":     "info",
					"message":   "Sample log message from agent " + agentID,
				},
			}
			
			if err := conn.WriteJSON(logMessage); err != nil {
				return // 연결 종료
			}
		}
	}
}

// 이벤트 스트리밍 헬퍼 함수
func (ac *AgentController) startEventStreaming(ctx context.Context, conn *websocket.Conn, agentID string) {
	// EventBus에서 해당 에이전트의 이벤트 구독
	// TODO: MonitoringService를 통해 이벤트 구독 구현
	
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Mock 이벤트 메시지 전송
			eventMessage := map[string]interface{}{
				"type": "event",
				"data": agent.AgentEvent{
					Type:      agent.AgentEventStarted,
					AgentID:   agentID,
					Timestamp: time.Now(),
					Message:   "Mock event for agent " + agentID,
				},
			}
			
			if err := conn.WriteJSON(eventMessage); err != nil {
				return // 연결 종료
			}
		}
	}
}

// 전역 이벤트 스트리밍 헬퍼 함수
func (ac *AgentController) startGlobalEventStreaming(ctx context.Context, conn *websocket.Conn) {
	// 전역 EventBus 구독
	// TODO: MonitoringService를 통해 전역 이벤트 구독 구현
	
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Mock 전역 이벤트 메시지 전송
			eventMessage := map[string]interface{}{
				"type": "global_event",
				"data": agent.AgentEvent{
					Type:      agent.AgentEventCreated,
					AgentID:   "global-agent-id",
					Timestamp: time.Now(),
					Message:   "Mock global event",
				},
			}
			
			if err := conn.WriteJSON(eventMessage); err != nil {
				return // 연결 종료
			}
		}
	}
}