---
task_id: TX06_S01_M03
task_name: Integration Tests
sprint_id: S01_M03
complexity: medium
priority: high
status: pending
created_at: 2025-07-21 23:00
---

# TX06_S01: Integration Tests

## 📋 작업 개요

Claude CLI 통합의 모든 컴포넌트에 대한 포괄적인 통합 테스트를 작성합니다. 실제 Claude CLI와의 상호작용을 테스트하고 전체 시스템의 안정성을 검증합니다.

## 🎯 작업 목표

1. 프로세스 관리 통합 테스트 작성
2. 스트림 처리 통합 테스트 구현
3. E2E 시나리오 테스트 개발
4. 성능 및 안정성 벤치마크

## 📝 상세 작업 내용

### 1. 테스트 환경 설정

```go
// internal/claude/testing/test_helpers.go
type TestEnvironment struct {
    TempDir      string
    MockClaude   *MockClaudeServer
    RealClaude   bool // 실제 Claude CLI 사용 여부
    TestData     *TestDataProvider
}

func NewTestEnvironment(t *testing.T) *TestEnvironment {
    env := &TestEnvironment{
        TempDir:    t.TempDir(),
        RealClaude: os.Getenv("TEST_REAL_CLAUDE") == "true",
    }
    
    if !env.RealClaude {
        // Mock Claude 서버 시작
        env.MockClaude = NewMockClaudeServer()
        t.Cleanup(env.MockClaude.Stop)
    }
    
    return env
}

// Mock Claude CLI
type MockClaudeServer struct {
    server   *httptest.Server
    responses map[string][]byte
    mu       sync.Mutex
}

func (m *MockClaudeServer) SetResponse(pattern string, response []byte) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.responses[pattern] = response
}
```

### 2. 프로세스 관리 통합 테스트

```go
// internal/claude/process_manager_integration_test.go
func TestProcessManagerIntegration(t *testing.T) {
    env := NewTestEnvironment(t)
    
    t.Run("프로세스 생성 및 종료", func(t *testing.T) {
        config := SessionConfig{
            WorkingDir:   env.TempDir,
            SystemPrompt: "You are a helpful assistant",
            MaxTurns:     5,
        }
        
        // 프로세스 생성
        pm := NewProcessManager()
        process, err := pm.CreateProcess(context.Background(), config)
        require.NoError(t, err)
        require.NotNil(t, process)
        
        // 상태 확인
        assert.Equal(t, ProcessStateRunning, process.State())
        
        // 프롬프트 전송
        err = process.SendPrompt("Hello, Claude!")
        require.NoError(t, err)
        
        // 응답 대기
        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()
        
        response := waitForResponse(t, ctx, process)
        assert.Contains(t, response, "Hello")
        
        // 프로세스 종료
        err = process.Close()
        require.NoError(t, err)
        assert.Equal(t, ProcessStateClosed, process.State())
    })
    
    t.Run("동시 다중 프로세스", func(t *testing.T) {
        const numProcesses = 5
        var wg sync.WaitGroup
        
        for i := 0; i < numProcesses; i++ {
            wg.Add(1)
            go func(id int) {
                defer wg.Done()
                
                config := SessionConfig{
                    WorkingDir: filepath.Join(env.TempDir, fmt.Sprintf("proc_%d", id)),
                }
                
                process, err := pm.CreateProcess(context.Background(), config)
                require.NoError(t, err)
                defer process.Close()
                
                // 각 프로세스에 다른 프롬프트
                prompt := fmt.Sprintf("Process %d: Calculate %d + %d", id, id*10, id*20)
                err = process.SendPrompt(prompt)
                require.NoError(t, err)
                
                // 응답 검증
                response := waitForResponse(t, context.Background(), process)
                expected := fmt.Sprintf("%d", id*30)
                assert.Contains(t, response, expected)
            }(i)
        }
        
        wg.Wait()
    })
}
```

### 3. 스트림 처리 통합 테스트

```go
// internal/claude/stream_integration_test.go
func TestStreamProcessingIntegration(t *testing.T) {
    env := NewTestEnvironment(t)
    
    t.Run("JSON 스트림 파싱", func(t *testing.T) {
        // 테스트 데이터 준비
        testStream := env.TestData.LoadStreamData("complex_response.jsonl")
        reader := bytes.NewReader(testStream)
        
        // 스트림 핸들러 생성
        handler := NewStreamHandler()
        ctx := context.Background()
        
        messages, err := handler.Stream(ctx, reader)
        require.NoError(t, err)
        
        // 메시지 수집
        var collected []Message
        for msg := range messages {
            collected = append(collected, msg)
        }
        
        // 검증
        assert.Greater(t, len(collected), 0)
        assertMessageTypes(t, collected, []string{"text", "tool_use", "text"})
    })
    
    t.Run("백프레셔 처리", func(t *testing.T) {
        // 빠른 생산자, 느린 소비자 시뮬레이션
        producer := make(chan []byte, 1000)
        
        // 대량 데이터 생성
        go func() {
            for i := 0; i < 10000; i++ {
                msg := fmt.Sprintf(`{"type":"text","content":"Message %d"}\n`, i)
                producer <- []byte(msg)
            }
            close(producer)
        }()
        
        // 스트림 처리 (느린 소비)
        handler := NewStreamHandler()
        handler.SetBufferSize(100) // 작은 버퍼
        
        processed := 0
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
        
        messages, _ := handler.StreamFromChannel(ctx, producer)
        for range messages {
            processed++
            time.Sleep(time.Millisecond) // 인위적 지연
        }
        
        // 백프레셔로 인한 드롭 확인
        assert.Less(t, processed, 10000)
        metrics := handler.GetMetrics()
        assert.Greater(t, metrics.BackpressureEvents, int64(0))
    })
}
```

### 4. E2E 시나리오 테스트

```go
// internal/claude/e2e_test.go
func TestE2EScenarios(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping E2E tests in short mode")
    }
    
    env := NewTestEnvironment(t)
    
    t.Run("코드 생성 시나리오", func(t *testing.T) {
        // API 서버 시작
        server := startTestAPIServer(t, env)
        defer server.Close()
        
        // 클라이언트 생성
        client := NewTestClient(server.URL)
        
        // 1. 세션 생성
        session, err := client.CreateSession(SessionConfig{
            SystemPrompt: "You are a code generator",
            Tools:        []string{"Write", "Read"},
        })
        require.NoError(t, err)
        
        // 2. 코드 생성 요청
        execution, err := client.Execute(session.ID, "Create a simple Go HTTP server")
        require.NoError(t, err)
        
        // 3. WebSocket 연결 및 스트림 수신
        ws, err := client.ConnectWebSocket(execution.WebSocketURL)
        require.NoError(t, err)
        defer ws.Close()
        
        // 4. 메시지 수집
        var messages []WebSocketMessage
        timeout := time.After(30 * time.Second)
        
        for {
            select {
            case msg := <-ws.Messages:
                messages = append(messages, msg)
                if msg.Type == "completion" {
                    goto done
                }
            case <-timeout:
                t.Fatal("Timeout waiting for completion")
            }
        }
    done:
        
        // 5. 결과 검증
        assert.Greater(t, len(messages), 5)
        assertContainsToolUse(t, messages, "Write")
        assertGeneratedCode(t, messages, "http.ListenAndServe")
    })
    
    t.Run("에러 복구 시나리오", func(t *testing.T) {
        wrapper := NewClaudeWrapper()
        
        // 1. 정상 세션 생성
        session, err := wrapper.CreateSession(context.Background(), SessionConfig{})
        require.NoError(t, err)
        
        // 2. 프로세스 강제 종료
        process := getProcessFromSession(session)
        process.cmd.Process.Kill()
        
        // 3. 복구 시도
        _, err = wrapper.Execute(context.Background(), session.ID, "Test after crash")
        require.NoError(t, err) // 자동 복구되어야 함
        
        // 4. 세션 상태 확인
        recovered, err := wrapper.GetSession(session.ID)
        require.NoError(t, err)
        assert.Equal(t, SessionStateReady, recovered.State)
    })
}
```

### 5. 성능 벤치마크

```go
// internal/claude/benchmark_test.go
func BenchmarkStreamProcessing(b *testing.B) {
    data := generateLargeStreamData(1024 * 1024) // 1MB
    
    b.Run("JSON파싱", func(b *testing.B) {
        b.SetBytes(int64(len(data)))
        b.ResetTimer()
        
        for i := 0; i < b.N; i++ {
            reader := bytes.NewReader(data)
            handler := NewStreamHandler()
            
            messages, _ := handler.Stream(context.Background(), reader)
            for range messages {
                // 소비
            }
        }
    })
    
    b.Run("동시스트림", func(b *testing.B) {
        b.RunParallel(func(pb *testing.PB) {
            for pb.Next() {
                reader := bytes.NewReader(data)
                handler := NewStreamHandler()
                
                messages, _ := handler.Stream(context.Background(), reader)
                for range messages {
                    // 소비
                }
            }
        })
    })
}

func BenchmarkSessionManagement(b *testing.B) {
    manager := NewSessionManager()
    
    b.Run("세션생성", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            session, _ := manager.CreateSession(context.Background(), SessionConfig{})
            manager.CloseSession(session.ID)
        }
    })
    
    b.Run("세션재사용", func(b *testing.B) {
        // 세션 풀 미리 채우기
        for i := 0; i < 10; i++ {
            manager.CreateSession(context.Background(), SessionConfig{})
        }
        
        b.ResetTimer()
        for i := 0; i < b.N; i++ {
            session, _ := manager.GetAvailableSession()
            manager.ReleaseSession(session.ID)
        }
    })
}
```

### 6. 테스트 유틸리티

```go
// internal/claude/testing/assertions.go
func assertMessageTypes(t *testing.T, messages []Message, expected []string) {
    require.Equal(t, len(expected), len(messages))
    for i, msg := range messages {
        assert.Equal(t, expected[i], msg.Type)
    }
}

func assertContainsToolUse(t *testing.T, messages []WebSocketMessage, toolName string) {
    for _, msg := range messages {
        if msg.Type == "claude_message" {
            if claudeMsg, ok := msg.Data.(Message); ok && claudeMsg.Type == "tool_use" {
                if claudeMsg.ToolName == toolName {
                    return
                }
            }
        }
    }
    t.Errorf("Expected tool use '%s' not found", toolName)
}

func waitForResponse(t *testing.T, ctx context.Context, process *Process) string {
    var response strings.Builder
    messages := process.StreamOutput(ctx)
    
    for msg := range messages {
        if msg.Type == "text" {
            response.WriteString(msg.Content)
        }
    }
    
    return response.String()
}
```

## ✅ 완료 조건

- [ ] 모든 통합 테스트 통과
- [ ] E2E 시나리오 커버리지 90%
- [ ] 성능 벤치마크 기준 충족
- [ ] 테스트 문서화 완료
- [ ] CI 파이프라인 통합
- [ ] 테스트 데이터 준비

## 🧪 테스트 전략

### 테스트 레벨
- 단위 테스트: 개별 컴포넌트
- 통합 테스트: 컴포넌트 간 상호작용
- E2E 테스트: 전체 시스템 플로우
- 성능 테스트: 부하 및 스트레스

### 테스트 환경
- Mock Claude: 빠른 테스트
- Real Claude: 정확성 검증
- Docker 환경: 격리된 테스트
- CI 환경: 자동화 테스트

## 📚 참고 자료

- Go testing 패키지
- testify 프레임워크
- httptest 패키지
- 벤치마크 best practices

## 🔄 의존성

- 모든 claude 패키지 컴포넌트
- testing/testify
- net/http/httptest
- 테스트 데이터 셋

## 💡 구현 힌트

1. 테스트 격리 철저히
2. 병렬 테스트 활용
3. 테스트 데이터 재사용
4. 타임아웃 적절히 설정
5. 실패 시 디버그 정보 충분히

## 🔧 기술 가이드

### 코드베이스 통합 포인트

1. **테스트 프레임워크**
   - 테스트 유틸: `test/utils/`
   - 통합 테스트 디렉토리: `test/integration/`
   - E2E 테스트: `test/e2e/`
   - 테스트 픽스처: `test/fixtures/`

2. **기존 테스트 패턴**
   - Mock 프로세스: `internal/claude/process_manager_test.go`
   - 스트림 테스트: `internal/claude/stream_integration_test.go`
   - API 테스트: `internal/server/handlers/*_test.go`

3. **테스트 도구**
   - testify: assertion 및 mock
   - httptest: HTTP 테스트
   - testcontainers: Docker 통합 테스트

4. **CI/CD 통합**
   - GitHub Actions: `.github/workflows/test.yml`
   - 테스트 커버리지: `make test-coverage`

### 구현 접근법

1. **프로세스 관리 통합 테스트**
   - 새 파일: `test/integration/process_test.go`
   - 실제 Claude CLI 바이너리 사용
   - 프로세스 생명주기 테스트

2. **스트림 처리 통합 테스트**
   - 새 파일: `test/integration/stream_test.go`
   - 대용량 데이터 처리
   - 백프레셔 시나리오

3. **E2E 시나리오 테스트**
   - 새 파일: `test/e2e/claude_workflow_test.go`
   - CLI → API → Claude 전체 플로우
   - 실제 사용 시나리오

4. **테스트 헬퍼 및 유틸리티**
   - 테스트 서버 설정
   - Mock Claude CLI
   - 테스트 데이터 생성

### 테스트 접근법

1. **통합 테스트 전략**
   - 독립적인 테스트 환경
   - 테스트 데이터 격리
   - 병렬 실행 가능

2. **테스트 커버리지**
   - 핵심 경로 커버리지 90% 이상
   - 에러 처리 경로 포함
   - 경계 조건 테스트

3. **CI/CD 통합**
   - PR마다 자동 실행
   - 테스트 실패 시 머지 블록
   - 테스트 리포트 생성