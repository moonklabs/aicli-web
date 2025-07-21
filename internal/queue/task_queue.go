package queue

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/aicli/aicli-web/internal/models"
)

// TaskExecutor 태스크 실행 함수 타입
type TaskExecutor func(ctx context.Context, task *models.Task) (string, error)

// TaskQueue 태스크 큐 구조체
type TaskQueue struct {
	// 큐 설정
	maxWorkers   int
	maxQueueSize int
	
	// 채널들
	taskChan   chan *models.Task
	resultChan chan *TaskResult
	stopChan   chan struct{}
	
	// 태스크 관리
	tasks    map[string]*models.Task
	tasksMux sync.RWMutex
	
	// 워커 관리
	workerWg sync.WaitGroup
	running  bool
	runMux   sync.RWMutex
	
	// 실행기
	executor TaskExecutor
}

// TaskResult 태스크 실행 결과
type TaskResult struct {
	TaskID string
	Output string
	Error  error
}

// TaskQueueConfig 태스크 큐 설정
type TaskQueueConfig struct {
	MaxWorkers   int           // 최대 워커 수
	MaxQueueSize int           // 최대 큐 크기
	Executor     TaskExecutor  // 태스크 실행기
}

// NewTaskQueue 새 태스크 큐 생성
func NewTaskQueue(config *TaskQueueConfig) *TaskQueue {
	if config == nil {
		config = &TaskQueueConfig{
			MaxWorkers:   5,
			MaxQueueSize: 100,
		}
	}
	
	if config.MaxWorkers <= 0 {
		config.MaxWorkers = 5
	}
	
	if config.MaxQueueSize <= 0 {
		config.MaxQueueSize = 100
	}
	
	tq := &TaskQueue{
		maxWorkers:   config.MaxWorkers,
		maxQueueSize: config.MaxQueueSize,
		taskChan:     make(chan *models.Task, config.MaxQueueSize),
		resultChan:   make(chan *TaskResult, config.MaxQueueSize),
		stopChan:     make(chan struct{}),
		tasks:        make(map[string]*models.Task),
		executor:     config.Executor,
	}
	
	return tq
}

// Start 태스크 큐 시작
func (tq *TaskQueue) Start(ctx context.Context) error {
	tq.runMux.Lock()
	defer tq.runMux.Unlock()
	
	if tq.running {
		return fmt.Errorf("태스크 큐가 이미 실행 중입니다")
	}
	
	tq.running = true
	
	// 워커 고루틴 시작
	for i := 0; i < tq.maxWorkers; i++ {
		tq.workerWg.Add(1)
		go tq.worker(ctx, i)
	}
	
	// 결과 처리 고루틴 시작
	go tq.resultProcessor(ctx)
	
	log.Printf("태스크 큐 시작됨 (워커: %d, 큐 크기: %d)", tq.maxWorkers, tq.maxQueueSize)
	return nil
}

// Stop 태스크 큐 중지
func (tq *TaskQueue) Stop() {
	tq.runMux.Lock()
	defer tq.runMux.Unlock()
	
	if !tq.running {
		return
	}
	
	tq.running = false
	close(tq.stopChan)
	
	// 모든 워커가 종료될 때까지 대기
	tq.workerWg.Wait()
	
	log.Println("태스크 큐 중지됨")
}

// Submit 태스크 제출
func (tq *TaskQueue) Submit(task *models.Task) error {
	tq.runMux.RLock()
	defer tq.runMux.RUnlock()
	
	if !tq.running {
		return fmt.Errorf("태스크 큐가 실행 중이 아닙니다")
	}
	
	// 태스크 등록
	tq.tasksMux.Lock()
	tq.tasks[task.ID] = task
	tq.tasksMux.Unlock()
	
	// 큐에 추가 (블로킹)
	select {
	case tq.taskChan <- task:
		log.Printf("태스크 제출됨: %s", task.ID)
		return nil
	default:
		// 큐가 가득 참
		tq.tasksMux.Lock()
		delete(tq.tasks, task.ID)
		tq.tasksMux.Unlock()
		return fmt.Errorf("태스크 큐가 가득 찼습니다")
	}
}

// Cancel 태스크 취소
func (tq *TaskQueue) Cancel(taskID string) error {
	tq.tasksMux.Lock()
	defer tq.tasksMux.Unlock()
	
	task, exists := tq.tasks[taskID]
	if !exists {
		return fmt.Errorf("태스크를 찾을 수 없습니다: %s", taskID)
	}
	
	if !task.CanCancel() {
		return fmt.Errorf("태스크를 취소할 수 없습니다: %s (상태: %s)", taskID, task.Status)
	}
	
	task.SetCancelled()
	log.Printf("태스크 취소됨: %s", taskID)
	return nil
}

// GetTask 태스크 조회
func (tq *TaskQueue) GetTask(taskID string) (*models.Task, bool) {
	tq.tasksMux.RLock()
	defer tq.tasksMux.RUnlock()
	
	task, exists := tq.tasks[taskID]
	return task, exists
}

// GetActiveTasks 활성 태스크 목록 조회
func (tq *TaskQueue) GetActiveTasks() []*models.Task {
	tq.tasksMux.RLock()
	defer tq.tasksMux.RUnlock()
	
	var activeTasks []*models.Task
	for _, task := range tq.tasks {
		if task.IsActive() {
			activeTasks = append(activeTasks, task)
		}
	}
	
	return activeTasks
}

// GetQueueSize 큐 크기 조회
func (tq *TaskQueue) GetQueueSize() int {
	return len(tq.taskChan)
}

// GetStats 큐 통계 조회
func (tq *TaskQueue) GetStats() map[string]interface{} {
	tq.tasksMux.RLock()
	defer tq.tasksMux.RUnlock()
	
	stats := map[string]interface{}{
		"total_tasks":    len(tq.tasks),
		"queue_size":     len(tq.taskChan),
		"max_workers":    tq.maxWorkers,
		"max_queue_size": tq.maxQueueSize,
		"running":        tq.running,
	}
	
	// 상태별 카운트
	statusCount := make(map[models.TaskStatus]int)
	for _, task := range tq.tasks {
		statusCount[task.Status]++
	}
	stats["status_count"] = statusCount
	
	return stats
}

// worker 워커 고루틴
func (tq *TaskQueue) worker(ctx context.Context, workerID int) {
	defer tq.workerWg.Done()
	
	log.Printf("워커 %d 시작됨", workerID)
	defer log.Printf("워커 %d 종료됨", workerID)
	
	for {
		select {
		case task := <-tq.taskChan:
			tq.processTask(ctx, task, workerID)
		case <-tq.stopChan:
			return
		case <-ctx.Done():
			return
		}
	}
}

// processTask 태스크 처리
func (tq *TaskQueue) processTask(ctx context.Context, task *models.Task, workerID int) {
	log.Printf("워커 %d: 태스크 %s 처리 시작", workerID, task.ID)
	
	// 취소 상태 확인
	if task.Status == models.TaskCancelled {
		log.Printf("워커 %d: 태스크 %s 이미 취소됨", workerID, task.ID)
		return
	}
	
	// 태스크 실행 시작
	task.SetRunning()
	
	// 실행기가 없으면 기본 처리
	if tq.executor == nil {
		time.Sleep(100 * time.Millisecond) // 기본 대기
		task.SetCompleted("기본 실행 완료")
		
		tq.resultChan <- &TaskResult{
			TaskID: task.ID,
			Output: "기본 실행 완료",
			Error:  nil,
		}
		return
	}
	
	// 컨텍스트 타임아웃 설정
	taskCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	
	// 태스크 실행
	output, err := tq.executor(taskCtx, task)
	
	// 결과 처리
	result := &TaskResult{
		TaskID: task.ID,
		Output: output,
		Error:  err,
	}
	
	if err != nil {
		task.SetFailed(err.Error())
		log.Printf("워커 %d: 태스크 %s 실행 실패: %v", workerID, task.ID, err)
	} else {
		task.SetCompleted(output)
		log.Printf("워커 %d: 태스크 %s 실행 완료", workerID, task.ID)
	}
	
	// 결과 전송
	select {
	case tq.resultChan <- result:
	default:
		log.Printf("워커 %d: 결과 채널 가득 참 (태스크: %s)", workerID, task.ID)
	}
}

// resultProcessor 결과 처리기
func (tq *TaskQueue) resultProcessor(ctx context.Context) {
	log.Println("결과 처리기 시작됨")
	defer log.Println("결과 처리기 종료됨")
	
	for {
		select {
		case result := <-tq.resultChan:
			// 여기서 추가적인 결과 처리 로직을 구현할 수 있음
			// 예: 데이터베이스 업데이트, 알림 전송 등
			log.Printf("결과 처리됨: 태스크 %s", result.TaskID)
			
		case <-tq.stopChan:
			return
		case <-ctx.Done():
			return
		}
	}
}

// cleanupCompletedTasks 완료된 태스크 정리
func (tq *TaskQueue) cleanupCompletedTasks(maxAge time.Duration) {
	tq.tasksMux.Lock()
	defer tq.tasksMux.Unlock()
	
	now := time.Now()
	var toDelete []string
	
	for id, task := range tq.tasks {
		if task.IsTerminal() && task.CompletedAt != nil && now.Sub(*task.CompletedAt) > maxAge {
			toDelete = append(toDelete, id)
		}
	}
	
	for _, id := range toDelete {
		delete(tq.tasks, id)
	}
	
	if len(toDelete) > 0 {
		log.Printf("완료된 태스크 %d개 정리됨", len(toDelete))
	}
}

// StartCleanupRoutine 정리 루틴 시작
func (tq *TaskQueue) StartCleanupRoutine(ctx context.Context, interval time.Duration, maxAge time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		
		for {
			select {
			case <-ticker.C:
				tq.cleanupCompletedTasks(maxAge)
			case <-ctx.Done():
				return
			case <-tq.stopChan:
				return
			}
		}
	}()
}