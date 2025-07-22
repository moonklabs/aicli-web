// Package integration provides performance benchmarks 
// for the workspace Docker system
package integration

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/aicli/aicli-web/internal/models"
)

// WorkspacePerformanceTestSuite provides performance benchmarks and load testing
type WorkspacePerformanceTestSuite struct {
	WorkspaceDockerTestSuite
	
	// 성능 메트릭
	creationTimes []time.Duration
	deletionTimes []time.Duration
	startupTimes  []time.Duration
}

// TestWorkspacePerformanceSuite runs performance tests
func TestWorkspacePerformanceSuite(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker daemon not available, skipping performance tests")
	}
	
	suite.Run(t, new(WorkspacePerformanceTestSuite))
}

// SetupSuite initializes the performance test suite
func (suite *WorkspacePerformanceTestSuite) SetupSuite() {
	suite.WorkspaceDockerTestSuite.SetupSuite()
	
	// 성능 메트릭 초기화
	suite.creationTimes = make([]time.Duration, 0)
	suite.deletionTimes = make([]time.Duration, 0)
	suite.startupTimes = make([]time.Duration, 0)
}

// TestWorkspaceCreationPerformance benchmarks workspace creation performance
func (suite *WorkspacePerformanceTestSuite) TestWorkspaceCreationPerformance() {
	suite.T().Log("Testing workspace creation performance...")
	
	iterations := 10
	ctx := context.Background()
	
	for i := 0; i < iterations; i++ {
		start := time.Now()
		
		req := &models.CreateWorkspaceRequest{
			Name:        fmt.Sprintf("performance-workspace-%d", i),
			ProjectPath: suite.createTempProject(),
			Description: fmt.Sprintf("Performance test workspace %d", i),
		}
		
		workspace, err := suite.dockerService.CreateWorkspace(ctx, req, suite.testUser)
		require.NoError(suite.T(), err)
		
		// 완전히 생성될 때까지 대기
		suite.waitForContainerRunning(workspace.ID, 30*time.Second)
		
		duration := time.Since(start)
		suite.creationTimes = append(suite.creationTimes, duration)
		suite.testWorkspaces = append(suite.testWorkspaces, workspace)
		
		suite.T().Logf("Workspace %d creation time: %v", i+1, duration)
	}
	
	// 성능 분석
	avgDuration := suite.calculateAverage(suite.creationTimes)
	minDuration := suite.calculateMin(suite.creationTimes)
	maxDuration := suite.calculateMax(suite.creationTimes)
	
	suite.T().Logf("Creation Performance Summary:")
	suite.T().Logf("  Average: %v", avgDuration)
	suite.T().Logf("  Min: %v", minDuration)
	suite.T().Logf("  Max: %v", maxDuration)
	suite.T().Logf("  Iterations: %d", iterations)
	
	// 성능 요구사항 검증
	require.Less(suite.T(), avgDuration, 10*time.Second, 
		"Average workspace creation time should be less than 10 seconds")
	require.Less(suite.T(), maxDuration, 15*time.Second,
		"Maximum workspace creation time should be less than 15 seconds")
}

// TestConcurrentOperationsPerformance tests concurrent operations performance
func (suite *WorkspacePerformanceTestSuite) TestConcurrentOperationsPerformance() {
	suite.T().Log("Testing concurrent operations performance...")
	
	concurrency := 20
	ctx := context.Background()
	start := time.Now()
	
	var wg sync.WaitGroup
	results := make(chan time.Duration, concurrency)
	errors := make(chan error, concurrency)
	
	// CPU 코어 수 확인
	numCPU := runtime.NumCPU()
	suite.T().Logf("Running concurrent test on %d CPU cores with %d goroutines", numCPU, concurrency)
	
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			
			goroutineStart := time.Now()
			
			req := &models.CreateWorkspaceRequest{
				Name:        fmt.Sprintf("concurrent-perf-workspace-%d", index),
				ProjectPath: suite.createTempProject(),
				Description: fmt.Sprintf("Concurrent performance test workspace %d", index),
			}
			
			workspace, err := suite.dockerService.CreateWorkspace(ctx, req, suite.testUser)
			if err != nil {
				errors <- err
				return
			}
			
			// 컨테이너가 실행될 때까지 대기
			suite.waitForContainerRunning(workspace.ID, 30*time.Second)
			
			duration := time.Since(goroutineStart)
			results <- duration
			
			suite.testWorkspaces = append(suite.testWorkspaces, workspace)
		}(i)
	}
	
	// 완료 대기
	go func() {
		wg.Wait()
		close(results)
		close(errors)
	}()
	
	// 결과 수집
	var durations []time.Duration
	successCount := 0
	
	for duration := range results {
		durations = append(durations, duration)
		successCount++
	}
	
	// 에러 확인
	errorCount := 0
	for err := range errors {
		suite.T().Logf("Concurrent operation error: %v", err)
		errorCount++
	}
	
	totalDuration := time.Since(start)
	successRate := float64(successCount) / float64(concurrency) * 100
	
	// 성능 분석
	if len(durations) > 0 {
		avgDuration := suite.calculateAverage(durations)
		minDuration := suite.calculateMin(durations)
		maxDuration := suite.calculateMax(durations)
		
		suite.T().Logf("Concurrent Performance Summary:")
		suite.T().Logf("  Total Duration: %v", totalDuration)
		suite.T().Logf("  Success Rate: %.2f%% (%d/%d)", successRate, successCount, concurrency)
		suite.T().Logf("  Average Individual Time: %v", avgDuration)
		suite.T().Logf("  Min Individual Time: %v", minDuration)
		suite.T().Logf("  Max Individual Time: %v", maxDuration)
		suite.T().Logf("  Throughput: %.2f workspaces/sec", float64(successCount)/totalDuration.Seconds())
	}
	
	// 성능 요구사항 검증
	require.GreaterOrEqual(suite.T(), successRate, 80.0, "Success rate should be at least 80%")
	require.Less(suite.T(), totalDuration, 60*time.Second, "Total concurrent operations should complete within 60 seconds")
}

// TestMemoryUsageMonitoring tests memory usage during operations
func (suite *WorkspacePerformanceTestSuite) TestMemoryUsageMonitoring() {
	suite.T().Log("Testing memory usage monitoring...")
	
	// 초기 메모리 상태
	var m1 runtime.MemStats
	runtime.GC() // 가비지 컬렉션 실행
	runtime.ReadMemStats(&m1)
	
	initialAlloc := m1.Alloc
	suite.T().Logf("Initial memory allocation: %d KB", initialAlloc/1024)
	
	ctx := context.Background()
	numWorkspaces := 10
	
	// 워크스페이스 생성
	for i := 0; i < numWorkspaces; i++ {
		req := &models.CreateWorkspaceRequest{
			Name:        fmt.Sprintf("memory-test-workspace-%d", i),
			ProjectPath: suite.createTempProject(),
			Description: fmt.Sprintf("Memory test workspace %d", i),
		}
		
		workspace, err := suite.dockerService.CreateWorkspace(ctx, req, suite.testUser)
		require.NoError(suite.T(), err)
		
		suite.testWorkspaces = append(suite.testWorkspaces, workspace)
		
		// 주기적으로 메모리 사용량 확인
		if i%3 == 0 {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			suite.T().Logf("Memory after %d workspaces: %d KB (heap: %d KB)", 
				i+1, m.Alloc/1024, m.HeapAlloc/1024)
		}
	}
	
	// 최종 메모리 상태
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)
	
	finalAlloc := m2.Alloc
	memoryIncrease := finalAlloc - initialAlloc
	memoryPerWorkspace := memoryIncrease / uint64(numWorkspaces)
	
	suite.T().Logf("Memory Usage Summary:")
	suite.T().Logf("  Initial: %d KB", initialAlloc/1024)
	suite.T().Logf("  Final: %d KB", finalAlloc/1024)
	suite.T().Logf("  Increase: %d KB", memoryIncrease/1024)
	suite.T().Logf("  Per Workspace: %d KB", memoryPerWorkspace/1024)
	suite.T().Logf("  Heap Objects: %d", m2.HeapObjects)
	suite.T().Logf("  GC Cycles: %d", m2.NumGC-m1.NumGC)
	
	// 메모리 사용량이 합리적인 범위 내인지 확인
	require.Less(suite.T(), memoryPerWorkspace, uint64(50*1024*1024), // 50MB per workspace
		"Memory usage per workspace should be reasonable")
}

// TestResourceCleanupEfficiency tests resource cleanup efficiency
func (suite *WorkspacePerformanceTestSuite) TestResourceCleanupEfficiency() {
	suite.T().Log("Testing resource cleanup efficiency...")
	
	ctx := context.Background()
	numWorkspaces := 5
	
	// 워크스페이스 생성
	var workspaces []*models.Workspace
	creationStart := time.Now()
	
	for i := 0; i < numWorkspaces; i++ {
		req := &models.CreateWorkspaceRequest{
			Name:        fmt.Sprintf("cleanup-test-workspace-%d", i),
			ProjectPath: suite.createTempProject(),
			Description: fmt.Sprintf("Cleanup test workspace %d", i),
		}
		
		workspace, err := suite.dockerService.CreateWorkspace(ctx, req, suite.testUser)
		require.NoError(suite.T(), err)
		
		workspaces = append(workspaces, workspace)
	}
	
	creationTime := time.Since(creationStart)
	suite.T().Logf("Created %d workspaces in %v", numWorkspaces, creationTime)
	
	// 워크스페이스 삭제 및 정리 시간 측정
	cleanupStart := time.Now()
	
	for _, workspace := range workspaces {
		deleteStart := time.Now()
		
		err := suite.dockerService.DeleteWorkspace(ctx, workspace.ID, suite.testUser)
		require.NoError(suite.T(), err)
		
		deleteTime := time.Since(deleteStart)
		suite.deletionTimes = append(suite.deletionTimes, deleteTime)
		
		suite.T().Logf("Deleted workspace %s in %v", workspace.ID, deleteTime)
	}
	
	totalCleanupTime := time.Since(cleanupStart)
	
	// 정리 완료 확인
	suite.Eventually(func() bool {
		for _, workspace := range workspaces {
			_, err := suite.dockerService.GetWorkspace(ctx, workspace.ID, suite.testUser)
			if err == nil {
				return false // 아직 존재
			}
		}
		return true
	}, 30*time.Second, 1*time.Second, "All workspaces should be deleted")
	
	// 성능 분석
	avgDeletionTime := suite.calculateAverage(suite.deletionTimes)
	
	suite.T().Logf("Cleanup Performance Summary:")
	suite.T().Logf("  Total Cleanup Time: %v", totalCleanupTime)
	suite.T().Logf("  Average Deletion Time: %v", avgDeletionTime)
	suite.T().Logf("  Cleanup Efficiency: %.2f workspaces/sec", 
		float64(numWorkspaces)/totalCleanupTime.Seconds())
	
	// 성능 요구사항 검증
	require.Less(suite.T(), avgDeletionTime, 5*time.Second,
		"Average deletion time should be less than 5 seconds")
}

// BenchmarkWorkspaceCreation benchmarks workspace creation
func (suite *WorkspacePerformanceTestSuite) BenchmarkWorkspaceCreation() {
	if !isDockerAvailable() {
		suite.T().Skip("Docker not available")
	}
	
	ctx := context.Background()
	
	b := testing.Benchmark(func(b *testing.B) {
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			req := &models.CreateWorkspaceRequest{
				Name:        fmt.Sprintf("benchmark-workspace-%d", i),
				ProjectPath: suite.createTempProject(),
				Description: fmt.Sprintf("Benchmark workspace %d", i),
			}
			
			workspace, err := suite.dockerService.CreateWorkspace(ctx, req, suite.testUser)
			if err != nil {
				b.Fatal(err)
			}
			
			// 정리
			suite.dockerService.DeleteWorkspace(ctx, workspace.ID, suite.testUser)
		}
	})
	
	suite.T().Logf("Benchmark Results:")
	suite.T().Logf("  Iterations: %d", b.N)
	suite.T().Logf("  Total Time: %v", b.T)
	suite.T().Logf("  Average Time per Operation: %v", time.Duration(b.T.Nanoseconds()/int64(b.N)))
	suite.T().Logf("  Operations per Second: %.2f", float64(b.N)/b.T.Seconds())
}

// waitForContainerRunning waits for a container to be running
func (suite *WorkspacePerformanceTestSuite) waitForContainerRunning(workspaceID string, timeout time.Duration) {
	suite.Eventually(func() bool {
		containers, err := suite.getWorkspaceContainers(workspaceID)
		return err == nil && len(containers) > 0 && 
			containers[0].State == "running"
	}, timeout, 1*time.Second, "Container should be running")
}

// calculateAverage calculates the average duration
func (suite *WorkspacePerformanceTestSuite) calculateAverage(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	
	var total time.Duration
	for _, d := range durations {
		total += d
	}
	
	return total / time.Duration(len(durations))
}

// calculateMin calculates the minimum duration
func (suite *WorkspacePerformanceTestSuite) calculateMin(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	
	min := durations[0]
	for _, d := range durations[1:] {
		if d < min {
			min = d
		}
	}
	
	return min
}

// calculateMax calculates the maximum duration
func (suite *WorkspacePerformanceTestSuite) calculateMax(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	
	max := durations[0]
	for _, d := range durations[1:] {
		if d > max {
			max = d
		}
	}
	
	return max
}