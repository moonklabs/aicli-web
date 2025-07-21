package claude

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackpressureHandler(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	t.Run("Basic Drop Policy", func(t *testing.T) {
		config := BackpressureConfig{
			MaxBufferSize:     10,
			DropPolicy:        DropOldest,
			AdaptiveBuffering: false,
		}
		handler := NewBackpressureHandler(config, logger)

		// 버퍼 채우기
		for i := 0; i < 10; i++ {
			handler.IncrementBuffer()
		}

		assert.True(t, handler.ShouldDrop())
		assert.Equal(t, int64(10), handler.GetCurrentBufferSize())

		// 드롭 처리
		messages := []interface{}{1, 2, 3}
		result, err := handler.HandleDrop(messages)
		assert.NoError(t, err)
		assert.Len(t, result, 2) // 가장 오래된 것 제거
		assert.Equal(t, []interface{}{2, 3}, result)
	})

	t.Run("Block Until Ready", func(t *testing.T) {
		config := BackpressureConfig{
			MaxBufferSize:     5,
			DropPolicy:        BlockUntilReady,
			AdaptiveBuffering: false,
		}
		handler := NewBackpressureHandler(config, logger)

		// 버퍼 채우기
		for i := 0; i < 5; i++ {
			handler.IncrementBuffer()
		}

		assert.True(t, handler.ShouldDrop())

		// 비동기로 공간 확보
		go func() {
			time.Sleep(50 * time.Millisecond)
			handler.DecrementBuffer()
		}()

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		start := time.Now()
		err := handler.WaitForSpace(ctx)
		elapsed := time.Since(start)

		assert.NoError(t, err)
		assert.True(t, elapsed >= 50*time.Millisecond)
		assert.True(t, elapsed < 100*time.Millisecond)
	})

	t.Run("Adaptive Buffering", func(t *testing.T) {
		config := BackpressureConfig{
			MaxBufferSize:     100,
			DropPolicy:        DropNewest,
			AdaptiveBuffering: true,
			MinBufferSize:     50,
			BufferGrowthRate:  1.5,
			BufferShrinkRate:  0.8,
		}
		handler := NewBackpressureHandler(config, logger)

		// 고사용률 시뮬레이션
		for i := 0; i < 85; i++ {
			handler.IncrementBuffer()
		}

		originalSize := handler.GetMaxBufferSize()
		handler.AdjustBufferSize()
		newSize := handler.GetMaxBufferSize()

		assert.Greater(t, newSize, originalSize)
		assert.Equal(t, int(float64(originalSize)*1.5), newSize)

		// 버퍼 비우기
		for i := 0; i < 80; i++ {
			handler.DecrementBuffer()
		}

		// 충분한 시간 대기 후 재조정
		time.Sleep(1100 * time.Millisecond)
		handler.AdjustBufferSize()
		finalSize := handler.GetMaxBufferSize()

		assert.Less(t, finalSize, newSize)
	})

	t.Run("Slow Consumer Detection", func(t *testing.T) {
		config := BackpressureConfig{
			MaxBufferSize:     10,
			DropPolicy:        DropOldest,
			AdaptiveBuffering: true,
		}
		handler := NewBackpressureHandler(config, logger)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// 모니터링 시작
		go handler.MonitorSlowConsumers(ctx)

		// 느린 소비자 시뮬레이션
		for i := 0; i < 15; i++ {
			for j := 0; j < 10; j++ {
				handler.IncrementBuffer()
			}
			handler.ShouldDrop() // 백프레셔 이벤트 트리거
			time.Sleep(10 * time.Millisecond)
		}

		metrics := handler.GetMetrics()
		assert.Greater(t, metrics.SlowConsumerCount, int64(0))
		assert.Greater(t, metrics.BackpressureEvents, int64(0))
	})

	t.Run("Concurrent Access", func(t *testing.T) {
		config := BackpressureConfig{
			MaxBufferSize:     1000,
			DropPolicy:        DropNewest,
			AdaptiveBuffering: false,
		}
		handler := NewBackpressureHandler(config, logger)

		var wg sync.WaitGroup
		numGoroutines := 10
		opsPerGoroutine := 1000

		// 동시 접근 테스트
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < opsPerGoroutine; j++ {
					if id%2 == 0 {
						handler.IncrementBuffer()
						handler.ShouldDrop()
					} else {
						handler.DecrementBuffer()
					}
				}
			}(i)
		}

		wg.Wait()

		// 메트릭 확인
		metrics := handler.GetMetrics()
		assert.NotNil(t, metrics)
	})

	t.Run("Metrics Accuracy", func(t *testing.T) {
		config := BackpressureConfig{
			MaxBufferSize:     5,
			DropPolicy:        DropOldest,
			AdaptiveBuffering: false,
		}
		handler := NewBackpressureHandler(config, logger)

		// 정확한 메트릭 추적
		var expectedDrops int64
		var expectedEvents int64

		for i := 0; i < 10; i++ {
			handler.IncrementBuffer()
			if handler.ShouldDrop() {
				expectedEvents++
				messages := []interface{}{i}
				handler.HandleDrop(messages)
				expectedDrops++
			}
		}

		metrics := handler.GetMetrics()
		assert.Equal(t, expectedDrops, metrics.DroppedMessages)
		assert.Equal(t, expectedEvents, metrics.BackpressureEvents)
	})
}

func TestBackpressureIntegration(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	t.Run("Producer Consumer Balance", func(t *testing.T) {
		config := BackpressureConfig{
			MaxBufferSize:     100,
			DropPolicy:        BlockUntilReady,
			AdaptiveBuffering: false,
		}
		handler := NewBackpressureHandler(config, logger)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// 프로듀서
		producerDone := make(chan struct{})
		var produced int64
		go func() {
			defer close(producerDone)
			for {
				select {
				case <-ctx.Done():
					return
				default:
					if !handler.ShouldDrop() {
						handler.IncrementBuffer()
						atomic.AddInt64(&produced, 1)
					} else {
						handler.WaitForSpace(ctx)
					}
				}
			}
		}()

		// 소비자
		consumerDone := make(chan struct{})
		var consumed int64
		go func() {
			defer close(consumerDone)
			ticker := time.NewTicker(10 * time.Millisecond)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					if handler.GetCurrentBufferSize() > 0 {
						handler.DecrementBuffer()
						atomic.AddInt64(&consumed, 1)
					}
				}
			}
		}()

		// 실행
		time.Sleep(1 * time.Second)
		cancel()

		<-producerDone
		<-consumerDone

		// 검증
		producedCount := atomic.LoadInt64(&produced)
		consumedCount := atomic.LoadInt64(&consumed)
		
		t.Logf("Produced: %d, Consumed: %d", producedCount, consumedCount)
		
		// 버퍼 크기를 고려한 차이 검증
		diff := producedCount - consumedCount
		assert.True(t, diff >= 0 && diff <= 100, "Buffer overflow detected")
	})

	t.Run("Multiple Drop Policies", func(t *testing.T) {
		policies := []struct {
			name   string
			policy DropPolicy
		}{
			{"DropOldest", DropOldest},
			{"DropNewest", DropNewest},
			{"BlockUntilReady", BlockUntilReady},
		}

		for _, p := range policies {
			t.Run(p.name, func(t *testing.T) {
				config := BackpressureConfig{
					MaxBufferSize:     50,
					DropPolicy:        p.policy,
					AdaptiveBuffering: false,
				}
				handler := NewBackpressureHandler(config, logger)

				// 오버플로우 시뮬레이션
				for i := 0; i < 100; i++ {
					if p.policy == BlockUntilReady && handler.ShouldDrop() {
						// 일부 공간 확보
						for j := 0; j < 10; j++ {
							handler.DecrementBuffer()
						}
					}
					handler.IncrementBuffer()
				}

				metrics := handler.GetMetrics()
				if p.policy != BlockUntilReady {
					assert.Greater(t, metrics.BackpressureEvents, int64(0))
				}
			})
		}
	})
}

func BenchmarkBackpressureOperations(b *testing.B) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	b.Run("IncrementDecrement", func(b *testing.B) {
		config := BackpressureConfig{
			MaxBufferSize: 1000,
			DropPolicy:    DropOldest,
		}
		handler := NewBackpressureHandler(config, logger)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			handler.IncrementBuffer()
			handler.DecrementBuffer()
		}
	})

	b.Run("ShouldDrop", func(b *testing.B) {
		config := BackpressureConfig{
			MaxBufferSize: 100,
			DropPolicy:    DropNewest,
		}
		handler := NewBackpressureHandler(config, logger)

		// 반쯤 채우기
		for i := 0; i < 50; i++ {
			handler.IncrementBuffer()
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			handler.ShouldDrop()
		}
	})

	b.Run("ConcurrentAccess", func(b *testing.B) {
		config := BackpressureConfig{
			MaxBufferSize: 1000,
			DropPolicy:    DropOldest,
		}
		handler := NewBackpressureHandler(config, logger)

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				handler.IncrementBuffer()
				handler.ShouldDrop()
				handler.DecrementBuffer()
			}
		})
	})
}

func TestBackpressureReset(t *testing.T) {
	logger := logrus.New()
	config := BackpressureConfig{
		MaxBufferSize: 100,
		DropPolicy:    DropOldest,
	}
	handler := NewBackpressureHandler(config, logger)

	// 상태 설정
	for i := 0; i < 50; i++ {
		handler.IncrementBuffer()
	}
	handler.ShouldDrop() // 이벤트 생성

	// 리셋
	handler.Reset()

	// 검증
	assert.Equal(t, int64(0), handler.GetCurrentBufferSize())
	
	metrics := handler.GetMetrics()
	assert.Equal(t, int64(0), metrics.DroppedMessages)
	assert.Equal(t, int64(0), metrics.BackpressureEvents)
	assert.Equal(t, float64(0), metrics.AvgBufferUsage)
}