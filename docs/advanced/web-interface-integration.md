# Real-time Web Interface Integration

실시간 웹 인터페이스 통합을 통해 Claude AI와의 상호작용을 웹 애플리케이션에서 직접 구현할 수 있습니다. WebSocket 기반 실시간 통신과 다중 사용자 협업을 지원합니다.

## 📋 목차

- [개요](#개요)
- [아키텍처](#아키텍처)
- [WebSocket 통신](#websocket-통신)
- [메시지 라우팅](#메시지-라우팅)
- [다중 사용자 협업](#다중-사용자-협업)
- [파일 관리](#파일-관리)
- [보안 및 인증](#보안-및-인증)
- [성능 최적화](#성능-최적화)
- [문제 해결](#문제-해결)

## 🎯 개요

### 주요 기능

- **실시간 WebSocket 통신**: 양방향 실시간 메시지 교환
- **스트리밍 응답**: Claude의 응답을 실시간으로 스트리밍
- **다중 사용자 지원**: 동시 사용자 세션 관리 및 협업
- **메시지 라우팅**: 지능형 메시지 라우팅 및 필터링
- **파일 공유**: 실시간 파일 업로드 및 공유
- **연결 관리**: 자동 재연결 및 연결 상태 관리

### 성능 목표

```bash
# 목표 성능 지표
Concurrent Connections: 1,000+
Message Latency: <50ms
Throughput: 1,000+ messages/sec
Connection Uptime: 99.9%
File Upload Speed: 10MB/sec
```

## 🏗️ 아키텍처

```
┌─────────────────┐    WebSocket    ┌──────────────────┐    Message Queue    ┌─────────────────┐
│   Web Client    │◄──────────────►│  WebSocket       │◄─────────────────►│  Message        │
│   (Browser)     │                │  Gateway         │                    │  Router         │
└─────────────────┘                └──────────────────┘                    └─────────────────┘
                                            │                                         │
                                            │                                         │
┌─────────────────┐    HTTP/REST    ┌──────────────────┐    Session Pool    ┌─────────────────┐
│   Mobile App    │◄──────────────►│  API Gateway     │◄─────────────────►│  Claude AI      │
│   (React/Vue)   │                │                  │                    │  Sessions       │
└─────────────────┘                └──────────────────┘                    └─────────────────┘
                                            │                                         │
                                            │                                         │
┌─────────────────┐    Server-Sent  ┌──────────────────┐    File System    ┌─────────────────┐
│   Desktop App   │◄──────────────►│  Connection      │◄─────────────────►│  File           │
│   (Electron)    │     Events      │  Manager         │                    │  Manager        │
└─────────────────┘                └──────────────────┘                    └─────────────────┘
```

### 핵심 컴포넌트

1. **WebSocket Gateway**: 클라이언트 연결 관리
2. **Message Router**: 메시지 라우팅 및 필터링
3. **Connection Manager**: 연결 풀 및 상태 관리
4. **Web Session Manager**: 웹 세션 생명주기 관리
5. **File Manager**: 파일 업로드/다운로드 관리
6. **Stream Processor**: 실시간 응답 스트리밍

## 🔌 WebSocket 통신

### 연결 설정

#### 클라이언트 (JavaScript)

```javascript
// WebSocket 연결 설정
class ClaudeWebSocket {
    constructor(options = {}) {
        this.url = options.url || 'wss://localhost:8080/ws';
        this.reconnectAttempts = 0;
        this.maxReconnectAttempts = 5;
        this.reconnectInterval = 1000;
        this.handlers = new Map();
    }
    
    connect() {
        return new Promise((resolve, reject) => {
            this.ws = new WebSocket(this.url);
            
            this.ws.onopen = (event) => {
                console.log('WebSocket connected');
                this.reconnectAttempts = 0;
                resolve(event);
            };
            
            this.ws.onmessage = (event) => {
                this.handleMessage(JSON.parse(event.data));
            };
            
            this.ws.onclose = (event) => {
                this.handleDisconnect(event);
            };
            
            this.ws.onerror = (error) => {
                console.error('WebSocket error:', error);
                reject(error);
            };
        });
    }
    
    // 메시지 전송
    send(type, data) {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            const message = {
                type: type,
                data: data,
                timestamp: Date.now(),
                id: this.generateMessageId()
            };
            
            this.ws.send(JSON.stringify(message));
        } else {
            throw new Error('WebSocket is not connected');
        }
    }
    
    // Claude에게 메시지 전송
    sendToClaude(message, sessionId = null) {
        this.send('claude_message', {
            session_id: sessionId,
            content: message,
            stream: true  // 스트리밍 응답 요청
        });
    }
    
    // 파일 업로드
    uploadFile(file, sessionId = null) {
        return new Promise((resolve, reject) => {
            const reader = new FileReader();
            
            reader.onload = (e) => {
                this.send('file_upload', {
                    session_id: sessionId,
                    filename: file.name,
                    content_type: file.type,
                    size: file.size,
                    data: e.target.result.split(',')[1]  // base64 데이터
                });
            };
            
            reader.onerror = reject;
            reader.readAsDataURL(file);
        });
    }
    
    // 이벤트 핸들러 등록
    on(eventType, handler) {
        if (!this.handlers.has(eventType)) {
            this.handlers.set(eventType, []);
        }
        this.handlers.get(eventType).push(handler);
    }
    
    // 메시지 처리
    handleMessage(message) {
        const handlers = this.handlers.get(message.type) || [];
        handlers.forEach(handler => handler(message.data));
    }
    
    // 자동 재연결
    handleDisconnect(event) {
        if (this.reconnectAttempts < this.maxReconnectAttempts) {
            setTimeout(() => {
                this.reconnectAttempts++;
                console.log(`Reconnecting... (${this.reconnectAttempts}/${this.maxReconnectAttempts})`);
                this.connect();
            }, this.reconnectInterval * Math.pow(2, this.reconnectAttempts));
        }
    }
    
    generateMessageId() {
        return Date.now() + '-' + Math.random().toString(36).substr(2, 9);
    }
}

// 사용 예제
const claude = new ClaudeWebSocket({
    url: 'wss://localhost:8080/ws'
});

// 이벤트 핸들러 등록
claude.on('claude_response', (data) => {
    if (data.stream) {
        // 스트리밍 응답 처리
        appendToResponse(data.content);
    } else {
        // 완전한 응답 처리
        setResponse(data.content);
    }
});

claude.on('file_uploaded', (data) => {
    console.log('File uploaded:', data.file_id);
});

claude.on('session_created', (data) => {
    console.log('Session created:', data.session_id);
});

// 연결 및 사용
claude.connect().then(() => {
    // Claude에게 메시지 전송
    claude.sendToClaude('안녕하세요, Claude!');
    
    // 파일 업로드
    const fileInput = document.getElementById('file-input');
    fileInput.addEventListener('change', (e) => {
        const file = e.target.files[0];
        if (file) {
            claude.uploadFile(file);
        }
    });
});
```

#### React Hook 예제

```tsx
// useClaudeWebSocket.ts
import { useState, useEffect, useCallback, useRef } from 'react';

interface Message {
    id: string;
    type: 'user' | 'assistant';
    content: string;
    timestamp: number;
    streaming?: boolean;
}

export const useClaudeWebSocket = (sessionId?: string) => {
    const [messages, setMessages] = useState<Message[]>([]);
    const [isConnected, setIsConnected] = useState(false);
    const [isLoading, setIsLoading] = useState(false);
    const wsRef = useRef<ClaudeWebSocket | null>(null);
    
    useEffect(() => {
        const claude = new ClaudeWebSocket();
        wsRef.current = claude;
        
        // 이벤트 핸들러 등록
        claude.on('claude_response', (data) => {
            if (data.stream) {
                // 스트리밍 응답
                setMessages(prev => {
                    const lastMessage = prev[prev.length - 1];
                    if (lastMessage && lastMessage.streaming) {
                        // 기존 메시지 업데이트
                        return prev.map((msg, index) => 
                            index === prev.length - 1 
                                ? { ...msg, content: msg.content + data.content }
                                : msg
                        );
                    } else {
                        // 새 스트리밍 메시지 생성
                        return [...prev, {
                            id: data.message_id,
                            type: 'assistant',
                            content: data.content,
                            timestamp: Date.now(),
                            streaming: true
                        }];
                    }
                });
            } else {
                // 완전한 응답
                setMessages(prev => 
                    prev.map(msg => 
                        msg.id === data.message_id 
                            ? { ...msg, streaming: false }
                            : msg
                    )
                );
                setIsLoading(false);
            }
        });
        
        claude.on('connected', () => setIsConnected(true));
        claude.on('disconnected', () => setIsConnected(false));
        
        claude.connect();
        
        return () => {
            if (wsRef.current) {
                wsRef.current.disconnect();
            }
        };
    }, []);
    
    const sendMessage = useCallback((content: string) => {
        if (!wsRef.current || !isConnected) {
            throw new Error('WebSocket not connected');
        }
        
        // 사용자 메시지 추가
        const userMessage: Message = {
            id: Date.now().toString(),
            type: 'user',
            content,
            timestamp: Date.now()
        };
        
        setMessages(prev => [...prev, userMessage]);
        setIsLoading(true);
        
        // Claude에게 전송
        wsRef.current.sendToClaude(content, sessionId);
    }, [isConnected, sessionId]);
    
    const uploadFile = useCallback((file: File) => {
        if (!wsRef.current || !isConnected) {
            throw new Error('WebSocket not connected');
        }
        
        return wsRef.current.uploadFile(file, sessionId);
    }, [isConnected, sessionId]);
    
    return {
        messages,
        isConnected,
        isLoading,
        sendMessage,
        uploadFile
    };
};

// Chat 컴포넌트에서 사용
const ChatComponent: React.FC = () => {
    const { messages, isConnected, isLoading, sendMessage, uploadFile } = useClaudeWebSocket();
    const [input, setInput] = useState('');
    
    const handleSubmit = (e: React.FormEvent) => {
        e.preventDefault();
        if (input.trim() && !isLoading) {
            sendMessage(input);
            setInput('');
        }
    };
    
    return (
        <div className="chat-container">
            <div className="connection-status">
                {isConnected ? '🟢 연결됨' : '🔴 연결 끊김'}
            </div>
            
            <div className="messages">
                {messages.map(message => (
                    <div key={message.id} className={`message ${message.type}`}>
                        <div className="content">{message.content}</div>
                        {message.streaming && <div className="typing-indicator">...</div>}
                    </div>
                ))}
                {isLoading && <div className="loading">Claude가 응답 중...</div>}
            </div>
            
            <form onSubmit={handleSubmit} className="input-form">
                <input
                    type="text"
                    value={input}
                    onChange={(e) => setInput(e.target.value)}
                    placeholder="메시지를 입력하세요..."
                    disabled={!isConnected || isLoading}
                />
                <button type="submit" disabled={!isConnected || isLoading}>
                    전송
                </button>
            </form>
        </div>
    );
};
```

### 서버 구현 (Go)

```go
// websocket_handler.go
package claude

import (
    "context"
    "encoding/json"
    "log"
    "net/http"
    "sync"
    "time"
    
    "github.com/gorilla/websocket"
)

type WebSocketHandler struct {
    upgrader        websocket.Upgrader
    connections     map[string]*WebSocketConnection
    connectionsMu   sync.RWMutex
    messageRouter   *MessageRouter
    sessionManager  *WebSessionManager
}

type WebSocketConnection struct {
    ID          string
    Conn        *websocket.Conn
    SessionID   string
    UserID      string
    SendChan    chan []byte
    CloseChan   chan struct{}
    LastPing    time.Time
}

type Message struct {
    Type      string      `json:"type"`
    Data      interface{} `json:"data"`
    Timestamp int64       `json:"timestamp"`
    ID        string      `json:"id"`
}

func NewWebSocketHandler(messageRouter *MessageRouter, sessionManager *WebSessionManager) *WebSocketHandler {
    return &WebSocketHandler{
        upgrader: websocket.Upgrader{
            CheckOrigin: func(r *http.Request) bool {
                // CORS 설정 - 프로덕션에서는 적절히 제한
                return true
            },
            ReadBufferSize:  1024,
            WriteBufferSize: 1024,
        },
        connections:    make(map[string]*WebSocketConnection),
        messageRouter:  messageRouter,
        sessionManager: sessionManager,
    }
}

func (h *WebSocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
    conn, err := h.upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Printf("WebSocket upgrade failed: %v", err)
        return
    }
    
    // 연결 정보 생성
    wsConn := &WebSocketConnection{
        ID:        generateConnectionID(),
        Conn:      conn,
        SendChan:  make(chan []byte, 256),
        CloseChan: make(chan struct{}),
        LastPing:  time.Now(),
    }
    
    // 사용자 인증 및 세션 ID 추출
    userID, sessionID := h.authenticateConnection(r)
    wsConn.UserID = userID
    wsConn.SessionID = sessionID
    
    // 연결 등록
    h.registerConnection(wsConn)
    defer h.unregisterConnection(wsConn)
    
    // 메시지 처리 고루틴 시작
    go h.handleMessages(wsConn)
    go h.handleWrites(wsConn)
    
    // Ping/Pong 헬스체크
    go h.pingHandler(wsConn)
    
    // 연결 성공 메시지 전송
    h.sendMessage(wsConn, Message{
        Type:      "connected",
        Data:      map[string]interface{}{"connection_id": wsConn.ID},
        Timestamp: time.Now().Unix(),
        ID:        generateMessageID(),
    })
    
    // 메인 읽기 루프
    h.readPump(wsConn)
}

func (h *WebSocketHandler) readPump(wsConn *WebSocketConnection) {
    defer func() {
        close(wsConn.CloseChan)
        wsConn.Conn.Close()
    }()
    
    // 읽기 타임아웃 설정
    wsConn.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
    wsConn.Conn.SetPongHandler(func(string) error {
        wsConn.LastPing = time.Now()
        wsConn.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
        return nil
    })
    
    for {
        _, messageBytes, err := wsConn.Conn.ReadMessage()
        if err != nil {
            if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
                log.Printf("WebSocket error: %v", err)
            }
            break
        }
        
        var message Message
        if err := json.Unmarshal(messageBytes, &message); err != nil {
            log.Printf("Invalid message format: %v", err)
            continue
        }
        
        // 메시지 라우팅
        h.messageRouter.RouteMessage(wsConn, &message)
    }
}

func (h *WebSocketHandler) handleWrites(wsConn *WebSocketConnection) {
    ticker := time.NewTicker(54 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case message, ok := <-wsConn.SendChan:
            wsConn.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
            if !ok {
                wsConn.Conn.WriteMessage(websocket.CloseMessage, []byte{})
                return
            }
            
            if err := wsConn.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
                log.Printf("WebSocket write error: %v", err)
                return
            }
            
        case <-ticker.C:
            // Ping 전송
            wsConn.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
            if err := wsConn.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
                return
            }
            
        case <-wsConn.CloseChan:
            return
        }
    }
}

// Claude 응답 스트리밍
func (h *WebSocketHandler) StreamClaudeResponse(wsConn *WebSocketConnection, sessionID string, userMessage string) {
    session, err := h.sessionManager.GetSession(sessionID)
    if err != nil {
        h.sendError(wsConn, "Session not found")
        return
    }
    
    // Claude API 스트리밍 요청
    responseStream, err := session.SendMessageStream(context.Background(), userMessage)
    if err != nil {
        h.sendError(wsConn, "Failed to send message to Claude")
        return
    }
    
    messageID := generateMessageID()
    
    // 스트리밍 응답 처리
    go func() {
        defer responseStream.Close()
        
        for {
            chunk, err := responseStream.Recv()
            if err != nil {
                if err == io.EOF {
                    // 스트리밍 완료
                    h.sendMessage(wsConn, Message{
                        Type: "claude_response",
                        Data: map[string]interface{}{
                            "message_id": messageID,
                            "content":    "",
                            "stream":     false,
                            "completed":  true,
                        },
                        Timestamp: time.Now().Unix(),
                        ID:        generateMessageID(),
                    })
                    break
                }
                log.Printf("Stream error: %v", err)
                h.sendError(wsConn, "Stream error occurred")
                break
            }
            
            // 청크 전송
            h.sendMessage(wsConn, Message{
                Type: "claude_response",
                Data: map[string]interface{}{
                    "message_id": messageID,
                    "content":    chunk.Content,
                    "stream":     true,
                    "completed":  false,
                },
                Timestamp: time.Now().Unix(),
                ID:        generateMessageID(),
            })
        }
    }()
}

func (h *WebSocketHandler) sendMessage(wsConn *WebSocketConnection, message Message) {
    data, err := json.Marshal(message)
    if err != nil {
        log.Printf("Failed to marshal message: %v", err)
        return
    }
    
    select {
    case wsConn.SendChan <- data:
    default:
        // 채널이 가득 찬 경우 연결 종료
        close(wsConn.SendChan)
    }
}

func (h *WebSocketHandler) sendError(wsConn *WebSocketConnection, errorMsg string) {
    h.sendMessage(wsConn, Message{
        Type: "error",
        Data: map[string]interface{}{
            "message": errorMsg,
        },
        Timestamp: time.Now().Unix(),
        ID:        generateMessageID(),
    })
}
```

## 🔀 메시지 라우팅

### 라우팅 규칙

```go
// message_router.go
type MessageRouter struct {
    routes     map[string]MessageHandler
    middleware []MiddlewareFunc
    filters    []MessageFilter
}

type MessageHandler interface {
    HandleMessage(conn *WebSocketConnection, message *Message) error
}

type MiddlewareFunc func(next MessageHandler) MessageHandler

type MessageFilter interface {
    ShouldProcess(conn *WebSocketConnection, message *Message) bool
}

// 메시지 핸들러 등록
func (r *MessageRouter) RegisterHandler(messageType string, handler MessageHandler) {
    r.routes[messageType] = handler
}

// 미들웨어 추가
func (r *MessageRouter) Use(middleware MiddlewareFunc) {
    r.middleware = append(r.middleware, middleware)
}

// 필터 추가
func (r *MessageRouter) AddFilter(filter MessageFilter) {
    r.filters = append(r.filters, filter)
}

// 메시지 라우팅
func (r *MessageRouter) RouteMessage(conn *WebSocketConnection, message *Message) {
    // 필터 검사
    for _, filter := range r.filters {
        if !filter.ShouldProcess(conn, message) {
            return
        }
    }
    
    // 핸들러 조회
    handler, exists := r.routes[message.Type]
    if !exists {
        log.Printf("No handler for message type: %s", message.Type)
        return
    }
    
    // 미들웨어 체인 구성
    finalHandler := handler
    for i := len(r.middleware) - 1; i >= 0; i-- {
        finalHandler = r.middleware[i](finalHandler)
    }
    
    // 메시지 처리
    if err := finalHandler.HandleMessage(conn, message); err != nil {
        log.Printf("Message handling error: %v", err)
    }
}

// Claude 메시지 핸들러
type ClaudeMessageHandler struct {
    sessionManager *WebSessionManager
    wsHandler      *WebSocketHandler
}

func (h *ClaudeMessageHandler) HandleMessage(conn *WebSocketConnection, message *Message) error {
    data, ok := message.Data.(map[string]interface{})
    if !ok {
        return fmt.Errorf("invalid message data format")
    }
    
    content, ok := data["content"].(string)
    if !ok {
        return fmt.Errorf("missing or invalid content")
    }
    
    sessionID := conn.SessionID
    if sid, exists := data["session_id"]; exists {
        if s, ok := sid.(string); ok {
            sessionID = s
        }
    }
    
    // 스트리밍 여부 확인
    stream, _ := data["stream"].(bool)
    
    if stream {
        // 스트리밍 응답
        h.wsHandler.StreamClaudeResponse(conn, sessionID, content)
    } else {
        // 일반 응답
        response, err := h.sessionManager.SendMessage(sessionID, content)
        if err != nil {
            return err
        }
        
        h.wsHandler.sendMessage(conn, Message{
            Type: "claude_response",
            Data: map[string]interface{}{
                "content":   response,
                "stream":    false,
                "completed": true,
            },
            Timestamp: time.Now().Unix(),
            ID:        generateMessageID(),
        })
    }
    
    return nil
}

// 파일 업로드 핸들러
type FileUploadHandler struct {
    fileManager *FileManager
    wsHandler   *WebSocketHandler
}

func (h *FileUploadHandler) HandleMessage(conn *WebSocketConnection, message *Message) error {
    data, ok := message.Data.(map[string]interface{})
    if !ok {
        return fmt.Errorf("invalid file upload data")
    }
    
    // 파일 정보 추출
    filename, _ := data["filename"].(string)
    contentType, _ := data["content_type"].(string)
    size, _ := data["size"].(float64)
    fileData, _ := data["data"].(string)
    
    // 파일 저장
    fileID, err := h.fileManager.SaveFile(conn.SessionID, filename, contentType, []byte(fileData))
    if err != nil {
        return err
    }
    
    // 업로드 완료 알림
    h.wsHandler.sendMessage(conn, Message{
        Type: "file_uploaded",
        Data: map[string]interface{}{
            "file_id":      fileID,
            "filename":     filename,
            "content_type": contentType,
            "size":         size,
        },
        Timestamp: time.Now().Unix(),
        ID:        generateMessageID(),
    })
    
    return nil
}

// 인증 미들웨어
func AuthenticationMiddleware(next MessageHandler) MessageHandler {
    return MessageHandlerFunc(func(conn *WebSocketConnection, message *Message) error {
        if conn.UserID == "" {
            return fmt.Errorf("unauthorized")
        }
        return next.HandleMessage(conn, message)
    })
}

// 로깅 미들웨어
func LoggingMiddleware(next MessageHandler) MessageHandler {
    return MessageHandlerFunc(func(conn *WebSocketConnection, message *Message) error {
        start := time.Now()
        err := next.HandleMessage(conn, message)
        
        log.Printf("Message processed: type=%s, user=%s, duration=%v, error=%v",
            message.Type, conn.UserID, time.Since(start), err)
        
        return err
    })
}

// 속도 제한 필터
type RateLimitFilter struct {
    limits map[string]*rate.Limiter
    mu     sync.RWMutex
}

func (f *RateLimitFilter) ShouldProcess(conn *WebSocketConnection, message *Message) bool {
    f.mu.Lock()
    defer f.mu.Unlock()
    
    limiter, exists := f.limits[conn.UserID]
    if !exists {
        // 사용자당 초당 10개 메시지 제한
        limiter = rate.NewLimiter(rate.Limit(10), 10)
        f.limits[conn.UserID] = limiter
    }
    
    return limiter.Allow()
}
```

## 👥 다중 사용자 협업

### 협업 세션 관리

```go
// collaboration.go
type CollaborationManager struct {
    sessions    map[string]*CollaborationSession
    sessionsMu  sync.RWMutex
    wsHandler   *WebSocketHandler
}

type CollaborationSession struct {
    ID           string
    Participants map[string]*Participant
    SharedState  *SharedState
    CreatedAt    time.Time
    LastActivity time.Time
}

type Participant struct {
    UserID       string
    ConnectionID string
    Role         ParticipantRole
    JoinedAt     time.Time
    IsActive     bool
}

type ParticipantRole string

const (
    RoleOwner       ParticipantRole = "owner"
    RoleCollaborator ParticipantRole = "collaborator"
    RoleViewer      ParticipantRole = "viewer"
)

type SharedState struct {
    CurrentCode    string                 `json:"current_code"`
    CursorPositions map[string]CursorPos  `json:"cursor_positions"`
    Messages       []SharedMessage        `json:"messages"`
    Files          []SharedFile           `json:"files"`
}

type CursorPos struct {
    Line   int `json:"line"`
    Column int `json:"column"`
}

type SharedMessage struct {
    ID        string    `json:"id"`
    UserID    string    `json:"user_id"`
    Content   string    `json:"content"`
    Type      string    `json:"type"`
    Timestamp time.Time `json:"timestamp"`
}

// 협업 세션 생성
func (cm *CollaborationManager) CreateSession(ownerID string) (*CollaborationSession, error) {
    cm.sessionsMu.Lock()
    defer cm.sessionsMu.Unlock()
    
    sessionID := generateSessionID()
    session := &CollaborationSession{
        ID:           sessionID,
        Participants: make(map[string]*Participant),
        SharedState:  &SharedState{
            CursorPositions: make(map[string]CursorPos),
            Messages:        make([]SharedMessage, 0),
            Files:           make([]SharedFile, 0),
        },
        CreatedAt:    time.Now(),
        LastActivity: time.Now(),
    }
    
    // 소유자 추가
    session.Participants[ownerID] = &Participant{
        UserID:   ownerID,
        Role:     RoleOwner,
        JoinedAt: time.Now(),
        IsActive: true,
    }
    
    cm.sessions[sessionID] = session
    return session, nil
}

// 참가자 추가
func (cm *CollaborationManager) JoinSession(sessionID, userID, connectionID string, role ParticipantRole) error {
    cm.sessionsMu.Lock()
    defer cm.sessionsMu.Unlock()
    
    session, exists := cm.sessions[sessionID]
    if !exists {
        return fmt.Errorf("session not found")
    }
    
    participant := &Participant{
        UserID:       userID,
        ConnectionID: connectionID,
        Role:         role,
        JoinedAt:     time.Now(),
        IsActive:     true,
    }
    
    session.Participants[userID] = participant
    session.LastActivity = time.Now()
    
    // 기존 참가자들에게 새 참가자 알림
    cm.broadcastToSession(sessionID, Message{
        Type: "participant_joined",
        Data: map[string]interface{}{
            "user_id": userID,
            "role":    role,
        },
        Timestamp: time.Now().Unix(),
        ID:        generateMessageID(),
    })
    
    // 새 참가자에게 현재 상태 전송
    conn := cm.wsHandler.getConnection(connectionID)
    if conn != nil {
        cm.wsHandler.sendMessage(conn, Message{
            Type: "session_state",
            Data: session.SharedState,
            Timestamp: time.Now().Unix(),
            ID:        generateMessageID(),
        })
    }
    
    return nil
}

// 실시간 코드 동기화
func (cm *CollaborationManager) UpdateCode(sessionID, userID string, codeChange CodeChange) error {
    cm.sessionsMu.Lock()
    defer cm.sessionsMu.Unlock()
    
    session, exists := cm.sessions[sessionID]
    if !exists {
        return fmt.Errorf("session not found")
    }
    
    participant, exists := session.Participants[userID]
    if !exists || !participant.IsActive {
        return fmt.Errorf("participant not found or inactive")
    }
    
    // 권한 확인
    if participant.Role == RoleViewer {
        return fmt.Errorf("viewer cannot edit code")
    }
    
    // 코드 변경 적용
    if err := cm.applyCodeChange(session.SharedState, codeChange); err != nil {
        return err
    }
    
    session.LastActivity = time.Now()
    
    // 다른 참가자들에게 변경사항 브로드캐스트
    cm.broadcastToSessionExcept(sessionID, userID, Message{
        Type: "code_changed",
        Data: map[string]interface{}{
            "user_id": userID,
            "change":  codeChange,
        },
        Timestamp: time.Now().Unix(),
        ID:        generateMessageID(),
    })
    
    return nil
}

// 커서 위치 업데이트
func (cm *CollaborationManager) UpdateCursor(sessionID, userID string, position CursorPos) error {
    cm.sessionsMu.Lock()
    defer cm.sessionsMu.Unlock()
    
    session, exists := cm.sessions[sessionID]
    if !exists {
        return fmt.Errorf("session not found")
    }
    
    session.SharedState.CursorPositions[userID] = position
    session.LastActivity = time.Now()
    
    // 다른 참가자들에게 커서 위치 브로드캐스트
    cm.broadcastToSessionExcept(sessionID, userID, Message{
        Type: "cursor_moved",
        Data: map[string]interface{}{
            "user_id":  userID,
            "position": position,
        },
        Timestamp: time.Now().Unix(),
        ID:        generateMessageID(),
    })
    
    return nil
}

// 세션에 브로드캐스트
func (cm *CollaborationManager) broadcastToSession(sessionID string, message Message) {
    session, exists := cm.sessions[sessionID]
    if !exists {
        return
    }
    
    for _, participant := range session.Participants {
        if participant.IsActive && participant.ConnectionID != "" {
            if conn := cm.wsHandler.getConnection(participant.ConnectionID); conn != nil {
                cm.wsHandler.sendMessage(conn, message)
            }
        }
    }
}

// 특정 사용자 제외하고 브로드캐스트
func (cm *CollaborationManager) broadcastToSessionExcept(sessionID, exceptUserID string, message Message) {
    session, exists := cm.sessions[sessionID]
    if !exists {
        return
    }
    
    for userID, participant := range session.Participants {
        if userID != exceptUserID && participant.IsActive && participant.ConnectionID != "" {
            if conn := cm.wsHandler.getConnection(participant.ConnectionID); conn != nil {
                cm.wsHandler.sendMessage(conn, message)
            }
        }
    }
}
```

### 클라이언트 협업 기능

```typescript
// collaboration.ts
interface CollaborationClient {
    sessionId: string;
    userId: string;
    role: ParticipantRole;
    wsClient: ClaudeWebSocket;
    codeEditor: CodeEditor;
    participants: Map<string, Participant>;
}

class CollaborationClient {
    constructor(sessionId: string, userId: string, wsClient: ClaudeWebSocket) {
        this.sessionId = sessionId;
        this.userId = userId;
        this.wsClient = wsClient;
        this.participants = new Map();
        
        this.setupEventHandlers();
    }
    
    private setupEventHandlers() {
        // 참가자 입장/퇴장
        this.wsClient.on('participant_joined', (data) => {
            this.addParticipant(data.user_id, data.role);
        });
        
        this.wsClient.on('participant_left', (data) => {
            this.removeParticipant(data.user_id);
        });
        
        // 코드 변경
        this.wsClient.on('code_changed', (data) => {
            this.applyCodeChange(data.change);
            this.showUserIndicator(data.user_id, data.change.position);
        });
        
        // 커서 이동
        this.wsClient.on('cursor_moved', (data) => {
            this.updateRemoteCursor(data.user_id, data.position);
        });
        
        // 세션 상태 동기화
        this.wsClient.on('session_state', (data) => {
            this.syncSessionState(data);
        });
        
        // 코드 에디터 이벤트
        this.codeEditor.onDidChangeModelContent((e) => {
            this.handleLocalCodeChange(e);
        });
        
        this.codeEditor.onDidChangeCursorPosition((e) => {
            this.handleCursorMove(e.position);
        });
    }
    
    // 로컬 코드 변경 처리
    private handleLocalCodeChange(event: any) {
        const changes = event.changes.map(change => ({
            range: change.range,
            text: change.text,
            rangeLength: change.rangeLength
        }));
        
        // 서버에 변경사항 전송
        this.wsClient.send('code_change', {
            session_id: this.sessionId,
            changes: changes,
            timestamp: Date.now()
        });
    }
    
    // 원격 코드 변경 적용
    private applyCodeChange(change: CodeChange) {
        // 로컬 이벤트 핸들러 임시 비활성화
        this.codeEditor.suspendEvents(() => {
            this.codeEditor.applyEdits([{
                range: change.range,
                text: change.text
            }]);
        });
    }
    
    // 커서 이동 처리
    private handleCursorMove(position: Position) {
        // 다른 사용자들에게 커서 위치 전송
        this.wsClient.send('cursor_update', {
            session_id: this.sessionId,
            position: {
                line: position.lineNumber,
                column: position.column
            }
        });
    }
    
    // 원격 커서 업데이트
    private updateRemoteCursor(userId: string, position: CursorPos) {
        const participant = this.participants.get(userId);
        if (!participant) return;
        
        // 커서 표시 업데이트
        this.codeEditor.updateRemoteCursor(userId, {
            lineNumber: position.line,
            column: position.column
        }, participant.color);
    }
    
    // 참가자 추가
    private addParticipant(userId: string, role: ParticipantRole) {
        const color = this.generateUserColor(userId);
        
        this.participants.set(userId, {
            userId,
            role,
            color,
            isActive: true
        });
        
        // UI 업데이트
        this.updateParticipantsList();
        this.showNotification(`${userId}님이 세션에 참가했습니다.`);
    }
    
    // 참가자 제거
    private removeParticipant(userId: string) {
        this.participants.delete(userId);
        this.codeEditor.removeRemoteCursor(userId);
        
        // UI 업데이트
        this.updateParticipantsList();
        this.showNotification(`${userId}님이 세션을 떠났습니다.`);
    }
    
    // 사용자 색상 생성
    private generateUserColor(userId: string): string {
        const colors = [
            '#FF6B6B', '#4ECDC4', '#45B7D1', '#96CEB4', 
            '#FFEAA7', '#DDA0DD', '#98D8C8', '#F7DC6F'
        ];
        
        let hash = 0;
        for (let i = 0; i < userId.length; i++) {
            hash = userId.charCodeAt(i) + ((hash << 5) - hash);
        }
        
        return colors[Math.abs(hash) % colors.length];
    }
}

// React Hook으로 협업 기능 제공
export const useCollaboration = (sessionId: string, userId: string) => {
    const [participants, setParticipants] = useState<Map<string, Participant>>(new Map());
    const [isOwner, setIsOwner] = useState(false);
    const collaborationRef = useRef<CollaborationClient | null>(null);
    
    useEffect(() => {
        const wsClient = new ClaudeWebSocket();
        const collaboration = new CollaborationClient(sessionId, userId, wsClient);
        
        collaborationRef.current = collaboration;
        
        // 협업 세션 입장
        wsClient.send('join_collaboration', {
            session_id: sessionId,
            user_id: userId
        });
        
        return () => {
            // 세션 떠나기
            wsClient.send('leave_collaboration', {
                session_id: sessionId,
                user_id: userId
            });
            
            wsClient.disconnect();
        };
    }, [sessionId, userId]);
    
    const inviteUser = useCallback((targetUserId: string, role: ParticipantRole) => {
        if (collaborationRef.current) {
            collaborationRef.current.wsClient.send('invite_user', {
                session_id: sessionId,
                target_user_id: targetUserId,
                role: role
            });
        }
    }, [sessionId]);
    
    const removeUser = useCallback((targetUserId: string) => {
        if (collaborationRef.current && isOwner) {
            collaborationRef.current.wsClient.send('remove_user', {
                session_id: sessionId,
                target_user_id: targetUserId
            });
        }
    }, [sessionId, isOwner]);
    
    return {
        participants,
        isOwner,
        inviteUser,
        removeUser
    };
};
```

## 📁 파일 관리

### 실시간 파일 공유

```go
// file_manager.go
type FileManager struct {
    storage    FileStorage
    wsHandler  *WebSocketHandler
    sessions   map[string]*FileSession
    sessionsMu sync.RWMutex
}

type FileSession struct {
    SessionID   string
    Files       map[string]*SharedFile
    Subscribers map[string]bool  // Connection IDs
}

type SharedFile struct {
    ID          string    `json:"id"`
    Name        string    `json:"name"`
    Path        string    `json:"path"`
    ContentType string    `json:"content_type"`
    Size        int64     `json:"size"`
    Content     []byte    `json:"-"`
    UploadedBy  string    `json:"uploaded_by"`
    UploadedAt  time.Time `json:"uploaded_at"`
    IsShared    bool      `json:"is_shared"`
}

func (fm *FileManager) SaveFile(sessionID, filename, contentType string, data []byte) (string, error) {
    fileID := generateFileID()
    
    // 파일 저장
    file := &SharedFile{
        ID:          fileID,
        Name:        filename,
        ContentType: contentType,
        Size:        int64(len(data)),
        Content:     data,
        UploadedAt:  time.Now(),
        IsShared:    true,
    }
    
    // 스토리지에 저장
    if err := fm.storage.SaveFile(fileID, data); err != nil {
        return "", err
    }
    
    // 세션에 파일 추가
    fm.sessionsMu.Lock()
    session, exists := fm.sessions[sessionID]
    if !exists {
        session = &FileSession{
            SessionID:   sessionID,
            Files:       make(map[string]*SharedFile),
            Subscribers: make(map[string]bool),
        }
        fm.sessions[sessionID] = session
    }
    
    session.Files[fileID] = file
    fm.sessionsMu.Unlock()
    
    // 세션 참가자들에게 알림
    fm.notifyFileUpload(sessionID, file)
    
    return fileID, nil
}

func (fm *FileManager) GetFile(fileID string) (*SharedFile, error) {
    // 파일 메타데이터 조회
    for _, session := range fm.sessions {
        if file, exists := session.Files[fileID]; exists {
            // 컨텐츠 로드
            content, err := fm.storage.LoadFile(fileID)
            if err != nil {
                return nil, err
            }
            
            file.Content = content
            return file, nil
        }
    }
    
    return nil, fmt.Errorf("file not found")
}

func (fm *FileManager) ShareFileToSession(fileID, targetSessionID string) error {
    var sourceFile *SharedFile
    
    // 원본 파일 찾기
    fm.sessionsMu.RLock()
    for _, session := range fm.sessions {
        if file, exists := session.Files[fileID]; exists {
            sourceFile = file
            break
        }
    }
    fm.sessionsMu.RUnlock()
    
    if sourceFile == nil {
        return fmt.Errorf("source file not found")
    }
    
    // 대상 세션에 파일 추가
    fm.sessionsMu.Lock()
    targetSession, exists := fm.sessions[targetSessionID]
    if !exists {
        targetSession = &FileSession{
            SessionID:   targetSessionID,
            Files:       make(map[string]*SharedFile),
            Subscribers: make(map[string]bool),
        }
        fm.sessions[targetSessionID] = targetSession
    }
    
    // 파일 복사
    sharedFile := *sourceFile
    sharedFile.IsShared = true
    targetSession.Files[fileID] = &sharedFile
    
    fm.sessionsMu.Unlock()
    
    // 대상 세션 참가자들에게 알림
    fm.notifyFileShared(targetSessionID, &sharedFile)
    
    return nil
}

func (fm *FileManager) notifyFileUpload(sessionID string, file *SharedFile) {
    message := Message{
        Type: "file_uploaded",
        Data: map[string]interface{}{
            "file_id":      file.ID,
            "name":         file.Name,
            "content_type": file.ContentType,
            "size":         file.Size,
            "uploaded_by":  file.UploadedBy,
            "uploaded_at":  file.UploadedAt,
            "is_shared":    file.IsShared,
        },
        Timestamp: time.Now().Unix(),
        ID:        generateMessageID(),
    }
    
    fm.broadcastToSession(sessionID, message)
}

func (fm *FileManager) notifyFileShared(sessionID string, file *SharedFile) {
    message := Message{
        Type: "file_shared",
        Data: map[string]interface{}{
            "file_id":     file.ID,
            "name":        file.Name,
            "content_type": file.ContentType,
            "size":        file.Size,
            "shared_by":   file.UploadedBy,
            "shared_at":   time.Now(),
        },
        Timestamp: time.Now().Unix(),
        ID:        generateMessageID(),
    }
    
    fm.broadcastToSession(sessionID, message)
}
```

### 클라이언트 파일 처리

```typescript
// file-manager.ts
interface FileUploadProgress {
    fileId: string;
    filename: string;
    progress: number;
    speed: number;
    remaining: number;
}

class ClientFileManager {
    private wsClient: ClaudeWebSocket;
    private uploadQueue: File[] = [];
    private activeUploads: Map<string, FileUploadProgress> = new Map();
    
    constructor(wsClient: ClaudeWebSocket) {
        this.wsClient = wsClient;
        this.setupEventHandlers();
    }
    
    private setupEventHandlers() {
        this.wsClient.on('file_uploaded', (data) => {
            this.handleFileUploaded(data);
        });
        
        this.wsClient.on('file_shared', (data) => {
            this.handleFileShared(data);
        });
        
        this.wsClient.on('upload_progress', (data) => {
            this.updateUploadProgress(data);
        });
    }
    
    // 파일 업로드 (청크 단위)
    async uploadFile(file: File, sessionId?: string): Promise<string> {
        const fileId = this.generateFileId();
        const chunkSize = 64 * 1024; // 64KB 청크
        const totalChunks = Math.ceil(file.size / chunkSize);
        
        // 업로드 진행 상황 초기화
        this.activeUploads.set(fileId, {
            fileId,
            filename: file.name,
            progress: 0,
            speed: 0,
            remaining: file.size
        });
        
        // 업로드 시작 알림
        this.wsClient.send('upload_start', {
            file_id: fileId,
            filename: file.name,
            content_type: file.type,
            size: file.size,
            total_chunks: totalChunks,
            session_id: sessionId
        });
        
        // 청크 업로드
        for (let chunkIndex = 0; chunkIndex < totalChunks; chunkIndex++) {
            const start = chunkIndex * chunkSize;
            const end = Math.min(start + chunkSize, file.size);
            const chunk = file.slice(start, end);
            
            const chunkData = await this.fileToBase64(chunk);
            
            this.wsClient.send('upload_chunk', {
                file_id: fileId,
                chunk_index: chunkIndex,
                data: chunkData,
                is_last: chunkIndex === totalChunks - 1
            });
            
            // 진행 상황 업데이트
            const progress = this.activeUploads.get(fileId);
            if (progress) {
                progress.progress = ((chunkIndex + 1) / totalChunks) * 100;
                progress.remaining = file.size - end;
                this.activeUploads.set(fileId, progress);
                
                this.onUploadProgress?.(progress);
            }
            
            // 업로드 속도 제한 (선택사항)
            await this.sleep(10);
        }
        
        return fileId;
    }
    
    // 대용량 파일 업로드 (FormData 사용)
    async uploadLargeFile(file: File, sessionId?: string): Promise<string> {
        const formData = new FormData();
        formData.append('file', file);
        formData.append('session_id', sessionId || '');
        
        const response = await fetch('/api/v1/files/upload', {
            method: 'POST',
            body: formData,
            headers: {
                'Authorization': `Bearer ${this.getAuthToken()}`
            }
        });
        
        if (!response.ok) {
            throw new Error('File upload failed');
        }
        
        const result = await response.json();
        return result.file_id;
    }
    
    // 파일 다운로드
    async downloadFile(fileId: string): Promise<Blob> {
        const response = await fetch(`/api/v1/files/${fileId}`, {
            headers: {
                'Authorization': `Bearer ${this.getAuthToken()}`
            }
        });
        
        if (!response.ok) {
            throw new Error('File download failed');
        }
        
        return response.blob();
    }
    
    // 파일 공유
    async shareFile(fileId: string, targetSessionId: string): Promise<void> {
        this.wsClient.send('share_file', {
            file_id: fileId,
            target_session_id: targetSessionId
        });
    }
    
    // 파일 삭제
    async deleteFile(fileId: string): Promise<void> {
        this.wsClient.send('delete_file', {
            file_id: fileId
        });
    }
    
    private async fileToBase64(file: Blob): Promise<string> {
        return new Promise((resolve, reject) => {
            const reader = new FileReader();
            reader.onload = () => {
                const result = reader.result as string;
                resolve(result.split(',')[1]); // base64 데이터만 추출
            };
            reader.onerror = reject;
            reader.readAsDataURL(file);
        });
    }
    
    private sleep(ms: number): Promise<void> {
        return new Promise(resolve => setTimeout(resolve, ms));
    }
    
    private generateFileId(): string {
        return Date.now() + '-' + Math.random().toString(36).substr(2, 9);
    }
    
    private getAuthToken(): string {
        return localStorage.getItem('auth_token') || '';
    }
    
    // 이벤트 핸들러
    onUploadProgress?: (progress: FileUploadProgress) => void;
    onFileUploaded?: (fileData: any) => void;
    onFileShared?: (fileData: any) => void;
    
    private handleFileUploaded(data: any) {
        this.activeUploads.delete(data.file_id);
        this.onFileUploaded?.(data);
    }
    
    private handleFileShared(data: any) {
        this.onFileShared?.(data);
    }
    
    private updateUploadProgress(data: any) {
        const progress = this.activeUploads.get(data.file_id);
        if (progress) {
            Object.assign(progress, data);
            this.onUploadProgress?.(progress);
        }
    }
}

// React 컴포넌트에서 사용
const FileUploadComponent: React.FC = () => {
    const [files, setFiles] = useState<any[]>([]);
    const [uploadProgress, setUploadProgress] = useState<Map<string, FileUploadProgress>>(new Map());
    const fileManagerRef = useRef<ClientFileManager | null>(null);
    
    useEffect(() => {
        const wsClient = new ClaudeWebSocket();
        const fileManager = new ClientFileManager(wsClient);
        
        fileManager.onUploadProgress = (progress) => {
            setUploadProgress(prev => new Map(prev.set(progress.fileId, progress)));
        };
        
        fileManager.onFileUploaded = (fileData) => {
            setFiles(prev => [...prev, fileData]);
            setUploadProgress(prev => {
                const newMap = new Map(prev);
                newMap.delete(fileData.file_id);
                return newMap;
            });
        };
        
        fileManagerRef.current = fileManager;
        
        return () => {
            wsClient.disconnect();
        };
    }, []);
    
    const handleFileSelect = async (event: React.ChangeEvent<HTMLInputElement>) => {
        const selectedFiles = Array.from(event.target.files || []);
        
        for (const file of selectedFiles) {
            try {
                await fileManagerRef.current?.uploadFile(file);
            } catch (error) {
                console.error('File upload failed:', error);
            }
        }
    };
    
    return (
        <div className="file-upload-container">
            <input
                type="file"
                multiple
                onChange={handleFileSelect}
                className="file-input"
            />
            
            {/* 업로드 진행 상황 */}
            {Array.from(uploadProgress.values()).map(progress => (
                <div key={progress.fileId} className="upload-progress">
                    <span>{progress.filename}</span>
                    <div className="progress-bar">
                        <div 
                            className="progress-fill"
                            style={{ width: `${progress.progress}%` }}
                        />
                    </div>
                    <span>{progress.progress.toFixed(1)}%</span>
                </div>
            ))}
            
            {/* 업로드된 파일 목록 */}
            <div className="file-list">
                {files.map(file => (
                    <div key={file.file_id} className="file-item">
                        <span>{file.name}</span>
                        <span>{formatFileSize(file.size)}</span>
                        <button onClick={() => fileManagerRef.current?.downloadFile(file.file_id)}>
                            다운로드
                        </button>
                        <button onClick={() => fileManagerRef.current?.shareFile(file.file_id, 'target-session')}>
                            공유
                        </button>
                    </div>
                ))}
            </div>
        </div>
    );
};
```

---

**다음 단계**: [WebSocket 프로토콜 명세](../api/websocket-protocol.md)에서 상세한 통신 프로토콜을 확인하세요.