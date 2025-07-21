---
task_id: T07_S02
task_name: WebSocket Foundation for Real-time Communication
status: done
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

- [x] /ws 엔드포인트에서 WebSocket 연결 수락
- [x] JWT 토큰 기반 WebSocket 인증
- [x] 연결별 고유 ID 할당 및 관리
- [x] 표준화된 메시지 포맷 구현
- [x] 연결 상태 모니터링
- [x] 자동 재연결 지원 (클라이언트 가이드)
- [x] 연결 풀 관리 및 제한
- [x] 단위 및 통합 테스트 작성

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

- [x] WebSocket 허브 구현
- [x] 클라이언트 관리 로직
- [x] 메시지 프로토콜 정의
- [x] WebSocket 인증 구현
- [x] 메시지 핸들러 구현
- [x] 이벤트 버스 통합
- [x] 연결 모니터링
- [x] 테스트 작성

## 관련 링크

- Gorilla WebSocket: https://github.com/gorilla/websocket
- WebSocket 프로토콜: https://tools.ietf.org/html/rfc6455
- Go WebSocket 패턴: https://github.com/gorilla/websocket/tree/master/examples/chat

## 구현 결과

### 완료된 작업

1. **메시지 프로토콜 정의** (`internal/websocket/message.go`)
   - 표준화된 메시지 구조체 및 타입 정의
   - 시스템 메시지와 비즈니스 메시지 분류
   - 각 메시지 타입별 전용 구조체 (AuthMessage, LogMessage, TaskMessage 등)
   - 메시지 생성 헬퍼 함수들
   - 채널 명명 규칙 및 헬퍼 함수

2. **클라이언트 관리 시스템** (`internal/websocket/client.go`)
   - 클라이언트 연결 생명주기 관리 (생성, 시작, 중지)
   - 채널 구독/구독취소 시스템
   - 메시지 송수신 처리 (readPump, writePump)
   - 핑/퐁 기반 연결 상태 모니터링
   - 인증 상태 관리 및 메시지별 처리
   - 클라이언트 통계 수집

3. **WebSocket 허브 구현** (`internal/websocket/hub.go`)
   - 중앙 집중식 연결 관리
   - 채널 기반 메시지 브로드캐스트
   - 연결 제한 및 정리 메커니즘
   - 실시간 통계 수집 및 모니터링
   - 하트비트 및 헬스체크 시스템
   - 동시성 안전 구현 (sync.RWMutex 활용)

4. **WebSocket 인증 시스템** (`internal/websocket/auth.go`)
   - JWT 기반 WebSocket 인증
   - 연결 시점 및 메시지 기반 인증 지원
   - 채널별 접근 권한 제어
   - 역할 기반 권한 관리
   - 토큰 블랙리스트 통합
   - 모의 인증기 (테스트용)

5. **메시지 핸들러 및 서버 통합** (`internal/websocket/handler.go`)
   - Gin 기반 WebSocket 엔드포인트 핸들러
   - 브로드캐스트 관리자 (워크스페이스, 세션, 태스크별)
   - 이벤트 핸들러 (태스크, 세션, 로그 업데이트)
   - 메트릭 수집기 및 헬스 체커
   - 연결 제한기 (전체/사용자별)

6. **서버 통합** (`internal/server/server.go`, `internal/server/router.go`)
   - WebSocket 허브 및 핸들러 서버 통합
   - `/ws` 엔드포인트 추가
   - JWT 인증 시스템과 연동
   - 서버 생명주기에 WebSocket 서비스 포함

7. **의존성 및 설정**
   - `go.mod`에 gorilla/websocket 라이브러리 추가
   - 모듈명 수정 (github.com/aicli/aicli-web)
   - WebSocket 설정 구조체들 (HubConfig, ClientConfig, HandlerConfig)

8. **테스트 작성**
   - 메시지 프로토콜 테스트 (`internal/websocket/message_test.go`)
   - 클라이언트 관리 테스트 (`internal/websocket/client_test.go`)
   - 주요 기능별 단위 테스트 및 통합 테스트

### 주요 기능

- **실시간 통신**: WebSocket 기반 양방향 실시간 통신
- **채널 시스템**: 워크스페이스, 세션, 태스크, 사용자별 채널 구독
- **인증 및 보안**: JWT 기반 인증, 채널별 접근 제어, 메시지 크기 제한
- **연결 관리**: 최대 연결 수 제한, 자동 정리, 헬스체크
- **브로드캐스트**: 채널별, 사용자별 메시지 브로드캐스트
- **모니터링**: 실시간 통계, 연결 상태 추적, 메트릭 수집

### WebSocket API

| 엔드포인트 | 설명 |
|-----------|------|
| `GET /ws` | WebSocket 연결 수립 |
| `?token=<jwt>` | 쿼리 파라미터로 JWT 토큰 전달 |

### 메시지 타입

**시스템 메시지:**
- `auth`: 인증
- `ping/pong`: 연결 상태 확인
- `subscribe/unsubscribe`: 채널 구독 관리
- `error/success`: 응답 메시지

**비즈니스 메시지:**
- `log`: 실시간 로그 스트림
- `task`: 태스크 상태 업데이트
- `session`: 세션 상태 업데이트
- `event`: 시스템 이벤트
- `command`: 명령 실행

### 채널 시스템

- `workspace:<id>`: 워크스페이스별 채널
- `session:<id>`: 세션별 채널
- `task:<id>`: 태스크별 채널
- `user:<id>`: 사용자별 개인 채널
- `system`: 시스템 채널
- `broadcast`: 전체 브로드캐스트

### 설정 옵션

- **허브**: 최대 클라이언트 수 (기본 1000), 정리 주기, 통계 업데이트 주기
- **클라이언트**: 메시지 버퍼 크기, 타임아웃 설정, 최대 메시지 크기
- **핸들러**: CORS 설정, 버퍼 크기, 압축 설정

### 다음 단계 권장사항

1. **Claude CLI 통합**: 실제 Claude 세션과 태스크 이벤트 연동
2. **Rate Limiting**: 메시지 전송 빈도 제한
3. **메시지 영속화**: 중요한 메시지의 데이터베이스 저장
4. **클러스터링**: 여러 서버 인스턴스 간 메시지 동기화
5. **웹 클라이언트**: 프론트엔드 WebSocket 클라이언트 라이브러리
6. **모니터링 대시보드**: WebSocket 연결 상태 실시간 모니터링