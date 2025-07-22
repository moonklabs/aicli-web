# T02_S01_M04_Docker_SDK_기본_클라이언트_구현

**태스크 ID**: T02_S01_M04  
**제목**: Docker SDK 기본 클라이언트 구현  
**설명**: Go Docker SDK를 사용하여 기본적인 Docker 클라이언트 및 설정 구현  
**우선순위**: 높음  
**복잡도**: 보통  
**예상 소요시간**: 4-6시간  
**상태**: 완료  
**시작 시간**: 2025-07-22 20:30  
**완료 시간**: 2025-07-22 21:00  
**실제 소요시간**: 30분  

## 📋 작업 개요

Go의 공식 Docker SDK를 통해 기본적인 Docker 클라이언트를 구현하고, 네트워크 설정 및 연결 관리 기능을 추가합니다. 향후 컬테이너 생성 및 관리를 위한 기초를 마련합니다.

## 🎯 목표

1. **Docker 클라이언트 초기화**: Docker daemon과의 연결 설정
2. **네트워크 관리**: aicli 전용 Docker 네트워크 생성 및 관리
3. **설정 관리**: 보안, 리소스 제한 등 기본 설정
4. **헬스체크 시스템**: Docker daemon 상태 모니터링

## 📂 코드베이스 분석

### 현재 상태
```
internal/docker/
├── doc.go          # 패키지 설명만 존재
└── (기타 구현 파일 없음)
```

### 참고 자료
- `/docs/cli-design/docker-integration.md` - 상세한 설계 문서 존재
- Docker SDK 사용법 및 베스트 프랙티스 가이드 포함

## 🛠️ 기술 가이드

### 1. Docker 클라이언트 구조

```go
// internal/docker/client.go
package docker

import (
    "context"
    "fmt"
    "time"
    
    "github.com/docker/docker/client"
    "github.com/docker/docker/api/types"
)

type Client struct {
    cli         *client.Client
    config      *Config
    networkID   string
    labelPrefix string
}

type Config struct {
    // 연결 설정
    Host        string        `yaml:"host" json:"host"`
    Version     string        `yaml:"version" json:"version"`
    Timeout     time.Duration `yaml:"timeout" json:"timeout"`
    
    // 기본값
    DefaultImage string   `yaml:"default_image" json:"default_image"`
    DefaultShell []string `yaml:"default_shell" json:"default_shell"`
    NetworkName  string   `yaml:"network_name" json:"network_name"`
    
    // 리소스 제한
    CPULimit    float64 `yaml:"cpu_limit" json:"cpu_limit"`
    MemoryLimit int64   `yaml:"memory_limit" json:"memory_limit"`
    
    // 보안 설정
    Privileged   bool     `yaml:"privileged" json:"privileged"`
    ReadOnly     bool     `yaml:"read_only" json:"read_only"`
    SecurityOpts []string `yaml:"security_opts" json:"security_opts"`
}

func NewClient(config *Config) (*Client, error) {
    if config == nil {
        config = DefaultConfig()
    }
    
    // Docker 클라이언트 생성
    cli, err := client.NewClientWithOpts(
        client.WithHost(config.Host),
        client.WithVersion(config.Version),
        client.WithTimeout(config.Timeout),
    )
    if err != nil {
        return nil, fmt.Errorf("create docker client: %w", err)
    }
    
    // 연결 테스트
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    if _, err := cli.Ping(ctx); err != nil {
        return nil, fmt.Errorf("ping docker daemon: %w", err)
    }
    
    dockerClient := &Client{
        cli:         cli,
        config:      config,
        labelPrefix: "aicli",
    }
    
    // 네트워크 설정
    if err := dockerClient.setupNetwork(context.Background()); err != nil {
        return nil, fmt.Errorf("setup network: %w", err)
    }
    
    return dockerClient, nil
}

func DefaultConfig() *Config {
    return &Config{
        Host:         client.DefaultDockerHost,
        Version:      "1.41", // Docker API 1.41
        Timeout:      30 * time.Second,
        DefaultImage: "alpine:latest",
        DefaultShell: []string{"/bin/sh"},
        NetworkName:  "aicli-network",
        CPULimit:     1.0, // 1 CPU
        MemoryLimit:  512 * 1024 * 1024, // 512MB
        Privileged:   false,
        ReadOnly:     true,
        SecurityOpts: []string{"no-new-privileges:true"},
    }
}
```

### 2. 네트워크 관리

```go
func (c *Client) setupNetwork(ctx context.Context) error {
    // 기존 네트워크 확인
    networks, err := c.cli.NetworkList(ctx, types.NetworkListOptions{})
    if err != nil {
        return fmt.Errorf("list networks: %w", err)
    }
    
    for _, network := range networks {
        if network.Name == c.config.NetworkName {
            c.networkID = network.ID
            return nil
        }
    }
    
    // 새 네트워크 생성
    resp, err := c.cli.NetworkCreate(ctx, c.config.NetworkName, types.NetworkCreate{
        Driver:     "bridge",
        Attachable: true,
        Internal:   false, // 외부 인터넷 접근 허용
        Labels: map[string]string{
            c.labelKey("managed"): "true",
            c.labelKey("created"): time.Now().Format(time.RFC3339),
        },
    })
    if err != nil {
        return fmt.Errorf("create network: %w", err)
    }
    
    c.networkID = resp.ID
    return nil
}

func (c *Client) labelKey(key string) string {
    return fmt.Sprintf("%s.%s", c.labelPrefix, key)
}

func (c *Client) GetNetworkID() string {
    return c.networkID
}

func (c *Client) GetConfig() *Config {
    return c.config
}
```

### 3. 헬스체크 시스템

```go
// internal/docker/health.go
package docker

import (
    "context"
    "time"
)

type HealthChecker struct {
    client   *Client
    interval time.Duration
}

func NewHealthChecker(client *Client, interval time.Duration) *HealthChecker {
    if interval == 0 {
        interval = 30 * time.Second
    }
    
    return &HealthChecker{
        client:   client,
        interval: interval,
    }
}

func (h *HealthChecker) CheckDaemon(ctx context.Context) error {
    _, err := h.client.cli.Ping(ctx)
    return err
}

func (h *HealthChecker) StartMonitoring(ctx context.Context, callback func(error)) {
    ticker := time.NewTicker(h.interval)
    go func() {
        defer ticker.Stop()
        
        for {
            select {
            case <-ctx.Done():
                return
            case <-ticker.C:
                err := h.CheckDaemon(ctx)
                if callback != nil {
                    callback(err)
                }
            }
        }
    )()
}

// 시스템 정보 조회
func (h *HealthChecker) GetSystemInfo(ctx context.Context) (*types.Info, error) {
    return h.client.cli.Info(ctx)
}

// Docker 버전 정보
func (h *HealthChecker) GetVersion(ctx context.Context) (types.Version, error) {
    return h.client.cli.ServerVersion(ctx)
}
```

### 4. 공유 유틸리티

```go
// internal/docker/utils.go
package docker

import (
    "fmt"
    "strings"
)

// 레이블 유틸리티
func (c *Client) WorkspaceLabels(workspaceID, name string) map[string]string {
    return map[string]string{
        c.labelKey("managed"):      "true",
        c.labelKey("workspace.id"): workspaceID,
        c.labelKey("workspace.name"): name,
        c.labelKey("created"):      time.Now().Format(time.RFC3339),
    }
}

// 이미지 태그 생성
func (c *Client) GenerateImageTag(workspaceID string) string {
    return fmt.Sprintf("aicli-workspace:%s", workspaceID)
}

// 컬테이너 이름 생성
func (c *Client) GenerateContainerName(workspaceID string) string {
    return fmt.Sprintf("workspace_%s", workspaceID)
}

// 안전한 이름 만들기
func SanitizeName(name string) string {
    // Docker 네이밍 규칙에 맞게 정리
    name = strings.ToLower(name)
    name = strings.ReplaceAll(name, " ", "-")
    name = strings.ReplaceAll(name, "_", "-")
    
    // 허용된 문자만 유지
    var result strings.Builder
    for _, char := range name {
        if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-' {
            result.WriteRune(char)
        }
    }
    
    return result.String()
}
```

### 5. 에러 처리 및 로깅

```go
// internal/docker/errors.go
package docker

import "errors"

var (
    ErrDockerNotAvailable = errors.New("docker daemon not available")
    ErrNetworkNotFound    = errors.New("docker network not found")
    ErrImageNotFound      = errors.New("docker image not found")
    ErrContainerNotFound  = errors.New("container not found")
    ErrInvalidConfig      = errors.New("invalid docker configuration")
)

func IsDockerError(err error) bool {
    if err == nil {
        return false
    }
    
    // Docker SDK에서 발생하는 에러 타입 확인
    return strings.Contains(err.Error(), "docker") ||
           strings.Contains(err.Error(), "container") ||
           strings.Contains(err.Error(), "daemon")
}
```

## ✅ 완료 기준

### 기능적 요구사항
- [ ] Docker 클라이언트 초기화 및 연결 테스트
- [ ] aicli 전용 Docker 네트워크 생성 및 관리
- [ ] 기본 설정 및 리소스 제한 설정
- [ ] 헬스체크 시스템 및 모니터링
- [ ] 공유 유틸리티 함수 구현

### 비기능적 요구사항
- [ ] Docker daemon 연결 실패 시 적절한 에러 처리
- [ ] 네트워크 설정 비딩 방지
- [ ] 연결 타임아웃 및 재시도 로직
- [ ] 로깅 및 모니터링 기능

## 🧪 테스트 전략

### 1. 단위 테스트
```go
func TestNewClient_Success(t *testing.T) {
    // Docker daemon이 실행 중인 경우에만 테스트
    if !isDockerAvailable() {
        t.Skip("Docker daemon not available")
    }
    
    config := DefaultConfig()
    client, err := NewClient(config)
    
    assert.NoError(t, err)
    assert.NotNil(t, client)
    assert.NotEmpty(t, client.networkID)
}

func TestHealthChecker_CheckDaemon(t *testing.T) {
    // Mock Docker 환경에서 테스트
    mockClient := &mockDockerClient{}
    client := &Client{cli: mockClient}
    checker := NewHealthChecker(client, time.Second)
    
    err := checker.CheckDaemon(context.Background())
    assert.NoError(t, err)
}
```

### 2. 통합 테스트
- 실제 Docker daemon과의 연결 테스트
- 네트워크 생성 및 삭제 테스트
- 헬스체크 모니터링 테스트

## 📝 구현 단계

1. **Phase 1**: Docker 클라이언트 기본 구조 및 Config (1.5시간)
2. **Phase 2**: 네트워크 설정 및 관리 (1.5시간)
3. **Phase 3**: 헬스체크 시스템 및 모니터링 (1시간)
4. **Phase 4**: 유틸리티 및 에러 처리 (1시간)
5. **Phase 5**: 테스트 작성 및 검증 (1-2시간)

## 🔗 연관 태스크

- **의존성**: 없음 (독립적인 기본 기능)
- **후속 작업**: T03_S01_M04 (컬테이너 생명주기 관리자)
- **비동기 작업**: T01_S01_M04 (워크스페이스 서비스)

## 📚 참고 자료

- [Docker 통합 가이드](/docs/cli-design/docker-integration.md)
- [Docker Go SDK 공식 문서](https://docs.docker.com/engine/api/sdk/)
- [Docker API Reference](https://docs.docker.com/engine/api/v1.41/)
- [Docker Network 가이드](https://docs.docker.com/network/)