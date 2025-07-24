package claude

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

// IntelligentRetrier는 지능형 재시도 인터페이스입니다
type IntelligentRetrier interface {
	// 적응형 백오프 재시도
	RetryWithBackoff(ctx context.Context, operation Operation) error
	
	// 재시도 정책 설정
	SetRetryPolicy(errorType ErrorType, policy RetryPolicy) error
	
	// 재시도 통계 조회
	GetRetryStats() RetryStatistics
	
	// 동적 정책 조정
	AdjustPolicy(errorType ErrorType, successRate float64) error
	
	// 재시도 중지
	Stop()
}

// Operation은 재시도 가능한 작업을 정의합니다
type Operation func(ctx context.Context, attempt int) error

// RetryPolicy는 재시도 정책입니다
type RetryPolicy struct {
	MaxAttempts     int           `json:"max_attempts"`
	BaseDelay       time.Duration `json:"base_delay"`
	MaxDelay        time.Duration `json:"max_delay"`
	BackoffStrategy BackoffType   `json:"backoff_strategy"`
	Jitter          bool          `json:"jitter"`
	JitterFactor    float64       `json:"jitter_factor"`
	RetryableErrors []ErrorType   `json:"retryable_errors"`
	CircuitBreaker  bool          `json:"circuit_breaker"`
	Timeout         time.Duration `json:"timeout"`
	
	// 적응형 설정
	AdaptiveEnabled   bool    `json:"adaptive_enabled"`
	SuccessThreshold  float64 `json:"success_threshold"`
	FailureThreshold  float64 `json:"failure_threshold"`
	AdjustmentFactor  float64 `json:"adjustment_factor"`
}

// BackoffType은 백오프 전략 유형입니다
type BackoffType int

const (
	LinearBackoffType BackoffType = iota
	ExponentialBackoffType
	FixedDelayBackoffType
	AdaptiveBackoffType
	DecorrelatedJitterBackoffType
)

// RetryStatistics는 재시도 통계입니다
type RetryStatistics struct {
	TotalAttempts     int64                    `json:"total_attempts"`
	SuccessfulRetries int64                    `json:"successful_retries"`
	FailedRetries     int64                    `json:"failed_retries"`
	AttemptsByType    map[ErrorType]int64      `json:"attempts_by_type"`
	SuccessByType     map[ErrorType]int64      `json:"success_by_type"`
	AvgAttempts       float64                  `json:"avg_attempts"`
	AvgDelay          time.Duration            `json:"avg_delay"`
	TotalDelay        time.Duration            `json:"total_delay"`
	SuccessRates      map[ErrorType]float64    `json:"success_rates"`
	RecentAttempts    []RetryAttempt           `json:"recent_attempts"`
	StartTime         time.Time                `json:"start_time"`
	LastUpdated       time.Time                `json:"last_updated"`
}

// RetryAttempt는 재시도 시도 정보입니다
type RetryAttempt struct {
	ID           string        `json:"id"`
	ErrorType    ErrorType     `json:"error_type"`
	AttemptCount int           `json:"attempt_count"`
	TotalDelay   time.Duration `json:"total_delay"`
	Success      bool          `json:"success"`
	Error        string        `json:"error,omitempty"`
	Timestamp    time.Time     `json:"timestamp"`
	Context      map[string]interface{} `json:"context"`
}

// AdaptiveRetrier는 적응형 재시도 구현체입니다
type AdaptiveRetrier struct {
	// 정책 관리
	policies     map[ErrorType]RetryPolicy
	policiesMutex sync.RWMutex
	
	// 통계
	statistics   *RetryStatistics
	statsMutex   sync.RWMutex
	
	// 에러 분류기
	classifier   ErrorClassifier
	
	// Circuit Breaker 통합
	circuitBreaker AdvancedCircuitBreaker
	
	// 백오프 계산기
	backoffCalculator BackoffCalculator
	
	// 적응형 조정
	adaptiveState map[ErrorType]*AdaptiveState
	adaptiveMutex sync.RWMutex
	
	// 생명주기
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	
	// 설정
	config AdaptiveRetrierConfig
}

// AdaptiveState는 적응형 상태입니다
type AdaptiveState struct {
	SuccessCount    int64     `json:"success_count"`
	FailureCount    int64     `json:"failure_count"`
	RecentSuccesses []bool    `json:"recent_successes"`
	LastAdjustment  time.Time `json:"last_adjustment"`
	CurrentPolicy   RetryPolicy `json:"current_policy"`
	Trend          float64   `json:"trend"`
}

// AdaptiveRetrierConfig는 적응형 재시도 설정입니다
type AdaptiveRetrierConfig struct {
	DefaultMaxAttempts   int           `json:"default_max_attempts"`
	DefaultBaseDelay     time.Duration `json:"default_base_delay"`
	DefaultMaxDelay      time.Duration `json:"default_max_delay"`
	DefaultJitterFactor  float64       `json:"default_jitter_factor"`
	AdaptiveWindow       int           `json:"adaptive_window"`
	AdaptiveInterval     time.Duration `json:"adaptive_interval"`
	MinAdjustmentFactor  float64       `json:"min_adjustment_factor"`
	MaxAdjustmentFactor  float64       `json:"max_adjustment_factor"`
	StatsRetentionPeriod time.Duration `json:"stats_retention_period"`
}

// NewAdaptiveRetrier는 새로운 적응형 재시도기를 생성합니다
func NewAdaptiveRetrier(classifier ErrorClassifier, config AdaptiveRetrierConfig) *AdaptiveRetrier {
	ctx, cancel := context.WithCancel(context.Background())
	
	retrier := &AdaptiveRetrier{
		policies:      make(map[ErrorType]RetryPolicy),
		classifier:    classifier,
		adaptiveState: make(map[ErrorType]*AdaptiveState),
		ctx:           ctx,
		cancel:        cancel,
		config:        config,
		statistics: &RetryStatistics{
			AttemptsByType: make(map[ErrorType]int64),
			SuccessByType:  make(map[ErrorType]int64),
			SuccessRates:   make(map[ErrorType]float64),
			RecentAttempts: make([]RetryAttempt, 0, 100),
			StartTime:      time.Now(),
		},
		backoffCalculator: NewSmartBackoffCalculator(),
	}
	
	// 기본 정책 초기화
	retrier.initializeDefaultPolicies()
	
	// 적응형 조정 고루틴 시작
	retrier.wg.Add(1)
	go retrier.adaptiveTuningLoop()
	
	return retrier
}

// RetryWithBackoff는 적응형 백오프로 재시도합니다
func (r *AdaptiveRetrier) RetryWithBackoff(ctx context.Context, operation Operation) error {
	attemptID := fmt.Sprintf("attempt_%d", time.Now().UnixNano())
	
	var lastErr error
	startTime := time.Now()
	
	// 첫 번째 시도
	lastErr = operation(ctx, 1)
	if lastErr == nil {
		r.recordSuccess(attemptID, UnknownError, 1, time.Since(startTime))
		return nil
	}
	
	// 에러 분류
	errorClass := r.classifier.ClassifyError(lastErr)
	if !r.classifier.IsRetryable(lastErr) {
		r.recordFailure(attemptID, errorClass.Type, 1, time.Since(startTime), lastErr)
		return lastErr
	}
	
	// 재시도 정책 조회
	policy := r.getRetryPolicy(errorClass.Type)
	
	// Circuit Breaker 확인
	if policy.CircuitBreaker && r.circuitBreaker != nil {
		if r.circuitBreaker.GetState() == CircuitOpen {
			return fmt.Errorf("circuit breaker is open: %w", lastErr)
		}
	}
	
	// 재시도 루프
	for attempt := 2; attempt <= policy.MaxAttempts; attempt++ {
		// 컨텍스트 취소 확인
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		
		// 백오프 지연 계산
		delay := r.backoffCalculator.Calculate(
			attempt-1, 
			policy.BaseDelay, 
			policy.MaxDelay, 
			policy.BackoffStrategy,
			lastErr,
		)
		
		// Jitter 적용
		if policy.Jitter {
			delay = r.applyJitter(delay, policy.JitterFactor)
		}
		
		// 지연 실행
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
		
		// 재시도 실행
		lastErr = operation(ctx, attempt)
		if lastErr == nil {
			r.recordSuccess(attemptID, errorClass.Type, attempt, time.Since(startTime))
			return nil
		}
		
		// 새로운 에러 분류 (에러가 변경될 수 있음)
		newErrorClass := r.classifier.ClassifyError(lastErr)
		if !r.classifier.IsRetryable(lastErr) {
			r.recordFailure(attemptID, newErrorClass.Type, attempt, time.Since(startTime), lastErr)
			return lastErr
		}
		
		// Circuit Breaker 업데이트
		if policy.CircuitBreaker && r.circuitBreaker != nil {
			r.circuitBreaker.HandlePartialFailure(0, 1)
		}
	}
	
	// 모든 재시도 실패
	r.recordFailure(attemptID, errorClass.Type, policy.MaxAttempts, time.Since(startTime), lastErr)
	return fmt.Errorf("retry failed after %d attempts: %w", policy.MaxAttempts, lastErr)
}

// SetRetryPolicy는 재시도 정책을 설정합니다
func (r *AdaptiveRetrier) SetRetryPolicy(errorType ErrorType, policy RetryPolicy) error {
	if policy.MaxAttempts <= 0 {
		return fmt.Errorf("max attempts must be positive")
	}
	
	if policy.BaseDelay <= 0 {
		return fmt.Errorf("base delay must be positive")
	}
	
	if policy.MaxDelay < policy.BaseDelay {
		return fmt.Errorf("max delay must be greater than or equal to base delay")
	}
	
	r.policiesMutex.Lock()
	r.policies[errorType] = policy
	r.policiesMutex.Unlock()
	
	// 적응형 상태 초기화
	if policy.AdaptiveEnabled {
		r.adaptiveMutex.Lock()
		r.adaptiveState[errorType] = &AdaptiveState{
			RecentSuccesses: make([]bool, 0, r.config.AdaptiveWindow),
			LastAdjustment:  time.Now(),
			CurrentPolicy:   policy,
		}
		r.adaptiveMutex.Unlock()
	}
	
	return nil
}

// GetRetryStats는 재시도 통계를 조회합니다
func (r *AdaptiveRetrier) GetRetryStats() RetryStatistics {
	r.statsMutex.RLock()
	defer r.statsMutex.RUnlock()
	
	// 성공률 계산
	stats := *r.statistics
	stats.SuccessRates = make(map[ErrorType]float64)
	
	for errorType, attempts := range r.statistics.AttemptsByType {
		if attempts > 0 {
			successes := r.statistics.SuccessByType[errorType]
			stats.SuccessRates[errorType] = float64(successes) / float64(attempts)
		}
	}
	
	// 평균 계산
	if r.statistics.TotalAttempts > 0 {
		stats.AvgAttempts = float64(r.statistics.SuccessfulRetries+r.statistics.FailedRetries) / float64(r.statistics.TotalAttempts)
	}
	
	return stats
}

// AdjustPolicy는 동적으로 정책을 조정합니다
func (r *AdaptiveRetrier) AdjustPolicy(errorType ErrorType, successRate float64) error {
	r.policiesMutex.Lock()
	policy, exists := r.policies[errorType]
	if !exists {
		r.policiesMutex.Unlock()
		return fmt.Errorf("policy not found for error type: %v", errorType)
	}
	r.policiesMutex.Unlock()
	
	if !policy.AdaptiveEnabled {
		return fmt.Errorf("adaptive adjustment is not enabled for error type: %v", errorType)
	}
	
	// 정책 조정 로직
	adjustmentFactor := 1.0
	
	if successRate < policy.FailureThreshold {
		// 성공률이 낮으면 더 보수적으로 (지연 증가, 시도 횟수 감소)
		adjustmentFactor = 1.0 + policy.AdjustmentFactor
	} else if successRate > policy.SuccessThreshold {
		// 성공률이 높으면 더 적극적으로 (지연 감소, 시도 횟수 증가)
		adjustmentFactor = 1.0 - policy.AdjustmentFactor
	}
	
	// 조정 제한 적용
	if adjustmentFactor < r.config.MinAdjustmentFactor {
		adjustmentFactor = r.config.MinAdjustmentFactor
	} else if adjustmentFactor > r.config.MaxAdjustmentFactor {
		adjustmentFactor = r.config.MaxAdjustmentFactor
	}
	
	// 새 정책 적용
	newPolicy := policy
	newPolicy.BaseDelay = time.Duration(float64(policy.BaseDelay) * adjustmentFactor)
	newPolicy.MaxDelay = time.Duration(float64(policy.MaxDelay) * adjustmentFactor)
	
	if adjustmentFactor < 1.0 {
		// 성공률이 높으면 시도 횟수 늘림
		if newPolicy.MaxAttempts < 10 {
			newPolicy.MaxAttempts++
		}
	} else if adjustmentFactor > 1.0 {
		// 성공률이 낮으면 시도 횟수 줄임
		if newPolicy.MaxAttempts > 1 {
			newPolicy.MaxAttempts--
		}
	}
	
	r.policiesMutex.Lock()
	r.policies[errorType] = newPolicy
	r.policiesMutex.Unlock()
	
	// 적응형 상태 업데이트
	r.adaptiveMutex.Lock()
	if state, exists := r.adaptiveState[errorType]; exists {
		state.CurrentPolicy = newPolicy
		state.LastAdjustment = time.Now()
		state.Trend = adjustmentFactor - 1.0
	}
	r.adaptiveMutex.Unlock()
	
	return nil
}

// Stop은 재시도기를 중지합니다
func (r *AdaptiveRetrier) Stop() {
	r.cancel()
	r.wg.Wait()
}

// 내부 메서드들

func (r *AdaptiveRetrier) initializeDefaultPolicies() {
	defaultPolicies := map[ErrorType]RetryPolicy{
		NetworkError: {
			MaxAttempts:     5,
			BaseDelay:       1 * time.Second,
			MaxDelay:        30 * time.Second,
			BackoffStrategy: ExponentialBackoffType,
			Jitter:          true,
			JitterFactor:    0.1,
			RetryableErrors: []ErrorType{NetworkError},
			CircuitBreaker:  true,
			Timeout:         5 * time.Minute,
			AdaptiveEnabled: true,
			SuccessThreshold: 0.8,
			FailureThreshold: 0.3,
			AdjustmentFactor: 0.2,
		},
		TimeoutError: {
			MaxAttempts:     3,
			BaseDelay:       2 * time.Second,
			MaxDelay:        60 * time.Second,
			BackoffStrategy: LinearBackoffType,
			Jitter:          true,
			JitterFactor:    0.2,
			RetryableErrors: []ErrorType{TimeoutError},
			CircuitBreaker:  false,
			Timeout:         10 * time.Minute,
			AdaptiveEnabled: true,
			SuccessThreshold: 0.7,
			FailureThreshold: 0.4,
			AdjustmentFactor: 0.3,
		},
		ProcessError: {
			MaxAttempts:     3,
			BaseDelay:       5 * time.Second,
			MaxDelay:        120 * time.Second,
			BackoffStrategy: ExponentialBackoffType,
			Jitter:          false,
			RetryableErrors: []ErrorType{ProcessError},
			CircuitBreaker:  true,
			Timeout:         15 * time.Minute,
			AdaptiveEnabled: false, // 프로세스 에러는 수동 조정
		},
		ResourceError: {
			MaxAttempts:     4,
			BaseDelay:       10 * time.Second,
			MaxDelay:        300 * time.Second,
			BackoffStrategy: ExponentialBackoffType,
			Jitter:          true,
			JitterFactor:    0.3,
			RetryableErrors: []ErrorType{ResourceError},
			CircuitBreaker:  true,
			Timeout:         20 * time.Minute,
			AdaptiveEnabled: true,
			SuccessThreshold: 0.6,
			FailureThreshold: 0.2,
			AdjustmentFactor: 0.4,
		},
		QuotaError: {
			MaxAttempts:     2,
			BaseDelay:       60 * time.Second,
			MaxDelay:        600 * time.Second,
			BackoffStrategy: FixedDelayBackoffType,
			Jitter:          false,
			RetryableErrors: []ErrorType{QuotaError},
			CircuitBreaker:  false,
			Timeout:         30 * time.Minute,
			AdaptiveEnabled: false, // 할당량 에러는 고정 지연
		},
	}
	
	for errorType, policy := range defaultPolicies {
		r.SetRetryPolicy(errorType, policy)
	}
}

func (r *AdaptiveRetrier) getRetryPolicy(errorType ErrorType) RetryPolicy {
	r.policiesMutex.RLock()
	defer r.policiesMutex.RUnlock()
	
	if policy, exists := r.policies[errorType]; exists {
		return policy
	}
	
	// 기본 정책 반환
	return RetryPolicy{
		MaxAttempts:     r.config.DefaultMaxAttempts,
		BaseDelay:       r.config.DefaultBaseDelay,
		MaxDelay:        r.config.DefaultMaxDelay,
		BackoffStrategy: ExponentialBackoffType,
		Jitter:          true,
		JitterFactor:    r.config.DefaultJitterFactor,
		Timeout:         5 * time.Minute,
	}
}

func (r *AdaptiveRetrier) applyJitter(delay time.Duration, jitterFactor float64) time.Duration {
	if jitterFactor <= 0 {
		return delay
	}
	
	// ±jitterFactor 범위의 무작위 조정
	jitter := 1.0 + (rand.Float64()-0.5)*2*jitterFactor
	return time.Duration(float64(delay) * jitter)
}

func (r *AdaptiveRetrier) recordSuccess(attemptID string, errorType ErrorType, attempts int, duration time.Duration) {
	r.statsMutex.Lock()
	defer r.statsMutex.Unlock()
	
	r.statistics.TotalAttempts++
	r.statistics.SuccessfulRetries++
	r.statistics.AttemptsByType[errorType]++
	r.statistics.SuccessByType[errorType]++
	r.statistics.TotalDelay += duration
	r.statistics.LastUpdated = time.Now()
	
	// 최근 시도에 추가
	attempt := RetryAttempt{
		ID:           attemptID,
		ErrorType:    errorType,
		AttemptCount: attempts,
		TotalDelay:   duration,
		Success:      true,
		Timestamp:    time.Now(),
		Context:      make(map[string]interface{}),
	}
	
	r.statistics.RecentAttempts = append(r.statistics.RecentAttempts, attempt)
	if len(r.statistics.RecentAttempts) > 100 {
		r.statistics.RecentAttempts = r.statistics.RecentAttempts[1:]
	}
	
	// 적응형 상태 업데이트
	r.updateAdaptiveState(errorType, true)
}

func (r *AdaptiveRetrier) recordFailure(attemptID string, errorType ErrorType, attempts int, duration time.Duration, err error) {
	r.statsMutex.Lock()
	defer r.statsMutex.Unlock()
	
	r.statistics.TotalAttempts++
	r.statistics.FailedRetries++
	r.statistics.AttemptsByType[errorType]++
	r.statistics.TotalDelay += duration
	r.statistics.LastUpdated = time.Now()
	
	// 최근 시도에 추가
	attempt := RetryAttempt{
		ID:           attemptID,
		ErrorType:    errorType,
		AttemptCount: attempts,
		TotalDelay:   duration,
		Success:      false,
		Error:        err.Error(),
		Timestamp:    time.Now(),
		Context:      make(map[string]interface{}),
	}
	
	r.statistics.RecentAttempts = append(r.statistics.RecentAttempts, attempt)
	if len(r.statistics.RecentAttempts) > 100 {
		r.statistics.RecentAttempts = r.statistics.RecentAttempts[1:]
	}
	
	// 적응형 상태 업데이트
	r.updateAdaptiveState(errorType, false)
}

func (r *AdaptiveRetrier) updateAdaptiveState(errorType ErrorType, success bool) {
	r.adaptiveMutex.Lock()
	defer r.adaptiveMutex.Unlock()
	
	state, exists := r.adaptiveState[errorType]
	if !exists {
		return
	}
	
	if success {
		atomic.AddInt64(&state.SuccessCount, 1)
	} else {
		atomic.AddInt64(&state.FailureCount, 1)
	}
	
	// 최근 성공/실패 기록 업데이트
	state.RecentSuccesses = append(state.RecentSuccesses, success)
	if len(state.RecentSuccesses) > r.config.AdaptiveWindow {
		state.RecentSuccesses = state.RecentSuccesses[1:]
	}
}

func (r *AdaptiveRetrier) adaptiveTuningLoop() {
	defer r.wg.Done()
	
	ticker := time.NewTicker(r.config.AdaptiveInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-r.ctx.Done():
			return
		case <-ticker.C:
			r.performAdaptiveTuning()
		}
	}
}

func (r *AdaptiveRetrier) performAdaptiveTuning() {
	r.adaptiveMutex.RLock()
	states := make(map[ErrorType]*AdaptiveState)
	for errorType, state := range r.adaptiveState {
		// 상태 복사
		stateCopy := *state
		stateCopy.RecentSuccesses = make([]bool, len(state.RecentSuccesses))
		copy(stateCopy.RecentSuccesses, state.RecentSuccesses)
		states[errorType] = &stateCopy
	}
	r.adaptiveMutex.RUnlock()
	
	for errorType, state := range states {
		if len(state.RecentSuccesses) < r.config.AdaptiveWindow/2 {
			continue // 충분한 데이터가 없음
		}
		
		// 성공률 계산
		successCount := 0
		for _, success := range state.RecentSuccesses {
			if success {
				successCount++
			}
		}
		
		successRate := float64(successCount) / float64(len(state.RecentSuccesses))
		
		// 마지막 조정 이후 충분한 시간이 지났는지 확인
		if time.Since(state.LastAdjustment) < r.config.AdaptiveInterval*2 {
			continue
		}
		
		// 정책 조정
		r.AdjustPolicy(errorType, successRate)
	}
}