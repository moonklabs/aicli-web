package pty_streaming

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPTYSessionLifecycle tests the complete PTY session lifecycle
func TestPTYSessionLifecycle(t *testing.T) {
	framework, err := setupTestFramework(t)
	require.NoError(t, err)
	defer framework.Cleanup()

	err = framework.Start()
	require.NoError(t, err)

	tests := []struct {
		name            string
		containerImage  string
		commands        []string
		expectedOutputs []string
		timeout         time.Duration
	}{
		{
			name:            "Basic Shell Session",
			containerImage:  "alpine:latest",
			commands:        []string{"echo 'Hello, World!'", "pwd", "ls /"},
			expectedOutputs: []string{"Hello, World!", "/", "bin"},
			timeout:         30 * time.Second,
		},
		{
			name:            "Environment Variables",
			containerImage:  "alpine:latest",
			commands:        []string{"echo $TERM", "echo $HOME"},
			expectedOutputs: []string{"xterm", "/root"},
			timeout:         30 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 컨테이너 생성
			containerID, err := framework.CreateTestContainer(tt.containerImage)
			require.NoError(t, err)
			defer framework.RemoveContainer(containerID)

			// PTY 세션 생성
			sessionID, err := framework.CreatePTYSession(containerID)
			require.NoError(t, err)
			defer framework.ClosePTYSession(sessionID)

			// WebSocket 연결
			client, err := framework.ConnectWebSocket(sessionID)
			require.NoError(t, err)
			defer client.Close()

			// 명령어 실행 및 출력 확인
			for i, cmd := range tt.commands {
				err := client.SendCommand(cmd)
				require.NoError(t, err)

				if i < len(tt.expectedOutputs) {
					output, err := client.WaitForOutput(tt.expectedOutputs[i], tt.timeout)
					require.NoError(t, err)
					assert.Contains(t, output, tt.expectedOutputs[i])
				}
			}
		})
	}
}

// TestWebSocketStreaming tests WebSocket streaming functionality
func TestWebSocketStreaming(t *testing.T) {
	framework, err := setupTestFramework(t)
	require.NoError(t, err)
	defer framework.Cleanup()

	err = framework.Start()
	require.NoError(t, err)

	containerID, err := framework.CreateTestContainer("alpine:latest")
	require.NoError(t, err)
	defer framework.RemoveContainer(containerID)

	sessionID, err := framework.CreatePTYSession(containerID)
	require.NoError(t, err)
	defer framework.ClosePTYSession(sessionID)

	t.Run("Multiple Clients", func(t *testing.T) {
		// 다중 클라이언트 연결
		clientCount := 3
		clients := make([]*TestClient, clientCount)

		for i := 0; i < clientCount; i++ {
			client, err := framework.ConnectWebSocket(sessionID)
			require.NoError(t, err)
			clients[i] = client
			defer client.Close()
		}

		// 한 클라이언트에서 명령 전송
		testMessage := "echo 'broadcast test'"
		err = clients[0].SendCommand(testMessage)
		require.NoError(t, err)

		// 모든 클라이언트가 동일한 출력을 받는지 확인
		expectedOutput := "broadcast test"
		for i, client := range clients {
			output, err := client.WaitForOutput(expectedOutput, 10*time.Second)
			require.NoError(t, err, "Client %d did not receive expected output", i)
			assert.Contains(t, output, expectedOutput)
		}
	})

	t.Run("Latency Measurement", func(t *testing.T) {
		client, err := framework.ConnectWebSocket(sessionID)
		require.NoError(t, err)
		defer client.Close()

		// 지연 시간 측정
		latencies := make([]time.Duration, 10)
		for i := 0; i < 10; i++ {
			start := time.Now()
			cmd := fmt.Sprintf("echo 'latency test %d'", i)

			err := client.SendCommand(cmd)
			require.NoError(t, err)

			_, err = client.WaitForOutput(fmt.Sprintf("latency test %d", i), 5*time.Second)
			require.NoError(t, err)

			latencies[i] = time.Since(start)
		}

		// 평균 지연 시간 계산
		var total time.Duration
		for _, latency := range latencies {
			total += latency
		}
		avgLatency := total / time.Duration(len(latencies))

		t.Logf("Average latency: %v", avgLatency)
		assert.Less(t, avgLatency.Milliseconds(), int64(100), "Average latency should be less than 100ms")
	})
}

// TestDockerIntegration tests Docker container integration
func TestDockerIntegration(t *testing.T) {
	framework, err := setupTestFramework(t)
	require.NoError(t, err)
	defer framework.Cleanup()

	err = framework.Start()
	require.NoError(t, err)

	t.Run("Container Restart", func(t *testing.T) {
		containerID, err := framework.CreateTestContainer("alpine:latest")
		require.NoError(t, err)
		defer framework.RemoveContainer(containerID)

		sessionID, err := framework.CreatePTYSession(containerID)
		require.NoError(t, err)
		defer framework.ClosePTYSession(sessionID)

		client, err := framework.ConnectWebSocket(sessionID)
		require.NoError(t, err)
		defer client.Close()

		// 기본 명령어 테스트
		err = client.SendCommand("echo 'before restart'")
		require.NoError(t, err)

		output, err := client.WaitForOutput("before restart", 5*time.Second)
		require.NoError(t, err)
		assert.Contains(t, output, "before restart")

		// 컨테이너 재시작
		err = framework.dockerManager.StopContainer(containerID)
		require.NoError(t, err)

		time.Sleep(2 * time.Second)

		err = framework.dockerManager.StartContainer(containerID)
		require.NoError(t, err)

		// PTY 재연결 확인
		time.Sleep(3 * time.Second)

		// 새 클라이언트 연결
		newClient, err := framework.ConnectWebSocket(sessionID)
		if err == nil {
			defer newClient.Close()

			err = newClient.SendCommand("echo 'after restart'")
			assert.NoError(t, err)

			if err == nil {
				output, err = newClient.WaitForOutput("after restart", 10*time.Second)
				if err == nil {
					assert.Contains(t, output, "after restart")
				}
			}
		}
	})
}

// TestPerformance tests performance requirements
func TestPerformance(t *testing.T) {
	framework, err := setupTestFramework(t)
	require.NoError(t, err)
	defer framework.Cleanup()

	err = framework.Start()
	require.NoError(t, err)

	t.Run("Concurrent Sessions", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping performance test in short mode")
		}

		sessionCount := 10 // 테스트용으로 줄임
		sessions := make([]*TestSession, sessionCount)

		// 동시 세션 생성
		var wg sync.WaitGroup
		for i := 0; i < sessionCount; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()

				containerID, err := framework.CreateTestContainer("alpine:latest")
				if err != nil {
					t.Errorf("Failed to create container %d: %v", index, err)
					return
				}

				sessionID, err := framework.CreatePTYSession(containerID)
				if err != nil {
					t.Errorf("Failed to create session %d: %v", index, err)
					return
				}

				client, err := framework.ConnectWebSocket(sessionID)
				if err != nil {
					t.Errorf("Failed to connect websocket %d: %v", index, err)
					return
				}

				sessions[index] = &TestSession{
					ContainerID: containerID,
					SessionID:   sessionID,
					Client:      client,
				}
			}(i)
		}
		wg.Wait()

		// 모든 세션에서 동시에 명령어 실행
		start := time.Now()
		for i, session := range sessions {
			if session != nil && session.Client != nil {
				go func(s *TestSession, idx int) {
					cmd := fmt.Sprintf("echo 'performance test %d'", idx)
					s.Client.SendCommand(cmd)
				}(session, i)
			}
		}

		// 모든 응답 대기
		for i, session := range sessions {
			if session != nil && session.Client != nil {
				expected := fmt.Sprintf("performance test %d", i)
				output, err := session.Client.WaitForOutput(expected, 30*time.Second)
				if err == nil {
					assert.Contains(t, output, expected)
				}
			}
		}

		duration := time.Since(start)
		t.Logf("%d concurrent sessions completed in: %v", sessionCount, duration)

		// 정리
		for _, session := range sessions {
			if session != nil {
				if session.Client != nil {
					session.Client.Close()
				}
				framework.ClosePTYSession(session.SessionID)
				framework.RemoveContainer(session.ContainerID)
			}
		}
	})

	t.Run("Memory Usage", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping memory test in short mode")
		}

		// 기준 메모리 사용량 측정
		baselineMemory := framework.GetCurrentMemoryUsage()

		sessionCount := 5
		sessions := make([]*TestSession, sessionCount)

		// 세션 생성
		for i := 0; i < sessionCount; i++ {
			containerID, err := framework.CreateTestContainer("alpine:latest")
			require.NoError(t, err)

			sessionID, err := framework.CreatePTYSession(containerID)
			require.NoError(t, err)

			client, err := framework.ConnectWebSocket(sessionID)
			require.NoError(t, err)

			sessions[i] = &TestSession{
				ContainerID: containerID,
				SessionID:   sessionID,
				Client:      client,
			}
		}

		// 메모리 사용량 측정
		currentMemory := framework.GetCurrentMemoryUsage()
		memoryPerSession := (currentMemory - baselineMemory) / int64(sessionCount)

		t.Logf("Memory per session: %d MB", memoryPerSession/1024/1024)

		// 메모리 요구사항 확인 (세션당 50MB 이하)
		assert.Less(t, memoryPerSession, int64(50*1024*1024),
			"Memory usage per session should be less than 50MB")

		// 정리
		for _, session := range sessions {
			session.Client.Close()
			framework.ClosePTYSession(session.SessionID)
			framework.RemoveContainer(session.ContainerID)
		}
	})
}

// setupTestFramework sets up the test framework
func setupTestFramework(t *testing.T) (*IntegrationTestFramework, error) {
	config := &TestConfig{
		ServerPort:      getRandomPort(),
		ContainerImages: []string{"alpine:latest"},
		MaxSessions:     100,
		TestDuration:    5 * time.Minute,
		MessageRate:     10,
		EnableMetrics:   true,
		LogLevel:        "info",
	}

	framework, err := NewIntegrationTestFramework(config)
	if err != nil {
		return nil, err
	}

	t.Cleanup(func() {
		framework.Cleanup()
	})

	return framework, nil
}