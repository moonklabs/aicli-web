package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

// int64Ptr returns a pointer to an int64
func int64Ptr(i int64) *int64 {
	return &i
}

// ResourceMonitor monitors and manages container resources
type ResourceMonitor struct {
	client          *client.Client
	containerLimits map[string]*PTYResourceLimits
	history         map[string][]*ResourceSnapshot
	alerts          chan *PTYResourceAlert
	config          *ResourceMonitorConfig
	mutex           sync.RWMutex
	stopCh          chan struct{}
}

// ResourceMonitorConfig contains resource monitoring configuration
type ResourceMonitorConfig struct {
	HistorySize     int
	SamplingRate    time.Duration
	AlertThreshold  float64
	EnableAutoScale bool
	MaxHistoryAge   time.Duration
}

// ResourceSnapshot represents a point-in-time resource usage
type ResourceSnapshot struct {
	ContainerID   string
	CPUPercent    float64
	MemoryPercent float64
	MemoryUsage   int64
	NetworkRx     uint64
	NetworkTx     uint64
	DiskRead      uint64
	DiskWrite     uint64
	Timestamp     time.Time
}

// PTYResourceAlert represents a resource usage alert for PTY sessions
type PTYResourceAlert struct {
	ContainerID string
	AlertType   AlertType
	Message     string
	Value       float64
	Threshold   float64
	Timestamp   time.Time
}

// AlertType represents the type of resource alert
type AlertType int

const (
	AlertTypeCPU AlertType = iota
	AlertTypeMemory
	AlertTypeDisk
	AlertTypeNetwork
	AlertTypePID
)

// NewResourceMonitor creates a new resource monitor
func NewResourceMonitor(dockerClient *client.Client) *ResourceMonitor {
	return &ResourceMonitor{
		client:          dockerClient,
		containerLimits: make(map[string]*PTYResourceLimits),
		history:         make(map[string][]*ResourceSnapshot),
		alerts:          make(chan *PTYResourceAlert, 100),
		config: &ResourceMonitorConfig{
			HistorySize:     100,
			SamplingRate:    5 * time.Second,
			AlertThreshold:  80.0,
			EnableAutoScale: false,
			MaxHistoryAge:   1 * time.Hour,
		},
		stopCh: make(chan struct{}),
	}
}

// Start starts the resource monitor
func (rm *ResourceMonitor) Start() {
	go rm.monitorLoop()
	go rm.cleanupLoop()
}

// Stop stops the resource monitor
func (rm *ResourceMonitor) Stop() {
	close(rm.stopCh)
}

// SetContainerLimits sets resource limits for a container
func (rm *ResourceMonitor) SetContainerLimits(containerID string, limits *PTYResourceLimits) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()
	rm.containerLimits[containerID] = limits
}

// GetContainerHistory returns resource usage history for a container
func (rm *ResourceMonitor) GetContainerHistory(containerID string) []*ResourceSnapshot {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	if history, exists := rm.history[containerID]; exists {
		// 복사본 반환
		result := make([]*ResourceSnapshot, len(history))
		copy(result, history)
		return result
	}
	return nil
}

// GetAlertChannel returns the alert channel
func (rm *ResourceMonitor) GetAlertChannel() <-chan *PTYResourceAlert {
	return rm.alerts
}

// monitorLoop continuously monitors container resources
func (rm *ResourceMonitor) monitorLoop() {
	ticker := time.NewTicker(rm.config.SamplingRate)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rm.collectResourceMetrics()
		case <-rm.stopCh:
			return
		}
	}
}

// cleanupLoop periodically cleans up old history data
func (rm *ResourceMonitor) cleanupLoop() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rm.cleanupOldHistory()
		case <-rm.stopCh:
			return
		}
	}
}

// collectResourceMetrics collects metrics for all monitored containers
func (rm *ResourceMonitor) collectResourceMetrics() {
	rm.mutex.RLock()
	containerIDs := make([]string, 0, len(rm.containerLimits))
	for id := range rm.containerLimits {
		containerIDs = append(containerIDs, id)
	}
	rm.mutex.RUnlock()

	for _, containerID := range containerIDs {
		go rm.collectContainerMetrics(containerID)
	}
}

// collectContainerMetrics collects metrics for a specific container
func (rm *ResourceMonitor) collectContainerMetrics(containerID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 컨테이너 통계 가져오기
	stats, err := rm.getContainerStats(ctx, containerID)
	if err != nil {
		log.Errorf("Failed to collect metrics for container %s: %v", containerID, err)
		return
	}

	// 스냅샷 생성
	snapshot := rm.createSnapshot(containerID, stats)

	// 히스토리에 추가
	rm.addToHistory(containerID, snapshot)

	// 임계값 확인
	rm.checkThresholds(containerID, snapshot)
}

// getContainerStats retrieves container statistics
func (rm *ResourceMonitor) getContainerStats(ctx context.Context, containerID string) (*types.StatsJSON, error) {
	statsResponse, err := rm.client.ContainerStats(ctx, containerID, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get container stats: %w", err)
	}
	defer statsResponse.Body.Close()

	var stats types.StatsJSON
	if err := json.NewDecoder(statsResponse.Body).Decode(&stats); err != nil {
		return nil, fmt.Errorf("failed to decode stats: %w", err)
	}

	return &stats, nil
}

// createSnapshot creates a resource snapshot from stats
func (rm *ResourceMonitor) createSnapshot(containerID string, stats *types.StatsJSON) *ResourceSnapshot {
	// CPU 사용률 계산
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stats.CPUStats.SystemUsage - stats.PreCPUStats.SystemUsage)
	numberCPUs := float64(stats.CPUStats.OnlineCPUs)
	if numberCPUs == 0 {
		numberCPUs = float64(len(stats.CPUStats.CPUUsage.PercpuUsage))
	}

	var cpuPercent float64
	if systemDelta > 0 && cpuDelta > 0 {
		cpuPercent = (cpuDelta / systemDelta) * numberCPUs * 100.0
	}

	// 메모리 사용률 계산
	memoryUsage := stats.MemoryStats.Usage - stats.MemoryStats.Stats["cache"]
	memoryLimit := stats.MemoryStats.Limit
	var memoryPercent float64
	if memoryLimit > 0 {
		memoryPercent = (float64(memoryUsage) / float64(memoryLimit)) * 100.0
	}

	// 네트워크 I/O
	var rxBytes, txBytes uint64
	for _, network := range stats.Networks {
		rxBytes += network.RxBytes
		txBytes += network.TxBytes
	}

	// 디스크 I/O
	var readBytes, writeBytes uint64
	for _, blkio := range stats.BlkioStats.IoServiceBytesRecursive {
		if blkio.Op == "Read" {
			readBytes += blkio.Value
		} else if blkio.Op == "Write" {
			writeBytes += blkio.Value
		}
	}

	return &ResourceSnapshot{
		ContainerID:   containerID,
		CPUPercent:    cpuPercent,
		MemoryPercent: memoryPercent,
		MemoryUsage:   int64(memoryUsage),
		NetworkRx:     rxBytes,
		NetworkTx:     txBytes,
		DiskRead:      readBytes,
		DiskWrite:     writeBytes,
		Timestamp:     time.Now(),
	}
}

// addToHistory adds a snapshot to the history
func (rm *ResourceMonitor) addToHistory(containerID string, snapshot *ResourceSnapshot) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	history := rm.history[containerID]
	history = append(history, snapshot)

	// 히스토리 크기 제한
	if len(history) > rm.config.HistorySize {
		history = history[len(history)-rm.config.HistorySize:]
	}

	rm.history[containerID] = history
}

// checkThresholds checks if resource usage exceeds thresholds
func (rm *ResourceMonitor) checkThresholds(containerID string, snapshot *ResourceSnapshot) {
	rm.mutex.RLock()
	limits, hasLimits := rm.containerLimits[containerID]
	rm.mutex.RUnlock()

	if !hasLimits {
		return
	}

	// CPU 임계값 확인
	if snapshot.CPUPercent > rm.config.AlertThreshold {
		alert := &PTYResourceAlert{
			ContainerID: containerID,
			AlertType:   AlertTypeCPU,
			Message:     fmt.Sprintf("CPU usage exceeds threshold: %.2f%%", snapshot.CPUPercent),
			Value:       snapshot.CPUPercent,
			Threshold:   rm.config.AlertThreshold,
			Timestamp:   time.Now(),
		}
		rm.sendAlert(alert)
	}

	// 메모리 임계값 확인
	if snapshot.MemoryPercent > rm.config.AlertThreshold {
		alert := &PTYResourceAlert{
			ContainerID: containerID,
			AlertType:   AlertTypeMemory,
			Message:     fmt.Sprintf("Memory usage exceeds threshold: %.2f%%", snapshot.MemoryPercent),
			Value:       snapshot.MemoryPercent,
			Threshold:   rm.config.AlertThreshold,
			Timestamp:   time.Now(),
		}
		rm.sendAlert(alert)
	}

	// 자동 스케일링 확인
	if rm.config.EnableAutoScale {
		rm.checkAutoScale(containerID, snapshot, limits)
	}
}

// sendAlert sends an alert
func (rm *ResourceMonitor) sendAlert(alert *PTYResourceAlert) {
	select {
	case rm.alerts <- alert:
		log.Warnf("Resource alert: %s", alert.Message)
	default:
		log.Warnf("Alert channel full, dropping alert: %s", alert.Message)
	}
}

// checkAutoScale checks if auto-scaling is needed
func (rm *ResourceMonitor) checkAutoScale(containerID string, snapshot *ResourceSnapshot, limits *PTYResourceLimits) {
	// CPU 자동 스케일링
	if snapshot.CPUPercent > 90 && limits.CPULimit < 4.0 {
		newLimit := limits.CPULimit * 1.5
		if newLimit > 4.0 {
			newLimit = 4.0
		}
		log.Infof("Auto-scaling CPU limit for container %s: %.2f -> %.2f",
			containerID, limits.CPULimit, newLimit)
		limits.CPULimit = newLimit
		rm.applyResourceLimits(containerID, limits)
	}

	// 메모리 자동 스케일링
	if snapshot.MemoryPercent > 90 {
		newLimit := limits.MemoryLimit * 2
		maxMemory := int64(8 * 1024 * 1024 * 1024) // 8GB
		if newLimit > maxMemory {
			newLimit = maxMemory
		}
		log.Infof("Auto-scaling memory limit for container %s: %d -> %d",
			containerID, limits.MemoryLimit, newLimit)
		limits.MemoryLimit = newLimit
		rm.applyResourceLimits(containerID, limits)
	}
}

// applyResourceLimits applies resource limits to a container
func (rm *ResourceMonitor) applyResourceLimits(containerID string, limits *PTYResourceLimits) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	updateConfig := container.UpdateConfig{
		Resources: container.Resources{
			Memory:     limits.MemoryLimit,
			CPUQuota:   int64(limits.CPULimit * 100000),
			CPUPeriod:  100000,
			PidsLimit:  int64Ptr(int64(limits.PIDs)),
		},
	}

	if _, err := rm.client.ContainerUpdate(ctx, containerID, updateConfig); err != nil {
		log.Errorf("Failed to apply resource limits to container %s: %v", containerID, err)
	}
}

// cleanupOldHistory removes old history entries
func (rm *ResourceMonitor) cleanupOldHistory() {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	cutoffTime := time.Now().Add(-rm.config.MaxHistoryAge)

	for containerID, history := range rm.history {
		var newHistory []*ResourceSnapshot
		for _, snapshot := range history {
			if snapshot.Timestamp.After(cutoffTime) {
				newHistory = append(newHistory, snapshot)
			}
		}

		if len(newHistory) > 0 {
			rm.history[containerID] = newHistory
		} else {
			delete(rm.history, containerID)
		}
	}
}

// GetResourceSummary returns a summary of resource usage
func (rm *ResourceMonitor) GetResourceSummary(containerID string) (*ResourceSummary, error) {
	rm.mutex.RLock()
	history, exists := rm.history[containerID]
	rm.mutex.RUnlock()

	if !exists || len(history) == 0 {
		return nil, fmt.Errorf("no history available for container %s", containerID)
	}

	// 통계 계산
	var totalCPU, totalMemory float64
	var maxCPU, maxMemory float64
	var minCPU, minMemory float64 = 100, 100

	for _, snapshot := range history {
		totalCPU += snapshot.CPUPercent
		totalMemory += snapshot.MemoryPercent

		if snapshot.CPUPercent > maxCPU {
			maxCPU = snapshot.CPUPercent
		}
		if snapshot.CPUPercent < minCPU {
			minCPU = snapshot.CPUPercent
		}
		if snapshot.MemoryPercent > maxMemory {
			maxMemory = snapshot.MemoryPercent
		}
		if snapshot.MemoryPercent < minMemory {
			minMemory = snapshot.MemoryPercent
		}
	}

	count := float64(len(history))
	return &ResourceSummary{
		ContainerID:   containerID,
		AvgCPU:        totalCPU / count,
		MaxCPU:        maxCPU,
		MinCPU:        minCPU,
		AvgMemory:     totalMemory / count,
		MaxMemory:     maxMemory,
		MinMemory:     minMemory,
		SampleCount:   len(history),
		TimeRange:     history[len(history)-1].Timestamp.Sub(history[0].Timestamp),
	}, nil
}

// ResourceSummary represents a summary of resource usage
type ResourceSummary struct {
	ContainerID string
	AvgCPU      float64
	MaxCPU      float64
	MinCPU      float64
	AvgMemory   float64
	MaxMemory   float64
	MinMemory   float64
	SampleCount int
	TimeRange   time.Duration
}