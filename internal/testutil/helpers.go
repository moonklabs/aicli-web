package testutil

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TempDir 테스트용 임시 디렉토리 생성 및 정리
func TempDir(t *testing.T, prefix string) string {
	t.Helper()
	dir, err := os.MkdirTemp("", prefix)
	if err != nil {
		t.Fatalf("임시 디렉토리 생성 실패: %v", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(dir)
	})
	return dir
}

// TempFile 테스트용 임시 파일 생성
func TempFile(t *testing.T, dir, pattern string, content string) string {
	t.Helper()
	file, err := os.CreateTemp(dir, pattern)
	if err != nil {
		t.Fatalf("임시 파일 생성 실패: %v", err)
	}
	defer file.Close()

	if content != "" {
		if _, err := file.WriteString(content); err != nil {
			t.Fatalf("파일 쓰기 실패: %v", err)
		}
	}

	return file.Name()
}

// CaptureOutput 표준 출력 캡처
func CaptureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

// CaptureError 표준 에러 출력 캡처
func CaptureError(f func()) string {
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	f()

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

// CreateTestProject 테스트용 프로젝트 구조 생성
func CreateTestProject(t *testing.T, dir string) {
	t.Helper()
	
	// 프로젝트 디렉토리 구조 생성
	dirs := []string{
		filepath.Join(dir, "cmd", "aicli"),
		filepath.Join(dir, "cmd", "api"),
		filepath.Join(dir, "internal", "cli"),
		filepath.Join(dir, "internal", "server"),
		filepath.Join(dir, "pkg", "config"),
	}

	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatalf("디렉토리 생성 실패 %s: %v", d, err)
		}
	}

	// 기본 설정 파일 생성
	configContent := `{
  "version": "1.0.0",
  "api": {
    "port": 8080
  }
}`
	TempFile(t, dir, "config.json", configContent)
}

// AssertEqual 값 비교 헬퍼
func AssertEqual(t *testing.T, expected, actual interface{}) {
	t.Helper()
	if expected != actual {
		t.Errorf("기대값과 실제값이 다름: expected=%v, actual=%v", expected, actual)
	}
}

// AssertNotNil nil 체크 헬퍼
func AssertNotNil(t *testing.T, value interface{}) {
	t.Helper()
	if value == nil {
		t.Error("값이 nil이면 안됨")
	}
}

// AssertNil nil 체크 헬퍼
func AssertNil(t *testing.T, value interface{}) {
	t.Helper()
	if value != nil {
		t.Errorf("값이 nil이어야 함: %v", value)
	}
}

// AssertContains 문자열 포함 체크
func AssertContains(t *testing.T, str, substr string) {
	t.Helper()
	if !contains(str, substr) {
		t.Errorf("문자열에 부분 문자열이 포함되지 않음: str=%q, substr=%q", str, substr)
	}
}

func contains(str, substr string) bool {
	return strings.Contains(str, substr)
}