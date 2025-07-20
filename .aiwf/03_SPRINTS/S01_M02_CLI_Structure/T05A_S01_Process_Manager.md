---
task_id: T05A_S01
sprint_sequence_id: S01_M02
status: open
complexity: High
last_updated: 2025-07-21T06:30:00Z
github_issue: # Optional: GitHub issue number
---

# Task: Claude CLI 프로세스 관리자 구현

## Description
Claude CLI 프로세스의 생명주기를 관리하는 핵심 모듈을 구현합니다. 프로세스 시작, 중지, 상태 관리, 시그널 처리를 포함한 안정적인 프로세스 관리 시스템을 구축합니다.

## Goal / Objectives
- Claude CLI 프로세스 생명주기 관리
- 프로세스 상태 머신 구현
- 시그널 처리 및 우아한 종료
- 프로세스 모니터링 및 헬스체크

## Acceptance Criteria
- [ ] 프로세스 시작/중지 기능 구현
- [ ] 프로세스 상태 추적 및 상태 머신 구현
- [ ] 시그널 처리 (SIGTERM, SIGKILL) 구현
- [ ] 프로세스 헬스체크 및 모니터링
- [ ] 컨텍스트 기반 취소 처리
- [ ] 프로세스 메타데이터 관리

## Subtasks
- [ ] 프로세스 관리자 인터페이스 설계
- [ ] 프로세스 상태 머신 구현
- [ ] 프로세스 시작 로직 구현
- [ ] 프로세스 중지 및 시그널 처리
- [ ] 헬스체크 및 모니터링
- [ ] 에러 처리 및 로깅 통합
- [ ] 프로세스 관리자 테스트

## 기술 가이드

### 프로세스 관리자 인터페이스
```go
type ProcessManager interface {
    Start(ctx context.Context, config *ProcessConfig) error
    Stop(timeout time.Duration) error
    Kill() error
    IsRunning() bool
    GetStatus() ProcessStatus
    GetPID() int
    Wait() error
    HealthCheck() error
}

type ProcessConfig struct {
    Command     string
    Args        []string
    WorkingDir  string
    Environment map[string]string
    Timeout     time.Duration
}

type ProcessStatus int

const (
    StatusStopped ProcessStatus = iota
    StatusStarting
    StatusRunning
    StatusStopping
    StatusError
    StatusUnknown
)

func (s ProcessStatus) String() string {
    switch s {
    case StatusStopped:
        return "stopped"
    case StatusStarting:
        return "starting"
    case StatusRunning:
        return "running"
    case StatusStopping:
        return "stopping"
    case StatusError:
        return "error"
    default:
        return "unknown"
    }
}
```

### 프로세스 관리자 구현
```go
type claudeProcessManager struct {
    cmd        *exec.Cmd
    config     *ProcessConfig
    status     ProcessStatus
    pid        int
    startTime  time.Time
    mutex      sync.RWMutex
    ctx        context.Context
    cancel     context.CancelFunc
    done       chan error
    logger     *logrus.Logger
}

func NewProcessManager(logger *logrus.Logger) ProcessManager {
    return &claudeProcessManager{
        status: StatusStopped,
        logger: logger,
        done:   make(chan error, 1),
    }
}

func (pm *claudeProcessManager) Start(ctx context.Context, config *ProcessConfig) error {
    pm.mutex.Lock()
    defer pm.mutex.Unlock()
    
    if pm.status != StatusStopped {
        return fmt.Errorf("process is already running or starting")
    }
    
    pm.config = config
    pm.status = StatusStarting
    
    // 컨텍스트 설정
    pm.ctx, pm.cancel = context.WithCancel(ctx)
    
    // 명령어 준비
    pm.cmd = exec.CommandContext(pm.ctx, config.Command, config.Args...)
    pm.cmd.Dir = config.WorkingDir
    
    // 환경 변수 설정
    if len(config.Environment) > 0 {
        env := os.Environ()
        for key, value := range config.Environment {
            env = append(env, fmt.Sprintf("%s=%s", key, value))
        }
        pm.cmd.Env = env
    }
    
    // 프로세스 시작
    if err := pm.cmd.Start(); err != nil {
        pm.status = StatusError
        return fmt.Errorf("failed to start process: %w", err)
    }
    
    pm.pid = pm.cmd.Process.Pid
    pm.startTime = time.Now()
    pm.status = StatusRunning
    
    pm.logger.WithFields(logrus.Fields{
        "pid":     pm.pid,
        "command": config.Command,
        "args":    config.Args,
    }).Info("Process started successfully")
    
    // 비동기 프로세스 모니터링
    go pm.monitor()
    
    return nil
}
```

### 프로세스 상태 관리
```go
func (pm *claudeProcessManager) monitor() {
    defer close(pm.done)
    
    err := pm.cmd.Wait()
    
    pm.mutex.Lock()
    defer pm.mutex.Unlock()
    
    if pm.status == StatusStopping {
        pm.status = StatusStopped
        pm.logger.Info("Process stopped gracefully")
    } else {
        pm.status = StatusError
        pm.logger.WithError(err).Error("Process exited unexpectedly")
    }
    
    pm.done <- err
}

func (pm *claudeProcessManager) Stop(timeout time.Duration) error {
    pm.mutex.Lock()
    defer pm.mutex.Unlock()
    
    if pm.status != StatusRunning {
        return fmt.Errorf("process is not running")
    }
    
    pm.status = StatusStopping
    pm.logger.Info("Initiating graceful shutdown")
    
    // SIGTERM 전송
    if err := pm.cmd.Process.Signal(os.Interrupt); err != nil {
        return fmt.Errorf("failed to send interrupt signal: %w", err)
    }
    
    // 타임아웃 기다리기
    select {
    case err := <-pm.done:
        return err
    case <-time.After(timeout):
        pm.logger.Warn("Graceful shutdown timeout, forcing kill")
        return pm.Kill()
    }
}

func (pm *claudeProcessManager) Kill() error {
    pm.mutex.Lock()
    defer pm.mutex.Unlock()
    
    if pm.status == StatusStopped {
        return nil
    }
    
    if pm.cmd.Process != nil {
        if err := pm.cmd.Process.Kill(); err != nil {
            return fmt.Errorf("failed to kill process: %w", err)
        }
    }
    
    if pm.cancel != nil {
        pm.cancel()
    }
    
    pm.status = StatusStopped
    pm.logger.Info("Process killed")
    
    return nil
}
```

### 헬스체크 및 모니터링
```go
func (pm *claudeProcessManager) HealthCheck() error {
    pm.mutex.RLock()
    defer pm.mutex.RUnlock()
    
    if pm.status != StatusRunning {
        return fmt.Errorf("process is not running (status: %s)", pm.status)
    }
    
    if pm.cmd.Process == nil {
        return fmt.Errorf("process object is nil")
    }
    
    // 프로세스가 실제로 실행 중인지 확인
    if err := pm.cmd.Process.Signal(syscall.Signal(0)); err != nil {
        return fmt.Errorf("process health check failed: %w", err)
    }
    
    return nil
}

func (pm *claudeProcessManager) GetStatus() ProcessStatus {
    pm.mutex.RLock()
    defer pm.mutex.RUnlock()
    return pm.status
}

func (pm *claudeProcessManager) IsRunning() bool {
    return pm.GetStatus() == StatusRunning
}

func (pm *claudeProcessManager) GetPID() int {
    pm.mutex.RLock()
    defer pm.mutex.RUnlock()
    return pm.pid
}

func (pm *claudeProcessManager) Wait() error {
    select {
    case err := <-pm.done:
        return err
    case <-pm.ctx.Done():
        return pm.ctx.Err()
    }
}
```

### 에러 처리 및 로깅
```go
type ProcessError struct {
    Type    string
    Message string
    Cause   error
    PID     int
    Status  ProcessStatus
}

func (e *ProcessError) Error() string {
    if e.Cause != nil {
        return fmt.Sprintf("%s (PID: %d, Status: %s): %s: %v", 
            e.Type, e.PID, e.Status, e.Message, e.Cause)
    }
    return fmt.Sprintf("%s (PID: %d, Status: %s): %s", 
        e.Type, e.PID, e.Status, e.Message)
}

func NewProcessError(errorType, message string, cause error, pid int, status ProcessStatus) *ProcessError {
    return &ProcessError{
        Type:    errorType,
        Message: message,
        Cause:   cause,
        PID:     pid,
        Status:  status,
    }
}
```

### 보안 및 성능 고려사항
- **권한 최소화**: Claude CLI 프로세스의 실행 권한 제한
- **리소스 제한**: 메모리 및 CPU 사용량 모니터링
- **좀비 프로세스 방지**: Wait()를 통한 적절한 프로세스 정리
- **시그널 안전성**: 시그널 핸들러의 안전한 구현

## Output Log
*(This section is populated as work progresses on the task)*