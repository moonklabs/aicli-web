package flow

import (
	"context"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"
	
	"golang.org/x/time/rate"
)

// DynamicThrottler 동적 스로틀링 관리자
type DynamicThrottler struct {
	throttleRates map[string]*ThrottleState
	limiters      map[string]*rate.Limiter
	config        *ThrottleConfig
	metrics       *ThrottleMetrics
	mutex         sync.RWMutex
	
	// 적응형 모드 관련
	adaptiveController *AdaptiveController
}

// ThrottleState 스로틀 상태
type ThrottleState struct {
	ConnectionID    string
	CurrentRate     float64 // messages/sec
	OriginalRate    float64
	LastAdjustment  time.Time
	Reason          ThrottleReason
	Active          bool
	
	// 히스토리
	RateHistory     []float64
	PerformanceHist []float64
	
	// 통계
	AdjustmentCount uint32
	ThrottledCount  uint64
}

// ThrottleReason 스로틀 이유
type ThrottleReason int

const (
	ThrottleBackpressure ThrottleReason = iota
	ThrottleSystemLoad
	ThrottleNetworkQuality
	ThrottleClientCapacity
	ThrottleManual
)

// ThrottleMetrics 스로틀링 메트릭
type ThrottleMetrics struct {
	ActiveThrottles   int32
	TotalAdjustments  uint64
	AverageRate       float64
	MinRate           float64
	MaxRate           float64
	LastUpdate        time.Time
}

// AdaptiveController 적응형 제어기
type AdaptiveController struct {
	learningRate     float64
	momentum         float64
	weights          map[string]float64
	previousGradient map[string]float64
	mutex            sync.RWMutex
}

// NewDynamicThrottler 새 동적 스로틀러 생성
func NewDynamicThrottler(config *ThrottleConfig) *DynamicThrottler {
	dt := &DynamicThrottler{
		throttleRates: make(map[string]*ThrottleState),
		limiters:      make(map[string]*rate.Limiter),
		config:        config,
		metrics: &ThrottleMetrics{
			LastUpdate: time.Now(),
		},
	}
	
	if config.AdaptiveMode {
		dt.adaptiveController = NewAdaptiveController()
	}
	
	return dt
}

// NewAdaptiveController 적응형 컨트롤러 생성
func NewAdaptiveController() *AdaptiveController {
	return &AdaptiveController{
		learningRate:     0.01,
		momentum:         0.9,
		weights:          make(map[string]float64),
		previousGradient: make(map[string]float64),
	}
}

// ApplyThrottle 스로틀링 적용
func (dt *DynamicThrottler) ApplyThrottle(connectionID string, level BackpressureLevel) error {
	dt.mutex.Lock()
	defer dt.mutex.Unlock()
	
	throttleState, exists := dt.throttleRates[connectionID]
	if !exists {
		throttleState = &ThrottleState{
			ConnectionID:    connectionID,
			CurrentRate:     dt.config.MaxRate,
			OriginalRate:    dt.config.MaxRate,
			RateHistory:     make([]float64, 0, 100),
			PerformanceHist: make([]float64, 0, 100),
		}
		dt.throttleRates[connectionID] = throttleState
		
		// Rate limiter 초기화
		dt.limiters[connectionID] = rate.NewLimiter(rate.Limit(dt.config.MaxRate), int(dt.config.MaxRate))
	}
	
	now := time.Now()
	
	// 조정 간격 확인
	if now.Sub(throttleState.LastAdjustment) < dt.config.AdjustmentInterval {
		return nil
	}
	
	var newRate float64
	
	if dt.config.AdaptiveMode && dt.adaptiveController != nil {
		// 적응형 모드: ML 기반 rate 조정
		newRate = dt.adaptiveAdjustment(connectionID, throttleState, level)
	} else {
		// 일반 모드: 규칙 기반 조정
		newRate = dt.ruleBasedAdjustment(throttleState, level)
	}
	
	// Rate 범위 제한
	newRate = math.Max(dt.config.MinRate, math.Min(dt.config.MaxRate, newRate))
	
	// Rate 변경 적용
	if newRate != throttleState.CurrentRate {
		throttleState.CurrentRate = newRate
		throttleState.LastAdjustment = now
		atomic.AddUint32(&throttleState.AdjustmentCount, 1)
		
		// Rate limiter 업데이트
		dt.limiters[connectionID].SetLimit(rate.Limit(newRate))
		dt.limiters[connectionID].SetBurst(int(newRate))
		
		// 메트릭 업데이트
		atomic.AddUint64(&dt.metrics.TotalAdjustments, 1)
		
		if level > BackpressureNone {
			throttleState.Active = true
			throttleState.Reason = ThrottleBackpressure
			atomic.AddInt32(&dt.metrics.ActiveThrottles, 1)
		} else if throttleState.Active {
			throttleState.Active = false
			atomic.AddInt32(&dt.metrics.ActiveThrottles, -1)
		}
		
		// 히스토리 업데이트
		dt.updateHistory(throttleState, newRate)
		
		log.Debugf("Throttle adjusted for %s: %.2f -> %.2f (level: %s)",
			connectionID, throttleState.CurrentRate, newRate, level.String())
	}
	
	return nil
}

// ruleBasedAdjustment 규칙 기반 rate 조정
func (dt *DynamicThrottler) ruleBasedAdjustment(state *ThrottleState, level BackpressureLevel) float64 {
	currentRate := state.CurrentRate
	
	switch level {
	case BackpressureNone:
		// 점진적 복구
		if state.Active {
			newRate := currentRate * (1 + dt.config.RecoveryRate)
			if newRate >= state.OriginalRate {
				return state.OriginalRate
			}
			return newRate
		}
		return currentRate
		
	case BackpressureLow:
		// 약간 감소 (10%)
		return currentRate * (1 - dt.config.AdjustmentFactor*0.1)
		
	case BackpressureMedium:
		// 중간 감소 (30%)
		return currentRate * (1 - dt.config.AdjustmentFactor*0.3)
		
	case BackpressureHigh:
		// 큰 감소 (50%)
		return currentRate * (1 - dt.config.AdjustmentFactor*0.5)
		
	case BackpressureCritical:
		// 최대 감소 (70%)
		return math.Max(dt.config.MinRate, currentRate*(1-dt.config.AdjustmentFactor*0.7))
		
	default:
		return currentRate
	}
}

// adaptiveAdjustment 적응형 rate 조정
func (dt *DynamicThrottler) adaptiveAdjustment(connectionID string, state *ThrottleState, level BackpressureLevel) float64 {
	dt.adaptiveController.mutex.Lock()
	defer dt.adaptiveController.mutex.Unlock()
	
	// 특징 추출
	features := dt.extractFeatures(state, level)
	
	// 예측 rate 계산
	predictedRate := dt.adaptiveController.predict(features)
	
	// 실제 성능 기반 학습
	if len(state.PerformanceHist) > 0 {
		actualPerformance := state.PerformanceHist[len(state.PerformanceHist)-1]
		dt.adaptiveController.learn(features, actualPerformance, predictedRate)
	}
	
	// 안전성 검증
	safeRate := dt.validateSafetyBounds(predictedRate, state.CurrentRate, level)
	
	return safeRate
}

// extractFeatures 특징 추출
func (dt *DynamicThrottler) extractFeatures(state *ThrottleState, level BackpressureLevel) map[string]float64 {
	features := make(map[string]float64)
	
	// 기본 특징
	features["current_rate"] = state.CurrentRate
	features["backpressure_level"] = float64(level)
	features["throttle_active"] = 0
	if state.Active {
		features["throttle_active"] = 1
	}
	
	// 히스토리 기반 특징
	if len(state.RateHistory) > 0 {
		features["avg_rate"] = calculateAverage(state.RateHistory)
		features["rate_variance"] = calculateVariance(state.RateHistory)
		features["rate_trend"] = calculateTrend(state.RateHistory)
	}
	
	// 성능 특징
	if len(state.PerformanceHist) > 0 {
		features["avg_performance"] = calculateAverage(state.PerformanceHist)
		features["performance_stability"] = calculateStability(state.PerformanceHist)
	}
	
	return features
}

// predict 예측 수행
func (ac *AdaptiveController) predict(features map[string]float64) float64 {
	var prediction float64
	
	for feature, value := range features {
		weight, exists := ac.weights[feature]
		if !exists {
			// 새 특징에 대한 초기 가중치
			ac.weights[feature] = 0.1
			weight = 0.1
		}
		prediction += weight * value
	}
	
	// Sigmoid 활성화 함수로 정규화
	return sigmoid(prediction)
}

// learn 학습 수행
func (ac *AdaptiveController) learn(features map[string]float64, actualPerformance, prediction float64) {
	error := actualPerformance - prediction
	
	for feature, value := range features {
		// 그래디언트 계산
		gradient := error * value
		
		// 모멘텀 적용
		prevGradient := ac.previousGradient[feature]
		gradient = gradient*ac.learningRate + prevGradient*ac.momentum
		
		// 가중치 업데이트
		ac.weights[feature] += gradient
		ac.previousGradient[feature] = gradient
	}
}

// validateSafetyBounds 안전 범위 검증
func (dt *DynamicThrottler) validateSafetyBounds(predictedRate, currentRate float64, level BackpressureLevel) float64 {
	// 급격한 변화 방지
	maxChange := currentRate * 0.5 // 최대 50% 변화
	
	if math.Abs(predictedRate-currentRate) > maxChange {
		if predictedRate > currentRate {
			predictedRate = currentRate + maxChange
		} else {
			predictedRate = currentRate - maxChange
		}
	}
	
	// 백프레셔 레벨에 따른 상한 설정
	switch level {
	case BackpressureCritical:
		predictedRate = math.Min(predictedRate, dt.config.MinRate*2)
	case BackpressureHigh:
		predictedRate = math.Min(predictedRate, dt.config.MaxRate*0.3)
	case BackpressureMedium:
		predictedRate = math.Min(predictedRate, dt.config.MaxRate*0.5)
	case BackpressureLow:
		predictedRate = math.Min(predictedRate, dt.config.MaxRate*0.7)
	}
	
	return predictedRate
}

// Wait 전송 대기 (rate limiting 적용)
func (dt *DynamicThrottler) Wait(ctx context.Context, connectionID string) error {
	dt.mutex.RLock()
	limiter, exists := dt.limiters[connectionID]
	dt.mutex.RUnlock()
	
	if !exists {
		// 새 연결인 경우 기본 limiter 생성
		dt.mutex.Lock()
		limiter = rate.NewLimiter(rate.Limit(dt.config.MaxRate), int(dt.config.MaxRate))
		dt.limiters[connectionID] = limiter
		dt.mutex.Unlock()
	}
	
	// Rate limiting 적용
	if err := limiter.Wait(ctx); err != nil {
		return fmt.Errorf("rate limit wait failed: %w", err)
	}
	
	// 스로틀 카운트 증가
	dt.mutex.RLock()
	if state, exists := dt.throttleRates[connectionID]; exists && state.Active {
		atomic.AddUint64(&state.ThrottledCount, 1)
	}
	dt.mutex.RUnlock()
	
	return nil
}

// GetCurrentRate 현재 rate 조회
func (dt *DynamicThrottler) GetCurrentRate(connectionID string) (float64, error) {
	dt.mutex.RLock()
	defer dt.mutex.RUnlock()
	
	state, exists := dt.throttleRates[connectionID]
	if !exists {
		return dt.config.MaxRate, nil
	}
	
	return state.CurrentRate, nil
}

// RemoveConnection 연결 제거
func (dt *DynamicThrottler) RemoveConnection(connectionID string) {
	dt.mutex.Lock()
	defer dt.mutex.Unlock()
	
	if state, exists := dt.throttleRates[connectionID]; exists && state.Active {
		atomic.AddInt32(&dt.metrics.ActiveThrottles, -1)
	}
	
	delete(dt.throttleRates, connectionID)
	delete(dt.limiters, connectionID)
}

// GetMetrics 메트릭 조회
func (dt *DynamicThrottler) GetMetrics() *ThrottleMetrics {
	dt.mutex.RLock()
	defer dt.mutex.RUnlock()
	
	metrics := &ThrottleMetrics{
		ActiveThrottles:  atomic.LoadInt32(&dt.metrics.ActiveThrottles),
		TotalAdjustments: atomic.LoadUint64(&dt.metrics.TotalAdjustments),
		LastUpdate:       time.Now(),
	}
	
	// 평균, 최소, 최대 rate 계산
	var totalRate float64
	minRate := dt.config.MaxRate
	maxRate := dt.config.MinRate
	count := 0
	
	for _, state := range dt.throttleRates {
		rate := state.CurrentRate
		totalRate += rate
		count++
		
		if rate < minRate {
			minRate = rate
		}
		if rate > maxRate {
			maxRate = rate
		}
	}
	
	if count > 0 {
		metrics.AverageRate = totalRate / float64(count)
		metrics.MinRate = minRate
		metrics.MaxRate = maxRate
	}
	
	return metrics
}

// updateHistory 히스토리 업데이트
func (dt *DynamicThrottler) updateHistory(state *ThrottleState, newRate float64) {
	// Rate 히스토리
	state.RateHistory = append(state.RateHistory, newRate)
	if len(state.RateHistory) > 100 {
		state.RateHistory = state.RateHistory[1:]
	}
	
	// 성능 히스토리 (실제 구현에서는 실제 성능 메트릭 수집)
	performance := calculatePerformance(state)
	state.PerformanceHist = append(state.PerformanceHist, performance)
	if len(state.PerformanceHist) > 100 {
		state.PerformanceHist = state.PerformanceHist[1:]
	}
}

// Helper functions

func sigmoid(x float64) float64 {
	return 1.0 / (1.0 + math.Exp(-x))
}

func calculateAverage(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func calculateVariance(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	
	avg := calculateAverage(values)
	sumSquares := 0.0
	
	for _, v := range values {
		diff := v - avg
		sumSquares += diff * diff
	}
	
	return sumSquares / float64(len(values)-1)
}

func calculateTrend(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	
	// 간단한 선형 회귀
	n := float64(len(values))
	sumX := 0.0
	sumY := 0.0
	sumXY := 0.0
	sumX2 := 0.0
	
	for i, y := range values {
		x := float64(i)
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}
	
	denominator := n*sumX2 - sumX*sumX
	if denominator == 0 {
		return 0
	}
	
	slope := (n*sumXY - sumX*sumY) / denominator
	return slope
}

func calculateStability(values []float64) float64 {
	if len(values) < 2 {
		return 1.0
	}
	
	variance := calculateVariance(values)
	avg := calculateAverage(values)
	
	if avg == 0 {
		return 0
	}
	
	// 변동계수의 역수 (안정성 지표)
	cv := math.Sqrt(variance) / avg
	if cv == 0 {
		return 1.0
	}
	
	return 1.0 / (1.0 + cv)
}

func calculatePerformance(state *ThrottleState) float64 {
	// 성능 지표 계산 (실제 구현에서는 실제 메트릭 사용)
	// 여기서는 단순화된 예시
	if state.ThrottledCount == 0 {
		return 1.0
	}
	
	// 스로틀링 비율의 역수를 성능 지표로 사용
	totalMessages := float64(state.ThrottledCount + state.AdjustmentCount*100)
	if totalMessages == 0 {
		return 1.0
	}
	
	return 1.0 - (float64(state.ThrottledCount) / totalMessages)
}

// String 스로틀 이유 문자열 변환
func (r ThrottleReason) String() string {
	switch r {
	case ThrottleBackpressure:
		return "Backpressure"
	case ThrottleSystemLoad:
		return "SystemLoad"
	case ThrottleNetworkQuality:
		return "NetworkQuality"
	case ThrottleClientCapacity:
		return "ClientCapacity"
	case ThrottleManual:
		return "Manual"
	default:
		return "Unknown"
	}
}