package pty

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

// TestDockerPTYManagerCreation Docker PTY 관리자 생성 테스트
func TestDockerPTYManagerCreation(t *testing.T) {
	config := DefaultDockerPTYConfig()
	
	// Docker 데몬이 없을 수 있으므로 스킵 가능
	manager, err := NewDockerPTYManager(config)
	if err != nil {
		t.Skipf("Docker daemon not available: %v", err)
		return
	}
	defer manager.Close()

	if manager == nil {
		t.Fatal("Manager should not be nil")
	}

	if manager.client == nil {
		t.Error("Docker client should be initialized")
	}

	stats := manager.GetStats()
	if active, ok := stats["active_sessions"].(int); ok {
		if active != 0 {
			t.Errorf("Expected 0 active sessions, got %d", active)
		}
	}
}

// TestContainerOperations 컨테이너 작업 테스트
func TestContainerOperations(t *testing.T) {
	manager, err := NewDockerPTYManager(nil)
	if err != nil {
		t.Skipf("Docker daemon not available: %v", err)
		return
	}
	defer manager.Close()

	ctx := context.Background()

	// 테스트 컨테이너 생성
	containerConfig := &ContainerConfig{
		Name:       "test-pty-container",
		Image:      "alpine:latest",
		Command:    []string{"/bin/sh"},
		AutoRemove: true,
	}

	containerID, err := manager.CreateContainer(ctx, containerConfig)
	if err != nil {
		t.Skipf("Failed to create test container (image might not be available): %v", err)
		return
	}

	// 컨테이너 실행 상태 확인
	running, err := manager.IsContainerRunning(containerID)
	if err != nil {
		t.Fatalf("Failed to check container status: %v", err)
	}

	if !running {
		t.Error("Container should be running")
	}

	// 컨테이너 정보 조회
	info, err := manager.GetContainerInfo(containerID)
	if err != nil {
		t.Fatalf("Failed to get container info: %v", err)
	}

	if info["id"] != containerID {
		t.Errorf("Container ID mismatch: expected %s, got %v", containerID, info["id"])
	}

	// PTY 연결 테스트
	ptyConfig := DefaultPTYConfig()
	ptyFile, err := manager.AttachPTY(ctx, containerID, ptyConfig)
	if err != nil {
		t.Fatalf("Failed to attach PTY: %v", err)
	}

	if ptyFile == nil {
		t.Fatal("PTY file should not be nil")
	}

	// 통계 확인
	stats := manager.GetStats()
	if attached, ok := stats["total_attached"].(uint64); ok {
		if attached != 1 {
			t.Errorf("Expected 1 attached session, got %d", attached)
		}
	}

	// 정리
	ptyFile.Close()
}

// TestPTYResize PTY 크기 조정 테스트
func TestPTYResize(t *testing.T) {
	manager, err := NewDockerPTYManager(nil)
	if err != nil {
		t.Skipf("Docker daemon not available: %v", err)
		return
	}
	defer manager.Close()

	ctx := context.Background()

	// 테스트 컨테이너 생성
	containerConfig := &ContainerConfig{
		Name:       "test-resize-container",
		Image:      "alpine:latest",
		Command:    []string{"/bin/sh"},
		AutoRemove: true,
	}

	containerID, err := manager.CreateContainer(ctx, containerConfig)
	if err != nil {
		t.Skipf("Failed to create test container: %v", err)
		return
	}

	// PTY 연결
	ptyConfig := DefaultPTYConfig()
	ptyFile, err := manager.AttachPTY(ctx, containerID, ptyConfig)
	if err != nil {
		t.Fatalf("Failed to attach PTY: %v", err)
	}
	defer ptyFile.Close()

	// 세션 ID 찾기 (테스트용)
	var sessionID string
	manager.mutex.RLock()
	for id := range manager.sessions {
		sessionID = id
		break
	}
	manager.mutex.RUnlock()

	if sessionID == "" {
		t.Fatal("No session found")
	}

	// 크기 조정
	err = manager.ResizePTY(sessionID, 40, 120)
	if err != nil {
		t.Errorf("Failed to resize PTY: %v", err)
	}
}

// TestConcurrentPTYOperations 동시 PTY 작업 테스트
func TestConcurrentPTYOperations(t *testing.T) {
	manager, err := NewDockerPTYManager(nil)
	if err != nil {
		t.Skipf("Docker daemon not available: %v", err)
		return
	}
	defer manager.Close()

	ctx := context.Background()

	// 여러 컨테이너 생성
	containerIDs := make([]string, 0, 3)
	for i := 0; i < 3; i++ {
		containerConfig := &ContainerConfig{
			Name:       fmt.Sprintf("test-concurrent-%d", i),
			Image:      "alpine:latest",
			Command:    []string{"/bin/sh"},
			AutoRemove: true,
		}

		containerID, err := manager.CreateContainer(ctx, containerConfig)
		if err != nil {
			t.Skipf("Failed to create container %d: %v", i, err)
			continue
		}
		containerIDs = append(containerIDs, containerID)
	}

	if len(containerIDs) == 0 {
		t.Skip("No containers created")
		return
	}

	// 동시에 PTY 연결
	ptys := make([]*os.File, 0, len(containerIDs))
	errCh := make(chan error, len(containerIDs))

	for _, containerID := range containerIDs {
		go func(cid string) {
			ptyConfig := DefaultPTYConfig()
			ptyFile, err := manager.AttachPTY(ctx, cid, ptyConfig)
			if err != nil {
				errCh <- err
				return
			}
			ptys = append(ptys, ptyFile)
			errCh <- nil
		}(containerID)
	}

	// 결과 대기
	for i := 0; i < len(containerIDs); i++ {
		if err := <-errCh; err != nil {
			t.Errorf("Failed to attach PTY: %v", err)
		}
	}

	// 통계 확인
	stats := manager.GetStats()
	if active, ok := stats["active_sessions"].(int); ok {
		if active != len(containerIDs) {
			t.Errorf("Expected %d active sessions, got %d", len(containerIDs), active)
		}
	}

	// 정리
	for _, pty := range ptys {
		if pty != nil {
			pty.Close()
		}
	}
}

// BenchmarkPTYAttach PTY 연결 벤치마크
func BenchmarkPTYAttach(b *testing.B) {
	manager, err := NewDockerPTYManager(nil)
	if err != nil {
		b.Skipf("Docker daemon not available: %v", err)
		return
	}
	defer manager.Close()

	ctx := context.Background()

	// 테스트 컨테이너 생성
	containerConfig := &ContainerConfig{
		Name:       "bench-pty-container",
		Image:      "alpine:latest",
		Command:    []string{"/bin/sh"},
		AutoRemove: true,
	}

	containerID, err := manager.CreateContainer(ctx, containerConfig)
	if err != nil {
		b.Skipf("Failed to create container: %v", err)
		return
	}

	ptyConfig := DefaultPTYConfig()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ptyFile, err := manager.AttachPTY(ctx, containerID, ptyConfig)
		if err != nil {
			b.Fatalf("Failed to attach PTY: %v", err)
		}
		
		// 세션 찾기 및 정리
		manager.mutex.RLock()
		var sessionID string
		for id := range manager.sessions {
			sessionID = id
			break
		}
		manager.mutex.RUnlock()
		
		if sessionID != "" {
			manager.DetachPTY(sessionID)
		}
		ptyFile.Close()
	}
}