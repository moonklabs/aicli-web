package testutil

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

// CreateTempWorkspace 임시 워크스페이스 생성
func CreateTempWorkspace(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "aicli-test-workspace-*")
	if err != nil {
		t.Fatalf("임시 워크스페이스 생성 실패: %v", err)
	}
	
	t.Cleanup(func() {
		os.RemoveAll(dir)
	})
	
	return dir
}

// CreateTestConfig 테스트용 설정 파일 생성
func CreateTestConfig(t *testing.T, config map[string]interface{}) string {
	t.Helper()
	workspaceDir := CreateTempWorkspace(t)
	configFile := filepath.Join(workspaceDir, "config.yaml")
	
	data, err := yaml.Marshal(config)
	if err != nil {
		t.Fatalf("설정 파일 마셜링 실패: %v", err)
	}
	
	err = os.WriteFile(configFile, data, 0644)
	if err != nil {
		t.Fatalf("설정 파일 쓰기 실패: %v", err)
	}
	
	return configFile
}

// CreateTestConfigWithPath 지정된 경로에 테스트 설정 파일 생성
func CreateTestConfigWithPath(t *testing.T, dir string, filename string, config map[string]interface{}) string {
	t.Helper()
	configFile := filepath.Join(dir, filename)
	
	data, err := yaml.Marshal(config)
	if err != nil {
		t.Fatalf("설정 파일 마셜링 실패: %v", err)
	}
	
	err = os.WriteFile(configFile, data, 0644)
	if err != nil {
		t.Fatalf("설정 파일 쓰기 실패: %v", err)
	}
	
	return configFile
}

// CreateTestProjectStructure 테스트용 프로젝트 구조 생성
func CreateTestProjectStructure(t *testing.T, workspaceDir string) {
	t.Helper()
	
	// 기본 디렉토리 구조
	dirs := []string{
		"src",
		"tests", 
		"docs",
		"config",
		".git",
		"build",
		"scripts",
	}
	
	for _, dir := range dirs {
		dirPath := filepath.Join(workspaceDir, dir)
		err := os.MkdirAll(dirPath, 0755)
		if err != nil {
			t.Fatalf("디렉토리 생성 실패 %s: %v", dirPath, err)
		}
	}
	
	// 기본 파일들
	files := map[string]string{
		"README.md":           "# Test Project\n\nThis is a test project.",
		"package.json":        `{"name": "test-project", "version": "1.0.0"}`,
		".gitignore":          "node_modules/\n*.log\n.env",
		"src/main.js":         "console.log('Hello, World!');",
		"tests/test.js":       "// Test file",
		"config/app.yaml":     "app:\n  name: test-app\n  version: 1.0.0",
		"docs/api.md":         "# API Documentation",
		"scripts/build.sh":    "#!/bin/bash\necho 'Building...'",
	}
	
	for filename, content := range files {
		filePath := filepath.Join(workspaceDir, filename)
		
		// 디렉토리가 없으면 생성
		dir := filepath.Dir(filePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("파일 디렉토리 생성 실패 %s: %v", dir, err)
		}
		
		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("파일 생성 실패 %s: %v", filePath, err)
		}
	}
}

// AssertFileExists 파일 존재 여부 확인
func AssertFileExists(t *testing.T, filepath string) {
	t.Helper()
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		t.Errorf("파일이 존재하지 않음: %s", filepath)
	}
}

// AssertFileNotExists 파일이 존재하지 않는지 확인
func AssertFileNotExists(t *testing.T, filepath string) {
	t.Helper()
	if _, err := os.Stat(filepath); !os.IsNotExist(err) {
		t.Errorf("파일이 존재해서는 안됨: %s", filepath)
	}
}

// AssertFileContent 파일 내용 확인
func AssertFileContent(t *testing.T, filepath, expected string) {
	t.Helper()
	content, err := os.ReadFile(filepath)
	if err != nil {
		t.Fatalf("파일 읽기 실패: %v", err)
	}
	
	actual := string(content)
	if actual != expected {
		t.Errorf("파일 내용이 일치하지 않음:\n기대값: %q\n실제값: %q", expected, actual)
	}
}

// AssertFileContains 파일에 특정 내용이 포함되어 있는지 확인
func AssertFileContains(t *testing.T, filepath, expected string) {
	t.Helper()
	content, err := os.ReadFile(filepath)
	if err != nil {
		t.Fatalf("파일 읽기 실패: %v", err)
	}
	
	actual := string(content)
	if !strings.Contains(actual, expected) {
		t.Errorf("파일에 예상 내용이 포함되지 않음:\n파일: %s\n찾는 내용: %q\n실제 내용: %q", filepath, expected, actual)
	}
}

// AssertDirExists 디렉토리 존재 여부 확인
func AssertDirExists(t *testing.T, dirpath string) {
	t.Helper()
	info, err := os.Stat(dirpath)
	if os.IsNotExist(err) {
		t.Errorf("디렉토리가 존재하지 않음: %s", dirpath)
		return
	}
	if !info.IsDir() {
		t.Errorf("경로가 디렉토리가 아님: %s", dirpath)
	}
}

// CreateTempConfigFile 임시 설정 파일 생성 (간단 버전)
func CreateTempConfigFile(t *testing.T, content string) string {
	t.Helper()
	file := TempFile(t, "", "config-*.yaml", content)
	return file
}

// WithEnv 환경 변수를 임시로 설정하고 테스트 실행
func WithEnv(t *testing.T, env map[string]string, fn func()) {
	t.Helper()
	
	// 기존 환경 변수 백업
	oldEnv := make(map[string]string)
	for key := range env {
		oldEnv[key] = os.Getenv(key)
	}
	
	// 테스트 후 환경 변수 복원
	defer func() {
		for key, oldValue := range oldEnv {
			if oldValue == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, oldValue)
			}
		}
	}()
	
	// 새 환경 변수 설정
	for key, value := range env {
		os.Setenv(key, value)
	}
	
	// 테스트 함수 실행
	fn()
}

// WithWorkingDir 작업 디렉토리를 임시로 변경하고 테스트 실행
func WithWorkingDir(t *testing.T, dir string, fn func()) {
	t.Helper()
	
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("현재 작업 디렉토리 확인 실패: %v", err)
	}
	
	defer func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Errorf("작업 디렉토리 복원 실패: %v", err)
		}
	}()
	
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("작업 디렉토리 변경 실패: %v", err)
	}
	
	fn()
}

// CaptureStdout 표준 출력 캡처 (개선된 버전)
func CaptureStdout(t *testing.T, fn func()) string {
	t.Helper()
	
	// 파이프 생성
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("파이프 생성 실패: %v", err)
	}
	defer r.Close()
	
	// 원본 stdout 백업
	originalStdout := os.Stdout
	defer func() {
		os.Stdout = originalStdout
	}()
	
	// stdout 리다이렉트
	os.Stdout = w
	
	// 출력 읽기를 위한 채널
	outputChan := make(chan string)
	go func() {
		defer close(outputChan)
		output, _ := io.ReadAll(r)
		outputChan <- string(output)
	}()
	
	// 테스트 함수 실행
	fn()
	
	// 파이프 닫기 (EOF 신호)
	w.Close()
	
	// 출력 수집
	output := <-outputChan
	
	return output
}

// CaptureStderr 표준 에러 캡처 (개선된 버전)
func CaptureStderr(t *testing.T, fn func()) string {
	t.Helper()
	
	// 파이프 생성
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("파이프 생성 실패: %v", err)
	}
	defer r.Close()
	
	// 원본 stderr 백업
	originalStderr := os.Stderr
	defer func() {
		os.Stderr = originalStderr
	}()
	
	// stderr 리다이렉트
	os.Stderr = w
	
	// 출력 읽기를 위한 채널
	outputChan := make(chan string)
	go func() {
		defer close(outputChan)
		output, _ := io.ReadAll(r)
		outputChan <- string(output)
	}()
	
	// 테스트 함수 실행
	fn()
	
	// 파이프 닫기 (EOF 신호)
	w.Close()
	
	// 출력 수집
	output := <-outputChan
	
	return output
}

// AssertCommandSuccess 명령어가 성공적으로 실행되었는지 확인
func AssertCommandSuccess(t *testing.T, runner *CLITestRunner, args ...string) {
	t.Helper()
	
	err := runner.RunCommand(args...)
	if err != nil {
		t.Errorf("명령어 실행 실패: %v\n출력: %s\n에러: %s", 
			err, runner.GetOutput(), runner.GetError())
	}
}

// AssertCommandFailure 명령어가 실패했는지 확인
func AssertCommandFailure(t *testing.T, runner *CLITestRunner, args ...string) {
	t.Helper()
	
	err := runner.RunCommand(args...)
	if err == nil {
		t.Errorf("명령어가 실패해야 했으나 성공함\n출력: %s", runner.GetOutput())
	}
}

// Retry 테스트 재시도 유틸리티
func Retry(t *testing.T, attempts int, delay time.Duration, fn func() error) {
	t.Helper()
	
	var lastErr error
	for i := 0; i < attempts; i++ {
		if lastErr = fn(); lastErr == nil {
			return
		}
		
		if i < attempts-1 {
			time.Sleep(delay)
		}
	}
	
	t.Fatalf("모든 재시도 실패 (%d번 시도): %v", attempts, lastErr)
}

// Eventually 조건이 만족될 때까지 기다리기
func Eventually(t *testing.T, condition func() bool, timeout time.Duration, interval time.Duration) {
	t.Helper()
	
	deadline := time.Now().Add(timeout)
	
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(interval)
	}
	
	t.Fatalf("조건이 %v 시간 내에 만족되지 않음", timeout)
}

// CreateMockClaudeResponse 모의 Claude 응답 생성
func CreateMockClaudeResponse(content string, status string) *MockClaudeResponse {
	return &MockClaudeResponse{
		Content:   content,
		Status:    status,
		Timestamp: time.Now(),
		Metadata: map[string]string{
			"model":      "claude-3",
			"session_id": "test-session",
		},
	}
}

// CreateMockWorkspaceInfo 모의 워크스페이스 정보 생성
func CreateMockWorkspaceInfo(id, name, path string) *WorkspaceInfo {
	return &WorkspaceInfo{
		ID:         id,
		Name:       name,
		Path:       path,
		Status:     "active",
		CreatedAt:  time.Now(),
		LastAccess: time.Now(),
		Metadata: map[string]string{
			"version": "1.0",
			"type":    "test",
		},
	}
}

// LogTestStep 테스트 단계 로깅
func LogTestStep(t *testing.T, step string, args ...interface{}) {
	t.Helper()
	message := fmt.Sprintf(step, args...)
	t.Logf("🔍 %s", message)
}

// MeasureTime 실행 시간 측정
func MeasureTime(t *testing.T, name string, fn func()) {
	t.Helper()
	start := time.Now()
	fn()
	elapsed := time.Since(start)
	t.Logf("⏱️  %s 실행 시간: %v", name, elapsed)
}