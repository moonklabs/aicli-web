package flow

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
	
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("component", "flow")

// FlowController 플로우 제어 관리자
type FlowController struct {
	connections   map[string]*ConnectionFlowState
	globalMetrics *GlobalFlowMetrics
	config        *FlowControlConfig
	monitor       *FlowMonitor
	throttler     *DynamicThrottler
	buffer        *AdaptiveBuffer
	prioritizer   *MessagePrioritizer
	
	memoryManager *MemoryManager
	
	mutex         sync.RWMutex
	stopCh        chan struct{}
	wg            sync.WaitGroup
	
	// 통계
	stats         *FlowStatistics
}

// ConnectionFlowState 연결별 플로우 상태
type ConnectionFlowState struct {
	ConnectionID      string
	SessionID         string
	BufferSize        int32
	MaxBufferSize     int32
	CurrentLoad       float64
	ProcessingRate    float64
	LastActivity      time.Time
	BackpressureLevel BackpressureLevel
	QualityMetrics    *ConnectionQuality
	Throttled         bool
	Priority          Priority
	
	// 통계
	MessagesSent      uint64
	MessagesDropped   uint64
	BytesProcessed    uint64
	
	// 내부 상태
	lastUpdate        time.Time
	historyWindow     []float64
	mutex             sync.RWMutex
}

// BackpressureLevel 백프레셔 레벨
type BackpressureLevel int

const (
	BackpressureNone BackpressureLevel = iota
	BackpressureLow
	BackpressureMedium
	BackpressureHigh
	BackpressureCritical
)

// Priority 메시지 우선순위
type Priority int

const (
	PriorityLow Priority = iota
	PriorityNormal
	PriorityHigh
	PriorityCritical
)

// MessageType 메시지 타입
type MessageType int

const (
	MessageTypeData MessageType = iota
	MessageTypeControl
	MessageTypeSystem
	MessageTypeError
)

// DropPolicy 메시지 드롭 정책
type DropPolicy int

const (
	DropOldest DropPolicy = iota
	DropLowestPriority
	DropBySize
	DropRandom
)

// GlobalFlowMetrics 전역 플로우 메트릭
type GlobalFlowMetrics struct {
	TotalConnections  int32
	ActiveConnections int32
	AverageLatency    time.Duration
	TotalThroughput   float64
	SystemLoad        float64
	MemoryPressure    float64
	timestamp         time.Time
	mutex             sync.RWMutex
}

// ConnectionQuality 연결 품질 메트릭
type ConnectionQuality struct {
	Latency       time.Duration
	PacketLoss    float64
	Jitter        time.Duration
	Bandwidth     float64
	LastMeasured  time.Time
}

// FlowStatistics 플로우 통계
type FlowStatistics struct {
	TotalMessages     uint64
	DroppedMessages   uint64
	ThrottledCount    uint64
	BackpressureHits  uint64
	BufferResizes     uint64
	StartTime         time.Time
}

// NewFlowController 새 플로우 컨트롤러 생성
func NewFlowController(config *FlowControlConfig) (*FlowController, error) {
	if config == nil {
		config = DefaultFlowControlConfig()
	}
	
	fc := &FlowController{
		connections:   make(map[string]*ConnectionFlowState),
		globalMetrics: &GlobalFlowMetrics{
			timestamp: time.Now(),
		},
		config:        config,
		stopCh:        make(chan struct{}),
		stats:         &FlowStatistics{StartTime: time.Now()},
		memoryManager: NewMemoryManager(config.GlobalMemoryLimit),
	}
	
	// 컴포넌트 초기화
	fc.throttler = NewDynamicThrottler(&config.Throttle)
	fc.buffer = NewAdaptiveBuffer(&config.Buffer, fc.memoryManager)
	fc.prioritizer = NewMessagePrioritizer(config.DropPolicy)
	fc.monitor = NewFlowMonitor(fc, &config.Monitor)
	
	return fc, nil
}

// Start 플로우 컨트롤러 시작
func (fc *FlowController) Start(ctx context.Context) error {
	log.Info("Starting flow controller")
	
	// 모니터 시작
	if err := fc.monitor.Start(); err != nil {
		return fmt.Errorf("failed to start monitor: %w", err)
	}
	
	// 백그라운드 작업 시작
	fc.wg.Add(1)
	go fc.backgroundWorker(ctx)
	
	log.Info("Flow controller started successfully")
	return nil
}

// Stop 플로우 컨트롤러 중지
func (fc *FlowController) Stop() error {
	log.Info("Stopping flow controller")
	
	close(fc.stopCh)
	fc.wg.Wait()
	
	if err := fc.monitor.Stop(); err != nil {
		log.Warnf("Error stopping monitor: %v", err)
	}
	
	log.Info("Flow controller stopped")
	return nil
}

// RegisterConnection 새 연결 등록
func (fc *FlowController) RegisterConnection(connectionID, sessionID string) error {
	fc.mutex.Lock()
	defer fc.mutex.Unlock()
	
	if _, exists := fc.connections[connectionID]; exists {
		return fmt.Errorf("connection already registered: %s", connectionID)
	}
	
	flowState := &ConnectionFlowState{
		ConnectionID:   connectionID,
		SessionID:      sessionID,
		BufferSize:     int32(fc.config.Buffer.InitialSize),
		MaxBufferSize:  int32(fc.config.Buffer.MaxSize),
		LastActivity:   time.Now(),
		Priority:       PriorityNormal,
		QualityMetrics: &ConnectionQuality{},
		lastUpdate:     time.Now(),
		historyWindow:  make([]float64, 0, fc.config.Buffer.UtilizationWindow),
	}
	
	fc.connections[connectionID] = flowState
	atomic.AddInt32(&fc.globalMetrics.ActiveConnections, 1)
	atomic.AddInt32(&fc.globalMetrics.TotalConnections, 1)
	
	log.Infof("Connection registered: %s", connectionID)
	return nil
}

// UnregisterConnection 연결 등록 해제
func (fc *FlowController) UnregisterConnection(connectionID string) error {
	fc.mutex.Lock()
	defer fc.mutex.Unlock()
	
	if _, exists := fc.connections[connectionID]; !exists {
		return fmt.Errorf("connection not found: %s", connectionID)
	}
	
	delete(fc.connections, connectionID)
	atomic.AddInt32(&fc.globalMetrics.ActiveConnections, -1)
	
	// 관련 리소스 정리
	fc.throttler.RemoveConnection(connectionID)
	fc.buffer.RemoveBuffer(connectionID)
	
	log.Infof("Connection unregistered: %s", connectionID)
	return nil
}

// ProcessMessage 메시지 처리 및 플로우 제어 적용
func (fc *FlowController) ProcessMessage(connectionID string, data []byte, priority Priority) error {
	fc.mutex.RLock()
	flowState, exists := fc.connections[connectionID]
	fc.mutex.RUnlock()
	
	if !exists {
		return fmt.Errorf("connection not found: %s", connectionID)
	}
	
	// 백프레셔 감지
	level, err := fc.DetectBackpressure(connectionID)
	if err != nil {
		return fmt.Errorf("failed to detect backpressure: %w", err)
	}
	
	// 스로틀링 적용
	if err := fc.throttler.ApplyThrottle(connectionID, level); err != nil {
		return fmt.Errorf("failed to apply throttle: %w", err)
	}
	
	// 메시지 우선순위 처리
	message := PriorityMessage{
		Data:        data,
		Priority:    priority,
		Timestamp:   time.Now(),
		SessionID:   flowState.SessionID,
		MessageType: MessageTypeData,
	}
	
	// 오버플로우 처리
	if level >= BackpressureHigh {
		if err := fc.prioritizer.HandleOverflow(connectionID, message); err != nil {
			atomic.AddUint64(&flowState.MessagesDropped, 1)
			atomic.AddUint64(&fc.stats.DroppedMessages, 1)
			return fmt.Errorf("message dropped due to overflow: %w", err)
		}
	}
	
	// 버퍼 크기 조정
	metrics := &BackpressureMetrics{
		BufferUtilization: float64(flowState.BufferSize) / float64(flowState.MaxBufferSize),
		ProcessingRate:    flowState.ProcessingRate,
	}
	
	if err := fc.buffer.AdjustBufferSize(connectionID, metrics); err != nil {
		log.Warnf("Failed to adjust buffer size: %v", err)
	}
	
	// 통계 업데이트
	atomic.AddUint64(&flowState.MessagesSent, 1)
	atomic.AddUint64(&flowState.BytesProcessed, uint64(len(data)))
	atomic.AddUint64(&fc.stats.TotalMessages, 1)
	
	flowState.LastActivity = time.Now()
	
	return nil
}

// DetectBackpressure 백프레셔 감지
func (fc *FlowController) DetectBackpressure(connectionID string) (BackpressureLevel, error) {
	fc.mutex.RLock()
	flowState, exists := fc.connections[connectionID]
	fc.mutex.RUnlock()
	
	if !exists {
		return BackpressureNone, fmt.Errorf("connection not found: %s", connectionID)
	}
	
	// 버퍼 사용률 확인
	bufferUtilization := float64(atomic.LoadInt32(&flowState.BufferSize)) / 
	                    float64(atomic.LoadInt32(&flowState.MaxBufferSize))
	
	// 처리 속도 확인
	processingRate := fc.calculateProcessingRate(flowState)
	
	// 지연 시간 확인
	latency := fc.calculateLatency(flowState)
	
	// 메모리 압박 확인
	fc.globalMetrics.mutex.RLock()
	memoryPressure := fc.globalMetrics.MemoryPressure
	fc.globalMetrics.mutex.RUnlock()
	
	// 백프레셔 레벨 결정
	level := fc.calculateBackpressureLevel(bufferUtilization, processingRate, latency, memoryPressure)
	
	// 상태 업데이트
	flowState.mutex.Lock()
	flowState.BackpressureLevel = level
	flowState.LastActivity = time.Now()
	flowState.mutex.Unlock()
	
	if level > BackpressureNone {
		atomic.AddUint64(&fc.stats.BackpressureHits, 1)
	}
	
	return level, nil
}

// calculateProcessingRate 처리 속도 계산
func (fc *FlowController) calculateProcessingRate(state *ConnectionFlowState) float64 {
	state.mutex.RLock()
	defer state.mutex.RUnlock()
	
	elapsed := time.Since(state.lastUpdate).Seconds()
	if elapsed <= 0 {
		return 0
	}
	
	rate := float64(state.BytesProcessed) / elapsed
	return rate
}

// calculateLatency 지연 시간 계산
func (fc *FlowController) calculateLatency(state *ConnectionFlowState) time.Duration {
	state.mutex.RLock()
	defer state.mutex.RUnlock()
	
	if state.QualityMetrics != nil {
		return state.QualityMetrics.Latency
	}
	
	return 0
}

// calculateBackpressureLevel 백프레셔 레벨 계산
func (fc *FlowController) calculateBackpressureLevel(
	bufferUtil, processingRate float64,
	latency time.Duration,
	memoryPressure float64) BackpressureLevel {
	
	thresholds := fc.config.Thresholds
	
	// 임계 상황 확인
	if bufferUtil >= thresholds.BufferUtilizationHigh ||
		memoryPressure >= thresholds.MemoryPressureMax ||
		latency.Milliseconds() > thresholds.LatencyThresholdMs {
		return BackpressureCritical
	}
	
	// 높은 백프레셔
	if bufferUtil >= thresholds.BufferUtilizationMedium ||
		processingRate < thresholds.ProcessingRateMin*0.5 {
		return BackpressureHigh
	}
	
	// 중간 백프레셔
	if bufferUtil >= thresholds.BufferUtilizationLow ||
		processingRate < thresholds.ProcessingRateMin*0.8 {
		return BackpressureMedium
	}
	
	// 낮은 백프레셔
	if bufferUtil >= 0.4 || processingRate < thresholds.ProcessingRateMin {
		return BackpressureLow
	}
	
	return BackpressureNone
}

// GetConnectionState 연결 상태 조회
func (fc *FlowController) GetConnectionState(connectionID string) (*ConnectionFlowState, error) {
	fc.mutex.RLock()
	defer fc.mutex.RUnlock()
	
	state, exists := fc.connections[connectionID]
	if !exists {
		return nil, fmt.Errorf("connection not found: %s", connectionID)
	}
	
	return state, nil
}

// GetGlobalMetrics 전역 메트릭 조회
func (fc *FlowController) GetGlobalMetrics() *GlobalFlowMetrics {
	return fc.globalMetrics
}

// GetStatistics 통계 조회
func (fc *FlowController) GetStatistics() *FlowStatistics {
	return fc.stats
}

// backgroundWorker 백그라운드 작업 처리
func (fc *FlowController) backgroundWorker(ctx context.Context) {
	defer fc.wg.Done()
	
	ticker := time.NewTicker(fc.config.Monitor.MonitorInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-fc.stopCh:
			return
		case <-ticker.C:
			fc.updateGlobalMetrics()
		}
	}
}

// updateGlobalMetrics 전역 메트릭 업데이트
func (fc *FlowController) updateGlobalMetrics() {
	fc.globalMetrics.mutex.Lock()
	defer fc.globalMetrics.mutex.Unlock()
	
	// 시스템 부하 계산
	fc.globalMetrics.SystemLoad = fc.calculateSystemLoad()
	
	// 메모리 압박 계산
	fc.globalMetrics.MemoryPressure = fc.memoryManager.GetMemoryPressure()
	
	// 평균 처리량 계산
	fc.globalMetrics.TotalThroughput = fc.calculateTotalThroughput()
	
	fc.globalMetrics.timestamp = time.Now()
}

// calculateSystemLoad 시스템 부하 계산
func (fc *FlowController) calculateSystemLoad() float64 {
	// 실제 구현에서는 시스템 메트릭을 수집
	// 여기서는 간단한 예시
	activeConns := atomic.LoadInt32(&fc.globalMetrics.ActiveConnections)
	maxConns := int32(1000) // 최대 연결 수
	
	if maxConns == 0 {
		return 0
	}
	
	return float64(activeConns) / float64(maxConns)
}

// calculateTotalThroughput 전체 처리량 계산
func (fc *FlowController) calculateTotalThroughput() float64 {
	fc.mutex.RLock()
	defer fc.mutex.RUnlock()
	
	var totalThroughput float64
	for _, state := range fc.connections {
		totalThroughput += state.ProcessingRate
	}
	
	return totalThroughput
}

// String 백프레셔 레벨 문자열 변환
func (b BackpressureLevel) String() string {
	switch b {
	case BackpressureNone:
		return "None"
	case BackpressureLow:
		return "Low"
	case BackpressureMedium:
		return "Medium"
	case BackpressureHigh:
		return "High"
	case BackpressureCritical:
		return "Critical"
	default:
		return "Unknown"
	}
}