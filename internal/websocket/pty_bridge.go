package websocket

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/aicli/aicli-web/internal/pty"
	log "github.com/sirupsen/logrus"
)

// PTYBridge PTY와 WebSocket 간 브리지
type PTYBridge struct {
	streamManager *StreamManager
	ptyManager    pty.PTYSessionInterface
	bridges       map[string]*BridgeConnection
	mutex         sync.RWMutex
	config        *BridgeConfig
	
	// 메트릭
	totalBridges  uint64
	totalBytes    uint64
}

// BridgeConnection 브리지 연결
type BridgeConnection struct {
	ID             string
	PTYSessionID   string
	StreamSessionID string
	PTYFile        io.ReadWriteCloser
	Context        context.Context
	Cancel         context.CancelFunc
	Active         bool
	CreatedAt      time.Time
	LastActive     time.Time
	
	// 버퍼
	readBuffer     []byte
	writeBuffer    []byte
	
	// 통계
	bytesFromPTY   uint64
	bytesToPTY     uint64
}

// BridgeConfig 브리지 설정
type BridgeConfig struct {
	BufferSize       int
	ReadTimeout      time.Duration
	WriteTimeout     time.Duration
	ChunkSize        int
	EnableBase64     bool
	EnableANSIParse  bool
}

// DefaultBridgeConfig 기본 브리지 설정
func DefaultBridgeConfig() *BridgeConfig {
	return &BridgeConfig{
		BufferSize:      8192,
		ReadTimeout:     30 * time.Second,
		WriteTimeout:    10 * time.Second,
		ChunkSize:       4096,
		EnableBase64:    false,
		EnableANSIParse: true,
	}
}

// NewPTYBridge 새 PTY 브리지 생성
func NewPTYBridge(streamManager *StreamManager, ptyManager pty.PTYSessionInterface, config *BridgeConfig) *PTYBridge {
	if config == nil {
		config = DefaultBridgeConfig()
	}
	
	return &PTYBridge{
		streamManager: streamManager,
		ptyManager:    ptyManager,
		bridges:       make(map[string]*BridgeConnection),
		config:        config,
	}
}

// CreateBridge PTY 세션과 WebSocket 스트림 연결
func (pb *PTYBridge) CreateBridge(ptySessionID string) (*BridgeConnection, error) {
	pb.mutex.Lock()
	defer pb.mutex.Unlock()
	
	// PTY 세션 확인
	ptySession, err := pb.ptyManager.GetSession(ptySessionID)
	if err != nil {
		return nil, fmt.Errorf("PTY session not found: %w", err)
	}
	
	// 스트림 세션 생성
	streamSession, err := pb.streamManager.CreateSession(ptySessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to create stream session: %w", err)
	}
	
	// 브리지 연결 생성
	ctx, cancel := context.WithCancel(context.Background())
	bridge := &BridgeConnection{
		ID:              generateBridgeID(),
		PTYSessionID:    ptySessionID,
		StreamSessionID: streamSession.ID,
		PTYFile:         ptySession.PTY,
		Context:         ctx,
		Cancel:          cancel,
		Active:          true,
		CreatedAt:       time.Now(),
		LastActive:      time.Now(),
		readBuffer:      make([]byte, pb.config.BufferSize),
		writeBuffer:     make([]byte, pb.config.BufferSize),
	}
	
	pb.bridges[bridge.ID] = bridge
	pb.totalBridges++
	
	// 브리지 처리 시작
	go pb.processPTYToWebSocket(bridge, streamSession)
	go pb.processWebSocketToPTY(bridge, streamSession)
	
	log.Infof("PTY bridge created: %s (PTY: %s, Stream: %s)", 
		bridge.ID, ptySessionID, streamSession.ID)
	
	return bridge, nil
}

// processPTYToWebSocket PTY에서 WebSocket으로 데이터 전송
func (pb *PTYBridge) processPTYToWebSocket(bridge *BridgeConnection, streamSession *StreamSession) {
	defer func() {
		bridge.Active = false
		bridge.Cancel()
		pb.removeBridge(bridge.ID)
	}()
	
	for {
		select {
		case <-bridge.Context.Done():
			return
		default:
			// PTY에서 읽기
			n, err := bridge.PTYFile.Read(bridge.readBuffer)
			if err != nil {
				if err != io.EOF {
					log.Errorf("PTY read error in bridge %s: %v", bridge.ID, err)
					streamSession.ErrorChan <- err
				}
				return
			}
			
			if n > 0 {
				data := bridge.readBuffer[:n]
				
				// 데이터 처리
				processedData := pb.processOutput(data)
				
				// WebSocket으로 전송
				message := pb.createMessage("output", processedData)
				
				select {
				case streamSession.OutputChan <- message:
					bridge.bytesFromPTY += uint64(n)
					pb.totalBytes += uint64(n)
					bridge.LastActive = time.Now()
				case <-time.After(pb.config.WriteTimeout):
					log.Warnf("Timeout sending to WebSocket in bridge %s", bridge.ID)
				}
			}
		}
	}
}

// processWebSocketToPTY WebSocket에서 PTY로 데이터 전송
func (pb *PTYBridge) processWebSocketToPTY(bridge *BridgeConnection, streamSession *StreamSession) {
	for {
		select {
		case <-bridge.Context.Done():
			return
			
		case input := <-streamSession.InputChan:
			// 입력 메시지 파싱
			var msg InputMessage
			if err := json.Unmarshal(input, &msg); err != nil {
				// 원시 데이터로 처리
				msg.Type = "input"
				msg.Data = string(input)
			}
			
			switch msg.Type {
			case "input":
				// PTY로 전송
				data := []byte(msg.Data)
				if pb.config.EnableBase64 {
					decoded, err := base64.StdEncoding.DecodeString(msg.Data)
					if err == nil {
						data = decoded
					}
				}
				
				n, err := bridge.PTYFile.Write(data)
				if err != nil {
					log.Errorf("PTY write error in bridge %s: %v", bridge.ID, err)
					streamSession.ErrorChan <- err
					return
				}
				
				bridge.bytesToPTY += uint64(n)
				bridge.LastActive = time.Now()
				
			case "resize":
				// PTY 크기 조정
				if msg.Rows > 0 && msg.Cols > 0 {
					pb.handleResize(bridge.PTYSessionID, msg.Rows, msg.Cols)
				}
				
			case "command":
				// 특수 명령 처리
				pb.handleCommand(bridge, msg.Command)
			}
		}
	}
}

// processOutput 출력 데이터 처리
func (pb *PTYBridge) processOutput(data []byte) []byte {
	// ANSI 이스케이프 시퀀스 처리
	if pb.config.EnableANSIParse {
		// TODO: ANSI 파서 통합
		return data
	}
	
	// Base64 인코딩
	if pb.config.EnableBase64 {
		encoded := base64.StdEncoding.EncodeToString(data)
		return []byte(encoded)
	}
	
	return data
}

// createMessage 메시지 생성
func (pb *PTYBridge) createMessage(msgType string, data []byte) []byte {
	message := OutputMessage{
		Type:      msgType,
		Data:      string(data),
		Timestamp: time.Now().UnixMilli(),
	}
	
	if pb.config.EnableBase64 {
		message.Encoding = "base64"
	}
	
	jsonData, err := json.Marshal(message)
	if err != nil {
		log.Errorf("Failed to marshal message: %v", err)
		return data
	}
	
	return jsonData
}

// handleResize PTY 크기 조정 처리
func (pb *PTYBridge) handleResize(ptySessionID string, rows, cols int) {
	// TODO: PTY 크기 조정 구현
	log.Infof("Resize PTY %s to %dx%d", ptySessionID, cols, rows)
}

// handleCommand 특수 명령 처리
func (pb *PTYBridge) handleCommand(bridge *BridgeConnection, command string) {
	switch command {
	case "pause":
		bridge.Active = false
		log.Infof("Bridge %s paused", bridge.ID)
		
	case "resume":
		bridge.Active = true
		log.Infof("Bridge %s resumed", bridge.ID)
		
	case "clear":
		// 터미널 클리어
		bridge.PTYFile.Write([]byte("\033[H\033[2J"))
		
	case "reset":
		// 터미널 리셋
		bridge.PTYFile.Write([]byte("\033c"))
		
	default:
		log.Warnf("Unknown command in bridge %s: %s", bridge.ID, command)
	}
}

// CloseBridge 브리지 종료
func (pb *PTYBridge) CloseBridge(bridgeID string) error {
	pb.mutex.Lock()
	bridge, exists := pb.bridges[bridgeID]
	pb.mutex.Unlock()
	
	if !exists {
		return fmt.Errorf("bridge %s not found", bridgeID)
	}
	
	// 컨텍스트 취소
	bridge.Cancel()
	
	// 스트림 세션 종료
	pb.streamManager.CloseSession(bridge.StreamSessionID)
	
	// PTY 세션 종료
	pb.ptyManager.CloseSession(bridge.PTYSessionID)
	
	pb.removeBridge(bridgeID)
	
	log.Infof("Bridge %s closed", bridgeID)
	return nil
}

// removeBridge 브리지 제거
func (pb *PTYBridge) removeBridge(bridgeID string) {
	pb.mutex.Lock()
	defer pb.mutex.Unlock()
	
	delete(pb.bridges, bridgeID)
}

// GetBridge 브리지 조회
func (pb *PTYBridge) GetBridge(bridgeID string) (*BridgeConnection, error) {
	pb.mutex.RLock()
	defer pb.mutex.RUnlock()
	
	bridge, exists := pb.bridges[bridgeID]
	if !exists {
		return nil, fmt.Errorf("bridge %s not found", bridgeID)
	}
	
	return bridge, nil
}

// ListBridges 브리지 목록 조회
func (pb *PTYBridge) ListBridges() []*BridgeConnection {
	pb.mutex.RLock()
	defer pb.mutex.RUnlock()
	
	bridges := make([]*BridgeConnection, 0, len(pb.bridges))
	for _, bridge := range pb.bridges {
		bridges = append(bridges, bridge)
	}
	
	return bridges
}

// GetStats 통계 조회
func (pb *PTYBridge) GetStats() map[string]interface{} {
	pb.mutex.RLock()
	defer pb.mutex.RUnlock()
	
	activeBridges := 0
	for _, bridge := range pb.bridges {
		if bridge.Active {
			activeBridges++
		}
	}
	
	return map[string]interface{}{
		"total_bridges":  pb.totalBridges,
		"active_bridges": activeBridges,
		"total_bytes":    pb.totalBytes,
	}
}

// InputMessage 입력 메시지
type InputMessage struct {
	Type    string `json:"type"`
	Data    string `json:"data"`
	Command string `json:"command,omitempty"`
	Rows    int    `json:"rows,omitempty"`
	Cols    int    `json:"cols,omitempty"`
}

// OutputMessage 출력 메시지
type OutputMessage struct {
	Type      string `json:"type"`
	Data      string `json:"data"`
	Encoding  string `json:"encoding,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

// generateBridgeID 브리지 ID 생성
func generateBridgeID() string {
	return fmt.Sprintf("bridge-%d-%d", time.Now().UnixNano(), randInt())
}