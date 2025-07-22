package mount

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aicli/aicli-web/internal/models"
)

func TestManager_CreateWorkspaceMount(t *testing.T) {
	manager := NewManager()
	tempDir := t.TempDir()

	workspace := &models.Workspace{
		ID:          "test-workspace-1",
		ProjectPath: tempDir,
	}

	t.Run("successful creation", func(t *testing.T) {
		config, err := manager.CreateWorkspaceMount(workspace)
		require.NoError(t, err)

		assert.NotNil(t, config)
		assert.Equal(t, tempDir, config.SourcePath[:len(tempDir)]) // 절대 경로로 변환되므로 prefix 확인
		assert.Equal(t, "/workspace", config.TargetPath)
		assert.False(t, config.ReadOnly)
		assert.Equal(t, 1000, config.UserID)
		assert.Equal(t, 1000, config.GroupID)
		assert.True(t, config.NoSuid)
		assert.True(t, config.NoDev)
		assert.Equal(t, workspace.ID, config.WorkspaceID)
		assert.NotEmpty(t, config.ExcludePatterns)
		assert.NotZero(t, config.CreatedAt)
	})

	t.Run("invalid project path", func(t *testing.T) {
		invalidWorkspace := &models.Workspace{
			ID:          "test-workspace-invalid",
			ProjectPath: "/non/existent/path",
		}

		config, err := manager.CreateWorkspaceMount(invalidWorkspace)
		assert.Error(t, err)
		assert.Nil(t, config)
		assert.Contains(t, err.Error(), "invalid project path")
	})
}

func TestManager_CreateCustomMount(t *testing.T) {
	manager := NewManager()
	tempDir := t.TempDir()

	t.Run("successful custom mount", func(t *testing.T) {
		req := &CreateMountRequest{
			WorkspaceID:     "test-workspace",
			SourcePath:      tempDir,
			TargetPath:      "/data",
			ReadOnly:        true,
			UserID:          1001,
			GroupID:         1001,
			SyncMode:        SyncModeCached,
			ExcludePatterns: []string{"*.tmp"},
			NoExec:          true,
			NoSuid:          true,
			NoDev:           true,
		}

		config, err := manager.CreateCustomMount(req)
		require.NoError(t, err)

		assert.NotNil(t, config)
		assert.Equal(t, req.TargetPath, config.TargetPath)
		assert.Equal(t, req.ReadOnly, config.ReadOnly)
		assert.Equal(t, req.UserID, config.UserID)
		assert.Equal(t, req.GroupID, config.GroupID)
		assert.Equal(t, req.SyncMode, config.SyncMode)
		assert.Equal(t, req.NoExec, config.NoExec)
		assert.Contains(t, config.ExcludePatterns, "*.tmp")
	})

	t.Run("auto sync mode selection", func(t *testing.T) {
		req := &CreateMountRequest{
			WorkspaceID: "test-workspace",
			SourcePath:  tempDir,
			TargetPath:  "/data",
			// SyncMode 비워두면 자동 선택
		}

		config, err := manager.CreateCustomMount(req)
		require.NoError(t, err)

		// 자동으로 SyncMode가 설정되어야 함
		assert.NotEmpty(t, config.SyncMode)
	})

	t.Run("invalid source path", func(t *testing.T) {
		req := &CreateMountRequest{
			SourcePath: "/non/existent",
			TargetPath: "/data",
		}

		config, err := manager.CreateCustomMount(req)
		assert.Error(t, err)
		assert.Nil(t, config)
	})
}

func TestManager_ValidateMountConfig(t *testing.T) {
	manager := NewManager()
	tempDir := t.TempDir()

	t.Run("valid config", func(t *testing.T) {
		config := &MountConfig{
			SourcePath: tempDir,
			TargetPath: "/workspace",
			UserID:     1000,
			GroupID:    1000,
			SyncMode:   SyncModeNative,
		}

		err := manager.ValidateMountConfig(config)
		assert.NoError(t, err)
	})

	t.Run("nil config", func(t *testing.T) {
		err := manager.ValidateMountConfig(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "mount config is required")
	})

	t.Run("empty source path", func(t *testing.T) {
		config := &MountConfig{
			TargetPath: "/workspace",
			UserID:     1000,
			GroupID:    1000,
			SyncMode:   SyncModeNative,
		}

		err := manager.ValidateMountConfig(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "source path is required")
	})

	t.Run("invalid user ID", func(t *testing.T) {
		config := &MountConfig{
			SourcePath: tempDir,
			TargetPath: "/workspace",
			UserID:     -1,
			GroupID:    1000,
			SyncMode:   SyncModeNative,
		}

		err := manager.ValidateMountConfig(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid user ID")
	})

	t.Run("invalid sync mode", func(t *testing.T) {
		config := &MountConfig{
			SourcePath: tempDir,
			TargetPath: "/workspace",
			UserID:     1000,
			GroupID:    1000,
			SyncMode:   SyncMode("invalid"),
		}

		err := manager.ValidateMountConfig(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid sync mode")
	})
}

func TestManager_ToDockerMount(t *testing.T) {
	manager := NewManager()
	tempDir := t.TempDir()

	t.Run("successful conversion", func(t *testing.T) {
		config := &MountConfig{
			SourcePath: tempDir,
			TargetPath: "/workspace",
			ReadOnly:   false,
			UserID:     1000,
			GroupID:    1000,
			SyncMode:   SyncModeCached,
		}

		dockerMount, err := manager.ToDockerMount(config)
		require.NoError(t, err)

		assert.Equal(t, "bind", string(dockerMount.Type))
		assert.Equal(t, config.SourcePath, dockerMount.Source)
		assert.Equal(t, config.TargetPath, dockerMount.Target)
		assert.Equal(t, config.ReadOnly, dockerMount.ReadOnly)
		assert.NotNil(t, dockerMount.BindOptions)
		assert.Equal(t, "cached", string(dockerMount.Consistency))
	})

	t.Run("read-only mount", func(t *testing.T) {
		config := &MountConfig{
			SourcePath: tempDir,
			TargetPath: "/workspace",
			ReadOnly:   true,
			UserID:     1000,
			GroupID:    1000,
			SyncMode:   SyncModeNative,
		}

		dockerMount, err := manager.ToDockerMount(config)
		require.NoError(t, err)

		assert.True(t, dockerMount.ReadOnly)
		assert.Empty(t, string(dockerMount.Consistency)) // Native 모드는 기본 일관성 사용
	})

	t.Run("invalid config", func(t *testing.T) {
		config := &MountConfig{
			// SourcePath 누락
			TargetPath: "/workspace",
		}

		dockerMount, err := manager.ToDockerMount(config)
		assert.Error(t, err)
		assert.Equal(t, "", dockerMount.Source)
	})
}

func TestManager_GetMountStatus(t *testing.T) {
	manager := NewManager()
	tempDir := t.TempDir()
	ctx := context.Background()

	// 테스트 파일 생성
	testFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	config := &MountConfig{
		SourcePath: tempDir,
		TargetPath: "/workspace",
		UserID:     1000,
		GroupID:    1000,
		SyncMode:   SyncModeNative,
	}

	t.Run("available mount", func(t *testing.T) {
		status, err := manager.GetMountStatus(ctx, config)
		require.NoError(t, err)

		assert.NotNil(t, status)
		assert.True(t, status.Available)
		assert.Empty(t, status.Error)
		assert.Equal(t, config.SourcePath, status.SourcePath)
		assert.Equal(t, config.TargetPath, status.TargetPath)
		assert.NotZero(t, status.CheckedAt)
		assert.NotNil(t, status.DiskUsage)
	})

	t.Run("unavailable mount", func(t *testing.T) {
		invalidConfig := &MountConfig{
			SourcePath: "/non/existent/path",
			TargetPath: "/workspace",
		}

		status, err := manager.GetMountStatus(ctx, invalidConfig)
		require.NoError(t, err)

		assert.NotNil(t, status)
		assert.False(t, status.Available)
		assert.NotEmpty(t, status.Error)
	})
}

func TestManager_RefreshMountConfig(t *testing.T) {
	manager := NewManager()
	tempDir := t.TempDir()

	config := &MountConfig{
		SourcePath:   tempDir,
		TargetPath:   "/workspace",
		UserID:       1000,
		GroupID:      1000,
		SyncMode:     SyncModeNative,
		LastChecked:  time.Time{}, // 초기값
	}

	t.Run("successful refresh", func(t *testing.T) {
		err := manager.RefreshMountConfig(config)
		require.NoError(t, err)

		assert.NotZero(t, config.LastChecked)
	})

	t.Run("invalid path refresh", func(t *testing.T) {
		invalidConfig := &MountConfig{
			SourcePath: "/non/existent",
			TargetPath: "/workspace",
		}

		err := manager.RefreshMountConfig(invalidConfig)
		assert.Error(t, err)
	})
}

func TestManager_getDefaultExcludePatterns(t *testing.T) {
	manager := NewManager()

	patterns := manager.getDefaultExcludePatterns()

	assert.NotEmpty(t, patterns)
	
	// 중요한 제외 패턴들이 포함되어 있는지 확인
	expectedPatterns := []string{".git", "node_modules", "*.log", ".DS_Store"}
	for _, expected := range expectedPatterns {
		assert.Contains(t, patterns, expected)
	}
}

func TestManager_syncModeToConsistency(t *testing.T) {
	manager := NewManager()

	tests := []struct {
		mode     SyncMode
		expected string
	}{
		{SyncModeCached, "cached"},
		{SyncModeDelegated, "delegated"},
		{SyncModeNative, ""},
		{SyncModeOptimized, ""},
	}

	for _, test := range tests {
		result := manager.syncModeToConsistency(test.mode)
		assert.Equal(t, test.expected, result, "SyncMode %s should return %s", test.mode, test.expected)
	}
}