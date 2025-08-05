---
task_id: T04_S02_Docker_PTY_Integration
sprint_id: S02_M06_PTY_Streaming
milestone_id: M06
title: Docker 컨테이너 PTY 통합 구현
type: implementation
complexity: High
status: pending
assignee: unassigned
created: 2025-08-05T10:00:00+0900
last_updated: 2025-08-05T10:00:00+0900
depends_on: [T01_S02_PTY_Session_Manager]
blocks: [T08_S02_Integration_Tests]
epic: PTY_Streaming_System
---

# Task: Docker 컨테이너 PTY 통합 구현

## Task Summary
Docker SDK를 활용하여 컨테이너와 PTY 세션을 연결하는 통합 시스템을 구현합니다. 컨테이너 내부 셸 환경 설정, 리소스 모니터링, 생명주기 관리를 포함한 완전한 Docker PTY 통합을 제공합니다.

## Acceptance Criteria

### 기능 요구사항
- [ ] Docker 컨테이너와 PTY 세션 연결 시스템
- [ ] 컨테이너 내부 셸 환경 자동 설정
- [ ] 환경변수 및 작업 디렉토리 동적 관리
- [ ] 컨테이너 리소스 실시간 모니터링
- [ ] 컨테이너 재시작 시 PTY 자동 재연결
- [ ] 다중 컨테이너 동시 PTY 연결 지원
- [ ] 컨테이너 이미지별 맞춤 설정 지원

### 성능 요구사항
- [ ] PTY 연결 설정 시간 < 2초
- [ ] 컨테이너 상태 확인 주기 < 5초
- [ ] 리소스 모니터링 오버헤드 < 5%
- [ ] 동시 50개 컨테이너 PTY 연결 지원
- [ ] 메모리 사용량 연결당 < 20MB

### 안정성 요구사항
- [ ] 컨테이너 중단 시 우아한 PTY 해제
- [ ] 네트워크 단절 시 자동 복구
- [ ] Docker 데몬 재시작 대응
- [ ] 리소스 누수 방지

## Implementation Details

### 1. Docker PTY 통합 관리자 구조

```go
// internal/docker/pty_integration.go
type DockerPTYIntegration struct {
    client      *client.Client
    sessions    map[string]*DockerPTYSession
    config      *DockerPTYConfig
    monitor     *ContainerMonitor
    mutex       sync.RWMutex
    stopCh      chan struct{}
}

type DockerPTYSession struct {
    SessionID       string
    ContainerID     string
    Container       *DockerContainer
    PTYConn         types.HijackedResponse
    ExecID          string
    Config          *PTYSessionConfig
    Status          DockerPTYStatus
    CreatedAt       time.Time
    LastActivity    time.Time
    ResourceStats   *ContainerResourceStats
    cancel          context.CancelFunc
}

type DockerContainer struct {
    ID              string
    Name            string
    Image           string
    Status          string
    Networks        map[string]types.EndpointSettings
    Mounts          []types.MountPoint
    Environment     []string
    WorkingDir      string
    User            string
}

type DockerPTYStatus int
const (
    PTYConnecting DockerPTYStatus = iota
    PTYActive
    PTYReconnecting
    PTYTerminated
    PTYError
)
```

### 2. 컨테이너 연결 및 PTY 생성

```go
// Docker PTY 연결 인터페이스
type DockerPTYInterface interface {
    ConnectContainer(ctx context.Context, containerID string, config *PTYSessionConfig) (*DockerPTYSession, error)
    DisconnectContainer(sessionID string) error
    GetContainerInfo(containerID string) (*DockerContainer, error)
    ListActiveSessions() map[string]*DockerPTYSession
    MonitorContainer(sessionID string) (*ContainerResourceStats, error)
    RestartPTYConnection(sessionID string) error
}

// PTY 세션 설정
type PTYSessionConfig struct {
    Shell           string
    Term            string
    Rows            int
    Cols            int
    WorkingDir      string
    Environment     map[string]string
    User            string
    Privileged      bool
    AttachStdin     bool
    AttachStdout    bool
    AttachStderr    bool
    Detach          bool
}

// 컨테이너 연결 구현
func (dpi *DockerPTYIntegration) ConnectContainer(ctx context.Context, containerID string, config *PTYSessionConfig) (*DockerPTYSession, error) {
    // 컨테이너 존재 및 상태 확인
    containerInfo, err := dpi.inspectContainer(ctx, containerID)
    if err != nil {
        return nil, fmt.Errorf("failed to inspect container: %w", err)
    }
    
    if containerInfo.State.Status != "running" {
        return nil, fmt.Errorf("container is not running: %s", containerInfo.State.Status)
    }
    
    // Exec 인스턴스 생성
    execConfig := types.ExecConfig{
        Cmd:          []string{config.Shell},
        AttachStdin:  config.AttachStdin,
        AttachStdout: config.AttachStdout,
        AttachStderr: config.AttachStderr,
        Tty:          true,
        Env:          dpi.buildEnvironment(config.Environment),
        WorkingDir:   config.WorkingDir,
        User:         config.User,
        Privileged:   config.Privileged,
    }
    
    execResp, err := dpi.client.ContainerExecCreate(ctx, containerID, execConfig)
    if err != nil {
        return nil, fmt.Errorf("failed to create exec: %w", err)
    }
    
    // PTY 연결 생성
    ptyConn, err := dpi.client.ContainerExecAttach(ctx, execResp.ID, types.ExecStartCheck{
        Detach: config.Detach,
        Tty:    true,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to attach exec: %w", err)
    }
    
    // 세션 객체 생성
    sessionCtx, cancel := context.WithCancel(ctx)
    session := &DockerPTYSession{
        SessionID:    generateSessionID(),
        ContainerID:  containerID,
        Container:    dpi.convertContainerInfo(containerInfo),
        PTYConn:      ptyConn,
        ExecID:       execResp.ID,
        Config:       config,
        Status:       PTYConnecting,
        CreatedAt:    time.Now(),
        LastActivity: time.Now(),
        cancel:       cancel,
    }
    
    // Exec 시작
    if err := dpi.client.ContainerExecStart(sessionCtx, execResp.ID, types.ExecStartCheck{
        Detach: config.Detach,
        Tty:    true,
    }); err != nil {
        cancel()
        ptyConn.Close()
        return nil, fmt.Errorf("failed to start exec: %w", err)
    }
    
    session.Status = PTYActive
    
    // 세션 등록 및 모니터링 시작
    dpi.registerSession(session)
    go dpi.monitorSession(session)
    
    return session, nil
}
```

### 3. 컨테이너 리소스 모니터링

```go
// 컨테이너 리소스 모니터링
type ContainerMonitor struct {
    client          *client.Client
    sessions        map[string]*DockerPTYSession
    monitorTicker   *time.Ticker
    config          *MonitorConfig
    stopCh          chan struct{}
}

type ContainerResourceStats struct {
    ContainerID     string
    CPUUsage        float64
    MemoryUsage     int64
    MemoryLimit     int64
    NetworkIO       NetworkIOStats
    BlockIO         BlockIOStats
    PIDs            int
    Timestamp       time.Time
}

type NetworkIOStats struct {
    RxBytes   uint64
    TxBytes   uint64
    RxPackets uint64
    TxPackets uint64
}

type BlockIOStats struct {
    ReadBytes  uint64
    WriteBytes uint64
    ReadOps    uint64
    WriteOps   uint64
}

func (cm *ContainerMonitor) startMonitoring() {
    cm.monitorTicker = time.NewTicker(cm.config.MonitorInterval)
    
    go func() {
        defer cm.monitorTicker.Stop()
        
        for {
            select {
            case <-cm.monitorTicker.C:
                cm.collectResourceStats()
            case <-cm.stopCh:
                return
            }
        }
    }()
}

func (cm *ContainerMonitor) collectResourceStats() {
    for sessionID, session := range cm.sessions {
        go func(s *DockerPTYSession, id string) {
            stats, err := cm.getContainerStats(s.ContainerID)
            if err != nil {
                log.Errorf("Failed to collect stats for container %s: %v", s.ContainerID, err)
                return
            }
            
            s.ResourceStats = stats
            s.LastActivity = time.Now()
            
            // 리소스 임계값 확인
            if err := cm.checkResourceLimits(s, stats); err != nil {
                log.Warnf("Resource limit exceeded for session %s: %v", id, err)
            }
        }(session, sessionID)
    }
}
```

### 4. 컨테이너 생명주기 관리

```go
// 컨테이너 상태 모니터링 및 PTY 재연결
func (dpi *DockerPTYIntegration) monitorSession(session *DockerPTYSession) {
    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            if err := dpi.checkContainerHealth(session); err != nil {
                log.Errorf("Container health check failed: %v", err)
                
                if dpi.shouldReconnect(session, err) {
                    go dpi.attemptReconnection(session)
                }
            }
        case <-session.cancel.Done():
            return
        case <-dpi.stopCh:
            return
        }
    }
}

func (dpi *DockerPTYIntegration) checkContainerHealth(session *DockerPTYSession) error {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    
    containerInfo, err := dpi.client.ContainerInspect(ctx, session.ContainerID)
    if err != nil {
        return fmt.Errorf("failed to inspect container: %w", err)
    }
    
    if containerInfo.State.Status != "running" {
        return fmt.Errorf("container is not running: %s", containerInfo.State.Status)
    }
    
    // Exec 상태 확인
    execInfo, err := dpi.client.ContainerExecInspect(ctx, session.ExecID)
    if err != nil {
        return fmt.Errorf("failed to inspect exec: %w", err)
    }
    
    if execInfo.ExitCode != nil && *execInfo.ExitCode != 0 {
        return fmt.Errorf("exec process exited with code: %d", *execInfo.ExitCode)
    }
    
    return nil
}

func (dpi *DockerPTYIntegration) attemptReconnection(session *DockerPTYSession) {
    session.Status = PTYReconnecting
    
    maxRetries := 3
    backoffDelay := time.Second
    
    for attempt := 1; attempt <= maxRetries; attempt++ {
        log.Infof("Attempting PTY reconnection for session %s (attempt %d/%d)", 
                 session.SessionID, attempt, maxRetries)
        
        newSession, err := dpi.ConnectContainer(
            context.Background(), 
            session.ContainerID, 
            session.Config,
        )
        
        if err == nil {
            // 기존 세션 정리
            dpi.cleanupSession(session)
            
            // 새 세션으로 교체
            newSession.SessionID = session.SessionID
            dpi.replaceSession(session.SessionID, newSession)
            
            log.Infof("PTY reconnection successful for session %s", session.SessionID)
            return
        }
        
        log.Errorf("PTY reconnection attempt %d failed: %v", attempt, err)
        
        if attempt < maxRetries {
            time.Sleep(backoffDelay)
            backoffDelay *= 2
        }
    }
    
    // 재연결 실패 시 세션 종료
    session.Status = PTYError
    dpi.cleanupSession(session)
    log.Errorf("PTY reconnection failed for session %s after %d attempts", 
              session.SessionID, maxRetries)
}
```

### 5. 환경 설정 및 이미지별 최적화

```go
// 이미지별 최적화 설정
type ImageOptimizer struct {
    profiles map[string]*ImageProfile
    mutex    sync.RWMutex
}

type ImageProfile struct {
    ImagePattern    string
    DefaultShell    string
    Environment     map[string]string
    WorkingDir      string
    PreExecCommands []string
    PostExecCommands []string
    ResourceLimits  *ResourceLimits
}

type ResourceLimits struct {
    CPULimit    float64
    MemoryLimit int64
    PIDs        int
}

func (io *ImageOptimizer) GetProfileForImage(imageName string) *ImageProfile {
    io.mutex.RLock()
    defer io.mutex.RUnlock()
    
    for pattern, profile := range io.profiles {
        if matched, _ := filepath.Match(pattern, imageName); matched {
            return profile
        }
    }
    
    return io.getDefaultProfile()
}

// 환경변수 빌드
func (dpi *DockerPTYIntegration) buildEnvironment(customEnv map[string]string) []string {
    defaultEnv := map[string]string{
        "TERM":     "xterm-256color",
        "LANG":     "en_US.UTF-8",
        "LC_ALL":   "en_US.UTF-8",
        "PS1":      "\\u@\\h:\\w\\$ ",
        "HOME":     "/root",
        "PATH":     "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
    }
    
    // 기본 환경변수와 사용자 정의 환경변수 병합
    for key, value := range customEnv {
        defaultEnv[key] = value
    }
    
    var envList []string
    for key, value := range defaultEnv {
        envList = append(envList, fmt.Sprintf("%s=%s", key, value))
    }
    
    return envList
}
```

## 파일 구조

```
internal/docker/
├── pty_integration.go     # 메인 Docker PTY 통합
├── container_monitor.go   # 컨테이너 모니터링
├── session_manager.go     # PTY 세션 관리
├── image_optimizer.go     # 이미지별 최적화
├── resource_monitor.go    # 리소스 모니터링
├── reconnection.go        # 재연결 관리
└── config.go             # 설정 관리

internal/docker/types/
├── container.go          # 컨테이너 타입 정의
├── session.go           # 세션 타입 정의
├── stats.go             # 통계 타입 정의
└── config.go            # 설정 타입 정의

internal/docker/test/
├── integration_test.go
├── monitor_test.go
├── reconnection_test.go
└── mock_docker.go
```

## 핵심 구현 사항

### 1. Docker SDK 최적화
- 연결 풀을 사용한 Docker 클라이언트 관리
- 비동기 API 호출을 통한 성능 향상
- 에러 처리 및 재시도 로직 구현

### 2. 리소스 효율성
- 컨테이너별 리소스 제한 설정
- 메모리 누수 방지를 위한 정기적 정리
- CPU 및 메모리 사용량 모니터링

### 3. 안정성 및 복구
- 컨테이너 상태 실시간 모니터링
- 네트워크 단절 시 자동 재연결
- 우아한 종료 및 리소스 정리

## Dependencies

### 필수 패키지
```go
import (
    "context"
    "sync"
    "time"
    "path/filepath"
    "fmt"
    
    // Docker SDK
    "github.com/docker/docker/client"
    "github.com/docker/docker/api/types"
    "github.com/docker/docker/api/types/container"
    
    // 로깅
    "github.com/sirupsen/logrus"
    
    // 메트릭
    "github.com/prometheus/client_golang/prometheus"
)
```

## API 엔드포인트 통합

```go
// cmd/api/handlers/docker_pty.go
func (h *DockerPTYHandler) ConnectContainer(w http.ResponseWriter, r *http.Request) {
    var req ConnectContainerRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }
    
    session, err := h.dockerPTY.ConnectContainer(r.Context(), req.ContainerID, req.Config)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(ConnectContainerResponse{
        SessionID: session.SessionID,
        Status:    "connected",
    })
}

type ConnectContainerRequest struct {
    ContainerID string               `json:"container_id"`
    Config      *PTYSessionConfig    `json:"config"`
}

type ConnectContainerResponse struct {
    SessionID string `json:"session_id"`
    Status    string `json:"status"`
}
```

## 테스트 계획

### 단위 테스트
- Docker 컨테이너 연결/해제 테스트
- 리소스 모니터링 테스트
- 재연결 로직 테스트
- 환경변수 설정 테스트

### 통합 테스트
- 실제 Docker 컨테이너와의 통합 테스트
- 다양한 이미지별 연결 테스트
- 장시간 실행 안정성 테스트

### 성능 테스트
- 동시 연결 처리 성능 테스트
- 리소스 사용량 측정
- 재연결 성능 벤치마크

## Definition of Done
- [ ] Docker 컨테이너 PTY 연결 시스템 구현 완료
- [ ] 컨테이너 모니터링 및 재연결 기능 완료
- [ ] 이미지별 최적화 설정 적용 완료
- [ ] 단위 테스트 및 통합 테스트 통과
- [ ] 성능 요구사항 달성 확인
- [ ] 에러 처리 및 복구 메커니즘 검증 완료
- [ ] 코드 리뷰 완료

## Notes
- Docker API 버전 호환성 확인 필요
- 보안을 위해 Privileged 실행은 제한적으로 허용
- 컨테이너 로그와 PTY 출력의 차이점 고려
- Windows 컨테이너는 향후 지원 고려