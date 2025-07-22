package docker

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestContainerManager_FullWorkflow 전체 워크플로우 통합 테스트
func TestContainerManager_FullWorkflow(t *testing.T) {
	if os.Getenv("SKIP_DOCKER_TESTS") == "true" {
		t.Skip("Docker tests are disabled")
	}

	client, manager, cleanup := setupTestEnvironment(t)
	defer cleanup()

	projectDir, projectCleanup := createTestProjectDir(t)
	defer projectCleanup()

	// 생명주기 매니저와 통합 테스트
	lifecycleManager := NewLifecycleManager(client)
	defer lifecycleManager.Close()

	workspaceID := "integration-test-workspace"
	
	// 이벤트 추적
	var events []ContainerEvent
	var eventsMu sync.Mutex

	handler := func(event ContainerEvent) {
		eventsMu.Lock()
		events = append(events, event)
		eventsMu.Unlock()
	}

	lifecycleManager.Subscribe(workspaceID, handler)
	defer lifecycleManager.Unsubscribe(workspaceID)

	ctx := context.Background()

	// Phase 1: 컨테이너 생성
	req := &CreateContainerRequest{
		WorkspaceID: workspaceID,
		Name:        "integration-test",
		ProjectPath: projectDir,
		Image:       "alpine:latest",
		Command:     []string{"sh", "-c", "sleep 60"},
		Environment: map[string]string{
			"TEST_ENV": "integration",
		},
		CPULimit:    0.5,
		MemoryLimit: 128 * 1024 * 1024, // 128MB
	}

	t.Log("Phase 1: Creating container...")
	container, err := manager.CreateWorkspaceContainer(ctx, req)
	require.NoError(t, err)
	assert.NotEmpty(t, container.ID)
	assert.Equal(t, ContainerStateCreated, container.State)

	// Phase 2: 컨테이너 시작
	t.Log("Phase 2: Starting container...")
	err = manager.StartContainer(ctx, container.ID)
	require.NoError(t, err)

	// 시작 상태 확인
	time.Sleep(2 * time.Second)
	updatedContainer, err := manager.InspectContainer(ctx, container.ID)
	require.NoError(t, err)
	assert.Equal(t, ContainerStateRunning, updatedContainer.State)
	assert.NotNil(t, updatedContainer.Started)

	// Phase 3: 컨테이너 목록 조회
	t.Log("Phase 3: Listing containers...")
	containers, err := manager.ListWorkspaceContainers(ctx, workspaceID)
	require.NoError(t, err)
	assert.Len(t, containers, 1)
	assert.Equal(t, container.ID, containers[0].ID)

	// Phase 4: 컨테이너 재시작
	t.Log("Phase 4: Restarting container...")
	err = manager.RestartContainer(ctx, container.ID, 10*time.Second)
	require.NoError(t, err)

	// 재시작 확인
	time.Sleep(2 * time.Second)
	updatedContainer, err = manager.InspectContainer(ctx, container.ID)
	require.NoError(t, err)
	assert.Equal(t, ContainerStateRunning, updatedContainer.State)

	// Phase 5: 컨테이너 중지
	t.Log("Phase 5: Stopping container...")
	err = manager.StopContainer(ctx, container.ID, 10*time.Second)
	require.NoError(t, err)

	// 중지 상태 확인
	time.Sleep(2 * time.Second)
	updatedContainer, err = manager.InspectContainer(ctx, container.ID)
	require.NoError(t, err)
	assert.Equal(t, ContainerStateExited, updatedContainer.State)

	// Phase 6: 컨테이너 삭제
	t.Log("Phase 6: Removing container...")
	err = manager.RemoveContainer(ctx, container.ID, false)
	require.NoError(t, err)

	// 삭제 확인
	_, err = manager.InspectContainer(ctx, container.ID)
	assert.Error(t, err) // 컨테이너가 없어야 함

	// Phase 7: 이벤트 확인
	t.Log("Phase 7: Verifying events...")
	time.Sleep(1 * time.Second) // 이벤트 처리 대기

	eventsMu.Lock()
	defer eventsMu.Unlock()

	t.Logf("Collected %d events", len(events))
	for i, event := range events {
		t.Logf("Event %d: Type=%s, Status=%s, Container=%s", 
			i, event.Type, event.Status, event.ContainerID[:12])
	}

	// 최소한의 이벤트가 있는지 확인
	assert.GreaterOrEqual(t, len(events), 3, "Should have at least 3 events")

	// 모든 이벤트가 올바른 컨테이너에 속하는지 확인
	for _, event := range events {
		assert.Equal(t, container.ID, event.ContainerID)
		assert.Equal(t, workspaceID, event.WorkspaceID)
	}
}

// TestContainerManager_ConcurrentOperations 동시 작업 테스트
func TestContainerManager_ConcurrentOperations(t *testing.T) {
	if os.Getenv("SKIP_DOCKER_TESTS") == "true" {
		t.Skip("Docker tests are disabled")
	}

	client, manager, cleanup := setupTestEnvironment(t)
	defer cleanup()

	projectDir, projectCleanup := createTestProjectDir(t)
	defer projectCleanup()

	ctx := context.Background()
	workspaceID := "concurrent-test-workspace"

	// 동시에 여러 컨테이너 생성
	const numContainers = 5
	var wg sync.WaitGroup
	containers := make([]*WorkspaceContainer, numContainers)
	errors := make([]error, numContainers)

	t.Log("Creating containers concurrently...")
	for i := 0; i < numContainers; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			req := &CreateContainerRequest{
				WorkspaceID: workspaceID,
				Name:        fmt.Sprintf("concurrent-test-%d", idx),
				ProjectPath: projectDir,
				Command:     []string{"sh", "-c", "sleep 30"},
			}

			container, err := manager.CreateWorkspaceContainer(ctx, req)
			containers[idx] = container
			errors[idx] = err
		}(i)
	}

	wg.Wait()

	// 모든 생성이 성공했는지 확인
	for i, err := range errors {
		assert.NoError(t, err, "Container %d creation failed", i)
		assert.NotNil(t, containers[i], "Container %d is nil", i)
	}

	// 동시에 모든 컨테이너 시작
	t.Log("Starting containers concurrently...")
	for i := 0; i < numContainers; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			errors[idx] = manager.StartContainer(ctx, containers[idx].ID)
		}(i)
	}

	wg.Wait()

	// 모든 시작이 성공했는지 확인
	for i, err := range errors {
		assert.NoError(t, err, "Container %d start failed", i)
	}

	// 실행 상태 확인
	time.Sleep(2 * time.Second)
	runningCount := 0
	for i, container := range containers {
		updated, err := manager.InspectContainer(ctx, container.ID)
		assert.NoError(t, err, "Container %d inspect failed", i)
		if updated.State == ContainerStateRunning {
			runningCount++
		}
	}

	assert.Equal(t, numContainers, runningCount, "All containers should be running")

	// 워크스페이스 정리로 모든 컨테이너 한번에 삭제
	t.Log("Cleaning up workspace...")
	err := manager.CleanupWorkspace(ctx, workspaceID, true)
	assert.NoError(t, err)

	// 모든 컨테이너가 삭제되었는지 확인
	list, err := manager.ListWorkspaceContainers(ctx, workspaceID)
	assert.NoError(t, err)
	assert.Empty(t, list, "All containers should be removed")
}

// TestContainerManager_ResourceConstraints 리소스 제약 테스트
func TestContainerManager_ResourceConstraints(t *testing.T) {
	if os.Getenv("SKIP_DOCKER_TESTS") == "true" {
		t.Skip("Docker tests are disabled")
	}

	client, manager, cleanup := setupTestEnvironment(t)
	defer cleanup()

	projectDir, projectCleanup := createTestProjectDir(t)
	defer projectCleanup()

	ctx := context.Background()

	// 다양한 리소스 제약으로 테스트
	testCases := []struct {
		name        string
		cpuLimit    float64
		memoryLimit int64
		ports       map[string]string
	}{
		{
			name:        "low-resources",
			cpuLimit:    0.1,
			memoryLimit: 32 * 1024 * 1024, // 32MB
		},
		{
			name:        "with-ports",
			cpuLimit:    0.5,
			memoryLimit: 128 * 1024 * 1024, // 128MB
			ports: map[string]string{
				"8080": "8080",
				"9090": "9091",
			},
		},
		{
			name:        "high-resources",
			cpuLimit:    1.0,
			memoryLimit: 512 * 1024 * 1024, // 512MB
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := &CreateContainerRequest{
				WorkspaceID: fmt.Sprintf("resource-test-%s", tc.name),
				Name:        tc.name,
				ProjectPath: projectDir,
				Command:     []string{"sh", "-c", "sleep 10"},
				CPULimit:    tc.cpuLimit,
				MemoryLimit: tc.memoryLimit,
				Ports:       tc.ports,
			}

			// 컨테이너 생성
			container, err := manager.CreateWorkspaceContainer(ctx, req)
			require.NoError(t, err)

			// 설정 확인
			inspect, err := client.cli.ContainerInspect(ctx, container.ID)
			require.NoError(t, err)

			// CPU 제한 확인
			expectedCPUQuota := int64(tc.cpuLimit * 100000)
			assert.Equal(t, expectedCPUQuota, inspect.HostConfig.CPUQuota)

			// 메모리 제한 확인
			assert.Equal(t, tc.memoryLimit, inspect.HostConfig.Memory)

			// 포트 확인
			if len(tc.ports) > 0 {
				assert.NotEmpty(t, inspect.HostConfig.PortBindings)
			}

			// 정리
			err = manager.RemoveContainer(ctx, container.ID, true)
			assert.NoError(t, err)
		})
	}
}

// TestContainerManager_ErrorRecovery 에러 복구 테스트
func TestContainerManager_ErrorRecovery(t *testing.T) {
	if os.Getenv("SKIP_DOCKER_TESTS") == "true" {
		t.Skip("Docker tests are disabled")
	}

	client, manager, cleanup := setupTestEnvironment(t)
	defer cleanup()

	projectDir, projectCleanup := createTestProjectDir(t)
	defer projectCleanup()

	ctx := context.Background()
	workspaceID := "error-recovery-workspace"

	// 1. 동일한 이름으로 컨테이너 중복 생성 시도
	req := &CreateContainerRequest{
		WorkspaceID: workspaceID,
		Name:        "duplicate-test",
		ProjectPath: projectDir,
		Command:     []string{"sh", "-c", "sleep 30"},
	}

	t.Log("Creating first container...")
	container1, err := manager.CreateWorkspaceContainer(ctx, req)
	require.NoError(t, err)

	t.Log("Creating second container with same workspace ID...")
	container2, err := manager.CreateWorkspaceContainer(ctx, req)
	require.NoError(t, err)
	
	// 두 번째 컨테이너는 새로운 ID를 가져야 함 (기존 컨테이너가 정리됨)
	assert.NotEqual(t, container1.ID, container2.ID)

	// 2. 이미 시작된 컨테이너를 다시 시작하려고 시도
	err = manager.StartContainer(ctx, container2.ID)
	require.NoError(t, err)

	// 이미 실행 중인 컨테이너를 다시 시작해도 에러가 발생하지 않아야 함
	err = manager.StartContainer(ctx, container2.ID)
	assert.NoError(t, err) // Docker는 이미 실행 중인 컨테이너 시작 요청을 무시함

	// 3. 존재하지 않는 컨테이너 조작
	nonExistentID := "non-existent-container-id"
	
	err = manager.StartContainer(ctx, nonExistentID)
	assert.Error(t, err)

	err = manager.StopContainer(ctx, nonExistentID, 5*time.Second)
	assert.Error(t, err)

	_, err = manager.InspectContainer(ctx, nonExistentID)
	assert.Error(t, err)

	// 4. 강제 삭제 테스트
	t.Log("Testing force removal...")
	err = manager.RemoveContainer(ctx, container2.ID, true)
	assert.NoError(t, err) // 실행 중이어도 강제 삭제는 성공해야 함

	// 정리
	err = manager.CleanupWorkspace(ctx, workspaceID, true)
	assert.NoError(t, err)
}

// BenchmarkContainerManager_Operations 컨테이너 작업 벤치마크
func BenchmarkContainerManager_Operations(b *testing.B) {
	if os.Getenv("SKIP_DOCKER_TESTS") == "true" {
		b.Skip("Docker tests are disabled")
	}

	client, err := NewClient(DefaultConfig())
	if err != nil {
		b.Fatal(err)
	}
	defer client.Close()

	manager := NewContainerManager(client)
	projectDir := b.TempDir()

	ctx := context.Background()

	b.Run("CreateContainer", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			req := &CreateContainerRequest{
				WorkspaceID: fmt.Sprintf("bench-create-%d", i),
				Name:        fmt.Sprintf("bench-test-%d", i),
				ProjectPath: projectDir,
				Command:     []string{"sh", "-c", "sleep 1"},
			}

			container, err := manager.CreateWorkspaceContainer(ctx, req)
			if err != nil {
				b.Fatal(err)
			}

			// 즉시 정리
			_ = manager.RemoveContainer(ctx, container.ID, true)
		}
	})

	// 벤치마크를 위한 컨테이너 생성
	containers := make([]*WorkspaceContainer, 10)
	workspaceID := "benchmark-workspace"

	for i := 0; i < len(containers); i++ {
		req := &CreateContainerRequest{
			WorkspaceID: workspaceID,
			Name:        fmt.Sprintf("bench-container-%d", i),
			ProjectPath: projectDir,
			Command:     []string{"sh", "-c", "sleep 60"},
		}

		container, err := manager.CreateWorkspaceContainer(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
		containers[i] = container
	}

	defer func() {
		// 벤치마크 후 정리
		for _, container := range containers {
			_ = manager.RemoveContainer(ctx, container.ID, true)
		}
	}()

	b.Run("StartContainer", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			idx := i % len(containers)
			_ = manager.StartContainer(ctx, containers[idx].ID)
		}
	})

	b.Run("InspectContainer", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			idx := i % len(containers)
			_, _ = manager.InspectContainer(ctx, containers[idx].ID)
		}
	})

	b.Run("ListWorkspaceContainers", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = manager.ListWorkspaceContainers(ctx, workspaceID)
		}
	})
}