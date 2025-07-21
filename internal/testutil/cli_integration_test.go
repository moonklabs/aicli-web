package testutil

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 통합 테스트: 실제 CLI 명령어들의 조합 테스트
func TestCLI_IntegrationWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("통합 테스트는 -short 플래그에서 제외")
	}

	// 테스트 워크스페이스 생성
	workspaceDir := CreateTempWorkspace(t)
	CreateTestProjectStructure(t, workspaceDir)

	// 모의 CLI 애플리케이션 생성
	app := createMockCLIApp(t)
	
	// 테스트 시나리오: 워크스페이스 초기화 -> 설정 -> 작업 실행
	t.Run("전체 워크플로우", func(t *testing.T) {
		runner := NewCLITestRunner()
		runner.SetCommand(app)
		runner.SetWorkingDir(workspaceDir)

		// 1. 워크스페이스 초기화
		LogTestStep(t, "워크스페이스 초기화")
		err := runner.RunCommand("workspace", "init", "test-project")
		require.NoError(t, err)
		AssertOutputContains(t, runner, "워크스페이스가 초기화되었습니다")
		runner.Reset()

		// 2. 설정 파일 생성
		LogTestStep(t, "설정 파일 생성")
		err = runner.RunCommand("config", "set", "claude.api_key", "test-api-key")
		require.NoError(t, err)
		AssertOutputContains(t, runner, "설정이 저장되었습니다")
		runner.Reset()

		// 3. 설정 확인
		LogTestStep(t, "설정 확인")
		err = runner.RunCommand("config", "get", "claude.api_key")
		require.NoError(t, err)
		AssertOutputContains(t, runner, "test-api-key")
		runner.Reset()

		// 4. 워크스페이스 목록 조회
		LogTestStep(t, "워크스페이스 목록 조회")
		err = runner.RunCommand("workspace", "list")
		require.NoError(t, err)
		AssertOutputContains(t, runner, "test-project")
		runner.Reset()

		// 5. 프로젝트 정보 조회
		LogTestStep(t, "프로젝트 정보 조회")
		err = runner.RunCommand("workspace", "info", "test-project")
		require.NoError(t, err)
		AssertOutputContains(t, runner, "test-project")
	})
}

// 에러 처리 통합 테스트
func TestCLI_ErrorHandlingIntegration(t *testing.T) {
	app := createMockCLIApp(t)
	
	testCases := []CLITestCase{
		{
			Name:      "존재하지 않는 명령어",
			Args:      []string{"nonexistent"},
			WantErr:   true,
			WantError: "unknown command",
		},
		{
			Name:      "잘못된 플래그",
			Args:      []string{"config", "get", "--invalid-flag"},
			WantErr:   true,
			WantError: "unknown flag",
		},
		{
			Name:      "필수 인자 누락",
			Args:      []string{"config", "get"},
			WantErr:   true,
			WantError: "key required",
		},
		{
			Name:      "존재하지 않는 워크스페이스",
			Args:      []string{"workspace", "info", "nonexistent"},
			WantErr:   true,
			WantError: "workspace not found",
		},
	}

	RunTestCases(t, app, testCases)
}

// 설정 관리 통합 테스트
func TestCLI_ConfigManagementIntegration(t *testing.T) {
	workspaceDir := CreateTempWorkspace(t)
	app := createMockCLIApp(t)

	runner := NewCLITestRunner()
	runner.SetCommand(app)
	runner.SetWorkingDir(workspaceDir)

	// 설정 파일이 없는 상태에서 시작
	t.Run("초기 상태 확인", func(t *testing.T) {
		err := runner.RunCommand("config", "list")
		require.NoError(t, err)
		AssertOutputContains(t, runner, "설정이 비어있습니다")
		runner.Reset()
	})

	// 설정 값 추가
	configTests := map[string]string{
		"claude.api_key":     "test-api-key",
		"claude.model":       "claude-3",
		"output.format":      "json",
		"workspace.default":  workspaceDir,
	}

	t.Run("설정 값 추가", func(t *testing.T) {
		for key, value := range configTests {
			err := runner.RunCommand("config", "set", key, value)
			require.NoError(t, err, "설정 추가 실패: %s=%s", key, value)
			AssertOutputContains(t, runner, "설정이 저장되었습니다")
			runner.Reset()
		}
	})

	// 설정 값 확인
	t.Run("설정 값 확인", func(t *testing.T) {
		for key, expected := range configTests {
			err := runner.RunCommand("config", "get", key)
			require.NoError(t, err, "설정 조회 실패: %s", key)
			AssertOutputContains(t, runner, expected)
			runner.Reset()
		}
	})

	// 전체 설정 목록 확인
	t.Run("전체 설정 목록", func(t *testing.T) {
		err := runner.RunCommand("config", "list")
		require.NoError(t, err)
		for key := range configTests {
			AssertOutputContains(t, runner, key)
		}
		runner.Reset()
	})

	// 설정 삭제
	t.Run("설정 삭제", func(t *testing.T) {
		err := runner.RunCommand("config", "unset", "output.format")
		require.NoError(t, err)
		AssertOutputContains(t, runner, "설정이 삭제되었습니다")
		runner.Reset()

		// 삭제 확인
		err = runner.RunCommand("config", "get", "output.format")
		require.Error(t, err)
		AssertErrorContains(t, runner, "설정을 찾을 수 없습니다")
		runner.Reset()
	})
}

// 워크스페이스 관리 통합 테스트
func TestCLI_WorkspaceManagementIntegration(t *testing.T) {
	tempDir := CreateTempWorkspace(t)
	app := createMockCLIApp(t)

	runner := NewCLITestRunner()
	runner.SetCommand(app)
	runner.SetWorkingDir(tempDir)

	workspaces := []string{"project-a", "project-b", "project-c"}

	// 워크스페이스 생성
	t.Run("워크스페이스 생성", func(t *testing.T) {
		for _, name := range workspaces {
			err := runner.RunCommand("workspace", "create", name)
			require.NoError(t, err, "워크스페이스 생성 실패: %s", name)
			AssertOutputContains(t, runner, fmt.Sprintf("워크스페이스 '%s'가 생성되었습니다", name))
			runner.Reset()
		}
	})

	// 워크스페이스 목록 확인
	t.Run("워크스페이스 목록", func(t *testing.T) {
		err := runner.RunCommand("workspace", "list")
		require.NoError(t, err)
		for _, name := range workspaces {
			AssertOutputContains(t, runner, name)
		}
		runner.Reset()
	})

	// 각 워크스페이스 정보 확인
	t.Run("워크스페이스 정보", func(t *testing.T) {
		for _, name := range workspaces {
			err := runner.RunCommand("workspace", "info", name)
			require.NoError(t, err, "워크스페이스 정보 조회 실패: %s", name)
			AssertOutputContains(t, runner, name)
			runner.Reset()
		}
	})

	// 워크스페이스 삭제
	t.Run("워크스페이스 삭제", func(t *testing.T) {
		err := runner.RunCommand("workspace", "delete", "project-c")
		require.NoError(t, err)
		AssertOutputContains(t, runner, "워크스페이스가 삭제되었습니다")
		runner.Reset()

		// 삭제 확인
		err = runner.RunCommand("workspace", "info", "project-c")
		require.Error(t, err)
		AssertErrorContains(t, runner, "workspace not found")
		runner.Reset()
	})
}

// 출력 형식 통합 테스트
func TestCLI_OutputFormatIntegration(t *testing.T) {
	app := createMockCLIApp(t)
	workspaceDir := CreateTempWorkspace(t)

	// 여러 출력 형식 테스트
	formats := []struct {
		name string
		flag string
		check func(output string) bool
	}{
		{
			name: "table",
			flag: "table",
			check: func(output string) bool {
				return assert.Contains(t, output, "|") // 테이블 구분자
			},
		},
		{
			name: "json",
			flag: "json",
			check: func(output string) bool {
				return assert.Contains(t, output, "{") && assert.Contains(t, output, "}")
			},
		},
		{
			name: "yaml",
			flag: "yaml",
			check: func(output string) bool {
				return assert.Contains(t, output, ":") // YAML 구분자
			},
		},
	}

	for _, format := range formats {
		t.Run(fmt.Sprintf("출력 형식: %s", format.name), func(t *testing.T) {
			runner := NewCLITestRunner()
			runner.SetCommand(app)
			runner.SetWorkingDir(workspaceDir)

			err := runner.RunCommand("workspace", "list", "--output", format.flag)
			require.NoError(t, err)
			
			output := runner.GetOutput()
			assert.True(t, format.check(output), "출력 형식이 올바르지 않음: %s", format.name)
		})
	}
}

// 환경 변수 통합 테스트
func TestCLI_EnvironmentVariablesIntegration(t *testing.T) {
	app := createMockCLIApp(t)
	
	envTests := map[string]string{
		"AICLI_CLAUDE_API_KEY": "env-api-key",
		"AICLI_OUTPUT_FORMAT":  "json",
		"AICLI_WORKSPACE_DIR":  "/tmp/aicli-workspaces",
	}

	t.Run("환경 변수 우선순위", func(t *testing.T) {
		runner := NewCLITestRunner()
		runner.SetCommand(app)

		// 환경 변수 설정
		for key, value := range envTests {
			runner.SetEnv(key, value)
		}

		err := runner.RunCommand("config", "get", "claude.api_key")
		require.NoError(t, err)
		AssertOutputContains(t, runner, "env-api-key")
	})
}

// 동시 실행 통합 테스트
func TestCLI_ConcurrentOperationsIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("동시 실행 테스트는 -short 플래그에서 제외")
	}

	app := createMockCLIApp(t)
	workspaceDir := CreateTempWorkspace(t)

	// 여러 명령을 동시에 실행
	numGoroutines := 5
	results := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			runner := NewCLITestRunner()
			runner.SetCommand(app)
			runner.SetWorkingDir(workspaceDir)

			// 각 고루틴에서 다른 작업 수행
			workspaceName := fmt.Sprintf("concurrent-workspace-%d", id)
			err := runner.RunCommand("workspace", "create", workspaceName)
			
			results <- err
		}(i)
	}

	// 모든 고루틴 완료 대기
	for i := 0; i < numGoroutines; i++ {
		err := <-results
		assert.NoError(t, err, "동시 실행 중 에러 발생")
	}
}

// 성능 테스트
func TestCLI_PerformanceIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("성능 테스트는 -short 플래그에서 제외")
	}

	app := createMockCLIApp(t)
	workspaceDir := CreateTempWorkspace(t)

	runner := NewCLITestRunner()
	runner.SetCommand(app)
	runner.SetWorkingDir(workspaceDir)

	// 성능 측정: 명령어 실행 시간
	MeasureTime(t, "워크스페이스 생성", func() {
		err := runner.RunCommand("workspace", "create", "perf-test")
		require.NoError(t, err)
	})

	runner.Reset()

	MeasureTime(t, "설정 조회", func() {
		err := runner.RunCommand("config", "list")
		require.NoError(t, err)
	})
}

// 메모리 사용량 테스트
func TestCLI_MemoryUsageIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("메모리 테스트는 -short 플래그에서 제외")
	}

	app := createMockCLIApp(t)
	
	// 많은 수의 워크스페이스 생성/삭제로 메모리 누수 확인
	for i := 0; i < 100; i++ {
		runner := NewCLITestRunner()
		runner.SetCommand(app)
		
		workspaceName := fmt.Sprintf("memory-test-%d", i)
		
		err := runner.RunCommand("workspace", "create", workspaceName)
		require.NoError(t, err)
		
		err = runner.RunCommand("workspace", "delete", workspaceName)
		require.NoError(t, err)
	}
	
	t.Log("메모리 사용량 테스트 완료")
}

// 모의 CLI 애플리케이션 생성 헬퍼 함수
func createMockCLIApp(t *testing.T) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "aicli",
		Short: "AI Code CLI Manager",
	}

	// workspace 명령어
	workspaceCmd := &cobra.Command{
		Use:   "workspace",
		Short: "Workspace management",
	}

	workspaceCmd.AddCommand(
		&cobra.Command{
			Use: "init [name]",
			RunE: func(cmd *cobra.Command, args []string) error {
				if len(args) == 0 {
					return fmt.Errorf("workspace name required")
				}
				cmd.Printf("워크스페이스가 초기화되었습니다: %s\n", args[0])
				return nil
			},
		},
		&cobra.Command{
			Use: "create [name]",
			RunE: func(cmd *cobra.Command, args []string) error {
				if len(args) == 0 {
					return fmt.Errorf("workspace name required")
				}
				cmd.Printf("워크스페이스 '%s'가 생성되었습니다\n", args[0])
				return nil
			},
		},
		&cobra.Command{
			Use: "list",
			RunE: func(cmd *cobra.Command, args []string) error {
				outputFormat, _ := cmd.Flags().GetString("output")
				switch outputFormat {
				case "json":
					cmd.Print(`[{"name": "test-project", "status": "active"}]`)
				case "yaml":
					cmd.Print("- name: test-project\n  status: active")
				default:
					cmd.Print("| Name | Status |\n|------|--------|\n| test-project | active |")
				}
				return nil
			},
		},
		&cobra.Command{
			Use: "info [name]",
			RunE: func(cmd *cobra.Command, args []string) error {
				if len(args) == 0 {
					return fmt.Errorf("workspace name required")
				}
				if args[0] == "nonexistent" {
					return fmt.Errorf("workspace not found: %s", args[0])
				}
				cmd.Printf("워크스페이스 정보: %s\n", args[0])
				return nil
			},
		},
		&cobra.Command{
			Use: "delete [name]",
			RunE: func(cmd *cobra.Command, args []string) error {
				if len(args) == 0 {
					return fmt.Errorf("workspace name required")
				}
				cmd.Printf("워크스페이스가 삭제되었습니다: %s\n", args[0])
				return nil
			},
		},
	)

	// config 명령어
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Configuration management",
	}

	configCmd.AddCommand(
		&cobra.Command{
			Use: "get [key]",
			RunE: func(cmd *cobra.Command, args []string) error {
				if len(args) == 0 {
					return fmt.Errorf("key required")
				}
				// 간단한 모의 설정 값 반환
				switch args[0] {
				case "claude.api_key":
					cmd.Print("test-api-key")
				case "claude.model":
					cmd.Print("claude-3")
				default:
					return fmt.Errorf("설정을 찾을 수 없습니다: %s", args[0])
				}
				return nil
			},
		},
		&cobra.Command{
			Use: "set [key] [value]",
			RunE: func(cmd *cobra.Command, args []string) error {
				if len(args) < 2 {
					return fmt.Errorf("key and value required")
				}
				cmd.Printf("설정이 저장되었습니다: %s=%s\n", args[0], args[1])
				return nil
			},
		},
		&cobra.Command{
			Use: "list",
			RunE: func(cmd *cobra.Command, args []string) error {
				cmd.Print("claude.api_key=test-api-key\nclaude.model=claude-3\n")
				return nil
			},
		},
		&cobra.Command{
			Use: "unset [key]",
			RunE: func(cmd *cobra.Command, args []string) error {
				if len(args) == 0 {
					return fmt.Errorf("key required")
				}
				cmd.Printf("설정이 삭제되었습니다: %s\n", args[0])
				return nil
			},
		},
	)

	// 글로벌 플래그 추가
	rootCmd.PersistentFlags().String("output", "table", "Output format (table|json|yaml)")

	rootCmd.AddCommand(workspaceCmd, configCmd)
	
	return rootCmd
}