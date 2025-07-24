package testutil

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

// CLITestRunner CLI 명령어 실행을 위한 테스트 러너
type CLITestRunner struct {
	cmd        *cobra.Command
	stdin      io.Reader
	stdout     *bytes.Buffer
	stderr     *bytes.Buffer
	env        map[string]string
	workingDir string
	timeout    time.Duration
}

// NewCLITestRunner 새로운 CLI 테스트 러너 생성
func NewCLITestRunner() *CLITestRunner {
	return &CLITestRunner{
		stdout: &bytes.Buffer{},
		stderr: &bytes.Buffer{},
		env:    make(map[string]string),
	}
}

// SetCommand 테스트할 명령어 설정
func (r *CLITestRunner) SetCommand(cmd *cobra.Command) {
	r.cmd = cmd
}

// SetStdin 표준 입력 설정
func (r *CLITestRunner) SetStdin(stdin io.Reader) {
	r.stdin = stdin
}

// SetEnv 환경 변수 설정
func (r *CLITestRunner) SetEnv(key, value string) {
	r.env[key] = value
}

// SetWorkingDir 작업 디렉토리 설정
func (r *CLITestRunner) SetWorkingDir(dir string) {
	r.workingDir = dir
}

// SetTimeout 명령어 실행 타임아웃 설정
func (r *CLITestRunner) SetTimeout(timeout time.Duration) {
	r.timeout = timeout
}


// RunCommand 명령어 실행
func (r *CLITestRunner) RunCommand(args ...string) error {
	if r.cmd == nil {
		panic("command not set - call SetCommand first")
	}

	// 환경 변수 설정
	oldEnv := r.setupEnvironment()
	defer r.restoreEnvironment(oldEnv)

	// 작업 디렉토리 설정
	oldDir := r.setupWorkingDir()
	defer r.restoreWorkingDir(oldDir)

	// 명령어 설정
	r.cmd.SetArgs(args)
	r.cmd.SetOut(r.stdout)
	r.cmd.SetErr(r.stderr)
	
	if r.stdin != nil {
		r.cmd.SetIn(r.stdin)
	}

	// 타임아웃 설정이 있는 경우 context 사용
	if r.timeout > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
		defer cancel()
		
		// context를 사용하여 실행 (단순화된 구현)
		done := make(chan error, 1)
		go func() {
			done <- r.cmd.Execute()
		}()
		
		select {
		case err := <-done:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return r.cmd.Execute()
}

// GetOutput 표준 출력 내용 반환
func (r *CLITestRunner) GetOutput() string {
	return r.stdout.String()
}

// GetError 표준 에러 내용 반환
func (r *CLITestRunner) GetError() string {
	return r.stderr.String()
}

// Reset 버퍼 초기화
func (r *CLITestRunner) Reset() {
	r.stdout.Reset()
	r.stderr.Reset()
}

// setupEnvironment 환경 변수 설정
func (r *CLITestRunner) setupEnvironment() map[string]string {
	oldEnv := make(map[string]string)
	
	for key, value := range r.env {
		oldEnv[key] = os.Getenv(key)
		os.Setenv(key, value)
	}
	
	return oldEnv
}

// restoreEnvironment 환경 변수 복원
func (r *CLITestRunner) restoreEnvironment(oldEnv map[string]string) {
	for key, value := range oldEnv {
		if value == "" {
			os.Unsetenv(key)
		} else {
			os.Setenv(key, value)
		}
	}
}

// setupWorkingDir 작업 디렉토리 설정
func (r *CLITestRunner) setupWorkingDir() string {
	if r.workingDir == "" {
		return ""
	}
	
	oldDir, _ := os.Getwd()
	os.Chdir(r.workingDir)
	return oldDir
}

// restoreWorkingDir 작업 디렉토리 복원
func (r *CLITestRunner) restoreWorkingDir(oldDir string) {
	if oldDir != "" {
		os.Chdir(oldDir)
	}
}

// CLITestCase CLI 테스트 케이스 정의
type CLITestCase struct {
	Name        string
	Args        []string
	Env         map[string]string
	Stdin       string
	WantErr     bool
	WantCode    int
	WantOutput  string
	WantError   string
	Setup       func(t *testing.T) string // 테스트 설정, 작업 디렉토리 반환
	Cleanup     func(t *testing.T)        // 테스트 정리
}

// RunTestCases CLI 테스트 케이스들을 실행
func RunTestCases(t *testing.T, cmd *cobra.Command, testCases []CLITestCase) {
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			runner := NewCLITestRunner()
			runner.SetCommand(cmd)

			// 환경 변수 설정
			for key, value := range tc.Env {
				runner.SetEnv(key, value)
			}

			// 표준 입력 설정
			if tc.Stdin != "" {
				runner.SetStdin(strings.NewReader(tc.Stdin))
			}

			// 테스트 설정
			var workDir string
			if tc.Setup != nil {
				workDir = tc.Setup(t)
				if workDir != "" {
					runner.SetWorkingDir(workDir)
				}
			}

			// 테스트 정리 등록
			if tc.Cleanup != nil {
				defer tc.Cleanup(t)
			}

			// 명령어 실행
			err := runner.RunCommand(tc.Args...)

			// 에러 검증
			if tc.WantErr {
				AssertNotNil(t, err)
				if tc.WantError != "" {
					AssertContains(t, err.Error(), tc.WantError)
				}
			} else {
				AssertNil(t, err)
			}

			// 출력 검증
			if tc.WantOutput != "" {
				output := runner.GetOutput()
				AssertContains(t, output, tc.WantOutput)
			}
		})
	}
}

// AssertOutputContains 출력에 특정 문자열이 포함되어 있는지 검증
func AssertOutputContains(t *testing.T, runner *CLITestRunner, expected string) {
	t.Helper()
	output := runner.GetOutput()
	AssertContains(t, output, expected)
}

// AssertErrorContains 에러 출력에 특정 문자열이 포함되어 있는지 검증
func AssertErrorContains(t *testing.T, runner *CLITestRunner, expected string) {
	t.Helper()
	errorOutput := runner.GetError()
	AssertContains(t, errorOutput, expected)
}

// AssertNoOutput 출력이 비어있는지 검증
func AssertNoOutput(t *testing.T, runner *CLITestRunner) {
	t.Helper()
	output := runner.GetOutput()
	if output != "" {
		t.Errorf("예상치 못한 출력: %q", output)
	}
}

// AssertNoError 에러 출력이 비어있는지 검증
func AssertNoError(t *testing.T, runner *CLITestRunner) {
	t.Helper()
	errorOutput := runner.GetError()
	if errorOutput != "" {
		t.Errorf("예상치 못한 에러 출력: %q", errorOutput)
	}
}