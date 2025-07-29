package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aicli/aicli-web/internal/models"
)

// monitoringService MonitoringService 인터페이스의 구현체
type monitoringService struct {
	// 활성 모니터링 세션
	activeMonitors map[string]*monitorSession
	monitorsMutex  sync.RWMutex

	// 메트릭 수집기
	metricsCollector MetricsCollector

	// 이벤트 버스
	eventBus EventBus

	// 설정
	config MonitoringConfig
}

// monitorSession 단일 에이전트의 모니터링 세션
type monitorSession struct {
	agentID     string
	agent       *models.Agent
	ticker      *time.Ticker
	ctx         context.Context
	cancel      context.CancelFunc
	eventChan   chan AgentEvent
	lastMetrics *AgentMetrics
	lastHealth  *HealthStatus
}

// MonitoringConfig 모니터링 설정
type MonitoringConfig struct {
	HealthCheckInterval time.Duration
	MetricsInterval     time.Duration
	EventBufferSize     int
	MaxRetries          int
	Timeout             time.Duration
}

// 인터페이스는 interfaces.go에 정의됨

// NewMonitoringService 새 모니터링 서비스 생성
func NewMonitoringService(collector MetricsCollector, eventBus EventBus, config MonitoringConfig) MonitoringService {
	if config.HealthCheckInterval == 0 {
		config.HealthCheckInterval = 30 * time.Second
	}
	if config.MetricsInterval == 0 {
		config.MetricsInterval = 10 * time.Second
	}
	if config.EventBufferSize == 0 {
		config.EventBufferSize = 100
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.Timeout == 0 {
		config.Timeout = 10 * time.Second
	}

	return &monitoringService{
		activeMonitors:   make(map[string]*monitorSession),
		metricsCollector: collector,
		eventBus:         eventBus,
		config:           config,
	}
}

// CheckAgentHealth 에이전트 헬스체크 수행
func (m *monitoringService) CheckAgentHealth(ctx context.Context, agent *models.Agent) (HealthStatus, error) {
	ctx, cancel := context.WithTimeout(ctx, m.config.Timeout)
	defer cancel()

	healthStatus := HealthStatus{
		Status:    "unknown",
		LastCheck: time.Now(),
		Checks:    make([]HealthCheck, 0),
		Metrics:   make(map[string]string),
	}

	// 기본 헬스체크들 수행
	checks := []func(context.Context, *models.Agent) HealthCheck{
		m.checkAgentStatus,
		m.checkContainerHealth,
		m.checkResourceUsage,
		m.checkNetworkConnectivity,
	}

	allHealthy := true
	for _, checkFunc := range checks {
		check := checkFunc(ctx, agent)
		healthStatus.Checks = append(healthStatus.Checks, check)
		
		if check.Status != "healthy" {
			allHealthy = false
		}
	}

	// 전반적인 상태 결정
	if allHealthy {
		healthStatus.Status = "healthy"
	} else {
		hasUnhealthy := false
		for _, check := range healthStatus.Checks {
			if check.Status == "unhealthy" {
				hasUnhealthy = true
				break
			}
		}
		if hasUnhealthy {
			healthStatus.Status = "unhealthy"
		} else {
			healthStatus.Status = "starting"
		}
	}

	return healthStatus, nil
}

// StartHealthMonitoring 에이전트 헬스 모니터링 시작
func (m *monitoringService) StartHealthMonitoring(ctx context.Context, agent *models.Agent) error {
	m.monitorsMutex.Lock()
	defer m.monitorsMutex.Unlock()

	// 이미 모니터링 중인지 확인
	if _, exists := m.activeMonitors[agent.ID]; exists {
		return fmt.Errorf("agent %s is already being monitored", agent.ID)
	}

	// 모니터링 세션 생성
	sessionCtx, cancel := context.WithCancel(ctx)
	session := &monitorSession{
		agentID:   agent.ID,
		agent:     agent,
		ticker:    time.NewTicker(m.config.HealthCheckInterval),
		ctx:       sessionCtx,
		cancel:    cancel,
		eventChan: make(chan AgentEvent, m.config.EventBufferSize),
	}

	m.activeMonitors[agent.ID] = session

	// 백그라운드 모니터링 시작
	go m.runMonitoringLoop(session)

	return nil
}

// StopHealthMonitoring 에이전트 헬스 모니터링 중지
func (m *monitoringService) StopHealthMonitoring(ctx context.Context, agentID string) error {
	m.monitorsMutex.Lock()
	defer m.monitorsMutex.Unlock()

	session, exists := m.activeMonitors[agentID]
	if !exists {
		return fmt.Errorf("agent %s is not being monitored", agentID)
	}

	// 모니터링 세션 중지
	session.cancel()
	session.ticker.Stop()
	close(session.eventChan)

	delete(m.activeMonitors, agentID)

	return nil
}

// CollectAgentMetrics 에이전트 메트릭 수집
func (m *monitoringService) CollectAgentMetrics(ctx context.Context, agent *models.Agent) (AgentMetrics, error) {
	if m.metricsCollector == nil {
		return AgentMetrics{}, fmt.Errorf("metrics collector not configured")
	}

	return m.metricsCollector.CollectAgentMetrics(ctx, agent)
}

// GetMetricsHistory 메트릭 히스토리 조회
func (m *monitoringService) GetMetricsHistory(ctx context.Context, agentID string, period time.Duration) ([]AgentMetrics, error) {
	if m.metricsCollector == nil {
		return nil, fmt.Errorf("metrics collector not configured")
	}

	return m.metricsCollector.GetMetricsHistory(ctx, agentID, period)
}

// PublishAgentEvent 에이전트 이벤트 발행
func (m *monitoringService) PublishAgentEvent(ctx context.Context, event AgentEvent) error {
	if m.eventBus == nil {
		return fmt.Errorf("event bus not configured")
	}

	return m.eventBus.Publish(ctx, event)
}

// SubscribeToAgentEvents 에이전트 이벤트 구독
func (m *monitoringService) SubscribeToAgentEvents(ctx context.Context, agentID string) (<-chan AgentEvent, error) {
	if m.eventBus == nil {
		return nil, fmt.Errorf("event bus not configured")
	}

	return m.eventBus.Subscribe(ctx, agentID)
}

// runMonitoringLoop 모니터링 루프 실행
func (m *monitoringService) runMonitoringLoop(session *monitorSession) {
	defer func() {
		if r := recover(); r != nil {
			// 패닉 복구 및 로깅
			event := AgentEvent{
				Type:      AgentEventError,
				AgentID:   session.agentID,
				Timestamp: time.Now(),
				Message:   fmt.Sprintf("monitoring loop panic: %v", r),
			}
			_ = m.PublishAgentEvent(context.Background(), event)
		}
	}()

	for {
		select {
		case <-session.ctx.Done():
			return
		case <-session.ticker.C:
			m.performHealthCheck(session)
			m.collectMetrics(session)
		}
	}
}

// performHealthCheck 헬스체크 수행
func (m *monitoringService) performHealthCheck(session *monitorSession) {
	ctx, cancel := context.WithTimeout(context.Background(), m.config.Timeout)
	defer cancel()

	health, err := m.CheckAgentHealth(ctx, session.agent)
	if err != nil {
		event := AgentEvent{
			Type:      AgentEventHealthCheckFailed,
			AgentID:   session.agentID,
			Timestamp: time.Now(),
			Message:   fmt.Sprintf("health check failed: %v", err),
		}
		_ = m.PublishAgentEvent(ctx, event)
		return
	}

	session.lastHealth = &health

	// 상태 변화 시 이벤트 발행
	if session.lastHealth != nil && session.lastHealth.Status != health.Status {
		event := AgentEvent{
			Type:      AgentEventError, // 상태에 따라 적절한 이벤트 타입 설정
			AgentID:   session.agentID,
			Timestamp: time.Now(),
			Data:      health,
			Message:   fmt.Sprintf("health status changed to %s", health.Status),
		}
		_ = m.PublishAgentEvent(ctx, event)
	}
}

// collectMetrics 메트릭 수집
func (m *monitoringService) collectMetrics(session *monitorSession) {
	if m.metricsCollector == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), m.config.Timeout)
	defer cancel()

	metrics, err := m.metricsCollector.CollectAgentMetrics(ctx, session.agent)
	if err != nil {
		return // 에러는 로깅만 하고 계속 진행
	}

	session.lastMetrics = &metrics

	// 메트릭 저장
	_ = m.metricsCollector.StoreMetrics(ctx, metrics)
}

// 개별 헬스체크 함수들

func (m *monitoringService) checkAgentStatus(ctx context.Context, agent *models.Agent) HealthCheck {
	start := time.Now()
	check := HealthCheck{
		Name:      "agent_status",
		CheckedAt: start,
		Status:    "healthy",
	}

	// 에이전트 상태 확인
	switch agent.Status {
	case models.AgentStatusRunning:
		check.Status = "healthy"
		check.Message = "Agent is running"
	case models.AgentStatusStarting:
		check.Status = "starting"
		check.Message = "Agent is starting"
	case models.AgentStatusStopped:
		check.Status = "unhealthy"
		check.Message = "Agent is stopped"
	case models.AgentStatusError:
		check.Status = "unhealthy"
		check.Message = "Agent is in error state"
	default:
		check.Status = "unknown"
		check.Message = "Unknown agent status"
	}

	check.Duration = time.Since(start)
	return check
}

func (m *monitoringService) checkContainerHealth(ctx context.Context, agent *models.Agent) HealthCheck {
	start := time.Now()
	check := HealthCheck{
		Name:      "container_health",
		CheckedAt: start,
		Status:    "healthy",
		Message:   "Container health check not implemented",
	}

	// TODO: Docker adapter를 통한 컨테이너 헬스체크 구현
	
	check.Duration = time.Since(start)
	return check
}

func (m *monitoringService) checkResourceUsage(ctx context.Context, agent *models.Agent) HealthCheck {
	start := time.Now()
	check := HealthCheck{
		Name:      "resource_usage",
		CheckedAt: start,
		Status:    "healthy",
		Message:   "Resource usage check not implemented",
	}

	// TODO: 리소스 사용량 체크 구현
	
	check.Duration = time.Since(start)
	return check
}

func (m *monitoringService) checkNetworkConnectivity(ctx context.Context, agent *models.Agent) HealthCheck {
	start := time.Now()
	check := HealthCheck{
		Name:      "network_connectivity",
		CheckedAt: start,
		Status:    "healthy",
		Message:   "Network connectivity check not implemented",
	}

	// TODO: 네트워크 연결성 체크 구현
	
	check.Duration = time.Since(start)
	return check
}