package services

import (
	"context"
	"testing"
	"time"

	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSessionService_Create(t *testing.T) {
	storage := storage.NewMemoryAdapter()
	projectService := NewProjectService(storage)
	sessionService := NewSessionService(storage, projectService, nil)

	// 먼저 워크스페이스와 프로젝트 생성
	workspace := &models.Workspace{
		Name:     "Test Workspace",
		OwnerID:  "user-123",
		ProjectPath: "/test/workspace",
	}
	err := storage.Workspace().Create(context.Background(), workspace)
	require.NoError(t, err)

	project := &models.Project{
		WorkspaceID: workspace.ID,
		Name:        "Test Project",
		Path:        "/test/path",
		Status:      models.ProjectStatusActive,
	}
	err = storage.Project().Create(context.Background(), project)
	require.NoError(t, err)

	// 세션 생성
	req := &models.SessionCreateRequest{
		ProjectID: project.ID,
		Metadata: map[string]string{
			"test": "value",
		},
	}

	session, err := sessionService.Create(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, session)
	assert.Equal(t, project.ID, session.ProjectID)
	assert.Equal(t, models.SessionPending, session.Status)
	assert.Equal(t, "value", session.Metadata["test"])
}

func TestSessionService_UpdateStatus(t *testing.T) {
	storage := storage.NewMemoryAdapter()
	projectService := NewProjectService(storage)
	sessionService := NewSessionService(storage, projectService, nil)

	// 테스트 데이터 준비
	workspace := &models.Workspace{
		Name:     "Test Workspace",
		OwnerID:  "user-123",
		ProjectPath: "/test/workspace",
	}
	err := storage.Workspace().Create(context.Background(), workspace)
	require.NoError(t, err)

	project := &models.Project{
		WorkspaceID: workspace.ID,
		Name:        "Test Project",
		Path:        "/test/path",
		Status:      models.ProjectStatusActive,
	}
	err = storage.Project().Create(context.Background(), project)
	require.NoError(t, err)

	session, err := sessionService.Create(context.Background(), &models.SessionCreateRequest{
		ProjectID: project.ID,
	})
	require.NoError(t, err)

	// 상태 업데이트 테스트
	testCases := []struct {
		name        string
		fromStatus  models.SessionStatus
		toStatus    models.SessionStatus
		shouldError bool
	}{
		{
			name:        "Pending to Active",
			fromStatus:  models.SessionPending,
			toStatus:    models.SessionActive,
			shouldError: false,
		},
		{
			name:        "Active to Idle",
			fromStatus:  models.SessionActive,
			toStatus:    models.SessionIdle,
			shouldError: false,
		},
		{
			name:        "Idle to Active",
			fromStatus:  models.SessionIdle,
			toStatus:    models.SessionActive,
			shouldError: false,
		},
		{
			name:        "Active to Ending",
			fromStatus:  models.SessionActive,
			toStatus:    models.SessionEnding,
			shouldError: false,
		},
		{
			name:        "Invalid transition: Ended to Active",
			fromStatus:  models.SessionEnded,
			toStatus:    models.SessionActive,
			shouldError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 초기 상태 설정
			session.Status = tc.fromStatus
			err := storage.Session().Update(context.Background(), session)
			require.NoError(t, err)

			// 상태 업데이트 시도
			err = sessionService.UpdateStatus(context.Background(), session.ID, tc.toStatus)

			if tc.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				
				// 업데이트된 상태 확인
				updatedSession, err := sessionService.GetByID(context.Background(), session.ID)
				require.NoError(t, err)
				assert.Equal(t, tc.toStatus, updatedSession.Status)
			}
		})
	}
}

func TestSessionService_ConcurrentLimit(t *testing.T) {
	config := &SessionServiceConfig{
		MaxConcurrent:   2,
		CleanupInterval: 1 * time.Minute,
	}

	storage := storage.NewMemoryAdapter()
	projectService := NewProjectService(storage)
	sessionService := NewSessionService(storage, projectService, config)
	defer sessionService.Stop()

	// 테스트 데이터 준비
	workspace := &models.Workspace{
		Name:     "Test Workspace",
		OwnerID:  "user-123",
		ProjectPath: "/test/workspace",
	}
	err := storage.Workspace().Create(context.Background(), workspace)
	require.NoError(t, err)

	project := &models.Project{
		WorkspaceID: workspace.ID,
		Name:        "Test Project",
		Path:        "/test/path",
		Status:      models.ProjectStatusActive,
	}
	err = storage.Project().Create(context.Background(), project)
	require.NoError(t, err)

	// 첫 번째 세션 생성
	session1, err := sessionService.Create(context.Background(), &models.SessionCreateRequest{
		ProjectID: project.ID,
	})
	assert.NoError(t, err)
	assert.NotNil(t, session1)

	// 두 번째 세션 생성
	session2, err := sessionService.Create(context.Background(), &models.SessionCreateRequest{
		ProjectID: project.ID,
	})
	assert.NoError(t, err)
	assert.NotNil(t, session2)

	// 세 번째 세션 생성 시도 (제한 초과)
	_, err = sessionService.Create(context.Background(), &models.SessionCreateRequest{
		ProjectID: project.ID,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "최대 동시 세션 수")

	// 첫 번째 세션 종료
	err = sessionService.Terminate(context.Background(), session1.ID)
	assert.NoError(t, err)

	// 이제 세션 생성 가능
	session3, err := sessionService.Create(context.Background(), &models.SessionCreateRequest{
		ProjectID: project.ID,
	})
	assert.NoError(t, err)
	assert.NotNil(t, session3)
}

func TestSessionService_UpdateActivity(t *testing.T) {
	storage := storage.NewMemoryAdapter()
	projectService := NewProjectService(storage)
	sessionService := NewSessionService(storage, projectService, nil)

	// 테스트 데이터 준비
	workspace := &models.Workspace{
		Name:     "Test Workspace",
		OwnerID:  "user-123",
		ProjectPath: "/test/workspace",
	}
	err := storage.Workspace().Create(context.Background(), workspace)
	require.NoError(t, err)

	project := &models.Project{
		WorkspaceID: workspace.ID,
		Name:        "Test Project",
		Path:        "/test/path",
		Status:      models.ProjectStatusActive,
	}
	err = storage.Project().Create(context.Background(), project)
	require.NoError(t, err)

	session, err := sessionService.Create(context.Background(), &models.SessionCreateRequest{
		ProjectID: project.ID,
	})
	require.NoError(t, err)

	// 세션을 Idle 상태로 변경
	err = sessionService.UpdateStatus(context.Background(), session.ID, models.SessionActive)
	require.NoError(t, err)
	err = sessionService.UpdateStatus(context.Background(), session.ID, models.SessionIdle)
	require.NoError(t, err)

	// 기존 LastActive 시간 저장
	idleSession, err := sessionService.GetByID(context.Background(), session.ID)
	require.NoError(t, err)
	oldLastActive := idleSession.LastActive

	// 약간 대기
	time.Sleep(10 * time.Millisecond)

	// 활동 업데이트
	err = sessionService.UpdateActivity(context.Background(), session.ID)
	assert.NoError(t, err)

	// 업데이트 확인
	updatedSession, err := sessionService.GetByID(context.Background(), session.ID)
	require.NoError(t, err)
	
	assert.Equal(t, models.SessionActive, updatedSession.Status) // Idle에서 Active로 변경됨
	assert.True(t, updatedSession.LastActive.After(oldLastActive)) // LastActive 시간 업데이트됨
}

func TestSessionService_List(t *testing.T) {
	storage := storage.NewMemoryAdapter()
	projectService := NewProjectService(storage)
	sessionService := NewSessionService(storage, projectService, nil)

	// 테스트 데이터 준비
	workspace := &models.Workspace{
		Name:     "Test Workspace",
		OwnerID:  "user-123",
		ProjectPath: "/test/workspace",
	}
	err := storage.Workspace().Create(context.Background(), workspace)
	require.NoError(t, err)

	project1 := &models.Project{
		WorkspaceID: workspace.ID,
		Name:        "Project 1",
		Path:        "/test/path1",
		Status:      models.ProjectStatusActive,
	}
	err = storage.Project().Create(context.Background(), project1)
	require.NoError(t, err)

	project2 := &models.Project{
		WorkspaceID: workspace.ID,
		Name:        "Project 2",
		Path:        "/test/path2",
		Status:      models.ProjectStatusActive,
	}
	err = storage.Project().Create(context.Background(), project2)
	require.NoError(t, err)

	// 여러 세션 생성
	sessions := make([]*models.Session, 0)
	for i := 0; i < 5; i++ {
		projectID := project1.ID
		if i%2 == 0 {
			projectID = project2.ID
		}
		
		session, err := sessionService.Create(context.Background(), &models.SessionCreateRequest{
			ProjectID: projectID,
		})
		require.NoError(t, err)
		
		// 일부 세션을 활성화
		if i < 3 {
			err = sessionService.UpdateStatus(context.Background(), session.ID, models.SessionActive)
			require.NoError(t, err)
		}
		
		sessions = append(sessions, session)
	}

	// 전체 목록 조회
	result, err := sessionService.List(context.Background(), nil, &models.PagingRequest{
		Page:  1,
		Limit: 10,
	})
	assert.NoError(t, err)
	assert.Equal(t, 5, result.Meta.Total)
	assert.Len(t, result.Data.([]*models.Session), 5)

	// 프로젝트별 필터링
	result, err = sessionService.List(context.Background(), &models.SessionFilter{
		ProjectID: project1.ID,
	}, &models.PagingRequest{
		Page:  1,
		Limit: 10,
	})
	assert.NoError(t, err)
	assert.Equal(t, 3, result.Meta.Total)

	// 활성 세션만 필터링
	active := true
	result, err = sessionService.List(context.Background(), &models.SessionFilter{
		Active: &active,
	}, &models.PagingRequest{
		Page:  1,
		Limit: 10,
	})
	assert.NoError(t, err)
	assert.Equal(t, 3, result.Meta.Total)
}

func TestSessionService_UpdateStats(t *testing.T) {
	storage := storage.NewMemoryAdapter()
	projectService := NewProjectService(storage)
	sessionService := NewSessionService(storage, projectService, nil)

	// 테스트 데이터 준비
	workspace := &models.Workspace{
		Name:     "Test Workspace",
		OwnerID:  "user-123",
		ProjectPath: "/test/workspace",
	}
	err := storage.Workspace().Create(context.Background(), workspace)
	require.NoError(t, err)

	project := &models.Project{
		WorkspaceID: workspace.ID,
		Name:        "Test Project",
		Path:        "/test/path",
		Status:      models.ProjectStatusActive,
	}
	err = storage.Project().Create(context.Background(), project)
	require.NoError(t, err)

	session, err := sessionService.Create(context.Background(), &models.SessionCreateRequest{
		ProjectID: project.ID,
	})
	require.NoError(t, err)

	// 통계 업데이트
	err = sessionService.UpdateStats(context.Background(), session.ID, 5, 1024, 2048, 1)
	assert.NoError(t, err)

	// 업데이트 확인
	updatedSession, err := sessionService.GetByID(context.Background(), session.ID)
	require.NoError(t, err)
	
	assert.Equal(t, int64(5), updatedSession.CommandCount)
	assert.Equal(t, int64(1024), updatedSession.BytesIn)
	assert.Equal(t, int64(2048), updatedSession.BytesOut)
	assert.Equal(t, int64(1), updatedSession.ErrorCount)

	// 추가 업데이트
	err = sessionService.UpdateStats(context.Background(), session.ID, 3, 512, 1024, 0)
	assert.NoError(t, err)

	// 누적 확인
	updatedSession, err = sessionService.GetByID(context.Background(), session.ID)
	require.NoError(t, err)
	
	assert.Equal(t, int64(8), updatedSession.CommandCount)
	assert.Equal(t, int64(1536), updatedSession.BytesIn)
	assert.Equal(t, int64(3072), updatedSession.BytesOut)
	assert.Equal(t, int64(1), updatedSession.ErrorCount)
}