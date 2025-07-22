---
task_id: T02A_S01
sprint_sequence_id: S01
status: completed
complexity: Medium
last_updated: 2025-07-22T20:35:00+0900
---

# Task: RBAC 데이터 모델 및 권한 로직 구현

## Description
RBAC 시스템의 핵심이 되는 데이터 모델(Role, Permission, Resource, UserGroup)과 권한 계산, 상속, 그룹 기반 권한 할당 로직을 구현합니다.

## Goal / Objectives
- 세분화된 역할/권한/리소스 데이터 모델 정의
- 권한 상속 및 그룹 기반 권한 계산 로직 구현
- 권한 매트릭스 시스템 구현
- 권한 검증 핵심 로직 구현

## Acceptance Criteria
- [x] Role, Permission, Resource, UserGroup 데이터 모델 완성
- [x] 권한 상속 로직 (부모 역할에서 하위 역할로) 구현
- [x] 그룹 기반 권한 계산 로직 구현
- [x] 사용자별 최종 권한 계산 함수 구현
- [x] 데이터베이스 마이그레이션 스크립트 완성
- [x] RBAC 핵심 로직 단위 테스트 완료

## Subtasks
- [x] RBAC 데이터 모델 구조체 정의
- [x] 데이터베이스 스키마 및 마이그레이션 작성
- [x] 권한 상속 계산 알고리즘 구현
- [x] 그룹 기반 권한 집계 로직 구현
- [x] 사용자 최종 권한 계산 함수 구현
- [x] 권한 검증 핵심 함수 구현
- [x] RBAC 저장소 인터페이스 구현
- [x] 단위 테스트 작성

## 기술 가이드

### 주요 인터페이스 및 통합 지점
- `internal/models/rbac.go` 새 파일: RBAC 데이터 모델
- `internal/auth/rbac.go` 새 파일: RBAC 로직 구현
- `internal/storage/rbac.go` 새 파일: RBAC 저장소 인터페이스
- 기존 `internal/models/user.go` 확장: RBAC 관계 추가

### 특정 임포트 및 모듈 참조
```go
"github.com/aicli/aicli-web/internal/models"
"github.com/aicli/aicli-web/internal/storage"
"time"
"context"
```

### 따라야 할 기존 패턴
- `internal/models/workspace.go`의 모델 구조 패턴
- `internal/storage/interface.go`의 인터페이스 패턴
- 기존 에러 처리 방식

### 작업할 데이터베이스 모델
- Role, Permission, Resource, UserGroup, RolePermission, UserRole, UserGroup 테이블
- 기존 User 모델과의 관계 설정

## Output Log
*태스크 진행 과정에서 작업된 내용들을 기록합니다*

[2025-07-22 19:35]: 태스크 시작 - RBAC 데이터 모델 및 권한 로직 구현
[2025-07-22 19:40]: Base 모델 정의 문제 해결 - models/base.go에 Base 타입 별칭 추가
[2025-07-22 19:45]: RBAC 데이터 모델 완성 - models/rbac.go 생성 (8개 핵심 모델, 유효성 검사 포함)
[2025-07-22 19:50]: 데이터베이스 스키마 완성 - storage/schema/sqlite/002_rbac_tables.sql (8개 테이블, 인덱스, 트리거, 기본 데이터)
[2025-07-22 20:00]: RBAC 로직 구현 완성 - auth/rbac.go (권한 상속, 그룹 권한, 조건 평가, 캐싱 지원)
[2025-07-22 20:10]: RBAC 저장소 인터페이스 완성 - storage/rbac.go (포괄적인 RBAC 저장소 인터페이스 및 확장 기능)
[2025-07-22 20:15]: User 모델에 RBAC 관계 추가 - models/user.go 확장
[2025-07-22 20:20]: RBAC 미들웨어 구현 완성 - auth/rbac_middleware.go (권한 확인, 역할 요구, 소유권 확인)
[2025-07-22 20:25]: 단위 테스트 완성 - auth/rbac_test.go (권한 확인, 캐싱, 상속, 충돌 해결 테스트)
[2025-07-22 20:30]: 모든 하위 태스크 완료 - RBAC 시스템 구현 완료