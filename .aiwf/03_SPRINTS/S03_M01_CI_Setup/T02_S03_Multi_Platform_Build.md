---
task_id: T02_S03
sprint_sequence_id: S03
status: open
complexity: Medium
last_updated: 2025-07-20T09:15:00Z
---

# Task: 멀티 플랫폼 빌드 자동화

## Description
GitHub Actions에서 Matrix 빌드를 활용하여 Linux, macOS, Windows용 바이너리를 자동으로 빌드하는 시스템을 구축합니다. 각 플랫폼별 최적화된 빌드 설정을 적용합니다.

## Goal / Objectives
- 멀티 플랫폼 (Linux/macOS/Windows) 빌드 자동화
- 각 플랫폼별 바이너리 생성
- 크로스 컴파일 설정 최적화
- 빌드 아티팩트 자동 업로드

## Acceptance Criteria
- [ ] Linux (amd64, arm64), macOS (amd64, arm64), Windows (amd64) 바이너리가 생성됨
- [ ] 각 플랫폼별 바이너리가 정상 실행됨
- [ ] 빌드 아티팩트가 GitHub Actions에 업로드됨
- [ ] 빌드 시간이 플랫폼당 3분 이내로 최적화됨
- [ ] 바이너리 크기가 최적화됨 (strip, upx 등)

## Subtasks
- [ ] Matrix 빌드 전략 설정
- [ ] 플랫폼별 GOOS/GOARCH 환경 변수 구성
- [ ] CGO 비활성화 설정 (순수 Go 바이너리)
- [ ] 빌드 후 바이너리 최적화 (strip, compress)
- [ ] 아티팩트 업로드 액션 구성
- [ ] 플랫폼별 테스트 검증
- [ ] 빌드 결과 요약 리포트 생성

## 기술 가이드 섹션

### 코드베이스의 주요 인터페이스 및 통합 지점
- Makefile의 build-all 타겟 활용
- cmd/aicli/main.go - CLI 진입점
- cmd/api/main.go - API 서버 진입점
- pkg/version/version.go - 버전 정보 임베딩

### 특정 임포트 및 모듈 참조
- actions/upload-artifact@v4
- actions/download-artifact@v4
- 크로스 컴파일을 위한 Go 표준 환경 변수

### 따라야 할 기존 패턴
- Makefile의 BINARY_NAME 변수 패턴
- 빌드 플래그: `-ldflags "-s -w"` (바이너리 크기 최적화)
- 버전 정보 주입 패턴: `-X pkg/version.Version=$(VERSION)`

### 작업할 데이터베이스 모델 또는 API 계약
- 해당 없음 (빌드 설정 작업)

### 유사한 코드에서 사용되는 오류 처리 접근법
- 빌드 실패 시 명확한 에러 메시지 출력
- 플랫폼별 빌드 상태 개별 추적

## 구현 노트 섹션

### 단계별 구현 접근법
1. ci.yml에 matrix 전략 추가
2. 플랫폼별 빌드 환경 변수 설정
3. CGO_ENABLED=0 설정으로 정적 바이너리 생성
4. 빌드 스크립트 최적화
5. 아티팩트 명명 규칙 정의 (aicli-{version}-{os}-{arch})
6. 압축 및 최적화 단계 추가
7. 업로드 및 다운로드 액션 구성

### 존중해야 할 주요 아키텍처 결정
- 순수 Go 바이너리 생성 (CGO 비활성화)
- 정적 링킹으로 의존성 최소화
- 플랫폼별 최적화 플래그 적용

### 기존 테스트 패턴을 바탕으로 한 테스트 접근법
- 각 플랫폼에서 `aicli version` 실행하여 검증
- 기본 명령어 실행 테스트

### 관련된 경우 성능 고려사항
- 병렬 빌드로 전체 시간 단축
- 빌드 캐시 재사용
- 바이너리 크기 최적화 (UPX 압축 고려)
- 불필요한 심볼 제거 (-s -w 플래그)

## Output Log
*(This section is populated as work progresses on the task)*