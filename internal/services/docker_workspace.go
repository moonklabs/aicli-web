package services

import (
	"context"
	"fmt"
	"sync"
	"time"
	
	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/storage/interfaces"
	"github.com/aicli/aicli-web/internal/docker"
	"github.com/aicli/aicli-web/internal/docker/status"
	"github.com/aicli/aicli-web/internal/docker/security"
)

// TaskType 워크스페이스 작업 타입
type TaskType string

const (
	TaskTypeCreate    TaskType = "create"
	TaskTypeStart     TaskType = "start"
	TaskTypeStop      TaskType = "stop"
	TaskTypeRestart   TaskType = "restart"
	TaskTypeDelete    TaskType = "delete"
	TaskTypeSync      TaskType = "sync"
)

// WorkspaceTask 비동기 워크스페이스 작업 정의
type WorkspaceTask struct {
	Type        TaskType             `json:"type"`
	WorkspaceID string               `json:"workspace_id"`
	Data        interface{}          `json:"data"`
	Callback    func(error)          `json:"-"`
	Timeout     time.Duration        `json:"timeout"`
	Retries     int                  `json:"retries"`
	Context     context.Context      `json:"-"`
	Cancel      context.CancelFunc   `json:"-"`
}

// DockerWorkspaceService Docker와 통합된 워크스페이스 서비스
type DockerWorkspaceService struct {
	// 기본 서비스
	baseService   WorkspaceService
	storage       interfaces.Storage
	
	// Docker 관리 컴포넌트
	dockerManager *docker.Manager
	statusTracker *status.Tracker
	isolationMgr  *security.IsolationManager
	
	// 비동기 작업 처리
	taskQueue     chan *WorkspaceTask
	workers       int
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	mu            sync.RWMutex
	
	// 배치 작업 관리
	batchJobs     map[string]*BatchJob
	batchMu       sync.RWMutex
}

// BatchJob 배치 작업 정의
type BatchJob struct {
	ID           string                 `json:"id"`
	Operation    string                 `json:"operation"`
	WorkspaceIDs []string               `json:"workspace_ids"`
	Status       BatchJobStatus         `json:"status"`
	Progress     BatchJobProgress       `json:"progress"`
	Results      map[string]interface{} `json:"results"`
	Errors       []string               `json:"errors"`
	StartTime    time.Time              `json:"start_time"`
	EndTime      *time.Time             `json:"end_time,omitempty"`
}

// BatchJobStatus 배치 작업 상태
type BatchJobStatus string

const (
	BatchStatusPending    BatchJobStatus = "pending"
	BatchStatusInProgress BatchJobStatus = "in_progress"
	BatchStatusCompleted  BatchJobStatus = "completed"
	BatchStatusFailed     BatchJobStatus = "failed"
	BatchStatusCancelled  BatchJobStatus = "cancelled"
)

// BatchJobProgress 배치 작업 진행 상황
type BatchJobProgress struct {
	Total     int `json:"total"`
	Completed int `json:"completed"`
	Failed    int `json:"failed"`
	Skipped   int `json:"skipped"`
}

// CreateContainerRequest Docker 컨테이너 생성 요청
type CreateContainerRequest struct {
	WorkspaceID string            `json:"workspace_id"`
	Name        string            `json:"name"`
	ProjectPath string            `json:"project_path"`
	Image       string            `json:"image"`
	Environment map[string]string `json:"environment,omitempty"`
	Volumes     []string          `json:"volumes,omitempty"`
}

// NewDockerWorkspaceService 새로운 Docker 통합 워크스페이스 서비스를 생성합니다
func NewDockerWorkspaceService(
	baseService WorkspaceService,
	storage interfaces.Storage,
	dockerManager *docker.Manager,
) *DockerWorkspaceService {
	
	ctx, cancel := context.WithCancel(context.Background())
	
	service := &DockerWorkspaceService{
		baseService:   baseService,
		storage:       storage,
		dockerManager: dockerManager,
		taskQueue:     make(chan *WorkspaceTask, 100),
		workers:       3, // 기본 3개 워커
		ctx:           ctx,
		cancel:        cancel,
		batchJobs:     make(map[string]*BatchJob),
	}
	
	// 상태 추적기 초기화 (선택적)
	// TODO: 상태 추적기 통합 구현
	service.statusTracker = nil
	
	// 격리 관리자 초기화
	service.isolationMgr = security.NewIsolationManager()
	
	// 워커 시작
	service.startWorkers()
	
	return service
}

// startWorkers 비동기 작업 워커들을 시작합니다
func (dws *DockerWorkspaceService) startWorkers() {
	for i := 0; i < dws.workers; i++ {
		dws.wg.Add(1)
		go dws.worker(i)
	}
}

// worker 개별 워커 고루틴
func (dws *DockerWorkspaceService) worker(id int) {
	defer dws.wg.Done()
	
	for {
		select {
		case task := <-dws.taskQueue:
			if task == nil {
				return // 채널이 닫힘
			}
			err := dws.executeTask(task)
			if task.Callback != nil {
				task.Callback(err)
			}
			
		case <-dws.ctx.Done():
			return
		}
	}
}

// CreateWorkspace Docker 컨테이너와 함께 워크스페이스를 생성합니다
func (dws *DockerWorkspaceService) CreateWorkspace(ctx context.Context, req *models.CreateWorkspaceRequest, ownerID string) (*models.Workspace, error) {
	// Phase 1: 기본 워크스페이스 생성 (DB)
	workspace, err := dws.baseService.CreateWorkspace(ctx, req, ownerID)
	if err != nil {
		return nil, fmt.Errorf("create base workspace: %w", err)
	}
	
	// Phase 2: Docker 컨테이너 생성 (비동기)
	createTask := &WorkspaceTask{
		Type:        TaskTypeCreate,
		WorkspaceID: workspace.ID,
		Data: CreateContainerRequest{
			WorkspaceID: workspace.ID,
			Name:        fmt.Sprintf("workspace-%s", workspace.ID),
			ProjectPath: workspace.ProjectPath,
			Image:       "aicli/workspace:latest", // 기본 워크스페이스 이미지
			Environment: map[string]string{
				"WORKSPACE_ID":   workspace.ID,
				"WORKSPACE_NAME": workspace.Name,
				"PROJECT_PATH":   workspace.ProjectPath,
			},
		},
		Timeout: 60 * time.Second,
		Retries: 3,
		Context: ctx,
	}
	
	// 동기 실행을 위한 채널
	resultChan := make(chan error, 1)
	createTask.Callback = func(err error) {
		resultChan <- err
	}
	
	// 작업 큐에 추가
	select {
	case dws.taskQueue <- createTask:
		// 큐에 추가 성공
	default:
		// 큐가 가득 찬 경우 즉시 실행
		go func() {
			err := dws.executeTask(createTask)
			createTask.Callback(err)
		}()
	}
	
	// 컨테이너 생성 대기 (5초 타임아웃)
	select {
	case err := <-resultChan:
		if err != nil {
			// 컨테이너 생성 실패 시 DB에서 워크스페이스 삭제
			if delErr := dws.baseService.DeleteWorkspace(ctx, workspace.ID, ownerID); delErr != nil {
				return nil, fmt.Errorf("create container failed and cleanup failed: %v (cleanup error: %v)", err, delErr)
			}
			return nil, fmt.Errorf("create container: %w", err)
		}
	case <-time.After(5 * time.Second):
		// 타임아웃 - 백그라운드에서 진행하되 상태를 비활성화로 변경
		workspace.Status = models.WorkspaceStatusInactive
		if err := dws.storage.Workspace().Update(ctx, workspace.ID, map[string]interface{}{
			"status": models.WorkspaceStatusInactive,
		}); err != nil {
			// 로그만 남기고 계속 진행
			fmt.Printf("failed to update workspace status to inactive: %v\n", err)
		}
	}
	
	return workspace, nil
}

// GetWorkspace 워크스페이스 정보를 조회합니다 (기본 서비스 위임)
func (dws *DockerWorkspaceService) GetWorkspace(ctx context.Context, id string, ownerID string) (*models.Workspace, error) {
	return dws.baseService.GetWorkspace(ctx, id, ownerID)
}

// UpdateWorkspace 워크스페이스를 수정합니다 (기본 서비스 위임)
func (dws *DockerWorkspaceService) UpdateWorkspace(ctx context.Context, id string, req *models.UpdateWorkspaceRequest, ownerID string) (*models.Workspace, error) {
	return dws.baseService.UpdateWorkspace(ctx, id, req, ownerID)
}

// DeleteWorkspace Docker 컨테이너와 함께 워크스페이스를 삭제합니다
func (dws *DockerWorkspaceService) DeleteWorkspace(ctx context.Context, id string, ownerID string) error {
	// Phase 1: 컨테이너 정리
	deleteTask := &WorkspaceTask{
		Type:        TaskTypeDelete,
		WorkspaceID: id,
		Timeout:     30 * time.Second,
		Retries:     2,
		Context:     ctx,
	}
	
	// 동기 실행을 위한 채널
	resultChan := make(chan error, 1)
	deleteTask.Callback = func(err error) {
		resultChan <- err
	}
	
	// 작업 큐에 추가
	select {
	case dws.taskQueue <- deleteTask:
		// 큐에 추가 성공
	default:
		// 큐가 가득 찬 경우 즉시 실행
		go func() {
			err := dws.executeTask(deleteTask)
			deleteTask.Callback(err)
		}()
	}
	
	// 컨테이너 삭제 대기 (10초 타임아웃)
	select {
	case containerErr := <-resultChan:
		if containerErr != nil {
			// 컨테이너 삭제 실패는 로그만 남기고 DB 삭제는 계속 진행
			fmt.Printf("failed to delete container for workspace %s: %v\n", id, containerErr)
		}
	case <-time.After(10 * time.Second):
		// 컨테이너 삭제 타임아웃은 무시하고 DB 삭제 진행
		fmt.Printf("container deletion timeout for workspace %s\n", id)
	}
	
	// Phase 2: DB에서 워크스페이스 삭제
	return dws.baseService.DeleteWorkspace(ctx, id, ownerID)
}

// ListWorkspaces 워크스페이스 목록을 조회합니다 (기본 서비스 위임)
func (dws *DockerWorkspaceService) ListWorkspaces(ctx context.Context, ownerID string, req *models.PaginationRequest) (*models.WorkspaceListResponse, error) {
	return dws.baseService.ListWorkspaces(ctx, ownerID, req)
}

// ValidateWorkspace 워크스페이스를 검증합니다 (기본 서비스 위임)
func (dws *DockerWorkspaceService) ValidateWorkspace(ctx context.Context, workspace *models.Workspace) error {
	return dws.baseService.ValidateWorkspace(ctx, workspace)
}

// ActivateWorkspace 워크스페이스를 활성화합니다
func (dws *DockerWorkspaceService) ActivateWorkspace(ctx context.Context, id string, ownerID string) error {
	// 컨테이너 시작 작업 추가
	startTask := &WorkspaceTask{
		Type:        TaskTypeStart,
		WorkspaceID: id,
		Timeout:     30 * time.Second,
		Retries:     2,
		Context:     ctx,
	}
	
	// 비동기 실행
	select {
	case dws.taskQueue <- startTask:
		// 큐에 추가 성공
	default:
		return fmt.Errorf("task queue is full, cannot activate workspace")
	}
	
	return dws.baseService.ActivateWorkspace(ctx, id, ownerID)
}

// DeactivateWorkspace 워크스페이스를 비활성화합니다
func (dws *DockerWorkspaceService) DeactivateWorkspace(ctx context.Context, id string, ownerID string) error {
	// 컨테이너 중지 작업 추가
	stopTask := &WorkspaceTask{
		Type:        TaskTypeStop,
		WorkspaceID: id,
		Timeout:     30 * time.Second,
		Retries:     2,
		Context:     ctx,
	}
	
	// 비동기 실행
	select {
	case dws.taskQueue <- stopTask:
		// 큐에 추가 성공
	default:
		return fmt.Errorf("task queue is full, cannot deactivate workspace")
	}
	
	return dws.baseService.DeactivateWorkspace(ctx, id, ownerID)
}

// ArchiveWorkspace 워크스페이스를 아카이브합니다 (기본 서비스 위임)
func (dws *DockerWorkspaceService) ArchiveWorkspace(ctx context.Context, id string, ownerID string) error {
	return dws.baseService.ArchiveWorkspace(ctx, id, ownerID)
}

// UpdateActiveTaskCount 활성 태스크 수를 업데이트합니다 (기본 서비스 위임)
func (dws *DockerWorkspaceService) UpdateActiveTaskCount(ctx context.Context, id string, delta int) error {
	return dws.baseService.UpdateActiveTaskCount(ctx, id, delta)
}

// GetWorkspaceStats 워크스페이스 통계를 조회합니다 (기본 서비스 위임)
func (dws *DockerWorkspaceService) GetWorkspaceStats(ctx context.Context, ownerID string) (*WorkspaceStats, error) {
	return dws.baseService.GetWorkspaceStats(ctx, ownerID)
}

// executeTask 개별 작업을 실행합니다
func (dws *DockerWorkspaceService) executeTask(task *WorkspaceTask) error {
	ctx := task.Context
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), task.Timeout)
		defer cancel()
	}
	
	switch task.Type {
	case TaskTypeCreate:
		return dws.executeCreateTask(ctx, task)
	case TaskTypeStart:
		return dws.executeStartTask(ctx, task)
	case TaskTypeStop:
		return dws.executeStopTask(ctx, task)
	case TaskTypeRestart:
		return dws.executeRestartTask(ctx, task)
	case TaskTypeDelete:
		return dws.executeDeleteTask(ctx, task)
	case TaskTypeSync:
		return dws.executeSyncTask(ctx, task)
	default:
		return fmt.Errorf("unknown task type: %s", task.Type)
	}
}

// executeCreateTask 컨테이너 생성 작업을 실행합니다
func (dws *DockerWorkspaceService) executeCreateTask(ctx context.Context, task *WorkspaceTask) error {
	req, ok := task.Data.(CreateContainerRequest)
	if !ok {
		return fmt.Errorf("invalid create task data type")
	}
	
	// Step 1: 워크스페이스 정보 조회
	workspace, err := dws.storage.Workspace().GetByID(ctx, task.WorkspaceID)
	if err != nil {
		return fmt.Errorf("get workspace: %w", err)
	}
	
	// Step 2: 격리 설정 생성
	// TODO: isolation 설정을 컨테이너 생성 요청에 추가
	_, err = dws.isolationMgr.CreateWorkspaceIsolation(workspace)
	if err != nil {
		return fmt.Errorf("create isolation config: %w", err)
	}
	
	// Step 3: 컨테이너 생성
	container, err := dws.dockerManager.Container().CreateWorkspaceContainer(ctx, &docker.CreateContainerRequest{
		WorkspaceID: req.WorkspaceID,
		Name:        req.Name,
		Image:       req.Image,
		ProjectPath: req.ProjectPath,
		Environment: req.Environment,
		WorkingDir:  "/workspace",
		CPULimit:    1.0,
		MemoryLimit: 1024 * 1024 * 1024, // 1GB
	})
	if err != nil {
		return fmt.Errorf("create container: %w", err)
	}
	
	// Step 4: 컨테이너 시작
	if err := dws.dockerManager.Container().StartContainer(ctx, container.ID); err != nil {
		// 실패 시 컨테이너 삭제
		if removeErr := dws.dockerManager.Container().RemoveContainer(ctx, container.ID, true); removeErr != nil {
			return fmt.Errorf("start container failed and cleanup failed: %v (cleanup error: %v)", err, removeErr)
		}
		return fmt.Errorf("start container: %w", err)
	}
	
	// Step 5: 데이터베이스 상태 업데이트
	updates := map[string]interface{}{
		"status":       models.WorkspaceStatusActive,
		"active_tasks": workspace.ActiveTasks + 1,
		"updated_at":   time.Now(),
	}
	
	if err := dws.storage.Workspace().Update(ctx, task.WorkspaceID, updates); err != nil {
		return fmt.Errorf("update workspace status: %w", err)
	}
	
	return nil
}

// executeStartTask 컨테이너 시작 작업을 실행합니다
func (dws *DockerWorkspaceService) executeStartTask(ctx context.Context, task *WorkspaceTask) error {
	containers, err := dws.dockerManager.Container().ListWorkspaceContainers(ctx, task.WorkspaceID)
	if err != nil {
		return fmt.Errorf("list workspace containers: %w", err)
	}
	
	for _, container := range containers {
		if err := dws.dockerManager.Container().StartContainer(ctx, container.ID); err != nil {
			return fmt.Errorf("start container %s: %w", container.ID, err)
		}
	}
	
	return nil
}

// executeStopTask 컨테이너 중지 작업을 실행합니다
func (dws *DockerWorkspaceService) executeStopTask(ctx context.Context, task *WorkspaceTask) error {
	containers, err := dws.dockerManager.Container().ListWorkspaceContainers(ctx, task.WorkspaceID)
	if err != nil {
		return fmt.Errorf("list workspace containers: %w", err)
	}
	
	for _, container := range containers {
		if err := dws.dockerManager.Container().StopContainer(ctx, container.ID, 10*time.Second); err != nil {
			return fmt.Errorf("stop container %s: %w", container.ID, err)
		}
	}
	
	return nil
}

// executeRestartTask 컨테이너 재시작 작업을 실행합니다
func (dws *DockerWorkspaceService) executeRestartTask(ctx context.Context, task *WorkspaceTask) error {
	containers, err := dws.dockerManager.Container().ListWorkspaceContainers(ctx, task.WorkspaceID)
	if err != nil {
		return fmt.Errorf("list workspace containers: %w", err)
	}
	
	for _, container := range containers {
		if err := dws.dockerManager.Container().RestartContainer(ctx, container.ID, 10*time.Second); err != nil {
			return fmt.Errorf("restart container %s: %w", container.ID, err)
		}
	}
	
	return nil
}

// executeDeleteTask 컨테이너 삭제 작업을 실행합니다
func (dws *DockerWorkspaceService) executeDeleteTask(ctx context.Context, task *WorkspaceTask) error {
	containers, err := dws.dockerManager.Container().ListWorkspaceContainers(ctx, task.WorkspaceID)
	if err != nil {
		return fmt.Errorf("list workspace containers: %w", err)
	}
	
	for _, container := range containers {
		// 컨테이너 중지 후 삭제
		if err := dws.dockerManager.Container().StopContainer(ctx, container.ID, 10*time.Second); err != nil {
			// 중지 실패는 로그만 남기고 계속 진행
			fmt.Printf("failed to stop container %s before deletion: %v\n", container.ID, err)
		}
		
		if err := dws.dockerManager.Container().RemoveContainer(ctx, container.ID, true); err != nil {
			return fmt.Errorf("remove container %s: %w", container.ID, err)
		}
	}
	
	return nil
}

// executeSyncTask 동기화 작업을 실행합니다
func (dws *DockerWorkspaceService) executeSyncTask(ctx context.Context, task *WorkspaceTask) error {
	// 워크스페이스 컨테이너들의 상태를 DB와 동기화
	containers, err := dws.dockerManager.Container().ListWorkspaceContainers(ctx, task.WorkspaceID)
	if err != nil {
		return fmt.Errorf("list workspace containers: %w", err)
	}
	
	// 컨테이너가 없으면 워크스페이스를 비활성 상태로 변경
	if len(containers) == 0 {
		updates := map[string]interface{}{
			"status":     models.WorkspaceStatusInactive,
			"updated_at": time.Now(),
		}
		return dws.storage.Workspace().Update(ctx, task.WorkspaceID, updates)
	}
	
	// 활성 컨테이너가 있으면 워크스페이스를 활성 상태로 변경
	hasRunning := false
	for _, container := range containers {
		if container.State == docker.ContainerStateRunning {
			hasRunning = true
			break
		}
	}
	
	status := models.WorkspaceStatusInactive
	if hasRunning {
		status = models.WorkspaceStatusActive
	}
	
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}
	
	return dws.storage.Workspace().Update(ctx, task.WorkspaceID, updates)
}

// Close 서비스를 종료하고 모든 리소스를 정리합니다
func (dws *DockerWorkspaceService) Close() error {
	// 컨텍스트 취소
	dws.cancel()
	
	// 태스크 큐 닫기
	close(dws.taskQueue)
	
	// 워커 고루틴들이 종료될 때까지 대기
	dws.wg.Wait()
	
	return nil
}