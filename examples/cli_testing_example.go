package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/mock"
	"github.com/aicli/aicli-web/internal/testutil"
)

// CLI 테스트 프레임워크 사용 예제

func ExampleCLITestRunner_Basic() {
	// 간단한 테스트 명령어 생성
	testCmd := &cobra.Command{
		Use:   "hello",
		Short: "Say hello",
		RunE: func(cmd *cobra.Command, args []string) error {
			name := "World"
			if len(args) > 0 {
				name = args[0]
			}
			cmd.Printf("Hello, %s!", name)
			return nil
		},
	}

	// CLI 테스트 러너 생성 및 설정
	runner := testutil.NewCLITestRunner()
	runner.SetCommand(testCmd)

	// 명령어 실행
	err := runner.RunCommand("Alice")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// 출력 확인
	output := runner.GetOutput()
	fmt.Printf("Output: %s\n", output)
	// Output: Hello, Alice!
}

func ExampleCLITestRunner_WithEnvironment() {
	// 환경 변수를 사용하는 명령어
	configCmd := &cobra.Command{
		Use: "config",
		RunE: func(cmd *cobra.Command, args []string) error {
			apiKey := getEnvOrDefault("API_KEY", "default-key")
			cmd.Printf("API Key: %s", apiKey)
			return nil
		},
	}

	runner := testutil.NewCLITestRunner()
	runner.SetCommand(configCmd)
	runner.SetEnv("API_KEY", "test-api-key-12345")

	err := runner.RunCommand()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	output := runner.GetOutput()
	fmt.Printf("Output: %s\n", output)
	// Output: API Key: test-api-key-12345
}

func ExampleRunTestCases() {
	// 테스트할 명령어
	mathCmd := &cobra.Command{
		Use: "math",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 3 {
				return fmt.Errorf("usage: math <num1> <op> <num2>")
			}

			num1, op, num2 := args[0], args[1], args[2]
			
			switch op {
			case "add":
				cmd.Printf("%s + %s = result", num1, num2)
			case "sub":
				cmd.Printf("%s - %s = result", num1, num2)
			default:
				return fmt.Errorf("unsupported operation: %s", op)
			}
			return nil
		},
	}

	// 테스트 케이스들 정의
	testCases := []testutil.CLITestCase{
		{
			Name:       "덧셈 테스트",
			Args:       []string{"5", "add", "3"},
			WantErr:    false,
			WantOutput: "5 + 3 = result",
		},
		{
			Name:       "뺄셈 테스트",
			Args:       []string{"10", "sub", "4"},
			WantErr:    false,
			WantOutput: "10 - 4 = result",
		},
		{
			Name:      "인자 부족 에러",
			Args:      []string{"5"},
			WantErr:   true,
			WantError: "usage: math",
		},
		{
			Name:      "지원하지 않는 연산",
			Args:      []string{"5", "mul", "3"},
			WantErr:   true,
			WantError: "unsupported operation",
		},
	}

	// 모든 테스트 케이스 실행 (실제 테스트에서는 testing.T를 사용)
	fmt.Println("테스트 케이스 실행 예제:")
	for _, tc := range testCases {
		fmt.Printf("- %s: %v\n", tc.Name, tc.Args)
	}
}

func ExampleMockClaudeWrapper() {
	// Claude CLI 래퍼의 모의 객체 사용
	mockClaude := &testutil.MockClaudeWrapper{}
	
	// Start 메서드 모킹
	mockClaude.On("Start", mock.Anything, "/test/workspace").Return(nil)
	
	// Execute 메서드 모킹
	expectedResponse := testutil.CreateMockClaudeResponse(
		"코드를 분석했습니다. 다음과 같은 개선사항을 제안합니다...",
		"success",
	)
	mockClaude.On("Execute", mock.Anything, "analyze code").Return(expectedResponse, nil)
	
	// IsRunning 메서드 모킹
	mockClaude.On("IsRunning").Return(true)

	// 모의 객체 사용
	ctx := context.Background()
	
	err := mockClaude.Start(ctx, "/test/workspace")
	if err != nil {
		fmt.Printf("Start error: %v\n", err)
		return
	}

	response, err := mockClaude.Execute(ctx, "analyze code")
	if err != nil {
		fmt.Printf("Execute error: %v\n", err)
		return
	}

	isRunning := mockClaude.IsRunning()

	fmt.Printf("Claude Started: %v\n", err == nil)
	fmt.Printf("Response: %s\n", response.Content[:50] + "...")
	fmt.Printf("Is Running: %v\n", isRunning)
	// Claude Started: true
	// Response: 코드를 분석했습니다. 다음과 같은 개선사항을 제안합니다...
	// Is Running: true
}

func ExampleWorkspaceIntegrationTest() {
	// 실제 CLI 통합 테스트 시나리오 예제
	fmt.Println("워크스페이스 통합 테스트 시나리오:")
	
	steps := []string{
		"1. 임시 워크스페이스 생성",
		"2. 프로젝트 구조 생성",
		"3. 워크스페이스 초기화 명령 실행",
		"4. 설정 파일 생성 및 검증",
		"5. Claude CLI와의 연동 테스트",
		"6. 출력 형식 변경 테스트",
		"7. 에러 처리 시나리오 테스트",
		"8. 정리 작업",
	}

	for _, step := range steps {
		fmt.Printf("  %s\n", step)
	}
}

func ExampleTestUtilities() {
	fmt.Println("테스트 유틸리티 함수들:")
	
	utilities := map[string]string{
		"CreateTempWorkspace":       "임시 작업 공간 생성",
		"CreateTestConfig":          "테스트용 설정 파일 생성",
		"CreateTestProjectStructure": "테스트용 프로젝트 구조 생성",
		"AssertFileExists":          "파일 존재 여부 확인",
		"AssertFileContains":        "파일 내용 확인",
		"AssertOutputContains":      "명령어 출력 확인",
		"WithEnv":                   "임시 환경 변수 설정",
		"WithWorkingDir":            "임시 작업 디렉토리 변경",
		"CaptureStdout":             "표준 출력 캡처",
		"CaptureStderr":             "표준 에러 캡처",
	}

	for name, desc := range utilities {
		fmt.Printf("  %-25s: %s\n", name, desc)
	}
}

func ExamplePerformanceTest() {
	fmt.Println("성능 테스트 예제:")
	
	// 실제 구현에서는 이런 식으로 성능을 측정
	fmt.Printf("  - CLI 시작 시간 측정\n")
	fmt.Printf("  - 명령어 실행 속도 벤치마크\n")
	fmt.Printf("  - 메모리 사용량 모니터링\n")
	fmt.Printf("  - 동시 실행 테스트\n")
	fmt.Printf("  - 대용량 출력 처리 테스트\n")
}

// 헬퍼 함수들
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	fmt.Println("=== CLI 테스트 프레임워크 사용 예제 ===\n")
	
	fmt.Println("1. 기본 CLI 테스트:")
	ExampleCLITestRunner_Basic()
	fmt.Println()
	
	fmt.Println("2. 환경 변수와 함께:")
	ExampleCLITestRunner_WithEnvironment()
	fmt.Println()
	
	fmt.Println("3. 테스트 케이스 배치 실행:")
	ExampleRunTestCases()
	fmt.Println()
	
	fmt.Println("4. Claude 모의 객체 사용:")
	ExampleMockClaudeWrapper()
	fmt.Println()
	
	fmt.Println("5. 통합 테스트 시나리오:")
	ExampleWorkspaceIntegrationTest()
	fmt.Println()
	
	fmt.Println("6. 테스트 유틸리티 함수들:")
	ExampleTestUtilities()
	fmt.Println()
	
	fmt.Println("7. 성능 테스트:")
	ExamplePerformanceTest()
}