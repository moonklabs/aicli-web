# AICode Manager (aicli-web)

AICode Manager는 Claude CLI를 웹 플랫폼으로 관리하는 시스템입니다. Go 언어로 개발된 네이티브 CLI 도구를 중심으로 각 프로젝트별 격리된 Docker 컨테이너에서 Claude CLI를 실행하고 관리합니다.

## 🚀 프로젝트 개요

AICode Manager는 다음과 같은 핵심 기능을 제공합니다:

- **멀티 프로젝트 지원**: 여러 프로젝트를 동시에 관리하고 실행
- **격리된 실행 환경**: Docker 컨테이너를 통한 프로젝트별 독립 환경
- **실시간 모니터링**: WebSocket을 통한 실시간 로그 스트리밍
- **Git 통합**: 자동 브랜치 생성 및 커밋 관리
- **웹 대시보드**: 직관적인 프로젝트 관리 인터페이스

## 🛠️ 기술 스택

- **Backend**: Go + Gin/Echo
- **Storage**: SQLite/BoltDB
- **Container**: Docker SDK
- **Real-time**: WebSocket/SSE
- **Build**: Make

## 📁 프로젝트 구조

```
aicli-web/
├── cmd/                    # 실행 가능한 프로그램의 진입점
│   ├── aicli/             # CLI 도구 메인 패키지
│   └── api/               # API 서버 메인 패키지
├── internal/              # 내부 패키지 (외부 접근 불가)
│   ├── cli/               # CLI 명령어 구현
│   ├── server/            # API 서버 구현
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
├── test/                  # 통합 테스트, E2E 테스트
├── examples/              # 사용 예제
├── docs/                  # 프로젝트 문서
│   └── cli-design/        # CLI 설계 문서
├── .aiwf/                 # AIWF 프레임워크 구조
├── go.mod                 # Go 모듈 정의
├── Makefile              # 빌드 자동화
└── README.md             # 프로젝트 문서
```

## 🚀 시작하기

### 사전 요구사항

- Go 1.21 이상
- Docker 20.10 이상
- Make

### 설치

```bash
# 저장소 클론
git clone https://github.com/drumcap/aicli-web.git
cd aicli-web

# 의존성 설치
go mod download

# 빌드
make build
```

### 개발 환경 실행

```bash
# CLI 도구 실행
make run-cli

# API 서버 실행
make run-api

# 개발 모드 (hot reload)
make dev
```

## 📝 주요 명령어

```bash
# 빌드
make build              # 모든 바이너리 빌드
make build-cli          # CLI 도구만 빌드
make build-api          # API 서버만 빌드

# 테스트
make test               # 모든 테스트 실행
make test-unit          # 단위 테스트만 실행
make test-integration   # 통합 테스트만 실행
make test-coverage      # 테스트 커버리지 리포트 생성

# 코드 품질
make lint               # 린트 실행
make fmt                # 코드 포맷팅

# Docker
make docker             # Docker 이미지 빌드
make docker-push        # Docker Hub에 푸시

# 정리
make clean              # 빌드 아티팩트 제거
```

## 🏗️ 아키텍처

AICode Manager는 다음과 같은 주요 컴포넌트로 구성됩니다:

1. **Claude CLI 래퍼**: 각 프로젝트별 격리된 Docker 컨테이너에서 Claude CLI 실행
2. **워크스페이스 관리**: 로컬 프로젝트 디렉토리를 Docker 볼륨으로 마운트
3. **API 서버**: RESTful API + WebSocket for 실시간 통신
4. **프론트엔드**: 실시간 로그 뷰어 및 멀티 프로젝트 대시보드

자세한 아키텍처 문서는 [docs/cli-design/architecture.md](docs/cli-design/architecture.md)를 참조하세요.

## 🤝 기여하기

기여를 환영합니다! 다음 가이드라인을 따라주세요:

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'feat: add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 라이선스

이 프로젝트는 MIT 라이선스를 따릅니다. 자세한 내용은 [LICENSE](LICENSE) 파일을 참조하세요.

## 📞 문의

- GitHub Issues: [https://github.com/drumcap/aicli-web/issues](https://github.com/drumcap/aicli-web/issues)
- Email: drumcap@example.com

---

> 이 프로젝트는 AIWF(AI Workflow) 프레임워크를 사용하여 관리됩니다. 프로젝트 진행 상황은 [.aiwf/00_PROJECT_MANIFEST.md](.aiwf/00_PROJECT_MANIFEST.md)에서 확인할 수 있습니다.