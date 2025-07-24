package claude

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// ActionRecord는 액션 기록입니다
type ActionRecord struct {
	Action    string
	Timestamp time.Time
	Success   bool
	Duration  time.Duration
}

// PoolMetrics 확장 필드
type poolMetricsExt struct {
	// 성능 통계 (실시간 계산)
	throughputRPS   atomic.Value // float64
	averageLatency  atomic.Value // time.Duration
	errorRate       atomic.Value // float64
	
	// 액션 추적
	actions         []ActionRecord
	actionMutex     sync.RWMutex
}

var metricsExtMap = sync.Map{} // *PoolMetrics -> *poolMetricsExt

// getOrCreateExt는 PoolMetrics에 대한 확장 데이터를 가져오거나 생성합니다
func getOrCreateExt(m *PoolMetrics) *poolMetricsExt {
	if ext, ok := metricsExtMap.Load(m); ok {
		return ext.(*poolMetricsExt)
	}
	
	ext := &poolMetricsExt{
		actions: make([]ActionRecord, 0, 1000),
	}
	metricsExtMap.Store(m, ext)
	return ext
}

// GetThroughputRPS는 초당 처리량을 반환합니다
func (m *PoolMetrics) GetThroughputRPS() float64 {
	ext := getOrCreateExt(m)
	if v := ext.throughputRPS.Load(); v != nil {
		return v.(float64)
	}
	return 0.0
}

// GetAverageLatency는 평균 지연시간을 반환합니다
func (m *PoolMetrics) GetAverageLatency() time.Duration {
	ext := getOrCreateExt(m)
	if v := ext.averageLatency.Load(); v != nil {
		return v.(time.Duration)
	}
	return 0
}

// GetErrorRate는 에러율을 반환합니다
func (m *PoolMetrics) GetErrorRate() float64 {
	ext := getOrCreateExt(m)
	if v := ext.errorRate.Load(); v != nil {
		return v.(float64)
	}
	return 0.0
}

// RecordAction은 액션을 기록합니다
func (m *PoolMetrics) RecordAction(action string, success bool, duration time.Duration) {
	ext := getOrCreateExt(m)
	ext.actionMutex.Lock()
	defer ext.actionMutex.Unlock()
	
	ext.actions = append(ext.actions, ActionRecord{
		Action:    action,
		Timestamp: time.Now(),
		Success:   success,
		Duration:  duration,
	})
	
	// 최대 1000개의 액션만 유지
	if len(ext.actions) > 1000 {
		ext.actions = ext.actions[1:]
	}
	
	// 통계 업데이트
	updateStats(ext)
}

// updateStats는 실시간 통계를 업데이트합니다
func updateStats(ext *poolMetricsExt) {
	// 최근 1분간의 데이터로 계산
	cutoff := time.Now().Add(-time.Minute)
	var recentActions []ActionRecord
	for _, action := range ext.actions {
		if action.Timestamp.After(cutoff) {
			recentActions = append(recentActions, action)
		}
	}
	
	if len(recentActions) == 0 {
		ext.throughputRPS.Store(0.0)
		ext.averageLatency.Store(time.Duration(0))
		ext.errorRate.Store(0.0)
		return
	}
	
	// 처리량 계산
	duration := time.Since(recentActions[0].Timestamp)
	if duration > 0 {
		rps := float64(len(recentActions)) / duration.Seconds()
		ext.throughputRPS.Store(rps)
	}
	
	// 평균 지연시간 계산
	var totalLatency time.Duration
	var errorCount int
	for _, action := range recentActions {
		totalLatency += action.Duration
		if !action.Success {
			errorCount++
		}
	}
	avgLatency := totalLatency / time.Duration(len(recentActions))
	ext.averageLatency.Store(avgLatency)
	
	// 에러율 계산
	errorRate := float64(errorCount) / float64(len(recentActions))
	ext.errorRate.Store(errorRate)
}

// Start는 메트릭 수집을 시작합니다
func (m *PoolMetrics) Start(ctx context.Context) {
	ext := getOrCreateExt(m)
	
	// 주기적으로 통계 업데이트
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				ext.actionMutex.RLock()
				updateStats(ext)
				ext.actionMutex.RUnlock()
			}
		}
	}()
}