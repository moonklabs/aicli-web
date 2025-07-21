// +build integration

package claude

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCLIIntegration은 CLI 명령어의 E2E 통합 테스트입니다.
func TestCLIIntegration(t *testing.T) {
	// aicli 바이너리가 빌드되어 있는지 확인
	if !isBinaryAvailable("aicli") {
		t.Skip("aicli binary not available, skipping CLI integration tests")
	}

	t.Run("Claude Run Command", func(t *testing.T) {
		testClaudeRunCommand(t)
	})

	t.Run("Claude Session Management", func(t *testing.T) {
		testClaudeSessionManagement(t)
	})

	t.Run("Claude Status Command", func(t *testing.T) {
		testClaudeStatusCommand(t)
	})

	t.Run("Output Formats", func(t *testing.T) {
		testOutputFormats(t)
	})
}

// testClaudeRunCommand는 claude run 명령어를 테스트합니다.
func testClaudeRunCommand(t *testing.T) {
	tempDir := t.TempDir()
	
	// 기본 텍스트 출력
	cmd := exec.Command("aicli", "claude", "run", "Hello, Claude!")
	cmd.Dir = tempDir
	cmd.Env = append(os.Environ(), "TEST_MODE=true")
	
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	err := cmd.Run()
	require.NoError(t, err, "stderr: %s", stderr.String())
	
	output := stdout.String()
	assert.Contains(t, output, "Claude CLI", "Claude CLI 실행 메시지가 포함되어야 함")
	assert.Contains(t, output, "완료", "완료 메시지가 포함되어야 함")
}

// testClaudeSessionManagement는 세션 관리 명령어를 테스트합니다.
func testClaudeSessionManagement(t *testing.T) {
	tempDir := t.TempDir()
	
	// 세션 목록 조회 (초기에는 비어있어야 함)
	cmd := exec.Command("aicli", "claude", "session", "list")
	cmd.Dir = tempDir
	cmd.Env = append(os.Environ(), "TEST_MODE=true")
	
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	
	err := cmd.Run()
	require.NoError(t, err)
	
	output := stdout.String()
	assert.Contains(t, output, "세션", "세션 관련 메시지가 포함되어야 함")
}

// testClaudeStatusCommand는 status 명령어를 테스트합니다.
func testClaudeStatusCommand(t *testing.T) {
	cmd := exec.Command("aicli", "claude", "status")
	cmd.Env = append(os.Environ(), "TEST_MODE=true")
	
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	
	err := cmd.Run()
	require.NoError(t, err)
	
	output := stdout.String()
	assert.Contains(t, output, "Claude CLI", "Claude CLI 상태 정보가 포함되어야 함")
}

// testOutputFormats는 다양한 출력 형식을 테스트합니다.
func testOutputFormats(t *testing.T) {
	tempDir := t.TempDir()
	
	formats := []string{"text", "json", "markdown"}
	
	for _, format := range formats {
		t.Run("Format_"+format, func(t *testing.T) {
			cmd := exec.Command("aicli", "claude", "run", "test message", "--format", format)
			cmd.Dir = tempDir
			cmd.Env = append(os.Environ(), "TEST_MODE=true")
			
			var stdout bytes.Buffer
			cmd.Stdout = &stdout
			
			err := cmd.Run()
			require.NoError(t, err, "Format %s should work", format)
			
			output := stdout.String()
			assert.NotEmpty(t, output, "Output should not be empty for format %s", format)
			
			// JSON 형식의 경우 JSON 구조 확인
			if format == "json" {
				assert.Contains(t, output, "{", "JSON output should contain braces")
			}
		})
	}
}

// TestCLIErrorHandling은 CLI 에러 처리를 테스트합니다.
func TestCLIErrorHandling(t *testing.T) {
	if !isBinaryAvailable("aicli") {
		t.Skip("aicli binary not available")
	}

	t.Run("Invalid Command", func(t *testing.T) {
		cmd := exec.Command("aicli", "claude", "invalid-command")
		cmd.Env = append(os.Environ(), "TEST_MODE=true")
		
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		
		err := cmd.Run()
		assert.Error(t, err, "Invalid command should return error")
		
		output := stderr.String()
		assert.Contains(t, output, "unknown command", "Error message should mention unknown command")
	})

	t.Run("Missing Required Args", func(t *testing.T) {
		cmd := exec.Command("aicli", "claude", "run")
		cmd.Env = append(os.Environ(), "TEST_MODE=true")
		
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		
		err := cmd.Run()
		assert.Error(t, err, "Missing args should return error")
	})
}

// TestCLIInteractiveMode는 인터랙티브 모드를 테스트합니다.
func TestCLIInteractiveMode(t *testing.T) {
	if !isBinaryAvailable("aicli") {
		t.Skip("aicli binary not available")
	}

	t.Run("Chat Mode Exit", func(t *testing.T) {
		cmd := exec.Command("aicli", "claude", "chat")
		cmd.Env = append(os.Environ(), "TEST_MODE=true")
		
		// /exit 명령으로 즉시 종료
		cmd.Stdin = strings.NewReader("/exit\n")
		
		var stdout bytes.Buffer
		cmd.Stdout = &stdout
		
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		
		done := make(chan error, 1)
		go func() {
			done <- cmd.Run()
		}()
		
		select {
		case err := <-done:
			// /exit으로 정상 종료되면 exit code 0이 될 수 있음
			if err != nil {
				// exit code가 있을 수 있지만 stderr에 panic이 없으면 OK
				assert.NotContains(t, stdout.String(), "panic", "Should not panic")
			}
		case <-ctx.Done():
			t.Fatal("Interactive mode did not exit within timeout")
		}
		
		output := stdout.String()
		assert.Contains(t, output, "인터랙티브", "Should show interactive mode message")
	})
}

// TestCLIConfigIntegration은 설정 통합을 테스트합니다.
func TestCLIConfigIntegration(t *testing.T) {
	if !isBinaryAvailable("aicli") {
		t.Skip("aicli binary not available")
	}

	tempDir := t.TempDir()
	
	t.Run("Config File Override", func(t *testing.T) {
		// 임시 설정 파일 생성
		configContent := `
claude:
  model: "claude-3-sonnet"
  max_turns: 20
`
		configPath := tempDir + "/aicli.yaml"
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)
		
		cmd := exec.Command("aicli", "claude", "run", "test", "--config", configPath)
		cmd.Dir = tempDir
		cmd.Env = append(os.Environ(), "TEST_MODE=true")
		
		var stdout bytes.Buffer
		cmd.Stdout = &stdout
		
		err = cmd.Run()
		require.NoError(t, err)
		
		// 설정 파일이 올바르게 로드되었는지는 내부 로직에서 확인해야 함
		output := stdout.String()
		assert.NotEmpty(t, output)
	})
}

// isBinaryAvailable은 바이너리가 사용 가능한지 확인합니다.
func isBinaryAvailable(name string) bool {
	cmd := exec.Command("which", name)
	err := cmd.Run()
	if err != nil {
		// PATH에서 찾을 수 없으면 현재 디렉토리에서 확인
		if _, err := os.Stat("./" + name); err == nil {
			return true
		}
		// 빌드 출력 디렉토리에서 확인
		if _, err := os.Stat("../../bin/" + name); err == nil {
			return true
		}
		return false
	}
	return true
}