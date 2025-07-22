package services

import (
	"context"
	"fmt"
	"sync"
	"time"
	
	"github.com/google/uuid"
)

// BatchOperationRequest 배치 작업 요청
type BatchOperationRequest struct {
	Operation    string                 `json:"operation" binding:"required,oneof=start stop restart delete"`
	WorkspaceIDs []string               `json:"workspace_ids" binding:"required,min=1"`
	Options      map[string]interface{} `json:"options,omitempty"`
}

// StartBatchOperation 배치 작업을 시작합니다
func (dws *DockerWorkspaceService) StartBatchOperation(ctx context.Context, req *BatchOperationRequest, ownerID string) (string, error) {
	// 배치 ID 생성
	batchID := uuid.New().String()
	
	// 배치 작업 생성
	batch := &BatchJob{
		ID:           batchID,
		Operation:    req.Operation,
		WorkspaceIDs: req.WorkspaceIDs,
		Status:       BatchStatusPending,
		Progress: BatchJobProgress{
			Total:     len(req.WorkspaceIDs),
			Completed: 0,
			Failed:    0,
			Skipped:   0,
		},
		Results:   make(map[string]interface{}),
		Errors:    make([]string, 0),
		StartTime: time.Now(),
	}
	
	// 배치 작업 등록
	dws.batchMu.Lock()
	dws.batchJobs[batchID] = batch
	dws.batchMu.Unlock()
	
	// 백그라운드에서 배치 작업 실행
	go dws.executeBatchOperation(ctx, batch, ownerID)
	
	return batchID, nil
}

// GetBatchOperationStatus 배치 작업 상태를 조회합니다
func (dws *DockerWorkspaceService) GetBatchOperationStatus(ctx context.Context, batchID string) (*BatchJob, error) {
	dws.batchMu.RLock()
	defer dws.batchMu.RUnlock()
	
	batch, exists := dws.batchJobs[batchID]
	if !exists {
		return nil, fmt.Errorf("batch job not found: %s", batchID)
	}
	
	// 복사본 반환 (동시성 보호)
	batchCopy := *batch
	return &batchCopy, nil
}

// CancelBatchOperation 배치 작업을 취소합니다
func (dws *DockerWorkspaceService) CancelBatchOperation(ctx context.Context, batchID string) error {
	dws.batchMu.Lock()
	defer dws.batchMu.Unlock()
	
	batch, exists := dws.batchJobs[batchID]
	if !exists {
		return fmt.Errorf("batch job not found: %s", batchID)
	}
	
	if batch.Status == BatchStatusCompleted || batch.Status == BatchStatusFailed {
		return fmt.Errorf("cannot cancel completed batch job: %s", batchID)
	}
	
	batch.Status = BatchStatusCancelled
	now := time.Now()
	batch.EndTime = &now
	
	return nil
}

// ListBatchOperations 배치 작업 목록을 조회합니다
func (dws *DockerWorkspaceService) ListBatchOperations(ctx context.Context, limit int) ([]*BatchJob, error) {
	dws.batchMu.RLock()
	defer dws.batchMu.RUnlock()
	
	var jobs []*BatchJob
	count := 0
	
	// 최신 순으로 정렬하여 반환
	for _, batch := range dws.batchJobs {
		if limit > 0 && count >= limit {
			break
		}
		batchCopy := *batch
		jobs = append(jobs, &batchCopy)
		count++
	}
	
	return jobs, nil
}

// CleanupBatchOperations 오래된 배치 작업들을 정리합니다
func (dws *DockerWorkspaceService) CleanupBatchOperations(ctx context.Context, olderThan time.Duration) error {
	dws.batchMu.Lock()
	defer dws.batchMu.Unlock()
	
	cutoff := time.Now().Add(-olderThan)
	
	for batchID, batch := range dws.batchJobs {
		if batch.StartTime.Before(cutoff) {
			delete(dws.batchJobs, batchID)
		}
	}
	
	return nil
}

// executeBatchOperation 배치 작업을 실행합니다
func (dws *DockerWorkspaceService) executeBatchOperation(ctx context.Context, batch *BatchJob, ownerID string) {
	// 상태를 진행 중으로 변경
	dws.updateBatchStatus(batch, BatchStatusInProgress)
	
	// 동시성 제어를 위한 세마포어
	semaphore := make(chan struct{}, 5) // 최대 5개 동시 실행
	var wg sync.WaitGroup
	var mu sync.Mutex
	
	for _, workspaceID := range batch.WorkspaceIDs {
		// 취소 확인
		if batch.Status == BatchStatusCancelled {
			break
		}
		
		wg.Add(1)
		go func(wsID string) {
			defer wg.Done()
			
			// 세마포어 획득
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			
			// 개별 워크스페이스 작업 실행
			err := dws.executeBatchWorkspaceOperation(ctx, batch.Operation, wsID, ownerID)
			
			// 결과 업데이트
			mu.Lock()
			if err != nil {
				batch.Progress.Failed++
				batch.Errors = append(batch.Errors, fmt.Sprintf("Workspace %s: %v", wsID, err))
				batch.Results[wsID] = map[string]interface{}{
					"status": "failed",
					"error":  err.Error(),
				}
			} else {
				batch.Progress.Completed++
				batch.Results[wsID] = map[string]interface{}{
					"status": "completed",
				}
			}
			mu.Unlock()
		}(workspaceID)
	}
	
	// 모든 작업 완료 대기
	wg.Wait()
	
	// 최종 상태 결정
	finalStatus := BatchStatusCompleted
	if batch.Status == BatchStatusCancelled {
		finalStatus = BatchStatusCancelled
	} else if batch.Progress.Failed > 0 {
		finalStatus = BatchStatusFailed
	}
	
	dws.updateBatchStatus(batch, finalStatus)
	now := time.Now()
	batch.EndTime = &now
}

// executeBatchWorkspaceOperation 개별 워크스페이스에 대한 배치 작업을 실행합니다
func (dws *DockerWorkspaceService) executeBatchWorkspaceOperation(ctx context.Context, operation, workspaceID, ownerID string) error {
	switch operation {
	case "start":
		return dws.ActivateWorkspace(ctx, workspaceID, ownerID)
	case "stop":
		return dws.DeactivateWorkspace(ctx, workspaceID, ownerID)
	case "restart":
		// 재시작: 중지 후 시작
		if err := dws.DeactivateWorkspace(ctx, workspaceID, ownerID); err != nil {
			return fmt.Errorf("stop workspace: %w", err)
		}
		// 잠시 대기
		time.Sleep(2 * time.Second)
		return dws.ActivateWorkspace(ctx, workspaceID, ownerID)
	case "delete":
		return dws.DeleteWorkspace(ctx, workspaceID, ownerID)
	default:
		return fmt.Errorf("unknown batch operation: %s", operation)
	}
}

// updateBatchStatus 배치 작업 상태를 업데이트합니다
func (dws *DockerWorkspaceService) updateBatchStatus(batch *BatchJob, status BatchJobStatus) {
	dws.batchMu.Lock()
	defer dws.batchMu.Unlock()
	batch.Status = status
}

// GetWorkspaceStatus 워크스페이스의 Docker 컨테이너 상태를 포함한 상태를 조회합니다
func (dws *DockerWorkspaceService) GetWorkspaceStatus(ctx context.Context, workspaceID string) (*WorkspaceStatus, error) {
	// 컨테이너 목록 조회
	containers, err := dws.dockerManager.Container().ListWorkspaceContainers(ctx, workspaceID)
	if err != nil {
		return &WorkspaceStatus{
			ContainerState: "unknown",
			LastError:      err.Error(),
		}, nil
	}
	
	if len(containers) == 0 {
		return &WorkspaceStatus{
			ContainerState: "none",
		}, nil
	}
	
	// 첫 번째 컨테이너 상태를 기준으로 함
	container := containers[0]
	
	status := &WorkspaceStatus{
		ContainerID:    container.ID,
		ContainerState: string(container.State),
		Uptime:         dws.calculateUptime(container.Started),
	}
	
	// 상태 추적기가 있으면 메트릭 추가
	if dws.statusTracker != nil {
		if metrics, err := dws.statusTracker.GetWorkspaceMetrics(workspaceID); err == nil {
			status.Metrics = metrics
		}
	}
	
	return status, nil
}

// WorkspaceStatus 워크스페이스 상태 정보
type WorkspaceStatus struct {
	ContainerID    string                   `json:"container_id,omitempty"`
	ContainerState string                   `json:"container_state"`
	Uptime         string                   `json:"uptime,omitempty"`
	Metrics        *status.WorkspaceMetrics `json:"metrics,omitempty"`
	LastError      string                   `json:"last_error,omitempty"`
}

// calculateUptime 컨테이너 시작 시간으로부터 업타임을 계산합니다
func (dws *DockerWorkspaceService) calculateUptime(startTime *time.Time) string {
	if startTime == nil || startTime.IsZero() {
		return ""
	}
	
	duration := time.Since(*startTime)
	
	if duration < time.Minute {
		return fmt.Sprintf("%.0f초", duration.Seconds())
	} else if duration < time.Hour {
		return fmt.Sprintf("%.0f분", duration.Minutes())
	} else if duration < 24*time.Hour {
		return fmt.Sprintf("%.1f시간", duration.Hours())
	} else {
		days := int(duration.Hours() / 24)
		hours := int(duration.Hours()) % 24
		return fmt.Sprintf("%d일 %d시간", days, hours)
	}
}