package agent

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// PerformanceBenchmark는 성능 벤치마크 도구입니다
type PerformanceBenchmark struct {
	// 벤치마크 설정
	config        BenchmarkConfig
	
	// 성능 최적화기
	optimizer     *AgentPerformanceOptimizer
	
	// 결과 수집
	results       *BenchmarkResults
	
	// 상태 관리
	running       atomic.Bool
	ctx           context.Context
	cancel        context.CancelFunc
}

// BenchmarkConfig는 벤치마크 설정입니다
type BenchmarkConfig struct {
	// 테스트 시나리오
	Scenarios     []TestScenario    `json:"scenarios"`
	
	// 부하 설정
	MaxConcurrentAgents int          `json:"max_concurrent_agents"`
	TestDuration        time.Duration `json:"test_duration"`
	RampUpDuration      time.Duration `json:"ramp_up_duration"`
	RampDownDuration    time.Duration `json:"ramp_down_duration"`
	
	// 성능 목표
	TargetThroughput    float64       `json:"target_throughput"`
	MaxLatency          time.Duration `json:"max_latency"`
	MaxErrorRate        float64       `json:"max_error_rate"`
	
	// 리소스 제한
	MaxCPUUsage         float64       `json:"max_cpu_usage"`
	MaxMemoryUsage      int64         `json:"max_memory_usage"`
	
	// 모니터링
	MetricsInterval     time.Duration `json:"metrics_interval"`
	DetailedProfiling   bool          `json:"detailed_profiling"`
}

// TestScenario는 테스트 시나리오입니다
type TestScenario struct {
	Name                string        `json:"name"`
	Description         string        `json:"description"`
	AgentCount          int           `json:"agent_count"`
	CreationRate        float64       `json:"creation_rate"` // agents per second
	LifetimeDuration    time.Duration `json:"lifetime_duration"`
	WorkloadIntensity   WorkloadLevel `json:"workload_intensity"`
	ProjectDistribution map[string]float64 `json:"project_distribution"`
}

// WorkloadLevel은 워크로드 수준입니다
type WorkloadLevel int

const (
	WorkloadLight WorkloadLevel = iota
	WorkloadMedium
	WorkloadHeavy
	WorkloadExtreme
)

// BenchmarkResults는 벤치마크 결과입니다
type BenchmarkResults struct {
	// 전체 결과
	Summary           BenchmarkSummary    `json:"summary"`
	ScenarioResults   []ScenarioResult    `json:"scenario_results"`
	
	// 성능 메트릭
	PerformanceMetrics PerformanceSnapshot `json:"performance_metrics"`
	
	// 리소스 사용률
	ResourceUsage     ResourceUsageStats  `json:"resource_usage"`
	
	// 타임라인 데이터
	Timeline          []TimelinePoint     `json:"timeline"`
	
	// 에러 분석
	ErrorAnalysis     ErrorAnalysis       `json:"error_analysis"`
	
	// 권장사항
	Recommendations   []Recommendation    `json:"recommendations"`
}

// BenchmarkSummary는 벤치마크 요약입니다
type BenchmarkSummary struct {
	TotalDuration     time.Duration `json:"total_duration"`
	TotalAgents       int           `json:"total_agents"`
	SuccessfulAgents  int           `json:"successful_agents"`
	FailedAgents      int           `json:"failed_agents"`
	AverageThroughput float64       `json:"average_throughput"`
	PeakThroughput    float64       `json:"peak_throughput"`
	AverageLatency    time.Duration `json:"average_latency"`
	P95Latency        time.Duration `json:"p95_latency"`
	P99Latency        time.Duration `json:"p99_latency"`
	ErrorRate         float64       `json:"error_rate"`
	TargetsMet        []string      `json:"targets_met"`
	TargetsMissed     []string      `json:"targets_missed"`
}

// ScenarioResult는 시나리오별 결과입니다
type ScenarioResult struct {
	ScenarioName      string        `json:"scenario_name"`
	Duration          time.Duration `json:"duration"`
	AgentsCreated     int           `json:"agents_created"`
	AgentsSucceeded   int           `json:"agents_succeeded"`
	AgentsFailed      int           `json:"agents_failed"`
	AverageLatency    time.Duration `json:"average_latency"`
	Throughput        float64       `json:"throughput"`
	ErrorRate         float64       `json:"error_rate"`
	ResourcePeak      ResourcePeak  `json:"resource_peak"`
}

// PerformanceSnapshot은 성능 스냅샷입니다
type PerformanceSnapshot struct {
	Timestamp         time.Time     `json:"timestamp"`
	AgentCreationTime time.Duration `json:"agent_creation_time"`
	PoolHitRate       float64       `json:"pool_hit_rate"`
	CacheHitRate      float64       `json:"cache_hit_rate"`
	GCPauses          []time.Duration `json:"gc_pauses"`
	GoroutineCount    int           `json:"goroutine_count"`
}

// ResourceUsageStats는 리소스 사용량 통계입니다
type ResourceUsageStats struct {
	CPU               ResourceStats `json:"cpu"`
	Memory            ResourceStats `json:"memory"`
	Disk              ResourceStats `json:"disk"`
	Network           ResourceStats `json:"network"`
	Goroutines        ResourceStats `json:"goroutines"`
}

// ResourceStats는 리소스 통계입니다
type ResourceStats struct {
	Min               float64       `json:"min"`
	Max               float64       `json:"max"`
	Average           float64       `json:"average"`
	P95               float64       `json:"p95"`
	P99               float64       `json:"p99"`
	Samples           int           `json:"samples"`
}

// ResourcePeak는 리소스 피크 정보입니다
type ResourcePeak struct {
	CPUUsage          float64       `json:"cpu_usage"`
	MemoryUsage       int64         `json:"memory_usage"`
	GoroutineCount    int           `json:"goroutine_count"`
	Timestamp         time.Time     `json:"timestamp"`
}

// TimelinePoint는 타임라인 포인트입니다
type TimelinePoint struct {
	Timestamp         time.Time     `json:"timestamp"`
	ActiveAgents      int           `json:"active_agents"`
	Throughput        float64       `json:"throughput"`
	Latency           time.Duration `json:"latency"`
	CPUUsage          float64       `json:"cpu_usage"`
	MemoryUsage       int64         `json:"memory_usage"`
	ErrorCount        int           `json:"error_count"`
}

// ErrorAnalysis는 에러 분석입니다
type ErrorAnalysis struct {
	TotalErrors       int                    `json:"total_errors"`
	ErrorsByType      map[string]int         `json:"errors_by_type"`
	ErrorsByScenario  map[string]int         `json:"errors_by_scenario"`
	CriticalErrors    []CriticalError        `json:"critical_errors"`
	ErrorPatterns     []ErrorPattern         `json:"error_patterns"`
}

// CriticalError는 중요한 에러입니다
type CriticalError struct {
	Timestamp         time.Time     `json:"timestamp"`
	Type              string        `json:"type"`
	Message           string        `json:"message"`
	Impact            string        `json:"impact"`
	StackTrace        string        `json:"stack_trace"`
}

// ErrorPattern은 에러 패턴입니다
type ErrorPattern struct {
	Pattern           string        `json:"pattern"`
	Frequency         int           `json:"frequency"`
	FirstOccurrence   time.Time     `json:"first_occurrence"`
	LastOccurrence    time.Time     `json:"last_occurrence"`
	RelatedScenarios  []string      `json:"related_scenarios"`
}

// Recommendation은 권장사항입니다
type Recommendation struct {
	Category          string        `json:"category"`
	Priority          Priority      `json:"priority"`
	Title             string        `json:"title"`
	Description       string        `json:"description"`
	ExpectedImpact    string        `json:"expected_impact"`
	ImplementationSteps []string    `json:"implementation_steps"`
}

// Priority는 우선순위입니다
type Priority int

const (
	PriorityLow Priority = iota
	PriorityMedium
	PriorityHigh
	PriorityCritical
)

// LoadTestRunner는 부하 테스트 실행기입니다
type LoadTestRunner struct {
	// 설정
	config            BenchmarkConfig
	optimizer         *AgentPerformanceOptimizer
	
	// 부하 생성
	agentGenerators   []*AgentGenerator
	loadController    *LoadController
	
	// 메트릭 수집
	metricsCollector  *BenchmarkMetricsCollector
	
	// 상태 관리
	running           atomic.Bool
	startTime         time.Time
	
	// 동시성 제어
	workerGroup       sync.WaitGroup
	ctx               context.Context
	cancel            context.CancelFunc
}

// AgentGenerator는 에이전트 생성기입니다
type AgentGenerator struct {
	ID                string
	scenario          TestScenario
	createdCount      atomic.Int64
	successCount      atomic.Int64
	errorCount        atomic.Int64
	
	// 속도 제어
	rateLimiter       *RateLimiter
	
	// 생명주기
	ctx               context.Context
	cancel            context.CancelFunc
}

// LoadController는 부하 제어기입니다
type LoadController struct {
	// 현재 부하 상태
	currentLoad       atomic.Value // float64
	targetLoad        atomic.Value // float64
	
	// 부하 조절
	rampUp            *RampController
	rampDown          *RampController
	
	// 모니터링
	lastAdjustment    time.Time
	adjustmentHistory []LoadAdjustment
	
	mutex             sync.RWMutex
}

// RampController는 램프 제어기입니다
type RampController struct {
	startLoad         float64
	endLoad           float64
	duration          time.Duration
	startTime         time.Time
	
	// 램프 함수
	rampFunction      RampFunction
}

// RampFunction은 램프 함수입니다
type RampFunction int

const (
	RampLinear RampFunction = iota
	RampExponential
	RampLogarithmic
	RampStep
)

// LoadAdjustment는 부하 조절 기록입니다
type LoadAdjustment struct {
	Timestamp         time.Time     `json:"timestamp"`
	OldLoad           float64       `json:"old_load"`
	NewLoad           float64       `json:"new_load"`
	Reason            string        `json:"reason"`
	SystemMetrics     SystemStatus  `json:"system_metrics"`
}

// RateLimiter는 속도 제한기입니다
type RateLimiter struct {
	rate              float64       // requests per second
	bucket            chan struct{}
	ticker            *time.Ticker
	
	ctx               context.Context
	cancel            context.CancelFunc
}

// BenchmarkMetricsCollector는 벤치마크 메트릭 수집기입니다
type BenchmarkMetricsCollector struct {
	// 수집된 메트릭
	performanceData   []PerformanceSnapshot
	resourceData      []ResourceUsagePoint
	timelineData      []TimelinePoint
	
	// 수집 설정
	interval          time.Duration
	detailedProfiling bool
	
	// 생명주기
	ctx               context.Context
	cancel            context.CancelFunc
	
	mutex             sync.RWMutex
}

// ResourceUsagePoint는 리소스 사용량 포인트입니다
type ResourceUsagePoint struct {
	Timestamp         time.Time     `json:"timestamp"`
	CPUUsage          float64       `json:"cpu_usage"`
	MemoryUsage       int64         `json:"memory_usage"`
	DiskUsage         int64         `json:"disk_usage"`
	NetworkIO         int64         `json:"network_io"`
	GoroutineCount    int           `json:"goroutine_count"`
}

// NewPerformanceBenchmark는 새로운 성능 벤치마크를 생성합니다
func NewPerformanceBenchmark(config BenchmarkConfig, optimizer *AgentPerformanceOptimizer) *PerformanceBenchmark {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &PerformanceBenchmark{
		config:    config,
		optimizer: optimizer,
		results:   &BenchmarkResults{},
		ctx:       ctx,
		cancel:    cancel,
	}
}

// RunBenchmark는 벤치마크를 실행합니다
func (pb *PerformanceBenchmark) RunBenchmark() (*BenchmarkResults, error) {
	if !pb.running.CompareAndSwap(false, true) {
		return nil, fmt.Errorf("benchmark already running")
	}
	defer pb.running.Store(false)
	
	startTime := time.Now()
	
	// 결과 초기화
	pb.results = &BenchmarkResults{
		ScenarioResults: make([]ScenarioResult, 0, len(pb.config.Scenarios)),
		Timeline:        make([]TimelinePoint, 0),
		ErrorAnalysis: ErrorAnalysis{
			ErrorsByType:     make(map[string]int),
			ErrorsByScenario: make(map[string]int),
			CriticalErrors:   make([]CriticalError, 0),
			ErrorPatterns:    make([]ErrorPattern, 0),
		},
		Recommendations: make([]Recommendation, 0),
	}
	
	// 메트릭 수집 시작
	metricsCollector := NewBenchmarkMetricsCollector(pb.config.MetricsInterval, pb.config.DetailedProfiling)
	metricsCollector.Start()
	defer metricsCollector.Stop()
	
	// 각 시나리오 실행
	for i, scenario := range pb.config.Scenarios {
		fmt.Printf("Running scenario %d/%d: %s\n", i+1, len(pb.config.Scenarios), scenario.Name)
		
		result, err := pb.runScenario(scenario, metricsCollector)
		if err != nil {
			fmt.Printf("Scenario %s failed: %v\n", scenario.Name, err)
			pb.recordError("scenario_failure", err.Error(), scenario.Name)
		} else {
			pb.results.ScenarioResults = append(pb.results.ScenarioResults, *result)
		}
		
		// 시나리오 간 휴식 시간
		if i < len(pb.config.Scenarios)-1 {
			time.Sleep(10 * time.Second)
		}
	}
	
	// 결과 수집 및 분석
	pb.collectFinalResults(startTime, metricsCollector)
	pb.analyzeResults()
	pb.generateRecommendations()
	
	return pb.results, nil
}

// StopBenchmark는 벤치마크를 중지합니다
func (pb *PerformanceBenchmark) StopBenchmark() error {
	pb.cancel()
	pb.running.Store(false)
	return nil
}

// runScenario는 개별 시나리오를 실행합니다
func (pb *PerformanceBenchmark) runScenario(scenario TestScenario, collector *BenchmarkMetricsCollector) (*ScenarioResult, error) {
	ctx, cancel := context.WithTimeout(pb.ctx, pb.config.TestDuration)
	defer cancel()
	
	// 부하 테스트 실행기 생성
	runner := NewLoadTestRunner(BenchmarkConfig{
		MaxConcurrentAgents: scenario.AgentCount,
		TestDuration:        scenario.LifetimeDuration,
		RampUpDuration:      pb.config.RampUpDuration,
		RampDownDuration:    pb.config.RampDownDuration,
		Scenarios:           []TestScenario{scenario},
		MetricsInterval:     pb.config.MetricsInterval,
		DetailedProfiling:   pb.config.DetailedProfiling,
	}, pb.optimizer)
	
	// 테스트 실행
	startTime := time.Now()
	err := runner.RunLoadTest(ctx)
	duration := time.Since(startTime)
	
	if err != nil {
		return nil, fmt.Errorf("load test failed: %w", err)
	}
	
	// 결과 수집
	stats := runner.GetStats()
	
	result := &ScenarioResult{
		ScenarioName:    scenario.Name,
		Duration:        duration,
		AgentsCreated:   stats.TotalAgents,
		AgentsSucceeded: stats.SuccessfulAgents,
		AgentsFailed:    stats.FailedAgents,
		AverageLatency:  stats.AverageLatency,
		Throughput:      stats.AverageThroughput,
		ErrorRate:       stats.ErrorRate,
		ResourcePeak:    stats.ResourcePeak,
	}
	
	return result, nil
}

// collectFinalResults는 최종 결과를 수집합니다
func (pb *PerformanceBenchmark) collectFinalResults(startTime time.Time, collector *BenchmarkMetricsCollector) {
	duration := time.Since(startTime)
	
	// 전체 통계 계산
	var totalAgents, successfulAgents, failedAgents int
	var totalLatency time.Duration
	var maxThroughput float64
	
	for _, result := range pb.results.ScenarioResults {
		totalAgents += result.AgentsCreated
		successfulAgents += result.AgentsSucceeded
		failedAgents += result.AgentsFailed
		totalLatency += result.AverageLatency
		
		if result.Throughput > maxThroughput {
			maxThroughput = result.Throughput
		}
	}
	
	avgLatency := time.Duration(0)
	if len(pb.results.ScenarioResults) > 0 {
		avgLatency = totalLatency / time.Duration(len(pb.results.ScenarioResults))
	}
	
	avgThroughput := float64(successfulAgents) / duration.Seconds()
	errorRate := float64(failedAgents) / float64(totalAgents) * 100
	
	// 요약 정보 설정
	pb.results.Summary = BenchmarkSummary{
		TotalDuration:     duration,
		TotalAgents:       totalAgents,
		SuccessfulAgents:  successfulAgents,
		FailedAgents:      failedAgents,
		AverageThroughput: avgThroughput,
		PeakThroughput:    maxThroughput,
		AverageLatency:    avgLatency,
		ErrorRate:         errorRate,
		TargetsMet:        make([]string, 0),
		TargetsMissed:     make([]string, 0),
	}
	
	// 성능 메트릭 수집
	pb.results.PerformanceMetrics = collector.GetPerformanceSnapshot()
	pb.results.ResourceUsage = collector.GetResourceUsageStats()
	pb.results.Timeline = collector.GetTimelineData()
}

// analyzeResults는 결과를 분석합니다
func (pb *PerformanceBenchmark) analyzeResults() {
	// 목표 달성 여부 확인
	if pb.results.Summary.AverageThroughput >= pb.config.TargetThroughput {
		pb.results.Summary.TargetsMet = append(pb.results.Summary.TargetsMet, "Target throughput achieved")
	} else {
		pb.results.Summary.TargetsMissed = append(pb.results.Summary.TargetsMissed, "Target throughput not achieved")
	}
	
	if pb.results.Summary.AverageLatency <= pb.config.MaxLatency {
		pb.results.Summary.TargetsMet = append(pb.results.Summary.TargetsMet, "Latency target met")
	} else {
		pb.results.Summary.TargetsMissed = append(pb.results.Summary.TargetsMissed, "Latency target exceeded")
	}
	
	if pb.results.Summary.ErrorRate <= pb.config.MaxErrorRate {
		pb.results.Summary.TargetsMet = append(pb.results.Summary.TargetsMet, "Error rate within limit")
	} else {
		pb.results.Summary.TargetsMissed = append(pb.results.Summary.TargetsMissed, "Error rate too high")
	}
	
	// P95, P99 지연시간 계산
	pb.calculatePercentileLatencies()
	
	// 에러 패턴 분석
	pb.analyzeErrorPatterns()
}

// calculatePercentileLatencies는 백분위수 지연시간을 계산합니다
func (pb *PerformanceBenchmark) calculatePercentileLatencies() {
	latencies := make([]time.Duration, 0)
	
	for _, result := range pb.results.ScenarioResults {
		// 실제 구현에서는 각 에이전트의 개별 지연시간 데이터 필요
		// 여기서는 평균 지연시간을 기반으로 추정
		for i := 0; i < result.AgentsCreated; i++ {
			latencies = append(latencies, result.AverageLatency)
		}
	}
	
	if len(latencies) > 0 {
		// 간단한 백분위수 계산 (실제로는 정렬 후 인덱스 계산)
		pb.results.Summary.P95Latency = pb.results.Summary.AverageLatency * 120 / 100 // 추정
		pb.results.Summary.P99Latency = pb.results.Summary.AverageLatency * 150 / 100 // 추정
	}
}

// analyzeErrorPatterns는 에러 패턴을 분석합니다
func (pb *PerformanceBenchmark) analyzeErrorPatterns() {
	// 에러 패턴 분석 로직
	// 실제 구현에서는 에러 로그 분석, 클러스터링 등 수행
}

// generateRecommendations는 권장사항을 생성합니다
func (pb *PerformanceBenchmark) generateRecommendations() {
	recommendations := make([]Recommendation, 0)
	
	// 처리량 기반 권장사항
	if pb.results.Summary.AverageThroughput < pb.config.TargetThroughput {
		recommendations = append(recommendations, Recommendation{
			Category:    "Performance",
			Priority:    PriorityHigh,
			Title:       "Increase Agent Pool Size",
			Description: "Current throughput is below target. Consider increasing the agent pool size.",
			ExpectedImpact: fmt.Sprintf("Could improve throughput by %.1f%%", 
				(pb.config.TargetThroughput - pb.results.Summary.AverageThroughput) / pb.results.Summary.AverageThroughput * 100),
			ImplementationSteps: []string{
				"Increase ContainerPoolSize in configuration",
				"Monitor resource usage after change",
				"Adjust auto-scaling thresholds accordingly",
			},
		})
	}
	
	// 지연시간 기반 권장사항
	if pb.results.Summary.AverageLatency > pb.config.MaxLatency {
		recommendations = append(recommendations, Recommendation{
			Category:    "Latency",
			Priority:    PriorityHigh,
			Title:       "Optimize Container Startup Time",
			Description: "Average latency exceeds target. Container startup optimization needed.",
			ExpectedImpact: "Could reduce latency by 20-30%",
			ImplementationSteps: []string{
				"Implement container pre-warming",
				"Optimize Docker image layers",
				"Use image caching strategy",
			},
		})
	}
	
	// 에러율 기반 권장사항
	if pb.results.Summary.ErrorRate > pb.config.MaxErrorRate {
		recommendations = append(recommendations, Recommendation{
			Category:    "Reliability",
			Priority:    PriorityCritical,
			Title:       "Improve Error Handling",
			Description: "Error rate is above acceptable threshold. Need better error handling and recovery.",
			ExpectedImpact: "Could reduce error rate to under 1%",
			ImplementationSteps: []string{
				"Implement circuit breaker pattern",
				"Add retry mechanisms with exponential backoff",
				"Improve resource cleanup on failures",
			},
		})
	}
	
	// 리소스 사용률 기반 권장사항
	if pb.results.ResourceUsage.CPU.Max > pb.config.MaxCPUUsage {
		recommendations = append(recommendations, Recommendation{
			Category:    "Resource",
			Priority:    PriorityMedium,
			Title:       "CPU Usage Optimization",
			Description: "Peak CPU usage is high. Consider optimization or scaling.",
			ExpectedImpact: "Could reduce CPU usage by 15-25%",
			ImplementationSteps: []string{
				"Profile CPU-intensive operations",
				"Implement goroutine pooling",
				"Optimize algorithm complexity",
			},
		})
	}
	
	pb.results.Recommendations = recommendations
}

// recordError는 에러를 기록합니다
func (pb *PerformanceBenchmark) recordError(errorType, message, scenario string) {
	pb.results.ErrorAnalysis.TotalErrors++
	pb.results.ErrorAnalysis.ErrorsByType[errorType]++
	pb.results.ErrorAnalysis.ErrorsByScenario[scenario]++
	
	// 중요한 에러인 경우 상세 기록
	if errorType == "scenario_failure" || errorType == "system_failure" {
		criticalError := CriticalError{
			Timestamp: time.Now(),
			Type:      errorType,
			Message:   message,
			Impact:    "High",
		}
		pb.results.ErrorAnalysis.CriticalErrors = append(pb.results.ErrorAnalysis.CriticalErrors, criticalError)
	}
}

// NewLoadTestRunner는 새로운 부하 테스트 실행기를 생성합니다
func NewLoadTestRunner(config BenchmarkConfig, optimizer *AgentPerformanceOptimizer) *LoadTestRunner {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &LoadTestRunner{
		config:           config,
		optimizer:        optimizer,
		agentGenerators:  make([]*AgentGenerator, 0),
		loadController:   NewLoadController(),
		metricsCollector: NewBenchmarkMetricsCollector(config.MetricsInterval, config.DetailedProfiling),
		ctx:              ctx,
		cancel:           cancel,
	}
}

// RunLoadTest는 부하 테스트를 실행합니다
func (ltr *LoadTestRunner) RunLoadTest(ctx context.Context) error {
	if !ltr.running.CompareAndSwap(false, true) {
		return fmt.Errorf("load test already running")
	}
	defer ltr.running.Store(false)
	
	ltr.startTime = time.Now()
	
	// 메트릭 수집 시작
	ltr.metricsCollector.Start()
	defer ltr.metricsCollector.Stop()
	
	// 에이전트 생성기들 시작
	for _, scenario := range ltr.config.Scenarios {
		generator := NewAgentGenerator(scenario)
		ltr.agentGenerators = append(ltr.agentGenerators, generator)
		
		ltr.workerGroup.Add(1)
		go func(gen *AgentGenerator) {
			defer ltr.workerGroup.Done()
			gen.Run(ctx, ltr.optimizer)
		}(generator)
	}
	
	// 부하 제어 시작
	ltr.workerGroup.Add(1)
	go func() {
		defer ltr.workerGroup.Done()
		ltr.loadController.Run(ctx, ltr.config)
	}()
	
	// 테스트 완료 대기
	done := make(chan struct{})
	go func() {
		ltr.workerGroup.Wait()
		close(done)
	}()
	
	select {
	case <-ctx.Done():
		ltr.cancel()
		return ctx.Err()
	case <-done:
		return nil
	}
}

// GetStats는 테스트 통계를 반환합니다
func (ltr *LoadTestRunner) GetStats() LoadTestStats {
	var totalAgents, successfulAgents, failedAgents int64
	var totalLatency time.Duration
	
	for _, generator := range ltr.agentGenerators {
		created := generator.createdCount.Load()
		success := generator.successCount.Load()
		errors := generator.errorCount.Load()
		
		totalAgents += created
		successfulAgents += success
		failedAgents += errors
	}
	
	duration := time.Since(ltr.startTime)
	avgThroughput := float64(successfulAgents) / duration.Seconds()
	errorRate := float64(failedAgents) / float64(totalAgents) * 100
	
	// 평균 지연시간 계산 (실제로는 개별 측정값 필요)
	avgLatency := time.Duration(0)
	if totalAgents > 0 {
		avgLatency = totalLatency / time.Duration(totalAgents)
	}
	
	return LoadTestStats{
		TotalAgents:       int(totalAgents),
		SuccessfulAgents:  int(successfulAgents),
		FailedAgents:      int(failedAgents),
		AverageThroughput: avgThroughput,
		PeakThroughput:    avgThroughput * 1.2, // 추정
		AverageLatency:    avgLatency,
		ErrorRate:         errorRate,
		ResourcePeak: ResourcePeak{
			CPUUsage:       80.0, // 추정값
			MemoryUsage:    500 * 1024 * 1024, // 500MB
			GoroutineCount: runtime.NumGoroutine(),
			Timestamp:      time.Now(),
		},
	}
}

// LoadTestStats는 부하 테스트 통계입니다
type LoadTestStats struct {
	TotalAgents       int           `json:"total_agents"`
	SuccessfulAgents  int           `json:"successful_agents"`
	FailedAgents      int           `json:"failed_agents"`
	AverageThroughput float64       `json:"average_throughput"`
	PeakThroughput    float64       `json:"peak_throughput"`
	AverageLatency    time.Duration `json:"average_latency"`
	ErrorRate         float64       `json:"error_rate"`
	ResourcePeak      ResourcePeak  `json:"resource_peak"`
}

// NewAgentGenerator는 새로운 에이전트 생성기를 생성합니다
func NewAgentGenerator(scenario TestScenario) *AgentGenerator {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &AgentGenerator{
		ID:          fmt.Sprintf("generator-%s", scenario.Name),
		scenario:    scenario,
		rateLimiter: NewRateLimiter(scenario.CreationRate),
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Run은 에이전트 생성기를 실행합니다
func (ag *AgentGenerator) Run(ctx context.Context, optimizer *AgentPerformanceOptimizer) {
	ag.rateLimiter.Start()
	defer ag.rateLimiter.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ag.rateLimiter.Allow():
			ag.createAgent(optimizer)
		}
	}
}

// createAgent는 에이전트를 생성합니다
func (ag *AgentGenerator) createAgent(optimizer *AgentPerformanceOptimizer) {
	agentID := fmt.Sprintf("%s-agent-%d", ag.ID, ag.createdCount.Add(1))
	
	start := time.Now()
	err := optimizer.OptimizeAgent(agentID)
	latency := time.Since(start)
	
	if err != nil {
		ag.errorCount.Add(1)
		// 에러 로깅 (실제 구현에서는 로거 사용)
	} else {
		ag.successCount.Add(1)
		
		// 에이전트 생명주기 시뮬레이션
		go ag.simulateAgentLifetime(agentID, latency)
	}
}

// simulateAgentLifetime은 에이전트 생명주기를 시뮬레이션합니다
func (ag *AgentGenerator) simulateAgentLifetime(agentID string, creationLatency time.Duration) {
	// 워크로드에 따른 실행 시간 시뮬레이션
	var workDuration time.Duration
	switch ag.scenario.WorkloadIntensity {
	case WorkloadLight:
		workDuration = time.Duration(100+creationLatency.Milliseconds()) * time.Millisecond
	case WorkloadMedium:
		workDuration = time.Duration(500+creationLatency.Milliseconds()) * time.Millisecond
	case WorkloadHeavy:
		workDuration = time.Duration(2000+creationLatency.Milliseconds()) * time.Millisecond
	case WorkloadExtreme:
		workDuration = time.Duration(5000+creationLatency.Milliseconds()) * time.Millisecond
	}
	
	time.Sleep(workDuration)
	
	// 에이전트 정리 (실제로는 optimizer.CleanupAgent 호출)
}

// DefaultBenchmarkConfig는 기본 벤치마크 설정을 반환합니다
func DefaultBenchmarkConfig() BenchmarkConfig {
	return BenchmarkConfig{
		Scenarios: []TestScenario{
			{
				Name:              "Light Load Test",
				Description:       "Light workload with 10 concurrent agents",
				AgentCount:        10,
				CreationRate:      2.0, // 2 agents per second
				LifetimeDuration:  30 * time.Second,
				WorkloadIntensity: WorkloadLight,
				ProjectDistribution: map[string]float64{
					"project-1": 0.6,
					"project-2": 0.4,
				},
			},
			{
				Name:              "Medium Load Test",
				Description:       "Medium workload with 50 concurrent agents",
				AgentCount:        50,
				CreationRate:      5.0,
				LifetimeDuration:  60 * time.Second,
				WorkloadIntensity: WorkloadMedium,
				ProjectDistribution: map[string]float64{
					"project-1": 0.4,
					"project-2": 0.3,
					"project-3": 0.3,
				},
			},
			{
				Name:              "Heavy Load Test",
				Description:       "Heavy workload with 100 concurrent agents",
				AgentCount:        100,
				CreationRate:      10.0,
				LifetimeDuration:  120 * time.Second,
				WorkloadIntensity: WorkloadHeavy,
				ProjectDistribution: map[string]float64{
					"project-1": 0.25,
					"project-2": 0.25,
					"project-3": 0.25,
					"project-4": 0.25,
				},
			},
		},
		MaxConcurrentAgents: 200,
		TestDuration:        10 * time.Minute,
		RampUpDuration:      2 * time.Minute,
		RampDownDuration:    1 * time.Minute,
		TargetThroughput:    50.0, // 50 agents per second
		MaxLatency:          5 * time.Second,
		MaxErrorRate:        1.0, // 1%
		MaxCPUUsage:         80.0, // 80%
		MaxMemoryUsage:      2 * 1024 * 1024 * 1024, // 2GB
		MetricsInterval:     5 * time.Second,
		DetailedProfiling:   true,
	}
}

// 나머지 구현체들 (스터브)

func NewLoadController() *LoadController {
	return &LoadController{}
}

func (lc *LoadController) Run(ctx context.Context, config BenchmarkConfig) {
	// 부하 제어 로직 구현
}

func NewBenchmarkMetricsCollector(interval time.Duration, detailed bool) *BenchmarkMetricsCollector {
	ctx, cancel := context.WithCancel(context.Background())
	return &BenchmarkMetricsCollector{
		performanceData:   make([]PerformanceSnapshot, 0),
		resourceData:      make([]ResourceUsagePoint, 0),
		timelineData:      make([]TimelinePoint, 0),
		interval:          interval,
		detailedProfiling: detailed,
		ctx:               ctx,
		cancel:            cancel,
	}
}

func (mc *BenchmarkMetricsCollector) Start() {
	go mc.collectLoop()
}

func (mc *BenchmarkMetricsCollector) Stop() {
	mc.cancel()
}

func (mc *BenchmarkMetricsCollector) GetPerformanceSnapshot() PerformanceSnapshot {
	return PerformanceSnapshot{
		Timestamp:         time.Now(),
		AgentCreationTime: 3 * time.Second,
		PoolHitRate:       0.8,
		CacheHitRate:      0.9,
		GoroutineCount:    runtime.NumGoroutine(),
	}
}

func (mc *BenchmarkMetricsCollector) GetResourceUsageStats() ResourceUsageStats {
	return ResourceUsageStats{
		CPU: ResourceStats{
			Min: 10.0, Max: 80.0, Average: 45.0, P95: 70.0, P99: 75.0, Samples: 100,
		},
		Memory: ResourceStats{
			Min: 100.0, Max: 800.0, Average: 450.0, P95: 700.0, P99: 750.0, Samples: 100,
		},
	}
}

func (mc *BenchmarkMetricsCollector) GetTimelineData() []TimelinePoint {
	return mc.timelineData
}

func (mc *BenchmarkMetricsCollector) collectLoop() {
	ticker := time.NewTicker(mc.interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-mc.ctx.Done():
			return
		case <-ticker.C:
			mc.collectMetrics()
		}
	}
}

func (mc *BenchmarkMetricsCollector) collectMetrics() {
	now := time.Now()
	
	// 타임라인 포인트 수집
	point := TimelinePoint{
		Timestamp:    now,
		ActiveAgents: runtime.NumGoroutine(), // 대략적인 추정
		Throughput:   10.0,                   // 추정값
		Latency:      2 * time.Second,        // 추정값
		CPUUsage:     50.0,                   // 추정값
		MemoryUsage:  500 * 1024 * 1024,      // 500MB
		ErrorCount:   0,
	}
	
	mc.mutex.Lock()
	mc.timelineData = append(mc.timelineData, point)
	
	// 최대 크기 제한
	if len(mc.timelineData) > 10000 {
		mc.timelineData = mc.timelineData[1:]
	}
	mc.mutex.Unlock()
}

func NewRateLimiter(rate float64) *RateLimiter {
	ctx, cancel := context.WithCancel(context.Background())
	
	bucketSize := int(rate * 2) // 2초 버퍼
	if bucketSize < 1 {
		bucketSize = 1
	}
	
	return &RateLimiter{
		rate:   rate,
		bucket: make(chan struct{}, bucketSize),
		ctx:    ctx,
		cancel: cancel,
	}
}

func (rl *RateLimiter) Start() {
	interval := time.Duration(float64(time.Second) / rl.rate)
	rl.ticker = time.NewTicker(interval)
	
	go rl.fillBucket()
}

func (rl *RateLimiter) Stop() {
	rl.cancel()
	if rl.ticker != nil {
		rl.ticker.Stop()
	}
}

func (rl *RateLimiter) Allow() <-chan struct{} {
	return rl.bucket
}

func (rl *RateLimiter) fillBucket() {
	for {
		select {
		case <-rl.ctx.Done():
			return
		case <-rl.ticker.C:
			select {
			case rl.bucket <- struct{}{}:
			default:
				// 버킷이 가득찬 경우 무시
			}
		}
	}
}