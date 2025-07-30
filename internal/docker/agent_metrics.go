package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
)

// ContainerStatsData 컨테이너 통계 데이터 (Docker API 호환)
type ContainerStatsData struct {
	CPUStats    CPUStatsData    `json:"cpu_stats"`
	PreCPUStats CPUStatsData    `json:"precpu_stats"`
	MemoryStats MemoryStatsData `json:"memory_stats"`
	BlkioStats  BlkioStatsData  `json:"blkio_stats"`
	Networks    map[string]NetworkStatsData `json:"networks"`
	PidsStats   PidsStatsData   `json:"pids_stats"`
}

// CPUStatsData CPU 통계 데이터
type CPUStatsData struct {
	CPUUsage      CPUUsageData `json:"cpu_usage"`
	SystemUsage   uint64       `json:"system_cpu_usage"`
	OnlineCPUs    uint32       `json:"online_cpus"`
	ThrottlingData ThrottlingData `json:"throttling_data"`
}

// CPUUsageData CPU 사용량 데이터
type CPUUsageData struct {
	TotalUsage        uint64   `json:"total_usage"`
	UsageInUsermode   uint64   `json:"usage_in_usermode"`
	UsageInKernelmode uint64   `json:"usage_in_kernelmode"`
	PercpuUsage       []uint64 `json:"percpu_usage"`
}

// ThrottlingData CPU 스로틀링 데이터
type ThrottlingData struct {
	Periods          uint64 `json:"periods"`
	ThrottledPeriods uint64 `json:"throttled_periods"`
	ThrottledTime    uint64 `json:"throttled_time"`
}

// MemoryStatsData 메모리 통계 데이터
type MemoryStatsData struct {
	Usage    uint64            `json:"usage"`
	MaxUsage uint64            `json:"max_usage"`
	Limit    uint64            `json:"limit"`
	Stats    map[string]uint64 `json:"stats"`
}

// NetworkStatsData 네트워크 통계 데이터
type NetworkStatsData struct {
	RxBytes   uint64 `json:"rx_bytes"`
	RxPackets uint64 `json:"rx_packets"`
	RxErrors  uint64 `json:"rx_errors"`
	RxDropped uint64 `json:"rx_dropped"`
	TxBytes   uint64 `json:"tx_bytes"`
	TxPackets uint64 `json:"tx_packets"`
	TxErrors  uint64 `json:"tx_errors"`
	TxDropped uint64 `json:"tx_dropped"`
}

// BlkioStatsData 블록 I/O 통계 데이터
type BlkioStatsData struct {
	IoServiceBytesRecursive []BlkioStatEntry `json:"io_service_bytes_recursive"`
	IoServicedRecursive     []BlkioStatEntry `json:"io_serviced_recursive"`
	IoTimeRecursive         []BlkioStatEntry `json:"io_time_recursive"`
	IoQueuedRecursive       []BlkioStatEntry `json:"io_queued_recursive"`
}

// BlkioStatEntry 블록 I/O 통계 항목
type BlkioStatEntry struct {
	Major uint64 `json:"major"`
	Minor uint64 `json:"minor"`
	Op    string `json:"op"`
	Value uint64 `json:"value"`
}

// PidsStatsData 프로세스 통계 데이터
type PidsStatsData struct {
	Current uint64 `json:"current"`
	Limit   uint64 `json:"limit"`
}

// AgentMetricsCollector 에이전트 메트릭 수집기
type AgentMetricsCollector struct {
	client        *Client
	mu            sync.RWMutex
	agentMetrics  map[string]*AgentMetrics
	running       bool
	cancel        context.CancelFunc
	collectTicker *time.Ticker
}

// AgentMetrics 에이전트 메트릭 정보
type AgentMetrics struct {
	AgentID      string                 `json:"agent_id"`
	ContainerID  string                 `json:"container_id"`
	
	// 현재 메트릭
	Current      AgentMetricSnapshot         `json:"current"`
	
	// 히스토리 (최근 100개 보관)
	History      []AgentMetricSnapshot       `json:"history"`
	
	// 집계 통계
	Statistics   AgentMetricStatistics       `json:"statistics"`
	
	// 알림 임계값
	Thresholds   AgentMetricThresholds       `json:"thresholds"`
	
	// 메타데이터
	FirstCollected time.Time            `json:"first_collected"`
	LastCollected  time.Time            `json:"last_collected"`
	CollectCount   int64                `json:"collect_count"`
	
	mu           sync.RWMutex
}

// AgentMetricSnapshot 특정 시점의 메트릭 스냅샷
type AgentMetricSnapshot struct {
	Timestamp    time.Time                  `json:"timestamp"`
	
	// CPU 메트릭
	CPU          AgentCPUMetrics            `json:"cpu"`
	
	// 메모리 메트릭
	Memory       AgentMemoryMetrics         `json:"memory"`
	
	// 네트워크 메트릭
	Network      AgentNetworkMetrics        `json:"network"`
	
	// 디스크 I/O 메트릭
	DiskIO       AgentDiskIOMetrics         `json:"disk_io"`
	
	// 프로세스 메트릭
	Process      AgentProcessMetrics        `json:"process"`
	
	// 컨테이너 상태
	Container    AgentContainerMetrics      `json:"container"`
}

// AgentCPUMetrics 에이전트 CPU 관련 메트릭
type AgentCPUMetrics struct {
	UsagePercent     float64           `json:"usage_percent"`
	UsageCores       float64           `json:"usage_cores"`
	UserModeTime     uint64            `json:"user_mode_time"`
	KernelModeTime   uint64            `json:"kernel_mode_time"`
	ThrottledPeriods uint64            `json:"throttled_periods"`
	ThrottledTime    uint64            `json:"throttled_time"`
	PerCPUUsage      []uint64          `json:"per_cpu_usage"`
}

// AgentMemoryMetrics 메모리 관련 메트릭
type AgentMemoryMetrics struct {
	Usage            uint64   `json:"usage"`
	MaxUsage         uint64   `json:"max_usage"`
	Limit            uint64   `json:"limit"`
	UsagePercent     float64  `json:"usage_percent"`
	Cache            uint64   `json:"cache"`
	RSS              uint64   `json:"rss"`
	Swap             uint64   `json:"swap"`
	KernelMemory     uint64   `json:"kernel_memory"`
	MemoryFailCount  uint64   `json:"memory_fail_count"`
}

// AgentNetworkMetrics 네트워크 관련 메트릭
type AgentNetworkMetrics struct {
	RxBytes         uint64   `json:"rx_bytes"`
	RxPackets       uint64   `json:"rx_packets"`
	RxErrors        uint64   `json:"rx_errors"`
	RxDropped       uint64   `json:"rx_dropped"`
	TxBytes         uint64   `json:"tx_bytes"`
	TxPackets       uint64   `json:"tx_packets"`
	TxErrors        uint64   `json:"tx_errors"`
	TxDropped       uint64   `json:"tx_dropped"`
	RxBytesPerSec   float64  `json:"rx_bytes_per_sec"`
	TxBytesPerSec   float64  `json:"tx_bytes_per_sec"`
}

// AgentDiskIOMetrics 디스크 I/O 관련 메트릭
type AgentDiskIOMetrics struct {
	ReadBytes       uint64   `json:"read_bytes"`
	WriteBytes      uint64   `json:"write_bytes"`
	ReadIOPS        uint64   `json:"read_iops"`
	WriteIOPS       uint64   `json:"write_iops"`
	ReadBytesPerSec float64  `json:"read_bytes_per_sec"`
	WriteBytesPerSec float64 `json:"write_bytes_per_sec"`
	TotalIOTime     uint64   `json:"total_io_time"`
	QueuedIOTime    uint64   `json:"queued_io_time"`
}

// AgentProcessMetrics 프로세스 관련 메트릭
type AgentProcessMetrics struct {
	PIDCount        uint64   `json:"pid_count"`
	PIDLimit        uint64   `json:"pid_limit"`
	ThreadCount     uint64   `json:"thread_count"`
	FDCount         uint64   `json:"fd_count"`
	FDLimit         uint64   `json:"fd_limit"`
}

// AgentContainerMetrics 컨테이너 관련 메트릭
type AgentContainerMetrics struct {
	Status          string    `json:"status"`
	Running         bool      `json:"running"`
	Uptime          time.Duration `json:"uptime"`
	RestartCount    int       `json:"restart_count"`
	OOMKillCount    int       `json:"oom_kill_count"`
}

// AgentMetricStatistics 메트릭 통계
type AgentMetricStatistics struct {
	// CPU 통계
	CPUUsageAvg      float64   `json:"cpu_usage_avg"`
	CPUUsageMax      float64   `json:"cpu_usage_max"`
	CPUUsageMin      float64   `json:"cpu_usage_min"`
	
	// 메모리 통계
	MemoryUsageAvg   float64   `json:"memory_usage_avg"`
	MemoryUsageMax   uint64    `json:"memory_usage_max"`
	MemoryUsageMin   uint64    `json:"memory_usage_min"`
	
	// 네트워크 통계
	NetworkRxTotal   uint64    `json:"network_rx_total"`
	NetworkTxTotal   uint64    `json:"network_tx_total"`
	NetworkRxPeak    float64   `json:"network_rx_peak"`
	NetworkTxPeak    float64   `json:"network_tx_peak"`
	
	// 디스크 I/O 통계
	DiskReadTotal    uint64    `json:"disk_read_total"`
	DiskWriteTotal   uint64    `json:"disk_write_total"`
	DiskReadPeak     float64   `json:"disk_read_peak"`
	DiskWritePeak    float64   `json:"disk_write_peak"`
	
	// 기간
	StatsPeriod      time.Duration `json:"stats_period"`
	LastUpdated      time.Time     `json:"last_updated"`
}

// AgentMetricThresholds 메트릭 임계값
type AgentMetricThresholds struct {
	CPUWarning       float64   `json:"cpu_warning"`        // CPU 사용률 경고 임계값 (%)
	CPUCritical      float64   `json:"cpu_critical"`       // CPU 사용률 위험 임계값 (%)
	MemoryWarning    float64   `json:"memory_warning"`     // 메모리 사용률 경고 임계값 (%)
	MemoryCritical   float64   `json:"memory_critical"`    // 메모리 사용률 위험 임계값 (%)
	NetworkRxWarning uint64    `json:"network_rx_warning"` // 네트워크 수신 경고 임계값 (bytes/sec)
	NetworkTxWarning uint64    `json:"network_tx_warning"` // 네트워크 송신 경고 임계값 (bytes/sec)
	DiskReadWarning  uint64    `json:"disk_read_warning"`  // 디스크 읽기 경고 임계값 (bytes/sec)
	DiskWriteWarning uint64    `json:"disk_write_warning"` // 디스크 쓰기 경고 임계값 (bytes/sec)
}

// AgentMetricAlert 메트릭 알림
type AgentMetricAlert struct {
	AgentID     string          `json:"agent_id"`
	MetricType  string          `json:"metric_type"`
	Threshold   string          `json:"threshold"`    // warning, critical
	Value       float64         `json:"value"`
	Limit       float64         `json:"limit"`
	Message     string          `json:"message"`
	Timestamp   time.Time       `json:"timestamp"`
	Severity    AlertSeverity   `json:"severity"`
}

// AgentMetricAlertHandler 메트릭 알림 핸들러
type AgentMetricAlertHandler func(alert AgentMetricAlert)

// AgentMetricsConfig 메트릭 수집 설정
type AgentMetricsConfig struct {
	CollectInterval    time.Duration `json:"collect_interval"`     // 수집 간격 (기본: 30초)
	HistoryRetention   int           `json:"history_retention"`    // 히스토리 보관 개수 (기본: 100)
	EnableAlerts       bool          `json:"enable_alerts"`        // 알림 활성화
	DefaultThresholds  AgentMetricThresholds `json:"default_thresholds"` // 기본 임계값
}

// NewAgentMetricsCollector 새로운 에이전트 메트릭 수집기 생성
func NewAgentMetricsCollector(client *Client) *AgentMetricsCollector {
	return &AgentMetricsCollector{
		client:       client,
		agentMetrics: make(map[string]*AgentMetrics),
	}
}

// Start 메트릭 수집 시작
func (amc *AgentMetricsCollector) Start(ctx context.Context, config AgentMetricsConfig) error {
	amc.mu.Lock()
	defer amc.mu.Unlock()

	if amc.running {
		return fmt.Errorf("metrics collector already running")
	}

	// 기본값 설정
	if config.CollectInterval == 0 {
		config.CollectInterval = 30 * time.Second
	}
	if config.HistoryRetention == 0 {
		config.HistoryRetention = 100
	}

	// 기본 임계값 설정
	if config.DefaultThresholds.CPUWarning == 0 {
		config.DefaultThresholds = AgentMetricThresholds{
			CPUWarning:       80.0,
			CPUCritical:      95.0,
			MemoryWarning:    80.0,
			MemoryCritical:   95.0,
			NetworkRxWarning: 100 * 1024 * 1024, // 100MB/s
			NetworkTxWarning: 100 * 1024 * 1024, // 100MB/s
			DiskReadWarning:  50 * 1024 * 1024,  // 50MB/s
			DiskWriteWarning: 50 * 1024 * 1024,  // 50MB/s
		}
	}

	collectCtx, cancel := context.WithCancel(ctx)
	amc.cancel = cancel
	amc.running = true

	// 주기적 메트릭 수집 시작
	amc.collectTicker = time.NewTicker(config.CollectInterval)
	go amc.runMetricCollection(collectCtx, config)

	return nil
}

// Stop 메트릭 수집 중지
func (amc *AgentMetricsCollector) Stop() error {
	amc.mu.Lock()
	defer amc.mu.Unlock()

	if !amc.running {
		return nil
	}

	if amc.cancel != nil {
		amc.cancel()
	}

	if amc.collectTicker != nil {
		amc.collectTicker.Stop()
	}

	amc.running = false
	return nil
}

// RegisterAgent 에이전트 메트릭 수집 등록
func (amc *AgentMetricsCollector) RegisterAgent(agentID, containerID string, thresholds *AgentMetricThresholds) error {
	amc.mu.Lock()
	defer amc.mu.Unlock()

	if _, exists := amc.agentMetrics[agentID]; exists {
		return fmt.Errorf("agent metrics already registered: %s", agentID)
	}

	now := time.Now()
	metrics := &AgentMetrics{
		AgentID:        agentID,
		ContainerID:    containerID,
		History:        make([]AgentMetricSnapshot, 0),
		FirstCollected: now,
		LastCollected:  time.Time{},
		CollectCount:   0,
	}

	// 임계값 설정
	if thresholds != nil {
		metrics.Thresholds = *thresholds
	} else {
		// 기본 임계값 사용 (Start 메서드에서 설정됨)
		metrics.Thresholds = AgentMetricThresholds{
			CPUWarning:       80.0,
			CPUCritical:      95.0,
			MemoryWarning:    80.0,
			MemoryCritical:   95.0,
			NetworkRxWarning: 100 * 1024 * 1024,
			NetworkTxWarning: 100 * 1024 * 1024,
			DiskReadWarning:  50 * 1024 * 1024,
			DiskWriteWarning: 50 * 1024 * 1024,
		}
	}

	amc.agentMetrics[agentID] = metrics

	return nil
}

// UnregisterAgent 에이전트 메트릭 수집 해제
func (amc *AgentMetricsCollector) UnregisterAgent(agentID string) {
	amc.mu.Lock()
	defer amc.mu.Unlock()

	delete(amc.agentMetrics, agentID)
}

// runMetricCollection 메트릭 수집 실행
func (amc *AgentMetricsCollector) runMetricCollection(ctx context.Context, config AgentMetricsConfig) {
	defer amc.collectTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-amc.collectTicker.C:
			amc.collectAllMetrics(ctx, config)
		}
	}
}

// collectAllMetrics 모든 에이전트 메트릭 수집
func (amc *AgentMetricsCollector) collectAllMetrics(ctx context.Context, config AgentMetricsConfig) {
	amc.mu.RLock()
	agents := make([]string, 0, len(amc.agentMetrics))
	for agentID := range amc.agentMetrics {
		agents = append(agents, agentID)
	}
	amc.mu.RUnlock()

	for _, agentID := range agents {
		go amc.collectAgentMetrics(ctx, agentID, config)
	}
}

// collectAgentMetrics 단일 에이전트 메트릭 수집
func (amc *AgentMetricsCollector) collectAgentMetrics(ctx context.Context, agentID string, config AgentMetricsConfig) {
	amc.mu.RLock()
	metrics, exists := amc.agentMetrics[agentID]
	amc.mu.RUnlock()

	if !exists {
		return
	}

	metrics.mu.Lock()
	defer metrics.mu.Unlock()

	// 컨테이너 통계 수집
	statsReader, err := amc.client.cli.ContainerStats(ctx, metrics.ContainerID, false)
	if err != nil {
		// 에러 로깅 (실제 구현에서는 로거 사용)
		return
	}
	defer statsReader.Body.Close()

	var containerStats ContainerStatsData
	if err := json.NewDecoder(statsReader.Body).Decode(&containerStats); err != nil {
		return
	}

	// 컨테이너 정보 수집
	containerInfo, err := amc.client.cli.ContainerInspect(ctx, metrics.ContainerID)
	if err != nil {
		return
	}

	// 메트릭 스냅샷 생성
	snapshot := amc.createMetricSnapshot(containerStats, containerInfo)

	// 이전 스냅샷과 비교하여 속도 계산
	if len(metrics.History) > 0 {
		lastSnapshot := metrics.History[len(metrics.History)-1]
		amc.calculateRates(&snapshot, &lastSnapshot)
	}

	// 현재 메트릭 업데이트
	metrics.Current = snapshot
	metrics.LastCollected = snapshot.Timestamp
	metrics.CollectCount++

	// 히스토리에 추가
	metrics.History = append(metrics.History, snapshot)
	if len(metrics.History) > config.HistoryRetention {
		metrics.History = metrics.History[1:]
	}

	// 통계 업데이트
	amc.updateStatistics(metrics)

	// 알림 확인
	if config.EnableAlerts {
		amc.checkAlerts(agentID, &snapshot, &metrics.Thresholds)
	}
}

// createMetricSnapshot 메트릭 스냅샷 생성  
func (amc *AgentMetricsCollector) createMetricSnapshot(containerStats ContainerStatsData, containerInfo types.ContainerJSON) AgentMetricSnapshot {
	snapshot := AgentMetricSnapshot{
		Timestamp: time.Now(),
	}

	// CPU 메트릭
	snapshot.CPU = amc.extractCPUMetrics(containerStats.CPUStats, containerStats.PreCPUStats)

	// 메모리 메트릭
	snapshot.Memory = amc.extractMemoryMetrics(containerStats.MemoryStats)

	// 네트워크 메트릭
	snapshot.Network = amc.extractNetworkMetrics(containerStats.Networks)

	// 디스크 I/O 메트릭
	snapshot.DiskIO = amc.extractDiskIOMetrics(containerStats.BlkioStats)

	// 프로세스 메트릭
	snapshot.Process = amc.extractProcessMetrics(containerStats.PidsStats)

	// 컨테이너 메트릭
	snapshot.Container = amc.extractContainerMetrics(containerInfo)

	return snapshot
}

// extractCPUMetrics CPU 메트릭 추출
func (amc *AgentMetricsCollector) extractCPUMetrics(cpuStats, preCPUStats CPUStatsData) AgentCPUMetrics {
	var metrics AgentCPUMetrics

	cpuDelta := float64(cpuStats.CPUUsage.TotalUsage - preCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(cpuStats.SystemUsage - preCPUStats.SystemUsage)

	if systemDelta > 0 && cpuDelta > 0 {
		metrics.UsagePercent = (cpuDelta / systemDelta) * float64(cpuStats.OnlineCPUs) * 100.0
		metrics.UsageCores = (cpuDelta / systemDelta) * float64(cpuStats.OnlineCPUs)
	}

	metrics.UserModeTime = cpuStats.CPUUsage.UsageInUsermode
	metrics.KernelModeTime = cpuStats.CPUUsage.UsageInKernelmode
	metrics.ThrottledPeriods = cpuStats.ThrottlingData.ThrottledPeriods
	metrics.ThrottledTime = cpuStats.ThrottlingData.ThrottledTime
	metrics.PerCPUUsage = cpuStats.CPUUsage.PercpuUsage

	return metrics
}

// extractMemoryMetrics 메모리 메트릭 추출
func (amc *AgentMetricsCollector) extractMemoryMetrics(memStats MemoryStatsData) AgentMemoryMetrics {
	metrics := AgentMemoryMetrics{
		Usage:    memStats.Usage,
		MaxUsage: memStats.MaxUsage,
		Limit:    memStats.Limit,
	}

	if memStats.Limit > 0 {
		metrics.UsagePercent = float64(memStats.Usage) / float64(memStats.Limit) * 100.0
	}

	if memStats.Stats != nil {
		metrics.Cache = memStats.Stats["cache"]
		metrics.RSS = memStats.Stats["rss"]
		if swap, exists := memStats.Stats["swap"]; exists {
			metrics.Swap = swap
		}
		if kernelMemory, exists := memStats.Stats["kernel"]; exists {
			metrics.KernelMemory = kernelMemory
		}
		if failCount, exists := memStats.Stats["failcnt"]; exists {
			metrics.MemoryFailCount = failCount
		}
	}

	return metrics
}

// extractNetworkMetrics 네트워크 메트릭 추출
func (amc *AgentMetricsCollector) extractNetworkMetrics(networks map[string]NetworkStatsData) AgentNetworkMetrics {
	var metrics AgentNetworkMetrics

	for _, netStats := range networks {
		metrics.RxBytes += netStats.RxBytes
		metrics.RxPackets += netStats.RxPackets
		metrics.RxErrors += netStats.RxErrors
		metrics.RxDropped += netStats.RxDropped
		metrics.TxBytes += netStats.TxBytes
		metrics.TxPackets += netStats.TxPackets
		metrics.TxErrors += netStats.TxErrors
		metrics.TxDropped += netStats.TxDropped
	}

	return metrics
}

// extractDiskIOMetrics 디스크 I/O 메트릭 추출
func (amc *AgentMetricsCollector) extractDiskIOMetrics(blkioStats BlkioStatsData) AgentDiskIOMetrics {
	var metrics AgentDiskIOMetrics

	// 읽기/쓰기 바이트 합계 계산
	for _, stat := range blkioStats.IoServiceBytesRecursive {
		switch stat.Op {
		case "Read":
			metrics.ReadBytes += stat.Value
		case "Write":
			metrics.WriteBytes += stat.Value
		}
	}

	// 읽기/쓰기 IOPS 합계 계산
	for _, stat := range blkioStats.IoServicedRecursive {
		switch stat.Op {
		case "Read":
			metrics.ReadIOPS += stat.Value
		case "Write":
			metrics.WriteIOPS += stat.Value
		}
	}

	// I/O 대기 시간 (사용 가능한 경우)
	for _, stat := range blkioStats.IoTimeRecursive {
		metrics.TotalIOTime += stat.Value
	}

	for _, stat := range blkioStats.IoQueuedRecursive {
		metrics.QueuedIOTime += stat.Value
	}

	return metrics
}

// extractProcessMetrics 프로세스 메트릭 추출
func (amc *AgentMetricsCollector) extractProcessMetrics(pidsStats PidsStatsData) AgentProcessMetrics {
	return AgentProcessMetrics{
		PIDCount: pidsStats.Current,
		PIDLimit: pidsStats.Limit,
		// ThreadCount와 FDCount는 추가 구현 필요
	}
}

// extractContainerMetrics 컨테이너 메트릭 추출
func (amc *AgentMetricsCollector) extractContainerMetrics(containerInfo types.ContainerJSON) AgentContainerMetrics {
	metrics := AgentContainerMetrics{
		Status:  containerInfo.State.Status,
		Running: containerInfo.State.Running,
	}

	// TODO: 업타임 계산 로직 개선 필요
	if containerInfo.State.Running {
		// 현재는 0으로 설정
		metrics.Uptime = 0
	}

	// 재시작 횟수
	metrics.RestartCount = containerInfo.RestartCount

	// OOM 킬 여부
	if containerInfo.State.OOMKilled {
		metrics.OOMKillCount = 1
	}

	return metrics
}

// calculateRates 속도 관련 메트릭 계산
func (amc *AgentMetricsCollector) calculateRates(current, previous *AgentMetricSnapshot) {
	timeDiff := current.Timestamp.Sub(previous.Timestamp).Seconds()
	if timeDiff <= 0 {
		return
	}

	// 네트워크 속도 계산
	current.Network.RxBytesPerSec = float64(current.Network.RxBytes-previous.Network.RxBytes) / timeDiff
	current.Network.TxBytesPerSec = float64(current.Network.TxBytes-previous.Network.TxBytes) / timeDiff

	// 디스크 I/O 속도 계산
	current.DiskIO.ReadBytesPerSec = float64(current.DiskIO.ReadBytes-previous.DiskIO.ReadBytes) / timeDiff
	current.DiskIO.WriteBytesPerSec = float64(current.DiskIO.WriteBytes-previous.DiskIO.WriteBytes) / timeDiff
}

// updateStatistics 통계 업데이트
func (amc *AgentMetricsCollector) updateStatistics(metrics *AgentMetrics) {
	if len(metrics.History) == 0 {
		return
	}

	stats := &metrics.Statistics
	stats.LastUpdated = time.Now()
	stats.StatsPeriod = stats.LastUpdated.Sub(metrics.FirstCollected)

	// CPU 통계 계산
	var cpuSum, cpuMax, cpuMin float64 = 0, 0, math.MaxFloat64
	
	// 메모리 통계 계산
	var memSum float64 = 0
	var memMax, memMin uint64 = 0, math.MaxUint64

	// 네트워크 통계 계산
	var rxTotal, txTotal uint64 = 0, 0
	var rxPeak, txPeak float64 = 0, 0

	// 디스크 I/O 통계 계산
	var diskReadTotal, diskWriteTotal uint64 = 0, 0
	var diskReadPeak, diskWritePeak float64 = 0, 0

	for _, snapshot := range metrics.History {
		// CPU 통계
		cpuSum += snapshot.CPU.UsagePercent
		if snapshot.CPU.UsagePercent > cpuMax {
			cpuMax = snapshot.CPU.UsagePercent
		}
		if snapshot.CPU.UsagePercent < cpuMin {
			cpuMin = snapshot.CPU.UsagePercent
		}

		// 메모리 통계
		memSum += snapshot.Memory.UsagePercent
		if snapshot.Memory.Usage > memMax {
			memMax = snapshot.Memory.Usage
		}
		if snapshot.Memory.Usage < memMin {
			memMin = snapshot.Memory.Usage
		}

		// 네트워크 통계
		rxTotal += snapshot.Network.RxBytes
		txTotal += snapshot.Network.TxBytes
		if snapshot.Network.RxBytesPerSec > rxPeak {
			rxPeak = snapshot.Network.RxBytesPerSec
		}
		if snapshot.Network.TxBytesPerSec > txPeak {
			txPeak = snapshot.Network.TxBytesPerSec
		}

		// 디스크 I/O 통계
		diskReadTotal += snapshot.DiskIO.ReadBytes
		diskWriteTotal += snapshot.DiskIO.WriteBytes
		if snapshot.DiskIO.ReadBytesPerSec > diskReadPeak {
			diskReadPeak = snapshot.DiskIO.ReadBytesPerSec
		}
		if snapshot.DiskIO.WriteBytesPerSec > diskWritePeak {
			diskWritePeak = snapshot.DiskIO.WriteBytesPerSec
		}
	}

	count := float64(len(metrics.History))

	// 평균값 계산
	stats.CPUUsageAvg = cpuSum / count
	stats.CPUUsageMax = cpuMax
	stats.CPUUsageMin = cpuMin

	stats.MemoryUsageAvg = memSum / count
	stats.MemoryUsageMax = memMax
	stats.MemoryUsageMin = memMin

	stats.NetworkRxTotal = rxTotal
	stats.NetworkTxTotal = txTotal
	stats.NetworkRxPeak = rxPeak
	stats.NetworkTxPeak = txPeak

	stats.DiskReadTotal = diskReadTotal
	stats.DiskWriteTotal = diskWriteTotal
	stats.DiskReadPeak = diskReadPeak
	stats.DiskWritePeak = diskWritePeak
}

// checkAlerts 알림 임계값 확인
func (amc *AgentMetricsCollector) checkAlerts(agentID string, snapshot *AgentMetricSnapshot, thresholds *AgentMetricThresholds) {
	alerts := make([]AgentMetricAlert, 0)

	// CPU 사용률 확인
	if snapshot.CPU.UsagePercent >= thresholds.CPUCritical {
		alerts = append(alerts, AgentMetricAlert{
			AgentID:    agentID,
			MetricType: "cpu_usage",
			Threshold:  "critical",
			Value:      snapshot.CPU.UsagePercent,
			Limit:      thresholds.CPUCritical,
			Message:    fmt.Sprintf("CPU usage is critical: %.2f%%", snapshot.CPU.UsagePercent),
			Timestamp:  snapshot.Timestamp,
			Severity:   AlertSeverityCritical,
		})
	} else if snapshot.CPU.UsagePercent >= thresholds.CPUWarning {
		alerts = append(alerts, AgentMetricAlert{
			AgentID:    agentID,
			MetricType: "cpu_usage",
			Threshold:  "warning",
			Value:      snapshot.CPU.UsagePercent,
			Limit:      thresholds.CPUWarning,
			Message:    fmt.Sprintf("CPU usage is high: %.2f%%", snapshot.CPU.UsagePercent),
			Timestamp:  snapshot.Timestamp,
			Severity:   AlertSeverityWarning,
		})
	}

	// 메모리 사용률 확인
	if snapshot.Memory.UsagePercent >= thresholds.MemoryCritical {
		alerts = append(alerts, AgentMetricAlert{
			AgentID:    agentID,
			MetricType: "memory_usage",
			Threshold:  "critical",
			Value:      snapshot.Memory.UsagePercent,
			Limit:      thresholds.MemoryCritical,
			Message:    fmt.Sprintf("Memory usage is critical: %.2f%%", snapshot.Memory.UsagePercent),
			Timestamp:  snapshot.Timestamp,
			Severity:   AlertSeverityCritical,
		})
	} else if snapshot.Memory.UsagePercent >= thresholds.MemoryWarning {
		alerts = append(alerts, AgentMetricAlert{
			AgentID:    agentID,
			MetricType: "memory_usage",
			Threshold:  "warning",
			Value:      snapshot.Memory.UsagePercent,
			Limit:      thresholds.MemoryWarning,
			Message:    fmt.Sprintf("Memory usage is high: %.2f%%", snapshot.Memory.UsagePercent),
			Timestamp:  snapshot.Timestamp,
			Severity:   AlertSeverityWarning,
		})
	}

	// 네트워크 사용량 확인
	if snapshot.Network.RxBytesPerSec >= float64(thresholds.NetworkRxWarning) {
		alerts = append(alerts, AgentMetricAlert{
			AgentID:    agentID,
			MetricType: "network_rx",
			Threshold:  "warning",
			Value:      snapshot.Network.RxBytesPerSec,
			Limit:      float64(thresholds.NetworkRxWarning),
			Message:    fmt.Sprintf("Network RX rate is high: %.2f MB/s", snapshot.Network.RxBytesPerSec/1024/1024),
			Timestamp:  snapshot.Timestamp,
			Severity:   AlertSeverityWarning,
		})
	}

	if snapshot.Network.TxBytesPerSec >= float64(thresholds.NetworkTxWarning) {
		alerts = append(alerts, AgentMetricAlert{
			AgentID:    agentID,
			MetricType: "network_tx",
			Threshold:  "warning",
			Value:      snapshot.Network.TxBytesPerSec,
			Limit:      float64(thresholds.NetworkTxWarning),
			Message:    fmt.Sprintf("Network TX rate is high: %.2f MB/s", snapshot.Network.TxBytesPerSec/1024/1024),
			Timestamp:  snapshot.Timestamp,
			Severity:   AlertSeverityWarning,
		})
	}

	// 디스크 I/O 확인
	if snapshot.DiskIO.ReadBytesPerSec >= float64(thresholds.DiskReadWarning) {
		alerts = append(alerts, AgentMetricAlert{
			AgentID:    agentID,
			MetricType: "disk_read",
			Threshold:  "warning",
			Value:      snapshot.DiskIO.ReadBytesPerSec,
			Limit:      float64(thresholds.DiskReadWarning),
			Message:    fmt.Sprintf("Disk read rate is high: %.2f MB/s", snapshot.DiskIO.ReadBytesPerSec/1024/1024),
			Timestamp:  snapshot.Timestamp,
			Severity:   AlertSeverityWarning,
		})
	}

	if snapshot.DiskIO.WriteBytesPerSec >= float64(thresholds.DiskWriteWarning) {
		alerts = append(alerts, AgentMetricAlert{
			AgentID:    agentID,
			MetricType: "disk_write",
			Threshold:  "warning",
			Value:      snapshot.DiskIO.WriteBytesPerSec,
			Limit:      float64(thresholds.DiskWriteWarning),
			Message:    fmt.Sprintf("Disk write rate is high: %.2f MB/s", snapshot.DiskIO.WriteBytesPerSec/1024/1024),
			Timestamp:  snapshot.Timestamp,
			Severity:   AlertSeverityWarning,
		})
	}

	// 알림 발송
	for _, alert := range alerts {
		amc.sendMetricAlert(alert)
	}
}

// sendMetricAlert 메트릭 알림 발송
func (amc *AgentMetricsCollector) sendMetricAlert(alert AgentMetricAlert) {
	// TODO: 실제 알림 시스템과 연동
	fmt.Printf("[METRIC ALERT] %s: %s - %s\n", 
		alert.Severity, alert.AgentID, alert.Message)
}

// GetAgentMetrics 에이전트 메트릭 조회
func (amc *AgentMetricsCollector) GetAgentMetrics(agentID string) (*AgentMetrics, bool) {
	amc.mu.RLock()
	defer amc.mu.RUnlock()

	metrics, exists := amc.agentMetrics[agentID]
	if !exists {
		return nil, false
	}

	metrics.mu.RLock()
	defer metrics.mu.RUnlock()

	// 복사본 반환
	metricsCopy := *metrics
	metricsCopy.History = make([]AgentMetricSnapshot, len(metrics.History))
	copy(metricsCopy.History, metrics.History)

	return &metricsCopy, true
}

// GetAllAgentMetrics 모든 에이전트 메트릭 조회
func (amc *AgentMetricsCollector) GetAllAgentMetrics() map[string]*AgentMetrics {
	amc.mu.RLock()
	defer amc.mu.RUnlock()

	result := make(map[string]*AgentMetrics)
	for agentID, metrics := range amc.agentMetrics {
		metrics.mu.RLock()
		metricsCopy := *metrics
		metricsCopy.History = make([]AgentMetricSnapshot, len(metrics.History))
		copy(metricsCopy.History, metrics.History)
		metrics.mu.RUnlock()
		
		result[agentID] = &metricsCopy
	}

	return result
}

// UpdateThresholds 에이전트 알림 임계값 업데이트
func (amc *AgentMetricsCollector) UpdateThresholds(agentID string, thresholds AgentMetricThresholds) error {
	amc.mu.RLock()
	metrics, exists := amc.agentMetrics[agentID]
	amc.mu.RUnlock()

	if !exists {
		return fmt.Errorf("agent metrics not found: %s", agentID)
	}

	metrics.mu.Lock()
	defer metrics.mu.Unlock()

	metrics.Thresholds = thresholds
	return nil
}

// IsRunning 메트릭 수집기 실행 상태 확인
func (amc *AgentMetricsCollector) IsRunning() bool {
	amc.mu.RLock()
	defer amc.mu.RUnlock()
	return amc.running
}