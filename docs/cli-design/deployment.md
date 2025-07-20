# CLI 기반 배포 가이드

## 🚀 개요

Go로 작성된 네이티브 CLI 도구와 API 서버의 빌드, 패키징, 배포 방법을 설명합니다.

## 🏗️ 빌드 시스템

### 1. Makefile 구성

```makefile
# Makefile
.PHONY: all build build-all clean test lint install docker help

# 변수 정의
BINARY_NAME := aicli
API_NAME := aicli-api
VERSION := $(shell git describe --tags --always --dirty)
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
COMMIT := $(shell git rev-parse --short HEAD)

# Go 관련 변수
GO := go
GOFLAGS := -v
LDFLAGS := -ldflags "-s -w \
	-X github.com/yourusername/aicli/pkg/version.Version=$(VERSION) \
	-X github.com/yourusername/aicli/pkg/version.BuildTime=$(BUILD_TIME) \
	-X github.com/yourusername/aicli/pkg/version.Commit=$(COMMIT)"

# 플랫폼별 빌드 변수
PLATFORMS := darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64
PLATFORM_TARGETS := $(addprefix build-, $(subst /,-,$(PLATFORMS)))

# 기본 타겟
all: test build

# 로컬 빌드
build: build-cli build-api

build-cli:
	@echo "Building CLI for local platform..."
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/aicli

build-api:
	@echo "Building API server..."
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o bin/$(API_NAME) ./cmd/api

# 크로스 플랫폼 빌드
build-all: $(PLATFORM_TARGETS)

$(PLATFORM_TARGETS): build-%:
	$(eval GOOS := $(word 1,$(subst -, ,$*)))
	$(eval GOARCH := $(word 2,$(subst -, ,$*)))
	$(eval OUTPUT := $(BINARY_NAME)-$(GOOS)-$(GOARCH))
	@if [ "$(GOOS)" = "windows" ]; then OUTPUT="$(OUTPUT).exe"; fi; \
	echo "Building for $(GOOS)/$(GOARCH)..."; \
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=0 \
		$(GO) build $(GOFLAGS) $(LDFLAGS) -o dist/$(OUTPUT) ./cmd/aicli

# 정적 빌드 (Alpine Linux용)
build-static:
	@echo "Building static binary..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
		$(GO) build $(GOFLAGS) \
		-ldflags "-s -w -extldflags '-static' \
			-X github.com/yourusername/aicli/pkg/version.Version=$(VERSION)" \
		-o dist/$(BINARY_NAME)-linux-amd64-static ./cmd/aicli

# 테스트
test:
	@echo "Running tests..."
	$(GO) test -v -race -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

test-integration:
	@echo "Running integration tests..."
	$(GO) test -v -tags=integration ./tests/integration/...

# 벤치마크
bench:
	@echo "Running benchmarks..."
	$(GO) test -bench=. -benchmem ./...

# 린트
lint:
	@echo "Running linters..."
	golangci-lint run --timeout=5m

# 포맷
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...
	goimports -w .

# 의존성 관리
deps:
	@echo "Downloading dependencies..."
	$(GO) mod download
	$(GO) mod tidy

# 설치
install: build
	@echo "Installing binaries..."
	sudo cp bin/$(BINARY_NAME) /usr/local/bin/
	sudo cp bin/$(API_NAME) /usr/local/bin/

# Docker 빌드
docker: docker-cli docker-api

docker-cli:
	@echo "Building CLI Docker image..."
	docker build -f docker/cli/Dockerfile -t $(BINARY_NAME):$(VERSION) .

docker-api:
	@echo "Building API Docker image..."
	docker build -f docker/api/Dockerfile -t $(API_NAME):$(VERSION) .

# 릴리즈
release: clean build-all
	@echo "Creating release artifacts..."
	@mkdir -p release
	@for file in dist/*; do \
		tar czf release/$$(basename $$file).tar.gz -C dist $$(basename $$file); \
	done
	@cd release && sha256sum *.tar.gz > checksums.txt

# 정리
clean:
	@echo "Cleaning..."
	rm -rf bin/ dist/ release/ coverage.* *.test

# 도움말
help:
	@echo "Available targets:"
	@echo "  make build        - Build for local platform"
	@echo "  make build-all    - Build for all platforms"
	@echo "  make test         - Run tests"
	@echo "  make lint         - Run linters"
	@echo "  make install      - Install binaries"
	@echo "  make docker       - Build Docker images"
	@echo "  make release      - Create release artifacts"
	@echo "  make clean        - Clean build artifacts"
```

### 2. 빌드 스크립트

```bash
#!/bin/bash
# scripts/build.sh

set -euo pipefail

# 색상 정의
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 함수 정의
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# 버전 정보
VERSION=$(git describe --tags --always --dirty)
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
COMMIT=$(git rev-parse --short HEAD)

log_info "Building AICLI CLI v${VERSION}"
log_info "Commit: ${COMMIT}"
log_info "Build time: ${BUILD_TIME}"

# Go 버전 확인
GO_VERSION=$(go version | awk '{print $3}')
REQUIRED_GO_VERSION="go1.21"

if [[ "${GO_VERSION}" < "${REQUIRED_GO_VERSION}" ]]; then
    log_error "Go version ${REQUIRED_GO_VERSION} or higher is required (current: ${GO_VERSION})"
    exit 1
fi

# 의존성 다운로드
log_info "Downloading dependencies..."
go mod download

# 테스트 실행
if [[ "${SKIP_TESTS:-false}" != "true" ]]; then
    log_info "Running tests..."
    go test -v -race ./...
else
    log_warning "Skipping tests (SKIP_TESTS=true)"
fi

# 빌드
log_info "Building binaries..."
make build-all

# 완료
log_info "Build completed successfully!"
ls -la dist/
```

## 🐳 Docker 이미지

### 1. CLI Docker 이미지

```dockerfile
# docker/cli/Dockerfile
# 빌드 스테이지
FROM golang:1.21-alpine AS builder

# 빌드 도구 설치
RUN apk add --no-cache git make

# 작업 디렉토리
WORKDIR /build

# 의존성 캐싱
COPY go.mod go.sum ./
RUN go mod download

# 소스 복사 및 빌드
COPY . .
RUN make build-static

# 실행 스테이지
FROM alpine:3.19

# 필수 패키지 설치
RUN apk add --no-cache \
    ca-certificates \
    git \
    openssh-client \
    docker-cli

# 사용자 생성
RUN adduser -D -u 1000 aicli

# 바이너리 복사
COPY --from=builder /build/dist/aicli-linux-amd64-static /usr/local/bin/aicli

# 설정 디렉토리
RUN mkdir -p /home/aicli/.aicli && chown -R aicli:aicli /home/aicli

USER aicli
WORKDIR /home/aicli

ENTRYPOINT ["aicli"]
CMD ["--help"]
```

### 2. API 서버 Docker 이미지

```dockerfile
# docker/api/Dockerfile
# 빌드 스테이지
FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git make

WORKDIR /build

# 의존성 캐싱
COPY go.mod go.sum ./
RUN go mod download

# 빌드
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o aicli-api ./cmd/api

# 실행 스테이지
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

# 사용자 생성
RUN addgroup -g 1000 -S aicli && \
    adduser -u 1000 -S aicli -G aicli

# 바이너리 복사
COPY --from=builder /build/aicli-api /usr/local/bin/aicli-api

# 설정 파일
COPY --chown=aicli:aicli configs/production.yaml /etc/aicli/config.yaml

USER aicli

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/usr/local/bin/aicli-api", "health"]

ENTRYPOINT ["aicli-api"]
CMD ["server", "--config", "/etc/aicli/config.yaml"]
```

## 📦 패키징

### 1. Homebrew Formula

```ruby
# Formula/aicli.rb
class Terry < Formula
  desc "AI-powered code management CLI"
  homepage "https://github.com/yourusername/aicli"
  version "1.0.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/yourusername/aicli/releases/download/v#{version}/aicli-darwin-arm64.tar.gz"
      sha256 "YOUR_SHA256_HERE"
    else
      url "https://github.com/yourusername/aicli/releases/download/v#{version}/aicli-darwin-amd64.tar.gz"
      sha256 "YOUR_SHA256_HERE"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/yourusername/aicli/releases/download/v#{version}/aicli-linux-arm64.tar.gz"
      sha256 "YOUR_SHA256_HERE"
    else
      url "https://github.com/yourusername/aicli/releases/download/v#{version}/aicli-linux-amd64.tar.gz"
      sha256 "YOUR_SHA256_HERE"
    end
  end

  def install
    bin.install "aicli"
    
    # 자동 완성 스크립트
    generate_completions_from_executable(bin/"aicli", "completion")
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/aicli version")
  end
end
```

### 2. APT 패키지 (Debian/Ubuntu)

```bash
#!/bin/bash
# scripts/build-deb.sh

VERSION=$1
ARCH=${2:-amd64}

# 디렉토리 구조 생성
mkdir -p aicli-${VERSION}/DEBIAN
mkdir -p aicli-${VERSION}/usr/bin
mkdir -p aicli-${VERSION}/usr/share/doc/aicli
mkdir -p aicli-${VERSION}/etc/aicli

# 바이너리 복사
cp dist/aicli-linux-${ARCH} aicli-${VERSION}/usr/bin/aicli
chmod 755 aicli-${VERSION}/usr/bin/aicli

# 문서 복사
cp README.md LICENSE aicli-${VERSION}/usr/share/doc/aicli/

# Control 파일 생성
cat > aicli-${VERSION}/DEBIAN/control << EOF
Package: aicli
Version: ${VERSION}
Section: utils
Priority: optional
Architecture: ${ARCH}
Maintainer: Your Name <your.email@example.com>
Description: AI-powered code management CLI
 Terry is a command-line interface for managing AI-powered coding tasks.
 It allows you to create workspaces, run Claude AI tasks, and monitor progress.
Homepage: https://github.com/yourusername/aicli
EOF

# postinst 스크립트
cat > aicli-${VERSION}/DEBIAN/postinst << 'EOF'
#!/bin/bash
set -e

# 설정 디렉토리 생성
mkdir -p /etc/aicli

# 자동 완성 설정
if [ -d /etc/bash_completion.d ]; then
    aicli completion bash > /etc/bash_completion.d/aicli
fi

exit 0
EOF
chmod 755 aicli-${VERSION}/DEBIAN/postinst

# 패키지 빌드
dpkg-deb --build aicli-${VERSION}
```

### 3. RPM 패키지 (RHEL/CentOS/Fedora)

```spec
# aicli.spec
Name:           aicli
Version:        1.0.0
Release:        1%{?dist}
Summary:        AI-powered code management CLI

License:        MIT
URL:            https://github.com/yourusername/aicli
Source0:        %{name}-%{version}.tar.gz

BuildRequires:  golang >= 1.21

%description
Terry is a command-line interface for managing AI-powered coding tasks.

%prep
%autosetup

%build
make build

%install
rm -rf $RPM_BUILD_ROOT
mkdir -p $RPM_BUILD_ROOT%{_bindir}
install -m 755 bin/aicli $RPM_BUILD_ROOT%{_bindir}/aicli

%files
%license LICENSE
%doc README.md
%{_bindir}/aicli

%changelog
* Mon Jan 20 2025 Your Name <your.email@example.com> - 1.0.0-1
- Initial package
```

## 🚀 배포 전략

### 1. GitHub Actions CI/CD

```yaml
# .github/workflows/release.yml
name: Release

on:
  push:
    tags:
      - 'v*'

permissions:
  contents: write

jobs:
  build:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        include:
          - {os: ubuntu-latest, goos: linux, goarch: amd64}
          - {os: ubuntu-latest, goos: linux, goarch: arm64}
          - {os: macos-latest, goos: darwin, goarch: amd64}
          - {os: macos-latest, goos: darwin, goarch: arm64}
          - {os: windows-latest, goos: windows, goarch: amd64}

    steps:
    - uses: actions/checkout@v4

    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.21'

    - name: Build
      env:
        GOOS: ${{ matrix.goos }}
        GOARCH: ${{ matrix.goarch }}
      run: |
        make build-${{ matrix.goos }}-${{ matrix.goarch }}

    - name: Upload artifacts
      uses: actions/upload-artifact@v3
      with:
        name: binaries
        path: dist/

  release:
    needs: build
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    
    - name: Download artifacts
      uses: actions/download-artifact@v3
      with:
        name: binaries
        path: dist/

    - name: Create Release
      uses: softprops/action-gh-release@v1
      with:
        files: dist/*
        generate_release_notes: true
        
  docker:
    needs: release
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    
    - name: Set up Docker Buildx
      uses: docker/setup-buildx-action@v2
    
    - name: Login to Docker Hub
      uses: docker/login-action@v2
      with:
        username: ${{ secrets.DOCKER_USERNAME }}
        password: ${{ secrets.DOCKER_PASSWORD }}
    
    - name: Build and push
      uses: docker/build-push-action@v4
      with:
        context: .
        file: ./docker/api/Dockerfile
        platforms: linux/amd64,linux/arm64
        push: true
        tags: |
          ${{ secrets.DOCKER_USERNAME }}/aicli-api:latest
          ${{ secrets.DOCKER_USERNAME }}/aicli-api:${{ github.ref_name }}
```

### 2. 설치 스크립트

```bash
#!/bin/bash
# install.sh

set -euo pipefail

# 변수
REPO="yourusername/aicli"
BINARY="aicli"

# OS와 아키텍처 감지
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "${ARCH}" in
    x86_64)
        ARCH="amd64"
        ;;
    aarch64|arm64)
        ARCH="arm64"
        ;;
    *)
        echo "Unsupported architecture: ${ARCH}"
        exit 1
        ;;
esac

# 최신 버전 가져오기
VERSION=$(curl -s https://api.github.com/repos/${REPO}/releases/latest | grep tag_name | cut -d '"' -f 4)

if [ -z "${VERSION}" ]; then
    echo "Failed to get latest version"
    exit 1
fi

echo "Installing AICLI ${VERSION} for ${OS}/${ARCH}..."

# 다운로드 URL
URL="https://github.com/${REPO}/releases/download/${VERSION}/${BINARY}-${OS}-${ARCH}"

# Windows인 경우 .exe 추가
if [ "${OS}" = "windows" ]; then
    URL="${URL}.exe"
    BINARY="${BINARY}.exe"
fi

# 다운로드
curl -L -o /tmp/${BINARY} "${URL}"
chmod +x /tmp/${BINARY}

# 설치
sudo mv /tmp/${BINARY} /usr/local/bin/${BINARY}

# 자동 완성 설치
if [ "${OS}" != "windows" ]; then
    # Bash
    if [ -d /etc/bash_completion.d ]; then
        ${BINARY} completion bash | sudo tee /etc/bash_completion.d/${BINARY} > /dev/null
    fi
    
    # Zsh
    if [ -d "${HOME}/.oh-my-zsh/custom/plugins" ]; then
        mkdir -p "${HOME}/.oh-my-zsh/custom/plugins/${BINARY}"
        ${BINARY} completion zsh > "${HOME}/.oh-my-zsh/custom/plugins/${BINARY}/_${BINARY}"
    fi
fi

echo "AICLI ${VERSION} installed successfully!"
${BINARY} version
```

## 🏃 런타임 설정

### 1. 시스템 서비스 (systemd)

```ini
# /etc/systemd/system/aicli-api.service
[Unit]
Description=Terry API Server
Documentation=https://github.com/yourusername/aicli
After=network.target

[Service]
Type=simple
User=aicli
Group=aicli
ExecStart=/usr/local/bin/aicli-api server --config /etc/aicli/config.yaml
Restart=on-failure
RestartSec=5

# 보안 설정
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/aicli /var/log/aicli

# 리소스 제한
LimitNOFILE=65535
LimitNPROC=4096

# 환경 변수
Environment="TERRY_ENV=production"
EnvironmentFile=-/etc/aicli/env

[Install]
WantedBy=multi-user.target
```

### 2. 환경 설정

```yaml
# /etc/aicli/config.yaml
server:
  host: 0.0.0.0
  port: 8080
  read_timeout: 30s
  write_timeout: 30s
  
database:
  type: sqlite
  path: /var/lib/aicli/aicli.db
  
logging:
  level: info
  format: json
  output: /var/log/aicli/api.log
  
security:
  jwt_secret: ${JWT_SECRET}
  allowed_origins:
    - http://localhost:3000
    - https://aicli.example.com
    
docker:
  host: unix:///var/run/docker.sock
  network: aicli_network
  
claude:
  max_concurrent_sessions: 10
  session_timeout: 30m
  
monitoring:
  metrics_enabled: true
  metrics_port: 9090
  trace_enabled: true
  trace_endpoint: http://jaeger:14268/api/traces
```

## 📊 모니터링 설정

### 1. Prometheus 설정

```yaml
# prometheus.yml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'aicli-api'
    static_configs:
      - targets: ['localhost:9090']
    
  - job_name: 'node-exporter'
    static_configs:
      - targets: ['localhost:9100']
```

### 2. Grafana 대시보드

```json
{
  "dashboard": {
    "title": "Terry Monitoring",
    "panels": [
      {
        "title": "API Request Rate",
        "targets": [
          {
            "expr": "rate(http_requests_total[5m])"
          }
        ]
      },
      {
        "title": "Task Execution Time",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(task_duration_seconds_bucket[5m]))"
          }
        ]
      }
    ]
  }
}
```

## 🔐 프로덕션 보안

### 1. 보안 체크리스트

```bash
#!/bin/bash
# scripts/security-check.sh

echo "Running security checks..."

# 의존성 취약점 검사
go list -json -deps ./... | nancy sleuth

# 정적 분석
gosec ./...

# 라이선스 확인
go-licenses check ./...

# Docker 이미지 스캔
trivy image aicli-api:latest
```

### 2. 강화 설정

```bash
# 파일 권한
chmod 600 /etc/aicli/config.yaml
chmod 700 /var/lib/aicli

# SELinux 컨텍스트 (RHEL/CentOS)
semanage fcontext -a -t bin_t /usr/local/bin/aicli
restorecon -v /usr/local/bin/aicli

# AppArmor 프로파일 (Ubuntu)
aa-complain /usr/local/bin/aicli-api
```

## 🎯 배포 완료 확인

```bash
#!/bin/bash
# scripts/verify-deployment.sh

echo "Verifying deployment..."

# API 헬스체크
curl -f http://localhost:8080/health || exit 1

# CLI 버전 확인
aicli version || exit 1

# 로그 확인
tail -n 100 /var/log/aicli/api.log

echo "Deployment verified successfully!"
```