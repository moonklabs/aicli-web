package docker

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ReconnectionManager manages PTY reconnection logic
type ReconnectionManager struct {
	integration    *DockerPTYIntegration
	reconnectQueue chan *DockerPTYSession
	config         *ReconnectionConfig
	mutex          sync.RWMutex
	stopCh         chan struct{}
}

// ReconnectionConfig contains reconnection configuration
type ReconnectionConfig struct {
	MaxRetries      int
	InitialDelay    time.Duration
	MaxDelay        time.Duration
	BackoffFactor   float64
	QueueSize       int
	WorkerCount     int
}

// ReconnectionResult represents the result of a reconnection attempt
type ReconnectionResult struct {
	SessionID string
	Success   bool
	Attempts  int
	Error     error
	Timestamp time.Time
}

// NewReconnectionManager creates a new reconnection manager
func NewReconnectionManager(integration *DockerPTYIntegration) *ReconnectionManager {
	config := &ReconnectionConfig{
		MaxRetries:    3,
		InitialDelay:  1 * time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
		QueueSize:     100,
		WorkerCount:   5,
	}

	return &ReconnectionManager{
		integration:    integration,
		reconnectQueue: make(chan *DockerPTYSession, config.QueueSize),
		config:         config,
		stopCh:         make(chan struct{}),
	}
}

// Start starts the reconnection manager
func (rm *ReconnectionManager) Start() {
	// 재연결 워커 시작
	for i := 0; i < rm.config.WorkerCount; i++ {
		go rm.reconnectionWorker(i)
	}
}

// Stop stops the reconnection manager
func (rm *ReconnectionManager) Stop() {
	close(rm.stopCh)
}

// QueueReconnection queues a session for reconnection
func (rm *ReconnectionManager) QueueReconnection(session *DockerPTYSession) error {
	select {
	case rm.reconnectQueue <- session:
		log.Infof("Session %s queued for reconnection", session.SessionID)
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("reconnection queue is full")
	}
}

// reconnectionWorker processes reconnection requests
func (rm *ReconnectionManager) reconnectionWorker(workerID int) {
	log.Infof("Reconnection worker %d started", workerID)

	for {
		select {
		case session := <-rm.reconnectQueue:
			result := rm.performReconnection(session)
			rm.handleReconnectionResult(result)

		case <-rm.stopCh:
			log.Infof("Reconnection worker %d stopped", workerID)
			return
		}
	}
}

// performReconnection attempts to reconnect a session
func (rm *ReconnectionManager) performReconnection(session *DockerPTYSession) *ReconnectionResult {
	result := &ReconnectionResult{
		SessionID: session.SessionID,
		Success:   false,
		Attempts:  0,
		Timestamp: time.Now(),
	}

	delay := rm.config.InitialDelay

	for attempt := 1; attempt <= rm.config.MaxRetries; attempt++ {
		result.Attempts = attempt
		log.Infof("Reconnection attempt %d/%d for session %s",
			attempt, rm.config.MaxRetries, session.SessionID)

		// 재연결 시도
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		newSession, err := rm.integration.ConnectContainer(ctx, session.ContainerID, session.Config)
		cancel()

		if err == nil {
			// 성공적으로 재연결
			if err := rm.replaceSession(session, newSession); err != nil {
				result.Error = fmt.Errorf("failed to replace session: %w", err)
				log.Errorf("Failed to replace session %s: %v", session.SessionID, err)
			} else {
				result.Success = true
				log.Infof("Successfully reconnected session %s after %d attempts",
					session.SessionID, attempt)
				return result
			}
		}

		log.Errorf("Reconnection attempt %d failed for session %s: %v",
			attempt, session.SessionID, err)
		result.Error = err

		// 마지막 시도가 아니면 대기
		if attempt < rm.config.MaxRetries {
			log.Infof("Waiting %v before next reconnection attempt for session %s",
				delay, session.SessionID)
			time.Sleep(delay)

			// 백오프 지연 증가
			delay = time.Duration(float64(delay) * rm.config.BackoffFactor)
			if delay > rm.config.MaxDelay {
				delay = rm.config.MaxDelay
			}
		}
	}

	return result
}

// replaceSession replaces an old session with a new one
func (rm *ReconnectionManager) replaceSession(oldSession, newSession *DockerPTYSession) error {
	// 기존 세션 정리
	rm.integration.cleanupSession(oldSession)

	// 새 세션 ID 유지
	newSession.SessionID = oldSession.SessionID

	// 세션 교체
	rm.integration.mutex.Lock()
	rm.integration.sessions[oldSession.SessionID] = newSession
	rm.integration.mutex.Unlock()

	return nil
}

// handleReconnectionResult handles the result of a reconnection attempt
func (rm *ReconnectionManager) handleReconnectionResult(result *ReconnectionResult) {
	if result.Success {
		log.Infof("Reconnection successful for session %s after %d attempts",
			result.SessionID, result.Attempts)
	} else {
		log.Errorf("Reconnection failed for session %s after %d attempts: %v",
			result.SessionID, result.Attempts, result.Error)

		// 실패한 세션 제거
		if err := rm.integration.DisconnectContainer(result.SessionID); err != nil {
			log.Errorf("Failed to disconnect failed session %s: %v",
				result.SessionID, err)
		}
	}
}

// GetQueueSize returns the current size of the reconnection queue
func (rm *ReconnectionManager) GetQueueSize() int {
	return len(rm.reconnectQueue)
}

// SetConfig updates the reconnection configuration
func (rm *ReconnectionManager) SetConfig(config *ReconnectionConfig) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()
	rm.config = config
}

// GetConfig returns the current reconnection configuration
func (rm *ReconnectionManager) GetConfig() *ReconnectionConfig {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()
	return rm.config
}