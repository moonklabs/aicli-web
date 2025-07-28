---
task_id: T04A_S01
sprint_sequence_id: S01
status: open
complexity: Medium
last_updated: 2025-07-27T20:30:00+0900
---

# Task: Docker 에이전트 이미지 및 컨테이너 기본 구현

## Description
Docker 에이전트 이미지를 설계하고 기본 컨테이너 관리 기능을 구현합니다. Claude CLI가 포함된 도커 이미지 생성, 컨테이너 생성/삭제, 기본 볼륨 마운트 기능을 구현합니다.

## Goal / Objectives
- Claude CLI 포함 Docker 이미지 설계 및 빌드
- 기본 컨테이너 생성/삭제 기능 구현
- 볼륨 마운트 시스템 통합
- Claude CLI 프로세스 실행 환경 구성

## Acceptance Criteria
- [ ] Docker 이미지가 성공적으로 빌드됨
- [ ] Claude CLI가 이미지에 올바르게 설치됨
- [ ] 컨테이너 생성/삭제가 정상 동작함
- [ ] 볼륨 마운트가 올바르게 설정됨
- [ ] 컨테이너 내에서 Claude CLI 실행 가능

## Subtasks
- [ ] Docker 에이전트 이미지 설계 (Ubuntu 기반)
- [ ] Dockerfile 작성 및 Claude CLI 설치 스크립트
- [ ] 이미지 빌드 및 최적화
- [ ] 기본 컨테이너 생성/삭제 로직 구현
- [ ] 볼륨 마운트 통합
- [ ] Claude CLI 프로세스 시작 로직
- [ ] 단위 테스트 작성

## 기술 가이드

### 주요 인터페이스 및 통합 지점
- **기존 Docker 매니저**: `internal/docker/container_manager.go`
- **Docker 팩토리**: `internal/docker/factory.go`
- **마운트 시스템**: `internal/mount/manager.go`
- **네트워크 매니저**: `internal/docker/network_manager.go`

### 특정 임포트 및 모듈 참조
```go
import (
    "github.com/docker/docker/api/types"
    "github.com/docker/docker/api/types/container"
    "github.com/docker/docker/api/types/mount"
    "github.com/docker/docker/api/types/network"
    "github.com/docker/docker/client"
)
```

### 따라야 할 기존 패턴
- 컨테이너 생성 패턴: `internal/docker/container_manager.go#CreateContainer`
- 볼륨 마운트: `internal/mount/manager.go#CreateMounts`
- 네트워크 설정: `internal/docker/network_manager.go#CreateNetwork`
- 리소스 제한: `internal/docker/client.go#parseResourceLimits`

### 작업할 데이터베이스 모델
- Agent 모델에 container_id 필드 연결
- ContainerInfo 저장 (상태, 리소스 사용량, 네트워크 정보)

### 오류 처리 접근법
- Docker API 에러 래핑
- 컨테이너 생성 실패 시 롤백
- 네트워크 연결 실패 시 재시도
- 리소스 부족 에러 처리

## 구현 노트

### 단계별 구현 접근법
1. 에이전트 Docker 이미지 정의 (Claude CLI 포함)
2. `internal/agent/docker_integration.go` 구현
3. 컨테이너 생성 및 설정 로직 구현
4. 볼륨 마운트 시스템 통합
5. Claude CLI 프로세스 시작 로직 추가
6. 모니터링 및 이벤트 처리 구현
7. 정리 및 삭제 로직 구현

### 주요 아키텍처 결정
- 각 에이전트는 독립된 컨테이너에서 실행
- 베이스 이미지는 Claude CLI가 사전 설치된 커스텀 이미지
- worktree는 읽기/쓰기 볼륨으로 마운트
- 에이전트별 격리된 네트워크 네임스페이스

### 테스트 접근법
- Docker-in-Docker 환경에서 테스트
- 컨테이너 생성/삭제 시나리오
- 볼륨 마운트 검증
- 네트워크 격리 테스트
- 리소스 제한 검증

### 성능 고려사항
- 컨테이너 이미지 레이어 캐싱
- 빠른 시작을 위한 이미지 최적화
- 리소스 풀링 (재사용 가능한 컨테이너)
- 병렬 컨테이너 생성 제한

### Docker 이미지 사양
```dockerfile
FROM ubuntu:22.04
RUN apt-get update && apt-get install -y \
    git \
    curl \
    python3 \
    && rm -rf /var/lib/apt/lists/*
    
# Claude CLI 설치
COPY scripts/install-claude-cli.sh /tmp/
RUN /tmp/install-claude-cli.sh

# 작업 디렉토리 설정
WORKDIR /workspace

# 엔트리포인트
ENTRYPOINT ["/usr/local/bin/claude"]
```

## Output Log
*(This section is populated as work progresses on the task)*