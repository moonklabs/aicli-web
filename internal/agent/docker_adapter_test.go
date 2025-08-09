package agent

import (
	"context"
	"io"
	"testing"
)

// Mock 설정을 완료하여 테스트 활성화

// Mock 정의는 mocks_test.go 파일에서 통합 관리됩니다.

func TestDockerAdapter_CreateContainer(t *testing.T) {

	// Arrange
	containerManager := &MockContainerManager{}
	client := &MockDockerClient{}
	statsCollector := &MockStatsCollector{}
	adapter := NewDockerAdapter(containerManager, client, statsCollector)

	config := ContainerConfig{
		Image:       "test:latest",
		Environment: map[string]string{"ENV": "test"},
		WorkingDir:  "/app",
		Labels:      map[string]string{"app": "test"},
	}

	// Act
	result, err := adapter.CreateContainer(context.Background(), config)

	// Assert
	if err != nil {
		t.Fatalf("CreateContainer 실패: %v", err)
	}

	if result.ID != "test-container-id" {
		t.Errorf("예상 ID: test-container-id, 실제: %s", result.ID)
	}
}

func TestDockerAdapter_StartContainer(t *testing.T) {

	// Arrange
	containerManager := &MockContainerManager{}
	client := &MockDockerClient{}
	statsCollector := &MockStatsCollector{}
	adapter := NewDockerAdapter(containerManager, client, statsCollector)

	// Act
	err := adapter.StartContainer(context.Background(), "test-container-id")

	// Assert
	if err != nil {
		t.Fatalf("StartContainer 실패: %v", err)
	}
}

func TestDockerAdapter_StopContainer(t *testing.T) {

	// Arrange
	containerManager := &MockContainerManager{}
	client := &MockDockerClient{}
	statsCollector := &MockStatsCollector{}
	adapter := NewDockerAdapter(containerManager, client, statsCollector)

	// Act
	err := adapter.StopContainer(context.Background(), "test-container-id")

	// Assert
	if err != nil {
		t.Fatalf("StopContainer 실패: %v", err)
	}
}

func TestDockerAdapter_GetContainerStatus(t *testing.T) {

	// Arrange
	containerManager := &MockContainerManager{}
	client := &MockDockerClient{}
	statsCollector := &MockStatsCollector{}
	adapter := NewDockerAdapter(containerManager, client, statsCollector)

	// Act
	status, err := adapter.GetContainerStatus(context.Background(), "test-container-id")

	// Assert
	if err != nil {
		t.Fatalf("GetContainerStatus 실패: %v", err)
	}

	if status.ID != "test-container-id" {
		t.Errorf("예상 ID: test-container-id, 실제: %s", status.ID)
	}

	if status.Status != "running" {
		t.Errorf("예상 상태: running, 실제: %s", status.Status)
	}
}

func TestDockerAdapter_GetContainerHealth(t *testing.T) {

	// Arrange
	containerManager := &MockContainerManager{}
	client := &MockDockerClient{}
	statsCollector := &MockStatsCollector{}
	adapter := NewDockerAdapter(containerManager, client, statsCollector)

	// Act
	health, err := adapter.GetContainerHealth(context.Background(), "test-container-id")

	// Assert
	if err != nil {
		t.Fatalf("GetContainerHealth 실패: %v", err)
	}

	if health.Status != "healthy" {
		t.Errorf("예상 상태: healthy, 실제: %s", health.Status)
	}
}

func TestDockerAdapter_GetContainerMetrics(t *testing.T) {
	// Arrange
	containerManager := &MockContainerManager{}
	client := &MockDockerClient{}
	statsCollector := &MockStatsCollector{}
	adapter := NewDockerAdapter(containerManager, client, statsCollector)

	// Act
	metrics, err := adapter.GetContainerMetrics(context.Background(), "test-container-id")

	// Assert
	if err != nil {
		t.Fatalf("GetContainerMetrics 실패: %v", err)
	}

	if metrics.ContainerID != "test-container-id" {
		t.Errorf("예상 컨테이너 ID: test-container-id, 실제: %s", metrics.ContainerID)
	}
}

func TestDockerAdapter_GetContainerLogs(t *testing.T) {
	// Arrange
	containerManager := &MockContainerManager{}
	client := &MockDockerClient{}
	statsCollector := &MockStatsCollector{}
	adapter := NewDockerAdapter(containerManager, client, statsCollector)

	opts := LogOptions{
		Follow:     false,
		Timestamps: true,
		Tail:       100,
	}

	// Act
	logStream, err := adapter.GetContainerLogs(context.Background(), "test-container-id", opts)

	// Assert
	if err != nil {
		t.Fatalf("GetContainerLogs 실패: %v", err)
	}

	if logStream == nil {
		t.Fatal("로그 스트림이 nil입니다")
	}

	// 로그 읽기 테스트
	data, err := logStream.Read()
	if err != nil && err != io.EOF {
		t.Fatalf("로그 읽기 실패: %v", err)
	}

	if len(data) == 0 {
		t.Error("로그 데이터가 비어있습니다")
	}

	// 스트림 종료
	err = logStream.Close()
	if err != nil {
		t.Fatalf("로그 스트림 종료 실패: %v", err)
	}
}

func TestDockerAdapter_ExecuteCommand(t *testing.T) {
	// Arrange
	containerManager := &MockContainerManager{}
	client := &MockDockerClient{}
	statsCollector := &MockStatsCollector{}
	adapter := NewDockerAdapter(containerManager, client, statsCollector)

	// Act
	result, err := adapter.ExecuteCommand(context.Background(), "test-container-id", []string{"echo", "test"})

	// Assert
	if err != nil {
		t.Fatalf("ExecuteCommand 실패: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("예상 종료 코드: 0, 실제: %d", result.ExitCode)
	}
}
