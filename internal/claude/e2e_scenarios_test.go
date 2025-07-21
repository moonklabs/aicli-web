// +build integration

package claude

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/drumcap/aicli-web/internal/storage"
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

// testCompleteWorkflow는 완전한 워크플로우를 테스트합니다.
func testCompleteWorkflow(t *testing.T) {
	// 테스트 환경 설정
	store, err := storage.New()
	require.NoError(t, err)
	defer store.Close()

	sessionManager := NewSessionManager(store.Session())
	
	ctx := context.Background()
	
	// 1. 세션 생성
	sessionID, err := sessionManager.Create(ctx, &SessionConfig{
		WorkspaceID:  "test-workspace",
		SystemPrompt: "You are a helpful assistant",
		MaxTurns:     10,
		Tools:        []string{"Read", "Write"},
	})
	require.NoError(t, err)
	assert.NotEmpty(t, sessionID)

	// 2. 세션 조회
	session, err := sessionManager.Get(ctx, sessionID)
	require.NoError(t, err)
	assert.Equal(t, sessionID, session.ID)
	assert.Equal(t, "test-workspace", session.Config.WorkspaceID)

	// 3. 세션 상태 업데이트 (시뮬레이션)
	err = sessionManager.Update(ctx, sessionID, SessionUpdate{
		State: &SessionState{
			Status:      "active",
			LastMessage: "Hello, Claude!",
			TurnCount:   1,
		},
	})
	require.NoError(t, err)

	// 4. 세션 목록 조회
	sessions, err := sessionManager.List(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(sessions), 1)

	foundSession := false
	for _, s := range sessions {
		if s.ID == sessionID {
			foundSession = true
			assert.Equal(t, "active", s.State.Status)
			break
		}
	}
	assert.True(t, foundSession, "Created session should be in list")

	// 5. 세션 종료
	err = sessionManager.Close(ctx, sessionID)
	require.NoError(t, err)

	// 6. 종료된 세션 상태 확인
	closedSession, err := sessionManager.Get(ctx, sessionID)
	require.NoError(t, err)
	assert.Equal(t, "closed", closedSession.State.Status)
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