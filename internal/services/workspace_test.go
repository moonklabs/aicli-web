package services

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/storage/memory"
)

func TestWorkspaceService_CreateWorkspace(t *testing.T) {
	// 메모리 스토리지 생성
	storage := memory.New()
	
	// 서비스 생성
	service := NewWorkspaceService(storage)
	
	ctx := context.Background()
	ownerID := "user-123"
	req := &models.CreateWorkspaceRequest{
		Name:        "test-workspace",
		ProjectPath: "/tmp/test-workspace",
	}
	
	// 실행
	workspace, err := service.CreateWorkspace(ctx, req, ownerID)
	
	// 검증
	assert.NoError(t, err)
	assert.NotNil(t, workspace)
	assert.Equal(t, req.Name, workspace.Name)
	assert.Equal(t, ownerID, workspace.OwnerID)
	assert.Equal(t, models.WorkspaceStatusActive, workspace.Status)
	assert.NotEmpty(t, workspace.ID)
}

/* 
func TestWorkspaceService_CreateWorkspace_DuplicateName(t *testing.T) {
	// Mock 스토리지 생성
	mockStorage := NewMockStorage()
	
	// 서비스 생성
	service := NewWorkspaceService(mockStorage)
	
	ctx := context.Background()
	ownerID := "user-123"
	req := &models.CreateWorkspaceRequest{
		Name: "existing-workspace",
	}
	
	// 중복 확인 Mock 설정 - 이미 존재함
	mockStorage.workspaceStorage.On("ExistsByName", ctx, ownerID, req.Name).Return(true, nil)
	
	// 실행
	workspace, err := service.CreateWorkspace(ctx, req, ownerID)
	
	// 검증
	assert.Error(t, err)
	assert.Nil(t, workspace)
	assert.Contains(t, err.Error(), "already exists")
	
	mockStorage.workspaceStorage.AssertExpectations(t)
}

func TestWorkspaceService_GetWorkspace(t *testing.T) {
	// Mock 스토리지 생성
	mockStorage := NewMockStorage()
	
	// 서비스 생성
	service := NewWorkspaceService(mockStorage)
	
	ctx := context.Background()
	workspaceID := "workspace-123"
	ownerID := "user-123"
	
	expectedWorkspace := &models.Workspace{
		ID:      workspaceID,
		Name:    "test-workspace",
		OwnerID: ownerID,
		Status:  models.WorkspaceStatusActive,
	}
	
	// GetByID Mock 설정
	mockStorage.workspaceStorage.On("GetByID", ctx, workspaceID).Return(expectedWorkspace, nil)
	
	// 실행
	workspace, err := service.GetWorkspace(ctx, workspaceID, ownerID)
	
	// 검증
	assert.NoError(t, err)
	assert.NotNil(t, workspace)
	assert.Equal(t, expectedWorkspace.ID, workspace.ID)
	assert.Equal(t, expectedWorkspace.Name, workspace.Name)
	
	mockStorage.workspaceStorage.AssertExpectations(t)
}

func TestWorkspaceService_GetWorkspace_NotOwner(t *testing.T) {
	// Mock 스토리지 생성
	mockStorage := NewMockStorage()
	
	// 서비스 생성
	service := NewWorkspaceService(mockStorage)
	
	ctx := context.Background()
	workspaceID := "workspace-123"
	ownerID := "user-456" // 다른 사용자
	
	expectedWorkspace := &models.Workspace{
		ID:      workspaceID,
		Name:    "test-workspace",
		OwnerID: "user-123", // 실제 소유자
		Status:  models.WorkspaceStatusActive,
	}
	
	// GetByID Mock 설정
	mockStorage.workspaceStorage.On("GetByID", ctx, workspaceID).Return(expectedWorkspace, nil)
	
	// 실행
	workspace, err := service.GetWorkspace(ctx, workspaceID, ownerID)
	
	// 검증
	assert.Error(t, err)
	assert.Nil(t, workspace)
	assert.Contains(t, err.Error(), "not found")
	
	mockStorage.workspaceStorage.AssertExpectations(t)
}

func TestWorkspaceService_UpdateWorkspace(t *testing.T) {
	// Mock 스토리지 생성
	mockStorage := NewMockStorage()
	
	// 서비스 생성
	service := NewWorkspaceService(mockStorage)
	
	ctx := context.Background()
	workspaceID := "workspace-123"
	ownerID := "user-123"
	
	existingWorkspace := &models.Workspace{
		ID:      workspaceID,
		Name:    "old-name",
		OwnerID: ownerID,
		Status:  models.WorkspaceStatusActive,
	}
	
	req := &models.UpdateWorkspaceRequest{
		Name: "new-name",
	}
	
	// GetByID Mock 설정
	mockStorage.workspaceStorage.On("GetByID", ctx, workspaceID).Return(existingWorkspace, nil)
	
	// 중복 확인 Mock 설정
	mockStorage.workspaceStorage.On("ExistsByName", ctx, ownerID, "new-name").Return(false, nil)
	
	// Update Mock 설정
	mockStorage.workspaceStorage.On("Update", ctx, workspaceID, map[string]interface{}{
		"name":       "new-name",
		"updated_at": mock.AnythingOfType("time.Time"),
	}).Return(nil)
	
	// 업데이트된 워크스페이스 조회 Mock 설정
	updatedWorkspace := &models.Workspace{
		ID:      workspaceID,
		Name:    "new-name",
		OwnerID: ownerID,
		Status:  models.WorkspaceStatusActive,
	}
	mockStorage.workspaceStorage.On("GetByID", ctx, workspaceID).Return(updatedWorkspace, nil).Once()
	
	// 실행
	result, err := service.UpdateWorkspace(ctx, workspaceID, req, ownerID)
	
	// 검증
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "new-name", result.Name)
	
	mockStorage.workspaceStorage.AssertExpectations(t)
}

func TestWorkspaceService_DeleteWorkspace(t *testing.T) {
	// Mock 스토리지 생성
	mockStorage := NewMockStorage()
	
	// 서비스 생성
	service := NewWorkspaceService(mockStorage)
	
	ctx := context.Background()
	workspaceID := "workspace-123"
	ownerID := "user-123"
	
	existingWorkspace := &models.Workspace{
		ID:      workspaceID,
		Name:    "test-workspace",
		OwnerID: ownerID,
		Status:  models.WorkspaceStatusActive,
	}
	
	// GetByID Mock 설정
	mockStorage.workspaceStorage.On("GetByID", ctx, workspaceID).Return(existingWorkspace, nil)
	
	// Delete Mock 설정
	mockStorage.workspaceStorage.On("Delete", ctx, workspaceID).Return(nil)
	
	// 실행
	err := service.DeleteWorkspace(ctx, workspaceID, ownerID)
	
	// 검증
	assert.NoError(t, err)
	
	mockStorage.workspaceStorage.AssertExpectations(t)
}

func TestWorkspaceService_ListWorkspaces(t *testing.T) {
	// Mock 스토리지 생성
	mockStorage := NewMockStorage()
	
	// 서비스 생성
	service := NewWorkspaceService(mockStorage)
	
	ctx := context.Background()
	ownerID := "user-123"
	pagination := &models.PaginationRequest{
		Page:  1,
		Limit: 10,
	}
	
	workspaces := []*models.Workspace{
		{
			ID:      "workspace-1",
			Name:    "workspace-1",
			OwnerID: ownerID,
			Status:  models.WorkspaceStatusActive,
		},
		{
			ID:      "workspace-2",
			Name:    "workspace-2",
			OwnerID: ownerID,
			Status:  models.WorkspaceStatusActive,
		},
	}
	
	// GetByOwnerID Mock 설정
	mockStorage.workspaceStorage.On("GetByOwnerID", ctx, ownerID, pagination).Return(workspaces, 2, nil)
	
	// 실행
	result, err := service.ListWorkspaces(ctx, ownerID, pagination)
	
	// 검증
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Data, 2)
	assert.Equal(t, 2, result.Meta.Total)
	assert.Equal(t, 1, result.Meta.TotalPages)
	
	mockStorage.workspaceStorage.AssertExpectations(t)
}

// 헬퍼 함수
func strPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}

func timePtr(t time.Time) *time.Time {
	return &t
}
*/