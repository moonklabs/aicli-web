// +build example

package claude

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/sijinhui/aicli-web/internal/storage/sqlite"
)

// SessionExample demonstrates how to use the session management system
func SessionExample() {
	// 1. 스토리지 초기화
	store, err := sqlite.New(":memory:")
	if err != nil {
		log.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	// 2. 프로세스 매니저 생성
	pmConfig := ProcessManagerConfig{
		MaxProcesses:     10,
		DefaultTimeout:   30 * time.Second,
		HealthCheckInterval: 10 * time.Second,
	}
	pm := NewProcessManager(pmConfig)

	// 3. 세션 매니저 생성
	sm := NewSessionManager(pm, store)

	// 4. 세션 풀 생성
	poolConfig := SessionPoolConfig{
		MaxSessions:     5,
		MaxIdleTime:     30 * time.Minute,
		MaxLifetime:     4 * time.Hour,
		CleanupInterval: 5 * time.Minute,
	}
	pool := NewSessionPool(sm, poolConfig)
	defer pool.Shutdown()

	// 5. 이벤트 리스너 설정
	eventLogger := NewSessionEventLogger(nil)
	sm.(*sessionManager).eventBus.Subscribe("", eventLogger)

	// 6. 세션 생성 및 사용
	ctx := context.Background()
	sessionConfig := SessionConfig{
		WorkingDir:   "/workspace",
		SystemPrompt: "You are a helpful assistant",
		MaxTurns:     50,
		Temperature:  0.7,
		AllowedTools: []string{"code_interpreter", "file_search"},
		Environment: map[string]string{
			"PROJECT_NAME": "example-project",
		},
		MaxMemory:   2 * 1024 * 1024 * 1024, // 2GB
		MaxCPU:      0.8,                     // 80%
		MaxDuration: 2 * time.Hour,
	}

	// 풀에서 세션 획득
	fmt.Println("=== Acquiring session from pool ===")
	pooledSession, err := pool.AcquireSession(ctx, sessionConfig)
	if err != nil {
		log.Fatalf("Failed to acquire session: %v", err)
	}
	fmt.Printf("Acquired session: %s (state: %s)\n", pooledSession.ID, pooledSession.State)

	// 세션 사용
	fmt.Println("\n=== Using session ===")
	// 여기서 실제 Claude CLI와 상호작용
	simulateSessionUsage(pooledSession)

	// 세션 반환
	fmt.Println("\n=== Releasing session back to pool ===")
	err = pool.ReleaseSession(pooledSession.ID)
	if err != nil {
		log.Printf("Failed to release session: %v", err)
	}

	// 풀 통계 확인
	stats := pool.GetPoolStats()
	fmt.Printf("\nPool stats: Total=%d, Active=%d, Idle=%d\n", 
		stats.TotalSessions, stats.ActiveSessions, stats.IdleSessions)

	// 7. 직접 세션 관리 예제
	fmt.Println("\n=== Direct session management ===")
	directSession, err := sm.CreateSession(ctx, sessionConfig)
	if err != nil {
		log.Fatalf("Failed to create direct session: %v", err)
	}
	fmt.Printf("Created direct session: %s\n", directSession.ID)

	// 세션 상태 변경
	activeState := SessionStateActive
	err = sm.UpdateSession(directSession.ID, SessionUpdate{
		State: &activeState,
		Metadata: map[string]interface{}{
			"task": "code_review",
			"start_time": time.Now(),
		},
	})
	if err != nil {
		log.Printf("Failed to update session: %v", err)
	}

	// 세션 목록 조회
	sessions, err := sm.ListSessions(SessionFilter{
		Active: &[]bool{true}[0],
	})
	if err != nil {
		log.Printf("Failed to list sessions: %v", err)
	}
	fmt.Printf("\nActive sessions: %d\n", len(sessions))

	// 세션 종료
	fmt.Println("\n=== Closing session ===")
	err = sm.CloseSession(directSession.ID)
	if err != nil {
		log.Printf("Failed to close session: %v", err)
	}

	// 8. 이벤트 기반 모니터링 예제
	fmt.Println("\n=== Event-based monitoring ===")
	monitoringCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 커스텀 이벤트 핸들러
	sm.(*sessionManager).eventBus.SubscribeToType(SessionEventStateChanged, func(event SessionEvent) {
		data := event.Data.(StateChangeData)
		fmt.Printf("[MONITOR] Session %s state changed: %s -> %s\n", 
			event.SessionID, data.OldState, data.NewState)
	})

	sm.(*sessionManager).eventBus.SubscribeToType(SessionEventError, func(event SessionEvent) {
		fmt.Printf("[MONITOR] Session %s error: %v\n", event.SessionID, event.Error)
	})

	// 모니터링 세션 생성
	monitorSession, err := sm.CreateSession(monitoringCtx, sessionConfig)
	if err != nil {
		log.Printf("Failed to create monitoring session: %v", err)
		return
	}

	// 여러 상태 전이 시뮬레이션
	states := []SessionState{
		SessionStateActive,
		SessionStateIdle,
		SessionStateSuspended,
		SessionStateReady,
		SessionStateClosing,
	}

	for _, state := range states {
		time.Sleep(500 * time.Millisecond)
		err = sm.UpdateSession(monitorSession.ID, SessionUpdate{
			State: &state,
		})
		if err != nil {
			log.Printf("Failed to update state to %s: %v", state, err)
		}
	}

	// 최종 정리
	sm.CloseSession(monitorSession.ID)
	time.Sleep(1 * time.Second) // 이벤트 처리 대기

	fmt.Println("\n=== Example completed ===")
}

// simulateSessionUsage simulates using a session
func simulateSessionUsage(session *PooledSession) {
	fmt.Printf("Using session %s for task execution...\n", session.ID)
	
	// 시뮬레이션: 여러 작업 수행
	tasks := []string{
		"Analyzing code structure",
		"Running tests",
		"Generating documentation",
		"Code review",
	}

	for i, task := range tasks {
		fmt.Printf("  [%d/%d] %s...\n", i+1, len(tasks), task)
		time.Sleep(500 * time.Millisecond)
		
		// 세션 활동 업데이트
		session.LastActive = time.Now()
	}
	
	fmt.Println("Task execution completed")
}

// ExampleSessionLifecycle demonstrates a typical session lifecycle
func ExampleSessionLifecycle() {
	ctx := context.Background()
	
	// 매니저 초기화
	pm := NewProcessManager(ProcessManagerConfig{})
	sm := NewSessionManager(pm, nil)
	
	// 세션 설정
	config := SessionConfig{
		WorkingDir:   "/workspace",
		SystemPrompt: "You are a code assistant",
		MaxTurns:     100,
		Temperature:  0.7,
		AllowedTools: []string{"code_interpreter"},
	}
	
	// 1. 세션 생성
	session, err := sm.CreateSession(ctx, config)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Session created: %s\n", session.ID)
	
	// 2. 세션 활성화
	activeState := SessionStateActive
	sm.UpdateSession(session.ID, SessionUpdate{State: &activeState})
	fmt.Println("Session activated")
	
	// 3. 작업 수행
	// ... Claude CLI와 상호작용 ...
	
	// 4. 유휴 상태로 전환
	idleState := SessionStateIdle
	sm.UpdateSession(session.ID, SessionUpdate{State: &idleState})
	fmt.Println("Session idle")
	
	// 5. 세션 종료
	sm.CloseSession(session.ID)
	fmt.Println("Session closed")
}

// ExampleSessionPoolUsage demonstrates session pool usage
func ExampleSessionPoolUsage() {
	ctx := context.Background()
	
	// 초기화
	pm := NewProcessManager(ProcessManagerConfig{})
	sm := NewSessionManager(pm, nil)
	pool := NewSessionPool(sm, DefaultSessionPoolConfig())
	defer pool.Shutdown()
	
	// 동일한 설정으로 여러 작업 수행
	config := SessionConfig{
		WorkingDir:  "/workspace",
		MaxTurns:    50,
		Temperature: 0.7,
	}
	
	// 작업 1
	session1, _ := pool.AcquireSession(ctx, config)
	fmt.Printf("Task 1 using session: %s\n", session1.ID)
	// ... 작업 수행 ...
	pool.ReleaseSession(session1.ID)
	
	// 작업 2 (같은 세션 재사용)
	session2, _ := pool.AcquireSession(ctx, config)
	fmt.Printf("Task 2 using session: %s (reused: %v)\n", 
		session2.ID, session2.ID == session1.ID)
	// ... 작업 수행 ...
	pool.ReleaseSession(session2.ID)
}