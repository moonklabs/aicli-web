package testing

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// PerformanceTestSuite는 성능 테스트 스위트입니다
type PerformanceTestSuite struct {
	config  TestConfig
	client  *http.Client
	results []TestResult
	mutex   sync.RWMutex
	running atomic.Bool
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// TestConfig는 테스트 설정입니다
type TestConfig struct {
	// 기본 설정
	BaseURL   string        `json:"base_url"`
	Timeout   time.Duration `json:"timeout"`
	UserAgent string        `json:"user_agent"`

	// 부하 테스트 설정
	ConcurrentUsers  int           `json:"concurrent_users"`
	TestDuration     time.Duration `json:"test_duration"`
	RampUpDuration   time.Duration `json:"ramp_up_duration"`
	RampDownDuration time.Duration `json:"ramp_down_duration"`

	// 성능 임계값
	MaxResponseTime time.Duration `json:"max_response_time"`
	MaxErrorRate    float64       `json:"max_error_rate"`
	MinThroughput   float64       `json:"min_throughput"`

	// 시나리오
	Scenarios []TestScenario `json:"scenarios"`

	// 보고서 설정
	ReportFormat   string `json:"report_format"`
	OutputDir      string `json:"output_dir"`
	EnableRealTime bool   `json:"enable_real_time"`
}

// TestScenario는 테스트 시나리오입니다
type TestScenario struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Weight      float64       `json:"weight"`
	Steps       []TestStep    `json:"steps"`
	ThinkTime   time.Duration `json:"think_time"`
	Iterations  int           `json:"iterations"`
}

// TestStep은 개별 테스트 단계입니다
type TestStep struct {
	Name         string            `json:"name"`
	Method       string            `json:"method"`
	URL          string            `json:"url"`
	Headers      map[string]string `json:"headers"`
	Body         string            `json:"body"`
	ExpectedCode int               `json:"expected_code"`
	Validation   []Validation      `json:"validation"`
	Timeout      time.Duration     `json:"timeout"`
}

// Validation은 응답 검증 방법입니다
type Validation struct {
	Type     string `json:"type"`     // "json_path", "regex", "contains", "header"
	Target   string `json:"target"`   // 검증 대상
	Expected string `json:"expected"` // 기대값
}

// TestResult는 테스트 결과입니다
type TestResult struct {
	ScenarioName string        `json:"scenario_name"`
	StepName     string        `json:"step_name"`
	URL          string        `json:"url"`
	Method       string        `json:"method"`
	StartTime    time.Time     `json:"start_time"`
	EndTime      time.Time     `json:"end_time"`
	ResponseTime time.Duration `json:"response_time"`
	StatusCode   int           `json:"status_code"`
	ResponseSize int64         `json:"response_size"`
	Success      bool          `json:"success"`
	Error        string        `json:"error,omitempty"`
	UserID       int           `json:"user_id"`
	Timestamp    int64         `json:"timestamp"`
}

// TestSummary는 테스트 요약입니다
type TestSummary struct {
	TotalRequests       int64                      `json:"total_requests"`
	SuccessfulRequests  int64                      `json:"successful_requests"`
	FailedRequests      int64                      `json:"failed_requests"`
	ErrorRate           float64                    `json:"error_rate"`
	Throughput          float64                    `json:"throughput"`
	AverageResponseTime time.Duration              `json:"average_response_time"`
	MinResponseTime     time.Duration              `json:"min_response_time"`
	MaxResponseTime     time.Duration              `json:"max_response_time"`
	P50ResponseTime     time.Duration              `json:"p50_response_time"`
	P95ResponseTime     time.Duration              `json:"p95_response_time"`
	P99ResponseTime     time.Duration              `json:"p99_response_time"`
	TotalDataTransfer   int64                      `json:"total_data_transfer"`
	StartTime           time.Time                  `json:"start_time"`
	EndTime             time.Time                  `json:"end_time"`
	Duration            time.Duration              `json:"duration"`
	ScenarioResults     map[string]ScenarioSummary `json:"scenario_results"`
}

// ScenarioSummary는 시나리오별 요약입니다
type ScenarioSummary struct {
	Name                string        `json:"name"`
	Requests            int64         `json:"requests"`
	Successes           int64         `json:"successes"`
	Failures            int64         `json:"failures"`
	ErrorRate           float64       `json:"error_rate"`
	AverageResponseTime time.Duration `json:"average_response_time"`
	Throughput          float64       `json:"throughput"`
}

// LoadTestReport는 부하 테스트 보고서입니다
type LoadTestReport struct {
	Summary         TestSummary         `json:"summary"`
	Results         []TestResult        `json:"results"`
	Metrics         PerformanceMetrics  `json:"metrics"`
	TimelineData    []TimelineDataPoint `json:"timeline_data"`
	Recommendations []string            `json:"recommendations"`
	Issues          []PerformanceIssue  `json:"issues"`
	GeneratedAt     time.Time           `json:"generated_at"`
}

// TimelineDataPoint는 시간별 데이터 포인트입니다
type TimelineDataPoint struct {
	Timestamp    int64   `json:"timestamp"`
	Throughput   float64 `json:"throughput"`
	ResponseTime float64 `json:"response_time"`
	ErrorRate    float64 `json:"error_rate"`
	ActiveUsers  int     `json:"active_users"`
	MemoryUsage  uint64  `json:"memory_usage"`
	CPUUsage     float64 `json:"cpu_usage"`
}

// PerformanceMetrics는 성능 메트릭입니다
type PerformanceMetrics struct {
	CPUUsage       float64 `json:"cpu_usage"`
	MemoryUsage    uint64  `json:"memory_usage"`
	GoroutineCount int     `json:"goroutine_count"`
	HeapSize       uint64  `json:"heap_size"`
	GCCount        uint32  `json:"gc_count"`
}

// PerformanceIssue는 성능 문제입니다
type PerformanceIssue struct {
	Type        string    `json:"type"`
	Severity    string    `json:"severity"`
	Description string    `json:"description"`
	URL         string    `json:"url"`
	Count       int       `json:"count"`
	FirstSeen   time.Time `json:"first_seen"`
	LastSeen    time.Time `json:"last_seen"`
}

// DefaultTestConfig는 기본 테스트 설정을 반환합니다
func DefaultTestConfig() TestConfig {
	return TestConfig{
		BaseURL:          "http://localhost:8080",
		Timeout:          30 * time.Second,
		UserAgent:        "AICLI-Performance-Test/1.0",
		ConcurrentUsers:  10,
		TestDuration:     5 * time.Minute,
		RampUpDuration:   30 * time.Second,
		RampDownDuration: 30 * time.Second,
		MaxResponseTime:  2 * time.Second,
		MaxErrorRate:     5.0,  // 5%
		MinThroughput:    10.0, // requests/second
		Scenarios: []TestScenario{
			{
				Name:        "Basic Health Check",
				Description: "Basic API health check scenario",
				Weight:      1.0,
				ThinkTime:   1 * time.Second,
				Iterations:  -1, // 무한
				Steps: []TestStep{
					{
						Name:         "Health Check",
						Method:       "GET",
						URL:          "/api/health",
						ExpectedCode: 200,
						Timeout:      5 * time.Second,
					},
				},
			},
		},
		ReportFormat:   "json",
		OutputDir:      "./test-results",
		EnableRealTime: true,
	}
}

// NewPerformanceTestSuite는 새로운 성능 테스트 스위트를 생성합니다
func NewPerformanceTestSuite(config TestConfig) *PerformanceTestSuite {
	ctx, cancel := context.WithCancel(context.Background())

	return &PerformanceTestSuite{
		config: config,
		client: &http.Client{
			Timeout: config.Timeout,
		},
		results: make([]TestResult, 0),
		ctx:     ctx,
		cancel:  cancel,
	}
}

// RunLoadTest는 부하 테스트를 실행합니다
func (pts *PerformanceTestSuite) RunLoadTest() (*LoadTestReport, error) {
	if !pts.running.CompareAndSwap(false, true) {
		return nil, fmt.Errorf("test suite is already running")
	}
	defer pts.running.Store(false)

	startTime := time.Now()
	fmt.Printf("테스트 시작: %d 동시 사용자, %v 지속\n", pts.config.ConcurrentUsers, pts.config.TestDuration)

	// 실시간 메트릭 수집 시작
	timelineData := make([]TimelineDataPoint, 0)
	if pts.config.EnableRealTime {
		pts.wg.Add(1)
		go pts.collectTimelineData(&timelineData)
	}

	// 램프 업 단계
	pts.runRampUp()

	// 메인 테스트 단계
	pts.runMainTest()

	// 램프 다운 단계
	pts.runRampDown()

	// 모든 고루틴 완료 대기
	pts.cancel()
	pts.wg.Wait()

	endTime := time.Now()
	fmt.Printf("테스트 완료: 총 %v 소요\n", endTime.Sub(startTime))

	// 결과 분석 및 보고서 생성
	return pts.generateReport(startTime, endTime, timelineData), nil
}

// RunSingleScenario는 단일 시나리오를 실행합니다
func (pts *PerformanceTestSuite) RunSingleScenario(scenarioName string) (*TestSummary, error) {
	var scenario *TestScenario
	for _, s := range pts.config.Scenarios {
		if s.Name == scenarioName {
			scenario = &s
			break
		}
	}

	if scenario == nil {
		return nil, fmt.Errorf("scenario not found: %s", scenarioName)
	}

	startTime := time.Now()
	results := pts.runScenario(*scenario, 0, 1)
	endTime := time.Now()

	return pts.calculateSummary(results, startTime, endTime), nil
}

// AddScenario는 새로운 시나리오를 추가합니다
func (pts *PerformanceTestSuite) AddScenario(scenario TestScenario) {
	pts.config.Scenarios = append(pts.config.Scenarios, scenario)
}

// GetResults는 현재까지의 테스트 결과를 반환합니다
func (pts *PerformanceTestSuite) GetResults() []TestResult {
	pts.mutex.RLock()
	defer pts.mutex.RUnlock()

	results := make([]TestResult, len(pts.results))
	copy(results, pts.results)
	return results
}

// ClearResults는 모든 테스트 결과를 지웁니다
func (pts *PerformanceTestSuite) ClearResults() {
	pts.mutex.Lock()
	defer pts.mutex.Unlock()

	pts.results = make([]TestResult, 0)
}

// Stop은 실행 중인 테스트를 중지합니다
func (pts *PerformanceTestSuite) Stop() {
	pts.cancel()
}

// 내부 메서드들

func (pts *PerformanceTestSuite) runRampUp() {
	if pts.config.RampUpDuration <= 0 {
		return
	}

	fmt.Printf("램프 업 시작: %v\n", pts.config.RampUpDuration)
	step := pts.config.RampUpDuration / time.Duration(pts.config.ConcurrentUsers)

	for i := 0; i < pts.config.ConcurrentUsers; i++ {
		select {
		case <-pts.ctx.Done():
			return
		default:
			pts.wg.Add(1)
			go pts.runUser(i)
			time.Sleep(step)
		}
	}
}

func (pts *PerformanceTestSuite) runMainTest() {
	fmt.Printf("메인 테스트 시작: %v\n", pts.config.TestDuration)

	// 램프 업이 없는 경우 모든 사용자 시작
	if pts.config.RampUpDuration <= 0 {
		for i := 0; i < pts.config.ConcurrentUsers; i++ {
			pts.wg.Add(1)
			go pts.runUser(i)
		}
	}

	time.Sleep(pts.config.TestDuration)
}

func (pts *PerformanceTestSuite) runRampDown() {
	if pts.config.RampDownDuration <= 0 {
		return
	}

	fmt.Printf("램프 다운 시작: %v\n", pts.config.RampDownDuration)
	// 램프 다운은 단순히 시간만 대기 (사용자들이 자연스럽게 종료)
	time.Sleep(pts.config.RampDownDuration)
}

func (pts *PerformanceTestSuite) runUser(userID int) {
	defer pts.wg.Done()

	for {
		select {
		case <-pts.ctx.Done():
			return
		default:
			// 시나리오 선택 (가중치 기반)
			scenario := pts.selectScenario()
			if scenario == nil {
				return
			}

			// 시나리오 실행
			results := pts.runScenario(*scenario, userID, 1)
			pts.addResults(results)

			// Think Time
			if scenario.ThinkTime > 0 {
				select {
				case <-pts.ctx.Done():
					return
				case <-time.After(scenario.ThinkTime):
				}
			}
		}
	}
}

func (pts *PerformanceTestSuite) selectScenario() *TestScenario {
	if len(pts.config.Scenarios) == 0 {
		return nil
	}

	// 가중치 기반 시나리오 선택
	totalWeight := 0.0
	for _, scenario := range pts.config.Scenarios {
		totalWeight += scenario.Weight
	}

	// 0.0-1.0 범위의 랜덤 값 생성
	random := float64(time.Now().UnixNano()%1000) / 1000.0
	runningWeight := 0.0

	for i, scenario := range pts.config.Scenarios {
		runningWeight += scenario.Weight / totalWeight
		if random <= runningWeight {
			return &pts.config.Scenarios[i]
		}
	}

	// 기본적으로 첫 번째 시나리오 반환
	return &pts.config.Scenarios[0]
}

func (pts *PerformanceTestSuite) runScenario(scenario TestScenario, userID int, iterations int) []TestResult {
	results := make([]TestResult, 0)

	iterationCount := 0
	for scenario.Iterations == -1 || iterationCount < scenario.Iterations {
		select {
		case <-pts.ctx.Done():
			return results
		default:
			// 시나리오의 모든 단계 실행
			for _, step := range scenario.Steps {
				result := pts.executeStep(scenario.Name, step, userID)
				results = append(results, result)

				// 실패 시 시나리오 중단
				if !result.Success {
					break
				}
			}

			iterationCount++
			if iterations > 0 && iterationCount >= iterations {
				break
			}
		}
	}

	return results
}

func (pts *PerformanceTestSuite) executeStep(scenarioName string, step TestStep, userID int) TestResult {
	startTime := time.Now()
	result := TestResult{
		ScenarioName: scenarioName,
		StepName:     step.Name,
		URL:          pts.config.BaseURL + step.URL,
		Method:       step.Method,
		StartTime:    startTime,
		UserID:       userID,
		Timestamp:    startTime.Unix(),
	}

	// HTTP 요청 생성
	req, err := http.NewRequestWithContext(pts.ctx, step.Method, result.URL, nil)
	if err != nil {
		result.EndTime = time.Now()
		result.ResponseTime = result.EndTime.Sub(result.StartTime)
		result.Error = fmt.Sprintf("failed to create request: %v", err)
		return result
	}

	// 헤더 설정
	req.Header.Set("User-Agent", pts.config.UserAgent)
	for key, value := range step.Headers {
		req.Header.Set(key, value)
	}

	// 요청 실행
	resp, err := pts.client.Do(req)
	result.EndTime = time.Now()
	result.ResponseTime = result.EndTime.Sub(result.StartTime)

	if err != nil {
		result.Error = fmt.Sprintf("request failed: %v", err)
		return result
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode
	result.ResponseSize = resp.ContentLength

	// 상태 코드 검증
	if step.ExpectedCode > 0 && resp.StatusCode != step.ExpectedCode {
		result.Error = fmt.Sprintf("unexpected status code: expected %d, got %d", step.ExpectedCode, resp.StatusCode)
		return result
	}

	// 응답 시간 검증
	if result.ResponseTime > pts.config.MaxResponseTime {
		result.Error = fmt.Sprintf("response time exceeded threshold: %v > %v", result.ResponseTime, pts.config.MaxResponseTime)
		return result
	}

	result.Success = true
	return result
}

func (pts *PerformanceTestSuite) addResults(results []TestResult) {
	pts.mutex.Lock()
	defer pts.mutex.Unlock()

	pts.results = append(pts.results, results...)
}

func (pts *PerformanceTestSuite) collectTimelineData(timelineData *[]TimelineDataPoint) {
	defer pts.wg.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-pts.ctx.Done():
			return
		case <-ticker.C:
			dataPoint := pts.collectCurrentMetrics()
			*timelineData = append(*timelineData, dataPoint)
		}
	}
}

func (pts *PerformanceTestSuite) collectCurrentMetrics() TimelineDataPoint {
	pts.mutex.RLock()
	results := make([]TestResult, len(pts.results))
	copy(results, pts.results)
	pts.mutex.RUnlock()

	now := time.Now()
	last5Seconds := now.Add(-5 * time.Second)

	// 최근 5초 내 결과들만 필터링
	recentResults := make([]TestResult, 0)
	for _, result := range results {
		if result.StartTime.After(last5Seconds) {
			recentResults = append(recentResults, result)
		}
	}

	dataPoint := TimelineDataPoint{
		Timestamp: now.Unix(),
	}

	if len(recentResults) > 0 {
		// 처리량 계산
		dataPoint.Throughput = float64(len(recentResults)) / 5.0

		// 평균 응답 시간 계산
		totalResponseTime := time.Duration(0)
		errorCount := 0
		for _, result := range recentResults {
			totalResponseTime += result.ResponseTime
			if !result.Success {
				errorCount++
			}
		}

		dataPoint.ResponseTime = float64(totalResponseTime.Milliseconds()) / float64(len(recentResults))
		dataPoint.ErrorRate = float64(errorCount) / float64(len(recentResults)) * 100.0
	}

	// 시스템 메트릭 수집
	dataPoint.ActiveUsers = pts.config.ConcurrentUsers

	// 실제 메모리 사용량 수집
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	dataPoint.MemoryUsage = memStats.HeapAlloc

	// CPU 사용률은 별도의 수집 로직 필요 (현재는 고루틴 수로 대체)
	dataPoint.CPUUsage = float64(runtime.NumGoroutine())

	return dataPoint
}

func (pts *PerformanceTestSuite) generateReport(startTime, endTime time.Time, timelineData []TimelineDataPoint) *LoadTestReport {
	pts.mutex.RLock()
	results := make([]TestResult, len(pts.results))
	copy(results, pts.results)
	pts.mutex.RUnlock()

	summary := pts.calculateSummary(results, startTime, endTime)

	report := &LoadTestReport{
		Summary:         *summary,
		Results:         results,
		TimelineData:    timelineData,
		Recommendations: pts.generateRecommendations(summary),
		Issues:          pts.detectIssues(results),
		GeneratedAt:     time.Now(),
	}

	return report
}

func (pts *PerformanceTestSuite) calculateSummary(results []TestResult, startTime, endTime time.Time) *TestSummary {
	summary := &TestSummary{
		StartTime:       startTime,
		EndTime:         endTime,
		Duration:        endTime.Sub(startTime),
		ScenarioResults: make(map[string]ScenarioSummary),
	}

	if len(results) == 0 {
		return summary
	}

	summary.TotalRequests = int64(len(results))

	// 응답 시간 데이터 수집
	responseTimes := make([]time.Duration, 0, len(results))
	totalResponseTime := time.Duration(0)
	successCount := int64(0)
	totalDataTransfer := int64(0)
	scenarioStats := make(map[string]*ScenarioSummary)

	for _, result := range results {
		responseTimes = append(responseTimes, result.ResponseTime)
		totalResponseTime += result.ResponseTime
		totalDataTransfer += result.ResponseSize

		if result.Success {
			successCount++
		}

		// 시나리오별 통계
		scenarioStat, exists := scenarioStats[result.ScenarioName]
		if !exists {
			scenarioStat = &ScenarioSummary{
				Name: result.ScenarioName,
			}
			scenarioStats[result.ScenarioName] = scenarioStat
		}

		scenarioStat.Requests++
		if result.Success {
			scenarioStat.Successes++
		} else {
			scenarioStat.Failures++
		}
	}

	summary.SuccessfulRequests = successCount
	summary.FailedRequests = summary.TotalRequests - successCount
	summary.ErrorRate = float64(summary.FailedRequests) / float64(summary.TotalRequests) * 100.0
	summary.Throughput = float64(summary.TotalRequests) / summary.Duration.Seconds()
	summary.TotalDataTransfer = totalDataTransfer

	// 응답 시간 통계
	if len(responseTimes) > 0 {
		summary.AverageResponseTime = time.Duration(int64(totalResponseTime) / int64(len(responseTimes)))

		// 응답 시간 정렬
		for i := 0; i < len(responseTimes)-1; i++ {
			for j := i + 1; j < len(responseTimes); j++ {
				if responseTimes[i] > responseTimes[j] {
					responseTimes[i], responseTimes[j] = responseTimes[j], responseTimes[i]
				}
			}
		}

		summary.MinResponseTime = responseTimes[0]
		summary.MaxResponseTime = responseTimes[len(responseTimes)-1]
		summary.P50ResponseTime = responseTimes[len(responseTimes)*50/100]
		summary.P95ResponseTime = responseTimes[len(responseTimes)*95/100]
		summary.P99ResponseTime = responseTimes[len(responseTimes)*99/100]
	}

	// 시나리오 요약 완성
	for name, stat := range scenarioStats {
		stat.ErrorRate = float64(stat.Failures) / float64(stat.Requests) * 100.0
		stat.Throughput = float64(stat.Requests) / summary.Duration.Seconds()
		summary.ScenarioResults[name] = *stat
	}

	return summary
}

func (pts *PerformanceTestSuite) generateRecommendations(summary *TestSummary) []string {
	recommendations := make([]string, 0)

	// 에러율 검사
	if summary.ErrorRate > pts.config.MaxErrorRate {
		recommendations = append(recommendations, fmt.Sprintf("에러율이 높습니다 (%.2f%% > %.2f%%). API 안정성을 확인하세요.", summary.ErrorRate, pts.config.MaxErrorRate))
	}

	// 처리량 검사
	if summary.Throughput < pts.config.MinThroughput {
		recommendations = append(recommendations, fmt.Sprintf("처리량이 낮습니다 (%.2f req/s < %.2f req/s). 성능 최적화를 고려하세요.", summary.Throughput, pts.config.MinThroughput))
	}

	// 응답 시간 검사
	if summary.AverageResponseTime > pts.config.MaxResponseTime {
		recommendations = append(recommendations, fmt.Sprintf("평균 응답 시간이 느립니다 (%v > %v). 데이터베이스 최적화나 캐싱을 고려하세요.", summary.AverageResponseTime, pts.config.MaxResponseTime))
	}

	// P95 응답 시간 검사
	if summary.P95ResponseTime > pts.config.MaxResponseTime*2 {
		recommendations = append(recommendations, "P95 응답 시간이 매우 높습니다. 이상 요청이나 느린 쿼리를 확인하세요.")
	}

	return recommendations
}

func (pts *PerformanceTestSuite) detectIssues(results []TestResult) []PerformanceIssue {
	issues := make([]PerformanceIssue, 0)
	errorCounts := make(map[string]int)
	firstSeen := make(map[string]time.Time)
	lastSeen := make(map[string]time.Time)

	// 에러 패턴 분석
	for _, result := range results {
		if !result.Success && result.Error != "" {
			errorCounts[result.Error]++
			if _, exists := firstSeen[result.Error]; !exists {
				firstSeen[result.Error] = result.StartTime
			}
			lastSeen[result.Error] = result.StartTime
		}
	}

	// 이슈 생성
	for errorMsg, count := range errorCounts {
		severity := "low"
		if count > 10 {
			severity = "high"
		} else if count > 5 {
			severity = "medium"
		}

		issue := PerformanceIssue{
			Type:        "error",
			Severity:    severity,
			Description: errorMsg,
			Count:       count,
			FirstSeen:   firstSeen[errorMsg],
			LastSeen:    lastSeen[errorMsg],
		}

		issues = append(issues, issue)
	}

	return issues
}
