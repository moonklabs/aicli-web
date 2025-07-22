package claude

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// RecoveryOrchestrator는 복구 작업을 총괄 관리합니다
type RecoveryOrchestrator struct {
	// 복구 전략들
	strategies    map[ErrorType][]RecoveryStrategy
	strategiesMutex sync.RWMutex
	
	// 핵심 컴포넌트들
	processManager *ProcessManager
	sessionManager SessionManager
	healthChecker  *HealthChecker
	alertManager   *AlertManager
	
	// 에러 분류기
	classifier ErrorClassifier
	
	// 활성 복구 작업들
	activeRecoveries map[string]*RecoveryExecution
	activeMutex      sync.RWMutex
	
	// 복구 히스토리
	recoveryHistory []RecoveryRecord
	historyMutex    sync.RWMutex
	maxHistory      int
	
	// 설정
	config RecoveryConfig
	
	// 생명주기
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// RecoveryStrategy는 복구 전략 인터페이스입니다
type RecoveryStrategy interface {
	// 복구 가능 여부 확인
	CanRecover(ctx context.Context, err error) bool
	
	// 복구 실행
	Execute(ctx context.Context, target RecoveryTarget) error
	
	// 예상 시간
	GetEstimatedTime() time.Duration
	
	// 성공률
	GetSuccessRate() float64
	
	// 전략 이름
	GetName() string
	
	// 우선순위
	GetPriority() int
	
	// 전제 조건
	GetPrerequisites() []string
}

// RecoveryTarget는 복구 대상입니다
type RecoveryTarget struct {
	Type       string                 `json:"type"`        // "process", "session", "resource"
	Identifier string                 `json:"identifier"`  // ID 또는 이름
	Context    map[string]interface{} `json:"context"`     // 추가 컨텍스트
	Priority   RecoveryPriority       `json:"priority"`    // 복구 우선순위
}

// RecoveryPriority는 복구 우선순위입니다
type RecoveryPriority int

const (
	RecoveryPriorityLow RecoveryPriority = iota
	RecoveryPriorityMedium
	RecoveryPriorityHigh
	RecoveryPriorityCritical
	RecoveryPriorityEmergency
)

// RecoveryExecution은 진행 중인 복구 작업입니다
type RecoveryExecution struct {
	ID          string           `json:"id"`
	Target      RecoveryTarget   `json:"target"`
	Strategy    RecoveryStrategy `json:"-"`
	Status      RecoveryStatus   `json:"status"`
	StartTime   time.Time        `json:"start_time"`
	EndTime     *time.Time       `json:"end_time,omitempty"`
	Progress    float64          `json:"progress"`
	Error       string           `json:"error,omitempty"`
	Steps       []RecoveryStep   `json:"steps"`
	
	// 실행 컨텍스트
	ctx    context.Context
	cancel context.CancelFunc
}

// RecoveryStatus는 복구 상태입니다
type RecoveryStatus int

const (
	RecoveryStatusPending RecoveryStatus = iota
	RecoveryStatusRunning
	RecoveryStatusCompleted
	RecoveryStatusFailed
	RecoveryStatusCancelled
	RecoveryStatusTimedOut
)

// RecoveryStep은 복구 단계입니다
type RecoveryStep struct {
	Name        string             `json:"name"`
	Status      RecoveryStepStatus `json:"status"`
	StartTime   time.Time          `json:"start_time"`
	EndTime     *time.Time         `json:"end_time,omitempty"`
	Duration    time.Duration      `json:"duration"`
	Error       string             `json:"error,omitempty"`
	Description string             `json:"description"`
}

// RecoveryStepStatus는 복구 단계 상태입니다
type RecoveryStepStatus int

const (
	StepStatusPending RecoveryStepStatus = iota
	StepStatusRunning
	StepStatusCompleted
	StepStatusFailed
	StepStatusSkipped
)

// RecoveryRecord는 복구 기록입니다
type RecoveryRecord struct {
	ExecutionID   string         `json:"execution_id"`
	Target        RecoveryTarget `json:"target"`
	StrategyName  string         `json:"strategy_name"`
	Status        RecoveryStatus `json:"status"`
	StartTime     time.Time      `json:"start_time"`
	Duration      time.Duration  `json:"duration"`
	Error         string         `json:"error,omitempty"`
	SuccessRate   float64        `json:"success_rate"`
}

// RecoveryConfig는 복구 설정입니다
type RecoveryConfig struct {
	MaxConcurrentRecoveries int           `json:"max_concurrent_recoveries"`
	DefaultTimeout          time.Duration `json:"default_timeout"`
	RetryAttempts           int           `json:"retry_attempts"`
	RetryDelay              time.Duration `json:"retry_delay"`
	HealthCheckInterval     time.Duration `json:"health_check_interval"`
	AlertThreshold          int           `json:"alert_threshold"`
	HistoryRetentionPeriod  time.Duration `json:"history_retention_period"`
}

// AlertManager는 알림 관리 인터페이스입니다
type AlertManager interface {
	SendAlert(level AlertLevel, message string, context map[string]interface{}) error
}

// AlertLevel은 알림 레벨입니다
type AlertLevel int

const (
	AlertLevelInfo AlertLevel = iota
	AlertLevelWarning
	AlertLevelError
	AlertLevelCritical
)

// NewRecoveryOrchestrator는 새로운 복구 오케스트레이터를 생성합니다
func NewRecoveryOrchestrator(
	processManager *ProcessManager,
	sessionManager SessionManager,
	healthChecker *HealthChecker,
	alertManager *AlertManager,
	classifier ErrorClassifier,
	config RecoveryConfig,
) *RecoveryOrchestrator {
	ctx, cancel := context.WithCancel(context.Background())
	
	orchestrator := &RecoveryOrchestrator{
		strategies:       make(map[ErrorType][]RecoveryStrategy),
		processManager:   processManager,
		sessionManager:   sessionManager,
		healthChecker:    healthChecker,
		alertManager:     alertManager,
		classifier:       classifier,
		activeRecoveries: make(map[string]*RecoveryExecution),
		recoveryHistory:  make([]RecoveryRecord, 0),
		maxHistory:       1000,
		config:           config,
		ctx:              ctx,
		cancel:           cancel,
	}
	
	// 기본 복구 전략 등록
	orchestrator.registerDefaultStrategies()
	
	// 백그라운드 작업 시작
	orchestrator.wg.Add(2)
	go orchestrator.healthMonitor()
	go orchestrator.historyCleanup()
	
	return orchestrator
}

// RegisterStrategy는 복구 전략을 등록합니다
func (ro *RecoveryOrchestrator) RegisterStrategy(errorType ErrorType, strategy RecoveryStrategy) {
	ro.strategiesMutex.Lock()
	defer ro.strategiesMutex.Unlock()
	
	if ro.strategies[errorType] == nil {
		ro.strategies[errorType] = make([]RecoveryStrategy, 0)
	}
	
	ro.strategies[errorType] = append(ro.strategies[errorType], strategy)
	
	// 우선순위 순으로 정렬
	strategies := ro.strategies[errorType]
	for i := 0; i < len(strategies); i++ {
		for j := i + 1; j < len(strategies); j++ {
			if strategies[i].GetPriority() < strategies[j].GetPriority() {
				strategies[i], strategies[j] = strategies[j], strategies[i]
			}
		}
	}
}

// RecoverFromError는 에러로부터 복구를 시작합니다
func (ro *RecoveryOrchestrator) RecoverFromError(ctx context.Context, err error, target RecoveryTarget) (*RecoveryExecution, error) {
	if err == nil {
		return nil, fmt.Errorf("error cannot be nil")
	}
	
	// 에러 분류
	errorClass := ro.classifier.ClassifyError(err)
	
	// 적절한 복구 전략 선택
	strategy := ro.selectRecoveryStrategy(ctx, errorClass.Type, err)
	if strategy == nil {
		return nil, fmt.Errorf("no suitable recovery strategy found for error type: %v", errorClass.Type)
	}
	
	// 동시 복구 작업 수 제한 확인
	if ro.getActiveRecoveryCount() >= ro.config.MaxConcurrentRecoveries {
		return nil, fmt.Errorf("maximum concurrent recoveries exceeded")
	}
	
	// 복구 실행 생성
	execution := ro.createRecoveryExecution(target, strategy)
	
	// 복구 시작
	go ro.executeRecovery(execution)
	
	// 알림 발송
	if ro.alertManager != nil {
		ro.alertManager.SendAlert(AlertLevelWarning, 
			fmt.Sprintf("Recovery started for %s", target.Type),
			map[string]interface{}{
				"target":   target,
				"strategy": strategy.GetName(),
				"error":    err.Error(),
			})
	}
	
	return execution, nil
}

// GetActiveRecoveries는 진행 중인 복구 작업들을 반환합니다
func (ro *RecoveryOrchestrator) GetActiveRecoveries() []*RecoveryExecution {
	ro.activeMutex.RLock()
	defer ro.activeMutex.RUnlock()
	
	executions := make([]*RecoveryExecution, 0, len(ro.activeRecoveries))
	for _, execution := range ro.activeRecoveries {
		executions = append(executions, execution)
	}
	
	return executions
}

// GetRecoveryHistory는 복구 히스토리를 반환합니다
func (ro *RecoveryOrchestrator) GetRecoveryHistory() []RecoveryRecord {
	ro.historyMutex.RLock()
	defer ro.historyMutex.RUnlock()
	
	history := make([]RecoveryRecord, len(ro.recoveryHistory))
	copy(history, ro.recoveryHistory)
	
	return history
}

// CancelRecovery는 복구 작업을 취소합니다
func (ro *RecoveryOrchestrator) CancelRecovery(executionID string) error {
	ro.activeMutex.RLock()
	execution, exists := ro.activeRecoveries[executionID]
	ro.activeMutex.RUnlock()
	
	if !exists {
		return fmt.Errorf("recovery execution not found: %s", executionID)
	}
	
	if execution.Status != RecoveryStatusRunning && execution.Status != RecoveryStatusPending {
		return fmt.Errorf("cannot cancel recovery in status: %v", execution.Status)
	}
	
	// 취소 신호 발송
	execution.cancel()
	execution.Status = RecoveryStatusCancelled
	
	return nil
}

// GetRecoveryStatus는 복구 상태를 조회합니다
func (ro *RecoveryOrchestrator) GetRecoveryStatus(executionID string) (*RecoveryExecution, error) {
	ro.activeMutex.RLock()
	execution, exists := ro.activeRecoveries[executionID]
	ro.activeMutex.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("recovery execution not found: %s", executionID)
	}
	
	return execution, nil
}

// Shutdown은 복구 오케스트레이터를 종료합니다
func (ro *RecoveryOrchestrator) Shutdown() {
	// 모든 활성 복구 작업 취소
	ro.activeMutex.RLock()
	for _, execution := range ro.activeRecoveries {
		execution.cancel()
	}
	ro.activeMutex.RUnlock()
	
	ro.cancel()
	ro.wg.Wait()
}

// 내부 메서드들

func (ro *RecoveryOrchestrator) selectRecoveryStrategy(ctx context.Context, errorType ErrorType, err error) RecoveryStrategy {
	ro.strategiesMutex.RLock()
	strategies := ro.strategies[errorType]
	ro.strategiesMutex.RUnlock()
	
	// 우선순위 순으로 전략 검토
	for _, strategy := range strategies {
		if strategy.CanRecover(ctx, err) {
			return strategy
		}
	}
	
	// 범용 전략 검토
	ro.strategiesMutex.RLock()
	universalStrategies := ro.strategies[UnknownError]
	ro.strategiesMutex.RUnlock()
	
	for _, strategy := range universalStrategies {
		if strategy.CanRecover(ctx, err) {
			return strategy
		}
	}
	
	return nil
}

func (ro *RecoveryOrchestrator) createRecoveryExecution(target RecoveryTarget, strategy RecoveryStrategy) *RecoveryExecution {
	ctx, cancel := context.WithTimeout(ro.ctx, ro.config.DefaultTimeout)
	
	execution := &RecoveryExecution{
		ID:        fmt.Sprintf("recovery_%d", time.Now().UnixNano()),
		Target:    target,
		Strategy:  strategy,
		Status:    RecoveryStatusPending,
		StartTime: time.Now(),
		Progress:  0.0,
		Steps:     make([]RecoveryStep, 0),
		ctx:       ctx,
		cancel:    cancel,
	}
	
	// 활성 복구 목록에 추가
	ro.activeMutex.Lock()
	ro.activeRecoveries[execution.ID] = execution
	ro.activeMutex.Unlock()
	
	return execution
}

func (ro *RecoveryOrchestrator) executeRecovery(execution *RecoveryExecution) {
	defer func() {
		// 완료 시 활성 목록에서 제거
		ro.activeMutex.Lock()
		delete(ro.activeRecoveries, execution.ID)
		ro.activeMutex.Unlock()
		
		// 히스토리에 추가
		ro.addToHistory(execution)
		
		// 알림 발송
		if ro.alertManager != nil {
			level := AlertLevelInfo
			if execution.Status == RecoveryStatusFailed {
				level = AlertLevelError
			}
			
			ro.alertManager.SendAlert(level,
				fmt.Sprintf("Recovery %s for %s", 
					ro.statusToString(execution.Status),
					execution.Target.Type),
				map[string]interface{}{
					"execution_id": execution.ID,
					"target":       execution.Target,
					"duration":     time.Since(execution.StartTime),
				})
		}
	}()
	
	// 실행 시작
	execution.Status = RecoveryStatusRunning
	ro.addStep(execution, "recovery_started", "Recovery execution started")
	
	// 복구 전략 실행
	err := execution.Strategy.Execute(execution.ctx, execution.Target)
	
	// 결과 처리
	now := time.Now()
	execution.EndTime = &now
	execution.Progress = 100.0
	
	if err != nil {
		execution.Status = RecoveryStatusFailed
		execution.Error = err.Error()
		ro.addStep(execution, "recovery_failed", fmt.Sprintf("Recovery failed: %v", err))
		
		// 재시도 로직
		if ro.shouldRetry(execution) {
			ro.retryRecovery(execution)
		}
	} else {
		execution.Status = RecoveryStatusCompleted
		ro.addStep(execution, "recovery_completed", "Recovery completed successfully")
	}
}

func (ro *RecoveryOrchestrator) addStep(execution *RecoveryExecution, name, description string) {
	step := RecoveryStep{
		Name:        name,
		Status:      StepStatusRunning,
		StartTime:   time.Now(),
		Description: description,
	}
	
	execution.Steps = append(execution.Steps, step)
}

func (ro *RecoveryOrchestrator) completeStep(execution *RecoveryExecution, stepIndex int, err error) {
	if stepIndex >= len(execution.Steps) {
		return
	}
	
	step := &execution.Steps[stepIndex]
	now := time.Now()
	step.EndTime = &now
	step.Duration = now.Sub(step.StartTime)
	
	if err != nil {
		step.Status = StepStatusFailed
		step.Error = err.Error()
	} else {
		step.Status = StepStatusCompleted
	}
}

func (ro *RecoveryOrchestrator) shouldRetry(execution *RecoveryExecution) bool {
	// 간단한 재시도 로직 (실제로는 더 복잡한 조건)
	return execution.Target.Priority >= RecoveryPriorityHigh
}

func (ro *RecoveryOrchestrator) retryRecovery(execution *RecoveryExecution) {
	// 재시도 구현 (실제로는 별도 고루틴에서)
	time.Sleep(ro.config.RetryDelay)
	
	// 새로운 실행 생성
	newExecution := ro.createRecoveryExecution(execution.Target, execution.Strategy)
	go ro.executeRecovery(newExecution)
}

func (ro *RecoveryOrchestrator) addToHistory(execution *RecoveryExecution) {
	record := RecoveryRecord{
		ExecutionID:  execution.ID,
		Target:       execution.Target,
		StrategyName: execution.Strategy.GetName(),
		Status:       execution.Status,
		StartTime:    execution.StartTime,
		Error:        execution.Error,
		SuccessRate:  execution.Strategy.GetSuccessRate(),
	}
	
	if execution.EndTime != nil {
		record.Duration = execution.EndTime.Sub(execution.StartTime)
	}
	
	ro.historyMutex.Lock()
	ro.recoveryHistory = append(ro.recoveryHistory, record)
	
	// 히스토리 크기 제한
	if len(ro.recoveryHistory) > ro.maxHistory {
		ro.recoveryHistory = ro.recoveryHistory[1:]
	}
	ro.historyMutex.Unlock()
}

func (ro *RecoveryOrchestrator) getActiveRecoveryCount() int {
	ro.activeMutex.RLock()
	defer ro.activeMutex.RUnlock()
	return len(ro.activeRecoveries)
}

func (ro *RecoveryOrchestrator) healthMonitor() {
	defer ro.wg.Done()
	
	ticker := time.NewTicker(ro.config.HealthCheckInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ro.ctx.Done():
			return
		case <-ticker.C:
			ro.performHealthCheck()
		}
	}
}

func (ro *RecoveryOrchestrator) performHealthCheck() {
	if ro.healthChecker == nil {
		return
	}
	
	// 전체 시스템 건강 상태 확인
	health := ro.healthChecker.GetOverallHealth()
	
	// 건강하지 않은 상태면 자동 복구 시도
	if health.Status != HealthHealthy {
		for _, issue := range health.Issues {
			target := RecoveryTarget{
				Type:       "system",
				Identifier: issue.Component,
				Context: map[string]interface{}{
					"issue": issue,
				},
				Priority: ro.mapHealthToPriority(health.Status),
			}
			
			// 복구 시도 (에러 생성)
			err := fmt.Errorf("health check failed: %s", issue.Description)
			ro.RecoverFromError(ro.ctx, err, target)
		}
	}
}

func (ro *RecoveryOrchestrator) mapHealthToPriority(status HealthStatus) RecoveryPriority {
	switch status {
	case HealthDegraded:
		return RecoveryPriorityMedium
	case HealthUnhealthy:
		return RecoveryPriorityHigh
	case HealthCritical:
		return RecoveryPriorityCritical
	default:
		return RecoveryPriorityLow
	}
}

func (ro *RecoveryOrchestrator) historyCleanup() {
	defer ro.wg.Done()
	
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	
	for {
		select {
		case <-ro.ctx.Done():
			return
		case <-ticker.C:
			ro.cleanupOldHistory()
		}
	}
}

func (ro *RecoveryOrchestrator) cleanupOldHistory() {
	cutoff := time.Now().Add(-ro.config.HistoryRetentionPeriod)
	
	ro.historyMutex.Lock()
	defer ro.historyMutex.Unlock()
	
	var filtered []RecoveryRecord
	for _, record := range ro.recoveryHistory {
		if record.StartTime.After(cutoff) {
			filtered = append(filtered, record)
		}
	}
	
	ro.recoveryHistory = filtered
}

func (ro *RecoveryOrchestrator) statusToString(status RecoveryStatus) string {
	switch status {
	case RecoveryStatusPending:
		return "pending"
	case RecoveryStatusRunning:
		return "running"
	case RecoveryStatusCompleted:
		return "completed"
	case RecoveryStatusFailed:
		return "failed"
	case RecoveryStatusCancelled:
		return "cancelled"
	case RecoveryStatusTimedOut:
		return "timed out"
	default:
		return "unknown"
	}
}

func (ro *RecoveryOrchestrator) registerDefaultStrategies() {
	// 프로세스 복구 전략
	processStrategy := &ProcessRecoveryStrategy{
		processManager: ro.processManager,
	}
	ro.RegisterStrategy(ProcessError, processStrategy)
	
	// 세션 복구 전략
	sessionStrategy := &SessionRecoveryStrategy{
		sessionManager: ro.sessionManager,
	}
	ro.RegisterStrategy(InternalError, sessionStrategy)
	
	// 리소스 복구 전략
	resourceStrategy := &ResourceRecoveryStrategy{}
	ro.RegisterStrategy(ResourceError, resourceStrategy)
	
	// 네트워크 복구 전략
	networkStrategy := &NetworkRecoveryStrategy{}
	ro.RegisterStrategy(NetworkError, networkStrategy)
}

// 기본 복구 전략 구현들

// ProcessRecoveryStrategy는 프로세스 복구 전략입니다
type ProcessRecoveryStrategy struct {
	processManager *ProcessManager
}

func (s *ProcessRecoveryStrategy) CanRecover(ctx context.Context, err error) bool {
	return s.processManager != nil
}

func (s *ProcessRecoveryStrategy) Execute(ctx context.Context, target RecoveryTarget) error {
	// 프로세스 재시작 로직
	return s.processManager.RestartProcess(target.Identifier)
}

func (s *ProcessRecoveryStrategy) GetEstimatedTime() time.Duration {
	return 30 * time.Second
}

func (s *ProcessRecoveryStrategy) GetSuccessRate() float64 {
	return 0.85
}

func (s *ProcessRecoveryStrategy) GetName() string {
	return "process_recovery"
}

func (s *ProcessRecoveryStrategy) GetPriority() int {
	return 10
}

func (s *ProcessRecoveryStrategy) GetPrerequisites() []string {
	return []string{"process_manager"}
}

// SessionRecoveryStrategy는 세션 복구 전략입니다
type SessionRecoveryStrategy struct {
	sessionManager SessionManager
}

func (s *SessionRecoveryStrategy) CanRecover(ctx context.Context, err error) bool {
	return s.sessionManager != nil
}

func (s *SessionRecoveryStrategy) Execute(ctx context.Context, target RecoveryTarget) error {
	// 세션 복구 로직 (실제 구현 필요)
	return nil
}

func (s *SessionRecoveryStrategy) GetEstimatedTime() time.Duration {
	return 15 * time.Second
}

func (s *SessionRecoveryStrategy) GetSuccessRate() float64 {
	return 0.90
}

func (s *SessionRecoveryStrategy) GetName() string {
	return "session_recovery"
}

func (s *SessionRecoveryStrategy) GetPriority() int {
	return 8
}

func (s *SessionRecoveryStrategy) GetPrerequisites() []string {
	return []string{"session_manager"}
}

// ResourceRecoveryStrategy는 리소스 복구 전략입니다
type ResourceRecoveryStrategy struct{}

func (s *ResourceRecoveryStrategy) CanRecover(ctx context.Context, err error) bool {
	return true
}

func (s *ResourceRecoveryStrategy) Execute(ctx context.Context, target RecoveryTarget) error {
	// 리소스 정리 및 복구 로직
	return nil
}

func (s *ResourceRecoveryStrategy) GetEstimatedTime() time.Duration {
	return 60 * time.Second
}

func (s *ResourceRecoveryStrategy) GetSuccessRate() float64 {
	return 0.75
}

func (s *ResourceRecoveryStrategy) GetName() string {
	return "resource_recovery"
}

func (s *ResourceRecoveryStrategy) GetPriority() int {
	return 6
}

func (s *ResourceRecoveryStrategy) GetPrerequisites() []string {
	return nil
}

// NetworkRecoveryStrategy는 네트워크 복구 전략입니다
type NetworkRecoveryStrategy struct{}

func (s *NetworkRecoveryStrategy) CanRecover(ctx context.Context, err error) bool {
	return true
}

func (s *NetworkRecoveryStrategy) Execute(ctx context.Context, target RecoveryTarget) error {
	// 네트워크 연결 재설정 로직
	return nil
}

func (s *NetworkRecoveryStrategy) GetEstimatedTime() time.Duration {
	return 10 * time.Second
}

func (s *NetworkRecoveryStrategy) GetSuccessRate() float64 {
	return 0.80
}

func (s *NetworkRecoveryStrategy) GetName() string {
	return "network_recovery"
}

func (s *NetworkRecoveryStrategy) GetPriority() int {
	return 7
}

func (s *NetworkRecoveryStrategy) GetPrerequisites() []string {
	return nil
}