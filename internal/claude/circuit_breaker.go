package claude

import (
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// CircuitBreakerState 회로 차단기 상태
type CircuitBreakerState int

const (
	// StateClosed 회로가 닫힌 상태 (정상 동작)
	StateClosed CircuitBreakerState = iota
	// StateOpen 회로가 열린 상태 (요청 차단)
	StateOpen
	// StateHalfOpen 회로가 반열린 상태 (제한적 요청 허용)
	StateHalfOpen
)

// String CircuitBreakerState를 문자열로 변환
func (s CircuitBreakerState) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreakerConfig 회로 차단기 설정
type CircuitBreakerConfig struct {
	// FailureThreshold 실패 임계값
	FailureThreshold int `yaml:"failure_threshold" json:"failure_threshold"`
	// RecoveryTimeout 복구 타임아웃
	RecoveryTimeout time.Duration `yaml:"recovery_timeout" json:"recovery_timeout"`
	// SuccessThreshold 성공 임계값 (half-open에서 closed로 전환)
	SuccessThreshold int `yaml:"success_threshold" json:"success_threshold"`
	// RequestVolumeThreshold 요청 볼륨 임계값
	RequestVolumeThreshold int `yaml:"request_volume_threshold" json:"request_volume_threshold"`
	// ErrorPercentageThreshold 에러 비율 임계값
	ErrorPercentageThreshold float64 `yaml:"error_percentage_threshold" json:"error_percentage_threshold"`
}

// CircuitBreakerStats 회로 차단기 통계
type CircuitBreakerStats struct {
	// State 현재 상태
	State CircuitBreakerState `json:"state"`
	// FailureCount 실패 횟수
	FailureCount int64 `json:"failure_count"`
	// SuccessCount 성공 횟수
	SuccessCount int64 `json:"success_count"`
	// LastFailure 마지막 실패 시간
	LastFailure time.Time `json:"last_failure"`
	// NextAttempt 다음 시도 허용 시간
	NextAttempt time.Time `json:"next_attempt"`
	// TotalRequests 총 요청 수
	TotalRequests int64 `json:"total_requests"`
	// StateChanges 상태 변경 횟수
	StateChanges int64 `json:"state_changes"`
	// ErrorPercentage 에러 비율
	ErrorPercentage float64 `json:"error_percentage"`
}

// CircuitBreaker 회로 차단기 인터페이스
type CircuitBreaker interface {
	// Allow 요청 허용 여부를 확인합니다
	Allow() bool
	// RecordSuccess 성공을 기록합니다
	RecordSuccess()
	// RecordError 실패를 기록합니다
	RecordError()
	// State 현재 상태를 반환합니다
	State() CircuitBreakerState
	// Stats 통계를 반환합니다
	Stats() CircuitBreakerStats
	// Reset 회로 차단기를 초기화합니다
	Reset()
	// IsOpen 회로가 열린 상태인지 확인합니다
	IsOpen() bool
	// IsHalfOpen 회로가 반열린 상태인지 확인합니다
	IsHalfOpen() bool
	// IsClosed 회로가 닫힌 상태인지 확인합니다
	IsClosed() bool
}

// circuitBreaker 회로 차단기 구현
type circuitBreaker struct {
	config       *CircuitBreakerConfig
	state        CircuitBreakerState
	failureCount int64
	successCount int64
	lastFailure  time.Time
	nextAttempt  time.Time
	totalRequests int64
	stateChanges int64
	mutex        sync.RWMutex
	logger       *logrus.Logger
	
	// 슬라이딩 윈도우를 위한 필드
	window       *slidingWindow
	windowSize   time.Duration
}

// NewCircuitBreaker 새로운 회로 차단기를 생성합니다
func NewCircuitBreaker(config *CircuitBreakerConfig, logger *logrus.Logger) CircuitBreaker {
	if config == nil {
		config = &CircuitBreakerConfig{
			FailureThreshold:         5,
			RecoveryTimeout:          60 * time.Second,
			SuccessThreshold:         3,
			RequestVolumeThreshold:   10,
			ErrorPercentageThreshold: 50.0,
		}
	}
	
	if logger == nil {
		logger = logrus.StandardLogger()
	}
	
	windowSize := 1 * time.Minute // 1분 슬라이딩 윈도우
	
	return &circuitBreaker{
		config:     config,
		state:      StateClosed,
		logger:     logger,
		window:     newSlidingWindow(windowSize),
		windowSize: windowSize,
	}
}

// Allow 요청 허용 여부를 확인합니다
func (cb *circuitBreaker) Allow() bool {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	
	cb.totalRequests++
	
	now := time.Now()
	
	switch cb.state {
	case StateClosed:
		return true
		
	case StateOpen:
		// 복구 타임아웃이 지났는지 확인
		if now.After(cb.nextAttempt) {
			cb.transitionToHalfOpen()
			return true
		}
		cb.logger.Debug("Circuit breaker is open, rejecting request")
		return false
		
	case StateHalfOpen:
		// half-open 상태에서는 제한적으로 요청 허용
		return true
		
	default:
		return false
	}
}

// RecordSuccess 성공을 기록합니다
func (cb *circuitBreaker) RecordSuccess() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	
	cb.successCount++
	cb.window.RecordSuccess()
	
	if cb.state == StateHalfOpen {
		if cb.successCount >= int64(cb.config.SuccessThreshold) {
			cb.transitionToClosed()
		}
	}
	
	cb.logger.WithFields(logrus.Fields{
		"state":         cb.state,
		"success_count": cb.successCount,
	}).Debug("Recorded success in circuit breaker")
}

// RecordError 실패를 기록합니다
func (cb *circuitBreaker) RecordError() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	
	cb.failureCount++
	cb.lastFailure = time.Now()
	cb.window.RecordFailure()
	
	cb.logger.WithFields(logrus.Fields{
		"state":         cb.state,
		"failure_count": cb.failureCount,
	}).Debug("Recorded error in circuit breaker")
	
	// 상태에 따른 처리
	switch cb.state {
	case StateClosed:
		if cb.shouldOpenCircuit() {
			cb.transitionToOpen()
		}
		
	case StateHalfOpen:
		// half-open 상태에서 실패하면 바로 open으로 전환
		cb.transitionToOpen()
	}
}

// shouldOpenCircuit 회로를 열어야 하는지 확인합니다
func (cb *circuitBreaker) shouldOpenCircuit() bool {
	// 최소 요청 수 확인
	if cb.window.TotalRequests() < int64(cb.config.RequestVolumeThreshold) {
		return false
	}
	
	// 에러 비율 확인
	errorPercentage := cb.window.ErrorPercentage()
	if errorPercentage >= cb.config.ErrorPercentageThreshold {
		return true
	}
	
	// 연속 실패 횟수 확인
	if cb.failureCount >= int64(cb.config.FailureThreshold) {
		return true
	}
	
	return false
}

// transitionToClosed closed 상태로 전환합니다
func (cb *circuitBreaker) transitionToClosed() {
	cb.state = StateClosed
	cb.failureCount = 0
	cb.successCount = 0
	cb.stateChanges++
	
	cb.logger.WithField("state", cb.state).Info("Circuit breaker transitioned to CLOSED")
}

// transitionToOpen open 상태로 전환합니다
func (cb *circuitBreaker) transitionToOpen() {
	cb.state = StateOpen
	cb.nextAttempt = time.Now().Add(cb.config.RecoveryTimeout)
	cb.stateChanges++
	
	cb.logger.WithFields(logrus.Fields{
		"state":        cb.state,
		"next_attempt": cb.nextAttempt,
	}).Warn("Circuit breaker transitioned to OPEN")
}

// transitionToHalfOpen half-open 상태로 전환합니다
func (cb *circuitBreaker) transitionToHalfOpen() {
	cb.state = StateHalfOpen
	cb.successCount = 0
	cb.stateChanges++
	
	cb.logger.WithField("state", cb.state).Info("Circuit breaker transitioned to HALF-OPEN")
}

// State 현재 상태를 반환합니다
func (cb *circuitBreaker) State() CircuitBreakerState {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.state
}

// Stats 통계를 반환합니다
func (cb *circuitBreaker) Stats() CircuitBreakerStats {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	
	return CircuitBreakerStats{
		State:           cb.state,
		FailureCount:    cb.failureCount,
		SuccessCount:    cb.successCount,
		LastFailure:     cb.lastFailure,
		NextAttempt:     cb.nextAttempt,
		TotalRequests:   cb.totalRequests,
		StateChanges:    cb.stateChanges,
		ErrorPercentage: cb.window.ErrorPercentage(),
	}
}

// Reset 회로 차단기를 초기화합니다
func (cb *circuitBreaker) Reset() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	
	cb.state = StateClosed
	cb.failureCount = 0
	cb.successCount = 0
	cb.lastFailure = time.Time{}
	cb.nextAttempt = time.Time{}
	cb.totalRequests = 0
	cb.stateChanges = 0
	cb.window.Reset()
	
	cb.logger.Info("Circuit breaker has been reset")
}

// IsOpen 회로가 열린 상태인지 확인합니다
func (cb *circuitBreaker) IsOpen() bool {
	return cb.State() == StateOpen
}

// IsHalfOpen 회로가 반열린 상태인지 확인합니다
func (cb *circuitBreaker) IsHalfOpen() bool {
	return cb.State() == StateHalfOpen
}

// IsClosed 회로가 닫힌 상태인지 확인합니다
func (cb *circuitBreaker) IsClosed() bool {
	return cb.State() == StateClosed
}

// slidingWindow 슬라이딩 윈도우 구현
type slidingWindow struct {
	buckets    []bucket
	bucketSize time.Duration
	windowSize time.Duration
	current    int
	mutex      sync.RWMutex
}

// bucket 시간 버킷
type bucket struct {
	timestamp time.Time
	requests  int64
	failures  int64
}

// newSlidingWindow 새로운 슬라이딩 윈도우를 생성합니다
func newSlidingWindow(windowSize time.Duration) *slidingWindow {
	bucketCount := 10
	bucketSize := windowSize / time.Duration(bucketCount)
	
	return &slidingWindow{
		buckets:    make([]bucket, bucketCount),
		bucketSize: bucketSize,
		windowSize: windowSize,
	}
}

// getCurrentBucket 현재 버킷을 반환합니다
func (sw *slidingWindow) getCurrentBucket() *bucket {
	now := time.Now()
	bucketIndex := int(now.UnixNano()/int64(sw.bucketSize)) % len(sw.buckets)
	
	bucket := &sw.buckets[bucketIndex]
	
	// 버킷이 오래된 경우 초기화
	if now.Sub(bucket.timestamp) > sw.bucketSize {
		bucket.timestamp = now
		bucket.requests = 0
		bucket.failures = 0
	}
	
	return bucket
}

// RecordSuccess 성공을 기록합니다
func (sw *slidingWindow) RecordSuccess() {
	sw.mutex.Lock()
	defer sw.mutex.Unlock()
	
	bucket := sw.getCurrentBucket()
	bucket.requests++
}

// RecordFailure 실패를 기록합니다
func (sw *slidingWindow) RecordFailure() {
	sw.mutex.Lock()
	defer sw.mutex.Unlock()
	
	bucket := sw.getCurrentBucket()
	bucket.requests++
	bucket.failures++
}

// TotalRequests 총 요청 수를 반환합니다
func (sw *slidingWindow) TotalRequests() int64 {
	sw.mutex.RLock()
	defer sw.mutex.RUnlock()
	
	var total int64
	now := time.Now()
	
	for _, bucket := range sw.buckets {
		if now.Sub(bucket.timestamp) <= sw.windowSize {
			total += bucket.requests
		}
	}
	
	return total
}

// TotalFailures 총 실패 수를 반환합니다
func (sw *slidingWindow) TotalFailures() int64 {
	sw.mutex.RLock()
	defer sw.mutex.RUnlock()
	
	var total int64
	now := time.Now()
	
	for _, bucket := range sw.buckets {
		if now.Sub(bucket.timestamp) <= sw.windowSize {
			total += bucket.failures
		}
	}
	
	return total
}

// ErrorPercentage 에러 비율을 반환합니다
func (sw *slidingWindow) ErrorPercentage() float64 {
	totalRequests := sw.TotalRequests()
	if totalRequests == 0 {
		return 0
	}
	
	totalFailures := sw.TotalFailures()
	return (float64(totalFailures) / float64(totalRequests)) * 100
}

// Reset 슬라이딩 윈도우를 초기화합니다
func (sw *slidingWindow) Reset() {
	sw.mutex.Lock()
	defer sw.mutex.Unlock()
	
	for i := range sw.buckets {
		sw.buckets[i] = bucket{}
	}
}