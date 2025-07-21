# AICode Manager (aicli-web)

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![Build Status](https://img.shields.io/github/actions/workflow/status/drumcap/aicli-web/ci.yml?branch=main)](https://github.com/drumcap/aicli-web/actions)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/drumcap/aicli-web)](https://goreportcard.com/report/github.com/drumcap/aicli-web)

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

#### 방법 1: 바이너리 다운로드 (권장)

최신 릴리스에서 운영체제에 맞는 바이너리를 다운로드하세요:

```bash
# Linux (amd64)
wget https://github.com/drumcap/aicli-web/releases/latest/download/aicli-linux-amd64.tar.gz
tar -xzf aicli-linux-amd64.tar.gz
sudo mv aicli /usr/local/bin/

# macOS (Intel)
wget https://github.com/drumcap/aicli-web/releases/latest/download/aicli-darwin-amd64.tar.gz
tar -xzf aicli-darwin-amd64.tar.gz
sudo mv aicli /usr/local/bin/

# macOS (Apple Silicon)
wget https://github.com/drumcap/aicli-web/releases/latest/download/aicli-darwin-arm64.tar.gz
tar -xzf aicli-darwin-arm64.tar.gz
sudo mv aicli /usr/local/bin/
```

#### 방법 2: Go install

```bash
go install github.com/drumcap/aicli-web/cmd/aicli@latest
```

#### 방법 3: 소스에서 빌드

```bash
# 저장소 클론
git clone https://github.com/drumcap/aicli-web.git
cd aicli-web

# 의존성 설치
go mod download

# 빌드
make build

# 바이너리를 PATH에 추가 (선택사항)
sudo cp build/aicli /usr/local/bin/
```

### 빠른 시작

5분 안에 AICode Manager를 시작하세요:

```bash
# 1. 설정 초기화
aicli config init

# 2. Claude API 키 설정
aicli config set claude.api_key "your-api-key"

# 3. 새 워크스페이스 생성
aicli workspace create my-project --path ./my-project

# 4. Claude CLI 실행
aicli task run --workspace my-project "코드 리뷰를 수행해주세요"

# 5. API 서버 시작 (웹 대시보드용)
aicli-api serve --port 8080
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
# 모든 바이너리 빌드
make build

# 특정 바이너리만 빌드
make build-cli          # CLI 도구만
make build-api          # API 서버만

# 멀티 플랫폼 빌드
make build-all          # 모든 플랫폼용 빌드

# Docker 이미지 빌드
make docker             # 프로덕션 이미지
make docker-dev         # 개발 이미지
```

### 테스트 실행

```bash
# 모든 테스트 실행
make test

# 단위 테스트만 실행
make test-unit

# 통합 테스트만 실행
make test-integration

# 테스트 커버리지 리포트
make test-coverage

# 특정 패키지 테스트
go test ./internal/cli/...
```

### 코드 품질 관리

```bash
# 린트 실행
make lint

# 코드 포맷팅
make fmt

# 정적 분석
make vet

# 모든 품질 검사
make check
```

### 문서 생성

```bash
# GoDoc 서버 실행
make docs

# API 문서 생성
make api-docs
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

- 📋 **이슈 트래커**: [GitHub Issues](https://github.com/drumcap/aicli-web/issues)
- 💬 **토론**: [GitHub Discussions](https://github.com/drumcap/aicli-web/discussions)
- 📧 **이메일**: drumcap@example.com
- 📚 **문서**: [프로젝트 위키](https://github.com/drumcap/aicli-web/wiki)

---

> 이 프로젝트는 AIWF(AI Workflow) 프레임워크를 사용하여 관리됩니다. 프로젝트 진행 상황은 [.aiwf/00_PROJECT_MANIFEST.md](.aiwf/00_PROJECT_MANIFEST.md)에서 확인할 수 있습니다.