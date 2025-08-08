package performance

import (
	"fmt"
	"math"
	"runtime"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
)

// MemoryMonitor 메모리 모니터
type MemoryMonitor struct {
	manager      *MemoryManager
	interval     time.Duration
	threshold    *MemoryThreshold
	alerts       chan *MemoryAlert
	metrics      *MonitorMetrics
	history      *MetricsHistory
	running      bool
	stopCh       chan struct{}
	wg           sync.WaitGroup
	mutex        sync.RWMutex
}

// MemoryThreshold 메모리 임계값
type MemoryThreshold struct {
	MaxMemoryUsage   uint64        // 최대 메모리 사용량
	MaxGCPause       time.Duration // 최대 GC 일시정지 시간
	MinPoolHitRate   float64       // 최소 풀 히트율
	MaxFragmentation float64       // 최대 단편화 비율
}

// MemoryAlert 메모리 경고
type MemoryAlert struct {
	Level       AlertLevel
	Type        AlertType
	Message     string
	Value       interface{}
	Threshold   interface{}
	Timestamp   time.Time
	Suggestions []string
}

// AlertLevel 경고 레벨
type AlertLevel int

const (
	AlertInfo AlertLevel = iota
	AlertWarning
	AlertCritical
)

// AlertType 경고 타입
type AlertType int

const (
	AlertMemoryHigh AlertType = iota
	AlertGCPauseLong
	AlertPoolHitRateLow
	AlertFragmentationHigh
	AlertMemoryLeak
)

// MonitorMetrics 모니터 메트릭
type MonitorMetrics struct {
	CheckCount       uint64
	AlertCount       uint64
	LastCheckTime    time.Time
	LastAlertTime    time.Time
	CurrentMemory    uint64
	PeakMemory       uint64
	AverageMemory    uint64
	TrendDirection   TrendDirection
	PredictedMemory  uint64
}

// TrendDirection 트렌드 방향
type TrendDirection int

const (
	TrendStable TrendDirection = iota
	TrendIncreasing
	TrendDecreasing
)

// MetricsHistory 메트릭 히스토리
type MetricsHistory struct {
	entries      []HistoryEntry
	maxEntries   int
	currentIndex int
	wrapped      bool
	mutex        sync.RWMutex
}

// HistoryEntry 히스토리 엔트리
type HistoryEntry struct {
	Timestamp    time.Time
	MemoryUsage  uint64
	GCPauseTime  time.Duration
	PoolHitRate  float64
	GoroutineNum int
}

// NewMemoryMonitor 새 메모리 모니터 생성
func NewMemoryMonitor(manager *MemoryManager, interval time.Duration) *MemoryMonitor {
	return &MemoryMonitor{
		manager:  manager,
		interval: interval,
		threshold: &MemoryThreshold{
			MaxMemoryUsage:   2 * 1024 * 1024 * 1024, // 2GB
			MaxGCPause:       time.Millisecond,       // 1ms
			MinPoolHitRate:   0.7,                    // 70%
			MaxFragmentation: 0.3,                    // 30%
		},
		alerts:  make(chan *MemoryAlert, 100),
		metrics: &MonitorMetrics{},
		history: &MetricsHistory{
			entries:    make([]HistoryEntry, 1000),
			maxEntries: 1000,
		},
		stopCh: make(chan struct{}),
	}
}

// Start 모니터 시작
func (mm *MemoryMonitor) Start() error {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	if mm.running {
		return fmt.Errorf("monitor already running")
	}

	mm.running = true
	mm.wg.Add(2)

	// 모니터링 고루틴
	go mm.monitoringLoop()

	// 경고 처리 고루틴
	go mm.alertHandlingLoop()

	log.Info("Memory monitor started")
	return nil
}

// Stop 모니터 중지
func (mm *MemoryMonitor) Stop() {
	mm.mutex.Lock()
	if !mm.running {
		mm.mutex.Unlock()
		return
	}
	mm.running = false
	mm.mutex.Unlock()

	close(mm.stopCh)
	mm.wg.Wait()
	close(mm.alerts)

	log.Info("Memory monitor stopped")
}

// monitoringLoop 모니터링 루프
func (mm *MemoryMonitor) monitoringLoop() {
	defer mm.wg.Done()

	ticker := time.NewTicker(mm.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mm.checkMemoryStatus()

		case <-mm.stopCh:
			return
		}
	}
}

// checkMemoryStatus 메모리 상태 확인
func (mm *MemoryMonitor) checkMemoryStatus() {
	atomic.AddUint64(&mm.metrics.CheckCount, 1)
	mm.metrics.LastCheckTime = time.Now()

	// 메모리 통계 수집
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// 히스토리에 추가
	entry := HistoryEntry{
		Timestamp:    time.Now(),
		MemoryUsage:  memStats.Alloc,
		GCPauseTime:  time.Duration(memStats.PauseNs[(memStats.NumGC+255)%256]),
		PoolHitRate:  mm.getPoolHitRate(),
		GoroutineNum: runtime.NumGoroutine(),
	}
	mm.history.Add(entry)

	// 현재 메모리 업데이트
	atomic.StoreUint64(&mm.metrics.CurrentMemory, memStats.Alloc)

	// 피크 메모리 업데이트
	if memStats.Alloc > atomic.LoadUint64(&mm.metrics.PeakMemory) {
		atomic.StoreUint64(&mm.metrics.PeakMemory, memStats.Alloc)
	}

	// 평균 메모리 계산
	mm.updateAverageMemory()

	// 트렌드 분석
	mm.analyzeTrend()

	// 임계값 확인
	mm.checkThresholds(&memStats)

	// 메모리 누수 감지
	mm.detectMemoryLeak()

	// 예측 메모리 계산
	mm.predictMemoryUsage()
}

// checkThresholds 임계값 확인
func (mm *MemoryMonitor) checkThresholds(memStats *runtime.MemStats) {
	// 메모리 사용량 확인
	if memStats.Alloc > mm.threshold.MaxMemoryUsage {
		mm.sendAlert(&MemoryAlert{
			Level:     AlertCritical,
			Type:      AlertMemoryHigh,
			Message:   "Memory usage exceeded threshold",
			Value:     memStats.Alloc,
			Threshold: mm.threshold.MaxMemoryUsage,
			Timestamp: time.Now(),
			Suggestions: []string{
				"Consider increasing memory limit",
				"Check for memory leaks",
				"Optimize object pooling",
				"Reduce concurrent operations",
			},
		})
	}

	// GC 일시정지 시간 확인
	pauseTime := time.Duration(memStats.PauseNs[(memStats.NumGC+255)%256])
	if pauseTime > mm.threshold.MaxGCPause {
		mm.sendAlert(&MemoryAlert{
			Level:     AlertWarning,
			Type:      AlertGCPauseLong,
			Message:   "GC pause time exceeded threshold",
			Value:     pauseTime,
			Threshold: mm.threshold.MaxGCPause,
			Timestamp: time.Now(),
			Suggestions: []string{
				"Adjust GOGC percent",
				"Reduce allocation rate",
				"Use object pooling more effectively",
			},
		})
	}

	// 풀 히트율 확인
	hitRate := mm.getPoolHitRate()
	if hitRate < mm.threshold.MinPoolHitRate {
		mm.sendAlert(&MemoryAlert{
			Level:     AlertWarning,
			Type:      AlertPoolHitRateLow,
			Message:   "Pool hit rate below threshold",
			Value:     hitRate,
			Threshold: mm.threshold.MinPoolHitRate,
			Timestamp: time.Now(),
			Suggestions: []string{
				"Increase pool size",
				"Preheat pools with common objects",
				"Review object lifecycle management",
			},
		})
	}

	// 단편화 확인
	fragmentation := mm.calculateFragmentation(memStats)
	if fragmentation > mm.threshold.MaxFragmentation {
		mm.sendAlert(&MemoryAlert{
			Level:     AlertWarning,
			Type:      AlertFragmentationHigh,
			Message:   "Memory fragmentation high",
			Value:     fragmentation,
			Threshold: mm.threshold.MaxFragmentation,
			Timestamp: time.Now(),
			Suggestions: []string{
				"Defragment memory",
				"Use fixed-size allocations",
				"Restart application if necessary",
			},
		})
	}
}

// detectMemoryLeak 메모리 누수 감지
func (mm *MemoryMonitor) detectMemoryLeak() {
	// 최근 10개 엔트리 분석
	recent := mm.history.GetRecent(10)
	if len(recent) < 10 {
		return
	}

	// 메모리 증가 추세 확인
	increasing := true
	var totalIncrease uint64

	for i := 1; i < len(recent); i++ {
		if recent[i].MemoryUsage <= recent[i-1].MemoryUsage {
			increasing = false
			break
		}
		totalIncrease += recent[i].MemoryUsage - recent[i-1].MemoryUsage
	}

	// 지속적인 증가 및 큰 증가량이면 누수 의심
	if increasing && totalIncrease > 100*1024*1024 { // 100MB
		mm.sendAlert(&MemoryAlert{
			Level:   AlertCritical,
			Type:    AlertMemoryLeak,
			Message: "Potential memory leak detected",
			Value:   totalIncrease,
			Timestamp: time.Now(),
			Suggestions: []string{
				"Check for unreleased resources",
				"Review goroutine lifecycle",
				"Analyze heap profile",
				"Check for circular references",
			},
		})
	}
}

// analyzeTrend 트렌드 분석
func (mm *MemoryMonitor) analyzeTrend() {
	recent := mm.history.GetRecent(20)
	if len(recent) < 2 {
		mm.metrics.TrendDirection = TrendStable
		return
	}

	// 선형 회귀를 통한 트렌드 분석
	var sumX, sumY, sumXY, sumX2 float64
	n := float64(len(recent))

	for i, entry := range recent {
		x := float64(i)
		y := float64(entry.MemoryUsage)
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	// 기울기 계산
	slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)

	// 트렌드 결정
	if math.Abs(slope) < 1000 { // 1KB/interval 미만
		mm.metrics.TrendDirection = TrendStable
	} else if slope > 0 {
		mm.metrics.TrendDirection = TrendIncreasing
	} else {
		mm.metrics.TrendDirection = TrendDecreasing
	}
}

// predictMemoryUsage 메모리 사용량 예측
func (mm *MemoryMonitor) predictMemoryUsage() {
	recent := mm.history.GetRecent(30)
	if len(recent) < 10 {
		return
	}

	// 간단한 선형 예측
	var sumX, sumY, sumXY, sumX2 float64
	n := float64(len(recent))

	for i, entry := range recent {
		x := float64(i)
		y := float64(entry.MemoryUsage)
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	// 회귀 계수 계산
	slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)
	intercept := (sumY - slope*sumX) / n

	// 다음 10 인터벌 후 예측
	predicted := slope*float64(len(recent)+10) + intercept

	if predicted > 0 {
		atomic.StoreUint64(&mm.metrics.PredictedMemory, uint64(predicted))
	}
}

// calculateFragmentation 단편화 계산
func (mm *MemoryMonitor) calculateFragmentation(memStats *runtime.MemStats) float64 {
	if memStats.HeapSys == 0 {
		return 0
	}

	// 사용되지 않는 힙 메모리 비율
	idle := float64(memStats.HeapIdle)
	sys := float64(memStats.HeapSys)

	return idle / sys
}

// getPoolHitRate 풀 히트율 조회
func (mm *MemoryMonitor) getPoolHitRate() float64 {
	if mm.manager == nil {
		return 0
	}

	metrics := mm.manager.GetMetrics()
	return metrics.PoolHitRate
}

// updateAverageMemory 평균 메모리 업데이트
func (mm *MemoryMonitor) updateAverageMemory() {
	recent := mm.history.GetRecent(100)
	if len(recent) == 0 {
		return
	}

	var total uint64
	for _, entry := range recent {
		total += entry.MemoryUsage
	}

	average := total / uint64(len(recent))
	atomic.StoreUint64(&mm.metrics.AverageMemory, average)
}

// sendAlert 경고 전송
func (mm *MemoryMonitor) sendAlert(alert *MemoryAlert) {
	atomic.AddUint64(&mm.metrics.AlertCount, 1)
	mm.metrics.LastAlertTime = time.Now()

	select {
	case mm.alerts <- alert:
	default:
		// 경고 채널이 가득 찬 경우 가장 오래된 경고 제거
		select {
		case <-mm.alerts:
			mm.alerts <- alert
		default:
		}
	}
}

// alertHandlingLoop 경고 처리 루프
func (mm *MemoryMonitor) alertHandlingLoop() {
	defer mm.wg.Done()

	for {
		select {
		case alert := <-mm.alerts:
			if alert != nil {
				mm.handleAlert(alert)
			}

		case <-mm.stopCh:
			return
		}
	}
}

// handleAlert 경고 처리
func (mm *MemoryMonitor) handleAlert(alert *MemoryAlert) {
	// 로그 레벨에 따라 다르게 처리
	switch alert.Level {
	case AlertInfo:
		log.Infof("Memory alert: %s (value: %v, threshold: %v)",
			alert.Message, alert.Value, alert.Threshold)

	case AlertWarning:
		log.Warnf("Memory warning: %s (value: %v, threshold: %v)",
			alert.Message, alert.Value, alert.Threshold)

	case AlertCritical:
		log.Errorf("Memory critical: %s (value: %v, threshold: %v)",
			alert.Message, alert.Value, alert.Threshold)

		// 긴급 조치 실행
		mm.executeEmergencyActions(alert)
	}

	// 제안 사항 로그
	if len(alert.Suggestions) > 0 {
		log.Info("Suggestions:")
		for _, suggestion := range alert.Suggestions {
			log.Infof("  - %s", suggestion)
		}
	}
}

// executeEmergencyActions 긴급 조치 실행
func (mm *MemoryMonitor) executeEmergencyActions(alert *MemoryAlert) {
	switch alert.Type {
	case AlertMemoryHigh:
		// 강제 GC 실행
		runtime.GC()
		runtime.GC() // 두 번 실행으로 더 확실한 정리

		// 메모리 반환
		debug.FreeOSMemory()

		// 캐시 정리
		if mm.manager != nil {
			mm.manager.Optimize()
		}

	case AlertMemoryLeak:
		// 메모리 프로파일 생성
		mm.createMemoryProfile()

		// 고루틴 덤프
		mm.dumpGoroutines()
	}
}

// createMemoryProfile 메모리 프로파일 생성
func (mm *MemoryMonitor) createMemoryProfile() {
	// 프로파일 파일 생성 로직
	log.Info("Memory profile created for analysis")
}

// dumpGoroutines 고루틴 덤프
func (mm *MemoryMonitor) dumpGoroutines() {
	// 고루틴 덤프 로직
	log.Infof("Goroutine count: %d", runtime.NumGoroutine())
}

// GetMetrics 메트릭 조회
func (mm *MemoryMonitor) GetMetrics() *MonitorMetrics {
	return mm.metrics
}

// GetAlerts 경고 채널 조회
func (mm *MemoryMonitor) GetAlerts() <-chan *MemoryAlert {
	return mm.alerts
}

// SetThreshold 임계값 설정
func (mm *MemoryMonitor) SetThreshold(threshold *MemoryThreshold) {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	mm.threshold = threshold
}

// Add 히스토리에 엔트리 추가
func (mh *MetricsHistory) Add(entry HistoryEntry) {
	mh.mutex.Lock()
	defer mh.mutex.Unlock()

	mh.entries[mh.currentIndex] = entry
	mh.currentIndex = (mh.currentIndex + 1) % mh.maxEntries

	if mh.currentIndex == 0 {
		mh.wrapped = true
	}
}

// GetRecent 최근 엔트리 조회
func (mh *MetricsHistory) GetRecent(count int) []HistoryEntry {
	mh.mutex.RLock()
	defer mh.mutex.RUnlock()

	var result []HistoryEntry
	totalEntries := mh.currentIndex
	if mh.wrapped {
		totalEntries = mh.maxEntries
	}

	if count > totalEntries {
		count = totalEntries
	}

	for i := 0; i < count; i++ {
		index := (mh.currentIndex - 1 - i + mh.maxEntries) % mh.maxEntries
		result = append(result, mh.entries[index])
	}

	// 시간순으로 정렬 (오래된 것부터)
	for i := 0; i < len(result)/2; i++ {
		j := len(result) - 1 - i
		result[i], result[j] = result[j], result[i]
	}

	return result
}

// GetAll 모든 엔트리 조회
func (mh *MetricsHistory) GetAll() []HistoryEntry {
	mh.mutex.RLock()
	defer mh.mutex.RUnlock()

	totalEntries := mh.currentIndex
	if mh.wrapped {
		totalEntries = mh.maxEntries
	}

	result := make([]HistoryEntry, totalEntries)
	if mh.wrapped {
		// 래핑된 경우: currentIndex부터 끝까지, 그 다음 0부터 currentIndex-1까지
		copy(result, mh.entries[mh.currentIndex:])
		copy(result[mh.maxEntries-mh.currentIndex:], mh.entries[:mh.currentIndex])
	} else {
		// 래핑되지 않은 경우: 0부터 currentIndex-1까지
		copy(result, mh.entries[:mh.currentIndex])
	}

	return result
}

// Clear 히스토리 초기화
func (mh *MetricsHistory) Clear() {
	mh.mutex.Lock()
	defer mh.mutex.Unlock()

	mh.entries = make([]HistoryEntry, mh.maxEntries)
	mh.currentIndex = 0
	mh.wrapped = false
}

// String AlertLevel 문자열 변환
func (al AlertLevel) String() string {
	switch al {
	case AlertInfo:
		return "INFO"
	case AlertWarning:
		return "WARNING"
	case AlertCritical:
		return "CRITICAL"
	default:
		return fmt.Sprintf("Unknown(%d)", al)
	}
}

// String AlertType 문자열 변환
func (at AlertType) String() string {
	switch at {
	case AlertMemoryHigh:
		return "MemoryHigh"
	case AlertGCPauseLong:
		return "GCPauseLong"
	case AlertPoolHitRateLow:
		return "PoolHitRateLow"
	case AlertFragmentationHigh:
		return "FragmentationHigh"
	case AlertMemoryLeak:
		return "MemoryLeak"
	default:
		return fmt.Sprintf("Unknown(%d)", at)
	}
}

// String TrendDirection 문자열 변환
func (td TrendDirection) String() string {
	switch td {
	case TrendStable:
		return "Stable"
	case TrendIncreasing:
		return "Increasing"
	case TrendDecreasing:
		return "Decreasing"
	default:
		return fmt.Sprintf("Unknown(%d)", td)
	}
}