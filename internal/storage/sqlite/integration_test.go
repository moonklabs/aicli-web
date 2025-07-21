package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/storage"
)

// TestSQLiteStorageIntegration SQLite 스토리지 통합 테스트
func TestSQLiteStorageIntegration(t *testing.T) {
	// 임시 데이터베이스 파일 생성
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_integration.db")
	
	config := DefaultConfig()
	config.DataSource = dbPath
	
	// 스토리지 생성
	storage, err := New(config)
	require.NoError(t, err)
	defer storage.Close()
	
	// 테스트 수행
	t.Run("WorkspaceOperations", func(t *testing.T) {
		testWorkspaceOperations(t, storage)
	})
	
	t.Run("ProjectOperations", func(t *testing.T) {
		testProjectOperations(t, storage)
	})
	
	t.Run("SessionOperations", func(t *testing.T) {
		testSessionOperations(t, storage)
	})
	
	t.Run("TaskOperations", func(t *testing.T) {
		testTaskOperations(t, storage)
	})
	
	t.Run("TransactionOperations", func(t *testing.T) {
		testTransactionOperations(t, storage)
	})
	
	t.Run("ConcurrentOperations", func(t *testing.T) {
		testConcurrentOperations(t, storage)
	})
}

func testWorkspaceOperations(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ws := s.Workspace()
	
	// 테스트 워크스페이스 생성
	workspace := &models.Workspace{
		ID:          "ws-test-001",
		Name:        "Integration Test Workspace",
		ProjectPath: "/tmp/test-workspace",
		Status:      models.WorkspaceStatusActive,
		OwnerID:     "user-001",
		ActiveTasks: 0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Version:     1,
	}
	
	// Create 테스트
	err := ws.Create(ctx, workspace)
	require.NoError(t, err)
	
	// GetByID 테스트
	retrieved, err := ws.GetByID(ctx, workspace.ID)
	require.NoError(t, err)
	assert.Equal(t, workspace.ID, retrieved.ID)
	assert.Equal(t, workspace.Name, retrieved.Name)
	assert.Equal(t, workspace.OwnerID, retrieved.OwnerID)
	
	// Update 테스트
	updates := map[string]interface{}{
		"name":         "Updated Workspace Name",
		"active_tasks": 5,
	}
	err = ws.Update(ctx, workspace.ID, updates)
	require.NoError(t, err)
	
	// Update 확인
	updated, err := ws.GetByID(ctx, workspace.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Workspace Name", updated.Name)
	assert.Equal(t, 5, updated.ActiveTasks)
	assert.Equal(t, int64(2), updated.Version) // 버전 증가 확인
	
	// GetByOwnerID 테스트
	pagination := &models.PaginationRequest{
		Page:  1,
		Limit: 10,
	}
	workspaces, total, err := ws.GetByOwnerID(ctx, workspace.OwnerID, pagination)
	require.NoError(t, err)
	assert.Greater(t, total, int64(0))
	assert.Len(t, workspaces, 1)
	
	// ExistsByName 테스트
	exists, err := ws.ExistsByName(ctx, workspace.OwnerID, workspace.Name)
	require.NoError(t, err)
	assert.True(t, exists)
	
	// List 테스트
	allWorkspaces, totalAll, err := ws.List(ctx, pagination)
	require.NoError(t, err)
	assert.Greater(t, totalAll, int64(0))
	assert.Len(t, allWorkspaces, 1)
	
	// Delete (Soft) 테스트
	err = ws.Delete(ctx, workspace.ID)
	require.NoError(t, err)
	
	// 삭제 확인 (GetByID는 soft delete된 항목을 반환하지 않아야 함)
	_, err = ws.GetByID(ctx, workspace.ID)
	assert.Error(t, err)
	assert.ErrorIs(t, err, storage.ErrNotFound)
}

func testProjectOperations(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ws := s.Workspace()
	ps := s.Project()
	
	// 워크스페이스 먼저 생성
	workspace := &models.Workspace{
		ID:          "ws-proj-001",
		Name:        "Project Test Workspace",
		ProjectPath: "/tmp/project-test",
		Status:      models.WorkspaceStatusActive,
		OwnerID:     "user-002",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Version:     1,
	}
	require.NoError(t, ws.Create(ctx, workspace))
	
	// 테스트 프로젝트 생성
	project := &models.Project{
		ID:          "proj-001",
		WorkspaceID: workspace.ID,
		Name:        "Test Project",
		Path:        "/tmp/project-test/proj-001",
		Language:    "go",
		Status:      models.ProjectStatusActive,
		Config: models.ProjectConfig{
			AutoSave:     true,
			BuildCommand: "go build",
			TestCommand:  "go test",
		},
		GitInfo: models.GitInfo{
			Branch:     "main",
			CommitHash: "abc123",
			Remote:     "origin",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Version:   1,
	}
	
	// Create 테스트
	err := ps.Create(ctx, project)
	require.NoError(t, err)
	
	// GetByID 테스트
	retrieved, err := ps.GetByID(ctx, project.ID)
	require.NoError(t, err)
	assert.Equal(t, project.ID, retrieved.ID)
	assert.Equal(t, project.WorkspaceID, retrieved.WorkspaceID)
	assert.Equal(t, project.Language, retrieved.Language)
	assert.Equal(t, project.Config.BuildCommand, retrieved.Config.BuildCommand)
	assert.Equal(t, project.GitInfo.Branch, retrieved.GitInfo.Branch)
	
	// GetByWorkspaceID 테스트
	pagination := &models.PaginationRequest{
		Page:  1,
		Limit: 10,
	}
	projects, total, err := ps.GetByWorkspaceID(ctx, workspace.ID, pagination)
	require.NoError(t, err)
	assert.Greater(t, total, int64(0))
	assert.Len(t, projects, 1)
	
	// GetByPath 테스트
	byPath, err := ps.GetByPath(ctx, project.Path)
	require.NoError(t, err)
	assert.Equal(t, project.ID, byPath.ID)
	
	// Update 테스트
	updates := map[string]interface{}{
		"language": "python",
		"config": models.ProjectConfig{
			AutoSave:     false,
			BuildCommand: "python setup.py build",
			TestCommand:  "python -m pytest",
		},
	}
	err = ps.Update(ctx, project.ID, updates)
	require.NoError(t, err)
	
	// Update 확인
	updated, err := ps.GetByID(ctx, project.ID)
	require.NoError(t, err)
	assert.Equal(t, "python", updated.Language)
	assert.Equal(t, "python setup.py build", updated.Config.BuildCommand)
	
	// Delete 테스트
	err = ps.Delete(ctx, project.ID)
	require.NoError(t, err)
	
	// 삭제 확인
	_, err = ps.GetByID(ctx, project.ID)
	assert.Error(t, err)
	assert.ErrorIs(t, err, storage.ErrNotFound)
}

func testSessionOperations(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ws := s.Workspace()
	ps := s.Project()
	ss := s.Session()
	
	// 워크스페이스와 프로젝트 생성
	workspace := &models.Workspace{
		ID:          "ws-sess-001",
		Name:        "Session Test Workspace",
		ProjectPath: "/tmp/session-test",
		Status:      models.WorkspaceStatusActive,
		OwnerID:     "user-003",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Version:     1,
	}
	require.NoError(t, ws.Create(ctx, workspace))
	
	project := &models.Project{
		ID:          "proj-sess-001",
		WorkspaceID: workspace.ID,
		Name:        "Session Test Project",
		Path:        "/tmp/session-test/project",
		Language:    "go",
		Status:      models.ProjectStatusActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Version:     1,
	}
	require.NoError(t, ps.Create(ctx, project))
	
	// 테스트 세션 생성
	startedAt := time.Now()
	session := &models.Session{
		ID:           "sess-001",
		ProjectID:    project.ID,
		ProcessID:    12345,
		Status:       models.SessionStatusActive,
		StartedAt:    &startedAt,
		LastActive:   time.Now(),
		CommandCount: 0,
		Metadata: map[string]interface{}{
			"user_agent": "test-client",
			"version":    "1.0.0",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Version:   1,
	}
	
	// Create 테스트
	err := ss.Create(ctx, session)
	require.NoError(t, err)
	
	// GetByID 테스트
	retrieved, err := ss.GetByID(ctx, session.ID)
	require.NoError(t, err)
	assert.Equal(t, session.ID, retrieved.ID)
	assert.Equal(t, session.ProjectID, retrieved.ProjectID)
	assert.Equal(t, session.ProcessID, retrieved.ProcessID)
	assert.Equal(t, session.Status, retrieved.Status)
	
	// GetByProjectID 테스트
	pagination := &models.PaginationRequest{
		Page:  1,
		Limit: 10,
	}
	sessions, total, err := ss.GetByProjectID(ctx, project.ID, pagination)
	require.NoError(t, err)
	assert.Greater(t, total, int64(0))
	assert.Len(t, sessions, 1)
	
	// GetActiveCount 테스트
	count, err := ss.GetActiveCount(ctx, project.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
	
	// Update 테스트
	updates := map[string]interface{}{
		"status":        models.SessionStatusIdle,
		"command_count": int64(10),
	}
	err = ss.Update(ctx, session.ID, updates)
	require.NoError(t, err)
	
	// Update 확인
	updated, err := ss.GetByID(ctx, session.ID)
	require.NoError(t, err)
	assert.Equal(t, models.SessionStatusIdle, updated.Status)
	assert.Equal(t, int64(10), updated.CommandCount)
	
	// Delete 테스트
	err = ss.Delete(ctx, session.ID)
	require.NoError(t, err)
	
	// 삭제 확인
	_, err = ss.GetByID(ctx, session.ID)
	assert.Error(t, err)
	assert.ErrorIs(t, err, storage.ErrNotFound)
}

func testTaskOperations(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ws := s.Workspace()
	ps := s.Project()
	ss := s.Session()
	ts := s.Task()
	
	// 필요한 상위 엔티티들 생성
	workspace := &models.Workspace{
		ID:          "ws-task-001",
		Name:        "Task Test Workspace",
		ProjectPath: "/tmp/task-test",
		Status:      models.WorkspaceStatusActive,
		OwnerID:     "user-004",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Version:     1,
	}
	require.NoError(t, ws.Create(ctx, workspace))
	
	project := &models.Project{
		ID:          "proj-task-001",
		WorkspaceID: workspace.ID,
		Name:        "Task Test Project",
		Path:        "/tmp/task-test/project",
		Language:    "go",
		Status:      models.ProjectStatusActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Version:     1,
	}
	require.NoError(t, ps.Create(ctx, project))
	
	startedAt := time.Now()
	session := &models.Session{
		ID:           "sess-task-001",
		ProjectID:    project.ID,
		ProcessID:    12346,
		Status:       models.SessionStatusActive,
		StartedAt:    &startedAt,
		LastActive:   time.Now(),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Version:      1,
	}
	require.NoError(t, ss.Create(ctx, session))
	
	// 테스트 태스크 생성
	taskStartedAt := time.Now()
	task := &models.Task{
		ID:        "task-001",
		SessionID: session.ID,
		Command:   "go build .",
		Status:    models.TaskStatusRunning,
		StartedAt: &taskStartedAt,
		Duration:  0,
		Output:    "Building project...",
		Error:     "",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Version:   1,
	}
	
	// Create 테스트
	err := ts.Create(ctx, task)
	require.NoError(t, err)
	
	// GetByID 테스트
	retrieved, err := ts.GetByID(ctx, task.ID)
	require.NoError(t, err)
	assert.Equal(t, task.ID, retrieved.ID)
	assert.Equal(t, task.SessionID, retrieved.SessionID)
	assert.Equal(t, task.Command, retrieved.Command)
	assert.Equal(t, task.Status, retrieved.Status)
	
	// GetBySessionID 테스트
	pagination := &models.PaginationRequest{
		Page:  1,
		Limit: 10,
	}
	tasks, total, err := ts.GetBySessionID(ctx, session.ID, pagination)
	require.NoError(t, err)
	assert.Greater(t, total, int64(0))
	assert.Len(t, tasks, 1)
	
	// GetByStatus 테스트
	runningTasks, runningTotal, err := ts.GetByStatus(ctx, models.TaskStatusRunning, pagination)
	require.NoError(t, err)
	assert.Greater(t, runningTotal, int64(0))
	assert.Len(t, runningTasks, 1)
	
	// GetActiveCount 테스트
	count, err := ts.GetActiveCount(ctx, session.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
	
	// Update 테스트
	completedAt := time.Now()
	updates := map[string]interface{}{
		"status":       models.TaskStatusCompleted,
		"completed_at": &completedAt,
		"duration":     int64(5000), // 5초
		"output":       "Build successful",
	}
	err = ts.Update(ctx, task.ID, updates)
	require.NoError(t, err)
	
	// Update 확인
	updated, err := ts.GetByID(ctx, task.ID)
	require.NoError(t, err)
	assert.Equal(t, models.TaskStatusCompleted, updated.Status)
	assert.Equal(t, int64(5000), updated.Duration)
	assert.Equal(t, "Build successful", updated.Output)
	assert.NotNil(t, updated.CompletedAt)
	
	// Delete 테스트
	err = ts.Delete(ctx, task.ID)
	require.NoError(t, err)
	
	// 삭제 확인
	_, err = ts.GetByID(ctx, task.ID)
	assert.Error(t, err)
	assert.ErrorIs(t, err, storage.ErrNotFound)
}

func testTransactionOperations(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	transactionalStorage, ok := s.(storage.TransactionalStorage)
	require.True(t, ok, "Storage must implement TransactionalStorage")
	
	// 트랜잭션 테스트
	workspace := &models.Workspace{
		ID:          "ws-tx-001",
		Name:        "Transaction Test Workspace",
		ProjectPath: "/tmp/tx-test",
		Status:      models.WorkspaceStatusActive,
		OwnerID:     "user-tx-001",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Version:     1,
	}
	
	// 성공적인 트랜잭션 테스트
	tx, err := transactionalStorage.BeginTx(ctx)
	require.NoError(t, err)
	
	err = tx.Workspace().Create(ctx, workspace)
	require.NoError(t, err)
	
	err = tx.Commit()
	require.NoError(t, err)
	
	// 커밋 확인
	retrieved, err := s.Workspace().GetByID(ctx, workspace.ID)
	require.NoError(t, err)
	assert.Equal(t, workspace.ID, retrieved.ID)
	
	// 롤백 트랜잭션 테스트
	workspace2 := &models.Workspace{
		ID:          "ws-tx-002",
		Name:        "Rollback Test Workspace",
		ProjectPath: "/tmp/rollback-test",
		Status:      models.WorkspaceStatusActive,
		OwnerID:     "user-tx-002",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Version:     1,
	}
	
	tx2, err := transactionalStorage.BeginTx(ctx)
	require.NoError(t, err)
	
	err = tx2.Workspace().Create(ctx, workspace2)
	require.NoError(t, err)
	
	// 롤백
	err = tx2.Rollback()
	require.NoError(t, err)
	
	// 롤백 확인 (데이터가 존재하지 않아야 함)
	_, err = s.Workspace().GetByID(ctx, workspace2.ID)
	assert.Error(t, err)
	assert.ErrorIs(t, err, storage.ErrNotFound)
}

func testConcurrentOperations(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ws := s.Workspace()
	
	const numGoroutines = 10
	const numOperationsPerGoroutine = 20
	
	// 동시성 테스트를 위한 채널
	done := make(chan bool, numGoroutines)
	errors := make(chan error, numGoroutines*numOperationsPerGoroutine)
	
	// 여러 고루틴에서 동시에 워크스페이스 생성/조회/업데이트
	for i := 0; i < numGoroutines; i++ {
		go func(workerID int) {
			defer func() { done <- true }()
			
			for j := 0; j < numOperationsPerGoroutine; j++ {
				workspace := &models.Workspace{
					ID:          fmt.Sprintf("ws-concurrent-%d-%d", workerID, j),
					Name:        fmt.Sprintf("Concurrent Workspace %d-%d", workerID, j),
					ProjectPath: fmt.Sprintf("/tmp/concurrent-%d-%d", workerID, j),
					Status:      models.WorkspaceStatusActive,
					OwnerID:     fmt.Sprintf("user-concurrent-%d", workerID),
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
					Version:     1,
				}
				
				// Create
				if err := ws.Create(ctx, workspace); err != nil {
					errors <- fmt.Errorf("worker %d, op %d, create: %w", workerID, j, err)
					continue
				}
				
				// Read
				if _, err := ws.GetByID(ctx, workspace.ID); err != nil {
					errors <- fmt.Errorf("worker %d, op %d, read: %w", workerID, j, err)
					continue
				}
				
				// Update
				updates := map[string]interface{}{
					"active_tasks": j + 1,
				}
				if err := ws.Update(ctx, workspace.ID, updates); err != nil {
					errors <- fmt.Errorf("worker %d, op %d, update: %w", workerID, j, err)
					continue
				}
			}
		}(i)
	}
	
	// 모든 고루틴 완료 대기
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
	
	// 에러 채널 닫고 확인
	close(errors)
	var allErrors []error
	for err := range errors {
		allErrors = append(allErrors, err)
	}
	
	// 동시성 에러가 없어야 함
	if len(allErrors) > 0 {
		for _, err := range allErrors[:10] { // 처음 10개 에러만 출력
			t.Logf("Concurrent error: %v", err)
		}
		t.Errorf("Got %d concurrent errors, expected 0", len(allErrors))
	}
	
	// 데이터 일관성 확인
	pagination := &models.PaginationRequest{
		Page:  1,
		Limit: 1000,
	}
	allWorkspaces, total, err := ws.List(ctx, pagination)
	require.NoError(t, err)
	
	expectedTotal := int64(numGoroutines * numOperationsPerGoroutine)
	assert.GreaterOrEqual(t, total, expectedTotal, "Expected at least %d workspaces, got %d", expectedTotal, total)
	assert.GreaterOrEqual(t, len(allWorkspaces), int(expectedTotal), "Expected at least %d workspaces in result, got %d", expectedTotal, len(allWorkspaces))
}

// TestSQLiteStorageConnectionManagement 연결 관리 테스트
func TestSQLiteStorageConnectionManagement(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_connection.db")
	
	config := DefaultConfig()
	config.DataSource = dbPath
	
	// 스토리지 생성
	storage, err := New(config)
	require.NoError(t, err)
	
	// HealthCheck 테스트
	err = storage.HealthCheck(context.Background())
	assert.NoError(t, err)
	
	// 통계 테스트
	stats := storage.Stats()
	assert.NotNil(t, stats)
	
	// 정상 종료 테스트
	err = storage.Close()
	assert.NoError(t, err)
	
	// 종료 후 HealthCheck는 실패해야 함
	err = storage.HealthCheck(context.Background())
	assert.Error(t, err)
}

// TestSQLiteStorageWithExistingDB 기존 DB 파일 사용 테스트
func TestSQLiteStorageWithExistingDB(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "existing.db")
	
	// 기존 DB 파일 생성
	db, err := sql.Open("sqlite3", dbPath)
	require.NoError(t, err)
	
	// 임시 테이블 생성
	_, err = db.Exec("CREATE TABLE temp_test (id TEXT PRIMARY KEY, name TEXT)")
	require.NoError(t, err)
	
	_, err = db.Exec("INSERT INTO temp_test (id, name) VALUES ('test-1', 'Test Name')")
	require.NoError(t, err)
	
	db.Close()
	
	// 기존 파일이 존재하는지 확인
	_, err = os.Stat(dbPath)
	require.NoError(t, err)
	
	// Storage로 열기
	config := DefaultConfig()
	config.DataSource = dbPath
	
	storage, err := New(config)
	require.NoError(t, err)
	defer storage.Close()
	
	// 기존 데이터 확인
	sqliteStorage := storage.(*Storage)
	row := sqliteStorage.db.QueryRow("SELECT name FROM temp_test WHERE id = ?", "test-1")
	
	var name string
	err = row.Scan(&name)
	require.NoError(t, err)
	assert.Equal(t, "Test Name", name)
}