package agent

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aicli/aicli-web/internal/docker"
	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/storage"
)

// PerformanceIntegration은 성능 최적화 시스템과 기존 에이전트 서비스의 통합점입니다
type PerformanceIntegration struct {
	// 성능 최적화 컴포넌트
	optimizer      *AgentPerformanceOptimizer
	benchmark      *PerformanceBenchmark
	
	// 기존 서비스 컴포넌트
	agentService   AgentService
	storage        storage.Storage
	dockerClient   docker.Client
	
	// 통합 설정
	config         IntegrationConfig
	
	// 상태 관리
	enabled        bool
	running        bool
}

// IntegrationConfig는 통합 설정입니다
type IntegrationConfig struct {
	// 성능 최적화 활성화
	EnableOptimization     bool                  `json:"enable_optimization"`
	EnableAutoScaling      bool                  `json:"enable_auto_scaling"`
	EnableProfiling        bool                  `json:"enable_profiling"`
	EnableBenchmarking     bool                  `json:"enable_benchmarking"`
	
	// 성능 설정
	PerformanceConfig      PerformanceConfig     `json:"performance_config"`
	BenchmarkConfig        BenchmarkConfig       `json:"benchmark_config"`
	
	// 모니터링 설정
	MetricsRetention       time.Duration         `json:"metrics_retention"`
	AlertingEnabled        bool                  `json:"alerting_enabled"`
	
	// 통합 정책
	OptimizationTriggers   []OptimizationTrigger `json:"optimization_triggers"`
	PerformanceThresholds  PerformanceThresholds `json:"performance_thresholds"`
}

// OptimizationTrigger는 최적화 트리거입니다
type OptimizationTrigger struct {
	Name               string        `json:"name"`
	Condition          string        `json:"condition"`
	Threshold          float64       `json:"threshold"`
	Action             string        `json:"action"`
	Cooldown           time.Duration `json:"cooldown"`
	LastTriggered      time.Time     `json:"last_triggered"`
}

// PerformanceThresholds는 성능 임계값입니다
type PerformanceThresholds struct {
	MaxAgentCreationTime   time.Duration `json:"max_agent_creation_time"`
	MinThroughput          float64       `json:"min_throughput"`
	MaxErrorRate           float64       `json:"max_error_rate"`
	MaxResourceUsage       ResourceLimits `json:"max_resource_usage"`
}

// ResourceLimits는 리소스 제한입니다
type ResourceLimits struct {
	CPUPercent     float64 `json:"cpu_percent"`
	MemoryBytes    int64   `json:"memory_bytes"`
	DiskBytes      int64   `json:"disk_bytes"`
	NetworkBps     int64   `json:"network_bps"`
}

// PerformanceEnhancedAgentService는 성능이 향상된 에이전트 서비스입니다
type PerformanceEnhancedAgentService struct {
	// 기본 서비스
	baseService    AgentService
	
	// 성능 최적화
	integration    *PerformanceIntegration
	
	// 성능 메트릭
	metrics        *EnhancedMetrics
}

// EnhancedMetrics는 향상된 메트릭입니다
type EnhancedMetrics struct {
	// 기본 메트릭
	TotalAgentsCreated     int64         `json:"total_agents_created"`
	SuccessfulCreations    int64         `json:"successful_creations"`
	FailedCreations        int64         `json:"failed_creations"`
	
	// 성능 메트릭
	AverageCreationTime    time.Duration `json:"average_creation_time"`
	P95CreationTime        time.Duration `json:"p95_creation_time"`
	P99CreationTime        time.Duration `json:"p99_creation_time"`
	
	// 최적화 메트릭
	OptimizationsApplied   int64         `json:"optimizations_applied"`
	PoolHitRate            float64       `json:"pool_hit_rate"`
	CacheHitRate           float64       `json:"cache_hit_rate"`
	
	// 리소스 메트릭
	CurrentCPUUsage        float64       `json:"current_cpu_usage"`
	CurrentMemoryUsage     int64         `json:"current_memory_usage"`
	PeakMemoryUsage        int64         `json:"peak_memory_usage"`
	
	// 시간 정보
	LastUpdated            time.Time     `json:"last_updated"`
	StartTime              time.Time     `json:"start_time"`
}

// NewPerformanceIntegration은 새로운 성능 통합을 생성합니다
func NewPerformanceIntegration(
	agentService AgentService,
	storage storage.Storage,
	dockerClient docker.Client,
	config IntegrationConfig,
) *PerformanceIntegration {
	
	// 성능 최적화기 생성
	optimizer := NewAgentPerformanceOptimizer(config.PerformanceConfig, dockerClient)
	
	// 벤치마크 도구 생성
	var benchmark *PerformanceBenchmark
	if config.EnableBenchmarking {
		benchmark = NewPerformanceBenchmark(config.BenchmarkConfig, optimizer)
	}
	
	return &PerformanceIntegration{
		optimizer:    optimizer,
		benchmark:    benchmark,
		agentService: agentService,
		storage:      storage,
		dockerClient: dockerClient,
		config:       config,
		enabled:      config.EnableOptimization,
	}
}

// Start는 성능 통합을 시작합니다
func (pi *PerformanceIntegration) Start() error {
	if !pi.enabled {
		log.Println("Performance optimization is disabled")
		return nil
	}
	
	if pi.running {
		return fmt.Errorf("performance integration already running")
	}
	
	// 성능 최적화기 시작
	if err := pi.optimizer.Start(); err != nil {
		return fmt.Errorf("failed to start performance optimizer: %w", err)
	}
	
	// 백그라운드 모니터링 시작
	go pi.monitoringLoop()
	go pi.optimizationLoop()
	
	pi.running = true
	log.Println("Performance integration started successfully")
	
	return nil
}

// Stop은 성능 통합을 중지합니다
func (pi *PerformanceIntegration) Stop() error {
	if !pi.running {
		return nil
	}
	
	// 성능 최적화기 중지
	if err := pi.optimizer.Stop(); err != nil {
		log.Printf("Error stopping performance optimizer: %v", err)
	}
	
	pi.running = false
	log.Println("Performance integration stopped")
	
	return nil
}

// OptimizeAgentCreation은 에이전트 생성을 최적화합니다
func (pi *PerformanceIntegration) OptimizeAgentCreation(
	ctx context.Context,
	req CreateAgentRequest,
) (*models.Agent, error) {
	
	if !pi.enabled {
		// 최적화가 비활성화된 경우 기본 서비스 사용
		return pi.agentService.CreateAgent(ctx, req)
	}
	
	start := time.Now()
	
	// 1. 성능 최적화 적용
	if err := pi.optimizer.OptimizeAgent(req.Name); err != nil {
		log.Printf("Optimization failed for agent %s: %v", req.Name, err)
		// 최적화 실패해도 기본 생성은 계속 진행
	}
	
	// 2. 기본 에이전트 생성
	agent, err := pi.agentService.CreateAgent(ctx, req)
	
	// 3. 성능 메트릭 기록
	duration := time.Since(start)
	pi.recordCreationMetrics(agent != nil, duration)
	
	// 4. 최적화 트리거 평가
	go pi.evaluateOptimizationTriggers()
	
	return agent, err
}

// GetPerformanceMetrics는 성능 메트릭을 반환합니다
func (pi *PerformanceIntegration) GetPerformanceMetrics() *PerformanceMetrics {
	if !pi.enabled {
		return nil
	}
	
	return pi.optimizer.GetPerformanceMetrics()
}

// GetSystemStatus는 시스템 상태를 반환합니다
func (pi *PerformanceIntegration) GetSystemStatus() *SystemStatus {
	if !pi.enabled {
		return &SystemStatus{
			CPUUsage:       0,
			MemoryUsage:    0,
			DiskUsage:      0,
			ActiveAgents:   0,
			QueuedRequests: 0,
			LastUpdated:    time.Now(),
		}
	}
	
	return pi.optimizer.GetSystemStatus()
}

// RunBenchmark는 성능 벤치마크를 실행합니다
func (pi *PerformanceIntegration) RunBenchmark() (*BenchmarkResults, error) {
	if !pi.enabled || pi.benchmark == nil {
		return nil, fmt.Errorf("benchmarking is not enabled")
	}
	
	return pi.benchmark.RunBenchmark()
}

// 내부 메서드들

func (pi *PerformanceIntegration) monitoringLoop() {
	ticker := time.NewTicker(pi.config.PerformanceConfig.MetricsInterval)
	defer ticker.Stop()
	
	for pi.running {
		select {
		case <-ticker.C:
			pi.collectAndAnalyzeMetrics()
		}
	}
}

func (pi *PerformanceIntegration) optimizationLoop() {
	ticker := time.NewTicker(pi.config.PerformanceConfig.OptimizationInterval)
	defer ticker.Stop()
	
	for pi.running {
		select {
		case <-ticker.C:
			pi.runOptimizationCycle()
		}
	}
}

func (pi *PerformanceIntegration) collectAndAnalyzeMetrics() {
	// 시스템 상태 수집
	systemStatus := pi.GetSystemStatus()
	if systemStatus == nil {
		return
	}
	
	// 성능 임계값 확인
	pi.checkPerformanceThresholds(systemStatus)
	
	// 알림 처리
	if pi.config.AlertingEnabled {
		pi.processAlerts(systemStatus)
	}
}

func (pi *PerformanceIntegration) runOptimizationCycle() {
	// 자동 최적화 실행
	log.Println("Running optimization cycle")
	
	// 메트릭 기반 최적화 결정
	metrics := pi.GetPerformanceMetrics()
	if metrics != nil {
		pi.applyMetricsBasedOptimization(metrics)
	}
}

func (pi *PerformanceIntegration) checkPerformanceThresholds(status *SystemStatus) {
	thresholds := pi.config.PerformanceThresholds
	
	// CPU 사용률 확인
	if status.CPUUsage > thresholds.MaxResourceUsage.CPUPercent {
		pi.triggerOptimization("high_cpu", status.CPUUsage)
	}
	
	// 메모리 사용량 확인
	if status.MemoryUsage > float64(thresholds.MaxResourceUsage.MemoryBytes) {
		pi.triggerOptimization("high_memory", status.MemoryUsage)
	}
}

func (pi *PerformanceIntegration) processAlerts(status *SystemStatus) {
	// 알림 처리 로직
	// 실제 구현에서는 알림 시스템과 연동
}

func (pi *PerformanceIntegration) applyMetricsBasedOptimization(metrics *PerformanceMetrics) {
	// 메트릭 기반 최적화 적용
	avgCreationTime := metrics.AverageCreationTime
	
	if avgCreationTime > pi.config.PerformanceThresholds.MaxAgentCreationTime {
		log.Printf("Agent creation time (%v) exceeds threshold (%v), applying optimizations",
			avgCreationTime, pi.config.PerformanceThresholds.MaxAgentCreationTime)
		
		// 컨테이너 풀 웜업
		pi.optimizer.containerPool.WarmupContainers()
		
		// 워크트리 풀 최적화
		pi.optimizer.worktreePool.Optimize()
	}
}

func (pi *PerformanceIntegration) triggerOptimization(triggerType string, value float64) {
	// 최적화 트리거 실행
	for _, trigger := range pi.config.OptimizationTriggers {
		if trigger.Name == triggerType {
			// 쿨다운 확인
			if time.Since(trigger.LastTriggered) < trigger.Cooldown {
				continue
			}
			
			// 임계값 확인
			if value > trigger.Threshold {
				pi.executeOptimizationAction(trigger.Action)
				trigger.LastTriggered = time.Now()
			}
		}
	}
}

func (pi *PerformanceIntegration) executeOptimizationAction(action string) {
	switch action {
	case "scale_up":
		pi.optimizer.autoScaler.ScaleUp()
	case "scale_down":
		pi.optimizer.autoScaler.ScaleDown()
	case "warmup_pools":
		pi.optimizer.containerPool.WarmupContainers()
		pi.optimizer.worktreePool.Optimize()
	case "gc":
		pi.optimizer.memoryManager.RecycleUnusedObjects()
	default:
		log.Printf("Unknown optimization action: %s", action)
	}
}

func (pi *PerformanceIntegration) recordCreationMetrics(success bool, duration time.Duration) {
	metrics := pi.optimizer.GetPerformanceMetrics()
	if metrics != nil {
		metrics.RecordAgentCreation(duration)
		
		if success {
			metrics.TotalAgentsCreated.Add(1)
		} else {
			metrics.FailedCreations.Add(1)
		}
	}
}

func (pi *PerformanceIntegration) evaluateOptimizationTriggers() {
	// 최적화 트리거 평가 로직
	// 비동기로 실행되어 성능에 영향을 주지 않음
}

// NewPerformanceEnhancedAgentService는 성능이 향상된 에이전트 서비스를 생성합니다
func NewPerformanceEnhancedAgentService(
	baseService AgentService,
	integration *PerformanceIntegration,
) *PerformanceEnhancedAgentService {
	
	return &PerformanceEnhancedAgentService{
		baseService: baseService,
		integration: integration,
		metrics:     NewEnhancedMetrics(),
	}
}

// CreateAgent는 성능이 최적화된 에이전트 생성을 수행합니다
func (peas *PerformanceEnhancedAgentService) CreateAgent(
	ctx context.Context,
	req CreateAgentRequest,
) (*models.Agent, error) {
	
	// 성능 통합을 통한 최적화된 생성
	agent, err := peas.integration.OptimizeAgentCreation(ctx, req)
	
	// 메트릭 업데이트
	peas.metrics.TotalAgentsCreated++
	if err != nil {
		peas.metrics.FailedCreations++
	} else {
		peas.metrics.SuccessfulCreations++
	}
	peas.metrics.LastUpdated = time.Now()
	
	return agent, err
}

// GetAgent는 에이전트를 조회합니다
func (peas *PerformanceEnhancedAgentService) GetAgent(
	ctx context.Context,
	agentID string,
) (*models.Agent, error) {
	return peas.baseService.GetAgent(ctx, agentID)
}

// ListAgents는 에이전트 목록을 조회합니다
func (peas *PerformanceEnhancedAgentService) ListAgents(
	ctx context.Context,
	projectID string,
) ([]*models.Agent, error) {
	return peas.baseService.GetAgentByProjectID(ctx, projectID)
}

// UpdateAgent는 에이전트를 업데이트합니다
func (peas *PerformanceEnhancedAgentService) UpdateAgent(
	ctx context.Context,
	agentID string,
	req UpdateAgentRequest,
) (*models.Agent, error) {
	return peas.baseService.UpdateAgent(ctx, agentID, req)
}

// DeleteAgent는 에이전트를 삭제합니다
func (peas *PerformanceEnhancedAgentService) DeleteAgent(
	ctx context.Context,
	agentID string,
) error {
	return peas.baseService.DeleteAgent(ctx, agentID)
}

// StartAgent는 에이전트를 시작합니다
func (peas *PerformanceEnhancedAgentService) StartAgent(
	ctx context.Context,
	agentID string,
) error {
	return peas.baseService.StartAgent(ctx, agentID)
}

// StopAgent는 에이전트를 중지합니다
func (peas *PerformanceEnhancedAgentService) StopAgent(
	ctx context.Context,
	agentID string,
) error {
	return peas.baseService.StopAgent(ctx, agentID)
}

// RestartAgent는 에이전트를 재시작합니다
func (peas *PerformanceEnhancedAgentService) RestartAgent(
	ctx context.Context,
	agentID string,
) error {
	return peas.baseService.RestartAgent(ctx, agentID)
}

// GetAgentStatus는 에이전트 상태를 조회합니다
func (peas *PerformanceEnhancedAgentService) GetAgentStatus(
	ctx context.Context,
	agentID string,
) (AgentStatusInfo, error) {
	return peas.baseService.GetAgentStatus(ctx, agentID)
}

// GetAgentHealth는 에이전트 건강 상태를 조회합니다
func (peas *PerformanceEnhancedAgentService) GetAgentHealth(
	ctx context.Context,
	agentID string,
) (HealthStatus, error) {
	return peas.baseService.GetHealthStatus(ctx, agentID)
}

// GetAgentMetrics는 에이전트 메트릭을 조회합니다
func (peas *PerformanceEnhancedAgentService) GetAgentMetrics(
	ctx context.Context,
	agentID string,
) (AgentMetrics, error) {
	return peas.baseService.GetAgentMetrics(ctx, agentID)
}

// GetEnhancedMetrics는 향상된 메트릭을 반환합니다
func (peas *PerformanceEnhancedAgentService) GetEnhancedMetrics() *EnhancedMetrics {
	// 최신 성능 메트릭으로 업데이트
	if perfMetrics := peas.integration.GetPerformanceMetrics(); perfMetrics != nil {
		peas.metrics.AverageCreationTime = perfMetrics.AverageCreationTime
		peas.metrics.P95CreationTime = perfMetrics.P95CreationTime
		peas.metrics.P99CreationTime = perfMetrics.P99CreationTime
	}
	
	if systemStatus := peas.integration.GetSystemStatus(); systemStatus != nil {
		peas.metrics.CurrentCPUUsage = systemStatus.CPUUsage
		peas.metrics.CurrentMemoryUsage = int64(systemStatus.MemoryUsage)
	}
	
	return peas.metrics
}

// NewEnhancedMetrics는 새로운 향상된 메트릭을 생성합니다
func NewEnhancedMetrics() *EnhancedMetrics {
	return &EnhancedMetrics{
		StartTime:   time.Now(),
		LastUpdated: time.Now(),
	}
}

// DefaultIntegrationConfig는 기본 통합 설정을 반환합니다
func DefaultIntegrationConfig() IntegrationConfig {
	return IntegrationConfig{
		EnableOptimization:  true,
		EnableAutoScaling:   true,
		EnableProfiling:     true,
		EnableBenchmarking:  false, // 기본적으로 비활성화
		PerformanceConfig:   DefaultPerformanceConfig(),
		BenchmarkConfig:     DefaultBenchmarkConfig(),
		MetricsRetention:    24 * time.Hour,
		AlertingEnabled:     true,
		OptimizationTriggers: []OptimizationTrigger{
			{
				Name:      "high_cpu",
				Condition: "cpu_usage > threshold",
				Threshold: 80.0,
				Action:    "scale_up",
				Cooldown:  5 * time.Minute,
			},
			{
				Name:      "high_memory",
				Condition: "memory_usage > threshold",
				Threshold: 80.0,
				Action:    "gc",
				Cooldown:  2 * time.Minute,
			},
			{
				Name:      "slow_creation",
				Condition: "creation_time > threshold",
				Threshold: 5.0, // 5초
				Action:    "warmup_pools",
				Cooldown:  10 * time.Minute,
			},
		},
		PerformanceThresholds: PerformanceThresholds{
			MaxAgentCreationTime: 5 * time.Second,
			MinThroughput:        10.0, // 10 agents per second
			MaxErrorRate:         1.0,  // 1%
			MaxResourceUsage: ResourceLimits{
				CPUPercent:  80.0,
				MemoryBytes: 2 * 1024 * 1024 * 1024, // 2GB
				DiskBytes:   10 * 1024 * 1024 * 1024, // 10GB
				NetworkBps:  100 * 1024 * 1024, // 100MB/s
			},
		},
	}
}