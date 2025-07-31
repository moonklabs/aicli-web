package docker

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// SnapshotManager 터미널 스냅샷 관리자
type SnapshotManager struct {
	ptyManager PTYSessionManagement
	capturers  map[string]*SnapshotCapturer
	mu         sync.RWMutex
	
	// 기본 설정
	defaultInterval   time.Duration
	defaultMaxHistory int
}

// NewSnapshotManager 새로운 스냅샷 관리자 생성
func NewSnapshotManager(ptyManager PTYSessionManagement) *SnapshotManager {
	return &SnapshotManager{
		ptyManager:        ptyManager,
		capturers:         make(map[string]*SnapshotCapturer),
		defaultInterval:   1 * time.Second,    // 1초 간격
		defaultMaxHistory: 3600,               // 1시간 분량 (3600개)
	}
}

// CreateCapturer 새로운 캡처러 생성
func (sm *SnapshotManager) CreateCapturer(sessionID string, interval time.Duration, maxHistory int) (*SnapshotCapturer, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// 이미 존재하는 캡처러 확인
	if _, exists := sm.capturers[sessionID]; exists {
		return nil, fmt.Errorf("capturer for session %s already exists", sessionID)
	}

	// PTY 세션 조회
	session, err := sm.ptyManager.GetSession(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get PTY session %s: %w", sessionID, err)
	}

	// 기본값 적용
	if interval == 0 {
		interval = sm.defaultInterval
	}
	if maxHistory == 0 {
		maxHistory = sm.defaultMaxHistory
	}

	// 캡처러 생성
	capturer := NewSnapshotCapturer(session, interval, maxHistory)
	sm.capturers[sessionID] = capturer

	return capturer, nil
}

// GetCapturer 캡처러 조회
func (sm *SnapshotManager) GetCapturer(sessionID string) (*SnapshotCapturer, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	capturer, exists := sm.capturers[sessionID]
	if !exists {
		return nil, fmt.Errorf("capturer for session %s not found", sessionID)
	}

	return capturer, nil
}

// RemoveCapturer 캡처러 제거
func (sm *SnapshotManager) RemoveCapturer(sessionID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	capturer, exists := sm.capturers[sessionID]
	if !exists {
		return fmt.Errorf("capturer for session %s not found", sessionID)
	}

	// 캡처 중지
	capturer.Stop()

	// 맵에서 제거
	delete(sm.capturers, sessionID)

	return nil
}

// StartCapture 캡처 시작
func (sm *SnapshotManager) StartCapture(sessionID string) error {
	sm.mu.RLock()
	capturer, exists := sm.capturers[sessionID]
	sm.mu.RUnlock()

	if !exists {
		// 캡처러가 없으면 자동 생성
		var err error
		capturer, err = sm.CreateCapturer(sessionID, sm.defaultInterval, sm.defaultMaxHistory)
		if err != nil {
			return fmt.Errorf("failed to create capturer: %w", err)
		}
	}

	return capturer.Start()
}

// StopCapture 캡처 중지
func (sm *SnapshotManager) StopCapture(sessionID string) error {
	sm.mu.RLock()
	capturer, exists := sm.capturers[sessionID]
	sm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("capturer for session %s not found", sessionID)
	}

	return capturer.Stop()
}

// GetSnapshot 최신 스냅샷 조회
func (sm *SnapshotManager) GetSnapshot(sessionID string) (TerminalSnapshot, error) {
	sm.mu.RLock()
	capturer, exists := sm.capturers[sessionID]
	sm.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("capturer for session %s not found", sessionID)
	}

	snapshot := capturer.GetLatestSnapshot()
	if snapshot == nil {
		return nil, fmt.Errorf("no snapshots available for session %s", sessionID)
	}

	return snapshot, nil
}

// GetSnapshotHistory 스냅샷 히스토리 조회
func (sm *SnapshotManager) GetSnapshotHistory(sessionID string) ([]TerminalSnapshot, error) {
	sm.mu.RLock()
	capturer, exists := sm.capturers[sessionID]
	sm.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("capturer for session %s not found", sessionID)
	}

	return capturer.GetSnapshotHistory(), nil
}

// GetAllCapturers 모든 캡처러 조회
func (sm *SnapshotManager) GetAllCapturers() map[string]*SnapshotCapturer {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// 복사본 생성
	result := make(map[string]*SnapshotCapturer)
	for sessionID, capturer := range sm.capturers {
		result[sessionID] = capturer
	}

	return result
}

// Shutdown 관리자 종료
func (sm *SnapshotManager) Shutdown() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// 모든 캡처러 중지 및 제거
	for sessionID, capturer := range sm.capturers {
		capturer.Stop()
		delete(sm.capturers, sessionID)
	}

	return nil
}

// GetStats 전체 통계 정보 조회
func (sm *SnapshotManager) GetStats() *SnapshotManagerStats {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	stats := &SnapshotManagerStats{
		TotalCapturers:  len(sm.capturers),
		DefaultInterval: sm.defaultInterval,
		DefaultMaxHistory: sm.defaultMaxHistory,
		CapturerStats:   make(map[string]*SnapshotCapturerStats),
	}

	var totalSnapshots int
	var totalMemoryUsage int64
	var runningCapturers int

	for sessionID, capturer := range sm.capturers {
		capturerStats := capturer.GetStats()
		stats.CapturerStats[sessionID] = capturerStats

		totalSnapshots += capturerStats.SnapshotCount
		totalMemoryUsage += capturerStats.MemoryUsage
		
		if capturerStats.IsRunning {
			runningCapturers++
		}
	}

	stats.TotalSnapshots = totalSnapshots
	stats.TotalMemoryUsage = totalMemoryUsage
	stats.RunningCapturers = runningCapturers

	return stats
}

// CleanupInactiveCapturers 비활성 캡처러 정리
func (sm *SnapshotManager) CleanupInactiveCapturers() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	var toRemove []string
	
	for sessionID, capturer := range sm.capturers {
		// PTY 세션이 살아있는지 확인
		if !capturer.ptySession.IsAlive() {
			toRemove = append(toRemove, sessionID)
		}
	}

	// 비활성 캡처러 제거
	for _, sessionID := range toRemove {
		if capturer := sm.capturers[sessionID]; capturer != nil {
			capturer.Stop()
			delete(sm.capturers, sessionID)
		}
	}

	if len(toRemove) > 0 {
		fmt.Printf("Cleaned up %d inactive snapshot capturers\n", len(toRemove))
	}
}

// AutoStartForNewSessions 새로운 세션에 대해 자동으로 캡처 시작
func (sm *SnapshotManager) AutoStartForNewSessions() {
	sessions := sm.ptyManager.ListSessions()
	
	for _, session := range sessions {
		sessionID := session.ID()
		
		sm.mu.RLock()
		_, exists := sm.capturers[sessionID]
		sm.mu.RUnlock()
		
		// 캡처러가 없고 세션이 활성 상태이면 자동 생성 및 시작
		if !exists && session.IsAlive() {
			capturer, err := sm.CreateCapturer(sessionID, sm.defaultInterval, sm.defaultMaxHistory)
			if err != nil {
				fmt.Printf("Failed to create capturer for session %s: %v\n", sessionID, err)
				continue
			}
			
			if err := capturer.Start(); err != nil {
				fmt.Printf("Failed to start capturer for session %s: %v\n", sessionID, err)
			} else {
				fmt.Printf("Auto-started snapshot capturer for session %s\n", sessionID)
			}
		}
	}
}

// SnapshotManagerStats 스냅샷 관리자 통계
type SnapshotManagerStats struct {
	TotalCapturers    int                               `json:"total_capturers"`
	RunningCapturers  int                               `json:"running_capturers"`
	TotalSnapshots    int                               `json:"total_snapshots"`
	TotalMemoryUsage  int64                             `json:"total_memory_usage"`
	DefaultInterval   time.Duration                     `json:"default_interval"`
	DefaultMaxHistory int                               `json:"default_max_history"`
	CapturerStats     map[string]*SnapshotCapturerStats `json:"capturer_stats"`
}

// StartPeriodicCleanup 주기적 정리 작업 시작
func (sm *SnapshotManager) StartPeriodicCleanup(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				sm.CleanupInactiveCapturers()
				sm.AutoStartForNewSessions()
			}
		}
	}()
}

// GetSnapshotsByTimeRange 시간 범위로 스냅샷 조회
func (sm *SnapshotManager) GetSnapshotsByTimeRange(sessionID string, startTime, endTime time.Time) ([]TerminalSnapshot, error) {
	sm.mu.RLock()
	capturer, exists := sm.capturers[sessionID]
	sm.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("capturer for session %s not found", sessionID)
	}

	allSnapshots := capturer.GetSnapshotHistory()
	var filteredSnapshots []TerminalSnapshot

	for _, snapshot := range allSnapshots {
		timestamp := snapshot.GetTimestamp()
		if (timestamp.Equal(startTime) || timestamp.After(startTime)) &&
		   (timestamp.Equal(endTime) || timestamp.Before(endTime)) {
			filteredSnapshots = append(filteredSnapshots, snapshot)
		}
	}

	return filteredSnapshots, nil
}

// ExportSnapshot 스냅샷을 JSON으로 내보내기
func (sm *SnapshotManager) ExportSnapshot(sessionID string, timestamp time.Time) ([]byte, error) {
	snapshots, err := sm.GetSnapshotHistory(sessionID)
	if err != nil {
		return nil, err
	}

	// 해당 타임스탬프의 스냅샷 찾기
	for _, snapshot := range snapshots {
		if snapshot.GetTimestamp().Equal(timestamp) {
			return snapshot.GetCompressedData()
		}
	}

	return nil, fmt.Errorf("snapshot not found for timestamp %v", timestamp)
}