# TX02_S02: Web Interface Integration

## 태스크 정보
- **태스크 ID**: TX02_S02_Web_Interface_Integration
- **스프린트**: S02_M03_Advanced_Integration
- **우선순위**: High
- **상태**: PENDING
- **담당자**: Claude Code
- **예상 소요시간**: 10시간
- **실제 소요시간**: TBD

## 목표
Claude 세션과 웹 인터페이스 간의 실시간 통합을 구현하여 브라우저에서 직접 Claude와 상호작용할 수 있는 시스템을 구축합니다.

## 상세 요구사항

### 1. 실시간 WebSocket 스트림 통합
```go
type WebSocketStreamHandler interface {
    // Claude 세션을 WebSocket으로 연결
    ConnectSession(sessionID string, ws *websocket.Conn) error
    
    // 스트림 메시지를 WebSocket으로 전달
    StreamToWebSocket(sessionID string, messages <-chan Message) error
    
    // WebSocket 입력을 Claude 세션으로 전달
    ForwardToSession(sessionID string, input string) error
    
    // 연결 상태 관리
    GetActiveConnections() map[string]*websocket.Conn
    CloseConnection(sessionID string) error
}
```

### 2. 세션 제어 API 확장
```go
type WebSessionController interface {
    // 웹 기반 세션 생성
    CreateWebSession(config WebSessionConfig) (*WebSession, error)
    
    // 실시간 세션 상태 동기화
    SyncSessionState(sessionID string) error
    
    // 세션 명령어 실행
    ExecuteCommand(sessionID string, command WebCommand) error
    
    // 파일 업로드/다운로드
    UploadFile(sessionID string, file WebFile) error
    DownloadFile(sessionID string, filename string) (*WebFile, error)
}

type WebSessionConfig struct {
    WorkspaceID   string            `json:"workspace_id"`
    SystemPrompt  string            `json:"system_prompt"`
    MaxTurns      int               `json:"max_turns"`
    Tools         []string          `json:"tools"`
    UISettings    map[string]string `json:"ui_settings"`
    Collaborative bool              `json:"collaborative"`
}
```

### 3. 실시간 상태 동기화
- **세션 상태**: 실시간 세션 상태 변경 알림
- **메시지 스트림**: 실시간 메시지 전달
- **프로그레스 표시**: 작업 진행률 실시간 업데이트
- **에러 알림**: 즉시 에러 상황 전달

### 4. 멀티유저 협업 지원
- **세션 공유**: 여러 사용자가 동일 세션 관찰
- **권한 관리**: 읽기/쓰기 권한 분리
- **충돌 해결**: 동시 입력 시 충돌 처리

## 구현 계획

### 1. WebSocket 핸들러 확장
```go
// internal/api/websocket/claude_stream.go
type ClaudeStreamHandler struct {
    sessionManager *claude.SessionManager
    connections    map[string][]*websocket.Conn
    messageRouter  *MessageRouter
    authValidator  *AuthValidator
}
```

### 2. 실시간 메시지 라우터
```go
// internal/api/websocket/message_router.go
type MessageRouter struct {
    routes        map[MessageType][]RouteHandler
    middleware    []MiddlewareFunc
    errorHandler  ErrorHandler
}

type RouteHandler func(ctx *MessageContext) error
type MiddlewareFunc func(next RouteHandler) RouteHandler
```

### 3. 웹 세션 컨트롤러
```go
// internal/api/handlers/web_session.go
type WebSessionHandler struct {
    sessionController *WebSessionController
    fileManager      *FileManager
    authService      *auth.Service
    rateLimiter      *RateLimiter
}
```

## API 엔드포인트

### 1. REST API 확장
```
POST   /api/v1/web-sessions              # 웹 세션 생성
GET    /api/v1/web-sessions              # 웹 세션 목록
GET    /api/v1/web-sessions/{id}         # 웹 세션 조회
PATCH  /api/v1/web-sessions/{id}         # 웹 세션 설정 수정
DELETE /api/v1/web-sessions/{id}         # 웹 세션 종료

POST   /api/v1/web-sessions/{id}/execute # 명령어 실행
POST   /api/v1/web-sessions/{id}/upload  # 파일 업로드
GET    /api/v1/web-sessions/{id}/download/{file} # 파일 다운로드

GET    /api/v1/web-sessions/{id}/share   # 세션 공유 링크 생성
POST   /api/v1/web-sessions/{id}/join    # 공유 세션 참여
```

### 2. WebSocket 프로토콜
```json
{
  "type": "session.connect",
  "session_id": "sess_123",
  "auth_token": "..."
}

{
  "type": "session.message",
  "session_id": "sess_123",
  "message": {
    "type": "text",
    "content": "Hello Claude!",
    "timestamp": "2025-07-22T08:00:00Z"
  }
}

{
  "type": "session.status",
  "session_id": "sess_123",
  "status": "active",
  "participants": ["user1", "user2"]
}
```

## 파일 구조
```
internal/api/
├── websocket/
│   ├── claude_stream.go      # Claude 스트림 WebSocket 핸들러
│   ├── message_router.go     # 메시지 라우팅
│   ├── connection_manager.go # 연결 관리
│   └── auth_middleware.go    # WebSocket 인증
├── handlers/
│   ├── web_session.go        # 웹 세션 REST API
│   ├── file_manager.go       # 파일 관리
│   └── collaboration.go      # 협업 기능
└── middleware/
    ├── rate_limiter.go       # 속도 제한
    └── session_validator.go  # 세션 검증
```

## 테스트 계획

### 1. 단위 테스트
- WebSocket 연결/해제 테스트
- 메시지 라우팅 로직 테스트
- 권한 검증 테스트

### 2. 통합 테스트
- Claude 세션 ↔ WebSocket 연동 테스트
- 멀티유저 동시 접속 테스트
- 파일 업로드/다운로드 테스트

### 3. E2E 테스트
- 브라우저 기반 E2E 테스트
- 실시간 협업 시나리오 테스트
- 네트워크 단절 복구 테스트

## 검증 기준
- [ ] 100개 이상 동시 WebSocket 연결 지원
- [ ] 메시지 전달 지연시간 < 50ms
- [ ] 파일 업로드 성공률 99% 이상
- [ ] 세션 상태 동기화 정확도 100%
- [ ] 멀티유저 협업 기능 정상 동작
- [ ] 네트워크 단절 시 자동 재연결

## 보안 고려사항
- WebSocket 연결 시 JWT 토큰 검증
- 세션 접근 권한 검증
- 파일 업로드 크기 및 타입 제한
- XSS/CSRF 공격 방지

## 의존성
- internal/api/websocket.go (기존)
- internal/claude/session_manager.go (기존)
- internal/auth/service.go (기존)

## 위험 요소
1. **연결 관리 복잡성**: 다수 WebSocket 연결 관리 어려움
2. **메모리 누수**: 연결이 제대로 해제되지 않을 경우
3. **동시성 이슈**: 멀티유저 환경에서 데이터 경합

## 완료 조건
1. 모든 API 엔드포인트 구현 완료
2. WebSocket 프로토콜 구현 완료
3. 멀티유저 협업 기능 구현 완료
4. 단위/통합/E2E 테스트 통과
5. 보안 검증 완료
6. 성능 벤치마크 통과