# M01: 기반 환경 구축 마일스톤 요구사항

## 마일스톤 개요

**목표**: Go 기반 AICode Manager 프로젝트의 초기 구조 설정 및 개발 환경 구축

**기간**: 1주 (3 스프린트)

**주요 산출물**:
- Go 프로젝트 초기화 완료
- 기본 프로젝트 구조 생성
- 개발 도구 및 CI/CD 파이프라인 설정
- 초기 문서화 완료

## 스프린트 계획

### S01_M01_Project_Init (2일)
**목표**: Go 프로젝트 초기화 및 기본 구조 설정

**작업 항목**:
- [ ] Go 모듈 초기화 (`go mod init`)
- [ ] 기본 디렉토리 구조 생성
- [ ] Makefile 작성
- [ ] 기본 main.go 파일 생성
- [ ] .gitignore 설정

**산출물**:
- go.mod, go.sum 파일
- 프로젝트 디렉토리 구조
- Makefile
- 초기 소스 코드

### S02_M01_Dev_Tools (2일)
**목표**: 개발 도구 및 테스트 환경 설정

**작업 항목**:
- [ ] 린터 설정 (golangci-lint)
- [ ] 테스트 프레임워크 설정
- [ ] pre-commit hooks 설정
- [ ] Docker 개발 환경 구성
- [ ] VS Code 설정 파일 추가

**산출물**:
- .golangci.yml
- .pre-commit-config.yaml
- Dockerfile.dev
- .vscode/settings.json

### S03_M01_CI_Setup (3일)
**목표**: CI/CD 파이프라인 및 문서화

**작업 항목**:
- [ ] GitHub Actions 워크플로우 설정
- [ ] 빌드 자동화 스크립트
- [ ] README.md 업데이트
- [ ] CONTRIBUTING.md 작성
- [ ] 개발 가이드 문서 작성

**산출물**:
- .github/workflows/ci.yml
- 업데이트된 README.md
- CONTRIBUTING.md
- docs/development-guide.md

## 기술 요구사항

### 필수 기술 스택
- **언어**: Go 1.21+
- **웹 프레임워크**: Gin (또는 Echo/Fiber - 스프린트 1에서 결정)
- **데이터베이스**: SQLite (임베디드)
- **컨테이너**: Docker 20.10+
- **CLI 프레임워크**: Cobra

### 개발 환경
- **OS**: macOS, Linux, Windows (WSL2)
- **IDE**: VS Code with Go extension (권장)
- **버전 관리**: Git

## 품질 기준

### 코드 품질
- [ ] 모든 Go 코드는 `gofmt` 적용
- [ ] `golangci-lint` 통과
- [ ] 기본 단위 테스트 작성
- [ ] 코드 커버리지 최소 70%

### 문서화
- [ ] 모든 public 함수에 GoDoc 주석
- [ ] README에 빌드 및 실행 방법 명시
- [ ] 아키텍처 다이어그램 포함

### 프로세스
- [ ] Git 커밋 메시지 규칙 준수 (한글)
- [ ] 브랜치 전략 수립 (Git Flow)
- [ ] PR 템플릿 작성

## 위험 요소 및 대응 방안

### 위험 요소
1. **Go 언어 학습 곡선**: 팀원들의 Go 경험 부족
   - **대응**: Go 기본 학습 자료 제공, 페어 프로그래밍

2. **Docker 환경 설정 복잡성**: 로컬 개발 환경 차이
   - **대응**: 상세한 설정 가이드 작성, Docker Compose 활용

3. **CI/CD 설정 지연**: GitHub Actions 설정 복잡성
   - **대응**: 단계적 접근, 기본 빌드부터 시작

## 완료 조건

- [ ] Go 프로젝트가 정상적으로 빌드됨
- [ ] 기본 테스트가 통과함
- [ ] CI 파이프라인이 동작함
- [ ] 문서화가 완료됨
- [ ] 팀원 모두가 로컬에서 개발 환경을 구축함

## 다음 마일스톤 예고

**M02: 코어 구조 구현** - CLI 명령어 구조, API 서버 기본 구조, 데이터베이스 스키마 설계