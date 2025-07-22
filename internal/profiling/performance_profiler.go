package profiling

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"sync"
	"sync/atomic"
	"time"
)

// PerformanceProfiler는 성능 프로파일링 관리자입니다
type PerformanceProfiler struct {
	// 프로파일러들
	cpuProfiler       *CPUProfiler
	memoryProfiler    *MemoryProfiler
	goroutineProfiler *GoroutineProfiler
	blockProfiler     *BlockProfiler
	
	// 설정
	config ProfilingConfig
	
	// 상태
	running atomic.Bool
	
	// 생명주기
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// ProfilingConfig는 프로파일링 설정입니다
type ProfilingConfig struct {
	EnableCPU        bool   `json:"enable_cpu"`
	EnableMemory     bool   `json:"enable_memory"`
	EnableGoroutine  bool   `json:"enable_goroutine"`
	EnableBlock      bool   `json:"enable_block"`
	EnableMutex      bool   `json:"enable_mutex"`
	
	SampleRate       int    `json:"sample_rate"`       // 샘플링 레이트 (Hz)
	OutputDir        string `json:"output_dir"`        // 출력 디렉토리
	FilePrefix       string `json:"file_prefix"`       // 파일 접두사
	
	// 수집 간격
	CollectionInterval time.Duration `json:"collection_interval"`
	RetentionPeriod    time.Duration `json:"retention_period"`
	
	// 임계값
	MemoryThreshold    int64   `json:"memory_threshold"`    // 메모리 임계값 (바이트)
	GoroutineThreshold int     `json:"goroutine_threshold"` // 고루틴 임계값
	CPUThreshold       float64 `json:"cpu_threshold"`       // CPU 사용률 임계값
	
	// 자동화
	AutoCapture        bool `json:"auto_capture"`        // 자동 캡처
	AutoAnalysis       bool `json:"auto_analysis"`       // 자동 분석
	AutoCleanup        bool `json:"auto_cleanup"`        // 자동 정리
}

// CPUProfiler는 CPU 프로파일러입니다
type CPUProfiler struct {
	config     CPUProfilerConfig
	isRunning  atomic.Bool
	outputFile *os.File
	mutex      sync.Mutex
	
	// 통계
	stats CPUStats
	statsMutex sync.RWMutex
}

// CPUProfilerConfig는 CPU 프로파일러 설정입니다
type CPUProfilerConfig struct {
	SampleRate int           `json:"sample_rate"`
	Duration   time.Duration `json:"duration"`
	OutputPath string        `json:"output_path"`
}

// CPUStats는 CPU 통계입니다
type CPUStats struct {
	SampleCount     int64         `json:"sample_count"`
	TotalDuration   time.Duration `json:"total_duration"`
	AverageCPU      float64       `json:"average_cpu"`
	PeakCPU         float64       `json:"peak_cpu"`
	LastCollection  time.Time     `json:"last_collection"`
}

// MemoryProfiler는 메모리 프로파일러입니다
type MemoryProfiler struct {
	config MemoryProfilerConfig
	
	// 통계
	stats MemoryStats
	statsMutex sync.RWMutex
}

// MemoryProfilerConfig는 메모리 프로파일러 설정입니다
type MemoryProfilerConfig struct {
	SampleInterval time.Duration `json:"sample_interval"`
	OutputPath     string        `json:"output_path"`
	IncludeInUse   bool          `json:"include_in_use"`
	IncludeAllocs  bool          `json:"include_allocs"`
}

// MemoryStats는 메모리 통계입니다
type MemoryStats struct {
	HeapAlloc      uint64    `json:"heap_alloc"`
	HeapSys        uint64    `json:"heap_sys"`
	HeapInuse      uint64    `json:"heap_inuse"`
	HeapReleased   uint64    `json:"heap_released"`
	StackInuse     uint64    `json:"stack_inuse"`
	StackSys       uint64    `json:"stack_sys"`
	MSpanInuse     uint64    `json:"mspan_inuse"`
	MSpanSys       uint64    `json:"mspan_sys"`
	MCacheInuse    uint64    `json:"mcache_inuse"`
	MCacheSys      uint64    `json:"mcache_sys"`
	GCSys          uint64    `json:"gc_sys"`
	NextGC         uint64    `json:"next_gc"`
	LastGC         uint64    `json:"last_gc"`
	NumGC          uint32    `json:"num_gc"`
	NumForcedGC    uint32    `json:"num_forced_gc"`
	GCCPUFraction  float64   `json:"gc_cpu_fraction"`
	LastCollection time.Time `json:"last_collection"`
}

// GoroutineProfiler는 고루틴 프로파일러입니다
type GoroutineProfiler struct {
	config GoroutineProfilerConfig
	
	// 통계
	stats GoroutineStats
	statsMutex sync.RWMutex
}

// GoroutineProfilerConfig는 고루틴 프로파일러 설정입니다
type GoroutineProfilerConfig struct {
	SampleInterval time.Duration `json:"sample_interval"`
	OutputPath     string        `json:"output_path"`
	StackDepth     int           `json:"stack_depth"`
}

// GoroutineStats는 고루틴 통계입니다
type GoroutineStats struct {
	Count          int       `json:"count"`
	PeakCount      int       `json:"peak_count"`
	AverageCount   float64   `json:"average_count"`
	CreatedTotal   int64     `json:"created_total"`
	LastCollection time.Time `json:"last_collection"`
}

// BlockProfiler는 블로킹 프로파일러입니다
type BlockProfiler struct {
	config BlockProfilerConfig
	
	// 통계
	stats BlockStats
	statsMutex sync.RWMutex
}

// BlockProfilerConfig는 블로킹 프로파일러 설정입니다
type BlockProfilerConfig struct {
	Rate       int    `json:"rate"`         // 블로킹 이벤트 샘플링 레이트
	OutputPath string `json:"output_path"`
}

// BlockStats는 블로킹 통계입니다
type BlockStats struct {
	TotalBlocks    int64         `json:"total_blocks"`
	TotalDuration  time.Duration `json:"total_duration"`
	AverageLatency time.Duration `json:"average_latency"`
	MaxLatency     time.Duration `json:"max_latency"`
	LastCollection time.Time     `json:"last_collection"`
}

// ProfilingReport는 프로파일링 리포트입니다
type ProfilingReport struct {
	Timestamp    time.Time        `json:"timestamp"`
	Duration     time.Duration    `json:"duration"`
	CPUStats     CPUStats         `json:"cpu_stats"`
	MemoryStats  MemoryStats      `json:"memory_stats"`
	GoroutineStats GoroutineStats `json:"goroutine_stats"`
	BlockStats   BlockStats       `json:"block_stats"`
	Files        []string         `json:"files"`
	Summary      ReportSummary    `json:"summary"`
}

// ReportSummary는 리포트 요약입니다
type ReportSummary struct {
	PerformanceScore float64 `json:"performance_score"`
	MemoryEfficiency float64 `json:"memory_efficiency"`
	CPUEfficiency    float64 `json:"cpu_efficiency"`
	Recommendations  []string `json:"recommendations"`
	Warnings         []string `json:"warnings"`
	Errors           []string `json:"errors"`
}

// DefaultProfilingConfig는 기본 프로파일링 설정을 반환합니다
func DefaultProfilingConfig() ProfilingConfig {
	return ProfilingConfig{
		EnableCPU:          true,
		EnableMemory:       true,
		EnableGoroutine:    true,
		EnableBlock:        true,
		EnableMutex:        false,
		SampleRate:         100,
		OutputDir:          "./profiles",
		FilePrefix:         "aicli",
		CollectionInterval: 30 * time.Second,
		RetentionPeriod:    24 * time.Hour,
		MemoryThreshold:    100 * 1024 * 1024, // 100MB
		GoroutineThreshold: 1000,
		CPUThreshold:       80.0, // 80%
		AutoCapture:        true,
		AutoAnalysis:       true,
		AutoCleanup:        true,
	}
}

// NewPerformanceProfiler는 새로운 성능 프로파일러를 생성합니다
func NewPerformanceProfiler(config ProfilingConfig) (*PerformanceProfiler, error) {
	// 출력 디렉토리 생성
	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	profiler := &PerformanceProfiler{
		config: config,
		ctx:    ctx,
		cancel: cancel,
	}
	
	// 개별 프로파일러들 초기화
	if config.EnableCPU {
		cpuConfig := CPUProfilerConfig{
			SampleRate: config.SampleRate,
			Duration:   config.CollectionInterval,
			OutputPath: filepath.Join(config.OutputDir, config.FilePrefix+"_cpu.prof"),
		}
		profiler.cpuProfiler = NewCPUProfiler(cpuConfig)
	}
	
	if config.EnableMemory {
		memConfig := MemoryProfilerConfig{
			SampleInterval: config.CollectionInterval,
			OutputPath:     filepath.Join(config.OutputDir, config.FilePrefix+"_mem.prof"),
			IncludeInUse:   true,
			IncludeAllocs:  true,
		}
		profiler.memoryProfiler = NewMemoryProfiler(memConfig)
	}
	
	if config.EnableGoroutine {
		goroutineConfig := GoroutineProfilerConfig{
			SampleInterval: config.CollectionInterval,
			OutputPath:     filepath.Join(config.OutputDir, config.FilePrefix+"_goroutine.prof"),
			StackDepth:     32,
		}
		profiler.goroutineProfiler = NewGoroutineProfiler(goroutineConfig)
	}
	
	if config.EnableBlock {
		blockConfig := BlockProfilerConfig{
			Rate:       1,
			OutputPath: filepath.Join(config.OutputDir, config.FilePrefix+"_block.prof"),
		}
		profiler.blockProfiler = NewBlockProfiler(blockConfig)
	}
	
	return profiler, nil
}

// Start는 프로파일링을 시작합니다
func (pp *PerformanceProfiler) Start() error {
	if !pp.running.CompareAndSwap(false, true) {
		return fmt.Errorf("profiler is already running")
	}
	
	// 개별 프로파일러들 시작
	if pp.cpuProfiler != nil {
		if err := pp.cpuProfiler.Start(); err != nil {
			return fmt.Errorf("failed to start CPU profiler: %w", err)
		}
	}
	
	if pp.memoryProfiler != nil {
		if err := pp.memoryProfiler.Start(); err != nil {
			return fmt.Errorf("failed to start memory profiler: %w", err)
		}
	}
	
	if pp.goroutineProfiler != nil {
		if err := pp.goroutineProfiler.Start(); err != nil {
			return fmt.Errorf("failed to start goroutine profiler: %w", err)
		}
	}
	
	if pp.blockProfiler != nil {
		if err := pp.blockProfiler.Start(); err != nil {
			return fmt.Errorf("failed to start block profiler: %w", err)
		}
	}
	
	// 백그라운드 작업들 시작
	if pp.config.AutoCapture {
		pp.wg.Add(1)
		go pp.autoCapture()
	}
	
	if pp.config.AutoCleanup {
		pp.wg.Add(1)
		go pp.autoCleanup()
	}
	
	return nil
}

// Stop은 프로파일링을 중지합니다
func (pp *PerformanceProfiler) Stop() error {
	if !pp.running.CompareAndSwap(true, false) {
		return nil
	}
	
	// 컨텍스트 취소
	pp.cancel()
	
	// 백그라운드 작업 완료 대기
	pp.wg.Wait()
	
	// 개별 프로파일러들 중지
	if pp.cpuProfiler != nil {
		pp.cpuProfiler.Stop()
	}
	
	if pp.memoryProfiler != nil {
		pp.memoryProfiler.Stop()
	}
	
	if pp.goroutineProfiler != nil {
		pp.goroutineProfiler.Stop()
	}
	
	if pp.blockProfiler != nil {
		pp.blockProfiler.Stop()
	}
	
	return nil
}

// Capture는 현재 시점의 프로파일을 캡처합니다
func (pp *PerformanceProfiler) Capture() (*ProfilingReport, error) {
	if !pp.running.Load() {
		return nil, fmt.Errorf("profiler is not running")
	}
	
	timestamp := time.Now()
	report := &ProfilingReport{
		Timestamp: timestamp,
		Files:     make([]string, 0),
	}
	
	// CPU 프로파일 캡처
	if pp.cpuProfiler != nil {
		if err := pp.cpuProfiler.Capture(); err == nil {
			report.CPUStats = pp.cpuProfiler.GetStats()
			report.Files = append(report.Files, pp.cpuProfiler.config.OutputPath)
		}
	}
	
	// 메모리 프로파일 캡처
	if pp.memoryProfiler != nil {
		if err := pp.memoryProfiler.Capture(); err == nil {
			report.MemoryStats = pp.memoryProfiler.GetStats()
			report.Files = append(report.Files, pp.memoryProfiler.config.OutputPath)
		}
	}
	
	// 고루틴 프로파일 캡처
	if pp.goroutineProfiler != nil {
		if err := pp.goroutineProfiler.Capture(); err == nil {
			report.GoroutineStats = pp.goroutineProfiler.GetStats()
			report.Files = append(report.Files, pp.goroutineProfiler.config.OutputPath)
		}
	}
	
	// 블록 프로파일 캡처
	if pp.blockProfiler != nil {
		if err := pp.blockProfiler.Capture(); err == nil {
			report.BlockStats = pp.blockProfiler.GetStats()
			report.Files = append(report.Files, pp.blockProfiler.config.OutputPath)
		}
	}
	
	// 리포트 분석
	if pp.config.AutoAnalysis {
		report.Summary = pp.analyzeReport(report)
	}
	
	report.Duration = time.Since(timestamp)
	
	return report, nil
}

// GetCurrentStats는 현재 통계를 반환합니다
func (pp *PerformanceProfiler) GetCurrentStats() map[string]interface{} {
	stats := make(map[string]interface{})
	
	if pp.cpuProfiler != nil {
		stats["cpu"] = pp.cpuProfiler.GetStats()
	}
	
	if pp.memoryProfiler != nil {
		stats["memory"] = pp.memoryProfiler.GetStats()
	}
	
	if pp.goroutineProfiler != nil {
		stats["goroutine"] = pp.goroutineProfiler.GetStats()
	}
	
	if pp.blockProfiler != nil {
		stats["block"] = pp.blockProfiler.GetStats()
	}
	
	return stats
}

// 내부 메서드들

func (pp *PerformanceProfiler) autoCapture() {
	defer pp.wg.Done()
	
	ticker := time.NewTicker(pp.config.CollectionInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-pp.ctx.Done():
			return
		case <-ticker.C:
			if report, err := pp.Capture(); err == nil {
				pp.checkThresholds(report)
			}
		}
	}
}

func (pp *PerformanceProfiler) autoCleanup() {
	defer pp.wg.Done()
	
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	
	for {
		select {
		case <-pp.ctx.Done():
			return
		case <-ticker.C:
			pp.cleanupOldFiles()
		}
	}
}

func (pp *PerformanceProfiler) checkThresholds(report *ProfilingReport) {
	// 메모리 임계값 확인
	if report.MemoryStats.HeapAlloc > uint64(pp.config.MemoryThreshold) {
		fmt.Printf("Warning: Memory usage exceeded threshold: %d bytes\n", report.MemoryStats.HeapAlloc)
	}
	
	// 고루틴 임계값 확인
	if report.GoroutineStats.Count > pp.config.GoroutineThreshold {
		fmt.Printf("Warning: Goroutine count exceeded threshold: %d\n", report.GoroutineStats.Count)
	}
	
	// CPU 임계값 확인
	if report.CPUStats.AverageCPU > pp.config.CPUThreshold {
		fmt.Printf("Warning: CPU usage exceeded threshold: %.2f%%\n", report.CPUStats.AverageCPU)
	}
}

func (pp *PerformanceProfiler) cleanupOldFiles() {
	cutoff := time.Now().Add(-pp.config.RetentionPeriod)
	
	filepath.Walk(pp.config.OutputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		
		if !info.IsDir() && info.ModTime().Before(cutoff) {
			os.Remove(path)
		}
		
		return nil
	})
}

func (pp *PerformanceProfiler) analyzeReport(report *ProfilingReport) ReportSummary {
	summary := ReportSummary{
		Recommendations: make([]string, 0),
		Warnings:        make([]string, 0),
		Errors:          make([]string, 0),
	}
	
	// 메모리 효율성 계산
	if report.MemoryStats.HeapAlloc > 0 {
		summary.MemoryEfficiency = float64(report.MemoryStats.HeapInuse) / float64(report.MemoryStats.HeapAlloc)
	}
	
	// CPU 효율성 계산
	summary.CPUEfficiency = 100.0 - report.CPUStats.AverageCPU
	
	// 성능 점수 계산
	summary.PerformanceScore = (summary.MemoryEfficiency + summary.CPUEfficiency/100.0) / 2.0
	
	// 권고사항 생성
	if report.GoroutineStats.Count > 500 {
		summary.Recommendations = append(summary.Recommendations, "Consider reducing the number of goroutines")
	}
	
	if report.MemoryStats.GCCPUFraction > 0.1 {
		summary.Recommendations = append(summary.Recommendations, "High GC overhead detected, consider memory optimization")
	}
	
	if report.CPUStats.AverageCPU > 70.0 {
		summary.Warnings = append(summary.Warnings, "High CPU usage detected")
	}
	
	return summary
}

// CPU 프로파일러 구현

// NewCPUProfiler는 새로운 CPU 프로파일러를 생성합니다
func NewCPUProfiler(config CPUProfilerConfig) *CPUProfiler {
	return &CPUProfiler{
		config: config,
	}
}

// Start는 CPU 프로파일링을 시작합니다
func (cp *CPUProfiler) Start() error {
	if !cp.isRunning.CompareAndSwap(false, true) {
		return fmt.Errorf("CPU profiler is already running")
	}
	
	runtime.SetCPUProfileRate(cp.config.SampleRate)
	return nil
}

// Stop은 CPU 프로파일링을 중지합니다
func (cp *CPUProfiler) Stop() error {
	if !cp.isRunning.CompareAndSwap(true, false) {
		return nil
	}
	
	if cp.outputFile != nil {
		pprof.StopCPUProfile()
		cp.outputFile.Close()
		cp.outputFile = nil
	}
	
	return nil
}

// Capture는 CPU 프로파일을 캡처합니다
func (cp *CPUProfiler) Capture() error {
	cp.mutex.Lock()
	defer cp.mutex.Unlock()
	
	// 기존 파일 닫기
	if cp.outputFile != nil {
		pprof.StopCPUProfile()
		cp.outputFile.Close()
	}
	
	// 새 파일 생성
	file, err := os.Create(cp.config.OutputPath)
	if err != nil {
		return fmt.Errorf("failed to create CPU profile file: %w", err)
	}
	
	cp.outputFile = file
	
	// 프로파일링 시작
	if err := pprof.StartCPUProfile(file); err != nil {
		file.Close()
		return fmt.Errorf("failed to start CPU profile: %w", err)
	}
	
	// 지정된 시간 후 자동 중지
	go func() {
		time.Sleep(cp.config.Duration)
		cp.mutex.Lock()
		if cp.outputFile != nil {
			pprof.StopCPUProfile()
			cp.outputFile.Close()
			cp.outputFile = nil
		}
		cp.mutex.Unlock()
	}()
	
	// 통계 업데이트
	cp.statsMutex.Lock()
	cp.stats.SampleCount++
	cp.stats.LastCollection = time.Now()
	cp.statsMutex.Unlock()
	
	return nil
}

// GetStats는 CPU 통계를 반환합니다
func (cp *CPUProfiler) GetStats() CPUStats {
	cp.statsMutex.RLock()
	defer cp.statsMutex.RUnlock()
	
	return cp.stats
}

// 메모리 프로파일러 구현

// NewMemoryProfiler는 새로운 메모리 프로파일러를 생성합니다
func NewMemoryProfiler(config MemoryProfilerConfig) *MemoryProfiler {
	return &MemoryProfiler{
		config: config,
	}
}

// Start는 메모리 프로파일링을 시작합니다
func (mp *MemoryProfiler) Start() error {
	return nil
}

// Stop은 메모리 프로파일링을 중지합니다
func (mp *MemoryProfiler) Stop() error {
	return nil
}

// Capture는 메모리 프로파일을 캡처합니다
func (mp *MemoryProfiler) Capture() error {
	// 메모리 통계 수집
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	mp.statsMutex.Lock()
	mp.stats.HeapAlloc = memStats.HeapAlloc
	mp.stats.HeapSys = memStats.HeapSys
	mp.stats.HeapInuse = memStats.HeapInuse
	mp.stats.HeapReleased = memStats.HeapReleased
	mp.stats.StackInuse = memStats.StackInuse
	mp.stats.StackSys = memStats.StackSys
	mp.stats.MSpanInuse = memStats.MSpanInuse
	mp.stats.MSpanSys = memStats.MSpanSys
	mp.stats.MCacheInuse = memStats.MCacheInuse
	mp.stats.MCacheSys = memStats.MCacheSys
	mp.stats.GCSys = memStats.GCSys
	mp.stats.NextGC = memStats.NextGC
	mp.stats.LastGC = memStats.LastGC
	mp.stats.NumGC = memStats.NumGC
	mp.stats.NumForcedGC = memStats.NumForcedGC
	mp.stats.GCCPUFraction = memStats.GCCPUFraction
	mp.stats.LastCollection = time.Now()
	mp.statsMutex.Unlock()
	
	// 프로파일 파일 생성
	file, err := os.Create(mp.config.OutputPath)
	if err != nil {
		return fmt.Errorf("failed to create memory profile file: %w", err)
	}
	defer file.Close()
	
	runtime.GC() // GC 실행 후 힙 프로파일 생성
	
	if err := pprof.WriteHeapProfile(file); err != nil {
		return fmt.Errorf("failed to write heap profile: %w", err)
	}
	
	return nil
}

// GetStats는 메모리 통계를 반환합니다
func (mp *MemoryProfiler) GetStats() MemoryStats {
	mp.statsMutex.RLock()
	defer mp.statsMutex.RUnlock()
	
	return mp.stats
}

// 고루틴 프로파일러 구현

// NewGoroutineProfiler는 새로운 고루틴 프로파일러를 생성합니다
func NewGoroutineProfiler(config GoroutineProfilerConfig) *GoroutineProfiler {
	return &GoroutineProfiler{
		config: config,
	}
}

// Start는 고루틴 프로파일링을 시작합니다
func (gp *GoroutineProfiler) Start() error {
	return nil
}

// Stop은 고루틴 프로파일링을 중지합니다
func (gp *GoroutineProfiler) Stop() error {
	return nil
}

// Capture는 고루틴 프로파일을 캡처합니다
func (gp *GoroutineProfiler) Capture() error {
	count := runtime.NumGoroutine()
	
	gp.statsMutex.Lock()
	gp.stats.Count = count
	if count > gp.stats.PeakCount {
		gp.stats.PeakCount = count
	}
	gp.stats.AverageCount = (gp.stats.AverageCount + float64(count)) / 2.0
	gp.stats.LastCollection = time.Now()
	gp.statsMutex.Unlock()
	
	// 프로파일 파일 생성
	file, err := os.Create(gp.config.OutputPath)
	if err != nil {
		return fmt.Errorf("failed to create goroutine profile file: %w", err)
	}
	defer file.Close()
	
	if err := pprof.Lookup("goroutine").WriteTo(file, 1); err != nil {
		return fmt.Errorf("failed to write goroutine profile: %w", err)
	}
	
	return nil
}

// GetStats는 고루틴 통계를 반환합니다
func (gp *GoroutineProfiler) GetStats() GoroutineStats {
	gp.statsMutex.RLock()
	defer gp.statsMutex.RUnlock()
	
	return gp.stats
}

// 블록 프로파일러 구현

// NewBlockProfiler는 새로운 블록 프로파일러를 생성합니다
func NewBlockProfiler(config BlockProfilerConfig) *BlockProfiler {
	return &BlockProfiler{
		config: config,
	}
}

// Start는 블록 프로파일링을 시작합니다
func (bp *BlockProfiler) Start() error {
	runtime.SetBlockProfileRate(bp.config.Rate)
	return nil
}

// Stop은 블록 프로파일링을 중지합니다
func (bp *BlockProfiler) Stop() error {
	runtime.SetBlockProfileRate(0)
	return nil
}

// Capture는 블록 프로파일을 캡처합니다
func (bp *BlockProfiler) Capture() error {
	// 프로파일 파일 생성
	file, err := os.Create(bp.config.OutputPath)
	if err != nil {
		return fmt.Errorf("failed to create block profile file: %w", err)
	}
	defer file.Close()
	
	if err := pprof.Lookup("block").WriteTo(file, 0); err != nil {
		return fmt.Errorf("failed to write block profile: %w", err)
	}
	
	// 통계 업데이트
	bp.statsMutex.Lock()
	bp.stats.LastCollection = time.Now()
	bp.statsMutex.Unlock()
	
	return nil
}

// GetStats는 블록 통계를 반환합니다
func (bp *BlockProfiler) GetStats() BlockStats {
	bp.statsMutex.RLock()
	defer bp.statsMutex.RUnlock()
	
	return bp.stats
}