# T01_S01_Go_Project_Init.md

## 태스크 정보
- **태스크 ID**: T01_S01
- **태스크명**: Go 프로젝트 초기화 및 디렉토리 구조 생성
- **스프린트**: S01_M01_Project_Init
- **상태**: PLANNED
- **우선순위**: HIGH
- **예상 소요 시간**: 2시간
- **실제 소요 시간**: -
- **담당자**: TBD
- **생성일**: 2025-01-20
- **시작일**: -
- **완료일**: -

## 태스크 설명

Go 언어 기반 AICLI 프로젝트의 초기 구조를 설정하고 필요한 디렉토리를 생성합니다. 기존 설계 문서(docs/cli-design/)를 참고하여 Go 표준 프로젝트 레이아웃을 따릅니다.

## 선행 조건

1. Go 1.21+ 설치 확인
2. Git 리포지토리 초기화 완료
3. 프로젝트 루트 디렉토리 준비

## 기술 가이드

### 1. Go 모듈 초기화

- 모듈명: `github.com/drumcap/aicli-web`
- Go 버전: 1.21 이상
- 프록시 설정 확인 (GOPROXY)

### 2. 표준 디렉토리 구조

#### 필수 디렉토리
- `cmd/`: 실행 가능한 프로그램의 진입점
  - `cmd/aicli/`: CLI 도구 메인 패키지
  - `cmd/api/`: API 서버 메인 패키지
  
- `internal/`: 내부 패키지 (외부 접근 불가)
  - `internal/cli/`: CLI 명령어 구현
  - `internal/server/`: API 서버 구현
  - `internal/claude/`: Claude CLI 래퍼
  - `internal/docker/`: Docker SDK 통합
  - `internal/storage/`: 데이터 저장소 인터페이스
  - `internal/models/`: 도메인 모델
  - `internal/config/`: 설정 관리
  
- `pkg/`: 외부 공개 패키지
  - `pkg/version/`: 버전 정보 관리
  - `pkg/utils/`: 공용 유틸리티
  
#### 프로젝트 지원 디렉토리
- `build/`: 빌드 관련 스크립트
- `scripts/`: 개발/배포 자동화 스크립트
- `configs/`: 기본 설정 파일
- `deployments/`: 배포 관련 파일 (Docker, K8s)
- `test/`: 통합 테스트, E2E 테스트
- `examples/`: 사용 예제

### 3. 주요 파일 생성 가이드

#### go.mod
- 모듈 경로 설정
- Go 버전 명시
- 의존성은 아직 추가하지 않음

#### Makefile
- 빌드 타겟: build, build-all
- 테스트 타겟: test, test-coverage
- 개발 타겟: run, dev, fmt, lint
- 정리 타겟: clean, clean-all

#### .gitignore
- Go 바이너리 파일
- 빌드 산출물
- IDE 설정 파일
- 로컬 설정 파일

### 4. 패키지 구조 원칙

#### internal 패키지
- 프로젝트 내부에서만 사용
- 비즈니스 로직 구현
- 외부 노출 방지

#### pkg 패키지
- 다른 프로젝트에서 재사용 가능
- 일반적인 유틸리티 함수
- 명확한 인터페이스 제공

### 5. 네이밍 컨벤션

#### 디렉토리명
- 소문자 사용
- 하이픈(-) 불가, 언더스코어(_) 사용
- 단수형 사용 (예: model이 아닌 models)

#### 파일명
- 소문자와 언더스코어 사용
- 테스트 파일: _test.go 접미사
- 플랫폼별 파일: _linux.go, _windows.go

### 6. 모듈 초기화 체크리스트

- [ ] go mod init 실행
- [ ] 디렉토리 구조 생성
- [ ] 각 패키지에 doc.go 파일 추가
- [ ] 기본 Makefile 작성
- [ ] .gitignore 설정
- [ ] README.md 업데이트

## 검증 기준

1. `go mod tidy` 실행 시 오류 없음
2. 모든 필수 디렉토리 존재
3. 디렉토리 구조가 Go 표준 레이아웃 준수
4. Makefile의 기본 타겟 동작 확인

## 구현 노트

### 참고 사항
- CLI 도구명이 `terry`에서 `aicli`로 변경됨
- Docker 이미지명도 `aicli-*` 형식 사용
- 한글 주석 사용, 변수명은 영어

### 주의 사항
- vendor 디렉토리는 생성하지 않음 (Go modules 사용)
- internal 패키지는 외부에서 import 불가
- 순환 의존성 방지를 위한 패키지 설계

## 관련 문서

- `/docs/cli-design/architecture.md`: 전체 아키텍처 설계
- `/docs/cli-design/cli-implementation.md`: CLI 구현 가이드
- `/.aiwf/00_PROJECT_MANIFEST.md`: 프로젝트 매니페스트

## 업데이트 로그

- 2025-01-20: 태스크 생성, 구조 가이드 작성