package flow

import (
	"context"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

// FlowMonitor 플로우 제어 모니터
type FlowMonitor struct {
	controller *FlowController
	metrics    *FlowMetrics
	alerts     *AlertManager
	config     *MonitorConfig
	
	// 실행 제어
	ticker     *time.Ticker
	stopCh     chan struct{}
	wg         sync.WaitGroup
	
	// 메트릭 히스토리
	metricsHistory []FlowMetrics
	historyMutex   sync.RWMutex
}

// FlowMetrics 플로우 메트릭
type FlowMetrics struct {
	ConnectionCount       int
	AverageBackpressure   float64
	ThrottledConnections  int
	DroppedMessages       uint64
	BufferUtilization     float64
	SystemThroughput      float64
	MemoryUsage           int64
	CPUUsage              float64
	Timestamp             time.Time
}

// AlertManager 경고 관리자
type AlertManager struct {
	alerts       []Alert
	thresholds   AlertThresholds
	handlers     map[AlertLevel][]AlertHandler
	mutex        sync.RWMutex
}

// Alert 경고
type Alert struct {
	ID          string
	Level       AlertLevel
	Type        AlertType
	Message     string
	Details     map[string]interface{}
	Timestamp   time.Time
	Resolved    bool
	ResolvedAt  time.Time
}

// AlertLevel 경고 레벨
type AlertLevel int

const (
	AlertLevelInfo AlertLevel = iota
	AlertLevelWarning
	AlertLevelError
	AlertLevelCritical
)

// AlertType 경고 타입
type AlertType int

const (
	AlertTypeBackpressure AlertType = iota
	AlertTypeMemory
	AlertTypeThroughput
	AlertTypeDropRate
	AlertTypeSystemLoad
)

// AlertThresholds 경고 임계값
type AlertThresholds struct {
	BackpressureWarning  float64
	BackpressureCritical float64
	MemoryWarning        float64
	MemoryCritical       float64
	DropRateWarning      float64
	DropRateCritical     float64
	ThroughputMin        float64
}

// AlertHandler 경고 처리기 인터페이스
type AlertHandler interface {
	HandleAlert(alert Alert) error
}

// NewFlowMonitor 새 플로우 모니터 생성
func NewFlowMonitor(controller *FlowController, config *MonitorConfig) *FlowMonitor {
	fm := &FlowMonitor{
		controller:     controller,
		config:         config,
		stopCh:         make(chan struct{}),
		metricsHistory: make([]FlowMetrics, 0, config.MetricsWindow),
	}
	
	// 경고 관리자 초기화
	fm.alerts = NewAlertManager()
	
	return fm
}

// NewAlertManager 새 경고 관리자 생성
func NewAlertManager() *AlertManager {
	return &AlertManager{
		alerts:   make([]Alert, 0),
		handlers: make(map[AlertLevel][]AlertHandler),
		thresholds: AlertThresholds{
			BackpressureWarning:  2.5, // 평균 백프레셔 레벨
			BackpressureCritical: 3.5,
			MemoryWarning:        0.7,
			MemoryCritical:       0.9,
			DropRateWarning:      0.01, // 1%
			DropRateCritical:     0.05, // 5%
			ThroughputMin:        1000, // bytes/sec
		},
	}
}

// Start 모니터링 시작
func (fm *FlowMonitor) Start() error {
	if fm.ticker != nil {
		return fmt.Errorf("monitor already started")
	}
	
	fm.ticker = time.NewTicker(fm.config.MonitorInterval)
	
	fm.wg.Add(1)
	go fm.monitorLoop()
	
	// 통계 리포터 시작
	if fm.config.StatsReportInterval > 0 {
		fm.wg.Add(1)
		go fm.statsReporter()
	}
	
	log.Info("Flow monitor started")
	return nil
}

// Stop 모니터링 중지
func (fm *FlowMonitor) Stop() error {
	if fm.ticker == nil {
		return fmt.Errorf("monitor not started")
	}
	
	fm.ticker.Stop()
	close(fm.stopCh)
	fm.wg.Wait()
	
	log.Info("Flow monitor stopped")
	return nil
}

// monitorLoop 모니터링 루프
func (fm *FlowMonitor) monitorLoop() {
	defer fm.wg.Done()
	
	for {
		select {
		case <-fm.ticker.C:
			if err := fm.collectMetrics(); err != nil {
				log.Errorf("Failed to collect flow metrics: %v", err)
			}
			
			if err := fm.checkAlerts(); err != nil {
				log.Errorf("Failed to check flow alerts: %v", err)
			}
			
		case <-fm.stopCh:
			return
		}
	}
}

// collectMetrics 메트릭 수집
func (fm *FlowMonitor) collectMetrics() error {
	metrics := &FlowMetrics{
		Timestamp: time.Now(),
	}
	
	fm.controller.mutex.RLock()
	defer fm.controller.mutex.RUnlock()
	
	metrics.ConnectionCount = len(fm.controller.connections)
	
	var totalBackpressure float64
	var throttledCount int
	var totalBufferUtil float64
	
	for _, flowState := range fm.controller.connections {
		// 백프레셔 레벨 수치화
		backpressureValue := float64(flowState.BackpressureLevel)
		totalBackpressure += backpressureValue
		
		if flowState.Throttled {
			throttledCount++
		}
		
		bufferUtil := float64(flowState.BufferSize) / float64(flowState.MaxBufferSize)
		totalBufferUtil += bufferUtil
	}
	
	if metrics.ConnectionCount > 0 {
		metrics.AverageBackpressure = totalBackpressure / float64(metrics.ConnectionCount)
		metrics.BufferUtilization = totalBufferUtil / float64(metrics.ConnectionCount)
	}
	
	metrics.ThrottledConnections = throttledCount
	metrics.SystemThroughput = fm.controller.globalMetrics.TotalThroughput
	
	// 메모리 사용량
	if fm.controller.buffer != nil {
		metrics.MemoryUsage = atomic.LoadInt64(&fm.controller.buffer.totalMemory)
	}
	
	// 드롭된 메시지
	if fm.controller.stats != nil {
		metrics.DroppedMessages = atomic.LoadUint64(&fm.controller.stats.DroppedMessages)
	}
	
	// CPU 사용률 (실제 구현에서는 시스템 메트릭 수집)
	metrics.CPUUsage = fm.estimateCPUUsage()
	
	fm.metrics = metrics
	
	// 히스토리 업데이트
	fm.updateMetricsHistory(*metrics)
	
	return nil
}

// checkAlerts 경고 확인
func (fm *FlowMonitor) checkAlerts() error {
	if fm.metrics == nil {
		return nil
	}
	
	// 백프레셔 경고
	if fm.metrics.AverageBackpressure > fm.alerts.thresholds.BackpressureCritical {
		fm.alerts.CreateAlert(AlertLevelCritical, AlertTypeBackpressure,
			fmt.Sprintf("Critical backpressure level: %.2f", fm.metrics.AverageBackpressure),
			map[string]interface{}{
				"average_backpressure": fm.metrics.AverageBackpressure,
				"connections":          fm.metrics.ConnectionCount,
			})
	} else if fm.metrics.AverageBackpressure > fm.alerts.thresholds.BackpressureWarning {
		fm.alerts.CreateAlert(AlertLevelWarning, AlertTypeBackpressure,
			fmt.Sprintf("High backpressure level: %.2f", fm.metrics.AverageBackpressure),
			map[string]interface{}{
				"average_backpressure": fm.metrics.AverageBackpressure,
				"connections":          fm.metrics.ConnectionCount,
			})
	}
	
	// 메모리 경고
	memoryPressure := fm.controller.memoryManager.GetMemoryPressure()
	if memoryPressure > fm.alerts.thresholds.MemoryCritical {
		fm.alerts.CreateAlert(AlertLevelCritical, AlertTypeMemory,
			fmt.Sprintf("Critical memory pressure: %.2f%%", memoryPressure*100),
			map[string]interface{}{
				"memory_pressure": memoryPressure,
				"memory_usage":    fm.metrics.MemoryUsage,
			})
	} else if memoryPressure > fm.alerts.thresholds.MemoryWarning {
		fm.alerts.CreateAlert(AlertLevelWarning, AlertTypeMemory,
			fmt.Sprintf("High memory pressure: %.2f%%", memoryPressure*100),
			map[string]interface{}{
				"memory_pressure": memoryPressure,
				"memory_usage":    fm.metrics.MemoryUsage,
			})
	}
	
	// 드롭률 경고
	dropRate := fm.calculateDropRate()
	if dropRate > fm.alerts.thresholds.DropRateCritical {
		fm.alerts.CreateAlert(AlertLevelCritical, AlertTypeDropRate,
			fmt.Sprintf("Critical message drop rate: %.2f%%", dropRate*100),
			map[string]interface{}{
				"drop_rate":        dropRate,
				"dropped_messages": fm.metrics.DroppedMessages,
			})
	} else if dropRate > fm.alerts.thresholds.DropRateWarning {
		fm.alerts.CreateAlert(AlertLevelWarning, AlertTypeDropRate,
			fmt.Sprintf("High message drop rate: %.2f%%", dropRate*100),
			map[string]interface{}{
				"drop_rate":        dropRate,
				"dropped_messages": fm.metrics.DroppedMessages,
			})
	}
	
	// 처리량 경고
	if fm.metrics.SystemThroughput < fm.alerts.thresholds.ThroughputMin {
		fm.alerts.CreateAlert(AlertLevelWarning, AlertTypeThroughput,
			fmt.Sprintf("Low system throughput: %.2f bytes/sec", fm.metrics.SystemThroughput),
			map[string]interface{}{
				"throughput": fm.metrics.SystemThroughput,
				"minimum":    fm.alerts.thresholds.ThroughputMin,
			})
	}
	
	return nil
}

// updateMetricsHistory 메트릭 히스토리 업데이트
func (fm *FlowMonitor) updateMetricsHistory(metrics FlowMetrics) {
	fm.historyMutex.Lock()
	defer fm.historyMutex.Unlock()
	
	fm.metricsHistory = append(fm.metricsHistory, metrics)
	
	// 윈도우 크기 유지
	if len(fm.metricsHistory) > fm.config.MetricsWindow {
		fm.metricsHistory = fm.metricsHistory[1:]
	}
}

// GetMetricsHistory 메트릭 히스토리 조회
func (fm *FlowMonitor) GetMetricsHistory() []FlowMetrics {
	fm.historyMutex.RLock()
	defer fm.historyMutex.RUnlock()
	
	history := make([]FlowMetrics, len(fm.metricsHistory))
	copy(history, fm.metricsHistory)
	return history
}

// GetCurrentMetrics 현재 메트릭 조회
func (fm *FlowMonitor) GetCurrentMetrics() *FlowMetrics {
	return fm.metrics
}

// calculateDropRate 드롭률 계산
func (fm *FlowMonitor) calculateDropRate() float64 {
	totalMessages := atomic.LoadUint64(&fm.controller.stats.TotalMessages)
	droppedMessages := atomic.LoadUint64(&fm.controller.stats.DroppedMessages)
	
	if totalMessages == 0 {
		return 0
	}
	
	return float64(droppedMessages) / float64(totalMessages)
}

// estimateCPUUsage CPU 사용률 추정
func (fm *FlowMonitor) estimateCPUUsage() float64 {
	// 실제 구현에서는 시스템 메트릭 수집
	// 여기서는 간단한 추정
	
	activeConnections := fm.metrics.ConnectionCount
	throttledConnections := fm.metrics.ThrottledConnections
	
	// 연결당 예상 CPU 사용률
	baseUsagePerConnection := 0.01
	throttledUsagePerConnection := 0.005
	
	normalConnections := activeConnections - throttledConnections
	usage := float64(normalConnections)*baseUsagePerConnection +
		float64(throttledConnections)*throttledUsagePerConnection
	
	return math.Min(usage, 1.0)
}

// statsReporter 통계 리포터
func (fm *FlowMonitor) statsReporter() {
	defer fm.wg.Done()
	
	ticker := time.NewTicker(fm.config.StatsReportInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			fm.reportStatistics()
			
		case <-fm.stopCh:
			return
		}
	}
}

// reportStatistics 통계 보고
func (fm *FlowMonitor) reportStatistics() {
	if fm.metrics == nil {
		return
	}
	
	stats := fm.controller.GetStatistics()
	globalMetrics := fm.controller.GetGlobalMetrics()
	
	log.WithFields(map[string]interface{}{
		"connections":         fm.metrics.ConnectionCount,
		"active_connections":  atomic.LoadInt32(&globalMetrics.ActiveConnections),
		"avg_backpressure":    fm.metrics.AverageBackpressure,
		"throttled":           fm.metrics.ThrottledConnections,
		"dropped_messages":    atomic.LoadUint64(&stats.DroppedMessages),
		"total_messages":      atomic.LoadUint64(&stats.TotalMessages),
		"buffer_utilization":  fm.metrics.BufferUtilization,
		"system_throughput":   fm.metrics.SystemThroughput,
		"memory_usage":        fm.metrics.MemoryUsage,
		"cpu_usage":           fm.metrics.CPUUsage,
		"backpressure_hits":   atomic.LoadUint64(&stats.BackpressureHits),
		"throttled_count":     atomic.LoadUint64(&stats.ThrottledCount),
		"buffer_resizes":      atomic.LoadUint64(&stats.BufferResizes),
		"uptime":              time.Since(stats.StartTime).String(),
	}).Info("Flow control statistics")
}

// AlertManager methods

// CreateAlert 경고 생성
func (am *AlertManager) CreateAlert(level AlertLevel, alertType AlertType, message string, details map[string]interface{}) {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	
	alert := Alert{
		ID:        fmt.Sprintf("%s-%d-%d", time.Now().Format("20060102150405"), level, alertType),
		Level:     level,
		Type:      alertType,
		Message:   message,
		Details:   details,
		Timestamp: time.Now(),
		Resolved:  false,
	}
	
	// 중복 경고 확인
	for i, existing := range am.alerts {
		if existing.Type == alertType && existing.Level == level && !existing.Resolved {
			// 기존 경고 업데이트
			am.alerts[i] = alert
			am.processAlert(alert)
			return
		}
	}
	
	// 새 경고 추가
	am.alerts = append(am.alerts, alert)
	am.processAlert(alert)
	
	// 경고 로깅
	switch level {
	case AlertLevelCritical:
		log.Errorf("CRITICAL ALERT: %s", message)
	case AlertLevelError:
		log.Errorf("ERROR ALERT: %s", message)
	case AlertLevelWarning:
		log.Warnf("WARNING ALERT: %s", message)
	case AlertLevelInfo:
		log.Infof("INFO ALERT: %s", message)
	}
}

// processAlert 경고 처리
func (am *AlertManager) processAlert(alert Alert) {
	handlers, exists := am.handlers[alert.Level]
	if !exists {
		return
	}
	
	for _, handler := range handlers {
		if err := handler.HandleAlert(alert); err != nil {
			log.Errorf("Failed to handle alert: %v", err)
		}
	}
}

// ResolveAlert 경고 해결
func (am *AlertManager) ResolveAlert(alertID string) {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	
	for i, alert := range am.alerts {
		if alert.ID == alertID && !alert.Resolved {
			am.alerts[i].Resolved = true
			am.alerts[i].ResolvedAt = time.Now()
			log.Infof("Alert resolved: %s", alertID)
			break
		}
	}
}

// GetActiveAlerts 활성 경고 조회
func (am *AlertManager) GetActiveAlerts() []Alert {
	am.mutex.RLock()
	defer am.mutex.RUnlock()
	
	activeAlerts := make([]Alert, 0)
	for _, alert := range am.alerts {
		if !alert.Resolved {
			activeAlerts = append(activeAlerts, alert)
		}
	}
	
	return activeAlerts
}

// RegisterHandler 경고 처리기 등록
func (am *AlertManager) RegisterHandler(level AlertLevel, handler AlertHandler) {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	
	am.handlers[level] = append(am.handlers[level], handler)
}

// String 경고 레벨 문자열 변환
func (l AlertLevel) String() string {
	switch l {
	case AlertLevelInfo:
		return "INFO"
	case AlertLevelWarning:
		return "WARNING"
	case AlertLevelError:
		return "ERROR"
	case AlertLevelCritical:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}

// String 경고 타입 문자열 변환
func (t AlertType) String() string {
	switch t {
	case AlertTypeBackpressure:
		return "Backpressure"
	case AlertTypeMemory:
		return "Memory"
	case AlertTypeThroughput:
		return "Throughput"
	case AlertTypeDropRate:
		return "DropRate"
	case AlertTypeSystemLoad:
		return "SystemLoad"
	default:
		return "Unknown"
	}
}