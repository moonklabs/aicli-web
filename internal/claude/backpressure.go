package claude

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

// DropPolicy는 백프레셔 상황에서의 메시지 드롭 정책을 정의합니다.
type DropPolicy int

const (
	// DropOldest는 버퍼가 가득 찰 때 가장 오래된 메시지를 드롭합니다.
	DropOldest DropPolicy = iota
	// DropNewest는 버퍼가 가득 찰 때 새로운 메시지를 드롭합니다.
	DropNewest
	// BlockUntilReady는 버퍼에 공간이 생길 때까지 블로킹합니다.
	BlockUntilReady
)

// BackpressureHandler는 백프레셔 처리를 담당하는 구조체입니다.
type BackpressureHandler struct {
	maxBufferSize     int
	currentBufferSize int64
	dropPolicy        DropPolicy
	slowConsumerCh    chan struct{}
	metrics           *BackpressureMetrics
	logger            *logrus.Logger
	mu                sync.RWMutex
	
	// 적응형 버퍼 크기 조정
	adaptiveBuffering  bool
	minBufferSize      int
	bufferGrowthRate   float64
	bufferShrinkRate   float64
	lastAdjustmentTime time.Time
}

// BackpressureMetrics는 백프레셔 관련 메트릭을 추적합니다.
type BackpressureMetrics struct {
	DroppedMessages   int64
	BackpressureEvents int64
	BufferResizes     int64
	SlowConsumerCount int64
	AvgBufferUsage    float64
	mu                sync.RWMutex
}

// BackpressureConfig는 백프레셔 핸들러 설정을 정의합니다.
type BackpressureConfig struct {
	MaxBufferSize     int
	DropPolicy        DropPolicy
	AdaptiveBuffering bool
	MinBufferSize     int
	BufferGrowthRate  float64
	BufferShrinkRate  float64
}

// NewBackpressureHandler는 새로운 백프레셔 핸들러를 생성합니다.
func NewBackpressureHandler(config BackpressureConfig, logger *logrus.Logger) *BackpressureHandler {
	if config.MaxBufferSize <= 0 {
		config.MaxBufferSize = 1000
	}
	if config.MinBufferSize <= 0 {
		config.MinBufferSize = 100
	}
	if config.BufferGrowthRate <= 0 {
		config.BufferGrowthRate = 1.5
	}
	if config.BufferShrinkRate <= 0 {
		config.BufferShrinkRate = 0.8
	}

	return &BackpressureHandler{
		maxBufferSize:      config.MaxBufferSize,
		dropPolicy:         config.DropPolicy,
		slowConsumerCh:     make(chan struct{}, 1),
		metrics:            &BackpressureMetrics{},
		logger:             logger,
		adaptiveBuffering:  config.AdaptiveBuffering,
		minBufferSize:      config.MinBufferSize,
		bufferGrowthRate:   config.BufferGrowthRate,
		bufferShrinkRate:   config.BufferShrinkRate,
		lastAdjustmentTime: time.Now(),
	}
}

// ShouldDrop은 현재 버퍼 상태를 기반으로 메시지를 드롭해야 하는지 결정합니다.
func (bh *BackpressureHandler) ShouldDrop() bool {
	currentSize := atomic.LoadInt64(&bh.currentBufferSize)
	
	bh.mu.RLock()
	maxSize := int64(bh.maxBufferSize)
	policy := bh.dropPolicy
	bh.mu.RUnlock()

	if currentSize >= maxSize {
		// 백프레셔 이벤트 기록
		atomic.AddInt64(&bh.metrics.BackpressureEvents, 1)
		
		// 느린 소비자 알림
		select {
		case bh.slowConsumerCh <- struct{}{}:
		default:
		}

		return policy != BlockUntilReady
	}

	return false
}

// WaitForSpace는 버퍼에 공간이 생길 때까지 대기합니다.
func (bh *BackpressureHandler) WaitForSpace(ctx context.Context) error {
	if bh.dropPolicy != BlockUntilReady {
		return nil
	}

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			currentSize := atomic.LoadInt64(&bh.currentBufferSize)
			bh.mu.RLock()
			maxSize := int64(bh.maxBufferSize)
			bh.mu.RUnlock()
			
			if currentSize < maxSize {
				return nil
			}
		}
	}
}

// IncrementBuffer는 버퍼 크기를 증가시킵니다.
func (bh *BackpressureHandler) IncrementBuffer() {
	atomic.AddInt64(&bh.currentBufferSize, 1)
	bh.updateMetrics()
}

// DecrementBuffer는 버퍼 크기를 감소시킵니다.
func (bh *BackpressureHandler) DecrementBuffer() {
	atomic.AddInt64(&bh.currentBufferSize, -1)
	bh.updateMetrics()
}

// HandleDrop은 드롭 정책에 따라 메시지를 처리합니다.
func (bh *BackpressureHandler) HandleDrop(messages []interface{}) ([]interface{}, error) {
	bh.mu.RLock()
	policy := bh.dropPolicy
	bh.mu.RUnlock()

	switch policy {
	case DropOldest:
		if len(messages) > 0 {
			atomic.AddInt64(&bh.metrics.DroppedMessages, 1)
			bh.logger.Debug("Dropping oldest message due to backpressure")
			return messages[1:], nil
		}
	case DropNewest:
		atomic.AddInt64(&bh.metrics.DroppedMessages, 1)
		bh.logger.Debug("Dropping newest message due to backpressure")
		return messages, fmt.Errorf("message dropped due to backpressure")
	case BlockUntilReady:
		// BlockUntilReady는 WaitForSpace에서 처리됨
		return messages, nil
	}

	return messages, nil
}

// AdjustBufferSize는 현재 사용률을 기반으로 버퍼 크기를 조정합니다.
func (bh *BackpressureHandler) AdjustBufferSize() {
	if !bh.adaptiveBuffering {
		return
	}

	bh.mu.Lock()
	defer bh.mu.Unlock()

	// 조정 간격 확인 (최소 1초)
	if time.Since(bh.lastAdjustmentTime) < time.Second {
		return
	}

	currentSize := atomic.LoadInt64(&bh.currentBufferSize)
	usage := float64(currentSize) / float64(bh.maxBufferSize)

	// 고사용률: 버퍼 크기 증가
	if usage > 0.8 {
		newSize := int(float64(bh.maxBufferSize) * bh.bufferGrowthRate)
		bh.maxBufferSize = newSize
		atomic.AddInt64(&bh.metrics.BufferResizes, 1)
		bh.logger.WithField("new_size", newSize).Info("Increased buffer size due to high usage")
	} else if usage < 0.2 && bh.maxBufferSize > bh.minBufferSize {
		// 저사용률: 버퍼 크기 감소
		newSize := int(float64(bh.maxBufferSize) * bh.bufferShrinkRate)
		if newSize < bh.minBufferSize {
			newSize = bh.minBufferSize
		}
		bh.maxBufferSize = newSize
		atomic.AddInt64(&bh.metrics.BufferResizes, 1)
		bh.logger.WithField("new_size", newSize).Info("Decreased buffer size due to low usage")
	}

	bh.lastAdjustmentTime = time.Now()
}

// MonitorSlowConsumers는 느린 소비자를 모니터링합니다.
func (bh *BackpressureHandler) MonitorSlowConsumers(ctx context.Context) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	consecutiveSlowCount := 0
	const slowThreshold = 10

	for {
		select {
		case <-ctx.Done():
			return
		case <-bh.slowConsumerCh:
			consecutiveSlowCount++
			atomic.AddInt64(&bh.metrics.SlowConsumerCount, 1)
			
			if consecutiveSlowCount >= slowThreshold {
				bh.logger.Warn("Detected consistently slow consumer")
				bh.AdjustBufferSize()
				consecutiveSlowCount = 0
			}
		case <-ticker.C:
			// 정기적으로 카운터 리셋
			if consecutiveSlowCount > 0 {
				consecutiveSlowCount--
			}
		}
	}
}

// updateMetrics는 메트릭을 업데이트합니다.
func (bh *BackpressureHandler) updateMetrics() {
	currentSize := atomic.LoadInt64(&bh.currentBufferSize)
	
	bh.mu.RLock()
	maxSize := float64(bh.maxBufferSize)
	bh.mu.RUnlock()

	bh.metrics.mu.Lock()
	defer bh.metrics.mu.Unlock()

	// 이동 평균 계산
	alpha := 0.1 // 평활 계수
	currentUsage := float64(currentSize) / maxSize
	bh.metrics.AvgBufferUsage = alpha*currentUsage + (1-alpha)*bh.metrics.AvgBufferUsage
}

// GetMetrics는 백프레셔 메트릭을 반환합니다.
func (bh *BackpressureHandler) GetMetrics() BackpressureMetrics {
	bh.metrics.mu.RLock()
	defer bh.metrics.mu.RUnlock()

	return BackpressureMetrics{
		DroppedMessages:    atomic.LoadInt64(&bh.metrics.DroppedMessages),
		BackpressureEvents: atomic.LoadInt64(&bh.metrics.BackpressureEvents),
		BufferResizes:      atomic.LoadInt64(&bh.metrics.BufferResizes),
		SlowConsumerCount:  atomic.LoadInt64(&bh.metrics.SlowConsumerCount),
		AvgBufferUsage:     bh.metrics.AvgBufferUsage,
	}
}

// GetCurrentBufferSize는 현재 버퍼 크기를 반환합니다.
func (bh *BackpressureHandler) GetCurrentBufferSize() int64 {
	return atomic.LoadInt64(&bh.currentBufferSize)
}

// GetMaxBufferSize는 최대 버퍼 크기를 반환합니다.
func (bh *BackpressureHandler) GetMaxBufferSize() int {
	bh.mu.RLock()
	defer bh.mu.RUnlock()
	return bh.maxBufferSize
}

// Reset은 백프레셔 핸들러를 초기화합니다.
func (bh *BackpressureHandler) Reset() {
	atomic.StoreInt64(&bh.currentBufferSize, 0)
	atomic.StoreInt64(&bh.metrics.DroppedMessages, 0)
	atomic.StoreInt64(&bh.metrics.BackpressureEvents, 0)
	atomic.StoreInt64(&bh.metrics.BufferResizes, 0)
	atomic.StoreInt64(&bh.metrics.SlowConsumerCount, 0)
	
	bh.metrics.mu.Lock()
	bh.metrics.AvgBufferUsage = 0
	bh.metrics.mu.Unlock()
}