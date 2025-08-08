package performance

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"
)

// TestMemoryManager 메모리 관리자 테스트
func TestMemoryManager(t *testing.T) {
	config := &MemoryConfig{
		MaxPoolSize:     10,
		BufferSizes:     []int{512, 1024, 2048},
		MaxBufferSize:   4096,
		GCOptimization:  true,
		MonitorInterval: 0, // 모니터 비활성화
		MemoryLimit:     100 * 1024 * 1024, // 100MB
	}

	mm := NewMemoryManager(config)
	defer mm.Shutdown()

	t.Run("ObjectPool", func(t *testing.T) {
		// PTY 세션 객체 가져오기
		obj1 := mm.GetObject(ObjectPTYSession)
		if obj1 == nil {
			t.Error("Failed to get PTY session object")
		}

		// 객체 반환
		mm.ReturnObject(ObjectPTYSession, obj1)

		// 재사용 확인
		obj2 := mm.GetObject(ObjectPTYSession)
		if obj2 == nil {
			t.Error("Failed to get reused PTY session object")
		}

		// 메트릭 확인
		metrics := mm.GetMetrics()
		if metrics.PoolHitRate == 0 {
			t.Error("Pool hit rate should be greater than 0")
		}
	})

	t.Run("BufferPool", func(t *testing.T) {
		// 버퍼 가져오기
		buf1 := mm.GetBuffer(1024)
		if len(buf1) != 1024 {
			t.Errorf("Expected buffer size 1024, got %d", len(buf1))
		}

		// 버퍼 반환
		mm.ReturnBuffer(buf1)

		// 재사용 확인
		buf2 := mm.GetBuffer(1024)
		if len(buf2) != 1024 {
			t.Errorf("Expected buffer size 1024, got %d", len(buf2))
		}
	})

	t.Run("CustomAllocator", func(t *testing.T) {
		// 메모리 할당
		data, err := mm.Allocate(1024)
		if err != nil {
			t.Errorf("Failed to allocate memory: %v", err)
		}

		if len(data) != 1024 {
			t.Errorf("Expected allocated size 1024, got %d", len(data))
		}

		// 메모리 해제
		mm.Free(data)
	})

	t.Run("Optimization", func(t *testing.T) {
		// 최적화 실행
		mm.Optimize()

		// GC 최적화 확인
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)
		
		if memStats.NumGC == 0 {
			t.Log("Warning: No GC cycles detected")
		}
	})
}

// TestObjectPoolConcurrency 객체 풀 동시성 테스트
func TestObjectPoolConcurrency(t *testing.T) {
	config := DefaultMemoryConfig()
	mm := NewMemoryManager(config)
	defer mm.Shutdown()

	const goroutines = 100
	const iterations = 1000

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()

			for j := 0; j < iterations; j++ {
				// 다양한 객체 타입 테스트
				objTypes := []ObjectType{
					ObjectPTYSession,
					ObjectWebSocketConn,
					ObjectTerminalScreen,
					ObjectANSICommand,
					ObjectSnapshot,
				}

				for _, objType := range objTypes {
					obj := mm.GetObject(objType)
					if obj == nil {
						t.Errorf("Failed to get object of type %v", objType)
						continue
					}

					// 짧은 작업 시뮬레이션
					time.Sleep(time.Microsecond)

					mm.ReturnObject(objType, obj)
				}
			}
		}()
	}

	wg.Wait()

	// 메트릭 확인
	metrics := mm.GetMetrics()
	t.Logf("Pool hit rate: %.2f%%", metrics.PoolHitRate*100)
	t.Logf("Current memory usage: %d bytes", metrics.CurrentUsage)
	t.Logf("Total allocated: %d bytes", metrics.TotalAllocated)
}

// TestBufferPoolSizes 버퍼 풀 크기별 테스트
func TestBufferPoolSizes(t *testing.T) {
	config := DefaultMemoryConfig()
	mm := NewMemoryManager(config)
	defer mm.Shutdown()

	testSizes := []int{256, 512, 1024, 2048, 4096, 8192, 16384, 32768, 65536}

	for _, size := range testSizes {
		t.Run(fmt.Sprintf("Size_%d", size), func(t *testing.T) {
			buf := mm.GetBuffer(size)
			if len(buf) != size {
				t.Errorf("Expected buffer size %d, got %d", size, len(buf))
			}

			// 버퍼에 데이터 쓰기
			for i := range buf {
				buf[i] = byte(i % 256)
			}

			// 버퍼 반환
			mm.ReturnBuffer(buf)

			// 재사용 확인
			buf2 := mm.GetBuffer(size)
			if len(buf2) != size {
				t.Errorf("Expected reused buffer size %d, got %d", size, len(buf2))
			}

			// 버퍼가 초기화되었는지 확인
			allZero := true
			for _, b := range buf2 {
				if b != 0 {
					allZero = false
					break
				}
			}

			if !allZero {
				t.Error("Buffer was not properly cleared before reuse")
			}

			mm.ReturnBuffer(buf2)
		})
	}
}

// TestMemoryLimit 메모리 제한 테스트
func TestMemoryLimit(t *testing.T) {
	config := &MemoryConfig{
		MaxPoolSize:     10,
		BufferSizes:     []int{1024},
		MaxBufferSize:   1024,
		GCOptimization:  false,
		MonitorInterval: 0,
		MemoryLimit:     1024, // 1KB 제한
	}

	mm := NewMemoryManager(config)
	defer mm.Shutdown()

	// 첫 번째 할당 (성공해야 함)
	data1, err := mm.Allocate(512)
	if err != nil {
		t.Errorf("First allocation should succeed: %v", err)
	}

	// 두 번째 할당 (성공해야 함)
	data2, err := mm.Allocate(512)
	if err != nil {
		t.Errorf("Second allocation should succeed: %v", err)
	}

	// 세 번째 할당 (실패해야 함 - 제한 초과)
	_, err = mm.Allocate(512)
	if err == nil {
		t.Error("Third allocation should fail due to memory limit")
	}

	// 메모리 해제
	mm.Free(data1)
	mm.Free(data2)

	// 해제 후 다시 할당 (성공해야 함)
	data3, err := mm.Allocate(1024)
	if err != nil {
		t.Errorf("Allocation after free should succeed: %v", err)
	}

	mm.Free(data3)
}

// BenchmarkObjectPool 객체 풀 벤치마크
func BenchmarkObjectPool(b *testing.B) {
	config := DefaultMemoryConfig()
	mm := NewMemoryManager(config)
	defer mm.Shutdown()

	b.ResetTimer()

	b.Run("Get_Return", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			obj := mm.GetObject(ObjectPTYSession)
			mm.ReturnObject(ObjectPTYSession, obj)
		}
	})

	b.Run("Parallel", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				obj := mm.GetObject(ObjectPTYSession)
				mm.ReturnObject(ObjectPTYSession, obj)
			}
		})
	})
}

// BenchmarkBufferPool 버퍼 풀 벤치마크
func BenchmarkBufferPool(b *testing.B) {
	config := DefaultMemoryConfig()
	mm := NewMemoryManager(config)
	defer mm.Shutdown()

	b.ResetTimer()

	b.Run("Small_Buffer", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			buf := mm.GetBuffer(512)
			mm.ReturnBuffer(buf)
		}
	})

	b.Run("Medium_Buffer", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			buf := mm.GetBuffer(4096)
			mm.ReturnBuffer(buf)
		}
	})

	b.Run("Large_Buffer", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			buf := mm.GetBuffer(65536)
			mm.ReturnBuffer(buf)
		}
	})
}

// BenchmarkAllocator 할당자 벤치마크
func BenchmarkAllocator(b *testing.B) {
	config := DefaultMemoryConfig()
	mm := NewMemoryManager(config)
	defer mm.Shutdown()

	b.ResetTimer()

	b.Run("Allocate_Free", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			data, _ := mm.Allocate(1024)
			mm.Free(data)
		}
	})

	b.Run("Parallel_Allocate", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				data, _ := mm.Allocate(1024)
				mm.Free(data)
			}
		})
	})
}