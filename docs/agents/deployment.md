# Multi-Agent Platform 배포 가이드

AICode Manager Multi-Agent Platform을 다양한 환경에 배포하는 방법을 안내합니다.

## 시스템 요구사항

### 최소 요구사항
- **CPU**: 4 cores
- **메모리**: 8GB RAM
- **디스크**: 50GB SSD
- **OS**: Ubuntu 20.04+ / CentOS 8+ / macOS 11+
- **Docker**: 20.10+
- **Git**: 2.30+

### 권장 요구사항 (100개 에이전트)
- **CPU**: 16 cores
- **메모리**: 32GB RAM  
- **디스크**: 200GB SSD
- **네트워크**: 1Gbps

## 개발 환경 배포

### 1. 저장소 클론

```bash
git clone https://github.com/aicli/aicli-web.git
cd aicli-web
```

### 2. 환경 설정

```bash
# 환경 변수 파일 생성
cp .env.example .env

# 필수 환경 변수 설정
cat > .env << EOF
# 기본 설정
PORT=8080
WEB_PORT=3000
ENV=development

# 데이터베이스
DATABASE_URL=sqlite:///data/aicli.db

# Claude API
CLAUDE_API_KEY=your_claude_api_key_here

# JWT 설정
JWT_SECRET=your_jwt_secret_here

# Docker 설정
DOCKER_HOST=unix:///var/run/docker.sock

# Git 설정
GIT_WORKSPACES_PATH=/tmp/workspaces
EOF
```

### 3. Docker Compose 실행

```bash
# 서비스 시작
docker-compose up -d

# 로그 확인
docker-compose logs -f
```

### 4. 초기 데이터 설정

```bash
# 데이터베이스 마이그레이션
make migrate

# 관리자 계정 생성
make create-admin
```

### 5. 확인

```bash
# API 서버 확인
curl http://localhost:8080/health

# 웹 인터페이스 접속
open http://localhost:3000
```

## 프로덕션 배포

### Docker 기반 배포

#### 1. 프로덕션 이미지 빌드

```bash
# API 서버 이미지 빌드
docker build -t aicli-api:prod -f docker/Dockerfile.api .

# 웹 서버 이미지 빌드
docker build -t aicli-web:prod -f docker/Dockerfile.web .
```

#### 2. Docker Compose 프로덕션 설정

```yaml
# docker-compose.prod.yml
version: '3.8'

services:
  api:
    image: aicli-api:prod
    ports:
      - "8080:8080"
    environment:
      - ENV=production
      - DATABASE_URL=postgresql://user:pass@db:5432/aicli
      - REDIS_URL=redis://redis:6379
      - CLAUDE_API_KEY=${CLAUDE_API_KEY}
      - JWT_SECRET=${JWT_SECRET}
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ./data:/data
      - ./workspaces:/workspaces
    depends_on:
      - db
      - redis
    restart: unless-stopped
    
  web:
    image: aicli-web:prod
    ports:
      - "3000:3000"
    environment:
      - API_URL=http://api:8080
    depends_on:
      - api
    restart: unless-stopped
    
  db:
    image: postgres:15
    environment:
      - POSTGRES_DB=aicli
      - POSTGRES_USER=aicli
      - POSTGRES_PASSWORD=${DB_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data
    restart: unless-stopped
    
  redis:
    image: redis:7-alpine
    volumes:
      - redis_data:/data
    restart: unless-stopped
    
  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf
      - ./ssl:/etc/nginx/ssl
    depends_on:
      - web
      - api
    restart: unless-stopped

volumes:
  postgres_data:
  redis_data:
```

#### 3. Nginx 설정

```nginx
# nginx.conf
events {
    worker_connections 1024;
}

http {
    upstream api {
        server api:8080;
    }
    
    upstream web {
        server web:3000;
    }
    
    # API 서버 프록시
    server {
        listen 80;
        server_name api.yourdomain.com;
        
        location / {
            proxy_pass http://api;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
        }
        
        # WebSocket 지원
        location /api/v1/agents/*/logs/stream {
            proxy_pass http://api;
            proxy_http_version 1.1;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection "upgrade";
            proxy_set_header Host $host;
            proxy_cache_bypass $http_upgrade;
        }
    }
    
    # 웹 인터페이스 프록시
    server {
        listen 80;
        server_name yourdomain.com;
        
        location / {
            proxy_pass http://web;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
        }
    }
}
```

### Kubernetes 배포

#### 1. Namespace 생성

```yaml
# namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: aicli-system
```

#### 2. ConfigMap 설정

```yaml
# configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: aicli-config
  namespace: aicli-system
data:
  DATABASE_URL: "postgresql://aicli:password@postgres:5432/aicli"
  REDIS_URL: "redis://redis:6379"
  GIT_WORKSPACES_PATH: "/workspaces"
```

#### 3. Secret 설정

```yaml
# secret.yaml
apiVersion: v1
kind: Secret
metadata:
  name: aicli-secrets
  namespace: aicli-system
type: Opaque
data:
  claude-api-key: <base64-encoded-key>
  jwt-secret: <base64-encoded-secret>
  db-password: <base64-encoded-password>
```

#### 4. API 서버 배포

```yaml
# api-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: aicli-api
  namespace: aicli-system
spec:
  replicas: 3
  selector:
    matchLabels:
      app: aicli-api
  template:
    metadata:
      labels:
        app: aicli-api
    spec:
      containers:
      - name: api
        image: aicli-api:prod
        ports:
        - containerPort: 8080
        env:
        - name: ENV
          value: "production"
        - name: DATABASE_URL
          valueFrom:
            configMapKeyRef:
              name: aicli-config
              key: DATABASE_URL
        - name: CLAUDE_API_KEY
          valueFrom:
            secretKeyRef:
              name: aicli-secrets
              key: claude-api-key
        - name: JWT_SECRET
          valueFrom:
            secretKeyRef:
              name: aicli-secrets
              key: jwt-secret
        volumeMounts:
        - name: docker-sock
          mountPath: /var/run/docker.sock
        - name: workspaces
          mountPath: /workspaces
        resources:
          requests:
            memory: "512Mi"
            cpu: "500m"
          limits:
            memory: "2Gi"
            cpu: "2000m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
      volumes:
      - name: docker-sock
        hostPath:
          path: /var/run/docker.sock
      - name: workspaces
        persistentVolumeClaim:
          claimName: workspaces-pvc
```

#### 5. 서비스 설정

```yaml
# service.yaml
apiVersion: v1
kind: Service
metadata:
  name: aicli-api-service
  namespace: aicli-system
spec:
  selector:
    app: aicli-api
  ports:
  - port: 80
    targetPort: 8080
  type: LoadBalancer
```

#### 6. Ingress 설정

```yaml
# ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: aicli-ingress
  namespace: aicli-system
  annotations:
    nginx.ingress.kubernetes.io/websocket-services: "aicli-api-service"
    nginx.ingress.kubernetes.io/proxy-read-timeout: "3600"
    nginx.ingress.kubernetes.io/proxy-send-timeout: "3600"
spec:
  tls:
  - hosts:
    - api.yourdomain.com
    - yourdomain.com
    secretName: tls-secret
  rules:
  - host: api.yourdomain.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: aicli-api-service
            port:
              number: 80
  - host: yourdomain.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: aicli-web-service
            port:
              number: 80
```

## 환경별 설정

### 개발 환경
```bash
# .env.development
ENV=development
LOG_LEVEL=debug
DATABASE_URL=sqlite:///data/aicli.db
ENABLE_SWAGGER=true
CORS_ORIGINS=http://localhost:3000
```

### 스테이징 환경
```bash
# .env.staging
ENV=staging
LOG_LEVEL=info
DATABASE_URL=postgresql://aicli:password@staging-db:5432/aicli
ENABLE_SWAGGER=true
CORS_ORIGINS=https://staging.yourdomain.com
```

### 프로덕션 환경
```bash
# .env.production
ENV=production
LOG_LEVEL=warn
DATABASE_URL=postgresql://aicli:password@prod-db:5432/aicli
ENABLE_SWAGGER=false
CORS_ORIGINS=https://yourdomain.com
```

## 모니터링 설정

### Prometheus 메트릭

```yaml
# prometheus.yml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'aicli-api'
    static_configs:
      - targets: ['api:8080']
    metrics_path: /metrics
    scrape_interval: 10s
```

### Grafana 대시보드

```json
{
  "dashboard": {
    "title": "AICode Manager Agents",
    "panels": [
      {
        "title": "Active Agents",
        "type": "stat",
        "targets": [
          {
            "expr": "aicli_active_agents_total"
          }
        ]
      },
      {
        "title": "Agent Creation Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(aicli_agents_created_total[5m])"
          }
        ]
      }
    ]
  }
}
```

## 보안 설정

### SSL/TLS 인증서

```bash
# Let's Encrypt 인증서 생성
certbot certonly --webroot \
  -w /var/www/html \
  -d yourdomain.com \
  -d api.yourdomain.com
```

### 방화벽 설정

```bash
# UFW 방화벽 설정
ufw allow 22/tcp    # SSH
ufw allow 80/tcp    # HTTP
ufw allow 443/tcp   # HTTPS
ufw enable
```

### Docker 보안

```bash
# Docker 데몬 보안 설정
cat > /etc/docker/daemon.json << EOF
{
  "live-restore": true,
  "userland-proxy": false,
  "no-new-privileges": true,
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "10m",
    "max-file": "3"
  }
}
EOF

systemctl restart docker
```

## 백업 및 복구

### 데이터베이스 백업

```bash
#!/bin/bash
# backup-db.sh

BACKUP_DIR="/backups"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="$BACKUP_DIR/aicli_$TIMESTAMP.sql"

# PostgreSQL 백업
pg_dump $DATABASE_URL > $BACKUP_FILE

# 압축
gzip $BACKUP_FILE

# 오래된 백업 정리 (30일 이상)
find $BACKUP_DIR -name "*.sql.gz" -mtime +30 -delete
```

### Git Worktrees 백업

```bash
#!/bin/bash
# backup-workspaces.sh

WORKSPACE_DIR="/workspaces"
BACKUP_DIR="/backups/workspaces"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

tar -czf "$BACKUP_DIR/workspaces_$TIMESTAMP.tar.gz" -C "$WORKSPACE_DIR" .
```

## 성능 튜닝

### 시스템 설정

```bash
# /etc/sysctl.conf
net.core.somaxconn = 1024
net.ipv4.tcp_max_syn_backlog = 2048
vm.max_map_count = 262144
fs.file-max = 65536
```

### Docker 최적화

```bash
# Docker 컨테이너 리소스 제한
docker run -d \
  --memory="2g" \
  --cpus="1.0" \
  --restart=unless-stopped \
  aicli-api:prod
```

### 데이터베이스 튜닝

```sql
-- PostgreSQL 설정
ALTER SYSTEM SET shared_buffers = '256MB';
ALTER SYSTEM SET effective_cache_size = '1GB';
ALTER SYSTEM SET random_page_cost = 1.1;
SELECT pg_reload_conf();
```

## 문제 해결

### 일반적인 문제

**Docker 권한 오류**:
```bash
# Docker 그룹에 사용자 추가
sudo usermod -aG docker $USER
newgrp docker
```

**메모리 부족**:
```bash
# 스왑 메모리 추가
sudo fallocate -l 2G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile
```

**포트 충돌**:
```bash
# 포트 사용 확인
sudo netstat -tulpn | grep :8080
sudo fuser -k 8080/tcp
```

### 로그 확인

```bash
# Docker Compose 로그
docker-compose logs -f api

# Kubernetes 로그
kubectl logs -f deployment/aicli-api -n aicli-system

# 시스템 로그
journalctl -u docker -f
```

## 운영 체크리스트

### 배포 전 확인사항
- [ ] 모든 환경 변수 설정 완료
- [ ] SSL 인증서 설정 완료
- [ ] 데이터베이스 연결 테스트 통과
- [ ] Docker 이미지 보안 스캔 통과
- [ ] 백업 시스템 구성 완료

### 배포 후 확인사항
- [ ] API 헬스체크 통과
- [ ] 웹 인터페이스 접근 가능
- [ ] 에이전트 생성/시작/중지 테스트 통과
- [ ] WebSocket 연결 테스트 통과
- [ ] 모니터링 대시보드 정상 작동

### 정기 점검사항
- [ ] 디스크 사용량 확인
- [ ] 메모리 사용량 확인
- [ ] 에러 로그 검토
- [ ] 백업 파일 확인
- [ ] 보안 업데이트 적용

이 가이드를 따라 배포하시면 안정적이고 확장 가능한 Multi-Agent Platform을 운영할 수 있습니다.