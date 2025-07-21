package claude

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// errorRecoveryManager 에러 복구 관리자 구현
type errorRecoveryManager struct {
	policy          *RecoveryPolicy
	classifier      *ErrorClassifier
	stats           *RecoveryStats
	circuitBreaker  CircuitBreaker
	backoff         BackoffStrategy
	processManager  ProcessManager
	logger          *logrus.Logger
	mutex           sync.RWMutex
	ctx             context.Context
	cancel          context.CancelFunc
	enabled         bool
	
	// 복구 정책 관련
	restartWindow   time.Duration
	restartCounts   []time.Time
	
	// 메트릭 수집 관련
	metricsChannel  chan RecoveryMetric
	metricsEnabled  bool
	
	// 이벤트 핸들러
	onRecoveryStart func(error)
	onRecoveryEnd   func(error, bool)
	onRestart       func(int64)
}

// RecoveryMetric 복구 메트릭
type RecoveryMetric struct {
	Timestamp   time.Time     `json:"timestamp"`
	ErrorType   ErrorType     `json:"error_type"`
	Action      RecoveryAction `json:"action"`
	Success     bool          `json:"success"`
	Duration    time.Duration `json:"duration"`
	Attempt     int           `json:"attempt"`
}

// NewErrorRecoveryManager 새로운 에러 복구 관리자를 생성합니다
func NewErrorRecoveryManager(
	policy *RecoveryPolicy,
	processManager ProcessManager,
	logger *logrus.Logger,
) ErrorRecovery {
	if policy == nil {
		policy = DefaultRecoveryPolicy()
	}
	
	if logger == nil {
		logger = logrus.StandardLogger()
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	manager := &errorRecoveryManager{
		policy:         policy,
		classifier:     NewErrorClassifier(),
		stats:          NewRecoveryStats(),
		circuitBreaker: NewCircuitBreaker(policy.CircuitBreakerConfig, logger),
		backoff:        NewExponentialBackoff(policy),
		processManager: processManager,
		logger:         logger,
		ctx:            ctx,
		cancel:         cancel,
		enabled:        policy.Enabled,
		restartWindow:  1 * time.Hour, // 1시간 윈도우
		restartCounts:  make([]time.Time, 0),
		metricsChannel: make(chan RecoveryMetric, 100),
		metricsEnabled: true,
	}
	
	// 백그라운드 작업 시작
	go manager.startBackgroundTasks()
	
	return manager
}

// HandleError 에러를 처리하고 적절한 복구 액션을 반환합니다
func (erm *errorRecoveryManager) HandleError(err error) RecoveryAction {
	if !erm.enabled {
		return ActionIgnore
	}
	
	if err == nil {
		return ActionIgnore
	}
	
	erm.mutex.Lock()
	defer erm.mutex.Unlock()
	
	startTime := time.Now()
	
	// 통계 업데이트
	erm.stats.TotalErrors++
	erm.stats.LastError = err
	
	// 에러 분류
	errorType, action := erm.classifier.ClassifyError(err)
	erm.stats.IncrementError(errorType)
	
	erm.logger.WithFields(logrus.Fields{
		"error":      err.Error(),
		"error_type": errorType,
		"action":     action,
		"attempt":    erm.stats.TotalErrors,
	}).Warn("에러가 감지되어 복구 액션을 결정합니다")
	
	// Circuit Breaker 상태 확인
	if erm.circuitBreaker.IsOpen() {
		erm.logger.Warn("Circuit breaker가 열린 상태이므로 복구 시도를 차단합니다")
		action = ActionFail
	}
	
	// 액션에 따른 처리
	finalAction := erm.processAction(action, errorType, err)
	
	// 통계 업데이트
	erm.stats.IncrementAction(finalAction)
	
	// 메트릭 수집
	if erm.metricsEnabled {
		metric := RecoveryMetric{
			Timestamp: startTime,
			ErrorType: errorType,
			Action:    finalAction,
			Success:   finalAction != ActionFail,
			Duration:  time.Since(startTime),
			Attempt:   int(erm.stats.TotalErrors),
		}
		
		select {
		case erm.metricsChannel <- metric:
		default:
			erm.logger.Warn("메트릭 채널이 가득 참, 메트릭 드롭")
		}
	}
	
	return finalAction
}

// processAction 액션을 처리하고 최종 액션을 반환합니다
func (erm *errorRecoveryManager) processAction(action RecoveryAction, errorType ErrorType, err error) RecoveryAction {
	switch action {
	case ActionRestart:
		if erm.shouldAllowRestart() {
			return ActionRestart
		}
		erm.logger.Error("재시작 제한에 도달하여 실패로 처리합니다")
		return ActionFail
		
	case ActionRetry:
		if erm.circuitBreaker.Allow() {
			return ActionRetry
		}
		erm.logger.Warn("Circuit breaker에 의해 재시도가 차단되었습니다")
		return ActionCircuitBreak
		
	case ActionCircuitBreak:
		erm.circuitBreaker.RecordError()
		return ActionCircuitBreak
		
	default:
		return action
	}
}

// shouldAllowRestart 재시작을 허용할지 확인합니다
func (erm *errorRecoveryManager) shouldAllowRestart() bool {
	// 최대 재시작 횟수 확인
	if erm.stats.RestartCount >= int64(erm.policy.MaxRestarts) {
		erm.logger.WithFields(logrus.Fields{
			"restart_count": erm.stats.RestartCount,
			"max_restarts":  erm.policy.MaxRestarts,
		}).Error("최대 재시작 횟수에 도달했습니다")
		return false
	}
	
	// 마지막 재시작 이후 시간 확인
	if !erm.stats.LastRestart.IsZero() {
		timeSinceLastRestart := time.Since(erm.stats.LastRestart)
		minInterval := erm.backoff.NextBackoff()
		
		if timeSinceLastRestart < minInterval {
			erm.logger.WithFields(logrus.Fields{
				"time_since_last": timeSinceLastRestart,
				"min_interval":    minInterval,
			}).Warn("재시작이 너무 빨리 시도되었습니다")
			return false
		}
	}
	
	// 시간 윈도우 내 재시작 횟수 확인
	now := time.Now()
	windowStart := now.Add(-erm.restartWindow)
	
	// 오래된 재시작 기록 제거
	validRestarts := make([]time.Time, 0)
	for _, restartTime := range erm.restartCounts {
		if restartTime.After(windowStart) {
			validRestarts = append(validRestarts, restartTime)
		}
	}
	erm.restartCounts = validRestarts
	
	// 윈도우 내 재시작 횟수가 제한을 초과하는지 확인
	windowRestartLimit := erm.policy.MaxRestarts * 2 // 윈도우 내에서는 좀 더 관대하게
	if len(erm.restartCounts) >= windowRestartLimit {
		erm.logger.WithFields(logrus.Fields{
			"window_restarts": len(erm.restartCounts),
			"window_limit":    windowRestartLimit,
			"window_duration": erm.restartWindow,
		}).Error("시간 윈도우 내 재시작 제한에 도달했습니다")
		return false
	}
	
	return true
}

// ShouldRestart 재시작해야 하는지 확인합니다
func (erm *errorRecoveryManager) ShouldRestart(err error) bool {
	if !erm.enabled {
		return false
	}
	
	action := erm.HandleError(err)
	return action == ActionRestart
}

// Restart 프로세스를 재시작합니다
func (erm *errorRecoveryManager) Restart(ctx context.Context) error {
	if !erm.enabled {
		return fmt.Errorf("에러 복구가 비활성화되어 있습니다")
	}
	
	erm.mutex.Lock()
	defer erm.mutex.Unlock()
	
	erm.logger.Info("프로세스 재시작을 시작합니다")
	
	if erm.onRecoveryStart != nil {
		erm.onRecoveryStart(erm.stats.LastError)
	}
	
	startTime := time.Now()
	
	// 현재 프로세스 중지
	if erm.processManager.IsRunning() {
		erm.logger.Info("현재 실행 중인 프로세스를 중지합니다")
		
		if err := erm.processManager.Stop(10 * time.Second); err != nil {
			erm.logger.WithError(err).Warn("정상 종료 실패, 강제 종료를 시도합니다")
			if err := erm.processManager.Kill(); err != nil {
				erm.circuitBreaker.RecordError()
				return fmt.Errorf("프로세스 강제 종료 실패: %w", err)
			}
		}
	}
	
	// 백오프 대기
	backoffDuration := erm.backoff.NextBackoff()
	erm.logger.WithField("backoff_duration", backoffDuration).Info("재시작 전 대기 중")
	
	select {
	case <-time.After(backoffDuration):
		// 정상적으로 대기 완료
	case <-ctx.Done():
		return ctx.Err()
	}
	
	// 프로세스 재시작
	config := &ProcessConfig{
		Command:    "claude",
		Args:       []string{"--workspace", "/tmp/workspace"},
		WorkingDir: "/tmp/workspace",
		Timeout:    30 * time.Second,
	}
	
	if err := erm.processManager.Start(ctx, config); err != nil {
		erm.circuitBreaker.RecordError()
		duration := time.Since(startTime)
		
		erm.logger.WithFields(logrus.Fields{
			"error":    err,
			"duration": duration,
		}).Error("프로세스 재시작 실패")
		
		if erm.onRecoveryEnd != nil {
			erm.onRecoveryEnd(err, false)
		}
		
		return fmt.Errorf("프로세스 재시작 실패: %w", err)
	}
	
	// 성공 처리
	erm.stats.IncrementRestart()
	erm.restartCounts = append(erm.restartCounts, time.Now())
	erm.circuitBreaker.RecordSuccess()
	erm.backoff.Reset() // 성공 시 백오프 리셋
	
	duration := time.Since(startTime)
	
	erm.logger.WithFields(logrus.Fields{
		"restart_count": erm.stats.RestartCount,
		"duration":      duration,
	}).Info("프로세스가 성공적으로 재시작되었습니다")
	
	if erm.onRestart != nil {
		erm.onRestart(erm.stats.RestartCount)
	}
	
	if erm.onRecoveryEnd != nil {
		erm.onRecoveryEnd(nil, true)
	}
	
	return nil
}

// GetRecoveryStats 복구 통계를 반환합니다
func (erm *errorRecoveryManager) GetRecoveryStats() *RecoveryStats {
	erm.mutex.RLock()
	defer erm.mutex.RUnlock()
	
	// 깊은 복사를 통해 안전한 통계 반환
	stats := &RecoveryStats{
		TotalErrors:   erm.stats.TotalErrors,
		RestartCount:  erm.stats.RestartCount,
		LastError:     erm.stats.LastError,
		LastRestart:   erm.stats.LastRestart,
		SuccessfulRuns: erm.stats.SuccessfulRuns,
		AverageUptime: erm.calculateAverageUptime(),
		ErrorsByType:  make(map[ErrorType]int64),
		ActionsByType: make(map[RecoveryAction]int64),
	}
	
	// 맵 복사
	for k, v := range erm.stats.ErrorsByType {
		stats.ErrorsByType[k] = v
	}
	
	for k, v := range erm.stats.ActionsByType {
		stats.ActionsByType[k] = v
	}
	
	return stats
}

// calculateAverageUptime 평균 가동 시간을 계산합니다
func (erm *errorRecoveryManager) calculateAverageUptime() time.Duration {
	if erm.stats.RestartCount == 0 {
		return 0
	}
	
	// 간단한 계산: 전체 실행 시간 / 재시작 횟수
	// 실제로는 더 정교한 계산이 필요할 수 있음
	if !erm.stats.LastRestart.IsZero() {
		totalRunTime := time.Since(erm.stats.LastRestart)
		return totalRunTime / time.Duration(erm.stats.RestartCount)
	}
	
	return 0
}

// SetRecoveryPolicy 복구 정책을 설정합니다
func (erm *errorRecoveryManager) SetRecoveryPolicy(policy *RecoveryPolicy) {
	if policy == nil {
		return
	}
	
	erm.mutex.Lock()
	defer erm.mutex.Unlock()
	
	erm.policy = policy
	erm.enabled = policy.Enabled
	
	// 백오프 전략 업데이트
	erm.backoff = NewExponentialBackoff(policy)
	
	// Circuit Breaker 업데이트
	if policy.CircuitBreakerConfig != nil {
		erm.circuitBreaker = NewCircuitBreaker(policy.CircuitBreakerConfig, erm.logger)
	}
	
	erm.logger.WithField("policy", policy).Info("복구 정책이 업데이트되었습니다")
}

// Start 에러 복구 시스템을 시작합니다
func (erm *errorRecoveryManager) Start(ctx context.Context) error {
	erm.mutex.Lock()
	defer erm.mutex.Unlock()
	
	if erm.enabled {
		erm.logger.Info("에러 복구 시스템이 이미 활성화되어 있습니다")
		return nil
	}
	
	erm.enabled = true
	erm.ctx, erm.cancel = context.WithCancel(ctx)
	
	// 백그라운드 작업 재시작
	go erm.startBackgroundTasks()
	
	erm.logger.Info("에러 복구 시스템이 시작되었습니다")
	return nil
}

// Stop 에러 복구 시스템을 중지합니다
func (erm *errorRecoveryManager) Stop() error {
	erm.mutex.Lock()
	defer erm.mutex.Unlock()
	
	if !erm.enabled {
		return nil
	}
	
	erm.enabled = false
	
	if erm.cancel != nil {
		erm.cancel()
	}
	
	erm.logger.Info("에러 복구 시스템이 중지되었습니다")
	return nil
}

// IsEnabled 에러 복구 시스템이 활성화되어 있는지 확인합니다
func (erm *errorRecoveryManager) IsEnabled() bool {
	erm.mutex.RLock()
	defer erm.mutex.RUnlock()
	return erm.enabled
}

// startBackgroundTasks 백그라운드 작업들을 시작합니다
func (erm *errorRecoveryManager) startBackgroundTasks() {
	// 메트릭 수집기
	go erm.metricsCollector()
	
	// 통계 로깅
	go erm.statsLogger()
	
	// 헬스체크
	go erm.healthChecker()
}

// metricsCollector 메트릭을 수집합니다
func (erm *errorRecoveryManager) metricsCollector() {
	for {
		select {
		case metric := <-erm.metricsChannel:
			// 메트릭 처리 (예: 외부 모니터링 시스템으로 전송)
			erm.logger.WithFields(logrus.Fields{
				"timestamp":   metric.Timestamp,
				"error_type":  metric.ErrorType,
				"action":      metric.Action,
				"success":     metric.Success,
				"duration":    metric.Duration,
				"attempt":     metric.Attempt,
			}).Debug("복구 메트릭 수집됨")
			
		case <-erm.ctx.Done():
			return
		}
	}
}

// statsLogger 주기적으로 통계를 로깅합니다
func (erm *errorRecoveryManager) statsLogger() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			stats := erm.GetRecoveryStats()
			cbStats := erm.circuitBreaker.Stats()
			
			erm.logger.WithFields(logrus.Fields{
				"total_errors":     stats.TotalErrors,
				"restart_count":    stats.RestartCount,
				"successful_runs":  stats.SuccessfulRuns,
				"average_uptime":   stats.AverageUptime,
				"circuit_state":    cbStats.State,
				"circuit_failures": cbStats.FailureCount,
			}).Info("에러 복구 시스템 통계")
			
		case <-erm.ctx.Done():
			return
		}
	}
}

// healthChecker 주기적으로 프로세스 상태를 확인합니다
func (erm *errorRecoveryManager) healthChecker() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			if erm.processManager.IsRunning() {
				if err := erm.processManager.HealthCheck(); err != nil {
					erm.logger.WithError(err).Warn("프로세스 헬스체크 실패")
					erm.HandleError(err)
				} else {
					erm.stats.IncrementSuccessfulRun()
					erm.circuitBreaker.RecordSuccess()
				}
			}
			
		case <-erm.ctx.Done():
			return
		}
	}
}

// SetEventHandlers 이벤트 핸들러를 설정합니다
func (erm *errorRecoveryManager) SetEventHandlers(
	onRecoveryStart func(error),
	onRecoveryEnd func(error, bool),
	onRestart func(int64),
) {
	erm.mutex.Lock()
	defer erm.mutex.Unlock()
	
	erm.onRecoveryStart = onRecoveryStart
	erm.onRecoveryEnd = onRecoveryEnd
	erm.onRestart = onRestart
}