package websocket

import (
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// BackpressureController 백프레셔 제어기
type BackpressureController struct {
	limit          int
	pendingCount   int
	droppedCount   uint64
	sentCount      uint64
	mutex          sync.RWMutex
	lastThrottle   time.Time
	throttleCount  uint64
}

// NewBackpressureController 새 백프레셔 제어기 생성
func NewBackpressureController(limit int) *BackpressureController {
	if limit <= 0 {
		limit = 100
	}
	
	return &BackpressureController{
		limit: limit,
	}
}

// ShouldThrottle 스로틀링 필요 여부 확인
func (bc *BackpressureController) ShouldThrottle() bool {
	bc.mutex.RLock()
	defer bc.mutex.RUnlock()
	
	shouldThrottle := bc.pendingCount > bc.limit
	
	if shouldThrottle {
		bc.mutex.RUnlock()
		bc.mutex.Lock()
		bc.lastThrottle = time.Now()
		bc.throttleCount++
		bc.mutex.Unlock()
		bc.mutex.RLock()
		
		if bc.throttleCount%100 == 0 {
			log.Warnf("Backpressure throttling active: pending=%d, limit=%d, throttled=%d times",
				bc.pendingCount, bc.limit, bc.throttleCount)
		}
	}
	
	return shouldThrottle
}

// MessagePending 메시지 대기 중
func (bc *BackpressureController) MessagePending() {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()
	bc.pendingCount++
}

// MessageSent 메시지 전송됨
func (bc *BackpressureController) MessageSent() {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()
	
	if bc.pendingCount > 0 {
		bc.pendingCount--
	}
	bc.sentCount++
}

// MessageDropped 메시지 드롭됨
func (bc *BackpressureController) MessageDropped() {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()
	
	if bc.pendingCount > 0 {
		bc.pendingCount--
	}
	bc.droppedCount++
	
	if bc.droppedCount%100 == 0 {
		log.Warnf("Messages dropped due to backpressure: %d", bc.droppedCount)
	}
}

// Reset 백프레셔 리셋
func (bc *BackpressureController) Reset() {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()
	
	bc.pendingCount = 0
	bc.droppedCount = 0
	bc.sentCount = 0
	bc.throttleCount = 0
}

// GetStats 통계 조회
func (bc *BackpressureController) GetStats() map[string]interface{} {
	bc.mutex.RLock()
	defer bc.mutex.RUnlock()
	
	dropRate := float64(0)
	if bc.sentCount+bc.droppedCount > 0 {
		dropRate = float64(bc.droppedCount) / float64(bc.sentCount+bc.droppedCount) * 100
	}
	
	return map[string]interface{}{
		"pending_count":  bc.pendingCount,
		"sent_count":     bc.sentCount,
		"dropped_count":  bc.droppedCount,
		"throttle_count": bc.throttleCount,
		"drop_rate":      dropRate,
		"limit":          bc.limit,
		"last_throttle":  bc.lastThrottle,
	}
}

// UpdateLimit 제한 업데이트
func (bc *BackpressureController) UpdateLimit(newLimit int) {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()
	
	oldLimit := bc.limit
	bc.limit = newLimit
	
	log.Infof("Backpressure limit updated from %d to %d", oldLimit, newLimit)
}