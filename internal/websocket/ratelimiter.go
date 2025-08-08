package websocket

import (
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// RateLimiter 레이트 리미터
type RateLimiter struct {
	rate          int           // 초당 허용 메시지 수
	tokens        int           // 현재 토큰 수
	maxTokens     int           // 최대 토큰 수
	lastRefill    time.Time     // 마지막 리필 시간
	mutex         sync.Mutex
	
	// 통계
	totalAllowed  uint64
	totalDenied   uint64
	totalWaited   uint64
}

// NewRateLimiter 새 레이트 리미터 생성
func NewRateLimiter(ratePerSecond int) *RateLimiter {
	if ratePerSecond <= 0 {
		ratePerSecond = 1000
	}
	
	rl := &RateLimiter{
		rate:       ratePerSecond,
		tokens:     ratePerSecond,
		maxTokens:  ratePerSecond * 2, // 버스트 허용
		lastRefill: time.Now(),
	}
	
	// 토큰 리필 고루틴
	go rl.refillLoop()
	
	return rl
}

// Allow 요청 허용 여부 확인 (논블로킹)
func (rl *RateLimiter) Allow() bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	
	rl.refill()
	
	if rl.tokens > 0 {
		rl.tokens--
		rl.totalAllowed++
		return true
	}
	
	rl.totalDenied++
	return false
}

// Wait 토큰 사용 가능할 때까지 대기
func (rl *RateLimiter) Wait() {
	for !rl.Allow() {
		rl.totalWaited++
		time.Sleep(time.Duration(1000/rl.rate) * time.Millisecond)
	}
}

// WaitN N개 토큰 사용 가능할 때까지 대기
func (rl *RateLimiter) WaitN(n int) {
	for i := 0; i < n; i++ {
		rl.Wait()
	}
}

// refill 토큰 리필
func (rl *RateLimiter) refill() {
	now := time.Now()
	elapsed := now.Sub(rl.lastRefill)
	
	// 경과 시간에 따라 토큰 추가
	tokensToAdd := int(elapsed.Seconds() * float64(rl.rate))
	
	if tokensToAdd > 0 {
		rl.tokens = min(rl.tokens+tokensToAdd, rl.maxTokens)
		rl.lastRefill = now
	}
}

// refillLoop 주기적 토큰 리필
func (rl *RateLimiter) refillLoop() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	
	for range ticker.C {
		rl.mutex.Lock()
		rl.refill()
		rl.mutex.Unlock()
	}
}

// UpdateRate 레이트 업데이트
func (rl *RateLimiter) UpdateRate(newRate int) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	
	oldRate := rl.rate
	rl.rate = newRate
	rl.maxTokens = newRate * 2
	
	// 토큰 조정
	if rl.tokens > rl.maxTokens {
		rl.tokens = rl.maxTokens
	}
	
	log.Infof("Rate limit updated from %d/s to %d/s", oldRate, newRate)
}

// GetStats 통계 조회
func (rl *RateLimiter) GetStats() map[string]interface{} {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	
	allowRate := float64(0)
	if rl.totalAllowed+rl.totalDenied > 0 {
		allowRate = float64(rl.totalAllowed) / float64(rl.totalAllowed+rl.totalDenied) * 100
	}
	
	return map[string]interface{}{
		"rate":           rl.rate,
		"current_tokens": rl.tokens,
		"max_tokens":     rl.maxTokens,
		"total_allowed":  rl.totalAllowed,
		"total_denied":   rl.totalDenied,
		"total_waited":   rl.totalWaited,
		"allow_rate":     allowRate,
	}
}

// Reset 레이트 리미터 리셋
func (rl *RateLimiter) Reset() {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	
	rl.tokens = rl.rate
	rl.totalAllowed = 0
	rl.totalDenied = 0
	rl.totalWaited = 0
	rl.lastRefill = time.Now()
}

// min 최솟값
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}