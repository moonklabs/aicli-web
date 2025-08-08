package pty_streaming

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	docker_client "github.com/docker/docker/client"
	"github.com/google/uuid"
)

// CreateContainer creates a test container
func (tdm *TestDockerManager) CreateContainer(image string) (string, error) {
	tdm.mutex.Lock()
	defer tdm.mutex.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 컨테이너 설정
	config := &container.Config{
		Image: image,
		Cmd:   []string{"sleep", "3600"}, // 1시간 대기
		Tty:   true,
		Env: []string{
			"TERM=xterm-256color",
		},
	}

	hostConfig := &container.HostConfig{
		AutoRemove: false,
		Resources: container.Resources{
			Memory:   512 * 1024 * 1024, // 512MB
			CPUQuota: 100000,
		},
	}

	// 컨테이너 이름 생성
	containerName := fmt.Sprintf("test-container-%s", uuid.New().String()[:8])

	// 컨테이너 생성
	resp, err := tdm.client.ContainerCreate(ctx, config, hostConfig, nil, nil, containerName)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	// 컨테이너 시작
	if err := tdm.client.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		// 실패 시 컨테이너 제거
		tdm.client.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{Force: true})
		return "", fmt.Errorf("failed to start container: %w", err)
	}

	// 컨테이너 정보 저장
	testContainer := &TestContainer{
		ID:        resp.ID,
		Image:     image,
		Name:      containerName,
		CreatedAt: time.Now(),
		Status:    "running",
	}
	tdm.containers[resp.ID] = testContainer

	log.Infof("Created test container: %s (%s)", containerName, resp.ID)
	return resp.ID, nil
}

// RemoveContainer removes a container
func (tdm *TestDockerManager) RemoveContainer(containerID string) error {
	tdm.mutex.Lock()
	defer tdm.mutex.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 컨테이너 정지 및 제거
	if err := tdm.client.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{
		Force:         true,
		RemoveVolumes: true,
	}); err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}

	delete(tdm.containers, containerID)
	log.Infof("Removed test container: %s", containerID)
	return nil
}

// StopContainer stops a container
func (tdm *TestDockerManager) StopContainer(containerID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	timeout := 10
	if err := tdm.client.ContainerStop(ctx, containerID, container.StopOptions{
		Timeout: &timeout,
	}); err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}

	tdm.mutex.Lock()
	if container, exists := tdm.containers[containerID]; exists {
		container.Status = "stopped"
	}
	tdm.mutex.Unlock()

	return nil
}

// StartContainer starts a stopped container
func (tdm *TestDockerManager) StartContainer(containerID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := tdm.client.ContainerStart(ctx, containerID, types.ContainerStartOptions{}); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	tdm.mutex.Lock()
	if container, exists := tdm.containers[containerID]; exists {
		container.Status = "running"
	}
	tdm.mutex.Unlock()

	return nil
}

// RemoveAllContainers removes all test containers
func (tdm *TestDockerManager) RemoveAllContainers() {
	tdm.mutex.Lock()
	containerIDs := make([]string, 0, len(tdm.containers))
	for id := range tdm.containers {
		containerIDs = append(containerIDs, id)
	}
	tdm.mutex.Unlock()

	for _, id := range containerIDs {
		if err := tdm.RemoveContainer(id); err != nil {
			log.Errorf("Failed to remove container %s: %v", id, err)
		}
	}
}

// GetContainer returns container information
func (tdm *TestDockerManager) GetContainer(containerID string) (*TestContainer, error) {
	tdm.mutex.RLock()
	defer tdm.mutex.RUnlock()

	container, exists := tdm.containers[containerID]
	if !exists {
		return nil, fmt.Errorf("container not found: %s", containerID)
	}

	return container, nil
}

// ListContainers returns all test containers
func (tdm *TestDockerManager) ListContainers() []*TestContainer {
	tdm.mutex.RLock()
	defer tdm.mutex.RUnlock()

	containers := make([]*TestContainer, 0, len(tdm.containers))
	for _, container := range tdm.containers {
		containers = append(containers, container)
	}

	return containers
}

// GetContainerStats returns container resource statistics
func (tdm *TestDockerManager) GetContainerStats(containerID string) (*ContainerStats, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	statsResponse, err := tdm.client.ContainerStats(ctx, containerID, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get container stats: %w", err)
	}
	defer statsResponse.Body.Close()

	// 기본 통계 반환
	return &ContainerStats{
		ContainerID: containerID,
		Timestamp:   time.Now(),
	}, nil
}

// ContainerStats represents container statistics
type ContainerStats struct {
	ContainerID   string
	CPUUsage      float64
	MemoryUsage   int64
	MemoryLimit   int64
	NetworkRxBytes uint64
	NetworkTxBytes uint64
	Timestamp     time.Time
}