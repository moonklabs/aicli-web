package pty

import (
	"sync"
	"time"
	
	log "github.com/sirupsen/logrus"
)

// SessionPool 세션 풀 관리
type SessionPool struct {
	inactive    []*PTYSession  // 비활성 세션 풀
	maxSize     int            // 최대 풀 크기
	currentSize int            // 현재 풀 크기
	mutex       sync.Mutex     // 풀 보호
	
	// 메트릭
	hits        uint64         // 풀 히트 수
	misses      uint64         // 풀 미스 수
	returned    uint64         // 반환된 세션 수
}

// NewSessionPool 새 세션 풀 생성
func NewSessionPool(maxSize int) *SessionPool {
	return &SessionPool{
		inactive: make([]*PTYSession, 0, maxSize),
		maxSize:  maxSize,
	}
}

// Get 풀에서 세션 가져오기
func (sp *SessionPool) Get() *PTYSession {
	sp.mutex.Lock()
	defer sp.mutex.Unlock()

	if len(sp.inactive) == 0 {
		sp.misses++
		return nil
	}

	// 풀에서 세션 가져오기 (LIFO)
	session := sp.inactive[len(sp.inactive)-1]
	sp.inactive = sp.inactive[:len(sp.inactive)-1]
	sp.currentSize--
	sp.hits++

	log.Debugf("Got session from pool (pool size: %d/%d)", sp.currentSize, sp.maxSize)
	return session
}

// Return 세션을 풀에 반환
func (sp *SessionPool) Return(session *PTYSession) bool {
	sp.mutex.Lock()
	defer sp.mutex.Unlock()

	// 풀이 가득 찬 경우
	if sp.currentSize >= sp.maxSize {
		log.Debugf("Pool is full, discarding session (pool size: %d/%d)", sp.currentSize, sp.maxSize)
		return false
	}

	// 종료된 세션은 반환하지 않음
	if session.IsTerminated() {
		log.Debug("Cannot return terminated session to pool")
		return false
	}

	// 세션 초기화
	session.Status = SessionIdle
	session.LastActive = time.Now()
	
	// 풀에 추가
	sp.inactive = append(sp.inactive, session)
	sp.currentSize++
	sp.returned++

	log.Debugf("Returned session to pool (pool size: %d/%d)", sp.currentSize, sp.maxSize)
	return true
}

// CanReturn 풀에 반환 가능한지 확인
func (sp *SessionPool) CanReturn() bool {
	sp.mutex.Lock()
	defer sp.mutex.Unlock()

	return sp.currentSize < sp.maxSize
}

// Size 현재 풀 크기
func (sp *SessionPool) Size() int {
	sp.mutex.Lock()
	defer sp.mutex.Unlock()

	return sp.currentSize
}

// Cleanup 풀 정리 (오래된 세션 제거)
func (sp *SessionPool) Cleanup() int {
	sp.mutex.Lock()
	defer sp.mutex.Unlock()

	if len(sp.inactive) == 0 {
		return 0
	}

	// 1시간 이상 유휴 상태인 세션 제거
	maxIdleTime := 1 * time.Hour
	var active []*PTYSession
	removed := 0

	for _, session := range sp.inactive {
		if session.GetIdleTime() < maxIdleTime {
			active = append(active, session)
		} else {
			// 오래된 세션 종료
			go func(s *PTYSession) {
				if err := s.Terminate(); err != nil {
					log.Errorf("Failed to terminate old pooled session: %v", err)
				}
			}(session)
			removed++
		}
	}

	sp.inactive = active
	sp.currentSize = len(active)

	if removed > 0 {
		log.Infof("Cleaned up %d old sessions from pool", removed)
	}

	return removed
}

// Destroy 풀 파괴 (모든 세션 종료)
func (sp *SessionPool) Destroy() {
	sp.mutex.Lock()
	defer sp.mutex.Unlock()

	for _, session := range sp.inactive {
		go func(s *PTYSession) {
			if err := s.Terminate(); err != nil {
				log.Errorf("Failed to terminate pooled session during destroy: %v", err)
			}
		}(session)
	}

	sp.inactive = nil
	sp.currentSize = 0

	log.Info("Session pool destroyed")
}

// GetStats 풀 통계 조회
func (sp *SessionPool) GetStats() map[string]interface{} {
	sp.mutex.Lock()
	defer sp.mutex.Unlock()

	hitRate := float64(0)
	if sp.hits+sp.misses > 0 {
		hitRate = float64(sp.hits) / float64(sp.hits+sp.misses) * 100
	}

	return map[string]interface{}{
		"pool_size":    sp.currentSize,
		"max_size":     sp.maxSize,
		"hits":         sp.hits,
		"misses":       sp.misses,
		"returned":     sp.returned,
		"hit_rate":     hitRate,
		"utilization":  float64(sp.currentSize) / float64(sp.maxSize) * 100,
	}
}

// Resize 풀 크기 조정
func (sp *SessionPool) Resize(newSize int) {
	sp.mutex.Lock()
	defer sp.mutex.Unlock()

	oldSize := sp.maxSize
	sp.maxSize = newSize

	// 크기가 줄어든 경우 초과 세션 제거
	if sp.currentSize > newSize {
		toRemove := sp.currentSize - newSize
		for i := 0; i < toRemove && len(sp.inactive) > 0; i++ {
			session := sp.inactive[len(sp.inactive)-1]
			sp.inactive = sp.inactive[:len(sp.inactive)-1]
			sp.currentSize--

			go func(s *PTYSession) {
				if err := s.Terminate(); err != nil {
					log.Errorf("Failed to terminate excess pooled session: %v", err)
				}
			}(session)
		}
	}

	log.Infof("Pool resized from %d to %d", oldSize, newSize)
}

// Preload 풀 사전 로드
func (sp *SessionPool) Preload(count int) int {
	sp.mutex.Lock()
	defer sp.mutex.Unlock()

	loaded := 0
	for i := 0; i < count && sp.currentSize < sp.maxSize; i++ {
		session := NewPTYSession("", nil)
		session.Status = SessionIdle
		sp.inactive = append(sp.inactive, session)
		sp.currentSize++
		loaded++
	}

	if loaded > 0 {
		log.Infof("Preloaded %d sessions into pool", loaded)
	}

	return loaded
}