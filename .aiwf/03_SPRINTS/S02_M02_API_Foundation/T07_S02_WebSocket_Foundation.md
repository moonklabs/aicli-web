---
task_id: T07_S02
task_name: WebSocket Foundation for Real-time Communication
status: pending
complexity: medium
priority: high
created_at: 2025-07-21
updated_at: 2025-07-21
sprint_id: S02_M02
---

# T07_S02: WebSocket Foundation for Real-time Communication

## 태스크 개요

실시간 통신을 위한 WebSocket 기반 인프라를 구축합니다. Claude 세션의 실시간 로그 스트리밍과 상태 업데이트를 위한 기초를 마련합니다.

## 목표

- WebSocket 서버 구현
- 연결 관리 시스템 구축
- 메시지 프로토콜 정의
- 인증된 WebSocket 연결 지원

## 수용 기준

- [ ] /ws 엔드포인트에서 WebSocket 연결 수락
- [ ] JWT 토큰 기반 WebSocket 인증
- [ ] 연결별 고유 ID 할당 및 관리
- [ ] 표준화된 메시지 포맷 구현
- [ ] 연결 상태 모니터링
- [ ] 자동 재연결 지원 (클라이언트 가이드)
- [ ] 연결 풀 관리 및 제한
- [ ] 단위 및 통합 테스트 작성

## 기술 가이드

### 주요 인터페이스 및 통합 지점

1. **WebSocket 라이브러리**: gorilla/websocket 사용
2. **라우터 통합**: `internal/server/router.go`에 WebSocket 라우트 추가
3. **인증 통합**: JWT 미들웨어와 연동
4. **이벤트 시스템**: 내부 이벤트 버스와 통합

### 구현 구조

```
internal/
├── websocket/
│   ├── hub.go           # WebSocket 허브 (연결 관리)
│   ├── client.go        # 클라이언트 연결 관리
│   ├── message.go       # 메시지 타입 정의
│   ├── handler.go       # 메시지 핸들러
│   └── auth.go          # WebSocket 인증
├── api/
│   └── handlers/
│       └── websocket.go # WebSocket 엔드포인트
└── events/
    └── bus.go           # 이벤트 버스 (내부 통신)
```

### 기존 패턴 참조

- 인증: JWT 인증 미들웨어 재사용
- 에러 처리: 표준 에러 포맷 적용
- 로깅: 기존 로깅 미들웨어 패턴 활용

## 구현 노트

### 단계별 접근법

1. WebSocket 허브 구현 (연결 관리)
2. 클라이언트 연결 처리
3. 메시지 프로토콜 정의
4. 인증 메커니즘 구현
5. 메시지 라우팅 시스템
6. 이벤트 버스 통합
7. 연결 상태 관리

### 메시지 프로토콜

```go
type Message struct {
    Type      MessageType       `json:"type"`
    ID        string           `json:"id,omitempty"`
    Channel   string           `json:"channel,omitempty"`
    Data      json.RawMessage  `json:"data"`
    Timestamp time.Time        `json:"timestamp"`
}

type MessageType string

const (
    // 시스템 메시지
    MessageTypeAuth       MessageType = "auth"
    MessageTypePing       MessageType = "ping"
    MessageTypePong       MessageType = "pong"
    MessageTypeError      MessageType = "error"
    
    // 비즈니스 메시지
    MessageTypeLog        MessageType = "log"
    MessageTypeStatus     MessageType = "status"
    MessageTypeEvent      MessageType = "event"
    MessageTypeCommand    MessageType = "command"
)
```

### 연결 관리

- 연결당 고루틴 2개 (읽기/쓰기)
- 연결 풀 크기 제한 (기본 1000)
- 유휴 연결 타임아웃 (5분)
- 핑/퐁 헬스체크 (30초 간격)

### 채널 구독

- 워크스페이스별 채널
- 세션별 채널
- 시스템 브로드캐스트 채널
- 동적 구독/구독 취소

### 보안 고려사항

- WebSocket 핸드셰이크 시 JWT 검증
- 채널별 접근 권한 확인
- Rate limiting 적용
- 메시지 크기 제한 (1MB)

## 서브태스크

- [ ] WebSocket 허브 구현
- [ ] 클라이언트 관리 로직
- [ ] 메시지 프로토콜 정의
- [ ] WebSocket 인증 구현
- [ ] 메시지 핸들러 구현
- [ ] 이벤트 버스 통합
- [ ] 연결 모니터링
- [ ] 테스트 작성

## 관련 링크

- Gorilla WebSocket: https://github.com/gorilla/websocket
- WebSocket 프로토콜: https://tools.ietf.org/html/rfc6455
- Go WebSocket 패턴: https://github.com/gorilla/websocket/tree/master/examples/chat