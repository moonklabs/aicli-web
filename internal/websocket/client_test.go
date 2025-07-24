package websocket

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestWebSocket(t *testing.T) (*websocket.Conn, *websocket.Conn) {
	// WebSocket 업그레이더 생성
	upgrader := websocket.Upgrader{}
	
	// 테스트 서버 생성
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()
		
		// 간단한 에코 서버
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				break
			}
			err = conn.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				break
			}
		}
	}))
	
	// 클라이언트 연결
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	clientConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	
	// 서버측 연결은 실제로는 얻을 수 없으므로 클라이언트 연결을 반환
	return clientConn, clientConn
}

func TestClient_Basic(t *testing.T) {
	// 모의 허브 생성
	hub := NewHub(nil)
	
	// 모의 연결 생성 (실제로는 nil이지만 테스트용)
	client := NewClient("test-client", "test-user", nil, hub, nil)
	
	assert.Equal(t, "test-client", client.ID)
	assert.Equal(t, "test-user", client.UserID)
	assert.False(t, client.IsAuthenticated())
	assert.True(t, client.IsConnected())
	assert.Empty(t, client.GetChannels())
}

func TestClient_Authentication(t *testing.T) {
	hub := NewHub(nil)
	client := NewClient("test-client", "test-user", nil, hub, nil)
	
	// 초기 상태
	assert.False(t, client.IsAuthenticated())
	
	// 인증 설정
	client.SetAuthenticated(true)
	assert.True(t, client.IsAuthenticated())
	
	// 인증 해제
	client.SetAuthenticated(false)
	assert.False(t, client.IsAuthenticated())
}

func TestClient_ChannelSubscription(t *testing.T) {
	hub := NewHub(nil)
	client := NewClient("test-client", "test-user", nil, hub, nil)
	
	// 초기 상태 - 구독한 채널 없음
	assert.Empty(t, client.GetChannels())
	assert.False(t, client.IsSubscribed("test-channel"))
	
	// 채널 구독
	client.Subscribe("channel1", "channel2")
	channels := client.GetChannels()
	assert.Len(t, channels, 2)
	assert.True(t, client.IsSubscribed("channel1"))
	assert.True(t, client.IsSubscribed("channel2"))
	assert.False(t, client.IsSubscribed("channel3"))
	
	// 추가 구독
	client.Subscribe("channel3")
	assert.Len(t, client.GetChannels(), 3)
	assert.True(t, client.IsSubscribed("channel3"))
	
	// 구독 취소
	client.Unsubscribe("channel1")
	assert.Len(t, client.GetChannels(), 2)
	assert.False(t, client.IsSubscribed("channel1"))
	assert.True(t, client.IsSubscribed("channel2"))
	
	// 여러 채널 구독 취소
	client.Unsubscribe("channel2", "channel3")
	assert.Empty(t, client.GetChannels())
}

func TestClient_MessageSending(t *testing.T) {
	hub := NewHub(nil)
	client := NewClient("test-client", "test-user", nil, hub, nil)
	
	// 메시지 전송 시뮬레이션 (실제 연결 없이)
	message := NewMessage(MessageTypeAuth, AuthMessage{Token: "test"})
	
	// SendMessage는 실제 연결이 없으면 false 반환
	result := client.SendMessage(message)
	assert.False(t, result) // 실제 연결이 없으므로 false
}

func TestClient_ErrorAndSuccessMessages(t *testing.T) {
	hub := NewHub(nil)
	client := NewClient("test-client", "test-user", nil, hub, nil)
	
	// 에러/성공 메시지 전송은 실제 연결 없이는 테스트 어려움
	// 메서드 호출만 확인
	client.SendError("TEST_ERROR", "테스트 에러", "상세정보")
	client.SendSuccess("성공", map[string]string{"key": "value"})
	
	// 패닉이 발생하지 않으면 성공
}

func TestClient_Config(t *testing.T) {
	// 기본 설정 테스트
	config := DefaultClientConfig()
	assert.Equal(t, 256, config.SendBufferSize)
	assert.Equal(t, 10*time.Second, config.WriteTimeout)
	assert.Equal(t, 60*time.Second, config.ReadTimeout)
	assert.Equal(t, 30*time.Second, config.PingInterval)
	assert.Equal(t, 10*time.Second, config.PongTimeout)
	assert.Equal(t, int64(1024*1024), config.MaxMessageSize)
}

func TestClient_HandleAuthMessage(t *testing.T) {
	hub := NewHub(nil)
	client := NewClient("test-client", "test-user", nil, hub, nil)
	
	// 유효한 토큰으로 인증 메시지 처리
	authMsg := NewMessage(MessageTypeAuth, AuthMessage{Token: "valid-token"})
	err := client.handleAuthMessage(authMsg)
	assert.NoError(t, err)
	assert.True(t, client.IsAuthenticated())
	
	// 사용자 채널 구독 확인
	assert.True(t, client.IsSubscribed(GetUserChannel("test-user")))
	
	// 빈 토큰으로 인증 시도
	client.SetAuthenticated(false)
	authMsg = NewMessage(MessageTypeAuth, AuthMessage{Token: ""})
	err = client.handleAuthMessage(authMsg)
	assert.NoError(t, err)
	assert.False(t, client.IsAuthenticated())
}

func TestClient_HandlePingMessage(t *testing.T) {
	hub := NewHub(nil)
	client := NewClient("test-client", "test-user", nil, hub, nil)
	
	// 핑 메시지 처리
	pingMsg := NewMessage(MessageTypePing, nil)
	err := client.handlePingMessage(pingMsg)
	assert.NoError(t, err)
}

func TestClient_HandleSubscribeMessage(t *testing.T) {
	hub := NewHub(nil)
	client := NewClient("test-client", "test-user", nil, hub, nil)
	
	// 인증되지 않은 상태에서 구독 시도
	subMsg := NewMessage(MessageTypeSubscribe, SubscribeMessage{
		Channels: []string{"channel1", "channel2"},
	})
	err := client.handleSubscribeMessage(subMsg)
	assert.NoError(t, err)
	assert.Empty(t, client.GetChannels()) // 인증되지 않았으므로 구독 안됨
	
	// 인증 후 구독 시도
	client.SetAuthenticated(true)
	err = client.handleSubscribeMessage(subMsg)
	assert.NoError(t, err)
	assert.Len(t, client.GetChannels(), 2)
	assert.True(t, client.IsSubscribed("channel1"))
	assert.True(t, client.IsSubscribed("channel2"))
}

func TestClient_HandleUnsubscribeMessage(t *testing.T) {
	hub := NewHub(nil)
	client := NewClient("test-client", "test-user", nil, hub, nil)
	
	// 채널 구독
	client.Subscribe("channel1", "channel2", "channel3")
	assert.Len(t, client.GetChannels(), 3)
	
	// 구독 취소 메시지 처리
	unsubMsg := NewMessage(MessageTypeUnsubscribe, UnsubscribeMessage{
		Channels: []string{"channel1", "channel2"},
	})
	err := client.handleUnsubscribeMessage(unsubMsg)
	assert.NoError(t, err)
	assert.Len(t, client.GetChannels(), 1)
	assert.False(t, client.IsSubscribed("channel1"))
	assert.False(t, client.IsSubscribed("channel2"))
	assert.True(t, client.IsSubscribed("channel3"))
}

func TestClient_HandleCommandMessage(t *testing.T) {
	hub := NewHub(nil)
	client := NewClient("test-client", "test-user", nil, hub, nil)
	
	// 인증되지 않은 상태에서 명령 시도
	cmdMsg := NewMessage(MessageTypeCommand, CommandMessage{
		Command:   "test-command",
		SessionID: "session-123",
	})
	err := client.handleCommandMessage(cmdMsg)
	assert.NoError(t, err) // 에러는 없지만 인증 에러 메시지 전송됨
	
	// 인증 후 명령 시도
	client.SetAuthenticated(true)
	err = client.handleCommandMessage(cmdMsg)
	assert.NoError(t, err)
}

func TestClient_GetStats(t *testing.T) {
	hub := NewHub(nil)
	client := NewClient("test-client", "test-user", nil, hub, nil)
	
	stats := client.GetStats()
	assert.NotNil(t, stats)
	assert.Equal(t, "test-client", stats["id"])
	assert.Equal(t, "test-user", stats["user_id"])
	assert.Equal(t, false, stats["is_authenticated"])
	assert.Equal(t, int64(0), stats["messages_received"])
	assert.Equal(t, int64(0), stats["messages_sent"])
	assert.NotNil(t, stats["connected_at"])
	assert.NotNil(t, stats["uptime"])
}

func TestClient_Stop(t *testing.T) {
	hub := NewHub(nil)
	client := NewClient("test-client", "test-user", nil, hub, nil)
	
	assert.True(t, client.IsConnected())
	
	// 클라이언트 중지
	client.Stop()
	
	// 약간 대기 후 연결 상태 확인
	time.Sleep(10 * time.Millisecond)
	
	// context가 취소되면 done 채널이 닫히지 않더라도 
	// cancel이 호출되었는지 확인할 수 있음
	select {
	case <-client.ctx.Done():
		// context가 취소됨
	default:
		t.Error("클라이언트 context가 취소되지 않음")
	}
}

func TestClientConfig_Validation(t *testing.T) {
	config := DefaultClientConfig()
	
	// 설정값이 합리적인지 확인
	assert.Greater(t, config.SendBufferSize, 0)
	assert.Greater(t, config.WriteTimeout, time.Duration(0))
	assert.Greater(t, config.ReadTimeout, time.Duration(0))
	assert.Greater(t, config.PingInterval, time.Duration(0))
	assert.Greater(t, config.PongTimeout, time.Duration(0))
	assert.Greater(t, config.MaxMessageSize, int64(0))
}

func TestClient_MessageHandling(t *testing.T) {
	hub := NewHub(nil)
	client := NewClient("test-client", "test-user", nil, hub, nil)
	
	tests := []struct {
		name    string
		msgType MessageType
		data    interface{}
		wantErr bool
	}{
		{
			name:    "Auth message",
			msgType: MessageTypeAuth,
			data:    AuthMessage{Token: "test"},
			wantErr: false,
		},
		{
			name:    "Ping message",
			msgType: MessageTypePing,
			data:    nil,
			wantErr: false,
		},
		{
			name:    "Subscribe message",
			msgType: MessageTypeSubscribe,
			data:    SubscribeMessage{Channels: []string{"test"}},
			wantErr: false,
		},
		{
			name:    "Unknown message",
			msgType: MessageType("unknown"),
			data:    nil,
			wantErr: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := NewMessage(tt.msgType, tt.data)
			data, err := msg.ToJSON()
			require.NoError(t, err)
			
			err = client.handleMessage(data)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}