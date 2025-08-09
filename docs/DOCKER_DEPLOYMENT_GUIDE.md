# 🐳 Docker 기반 AICLI-Web 실행 가이드

> **Claude Code를 이용한 개발자를 위한 개인 서버 Docker 배포 가이드**

## 📋 목차

1. [소개](#-소개)
2. [사전 요구사항](#-사전-요구사항)
3. [빠른 시작](#-빠른-시작)
4. [상세 설정](#-상세-설정)
5. [운영 가이드](#-운영-가이드)
6. [트러블슈팅](#-트러블슈팅)
7. [고급 설정](#-고급-설정)

---

## 🎯 소개

AICLI-Web은 Claude CLI를 웹 환경에서 관리할 수 있는 도구입니다. 이 가이드는 개인 서버에서 Docker를 사용하여 AICLI-Web을 실행하는 방법을 설명합니다.

### 주요 특징
- 🔒 **격리된 환경**: 각 Claude 인스턴스가 독립된 Docker 컨테이너에서 실행
- 🚀 **병렬 처리**: 여러 프로젝트 동시 작업 가능
- 📊 **실시간 모니터링**: WebSocket을 통한 실시간 로그 스트리밍
- 🔧 **자동 재시작**: Air를 통한 핫 리로드 지원

---

## 📦 사전 요구사항

### 필수 소프트웨어

```bash
# Docker 및 Docker Compose 버전 확인
docker --version    # Docker 20.10+ 필요
docker-compose --version  # Docker Compose 2.0+ 필요

# Git (코드 다운로드용)
git --version

# Make (빌드 자동화)
make --version
```

### 시스템 요구사항

- **OS**: Linux (Ubuntu 20.04+, CentOS 8+) 또는 macOS
- **CPU**: 2 코어 이상
- **RAM**: 4GB 이상 (권장 8GB)
- **Storage**: 10GB 이상 여유 공간
- **Network**: 포트 8080 (API), 3000 (웹 UI) 사용 가능

---

## 🚀 빠른 시작

### 1단계: 코드 클론

```bash
# 프로젝트 클론
git clone https://github.com/your-org/aicli-web.git
cd aicli-web

# 브랜치 확인
git branch -a
```

### 2단계: 환경 변수 설정

```bash
# .env 파일 생성
cat > .env << 'EOF'
# Claude API 설정
CLAUDE_API_KEY=your_claude_api_key_here

# 서버 설정
API_PORT=8080
WEB_PORT=3000
NODE_ENV=production

# Docker 설정
DOCKER_NETWORK=aicli-network
DOCKER_VOLUME_PREFIX=aicli

# 보안 설정
JWT_SECRET=$(openssl rand -hex 32)
SESSION_SECRET=$(openssl rand -hex 32)

# 로깅
LOG_LEVEL=info
LOG_FORMAT=json
EOF

# 권한 설정
chmod 600 .env
```

### 3단계: Docker 이미지 빌드 및 실행

```bash
# 모든 서비스 빌드 및 시작
docker-compose up -d --build

# 로그 확인
docker-compose logs -f

# 서비스 상태 확인
docker-compose ps
```

### 4단계: 접속 확인

```bash
# API 헬스체크
curl http://localhost:8080/health

# 웹 UI 접속
open http://localhost:3000
```

---

## ⚙️ 상세 설정

### Docker Compose 구성

```yaml
# docker-compose.prod.yml
version: '3.8'

services:
  # API 서버
  api:
    build:
      context: .
      dockerfile: Dockerfile.prod
      target: api
    container_name: aicli-api
    restart: unless-stopped
    ports:
      - "${API_PORT:-8080}:8080"
    environment:
      - GO_ENV=production
      - CLAUDE_API_KEY=${CLAUDE_API_KEY}
      - JWT_SECRET=${JWT_SECRET}
      - LOG_LEVEL=${LOG_LEVEL:-info}
    volumes:
      - ./data:/data
      - /var/run/docker.sock:/var/run/docker.sock:ro
    networks:
      - aicli-network
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 10s

  # 웹 프론트엔드
  web:
    build:
      context: ./web
      dockerfile: Dockerfile
    container_name: aicli-web
    restart: unless-stopped
    ports:
      - "${WEB_PORT:-3000}:3000"
    environment:
      - NODE_ENV=production
      - API_URL=http://api:8080
    depends_on:
      - api
    networks:
      - aicli-network

  # Claude 워크스페이스 (격리된 실행 환경)
  workspace:
    image: aicli/workspace:latest
    container_name: aicli-workspace
    restart: unless-stopped
    volumes:
      - workspace-data:/workspace
      - claude-cache:/home/claude/.cache
    environment:
      - CLAUDE_API_KEY=${CLAUDE_API_KEY}
    networks:
      - aicli-network
    security_opt:
      - no-new-privileges:true
      - seccomp:unconfined
    cap_drop:
      - ALL
    cap_add:
      - SYS_PTRACE  # PTY 지원

networks:
  aicli-network:
    driver: bridge
    ipam:
      config:
        - subnet: 172.20.0.0/16

volumes:
  workspace-data:
  claude-cache:
```

### Production Dockerfile

```dockerfile
# Dockerfile.prod
# 멀티스테이지 빌드로 최적화

# 빌드 스테이지
FROM golang:1.21-alpine AS builder
RUN apk add --no-cache git make
WORKDIR /build
COPY go.* ./
RUN go mod download
COPY . .
RUN make build

# API 서버 스테이지
FROM alpine:3.18 AS api
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /build/build/aicli-api /app/
EXPOSE 8080
CMD ["./aicli-api", "serve"]

# CLI 도구 스테이지
FROM alpine:3.18 AS cli
RUN apk add --no-cache ca-certificates tzdata bash
WORKDIR /app
COPY --from=builder /build/build/aicli /app/
ENTRYPOINT ["./aicli"]
```

---

## 📊 운영 가이드

### 서비스 관리

```bash
# 서비스 시작
docker-compose -f docker-compose.prod.yml up -d

# 서비스 중지
docker-compose -f docker-compose.prod.yml down

# 서비스 재시작
docker-compose -f docker-compose.prod.yml restart

# 특정 서비스만 재시작
docker-compose -f docker-compose.prod.yml restart api

# 로그 모니터링
docker-compose -f docker-compose.prod.yml logs -f --tail=100

# 리소스 사용량 확인
docker stats
```

### 백업 및 복원

```bash
# 데이터 백업
#!/bin/bash
BACKUP_DIR="/backup/aicli/$(date +%Y%m%d_%H%M%S)"
mkdir -p $BACKUP_DIR

# 볼륨 백업
docker run --rm \
  -v aicli_workspace-data:/data \
  -v $BACKUP_DIR:/backup \
  alpine tar czf /backup/workspace-data.tar.gz -C /data .

# 환경 설정 백업
cp .env $BACKUP_DIR/
cp docker-compose.prod.yml $BACKUP_DIR/

echo "백업 완료: $BACKUP_DIR"
```

### 모니터링

```bash
# Prometheus 메트릭 엔드포인트 활성화
docker-compose exec api curl http://localhost:8080/metrics

# 헬스체크 스크립트
cat > healthcheck.sh << 'EOF'
#!/bin/bash
API_URL="http://localhost:8080/health"
WEB_URL="http://localhost:3000"

check_service() {
    local url=$1
    local name=$2
    
    if curl -f -s $url > /dev/null; then
        echo "✅ $name is healthy"
        return 0
    else
        echo "❌ $name is unhealthy"
        return 1
    fi
}

check_service $API_URL "API Server"
check_service $WEB_URL "Web UI"

# Docker 컨테이너 상태 확인
docker-compose ps
EOF

chmod +x healthcheck.sh
./healthcheck.sh
```

### 로그 관리

```bash
# 로그 로테이션 설정
cat > /etc/logrotate.d/aicli << 'EOF'
/var/lib/docker/containers/*/*.log {
    daily
    rotate 7
    compress
    delaycompress
    missingok
    notifempty
    maxsize 100M
}
EOF

# 중앙화된 로깅 (선택사항)
docker-compose -f docker-compose.logging.yml up -d
```

---

## 🔧 트러블슈팅

### 일반적인 문제 해결

#### 1. Docker 권한 문제

```bash
# 현재 사용자를 docker 그룹에 추가
sudo usermod -aG docker $USER
newgrp docker

# Docker 소켓 권한 확인
ls -la /var/run/docker.sock
```

#### 2. 포트 충돌

```bash
# 사용 중인 포트 확인
sudo lsof -i :8080
sudo lsof -i :3000

# 다른 포트로 변경
export API_PORT=8081
export WEB_PORT=3001
docker-compose -f docker-compose.prod.yml up -d
```

#### 3. 메모리 부족

```bash
# Docker 리소스 제한 설정
cat >> docker-compose.prod.yml << 'EOF'
services:
  api:
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 2G
        reservations:
          cpus: '1'
          memory: 1G
EOF
```

#### 4. Claude API 연결 실패

```bash
# API 키 확인
docker-compose exec api env | grep CLAUDE

# 네트워크 연결 테스트
docker-compose exec api ping -c 3 api.anthropic.com

# 프록시 설정 (필요한 경우)
export HTTP_PROXY=http://proxy.company.com:8080
export HTTPS_PROXY=http://proxy.company.com:8080
```

### 디버깅 명령어

```bash
# 컨테이너 내부 접속
docker-compose exec api sh
docker-compose exec web sh

# 프로세스 목록 확인
docker-compose exec api ps aux

# 네트워크 상태 확인
docker network inspect aicli-network

# 볼륨 검사
docker volume inspect aicli_workspace-data

# 컨테이너 로그 상세 보기
docker logs -f --details aicli-api

# 시스템 이벤트 모니터링
docker events --filter container=aicli-api
```

---

## 🚀 고급 설정

### 1. HTTPS 설정 (Traefik 사용)

```yaml
# docker-compose.traefik.yml
version: '3.8'

services:
  traefik:
    image: traefik:v2.10
    container_name: traefik
    command:
      - "--api.insecure=true"
      - "--providers.docker=true"
      - "--entrypoints.web.address=:80"
      - "--entrypoints.websecure.address=:443"
      - "--certificatesresolvers.myresolver.acme.tlschallenge=true"
      - "--certificatesresolvers.myresolver.acme.email=admin@example.com"
      - "--certificatesresolvers.myresolver.acme.storage=/letsencrypt/acme.json"
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - ./letsencrypt:/letsencrypt
    networks:
      - aicli-network

  api:
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.api.rule=Host(`api.aicli.example.com`)"
      - "traefik.http.routers.api.entrypoints=websecure"
      - "traefik.http.routers.api.tls.certresolver=myresolver"
```

### 2. 수평 확장 (Swarm 모드)

```bash
# Docker Swarm 초기화
docker swarm init

# 서비스 배포
docker stack deploy -c docker-compose.swarm.yml aicli

# 스케일링
docker service scale aicli_api=3
docker service scale aicli_workspace=5

# 서비스 상태 확인
docker service ls
docker service ps aicli_api
```

### 3. 사용자 정의 네트워크 보안

```bash
# 방화벽 규칙 설정
sudo iptables -A INPUT -p tcp --dport 8080 -s 192.168.1.0/24 -j ACCEPT
sudo iptables -A INPUT -p tcp --dport 8080 -j DROP

# UFW 사용 시
sudo ufw allow from 192.168.1.0/24 to any port 8080
sudo ufw enable
```

### 4. 성능 최적화

```yaml
# docker-compose.optimized.yml
services:
  api:
    sysctls:
      - net.core.somaxconn=65535
      - net.ipv4.tcp_syncookies=1
    ulimits:
      nofile:
        soft: 65536
        hard: 65536
```

### 5. 자동 업데이트 (Watchtower)

```bash
# Watchtower로 자동 업데이트 설정
docker run -d \
  --name watchtower \
  -v /var/run/docker.sock:/var/run/docker.sock \
  containrrr/watchtower \
  --schedule "0 0 4 * * *" \
  --cleanup \
  aicli-api aicli-web
```

---

## 📚 추가 리소스

### 유용한 스크립트

```bash
# install.sh - 원클릭 설치 스크립트
#!/bin/bash
set -e

echo "🚀 AICLI-Web 설치를 시작합니다..."

# Docker 설치 확인
if ! command -v docker &> /dev/null; then
    echo "Docker를 설치합니다..."
    curl -fsSL https://get.docker.com | sh
fi

# 프로젝트 클론
git clone https://github.com/your-org/aicli-web.git
cd aicli-web

# 환경 설정
read -p "Claude API 키를 입력하세요: " api_key
echo "CLAUDE_API_KEY=$api_key" > .env

# 빌드 및 실행
docker-compose -f docker-compose.prod.yml up -d --build

echo "✅ 설치 완료! http://localhost:3000 으로 접속하세요."
```

### 모니터링 대시보드

```yaml
# docker-compose.monitoring.yml
version: '3.8'

services:
  grafana:
    image: grafana/grafana:latest
    container_name: grafana
    ports:
      - "3001:3000"
    volumes:
      - grafana-data:/var/lib/grafana
      - ./grafana/dashboards:/etc/grafana/provisioning/dashboards
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
      - GF_INSTALL_PLUGINS=redis-datasource
    networks:
      - aicli-network

  prometheus:
    image: prom/prometheus:latest
    container_name: prometheus
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus/prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus-data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
    networks:
      - aicli-network

volumes:
  grafana-data:
  prometheus-data:
```

---

## 📞 지원 및 문의

- **문서**: [공식 문서](https://docs.aicli.io)
- **이슈 트래커**: [GitHub Issues](https://github.com/your-org/aicli-web/issues)
- **커뮤니티**: [Discord](https://discord.gg/aicli)
- **이메일**: support@aicli.io

---

## 📄 라이선스

MIT License - 자유롭게 사용하고 수정할 수 있습니다.

---

*마지막 업데이트: 2025-08-09*
*버전: 1.0.0*