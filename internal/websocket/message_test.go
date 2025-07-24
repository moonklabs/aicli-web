package websocket

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessage_Basic(t *testing.T) {
	// 기본 메시지 생성
	data := map[string]string{"test": "value"}
	msg := NewMessage(MessageTypeLog, data)
	
	assert.Equal(t, MessageTypeLog, msg.Type)
	assert.NotEmpty(t, msg.Data)
	assert.False(t, msg.Timestamp.IsZero())
}

func TestMessage_WithMethods(t *testing.T) {
	msg := NewMessage(MessageTypeAuth, nil)
	
	// 메서드 체이닝 테스트
	msg.WithChannel("test-channel").WithID("test-id").WithUserID("user-123")
	
	assert.Equal(t, "test-channel", msg.Channel)
	assert.Equal(t, "test-id", msg.ID)
	assert.Equal(t, "user-123", msg.UserID)
}

func TestMessage_IsSystemMessage(t *testing.T) {
	tests := []struct {
		msgType  MessageType
		expected bool
	}{
		{MessageTypeAuth, true},
		{MessageTypePing, true},
		{MessageTypePong, true},
		{MessageTypeError, true},
		{MessageTypeSuccess, true},
		{MessageTypeSubscribe, true},
		{MessageTypeUnsubscribe, true},
		{MessageTypeLog, false},
		{MessageTypeStatus, false},
		{MessageTypeEvent, false},
		{MessageTypeCommand, false},
		{MessageTypeTask, false},
		{MessageTypeSession, false},
	}
	
	for _, tt := range tests {
		t.Run(string(tt.msgType), func(t *testing.T) {
			msg := NewMessage(tt.msgType, nil)
			assert.Equal(t, tt.expected, msg.IsSystemMessage())
			assert.Equal(t, !tt.expected, msg.IsBusinessMessage())
		})
	}
}

func TestMessage_ToJSON(t *testing.T) {
	data := map[string]string{"key": "value"}
	msg := NewMessage(MessageTypeLog, data)
	msg.WithChannel("test").WithID("123")
	
	jsonData, err := msg.ToJSON()
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)
	
	// JSON 파싱 테스트
	var parsed Message
	err = json.Unmarshal(jsonData, &parsed)
	assert.NoError(t, err)
	assert.Equal(t, msg.Type, parsed.Type)
	assert.Equal(t, msg.Channel, parsed.Channel)
	assert.Equal(t, msg.ID, parsed.ID)
}

func TestParseMessage(t *testing.T) {
	// 원본 메시지
	original := NewMessage(MessageTypeAuth, AuthMessage{Token: "test-token"})
	original.WithChannel("auth").WithID("auth-123")
	
	// JSON으로 변환
	jsonData, err := original.ToJSON()
	require.NoError(t, err)
	
	// 다시 파싱
	parsed, err := ParseMessage(jsonData)
	assert.NoError(t, err)
	assert.Equal(t, original.Type, parsed.Type)
	assert.Equal(t, original.Channel, parsed.Channel)
	assert.Equal(t, original.ID, parsed.ID)
	
	// 잘못된 JSON 테스트
	_, err = ParseMessage([]byte("invalid json"))
	assert.Error(t, err)
}

func TestNewErrorMessage(t *testing.T) {
	msg := NewErrorMessage("TEST_ERROR", "테스트 에러", "상세 정보")
	
	assert.Equal(t, MessageTypeError, msg.Type)
	
	// 에러 데이터 파싱
	var errorData ErrorMessage
	err := json.Unmarshal(msg.Data, &errorData)
	assert.NoError(t, err)
	assert.Equal(t, "TEST_ERROR", errorData.Code)
	assert.Equal(t, "테스트 에러", errorData.Message)
	assert.Equal(t, "상세 정보", errorData.Details)
}

func TestNewSuccessMessage(t *testing.T) {
	data := map[string]interface{}{"result": "success"}
	msg := NewSuccessMessage("작업 완료", data)
	
	assert.Equal(t, MessageTypeSuccess, msg.Type)
	
	// 성공 데이터 파싱
	var successData SuccessMessage
	err := json.Unmarshal(msg.Data, &successData)
	assert.NoError(t, err)
	assert.Equal(t, "작업 완료", successData.Message)
	assert.NotNil(t, successData.Data)
}

func TestNewLogMessage(t *testing.T) {
	msg := NewLogMessage("INFO", "테스트 로그", "test-source", "session-123", "task-456")
	
	assert.Equal(t, MessageTypeLog, msg.Type)
	
	// 로그 데이터 파싱
	var logData LogMessage
	err := json.Unmarshal(msg.Data, &logData)
	assert.NoError(t, err)
	assert.Equal(t, "INFO", logData.Level)
	assert.Equal(t, "테스트 로그", logData.Message)
	assert.Equal(t, "test-source", logData.Source)
	assert.Equal(t, "session-123", logData.SessionID)
	assert.Equal(t, "task-456", logData.TaskID)
	assert.False(t, logData.Timestamp.IsZero())
}

func TestNewTaskMessage(t *testing.T) {
	data := map[string]interface{}{"extra": "info"}
	msg := NewTaskMessage("task-123", "session-456", "running", "output", "error", data)
	
	assert.Equal(t, MessageTypeTask, msg.Type)
	
	// 태스크 데이터 파싱
	var taskData TaskMessage
	err := json.Unmarshal(msg.Data, &taskData)
	assert.NoError(t, err)
	assert.Equal(t, "task-123", taskData.TaskID)
	assert.Equal(t, "session-456", taskData.SessionID)
	assert.Equal(t, "running", taskData.Status)
	assert.Equal(t, "output", taskData.Output)
	assert.Equal(t, "error", taskData.Error)
	assert.NotNil(t, taskData.Data)
}

func TestNewSessionMessage(t *testing.T) {
	data := map[string]interface{}{"info": "value"}
	msg := NewSessionMessage("session-123", "project-456", "active", data)
	
	assert.Equal(t, MessageTypeSession, msg.Type)
	
	// 세션 데이터 파싱
	var sessionData SessionMessage
	err := json.Unmarshal(msg.Data, &sessionData)
	assert.NoError(t, err)
	assert.Equal(t, "session-123", sessionData.SessionID)
	assert.Equal(t, "project-456", sessionData.ProjectID)
	assert.Equal(t, "active", sessionData.Status)
	assert.NotNil(t, sessionData.Data)
}

func TestMessage_ParseAuthMessage(t *testing.T) {
	// 인증 메시지 생성
	authMsg := AuthMessage{Token: "test-token"}
	msg := NewMessage(MessageTypeAuth, authMsg)
	
	// 파싱 테스트
	parsed, err := msg.ParseAuthMessage()
	assert.NoError(t, err)
	assert.Equal(t, "test-token", parsed.Token)
	
	// 잘못된 데이터로 파싱 시도
	invalidMsg := NewMessage(MessageTypeAuth, "invalid")
	_, err = invalidMsg.ParseAuthMessage()
	assert.Error(t, err)
}

func TestMessage_ParseSubscribeMessage(t *testing.T) {
	// 구독 메시지 생성
	subMsg := SubscribeMessage{Channels: []string{"channel1", "channel2"}}
	msg := NewMessage(MessageTypeSubscribe, subMsg)
	
	// 파싱 테스트
	parsed, err := msg.ParseSubscribeMessage()
	assert.NoError(t, err)
	assert.Equal(t, []string{"channel1", "channel2"}, parsed.Channels)
}

func TestMessage_ParseUnsubscribeMessage(t *testing.T) {
	// 구독 취소 메시지 생성
	unsubMsg := UnsubscribeMessage{Channels: []string{"channel1"}}
	msg := NewMessage(MessageTypeUnsubscribe, unsubMsg)
	
	// 파싱 테스트
	parsed, err := msg.ParseUnsubscribeMessage()
	assert.NoError(t, err)
	assert.Equal(t, []string{"channel1"}, parsed.Channels)
}

func TestMessage_ParseCommandMessage(t *testing.T) {
	// 명령 메시지 생성
	cmdMsg := CommandMessage{
		Command:   "test-command",
		SessionID: "session-123",
		Args:      map[string]string{"arg1": "value1"},
	}
	msg := NewMessage(MessageTypeCommand, cmdMsg)
	
	// 파싱 테스트
	parsed, err := msg.ParseCommandMessage()
	assert.NoError(t, err)
	assert.Equal(t, "test-command", parsed.Command)
	assert.Equal(t, "session-123", parsed.SessionID)
	assert.Equal(t, map[string]string{"arg1": "value1"}, parsed.Args)
}

func TestChannelHelpers(t *testing.T) {
	// 채널 헬퍼 함수 테스트
	assert.Equal(t, "workspace:ws-123", GetWorkspaceChannel("ws-123"))
	assert.Equal(t, "session:sess-456", GetSessionChannel("sess-456"))
	assert.Equal(t, "task:task-789", GetTaskChannel("task-789"))
	assert.Equal(t, "user:user-abc", GetUserChannel("user-abc"))
}

func TestMessageTypes(t *testing.T) {
	// 메시지 타입 상수 확인
	systemTypes := []MessageType{
		MessageTypeAuth,
		MessageTypePing,
		MessageTypePong,
		MessageTypeError,
		MessageTypeSuccess,
		MessageTypeSubscribe,
		MessageTypeUnsubscribe,
	}
	
	businessTypes := []MessageType{
		MessageTypeLog,
		MessageTypeStatus,
		MessageTypeEvent,
		MessageTypeCommand,
		MessageTypeTask,
		MessageTypeSession,
	}
	
	for _, msgType := range systemTypes {
		msg := NewMessage(msgType, nil)
		assert.True(t, msg.IsSystemMessage(), "Expected %s to be system message", msgType)
		assert.False(t, msg.IsBusinessMessage(), "Expected %s to not be business message", msgType)
	}
	
	for _, msgType := range businessTypes {
		msg := NewMessage(msgType, nil)
		assert.False(t, msg.IsSystemMessage(), "Expected %s to not be system message", msgType)
		assert.True(t, msg.IsBusinessMessage(), "Expected %s to be business message", msgType)
	}
}

func TestChannelConstants(t *testing.T) {
	// 채널 상수 확인
	assert.Equal(t, "workspace", ChannelWorkspace)
	assert.Equal(t, "session", ChannelSession)
	assert.Equal(t, "task", ChannelTask)
	assert.Equal(t, "system", ChannelSystem)
	assert.Equal(t, "broadcast", ChannelBroadcast)
}