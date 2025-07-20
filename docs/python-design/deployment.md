# 배포 가이드

## 🚀 개요

이 문서는 AICode Manager를 개발 환경부터 프로덕션 환경까지 배포하는 방법을 설명합니다.

## 🛠️ 개발 환경 설정

### 1. 사전 요구사항

```bash
# 시스템 요구사항
- Ubuntu 22.04 LTS 이상 (또는 macOS)
- Docker 24.0+ 및 Docker Compose 2.20+
- Python 3.11+
- Node.js 20+ (Claude CLI용)
- Git 2.40+
- 최소 8GB RAM, 20GB 디스크 공간
```

### 2. 프로젝트 클론 및 설정

```bash
# 저장소 클론
git clone https://github.com/yourusername/aicli-web.git
cd aicli-web

# Python 가상환경 설정
python -m venv venv
source venv/bin/activate  # Windows: venv\Scripts\activate

# 의존성 설치
pip install -r requirements-dev.txt

# Node.js 의존성 설치
npm install -g @anthropic-ai/claude-code

# 환경 변수 설정
cp .env.example .env
# .env 파일을 편집하여 필요한 값 입력
```

### 3. 개발 서버 실행

```bash
# Docker 서비스 시작
docker-compose up -d postgres redis minio

# FastAPI 개발 서버 실행
uvicorn backend.main:app --reload --host 0.0.0.0 --port 8000

# 프론트엔드 개발 서버 (별도 터미널)
cd frontend
npm install
npm run dev
```

### 4. 개발 도구 설정

```bash
# Pre-commit hooks 설치
pre-commit install

# 코드 포맷팅
black backend/
isort backend/
flake8 backend/

# 타입 체크
mypy backend/

# 테스트 실행
pytest backend/tests/ -v --cov=backend
```

## 🏭 프로덕션 배포

### 1. 서버 준비

#### Ubuntu 서버 초기 설정
```bash
# 시스템 업데이트
sudo apt update && sudo apt upgrade -y

# 필수 패키지 설치
sudo apt install -y \
    apt-transport-https \
    ca-certificates \
    curl \
    gnupg \
    lsb-release \
    ufw \
    fail2ban

# Docker 설치
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
sudo usermod -aG docker $USER

# Docker Compose 설치
sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose
```

#### 방화벽 설정
```bash
# UFW 방화벽 설정
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow ssh
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw enable

# Fail2ban 설정
sudo cp /etc/fail2ban/jail.conf /etc/fail2ban/jail.local
sudo systemctl enable fail2ban
sudo systemctl start fail2ban
```

### 2. SSL 인증서 설정

#### Let's Encrypt 인증서
```bash
# Certbot 설치
sudo apt install certbot python3-certbot-nginx

# 인증서 발급
sudo certbot certonly --standalone -d your-domain.com

# 자동 갱신 설정
sudo systemctl enable certbot.timer
sudo systemctl start certbot.timer
```

#### Nginx SSL 설정
```nginx
# docker/nginx/nginx.prod.conf
server {
    listen 80;
    server_name your-domain.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name your-domain.com;

    ssl_certificate /etc/letsencrypt/live/your-domain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/your-domain.com/privkey.pem;
    
    # SSL 보안 설정
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 10m;
    
    # HSTS
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    
    location / {
        proxy_pass http://api:8000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
    
    location /ws {
        proxy_pass http://api:8000/ws;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

### 3. 프로덕션 배포 스크립트

```bash
#!/bin/bash
# deploy.sh

set -e

echo "🚀 Starting deployment..."

# 환경 변수 확인
if [ ! -f .env.prod ]; then
    echo "❌ .env.prod file not found!"
    exit 1
fi

# Git 최신 버전 가져오기
echo "📥 Pulling latest code..."
git pull origin main

# Docker 이미지 빌드
echo "🏗️ Building Docker images..."
docker-compose -f docker-compose.yml -f docker-compose.prod.yml build

# 데이터베이스 마이그레이션
echo "🔄 Running database migrations..."
docker-compose -f docker-compose.yml -f docker-compose.prod.yml run --rm api alembic upgrade head

# 이전 컨테이너 중지 및 새 컨테이너 시작
echo "🔄 Restarting services..."
docker-compose -f docker-compose.yml -f docker-compose.prod.yml up -d

# 헬스 체크
echo "🏥 Health check..."
sleep 10
curl -f http://localhost/api/health || exit 1

# 로그 확인
echo "📋 Checking logs..."
docker-compose logs --tail=50

echo "✅ Deployment completed successfully!"
```

### 4. 환경 변수 관리

#### .env.prod 예제
```bash
# 애플리케이션
ENVIRONMENT=production
DEBUG=false
SECRET_KEY=your-very-secure-secret-key
ALLOWED_HOSTS=your-domain.com,www.your-domain.com

# 데이터베이스
DB_HOST=postgres
DB_PORT=5432
DB_NAME=aicli_prod
DB_USER=aicli_prod_user
DB_PASSWORD=very-secure-password

# Redis
REDIS_URL=redis://redis:6379/0

# Claude
CLAUDE_CODE_OAUTH_TOKEN=your-production-oauth-token

# Supabase
SUPABASE_URL=https://your-project.supabase.co
SUPABASE_ANON_KEY=your-production-anon-key
SUPABASE_SERVICE_KEY=your-production-service-key

# 모니터링
SENTRY_DSN=https://your-sentry-dsn
LOG_LEVEL=INFO
```

## 🐋 Docker 배포 전략

### 1. 블루-그린 배포

```bash
#!/bin/bash
# blue-green-deploy.sh

# 현재 활성 환경 확인
CURRENT=$(docker ps --format "table {{.Names}}" | grep -E "aicli_api_(blue|green)" | head -1 | awk -F_ '{print $3}')

if [ "$CURRENT" == "blue" ]; then
    NEW="green"
else
    NEW="blue"
fi

echo "Current: $CURRENT, Deploying to: $NEW"

# 새 환경 빌드 및 시작
docker-compose -f docker-compose.$NEW.yml build
docker-compose -f docker-compose.$NEW.yml up -d

# 헬스 체크
sleep 10
if curl -f http://localhost:800$([[ $NEW == "blue" ]] && echo "1" || echo "2")/health; then
    # Nginx 설정 변경
    sudo cp /etc/nginx/sites-available/aicli.$NEW /etc/nginx/sites-enabled/aicli
    sudo nginx -t && sudo nginx -s reload
    
    # 이전 환경 중지
    sleep 5
    docker-compose -f docker-compose.$CURRENT.yml down
    
    echo "✅ Deployment successful!"
else
    echo "❌ Health check failed, rolling back..."
    docker-compose -f docker-compose.$NEW.yml down
    exit 1
fi
```

### 2. 롤링 업데이트

```yaml
# docker-compose.prod.yml
services:
  api:
    deploy:
      replicas: 3
      update_config:
        parallelism: 1
        delay: 10s
        failure_action: rollback
        monitor: 30s
        max_failure_ratio: 0.3
      restart_policy:
        condition: any
        delay: 5s
        max_attempts: 3
        window: 120s
```

## 📊 모니터링 설정

### 1. Prometheus + Grafana

```yaml
# monitoring/docker-compose.yml
version: '3.8'

services:
  prometheus:
    image: prom/prometheus:latest
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus_data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
    ports:
      - "9090:9090"

  grafana:
    image: grafana/grafana:latest
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=${GRAFANA_PASSWORD}
      - GF_INSTALL_PLUGINS=grafana-clock-panel,grafana-simple-json-datasource
    volumes:
      - grafana_data:/var/lib/grafana
      - ./grafana/dashboards:/etc/grafana/provisioning/dashboards
      - ./grafana/datasources:/etc/grafana/provisioning/datasources
    ports:
      - "3000:3000"

  node-exporter:
    image: prom/node-exporter:latest
    ports:
      - "9100:9100"

  cadvisor:
    image: gcr.io/cadvisor/cadvisor:latest
    volumes:
      - /:/rootfs:ro
      - /var/run:/var/run:ro
      - /sys:/sys:ro
      - /var/lib/docker/:/var/lib/docker:ro
    ports:
      - "8080:8080"

volumes:
  prometheus_data:
  grafana_data:
```

### 2. 로그 집계 (ELK Stack)

```yaml
# logging/docker-compose.yml
version: '3.8'

services:
  elasticsearch:
    image: docker.elastic.co/elasticsearch/elasticsearch:8.11.0
    environment:
      - discovery.type=single-node
      - "ES_JAVA_OPTS=-Xms512m -Xmx512m"
    volumes:
      - elasticsearch_data:/usr/share/elasticsearch/data

  logstash:
    image: docker.elastic.co/logstash/logstash:8.11.0
    volumes:
      - ./logstash/pipeline:/usr/share/logstash/pipeline
    depends_on:
      - elasticsearch

  kibana:
    image: docker.elastic.co/kibana/kibana:8.11.0
    environment:
      - ELASTICSEARCH_HOSTS=http://elasticsearch:9200
    ports:
      - "5601:5601"
    depends_on:
      - elasticsearch

volumes:
  elasticsearch_data:
```

## 🔧 유지보수

### 1. 백업 전략

```bash
#!/bin/bash
# backup.sh

BACKUP_DIR="/backups/$(date +%Y%m%d_%H%M%S)"
mkdir -p $BACKUP_DIR

# 데이터베이스 백업
docker-compose exec -T postgres pg_dump -U $DB_USER $DB_NAME | gzip > $BACKUP_DIR/postgres.sql.gz

# Redis 백업
docker-compose exec -T redis redis-cli BGSAVE
docker cp $(docker-compose ps -q redis):/data/dump.rdb $BACKUP_DIR/

# MinIO 백업
docker run --rm -v minio_data:/data -v $BACKUP_DIR:/backup alpine tar czf /backup/minio.tar.gz /data

# 워크스페이스 백업
tar czf $BACKUP_DIR/workspaces.tar.gz /workspaces

# S3 업로드 (선택사항)
aws s3 sync $BACKUP_DIR s3://your-backup-bucket/aicli-backups/

# 오래된 백업 삭제 (30일 이상)
find /backups -type d -mtime +30 -exec rm -rf {} \;
```

### 2. 복구 절차

```bash
#!/bin/bash
# restore.sh

BACKUP_DATE=$1
BACKUP_DIR="/backups/$BACKUP_DATE"

if [ ! -d "$BACKUP_DIR" ]; then
    echo "Backup not found: $BACKUP_DIR"
    exit 1
fi

# 서비스 중지
docker-compose down

# 데이터베이스 복구
docker-compose up -d postgres
sleep 10
gunzip -c $BACKUP_DIR/postgres.sql.gz | docker-compose exec -T postgres psql -U $DB_USER $DB_NAME

# Redis 복구
docker cp $BACKUP_DIR/dump.rdb $(docker-compose ps -q redis):/data/
docker-compose restart redis

# MinIO 복구
docker run --rm -v minio_data:/data -v $BACKUP_DIR:/backup alpine tar xzf /backup/minio.tar.gz -C /

# 워크스페이스 복구
tar xzf $BACKUP_DIR/workspaces.tar.gz -C /

# 서비스 재시작
docker-compose up -d
```

### 3. 성능 튜닝

```python
# backend/config.py
class ProductionConfig:
    # 데이터베이스 연결 풀
    SQLALCHEMY_ENGINE_OPTIONS = {
        'pool_size': 20,
        'pool_recycle': 3600,
        'pool_pre_ping': True,
        'max_overflow': 40
    }
    
    # Redis 연결 풀
    REDIS_POOL_SIZE = 50
    REDIS_DECODE_RESPONSES = True
    
    # 워커 설정
    CELERY_WORKER_CONCURRENCY = 4
    CELERY_WORKER_MAX_TASKS_PER_CHILD = 1000
    
    # API Rate Limiting
    RATELIMIT_STORAGE_URL = "redis://redis:6379/1"
    RATELIMIT_DEFAULT = "100/minute"
```

## 🚨 트러블슈팅

### 일반적인 문제 해결

1. **메모리 부족**
   ```bash
   # 스왑 파일 추가
   sudo fallocate -l 4G /swapfile
   sudo chmod 600 /swapfile
   sudo mkswap /swapfile
   sudo swapon /swapfile
   ```

2. **디스크 공간 부족**
   ```bash
   # Docker 정리
   docker system prune -a --volumes
   
   # 로그 정리
   find /var/log -type f -name "*.log" -mtime +7 -delete
   ```

3. **성능 문제**
   ```bash
   # 프로세스 모니터링
   htop
   
   # Docker 통계
   docker stats
   
   # 네트워크 모니터링
   iftop
   ```

## 📈 확장 가이드

### 수평 확장
1. **로드 밸런서 추가** (HAProxy/Nginx)
2. **데이터베이스 복제** (Master-Slave)
3. **Redis 클러스터** 구성
4. **CDN** 통합 (정적 자산)

### 수직 확장
1. **서버 스펙 업그레이드**
2. **데이터베이스 최적화**
3. **캐싱 전략 개선**
4. **비동기 처리 확대**