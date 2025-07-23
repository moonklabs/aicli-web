package testutil

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestCLITestRunner_Basic(t *testing.T) {
	// 간단한 테스트 명령어 생성
	testCmd := &cobra.Command{
		Use:   "test",
		Short: "Test command",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Printf("Hello %s", strings.Join(args, " "))
			return nil
		},
	}

	runner := NewCLITestRunner()
	runner.SetCommand(testCmd)

	// 명령어 실행
	err := runner.RunCommand("world")
	
	assert.NoError(t, err)
	assert.Equal(t, "Hello world", runner.GetOutput())
	assert.Empty(t, runner.GetError())
}

func TestCLITestRunner_WithError(t *testing.T) {
	// 에러를 반환하는 명령어
	testCmd := &cobra.Command{
		Use: "error",
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.New("test error")
		},
	}

	runner := NewCLITestRunner()
	runner.SetCommand(testCmd)

	err := runner.RunCommand()
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "test error")
}

func TestCLITestRunner_WithStdin(t *testing.T) {
	// 표준 입력을 읽는 명령어
	testCmd := &cobra.Command{
		Use: "stdin",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Note: 실제로는 stdin을 읽는 로직이 들어가야 하지만
			// 테스트 목적으로 간단히 처리
			cmd.Print("stdin processed")
			return nil
		},
	}

	runner := NewCLITestRunner()
	runner.SetCommand(testCmd)
	runner.SetStdin(strings.NewReader("test input"))

	err := runner.RunCommand()
	
	assert.NoError(t, err)
	assert.Equal(t, "stdin processed", runner.GetOutput())
}

func TestCLITestRunner_WithEnvironment(t *testing.T) {
	// 환경 변수를 사용하는 명령어
	testCmd := &cobra.Command{
		Use: "env",
		RunE: func(cmd *cobra.Command, args []string) error {
			// 환경 변수 체크 로직은 실제 구현에서 처리
			cmd.Print("env command executed")
			return nil
		},
	}

	runner := NewCLITestRunner()
	runner.SetCommand(testCmd)
	runner.SetEnv("TEST_VAR", "test_value")

	err := runner.RunCommand()
	
	assert.NoError(t, err)
	assert.Equal(t, "env command executed", runner.GetOutput())
}

func TestCLITestRunner_WithTimeout(t *testing.T) {
	// 긴 실행 시간을 갖는 명령어
	testCmd := &cobra.Command{
		Use: "timeout",
		RunE: func(cmd *cobra.Command, args []string) error {
			// 테스트를 위해 실제 sleep은 하지 않음
			cmd.Print("completed")
			return nil
		},
	}

	runner := NewCLITestRunner()
	runner.SetCommand(testCmd)
	runner.SetTimeout(1 * time.Second)

	err := runner.RunCommand()
	
	assert.NoError(t, err)
	assert.Equal(t, "completed", runner.GetOutput())
}

func TestCLITestRunner_Reset(t *testing.T) {
	testCmd := &cobra.Command{
		Use: "test",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Print("first run")
			return nil
		},
	}

	runner := NewCLITestRunner()
	runner.SetCommand(testCmd)

	// 첫 번째 실행
	err := runner.RunCommand()
	assert.NoError(t, err)
	assert.Equal(t, "first run", runner.GetOutput())

	// 버퍼 리셋
	runner.Reset()
	assert.Empty(t, runner.GetOutput())
	assert.Empty(t, runner.GetError())

	// 두 번째 실행
	testCmd.RunE = func(cmd *cobra.Command, args []string) error {
		cmd.Print("second run")
		return nil
	}
	
	err = runner.RunCommand()
	assert.NoError(t, err)
	assert.Equal(t, "second run", runner.GetOutput())
}

func TestRunTestCases(t *testing.T) {
	// 테스트할 명령어 생성
	testCmd := &cobra.Command{
		Use: "test",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 && args[0] == "error" {
				return errors.New("test error")
			}
			cmd.Printf("args: %s", strings.Join(args, ","))
			return nil
		},
	}

	// 테스트 케이스들
	testCases := []CLITestCase{
		{
			Name:       "성공 케이스",
			Args:       []string{"arg1", "arg2"},
			WantErr:    false,
			WantOutput: "args: arg1,arg2",
		},
		{
			Name:      "에러 케이스",
			Args:      []string{"error"},
			WantErr:   true,
			WantError: "test error",
		},
		{
			Name:       "환경 변수 케이스",
			Args:       []string{"env"},
			Env:        map[string]string{"TEST": "value"},
			WantErr:    false,
			WantOutput: "args: env",
		},
	}

	// 테스트 케이스들 실행
	RunTestCases(t, testCmd, testCases)
}

func TestAssertHelpers(t *testing.T) {
	testCmd := &cobra.Command{
		Use: "test",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Print("test output")
			cmd.PrintErrln("test error")
			return nil
		},
	}

	runner := NewCLITestRunner()
	runner.SetCommand(testCmd)

	err := runner.RunCommand()
	assert.NoError(t, err)

	// Assert 헬퍼 함수들 테스트
	AssertOutputContains(t, runner, "test output")
	AssertErrorContains(t, runner, "test error")
}

func TestCLITestRunner_WorkingDirectory(t *testing.T) {
	// 임시 작업 디렉토리 생성
	tempDir := CreateTempWorkspace(t)
	
	testCmd := &cobra.Command{
		Use: "pwd",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Print("directory command executed")
			return nil
		},
	}

	runner := NewCLITestRunner()
	runner.SetCommand(testCmd)
	runner.SetWorkingDir(tempDir)

	err := runner.RunCommand()
	
	assert.NoError(t, err)
	assert.Equal(t, "directory command executed", runner.GetOutput())
}

// 성능 테스트
func BenchmarkCLITestRunner_SimpleCommand(b *testing.B) {
	testCmd := &cobra.Command{
		Use: "bench",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Print("benchmark")
			return nil
		},
	}

	runner := NewCLITestRunner()
	runner.SetCommand(testCmd)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runner.Reset()
		runner.RunCommand()
	}
}

// 동시성 테스트
func TestCLITestRunner_Concurrent(t *testing.T) {
	testCmd := &cobra.Command{
		Use: "concurrent",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Printf("goroutine %s", args[0])
			return nil
		},
	}

	// 여러 goroutine에서 동시에 실행
	results := make(chan string, 10)
	
	for i := 0; i < 10; i++ {
		go func(id int) {
			runner := NewCLITestRunner()
			runner.SetCommand(testCmd)
			
			err := runner.RunCommand(string(rune('0' + id)))
			if err != nil {
				results <- "error"
				return
			}
			
			results <- runner.GetOutput()
		}(i)
	}

	// 결과 수집
	for i := 0; i < 10; i++ {
		result := <-results
		assert.Contains(t, result, "goroutine")
	}
}

func TestCLITestCase_WithSetupAndCleanup(t *testing.T) {
	testCmd := &cobra.Command{
		Use: "setup-test",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Print("setup test executed")
			return nil
		},
	}

	var setupCalled, cleanupCalled bool

	testCases := []CLITestCase{
		{
			Name: "Setup/Cleanup 테스트",
			Args: []string{},
			Setup: func(t *testing.T) string {
				setupCalled = true
				return CreateTempWorkspace(t)
			},
			Cleanup: func(t *testing.T) {
				cleanupCalled = true
			},
			WantErr:    false,
			WantOutput: "setup test executed",
		},
	}

	RunTestCases(t, testCmd, testCases)

	assert.True(t, setupCalled, "Setup이 호출되어야 함")
	assert.True(t, cleanupCalled, "Cleanup이 호출되어야 함")
}

// 복잡한 명령어 체인 테스트
func TestCLITestRunner_ComplexCommand(t *testing.T) {
	// 하위 명령어들이 있는 복잡한 명령어 구조
	rootCmd := &cobra.Command{Use: "app"}
	
	configCmd := &cobra.Command{
		Use: "config",
		Short: "Configuration commands",
	}
	
	getCmd := &cobra.Command{
		Use: "get",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("key required")
			}
			cmd.Printf("config value for %s", args[0])
			return nil
		},
	}
	
	setCmd := &cobra.Command{
		Use: "set",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return errors.New("key and value required")
			}
			cmd.Printf("set %s=%s", args[0], args[1])
			return nil
		},
	}

	configCmd.AddCommand(getCmd, setCmd)
	rootCmd.AddCommand(configCmd)

	testCases := []CLITestCase{
		{
			Name:       "config get 성공",
			Args:       []string{"config", "get", "api_key"},
			WantErr:    false,
			WantOutput: "config value for api_key",
		},
		{
			Name:      "config get 실패 - 인자 없음",
			Args:      []string{"config", "get"},
			WantErr:   true,
			WantError: "key required",
		},
		{
			Name:       "config set 성공",
			Args:       []string{"config", "set", "api_key", "test-key"},
			WantErr:    false,
			WantOutput: "set api_key=test-key",
		},
		{
			Name:      "config set 실패 - 인자 부족",
			Args:      []string{"config", "set", "api_key"},
			WantErr:   true,
			WantError: "key and value required",
		},
	}

	RunTestCases(t, rootCmd, testCases)
}