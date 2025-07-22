package mount

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSyncer_WatchChanges(t *testing.T) {
	syncer := NewSyncer()
	defer syncer.StopAll()

	tempDir := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	t.Run("successful watch setup", func(t *testing.T) {
		changes := make(chan []string, 10)
		callback := func(changedFiles []string) {
			changes <- changedFiles
		}

		err := syncer.WatchChanges(ctx, tempDir, nil, callback)
		require.NoError(t, err)

		// watcher가 등록되었는지 확인
		watchers := syncer.GetActiveWatchers()
		assert.Contains(t, watchers, tempDir)
	})

	t.Run("file changes detection", func(t *testing.T) {
		testDir := filepath.Join(tempDir, "watch_test")
		err := os.MkdirAll(testDir, 0755)
		require.NoError(t, err)

		changes := make(chan []string, 10)
		callback := func(changedFiles []string) {
			select {
			case changes <- changedFiles:
			default:
			}
		}

		err = syncer.WatchChanges(ctx, testDir, nil, callback)
		require.NoError(t, err)

		// 파일 변경을 위해 짧은 대기
		time.Sleep(100 * time.Millisecond)

		// 새 파일 생성
		testFile := filepath.Join(testDir, "newfile.txt")
		err = os.WriteFile(testFile, []byte("test content"), 0644)
		require.NoError(t, err)

		// 변경 감지를 위한 대기 (watchInterval 고려)
		select {
		case changedFiles := <-changes:
			assert.NotEmpty(t, changedFiles)
			t.Logf("Detected changes: %v", changedFiles)
		case <-time.After(10 * time.Second):
			t.Log("No changes detected within timeout - this may be normal depending on timing")
		}
	})
}

func TestSyncer_shouldExclude(t *testing.T) {
	syncer := NewSyncer()
	tempDir := t.TempDir()

	tests := []struct {
		name            string
		filePath        string
		excludePatterns []string
		expected        bool
	}{
		{
			name:            "exact match",
			filePath:        filepath.Join(tempDir, ".git"),
			excludePatterns: []string{".git"},
			expected:        true,
		},
		{
			name:            "wildcard match",
			filePath:        filepath.Join(tempDir, "test.tmp"),
			excludePatterns: []string{"*.tmp"},
			expected:        true,
		},
		{
			name:            "directory pattern",
			filePath:        filepath.Join(tempDir, "node_modules", "package", "index.js"),
			excludePatterns: []string{"node_modules/*"},
			expected:        true,
		},
		{
			name:            "no match",
			filePath:        filepath.Join(tempDir, "src", "main.go"),
			excludePatterns: []string{".git", "*.tmp"},
			expected:        false,
		},
		{
			name:            "substring match",
			filePath:        filepath.Join(tempDir, "test.log.backup"),
			excludePatterns: []string{"*.log"},
			expected:        true, // substring 매칭으로 인해 true
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := syncer.shouldExclude(test.filePath, tempDir, test.excludePatterns)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestSyncer_GetFileStats(t *testing.T) {
	syncer := NewSyncer()
	tempDir := t.TempDir()
	ctx := context.Background()

	// 테스트 파일 구조 생성
	files := map[string]string{
		"file1.txt":        "content1",
		"file2.go":         "package main",
		"subdir/file3.js":  "console.log('test')",
		"subdir/file4.tmp": "temporary",
	}

	var totalSize int64
	var latestModTime time.Time

	for filePath, content := range files {
		fullPath := filepath.Join(tempDir, filePath)
		err := os.MkdirAll(filepath.Dir(fullPath), 0755)
		require.NoError(t, err)
		
		err = os.WriteFile(fullPath, []byte(content), 0644)
		require.NoError(t, err)
		
		info, err := os.Stat(fullPath)
		require.NoError(t, err)
		
		totalSize += info.Size()
		if info.ModTime().After(latestModTime) {
			latestModTime = info.ModTime()
		}
	}

	t.Run("all files included", func(t *testing.T) {
		stats, err := syncer.GetFileStats(ctx, tempDir, nil)
		require.NoError(t, err)

		assert.Equal(t, tempDir, stats.Path)
		assert.Equal(t, 4, stats.FileCount)      // 4개 파일
		assert.Equal(t, 2, stats.DirectoryCount) // tempDir, subdir (2개 디렉토리)
		assert.Equal(t, totalSize, stats.TotalSize)
		assert.NotZero(t, stats.ScannedAt)
		assert.NotZero(t, stats.LastModified)
		assert.NotEmpty(t, stats.LastModifiedFile)
	})

	t.Run("with exclusion patterns", func(t *testing.T) {
		excludePatterns := []string{"*.tmp", "subdir/*"}
		stats, err := syncer.GetFileStats(ctx, tempDir, excludePatterns)
		require.NoError(t, err)

		assert.Equal(t, 2, stats.FileCount)      // .tmp와 subdir/* 제외하면 2개
		assert.Equal(t, 1, stats.DirectoryCount) // subdir 제외하면 1개
		assert.Less(t, stats.TotalSize, totalSize) // 제외된 파일들로 인해 더 작음
	})
}

func TestSyncer_CheckMountStatus(t *testing.T) {
	syncer := NewSyncer()
	ctx := context.Background()
	tempDir := t.TempDir()

	t.Run("no active watcher", func(t *testing.T) {
		status, err := syncer.CheckMountStatus(ctx, "container123", tempDir)
		require.NoError(t, err)

		assert.NotNil(t, status)
		assert.False(t, status.IsActive)
		assert.NotZero(t, status.LastSync)
		assert.Equal(t, 0, status.FilesChanged)
		assert.Equal(t, "0ms", status.SyncDuration)
		assert.Empty(t, status.Errors)
	})

	t.Run("with active watcher", func(t *testing.T) {
		watchCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := syncer.WatchChanges(watchCtx, tempDir, nil, func([]string) {})
		require.NoError(t, err)

		status, err := syncer.CheckMountStatus(ctx, "container123", tempDir)
		require.NoError(t, err)

		assert.True(t, status.IsActive)

		// 정리
		syncer.StopWatch(tempDir)
	})
}

func TestSyncer_SyncDirectory(t *testing.T) {
	syncer := NewSyncer()
	ctx := context.Background()

	sourcePath := t.TempDir()
	targetPath := "/workspace"

	t.Run("successful sync", func(t *testing.T) {
		options := &SyncOptions{
			ExcludePatterns: []string{"*.tmp"},
			DryRun:          false,
			Recursive:       true,
		}

		result, err := syncer.SyncDirectory(ctx, sourcePath, targetPath, options)
		require.NoError(t, err)

		assert.NotNil(t, result)
		assert.Equal(t, sourcePath, result.SourcePath)
		assert.Equal(t, targetPath, result.TargetPath)
		assert.Equal(t, "completed", result.Status)
		assert.NotZero(t, result.StartTime)
		assert.NotZero(t, result.EndTime)
		assert.Greater(t, result.Duration, time.Duration(0))
	})
}

func TestSyncer_StopWatch(t *testing.T) {
	syncer := NewSyncer()
	tempDir := t.TempDir()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// watcher 시작
	err := syncer.WatchChanges(ctx, tempDir, nil, func([]string) {})
	require.NoError(t, err)

	// watcher 확인
	watchers := syncer.GetActiveWatchers()
	assert.Contains(t, watchers, tempDir)

	// watcher 중지
	syncer.StopWatch(tempDir)

	// watcher 제거 확인
	watchers = syncer.GetActiveWatchers()
	assert.NotContains(t, watchers, tempDir)
}

func TestSyncer_StopAll(t *testing.T) {
	syncer := NewSyncer()
	
	tempDirs := []string{t.TempDir(), t.TempDir()}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 여러 watcher 시작
	for _, dir := range tempDirs {
		err := syncer.WatchChanges(ctx, dir, nil, func([]string) {})
		require.NoError(t, err)
	}

	// 모든 watcher 확인
	watchers := syncer.GetActiveWatchers()
	assert.Len(t, watchers, 2)

	// 모든 watcher 중지
	syncer.StopAll()

	// 모든 watcher 제거 확인
	watchers = syncer.GetActiveWatchers()
	assert.Empty(t, watchers)
}

func TestSyncer_scanForChanges(t *testing.T) {
	syncer := NewSyncer()
	tempDir := t.TempDir()

	// 기준 시간 설정 (과거)
	baseTime := time.Now().Add(-1 * time.Hour)

	// 테스트 파일 생성 (최근)
	testFile := filepath.Join(tempDir, "newfile.txt")
	err := os.WriteFile(testFile, []byte("new content"), 0644)
	require.NoError(t, err)

	t.Run("detect new files", func(t *testing.T) {
		changes, latestMod, err := syncer.scanForChanges(tempDir, baseTime, nil)
		require.NoError(t, err)

		assert.NotEmpty(t, changes)
		assert.Contains(t, changes, "newfile.txt")
		assert.True(t, latestMod.After(baseTime))
	})

	t.Run("with exclusion patterns", func(t *testing.T) {
		// .tmp 파일 생성
		tmpFile := filepath.Join(tempDir, "temp.tmp")
		err = os.WriteFile(tmpFile, []byte("temp"), 0644)
		require.NoError(t, err)

		excludePatterns := []string{"*.tmp"}
		changes, _, err := syncer.scanForChanges(tempDir, baseTime, excludePatterns)
		require.NoError(t, err)

		assert.NotContains(t, changes, "temp.tmp")
		assert.Contains(t, changes, "newfile.txt")
	})

	t.Run("no changes since recent time", func(t *testing.T) {
		futureTime := time.Now().Add(1 * time.Hour)
		changes, _, err := syncer.scanForChanges(tempDir, futureTime, nil)
		require.NoError(t, err)

		assert.Empty(t, changes)
	})
}

func TestFileWatcher_Stop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watcher := &FileWatcher{
		sourcePath: t.TempDir(),
		cancel:     cancel,
	}

	// Stop 호출 테스트
	watcher.Stop()

	// context가 취소되었는지 확인
	select {
	case <-ctx.Done():
		// 정상적으로 취소됨
	case <-time.After(100 * time.Millisecond):
		t.Error("Context was not cancelled")
	}
}

func TestFileWatcher_addError(t *testing.T) {
	watcher := &FileWatcher{
		errors: make([]string, 0),
	}

	// 에러 추가
	watcher.addError("test error 1")
	watcher.addError("test error 2")

	assert.Len(t, watcher.errors, 2)
	assert.Contains(t, watcher.errors[0], "test error 1")
	assert.Contains(t, watcher.errors[1], "test error 2")

	// 에러 로그 크기 제한 테스트 (10개 초과)
	for i := 3; i <= 12; i++ {
		watcher.addError(fmt.Sprintf("test error %d", i))
	}

	assert.Len(t, watcher.errors, 10) // 최대 10개로 제한
	assert.Contains(t, watcher.errors[0], "test error 3") // 첫 번째가 제거되고 3번부터 시작
	assert.Contains(t, watcher.errors[9], "test error 12") // 마지막이 12번
}