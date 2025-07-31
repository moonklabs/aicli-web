package docker

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/google/uuid"
)

// enhancedPTYSession 향상된 PTY 세션 구현
type enhancedPTYSession struct {
	id           string
	containerID  string
	execID       string
	client       ClientInterface
	config       PTYConfig
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
	
	// 재연결 관련
	reconnectAttempts int
	maxReconnectAttempts int
	reconnectBackoff time.Duration
	
	// 환경 변수 추적
	currentEnv map[string]string
	currentWorkingDir string
}

// NewEnhancedPTYSession 새로운 향상된 PTY 세션 생성
func NewEnhancedPTYSession(containerID string, client ClientInterface, config PTYConfig) PTYSession {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &enhancedPTYSession{
		id:                   uuid.New().String(),
		containerID:          containerID,
		client:               client,
		config:               config,
		createdAt:            time.Now(),
		lastActivity:         time.Now(),
		isAlive:              false,
		ctx:                  ctx,
		cancel:               cancel,
		maxReconnectAttempts: 3,
		reconnectBackoff:     2 * time.Second,
		currentEnv:           make(map[string]string),
		currentWorkingDir:    config.WorkingDir,
	}
}

// ID 세션 ID 반환
func (s *enhancedPTYSession) ID() string {
	return s.id
}

// ContainerID 컨테이너 ID 반환
func (s *enhancedPTYSession) ContainerID() string {
	return s.containerID
}

// Start PTY 세션 시작
func (s *enhancedPTYSession) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isAlive {
		return fmt.Errorf("PTY session %s is already started", s.id)
	}

	// 환경 변수 준비
	var envVars []string
	for key, value := range s.config.Environment {
		envVars = append(envVars, fmt.Sprintf("%s=%s", key, value))
		s.currentEnv[key] = value
	}

	// Exec 설정 생성
	execConfig := ExecConfig{
		Cmd:          []string{s.config.Shell},
		AttachStdout: true,
		AttachStderr: true,
		AttachStdin:  true,
		Tty:          s.config.Tty,
		User:         s.config.User,
		WorkingDir:   s.config.WorkingDir,
	}

	// Exec 인스턴스 생성
	createResp, err := s.client.ContainerExecCreate(ctx, s.containerID, execConfig)
	if err != nil {
		return fmt.Errorf("failed to create exec instance: %w", err)
	}

	s.execID = createResp.ID

	// Exec 시작 및 Hijacked 연결 설정
	startConfig := ExecStartConfig{
		Detach: false,
		Tty:    s.config.Tty,
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
	s.reconnectAttempts = 0

	// 초기 환경 설정
	if err := s.setupInitialEnvironment(); err != nil {
		// 경고만 출력하고 계속 진행
		fmt.Printf("Warning: failed to setup initial environment: %v\n", err)
	}

	return nil
}

// setupInitialEnvironment 초기 환경 설정
func (s *enhancedPTYSession) setupInitialEnvironment() error {
	// 작업 디렉토리 설정
	if s.config.WorkingDir != "" {
		cdCommand := fmt.Sprintf("cd %s\n", s.config.WorkingDir)
		if _, err := s.stdin.Write([]byte(cdCommand)); err != nil {
			return fmt.Errorf("failed to set working directory: %w", err)
		}
	}

	// 환경 변수 설정
	for key, value := range s.config.Environment {
		exportCommand := fmt.Sprintf("export %s='%s'\n", key, value)
		if _, err := s.stdin.Write([]byte(exportCommand)); err != nil {
			return fmt.Errorf("failed to set environment variable %s: %w", key, err)
		}
	}

	// 터미널 크기 설정 (가능한 경우)
	if s.config.Size.Width > 0 && s.config.Size.Height > 0 {
		resizeCommand := fmt.Sprintf("stty cols %d rows %d\n", s.config.Size.Width, s.config.Size.Height)
		if _, err := s.stdin.Write([]byte(resizeCommand)); err != nil {
			// 크기 설정 실패는 중요하지 않으므로 로그만 남김
			fmt.Printf("Warning: failed to set terminal size: %v\n", err)
		}
	}

	return nil
}

// Stop PTY 세션 중지
func (s *enhancedPTYSession) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isAlive {
		return nil
	}

	s.isAlive = false
	s.cancel()

	// graceful shutdown 시도
	if s.stdin != nil {
		// exit 명령 전송
		s.stdin.Write([]byte("exit\n"))
		
		// 잠시 대기 후 강제 종료
		time.Sleep(1 * time.Second)
	}

	// Hijacked 연결 정리
	if s.hijacked.Conn != nil {
		s.hijacked.Conn.Close()
	}

	return nil
}

// Write PTY 세션에 데이터 쓰기
func (s *enhancedPTYSession) Write(data []byte) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.isAlive || s.stdin == nil {
		return 0, fmt.Errorf("PTY session %s is not active", s.id)
	}

	n, err := s.stdin.Write(data)
	if err != nil {
		// 연결 실패 시 재연결 시도
		if s.shouldReconnect(err) {
			go s.attemptReconnect()
		}
		return n, err
	}

	if n > 0 {
		s.lastActivity = time.Now()
	}

	return n, err
}

// Read PTY 세션에서 데이터 읽기
func (s *enhancedPTYSession) Read(data []byte) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.isAlive || s.stdout == nil {
		return 0, fmt.Errorf("PTY session %s is not active", s.id)
	}

	n, err := s.stdout.Read(data)
	if err != nil {
		// 연결 실패 시 재연결 시도
		if s.shouldReconnect(err) {
			go s.attemptReconnect()
		}
		return n, err
	}

	if n > 0 {
		s.lastActivity = time.Now()
	}

	return n, err
}

// Resize PTY 세션의 터미널 크기 조정
func (s *enhancedPTYSession) Resize(width, height int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isAlive {
		return fmt.Errorf("PTY session %s is not active", s.id)
	}

	// 설정 업데이트
	s.config.Size.Width = width
	s.config.Size.Height = height

	// stty 명령을 통한 터미널 크기 조정
	resizeCommand := fmt.Sprintf("stty cols %d rows %d\n", width, height)
	if _, err := s.stdin.Write([]byte(resizeCommand)); err != nil {
		return fmt.Errorf("failed to resize terminal: %w", err)
	}

	return nil
}

// IsAlive PTY 세션 활성 상태 확인
func (s *enhancedPTYSession) IsAlive() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isAlive
}

// GetCreatedAt 세션 생성 시간 반환
func (s *enhancedPTYSession) GetCreatedAt() time.Time {
	return s.createdAt
}

// GetLastActivity 마지막 활동 시간 반환
func (s *enhancedPTYSession) GetLastActivity() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastActivity
}

// shouldReconnect 재연결이 필요한 에러인지 확인
func (s *enhancedPTYSession) shouldReconnect(err error) bool {
	if err == nil {
		return false
	}

	// EOF나 연결 종료 관련 에러인 경우 재연결 시도
	errStr := err.Error()
	reconnectErrors := []string{
		"EOF",
		"connection reset",
		"broken pipe",
		"connection refused",
		"use of closed network connection",
	}

	for _, reconnectErr := range reconnectErrors {
		if strings.Contains(errStr, reconnectErr) {
			return true
		}
	}

	return false
}

// attemptReconnect 재연결 시도
func (s *enhancedPTYSession) attemptReconnect() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.reconnectAttempts >= s.maxReconnectAttempts {
		fmt.Printf("Max reconnect attempts reached for PTY session %s\n", s.id)
		s.isAlive = false
		return
	}

	s.reconnectAttempts++
	fmt.Printf("Attempting to reconnect PTY session %s (attempt %d/%d)\n", 
		s.id, s.reconnectAttempts, s.maxReconnectAttempts)

	// 기존 연결 정리
	if s.hijacked.Conn != nil {
		s.hijacked.Conn.Close()
	}

	// 재연결 백오프
	time.Sleep(s.reconnectBackoff * time.Duration(s.reconnectAttempts))

	// 새로운 연결 시도
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.Start(ctx); err != nil {
		fmt.Printf("Failed to reconnect PTY session %s: %v\n", s.id, err)
		
		if s.reconnectAttempts >= s.maxReconnectAttempts {
			s.isAlive = false
		}
		return
	}

	fmt.Printf("Successfully reconnected PTY session %s\n", s.id)
	s.reconnectAttempts = 0 // 성공 시 재연결 카운터 리셋
}

// UpdateEnvironment 환경 변수 동적 업데이트
func (s *enhancedPTYSession) UpdateEnvironment(env map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isAlive || s.stdin == nil {
		return fmt.Errorf("PTY session %s is not active", s.id)
	}

	for key, value := range env {
		exportCommand := fmt.Sprintf("export %s='%s'\n", key, value)
		if _, err := s.stdin.Write([]byte(exportCommand)); err != nil {
			return fmt.Errorf("failed to set environment variable %s: %w", key, err)
		}
		s.currentEnv[key] = value
	}

	s.lastActivity = time.Now()
	return nil
}

// ChangeWorkingDirectory 작업 디렉토리 변경
func (s *enhancedPTYSession) ChangeWorkingDirectory(dir string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isAlive || s.stdin == nil {
		return fmt.Errorf("PTY session %s is not active", s.id)
	}

	cdCommand := fmt.Sprintf("cd %s\n", dir)
	if _, err := s.stdin.Write([]byte(cdCommand)); err != nil {
		return fmt.Errorf("failed to change working directory: %w", err)
	}

	s.currentWorkingDir = dir
	s.lastActivity = time.Now()
	return nil
}

// GetCurrentEnvironment 현재 환경 변수 조회
func (s *enhancedPTYSession) GetCurrentEnvironment() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 복사본 반환
	envCopy := make(map[string]string)
	for k, v := range s.currentEnv {
		envCopy[k] = v
	}
	return envCopy
}

// GetCurrentWorkingDirectory 현재 작업 디렉토리 조회
func (s *enhancedPTYSession) GetCurrentWorkingDirectory() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.currentWorkingDir
}

// GetConfig PTY 설정 조회
func (s *enhancedPTYSession) GetConfig() PTYConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

// UpdateConfig PTY 설정 업데이트
func (s *enhancedPTYSession) UpdateConfig(config PTYConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 일부 설정만 동적으로 업데이트 가능
	s.config.Size = config.Size
	
	// 환경 변수 업데이트
	if len(config.Environment) > 0 {
		for k, v := range config.Environment {
			s.config.Environment[k] = v
		}
	}

	// 작업 디렉토리 업데이트
	if config.WorkingDir != "" && config.WorkingDir != s.config.WorkingDir {
		s.config.WorkingDir = config.WorkingDir
	}

	return nil
}

// GetReconnectStats 재연결 통계 조회
func (s *enhancedPTYSession) GetReconnectStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"reconnect_attempts":     s.reconnectAttempts,
		"max_reconnect_attempts": s.maxReconnectAttempts,
		"reconnect_backoff":      s.reconnectBackoff,
	}
}