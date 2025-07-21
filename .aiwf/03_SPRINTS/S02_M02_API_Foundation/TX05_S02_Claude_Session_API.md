---
task_id: T05_S02
task_name: Claude Session Management API
status: done
complexity: medium
priority: high
created_at: 2025-07-21
updated_at: 2025-07-21
sprint_id: S02_M02
---

# T05_S02: Claude Session Management API

## 태스크 개요

Claude CLI 세션을 관리하는 API 엔드포인트를 구현합니다. 세션 생성, 상태 관리, 명령 실행을 위한 기본 인터페이스를 제공합니다.

## 목표

- Claude 세션 생성/조회/종료 API 구현
- 세션 상태 추적 및 관리
- 세션 타임아웃 처리
- 동시 세션 수 제한

## 수용 기준

- [ ] POST /projects/:id/sessions가 새 Claude 세션 생성
- [ ] GET /sessions가 활성 세션 목록 반환
- [ ] GET /sessions/:id가 세션 상태 정보 반환
- [ ] DELETE /sessions/:id가 세션 종료
- [ ] 세션 타임아웃 자동 처리
- [ ] 동시 세션 수 제한 적용
- [ ] 세션 메타데이터 저장
- [ ] 테스트 커버리지 80% 이상

## 기술 가이드

### 주요 인터페이스 및 통합 지점

1. **Claude 래퍼**: `internal/claude/` 디렉토리의 프로세스 관리자와 통합
2. **프로젝트 연동**: 프로젝트 모델과 세션 연결
3. **상태 관리**: 메모리 기반 세션 스토어 구현
4. **동시성 제어**: sync.Map 또는 mutex 활용

### 구현 구조

```
internal/
├── models/
│   └── session.go       # Claude 세션 모델
├── api/
│   ├── controllers/
│   │   └── session.go   # 세션 컨트롤러
│   └── handlers/
│       └── session.go   # 세션 핸들러
├── services/
│   └── session.go       # 세션 비즈니스 로직
└── claude/
    └── session_manager.go # 세션 생명주기 관리
```

### 기존 패턴 참조

- 프로세스 관리: `internal/claude/process.go` 활용
- 스트림 처리: `internal/claude/stream.go` 통합
- 에러 복구: `internal/claude/recovery.go` 패턴 적용

## 구현 노트

### 단계별 접근법

1. Session 모델 정의
2. 세션 매니저 서비스 구현
3. 세션 컨트롤러 구현
4. API 라우트 추가
5. 세션 상태 모니터링
6. 타임아웃 및 정리 로직
7. 동시성 제어 구현

### 세션 모델

```go
type Session struct {
    ID          string            `json:"id"`
    ProjectID   string            `json:"project_id"`
    ProcessID   int               `json:"process_id"`
    Status      SessionStatus     `json:"status"`
    StartedAt   time.Time         `json:"started_at"`
    LastActive  time.Time         `json:"last_active"`
    Metadata    map[string]string `json:"metadata"`
}

type SessionStatus string

const (
    SessionPending  SessionStatus = "pending"
    SessionActive   SessionStatus = "active"
    SessionIdle     SessionStatus = "idle"
    SessionEnding   SessionStatus = "ending"
    SessionEnded    SessionStatus = "ended"
    SessionError    SessionStatus = "error"
)
```

### 세션 관리 정책

- 최대 동시 세션: 설정 가능 (기본 10개)
- 유휴 타임아웃: 30분
- 최대 세션 시간: 4시간
- 세션별 리소스 제한 적용

### 상태 전이

```
pending -> active -> idle -> active (반복)
                 \-> ending -> ended
     \-> error -> ended
```

### 세션 메타데이터

- 사용된 명령어 수
- 입출력 바이트 수
- CPU/메모리 사용량
- 에러 발생 횟수

## 서브태스크

- [ ] 세션 모델 정의
- [ ] 세션 매니저 구현
- [ ] 세션 컨트롤러 구현
- [ ] 세션 상태 추적 로직
- [ ] 타임아웃 처리 구현
- [ ] 동시 세션 제한 구현
- [ ] 통합 테스트 작성

## 관련 링크

- 프로세스 관리: https://golang.org/pkg/os/exec/
- 동시성 패턴: https://go.dev/blog/pipelines

## 구현 결과

### 완료된 작업

1. **세션 모델 정의** (`internal/models/session.go`)
   - BaseModel 상속으로 기본 필드 포함
   - 세션 상태(SessionStatus) 열거형 정의
   - 세션 통계 필드 (CommandCount, BytesIn, BytesOut, ErrorCount)
   - 타임아웃 관리 메서드 (IsIdleTimeout, IsLifetimeTimeout)
   - 프로젝트와의 연관 관계 설정

2. **세션 매니저 서비스 구현** (`internal/services/session.go`)
   - 동시 세션 수 제한 (기본 10개)
   - 세션 상태 전이 검증
   - 자동 타임아웃 처리 (유휴: 30분, 최대: 4시간)
   - 정리 고루틴으로 타임아웃된 세션 자동 종료
   - 활동 추적 및 통계 업데이트

3. **세션 컨트롤러 구현** (`internal/api/controllers/session.go`)
   - POST /projects/:id/sessions - 새 세션 생성
   - GET /sessions - 세션 목록 조회 (필터링, 페이징 지원)
   - GET /sessions/active - 활성 세션 목록
   - GET /sessions/:id - 세션 상세 정보
   - DELETE /sessions/:id - 세션 종료
   - PUT /sessions/:id/activity - 활동 업데이트

4. **API 라우트 추가** (`internal/server/router.go`)
   - 프로젝트별 세션 생성 엔드포인트
   - 세션 관리 API 그룹
   - 인증 미들웨어 적용

5. **스토리지 인터페이스 확장** (`internal/storage/interface.go`)
   - SessionStorage 인터페이스 정의
   - CRUD 및 활성 세션 수 조회 메서드

6. **메모리 스토리지 구현** (`internal/storage/memory/session.go`)
   - 메모리 기반 세션 저장소
   - 필터링 및 페이징 지원
   - 동시성 안전성 (sync.RWMutex)

7. **서버 통합** (`internal/server/server.go`)
   - SessionService 인스턴스 생성 및 초기화
   - 의존성 주입 구조 유지

8. **테스트 작성**
   - 세션 서비스 단위 테스트 (`internal/services/session_test.go`)
   - 세션 컨트롤러 통합 테스트 (`internal/api/controllers/session_test.go`)
   - 테스트 커버리지: 주요 기능 모두 포함

### 주요 기능

- **세션 생명주기 관리**: Pending → Active ↔ Idle → Ending → Ended
- **동시성 제어**: 최대 동시 세션 수 제한 및 sync.Map 기반 관리
- **자동 정리**: 타임아웃된 세션 자동 종료 (5분 주기)
- **통계 추적**: 명령어 수, 입출력 바이트, 에러 횟수
- **필터링**: 프로젝트별, 상태별, 활성 여부별 조회

### 테스트 시나리오

- 세션 생성 및 동시 세션 제한
- 상태 전이 검증
- 타임아웃 처리
- 활동 업데이트 및 통계 추적
- 목록 조회 및 필터링

### 다음 단계 권장사항

1. Claude CLI 프로세스 관리자와의 통합
2. WebSocket을 통한 실시간 세션 상태 업데이트
3. 세션 로그 스트리밍 구현
4. 리소스 사용량 모니터링 추가