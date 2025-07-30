package docker

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
)

// AgentRecoveryManager 에이전트 자동 복구 관리자
type AgentRecoveryManager struct {
	client      *Client
	syncMgr     *AgentStateSynchronizer
	healthMgr   *AgentHealthMonitor
	networkMgr  *AgentNetworkManager
	resourceMgr *AgentResourceManager
	
	mu           sync.RWMutex
	recoveryJobs map[string]*RecoveryJob
	running      bool
	cancel       context.CancelFunc
}

// RecoveryJob 복구 작업 정보
type RecoveryJob struct {
	AgentID       string            `json:"agent_id"`
	ContainerID   string            `json:"container_id"`
	FailureType   FailureType       `json:"failure_type"`
	FailureReason string            `json:"failure_reason"`
	Strategy      RecoveryStrategy  `json:"strategy"`
	Status        RecoveryStatus    `json:"status"`
	Attempts      int               `json:"attempts"`
	MaxAttempts   int               `json:"max_attempts"`
	NextAttempt   time.Time         `json:"next_attempt"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
	History       []RecoveryAttempt `json:"history"`
	
	mu            sync.RWMutex
}

// FailureType 장애 유형
type FailureType string

const (
	FailureTypeContainerStopped   FailureType = "container_stopped"
	FailureTypeContainerDied      FailureType = "container_died"
	FailureTypeHealthCheckFailed  FailureType = "health_check_failed"
	FailureTypeNetworkDisconnected FailureType = "network_disconnected"
	FailureTypeResourceExhausted  FailureType = "resource_exhausted"
	FailureTypeOOMKilled         FailureType = "oom_killed"
)

// RecoveryStrategy 복구 전략
type RecoveryStrategy string

const (
	RecoveryStrategyRestart       RecoveryStrategy = "restart"        // 컨테이너 재시작
	RecoveryStrategyRecreate      RecoveryStrategy = "recreate"       // 컨테이너 재생성
	RecoveryStrategyNetworkRepair RecoveryStrategy = "network_repair" // 네트워크 복구
	RecoveryStrategyResourceScale RecoveryStrategy = "resource_scale" // 리소스 확장
	RecoveryStrategyManual        RecoveryStrategy = "manual"         // 수동 개입 필요
)

// RecoveryStatus 복구 상태
type RecoveryStatus string

const (
	RecoveryStatusPending    RecoveryStatus = "pending"    // 대기 중
	RecoveryStatusRunning    RecoveryStatus = "running"    // 실행 중
	RecoveryStatusSucceeded  RecoveryStatus = "succeeded"  // 성공
	RecoveryStatusFailed     RecoveryStatus = "failed"     // 실패
	RecoveryStatusAbandoned  RecoveryStatus = "abandoned"  // 포기됨
)

// RecoveryAttempt 복구 시도 기록
type RecoveryAttempt struct {
	AttemptNumber int               `json:"attempt_number"`
	Strategy      RecoveryStrategy  `json:"strategy"`
	StartTime     time.Time         `json:"start_time"`
	EndTime       time.Time         `json:"end_time"`
	Duration      time.Duration     `json:"duration"`
	Success       bool              `json:"success"`
	Error         string            `json:"error,omitempty"`
	Actions       []RecoveryAction  `json:"actions"`
}

// RecoveryAction 복구 작업 세부 동작
type RecoveryAction struct {
	Type        string    `json:"type"`
	Description string    `json:"description"`
	Timestamp   time.Time `json:"timestamp"`
	Success     bool      `json:"success"`
	Error       string    `json:"error,omitempty"`
}

// RecoveryConfig 복구 설정
type RecoveryConfig struct {
	Enabled              bool          `json:"enabled"`
	MaxAttempts          int           `json:"max_attempts"`           // 최대 복구 시도 횟수
	RetryInterval        time.Duration `json:"retry_interval"`         // 재시도 간격
	BackoffMultiplier    float64       `json:"backoff_multiplier"`     // 백오프 배수
	MaxRetryInterval     time.Duration `json:"max_retry_interval"`     // 최대 재시도 간격
	EnableAutoRestart    bool          `json:"enable_auto_restart"`    // 자동 재시작
	EnableAutoRecreate   bool          `json:"enable_auto_recreate"`   // 자동 재생성
	EnableNetworkRepair  bool          `json:"enable_network_repair"`  // 네트워크 복구
	EnableResourceScale  bool          `json:"enable_resource_scale"`  // 리소스 확장
	MonitoringInterval   time.Duration `json:"monitoring_interval"`    // 모니터링 간격
}

// RecoveryAlert 복구 알림
type RecoveryAlert struct {
	AgentID     string           `json:"agent_id"`
	JobID       string           `json:"job_id"`
	Type        RecoveryAlertType `json:"type"`
	Message     string           `json:"message"`
	Timestamp   time.Time        `json:"timestamp"`
	Severity    AlertSeverity    `json:"severity"`
}

// RecoveryAlertType 복구 알림 타입
type RecoveryAlertType string

const (
	RecoveryAlertTypeStarted   RecoveryAlertType = "started"
	RecoveryAlertTypeSucceeded RecoveryAlertType = "succeeded"
	RecoveryAlertTypeFailed    RecoveryAlertType = "failed"
	RecoveryAlertTypeAbandoned RecoveryAlertType = "abandoned"
)

// NewAgentRecoveryManager 새로운 에이전트 복구 관리자 생성
func NewAgentRecoveryManager(
	client *Client,
	syncMgr *AgentStateSynchronizer,
	healthMgr *AgentHealthMonitor,
	networkMgr *AgentNetworkManager,
	resourceMgr *AgentResourceManager,
) *AgentRecoveryManager {
	return &AgentRecoveryManager{
		client:       client,
		syncMgr:      syncMgr,
		healthMgr:    healthMgr,
		networkMgr:   networkMgr,
		resourceMgr:  resourceMgr,
		recoveryJobs: make(map[string]*RecoveryJob),
	}
}

// Start 복구 관리자 시작
func (arm *AgentRecoveryManager) Start(ctx context.Context, config RecoveryConfig) error {
	arm.mu.Lock()
	defer arm.mu.Unlock()

	if arm.running {
		return fmt.Errorf("recovery manager already running")
	}

	// 기본값 설정
	if config.MaxAttempts == 0 {
		config.MaxAttempts = 3
	}
	if config.RetryInterval == 0 {
		config.RetryInterval = 30 * time.Second
	}
	if config.BackoffMultiplier == 0 {
		config.BackoffMultiplier = 2.0
	}
	if config.MaxRetryInterval == 0 {
		config.MaxRetryInterval = 10 * time.Minute
	}
	if config.MonitoringInterval == 0 {
		config.MonitoringInterval = 10 * time.Second
	}

	recoveryCtx, cancel := context.WithCancel(ctx)
	arm.cancel = cancel
	arm.running = true

	// 상태 동기화 핸들러 등록
	if arm.syncMgr != nil {
		arm.setupStateSyncHandler(config)
	}

	// 복구 작업 처리 고루틴 시작
	go arm.runRecoveryWorker(recoveryCtx, config)

	return nil
}

// Stop 복구 관리자 중지
func (arm *AgentRecoveryManager) Stop() error {
	arm.mu.Lock()
	defer arm.mu.Unlock()

	if !arm.running {
		return nil
	}

	if arm.cancel != nil {
		arm.cancel()
	}

	arm.running = false
	return nil
}

// TriggerRecovery 수동으로 복구 작업 트리거
func (arm *AgentRecoveryManager) TriggerRecovery(agentID string, failureType FailureType, reason string) error {
	arm.mu.Lock()
	defer arm.mu.Unlock()

	// 이미 진행 중인 복구 작업 확인
	if job, exists := arm.recoveryJobs[agentID]; exists {
		if job.Status == RecoveryStatusRunning || job.Status == RecoveryStatusPending {
			return fmt.Errorf("recovery job already in progress for agent: %s", agentID)
		}
	}

	// 에이전트 상태 조회
	agentState, exists := arm.syncMgr.GetAgentState(agentID)
	if !exists {
		return fmt.Errorf("agent state not found: %s", agentID)
	}

	// 복구 전략 결정
	strategy := arm.determineRecoveryStrategy(failureType, agentState)

	// 복구 작업 생성
	job := &RecoveryJob{
		AgentID:       agentID,
		ContainerID:   agentState.ContainerID,
		FailureType:   failureType,
		FailureReason: reason,
		Strategy:      strategy,
		Status:        RecoveryStatusPending,
		Attempts:      0,
		MaxAttempts:   3, // 기본값
		NextAttempt:   time.Now(),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		History:       make([]RecoveryAttempt, 0),
	}

	arm.recoveryJobs[agentID] = job

	// 복구 시작 알림
	alert := RecoveryAlert{
		AgentID:   agentID,
		JobID:     fmt.Sprintf("%s-%d", agentID, time.Now().Unix()),
		Type:      RecoveryAlertTypeStarted,
		Message:   fmt.Sprintf("Recovery started for failure: %s", failureType),
		Timestamp: time.Now(),
		Severity:  AlertSeverityWarning,
	}
	arm.sendRecoveryAlert(alert)

	return nil
}

// setupStateSyncHandler 상태 동기화 핸들러 설정
func (arm *AgentRecoveryManager) setupStateSyncHandler(config RecoveryConfig) {
	handler := func(event StateSyncEvent) {
		if !config.Enabled {
			return
		}

		// 상태 변경 기반 복구 트리거
		if event.Type == SyncEventStateChanged {
			arm.handleStateChange(event, config)
		}
	}

	arm.syncMgr.RegisterSyncHandler(handler)
}

// handleStateChange 상태 변경 처리
func (arm *AgentRecoveryManager) handleStateChange(event StateSyncEvent, config RecoveryConfig) {
	newState := event.NewState
	
	// 컨테이너 중지 감지
	if !newState.ContainerState.Running && newState.ContainerState.Status != "created" {
		failureType := FailureTypeContainerStopped
		reason := fmt.Sprintf("Container status: %s", newState.ContainerState.Status)
		
		if newState.ContainerState.OOMKilled {
			failureType = FailureTypeOOMKilled
			reason = "Container killed by OOM killer"
		} else if newState.ContainerState.ExitCode != 0 {
			failureType = FailureTypeContainerDied
			reason = fmt.Sprintf("Container died with exit code: %d", newState.ContainerState.ExitCode)
		}

		if config.EnableAutoRestart {
			arm.TriggerRecovery(event.AgentID, failureType, reason)
		}
	}

	// 헬스체크 실패 감지
	if newState.HealthState != nil && 
		(newState.HealthState.Status == AgentHealthStatusUnhealthy || 
		 newState.HealthState.Status == AgentHealthStatusCritical) {
		
		reason := fmt.Sprintf("Health check failed: %s", newState.HealthState.Status)
		arm.TriggerRecovery(event.AgentID, FailureTypeHealthCheckFailed, reason)
	}

	// 네트워크 연결 문제 감지
	if newState.NetworkState == nil && config.EnableNetworkRepair {
		reason := "Network disconnected"
		arm.TriggerRecovery(event.AgentID, FailureTypeNetworkDisconnected, reason)
	}
}

// runRecoveryWorker 복구 작업 처리 워커
func (arm *AgentRecoveryManager) runRecoveryWorker(ctx context.Context, config RecoveryConfig) {
	ticker := time.NewTicker(config.MonitoringInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			arm.processRecoveryJobs(ctx, config)
		}
	}
}

// processRecoveryJobs 복구 작업 처리
func (arm *AgentRecoveryManager) processRecoveryJobs(ctx context.Context, config RecoveryConfig) {
	arm.mu.RLock()
	jobs := make([]*RecoveryJob, 0)
	for _, job := range arm.recoveryJobs {
		jobs = append(jobs, job)
	}
	arm.mu.RUnlock()

	for _, job := range jobs {
		if job.Status == RecoveryStatusPending && time.Now().After(job.NextAttempt) {
			go arm.executeRecoveryJob(ctx, job, config)
		}
	}
}

// executeRecoveryJob 복구 작업 실행
func (arm *AgentRecoveryManager) executeRecoveryJob(ctx context.Context, job *RecoveryJob, config RecoveryConfig) {
	job.mu.Lock()
	defer job.mu.Unlock()

	if job.Status != RecoveryStatusPending {
		return
	}

	job.Status = RecoveryStatusRunning
	job.Attempts++
	job.UpdatedAt = time.Now()

	// 복구 시도 기록 시작
	attempt := RecoveryAttempt{
		AttemptNumber: job.Attempts,
		Strategy:      job.Strategy,
		StartTime:     time.Now(),
		Actions:       make([]RecoveryAction, 0),
	}

	success := false
	var recoveryError error

	// 복구 전략별 실행
	switch job.Strategy {
	case RecoveryStrategyRestart:
		success, recoveryError = arm.executeRestartStrategy(ctx, job, &attempt)
	case RecoveryStrategyRecreate:
		success, recoveryError = arm.executeRecreateStrategy(ctx, job, &attempt)
	case RecoveryStrategyNetworkRepair:
		success, recoveryError = arm.executeNetworkRepairStrategy(ctx, job, &attempt)
	case RecoveryStrategyResourceScale:
		success, recoveryError = arm.executeResourceScaleStrategy(ctx, job, &attempt)
	case RecoveryStrategyManual:
		// 수동 개입 필요 - 알림만 발송
		success = false
		recoveryError = fmt.Errorf("manual intervention required")
	}

	// 복구 시도 완료
	attempt.EndTime = time.Now()
	attempt.Duration = attempt.EndTime.Sub(attempt.StartTime)
	attempt.Success = success
	if recoveryError != nil {
		attempt.Error = recoveryError.Error()
	}

	job.History = append(job.History, attempt)

	// 결과에 따른 상태 업데이트
	if success {
		job.Status = RecoveryStatusSucceeded
		
		// 성공 알림 발송
		alert := RecoveryAlert{
			AgentID:   job.AgentID,
			Type:      RecoveryAlertTypeSucceeded,
			Message:   fmt.Sprintf("Recovery succeeded after %d attempts", job.Attempts),
			Timestamp: time.Now(),
			Severity:  AlertSeverityInfo,
		}
		arm.sendRecoveryAlert(alert)

		// 상태 동기화 트리거
		if arm.syncMgr != nil {
			go arm.syncMgr.SyncAgentState(context.Background(), job.AgentID)
		}
	} else {
		if job.Attempts >= job.MaxAttempts {
			job.Status = RecoveryStatusAbandoned
			
			// 포기 알림 발송
			alert := RecoveryAlert{
				AgentID:   job.AgentID,
				Type:      RecoveryAlertTypeAbandoned,
				Message:   fmt.Sprintf("Recovery abandoned after %d attempts: %v", job.Attempts, recoveryError),
				Timestamp: time.Now(),
				Severity:  AlertSeverityCritical,
			}
			arm.sendRecoveryAlert(alert)
		} else {
			job.Status = RecoveryStatusPending
			
			// 다음 시도 시간 계산 (백오프)
			backoffInterval := time.Duration(float64(config.RetryInterval) * 
				math.Pow(config.BackoffMultiplier, float64(job.Attempts-1)))
			if backoffInterval > config.MaxRetryInterval {
				backoffInterval = config.MaxRetryInterval
			}
			job.NextAttempt = time.Now().Add(backoffInterval)
			
			// 실패 알림 발송
			alert := RecoveryAlert{
				AgentID:   job.AgentID,
				Type:      RecoveryAlertTypeFailed,
				Message:   fmt.Sprintf("Recovery attempt %d failed: %v", job.Attempts, recoveryError),
				Timestamp: time.Now(),
				Severity:  AlertSeverityError,
			}
			arm.sendRecoveryAlert(alert)
		}
	}

	job.UpdatedAt = time.Now()
}

// executeRestartStrategy 재시작 전략 실행
func (arm *AgentRecoveryManager) executeRestartStrategy(ctx context.Context, job *RecoveryJob, attempt *RecoveryAttempt) (bool, error) {
	// 컨테이너 재시작
	attempt.Actions = append(attempt.Actions, RecoveryAction{
		Type:        "container_restart",
		Description: "Restarting container",
		Timestamp:   time.Now(),
	})

	// TODO: Docker API 호환성 문제로 임시 구현
	if err := arm.client.cli.ContainerRestart(ctx, job.ContainerID, container.StopOptions{}); err != nil {
		attempt.Actions[len(attempt.Actions)-1].Success = false
		attempt.Actions[len(attempt.Actions)-1].Error = err.Error()
		return false, fmt.Errorf("container restart failed: %w", err)
	}

	attempt.Actions[len(attempt.Actions)-1].Success = true

	// 시작 대기
	time.Sleep(5 * time.Second)

	// 컨테이너 상태 확인
	containerInfo, err := arm.client.cli.ContainerInspect(ctx, job.ContainerID)
	if err != nil {
		return false, fmt.Errorf("container inspect failed: %w", err)
	}

	if !containerInfo.State.Running {
		return false, fmt.Errorf("container not running after restart")
	}

	return true, nil
}

// executeRecreateStrategy 재생성 전략 실행
func (arm *AgentRecoveryManager) executeRecreateStrategy(ctx context.Context, job *RecoveryJob, attempt *RecoveryAttempt) (bool, error) {
	// TODO: 컨테이너 정보 백업 기능 구현 필요
	_ = ctx // 임시로 사용하지 않음 표시

	// 컨테이너 중지 및 제거
	attempt.Actions = append(attempt.Actions, RecoveryAction{
		Type:        "container_remove",
		Description: "Removing old container",
		Timestamp:   time.Now(),
	})

	// TODO: Docker API 호환성 문제로 임시 구현
	if err := arm.client.cli.ContainerStop(ctx, job.ContainerID, container.StopOptions{}); err != nil {
		// 이미 중지된 경우는 무시
	}

	if err := arm.client.cli.ContainerRemove(ctx, job.ContainerID, container.RemoveOptions{
		Force: true,
	}); err != nil {
		attempt.Actions[len(attempt.Actions)-1].Success = false
		attempt.Actions[len(attempt.Actions)-1].Error = err.Error()
		return false, fmt.Errorf("container remove failed: %w", err)
	}

	attempt.Actions[len(attempt.Actions)-1].Success = true

	// 새 컨테이너 생성
	attempt.Actions = append(attempt.Actions, RecoveryAction{
		Type:        "container_create",
		Description: "Creating new container",
		Timestamp:   time.Now(),
	})

	// TODO: 새 컨테이너 생성 로직 구현
	// 이 부분은 에이전트 생성 로직과 연동해야 함

	return true, nil
}

// executeNetworkRepairStrategy 네트워크 복구 전략 실행
func (arm *AgentRecoveryManager) executeNetworkRepairStrategy(ctx context.Context, job *RecoveryJob, attempt *RecoveryAttempt) (bool, error) {
	if arm.networkMgr == nil {
		return false, fmt.Errorf("network manager not available")
	}

	// 에이전트 네트워크 재생성
	attempt.Actions = append(attempt.Actions, RecoveryAction{
		Type:        "network_recreate",
		Description: "Recreating agent network",
		Timestamp:   time.Now(),
	})

	// 기존 네트워크 정리
	if err := arm.networkMgr.DeleteAgentNetwork(ctx, job.AgentID); err != nil {
		// 네트워크가 없는 경우는 무시
	}

	// 새 네트워크 생성
	_, err := arm.networkMgr.CreateAgentNetwork(ctx, job.AgentID)
	if err != nil {
		attempt.Actions[len(attempt.Actions)-1].Success = false
		attempt.Actions[len(attempt.Actions)-1].Error = err.Error()
		return false, fmt.Errorf("network recreation failed: %w", err)
	}

	attempt.Actions[len(attempt.Actions)-1].Success = true

	// 컨테이너를 네트워크에 연결
	attempt.Actions = append(attempt.Actions, RecoveryAction{
		Type:        "network_connect",
		Description: "Connecting container to network",
		Timestamp:   time.Now(),
	})

	if err := arm.networkMgr.ConnectAgentContainer(ctx, job.AgentID, job.ContainerID); err != nil {
		attempt.Actions[len(attempt.Actions)-1].Success = false
		attempt.Actions[len(attempt.Actions)-1].Error = err.Error()
		return false, fmt.Errorf("network connection failed: %w", err)
	}

	attempt.Actions[len(attempt.Actions)-1].Success = true
	return true, nil
}

// executeResourceScaleStrategy 리소스 확장 전략 실행
func (arm *AgentRecoveryManager) executeResourceScaleStrategy(ctx context.Context, job *RecoveryJob, attempt *RecoveryAttempt) (bool, error) {
	// TODO: 리소스 확장 로직 구현
	// 컨테이너 재생성 시 더 많은 리소스 할당
	return false, fmt.Errorf("resource scaling not implemented yet")
}

// determineRecoveryStrategy 복구 전략 결정
func (arm *AgentRecoveryManager) determineRecoveryStrategy(failureType FailureType, agentState *AgentState) RecoveryStrategy {
	switch failureType {
	case FailureTypeContainerStopped, FailureTypeContainerDied:
		if agentState.ContainerState.ExitCode == 0 {
			return RecoveryStrategyRestart
		} else {
			return RecoveryStrategyRecreate
		}
	case FailureTypeOOMKilled:
		return RecoveryStrategyResourceScale
	case FailureTypeHealthCheckFailed:
		return RecoveryStrategyRestart
	case FailureTypeNetworkDisconnected:
		return RecoveryStrategyNetworkRepair
	case FailureTypeResourceExhausted:
		return RecoveryStrategyResourceScale
	default:
		return RecoveryStrategyManual
	}
}

// sendRecoveryAlert 복구 알림 발송
func (arm *AgentRecoveryManager) sendRecoveryAlert(alert RecoveryAlert) {
	// TODO: 실제 알림 시스템과 연동
	fmt.Printf("[RECOVERY ALERT] %s: %s - %s\n", 
		alert.Severity, alert.AgentID, alert.Message)
}

// GetRecoveryJob 복구 작업 정보 조회
func (arm *AgentRecoveryManager) GetRecoveryJob(agentID string) (*RecoveryJob, bool) {
	arm.mu.RLock()
	defer arm.mu.RUnlock()

	job, exists := arm.recoveryJobs[agentID]
	if !exists {
		return nil, false
	}

	job.mu.RLock()
	defer job.mu.RUnlock()

	// 복사본 반환
	jobCopy := *job
	jobCopy.History = make([]RecoveryAttempt, len(job.History))
	copy(jobCopy.History, job.History)

	return &jobCopy, true
}

// GetAllRecoveryJobs 모든 복구 작업 조회
func (arm *AgentRecoveryManager) GetAllRecoveryJobs() map[string]*RecoveryJob {
	arm.mu.RLock()
	defer arm.mu.RUnlock()

	result := make(map[string]*RecoveryJob)
	for agentID, job := range arm.recoveryJobs {
		job.mu.RLock()
		jobCopy := *job
		jobCopy.History = make([]RecoveryAttempt, len(job.History))
		copy(jobCopy.History, job.History)
		job.mu.RUnlock()
		
		result[agentID] = &jobCopy
	}

	return result
}

// CancelRecoveryJob 복구 작업 취소
func (arm *AgentRecoveryManager) CancelRecoveryJob(agentID string) error {
	arm.mu.Lock()
	defer arm.mu.Unlock()

	job, exists := arm.recoveryJobs[agentID]
	if !exists {
		return fmt.Errorf("recovery job not found: %s", agentID)
	}

	job.mu.Lock()
	defer job.mu.Unlock()

	if job.Status == RecoveryStatusRunning {
		return fmt.Errorf("cannot cancel running recovery job")
	}

	job.Status = RecoveryStatusAbandoned
	job.UpdatedAt = time.Now()

	return nil
}

// CleanupCompletedJobs 완료된 복구 작업 정리
func (arm *AgentRecoveryManager) CleanupCompletedJobs(olderThan time.Duration) {
	arm.mu.Lock()
	defer arm.mu.Unlock()

	cutoff := time.Now().Add(-olderThan)
	
	for agentID, job := range arm.recoveryJobs {
		job.mu.RLock()
		shouldRemove := (job.Status == RecoveryStatusSucceeded || 
			job.Status == RecoveryStatusAbandoned) &&
			job.UpdatedAt.Before(cutoff)
		job.mu.RUnlock()

		if shouldRemove {
			delete(arm.recoveryJobs, agentID)
		}
	}
}

// IsRunning 복구 관리자 실행 상태 확인
func (arm *AgentRecoveryManager) IsRunning() bool {
	arm.mu.RLock()
	defer arm.mu.RUnlock()
	return arm.running
}