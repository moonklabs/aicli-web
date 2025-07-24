package claude

import (
	"context"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

// AutoScaler는 자동 스케일링을 담당합니다
type AutoScaler struct {
	pool         *AdvancedSessionPool
	config       AutoScalingConfig
	
	// 상태 관리
	running      atomic.Bool
	lastAction   atomic.Value // string
	lastScaleUp  time.Time
	lastScaleDown time.Time
	
	// 메트릭 수집
	metricsWindow []ScalingMetric
	windowSize    int
	windowMutex   sync.RWMutex
	
	// 생명주기 관리
	ctx          context.Context
	cancel       context.CancelFunc
	ticker       *time.Ticker
}

// ScalingMetric은 스케일링 판단을 위한 메트릭입니다
type ScalingMetric struct {
	Timestamp     time.Time `json:"timestamp"`
	Utilization   float64   `json:"utilization"`
	CPUUsage      float64   `json:"cpu_usage"`
	MemoryUsage   int64     `json:"memory_usage"`
	QueueLength   int       `json:"queue_length"`
	ResponseTime  time.Duration `json:"response_time"`
	ErrorRate     float64   `json:"error_rate"`
	ThroughputRPS float64   `json:"throughput_rps"`
}

// ScalingDecision은 스케일링 결정 정보입니다
type ScalingDecision struct {
	Action       ScalingAction `json:"action"`
	Reason       string        `json:"reason"`
	CurrentSize  int           `json:"current_size"`
	TargetSize   int           `json:"target_size"`
	Confidence   float64       `json:"confidence"`
	Timestamp    time.Time     `json:"timestamp"`
}

// ScalingAction은 스케일링 액션 타입입니다
type ScalingAction int

const (
	ScaleNone ScalingAction = iota
	ScaleUp
	ScaleDown
	ScaleOut  // 긴급 확장
	ScaleIn   // 강제 축소
)

// NewAutoScaler는 새로운 자동 스케일러를 생성합니다
func NewAutoScaler(pool *AdvancedSessionPool, config AutoScalingConfig) *AutoScaler {
	ctx, cancel := context.WithCancel(context.Background())
	
	scaler := &AutoScaler{
		pool:        pool,
		config:      config,
		ctx:         ctx,
		cancel:      cancel,
		windowSize:  20, // 20개 메트릭 윈도우
		ticker:      time.NewTicker(30 * time.Second), // 30초마다 평가
	}
	
	// 초기 액션 설정
	scaler.lastAction.Store("none")
	
	return scaler
}

// Start는 자동 스케일러를 시작합니다
func (s *AutoScaler) Start() {
	if !s.running.CompareAndSwap(false, true) {
		return // 이미 실행 중
	}
	
	go s.scalingLoop()
}

// Stop은 자동 스케일러를 중지합니다
func (s *AutoScaler) Stop() {
	if !s.running.CompareAndSwap(true, false) {
		return // 이미 중지됨
	}
	
	s.cancel()
	s.ticker.Stop()
}

// ScaleUp은 수동 스케일업을 수행합니다
func (s *AutoScaler) ScaleUp() error {
	if !s.canScaleUp() {
		return fmt.Errorf("scale up cooldown not elapsed")
	}
	
	currentSize := s.pool.basePool.GetPoolStats().Total
	targetSize := int(math.Ceil(float64(currentSize) * s.config.ScaleFactor))
	
	if targetSize > s.config.MaxSessions {
		targetSize = s.config.MaxSessions
	}
	
	return s.scaleToSize(targetSize, "manual_scale_up")
}

// ScaleDown은 수동 스케일다운을 수행합니다
func (s *AutoScaler) ScaleDown() error {
	if !s.canScaleDown() {
		return fmt.Errorf("scale down cooldown not elapsed")
	}
	
	currentSize := s.pool.basePool.GetPoolStats().Total
	targetSize := int(math.Floor(float64(currentSize) / s.config.ScaleFactor))
	
	if targetSize < s.config.MinSessions {
		targetSize = s.config.MinSessions
	}
	
	return s.scaleToSize(targetSize, "manual_scale_down")
}

// ScaleUpBy는 지정된 수만큼 스케일업합니다
func (s *AutoScaler) ScaleUpBy(count int) error {
	currentSize := s.pool.basePool.GetPoolStats().Total
	targetSize := currentSize + count
	
	if targetSize > s.config.MaxSessions {
		targetSize = s.config.MaxSessions
	}
	
	return s.scaleToSize(targetSize, fmt.Sprintf("scale_up_by_%d", count))
}

// ScaleDownBy는 지정된 수만큼 스케일다운합니다
func (s *AutoScaler) ScaleDownBy(count int) error {
	currentSize := s.pool.basePool.GetPoolStats().Total
	targetSize := currentSize - count
	
	if targetSize < s.config.MinSessions {
		targetSize = s.config.MinSessions
	}
	
	return s.scaleToSize(targetSize, fmt.Sprintf("scale_down_by_%d", count))
}

// ConsiderScaleDown은 스케일다운을 검토합니다
func (s *AutoScaler) ConsiderScaleDown() {
	if !s.running.Load() || !s.canScaleDown() {
		return
	}
	
	decision := s.evaluateScaling()
	if decision.Action == ScaleDown {
		s.executeScalingDecision(decision)
	}
}

// GetLastAction은 마지막 스케일링 액션을 반환합니다
func (s *AutoScaler) GetLastAction() string {
	return s.lastAction.Load().(string)
}

// GetScalingHistory는 스케일링 히스토리를 반환합니다
func (s *AutoScaler) GetScalingHistory() []ScalingMetric {
	s.windowMutex.RLock()
	defer s.windowMutex.RUnlock()
	
	// 복사본 반환
	history := make([]ScalingMetric, len(s.metricsWindow))
	copy(history, s.metricsWindow)
	
	return history
}

// 내부 메서드들

func (s *AutoScaler) scalingLoop() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-s.ticker.C:
			s.collectMetrics()
			s.evaluateAndScale()
		}
	}
}

func (s *AutoScaler) collectMetrics() {
	stats := s.pool.GetPoolStats()
	
	metric := ScalingMetric{
		Timestamp:     time.Now(),
		Utilization:   stats.Utilization,
		CPUUsage:      stats.CPUUsage,
		MemoryUsage:   stats.MemoryUsage,
		ResponseTime:  stats.AverageLatency,
		ErrorRate:     stats.ErrorRate,
		ThroughputRPS: stats.ThroughputRPS,
	}
	
	s.windowMutex.Lock()
	defer s.windowMutex.Unlock()
	
	// 윈도우에 메트릭 추가
	s.metricsWindow = append(s.metricsWindow, metric)
	
	// 윈도우 크기 제한
	if len(s.metricsWindow) > s.windowSize {
		s.metricsWindow = s.metricsWindow[1:]
	}
}

func (s *AutoScaler) evaluateAndScale() {
	decision := s.evaluateScaling()
	
	if decision.Action != ScaleNone {
		s.executeScalingDecision(decision)
	}
}

func (s *AutoScaler) evaluateScaling() ScalingDecision {
	if len(s.metricsWindow) < 3 {
		return ScalingDecision{Action: ScaleNone, Reason: "insufficient_metrics"}
	}
	
	// 최근 메트릭 분석
	avgUtilization := s.calculateAverageUtilization()
	avgCPUUsage := s.calculateAverageCPUUsage()
	avgResponseTime := s.calculateAverageResponseTime()
	currentSize := s.pool.basePool.GetPoolStats().Total
	
	// 스케일업 검토
	if s.shouldScaleUp(avgUtilization, avgCPUUsage, avgResponseTime) {
		targetSize := s.calculateScaleUpTarget(currentSize, avgUtilization)
		confidence := s.calculateScaleUpConfidence(avgUtilization, avgCPUUsage)
		
		return ScalingDecision{
			Action:      ScaleUp,
			Reason:      fmt.Sprintf("high_utilization_%.2f", avgUtilization),
			CurrentSize: currentSize,
			TargetSize:  targetSize,
			Confidence:  confidence,
			Timestamp:   time.Now(),
		}
	}
	
	// 스케일다운 검토
	if s.shouldScaleDown(avgUtilization, avgCPUUsage, avgResponseTime) {
		targetSize := s.calculateScaleDownTarget(currentSize, avgUtilization)
		confidence := s.calculateScaleDownConfidence(avgUtilization, avgCPUUsage)
		
		return ScalingDecision{
			Action:      ScaleDown,
			Reason:      fmt.Sprintf("low_utilization_%.2f", avgUtilization),
			CurrentSize: currentSize,
			TargetSize:  targetSize,
			Confidence:  confidence,
			Timestamp:   time.Now(),
		}
	}
	
	return ScalingDecision{
		Action:    ScaleNone,
		Reason:    "utilization_within_target",
		Timestamp: time.Now(),
	}
}

func (s *AutoScaler) shouldScaleUp(utilization, cpuUsage float64, responseTime time.Duration) bool {
	// 쿨다운 체크
	if !s.canScaleUp() {
		return false
	}
	
	// 이용률 기반 판단
	if utilization > s.config.ScaleUpThreshold {
		return true
	}
	
	// CPU 사용률 기반 판단
	if cpuUsage > 0.8 {
		return true
	}
	
	// 응답 시간 기반 판단
	if responseTime > 5*time.Second {
		return true
	}
	
	// 에러율 기반 판단
	if s.getRecentErrorRate() > 0.05 { // 5% 이상
		return true
	}
	
	return false
}

func (s *AutoScaler) shouldScaleDown(utilization, cpuUsage float64, responseTime time.Duration) bool {
	// 쿨다운 체크
	if !s.canScaleDown() {
		return false
	}
	
	// 최소 세션 수 보호
	currentSize := s.pool.basePool.GetPoolStats().Total
	if currentSize <= s.config.MinSessions {
		return false
	}
	
	// 이용률 기반 판단
	if utilization < s.config.ScaleDownThreshold {
		return true
	}
	
	// CPU 사용률이 매우 낮을 때
	if cpuUsage < 0.2 && utilization < 0.5 {
		return true
	}
	
	return false
}

func (s *AutoScaler) executeScalingDecision(decision ScalingDecision) {
	switch decision.Action {
	case ScaleUp:
		if err := s.scaleToSize(decision.TargetSize, decision.Reason); err != nil {
			fmt.Printf("Scale up failed: %v\n", err)
		}
	case ScaleDown:
		if err := s.scaleToSize(decision.TargetSize, decision.Reason); err != nil {
			fmt.Printf("Scale down failed: %v\n", err)
		}
	}
}

func (s *AutoScaler) scaleToSize(targetSize int, reason string) error {
	currentSize := s.pool.basePool.GetPoolStats().Total
	
	if targetSize == currentSize {
		return nil // 변경 불필요
	}
	
	// 실제 스케일링 실행
	err := s.pool.Scale(targetSize)
	if err != nil {
		return fmt.Errorf("scaling failed: %w", err)
	}
	
	// 스케일링 기록 업데이트
	now := time.Now()
	if targetSize > currentSize {
		s.lastScaleUp = now
		s.lastAction.Store(fmt.Sprintf("scale_up_to_%d_%s", targetSize, reason))
	} else {
		s.lastScaleDown = now
		s.lastAction.Store(fmt.Sprintf("scale_down_to_%d_%s", targetSize, reason))
	}
	
	s.pool.lastScaleTime = now
	
	return nil
}

func (s *AutoScaler) canScaleUp() bool {
	return time.Since(s.lastScaleUp) >= s.config.ScaleUpCooldown
}

func (s *AutoScaler) canScaleDown() bool {
	return time.Since(s.lastScaleDown) >= s.config.ScaleDownCooldown
}

func (s *AutoScaler) calculateAverageUtilization() float64 {
	s.windowMutex.RLock()
	defer s.windowMutex.RUnlock()
	
	if len(s.metricsWindow) == 0 {
		return 0.0
	}
	
	var sum float64
	for _, metric := range s.metricsWindow {
		sum += metric.Utilization
	}
	
	return sum / float64(len(s.metricsWindow))
}

func (s *AutoScaler) calculateAverageCPUUsage() float64 {
	s.windowMutex.RLock()
	defer s.windowMutex.RUnlock()
	
	if len(s.metricsWindow) == 0 {
		return 0.0
	}
	
	var sum float64
	for _, metric := range s.metricsWindow {
		sum += metric.CPUUsage
	}
	
	return sum / float64(len(s.metricsWindow))
}

func (s *AutoScaler) calculateAverageResponseTime() time.Duration {
	s.windowMutex.RLock()
	defer s.windowMutex.RUnlock()
	
	if len(s.metricsWindow) == 0 {
		return 0
	}
	
	var sum time.Duration
	for _, metric := range s.metricsWindow {
		sum += metric.ResponseTime
	}
	
	return sum / time.Duration(len(s.metricsWindow))
}

func (s *AutoScaler) calculateScaleUpTarget(currentSize int, utilization float64) int {
	// 이용률 기반 목표 크기 계산
	factor := s.config.ScaleFactor
	
	// 매우 높은 이용률일 때 더 공격적으로 확장
	if utilization > 0.9 {
		factor = math.Min(factor * 1.5, 3.0)
	}
	
	targetSize := int(math.Ceil(float64(currentSize) * factor))
	
	// 최대값 제한
	if targetSize > s.config.MaxSessions {
		targetSize = s.config.MaxSessions
	}
	
	return targetSize
}

func (s *AutoScaler) calculateScaleDownTarget(currentSize int, utilization float64) int {
	// 이용률 기반 목표 크기 계산
	factor := 1.0 / s.config.ScaleFactor
	
	// 매우 낮은 이용률일 때 더 공격적으로 축소
	if utilization < 0.1 {
		factor = math.Max(factor * 1.5, 0.3)
	}
	
	targetSize := int(math.Floor(float64(currentSize) * factor))
	
	// 최소값 제한
	if targetSize < s.config.MinSessions {
		targetSize = s.config.MinSessions
	}
	
	return targetSize
}

func (s *AutoScaler) calculateScaleUpConfidence(utilization, cpuUsage float64) float64 {
	confidence := 0.0
	
	// 이용률 신뢰도
	if utilization > s.config.ScaleUpThreshold {
		confidence += 0.4 * math.Min((utilization-s.config.ScaleUpThreshold)/(1.0-s.config.ScaleUpThreshold), 1.0)
	}
	
	// CPU 사용률 신뢰도
	if cpuUsage > 0.7 {
		confidence += 0.3 * math.Min((cpuUsage-0.7)/0.3, 1.0)
	}
	
	// 추세 신뢰도
	if s.isUtilizationIncreasing() {
		confidence += 0.3
	}
	
	return math.Min(confidence, 1.0)
}

func (s *AutoScaler) calculateScaleDownConfidence(utilization, cpuUsage float64) float64 {
	confidence := 0.0
	
	// 이용률 신뢰도
	if utilization < s.config.ScaleDownThreshold {
		confidence += 0.4 * math.Min((s.config.ScaleDownThreshold-utilization)/s.config.ScaleDownThreshold, 1.0)
	}
	
	// CPU 사용률 신뢰도
	if cpuUsage < 0.3 {
		confidence += 0.3 * math.Min((0.3-cpuUsage)/0.3, 1.0)
	}
	
	// 추세 신뢰도
	if s.isUtilizationDecreasing() {
		confidence += 0.3
	}
	
	return math.Min(confidence, 1.0)
}

func (s *AutoScaler) isUtilizationIncreasing() bool {
	s.windowMutex.RLock()
	defer s.windowMutex.RUnlock()
	
	if len(s.metricsWindow) < 5 {
		return false
	}
	
	recent := s.metricsWindow[len(s.metricsWindow)-3:]
	older := s.metricsWindow[len(s.metricsWindow)-6 : len(s.metricsWindow)-3]
	
	var recentAvg, olderAvg float64
	for _, m := range recent {
		recentAvg += m.Utilization
	}
	for _, m := range older {
		olderAvg += m.Utilization
	}
	
	recentAvg /= float64(len(recent))
	olderAvg /= float64(len(older))
	
	return recentAvg > olderAvg+0.1 // 10% 이상 증가
}

func (s *AutoScaler) isUtilizationDecreasing() bool {
	s.windowMutex.RLock()
	defer s.windowMutex.RUnlock()
	
	if len(s.metricsWindow) < 5 {
		return false
	}
	
	recent := s.metricsWindow[len(s.metricsWindow)-3:]
	older := s.metricsWindow[len(s.metricsWindow)-6 : len(s.metricsWindow)-3]
	
	var recentAvg, olderAvg float64
	for _, m := range recent {
		recentAvg += m.Utilization
	}
	for _, m := range older {
		olderAvg += m.Utilization
	}
	
	recentAvg /= float64(len(recent))
	olderAvg /= float64(len(older))
	
	return olderAvg > recentAvg+0.1 // 10% 이상 감소
}

func (s *AutoScaler) getRecentErrorRate() float64 {
	s.windowMutex.RLock()
	defer s.windowMutex.RUnlock()
	
	if len(s.metricsWindow) == 0 {
		return 0.0
	}
	
	// 최근 3개 메트릭의 평균 에러율
	count := math.Min(3, float64(len(s.metricsWindow)))
	start := len(s.metricsWindow) - int(count)
	
	var sum float64
	for i := start; i < len(s.metricsWindow); i++ {
		sum += s.metricsWindow[i].ErrorRate
	}
	
	return sum / count
}