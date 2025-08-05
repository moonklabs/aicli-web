---
task_id: T06_S02_Flow_Control
sprint_id: S02_M06_PTY_Streaming
milestone_id: M06
title: 백프레셔 및 플로우 컨트롤 구현
type: implementation
complexity: Medium
status: pending
assignee: unassigned
created: 2025-08-05T10:00:00+0900
last_updated: 2025-08-05T10:00:00+0900
depends_on: [T02_S02_WebSocket_Streaming]
blocks: [T07_S02_Performance_Optimization]
epic: PTY_Streaming_System
---

# Task: 백프레셔 및 플로우 컨트롤 구현

## Task Summary
PTY 스트리밍 시스템에서 데이터 흐름을 제어하고 백프레셔(backpressure)를 처리하는 시스템을 구현합니다. 클라이언트 처리 속도와 서버 생성 속도의 불균형을 해결하여 안정적인 실시간 통신을 보장합니다.

## Acceptance Criteria

### 기능 요구사항
- [ ] 동적 백프레셔 감지 및 대응 시스템
- [ ] 클라이언트별 처리 속도 모니터링
- [ ] 적응형 버퍼 크기 조절 메커니즘
- [ ] 우선순위 기반 메시지 드롭 정책
- [ ] 네트워크 대역폭 기반 플로우 제어
- [ ] 클라이언트 연결 품질 자동 조정
- [ ] 서버 부하 기반 동적 스로틀링

### 성능 요구사항
- [ ] 플로우 제어 오버헤드 < 5%
- [ ] 백프레셔 감지 시간 < 100ms
- [ ] 버퍼 조정 응답 시간 < 200ms
- [ ] 메모리 사용량 증가 < 20%
- [ ] CPU 사용률 증가 < 10%

### 안정성 요구사항
- [ ] 메모리 고갈 방지
- [ ] 느린 클라이언트로 인한 전체 시스템 영향 차단
- [ ] 플로우 제어 실패 시 우아한 성능 저하
- [ ] 네트워크 단절 시 리소스 즉시 해제

## Implementation Details

### 1. 플로우 제어 관리자 구조

```go
// internal/flow/flow_controller.go
type FlowController struct {
    connections     map[string]*ConnectionFlowState
    globalMetrics   *GlobalFlowMetrics
    config          *FlowControlConfig
    monitor         *FlowMonitor
    throttler       *DynamicThrottler
    mutex           sync.RWMutex
    stopCh          chan struct{}
}

type ConnectionFlowState struct {
    ConnectionID    string
    SessionID       string
    BufferSize      int
    MaxBufferSize   int
    CurrentLoad     float64
    ProcessingRate  float64
    LastActivity    time.Time
    BackpressureLevel BackpressureLevel
    QualityMetrics  *ConnectionQuality
    Throttled       bool
    Priority        Priority
}

type BackpressureLevel int
const (
    BackpressureNone BackpressureLevel = iota
    BackpressureLow
    BackpressureMedium
    BackpressureHigh
    BackpressureCritical
)

type Priority int
const (
    PriorityLow Priority = iota
    PriorityNormal
    PriorityHigh
    PriorityCritical
)

type GlobalFlowMetrics struct {
    TotalConnections    int
    ActiveConnections   int
    AverageLatency      time.Duration
    TotalThroughput     float64
    SystemLoad          float64
    MemoryPressure      float64
}
```

### 2. 백프레셔 감지 시스템

```go
// 백프레셔 감지 인터페이스
type BackpressureDetector interface {
    DetectBackpressure(connectionID string) (BackpressureLevel, error)
    MonitorConnection(connectionID string) error
    SetThresholds(config *BackpressureThresholds) error
    GetMetrics(connectionID string) (*BackpressureMetrics, error)
}

type BackpressureThresholds struct {
    BufferUtilizationLow    float64 // 0.6
    BufferUtilizationMedium float64 // 0.75
    BufferUtilizationHigh   float64 // 0.9
    ProcessingRateMin       float64 // bytes/sec
    LatencyThresholdMs      int64   // ms
    MemoryPressureMax       float64 // 0.8
}

type BackpressureMetrics struct {
    BufferUtilization   float64
    ProcessingRate      float64
    AverageLatency      time.Duration
    QueueDepth          int
    DroppedMessages     uint64
    LastBackpressure    time.Time
}

// 백프레셔 감지 구현
func (fc *FlowController) DetectBackpressure(connectionID string) (BackpressureLevel, error) {
    fc.mutex.RLock()
    flowState, exists := fc.connections[connectionID]
    fc.mutex.RUnlock()
    
    if !exists {
        return BackpressureNone, fmt.Errorf("connection not found: %s", connectionID)
    }
    
    // 버퍼 사용률 확인
    bufferUtilization := float64(flowState.BufferSize) / float64(flowState.MaxBufferSize)
    
    // 처리 속도 확인
    processingRate := fc.calculateProcessingRate(flowState)
    
    // 지연 시간 확인
    latency := fc.calculateLatency(flowState)
    
    // 메모리 압박 확인
    memoryPressure := fc.globalMetrics.MemoryPressure
    
    // 백프레셔 레벨 결정
    level := fc.calculateBackpressureLevel(bufferUtilization, processingRate, latency, memoryPressure)
    
    // 상태 업데이트
    flowState.BackpressureLevel = level
    flowState.LastActivity = time.Now()
    
    return level, nil
}

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
       processingRate < thresholds.ProcessingRateMin * 0.5 {
        return BackpressureHigh
    }
    
    // 중간 백프레셔
    if bufferUtil >= thresholds.BufferUtilizationLow ||
       processingRate < thresholds.ProcessingRateMin * 0.8 {
        return BackpressureMedium
    }
    
    // 낮은 백프레셔
    if bufferUtil >= 0.4 || processingRate < thresholds.ProcessingRateMin {
        return BackpressureLow
    }
    
    return BackpressureNone
}
```

### 3. 동적 스로틀링 시스템

```go
// 동적 스로틀링 관리
type DynamicThrottler struct {
    throttleRates   map[string]*ThrottleState
    config          *ThrottleConfig
    metrics         *ThrottleMetrics
    mutex           sync.RWMutex
}

type ThrottleState struct {
    ConnectionID      string
    CurrentRate       float64    // messages/sec
    OriginalRate      float64
    LastAdjustment    time.Time
    Reason            ThrottleReason
    Active            bool
}

type ThrottleReason int
const (
    ThrottleBackpressure ThrottleReason = iota
    ThrottleSystemLoad
    ThrottleNetworkQuality
    ThrottleClientCapacity
)

type ThrottleConfig struct {
    MinRate               float64
    MaxRate               float64
    AdjustmentFactor      float64
    AdjustmentInterval    time.Duration
    RecoveryRate          float64
    SystemLoadThreshold   float64
}

// 스로틀링 적용
func (dt *DynamicThrottler) ApplyThrottle(connectionID string, level BackpressureLevel) error {
    dt.mutex.Lock()
    defer dt.mutex.Unlock()
    
    throttleState, exists := dt.throttleRates[connectionID]
    if !exists {
        throttleState = &ThrottleState{
            ConnectionID: connectionID,
            CurrentRate:  dt.config.MaxRate,
            OriginalRate: dt.config.MaxRate,
        }
        dt.throttleRates[connectionID] = throttleState
    }
    
    now := time.Now()
    
    // 조정 간격 확인
    if now.Sub(throttleState.LastAdjustment) < dt.config.AdjustmentInterval {
        return nil
    }
    
    switch level {
    case BackpressureNone:
        // 점진적 복구
        if throttleState.Active {
            newRate := throttleState.CurrentRate * (1 + dt.config.RecoveryRate)
            if newRate >= throttleState.OriginalRate {
                throttleState.CurrentRate = throttleState.OriginalRate
                throttleState.Active = false
            } else {
                throttleState.CurrentRate = newRate
            }
        }
        
    case BackpressureLow:
        // 약간 감소
        throttleState.CurrentRate *= (1 - dt.config.AdjustmentFactor * 0.1)
        throttleState.Active = true
        throttleState.Reason = ThrottleBackpressure
        
    case BackpressureMedium:
        // 중간 감소
        throttleState.CurrentRate *= (1 - dt.config.AdjustmentFactor * 0.3)
        throttleState.Active = true
        throttleState.Reason = ThrottleBackpressure
        
    case BackpressureHigh:
        // 큰 감소
        throttleState.CurrentRate *= (1 - dt.config.AdjustmentFactor * 0.5)
        throttleState.Active = true
        throttleState.Reason = ThrottleBackpressure
        
    case BackpressureCritical:
        // 최대 감소
        throttleState.CurrentRate = math.Max(
            dt.config.MinRate,
            throttleState.CurrentRate * (1 - dt.config.AdjustmentFactor),
        )
        throttleState.Active = true
        throttleState.Reason = ThrottleBackpressure
    }
    
    // 최소/최대 범위 확인
    throttleState.CurrentRate = math.Max(dt.config.MinRate,
        math.Min(dt.config.MaxRate, throttleState.CurrentRate))
    
    throttleState.LastAdjustment = now
    
    return nil
}
```

### 4. 적응형 버퍼 관리

```go
// 적응형 버퍼 관리자
type AdaptiveBuffer struct {
    buffers         map[string]*BufferState
    config          *BufferConfig
    globalMemory    *MemoryManager
    mutex           sync.RWMutex
}

type BufferState struct {
    ConnectionID    string
    CurrentSize     int
    MaxSize         int
    MinSize         int
    UtilizationHist []float64
    LastResize      time.Time
    ResizeCount     int
}

type BufferConfig struct {
    InitialSize       int
    MinSize           int
    MaxSize           int
    GrowthFactor      float64
    ShrinkFactor      float64
    ResizeInterval    time.Duration
    UtilizationWindow int
    MemoryLimit       int64
}

// 버퍼 크기 동적 조정
func (ab *AdaptiveBuffer) AdjustBufferSize(connectionID string, metrics *BackpressureMetrics) error {
    ab.mutex.Lock()
    defer ab.mutex.Unlock()
    
    bufferState, exists := ab.buffers[connectionID]
    if !exists {
        bufferState = &BufferState{
            ConnectionID:    connectionID,
            CurrentSize:     ab.config.InitialSize,
            MaxSize:         ab.config.MaxSize,
            MinSize:         ab.config.MinSize,
            UtilizationHist: make([]float64, 0, ab.config.UtilizationWindow),
        }
        ab.buffers[connectionID] = bufferState
    }
    
    now := time.Now()
    
    // 조정 간격 확인
    if now.Sub(bufferState.LastResize) < ab.config.ResizeInterval {
        return nil
    }
    
    // 사용률 히스토리 업데이트
    bufferState.UtilizationHist = append(bufferState.UtilizationHist, metrics.BufferUtilization)
    if len(bufferState.UtilizationHist) > ab.config.UtilizationWindow {
        bufferState.UtilizationHist = bufferState.UtilizationHist[1:]
    }
    
    // 평균 사용률 계산
    avgUtilization := ab.calculateAverageUtilization(bufferState.UtilizationHist)
    
    // 메모리 제약 확인
    availableMemory := ab.globalMemory.GetAvailableMemory()
    
    var newSize int
    
    // 크기 조정 결정
    if avgUtilization > 0.8 && availableMemory > int64(bufferState.CurrentSize) {
        // 버퍼 증가
        newSize = int(float64(bufferState.CurrentSize) * ab.config.GrowthFactor)
        newSize = int(math.Min(float64(newSize), float64(ab.config.MaxSize)))
        
        // 메모리 제한 확인
        if int64(newSize) > availableMemory/2 {
            newSize = int(availableMemory / 2)
        }
    } else if avgUtilization < 0.3 {
        // 버퍼 감소
        newSize = int(float64(bufferState.CurrentSize) * ab.config.ShrinkFactor)
        newSize = int(math.Max(float64(newSize), float64(ab.config.MinSize)))
    } else {
        // 변경 없음
        return nil
    }
    
    if newSize != bufferState.CurrentSize {
        bufferState.CurrentSize = newSize
        bufferState.LastResize = now
        bufferState.ResizeCount++
        
        log.Infof("Buffer resized for connection %s: %d -> %d (utilization: %.2f%%)",
            connectionID, bufferState.CurrentSize, newSize, avgUtilization*100)
    }
    
    return nil
}
```

### 5. 메시지 우선순위 및 드롭 정책

```go
// 메시지 우선순위 관리
type MessagePrioritizer struct {
    queues      map[Priority]*PriorityQueue
    dropPolicy  DropPolicy
    stats       *DropStatistics
    mutex       sync.RWMutex
}

type PriorityQueue struct {
    messages    []PriorityMessage
    maxSize     int
    currentSize int
    mutex       sync.RWMutex
}

type PriorityMessage struct {
    Data        []byte
    Priority    Priority
    Timestamp   time.Time
    SessionID   string
    MessageType MessageType
}

type DropPolicy int
const (
    DropOldest DropPolicy = iota
    DropLowestPriority
    DropBySize
    DropRandom
)

type DropStatistics struct {
    TotalDropped      uint64
    DroppedByPriority map[Priority]uint64
    DroppedByPolicy   map[DropPolicy]uint64
    LastDrop          time.Time
}

// 메시지 드롭 처리
func (mp *MessagePrioritizer) HandleOverflow(connectionID string, newMessage PriorityMessage) error {
    mp.mutex.Lock()
    defer mp.mutex.Unlock()
    
    queue := mp.queues[newMessage.Priority]
    if queue == nil {
        return fmt.Errorf("priority queue not found: %d", newMessage.Priority)
    }
    
    if queue.currentSize >= queue.maxSize {
        // 드롭 정책 적용
        droppedMsg, err := mp.applyDropPolicy(queue, newMessage)
        if err != nil {
            return err
        }
        
        if droppedMsg != nil {
            mp.stats.TotalDropped++
            mp.stats.DroppedByPriority[droppedMsg.Priority]++
            mp.stats.DroppedByPolicy[mp.dropPolicy]++
            mp.stats.LastDrop = time.Now()
            
            log.Warnf("Message dropped for connection %s, priority %d, policy %d",
                connectionID, droppedMsg.Priority, mp.dropPolicy)
        }
    }
    
    // 새 메시지 추가
    return queue.Enqueue(newMessage)
}

func (mp *MessagePrioritizer) applyDropPolicy(queue *PriorityQueue, newMessage PriorityMessage) (*PriorityMessage, error) {
    switch mp.dropPolicy {
    case DropOldest:
        return queue.DequeueOldest(), nil
    case DropLowestPriority:
        return queue.DequeueLowestPriority(), nil
    case DropBySize:
        return queue.DequeueLargest(), nil
    case DropRandom:
        return queue.DequeueRandom(), nil
    default:
        return nil, fmt.Errorf("unknown drop policy: %d", mp.dropPolicy)
    }
}
```

### 6. 플로우 제어 모니터링

```go
// 플로우 제어 모니터
type FlowMonitor struct {
    controller  *FlowController
    metrics     *FlowMetrics
    alerts      *AlertManager
    config      *MonitorConfig
    ticker      *time.Ticker
    stopCh      chan struct{}
}

type FlowMetrics struct {
    ConnectionCount       int
    AverageBackpressure   float64
    ThrottledConnections  int
    DroppedMessages       uint64
    BufferUtilization     float64
    SystemThroughput      float64
    timestamp             time.Time
}

// 모니터링 시작
func (fm *FlowMonitor) StartMonitoring() {
    fm.ticker = time.NewTicker(fm.config.MonitorInterval)
    
    go func() {
        defer fm.ticker.Stop()
        
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
    }()
}

func (fm *FlowMonitor) collectMetrics() error {
    metrics := &FlowMetrics{
        timestamp: time.Now(),
    }
    
    fm.controller.mutex.RLock()
    defer fm.controller.mutex.RUnlock()
    
    metrics.ConnectionCount = len(fm.controller.connections)
    
    var totalBackpressure float64
    var throttledCount int
    var totalBufferUtil float64
    
    for _, flowState := range fm.controller.connections {
        // 백프레셔 레벨을 수치로 변환
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
    
    fm.metrics = metrics
    
    return nil
}
```

## 파일 구조

```
internal/flow/
├── flow_controller.go     # 메인 플로우 제어 관리자
├── backpressure.go        # 백프레셔 감지 시스템
├── throttler.go           # 동적 스로틀링
├── buffer.go              # 적응형 버퍼 관리
├── prioritizer.go         # 메시지 우선순위 관리
├── monitor.go             # 플로우 모니터링
├── metrics.go             # 메트릭 수집 및 관리
└── config.go              # 설정 관리

internal/flow/test/
├── flow_controller_test.go
├── backpressure_test.go
├── throttler_test.go
├── buffer_test.go
└── integration_test.go
```

## 테스트 계획

### 단위 테스트
- 백프레셔 감지 로직 테스트
- 스로틀링 알고리즘 테스트
- 버퍼 크기 조정 테스트
- 메시지 드롭 정책 테스트

### 통합 테스트
- 실제 부하 상황에서의 플로우 제어 테스트
- 다양한 클라이언트 속도에서의 안정성 테스트
- 메모리 제한 상황에서의 동작 테스트

### 성능 테스트
- 플로우 제어 오버헤드 측정
- 스케일링 테스트 (1000+ 동시 연결)
- 메모리 사용량 프로파일링

## Definition of Done
- [ ] 백프레셔 감지 및 대응 시스템 구현 완료
- [ ] 동적 스로틀링 시스템 구현 완료
- [ ] 적응형 버퍼 관리 시스템 구현 완료
- [ ] 메시지 우선순위 및 드롭 정책 구현 완료
- [ ] 플로우 제어 모니터링 시스템 구현 완료
- [ ] 성능 요구사항 달성 확인
- [ ] 단위 테스트 및 통합 테스트 통과
- [ ] 코드 리뷰 완료

## Notes
- 플로우 제어는 시스템 전체 성능에 큰 영향을 미치므로 신중한 튜닝 필요
- 메모리 사용량과 응답성 간의 트레이드오프 고려
- 실제 운영 환경에서의 지속적인 모니터링 및 조정 필요
- 클라이언트별 특성을 고려한 개별 최적화 지원