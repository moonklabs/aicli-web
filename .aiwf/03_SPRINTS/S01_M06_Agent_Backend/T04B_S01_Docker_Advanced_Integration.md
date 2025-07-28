---
task_id: T04B_S01
sprint_sequence_id: S01
status: open
complexity: Medium
last_updated: 2025-07-27T20:30:00+0900
---

# Task: Docker 고급 통합 및 관리

## Description
Docker 컨테이너의 고급 통합 기능을 구현합니다. 네트워크 격리, 리소스 제한, 생명주기 이벤트 처리, 상태 동기화 등 엔터프라이즈급 컨테이너 관리 기능을 구현합니다.

## Goal / Objectives
- 컨테이너 간 네트워크 격리 구현
- CPU/메모리 리소스 제한 및 모니터링
- 컨테이너 생명주기 이벤트 처리
- 에이전트와 컨테이너 상태 동기화

## Acceptance Criteria
- [ ] 컨테이너 간 네트워크 격리가 보장됨
- [ ] CPU/메모리 리소스 제한이 적용됨
- [ ] 컨테이너 상태 변경이 실시간으로 감지됨
- [ ] 에이전트 상태와 컨테이너 상태가 동기화됨
- [ ] 리소스 사용량 모니터링이 가능함
- [ ] 컨테이너 장애 시 자동 복구됨

## Subtasks
- [ ] 네트워크 격리 설정 구현 (브리지 네트워크)
- [ ] 리소스 제한 설정 (CPU, 메모리, 디스크 I/O)
- [ ] Docker 이벤트 모니터링 시스템 구현
- [ ] 컨테이너 헬스체크 구현
- [ ] 상태 동기화 메커니즘 구현
- [ ] 자동 복구 로직 구현
- [ ] 리소스 사용량 메트릭 수집
- [ ] 통합 테스트 작성

## 기술 가이드

### 주요 인터페이스 및 통합 지점
- **네트워크 매니저**: `internal/docker/network_manager.go`
- **리소스 제한**: `internal/docker/client.go#parseResourceLimits`
- **이벤트 모니터링**: `internal/docker/lifecycle_manager.go`
- **헬스체커**: `internal/docker/health_checker.go`

### 특정 임포트 및 모듈 참조
```go
import (
    "github.com/docker/docker/api/types/events"
    "github.com/docker/docker/api/types/container"
    "github.com/docker/docker/api/types/network"
    "github.com/shirou/gopsutil/v3/docker"
)
```

### 네트워크 격리 설정
```go
// 에이전트별 격리된 네트워크
NetworkMode: "bridge"
NetworkConfig: &network.NetworkingConfig{
    EndpointsConfig: map[string]*network.EndpointSettings{
        "agent-network": {
            NetworkID: agentNetworkID,
            IPAMConfig: &network.EndpointIPAMConfig{
                IPv4Address: assignedIP,
            },
        },
    },
}
```

### 리소스 제한 설정
```go
Resources: container.Resources{
    CPUQuota:  100000,  // 1 CPU
    Memory:    2 * 1024 * 1024 * 1024, // 2GB
    PidsLimit: &pidsLimit, // 1000
    Ulimits: []units.Ulimit{
        {Name: "nofile", Soft: 1024, Hard: 2048},
    },
}
```

## 구현 노트

### 단계별 구현 접근법
1. 네트워크 격리 레이어 구현
2. 리소스 제한 설정 통합
3. Docker 이벤트 리스너 구현
4. 헬스체크 및 모니터링 추가
5. 상태 동기화 로직 구현
6. 자동 복구 메커니즘 추가

### 주요 아키텍처 결정
- 각 에이전트는 독립된 브리지 네트워크 사용
- 기본 리소스 제한: 1 CPU, 2GB 메모리
- 헬스체크는 30초마다 실행
- 3회 연속 실패 시 컨테이너 재시작

### 테스트 접근법
- 네트워크 격리 테스트: ping 테스트로 격리 검증
- 리소스 제한 테스트: 스트레스 테스트
- 이벤트 처리 테스트: 모의 이벤트 발생
- 복구 테스트: 컨테이너 강제 종료 후 복구 확인

### 모니터링 메트릭
- CPU 사용률 (%)
- 메모리 사용량 (MB)
- 네트워크 I/O (bytes/sec)
- 디스크 I/O (ops/sec)
- 컨테이너 재시작 횟수

## Output Log
*(This section is populated as work progresses on the task)*