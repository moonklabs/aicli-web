---
task_id: T02B_S01
sprint_sequence_id: S01
status: open
complexity: Medium
last_updated: 2025-07-22T18:30:00+0900
---

# Task: RBAC 미들웨어 및 관리 API 구현

## Description
RBAC 데이터 모델을 기반으로 고성능 권한 검사 미들웨어와 역할/권한 관리를 위한 REST API를 구현합니다. Redis 기반 캐싱으로 성능을 최적화합니다.

## Goal / Objectives
- 고성능 RBAC 미들웨어 구현 (기존 auth 미들웨어 확장)
- 역할/권한 관리 REST API 구현
- Redis 기반 권한 캐싱 시스템 구현
- 권한 변경 시 실시간 캐시 무효화

## Acceptance Criteria
- [ ] 권한 검사 미들웨어 구현 (성능 최적화됨)
- [ ] 역할/권한 CRUD API 엔드포인트 완성
- [ ] Redis 기반 권한 캐싱 시스템 동작
- [ ] 권한 변경 시 실시간 캐시 갱신 동작
- [ ] 기존 `RequireRole` 미들웨어와 하위 호환성 유지
- [ ] API 문서화 및 통합 테스트 완료

## Subtasks
- [ ] 기존 auth 미들웨어에 RBAC 로직 통합
- [ ] Redis 기반 권한 캐싱 시스템 구현
- [ ] 역할 관리 API 컨트롤러 구현
- [ ] 권한 관리 API 컨트롤러 구현
- [ ] 권한 변경 이벤트 시스템 구현
- [ ] 캐시 무효화 로직 구현
- [ ] API 엔드포인트 라우팅 설정
- [ ] 통합 테스트 및 성능 테스트 작성

## 기술 가이드

### 주요 인터페이스 및 통합 지점
- `internal/middleware/auth.go` 확장: RBAC 로직 추가
- `internal/api/controllers/rbac.go` 새 컨트롤러
- `internal/auth/rbac_cache.go` 새 파일: 캐싱 시스템
- `internal/server/router.go` 수정: RBAC API 라우팅

### 성능 고려사항
- 권한 검사 O(1) 복잡도 유지
- Redis 캐싱으로 데이터베이스 조회 최소화
- 배치 권한 업데이트 지원

## Output Log
*(This section is populated as work progresses on the task)*