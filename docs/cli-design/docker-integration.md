# Docker 통합 가이드 (Go 구현)

## 🐳 개요

Go의 Docker SDK를 사용하여 워크스페이스 컨테이너를 효율적으로 관리하는 방법을 설명합니다.

## 📦 Docker Client 설정

### 1. 기본 클라이언트 구성

```go
// internal/docker/client.go
package docker

import (
    "context"
    "fmt"
    "io"
    "time"

    "github.com/docker/docker/api/types"
    "github.com/docker/docker/api/types/container"
    "github.com/docker/docker/api/types/mount"
    "github.com/docker/docker/api/types/network"
    "github.com/docker/docker/client"
    "github.com/docker/go-connections/nat"
)

type Client struct {
    cli           *client.Client
    config        *Config
    networkID     string
    labelPrefix   string
}

type Config struct {
    // 연결 설정
    Host          string        // Docker daemon 주소
    Version       string        // API 버전
    Timeout       time.Duration // 연결 타임아웃
    
    // 컨테이너 기본값
    DefaultImage  string        // 기본 이미지
    DefaultShell  []string      // 기본 쉘
    NetworkName   string        // 네트워크 이름
    
    // 리소스 제한
    CPULimit      float64       // CPU 제한 (1.0 = 1 CPU)
    MemoryLimit   int64         // 메모리 제한 (bytes)
    DiskLimit     int64         // 디스크 제한 (bytes)
    
    // 보안 설정
    Privileged    bool          // 특권 모드
    ReadOnly      bool          // 읽기 전용 루트
    SecurityOpts  []string      // 보안 옵션
}

func NewClient(config *Config) (*Client, error) {
    if config.Host == "" {
        // 기본값: 환경 변수 또는 Unix 소켓
        config.Host = client.DefaultDockerHost
    }
    
    cli, err := client.NewClientWithOpts(
        client.WithHost(config.Host),
        client.WithVersion(config.Version),
        client.WithTimeout(config.Timeout),
    )
    if err != nil {
        return nil, fmt.Errorf("create docker client: %w", err)
    }
    
    // Docker 연결 테스트
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    if _, err := cli.Ping(ctx); err != nil {
        return nil, fmt.Errorf("ping docker daemon: %w", err)
    }
    
    c := &Client{
        cli:         cli,
        config:      config,
        labelPrefix: "aicli",
    }
    
    // 네트워크 설정
    if err := c.setupNetwork(context.Background()); err != nil {
        return nil, fmt.Errorf("setup network: %w", err)
    }
    
    return c, nil
}

func (c *Client) setupNetwork(ctx context.Context) error {
    // 기존 네트워크 확인
    networks, err := c.cli.NetworkList(ctx, types.NetworkListOptions{})
    if err != nil {
        return err
    }
    
    for _, net := range networks {
        if net.Name == c.config.NetworkName {
            c.networkID = net.ID
            return nil
        }
    }
    
    // 새 네트워크 생성
    resp, err := c.cli.NetworkCreate(ctx, c.config.NetworkName, types.NetworkCreate{
        Driver:     "bridge",
        Attachable: true,
        Labels: map[string]string{
            c.labelKey("managed"): "true",
        },
    })
    if err != nil {
        return err
    }
    
    c.networkID = resp.ID
    return nil
}

func (c *Client) labelKey(key string) string {
    return fmt.Sprintf("%s.%s", c.labelPrefix, key)
}
```

### 2. 컨테이너 관리자

```go
// internal/docker/container.go
package docker

import (
    "context"
    "fmt"
    "io"
    "strings"
    
    "github.com/docker/docker/api/types"
    "github.com/docker/docker/api/types/container"
    "github.com/docker/docker/api/types/filters"
)

type ContainerManager struct {
    client *Client
}

type WorkspaceContainer struct {
    ID          string
    Name        string
    WorkspaceID string
    State       string
    Created     time.Time
    Stats       ContainerStats
}

type ContainerStats struct {
    CPUPercent    float64
    MemoryUsage   int64
    MemoryLimit   int64
    NetworkRxMB   float64
    NetworkTxMB   float64
}

func (m *ContainerManager) CreateWorkspace(ctx context.Context, req CreateWorkspaceRequest) (*WorkspaceContainer, error) {
    containerName := fmt.Sprintf("workspace_%s", req.WorkspaceID)
    
    // 기존 컨테이너 확인 및 제거
    if existing, err := m.getContainer(ctx, containerName); err == nil {
        if err := m.client.cli.ContainerRemove(ctx, existing.ID, types.ContainerRemoveOptions{
            Force: true,
        }); err != nil {
            return nil, fmt.Errorf("remove existing container: %w", err)
        }
    }
    
    // 컨테이너 설정
    config := &container.Config{
        Image:      req.Image,
        Hostname:   containerName,
        WorkingDir: "/workspace",
        Env: []string{
            fmt.Sprintf("WORKSPACE_ID=%s", req.WorkspaceID),
            fmt.Sprintf("WORKSPACE_NAME=%s", req.Name),
        },
        Labels: map[string]string{
            m.client.labelKey("workspace.id"):   req.WorkspaceID,
            m.client.labelKey("workspace.name"): req.Name,
            m.client.labelKey("managed"):        "true",
        },
        AttachStdin:  true,
        AttachStdout: true,
        AttachStderr: true,
        OpenStdin:    true,
        StdinOnce:    false,
        Tty:          true,
    }
    
    // 기본 쉘 설정
    if len(req.Shell) > 0 {
        config.Cmd = req.Shell
    } else {
        config.Cmd = m.client.config.DefaultShell
    }
    
    // 호스트 설정
    hostConfig := &container.HostConfig{
        // 볼륨 마운트
        Mounts: []mount.Mount{
            {
                Type:   mount.TypeBind,
                Source: req.ProjectPath,
                Target: "/workspace",
                BindOptions: &mount.BindOptions{
                    Propagation: mount.PropagationRPrivate,
                },
            },
        },
        
        // 리소스 제한
        Resources: container.Resources{
            CPUQuota:  int64(m.client.config.CPULimit * 100000),
            CPUPeriod: 100000,
            Memory:    m.client.config.MemoryLimit,
            MemorySwap: m.client.config.MemoryLimit, // swap 비활성화
        },
        
        // 보안 설정
        Privileged:  m.client.config.Privileged,
        ReadonlyRootfs: m.client.config.ReadOnly,
        SecurityOpt: m.client.config.SecurityOpts,
        CapDrop:     []string{"ALL"},
        CapAdd:      []string{"CHOWN", "SETUID", "SETGID"},
        
        // 자동 재시작
        RestartPolicy: container.RestartPolicy{
            Name: "unless-stopped",
        },
    }
    
    // Docker 소켓 마운트 (선택사항)
    if req.MountDockerSocket {
        hostConfig.Mounts = append(hostConfig.Mounts, mount.Mount{
            Type:   mount.TypeBind,
            Source: "/var/run/docker.sock",
            Target: "/var/run/docker.sock",
            ReadOnly: true,
        })
    }
    
    // 네트워크 설정
    networkConfig := &network.NetworkingConfig{
        EndpointsConfig: map[string]*network.EndpointSettings{
            m.client.config.NetworkName: {
                NetworkID: m.client.networkID,
            },
        },
    }
    
    // 컨테이너 생성
    resp, err := m.client.cli.ContainerCreate(
        ctx,
        config,
        hostConfig,
        networkConfig,
        nil,
        containerName,
    )
    if err != nil {
        return nil, fmt.Errorf("create container: %w", err)
    }
    
    // 컨테이너 시작
    if err := m.client.cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
        // 실패 시 정리
        m.client.cli.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{Force: true})
        return nil, fmt.Errorf("start container: %w", err)
    }
    
    return &WorkspaceContainer{
        ID:          resp.ID,
        Name:        containerName,
        WorkspaceID: req.WorkspaceID,
        State:       "running",
        Created:     time.Now(),
    }, nil
}

type CreateWorkspaceRequest struct {
    WorkspaceID       string
    Name              string
    ProjectPath       string
    Image             string
    Shell             []string
    MountDockerSocket bool
}
```

### 3. 명령 실행

```go
// internal/docker/exec.go
package docker

import (
    "bytes"
    "context"
    "io"
    
    "github.com/docker/docker/api/types"
)

type ExecManager struct {
    client *Client
}

type ExecResult struct {
    ExitCode int
    Stdout   string
    Stderr   string
}

func (m *ExecManager) Execute(ctx context.Context, containerID string, cmd []string) (*ExecResult, error) {
    // Exec 인스턴스 생성
    execConfig := types.ExecConfig{
        Cmd:          cmd,
        AttachStdout: true,
        AttachStderr: true,
        Tty:          false,
    }
    
    execResp, err := m.client.cli.ContainerExecCreate(ctx, containerID, execConfig)
    if err != nil {
        return nil, fmt.Errorf("create exec: %w", err)
    }
    
    // Exec 시작
    resp, err := m.client.cli.ContainerExecAttach(ctx, execResp.ID, types.ExecStartCheck{})
    if err != nil {
        return nil, fmt.Errorf("attach exec: %w", err)
    }
    defer resp.Close()
    
    // 출력 읽기
    var stdout, stderr bytes.Buffer
    if _, err := stdcopy.StdCopy(&stdout, &stderr, resp.Reader); err != nil {
        return nil, fmt.Errorf("read output: %w", err)
    }
    
    // 종료 코드 확인
    inspect, err := m.client.cli.ContainerExecInspect(ctx, execResp.ID)
    if err != nil {
        return nil, fmt.Errorf("inspect exec: %w", err)
    }
    
    return &ExecResult{
        ExitCode: inspect.ExitCode,
        Stdout:   stdout.String(),
        Stderr:   stderr.String(),
    }, nil
}

func (m *ExecManager) Stream(ctx context.Context, containerID string, cmd []string) (*ExecStream, error) {
    execConfig := types.ExecConfig{
        Cmd:          cmd,
        AttachStdin:  true,
        AttachStdout: true,
        AttachStderr: true,
        Tty:          true,
    }
    
    execResp, err := m.client.cli.ContainerExecCreate(ctx, containerID, execConfig)
    if err != nil {
        return nil, err
    }
    
    resp, err := m.client.cli.ContainerExecAttach(ctx, execResp.ID, types.ExecStartCheck{
        Tty: true,
    })
    if err != nil {
        return nil, err
    }
    
    return &ExecStream{
        ID:     execResp.ID,
        Conn:   resp.Conn,
        Reader: resp.Reader,
        client: m.client,
    }, nil
}

type ExecStream struct {
    ID     string
    Conn   io.ReadWriteCloser
    Reader io.Reader
    client *Client
}

func (s *ExecStream) Write(data []byte) (int, error) {
    return s.Conn.Write(data)
}

func (s *ExecStream) Read(p []byte) (int, error) {
    return s.Reader.Read(p)
}

func (s *ExecStream) Close() error {
    return s.Conn.Close()
}
```

### 4. 로그 스트리밍

```go
// internal/docker/logs.go
package docker

import (
    "bufio"
    "context"
    "encoding/json"
    "time"
    
    "github.com/docker/docker/api/types"
)

type LogStreamer struct {
    client *Client
}

type LogEntry struct {
    Timestamp   time.Time `json:"timestamp"`
    Stream      string    `json:"stream"` // stdout/stderr
    Message     string    `json:"message"`
    ContainerID string    `json:"container_id"`
}

func (s *LogStreamer) Stream(ctx context.Context, containerID string, since time.Time) (<-chan LogEntry, error) {
    options := types.ContainerLogsOptions{
        ShowStdout: true,
        ShowStderr: true,
        Follow:     true,
        Timestamps: true,
        Since:      since.Format(time.RFC3339),
    }
    
    reader, err := s.client.cli.ContainerLogs(ctx, containerID, options)
    if err != nil {
        return nil, err
    }
    
    logChan := make(chan LogEntry, 100)
    
    go func() {
        defer close(logChan)
        defer reader.Close()
        
        scanner := bufio.NewScanner(reader)
        for scanner.Scan() {
            select {
            case <-ctx.Done():
                return
            default:
            }
            
            line := scanner.Text()
            entry := s.parseLogLine(line, containerID)
            
            select {
            case logChan <- entry:
            case <-ctx.Done():
                return
            }
        }
    }()
    
    return logChan, nil
}

func (s *LogStreamer) parseLogLine(line string, containerID string) LogEntry {
    // Docker 로그 형식 파싱
    // 예: 2024-01-20T10:30:00.123456789Z stdout P Hello World
    
    entry := LogEntry{
        ContainerID: containerID,
        Timestamp:   time.Now(),
    }
    
    if len(line) > 30 {
        // 타임스탬프 파싱
        if t, err := time.Parse(time.RFC3339Nano, line[:30]); err == nil {
            entry.Timestamp = t
            line = line[31:] // 타임스탬프 + 공백 제거
        }
    }
    
    // 스트림 타입 파싱
    if strings.HasPrefix(line, "stdout ") {
        entry.Stream = "stdout"
        entry.Message = line[7:]
    } else if strings.HasPrefix(line, "stderr ") {
        entry.Stream = "stderr"
        entry.Message = line[7:]
    } else {
        entry.Message = line
    }
    
    return entry
}

// 집계된 로그 스트리밍
func (s *LogStreamer) StreamAll(ctx context.Context, workspaceID string) (<-chan LogEntry, error) {
    // 워크스페이스의 모든 컨테이너 찾기
    filters := filters.NewArgs()
    filters.Add("label", fmt.Sprintf("%s.workspace.id=%s", s.client.labelPrefix, workspaceID))
    
    containers, err := s.client.cli.ContainerList(ctx, types.ContainerListOptions{
        Filters: filters,
    })
    if err != nil {
        return nil, err
    }
    
    aggregated := make(chan LogEntry, 100)
    var wg sync.WaitGroup
    
    for _, container := range containers {
        wg.Add(1)
        go func(containerID string) {
            defer wg.Done()
            
            logChan, err := s.Stream(ctx, containerID, time.Now())
            if err != nil {
                return
            }
            
            for log := range logChan {
                select {
                case aggregated <- log:
                case <-ctx.Done():
                    return
                }
            }
        }(container.ID)
    }
    
    go func() {
        wg.Wait()
        close(aggregated)
    }()
    
    return aggregated, nil
}
```

### 5. 리소스 모니터링

```go
// internal/docker/stats.go
package docker

import (
    "context"
    "encoding/json"
    "sync"
    
    "github.com/docker/docker/api/types"
)

type StatsCollector struct {
    client *Client
    cache  sync.Map // containerID -> ContainerStats
}

func (c *StatsCollector) Collect(ctx context.Context, containerID string) (*ContainerStats, error) {
    stats, err := c.client.cli.ContainerStats(ctx, containerID, false)
    if err != nil {
        return nil, err
    }
    defer stats.Body.Close()
    
    var v types.StatsJSON
    if err := json.NewDecoder(stats.Body).Decode(&v); err != nil {
        return nil, err
    }
    
    // CPU 사용률 계산
    cpuDelta := float64(v.CPUStats.CPUUsage.TotalUsage - v.PreCPUStats.CPUUsage.TotalUsage)
    systemDelta := float64(v.CPUStats.SystemUsage - v.PreCPUStats.SystemUsage)
    cpuPercent := 0.0
    if systemDelta > 0.0 && cpuDelta > 0.0 {
        cpuPercent = (cpuDelta / systemDelta) * float64(len(v.CPUStats.CPUUsage.PercpuUsage)) * 100.0
    }
    
    // 메모리 사용량
    memUsage := v.MemoryStats.Usage - v.MemoryStats.Stats["cache"]
    memLimit := v.MemoryStats.Limit
    
    // 네트워크 통계
    var rxBytes, txBytes uint64
    for _, net := range v.Networks {
        rxBytes += net.RxBytes
        txBytes += net.TxBytes
    }
    
    stats := &ContainerStats{
        CPUPercent:  cpuPercent,
        MemoryUsage: int64(memUsage),
        MemoryLimit: int64(memLimit),
        NetworkRxMB: float64(rxBytes) / 1024 / 1024,
        NetworkTxMB: float64(txBytes) / 1024 / 1024,
    }
    
    // 캐시 업데이트
    c.cache.Store(containerID, stats)
    
    return stats, nil
}

func (c *StatsCollector) CollectAll(ctx context.Context) (map[string]*ContainerStats, error) {
    containers, err := c.client.cli.ContainerList(ctx, types.ContainerListOptions{
        Filters: filters.NewArgs(
            filters.Arg("label", fmt.Sprintf("%s.managed=true", c.client.labelPrefix)),
        ),
    })
    if err != nil {
        return nil, err
    }
    
    result := make(map[string]*ContainerStats)
    var wg sync.WaitGroup
    var mu sync.Mutex
    
    for _, container := range containers {
        wg.Add(1)
        go func(id string) {
            defer wg.Done()
            
            stats, err := c.Collect(ctx, id)
            if err == nil {
                mu.Lock()
                result[id] = stats
                mu.Unlock()
            }
        }(container.ID)
    }
    
    wg.Wait()
    return result, nil
}

// 실시간 모니터링
func (c *StatsCollector) Monitor(ctx context.Context, containerID string) (<-chan *ContainerStats, error) {
    statsChan := make(chan *ContainerStats, 10)
    
    go func() {
        defer close(statsChan)
        
        ticker := time.NewTicker(time.Second)
        defer ticker.Stop()
        
        for {
            select {
            case <-ctx.Done():
                return
            case <-ticker.C:
                stats, err := c.Collect(ctx, containerID)
                if err != nil {
                    continue
                }
                
                select {
                case statsChan <- stats:
                case <-ctx.Done():
                    return
                }
            }
        }
    }()
    
    return statsChan, nil
}
```

### 6. 이미지 관리

```go
// internal/docker/image.go
package docker

import (
    "context"
    "io"
    
    "github.com/docker/docker/api/types"
)

type ImageManager struct {
    client *Client
}

func (m *ImageManager) Pull(ctx context.Context, image string, output io.Writer) error {
    reader, err := m.client.cli.ImagePull(ctx, image, types.ImagePullOptions{})
    if err != nil {
        return fmt.Errorf("pull image: %w", err)
    }
    defer reader.Close()
    
    // 진행 상황 파싱 및 출력
    decoder := json.NewDecoder(reader)
    for {
        var event map[string]interface{}
        if err := decoder.Decode(&event); err != nil {
            if err == io.EOF {
                break
            }
            return err
        }
        
        if output != nil {
            json.NewEncoder(output).Encode(event)
        }
    }
    
    return nil
}

func (m *ImageManager) Build(ctx context.Context, req BuildRequest) error {
    // Dockerfile을 tar 아카이브로 패키징
    tar, err := m.createBuildContext(req.ContextPath, req.Dockerfile)
    if err != nil {
        return fmt.Errorf("create build context: %w", err)
    }
    defer tar.Close()
    
    options := types.ImageBuildOptions{
        Tags:       []string{req.Tag},
        Dockerfile: "Dockerfile",
        Remove:     true,
        Labels: map[string]string{
            m.client.labelKey("managed"): "true",
            m.client.labelKey("built"):   time.Now().Format(time.RFC3339),
        },
    }
    
    resp, err := m.client.cli.ImageBuild(ctx, tar, options)
    if err != nil {
        return fmt.Errorf("build image: %w", err)
    }
    defer resp.Body.Close()
    
    // 빌드 출력 처리
    if req.Output != nil {
        if _, err := io.Copy(req.Output, resp.Body); err != nil {
            return fmt.Errorf("read build output: %w", err)
        }
    } else {
        io.Copy(io.Discard, resp.Body)
    }
    
    return nil
}

type BuildRequest struct {
    ContextPath string
    Dockerfile  string
    Tag         string
    Output      io.Writer
}
```

## 🔧 고급 기능

### 1. 헬스체크

```go
type HealthChecker struct {
    client   *Client
    interval time.Duration
}

func (h *HealthChecker) Check(ctx context.Context, containerID string) (bool, error) {
    inspect, err := h.client.cli.ContainerInspect(ctx, containerID)
    if err != nil {
        return false, err
    }
    
    if inspect.State.Health != nil {
        return inspect.State.Health.Status == "healthy", nil
    }
    
    // 헬스체크가 없으면 실행 상태만 확인
    return inspect.State.Running, nil
}

func (h *HealthChecker) WaitHealthy(ctx context.Context, containerID string, timeout time.Duration) error {
    ctx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()
    
    ticker := time.NewTicker(h.interval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return fmt.Errorf("timeout waiting for container health")
        case <-ticker.C:
            healthy, err := h.Check(ctx, containerID)
            if err != nil {
                return err
            }
            if healthy {
                return nil
            }
        }
    }
}
```

### 2. 자동 정리

```go
type Cleaner struct {
    client   *Client
    maxAge   time.Duration
    interval time.Duration
}

func (c *Cleaner) Start(ctx context.Context) {
    ticker := time.NewTicker(c.interval)
    go func() {
        defer ticker.Stop()
        
        for {
            select {
            case <-ctx.Done():
                return
            case <-ticker.C:
                c.cleanup(ctx)
            }
        }
    }()
}

func (c *Cleaner) cleanup(ctx context.Context) error {
    // 중지된 컨테이너 정리
    containers, err := c.client.cli.ContainerList(ctx, types.ContainerListOptions{
        All: true,
        Filters: filters.NewArgs(
            filters.Arg("label", fmt.Sprintf("%s.managed=true", c.client.labelPrefix)),
            filters.Arg("status", "exited"),
        ),
    })
    if err != nil {
        return err
    }
    
    now := time.Now()
    for _, container := range containers {
        created := time.Unix(container.Created, 0)
        if now.Sub(created) > c.maxAge {
            c.client.cli.ContainerRemove(ctx, container.ID, types.ContainerRemoveOptions{
                Force: true,
            })
        }
    }
    
    // 사용하지 않는 이미지 정리
    c.client.cli.ImagesPrune(ctx, filters.Args{})
    
    // 사용하지 않는 볼륨 정리
    c.client.cli.VolumesPrune(ctx, filters.Args{})
    
    return nil
}
```

## 📊 메트릭 수집

```go
type MetricsCollector struct {
    stats    *StatsCollector
    interval time.Duration
    store    MetricsStore
}

type MetricsStore interface {
    Store(containerID string, metrics ContainerMetrics) error
}

type ContainerMetrics struct {
    Timestamp   time.Time
    CPUPercent  float64
    MemoryMB    float64
    NetworkRxMB float64
    NetworkTxMB float64
    DiskIOMB    float64
}

func (m *MetricsCollector) Start(ctx context.Context) {
    ticker := time.NewTicker(m.interval)
    go func() {
        defer ticker.Stop()
        
        for {
            select {
            case <-ctx.Done():
                return
            case <-ticker.C:
                m.collect(ctx)
            }
        }
    }()
}

func (m *MetricsCollector) collect(ctx context.Context) {
    stats, err := m.stats.CollectAll(ctx)
    if err != nil {
        return
    }
    
    for containerID, stat := range stats {
        metrics := ContainerMetrics{
            Timestamp:   time.Now(),
            CPUPercent:  stat.CPUPercent,
            MemoryMB:    float64(stat.MemoryUsage) / 1024 / 1024,
            NetworkRxMB: stat.NetworkRxMB,
            NetworkTxMB: stat.NetworkTxMB,
        }
        
        m.store.Store(containerID, metrics)
    }
}
```

## 🛡️ 보안 강화

```go
type SecurityConfig struct {
    // AppArmor 프로파일
    AppArmorProfile string
    
    // Seccomp 프로파일
    SeccompProfile string
    
    // SELinux 레이블
    SELinuxLabel string
    
    // 사용자 네임스페이스
    UsernsMode string
    
    // PID 제한
    PidsLimit int64
}

func (c *Client) applySecurityConfig(hostConfig *container.HostConfig, sec SecurityConfig) {
    if sec.AppArmorProfile != "" {
        hostConfig.SecurityOpt = append(hostConfig.SecurityOpt, 
            fmt.Sprintf("apparmor=%s", sec.AppArmorProfile))
    }
    
    if sec.SeccompProfile != "" {
        hostConfig.SecurityOpt = append(hostConfig.SecurityOpt,
            fmt.Sprintf("seccomp=%s", sec.SeccompProfile))
    }
    
    if sec.SELinuxLabel != "" {
        hostConfig.SecurityOpt = append(hostConfig.SecurityOpt,
            fmt.Sprintf("label=%s", sec.SELinuxLabel))
    }
    
    if sec.UsernsMode != "" {
        hostConfig.UsernsMode = container.UsernsMode(sec.UsernsMode)
    }
    
    if sec.PidsLimit > 0 {
        hostConfig.PidsLimit = &sec.PidsLimit
    }
}
```