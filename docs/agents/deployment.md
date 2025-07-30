# 배포 및 운영 가이드

AICode Manager 멀티 에이전트 플랫폼을 프로덕션 환경에 배포하고 운영하는 방법을 설명합니다.

## 🎯 배포 개요

### 지원되는 배포 방식

1. **단일 서버 배포** - 소규모 팀용 (1-20명)
2. **Docker Compose 배포** - 중규모 팀용 (20-100명)  
3. **Kubernetes 배포** - 대규모 팀용 (100명 이상)
4. **클라우드 관리 서비스** - 완전 관리형 솔루션

### 시스템 요구사항

#### 최소 요구사항
- **CPU**: 4 코어
- **메모리**: 8GB RAM
- **디스크**: 100GB SSD
- **네트워크**: 1Gbps
- **운영체제**: Linux (Ubuntu 20.04+ 권장)

#### 권장 요구사항
- **CPU**: 8 코어 (Intel Xeon 또는 동급)
- **메모리**: 32GB RAM
- **디스크**: 500GB NVMe SSD
- **네트워크**: 10Gbps
- **운영체제**: Ubuntu 22.04 LTS

## 🚀 단일 서버 배포

### 1. 시스템 준비

#### 필수 소프트웨어 설치
```bash
# 시스템 업데이트
sudo apt update && sudo apt upgrade -y

# Docker 설치
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
sudo usermod -aG docker $USER

# Docker Compose 설치
sudo curl -L "https://github.com/docker/compose/releases/download/v2.20.0/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose

# Git 설치 (2.28+ 필요)
sudo apt install git -y
git --version

# 기타 유틸리티
sudo apt install curl wget jq htop -y
```

#### 시스템 최적화
```bash
# 파일 디스크립터 제한 증가
echo "* soft nofile 65536" | sudo tee -a /etc/security/limits.conf
echo "* hard nofile 65536" | sudo tee -a /etc/security/limits.conf

# 커널 파라미터 최적화
cat << EOF | sudo tee -a /etc/sysctl.conf
# AICode Manager 최적화
fs.file-max = 2097152
net.core.somaxconn = 65535
net.core.rmem_max = 16777216
net.core.wmem_max = 16777216
vm.max_map_count = 262144
EOF

sudo sysctl -p
```

### 2. 애플리케이션 배포

#### 릴리즈 다운로드
```bash
# 작업 디렉토리 생성
sudo mkdir -p /opt/aicli-web
sudo chown $USER:$USER /opt/aicli-web
cd /opt/aicli-web

# 최신 릴리즈 다운로드
LATEST_VERSION=$(curl -s https://api.github.com/repos/aicli/aicli-web/releases/latest | jq -r .tag_name)
wget https://github.com/aicli/aicli-web/releases/download/${LATEST_VERSION}/aicli-api-linux-amd64
chmod +x aicli-api-linux-amd64

# 심볼릭 링크 생성
ln -sf aicli-api-linux-amd64 aicli-api
```

#### 설정 파일 생성
```bash
# 설정 디렉토리 생성
mkdir -p config data logs

# 주 설정 파일
cat << EOF > config/config.yaml
server:
  host: "0.0.0.0"
  port: 8080
  read_timeout: 30s
  write_timeout: 30s
  
database:
  type: "sqlite"
  path: "/opt/aicli-web/data/aicli.db"
  
docker:
  socket: "/var/run/docker.sock"
  network: "aicli-network"
  
git:
  workspace_root: "/opt/aicli-web/workspaces"
  cleanup_interval: "10m"
  
performance:
  agent_pool:
    standard:
      min_size: 5
      max_size: 20
    gpu:
      min_size: 2
      max_size: 8
  concurrency:
    max_concurrent_agents: 50
    container_creation_limit: 10
    
logging:
  level: "info"
  format: "json"
  output: "/opt/aicli-web/logs/aicli.log"
  
metrics:
  enabled: true
  port: 9090
  path: "/metrics"
EOF
```

#### 환경 변수 설정
```bash
# 환경 변수 파일
cat << EOF > .env
# API 설정
AICLI_CONFIG_PATH=/opt/aicli-web/config/config.yaml
AICLI_LOG_LEVEL=info

# Docker 설정
DOCKER_API_VERSION=1.41
DOCKER_HOST=unix:///var/run/docker.sock

# 성능 최적화
GOMEMLIMIT=4GiB
GOMAXPROCS=4

# 보안 설정
AICLI_JWT_SECRET=your-super-secret-jwt-key-change-this
AICLI_API_TOKEN=your-api-token-for-authentication

# 모니터링
PROMETHEUS_ENABLED=true
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
EOF

# 권한 설정
chmod 600 .env
```

#### systemd 서비스 생성
```bash
# 서비스 파일 생성
sudo cat << EOF > /etc/systemd/system/aicli-web.service
[Unit]
Description=AICode Manager Agent Platform
After=network.target docker.service
Requires=docker.service

[Service]
Type=simple
User=aicli
Group=aicli
WorkingDirectory=/opt/aicli-web
EnvironmentFile=/opt/aicli-web/.env
ExecStart=/opt/aicli-web/aicli-api serve --config /opt/aicli-web/config/config.yaml
ExecReload=/bin/kill -HUP \$MAINPID
KillMode=mixed
Restart=always
RestartSec=5
LimitNOFILE=65536

# 보안 설정
NoNewPrivileges=true
PrivateTmp=true
ProtectKernelTunables=true
ProtectControlGroups=true
RestrictRealtime=true

[Install]
WantedBy=multi-user.target
EOF

# 사용자 생성 및 권한 설정
sudo useradd -r -d /opt/aicli-web -s /bin/bash aicli
sudo usermod -aG docker aicli
sudo chown -R aicli:aicli /opt/aicli-web

# 서비스 활성화
sudo systemctl daemon-reload
sudo systemctl enable aicli-web
```

### 3. 서비스 시작 및 확인

#### 서비스 시작
```bash
# 서비스 시작
sudo systemctl start aicli-web

# 상태 확인
sudo systemctl status aicli-web

# 로그 확인
sudo journalctl -u aicli-web -f
```

#### 헬스체크
```bash
# API 서버 확인
curl http://localhost:8080/health

# 메트릭 확인
curl http://localhost:9090/metrics

# 첫 번째 에이전트 생성 테스트
curl -X POST http://localhost:8080/api/v1/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-agent",
    "project_id": "deployment-test",
    "agent_type": "standard"
  }'
```

## 🐳 Docker Compose 배포

### 1. Docker Compose 설정

#### docker-compose.yml
```yaml
version: '3.8'

services:
  aicli-api:
    image: aicli/aicli-api:latest
    container_name: aicli-api
    restart: unless-stopped
    ports:
      - "8080:8080"
      - "9090:9090"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ./data:/data
      - ./config:/config
      - ./workspaces:/workspaces
      - ./logs:/logs
    environment:
      - AICLI_CONFIG_PATH=/config/config.yaml
      - AICLI_LOG_LEVEL=info
      - DOCKER_HOST=unix:///var/run/docker.sock
    networks:
      - aicli-network
    depends_on:
      - redis
      - prometheus
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 60s

  redis:
    image: redis:7-alpine
    container_name: aicli-redis
    restart: unless-stopped
    ports:
      - "6379:6379"
    volumes:
      - redis-data:/data
    command: redis-server --appendonly yes --maxmemory 512mb --maxmemory-policy allkeys-lru
    networks:
      - aicli-network

  prometheus:
    image: prom/prometheus:latest
    container_name: aicli-prometheus
    restart: unless-stopped
    ports:
      - "9091:9090"
    volumes:
      - ./monitoring/prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus-data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--web.console.libraries=/etc/prometheus/console_libraries'
      - '--web.console.templates=/etc/prometheus/consoles'
    networks:
      - aicli-network

  grafana:
    image: grafana/grafana:latest
    container_name: aicli-grafana
    restart: unless-stopped
    ports:
      - "3000:3000"
    volumes:
      - grafana-data:/var/lib/grafana
      - ./monitoring/grafana/dashboards:/etc/grafana/provisioning/dashboards
      - ./monitoring/grafana/datasources:/etc/grafana/provisioning/datasources
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin123
      - GF_USERS_ALLOW_SIGN_UP=false
    networks:
      - aicli-network

volumes:
  redis-data:
  prometheus-data:
  grafana-data:

networks:
  aicli-network:
    driver: bridge
    ipam:
      config:
        - subnet: 172.20.0.0/16
```

### 2. 모니터링 설정

#### Prometheus 설정
```yaml
# monitoring/prometheus.yml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

rule_files:
  - "alert_rules.yml"

scrape_configs:
  - job_name: 'aicli-api'
    static_configs:
      - targets: ['aicli-api:9090']
    scrape_interval: 30s
    metrics_path: /metrics

  - job_name: 'redis'
    static_configs:
      - targets: ['redis:6379']

alerting:
  alertmanagers:
    - static_configs:
        - targets:
          - alertmanager:9093
```

#### Grafana 대시보드
```json
# monitoring/grafana/dashboards/aicli-dashboard.json
{
  "dashboard": {
    "title": "AICode Manager Dashboard",
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
            "expr": "rate(aicli_agent_creation_total[5m])"
          }
        ]
      },
      {
        "title": "Pool Hit Rate",
        "type": "stat",
        "targets": [
          {
            "expr": "aicli_pool_hit_rate"
          }
        ]
      }
    ]
  }
}
```

### 3. 배포 실행

#### 환경 준비
```bash
# 프로젝트 디렉토리 생성
mkdir -p /opt/aicli-web-docker
cd /opt/aicli-web-docker

# 필요한 디렉토리 생성
mkdir -p {data,config,workspaces,logs,monitoring/grafana/{dashboards,datasources}}

# 설정 파일 복사
# (위의 설정 파일들을 각각 해당 위치에 생성)
```

#### 컨테이너 시작
```bash
# 모든 서비스 시작
docker-compose up -d

# 로그 확인
docker-compose logs -f aicli-api

# 상태 확인
docker-compose ps
```

#### 접속 확인
```bash
# API 서버
curl http://localhost:8080/health

# Grafana 대시보드
open http://localhost:3000  # admin/admin123

# Prometheus
open http://localhost:9091
```

## ☸️ Kubernetes 배포

### 1. Kubernetes 매니페스트

#### 네임스페이스 및 ConfigMap
```yaml
# k8s/namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: aicli-system

---
# k8s/configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: aicli-config
  namespace: aicli-system
data:
  config.yaml: |
    server:
      host: "0.0.0.0"
      port: 8080
    database:
      type: "postgres"
      host: "postgres-service"
      port: 5432
      database: "aicli"
    performance:
      agent_pool:
        standard:
          min_size: 10
          max_size: 100
      concurrency:
        max_concurrent_agents: 200
```

#### Secret 설정
```yaml
# k8s/secret.yaml
apiVersion: v1
kind: Secret
metadata:
  name: aicli-secrets
  namespace: aicli-system
type: Opaque
data:
  jwt-secret: <base64-encoded-jwt-secret>
  api-token: <base64-encoded-api-token>
  db-password: <base64-encoded-db-password>
```

#### Deployment
```yaml
# k8s/deployment.yaml
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
      serviceAccountName: aicli-service-account
      containers:
      - name: aicli-api
        image: aicli/aicli-api:latest
        ports:
        - containerPort: 8080
        - containerPort: 9090
        resources:
          requests:
            memory: "1Gi"
            cpu: "500m"
          limits:
            memory: "2Gi"
            cpu: "1000m"
        env:
        - name: AICLI_CONFIG_PATH
          value: "/config/config.yaml"
        - name: AICLI_JWT_SECRET
          valueFrom:
            secretKeyRef:
              name: aicli-secrets
              key: jwt-secret
        volumeMounts:
        - name: config-volume
          mountPath: /config
        - name: docker-socket
          mountPath: /var/run/docker.sock
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
      volumes:
      - name: config-volume
        configMap:
          name: aicli-config
      - name: docker-socket
        hostPath:
          path: /var/run/docker.sock
```

#### Service 및 Ingress
```yaml
# k8s/service.yaml
apiVersion: v1
kind: Service
metadata:
  name: aicli-api-service
  namespace: aicli-system
spec:
  selector:
    app: aicli-api
  ports:
  - name: http
    port: 80
    targetPort: 8080
  - name: metrics
    port: 9090
    targetPort: 9090
  type: ClusterIP

---
# k8s/ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: aicli-ingress
  namespace: aicli-system
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
spec:
  tls:
  - hosts:
    - aicli.yourdomain.com
    secretName: aicli-tls
  rules:
  - host: aicli.yourdomain.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: aicli-api-service
            port:
              number: 80
```

### 2. Kubernetes 배포 실행

#### 매니페스트 적용
```bash
# 매니페스트 적용
kubectl apply -f k8s/

# 배포 상태 확인
kubectl get all -n aicli-system

# Pod 로그 확인
kubectl logs -f deployment/aicli-api -n aicli-system
```

#### 스케일링 설정
```yaml
# k8s/hpa.yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: aicli-api-hpa
  namespace: aicli-system
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: aicli-api
  minReplicas: 3
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

## 🔒 보안 설정

### 1. TLS/SSL 설정

#### Let's Encrypt 인증서
```bash
# cert-manager 설치
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.12.0/cert-manager.yaml

# ClusterIssuer 생성
cat << EOF | kubectl apply -f -
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-prod
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: admin@yourdomain.com
    privateKeySecretRef:
      name: letsencrypt-prod
    solvers:
    - http01:
        ingress:
          class: nginx
EOF
```

### 2. 네트워크 보안

#### 방화벽 설정
```bash
# UFW 방화벽 설정
sudo ufw enable
sudo ufw default deny incoming
sudo ufw default allow outgoing

# 필요한 포트만 개방
sudo ufw allow 22    # SSH
sudo ufw allow 80    # HTTP
sudo ufw allow 443   # HTTPS
sudo ufw allow 8080  # API (내부 네트워크만)
```

#### Docker 네트워크 격리
```bash
# 에이전트 전용 네트워크 생성
docker network create \
  --driver bridge \
  --subnet=172.18.0.0/16 \
  --ip-range=172.18.1.0/24 \
  --gateway=172.18.0.1 \
  aicli-agents
```

### 3. 인증 및 권한 관리

#### JWT 토큰 설정
```bash
# 강력한 JWT 시크릿 생성
openssl rand -base64 64 > /opt/aicli-web/jwt-secret

# API 토큰 생성
openssl rand -hex 32 > /opt/aicli-web/api-token
```

#### RBAC 설정 (Kubernetes)
```yaml
# k8s/rbac.yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: aicli-service-account
  namespace: aicli-system

---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: aicli-cluster-role
rules:
- apiGroups: [""]
  resources: ["pods", "services", "endpoints"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]

---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: aicli-cluster-role-binding
subjects:
- kind: ServiceAccount
  name: aicli-service-account
  namespace: aicli-system
roleRef:
  kind: ClusterRole
  name: aicli-cluster-role
  apiGroup: rbac.authorization.k8s.io
```

## 📊 모니터링 및 로깅

### 1. 메트릭 수집

#### Prometheus 설정 (Kubernetes)
```yaml
# k8s/monitoring/prometheus.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: prometheus-config
  namespace: aicli-system
data:
  prometheus.yml: |
    global:
      scrape_interval: 15s
    scrape_configs:
    - job_name: 'aicli-api'
      kubernetes_sd_configs:
      - role: endpoints
        namespaces:
          names: [aicli-system]
      relabel_configs:
      - source_labels: [__meta_kubernetes_service_name]
        action: keep
        regex: aicli-api-service
```

### 2. 로그 관리

#### ELK 스택 통합
```yaml
# k8s/logging/filebeat.yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: filebeat
  namespace: aicli-system
spec:
  selector:
    matchLabels:
      app: filebeat
  template:
    metadata:
      labels:
        app: filebeat
    spec:
      containers:
      - name: filebeat
        image: elastic/filebeat:8.8.0
        volumeMounts:
        - name: config
          mountPath: /usr/share/filebeat/filebeat.yml
          subPath: filebeat.yml
        - name: varlog
          mountPath: /var/log
          readOnly: true
        - name: varlibdockercontainers
          mountPath: /var/lib/docker/containers
          readOnly: true
      volumes:
      - name: config
        configMap:
          name: filebeat-config
      - name: varlog
        hostPath:
          path: /var/log
      - name: varlibdockercontainers
        hostPath:
          path: /var/lib/docker/containers
```

## 🔄 백업 및 복구

### 1. 데이터베이스 백업

#### 자동 백업 스크립트
```bash
#!/bin/bash
# backup-database.sh

BACKUP_DIR="/opt/aicli-web/backups"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="aicli_backup_${TIMESTAMP}.sql"

# 디렉토리 생성
mkdir -p ${BACKUP_DIR}

# PostgreSQL 백업
if [ "$DB_TYPE" = "postgres" ]; then
    pg_dump -h $DB_HOST -U $DB_USER -d $DB_NAME > ${BACKUP_DIR}/${BACKUP_FILE}
# SQLite 백업
else
    cp /opt/aicli-web/data/aicli.db ${BACKUP_DIR}/aicli_${TIMESTAMP}.db
fi

# 7일 이상 된 백업 삭제
find ${BACKUP_DIR} -name "aicli_backup_*" -mtime +7 -delete

echo "Backup completed: ${BACKUP_FILE}"
```

#### Cron 설정
```bash
# 매일 새벽 2시 백업
0 2 * * * /opt/aicli-web/scripts/backup-database.sh
```

### 2. 설정 백업

#### 설정 파일 백업
```bash
#!/bin/bash
# backup-config.sh

tar -czf /opt/aicli-web/backups/config_$(date +%Y%m%d).tar.gz \
  /opt/aicli-web/config/ \
  /opt/aicli-web/.env \
  /etc/systemd/system/aicli-web.service
```

## 🚨 장애 대응

### 1. 일반적인 문제 해결

#### 서비스 재시작
```bash
# systemd 서비스 재시작
sudo systemctl restart aicli-web

# Docker Compose 재시작
docker-compose restart aicli-api

# Kubernetes 재시작
kubectl rollout restart deployment/aicli-api -n aicli-system
```

#### 로그 분석
```bash
# 시스템 로그
sudo journalctl -u aicli-web -f

# 애플리케이션 로그
tail -f /opt/aicli-web/logs/aicli.log

# Docker 로그
docker logs -f aicli-api

# Kubernetes 로그
kubectl logs -f deployment/aicli-api -n aicli-system
```

### 2. 성능 문제 해결

#### 리소스 모니터링
```bash
# 시스템 리소스 확인
htop
iotop
nethogs

# Docker 리소스 확인
docker stats

# Kubernetes 리소스 확인
kubectl top pods -n aicli-system
kubectl top nodes
```

#### 데이터베이스 최적화
```sql
-- PostgreSQL 성능 튜닝
ANALYZE;
REINDEX DATABASE aicli;

-- SQLite 성능 튜닝
PRAGMA optimize;
VACUUM;
```

## 📋 운영 체크리스트

### 일일 점검 항목
- [ ] 서비스 상태 확인 (`systemctl status aicli-web`)
- [ ] 디스크 사용량 확인 (`df -h`)
- [ ] 로그 에러 확인 (`grep ERROR /opt/aicli-web/logs/aicli.log`)
- [ ] 활성 에이전트 수 확인
- [ ] 성능 메트릭 확인

### 주간 점검 항목
- [ ] 백업 상태 확인
- [ ] 시스템 업데이트 확인
- [ ] 성능 트렌드 분석
- [ ] 용량 계획 검토
- [ ] 보안 패치 확인

### 월간 점검 항목
- [ ] 전체 시스템 성능 리뷰
- [ ] 백업 복구 테스트
- [ ] 보안 감사
- [ ] 용량 확장 계획
- [ ] DR 테스트

---

이 가이드를 통해 AICode Manager를 안전하고 효율적으로 프로덕션 환경에 배포하고 운영할 수 있습니다.

**다음**: [문제 해결 가이드](troubleshooting.md)에서 일반적인 문제와 해결 방법을 확인하세요.