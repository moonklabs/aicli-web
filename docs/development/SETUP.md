# 개발 환경 설정 가이드

이 문서는 AICode Manager 프로젝트의 개발 환경을 설정하는 방법을 안내합니다.

## 사전 요구사항

### 필수 도구
- **Go**: 1.21 이상
- **Docker**: 20.10 이상
- **Make**: 빌드 자동화
- **Git**: 버전 관리

### 권장 도구
- **golangci-lint**: 코드 린팅
- **gosec**: 보안 검사
- **air**: Hot reload 개발

## 설치 가이드

### 1. Go 설치
```bash
# macOS (Homebrew)
brew install go

# Ubuntu/Debian
sudo apt update
sudo apt install golang-go

# 설치 확인
go version
```

### 2. Docker 설치
```bash
# macOS (Homebrew)
brew install --cask docker

# Ubuntu/Debian
sudo apt install docker.io docker-compose

# 사용자를 docker 그룹에 추가
sudo usermod -aG docker $USER

# 설치 확인
docker --version
```

### 3. 개발 도구 설치
```bash
# golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# gosec
go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest

# air (hot reload)
go install github.com/cosmtrek/air@latest
```

## 프로젝트 설정

### 1. 저장소 클론
```bash
# 포크 후 클론 (권장)
git clone https://github.com/YOUR_USERNAME/aicli-web.git
cd aicli-web

# 원본 저장소를 upstream으로 추가
git remote add upstream https://github.com/drumcap/aicli-web.git
```

### 2. 의존성 설치
```bash
# Go 모듈 의존성 설치
make deps

# 또는
go mod download
go mod tidy
```

### 3. 환경 설정
```bash
# .env 파일 생성 (선택)
cp .env.example .env

# 환경 변수 설정 예시
export AICLI_PORT=8080
export AICLI_ENV=development
export AICLI_LOG_LEVEL=debug
```

## 개발 워크플로우

### 1. 코드 작성 및 실행
```bash
# Hot reload 개발 모드
make dev

# 또는 개별 실행
make run-cli    # CLI 도구 실행
make run-api    # API 서버 실행
```

### 2. 코드 품질 확인
```bash
# 코드 포맷팅
make fmt

# 정적 분석
make vet

# 린팅
make lint

# 보안 검사
make security

# 모든 품질 검사
make check
```

### 3. 테스트
```bash
# 모든 테스트 실행
make test

# 단위 테스트만
make test-unit

# 커버리지 리포트
make test-coverage

# 벤치마크 테스트
make test-bench
```

### 4. 빌드
```bash
# 현재 플랫폼용 빌드
make build

# 모든 플랫폼용 빌드
make build-all

# Docker 이미지 빌드
make docker
```

## 디버깅

### 1. VS Code 설정
`.vscode/launch.json` 파일 생성:
```json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Debug CLI",
            "type": "go",
            "request": "launch",
            "mode": "auto",
            "program": "./cmd/aicli",
            "args": ["workspace", "list"],
            "cwd": "${workspaceFolder}"
        },
        {
            "name": "Debug API Server",
            "type": "go",
            "request": "launch",
            "mode": "auto",
            "program": "./cmd/api",
            "env": {
                "AICLI_ENV": "development",
                "AICLI_PORT": "8080"
            },
            "cwd": "${workspaceFolder}"
        }
    ]
}
```

### 2. 로그 확인
```bash
# API 서버 로그
tail -f /var/log/aicli-api.log

# 특정 레벨 로그만 확인
grep "ERROR\|WARN" /var/log/aicli-api.log
```

### 3. 프로파일링
```bash
# CPU 프로파일링
go tool pprof http://localhost:8080/debug/pprof/profile

# 메모리 프로파일링
go tool pprof http://localhost:8080/debug/pprof/heap
```

## 일반적인 문제 해결

### 1. 빌드 오류
```bash
# 모듈 캐시 정리
go clean -modcache

# 의존성 재설치
go mod download
```

### 2. Docker 권한 문제
```bash
# Docker 그룹에 사용자 추가
sudo usermod -aG docker $USER

# 로그아웃 후 재로그인 또는
newgrp docker
```

### 3. 포트 충돌
```bash
# 사용 중인 포트 확인
netstat -tulpn | grep :8080

# 프로세스 종료
kill -9 $(lsof -t -i:8080)
```

## 코드 에디터 설정

### VS Code 확장 프로그램
```json
{
    "recommendations": [
        "golang.go",
        "ms-azuretools.vscode-docker",
        "bradlc.vscode-tailwindcss",
        "esbenp.prettier-vscode"
    ]
}
```

### Go 설정
```json
{
    "go.useLanguageServer": true,
    "go.lintTool": "golangci-lint",
    "go.lintOnSave": "package",
    "go.formatTool": "goimports",
    "go.testFlags": ["-v", "-race"],
    "go.coverOnSave": true
}
```

## 성능 최적화

### 1. 빌드 성능
```bash
# 빌드 캐시 활용
export GOCACHE=/tmp/go-cache
export GOMODCACHE=/tmp/go-mod-cache

# 병렬 빌드
export GOMAXPROCS=$(nproc)
```

### 2. 테스트 성능
```bash
# 병렬 테스트
go test -parallel 4 ./...

# 짧은 테스트만 실행
go test -short ./...
```

## 보안 고려사항

### 1. 개발 환경 보안
- API 키는 환경 변수로 관리
- .env 파일은 git에 커밋하지 않음
- 로컬 데이터베이스 암호화

### 2. 코드 보안
```bash
# 보안 취약점 스캔
make security

# 의존성 취약점 검사
go list -m all | nancy sleuth
```

## 문서 작성

### 1. API 문서
- Swagger/OpenAPI 사용
- 예제 코드 포함
- 에러 응답 명시

### 2. 코드 문서
- GoDoc 형식 주석
- 예제 함수 작성
- 사용법 가이드

## 기여 워크플로우

### 1. 브랜치 전략
```bash
# 기능 브랜치 생성
git checkout -b feature/workspace-create

# 변경사항 커밋
git add .
git commit -m "feat(workspace): add create command"

# upstream과 동기화
git fetch upstream
git rebase upstream/main
```

### 2. Pull Request
- 모든 테스트 통과 확인
- 코드 리뷰 요청
- 문서 업데이트

## 참고 자료

- [Go 공식 문서](https://golang.org/doc/)
- [Gin 프레임워크](https://gin-gonic.com/)
- [Cobra CLI](https://cobra.dev/)
- [Docker 문서](https://docs.docker.com/)
- [프로젝트 기여 가이드](../../CONTRIBUTING.md)