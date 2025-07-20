---
task_id: T03_S03
sprint_sequence_id: S03
status: open
complexity: Low
last_updated: 2025-07-20T09:15:00Z
---

# Task: 릴리스 자동화 프로세스 구축

## Description
Git 태그 기반 릴리스 자동화 시스템을 구축합니다. 태그가 푸시되면 자동으로 릴리스를 생성하고, 빌드된 바이너리를 첨부하며, 변경사항을 문서화합니다.

## Goal / Objectives
- Git 태그 기반 릴리스 트리거 설정
- 자동 릴리스 노트 생성
- 빌드된 바이너리 자동 첨부
- 체크섬 파일 생성 및 첨부

## Acceptance Criteria
- [ ] v* 형식의 태그 푸시 시 자동으로 릴리스가 생성됨
- [ ] 모든 플랫폼 바이너리가 릴리스에 첨부됨
- [ ] SHA256 체크섬 파일이 생성되고 첨부됨
- [ ] 자동 생성된 릴리스 노트에 변경사항이 포함됨
- [ ] 릴리스 생성 실패 시 알림이 발송됨

## Subtasks
- [ ] .github/workflows/release.yml 파일 생성
- [ ] 태그 트리거 설정 (v* 패턴)
- [ ] 릴리스 노트 자동 생성 설정
- [ ] 바이너리 빌드 및 수집
- [ ] 체크섬 생성 스크립트 작성
- [ ] GitHub Release API 연동
- [ ] 실패 시 롤백 메커니즘 구현

## 기술 가이드 섹션

### 코드베이스의 주요 인터페이스 및 통합 지점
- pkg/version/version.go - 버전 정보 관리
- Makefile의 dist 타겟 활용
- 기존 CI 파이프라인 재사용

### 특정 임포트 및 모듈 참조
- softprops/action-gh-release@v1
- actions/create-release@v1 (대안)
- Git 태그에서 버전 추출 패턴

### 따라야 할 기존 패턴
- 시맨틱 버저닝 (v1.0.0 형식)
- 바이너리 명명: aicli-{version}-{os}-{arch}
- 체크섬 파일: checksums.txt

### 작업할 데이터베이스 모델 또는 API 계약
- 해당 없음 (릴리스 자동화 작업)

### 유사한 코드에서 사용되는 오류 처리 접근법
- 릴리스 생성 실패 시 워크플로우 실패 처리
- 부분 성공 방지 (all-or-nothing)

## 구현 노트 섹션

### 단계별 구현 접근법
1. release.yml 워크플로우 파일 생성
2. 태그 푸시 이벤트 트리거 설정
3. 버전 정보 추출 (태그에서)
4. CI 워크플로우 재사용하여 빌드
5. 체크섬 생성 단계 추가
6. 릴리스 생성 및 아티팩트 업로드
7. 릴리스 노트 템플릿 설정

### 존중해야 할 주요 아키텍처 결정
- 시맨틱 버저닝 준수
- 모든 플랫폼 동시 릴리스
- 체크섬으로 무결성 보장

### 기존 테스트 패턴을 바탕으로 한 테스트 접근법
- 릴리스 전 전체 테스트 스위트 실행
- 바이너리 무결성 검증

### 관련된 경우 성능 고려사항
- 릴리스 아티팩트 병렬 업로드
- 대용량 파일 업로드 최적화

## Output Log
*(This section is populated as work progresses on the task)*