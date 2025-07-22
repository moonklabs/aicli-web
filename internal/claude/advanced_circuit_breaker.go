package claude

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// AdvancedCircuitBreaker는 고급 Circuit Breaker 인터페이스입니다
type AdvancedCircuitBreaker interface {
	// 상태별 세밀한 제어
	GetState() CircuitState
	SetThresholds(thresholds CircuitThresholds) error
	
	// 부분적 실패 처리
	HandlePartialFailure(success int, failure int) error
	
	// 동적 임계값 조정
	AdjustThresholds(load float64) error
	
	// 복구 전략 실행
	ExecuteRecovery() error
	
	// 작업 실행
	Execute(ctx context.Context, operation CircuitOperation) error
	
	// 메트릭 조회
	GetMetrics() CircuitMetrics
	
	// 상태 변화 리스너 등록
	RegisterStateListener(listener StateChangeListener)
	
	// 수동 상태 변경
	ForceState(state CircuitState) error
	
	// 리셋
	Reset() error
}

// CircuitState는 Circuit Breaker 상태입니다
type CircuitState int

const (
	CircuitClosed CircuitState = iota
	CircuitHalfOpen
	CircuitOpen
	CircuitForcedOpen
	CircuitForcedClosed
)

// CircuitOperation은 Circuit Breaker로 보호되는 작업입니다
type CircuitOperation func(ctx context.Context) error

// StateChangeListener는 상태 변화 리스너입니다
type StateChangeListener func(oldState, newState CircuitState, metrics CircuitMetrics)

// CircuitThresholds는 Circuit Breaker 임계값입니다
type CircuitThresholds struct {
	FailureRate      float64       `json:"failure_rate"`      // 실패율 임계값 (0.0-1.0)
	SlowCallRate     float64       `json:"slow_call_rate"`    // 느린 호출 비율 임계값
	MinCalls         int           `json:"min_calls"`         // 최소 호출 수
	SlidingWindow    time.Duration `json:"sliding_window"`    // 슬라이딩 윈도우 크기
	HalfOpenMaxCalls int           `json:"half_open_max_calls"` // Half-open 상태 최대 호출 수
	SlowCallTimeout  time.Duration `json:"slow_call_timeout"` // 느린 호출 타임아웃
	OpenTimeout      time.Duration `json:"open_timeout"`      // Open 상태 유지 시간
	
	// 동적 조정 설정
	DynamicAdjustment bool    `json:"dynamic_adjustment"`
	LoadThreshold     float64 `json:"load_threshold"`
	AdjustmentFactor  float64 `json:"adjustment_factor"`
}

// CircuitMetrics는 Circuit Breaker 메트릭입니다
type CircuitMetrics struct {
	TotalCalls       int64     `json:"total_calls"`
	SuccessCalls     int64     `json:"success_calls"`
	FailureCalls     int64     `json:"failure_calls"`
	SlowCalls        int64     `json:"slow_calls"`
	RejectedCalls    int64     `json:"rejected_calls"`
	
	FailureRate      float64   `json:"failure_rate"`
	SlowCallRate     float64   `json:"slow_call_rate"`
	
	LastFailure      time.Time `json:"last_failure"`
	LastSuccess      time.Time `json:"last_success"`
	LastStateChange  time.Time `json:"last_state_change"`
	
	WindowStart      time.Time `json:"window_start"`
	WindowCalls      int64     `json:"window_calls"`
	WindowFailures   int64     `json:"window_failures"`
	WindowSlowCalls  int64     `json:"window_slow_calls"`
	
	// 응답 시간 통계
	AvgResponseTime  time.Duration `json:"avg_response_time"`
	P95ResponseTime  time.Duration `json:"p95_response_time"`
	P99ResponseTime  time.Duration `json:"p99_response_time"`
}

// StateTransition은 상태 전환 정보입니다
type StateTransition struct {
	FromState   CircuitState  `json:"from_state"`
	ToState     CircuitState  `json:"to_state"`
	Timestamp   time.Time     `json:"timestamp"`
	Reason      string        `json:"reason"`
	Metrics     CircuitMetrics `json:"metrics"`
}

// CallResult는 호출 결과입니다
type CallResult struct {
	Success      bool          `json:"success"`
	Duration     time.Duration `json:"duration"`
	Error        error         `json:"error"`
	Timestamp    time.Time     `json:"timestamp"`
}

// SmartCircuitBreaker는 고급 Circuit Breaker 구현체입니다
type SmartCircuitBreaker struct {
	// 기본 상태
	state         CircuitState
	stateMutex    sync.RWMutex
	
	// 설정
	thresholds    CircuitThresholds
	thresholdMutex sync.RWMutex
	
	// 메트릭
	metrics       CircuitMetrics
	metricsMutex  sync.RWMutex
	
	// 상태 기록
	stateHistory  []StateTransition
	historyMutex  sync.RWMutex
	maxHistory    int
	
	// 호출 기록 (슬라이딩 윈도우)
	callHistory   []CallResult
	callMutex     sync.RWMutex
	maxCalls      int
	
	// 응답 시간 추적
	responseTimes []time.Duration
	rtMutex       sync.RWMutex
	maxRTSamples  int
	
	// 리스너들
	listeners     []StateChangeListener
	listenerMutex sync.RWMutex
	
	// Half-open 상태 관리
	halfOpenCalls int32
	
	// 동적 조정
	loadBalancer  LoadBalancerMetrics
	
	// 생명주기
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
}

// LoadBalancerMetrics는 로드 밸런서 메트릭 인터페이스입니다
type LoadBalancerMetrics interface {
	GetCurrentLoad() float64
	GetAverageResponseTime() time.Duration
}

// DefaultCircuitThresholds는 기본 임계값을 반환합니다
func DefaultCircuitThresholds() CircuitThresholds {
	return CircuitThresholds{
		FailureRate:       0.5,  // 50% 실패율
		SlowCallRate:      0.3,  // 30% 느린 호출
		MinCalls:          10,   // 최소 10회 호출
		SlidingWindow:     60 * time.Second,
		HalfOpenMaxCalls:  5,    // Half-open에서 최대 5회 시도
		SlowCallTimeout:   5 * time.Second,
		OpenTimeout:       30 * time.Second,
		DynamicAdjustment: true,
		LoadThreshold:     0.8,  // 80% 부하
		AdjustmentFactor:  0.2,  // 20% 조정
	}
}

// NewSmartCircuitBreaker는 새로운 스마트 Circuit Breaker를 생성합니다
func NewSmartCircuitBreaker(thresholds CircuitThresholds) *SmartCircuitBreaker {
	ctx, cancel := context.WithCancel(context.Background())
	
	cb := &SmartCircuitBreaker{
		state:         CircuitClosed,
		thresholds:    thresholds,
		stateHistory:  make([]StateTransition, 0),
		maxHistory:    100,
		callHistory:   make([]CallResult, 0),
		maxCalls:      1000,
		responseTimes: make([]time.Duration, 0),
		maxRTSamples:  500,
		listeners:     make([]StateChangeListener, 0),
		ctx:           ctx,
		cancel:        cancel,
		metrics: CircuitMetrics{
			WindowStart: time.Now(),
		},
	}
	
	// 백그라운드 작업 시작
	cb.wg.Add(2)
	go cb.metricsUpdater()
	go cb.stateMonitor()
	
	return cb
}

// GetState는 현재 상태를 반환합니다
func (cb *SmartCircuitBreaker) GetState() CircuitState {
	cb.stateMutex.RLock()
	defer cb.stateMutex.RUnlock()
	return cb.state
}

// SetThresholds는 임계값을 설정합니다
func (cb *SmartCircuitBreaker) SetThresholds(thresholds CircuitThresholds) error {
	if thresholds.FailureRate < 0 || thresholds.FailureRate > 1 {
		return fmt.Errorf("failure rate must be between 0 and 1")
	}
	
	if thresholds.SlowCallRate < 0 || thresholds.SlowCallRate > 1 {
		return fmt.Errorf("slow call rate must be between 0 and 1")
	}
	
	if thresholds.MinCalls <= 0 {
		return fmt.Errorf("min calls must be positive")
	}
	
	cb.thresholdMutex.Lock()
	cb.thresholds = thresholds
	cb.thresholdMutex.Unlock()
	
	return nil
}

// HandlePartialFailure는 부분적 실패를 처리합니다
func (cb *SmartCircuitBreaker) HandlePartialFailure(success int, failure int) error {
	if success < 0 || failure < 0 {
		return fmt.Errorf("success and failure counts must be non-negative")
	}
	
	now := time.Now()
	
	// 성공 호출 기록
	for i := 0; i < success; i++ {
		cb.recordCall(CallResult{
			Success:   true,
			Duration:  0, // 부분 실패에서는 개별 시간 미제공
			Timestamp: now,
		})
	}
	
	// 실패 호출 기록
	for i := 0; i < failure; i++ {
		cb.recordCall(CallResult{
			Success:   false,
			Duration:  0,
			Timestamp: now,
		})
	}
	
	// 상태 평가
	cb.evaluateState()
	
	return nil
}

// AdjustThresholds는 부하에 따라 동적으로 임계값을 조정합니다
func (cb *SmartCircuitBreaker) AdjustThresholds(load float64) error {
	cb.thresholdMutex.RLock()
	if !cb.thresholds.DynamicAdjustment {
		cb.thresholdMutex.RUnlock()
		return nil
	}
	
	loadThreshold := cb.thresholds.LoadThreshold
	adjustmentFactor := cb.thresholds.AdjustmentFactor
	cb.thresholdMutex.RUnlock()
	
	if load < loadThreshold {
		return nil // 조정 불필요
	}
	
	// 부하가 높으면 더 엄격한 임계값 적용
	cb.thresholdMutex.Lock()
	defer cb.thresholdMutex.Unlock()
	
	// 실패율 임계값 감소 (더 민감하게)
	newFailureRate := cb.thresholds.FailureRate * (1.0 - adjustmentFactor)
	if newFailureRate < 0.1 {
		newFailureRate = 0.1 // 최소값 제한
	}
	
	// 느린 호출 임계값 감소
	newSlowCallRate := cb.thresholds.SlowCallRate * (1.0 - adjustmentFactor)
	if newSlowCallRate < 0.1 {
		newSlowCallRate = 0.1
	}
	
	cb.thresholds.FailureRate = newFailureRate
	cb.thresholds.SlowCallRate = newSlowCallRate
	
	return nil
}

// ExecuteRecovery는 복구 전략을 실행합니다
func (cb *SmartCircuitBreaker) ExecuteRecovery() error {
	currentState := cb.GetState()
	
	switch currentState {
	case CircuitOpen:
		// Open → Half-open 전환 시도
		return cb.transitionToHalfOpen()
	case CircuitHalfOpen:
		// Half-open에서 상태 재평가
		cb.evaluateState()
		return nil
	case CircuitForcedOpen:
		return fmt.Errorf("circuit is forced open, manual intervention required")
	default:
		return nil // 복구 불필요
	}
}

// Execute는 Circuit Breaker로 보호된 작업을 실행합니다
func (cb *SmartCircuitBreaker) Execute(ctx context.Context, operation CircuitOperation) error {
	// 호출 허용 여부 확인
	if !cb.allowCall() {
		atomic.AddInt64(&cb.metrics.RejectedCalls, 1)
		return fmt.Errorf("circuit breaker is open")
	}
	
	start := time.Now()
	err := operation(ctx)
	duration := time.Since(start)
	
	// 결과 기록
	cb.recordCall(CallResult{
		Success:   err == nil,
		Duration:  duration,
		Error:     err,
		Timestamp: time.Now(),
	})
	
	// 응답 시간 기록
	cb.recordResponseTime(duration)
	
	// 상태 평가
	cb.evaluateState()
	
	return err
}

// GetMetrics는 메트릭을 반환합니다
func (cb *SmartCircuitBreaker) GetMetrics() CircuitMetrics {
	cb.metricsMutex.RLock()
	defer cb.metricsMutex.RUnlock()
	
	// 복사본 반환
	metrics := cb.metrics
	return metrics
}

// RegisterStateListener는 상태 변화 리스너를 등록합니다
func (cb *SmartCircuitBreaker) RegisterStateListener(listener StateChangeListener) {
	cb.listenerMutex.Lock()
	cb.listeners = append(cb.listeners, listener)
	cb.listenerMutex.Unlock()
}

// ForceState는 수동으로 상태를 변경합니다
func (cb *SmartCircuitBreaker) ForceState(state CircuitState) error {
	oldState := cb.GetState()
	
	cb.stateMutex.Lock()
	cb.state = state
	cb.stateMutex.Unlock()
	
	// 상태 전환 기록
	cb.recordStateTransition(oldState, state, "manual force")
	
	// 리스너 알림
	cb.notifyStateChange(oldState, state)
	
	return nil
}

// Reset은 Circuit Breaker를 리셋합니다
func (cb *SmartCircuitBreaker) Reset() error {
	cb.stateMutex.Lock()
	oldState := cb.state
	cb.state = CircuitClosed
	cb.stateMutex.Unlock()
	
	cb.metricsMutex.Lock()
	cb.metrics = CircuitMetrics{
		WindowStart: time.Now(),
	}
	cb.metricsMutex.Unlock()
	
	cb.callMutex.Lock()
	cb.callHistory = cb.callHistory[:0]
	cb.callMutex.Unlock()
	
	cb.rtMutex.Lock()
	cb.responseTimes = cb.responseTimes[:0]
	cb.rtMutex.Unlock()
	
	atomic.StoreInt32(&cb.halfOpenCalls, 0)
	
	// 상태 전환 기록
	cb.recordStateTransition(oldState, CircuitClosed, "manual reset")
	
	// 리스너 알림
	cb.notifyStateChange(oldState, CircuitClosed)
	
	return nil
}

// 내부 메서드들

func (cb *SmartCircuitBreaker) allowCall() bool {
	state := cb.GetState()
	
	switch state {
	case CircuitClosed, CircuitForcedClosed:
		return true
	case CircuitOpen, CircuitForcedOpen:
		// Open 상태에서는 시간이 지났는지 확인
		if state == CircuitOpen {
			cb.metricsMutex.RLock()
			elapsed := time.Since(cb.metrics.LastStateChange)
			openTimeout := cb.thresholds.OpenTimeout
			cb.metricsMutex.RUnlock()
			
			if elapsed > openTimeout {
				// Half-open으로 전환 시도
				cb.transitionToHalfOpen()
				return true
			}
		}
		return false
	case CircuitHalfOpen:
		// Half-open에서는 제한된 호출만 허용
		cb.thresholdMutex.RLock()
		maxCalls := cb.thresholds.HalfOpenMaxCalls
		cb.thresholdMutex.RUnlock()
		
		currentCalls := atomic.LoadInt32(&cb.halfOpenCalls)
		if int(currentCalls) < maxCalls {
			atomic.AddInt32(&cb.halfOpenCalls, 1)
			return true
		}
		return false
	default:
		return false
	}
}

func (cb *SmartCircuitBreaker) recordCall(result CallResult) {
	cb.callMutex.Lock()
	defer cb.callMutex.Unlock()
	
	// 호출 기록 추가
	cb.callHistory = append(cb.callHistory, result)
	
	// 최대 기록 수 제한
	if len(cb.callHistory) > cb.maxCalls {
		cb.callHistory = cb.callHistory[1:]
	}
	
	// 메트릭 업데이트
	cb.updateMetrics(result)
}

func (cb *SmartCircuitBreaker) updateMetrics(result CallResult) {
	cb.metricsMutex.Lock()
	defer cb.metricsMutex.Unlock()
	
	cb.metrics.TotalCalls++
	
	if result.Success {
		cb.metrics.SuccessCalls++
		cb.metrics.LastSuccess = result.Timestamp
	} else {
		cb.metrics.FailureCalls++
		cb.metrics.LastFailure = result.Timestamp
	}
	
	// 느린 호출 확인
	cb.thresholdMutex.RLock()
	slowTimeout := cb.thresholds.SlowCallTimeout
	cb.thresholdMutex.RUnlock()
	
	if result.Duration > slowTimeout {
		cb.metrics.SlowCalls++
	}
	
	// 윈도우 기반 메트릭 업데이트
	cb.updateWindowMetrics()
}

func (cb *SmartCircuitBreaker) updateWindowMetrics() {
	now := time.Now()
	
	cb.thresholdMutex.RLock()
	windowSize := cb.thresholds.SlidingWindow
	cb.thresholdMutex.RUnlock()
	
	windowStart := now.Add(-windowSize)
	
	var windowCalls, windowFailures, windowSlowCalls int64
	
	cb.callMutex.RLock()
	for _, call := range cb.callHistory {
		if call.Timestamp.After(windowStart) {
			windowCalls++
			if !call.Success {
				windowFailures++
			}
			
			cb.thresholdMutex.RLock()
			slowTimeout := cb.thresholds.SlowCallTimeout
			cb.thresholdMutex.RUnlock()
			
			if call.Duration > slowTimeout {
				windowSlowCalls++
			}
		}
	}
	cb.callMutex.RUnlock()
	
	cb.metrics.WindowStart = windowStart
	cb.metrics.WindowCalls = windowCalls
	cb.metrics.WindowFailures = windowFailures
	cb.metrics.WindowSlowCalls = windowSlowCalls
	
	// 비율 계산
	if windowCalls > 0 {
		cb.metrics.FailureRate = float64(windowFailures) / float64(windowCalls)
		cb.metrics.SlowCallRate = float64(windowSlowCalls) / float64(windowCalls)
	} else {
		cb.metrics.FailureRate = 0
		cb.metrics.SlowCallRate = 0
	}
}

func (cb *SmartCircuitBreaker) recordResponseTime(duration time.Duration) {
	cb.rtMutex.Lock()
	defer cb.rtMutex.Unlock()
	
	cb.responseTimes = append(cb.responseTimes, duration)
	
	// 최대 샘플 수 제한
	if len(cb.responseTimes) > cb.maxRTSamples {
		cb.responseTimes = cb.responseTimes[1:]
	}
	
	// 통계 계산
	cb.calculateResponseTimeStats()
}

func (cb *SmartCircuitBreaker) calculateResponseTimeStats() {
	if len(cb.responseTimes) == 0 {
		return
	}
	
	// 평균 계산
	var total time.Duration
	for _, rt := range cb.responseTimes {
		total += rt
	}
	
	cb.metricsMutex.Lock()
	cb.metrics.AvgResponseTime = total / time.Duration(len(cb.responseTimes))
	
	// 퍼센타일 계산 (간단한 구현)
	sorted := make([]time.Duration, len(cb.responseTimes))
	copy(sorted, cb.responseTimes)
	
	// 간단한 버블 정렬 (성능보다 정확성 우선)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	
	p95Index := int(float64(len(sorted)) * 0.95)
	p99Index := int(float64(len(sorted)) * 0.99)
	
	if p95Index < len(sorted) {
		cb.metrics.P95ResponseTime = sorted[p95Index]
	}
	if p99Index < len(sorted) {
		cb.metrics.P99ResponseTime = sorted[p99Index]
	}
	
	cb.metricsMutex.Unlock()
}

func (cb *SmartCircuitBreaker) evaluateState() {
	currentState := cb.GetState()
	
	cb.metricsMutex.RLock()
	windowCalls := cb.metrics.WindowCalls
	failureRate := cb.metrics.FailureRate
	slowCallRate := cb.metrics.SlowCallRate
	cb.metricsMutex.RUnlock()
	
	cb.thresholdMutex.RLock()
	minCalls := cb.thresholds.MinCalls
	failureThreshold := cb.thresholds.FailureRate
	slowCallThreshold := cb.thresholds.SlowCallRate
	cb.thresholdMutex.RUnlock()
	
	// 충분한 호출이 있는 경우에만 평가
	if windowCalls < int64(minCalls) {
		return
	}
	
	switch currentState {
	case CircuitClosed:
		// Closed → Open 전환 조건
		if failureRate >= failureThreshold || slowCallRate >= slowCallThreshold {
			cb.transitionToOpen("threshold exceeded")
		}
	case CircuitHalfOpen:
		// Half-open에서의 평가
		halfOpenCalls := atomic.LoadInt32(&cb.halfOpenCalls)
		if halfOpenCalls >= int32(cb.thresholds.HalfOpenMaxCalls) {
			// 충분한 호출이 있으면 상태 결정
			if failureRate < failureThreshold && slowCallRate < slowCallThreshold {
				cb.transitionToClosed("recovery successful")
			} else {
				cb.transitionToOpen("recovery failed")
			}
		}
	}
}

func (cb *SmartCircuitBreaker) transitionToOpen(reason string) {
	oldState := cb.GetState()
	
	cb.stateMutex.Lock()
	cb.state = CircuitOpen
	cb.stateMutex.Unlock()
	
	cb.metricsMutex.Lock()
	cb.metrics.LastStateChange = time.Now()
	cb.metricsMutex.Unlock()
	
	cb.recordStateTransition(oldState, CircuitOpen, reason)
	cb.notifyStateChange(oldState, CircuitOpen)
}

func (cb *SmartCircuitBreaker) transitionToHalfOpen() error {
	oldState := cb.GetState()
	
	cb.stateMutex.Lock()
	cb.state = CircuitHalfOpen
	cb.stateMutex.Unlock()
	
	cb.metricsMutex.Lock()
	cb.metrics.LastStateChange = time.Now()
	cb.metricsMutex.Unlock()
	
	atomic.StoreInt32(&cb.halfOpenCalls, 0)
	
	cb.recordStateTransition(oldState, CircuitHalfOpen, "attempting recovery")
	cb.notifyStateChange(oldState, CircuitHalfOpen)
	
	return nil
}

func (cb *SmartCircuitBreaker) transitionToClosed(reason string) {
	oldState := cb.GetState()
	
	cb.stateMutex.Lock()
	cb.state = CircuitClosed
	cb.stateMutex.Unlock()
	
	cb.metricsMutex.Lock()
	cb.metrics.LastStateChange = time.Now()
	cb.metricsMutex.Unlock()
	
	atomic.StoreInt32(&cb.halfOpenCalls, 0)
	
	cb.recordStateTransition(oldState, CircuitClosed, reason)
	cb.notifyStateChange(oldState, CircuitClosed)
}

func (cb *SmartCircuitBreaker) recordStateTransition(from, to CircuitState, reason string) {
	transition := StateTransition{
		FromState: from,
		ToState:   to,
		Timestamp: time.Now(),
		Reason:    reason,
		Metrics:   cb.GetMetrics(),
	}
	
	cb.historyMutex.Lock()
	cb.stateHistory = append(cb.stateHistory, transition)
	if len(cb.stateHistory) > cb.maxHistory {
		cb.stateHistory = cb.stateHistory[1:]
	}
	cb.historyMutex.Unlock()
}

func (cb *SmartCircuitBreaker) notifyStateChange(oldState, newState CircuitState) {
	cb.listenerMutex.RLock()
	listeners := make([]StateChangeListener, len(cb.listeners))
	copy(listeners, cb.listeners)
	cb.listenerMutex.RUnlock()
	
	metrics := cb.GetMetrics()
	
	for _, listener := range listeners {
		go listener(oldState, newState, metrics)
	}
}

func (cb *SmartCircuitBreaker) metricsUpdater() {
	defer cb.wg.Done()
	
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-cb.ctx.Done():
			return
		case <-ticker.C:
			cb.metricsMutex.Lock()
			cb.updateWindowMetrics()
			cb.metricsMutex.Unlock()
		}
	}
}

func (cb *SmartCircuitBreaker) stateMonitor() {
	defer cb.wg.Done()
	
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-cb.ctx.Done():
			return
		case <-ticker.C:
			cb.evaluateState()
			
			// 동적 조정 (로드 밸런서가 있는 경우)
			if cb.loadBalancer != nil {
				load := cb.loadBalancer.GetCurrentLoad()
				cb.AdjustThresholds(load)
			}
		}
	}
}

// Shutdown은 Circuit Breaker를 정리합니다
func (cb *SmartCircuitBreaker) Shutdown() {
	cb.cancel()
	cb.wg.Wait()
}