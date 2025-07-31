---
task_id: T02_S02
sprint_sequence_id: S02
status: in_progress
complexity: High
last_updated: 2025-07-31T08:25:00+0900
---

# Task: 실시간 WebSocket 스트리밍 구현

## Description
PTY 세션과 웹 클라이언트 간의 실시간 양방향 통신을 위한 WebSocket 스트리밍 시스템을 구현합니다. 이 시스템은 PTY 입출력을 WebSocket을 통해 실시간으로 스트리밍하고, 바이너리 데이터 처리, UTF-8 인코딩 관리, 백프레셜 및 플로우 컨트롤을 포함합니다.

## Goal / Objectives
- PTY 입출력의 실시간 WebSocket 스트리밍 구현
- 바이너리 데이터 처리 및 UTF-8 인코딩 관리
- 백프레셜 및 플로우 컨트롤 시스템
- 연결 끊김 감지 및 자동 재연결 메커니즘
- WebSocket 연결 상태 관리 및 모니터링

## Acceptance Criteria
- [ ] WebSocket 핸들러 및 업그레이드 로직 구현
- [ ] PTY 세션과 WebSocket 간 데이터 브리징 시스템
- [ ] 바이너리 데이터 처리 및 UTF-8 인코딩/디코딩
- [ ] 백프레셜 및 플로우 컨트롤 메커니즘
- [ ] 연결 상태 관리 및 헬스체크 시스템
- [ ] 에러 처리 및 자동 복구 메커니즘
- [ ] WebSocket 메시지 큐잉 및 버퍼링
- [ ] 성능 최적화 (지연 시간 < 100ms)

## Subtasks
- [ ] WebSocket 핸들러 인터페이스 정의
- [ ] WebSocket 업그레이드 및 연결 관리
- [ ] PTY-WebSocket 데이터 브리지 구현
- [ ] 바이너리 데이터 및 UTF-8 처리
- [ ] 백프레셜 및 플로우 컨트롤 시스템
- [ ] 연결 상태 모니터링 및 헬스체크
- [ ] 메시지 큐잉 및 버퍼 관리
- [ ] 단위 테스트 및 통합 테스트 작성

## Technical Guidelines

### 주요 기술 스택
- **WebSocket**: Gorilla WebSocket 라이브러리
- **HTTP**: Go 표준 http 패키지
- **동시성**: Go 고루틴 및 채널
- **인코딩**: UTF-8, Base64 (바이너리 데이터)
- **테스트**: Go 표준 테스트 + WebSocket 테스트 도구

### 아키텍처 설계
```go
// WebSocket 스트리밍 인터페이스
type PTYWebSocketStreamer interface {
    HandleWebSocket(w http.ResponseWriter, r *http.Request) error
    AttachPTYSession(sessionID string) error
    DetachPTYSession() error
    SendMessage(data []byte) error
    Close() error
    IsConnected() bool
}

// WebSocket 연결 관리자
type WebSocketManager struct {
    connections map[string]*WebSocketConnection
    ptyManager  PTYSessionManagement
    mu          sync.RWMutex
}

// 개별 WebSocket 연결
type WebSocketConnection struct {
    conn        *websocket.Conn
    ptySession  PTYSession
    sendChan    chan []byte
    receiveChan chan []byte
    closeChan   chan struct{}
    isConnected bool
}
```

### 데이터 플로우
1. **WebSocket 연결**: 클라이언트 연결 업그레이드
2. **PTY 연결**: 기존 PTY 세션에 연결
3. **양방향 스트리밍**: PTY ↔ WebSocket 실시간 데이터 전송
4. **에러 처리**: 연결 끊김 감지 및 정리

## Implementation Notes

### WebSocket 메시지 타입
```json
{
  "type": "data",           // 터미널 데이터
  "data": "base64_encoded", // 실제 데이터 (Base64 인코딩)
  "timestamp": "2025-07-31T08:25:00Z"
}

{
  "type": "resize",         // 터미널 크기 변경
  "width": 80,
  "height": 24
}

{
  "type": "ping",           // 연결 상태 확인
  "timestamp": "2025-07-31T08:25:00Z"
}
```

### 성능 요구사항
- WebSocket 메시지 지연 < 100ms
- 동시 연결 최대 1000개 지원
- 메모리 사용량 연결당 < 10MB
- CPU 사용률 정상 부하 시 < 15%

### 에러 처리 전략
- WebSocket 연결 실패 시 재시도 로직
- PTY 세션 연결 끊김 감지
- 메시지 전송 실패 시 큐잉 및 재전송
- 상세한 로깅 및 메트릭 수집

## Dependencies
- PTY 세션 관리 시스템 (T01_S02 완료)
- Gorilla WebSocket 라이브러리
- 기존 Docker Agent Integration
- HTTP 서버 인프라

## Output Log
*(This section is populated as work progresses on the task)*

[2025-07-31 08:25] 태스크 생성됨 - T02_S02_WebSocket_Streaming 시작