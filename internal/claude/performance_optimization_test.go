package claude

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/aicli-web/internal/cache"
	"github.com/aicli-web/internal/profiling"
)

// 메모리 풀 테스트

func TestMemoryPoolManager(t *testing.T) {
	config := DefaultMemoryPoolConfig()
	manager := NewOptimizedMemoryManager(config)
	
	if err := manager.Start(); err != nil {
		t.Fatalf("Failed to start memory manager: %v", err)
	}
	defer manager.Stop()
	
	// 메시지 풀 테스트
	messagePool := manager.GetMessagePool()
	if messagePool == nil {
		t.Fatal("Message pool should not be nil")
	}
	
	// 메시지 생성 및 반환
	msg := messagePool.GetMessage()
	if msg == nil {
		t.Fatal("Message should not be nil")
	}
	
	msg.Type = "test"
	msg.Data["key"] = "value"
	
	messagePool.PutMessage(msg)
	
	// 재사용 확인
	msg2 := messagePool.GetMessage()
	if msg2 == nil {
		t.Fatal("Reused message should not be nil")
	}
	
	// 초기화 확인
	if msg2.Type != "" {
		t.Error("Message should be reset")
	}
	
	if len(msg2.Data) != 0 {
		t.Error("Message data should be reset")
	}
}

func TestBufferPool(t *testing.T) {
	config := DefaultMemoryPoolConfig()
	manager := NewOptimizedMemoryManager(config)
	
	if err := manager.Start(); err != nil {
		t.Fatalf("Failed to start memory manager: %v", err)
	}
	defer manager.Stop()
	
	bufferPool := manager.GetBufferPool()
	if bufferPool == nil {
		t.Fatal("Buffer pool should not be nil")
	}
	
	// 다양한 크기의 버퍼 테스트
	sizes := []int{1024, 4096, 16384}
	
	for _, size := range sizes {
		buf := bufferPool.GetBuffer(size)
		if buf == nil {
			t.Fatalf("Buffer should not be nil for size %d", size)
		}
		
		if buf.Capacity < size {
			t.Errorf("Buffer capacity %d should be at least %d", buf.Capacity, size)
		}
		
		bufferPool.PutBuffer(buf)
	}
}

func TestMemoryPoolStats(t *testing.T) {
	config := DefaultMemoryPoolConfig()
	manager := NewOptimizedMemoryManager(config)
	
	if err := manager.Start(); err != nil {
		t.Fatalf("Failed to start memory manager: %v", err)
	}
	defer manager.Stop()
	
	// 몇 개의 객체 사용
	messagePool := manager.GetMessagePool()
	for i := 0; i < 10; i++ {
		msg := messagePool.GetMessage()
		messagePool.PutMessage(msg)
	}
	
	stats := manager.GetPoolStatistics()
	
	if stats.MessagePool.Gets.Load() != 10 {
		t.Errorf("Expected 10 gets, got %d", stats.MessagePool.Gets.Load())
	}
	
	if stats.MessagePool.Puts.Load() != 10 {
		t.Errorf("Expected 10 puts, got %d", stats.MessagePool.Puts.Load())
	}
	
	if stats.HitRate <= 0 {
		t.Error("Hit rate should be positive")
	}
}

// 고루틴 매니저 테스트

func TestWorkerPoolManager(t *testing.T) {
	config := DefaultWorkerPoolConfig()
	config.MinWorkers = 2
	config.MaxWorkers = 5
	
	manager := NewWorkerPoolManager(config)
	
	if err := manager.Start(); err != nil {
		t.Fatalf("Failed to start worker pool: %v", err)
	}
	defer manager.Stop()
	
	// 활성 고루틴 수 확인
	activeCount := manager.GetActiveGoroutines()
	if activeCount < 0 {
		t.Error("Active goroutines count should be non-negative")
	}
	
	stats := manager.GetGoroutineStats()
	if stats.Total < config.MinWorkers {
		t.Errorf("Total workers %d should be at least %d", stats.Total, config.MinWorkers)
	}
}

// 간단한 태스크 구현
type TestTask struct {
	name     string
	duration time.Duration
	priority TaskPriority
}

func (t *TestTask) Execute(ctx context.Context) error {
	time.Sleep(t.duration)
	return nil
}

func (t *TestTask) GetPriority() TaskPriority {
	return t.priority
}

func (t *TestTask) GetEstimatedDuration() time.Duration {
	return t.duration
}

func (t *TestTask) GetDescription() string {
	return t.name
}

func TestWorkerPoolTaskExecution(t *testing.T) {
	config := DefaultWorkerPoolConfig()
	config.MinWorkers = 1
	config.MaxWorkers = 3
	config.TaskTimeout = time.Second
	
	manager := NewWorkerPoolManager(config)
	
	if err := manager.Start(); err != nil {
		t.Fatalf("Failed to start worker pool: %v", err)
	}
	defer manager.Stop()
	
	// 태스크 실행
	task := &TestTask{
		name:     "test_task",
		duration: 100 * time.Millisecond,
		priority: TaskPriorityNormal,
	}
	
	if err := manager.SpawnWorker(task); err != nil {
		t.Fatalf("Failed to spawn worker: %v", err)
	}
	
	// 완료 대기
	time.Sleep(200 * time.Millisecond)
	
	stats := manager.GetGoroutineStats()
	if stats.Completed == 0 {
		t.Error("At least one task should be completed")
	}
}

func TestWorkerPoolConcurrency(t *testing.T) {
	config := DefaultWorkerPoolConfig()
	config.MinWorkers = 2
	config.MaxWorkers = 5
	
	manager := NewWorkerPoolManager(config)
	
	if err := manager.Start(); err != nil {
		t.Fatalf("Failed to start worker pool: %v", err)
	}
	defer manager.Stop()
	
	// 여러 태스크 동시 실행
	var wg sync.WaitGroup
	taskCount := 10
	
	for i := 0; i < taskCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			task := &TestTask{
				name:     fmt.Sprintf("concurrent_task_%d", id),
				duration: 50 * time.Millisecond,
				priority: TaskPriorityNormal,
			}
			
			if err := manager.SpawnWorker(task); err != nil {
				t.Errorf("Failed to spawn worker %d: %v", id, err)
			}
		}(i)
	}
	
	wg.Wait()
	
	// 완료 대기
	time.Sleep(500 * time.Millisecond)
	
	stats := manager.GetGoroutineStats()
	if stats.Completed < int64(taskCount) {
		t.Errorf("Expected at least %d completed tasks, got %d", taskCount, stats.Completed)
	}
}

// 캐시 매니저 테스트

func TestMultiLevelCacheManager(t *testing.T) {
	config := cache.DefaultCacheConfig()
	config.L1MaxSize = 1024 * 1024 // 1MB
	config.L2MaxSize = 10 * 1024 * 1024 // 10MB
	config.L2Directory = t.TempDir()
	
	manager, err := cache.NewMultiLevelCacheManager(config)
	if err != nil {
		t.Fatalf("Failed to create cache manager: %v", err)
	}
	
	if err := manager.Start(); err != nil {
		t.Fatalf("Failed to start cache manager: %v", err)
	}
	defer manager.Stop()
	
	// 기본 설정/조회 테스트
	key := "test_key"
	value := "test_value"
	
	if err := manager.Set(key, value, time.Minute); err != nil {
		t.Fatalf("Failed to set cache value: %v", err)
	}
	
	retrievedValue, exists := manager.Get(key)
	if !exists {
		t.Fatal("Cache value should exist")
	}
	
	if retrievedValue != value {
		t.Errorf("Expected %v, got %v", value, retrievedValue)
	}
}

func TestCacheHitMissRatio(t *testing.T) {
	config := cache.DefaultCacheConfig()
	config.L2Directory = t.TempDir()
	
	manager, err := cache.NewMultiLevelCacheManager(config)
	if err != nil {
		t.Fatalf("Failed to create cache manager: %v", err)
	}
	
	if err := manager.Start(); err != nil {
		t.Fatalf("Failed to start cache manager: %v", err)
	}
	defer manager.Stop()
	
	// 여러 키 설정
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("key_%d", i)
		value := fmt.Sprintf("value_%d", i)
		manager.Set(key, value, time.Minute)
	}
	
	// 히트 테스트
	hitCount := 0
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("key_%d", i)
		if _, exists := manager.Get(key); exists {
			hitCount++
		}
	}
	
	if hitCount != 10 {
		t.Errorf("Expected 10 hits, got %d", hitCount)
	}
	
	// 미스 테스트
	if _, exists := manager.Get("nonexistent_key"); exists {
		t.Error("Nonexistent key should not exist")
	}
	
	stats := manager.GetCacheStats()
	if stats.HitRate <= 0 {
		t.Error("Hit rate should be positive")
	}
}

func TestCacheEviction(t *testing.T) {
	config := cache.DefaultCacheConfig()
	config.L1MaxSize = 100 // 매우 작은 크기
	config.L1MaxEntries = 3
	config.L2Directory = t.TempDir()
	
	manager, err := cache.NewMultiLevelCacheManager(config)
	if err != nil {
		t.Fatalf("Failed to create cache manager: %v", err)
	}
	
	if err := manager.Start(); err != nil {
		t.Fatalf("Failed to start cache manager: %v", err)
	}
	defer manager.Stop()
	
	// 용량 초과하여 설정
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("key_%d", i)
		value := fmt.Sprintf("long_value_to_exceed_capacity_%d", i)
		manager.Set(key, value, time.Minute)
	}
	
	// 축출 확인
	stats := manager.GetCacheStats()
	if stats.L1Stats.Entries > config.L1MaxEntries {
		t.Errorf("L1 entries %d should not exceed max %d", stats.L1Stats.Entries, config.L1MaxEntries)
	}
}

// 성능 프로파일러 테스트

func TestPerformanceProfiler(t *testing.T) {
	config := profiling.DefaultProfilingConfig()
	config.OutputDir = t.TempDir()
	config.AutoCapture = false
	config.AutoCleanup = false
	
	profiler, err := profiling.NewPerformanceProfiler(config)
	if err != nil {
		t.Fatalf("Failed to create profiler: %v", err)
	}
	
	if err := profiler.Start(); err != nil {
		t.Fatalf("Failed to start profiler: %v", err)
	}
	defer profiler.Stop()
	
	// 프로파일 캡처
	report, err := profiler.Capture()
	if err != nil {
		t.Fatalf("Failed to capture profile: %v", err)
	}
	
	if report == nil {
		t.Fatal("Report should not be nil")
	}
	
	if len(report.Files) == 0 {
		t.Error("Report should contain profile files")
	}
}

func TestProfilerStats(t *testing.T) {
	config := profiling.DefaultProfilingConfig()
	config.OutputDir = t.TempDir()
	
	profiler, err := profiling.NewPerformanceProfiler(config)
	if err != nil {
		t.Fatalf("Failed to create profiler: %v", err)
	}
	
	if err := profiler.Start(); err != nil {
		t.Fatalf("Failed to start profiler: %v", err)
	}
	defer profiler.Stop()
	
	stats := profiler.GetCurrentStats()
	if stats == nil {
		t.Fatal("Stats should not be nil")
	}
	
	// 메모리 통계 확인
	if memStats, exists := stats["memory"]; exists {
		if memStats == nil {
			t.Error("Memory stats should not be nil")
		}
	}
	
	// 고루틴 통계 확인
	if goroutineStats, exists := stats["goroutine"]; exists {
		if goroutineStats == nil {
			t.Error("Goroutine stats should not be nil")
		}
	}
}

// 벤치마크 테스트

func BenchmarkMemoryPooling(b *testing.B) {
	config := DefaultMemoryPoolConfig()
	manager := NewOptimizedMemoryManager(config)
	manager.Start()
	defer manager.Stop()
	
	messagePool := manager.GetMessagePool()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			msg := messagePool.GetMessage()
			msg.Type = "benchmark"
			msg.Data["test"] = "value"
			messagePool.PutMessage(msg)
		}
	})
}

func BenchmarkWorkerPoolExecution(b *testing.B) {
	config := DefaultWorkerPoolConfig()
	config.MinWorkers = 4
	config.MaxWorkers = 8
	
	manager := NewWorkerPoolManager(config)
	manager.Start()
	defer manager.Stop()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			task := &TestTask{
				name:     "benchmark_task",
				duration: time.Microsecond,
				priority: TaskPriorityNormal,
			}
			manager.SpawnWorker(task)
		}
	})
}

func BenchmarkCacheOperations(b *testing.B) {
	config := cache.DefaultCacheConfig()
	config.L2Directory = b.TempDir()
	
	manager, err := cache.NewMultiLevelCacheManager(config)
	if err != nil {
		b.Fatalf("Failed to create cache manager: %v", err)
	}
	
	manager.Start()
	defer manager.Stop()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			key := fmt.Sprintf("bench_key_%d", b.N)
			value := fmt.Sprintf("bench_value_%d", b.N)
			
			manager.Set(key, value, time.Minute)
			manager.Get(key)
		}
	})
}

func BenchmarkBufferPooling(b *testing.B) {
	config := DefaultMemoryPoolConfig()
	manager := NewOptimizedMemoryManager(config)
	manager.Start()
	defer manager.Stop()
	
	bufferPool := manager.GetBufferPool()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf := bufferPool.GetBuffer(4096)
			// 버퍼 사용 시뮬레이션
			buf.Data = append(buf.Data, make([]byte, 1024)...)
			bufferPool.PutBuffer(buf)
		}
	})
}

// 메모리 사용량 테스트
func TestMemoryUsage(t *testing.T) {
	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)
	
	// 메모리 풀을 사용하지 않는 경우
	messages := make([]*PoolableMessage, 1000)
	for i := 0; i < 1000; i++ {
		messages[i] = &PoolableMessage{
			Type: "test",
			Data: make(map[string]interface{}),
			Timestamp: time.Now(),
		}
	}
	
	runtime.GC()
	runtime.ReadMemStats(&m2)
	
	withoutPooling := m2.HeapAlloc - m1.HeapAlloc
	
	// 메모리 초기화
	messages = nil
	runtime.GC()
	runtime.ReadMemStats(&m1)
	
	// 메모리 풀을 사용하는 경우
	config := DefaultMemoryPoolConfig()
	manager := NewOptimizedMemoryManager(config)
	manager.Start()
	defer manager.Stop()
	
	messagePool := manager.GetMessagePool()
	pooledMessages := make([]*PoolableMessage, 1000)
	
	for i := 0; i < 1000; i++ {
		pooledMessages[i] = messagePool.GetMessage()
		pooledMessages[i].Type = "test"
	}
	
	for i := 0; i < 1000; i++ {
		messagePool.PutMessage(pooledMessages[i])
	}
	
	runtime.GC()
	runtime.ReadMemStats(&m2)
	
	withPooling := m2.HeapAlloc - m1.HeapAlloc
	
	t.Logf("Memory usage without pooling: %d bytes", withoutPooling)
	t.Logf("Memory usage with pooling: %d bytes", withPooling)
	
	if withPooling > withoutPooling {
		t.Log("Warning: Pooling used more memory than expected (this may be normal for small allocations)")
	}
}