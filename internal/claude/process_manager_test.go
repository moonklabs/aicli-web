package claude

import (
	"context"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessStatus_String(t *testing.T) {
	tests := []struct {
		status   ProcessStatus
		expected string
	}{
		{StatusStopped, "stopped"},
		{StatusStarting, "starting"},
		{StatusRunning, "running"},
		{StatusStopping, "stopping"},
		{StatusError, "error"},
		{StatusUnknown, "unknown"},
		{ProcessStatus(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status.String())
		})
	}
}

func TestNewProcessManager(t *testing.T) {
	t.Run("with logger", func(t *testing.T) {
		logger := logrus.New()
		pm := NewProcessManager(logger)
		assert.NotNil(t, pm)
		assert.Equal(t, StatusStopped, pm.GetStatus())
		assert.Equal(t, 0, pm.GetPID())
	})

	t.Run("without logger", func(t *testing.T) {
		pm := NewProcessManager(nil)
		assert.NotNil(t, pm)
		assert.Equal(t, StatusStopped, pm.GetStatus())
	})
}

func TestProcessManager_Start(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	t.Run("simple command", func(t *testing.T) {
		pm := NewProcessManager(logger)
		ctx := context.Background()

		// 플랫폼별 명령어 선택
		var cmd string
		var args []string
		if runtime.GOOS == "windows" {
			cmd = "cmd"
			args = []string{"/c", "echo", "hello"}
		} else {
			cmd = "echo"
			args = []string{"hello"}
		}

		config := &ProcessConfig{
			Command: cmd,
			Args:    args,
		}

		err := pm.Start(ctx, config)
		require.NoError(t, err)
		assert.True(t, pm.IsRunning())
		assert.Greater(t, pm.GetPID(), 0)

		// 프로세스가 종료될 때까지 대기
		err = pm.Wait()
		assert.NoError(t, err)
		assert.Equal(t, StatusStopped, pm.GetStatus())
	})

	t.Run("with working directory", func(t *testing.T) {
		pm := NewProcessManager(logger)
		ctx := context.Background()

		tempDir, err := os.MkdirTemp("", "process_test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		var cmd string
		var args []string
		if runtime.GOOS == "windows" {
			cmd = "cmd"
			args = []string{"/c", "cd"}
		} else {
			cmd = "pwd"
			args = []string{}
		}

		config := &ProcessConfig{
			Command:    cmd,
			Args:       args,
			WorkingDir: tempDir,
		}

		err = pm.Start(ctx, config)
		require.NoError(t, err)

		err = pm.Wait()
		assert.NoError(t, err)
	})

	t.Run("with environment variables", func(t *testing.T) {
		pm := NewProcessManager(logger)
		ctx := context.Background()

		var cmd string
		var args []string
		if runtime.GOOS == "windows" {
			cmd = "cmd"
			args = []string{"/c", "echo", "%TEST_VAR%"}
		} else {
			cmd = "sh"
			args = []string{"-c", "echo $TEST_VAR"}
		}

		config := &ProcessConfig{
			Command: cmd,
			Args:    args,
			Environment: map[string]string{
				"TEST_VAR": "test_value",
			},
		}

		err := pm.Start(ctx, config)
		require.NoError(t, err)

		err = pm.Wait()
		assert.NoError(t, err)
	})

	t.Run("with OAuth token", func(t *testing.T) {
		pm := NewProcessManager(logger)
		ctx := context.Background()

		var cmd string
		var args []string
		if runtime.GOOS == "windows" {
			cmd = "cmd"
			args = []string{"/c", "echo", "%CLAUDE_CODE_OAUTH_TOKEN%"}
		} else {
			cmd = "sh"
			args = []string{"-c", "echo $CLAUDE_CODE_OAUTH_TOKEN"}
		}

		config := &ProcessConfig{
			Command:    cmd,
			Args:       args,
			OAuthToken: "test-oauth-token",
		}

		err := pm.Start(ctx, config)
		require.NoError(t, err)

		err = pm.Wait()
		assert.NoError(t, err)
	})

	t.Run("with API key", func(t *testing.T) {
		pm := NewProcessManager(logger)
		ctx := context.Background()

		var cmd string
		var args []string
		if runtime.GOOS == "windows" {
			cmd = "cmd"
			args = []string{"/c", "echo", "%CLAUDE_API_KEY%"}
		} else {
			cmd = "sh"
			args = []string{"-c", "echo $CLAUDE_API_KEY"}
		}

		config := &ProcessConfig{
			Command: cmd,
			Args:    args,
			APIKey:  "test-api-key",
		}

		err := pm.Start(ctx, config)
		require.NoError(t, err)

		err = pm.Wait()
		assert.NoError(t, err)
	})

	t.Run("with resource limits", func(t *testing.T) {
		pm := NewProcessManager(logger)
		ctx := context.Background()

		var cmd string
		var args []string
		if runtime.GOOS == "windows" {
			cmd = "cmd"
			args = []string{"/c", "echo", "hello"}
		} else {
			cmd = "echo"
			args = []string{"hello"}
		}

		config := &ProcessConfig{
			Command: cmd,
			Args:    args,
			ResourceLimits: &ResourceLimits{
				MaxCPU:    1.0,
				MaxMemory: 1024 * 1024 * 512, // 512MB
				MaxDiskIO: 1024 * 1024,        // 1MB/s
				Timeout:   5 * time.Second,
			},
		}

		err := pm.Start(ctx, config)
		require.NoError(t, err)

		err = pm.Wait()
		assert.NoError(t, err)
	})

	t.Run("with health check", func(t *testing.T) {
		pm := NewProcessManager(logger)
		ctx := context.Background()

		var cmd string
		var args []string
		if runtime.GOOS == "windows" {
			cmd = "cmd"
			args = []string{"/c", "timeout", "/t", "2"}
		} else {
			cmd = "sleep"
			args = []string{"2"}
		}

		config := &ProcessConfig{
			Command:             cmd,
			Args:                args,
			HealthCheckInterval: 500 * time.Millisecond,
		}

		err := pm.Start(ctx, config)
		require.NoError(t, err)

		// 헬스체크가 실행될 시간을 줌
		time.Sleep(1 * time.Second)

		// 프로세스가 여전히 실행 중인지 확인
		assert.True(t, pm.IsRunning())

		// 정리
		_ = pm.Kill()
	})

	t.Run("invalid config", func(t *testing.T) {
		pm := NewProcessManager(logger)
		ctx := context.Background()

		// nil config
		err := pm.Start(ctx, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "프로세스 설정이 nil입니다")

		// empty command
		config := &ProcessConfig{}
		err = pm.Start(ctx, config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "실행할 명령어가 지정되지 않았습니다")
	})

	t.Run("already running", func(t *testing.T) {
		pm := NewProcessManager(logger)
		ctx := context.Background()

		var cmd string
		var args []string
		if runtime.GOOS == "windows" {
			cmd = "cmd"
			args = []string{"/c", "timeout", "/t", "2"}
		} else {
			cmd = "sleep"
			args = []string{"2"}
		}

		config := &ProcessConfig{
			Command: cmd,
			Args:    args,
		}

		err := pm.Start(ctx, config)
		require.NoError(t, err)

		// 이미 실행 중인 상태에서 다시 시작 시도
		err = pm.Start(ctx, config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "프로세스가 이미 실행 중이거나 시작 중입니다")

		// 정리
		_ = pm.Kill()
	})
}

func TestProcessManager_Stop(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	t.Run("graceful stop", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Windows에서는 SIGTERM을 지원하지 않습니다")
		}

		pm := NewProcessManager(logger)
		ctx := context.Background()

		// 시그널을 받을 수 있는 프로세스 실행
		config := &ProcessConfig{
			Command: "sh",
			Args:    []string{"-c", "trap 'exit 0' TERM; sleep 10"},
		}

		err := pm.Start(ctx, config)
		require.NoError(t, err)

		// 프로세스가 시작될 때까지 대기
		time.Sleep(100 * time.Millisecond)

		// 정상 종료
		err = pm.Stop(2 * time.Second)
		assert.NoError(t, err)
		assert.Equal(t, StatusStopped, pm.GetStatus())
	})

	t.Run("stop with timeout", func(t *testing.T) {
		pm := NewProcessManager(logger)
		ctx := context.Background()

		var cmd string
		var args []string
		if runtime.GOOS == "windows" {
			cmd = "cmd"
			args = []string{"/c", "timeout", "/t", "10"}
		} else {
			cmd = "sh"
			args = []string{"-c", "trap '' TERM; sleep 10"}
		}

		config := &ProcessConfig{
			Command: cmd,
			Args:    args,
		}

		err := pm.Start(ctx, config)
		require.NoError(t, err)

		// 짧은 타임아웃으로 중지 시도
		err = pm.Stop(100 * time.Millisecond)
		assert.NoError(t, err) // Kill이 호출되어 성공해야 함
		assert.Equal(t, StatusStopped, pm.GetStatus())
	})

	t.Run("stop not running", func(t *testing.T) {
		pm := NewProcessManager(logger)
		err := pm.Stop(1 * time.Second)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "프로세스가 실행 중이 아닙니다")
	})
}

func TestProcessManager_Kill(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	t.Run("kill running process", func(t *testing.T) {
		pm := NewProcessManager(logger)
		ctx := context.Background()

		var cmd string
		var args []string
		if runtime.GOOS == "windows" {
			cmd = "cmd"
			args = []string{"/c", "timeout", "/t", "10"}
		} else {
			cmd = "sleep"
			args = []string{"10"}
		}

		config := &ProcessConfig{
			Command: cmd,
			Args:    args,
		}

		err := pm.Start(ctx, config)
		require.NoError(t, err)

		// 프로세스 강제 종료
		err = pm.Kill()
		assert.NoError(t, err)
		assert.Equal(t, StatusStopped, pm.GetStatus())
	})

	t.Run("kill already stopped", func(t *testing.T) {
		pm := NewProcessManager(logger)
		err := pm.Kill()
		assert.NoError(t, err) // 이미 중지된 상태에서는 에러 없음
	})
}

func TestProcessManager_HealthCheck(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	t.Run("healthy process", func(t *testing.T) {
		pm := NewProcessManager(logger)
		ctx := context.Background()

		var cmd string
		var args []string
		if runtime.GOOS == "windows" {
			cmd = "cmd"
			args = []string{"/c", "timeout", "/t", "2"}
		} else {
			cmd = "sleep"
			args = []string{"2"}
		}

		config := &ProcessConfig{
			Command: cmd,
			Args:    args,
		}

		err := pm.Start(ctx, config)
		require.NoError(t, err)

		// 헬스체크
		err = pm.HealthCheck()
		assert.NoError(t, err)

		// 정리
		_ = pm.Kill()
	})

	t.Run("not running", func(t *testing.T) {
		pm := NewProcessManager(logger)
		err := pm.HealthCheck()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "프로세스가 실행 중이 아닙니다")
	})
}

func TestProcessManager_ConcurrentOperations(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	pm := NewProcessManager(logger)
	ctx := context.Background()

	var cmd string
	var args []string
	if runtime.GOOS == "windows" {
		cmd = "cmd"
		args = []string{"/c", "timeout", "/t", "2"}
	} else {
		cmd = "sleep"
		args = []string{"2"}
	}

	config := &ProcessConfig{
		Command: cmd,
		Args:    args,
	}

	// 프로세스 시작
	err := pm.Start(ctx, config)
	require.NoError(t, err)

	// 동시에 여러 작업 수행
	done := make(chan bool, 4)

	// 상태 확인
	go func() {
		for i := 0; i < 10; i++ {
			_ = pm.GetStatus()
			_ = pm.IsRunning()
			time.Sleep(10 * time.Millisecond)
		}
		done <- true
	}()

	// PID 확인
	go func() {
		for i := 0; i < 10; i++ {
			_ = pm.GetPID()
			time.Sleep(10 * time.Millisecond)
		}
		done <- true
	}()

	// 헬스체크
	go func() {
		for i := 0; i < 10; i++ {
			_ = pm.HealthCheck()
			time.Sleep(10 * time.Millisecond)
		}
		done <- true
	}()

	// 모든 고루틴 완료 대기
	for i := 0; i < 3; i++ {
		<-done
	}

	// 정리
	_ = pm.Kill()
}

func TestProcessError(t *testing.T) {
	t.Run("with cause", func(t *testing.T) {
		cause := assert.AnError
		err := NewProcessError(
			ErrTypeStartFailed,
			"프로세스 시작 실패",
			cause,
			1234,
			StatusError,
		)

		assert.Contains(t, err.Error(), "START_FAILED")
		assert.Contains(t, err.Error(), "PID: 1234")
		assert.Contains(t, err.Error(), "상태: error")
		assert.Contains(t, err.Error(), "프로세스 시작 실패")
		assert.Equal(t, cause, err.Unwrap())
	})

	t.Run("without cause", func(t *testing.T) {
		err := NewProcessError(
			ErrTypeHealthCheckFailed,
			"헬스체크 실패",
			nil,
			5678,
			StatusRunning,
		)

		assert.Contains(t, err.Error(), "HEALTH_CHECK_FAILED")
		assert.Contains(t, err.Error(), "PID: 5678")
		assert.Contains(t, err.Error(), "상태: running")
		assert.Contains(t, err.Error(), "헬스체크 실패")
		assert.Nil(t, err.Unwrap())
	})
}

