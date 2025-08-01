package git

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAdvancedWorktreeManager_Concurrency 동시성 제어 테스트
func TestAdvancedWorktreeManager_Concurrency(t *testing.T) {
	// 기본 매니저 설정
	baseManager, _ := setupTestManager(t)
	ctx := context.Background()

	// 테스트용 저장소 복제
	testRepoURL := "https://github.com/git-fixtures/basic.git"
	repo, err := baseManager.Clone(ctx, testRepoURL, "", CloneOptions{Timeout: 30 * time.Second})
	require.NoError(t, err)

	// 고급 매니저 생성 (동시성 제한: 3)
	advManager := NewAdvancedWorktreeManager(baseManager, AdvancedOptions{
		ConcurrencyLimit:  3,
		MaxCacheSize:      10,
		DisablePrometheus: true,
	})

	// 10개의 동시 worktree 생성 시도
	var wg sync.WaitGroup
	errors := make([]error, 10)
	startTime := time.Now()

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			name := fmt.Sprintf("concurrent-wt-%d", idx)
			_, err := advManager.CreateWorktree(ctx, repo, name, WorktreeOptions{})
			errors[idx] = err
		}(i)
	}

	wg.Wait()
	duration := time.Since(startTime)

	// 모든 작업이 성공해야 함
	for i, err := range errors {
		assert.NoError(t, err, "Worktree %d 생성 실패", i)
	}

	// 동시성 제한이 작동하는지 확인
	// 최소한 일부 작업은 대기해야 함
	// 10개 작업, 동시 3개 = 최소 4번의 배치
	// 각 작업이 0.05초 이상 걸린다고 가정하면 최소 0.2초
	assert.Greater(t, duration.Seconds(), 0.05, "너무 빨리 완료됨 - 동시성 제한이 작동하지 않음")

	// 정리
	advManager.Stop()
}

// TestAdvancedWorktreeManager_LRUCache LRU 캐시 테스트
func TestAdvancedWorktreeManager_LRUCache(t *testing.T) {
	// 기본 매니저 설정
	baseManager, _ := setupTestManager(t)
	ctx := context.Background()

	// 테스트용 저장소 복제
	testRepoURL := "https://github.com/git-fixtures/basic.git"
	repo, err := baseManager.Clone(ctx, testRepoURL, "", CloneOptions{Timeout: 30 * time.Second})
	require.NoError(t, err)

	// 고급 매니저 생성 (캐시 크기: 3)
	advManager := NewAdvancedWorktreeManager(baseManager, AdvancedOptions{
		MaxCacheSize:      3,
		DisablePrometheus: true,
	})

	// 첫 번째 worktree 생성
	wt1, err := advManager.CreateWorktree(ctx, repo, "cache-test-1", WorktreeOptions{})
	require.NoError(t, err)
	assert.NotNil(t, wt1)

	// 캐시에서 조회 (캐시 히트)
	cached := advManager.GetWorktree(repo, "cache-test-1")
	assert.NotNil(t, cached)
	assert.Equal(t, wt1.ID, cached.ID)

	// 캐시 히트 확인
	metrics := advManager.GetMetrics()
	assert.Equal(t, int64(1), metrics.CacheHits)

	// 존재하지 않는 worktree 조회 (캐시 미스는 기록하지 않음)
	notFound := advManager.GetWorktree(repo, "not-exists")
	assert.Nil(t, notFound)

	// 캐시 크기 초과 테스트
	for i := 2; i <= 4; i++ {
		name := fmt.Sprintf("cache-test-%d", i)
		_, err := advManager.CreateWorktree(ctx, repo, name, WorktreeOptions{})
		require.NoError(t, err)
	}

	// 캐시 크기 확인 (최대 3개)
	assert.Equal(t, 3, len(advManager.cache))

	// 정리
	advManager.Stop()
}

// TestAdvancedWorktreeManager_SparseCheckout sparse checkout 테스트
func TestAdvancedWorktreeManager_SparseCheckout(t *testing.T) {
	// 기본 매니저 설정
	baseManager, _ := setupTestManager(t)
	ctx := context.Background()

	// 테스트용 저장소 복제
	testRepoURL := "https://github.com/git-fixtures/basic.git"
	repo, err := baseManager.Clone(ctx, testRepoURL, "", CloneOptions{Timeout: 30 * time.Second})
	require.NoError(t, err)

	// sparse checkout 경로 지정
	sparsePaths := []string{"src/", "README.md"}

	// worktree 생성 with sparse checkout
	wt, err := baseManager.CreateWorktree(ctx, repo, "sparse-test", WorktreeOptions{
		SparseCheckoutPaths: sparsePaths,
	})
	require.NoError(t, err)
	assert.NotNil(t, wt)

	// sparse checkout 매니저로 경로 확인
	// 테스트용으로 임시 디렉토리 사용
	sparseManager := NewSparseCheckoutManager("")
	paths, err := sparseManager.GetSparseCheckoutPaths(wt.Path)
	require.NoError(t, err)
	assert.ElementsMatch(t, sparsePaths, paths)
}

// TestAdvancedWorktreeManager_Metrics 메트릭 수집 테스트
func TestAdvancedWorktreeManager_Metrics(t *testing.T) {
	// 기본 매니저 설정
	baseManager, _ := setupTestManager(t)
	ctx := context.Background()

	// 고급 매니저 생성
	advManager := NewAdvancedWorktreeManager(baseManager, AdvancedOptions{
		DisablePrometheus: true,
	})

	// 테스트용 저장소 복제
	testRepoURL := "https://github.com/git-fixtures/basic.git"
	repo, err := advManager.Clone(ctx, testRepoURL, "", CloneOptions{Timeout: 30 * time.Second})
	require.NoError(t, err)

	// worktree 생성
	_, err = advManager.CreateWorktree(ctx, repo, "metrics-test", WorktreeOptions{})
	require.NoError(t, err)

	// 메트릭 확인
	metrics := advManager.GetMetrics()
	assert.Equal(t, int64(1), metrics.CloneOperations)
	assert.Equal(t, int64(1), metrics.TotalCreated)
	assert.Equal(t, int64(0), metrics.TotalDeleted)
	assert.Greater(t, len(metrics.CloneDurations), 0)
	assert.Greater(t, len(metrics.CreateDurations), 0)

	// worktree 삭제
	err = advManager.RemoveWorktree(ctx, repo, "metrics-test")
	require.NoError(t, err)

	// 삭제 메트릭 확인
	metrics = advManager.GetMetrics()
	assert.Equal(t, int64(1), metrics.TotalDeleted)

	// 정리
	advManager.Stop()
}

// TestAdvancedWorktreeManager_ShallowClone shallow clone 테스트
func TestAdvancedWorktreeManager_ShallowClone(t *testing.T) {
	// 기본 매니저 설정
	baseManager, _ := setupTestManager(t)
	ctx := context.Background()

	// 고급 매니저 생성
	advManager := NewAdvancedWorktreeManager(baseManager, AdvancedOptions{
		DisablePrometheus: true,
	})

	// shallow clone 실행
	testRepoURL := "https://github.com/git-fixtures/basic.git"
	repo, err := advManager.Clone(ctx, testRepoURL, "", CloneOptions{
		// Depth가 0이면 자동으로 shallow clone 사용
		Depth: 0,
	})
	require.NoError(t, err)
	assert.NotNil(t, repo)

	// 메트릭에서 clone 작업 확인
	metrics := advManager.GetMetrics()
	assert.Equal(t, int64(1), metrics.CloneOperations)

	// 정리
	advManager.Stop()
}

// TestSparseCheckoutManager sparse checkout 매니저 단위 테스트
func TestSparseCheckoutManager(t *testing.T) {
	manager, tempDir := setupTestManager(t)
	ctx := context.Background()

	// 테스트용 저장소 복제
	testRepoURL := "https://github.com/git-fixtures/basic.git"
	repo, err := manager.Clone(ctx, testRepoURL, "", CloneOptions{Timeout: 30 * time.Second})
	require.NoError(t, err)

	// worktree 생성
	wt, err := manager.CreateWorktree(ctx, repo, "sparse-unit-test", WorktreeOptions{})
	require.NoError(t, err)

	sparseManager := NewSparseCheckoutManager(tempDir)

	t.Run("Enable sparse checkout", func(t *testing.T) {
		paths := []string{"src/", "*.md"}
		err := sparseManager.EnableSparseCheckout(wt.Path, paths)
		assert.NoError(t, err)

		// 경로 확인
		actualPaths, err := sparseManager.GetSparseCheckoutPaths(wt.Path)
		require.NoError(t, err)
		assert.ElementsMatch(t, paths, actualPaths)
	})

	t.Run("Update sparse checkout paths", func(t *testing.T) {
		newPaths := []string{"docs/", "LICENSE"}
		err := sparseManager.UpdateSparseCheckoutPaths(wt.Path, newPaths)
		assert.NoError(t, err)

		// 업데이트된 경로 확인
		actualPaths, err := sparseManager.GetSparseCheckoutPaths(wt.Path)
		require.NoError(t, err)
		assert.ElementsMatch(t, newPaths, actualPaths)
	})

	t.Run("Disable sparse checkout", func(t *testing.T) {
		err := sparseManager.DisableSparseCheckout(wt.Path)
		assert.NoError(t, err)

		// sparse checkout 비활성화 확인
		paths, err := sparseManager.GetSparseCheckoutPaths(wt.Path)
		require.NoError(t, err)
		assert.Empty(t, paths)
	})

	t.Run("Invalid paths", func(t *testing.T) {
		invalidPaths := []string{
			"/absolute/path", // 절대 경로
			"../parent",      // 상위 디렉토리 참조
			"",               // 빈 경로
		}

		for _, path := range invalidPaths {
			err := sparseManager.EnableSparseCheckout(wt.Path, []string{path})
			assert.Error(t, err)
			gitErr, ok := err.(*Error)
			require.True(t, ok)
			assert.Equal(t, ErrCodeInvalidRef, gitErr.Code)
		}
	})
}

// TestGarbageCollection GC 테스트
func TestGarbageCollection(t *testing.T) {
	// 기본 매니저 설정
	baseManager, _ := setupTestManager(t)

	// 고급 매니저 생성 (짧은 GC 간격)
	advManager := NewAdvancedWorktreeManager(baseManager, AdvancedOptions{
		GCInterval:        100 * time.Millisecond,
		MaxAge:            1 * time.Second,
		DisablePrometheus: true,
	})

	// 캐시에 오래된 엔트리 추가
	advManager.cache["old-entry"] = &cacheEntry{
		worktree: &Worktree{ID: "old"},
		lastUsed: time.Now().Add(-2 * time.Second),
	}

	// GC가 실행될 때까지 대기
	time.Sleep(200 * time.Millisecond)

	// 오래된 엔트리가 삭제되었는지 확인
	_, exists := advManager.cache["old-entry"]
	assert.False(t, exists, "오래된 캐시 엔트리가 삭제되지 않음")

	// 정리
	advManager.Stop()
}
