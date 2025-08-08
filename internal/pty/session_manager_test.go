package pty

import (
	"context"
	"sync"
	"testing"
	"time"
)

// TestSessionManagerCreation 세션 관리자 생성 테스트
func TestSessionManagerCreation(t *testing.T) {
	config := &SessionConfig{
		MaxSessions:     10,
		IdleTimeout:     5 * time.Minute,
		CleanupInterval: 30 * time.Second,
		EnablePooling:   true,
		PoolSize:        5,
	}

	sm := NewSessionManager(config)
	defer sm.Shutdown()

	if sm == nil {
		t.Fatal("Failed to create session manager")
	}

	if sm.config.MaxSessions != 10 {
		t.Errorf("Expected max sessions 10, got %d", sm.config.MaxSessions)
	}

	if sm.pool == nil {
		t.Error("Pool should be initialized when pooling is enabled")
	}
}

// TestSessionCreation 세션 생성 테스트
func TestSessionCreation(t *testing.T) {
	sm := NewSessionManager(nil)
	defer sm.Shutdown()

	ctx := context.Background()
	config := DefaultPTYConfig()

	session, err := sm.CreateSession(ctx, "test-container", config)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	if session == nil {
		t.Fatal("Session should not be nil")
	}

	if session.ContainerID != "test-container" {
		t.Errorf("Expected container ID 'test-container', got '%s'", session.ContainerID)
	}

	if session.Status != SessionActive {
		t.Errorf("Expected session status Active, got %s", session.Status)
	}

	// 세션 조회 테스트
	retrieved, err := sm.GetSession(session.ID)
	if err != nil {
		t.Fatalf("Failed to get session: %v", err)
	}

	if retrieved.ID != session.ID {
		t.Errorf("Retrieved session ID mismatch: expected %s, got %s", 
			session.ID, retrieved.ID)
	}
}

// TestMaxSessions 최대 세션 수 제한 테스트
func TestMaxSessions(t *testing.T) {
	config := &SessionConfig{
		MaxSessions:     3,
		CleanupInterval: 1 * time.Hour, // 자동 정리 비활성화
	}

	sm := NewSessionManager(config)
	defer sm.Shutdown()

	ctx := context.Background()
	ptyConfig := DefaultPTYConfig()

	// 최대 세션 수만큼 생성
	var sessions []*PTYSession
	for i := 0; i < 3; i++ {
		session, err := sm.CreateSession(ctx, "container", ptyConfig)
		if err != nil {
			t.Fatalf("Failed to create session %d: %v", i, err)
		}
		sessions = append(sessions, session)
	}

	// 추가 세션 생성 시도 (실패해야 함)
	_, err := sm.CreateSession(ctx, "container", ptyConfig)
	if err == nil {
		t.Error("Should fail when max sessions reached")
	}

	// 세션 하나 종료
	if err := sm.CloseSession(sessions[0].ID); err != nil {
		t.Fatalf("Failed to close session: %v", err)
	}

	// 이제 생성 가능해야 함
	newSession, err := sm.CreateSession(ctx, "container", ptyConfig)
	if err != nil {
		t.Fatalf("Should be able to create session after closing one: %v", err)
	}

	if newSession == nil {
		t.Error("New session should not be nil")
	}
}

// TestSessionCleanup 세션 정리 테스트
func TestSessionCleanup(t *testing.T) {
	config := &SessionConfig{
		MaxSessions:     10,
		IdleTimeout:     100 * time.Millisecond,
		CleanupInterval: 50 * time.Millisecond,
	}

	sm := NewSessionManager(config)
	defer sm.Shutdown()

	ctx := context.Background()
	ptyConfig := DefaultPTYConfig()

	// 세션 생성
	session, err := sm.CreateSession(ctx, "container", ptyConfig)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	sessionID := session.ID

	// 세션을 Idle 상태로 변경
	session.SetIdle()

	// 정리 대기
	time.Sleep(200 * time.Millisecond)

	// 세션이 정리되었는지 확인
	_, err = sm.GetSession(sessionID)
	if err == nil {
		t.Error("Session should be cleaned up after idle timeout")
	}
}

// TestConcurrentAccess 동시 접근 테스트
func TestConcurrentAccess(t *testing.T) {
	sm := NewSessionManager(nil)
	defer sm.Shutdown()

	ctx := context.Background()
	config := DefaultPTYConfig()

	var wg sync.WaitGroup
	sessionIDs := make([]string, 0, 10)
	mu := sync.Mutex{}

	// 동시에 10개 세션 생성
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			session, err := sm.CreateSession(ctx, "container", config)
			if err != nil {
				t.Errorf("Failed to create session %d: %v", idx, err)
				return
			}

			mu.Lock()
			sessionIDs = append(sessionIDs, session.ID)
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	// 모든 세션이 생성되었는지 확인
	if len(sessionIDs) != 10 {
		t.Errorf("Expected 10 sessions, got %d", len(sessionIDs))
	}

	// 동시에 세션 조회
	for _, id := range sessionIDs {
		wg.Add(1)
		go func(sessionID string) {
			defer wg.Done()

			if _, err := sm.GetSession(sessionID); err != nil {
				t.Errorf("Failed to get session %s: %v", sessionID, err)
			}
		}(id)
	}

	wg.Wait()

	// 동시에 세션 종료
	for _, id := range sessionIDs {
		wg.Add(1)
		go func(sessionID string) {
			defer wg.Done()

			if err := sm.CloseSession(sessionID); err != nil {
				t.Errorf("Failed to close session %s: %v", sessionID, err)
			}
		}(id)
	}

	wg.Wait()

	// 모든 세션이 종료되었는지 확인
	sessions := sm.ListSessions()
	if len(sessions) != 0 {
		t.Errorf("Expected 0 sessions after cleanup, got %d", len(sessions))
	}
}

// TestSessionPool 세션 풀 테스트
func TestSessionPool(t *testing.T) {
	config := &SessionConfig{
		MaxSessions:   10,
		EnablePooling: true,
		PoolSize:      5,
	}

	sm := NewSessionManager(config)
	defer sm.Shutdown()

	ctx := context.Background()
	ptyConfig := DefaultPTYConfig()

	// 세션 생성 및 종료
	session1, err := sm.CreateSession(ctx, "container1", ptyConfig)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	sessionID1 := session1.ID

	// 세션 종료 (풀에 반환되어야 함)
	if err := sm.CloseSession(sessionID1); err != nil {
		t.Fatalf("Failed to close session: %v", err)
	}

	// 새 세션 생성 (풀에서 재활용되어야 함)
	session2, err := sm.CreateSession(ctx, "container2", ptyConfig)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// 통계 확인
	stats := sm.GetStats()
	if recycled, ok := stats["total_recycled"].(uint64); ok {
		if recycled == 0 {
			t.Error("Session should be recycled from pool")
		}
	}

	// 세션 ID는 달라야 함 (재활용되더라도)
	if session2.ID == sessionID1 {
		t.Error("Recycled session should have different ID")
	}

	// 컨테이너 ID는 업데이트되어야 함
	if session2.ContainerID != "container2" {
		t.Errorf("Expected container ID 'container2', got '%s'", session2.ContainerID)
	}
}

// TestSessionActivity 세션 활동 업데이트 테스트
func TestSessionActivity(t *testing.T) {
	sm := NewSessionManager(nil)
	defer sm.Shutdown()

	ctx := context.Background()
	config := DefaultPTYConfig()

	session, err := sm.CreateSession(ctx, "container", config)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	initialActive := session.LastActive

	// 잠시 대기
	time.Sleep(100 * time.Millisecond)

	// 활동 업데이트
	if err := sm.UpdateSessionActivity(session.ID); err != nil {
		t.Fatalf("Failed to update session activity: %v", err)
	}

	// 시간이 업데이트되었는지 확인
	updatedSession, _ := sm.GetSession(session.ID)
	if !updatedSession.LastActive.After(initialActive) {
		t.Error("Last active time should be updated")
	}
}

// TestSessionStats 세션 통계 테스트
func TestSessionStats(t *testing.T) {
	sm := NewSessionManager(nil)
	defer sm.Shutdown()

	ctx := context.Background()
	config := DefaultPTYConfig()

	// 여러 세션 생성
	for i := 0; i < 3; i++ {
		_, err := sm.CreateSession(ctx, "container", config)
		if err != nil {
			t.Fatalf("Failed to create session %d: %v", i, err)
		}
	}

	stats := sm.GetStats()

	if active, ok := stats["active_sessions"].(int); ok {
		if active != 3 {
			t.Errorf("Expected 3 active sessions, got %d", active)
		}
	} else {
		t.Error("active_sessions stat not found")
	}

	if created, ok := stats["total_created"].(uint64); ok {
		if created != 3 {
			t.Errorf("Expected 3 total created sessions, got %d", created)
		}
	} else {
		t.Error("total_created stat not found")
	}
}

// BenchmarkSessionCreation 세션 생성 벤치마크
func BenchmarkSessionCreation(b *testing.B) {
	sm := NewSessionManager(nil)
	defer sm.Shutdown()

	ctx := context.Background()
	config := DefaultPTYConfig()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		session, err := sm.CreateSession(ctx, "container", config)
		if err != nil {
			b.Fatalf("Failed to create session: %v", err)
		}
		sm.CloseSession(session.ID)
	}
}

// BenchmarkConcurrentSessionAccess 동시 세션 접근 벤치마크
func BenchmarkConcurrentSessionAccess(b *testing.B) {
	sm := NewSessionManager(nil)
	defer sm.Shutdown()

	ctx := context.Background()
	config := DefaultPTYConfig()

	// 세션 미리 생성
	sessions := make([]*PTYSession, 10)
	for i := 0; i < 10; i++ {
		session, _ := sm.CreateSession(ctx, "container", config)
		sessions[i] = session
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			idx := time.Now().Nanosecond() % 10
			sm.GetSession(sessions[idx].ID)
		}
	})
}