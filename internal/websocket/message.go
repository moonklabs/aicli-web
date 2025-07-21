package websocket

import (
	"encoding/json"
	"time"
)

// MessageType 메시지 타입 열거형
type MessageType string

const (
	// 시스템 메시지
	MessageTypeAuth       MessageType = "auth"        // 인증
	MessageTypePing       MessageType = "ping"        // 핑
	MessageTypePong       MessageType = "pong"        // 퐁
	MessageTypeError      MessageType = "error"       // 에러
	MessageTypeSuccess    MessageType = "success"     // 성공
	MessageTypeSubscribe  MessageType = "subscribe"   // 채널 구독
	MessageTypeUnsubscribe MessageType = "unsubscribe" // 구독 취소
	
	// 비즈니스 메시지
	MessageTypeLog        MessageType = "log"         // 로그 스트림
	MessageTypeStatus     MessageType = "status"      // 상태 업데이트
	MessageTypeEvent      MessageType = "event"       // 이벤트
	MessageTypeCommand    MessageType = "command"     // 명령
	MessageTypeTask       MessageType = "task"        // 태스크 업데이트
	MessageTypeSession    MessageType = "session"     // 세션 업데이트
)

// Message WebSocket 메시지 구조체
type Message struct {
	Type      MessageType     `json:"type"`
	ID        string          `json:"id,omitempty"`
	Channel   string          `json:"channel,omitempty"`
	Data      json.RawMessage `json:"data"`
	Timestamp time.Time       `json:"timestamp"`
	UserID    string          `json:"user_id,omitempty"`
}

// AuthMessage 인증 메시지 데이터
type AuthMessage struct {
	Token string `json:"token"`
}

// ErrorMessage 에러 메시지 데이터
type ErrorMessage struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// SuccessMessage 성공 메시지 데이터
type SuccessMessage struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// SubscribeMessage 구독 메시지 데이터
type SubscribeMessage struct {
	Channels []string `json:"channels"`
}

// UnsubscribeMessage 구독 취소 메시지 데이터
type UnsubscribeMessage struct {
	Channels []string `json:"channels"`
}

// LogMessage 로그 메시지 데이터
type LogMessage struct {
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Source    string    `json:"source"`
	SessionID string    `json:"session_id,omitempty"`
	TaskID    string    `json:"task_id,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// StatusMessage 상태 메시지 데이터
type StatusMessage struct {
	Resource string                 `json:"resource"` // session, task, workspace 등
	ID       string                 `json:"id"`
	Status   string                 `json:"status"`
	Data     map[string]interface{} `json:"data,omitempty"`
}

// EventMessage 이벤트 메시지 데이터
type EventMessage struct {
	Type      string                 `json:"type"`
	Source    string                 `json:"source"`
	SessionID string                 `json:"session_id,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// CommandMessage 명령 메시지 데이터
type CommandMessage struct {
	Command   string            `json:"command"`
	SessionID string            `json:"session_id"`
	Args      map[string]string `json:"args,omitempty"`
}

// TaskMessage 태스크 메시지 데이터
type TaskMessage struct {
	TaskID    string                 `json:"task_id"`
	SessionID string                 `json:"session_id"`
	Status    string                 `json:"status"`
	Output    string                 `json:"output,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// SessionMessage 세션 메시지 데이터
type SessionMessage struct {
	SessionID string                 `json:"session_id"`
	ProjectID string                 `json:"project_id"`
	Status    string                 `json:"status"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// NewMessage 새 메시지 생성
func NewMessage(msgType MessageType, data interface{}) *Message {
	dataBytes, _ := json.Marshal(data)
	return &Message{
		Type:      msgType,
		Data:      dataBytes,
		Timestamp: time.Now(),
	}
}

// NewErrorMessage 에러 메시지 생성
func NewErrorMessage(code, message, details string) *Message {
	return NewMessage(MessageTypeError, ErrorMessage{
		Code:    code,
		Message: message,
		Details: details,
	})
}

// NewSuccessMessage 성공 메시지 생성
func NewSuccessMessage(message string, data interface{}) *Message {
	return NewMessage(MessageTypeSuccess, SuccessMessage{
		Message: message,
		Data:    data,
	})
}

// NewLogMessage 로그 메시지 생성
func NewLogMessage(level, message, source, sessionID, taskID string) *Message {
	return NewMessage(MessageTypeLog, LogMessage{
		Level:     level,
		Message:   message,
		Source:    source,
		SessionID: sessionID,
		TaskID:    taskID,
		Timestamp: time.Now(),
	})
}

// NewStatusMessage 상태 메시지 생성
func NewStatusMessage(resource, id, status string, data map[string]interface{}) *Message {
	return NewMessage(MessageTypeStatus, StatusMessage{
		Resource: resource,
		ID:       id,
		Status:   status,
		Data:     data,
	})
}

// NewTaskMessage 태스크 메시지 생성
func NewTaskMessage(taskID, sessionID, status, output, error string, data map[string]interface{}) *Message {
	return NewMessage(MessageTypeTask, TaskMessage{
		TaskID:    taskID,
		SessionID: sessionID,
		Status:    status,
		Output:    output,
		Error:     error,
		Data:      data,
	})
}

// NewSessionMessage 세션 메시지 생성
func NewSessionMessage(sessionID, projectID, status string, data map[string]interface{}) *Message {
	return NewMessage(MessageTypeSession, SessionMessage{
		SessionID: sessionID,
		ProjectID: projectID,
		Status:    status,
		Data:      data,
	})
}

// WithChannel 메시지에 채널 설정
func (m *Message) WithChannel(channel string) *Message {
	m.Channel = channel
	return m
}

// WithID 메시지에 ID 설정
func (m *Message) WithID(id string) *Message {
	m.ID = id
	return m
}

// WithUserID 메시지에 사용자 ID 설정
func (m *Message) WithUserID(userID string) *Message {
	m.UserID = userID
	return m
}

// IsSystemMessage 시스템 메시지인지 확인
func (m *Message) IsSystemMessage() bool {
	switch m.Type {
	case MessageTypeAuth, MessageTypePing, MessageTypePong, MessageTypeError, MessageTypeSuccess,
		 MessageTypeSubscribe, MessageTypeUnsubscribe:
		return true
	default:
		return false
	}
}

// IsBusinessMessage 비즈니스 메시지인지 확인
func (m *Message) IsBusinessMessage() bool {
	return !m.IsSystemMessage()
}

// ToJSON 메시지를 JSON으로 변환
func (m *Message) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}

// ParseMessage JSON에서 메시지 파싱
func ParseMessage(data []byte) (*Message, error) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

// ParseAuthMessage 인증 메시지 데이터 파싱
func (m *Message) ParseAuthMessage() (*AuthMessage, error) {
	var auth AuthMessage
	if err := json.Unmarshal(m.Data, &auth); err != nil {
		return nil, err
	}
	return &auth, nil
}

// ParseSubscribeMessage 구독 메시지 데이터 파싱
func (m *Message) ParseSubscribeMessage() (*SubscribeMessage, error) {
	var sub SubscribeMessage
	if err := json.Unmarshal(m.Data, &sub); err != nil {
		return nil, err
	}
	return &sub, nil
}

// ParseUnsubscribeMessage 구독 취소 메시지 데이터 파싱
func (m *Message) ParseUnsubscribeMessage() (*UnsubscribeMessage, error) {
	var unsub UnsubscribeMessage
	if err := json.Unmarshal(m.Data, &unsub); err != nil {
		return nil, err
	}
	return &unsub, nil
}

// ParseCommandMessage 명령 메시지 데이터 파싱
func (m *Message) ParseCommandMessage() (*CommandMessage, error) {
	var cmd CommandMessage
	if err := json.Unmarshal(m.Data, &cmd); err != nil {
		return nil, err
	}
	return &cmd, nil
}

// Channel 상수들
const (
	ChannelWorkspace = "workspace"  // 워크스페이스 채널
	ChannelSession   = "session"    // 세션 채널
	ChannelTask      = "task"       // 태스크 채널
	ChannelSystem    = "system"     // 시스템 채널
	ChannelBroadcast = "broadcast"  // 브로드캐스트 채널
)

// GetWorkspaceChannel 워크스페이스 채널명 생성
func GetWorkspaceChannel(workspaceID string) string {
	return ChannelWorkspace + ":" + workspaceID
}

// GetSessionChannel 세션 채널명 생성
func GetSessionChannel(sessionID string) string {
	return ChannelSession + ":" + sessionID
}

// GetTaskChannel 태스크 채널명 생성
func GetTaskChannel(taskID string) string {
	return ChannelTask + ":" + taskID
}

// GetUserChannel 사용자 채널명 생성
func GetUserChannel(userID string) string {
	return "user:" + userID
}