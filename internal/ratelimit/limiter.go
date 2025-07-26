package ratelimit

import (
	"time"
)

// RateLimiter는 Rate Limiting 기능을 제공하는 인터페이스입니다.
type RateLimiter interface {
	// Allow는 주어진 키에 대해 요청을 허용할지 확인합니다.
	Allow(key string) bool
	
	// Reset은 주어진 키의 rate limit을 리셋합니다.
	Reset(key string)
	
	// Remaining은 남은 요청 수를 반환합니다.
	Remaining(key string) int
	
	// Limit은 제한된 요청 수를 반환합니다.
	Limit(key string) int
	
	// ResetTime은 리셋 시간을 반환합니다.
	ResetTime(key string) time.Time
	
	// GetStats는 현재 limiter 통계를 반환합니다.
	GetStats() map[string]interface{}
	
	// Close는 limiter 리소스를 정리합니다.
	Close() error
}

// LimiterConfig는 Rate Limiter 설정입니다.
type LimiterConfig struct {
	// Rate는 분당 허용 요청 수입니다.
	Rate int
	
	// Burst는 버스트 허용량입니다.
	Burst int
	
	// Window는 시간 윈도우 크기입니다 (초).
	Window int
}

// DefaultLimiterConfig는 기본 Rate Limiter 설정을 반환합니다.
func DefaultLimiterConfig() *LimiterConfig {
	return &LimiterConfig{
		Rate:   60,  // 60 requests per minute
		Burst:  10,  // burst of 10 requests
		Window: 60,  // 1 minute window
	}
}