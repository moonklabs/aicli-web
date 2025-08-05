//go:build integration
// +build integration

package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aicli/aicli-web/internal/storage"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestE2EScenarios는 전체 시스템의 E2E 시나리오를 테스트합니다.
func TestE2EScenarios(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E tests in short mode")
	}

	t.Run("Complete Workflow", func(t *testing.T) {
		testCompleteWorkflow(t)
	})

	t.Run("Concurrent Sessions", func(t *testing.T) {
		testConcurrentSessions(t)
	})

	t.Run("Session Recovery", func(t *testing.T) {
		testSessionRecovery(t)
	})

	t.Run("Resource Management", func(t *testing.T) {
		testResourceManagement(t)
	})
}

// testCompleteWorkflow는 완전한 에이전트 워크플로우를 테스트합니다.
func testCompleteWorkflow(t *testing.T) {
	ctx := context.Background()

	// Docker 클라이언트 초기화
	dockerHost := os.Getenv("DOCKER_HOST")
	if dockerHost == "" {
		dockerHost = "unix:///var/run/docker.sock"
	}

	dockerClient, err := client.NewClientWithOpts(
		client.WithHost(dockerHost),
		client.WithAPIVersionNegotiation(),
	)
	require.NoError(t, err)
	defer dockerClient.Close()

	// Docker 연결 확인
	_, err = dockerClient.Info(ctx)
	require.NoError(t, err, "Docker 데몬에 연결할 수 없습니다")

	// 테스트 환경 설정
	store, err := storage.New()
	require.NoError(t, err)
	defer store.Close()

	// 세션 매니저 및 에이전트 서비스 초기화 (가상의 서비스)
	t.Log("=== 에이전트 생명주기 E2E 테스트 시작 ===")

	// 1. 에이전트 컨테이너 생성 테스트
	containerName := fmt.Sprintf("test-agent-%d", time.Now().Unix())
	createResp, err := dockerClient.ContainerCreate(ctx,
		&container.Config{
			Image: "alpine:latest",
			Cmd:   []string{"sleep", "60"},
			Labels: map[string]string{
				"test.type":    "e2e",
				"test.purpose": "agent-lifecycle",
			},
		},
		&container.HostConfig{
			AutoRemove: true,
			Resources: container.Resources{
				Memory:   128 * 1024 * 1024, // 128MB
				NanoCPUs: 100000000,         // 0.1 CPU
			},
		},
		nil,
		nil,
		containerName,
	)
	require.NoError(t, err, "에이전트 컨테이너 생성 실패")
	
	containerID := createResp.ID
	t.Logf("에이전트 컨테이너 생성 완료: %s", containerID[:12])

	// 정리 함수 등록
	defer func() {
		if err := dockerClient.ContainerStop(ctx, containerID, container.StopOptions{}); err != nil {
			t.Logf("컨테이너 중지 실패: %v", err)
		}
		if err := dockerClient.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{Force: true}); err != nil {
			t.Logf("컨테이너 제거 실패: %v", err)
		}
	}()

	// 2. 에이전트 시작
	err = dockerClient.ContainerStart(ctx, containerID, types.ContainerStartOptions{})
	require.NoError(t, err, "에이전트 컨테이너 시작 실패")
	t.Log("에이전트 컨테이너 시작 완료")

	// 3. 에이전트 상태 확인 (실행 중 대기)
	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			t.Fatal("에이전트가 30초 내에 실행 상태가 되지 않음")
		case <-ticker.C:
			inspect, err := dockerClient.ContainerInspect(ctx, containerID)
			require.NoError(t, err)
			
			if inspect.State.Running {
				t.Log("에이전트가 실행 중 상태가 되었습니다")
				goto AgentRunning
			}
		}
	}

AgentRunning:
	// 4. 에이전트 작업 실행 시뮬레이션 (exec를 통한 명령 실행)
	execConfig := types.ExecConfig{
		Cmd:          []string{"echo", "Hello from E2E test agent"},
		AttachStdout: true,
		AttachStderr: true,
	}

	execResp, err := dockerClient.ContainerExecCreate(ctx, containerID, execConfig)
	require.NoError(t, err, "명령 실행 준비 실패")

	execAttach, err := dockerClient.ContainerExecAttach(ctx, execResp.ID, types.ExecStartCheck{})
	require.NoError(t, err, "명령 실행 연결 실패")
	defer execAttach.Close()

	// 실행 결과 읽기
	output, err := io.ReadAll(execAttach.Reader)
	require.NoError(t, err, "명령 실행 결과 읽기 실패")
	
	outputStr := string(output)
	assert.Contains(t, outputStr, "Hello from E2E test agent", "명령 실행 결과가 예상과 다름")
	t.Logf("에이전트 작업 실행 완료: %s", strings.TrimSpace(outputStr))

	// 5. 에이전트 로그 확인
	logOptions := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: true,
		Details:    true,
	}

	logReader, err := dockerClient.ContainerLogs(ctx, containerID, logOptions)
	require.NoError(t, err, "로그 조회 실패")
	defer logReader.Close()

	logData, err := io.ReadAll(logReader)
	require.NoError(t, err, "로그 읽기 실패")
	assert.NotEmpty(t, logData, "로그가 비어있음")
	t.Logf("에이전트 로그 확인 완료: %d bytes", len(logData))

	// 6. 에이전트 리소스 사용량 확인
	stats, err := dockerClient.ContainerStats(ctx, containerID, false)
	require.NoError(t, err, "리소스 통계 조회 실패")
	defer stats.Body.Close()

	var statsData types.StatsJSON
	decoder := json.NewDecoder(stats.Body)
	err = decoder.Decode(&statsData)
	require.NoError(t, err, "리소스 통계 파싱 실패")

	// 메모리 사용량 확인 (128MB 제한 내에 있는지)
	memoryUsage := statsData.MemoryStats.Usage
	memoryLimit := statsData.MemoryStats.Limit
	assert.LessOrEqual(t, memoryUsage, memoryLimit, "메모리 사용량이 제한을 초과함")
	t.Logf("에이전트 리소스 사용량 - 메모리: %d/%d bytes", memoryUsage, memoryLimit)

	// 7. 에이전트 중지
	timeout = 10 * time.Second
	err = dockerClient.ContainerStop(ctx, containerID, &timeout)
	require.NoError(t, err, "에이전트 중지 실패")
	t.Log("에이전트 중지 완료")

	// 8. 에이전트 상태 확인 (중지됨)
	inspect, err := dockerClient.ContainerInspect(ctx, containerID)
	require.NoError(t, err)
	assert.False(t, inspect.State.Running, "에이전트가 아직 실행 중입니다")
	assert.Equal(t, "exited", inspect.State.Status, "에이전트 상태가 exited가 아닙니다")
	t.Log("에이전트 상태 확인 완료: 정상적으로 중지됨")

	t.Log("=== 에이전트 생명주기 E2E 테스트 완료 ===")
}

// testConcurrentSessions는 동시 세션 처리를 테스트합니다.
func testConcurrentSessions(t *testing.T) {
	store, err := storage.New()
	require.NoError(t, err)
	defer store.Close()

	sessionManager := NewSessionManager(store.Session())
	ctx := context.Background()

	const numSessions = 10
	var wg sync.WaitGroup
	sessionIDs := make([]string, numSessions)
	errors := make([]error, numSessions)

	// 동시에 여러 세션 생성
	for i := 0; i < numSessions; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			sessionID, err := sessionManager.Create(ctx, &SessionConfig{
				WorkspaceID:  fmt.Sprintf("workspace-%d", index),
				SystemPrompt: fmt.Sprintf("Assistant %d", index),
				MaxTurns:     5,
			})

			sessionIDs[index] = sessionID
			errors[index] = err
		}(i)
	}

	wg.Wait()

	// 모든 세션이 성공적으로 생성되었는지 확인
	createdCount := 0
	for i := 0; i < numSessions; i++ {
		if errors[i] == nil && sessionIDs[i] != "" {
			createdCount++
		}
	}

	assert.Equal(t, numSessions, createdCount, "All sessions should be created successfully")

	// 모든 세션 ID가 고유한지 확인
	uniqueIDs := make(map[string]bool)
	for _, id := range sessionIDs {
		if id != "" {
			assert.False(t, uniqueIDs[id], "Session ID should be unique: %s", id)
			uniqueIDs[id] = true
		}
	}

	// 세션 정리
	for _, id := range sessionIDs {
		if id != "" {
			err := sessionManager.Close(ctx, id)
			assert.NoError(t, err)
		}
	}
}

// testSessionRecovery는 세션 복구 기능을 테스트합니다.
func testSessionRecovery(t *testing.T) {
	store, err := storage.New()
	require.NoError(t, err)
	defer store.Close()

	sessionManager := NewSessionManager(store.Session())
	ctx := context.Background()

	// 1. 세션 생성
	sessionID, err := sessionManager.Create(ctx, &SessionConfig{
		WorkspaceID:  "recovery-test",
		SystemPrompt: "Test assistant",
		MaxTurns:     10,
	})
	require.NoError(t, err)

	// 2. 세션 상태를 "error"로 설정
	err = sessionManager.Update(ctx, sessionID, SessionUpdate{
		State: &SessionState{
			Status:      "error",
			LastMessage: "Connection lost",
			ErrorCount:  1,
		},
	})
	require.NoError(t, err)

	// 3. 에러 상태의 세션 조회
	session, err := sessionManager.Get(ctx, sessionID)
	require.NoError(t, err)
	assert.Equal(t, "error", session.State.Status)
	assert.Equal(t, 1, session.State.ErrorCount)

	// 4. 세션 복구 (상태를 다시 active로)
	err = sessionManager.Update(ctx, sessionID, SessionUpdate{
		State: &SessionState{
			Status:      "active",
			LastMessage: "Recovered",
			ErrorCount:  0,
		},
	})
	require.NoError(t, err)

	// 5. 복구된 세션 확인
	recoveredSession, err := sessionManager.Get(ctx, sessionID)
	require.NoError(t, err)
	assert.Equal(t, "active", recoveredSession.State.Status)
	assert.Equal(t, 0, recoveredSession.State.ErrorCount)

	// 정리
	err = sessionManager.Close(ctx, sessionID)
	require.NoError(t, err)
}

// testResourceManagement는 리소스 관리를 테스트합니다.
func testResourceManagement(t *testing.T) {
	store, err := storage.New()
	require.NoError(t, err)
	defer store.Close()

	sessionManager := NewSessionManager(store.Session())
	ctx := context.Background()

	// 리소스 제한이 있는 세션 생성
	sessionID, err := sessionManager.Create(ctx, &SessionConfig{
		WorkspaceID: "resource-test",
		MaxTurns:    5,
		MaxMemory:   100 * 1024 * 1024, // 100MB
		MaxCPU:      0.5,               // 50% CPU
		MaxDuration: 10 * time.Minute,
	})
	require.NoError(t, err)

	session, err := sessionManager.Get(ctx, sessionID)
	require.NoError(t, err)

	// 리소스 제한이 올바르게 설정되었는지 확인
	assert.Equal(t, int64(100*1024*1024), session.Config.MaxMemory)
	assert.Equal(t, 0.5, session.Config.MaxCPU)
	assert.Equal(t, 10*time.Minute, session.Config.MaxDuration)

	// 정리
	err = sessionManager.Close(ctx, sessionID)
	require.NoError(t, err)
}

// TestStreamingIntegration은 스트리밍 통합을 테스트합니다.
func TestStreamingIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping streaming tests in short mode")
	}

	t.Run("Message Streaming", func(t *testing.T) {
		testMessageStreaming(t)
	})

	t.Run("Backpressure Handling", func(t *testing.T) {
		testBackpressureHandling(t)
	})
}

// testMessageStreaming은 메시지 스트리밍을 테스트합니다.
func testMessageStreaming(t *testing.T) {
	// Mock 스트림 생성
	streamData := `{"type":"text","content":"Hello","id":"msg1"}
{"type":"text","content":" World","id":"msg2"}
{"type":"system","content":"Complete","id":"msg3"}`

	parser := NewJSONStreamParser(strings.NewReader(streamData), nil)

	messages := make([]Message, 0)

	// 스트림에서 모든 메시지 읽기
	for {
		response, err := parser.ParseNext()
		if err != nil {
			if err == io.EOF {
				break
			}
			require.NoError(t, err)
		}

		messages = append(messages, Message{
			Type:    response.Type,
			Content: response.Content,
			ID:      response.MessageID,
		})
	}

	// 메시지 검증
	require.Len(t, messages, 3)
	assert.Equal(t, "text", messages[0].Type)
	assert.Equal(t, "Hello", messages[0].Content)
	assert.Equal(t, "text", messages[1].Type)
	assert.Equal(t, " World", messages[1].Content)
	assert.Equal(t, "system", messages[2].Type)
	assert.Equal(t, "Complete", messages[2].Content)
}

// testBackpressureHandling은 백프레셔 처리를 테스트합니다.
func testBackpressureHandling(t *testing.T) {
	// 대용량 메시지 스트림 생성
	var streamBuilder strings.Builder
	for i := 0; i < 1000; i++ {
		streamBuilder.WriteString(fmt.Sprintf(`{"type":"text","content":"Message %d","id":"msg%d"}`+"\n", i, i))
	}

	parser := NewJSONStreamParser(strings.NewReader(streamBuilder.String()), nil)

	// 백프레셔 핸들러 생성
	handler := NewBackpressureHandler(100, DropOldest) // 버퍼 크기 100

	messageCount := 0
	droppedCount := 0

	// 스트림 처리
	for {
		response, err := parser.ParseNext()
		if err != nil {
			if err == io.EOF {
				break
			}
			require.NoError(t, err)
		}

		msg := Message{
			Type:    response.Type,
			Content: response.Content,
			ID:      response.MessageID,
		}

		accepted := handler.Submit(msg)
		if accepted {
			messageCount++
		} else {
			droppedCount++
		}
	}

	// 백프레셔로 인해 일부 메시지가 드롭되었는지 확인
	t.Logf("Processed: %d, Dropped: %d", messageCount, droppedCount)
	assert.Greater(t, droppedCount, 0, "Some messages should be dropped due to backpressure")
	assert.Less(t, messageCount, 1000, "Not all messages should be processed")
}
