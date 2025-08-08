package flow

import (
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

// AdaptiveBuffer 적응형 버퍼 관리자
type AdaptiveBuffer struct {
	buffers       map[string]*BufferState
	config        *BufferConfig
	globalMemory  *MemoryManager
	mutex         sync.RWMutex
	
	// 통계
	totalResizes  uint64
	totalMemory   int64
}

// BufferState 버퍼 상태
type BufferState struct {
	ConnectionID     string
	CurrentSize      int32
	MaxSize          int32
	MinSize          int32
	UtilizationHist  []float64
	LastResize       time.Time
	ResizeCount      uint32
	
	// 실제 버퍼
	buffer           []byte
	writePos         int32
	readPos          int32
	
	// 성능 메트릭
	TotalWrites      uint64
	TotalReads       uint64
	OverflowCount    uint64
	UnderflowCount   uint64
	
	mutex            sync.RWMutex
}

// MemoryManager 전역 메모리 관리자
type MemoryManager struct {
	totalLimit    int64
	currentUsage  int64
	reservations  map[string]int64
	mutex         sync.RWMutex
}

// BackpressureMetrics 백프레셔 메트릭
type BackpressureMetrics struct {
	BufferUtilization   float64
	ProcessingRate      float64
	AverageLatency      time.Duration
	QueueDepth          int
	DroppedMessages     uint64
	LastBackpressure    time.Time
}

// NewAdaptiveBuffer 새 적응형 버퍼 생성
func NewAdaptiveBuffer(config *BufferConfig, memoryManager *MemoryManager) *AdaptiveBuffer {
	if memoryManager == nil {
		memoryManager = NewMemoryManager(config.MemoryLimit)
	}
	
	return &AdaptiveBuffer{
		buffers:      make(map[string]*BufferState),
		config:       config,
		globalMemory: memoryManager,
	}
}

// NewMemoryManager 새 메모리 관리자 생성
func NewMemoryManager(limit int64) *MemoryManager {
	return &MemoryManager{
		totalLimit:   limit,
		reservations: make(map[string]int64),
	}
}

// CreateBuffer 새 버퍼 생성
func (ab *AdaptiveBuffer) CreateBuffer(connectionID string) error {
	ab.mutex.Lock()
	defer ab.mutex.Unlock()
	
	if _, exists := ab.buffers[connectionID]; exists {
		return fmt.Errorf("buffer already exists for connection: %s", connectionID)
	}
	
	initialSize := ab.config.InitialSize
	
	// 메모리 예약
	if err := ab.globalMemory.Reserve(connectionID, int64(initialSize)); err != nil {
		return fmt.Errorf("failed to reserve memory: %w", err)
	}
	
	bufferState := &BufferState{
		ConnectionID:    connectionID,
		CurrentSize:     int32(initialSize),
		MaxSize:         int32(ab.config.MaxSize),
		MinSize:         int32(ab.config.MinSize),
		UtilizationHist: make([]float64, 0, ab.config.UtilizationWindow),
		LastResize:      time.Now(),
		buffer:          make([]byte, initialSize),
	}
	
	ab.buffers[connectionID] = bufferState
	atomic.AddInt64(&ab.totalMemory, int64(initialSize))
	
	log.Debugf("Buffer created for connection %s: size=%d", connectionID, initialSize)
	return nil
}

// RemoveBuffer 버퍼 제거
func (ab *AdaptiveBuffer) RemoveBuffer(connectionID string) {
	ab.mutex.Lock()
	defer ab.mutex.Unlock()
	
	bufferState, exists := ab.buffers[connectionID]
	if !exists {
		return
	}
	
	// 메모리 해제
	ab.globalMemory.Release(connectionID)
	atomic.AddInt64(&ab.totalMemory, -int64(bufferState.CurrentSize))
	
	delete(ab.buffers, connectionID)
	log.Debugf("Buffer removed for connection %s", connectionID)
}

// AdjustBufferSize 버퍼 크기 동적 조정
func (ab *AdaptiveBuffer) AdjustBufferSize(connectionID string, metrics *BackpressureMetrics) error {
	ab.mutex.Lock()
	defer ab.mutex.Unlock()
	
	bufferState, exists := ab.buffers[connectionID]
	if !exists {
		// 버퍼가 없으면 생성
		ab.mutex.Unlock()
		if err := ab.CreateBuffer(connectionID); err != nil {
			return err
		}
		ab.mutex.Lock()
		bufferState = ab.buffers[connectionID]
	}
	
	now := time.Now()
	
	// 조정 간격 확인
	if now.Sub(bufferState.LastResize) < ab.config.ResizeInterval {
		return nil
	}
	
	// 사용률 히스토리 업데이트
	bufferState.mutex.Lock()
	bufferState.UtilizationHist = append(bufferState.UtilizationHist, metrics.BufferUtilization)
	if len(bufferState.UtilizationHist) > ab.config.UtilizationWindow {
		bufferState.UtilizationHist = bufferState.UtilizationHist[1:]
	}
	
	// 평균 사용률 계산
	avgUtilization := ab.calculateAverageUtilization(bufferState.UtilizationHist)
	
	// 트렌드 분석
	trend := ab.analyzeUtilizationTrend(bufferState.UtilizationHist)
	bufferState.mutex.Unlock()
	
	// 메모리 제약 확인
	availableMemory := ab.globalMemory.GetAvailableMemory()
	
	// 크기 조정 결정
	newSize := ab.calculateNewSize(bufferState, avgUtilization, trend, availableMemory)
	
	// 크기 변경 적용
	if newSize != int(bufferState.CurrentSize) {
		if err := ab.resizeBuffer(bufferState, newSize); err != nil {
			return fmt.Errorf("failed to resize buffer: %w", err)
		}
		
		atomic.AddUint64(&ab.totalResizes, 1)
		atomic.AddUint32(&bufferState.ResizeCount, 1)
		
		log.Infof("Buffer resized for connection %s: %d -> %d (utilization: %.2f%%, trend: %.2f)",
			connectionID, bufferState.CurrentSize, newSize, avgUtilization*100, trend)
	}
	
	return nil
}

// calculateNewSize 새 버퍼 크기 계산
func (ab *AdaptiveBuffer) calculateNewSize(state *BufferState, avgUtil, trend float64, availableMemory int64) int {
	currentSize := int(atomic.LoadInt32(&state.CurrentSize))
	
	// 성능 지표 기반 조정
	performanceScore := ab.calculatePerformanceScore(state)
	
	var targetSize int
	
	// 사용률과 트렌드 기반 결정
	if avgUtil > 0.85 && trend > 0 {
		// 버퍼 부족 - 증가 필요
		growthFactor := ab.config.GrowthFactor
		
		// 트렌드가 급격하면 더 많이 증가
		if trend > 0.2 {
			growthFactor *= 1.2
		}
		
		targetSize = int(float64(currentSize) * growthFactor)
		
		// 오버플로우가 자주 발생하면 더 크게 증가
		if state.OverflowCount > 10 {
			targetSize = int(float64(targetSize) * 1.1)
		}
		
	} else if avgUtil < 0.3 && trend < 0 {
		// 버퍼 과잉 - 감소 필요
		shrinkFactor := ab.config.ShrinkFactor
		
		// 성능이 좋으면 더 적극적으로 감소
		if performanceScore > 0.9 {
			shrinkFactor *= 0.9
		}
		
		targetSize = int(float64(currentSize) * shrinkFactor)
		
	} else if avgUtil > 0.7 && avgUtil < 0.85 {
		// 최적 범위 - 미세 조정
		if trend > 0.1 {
			targetSize = int(float64(currentSize) * 1.1)
		} else if trend < -0.1 {
			targetSize = int(float64(currentSize) * 0.95)
		} else {
			targetSize = currentSize
		}
		
	} else {
		// 변경 없음
		targetSize = currentSize
	}
	
	// 메모리 제약 적용
	if targetSize > currentSize {
		maxIncrease := int(availableMemory / 2)
		if targetSize-currentSize > maxIncrease {
			targetSize = currentSize + maxIncrease
		}
	}
	
	// 범위 제한
	targetSize = int(math.Max(float64(state.MinSize), math.Min(float64(state.MaxSize), float64(targetSize))))
	
	return targetSize
}

// resizeBuffer 버퍼 크기 변경
func (ab *AdaptiveBuffer) resizeBuffer(state *BufferState, newSize int) error {
	state.mutex.Lock()
	defer state.mutex.Unlock()
	
	currentSize := int(state.CurrentSize)
	sizeDiff := int64(newSize - currentSize)
	
	// 메모리 예약 변경
	if sizeDiff > 0 {
		// 증가
		if err := ab.globalMemory.Reserve(state.ConnectionID, sizeDiff); err != nil {
			return fmt.Errorf("failed to reserve additional memory: %w", err)
		}
	} else if sizeDiff < 0 {
		// 감소
		ab.globalMemory.ReleasePartial(state.ConnectionID, -sizeDiff)
	}
	
	// 새 버퍼 할당
	newBuffer := make([]byte, newSize)
	
	// 기존 데이터 복사
	if state.buffer != nil {
		dataSize := state.writePos - state.readPos
		if dataSize > 0 && dataSize <= int32(newSize) {
			copy(newBuffer, state.buffer[state.readPos:state.writePos])
			state.writePos = dataSize
			state.readPos = 0
		} else {
			// 데이터가 새 버퍼보다 크면 일부만 복사
			if dataSize > int32(newSize) {
				copy(newBuffer, state.buffer[state.writePos-int32(newSize):state.writePos])
				state.writePos = int32(newSize)
				state.readPos = 0
				atomic.AddUint64(&state.OverflowCount, 1)
			}
		}
	}
	
	// 버퍼 교체
	state.buffer = newBuffer
	atomic.StoreInt32(&state.CurrentSize, int32(newSize))
	state.LastResize = time.Now()
	
	// 전역 메모리 통계 업데이트
	atomic.AddInt64(&ab.totalMemory, sizeDiff)
	
	return nil
}

// Write 버퍼에 데이터 쓰기
func (ab *AdaptiveBuffer) Write(connectionID string, data []byte) (int, error) {
	ab.mutex.RLock()
	bufferState, exists := ab.buffers[connectionID]
	ab.mutex.RUnlock()
	
	if !exists {
		return 0, fmt.Errorf("buffer not found for connection: %s", connectionID)
	}
	
	bufferState.mutex.Lock()
	defer bufferState.mutex.Unlock()
	
	dataLen := len(data)
	bufferSize := int(bufferState.CurrentSize)
	availableSpace := bufferSize - int(bufferState.writePos)
	
	if dataLen > availableSpace {
		// 버퍼 오버플로우
		atomic.AddUint64(&bufferState.OverflowCount, 1)
		
		// 순환 버퍼 처리
		if bufferState.readPos > 0 {
			// 읽은 데이터 압축
			remaining := bufferState.writePos - bufferState.readPos
			copy(bufferState.buffer, bufferState.buffer[bufferState.readPos:bufferState.writePos])
			bufferState.writePos = remaining
			bufferState.readPos = 0
			availableSpace = bufferSize - int(bufferState.writePos)
		}
		
		if dataLen > availableSpace {
			// 여전히 공간 부족
			return 0, fmt.Errorf("buffer overflow: need %d, available %d", dataLen, availableSpace)
		}
	}
	
	// 데이터 쓰기
	copy(bufferState.buffer[bufferState.writePos:], data)
	bufferState.writePos += int32(dataLen)
	atomic.AddUint64(&bufferState.TotalWrites, 1)
	
	return dataLen, nil
}

// Read 버퍼에서 데이터 읽기
func (ab *AdaptiveBuffer) Read(connectionID string, size int) ([]byte, error) {
	ab.mutex.RLock()
	bufferState, exists := ab.buffers[connectionID]
	ab.mutex.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("buffer not found for connection: %s", connectionID)
	}
	
	bufferState.mutex.Lock()
	defer bufferState.mutex.Unlock()
	
	availableData := int(bufferState.writePos - bufferState.readPos)
	if availableData <= 0 {
		// 버퍼 언더플로우
		atomic.AddUint64(&bufferState.UnderflowCount, 1)
		return nil, nil
	}
	
	readSize := size
	if readSize > availableData {
		readSize = availableData
	}
	
	// 데이터 읽기
	data := make([]byte, readSize)
	copy(data, bufferState.buffer[bufferState.readPos:bufferState.readPos+int32(readSize)])
	bufferState.readPos += int32(readSize)
	atomic.AddUint64(&bufferState.TotalReads, 1)
	
	// 버퍼가 비었으면 리셋
	if bufferState.readPos == bufferState.writePos {
		bufferState.readPos = 0
		bufferState.writePos = 0
	}
	
	return data, nil
}

// GetBufferState 버퍼 상태 조회
func (ab *AdaptiveBuffer) GetBufferState(connectionID string) (*BufferState, error) {
	ab.mutex.RLock()
	defer ab.mutex.RUnlock()
	
	state, exists := ab.buffers[connectionID]
	if !exists {
		return nil, fmt.Errorf("buffer not found for connection: %s", connectionID)
	}
	
	return state, nil
}

// GetUtilization 버퍼 사용률 조회
func (ab *AdaptiveBuffer) GetUtilization(connectionID string) (float64, error) {
	state, err := ab.GetBufferState(connectionID)
	if err != nil {
		return 0, err
	}
	
	state.mutex.RLock()
	defer state.mutex.RUnlock()
	
	used := state.writePos - state.readPos
	total := state.CurrentSize
	
	if total == 0 {
		return 0, nil
	}
	
	return float64(used) / float64(total), nil
}

// calculateAverageUtilization 평균 사용률 계산
func (ab *AdaptiveBuffer) calculateAverageUtilization(history []float64) float64 {
	if len(history) == 0 {
		return 0
	}
	
	sum := 0.0
	for _, util := range history {
		sum += util
	}
	
	return sum / float64(len(history))
}

// analyzeUtilizationTrend 사용률 트렌드 분석
func (ab *AdaptiveBuffer) analyzeUtilizationTrend(history []float64) float64 {
	if len(history) < 2 {
		return 0
	}
	
	// 간단한 선형 회귀로 트렌드 계산
	n := float64(len(history))
	sumX := 0.0
	sumY := 0.0
	sumXY := 0.0
	sumX2 := 0.0
	
	for i, y := range history {
		x := float64(i)
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}
	
	denominator := n*sumX2 - sumX*sumX
	if denominator == 0 {
		return 0
	}
	
	slope := (n*sumXY - sumX*sumY) / denominator
	return slope
}

// calculatePerformanceScore 성능 점수 계산
func (ab *AdaptiveBuffer) calculatePerformanceScore(state *BufferState) float64 {
	totalOps := state.TotalWrites + state.TotalReads
	if totalOps == 0 {
		return 1.0
	}
	
	// 오버플로우/언더플로우 비율
	errorRate := float64(state.OverflowCount+state.UnderflowCount) / float64(totalOps)
	
	// 성능 점수 (에러율의 역수)
	score := 1.0 - math.Min(errorRate, 1.0)
	return score
}

// MemoryManager methods

// Reserve 메모리 예약
func (mm *MemoryManager) Reserve(connectionID string, size int64) error {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()
	
	if mm.currentUsage+size > mm.totalLimit {
		return fmt.Errorf("memory limit exceeded: requested %d, available %d",
			size, mm.totalLimit-mm.currentUsage)
	}
	
	if current, exists := mm.reservations[connectionID]; exists {
		mm.currentUsage += size - current
	} else {
		mm.currentUsage += size
	}
	
	mm.reservations[connectionID] = size
	return nil
}

// Release 메모리 해제
func (mm *MemoryManager) Release(connectionID string) {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()
	
	if size, exists := mm.reservations[connectionID]; exists {
		mm.currentUsage -= size
		delete(mm.reservations, connectionID)
	}
}

// ReleasePartial 부분 메모리 해제
func (mm *MemoryManager) ReleasePartial(connectionID string, size int64) {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()
	
	if current, exists := mm.reservations[connectionID]; exists {
		newSize := current - size
		if newSize <= 0 {
			mm.currentUsage -= current
			delete(mm.reservations, connectionID)
		} else {
			mm.currentUsage -= size
			mm.reservations[connectionID] = newSize
		}
	}
}

// GetAvailableMemory 사용 가능한 메모리 조회
func (mm *MemoryManager) GetAvailableMemory() int64 {
	mm.mutex.RLock()
	defer mm.mutex.RUnlock()
	
	return mm.totalLimit - mm.currentUsage
}

// GetMemoryPressure 메모리 압박 수준 조회
func (mm *MemoryManager) GetMemoryPressure() float64 {
	mm.mutex.RLock()
	defer mm.mutex.RUnlock()
	
	if mm.totalLimit == 0 {
		return 0
	}
	
	return float64(mm.currentUsage) / float64(mm.totalLimit)
}