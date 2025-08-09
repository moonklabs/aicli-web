package pty_streaming

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// isTemporaryError checks if the error is temporary
func isTemporaryError(err error) bool {
	if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
		return false
	}
	return websocket.IsCloseError(err, websocket.CloseAbnormalClosure)
}

// SendCommand sends a command to the PTY session
func (tc *TestClient) SendCommand(command string) error {
	tc.mutex.Lock()
	defer tc.mutex.Unlock()

	if !tc.connected {
		return fmt.Errorf("client not connected")
	}

	message := map[string]interface{}{
		"type": "input",
		"data": command + "\n",
	}

	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	if err := tc.wsConn.WriteMessage(websocket.TextMessage, data); err != nil {
		tc.errorCount++
		return fmt.Errorf("failed to send command: %w", err)
	}

	tc.messageCount++
	tc.lastMessage = time.Now()
	return nil
}

// WaitForOutput waits for specific output from the PTY session
func (tc *TestClient) WaitForOutput(expected string, timeout time.Duration) (string, error) {
	tc.mutex.RLock()
	if !tc.connected {
		tc.mutex.RUnlock()
		return "", fmt.Errorf("client not connected")
	}
	tc.mutex.RUnlock()

	deadline := time.Now().Add(timeout)
	var output strings.Builder

	for time.Now().Before(deadline) {
		tc.wsConn.SetReadDeadline(time.Now().Add(1 * time.Second))

		_, message, err := tc.wsConn.ReadMessage()
		if err != nil {
			// 임시 오류 검사를 대체하는 로직
			if isTemporaryError(err) {
				continue
			}
			tc.mutex.Lock()
			tc.errorCount++
			tc.mutex.Unlock()
			return output.String(), fmt.Errorf("failed to read message: %w", err)
		}

		var msg map[string]interface{}
		if err := json.Unmarshal(message, &msg); err != nil {
			// 일반 텍스트 메시지일 수도 있음
			output.WriteString(string(message))
		} else if msgType, ok := msg["type"].(string); ok && msgType == "output" {
			if data, ok := msg["data"].(string); ok {
				output.WriteString(data)
			}
		}

		tc.mutex.Lock()
		tc.messageCount++
		tc.lastMessage = time.Now()
		tc.mutex.Unlock()

		if strings.Contains(output.String(), expected) {
			return output.String(), nil
		}
	}

	return output.String(), fmt.Errorf("timeout waiting for output: expected '%s', got '%s'", expected, output.String())
}

// Close closes the client connection
func (tc *TestClient) Close() error {
	tc.mutex.Lock()
	defer tc.mutex.Unlock()

	if !tc.connected {
		return nil
	}

	tc.connected = false
	if tc.wsConn != nil {
		return tc.wsConn.Close()
	}
	return nil
}

// ForceDisconnect forcefully disconnects the client
func (tc *TestClient) ForceDisconnect() {
	tc.mutex.Lock()
	defer tc.mutex.Unlock()

	tc.connected = false
	if tc.wsConn != nil {
		tc.wsConn.Close()
	}
}

// GetStats returns client statistics
func (tc *TestClient) GetStats() ClientStats {
	tc.mutex.RLock()
	defer tc.mutex.RUnlock()

	return ClientStats{
		ClientID:     tc.clientID,
		SessionID:    tc.sessionID,
		Connected:    tc.connected,
		MessageCount: tc.messageCount,
		ErrorCount:   tc.errorCount,
		LastMessage:  tc.lastMessage,
	}
}

// ClientStats represents client statistics
type ClientStats struct {
	ClientID     string
	SessionID    string
	Connected    bool
	MessageCount int64
	ErrorCount   int64
	LastMessage  time.Time
}

// TestSession represents a test session
type TestSession struct {
	ContainerID string
	SessionID   string
	Client      *TestClient
	mutex       sync.RWMutex
}

// Execute executes a command and waits for output
func (ts *TestSession) Execute(command string, expectedOutput string, timeout time.Duration) error {
	if err := ts.Client.SendCommand(command); err != nil {
		return fmt.Errorf("failed to send command: %w", err)
	}

	output, err := ts.Client.WaitForOutput(expectedOutput, timeout)
	if err != nil {
		return fmt.Errorf("command execution failed: %w", err)
	}

	if !strings.Contains(output, expectedOutput) {
		return fmt.Errorf("unexpected output: expected '%s', got '%s'", expectedOutput, output)
	}

	return nil
}

// Close closes the test session
func (ts *TestSession) Close() error {
	ts.mutex.Lock()
	defer ts.mutex.Unlock()

	if ts.Client != nil {
		return ts.Client.Close()
	}
	return nil
}