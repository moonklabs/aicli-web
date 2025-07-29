# AICode Manager Agent Docker Image Makefile

IMAGE_NAME := aicli-agent
IMAGE_TAG := latest
DOCKER_REGISTRY ?= 
FULL_IMAGE_NAME := $(if $(DOCKER_REGISTRY),$(DOCKER_REGISTRY)/$(IMAGE_NAME),$(IMAGE_NAME))

.PHONY: all build push test clean help

# 기본 타겟
all: build

# Docker 이미지 빌드
build:
	@echo "Building Docker image: $(FULL_IMAGE_NAME):$(IMAGE_TAG)"
	docker build \
		--tag $(FULL_IMAGE_NAME):$(IMAGE_TAG) \
		--file Dockerfile \
		.
	@echo "Successfully built $(FULL_IMAGE_NAME):$(IMAGE_TAG)"

# Docker 이미지 푸시
push: build
	@echo "Pushing Docker image: $(FULL_IMAGE_NAME):$(IMAGE_TAG)"
	docker push $(FULL_IMAGE_NAME):$(IMAGE_TAG)

# 개발용 테스트 실행
test: build
	@echo "Running test container..."
	docker run --rm -it \
		--name aicli-agent-test \
		-v $$(pwd)/test-workspace:/workspace \
		$(FULL_IMAGE_NAME):$(IMAGE_TAG) \
		--version

# Docker Compose로 개발 환경 실행
dev: build
	@echo "Starting development environment..."
	docker-compose up -d
	@echo "Development environment is running"
	@echo "To view logs: docker-compose logs -f"
	@echo "To stop: docker-compose down"

# Docker Compose 중지
down:
	docker-compose down -v

# 로그 확인
logs:
	docker-compose logs -f agent

# 컨테이너 쉘 접속
shell:
	docker-compose exec agent bash

# 이미지 및 컨테이너 정리
clean:
	@echo "Cleaning up..."
	-docker-compose down -v
	-docker rmi $(FULL_IMAGE_NAME):$(IMAGE_TAG)
	-docker volume prune -f
	@echo "Cleanup complete"

# 이미지 크기 확인
size: build
	@echo "Image size information:"
	@docker images $(FULL_IMAGE_NAME):$(IMAGE_TAG) --format "table {{.Repository}}\t{{.Tag}}\t{{.Size}}"

# 이미지 레이어 분석
layers: build
	@echo "Analyzing image layers..."
	docker history $(FULL_IMAGE_NAME):$(IMAGE_TAG)

# 보안 스캔 (Trivy 필요)
scan: build
	@echo "Scanning image for vulnerabilities..."
	@if command -v trivy >/dev/null 2>&1; then \
		trivy image $(FULL_IMAGE_NAME):$(IMAGE_TAG); \
	else \
		echo "Trivy not installed. Install with: brew install aquasecurity/trivy/trivy"; \
	fi

# 도움말
help:
	@echo "AICode Manager Agent Docker Image Makefile"
	@echo ""
	@echo "Available targets:"
	@echo "  make build    - Build the Docker image"
	@echo "  make push     - Build and push the image to registry"
	@echo "  make test     - Run a test container"
	@echo "  make dev      - Start development environment with docker-compose"
	@echo "  make down     - Stop development environment"
	@echo "  make logs     - View container logs"
	@echo "  make shell    - Access container shell"
	@echo "  make clean    - Remove images and volumes"
	@echo "  make size     - Show image size"
	@echo "  make layers   - Analyze image layers"
	@echo "  make scan     - Security scan with Trivy"
	@echo "  make help     - Show this help message"
	@echo ""
	@echo "Environment variables:"
	@echo "  DOCKER_REGISTRY - Docker registry URL (optional)"
	@echo "  IMAGE_TAG      - Image tag (default: latest)"