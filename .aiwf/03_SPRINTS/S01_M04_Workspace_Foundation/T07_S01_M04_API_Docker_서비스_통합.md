# T07_S01_M04_API_Docker_서비스_통합

**태스크 ID**: T07_S01_M04  
**제목**: API-Docker 서비스 통합  
**설명**: 워크스페이스 API와 Docker 관리 서비스 간의 통합 인터페이스 구현  
**우선순위**: 높음  
**복잡도**: 보통  
**예상 소요시간**: 5-6시간  
**상태**: completed  
**시작 시간**: 2025-07-22 09:00:00+0900  
**완료 시간**: 2025-07-22 16:30:00+0900  

## 📋 작업 개요

워크스페이스 API와 Docker 관리 기능을 통합하여, 사용자가 API를 통해 컬테이너 기반 워크스페이스를 생성, 관리, 삭제할 수 있도록 합니다. 전체 시스템을 하나의 응집된 인터페이스로 연결합니다.

## 🎯 목표

1. **서비스 통합**: 워크스페이스 서비스에 Docker 기능 통합
2. **전체 라이프사이클**: 생성부터 삭제까지 완전한 관리 플로우
3. **에러 처리**: Docker 오류 및 복구 로직 통합
4. **상태 동기화**: 실시간 워크스페이스 상태 반영
5. **성능 최적화**: 비동기 작업 및 배치 처리

## 📂 코드베이스 분석

### 의존성
- `T01_S01_M04`: 워크스페이스 서비스 계층 (필수)
- `T03_S01_M04`: 컬테이너 생명주기 관리자 (필수)
- `T05_S01_M04`: 상태 추적 시스템 (선택적)
- `T06_S01_M04`: 격리 환경 설정 (선택적)

### 기존 API 구조
```go
// internal/api/controllers/workspace.go - 기존 컨트롤러 확장
func (wc *WorkspaceController) CreateWorkspace(c *gin.Context) {
    // 기존: 데이터베이스에만 저장
    // 신규: Docker 컬테이너도 함께 생성
}
```

### 구현 위치
```
internal/services/
├── workspace.go            # 기존 서비스 확장
└── docker_workspace.go     # Docker 통합 서비스 (새로 생성)

internal/api/controllers/
└── workspace.go            # 기존 컨트롤러 수정
```

## 🛠️ 기술 가이드

### 1. Docker 통합 서비스 계층

```go
// internal/services/docker_workspace.go
package services

import (
    "context"
    "fmt"
    "time"
    
    "github.com/aicli/aicli-web/internal/models"
    "github.com/aicli/aicli-web/internal/storage"
    "github.com/aicli/aicli-web/internal/docker"
    "github.com/aicli/aicli-web/internal/docker/status"
    "github.com/aicli/aicli-web/internal/docker/security"
)

// Docker와 통합된 워크스페이스 서비스
type DockerWorkspaceService struct {
    // 기본 서비스
    baseService    WorkspaceService
    storage        storage.Storage
    
    // Docker 관리 컴포넌트
    containerMgr   *docker.ContainerManager
    statusTracker  *status.Tracker
    isolationMgr   *security.IsolationManager
    
    // 비동기 작업 처리
    taskQueue      chan *WorkspaceTask
    workers        int
}

type WorkspaceTask struct {
    Type        TaskType    `json:"type"`
    WorkspaceID string      `json:"workspace_id"`
    Data        interface{} `json:"data"`
    Callback    func(error) `json:"-"`
    Timeout     time.Duration `json:"timeout"`
    Retries     int         `json:"retries"`
}

type TaskType string

const (
    TaskTypeCreate    TaskType = "create"
    TaskTypeStart     TaskType = "start"
    TaskTypeStop      TaskType = "stop"
    TaskTypeRestart   TaskType = "restart"
    TaskTypeDelete    TaskType = "delete"
    TaskTypeSync      TaskType = "sync"
)

func NewDockerWorkspaceService(
    baseService WorkspaceService,
    storage storage.Storage,
    containerMgr *docker.ContainerManager,
    statusTracker *status.Tracker,
    isolationMgr *security.IsolationManager,
) *DockerWorkspaceService {
    
    service := &DockerWorkspaceService{
        baseService:   baseService,
        storage:       storage,
        containerMgr:  containerMgr,
        statusTracker: statusTracker,
        isolationMgr:  isolationMgr,
        taskQueue:     make(chan *WorkspaceTask, 100),
        workers:       3, // 기본 3개 워커
    }
    
    // 워커 시작
    service.startWorkers()
    
    return service
}

func (dws *DockerWorkspaceService) startWorkers() {
    for i := 0; i < dws.workers; i++ {
        go dws.worker(i)
    }
}

func (dws *DockerWorkspaceService) worker(id int) {
    for task := range dws.taskQueue {
        err := dws.executeTask(task)
        if task.Callback != nil {
            task.Callback(err)
        }
    }
}
```

### 2. 전체 라이프사이클 관리

```go
// 워크스페이스 생성 (전체 플로우)
func (dws *DockerWorkspaceService) CreateWorkspace(ctx context.Context, req *models.CreateWorkspaceRequest, ownerID string) (*models.Workspace, error) {
    // Phase 1: 기본 워크스페이스 생성 (DB)
    workspace, err := dws.baseService.CreateWorkspace(ctx, req, ownerID)
    if err != nil {
        return nil, fmt.Errorf("create base workspace: %w", err)
    }
    
    // Phase 2: Docker 컬테이너 생성 (비동기)
    createTask := &WorkspaceTask{
        Type:        TaskTypeCreate,
        WorkspaceID: workspace.ID,
        Data: docker.CreateContainerRequest{
            WorkspaceID: workspace.ID,
            Name:        workspace.Name,
            ProjectPath: workspace.ProjectPath,
            Image:       "alpine:latest", // 기본 이미지
        },
        Timeout: 60 * time.Second,
        Retries: 3,
    }
    
    // 동기 실행 (응답 시간 단축을 위해)
    resultChan := make(chan error, 1)
    createTask.Callback = func(err error) {
        resultChan <- err
    }
    
    select {
    case dws.taskQueue <- createTask:
        // 작업 대기열 추가 성공
    default:
        // 대기열이 가득 찬 경우 즉시 실행
        go dws.executeTask(createTask)
    }
    
    // 컬테이너 생성 대기 (5초 타임아웃)
    select {
    case err := <-resultChan:
        if err != nil {
            // 컨테이너 생성 실패 시 DB에서 워크스페이스 삭제
            dws.baseService.DeleteWorkspace(ctx, workspace.ID, ownerID)
            return nil, fmt.Errorf("create container: %w", err)
        }
    case <-time.After(5 * time.Second):
        // 타임아웃 - 백그라운드에서 진행
        workspace.Status = models.WorkspaceStatusInactive
        dws.storage.Workspace().Update(ctx, workspace.ID, map[string]interface{}{
            "status": models.WorkspaceStatusInactive,
        })
    }
    
    return workspace, nil
}

// 작업 실행 로직
func (dws *DockerWorkspaceService) executeTask(task *WorkspaceTask) error {
    ctx, cancel := context.WithTimeout(context.Background(), task.Timeout)
    defer cancel()
    
    switch task.Type {
    case TaskTypeCreate:
        return dws.executeCreateTask(ctx, task)
    case TaskTypeStart:
        return dws.executeStartTask(ctx, task)
    case TaskTypeStop:
        return dws.executeStopTask(ctx, task)
    case TaskTypeDelete:
        return dws.executeDeleteTask(ctx, task)
    case TaskTypeSync:
        return dws.executeSyncTask(ctx, task)
    default:
        return fmt.Errorf("unknown task type: %s", task.Type)
    }
}

func (dws *DockerWorkspaceService) executeCreateTask(ctx context.Context, task *WorkspaceTask) error {
    req := task.Data.(docker.CreateContainerRequest)
    
    // Step 1: 격리 설정 생성
    workspace, err := dws.storage.Workspace().GetByID(ctx, task.WorkspaceID)
    if err != nil {
        return fmt.Errorf("get workspace: %w", err)
    }
    
    isolation, err := dws.isolationMgr.CreateWorkspaceIsolation(workspace)
    if err != nil {
        return fmt.Errorf("create isolation config: %w", err)
    }
    
    // Step 2: 컬테이너 생성
    container, err := dws.containerMgr.CreateWorkspaceContainer(ctx, &req)
    if err != nil {
        return fmt.Errorf("create container: %w", err)
    }
    
    // Step 3: 컨테이너 시작
    if err := dws.containerMgr.StartContainer(ctx, container.ID); err != nil {
        // 실패 시 컨테이너 삭제
        dws.containerMgr.RemoveContainer(ctx, container.ID, true)
        return fmt.Errorf("start container: %w", err)
    }
    
    // Step 4: 데이터베이스 상태 업데이트
    updates := map[string]interface{}{
        "status": models.WorkspaceStatusActive,
        "active_tasks": workspace.ActiveTasks + 1,
    }
    
    return dws.storage.Workspace().Update(ctx, task.WorkspaceID, updates)
}
```

### 3. 에러 처리 및 복구

```go
// 에러 처리 전략
type ErrorRecoveryStrategy struct {
    maxRetries      int
    backoffDuration time.Duration
    fallbackAction  string // "stop", "remove", "ignore"
}

func (dws *DockerWorkspaceService) handleWorkspaceError(workspaceID string, err error, strategy *ErrorRecoveryStrategy) error {
    workspace, getErr := dws.storage.Workspace().GetByID(context.Background(), workspaceID)
    if getErr != nil {
        return fmt.Errorf("get workspace for error handling: %w", getErr)
    }
    
    // 에러 로깅
    dws.logWorkspaceError(workspace, err)
    
    // 에러 유형별 처리
    switch {
    case docker.IsDockerError(err):
        return dws.handleDockerError(workspace, err, strategy)
    case isNetworkError(err):
        return dws.handleNetworkError(workspace, err, strategy)
    case isStorageError(err):
        return dws.handleStorageError(workspace, err, strategy)
    default:
        return dws.handleGenericError(workspace, err, strategy)
    }
}

func (dws *DockerWorkspaceService) handleDockerError(workspace *models.Workspace, err error, strategy *ErrorRecoveryStrategy) error {
    // Docker 특이적 에러 처리
    containers, listErr := dws.containerMgr.ListWorkspaceContainers(context.Background(), workspace.ID)
    if listErr != nil {
        return listErr
    }
    
    // 컬테이너 상태에 따른 복구 전략
    for _, container := range containers {
        switch container.State {
        case docker.ContainerStateExited:
            // 재시작 시도
            return dws.restartWorkspace(workspace.ID)
        case docker.ContainerStateDead:
            // 컬테이너 삭제 후 재생성
            return dws.recreateWorkspace(workspace.ID)
        }
    }
    
    return nil
}

func (dws *DockerWorkspaceService) restartWorkspace(workspaceID string) error {
    task := &WorkspaceTask{
        Type:        TaskTypeRestart,
        WorkspaceID: workspaceID,
        Timeout:     30 * time.Second,
        Retries:     2,
    }
    
    select {
    case dws.taskQueue <- task:
        return nil
    default:
        return fmt.Errorf("task queue full, cannot restart workspace")
    }
}

func (dws *DockerWorkspaceService) recreateWorkspace(workspaceID string) error {
    // 기존 컬테이너 정리
    cleanupTask := &WorkspaceTask{
        Type:        TaskTypeDelete,
        WorkspaceID: workspaceID,
        Timeout:     30 * time.Second,
    }
    
    // 새 컬테이너 생성
    createTask := &WorkspaceTask{
        Type:        TaskTypeCreate,
        WorkspaceID: workspaceID,
        Timeout:     60 * time.Second,
        Retries:     2,
    }
    
    // 순차적 실행
    dws.taskQueue <- cleanupTask
    dws.taskQueue <- createTask
    
    return nil
}
```

### 4. API 컨트롤러 통합

```go
// internal/api/controllers/workspace.go (수정)
// Docker 통합 서비스를 사용하도록 컨트롤러 수정
type WorkspaceController struct {
    service        services.WorkspaceService
    dockerService  *services.DockerWorkspaceService // 추가
}

func NewWorkspaceController(service services.WorkspaceService, dockerService *services.DockerWorkspaceService) *WorkspaceController {
    return &WorkspaceController{
        service:       service,
        dockerService: dockerService,
    }
}

// 워크스페이스 생성 API (수정)
func (wc *WorkspaceController) CreateWorkspace(c *gin.Context) {
    claims := c.MustGet("claims").(*auth.Claims)
    
    var req models.CreateWorkspaceRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        middleware.ValidationError(c, "요청 데이터가 올바르지 않습니다", err.Error())
        return
    }
    
    // Docker 통합 서비스 사용
    workspace, err := wc.dockerService.CreateWorkspace(c, &req, claims.UserID)
    if err != nil {
        middleware.HandleServiceError(c, err)
        return
    }
    
    c.JSON(http.StatusCreated, models.SuccessResponse{
        Success: true,
        Message: "워크스페이스가 생성되었습니다 (컬테이너 설정 중)",
        Data:    workspace,
    })
}

// 워크스페이스 상태 조회 API (신규)
func (wc *WorkspaceController) GetWorkspaceStatus(c *gin.Context) {
    claims := c.MustGet("claims").(*auth.Claims)
    workspaceID := c.Param("id")
    
    // 기본 워크스페이스 정보
    workspace, err := wc.service.GetWorkspace(c, workspaceID, claims.UserID)
    if err != nil {
        middleware.HandleServiceError(c, err)
        return
    }
    
    // Docker 컨테이너 상태
    status, err := wc.dockerService.GetWorkspaceStatus(c, workspaceID)
    if err != nil {
        // Docker 상태 조회 실패는 경고만 출력
        status = &WorkspaceStatus{
            ContainerState: "unknown",
            LastError:      err.Error(),
        }
    }
    
    response := WorkspaceStatusResponse{
        Workspace:       workspace,
        ContainerStatus: status,
        LastUpdated:     time.Now(),
    }
    
    c.JSON(http.StatusOK, models.SuccessResponse{
        Success: true,
        Data:    response,
    })
}

type WorkspaceStatusResponse struct {
    Workspace       *models.Workspace `json:"workspace"`
    ContainerStatus *WorkspaceStatus  `json:"container_status"`
    LastUpdated     time.Time         `json:"last_updated"`
}

type WorkspaceStatus struct {
    ContainerID    string                    `json:"container_id,omitempty"`
    ContainerState docker.ContainerState     `json:"container_state"`
    Uptime         string                    `json:"uptime,omitempty"`
    Metrics        *status.WorkspaceMetrics  `json:"metrics,omitempty"`
    LastError      string                    `json:"last_error,omitempty"`
}
```

### 5. 비동기 작업 관리

```go
// 대량 워크스페이스 처리를 위한 배치 API
func (wc *WorkspaceController) BatchWorkspaceOperation(c *gin.Context) {
    claims := c.MustGet("claims").(*auth.Claims)
    
    var req BatchOperationRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        middleware.ValidationError(c, "요청 데이터가 올바르지 않습니다", err.Error())
        return
    }
    
    // 비동기 배치 작업 시작
    batchID, err := wc.dockerService.StartBatchOperation(c, &req, claims.UserID)
    if err != nil {
        middleware.HandleServiceError(c, err)
        return
    }
    
    c.JSON(http.StatusAccepted, models.SuccessResponse{
        Success: true,
        Message: fmt.Sprintf("배치 작업이 시작되었습니다. 배치 ID: %s", batchID),
        Data: gin.H{
            "batch_id":       batchID,
            "operation_type": req.Operation,
            "workspace_count": len(req.WorkspaceIDs),
            "status_url":     fmt.Sprintf("/api/workspaces/batch/%s/status", batchID),
        },
    })
}

type BatchOperationRequest struct {
    Operation    string   `json:"operation" binding:"required,oneof=start stop restart delete"`
    WorkspaceIDs []string `json:"workspace_ids" binding:"required,min=1"`
    Options      map[string]interface{} `json:"options,omitempty"`
}

// 배치 작업 상태 조회
func (wc *WorkspaceController) GetBatchOperationStatus(c *gin.Context) {
    batchID := c.Param("batch_id")
    
    status, err := wc.dockerService.GetBatchOperationStatus(c, batchID)
    if err != nil {
        middleware.HandleServiceError(c, err)
        return
    }
    
    c.JSON(http.StatusOK, models.SuccessResponse{
        Success: true,
        Data:    status,
    })
}
```

## ✅ 완료 기준

### 기능적 요구사항
- [ ] 워크스페이스 서비스에 Docker 기능 통합
- [ ] API를 통한 전체 워크스페이스 라이프사이클 관리
- [ ] 비동기 작업 처리 및 상태 추적
- [ ] Docker 오류 처리 및 자동 복구
- [ ] 배치 작업 지원 (대량 워크스페이스 일괄 처리)

### 비기능적 요구사항
- [ ] 워크스페이스 생성 API 응답 시간 < 3초
- [ ] Docker 작업 오류율 < 1%
- [ ] 비동기 작업 지연 시간 < 10초
- [ ] 동시 처리 가능 워크스페이스 수 > 50개

## 🧪 테스트 전략

### 1. 단위 테스트
```go
func TestDockerWorkspaceService_CreateWorkspace(t *testing.T) {
    // Mock 서비스 들 설정
    mockBase := &MockWorkspaceService{}
    mockContainer := &MockContainerManager{}
    mockTracker := &MockStatusTracker{}
    
    service := NewDockerWorkspaceService(mockBase, nil, mockContainer, mockTracker, nil)
    
    req := &models.CreateWorkspaceRequest{
        Name:        "test-workspace",
        ProjectPath: "/tmp/test",
    }
    
    workspace, err := service.CreateWorkspace(context.Background(), req, "user1")
    
    assert.NoError(t, err)
    assert.Equal(t, models.WorkspaceStatusActive, workspace.Status)
    
    // Mock 호출 검증
    assert.True(t, mockContainer.CreateCalled)
    assert.True(t, mockContainer.StartCalled)
}

func TestWorkspaceController_BatchOperation(t *testing.T) {
    controller := setupTestController(t)
    
    req := BatchOperationRequest{
        Operation:    "start",
        WorkspaceIDs: []string{"ws1", "ws2", "ws3"},
    }
    
    w := httptest.NewRecorder()
    c, _ := gin.CreateTestContext(w)
    
    controller.BatchWorkspaceOperation(c)
    
    assert.Equal(t, http.StatusAccepted, w.Code)
    var response map[string]interface{}
    json.Unmarshal(w.Body.Bytes(), &response)
    assert.NotEmpty(t, response["data"].(map[string]interface{})["batch_id"])
}
```

### 2. 통합 테스트
- 실제 Docker daemon과 통합 테스트
- 전체 워크스페이스 라이프사이클 테스트
- 오류 상황에서의 복구 동작 검증
- 동시 다중 워크스페이스 작업 테스트

## 📝 구현 단계

1. **Phase 1**: Docker 통합 서비스 기본 구조 (1.5시간)
2. **Phase 2**: 전체 라이프사이클 관리 로직 (2시간)
3. **Phase 3**: 에러 처리 및 복구 메커니즘 (1시간)
4. **Phase 4**: API 컨트롤러 통합 및 배치 작업 (1시간)
5. **Phase 5**: 테스트 작성 및 검증 (0.5시간)

## 🔗 연관 태스크

- **의존성**: T01_S01_M04 (서비스), T03_S01_M04 (컬테이너), T05_S01_M04 (상태 추적)
- **후속 작업**: T08_S01_M04 (통합 테스트), 향후 Claude CLI 통합
- **비동기 작업**: 전체 Docker 관리 시스템과 통합

## 📚 참고 자료

- [기존 워크스페이스 API](/internal/api/controllers/workspace.go)
- [Docker 컬테이너 관리자](/internal/docker/container_manager.go)
- [Go 비동기 패턴](https://golang.org/doc/effective_go#goroutines)
- [RESTful API Design](https://restfulapi.net/)