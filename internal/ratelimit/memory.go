package ratelimit

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// MemoryLimiter는 메모리 기반의 Rate Limiter입니다.
type MemoryLimiter struct {
	config   *LimiterConfig
	limiters map[string]*rateLimiterEntry
	mu       sync.RWMutex
	
	// cleanup 관련
	cleanupInterval time.Duration
	maxIdleTime     time.Duration
	stopCleanup     chan struct{}
}

// rateLimiterEntry는 개별 키에 대한 rate limiter 엔트리입니다.
type rateLimiterEntry struct {
	limiter  *rate.Limiter
	lastUsed time.Time
	resetAt  time.Time
	config   *LimiterConfig
	tokens   int // 현재 토큰 수
}

// NewMemoryLimiter는 새로운 메모리 기반 Rate Limiter를 생성합니다.
func NewMemoryLimiter(config *LimiterConfig) *MemoryLimiter {
	if config == nil {
		config = DefaultLimiterConfig()
	}
	
	ml := &MemoryLimiter{
		config:          config,
		limiters:        make(map[string]*rateLimiterEntry),
		cleanupInterval: 10 * time.Minute,
		maxIdleTime:     30 * time.Minute,
		stopCleanup:     make(chan struct{}),
	}
	
	// 백그라운드 정리 작업 시작
	go ml.cleanupLoop()
	
	return ml
}

// Allow는 주어진 키에 대해 요청을 허용할지 확인합니다.
func (ml *MemoryLimiter) Allow(key string) bool {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	
	entry, exists := ml.limiters[key]
	now := time.Now()
	
	if !exists {
		// 새로운 키에 대한 limiter 생성
		limiter := rate.NewLimiter(
			rate.Every(time.Duration(60/ml.config.Rate)*time.Second),
			ml.config.Burst,
		)
		
		entry = &rateLimiterEntry{
			limiter:  limiter,
			lastUsed: now,
			resetAt:  now.Add(time.Duration(ml.config.Window) * time.Second),
			config:   ml.config,
			tokens:   ml.config.Burst - 1, // 한 개 소모
		}
		
		ml.limiters[key] = entry
		return true
	}
	
	// 시간 윈도우가 지났으면 리셋
	if now.After(entry.resetAt) {
		entry.limiter = rate.NewLimiter(
			rate.Every(time.Duration(60/ml.config.Rate)*time.Second),
			ml.config.Burst,
		)
		entry.resetAt = now.Add(time.Duration(ml.config.Window) * time.Second)
		entry.tokens = ml.config.Burst
	}
	
	entry.lastUsed = now
	
	// 토큰 사용 가능 여부 확인
	if entry.limiter.Allow() {
		if entry.tokens > 0 {
			entry.tokens--
		}
		return true
	}
	
	return false
}

// Reset은 주어진 키의 rate limit을 리셋합니다.
func (ml *MemoryLimiter) Reset(key string) {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	
	delete(ml.limiters, key)
}

// Remaining은 남은 요청 수를 반환합니다.
func (ml *MemoryLimiter) Remaining(key string) int {
	ml.mu.RLock()
	defer ml.mu.RUnlock()
	
	entry, exists := ml.limiters[key]
	if !exists {
		return ml.config.Burst
	}
	
	// 시간이 지났으면 전체 토큰으로 계산
	if time.Now().After(entry.resetAt) {
		return ml.config.Burst
	}
	
	return entry.tokens
}

// Limit은 제한된 요청 수를 반환합니다.
func (ml *MemoryLimiter) Limit(key string) int {
	return ml.config.Rate
}

// ResetTime은 리셋 시간을 반환합니다.
func (ml *MemoryLimiter) ResetTime(key string) time.Time {
	ml.mu.RLock()
	defer ml.mu.RUnlock()
	
	entry, exists := ml.limiters[key]
	if !exists {
		return time.Now().Add(time.Duration(ml.config.Window) * time.Second)
	}
	
	return entry.resetAt
}

// Close는 limiter 리소스를 정리합니다.
func (ml *MemoryLimiter) Close() error {
	close(ml.stopCleanup)
	return nil
}

// cleanupLoop는 주기적으로 사용되지 않는 limiter를 정리합니다.
func (ml *MemoryLimiter) cleanupLoop() {
	ticker := time.NewTicker(ml.cleanupInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			ml.cleanup()
		case <-ml.stopCleanup:
			return
		}
	}
}

// cleanup은 오래된 limiter 엔트리를 제거합니다.
func (ml *MemoryLimiter) cleanup() {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	
	now := time.Now()
	for key, entry := range ml.limiters {
		if now.Sub(entry.lastUsed) > ml.maxIdleTime {
			delete(ml.limiters, key)
		}
	}
}

// GetStats는 현재 limiter 통계를 반환합니다.
func (ml *MemoryLimiter) GetStats() map[string]interface{} {
	ml.mu.RLock()
	defer ml.mu.RUnlock()
	
	return map[string]interface{}{
		"total_limiters": len(ml.limiters),
		"config": map[string]interface{}{
			"rate":   ml.config.Rate,
			"burst":  ml.config.Burst,
			"window": ml.config.Window,
		},
	}
}