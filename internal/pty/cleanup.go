package pty

import (
	"context"
	"sync"
	"time"
	
	log "github.com/sirupsen/logrus"
)

// CleanupManager 정리 작업 관리자
type CleanupManager struct {
	sessions     PTYSessionInterface // 세션 관리자 인터페이스
	config       *CleanupConfig      // 정리 설정
	stopCh       chan struct{}       // 종료 채널
	wg           sync.WaitGroup      // 대기 그룹
	
	// 메트릭
	totalCleaned uint64              // 총 정리된 세션 수
	lastCleanup  time.Time           // 마지막 정리 시간
}

// CleanupConfig 정리 설정
type CleanupConfig struct {
	IdleTimeout      time.Duration // 유휴 타임아웃
	CleanupInterval  time.Duration // 정리 주기
	MaxSessionAge    time.Duration // 최대 세션 수명
	BatchSize        int           // 배치 크기
	GracePeriod      time.Duration // 유예 기간
}

// DefaultCleanupConfig 기본 정리 설정
func DefaultCleanupConfig() *CleanupConfig {
	return &CleanupConfig{
		IdleTimeout:     15 * time.Minute,
		CleanupInterval: 1 * time.Minute,
		MaxSessionAge:   24 * time.Hour,
		BatchSize:       10,
		GracePeriod:     30 * time.Second,
	}
}

// NewCleanupManager 새 정리 관리자 생성
func NewCleanupManager(sessions PTYSessionInterface, config *CleanupConfig) *CleanupManager {
	if config == nil {
		config = DefaultCleanupConfig()
	}

	return &CleanupManager{
		sessions: sessions,
		config:   config,
		stopCh:   make(chan struct{}),
	}
}

// Start 정리 관리자 시작
func (cm *CleanupManager) Start() {
	cm.wg.Add(1)
	go cm.cleanupLoop()
	
	log.Info("Cleanup manager started")
}

// Stop 정리 관리자 중지
func (cm *CleanupManager) Stop() {
	close(cm.stopCh)
	cm.wg.Wait()
	
	log.Info("Cleanup manager stopped")
}

// cleanupLoop 정리 루프
func (cm *CleanupManager) cleanupLoop() {
	defer cm.wg.Done()

	ticker := time.NewTicker(cm.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cm.performCleanup()
			
		case <-cm.stopCh:
			// 마지막 정리 수행
			cm.performFinalCleanup()
			return
		}
	}
}

// performCleanup 정리 수행
func (cm *CleanupManager) performCleanup() {
	startTime := time.Now()
	
	// 유휴 세션 정리
	cleaned := cm.sessions.CleanupIdleSessions(cm.config.IdleTimeout)
	
	// 오래된 세션 정리
	cleaned += cm.cleanupOldSessions()
	
	// 메트릭 업데이트
	cm.totalCleaned += uint64(cleaned)
	cm.lastCleanup = time.Now()
	
	if cleaned > 0 {
		log.Infof("Cleanup completed: %d sessions cleaned (took %v)", 
			cleaned, time.Since(startTime))
	}
}

// cleanupOldSessions 오래된 세션 정리
func (cm *CleanupManager) cleanupOldSessions() int {
	sessions := cm.sessions.ListSessions()
	if len(sessions) == 0 {
		return 0
	}

	var toCleanup []string
	now := time.Now()

	for id, session := range sessions {
		// 최대 수명 확인
		if cm.config.MaxSessionAge > 0 {
			if now.Sub(session.CreatedAt) > cm.config.MaxSessionAge {
				toCleanup = append(toCleanup, id)
				log.Debugf("Session %s marked for cleanup (age: %v)", 
					id, now.Sub(session.CreatedAt))
			}
		}
	}

	// 배치로 정리
	cleaned := 0
	for i := 0; i < len(toCleanup); i += cm.config.BatchSize {
		end := i + cm.config.BatchSize
		if end > len(toCleanup) {
			end = len(toCleanup)
		}

		batch := toCleanup[i:end]
		for _, id := range batch {
			if err := cm.sessions.CloseSession(id); err != nil {
				log.Errorf("Failed to cleanup session %s: %v", id, err)
			} else {
				cleaned++
			}
		}

		// 배치 간 잠시 대기
		if end < len(toCleanup) {
			time.Sleep(100 * time.Millisecond)
		}
	}

	return cleaned
}

// performFinalCleanup 최종 정리 수행
func (cm *CleanupManager) performFinalCleanup() {
	log.Info("Performing final cleanup")
	
	sessions := cm.sessions.ListSessions()
	
	// 유예 기간 부여
	if cm.config.GracePeriod > 0 {
		log.Infof("Grace period of %v before final cleanup", cm.config.GracePeriod)
		time.Sleep(cm.config.GracePeriod)
	}
	
	// 모든 세션 종료
	for id := range sessions {
		if err := cm.sessions.CloseSession(id); err != nil {
			log.Errorf("Failed to cleanup session %s during final cleanup: %v", id, err)
		}
	}
	
	log.Infof("Final cleanup completed: %d sessions closed", len(sessions))
}

// GetStats 통계 조회
func (cm *CleanupManager) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"total_cleaned":   cm.totalCleaned,
		"last_cleanup":    cm.lastCleanup,
		"idle_timeout":    cm.config.IdleTimeout.String(),
		"cleanup_interval": cm.config.CleanupInterval.String(),
		"max_session_age": cm.config.MaxSessionAge.String(),
	}
}

// ForceCleanup 강제 정리
func (cm *CleanupManager) ForceCleanup() int {
	log.Info("Force cleanup requested")
	return cm.performCleanup()
}

// CleanupSession 특정 세션 정리
func (cm *CleanupManager) CleanupSession(sessionID string) error {
	return cm.sessions.CloseSession(sessionID)
}

// ScheduleCleanup 정리 예약
func (cm *CleanupManager) ScheduleCleanup(sessionID string, delay time.Duration) {
	go func() {
		time.Sleep(delay)
		
		select {
		case <-cm.stopCh:
			return
		default:
			if err := cm.CleanupSession(sessionID); err != nil {
				log.Errorf("Scheduled cleanup failed for session %s: %v", sessionID, err)
			} else {
				log.Debugf("Scheduled cleanup completed for session %s", sessionID)
			}
		}
	}()
}

// SessionCleaner 세션별 정리기
type SessionCleaner struct {
	sessionID string
	session   *PTYSession
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewSessionCleaner 새 세션 정리기 생성
func NewSessionCleaner(sessionID string, session *PTYSession) *SessionCleaner {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &SessionCleaner{
		sessionID: sessionID,
		session:   session,
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Cleanup 세션 정리
func (sc *SessionCleaner) Cleanup() error {
	// 취소 함수 호출
	if sc.cancel != nil {
		sc.cancel()
	}
	
	// 세션 종료
	if sc.session != nil {
		return sc.session.Terminate()
	}
	
	return nil
}

// CleanupWithTimeout 타임아웃과 함께 정리
func (sc *SessionCleaner) CleanupWithTimeout(timeout time.Duration) error {
	done := make(chan error, 1)
	
	go func() {
		done <- sc.Cleanup()
	}()
	
	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		// 강제 종료
		if sc.session != nil {
			sc.session.Terminate()
		}
		return nil
	}
}