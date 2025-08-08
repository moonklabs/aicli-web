package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

// ContainerMonitor monitors Docker container resources
type ContainerMonitor struct {
	client        *client.Client
	sessions      map[string]string // sessionID -> containerID
	stats         map[string]*ContainerResourceStats
	monitorTicker *time.Ticker
	config        *MonitorConfig
	mutex         sync.RWMutex
	stopCh        chan struct{}
}

// MonitorConfig contains monitoring configuration
type MonitorConfig struct {
	MonitorInterval time.Duration
	CPUThreshold    float64
	MemoryThreshold float64
	EnableAlerts    bool
}

// ContainerResourceStats contains container resource statistics
type ContainerResourceStats struct {
	ContainerID string
	CPUUsage    float64
	MemoryUsage int64
	MemoryLimit int64
	NetworkIO   NetworkIOStats
	BlockIO     BlockIOStats
	PIDs        int
	Timestamp   time.Time
}

// NetworkIOStats contains network I/O statistics
type NetworkIOStats struct {
	RxBytes   uint64
	TxBytes   uint64
	RxPackets uint64
	TxPackets uint64
}

// BlockIOStats contains block I/O statistics
type BlockIOStats struct {
	ReadBytes  uint64
	WriteBytes uint64
	ReadOps    uint64
	WriteOps   uint64
}

// NewContainerMonitor creates a new container monitor
func NewContainerMonitor(dockerClient *client.Client, interval time.Duration) *ContainerMonitor {
	if interval == 0 {
		interval = 5 * time.Second
	}

	return &ContainerMonitor{
		client:   dockerClient,
		sessions: make(map[string]string),
		stats:    make(map[string]*ContainerResourceStats),
		config: &MonitorConfig{
			MonitorInterval: interval,
			CPUThreshold:    80.0,
			MemoryThreshold: 90.0,
			EnableAlerts:    true,
		},
		stopCh: make(chan struct{}),
	}
}

// Start starts the container monitor
func (cm *ContainerMonitor) Start() {
	cm.monitorTicker = time.NewTicker(cm.config.MonitorInterval)

	go func() {
		defer cm.monitorTicker.Stop()

		for {
			select {
			case <-cm.monitorTicker.C:
				cm.collectResourceStats()
			case <-cm.stopCh:
				return
			}
		}
	}()
}

// Stop stops the container monitor
func (cm *ContainerMonitor) Stop() {
	close(cm.stopCh)
}

// AddSession adds a session to monitor
func (cm *ContainerMonitor) AddSession(sessionID, containerID string) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	cm.sessions[sessionID] = containerID
}

// RemoveSession removes a session from monitoring
func (cm *ContainerMonitor) RemoveSession(sessionID string) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	containerID := cm.sessions[sessionID]
	delete(cm.sessions, sessionID)
	delete(cm.stats, containerID)
}

// GetContainerStats returns statistics for a container
func (cm *ContainerMonitor) GetContainerStats(containerID string) (*ContainerResourceStats, error) {
	cm.mutex.RLock()
	stats, exists := cm.stats[containerID]
	cm.mutex.RUnlock()

	if !exists {
		// 즉시 수집 시도
		return cm.getContainerStats(containerID)
	}

	return stats, nil
}

// collectResourceStats collects resource statistics for all monitored containers
func (cm *ContainerMonitor) collectResourceStats() {
	cm.mutex.RLock()
	containerIDs := make([]string, 0, len(cm.sessions))
	for _, containerID := range cm.sessions {
		containerIDs = append(containerIDs, containerID)
	}
	cm.mutex.RUnlock()

	// 비동기 통계 수집
	var wg sync.WaitGroup
	for _, containerID := range containerIDs {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()

			stats, err := cm.getContainerStats(id)
			if err != nil {
				log.Errorf("Failed to collect stats for container %s: %v", id, err)
				return
			}

			cm.mutex.Lock()
			cm.stats[id] = stats
			cm.mutex.Unlock()

			// 리소스 임계값 확인
			if cm.config.EnableAlerts {
				cm.checkResourceLimits(id, stats)
			}
		}(containerID)
	}
	wg.Wait()
}

// getContainerStats gets statistics for a specific container
func (cm *ContainerMonitor) getContainerStats(containerID string) (*ContainerResourceStats, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Docker Stats API 호출
	statsResponse, err := cm.client.ContainerStats(ctx, containerID, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get container stats: %w", err)
	}
	defer statsResponse.Body.Close()

	// Stats 디코딩
	var stats types.StatsJSON
	decoder := json.NewDecoder(statsResponse.Body)
	if err := decoder.Decode(&stats); err != nil {
		if err != io.EOF {
			return nil, fmt.Errorf("failed to decode stats: %w", err)
		}
	}

	// CPU 사용률 계산
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stats.CPUStats.SystemUsage - stats.PreCPUStats.SystemUsage)
	numberCPUs := float64(stats.CPUStats.OnlineCPUs)
	if numberCPUs == 0 {
		numberCPUs = float64(len(stats.CPUStats.CPUUsage.PercpuUsage))
	}

	var cpuUsage float64
	if systemDelta > 0 && cpuDelta > 0 {
		cpuUsage = (cpuDelta / systemDelta) * numberCPUs * 100.0
	}

	// 메모리 사용량
	memoryUsage := stats.MemoryStats.Usage - stats.MemoryStats.Stats["cache"]
	memoryLimit := stats.MemoryStats.Limit

	// 네트워크 I/O 통계
	var rxBytes, txBytes, rxPackets, txPackets uint64
	for _, network := range stats.Networks {
		rxBytes += network.RxBytes
		txBytes += network.TxBytes
		rxPackets += network.RxPackets
		txPackets += network.TxPackets
	}

	// 블록 I/O 통계
	var readBytes, writeBytes, readOps, writeOps uint64
	for _, blkio := range stats.BlkioStats.IoServiceBytesRecursive {
		if blkio.Op == "Read" {
			readBytes += blkio.Value
		} else if blkio.Op == "Write" {
			writeBytes += blkio.Value
		}
	}
	for _, blkio := range stats.BlkioStats.IoServicedRecursive {
		if blkio.Op == "Read" {
			readOps += blkio.Value
		} else if blkio.Op == "Write" {
			writeOps += blkio.Value
		}
	}

	return &ContainerResourceStats{
		ContainerID: containerID,
		CPUUsage:    cpuUsage,
		MemoryUsage: int64(memoryUsage),
		MemoryLimit: int64(memoryLimit),
		NetworkIO: NetworkIOStats{
			RxBytes:   rxBytes,
			TxBytes:   txBytes,
			RxPackets: rxPackets,
			TxPackets: txPackets,
		},
		BlockIO: BlockIOStats{
			ReadBytes:  readBytes,
			WriteBytes: writeBytes,
			ReadOps:    readOps,
			WriteOps:   writeOps,
		},
		PIDs:      int(stats.PidsStats.Current),
		Timestamp: time.Now(),
	}, nil
}

// checkResourceLimits checks if resource usage exceeds thresholds
func (cm *ContainerMonitor) checkResourceLimits(containerID string, stats *ContainerResourceStats) {
	// CPU 사용률 확인
	if stats.CPUUsage > cm.config.CPUThreshold {
		log.Warnf("Container %s CPU usage (%.2f%%) exceeds threshold (%.2f%%)",
			containerID, stats.CPUUsage, cm.config.CPUThreshold)
	}

	// 메모리 사용률 확인
	if stats.MemoryLimit > 0 {
		memoryPercent := (float64(stats.MemoryUsage) / float64(stats.MemoryLimit)) * 100
		if memoryPercent > cm.config.MemoryThreshold {
			log.Warnf("Container %s memory usage (%.2f%%) exceeds threshold (%.2f%%)",
				containerID, memoryPercent, cm.config.MemoryThreshold)
		}
	}
}

// GetAllStats returns statistics for all monitored containers
func (cm *ContainerMonitor) GetAllStats() map[string]*ContainerResourceStats {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	allStats := make(map[string]*ContainerResourceStats)
	for containerID, stats := range cm.stats {
		allStats[containerID] = stats
	}
	return allStats
}

// SetThresholds sets resource monitoring thresholds
func (cm *ContainerMonitor) SetThresholds(cpu, memory float64) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cm.config.CPUThreshold = cpu
	cm.config.MemoryThreshold = memory
}