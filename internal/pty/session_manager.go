package pty

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("component", "pty")

// SessionConfig 세션 관리자 설정
type SessionConfig struct {
	MaxSessions      int           // 최대 세션 수
	IdleTimeout      time.Duration // 유휴 타임아웃
	CleanupInterval  time.Duration // 정리 주기
	MaxSessionAge    time.Duration // 최대 세션 수명
	EnablePooling    bool          // 세션 풀링 활성화
	PoolSize         int           // 풀 크기
}

// DefaultSessionConfig 기본 세션 설정
func DefaultSessionConfig() *SessionConfig {
	return &SessionConfig{
		MaxSessions:     100,
		IdleTimeout:     15 * time.Minute,
		CleanupInterval: 1 * time.Minute,
		MaxSessionAge:   24 * time.Hour,
		EnablePooling:   true,
		PoolSize:        10,
	}
}

// SessionManager PTY 세션 관리자
type SessionManager struct {
	sessions    map[string]*PTYSession // 활성 세션 맵
	mutex       sync.RWMutex           // 세션 맵 보호
	config      *SessionConfig         // 관리자 설정
	cleanup     chan string            // 정리 요청 채널
	stopCh      chan struct{}          // 종료 신호 채널
	wg          sync.WaitGroup         // 고루틴 대기 그룹
	pool        *SessionPool           // 세션 풀
	
	// 메트릭
	totalCreated   uint64 // 총 생성된 세션 수
	totalClosed    uint64 // 총 종료된 세션 수
	totalRecycled  uint64 // 총 재활용된 세션 수
}

// NewSessionManager 새 세션 관리자 생성
func NewSessionManager(config *SessionConfig) *SessionManager {
	if config == nil {
		config = DefaultSessionConfig()
	}

	sm := &SessionManager{
		sessions: make(map[string]*PTYSession),
		config:   config,
		cleanup:  make(chan string, 100),
		stopCh:   make(chan struct{}),
	}

	// 세션 풀 초기화
	if config.EnablePooling {
		sm.pool = NewSessionPool(config.PoolSize)
	}

	// 정리 워커 시작
	sm.startCleanupWorker()

	log.Info("Session manager initialized")
	return sm
}

// CreateSession 새 세션 생성
func (sm *SessionManager) CreateSession(ctx context.Context, containerID string, config *PTYConfig) (*PTYSession, error) {
	// 세션 수 제한 확인
	sm.mutex.RLock()
	sessionCount := len(sm.sessions)
	sm.mutex.RUnlock()

	if sessionCount >= sm.config.MaxSessions {
		return nil, fmt.Errorf("maximum number of sessions (%d) reached", sm.config.MaxSessions)
	}

	// 세션 생성 (풀에서 가져오거나 새로 생성)
	var session *PTYSession
	if sm.pool != nil {
		session = sm.pool.Get()
		if session != nil {
			// 재활용된 세션 초기화
			session.ContainerID = containerID
			session.Config = config
			session.CreatedAt = time.Now()
			session.LastActive = time.Now()
			session.Status = SessionActive
			sm.totalRecycled++
			log.Debugf("Recycled session from pool: %s", session.ID)
		}
	}

	if session == nil {
		// 새 세션 생성
		session = NewPTYSession(containerID, config)
		sm.totalCreated++
		log.Debugf("Created new session: %s", session.ID)
	}

	// 컨텍스트 취소 함수 설정
	ctx, cancel := context.WithCancel(ctx)
	session.SetCancel(cancel)

	// 세션 맵에 추가
	sm.mutex.Lock()
	sm.sessions[session.ID] = session
	sm.mutex.Unlock()

	log.Infof("Session created: %s for container %s", session.ID, containerID)
	return session, nil
}

// GetSession 세션 조회
func (sm *SessionManager) GetSession(sessionID string) (*PTYSession, error) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	// 활동 시간 업데이트
	session.UpdateActivity()

	return session, nil
}

// CloseSession 세션 종료
func (sm *SessionManager) CloseSession(sessionID string) error {
	sm.mutex.Lock()
	session, exists := sm.sessions[sessionID]
	if !exists {
		sm.mutex.Unlock()
		return fmt.Errorf("session not found: %s", sessionID)
	}

	// 세션 맵에서 제거
	delete(sm.sessions, sessionID)
	sm.mutex.Unlock()

	// 세션 종료
	if err := session.Terminate(); err != nil {
		log.Errorf("Failed to terminate session %s: %v", sessionID, err)
		return err
	}

	// 풀에 반환 (가능한 경우)
	if sm.pool != nil && sm.pool.CanReturn() {
		// 세션 초기화 후 풀에 반환
		session.ContainerID = ""
		session.metadata = make(map[string]interface{})
		sm.pool.Return(session)
		log.Debugf("Returned session to pool: %s", sessionID)
	}

	sm.totalClosed++
	log.Infof("Session closed: %s", sessionID)
	return nil
}

// ListSessions 모든 세션 목록 조회
func (sm *SessionManager) ListSessions() map[string]*PTYSession {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	// 세션 맵 복사
	sessions := make(map[string]*PTYSession, len(sm.sessions))
	for id, session := range sm.sessions {
		sessions[id] = session
	}

	return sessions
}

// CleanupIdleSessions 유휴 세션 정리
func (sm *SessionManager) CleanupIdleSessions(timeout time.Duration) int {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	var toRemove []string
	now := time.Now()

	for id, session := range sm.sessions {
		// 유휴 시간 확인
		if session.GetIdleTime() > timeout {
			toRemove = append(toRemove, id)
			log.Debugf("Session %s marked for cleanup (idle for %v)", id, session.GetIdleTime())
		}

		// 최대 수명 확인
		if sm.config.MaxSessionAge > 0 && now.Sub(session.CreatedAt) > sm.config.MaxSessionAge {
			if !contains(toRemove, id) {
				toRemove = append(toRemove, id)
				log.Debugf("Session %s marked for cleanup (age: %v)", id, now.Sub(session.CreatedAt))
			}
		}
	}

	// 세션 정리
	for _, id := range toRemove {
		session := sm.sessions[id]
		delete(sm.sessions, id)

		// 비동기 종료
		go func(s *PTYSession) {
			if err := s.Terminate(); err != nil {
				log.Errorf("Failed to terminate session during cleanup: %v", err)
			}
		}(session)
	}

	if len(toRemove) > 0 {
		log.Infof("Cleaned up %d idle sessions", len(toRemove))
	}

	return len(toRemove)
}

// startCleanupWorker 정리 워커 시작
func (sm *SessionManager) startCleanupWorker() {
	sm.wg.Add(1)
	go func() {
		defer sm.wg.Done()

		ticker := time.NewTicker(sm.config.CleanupInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// 정기 정리
				sm.cleanupExpiredSessions()

			case sessionID := <-sm.cleanup:
				// 강제 정리 요청
				if err := sm.CloseSession(sessionID); err != nil {
					log.Errorf("Failed to cleanup session %s: %v", sessionID, err)
				}

			case <-sm.stopCh:
				// 종료 신호
				log.Info("Cleanup worker stopping")
				return
			}
		}
	}()

	log.Info("Cleanup worker started")
}

// cleanupExpiredSessions 만료된 세션 정리
func (sm *SessionManager) cleanupExpiredSessions() {
	cleaned := sm.CleanupIdleSessions(sm.config.IdleTimeout)
	if cleaned > 0 {
		log.Debugf("Periodic cleanup: removed %d sessions", cleaned)
	}

	// 풀 상태 확인
	if sm.pool != nil {
		sm.pool.Cleanup()
	}
}

// forceCleanupSession 세션 강제 정리
func (sm *SessionManager) forceCleanupSession(sessionID string) {
	if err := sm.CloseSession(sessionID); err != nil {
		log.Errorf("Failed to force cleanup session %s: %v", sessionID, err)
	}
}

// Shutdown 세션 관리자 종료
func (sm *SessionManager) Shutdown() error {
	log.Info("Shutting down session manager")

	// 종료 신호 전송
	close(sm.stopCh)

	// 모든 세션 종료
	sm.mutex.Lock()
	for id, session := range sm.sessions {
		if err := session.Terminate(); err != nil {
			log.Errorf("Failed to terminate session %s during shutdown: %v", id, err)
		}
	}
	sm.sessions = make(map[string]*PTYSession)
	sm.mutex.Unlock()

	// 워커 종료 대기
	sm.wg.Wait()

	// 풀 정리
	if sm.pool != nil {
		sm.pool.Destroy()
	}

	log.Info("Session manager shutdown complete")
	return nil
}

// GetStats 통계 조회
func (sm *SessionManager) GetStats() map[string]interface{} {
	sm.mutex.RLock()
	activeCount := len(sm.sessions)
	sm.mutex.RUnlock()

	poolStats := make(map[string]interface{})
	if sm.pool != nil {
		poolStats = sm.pool.GetStats()
	}

	return map[string]interface{}{
		"active_sessions":  activeCount,
		"total_created":    sm.totalCreated,
		"total_closed":     sm.totalClosed,
		"total_recycled":   sm.totalRecycled,
		"max_sessions":     sm.config.MaxSessions,
		"pool_stats":       poolStats,
	}
}

// RequestCleanup 세션 정리 요청
func (sm *SessionManager) RequestCleanup(sessionID string) {
	select {
	case sm.cleanup <- sessionID:
		log.Debugf("Cleanup requested for session: %s", sessionID)
	default:
		// 채널이 가득 찬 경우 즉시 정리
		go sm.forceCleanupSession(sessionID)
	}
}

// UpdateSessionActivity 세션 활동 업데이트
func (sm *SessionManager) UpdateSessionActivity(sessionID string) error {
	sm.mutex.RLock()
	session, exists := sm.sessions[sessionID]
	sm.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	session.UpdateActivity()
	return nil
}

// contains 문자열 슬라이스에 값이 포함되어 있는지 확인
func contains(slice []string, value string) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}