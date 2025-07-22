package status

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aicli/aicli-web/internal/docker"
)

// ResourceMonitor 리소스 모니터링 시스템
type ResourceMonitor struct {
	// 의존성
	containerManager docker.ContainerManagement
	statsCollector   docker.StatsCollection
	dockerManager    docker.DockerManager

	// 캐시
	statsCache     map[string]*docker.ContainerStats
	cacheMutex     sync.RWMutex
	cacheExpiry    time.Duration

	// 설정
	collectInterval time.Duration
	retentionPeriod time.Duration

	// 제어
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	logger  Logger

	// 모니터링 채널들
	activeMonitors map[string]chan *WorkspaceMetrics
	monitorsMutex  sync.RWMutex
}

// NewResourceMonitor 새로운 리소스 모니터 생성
func NewResourceMonitor(
	containerManager docker.ContainerManagement,
	dockerManager docker.DockerManager,
) *ResourceMonitor {
	ctx, cancel := context.WithCancel(context.Background())

	return &ResourceMonitor{
		containerManager: containerManager,
		statsCollector:   dockerManager.Stats(),
		dockerManager:    dockerManager,
		statsCache:       make(map[string]*docker.ContainerStats),
		cacheExpiry:      30 * time.Second,
		collectInterval:  10 * time.Second,
		retentionPeriod:  24 * time.Hour,
		ctx:              ctx,
		cancel:           cancel,
		logger:           &defaultLogger{},
		activeMonitors:   make(map[string]chan *WorkspaceMetrics),
	}
}

// SetLogger 로거 설정
func (rm *ResourceMonitor) SetLogger(logger Logger) {
	rm.logger = logger
}

// SetCollectInterval 수집 간격 설정
func (rm *ResourceMonitor) SetCollectInterval(interval time.Duration) {
	rm.collectInterval = interval
}

// Start 리소스 모니터링 시작
func (rm *ResourceMonitor) Start() error {
	rm.logger.Info("리소스 모니터 시작 - 수집 간격: %v", rm.collectInterval)
	
	rm.wg.Add(1)
	go rm.cleanupLoop()
	
	return nil
}

// Stop 리소스 모니터링 중지
func (rm *ResourceMonitor) Stop() error {
	rm.logger.Info("리소스 모니터 중지 중...")
	
	rm.cancel()
	rm.wg.Wait()
	
	// 활성 모니터들 정리
	rm.monitorsMutex.Lock()
	for workspaceID, ch := range rm.activeMonitors {
		close(ch)
		delete(rm.activeMonitors, workspaceID)
	}
	rm.monitorsMutex.Unlock()
	
	rm.logger.Info("리소스 모니터 중지 완료")
	return nil
}

// StartMonitoring 특정 워크스페이스 모니터링 시작
func (rm *ResourceMonitor) StartMonitoring(ctx context.Context, workspaceID string) (<-chan *WorkspaceMetrics, error) {
	rm.logger.Debug("워크스페이스 모니터링 시작: %s", workspaceID)
	
	metricsChan := make(chan *WorkspaceMetrics, 10)
	
	// 활성 모니터에 추가
	rm.monitorsMutex.Lock()
	if existingCh, exists := rm.activeMonitors[workspaceID]; exists {
		close(existingCh)
	}
	rm.activeMonitors[workspaceID] = metricsChan
	rm.monitorsMutex.Unlock()

	rm.wg.Add(1)
	go func() {
		defer rm.wg.Done()
		defer func() {
			rm.monitorsMutex.Lock()
			delete(rm.activeMonitors, workspaceID)
			rm.monitorsMutex.Unlock()
			close(metricsChan)
		}()

		ticker := time.NewTicker(rm.collectInterval)
		defer ticker.Stop()

		rm.logger.Debug("워크스페이스 %s 메트릭 수집 루프 시작", workspaceID)

		for {
			select {
			case <-ctx.Done():
				rm.logger.Debug("워크스페이스 %s 모니터링 컨텍스트 취소", workspaceID)
				return
			case <-rm.ctx.Done():
				rm.logger.Debug("워크스페이스 %s 모니터링 전역 종료", workspaceID)
				return
			case <-ticker.C:
				metrics := rm.collectMetrics(ctx, workspaceID)
				if metrics != nil {
					select {
					case metricsChan <- metrics:
					case <-ctx.Done():
						return
					case <-rm.ctx.Done():
						return
					default:
						// 채널이 가득 찬 경우 가장 오래된 데이터 제거
						select {
						case <-metricsChan:
							metricsChan <- metrics
						default:
						}
					}
				}
			}
		}
	}()

	return metricsChan, nil
}

// StopMonitoring 특정 워크스페이스 모니터링 중지
func (rm *ResourceMonitor) StopMonitoring(workspaceID string) {
	rm.logger.Debug("워크스페이스 모니터링 중지: %s", workspaceID)
	
	rm.monitorsMutex.Lock()
	if ch, exists := rm.activeMonitors[workspaceID]; exists {
		close(ch)
		delete(rm.activeMonitors, workspaceID)
	}
	rm.monitorsMutex.Unlock()
}

// collectMetrics 메트릭 수집
func (rm *ResourceMonitor) collectMetrics(ctx context.Context, workspaceID string) *WorkspaceMetrics {
	// 워크스페이스의 컨테이너 조회
	containers, err := rm.containerManager.ListWorkspaceContainers(ctx, workspaceID)
	if err != nil {
		rm.logger.Error("컨테이너 목록 조회 실패", err, "workspace_id", workspaceID)
		return nil
	}

	if len(containers) == 0 {
		return nil
	}

	// 가장 최근 컨테이너의 통계 수집
	container := containers[0]
	if container.GetState() != "running" {
		return nil
	}

	// 캐시에서 이전 통계 확인
	rm.cacheMutex.RLock()
	cachedStats, hasCached := rm.statsCache[container.GetID()]
	rm.cacheMutex.RUnlock()

	// 새로운 통계 수집
	stats, err := rm.statsCollector.Collect(ctx, container.GetID())
	if err != nil {
		rm.logger.Error("컨테이너 통계 수집 실패", err, 
			"workspace_id", workspaceID, 
			"container_id", container.GetID())
		return nil
	}

	// 캐시 업데이트
	rm.cacheMutex.Lock()
	rm.statsCache[container.GetID()] = stats
	rm.cacheMutex.Unlock()

	// 이전 통계와 비교하여 uptime 계산
	var uptime string
	if createdAt := container.GetCreatedAt(); !createdAt.IsZero() {
		uptime = time.Since(createdAt).String()
	}

	metrics := &WorkspaceMetrics{
		CPUPercent:   stats.CPUPercent,
		MemoryUsage:  stats.MemoryUsage,
		MemoryLimit:  stats.MemoryLimit,
		NetworkRxMB:  stats.NetworkRxMB,
		NetworkTxMB:  stats.NetworkTxMB,
		Uptime:       uptime,
		LastActivity: time.Now(),
		ErrorCount:   0, // 에러 카운터는 별도로 관리
	}

	// CPU 사용률 이상 감지
	if stats.CPUPercent > 80.0 {
		rm.logger.Warn("높은 CPU 사용률 감지: %.2f%% (워크스페이스: %s)", 
			stats.CPUPercent, workspaceID)
	}

	// 메모리 사용률 이상 감지
	if stats.MemoryLimit > 0 {
		memoryPercent := float64(stats.MemoryUsage) / float64(stats.MemoryLimit) * 100
		if memoryPercent > 85.0 {
			rm.logger.Warn("높은 메모리 사용률 감지: %.2f%% (워크스페이스: %s)", 
				memoryPercent, workspaceID)
		}
	}

	// 네트워크 사용량 추이 분석
	if hasCached && cachedStats != nil {
		rxDelta := stats.NetworkRxMB - cachedStats.NetworkRxMB
		txDelta := stats.NetworkTxMB - cachedStats.NetworkTxMB
		
		if rxDelta > 100 || txDelta > 100 { // 100MB 이상 변화
			rm.logger.Debug("높은 네트워크 활동 감지: RX +%.2fMB, TX +%.2fMB (워크스페이스: %s)",
				rxDelta, txDelta, workspaceID)
		}
	}

	return metrics
}

// GetResourceSummary 전체 리소스 사용량 집계
func (rm *ResourceMonitor) GetResourceSummary(ctx context.Context) (*ResourceSummary, error) {
	rm.logger.Debug("전체 리소스 사용량 집계 시작")

	// 시스템 통계 수집
	systemStats, err := rm.statsCollector.GetSystemStats(ctx)
	if err != nil {
		rm.logger.Error("시스템 통계 수집 실패", err)
		return nil, err
	}

	// 모든 컨테이너 통계 수집
	allStats, err := rm.statsCollector.CollectAll(ctx)
	if err != nil {
		rm.logger.Error("모든 컨테이너 통계 수집 실패", err)
		return nil, err
	}

	// 집계 계산
	var totalCPU float64
	var totalMemory int64
	var totalNetworkIO float64
	var activeContainers int

	for containerID, stats := range allStats {
		if stats != nil {
			activeContainers++
			totalCPU += stats.CPUPercent
			totalMemory += stats.MemoryUsage
			totalNetworkIO += stats.NetworkRxMB + stats.NetworkTxMB
		}
		rm.logger.Debug("컨테이너 %s: CPU=%.2f%%, MEM=%dMB", 
			containerID[:12], stats.CPUPercent, stats.MemoryUsage/(1024*1024))
	}

	summary := &ResourceSummary{
		TotalWorkspaces:  len(allStats), // 컨테이너 수로 근사
		ActiveContainers: activeContainers,
		TotalCPUUsage:    totalCPU,
		TotalMemoryUsage: totalMemory,
		TotalNetworkIO:   totalNetworkIO,
		LastUpdated:      time.Now(),
		SystemStats:      systemStats,
	}

	rm.logger.Debug("리소스 집계 완료: 워크스페이스 %d개, 활성 컨테이너 %d개, 평균 CPU %.2f%%",
		summary.TotalWorkspaces, summary.ActiveContainers, totalCPU/float64(activeContainers))

	return summary, nil
}

// GetMetricsHistory 메트릭 히스토리 조회 (향후 구현)
func (rm *ResourceMonitor) GetMetricsHistory(workspaceID string, since time.Time) ([]*WorkspaceMetrics, error) {
	// TODO: 메트릭 히스토리 저장 및 조회 구현
	return nil, fmt.Errorf("메트릭 히스토리 기능은 아직 구현되지 않았습니다")
}

// GetActiveMonitors 활성 모니터 목록 조회
func (rm *ResourceMonitor) GetActiveMonitors() []string {
	rm.monitorsMutex.RLock()
	defer rm.monitorsMutex.RUnlock()

	monitors := make([]string, 0, len(rm.activeMonitors))
	for workspaceID := range rm.activeMonitors {
		monitors = append(monitors, workspaceID)
	}

	return monitors
}

// cleanupLoop 캐시 정리 루프
func (rm *ResourceMonitor) cleanupLoop() {
	defer rm.wg.Done()

	ticker := time.NewTicker(5 * time.Minute) // 5분마다 정리
	defer ticker.Stop()

	for {
		select {
		case <-rm.ctx.Done():
			return
		case <-ticker.C:
			rm.cleanupCache()
		}
	}
}

// cleanupCache 캐시 정리
func (rm *ResourceMonitor) cleanupCache() {
	rm.cacheMutex.Lock()
	defer rm.cacheMutex.Unlock()

	now := time.Now()
	for containerID, stats := range rm.statsCache {
		if stats != nil && now.Sub(stats.Timestamp) > rm.cacheExpiry {
			delete(rm.statsCache, containerID)
			rm.logger.Debug("만료된 캐시 제거: %s", containerID[:12])
		}
	}
}

// GetCacheStats 캐시 통계 조회
func (rm *ResourceMonitor) GetCacheStats() CacheStats {
	rm.cacheMutex.RLock()
	defer rm.cacheMutex.RUnlock()

	return CacheStats{
		CachedContainers: len(rm.statsCache),
		CacheExpiry:      rm.cacheExpiry,
		LastCleanup:      time.Now(), // 실제로는 마지막 정리 시간을 저장해야 함
	}
}

// ResourceSummary 리소스 사용량 요약
type ResourceSummary struct {
	// 기본 정보
	TotalWorkspaces  int       `json:"total_workspaces"`
	ActiveContainers int       `json:"active_containers"`
	LastUpdated      time.Time `json:"last_updated"`

	// 리소스 사용량
	TotalCPUUsage    float64 `json:"total_cpu_usage"`
	TotalMemoryUsage int64   `json:"total_memory_usage"`
	TotalNetworkIO   float64 `json:"total_network_io_mb"`

	// 시스템 통계
	SystemStats *docker.SystemStats `json:"system_stats,omitempty"`

	// 추가 메트릭
	AverageCPUUsage    float64 `json:"average_cpu_usage"`
	AverageMemoryUsage int64   `json:"average_memory_usage"`
	PeakCPUUsage       float64 `json:"peak_cpu_usage"`
	PeakMemoryUsage    int64   `json:"peak_memory_usage"`
}

// CacheStats 캐시 통계
type CacheStats struct {
	CachedContainers int           `json:"cached_containers"`
	CacheExpiry      time.Duration `json:"cache_expiry"`
	LastCleanup      time.Time     `json:"last_cleanup"`
}

// MonitorStats 모니터 통계
type MonitorStats struct {
	ActiveMonitors   int           `json:"active_monitors"`
	CollectInterval  time.Duration `json:"collect_interval"`
	CacheStats       CacheStats    `json:"cache_stats"`
	TotalCollections int64         `json:"total_collections"`
	FailedCollections int64        `json:"failed_collections"`
}

// GetMonitorStats 모니터 통계 조회
func (rm *ResourceMonitor) GetMonitorStats() MonitorStats {
	rm.monitorsMutex.RLock()
	activeCount := len(rm.activeMonitors)
	rm.monitorsMutex.RUnlock()

	return MonitorStats{
		ActiveMonitors:  activeCount,
		CollectInterval: rm.collectInterval,
		CacheStats:      rm.GetCacheStats(),
		// TODO: 수집 통계는 별도로 추적해야 함
	}
}