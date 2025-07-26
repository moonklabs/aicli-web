package services

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"

	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/queue"
	"github.com/aicli/aicli-web/internal/storage"
)

// TaskService 태스크 서비스
type TaskService struct {
	storage        storage.Storage
	sessionService *SessionService
	taskQueue      *queue.TaskQueue
	config         *TaskServiceConfig
}

// TaskServiceConfig 태스크 서비스 설정
type TaskServiceConfig struct {
	MaxWorkers      int           // 최대 워커 수
	MaxQueueSize    int           // 최대 큐 크기
	TaskTimeout     time.Duration // 태스크 타임아웃
	CleanupInterval time.Duration // 정리 주기
	CleanupMaxAge   time.Duration // 정리할 태스크 최대 나이
}

// DefaultTaskServiceConfig 기본 태스크 서비스 설정
func DefaultTaskServiceConfig() *TaskServiceConfig {
	return &TaskServiceConfig{
		MaxWorkers:      5,
		MaxQueueSize:    100,
		TaskTimeout:     5 * time.Minute,
		CleanupInterval: 10 * time.Minute,
		CleanupMaxAge:   1 * time.Hour,
	}
}

// NewTaskService 새 태스크 서비스 생성
func NewTaskService(storage storage.Storage, sessionService *SessionService, config *TaskServiceConfig) *TaskService {
	if config == nil {
		config = DefaultTaskServiceConfig()
	}
	
	ts := &TaskService{
		storage:        storage,
		sessionService: sessionService,
		config:         config,
	}
	
	// 태스크 큐 초기화
	queueConfig := &queue.TaskQueueConfig{
		MaxWorkers:   config.MaxWorkers,
		MaxQueueSize: config.MaxQueueSize,
		Executor:     ts.executeTask,
	}
	ts.taskQueue = queue.NewTaskQueue(queueConfig)
	
	return ts
}

// Start 태스크 서비스 시작
func (ts *TaskService) Start(ctx context.Context) error {
	// 태스크 큐 시작
	if err := ts.taskQueue.Start(ctx); err != nil {
		return fmt.Errorf("태스크 큐 시작 실패: %w", err)
	}
	
	// 정리 루틴 시작
	ts.taskQueue.StartCleanupRoutine(ctx, ts.config.CleanupInterval, ts.config.CleanupMaxAge)
	
	log.Println("태스크 서비스 시작됨")
	return nil
}

// Stop 태스크 서비스 중지
func (ts *TaskService) Stop() {
	ts.taskQueue.Stop()
	log.Println("태스크 서비스 중지됨")
}

// Create 새 태스크 생성
func (ts *TaskService) Create(ctx context.Context, req *models.TaskCreateRequest) (*models.Task, error) {
	// 빈 명령어 검증
	if strings.TrimSpace(req.Command) == "" {
		return nil, fmt.Errorf("명령어가 비어있습니다")
	}
	
	// 세션 존재 확인
	session, err := ts.sessionService.GetByID(ctx, req.SessionID)
	if err != nil {
		return nil, fmt.Errorf("세션을 찾을 수 없습니다: %w", err)
	}
	
	// 세션이 활성 상태인지 확인
	if !session.IsActive() {
		return nil, fmt.Errorf("세션이 활성 상태가 아닙니다: %s", session.Status)
	}
	
	// 태스크 생성
	task := &models.Task{
		SessionID: req.SessionID,
		Command:   req.Command,
		Status:    models.TaskPending,
	}
	
	// 데이터베이스에 저장
	if err := ts.storage.Task().Create(ctx, task); err != nil {
		return nil, fmt.Errorf("태스크 생성 실패: %w", err)
	}
	
	// 태스크 큐에 제출
	if err := ts.taskQueue.Submit(task); err != nil {
		// 큐 제출 실패 시 태스크 삭제
		_ = ts.storage.Task().Delete(ctx, task.ID)
		return nil, fmt.Errorf("태스크 큐 제출 실패: %w", err)
	}
	
	log.Printf("태스크 생성됨: %s (세션: %s)", task.ID, task.SessionID)
	return task, nil
}

// GetByID ID로 태스크 조회
func (ts *TaskService) GetByID(ctx context.Context, id string) (*models.Task, error) {
	if id == "" {
		return nil, fmt.Errorf("태스크 ID가 필요합니다")
	}
	
	// 먼저 큐에서 조회 (최신 상태)
	if task, exists := ts.taskQueue.GetTask(id); exists {
		return task, nil
	}
	
	// 큐에 없으면 데이터베이스에서 조회
	task, err := ts.storage.Task().GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("태스크 조회 실패: %w", err)
	}
	
	return task, nil
}

// List 태스크 목록 조회
func (ts *TaskService) List(ctx context.Context, filter *models.TaskFilter, paging *models.PagingRequest) (*models.PagingResponse, error) {
	tasks, total, err := ts.storage.Task().List(ctx, filter, paging)
	if err != nil {
		return nil, fmt.Errorf("태스크 목록 조회 실패: %w", err)
	}
	
	return &models.PagingResponse{
		Data: tasks,
		Meta: models.NewPaginationMeta(paging.Page, paging.Limit, total),
	}, nil
}

// Cancel 태스크 취소
func (ts *TaskService) Cancel(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("태스크 ID가 필요합니다")
	}
	
	// 큐에서 취소 시도
	if err := ts.taskQueue.Cancel(id); err != nil {
		// 큐에 없으면 데이터베이스에서 조회하여 취소
		task, dbErr := ts.storage.Task().GetByID(ctx, id)
		if dbErr != nil {
			return fmt.Errorf("태스크를 찾을 수 없습니다: %s", id)
		}
		
		if !task.CanCancel() {
			return fmt.Errorf("태스크를 취소할 수 없습니다: %s (상태: %s)", id, task.Status)
		}
		
		task.SetCancelled()
		if updateErr := ts.storage.Task().Update(ctx, task); updateErr != nil {
			return fmt.Errorf("태스크 취소 업데이트 실패: %w", updateErr)
		}
	}
	
	log.Printf("태스크 취소됨: %s", id)
	return nil
}

// GetActiveTasks 활성 태스크 목록 조회
func (ts *TaskService) GetActiveTasks(ctx context.Context) ([]*models.TaskResponse, error) {
	// 큐에서 활성 태스크 조회
	activeTasks := ts.taskQueue.GetActiveTasks()
	
	responses := make([]*models.TaskResponse, len(activeTasks))
	for i, task := range activeTasks {
		responses[i] = task.ToResponse()
	}
	
	return responses, nil
}

// GetStats 태스크 통계 조회
func (ts *TaskService) GetStats(ctx context.Context) (map[string]interface{}, error) {
	// 큐 통계
	queueStats := ts.taskQueue.GetStats()
	
	// 데이터베이스 통계도 추가할 수 있음
	
	return queueStats, nil
}

// UpdateStatus 태스크 상태 업데이트 (내부용)
func (ts *TaskService) UpdateStatus(ctx context.Context, id string, status models.TaskStatus) error {
	task, err := ts.storage.Task().GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("태스크 조회 실패: %w", err)
	}
	
	task.Status = status
	if err := ts.storage.Task().Update(ctx, task); err != nil {
		return fmt.Errorf("태스크 상태 업데이트 실패: %w", err)
	}
	
	return nil
}

// executeTask 태스크 실행 (큐에서 호출됨)
func (ts *TaskService) executeTask(ctx context.Context, task *models.Task) (string, error) {
	log.Printf("태스크 실행 시작: %s", task.ID)
	
	// 세션 정보 조회
	session, err := ts.sessionService.GetByID(ctx, task.SessionID)
	if err != nil {
		return "", fmt.Errorf("세션 조회 실패: %w", err)
	}
	
	// 세션 활동 업데이트
	_ = ts.sessionService.UpdateActivity(ctx, session.ID)
	
	// 실제 명령 실행
	output, err := ts.executeCommand(ctx, task.Command, session)
	
	// 통계 업데이트
	if err == nil {
		_ = ts.sessionService.UpdateStats(ctx, session.ID, 1, int64(len(task.Command)), int64(len(output)), 0)
	} else {
		_ = ts.sessionService.UpdateStats(ctx, session.ID, 1, int64(len(task.Command)), 0, 1)
	}
	
	// 데이터베이스 업데이트
	if err := ts.storage.Task().Update(ctx, task); err != nil {
		log.Printf("태스크 업데이트 실패: %v", err)
	}
	
	return output, err
}

// executeCommand 명령 실행
func (ts *TaskService) executeCommand(ctx context.Context, command string, session *models.Session) (string, error) {
	// 타임아웃 컨텍스트 생성
	cmdCtx, cancel := context.WithTimeout(ctx, ts.config.TaskTimeout)
	defer cancel()
	
	// 명령어 파싱
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return "", fmt.Errorf("빈 명령어입니다")
	}
	
	// 보안을 위한 명령어 검증
	if err := ts.validateCommand(parts[0]); err != nil {
		return "", err
	}
	
	// 명령 실행
	var cmd *exec.Cmd
	if len(parts) == 1 {
		cmd = exec.CommandContext(cmdCtx, parts[0])
	} else {
		cmd = exec.CommandContext(cmdCtx, parts[0], parts[1:]...)
	}
	
	// 작업 디렉토리 설정 (프로젝트 경로)
	if session.ProjectID != "" {
		project, err := ts.storage.Project().GetByID(ctx, session.ProjectID)
		if err == nil && project.Path != "" {
			cmd.Dir = project.Path
		}
	}
	
	// 명령 실행
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("명령 실행 실패: %w", err)
	}
	
	return string(output), nil
}

// validateCommand 명령어 검증
func (ts *TaskService) validateCommand(command string) error {
	// 위험한 명령어 차단
	dangerousCommands := []string{
		"rm", "del", "format", "fdisk", "mkfs",
		"sudo", "su", "chmod", "chown",
		"reboot", "shutdown", "halt",
	}
	
	for _, dangerous := range dangerousCommands {
		if command == dangerous {
			return fmt.Errorf("위험한 명령어는 실행할 수 없습니다: %s", command)
		}
	}
	
	// 허용된 명령어 목록 (화이트리스트)
	allowedCommands := []string{
		"echo", "cat", "ls", "pwd", "date", "whoami",
		"git", "node", "npm", "go", "python", "python3",
		"docker", "kubectl", "curl", "wget",
		"grep", "find", "head", "tail", "wc",
	}
	
	for _, allowed := range allowedCommands {
		if command == allowed {
			return nil
		}
	}
	
	return fmt.Errorf("허용되지 않은 명령어입니다: %s", command)
}