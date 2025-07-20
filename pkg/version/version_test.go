package version

import (
	"testing"
)

func TestVersion(t *testing.T) {
	// 기본 버전 정보 테스트
	if Version == "" {
		t.Log("Version is empty, which is expected during development")
	}
}

func TestGet(t *testing.T) {
	// 버전 정보 Get 함수 테스트
	info := Get()
	if info.Version == "" {
		t.Error("Version should not be empty")
	}
	if info.GoVersion == "" {
		t.Error("GoVersion should not be empty")
	}
	if info.Platform == "" {
		t.Error("Platform should not be empty")
	}
}