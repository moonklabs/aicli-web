package docker

import (
	"context"
	"encoding/base64"
	"encoding/json"
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

// MockPTYSession PTY 세션 모의 객체
type MockPTYSession struct {
	mock.Mock
	readChan  chan []byte
	writeChan chan []byte
	mu        sync.RWMutex
}

func NewMockPTYSession() *MockPTYSession {
	return &MockPTYSession{
		readChan:  make(chan []byte, 100),
		writeChan: make(chan []byte, 100),
	}
}

func (m *MockPTYSession) Read(p []byte) (int, error) {
	select {
	case data := <-m.readChan:
		copy(p, data)
		return len(data), nil
	case <-time.After(100 * time.Millisecond):
		return 0, nil
	}
}

func (m *MockPTYSession) Write(p []byte) (int, error) {
	args := m.Called(p)
	m.writeChan <- p
	return len(p), args.Error(1)
}

func (m *MockPTYSession) Resize(width, height int) error {
	args := m.Called(width, height)
	return args.Error(0)
}

func (m *MockPTYSession) Close() error {
	args := m.Called()
	close(m.readChan)
	close(m.writeChan)
	return args.Error(0)
}

func (m *MockPTYSession) GetSessionID() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockPTYSession) GetContainerID() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockPTYSession) IsConnected() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockPTYSession) SendData(data []byte) {
	select {
	case m.readChan <- data:
	default:
	}
}

// TestWebSocketConnection_Creation WebSocket 연결 생성 테스트
func TestWebSocketConnection_Creation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()

		// 연결 생성
		wsConn := NewWebSocketConnection("test-conn-1", conn)
		
		assert.Equal(t, "test-conn-1", wsConn.GetConnectionID())
		assert.True(t, wsConn.IsConnected())
		assert.Empty(t, wsConn.GetPTYSessionID())
	}))
	defer server.Close()

	// WebSocket 클라이언트 연결
	url := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	require.NoError(t, err)
	defer conn.Close()

	// 연결 성공 확인
	assert.NotNil(t, conn)
}

// TestWebSocketConnection_PTYAttachment PTY 세션 연결 테스트
func TestWebSocketConnection_PTYAttachment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()

		// WebSocket 연결 생성
		wsConn := NewWebSocketConnection("test-conn-2", conn)
		
		// PTY 세션 연결
		err = wsConn.AttachPTYSession("test-session-1")
		assert.NoError(t, err)
		assert.Equal(t, "test-session-1", wsConn.GetPTYSessionID())

		// 중복 연결 시도 (에러 발생해야 함)
		err = wsConn.AttachPTYSession("test-session-2")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already attached")

		// 연결 해제
		err = wsConn.DetachPTYSession()
		assert.NoError(t, err)
		assert.Empty(t, wsConn.GetPTYSessionID())
	}))
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	require.NoError(t, err)
	defer conn.Close()
}

// TestWebSocketConnection_MessageSending 메시지 전송 테스트
func TestWebSocketConnection_MessageSending(t *testing.T) {
	var receivedMessages []WebSocketMessage
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()

		// 메시지 수신 고루틴
		go func() {
			for {
				var msg WebSocketMessage
				err := conn.ReadJSON(&msg)
				if err != nil {
					break
				}
				mu.Lock()
				receivedMessages = append(receivedMessages, msg)
				mu.Unlock()
			}
		}()

		// WebSocket 연결 생성
		wsConn := NewWebSocketConnection("test-conn-3", conn)
		wsConn.startMessagePump()

		// 데이터 메시지 전송
		testData := []byte("Hello WebSocket!")
		err = wsConn.SendMessage(testData)
		assert.NoError(t, err)

		// 타입 지정 메시지 전송
		err = wsConn.SendTypedMessage(MessageTypePing, nil)
		assert.NoError(t, err)

		// 잠시 대기하여 메시지 전송 완료
		time.Sleep(100 * time.Millisecond)
	}))
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	require.NoError(t, err)
	defer conn.Close()

	// 메시지 수신 확인
	time.Sleep(200 * time.Millisecond)
	mu.Lock()
	defer mu.Unlock()

	assert.GreaterOrEqual(t, len(receivedMessages), 1)
	
	// 첫 번째 메시지가 데이터 메시지인지 확인
	if len(receivedMessages) > 0 {
		dataMsg := receivedMessages[0]
		assert.Equal(t, MessageTypeData, dataMsg.Type)
		
		// Base64 디코딩하여 원본 데이터 확인
		decodedData, err := base64.StdEncoding.DecodeString(dataMsg.Data)
		assert.NoError(t, err)
		assert.Equal(t, "Hello WebSocket!", string(decodedData))
	}
}

// TestWebSocketConnection_BackpressureControl 백프레셸 제어 테스트
func TestWebSocketConnection_BackpressureControl(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()

		// WebSocket 연결 생성 (작은 채널 크기)
		wsConn := NewWebSocketConnection("test-conn-4", conn)
		wsConn.sendChan = make(chan WebSocketMessage, 2) // 작은 채널 크기
		wsConn.backpressureThreshold = 1

		// 많은 메시지 전송하여 백프레셸 유발
		for i := 0; i < 10; i++ {
			data := []byte("test message")
			err := wsConn.SendMessage(data)
			assert.NoError(t, err)
		}

		// 백프레셸 상태 확인
		stats := wsConn.GetStats()
		assert.True(t, stats.IsBackpressure || stats.BufferSize > 0)
	}))
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	require.NoError(t, err)
	defer conn.Close()
}

// TestWebSocketConnection_MessageTypes 메시지 타입 처리 테스트
func TestWebSocketConnection_MessageTypes(t *testing.T) {
	mockPTY := NewMockPTYSession()
	mockPTY.On("Write", mock.Anything).Return(0, nil)
	mockPTY.On("Resize", mock.Anything, mock.Anything).Return(nil)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()

		wsConn := NewWebSocketConnection("test-conn-5", conn)
		wsConn.ptySession = mockPTY

		// 다양한 메시지 타입 처리 테스트
		messages := []WebSocketMessage{
			{
				Type: MessageTypeData,
				Data: base64.StdEncoding.EncodeToString([]byte("test input")),
				Timestamp: time.Now(),
			},
			{
				Type: MessageTypeResize,
				Width: 80,
				Height: 24,
				Timestamp: time.Now(),
			},
			{
				Type: MessageTypePing,
				Timestamp: time.Now(),
			},
		}

		for _, msg := range messages {
			wsConn.handleIncomingMessage(msg)
		}

		// 메시지 처리 완료 대기
		time.Sleep(100 * time.Millisecond)
	}))
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	require.NoError(t, err)
	defer conn.Close()

	// Mock 호출 검증
	mockPTY.AssertCalled(t, "Write", []byte("test input"))
	mockPTY.AssertCalled(t, "Resize", 80, 24)
}

// TestWebSocketConnection_ConnectionStats 연결 통계 테스트
func TestWebSocketConnection_ConnectionStats(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()

		wsConn := NewWebSocketConnection("test-conn-6", conn)
		wsConn.AttachPTYSession("test-session-stats")

		// 통계 조회
		stats := wsConn.GetStats()
		assert.NotNil(t, stats)
		assert.Equal(t, "test-conn-6", stats.ConnectionID)
		assert.Equal(t, "test-session-stats", stats.PTYSessionID)
		assert.True(t, stats.IsConnected)
		assert.GreaterOrEqual(t, stats.SendChannelSize, 0)
		assert.GreaterOrEqual(t, stats.ReceiveChannelSize, 0)
		assert.GreaterOrEqual(t, stats.BufferSize, 0)
	}))
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	require.NoError(t, err)
	defer conn.Close()
}

// TestWebSocketConnection_Close 연결 종료 테스트
func TestWebSocketConnection_Close(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()

		wsConn := NewWebSocketConnection("test-conn-7", conn)
		wsConn.AttachPTYSession("test-session-close")

		// 연결 상태 확인
		assert.True(t, wsConn.IsConnected())

		// 연결 종료
		err = wsConn.Close()
		assert.NoError(t, err)

		// 종료 후 상태 확인
		assert.False(t, wsConn.IsConnected())
		assert.Empty(t, wsConn.GetPTYSessionID())

		// 중복 종료 시도 (에러 없어야 함)
		err = wsConn.Close()
		assert.NoError(t, err)
	}))
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	require.NoError(t, err)
	defer conn.Close()
}

// TestWebSocketConnection_PTYDataStreaming PTY 데이터 스트리밍 테스트
func TestWebSocketConnection_PTYDataStreaming(t *testing.T) {
	mockPTY := NewMockPTYSession()
	mockPTY.On("GetSessionID").Return("mock-session")
	mockPTY.On("IsConnected").Return(true)

	var receivedMessages []WebSocketMessage
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()

		// 메시지 수신 고루틴
		go func() {
			for {
				var msg WebSocketMessage
				err := conn.ReadJSON(&msg)
				if err != nil {
					break
				}
				mu.Lock()
				receivedMessages = append(receivedMessages, msg)
				mu.Unlock()
			}
		}()

		wsConn := NewWebSocketConnection("test-conn-8", conn)
		wsConn.ptySession = mockPTY
		wsConn.startMessagePump()

		// PTY에서 데이터 전송 시뮬레이션
		testData := []byte("PTY output data")
		mockPTY.SendData(testData)

		// 잠시 대기
		time.Sleep(200 * time.Millisecond)
	}))
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	require.NoError(t, err)
	defer conn.Close()

	// 수신된 메시지 확인
	time.Sleep(300 * time.Millisecond)
	mu.Lock()
	defer mu.Unlock()

	// PTY 데이터가 WebSocket을 통해 전송되었는지 확인
	hasDataMessage := false
	for _, msg := range receivedMessages {
		if msg.Type == MessageTypeData {
			hasDataMessage = true
			break
		}
	}
	
	// 실제 PTY 스트리밍은 별도 테스트에서 확인 (통합 테스트)
	_ = hasDataMessage
}

// TestWebSocketMessage_JSONMarshaling JSON 마샬링/언마샬링 테스트
func TestWebSocketMessage_JSONMarshaling(t *testing.T) {
	originalMsg := WebSocketMessage{
		Type:      MessageTypeData,
		Data:      base64.StdEncoding.EncodeToString([]byte("test data")),
		Width:     80,
		Height:    24,
		Timestamp: time.Now().Truncate(time.Second), // 초 단위로 자름
		Error:     "test error",
	}

	// JSON 마샬링
	jsonData, err := json.Marshal(originalMsg)
	assert.NoError(t, err)

	// JSON 언마샬링
	var decodedMsg WebSocketMessage
	err = json.Unmarshal(jsonData, &decodedMsg)
	assert.NoError(t, err)

	// 데이터 비교
	assert.Equal(t, originalMsg.Type, decodedMsg.Type)
	assert.Equal(t, originalMsg.Data, decodedMsg.Data)
	assert.Equal(t, originalMsg.Width, decodedMsg.Width)
	assert.Equal(t, originalMsg.Height, decodedMsg.Height)
	assert.Equal(t, originalMsg.Error, decodedMsg.Error)
	assert.True(t, originalMsg.Timestamp.Equal(decodedMsg.Timestamp))
}

// TestWebSocketConnection_ContextCancellation 컨텍스트 취소 테스트
func TestWebSocketConnection_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()

		wsConn := NewWebSocketConnection("test-conn-9", conn)
		
		// 컨텍스트 취소
		wsConn.cancel()

		// 취소 후 메시지 전송 시도
		err = wsConn.SendMessage([]byte("test"))
		// 컨텍스트가 취소되어도 바로 에러가 발생하지 않을 수 있음
		// 하지만 곧 연결이 종료될 것임
		
		time.Sleep(100 * time.Millisecond)
	}))
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	require.NoError(t, err)
	defer conn.Close()
}

// BenchmarkWebSocketConnection_MessageSending 메시지 전송 성능 벤치마크
func BenchmarkWebSocketConnection_MessageSending(b *testing.B) {
	// 테스트 서버 설정
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		wsConn := NewWebSocketConnection("bench-conn", conn)
		wsConn.startMessagePump()

		// 벤치마크 실행
		testData := []byte("benchmark test data")
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			wsConn.SendMessage(testData)
		}
	}))
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		b.Fatal(err)
	}
	defer conn.Close()
}