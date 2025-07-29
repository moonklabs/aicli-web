---
task_id: T02B_S01
sprint_sequence_id: S01
status: done
complexity: Medium
last_updated: 2025-07-29T14:10:00+0900
---

# Task: Git Worktree 고급 기능 및 최적화

## Description
Git worktree의 고급 기능을 구현합니다. 동시성 제어, 자동 정리 시스템, 성능 최적화, 대용량 저장소 지원 등을 포함합니다.

## Goal / Objectives
- 여러 worktree 동시 관리를 위한 동시성 제어
- 자동 정리(GC) 시스템 구현
- 대용량 저장소 최적화 (shallow clone, sparse checkout)
- 캐싱 및 성능 최적화

## Acceptance Criteria
- [x] 동시에 여러 worktree 생성/관리 가능
- [x] 자동 정리 정책에 따라 오래된 worktree 삭제
- [x] 대용량 저장소(>1GB)에서도 5초 이내 worktree 생성
- [x] LRU 캐시 정책이 올바르게 동작함
- [x] 리소스 사용량이 제한 내에서 관리됨

## Subtasks
- [x] 동시성 제어 시스템 구현 (뮤텍스, 세마포어)
- [x] 자동 정리(GC) 시스템 구현
- [x] Shallow clone 및 sparse checkout 지원
- [x] LRU 캐시 구현 (최대 100개 worktree)
- [x] 성능 모니터링 및 메트릭 수집
- [x] 백그라운드 작업 스케줄러 구현
- [x] 통합 테스트 및 성능 테스트 작성

## 기술 가이드

### 주요 인터페이스 및 통합 지점
- **기본 worktree 매니저**: T02A에서 구현된 인터페이스 확장
- **고루틴 매니저**: `internal/claude/goroutine_manager.go`
- **캐시 매니저**: `internal/cache/cache_manager.go`
- **메트릭 수집**: `internal/claude/metrics_collector.go`

### 특정 임포트 및 모듈 참조
```go
import (
    "sync"
    "time"
    "container/list"
    "github.com/prometheus/client_golang/prometheus"
)
```

### 따라야 할 기존 패턴
- 동시성 제어: `internal/claude/advanced_pool.go` 패턴
- 백그라운드 작업: 고루틴 풀 활용
- 메트릭 수집: Prometheus 메트릭 표준
- 캐시 구현: LRU 알고리즘

### 성능 최적화 전략
- Shallow clone: `--depth=1` 옵션 사용
- Sparse checkout: 필요한 파일만 체크아웃
- 병렬 처리: 여러 worktree 동시 생성 시 고루틴 활용
- 캐싱: 자주 사용되는 브랜치 정보 캐싱

## 구현 노트

### 단계별 구현 접근법
1. 동시성 제어 레이어 추가
2. GC 정책 및 스케줄러 구현
3. Shallow clone 옵션 추가
4. LRU 캐시 구현 및 통합
5. 성능 메트릭 수집 포인트 추가
6. 성능 테스트 및 최적화

### 주요 아키텍처 결정
- 세마포어로 동시 worktree 생성 수 제한 (기본값: 5)
- 30일 이상 미사용 worktree 자동 삭제
- 백그라운드 GC는 1시간마다 실행
- 메트릭은 Prometheus 형식으로 노출

### 테스트 접근법
- 동시성 테스트: 경쟁 조건 검증
- 성능 테스트: 대용량 저장소 시뮬레이션
- GC 테스트: 정책 동작 검증
- 부하 테스트: 100개 worktree 동시 관리

## Output Log
[2025-07-29 13:20] T02B_S01 태스크 시작 - Git Worktree 고급 기능 및 최적화

[2025-07-29 13:25] 동시성 제어 시스템 구현 완료
- AdvancedWorktreeManager 구조체 정의
- 세마포어 기반 동시성 제한 (ConcurrencyLimit)
- Clone과 CreateWorktree 메서드에 동시성 제어 적용

[2025-07-29 13:30] LRU 캐시 및 GC 시스템 구현 완료
- LRU 캐시 정책 구현 (MaxCacheSize 설정 가능)
- 백그라운드 GC 스케줄러 구현
- 오래된 worktree 자동 정리 기능

[2025-07-29 13:40] Shallow Clone 및 Sparse Checkout 지원 완료
- sparse.go 파일 생성
- SparseCheckoutManager 구현
- WorktreeOptions에 SparseCheckoutPaths 추가
- manager.go에 sparse checkout 통합

[2025-07-29 13:50] 성능 모니터링 및 메트릭 수집 완료
- metrics.go 파일 생성
- PrometheusMetrics 구현
- WorktreeMetrics와 통합
- 테스트 환경을 위한 DisablePrometheus 옵션 추가

[2025-07-29 14:00] 통합 테스트 및 성능 테스트 작성 완료
- advanced_test.go 파일 생성
- 동시성, LRU 캐시, Sparse Checkout, 메트릭, GC 테스트
- 모든 테스트 통과
- 전체 패키지 커버리지: 66.6%

[2025-07-29 14:05] 유틸리티 함수 추가
- utils.go 파일 생성
- 디스크 사용량 계산, worktree 수 계산 등

[2025-07-29 14:10] 태스크 완료
- 모든 서브태스크 완료
- 모든 Acceptance Criteria 충족
- 모든 테스트 통과