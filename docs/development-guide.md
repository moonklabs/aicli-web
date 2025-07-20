# AICode Manager 개발 가이드

이 문서는 AICode Manager 프로젝트의 상세한 개발 가이드입니다. 프로젝트 아키텍처부터 디버깅, 테스트, 배포까지 개발자가 알아야 할 모든 정보를 담고 있습니다.

## 목차

- [프로젝트 개요](#프로젝트-개요)
- [아키텍처](#아키텍처)
- [개발 환경 설정](#개발-환경-설정)
- [코드 구조](#코드-구조)
- [개발 워크플로우](#개발-워크플로우)
- [디버깅 가이드](#디버깅-가이드)
- [테스트 전략](#테스트-전략)
- [성능 최적화](#성능-최적화)
- [배포 가이드](#배포-가이드)
- [문제 해결](#문제-해결)
- [개발 팁](#개발-팁)

## 프로젝트 개요

AICode Manager는 Claude CLI를 웹 플랫폼으로 관리하는 로컬 우선 시스템입니다. Go 언어로 개발되었으며, 다음과 같은 핵심 기능을 제공합니다:

### 핵심 기능

- **멀티 프로젝트 관리**: 여러 프로젝트 동시 실행 및 관리
- **격리된 실행 환경**: Docker 컨테이너를 통한 프로젝트별 독립 환경
- **실시간 모니터링**: WebSocket을 통한 실시간 로그 스트리밍
- **Git 워크플로우 통합**: 자동 브랜치 생성, 커밋, PR 관리
- **RESTful API**: 프로그래밍 가능한 인터페이스

### 기술 스택

- **언어**: Go 1.21+
- **웹 프레임워크**: Gin
- **CLI 프레임워크**: Cobra
- **데이터베이스**: SQLite (임베디드)
- **컨테이너**: Docker SDK
- **실시간 통신**: WebSocket/SSE
- **빌드 도구**: Make
- **테스트**: testify

## 아키텍처

### 전체 아키텍처

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Web Client    │    │   CLI Client    │    │  External API   │
│                 │    │                 │    │                 │
└─────────┬───────┘    └─────────┬───────┘    └─────────┬───────┘
          │                      │                      │
          │ HTTP/WebSocket       │ Commands             │ REST API
          │                      │                      │
    ┌─────┴──────────────────────┴──────────────────────┴─────┐
    │                API Gateway                              │
    │              (Gin HTTP Server)                         │
    └─────────────────────┬───────────────────────────────────┘
                          │
    ┌─────────────────────┴───────────────────────────────────┐
    │                Core Services                            │
    │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │
    │  │ Workspace   │  │   Task      │  │   Config    │     │
    │  │ Manager     │  │ Manager     │  │ Manager     │     │
    │  └─────────────┘  └─────────────┘  └─────────────┘     │
    └─────────────────────┬───────────────────────────────────┘
                          │
    ┌─────────────────────┴───────────────────────────────────┐
    │              Infrastructure Layer                       │
    │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │
    │  │   Docker    │  │   Storage   │  │   Claude    │     │
    │  │    SDK      │  │   Layer     │  │ CLI Wrapper │     │
    │  └─────────────┘  └─────────────┘  └─────────────┘     │
    └─────────────────────────────────────────────────────────┘
```

### 컴포넌트 설명

#### 1. API Gateway (Gin HTTP Server)
- HTTP 요청 라우팅 및 미들웨어 처리
- WebSocket 연결 관리
- 인증/인가 처리
- 요청/응답 로깅

#### 2. Core Services
- **Workspace Manager**: 프로젝트 워크스페이스 생명주기 관리
- **Task Manager**: 태스크 실행 및 상태 관리
- **Config Manager**: 설정 관리 및 변경 감지

#### 3. Infrastructure Layer
- **Docker SDK**: 컨테이너 생명주기 관리
- **Storage Layer**: 데이터 영속성 및 캐싱
- **Claude CLI Wrapper**: Claude CLI 프로세스 관리

### 디렉토리 구조

```
aicli-web/
├── cmd/                    # 실행 가능한 프로그램
│   ├── aicli/             # CLI 도구 진입점
│   │   └── main.go
│   └── api/               # API 서버 진입점
│       └── main.go
├── internal/              # 내부 패키지 (외부 접근 불가)
│   ├── cli/               # CLI 명령어 구현
│   │   ├── commands/      # 개별 명령어
│   │   └── root.go        # 루트 명령어
│   ├── server/            # API 서버 구현
│   │   ├── handlers/      # HTTP 핸들러
│   │   ├── middleware/    # 미들웨어
│   │   └── router.go      # 라우터 설정
│   ├── claude/            # Claude CLI 래퍼
│   │   ├── manager.go     # 매니저 인터페이스
│   │   └── process.go     # 프로세스 관리
│   ├── docker/            # Docker SDK 통합
│   │   ├── client.go      # Docker 클라이언트
│   │   └── container.go   # 컨테이너 관리
│   ├── storage/           # 데이터 저장소
│   │   ├── sqlite.go      # SQLite 구현
│   │   └── interface.go   # 저장소 인터페이스
│   ├── models/            # 도메인 모델
│   │   ├── workspace.go   # 워크스페이스 모델
│   │   └── task.go        # 태스크 모델
│   └── config/            # 설정 관리
│       ├── config.go      # 설정 구조체
│       └── loader.go      # 설정 로더
├── pkg/                   # 외부 공개 패키지
│   ├── version/           # 버전 정보
│   └── utils/             # 공용 유틸리티
└── test/                  # 테스트 파일
    ├── integration/       # 통합 테스트
    └── e2e/               # E2E 테스트
```

## 개발 환경 설정

### 사전 요구사항

- Go 1.21 이상
- Docker 20.10 이상
- Make
- Git

### 로컬 개발 환경

1. **저장소 클론**
   ```bash
   git clone https://github.com/drumcap/aicli-web.git
   cd aicli-web
   ```

2. **의존성 설치**
   ```bash
   make deps
   ```

3. **개발 도구 설치**
   ```bash
   # golangci-lint 설치
   go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
   
   # Delve 디버거 설치
   go install github.com/go-delve/delve/cmd/dlv@latest
   
   # Air (hot reload) 설치
   go install github.com/cosmtrek/air@latest
   
   # pre-commit 설치
   pip install pre-commit
   pre-commit install
   ```

4. **환경 변수 설정**
   ```bash
   # .env 파일 생성
   cp .env.example .env
   
   # 필요한 환경 변수 설정
   export CLAUDE_API_KEY="your-api-key"
   export AICLI_LOG_LEVEL="debug"
   export AICLI_DATA_DIR="./data"
   ```

### Docker 개발 환경

1. **개발 환경 시작**
   ```bash
   make docker-dev
   ```

2. **서비스별 로그 확인**
   ```bash
   # 모든 서비스 로그
   make docker-dev-logs
   
   # API 서버 로그만
   docker-compose logs -f api-dev
   ```

3. **개발 환경 정리**
   ```bash
   make docker-dev-down
   ```

### VS Code 설정

프로젝트에는 `.vscode/settings.json` 파일이 포함되어 있어 다음과 같은 설정이 자동으로 적용됩니다:

```json
{
    "go.toolsManagement.checkForUpdates": "local",
    "go.useLanguageServer": true,
    "go.lintOnSave": "workspace",
    "go.formatTool": "goimports",
    "go.lintTool": "golangci-lint",
    "go.buildOnSave": "workspace",
    "go.testOnSave": false,
    "editor.formatOnSave": true,
    "editor.codeActionsOnSave": {
        "source.organizeImports": true
    }
}
```

권장 확장 프로그램:
- Go
- Go Outliner
- REST Client
- GitLens
- Docker
- Todo Tree

## 코드 구조

### 패키지 설계 원칙

1. **Clean Architecture**: 의존성 역전 원칙 준수
2. **Domain-Driven Design**: 도메인 모델 중심 설계
3. **Interface Segregation**: 작은 인터페이스 선호
4. **Dependency Injection**: 생성자 주입 패턴

### 주요 인터페이스

```go
// WorkspaceManager는 워크스페이스 관리 인터페이스입니다.
type WorkspaceManager interface {
    Create(ctx context.Context, workspace *models.Workspace) error
    Get(ctx context.Context, id string) (*models.Workspace, error)
    List(ctx context.Context) ([]*models.Workspace, error)
    Update(ctx context.Context, workspace *models.Workspace) error
    Delete(ctx context.Context, id string) error
}

// TaskManager는 태스크 관리 인터페이스입니다.
type TaskManager interface {
    Start(ctx context.Context, task *models.Task) error
    Stop(ctx context.Context, taskID string) error
    GetStatus(ctx context.Context, taskID string) (*models.TaskStatus, error)
    GetLogs(ctx context.Context, taskID string) ([]string, error)
}

// Storage는 데이터 저장소 인터페이스입니다.
type Storage interface {
    Save(ctx context.Context, key string, value interface{}) error
    Load(ctx context.Context, key string, value interface{}) error
    Delete(ctx context.Context, key string) error
    List(ctx context.Context, prefix string) ([]string, error)
}
```

### 에러 처리 패턴

```go
// 커스텀 에러 타입 정의
type ErrorCode string

const (
    ErrCodeNotFound     ErrorCode = "NOT_FOUND"
    ErrCodeInvalidInput ErrorCode = "INVALID_INPUT"
    ErrCodeInternal     ErrorCode = "INTERNAL_ERROR"
)

type AppError struct {
    Code    ErrorCode `json:"code"`
    Message string    `json:"message"`
    Cause   error     `json:"-"`
}

func (e *AppError) Error() string {
    if e.Cause != nil {
        return fmt.Sprintf("%s: %s (caused by: %v)", e.Code, e.Message, e.Cause)
    }
    return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// 에러 생성 헬퍼 함수
func NewNotFoundError(resource string) *AppError {
    return &AppError{
        Code:    ErrCodeNotFound,
        Message: fmt.Sprintf("%s를 찾을 수 없습니다", resource),
    }
}

func WrapError(err error, code ErrorCode, message string) *AppError {
    return &AppError{
        Code:    code,
        Message: message,
        Cause:   err,
    }
}
```

## 개발 워크플로우

### 일반적인 개발 플로우

1. **기능 브랜치 생성**
   ```bash
   git checkout -b feature/new-feature
   ```

2. **개발 서버 시작**
   ```bash
   # Hot reload 개발
   make dev
   
   # 또는 개별 서비스
   make run-api
   make run-cli
   ```

3. **코드 작성 및 테스트**
   ```bash
   # 코드 변경 후 자동 테스트
   make test-watch
   
   # 특정 패키지 테스트
   go test -v ./internal/models/...
   ```

4. **품질 검사**
   ```bash
   # 모든 품질 검사
   make check
   
   # 개별 검사
   make lint
   make fmt
   make vet
   ```

5. **커밋 및 푸시**
   ```bash
   git add .
   git commit -m "feat: 새로운 기능 추가"
   git push origin feature/new-feature
   ```

### 빌드 시스템

Makefile을 통해 다양한 빌드 옵션을 제공합니다:

```bash
# 개발용 빌드
make build-dev

# 프로덕션 빌드
make build

# 멀티 플랫폼 빌드
make build-all

# Docker 이미지 빌드
make docker

# 릴리스 빌드
make release
```

각 빌드는 다음과 같은 최적화를 적용합니다:

- **Static Linking**: CGO_ENABLED=0으로 정적 바이너리 생성
- **Size Optimization**: -ldflags="-s -w"로 디버그 정보 제거
- **Version Injection**: 빌드 시점의 버전 정보 주입

## 디버깅 가이드

### Delve 디버거 사용

1. **기본 디버깅**
   ```bash
   # API 서버 디버깅
   dlv debug ./cmd/api
   
   # CLI 도구 디버깅
   dlv debug ./cmd/aicli -- workspace list
   ```

2. **원격 디버깅**
   ```bash
   # 디버그 모드로 서버 실행
   dlv debug --headless --listen=:2345 --api-version=2 ./cmd/api
   
   # VS Code에서 연결
   ```

3. **테스트 디버깅**
   ```bash
   # 특정 테스트 디버깅
   dlv test ./internal/models -- -test.run TestWorkspace
   ```

### VS Code 디버깅 설정

`.vscode/launch.json`:

```json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Debug API Server",
            "type": "go",
            "request": "launch",
            "mode": "auto",
            "program": "${workspaceFolder}/cmd/api",
            "env": {
                "AICLI_LOG_LEVEL": "debug"
            },
            "args": []
        },
        {
            "name": "Debug CLI",
            "type": "go",
            "request": "launch",
            "mode": "auto",
            "program": "${workspaceFolder}/cmd/aicli",
            "args": ["workspace", "list"]
        },
        {
            "name": "Attach to running process",
            "type": "go",
            "request": "attach",
            "mode": "remote",
            "remotePath": "${workspaceFolder}",
            "port": 2345,
            "host": "127.0.0.1"
        }
    ]
}
```

### 로깅 및 추적

```go
// 구조화된 로깅 사용
import (
    "go.uber.org/zap"
    "context"
)

// 컨텍스트에서 로거 가져오기
func getLogger(ctx context.Context) *zap.Logger {
    if logger := ctx.Value("logger"); logger != nil {
        return logger.(*zap.Logger)
    }
    return zap.L() // 글로벌 로거
}

// 사용 예제
func (w *WorkspaceManager) Create(ctx context.Context, workspace *models.Workspace) error {
    logger := getLogger(ctx).With(
        zap.String("operation", "workspace.create"),
        zap.String("workspace_id", workspace.ID),
    )
    
    logger.Info("워크스페이스 생성 시작")
    
    if err := w.validate(workspace); err != nil {
        logger.Error("워크스페이스 검증 실패", zap.Error(err))
        return err
    }
    
    logger.Info("워크스페이스 생성 완료")
    return nil
}
```

### Docker 환경 디버깅

```bash
# 컨테이너 내부 접속
docker-compose exec api-dev sh

# 컨테이너 로그 확인
docker-compose logs -f api-dev

# 디버그 모드로 실행
make docker-dev-debug
```

## 테스트 전략

### 테스트 피라미드

```
    ┌─────────────┐
    │     E2E     │  (소수: 5-10%)
    │   Tests     │
    ├─────────────┤
    │ Integration │  (중간: 20-30%)
    │   Tests     │
    ├─────────────┤
    │    Unit     │  (다수: 60-70%)
    │   Tests     │
    └─────────────┘
```

### 단위 테스트

```go
// 테스트 파일 예제: internal/models/workspace_test.go
package models

import (
    "context"
    "testing"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestWorkspace_Validate(t *testing.T) {
    tests := []struct {
        name        string
        workspace   *Workspace
        wantErr     bool
        expectedErr string
    }{
        {
            name: "유효한 워크스페이스",
            workspace: &Workspace{
                Name: "test-workspace",
                Path: "/tmp/test",
            },
            wantErr: false,
        },
        {
            name: "빈 이름",
            workspace: &Workspace{
                Name: "",
                Path: "/tmp/test",
            },
            wantErr:     true,
            expectedErr: "워크스페이스 이름은 필수입니다",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.workspace.Validate()
            
            if tt.wantErr {
                require.Error(t, err)
                assert.Contains(t, err.Error(), tt.expectedErr)
            } else {
                require.NoError(t, err)
            }
        })
    }
}
```

### 통합 테스트

```go
// test/integration/api_test.go
package integration

import (
    "context"
    "net/http"
    "testing"
    
    "github.com/stretchr/testify/suite"
)

type APITestSuite struct {
    suite.Suite
    server   *http.Server
    client   *http.Client
    cleanup  func()
}

func (s *APITestSuite) SetupSuite() {
    // 테스트 서버 시작
    s.server, s.cleanup = startTestServer()
    s.client = &http.Client{}
}

func (s *APITestSuite) TearDownSuite() {
    s.cleanup()
}

func (s *APITestSuite) TestWorkspaceAPI() {
    // POST /api/v1/workspaces
    workspace := map[string]interface{}{
        "name": "test-workspace",
        "path": "/tmp/test",
    }
    
    resp, err := s.postJSON("/api/v1/workspaces", workspace)
    s.Require().NoError(err)
    s.Equal(http.StatusCreated, resp.StatusCode)
    
    // GET /api/v1/workspaces
    resp, err = s.client.Get(s.baseURL() + "/api/v1/workspaces")
    s.Require().NoError(err)
    s.Equal(http.StatusOK, resp.StatusCode)
}

func TestAPITestSuite(t *testing.T) {
    suite.Run(t, new(APITestSuite))
}
```

### E2E 테스트

```go
// test/e2e/cli_test.go
package e2e

import (
    "os/exec"
    "testing"
    
    "github.com/stretchr/testify/assert"
)

func TestCLIWorkflow(t *testing.T) {
    // 바이너리 빌드
    cmd := exec.Command("make", "build")
    err := cmd.Run()
    assert.NoError(t, err)
    
    // 워크스페이스 생성
    cmd = exec.Command("./build/aicli", "workspace", "create", "test-workspace", "--path", "/tmp/test")
    output, err := cmd.CombinedOutput()
    assert.NoError(t, err)
    assert.Contains(t, string(output), "워크스페이스가 생성되었습니다")
    
    // 워크스페이스 목록 확인
    cmd = exec.Command("./build/aicli", "workspace", "list")
    output, err = cmd.CombinedOutput()
    assert.NoError(t, err)
    assert.Contains(t, string(output), "test-workspace")
    
    // 정리
    cmd = exec.Command("./build/aicli", "workspace", "delete", "test-workspace")
    err = cmd.Run()
    assert.NoError(t, err)
}
```

### 테스트 실행

```bash
# 모든 테스트
make test

# 단위 테스트만
make test-unit

# 통합 테스트만
make test-integration

# E2E 테스트만
make test-e2e

# 커버리지 리포트
make test-coverage

# 테스트 감시 모드
make test-watch

# 벤치마크 테스트
make test-bench
```

### 목(Mock) 사용

```go
// internal/testutil/mocks.go
//go:generate mockery --name=WorkspaceManager --dir=../models --output=./mocks

type MockWorkspaceManager struct {
    mock.Mock
}

func (m *MockWorkspaceManager) Create(ctx context.Context, workspace *models.Workspace) error {
    args := m.Called(ctx, workspace)
    return args.Error(0)
}

// 테스트에서 사용
func TestWorkspaceService_Create(t *testing.T) {
    mockManager := &testutil.MockWorkspaceManager{}
    service := NewWorkspaceService(mockManager)
    
    workspace := &models.Workspace{Name: "test"}
    mockManager.On("Create", mock.Anything, workspace).Return(nil)
    
    err := service.Create(context.Background(), workspace)
    assert.NoError(t, err)
    mockManager.AssertExpectations(t)
}
```

## 성능 최적화

### 프로파일링

1. **CPU 프로파일링**
   ```bash
   # pprof 서버 시작 (개발 모드에서만)
   go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30
   
   # 프로파일 분석
   (pprof) top
   (pprof) list main.processRequest
   (pprof) web
   ```

2. **메모리 프로파일링**
   ```bash
   go tool pprof http://localhost:6060/debug/pprof/heap
   ```

3. **고루틴 분석**
   ```bash
   go tool pprof http://localhost:6060/debug/pprof/goroutine
   ```

### 벤치마크 테스트

```go
// internal/models/workspace_bench_test.go
func BenchmarkWorkspace_Validate(b *testing.B) {
    workspace := &Workspace{
        Name: "test-workspace",
        Path: "/tmp/test",
    }
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = workspace.Validate()
    }
}

func BenchmarkWorkspaceManager_Create(b *testing.B) {
    manager := NewWorkspaceManager(storage)
    workspace := &Workspace{
        Name: "test-workspace",
        Path: "/tmp/test",
    }
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = manager.Create(context.Background(), workspace)
    }
}
```

### 성능 최적화 팁

1. **메모리 할당 최소화**
   ```go
   // 나쁜 예: 불필요한 문자열 연결
   result := ""
   for _, item := range items {
       result += item + ","
   }
   
   // 좋은 예: strings.Builder 사용
   var builder strings.Builder
   for _, item := range items {
       builder.WriteString(item)
       builder.WriteString(",")
   }
   result := builder.String()
   ```

2. **고루틴 풀 사용**
   ```go
   type WorkerPool struct {
       workers chan chan WorkRequest
       quit    chan bool
   }
   
   func (p *WorkerPool) Start() {
       for i := 0; i < p.maxWorkers; i++ {
           worker := NewWorker(p.workers)
           worker.Start()
       }
   }
   ```

3. **캐싱 전략**
   ```go
   type CachedWorkspaceManager struct {
       cache   map[string]*models.Workspace
       manager WorkspaceManager
       mu      sync.RWMutex
   }
   
   func (c *CachedWorkspaceManager) Get(ctx context.Context, id string) (*models.Workspace, error) {
       c.mu.RLock()
       if workspace, ok := c.cache[id]; ok {
           c.mu.RUnlock()
           return workspace, nil
       }
       c.mu.RUnlock()
       
       workspace, err := c.manager.Get(ctx, id)
       if err != nil {
           return nil, err
       }
       
       c.mu.Lock()
       c.cache[id] = workspace
       c.mu.Unlock()
       
       return workspace, nil
   }
   ```

## 배포 가이드

### 로컬 배포

```bash
# 바이너리 빌드
make build

# 시스템에 설치
sudo cp build/aicli /usr/local/bin/
sudo cp build/aicli-api /usr/local/bin/

# 서비스 등록 (systemd)
sudo cp deployments/systemd/aicli-api.service /etc/systemd/system/
sudo systemctl enable aicli-api
sudo systemctl start aicli-api
```

### Docker 배포

```bash
# 이미지 빌드
make docker

# 컨테이너 실행
docker run -d \
  --name aicli-api \
  -p 8080:8080 \
  -v /var/lib/aicli:/data \
  -e CLAUDE_API_KEY="your-api-key" \
  aicli-web:latest

# Docker Compose 사용
docker-compose -f docker-compose.prod.yml up -d
```

### 쿠버네티스 배포

```yaml
# deployments/k8s/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: aicli-api
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
      - name: aicli-api
        image: drumcap/aicli-web:v0.1.0
        ports:
        - containerPort: 8080
        env:
        - name: CLAUDE_API_KEY
          valueFrom:
            secretKeyRef:
              name: aicli-secrets
              key: claude-api-key
```

### 환경별 설정

```yaml
# configs/production.yaml
server:
  port: 8080
  host: "0.0.0.0"
  
database:
  path: "/data/aicli.db"
  
logging:
  level: "info"
  format: "json"
  
claude:
  timeout: "30s"
  retries: 3
```

## 문제 해결

### 일반적인 문제들

#### 1. Go 모듈 문제

**문제**: `go mod download` 실패
```bash
go: module github.com/drumcap/aicli-web: reading at revision v0.1.0: unknown revision v0.1.0
```

**해결**:
```bash
# 모듈 캐시 정리
go clean -modcache

# 의존성 다시 다운로드
go mod download

# go.sum 재생성
go mod tidy
```

#### 2. Docker 빌드 실패

**문제**: Docker 이미지 빌드 시 Go 모듈 에러

**해결**:
```dockerfile
# Dockerfile에서 모듈 다운로드 단계 분리
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build ...
```

#### 3. 테스트 실패

**문제**: 경쟁 상태(Race Condition) 감지

**해결**:
```bash
# 경쟁 상태 감지
go test -race ./...

# 동기화 추가
var mu sync.RWMutex
mu.Lock()
defer mu.Unlock()
```

#### 4. 메모리 누수

**문제**: 장시간 실행 시 메모리 사용량 증가

**해결**:
```bash
# 메모리 프로파일링
go tool pprof http://localhost:6060/debug/pprof/heap

# 고루틴 누수 확인
go tool pprof http://localhost:6060/debug/pprof/goroutine
```

### 디버깅 체크리스트

- [ ] 로그 레벨을 debug로 설정했는가?
- [ ] 환경 변수가 올바르게 설정되었는가?
- [ ] 네트워크 연결이 가능한가?
- [ ] 파일 권한이 올바른가?
- [ ] 의존성 버전이 호환되는가?
- [ ] 고루틴이 정상적으로 종료되는가?
- [ ] 리소스가 정리되고 있는가?

## 개발 팁

### 생산성 향상 팁

1. **개발 환경 스크립트**
   ```bash
   # scripts/dev-setup.sh
   #!/bin/bash
   
   # 개발 환경 한 번에 설정
   make deps
   make pre-commit-install
   make docker-dev
   
   echo "개발 환경 설정 완료!"
   echo "API 서버: http://localhost:8080"
   echo "pprof: http://localhost:6060/debug/pprof"
   ```

2. **유용한 별칭**
   ```bash
   # ~/.bashrc 또는 ~/.zshrc
   alias aicli-dev="cd ~/dev/aicli-web && make dev"
   alias aicli-test="cd ~/dev/aicli-web && make test-watch"
   alias aicli-logs="cd ~/dev/aicli-web && make docker-dev-logs"
   ```

3. **Git 훅 활용**
   ```bash
   # .git/hooks/pre-push
   #!/bin/bash
   make test
   make lint
   ```

### 코딩 컨벤션

1. **네이밍**
   - 패키지: 소문자, 단수형 (models, storage)
   - 변수: camelCase (userName, configPath)
   - 상수: UPPER_SNAKE_CASE (MAX_RETRIES)
   - 인터페이스: -er 접미사 (WorkspaceManager)

2. **주석**
   ```go
   // Package models는 도메인 모델을 정의합니다.
   package models
   
   // Workspace는 프로젝트 작업 공간을 나타냅니다.
   type Workspace struct {
       ID   string `json:"id"`
       Name string `json:"name"`
   }
   
   // Create는 새로운 워크스페이스를 생성합니다.
   // 이름이 중복되면 ErrDuplicateName을 반환합니다.
   func (w *Workspace) Create(ctx context.Context) error {
       // 구현...
   }
   ```

3. **에러 처리**
   ```go
   // 에러는 항상 명시적으로 처리
   if err != nil {
       return fmt.Errorf("워크스페이스 생성 실패: %w", err)
   }
   
   // 컨텍스트 취소 확인
   select {
   case <-ctx.Done():
       return ctx.Err()
   default:
   }
   ```

### 유용한 도구

1. **goimports**: import 자동 정리
2. **golangci-lint**: 종합 린터
3. **air**: hot reload
4. **dlv**: 디버거
5. **pprof**: 프로파일러
6. **go-swagger**: API 문서 생성
7. **mockery**: 모킹 도구

### 개발 프로세스 자동화

```bash
# scripts/quick-commit.sh
#!/bin/bash

# 빠른 커밋 스크립트
make fmt
make lint
make test-unit

if [ $? -eq 0 ]; then
    git add .
    git commit -m "$1"
    echo "커밋 완료: $1"
else
    echo "테스트 실패. 커밋이 취소되었습니다."
    exit 1
fi
```

---

## 참고 자료

- [Go 공식 문서](https://golang.org/doc/)
- [Gin 프레임워크](https://gin-gonic.com/)
- [Cobra CLI](https://cobra.dev/)
- [Docker SDK for Go](https://docs.docker.com/engine/api/sdk/)
- [testify 테스팅 프레임워크](https://github.com/stretchr/testify)

## 기여하기

이 개발 가이드의 개선사항이나 추가할 내용이 있다면 언제든 Pull Request를 보내주세요. 모든 기여를 환영합니다!

---

**마지막 업데이트**: 2025-07-21  
**버전**: 1.0