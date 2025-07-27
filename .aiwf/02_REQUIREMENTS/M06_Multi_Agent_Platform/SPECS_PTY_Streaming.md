# SPECS: PTY Streaming

## 개요

실시간 터미널 스트리밍 시스템 기술 사양

## PTY 세션 관리

### PTY 세션 구조
```go
type PTYSession struct {
    ID          string
    AgentID     string
    ContainerID string
    PTY         *os.File
    Input       io.WriteCloser
    Output      io.ReadCloser
    ErrorOutput io.ReadCloser
    Size        WindowSize
}
```

### Docker PTY 통합
- 컨테이너별 PTY 어태치
- 터미널 크기 동기화
- 프로세스 생명주기 관리

## WebSocket 스트리밍

### 프로토콜
```javascript
// 클라이언트 → 서버
{
  type: "pty_input",
  agentId: "agent-123",
  data: "ls -la\n"
}

// 서버 → 클라이언트
{
  type: "pty_output",
  agentId: "agent-123",
  data: "\x1b[34mfile.txt\x1b[0m\n"
}

// 터미널 크기 조정
{
  type: "pty_resize",
  agentId: "agent-123",
  cols: 120,
  rows: 40
}
```

### 스트리밍 최적화
- 백프레셔 처리
- 버퍼링 메커니즘
- 자동 재연결

## 터미널 스냅샷

### 스냅샷 구조
```go
type TerminalSnapshot struct {
    AgentID   string
    Content   []string  // 마지막 N줄
    Cursor    CursorPosition
    Timestamp time.Time
    IsActive  bool
}
```

### 캡처 최적화
- 1초 간격 주기적 캡처
- 순환 버퍼 사용
- ANSI 코드 파싱

## 성능 요구사항

- PTY 응답 시간 < 50ms (P95)
- WebSocket 재연결 < 1초
- 스냅샷 생성 시간 < 10ms
- ANSI 코드 호환성 95% 이상

## 파일 구조
```
internal/
├── pty/
│   ├── session.go           # PTY 세션 관리
│   └── manager.go           # PTY 매니저
├── websocket/
│   ├── pty_handler.go       # WebSocket 핸들러
│   └── pty_stream.go        # 스트리밍 로직
└── snapshot/
    ├── terminal_snapshot.go # 스냅샷 생성
    └── manager.go           # 스냅샷 매니저
```