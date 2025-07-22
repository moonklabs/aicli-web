// +build integration

package claude_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/aicli/aicli-web/internal/claude"
)

// 통합 테스트는 실제 프로세스를 실행하므로 더 긴 시간이 걸릴 수 있습니다.
// go test -tags=integration ./internal/claude/...

func TestProcessManagerIntegration_ClaudeCLISimulation(t *testing.T) {
	logger := createTestLogger()
	pm := claude.NewProcessManager(logger)
	
	// Claude CLI를 시뮬레이션하는 스크립트 생성
	scriptPath := createClaudeSimScript(t)
	defer os.Remove(scriptPath)
	
	ctx := context.Background()
	config := &claude.ProcessConfig{
		Command: getShellCommand(),
		Args:    []string{scriptPath},
	}
	
	// 프로세스 시작
	err := pm.Start(ctx, config)
	require.NoError(t, err)
	
	// 프로세스 상태 확인
	assert.Equal(t, claude.StatusRunning, pm.GetStatus())
	assert.Greater(t, pm.GetPID(), 0)
	assert.True(t, pm.IsRunning())
	
	// 5초 동안 실행
	time.Sleep(5 * time.Second)
	
	// 정상 종료
	err = pm.Stop(10 * time.Second)
	assert.NoError(t, err)
	
	// 최종 상태 확인
	assert.Equal(t, claude.StatusStopped, pm.GetStatus())
	assert.False(t, pm.IsRunning())
}

func TestProcessManagerIntegration_MultipleProcesses(t *testing.T) {
	logger := createTestLogger()
	
	// 여러 프로세스 동시 실행
	processes := make([]claude.ProcessManager, 3)
	for i := range processes {
		processes[i] = claude.NewProcessManager(logger)
	}
	
	ctx := context.Background()
	
	// 모든 프로세스 시작
	for i, pm := range processes {
		config := &claude.ProcessConfig{
			Command: getEchoCommand(),
			Args:    []string{fmt.Sprintf("Process %d", i+1)},
		}
		
		err := pm.Start(ctx, config)
		require.NoError(t, err)
	}
	
	// 모든 프로세스가 실행 중인지 확인
	for _, pm := range processes {
		assert.True(t, pm.IsRunning())
	}
	
	// 모든 프로세스 종료 대기
	for _, pm := range processes {
		err := pm.Wait()
		assert.NoError(t, err)
	}
	
	// 모든 프로세스가 종료되었는지 확인
	for _, pm := range processes {
		assert.Equal(t, claude.StatusStopped, pm.GetStatus())
	}
}

// 입출력 테스트는 태스크 요구사항에 없으므로 제거됨

func TestProcessManagerIntegration_ErrorHandling(t *testing.T) {
	logger := createTestLogger()
	pm := claude.NewProcessManager(logger)
	
	ctx := context.Background()
	
	t.Run("command not found", func(t *testing.T) {
		config := &claude.ProcessConfig{
			Command: "nonexistent_command_12345",
			Args:    []string{"--help"},
		}
		
		err := pm.Start(ctx, config)
		assert.Error(t, err)
		
		processErr, ok := err.(*claude.ProcessError)
		assert.True(t, ok)
		assert.Equal(t, claude.ErrTypeStartFailed, processErr.Type)
		assert.Equal(t, claude.StatusError, pm.GetStatus())
	})
	
	t.Run("exit with error code", func(t *testing.T) {
		pm2 := claude.NewProcessManager(logger)
		
		config := &claude.ProcessConfig{
			Command: getShellCommand(),
			Args:    getExitErrorArgs(),
		}
		
		err := pm2.Start(ctx, config)
		require.NoError(t, err)
		
		// 프로세스 완료 대기
		err = pm2.Wait()
		assert.Error(t, err)
		
		assert.Equal(t, claude.StatusError, pm2.GetStatus())
	})
}

func TestProcessManagerIntegration_Timeout(t *testing.T) {
	logger := createTestLogger()
	pm := claude.NewProcessManager(logger)
	
	ctx := context.Background()
	config := &claude.ProcessConfig{
		Command: getSleepCommand(),
		Args:    []string{"10"},
	}
	
	// 프로세스 시작
	err := pm.Start(ctx, config)
	require.NoError(t, err)
	
	// 정상 종료
	err = pm.Stop(3 * time.Second)
	assert.NoError(t, err)
	
	// 프로세스가 종료되었는지 확인
	assert.Equal(t, claude.StatusStopped, pm.GetStatus())
}

func TestProcessManagerIntegration_HealthCheck(t *testing.T) {
	logger := createTestLogger()
	pm := claude.NewProcessManager(logger)
	
	ctx := context.Background()
	config := &claude.ProcessConfig{
		Command: getSleepCommand(),
		Args:    []string{"5"},
	}
	
	// 프로세스 시작
	err := pm.Start(ctx, config)
	require.NoError(t, err)
	
	// 헬스체크 수행
	for i := 0; i < 3; i++ {
		err = pm.HealthCheck()
		assert.NoError(t, err)
		time.Sleep(1 * time.Second)
	}
	
	// 프로세스 종료
	err = pm.Stop(5 * time.Second)
	assert.NoError(t, err)
	
	// 종료 후 헬스체크는 실패해야 함
	err = pm.HealthCheck()
	assert.Error(t, err)
}

// 헬퍼 함수들

func createTestLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
	return logger
}

func createClaudeSimScript(t *testing.T) string {
	script := `#!/bin/bash
echo "Claude CLI Simulation Started"
echo "{"
echo '  "event": "session_started",'
echo '  "timestamp": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'"'
echo "}"

for i in {1..10}; do
  sleep 1
  echo "{"
  echo '  "event": "progress",'
  echo '  "percentage": '$((i * 10))','
  echo '  "message": "Processing... ('$i'/10)"'
  echo "}"
done

echo "{"
echo '  "event": "session_completed",'
echo '  "timestamp": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'"'
echo "}"
`
	
	tmpfile, err := os.CreateTemp("", "claude_sim_*.sh")
	require.NoError(t, err)
	
	_, err = tmpfile.WriteString(script)
	require.NoError(t, err)
	
	err = tmpfile.Close()
	require.NoError(t, err)
	
	// 실행 권한 부여
	err = os.Chmod(tmpfile.Name(), 0755)
	require.NoError(t, err)
	
	return tmpfile.Name()
}

func getShellCommand() string {
	if runtime.GOOS == "windows" {
		return "cmd"
	}
	return "bash"
}

func getEchoCommand() string {
	if runtime.GOOS == "windows" {
		return "cmd"
	}
	return "echo"
}

func getCatCommand() string {
	if runtime.GOOS == "windows" {
		return "type"
	}
	return "cat"
}

func getSleepCommand() string {
	if runtime.GOOS == "windows" {
		return "timeout"
	}
	return "sleep"
}

func getExitErrorArgs() []string {
	if runtime.GOOS == "windows" {
		return []string{"/c", "exit 1"}
	}
	return []string{"-c", "exit 1"}
}