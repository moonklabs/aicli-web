---
task_id: T05_S03
sprint_sequence_id: S03
status: open
complexity: Low
last_updated: 2025-07-20T09:15:00Z
---

# Task: CONTRIBUTING.md 작성

## Description
오픈소스 프로젝트 기여자를 위한 상세한 가이드라인을 작성합니다. 개발 프로세스, 코딩 스타일, PR 절차 등을 문서화하여 새로운 기여자가 쉽게 참여할 수 있도록 합니다.

## Goal / Objectives
- 기여 프로세스 전체 플로우 문서화
- 코딩 스타일 가이드라인 제공
- PR 및 이슈 템플릿과 연계
- 개발 환경 설정 가이드 제공

## Acceptance Criteria
- [ ] CONTRIBUTING.md 파일이 프로젝트 루트에 생성됨
- [ ] 기여 프로세스가 단계별로 명확히 설명됨
- [ ] 코드 스타일, 커밋 메시지 규칙이 문서화됨
- [ ] 개발 환경 설정 방법이 포함됨
- [ ] 테스트 작성 가이드라인이 제공됨

## Subtasks
- [ ] CONTRIBUTING.md 파일 생성 및 구조 설계
- [ ] 기여 시작하기 섹션 작성
- [ ] 개발 환경 설정 가이드 작성
- [ ] 코딩 스타일 가이드라인 정의
- [ ] 커밋 메시지 컨벤션 문서화
- [ ] PR 프로세스 및 체크리스트 작성
- [ ] 이슈 보고 가이드라인 추가
- [ ] 행동 강령(Code of Conduct) 참조 추가

## 기술 가이드 섹션

### 코드베이스의 주요 인터페이스 및 통합 지점
- .golangci.yml - 린트 규칙 참조
- .pre-commit-config.yaml - pre-commit 훅 설정
- Makefile - 개발 명령어 참조
- docker-compose.yml - 개발 환경 설정

### 특정 임포트 및 모듈 참조
- Go 1.21+ 요구사항
- golangci-lint 설정
- pre-commit 프레임워크

### 따라야 할 기존 패턴
- gofmt/goimports 코드 포맷팅
- 한국어 주석 작성 규칙
- 테스트 파일 명명 규칙 (*_test.go)

### 작업할 데이터베이스 모델 또는 API 계약
- 해당 없음 (문서 작업)

### 유사한 코드에서 사용되는 오류 처리 접근법
- 에러 래핑 패턴 설명
- 로깅 규칙 문서화

## 구현 노트 섹션

### 단계별 구현 접근법
1. CONTRIBUTING.md 템플릿 구조 작성
2. 프로젝트별 내용으로 커스터마이징
3. 개발 워크플로우 다이어그램 추가
4. 코드 예제와 안티패턴 추가
5. 외부 리소스 링크 추가
6. 다른 문서와의 일관성 확인

### 존중해야 할 주요 아키텍처 결정
- AIWF 프레임워크 사용 설명
- 마일스톤/스프린트 기반 개발
- TDD 접근 방식 권장

### 기존 테스트 패턴을 바탕으로 한 테스트 접근법
- 유닛 테스트 작성 규칙
- 통합 테스트 가이드라인
- 테스트 커버리지 목표 (80%)

### 관련된 경우 성능 고려사항
- 해당 없음 (문서 작업)

## Output Log
*(This section is populated as work progresses on the task)*