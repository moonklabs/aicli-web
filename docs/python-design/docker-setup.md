# Docker 환경 설정 가이드

## 🐳 개요

이 문서는 AICode Manager의 Docker 기반 환경 구성 방법을 설명합니다. 각 워크스페이스는 독립적인 Docker 컨테이너에서 실행되며, 안전하고 격리된 실행 환경을 제공합니다.

## 📁 프로젝트 구조

```
aicli-web/
├── docker/
│   ├── api/
│   │   └── Dockerfile          # API 서버 이미지
│   ├── claude/
│   │   └── Dockerfile          # Claude CLI 이미지
│   ├── workspace/
│   │   └── Dockerfile          # 워크스페이스 템플릿 이미지
│   └── nginx/
│       ├── Dockerfile          # Nginx 이미지
│       └── nginx.conf          # Nginx 설정
├── docker-compose.yml          # 개발 환경
├── docker-compose.prod.yml     # 프로덕션 환경
└── .env.example               # 환경 변수 예제
```

## 🔧 Docker 이미지 구성

### 1. API 서버 이미지
```dockerfile
# docker/api/Dockerfile
FROM python:3.11-slim

# 시스템 의존성 설치
RUN apt-get update && apt-get install -y \
    git \
    curl \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

# 작업 디렉토리 설정
WORKDIR /app

# Python 의존성 설치
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# 애플리케이션 코드 복사
COPY backend/ ./backend/

# 사용자 생성 (보안)
RUN useradd -m -u 1000 apiuser && chown -R apiuser:apiuser /app
USER apiuser

# 헬스체크
HEALTHCHECK --interval=30s --timeout=3s --start-period=40s --retries=3 \
    CMD python -c "import requests; requests.get('http://localhost:8000/health')"

# 서버 실행
CMD ["uvicorn", "backend.main:app", "--host", "0.0.0.0", "--port", "8000", "--reload"]
```

### 2. Claude CLI 이미지
```dockerfile
# docker/claude/Dockerfile
FROM node:20-alpine AS builder

# Claude CLI 설치
RUN npm install -g @anthropic-ai/claude-code

# Python 환경 추가
FROM python:3.11-alpine

# Node.js 복사
COPY --from=builder /usr/local/bin/node /usr/local/bin/
COPY --from=builder /usr/local/lib/node_modules /usr/local/lib/node_modules
RUN ln -s /usr/local/lib/node_modules/@anthropic-ai/claude-code/bin/claude /usr/local/bin/claude

# 필수 도구 설치
RUN apk add --no-cache \
    git \
    bash \
    openssh-client \
    docker-cli \
    curl

# 작업 디렉토리
WORKDIR /workspace

# 비특권 사용자
RUN adduser -D -u 1000 claude
USER claude

# Claude 설정 디렉토리
RUN mkdir -p ~/.config/claude

# 기본 명령
CMD ["claude", "--help"]
```

### 3. 워크스페이스 템플릿 이미지
```dockerfile
# docker/workspace/Dockerfile
FROM ubuntu:22.04

# 개발 도구 설치
RUN apt-get update && apt-get install -y \
    build-essential \
    git \
    curl \
    wget \
    vim \
    python3 \
    python3-pip \
    nodejs \
    npm \
    docker.io \
    && rm -rf /var/lib/apt/lists/*

# 작업 디렉토리
WORKDIR /workspace

# 볼륨 마운트 포인트
VOLUME ["/workspace", "/home/developer"]

# 개발자 사용자 생성
RUN useradd -m -s /bin/bash developer
USER developer

# 환경 변수
ENV WORKSPACE_DIR=/workspace

# 대기 명령
CMD ["tail", "-f", "/dev/null"]
```

## 🚀 Docker Compose 설정

### 1. 개발 환경 (docker-compose.yml)
```yaml
version: '3.8'

services:
  # PostgreSQL 데이터베이스
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: aicli_db
      POSTGRES_USER: ${DB_USER:-aicli}
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "5432:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${DB_USER:-aicli}"]
      interval: 10s
      timeout: 5s
      retries: 5

  # Redis 캐시
  redis:
    image: redis:7-alpine
    command: redis-server --appendonly yes
    volumes:
      - redis_data:/data
    ports:
      - "6379:6379"
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

  # MinIO 객체 스토리지
  minio:
    image: minio/minio:latest
    command: server /data --console-address ":9001"
    environment:
      MINIO_ROOT_USER: ${MINIO_ROOT_USER:-minioadmin}
      MINIO_ROOT_PASSWORD: ${MINIO_ROOT_PASSWORD}
    volumes:
      - minio_data:/data
    ports:
      - "9000:9000"
      - "9001:9001"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:9000/minio/health/live"]
      interval: 30s
      timeout: 20s
      retries: 3

  # API 서버
  api:
    build:
      context: .
      dockerfile: docker/api/Dockerfile
    environment:
      DATABASE_URL: postgresql://${DB_USER}:${DB_PASSWORD}@postgres:5432/aicli_db
      REDIS_URL: redis://redis:6379
      MINIO_ENDPOINT: minio:9000
      MINIO_ACCESS_KEY: ${MINIO_ROOT_USER:-minioadmin}
      MINIO_SECRET_KEY: ${MINIO_ROOT_PASSWORD}
      CLAUDE_CODE_OAUTH_TOKEN: ${CLAUDE_CODE_OAUTH_TOKEN}
      SUPABASE_URL: ${SUPABASE_URL}
      SUPABASE_ANON_KEY: ${SUPABASE_ANON_KEY}
    volumes:
      - ./backend:/app/backend
      - ./workspaces:/workspaces
      - /var/run/docker.sock:/var/run/docker.sock
    ports:
      - "8000:8000"
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
      minio:
        condition: service_healthy
    restart: unless-stopped

  # Nginx 리버스 프록시
  nginx:
    build:
      context: .
      dockerfile: docker/nginx/Dockerfile
    volumes:
      - ./docker/nginx/nginx.conf:/etc/nginx/nginx.conf:ro
      - ./frontend/dist:/usr/share/nginx/html:ro
    ports:
      - "80:80"
      - "443:443"
    depends_on:
      - api
    restart: unless-stopped

  # Celery 워커
  celery:
    build:
      context: .
      dockerfile: docker/api/Dockerfile
    command: celery -A backend.tasks worker --loglevel=info
    environment:
      DATABASE_URL: postgresql://${DB_USER}:${DB_PASSWORD}@postgres:5432/aicli_db
      REDIS_URL: redis://redis:6379
      CLAUDE_CODE_OAUTH_TOKEN: ${CLAUDE_CODE_OAUTH_TOKEN}
    volumes:
      - ./backend:/app/backend
      - ./workspaces:/workspaces
      - /var/run/docker.sock:/var/run/docker.sock
    depends_on:
      - postgres
      - redis
    restart: unless-stopped

  # Celery Beat (스케줄러)
  celery-beat:
    build:
      context: .
      dockerfile: docker/api/Dockerfile
    command: celery -A backend.tasks beat --loglevel=info
    environment:
      DATABASE_URL: postgresql://${DB_USER}:${DB_PASSWORD}@postgres:5432/aicli_db
      REDIS_URL: redis://redis:6379
    volumes:
      - ./backend:/app/backend
    depends_on:
      - postgres
      - redis
    restart: unless-stopped

# 볼륨 정의
volumes:
  postgres_data:
  redis_data:
  minio_data:

# 네트워크 정의
networks:
  default:
    name: aicli_network
    driver: bridge
```

### 2. 프로덕션 환경 (docker-compose.prod.yml)
```yaml
version: '3.8'

services:
  api:
    image: ${DOCKER_REGISTRY}/aicli-api:${VERSION:-latest}
    environment:
      - ENVIRONMENT=production
      - LOG_LEVEL=info
    deploy:
      replicas: 3
      resources:
        limits:
          cpus: '2'
          memory: 2G
        reservations:
          cpus: '1'
          memory: 1G
      restart_policy:
        condition: on-failure
        delay: 5s
        max_attempts: 3

  nginx:
    image: ${DOCKER_REGISTRY}/aicli-nginx:${VERSION:-latest}
    volumes:
      - ./ssl:/etc/nginx/ssl:ro
    deploy:
      replicas: 2
      placement:
        constraints:
          - node.role == manager

  # 프로덕션 전용 모니터링
  prometheus:
    image: prom/prometheus:latest
    volumes:
      - ./monitoring/prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus_data:/prometheus
    ports:
      - "9090:9090"

  grafana:
    image: grafana/grafana:latest
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=${GRAFANA_PASSWORD}
    volumes:
      - grafana_data:/var/lib/grafana
    ports:
      - "3000:3000"
    depends_on:
      - prometheus

volumes:
  prometheus_data:
  grafana_data:
```

## 🔐 환경 변수 설정

### .env.example
```bash
# 데이터베이스
DB_USER=aicli
DB_PASSWORD=your_secure_password_here

# Redis
REDIS_PASSWORD=your_redis_password_here

# MinIO
MINIO_ROOT_USER=minioadmin
MINIO_ROOT_PASSWORD=your_minio_password_here

# Claude
CLAUDE_CODE_OAUTH_TOKEN=your_claude_oauth_token_here

# Supabase Auth
SUPABASE_URL=https://your-project.supabase.co
SUPABASE_ANON_KEY=your_supabase_anon_key_here
SUPABASE_SERVICE_KEY=your_supabase_service_key_here

# Docker Registry (프로덕션)
DOCKER_REGISTRY=your-registry.com
VERSION=1.0.0

# 모니터링
GRAFANA_PASSWORD=your_grafana_password_here
```

## 🚦 워크스페이스 컨테이너 관리

### 1. 동적 컨테이너 생성
```python
# backend/services/container_manager.py
import docker
from typing import Dict, Any

class ContainerManager:
    def __init__(self):
        self.client = docker.from_env()
        self.network_name = "aicli_network"
    
    async def create_workspace_container(
        self, 
        workspace_id: str,
        project_path: str
    ) -> str:
        """워크스페이스용 컨테이너 생성"""
        container_name = f"workspace_{workspace_id}"
        
        # 기존 컨테이너 확인
        try:
            existing = self.client.containers.get(container_name)
            existing.remove(force=True)
        except docker.errors.NotFound:
            pass
        
        # 새 컨테이너 생성
        container = self.client.containers.run(
            image="aicli/workspace:latest",
            name=container_name,
            detach=True,
            remove=False,
            volumes={
                project_path: {
                    'bind': '/workspace',
                    'mode': 'rw'
                },
                '/var/run/docker.sock': {
                    'bind': '/var/run/docker.sock',
                    'mode': 'ro'
                }
            },
            environment={
                'WORKSPACE_ID': workspace_id,
                'PROJECT_PATH': project_path
            },
            network=self.network_name,
            mem_limit='2g',
            cpu_quota=100000,  # 1 CPU
            labels={
                'workspace_id': workspace_id,
                'managed_by': 'aicli'
            }
        )
        
        return container.id
    
    async def execute_in_container(
        self,
        container_id: str,
        command: List[str]
    ) -> Dict[str, Any]:
        """컨테이너 내에서 명령 실행"""
        container = self.client.containers.get(container_id)
        
        # 명령 실행
        exec_result = container.exec_run(
            cmd=command,
            stream=True,
            demux=True,
            workdir='/workspace'
        )
        
        # 스트림 처리
        for stdout, stderr in exec_result.output:
            if stdout:
                yield {'type': 'stdout', 'data': stdout.decode()}
            if stderr:
                yield {'type': 'stderr', 'data': stderr.decode()}
```

### 2. 컨테이너 리소스 모니터링
```python
async def monitor_container_stats(self, container_id: str):
    """컨테이너 리소스 사용량 모니터링"""
    container = self.client.containers.get(container_id)
    
    # 실시간 통계 스트림
    for stats in container.stats(stream=True, decode=True):
        cpu_delta = stats['cpu_stats']['cpu_usage']['total_usage'] - \
                   stats['precpu_stats']['cpu_usage']['total_usage']
        system_delta = stats['cpu_stats']['system_cpu_usage'] - \
                      stats['precpu_stats']['system_cpu_usage']
        
        cpu_percent = (cpu_delta / system_delta) * 100.0
        
        memory_usage = stats['memory_stats']['usage']
        memory_limit = stats['memory_stats']['limit']
        memory_percent = (memory_usage / memory_limit) * 100.0
        
        yield {
            'cpu_percent': round(cpu_percent, 2),
            'memory_mb': round(memory_usage / 1024 / 1024, 2),
            'memory_percent': round(memory_percent, 2),
            'network_rx_mb': round(stats['networks']['eth0']['rx_bytes'] / 1024 / 1024, 2),
            'network_tx_mb': round(stats['networks']['eth0']['tx_bytes'] / 1024 / 1024, 2)
        }
```

## 🛡️ 보안 설정

### 1. Docker 소켓 보안
```yaml
# docker-compose.security.yml
services:
  docker-proxy:
    image: tecnativa/docker-socket-proxy
    environment:
      CONTAINERS: 1
      IMAGES: 1
      NETWORKS: 1
      VOLUMES: 1
      POST: 1
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
    networks:
      - docker-proxy
    restart: unless-stopped

  api:
    environment:
      DOCKER_HOST: tcp://docker-proxy:2375
    depends_on:
      - docker-proxy
    networks:
      - default
      - docker-proxy

networks:
  docker-proxy:
    internal: true
```

### 2. 컨테이너 보안 정책
```python
# 보안 옵션 설정
security_opt = [
    'no-new-privileges:true',
    'apparmor:docker-default',
    'seccomp:default'
]

# 읽기 전용 루트 파일시스템
read_only_root_filesystem = True

# 캡 드롭
cap_drop = ['ALL']
cap_add = ['CHOWN', 'SETUID', 'SETGID']
```

## 🔧 유용한 Docker 명령어

```bash
# 전체 스택 시작
docker-compose up -d

# 로그 확인
docker-compose logs -f api

# 특정 서비스 재시작
docker-compose restart api

# 컨테이너 쉘 접속
docker-compose exec api bash

# 빌드 및 시작
docker-compose up -d --build

# 정리
docker-compose down -v

# 프로덕션 배포
docker-compose -f docker-compose.yml -f docker-compose.prod.yml up -d
```

## 🚨 문제 해결

### 일반적인 문제들

1. **권한 문제**
   ```bash
   # Docker 소켓 권한
   sudo usermod -aG docker $USER
   ```

2. **포트 충돌**
   ```bash
   # 사용 중인 포트 확인
   sudo lsof -i :8000
   ```

3. **볼륨 권한**
   ```bash
   # 볼륨 소유권 변경
   sudo chown -R 1000:1000 ./workspaces
   ```

4. **메모리 부족**
   ```bash
   # Docker 메모리 증가
   docker system prune -a
   ```