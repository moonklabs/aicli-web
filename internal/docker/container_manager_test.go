package docker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestEnvironment 테스트 환경을 설정합니다.
func setupTestEnvironment(t *testing.T) (*Client, *ContainerManager, func()) {
	t.Helper()

	// Docker 연결 확인
	if os.Getenv("SKIP_DOCKER_TESTS") == "true" {
		t.Skip("Docker tests are disabled")
	}

	// 테스트용 클라이언트 생성
	config := DefaultConfig()
	config.NetworkName = "aicli-test-network"
	config.DefaultImage = "alpine:latest"
	
	client, err := NewClient(config)
	require.NoError(t, err)

	manager := NewContainerManager(client)

	// 정리 함수
	cleanup := func() {
		// 테스트용 컨테이너 정리
		ctx := context.Background()
		containers, _ := client.cli.ContainerList(ctx, types.ContainerListOptions{
			All: true,
			Filters: filters.NewArgs(
				filters.Arg("label", client.labelKey("managed")),
			),
		})
		
		for _, container := range containers {
			client.cli.ContainerRemove(ctx, container.ID, types.ContainerRemoveOptions{
				Force:         true,
				RemoveVolumes: true,
			})
		}

		// 테스트 네트워크 정리
		client.cli.NetworkRemove(ctx, config.NetworkName)
		client.Close()
	}

	return client, manager, cleanup
}

// createTestProjectDir 테스트용 프로젝트 디렉토리를 생성합니다.
func createTestProjectDir(t *testing.T) (string, func()) {
	t.Helper()

	tempDir := t.TempDir()
	projectDir := filepath.Join(tempDir, "test-project")
	
	err := os.MkdirAll(projectDir, 0755)
	require.NoError(t, err)

	// 테스트 파일 생성
	testFile := filepath.Join(projectDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	return projectDir, cleanup
}

func TestContainerManager_CreateWorkspaceContainer(t *testing.T) {
	client, manager, cleanup := setupTestEnvironment(t)
	defer cleanup()

	projectDir, projectCleanup := createTestProjectDir(t)
	defer projectCleanup()

	req := &CreateContainerRequest{
		WorkspaceID: "test-workspace-001",
		Name:        "test-container",
		ProjectPath: projectDir,
		Image:       "alpine:latest",
		Command:     []string{"sh", "-c", "sleep 30"},
		Environment: map[string]string{
			"TEST_VAR": "test-value",
		},
		CPULimit:    0.5,
		MemoryLimit: 128 * 1024 * 1024, // 128MB
	}

	ctx := context.Background()

	// 컨테이너 생성
	container, err := manager.CreateWorkspaceContainer(ctx, req)
	assert.NoError(t, err)
	assert.NotEmpty(t, container.ID)
	assert.Equal(t, ContainerStateCreated, container.State)
	assert.Equal(t, req.WorkspaceID, container.WorkspaceID)

	// 생성된 컨테이너 검증
	inspect, err := client.cli.ContainerInspect(ctx, container.ID)
	assert.NoError(t, err)
	assert.Equal(t, req.WorkspaceID, inspect.Config.Labels[client.labelKey("workspace.id")])
	assert.Equal(t, req.Name, inspect.Config.Labels[client.labelKey("workspace.name")])

	// 마운트 확인
	assert.Len(t, inspect.Mounts, 1)
	assert.Equal(t, projectDir, inspect.Mounts[0].Source)
	assert.Equal(t, "/workspace", inspect.Mounts[0].Destination)

	// 환경 변수 확인
	envFound := false
	for _, env := range inspect.Config.Env {
		if env == "TEST_VAR=test-value" {
			envFound = true
			break
		}
	}
	assert.True(t, envFound, "Environment variable not found")

	// 리소스 제한 확인
	assert.Equal(t, int64(50000), inspect.HostConfig.CPUQuota)    // 0.5 CPU
	assert.Equal(t, int64(128*1024*1024), inspect.HostConfig.Memory) // 128MB

	// 정리
	err = manager.RemoveContainer(ctx, container.ID, true)
	assert.NoError(t, err)
}

func TestContainerManager_ContainerLifecycle(t *testing.T) {
	client, manager, cleanup := setupTestEnvironment(t)
	defer cleanup()

	projectDir, projectCleanup := createTestProjectDir(t)
	defer projectCleanup()

	req := &CreateContainerRequest{
		WorkspaceID: "test-workspace-002",
		Name:        "lifecycle-test",
		ProjectPath: projectDir,
		Command:     []string{"sh", "-c", "sleep 30"},
	}

	ctx := context.Background()

	// 1. 컨테이너 생성
	container, err := manager.CreateWorkspaceContainer(ctx, req)
	require.NoError(t, err)

	// 2. 컨테이너 시작
	err = manager.StartContainer(ctx, container.ID)
	assert.NoError(t, err)

	// 시작 상태 확인
	time.Sleep(1 * time.Second)
	updatedContainer, err := manager.InspectContainer(ctx, container.ID)
	assert.NoError(t, err)
	assert.Equal(t, ContainerStateRunning, updatedContainer.State)
	assert.NotNil(t, updatedContainer.Started)

	// 3. 컨테이너 중지
	err = manager.StopContainer(ctx, container.ID, 5*time.Second)
	assert.NoError(t, err)

	// 중지 상태 확인
	time.Sleep(1 * time.Second)
	updatedContainer, err = manager.InspectContainer(ctx, container.ID)
	assert.NoError(t, err)
	assert.Equal(t, ContainerStateExited, updatedContainer.State)
	assert.NotNil(t, updatedContainer.Finished)

	// 4. 컨테이너 재시작
	err = manager.RestartContainer(ctx, container.ID, 5*time.Second)
	assert.NoError(t, err)

	// 재시작 상태 확인
	time.Sleep(1 * time.Second)
	updatedContainer, err = manager.InspectContainer(ctx, container.ID)
	assert.NoError(t, err)
	assert.Equal(t, ContainerStateRunning, updatedContainer.State)

	// 5. 컨테이너 삭제
	err = manager.StopContainer(ctx, container.ID, 5*time.Second)
	assert.NoError(t, err)
	
	err = manager.RemoveContainer(ctx, container.ID, false)
	assert.NoError(t, err)

	// 삭제 확인
	_, err = manager.InspectContainer(ctx, container.ID)
	assert.Error(t, err) // 컨테이너가 없어야 함
}

func TestContainerManager_ListWorkspaceContainers(t *testing.T) {
	client, manager, cleanup := setupTestEnvironment(t)
	defer cleanup()

	projectDir, projectCleanup := createTestProjectDir(t)
	defer projectCleanup()

	workspaceID := "test-workspace-003"
	ctx := context.Background()

	// 여러 컨테이너 생성
	var containers []*WorkspaceContainer
	for i := 0; i < 3; i++ {
		req := &CreateContainerRequest{
			WorkspaceID: workspaceID,
			Name:        fmt.Sprintf("test-container-%d", i),
			ProjectPath: projectDir,
			Command:     []string{"sh", "-c", "sleep 30"},
		}

		container, err := manager.CreateWorkspaceContainer(ctx, req)
		require.NoError(t, err)
		containers = append(containers, container)
	}

	// 워크스페이스 컨테이너 목록 조회
	list, err := manager.ListWorkspaceContainers(ctx, workspaceID)
	assert.NoError(t, err)
	assert.Len(t, list, 3)

	// 각 컨테이너 확인
	for _, listedContainer := range list {
		assert.Equal(t, workspaceID, listedContainer.WorkspaceID)
		assert.Equal(t, ContainerStateCreated, listedContainer.State)
	}

	// 다른 워크스페이스의 컨테이너 목록은 비어있어야 함
	emptyList, err := manager.ListWorkspaceContainers(ctx, "non-existent-workspace")
	assert.NoError(t, err)
	assert.Empty(t, emptyList)

	// 정리
	err = manager.CleanupWorkspace(ctx, workspaceID, true)
	assert.NoError(t, err)
}

func TestContainerManager_CleanupWorkspace(t *testing.T) {
	client, manager, cleanup := setupTestEnvironment(t)
	defer cleanup()

	projectDir, projectCleanup := createTestProjectDir(t)
	defer projectCleanup()

	workspaceID := "test-workspace-004"
	ctx := context.Background()

	// 여러 컨테이너 생성 및 일부 시작
	var containers []*WorkspaceContainer
	for i := 0; i < 3; i++ {
		req := &CreateContainerRequest{
			WorkspaceID: workspaceID,
			Name:        fmt.Sprintf("cleanup-test-%d", i),
			ProjectPath: projectDir,
			Command:     []string{"sh", "-c", "sleep 60"},
		}

		container, err := manager.CreateWorkspaceContainer(ctx, req)
		require.NoError(t, err)
		containers = append(containers, container)

		// 첫 번째와 두 번째 컨테이너 시작
		if i < 2 {
			err = manager.StartContainer(ctx, container.ID)
			require.NoError(t, err)
		}
	}

	// 시작 상태 확인
	time.Sleep(1 * time.Second)

	// 워크스페이스 정리
	err := manager.CleanupWorkspace(ctx, workspaceID, false)
	assert.NoError(t, err)

	// 정리 후 컨테이너 목록이 비어있는지 확인
	list, err := manager.ListWorkspaceContainers(ctx, workspaceID)
	assert.NoError(t, err)
	assert.Empty(t, list)

	// 개별 컨테이너들이 실제로 삭제되었는지 확인
	for _, container := range containers {
		_, err := manager.InspectContainer(ctx, container.ID)
		assert.Error(t, err) // 컨테이너가 없어야 함
	}
}

func TestContainerManager_PortBindings(t *testing.T) {
	client, manager, cleanup := setupTestEnvironment(t)
	defer cleanup()

	projectDir, projectCleanup := createTestProjectDir(t)
	defer projectCleanup()

	req := &CreateContainerRequest{
		WorkspaceID: "test-workspace-005",
		Name:        "port-test",
		ProjectPath: projectDir,
		Command:     []string{"sh", "-c", "sleep 30"},
		Ports: map[string]string{
			"8080": "8080",
			"3000": "3001",
		},
	}

	ctx := context.Background()

	// 컨테이너 생성
	container, err := manager.CreateWorkspaceContainer(ctx, req)
	require.NoError(t, err)

	// 포트 바인딩 확인
	inspect, err := client.cli.ContainerInspect(ctx, container.ID)
	assert.NoError(t, err)

	// 포트 바인딩이 설정되었는지 확인
	assert.NotEmpty(t, inspect.HostConfig.PortBindings)

	// 정리
	err = manager.RemoveContainer(ctx, container.ID, true)
	assert.NoError(t, err)
}

func TestContainerManager_ResourceLimits(t *testing.T) {
	client, manager, cleanup := setupTestEnvironment(t)
	defer cleanup()

	projectDir, projectCleanup := createTestProjectDir(t)
	defer projectCleanup()

	req := &CreateContainerRequest{
		WorkspaceID: "test-workspace-006",
		Name:        "resource-test",
		ProjectPath: projectDir,
		CPULimit:    0.25,                    // 25% CPU
		MemoryLimit: 64 * 1024 * 1024,       // 64MB
		Command:     []string{"sh", "-c", "sleep 30"},
	}

	ctx := context.Background()

	// 컨테이너 생성
	container, err := manager.CreateWorkspaceContainer(ctx, req)
	require.NoError(t, err)

	// 리소스 제한 확인
	inspect, err := client.cli.ContainerInspect(ctx, container.ID)
	assert.NoError(t, err)

	assert.Equal(t, int64(25000), inspect.HostConfig.CPUQuota)    // 0.25 * 100000
	assert.Equal(t, int64(100000), inspect.HostConfig.CPUPeriod)
	assert.Equal(t, int64(64*1024*1024), inspect.HostConfig.Memory)
	assert.Equal(t, int64(64*1024*1024), inspect.HostConfig.MemorySwap)

	// PID 제한 확인
	assert.NotNil(t, inspect.HostConfig.PidsLimit)
	assert.Equal(t, int64(100), *inspect.HostConfig.PidsLimit)

	// 정리
	err = manager.RemoveContainer(ctx, container.ID, true)
	assert.NoError(t, err)
}

func TestContainerManager_SecuritySettings(t *testing.T) {
	client, manager, cleanup := setupTestEnvironment(t)
	defer cleanup()

	projectDir, projectCleanup := createTestProjectDir(t)
	defer projectCleanup()

	req := &CreateContainerRequest{
		WorkspaceID: "test-workspace-007",
		Name:        "security-test",
		ProjectPath: projectDir,
		ReadOnly:    true,
		Command:     []string{"sh", "-c", "sleep 30"},
	}

	ctx := context.Background()

	// 컨테이너 생성
	container, err := manager.CreateWorkspaceContainer(ctx, req)
	require.NoError(t, err)

	// 보안 설정 확인
	inspect, err := client.cli.ContainerInspect(ctx, container.ID)
	assert.NoError(t, err)

	// 읽기 전용 루트 파일시스템 확인
	assert.True(t, inspect.HostConfig.ReadonlyRootfs)

	// Capability 설정 확인
	assert.Contains(t, inspect.HostConfig.CapDrop, "ALL")
	assert.Contains(t, inspect.HostConfig.CapAdd, "CHOWN")
	assert.Contains(t, inspect.HostConfig.CapAdd, "SETUID")
	assert.Contains(t, inspect.HostConfig.CapAdd, "SETGID")
	assert.Contains(t, inspect.HostConfig.CapAdd, "DAC_OVERRIDE")

	// 보안 옵션 확인
	assert.Contains(t, inspect.HostConfig.SecurityOpt, "no-new-privileges:true")

	// 재시작 정책 확인
	assert.Equal(t, "unless-stopped", inspect.HostConfig.RestartPolicy.Name)

	// 정리
	err = manager.RemoveContainer(ctx, container.ID, true)
	assert.NoError(t, err)
}

func TestContainerManager_ErrorHandling(t *testing.T) {
	client, manager, cleanup := setupTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()

	// 잘못된 이미지로 컨테이너 생성 시도
	req := &CreateContainerRequest{
		WorkspaceID: "test-workspace-008",
		Name:        "error-test",
		ProjectPath: "/nonexistent/path",
		Image:       "nonexistent:image",
		Command:     []string{"sh", "-c", "sleep 30"},
	}

	_, err := manager.CreateWorkspaceContainer(ctx, req)
	assert.Error(t, err)

	// 존재하지 않는 컨테이너 조작 시도
	err = manager.StartContainer(ctx, "nonexistent-container-id")
	assert.Error(t, err)

	err = manager.StopContainer(ctx, "nonexistent-container-id", 5*time.Second)
	assert.Error(t, err)

	err = manager.RestartContainer(ctx, "nonexistent-container-id", 5*time.Second)
	assert.Error(t, err)

	err = manager.RemoveContainer(ctx, "nonexistent-container-id", false)
	assert.Error(t, err)

	_, err = manager.InspectContainer(ctx, "nonexistent-container-id")
	assert.Error(t, err)
}

// 벤치마크 테스트
func BenchmarkContainerManager_CreateContainer(b *testing.B) {
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

	req := &CreateContainerRequest{
		WorkspaceID: "benchmark-workspace",
		Name:        "benchmark-test",
		ProjectPath: projectDir,
		Command:     []string{"sh", "-c", "sleep 1"},
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req.WorkspaceID = fmt.Sprintf("benchmark-workspace-%d", i)
		container, err := manager.CreateWorkspaceContainer(ctx, req)
		if err != nil {
			b.Fatal(err)
		}

		// 즉시 정리
		_ = manager.RemoveContainer(ctx, container.ID, true)
	}
}

func BenchmarkContainerManager_ListContainers(b *testing.B) {
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
	workspaceID := "benchmark-list-workspace"

	// 테스트용 컨테이너 생성
	var containers []*WorkspaceContainer
	for i := 0; i < 10; i++ {
		req := &CreateContainerRequest{
			WorkspaceID: workspaceID,
			Name:        fmt.Sprintf("benchmark-list-%d", i),
			ProjectPath: projectDir,
			Command:     []string{"sh", "-c", "sleep 60"},
		}

		container, err := manager.CreateWorkspaceContainer(context.Background(), req)
		if err != nil {
			b.Fatal(err)
		}
		containers = append(containers, container)
	}

	defer func() {
		// 정리
		for _, container := range containers {
			_ = manager.RemoveContainer(context.Background(), container.ID, true)
		}
	}()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := manager.ListWorkspaceContainers(ctx, workspaceID)
		if err != nil {
			b.Fatal(err)
		}
	}
}