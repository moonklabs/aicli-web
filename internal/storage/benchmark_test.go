package storage

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/storage/boltdb"
	"github.com/aicli/aicli-web/internal/storage/cache"
	"github.com/aicli/aicli-web/internal/storage/memory"
	"github.com/aicli/aicli-web/internal/storage/monitoring"
	"github.com/aicli/aicli-web/internal/storage/sqlite"
)

// BenchmarkConfig 벤치마크 설정
type BenchmarkConfig struct {
	WorkspaceCount int
	ProjectCount   int
	SessionCount   int
	TaskCount      int
	Iterations     int
	Concurrency    int
	Logger         *zap.Logger
}

// DefaultBenchmarkConfig 기본 벤치마크 설정
func DefaultBenchmarkConfig() BenchmarkConfig {
	return BenchmarkConfig{
		WorkspaceCount: 100,
		ProjectCount:   500,
		SessionCount:   1000,
		TaskCount:      5000,
		Iterations:     1000,
		Concurrency:    10,
		Logger:         zap.NewNop(),
	}
}

// 기본 데이터 생성 함수들

func generateTestWorkspaces(count int) []*models.Workspace {
	workspaces := make([]*models.Workspace, count)
	
	for i := 0; i < count; i++ {
		workspaces[i] = &models.Workspace{
			ID:          fmt.Sprintf("ws-%06d", i),
			Name:        fmt.Sprintf("workspace-%d", i),
			ProjectPath: fmt.Sprintf("/tmp/workspace-%d", i),
			Status:      models.WorkspaceStatusActive,
			OwnerID:     fmt.Sprintf("user-%d", i%10), // 10명의 사용자
			ActiveTasks: rand.Intn(5),
			CreatedAt:   time.Now().Add(-time.Duration(i) * time.Hour),
			UpdatedAt:   time.Now(),
			Version:     1,
		}
	}
	
	return workspaces
}

func generateTestProjects(count int, workspaces []*models.Workspace) []*models.Project {
	projects := make([]*models.Project, count)
	
	for i := 0; i < count; i++ {
		workspace := workspaces[i%len(workspaces)]
		
		projects[i] = &models.Project{
			ID:          fmt.Sprintf("proj-%06d", i),
			WorkspaceID: workspace.ID,
			Name:        fmt.Sprintf("project-%d", i),
			Path:        fmt.Sprintf("/tmp/project-%d", i),
			Language:    []string{"go", "python", "javascript", "java"}[i%4],
			Status:      models.ProjectStatusActive,
			CreatedAt:   time.Now().Add(-time.Duration(i) * time.Minute),
			UpdatedAt:   time.Now(),
			Version:     1,
		}
	}
	
	return projects
}

func generateTestSessions(count int, projects []*models.Project) []*models.Session {
	sessions := make([]*models.Session, count)
	
	for i := 0; i < count; i++ {
		project := projects[i%len(projects)]
		
		sessions[i] = &models.Session{
			ID:           fmt.Sprintf("sess-%06d", i),
			ProjectID:    project.ID,
			ProcessID:    int32(1000 + i),
			Status:       []models.SessionStatus{models.SessionStatusActive, models.SessionStatusIdle, models.SessionStatusEnded}[i%3],
			StartedAt:    timePtr(time.Now().Add(-time.Duration(i) * time.Minute)),
			LastActive:   time.Now().Add(-time.Duration(i) * time.Second),
			CommandCount: int64(rand.Intn(100)),
			CreatedAt:    time.Now().Add(-time.Duration(i) * time.Minute),
			UpdatedAt:    time.Now(),
			Version:      1,
		}
	}
	
	return sessions
}

func generateTestTasks(count int, sessions []*models.Session) []*models.Task {
	tasks := make([]*models.Task, count)
	
	for i := 0; i < count; i++ {
		session := sessions[i%len(sessions)]
		
		tasks[i] = &models.Task{
			ID:          fmt.Sprintf("task-%06d", i),
			SessionID:   session.ID,
			Command:     fmt.Sprintf("command-%d", i),
			Status:      []models.TaskStatus{models.TaskStatusCompleted, models.TaskStatusRunning, models.TaskStatusPending}[i%3],
			StartedAt:   timePtr(time.Now().Add(-time.Duration(i) * time.Second)),
			CompletedAt: timePtr(time.Now().Add(-time.Duration(i-1) * time.Second)),
			Duration:    int64(rand.Intn(10000)), // 0-10초
			CreatedAt:   time.Now().Add(-time.Duration(i) * time.Second),
			UpdatedAt:   time.Now(),
			Version:     1,
		}
	}
	
	return tasks
}

func timePtr(t time.Time) *time.Time {
	return &t
}

// 스토리지별 벤치마크 테스트

// BenchmarkMemoryStorage 메모리 스토리지 벤치마크
func BenchmarkMemoryStorage(b *testing.B) {
	config := DefaultBenchmarkConfig()
	
	b.Run("Memory-Create", func(b *testing.B) {
		benchmarkStorageCreate(b, func() (Storage, func()) {
			storage := memory.New()
			return storage, func() { storage.Close() }
		}, config)
	})
	
	b.Run("Memory-Read", func(b *testing.B) {
		benchmarkStorageRead(b, func() (Storage, func()) {
			storage := memory.New()
			return storage, func() { storage.Close() }
		}, config)
	})
	
	b.Run("Memory-Update", func(b *testing.B) {
		benchmarkStorageUpdate(b, func() (Storage, func()) {
			storage := memory.New()
			return storage, func() { storage.Close() }
		}, config)
	})
	
	b.Run("Memory-List", func(b *testing.B) {
		benchmarkStorageList(b, func() (Storage, func()) {
			storage := memory.New()
			return storage, func() { storage.Close() }
		}, config)
	})
}

// BenchmarkSQLiteStorage SQLite 스토리지 벤치마크
func BenchmarkSQLiteStorage(b *testing.B) {
	config := DefaultBenchmarkConfig()
	
	b.Run("SQLite-Create", func(b *testing.B) {
		benchmarkStorageCreate(b, createSQLiteStorage, config)
	})
	
	b.Run("SQLite-Read", func(b *testing.B) {
		benchmarkStorageRead(b, createSQLiteStorage, config)
	})
	
	b.Run("SQLite-Update", func(b *testing.B) {
		benchmarkStorageUpdate(b, createSQLiteStorage, config)
	})
	
	b.Run("SQLite-List", func(b *testing.B) {
		benchmarkStorageList(b, createSQLiteStorage, config)
	})
	
	b.Run("SQLite-IndexOptimized", func(b *testing.B) {
		benchmarkSQLiteIndexOptimized(b, config)
	})
}

// BenchmarkBoltDBStorage BoltDB 스토리지 벤치마크
func BenchmarkBoltDBStorage(b *testing.B) {
	config := DefaultBenchmarkConfig()
	
	b.Run("BoltDB-Create", func(b *testing.B) {
		benchmarkStorageCreate(b, createBoltDBStorage, config)
	})
	
	b.Run("BoltDB-Read", func(b *testing.B) {
		benchmarkStorageRead(b, createBoltDBStorage, config)
	})
	
	b.Run("BoltDB-Update", func(b *testing.B) {
		benchmarkStorageUpdate(b, createBoltDBStorage, config)
	})
	
	b.Run("BoltDB-List", func(b *testing.B) {
		benchmarkStorageList(b, createBoltDBStorage, config)
	})
	
	b.Run("BoltDB-Batch", func(b *testing.B) {
		benchmarkBoltDBBatch(b, config)
	})
}

// 공통 벤치마크 함수들

func benchmarkStorageCreate(b *testing.B, createStorage func() (Storage, func()), config BenchmarkConfig) {
	storage, cleanup := createStorage()
	defer cleanup()
	
	workspaces := generateTestWorkspaces(config.WorkspaceCount)
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		workspace := workspaces[i%len(workspaces)]
		workspace.ID = fmt.Sprintf("bench-ws-%d-%d", i, time.Now().UnixNano())
		
		err := storage.Workspace().Create(context.Background(), workspace)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func benchmarkStorageRead(b *testing.B, createStorage func() (Storage, func()), config BenchmarkConfig) {
	storage, cleanup := createStorage()
	defer cleanup()
	
	// 테스트 데이터 준비
	workspaces := generateTestWorkspaces(config.WorkspaceCount)
	for _, workspace := range workspaces {
		err := storage.Workspace().Create(context.Background(), workspace)
		if err != nil {
			b.Fatal(err)
		}
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		workspace := workspaces[i%len(workspaces)]
		_, err := storage.Workspace().GetByID(context.Background(), workspace.ID)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func benchmarkStorageUpdate(b *testing.B, createStorage func() (Storage, func()), config BenchmarkConfig) {
	storage, cleanup := createStorage()
	defer cleanup()
	
	// 테스트 데이터 준비
	workspaces := generateTestWorkspaces(config.WorkspaceCount)
	for _, workspace := range workspaces {
		err := storage.Workspace().Create(context.Background(), workspace)
		if err != nil {
			b.Fatal(err)
		}
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		workspace := workspaces[i%len(workspaces)]
		updates := map[string]interface{}{
			"name":         fmt.Sprintf("updated-workspace-%d", i),
			"active_tasks": rand.Intn(10),
		}
		
		err := storage.Workspace().Update(context.Background(), workspace.ID, updates)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func benchmarkStorageList(b *testing.B, createStorage func() (Storage, func()), config BenchmarkConfig) {
	storage, cleanup := createStorage()
	defer cleanup()
	
	// 테스트 데이터 준비
	workspaces := generateTestWorkspaces(config.WorkspaceCount)
	for _, workspace := range workspaces {
		err := storage.Workspace().Create(context.Background(), workspace)
		if err != nil {
			b.Fatal(err)
		}
	}
	
	pagination := &models.PaginationRequest{
		Page:  1,
		Limit: 20,
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		ownerID := fmt.Sprintf("user-%d", i%10)
		_, _, err := storage.Workspace().GetByOwnerID(context.Background(), ownerID, pagination)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// SQLite 특화 벤치마크

func benchmarkSQLiteIndexOptimized(b *testing.B, config BenchmarkConfig) {
	storage, cleanup := createSQLiteStorage()
	defer cleanup()
	
	sqliteStorage := storage.(*sqlite.Storage)
	
	// 테스트 데이터 준비
	workspaces := generateTestWorkspaces(config.WorkspaceCount * 10) // 더 많은 데이터
	for _, workspace := range workspaces {
		err := storage.Workspace().Create(context.Background(), workspace)
		if err != nil {
			b.Fatal(err)
		}
	}
	
	// 인덱스 최적화 적용
	err := sqliteStorage.OptimizeIndexes(context.Background())
	if err != nil {
		b.Fatal(err)
	}
	
	pagination := &models.PaginationRequest{
		Page:  1,
		Limit: 100,
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		ownerID := fmt.Sprintf("user-%d", i%10)
		_, _, err := storage.Workspace().GetByOwnerID(context.Background(), ownerID, pagination)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BoltDB 특화 벤치마크

func benchmarkBoltDBBatch(b *testing.B, config BenchmarkConfig) {
	storage, cleanup := createBoltDBStorage()
	defer cleanup()
	
	boltStorage := storage.(*boltdb.Storage)
	batchWriters := boltStorage.GetBatchWriters()
	
	workspaces := generateTestWorkspaces(config.WorkspaceCount)
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		workspace := workspaces[i%len(workspaces)]
		workspace.ID = fmt.Sprintf("batch-ws-%d-%d", i, time.Now().UnixNano())
		
		err := batchWriters.Workspace.WriteWorkspace(workspace, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
	
	// 배치 처리 완료 대기
	err := boltStorage.FlushAndSync(10 * time.Second)
	if err != nil {
		b.Fatal(err)
	}
}

// 동시성 벤치마크

func BenchmarkConcurrentWorkspaceOperations(b *testing.B) {
	config := DefaultBenchmarkConfig()
	
	b.Run("Memory-Concurrent", func(b *testing.B) {
		benchmarkConcurrentOperations(b, func() (Storage, func()) {
			storage := memory.New()
			return storage, func() { storage.Close() }
		}, config)
	})
	
	b.Run("SQLite-Concurrent", func(b *testing.B) {
		benchmarkConcurrentOperations(b, createSQLiteStorage, config)
	})
	
	b.Run("BoltDB-Concurrent", func(b *testing.B) {
		benchmarkConcurrentOperations(b, createBoltDBStorage, config)
	})
}

func benchmarkConcurrentOperations(b *testing.B, createStorage func() (Storage, func()), config BenchmarkConfig) {
	storage, cleanup := createStorage()
	defer cleanup()
	
	workspaces := generateTestWorkspaces(config.WorkspaceCount)
	
	// 테스트 데이터 준비
	for _, workspace := range workspaces {
		err := storage.Workspace().Create(context.Background(), workspace)
		if err != nil {
			b.Fatal(err)
		}
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			workspace := workspaces[i%len(workspaces)]
			
			// 50% 읽기, 30% 업데이트, 20% 생성
			switch i % 10 {
			case 0, 1, 2, 3, 4: // 읽기
				_, err := storage.Workspace().GetByID(context.Background(), workspace.ID)
				if err != nil {
					b.Error(err)
				}
			case 5, 6, 7: // 업데이트
				updates := map[string]interface{}{
					"active_tasks": rand.Intn(10),
				}
				err := storage.Workspace().Update(context.Background(), workspace.ID, updates)
				if err != nil {
					b.Error(err)
				}
			case 8, 9: // 생성
				newWorkspace := &models.Workspace{
					ID:          fmt.Sprintf("concurrent-ws-%d-%d", i, time.Now().UnixNano()),
					Name:        fmt.Sprintf("concurrent-workspace-%d", i),
					ProjectPath: fmt.Sprintf("/tmp/concurrent-%d", i),
					Status:      models.WorkspaceStatusActive,
					OwnerID:     fmt.Sprintf("user-%d", i%10),
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
					Version:     1,
				}
				err := storage.Workspace().Create(context.Background(), newWorkspace)
				if err != nil {
					b.Error(err)
				}
			}
			
			i++
		}
	})
}

// 캐시 벤치마크

func BenchmarkQueryCacheOperations(b *testing.B) {
	memCache := cache.NewMemoryCache(cache.DefaultMemoryCacheConfig())
	queryCache := cache.NewQueryCache(cache.DefaultQueryCacheConfig(memCache))
	defer queryCache.Close()
	
	workspace := &models.Workspace{
		ID:          "test-workspace",
		Name:        "Test Workspace",
		ProjectPath: "/tmp/test",
		Status:      models.WorkspaceStatusActive,
		OwnerID:     "test-user",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Version:     1,
	}
	
	// 캐시에 데이터 저장
	err := queryCache.SetWorkspace(context.Background(), workspace)
	require.NoError(b, err)
	
	b.Run("Cache-Hit", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			_, err := queryCache.GetWorkspace(context.Background(), workspace.ID)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
	
	b.Run("Cache-Miss", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			_, err := queryCache.GetWorkspace(context.Background(), fmt.Sprintf("missing-%d", i))
			if err != cache.ErrCacheMiss {
				b.Fatal("Expected cache miss")
			}
		}
	})
	
	b.Run("Cache-Set", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			testWs := &models.Workspace{
				ID:          fmt.Sprintf("bench-ws-%d", i),
				Name:        fmt.Sprintf("Bench Workspace %d", i),
				ProjectPath: fmt.Sprintf("/tmp/bench-%d", i),
				Status:      models.WorkspaceStatusActive,
				OwnerID:     "bench-user",
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
				Version:     1,
			}
			
			err := queryCache.SetWorkspace(context.Background(), testWs)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// 모니터링 벤치마크

func BenchmarkQueryMonitoring(b *testing.B) {
	logger := zap.NewNop()
	monitor := monitoring.NewMonitor(monitoring.DefaultMonitorConfig())
	defer monitor.Disable()
	
	b.Run("Monitoring-Disabled", func(b *testing.B) {
		monitor.Disable()
		
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			opts := monitoring.WrapOptions{
				Query:       "SELECT * FROM workspaces WHERE owner_id = ?",
				QueryType:   "select",
				StorageType: "test",
				Operation:   "benchmark",
			}
			
			err := monitor.WrapQuery(context.Background(), opts, func() error {
				time.Sleep(time.Microsecond) // 시뮬레이션
				return nil
			})
			if err != nil {
				b.Fatal(err)
			}
		}
	})
	
	b.Run("Monitoring-Enabled", func(b *testing.B) {
		monitor.Enable()
		
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			opts := monitoring.WrapOptions{
				Query:       "SELECT * FROM workspaces WHERE owner_id = ?",
				QueryType:   "select",
				StorageType: "test",
				Operation:   "benchmark",
			}
			
			err := monitor.WrapQuery(context.Background(), opts, func() error {
				time.Sleep(time.Microsecond) // 시뮬레이션
				return nil
			})
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// 스트레스 테스트

func BenchmarkStressTest(b *testing.B) {
	if testing.Short() {
		b.Skip("스트레스 테스트는 -short 플래그와 함께 실행되지 않습니다")
	}
	
	config := BenchmarkConfig{
		WorkspaceCount: 10000,
		ProjectCount:   50000,
		SessionCount:   100000,
		TaskCount:      500000,
		Iterations:     10000,
		Concurrency:    50,
		Logger:         zap.NewNop(),
	}
	
	b.Run("SQLite-Stress", func(b *testing.B) {
		benchmarkStressTest(b, createSQLiteStorage, config)
	})
	
	b.Run("BoltDB-Stress", func(b *testing.B) {
		benchmarkStressTest(b, createBoltDBStorage, config)
	})
}

func benchmarkStressTest(b *testing.B, createStorage func() (Storage, func()), config BenchmarkConfig) {
	storage, cleanup := createStorage()
	defer cleanup()
	
	// 대량 데이터 생성 및 삽입
	workspaces := generateTestWorkspaces(config.WorkspaceCount)
	projects := generateTestProjects(config.ProjectCount, workspaces)
	sessions := generateTestSessions(config.SessionCount, projects)
	tasks := generateTestTasks(config.TaskCount, sessions)
	
	b.Logf("데이터 준비: 워크스페이스 %d, 프로젝트 %d, 세션 %d, 태스크 %d",
		len(workspaces), len(projects), len(sessions), len(tasks))
	
	// 워크스페이스 삽입
	for _, workspace := range workspaces {
		err := storage.Workspace().Create(context.Background(), workspace)
		if err != nil {
			b.Fatal(err)
		}
	}
	
	// 프로젝트 삽입  
	for _, project := range projects {
		err := storage.Project().Create(context.Background(), project)
		if err != nil {
			b.Fatal(err)
		}
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	// 동시 읽기/쓰기 부하 테스트
	var wg sync.WaitGroup
	
	for i := 0; i < config.Concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			
			for j := 0; j < b.N/config.Concurrency; j++ {
				// 다양한 작업 수행
				switch j % 4 {
				case 0: // 워크스페이스 조회
					workspace := workspaces[j%len(workspaces)]
					_, err := storage.Workspace().GetByID(context.Background(), workspace.ID)
					if err != nil {
						b.Error(err)
					}
					
				case 1: // 프로젝트 목록 조회
					workspace := workspaces[j%len(workspaces)]
					pagination := &models.PaginationRequest{Page: 1, Limit: 20}
					_, _, err := storage.Project().GetByWorkspaceID(context.Background(), workspace.ID, pagination)
					if err != nil {
						b.Error(err)
					}
					
				case 2: // 워크스페이스 업데이트
					workspace := workspaces[j%len(workspaces)]
					updates := map[string]interface{}{
						"active_tasks": rand.Intn(10),
					}
					err := storage.Workspace().Update(context.Background(), workspace.ID, updates)
					if err != nil {
						b.Error(err)
					}
					
				case 3: // 소유자별 워크스페이스 목록
					ownerID := fmt.Sprintf("user-%d", j%10)
					pagination := &models.PaginationRequest{Page: 1, Limit: 10}
					_, _, err := storage.Workspace().GetByOwnerID(context.Background(), ownerID, pagination)
					if err != nil {
						b.Error(err)
					}
				}
			}
		}(i)
	}
	
	wg.Wait()
}

// 헬퍼 함수들

func createSQLiteStorage() (Storage, func()) {
	tempDir, err := os.MkdirTemp("", "aicli-bench-sqlite-*")
	if err != nil {
		panic(err)
	}
	
	dbPath := filepath.Join(tempDir, "test.db")
	config := sqlite.DefaultConfig()
	config.DataSource = dbPath
	config.Logger = zap.NewNop()
	
	storage, err := sqlite.New(config)
	if err != nil {
		os.RemoveAll(tempDir)
		panic(err)
	}
	
	cleanup := func() {
		storage.Close()
		os.RemoveAll(tempDir)
	}
	
	return storage, cleanup
}

func createBoltDBStorage() (Storage, func()) {
	tempDir, err := os.MkdirTemp("", "aicli-bench-bolt-*")
	if err != nil {
		panic(err)
	}
	
	dbPath := filepath.Join(tempDir, "test.db")
	config := boltdb.DefaultConfig(dbPath)
	
	storage, err := boltdb.New(config)
	if err != nil {
		os.RemoveAll(tempDir)
		panic(err)
	}
	
	cleanup := func() {
		storage.Close()
		os.RemoveAll(tempDir)
	}
	
	return storage, cleanup
}

// 벤치마크 결과 분석을 위한 도구

type BenchmarkResult struct {
	Name            string        `json:"name"`
	Duration        time.Duration `json:"duration"`
	OperationsPerSec float64      `json:"operations_per_sec"`
	AllocsPerOp     int64         `json:"allocs_per_op"`
	BytesPerOp      int64         `json:"bytes_per_op"`
	Success         bool          `json:"success"`
}

func BenchmarkComparison(b *testing.B) {
	if testing.Short() {
		b.Skip("비교 벤치마크는 -short 플래그와 함께 실행되지 않습니다")
	}
	
	storageTypes := []struct {
		name    string
		factory func() (Storage, func())
	}{
		{"Memory", func() (Storage, func()) {
			storage := memory.New()
			return storage, func() { storage.Close() }
		}},
		{"SQLite", createSQLiteStorage},
		{"BoltDB", createBoltDBStorage},
	}
	
	operations := []struct {
		name string
		fn   func(b *testing.B, createStorage func() (Storage, func()))
	}{
		{"Create", func(b *testing.B, createStorage func() (Storage, func())) {
			benchmarkStorageCreate(b, createStorage, DefaultBenchmarkConfig())
		}},
		{"Read", func(b *testing.B, createStorage func() (Storage, func())) {
			benchmarkStorageRead(b, createStorage, DefaultBenchmarkConfig())
		}},
		{"Update", func(b *testing.B, createStorage func() (Storage, func())) {
			benchmarkStorageUpdate(b, createStorage, DefaultBenchmarkConfig())
		}},
		{"List", func(b *testing.B, createStorage func() (Storage, func())) {
			benchmarkStorageList(b, createStorage, DefaultBenchmarkConfig())
		}},
	}
	
	for _, storage := range storageTypes {
		for _, operation := range operations {
			testName := fmt.Sprintf("%s-%s", storage.name, operation.name)
			
			b.Run(testName, func(b *testing.B) {
				operation.fn(b, storage.factory)
			})
		}
	}
}