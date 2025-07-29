package agent

import (
	"time"

	"github.com/aicli/aicli-web/internal/git"
	"github.com/aicli/aicli-web/internal/storage"
)

// AgentServiceFactory 에이전트 서비스 팩토리
type AgentServiceFactory struct {
	storage         storage.Storage
	dockerAdapter   DockerAdapter
	worktreeManager git.WorktreeManager
}

// NewAgentServiceFactory 새 에이전트 서비스 팩토리 생성
func NewAgentServiceFactory(
	storage storage.Storage,
	dockerAdapter DockerAdapter,
	worktreeManager git.WorktreeManager,
) *AgentServiceFactory {
	return &AgentServiceFactory{
		storage:         storage,
		dockerAdapter:   dockerAdapter,
		worktreeManager: worktreeManager,
	}
}

// CreateCompleteAgentService 완전한 에이전트 서비스 생성 (모든 컴포넌트 통합)
func (f *AgentServiceFactory) CreateCompleteAgentService() AgentService {
	// 1. EventBus 생성
	eventBusConfig := EventBusConfig{
		BufferSize:       100,
		MaxHistorySize:   1000,
		HistoryRetention: 24 * time.Hour,
		PublishTimeout:   5 * time.Second,
		EnableHistory:    true,
	}
	eventBus := NewBasicEventBus(eventBusConfig)

	// 2. MetricsCollector 생성
	metricsCollector := NewBasicMetricsCollector(f.dockerAdapter)

	// 3. MonitoringService 생성
	monitoringConfig := MonitoringConfig{
		HealthCheckInterval: 30 * time.Second,
		MetricsInterval:     60 * time.Second,
		EventBufferSize:     50,
		MaxRetries:          3,
		Timeout:             30 * time.Second,
	}
	monitoring := NewMonitoringService(metricsCollector, eventBus, monitoringConfig)

	// 4. EventPublisher 생성
	eventPublisher := NewBasicEventPublisher(eventBus)

	// 5. AgentService 생성
	return NewAgentService(
		f.storage,
		f.dockerAdapter,
		monitoring,
		eventPublisher,
		f.worktreeManager,
	)
}

// CreateMinimalAgentService 최소한의 에이전트 서비스 생성 (테스트용)
func (f *AgentServiceFactory) CreateMinimalAgentService() AgentService {
	return NewAgentService(
		f.storage,
		f.dockerAdapter,
		nil, // monitoring 없음
		nil, // eventPublisher 없음
		f.worktreeManager,
	)
}

// CreateAgentServiceWithCustomComponents 커스텀 컴포넌트로 에이전트 서비스 생성
func (f *AgentServiceFactory) CreateAgentServiceWithCustomComponents(
	monitoring MonitoringService,
	eventPublisher EventPublisher,
) AgentService {
	return NewAgentService(
		f.storage,
		f.dockerAdapter,
		monitoring,
		eventPublisher,
		f.worktreeManager,
	)
}

// CreateStandaloneEventBus 독립적인 EventBus 생성
func CreateStandaloneEventBus(config EventBusConfig) EventBus {
	return NewBasicEventBus(config)
}

// CreateStandaloneMetricsCollector 독립적인 MetricsCollector 생성
func CreateStandaloneMetricsCollector(dockerAdapter DockerAdapter) MetricsCollector {
	return NewBasicMetricsCollector(dockerAdapter)
}

// CreateStandaloneMonitoringService 독립적인 MonitoringService 생성
func CreateStandaloneMonitoringService(
	collector MetricsCollector,
	eventBus EventBus,
	config MonitoringConfig,
) MonitoringService {
	return NewMonitoringService(collector, eventBus, config)
}

// CreateStandaloneEventPublisher 독립적인 EventPublisher 생성
func CreateStandaloneEventPublisher(eventBus EventBus) EventPublisher {
	return NewBasicEventPublisher(eventBus)
}

// DefaultEventBusConfig 기본 EventBus 설정 반환
func DefaultEventBusConfig() EventBusConfig {
	return EventBusConfig{
		BufferSize:       100,
		MaxHistorySize:   1000,
		HistoryRetention: 24 * time.Hour,
		PublishTimeout:   5 * time.Second,
		EnableHistory:    true,
	}
}

// DefaultMonitoringConfig 기본 Monitoring 설정 반환
func DefaultMonitoringConfig() MonitoringConfig {
	return MonitoringConfig{
		HealthCheckInterval: 30 * time.Second,
		MetricsInterval:     60 * time.Second,
		EventBufferSize:     50,
		MaxRetries:          3,
		Timeout:             30 * time.Second,
	}
}

// IntegratedAgentSystemComponents 통합된 에이전트 시스템의 모든 컴포넌트
type IntegratedAgentSystemComponents struct {
	AgentService     AgentService
	EventBus         EventBus
	EventPublisher   EventPublisher
	MetricsCollector MetricsCollector
	MonitoringService MonitoringService
}

// CreateIntegratedAgentSystem 모든 컴포넌트가 통합된 완전한 에이전트 시스템 생성
func (f *AgentServiceFactory) CreateIntegratedAgentSystem() *IntegratedAgentSystemComponents {
	// EventBus 생성
	eventBus := NewBasicEventBus(DefaultEventBusConfig())
	
	// MetricsCollector 생성
	metricsCollector := NewBasicMetricsCollector(f.dockerAdapter)
	
	// MonitoringService 생성
	monitoring := NewMonitoringService(metricsCollector, eventBus, DefaultMonitoringConfig())
	
	// EventPublisher 생성
	eventPublisher := NewBasicEventPublisher(eventBus)
	
	// AgentService 생성
	agentService := NewAgentService(
		f.storage,
		f.dockerAdapter,
		monitoring,
		eventPublisher,
		f.worktreeManager,
	)

	return &IntegratedAgentSystemComponents{
		AgentService:      agentService,
		EventBus:          eventBus,
		EventPublisher:    eventPublisher,
		MetricsCollector:  metricsCollector,
		MonitoringService: monitoring,
	}
}