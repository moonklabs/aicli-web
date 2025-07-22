package docker

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
)

// StatsCollector Docker 컨테이너 통계를 수집합니다.
type StatsCollector struct {
	client *Client
	cache  sync.Map // containerID -> ContainerStats
}

// NewStatsCollector 새로운 통계 수집기를 생성합니다.
func NewStatsCollector(client *Client) *StatsCollector {
	return &StatsCollector{
		client: client,
	}
}

// ContainerStats 컨테이너 통계 정보
type ContainerStats struct {
	ContainerID  string    `json:"container_id"`
	CPUPercent   float64   `json:"cpu_percent"`
	MemoryUsage  int64     `json:"memory_usage"`
	MemoryLimit  int64     `json:"memory_limit"`
	MemoryPercent float64  `json:"memory_percent"`
	NetworkRxMB  float64   `json:"network_rx_mb"`
	NetworkTxMB  float64   `json:"network_tx_mb"`
	BlockRead    int64     `json:"block_read"`
	BlockWrite   int64     `json:"block_write"`
	PidsCount    int64     `json:"pids_count"`
	Timestamp    time.Time `json:"timestamp"`
}

// Collect 단일 컨테이너의 통계를 수집합니다.
func (sc *StatsCollector) Collect(ctx context.Context, containerID string) (*ContainerStats, error) {
	stats, err := sc.client.cli.ContainerStats(ctx, containerID, false)
	if err != nil {
		return nil, err
	}
	defer stats.Body.Close()

	var v types.StatsJSON
	if err := json.NewDecoder(stats.Body).Decode(&v); err != nil {
		return nil, err
	}

	// CPU 사용률 계산
	cpuPercent := calculateCPUPercent(&v)

	// 메모리 사용량 계산
	memUsage := v.MemoryStats.Usage
	if v.MemoryStats.Stats["cache"] > 0 {
		memUsage -= v.MemoryStats.Stats["cache"]
	}
	memLimit := v.MemoryStats.Limit
	memPercent := 0.0
	if memLimit > 0 {
		memPercent = (float64(memUsage) / float64(memLimit)) * 100.0
	}

	// 네트워크 통계 계산
	var rxBytes, txBytes uint64
	for _, net := range v.Networks {
		rxBytes += net.RxBytes
		txBytes += net.TxBytes
	}

	// 블록 I/O 통계
	var blockRead, blockWrite uint64
	for _, io := range v.BlkioStats.IoServiceBytesRecursive {
		switch io.Op {
		case "Read":
			blockRead += io.Value
		case "Write":
			blockWrite += io.Value
		}
	}

	containerStats := &ContainerStats{
		ContainerID:   containerID,
		CPUPercent:    cpuPercent,
		MemoryUsage:   int64(memUsage),
		MemoryLimit:   int64(memLimit),
		MemoryPercent: memPercent,
		NetworkRxMB:   float64(rxBytes) / 1024 / 1024,
		NetworkTxMB:   float64(txBytes) / 1024 / 1024,
		BlockRead:     int64(blockRead),
		BlockWrite:    int64(blockWrite),
		PidsCount:     int64(v.PidsStats.Current),
		Timestamp:     time.Now(),
	}

	// 캐시에 저장
	sc.cache.Store(containerID, containerStats)

	return containerStats, nil
}

// calculateCPUPercent CPU 사용률을 계산합니다.
func calculateCPUPercent(stats *types.StatsJSON) float64 {
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stats.CPUStats.SystemUsage - stats.PreCPUStats.SystemUsage)

	if systemDelta > 0.0 && cpuDelta > 0.0 {
		return (cpuDelta / systemDelta) * float64(len(stats.CPUStats.CPUUsage.PercpuUsage)) * 100.0
	}
	return 0.0
}

// CollectAll aicli에서 관리하는 모든 컨테이너의 통계를 수집합니다.
func (sc *StatsCollector) CollectAll(ctx context.Context) (map[string]*ContainerStats, error) {
	// aicli 관리 컨테이너 조회
	containers, err := sc.client.cli.ContainerList(ctx, types.ContainerListOptions{
		Filters: filters.NewArgs(
			filters.Arg("label", sc.client.labelKey("managed")+"=true"),
		),
	})
	if err != nil {
		return nil, err
	}

	result := make(map[string]*ContainerStats)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, container := range containers {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()

			stats, err := sc.Collect(ctx, id)
			if err == nil {
				mu.Lock()
				result[id] = stats
				mu.Unlock()
			}
		}(container.ID)
	}

	wg.Wait()
	return result, nil
}

// GetCachedStats 캐시된 통계를 반환합니다.
func (sc *StatsCollector) GetCachedStats(containerID string) (*ContainerStats, bool) {
	if stats, ok := sc.cache.Load(containerID); ok {
		return stats.(*ContainerStats), true
	}
	return nil, false
}

// ClearCache 특정 컨테이너의 캐시를 지웁니다.
func (sc *StatsCollector) ClearCache(containerID string) {
	sc.cache.Delete(containerID)
}

// ClearAllCache 모든 캐시를 지웁니다.
func (sc *StatsCollector) ClearAllCache() {
	sc.cache.Range(func(key, value interface{}) bool {
		sc.cache.Delete(key)
		return true
	})
}

// Monitor 컨테이너의 실시간 통계 모니터링을 시작합니다.
func (sc *StatsCollector) Monitor(ctx context.Context, containerID string, interval time.Duration) (<-chan *ContainerStats, error) {
	if interval == 0 {
		interval = time.Second
	}

	statsChan := make(chan *ContainerStats, 10)

	go func() {
		defer close(statsChan)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				stats, err := sc.Collect(ctx, containerID)
				if err != nil {
					continue // 에러 발생 시 다음 수집까지 대기
				}

				select {
				case statsChan <- stats:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return statsChan, nil
}

// MonitorAll 모든 관리 컨테이너의 실시간 통계 모니터링을 시작합니다.
func (sc *StatsCollector) MonitorAll(ctx context.Context, interval time.Duration) (<-chan map[string]*ContainerStats, error) {
	if interval == 0 {
		interval = 5 * time.Second
	}

	statsChan := make(chan map[string]*ContainerStats, 10)

	go func() {
		defer close(statsChan)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				stats, err := sc.CollectAll(ctx)
				if err != nil {
					continue // 에러 발생 시 다음 수집까지 대기
				}

				select {
				case statsChan <- stats:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return statsChan, nil
}

// SystemStats 시스템 통계 정보
type SystemStats struct {
	ContainersTotal   int     `json:"containers_total"`
	ContainersRunning int     `json:"containers_running"`
	ContainersStopped int     `json:"containers_stopped"`
	ImagesTotal       int     `json:"images_total"`
	MemoryTotal       int64   `json:"memory_total"`
	MemoryUsed        int64   `json:"memory_used"`
	CPUCount          int     `json:"cpu_count"`
	DockerVersion     string  `json:"docker_version"`
	Timestamp         time.Time `json:"timestamp"`
}

// GetSystemStats 시스템 전체 통계를 조회합니다.
func (sc *StatsCollector) GetSystemStats(ctx context.Context) (*SystemStats, error) {
	info, err := sc.client.cli.Info(ctx)
	if err != nil {
		return nil, err
	}

	version, err := sc.client.cli.ServerVersion(ctx)
	if err != nil {
		return nil, err
	}

	return &SystemStats{
		ContainersTotal:   info.Containers,
		ContainersRunning: info.ContainersRunning,
		ContainersStopped: info.ContainersStopped,
		ImagesTotal:       info.Images,
		MemoryTotal:       info.MemTotal,
		CPUCount:          info.NCPU,
		DockerVersion:     version.Version,
		Timestamp:         time.Now(),
	}, nil
}

// AggregatedStats 집계된 통계 정보
type AggregatedStats struct {
	TotalContainers   int     `json:"total_containers"`
	RunningContainers int     `json:"running_containers"`
	TotalCPUUsage     float64 `json:"total_cpu_usage"`
	TotalMemoryUsage  int64   `json:"total_memory_usage"`
	TotalMemoryLimit  int64   `json:"total_memory_limit"`
	TotalNetworkRx    float64 `json:"total_network_rx"`
	TotalNetworkTx    float64 `json:"total_network_tx"`
	AverageCPUUsage   float64 `json:"average_cpu_usage"`
	AverageMemoryUsage float64 `json:"average_memory_usage"`
	Timestamp         time.Time `json:"timestamp"`
}

// GetAggregatedStats 집계된 통계를 계산합니다.
func (sc *StatsCollector) GetAggregatedStats(ctx context.Context) (*AggregatedStats, error) {
	statsMap, err := sc.CollectAll(ctx)
	if err != nil {
		return nil, err
	}

	if len(statsMap) == 0 {
		return &AggregatedStats{
			Timestamp: time.Now(),
		}, nil
	}

	aggregated := &AggregatedStats{
		TotalContainers: len(statsMap),
		Timestamp:       time.Now(),
	}

	var totalCPU, totalMemory float64
	for _, stats := range statsMap {
		if stats != nil {
			aggregated.TotalCPUUsage += stats.CPUPercent
			aggregated.TotalMemoryUsage += stats.MemoryUsage
			aggregated.TotalMemoryLimit += stats.MemoryLimit
			aggregated.TotalNetworkRx += stats.NetworkRxMB
			aggregated.TotalNetworkTx += stats.NetworkTxMB

			totalCPU += stats.CPUPercent
			totalMemory += stats.MemoryPercent

			if stats.CPUPercent > 0 { // 실행 중인 컨테이너로 간주
				aggregated.RunningContainers++
			}
		}
	}

	if aggregated.TotalContainers > 0 {
		aggregated.AverageCPUUsage = totalCPU / float64(aggregated.TotalContainers)
		aggregated.AverageMemoryUsage = totalMemory / float64(aggregated.TotalContainers)
	}

	return aggregated, nil
}