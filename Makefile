# AICode Manager Makefile
# Go 기반 CLI 도구 및 API 서버 빌드 자동화

BINARY_NAME_CLI=aicli
BINARY_NAME_API=aicli-api
GO=/home/node/go-local/go/bin/go
GOFLAGS=-v
BUILD_DIR=./build
SCRIPTS_DIR=./scripts

# 버전 정보
VERSION?=0.1.0
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GIT_BRANCH=$(shell git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")

# 멀티플랫폼 빌드 설정
PLATFORMS=linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64
DIST_DIR=./dist

# 빌드 플래그 (최적화 포함)
LDFLAGS=-ldflags "-s -w -extldflags '-static' \
	-X github.com/aicli/aicli-web/pkg/version.Version=${VERSION} \
	-X github.com/aicli/aicli-web/pkg/version.BuildTime=${BUILD_TIME} \
	-X github.com/aicli/aicli-web/pkg/version.GitCommit=${GIT_COMMIT} \
	-X github.com/aicli/aicli-web/pkg/version.GitBranch=${GIT_BRANCH}"

# 컬러 출력
RED=\033[0;31m
GREEN=\033[0;32m
YELLOW=\033[0;33m
BLUE=\033[0;34m
NC=\033[0m # No Color

.PHONY: all build build-cli build-api build-all clean test test-unit test-integration lint lint-fix lint-all lint-report fmt dev help \
	run-cli run-api install docker docker-push vet deps check security release pre-commit-install pre-commit-update pre-commit-run \
	swagger swagger-fmt test-docker test-docker-skip test-container test-docker-bench test-mount test-mount-integration test-status test-status-integration \
	test-security test-security-integration test-security-bench test-workspace-integration test-workspace-performance test-workspace-complete \
	test-e2e-workspace test-workspace-isolation test-workspace-chaos

# 기본 타겟
all: build

# 빌드 타겟
build: build-cli build-api

build-cli:
	@printf "${BLUE}Building CLI tool...${NC}\n"
	@mkdir -p ${BUILD_DIR}
	${GO} build ${GOFLAGS} ${LDFLAGS} -trimpath -o ${BUILD_DIR}/${BINARY_NAME_CLI} ./cmd/aicli
	@printf "${GREEN}✓ CLI tool built successfully${NC}\n"

build-api:
	@printf "${BLUE}Building API server...${NC}\n"
	@mkdir -p ${BUILD_DIR}
	${GO} build ${GOFLAGS} ${LDFLAGS} -trimpath -o ${BUILD_DIR}/${BINARY_NAME_API} ./cmd/api
	@printf "${GREEN}✓ API server built successfully${NC}\n"

# 멀티플랫폼 빌드
build-all:
	@printf "${BLUE}Building for all platforms...${NC}\n"
	@mkdir -p ${DIST_DIR}
	@for platform in ${PLATFORMS}; do \
		OS=$$(echo $$platform | cut -d'/' -f1); \
		ARCH=$$(echo $$platform | cut -d'/' -f2); \
		printf "${YELLOW}Building for $$OS/$$ARCH...${NC}\n"; \
		\
		CLI_OUTPUT=${DIST_DIR}/${BINARY_NAME_CLI}-$$OS-$$ARCH; \
		API_OUTPUT=${DIST_DIR}/${BINARY_NAME_API}-$$OS-$$ARCH; \
		\
		if [ "$$OS" = "windows" ]; then \
			CLI_OUTPUT=$$CLI_OUTPUT.exe; \
			API_OUTPUT=$$API_OUTPUT.exe; \
		fi; \
		\
		CGO_ENABLED=0 GOOS=$$OS GOARCH=$$ARCH ${GO} build ${LDFLAGS} -trimpath -o $$CLI_OUTPUT ./cmd/aicli; \
		CGO_ENABLED=0 GOOS=$$OS GOARCH=$$ARCH ${GO} build ${LDFLAGS} -trimpath -o $$API_OUTPUT ./cmd/api; \
		\
		if [ $$? -eq 0 ]; then \
			printf "${GREEN}✓ Built $$OS/$$ARCH${NC}\n"; \
		else \
			printf "${RED}✗ Failed to build $$OS/$$ARCH${NC}\n"; \
		fi; \
	done
	@printf "${GREEN}✓ Multi-platform build completed${NC}\n"

# 개발 타겟
dev:
	@printf "${BLUE}Starting development mode...${NC}\n"
	@printf "${YELLOW}Note: Install air for hot reload: go install github.com/cosmtrek/air@latest${NC}\n"
	@which air > /dev/null || (printf "${RED}air not installed${NC}\n" && exit 1)
	air

# 의존성 관리
deps:
	@printf "${BLUE}Installing dependencies...${NC}\n"
	${GO} mod download
	${GO} mod tidy
	@printf "${GREEN}✓ Dependencies installed${NC}\n"

# 코드 분석
vet:
	@printf "${BLUE}Running go vet...${NC}\n"
	${GO} vet ./...
	@printf "${GREEN}✓ Code analysis completed${NC}\n"

# 테스트 타겟
test: test-unit test-integration

test-all: test-unit test-integration test-e2e test-benchmark

test-unit:
	@printf "${BLUE}Running unit tests...${NC}\n"
	${GO} test -v -race -cover ./internal/... ./pkg/...
	@printf "${GREEN}✓ Unit tests completed${NC}\n"

test-integration:
	@printf "${BLUE}Running integration tests...${NC}\n"
	${GO} test -v -race -tags=integration ./internal/testing/...
	@printf "${GREEN}✓ Integration tests completed${NC}\n"

test-e2e:
	@printf "${BLUE}Running E2E tests...${NC}\n"
	${GO} test -v -race -tags=e2e ./test/e2e/...
	@printf "${GREEN}✓ E2E tests completed${NC}\n"

test-benchmark:
	@printf "${BLUE}Running performance benchmarks...${NC}\n"
	${GO} test -v -race -tags=benchmark -run=^$$ -bench=. -benchmem ./test/benchmark/...
	@printf "${GREEN}✓ Benchmarks completed${NC}\n"

test-stress:
	@printf "${BLUE}Running stress tests...${NC}\n"
	${GO} test -v -race -tags=benchmark -run=TestStressTest ./test/benchmark/...
	@printf "${GREEN}✓ Stress tests completed${NC}\n"

test-coverage:
	@printf "${BLUE}Generating test coverage report...${NC}\n"
	@mkdir -p reports
	${GO} test -v -race -coverprofile=coverage.out ./...
	${GO} tool cover -html=coverage.out -o coverage.html
	@printf "${GREEN}✓ Coverage report generated: coverage.html${NC}\n"
	@printf "${BLUE}Generating XML test report...${NC}\n"
	@which go-junit-report > /dev/null || (printf "${YELLOW}Installing go-junit-report...${NC}\n" && go install github.com/jstemmer/go-junit-report/v2@latest)
	${GO} test -v ./... 2>&1 | go-junit-report -set-exit-code > reports/test-report.xml || true
	@printf "${GREEN}✓ XML test report generated: reports/test-report.xml${NC}\n"

# 벤치마크 테스트
test-bench:
	@printf "${BLUE}Running benchmark tests...${NC}\n"
	${GO} test -bench=. -benchmem ./...

# Docker 관련 테스트
test-docker:
	@printf "${BLUE}Running Docker integration tests...${NC}\n"
	${GO} test -v -race -timeout=5m ./internal/docker/...
	@printf "${GREEN}✓ Docker tests completed${NC}\n"

test-docker-skip:
	@printf "${BLUE}Running tests without Docker...${NC}\n"
	SKIP_DOCKER_TESTS=true ${GO} test -v -race ./internal/docker/...
	@printf "${GREEN}✓ Tests completed (Docker skipped)${NC}\n"

# 컨테이너 생명주기 테스트
test-container:
	@printf "${BLUE}Running container lifecycle tests...${NC}\n"
	${GO} test -v -race -run="TestContainerManager" ./internal/docker/...
	@printf "${GREEN}✓ Container tests completed${NC}\n"

# Docker 벤치마크 테스트
test-docker-bench:
	@printf "${BLUE}Running Docker benchmark tests...${NC}\n"
	${GO} test -v -race -bench="BenchmarkContainerManager" ./internal/docker/... -benchmem
	@printf "${GREEN}✓ Docker benchmarks completed${NC}\n"
	@printf "${GREEN}✓ Benchmark tests completed${NC}\n"

# 고급 통합 테스트 명령어들
test-advanced:
	@printf "${BLUE}Running advanced integration tests...${NC}\n"
	${GO} test -v -race -timeout=10m ./internal/testing/
	@printf "${GREEN}✓ Advanced integration tests completed${NC}\n"

test-performance:
	@printf "${BLUE}Running performance tests...${NC}\n"
	${GO} test -v -race -timeout=15m -run="TestPerformanceOptimization|TestHighLoadScenario" ./internal/testing/
	@printf "${GREEN}✓ Performance tests completed${NC}\n"

test-chaos:
	@printf "${BLUE}Running chaos engineering tests...${NC}\n"
	${GO} test -v -race -timeout=20m -run="TestChaosEngineering" ./internal/testing/
	@printf "${GREEN}✓ Chaos tests completed${NC}\n"

test-benchmarks:
	@printf "${BLUE}Running comprehensive benchmarks...${NC}\n"
	${GO} test -v -race -bench=. -benchmem -timeout=30m ./internal/testing/
	@printf "${GREEN}✓ Benchmark tests completed${NC}\n"

test-dev:
	@printf "${BLUE}Running development tests (fast)...${NC}\n"
	${GO} test -v -race -short ./internal/testing/
	@printf "${GREEN}✓ Development tests completed${NC}\n"

test-ci:
	@printf "${BLUE}Running CI tests (comprehensive)...${NC}\n"
	${GO} test -v -race -timeout=30m ./internal/testing/
	@printf "${GREEN}✓ CI tests completed${NC}\n"

test-production:
	@printf "${BLUE}Running production stability tests...${NC}\n"
	${GO} test -v -race -timeout=60m -run="TestHighLoadScenario|TestChaosEngineering" ./internal/testing/
	@printf "${GREEN}✓ Production tests completed${NC}\n"

# 마운트 시스템 테스트
test-mount:
	@printf "${BLUE}Running mount system tests...${NC}\n"
	${GO} test -v -short ./internal/docker/mount/...
	@printf "${GREEN}✓ Mount system tests completed${NC}\n"

test-mount-integration:
	@printf "${BLUE}Running mount integration tests...${NC}\n"
	DOCKER_INTEGRATION_TEST=1 ${GO} test -v -timeout=10m ./internal/docker/mount_manager_integration_test.go
	@printf "${GREEN}✓ Mount integration tests completed${NC}\n"

# 코드 품질 타겟
lint:
	@printf "${BLUE}Running linters...${NC}\n"
	@which golangci-lint > /dev/null || (printf "${RED}golangci-lint not installed${NC}\n" && exit 1)
	golangci-lint run
	@printf "${GREEN}✓ Linting completed${NC}\n"

# 자동 수정 가능한 린팅 이슈 수정
lint-fix:
	@printf "${BLUE}Fixing linting issues...${NC}\n"
	@which golangci-lint > /dev/null || (printf "${RED}golangci-lint not installed${NC}\n" && exit 1)
	golangci-lint run --fix
	@printf "${GREEN}✓ Linting issues fixed${NC}\n"

# 전체 린팅 (캐시 무시)
lint-all:
	@printf "${BLUE}Running full lint check...${NC}\n"
	@which golangci-lint > /dev/null || (printf "${RED}golangci-lint not installed${NC}\n" && exit 1)
	golangci-lint run --no-config --enable-all
	@printf "${GREEN}✓ Full lint check completed${NC}\n"

# 린팅 리포트 생성
lint-report:
	@printf "${BLUE}Generating lint report...${NC}\n"
	@mkdir -p reports
	@which golangci-lint > /dev/null || (printf "${RED}golangci-lint not installed${NC}\n" && exit 1)
	golangci-lint run --out-format html > reports/lint-report.html || true
	golangci-lint run --out-format junit-xml > reports/lint-report.xml || true
	@printf "${GREEN}✓ Lint report generated in reports/ directory${NC}\n"

fmt:
	@printf "${BLUE}Formatting code...${NC}\n"
	${GO} fmt ./...
	${GO} mod tidy
	@printf "${GREEN}✓ Code formatting completed${NC}\n"

# 종합 품질 검사
check: deps vet lint test
	@printf "${GREEN}✓ All quality checks passed${NC}\n"

# 보안 검사
security:
	@printf "${BLUE}Running security checks...${NC}\n"
	@which gosec > /dev/null || (printf "${YELLOW}Installing gosec...${NC}\n" && go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest)
	gosec ./...
	@printf "${GREEN}✓ Security check completed${NC}\n"

# Swagger 문서 생성
swagger:
	@printf "${BLUE}Generating Swagger documentation...${NC}\n"
	@if ! command -v swag >/dev/null 2>&1; then \
		printf "${YELLOW}Installing swag...${NC}\n"; \
		go install github.com/swaggo/swag/cmd/swag@latest; \
	fi
	@swag init -g cmd/api/main.go -o docs --parseDependency --parseInternal --parseDepth 1
	@printf "${GREEN}✓ Swagger documentation generated${NC}\n"

swagger-fmt:
	@printf "${BLUE}Formatting Swagger comments...${NC}\n"
	@swag fmt -g cmd/api/main.go
	@printf "${GREEN}✓ Swagger comments formatted${NC}\n"

# 정리 타겟
clean:
	@printf "${BLUE}Cleaning build artifacts...${NC}\n"
	@rm -rf ${BUILD_DIR}
	@rm -rf ${DIST_DIR}
	@rm -rf reports
	@rm -f coverage.out coverage.html
	@printf "${GREEN}✓ Cleanup completed${NC}\n"

# 완전 정리 (의존성 캐시 포함)
clean-all: clean
	@printf "${BLUE}Cleaning all caches...${NC}\n"
	${GO} clean -cache -modcache -testcache
	@printf "${GREEN}✓ All caches cleaned${NC}\n"

# 설치 타겟
install:
	@printf "${BLUE}Installing binaries...${NC}\n"
	${GO} install ${LDFLAGS} ./cmd/aicli
	${GO} install ${LDFLAGS} ./cmd/api
	@printf "${GREEN}✓ Binaries installed${NC}\n"

# Docker 타겟
docker:
	@printf "${BLUE}Building Docker images...${NC}\n"
	docker build -t aicli-web:${VERSION} -f deployments/Dockerfile .
	@printf "${GREEN}✓ Docker image built: aicli-web:${VERSION}${NC}\n"

docker-push:
	@printf "${BLUE}Pushing Docker images...${NC}\n"
	docker tag aicli-web:${VERSION} aicli/aicli-web:${VERSION}
	docker push aicli/aicli-web:${VERSION}
	@printf "${GREEN}✓ Docker image pushed${NC}\n"

# Docker 개발 환경 타겟
.PHONY: docker-dev docker-dev-build docker-dev-cli docker-dev-api docker-dev-test docker-dev-lint docker-dev-down docker-dev-logs

docker-dev-build:
	@printf "${BLUE}Building Docker development images...${NC}\n"
	docker-compose build
	@printf "${GREEN}✓ Development images built${NC}\n"

docker-dev: docker-dev-build
	@printf "${BLUE}Starting Docker development environment...${NC}\n"
	docker-compose up -d
	@printf "${GREEN}✓ Development environment started${NC}\n"
	@printf "${YELLOW}Tip: Use 'make docker-dev-logs' to view logs${NC}\n"

docker-dev-cli:
	@printf "${BLUE}Starting CLI development container...${NC}\n"
	docker-compose run --rm aicli-dev
	@printf "${GREEN}✓ CLI development session ended${NC}\n"

docker-dev-api:
	@printf "${BLUE}Starting API development server...${NC}\n"
	docker-compose up api-dev
	@printf "${GREEN}✓ API server stopped${NC}\n"

docker-dev-test:
	@printf "${BLUE}Running tests in Docker...${NC}\n"
	docker-compose run --rm test
	@printf "${GREEN}✓ Tests completed${NC}\n"

docker-dev-lint:
	@printf "${BLUE}Running linters in Docker...${NC}\n"
	docker-compose run --rm lint
	@printf "${GREEN}✓ Linting completed${NC}\n"

docker-dev-down:
	@printf "${BLUE}Stopping Docker development environment...${NC}\n"
	docker-compose down
	@printf "${GREEN}✓ Development environment stopped${NC}\n"

docker-dev-logs:
	@printf "${BLUE}Showing Docker development logs...${NC}\n"
	docker-compose logs -f

# Docker 디버그 환경
docker-dev-debug:
	@printf "${BLUE}Starting API server in debug mode...${NC}\n"
	docker-compose run --rm -p 2345:2345 api-dev air -c .air.debug.toml
	@printf "${GREEN}✓ Debug session ended${NC}\n"

# 실행 타겟
run-cli:
	@printf "${BLUE}Running CLI tool...${NC}\n"
	${GO} run ./cmd/aicli

run-api:
	@printf "${BLUE}Running API server...${NC}\n"
	${GO} run ./cmd/api

# 릴리스 준비
release: clean check build-all
	@printf "${GREEN}✓ Release build completed${NC}\n"
	@printf "${YELLOW}Release artifacts available in ${DIST_DIR}${NC}\n"

# Pre-commit 관련 타겟
pre-commit-install:
	@printf "${BLUE}Installing pre-commit hooks...${NC}\n"
	@which pre-commit > /dev/null || (printf "${YELLOW}Installing pre-commit...${NC}\n" && pip install pre-commit)
	pre-commit install
	pre-commit install --hook-type commit-msg
	@printf "${GREEN}✓ Pre-commit hooks installed${NC}\n"

pre-commit-update:
	@printf "${BLUE}Updating pre-commit hooks...${NC}\n"
	@which pre-commit > /dev/null || (printf "${RED}pre-commit not installed${NC}\n" && exit 1)
	pre-commit autoupdate
	@printf "${GREEN}✓ Pre-commit hooks updated${NC}\n"

pre-commit-run:
	@printf "${BLUE}Running pre-commit on all files...${NC}\n"
	@which pre-commit > /dev/null || (printf "${RED}pre-commit not installed${NC}\n" && exit 1)
	pre-commit run --all-files
	@printf "${GREEN}✓ Pre-commit checks completed${NC}\n"

# 도움말 타겟
help:
	@printf "${BLUE}AICode Manager Build System${NC}\n"
	@printf "${BLUE}===========================${NC}\n"
	@echo ""
	@printf "${YELLOW}🔨 Build Commands:${NC}\n"
	@echo "  make build          - Build both CLI and API for current platform"
	@echo "  make build-cli      - Build CLI tool only"
	@echo "  make build-api      - Build API server only"
	@echo "  make build-all      - Build for all platforms (linux/darwin/windows)"
	@echo "  make install        - Install binaries to GOPATH/bin"
	@echo ""
	@printf "${YELLOW}🧪 Test Commands:${NC}\n"
	@echo "  make test           - Run basic tests (unit + integration)"
	@echo "  make test-all       - Run all tests (unit + integration + e2e + benchmarks)"
	@echo "  make test-unit      - Run unit tests only"
	@echo "  make test-integration - Run integration tests only"
	@echo "  make test-e2e       - Run end-to-end tests only"
	@echo "  make test-benchmark - Run performance benchmarks"
	@echo "  make test-stress    - Run stress tests"
	@echo "  make test-coverage  - Generate test coverage report"
	@echo "  make test-bench     - Run benchmark tests"
	@echo ""
	@printf "${YELLOW}🔍 Quality Commands:${NC}\n"
	@echo "  make lint           - Run golangci-lint"
	@echo "  make fmt            - Format code and tidy modules"
	@echo "  make vet            - Run go vet analysis"
	@echo "  make security       - Run security checks with gosec"
	@echo "  make check          - Run all quality checks (deps + vet + lint + test)"
	@echo ""
	@printf "${YELLOW}🚀 Development Commands:${NC}\n"
	@echo "  make dev            - Start development mode with hot reload"
	@echo "  make run-cli        - Run CLI tool"
	@echo "  make run-api        - Run API server"
	@echo "  make deps           - Install and tidy dependencies"
	@echo ""
	@printf "${YELLOW}🔗 Pre-commit Commands:${NC}\n"
	@echo "  make pre-commit-install - Install pre-commit hooks"
	@echo "  make pre-commit-update  - Update pre-commit hooks"
	@echo "  make pre-commit-run     - Run pre-commit on all files"
	@echo ""
	@printf "${YELLOW}🐳 Docker Commands:${NC}\n"
	@echo "  make docker         - Build Docker image"
	@echo "  make docker-push    - Push Docker image to registry"
	@echo "  make docker-dev     - Start development environment"
	@echo "  make docker-dev-cli - Run CLI in development container"
	@echo "  make docker-dev-api - Run API server with hot reload"
	@echo "  make docker-dev-test - Run tests in Docker"
	@echo "  make docker-dev-lint - Run linters in Docker"
	@echo "  make docker-dev-debug - Start API in debug mode"
	@echo "  make docker-dev-logs - View development logs"
	@echo "  make docker-dev-down - Stop development environment"
	@echo ""
	@printf "${YELLOW}🗑️  Cleanup Commands:${NC}\n"
	@echo "  make clean          - Clean build artifacts"
	@echo "  make clean-all      - Clean all caches and artifacts"
	@echo ""
	@printf "${YELLOW}📦 Release Commands:${NC}\n"
	@echo "  make release        - Build release artifacts for all platforms"
	@echo ""
	@printf "${YELLOW}💡 Info Commands:${NC}\n"
	@echo "  make help           - Show this help message"
	@echo ""
	@printf "${GREEN}Version: ${VERSION}${NC}\n"
	@printf "${GREEN}Git Commit: ${GIT_COMMIT}${NC}\n"
	@printf "${GREEN}Git Branch: ${GIT_BRANCH}${NC}\n"

# 상태 추적 시스템 테스트
test-status:
	@printf "${BLUE}Running status tracking system tests...${NC}\n"
	${GO} test -v -race ./internal/docker/status/...
	@printf "${GREEN}✓ Status tracking tests completed${NC}\n"

# 상태 추적 시스템 통합 테스트
test-status-integration:
	@printf "${BLUE}Running status tracking integration tests...${NC}\n"
	${GO} test -v -race -run="TestIntegration_.*" ./internal/docker/status/...
	@printf "${GREEN}✓ Status tracking integration tests completed${NC}\n"

# 보안 격리 시스템 테스트
test-security:
	@printf "${BLUE}Running security module tests...${NC}\n"
	${GO} test -v -race ./internal/docker/security/...
	@printf "${GREEN}✓ Security module tests completed${NC}\n"

test-security-integration:
	@printf "${BLUE}Running security integration tests...${NC}\n"
	DOCKER_INTEGRATION_TEST=1 ${GO} test -v -timeout=10m ./internal/docker/security/...
	@printf "${GREEN}✓ Security integration tests completed${NC}\n"

test-security-bench:
	@printf "${BLUE}Running security benchmark tests...${NC}\n"
	${GO} test -v -race -bench="Benchmark.*" ./internal/docker/security/... -benchmem
	@printf "${GREEN}✓ Security benchmark tests completed${NC}\n"

# 워크스페이스 통합 테스트 타겟
test-workspace-integration:
	@printf "${BLUE}Running workspace integration tests...${NC}\n"
	${GO} test -v -race -timeout 10m ./test/integration/workspace_basic_test.go
	@printf "${GREEN}✓ Workspace integration tests completed${NC}\n"

test-workspace-performance:
	@printf "${BLUE}Running workspace performance tests...${NC}\n"
	${GO} test -v -race -timeout 15m ./test/integration/workspace_performance_simple_test.go
	@printf "${GREEN}✓ Workspace performance tests completed${NC}\n"

test-e2e-workspace:
	@printf "${BLUE}Running workspace E2E tests...${NC}\n"
	${GO} test -v -race -timeout 15m ./test/e2e/workspace_complete_flow_test.go
	@printf "${GREEN}✓ Workspace E2E tests completed${NC}\n"

test-workspace-complete:
	@printf "${BLUE}Running complete workspace test suite...${NC}\n"
	@if ! docker info >/dev/null 2>&1; then \
		printf "${RED}✗ Docker daemon not available, running E2E tests only${NC}\n"; \
		make test-e2e-workspace; \
	else \
		make test-workspace-integration && \
		make test-workspace-performance && \
		make test-e2e-workspace; \
	fi
	@printf "${GREEN}✓ Complete workspace test suite completed${NC}\n"

test-workspace-isolation:
	@printf "${BLUE}Running workspace isolation tests...${NC}\n"
	@if ! docker info >/dev/null 2>&1; then \
		printf "${RED}✗ Docker daemon not available, skipping isolation tests${NC}\n"; \
		exit 1; \
	fi
	${GO} test -v -race -timeout 10m -run "TestWorkspaceResourceIsolation|TestSecurityConstraints|TestMultiUserWorkspaceIsolation" ./test/integration/... ./test/e2e/...
	@printf "${GREEN}✓ Workspace isolation tests completed${NC}\n"

test-workspace-chaos:
	@printf "${BLUE}Running workspace chaos engineering tests...${NC}\n"
	@if ! docker info >/dev/null 2>&1; then \
		printf "${RED}✗ Docker daemon not available, skipping chaos tests${NC}\n"; \
		exit 1; \
	fi
	${GO} test -v -race -timeout 20m -run "TestErrorRecoveryScenarios|TestConcurrentWorkspaceOperations" ./test/integration/...
	@printf "${GREEN}✓ Workspace chaos tests completed${NC}\n"