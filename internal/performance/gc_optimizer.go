package performance

import (
	"runtime"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

// GCOptimizer 가비지 컬렉션 최적화기
type GCOptimizer struct {
	enabled       bool
	targetPause   time.Duration
	gcPercent     int
	memLimit      int64
	lastGC        time.Time
	gcCount       uint32
	totalPause    time.Duration
	maxPause      time.Duration
	metrics       *GCMetrics
	mutex         sync.RWMutex
}

// GCMetrics GC 메트릭
type GCMetrics struct {
	NumGC          uint32
	PauseTotal     time.Duration
	PauseAvg       time.Duration
	PauseMax       time.Duration
	LastGC         time.Time
	NextGC         uint64
	HeapAlloc      uint64
	HeapSys        uint64
	HeapIdle       uint64
	HeapInuse      uint64
	HeapReleased   uint64
}

// NewGCOptimizer 새 GC 최적화기 생성
func NewGCOptimizer(enabled bool) *GCOptimizer {
	gco := &GCOptimizer{
		enabled:     enabled,
		targetPause: time.Millisecond, // 목표 STW 시간 1ms
		gcPercent:   100,               // 기본 GC 퍼센트
		memLimit:    0,                 // 메모리 제한 없음
		metrics:     &GCMetrics{},
	}
	
	if enabled {
		gco.initialize()
	}
	
	return gco
}

// initialize GC 최적화 초기화
func (gco *GCOptimizer) initialize() {
	// GOGC 설정
	debug.SetGCPercent(gco.gcPercent)
	
	// 메모리 제한 설정 (Go 1.19+)
	if gco.memLimit > 0 {
		debug.SetMemoryLimit(gco.memLimit)
	}
	
	// 초기 GC 통계 수집
	gco.collectStats()
}

// Optimize GC 최적화 실행
func (gco *GCOptimizer) Optimize() {
	if !gco.enabled {
		return
	}
	
	gco.mutex.Lock()
	defer gco.mutex.Unlock()
	
	// 현재 메모리 상태 확인
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	// 동적 GC 퍼센트 조정
	gco.adjustGCPercent(&memStats)
	
	// 필요시 수동 GC 트리거
	if gco.shouldTriggerGC(&memStats) {
		gco.triggerGC()
	}
	
	// 메모리 반환
	if gco.shouldReleaseMemory(&memStats) {
		gco.releaseMemory()
	}
	
	// 통계 업데이트
	gco.collectStats()
}

// adjustGCPercent GC 퍼센트 동적 조정
func (gco *GCOptimizer) adjustGCPercent(memStats *runtime.MemStats) {
	// 평균 일시정지 시간 계산
	pauseNs := memStats.PauseTotalNs
	numGC := memStats.NumGC
	
	if numGC == 0 {
		return
	}
	
	avgPause := time.Duration(pauseNs / uint64(numGC))
	
	// 목표 일시정지 시간과 비교
	if avgPause > gco.targetPause*2 {
		// 일시정지가 너무 길면 GC 빈도 감소
		newPercent := gco.gcPercent + 20
		if newPercent > 200 {
			newPercent = 200
		}
		
		if newPercent != gco.gcPercent {
			debug.SetGCPercent(newPercent)
			gco.gcPercent = newPercent
			log.Debugf("Adjusted GC percent to %d (avg pause: %v)", newPercent, avgPause)
		}
	} else if avgPause < gco.targetPause/2 {
		// 일시정지가 충분히 짧으면 GC 빈도 증가 (메모리 절약)
		newPercent := gco.gcPercent - 10
		if newPercent < 50 {
			newPercent = 50
		}
		
		if newPercent != gco.gcPercent {
			debug.SetGCPercent(newPercent)
			gco.gcPercent = newPercent
			log.Debugf("Adjusted GC percent to %d (avg pause: %v)", newPercent, avgPause)
		}
	}
}

// shouldTriggerGC 수동 GC 트리거 여부 결정
func (gco *GCOptimizer) shouldTriggerGC(memStats *runtime.MemStats) bool {
	// 마지막 GC 이후 시간
	timeSinceGC := time.Since(gco.lastGC)
	
	// 메모리 압박 상태 확인
	heapInUse := memStats.HeapInuse
	heapSys := memStats.HeapSys
	
	if heapSys == 0 {
		return false
	}
	
	memoryPressure := float64(heapInUse) / float64(heapSys)
	
	// 조건 확인
	return timeSinceGC > 30*time.Second && memoryPressure > 0.8
}

// triggerGC 수동 GC 실행
func (gco *GCOptimizer) triggerGC() {
	startTime := time.Now()
	runtime.GC()
	duration := time.Since(startTime)
	
	atomic.AddUint32(&gco.gcCount, 1)
	gco.lastGC = time.Now()
	gco.totalPause += duration
	
	if duration > gco.maxPause {
		gco.maxPause = duration
	}
	
	log.Debugf("Manual GC triggered (duration: %v)", duration)
}

// shouldReleaseMemory 메모리 반환 여부 결정
func (gco *GCOptimizer) shouldReleaseMemory(memStats *runtime.MemStats) bool {
	// 유휴 힙이 전체 힙의 30% 이상인 경우
	if memStats.HeapSys == 0 {
		return false
	}
	
	idleRatio := float64(memStats.HeapIdle) / float64(memStats.HeapSys)
	return idleRatio > 0.3
}

// releaseMemory OS에 메모리 반환
func (gco *GCOptimizer) releaseMemory() {
	debug.FreeOSMemory()
	log.Debug("Released memory to OS")
}

// collectStats GC 통계 수집
func (gco *GCOptimizer) collectStats() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	gco.metrics.NumGC = memStats.NumGC
	gco.metrics.PauseTotal = time.Duration(memStats.PauseTotalNs)
	
	if memStats.NumGC > 0 {
		gco.metrics.PauseAvg = time.Duration(memStats.PauseTotalNs / uint64(memStats.NumGC))
	}
	
	// 최근 일시정지 시간 중 최대값
	if len(memStats.PauseNs) > 0 {
		maxPause := uint64(0)
		for i := 0; i < len(memStats.PauseNs) && memStats.PauseNs[i] > 0; i++ {
			if memStats.PauseNs[i] > maxPause {
				maxPause = memStats.PauseNs[i]
			}
		}
		gco.metrics.PauseMax = time.Duration(maxPause)
	}
	
	gco.metrics.LastGC = time.Unix(0, int64(memStats.LastGC))
	gco.metrics.NextGC = memStats.NextGC
	gco.metrics.HeapAlloc = memStats.HeapAlloc
	gco.metrics.HeapSys = memStats.HeapSys
	gco.metrics.HeapIdle = memStats.HeapIdle
	gco.metrics.HeapInuse = memStats.HeapInuse
	gco.metrics.HeapReleased = memStats.HeapReleased
}

// GetMetrics 메트릭 조회
func (gco *GCOptimizer) GetMetrics() *GCMetrics {
	gco.mutex.RLock()
	defer gco.mutex.RUnlock()
	
	return gco.metrics
}

// SetTargetPause 목표 일시정지 시간 설정
func (gco *GCOptimizer) SetTargetPause(pause time.Duration) {
	gco.mutex.Lock()
	defer gco.mutex.Unlock()
	
	gco.targetPause = pause
	log.Infof("GC target pause set to %v", pause)
}

// SetMemoryLimit 메모리 제한 설정
func (gco *GCOptimizer) SetMemoryLimit(limit int64) {
	gco.mutex.Lock()
	defer gco.mutex.Unlock()
	
	gco.memLimit = limit
	if gco.enabled && limit > 0 {
		debug.SetMemoryLimit(limit)
		log.Infof("Memory limit set to %d bytes", limit)
	}
}

// Enable GC 최적화 활성화
func (gco *GCOptimizer) Enable() {
	gco.mutex.Lock()
	defer gco.mutex.Unlock()
	
	if !gco.enabled {
		gco.enabled = true
		gco.initialize()
		log.Info("GC optimizer enabled")
	}
}

// Disable GC 최적화 비활성화
func (gco *GCOptimizer) Disable() {
	gco.mutex.Lock()
	defer gco.mutex.Unlock()
	
	if gco.enabled {
		gco.enabled = false
		debug.SetGCPercent(100) // 기본값으로 복원
		log.Info("GC optimizer disabled")
	}
}

// ForceGC 강제 GC 실행
func (gco *GCOptimizer) ForceGC() {
	gco.triggerGC()
}

// TuneForLatency 지연시간 최적화 튜닝
func (gco *GCOptimizer) TuneForLatency() {
	gco.mutex.Lock()
	defer gco.mutex.Unlock()
	
	// 낮은 지연시간을 위한 설정
	gco.gcPercent = 50
	gco.targetPause = 500 * time.Microsecond
	
	if gco.enabled {
		debug.SetGCPercent(gco.gcPercent)
	}
	
	log.Info("GC tuned for low latency")
}

// TuneForThroughput 처리량 최적화 튜닝
func (gco *GCOptimizer) TuneForThroughput() {
	gco.mutex.Lock()
	defer gco.mutex.Unlock()
	
	// 높은 처리량을 위한 설정
	gco.gcPercent = 200
	gco.targetPause = 10 * time.Millisecond
	
	if gco.enabled {
		debug.SetGCPercent(gco.gcPercent)
	}
	
	log.Info("GC tuned for high throughput")
}

// TuneForMemory 메모리 최적화 튜닝
func (gco *GCOptimizer) TuneForMemory() {
	gco.mutex.Lock()
	defer gco.mutex.Unlock()
	
	// 낮은 메모리 사용을 위한 설정
	gco.gcPercent = 30
	gco.targetPause = 2 * time.Millisecond
	
	if gco.enabled {
		debug.SetGCPercent(gco.gcPercent)
	}
	
	log.Info("GC tuned for low memory usage")
}