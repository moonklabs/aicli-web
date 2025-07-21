# Claude CLI 통합 사용 가이드

## 개요

AICode Manager의 Claude CLI 통합은 Go 네이티브 프로세스 관리를 통해 안정적이고 효율적인 Claude 실행 환경을 제공합니다. 실시간 스트림 처리, 세션 관리, 에러 복구 등의 고급 기능을 포함합니다.

## 빠른 시작

### CLI 사용

```bash
# 단일 프롬프트 실행
aicli claude run "Hello, Claude!" --workspace my-project

# 인터랙티브 채팅
aicli claude chat --system "You are a helpful coding assistant"

# 파일 기반 프롬프트 실행
aicli claude run --file prompt.txt --workspace my-project

# 도구 권한 지정
aicli claude run "Write a Go function" --tools Read,Write,Bash

# 세션 관리
aicli claude session list
aicli claude session show <session-id>
aicli claude session clean  # 비활성 세션 정리
```

### API 사용

```go
// Go 클라이언트 예제
package main

import (
    "context"
    "fmt"
    "log"
    
    "aicli-web/internal/claude"
    "aicli-web/internal/storage"
)

func main() {
    // 스토리지 초기화
    storage, err := storage.New("sqlite", map[string]interface{}{
        "path": "./aicli.db",
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // Claude 세션 매니저 생성
    sessionMgr := claude.NewSessionManager(storage)
    
    // 세션 생성
    session, err := sessionMgr.Create(&claude.SessionConfig{
        WorkspaceID:    "my-project",
        SystemPrompt:   "You are a helpful assistant",
        MaxTurns:       10,
        AllowedTools:   []string{"Read", "Write"},
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // 프롬프트 실행
    ctx := context.Background()
    stream, err := sessionMgr.Execute(ctx, session.ID, "Write a hello world in Go")
    if err != nil {
        log.Fatal(err)
    }
    
    // 스트림 처리
    for msg := range stream {
        fmt.Printf("Type: %s, Content: %s\n", msg.Type, msg.Content)
    }
}
```

## 주요 기능

### 1. 세션 관리

#### 세션 생성 및 설정
```go
config := &claude.SessionConfig{
    WorkspaceID:    "my-project",
    SystemPrompt:   "You are an expert Go developer",
    MaxTurns:       20,
    AllowedTools:   []string{"Read", "Write", "Bash"},
    Timeout:        time.Minute * 5,
    MaxRetries:     3,
}

session, err := sessionManager.Create(config)
```

#### 세션 재사용
- 세션 풀링을 통한 성능 최적화
- 자동 세션 정리 및 재할당
- 세션 상태 추적 (Idle, Running, Closed, Error)

#### 세션 모니터링
```bash
# 활성 세션 목록
aicli claude session list --status active

# 세션 상세 정보
aicli claude session show session-123 --verbose

# 세션 통계
aicli claude stats
```

### 2. 스트림 처리

#### 실시간 출력 스트리밍
```go
// 스트림 핸들러 등록
handler := claude.NewStreamHandler()
handler.RegisterHandler("text", func(msg claude.StreamMessage) {
    fmt.Print(msg.Content)
})

handler.RegisterHandler("tool_use", func(msg claude.StreamMessage) {
    fmt.Printf("Using tool: %s\n", msg.Tool)
})

// 스트림 실행
stream, err := session.StreamWithCallback(prompt, handler)
```

#### 백프레셔 처리
- DropOldest: 오래된 메시지 제거
- DropNewest: 새 메시지 제거 
- BlockUntilReady: 버퍼 공간까지 대기

#### 메시지 라우팅
```go
// 메시지 타입별 처리
router := claude.NewMessageRouter()
router.AddHandler("error", priorityHigh, errorHandler)
router.AddHandler("text", priorityMedium, textHandler)
router.AddHandler("tool_use", priorityLow, toolHandler)
```

### 3. 에러 처리

#### 자동 재시도
```yaml
# config.yaml
claude:
  error_handling:
    max_retries: 3
    retry_backoff: exponential  # linear, exponential
    base_delay: 1s
    max_delay: 30s
```

#### 회로 차단기
```go
// 회로 차단기 설정
breaker := claude.NewCircuitBreaker(claude.CircuitBreakerConfig{
    Threshold:     5,          // 실패 임계값
    HalfOpenDelay: time.Second * 30, // 반열림 대기시간
    TestRequests:  1,          // 테스트 요청 수
})
```

#### 에러 복구
- 프로세스 크래시 자동 감지
- 새 프로세스 생성 및 세션 복구
- 상태 일관성 보장

## 설정 옵션

### 환경 변수

| 변수명 | 설명 | 기본값 | 필수 |
|--------|------|--------|------|
| `CLAUDE_CODE_OAUTH_TOKEN` | Claude OAuth 토큰 | - | ✓ |
| `CLAUDE_API_KEY` | Claude API 키 (대체 인증) | - | ✗ |
| `CLAUDE_MAX_SESSIONS` | 최대 동시 세션 수 | 10 | ✗ |
| `CLAUDE_SESSION_TIMEOUT` | 세션 타임아웃 | 30m | ✗ |
| `CLAUDE_LOG_LEVEL` | 로그 레벨 | info | ✗ |

### 설정 파일 예시

```yaml
# config.yaml
claude:
  # 프로세스 설정
  process:
    max_concurrent: 10
    startup_timeout: 30s
    shutdown_timeout: 10s
    health_check_interval: 30s
  
  # 스트림 설정
  stream:
    buffer_size: 1024
    max_line_size: 1MB
    backpressure_threshold: 0.8
    backpressure_policy: drop_oldest
  
  # 세션 설정
  session:
    pool_size: 5
    reuse_timeout: 10m
    max_idle_time: 30m
    auto_cleanup: true
  
  # 에러 처리
  error_handling:
    max_retries: 3
    retry_backoff: exponential
    circuit_breaker_threshold: 5
    recovery_enabled: true
    
  # 도구 권한
  tools:
    allowed:
      - Read
      - Write
      - Bash
      - Search
    config:
      Bash:
        timeout: 30s
        allowed_commands:
          - ls
          - cat
          - grep
      Write:
        max_file_size: 10MB
        allowed_paths:
          - /workspace
```

## 성능 최적화

### 세션 풀링
```go
// 세션 풀 설정
poolConfig := &claude.SessionPoolConfig{
    InitialSize:    3,
    MaxSize:        10,
    MaxIdleTime:    time.Minute * 30,
    CleanupInterval: time.Minute * 5,
}

pool := claude.NewSessionPool(poolConfig)
```

### 리소스 제한
```yaml
resources:
  limits:
    cpu: 2.0      # CPU 코어 수
    memory: 2Gi   # 메모리
    processes: 50 # 최대 프로세스 수
  requests:
    cpu: 0.5
    memory: 512Mi
```

### 모니터링 메트릭
- `claude_sessions_active`: 활성 세션 수
- `claude_requests_total`: 총 요청 수
- `claude_errors_total`: 총 에러 수
- `claude_response_time`: 응답 시간
- `claude_memory_usage`: 메모리 사용량

## 보안 고려사항

### 토큰 관리
```go
// 안전한 토큰 관리
tokenManager := claude.NewTokenManager(&claude.TokenConfig{
    RefreshThreshold: time.Minute * 30,
    SecureStorage:    true,
    EncryptionKey:    os.Getenv("ENCRYPTION_KEY"),
})
```

### 권한 제한
```yaml
security:
  allowed_hosts:
    - "api.claude.ai"
    - "claude.ai"
  network_isolation: true
  filesystem_restrictions:
    - "/workspace"
    - "/tmp"
```

### 감사 로그
- 모든 Claude 요청 로깅
- 사용자별 활동 추적
- 에러 및 보안 이벤트 기록

## 문제 해결

### 일반적인 문제

#### 1. 세션 시작 실패
```
Error: Failed to start Claude session
```

**원인 및 해결**:
- Claude CLI 설치 확인: `which claude`
- OAuth 토큰 확인: `aicli auth status`
- 권한 확인: `ls -la $(which claude)`

#### 2. 스트림 처리 오류
```
Error: Stream parsing failed
```

**원인 및 해결**:
- 네트워크 연결 확인
- 버퍼 크기 증가
- 타임아웃 값 조정

#### 3. 메모리 부족
```
Error: Out of memory
```

**원인 및 해결**:
- 세션 풀 크기 조정
- 가비지 컬렉션 튜닝
- 리소스 제한 설정

### 디버깅 가이드

#### 로그 활성화
```bash
export CLAUDE_LOG_LEVEL=debug
aicli claude run "test prompt" --debug
```

#### 상태 확인
```bash
# 프로세스 상태
aicli claude status --verbose

# 시스템 리소스
aicli system resources

# 메트릭 확인
curl http://localhost:8080/metrics | grep claude
```

#### 진단 정보 수집
```bash
# 진단 보고서 생성
aicli claude diagnose > debug-report.txt

# 세션 덤프
aicli claude session dump session-123 > session-dump.json
```

## 예제 및 사용 사례

### 1. 코드 생성
```bash
aicli claude run "Write a REST API server in Go with user authentication" \
  --tools Write,Read \
  --system "You are an expert Go developer" \
  --workspace ./my-api-project
```

### 2. 코드 리뷰
```bash
aicli claude run --file review-prompt.txt \
  --tools Read \
  --system "You are a senior code reviewer"
```

### 3. 배치 처리
```bash
#!/bin/bash
# batch_process.sh

for file in *.go; do
  echo "Processing $file..."
  aicli claude run "Add comprehensive comments to this Go file" \
    --tools Read,Write \
    --input "$file" \
    --output "$file.commented"
done
```

### 4. 인터랙티브 개발
```bash
# 지속적인 세션으로 개발 작업
aicli claude chat \
  --session dev-session \
  --system "You are my coding pair partner" \
  --tools Read,Write,Bash
```

## API 참조

자세한 API 문서는 [API 레퍼런스](./api-reference.md)를 참조하세요.

## 추가 리소스

- [설정 가이드](./configuration.md)
- [아키텍처 문서](./architecture.md)
- [트러블슈팅 가이드](./troubleshooting.md)
- [성능 최적화 가이드](./performance.md)