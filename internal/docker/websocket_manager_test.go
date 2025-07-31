package docker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockPTYSessionManagement PTY 세션 관리 모의 객체
type MockPTYSessionManagement struct {
	mock.Mock
	sessions map[string]PTYSession
	mu       sync.RWMutex
}

func NewMockPTYSessionManagement() *MockPTYSessionManagement {
	return &MockPTYSessionManagement{
		sessions: make(map[string]PTYSession),
	}
}

func (m *MockPTYSessionManagement) CreateSession(containerID string, config PTYConfig) (PTYSession, error) {
	args := m.Called(containerID, config)
	return args.Get(0).(PTYSession), args.Error(1)
}

func (m *MockPTYSessionManagement) GetSession(sessionID string) (PTYSession, error) {
	args := m.Called(sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(PTYSession), args.Error(1)
}

func (m *MockPTYSessionManagement) ListSessions() []PTYSession {
	args := m.Called()
	return args.Get(0).([]PTYSession)
}

func (m *MockPTYSessionManagement) CloseSession(sessionID string) error {
	args := m.Called(sessionID)
	return args.Error(0)
}

func (m *MockPTYSessionManagement) AddMockSession(sessionID string, session PTYSession) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[sessionID] = session
}

// MockContainerPTYManagement 컨테이너 PTY 관리 모의 객체
type MockContainerPTYManagement struct {
	mock.Mock
}

func (m *MockContainerPTYManagement) AttachToPTY(containerID string, config PTYConfig) (PTYSession, error) {
	args := m.Called(containerID, config)
	return args.Get(0).(PTYSession), args.Error(1)
}

func (m *MockContainerPTYManagement) CreateContainerPTY(opts ContainerPTYOptions) (PTYSession, error) {
	args := m.Called(opts)
	return args.Get(0).(PTYSession), args.Error(1)
}

// TestWebSocketManager_Creation WebSocket 관리자 생성 테스트
func TestWebSocketManager_Creation(t *testing.T) {
	mockPTYManager := NewMockPTYSessionManagement()
	maxConnections := 100

	manager := NewWebSocketManager(mockPTYManager, maxConnections)

	assert.NotNil(t, manager)
	assert.NotNil(t, manager.connections)
	assert.Equal(t, maxConnections, manager.maxConnections)
	assert.Equal(t, 0, manager.activeConnections)
	assert.Equal(t, int64(0), manager.totalConnections)
}

// TestWebSocketManager_HandleWebSocket WebSocket 연결 처리 테스트
func TestWebSocketManager_HandleWebSocket(t *testing.T) {
	mockPTYManager := NewMockPTYSessionManagement()
	manager := NewWebSocketManager(mockPTYManager, 10)
	defer manager.Shutdown()

	// 테스트 서버 설정
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := manager.HandleWebSocket(w, r)
		assert.NoError(t, err)
	}))
	defer server.Close()

	// WebSocket 클라이언트 연결
	url := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	require.NoError(t, err)
	defer conn.Close()

	// 연결이 관리자에 등록되었는지 확인
	time.Sleep(100 * time.Millisecond)
	
	connections := manager.ListConnections()
	assert.Len(t, connections, 1)
	assert.Equal(t, 1, manager.activeConnections)
	assert.Equal(t, int64(1), manager.totalConnections)
}

// TestWebSocketManager_ConnectionLimit 연결 제한 테스트
func TestWebSocketManager_ConnectionLimit(t *testing.T) {
	mockPTYManager := NewMockPTYSessionManagement()
	maxConnections := 2
	manager := NewWebSocketManager(mockPTYManager, maxConnections)
	defer manager.Shutdown()

	connectionsEstablished := 0
	connectionsFailed := 0

	// 연결 제한을 초과하는 연결 시도
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := manager.HandleWebSocket(w, r)
		if err != nil {
			connectionsFailed++
			http.Error(w, "Connection limit exceeded", http.StatusTooManyRequests)
			return
		}
		connectionsEstablished++
	}))
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http")

	// 제한 내 연결들
	var validConns []*websocket.Conn
	for i := 0; i < maxConnections; i++ {
		conn, resp, err := websocket.DefaultDialer.Dial(url, nil)
		if err == nil {
			validConns = append(validConns, conn)
		} else if resp != nil && resp.StatusCode == http.StatusTooManyRequests {
			// 연결 제한으로 인한 실패는 예상됨
		}
	}

	// 초과 연결 시도
	_, resp, err := websocket.DefaultDialer.Dial(url, nil)
	if resp != nil {
		assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode)
	}
	assert.Error(t, err)

	// 연결 정리
	for _, conn := range validConns {
		conn.Close()
	}

	time.Sleep(100 * time.Millisecond)
	assert.LessOrEqual(t, manager.activeConnections, maxConnections)
}

// TestWebSocketManager_PTYAttachment PTY 세션 연결 테스트
func TestWebSocketManager_PTYAttachment(t *testing.T) {
	mockPTYManager := NewMockPTYSessionManagement()
	mockPTYSession := NewMockPTYSession()
	
	mockPTYManager.On("GetSession", "test-session").Return(mockPTYSession, nil)
	mockPTYSession.On("GetSessionID").Return("test-session")
	mockPTYSession.On("IsConnected").Return(true)

	manager := NewWebSocketManager(mockPTYManager, 10)
	defer manager.Shutdown()

	// WebSocket 연결 생성
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := manager.HandleWebSocket(w, r)
		assert.NoError(t, err)
	}))
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	require.NoError(t, err)
	defer conn.Close()

	time.Sleep(100 * time.Millisecond)

	// 연결 ID 가져오기
	connections := manager.ListConnections()
	require.Len(t, connections, 1)
	connectionID := connections[0].GetConnectionID()

	// PTY 세션 연결
	err = manager.AttachPTYToConnection(connectionID, "test-session")
	assert.NoError(t, err)

	// 연결된 PTY 세션 확인
	wsConn, err := manager.GetConnection(connectionID)
	assert.NoError(t, err)
	assert.Equal(t, "test-session", wsConn.GetPTYSessionID())

	// PTY 세션 분리
	err = manager.DetachPTYFromConnection(connectionID)
	assert.NoError(t, err)
	assert.Empty(t, wsConn.GetPTYSessionID())
}

// TestWebSocketManager_BroadcastToPTYSessions PTY 세션 브로드캐스트 테스트
func TestWebSocketManager_BroadcastToPTYSessions(t *testing.T) {
	mockPTYManager := NewMockPTYSessionManagement()
	mockPTYSession := NewMockPTYSession()
	
	mockPTYManager.On("GetSession", "broadcast-session").Return(mockPTYSession, nil)
	mockPTYSession.On("GetSessionID").Return("broadcast-session")

	manager := NewWebSocketManager(mockPTYManager, 10)
	defer manager.Shutdown()

	// 여러 WebSocket 연결 생성
	var connections []*websocket.Conn
	connectionIDs := make([]string, 0)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := manager.HandleWebSocket(w, r)
		assert.NoError(t, err)
	}))
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http")

	// 3개의 연결 생성
	for i := 0; i < 3; i++ {
		conn, _, err := websocket.DefaultDialer.Dial(url, nil)
		require.NoError(t, err)
		connections = append(connections, conn)
	}

	time.Sleep(100 * time.Millisecond)

	// 각 연결에 동일한 PTY 세션 연결
	managerConns := manager.ListConnections()
	for _, wsConn := range managerConns {
		connectionIDs = append(connectionIDs, wsConn.GetConnectionID())
		err := manager.AttachPTYToConnection(wsConn.GetConnectionID(), "broadcast-session")
		assert.NoError(t, err)
	}

	// 브로드캐스트 테스트
	testData := []byte("broadcast message")
	err := manager.BroadcastToPTYSessions("broadcast-session", MessageTypeData, testData)
	assert.NoError(t, err)

	// 연결 정리
	for _, conn := range connections {
		conn.Close()
	}
}

// TestWebSocketManager_ConnectionCleanup 연결 정리 테스트
func TestWebSocketManager_ConnectionCleanup(t *testing.T) {
	mockPTYManager := NewMockPTYSessionManagement()
	manager := NewWebSocketManager(mockPTYManager, 10)
	// 정리 간격을 짧게 설정
	manager.cleanupInterval = 100 * time.Millisecond
	defer manager.Shutdown()

	// WebSocket 연결 생성 후 즉시 종료
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := manager.HandleWebSocket(w, r)
		assert.NoError(t, err)
	}))
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	require.NoError(t, err)

	time.Sleep(50 * time.Millisecond)
	
	// 연결이 등록되었는지 확인
	connections := manager.ListConnections()
	assert.Len(t, connections, 1)

	// 연결 종료
	conn.Close()

	// 정리 루틴이 실행될 때까지 대기
	time.Sleep(200 * time.Millisecond)

	// 연결이 정리되었는지 확인
	connections = manager.ListConnections()
	assert.Len(t, connections, 0)
	assert.Equal(t, 0, manager.activeConnections)
}

// TestWebSocketManager_GetStats 통계 조회 테스트
func TestWebSocketManager_GetStats(t *testing.T) {
	mockPTYManager := NewMockPTYSessionManagement()
	manager := NewWebSocketManager(mockPTYManager, 10)
	defer manager.Shutdown()

	// 초기 통계 확인
	stats := manager.GetStats()
	assert.NotNil(t, stats)
	assert.Equal(t, int64(0), stats.TotalConnections)
	assert.Equal(t, 0, stats.ActiveConnections)
	assert.Equal(t, 10, stats.MaxConnections)
	assert.Equal(t, int64(0), stats.TotalMessages)
	assert.Equal(t, int64(0), stats.TotalErrors)

	// WebSocket 연결 생성
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := manager.HandleWebSocket(w, r)
		assert.NoError(t, err)
	}))
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	require.NoError(t, err)
	defer conn.Close()

	time.Sleep(100 * time.Millisecond)

	// 연결 후 통계 확인
	stats = manager.GetStats()
	assert.Equal(t, int64(1), stats.TotalConnections)
	assert.Equal(t, 1, stats.ActiveConnections)
	assert.Len(t, stats.ConnectionStats, 1)
}

// TestWebSocketManager_Shutdown 종료 테스트
func TestWebSocketManager_Shutdown(t *testing.T) {
	mockPTYManager := NewMockPTYSessionManagement()
	manager := NewWebSocketManager(mockPTYManager, 10)

	// WebSocket 연결 생성
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := manager.HandleWebSocket(w, r)
		assert.NoError(t, err)
	}))
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	require.NoError(t, err)
	defer conn.Close()

	time.Sleep(100 * time.Millisecond)

	// 연결이 활성화되었는지 확인
	assert.Equal(t, 1, manager.activeConnections)

	// 관리자 종료
	err = manager.Shutdown()
	assert.NoError(t, err)

	// 종료 후 상태 확인
	assert.Equal(t, 0, manager.activeConnections)
	assert.Len(t, manager.connections, 0)
}

// TestWebSocketStreamingManager_Creation 통합 스트리밍 관리자 생성 테스트
func TestWebSocketStreamingManager_Creation(t *testing.T) {
	mockPTYManager := NewMockPTYSessionManagement()
	mockContainerPTY := &MockContainerPTYManagement{}

	streamingManager := NewWebSocketStreamingManager(mockPTYManager, mockContainerPTY, 10)

	assert.NotNil(t, streamingManager)
	assert.NotNil(t, streamingManager.wsManager)
	assert.Equal(t, mockPTYManager, streamingManager.ptyManager)
	assert.Equal(t, mockContainerPTY, streamingManager.containerPTY)
}

// TestWebSocketStreamingManager_HandleConnection 통합 연결 처리 테스트
func TestWebSocketStreamingManager_HandleConnection(t *testing.T) {
	mockPTYManager := NewMockPTYSessionManagement()
	mockContainerPTY := &MockContainerPTYManagement{}
	mockPTYSession := NewMockPTYSession()

	mockPTYManager.On("GetSession", "integrated-session").Return(mockPTYSession, nil)
	mockPTYSession.On("GetSessionID").Return("integrated-session")

	streamingManager := NewWebSocketStreamingManager(mockPTYManager, mockContainerPTY, 10)
	defer streamingManager.Shutdown()

	// 테스트 서버 설정
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := streamingManager.HandleWebSocketConnection(w, r, "integrated-session")
		assert.NoError(t, err)
	}))
	defer server.Close()

	// WebSocket 클라이언트 연결
	url := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	require.NoError(t, err)
	defer conn.Close()

	time.Sleep(100 * time.Millisecond)

	// 연결 및 PTY 세션 연결 확인
	connections := streamingManager.GetManager().ListConnections()
	assert.Len(t, connections, 1)
	assert.Equal(t, "integrated-session", connections[0].GetPTYSessionID())
}

// TestWebSocketManager_ConcurrentConnections 동시 연결 테스트
func TestWebSocketManager_ConcurrentConnections(t *testing.T) {
	mockPTYManager := NewMockPTYSessionManagement()
	manager := NewWebSocketManager(mockPTYManager, 50)
	defer manager.Shutdown()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := manager.HandleWebSocket(w, r)
		assert.NoError(t, err)
	}))
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http")

	// 동시에 여러 연결 생성
	numConnections := 10
	var wg sync.WaitGroup
	var connections []*websocket.Conn
	var mu sync.Mutex

	wg.Add(numConnections)
	for i := 0; i < numConnections; i++ {
		go func() {
			defer wg.Done()
			conn, _, err := websocket.DefaultDialer.Dial(url, nil)
			if err == nil {
				mu.Lock()
				connections = append(connections, conn)
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	time.Sleep(200 * time.Millisecond)

	// 모든 연결이 성공했는지 확인
	mu.Lock()
	defer mu.Unlock()
	
	managerConnections := manager.ListConnections()
	assert.GreaterOrEqual(t, len(managerConnections), numConnections-2) // 약간의 오차 허용
	assert.LessOrEqual(t, len(managerConnections), numConnections)

	// 연결 정리
	for _, conn := range connections {
		conn.Close()
	}
}

// TestWebSocketManager_ErrorHandling 에러 처리 테스트
func TestWebSocketManager_ErrorHandling(t *testing.T) {
	mockPTYManager := NewMockPTYSessionManagement()
	manager := NewWebSocketManager(mockPTYManager, 10)
	defer manager.Shutdown()

	// 존재하지 않는 연결에 대한 조회
	_, err := manager.GetConnection("non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// 존재하지 않는 연결 종료 시도
	err = manager.CloseConnection("non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// 존재하지 않는 PTY 세션으로 브로드캐스트
	err = manager.BroadcastToPTYSessions("non-existent-session", MessageTypeData, []byte("test"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no WebSocket connections found")
}

// BenchmarkWebSocketManager_HandleConnections 연결 처리 성능 벤치마크 
func BenchmarkWebSocketManager_HandleConnections(b *testing.B) {
	mockPTYManager := NewMockPTYSessionManagement()
	manager := NewWebSocketManager(mockPTYManager, 1000)
	defer manager.Shutdown()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		manager.HandleWebSocket(w, r)
	}))
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			conn, _, err := websocket.DefaultDialer.Dial(url, nil)
			if err == nil {
				conn.Close()
			}
		}
	})
}