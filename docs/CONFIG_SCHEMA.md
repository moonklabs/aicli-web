# AICode Manager 설정 스키마 문서

## 개요

AICode Manager는 YAML 형식의 설정 파일을 사용하며, 환경 변수를 통한 설정 재정의를 지원합니다. 설정 파일은 기본적으로 `~/.aicli/config.yaml`에 위치합니다.

## 설정 파일 구조

```yaml
# Claude CLI 관련 설정
claude:
  api_key: "your-api-key"              # 필수: Claude API 키
  model: "claude-3-opus-20240229"      # Claude 모델 (기본값: claude-3-opus-20240229)
  temperature: 0.7                     # 생성 온도 (0.0-1.0, 기본값: 0.7)
  timeout: 300                         # 타임아웃 (초, 기본값: 300)
  max_tokens: 100000                   # 최대 토큰 수 (기본값: 100000)
  retry_count: 3                       # 재시도 횟수 (기본값: 3)
  retry_delay: "2s"                    # 재시도 간격 (기본값: 2s)

# 워크스페이스 관련 설정
workspace:
  default_path: "~/.aicli/workspaces"  # 기본 워크스페이스 경로
  auto_sync: true                      # 자동 동기화 활성화 (기본값: true)
  max_projects: 10                     # 최대 동시 프로젝트 수 (1-100, 기본값: 10)
  isolation_mode: "docker"             # 격리 모드: docker, process, none (기본값: docker)
  watch_files: true                    # 파일 감시 활성화 (기본값: true)
  exclude_patterns:                    # 제외 패턴 (glob)
    - "*.tmp"
    - "*.log"
    - ".git/**"
    - "node_modules/**"

# 출력 형식 관련 설정
output:
  format: "table"                      # 출력 형식: table, json, yaml, pretty, plain (기본값: table)
  color_mode: "auto"                   # 색상 모드: auto, always, never (기본값: auto)
  width: 120                           # 출력 너비 (40-300, 기본값: 120)
  verbosity: 1                         # 상세 레벨 (0-3, 기본값: 1)
  show_progress: true                  # 진행 표시기 활성화 (기본값: true)
  show_timestamp: false                # 타임스탬프 표시 (기본값: false)

# 로깅 관련 설정
logging:
  level: "info"                        # 로그 레벨: debug, info, warn, error, fatal (기본값: info)
  file_path: "~/.aicli/logs/aicli.log" # 로그 파일 경로
  max_size: 100                        # 로그 파일 최대 크기 (MB, 기본값: 100)
  max_backups: 5                       # 최대 백업 파일 수 (기본값: 5)
  max_age: 30                          # 최대 보관 일수 (기본값: 30)
  compress: true                       # 로그 압축 활성화 (기본값: true)
  json_format: false                   # JSON 형식 로깅 (기본값: false)

# Docker 관련 설정
docker:
  socket_path: "/var/run/docker.sock"  # Docker 소켓 경로
  default_image: "aicli-workspace:latest" # 기본 Docker 이미지
  memory_limit: 2048                   # 메모리 제한 (MB, 최소 128, 기본값: 2048)
  cpu_limit: 2.0                       # CPU 제한 (최소 0.1, 기본값: 2.0)
  network_mode: "bridge"               # 네트워크 모드: bridge, host, none (기본값: bridge)
  auto_cleanup: true                   # 자동 정리 활성화 (기본값: true)
  container_prefix: "aicli"            # 컨테이너 접두사 (기본값: aicli)

# API 서버 관련 설정
api:
  address: "localhost:8080"            # 리스닝 주소 (기본값: localhost:8080)
  tls_enabled: false                   # TLS 활성화 (기본값: false)
  tls_cert_path: ""                    # TLS 인증서 경로
  tls_key_path: ""                     # TLS 키 경로
  cors_origins:                        # CORS 허용 오리진
    - "http://localhost:3000"
  rate_limit: 100                      # 요청 제한 (분당, 기본값: 100)
  jwt_secret: ""                       # JWT 비밀 키 (최소 32자)
  jwt_expiration: "24h"                # JWT 만료 시간 (기본값: 24h)
```

## 환경 변수 매핑

모든 설정은 환경 변수를 통해 재정의할 수 있습니다. 환경 변수 이름은 `AICLI_` 접두사로 시작하며, 중첩된 설정은 언더스코어(`_`)로 구분합니다.

### Claude 설정
- `AICLI_CLAUDE_API_KEY` → `claude.api_key`
- `AICLI_CLAUDE_MODEL` → `claude.model`
- `AICLI_CLAUDE_TEMPERATURE` → `claude.temperature`
- `AICLI_CLAUDE_TIMEOUT` → `claude.timeout`
- `AICLI_CLAUDE_MAX_TOKENS` → `claude.max_tokens`
- `AICLI_CLAUDE_RETRY_COUNT` → `claude.retry_count`
- `AICLI_CLAUDE_RETRY_DELAY` → `claude.retry_delay`

### 워크스페이스 설정
- `AICLI_WORKSPACE_DEFAULT_PATH` → `workspace.default_path`
- `AICLI_WORKSPACE_AUTO_SYNC` → `workspace.auto_sync`
- `AICLI_WORKSPACE_MAX_PROJECTS` → `workspace.max_projects`
- `AICLI_WORKSPACE_ISOLATION_MODE` → `workspace.isolation_mode`
- `AICLI_WORKSPACE_WATCH_FILES` → `workspace.watch_files`
- `AICLI_WORKSPACE_EXCLUDE_PATTERNS` → `workspace.exclude_patterns` (쉼표로 구분)

### 출력 설정
- `AICLI_OUTPUT_FORMAT` → `output.format`
- `AICLI_OUTPUT_COLOR_MODE` → `output.color_mode`
- `AICLI_OUTPUT_WIDTH` → `output.width`
- `AICLI_OUTPUT_VERBOSITY` → `output.verbosity`
- `AICLI_OUTPUT_SHOW_PROGRESS` → `output.show_progress`
- `AICLI_OUTPUT_SHOW_TIMESTAMP` → `output.show_timestamp`

### 로깅 설정
- `AICLI_LOG_LEVEL` → `logging.level`
- `AICLI_LOG_FILE_PATH` → `logging.file_path`
- `AICLI_LOG_MAX_SIZE` → `logging.max_size`
- `AICLI_LOG_MAX_BACKUPS` → `logging.max_backups`
- `AICLI_LOG_MAX_AGE` → `logging.max_age`
- `AICLI_LOG_COMPRESS` → `logging.compress`
- `AICLI_LOG_JSON_FORMAT` → `logging.json_format`

### Docker 설정
- `AICLI_DOCKER_SOCKET_PATH` → `docker.socket_path`
- `AICLI_DOCKER_DEFAULT_IMAGE` → `docker.default_image`
- `AICLI_DOCKER_MEMORY_LIMIT` → `docker.memory_limit`
- `AICLI_DOCKER_CPU_LIMIT` → `docker.cpu_limit`
- `AICLI_DOCKER_NETWORK_MODE` → `docker.network_mode`
- `AICLI_DOCKER_AUTO_CLEANUP` → `docker.auto_cleanup`
- `AICLI_DOCKER_CONTAINER_PREFIX` → `docker.container_prefix`

### API 설정
- `AICLI_API_ADDRESS` → `api.address`
- `AICLI_API_TLS_ENABLED` → `api.tls_enabled`
- `AICLI_API_TLS_CERT_PATH` → `api.tls_cert_path`
- `AICLI_API_TLS_KEY_PATH` → `api.tls_key_path`
- `AICLI_API_CORS_ORIGINS` → `api.cors_origins` (쉼표로 구분)
- `AICLI_API_RATE_LIMIT` → `api.rate_limit`
- `AICLI_API_JWT_SECRET` → `api.jwt_secret`
- `AICLI_API_JWT_EXPIRATION` → `api.jwt_expiration`

## 설정 우선순위

설정은 다음 순서로 적용됩니다 (높은 우선순위부터):

1. 명령줄 플래그
2. 환경 변수
3. 설정 파일 (`~/.aicli/config.yaml`)
4. 기본값

## 검증 규칙

### 필수 항목
- `claude.api_key`: Claude API 키 (최소 20자)
- `api.jwt_secret`: JWT 비밀 키 (API 서버 사용 시, 최소 32자)

### 값 범위 제한
- `claude.temperature`: 0.0 ~ 1.0
- `claude.timeout`: 1 ~ 3600 (초)
- `claude.max_tokens`: 1 ~ 200000
- `claude.retry_count`: 0 ~ 10
- `workspace.max_projects`: 1 ~ 100
- `output.width`: 40 ~ 300
- `output.verbosity`: 0 ~ 3
- `logging.max_size`: 1 ~ 1000 (MB)
- `logging.max_backups`: 0 ~ 100
- `logging.max_age`: 0 ~ 365 (일)
- `docker.memory_limit`: 최소 128 (MB)
- `docker.cpu_limit`: 최소 0.1
- `api.rate_limit`: 0 ~ 10000

### 열거형 값
- `claude.model`: 
  - `claude-3-opus-20240229`
  - `claude-3-sonnet-20240229`
  - `claude-3-haiku-20240307`
- `workspace.isolation_mode`: `docker`, `process`, `none`
- `output.format`: `table`, `json`, `yaml`, `pretty`, `plain`
- `output.color_mode`: `auto`, `always`, `never`
- `logging.level`: `debug`, `info`, `warn`, `error`, `fatal`
- `docker.network_mode`: `bridge`, `host`, `none`

## 예제 설정

### 최소 설정
```yaml
claude:
  api_key: "sk-ant-api03-..."
```

### 개발 환경 설정
```yaml
claude:
  api_key: "sk-ant-api03-..."
  model: "claude-3-opus-20240229"
  temperature: 0.7

workspace:
  default_path: "~/projects/aicli-workspaces"
  max_projects: 5
  isolation_mode: "docker"

output:
  format: "pretty"
  color_mode: "always"
  verbosity: 2

logging:
  level: "debug"
  json_format: true
```

### 프로덕션 환경 설정
```yaml
claude:
  api_key: "${CLAUDE_API_KEY}"  # 환경 변수에서 읽기
  timeout: 600
  retry_count: 5

workspace:
  max_projects: 20
  isolation_mode: "docker"

api:
  address: "0.0.0.0:8443"
  tls_enabled: true
  tls_cert_path: "/etc/aicli/tls/cert.pem"
  tls_key_path: "/etc/aicli/tls/key.pem"
  rate_limit: 1000
  jwt_secret: "${JWT_SECRET}"  # 환경 변수에서 읽기

logging:
  level: "info"
  max_size: 500
  max_backups: 10
  compress: true
```

## 설정 파일 위치

설정 파일은 다음 위치에서 순서대로 검색됩니다:

1. `--config` 플래그로 지정된 경로
2. `$AICLI_CONFIG_PATH` 환경 변수
3. `~/.aicli/config.yaml` (기본값)
4. `/etc/aicli/config.yaml` (시스템 전역)

## 보안 고려사항

- API 키와 JWT Secret은 절대 설정 파일에 직접 저장하지 마세요
- 환경 변수나 보안 저장소를 사용하세요
- 설정 파일 권한을 `600`으로 설정하세요: `chmod 600 ~/.aicli/config.yaml`
- Git 저장소에 설정 파일을 커밋하지 마세요