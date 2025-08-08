package pty

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
)

// SessionStatus 세션 상태
type SessionStatus int

const (
	SessionActive SessionStatus = iota
	SessionIdle
	SessionTerminated
)

// String SessionStatus 문자열 변환
func (s SessionStatus) String() string {
	switch s {
	case SessionActive:
		return "active"
	case SessionIdle:
		return "idle"
	case SessionTerminated:
		return "terminated"
	default:
		return fmt.Sprintf("unknown(%d)", s)
	}
}

// PTYConfig PTY 설정
type PTYConfig struct {
	Rows        int               // 터미널 행 수
	Cols        int               // 터미널 열 수
	Term        string            // 터미널 타입 (예: xterm-256color)
	Shell       string            // 사용할 셸 (예: /bin/bash)
	WorkingDir  string            // 작업 디렉토리
	Environment map[string]string // 환경 변수
}

// DefaultPTYConfig 기본 PTY 설정 생성
func DefaultPTYConfig() *PTYConfig {
	return &PTYConfig{
		Rows:        24,
		Cols:        80,
		Term:        "xterm-256color",
		Shell:       "/bin/bash",
		WorkingDir:  "/",
		Environment: make(map[string]string),
	}
}

// PTYSession PTY 세션
type PTYSession struct {
	ID          string           // 세션 고유 ID
	ContainerID string           // Docker 컨테이너 ID
	PTY         *os.File         // PTY 파일 디스크립터
	CreatedAt   time.Time        // 생성 시간
	LastActive  time.Time        // 마지막 활성 시간
	Status      SessionStatus    // 세션 상태
	Config      *PTYConfig       // PTY 설정
	cancel      context.CancelFunc // 컨텍스트 취소 함수
	mutex       sync.RWMutex     // 세션 보호용 뮤텍스
	
	// 메타데이터
	metadata    map[string]interface{} // 추가 메타데이터
	bytesRead   uint64                 // 읽은 바이트 수
	bytesWritten uint64                // 쓴 바이트 수
}

// NewPTYSession 새 PTY 세션 생성
func NewPTYSession(containerID string, config *PTYConfig) *PTYSession {
	if config == nil {
		config = DefaultPTYConfig()
	}

	return &PTYSession{
		ID:          generateSessionID(),
		ContainerID: containerID,
		CreatedAt:   time.Now(),
		LastActive:  time.Now(),
		Status:      SessionActive,
		Config:      config,
		metadata:    make(map[string]interface{}),
	}
}

// generateSessionID 세션 ID 생성
func generateSessionID() string {
	return fmt.Sprintf("pty-%s", uuid.New().String())
}

// UpdateActivity 활동 시간 업데이트
func (s *PTYSession) UpdateActivity() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	s.LastActive = time.Now()
	if s.Status == SessionIdle {
		s.Status = SessionActive
	}
}

// SetIdle 세션을 Idle 상태로 설정
func (s *PTYSession) SetIdle() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	if s.Status == SessionActive {
		s.Status = SessionIdle
	}
}

// Terminate 세션 종료
func (s *PTYSession) Terminate() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	if s.Status == SessionTerminated {
		return nil
	}
	
	s.Status = SessionTerminated
	
	// PTY 파일 디스크립터 닫기
	if s.PTY != nil {
		if err := s.PTY.Close(); err != nil {
			return fmt.Errorf("failed to close PTY: %w", err)
		}
		s.PTY = nil
	}
	
	// 컨텍스트 취소
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
	
	return nil
}

// IsActive 세션 활성 상태 확인
func (s *PTYSession) IsActive() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	return s.Status == SessionActive
}

// IsIdle 세션 유휴 상태 확인
func (s *PTYSession) IsIdle() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	return s.Status == SessionIdle
}

// IsTerminated 세션 종료 상태 확인
func (s *PTYSession) IsTerminated() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	return s.Status == SessionTerminated
}

// GetIdleTime 유휴 시간 계산
func (s *PTYSession) GetIdleTime() time.Duration {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	return time.Since(s.LastActive)
}

// SetMetadata 메타데이터 설정
func (s *PTYSession) SetMetadata(key string, value interface{}) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	s.metadata[key] = value
}

// GetMetadata 메타데이터 조회
func (s *PTYSession) GetMetadata(key string) (interface{}, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	value, exists := s.metadata[key]
	return value, exists
}

// SetCancel 취소 함수 설정
func (s *PTYSession) SetCancel(cancel context.CancelFunc) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	s.cancel = cancel
}

// SetPTY PTY 파일 디스크립터 설정
func (s *PTYSession) SetPTY(pty *os.File) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	s.PTY = pty
}

// GetPTY PTY 파일 디스크립터 조회
func (s *PTYSession) GetPTY() *os.File {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	return s.PTY
}

// ResizePTY PTY 크기 조정
func (s *PTYSession) ResizePTY(rows, cols int) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	s.Config.Rows = rows
	s.Config.Cols = cols
	
	// 실제 PTY 크기 조정은 플랫폼별 구현 필요
	// 여기서는 설정만 업데이트
	
	s.UpdateActivity()
	return nil
}

// Clone 세션 설정 복제 (새 세션 생성용)
func (s *PTYSession) Clone() *PTYConfig {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	config := &PTYConfig{
		Rows:        s.Config.Rows,
		Cols:        s.Config.Cols,
		Term:        s.Config.Term,
		Shell:       s.Config.Shell,
		WorkingDir:  s.Config.WorkingDir,
		Environment: make(map[string]string),
	}
	
	for k, v := range s.Config.Environment {
		config.Environment[k] = v
	}
	
	return config
}

// GetStats 세션 통계 조회
func (s *PTYSession) GetStats() map[string]interface{} {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	return map[string]interface{}{
		"id":            s.ID,
		"container_id":  s.ContainerID,
		"status":        s.Status.String(),
		"created_at":    s.CreatedAt,
		"last_active":   s.LastActive,
		"idle_time":     time.Since(s.LastActive).Seconds(),
		"bytes_read":    s.bytesRead,
		"bytes_written": s.bytesWritten,
	}
}