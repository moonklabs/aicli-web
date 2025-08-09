# PTY Streaming 운영 가이드

## 목차
1. [시스템 요구사항](#시스템-요구사항)
2. [설치 및 배포](#설치-및-배포)
3. [구성 관리](#구성-관리)
4. [모니터링](#모니터링)
5. [성능 튜닝](#성능-튜닝)
6. [장애 처리](#장애-처리)
7. [백업 및 복구](#백업-및-복구)
8. [보안 관리](#보안-관리)
9. [유지보수](#유지보수)

## 시스템 요구사항

### 하드웨어 요구사항

| 구분 | 최소 사양 | 권장 사양 |
|------|------------|------------|
| CPU | 2 Core | 4 Core 이상 |
| RAM | 4GB | 8GB 이상 |
| Disk | 20GB SSD | 50GB SSD 이상 |
| Network | 100Mbps | 1Gbps |

### 소프트웨어 요구사항

- **운영체제:** Linux (Ubuntu 20.04+, CentOS 8+)
- **Docker:** 20.10.0 이상
- **Go:** 1.21 이상 (개발 환경)
- **Node.js:** 18.0 이상 (프론트엔드)

### 포트 요구사항

| 포트 | 용도 | 프로토콜 |
|------|------|----------|
| 8080 | API 서버 | HTTP/HTTPS |
| 8081 | WebSocket | WS/WSS |
| 9090 | Prometheus 메트릭 | HTTP |
| 6060 | pprof (디버그) | HTTP |

## 설치 및 배포

### Docker Compose 배포

#### 1. docker-compose.yml 작성

```yaml
version: '3.8'

services:
  aicli-api:
    image: aicli/api:latest
    container_name: aicli-api
    ports:
      - "8080:8080"
      - "8081:8081"
    environment:
      - ENV=production
      - LOG_LEVEL=info
      - DB_PATH=/data/aicli.db
      - DOCKER_HOST=unix:///var/run/docker.sock
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - ./data:/data
      - ./config:/config:ro
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
    networks:
      - aicli-network
  
  prometheus:
    image: prom/prometheus:latest
    container_name: prometheus
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml:ro
      - prometheus-data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
    restart: unless-stopped
    networks:
      - aicli-network
  
  grafana:
    image: grafana/grafana:latest
    container_name: grafana
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_USER=admin
      - GF_SECURITY_ADMIN_PASSWORD=admin
    volumes:
      - grafana-data:/var/lib/grafana
      - ./grafana/dashboards:/etc/grafana/provisioning/dashboards:ro
      - ./grafana/datasources:/etc/grafana/provisioning/datasources:ro
    restart: unless-stopped
    networks:
      - aicli-network

volumes:
  prometheus-data:
  grafana-data:

networks:
  aicli-network:
    driver: bridge
```

#### 2. 배포 스크립트

```bash
#!/bin/bash
# deploy.sh

set -e

# 환경 변수 로드
source .env

# Docker 이미지 풀
echo "Pulling latest images..."
docker-compose pull

# 기존 컨테이너 중지
echo "Stopping existing containers..."
docker-compose down

# 데이터베이스 백업
echo "Backing up database..."
cp data/aicli.db data/aicli.db.backup.$(date +%Y%m%d_%H%M%S)

# 새 컨테이너 시작
echo "Starting new containers..."
docker-compose up -d

# 헬스 체크 대기
echo "Waiting for health check..."
sleep 10

# 상태 확인
if curl -f http://localhost:8080/health; then
    echo "Deployment successful!"
    docker-compose ps
else
    echo "Health check failed! Rolling back..."
    docker-compose down
    docker-compose up -d --scale aicli-api=0
    exit 1
fi
```

### Kubernetes 배포

#### Helm Chart 설치

```bash
# Helm 차트 설치
helm repo add aicli https://charts.aicli.io
helm repo update

# 값 파일 생성
cat > values.yaml <<EOF
replicaCount: 3

image:
  repository: aicli/api
  tag: latest
  pullPolicy: IfNotPresent

service:
  type: LoadBalancer
  port: 8080
  websocketPort: 8081

ingress:
  enabled: true
  className: nginx
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
    nginx.ingress.kubernetes.io/websocket-services: aicli-websocket
  hosts:
    - host: api.aicli.io
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: aicli-tls
      hosts:
        - api.aicli.io

resources:
  limits:
    cpu: 1000m
    memory: 1Gi
  requests:
    cpu: 500m
    memory: 512Mi

autoscaling:
  enabled: true
  minReplicas: 2
  maxReplicas: 10
  targetCPUUtilizationPercentage: 70
  targetMemoryUtilizationPercentage: 80

persistence:
  enabled: true
  storageClass: fast-ssd
  size: 10Gi

monitoring:
  enabled: true
  serviceMonitor: true
EOF

# Helm 차트 설치
helm install aicli aicli/aicli-manager -f values.yaml
```

## 구성 관리

### 주요 설정 파일

#### config.yaml

```yaml
# 서버 설정
server:
  host: 0.0.0.0
  port: 8080
  websocket_port: 8081
  read_timeout: 30s
  write_timeout: 30s
  idle_timeout: 120s

# PTY 설정
pty:
  max_sessions: 100
  session_timeout: 30m
  idle_timeout: 10m
  default_rows: 24
  default_cols: 80
  buffer_size: 4096

# WebSocket 설정
websocket:
  max_connections: 1000
  ping_interval: 30s
  pong_timeout: 60s
  write_deadline: 10s
  read_deadline: 60s
  max_message_size: 524288  # 512KB
  enable_compression: true
  compression_level: 1

# 플로우 제어 설정
flow_control:
  enabled: true
  backpressure_threshold: 80  # %
  throttle_enabled: true
  min_rate: 100     # messages/sec
  max_rate: 10000   # messages/sec
  adaptive_mode: true

# 성능 최적화
optimization:
  memory_pool_enabled: true
  pool_size: 100
  gc_tuning: true
  gc_percent: 50
  io_buffer_size: 65536
  batch_processing: true
  batch_size: 100
  batch_timeout: 100ms

# Docker 설정
docker:
  endpoint: unix:///var/run/docker.sock
  api_version: 1.41
  timeout: 30s
  max_concurrent_operations: 10

# 로그 설정
logging:
  level: info
  format: json
  output: stdout
  file:
    enabled: true
    path: /var/log/aicli
    max_size: 100M
    max_backups: 10
    max_age: 30
    compress: true

# 모니터링
monitoring:
  metrics_enabled: true
  metrics_port: 9090
  pprof_enabled: false
  pprof_port: 6060
  health_check_interval: 30s
  stats_interval: 60s

# 보안 설정
security:
  tls_enabled: false
  tls_cert: /etc/ssl/certs/server.crt
  tls_key: /etc/ssl/private/server.key
  auth_enabled: true
  jwt_secret: ${JWT_SECRET}
  session_timeout: 24h
  max_login_attempts: 5
  lockout_duration: 15m
```

### 환경별 설정

#### 개발 환경

```yaml
# config.dev.yaml
extends: config.yaml

server:
  port: 8080

logging:
  level: debug

monitoring:
  pprof_enabled: true

security:
  tls_enabled: false
  auth_enabled: false
```

#### 프로덕션 환경

```yaml
# config.prod.yaml
extends: config.yaml

server:
  port: 443

logging:
  level: warn
  output: file

monitoring:
  pprof_enabled: false

security:
  tls_enabled: true
  auth_enabled: true
```

## 모니터링

### Prometheus 설정

```yaml
# prometheus.yml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'aicli-api'
    static_configs:
      - targets: ['aicli-api:9090']
    metrics_path: '/metrics'

  - job_name: 'docker'
    static_configs:
      - targets: ['docker-exporter:9323']

alert_rules:
  - name: aicli_alerts
    interval: 30s
    rules:
      - alert: HighMemoryUsage
        expr: process_resident_memory_bytes / 1024 / 1024 > 1000
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High memory usage detected"
          description: "Memory usage is above 1GB (current: {{ $value }}MB)"
      
      - alert: TooManyPTYSessions
        expr: pty_sessions_active > 100
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "Too many active PTY sessions"
          description: "Active sessions: {{ $value }}"
      
      - alert: HighBackpressure
        expr: flow_backpressure_level > 3
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High backpressure detected"
          description: "Backpressure level: {{ $value }}"
```

### Grafana 대시보드

#### 주요 메트릭 패널

1. **시스템 개요**
   - Active PTY Sessions
   - WebSocket Connections
   - CPU Usage
   - Memory Usage
   - Network I/O

2. **PTY 세션 메트릭**
   - Sessions Created/Terminated
   - Average Session Duration
   - Idle Sessions
   - Session Error Rate

3. **WebSocket 메트릭**
   - Connection Count
   - Message Rate
   - Bytes Transferred
   - Connection Errors

4. **플로우 제어 메트릭**
   - Backpressure Level
   - Throttled Connections
   - Dropped Messages
   - Buffer Utilization

### 로그 모니터링

#### ELK Stack 통합

```yaml
# filebeat.yml
filebeat.inputs:
  - type: docker
    containers:
      ids:
        - "*"
    processors:
      - add_docker_metadata: ~
      - decode_json_fields:
          fields: ["message"]
          target: ""
          overwrite_keys: true

output.elasticsearch:
  hosts: ["elasticsearch:9200"]
  index: "aicli-%{+yyyy.MM.dd}"

processors:
  - add_host_metadata:
      when.not.contains:
        tags: forwarded
```

#### 로그 패턴 분석

```json
// Kibana 검색 쿼리 예제

// 에러 로그 검색
{
  "query": {
    "bool": {
      "must": [
        { "match": { "level": "error" }},
        { "range": { "@timestamp": { "gte": "now-1h" }}}
      ]
    }
  }
}

// PTY 세션 분석
{
  "query": {
    "match": { "module": "pty" }
  },
  "aggs": {
    "sessions_per_hour": {
      "date_histogram": {
        "field": "@timestamp",
        "interval": "hour"
      }
    }
  }
}
```

## 성능 튜닝

### 시스템 튜닝

#### Linux 커널 튜닝

```bash
# /etc/sysctl.d/99-aicli.conf

# 네트워크 튜닝
net.core.somaxconn = 65535
net.ipv4.tcp_max_syn_backlog = 8192
net.core.netdev_max_backlog = 65536
net.ipv4.tcp_tw_reuse = 1
net.ipv4.tcp_fin_timeout = 30
net.ipv4.ip_local_port_range = 10000 65000

# 파일 디스크립터 제한
fs.file-max = 2097152
fs.nr_open = 1048576

# 메모리 튜닝
vm.swappiness = 10
vm.dirty_ratio = 15
vm.dirty_background_ratio = 5
```

#### Docker 튜닝

```json
// /etc/docker/daemon.json
{
  "storage-driver": "overlay2",
  "storage-opts": [
    "overlay2.override_kernel_check=true"
  ],
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "10m",
    "max-file": "3"
  },
  "default-ulimits": {
    "nofile": {
      "Name": "nofile",
      "Hard": 65536,
      "Soft": 65536
    }
  }
}
```

### 애플리케이션 튜닝

#### Go 런타임 튜닝

```go
// main.go
import "runtime"

func init() {
    // CPU 코어 활용
    runtime.GOMAXPROCS(runtime.NumCPU())
    
    // GC 튜닝
    runtime.SetGCPercent(50)
    
    // 메모리 제한
    runtime.MemProfileRate = 0
}
```

#### 커넥션 풀 튜닝

```yaml
# 연결 풀 설정
connection_pool:
  max_idle: 100
  max_open: 1000
  max_lifetime: 30m
  idle_timeout: 10m
```

## 장애 처리

### 일반적인 장애 시나리오

#### 1. 서비스 응답 없음

**증상:**
- API 요청에 응답하지 않음
- Health check 실패

**진단 단계:**

```bash
# 1. 프로세스 상태 확인
systemctl status aicli-api

# 2. 로그 확인
journalctl -u aicli-api -n 100

# 3. 포트 상태 확인
netstat -tlnp | grep 8080

# 4. Docker 상태 확인
docker ps -a | grep aicli
```

**해결 방법:**

```bash
# 서비스 재시작
systemctl restart aicli-api

# Docker 컨테이너 재시작
docker restart aicli-api

# 강제 종료 후 재시작
docker kill aicli-api
docker start aicli-api
```

#### 2. 메모리 부족

**증상:**
- OOM (Out of Memory) 에러
- 응답 속도 저하

**진단:**

```bash
# 메모리 사용량 확인
free -h
top -p $(pgrep aicli)

# Docker 메모리 사용량
docker stats aicli-api

# 메모리 덤프 분석
curl http://localhost:6060/debug/pprof/heap > heap.prof
go tool pprof heap.prof
```

**해결:**

```yaml
# Docker 메모리 제한 증가
services:
  aicli-api:
    deploy:
      resources:
        limits:
          memory: 2G
        reservations:
          memory: 1G
```

#### 3. PTY 세션 누수

**증상:**
- 세션이 종료되지 않음
- 리소스 고갈

**진단:**

```bash
# 활성 세션 확인
curl http://localhost:8080/api/v1/pty/sessions | jq '.[] | select(.status=="active")' | wc -l

# Docker 프로세스 확인
docker exec aicli-api ps aux | grep bash | wc -l

# 세션 상세 정보
curl http://localhost:8080/api/v1/metrics | jq '.pty_sessions'
```

**해결:**

```bash
# 유휴 세션 정리
curl -X POST http://localhost:8080/api/v1/admin/cleanup-sessions

# 모든 세션 강제 종료
for session in $(curl http://localhost:8080/api/v1/pty/sessions | jq -r '.[].id'); do
  curl -X DELETE http://localhost:8080/api/v1/pty/sessions/$session
done
```

### 장애 복구 프로세스

#### 자동 복구 스크립트

```bash
#!/bin/bash
# recovery.sh

SERVICE="aicli-api"
MAX_RETRIES=3
RETRY_DELAY=10

log() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] $1" | tee -a /var/log/aicli-recovery.log
}

check_health() {
    curl -f -s http://localhost:8080/health > /dev/null 2>&1
    return $?
}

restart_service() {
    log "Restarting $SERVICE..."
    docker restart $SERVICE
    sleep $RETRY_DELAY
}

cleanup_resources() {
    log "Cleaning up resources..."
    
    # 유휴 세션 정리
    curl -X POST http://localhost:8080/api/v1/admin/cleanup-sessions
    
    # 메모리 정리
    docker exec $SERVICE sh -c 'echo 3 > /proc/sys/vm/drop_caches'
    
    # 로그 로테이션
    find /var/log/aicli -name "*.log" -mtime +7 -delete
}

# 메인 복구 루프
retries=0
while [ $retries -lt $MAX_RETRIES ]; do
    if check_health; then
        log "Service is healthy"
        exit 0
    fi
    
    log "Health check failed (attempt $((retries+1))/$MAX_RETRIES)"
    restart_service
    
    if check_health; then
        log "Service recovered successfully"
        cleanup_resources
        exit 0
    fi
    
    retries=$((retries+1))
done

log "Service recovery failed after $MAX_RETRIES attempts"
log "Sending alert..."

# 알림 전송
curl -X POST https://hooks.slack.com/services/xxx \
    -H 'Content-Type: application/json' \
    -d '{"text":"CRITICAL: AICode API service recovery failed!"}'

exit 1
```

## 백업 및 복구

### 백업 전략

#### 데이터 백업

```bash
#!/bin/bash
# backup.sh

BACKUP_DIR="/backup/aicli"
DATE=$(date +%Y%m%d_%H%M%S)
RETENTION_DAYS=30

# 디렉토리 생성
mkdir -p $BACKUP_DIR/$DATE

# 데이터베이스 백업
cp /data/aicli.db $BACKUP_DIR/$DATE/

# 설정 파일 백업
tar czf $BACKUP_DIR/$DATE/config.tar.gz /config

# 세션 스냅샷 백업
tar czf $BACKUP_DIR/$DATE/snapshots.tar.gz /data/snapshots

# S3 업로드 (선택적)
aws s3 sync $BACKUP_DIR/$DATE s3://aicli-backups/$DATE/

# 오래된 백업 삭제
find $BACKUP_DIR -type d -mtime +$RETENTION_DAYS -exec rm -rf {} +

echo "Backup completed: $BACKUP_DIR/$DATE"
```

#### 자동 백업 스케줄링

```cron
# crontab -e

# 매일 새벽 2시 백업
0 2 * * * /opt/aicli/scripts/backup.sh

# 매주 일요일 전체 백업
0 3 * * 0 /opt/aicli/scripts/full-backup.sh

# 매월 1일 아카이브
0 4 1 * * /opt/aicli/scripts/archive-backup.sh
```

### 복구 프로세스

#### 복구 스크립트

```bash
#!/bin/bash
# restore.sh

if [ $# -ne 1 ]; then
    echo "Usage: $0 <backup-date>"
    echo "Example: $0 20240101_020000"
    exit 1
fi

BACKUP_DATE=$1
BACKUP_DIR="/backup/aicli/$BACKUP_DATE"

if [ ! -d "$BACKUP_DIR" ]; then
    echo "Backup not found: $BACKUP_DIR"
    exit 1
fi

echo "Restoring from backup: $BACKUP_DATE"

# 서비스 중지
docker-compose down

# 현재 데이터 백업
mkdir -p /backup/aicli/before-restore
cp -r /data/* /backup/aicli/before-restore/

# 데이터 복구
cp $BACKUP_DIR/aicli.db /data/
tar xzf $BACKUP_DIR/config.tar.gz -C /
tar xzf $BACKUP_DIR/snapshots.tar.gz -C /

# 권한 설정
chown -R aicli:aicli /data
chown -R aicli:aicli /config

# 서비스 시작
docker-compose up -d

# 헬스 체크
sleep 10
if curl -f http://localhost:8080/health; then
    echo "Restore completed successfully"
else
    echo "Restore failed! Rolling back..."
    docker-compose down
    cp -r /backup/aicli/before-restore/* /data/
    docker-compose up -d
    exit 1
fi
```

## 보안 관리

### 접근 제어

#### 네트워크 보안

```bash
# iptables 규칙
iptables -A INPUT -p tcp --dport 8080 -s 10.0.0.0/8 -j ACCEPT
iptables -A INPUT -p tcp --dport 8080 -j DROP
iptables -A INPUT -p tcp --dport 9090 -s 127.0.0.1 -j ACCEPT
iptables -A INPUT -p tcp --dport 9090 -j DROP
```

#### 인증 및 권한

```yaml
# RBAC 설정
auth:
  providers:
    - type: jwt
      secret: ${JWT_SECRET}
      issuer: aicli
      audience: api
    - type: oauth2
      client_id: ${OAUTH_CLIENT_ID}
      client_secret: ${OAUTH_CLIENT_SECRET}
      redirect_url: https://api.aicli.io/callback

roles:
  - name: admin
    permissions:
      - pty:*
      - system:*
  - name: user
    permissions:
      - pty:read
      - pty:write
  - name: viewer
    permissions:
      - pty:read
      - metrics:read
```

### 보안 감사

#### 감사 로그

```json
// 감사 로그 형식
{
  "timestamp": "2024-01-01T12:00:00Z",
  "event_type": "session_created",
  "user_id": "user123",
  "session_id": "sess-abc123",
  "container_id": "container-xyz",
  "source_ip": "192.168.1.100",
  "user_agent": "Mozilla/5.0...",
  "action": "CREATE_PTY_SESSION",
  "result": "SUCCESS",
  "metadata": {
    "rows": 24,
    "cols": 80,
    "shell": "/bin/bash"
  }
}
```

#### 보안 스캐닝

```bash
# 취약점 스캐닝
trivy image aicli/api:latest

# 정적 코드 분석
gosec ./...

# 의존성 검사
go list -m all | nancy sleuth
```

## 유지보수

### 정기 점검 체크리스트

#### 일일 점검
- [ ] 시스템 헬스 체크
- [ ] 로그 에러 확인
- [ ] 리소스 사용량 확인
- [ ] 백업 상태 확인

#### 주간 점검
- [ ] 성능 메트릭 분석
- [ ] 보안 로그 검토
- [ ] 시스템 업데이트 확인
- [ ] 용량 계획 검토

#### 월간 점검
- [ ] 전체 시스템 백업
- [ ] 복구 테스트
- [ ] 보안 패치 적용
- [ ] 성능 튜닝 검토

### 업그레이드 프로세스

#### 무중단 배포

```bash
#!/bin/bash
# rolling-update.sh

IMAGE_TAG=$1
SERVICE="aicli-api"
REPLICAS=3

# Blue-Green 배포
for i in $(seq 1 $REPLICAS); do
    echo "Updating replica $i/$REPLICAS"
    
    # 새 컨테이너 시작
    docker run -d --name ${SERVICE}-new-$i $IMAGE_TAG
    
    # 헬스 체크 대기
    sleep 30
    
    # 트래픽 전환
    docker exec nginx nginx -s reload
    
    # 기존 컨테이너 종료
    docker stop ${SERVICE}-old-$i
    docker rm ${SERVICE}-old-$i
    
    # 이름 변경
    docker rename ${SERVICE}-new-$i ${SERVICE}-$i
done

echo "Rolling update completed"
```

### 트러블슈팅 가이드

자세한 트러블슈팅 가이드는 [개발자 가이드](developer-guide.md#문제-해결)를 참조하세요.

## 참고 자료

- [개발자 가이드](developer-guide.md)
- [API 레퍼런스](/docs/api/pty-streaming-openapi.yaml)
- [Docker 공식 문서](https://docs.docker.com)
- [Prometheus 모니터링](https://prometheus.io/docs)
- [Grafana 대시보드](https://grafana.com/docs)