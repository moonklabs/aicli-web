package agent

import (
	"context"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/aicli/aicli-web/internal/models"
)

// BenchmarkAgentCreation 에이전트 생성 성능 벤치마크
func BenchmarkAgentCreation(b *testing.B) {
	b.Run("Sequential", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			agent := &models.Agent{
				ID:        generateTestID(),
				ProjectID: "test-project-id",
				Name:      "test-agent",
				Type:      models.AgentTypeClaude,
				Status:    models.AgentStatusCreated,
				CreatedAt: time.Now(),
			}
			_ = agent
		}
	})

	b.Run("Parallel", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				agent := &models.Agent{
					ID:        generateTestID(),
					ProjectID: "test-project-id",
					Name:      "test-agent",
					Type:      models.AgentTypeClaude,
					Status:    models.AgentStatusCreated,
					CreatedAt: time.Now(),
				}
				_ = agent
			}
		})
	})
}

// BenchmarkConcurrentAgents 동시 에이전트 처리 성능 벤치마크
func BenchmarkConcurrentAgents(b *testing.B) {
	tests := []struct {
		name        string
		concurrency int
	}{
		{"1_Agent", 1},
		{"10_Agents", 10},
		{"50_Agents", 50},
		{"100_Agents", 100},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			var wg sync.WaitGroup
			agents := make([]*models.Agent, tt.concurrency)
			
			// 에이전트 준비
			for i := 0; i < tt.concurrency; i++ {
				agents[i] = &models.Agent{
					ID:        generateTestID(),
					ProjectID: "benchmark-project-id",
					Name:      "benchmark-agent",
					Type:      models.AgentTypeClaude,
					Status:    models.AgentStatusCreated,
					CreatedAt: time.Now(),
				}
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				wg.Add(tt.concurrency)
				for j := 0; j < tt.concurrency; j++ {
					go func(agent *models.Agent) {
						defer wg.Done()
						// 가상의 작업 시뮬레이션
						agent.Status = models.AgentStatusRunning
						time.Sleep(time.Microsecond) // 최소한의 작업
						agent.Status = models.AgentStatusStopped
					}(agents[j])
				}
				wg.Wait()
			}
		})
	}
}

// BenchmarkMemoryAllocation 메모리 할당 성능 벤치마크
func BenchmarkMemoryAllocation(b *testing.B) {
	b.Run("AgentStructs", func(b *testing.B) {
		var m1, m2 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m1)
		
		b.ResetTimer()
		agents := make([]*models.Agent, b.N)
		for i := 0; i < b.N; i++ {
			agents[i] = &models.Agent{
				ID:        generateTestID(),
				ProjectID: "memory-test-project-id",
				Name:      "memory-test-agent",
				Type:      models.AgentTypeClaude,
				Status:    models.AgentStatusCreated,
				CreatedAt: time.Now(),
			}
		}
		b.StopTimer()

		runtime.GC()
		runtime.ReadMemStats(&m2)
		b.ReportMetric(float64(m2.TotalAlloc-m1.TotalAlloc)/float64(b.N), "bytes/op")
	})

	b.Run("AgentSlices", func(b *testing.B) {
		var m1, m2 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m1)
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			agents := make([]*models.Agent, 100)
			for j := 0; j < 100; j++ {
				agents[j] = &models.Agent{
					ID:        generateTestID(),
					ProjectID: "slice-test-project-id",
					Name:      "slice-test-agent",
					Type:      models.AgentTypeClaude,
					Status:    models.AgentStatusCreated,
				}
			}
			_ = agents
		}
		b.StopTimer()

		runtime.GC()
		runtime.ReadMemStats(&m2)
		b.ReportMetric(float64(m2.TotalAlloc-m1.TotalAlloc)/float64(b.N), "bytes/op")
	})
}

// BenchmarkContextOperations 컨텍스트 처리 성능 벤치마크
func BenchmarkContextOperations(b *testing.B) {
	b.Run("WithTimeout", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			select {
			case <-ctx.Done():
			default:
			}
			cancel()
		}
	})

	b.Run("WithCancel", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ctx, cancel := context.WithCancel(context.Background())
			select {
			case <-ctx.Done():
			default:
			}
			cancel()
		}
	})
}

// BenchmarkConcurrentMap 동시 맵 접근 성능 벤치마크
func BenchmarkConcurrentMap(b *testing.B) {
	agentMap := make(map[string]*models.Agent)
	var mutex sync.RWMutex

	// 초기 데이터 준비
	for i := 0; i < 1000; i++ {
		id := generateTestID()
		agentMap[id] = &models.Agent{
			ID:        id,
			ProjectID: "concurrent-test-project-id",
			Name:      "concurrent-test-agent",
			Type:      models.AgentTypeClaude,
			Status:    models.AgentStatusRunning,
		}
	}

	b.Run("ReadOnly", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				mutex.RLock()
				for range agentMap {
					break // 첫 번째 항목만 읽기
				}
				mutex.RUnlock()
			}
		})
	})

	b.Run("ReadWrite", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				if i%10 == 0 { // 10회 중 1회는 쓰기
					mutex.Lock()
					id := generateTestID()
					agentMap[id] = &models.Agent{
						ID:        id,
						ProjectID: "readwrite-test-project-id",
						Name:      "readwrite-test-agent",
						Type:      models.AgentTypeClaude,
						Status:    models.AgentStatusCreated,
					}
					mutex.Unlock()
				} else { // 나머지는 읽기
					mutex.RLock()
					for range agentMap {
						break
					}
					mutex.RUnlock()
				}
				i++
			}
		})
	})
}

// generateTestID 테스트용 ID 생성
func generateTestID() string {
	return time.Now().Format("20060102150405.000") + "-test"
}

// TestBaselinePerformanceMetrics 성능 베이스라인 메트릭 수집 테스트
func TestBaselinePerformanceMetrics(t *testing.T) {
	start := time.Now()
	
	var m1, m2 runtime.MemStats
	runtime.ReadMemStats(&m1)

	// 100개 에이전트 시뮬레이션
	agents := make([]*models.Agent, 100)
	for i := 0; i < 100; i++ {
		agents[i] = &models.Agent{
			ID:        generateTestID(),
			ProjectID: "perf-test-project-id",
			Name:      "perf-test-agent",
			Type:      models.AgentTypeClaude,
			Status:    models.AgentStatusRunning,
			CreatedAt: time.Now(),
		}
	}

	runtime.ReadMemStats(&m2)
	duration := time.Since(start)

	t.Logf("성능 베이스라인 메트릭:")
	t.Logf("- 100개 에이전트 생성 시간: %v", duration)
	t.Logf("- 에이전트당 평균 생성 시간: %v", duration/100)
	t.Logf("- 할당된 메모리: %d bytes", m2.TotalAlloc-m1.TotalAlloc)
	t.Logf("- 에이전트당 메모리 사용량: %d bytes", (m2.TotalAlloc-m1.TotalAlloc)/100)
	t.Logf("- 현재 고루틴 수: %d", runtime.NumGoroutine())
	t.Logf("- 현재 CPU 코어 수: %d", runtime.NumCPU())

	// 성능 목표 확인
	avgCreationTime := duration / 100
	if avgCreationTime > 50*time.Millisecond {
		t.Logf("경고: 에이전트 생성 시간이 목표(50ms)를 초과함: %v", avgCreationTime)
	}

	memoryPerAgent := (m2.TotalAlloc - m1.TotalAlloc) / 100
	if memoryPerAgent > 1024*1024 { // 1MB
		t.Logf("경고: 에이전트당 메모리 사용량이 목표(1MB)를 초과함: %d bytes", memoryPerAgent)
	}
}