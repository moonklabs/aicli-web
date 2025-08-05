package agent

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

// sync와 atomic 패키지 사용을 명시적으로 보장
var _ = sync.RWMutex{}
var _ = atomic.Value{}

// NewAgentProfiler는 새로운 에이전트 프로파일러를 생성합니다
func NewAgentProfiler(interval time.Duration) *AgentProfiler {
	ctx, cancel := context.WithCancel(context.Background())

	return &AgentProfiler{
		cpuProfile:        make([]CPUProfile, 0, 1000),
		memoryProfile:     make([]MemoryProfile, 0, 1000),
		goroutineProfile:  make([]GoroutineProfile, 0, 1000),
		profilingEnabled:  true,
		profilingInterval: interval,
		profileRetention:  24 * time.Hour,
		ctx:               ctx,
		cancel:            cancel,
	}
}

// Start는 프로파일러를 시작합니다
func (ap *AgentProfiler) Start() error {
	if !ap.profilingEnabled {
		return nil
	}

	go ap.profilingLoop()
	return nil
}

// Stop은 프로파일러를 중지합니다
func (ap *AgentProfiler) Stop() error {
	ap.cancel()
	return nil
}

// CollectProfile은 프로파일 데이터를 수집합니다
func (ap *AgentProfiler) CollectProfile() {
	if !ap.profilingEnabled {
		return
	}

	now := time.Now()

	// CPU 프로파일 수집
	ap.collectCPUProfile(now)

	// 메모리 프로파일 수집
	ap.collectMemoryProfile(now)

	// 고루틴 프로파일 수집
	ap.collectGoroutineProfile(now)

	// 오래된 프로파일 데이터 정리
	ap.cleanupOldProfiles(now)
}

// GetCPUProfile은 CPU 프로파일을 반환합니다
func (ap *AgentProfiler) GetCPUProfile(duration time.Duration) []CPUProfile {
	ap.mutex.RLock()
	defer ap.mutex.RUnlock()

	cutoff := time.Now().Add(-duration)
	profiles := make([]CPUProfile, 0)

	for _, profile := range ap.cpuProfile {
		if profile.Timestamp.After(cutoff) {
			profiles = append(profiles, profile)
		}
	}

	return profiles
}

// GetMemoryProfile은 메모리 프로파일을 반환합니다
func (ap *AgentProfiler) GetMemoryProfile(duration time.Duration) []MemoryProfile {
	ap.mutex.RLock()
	defer ap.mutex.RUnlock()

	cutoff := time.Now().Add(-duration)
	profiles := make([]MemoryProfile, 0)

	for _, profile := range ap.memoryProfile {
		if profile.Timestamp.After(cutoff) {
			profiles = append(profiles, profile)
		}
	}

	return profiles
}

// GetGoroutineProfile은 고루틴 프로파일을 반환합니다
func (ap *AgentProfiler) GetGoroutineProfile(duration time.Duration) []GoroutineProfile {
	ap.mutex.RLock()
	defer ap.mutex.RUnlock()

	cutoff := time.Now().Add(-duration)
	profiles := make([]GoroutineProfile, 0)

	for _, profile := range ap.goroutineProfile {
		if profile.Timestamp.After(cutoff) {
			profiles = append(profiles, profile)
		}
	}

	return profiles
}

func (ap *AgentProfiler) profilingLoop() {
	ticker := time.NewTicker(ap.profilingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ap.ctx.Done():
			return
		case <-ticker.C:
			ap.CollectProfile()
		}
	}
}

func (ap *AgentProfiler) collectCPUProfile(timestamp time.Time) {
	ap.mutex.Lock()
	defer ap.mutex.Unlock()

	// CPU 사용률 수집
	cpuPercent, err := cpu.Percent(100*time.Millisecond, false)
	if err != nil || len(cpuPercent) == 0 {
		return
	}

	// 프로세스 정보 수집
	processes := ap.collectProcessInfo()

	profile := CPUProfile{
		Timestamp: timestamp,
		Usage:     cpuPercent[0],
		Processes: processes,
	}

	ap.cpuProfile = append(ap.cpuProfile, profile)

	// 최대 개수 제한
	if len(ap.cpuProfile) > 1000 {
		ap.cpuProfile = ap.cpuProfile[1:]
	}
}

func (ap *AgentProfiler) collectMemoryProfile(timestamp time.Time) {
	ap.mutex.Lock()
	defer ap.mutex.Unlock()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// 할당 속도 계산
	var allocRate float64
	if len(ap.memoryProfile) > 0 {
		lastProfile := ap.memoryProfile[len(ap.memoryProfile)-1]
		timeDiff := timestamp.Sub(lastProfile.Timestamp).Seconds()
		if timeDiff > 0 {
			allocDiff := float64(m.TotalAlloc - lastProfile.HeapAlloc)
			allocRate = allocDiff / timeDiff
		}
	}

	profile := MemoryProfile{
		Timestamp:    timestamp,
		HeapAlloc:    m.Alloc,
		HeapSys:      m.HeapSys,
		HeapInuse:    m.HeapInuse,
		StackInuse:   m.StackInuse,
		NumGC:        m.NumGC,
		PauseTotalNs: m.PauseTotalNs,
		AllocRate:    allocRate,
	}

	ap.memoryProfile = append(ap.memoryProfile, profile)

	// 최대 개수 제한
	if len(ap.memoryProfile) > 1000 {
		ap.memoryProfile = ap.memoryProfile[1:]
	}
}

func (ap *AgentProfiler) collectGoroutineProfile(timestamp time.Time) {
	ap.mutex.Lock()
	defer ap.mutex.Unlock()

	numGoroutine := runtime.NumGoroutine()

	// 고루틴 상태별 분류 (실제 구현에서는 runtime 패키지의 고급 기능 사용)
	goroutinesByState := map[string]int{
		"running": numGoroutine / 4,     // 추정치
		"waiting": numGoroutine * 3 / 4, // 추정치
		"blocked": 0,                    // 추정치
	}

	profile := GoroutineProfile{
		Timestamp:         timestamp,
		NumGoroutine:      numGoroutine,
		GoroutinesByState: goroutinesByState,
	}

	ap.goroutineProfile = append(ap.goroutineProfile, profile)

	// 최대 개수 제한
	if len(ap.goroutineProfile) > 1000 {
		ap.goroutineProfile = ap.goroutineProfile[1:]
	}
}

func (ap *AgentProfiler) collectProcessInfo() []ProcessInfo {
	// 간단한 프로세스 정보 수집 (실제로는 더 정교한 구현 필요)
	return []ProcessInfo{
		{
			PID:           int32(1),
			Name:          "aicli-agent",
			CPUPercent:    10.0,
			MemoryPercent: 5.0,
		},
	}
}

func (ap *AgentProfiler) cleanupOldProfiles(now time.Time) {
	cutoff := now.Add(-ap.profileRetention)

	// CPU 프로파일 정리
	newCPUProfiles := make([]CPUProfile, 0, len(ap.cpuProfile))
	for _, profile := range ap.cpuProfile {
		if profile.Timestamp.After(cutoff) {
			newCPUProfiles = append(newCPUProfiles, profile)
		}
	}
	ap.cpuProfile = newCPUProfiles

	// 메모리 프로파일 정리
	newMemoryProfiles := make([]MemoryProfile, 0, len(ap.memoryProfile))
	for _, profile := range ap.memoryProfile {
		if profile.Timestamp.After(cutoff) {
			newMemoryProfiles = append(newMemoryProfiles, profile)
		}
	}
	ap.memoryProfile = newMemoryProfiles

	// 고루틴 프로파일 정리
	newGoroutineProfiles := make([]GoroutineProfile, 0, len(ap.goroutineProfile))
	for _, profile := range ap.goroutineProfile {
		if profile.Timestamp.After(cutoff) {
			newGoroutineProfiles = append(newGoroutineProfiles, profile)
		}
	}
	ap.goroutineProfile = newGoroutineProfiles
}

// NewSystemResourceMonitor는 새로운 시스템 리소스 모니터를 생성합니다
func NewSystemResourceMonitor(interval time.Duration) *SystemResourceMonitor {
	ctx, cancel := context.WithCancel(context.Background())

	monitor := &SystemResourceMonitor{
		monitoringInterval: interval,
		alertThresholds: AlertThresholds{
			HighCPUUsage:    80.0,
			HighMemoryUsage: 80.0,
			HighDiskUsage:   80.0,
			HighErrorRate:   5.0,
			HighLatency:     time.Second,
			LowThroughput:   10.0,
		},
		ctx:          ctx,
		cancel:       cancel,
		alertChannel: make(chan PerformanceAlert, 100),
	}

	// 초기값 설정
	monitor.currentCPU.Store(0.0)
	monitor.currentMemory.Store(&mem.VirtualMemoryStat{})
	monitor.currentDisk.Store(int64(0))

	return monitor
}

// Start는 시스템 리소스 모니터를 시작합니다
func (srm *SystemResourceMonitor) Start() error {
	go srm.monitoringLoop()
	return nil
}

// Stop은 시스템 리소스 모니터를 중지합니다
func (srm *SystemResourceMonitor) Stop() error {
	srm.cancel()
	close(srm.alertChannel)
	return nil
}

// GetCPUUsage는 현재 CPU 사용률을 반환합니다
func (srm *SystemResourceMonitor) GetCPUUsage() float64 {
	return srm.currentCPU.Load().(float64)
}

// GetMemoryUsage는 현재 메모리 사용률을 반환합니다
func (srm *SystemResourceMonitor) GetMemoryUsage() float64 {
	memStat := srm.currentMemory.Load().(*mem.VirtualMemoryStat)
	return memStat.UsedPercent
}

// GetDiskUsage는 현재 디스크 사용량을 반환합니다
func (srm *SystemResourceMonitor) GetDiskUsage() float64 {
	diskUsage := srm.currentDisk.Load().(int64)
	maxDisk := int64(100 * 1024 * 1024 * 1024) // 100GB 가정
	return float64(diskUsage) / float64(maxDisk) * 100
}

func (srm *SystemResourceMonitor) monitoringLoop() {
	ticker := time.NewTicker(srm.monitoringInterval)
	defer ticker.Stop()

	for {
		select {
		case <-srm.ctx.Done():
			return
		case <-ticker.C:
			srm.collectSystemMetrics()
		}
	}
}

func (srm *SystemResourceMonitor) collectSystemMetrics() {
	// CPU 사용률 수집
	if cpuPercent, err := cpu.Percent(time.Second, false); err == nil && len(cpuPercent) > 0 {
		srm.currentCPU.Store(cpuPercent[0])

		// CPU 알림 체크
		if cpuPercent[0] > srm.alertThresholds.HighCPUUsage {
			srm.sendAlert(PerformanceAlert{
				Type:      AlertTypeHighCPU,
				Severity:  AlertSeverityWarning,
				Message:   "High CPU usage detected",
				Timestamp: time.Now(),
				Metrics: map[string]float64{
					"cpu_usage": cpuPercent[0],
				},
				RecommendedActions: []string{
					"Scale up agents",
					"Optimize agent workload",
					"Check for resource leaks",
				},
			})
		}
	}

	// 메모리 사용률 수집
	if memStat, err := mem.VirtualMemory(); err == nil {
		srm.currentMemory.Store(memStat)

		// 메모리 알림 체크
		if memStat.UsedPercent > srm.alertThresholds.HighMemoryUsage {
			srm.sendAlert(PerformanceAlert{
				Type:      AlertTypeHighMemory,
				Severity:  AlertSeverityWarning,
				Message:   "High memory usage detected",
				Timestamp: time.Now(),
				Metrics: map[string]float64{
					"memory_usage_percent": memStat.UsedPercent,
					"memory_usage_bytes":   float64(memStat.Used),
				},
				RecommendedActions: []string{
					"Run garbage collection",
					"Scale down idle agents",
					"Optimize memory pools",
				},
			})
		}
	}
}

func (srm *SystemResourceMonitor) sendAlert(alert PerformanceAlert) {
	select {
	case srm.alertChannel <- alert:
	default:
		// 알림 채널이 가득찬 경우 무시 (백프레셔 방지)
	}
}

// NewAgentAutoScaler는 새로운 에이전트 자동 스케일러를 생성합니다
func NewAgentAutoScaler(config AutoScalingConfig) *AgentAutoScaler {
	ctx, cancel := context.WithCancel(context.Background())

	scaler := &AgentAutoScaler{
		config:          config,
		ctx:             ctx,
		cancel:          cancel,
		loadPredictor:   NewLoadPredictor(),
		capacityPlanner: NewCapacityPlanner(),
	}

	scaler.currentCapacity.Store(int32(config.MinAgents))
	scaler.targetCapacity.Store(int32(config.MinAgents))
	scaler.lastScaleAction.Store("initialized")
	scaler.lastScaleTime.Store(time.Now())

	return scaler
}

// Start는 자동 스케일러를 시작합니다
func (aas *AgentAutoScaler) Start() error {
	if !aas.config.Enabled {
		return nil
	}

	go aas.scalingLoop()
	return nil
}

// Stop은 자동 스케일러를 중지합니다
func (aas *AgentAutoScaler) Stop() error {
	aas.cancel()
	return nil
}

// EvaluateScaling은 스케일링 필요성을 평가합니다
func (aas *AgentAutoScaler) EvaluateScaling(systemStatus *SystemStatus) error {
	if !aas.config.Enabled {
		return nil
	}

	aas.scalingMutex.Lock()
	defer aas.scalingMutex.Unlock()

	currentCapacity := int(aas.currentCapacity.Load())

	// 부하 기반 스케일링 결정
	targetCapacity := aas.calculateTargetCapacity(systemStatus)

	if targetCapacity > currentCapacity {
		return aas.scaleUp(targetCapacity - currentCapacity)
	} else if targetCapacity < currentCapacity {
		return aas.scaleDown(currentCapacity - targetCapacity)
	}

	return nil
}

// ScaleUp은 에이전트를 스케일 업합니다
func (aas *AgentAutoScaler) ScaleUp() error {
	return aas.scaleUp(1)
}

// ScaleDown은 에이전트를 스케일 다운합니다
func (aas *AgentAutoScaler) ScaleDown() error {
	return aas.scaleDown(1)
}

// GetLastAction은 마지막 스케일링 액션을 반환합니다
func (aas *AgentAutoScaler) GetLastAction() string {
	return aas.lastScaleAction.Load().(string)
}

func (aas *AgentAutoScaler) scalingLoop() {
	ticker := time.NewTicker(30 * time.Second) // 30초마다 스케일링 평가
	defer ticker.Stop()

	for {
		select {
		case <-aas.ctx.Done():
			return
		case <-ticker.C:
			aas.evaluateAutomaticScaling()
		}
	}
}

func (aas *AgentAutoScaler) evaluateAutomaticScaling() {
	// 예측 기반 스케일링 (실제 구현에서는 더 정교한 로직 필요)
	if aas.config.PredictiveScaling {
		prediction := aas.loadPredictor.PredictLoad(time.Now().Add(5 * time.Minute))
		if prediction.ExpectedLoad > 0.8 {
			aas.scaleUp(1)
		} else if prediction.ExpectedLoad < 0.3 {
			aas.scaleDown(1)
		}
	}
}

func (aas *AgentAutoScaler) calculateTargetCapacity(systemStatus *SystemStatus) int {
	// 간단한 부하 기반 용량 계산
	cpuLoad := systemStatus.CPUUsage / 100.0
	memoryLoad := systemStatus.MemoryUsage / 100.0

	// 최대 부하를 기준으로 용량 계산
	maxLoad := cpuLoad
	if memoryLoad > maxLoad {
		maxLoad = memoryLoad
	}

	currentCapacity := int(aas.currentCapacity.Load())

	if maxLoad > aas.config.ScaleUpThreshold {
		// 스케일 업 필요
		factor := maxLoad / aas.config.TargetUtilization
		targetCapacity := int(float64(currentCapacity) * factor)

		if targetCapacity > aas.config.MaxAgents {
			targetCapacity = aas.config.MaxAgents
		}

		return targetCapacity
	} else if maxLoad < aas.config.ScaleDownThreshold {
		// 스케일 다운 가능
		factor := maxLoad / aas.config.TargetUtilization
		targetCapacity := int(float64(currentCapacity) * factor)

		if targetCapacity < aas.config.MinAgents {
			targetCapacity = aas.config.MinAgents
		}

		return targetCapacity
	}

	return currentCapacity
}

func (aas *AgentAutoScaler) scaleUp(count int) error {
	lastScaleTime := aas.lastScaleTime.Load().(time.Time)
	if time.Since(lastScaleTime) < aas.config.ScaleUpCooldown {
		return nil // 쿨다운 중
	}

	currentCapacity := int(aas.currentCapacity.Load())
	newCapacity := currentCapacity + count

	if newCapacity > aas.config.MaxAgents {
		newCapacity = aas.config.MaxAgents
		count = newCapacity - currentCapacity
	}

	if count <= 0 {
		return nil
	}

	// 실제 스케일 업 로직 (에이전트 생성)
	// 여기서는 상태만 업데이트
	aas.currentCapacity.Store(int32(newCapacity))
	aas.targetCapacity.Store(int32(newCapacity))
	aas.lastScaleAction.Store(fmt.Sprintf("scale_up_%d", count))
	aas.lastScaleTime.Store(time.Now())

	return nil
}

func (aas *AgentAutoScaler) scaleDown(count int) error {
	lastScaleTime := aas.lastScaleTime.Load().(time.Time)
	if time.Since(lastScaleTime) < aas.config.ScaleDownCooldown {
		return nil // 쿨다운 중
	}

	currentCapacity := int(aas.currentCapacity.Load())
	newCapacity := currentCapacity - count

	if newCapacity < aas.config.MinAgents {
		newCapacity = aas.config.MinAgents
		count = currentCapacity - newCapacity
	}

	if count <= 0 {
		return nil
	}

	// 실제 스케일 다운 로직 (에이전트 제거)
	// 여기서는 상태만 업데이트
	aas.currentCapacity.Store(int32(newCapacity))
	aas.targetCapacity.Store(int32(newCapacity))
	aas.lastScaleAction.Store(fmt.Sprintf("scale_down_%d", count))
	aas.lastScaleTime.Store(time.Now())

	return nil
}

// NewLoadPredictor는 새로운 부하 예측기를 생성합니다
func NewLoadPredictor() *LoadPredictor {
	return &LoadPredictor{
		loadHistory:    make([]LoadDataPoint, 0, 1000),
		maxHistorySize: 1000,
		dailyPatterns:  make(map[int]float64),
		weeklyPatterns: make(map[int]float64),
		model: PredictionModel{
			Type:             ModelTypeMovingAverage,
			Parameters:       make(map[string]float64),
			LastTrained:      time.Now(),
			TrainingAccuracy: 0.8,
		},
		accuracy: 0.8,
	}
}

// PredictLoad는 지정된 시간의 부하를 예측합니다
func (lp *LoadPredictor) PredictLoad(targetTime time.Time) LoadPrediction {
	lp.mutex.RLock()
	defer lp.mutex.RUnlock()

	// 간단한 이동평균 기반 예측
	if len(lp.loadHistory) == 0 {
		return LoadPrediction{
			Timestamp:    targetTime,
			ExpectedLoad: 0.5, // 기본값
			Confidence:   0.5,
		}
	}

	// 최근 데이터 기반 평균 계산
	recentCount := 10
	if len(lp.loadHistory) < recentCount {
		recentCount = len(lp.loadHistory)
	}

	var totalLoad float64
	for i := len(lp.loadHistory) - recentCount; i < len(lp.loadHistory); i++ {
		totalLoad += lp.loadHistory[i].CPUUsage
	}

	avgLoad := totalLoad / float64(recentCount)

	// 시간 기반 패턴 적용
	hour := targetTime.Hour()
	if pattern, exists := lp.dailyPatterns[hour]; exists {
		avgLoad = avgLoad * pattern
	}

	return LoadPrediction{
		Timestamp:    targetTime,
		ExpectedLoad: avgLoad / 100.0, // 백분율을 비율로 변환
		Confidence:   lp.accuracy,
	}
}

// LoadPrediction은 부하 예측 결과입니다
type LoadPrediction struct {
	Timestamp    time.Time `json:"timestamp"`
	ExpectedLoad float64   `json:"expected_load"`
	Confidence   float64   `json:"confidence"`
}

// RecordLoad는 부하 데이터를 기록합니다
func (lp *LoadPredictor) RecordLoad(dataPoint LoadDataPoint) {
	lp.mutex.Lock()
	defer lp.mutex.Unlock()

	lp.loadHistory = append(lp.loadHistory, dataPoint)

	// 최대 크기 제한
	if len(lp.loadHistory) > lp.maxHistorySize {
		lp.loadHistory = lp.loadHistory[1:]
	}

	// 패턴 업데이트
	lp.updatePatterns(dataPoint)
}

func (lp *LoadPredictor) updatePatterns(dataPoint LoadDataPoint) {
	hour := dataPoint.Timestamp.Hour()
	weekday := int(dataPoint.Timestamp.Weekday())

	// 일일 패턴 업데이트 (지수 이동평균)
	alpha := 0.1
	if existing, exists := lp.dailyPatterns[hour]; exists {
		lp.dailyPatterns[hour] = alpha*dataPoint.CPUUsage + (1-alpha)*existing
	} else {
		lp.dailyPatterns[hour] = dataPoint.CPUUsage
	}

	// 주간 패턴 업데이트
	if existing, exists := lp.weeklyPatterns[weekday]; exists {
		lp.weeklyPatterns[weekday] = alpha*dataPoint.CPUUsage + (1-alpha)*existing
	} else {
		lp.weeklyPatterns[weekday] = dataPoint.CPUUsage
	}
}

// NewCapacityPlanner는 새로운 용량 계획기를 생성합니다
func NewCapacityPlanner() *CapacityPlanner {
	return &CapacityPlanner{
		currentPlan:   CapacityPlan{},
		futurePlans:   make([]CapacityPlan, 0),
		resourceModel: DefaultResourceModel(),
	}
}

// DefaultResourceModel은 기본 리소스 모델을 반환합니다
func DefaultResourceModel() ResourceModel {
	return ResourceModel{
		CPUPerAgent:     0.1,               // 0.1 코어
		MemoryPerAgent:  100 * 1024 * 1024, // 100MB
		DiskPerAgent:    10 * 1024 * 1024,  // 10MB
		NetworkPerAgent: 1024,              // 1KB/s
		SystemOverhead: ResourceRequirements{
			CPU:     0.5,                // 0.5 코어
			Memory:  512 * 1024 * 1024,  // 512MB
			Disk:    1024 * 1024 * 1024, // 1GB
			Network: 10 * 1024,          // 10KB/s
		},
		ScalingOverhead: ResourceRequirements{
			CPU:     0.05,             // 0.05 코어 per scaling operation
			Memory:  10 * 1024 * 1024, // 10MB
			Disk:    1024 * 1024,      // 1MB
			Network: 1024,             // 1KB/s
		},
	}
}

// NewAgentLoadBalancer는 새로운 에이전트 로드 밸런서를 생성합니다
func NewAgentLoadBalancer() *AgentLoadBalancer {
	return &AgentLoadBalancer{
		strategy:       StrategyAdaptive,
		agentPools:     make(map[string]*AgentPool),
		routingTable:   NewRoutingTable(),
		healthChecker:  NewAgentHealthChecker(),
		routingMetrics: NewRoutingMetrics(),
	}
}

// NewRoutingTable은 새로운 라우팅 테이블을 생성합니다
func NewRoutingTable() *RoutingTable {
	return &RoutingTable{
		Rules:       make([]RoutingRule, 0),
		DefaultPool: "default",
	}
}

// NewAgentHealthChecker는 새로운 에이전트 건강 체커를 생성합니다
func NewAgentHealthChecker() *AgentHealthChecker {
	ctx, cancel := context.WithCancel(context.Background())

	return &AgentHealthChecker{
		checkInterval: 30 * time.Second,
		checkTimeout:  5 * time.Second,
		agentHealth:   make(map[string]*AgentHealth),
		ctx:           ctx,
		cancel:        cancel,
	}
}

// NewRoutingMetrics는 새로운 라우팅 메트릭을 생성합니다
func NewRoutingMetrics() *RoutingMetrics {
	return &RoutingMetrics{
		PoolMetrics: make(map[string]*AgentPoolMetrics),
	}
}

// NewPerformanceAlertManager는 새로운 성능 알림 관리자를 생성합니다
func NewPerformanceAlertManager() *PerformanceAlertManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &PerformanceAlertManager{
		alertChannel:   make(chan PerformanceAlert, 1000),
		alertRules:     make([]AlertRule, 0),
		alertHistory:   make([]PerformanceAlert, 0, 10000),
		maxHistorySize: 10000,
		ctx:            ctx,
		cancel:         cancel,
	}
}

// Start는 성능 알림 관리자를 시작합니다
func (pam *PerformanceAlertManager) Start() error {
	go pam.alertProcessingLoop()
	return nil
}

// Stop은 성능 알림 관리자를 중지합니다
func (pam *PerformanceAlertManager) Stop() error {
	pam.cancel()
	close(pam.alertChannel)
	return nil
}

func (pam *PerformanceAlertManager) alertProcessingLoop() {
	for {
		select {
		case <-pam.ctx.Done():
			return
		case alert := <-pam.alertChannel:
			pam.processAlert(alert)
		}
	}
}

func (pam *PerformanceAlertManager) processAlert(alert PerformanceAlert) {
	pam.mutex.Lock()
	defer pam.mutex.Unlock()

	// 알림 히스토리에 추가
	pam.alertHistory = append(pam.alertHistory, alert)

	// 최대 크기 제한
	if len(pam.alertHistory) > pam.maxHistorySize {
		pam.alertHistory = pam.alertHistory[1:]
	}

	// 알림 규칙 실행
	for _, rule := range pam.alertRules {
		if rule.Enabled && pam.matchesRule(alert, rule) {
			pam.executeAlertActions(alert, rule)
		}
	}
}

func (pam *PerformanceAlertManager) matchesRule(alert PerformanceAlert, rule AlertRule) bool {
	// 간단한 규칙 매칭 (실제로는 더 복잡한 조건 평가)
	return alert.Severity >= rule.Severity
}

func (pam *PerformanceAlertManager) executeAlertActions(alert PerformanceAlert, rule AlertRule) {
	for _, action := range rule.Actions {
		switch action.Type {
		case ActionTypeLog:
			// 로그 기록
		case ActionTypeEmail:
			// 이메일 발송
		case ActionTypeSlack:
			// Slack 알림
		case ActionTypeWebhook:
			// 웹훅 호출
		case ActionTypeScale:
			// 자동 스케일링 트리거
		}
	}
}
