package server

import (
	"encoding/json"
	
	"github.com/aicli/aicli-web/internal/claude"
	"github.com/aicli/aicli-web/internal/websocket"
)

// MessageBroadcasterAdapter는 websocket.Hub를 claude.MessageBroadcaster로 변환하는 어댑터입니다
type MessageBroadcasterAdapter struct {
	hub *websocket.Hub
}

// NewMessageBroadcasterAdapter 새로운 어댑터를 생성합니다
func NewMessageBroadcasterAdapter(hub *websocket.Hub) claude.MessageBroadcaster {
	return &MessageBroadcasterAdapter{
		hub: hub,
	}
}

// Broadcast 메시지를 브로드캐스트합니다
func (adapter *MessageBroadcasterAdapter) Broadcast(message interface{}) {
	if adapter.hub == nil {
		return
	}
	
	// interface{}를 websocket.Message로 변환
	wsMessage := adapter.convertToWebSocketMessage(message)
	if wsMessage != nil {
		adapter.hub.Broadcast(wsMessage)
	}
}

// BroadcastToWorkspace 특정 워크스페이스에 메시지를 브로드캐스트합니다
func (adapter *MessageBroadcasterAdapter) BroadcastToWorkspace(workspaceID string, message interface{}) {
	if adapter.hub == nil {
		return
	}
	
	// interface{}를 websocket.Message로 변환
	wsMessage := adapter.convertToWebSocketMessage(message)
	if wsMessage != nil {
		// 워크스페이스별 채널로 브로드캐스트
		channelName := "workspace:" + workspaceID
		adapter.hub.Broadcast(wsMessage, channelName)
	}
}

// convertToWebSocketMessage interface{}를 websocket.Message로 변환합니다
func (adapter *MessageBroadcasterAdapter) convertToWebSocketMessage(message interface{}) *websocket.Message {
	// JSON으로 직렬화 가능한지 확인
	data, err := json.Marshal(message)
	if err != nil {
		return nil
	}
	
	// websocket.Message 생성
	return &websocket.Message{
		Type: "claude_execution",
		Data: data,
	}
}