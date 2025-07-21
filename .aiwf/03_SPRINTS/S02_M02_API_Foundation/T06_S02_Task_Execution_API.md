---
task_id: T06_S02
task_name: Task Execution API Endpoints
status: pending
complexity: low
priority: medium
created_at: 2025-07-21
updated_at: 2025-07-21
sprint_id: S02_M02
---

# T06_S02: Task Execution API Endpoints

## 태스크 개요

Claude 세션에서 실행되는 태스크를 관리하는 API 엔드포인트를 구현합니다. 명령 실행, 상태 추적, 결과 조회 기능을 제공합니다.

## 목표

- 태스크 실행 요청 API 구현
- 태스크 상태 조회 API 구현
- 태스크 취소 기능 구현
- 태스크 실행 이력 관리

## 수용 기준

- [ ] POST /sessions/:id/tasks가 새 태스크 실행
- [ ] GET /tasks/:id가 태스크 상태 반환
- [ ] GET /tasks가 태스크 목록 조회 (필터링 지원)
- [ ] DELETE /tasks/:id가 실행 중인 태스크 취소
- [ ] 태스크 실행 결과 저장
- [ ] 비동기 실행 및 상태 업데이트
- [ ] 통합 테스트 작성

## 기술 가이드

### 주요 인터페이스 및 통합 지점

1. **세션 통합**: 세션 컨텍스트 내에서 태스크 실행
2. **기존 핸들러**: `internal/server/handlers/task.go` 스텁 완성
3. **프로세스 관리**: Claude 프로세스에 명령 전달
4. **상태 추적**: 비동기 태스크 상태 업데이트

### 구현 구조

```
internal/
├── models/
│   └── task.go          # 태스크 모델 정의
├── server/
│   └── handlers/
│       └── task.go      # 태스크 핸들러 완성
├── services/
│   └── task.go          # 태스크 실행 서비스
└── queue/
    └── task_queue.go    # 태스크 큐 관리
```

### 기존 패턴 참조

- 비동기 처리: Go 루틴과 채널 활용
- 상태 관리: 기존 세션 상태 관리 패턴 참조
- 에러 처리: 표준 에러 응답 형식 유지

## 구현 노트

### 단계별 접근법

1. Task 모델 정의
2. 태스크 큐 시스템 구현
3. 태스크 실행 서비스 구현
4. 핸들러 로직 완성
5. 상태 추적 메커니즘
6. 태스크 취소 로직
7. 테스트 작성

### 태스크 모델

```go
type Task struct {
    ID          string      `json:"id"`
    SessionID   string      `json:"session_id"`
    Command     string      `json:"command" binding:"required"`
    Status      TaskStatus  `json:"status"`
    Output      string      `json:"output,omitempty"`
    Error       string      `json:"error,omitempty"`
    StartedAt   time.Time   `json:"started_at"`
    CompletedAt *time.Time  `json:"completed_at,omitempty"`
}

type TaskStatus string

const (
    TaskPending   TaskStatus = "pending"
    TaskRunning   TaskStatus = "running"
    TaskCompleted TaskStatus = "completed"
    TaskFailed    TaskStatus = "failed"
    TaskCancelled TaskStatus = "cancelled"
)
```

### 태스크 실행 플로우

1. 태스크 요청 수신
2. 세션 유효성 검증
3. 태스크 큐에 추가
4. 비동기 실행 시작
5. 상태 업데이트 (실시간)
6. 결과 저장

### 태스크 필터링

- 세션별 태스크 조회
- 상태별 필터링
- 시간 범위 필터
- 페이지네이션 지원

## 서브태스크

- [ ] 태스크 모델 정의
- [ ] 태스크 큐 구현
- [ ] 태스크 실행 서비스
- [ ] 핸들러 로직 완성
- [ ] 상태 추적 구현
- [ ] 태스크 취소 기능
- [ ] 테스트 작성

## 관련 링크

- Go 동시성: https://go.dev/doc/effective_go#concurrency
- 작업 큐 패턴: https://gobyexample.com/worker-pools