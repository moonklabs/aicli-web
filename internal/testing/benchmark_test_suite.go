package testing

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// BenchmarkTestSuite는 벤치마크 테스트 스위트입니다
type BenchmarkTestSuite struct {
	config  BenchmarkConfig
	results []BenchmarkResult
	mutex   sync.RWMutex
	running atomic.Bool
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// BenchmarkConfig는 벤치마크 설정입니다
type BenchmarkConfig struct {
	// 기본 설정
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Duration    time.Duration `json:"duration"`
	Iterations  int           `json:"iterations"`

	// 동시성 설정
	Parallelism int `json:"parallelism"`

	// 리소스 제한
	MaxMemory     uint64 `json:"max_memory"`
	MaxGoroutines int    `json:"max_goroutines"`

	// 워밍업
	WarmupDuration   time.Duration `json:"warmup_duration"`
	WarmupIterations int           `json:"warmup_iterations"`
}

// BenchmarkResult는 벤치마크 결과입니다
type BenchmarkResult struct {
	TestName            string        `json:"test_name"`
	TotalOperations     int64         `json:"total_operations"`
	TotalTime           time.Duration `json:"total_time"`
	OperationsPerSecond float64       `json:"operations_per_second"`
	AverageLatency      time.Duration `json:"average_latency"`
	MinLatency          time.Duration `json:"min_latency"`
	MaxLatency          time.Duration `json:"max_latency"`
	P50Latency          time.Duration `json:"p50_latency"`
	P95Latency          time.Duration `json:"p95_latency"`
	P99Latency          time.Duration `json:"p99_latency"`

	// 메모리 통계
	AllocatedMemory  uint64 `json:"allocated_memory"`
	AllocationsCount int64  `json:"allocations_count"`

	// 시스템 리소스
	GoroutineCount int           `json:"goroutine_count"`
	CPUTime        time.Duration `json:"cpu_time"`

	// 에러 통계
	ErrorCount int64   `json:"error_count"`
	ErrorRate  float64 `json:"error_rate"`

	Timestamp time.Time `json:"timestamp"`
}

// BenchmarkFunction은 벤치마크할 함수 타입입니다
type BenchmarkFunction func(ctx context.Context, iteration int) error

// BenchmarkSuite는 벤치마크 테스트들의 모음입니다
type BenchmarkSuite struct {
	Name         string                       `json:"name"`
	Tests        map[string]BenchmarkFunction `json:"-"`
	SetupFunc    func() error                 `json:"-"`
	TeardownFunc func() error                 `json:"-"`
}

// DefaultBenchmarkConfig는 기본 벤치마크 설정을 반환합니다
func DefaultBenchmarkConfig() BenchmarkConfig {
	return BenchmarkConfig{
		Name:             "Default Benchmark",
		Description:      "기본 벤치마크 테스트",
		Duration:         30 * time.Second,
		Iterations:       1000,
		Parallelism:      runtime.NumCPU(),
		MaxMemory:        100 * 1024 * 1024, // 100MB
		MaxGoroutines:    1000,
		WarmupDuration:   5 * time.Second,
		WarmupIterations: 100,
	}
}

// NewBenchmarkTestSuite는 새로운 벤치마크 테스트 스위트를 생성합니다
func NewBenchmarkTestSuite(config BenchmarkConfig) *BenchmarkTestSuite {
	ctx, cancel := context.WithCancel(context.Background())

	return &BenchmarkTestSuite{
		config:  config,
		results: make([]BenchmarkResult, 0),
		ctx:     ctx,
		cancel:  cancel,
	}
}

// RunBenchmark는 단일 벤치마크를 실행합니다
func (bts *BenchmarkTestSuite) RunBenchmark(name string, fn BenchmarkFunction) (*BenchmarkResult, error) {
	if !bts.running.CompareAndSwap(false, true) {
		return nil, fmt.Errorf("benchmark suite is already running")
	}
	defer bts.running.Store(false)

	fmt.Printf("벤치마크 시작: %s\n", name)

	// 워밍업 실행
	if bts.config.WarmupDuration > 0 || bts.config.WarmupIterations > 0 {
		fmt.Printf("워밍업 실행 중...\n")
		bts.runWarmup(fn)
	}

	// 초기 메모리 상태 기록
	var startMemStats, endMemStats runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&startMemStats)

	startTime := time.Now()
	startGoroutines := runtime.NumGoroutine()

	// 벤치마크 실행
	result := bts.executeBenchmark(name, fn)

	endTime := time.Now()
	runtime.ReadMemStats(&endMemStats)
	endGoroutines := runtime.NumGoroutine()

	// 결과 계산
	result.TotalTime = endTime.Sub(startTime)
	result.AllocatedMemory = endMemStats.TotalAlloc - startMemStats.TotalAlloc
	result.AllocationsCount = int64(endMemStats.Mallocs - startMemStats.Mallocs)
	result.GoroutineCount = endGoroutines - startGoroutines
	result.Timestamp = startTime

	if result.TotalOperations > 0 {
		result.OperationsPerSecond = float64(result.TotalOperations) / result.TotalTime.Seconds()
		result.ErrorRate = float64(result.ErrorCount) / float64(result.TotalOperations) * 100.0
	}

	// 결과 저장
	bts.mutex.Lock()
	bts.results = append(bts.results, *result)
	bts.mutex.Unlock()

	fmt.Printf("벤치마크 완료: %s - %.2f ops/sec, %v 평균 지연\n",
		name, result.OperationsPerSecond, result.AverageLatency)

	return result, nil
}

// RunBenchmarkSuite는 벤치마크 스위트를 실행합니다
func (bts *BenchmarkTestSuite) RunBenchmarkSuite(suite BenchmarkSuite) (map[string]*BenchmarkResult, error) {
	fmt.Printf("벤치마크 스위트 시작: %s\n", suite.Name)

	results := make(map[string]*BenchmarkResult)

	// 설정 함수 실행
	if suite.SetupFunc != nil {
		if err := suite.SetupFunc(); err != nil {
			return nil, fmt.Errorf("setup failed: %w", err)
		}
	}

	// 정리 함수는 defer로 실행
	if suite.TeardownFunc != nil {
		defer func() {
			if err := suite.TeardownFunc(); err != nil {
				fmt.Printf("Teardown error: %v\n", err)
			}
		}()
	}

	// 각 테스트 실행
	for testName, testFunc := range suite.Tests {
		result, err := bts.RunBenchmark(testName, testFunc)
		if err != nil {
			fmt.Printf("벤치마크 실패: %s - %v\n", testName, err)
			continue
		}
		results[testName] = result
	}

	return results, nil
}

// 내부 메서드들

func (bts *BenchmarkTestSuite) runWarmup(fn BenchmarkFunction) {
	warmupCtx, warmupCancel := context.WithTimeout(bts.ctx, bts.config.WarmupDuration)
	defer warmupCancel()

	iterations := 0
	for {
		select {
		case <-warmupCtx.Done():
			return
		default:
			if bts.config.WarmupIterations > 0 && iterations >= bts.config.WarmupIterations {
				return
			}

			fn(warmupCtx, iterations)
			iterations++
		}
	}
}

func (bts *BenchmarkTestSuite) executeBenchmark(name string, fn BenchmarkFunction) *BenchmarkResult {
	result := &BenchmarkResult{
		TestName:   name,
		MinLatency: time.Duration(^uint64(0) >> 1), // 최대값으로 초기화
	}

	latencies := make([]time.Duration, 0, bts.config.Iterations)
	totalOperations := int64(0)
	errorCount := int64(0)

	// 동시성 제어
	semaphore := make(chan struct{}, bts.config.Parallelism)
	var wg sync.WaitGroup
	var mu sync.Mutex

	// 시간 기반 또는 반복 기반 실행
	benchCtx := bts.ctx
	if bts.config.Duration > 0 {
		var cancel context.CancelFunc
		benchCtx, cancel = context.WithTimeout(bts.ctx, bts.config.Duration)
		defer cancel()
	}

	iteration := 0
	for {
		select {
		case <-benchCtx.Done():
			goto done
		default:
			if bts.config.Iterations > 0 && iteration >= bts.config.Iterations {
				goto done
			}

			// 고루틴 수 제한 확인
			if runtime.NumGoroutine() > bts.config.MaxGoroutines {
				time.Sleep(time.Millisecond)
				continue
			}

			semaphore <- struct{}{}
			wg.Add(1)

			go func(iter int) {
				defer wg.Done()
				defer func() { <-semaphore }()

				startTime := time.Now()
				err := fn(benchCtx, iter)
				latency := time.Since(startTime)

				mu.Lock()
				totalOperations++
				if err != nil {
					errorCount++
				}

				latencies = append(latencies, latency)

				// 최소/최대 지연 시간 업데이트
				if latency < result.MinLatency {
					result.MinLatency = latency
				}
				if latency > result.MaxLatency {
					result.MaxLatency = latency
				}
				mu.Unlock()
			}(iteration)

			iteration++
		}
	}

done:
	wg.Wait()

	result.TotalOperations = totalOperations
	result.ErrorCount = errorCount

	// 지연 시간 통계 계산
	if len(latencies) > 0 {
		result.AverageLatency = bts.calculateAverageLatency(latencies)
		result.P50Latency = bts.calculatePercentile(latencies, 50)
		result.P95Latency = bts.calculatePercentile(latencies, 95)
		result.P99Latency = bts.calculatePercentile(latencies, 99)
	}

	return result
}

func (bts *BenchmarkTestSuite) calculateAverageLatency(latencies []time.Duration) time.Duration {
	if len(latencies) == 0 {
		return 0
	}

	total := time.Duration(0)
	for _, latency := range latencies {
		total += latency
	}

	return total / time.Duration(len(latencies))
}

func (bts *BenchmarkTestSuite) calculatePercentile(latencies []time.Duration, percentile int) time.Duration {
	if len(latencies) == 0 {
		return 0
	}

	// 지연 시간 정렬
	sortedLatencies := make([]time.Duration, len(latencies))
	copy(sortedLatencies, latencies)

	// 간단한 버블 정렬
	for i := 0; i < len(sortedLatencies)-1; i++ {
		for j := i + 1; j < len(sortedLatencies); j++ {
			if sortedLatencies[i] > sortedLatencies[j] {
				sortedLatencies[i], sortedLatencies[j] = sortedLatencies[j], sortedLatencies[i]
			}
		}
	}

	index := len(sortedLatencies) * percentile / 100
	if index >= len(sortedLatencies) {
		index = len(sortedLatencies) - 1
	}

	return sortedLatencies[index]
}

// GetResults는 현재까지의 벤치마크 결과를 반환합니다
func (bts *BenchmarkTestSuite) GetResults() []BenchmarkResult {
	bts.mutex.RLock()
	defer bts.mutex.RUnlock()

	results := make([]BenchmarkResult, len(bts.results))
	copy(results, bts.results)
	return results
}

// ClearResults는 모든 벤치마크 결과를 지웁니다
func (bts *BenchmarkTestSuite) ClearResults() {
	bts.mutex.Lock()
	defer bts.mutex.Unlock()

	bts.results = make([]BenchmarkResult, 0)
}

// Stop은 실행 중인 벤치마크를 중지합니다
func (bts *BenchmarkTestSuite) Stop() {
	bts.cancel()
}

// GenerateReport는 벤치마크 보고서를 생성합니다
func (bts *BenchmarkTestSuite) GenerateReport() *BenchmarkReport {
	bts.mutex.RLock()
	results := make([]BenchmarkResult, len(bts.results))
	copy(results, bts.results)
	bts.mutex.RUnlock()

	report := &BenchmarkReport{
		SuiteName:   bts.config.Name,
		Config:      bts.config,
		Results:     results,
		Summary:     bts.calculateSummary(results),
		GeneratedAt: time.Now(),
	}

	return report
}

// BenchmarkReport는 벤치마크 보고서입니다
type BenchmarkReport struct {
	SuiteName   string            `json:"suite_name"`
	Config      BenchmarkConfig   `json:"config"`
	Results     []BenchmarkResult `json:"results"`
	Summary     BenchmarkSummary  `json:"summary"`
	GeneratedAt time.Time         `json:"generated_at"`
}

// BenchmarkSummary는 벤치마크 요약입니다
type BenchmarkSummary struct {
	TotalTests          int           `json:"total_tests"`
	TotalOperations     int64         `json:"total_operations"`
	TotalTime           time.Duration `json:"total_time"`
	AverageOpsPerSecond float64       `json:"average_ops_per_second"`
	TotalAllocations    int64         `json:"total_allocations"`
	TotalMemory         uint64        `json:"total_memory"`
	AverageLatency      time.Duration `json:"average_latency"`
	BestPerformance     string        `json:"best_performance"`
	WorstPerformance    string        `json:"worst_performance"`
}

func (bts *BenchmarkTestSuite) calculateSummary(results []BenchmarkResult) BenchmarkSummary {
	summary := BenchmarkSummary{
		TotalTests: len(results),
	}

	if len(results) == 0 {
		return summary
	}

	totalLatency := time.Duration(0)
	bestOps := 0.0
	worstOps := float64(^uint64(0) >> 1) // 최대값
	bestTest := ""
	worstTest := ""

	for _, result := range results {
		summary.TotalOperations += result.TotalOperations
		summary.TotalTime += result.TotalTime
		summary.TotalAllocations += result.AllocationsCount
		summary.TotalMemory += result.AllocatedMemory
		totalLatency += result.AverageLatency

		if result.OperationsPerSecond > bestOps {
			bestOps = result.OperationsPerSecond
			bestTest = result.TestName
		}

		if result.OperationsPerSecond < worstOps {
			worstOps = result.OperationsPerSecond
			worstTest = result.TestName
		}
	}

	summary.AverageOpsPerSecond = float64(summary.TotalOperations) / summary.TotalTime.Seconds()
	summary.AverageLatency = totalLatency / time.Duration(len(results))
	summary.BestPerformance = fmt.Sprintf("%s (%.2f ops/sec)", bestTest, bestOps)
	summary.WorstPerformance = fmt.Sprintf("%s (%.2f ops/sec)", worstTest, worstOps)

	return summary
}
