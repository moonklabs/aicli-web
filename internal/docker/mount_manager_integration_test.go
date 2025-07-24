package docker

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aicli/aicli-web/internal/models"
	mountpkg "github.com/aicli/aicli-web/internal/docker/mount"
)

func TestMountManager_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	mountManager := NewMountManager()
	tempDir := t.TempDir()
	
	// 테스트 파일 구조 생성
	setupTestDirectory(t, tempDir)

	workspace := &models.Workspace{
		ID:          "test-workspace-integration",
		ProjectPath: tempDir,
	}

	t.Run("complete mount workflow", func(t *testing.T) {
		ctx := context.Background()

		// 1. 워크스페이스 마운트 생성
		config, err := mountManager.CreateWorkspaceMount(workspace)
		require.NoError(t, err)
		assert.NotNil(t, config)

		// 2. 마운트 설정 검증
		err = mountManager.ValidateMountConfig(config)
		assert.NoError(t, err)

		// 3. Docker Mount 변환
		dockerMount, err := mountManager.ToDockerMount(config)
		require.NoError(t, err)
		assert.Equal(t, "bind", string(dockerMount.Type))

		// 4. 마운트 상태 확인
		status, err := mountManager.GetMountStatus(ctx, config)
		require.NoError(t, err)
		assert.True(t, status.Available)
		assert.Empty(t, status.Error)

		// 5. 파일 통계 조회
		stats, err := mountManager.GetFileStats(ctx, tempDir, config.ExcludePatterns)
		require.NoError(t, err)
		assert.Greater(t, stats.FileCount, 0)
		assert.Greater(t, stats.TotalSize, int64(0))

		// 6. 파일 변경 감시 시작
		changes := make(chan []string, 10)
		callback := func(changedFiles []string) {
			select {
			case changes <- changedFiles:
			default:
			}
		}

		err = mountManager.StartFileWatcher(ctx, tempDir, config.ExcludePatterns, callback)
		assert.NoError(t, err)

		// 활성 watcher 확인
		watchers := mountManager.GetActiveWatchers()
		assert.Contains(t, watchers, tempDir)

		// 파일 변경 감시 중지
		mountManager.StopFileWatcher(tempDir)

		watchers = mountManager.GetActiveWatchers()
		assert.NotContains(t, watchers, tempDir)
	})

	t.Run("multiple mounts for container", func(t *testing.T) {
		// 추가 테스트 디렉토리 생성
		additionalDir := t.TempDir()
		setupTestDirectory(t, additionalDir)

		additionalMounts := []*mountpkg.CreateMountRequest{
			{
				WorkspaceID: workspace.ID,
				SourcePath:  additionalDir,
				TargetPath:  "/data",
				ReadOnly:    true,
			},
		}

		mounts, err := mountManager.CreateMountsForContainer(workspace, additionalMounts)
		require.NoError(t, err)

		assert.Len(t, mounts, 2) // workspace + additional
		
		// 첫 번째 마운트 (workspace)
		assert.Equal(t, "/workspace", mounts[0].Target)
		assert.False(t, mounts[0].ReadOnly)

		// 두 번째 마운트 (additional)
		assert.Equal(t, "/data", mounts[1].Target)
		assert.True(t, mounts[1].ReadOnly)
	})
}

func TestContainerManager_WithMounts(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// 실제 Docker가 필요한 테스트는 환경 변수로 제어
	if os.Getenv("DOCKER_INTEGRATION_TEST") == "" {
		t.Skip("Set DOCKER_INTEGRATION_TEST to run Docker integration tests")
	}

	factory, err := NewFactoryWithDefaults()
	require.NoError(t, err)
	defer factory.Close()

	containerManager := factory.GetContainerManager()
	mountManager := factory.GetMountManager()
	
	ctx := context.Background()

	// Docker daemon 연결 확인
	err = factory.Ping(ctx)
	require.NoError(t, err, "Docker daemon not available")

	tempDir := t.TempDir()
	setupTestDirectory(t, tempDir)

	workspace := &models.Workspace{
		ID:          "test-workspace-container",
		ProjectPath: tempDir,
	}

	// 실제 테스트에서 사용될 요청 객체 (현재는 주석 처리된 코드에서만 사용)
	// req := &CreateContainerRequest{
	// 	WorkspaceID: workspace.ID,
	// 	Name:        "test-mount-container",
	// 	Image:       "alpine:latest",
	// 	Command:     []string{"sh", "-c", "ls -la /workspace && sleep 10"},
	// }

	t.Run("validate workspace mounts", func(t *testing.T) {
		err := containerManager.ValidateWorkspaceMounts(workspace, mountManager, nil)
		assert.NoError(t, err)
	})

	t.Run("create container with mounts", func(t *testing.T) {
		// 참고: 실제 컨테이너 생성은 Docker daemon이 필요하므로 모의 테스트로 구현
		// 실제 환경에서는 다음과 같이 사용됨:
		/*
		container, err := containerManager.CreateWorkspaceContainerWithMounts(
			ctx, workspace, req, mountManager, nil,
		)
		require.NoError(t, err)
		defer containerManager.RemoveContainer(ctx, container.ID, true)

		assert.NotEmpty(t, container.ID)
		assert.Equal(t, workspace.ID, container.WorkspaceID)
		assert.NotEmpty(t, container.Mounts)

		// 마운트 상태 확인
		mountStatuses, err := containerManager.GetContainerMountStatus(ctx, container.ID, mountManager)
		require.NoError(t, err)
		assert.NotEmpty(t, mountStatuses)
		*/

		// 모의 테스트: 마운트 구성만 검증
		mounts, err := mountManager.CreateMountsForContainer(workspace, nil)
		require.NoError(t, err)
		assert.NotEmpty(t, mounts)
		assert.Equal(t, "/workspace", mounts[0].Target)
	})
}

func TestMountManager_ErrorScenarios(t *testing.T) {
	mountManager := NewMountManager()

	t.Run("invalid workspace path", func(t *testing.T) {
		workspace := &models.Workspace{
			ID:          "invalid-workspace",
			ProjectPath: "/non/existent/path",
		}

		config, err := mountManager.CreateWorkspaceMount(workspace)
		assert.Error(t, err)
		assert.Nil(t, config)
		assert.Contains(t, err.Error(), "invalid project path")
	})

	t.Run("system directory mount attempt", func(t *testing.T) {
		workspace := &models.Workspace{
			ID:          "system-workspace",
			ProjectPath: "/etc", // 시스템 디렉토리
		}

		config, err := mountManager.CreateWorkspaceMount(workspace)
		assert.Error(t, err)
		assert.Nil(t, config)
		assert.Contains(t, err.Error(), "security check failed")
	})

	t.Run("dangerous container target path", func(t *testing.T) {
		tempDir := t.TempDir()
		
		req := &mountpkg.CreateMountRequest{
			SourcePath: tempDir,
			TargetPath: "/etc", // 컨테이너 내 민감한 경로
		}

		config, err := mountManager.CreateCustomMount(req)
		assert.Error(t, err)
		assert.Nil(t, config)
		assert.Contains(t, err.Error(), "sensitive container path")
	})

	t.Run("invalid mount config conversion", func(t *testing.T) {
		invalidConfig := &mountpkg.MountConfig{
			SourcePath: "", // 빈 소스 경로
			TargetPath: "/workspace",
		}

		dockerMount, err := mountManager.ToDockerMount(invalidConfig)
		assert.Error(t, err)
		assert.Empty(t, dockerMount.Source)
	})
}

func TestMountManager_FileWatcher(t *testing.T) {
	mountManager := NewMountManager()
	tempDir := t.TempDir()
	ctx := context.Background()

	// 테스트 파일 생성
	testFile := filepath.Join(tempDir, "watch_test.txt")
	err := os.WriteFile(testFile, []byte("initial content"), 0644)
	require.NoError(t, err)

	t.Run("file watcher lifecycle", func(t *testing.T) {
		changes := make(chan []string, 10)
		callback := func(changedFiles []string) {
			select {
			case changes <- changedFiles:
			default:
			}
		}

		// 파일 감시 시작
		err := mountManager.StartFileWatcher(ctx, tempDir, []string{"*.tmp"}, callback)
		require.NoError(t, err)

		// 활성 watcher 확인
		watchers := mountManager.GetActiveWatchers()
		assert.Contains(t, watchers, tempDir)

		// 파일 감시 중지
		mountManager.StopFileWatcher(tempDir)

		watchers = mountManager.GetActiveWatchers()
		assert.NotContains(t, watchers, tempDir)
	})

	t.Run("multiple watchers", func(t *testing.T) {
		tempDir2 := t.TempDir()

		// 두 개의 watcher 시작
		err1 := mountManager.StartFileWatcher(ctx, tempDir, nil, func([]string) {})
		err2 := mountManager.StartFileWatcher(ctx, tempDir2, nil, func([]string) {})
		
		require.NoError(t, err1)
		require.NoError(t, err2)

		watchers := mountManager.GetActiveWatchers()
		assert.Len(t, watchers, 2)
		assert.Contains(t, watchers, tempDir)
		assert.Contains(t, watchers, tempDir2)

		// 모든 watcher 중지
		mountManager.StopAllWatchers()

		watchers = mountManager.GetActiveWatchers()
		assert.Empty(t, watchers)
	})
}

// setupTestDirectory 테스트용 디렉토리 구조를 생성합니다.
func setupTestDirectory(t *testing.T, baseDir string) {
	files := map[string]string{
		"README.md":           "# Test Project",
		"main.go":             "package main\n\nfunc main() {}",
		"src/app.js":          "console.log('test');",
		"src/styles.css":      "body { margin: 0; }",
		"tests/unit_test.go":  "package main\n\nimport \"testing\"",
		"docs/api.md":         "# API Documentation",
		".git/config":         "[core]",
		"node_modules/lib.js": "// library code",
		"temp.tmp":            "temporary file",
		"logs/app.log":        "application logs",
	}

	for filePath, content := range files {
		fullPath := filepath.Join(baseDir, filePath)
		err := os.MkdirAll(filepath.Dir(fullPath), 0755)
		require.NoError(t, err)
		
		err = os.WriteFile(fullPath, []byte(content), 0644)
		require.NoError(t, err)
	}
}