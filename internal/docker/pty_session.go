package docker

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/google/uuid"
)

// PTYSession PTY 세션을 나타내는 인터페이스
type PTYSession interface {
	ID() string
	ContainerID() string
	Start(ctx context.Context) error
	Stop() error
	Write(data []byte) (int, error)
	Read(data []byte) (int, error)
	Resize(width, height int) error
	IsAlive() bool
	GetCreatedAt() time.Time
	GetLastActivity() time.Time
}

// ptySession PTY 세션의 실제 구현
type ptySession struct {
	id           string
	containerID  string
	execID       string
	client       ClientInterface
	hijacked     types.HijackedResponse
	createdAt    time.Time
	lastActivity time.Time
	isAlive      bool
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	stdin        io.WriteCloser
	stdout       io.ReadCloser
	stderr       io.ReadCloser
	width        int
	height       int
}

// NewPTYSession 새로운 PTY 세션을 생성합니다
func NewPTYSession(containerID string, client ClientInterface) PTYSession {
	ctx, cancel := context.WithCancel(context.Background())
	return &ptySession{
		id:           uuid.New().String(),
		containerID:  containerID,
		client:       client,
		createdAt:    time.Now(),
		lastActivity: time.Now(),
		isAlive:      false,
		ctx:          ctx,
		cancel:       cancel,
		width:        80,  // 기본 터미널 너비
		height:       24,  // 기본 터미널 높이
	}
}

// ID 세션 ID를 반환합니다
func (s *ptySession) ID() string {
	return s.id
}

// ContainerID 컨테이너 ID를 반환합니다
func (s *ptySession) ContainerID() string {
	return s.containerID
}

// Start PTY 세션을 시작합니다
func (s *ptySession) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isAlive {
		return fmt.Errorf("PTY session %s is already started", s.id)
	}

	// Exec 인스턴스 생성
	execConfig := ExecConfig{
		Cmd:          []string{"/bin/bash"},
		AttachStdout: true,
		AttachStderr: true,
		AttachStdin:  true,
		Tty:          true,
		User:         "root",
		WorkingDir:   "/workspace",
	}

	createResp, err := s.client.ContainerExecCreate(ctx, s.containerID, execConfig)
	if err != nil {
		return fmt.Errorf("failed to create exec instance: %w", err)
	}

	s.execID = createResp.ID

	// Exec 시작 및 Hijacked 연결 설정
	startConfig := ExecStartConfig{
		Detach: false,
		Tty:    true,
	}

	hijacked, err := s.client.ContainerExecStart(ctx, s.execID, startConfig)
	if err != nil {
		return fmt.Errorf("failed to start exec instance: %w", err)
	}

	s.hijacked = hijacked
	s.stdin = hijacked.Conn
	s.stdout = hijacked.Reader
	s.stderr = hijacked.Reader // TTY 모드에서는 stderr이 stdout으로 병합됨
	s.isAlive = true
	s.lastActivity = time.Now()

	return nil
}

// Stop PTY 세션을 중지합니다
func (s *ptySession) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isAlive {
		return nil
	}

	s.isAlive = false
	s.cancel()

	// Hijacked 연결 정리
	if s.hijacked.Conn != nil {
		s.hijacked.Conn.Close()
	}

	// Exec 인스턴스 종료 (선택적)
	if s.execID != "" {
		// Docker는 일반적으로 연결이 끊어지면 exec 인스턴스를 자동으로 정리합니다
	}

	return nil
}

// Write PTY 세션에 데이터를 씁니다 (stdin)
func (s *ptySession) Write(data []byte) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.isAlive || s.stdin == nil {
		return 0, fmt.Errorf("PTY session %s is not active", s.id)
	}

	n, err := s.stdin.Write(data)
	if err == nil {
		s.lastActivity = time.Now()
	}
	return n, err
}

// Read PTY 세션에서 데이터를 읽습니다 (stdout/stderr)
func (s *ptySession) Read(data []byte) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.isAlive || s.stdout == nil {
		return 0, fmt.Errorf("PTY session %s is not active", s.id)
	}

	n, err := s.stdout.Read(data)
	if err == nil && n > 0 {
		s.lastActivity = time.Now()
	}
	return n, err
}

// Resize PTY 세션의 터미널 크기를 조정합니다
func (s *ptySession) Resize(width, height int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isAlive {
		return fmt.Errorf("PTY session %s is not active", s.id)
	}

	s.width = width
	s.height = height

	// Docker API를 통한 터미널 크기 조정
	// Note: 현재 docker/docker/client에서 직접적인 터미널 크기 조정 API는 제한적입니다
	// 향후 필요시 docker/docker/client 확장 또는 대안적 구현 필요

	return nil
}

// IsAlive PTY 세션이 활성 상태인지 확인합니다
func (s *ptySession) IsAlive() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isAlive
}

// GetCreatedAt 세션 생성 시간을 반환합니다
func (s *ptySession) GetCreatedAt() time.Time {
	return s.createdAt
}

// GetLastActivity 마지막 활동 시간을 반환합니다
func (s *ptySession) GetLastActivity() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastActivity
}

// PTYSessionManager PTY 세션들을 관리하는 구조체
type PTYSessionManager struct {
	sessions    map[string]PTYSession
	maxSessions int
	client      ClientInterface
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
	cleanupTick *time.Ticker
}

// NewPTYSessionManager 새로운 PTY 세션 관리자를 생성합니다
func NewPTYSessionManager(client ClientInterface, maxSessions int) *PTYSessionManager {
	ctx, cancel := context.WithCancel(context.Background())
	
	manager := &PTYSessionManager{
		sessions:    make(map[string]PTYSession),
		maxSessions: maxSessions,
		client:      client,
		ctx:         ctx,
		cancel:      cancel,
		cleanupTick: time.NewTicker(30 * time.Second), // 30초마다 정리 작업
	}

	// 백그라운드 정리 작업 시작
	go manager.cleanupRoutine()

	return manager
}

// CreateSession 새로운 PTY 세션을 생성합니다
func (m *PTYSessionManager) CreateSession(ctx context.Context, containerID string) (PTYSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 최대 세션 수 확인
	if len(m.sessions) >= m.maxSessions {
		return nil, fmt.Errorf("maximum number of sessions (%d) reached", m.maxSessions)
	}

	// 컨테이너 유효성 확인
	_, err := m.client.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("container %s not found or not accessible: %w", containerID, err)
	}

	// 새 세션 생성
	session := NewPTYSession(containerID, m.client)
	err = session.Start(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start PTY session: %w", err)
	}

	m.sessions[session.ID()] = session
	return session, nil
}

// GetSession 세션 ID로 PTY 세션을 조회합니다
func (m *PTYSessionManager) GetSession(sessionID string) (PTYSession, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("PTY session %s not found", sessionID)
	}

	if !session.IsAlive() {
		return nil, fmt.Errorf("PTY session %s is not active", sessionID)
	}

	return session, nil
}

// ListSessions 모든 활성 PTY 세션을 조회합니다
func (m *PTYSessionManager) ListSessions() []PTYSession {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var sessions []PTYSession
	for _, session := range m.sessions {
		if session.IsAlive() {
			sessions = append(sessions, session)
		}
	}

	return sessions
}

// RemoveSession PTY 세션을 제거합니다
func (m *PTYSessionManager) RemoveSession(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return fmt.Errorf("PTY session %s not found", sessionID)
	}

	err := session.Stop()
	if err != nil {
		return fmt.Errorf("failed to stop PTY session %s: %w", sessionID, err)
	}

	delete(m.sessions, sessionID)
	return nil
}

// GetSessionCount 현재 활성 세션 수를 반환합니다
func (m *PTYSessionManager) GetSessionCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, session := range m.sessions {
		if session.IsAlive() {
			count++
		}
	}

	return count
}

// GetSessionsByContainer 특정 컨테이너의 모든 PTY 세션을 조회합니다
func (m *PTYSessionManager) GetSessionsByContainer(containerID string) []PTYSession {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var sessions []PTYSession
	for _, session := range m.sessions {
		if session.ContainerID() == containerID && session.IsAlive() {
			sessions = append(sessions, session)
		}
	}

	return sessions
}

// Shutdown PTY 세션 관리자를 종료합니다
func (m *PTYSessionManager) Shutdown() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 정리 작업 중지
	m.cancel()
	m.cleanupTick.Stop()

	// 모든 세션 종료
	for sessionID, session := range m.sessions {
		if err := session.Stop(); err != nil {
			// 로그에 기록하지만 계속 진행
			fmt.Printf("Warning: failed to stop PTY session %s: %v\n", sessionID, err)
		}
	}

	// 세션 맵 정리
	m.sessions = make(map[string]PTYSession)

	return nil
}

// cleanupRoutine 주기적으로 비활성 세션을 정리합니다
func (m *PTYSessionManager) cleanupRoutine() {
	defer m.cleanupTick.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-m.cleanupTick.C:
			m.cleanupInactiveSessions()
		}
	}
}

// cleanupInactiveSessions 비활성 세션들을 정리합니다
func (m *PTYSessionManager) cleanupInactiveSessions() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	inactiveThreshold := 30 * time.Minute // 30분 비활성 후 정리

	var toRemove []string
	for sessionID, session := range m.sessions {
		if !session.IsAlive() || now.Sub(session.GetLastActivity()) > inactiveThreshold {
			toRemove = append(toRemove, sessionID)
		}
	}

	for _, sessionID := range toRemove {
		if session, exists := m.sessions[sessionID]; exists {
			session.Stop()
			delete(m.sessions, sessionID)
		}
	}

	if len(toRemove) > 0 {
		fmt.Printf("Cleaned up %d inactive PTY sessions\n", len(toRemove))
	}
}

// PTYSessionStats PTY 세션 관리자의 통계 정보
type PTYSessionStats struct {
	TotalSessions    int           `json:"total_sessions"`
	ActiveSessions   int           `json:"active_sessions"`
	MaxSessions      int           `json:"max_sessions"`
	AverageLifetime  time.Duration `json:"average_lifetime"`
	OldestSession    time.Time     `json:"oldest_session"`
	ContainerCounts  map[string]int `json:"container_counts"`
}

// GetStats PTY 세션 관리자의 통계를 반환합니다
func (m *PTYSessionManager) GetStats() *PTYSessionStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := &PTYSessionStats{
		TotalSessions:   len(m.sessions),
		MaxSessions:     m.maxSessions,
		ContainerCounts: make(map[string]int),
	}

	var lifetimes []time.Duration
	now := time.Now()
	oldestTime := now

	for _, session := range m.sessions {
		if session.IsAlive() {
			stats.ActiveSessions++
		}

		// 생존 시간 계산
		lifetime := now.Sub(session.GetCreatedAt())
		lifetimes = append(lifetimes, lifetime)

		// 가장 오래된 세션 추적
		if session.GetCreatedAt().Before(oldestTime) {
			oldestTime = session.GetCreatedAt()
		}

		// 컨테이너별 세션 수
		containerID := session.ContainerID()
		stats.ContainerCounts[containerID]++
	}

	// 평균 생존 시간 계산
	if len(lifetimes) > 0 {
		var totalLifetime time.Duration
		for _, lifetime := range lifetimes {
			totalLifetime += lifetime
		}
		stats.AverageLifetime = totalLifetime / time.Duration(len(lifetimes))
	}

	if oldestTime != now {
		stats.OldestSession = oldestTime
	}

	return stats
}