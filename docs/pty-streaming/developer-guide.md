# PTY Streaming 개발자 가이드

## 목차
1. [개요](#개요)
2. [아키텍처](#아키텍처)
3. [핵심 컴포넌트](#핵심-컴포넌트)
4. [빠른 시작](#빠른-시작)
5. [API 사용법](#api-사용법)
6. [WebSocket 통신](#websocket-통신)
7. [성능 최적화](#성능-최적화)
8. [에러 처리](#에러-처리)
9. [모니터링](#모니터링)
10. [문제 해결](#문제-해결)

## 개요

PTY Streaming 시스템은 Docker 컨테이너 내에서 실행되는 터미널 세션을 웹 브라우저로 스트리밍하는 고성능 솔루션입니다.

### 주요 특징
- **실시간 스트리밍**: WebSocket을 통한 저지연 터미널 출력
- **플로우 제어**: 백프레셔 및 동적 스로틀링
- **스냅샷 지원**: 터미널 상태 저장 및 복원
- **성능 최적화**: 메모리 풀링, GC 튜닝, I/O 최적화
- **자동 재연결**: 네트워크 장애 시 자동 복구

## 아키텍처

### 시스템 구성도

```
┌─────────────┐     WebSocket      ┌──────────────┐
│   Browser   │◄──────────────────►│  API Server  │
└─────────────┘                    └──────────────┘
                                           │
                                           ▼
                                   ┌──────────────┐
                                   │ PTY Manager  │
                                   └──────────────┘
                                           │
                                           ▼
                                   ┌──────────────┐
                                   │ Docker SDK   │
                                   └──────────────┘
                                           │
                                           ▼
                                   ┌──────────────┐
                                   │  Container   │
                                   │   (PTY)      │
                                   └──────────────┘
```

### 주요 패키지 구조

```
internal/
├── pty/            # PTY 세션 관리
│   ├── session.go  # 세션 라이프사이클
│   ├── manager.go  # 세션 매니저
│   └── docker.go   # Docker 통합
├── websocket/      # WebSocket 스트리밍
│   ├── stream.go   # 스트림 관리
│   ├── backpressure.go # 백프레셔 제어
│   └── ratelimit.go    # 레이트 리미팅
├── snapshot/       # 터미널 스냅샷
│   ├── capture.go  # 스냅샷 캡처
│   └── ansi.go     # ANSI 파싱
├── flow/           # 플로우 제어
│   ├── controller.go # 플로우 컨트롤러
│   ├── throttler.go  # 동적 스로틀링
│   └── monitor.go    # 모니터링
└── docker/         # Docker 통합
    ├── pty_integration.go # PTY 통합
    └── container_monitor.go # 컨테이너 모니터링
```

## 핵심 컴포넌트

### 1. PTY Session Manager

PTY 세션의 생명주기를 관리합니다.

```go
// PTY 세션 생성
manager := pty.NewPTYManager(config)
session, err := manager.CreateSession(containerID, &pty.PTYConfig{
    Rows: 24,
    Cols: 80,
    Term: "xterm-256color",
    Shell: "/bin/bash",
})

// 세션 입력 처리
err = session.Write([]byte("ls -la\n"))

// 세션 출력 읽기
output := make([]byte, 4096)
n, err := session.Read(output)

// 세션 종료
err = session.Terminate()
```

### 2. WebSocket Streaming

WebSocket을 통한 실시간 스트리밍을 처리합니다.

```go
// 스트림 매니저 생성
streamManager := websocket.NewStreamManager(config)

// WebSocket 연결 처리
conn, err := streamManager.CreateConnection(ws)

// 스트리밍 세션 생성
streamSession, err := streamManager.CreateSession(ptySessionID)

// 연결과 세션 연결
err = streamManager.AttachConnection(conn.ID, streamSession.ID)

// 데이터 전송
err = streamManager.SendToSession(sessionID, data)
```

### 3. Terminal Snapshot

터미널 상태를 캡처하고 복원합니다.

```go
// 스냅샷 매니저 생성
snapshotManager := snapshot.NewSnapshotManager(config)

// 스냅샷 생성
snap, err := snapshotManager.CaptureSnapshot(sessionID)

// 스냅샷 저장
err = snapshotManager.SaveSnapshot(snap)

// 스냅샷 복원
err = snapshotManager.RestoreSnapshot(snapshotID, sessionID)
```

### 4. Flow Control

플로우 제어 및 백프레셔 관리를 담당합니다.

```go
// 플로우 컨트롤러 생성
flowController := flow.NewFlowController(config)

// 백프레셔 확인
level := flowController.GetBackpressureLevel(connectionID)

// 동적 스로틀링 적용
err = flowController.ApplyThrottle(connectionID, level)

// 플로우 통계 조회
stats := flowController.GetStatistics()
```

## 빠른 시작

### 1. 서버 시작

```bash
# 개발 모드
make dev

# 프로덕션 모드
make build
./bin/aicli-api --config config.yaml
```

### 2. PTY 세션 생성

```bash
curl -X POST http://localhost:8080/api/v1/pty/sessions \
  -H "Content-Type: application/json" \
  -d '{
    "container_id": "abc123",
    "config": {
      "rows": 24,
      "cols": 80,
      "term": "xterm-256color"
    }
  }'
```

### 3. WebSocket 연결

```javascript
// JavaScript 클라이언트 예제
const ws = new WebSocket('ws://localhost:8080/api/v1/ws/stream/session-id');

ws.onopen = () => {
  console.log('Connected to PTY stream');
};

ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  if (message.type === 'output') {
    const output = atob(message.data);
    terminal.write(output);
  }
};

// 입력 전송
terminal.onData((data) => {
  ws.send(JSON.stringify({
    type: 'input',
    data: btoa(data)
  }));
});
```

## API 사용법

### REST API

#### PTY 세션 생성
```http
POST /api/v1/pty/sessions
```

**요청 본문:**
```json
{
  "container_id": "container-123",
  "config": {
    "rows": 24,
    "cols": 80,
    "term": "xterm-256color",
    "shell": "/bin/bash",
    "working_dir": "/workspace",
    "environment": {
      "TERM": "xterm-256color",
      "LANG": "en_US.UTF-8"
    }
  }
}
```

**응답:**
```json
{
  "id": "sess-1234567890",
  "container_id": "container-123",
  "status": "active",
  "created_at": "2024-01-01T12:00:00Z",
  "config": { ... }
}
```

#### PTY 세션 목록 조회
```http
GET /api/v1/pty/sessions?status=active
```

#### PTY 세션 종료
```http
DELETE /api/v1/pty/sessions/{sessionId}
```

#### 터미널 크기 조정
```http
PATCH /api/v1/pty/sessions/{sessionId}
```

**요청 본문:**
```json
{
  "rows": 30,
  "cols": 120
}
```

### WebSocket API

#### 연결 URL
```
ws://localhost:8080/api/v1/ws/stream/{sessionId}
```

#### 메시지 형식

**클라이언트 → 서버 (입력):**
```json
{
  "type": "input",
  "data": "bHMgLWxhCg==" // base64 encoded
}
```

**서버 → 클라이언트 (출력):**
```json
{
  "type": "output",
  "data": "dG90YWwgMTI4Cg==", // base64 encoded
  "timestamp": "2024-01-01T12:00:00Z"
}
```

**서버 → 클라이언트 (에러):**
```json
{
  "type": "error",
  "error": "Session terminated",
  "timestamp": "2024-01-01T12:00:00Z"
}
```

## WebSocket 통신

### 연결 관리

```go
// 서버 측 구현
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
    // WebSocket 업그레이드
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        return
    }
    
    // 스트림 연결 생성
    streamConn, err := streamManager.CreateConnection(conn)
    if err != nil {
        return
    }
    
    // 세션 연결
    sessionID := r.PathValue("sessionId")
    err = streamManager.AttachConnection(streamConn.ID, sessionID)
}
```

### 메시지 처리

```javascript
// 클라이언트 측 구현
class PTYClient {
  constructor(sessionId) {
    this.sessionId = sessionId;
    this.ws = null;
    this.reconnectAttempts = 0;
  }
  
  connect() {
    const url = `ws://localhost:8080/api/v1/ws/stream/${this.sessionId}`;
    this.ws = new WebSocket(url);
    
    this.ws.onopen = () => {
      console.log('Connected');
      this.reconnectAttempts = 0;
    };
    
    this.ws.onmessage = (event) => {
      const message = JSON.parse(event.data);
      this.handleMessage(message);
    };
    
    this.ws.onerror = (error) => {
      console.error('WebSocket error:', error);
    };
    
    this.ws.onclose = () => {
      this.handleDisconnect();
    };
  }
  
  handleMessage(message) {
    switch(message.type) {
      case 'output':
        const output = atob(message.data);
        this.onOutput(output);
        break;
      case 'error':
        this.onError(message.error);
        break;
    }
  }
  
  sendInput(data) {
    if (this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({
        type: 'input',
        data: btoa(data)
      }));
    }
  }
  
  handleDisconnect() {
    // 자동 재연결 로직
    if (this.reconnectAttempts < 5) {
      setTimeout(() => {
        this.reconnectAttempts++;
        this.connect();
      }, Math.min(1000 * Math.pow(2, this.reconnectAttempts), 10000));
    }
  }
}
```

## 성능 최적화

### 1. 메모리 풀링

```go
// 버퍼 풀 사용
var bufferPool = sync.Pool{
    New: func() interface{} {
        return make([]byte, 4096)
    },
}

func processData() {
    buffer := bufferPool.Get().([]byte)
    defer bufferPool.Put(buffer)
    
    // 버퍼 사용
    n, err := reader.Read(buffer)
    // ...
}
```

### 2. 백프레셔 처리

```go
// 백프레셔 레벨에 따른 처리
level := backpressure.GetLevel()
switch level {
case BackpressureLow:
    // 정상 처리
case BackpressureMedium:
    // 처리 속도 감소
    time.Sleep(10 * time.Millisecond)
case BackpressureHigh:
    // 스로틀링 적용
    throttler.Apply(connectionID)
case BackpressureCritical:
    // 일시 중지
    session.Pause()
}
```

### 3. I/O 최적화

```go
// 배치 처리
type BatchWriter struct {
    writer    io.Writer
    buffer    []byte
    threshold int
    ticker    *time.Ticker
}

func (bw *BatchWriter) Write(p []byte) (n int, err error) {
    bw.buffer = append(bw.buffer, p...)
    
    if len(bw.buffer) >= bw.threshold {
        return bw.flush()
    }
    
    return len(p), nil
}

func (bw *BatchWriter) flush() (int, error) {
    n, err := bw.writer.Write(bw.buffer)
    bw.buffer = bw.buffer[:0]
    return n, err
}
```

## 에러 처리

### 에러 타입

```go
type PTYError struct {
    Code    string
    Message string
    Details map[string]interface{}
}

// 에러 코드
const (
    ErrSessionNotFound     = "SESSION_NOT_FOUND"
    ErrContainerNotRunning = "CONTAINER_NOT_RUNNING"
    ErrConnectionClosed    = "CONNECTION_CLOSED"
    ErrBackpressureLimit   = "BACKPRESSURE_LIMIT"
    ErrResourceExhausted   = "RESOURCE_EXHAUSTED"
)
```

### 에러 처리 예제

```go
func handlePTYSession(sessionID string) error {
    session, err := manager.GetSession(sessionID)
    if err != nil {
        if errors.Is(err, ErrSessionNotFound) {
            return &PTYError{
                Code:    ErrSessionNotFound,
                Message: "PTY session not found",
                Details: map[string]interface{}{
                    "session_id": sessionID,
                },
            }
        }
        return err
    }
    
    // 세션 처리
    return nil
}
```

## 모니터링

### 메트릭 수집

```go
// Prometheus 메트릭
var (
    ptySessionsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "pty_sessions_total",
            Help: "Total number of PTY sessions created",
        },
        []string{"status"},
    )
    
    wsConnectionsActive = prometheus.NewGauge(
        prometheus.GaugeOpts{
            Name: "ws_connections_active",
            Help: "Number of active WebSocket connections",
        },
    )
    
    streamBytesTransferred = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "stream_bytes_transferred_total",
            Help: "Total bytes transferred through streams",
        },
        []string{"direction"},
    )
)
```

### 헬스 체크

```go
func healthCheck() http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        health := map[string]interface{}{
            "status": "healthy",
            "checks": map[string]bool{
                "docker":    dockerClient.Ping() == nil,
                "database":  db.Ping() == nil,
                "websocket": streamManager.IsHealthy(),
            },
            "metrics": map[string]interface{}{
                "active_sessions":    manager.GetActiveSessionCount(),
                "active_connections": streamManager.GetConnectionCount(),
                "memory_usage":       getMemoryUsage(),
            },
        }
        
        json.NewEncoder(w).Encode(health)
    }
}
```

## 문제 해결

### 일반적인 문제

#### 1. WebSocket 연결 실패

**증상:** WebSocket 연결이 즉시 끊어짐

**원인:**
- CORS 설정 문제
- 프록시 설정 오류
- 인증 실패

**해결:**
```go
// CORS 설정
upgrader := websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool {
        // 개발 환경에서는 모든 origin 허용
        return true
    },
}
```

#### 2. 높은 메모리 사용량

**증상:** 메모리 사용량이 지속적으로 증가

**원인:**
- 메모리 누수
- 버퍼 크기 과다
- 종료되지 않은 세션

**해결:**
```go
// 세션 타임아웃 설정
func cleanupIdleSessions() {
    ticker := time.NewTicker(5 * time.Minute)
    defer ticker.Stop()
    
    for range ticker.C {
        sessions := manager.GetAllSessions()
        for _, session := range sessions {
            if session.GetIdleTime() > 30*time.Minute {
                session.Terminate()
            }
        }
    }
}
```

#### 3. 터미널 출력 깨짐

**증상:** ANSI 이스케이프 시퀀스가 제대로 처리되지 않음

**원인:**
- 인코딩 문제
- ANSI 파서 오류

**해결:**
```javascript
// xterm.js 설정
const term = new Terminal({
  convertEol: true,
  cursorBlink: true,
  fontSize: 14,
  fontFamily: 'Menlo, Monaco, "Courier New", monospace',
  theme: {
    background: '#000000',
    foreground: '#ffffff'
  }
});

// UTF-8 디코더 사용
const decoder = new TextDecoder('utf-8');
ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  if (message.type === 'output') {
    const bytes = Uint8Array.from(atob(message.data), c => c.charCodeAt(0));
    const text = decoder.decode(bytes);
    term.write(text);
  }
};
```

### 디버깅 팁

#### 1. 로그 레벨 설정

```yaml
# config.yaml
logging:
  level: debug
  format: json
  output: stdout
  modules:
    pty: debug
    websocket: info
    docker: warn
```

#### 2. 트레이싱

```go
// OpenTelemetry 트레이싱
import "go.opentelemetry.io/otel"

func processRequest(ctx context.Context) {
    tracer := otel.Tracer("pty-streaming")
    ctx, span := tracer.Start(ctx, "process-request")
    defer span.End()
    
    // 처리 로직
}
```

#### 3. 프로파일링

```go
// pprof 엔드포인트
import _ "net/http/pprof"

func main() {
    // 디버그 서버
    go func() {
        log.Println(http.ListenAndServe("localhost:6060", nil))
    }()
    
    // 메인 서버
    // ...
}
```

```bash
# CPU 프로파일링
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

# 메모리 프로파일링
go tool pprof http://localhost:6060/debug/pprof/heap

# 고루틴 덤프
curl http://localhost:6060/debug/pprof/goroutine?debug=2
```

## 보안 고려사항

### 1. 입력 검증

```go
// 명령어 필터링
func validateInput(input []byte) error {
    // 위험한 명령어 차단
    dangerous := []string{"rm -rf", "dd if=", "mkfs"}
    inputStr := string(input)
    
    for _, cmd := range dangerous {
        if strings.Contains(inputStr, cmd) {
            return fmt.Errorf("dangerous command detected")
        }
    }
    
    return nil
}
```

### 2. 리소스 제한

```go
// 컨테이너 리소스 제한
config := &container.Config{
    // ...
}

hostConfig := &container.HostConfig{
    Resources: container.Resources{
        Memory:     512 * 1024 * 1024, // 512MB
        MemorySwap: 512 * 1024 * 1024,
        CPUShares:  512,
        PidsLimit:  &pidsLimit,
    },
}
```

### 3. 세션 격리

```go
// 사용자별 세션 격리
type SessionManager struct {
    sessions map[string]map[string]*PTYSession // userID -> sessionID -> session
    mutex    sync.RWMutex
}

func (sm *SessionManager) GetSession(userID, sessionID string) (*PTYSession, error) {
    sm.mutex.RLock()
    defer sm.mutex.RUnlock()
    
    userSessions, exists := sm.sessions[userID]
    if !exists {
        return nil, ErrUnauthorized
    }
    
    session, exists := userSessions[sessionID]
    if !exists {
        return nil, ErrSessionNotFound
    }
    
    return session, nil
}
```

## 추가 자료

- [API Reference](/docs/api/pty-streaming-openapi.yaml)
- [Architecture Documentation](/docs/architecture.md)
- [Performance Tuning Guide](/docs/performance.md)
- [Security Best Practices](/docs/security.md)
- [Troubleshooting Guide](/docs/troubleshooting.md)

## 라이센스

MIT License - 자세한 내용은 [LICENSE](../../LICENSE) 파일을 참조하세요.