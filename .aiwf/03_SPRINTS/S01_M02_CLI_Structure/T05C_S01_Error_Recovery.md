---
task_id: T05C_S01
sprint_sequence_id: S01_M02
status: completed
complexity: Medium
last_updated: 2025-07-21T09:15:00Z
github_issue: # Optional: GitHub issue number
---

# Task: Claude CLI 에러 복구 및 재시작 메커니즘 구현

## Description
Claude CLI 프로세스의 에러 감지, 복구, 자동 재시작 메커니즘을 구현합니다. 시스템 안정성과 가용성을 확보하기 위한 포괄적인 장애 대응 시스템을 구축합니다.

## Goal / Objectives
- 에러 감지 및 분류 시스템 구현
- 자동 재시작 메커니즘 구현
- 상태 복구 및 세션 관리
- 로깅 및 모니터링 통합

## Acceptance Criteria
- [x] 에러 감지 및 분류 시스템 구현
- [x] 자동 재시작 정책 및 백오프 전략 구현
- [x] 세션 상태 복구 메커니즘 구현
- [x] Circuit Breaker 패턴 구현
- [x] 에러 메트릭 수집 및 알림 시스템
- [x] 복구 이력 추적 및 분석

## Subtasks
- [x] 에러 감지 시스템 구현
- [x] 재시작 정책 및 백오프 전략
- [x] 상태 복구 메커니즘 구현
- [x] Circuit Breaker 구현
- [x] 메트릭 수집 및 모니터링
- [x] 복구 테스트 시나리오 작성

## 기술 가이드

### 에러 복구 인터페이스
```go
type ErrorRecovery interface {
    HandleError(err error) RecoveryAction
    ShouldRestart(err error) bool
    Restart(ctx context.Context) error
    GetRecoveryStats() *RecoveryStats
    SetRecoveryPolicy(policy *RecoveryPolicy)
}

type RecoveryAction int

const (
    ActionIgnore RecoveryAction = iota
    ActionRetry
    ActionRestart
    ActionFail
    ActionCircuitBreak
)

type RecoveryPolicy struct {
    MaxRestarts     int           `yaml:"max_restarts"`
    RestartInterval time.Duration `yaml:"restart_interval"`
    BackoffMultiplier float64     `yaml:"backoff_multiplier"`
    MaxBackoff      time.Duration `yaml:"max_backoff"`
    CircuitBreakerConfig *CircuitBreakerConfig `yaml:"circuit_breaker"`
}

type RecoveryStats struct {
    TotalErrors     int64     `json:"total_errors"`
    RestartCount    int64     `json:"restart_count"`
    LastError       error     `json:"last_error"`
    LastRestart     time.Time `json:"last_restart"`
    SuccessfulRuns  int64     `json:"successful_runs"`
    AverageUptime   time.Duration `json:"average_uptime"`
}
```

### 에러 감지 및 분류 시스템
```go
type ErrorClassifier struct {
    rules map[ErrorType]ClassificationRule
}

type ErrorType int

const (
    ErrorTypeUnknown ErrorType = iota
    ErrorTypeTransient      // 일시적 오류 (네트워크, 타임아웃)
    ErrorTypePermanent      // 영구적 오류 (설정, 권한)
    ErrorTypeProcess        // 프로세스 관련 오류
    ErrorTypeResource       // 리소스 부족
    ErrorTypeAPI           // API 오류
)

type ClassificationRule struct {
    ErrorPattern string
    Action       RecoveryAction
    Retryable    bool
    BackoffType  BackoffType
}

type BackoffType int

const (
    BackoffFixed BackoffType = iota
    BackoffExponential
    BackoffLinear
)

func (ec *ErrorClassifier) ClassifyError(err error) (ErrorType, RecoveryAction) {
    errStr := err.Error()
    
    // 패턴 매칭을 통한 에러 분류
    switch {
    case strings.Contains(errStr, "connection refused"),
         strings.Contains(errStr, "timeout"):
        return ErrorTypeTransient, ActionRetry
        
    case strings.Contains(errStr, "permission denied"),
         strings.Contains(errStr, "invalid api key"):
        return ErrorTypePermanent, ActionFail
        
    case strings.Contains(errStr, "process exited"),
         strings.Contains(errStr, "signal"):
        return ErrorTypeProcess, ActionRestart
        
    case strings.Contains(errStr, "out of memory"),
         strings.Contains(errStr, "resource limit"):
        return ErrorTypeResource, ActionCircuitBreak
        
    default:
        return ErrorTypeUnknown, ActionIgnore
    }
}
```

### 에러 복구 관리자 구현
```go
type errorRecoveryManager struct {
    policy       *RecoveryPolicy
    classifier   *ErrorClassifier
    stats        *RecoveryStats
    circuitBreaker *CircuitBreaker
    backoff      BackoffStrategy
    processManager ProcessManager
    logger       *logrus.Logger
    mutex        sync.RWMutex
}

func NewErrorRecoveryManager(
    policy *RecoveryPolicy,
    processManager ProcessManager,
    logger *logrus.Logger,
) ErrorRecovery {
    return &errorRecoveryManager{
        policy:         policy,
        classifier:     NewErrorClassifier(),
        stats:          &RecoveryStats{},
        circuitBreaker: NewCircuitBreaker(policy.CircuitBreakerConfig),
        backoff:        NewExponentialBackoff(policy),
        processManager: processManager,
        logger:         logger,
    }
}

func (erm *errorRecoveryManager) HandleError(err error) RecoveryAction {
    erm.mutex.Lock()
    defer erm.mutex.Unlock()
    
    erm.stats.TotalErrors++
    erm.stats.LastError = err
    
    errorType, action := erm.classifier.ClassifyError(err)
    
    erm.logger.WithFields(logrus.Fields{
        "error":      err.Error(),
        "error_type": errorType,
        "action":     action,
    }).Warn("Error detected, determining recovery action")
    
    // Circuit Breaker 상태 확인
    if erm.circuitBreaker.State() == StateOpen {
        erm.logger.Warn("Circuit breaker is open, blocking recovery attempts")
        return ActionFail
    }
    
    switch action {
    case ActionRestart:
        if erm.shouldAllowRestart() {
            return ActionRestart
        }
        return ActionFail
    case ActionRetry:
        if erm.circuitBreaker.Allow() {
            return ActionRetry
        }
        return ActionCircuitBreak
    default:
        return action
    }
}

func (erm *errorRecoveryManager) shouldAllowRestart() bool {
    if erm.stats.RestartCount >= int64(erm.policy.MaxRestarts) {
        erm.logger.Error("Maximum restart limit reached")
        return false
    }
    
    timeSinceLastRestart := time.Since(erm.stats.LastRestart)
    minInterval := erm.backoff.NextBackoff()
    
    if timeSinceLastRestart < minInterval {
        erm.logger.WithFields(logrus.Fields{
            "time_since_last": timeSinceLastRestart,
            "min_interval":    minInterval,
        }).Warn("Restart attempted too soon")
        return false
    }
    
    return true
}
```

### 자동 재시작 메커니즘
```go
func (erm *errorRecoveryManager) Restart(ctx context.Context) error {
    erm.mutex.Lock()
    defer erm.mutex.Unlock()
    
    erm.logger.Info("Initiating process restart")
    
    // 현재 프로세스 중지
    if erm.processManager.IsRunning() {
        if err := erm.processManager.Stop(10 * time.Second); err != nil {
            erm.logger.WithError(err).Warn("Graceful stop failed, forcing kill")
            if err := erm.processManager.Kill(); err != nil {
                return fmt.Errorf("failed to kill process: %w", err)
            }
        }
    }
    
    // 백오프 대기
    backoffDuration := erm.backoff.NextBackoff()
    erm.logger.WithField("backoff", backoffDuration).Info("Waiting before restart")
    
    select {
    case <-time.After(backoffDuration):
    case <-ctx.Done():
        return ctx.Err()
    }
    
    // 프로세스 재시작
    config := &ProcessConfig{
        Command:    "claude",
        Args:       []string{"--workspace", "/tmp/workspace"},
        WorkingDir: "/tmp/workspace",
    }
    
    if err := erm.processManager.Start(ctx, config); err != nil {
        erm.circuitBreaker.RecordError()
        return fmt.Errorf("failed to restart process: %w", err)
    }
    
    erm.stats.RestartCount++
    erm.stats.LastRestart = time.Now()
    erm.circuitBreaker.RecordSuccess()
    
    erm.logger.WithField("restart_count", erm.stats.RestartCount).Info("Process restarted successfully")
    
    return nil
}
```

### Circuit Breaker 구현
```go
type CircuitBreakerState int

const (
    StateClosed CircuitBreakerState = iota
    StateOpen
    StateHalfOpen
)

type CircuitBreaker struct {
    config        *CircuitBreakerConfig
    state         CircuitBreakerState
    failureCount  int64
    successCount  int64
    lastFailure   time.Time
    nextAttempt   time.Time
    mutex         sync.RWMutex
}

type CircuitBreakerConfig struct {
    FailureThreshold int           `yaml:"failure_threshold"`
    RecoveryTimeout  time.Duration `yaml:"recovery_timeout"`
    SuccessThreshold int           `yaml:"success_threshold"`
}

func NewCircuitBreaker(config *CircuitBreakerConfig) *CircuitBreaker {
    return &CircuitBreaker{
        config: config,
        state:  StateClosed,
    }
}

func (cb *CircuitBreaker) Allow() bool {
    cb.mutex.RLock()
    defer cb.mutex.RUnlock()
    
    switch cb.state {
    case StateClosed:
        return true
    case StateOpen:
        return time.Now().After(cb.nextAttempt)
    case StateHalfOpen:
        return true
    default:
        return false
    }
}

func (cb *CircuitBreaker) RecordSuccess() {
    cb.mutex.Lock()
    defer cb.mutex.Unlock()
    
    cb.successCount++
    
    if cb.state == StateHalfOpen {
        if cb.successCount >= int64(cb.config.SuccessThreshold) {
            cb.state = StateClosed
            cb.failureCount = 0
            cb.successCount = 0
        }
    }
}

func (cb *CircuitBreaker) RecordError() {
    cb.mutex.Lock()
    defer cb.mutex.Unlock()
    
    cb.failureCount++
    cb.lastFailure = time.Now()
    
    if cb.failureCount >= int64(cb.config.FailureThreshold) {
        cb.state = StateOpen
        cb.nextAttempt = time.Now().Add(cb.config.RecoveryTimeout)
    }
}

func (cb *CircuitBreaker) State() CircuitBreakerState {
    cb.mutex.RLock()
    defer cb.mutex.RUnlock()
    return cb.state
}
```

### 백오프 전략
```go
type BackoffStrategy interface {
    NextBackoff() time.Duration
    Reset()
}

type ExponentialBackoff struct {
    initial    time.Duration
    multiplier float64
    max        time.Duration
    current    time.Duration
    attempts   int
}

func NewExponentialBackoff(policy *RecoveryPolicy) BackoffStrategy {
    return &ExponentialBackoff{
        initial:    policy.RestartInterval,
        multiplier: policy.BackoffMultiplier,
        max:        policy.MaxBackoff,
        current:    policy.RestartInterval,
    }
}

func (eb *ExponentialBackoff) NextBackoff() time.Duration {
    if eb.attempts == 0 {
        eb.attempts++
        return eb.initial
    }
    
    eb.current = time.Duration(float64(eb.current) * eb.multiplier)
    if eb.current > eb.max {
        eb.current = eb.max
    }
    
    eb.attempts++
    return eb.current
}

func (eb *ExponentialBackoff) Reset() {
    eb.current = eb.initial
    eb.attempts = 0
}
```

### 메트릭 수집 및 모니터링
```go
func (erm *errorRecoveryManager) collectMetrics() {
    ticker := time.NewTicker(1 * time.Minute)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            erm.mutex.RLock()
            stats := *erm.stats
            erm.mutex.RUnlock()
            
            // 메트릭 로깅
            erm.logger.WithFields(logrus.Fields{
                "total_errors":   stats.TotalErrors,
                "restart_count":  stats.RestartCount,
                "success_runs":   stats.SuccessfulRuns,
                "average_uptime": stats.AverageUptime,
            }).Info("Recovery metrics")
            
        case <-erm.ctx.Done():
            return
        }
    }
}
```

## Output Log

### 2025-07-21 - 에러 복구 시스템 구현 완료

#### 구현된 주요 컴포넌트:

1. **에러 분류 및 복구 시스템** (`error_recovery.go`)
   - `ErrorRecovery` 인터페이스 정의
   - `ErrorClassifier` - 패턴 매칭을 통한 에러 분류
   - 5가지 에러 타입 지원 (Transient, Permanent, Process, Resource, API)
   - 5가지 복구 액션 지원 (Ignore, Retry, Restart, Fail, CircuitBreak)

2. **백오프 전략** (`backoff.go`)
   - `BackoffStrategy` 인터페이스
   - 지수 백오프 (ExponentialBackoff)
   - 선형 백오프 (LinearBackoff)  
   - 고정 백오프 (FixedBackoff)
   - 지터 지원 (JitteredExponentialBackoff)

3. **Circuit Breaker 패턴** (`circuit_breaker.go`)
   - 3가지 상태 지원 (Closed, Open, Half-Open)
   - 슬라이딩 윈도우 기반 에러율 계산
   - 구성 가능한 임계값 (실패 횟수, 에러 비율, 복구 타임아웃)
   - 통계 수집 및 모니터링

4. **에러 복구 관리자** (`error_recovery_manager.go`)
   - 통합 에러 복구 관리
   - 백그라운드 헬스체크 및 메트릭 수집
   - 이벤트 핸들러 지원
   - 재시작 제한 및 윈도우 기반 제어
   - 실시간 통계 추적

5. **포괄적인 테스트** (`error_recovery_test.go`)
   - 단위 테스트 및 통합 테스트
   - 모킹을 통한 격리된 테스트
   - 다양한 에러 시나리오 검증
   - 백오프 및 Circuit Breaker 동작 검증

6. **사용 예시** (`examples/error_recovery_example.go`)
   - 실제 사용법 데모
   - 이벤트 핸들러 설정 예시
   - 모니터링 통합 가이드
   - 커스텀 설정 방법

#### 주요 기능:

- **지능형 에러 분류**: 에러 메시지 패턴 매칭을 통한 자동 분류
- **적응적 백오프**: 상황에 맞는 다양한 백오프 전략
- **Circuit Breaker**: 장애 전파 방지 및 자동 복구
- **실시간 모니터링**: 메트릭 수집 및 통계 추적
- **유연한 정책**: 런타임 정책 변경 지원
- **이벤트 기반**: 복구 과정의 각 단계별 이벤트 핸들링

#### 설정 가능한 정책:

- 최대 재시작 횟수
- 재시작 간격 및 백오프 배수
- Circuit Breaker 임계값
- 에러 비율 임계값
- 복구 타임아웃

#### 통계 및 모니터링:

- 총 에러 수 및 타입별 분류
- 재시작 횟수 및 성공률
- 평균 가동 시간
- Circuit Breaker 상태 변화
- 복구 액션별 통계

**태스크 상태**: ✅ 완료 (2025-07-21)
**구현 완료도**: 100%
**테스트 커버리지**: 주요 시나리오 완료