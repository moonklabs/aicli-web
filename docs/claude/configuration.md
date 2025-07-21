# Claude CLI 통합 설정 가이드

## 개요

이 문서는 AICode Manager의 Claude CLI 통합을 위한 설정 방법을 안내합니다. 환경 변수, 설정 파일, 런타임 설정 등을 포함합니다.

## 환경 변수

### 필수 환경 변수

#### CLAUDE_CODE_OAUTH_TOKEN
Claude CLI 인증을 위한 OAuth 토큰입니다.

```bash
export CLAUDE_CODE_OAUTH_TOKEN="your-oauth-token-here"
```

**획득 방법**:
1. Claude.ai 웹사이트 로그인
2. 개발자 설정에서 OAuth 토큰 생성
3. 토큰 복사 후 환경 변수로 설정

#### CLAUDE_API_KEY (대체 옵션)
OAuth 토큰 대신 사용할 수 있는 API 키입니다.

```bash
export CLAUDE_API_KEY="your-api-key-here"
```

### 선택적 환경 변수

#### Claude 시스템 설정

```bash
# 최대 동시 세션 수
export CLAUDE_MAX_SESSIONS=10

# 세션 타임아웃
export CLAUDE_SESSION_TIMEOUT=30m

# 로그 레벨
export CLAUDE_LOG_LEVEL=info

# 프로세스 시작 타임아웃
export CLAUDE_STARTUP_TIMEOUT=30s

# 헬스체크 간격
export CLAUDE_HEALTH_CHECK_INTERVAL=30s
```

#### 성능 튜닝

```bash
# 스트림 버퍼 크기
export CLAUDE_STREAM_BUFFER_SIZE=1024

# 최대 라인 크기
export CLAUDE_MAX_LINE_SIZE=1048576

# 백프레셔 임계값
export CLAUDE_BACKPRESSURE_THRESHOLD=0.8

# 세션 풀 크기
export CLAUDE_SESSION_POOL_SIZE=5
```

#### 에러 처리

```bash
# 최대 재시도 횟수
export CLAUDE_MAX_RETRIES=3

# 재시도 백오프 전략
export CLAUDE_RETRY_BACKOFF=exponential

# 회로 차단기 임계값
export CLAUDE_CIRCUIT_BREAKER_THRESHOLD=5

# 에러 복구 활성화
export CLAUDE_RECOVERY_ENABLED=true
```

## 설정 파일

### 기본 설정 파일 (config.yaml)

```yaml
# AICode Manager 전체 설정
app:
  name: "aicli-web"
  version: "0.1.0"
  debug: false

# 서버 설정
server:
  host: "0.0.0.0"
  port: 8080
  read_timeout: "30s"
  write_timeout: "30s"

# Claude CLI 통합 설정
claude:
  # 프로세스 관리
  process:
    max_concurrent: 10
    startup_timeout: "30s"
    shutdown_timeout: "10s"
    health_check_interval: "30s"
    restart_delay: "5s"
    kill_timeout: "15s"
    
  # 인증 설정
  auth:
    token_refresh_threshold: "30m"
    token_validation_interval: "5m"
    secure_storage: true
    
  # 스트림 처리
  stream:
    buffer_size: 1024
    max_line_size: "1MB"
    backpressure_threshold: 0.8
    backpressure_policy: "drop_oldest"  # drop_oldest, drop_newest, block_until_ready
    parser_timeout: "5s"
    
  # 세션 관리
  session:
    pool_size: 5
    max_idle_time: "30m"
    cleanup_interval: "5m"
    reuse_timeout: "10m"
    max_lifetime: "2h"
    auto_cleanup: true
    
  # 에러 처리
  error_handling:
    max_retries: 3
    retry_backoff: "exponential"  # linear, exponential
    base_delay: "1s"
    max_delay: "30s"
    circuit_breaker_threshold: 5
    circuit_breaker_timeout: "30s"
    recovery_enabled: true
    
  # 모니터링
  monitoring:
    metrics_enabled: true
    health_check_enabled: true
    log_requests: true
    trace_enabled: false
    
  # 리소스 제한
  resources:
    limits:
      cpu: "2.0"
      memory: "2Gi"
      processes: 50
      concurrent_executions: 20
    requests:
      cpu: "0.5"
      memory: "512Mi"

# 도구 권한 설정
tools:
  allowed:
    - "Read"
    - "Write"
    - "Bash"
    - "Search"
    - "WebSearch"
    
  # 도구별 세부 설정
  config:
    Bash:
      timeout: "30s"
      allowed_commands:
        - "ls"
        - "cat"
        - "grep"
        - "find"
        - "head"
        - "tail"
      blocked_commands:
        - "rm"
        - "sudo"
        - "chmod"
    Write:
      max_file_size: "10MB"
      allowed_paths:
        - "/workspace"
        - "/tmp"
      blocked_paths:
        - "/etc"
        - "/usr"
        - "/var"
    Read:
      max_file_size: "50MB"
      allowed_extensions:
        - ".go"
        - ".js"
        - ".ts"
        - ".py"
        - ".md"
        - ".yaml"
        - ".json"

# 보안 설정
security:
  allowed_hosts:
    - "api.claude.ai"
    - "claude.ai"
  network_isolation: true
  filesystem_restrictions:
    - "/workspace"
    - "/tmp"
  environment_isolation: true
  
# 로깅 설정
logging:
  level: "info"
  format: "json"
  output: "stdout"
  file:
    enabled: false
    path: "/var/log/aicli/claude.log"
    max_size: "100MB"
    max_backups: 5
    max_age: 30

# 데이터베이스 설정
database:
  type: "sqlite"
  path: "./data/aicli.db"
  connection_pool:
    max_open: 25
    max_idle: 25
    max_lifetime: "5m"
```

### 개발 환경 설정 (config.dev.yaml)

```yaml
# 개발 환경 전용 설정
app:
  debug: true

claude:
  process:
    max_concurrent: 3
  session:
    pool_size: 2
  monitoring:
    trace_enabled: true
  error_handling:
    max_retries: 1

logging:
  level: "debug"
  format: "text"

database:
  path: "./dev.db"
```

### 프로덕션 설정 (config.prod.yaml)

```yaml
# 프로덕션 환경 전용 설정
app:
  debug: false

claude:
  process:
    max_concurrent: 50
  session:
    pool_size: 20
    max_idle_time: "1h"
  resources:
    limits:
      cpu: "8.0"
      memory: "8Gi"
      processes: 200

logging:
  level: "warn"
  file:
    enabled: true
    path: "/var/log/aicli/claude.log"

database:
  path: "/data/aicli.db"
```

## 런타임 설정

### 동적 설정 변경

일부 설정은 런타임에 동적으로 변경할 수 있습니다:

```bash
# 로그 레벨 변경
aicli config set logging.level debug

# 세션 풀 크기 조정
aicli config set claude.session.pool_size 10

# 백프레셔 정책 변경
aicli config set claude.stream.backpressure_policy block_until_ready
```

### 설정 확인

```bash
# 현재 설정 확인
aicli config show

# 특정 설정 확인
aicli config get claude.session.pool_size

# 설정 파일 검증
aicli config validate
```

### 설정 리로드

```bash
# 설정 파일 리로드 (다운타임 없이)
aicli config reload

# 특정 모듈만 리로드
aicli config reload claude
```

## Docker 설정

### Docker Compose 설정

```yaml
# docker-compose.yml
version: '3.8'

services:
  aicli-web:
    build: .
    ports:
      - "8080:8080"
    environment:
      - CLAUDE_CODE_OAUTH_TOKEN=${CLAUDE_CODE_OAUTH_TOKEN}
      - CLAUDE_LOG_LEVEL=info
      - CLAUDE_MAX_SESSIONS=10
    volumes:
      - ./config:/app/config
      - ./data:/app/data
      - ./workspace:/app/workspace
    restart: unless-stopped
```

### 환경 변수 파일 (.env)

```bash
# .env 파일
CLAUDE_CODE_OAUTH_TOKEN=your-token-here
CLAUDE_LOG_LEVEL=info
CLAUDE_MAX_SESSIONS=10
CLAUDE_SESSION_TIMEOUT=30m

# 개발 환경
APP_ENV=development
DATABASE_PATH=./dev.db

# 프로덕션 환경
# APP_ENV=production
# DATABASE_PATH=/data/aicli.db
```

## 설정 우선순위

설정 값은 다음 우선순위에 따라 적용됩니다:

1. **명령줄 플래그** (최고 우선순위)
2. **환경 변수**
3. **설정 파일**
4. **기본값** (최저 우선순위)

### 예시

```bash
# 명령줄에서 로그 레벨 지정 (최고 우선순위)
aicli serve --log-level debug

# 환경 변수 (두 번째 우선순위)
export CLAUDE_LOG_LEVEL=info

# 설정 파일 (세 번째 우선순위)
logging:
  level: "warn"
  
# 기본값 (최저 우선순위)
# level: "info"
```

## 설정 템플릿

### 최소 설정

```yaml
# minimal-config.yaml
claude:
  process:
    max_concurrent: 5
  session:
    pool_size: 2
    
tools:
  allowed:
    - "Read"
    - "Write"
```

### 고성능 설정

```yaml
# high-performance-config.yaml
claude:
  process:
    max_concurrent: 100
  session:
    pool_size: 50
    max_idle_time: "2h"
  stream:
    buffer_size: 4096
    backpressure_threshold: 0.9
  resources:
    limits:
      cpu: "16.0"
      memory: "16Gi"
      concurrent_executions: 200
```

### 보안 강화 설정

```yaml
# secure-config.yaml
claude:
  auth:
    token_validation_interval: "1m"
    secure_storage: true
    
tools:
  allowed:
    - "Read"
  config:
    Read:
      max_file_size: "1MB"
      allowed_paths:
        - "/workspace/readonly"
        
security:
  network_isolation: true
  filesystem_restrictions:
    - "/workspace/readonly"
  environment_isolation: true
```

## 설정 검증

### 스키마 검증

```bash
# 설정 파일 문법 확인
aicli config validate --file config.yaml

# 스키마 규칙 확인
aicli config check --strict

# 설정 충돌 감지
aicli config conflicts
```

### 테스트 설정

```bash
# 설정 테스트 (실제 서비스 시작하지 않고 검증)
aicli config test

# 특정 설정으로 드라이런
aicli serve --dry-run --config config.yaml
```

## 모니터링 및 디버깅

### 설정 로깅

```yaml
logging:
  config_changes: true  # 설정 변경 로깅
  startup_config: true  # 시작 시 설정 덤프
```

### 설정 메트릭

```bash
# 설정 관련 메트릭 확인
curl http://localhost:8080/metrics | grep config

# 설정 변경 히스토리
aicli config history

# 활성 설정 덤프
aicli config dump > current-config.yaml
```

## 문제 해결

### 일반적인 설정 문제

#### 1. 토큰 인식 실패
```
Error: CLAUDE_CODE_OAUTH_TOKEN not set
```

**해결**:
- 환경 변수 확인: `echo $CLAUDE_CODE_OAUTH_TOKEN`
- 토큰 유효성 확인: `aicli auth validate`
- 설정 파일에 토큰 지정

#### 2. 설정 파일 로드 실패
```
Error: config file not found
```

**해결**:
- 파일 경로 확인: `ls -la config.yaml`
- 권한 확인: `ls -la config.yaml`
- 설정 경로 지정: `--config /path/to/config.yaml`

#### 3. 메모리 부족
```
Error: not enough memory available
```

**해결**:
- 리소스 제한 조정:
```yaml
claude:
  resources:
    limits:
      memory: "4Gi"
  session:
    pool_size: 3
```

### 디버깅 팁

```bash
# 설정 디버깅 모드
aicli serve --debug-config

# 상세 설정 정보
aicli config show --verbose

# 설정 소스 추적
aicli config trace logging.level
```

## 보안 고려사항

### 민감한 정보 보호

```bash
# 토큰을 파일에서 읽기
export CLAUDE_CODE_OAUTH_TOKEN=$(cat /secure/token.txt)

# 암호화된 설정 사용
aicli config encrypt config.yaml > config.enc

# 런타임에 복호화
aicli serve --encrypted-config config.enc
```

### 권한 최소화

```yaml
security:
  run_as_user: "claude"
  drop_capabilities: true
  readonly_filesystem: true
```

## 성능 튜닝

### CPU 집약적 워크로드

```yaml
claude:
  process:
    max_concurrent: 4  # CPU 코어 수와 동일
  session:
    pool_size: 2      # 적은 세션으로 메모리 절약
```

### 메모리 집약적 워크로드

```yaml
claude:
  process:
    max_concurrent: 20  # 많은 동시 실행
  session:
    pool_size: 10      # 큰 세션 풀
  stream:
    buffer_size: 4096   # 큰 버퍼
```

### 네트워크 집약적 워크로드

```yaml
claude:
  stream:
    backpressure_policy: "block_until_ready"
  error_handling:
    max_retries: 5
    retry_backoff: "exponential"
```

이 설정 가이드를 통해 다양한 환경과 요구사항에 맞는 Claude CLI 통합 설정을 구성할 수 있습니다.