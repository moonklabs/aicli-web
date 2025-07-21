package claude

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// PoolMonitor는 풀 모니터링을 담당합니다
type PoolMonitor struct {
	pool       *AdvancedSessionPool
	config     MonitoringConfig
	
	// 메트릭 저장소
	sessionMetrics map[string]*SessionMetrics
	poolMetrics    *PoolMetricsHistory
	systemMetrics  *SystemMetrics
	
	// 동시성 제어
	sessionMutex sync.RWMutex
	poolMutex    sync.RWMutex
	
	// 상태 관리
	running atomic.Bool
	
	// 생명주기 관리
	ctx    context.Context
	cancel context.CancelFunc
	ticker *time.Ticker
}

// PoolMetricsHistory는 풀 메트릭 히스토리입니다
type PoolMetricsHistory struct {
	entries []PoolMetricsEntry
	maxSize int
	mutex   sync.RWMutex
}

// PoolMetricsEntry는 개별 풀 메트릭 엔트리입니다
type PoolMetricsEntry struct {
	Timestamp       time.Time     `json:"timestamp"`
	TotalSessions   int           `json:"total_sessions"`
	ActiveSessions  int           `json:"active_sessions"`
	IdleSessions    int           `json:"idle_sessions"`
	CPUUsage        float64       `json:"cpu_usage"`
	MemoryUsage     int64         `json:"memory_usage"`
	ThroughputRPS   float64       `json:"throughput_rps"`
	AverageLatency  time.Duration `json:"average_latency"`
	ErrorRate       float64       `json:"error_rate"`
	SuccessRate     float64       `json:"success_rate"`
}

// SystemMetrics는 시스템 메트릭입니다
type SystemMetrics struct {
	CPUCores      int     `json:"cpu_cores"`
	TotalMemory   int64   `json:"total_memory"`
	UsedMemory    int64   `json:"used_memory"`
	CPUUsage      float64 `json:"cpu_usage"`
	GoroutineCount int    `json:"goroutine_count"`
	GCStats       GCStats `json:"gc_stats"`
}

// GCStats는 가비지 컬렉터 통계입니다
type GCStats struct {
	NumGC        uint32        `json:"num_gc"`
	LastGCTime   time.Time     `json:"last_gc_time"`
	GCPauseTotal time.Duration `json:"gc_pause_total"`
	HeapObjects  uint64        `json:"heap_objects"`
	HeapSize     uint64        `json:"heap_size"`
}

// AlertEvent는 알람 이벤트입니다
type AlertEvent struct {
	Type        AlertType     `json:"type"`
	Severity    AlertSeverity `json:"severity"`
	Message     string        `json:"message"`
	Timestamp   time.Time     `json:"timestamp"`
	SessionID   string        `json:"session_id,omitempty"`
	Value       float64       `json:"value"`
	Threshold   float64       `json:"threshold"`
}

// AlertType은 알람 타입입니다
type AlertType int

const (
	AlertHighCPU AlertType = iota
	AlertHighMemory
	AlertHighErrorRate
	AlertLowAvailability
	AlertSessionLeaked
	AlertPoolExhausted
)

// AlertSeverity는 알람 심각도입니다
type AlertSeverity int

const (
	AlertInfo AlertSeverity = iota
	AlertWarning
	AlertError
	AlertCritical
)

// NewPoolMonitor는 새로운 풀 모니터를 생성합니다
func NewPoolMonitor(pool *AdvancedSessionPool, config MonitoringConfig) *PoolMonitor {
	ctx, cancel := context.WithCancel(context.Background())
	
	monitor := &PoolMonitor{
		pool:           pool,
		config:         config,
		sessionMetrics: make(map[string]*SessionMetrics),
		poolMetrics:    NewPoolMetricsHistory(1440), // 24시간 (1분 간격)
		systemMetrics:  &SystemMetrics{},
		ctx:            ctx,
		cancel:         cancel,
		ticker:         time.NewTicker(config.MetricsInterval),
	}
	
	return monitor
}

// NewPoolMetricsHistory는 새로운 풀 메트릭 히스토리를 생성합니다
func NewPoolMetricsHistory(maxSize int) *PoolMetricsHistory {
	return &PoolMetricsHistory{
		entries: make([]PoolMetricsEntry, 0, maxSize),
		maxSize: maxSize,
	}
}

// Start는 모니터링을 시작합니다
func (m *PoolMonitor) Start() {
	if !m.running.CompareAndSwap(false, true) {
		return // 이미 실행 중
	}
	
	go m.monitoringLoop()
}

// Stop은 모니터링을 중지합니다
func (m *PoolMonitor) Stop() {
	if !m.running.CompareAndSwap(true, false) {
		return // 이미 중지됨
	}
	
	m.cancel()
	m.ticker.Stop()
}

// SetSessionMetrics는 세션 메트릭을 설정합니다
func (m *PoolMonitor) SetSessionMetrics(sessionID string, metrics SessionMetrics) {
	m.sessionMutex.Lock()
	defer m.sessionMutex.Unlock()
	
	m.sessionMetrics[sessionID] = &metrics
}

// UpdateSessionMetrics는 세션 메트릭을 업데이트합니다
func (m *PoolMonitor) UpdateSessionMetrics(sessionID string, action string) {
	m.sessionMutex.Lock()
	defer m.sessionMutex.Unlock()
	
	metrics, exists := m.sessionMetrics[sessionID]
	if !exists {
		return
	}
	
	// 액션에 따른 메트릭 업데이트
	switch action {
	case "request":
		metrics.RequestCount++
		metrics.LastUsed = time.Now()
	case "error":
		metrics.ErrorCount++
	case "response":
		// 응답 시간 업데이트 (별도 계산 필요)
		metrics.LastUsed = time.Now()
	}
}

// GetSessionMetrics는 모든 세션 메트릭을 반환합니다
func (m *PoolMonitor) GetSessionMetrics() []SessionMetrics {
	m.sessionMutex.RLock()
	defer m.sessionMutex.RUnlock()
	
	metrics := make([]SessionMetrics, 0, len(m.sessionMetrics))
	for _, metric := range m.sessionMetrics {
		if metric != nil {
			metrics = append(metrics, *metric)
		}
	}
	
	return metrics
}

// GetCurrentCPUUsage는 현재 CPU 사용률을 반환합니다
func (m *PoolMonitor) GetCurrentCPUUsage() float64 {
	return m.systemMetrics.CPUUsage
}

// GetPoolMetricsHistory는 풀 메트릭 히스토리를 반환합니다
func (m *PoolMonitor) GetPoolMetricsHistory() []PoolMetricsEntry {
	return m.poolMetrics.GetEntries()
}

// GetSystemMetrics는 시스템 메트릭을 반환합니다
func (m *PoolMonitor) GetSystemMetrics() SystemMetrics {
	return *m.systemMetrics
}

// RemoveSessionMetrics는 세션 메트릭을 제거합니다
func (m *PoolMonitor) RemoveSessionMetrics(sessionID string) {
	m.sessionMutex.Lock()
	defer m.sessionMutex.Unlock()
	
	delete(m.sessionMetrics, sessionID)
}

// 내부 메서드들

func (m *PoolMonitor) monitoringLoop() {
	for {
		select {
		case <-m.ctx.Done():
			return
		case <-m.ticker.C:
			m.collectMetrics()
			m.checkAlerts()
		}
	}
}

func (m *PoolMonitor) collectMetrics() {
	// 시스템 메트릭 수집
	m.collectSystemMetrics()
	
	// 풀 메트릭 수집
	m.collectPoolMetrics()
	
	// 세션 메트릭 업데이트
	m.updateSessionMetrics()
}

func (m *PoolMonitor) collectSystemMetrics() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	m.systemMetrics.CPUCores = runtime.NumCPU()
	m.systemMetrics.TotalMemory = int64(memStats.Sys)
	m.systemMetrics.UsedMemory = int64(memStats.Alloc)
	m.systemMetrics.CPUUsage = m.calculateCPUUsage()
	m.systemMetrics.GoroutineCount = runtime.NumGoroutine()
	
	// GC 통계
	m.systemMetrics.GCStats = GCStats{
		NumGC:        memStats.NumGC,
		LastGCTime:   time.Unix(0, int64(memStats.LastGC)),
		GCPauseTotal: time.Duration(memStats.PauseTotalNs),
		HeapObjects:  memStats.HeapObjects,
		HeapSize:     memStats.HeapSys,
	}
}

func (m *PoolMonitor) collectPoolMetrics() {
	poolStats := m.pool.GetPoolStats()
	
	entry := PoolMetricsEntry{
		Timestamp:      time.Now(),
		TotalSessions:  poolStats.Size,
		ActiveSessions: poolStats.ActiveSessions,
		IdleSessions:   poolStats.IdleSessions,
		CPUUsage:       poolStats.CPUUsage,
		MemoryUsage:    poolStats.MemoryUsage,
		ThroughputRPS:  poolStats.ThroughputRPS,
		AverageLatency: poolStats.AverageLatency,
		ErrorRate:      poolStats.ErrorRate,
		SuccessRate:    1.0 - poolStats.ErrorRate,
	}
	
	m.poolMetrics.AddEntry(entry)
}

func (m *PoolMonitor) updateSessionMetrics() {
	m.sessionMutex.Lock()
	defer m.sessionMutex.Unlock()
	
	now := time.Now()
	
	// 각 세션의 메트릭 업데이트
	for sessionID, metrics := range m.sessionMetrics {
		if metrics == nil {
			continue
		}
		
		// 메모리 사용량 추정 (실제로는 더 정확한 측정 필요)
		metrics.MemoryUsage = m.estimateSessionMemoryUsage(sessionID)
		
		// CPU 사용량 추정
		metrics.CPUUsage = m.estimateSessionCPUUsage(sessionID)
		
		// 가중치 계산 (성능 기반)
		metrics.Weight = m.calculateSessionWeight(metrics)
		
		// 응답 시간 업데이트 (실제 구현 필요)
		metrics.ResponseTime = m.calculateSessionResponseTime(sessionID)
	}
}

func (m *PoolMonitor) checkAlerts() {
	// CPU 사용률 알람
	if m.config.AlertThresholds.HighCPUUsage > 0 && 
	   m.systemMetrics.CPUUsage > m.config.AlertThresholds.HighCPUUsage {
		m.triggerAlert(AlertEvent{
			Type:      AlertHighCPU,
			Severity:  AlertWarning,
			Message:   "High CPU usage detected",
			Timestamp: time.Now(),
			Value:     m.systemMetrics.CPUUsage,
			Threshold: m.config.AlertThresholds.HighCPUUsage,
		})
	}
	
	// 메모리 사용량 알람
	if m.config.AlertThresholds.HighMemoryUsage > 0 && 
	   m.systemMetrics.UsedMemory > m.config.AlertThresholds.HighMemoryUsage {
		m.triggerAlert(AlertEvent{
			Type:      AlertHighMemory,
			Severity:  AlertWarning,
			Message:   "High memory usage detected",
			Timestamp: time.Now(),
			Value:     float64(m.systemMetrics.UsedMemory),
			Threshold: float64(m.config.AlertThresholds.HighMemoryUsage),
		})
	}
	
	// 에러율 알람
	poolStats := m.pool.GetPoolStats()
	if m.config.AlertThresholds.HighErrorRate > 0 && 
	   poolStats.ErrorRate > m.config.AlertThresholds.HighErrorRate {
		m.triggerAlert(AlertEvent{
			Type:      AlertHighErrorRate,
			Severity:  AlertError,
			Message:   "High error rate detected",
			Timestamp: time.Now(),
			Value:     poolStats.ErrorRate,
			Threshold: m.config.AlertThresholds.HighErrorRate,
		})
	}
	
	// 가용성 알람
	availability := m.calculateAvailability()
	if m.config.AlertThresholds.LowAvailability > 0 && 
	   availability < m.config.AlertThresholds.LowAvailability {
		m.triggerAlert(AlertEvent{
			Type:      AlertLowAvailability,
			Severity:  AlertCritical,
			Message:   "Low availability detected",
			Timestamp: time.Now(),
			Value:     availability,
			Threshold: m.config.AlertThresholds.LowAvailability,
		})
	}
}

func (m *PoolMonitor) calculateCPUUsage() float64 {
	// 실제 구현에서는 더 정확한 CPU 사용률 계산 필요
	// 여기서는 고루틴 수를 기반으로 추정
	goroutines := float64(runtime.NumGoroutine())
	cores := float64(runtime.NumCPU())
	
	// 간단한 추정 (실제로는 OS별 정확한 측정 필요)
	usage := goroutines / (cores * 100.0)
	if usage > 1.0 {
		usage = 1.0
	}
	
	return usage
}

func (m *PoolMonitor) estimateSessionMemoryUsage(sessionID string) int64 {
	// 실제 구현에서는 세션별 메모리 사용량 추적 필요
	// 여기서는 대략적인 추정
	return 50 * 1024 * 1024 // 50MB per session (example)
}

func (m *PoolMonitor) estimateSessionCPUUsage(sessionID string) float64 {
	// 실제 구현에서는 세션별 CPU 사용량 추적 필요
	// 여기서는 요청 수를 기반으로 추정
	m.sessionMutex.RLock()
	metrics, exists := m.sessionMetrics[sessionID]
	m.sessionMutex.RUnlock()
	
	if !exists || metrics == nil {
		return 0.0
	}
	
	// 최근 활동 기반 CPU 사용량 추정
	timeSinceLastUse := time.Since(metrics.LastUsed)
	if timeSinceLastUse > time.Minute {
		return 0.0 // 1분 이상 비활성 세션은 CPU 미사용
	}
	
	// 요청 수를 기반으로 CPU 사용량 추정
	return math.Min(float64(metrics.RequestCount)*0.01, 0.5)
}

func (m *PoolMonitor) calculateSessionWeight(metrics *SessionMetrics) float64 {
	// 성능 기반 가중치 계산
	weight := 1.0
	
	// 에러율 기반 가중치 감소
	if metrics.RequestCount > 0 {
		errorRate := float64(metrics.ErrorCount) / float64(metrics.RequestCount)
		weight *= (1.0 - errorRate)
	}
	
	// 응답 시간 기반 가중치 조정
	if metrics.ResponseTime > 0 {
		if metrics.ResponseTime < time.Second {
			weight *= 1.2 // 빠른 응답에 보너스
		} else if metrics.ResponseTime > 5*time.Second {
			weight *= 0.8 // 느린 응답에 페널티
		}
	}
	
	// CPU 사용률 기반 가중치 조정
	if metrics.CPUUsage > 0.8 {
		weight *= 0.7 // 높은 CPU 사용률에 페널티
	} else if metrics.CPUUsage < 0.3 {
		weight *= 1.1 // 낮은 CPU 사용률에 보너스
	}
	
	return math.Max(0.1, math.Min(2.0, weight))
}

func (m *PoolMonitor) calculateSessionResponseTime(sessionID string) time.Duration {
	// 실제 구현에서는 실제 응답 시간 측정 필요
	// 여기서는 더미 값
	return 500 * time.Millisecond
}

func (m *PoolMonitor) calculateAvailability() float64 {
	poolStats := m.pool.GetPoolStats()
	
	if poolStats.Size == 0 {
		return 0.0
	}
	
	// 활성 + 유휴 세션을 가용한 것으로 간주
	available := poolStats.ActiveSessions + poolStats.IdleSessions
	return float64(available) / float64(poolStats.Size)
}

func (m *PoolMonitor) triggerAlert(alert AlertEvent) {
	// 실제 구현에서는 알람 시스템으로 전송
	// 여기서는 로그 출력
	fmt.Printf("[ALERT] %s: %s (%.2f > %.2f)\n", 
		m.getAlertTypeString(alert.Type), 
		alert.Message, 
		alert.Value, 
		alert.Threshold)
}

func (m *PoolMonitor) getAlertTypeString(alertType AlertType) string {
	switch alertType {
	case AlertHighCPU:
		return "HIGH_CPU"
	case AlertHighMemory:
		return "HIGH_MEMORY"
	case AlertHighErrorRate:
		return "HIGH_ERROR_RATE"
	case AlertLowAvailability:
		return "LOW_AVAILABILITY"
	case AlertSessionLeaked:
		return "SESSION_LEAKED"
	case AlertPoolExhausted:
		return "POOL_EXHAUSTED"
	default:
		return "UNKNOWN"
	}
}

// PoolMetricsHistory 메서드들

func (h *PoolMetricsHistory) AddEntry(entry PoolMetricsEntry) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	
	h.entries = append(h.entries, entry)
	
	// 최대 크기 제한
	if len(h.entries) > h.maxSize {
		h.entries = h.entries[1:]
	}
}

func (h *PoolMetricsHistory) GetEntries() []PoolMetricsEntry {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	
	// 복사본 반환
	entries := make([]PoolMetricsEntry, len(h.entries))
	copy(entries, h.entries)
	
	return entries
}

func (h *PoolMetricsHistory) GetEntriesSince(since time.Time) []PoolMetricsEntry {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	
	var result []PoolMetricsEntry
	for _, entry := range h.entries {
		if entry.Timestamp.After(since) {
			result = append(result, entry)
		}
	}
	
	return result
}

func (h *PoolMetricsHistory) Clear() {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	
	h.entries = h.entries[:0]
}