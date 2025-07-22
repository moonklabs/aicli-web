package docker

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MetricsCollector 메트릭 수집기
type MetricsCollector struct {
	// 의존성
	statsCollector StatsCollection
	dockerManager  DockerManager

	// 집계 데이터
	aggregatedData map[string]*AggregatedMetrics
	dataMutex      sync.RWMutex

	// 설정
	collectionInterval time.Duration
	retentionPeriod    time.Duration

	// 제어
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	logger Logger
}

// AggregatedMetrics 집계된 메트릭
type AggregatedMetrics struct {
	// 기본 정보
	ContainerID  string    `json:"container_id"`
	WorkspaceID  string    `json:"workspace_id"`
	LastUpdated  time.Time `json:"last_updated"`
	DataPoints   int       `json:"data_points"`

	// CPU 메트릭
	CPUUsage    CPUMetrics `json:"cpu_usage"`

	// 메모리 메트릭
	MemoryUsage MemoryMetrics `json:"memory_usage"`

	// 네트워크 메트릭
	NetworkIO   NetworkMetrics `json:"network_io"`

	// 디스크 메트릭 (향후 확장)
	DiskIO      DiskMetrics `json:"disk_io"`

	// 타이밍 정보
	Uptime      time.Duration `json:"uptime"`
	ActiveTime  time.Duration `json:"active_time"`
}

// CPUMetrics CPU 관련 메트릭
type CPUMetrics struct {
	Current float64 `json:"current"`
	Average float64 `json:"average"`
	Peak    float64 `json:"peak"`
	Min     float64 `json:"min"`
	
	// 히스토리
	History []DataPoint `json:"history,omitempty"`
}

// MemoryMetrics 메모리 관련 메트릭
type MemoryMetrics struct {
	CurrentUsage int64   `json:"current_usage"`
	AverageUsage int64   `json:"average_usage"`
	PeakUsage    int64   `json:"peak_usage"`
	Limit        int64   `json:"limit"`
	UsagePercent float64 `json:"usage_percent"`
	
	// 히스토리
	History []DataPoint `json:"history,omitempty"`
}

// NetworkMetrics 네트워크 관련 메트릭
type NetworkMetrics struct {
	RxBytes       float64 `json:"rx_bytes"`
	TxBytes       float64 `json:"tx_bytes"`
	RxMB          float64 `json:"rx_mb"`
	TxMB          float64 `json:"tx_mb"`
	TotalMB       float64 `json:"total_mb"`
	
	// 속도 (MB/s)
	RxRate        float64 `json:"rx_rate"`
	TxRate        float64 `json:"tx_rate"`
	
	// 히스토리
	History []DataPoint `json:"history,omitempty"`
}

// DiskMetrics 디스크 관련 메트릭 (향후 구현)
type DiskMetrics struct {
	ReadBytes  int64   `json:"read_bytes"`
	WriteBytes int64   `json:"write_bytes"`
	ReadRate   float64 `json:"read_rate"`
	WriteRate  float64 `json:"write_rate"`
	
	// 히스토리
	History []DataPoint `json:"history,omitempty"`
}

// DataPoint 데이터 포인트
type DataPoint struct {
	Timestamp time.Time   `json:"timestamp"`
	Value     interface{} `json:"value"`
}

// Logger 로거 인터페이스 (상태 패키지와 동일)
type Logger interface {
	Info(msg string, args ...interface{})
	Error(msg string, err error, args ...interface{})
	Debug(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
}

// defaultMetricsLogger 기본 메트릭 로거
type defaultMetricsLogger struct{}

func (l *defaultMetricsLogger) Info(msg string, args ...interface{}) {
	fmt.Printf("[METRICS-INFO] "+msg+"\n", args...)
}

func (l *defaultMetricsLogger) Error(msg string, err error, args ...interface{}) {
	fmt.Printf("[METRICS-ERROR] "+msg+": %v\n", append(args, err)...)
}

func (l *defaultMetricsLogger) Debug(msg string, args ...interface{}) {
	fmt.Printf("[METRICS-DEBUG] "+msg+"\n", args...)
}

func (l *defaultMetricsLogger) Warn(msg string, args ...interface{}) {
	fmt.Printf("[METRICS-WARN] "+msg+"\n", args...)
}

// NewMetricsCollector 새로운 메트릭 수집기 생성
func NewMetricsCollector(statsCollector StatsCollection, dockerManager DockerManager) *MetricsCollector {
	ctx, cancel := context.WithCancel(context.Background())

	return &MetricsCollector{
		statsCollector:     statsCollector,
		dockerManager:      dockerManager,
		aggregatedData:     make(map[string]*AggregatedMetrics),
		collectionInterval: 30 * time.Second,
		retentionPeriod:    24 * time.Hour,
		ctx:                ctx,
		cancel:             cancel,
		logger:             &defaultMetricsLogger{},
	}
}

// SetLogger 로거 설정
func (mc *MetricsCollector) SetLogger(logger Logger) {
	mc.logger = logger
}

// Start 메트릭 수집 시작
func (mc *MetricsCollector) Start() error {
	mc.logger.Info("메트릭 수집기 시작 - 수집 간격: %v", mc.collectionInterval)
	
	mc.wg.Add(2)
	go mc.collectLoop()
	go mc.cleanupLoop()
	
	return nil
}

// Stop 메트릭 수집 중지
func (mc *MetricsCollector) Stop() error {
	mc.logger.Info("메트릭 수집기 중지 중...")
	
	mc.cancel()
	mc.wg.Wait()
	
	mc.logger.Info("메트릭 수집기 중지 완료")
	return nil
}

// collectLoop 메트릭 수집 루프
func (mc *MetricsCollector) collectLoop() {
	defer mc.wg.Done()

	ticker := time.NewTicker(mc.collectionInterval)
	defer ticker.Stop()

	mc.logger.Debug("메트릭 수집 루프 시작")

	for {
		select {
		case <-mc.ctx.Done():
			return
		case <-ticker.C:
			mc.collectAllMetrics()
		}
	}
}

// collectAllMetrics 모든 메트릭 수집
func (mc *MetricsCollector) collectAllMetrics() {
	mc.logger.Debug("전체 메트릭 수집 시작")

	// 모든 컨테이너 통계 수집
	allStats, err := mc.statsCollector.CollectAll(mc.ctx)
	if err != nil {
		mc.logger.Error("컨테이너 통계 수집 실패", err)
		return
	}

	// 각 컨테이너별 메트릭 처리
	for containerID, stats := range allStats {
		mc.processContainerMetrics(containerID, stats)
	}

	mc.logger.Debug("전체 메트릭 수집 완료: %d개 컨테이너", len(allStats))
}

// processContainerMetrics 컨테이너 메트릭 처리
func (mc *MetricsCollector) processContainerMetrics(containerID string, stats *ContainerStats) {
	if stats == nil {
		return
	}

	mc.dataMutex.Lock()
	defer mc.dataMutex.Unlock()

	// 기존 메트릭 조회
	existing, exists := mc.aggregatedData[containerID]
	if !exists {
		existing = &AggregatedMetrics{
			ContainerID: containerID,
			WorkspaceID: mc.extractWorkspaceID(containerID),
			LastUpdated: time.Now(),
			DataPoints:  0,
			CPUUsage: CPUMetrics{
				Min: 100.0, // 초기값을 높게 설정
			},
			MemoryUsage: MemoryMetrics{
				Limit: stats.MemoryLimit,
			},
		}
		mc.aggregatedData[containerID] = existing
	}

	// 메트릭 업데이트
	mc.updateCPUMetrics(&existing.CPUUsage, stats.CPUPercent)
	mc.updateMemoryMetrics(&existing.MemoryUsage, stats.MemoryUsage, stats.MemoryLimit)
	mc.updateNetworkMetrics(&existing.NetworkIO, stats.NetworkRxMB, stats.NetworkTxMB, existing.LastUpdated)

	// 메타 정보 업데이트
	existing.LastUpdated = time.Now()
	existing.DataPoints++

	mc.logger.Debug("컨테이너 %s 메트릭 업데이트: CPU=%.2f%%, MEM=%dMB", 
		containerID[:12], stats.CPUPercent, stats.MemoryUsage/(1024*1024))
}

// updateCPUMetrics CPU 메트릭 업데이트
func (mc *MetricsCollector) updateCPUMetrics(cpu *CPUMetrics, current float64) {
	cpu.Current = current
	
	// 평균 계산 (이동 평균)
	if cpu.Average == 0 {
		cpu.Average = current
	} else {
		cpu.Average = (cpu.Average*0.9) + (current*0.1)
	}
	
	// 최대값 업데이트
	if current > cpu.Peak {
		cpu.Peak = current
	}
	
	// 최소값 업데이트
	if current < cpu.Min {
		cpu.Min = current
	}

	// 히스토리 저장 (최근 100개 포인트만 유지)
	if len(cpu.History) >= 100 {
		cpu.History = cpu.History[1:] // 가장 오래된 것 제거
	}
	cpu.History = append(cpu.History, DataPoint{
		Timestamp: time.Now(),
		Value:     current,
	})
}

// updateMemoryMetrics 메모리 메트릭 업데이트
func (mc *MetricsCollector) updateMemoryMetrics(mem *MemoryMetrics, currentUsage, limit int64) {
	mem.CurrentUsage = currentUsage
	mem.Limit = limit
	
	// 사용률 계산
	if limit > 0 {
		mem.UsagePercent = float64(currentUsage) / float64(limit) * 100
	}
	
	// 평균 계산
	if mem.AverageUsage == 0 {
		mem.AverageUsage = currentUsage
	} else {
		mem.AverageUsage = int64(float64(mem.AverageUsage)*0.9 + float64(currentUsage)*0.1)
	}
	
	// 최대값 업데이트
	if currentUsage > mem.PeakUsage {
		mem.PeakUsage = currentUsage
	}

	// 히스토리 저장
	if len(mem.History) >= 100 {
		mem.History = mem.History[1:]
	}
	mem.History = append(mem.History, DataPoint{
		Timestamp: time.Now(),
		Value:     currentUsage,
	})
}

// updateNetworkMetrics 네트워크 메트릭 업데이트
func (mc *MetricsCollector) updateNetworkMetrics(net *NetworkMetrics, rxMB, txMB float64, lastUpdate time.Time) {
	// 이전 값 저장
	prevRxMB := net.RxMB
	prevTxMB := net.TxMB
	
	// 현재 값 업데이트
	net.RxMB = rxMB
	net.TxMB = txMB
	net.TotalMB = rxMB + txMB
	
	// 바이트 단위 계산
	net.RxBytes = rxMB * 1024 * 1024
	net.TxBytes = txMB * 1024 * 1024
	
	// 전송 속도 계산 (MB/s)
	if !lastUpdate.IsZero() {
		timeDiff := time.Since(lastUpdate).Seconds()
		if timeDiff > 0 {
			net.RxRate = (rxMB - prevRxMB) / timeDiff
			net.TxRate = (txMB - prevTxMB) / timeDiff
		}
	}

	// 히스토리 저장
	if len(net.History) >= 100 {
		net.History = net.History[1:]
	}
	net.History = append(net.History, DataPoint{
		Timestamp: time.Now(),
		Value: map[string]float64{
			"rx_mb": rxMB,
			"tx_mb": txMB,
			"rx_rate": net.RxRate,
			"tx_rate": net.TxRate,
		},
	})
}

// extractWorkspaceID 컨테이너 ID에서 워크스페이스 ID 추출
func (mc *MetricsCollector) extractWorkspaceID(containerID string) string {
	// Docker 라벨이나 이름에서 워크스페이스 ID 추출
	// 실제 구현에서는 Docker inspect를 통해 라벨을 확인해야 함
	// 현재는 간단한 매칭으로 구현
	
	// 컨테이너 정보 조회 (향후 구현)
	return "unknown"
}

// GetContainerMetrics 특정 컨테이너 메트릭 조회
func (mc *MetricsCollector) GetContainerMetrics(containerID string) (*AggregatedMetrics, bool) {
	mc.dataMutex.RLock()
	defer mc.dataMutex.RUnlock()

	metrics, exists := mc.aggregatedData[containerID]
	return metrics, exists
}

// GetWorkspaceMetrics 워크스페이스 메트릭 조회
func (mc *MetricsCollector) GetWorkspaceMetrics(workspaceID string) []*AggregatedMetrics {
	mc.dataMutex.RLock()
	defer mc.dataMutex.RUnlock()

	var result []*AggregatedMetrics
	for _, metrics := range mc.aggregatedData {
		if metrics.WorkspaceID == workspaceID {
			result = append(result, metrics)
		}
	}

	return result
}

// GetAllMetrics 모든 메트릭 조회
func (mc *MetricsCollector) GetAllMetrics() map[string]*AggregatedMetrics {
	mc.dataMutex.RLock()
	defer mc.dataMutex.RUnlock()

	// 복사본 생성
	result := make(map[string]*AggregatedMetrics)
	for k, v := range mc.aggregatedData {
		result[k] = v
	}

	return result
}

// cleanupLoop 정리 루프
func (mc *MetricsCollector) cleanupLoop() {
	defer mc.wg.Done()

	ticker := time.NewTicker(1 * time.Hour) // 1시간마다 정리
	defer ticker.Stop()

	for {
		select {
		case <-mc.ctx.Done():
			return
		case <-ticker.C:
			mc.cleanupOldMetrics()
		}
	}
}

// cleanupOldMetrics 오래된 메트릭 정리
func (mc *MetricsCollector) cleanupOldMetrics() {
	mc.dataMutex.Lock()
	defer mc.dataMutex.Unlock()

	now := time.Now()
	cleanedCount := 0

	for containerID, metrics := range mc.aggregatedData {
		if now.Sub(metrics.LastUpdated) > mc.retentionPeriod {
			delete(mc.aggregatedData, containerID)
			cleanedCount++
		}
	}

	if cleanedCount > 0 {
		mc.logger.Info("오래된 메트릭 정리 완료: %d개 컨테이너", cleanedCount)
	}
}

// GetStats 메트릭 수집기 통계 조회
func (mc *MetricsCollector) GetStats() MetricsStats {
	mc.dataMutex.RLock()
	defer mc.dataMutex.RUnlock()

	var totalDataPoints int
	for _, metrics := range mc.aggregatedData {
		totalDataPoints += metrics.DataPoints
	}

	return MetricsStats{
		TrackedContainers:  len(mc.aggregatedData),
		TotalDataPoints:    totalDataPoints,
		CollectionInterval: mc.collectionInterval,
		RetentionPeriod:    mc.retentionPeriod,
		LastCollection:     time.Now(), // 실제로는 마지막 수집 시간을 저장해야 함
	}
}

// MetricsStats 메트릭 수집기 통계
type MetricsStats struct {
	TrackedContainers  int           `json:"tracked_containers"`
	TotalDataPoints    int           `json:"total_data_points"`
	CollectionInterval time.Duration `json:"collection_interval"`
	RetentionPeriod    time.Duration `json:"retention_period"`
	LastCollection     time.Time     `json:"last_collection"`
}