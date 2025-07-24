package batch

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"go.uber.org/zap"
	
	"github.com/aicli/aicli-web/internal/storage"
)

// JobStatus 작업 상태
type JobStatus string

const (
	JobStatusPending    JobStatus = "pending"
	JobStatusRunning    JobStatus = "running"
	JobStatusCompleted  JobStatus = "completed"
	JobStatusFailed     JobStatus = "failed"
	JobStatusCancelled  JobStatus = "cancelled"
)

// JobType 작업 유형
type JobType string

const (
	JobTypeDataImport   JobType = "data_import"
	JobTypeDataExport   JobType = "data_export"
	JobTypeDataCleanup  JobType = "data_cleanup"
	JobTypeIndexRebuild JobType = "index_rebuild"
	JobTypeBackup       JobType = "backup"
	JobTypeRestore      JobType = "restore"
)

// Job 배치 작업
type Job struct {
	ID          string                 `json:"id"`
	Type        JobType                `json:"type"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Status      JobStatus              `json:"status"`
	Progress    int                    `json:"progress"` // 0-100
	Parameters  map[string]interface{} `json:"parameters"`
	CreatedAt   time.Time              `json:"created_at"`
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Result      map[string]interface{} `json:"result,omitempty"`
}

// JobHandler 작업 핸들러
type JobHandler func(ctx context.Context, job *Job, progressCallback func(int)) error

// JobScheduler 배치 작업 스케줄러
type JobScheduler struct {
	storage   storage.Storage
	logger    *zap.Logger
	jobs      map[string]*Job
	handlers  map[JobType]JobHandler
	queue     chan *Job
	workers   int
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

// SchedulerConfig 스케줄러 설정
type SchedulerConfig struct {
	Storage     storage.Storage
	Workers     int
	QueueSize   int
	Logger      *zap.Logger
}

// DefaultSchedulerConfig 기본 스케줄러 설정
func DefaultSchedulerConfig(storage storage.Storage) SchedulerConfig {
	return SchedulerConfig{
		Storage:   storage,
		Workers:   4,
		QueueSize: 100,
		Logger:    zap.NewNop(),
	}
}

// NewJobScheduler 새 작업 스케줄러 생성
func NewJobScheduler(config SchedulerConfig) *JobScheduler {
	if config.Logger == nil {
		config.Logger = zap.NewNop()
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	scheduler := &JobScheduler{
		storage:  config.Storage,
		logger:   config.Logger,
		jobs:     make(map[string]*Job),
		handlers: make(map[JobType]JobHandler),
		queue:    make(chan *Job, config.QueueSize),
		workers:  config.Workers,
		ctx:      ctx,
		cancel:   cancel,
	}
	
	// 기본 핸들러 등록
	scheduler.registerDefaultHandlers()
	
	// 워커 고루틴 시작
	for i := 0; i < config.Workers; i++ {
		scheduler.wg.Add(1)
		go scheduler.worker(i)
	}
	
	return scheduler
}

// RegisterHandler 작업 핸들러 등록
func (js *JobScheduler) RegisterHandler(jobType JobType, handler JobHandler) {
	js.mu.Lock()
	defer js.mu.Unlock()
	
	js.handlers[jobType] = handler
	
	js.logger.Info("작업 핸들러 등록됨",
		zap.String("job_type", string(jobType)),
	)
}

// SubmitJob 작업 제출
func (js *JobScheduler) SubmitJob(job *Job) error {
	if job.ID == "" {
		job.ID = generateJobID()
	}
	
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now()
	}
	
	job.Status = JobStatusPending
	
	js.mu.Lock()
	js.jobs[job.ID] = job
	js.mu.Unlock()
	
	select {
	case js.queue <- job:
		js.logger.Info("작업이 큐에 추가됨",
			zap.String("job_id", job.ID),
			zap.String("job_type", string(job.Type)),
			zap.String("job_name", job.Name),
		)
		return nil
	default:
		js.mu.Lock()
		job.Status = JobStatusFailed
		job.Error = "작업 큐가 가득참"
		js.mu.Unlock()
		
		return fmt.Errorf("작업 큐가 가득참")
	}
}

// GetJob 작업 조회
func (js *JobScheduler) GetJob(jobID string) (*Job, bool) {
	js.mu.RLock()
	defer js.mu.RUnlock()
	
	job, exists := js.jobs[jobID]
	if !exists {
		return nil, false
	}
	
	// 복사본 반환
	jobCopy := *job
	return &jobCopy, true
}

// ListJobs 작업 목록 조회
func (js *JobScheduler) ListJobs() []*Job {
	js.mu.RLock()
	defer js.mu.RUnlock()
	
	jobs := make([]*Job, 0, len(js.jobs))
	for _, job := range js.jobs {
		jobCopy := *job
		jobs = append(jobs, &jobCopy)
	}
	
	return jobs
}

// CancelJob 작업 취소
func (js *JobScheduler) CancelJob(jobID string) error {
	js.mu.Lock()
	defer js.mu.Unlock()
	
	job, exists := js.jobs[jobID]
	if !exists {
		return fmt.Errorf("작업을 찾을 수 없음: %s", jobID)
	}
	
	if job.Status == JobStatusRunning {
		// 실행 중인 작업은 즉시 취소할 수 없음
		// 실제로는 context 취소를 통해 처리해야 함
		return fmt.Errorf("실행 중인 작업은 취소할 수 없음: %s", jobID)
	}
	
	if job.Status == JobStatusPending {
		job.Status = JobStatusCancelled
		now := time.Now()
		job.CompletedAt = &now
		
		js.logger.Info("작업이 취소됨",
			zap.String("job_id", jobID),
		)
		
		return nil
	}
	
	return fmt.Errorf("취소할 수 없는 작업 상태: %s", job.Status)
}

// Close 스케줄러 종료
func (js *JobScheduler) Close() error {
	js.cancel()
	close(js.queue)
	
	// 모든 워커가 종료될 때까지 대기
	js.wg.Wait()
	
	js.logger.Info("작업 스케줄러가 종료됨")
	return nil
}

// worker 워커 고루틴
func (js *JobScheduler) worker(workerID int) {
	defer js.wg.Done()
	
	js.logger.Info("워커 시작됨",
		zap.Int("worker_id", workerID),
	)
	
	for {
		select {
		case job, ok := <-js.queue:
			if !ok {
				js.logger.Info("워커 종료됨",
					zap.Int("worker_id", workerID),
				)
				return
			}
			
			js.processJob(workerID, job)
			
		case <-js.ctx.Done():
			js.logger.Info("워커 컨텍스트 취소됨",
				zap.Int("worker_id", workerID),
			)
			return
		}
	}
}

// processJob 작업 처리
func (js *JobScheduler) processJob(workerID int, job *Job) {
	js.logger.Info("작업 처리 시작",
		zap.Int("worker_id", workerID),
		zap.String("job_id", job.ID),
		zap.String("job_type", string(job.Type)),
	)
	
	// 작업 상태 업데이트
	js.mu.Lock()
	job.Status = JobStatusRunning
	now := time.Now()
	job.StartedAt = &now
	job.Progress = 0
	js.mu.Unlock()
	
	// 핸들러 조회
	js.mu.RLock()
	handler, exists := js.handlers[job.Type]
	js.mu.RUnlock()
	
	if !exists {
		js.logger.Error("작업 핸들러를 찾을 수 없음",
			zap.String("job_id", job.ID),
			zap.String("job_type", string(job.Type)),
		)
		
		js.mu.Lock()
		job.Status = JobStatusFailed
		job.Error = fmt.Sprintf("핸들러를 찾을 수 없음: %s", job.Type)
		now := time.Now()
		job.CompletedAt = &now
		js.mu.Unlock()
		return
	}
	
	// 진행률 콜백
	progressCallback := func(progress int) {
		js.mu.Lock()
		job.Progress = progress
		js.mu.Unlock()
		
		js.logger.Debug("작업 진행률 업데이트",
			zap.String("job_id", job.ID),
			zap.Int("progress", progress),
		)
	}
	
	// 작업 실행
	start := time.Now()
	err := handler(js.ctx, job, progressCallback)
	duration := time.Since(start)
	
	// 결과 업데이트
	js.mu.Lock()
	now = time.Now()
	job.CompletedAt = &now
	
	if err != nil {
		job.Status = JobStatusFailed
		job.Error = err.Error()
		
		js.logger.Error("작업 실행 실패",
			zap.String("job_id", job.ID),
			zap.String("job_type", string(job.Type)),
			zap.Duration("duration", duration),
			zap.Error(err),
		)
	} else {
		job.Status = JobStatusCompleted
		job.Progress = 100
		
		js.logger.Info("작업 실행 완료",
			zap.String("job_id", job.ID),
			zap.String("job_type", string(job.Type)),
			zap.Duration("duration", duration),
		)
	}
	js.mu.Unlock()
}

// registerDefaultHandlers 기본 핸들러 등록
func (js *JobScheduler) registerDefaultHandlers() {
	// 데이터 정리 핸들러
	js.RegisterHandler(JobTypeDataCleanup, func(ctx context.Context, job *Job, progressCallback func(int)) error {
		js.logger.Info("데이터 정리 작업 시작", zap.String("job_id", job.ID))
		
		progressCallback(25)
		
		// 소프트 삭제된 워크스페이스 정리
		// TODO: 실제 구현
		time.Sleep(2 * time.Second) // 시뮬레이션
		
		progressCallback(50)
		
		// 소프트 삭제된 프로젝트 정리
		time.Sleep(2 * time.Second) // 시뮬레이션
		
		progressCallback(75)
		
		// 오래된 세션 정리
		time.Sleep(1 * time.Second) // 시뮬레이션
		
		progressCallback(100)
		
		if job.Result == nil {
			job.Result = make(map[string]interface{})
		}
		job.Result["cleaned_workspaces"] = 10
		job.Result["cleaned_projects"] = 50
		job.Result["cleaned_sessions"] = 100
		
		js.logger.Info("데이터 정리 작업 완료", zap.String("job_id", job.ID))
		return nil
	})
	
	// 인덱스 재구축 핸들러
	js.RegisterHandler(JobTypeIndexRebuild, func(ctx context.Context, job *Job, progressCallback func(int)) error {
		js.logger.Info("인덱스 재구축 작업 시작", zap.String("job_id", job.ID))
		
		progressCallback(20)
		
		// 워크스페이스 인덱스 재구축
		time.Sleep(3 * time.Second) // 시뮬레이션
		
		progressCallback(60)
		
		// 프로젝트 인덱스 재구축
		time.Sleep(3 * time.Second) // 시뮬레이션
		
		progressCallback(100)
		
		if job.Result == nil {
			job.Result = make(map[string]interface{})
		}
		job.Result["rebuilt_indexes"] = []string{"workspaces", "projects", "sessions", "tasks"}
		
		js.logger.Info("인덱스 재구축 작업 완료", zap.String("job_id", job.ID))
		return nil
	})
	
	// 백업 핸들러
	js.RegisterHandler(JobTypeBackup, func(ctx context.Context, job *Job, progressCallback func(int)) error {
		js.logger.Info("백업 작업 시작", zap.String("job_id", job.ID))
		
		backupPath, ok := job.Parameters["backup_path"].(string)
		if !ok || backupPath == "" {
			return fmt.Errorf("백업 경로가 지정되지 않았습니다")
		}
		
		progressCallback(10)
		
		// 백업 실행 (시뮬레이션)
		for i := 1; i <= 10; i++ {
			time.Sleep(500 * time.Millisecond)
			progressCallback(i * 10)
			
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
		}
		
		if job.Result == nil {
			job.Result = make(map[string]interface{})
		}
		job.Result["backup_path"] = backupPath
		job.Result["backup_size"] = "1.5GB"
		job.Result["tables_backed_up"] = 4
		
		js.logger.Info("백업 작업 완료", 
			zap.String("job_id", job.ID),
			zap.String("backup_path", backupPath),
		)
		return nil
	})
}

// generateJobID 작업 ID 생성
func generateJobID() string {
	return fmt.Sprintf("job_%d_%d", time.Now().UnixNano(), rand.Int31())
}

// JobStats 작업 통계
type JobStats struct {
	Total      int `json:"total"`
	Pending    int `json:"pending"`
	Running    int `json:"running"`
	Completed  int `json:"completed"`
	Failed     int `json:"failed"`
	Cancelled  int `json:"cancelled"`
}

// GetStats 작업 통계 반환
func (js *JobScheduler) GetStats() JobStats {
	js.mu.RLock()
	defer js.mu.RUnlock()
	
	stats := JobStats{}
	
	for _, job := range js.jobs {
		stats.Total++
		
		switch job.Status {
		case JobStatusPending:
			stats.Pending++
		case JobStatusRunning:
			stats.Running++
		case JobStatusCompleted:
			stats.Completed++
		case JobStatusFailed:
			stats.Failed++
		case JobStatusCancelled:
			stats.Cancelled++
		}
	}
	
	return stats
}

// CleanupCompletedJobs 완료된 작업 정리
func (js *JobScheduler) CleanupCompletedJobs(olderThan time.Duration) int {
	js.mu.Lock()
	defer js.mu.Unlock()
	
	cutoff := time.Now().Add(-olderThan)
	var cleaned int
	
	for jobID, job := range js.jobs {
		if job.Status == JobStatusCompleted || job.Status == JobStatusFailed || job.Status == JobStatusCancelled {
			if job.CompletedAt != nil && job.CompletedAt.Before(cutoff) {
				delete(js.jobs, jobID)
				cleaned++
			}
		}
	}
	
	if cleaned > 0 {
		js.logger.Info("완료된 작업 정리 완료",
			zap.Int("cleaned_count", cleaned),
			zap.Duration("older_than", olderThan),
		)
	}
	
	return cleaned
}

