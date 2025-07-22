package docker

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLifecycleManager_EventMonitoring(t *testing.T) {
	if os.Getenv("SKIP_DOCKER_TESTS") == "true" {
		t.Skip("Docker tests are disabled")
	}

	client, manager, cleanup := setupTestEnvironment(t)
	defer cleanup()

	projectDir, projectCleanup := createTestProjectDir(t)
	defer projectCleanup()

	// 생명주기 매니저 생성
	lifecycleManager := NewLifecycleManager(client)
	defer lifecycleManager.Close()

	workspaceID := "test-workspace-lifecycle"
	
	// 이벤트 수집용 채널
	events := make(chan ContainerEvent, 10)
	var eventsMu sync.Mutex
	var collectedEvents []ContainerEvent

	// 이벤트 핸들러 등록
	handler := func(event ContainerEvent) {
		eventsMu.Lock()
		collectedEvents = append(collectedEvents, event)
		eventsMu.Unlock()
		
		select {
		case events <- event:
		default:
		}
	}

	lifecycleManager.Subscribe(workspaceID, handler)
	defer lifecycleManager.Unsubscribe(workspaceID)

	ctx := context.Background()

	// 컨테이너 생성
	req := &CreateContainerRequest{
		WorkspaceID: workspaceID,
		Name:        "lifecycle-test",
		ProjectPath: projectDir,
		Command:     []string{"sh", "-c", "sleep 10"},
	}

	container, err := manager.CreateWorkspaceContainer(ctx, req)
	require.NoError(t, err)

	// 컨테이너 시작
	err = manager.StartContainer(ctx, container.ID)
	require.NoError(t, err)

	// 시작 이벤트 대기
	select {
	case event := <-events:
		assert.Equal(t, container.ID, event.ContainerID)
		assert.Equal(t, workspaceID, event.WorkspaceID)
		assert.Equal(t, EventTypeStart, event.Type)
	case <-time.After(5 * time.Second):
		t.Fatal("Start event not received within timeout")
	}

	// 컨테이너 중지
	err = manager.StopContainer(ctx, container.ID, 5*time.Second)
	require.NoError(t, err)

	// 중지 이벤트 대기
	select {
	case event := <-events:
		assert.Equal(t, container.ID, event.ContainerID)
		assert.Equal(t, workspaceID, event.WorkspaceID)
		// 중지 이벤트는 stop 또는 die일 수 있음
		assert.True(t, event.Type == EventTypeStop || event.Type == EventTypeDie)
	case <-time.After(5 * time.Second):
		t.Fatal("Stop event not received within timeout")
	}

	// 컨테이너 삭제
	err = manager.RemoveContainer(ctx, container.ID, false)
	require.NoError(t, err)

	// 삭제 이벤트 대기
	select {
	case event := <-events:
		assert.Equal(t, container.ID, event.ContainerID)
		assert.Equal(t, workspaceID, event.WorkspaceID)
		assert.Equal(t, EventTypeDestroy, event.Type)
	case <-time.After(5 * time.Second):
		t.Fatal("Destroy event not received within timeout")
	}

	// 수집된 이벤트 확인
	eventsMu.Lock()
	defer eventsMu.Unlock()
	
	assert.GreaterOrEqual(t, len(collectedEvents), 3, "Should have at least 3 events (start, stop/die, destroy)")
	
	// 모든 이벤트가 올바른 컨테이너와 워크스페이스에 속하는지 확인
	for _, event := range collectedEvents {
		assert.Equal(t, container.ID, event.ContainerID)
		assert.Equal(t, workspaceID, event.WorkspaceID)
		assert.NotZero(t, event.Timestamp)
	}
}

func TestLifecycleManager_WaitForContainerState(t *testing.T) {
	if os.Getenv("SKIP_DOCKER_TESTS") == "true" {
		t.Skip("Docker tests are disabled")
	}

	client, manager, cleanup := setupTestEnvironment(t)
	defer cleanup()

	projectDir, projectCleanup := createTestProjectDir(t)
	defer projectCleanup()

	// 생명주기 매니저 생성
	lifecycleManager := NewLifecycleManager(client)
	defer lifecycleManager.Close()

	ctx := context.Background()
	workspaceID := "test-workspace-wait"

	// 컨테이너 생성
	req := &CreateContainerRequest{
		WorkspaceID: workspaceID,
		Name:        "wait-test",
		ProjectPath: projectDir,
		Command:     []string{"sh", "-c", "sleep 30"},
	}

	container, err := manager.CreateWorkspaceContainer(ctx, req)
	require.NoError(t, err)

	// 현재 상태가 Created인지 확인
	currentContainer, err := manager.InspectContainer(ctx, container.ID)
	require.NoError(t, err)
	assert.Equal(t, ContainerStateCreated, currentContainer.State)

	// 컨테이너 시작 (별도 고루틴에서)
	go func() {
		time.Sleep(1 * time.Second)
		manager.StartContainer(ctx, container.ID)
	}()

	// Running 상태가 될 때까지 대기
	err = lifecycleManager.WaitForContainerState(ctx, container.ID, ContainerStateRunning, 10*time.Second)
	assert.NoError(t, err)

	// 실제로 Running 상태인지 확인
	currentContainer, err = manager.InspectContainer(ctx, container.ID)
	require.NoError(t, err)
	assert.Equal(t, ContainerStateRunning, currentContainer.State)

	// 컨테이너 정리
	err = manager.RemoveContainer(ctx, container.ID, true)
	require.NoError(t, err)
}

func TestLifecycleManager_WaitForContainerState_Timeout(t *testing.T) {
	if os.Getenv("SKIP_DOCKER_TESTS") == "true" {
		t.Skip("Docker tests are disabled")
	}

	client, manager, cleanup := setupTestEnvironment(t)
	defer cleanup()

	projectDir, projectCleanup := createTestProjectDir(t)
	defer projectCleanup()

	// 생명주기 매니저 생성
	lifecycleManager := NewLifecycleManager(client)
	defer lifecycleManager.Close()

	ctx := context.Background()
	workspaceID := "test-workspace-timeout"

	// 컨테이너 생성 (시작하지 않음)
	req := &CreateContainerRequest{
		WorkspaceID: workspaceID,
		Name:        "timeout-test",
		ProjectPath: projectDir,
		Command:     []string{"sh", "-c", "sleep 30"},
	}

	container, err := manager.CreateWorkspaceContainer(ctx, req)
	require.NoError(t, err)

	// Running 상태가 될 때까지 대기 (시작하지 않았으므로 타임아웃 발생해야 함)
	start := time.Now()
	err = lifecycleManager.WaitForContainerState(ctx, container.ID, ContainerStateRunning, 2*time.Second)
	duration := time.Since(start)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
	assert.GreaterOrEqual(t, duration, 2*time.Second)

	// 컨테이너 정리
	err = manager.RemoveContainer(ctx, container.ID, true)
	require.NoError(t, err)
}

func TestLifecycleManager_GetContainerHistory(t *testing.T) {
	if os.Getenv("SKIP_DOCKER_TESTS") == "true" {
		t.Skip("Docker tests are disabled")
	}

	client, manager, cleanup := setupTestEnvironment(t)
	defer cleanup()

	projectDir, projectCleanup := createTestProjectDir(t)
	defer projectCleanup()

	// 생명주기 매니저 생성
	lifecycleManager := NewLifecycleManager(client)
	defer lifecycleManager.Close()

	ctx := context.Background()
	workspaceID := "test-workspace-history"

	// 시작 시간 기록
	startTime := time.Now().Add(-1 * time.Second)

	// 컨테이너 생성
	req := &CreateContainerRequest{
		WorkspaceID: workspaceID,
		Name:        "history-test",
		ProjectPath: projectDir,
		Command:     []string{"sh", "-c", "sleep 10"},
	}

	container, err := manager.CreateWorkspaceContainer(ctx, req)
	require.NoError(t, err)

	// 컨테이너 시작
	err = manager.StartContainer(ctx, container.ID)
	require.NoError(t, err)

	// 잠시 대기
	time.Sleep(1 * time.Second)

	// 컨테이너 중지
	err = manager.StopContainer(ctx, container.ID, 5*time.Second)
	require.NoError(t, err)

	// 컨테이너 삭제
	err = manager.RemoveContainer(ctx, container.ID, false)
	require.NoError(t, err)

	// 이벤트 히스토리 조회
	history, err := lifecycleManager.GetContainerHistory(ctx, container.ID, startTime)
	assert.NoError(t, err)
	
	// 최소한 create, start, stop/die, destroy 이벤트가 있어야 함
	assert.GreaterOrEqual(t, len(history), 3)

	// 모든 이벤트가 올바른 컨테이너에 속하는지 확인
	for _, event := range history {
		assert.Equal(t, container.ID, event.ContainerID)
		assert.Equal(t, workspaceID, event.WorkspaceID)
		assert.True(t, event.Timestamp.After(startTime) || event.Timestamp.Equal(startTime))
	}
}

func TestLifecycleManager_MultipleSubscribers(t *testing.T) {
	if os.Getenv("SKIP_DOCKER_TESTS") == "true" {
		t.Skip("Docker tests are disabled")
	}

	client, manager, cleanup := setupTestEnvironment(t)
	defer cleanup()

	projectDir, projectCleanup := createTestProjectDir(t)
	defer projectCleanup()

	// 생명주기 매니저 생성
	lifecycleManager := NewLifecycleManager(client)
	defer lifecycleManager.Close()

	workspaceID := "test-workspace-multi"
	
	// 여러 구독자 생성
	const numSubscribers = 3
	events := make([]chan ContainerEvent, numSubscribers)
	
	for i := 0; i < numSubscribers; i++ {
		events[i] = make(chan ContainerEvent, 10)
		
		// 각 구독자를 위한 핸들러
		func(eventChan chan ContainerEvent) {
			handler := func(event ContainerEvent) {
				select {
				case eventChan <- event:
				default:
				}
			}
			lifecycleManager.Subscribe(workspaceID, handler)
		}(events[i])
	}

	ctx := context.Background()

	// 컨테이너 생성
	req := &CreateContainerRequest{
		WorkspaceID: workspaceID,
		Name:        "multi-test",
		ProjectPath: projectDir,
		Command:     []string{"sh", "-c", "sleep 5"},
	}

	container, err := manager.CreateWorkspaceContainer(ctx, req)
	require.NoError(t, err)

	// 컨테이너 시작
	err = manager.StartContainer(ctx, container.ID)
	require.NoError(t, err)

	// 모든 구독자가 시작 이벤트를 받았는지 확인
	for i := 0; i < numSubscribers; i++ {
		select {
		case event := <-events[i]:
			assert.Equal(t, container.ID, event.ContainerID)
			assert.Equal(t, workspaceID, event.WorkspaceID)
			assert.Equal(t, EventTypeStart, event.Type)
		case <-time.After(5 * time.Second):
			t.Fatalf("Subscriber %d did not receive start event within timeout", i)
		}
	}

	// 구독 해제
	lifecycleManager.Unsubscribe(workspaceID)

	// 컨테이너 정리
	err = manager.RemoveContainer(ctx, container.ID, true)
	require.NoError(t, err)
}

func TestLifecycleManager_StateMapping(t *testing.T) {
	lifecycleManager := &LifecycleManager{}

	testCases := []struct {
		dockerStatus   string
		expectedState  ContainerState
	}{
		{"create", ContainerStateCreated},
		{"start", ContainerStateRunning},
		{"running", ContainerStateRunning},
		{"stop", ContainerStateExited},
		{"die", ContainerStateExited},
		{"pause", ContainerStatePaused},
		{"unpause", ContainerStateRunning},
		{"destroy", ContainerStateRemoving},
		{"restart", ContainerStateRestarting},
		{"unknown_status", ContainerState("unknown_status")},
	}

	for _, tc := range testCases {
		result := lifecycleManager.mapDockerStatusToContainerState(tc.dockerStatus)
		assert.Equal(t, tc.expectedState, result, "Failed for docker status: %s", tc.dockerStatus)
	}
}

// 벤치마크 테스트
func BenchmarkLifecycleManager_EventProcessing(b *testing.B) {
	if os.Getenv("SKIP_DOCKER_TESTS") == "true" {
		b.Skip("Docker tests are disabled")
	}

	client, err := NewClient(DefaultConfig())
	if err != nil {
		b.Fatal(err)
	}
	defer client.Close()

	lifecycleManager := NewLifecycleManager(client)
	defer lifecycleManager.Close()

	workspaceID := "benchmark-lifecycle"
	
	// 간단한 핸들러
	handler := func(event ContainerEvent) {
		// 최소한의 처리
		_ = event.ContainerID
	}

	lifecycleManager.Subscribe(workspaceID, handler)

	// 모의 이벤트 생성
	mockEvent := ContainerEvent{
		ContainerID: "test-container",
		WorkspaceID: workspaceID,
		Type:        EventTypeStart,
		Status:      ContainerStateRunning,
		Message:     "Test event",
		Timestamp:   time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lifecycleManager.notifySubscribers(mockEvent)
	}
}