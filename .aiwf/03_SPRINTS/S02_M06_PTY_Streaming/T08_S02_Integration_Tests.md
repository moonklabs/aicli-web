---
task_id: T08_S02_Integration_Tests
sprint_id: S02_M06_PTY_Streaming
milestone_id: M06
title: PTY 스트리밍 시스템 통합 테스트 및 검증
type: testing
complexity: Medium
status: pending
assignee: unassigned
created: 2025-08-05T10:00:00+0900
last_updated: 2025-08-05T10:00:00+0900
depends_on: [T04_S02_Docker_PTY_Integration, T07_S02_Performance_Optimization]
blocks: [T09_S02_Documentation]
epic: PTY_Streaming_System
---

# Task: PTY 스트리밍 시스템 통합 테스트 및 검증

## Task Summary
PTY 스트리밍 시스템의 모든 컴포넌트가 통합되어 정상적으로 동작하는지 검증하는 포괄적인 테스트 시스템을 구현합니다. 단위 테스트, 통합 테스트, 성능 테스트, 안정성 테스트를 포함한 완전한 테스트 스위트를 제공합니다.

## Acceptance Criteria

### 기능 테스트 요구사항
- [ ] PTY 세션 생성부터 종료까지 전체 플로우 테스트
- [ ] WebSocket 연결 및 실시간 스트리밍 테스트
- [ ] Docker 컨테이너 연동 테스트
- [ ] ANSI 파싱 및 터미널 스냅샷 테스트
- [ ] 플로우 제어 및 백프레셔 처리 테스트
- [ ] 에러 시나리오 및 복구 테스트

### 성능 테스트 요구사항
- [ ] 100개 동시 PTY 세션 처리 검증
- [ ] 응답 시간 < 50ms 달성 확인
- [ ] 메모리 사용량 < 50MB/세션 확인
- [ ] CPU 사용률 < 20% 유지 확인
- [ ] 8시간 연속 실행 안정성 검증

### 신뢰성 테스트 요구사항
- [ ] 네트워크 단절 시나리오 테스트
- [ ] 컨테이너 재시작 시나리오 테스트
- [ ] 메모리 부족 상황 테스트
- [ ] 높은 부하 상황에서의 복구 테스트
- [ ] 데이터 무결성 검증

## Implementation Details

### 1. 통합 테스트 프레임워크

```go
// test/integration/framework.go
type IntegrationTestFramework struct {
    testServer    *TestServer
    dockerManager *TestDockerManager
    clients       map[string]*TestClient
    metrics       *TestMetrics
    config        *TestConfig
    cleanup       []func() error
}

type TestServer struct {
    server       *http.Server
    ptyManager   *pty.SessionManager
    wsManager    *websocket.StreamManager
    dockerPTY    *docker.PTYIntegration
    flowControl  *flow.FlowController
    port         int
    running      bool
}

type TestClient struct {
    clientID     string
    wsConn       *websocket.Conn
    sessionID    string
    containerID  string
    connected    bool
    lastMessage  time.Time
    messageCount int64
    errorCount   int64
}

type TestConfig struct {
    ServerPort        int
    ContainerImages   []string
    MaxSessions       int
    TestDuration      time.Duration
    MessageRate       int
    EnableMetrics     bool
    LogLevel          string
}

// 테스트 프레임워크 초기화
func NewIntegrationTestFramework(config *TestConfig) (*IntegrationTestFramework, error) {
    framework := &IntegrationTestFramework{
        clients: make(map[string]*TestClient),
        config:  config,
        cleanup: make([]func() error, 0),
    }
    
    // 테스트 서버 설정
    if err := framework.setupTestServer(); err != nil {
        return nil, fmt.Errorf("failed to setup test server: %w", err)
    }
    
    // Docker 테스트 환경 설정
    if err := framework.setupDockerEnvironment(); err != nil {
        return nil, fmt.Errorf("failed to setup docker environment: %w", err)
    }
    
    // 메트릭 수집 설정
    if config.EnableMetrics {
        framework.setupMetrics()
    }
    
    return framework, nil
}

func (itf *IntegrationTestFramework) setupTestServer() error {
    // PTY 매니저 초기화
    ptyManager, err := pty.NewSessionManager(&pty.SessionConfig{
        MaxSessions:     itf.config.MaxSessions,
        CleanupInterval: 30 * time.Second,
    })
    if err != nil {
        return err
    }
    
    // WebSocket 매니저 초기화
    wsManager, err := websocket.NewStreamManager(&websocket.StreamConfig{
        BufferSize:      4096,
        MaxConnections:  itf.config.MaxSessions,
        HeartbeatInterval: 30 * time.Second,
    })
    if err != nil {
        return err
    }
    
    // Docker PTY 통합 초기화
    dockerPTY, err := docker.NewPTYIntegration(&docker.DockerPTYConfig{
        MaxConnections: itf.config.MaxSessions,
        MonitorInterval: 5 * time.Second,
    })
    if err != nil {
        return err
    }
    
    // 플로우 제어 초기화
    flowControl, err := flow.NewFlowController(&flow.FlowControlConfig{
        MaxBufferSize: 1024 * 1024, // 1MB
        ThrottleThreshold: 0.8,
    })
    if err != nil {
        return err
    }
    
    itf.testServer = &TestServer{
        ptyManager:   ptyManager,
        wsManager:    wsManager,
        dockerPTY:    dockerPTY,
        flowControl:  flowControl,
        port:         itf.config.ServerPort,
    }
    
    return nil
}
```

### 2. PTY 세션 통합 테스트

```go
// test/integration/pty_session_test.go
func TestPTYSessionLifecycle(t *testing.T) {
    framework, err := setupTestFramework(t)
    require.NoError(t, err)
    defer framework.Cleanup()
    
    tests := []struct {
        name          string
        containerImage string
        commands      []string
        expectedOutputs []string
        timeout       time.Duration
    }{
        {
            name:           "Basic Shell Session",
            containerImage: "ubuntu:20.04",
            commands:       []string{"echo 'Hello, World!'", "pwd", "ls -la"},
            expectedOutputs: []string{"Hello, World!", "/", "total"},
            timeout:        30 * time.Second,
        },
        {
            name:           "Interactive Python Session",
            containerImage: "python:3.9",
            commands:       []string{"python", "print('Hello, Python!')", "exit()"},
            expectedOutputs: []string{">>>", "Hello, Python!"},
            timeout:        60 * time.Second,
        },
        {
            name:           "Long Running Process",
            containerImage: "ubuntu:20.04",
            commands:       []string{"ping -c 5 localhost"},
            expectedOutputs: []string{"PING localhost", "5 packets transmitted"},
            timeout:        30 * time.Second,
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
            
            // 세션 정리
            err = framework.ClosePTYSession(sessionID)
            require.NoError(t, err)
        })
    }
}

// WebSocket 스트리밍 테스트
func TestWebSocketStreaming(t *testing.T) {
    framework, err := setupTestFramework(t)
    require.NoError(t, err)
    defer framework.Cleanup()
    
    containerID, err := framework.CreateTestContainer("ubuntu:20.04")
    require.NoError(t, err)
    defer framework.RemoveContainer(containerID)
    
    sessionID, err := framework.CreatePTYSession(containerID)
    require.NoError(t, err)
    defer framework.ClosePTYSession(sessionID)
    
    // 다중 클라이언트 연결 테스트
    clientCount := 5
    clients := make([]*TestClient, clientCount)
    
    for i := 0; i < clientCount; i++ {
        client, err := framework.ConnectWebSocket(sessionID)
        require.NoError(t, err)
        clients[i] = client
        defer client.Close()
    }
    
    // 브로드캐스트 테스트
    testMessage := "echo 'Broadcast test message'"
    err = clients[0].SendCommand(testMessage)
    require.NoError(t, err)
    
    // 모든 클라이언트가 동일한 출력을 받는지 확인
    expectedOutput := "Broadcast test message"
    for i, client := range clients {
        output, err := client.WaitForOutput(expectedOutput, 10*time.Second)
        require.NoError(t, err, "Client %d did not receive expected output", i)
        assert.Contains(t, output, expectedOutput)
    }
    
    // 지연 시간 측정
    latencies := framework.MeasureLatencies(clients, 100) // 100개 메시지
    avgLatency := calculateAverageLatency(latencies)
    assert.Less(t, avgLatency.Milliseconds(), int64(100), "Average latency should be less than 100ms")
}
```

### 3. Docker 통합 테스트

```go
// test/integration/docker_integration_test.go
func TestDockerPTYIntegration(t *testing.T) {
    framework, err := setupTestFramework(t)
    require.NoError(t, err)
    defer framework.Cleanup()
    
    t.Run("Container Lifecycle", func(t *testing.T) {
        // 컨테이너 생성 및 PTY 연결
        containerID, err := framework.CreateTestContainer("alpine:latest")
        require.NoError(t, err)
        
        sessionID, err := framework.CreatePTYSession(containerID)
        require.NoError(t, err)
        
        client, err := framework.ConnectWebSocket(sessionID)
        require.NoError(t, err)
        defer client.Close()
        
        // 기본 명령어 테스트
        err = client.SendCommand("whoami")
        require.NoError(t, err)
        
        output, err := client.WaitForOutput("root", 5*time.Second)
        require.NoError(t, err)
        assert.Contains(t, output, "root")
        
        // 컨테이너 중지 및 재시작 테스트
        err = framework.StopContainer(containerID)
        require.NoError(t, err)
        
        err = framework.StartContainer(containerID)
        require.NoError(t, err)
        
        // PTY 재연결 확인
        time.Sleep(2 * time.Second) // 재연결 대기
        
        err = client.SendCommand("echo 'reconnected'")
        require.NoError(t, err)
        
        output, err = client.WaitForOutput("reconnected", 10*time.Second)
        require.NoError(t, err)
        assert.Contains(t, output, "reconnected")
        
        // 정리
        framework.RemoveContainer(containerID)
    })
    
    t.Run("Multiple Container Types", func(t *testing.T) {
        images := []string{
            "ubuntu:20.04",
            "python:3.9-slim",
            "node:16-alpine",
            "golang:1.19-alpine",
        }
        
        for _, image := range images {
            t.Run(image, func(t *testing.T) {
                containerID, err := framework.CreateTestContainer(image)
                require.NoError(t, err)
                defer framework.RemoveContainer(containerID)
                
                sessionID, err := framework.CreatePTYSession(containerID)
                require.NoError(t, err)
                defer framework.ClosePTYSession(sessionID)
                
                client, err := framework.ConnectWebSocket(sessionID)
                require.NoError(t, err)
                defer client.Close()
                
                // 기본 명령어가 각 이미지에서 작동하는지 확인
                err = client.SendCommand("echo 'test'")
                require.NoError(t, err)
                
                output, err := client.WaitForOutput("test", 10*time.Second)
                require.NoError(t, err)
                assert.Contains(t, output, "test")
            })
        }
    })
}

// 리소스 모니터링 테스트
func TestResourceMonitoring(t *testing.T) {
    framework, err := setupTestFramework(t)
    require.NoError(t, err)
    defer framework.Cleanup()
    
    containerID, err := framework.CreateTestContainer("ubuntu:20.04")
    require.NoError(t, err)
    defer framework.RemoveContainer(containerID)
    
    sessionID, err := framework.CreatePTYSession(containerID)
    require.NoError(t, err)
    defer framework.ClosePTYSession(sessionID)
    
    // 리소스 사용량 모니터링 시작
    monitor := framework.StartResourceMonitoring(containerID)
    defer monitor.Stop()
    
    client, err := framework.ConnectWebSocket(sessionID)
    require.NoError(t, err)
    defer client.Close()
    
    // CPU 집약적 작업 실행
    err = client.SendCommand("yes > /dev/null &")
    require.NoError(t, err)
    
    time.Sleep(5 * time.Second)
    
    // CPU 사용량 확인
    stats := monitor.GetLatestStats()
    assert.Greater(t, stats.CPUUsage, 50.0, "CPU usage should increase during intensive task")
    
    // 작업 종료
    err = client.SendCommand("killall yes")
    require.NoError(t, err)
    
    time.Sleep(2 * time.Second)
    
    // CPU 사용량 정상화 확인
    stats = monitor.GetLatestStats()
    assert.Less(t, stats.CPUUsage, 10.0, "CPU usage should normalize after task completion")
}
```

### 4. 성능 및 부하 테스트

```go
// test/integration/performance_test.go
func TestPerformanceRequirements(t *testing.T) {
    framework, err := setupTestFramework(t)
    require.NoError(t, err)
    defer framework.Cleanup()
    
    t.Run("Concurrent Sessions", func(t *testing.T) {
        sessionCount := 100
        sessions := make([]*TestSession, sessionCount)
        
        // 동시 세션 생성
        var wg sync.WaitGroup
        for i := 0; i < sessionCount; i++ {
            wg.Add(1)
            go func(index int) {
                defer wg.Done()
                
                containerID, err := framework.CreateTestContainer("alpine:latest")
                require.NoError(t, err)
                
                sessionID, err := framework.CreatePTYSession(containerID)
                require.NoError(t, err)
                
                client, err := framework.ConnectWebSocket(sessionID)
                require.NoError(t, err)
                
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
        for _, session := range sessions {
            go func(s *TestSession) {
                s.Client.SendCommand("echo 'performance test'")
            }(session)
        }
        
        // 모든 응답 대기
        for _, session := range sessions {
            output, err := session.Client.WaitForOutput("performance test", 30*time.Second)
            require.NoError(t, err)
            assert.Contains(t, output, "performance test")
        }
        
        duration := time.Since(start)
        t.Logf("100 concurrent sessions completed in: %v", duration)
        
        // 응답 시간 요구사항 확인
        assert.Less(t, duration.Seconds(), 10.0, "All sessions should respond within 10 seconds")
        
        // 정리
        for _, session := range sessions {
            session.Client.Close()
            framework.ClosePTYSession(session.SessionID)
            framework.RemoveContainer(session.ContainerID)
        }
    })
    
    t.Run("Memory Usage", func(t *testing.T) {
        // 기준 메모리 사용량 측정
        baselineMemory := framework.GetCurrentMemoryUsage()
        
        sessionCount := 50
        sessions := make([]*TestSession, sessionCount)
        
        // 세션 생성
        for i := 0; i < sessionCount; i++ {
            containerID, err := framework.CreateTestContainer("ubuntu:20.04")
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
        
        // 메모리 해제 확인
        runtime.GC()
        time.Sleep(2 * time.Second)
        
        finalMemory := framework.GetCurrentMemoryUsage()
        memoryLeak := finalMemory - baselineMemory
        
        // 메모리 누수 확인 (10MB 이하의 차이는 허용)
        assert.Less(t, memoryLeak, int64(10*1024*1024), 
            "Memory leak should be less than 10MB")
    })
}

// 안정성 테스트
func TestReliability(t *testing.T) {
    framework, err := setupTestFramework(t)
    require.NoError(t, err)
    defer framework.Cleanup()
    
    t.Run("Long Running Stability", func(t *testing.T) {
        if testing.Short() {
            t.Skip("Skipping long running test in short mode")
        }
        
        containerID, err := framework.CreateTestContainer("ubuntu:20.04")
        require.NoError(t, err)
        defer framework.RemoveContainer(containerID)
        
        sessionID, err := framework.CreatePTYSession(containerID)
        require.NoError(t, err)
        defer framework.ClosePTYSession(sessionID)
        
        client, err := framework.ConnectWebSocket(sessionID)
        require.NoError(t, err)
        defer client.Close()
        
        // 8시간 테스트 (실제로는 1분으로 단축)
        testDuration := 1 * time.Minute
        ticker := time.NewTicker(5 * time.Second)
        defer ticker.Stop()
        
        start := time.Now()
        commandCount := 0
        
        for time.Since(start) < testDuration {
            select {
            case <-ticker.C:
                cmd := fmt.Sprintf("echo 'stability test %d'", commandCount)
                err := client.SendCommand(cmd)
                require.NoError(t, err)
                
                expectedOutput := fmt.Sprintf("stability test %d", commandCount)
                output, err := client.WaitForOutput(expectedOutput, 10*time.Second)
                require.NoError(t, err)
                assert.Contains(t, output, expectedOutput)
                
                commandCount++
            }
        }
        
        t.Logf("Completed %d commands over %v", commandCount, testDuration)
        assert.Greater(t, commandCount, 5, "Should have completed multiple commands")
    })
    
    t.Run("Network Disconnection Recovery", func(t *testing.T) {
        containerID, err := framework.CreateTestContainer("ubuntu:20.04")
        require.NoError(t, err)
        defer framework.RemoveContainer(containerID)
        
        sessionID, err := framework.CreatePTYSession(containerID)
        require.NoError(t, err)
        defer framework.ClosePTYSession(sessionID)
        
        client, err := framework.ConnectWebSocket(sessionID)
        require.NoError(t, err)
        
        // 정상 동작 확인
        err = client.SendCommand("echo 'before disconnect'")
        require.NoError(t, err)
        
        output, err := client.WaitForOutput("before disconnect", 5*time.Second)
        require.NoError(t, err)
        assert.Contains(t, output, "before disconnect")
        
        // 연결 강제 종료
        client.ForceDisconnect()
        time.Sleep(1 * time.Second)
        
        // 재연결
        client, err = framework.ConnectWebSocket(sessionID)
        require.NoError(t, err)
        defer client.Close()
        
        // 재연결 후 정상 동작 확인
        err = client.SendCommand("echo 'after reconnect'")
        require.NoError(t, err)
        
        output, err = client.WaitForOutput("after reconnect", 10*time.Second)
        require.NoError(t, err)
        assert.Contains(t, output, "after reconnect")
    })
}
```

### 5. 테스트 유틸리티 및 헬퍼

```go
// test/integration/test_utils.go
type TestSession struct {
    ContainerID string
    SessionID   string
    Client      *TestClient
}

type TestMetrics struct {
    TotalSessions    int64
    ActiveSessions   int64
    TotalMessages    int64
    ErrorCount       int64
    AverageLatency   time.Duration
    MemoryUsage      int64
    CPUUsage         float64
    startTime        time.Time
    mutex            sync.RWMutex
}

func (tm *TestMetrics) RecordMessage(latency time.Duration) {
    tm.mutex.Lock()
    defer tm.mutex.Unlock()
    
    tm.TotalMessages++
    
    // 이동 평균으로 지연 시간 계산
    if tm.AverageLatency == 0 {
        tm.AverageLatency = latency
    } else {
        tm.AverageLatency = (tm.AverageLatency + latency) / 2
    }
}

func (tm *TestMetrics) RecordError() {
    atomic.AddInt64(&tm.ErrorCount, 1)
}

func (tm *TestMetrics) GetReport() *TestReport {
    tm.mutex.RLock()
    defer tm.mutex.RUnlock()
    
    duration := time.Since(tm.startTime)
    throughput := float64(tm.TotalMessages) / duration.Seconds()
    
    return &TestReport{
        Duration:       duration,
        TotalSessions:  tm.TotalSessions,
        TotalMessages:  tm.TotalMessages,
        ErrorCount:     tm.ErrorCount,
        ErrorRate:      float64(tm.ErrorCount) / float64(tm.TotalMessages),
        AverageLatency: tm.AverageLatency,
        Throughput:     throughput,
        MemoryUsage:    tm.MemoryUsage,
        CPUUsage:       tm.CPUUsage,
    }
}

type TestReport struct {
    Duration       time.Duration
    TotalSessions  int64
    TotalMessages  int64
    ErrorCount     int64
    ErrorRate      float64
    AverageLatency time.Duration
    Throughput     float64
    MemoryUsage    int64
    CPUUsage       float64
}

// 테스트 환경 설정 헬퍼
func setupTestFramework(t *testing.T) (*IntegrationTestFramework, error) {
    config := &TestConfig{
        ServerPort:      getRandomPort(),
        ContainerImages: []string{"ubuntu:20.04", "alpine:latest"},
        MaxSessions:     100,
        TestDuration:    5 * time.Minute,
        MessageRate:     10, // messages per second
        EnableMetrics:   true,
        LogLevel:        "info",
    }
    
    framework, err := NewIntegrationTestFramework(config)
    if err != nil {
        return nil, err
    }
    
    // 테스트 종료 시 정리 작업 등록
    t.Cleanup(func() {
        framework.Cleanup()
    })
    
    return framework, nil
}

func getRandomPort() int {
    listener, err := net.Listen("tcp", ":0")
    if err != nil {
        return 8080
    }
    defer listener.Close()
    
    return listener.Addr().(*net.TCPAddr).Port
}
```

## 파일 구조

```
test/integration/
├── framework.go           # 통합 테스트 프레임워크
├── pty_session_test.go    # PTY 세션 테스트
├── websocket_test.go      # WebSocket 스트리밍 테스트  
├── docker_integration_test.go # Docker 통합 테스트
├── performance_test.go    # 성능 및 부하 테스트
├── reliability_test.go    # 안정성 테스트
├── test_utils.go          # 테스트 유틸리티
├── test_client.go         # 테스트 클라이언트
├── docker_manager.go      # Docker 테스트 관리자
└── metrics.go             # 테스트 메트릭

test/unit/
├── pty/                   # PTY 관련 단위 테스트
├── websocket/             # WebSocket 관련 단위 테스트
├── docker/                # Docker 관련 단위 테스트
├── ansi/                  # ANSI 파서 단위 테스트
├── flow/                  # 플로우 제어 단위 테스트
└── performance/           # 성능 관련 단위 테스트

test/fixtures/
├── docker/
│   ├── Dockerfile.test-ubuntu
│   ├── Dockerfile.test-python
│   └── Dockerfile.test-node
├── ansi/
│   ├── color_sequences.txt
│   ├── cursor_movements.txt
│   └── complex_output.txt
└── websocket/
    ├── test_messages.json
    └── stress_test_data.bin
```

## CI/CD 통합

```yaml
# .github/workflows/integration-tests.yml
name: Integration Tests

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

jobs:
  integration-tests:
    runs-on: ubuntu-latest
    
    services:
      docker:
        image: docker:dind
        options: --privileged
        
    steps:
    - uses: actions/checkout@v3
    
    - name: Set up Go
      uses: actions/setup-go@v3
      with:
        go-version: 1.19
        
    - name: Start Docker daemon
      run: |
        sudo dockerd &
        sleep 10
        
    - name: Pull test images
      run: |
        docker pull ubuntu:20.04
        docker pull alpine:latest
        docker pull python:3.9-slim
        
    - name: Run integration tests
      run: |
        go test -v ./test/integration/... -timeout=30m
        
    - name: Run performance tests
      run: |
        go test -v ./test/integration/performance_test.go -timeout=60m
        
    - name: Generate test report
      run: |
        go test -v ./test/integration/... -json > test-report.json
        
    - name: Upload test artifacts
      uses: actions/upload-artifact@v3
      if: always()
      with:
        name: test-results
        path: |
          test-report.json
          coverage.out
```

## Definition of Done
- [ ] 통합 테스트 프레임워크 구현 완료
- [ ] PTY 세션 전체 라이프사이클 테스트 완료
- [ ] WebSocket 스트리밍 테스트 완료
- [ ] Docker 통합 테스트 완료
- [ ] 성능 및 부하 테스트 완료
- [ ] 안정성 및 복구 테스트 완료
- [ ] 모든 테스트 통과 (성공률 > 99%)
- [ ] CI/CD 파이프라인 통합 완료
- [ ] 테스트 커버리지 > 85% 달성
- [ ] 테스트 문서화 완료

## Notes
- 테스트는 실제 Docker 환경에서 실행되므로 CI/CD 환경 설정 중요
- 성능 테스트는 하드웨어 환경에 따라 결과가 달라질 수 있음
- 장시간 실행 테스트는 선택적으로 실행할 수 있도록 구성
- 테스트 데이터와 컨테이너는 테스트 종료 시 완전히 정리되어야 함