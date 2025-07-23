# AICode Manager (aicli-web)

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![Build Status](https://img.shields.io/github/actions/workflow/status/moonklabs/aicli-web/ci.yml?branch=main)](https://github.com/moonklabs/aicli-web/actions)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/moonklabs/aicli-web)](https://goreportcard.com/report/github.com/moonklabs/aicli-web)

AICode Manager는 Claude CLI를 웹 플랫폼으로 관리하는 로컬 우선 시스템입니다. Go 언어로 개발된 네이티브 CLI 도구를 중심으로 각 프로젝트별 격리된 Docker 컨테이너에서 Claude CLI를 실행하고 관리합니다.

## 목차

- [프로젝트 개요](#프로젝트-개요)
- [주요 기능](#주요-기능)
- [시작하기](#시작하기)
  - [사전 요구사항](#사전-요구사항)
  - [설치 방법](#설치-방법)
  - [빠른 시작](#빠른-시작)
- [사용법](#사용법)
  - [CLI 명령어](#cli-명령어)
  - [Claude CLI 통합](#claude-cli-통합)
  - [API 엔드포인트](#api-엔드포인트)
- [프로젝트 구조](#프로젝트-구조)
- [개발하기](#개발하기)
- [기여하기](#기여하기)
- [라이선스](#라이선스)

## 프로젝트 개요

AICode Manager는 개발자가 여러 프로젝트에서 Claude CLI를 효율적으로 관리하고 실행할 수 있도록 설계된 도구입니다. 각 프로젝트는 독립된 Docker 컨테이너에서 실행되어 완벽한 격리 환경을 제공하며, 웹 대시보드를 통해 실시간으로 모니터링할 수 있습니다.

### 왜 AICode Manager인가?

- **멀티 프로젝트 관리**: 여러 프로젝트를 동시에 실행하고 관리
- **격리된 실행 환경**: Docker를 통한 프로젝트별 독립적인 환경 보장
- **실시간 모니터링**: WebSocket을 통한 실시간 로그 스트리밍
- **Git 워크플로우 통합**: 자동 브랜치 생성, 커밋, PR 관리
- **로컬 우선 설계**: 클라우드 의존성 없이 완전히 로컬에서 실행

## 주요 기능

- 🚀 **Claude CLI 래핑**: 프로세스 격리 및 생명주기 관리
- 📁 **워크스페이스 관리**: 멀티 프로젝트 병렬 실행
- 📊 **실시간 로그 스트리밍**: WebSocket 기반 실시간 모니터링
- 🔄 **Git 워크플로우 통합**: 자동 브랜치/커밋/PR 관리
- 🔐 **사용자 인증 및 권한 관리**: 안전한 멀티 유저 환경
- 🌐 **RESTful API**: 프로그래밍 가능한 인터페이스
- 💻 **CLI 도구**: 강력한 커맨드라인 인터페이스

## 시작하기

### 사전 요구사항

- Go 1.21 이상
- Docker 20.10 이상
- Make (빌드 자동화용)
- Git

### 설치 방법

#### 방법 1: Go로 소스에서 빌드 (권장)

Go 1.21 이상이 설치되어 있어야 합니다:

```bash
# 저장소 클론
git clone https://github.com/moonklabs/aicli-web.git
cd aicli-web

# Go 모듈 의존성 다운로드
go mod download

# 개발 도구 설치 (선택사항)
make setup

# CLI 도구 빌드
make build-cli

# API 서버 빌드  
make build-api

# 또는 모든 바이너리 한 번에 빌드
make build

# 빌드된 바이너리 확인
ls -la build/
# build/aicli        (CLI 도구)
# build/aicli-api    (API 서버)

# 시스템 PATH에 추가 (선택사항)
sudo cp build/aicli /usr/local/bin/
sudo cp build/aicli-api /usr/local/bin/
```

#### 방법 2: Go install (CLI 도구만)

```bash
# CLI 도구 설치
go install github.com/moonklabs/aicli-web/cmd/aicli@latest

# API 서버 설치
go install github.com/moonklabs/aicli-web/cmd/api@latest
```

#### 방법 3: Docker로 실행

```bash
# Docker Compose로 전체 스택 실행
git clone https://github.com/moonklabs/aicli-web.git
cd aicli-web

# 개발 환경 실행
docker-compose up -d

# 또는 프로덕션 Docker 이미지 빌드
make docker
```

#### 방법 4: 바이너리 다운로드

릴리스가 준비되면 다음 링크에서 다운로드 가능합니다:

```bash
# Linux (amd64)
wget https://github.com/moonklabs/aicli-web/releases/latest/download/aicli-linux-amd64.tar.gz
tar -xzf aicli-linux-amd64.tar.gz
sudo mv aicli /usr/local/bin/

# macOS (Intel)
wget https://github.com/moonklabs/aicli-web/releases/latest/download/aicli-darwin-amd64.tar.gz
tar -xzf aicli-darwin-amd64.tar.gz
sudo mv aicli /usr/local/bin/

# macOS (Apple Silicon)
wget https://github.com/moonklabs/aicli-web/releases/latest/download/aicli-darwin-arm64.tar.gz
tar -xzf aicli-darwin-arm64.tar.gz
sudo mv aicli /usr/local/bin/
```

### 빠른 시작

5분 안에 AICode Manager를 시작하세요:

#### 1. 빌드 후 실행

```bash
# 프로젝트 빌드 (위의 설치 방법 참조)
make build

# 설정 초기화
./build/aicli config init

# Claude API 키 설정
./build/aicli config set claude.api_key "your-claude-api-key"

# Docker 데몬 확인 (필요한 경우)
docker --version
```

#### 2. API 서버 실행

```bash
# API 서버 시작 (백그라운드)
./build/aicli-api serve --port 8080 &

# 또는 포그라운드에서 실행 (로그 확인)
./build/aicli-api serve --port 8080

# 다른 터미널에서 헬스체크
curl http://localhost:8080/health
```

#### 3. CLI로 워크스페이스 관리

```bash
# 새 워크스페이스 생성
./build/aicli workspace create my-project --path ./my-project

# 워크스페이스 목록 확인
./build/aicli workspace list

# 워크스페이스 상태 확인
./build/aicli workspace get my-project
```

#### 4. Claude 태스크 실행

```bash
# 간단한 태스크 실행
./build/aicli task run --workspace my-project "현재 디렉토리의 Go 파일들을 분석해주세요"

# 태스크 목록 확인
./build/aicli task list

# 실시간 로그 스트리밍
./build/aicli logs follow <task-id>
```

#### 5. 웹 대시보드 접속

브라우저에서 `http://localhost:8080`으로 접속하여 웹 대시보드를 확인할 수 있습니다.

#### 개발 모드로 실행

```bash
# Hot reload로 개발 모드 실행
make dev

# 또는 Docker Compose로 전체 스택 실행
docker-compose up -d

# 로그 확인
docker-compose logs -f
```

## 실제 사용 예제

### 기본 워크플로우

```bash
# 1. 프로젝트 빌드
make build

# 2. 설정 초기화
./build/aicli config init

# 3. Claude API 키 설정
./build/aicli config set claude.api_key "your-api-key"

# 4. 새 워크스페이스 생성
./build/aicli workspace create my-go-project --path /path/to/my-go-project

# 5. API 서버 시작 (백그라운드)
./build/aicli-api serve --port 8080 &

# 6. 코드 분석 태스크 실행
./build/aicli task run --workspace my-go-project "이 Go 프로젝트의 구조를 분석하고 개선점을 제안해주세요"

# 7. 태스크 상태 확인
./build/aicli task list

# 8. 실시간 로그 확인
./build/aicli logs follow <task-id>
```

### 고급 사용 예제

```bash
# 여러 워크스페이스 동시 관리
./build/aicli workspace create frontend --path ./frontend
./build/aicli workspace create backend --path ./backend

# 병렬 태스크 실행
./build/aicli task run --workspace frontend "React 컴포넌트 최적화"
./build/aicli task run --workspace backend "API 성능 최적화"

# 워크스페이스 상태 모니터링
./build/aicli workspace get frontend
./build/aicli workspace get backend

# 설정 관리
./build/aicli config get
./build/aicli config set claude.temperature 0.7
```

## 사용법

### CLI 명령어

AICode Manager CLI는 직관적인 명령어 구조를 제공합니다:

#### 기본 명령어

```bash
# 버전 확인
aicli version

# 도움말
aicli help
aicli help <command>

# 자동 완성 설정
aicli completion bash > /etc/bash_completion.d/aicli
```

#### 워크스페이스 명령어

```bash
# 워크스페이스 목록 조회
aicli workspace list

# 새 워크스페이스 생성
aicli workspace create <name> --path <project-path>

# 워크스페이스 정보 조회
aicli workspace get <name>

# 워크스페이스 삭제
aicli workspace delete <name>
```

#### 태스크 명령어

```bash
# 태스크 실행
aicli task run --workspace <workspace-name> "작업 내용"

# 실행 중인 태스크 목록
aicli task list

# 태스크 상태 확인
aicli task get <task-id>

# 태스크 중지
aicli task cancel <task-id>
```

#### 로그 명령어

```bash
# 워크스페이스 로그 조회
aicli logs workspace <workspace-name>

# 태스크 로그 조회
aicli logs task <task-id>

# 실시간 로그 스트리밍
aicli logs follow <task-id>
```

#### 설정 명령어

```bash
# 설정 초기화
aicli config init

# 설정 조회
aicli config get
aicli config get <key>

# 설정 변경
aicli config set <key> <value>

# 설정 파일 위치
aicli config path
```

### Claude CLI 통합

AICode Manager의 핵심 기능인 Claude CLI 통합을 통해 강력한 AI 개발 도구를 활용할 수 있습니다.

#### Claude 명령어

```bash
# 단일 프롬프트 실행
aicli claude run "Write a Go function to reverse a string"

# 인터랙티브 세션
aicli claude chat --system "You are a helpful coding assistant"

# 세션 관리
aicli claude session list
aicli claude session show <session-id>
aicli claude session clean
```

#### 주요 특징

- 🔄 **세션 관리**: 재사용 가능한 세션으로 성능 최적화
- 📡 **실시간 스트리밍**: WebSocket을 통한 실시간 응답 스트리밍  
- 🛡️ **에러 복구**: 자동 재시도 및 회로 차단기 패턴
- 🎯 **백프레셔 처리**: 효율적인 스트림 버퍼 관리
- 📊 **모니터링**: 성능 메트릭 및 상세 로깅

#### 문서

- [사용 가이드](./docs/claude/usage-guide.md) - 기본 사용법과 설정
- [API 레퍼런스](./docs/claude/api-reference.md) - REST API 및 WebSocket API
- [설정 가이드](./docs/claude/configuration.md) - 환경 변수 및 설정 파일
- [아키텍처](./docs/claude/architecture.md) - 시스템 설계 및 구조
- [트러블슈팅](./docs/claude/troubleshooting.md) - 일반적인 문제 해결
- [예제 및 레시피](./docs/claude/examples.md) - 실용적인 사용 예제

### API 엔드포인트

RESTful API를 통해 프로그래밍 방식으로 AICode Manager를 제어할 수 있습니다:

#### 시스템 엔드포인트

```
GET  /                    # API 서버 정보
GET  /health              # 헬스체크
GET  /version             # 버전 정보
GET  /api/v1/system/info  # 시스템 정보
GET  /api/v1/system/status # 시스템 상태
```

#### 워크스페이스 API

```
GET    /api/v1/workspaces              # 워크스페이스 목록
POST   /api/v1/workspaces              # 워크스페이스 생성
GET    /api/v1/workspaces/:id          # 워크스페이스 조회
PUT    /api/v1/workspaces/:id          # 워크스페이스 수정
DELETE /api/v1/workspaces/:id          # 워크스페이스 삭제
```

#### 태스크 API

```
GET    /api/v1/tasks                   # 태스크 목록
POST   /api/v1/tasks                   # 태스크 생성
GET    /api/v1/tasks/:id               # 태스크 조회
DELETE /api/v1/tasks/:id               # 태스크 취소
```

#### 로그 API

```
GET    /api/v1/logs/workspaces/:id     # 워크스페이스 로그
GET    /api/v1/logs/tasks/:id          # 태스크 로그
WS     /api/v1/logs/stream/:id         # 실시간 로그 스트림 (WebSocket)
```

#### 설정 API

```
GET    /api/v1/config                  # 설정 조회
PUT    /api/v1/config                  # 설정 업데이트
```

### API 사용 예제

```bash
# 워크스페이스 생성
curl -X POST http://localhost:8080/api/v1/workspaces \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-project",
    "path": "/home/user/projects/my-project",
    "description": "내 프로젝트"
  }'

# 태스크 실행
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "workspace_id": "my-project",
    "command": "코드 리뷰를 수행해주세요"
  }'

# 실시간 로그 스트리밍 (JavaScript)
const ws = new WebSocket('ws://localhost:8080/api/v1/logs/stream/task-123');
ws.onmessage = (event) => {
  console.log('Log:', event.data);
};
```

## 프로젝트 구조

```
aicli-web/
├── cmd/                    # 실행 가능한 프로그램의 진입점
│   ├── aicli/             # CLI 도구 메인 패키지
│   └── api/               # API 서버 메인 패키지
├── internal/              # 내부 패키지 (외부 접근 불가)
│   ├── cli/               # CLI 명령어 구현
│   │   └── commands/      # 개별 명령어 구현
│   ├── server/            # API 서버 구현
│   ├── api/               # API 핸들러 및 컨트롤러
│   ├── claude/            # Claude CLI 래퍼
│   ├── docker/            # Docker SDK 통합
│   ├── storage/           # 데이터 저장소 인터페이스
│   ├── models/            # 도메인 모델
│   └── config/            # 설정 관리
├── pkg/                   # 외부 공개 패키지
│   ├── version/           # 버전 정보 관리
│   └── utils/             # 공용 유틸리티
├── build/                 # 빌드 관련 스크립트
├── scripts/               # 개발/배포 자동화 스크립트
├── configs/               # 기본 설정 파일
├── deployments/           # 배포 관련 파일
│   └── docker/           # Docker 관련 파일
├── test/                  # 통합 테스트, E2E 테스트
├── examples/              # 사용 예제
├── docs/                  # 프로젝트 문서
│   ├── claude/            # Claude CLI 통합 문서
│   │   ├── usage-guide.md
│   │   ├── api-reference.md
│   │   ├── configuration.md
│   │   ├── architecture.md
│   │   ├── troubleshooting.md
│   │   └── examples.md
│   ├── cli-design/        # CLI 설계 문서
│   └── development-guide.md # 개발 가이드
├── .aiwf/                 # AIWF 프레임워크 구조
├── .github/               # GitHub 관련 설정
│   └── workflows/        # GitHub Actions 워크플로우
├── go.mod                 # Go 모듈 정의
├── go.sum                 # Go 모듈 체크섬
├── Makefile              # 빌드 자동화
├── .golangci.yml         # 린터 설정
├── .pre-commit-config.yaml # Pre-commit 설정
├── Dockerfile            # 프로덕션 Docker 이미지
├── Dockerfile.dev        # 개발용 Docker 이미지
├── docker-compose.yml    # Docker Compose 설정
├── CONTRIBUTING.md       # 기여 가이드
├── LICENSE              # 라이선스
└── README.md            # 프로젝트 문서 (이 파일)
```

## 개발하기

### 개발 환경 설정

```bash
# 개발 의존성 설치
make setup

# pre-commit 훅 설치
pre-commit install

# 개발 모드 실행 (hot reload)
make dev

# Docker 개발 환경 실행
docker-compose up -d
```

### 빌드 명령어

```bash
# 모든 바이너리 빌드 (CLI + API)
make build

# 특정 바이너리만 빌드
make build-cli          # CLI 도구만
make build-api          # API 서버만

# 멀티 플랫폼 빌드 (Linux, macOS, Windows)
make build-all          # 모든 플랫폼용 빌드

# 의존성 관리
make deps               # 의존성 다운로드 및 정리

# 바이너리 설치
make install            # GOPATH/bin에 설치
```

### 테스트 실행

```bash
# 기본 테스트 (단위 + 통합)
make test

# 모든 테스트 (단위 + 통합 + E2E + 벤치마크)
make test-all

# 테스트 유형별 실행
make test-unit          # 단위 테스트만
make test-integration   # 통합 테스트만
make test-e2e           # E2E 테스트만
make test-benchmark     # 성능 벤치마크
make test-stress        # 스트레스 테스트

# 테스트 커버리지
make test-coverage      # HTML 리포트 생성

# Docker 관련 테스트
make test-docker        # Docker 통합 테스트
make test-container     # 컨테이너 생명주기 테스트

# 워크스페이스 테스트
make test-workspace-integration  # 워크스페이스 통합 테스트
make test-workspace-complete     # 전체 워크스페이스 테스트
```

### 코드 품질 관리

```bash
# 기본 린트
make lint

# 린트 자동 수정
make lint-fix

# 전체 린트 검사
make lint-all

# 린트 리포트 생성
make lint-report

# 코드 포맷팅
make fmt

# 정적 분석
make vet

# 보안 검사
make security

# 종합 품질 검사
make check              # deps + vet + lint + test
```

### Docker 개발 환경

```bash
# Docker 이미지 빌드
make docker             # 프로덕션 이미지
make docker-dev-build   # 개발 이미지 빌드

# 개발 환경 실행
make docker-dev         # 전체 개발 환경 시작
make docker-dev-api     # API 서버만 시작
make docker-dev-cli     # CLI 개발 컨테이너 실행

# 개발 환경 관리
make docker-dev-logs    # 로그 확인
make docker-dev-down    # 개발 환경 종료

# Docker에서 테스트/린트
make docker-dev-test    # Docker에서 테스트 실행
make docker-dev-lint    # Docker에서 린트 실행
```

### 문서 생성

```bash
# Swagger API 문서 생성
make swagger

# Swagger 주석 포맷팅
make swagger-fmt

# GoDoc 로컬 서버 실행
go doc -http=:6060
```

### Pre-commit 훅 관리

```bash
# Pre-commit 훅 설치
make pre-commit-install

# Pre-commit 훅 업데이트
make pre-commit-update

# 모든 파일에 pre-commit 실행
make pre-commit-run
```

### 정리 명령어

```bash
# 빌드 아티팩트 정리
make clean

# 모든 캐시 및 아티팩트 정리
make clean-all

# 릴리스 빌드
make release
```

## 기여하기

AICode Manager 프로젝트에 기여해주셔서 감사합니다! 다음 가이드라인을 따라주세요:

1. 이슈를 먼저 생성하여 작업 내용을 논의해주세요
2. 저장소를 Fork하고 feature 브랜치를 생성하세요
3. 커밋 메시지는 한글로 작성하며 다음 형식을 따라주세요:
   - `feat: 새로운 기능 추가`
   - `fix: 버그 수정`
   - `docs: 문서 업데이트`
   - `test: 테스트 추가 또는 수정`
   - `refactor: 코드 리팩토링`
4. 코드 변경 시 테스트를 함께 작성해주세요
5. `make check`가 통과하는지 확인해주세요
6. Pull Request를 생성해주세요

자세한 내용은 [CONTRIBUTING.md](CONTRIBUTING.md)를 참조하세요.

## 라이선스

이 프로젝트는 MIT 라이선스를 따릅니다. 자세한 내용은 [LICENSE](LICENSE) 파일을 참조하세요.

## 지원 및 문의

- 📋 **이슈 트래커**: [GitHub Issues](https://github.com/moonklabs/aicli-web/issues)
- 💬 **토론**: [GitHub Discussions](https://github.com/moonklabs/aicli-web/discussions)
- 📧 **이메일**: moonklabs@example.com
- 📚 **문서**: [프로젝트 위키](https://github.com/moonklabs/aicli-web/wiki)

---

> 이 프로젝트는 AIWF(AI Workflow) 프레임워크를 사용하여 관리됩니다. 프로젝트 진행 상황은 [.aiwf/00_PROJECT_MANIFEST.md](.aiwf/00_PROJECT_MANIFEST.md)에서 확인할 수 있습니다.