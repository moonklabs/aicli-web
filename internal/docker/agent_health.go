package docker

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
)

// AgentHealthMonitor 에이전트 컨테이너 헬스체크 모니터
type AgentHealthMonitor struct {
	client      *Client
	mu          sync.RWMutex
	healthChecks map[string]*AgentHealthCheck
	running     bool
	cancel      context.CancelFunc
}

// AgentHealthCheck 에이전트 헬스체크 정보
type AgentHealthCheck struct {
	AgentID      string                `json:"agent_id"`
	ContainerID  string                `json:"container_id"`
	Status       AgentHealthStatus     `json:"status"`
	Config       HealthCheckConfig     `json:"config"`
	History      []HealthCheckResult   `json:"history"`
	LastCheck    time.Time             `json:"last_check"`
	NextCheck    time.Time             `json:"next_check"`
	FailureCount int                   `json:"failure_count"`
	mu           sync.RWMutex
}

// AgentHealthStatus 에이전트 헬스체크 상태
type AgentHealthStatus string

const (
	AgentHealthStatusUnknown   AgentHealthStatus = "unknown"   // 상태 불명
	AgentHealthStatusStarting  AgentHealthStatus = "starting"  // 시작 중
	AgentHealthStatusHealthy   AgentHealthStatus = "healthy"   // 정상
	AgentHealthStatusUnhealthy AgentHealthStatus = "unhealthy" // 비정상
	AgentHealthStatusCritical  AgentHealthStatus = "critical"  // 심각한 문제
)

// HealthCheckConfig 헬스체크 설정
type HealthCheckConfig struct {
	Enabled           bool          `json:"enabled"`
	Command           []string      `json:"command"`           // 헬스체크 명령어
	Interval          time.Duration `json:"interval"`          // 체크 간격 (기본: 30초)
	Timeout           time.Duration `json:"timeout"`           // 타임아웃 (기본: 30초)
	StartPeriod       time.Duration `json:"start_period"`      // 시작 대기 시간 (기본: 0초)
	Retries           int           `json:"retries"`           // 실패 허용 횟수 (기본: 3)
	DisableInherit    bool          `json:"disable_inherit"`   // 상속 비활성화
	AutoRestart       bool          `json:"auto_restart"`      // 자동 재시작
	RestartThreshold  int           `json:"restart_threshold"` // 재시작 임계값
}

// HealthCheckResult 헬스체크 결과
type HealthCheckResult struct {
	Timestamp    time.Time         `json:"timestamp"`
	Status       AgentHealthStatus `json:"status"`
	ExitCode     int               `json:"exit_code"`
	Output       string            `json:"output"`
	Error        string            `json:"error,omitempty"`
	Duration     time.Duration     `json:"duration"`
}

// HealthAlert 헬스 알림
type HealthAlert struct {
	AgentID     string            `json:"agent_id"`
	ContainerID string            `json:"container_id"`
	Status      AgentHealthStatus `json:"status"`
	Message     string            `json:"message"`
	Timestamp   time.Time         `json:"timestamp"`
	Severity    AlertSeverity     `json:"severity"`
}

// AlertSeverity 알림 심각도
type AlertSeverity string

const (
	AlertSeverityInfo     AlertSeverity = "info"
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityError    AlertSeverity = "error"
	AlertSeverityCritical AlertSeverity = "critical"
)

// HealthAlertHandler 헬스 알림 핸들러
type HealthAlertHandler func(alert HealthAlert)

// NewAgentHealthMonitor 새로운 에이전트 헬스 모니터 생성
func NewAgentHealthMonitor(client *Client) *AgentHealthMonitor {
	return &AgentHealthMonitor{
		client:       client,
		healthChecks: make(map[string]*AgentHealthCheck),
	}
}

// Start 헬스 모니터링 시작
func (ahm *AgentHealthMonitor) Start(ctx context.Context) error {
	ahm.mu.Lock()
	defer ahm.mu.Unlock()

	if ahm.running {
		return fmt.Errorf("health monitor already running")
	}

	monitorCtx, cancel := context.WithCancel(ctx)
	ahm.cancel = cancel
	ahm.running = true

	// 헬스체크 실행 고루틴 시작
	go ahm.runHealthChecks(monitorCtx)

	return nil
}

// Stop 헬스 모니터링 중지
func (ahm *AgentHealthMonitor) Stop() error {
	ahm.mu.Lock()
	defer ahm.mu.Unlock()

	if !ahm.running {
		return nil
	}

	if ahm.cancel != nil {
		ahm.cancel()
	}

	ahm.running = false
	return nil
}

// RegisterHealthCheck 에이전트 헬스체크 등록
func (ahm *AgentHealthMonitor) RegisterHealthCheck(agentID, containerID string, config HealthCheckConfig) error {
	ahm.mu.Lock()
	defer ahm.mu.Unlock()

	// 기본값 설정
	if config.Interval == 0 {
		config.Interval = 30 * time.Second
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.Retries == 0 {
		config.Retries = 3
	}
	if config.RestartThreshold == 0 {
		config.RestartThreshold = 3
	}

	// 기본 헬스체크 명령어 설정
	if len(config.Command) == 0 {
		config.Command = []string{"/bin/sh", "-c", "claude --version"}
	}

	healthCheck := &AgentHealthCheck{
		AgentID:      agentID,
		ContainerID:  containerID,
		Status:       AgentHealthStatusStarting,
		Config:       config,
		History:      make([]HealthCheckResult, 0),
		NextCheck:    time.Now().Add(config.StartPeriod),
		FailureCount: 0,
	}

	ahm.healthChecks[agentID] = healthCheck

	return nil
}

// UnregisterHealthCheck 에이전트 헬스체크 해제
func (ahm *AgentHealthMonitor) UnregisterHealthCheck(agentID string) {
	ahm.mu.Lock()
	defer ahm.mu.Unlock()

	delete(ahm.healthChecks, agentID)
}

// GetHealthCheck 헬스체크 정보 조회
func (ahm *AgentHealthMonitor) GetHealthCheck(agentID string) (*AgentHealthCheck, bool) {
	ahm.mu.RLock()
	defer ahm.mu.RUnlock()

	healthCheck, exists := ahm.healthChecks[agentID]
	if !exists {
		return nil, false
	}

	// 복사본 반환 (동시성 안전)
	healthCheck.mu.RLock()
	defer healthCheck.mu.RUnlock()

	healthCopy := *healthCheck
	healthCopy.History = make([]HealthCheckResult, len(healthCheck.History))
	copy(healthCopy.History, healthCheck.History)

	return &healthCopy, true
}

// runHealthChecks 헬스체크 실행 루프
func (ahm *AgentHealthMonitor) runHealthChecks(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second) // 10초마다 체크
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ahm.performHealthChecks(ctx)
		}
	}
}

// performHealthChecks 모든 등록된 헬스체크 수행
func (ahm *AgentHealthMonitor) performHealthChecks(ctx context.Context) {
	ahm.mu.RLock()
	healthChecks := make([]*AgentHealthCheck, 0, len(ahm.healthChecks))
	for _, hc := range ahm.healthChecks {
		healthChecks = append(healthChecks, hc)
	}
	ahm.mu.RUnlock()

	for _, healthCheck := range healthChecks {
		if !healthCheck.Config.Enabled {
			continue
		}

		// 헬스체크 시간 확인
		if time.Now().Before(healthCheck.NextCheck) {
			continue
		}

		// 별도 고루틴에서 헬스체크 실행
		go ahm.performSingleHealthCheck(ctx, healthCheck)
	}
}

// performSingleHealthCheck 단일 헬스체크 수행
func (ahm *AgentHealthMonitor) performSingleHealthCheck(ctx context.Context, healthCheck *AgentHealthCheck) {
	healthCheck.mu.Lock()
	defer healthCheck.mu.Unlock()

	startTime := time.Now()
	
	// 컨테이너 상태 확인
	containerInfo, err := ahm.client.cli.ContainerInspect(ctx, healthCheck.ContainerID)
	if err != nil {
		ahm.recordHealthCheckResult(healthCheck, HealthCheckResult{
			Timestamp: startTime,
			Status:    AgentHealthStatusCritical,
			ExitCode:  -1,
			Error:     fmt.Sprintf("Container inspect failed: %v", err),
			Duration:  time.Since(startTime),
		})
		return
	}

	// 컨테이너가 실행 중이 아니면 비정상
	if !containerInfo.State.Running {
		ahm.recordHealthCheckResult(healthCheck, HealthCheckResult{
			Timestamp: startTime,
			Status:    AgentHealthStatusUnhealthy,
			ExitCode:  containerInfo.State.ExitCode,
			Output:    fmt.Sprintf("Container not running. State: %s", containerInfo.State.Status),
			Duration:  time.Since(startTime),
		})
		return
	}

	// 헬스체크 명령어 실행
	checkCtx, cancel := context.WithTimeout(ctx, healthCheck.Config.Timeout)
	defer cancel()

	execConfig := types.ExecConfig{
		Cmd:          healthCheck.Config.Command,
		AttachStdout: true,
		AttachStderr: true,
	}

	execResp, err := ahm.client.cli.ContainerExecCreate(checkCtx, healthCheck.ContainerID, execConfig)
	if err != nil {
		ahm.recordHealthCheckResult(healthCheck, HealthCheckResult{
			Timestamp: startTime,
			Status:    AgentHealthStatusUnhealthy,
			ExitCode:  -1,
			Error:     fmt.Sprintf("Exec create failed: %v", err),
			Duration:  time.Since(startTime),
		})
		return
	}

	attachResp, err := ahm.client.cli.ContainerExecAttach(checkCtx, execResp.ID, types.ExecStartCheck{})
	if err != nil {
		ahm.recordHealthCheckResult(healthCheck, HealthCheckResult{
			Timestamp: startTime,
			Status:    AgentHealthStatusUnhealthy,
			ExitCode:  -1,
			Error:     fmt.Sprintf("Exec attach failed: %v", err),
			Duration:  time.Since(startTime),
		})
		return
	}
	defer attachResp.Close()

	// 출력 읽기
	output, err := io.ReadAll(attachResp.Reader)
	if err != nil {
		ahm.recordHealthCheckResult(healthCheck, HealthCheckResult{
			Timestamp: startTime,
			Status:    AgentHealthStatusUnhealthy,
			ExitCode:  -1,
			Error:     fmt.Sprintf("Read output failed: %v", err),
			Duration:  time.Since(startTime),
		})
		return
	}

	// 실행 결과 확인
	execInspect, err := ahm.client.cli.ContainerExecInspect(checkCtx, execResp.ID)
	if err != nil {
		ahm.recordHealthCheckResult(healthCheck, HealthCheckResult{
			Timestamp: startTime,
			Status:    AgentHealthStatusUnhealthy,
			ExitCode:  -1,
			Error:     fmt.Sprintf("Exec inspect failed: %v", err),
			Duration:  time.Since(startTime),
		})
		return
	}

	// 결과 평가
	status := AgentHealthStatusHealthy
	if execInspect.ExitCode != 0 {
		status = AgentHealthStatusUnhealthy
	}

	// TODO: 컨테이너 통계 수집 기능은 추후 구현

	ahm.recordHealthCheckResult(healthCheck, HealthCheckResult{
		Timestamp: startTime,
		Status:    status,
		ExitCode:  execInspect.ExitCode,
		Output:    string(output),
		Duration:  time.Since(startTime),
	})
}

// recordHealthCheckResult 헬스체크 결과 기록
func (ahm *AgentHealthMonitor) recordHealthCheckResult(healthCheck *AgentHealthCheck, result HealthCheckResult) {
	healthCheck.LastCheck = result.Timestamp
	healthCheck.NextCheck = result.Timestamp.Add(healthCheck.Config.Interval)

	// 히스토리에 결과 추가 (최대 100개 유지)
	healthCheck.History = append(healthCheck.History, result)
	if len(healthCheck.History) > 100 {
		healthCheck.History = healthCheck.History[1:]
	}

	// 상태 업데이트
	previousStatus := healthCheck.Status
	healthCheck.Status = result.Status

	// 실패 카운트 업데이트
	if result.Status == AgentHealthStatusHealthy {
		healthCheck.FailureCount = 0
	} else {
		healthCheck.FailureCount++
	}

	// 자동 재시작 확인
	if healthCheck.Config.AutoRestart &&
		healthCheck.FailureCount >= healthCheck.Config.RestartThreshold &&
		result.Status != AgentHealthStatusCritical {
		go ahm.restartContainer(context.Background(), healthCheck)
	}

	// 상태 변경 시 알림 발송
	if previousStatus != result.Status {
		ahm.sendHealthAlert(healthCheck, result)
	}
}

// restartContainer 컨테이너 재시작
func (ahm *AgentHealthMonitor) restartContainer(ctx context.Context, healthCheck *AgentHealthCheck) {
	// TODO: Docker API 호환성 문제로 임시 구현
	if err := ahm.client.cli.ContainerRestart(ctx, healthCheck.ContainerID, container.StopOptions{}); err != nil {
		// 재시작 실패 알림
		alert := HealthAlert{
			AgentID:     healthCheck.AgentID,
			ContainerID: healthCheck.ContainerID,
			Status:      AgentHealthStatusCritical,
			Message:     fmt.Sprintf("Auto restart failed: %v", err),
			Timestamp:   time.Now(),
			Severity:    AlertSeverityCritical,
		}
		ahm.sendAlert(alert)
	} else {
		// 재시작 성공 알림
		alert := HealthAlert{
			AgentID:     healthCheck.AgentID,
			ContainerID: healthCheck.ContainerID,
			Status:      AgentHealthStatusUnknown,
			Message:     "Container restarted automatically",
			Timestamp:   time.Now(),
			Severity:    AlertSeverityWarning,
		}
		ahm.sendAlert(alert)

		// 실패 카운트 리셋
		healthCheck.mu.Lock()
		healthCheck.FailureCount = 0
		healthCheck.Status = AgentHealthStatusStarting
		healthCheck.mu.Unlock()
	}
}

// sendHealthAlert 헬스 상태 변경 알림 발송
func (ahm *AgentHealthMonitor) sendHealthAlert(healthCheck *AgentHealthCheck, result HealthCheckResult) {
	var severity AlertSeverity
	var message string

	switch result.Status {
	case AgentHealthStatusHealthy:
		severity = AlertSeverityInfo
		message = "Agent container is healthy"
	case AgentHealthStatusUnhealthy:
		severity = AlertSeverityWarning
		message = fmt.Sprintf("Agent container is unhealthy (failures: %d/%d)", 
			healthCheck.FailureCount, healthCheck.Config.Retries)
	case AgentHealthStatusCritical:
		severity = AlertSeverityCritical
		message = "Agent container is in critical state"
	case AgentHealthStatusStarting:
		severity = AlertSeverityInfo
		message = "Agent container is starting"
	default:
		severity = AlertSeverityWarning
		message = "Agent container status unknown"
	}

	alert := HealthAlert{
		AgentID:     healthCheck.AgentID,
		ContainerID: healthCheck.ContainerID,
		Status:      result.Status,
		Message:     message,
		Timestamp:   result.Timestamp,
		Severity:    severity,
	}

	ahm.sendAlert(alert)
}

// sendAlert 알림 발송 (실제 구현에서는 알림 시스템과 연동)
func (ahm *AgentHealthMonitor) sendAlert(alert HealthAlert) {
	// TODO: 실제 알림 시스템과 연동
	// 예: 로그, 웹소켓, 이메일, 슬랙 등
	fmt.Printf("[HEALTH ALERT] %s: %s - %s\n", 
		alert.Severity, alert.AgentID, alert.Message)
}

// GetAllHealthChecks 모든 헬스체크 정보 조회
func (ahm *AgentHealthMonitor) GetAllHealthChecks() map[string]*AgentHealthCheck {
	ahm.mu.RLock()
	defer ahm.mu.RUnlock()

	result := make(map[string]*AgentHealthCheck)
	for agentID, hc := range ahm.healthChecks {
		hc.mu.RLock()
		hcCopy := *hc
		hcCopy.History = make([]HealthCheckResult, len(hc.History))
		copy(hcCopy.History, hc.History)
		hc.mu.RUnlock()
		
		result[agentID] = &hcCopy
	}

	return result
}

// GetHealthyAgents 정상 상태인 에이전트 목록 조회
func (ahm *AgentHealthMonitor) GetHealthyAgents() []string {
	ahm.mu.RLock()
	defer ahm.mu.RUnlock()

	var healthyAgents []string
	for agentID, hc := range ahm.healthChecks {
		hc.mu.RLock()
		if hc.Status == AgentHealthStatusHealthy {
			healthyAgents = append(healthyAgents, agentID)
		}
		hc.mu.RUnlock()
	}

	return healthyAgents
}

// GetUnhealthyAgents 비정상 상태인 에이전트 목록 조회
func (ahm *AgentHealthMonitor) GetUnhealthyAgents() []string {
	ahm.mu.RLock()
	defer ahm.mu.RUnlock()

	var unhealthyAgents []string
	for agentID, hc := range ahm.healthChecks {
		hc.mu.RLock()
		if hc.Status == AgentHealthStatusUnhealthy || hc.Status == AgentHealthStatusCritical {
			unhealthyAgents = append(unhealthyAgents, agentID)
		}
		hc.mu.RUnlock()
	}

	return unhealthyAgents
}

// IsRunning 모니터링 실행 상태 확인
func (ahm *AgentHealthMonitor) IsRunning() bool {
	ahm.mu.RLock()
	defer ahm.mu.RUnlock()
	return ahm.running
}

// GetDefaultHealthCheckConfig 기본 헬스체크 설정 반환
func GetDefaultHealthCheckConfig() HealthCheckConfig {
	return HealthCheckConfig{
		Enabled:          true,
		Command:          []string{"/bin/sh", "-c", "claude --version"},
		Interval:         30 * time.Second,
		Timeout:          30 * time.Second,
		StartPeriod:      30 * time.Second,
		Retries:          3,
		AutoRestart:      true,
		RestartThreshold: 3,
	}
}