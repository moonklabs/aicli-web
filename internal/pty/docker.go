package pty

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/creack/pty"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"
)

// DockerPTYManager Docker PTY 관리자
type DockerPTYManager struct {
	client     *client.Client
	sessions   map[string]*DockerPTYSession
	mutex      sync.RWMutex
	
	// 설정
	config     *DockerPTYConfig
	
	// 메트릭
	totalAttached uint64
	totalDetached uint64
	activeCount   int
}

// DockerPTYSession Docker PTY 세션
type DockerPTYSession struct {
	SessionID    string
	ContainerID  string
	PTY          *os.File
	TTY          *os.File
	ExecID       string
	Context      context.Context
	Cancel       context.CancelFunc
	Status       SessionStatus
	CreatedAt    time.Time
	LastActive   time.Time
	
	// 스트림
	stdin        io.WriteCloser
	stdout       io.ReadCloser
	stderr       io.ReadCloser
	
	// 동기화
	mutex        sync.RWMutex
}

// DockerPTYConfig Docker PTY 설정
type DockerPTYConfig struct {
	APIVersion        string
	Endpoint          string
	TLSConfig         *TLSConfig
	Timeout           time.Duration
	MaxConnections    int
	EnableHealthCheck bool
}

// TLSConfig TLS 설정
type TLSConfig struct {
	CACert     string
	ClientCert string
	ClientKey  string
	Insecure   bool
}

// DefaultDockerPTYConfig 기본 Docker PTY 설정
func DefaultDockerPTYConfig() *DockerPTYConfig {
	return &DockerPTYConfig{
		APIVersion:        "1.41",
		Endpoint:          "unix:///var/run/docker.sock",
		Timeout:           30 * time.Second,
		MaxConnections:    100,
		EnableHealthCheck: true,
	}
}

// NewDockerPTYManager 새 Docker PTY 관리자 생성
func NewDockerPTYManager(config *DockerPTYConfig) (*DockerPTYManager, error) {
	if config == nil {
		config = DefaultDockerPTYConfig()
	}

	// Docker 클라이언트 생성
	opts := []client.Opt{
		client.WithAPIVersionNegotiation(),
		client.WithTimeout(config.Timeout),
	}

	if config.Endpoint != "" {
		opts = append(opts, client.WithHost(config.Endpoint))
	}

	if config.APIVersion != "" {
		opts = append(opts, client.WithVersion(config.APIVersion))
	}

	dockerClient, err := client.NewClientWithOpts(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}

	// 연결 테스트
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := dockerClient.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping docker daemon: %w", err)
	}

	manager := &DockerPTYManager{
		client:   dockerClient,
		sessions: make(map[string]*DockerPTYSession),
		config:   config,
	}

	log.Info("Docker PTY manager initialized")
	return manager, nil
}

// AttachPTY Docker 컨테이너에 PTY 연결
func (dm *DockerPTYManager) AttachPTY(ctx context.Context, containerID string, config *PTYConfig) (*os.File, error) {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()

	// 컨테이너 상태 확인
	containerInfo, err := dm.client.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}

	if !containerInfo.State.Running {
		return nil, fmt.Errorf("container %s is not running", containerID)
	}

	// Exec 인스턴스 생성
	execConfig := types.ExecConfig{
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          true,
		Env:          envMapToSlice(config.Environment),
		WorkingDir:   config.WorkingDir,
		Cmd:          []string{config.Shell},
	}

	execResp, err := dm.client.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create exec: %w", err)
	}

	// PTY 생성
	ptm, pts, err := pty.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open pty: %w", err)
	}

	// 터미널 크기 설정
	if err := pty.Setsize(ptm, &pty.Winsize{
		Rows: uint16(config.Rows),
		Cols: uint16(config.Cols),
	}); err != nil {
		ptm.Close()
		pts.Close()
		return nil, fmt.Errorf("failed to set pty size: %w", err)
	}

	// Exec 시작
	attachConfig := types.ExecStartCheck{
		Tty: true,
	}

	hijackedResp, err := dm.client.ContainerExecAttach(ctx, execResp.ID, attachConfig)
	if err != nil {
		ptm.Close()
		pts.Close()
		return nil, fmt.Errorf("failed to attach exec: %w", err)
	}

	// 세션 생성
	sessionCtx, cancel := context.WithCancel(ctx)
	session := &DockerPTYSession{
		SessionID:   generateSessionID(),
		ContainerID: containerID,
		PTY:         ptm,
		TTY:         pts,
		ExecID:      execResp.ID,
		Context:     sessionCtx,
		Cancel:      cancel,
		Status:      SessionActive,
		CreatedAt:   time.Now(),
		LastActive:  time.Now(),
	}

	// I/O 스트림 연결
	go dm.handleIO(session, hijackedResp)

	// 세션 저장
	dm.sessions[session.SessionID] = session
	dm.activeCount++
	dm.totalAttached++

	log.Infof("Attached PTY to container %s (session: %s)", containerID, session.SessionID)
	return ptm, nil
}

// handleIO I/O 처리
func (dm *DockerPTYManager) handleIO(session *DockerPTYSession, hijacked types.HijackedResponse) {
	defer func() {
		hijacked.Close()
		session.Cancel()
		
		dm.mutex.Lock()
		delete(dm.sessions, session.SessionID)
		dm.activeCount--
		dm.totalDetached++
		dm.mutex.Unlock()
		
		log.Infof("PTY session %s closed", session.SessionID)
	}()

	// 양방향 복사
	errCh := make(chan error, 2)

	// PTY -> Docker
	go func() {
		_, err := io.Copy(hijacked.Conn, session.PTY)
		errCh <- err
	}()

	// Docker -> PTY
	go func() {
		_, err := io.Copy(session.PTY, hijacked.Reader)
		errCh <- err
	}()

	// 에러 대기
	select {
	case err := <-errCh:
		if err != nil && err != io.EOF {
			log.Errorf("I/O error in session %s: %v", session.SessionID, err)
		}
	case <-session.Context.Done():
		log.Debugf("Session %s context cancelled", session.SessionID)
	}
}

// DetachPTY PTY 연결 해제
func (dm *DockerPTYManager) DetachPTY(sessionID string) error {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()

	session, exists := dm.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	// 컨텍스트 취소
	session.Cancel()

	// PTY 닫기
	if session.PTY != nil {
		session.PTY.Close()
	}
	if session.TTY != nil {
		session.TTY.Close()
	}

	// 세션 제거
	delete(dm.sessions, sessionID)
	dm.activeCount--
	dm.totalDetached++

	log.Infof("Detached PTY session %s", sessionID)
	return nil
}

// ResizePTY PTY 크기 조정
func (dm *DockerPTYManager) ResizePTY(sessionID string, rows, cols int) error {
	dm.mutex.RLock()
	session, exists := dm.sessions[sessionID]
	dm.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	// Docker exec 크기 조정
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := dm.client.ContainerExecResize(ctx, session.ExecID, types.ResizeOptions{
		Height: uint(rows),
		Width:  uint(cols),
	})
	if err != nil {
		return fmt.Errorf("failed to resize exec: %w", err)
	}

	// PTY 크기 조정
	if session.PTY != nil {
		if err := pty.Setsize(session.PTY, &pty.Winsize{
			Rows: uint16(rows),
			Cols: uint16(cols),
		}); err != nil {
			return fmt.Errorf("failed to resize pty: %w", err)
		}
	}

	log.Debugf("Resized PTY session %s to %dx%d", sessionID, cols, rows)
	return nil
}

// IsContainerRunning 컨테이너 실행 상태 확인
func (dm *DockerPTYManager) IsContainerRunning(containerID string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	containerInfo, err := dm.client.ContainerInspect(ctx, containerID)
	if err != nil {
		if client.IsErrNotFound(err) {
			return false, nil
		}
		return false, err
	}

	return containerInfo.State.Running, nil
}

// GetContainerInfo 컨테이너 정보 조회
func (dm *DockerPTYManager) GetContainerInfo(containerID string) (map[string]interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	containerInfo, err := dm.client.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}

	return map[string]interface{}{
		"id":         containerInfo.ID,
		"name":       containerInfo.Name,
		"image":      containerInfo.Config.Image,
		"state":      containerInfo.State.Status,
		"running":    containerInfo.State.Running,
		"created_at": containerInfo.Created,
		"labels":     containerInfo.Config.Labels,
		"env":        containerInfo.Config.Env,
	}, nil
}

// CreateContainer PTY 지원 컨테이너 생성
func (dm *DockerPTYManager) CreateContainer(ctx context.Context, config *ContainerConfig) (string, error) {
	// 컨테이너 설정
	containerConfig := &container.Config{
		Image:        config.Image,
		Env:          config.Environment,
		WorkingDir:   config.WorkingDir,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          true,
		OpenStdin:    true,
		StdinOnce:    false,
		Labels:       config.Labels,
	}

	if len(config.Command) > 0 {
		containerConfig.Cmd = config.Command
	}

	// 호스트 설정
	hostConfig := &container.HostConfig{
		AutoRemove: config.AutoRemove,
		Binds:      config.Volumes,
	}

	// 컨테이너 생성
	resp, err := dm.client.ContainerCreate(
		ctx,
		containerConfig,
		hostConfig,
		nil,
		nil,
		config.Name,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	// 컨테이너 시작
	if err := dm.client.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		// 실패 시 컨테이너 제거
		dm.client.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{Force: true})
		return "", fmt.Errorf("failed to start container: %w", err)
	}

	log.Infof("Created container %s (image: %s)", resp.ID[:12], config.Image)
	return resp.ID, nil
}

// ContainerConfig 컨테이너 설정
type ContainerConfig struct {
	Name        string
	Image       string
	Command     []string
	Environment []string
	WorkingDir  string
	Volumes     []string
	Labels      map[string]string
	AutoRemove  bool
}

// GetStats 통계 조회
func (dm *DockerPTYManager) GetStats() map[string]interface{} {
	dm.mutex.RLock()
	defer dm.mutex.RUnlock()

	return map[string]interface{}{
		"active_sessions": dm.activeCount,
		"total_attached":  dm.totalAttached,
		"total_detached":  dm.totalDetached,
		"docker_endpoint": dm.config.Endpoint,
		"api_version":     dm.config.APIVersion,
	}
}

// Close Docker PTY 관리자 종료
func (dm *DockerPTYManager) Close() error {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()

	// 모든 세션 종료
	for sessionID, session := range dm.sessions {
		session.Cancel()
		if session.PTY != nil {
			session.PTY.Close()
		}
		if session.TTY != nil {
			session.TTY.Close()
		}
		log.Debugf("Closed session %s during shutdown", sessionID)
	}

	dm.sessions = make(map[string]*DockerPTYSession)
	dm.activeCount = 0

	// Docker 클라이언트 종료
	if err := dm.client.Close(); err != nil {
		return fmt.Errorf("failed to close docker client: %w", err)
	}

	log.Info("Docker PTY manager closed")
	return nil
}

// envMapToSlice 환경변수 맵을 슬라이스로 변환
func envMapToSlice(envMap map[string]string) []string {
	var envSlice []string
	for key, value := range envMap {
		envSlice = append(envSlice, fmt.Sprintf("%s=%s", key, value))
	}
	return envSlice
}

// generateSessionID 세션 ID 생성
func generateSessionID() string {
	return fmt.Sprintf("pty-%d-%d", time.Now().UnixNano(), os.Getpid())
}