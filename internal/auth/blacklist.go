package auth

import (
	"sync"
	"time"
)

// BlacklistEntry 블랙리스트 항목
type BlacklistEntry struct {
	Token     string
	ExpiresAt time.Time
}

// Blacklist 토큰 블랙리스트 관리자
type Blacklist struct {
	mu      sync.RWMutex
	entries map[string]BlacklistEntry
}

// NewBlacklist 새로운 블랙리스트 생성
func NewBlacklist() *Blacklist {
	bl := &Blacklist{
		entries: make(map[string]BlacklistEntry),
	}
	
	// 만료된 항목 정리를 위한 고루틴 시작
	go bl.cleanupExpiredEntries()
	
	return bl
}

// Add 토큰을 블랙리스트에 추가
func (bl *Blacklist) Add(token string, expiresAt time.Time) {
	bl.mu.Lock()
	defer bl.mu.Unlock()
	
	bl.entries[token] = BlacklistEntry{
		Token:     token,
		ExpiresAt: expiresAt,
	}
}

// IsBlacklisted 토큰이 블랙리스트에 있는지 확인
func (bl *Blacklist) IsBlacklisted(token string) bool {
	bl.mu.RLock()
	defer bl.mu.RUnlock()
	
	entry, exists := bl.entries[token]
	if !exists {
		return false
	}
	
	// 만료된 경우 블랙리스트에서 제외
	if time.Now().After(entry.ExpiresAt) {
		return false
	}
	
	return true
}

// Remove 토큰을 블랙리스트에서 제거
func (bl *Blacklist) Remove(token string) {
	bl.mu.Lock()
	defer bl.mu.Unlock()
	
	delete(bl.entries, token)
}

// cleanupExpiredEntries 만료된 항목 정리 (백그라운드 작업)
func (bl *Blacklist) cleanupExpiredEntries() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		bl.mu.Lock()
		now := time.Now()
		
		for token, entry := range bl.entries {
			if now.After(entry.ExpiresAt) {
				delete(bl.entries, token)
			}
		}
		
		bl.mu.Unlock()
	}
}

// Size 블랙리스트 크기 반환 (테스트용)
func (bl *Blacklist) Size() int {
	bl.mu.RLock()
	defer bl.mu.RUnlock()
	
	return len(bl.entries)
}