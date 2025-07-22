package claude

import (
	"math"
	"math/rand"
	"strings"
	"time"
)

// BackoffCalculator는 백오프 지연 시간을 계산하는 인터페이스입니다
type BackoffCalculator interface {
	// 백오프 지연 시간 계산
	Calculate(attempt int, baseDelay, maxDelay time.Duration, strategy BackoffType, err error) time.Duration
	
	// 부하에 따른 지연 시간 조정
	AdjustForLoad(delay time.Duration, load float64) time.Duration
	
	// 에러 유형에 따른 지연 시간 조정
	AdjustForErrorType(delay time.Duration, errorType ErrorType) time.Duration
	
	// Jitter 적용
	ApplyJitter(delay time.Duration, jitterType JitterType, factor float64) time.Duration
}

// JitterType은 Jitter 유형입니다
type JitterType int

const (
	NoJitter JitterType = iota
	FullJitter
	EqualJitter
	DecorrelatedJitter
	ExponentialJitter
)

// SmartBackoffCalculator는 지능형 백오프 계산기 구현체입니다
type SmartBackoffCalculator struct {
	// 에러 유형별 가중치
	errorWeights map[ErrorType]float64
	
	// 시스템 부하 인식
	loadAware bool
	
	// 최근 지연 시간 추적
	recentDelays []time.Duration
	maxSamples   int
	
	// 적응형 계수
	adaptiveFactors map[BackoffType]float64
}

// NewSmartBackoffCalculator는 새로운 스마트 백오프 계산기를 생성합니다
func NewSmartBackoffCalculator() *SmartBackoffCalculator {
	return &SmartBackoffCalculator{
		errorWeights: map[ErrorType]float64{
			NetworkError:    1.0,
			ProcessError:    1.5,
			AuthError:      0.5, // 빠른 실패
			ResourceError:  2.0, // 긴 대기
			TimeoutError:   1.2,
			ValidationError: 0.3, // 매우 빠른 실패
			InternalError:  1.3,
			ConfigError:    0.8,
			DependencyError: 1.4,
			QuotaError:     3.0, // 매우 긴 대기
			UnknownError:   1.0,
		},
		loadAware:    true,
		recentDelays: make([]time.Duration, 0, 100),
		maxSamples:   100,
		adaptiveFactors: map[BackoffType]float64{
			LinearBackoff:             1.0,
			ExponentialBackoff:        1.0,
			FixedDelayBackoff:        1.0,
			AdaptiveBackoff:          1.0,
			DecorrelatedJitterBackoff: 1.0,
		},
	}
}

// Calculate는 백오프 지연 시간을 계산합니다
func (c *SmartBackoffCalculator) Calculate(
	attempt int, 
	baseDelay, maxDelay time.Duration, 
	strategy BackoffType, 
	err error,
) time.Duration {
	if attempt <= 0 {
		return 0
	}
	
	var delay time.Duration
	
	switch strategy {
	case LinearBackoff:
		delay = c.calculateLinearBackoff(attempt, baseDelay)
	case ExponentialBackoff:
		delay = c.calculateExponentialBackoff(attempt, baseDelay)
	case FixedDelayBackoff:
		delay = baseDelay
	case AdaptiveBackoff:
		delay = c.calculateAdaptiveBackoff(attempt, baseDelay)
	case DecorrelatedJitterBackoff:
		delay = c.calculateDecorrelatedJitterBackoff(attempt, baseDelay)
	default:
		delay = c.calculateExponentialBackoff(attempt, baseDelay)
	}
	
	// 최대 지연 시간 제한
	if delay > maxDelay {
		delay = maxDelay
	}
	
	// 에러 유형에 따른 조정
	if err != nil {
		// 에러 분류가 필요한 경우, 여기서는 간단한 휴리스틱 사용
		errorType := c.classifyErrorSimple(err)
		delay = c.AdjustForErrorType(delay, errorType)
	}
	
	// 최근 지연 시간 기록
	c.recordDelay(delay)
	
	return delay
}

// AdjustForLoad는 부하에 따라 지연 시간을 조정합니다
func (c *SmartBackoffCalculator) AdjustForLoad(delay time.Duration, load float64) time.Duration {
	if !c.loadAware || load <= 0 {
		return delay
	}
	
	// 부하가 높을수록 지연 시간 증가
	// load: 0.0 (부하 없음) ~ 1.0 (최대 부하)
	var adjustmentFactor float64
	
	switch {
	case load < 0.3:
		adjustmentFactor = 0.8 // 부하가 낮으면 지연 감소
	case load < 0.6:
		adjustmentFactor = 1.0 // 보통 부하
	case load < 0.8:
		adjustmentFactor = 1.5 // 높은 부하
	default:
		adjustmentFactor = 2.0 // 매우 높은 부하
	}
	
	return time.Duration(float64(delay) * adjustmentFactor)
}

// AdjustForErrorType은 에러 유형에 따라 지연 시간을 조정합니다
func (c *SmartBackoffCalculator) AdjustForErrorType(delay time.Duration, errorType ErrorType) time.Duration {
	weight, exists := c.errorWeights[errorType]
	if !exists {
		weight = 1.0
	}
	
	return time.Duration(float64(delay) * weight)
}

// ApplyJitter는 지터를 적용합니다
func (c *SmartBackoffCalculator) ApplyJitter(
	delay time.Duration, 
	jitterType JitterType, 
	factor float64,
) time.Duration {
	if jitterType == NoJitter || factor <= 0 {
		return delay
	}
	
	switch jitterType {
	case FullJitter:
		return c.applyFullJitter(delay, factor)
	case EqualJitter:
		return c.applyEqualJitter(delay, factor)
	case DecorrelatedJitter:
		return c.applyDecorrelatedJitter(delay, factor)
	case ExponentialJitter:
		return c.applyExponentialJitter(delay, factor)
	default:
		return c.applyFullJitter(delay, factor)
	}
}

// 내부 계산 메서드들

func (c *SmartBackoffCalculator) calculateLinearBackoff(attempt int, baseDelay time.Duration) time.Duration {
	return time.Duration(int64(baseDelay) * int64(attempt))
}

func (c *SmartBackoffCalculator) calculateExponentialBackoff(attempt int, baseDelay time.Duration) time.Duration {
	// 2^(attempt-1) * baseDelay
	multiplier := math.Pow(2, float64(attempt-1))
	
	// 오버플로우 방지
	if multiplier > 1000 {
		multiplier = 1000
	}
	
	return time.Duration(float64(baseDelay) * multiplier)
}

func (c *SmartBackoffCalculator) calculateAdaptiveBackoff(attempt int, baseDelay time.Duration) time.Duration {
	// 최근 지연 시간의 평균을 기반으로 조정
	avgDelay := c.getAverageRecentDelay()
	
	if avgDelay == 0 {
		// 데이터가 없으면 지수 백오프 사용
		return c.calculateExponentialBackoff(attempt, baseDelay)
	}
	
	// 평균 지연 시간과 시도 횟수를 고려한 적응형 계산
	adaptiveFactor := 1.0 + float64(attempt)*0.2
	if avgDelay > baseDelay {
		// 평균이 기본값보다 크면 더 보수적으로
		adaptiveFactor *= 1.3
	} else {
		// 평균이 기본값보다 작으면 더 적극적으로
		adaptiveFactor *= 0.8
	}
	
	return time.Duration(float64(baseDelay) * adaptiveFactor)
}

func (c *SmartBackoffCalculator) calculateDecorrelatedJitterBackoff(attempt int, baseDelay time.Duration) time.Duration {
	// AWS의 decorrelated jitter 알고리즘
	// delay = random(baseDelay, prevDelay * 3)
	
	if attempt == 1 {
		return baseDelay
	}
	
	// 이전 지연 시간 추정 (간단한 지수 백오프 사용)
	prevDelay := c.calculateExponentialBackoff(attempt-1, baseDelay)
	
	// baseDelay와 prevDelay*3 사이의 랜덤 값
	minDelay := baseDelay
	maxDelay := time.Duration(float64(prevDelay) * 3)
	
	if maxDelay < minDelay {
		maxDelay = minDelay
	}
	
	range_ := maxDelay - minDelay
	if range_ <= 0 {
		return minDelay
	}
	
	randomDelay := time.Duration(rand.Int63n(int64(range_)))
	return minDelay + randomDelay
}

func (c *SmartBackoffCalculator) applyFullJitter(delay time.Duration, factor float64) time.Duration {
	// 0부터 delay*factor까지의 랜덤 값
	maxJitter := time.Duration(float64(delay) * factor)
	if maxJitter <= 0 {
		return delay
	}
	
	jitter := time.Duration(rand.Int63n(int64(maxJitter)))
	return delay + jitter
}

func (c *SmartBackoffCalculator) applyEqualJitter(delay time.Duration, factor float64) time.Duration {
	// delay/2 + random(0, delay*factor/2)
	halfDelay := delay / 2
	maxJitter := time.Duration(float64(delay) * factor / 2)
	
	if maxJitter <= 0 {
		return halfDelay
	}
	
	jitter := time.Duration(rand.Int63n(int64(maxJitter)))
	return halfDelay + jitter
}

func (c *SmartBackoffCalculator) applyDecorrelatedJitter(delay time.Duration, factor float64) time.Duration {
	// AWS decorrelated jitter와 유사하지만 더 단순화
	baseJitter := time.Duration(float64(delay) * factor * 0.5)
	randomJitter := time.Duration(rand.Int63n(int64(baseJitter) + 1))
	
	// ±randomJitter
	if rand.Float64() < 0.5 {
		return delay - randomJitter
	}
	return delay + randomJitter
}

func (c *SmartBackoffCalculator) applyExponentialJitter(delay time.Duration, factor float64) time.Duration {
	// 지수적으로 증가하는 지터
	exponentialFactor := math.Pow(2, factor) - 1
	maxJitter := time.Duration(float64(delay) * exponentialFactor)
	
	if maxJitter <= 0 {
		return delay
	}
	
	jitter := time.Duration(rand.Int63n(int64(maxJitter)))
	return delay + jitter
}

func (c *SmartBackoffCalculator) classifyErrorSimple(err error) ErrorType {
	// 간단한 에러 분류 (실제로는 ErrorClassifier 사용)
	errorMsg := err.Error()
	
	if containsAny(errorMsg, []string{"network", "connection", "tcp", "dns"}) {
		return NetworkError
	}
	if containsAny(errorMsg, []string{"timeout", "deadline"}) {
		return TimeoutError
	}
	if containsAny(errorMsg, []string{"process", "exit"}) {
		return ProcessError
	}
	if containsAny(errorMsg, []string{"memory", "resource"}) {
		return ResourceError
	}
	if containsAny(errorMsg, []string{"auth", "unauthorized"}) {
		return AuthError
	}
	if containsAny(errorMsg, []string{"quota", "limit"}) {
		return QuotaError
	}
	
	return UnknownError
}

func (c *SmartBackoffCalculator) recordDelay(delay time.Duration) {
	c.recentDelays = append(c.recentDelays, delay)
	
	// 최대 샘플 수 제한
	if len(c.recentDelays) > c.maxSamples {
		c.recentDelays = c.recentDelays[1:]
	}
}

func (c *SmartBackoffCalculator) getAverageRecentDelay() time.Duration {
	if len(c.recentDelays) == 0 {
		return 0
	}
	
	var total time.Duration
	for _, delay := range c.recentDelays {
		total += delay
	}
	
	return total / time.Duration(len(c.recentDelays))
}

// 유틸리티 함수들

func containsAny(text string, patterns []string) bool {
	lowerText := strings.ToLower(text)
	for _, pattern := range patterns {
		if strings.Contains(lowerText, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}

// Advanced Backoff Strategies

// CongestionControlBackoff는 네트워크 혼잡 제어를 모방한 백오프입니다
type CongestionControlBackoff struct {
	windowSize    int
	slowStart     bool
	threshold     time.Duration
	currentWindow time.Duration
}

// NewCongestionControlBackoff는 혼잡 제어 백오프를 생성합니다
func NewCongestionControlBackoff(initialWindow time.Duration) *CongestionControlBackoff {
	return &CongestionControlBackoff{
		windowSize:    1,
		slowStart:     true,
		threshold:     initialWindow * 8,
		currentWindow: initialWindow,
	}
}

// Calculate는 혼잡 제어 알고리즘을 사용한 백오프를 계산합니다
func (c *CongestionControlBackoff) Calculate(success bool) time.Duration {
	if success {
		// 성공 시 윈도우 증가
		if c.slowStart {
			// Slow start: exponential increase
			c.currentWindow *= 2
			if c.currentWindow >= c.threshold {
				c.slowStart = false
			}
		} else {
			// Congestion avoidance: linear increase
			c.currentWindow += time.Millisecond * 100
		}
	} else {
		// 실패 시 윈도우 감소
		c.threshold = c.currentWindow / 2
		c.currentWindow = c.threshold
		c.slowStart = true
	}
	
	// 최소/최대 제한
	if c.currentWindow < time.Millisecond * 100 {
		c.currentWindow = time.Millisecond * 100
	}
	if c.currentWindow > time.Minute * 5 {
		c.currentWindow = time.Minute * 5
	}
	
	return c.currentWindow
}

// CircuitBreakerAwareBackoff는 Circuit Breaker 상태를 고려한 백오프입니다
type CircuitBreakerAwareBackoff struct {
	calculator     BackoffCalculator
	circuitBreaker CircuitBreakerState
}

// CircuitBreakerState는 Circuit Breaker 상태입니다
type CircuitBreakerState int

const (
	CircuitClosed CircuitBreakerState = iota
	CircuitHalfOpen
	CircuitOpen
)

// NewCircuitBreakerAwareBackoff는 Circuit Breaker 인식 백오프를 생성합니다
func NewCircuitBreakerAwareBackoff(calculator BackoffCalculator) *CircuitBreakerAwareBackoff {
	return &CircuitBreakerAwareBackoff{
		calculator:     calculator,
		circuitBreaker: CircuitClosed,
	}
}

// Calculate는 Circuit Breaker 상태를 고려한 백오프를 계산합니다
func (c *CircuitBreakerAwareBackoff) Calculate(
	attempt int, 
	baseDelay, maxDelay time.Duration, 
	strategy BackoffType, 
	err error,
) time.Duration {
	baseBackoff := c.calculator.Calculate(attempt, baseDelay, maxDelay, strategy, err)
	
	switch c.circuitBreaker {
	case CircuitClosed:
		return baseBackoff
	case CircuitHalfOpen:
		// Half-open 상태에서는 더 보수적
		return time.Duration(float64(baseBackoff) * 1.5)
	case CircuitOpen:
		// Open 상태에서는 훨씬 더 긴 대기
		return time.Duration(float64(baseBackoff) * 3.0)
	default:
		return baseBackoff
	}
}

// UpdateCircuitState는 Circuit Breaker 상태를 업데이트합니다
func (c *CircuitBreakerAwareBackoff) UpdateCircuitState(state CircuitBreakerState) {
	c.circuitBreaker = state
}