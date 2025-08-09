package websocket

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

// TestStreamManagerCreation 스트림 관리자 생성 테스트
func TestStreamManagerCreation(t *testing.T) {
	config := DefaultStreamConfig()
	sm := NewStreamManager(config)
	
	assert.NotNil(t, sm)
	assert.NotNil(t, sm.connections)
	assert.NotNil(t, sm.sessions)
	assert.Equal(t, config.MaxConnections, sm.config.MaxConnections)
}

// TestWebSocketConnection WebSocket 연결 테스트
func TestWebSocketConnection(t *testing.T) {
	sm := NewStreamManager(nil)
	
	// 테스트 서버 생성
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}
		
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatalf("Failed to upgrade: %v", err)
			return
		}
		
		// 연결 생성
		conn, err := sm.CreateConnection(ws)
		if err != nil {
			t.Fatalf("Failed to create connection: %v", err)
			return
		}
		
		assert.NotNil(t, conn)
		assert.NotEmpty(t, conn.ID)
		
		// 테스트 메시지 전송
		testMsg := []byte("test message")
		conn.SendChan <- testMsg
	}))
	defer server.Close()
	
	// WebSocket 클라이언트 연결
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer ws.Close()
	
	// 메시지 수신
	_, message, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read message: %v", err)
	}
	
	assert.Equal(t, "test message", string(message))
}

// TestStreamSession 스트리밍 세션 테스트
func TestStreamSession(t *testing.T) {
	sm := NewStreamManager(nil)
	
	// 세션 생성
	session, err := sm.CreateSession("test-pty-session")
	assert.NoError(t, err)
	assert.NotNil(t, session)
	assert.NotEmpty(t, session.ID)
	assert.Equal(t, "test-pty-session", session.PTYSessionID)
	assert.True(t, session.active)
	
	// 세션으로 데이터 전송
	testData := []byte("test output")
	err = sm.SendToSession(session.ID, testData)
	assert.NoError(t, err)
	
	// 세션 종료
	err = sm.CloseSession(session.ID)
	assert.NoError(t, err)
	
	// 종료된 세션으로 전송 시도 (실패해야 함)
	err = sm.SendToSession(session.ID, testData)
	assert.Error(t, err)
}

// TestBackpressureController 백프레셔 제어기 테스트
func TestBackpressureController(t *testing.T) {
	bc := NewBackpressureController(10)
	
	// 초기 상태
	assert.False(t, bc.ShouldThrottle())
	
	// 메시지 대기 추가
	for i := 0; i < 15; i++ {
		bc.MessagePending()
	}
	
	// 스로틀링 확인
	assert.True(t, bc.ShouldThrottle())
	
	// 메시지 전송
	for i := 0; i < 10; i++ {
		bc.MessageSent()
	}
	
	// 스로틀링 해제 확인
	assert.False(t, bc.ShouldThrottle())
	
	// 통계 확인
	stats := bc.GetStats()
	assert.Equal(t, 5, stats["pending_count"])
	assert.Equal(t, uint64(10), stats["sent_count"])
}

// TestRateLimiter 레이트 리미터 테스트
func TestRateLimiter(t *testing.T) {
	rl := NewRateLimiter(10) // 초당 10개
	
	// 초기에는 허용되어야 함
	for i := 0; i < 10; i++ {
		assert.True(t, rl.Allow())
	}
	
	// 한도 초과 시 거부되어야 함
	assert.False(t, rl.Allow())
	
	// 잠시 대기 후 다시 허용되어야 함
	time.Sleep(200 * time.Millisecond)
	assert.True(t, rl.Allow())
	
	// 통계 확인
	stats := rl.GetStats()
	assert.Equal(t, uint64(11), stats["total_allowed"])
	assert.Equal(t, uint64(1), stats["total_denied"])
}

// TestMaxConnections 최대 연결 수 테스트
func TestMaxConnections(t *testing.T) {
	config := &StreamConfig{
		MaxConnections: 2,
		BufferSize:     1024,
		WriteTimeout:   10 * time.Second,
		ReadTimeout:    10 * time.Second,
		PingInterval:   30 * time.Second,
		MaxMessageSize: 4096,
	}
	
	sm := NewStreamManager(config)
	
	// 최대 연결 수만큼 생성
	conns := make([]*StreamConnection, 0)
	for i := 0; i < 2; i++ {
		// 모의 WebSocket 연결
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			upgrader := websocket.Upgrader{
				CheckOrigin: func(r *http.Request) bool { return true },
			}
			ws, _ := upgrader.Upgrade(w, r, nil)
			conn, _ := sm.CreateConnection(ws)
			conns = append(conns, conn)
		}))
		
		wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
		ws, _, _ := websocket.DefaultDialer.Dial(wsURL, nil)
		defer ws.Close()
		defer server.Close()
	}
	
	// 추가 연결 시도 (실패해야 함)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}
		ws, _ := upgrader.Upgrade(w, r, nil)
		_, err := sm.CreateConnection(ws)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "max connections reached")
	}))
	defer server.Close()
	
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, _, _ := websocket.DefaultDialer.Dial(wsURL, nil)
	defer ws.Close()
}

// TestConcurrentSessions 동시 세션 테스트
func TestConcurrentSessions(t *testing.T) {
	sm := NewStreamManager(nil)
	
	// 동시에 여러 세션 생성
	sessionCount := 10
	sessions := make([]*StreamSession, sessionCount)
	errCh := make(chan error, sessionCount)
	
	for i := 0; i < sessionCount; i++ {
		go func(idx int) {
			session, err := sm.CreateSession(fmt.Sprintf("pty-%d", idx))
			if err != nil {
				errCh <- err
				return
			}
			sessions[idx] = session
			errCh <- nil
		}(i)
	}
	
	// 결과 확인
	for i := 0; i < sessionCount; i++ {
		err := <-errCh
		assert.NoError(t, err)
	}
	
	// 모든 세션이 생성되었는지 확인
	stats := sm.GetStats()
	assert.Equal(t, sessionCount, stats["active_sessions"])
	
	// 동시에 메시지 전송
	for _, session := range sessions {
		if session != nil {
			go func(s *StreamSession) {
				err := sm.SendToSession(s.ID, []byte("concurrent test"))
				assert.NoError(t, err)
			}(session)
		}
	}
	
	// 동시에 세션 종료
	for _, session := range sessions {
		if session != nil {
			go func(s *StreamSession) {
				err := sm.CloseSession(s.ID)
				assert.NoError(t, err)
			}(session)
		}
	}
	
	// 잠시 대기
	time.Sleep(100 * time.Millisecond)
	
	// 모든 세션이 종료되었는지 확인
	stats = sm.GetStats()
	assert.Equal(t, 0, stats["active_sessions"])
}

// BenchmarkStreamManager 스트림 관리자 벤치마크
func BenchmarkStreamManager(b *testing.B) {
	sm := NewStreamManager(nil)
	
	b.Run("CreateSession", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			session, _ := sm.CreateSession(fmt.Sprintf("pty-%d", i))
			sm.CloseSession(session.ID)
		}
	})
	
	b.Run("SendToSession", func(b *testing.B) {
		session, _ := sm.CreateSession("bench-pty")
		defer sm.CloseSession(session.ID)
		
		data := []byte("benchmark data")
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			sm.SendToSession(session.ID, data)
		}
	})
}

// BenchmarkBackpressure 백프레셔 벤치마크
func BenchmarkBackpressure(b *testing.B) {
	bc := NewBackpressureController(100)
	
	b.Run("ShouldThrottle", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			bc.ShouldThrottle()
		}
	})
	
	b.Run("MessageFlow", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			bc.MessagePending()
			if !bc.ShouldThrottle() {
				bc.MessageSent()
			} else {
				bc.MessageDropped()
			}
		}
	})
}