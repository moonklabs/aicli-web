package claude

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// GoroutineManager는 고루틴 생명주기 관리 인터페이스입니다
type GoroutineManager interface {
	// 고루틴 풀 관리
	SpawnWorker(task Task) error
	SpawnBoundedWorker(task Task, timeout time.Duration) error
	
	// 리소스 추적
	GetActiveGoroutines() int
	GetGoroutineStats() GoroutineStats
	
	// 정리 및 최적화
	CleanupIdleWorkers() error
	SetMaxGoroutines(max int) error
	
	// 생명주기
	Start() error
	Stop() error
}

// WorkerPoolManager는 고루틴 풀 관리자입니다
type WorkerPoolManager struct {
	// 워커 풀
	workers      map[int]*Worker
	workersMutex sync.RWMutex
	nextWorkerID int32
	
	// 태스크 큐
	taskQueue    chan TaskWrapper
	priorityQueue chan TaskWrapper
	
	// 설정
	config WorkerPoolConfig
	
	// 스케일러
	scaler   *WorkerScaler
	monitor  *GoroutineMonitor
	
	// 통계
	stats    GoroutineStats
	statsMutex sync.RWMutex
	
	// 생명주기
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	running  atomic.Bool
}

// Task는 실행할 작업입니다
type Task interface {
	Execute(ctx context.Context) error
	GetPriority() TaskPriority
	GetEstimatedDuration() time.Duration
	GetDescription() string
}

// TaskWrapper는 태스크 래퍼입니다
type TaskWrapper struct {
	Task      Task
	StartTime time.Time
	Timeout   time.Duration
	ResultCh  chan TaskResult
	Ctx       context.Context
	Cancel    context.CancelFunc
}

// TaskResult는 태스크 실행 결과입니다
type TaskResult struct {
	Success   bool          `json:"success"`
	Error     error         `json:"error"`
	Duration  time.Duration `json:"duration"`
	StartTime time.Time     `json:"start_time"`
	EndTime   time.Time     `json:"end_time"`
}

// TaskPriority는 태스크 우선순위입니다
type TaskPriority int

const (
	TaskPriorityLow TaskPriority = iota
	TaskPriorityNormal
	TaskPriorityHigh
	TaskPriorityCritical
)

// Worker는 작업 실행자입니다
type Worker struct {
	ID        int           `json:"id"`
	State     WorkerState   `json:"state"`
	TaskQueue chan TaskWrapper `json:"-"`
	QuitChan  chan bool     `json:"-"`
	Pool      *WorkerPoolManager `json:"-"`
	
	// 통계
	TasksProcessed int64     `json:"tasks_processed"`
	TasksCompleted int64     `json:"tasks_completed"`
	TasksFailed    int64     `json:"tasks_failed"`
	LastTaskTime   time.Time `json:"last_task_time"`
	StartTime      time.Time `json:"start_time"`
	IdleTime       time.Duration `json:"idle_time"`
	
	// 현재 태스크
	CurrentTask    *TaskWrapper `json:"current_task,omitempty"`
	currentMutex   sync.RWMutex
	
	// 생명주기
	ctx    context.Context
	cancel context.CancelFunc
}

// WorkerState는 워커 상태입니다
type WorkerState int

const (
	WorkerStateIdle WorkerState = iota
	WorkerStateBusy
	WorkerStateStopping
	WorkerStateStopped
)

// GoroutineStats는 고루틴 통계입니다
type GoroutineStats struct {
	Active       int `json:"active"`
	Idle         int `json:"idle"`
	Busy         int `json:"busy"`
	Total        int `json:"total"`
	Completed    int64 `json:"completed"`
	Failed       int64 `json:"failed"`
	
	// 성능 지표
	AvgLifetime     time.Duration `json:"avg_lifetime"`
	AvgTaskDuration time.Duration `json:"avg_task_duration"`
	ThroughputPerSec float64      `json:"throughput_per_sec"`
	
	// 큐 상태
	QueuedTasks      int `json:"queued_tasks"`
	PriorityTasks    int `json:"priority_tasks"`
	
	// 리소스 사용량
	MemoryUsage      int64   `json:"memory_usage"`
	CPUUsage         float64 `json:"cpu_usage"`
	GoroutineCount   int     `json:"goroutine_count"`
	
	// 시간 정보
	LastUpdate       time.Time `json:"last_update"`
	UptimeSeconds    int64     `json:"uptime_seconds"`
}

// WorkerPoolConfig는 워커 풀 설정입니다
type WorkerPoolConfig struct {
	MinWorkers         int           `json:"min_workers"`
	MaxWorkers         int           `json:"max_workers"`
	IdleTimeout        time.Duration `json:"idle_timeout"`
	TaskTimeout        time.Duration `json:"task_timeout"`
	QueueSize          int           `json:"queue_size"`
	PriorityQueueSize  int           `json:"priority_queue_size"`
	
	// 스케일링 설정
	ScaleUpThreshold   float64       `json:"scale_up_threshold"`
	ScaleDownThreshold float64       `json:"scale_down_threshold"`
	ScaleInterval      time.Duration `json:"scale_interval"`
	
	// 모니터링
	StatsInterval      time.Duration `json:"stats_interval"`
	HealthCheckInterval time.Duration `json:"health_check_interval"`
	
	// 성능 최적화
	EnableProfiling    bool `json:"enable_profiling"`
	GCThreshold        int  `json:"gc_threshold"`
}

// WorkerScaler는 워커 자동 스케일링을 담당합니다
type WorkerScaler struct {
	pool       *WorkerPoolManager
	config     ScalerConfig
	
	// 메트릭
	metrics    ScalerMetrics
	metricsMutex sync.RWMutex
	
	// 결정 이력
	decisions  []ScalingDecision
	decisionMutex sync.RWMutex
	maxDecisions int
}

// ScalerConfig는 스케일러 설정입니다
type ScalerConfig struct {
	Enabled            bool          `json:"enabled"`
	MinWorkers         int           `json:"min_workers"`
	MaxWorkers         int           `json:"max_workers"`
	ScaleUpFactor      float64       `json:"scale_up_factor"`
	ScaleDownFactor    float64       `json:"scale_down_factor"`
	CooldownPeriod     time.Duration `json:"cooldown_period"`
	EvaluationWindow   time.Duration `json:"evaluation_window"`
}

// ScalerMetrics는 스케일러 메트릭입니다
type ScalerMetrics struct {
	LastScaleUp        time.Time     `json:"last_scale_up"`
	LastScaleDown      time.Time     `json:"last_scale_down"`
	ScaleUpCount       int64         `json:"scale_up_count"`
	ScaleDownCount     int64         `json:"scale_down_count"`
	CurrentUtilization float64       `json:"current_utilization"`
	TargetUtilization  float64       `json:"target_utilization"`
	EfficiencyScore    float64       `json:"efficiency_score"`
}

// ScalingDecision는 스케일링 결정입니다
type ScalingDecision struct {
	Timestamp   time.Time `json:"timestamp"`
	Action      string    `json:"action"`
	FromWorkers int       `json:"from_workers"`
	ToWorkers   int       `json:"to_workers"`
	Reason      string    `json:"reason"`
	Utilization float64   `json:"utilization"`
}

// GoroutineMonitor는 고루틴 모니터링을 담당합니다
type GoroutineMonitor struct {
	pool         *WorkerPoolManager
	
	// 모니터링 데이터
	snapshots    []MonitoringSnapshot
	snapshotMutex sync.RWMutex
	maxSnapshots int
	
	// 알림
	alertManager AlertManager
	thresholds   MonitoringThresholds
}

// MonitoringSnapshot은 모니터링 스냅샷입니다
type MonitoringSnapshot struct {
	Timestamp      time.Time `json:"timestamp"`
	WorkerCount    int       `json:"worker_count"`
	ActiveWorkers  int       `json:"active_workers"`
	QueueLength    int       `json:"queue_length"`
	MemoryUsage    int64     `json:"memory_usage"`
	GoroutineCount int       `json:"goroutine_count"`
	CPUUsage       float64   `json:"cpu_usage"`
	ThroughputRate float64   `json:"throughput_rate"`
}

// MonitoringThresholds는 모니터링 임계값입니다
type MonitoringThresholds struct {
	MaxMemoryUsage    int64   `json:"max_memory_usage"`
	MaxGoroutineCount int     `json:"max_goroutine_count"`
	MaxCPUUsage       float64 `json:"max_cpu_usage"`
	MinThroughput     float64 `json:"min_throughput"`
}

// DefaultWorkerPoolConfig는 기본 워커 풀 설정을 반환합니다
func DefaultWorkerPoolConfig() WorkerPoolConfig {
	return WorkerPoolConfig{
		MinWorkers:          2,
		MaxWorkers:          runtime.NumCPU() * 2,
		IdleTimeout:         5 * time.Minute,
		TaskTimeout:         30 * time.Second,
		QueueSize:           1000,
		PriorityQueueSize:   100,
		ScaleUpThreshold:    0.8,
		ScaleDownThreshold:  0.3,
		ScaleInterval:       30 * time.Second,
		StatsInterval:       10 * time.Second,
		HealthCheckInterval: 30 * time.Second,
		EnableProfiling:     false,
		GCThreshold:         1000,
	}
}

// NewWorkerPoolManager는 새로운 워커 풀 관리자를 생성합니다
func NewWorkerPoolManager(config WorkerPoolConfig) *WorkerPoolManager {
	ctx, cancel := context.WithCancel(context.Background())
	
	manager := &WorkerPoolManager{
		workers:       make(map[int]*Worker),
		taskQueue:     make(chan TaskWrapper, config.QueueSize),
		priorityQueue: make(chan TaskWrapper, config.PriorityQueueSize),
		config:        config,
		ctx:           ctx,
		cancel:        cancel,
		stats: GoroutineStats{
			LastUpdate: time.Now(),
		},
	}
	
	// 스케일러 초기화
	scalerConfig := ScalerConfig{
		Enabled:          true,
		MinWorkers:       config.MinWorkers,
		MaxWorkers:       config.MaxWorkers,
		ScaleUpFactor:    1.5,
		ScaleDownFactor:  0.7,
		CooldownPeriod:   2 * time.Minute,
		EvaluationWindow: 5 * time.Minute,
	}
	manager.scaler = NewWorkerScaler(manager, scalerConfig)
	
	// 모니터 초기화
	manager.monitor = NewGoroutineMonitor(manager)
	
	return manager
}

// Start는 워커 풀을 시작합니다
func (wpm *WorkerPoolManager) Start() error {
	if !wpm.running.CompareAndSwap(false, true) {
		return fmt.Errorf("worker pool is already running")
	}
	
	// 초기 워커들 생성
	for i := 0; i < wpm.config.MinWorkers; i++ {
		if err := wpm.createWorker(); err != nil {
			return fmt.Errorf("failed to create initial worker: %w", err)
		}
	}
	
	// 백그라운드 작업들 시작
	wpm.wg.Add(3)
	go wpm.taskDispatcher()
	go wpm.statisticsCollector()
	go wpm.healthChecker()
	
	// 스케일러 시작
	if wpm.scaler != nil {
		wpm.scaler.Start()
	}
	
	// 모니터 시작
	if wpm.monitor != nil {
		wpm.monitor.Start()
	}
	
	return nil
}

// Stop은 워커 풀을 중지합니다
func (wpm *WorkerPoolManager) Stop() error {
	if !wpm.running.CompareAndSwap(true, false) {
		return nil // 이미 중지됨
	}
	
	// 스케일러 중지
	if wpm.scaler != nil {
		wpm.scaler.Stop()
	}
	
	// 모니터 중지
	if wpm.monitor != nil {
		wpm.monitor.Stop()
	}
	
	// 모든 워커들 중지
	wpm.workersMutex.Lock()
	for _, worker := range wpm.workers {
		worker.Stop()
	}
	wpm.workersMutex.Unlock()
	
	// 컨텍스트 취소
	wpm.cancel()
	
	// 백그라운드 작업들 완료 대기
	wpm.wg.Wait()
	
	// 큐 정리
	close(wpm.taskQueue)
	close(wpm.priorityQueue)
	
	return nil
}

// SpawnWorker는 새로운 작업을 생성합니다
func (wpm *WorkerPoolManager) SpawnWorker(task Task) error {
	return wpm.SpawnBoundedWorker(task, wpm.config.TaskTimeout)
}

// SpawnBoundedWorker는 타임아웃이 있는 작업을 생성합니다
func (wpm *WorkerPoolManager) SpawnBoundedWorker(task Task, timeout time.Duration) error {
	if !wpm.running.Load() {
		return fmt.Errorf("worker pool is not running")
	}
	
	ctx, cancel := context.WithTimeout(wpm.ctx, timeout)
	
	wrapper := TaskWrapper{
		Task:      task,
		StartTime: time.Now(),
		Timeout:   timeout,
		ResultCh:  make(chan TaskResult, 1),
		Ctx:       ctx,
		Cancel:    cancel,
	}
	
	// 우선순위에 따라 큐 선택
	if task.GetPriority() >= TaskPriorityHigh {
		select {
		case wpm.priorityQueue <- wrapper:
			return nil
		default:
			cancel()
			return fmt.Errorf("priority queue is full")
		}
	} else {
		select {
		case wpm.taskQueue <- wrapper:
			return nil
		default:
			cancel()
			return fmt.Errorf("task queue is full")
		}
	}
}

// GetActiveGoroutines는 활성 고루틴 수를 반환합니다
func (wpm *WorkerPoolManager) GetActiveGoroutines() int {
	wpm.workersMutex.RLock()
	defer wpm.workersMutex.RUnlock()
	
	active := 0
	for _, worker := range wpm.workers {
		if worker.State == WorkerStateBusy {
			active++
		}
	}
	
	return active
}

// GetGoroutineStats는 고루틴 통계를 반환합니다
func (wpm *WorkerPoolManager) GetGoroutineStats() GoroutineStats {
	wpm.statsMutex.RLock()
	defer wpm.statsMutex.RUnlock()
	
	stats := wpm.stats
	stats.LastUpdate = time.Now()
	
	return stats
}

// CleanupIdleWorkers는 유휴 워커들을 정리합니다
func (wpm *WorkerPoolManager) CleanupIdleWorkers() error {
	wpm.workersMutex.Lock()
	defer wpm.workersMutex.Unlock()
	
	now := time.Now()
	var toRemove []int
	
	for id, worker := range wpm.workers {
		if worker.State == WorkerStateIdle {
			idleTime := now.Sub(worker.LastTaskTime)
			if idleTime > wpm.config.IdleTimeout {
				toRemove = append(toRemove, id)
			}
		}
	}
	
	// 최소 워커 수 유지
	currentWorkers := len(wpm.workers)
	maxRemovable := currentWorkers - wpm.config.MinWorkers
	if maxRemovable < 0 {
		maxRemovable = 0
	}
	
	if len(toRemove) > maxRemovable {
		toRemove = toRemove[:maxRemovable]
	}
	
	// 워커들 제거
	for _, id := range toRemove {
		if worker, exists := wpm.workers[id]; exists {
			worker.Stop()
			delete(wpm.workers, id)
		}
	}
	
	return nil
}

// SetMaxGoroutines는 최대 고루틴 수를 설정합니다
func (wpm *WorkerPoolManager) SetMaxGoroutines(max int) error {
	if max < wpm.config.MinWorkers {
		return fmt.Errorf("max goroutines cannot be less than min workers")
	}
	
	wpm.config.MaxWorkers = max
	
	// 현재 워커 수가 새로운 최대값보다 많으면 조정
	wpm.workersMutex.Lock()
	currentWorkers := len(wpm.workers)
	wpm.workersMutex.Unlock()
	
	if currentWorkers > max {
		excess := currentWorkers - max
		for i := 0; i < excess; i++ {
			wpm.removeWorker()
		}
	}
	
	return nil
}

// 내부 메서드들

func (wpm *WorkerPoolManager) createWorker() error {
	wpm.workersMutex.Lock()
	defer wpm.workersMutex.Unlock()
	
	if len(wpm.workers) >= wpm.config.MaxWorkers {
		return fmt.Errorf("maximum worker limit reached")
	}
	
	workerID := int(atomic.AddInt32(&wpm.nextWorkerID, 1))
	ctx, cancel := context.WithCancel(wpm.ctx)
	
	worker := &Worker{
		ID:        workerID,
		State:     WorkerStateIdle,
		TaskQueue: make(chan TaskWrapper, 1),
		QuitChan:  make(chan bool, 1),
		Pool:      wpm,
		StartTime: time.Now(),
		ctx:       ctx,
		cancel:    cancel,
	}
	
	wpm.workers[workerID] = worker
	
	// 워커 고루틴 시작
	go worker.Run()
	
	return nil
}

func (wpm *WorkerPoolManager) removeWorker() error {
	wpm.workersMutex.Lock()
	defer wpm.workersMutex.Unlock()
	
	// 유휴 상태인 워커 찾기
	for id, worker := range wpm.workers {
		if worker.State == WorkerStateIdle {
			worker.Stop()
			delete(wpm.workers, id)
			return nil
		}
	}
	
	return fmt.Errorf("no idle worker found to remove")
}

func (wpm *WorkerPoolManager) taskDispatcher() {
	defer wpm.wg.Done()
	
	for {
		select {
		case <-wpm.ctx.Done():
			return
		case task := <-wpm.priorityQueue:
			wpm.assignTask(task)
		case task := <-wpm.taskQueue:
			wpm.assignTask(task)
		}
	}
}

func (wpm *WorkerPoolManager) assignTask(task TaskWrapper) {
	wpm.workersMutex.RLock()
	
	// 유휴 워커 찾기
	var idleWorker *Worker
	for _, worker := range wpm.workers {
		if worker.State == WorkerStateIdle {
			idleWorker = worker
			break
		}
	}
	wpm.workersMutex.RUnlock()
	
	if idleWorker == nil {
		// 유휴 워커가 없으면 새 워커 생성 시도
		if err := wpm.createWorker(); err == nil {
			wpm.workersMutex.RLock()
			for _, worker := range wpm.workers {
				if worker.State == WorkerStateIdle {
					idleWorker = worker
					break
				}
			}
			wpm.workersMutex.RUnlock()
		}
	}
	
	if idleWorker != nil {
		select {
		case idleWorker.TaskQueue <- task:
			// 태스크 할당 성공
		default:
			// 워커가 바쁘면 다시 큐에 추가
			go func() {
				if task.Task.GetPriority() >= TaskPriorityHigh {
					wpm.priorityQueue <- task
				} else {
					wpm.taskQueue <- task
				}
			}()
		}
	} else {
		// 모든 워커가 바쁘면 다시 큐에 추가
		go func() {
			if task.Task.GetPriority() >= TaskPriorityHigh {
				wpm.priorityQueue <- task
			} else {
				wpm.taskQueue <- task
			}
		}()
	}
}

func (wpm *WorkerPoolManager) statisticsCollector() {
	defer wpm.wg.Done()
	
	ticker := time.NewTicker(wpm.config.StatsInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-wpm.ctx.Done():
			return
		case <-ticker.C:
			wpm.updateStatistics()
		}
	}
}

func (wpm *WorkerPoolManager) updateStatistics() {
	wpm.workersMutex.RLock()
	workerCount := len(wpm.workers)
	activeWorkers := 0
	idleWorkers := 0
	var totalCompleted, totalFailed int64
	var totalLifetime time.Duration
	
	for _, worker := range wpm.workers {
		switch worker.State {
		case WorkerStateBusy:
			activeWorkers++
		case WorkerStateIdle:
			idleWorkers++
		}
		
		totalCompleted += worker.TasksCompleted
		totalFailed += worker.TasksFailed
		totalLifetime += time.Since(worker.StartTime)
	}
	wpm.workersMutex.RUnlock()
	
	wpm.statsMutex.Lock()
	wpm.stats.Active = activeWorkers
	wpm.stats.Idle = idleWorkers
	wpm.stats.Busy = activeWorkers
	wpm.stats.Total = workerCount
	wpm.stats.Completed = totalCompleted
	wpm.stats.Failed = totalFailed
	wpm.stats.QueuedTasks = len(wpm.taskQueue)
	wpm.stats.PriorityTasks = len(wpm.priorityQueue)
	wpm.stats.GoroutineCount = runtime.NumGoroutine()
	wpm.stats.LastUpdate = time.Now()
	
	if workerCount > 0 {
		wpm.stats.AvgLifetime = totalLifetime / time.Duration(workerCount)
	}
	wpm.statsMutex.Unlock()
}

func (wpm *WorkerPoolManager) healthChecker() {
	defer wpm.wg.Done()
	
	ticker := time.NewTicker(wpm.config.HealthCheckInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-wpm.ctx.Done():
			return
		case <-ticker.C:
			wpm.performHealthCheck()
		}
	}
}

func (wpm *WorkerPoolManager) performHealthCheck() {
	// 메모리 사용량 확인
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	wpm.statsMutex.Lock()
	wpm.stats.MemoryUsage = int64(m.Alloc)
	wpm.statsMutex.Unlock()
	
	// GC 임계값 확인
	if wpm.config.EnableProfiling && wpm.stats.Completed%int64(wpm.config.GCThreshold) == 0 {
		runtime.GC()
	}
	
	// 유휴 워커 정리
	wpm.CleanupIdleWorkers()
}

// Worker 메서드들

// Run은 워커를 실행합니다
func (w *Worker) Run() {
	defer func() {
		w.State = WorkerStateStopped
		if r := recover(); r != nil {
			// 패닉 복구
			fmt.Printf("Worker %d panicked: %v\n", w.ID, r)
		}
	}()
	
	for {
		w.State = WorkerStateIdle
		w.LastTaskTime = time.Now()
		
		select {
		case <-w.ctx.Done():
			return
		case <-w.QuitChan:
			return
		case task := <-w.TaskQueue:
			w.executeTask(task)
		}
	}
}

func (w *Worker) executeTask(wrapper TaskWrapper) {
	w.State = WorkerStateBusy
	w.TasksProcessed++
	
	w.currentMutex.Lock()
	w.CurrentTask = &wrapper
	w.currentMutex.Unlock()
	
	startTime := time.Now()
	err := wrapper.Task.Execute(wrapper.Ctx)
	endTime := time.Now()
	duration := endTime.Sub(startTime)
	
	result := TaskResult{
		Success:   err == nil,
		Error:     err,
		Duration:  duration,
		StartTime: startTime,
		EndTime:   endTime,
	}
	
	if err == nil {
		w.TasksCompleted++
	} else {
		w.TasksFailed++
	}
	
	// 결과 전송
	select {
	case wrapper.ResultCh <- result:
	default:
		// 채널이 막혀있으면 무시
	}
	
	// 컨텍스트 정리
	wrapper.Cancel()
	
	w.currentMutex.Lock()
	w.CurrentTask = nil
	w.currentMutex.Unlock()
	
	w.LastTaskTime = time.Now()
}

// Stop은 워커를 중지합니다
func (w *Worker) Stop() {
	if w.State != WorkerStateStopped {
		w.State = WorkerStateStopping
		
		// 현재 실행 중인 태스크 취소
		w.currentMutex.RLock()
		if w.CurrentTask != nil {
			w.CurrentTask.Cancel()
		}
		w.currentMutex.RUnlock()
		
		w.cancel()
		close(w.QuitChan)
	}
}

// NewWorkerScaler는 새로운 워커 스케일러를 생성합니다
func NewWorkerScaler(pool *WorkerPoolManager, config ScalerConfig) *WorkerScaler {
	return &WorkerScaler{
		pool:         pool,
		config:       config,
		decisions:    make([]ScalingDecision, 0),
		maxDecisions: 100,
	}
}

// Start는 스케일러를 시작합니다
func (ws *WorkerScaler) Start() {
	// 스케일러 구현은 간단화
}

// Stop은 스케일러를 중지합니다
func (ws *WorkerScaler) Stop() {
	// 스케일러 중지 로직
}

// NewGoroutineMonitor는 새로운 고루틴 모니터를 생성합니다
func NewGoroutineMonitor(pool *WorkerPoolManager) *GoroutineMonitor {
	return &GoroutineMonitor{
		pool:         pool,
		snapshots:    make([]MonitoringSnapshot, 0),
		maxSnapshots: 1000,
	}
}

// Start는 모니터를 시작합니다
func (gm *GoroutineMonitor) Start() {
	// 모니터 구현은 간단화
}

// Stop은 모니터를 중지합니다
func (gm *GoroutineMonitor) Stop() {
	// 모니터 중지 로직
}