package websocket

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/aicli-web/internal/pty"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

// PTYHandler PTY 전용 WebSocket 핸들러
type PTYHandler struct {
	upgrader      websocket.Upgrader
	streamManager *StreamManager
	ptyManager    pty.PTYSessionInterface
	ptyBridge     *PTYBridge
	config        *PTYHandlerConfig
}

// PTYHandlerConfig PTY 핸들러 설정
type PTYHandlerConfig struct {
	ReadBufferSize  int
	WriteBufferSize int
	CheckOrigin     bool
	AllowedOrigins  []string
	HandshakeTimeout time.Duration
}

// DefaultPTYHandlerConfig 기본 PTY 핸들러 설정
func DefaultPTYHandlerConfig() *PTYHandlerConfig {
	return &PTYHandlerConfig{
		ReadBufferSize:   4096,
		WriteBufferSize:  4096,
		CheckOrigin:      false,
		HandshakeTimeout: 10 * time.Second,
	}
}

// NewPTYHandler 새 PTY WebSocket 핸들러 생성
func NewPTYHandler(ptyManager pty.PTYSessionInterface, config *PTYHandlerConfig) *PTYHandler {
	if config == nil {
		config = DefaultPTYHandlerConfig()
	}
	
	// WebSocket 업그레이더 설정
	upgrader := websocket.Upgrader{
		ReadBufferSize:   config.ReadBufferSize,
		WriteBufferSize:  config.WriteBufferSize,
		HandshakeTimeout: config.HandshakeTimeout,
		CheckOrigin: func(r *http.Request) bool {
			if !config.CheckOrigin {
				return true
			}
			
			origin := r.Header.Get("Origin")
			for _, allowed := range config.AllowedOrigins {
				if origin == allowed {
					return true
				}
			}
			
			return false
		},
		EnableCompression: true,
	}
	
	// 스트림 관리자 생성
	streamManager := NewStreamManager(nil)
	
	// PTY 브리지 생성
	ptyBridge := NewPTYBridge(streamManager, ptyManager, nil)
	
	return &PTYHandler{
		upgrader:      upgrader,
		streamManager: streamManager,
		ptyManager:    ptyManager,
		ptyBridge:     ptyBridge,
		config:        config,
	}
}

// HandlePTYWebSocket PTY WebSocket 연결 처리 (Gin)
func (h *PTYHandler) HandlePTYWebSocket(c *gin.Context) {
	// WebSocket 업그레이드
	ws, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Errorf("Failed to upgrade WebSocket: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "WebSocket upgrade failed"})
		return
	}
	
	// 연결 생성
	conn, err := h.streamManager.CreateConnection(ws)
	if err != nil {
		log.Errorf("Failed to create connection: %v", err)
		ws.Close()
		return
	}
	
	// 세션 ID 가져오기
	sessionID := c.Param("sessionID")
	if sessionID != "" {
		// 기존 세션에 연결
		if err := h.streamManager.AttachConnection(conn.ID, sessionID); err != nil {
			log.Errorf("Failed to attach connection to session: %v", err)
			h.sendErrorMessage(conn, "ATTACH_FAILED", err.Error())
		}
	}
	
	// 초기 메시지 전송
	h.sendWelcomeMessage(conn)
	
	// 연결 정보 로깅
	log.Infof("PTY WebSocket connection established: %s from %s", 
		conn.ID, c.Request.RemoteAddr)
}

// HandlePTYConnect PTY 연결 요청 처리
func (h *PTYHandler) HandlePTYConnect(c *gin.Context) {
	var req PTYConnectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	
	// PTY 세션 생성
	ptyConfig := &pty.PTYConfig{
		Rows: req.Rows,
		Cols: req.Cols,
		Term: req.Term,
		Shell: req.Shell,
		WorkingDir: req.WorkingDir,
		Environment: req.Environment,
	}
	
	ctx := c.Request.Context()
	ptySession, err := h.ptyManager.CreateSession(ctx, req.ContainerID, ptyConfig)
	if err != nil {
		log.Errorf("Failed to create PTY session: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create PTY session"})
		return
	}
	
	// 브리지 생성
	bridge, err := h.ptyBridge.CreateBridge(ptySession.ID)
	if err != nil {
		log.Errorf("Failed to create bridge: %v", err)
		h.ptyManager.CloseSession(ptySession.ID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create bridge"})
		return
	}
	
	// 응답
	c.JSON(http.StatusOK, PTYConnectResponse{
		SessionID:       ptySession.ID,
		StreamSessionID: bridge.StreamSessionID,
		BridgeID:        bridge.ID,
		WebSocketPath:   "/api/pty/ws/" + bridge.StreamSessionID,
	})
}

// HandlePTYResize PTY 크기 조정 요청 처리
func (h *PTYHandler) HandlePTYResize(c *gin.Context) {
	sessionID := c.Param("sessionID")
	
	var req PTYResizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	
	// TODO: Docker PTY Manager를 통한 크기 조정 구현
	log.Infof("Resize PTY %s to %dx%d", sessionID, req.Cols, req.Rows)
	
	// 모든 연결에 크기 변경 알림
	resizeMsg := map[string]interface{}{
		"type": "resize",
		"rows": req.Rows,
		"cols": req.Cols,
	}
	
	if data, err := json.Marshal(resizeMsg); err == nil {
		if err := h.streamManager.SendToSession(sessionID, data); err != nil {
			log.Errorf("Failed to send resize message: %v", err)
		}
	}
	
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// HandlePTYDisconnect PTY 연결 해제 요청 처리
func (h *PTYHandler) HandlePTYDisconnect(c *gin.Context) {
	sessionID := c.Param("sessionID")
	
	// 브리지 찾기 및 종료
	bridges := h.ptyBridge.ListBridges()
	for _, bridge := range bridges {
		if bridge.PTYSessionID == sessionID {
			if err := h.ptyBridge.CloseBridge(bridge.ID); err != nil {
				log.Errorf("Failed to close bridge: %v", err)
			}
			break
		}
	}
	
	// PTY 세션 종료
	if err := h.ptyManager.CloseSession(sessionID); err != nil {
		log.Errorf("Failed to close PTY session: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to close session"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// HandlePTYList PTY 세션 목록 조회
func (h *PTYHandler) HandlePTYList(c *gin.Context) {
	sessions := h.ptyManager.ListSessions()
	bridges := h.ptyBridge.ListBridges()
	
	// 세션 정보 구성
	sessionList := make([]map[string]interface{}, 0)
	for id, session := range sessions {
		sessionInfo := map[string]interface{}{
			"session_id":   id,
			"container_id": session.ContainerID,
			"status":       session.Status.String(),
			"created_at":   session.CreatedAt,
			"last_active":  session.LastActive,
		}
		
		// 브리지 정보 추가
		for _, bridge := range bridges {
			if bridge.PTYSessionID == id {
				sessionInfo["bridge_id"] = bridge.ID
				sessionInfo["stream_session_id"] = bridge.StreamSessionID
				sessionInfo["bridge_active"] = bridge.Active
				break
			}
		}
		
		sessionList = append(sessionList, sessionInfo)
	}
	
	c.JSON(http.StatusOK, gin.H{
		"sessions": sessionList,
		"total":    len(sessionList),
	})
}

// HandlePTYStats PTY 통계 조회 처리
func (h *PTYHandler) HandlePTYStats(c *gin.Context) {
	stats := map[string]interface{}{
		"stream":     h.streamManager.GetStats(),
		"pty_bridge": h.ptyBridge.GetStats(),
		"pty_sessions": h.ptyManager.GetStats(),
	}
	
	c.JSON(http.StatusOK, stats)
}

// sendWelcomeMessage 환영 메시지 전송
func (h *PTYHandler) sendWelcomeMessage(conn *StreamConnection) {
	welcome := map[string]interface{}{
		"type":         "welcome",
		"message":      "PTY WebSocket connection established",
		"connectionID": conn.ID,
		"timestamp":    time.Now().UnixMilli(),
		"version":      "1.0",
	}
	
	data, err := json.Marshal(welcome)
	if err != nil {
		log.Errorf("Failed to marshal welcome message: %v", err)
		return
	}
	
	select {
	case conn.SendChan <- data:
	default:
		log.Warnf("Failed to send welcome message to connection %s", conn.ID)
	}
}

// sendErrorMessage 에러 메시지 전송
func (h *PTYHandler) sendErrorMessage(conn *StreamConnection, code, message string) {
	errorMsg := map[string]interface{}{
		"type":      "error",
		"code":      code,
		"message":   message,
		"timestamp": time.Now().UnixMilli(),
	}
	
	data, err := json.Marshal(errorMsg)
	if err != nil {
		log.Errorf("Failed to marshal error message: %v", err)
		return
	}
	
	select {
	case conn.SendChan <- data:
	default:
		log.Warnf("Failed to send error message to connection %s", conn.ID)
	}
}

// RegisterPTYRoutes Gin 라우터에 PTY 라우트 등록
func (h *PTYHandler) RegisterPTYRoutes(router *gin.RouterGroup) {
	// PTY WebSocket 엔드포인트
	router.GET("/pty/ws", h.HandlePTYWebSocket)
	router.GET("/pty/ws/:sessionID", h.HandlePTYWebSocket)
	
	// PTY 관리 엔드포인트
	router.POST("/pty/connect", h.HandlePTYConnect)
	router.POST("/pty/:sessionID/resize", h.HandlePTYResize)
	router.DELETE("/pty/:sessionID", h.HandlePTYDisconnect)
	router.GET("/pty/sessions", h.HandlePTYList)
	
	// 통계 엔드포인트
	router.GET("/pty/stats", h.HandlePTYStats)
}

// PTYConnectRequest PTY 연결 요청
type PTYConnectRequest struct {
	ContainerID string            `json:"container_id" binding:"required"`
	Rows        int               `json:"rows"`
	Cols        int               `json:"cols"`
	Term        string            `json:"term"`
	Shell       string            `json:"shell"`
	WorkingDir  string            `json:"working_dir"`
	Environment map[string]string `json:"environment"`
}

// PTYConnectResponse PTY 연결 응답
type PTYConnectResponse struct {
	SessionID       string `json:"session_id"`
	StreamSessionID string `json:"stream_session_id"`
	BridgeID        string `json:"bridge_id"`
	WebSocketPath   string `json:"websocket_path"`
}

// PTYResizeRequest PTY 크기 조정 요청
type PTYResizeRequest struct {
	Rows int `json:"rows" binding:"required,min=1"`
	Cols int `json:"cols" binding:"required,min=1"`
}