---
task_id: T02_S02_WebSocket_Streaming
sprint_id: S02_M06_PTY_Streaming
milestone_id: M06
title: 실시간 WebSocket 스트리밍 구현
type: implementation
complexity: High
status: pending
assignee: unassigned
created: 2025-08-05T10:00:00+0900
last_updated: 2025-08-05T10:00:00+0900
depends_on: [T01_S02_PTY_Session_Manager]
blocks: [T03_S02_Terminal_Snapshot, T06_S02_Flow_Control]
epic: PTY_Streaming_System
---

# Task: 실시간 WebSocket 스트리밍 구현

## Task Summary
PTY 세션의 입출력을 WebSocket을 통해 실시간으로 스트리밍하는 시스템을 구현합니다. 바이너리 데이터 처리, UTF-8 인코딩 관리, 연결 안정성을 핵심으로 하는 고성능 스트리밍 인프라를 제공합니다.

## Acceptance Criteria

### 기능 요구사항
- [ ] PTY 입출력의 실시간 WebSocket 스트리밍
- [ ] 바이너리 데이터 안전한 처리 및 전송
- [ ] UTF-8 인코딩/디코딩 자동 관리
- [ ] 클라이언트-서버 양방향 통신 지원
- [ ] 연결 끊김 감지 및 자동 재연결 메커니즘
- [ ] 메시지 큐잉 및 버퍼링 시스템
- [ ] 동시 다중 WebSocket 연결 처리

### 성능 요구사항
- [ ] WebSocket 메시지 전송 지연 < 100ms
- [ ] 초당 1000개 메시지 처리 능력
- [ ] 연결당 메모리 사용량 < 10MB
- [ ] CPU 사용률 정상 부하 시 < 15%
- [ ] 네트워크 대역폭 효율적 사용

### 안정성 요구사항
- [ ] 네트워크 단절 시 메시지 손실 방지
- [ ] 연결 복구 시 상태 동기화
- [ ] 메모리 누수 없는 연결 관리
- [ ] 8시간 연속 스트리밍 안정성

## Implementation Details

### 1. WebSocket 스트리밍 관리자 구조

```go
// internal/websocket/stream_manager.go
type StreamManager struct {
    connections map[string]*Connection
    hub         *Hub
    upgrader    websocket.Upgrader
    config      *StreamConfig
    metrics     *StreamMetrics
    mutex       sync.RWMutex
}

type Connection struct {
    ID        string
    SessionID string
    Conn      *websocket.Conn
    Send      chan []byte
    Receive   chan []byte
    Context   context.Context
    Cancel    context.CancelFunc
    LastPing  time.Time
    IsActive  bool
}

type Hub struct {
    connections map[*Connection]bool
    broadcast   chan []byte
    register    chan *Connection
    unregister  chan *Connection
    stopCh      chan struct{}
}
```

### 2. 실시간 메시지 처리 시스템

```go
// WebSocket 메시지 처리 인터페이스
type StreamInterface interface {
    StartStream(ctx context.Context, sessionID string, conn *websocket.Conn) error
    StopStream(connectionID string) error
    SendMessage(connectionID string, data []byte) error
    BroadcastToSession(sessionID string, data []byte) error
    GetActiveConnections(sessionID string) []*Connection
}

// 메시지 타입 정의
type MessageType int
const (
    TextMessage MessageType = iota
    BinaryMessage
    ControlMessage
    ErrorMessage
)

type StreamMessage struct {
    Type      MessageType
    SessionID string
    Data      []byte
    Timestamp time.Time
    Sequence  uint64
}
```

### 3. PTY-WebSocket 브릿지 시스템

```go
// PTY 출력을 WebSocket으로 스트리밍
func (sm *StreamManager) startPTYToWebSocketStream(session *PTYSession, conn *Connection) {
    go func() {
        defer conn.Cancel()
        
        buffer := make([]byte, 4096)
        for {
            select {
            case <-conn.Context.Done():
                return
            default:
                n, err := session.PTY.Read(buffer)
                if err != nil {
                    sm.handleReadError(conn, err)
                    return
                }
                
                // UTF-8 유효성 검사 및 변환
                data := sm.ensureUTF8(buffer[:n])
                
                select {
                case conn.Send <- data:
                case <-time.After(time.Second):
                    // 전송 타임아웃 처리
                    sm.handleSendTimeout(conn)
                    return
                }
            }
        }
    }()
}

// WebSocket 입력을 PTY로 전달
func (sm *StreamManager) startWebSocketToPTYStream(session *PTYSession, conn *Connection) {
    go func() {
        defer conn.Cancel()
        
        for {
            select {
            case <-conn.Context.Done():
                return
            case data := <-conn.Receive:
                _, err := session.PTY.Write(data)
                if err != nil {
                    sm.handleWriteError(conn, err)
                    return
                }
            }
        }
    }()
}
```

### 4. 연결 관리 및 재연결 시스템

```go
// 연결 상태 모니터링 및 재연결
type ReconnectionManager struct {
    connections map[string]*ReconnectionState
    config      *ReconnectionConfig
    mutex       sync.RWMutex
}

type ReconnectionState struct {
    ConnectionID    string
    SessionID      string
    LastSeen       time.Time
    RetryCount     int
    BackoffDelay   time.Duration
    MessageBuffer  []StreamMessage
}

type ReconnectionConfig struct {
    MaxRetries      int
    InitialDelay    time.Duration
    MaxDelay        time.Duration
    BackoffFactor   float64
    BufferSize      int
    BufferTimeout   time.Duration
}
```

### 5. 메시지 큐잉 및 버퍼링

```go
// 메시지 버퍼링 시스템
type MessageBuffer struct {
    sessionID   string
    messages    []StreamMessage
    maxSize     int
    mutex       sync.RWMutex
    lastFlush   time.Time
    flushTicker *time.Ticker
}

func (mb *MessageBuffer) Add(msg StreamMessage) error {
    mb.mutex.Lock()
    defer mb.mutex.Unlock()
    
    if len(mb.messages) >= mb.maxSize {
        // 오래된 메시지 제거 (FIFO)
        mb.messages = mb.messages[1:]
    }
    
    mb.messages = append(mb.messages, msg)
    return nil
}

func (mb *MessageBuffer) FlushTo(conn *Connection) error {
    mb.mutex.RLock()
    messages := make([]StreamMessage, len(mb.messages))
    copy(messages, mb.messages)
    mb.mutex.RUnlock()
    
    for _, msg := range messages {
        select {
        case conn.Send <- msg.Data:
        case <-time.After(100 * time.Millisecond):
            return fmt.Errorf("flush timeout")
        }
    }
    
    return nil
}
```

## 파일 구조

```
internal/websocket/
├── stream_manager.go       # 메인 스트리밍 관리자
├── connection.go          # WebSocket 연결 관리
├── hub.go                 # 연결 허브 및 브로드캐스트
├── message.go             # 메시지 처리 및 타입 정의
├── buffer.go              # 메시지 버퍼링 시스템
├── reconnection.go        # 재연결 관리
├── encoding.go            # UTF-8 인코딩 처리
├── metrics.go             # 성능 메트릭 수집
└── config.go              # 설정 관리

internal/websocket/handler/
├── websocket_handler.go   # HTTP WebSocket 핸들러
├── middleware.go          # WebSocket 미들웨어
└── auth.go               # WebSocket 인증

internal/websocket/test/
├── stream_manager_test.go
├── connection_test.go
├── message_test.go
└── integration_test.go
```

## 핵심 구현 사항

### 1. 고성능 메시지 처리
- 고루틴 풀을 사용한 동시 처리
- 제로 카피 최적화를 통한 메모리 효율성
- 배치 처리를 통한 네트워크 오버헤드 감소

### 2. 안전한 데이터 전송
- UTF-8 유효성 검사 및 자동 복구
- 바이너리 데이터의 Base64 인코딩 옵션
- 메시지 무결성 검증

### 3. 연결 안정성 보장
- 하트비트 메커니즘을 통한 연결 상태 감지
- 지수 백오프를 사용한 재연결 알고리즘
- 메시지 순서 보장 및 중복 제거

### 4. 리소스 관리
- 연결별 메모리 제한 및 모니터링
- 가비지 컬렉션 최적화
- 고루틴 리크 방지

## Dependencies

### 필수 패키지
```go
import (
    "context"
    "sync"
    "time"
    "encoding/json"
    "unicode/utf8"
    
    // WebSocket 라이브러리
    "github.com/gorilla/websocket"
    
    // 로깅
    "github.com/sirupsen/logrus"
    
    // 메트릭 수집
    "github.com/prometheus/client_golang/prometheus"
)
```

## HTTP 핸들러 통합

```go
// cmd/api/handlers/websocket.go
func (h *WebSocketHandler) HandleConnection(w http.ResponseWriter, r *http.Request) {
    // 인증 확인
    sessionID := r.URL.Query().Get("session_id")
    if !h.auth.ValidateSession(sessionID) {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    
    // WebSocket 업그레이드
    conn, err := h.upgrader.Upgrade(w, r, nil)
    if err != nil {
        h.logger.Error("WebSocket upgrade failed", err)
        return
    }
    
    // 스트리밍 시작
    if err := h.streamManager.StartStream(r.Context(), sessionID, conn); err != nil {
        h.logger.Error("Failed to start stream", err)
        conn.Close()
        return
    }
}
```

## 테스트 계획

### 단위 테스트
- WebSocket 연결 생성/종료 테스트
- 메시지 전송/수신 테스트
- UTF-8 인코딩/디코딩 테스트
- 버퍼링 및 큐잉 테스트

### 통합 테스트
- PTY-WebSocket 브릿지 테스트
- 다중 연결 동시 처리 테스트
- 재연결 시나리오 테스트

### 성능 테스트
- 메시지 처리량 벤치마크
- 메모리 사용량 프로파일링
- 네트워크 지연 시간 측정

## Risk Mitigation

### 주요 위험 요소
1. **메모리 누수**: 닫히지 않은 WebSocket 연결
2. **메시지 손실**: 네트워크 불안정 시 데이터 손실
3. **성능 저하**: 대량 메시지 처리 시 병목

### 완화 방안
- 연결 타임아웃 및 자동 정리
- 메시지 버퍼링 및 재전송 메커니즘
- 백프레셔 제어 및 플로우 컨트롤

## Definition of Done
- [ ] WebSocket 스트리밍 시스템 구현 완료
- [ ] PTY 세션과의 연동 테스트 성공
- [ ] 성능 요구사항 달성 확인
- [ ] 단위 테스트 및 통합 테스트 통과
- [ ] 코드 리뷰 완료
- [ ] 재연결 기능 검증 완료
- [ ] 메모리 누수 테스트 통과

## Notes
- WebSocket 라이브러리는 Gorilla WebSocket 사용 권장
- 메시지 압축 기능은 향후 추가 고려
- 클러스터 환경에서의 스케일링은 별도 태스크로 분리
- 보안 강화를 위한 WSS(WebSocket Secure) 지원 필수