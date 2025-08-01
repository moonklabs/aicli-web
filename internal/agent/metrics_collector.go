package agent

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/aicli/aicli-web/internal/models"
)

// basicMetricsCollector 기본 메트릭 수집기 구현
type basicMetricsCollector struct {
	// 메트릭 저장소 (메모리 기반)
	metricsStore map[string][]AgentMetrics
	storeMutex   sync.RWMutex

	// Docker 어댑터 (컨테이너 메트릭 수집용)
	dockerAdapter DockerAdapter

	// 설정
	maxHistorySize int
	retentionTime  time.Duration
}

// NewBasicMetricsCollector 새 기본 메트릭 수집기 생성
func NewBasicMetricsCollector(dockerAdapter DockerAdapter) MetricsCollector {
	return &basicMetricsCollector{
		metricsStore:   make(map[string][]AgentMetrics),
		dockerAdapter:  dockerAdapter,
		maxHistorySize: 1000,           // 에이전트당 최대 1000개 메트릭 보관
		retentionTime:  24 * time.Hour, // 24시간 보관
	}
}

// CollectAgentMetrics 에이전트 메트릭 수집
func (c *basicMetricsCollector) CollectAgentMetrics(ctx context.Context, agent *models.Agent) (AgentMetrics, error) {
	metrics := AgentMetrics{
		AgentID:   agent.ID,
		Timestamp: time.Now(),
		Custom:    make(map[string]interface{}),
	}

	// 시스템 메트릭 수집
	if err := c.collectSystemMetrics(&metrics); err != nil {
		return metrics, fmt.Errorf("failed to collect system metrics: %w", err)
	}

	// 컨테이너 메트릭 수집 (컨테이너가 있는 경우)
	if agent.ContainerID != "" && c.dockerAdapter != nil {
		if err := c.collectContainerMetrics(ctx, agent.ContainerID, &metrics); err != nil {
			// 컨테이너 메트릭 수집 실패는 경고만 하고 계속 진행
			metrics.Custom["container_metrics_error"] = err.Error()
		}
	}

	// 에이전트별 커스텀 메트릭 수집
	c.collectAgentSpecificMetrics(agent, &metrics)

	return metrics, nil
}

// StoreMetrics 메트릭 저장
func (c *basicMetricsCollector) StoreMetrics(ctx context.Context, metrics AgentMetrics) error {
	c.storeMutex.Lock()
	defer c.storeMutex.Unlock()

	agentMetrics := c.metricsStore[metrics.AgentID]

	// 새 메트릭 추가
	agentMetrics = append(agentMetrics, metrics)

	// 보관 정책 적용
	agentMetrics = c.applyRetentionPolicy(agentMetrics)

	c.metricsStore[metrics.AgentID] = agentMetrics

	return nil
}

// GetMetricsHistory 메트릭 히스토리 조회
func (c *basicMetricsCollector) GetMetricsHistory(ctx context.Context, agentID string, period time.Duration) ([]AgentMetrics, error) {
	c.storeMutex.RLock()
	defer c.storeMutex.RUnlock()

	allMetrics := c.metricsStore[agentID]
	if len(allMetrics) == 0 {
		return []AgentMetrics{}, nil
	}

	// 기간 내 메트릭 필터링
	cutoff := time.Now().Add(-period)
	var filteredMetrics []AgentMetrics

	for _, metric := range allMetrics {
		if metric.Timestamp.After(cutoff) {
			filteredMetrics = append(filteredMetrics, metric)
		}
	}

	return filteredMetrics, nil
}

// collectSystemMetrics 시스템 메트릭 수집
func (c *basicMetricsCollector) collectSystemMetrics(metrics *AgentMetrics) error {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// CPU 메트릭 (단순화된 버전)
	metrics.CPU = CPUMetrics{
		UsagePercent: 0.0, // TODO: 실제 CPU 사용률 계산
		SystemTime:   float64(runtime.NumCPU()),
		UserTime:     0.0,
	}

	// 메모리 메트릭
	metrics.Memory = MemoryMetrics{
		UsageBytes:   int64(m.Alloc),
		LimitBytes:   int64(m.Sys),
		UsagePercent: float64(m.Alloc) / float64(m.Sys) * 100,
	}

	// 네트워크 메트릭 (플레이스홀더)
	metrics.Network = NetworkMetrics{
		RxBytes:   0, // TODO: 실제 네트워크 통계 수집
		TxBytes:   0,
		RxPackets: 0,
		TxPackets: 0,
	}

	// 디스크 메트릭 (플레이스홀더)
	metrics.Disk = DiskMetrics{
		ReadBytes:  0, // TODO: 실제 디스크 I/O 통계 수집
		WriteBytes: 0,
		ReadOps:    0,
		WriteOps:   0,
	}

	return nil
}

// collectContainerMetrics 컨테이너 메트릭 수집
func (c *basicMetricsCollector) collectContainerMetrics(ctx context.Context, containerID string, metrics *AgentMetrics) error {
	containerMetrics, err := c.dockerAdapter.GetContainerMetrics(ctx, containerID)
	if err != nil {
		return fmt.Errorf("failed to get container metrics: %w", err)
	}

	// 컨테이너 메트릭을 에이전트 메트릭에 병합
	metrics.CPU = containerMetrics.CPU
	metrics.Memory = containerMetrics.Memory
	metrics.Network = containerMetrics.Network
	metrics.Disk = containerMetrics.BlockIO

	// 컨테이너 정보를 커스텀 메트릭에 추가
	metrics.Custom["container_id"] = containerID
	metrics.Custom["container_metrics_timestamp"] = containerMetrics.Timestamp

	return nil
}

// collectAgentSpecificMetrics 에이전트별 특화 메트릭 수집
func (c *basicMetricsCollector) collectAgentSpecificMetrics(agent *models.Agent, metrics *AgentMetrics) {
	// 에이전트 타입별 특화 메트릭
	metrics.Custom["agent_type"] = string(agent.Type)
	metrics.Custom["agent_status"] = string(agent.Status)
	metrics.Custom["project_id"] = agent.ProjectID

	if agent.WorktreeID != "" {
		metrics.Custom["worktree_id"] = agent.WorktreeID
	}

	if agent.SessionID != "" {
		metrics.Custom["session_id"] = agent.SessionID
	}

	// 마지막 활동 시간
	if !agent.LastActivity.IsZero() {
		metrics.Custom["last_activity"] = agent.LastActivity
		metrics.Custom["idle_duration"] = time.Since(agent.LastActivity).Seconds()
	}

	// 에러 상태
	if agent.ErrorMessage != "" {
		metrics.Custom["has_error"] = true
		metrics.Custom["error_message"] = agent.ErrorMessage
	} else {
		metrics.Custom["has_error"] = false
	}
}

// applyRetentionPolicy 보관 정책 적용
func (c *basicMetricsCollector) applyRetentionPolicy(metrics []AgentMetrics) []AgentMetrics {
	if len(metrics) == 0 {
		return metrics
	}

	// 시간 기반 필터링
	cutoff := time.Now().Add(-c.retentionTime)
	var validMetrics []AgentMetrics

	for _, metric := range metrics {
		if metric.Timestamp.After(cutoff) {
			validMetrics = append(validMetrics, metric)
		}
	}

	// 개수 기반 제한
	if len(validMetrics) > c.maxHistorySize {
		// 오래된 것부터 제거
		startIndex := len(validMetrics) - c.maxHistorySize
		validMetrics = validMetrics[startIndex:]
	}

	return validMetrics
}

// CleanupOldMetrics 오래된 메트릭 정리 (정기적으로 호출)
func (c *basicMetricsCollector) CleanupOldMetrics() {
	c.storeMutex.Lock()
	defer c.storeMutex.Unlock()

	for agentID, metrics := range c.metricsStore {
		cleaned := c.applyRetentionPolicy(metrics)
		if len(cleaned) == 0 {
			delete(c.metricsStore, agentID)
		} else {
			c.metricsStore[agentID] = cleaned
		}
	}
}

// GetStoredAgents 메트릭이 저장된 에이전트 목록 반환
func (c *basicMetricsCollector) GetStoredAgents() []string {
	c.storeMutex.RLock()
	defer c.storeMutex.RUnlock()

	agents := make([]string, 0, len(c.metricsStore))
	for agentID := range c.metricsStore {
		agents = append(agents, agentID)
	}

	return agents
}

// GetLatestMetrics 에이전트의 최신 메트릭 반환
func (c *basicMetricsCollector) GetLatestMetrics(agentID string) (AgentMetrics, error) {
	c.storeMutex.RLock()
	defer c.storeMutex.RUnlock()

	metrics := c.metricsStore[agentID]
	if len(metrics) == 0 {
		return AgentMetrics{}, fmt.Errorf("no metrics found for agent %s", agentID)
	}

	// 가장 최신 메트릭 반환
	return metrics[len(metrics)-1], nil
}
