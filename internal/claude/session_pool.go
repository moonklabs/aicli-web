package claude

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// SessionPool은 세션 풀을 관리합니다
type SessionPool struct {
	manager       SessionManager
	maxSessions   int
	maxIdleTime   time.Duration
	maxLifetime   time.Duration
	cleanupTicker *time.Ticker
	sessions      map[string]*PooledSession
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
}

// PooledSession은 풀에서 관리되는 세션입니다
type PooledSession struct {
	*Session
	inUse      bool
	lastUsed   time.Time
	useCount   int64
	poolReturn chan struct{} // 세션 반환 신호
}

// SessionPoolConfig는 세션 풀 설정입니다
type SessionPoolConfig struct {
	MaxSessions int           // 최대 세션 수
	MaxIdleTime time.Duration // 최대 유휴 시간
	MaxLifetime time.Duration // 세션 최대 수명
	CleanupInterval time.Duration // 정리 주기
}

// DefaultSessionPoolConfig는 기본 세션 풀 설정을 반환합니다
func DefaultSessionPoolConfig() SessionPoolConfig {
	return SessionPoolConfig{
		MaxSessions:     10,
		MaxIdleTime:     30 * time.Minute,
		MaxLifetime:     4 * time.Hour,
		CleanupInterval: 5 * time.Minute,
	}
}

// NewSessionPool은 새로운 세션 풀을 생성합니다
func NewSessionPool(manager SessionManager, config SessionPoolConfig) *SessionPool {
	ctx, cancel := context.WithCancel(context.Background())
	
	pool := &SessionPool{
		manager:     manager,
		maxSessions: config.MaxSessions,
		maxIdleTime: config.MaxIdleTime,
		maxLifetime: config.MaxLifetime,
		sessions:    make(map[string]*PooledSession),
		ctx:         ctx,
		cancel:      cancel,
	}

	// 정리 작업 시작
	if config.CleanupInterval > 0 {
		pool.cleanupTicker = time.NewTicker(config.CleanupInterval)
		go pool.cleanupLoop()
	}

	return pool
}

// AcquireSession은 풀에서 세션을 가져옵니다
func (p *SessionPool) AcquireSession(ctx context.Context, config SessionConfig) (*PooledSession, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 재사용 가능한 세션 찾기
	for _, pooledSession := range p.sessions {
		if !pooledSession.inUse && p.isCompatible(pooledSession.Session, config) {
			// 세션이 여전히 유효한지 확인
			if p.isSessionValid(pooledSession) {
				pooledSession.inUse = true
				pooledSession.lastUsed = time.Now()
				pooledSession.useCount++
				
				// 상태를 Active로 변경
				if err := p.manager.UpdateSession(pooledSession.ID, SessionUpdate{
					State: &[]SessionState{SessionStateActive}[0],
				}); err == nil {
					return pooledSession, nil
				}
			} else {
				// 유효하지 않은 세션은 제거
				p.removeSessionLocked(pooledSession.ID)
			}
		}
	}

	// 새 세션 생성 가능 여부 확인
	if len(p.sessions) >= p.maxSessions {
		// 가장 오래된 유휴 세션 찾기
		var oldestIdle *PooledSession
		var oldestTime time.Time
		
		for _, s := range p.sessions {
			if !s.inUse && (oldestIdle == nil || s.lastUsed.Before(oldestTime)) {
				oldestIdle = s
				oldestTime = s.lastUsed
			}
		}
		
		if oldestIdle != nil {
			// 오래된 유휴 세션 제거
			p.removeSessionLocked(oldestIdle.ID)
		} else {
			return nil, errors.New("session pool is full")
		}
	}

	// 새 세션 생성
	session, err := p.manager.CreateSession(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	pooledSession := &PooledSession{
		Session:    session,
		inUse:      true,
		lastUsed:   time.Now(),
		useCount:   1,
		poolReturn: make(chan struct{}, 1),
	}

	p.sessions[session.ID] = pooledSession
	return pooledSession, nil
}

// ReleaseSession은 세션을 풀에 반환합니다
func (p *SessionPool) ReleaseSession(sessionID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	pooledSession, exists := p.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found in pool: %s", sessionID)
	}

	if !pooledSession.inUse {
		return fmt.Errorf("session is not in use: %s", sessionID)
	}

	pooledSession.inUse = false
	pooledSession.lastUsed = time.Now()

	// 상태를 Idle로 변경
	idleState := SessionStateIdle
	if err := p.manager.UpdateSession(sessionID, SessionUpdate{
		State: &idleState,
	}); err != nil {
		// 에러가 발생하면 세션 제거
		p.removeSessionLocked(sessionID)
		return fmt.Errorf("failed to update session state: %w", err)
	}

	// 반환 신호 전송 (비블로킹)
	select {
	case pooledSession.poolReturn <- struct{}{}:
	default:
	}

	return nil
}

// RemoveSession은 풀에서 세션을 제거합니다
func (p *SessionPool) RemoveSession(sessionID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.removeSessionLocked(sessionID)
}

// removeSessionLocked은 락이 걸린 상태에서 세션을 제거합니다
func (p *SessionPool) removeSessionLocked(sessionID string) error {
	pooledSession, exists := p.sessions[sessionID]
	if !exists {
		return nil // 이미 제거됨
	}

	// 세션 종료
	if err := p.manager.CloseSession(sessionID); err != nil {
		// 에러를 로그하지만 계속 진행
		fmt.Printf("Failed to close session %s: %v\n", sessionID, err)
	}

	// 풀에서 제거
	delete(p.sessions, sessionID)

	// 반환 채널 닫기
	close(pooledSession.poolReturn)

	return nil
}

// GetPoolStats는 풀 통계를 반환합니다
func (p *SessionPool) GetPoolStats() PoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := PoolStats{
		TotalSessions:  len(p.sessions),
		ActiveSessions: 0,
		IdleSessions:   0,
	}

	for _, s := range p.sessions {
		if s.inUse {
			stats.ActiveSessions++
		} else {
			stats.IdleSessions++
		}
	}

	return stats
}

// PoolStats는 세션 풀 통계입니다
type PoolStats struct {
	TotalSessions  int `json:"total_sessions"`
	ActiveSessions int `json:"active_sessions"`
	IdleSessions   int `json:"idle_sessions"`
}

// Shutdown은 세션 풀을 종료합니다
func (p *SessionPool) Shutdown() error {
	p.cancel()

	if p.cleanupTicker != nil {
		p.cleanupTicker.Stop()
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// 모든 세션 종료
	var lastErr error
	for sessionID := range p.sessions {
		if err := p.removeSessionLocked(sessionID); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// cleanupLoop는 주기적으로 유효하지 않은 세션을 정리합니다
func (p *SessionPool) cleanupLoop() {
	for {
		select {
		case <-p.ctx.Done():
			return
		case <-p.cleanupTicker.C:
			p.cleanup()
		}
	}
}

// cleanup은 유효하지 않은 세션을 정리합니다
func (p *SessionPool) cleanup() {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	toRemove := []string{}

	for sessionID, pooledSession := range p.sessions {
		// 사용 중인 세션은 건너뛰기
		if pooledSession.inUse {
			continue
		}

		// 유휴 시간 초과 확인
		if p.maxIdleTime > 0 && now.Sub(pooledSession.lastUsed) > p.maxIdleTime {
			toRemove = append(toRemove, sessionID)
			continue
		}

		// 수명 초과 확인
		if p.maxLifetime > 0 && now.Sub(pooledSession.Created) > p.maxLifetime {
			toRemove = append(toRemove, sessionID)
			continue
		}

		// 세션 상태 확인
		if !p.isSessionValid(pooledSession) {
			toRemove = append(toRemove, sessionID)
		}
	}

	// 세션 제거
	for _, sessionID := range toRemove {
		p.removeSessionLocked(sessionID)
	}
}

// isSessionValid는 세션이 유효한지 확인합니다
func (p *SessionPool) isSessionValid(pooledSession *PooledSession) bool {
	// 프로세스가 살아있는지 확인
	if pooledSession.Process != nil {
		if process, err := p.manager.(*sessionManager).processManager.GetProcess(pooledSession.Process.ID); err != nil || process == nil {
			return false
		}
	}

	// 세션 상태 확인
	state := pooledSession.State
	return state != SessionStateClosed && state != SessionStateError && state != SessionStateClosing
}

// isCompatible은 세션이 요청된 설정과 호환되는지 확인합니다
func (p *SessionPool) isCompatible(session *Session, config SessionConfig) bool {
	// 작업 디렉토리 확인
	if session.Config.WorkingDir != config.WorkingDir {
		return false
	}

	// 시스템 프롬프트 확인
	if session.Config.SystemPrompt != config.SystemPrompt {
		return false
	}

	// 도구 권한 확인
	if len(session.Config.AllowedTools) != len(config.AllowedTools) {
		return false
	}
	
	toolMap := make(map[string]bool)
	for _, tool := range session.Config.AllowedTools {
		toolMap[tool] = true
	}
	for _, tool := range config.AllowedTools {
		if !toolMap[tool] {
			return false
		}
	}

	// 환경 변수 확인 (중요한 것만)
	for k, v := range config.Environment {
		if existing, ok := session.Config.Environment[k]; ok && existing != v {
			return false
		}
	}

	return true
}

// SetMaxSessions은 최대 세션 수를 설정합니다
func (p *SessionPool) SetMaxSessions(max int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.maxSessions = max
}

// SetMaxIdleTime은 최대 유휴 시간을 설정합니다
func (p *SessionPool) SetMaxIdleTime(duration time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.maxIdleTime = duration
}

// SetMaxLifetime은 세션 최대 수명을 설정합니다
func (p *SessionPool) SetMaxLifetime(duration time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.maxLifetime = duration
}