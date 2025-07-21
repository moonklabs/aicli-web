package claude

import (
	"math/rand"
	"time"
)

// BackoffStrategy 백오프 전략 인터페이스
type BackoffStrategy interface {
	// NextBackoff 다음 백오프 시간을 반환합니다
	NextBackoff() time.Duration
	// Reset 백오프를 초기화합니다
	Reset()
	// GetAttempts 현재 시도 횟수를 반환합니다
	GetAttempts() int
}

// ExponentialBackoff 지수 백오프 구현
type ExponentialBackoff struct {
	initial    time.Duration // 초기 백오프 시간
	multiplier float64       // 배수
	max        time.Duration // 최대 백오프 시간
	current    time.Duration // 현재 백오프 시간
	attempts   int           // 시도 횟수
}

// NewExponentialBackoff 새로운 지수 백오프를 생성합니다
func NewExponentialBackoff(policy *RecoveryPolicy) BackoffStrategy {
	return &ExponentialBackoff{
		initial:    policy.RestartInterval,
		multiplier: policy.BackoffMultiplier,
		max:        policy.MaxBackoff,
		current:    policy.RestartInterval,
	}
}

// NextBackoff 다음 백오프 시간을 반환합니다
func (eb *ExponentialBackoff) NextBackoff() time.Duration {
	if eb.attempts == 0 {
		eb.attempts++
		return eb.initial
	}
	
	eb.current = time.Duration(float64(eb.current) * eb.multiplier)
	if eb.current > eb.max {
		eb.current = eb.max
	}
	
	eb.attempts++
	return eb.current
}

// Reset 백오프를 초기화합니다
func (eb *ExponentialBackoff) Reset() {
	eb.current = eb.initial
	eb.attempts = 0
}

// GetAttempts 현재 시도 횟수를 반환합니다
func (eb *ExponentialBackoff) GetAttempts() int {
	return eb.attempts
}

// LinearBackoff 선형 백오프 구현
type LinearBackoff struct {
	initial   time.Duration // 초기 백오프 시간
	increment time.Duration // 증가량
	max       time.Duration // 최대 백오프 시간
	current   time.Duration // 현재 백오프 시간
	attempts  int           // 시도 횟수
}

// NewLinearBackoff 새로운 선형 백오프를 생성합니다
func NewLinearBackoff(initial, increment, max time.Duration) BackoffStrategy {
	return &LinearBackoff{
		initial:   initial,
		increment: increment,
		max:       max,
		current:   initial,
	}
}

// NextBackoff 다음 백오프 시간을 반환합니다
func (lb *LinearBackoff) NextBackoff() time.Duration {
	if lb.attempts == 0 {
		lb.attempts++
		return lb.initial
	}
	
	lb.current += lb.increment
	if lb.current > lb.max {
		lb.current = lb.max
	}
	
	lb.attempts++
	return lb.current
}

// Reset 백오프를 초기화합니다
func (lb *LinearBackoff) Reset() {
	lb.current = lb.initial
	lb.attempts = 0
}

// GetAttempts 현재 시도 횟수를 반환합니다
func (lb *LinearBackoff) GetAttempts() int {
	return lb.attempts
}

// FixedBackoff 고정 백오프 구현
type FixedBackoff struct {
	interval time.Duration // 고정 간격
	attempts int           // 시도 횟수
}

// NewFixedBackoff 새로운 고정 백오프를 생성합니다
func NewFixedBackoff(interval time.Duration) BackoffStrategy {
	return &FixedBackoff{
		interval: interval,
	}
}

// NextBackoff 다음 백오프 시간을 반환합니다
func (fb *FixedBackoff) NextBackoff() time.Duration {
	fb.attempts++
	return fb.interval
}

// Reset 백오프를 초기화합니다
func (fb *FixedBackoff) Reset() {
	fb.attempts = 0
}

// GetAttempts 현재 시도 횟수를 반환합니다
func (fb *FixedBackoff) GetAttempts() int {
	return fb.attempts
}

// JitteredExponentialBackoff 지터가 적용된 지수 백오프 구현
type JitteredExponentialBackoff struct {
	*ExponentialBackoff
	jitterFactor float64 // 지터 팩터 (0.0 ~ 1.0)
}

// NewJitteredExponentialBackoff 새로운 지터 지수 백오프를 생성합니다
func NewJitteredExponentialBackoff(policy *RecoveryPolicy, jitterFactor float64) BackoffStrategy {
	if jitterFactor < 0 {
		jitterFactor = 0
	}
	if jitterFactor > 1 {
		jitterFactor = 1
	}
	
	return &JitteredExponentialBackoff{
		ExponentialBackoff: &ExponentialBackoff{
			initial:    policy.RestartInterval,
			multiplier: policy.BackoffMultiplier,
			max:        policy.MaxBackoff,
			current:    policy.RestartInterval,
		},
		jitterFactor: jitterFactor,
	}
}

// NextBackoff 다음 백오프 시간을 반환합니다 (지터 적용)
func (jeb *JitteredExponentialBackoff) NextBackoff() time.Duration {
	baseBackoff := jeb.ExponentialBackoff.NextBackoff()
	
	if jeb.jitterFactor == 0 {
		return baseBackoff
	}
	
	// 지터 적용: ±jitterFactor 범위에서 랜덤 조정
	jitter := jeb.jitterFactor * float64(baseBackoff)
	randomFactor := (2*rand.Float64()) - 1 // -1 ~ 1 범위
	adjustedBackoff := float64(baseBackoff) + (jitter * randomFactor)
	
	if adjustedBackoff < 0 {
		adjustedBackoff = float64(baseBackoff) * 0.1 // 최소값 보장
	}
	
	return time.Duration(adjustedBackoff)
}

// BackoffFactory 백오프 전략 팩토리
type BackoffFactory struct{}

// NewBackoffFactory 새로운 백오프 팩토리를 생성합니다
func NewBackoffFactory() *BackoffFactory {
	return &BackoffFactory{}
}

// CreateBackoff 백오프 타입에 따른 백오프 전략을 생성합니다
func (bf *BackoffFactory) CreateBackoff(backoffType BackoffType, policy *RecoveryPolicy) BackoffStrategy {
	switch backoffType {
	case BackoffFixed:
		return NewFixedBackoff(policy.RestartInterval)
	case BackoffLinear:
		return NewLinearBackoff(
			policy.RestartInterval,
			policy.RestartInterval/2, // 증가량은 초기값의 절반
			policy.MaxBackoff,
		)
	case BackoffExponential:
		return NewExponentialBackoff(policy)
	default:
		// 기본값은 지수 백오프
		return NewExponentialBackoff(policy)
	}
}

// CreateJitteredBackoff 지터가 적용된 백오프 전략을 생성합니다
func (bf *BackoffFactory) CreateJitteredBackoff(policy *RecoveryPolicy, jitterFactor float64) BackoffStrategy {
	return NewJitteredExponentialBackoff(policy, jitterFactor)
}