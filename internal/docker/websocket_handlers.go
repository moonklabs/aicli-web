package docker

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// WebSocketHandler WebSocket HTTP 핸들러
type WebSocketHandler struct {
	streamingManager *WebSocketStreamingManager
	ptyManager       PTYSessionManagement
	containerPTY     ContainerPTYManagement
}

// NewWebSocketHandler 새로운 WebSocket 핸들러 생성
func NewWebSocketHandler(streamingManager *WebSocketStreamingManager, ptyManager PTYSessionManagement, containerPTY ContainerPTYManagement) *WebSocketHandler {
	return &WebSocketHandler{
		streamingManager: streamingManager,
		ptyManager:       ptyManager,
		containerPTY:     containerPTY,
	}
}

// HandlePTYWebSocket PTY WebSocket 연결 핸들러
// GET /api/pty/{sessionID}/ws
func (wsh *WebSocketHandler) HandlePTYWebSocket(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["sessionID"]

	if sessionID == "" {
		http.Error(w, "Session ID is required", http.StatusBadRequest)
		return
	}

	// PTY 세션 존재 확인
	_, err := wsh.ptyManager.GetSession(sessionID)
	if err != nil {
		http.Error(w, fmt.Sprintf("PTY session not found: %s", sessionID), http.StatusNotFound)
		return
	}

	// WebSocket 연결 및 PTY 세션 연결
	if err := wsh.streamingManager.HandleWebSocketConnection(w, r, sessionID); err != nil {
		http.Error(w, fmt.Sprintf("WebSocket connection failed: %v", err), http.StatusInternalServerError)
		return
	}
}

// HandleContainerWebSocket 컨테이너 WebSocket 연결 핸들러 (새 PTY 세션 생성)
// GET /api/container/{containerID}/ws
func (wsh *WebSocketHandler) HandleContainerWebSocket(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	containerID := vars["containerID"]

	if containerID == "" {
		http.Error(w, "Container ID is required", http.StatusBadRequest)
		return
	}

	// 쿼리 파라미터에서 설정 읽기
	shell := r.URL.Query().Get("shell")
	workingDir := r.URL.Query().Get("workingDir")
	widthStr := r.URL.Query().Get("width")
	heightStr := r.URL.Query().Get("height")

	// 기본값 설정
	if shell == "" {
		shell = "/bin/bash"
	}
	if workingDir == "" {
		workingDir = "/workspace"
	}

	width, height := 80, 24
	if widthStr != "" {
		if w, err := strconv.Atoi(widthStr); err == nil && w > 0 {
			width = w
		}
	}
	if heightStr != "" {
		if h, err := strconv.Atoi(heightStr); err == nil && h > 0 {
			height = h
		}
	}

	// PTY 설정
	config := PTYConfig{
		Shell:      shell,
		WorkingDir: workingDir,
		Size: PTYSize{
			Width:  width,
			Height: height,
		},
		User: "root",
		Tty:  true,
	}

	// 컨테이너에 새 PTY 세션 생성
	ptySession, err := wsh.containerPTY.CreateContainerPTY(r.Context(), containerID, config)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create PTY session: %v", err), http.StatusInternalServerError)
		return
	}

	// WebSocket 연결 및 PTY 세션 연결
	if err := wsh.streamingManager.HandleWebSocketConnection(w, r, ptySession.ID()); err != nil {
		http.Error(w, fmt.Sprintf("WebSocket connection failed: %v", err), http.StatusInternalServerError)
		return
	}
}

// HandleWebSocketStats WebSocket 통계 조회 핸들러
// GET /api/websocket/stats
func (wsh *WebSocketHandler) HandleWebSocketStats(w http.ResponseWriter, r *http.Request) {
	stats := wsh.streamingManager.GetManager().GetStats()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode stats: %v", err), http.StatusInternalServerError)
		return
	}
}

// HandleWebSocketConnections 활성 WebSocket 연결 목록 조회
// GET /api/websocket/connections
func (wsh *WebSocketHandler) HandleWebSocketConnections(w http.ResponseWriter, r *http.Request) {
	connections := wsh.streamingManager.GetManager().ListConnections()

	type connectionInfo struct {
		ID           string `json:"id"`
		PTYSessionID string `json:"pty_session_id"`
		IsConnected  bool   `json:"is_connected"`
	}

	connectionList := make([]connectionInfo, len(connections))
	for i, conn := range connections {
		connectionList[i] = connectionInfo{
			ID:           conn.GetConnectionID(),
			PTYSessionID: conn.GetPTYSessionID(),
			IsConnected:  conn.IsConnected(),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(connectionList); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode connections: %v", err), http.StatusInternalServerError)
		return
	}
}

// HandleCloseWebSocketConnection WebSocket 연결 종료
// DELETE /api/websocket/connections/{connectionID}
func (wsh *WebSocketHandler) HandleCloseWebSocketConnection(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	connectionID := vars["connectionID"]

	if connectionID == "" {
		http.Error(w, "Connection ID is required", http.StatusBadRequest)
		return
	}

	if err := wsh.streamingManager.GetManager().CloseConnection(connectionID); err != nil {
		http.Error(w, fmt.Sprintf("Failed to close connection: %v", err), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// RegisterRoutes HTTP 라우터에 WebSocket 핸들러 등록
func (wsh *WebSocketHandler) RegisterRoutes(router *mux.Router) {
	// PTY WebSocket 연결
	router.HandleFunc("/api/pty/{sessionID}/ws", wsh.HandlePTYWebSocket).Methods("GET")
	
	// 컨테이너 WebSocket 연결 (새 PTY 세션 생성)
	router.HandleFunc("/api/container/{containerID}/ws", wsh.HandleContainerWebSocket).Methods("GET")
	
	// WebSocket 통계 및 관리
	router.HandleFunc("/api/websocket/stats", wsh.HandleWebSocketStats).Methods("GET")
	router.HandleFunc("/api/websocket/connections", wsh.HandleWebSocketConnections).Methods("GET")
	router.HandleFunc("/api/websocket/connections/{connectionID}", wsh.HandleCloseWebSocketConnection).Methods("DELETE")
}

// WebSocketHealthCheck WebSocket 시스템 헬스체크
type WebSocketHealthCheck struct {
	streamingManager *WebSocketStreamingManager
}

// NewWebSocketHealthCheck 새로운 WebSocket 헬스체크 생성
func NewWebSocketHealthCheck(streamingManager *WebSocketStreamingManager) *WebSocketHealthCheck {
	return &WebSocketHealthCheck{
		streamingManager: streamingManager,
	}
}

// Check 헬스체크 수행
func (wshc *WebSocketHealthCheck) Check() error {
	stats := wshc.streamingManager.GetManager().GetStats()
	
	// 기본적인 상태 확인
	if stats.ActiveConnections < 0 {
		return fmt.Errorf("invalid active connections count: %d", stats.ActiveConnections)
	}

	if stats.TotalErrors > stats.TotalMessages {
		return fmt.Errorf("error rate too high: %d errors out of %d messages", stats.TotalErrors, stats.TotalMessages)
	}

	return nil
}

// GetHealthStatus 헬스 상태 조회
func (wshc *WebSocketHealthCheck) GetHealthStatus() map[string]interface{} {
	stats := wshc.streamingManager.GetManager().GetStats()
	
	status := map[string]interface{}{
		"healthy":             wshc.Check() == nil,
		"active_connections":  stats.ActiveConnections,
		"total_connections":   stats.TotalConnections,
		"max_connections":     stats.MaxConnections,
		"total_messages":      stats.TotalMessages,
		"total_errors":        stats.TotalErrors,
		"cleanup_interval":    stats.CleanupInterval.String(),
	}

	if stats.TotalMessages > 0 {
		errorRate := float64(stats.TotalErrors) / float64(stats.TotalMessages) * 100
		status["error_rate_percent"] = errorRate
	} else {
		status["error_rate_percent"] = 0.0
	}

	return status
}