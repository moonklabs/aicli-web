package mount

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidator_ValidateProjectPath(t *testing.T) {
	validator := NewValidator()

	t.Run("valid directory", func(t *testing.T) {
		tempDir := t.TempDir()
		err := validator.ValidateProjectPath(tempDir)
		assert.NoError(t, err)
	})

	t.Run("empty path", func(t *testing.T) {
		err := validator.ValidateProjectPath("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "project path is required")
	})

	t.Run("non-existent directory", func(t *testing.T) {
		nonExistentPath := "/non/existent/path"
		err := validator.ValidateProjectPath(nonExistentPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not exist")
	})

	t.Run("file instead of directory", func(t *testing.T) {
		tempFile, err := os.CreateTemp("", "testfile")
		require.NoError(t, err)
		defer os.Remove(tempFile.Name())
		tempFile.Close()

		err = validator.ValidateProjectPath(tempFile.Name())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not a directory")
	})

	t.Run("relative path conversion", func(t *testing.T) {
		// 현재 디렉토리를 상대 경로로 테스트
		err := validator.ValidateProjectPath(".")
		assert.NoError(t, err)
	})
}

func TestValidator_checkSecurity(t *testing.T) {
	validator := NewValidator()

	if runtime.GOOS == "windows" {
		t.Run("windows system paths blocked", func(t *testing.T) {
			systemPaths := []string{
				"C:\\Windows",
				"C:\\Program Files",
				"C:\\System Volume Information",
			}

			for _, path := range systemPaths {
				err := validator.checkSecurity(path)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "cannot mount system directory")
			}
		})
	} else {
		t.Run("unix system paths blocked", func(t *testing.T) {
			systemPaths := []string{
				"/etc",
				"/usr",
				"/bin",
				"/root",
			}

			for _, path := range systemPaths {
				err := validator.checkSecurity(path)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "cannot mount system directory")
			}
		})

		t.Run("docker socket blocked", func(t *testing.T) {
			path := "/var/run/docker.sock"
			err := validator.checkSecurity(path)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "cannot mount docker socket")
		})
	}

	t.Run("safe directory allowed", func(t *testing.T) {
		tempDir := t.TempDir()
		err := validator.checkSecurity(tempDir)
		assert.NoError(t, err)
	})
}

func TestValidator_ValidateMountPath(t *testing.T) {
	validator := NewValidator()
	tempDir := t.TempDir()

	t.Run("valid paths", func(t *testing.T) {
		err := validator.ValidateMountPath(tempDir, "/workspace")
		assert.NoError(t, err)
	})

	t.Run("invalid source path", func(t *testing.T) {
		err := validator.ValidateMountPath("/non/existent", "/workspace")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid source path")
	})

	t.Run("empty target path", func(t *testing.T) {
		err := validator.ValidateMountPath(tempDir, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "target path is required")
	})

	t.Run("relative target path", func(t *testing.T) {
		err := validator.ValidateMountPath(tempDir, "workspace")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "target path must be absolute")
	})

	t.Run("sensitive container paths blocked", func(t *testing.T) {
		sensitivePaths := []string{
			"/etc",
			"/usr",
			"/bin",
			"/root",
			"/var/run",
		}

		for _, path := range sensitivePaths {
			err := validator.ValidateMountPath(tempDir, path)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "cannot mount to sensitive container path")
		}
	})
}

func TestValidator_CanOptimizeMount(t *testing.T) {
	validator := NewValidator()
	tempDir := t.TempDir()

	canOptimize, syncMode := validator.CanOptimizeMount(tempDir)

	if runtime.GOOS == "windows" {
		// Windows에서는 기본적으로 최적화 불가
		assert.False(t, canOptimize)
		assert.Equal(t, SyncModeNative, syncMode)
	} else {
		// Unix 계열에서는 파일 시스템에 따라 결정
		assert.NotEqual(t, "", string(syncMode))
		// syncMode는 유효한 값이어야 함
		validModes := []SyncMode{SyncModeNative, SyncModeOptimized, SyncModeCached, SyncModeDelegated}
		assert.Contains(t, validModes, syncMode)
	}
}

func TestValidator_GetDiskUsage(t *testing.T) {
	validator := NewValidator()
	tempDir := t.TempDir()

	// 테스트 파일 생성
	testFile := filepath.Join(tempDir, "test.txt")
	content := "Hello, World!"
	err := os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(t, err)

	usage, err := validator.GetDiskUsage(tempDir)
	require.NoError(t, err)

	assert.NotNil(t, usage)
	assert.Greater(t, usage.Total, int64(0))
	assert.Greater(t, usage.Used, int64(0))

	if runtime.GOOS != "windows" {
		// Unix 계열에서는 Available도 계산됨
		assert.GreaterOrEqual(t, usage.Available, int64(0))
		assert.Equal(t, usage.Total, usage.Used+usage.Available)
	}
}

func TestValidator_checkSymlinks(t *testing.T) {
	validator := NewValidator()
	tempDir := t.TempDir()

	// 실제 디렉토리 생성
	realDir := filepath.Join(tempDir, "real")
	err := os.MkdirAll(realDir, 0755)
	require.NoError(t, err)

	// 심볼릭 링크 생성 (Unix 계열에서만)
	if runtime.GOOS != "windows" {
		linkPath := filepath.Join(tempDir, "link")
		err = os.Symlink(realDir, linkPath)
		require.NoError(t, err)

		t.Run("valid symlink", func(t *testing.T) {
			err := validator.checkSymlinks(linkPath)
			assert.NoError(t, err)
		})

		// 시스템 디렉토리로의 위험한 심볼릭 링크 테스트
		if runtime.GOOS == "linux" {
			dangerousLink := filepath.Join(tempDir, "dangerous")
			err = os.Symlink("/etc", dangerousLink)
			require.NoError(t, err)

			t.Run("dangerous symlink blocked", func(t *testing.T) {
				err := validator.checkSymlinks(dangerousLink)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "cannot mount system directory")
			})
		}
	}

	t.Run("non-symlink path", func(t *testing.T) {
		err := validator.checkSymlinks(realDir)
		assert.NoError(t, err)
	})
}

func TestValidator_calculateDirSize(t *testing.T) {
	validator := NewValidator()
	tempDir := t.TempDir()

	// 테스트 파일들 생성
	files := map[string]string{
		"file1.txt": "Hello",
		"file2.txt": "World",
		"subdir/file3.txt": "Test",
	}

	var expectedSize int64
	for filePath, content := range files {
		fullPath := filepath.Join(tempDir, filePath)
		err := os.MkdirAll(filepath.Dir(fullPath), 0755)
		require.NoError(t, err)
		err = os.WriteFile(fullPath, []byte(content), 0644)
		require.NoError(t, err)
		expectedSize += int64(len(content))
	}

	size, err := validator.calculateDirSize(tempDir)
	require.NoError(t, err)
	assert.Equal(t, expectedSize, size)
}

func TestSyncMode_Constants(t *testing.T) {
	// SyncMode 상수들이 올바르게 정의되었는지 확인
	assert.Equal(t, "native", string(SyncModeNative))
	assert.Equal(t, "optimized", string(SyncModeOptimized))
	assert.Equal(t, "cached", string(SyncModeCached))
	assert.Equal(t, "delegated", string(SyncModeDelegated))
}