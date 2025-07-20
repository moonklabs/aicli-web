---
task_id: T04_S02
sprint_sequence_id: S02
status: open
complexity: High
estimated_hours: 8
assigned_to: TBD
created_date: 2025-07-20
last_updated: 2025-07-20T04:00:00Z
---

# Task: Docker 개발 환경 구성

## Description
일관된 개발 환경을 제공하기 위한 Docker 기반 개발 환경을 구성합니다. 로컬 개발, 디버깅, 핫 리로드를 지원하는 컨테이너화된 개발 환경을 구축하여 팀원들 간의 환경 차이를 최소화합니다.

## Goal / Objectives
- Docker 기반 개발 환경 구축
- Hot-reload 기능을 포함한 개발용 Dockerfile 작성
- Docker Compose로 전체 서비스 오케스트레이션
- 개발용 데이터베이스 및 의존 서비스 컨테이너화
- 볼륨 마운팅을 통한 코드 동기화
- 디버깅 및 프로파일링 지원

## Acceptance Criteria
- [ ] Dockerfile.dev 개발용 이미지 작성
- [ ] docker-compose.dev.yml 개발 환경 구성
- [ ] Air를 사용한 Go 애플리케이션 핫 리로드
- [ ] 로컬 코드와 컨테이너 간 실시간 동기화
- [ ] 개발용 SQLite/PostgreSQL 컨테이너 설정
- [ ] 로그 및 디버깅 포트 노출
- [ ] `make docker-dev` 명령어로 개발 환경 실행
- [ ] 개발 환경 설정 가이드 문서화

## Subtasks
- [ ] 개발용 Dockerfile 작성 (멀티스테이지 빌드)
- [ ] Air (Hot reload) 설정 및 통합
- [ ] Docker Compose 개발 환경 구성
- [ ] 데이터베이스 컨테이너 설정 및 초기화
- [ ] 환경 변수 및 시크릿 관리
- [ ] 네트워킹 및 포트 매핑 설정
- [ ] 볼륨 마운팅 및 파일 권한 설정
- [ ] Makefile Docker 관련 타겟 추가
- [ ] 개발 환경 문제 해결 가이드 작성

## Technical Guide

### 개발용 Dockerfile (Dockerfile.dev)

#### 멀티스테이지 빌드 구조
```dockerfile
# 1단계: 개발 환경 베이스
FROM golang:1.21-alpine AS dev-base

# 개발 도구 설치
RUN apk add --no-cache \
    git \
    curl \
    make \
    gcc \
    musl-dev \
    sqlite \
    ca-certificates

# Air (hot reload) 설치
RUN go install github.com/cosmtrek/air@latest

# 디버깅 도구 설치
RUN go install github.com/go-delve/delve/cmd/dlv@latest

# 작업 디렉토리 설정
WORKDIR /app

# Go 모듈 설정
COPY go.mod go.sum ./
RUN go mod download

# 개발 환경 설정
ENV CGO_ENABLED=1
ENV GOOS=linux
ENV GOARCH=amd64

# Air 설정 파일 복사
COPY .air.toml .

# 개발 포트 노출
EXPOSE 8080 2345 6060

# Air로 핫 리로드 실행
CMD ["air", "-c", ".air.toml"]

# 2단계: 프로덕션 빌드 (향후 사용)
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY . .
RUN go mod download
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/api

# 3단계: 프로덕션 런타임
FROM alpine:latest AS production

RUN apk --no-cache add ca-certificates sqlite
WORKDIR /root/

COPY --from=builder /app/main .
COPY --from=builder /app/configs ./configs

EXPOSE 8080
CMD ["./main"]
```

### Air 설정 (.air.toml)

#### 핫 리로드 설정
```toml
root = "."
testdata_dir = "testdata"
tmp_dir = "tmp"

[build]
  args_bin = []
  bin = "./tmp/main"
  cmd = "go build -o ./tmp/main ./cmd/api"
  delay = 1000
  exclude_dir = ["tmp", "vendor", "testdata", "node_modules", ".git"]
  exclude_file = []
  exclude_regex = ["_test.go"]
  exclude_unchanged = false
  follow_symlink = false
  full_bin = ""
  include_dir = []
  include_ext = ["go", "tpl", "tmpl", "html", "yaml", "yml", "json"]
  include_file = []
  kill_delay = "0s"
  log = "build-errors.log"
  poll = false
  poll_interval = 0
  rerun = false
  rerun_delay = 500
  send_interrupt = false
  stop_on_root = false

[color]
  app = ""
  build = "yellow"
  main = "magenta"
  runner = "green"
  watcher = "cyan"

[log]
  main_only = false
  time = false

[misc]
  clean_on_exit = false

[screen]
  clear_on_rebuild = false
  keep_scroll = true
```

### Docker Compose 개발 환경 (docker-compose.dev.yml)

#### 전체 서비스 구성
```yaml
version: '3.8'

services:
  # API 서버 (개발용)
  aicli-api-dev:
    build:
      context: .
      dockerfile: Dockerfile.dev
      target: dev-base
    container_name: aicli-api-dev
    ports:
      - "8080:8080"     # API 서버
      - "2345:2345"     # Delve 디버거
      - "6060:6060"     # pprof 프로파일러
    volumes:
      - .:/app                    # 소스 코드 마운트
      - go-modules:/go/pkg/mod    # Go 모듈 캐시
    environment:
      - AICLI_ENV=development
      - AICLI_PORT=8080
      - AICLI_LOG_LEVEL=debug
      - AICLI_DB_PATH=/app/data/dev.db
      - AIR_BUILD_PATH=/app/tmp
    depends_on:
      - postgres-dev
      - redis-dev
    networks:
      - aicli-dev-network
    restart: unless-stopped

  # PostgreSQL (개발용)
  postgres-dev:
    image: postgres:15-alpine
    container_name: aicli-postgres-dev
    ports:
      - "5432:5432"
    environment:
      - POSTGRES_DB=aicli_dev
      - POSTGRES_USER=aicli
      - POSTGRES_PASSWORD=dev_password
    volumes:
      - postgres-dev-data:/var/lib/postgresql/data
      - ./scripts/init-db.sql:/docker-entrypoint-initdb.d/init.sql
    networks:
      - aicli-dev-network
    restart: unless-stopped

  # Redis (개발용 캐시)
  redis-dev:
    image: redis:7-alpine
    container_name: aicli-redis-dev
    ports:
      - "6379:6379"
    volumes:
      - redis-dev-data:/data
    networks:
      - aicli-dev-network
    restart: unless-stopped

  # Claude CLI 시뮬레이터 (개발용)
  claude-simulator:
    build:
      context: ./test/claude-simulator
      dockerfile: Dockerfile
    container_name: aicli-claude-simulator
    ports:
      - "9000:9000"
    environment:
      - SIMULATOR_MODE=development
    networks:
      - aicli-dev-network
    restart: unless-stopped

  # 로그 수집기 (개발용)
  fluentd-dev:
    image: fluent/fluentd:latest
    container_name: aicli-fluentd-dev
    ports:
      - "24224:24224"
    volumes:
      - ./configs/fluentd.conf:/fluentd/etc/fluent.conf
      - fluentd-logs:/var/log/fluentd
    networks:
      - aicli-dev-network
    restart: unless-stopped

volumes:
  go-modules:
    driver: local
  postgres-dev-data:
    driver: local
  redis-dev-data:
    driver: local
  fluentd-logs:
    driver: local

networks:
  aicli-dev-network:
    driver: bridge
    ipam:
      config:
        - subnet: 172.20.0.0/16
```

### 개발 환경 설정

#### 환경 변수 (.env.dev)
```env
# 개발 환경 설정
AICLI_ENV=development
AICLI_PORT=8080
AICLI_LOG_LEVEL=debug
AICLI_LOG_FORMAT=console

# 데이터베이스 설정
AICLI_DB_TYPE=postgres
AICLI_DB_HOST=postgres-dev
AICLI_DB_PORT=5432
AICLI_DB_NAME=aicli_dev
AICLI_DB_USER=aicli
AICLI_DB_PASSWORD=dev_password

# Redis 설정
AICLI_REDIS_HOST=redis-dev
AICLI_REDIS_PORT=6379
AICLI_REDIS_DB=0

# Claude 시뮬레이터 설정
CLAUDE_API_ENDPOINT=http://claude-simulator:9000
CLAUDE_API_KEY=dev_key_simulation

# Docker 설정
DOCKER_HOST=unix:///var/run/docker.sock
DOCKER_NETWORK=aicli-dev-network

# 디버깅 설정
AICLI_DEBUG=true
AICLI_PPROF_ENABLED=true
```

### Makefile Docker 타겟

#### 개발 환경 관련 명령어
```makefile
# Docker 개발 환경 시작
.PHONY: docker-dev
docker-dev:
	@printf "${BLUE}Starting development environment...${NC}\n"
	docker-compose -f docker-compose.dev.yml up --build -d
	@printf "${GREEN}Development environment started!${NC}\n"
	@printf "API Server: http://localhost:8080\n"
	@printf "PostgreSQL: localhost:5432\n"
	@printf "Redis: localhost:6379\n"

# Docker 개발 환경 중지
.PHONY: docker-dev-stop
docker-dev-stop:
	@printf "${YELLOW}Stopping development environment...${NC}\n"
	docker-compose -f docker-compose.dev.yml down

# Docker 개발 환경 재시작
.PHONY: docker-dev-restart
docker-dev-restart: docker-dev-stop docker-dev

# 로그 확인
.PHONY: docker-dev-logs
docker-dev-logs:
	docker-compose -f docker-compose.dev.yml logs -f

# 개발 컨테이너에 접속
.PHONY: docker-dev-shell
docker-dev-shell:
	docker exec -it aicli-api-dev /bin/sh

# 데이터베이스 초기화
.PHONY: docker-dev-db-reset
docker-dev-db-reset:
	@printf "${YELLOW}Resetting development database...${NC}\n"
	docker-compose -f docker-compose.dev.yml exec postgres-dev psql -U aicli -d aicli_dev -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
	docker-compose -f docker-compose.dev.yml restart aicli-api-dev

# 볼륨 정리
.PHONY: docker-dev-clean
docker-dev-clean:
	@printf "${RED}Cleaning development volumes...${NC}\n"
	docker-compose -f docker-compose.dev.yml down -v
	docker volume prune -f
```

### 디버깅 설정

#### VS Code launch.json (Docker 디버깅)
```json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Debug in Docker",
            "type": "go",
            "request": "attach",
            "mode": "remote",
            "remotePath": "/app",
            "port": 2345,
            "host": "localhost",
            "program": "/app",
            "cwd": "${workspaceFolder}",
            "env": {},
            "args": []
        }
    ]
}
```

#### 디버깅을 위한 Dockerfile 수정
```dockerfile
# 디버깅 모드용 CMD
CMD ["dlv", "debug", "./cmd/api", "--headless", "--listen=:2345", "--api-version=2", "--accept-multiclient"]
```

### 성능 모니터링

#### pprof 통합
```go
// cmd/api/main.go에 추가
import _ "net/http/pprof"

func main() {
    if os.Getenv("AICLI_PPROF_ENABLED") == "true" {
        go func() {
            log.Println(http.ListenAndServe("localhost:6060", nil))
        }()
    }
    // ... 기존 코드
}
```

### 구현 노트
- 개발 편의성과 성능의 균형 고려
- 로컬 환경과 최대한 유사한 설정 유지
- 빠른 피드백을 위한 핫 리로드 최적화
- 디버깅 및 프로파일링 도구 통합
- 네트워크 격리를 통한 서비스 간 통신 테스트

## Output Log

### [날짜 및 시간은 태스크 진행 시 업데이트]

<!-- 작업 진행 로그를 여기에 기록 -->

**상태**: 📋 대기 중