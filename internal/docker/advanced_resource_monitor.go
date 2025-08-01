package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
)

// ResourceMetrics 리소스 메트릭 구조체
type ResourceMetrics struct {
	ContainerID       string                 `json:"container_id"`
	AgentID          string                 `json:"agent_id"`
	Timestamp        time.Time              `json:"timestamp"`
	CPUUsage         CPUMetrics             `json:"cpu_usage"`
	MemoryUsage      MemoryMetrics          `json:"memory_usage"`
	NetworkIO        NetworkMetrics         `json:"network_io"`
	DiskIO           DiskMetrics            `json:"disk_io"`
	PIDs             PIDMetrics             `json:"pids"`
	ResourceLimits   ResourceLimits         `json:"resource_limits"`
	Utilization      UtilizationMetrics     `json:"utilization"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// CPUMetrics CPU 사용량 메트릭
type CPUMetrics struct {
	UsagePercent    float64 `json:"usage_percent"`
	UsageNanoseconds uint64  `json:"usage_nanoseconds"`
	SystemUsage     uint64  `json:"system_usage"`  
	UserUsage       uint64  `json:"user_usage"`
	ThrottleCount   uint64  `json:"throttle_count"`
	ThrottleTime    uint64  `json:"throttle_time"`
}

// MemoryMetrics 메모리 사용량 메트릭
type MemoryMetrics struct {
	UsageBytes      uint64  `json:"usage_bytes"`
	LimitBytes      uint64  `json:"limit_bytes"`
	UsagePercent    float64 `json:"usage_percent"`
	CacheBytes      uint64  `json:"cache_bytes"`
	RSSBytes        uint64  `json:"rss_bytes"`
	SwapUsageBytes  uint64  `json:"swap_usage_bytes"`
	SwapLimitBytes  uint64  `json:"swap_limit_bytes"`
	PageFaults      uint64  `json:"page_faults"`
	MajorPageFaults uint64  `json:"major_page_faults"`
}

// NetworkMetrics 네트워크 I/O 메트릭
type NetworkMetrics struct {
	RxBytes   uint64  `json:"rx_bytes"`
	TxBytes   uint64  `json:"tx_bytes"`
	RxPackets uint64  `json:"rx_packets"`
	TxPackets uint64  `json:"tx_packets"`
	RxErrors  uint64  `json:"rx_errors"`
	TxErrors  uint64  `json:"tx_errors"`
	RxDropped uint64  `json:"rx_dropped"`
	TxDropped uint64  `json:"tx_dropped"`
	RxRate    float64 `json:"rx_rate_bps"`
	TxRate    float64 `json:"tx_rate_bps"`
}

// DiskMetrics 디스크 I/O 메트릭
type DiskMetrics struct {
	ReadBytes       uint64  `json:"read_bytes"`
	WriteBytes      uint64  `json:"write_bytes"`
	ReadOperations  uint64  `json:"read_operations"`
	WriteOperations uint64  `json:"write_operations"`
	ReadRate        float64 `json:"read_rate_bps"`
	WriteRate       float64 `json:"write_rate_bps"`
	IOWaitTime      uint64  `json:"io_wait_time"`
}

// PIDMetrics PID 사용량 메트릭
type PIDMetrics struct {
	Current uint64 `json:"current"`
	Limit   uint64 `json:"limit"`
}

// ResourceLimits 리소스 제한
type ResourceLimits struct {
	CPUQuota      int64  `json:"cpu_quota"`
	CPUPeriod     uint64 `json:"cpu_period"`
	MemoryLimit   int64  `json:"memory_limit"`
	SwapLimit     int64  `json:"swap_limit"`
	PidsLimit     int64  `json:"pids_limit"`
	CPUShares     uint64 `json:"cpu_shares"`
}

// UtilizationMetrics 리소스 사용률 메트릭
type UtilizationMetrics struct {
	CPUUtilization     float64 `json:"cpu_utilization"`
	MemoryUtilization  float64 `json:"memory_utilization"`
	NetworkUtilization float64 `json:"network_utilization"`
	DiskUtilization    float64 `json:"disk_utilization"`
	OverallScore       float64 `json:"overall_score"`
}

// ResourceThreshold 리소스 임계값
type ResourceThreshold struct {
	CPUWarning     float64 `json:"cpu_warning"`
	CPUCritical    float64 `json:"cpu_critical"`
	MemoryWarning  float64 `json:"memory_warning"`
	MemoryCritical float64 `json:"memory_critical"`
	NetworkWarning float64 `json:"network_warning"`
	NetworkCritical float64 `json:"network_critical"`
}

// DefaultResourceThreshold 기본 리소스 임계값
func DefaultResourceThreshold() ResourceThreshold {
	return ResourceThreshold{
		CPUWarning:      70.0,
		CPUCritical:     90.0,
		MemoryWarning:   80.0,
		MemoryCritical:  95.0,
		NetworkWarning:  100 * 1024 * 1024, // 100MB/s
		NetworkCritical: 500 * 1024 * 1024, // 500MB/s
	}
}

// ResourceAlert 리소스 알람
type ResourceAlert struct {
	AgentID     string               `json:"agent_id"`
	ContainerID string               `json:"container_id"`
	Timestamp   time.Time            `json:"timestamp"`
	Level       AlertLevel           `json:"level"`
	Resource    string               `json:"resource"`
	Value       float64              `json:"value"`
	Threshold   float64              `json:"threshold"`
	Message     string               `json:"message"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// AlertLevel 알람 레벨
type AlertLevel string

const (
	AlertLevelInfo     AlertLevel = "info"
	AlertLevelWarning  AlertLevel = "warning"
	AlertLevelCritical AlertLevel = "critical"
)

// AlertHandler 알람 핸들러
type AlertHandler func(alert ResourceAlert)

// MonitoringConfig 모니터링 설정
type MonitoringConfig struct {
	Interval       time.Duration     `json:"interval"`
	HistorySize    int               `json:"history_size"`
	Threshold      ResourceThreshold `json:"threshold"`
	EnableAlerting bool              `json:"enable_alerting"`
	EnableAutoScale bool             `json:"enable_auto_scale"`
}

// DefaultMonitoringConfig 기본 모니터링 설정
func DefaultMonitoringConfig() MonitoringConfig {
	return MonitoringConfig{
		Interval:        10 * time.Second,
		HistorySize:     100,
		Threshold:       DefaultResourceThreshold(),
		EnableAlerting:  true,
		EnableAutoScale: false,
	}
}

// ContainerResourceState 컨테이너 리소스 상태
type ContainerResourceState struct {
	AgentID           string             `json:"agent_id"`
	ContainerID       string             `json:"container_id"`
	Config            MonitoringConfig   `json:"config"`
	MetricsHistory    []ResourceMetrics  `json:"metrics_history"`
	CurrentMetrics    *ResourceMetrics   `json:"current_metrics"`
	LastCollected     time.Time          `json:"last_collected"`
	AlertsGenerated   []ResourceAlert    `json:"alerts_generated"`
	IsMonitoring      bool               `json:"is_monitoring"`
	CollectionErrors  int                `json:"collection_errors"`
	PreviousMetrics   *ResourceMetrics   `json:"-"`
}

// AdvancedResourceMonitor 고급 리소스 모니터
type AdvancedResourceMonitor struct {
	client          *Client
	containerStates map[string]*ContainerResourceState
	alertHandlers   []AlertHandler
	mu              sync.RWMutex
	ctx             context.Context
	cancel          context.CancelFunc
	globalConfig    MonitoringConfig
}

// NewAdvancedResourceMonitor 새로운 고급 리소스 모니터 생성
func NewAdvancedResourceMonitor(client *Client) *AdvancedResourceMonitor {
	ctx, cancel := context.WithCancel(context.Background())

	arm := &AdvancedResourceMonitor{
		client:          client,
		containerStates: make(map[string]*ContainerResourceState),
		alertHandlers:   make([]AlertHandler, 0),
		ctx:             ctx,
		cancel:          cancel,
		globalConfig:    DefaultMonitoringConfig(),
	}

	// 주기적 모니터링 시작
	go arm.startPeriodicMonitoring()

	return arm
}

// RegisterContainer 컨테이너 등록
func (arm *AdvancedResourceMonitor) RegisterContainer(agentID, containerID string, config MonitoringConfig) {
	arm.mu.Lock()
	defer arm.mu.Unlock()

	arm.containerStates[agentID] = &ContainerResourceState{
		AgentID:         agentID,
		ContainerID:     containerID,
		Config:          config,
		MetricsHistory:  make([]ResourceMetrics, 0, config.HistorySize),
		AlertsGenerated: make([]ResourceAlert, 0),
		IsMonitoring:    true,
		LastCollected:   time.Now(),
	}
}

// UnregisterContainer 컨테이너 등록 해제
func (arm *AdvancedResourceMonitor) UnregisterContainer(agentID string) {
	arm.mu.Lock()
	defer arm.mu.Unlock()

	delete(arm.containerStates, agentID)
}

// AddAlertHandler 알람 핸들러 추가
func (arm *AdvancedResourceMonitor) AddAlertHandler(handler AlertHandler) {
	arm.mu.Lock()
	defer arm.mu.Unlock()

	arm.alertHandlers = append(arm.alertHandlers, handler)
}

// GetResourceMetrics 현재 리소스 메트릭 조회
func (arm *AdvancedResourceMonitor) GetResourceMetrics(agentID string) (*ResourceMetrics, bool) {
	arm.mu.RLock()
	defer arm.mu.RUnlock()

	state, exists := arm.containerStates[agentID]
	if !exists || state.CurrentMetrics == nil {
		return nil, false
	}

	return state.CurrentMetrics, true
}

// GetResourceHistory 리소스 사용 이력 조회
func (arm *AdvancedResourceMonitor) GetResourceHistory(agentID string) ([]ResourceMetrics, bool) {
	arm.mu.RLock()
	defer arm.mu.RUnlock()

	state, exists := arm.containerStates[agentID]
	if !exists {
		return nil, false
	}

	// 복사본 반환
	history := make([]ResourceMetrics, len(state.MetricsHistory))
	copy(history, state.MetricsHistory)

	return history, true
}

// GetContainerState 컨테이너 상태 조회
func (arm *AdvancedResourceMonitor) GetContainerState(agentID string) (*ContainerResourceState, bool) {
	arm.mu.RLock()
	defer arm.mu.RUnlock()

	state, exists := arm.containerStates[agentID]
	if !exists {
		return nil, false
	}

	// 복사본 반환 (깊은 복사)
	stateCopy := *state
	stateCopy.MetricsHistory = make([]ResourceMetrics, len(state.MetricsHistory))
	copy(stateCopy.MetricsHistory, state.MetricsHistory)
	
	stateCopy.AlertsGenerated = make([]ResourceAlert, len(state.AlertsGenerated))
	copy(stateCopy.AlertsGenerated, state.AlertsGenerated)

	return &stateCopy, true
}

// Close 모니터 종료
func (arm *AdvancedResourceMonitor) Close() {
	arm.cancel()
}

// startPeriodicMonitoring 주기적 모니터링 시작
func (arm *AdvancedResourceMonitor) startPeriodicMonitoring() {
	ticker := time.NewTicker(arm.globalConfig.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-arm.ctx.Done():
			return
		case <-ticker.C:
			arm.collectAllMetrics()
		}
	}
}

// collectAllMetrics 모든 컨테이너의 메트릭 수집
func (arm *AdvancedResourceMonitor) collectAllMetrics() {
	arm.mu.RLock()
	agentIDs := make([]string, 0, len(arm.containerStates))
	for agentID, state := range arm.containerStates {
		if state.IsMonitoring {
			agentIDs = append(agentIDs, agentID)
		}
	}
	arm.mu.RUnlock()

	// 각 컨테이너에 대해 메트릭 수집
	for _, agentID := range agentIDs {
		go arm.collectContainerMetrics(agentID)
	}
}

// collectContainerMetrics 특정 컨테이너의 메트릭 수집
func (arm *AdvancedResourceMonitor) collectContainerMetrics(agentID string) {
	arm.mu.RLock()
	state, exists := arm.containerStates[agentID]
	if !exists {
		arm.mu.RUnlock()
		return
	}
	containerID := state.ContainerID
	config := state.Config
	arm.mu.RUnlock()

	// Docker stats 조회
	stats, err := arm.getContainerStats(containerID)
	if err != nil {
		arm.mu.Lock()
		if state, exists := arm.containerStates[agentID]; exists {
			state.CollectionErrors++
		}
		arm.mu.Unlock()
		return
	}

	// 메트릭 변환
	metrics := arm.convertStatsToMetrics(agentID, containerID, stats)

	// 이전 메트릭과 비교하여 비율 계산
	arm.mu.Lock()
	if currentState, exists := arm.containerStates[agentID]; exists {
		if currentState.PreviousMetrics != nil {
			arm.calculateRates(metrics, currentState.PreviousMetrics)
		}
		
		// 사용률 계산
		arm.calculateUtilization(metrics)

		// 메트릭 저장
		currentState.CurrentMetrics = metrics
		currentState.PreviousMetrics = metrics
		currentState.LastCollected = time.Now()
		currentState.CollectionErrors = 0

		// 히스토리 업데이트
		currentState.MetricsHistory = append(currentState.MetricsHistory, *metrics)
		if len(currentState.MetricsHistory) > config.HistorySize {
			currentState.MetricsHistory = currentState.MetricsHistory[1:]
		}
	}
	arm.mu.Unlock()

	// 알람 확인
	if config.EnableAlerting {
		go arm.checkAlerts(agentID, metrics)
	}

	// 자동 스케일링
	if config.EnableAutoScale {
		go arm.checkAutoScale(agentID, metrics)
	}
}

// getContainerStats Docker stats 조회
func (arm *AdvancedResourceMonitor) getContainerStats(containerID string) (*types.StatsJSON, error) {
	ctx, cancel := context.WithTimeout(arm.ctx, 10*time.Second)
	defer cancel()

	statsReader, err := arm.client.cli.ContainerStats(ctx, containerID, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get container stats: %w", err)
	}
	defer statsReader.Body.Close()

	var stats types.StatsJSON
	if err := json.NewDecoder(statsReader.Body).Decode(&stats); err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to decode stats: %w", err)
	}

	return &stats, nil
}

// convertStatsToMetrics Docker stats를 메트릭으로 변환
func (arm *AdvancedResourceMonitor) convertStatsToMetrics(agentID, containerID string, stats *types.StatsJSON) *ResourceMetrics {
	metrics := &ResourceMetrics{
		ContainerID: containerID,
		AgentID:     agentID,
		Timestamp:   time.Now(),
		Metadata:    make(map[string]interface{}),
	}

	// CPU 메트릭
	if stats.CPUStats.SystemUsage > 0 {
		cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
		systemDelta := float64(stats.CPUStats.SystemUsage - stats.PreCPUStats.SystemUsage)
		
		if systemDelta > 0 {
			metrics.CPUUsage.UsagePercent = (cpuDelta / systemDelta) * float64(len(stats.CPUStats.CPUUsage.PercpuUsage)) * 100.0
		}
		
		metrics.CPUUsage.UsageNanoseconds = stats.CPUStats.CPUUsage.TotalUsage
		metrics.CPUUsage.SystemUsage = stats.CPUStats.CPUUsage.UsageInKernelmode
		metrics.CPUUsage.UserUsage = stats.CPUStats.CPUUsage.UsageInUsermode
		
		if stats.CPUStats.ThrottlingData.Periods > 0 {
			metrics.CPUUsage.ThrottleCount = stats.CPUStats.ThrottlingData.ThrottledPeriods
			metrics.CPUUsage.ThrottleTime = stats.CPUStats.ThrottlingData.ThrottledTime
		}
	}

	// 메모리 메트릭
	if stats.MemoryStats.Limit > 0 {
		metrics.MemoryUsage.UsageBytes = stats.MemoryStats.Usage
		metrics.MemoryUsage.LimitBytes = stats.MemoryStats.Limit
		metrics.MemoryUsage.UsagePercent = (float64(stats.MemoryStats.Usage) / float64(stats.MemoryStats.Limit)) * 100.0
		
		if stats.MemoryStats.Stats != nil {
			metrics.MemoryUsage.CacheBytes = stats.MemoryStats.Stats["cache"]
			metrics.MemoryUsage.RSSBytes = stats.MemoryStats.Stats["rss"]
			metrics.MemoryUsage.PageFaults = stats.MemoryStats.Stats["pgfault"]
			metrics.MemoryUsage.MajorPageFaults = stats.MemoryStats.Stats["pgmajfault"]
		}
	}

	// 네트워크 메트릭
	for _, netStats := range stats.Networks {
		metrics.NetworkIO.RxBytes += netStats.RxBytes
		metrics.NetworkIO.TxBytes += netStats.TxBytes
		metrics.NetworkIO.RxPackets += netStats.RxPackets
		metrics.NetworkIO.TxPackets += netStats.TxPackets
		metrics.NetworkIO.RxErrors += netStats.RxErrors
		metrics.NetworkIO.TxErrors += netStats.TxErrors
		metrics.NetworkIO.RxDropped += netStats.RxDropped
		metrics.NetworkIO.TxDropped += netStats.TxDropped
	}

	// 디스크 I/O 메트릭
	for _, ioStat := range stats.BlkioStats.IoServiceBytesRecursive {
		if ioStat.Op == "Read" {
			metrics.DiskIO.ReadBytes += ioStat.Value
		} else if ioStat.Op == "Write" {
			metrics.DiskIO.WriteBytes += ioStat.Value
		}
	}
	
	for _, ioStat := range stats.BlkioStats.IoServicedRecursive {
		if ioStat.Op == "Read" {
			metrics.DiskIO.ReadOperations += ioStat.Value
		} else if ioStat.Op == "Write" {
			metrics.DiskIO.WriteOperations += ioStat.Value
		}
	}

	// PID 메트릭
	if stats.PidsStats.Current > 0 {
		metrics.PIDs.Current = stats.PidsStats.Current
		metrics.PIDs.Limit = stats.PidsStats.Limit
	}

	// 리소스 제한
	inspect, err := arm.client.cli.ContainerInspect(arm.ctx, containerID)
	if err == nil {
		metrics.ResourceLimits.CPUQuota = inspect.HostConfig.CPUQuota
		metrics.ResourceLimits.CPUPeriod = uint64(inspect.HostConfig.CPUPeriod)
		metrics.ResourceLimits.MemoryLimit = inspect.HostConfig.Memory
		metrics.ResourceLimits.SwapLimit = inspect.HostConfig.MemorySwap
		metrics.ResourceLimits.PidsLimit = *inspect.HostConfig.PidsLimit
		metrics.ResourceLimits.CPUShares = uint64(inspect.HostConfig.CPUShares)
	}

	return metrics
}

// calculateRates 비율 계산 (초당 변화량)
func (arm *AdvancedResourceMonitor) calculateRates(current, previous *ResourceMetrics) {
	timeDelta := current.Timestamp.Sub(previous.Timestamp).Seconds()
	if timeDelta <= 0 {
		return
	}

	// 네트워크 비율
	current.NetworkIO.RxRate = float64(current.NetworkIO.RxBytes-previous.NetworkIO.RxBytes) / timeDelta
	current.NetworkIO.TxRate = float64(current.NetworkIO.TxBytes-previous.NetworkIO.TxBytes) / timeDelta

	// 디스크 I/O 비율
	current.DiskIO.ReadRate = float64(current.DiskIO.ReadBytes-previous.DiskIO.ReadBytes) / timeDelta
	current.DiskIO.WriteRate = float64(current.DiskIO.WriteBytes-previous.DiskIO.WriteBytes) / timeDelta
}

// calculateUtilization 사용률 계산
func (arm *AdvancedResourceMonitor) calculateUtilization(metrics *ResourceMetrics) {
	// CPU 사용률 (이미 계산됨)
	metrics.Utilization.CPUUtilization = metrics.CPUUsage.UsagePercent

	// 메모리 사용률
	metrics.Utilization.MemoryUtilization = metrics.MemoryUsage.UsagePercent

	// 네트워크 사용률 (1Gbps 기준으로 정규화)
	maxNetworkBandwidth := float64(1024 * 1024 * 1024) // 1GB/s
	networkTotal := metrics.NetworkIO.RxRate + metrics.NetworkIO.TxRate
	metrics.Utilization.NetworkUtilization = math.Min((networkTotal/maxNetworkBandwidth)*100, 100)

	// 디스크 사용률 (100MB/s 기준으로 정규화)
	maxDiskBandwidth := float64(100 * 1024 * 1024) // 100MB/s
	diskTotal := metrics.DiskIO.ReadRate + metrics.DiskIO.WriteRate
	metrics.Utilization.DiskUtilization = math.Min((diskTotal/maxDiskBandwidth)*100, 100)

	// 전체 점수 (가중 평균)
	cpuWeight := 0.4
	memoryWeight := 0.4
	networkWeight := 0.1
	diskWeight := 0.1

	metrics.Utilization.OverallScore = (metrics.Utilization.CPUUtilization * cpuWeight) +
		(metrics.Utilization.MemoryUtilization * memoryWeight) +
		(metrics.Utilization.NetworkUtilization * networkWeight) +
		(metrics.Utilization.DiskUtilization * diskWeight)
}

// checkAlerts 알람 확인
func (arm *AdvancedResourceMonitor) checkAlerts(agentID string, metrics *ResourceMetrics) {
	arm.mu.RLock()
	state, exists := arm.containerStates[agentID]
	if !exists {
		arm.mu.RUnlock()
		return
	}
	threshold := state.Config.Threshold
	alertHandlers := make([]AlertHandler, len(arm.alertHandlers))
	copy(alertHandlers, arm.alertHandlers)
	arm.mu.RUnlock()

	alerts := make([]ResourceAlert, 0)

	// CPU 알람
	if metrics.CPUUsage.UsagePercent >= threshold.CPUCritical {
		alerts = append(alerts, ResourceAlert{
			AgentID:     agentID,
			ContainerID: metrics.ContainerID,
			Timestamp:   time.Now(),
			Level:       AlertLevelCritical,
			Resource:    "cpu",
			Value:       metrics.CPUUsage.UsagePercent,
			Threshold:   threshold.CPUCritical,
			Message:     fmt.Sprintf("CPU usage critical: %.2f%%", metrics.CPUUsage.UsagePercent),
		})
	} else if metrics.CPUUsage.UsagePercent >= threshold.CPUWarning {
		alerts = append(alerts, ResourceAlert{
			AgentID:     agentID,
			ContainerID: metrics.ContainerID,
			Timestamp:   time.Now(),
			Level:       AlertLevelWarning,
			Resource:    "cpu",
			Value:       metrics.CPUUsage.UsagePercent,
			Threshold:   threshold.CPUWarning,
			Message:     fmt.Sprintf("CPU usage high: %.2f%%", metrics.CPUUsage.UsagePercent),
		})
	}

	// 메모리 알람
	if metrics.MemoryUsage.UsagePercent >= threshold.MemoryCritical {
		alerts = append(alerts, ResourceAlert{
			AgentID:     agentID,
			ContainerID: metrics.ContainerID,
			Timestamp:   time.Now(),
			Level:       AlertLevelCritical,
			Resource:    "memory",
			Value:       metrics.MemoryUsage.UsagePercent,
			Threshold:   threshold.MemoryCritical,
			Message:     fmt.Sprintf("Memory usage critical: %.2f%%", metrics.MemoryUsage.UsagePercent),
		})
	} else if metrics.MemoryUsage.UsagePercent >= threshold.MemoryWarning {
		alerts = append(alerts, ResourceAlert{
			AgentID:     agentID,
			ContainerID: metrics.ContainerID,
			Timestamp:   time.Now(),
			Level:       AlertLevelWarning,
			Resource:    "memory",
			Value:       metrics.MemoryUsage.UsagePercent,
			Threshold:   threshold.MemoryWarning,
			Message:     fmt.Sprintf("Memory usage high: %.2f%%", metrics.MemoryUsage.UsagePercent),
		})
	}

	// 네트워크 알람
	networkRate := metrics.NetworkIO.RxRate + metrics.NetworkIO.TxRate
	if networkRate >= threshold.NetworkCritical {
		alerts = append(alerts, ResourceAlert{
			AgentID:     agentID,
			ContainerID: metrics.ContainerID,
			Timestamp:   time.Now(),
			Level:       AlertLevelCritical,
			Resource:    "network",
			Value:       networkRate,
			Threshold:   threshold.NetworkCritical,
			Message:     fmt.Sprintf("Network usage critical: %.2f MB/s", networkRate/(1024*1024)),
		})
	} else if networkRate >= threshold.NetworkWarning {
		alerts = append(alerts, ResourceAlert{
			AgentID:     agentID,
			ContainerID: metrics.ContainerID,
			Timestamp:   time.Now(),
			Level:       AlertLevelWarning,
			Resource:    "network",
			Value:       networkRate,
			Threshold:   threshold.NetworkWarning,
			Message:     fmt.Sprintf("Network usage high: %.2f MB/s", networkRate/(1024*1024)),
		})
	}

	// 알람 저장 및 핸들러 실행
	if len(alerts) > 0 {
		arm.mu.Lock()
		if currentState, exists := arm.containerStates[agentID]; exists {
			currentState.AlertsGenerated = append(currentState.AlertsGenerated, alerts...)
		}
		arm.mu.Unlock()

		// 핸들러 실행
		for _, alert := range alerts {
			for _, handler := range alertHandlers {
				go handler(alert)
			}
		}
	}
}

// checkAutoScale 자동 스케일링 확인
func (arm *AdvancedResourceMonitor) checkAutoScale(agentID string, metrics *ResourceMetrics) {
	// CPU 사용률이 80% 이상이면 CPU 할당량 증가
	if metrics.CPUUsage.UsagePercent >= 80.0 && metrics.ResourceLimits.CPUQuota > 0 {
		newQuota := int64(float64(metrics.ResourceLimits.CPUQuota) * 1.2)
		arm.updateCPULimit(metrics.ContainerID, newQuota)
	}

	// 메모리 사용률이 85% 이상이면 메모리 한계 증가
	if metrics.MemoryUsage.UsagePercent >= 85.0 && metrics.ResourceLimits.MemoryLimit > 0 {
		newLimit := int64(float64(metrics.ResourceLimits.MemoryLimit) * 1.1)
		arm.updateMemoryLimit(metrics.ContainerID, newLimit)
	}
}

// updateCPULimit CPU 제한 업데이트
func (arm *AdvancedResourceMonitor) updateCPULimit(containerID string, newQuota int64) error {
	ctx, cancel := context.WithTimeout(arm.ctx, 10*time.Second)
	defer cancel()

	updateConfig := container.UpdateConfig{
		Resources: container.Resources{
			CPUQuota: newQuota,
		},
	}

	_, err := arm.client.cli.ContainerUpdate(ctx, containerID, updateConfig)
	return err
}

// updateMemoryLimit 메모리 제한 업데이트
func (arm *AdvancedResourceMonitor) updateMemoryLimit(containerID string, newLimit int64) error {
	ctx, cancel := context.WithTimeout(arm.ctx, 10*time.Second)
	defer cancel()

	updateConfig := container.UpdateConfig{
		Resources: container.Resources{
			Memory: newLimit,
		},
	}

	_, err := arm.client.cli.ContainerUpdate(ctx, containerID, updateConfig)
	return err
}

// GetMonitoringMetrics 모니터링 메트릭 조회
func (arm *AdvancedResourceMonitor) GetMonitoringMetrics() map[string]interface{} {
	arm.mu.RLock()
	defer arm.mu.RUnlock()

	totalContainers := len(arm.containerStates)
	monitoringContainers := 0
	totalAlerts := 0
	criticalAlerts := 0
	avgCollectionErrors := 0.0

	for _, state := range arm.containerStates {
		if state.IsMonitoring {
			monitoringContainers++
		}
		
		totalAlerts += len(state.AlertsGenerated)
		for _, alert := range state.AlertsGenerated {
			if alert.Level == AlertLevelCritical {
				criticalAlerts++
			}
		}
		
		avgCollectionErrors += float64(state.CollectionErrors)
	}

	if totalContainers > 0 {
		avgCollectionErrors /= float64(totalContainers)
	}

	return map[string]interface{}{
		"total_containers":       totalContainers,
		"monitoring_containers":  monitoringContainers,
		"total_alerts":          totalAlerts,
		"critical_alerts":       criticalAlerts,
		"avg_collection_errors": avgCollectionErrors,
		"monitoring_interval":   arm.globalConfig.Interval.Seconds(),
	}
}