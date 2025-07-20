# API 설계 명세 (Go 구현)

## 🌐 개요

Go로 구현된 RESTful API와 실시간 통신을 위한 WebSocket/SSE 엔드포인트 설계입니다.

## 🏗️ API 서버 구조

### 1. 서버 초기화

```go
// internal/api/server.go
package api

import (
    "context"
    "net/http"
    "time"
    
    "github.com/gin-gonic/gin"
    "github.com/gorilla/websocket"
    "go.uber.org/zap"
)

type Server struct {
    router       *gin.Engine
    config       *Config
    logger       *zap.Logger
    
    // 서비스
    authService      AuthService
    workspaceService WorkspaceService
    taskService      TaskService
    dockerService    DockerService
    claudeService    ClaudeService
    
    // WebSocket
    wsHub        *WebSocketHub
    wsUpgrader   websocket.Upgrader
    
    // 미들웨어
    rateLimiter  RateLimiter
    authMiddleware AuthMiddleware
}

type Config struct {
    Port            string
    ReadTimeout     time.Duration
    WriteTimeout    time.Duration
    MaxRequestSize  int64
    AllowedOrigins  []string
    TrustedProxies  []string
    EnableMetrics   bool
    EnableProfiling bool
}

func NewServer(config *Config, logger *zap.Logger) *Server {
    s := &Server{
        config: config,
        logger: logger,
        wsHub:  NewWebSocketHub(),
        wsUpgrader: websocket.Upgrader{
            ReadBufferSize:  1024,
            WriteBufferSize: 1024,
            CheckOrigin: func(r *http.Request) bool {
                return checkOrigin(r, config.AllowedOrigins)
            },
        },
    }
    
    s.setupRouter()
    return s
}

func (s *Server) setupRouter() {
    r := gin.New()
    
    // 기본 미들웨어
    r.Use(gin.Recovery())
    r.Use(s.loggerMiddleware())
    r.Use(s.corsMiddleware())
    r.Use(s.securityMiddleware())
    
    // 신뢰할 수 있는 프록시 설정
    r.SetTrustedProxies(s.config.TrustedProxies)
    
    // 헬스체크
    r.GET("/health", s.healthCheck)
    r.GET("/ready", s.readinessCheck)
    
    // API v1 라우트
    v1 := r.Group("/api/v1")
    {
        // 인증 라우트 (public)
        auth := v1.Group("/auth")
        {
            auth.POST("/login", s.login)
            auth.POST("/register", s.register)
            auth.POST("/refresh", s.refreshToken)
            auth.POST("/logout", s.authMiddleware.Require(), s.logout)
        }
        
        // 보호된 라우트
        protected := v1.Group("/", s.authMiddleware.Require())
        {
            // 워크스페이스
            protected.GET("/workspaces", s.listWorkspaces)
            protected.POST("/workspaces", s.createWorkspace)
            protected.GET("/workspaces/:id", s.getWorkspace)
            protected.PUT("/workspaces/:id", s.updateWorkspace)
            protected.DELETE("/workspaces/:id", s.deleteWorkspace)
            
            // 작업
            protected.GET("/tasks", s.listTasks)
            protected.POST("/tasks", s.createTask)
            protected.GET("/tasks/:id", s.getTask)
            protected.POST("/tasks/:id/cancel", s.cancelTask)
            protected.GET("/tasks/:id/logs", s.getTaskLogs)
            protected.GET("/tasks/:id/stream", s.streamTaskLogs)
            
            // 통계
            protected.GET("/stats", s.getStats)
            protected.GET("/metrics", s.getMetrics)
        }
    }
    
    // WebSocket
    r.GET("/ws", s.authMiddleware.Require(), s.handleWebSocket)
    
    // 정적 파일 (프론트엔드)
    r.Static("/static", "./static")
    r.NoRoute(func(c *gin.Context) {
        c.File("./static/index.html")
    })
    
    s.router = r
}

func (s *Server) Start() error {
    // WebSocket 허브 시작
    go s.wsHub.Run()
    
    // HTTP 서버 설정
    srv := &http.Server{
        Addr:         ":" + s.config.Port,
        Handler:      s.router,
        ReadTimeout:  s.config.ReadTimeout,
        WriteTimeout: s.config.WriteTimeout,
        MaxHeaderBytes: 1 << 20, // 1MB
    }
    
    s.logger.Info("Starting API server", zap.String("port", s.config.Port))
    return srv.ListenAndServe()
}
```

### 2. 미들웨어

```go
// internal/api/middleware.go
package api

import (
    "fmt"
    "net/http"
    "strings"
    "time"
    
    "github.com/gin-gonic/gin"
    "github.com/golang-jwt/jwt/v5"
    "go.uber.org/zap"
    "golang.org/x/time/rate"
)

// 로깅 미들웨어
func (s *Server) loggerMiddleware() gin.HandlerFunc {
    return gin.LoggerWithConfig(gin.LoggerConfig{
        Formatter: func(param gin.LogFormatterParams) string {
            s.logger.Info("HTTP Request",
                zap.String("method", param.Method),
                zap.String("path", param.Path),
                zap.Int("status", param.StatusCode),
                zap.Duration("latency", param.Latency),
                zap.String("client_ip", param.ClientIP),
                zap.String("user_agent", param.Request.UserAgent()),
            )
            return ""
        },
    })
}

// CORS 미들웨어
func (s *Server) corsMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        origin := c.Request.Header.Get("Origin")
        
        // 허용된 오리진 확인
        allowed := false
        for _, allowedOrigin := range s.config.AllowedOrigins {
            if allowedOrigin == "*" || allowedOrigin == origin {
                allowed = true
                break
            }
        }
        
        if allowed {
            c.Header("Access-Control-Allow-Origin", origin)
            c.Header("Access-Control-Allow-Credentials", "true")
            c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
            c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        }
        
        if c.Request.Method == "OPTIONS" {
            c.AbortWithStatus(http.StatusNoContent)
            return
        }
        
        c.Next()
    }
}

// 보안 헤더 미들웨어
func (s *Server) securityMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Header("X-Content-Type-Options", "nosniff")
        c.Header("X-Frame-Options", "DENY")
        c.Header("X-XSS-Protection", "1; mode=block")
        c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
        c.Header("Content-Security-Policy", "default-src 'self'")
        c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
        c.Next()
    }
}

// 인증 미들웨어
type AuthMiddleware struct {
    jwtSecret []byte
    logger    *zap.Logger
}

func (m *AuthMiddleware) Require() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Authorization 헤더에서 토큰 추출
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
            c.Abort()
            return
        }
        
        tokenString := strings.TrimPrefix(authHeader, "Bearer ")
        if tokenString == authHeader {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format"})
            c.Abort()
            return
        }
        
        // JWT 검증
        token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
            if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
                return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
            }
            return m.jwtSecret, nil
        })
        
        if err != nil || !token.Valid {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
            c.Abort()
            return
        }
        
        // 클레임 추출
        claims, ok := token.Claims.(jwt.MapClaims)
        if !ok {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
            c.Abort()
            return
        }
        
        // 컨텍스트에 사용자 정보 저장
        c.Set("user_id", claims["sub"])
        c.Set("user_email", claims["email"])
        c.Set("user_role", claims["role"])
        
        c.Next()
    }
}

// Rate Limiting 미들웨어
type RateLimiter struct {
    limiters sync.Map // key -> *rate.Limiter
    rate     int
    burst    int
}

func NewRateLimiter(r int, b int) *RateLimiter {
    return &RateLimiter{
        rate:  r,
        burst: b,
    }
}

func (rl *RateLimiter) Middleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        key := c.ClientIP() // IP 기반 rate limiting
        
        limiter, _ := rl.limiters.LoadOrStore(key, rate.NewLimiter(rate.Limit(rl.rate), rl.burst))
        
        if !limiter.(*rate.Limiter).Allow() {
            c.JSON(http.StatusTooManyRequests, gin.H{
                "error": "rate limit exceeded",
            })
            c.Abort()
            return
        }
        
        c.Next()
    }
}
```

### 3. 워크스페이스 핸들러

```go
// internal/api/workspace.go
package api

import (
    "net/http"
    
    "github.com/gin-gonic/gin"
)

type WorkspaceRequest struct {
    Name        string            `json:"name" binding:"required,min=1,max=100"`
    Description string            `json:"description" binding:"max=500"`
    Path        string            `json:"path" binding:"required"`
    GitURL      string            `json:"git_url" binding:"omitempty,url"`
    Branch      string            `json:"branch" binding:"omitempty"`
    Tags        []string          `json:"tags" binding:"omitempty,dive,min=1,max=50"`
    Metadata    map[string]string `json:"metadata" binding:"omitempty"`
}

type WorkspaceResponse struct {
    ID          string            `json:"id"`
    Name        string            `json:"name"`
    Description string            `json:"description"`
    Path        string            `json:"path"`
    Status      string            `json:"status"`
    GitURL      string            `json:"git_url,omitempty"`
    Branch      string            `json:"branch,omitempty"`
    Tags        []string          `json:"tags"`
    Metadata    map[string]string `json:"metadata"`
    Stats       WorkspaceStats    `json:"stats"`
    CreatedAt   time.Time         `json:"created_at"`
    UpdatedAt   time.Time         `json:"updated_at"`
}

type WorkspaceStats struct {
    TotalTasks     int `json:"total_tasks"`
    RunningTasks   int `json:"running_tasks"`
    CompletedTasks int `json:"completed_tasks"`
    FailedTasks    int `json:"failed_tasks"`
    DiskUsageMB    int `json:"disk_usage_mb"`
}

func (s *Server) listWorkspaces(c *gin.Context) {
    userID := c.GetString("user_id")
    
    // 쿼리 파라미터
    var query struct {
        Page   int    `form:"page,default=1" binding:"min=1"`
        Limit  int    `form:"limit,default=10" binding:"min=1,max=100"`
        Search string `form:"search"`
        Status string `form:"status" binding:"omitempty,oneof=active archived"`
        Sort   string `form:"sort,default=-created_at" binding:"omitempty,oneof=name -name created_at -created_at"`
    }
    
    if err := c.ShouldBindQuery(&query); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    // 서비스 호출
    workspaces, total, err := s.workspaceService.List(c, ListOptions{
        UserID: userID,
        Page:   query.Page,
        Limit:  query.Limit,
        Search: query.Search,
        Status: query.Status,
        Sort:   query.Sort,
    })
    
    if err != nil {
        s.logger.Error("Failed to list workspaces", zap.Error(err))
        c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
        return
    }
    
    // 응답
    c.JSON(http.StatusOK, gin.H{
        "data": workspaces,
        "pagination": gin.H{
            "page":  query.Page,
            "limit": query.Limit,
            "total": total,
            "pages": (total + query.Limit - 1) / query.Limit,
        },
    })
}

func (s *Server) createWorkspace(c *gin.Context) {
    userID := c.GetString("user_id")
    
    var req WorkspaceRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    // 경로 검증
    if err := s.validatePath(req.Path); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path: " + err.Error()})
        return
    }
    
    // 워크스페이스 생성
    workspace, err := s.workspaceService.Create(c, CreateWorkspaceOptions{
        UserID:      userID,
        Name:        req.Name,
        Description: req.Description,
        Path:        req.Path,
        GitURL:      req.GitURL,
        Branch:      req.Branch,
        Tags:        req.Tags,
        Metadata:    req.Metadata,
    })
    
    if err != nil {
        if err == ErrWorkspaceExists {
            c.JSON(http.StatusConflict, gin.H{"error": "workspace already exists"})
            return
        }
        s.logger.Error("Failed to create workspace", zap.Error(err))
        c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
        return
    }
    
    c.JSON(http.StatusCreated, workspace)
}

func (s *Server) getWorkspace(c *gin.Context) {
    userID := c.GetString("user_id")
    workspaceID := c.Param("id")
    
    workspace, err := s.workspaceService.Get(c, userID, workspaceID)
    if err != nil {
        if err == ErrWorkspaceNotFound {
            c.JSON(http.StatusNotFound, gin.H{"error": "workspace not found"})
            return
        }
        s.logger.Error("Failed to get workspace", zap.Error(err))
        c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
        return
    }
    
    c.JSON(http.StatusOK, workspace)
}

func (s *Server) validatePath(path string) error {
    // 경로 검증 로직
    // - 절대 경로인지 확인
    // - 존재하는 디렉토리인지 확인
    // - 접근 권한 확인
    // - 허용된 경로 내에 있는지 확인
    return nil
}
```

### 4. 작업 핸들러

```go
// internal/api/task.go
package api

import (
    "net/http"
    
    "github.com/gin-gonic/gin"
)

type TaskRequest struct {
    WorkspaceID  string            `json:"workspace_id" binding:"required,uuid"`
    Type         string            `json:"type" binding:"required,oneof=claude_chat git_operation docker_build"`
    Prompt       string            `json:"prompt" binding:"required_if=Type claude_chat"`
    Config       TaskConfig        `json:"config"`
    Priority     string            `json:"priority" binding:"omitempty,oneof=low medium high"`
    Tags         []string          `json:"tags" binding:"omitempty"`
    Metadata     map[string]string `json:"metadata" binding:"omitempty"`
}

type TaskConfig struct {
    SystemPrompt    string   `json:"system_prompt"`
    MaxTurns        int      `json:"max_turns" binding:"min=1,max=50"`
    AllowedTools    []string `json:"allowed_tools"`
    WorkingDir      string   `json:"working_dir"`
    Environment     map[string]string `json:"environment"`
    Timeout         int      `json:"timeout" binding:"min=0,max=3600"` // seconds
}

type TaskResponse struct {
    ID          string            `json:"id"`
    WorkspaceID string            `json:"workspace_id"`
    Type        string            `json:"type"`
    Status      string            `json:"status"`
    Progress    TaskProgress      `json:"progress"`
    Result      *TaskResult       `json:"result,omitempty"`
    Error       *TaskError        `json:"error,omitempty"`
    Config      TaskConfig        `json:"config"`
    Priority    string            `json:"priority"`
    Tags        []string          `json:"tags"`
    Metadata    map[string]string `json:"metadata"`
    CreatedAt   time.Time         `json:"created_at"`
    StartedAt   *time.Time        `json:"started_at,omitempty"`
    CompletedAt *time.Time        `json:"completed_at,omitempty"`
}

type TaskProgress struct {
    Current     int    `json:"current"`
    Total       int    `json:"total"`
    Percentage  int    `json:"percentage"`
    Message     string `json:"message"`
}

type TaskResult struct {
    Success bool                   `json:"success"`
    Summary string                 `json:"summary"`
    Details map[string]interface{} `json:"details"`
}

type TaskError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
    Details string `json:"details,omitempty"`
}

func (s *Server) createTask(c *gin.Context) {
    userID := c.GetString("user_id")
    
    var req TaskRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    // 워크스페이스 권한 확인
    if !s.hasWorkspaceAccess(c, userID, req.WorkspaceID) {
        c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
        return
    }
    
    // 기본값 설정
    if req.Priority == "" {
        req.Priority = "medium"
    }
    if req.Config.MaxTurns == 0 {
        req.Config.MaxTurns = 10
    }
    
    // 작업 생성
    task, err := s.taskService.Create(c, CreateTaskOptions{
        UserID:      userID,
        WorkspaceID: req.WorkspaceID,
        Type:        req.Type,
        Prompt:      req.Prompt,
        Config:      req.Config,
        Priority:    req.Priority,
        Tags:        req.Tags,
        Metadata:    req.Metadata,
    })
    
    if err != nil {
        s.logger.Error("Failed to create task", zap.Error(err))
        c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
        return
    }
    
    // 작업 실행 (비동기)
    go s.executeTask(task)
    
    c.JSON(http.StatusCreated, task)
}

func (s *Server) streamTaskLogs(c *gin.Context) {
    taskID := c.Param("id")
    userID := c.GetString("user_id")
    
    // 작업 권한 확인
    task, err := s.taskService.Get(c, taskID)
    if err != nil || !s.hasTaskAccess(c, userID, task) {
        c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
        return
    }
    
    // SSE 헤더 설정
    c.Header("Content-Type", "text/event-stream")
    c.Header("Cache-Control", "no-cache")
    c.Header("Connection", "keep-alive")
    c.Header("X-Accel-Buffering", "no") // Nginx 버퍼링 비활성화
    
    // 로그 스트림 구독
    logStream, err := s.taskService.StreamLogs(c, taskID)
    if err != nil {
        c.SSEvent("error", gin.H{"error": err.Error()})
        return
    }
    
    // 클라이언트 연결 감지
    clientGone := c.Request.Context().Done()
    
    c.Stream(func(w io.Writer) bool {
        select {
        case log, ok := <-logStream:
            if !ok {
                c.SSEvent("close", gin.H{"message": "stream closed"})
                return false
            }
            
            c.SSEvent("log", gin.H{
                "timestamp": log.Timestamp,
                "level":     log.Level,
                "message":   log.Message,
                "source":    log.Source,
            })
            
            return true
            
        case <-clientGone:
            return false
        }
    })
}
```

### 5. WebSocket 구현

```go
// internal/api/websocket.go
package api

import (
    "encoding/json"
    "net/http"
    "sync"
    "time"
    
    "github.com/gin-gonic/gin"
    "github.com/gorilla/websocket"
)

type WebSocketHub struct {
    clients    map[string]map[*WebSocketClient]bool // workspaceID -> clients
    broadcast  chan BroadcastMessage
    register   chan *WebSocketClient
    unregister chan *WebSocketClient
    mu         sync.RWMutex
}

type WebSocketClient struct {
    hub         *WebSocketHub
    conn        *websocket.Conn
    send        chan []byte
    userID      string
    workspaceID string
}

type BroadcastMessage struct {
    WorkspaceID string
    Message     []byte
}

type WSMessage struct {
    Type      string          `json:"type"`
    Action    string          `json:"action,omitempty"`
    Data      json.RawMessage `json:"data,omitempty"`
    ID        string          `json:"id,omitempty"`
    Timestamp time.Time       `json:"timestamp"`
}

func NewWebSocketHub() *WebSocketHub {
    return &WebSocketHub{
        clients:    make(map[string]map[*WebSocketClient]bool),
        broadcast:  make(chan BroadcastMessage),
        register:   make(chan *WebSocketClient),
        unregister: make(chan *WebSocketClient),
    }
}

func (h *WebSocketHub) Run() {
    for {
        select {
        case client := <-h.register:
            h.mu.Lock()
            if h.clients[client.workspaceID] == nil {
                h.clients[client.workspaceID] = make(map[*WebSocketClient]bool)
            }
            h.clients[client.workspaceID][client] = true
            h.mu.Unlock()
            
            // 연결 알림
            h.sendToWorkspace(client.workspaceID, WSMessage{
                Type:      "connection",
                Action:    "joined",
                Timestamp: time.Now(),
            })
            
        case client := <-h.unregister:
            h.mu.Lock()
            if clients, ok := h.clients[client.workspaceID]; ok {
                if _, ok := clients[client]; ok {
                    delete(clients, client)
                    close(client.send)
                    
                    // 워크스페이스에 클라이언트가 없으면 제거
                    if len(clients) == 0 {
                        delete(h.clients, client.workspaceID)
                    }
                }
            }
            h.mu.Unlock()
            
        case message := <-h.broadcast:
            h.mu.RLock()
            clients := h.clients[message.WorkspaceID]
            h.mu.RUnlock()
            
            for client := range clients {
                select {
                case client.send <- message.Message:
                default:
                    // 버퍼가 가득 찬 경우 클라이언트 제거
                    h.unregister <- client
                }
            }
        }
    }
}

func (s *Server) handleWebSocket(c *gin.Context) {
    userID := c.GetString("user_id")
    workspaceID := c.Query("workspace")
    
    if workspaceID == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "workspace parameter required"})
        return
    }
    
    // 워크스페이스 접근 권한 확인
    if !s.hasWorkspaceAccess(c, userID, workspaceID) {
        c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
        return
    }
    
    // WebSocket 업그레이드
    conn, err := s.wsUpgrader.Upgrade(c.Writer, c.Request, nil)
    if err != nil {
        s.logger.Error("WebSocket upgrade failed", zap.Error(err))
        return
    }
    
    client := &WebSocketClient{
        hub:         s.wsHub,
        conn:        conn,
        send:        make(chan []byte, 256),
        userID:      userID,
        workspaceID: workspaceID,
    }
    
    client.hub.register <- client
    
    // 고루틴으로 읽기/쓰기 처리
    go client.writePump()
    go client.readPump()
}

func (c *WebSocketClient) readPump() {
    defer func() {
        c.hub.unregister <- c
        c.conn.Close()
    }()
    
    c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
    c.conn.SetPongHandler(func(string) error {
        c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
        return nil
    })
    
    for {
        var msg WSMessage
        err := c.conn.ReadJSON(&msg)
        if err != nil {
            if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
                log.Printf("websocket error: %v", err)
            }
            break
        }
        
        // 메시지 처리
        c.handleMessage(msg)
    }
}

func (c *WebSocketClient) writePump() {
    ticker := time.NewTicker(54 * time.Second)
    defer func() {
        ticker.Stop()
        c.conn.Close()
    }()
    
    for {
        select {
        case message, ok := <-c.send:
            c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
            if !ok {
                c.conn.WriteMessage(websocket.CloseMessage, []byte{})
                return
            }
            
            c.conn.WriteMessage(websocket.TextMessage, message)
            
        case <-ticker.C:
            c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
            if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
                return
            }
        }
    }
}

func (c *WebSocketClient) handleMessage(msg WSMessage) {
    switch msg.Type {
    case "command":
        // 명령 처리
        c.handleCommand(msg)
    case "subscribe":
        // 구독 처리
        c.handleSubscribe(msg)
    case "ping":
        // Pong 응답
        c.send <- []byte(`{"type":"pong"}`)
    }
}
```

### 6. 응답 포맷

```go
// internal/api/response.go
package api

import (
    "net/http"
    
    "github.com/gin-gonic/gin"
)

// 표준 응답 구조체
type Response struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data,omitempty"`
    Error   *ErrorInfo  `json:"error,omitempty"`
    Meta    *MetaInfo   `json:"meta,omitempty"`
}

type ErrorInfo struct {
    Code    string                 `json:"code"`
    Message string                 `json:"message"`
    Details map[string]interface{} `json:"details,omitempty"`
}

type MetaInfo struct {
    RequestID string    `json:"request_id"`
    Timestamp time.Time `json:"timestamp"`
    Version   string    `json:"version"`
}

// 성공 응답
func (s *Server) success(c *gin.Context, data interface{}) {
    c.JSON(http.StatusOK, Response{
        Success: true,
        Data:    data,
        Meta: &MetaInfo{
            RequestID: c.GetString("request_id"),
            Timestamp: time.Now(),
            Version:   s.config.Version,
        },
    })
}

// 에러 응답
func (s *Server) error(c *gin.Context, statusCode int, code, message string, details ...map[string]interface{}) {
    var detailsMap map[string]interface{}
    if len(details) > 0 {
        detailsMap = details[0]
    }
    
    c.JSON(statusCode, Response{
        Success: false,
        Error: &ErrorInfo{
            Code:    code,
            Message: message,
            Details: detailsMap,
        },
        Meta: &MetaInfo{
            RequestID: c.GetString("request_id"),
            Timestamp: time.Now(),
            Version:   s.config.Version,
        },
    })
}

// 페이지네이션 응답
type PaginatedResponse struct {
    Success    bool        `json:"success"`
    Data       interface{} `json:"data"`
    Pagination Pagination  `json:"pagination"`
    Meta       *MetaInfo   `json:"meta,omitempty"`
}

type Pagination struct {
    Page       int  `json:"page"`
    Limit      int  `json:"limit"`
    Total      int  `json:"total"`
    TotalPages int  `json:"total_pages"`
    HasPrev    bool `json:"has_prev"`
    HasNext    bool `json:"has_next"`
}

func (s *Server) paginated(c *gin.Context, data interface{}, page, limit, total int) {
    totalPages := (total + limit - 1) / limit
    
    c.JSON(http.StatusOK, PaginatedResponse{
        Success: true,
        Data:    data,
        Pagination: Pagination{
            Page:       page,
            Limit:      limit,
            Total:      total,
            TotalPages: totalPages,
            HasPrev:    page > 1,
            HasNext:    page < totalPages,
        },
        Meta: &MetaInfo{
            RequestID: c.GetString("request_id"),
            Timestamp: time.Now(),
            Version:   s.config.Version,
        },
    })
}
```

### 7. 에러 처리

```go
// internal/api/errors.go
package api

import (
    "errors"
    "net/http"
)

// 에러 코드 정의
const (
    ErrCodeValidation       = "VALIDATION_ERROR"
    ErrCodeUnauthorized     = "UNAUTHORIZED"
    ErrCodeForbidden        = "FORBIDDEN"
    ErrCodeNotFound         = "NOT_FOUND"
    ErrCodeConflict         = "CONFLICT"
    ErrCodeRateLimit        = "RATE_LIMIT_EXCEEDED"
    ErrCodeInternal         = "INTERNAL_ERROR"
    ErrCodeServiceUnavailable = "SERVICE_UNAVAILABLE"
)

// 비즈니스 에러
var (
    ErrWorkspaceNotFound = errors.New("workspace not found")
    ErrWorkspaceExists   = errors.New("workspace already exists")
    ErrTaskNotFound      = errors.New("task not found")
    ErrTaskCancelled     = errors.New("task cancelled")
    ErrInvalidCredentials = errors.New("invalid credentials")
    ErrTokenExpired      = errors.New("token expired")
)

// 에러 핸들러
func (s *Server) handleError(c *gin.Context, err error) {
    switch {
    case errors.Is(err, ErrWorkspaceNotFound):
        s.error(c, http.StatusNotFound, ErrCodeNotFound, "Workspace not found")
    case errors.Is(err, ErrWorkspaceExists):
        s.error(c, http.StatusConflict, ErrCodeConflict, "Workspace already exists")
    case errors.Is(err, ErrInvalidCredentials):
        s.error(c, http.StatusUnauthorized, ErrCodeUnauthorized, "Invalid credentials")
    case errors.Is(err, ErrTokenExpired):
        s.error(c, http.StatusUnauthorized, ErrCodeUnauthorized, "Token expired")
    default:
        s.logger.Error("Unexpected error", zap.Error(err))
        s.error(c, http.StatusInternalServerError, ErrCodeInternal, "Internal server error")
    }
}
```

## 🔒 보안 기능

### API 키 인증

```go
type APIKeyAuth struct {
    store APIKeyStore
}

func (a *APIKeyAuth) Middleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        apiKey := c.GetHeader("X-API-Key")
        if apiKey == "" {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "missing API key"})
            c.Abort()
            return
        }
        
        keyInfo, err := a.store.Validate(c, apiKey)
        if err != nil {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid API key"})
            c.Abort()
            return
        }
        
        c.Set("api_key_info", keyInfo)
        c.Next()
    }
}
```

## 📊 메트릭 및 모니터링

```go
// internal/api/metrics.go
package api

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
    httpRequestsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "http_requests_total",
            Help: "Total number of HTTP requests",
        },
        []string{"method", "endpoint", "status"},
    )
    
    httpRequestDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "http_request_duration_seconds",
            Help:    "HTTP request duration in seconds",
            Buckets: prometheus.DefBuckets,
        },
        []string{"method", "endpoint"},
    )
)

func (s *Server) prometheusMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        
        c.Next()
        
        duration := time.Since(start).Seconds()
        status := fmt.Sprintf("%d", c.Writer.Status())
        
        httpRequestsTotal.WithLabelValues(c.Request.Method, c.FullPath(), status).Inc()
        httpRequestDuration.WithLabelValues(c.Request.Method, c.FullPath()).Observe(duration)
    }
}
```