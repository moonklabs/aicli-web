# Claude CLI 통합 트러블슈팅 가이드

## 개요

이 문서는 AICode Manager의 Claude CLI 통합 사용 중 발생할 수 있는 일반적인 문제들과 해결 방법을 안내합니다.

## 빠른 진단

### 기본 상태 확인

```bash
# 시스템 전체 상태 확인
aicli claude status

# 헬스체크 실행
aicli claude health

# 활성 세션 확인
aicli claude session list

# 시스템 리소스 확인
aicli system resources
```

### 로그 확인

```bash
# 최근 에러 로그 확인
aicli logs --level error --last 10

# 특정 세션 로그
aicli logs --session session-123

# 실시간 로그 모니터링
aicli logs --follow
```

## 일반적인 문제 및 해결 방법

### 1. 프로세스 시작 실패

#### 증상
```
Error: Failed to start Claude process
Exit code: 1
```

#### 원인 및 해결

**1.1 Claude CLI가 설치되지 않음**
```bash
# 확인
which claude
# 출력: claude not found

# 해결
# macOS
brew install claude-ai/tap/claude

# Linux
curl -L https://releases.claude.ai/latest/linux/claude -o /usr/local/bin/claude
chmod +x /usr/local/bin/claude

# Windows
# GitHub releases에서 다운로드 후 PATH에 추가
```

**1.2 PATH 설정 문제**
```bash
# 확인
echo $PATH | grep claude

# 해결
export PATH="/usr/local/bin:$PATH"
# .bashrc 또는 .zshrc에 추가
echo 'export PATH="/usr/local/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

**1.3 권한 부족**
```bash
# 확인
ls -la $(which claude)

# 해결
sudo chmod +x /usr/local/bin/claude
```

### 2. 인증 실패

#### 증상
```
Error: Authentication failed
Status: 401 Unauthorized
```

#### 원인 및 해결

**2.1 OAuth 토큰 없음 또는 잘못됨**
```bash
# 확인
echo $CLAUDE_CODE_OAUTH_TOKEN
# 또는
aicli auth status

# 해결
# 새 토큰 획득
aicli auth login
# 또는 환경 변수 설정
export CLAUDE_CODE_OAUTH_TOKEN="your-valid-token"
```

**2.2 토큰 만료**
```bash
# 확인
aicli auth validate

# 해결
aicli auth refresh
# 또는
aicli auth login --force
```

**2.3 토큰 저장소 문제**
```bash
# 확인
ls -la ~/.aicli/auth

# 해결
rm -rf ~/.aicli/auth
aicli auth login
```

### 3. 세션 관리 문제

#### 증상
```
Error: Session not found
Error: Maximum sessions exceeded
```

#### 원인 및 해결

**3.1 세션 한도 초과**
```bash
# 확인
aicli claude session list --status active | wc -l

# 해결
# 비활성 세션 정리
aicli claude session clean

# 또는 특정 세션 삭제
aicli claude session delete session-123

# 최대 세션 수 증가 (config.yaml)
claude:
  session:
    max_concurrent: 20
```

**3.2 세션 상태 불일치**
```bash
# 확인
aicli claude session show session-123 --verbose

# 해결
# 세션 강제 초기화
aicli claude session reset session-123

# 또는 세션 재생성
aicli claude session recreate session-123
```

**3.3 좀비 세션**
```bash
# 확인
ps aux | grep claude

# 해결
# 모든 Claude 프로세스 정리
pkill -f claude
aicli claude session clean --force

# 서비스 재시작
aicli restart
```

### 4. 스트림 처리 오류

#### 증상
```
Error: Stream parsing failed
Error: Connection lost during streaming
```

#### 원인 및 해결

**4.1 네트워크 연결 문제**
```bash
# 확인
curl -I https://api.claude.ai

# 해결
# 네트워크 연결 확인
ping api.claude.ai

# DNS 설정 확인
nslookup api.claude.ai

# 프록시 설정 (필요시)
export HTTP_PROXY="http://proxy.company.com:8080"
export HTTPS_PROXY="http://proxy.company.com:8080"
```

**4.2 버퍼 오버플로우**
```bash
# 증상: 메모리 사용량 급증
top -p $(pgrep aicli)

# 해결: 버퍼 크기 조정 (config.yaml)
claude:
  stream:
    buffer_size: 4096        # 기본값: 1024
    max_line_size: "2MB"     # 기본값: 1MB
    backpressure_threshold: 0.9  # 기본값: 0.8
```

**4.3 JSON 파싱 실패**
```bash
# 디버그 모드로 실행
aicli claude run "test" --debug

# 로그에서 원본 출력 확인
aicli logs --level debug | grep "raw_output"

# 해결: 파서 설정 조정
claude:
  stream:
    parser_timeout: "10s"
    strict_json: false
```

### 5. 메모리 부족

#### 증상
```
Error: Out of memory
Error: Cannot allocate memory
```

#### 원인 및 해결

**5.1 시스템 메모리 부족**
```bash
# 확인
free -h
top

# 해결
# 세션 수 제한
claude:
  session:
    pool_size: 3
    max_concurrent: 5

# 메모리 제한 설정
claude:
  resources:
    limits:
      memory: "1Gi"
```

**5.2 메모리 누수**
```bash
# 메모리 사용량 모니터링
watch -n 5 'ps -p $(pgrep aicli) -o pid,ppid,cmd,pmem,rss'

# 해결
# 가비지 컬렉션 강제 실행
kill -USR1 $(pgrep aicli)

# 또는 서비스 재시작
aicli restart
```

**5.3 대용량 응답 처리**
```bash
# 스트림 청킹 활성화
claude:
  stream:
    chunk_size: 1024
    enable_compression: true
```

### 6. 성능 문제

#### 증상
```
Slow response times
High CPU usage
Frequent timeouts
```

#### 원인 및 해결

**6.1 높은 CPU 사용량**
```bash
# 확인
htop
iostat 1

# 해결
# 동시 실행 수 제한
claude:
  process:
    max_concurrent: 4  # CPU 코어 수에 맞게 조정

# CPU 친화도 설정
taskset -c 0-3 aicli serve
```

**6.2 느린 응답 시간**
```bash
# 확인
aicli claude benchmark

# 해결
# 세션 풀 크기 증가
claude:
  session:
    pool_size: 10
    preload_sessions: 3

# 캐시 활성화
cache:
  enabled: true
  size: 100MB
```

**6.3 디스크 I/O 병목**
```bash
# 확인
iotop

# 해결
# 로그 레벨 낮추기
logging:
  level: "warn"

# 임시 디렉토리를 메모리로 변경
export TMPDIR=/dev/shm
```

### 7. WebSocket 연결 문제

#### 증상
```
WebSocket connection failed
Connection dropped during stream
```

#### 원인 및 해결

**7.1 방화벽 차단**
```bash
# 확인
telnet localhost 8080

# 해결
# 방화벽 규칙 추가 (Ubuntu)
sudo ufw allow 8080

# 방화벽 규칙 추가 (CentOS)
sudo firewall-cmd --permanent --add-port=8080/tcp
sudo firewall-cmd --reload
```

**7.2 프록시 설정 문제**
```bash
# WebSocket용 프록시 설정
# nginx.conf
location /ws/ {
    proxy_pass http://backend;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_set_header Host $host;
}
```

**7.3 연결 타임아웃**
```bash
# 타임아웃 설정 조정
websocket:
  read_timeout: "60s"
  write_timeout: "60s"
  ping_interval: "30s"
  pong_timeout: "10s"
```

## 고급 진단 도구

### 1. 진단 보고서 생성

```bash
# 종합 진단 보고서
aicli claude diagnose --output diagnosis-report.json

# 특정 세션 진단
aicli claude diagnose --session session-123

# 시스템 진단
aicli claude diagnose --system
```

### 2. 상세 로깅 활성화

```bash
# 디버그 모드 활성화
export CLAUDE_LOG_LEVEL=debug
export CLAUDE_TRACE_ENABLED=true

# 특정 컴포넌트만 디버그
export CLAUDE_DEBUG_COMPONENTS="session,stream,process"

# 로그를 파일로 저장
aicli serve --log-file debug.log 2>&1
```

### 3. 성능 프로파일링

```bash
# CPU 프로파일링
aicli profile cpu --duration 30s

# 메모리 프로파일링
aicli profile mem --output mem.prof

# 고루틴 프로파일링
aicli profile goroutine
```

### 4. 네트워크 트레이싱

```bash
# HTTP 요청/응답 로깅
export CLAUDE_HTTP_DEBUG=true

# WebSocket 메시지 로깅
export CLAUDE_WS_DEBUG=true

# 네트워크 패킷 캡처
sudo tcpdump -i any -w claude-network.pcap port 8080
```

## 환경별 문제 해결

### Docker 환경

```bash
# 컨테이너 로그 확인
docker logs aicli-web

# 컨테이너 내부 접근
docker exec -it aicli-web /bin/bash

# 리소스 제한 확인
docker stats aicli-web

# 네트워크 문제 확인
docker network ls
docker network inspect bridge
```

### Kubernetes 환경

```bash
# 파드 상태 확인
kubectl get pods -l app=aicli-web

# 파드 로그 확인
kubectl logs -f deployment/aicli-web

# 파드 내부 접근
kubectl exec -it deployment/aicli-web -- /bin/bash

# 리소스 사용량 확인
kubectl top pods
```

### 개발 환경

```bash
# 개발 서버 실행 (상세 로깅)
make dev-debug

# 테스트 실행
make test-integration

# 코드 핫 리로드
make watch
```

## 모니터링 설정

### 1. 메트릭 모니터링

```bash
# Prometheus 메트릭 확인
curl http://localhost:8080/metrics

# 주요 메트릭
# - claude_sessions_active
# - claude_requests_total
# - claude_errors_total
# - claude_response_time_seconds
```

### 2. 알림 설정

```yaml
# alerts.yaml (Prometheus AlertManager)
groups:
- name: claude
  rules:
  - alert: ClaudeHighErrorRate
    expr: rate(claude_errors_total[5m]) > 0.1
    for: 2m
    labels:
      severity: warning
    annotations:
      summary: "High Claude error rate detected"
      
  - alert: ClaudeMemoryHigh
    expr: claude_memory_usage_bytes > 1073741824  # 1GB
    for: 5m
    labels:
      severity: critical
    annotations:
      summary: "Claude memory usage is high"
```

### 3. 대시보드 설정

```json
// Grafana 대시보드 JSON
{
  "dashboard": {
    "title": "Claude CLI Integration",
    "panels": [
      {
        "title": "Active Sessions",
        "targets": [
          {
            "expr": "claude_sessions_active"
          }
        ]
      },
      {
        "title": "Request Rate",
        "targets": [
          {
            "expr": "rate(claude_requests_total[5m])"
          }
        ]
      }
    ]
  }
}
```

## 예방적 유지보수

### 1. 정기 점검

```bash
# 일일 점검 스크립트
#!/bin/bash
# daily-check.sh

echo "=== Daily Claude System Check ==="
echo "Date: $(date)"

# 시스템 상태
aicli claude health

# 세션 정리
aicli claude session clean

# 로그 로테이션
logrotate /etc/logrotate.d/aicli

# 디스크 사용량
df -h | grep -E "/(data|logs)"

# 메모리 사용량
free -h
```

### 2. 성능 튜닝

```bash
# 주간 성능 리포트
#!/bin/bash
# weekly-performance.sh

# 성능 메트릭 수집
aicli claude metrics --output metrics-$(date +%Y%m%d).json

# 느린 쿼리 분석
aicli claude analyze-slow-queries

# 리소스 사용 패턴 분석
aicli claude resource-analysis --days 7
```

### 3. 용량 계획

```bash
# 월간 용량 리포트
#!/bin/bash
# monthly-capacity.sh

# 세션 증가 추세
aicli claude trends --metric sessions --period month

# 스토리지 사용량 증가
aicli claude trends --metric storage --period month

# 예상 리소스 요구량
aicli claude capacity-planning --forecast 3months
```

## 에스컬레이션 절차

### Level 1: 기본 문제 해결
1. 기본 상태 확인
2. 로그 분석
3. 서비스 재시작
4. 설정 검증

### Level 2: 고급 진단
1. 진단 보고서 생성
2. 성능 프로파일링
3. 네트워크 분석
4. 시스템 리소스 분석

### Level 3: 전문가 지원
1. GitHub 이슈 생성
2. 진단 데이터 첨부
3. 재현 단계 문서화
4. 환경 정보 포함

### 연락처
- GitHub Issues: https://github.com/your-org/aicli-web/issues
- 지원 이메일: support@your-domain.com
- Slack 채널: #aicli-support

이 트러블슈팅 가이드를 통해 Claude CLI 통합 시스템의 대부분의 문제를 해결할 수 있습니다. 문제가 지속되거나 이 가이드에 없는 문제를 겪으신다면 위의 에스컬레이션 절차를 따라 지원을 요청해주세요.