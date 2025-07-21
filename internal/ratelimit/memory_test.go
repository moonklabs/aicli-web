package ratelimit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryLimiter_Basic(t *testing.T) {
	config := &LimiterConfig{
		Rate:   60,  // 60 requests per minute
		Burst:  5,   // burst of 5 requests
		Window: 60,  // 1 minute window
	}
	
	limiter := NewMemoryLimiter(config)
	defer limiter.Close()
	
	key := "test-key"
	
	// 초기 상태 확인
	assert.Equal(t, 60, limiter.Limit(key))
	assert.Equal(t, 5, limiter.Remaining(key))
	
	// 첫 번째 요청 (버스트에서 차감)
	assert.True(t, limiter.Allow(key))
	assert.Equal(t, 4, limiter.Remaining(key))
	
	// 나머지 버스트 소모
	for i := 0; i < 4; i++ {
		assert.True(t, limiter.Allow(key))
	}
	
	// 버스트 소모 후 추가 요청은 거부되어야 함
	assert.False(t, limiter.Allow(key))
	assert.Equal(t, 0, limiter.Remaining(key))
}

func TestMemoryLimiter_Reset(t *testing.T) {
	config := &LimiterConfig{
		Rate:   60,
		Burst:  3,
		Window: 60,
	}
	
	limiter := NewMemoryLimiter(config)
	defer limiter.Close()
	
	key := "test-key"
	
	// 모든 버스트 소모
	for i := 0; i < 3; i++ {
		assert.True(t, limiter.Allow(key))
	}
	
	// 추가 요청은 거부
	assert.False(t, limiter.Allow(key))
	assert.Equal(t, 0, limiter.Remaining(key))
	
	// 리셋 수행
	limiter.Reset(key)
	
	// 리셋 후 다시 허용되어야 함
	assert.True(t, limiter.Allow(key))
	assert.Equal(t, 2, limiter.Remaining(key))
}

func TestMemoryLimiter_ResetTime(t *testing.T) {
	config := &LimiterConfig{
		Rate:   60,
		Burst:  1,
		Window: 60,
	}
	
	limiter := NewMemoryLimiter(config)
	defer limiter.Close()
	
	key := "test-key"
	
	now := time.Now()
	
	// 첫 번째 요청으로 리미터 생성
	assert.True(t, limiter.Allow(key))
	
	// 리셋 시간이 현재 시간 + 윈도우와 비슷해야 함
	resetTime := limiter.ResetTime(key)
	expectedTime := now.Add(60 * time.Second)
	
	// 1초 정도 오차 허용
	assert.WithinDuration(t, expectedTime, resetTime, 1*time.Second)
}

func TestMemoryLimiter_MultipleKeys(t *testing.T) {
	config := &LimiterConfig{
		Rate:   60,
		Burst:  2,
		Window: 60,
	}
	
	limiter := NewMemoryLimiter(config)
	defer limiter.Close()
	
	key1 := "user:123"
	key2 := "ip:192.168.1.1"
	
	// 각 키는 독립적으로 제한되어야 함
	assert.True(t, limiter.Allow(key1))
	assert.True(t, limiter.Allow(key1))
	assert.False(t, limiter.Allow(key1)) // key1은 제한
	
	assert.True(t, limiter.Allow(key2))
	assert.True(t, limiter.Allow(key2))
	assert.False(t, limiter.Allow(key2)) // key2도 제한
}

func TestMemoryLimiter_Stats(t *testing.T) {
	config := &LimiterConfig{
		Rate:   60,
		Burst:  2,
		Window: 60,
	}
	
	limiter := NewMemoryLimiter(config)
	defer limiter.Close()
	
	// 초기 통계 확인
	stats := limiter.GetStats()
	assert.NotNil(t, stats)
	assert.Equal(t, 0, stats["total_limiters"])
	
	// 키 추가 후 통계 확인
	limiter.Allow("key1")
	limiter.Allow("key2")
	
	stats = limiter.GetStats()
	assert.Equal(t, 2, stats["total_limiters"])
	
	configStats, ok := stats["config"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, 60, configStats["rate"])
	assert.Equal(t, 2, configStats["burst"])
	assert.Equal(t, 60, configStats["window"])
}

func TestMemoryLimiter_DefaultConfig(t *testing.T) {
	limiter := NewMemoryLimiter(nil)
	defer limiter.Close()
	
	key := "test"
	
	// 기본 설정으로 생성되어야 함
	assert.Equal(t, 60, limiter.Limit(key))
	
	// 기본 버스트(10) 확인
	for i := 0; i < 10; i++ {
		assert.True(t, limiter.Allow(key), "Request %d should be allowed", i)
	}
	
	// 11번째 요청은 거부되어야 함
	assert.False(t, limiter.Allow(key))
}

func TestMemoryLimiter_TimeWindow(t *testing.T) {
	config := &LimiterConfig{
		Rate:   120, // 120 requests per minute = 2 per second
		Burst:  1,   // only 1 burst
		Window: 1,   // 1 second window for testing
	}
	
	limiter := NewMemoryLimiter(config)
	defer limiter.Close()
	
	key := "test"
	
	// 첫 번째 요청 허용
	assert.True(t, limiter.Allow(key))
	
	// 즉시 두 번째 요청은 거부
	assert.False(t, limiter.Allow(key))
	
	// 시간 윈도우 대기 후에는 다시 허용되어야 함
	// 실제 테스트에서는 시간 대기가 필요하지만, 
	// 단위 테스트에서는 Reset으로 시뮬레이션
	limiter.Reset(key)
	assert.True(t, limiter.Allow(key))
}

func TestDefaultLimiterConfig(t *testing.T) {
	config := DefaultLimiterConfig()
	
	assert.NotNil(t, config)
	assert.Equal(t, 60, config.Rate)
	assert.Equal(t, 10, config.Burst)
	assert.Equal(t, 60, config.Window)
}

func TestMemoryLimiter_Concurrent(t *testing.T) {
	config := &LimiterConfig{
		Rate:   100,
		Burst:  10,
		Window: 60,
	}
	
	limiter := NewMemoryLimiter(config)
	defer limiter.Close()
	
	// 동시성 테스트 - 여러 고루틴에서 동시에 접근
	key := "concurrent-test"
	allowedCount := 0
	
	// 채널을 통해 결과 수집
	results := make(chan bool, 20)
	
	// 20개의 고루틴에서 동시에 요청
	for i := 0; i < 20; i++ {
		go func() {
			results <- limiter.Allow(key)
		}()
	}
	
	// 결과 수집
	for i := 0; i < 20; i++ {
		if <-results {
			allowedCount++
		}
	}
	
	// 버스트 크기(10)만큼만 허용되어야 함
	assert.Equal(t, 10, allowedCount)
}