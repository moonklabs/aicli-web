---
task_id: T03_S01
sprint_sequence_id: S01
status: open
complexity: Medium
last_updated: 2025-07-27T20:20:00+0900
---

# Task: 에이전트 서비스 계층 구현

## Description
에이전트의 비즈니스 로직을 담당하는 서비스 계층을 구현합니다. 에이전트 생명주기 관리, Docker 컨테이너 통합, 상태 모니터링, 에러 처리 및 복구 메커니즘을 포함합니다.

## Goal / Objectives
- 에이전트 생명주기 관리 로직 구현 (생성, 시작, 중지, 삭제)
- Docker 컨테이너와의 통합 인터페이스 구축
- 실시간 상태 모니터링 시스템 구현
- 에러 처리 및 자동 복구 메커니즘 구축

## Acceptance Criteria
- [ ] AgentService 인터페이스 구현 완료
- [ ] 에이전트 CRUD 작업이 정상 동작함
- [ ] 에이전트 상태 전이가 올바르게 관리됨
- [ ] Docker 컨테이너와 에이전트가 1:1로 매핑됨
- [ ] 에러 발생 시 자동 복구 시도가 수행됨
- [ ] 동시에 100개 이상의 에이전트 관리 가능

## Subtasks
- [ ] AgentService 인터페이스 설계
- [ ] 에이전트 생명주기 관리자 구현
- [ ] Docker 통합 어댑터 구현
- [ ] 상태 모니터링 시스템 구현
- [ ] 에러 처리 및 복구 로직 구현
- [ ] 이벤트 시스템 통합
- [ ] 단위 테스트 및 통합 테스트 작성

## 기술 가이드

### 주요 인터페이스 및 통합 지점
- **WorkspaceService 패턴 참조**: `internal/workspace/service.go`
- **Docker 매니저 통합**: `internal/docker/container_manager.go`
- **스토리지 레이어**: `internal/storage/interfaces.go`의 Agent 인터페이스
- **이벤트 시스템**: `internal/claude/event_bus.go` 활용

### 특정 임포트 및 모듈 참조
```go
import (
    "context"
    "sync"
    "time"
    
    "aicli-web/internal/models"
    "aicli-web/internal/storage"
    "aicli-web/internal/docker"
    "aicli-web/internal/claude"
)
```

### 따라야 할 기존 패턴
- 서비스 계층 패턴: `internal/workspace/service.go` 구조
- 의존성 주입: 생성자에서 모든 의존성 주입
- 컨텍스트 기반 취소: 모든 메서드에 context.Context 사용
- 구조화된 에러: `internal/claude/errors.go` 패턴

### 작업할 데이터베이스 모델
- Agent 모델 (T01에서 정의됨)
- 트랜잭션 관리: `storage.Transaction()` 인터페이스 활용
- 상태 업데이트 시 낙관적 잠금 고려

### 오류 처리 접근법
- 서비스 레벨 에러 타입 정의
- 에러 분류: 일시적/영구적, 사용자/시스템
- 복구 가능한 에러는 자동 재시도
- Circuit Breaker 통합 (`internal/claude/circuit_breaker.go`)

## 구현 노트

### 단계별 구현 접근법
1. `internal/agent/service.go` 파일 생성
2. AgentService 인터페이스 정의
3. 기본 CRUD 메서드 구현
4. 생명주기 관리 메서드 구현 (Start, Stop, Restart)
5. Docker 통합 로직 추가
6. 모니터링 및 헬스체크 구현
7. 이벤트 발행 통합

### 주요 아키텍처 결정
- 서비스는 스토리지와 Docker 매니저에 의존
- 상태 관리는 유한 상태 머신(FSM) 패턴 사용
- 비동기 작업은 고루틴 풀 활용
- 모든 상태 변경은 이벤트로 발행

### 테스트 접근법
- Mock 의존성을 사용한 단위 테스트
- 실제 Docker 환경에서의 통합 테스트
- 동시성 시나리오 테스트
- 에러 복구 시나리오 테스트

### 성능 고려사항
- 에이전트 조회 시 캐싱 적용
- 배치 작업 지원 (여러 에이전트 동시 처리)
- 고루틴 풀 크기 제한
- 메트릭 수집 오버헤드 최소화

## Output Log
*(This section is populated as work progresses on the task)*