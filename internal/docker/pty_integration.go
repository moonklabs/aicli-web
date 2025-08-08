package docker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("component", "docker-pty")

// DockerPTYStatus represents PTY connection status
type DockerPTYStatus int

const (
	PTYConnecting DockerPTYStatus = iota
	PTYActive
	PTYReconnecting
	PTYTerminated
	PTYError
)

// DockerPTYIntegration manages Docker PTY sessions
type DockerPTYIntegration struct {
	client   *client.Client
	sessions map[string]*DockerPTYSession
	config   *DockerPTYConfig
	monitor  *ContainerMonitor
	mutex    sync.RWMutex
	stopCh   chan struct{}
}

// DockerPTYSession represents a PTY session with a Docker container
type DockerPTYSession struct {
	SessionID       string
	ContainerID     string
	Container       *DockerContainer
	PTYConn         types.HijackedResponse
	ExecID          string
	Config          *PTYSessionConfig
	Status          DockerPTYStatus
	CreatedAt       time.Time
	LastActivity    time.Time
	ResourceStats   *ContainerResourceStats
	cancel          context.CancelFunc
	mutex           sync.RWMutex
}

// DockerContainer contains container information
type DockerContainer struct {
	ID          string
	Name        string
	Image       string
	Status      string
	Networks    map[string]*network.EndpointSettings
	Mounts      []types.MountPoint
	Environment []string
	WorkingDir  string
	User        string
}

// DockerPTYConfig contains PTY integration configuration
type DockerPTYConfig struct {
	MaxSessions      int
	MonitorInterval  time.Duration
	ReconnectTimeout time.Duration
	HealthCheckInterval time.Duration
}

// PTYSessionConfig contains PTY session configuration
type PTYSessionConfig struct {
	Shell        string
	Term         string
	Rows         int
	Cols         int
	WorkingDir   string
	Environment  map[string]string
	User         string
	Privileged   bool
	AttachStdin  bool
	AttachStdout bool
	AttachStderr bool
	Detach       bool
}

// NewDockerPTYIntegration creates a new Docker PTY integration manager
func NewDockerPTYIntegration(dockerClient *client.Client, config *DockerPTYConfig) *DockerPTYIntegration {
	if config == nil {
		config = &DockerPTYConfig{
			MaxSessions:         50,
			MonitorInterval:     5 * time.Second,
			ReconnectTimeout:    30 * time.Second,
			HealthCheckInterval: 5 * time.Second,
		}
	}

	dpi := &DockerPTYIntegration{
		client:   dockerClient,
		sessions: make(map[string]*DockerPTYSession),
		config:   config,
		stopCh:   make(chan struct{}),
	}

	dpi.monitor = NewContainerMonitor(dockerClient, config.MonitorInterval)
	return dpi
}

// ConnectContainer creates a PTY connection to a Docker container
func (dpi *DockerPTYIntegration) ConnectContainer(ctx context.Context, containerID string, config *PTYSessionConfig) (*DockerPTYSession, error) {
	// 기본 설정 적용
	if config == nil {
		config = &PTYSessionConfig{
			Shell:        "/bin/sh",
			Term:         "xterm-256color",
			Rows:         24,
			Cols:         80,
			AttachStdin:  true,
			AttachStdout: true,
			AttachStderr: true,
		}
	}

	// 컨테이너 상태 확인
	containerInfo, err := dpi.client.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}

	if !containerInfo.State.Running {
		return nil, fmt.Errorf("container is not running: %s", containerInfo.State.Status)
	}

	// Exec 인스턴스 생성
	execConfig := types.ExecConfig{
		Cmd:          []string{config.Shell},
		AttachStdin:  config.AttachStdin,
		AttachStdout: config.AttachStdout,
		AttachStderr: config.AttachStderr,
		Tty:          true,
		Env:          dpi.buildEnvironment(config.Environment),
		WorkingDir:   config.WorkingDir,
		User:         config.User,
		Privileged:   config.Privileged,
	}

	execResp, err := dpi.client.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create exec: %w", err)
	}

	// PTY 연결 생성
	ptyConn, err := dpi.client.ContainerExecAttach(ctx, execResp.ID, types.ExecStartCheck{
		Detach: config.Detach,
		Tty:    true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to attach exec: %w", err)
	}

	// 세션 생성
	sessionCtx, cancel := context.WithCancel(ctx)
	session := &DockerPTYSession{
		SessionID:    generateSessionID(),
		ContainerID:  containerID,
		Container:    dpi.convertContainerInfo(&containerInfo),
		PTYConn:      ptyConn,
		ExecID:       execResp.ID,
		Config:       config,
		Status:       PTYConnecting,
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
		cancel:       cancel,
	}

	// Exec 시작
	if err := dpi.client.ContainerExecStart(sessionCtx, execResp.ID, types.ExecStartCheck{
		Detach: config.Detach,
		Tty:    true,
	}); err != nil {
		cancel()
		ptyConn.Close()
		return nil, fmt.Errorf("failed to start exec: %w", err)
	}

	session.Status = PTYActive

	// 세션 등록 및 모니터링 시작
	dpi.registerSession(session)
	go dpi.monitorSession(sessionCtx, session)

	return session, nil
}

// DisconnectContainer disconnects a PTY session
func (dpi *DockerPTYIntegration) DisconnectContainer(sessionID string) error {
	dpi.mutex.Lock()
	defer dpi.mutex.Unlock()

	session, exists := dpi.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	dpi.cleanupSession(session)
	delete(dpi.sessions, sessionID)
	return nil
}

// GetContainerInfo retrieves container information
func (dpi *DockerPTYIntegration) GetContainerInfo(containerID string) (*DockerContainer, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	containerInfo, err := dpi.client.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}

	return dpi.convertContainerInfo(&containerInfo), nil
}

// ListActiveSessions returns all active PTY sessions
func (dpi *DockerPTYIntegration) ListActiveSessions() map[string]*DockerPTYSession {
	dpi.mutex.RLock()
	defer dpi.mutex.RUnlock()

	sessions := make(map[string]*DockerPTYSession)
	for id, session := range dpi.sessions {
		sessions[id] = session
	}
	return sessions
}

// MonitorContainer returns resource statistics for a container
func (dpi *DockerPTYIntegration) MonitorContainer(sessionID string) (*ContainerResourceStats, error) {
	dpi.mutex.RLock()
	session, exists := dpi.sessions[sessionID]
	dpi.mutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	return dpi.monitor.GetContainerStats(session.ContainerID)
}

// RestartPTYConnection restarts a PTY connection
func (dpi *DockerPTYIntegration) RestartPTYConnection(sessionID string) error {
	dpi.mutex.RLock()
	session, exists := dpi.sessions[sessionID]
	dpi.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	go dpi.attemptReconnection(session)
	return nil
}

// Start starts the PTY integration manager
func (dpi *DockerPTYIntegration) Start() {
	dpi.monitor.Start()
}

// Stop stops the PTY integration manager
func (dpi *DockerPTYIntegration) Stop() {
	close(dpi.stopCh)
	dpi.monitor.Stop()

	// 모든 세션 정리
	dpi.mutex.Lock()
	for _, session := range dpi.sessions {
		dpi.cleanupSession(session)
	}
	dpi.sessions = make(map[string]*DockerPTYSession)
	dpi.mutex.Unlock()
}

// registerSession registers a new session
func (dpi *DockerPTYIntegration) registerSession(session *DockerPTYSession) {
	dpi.mutex.Lock()
	defer dpi.mutex.Unlock()
	dpi.sessions[session.SessionID] = session
	dpi.monitor.AddSession(session.SessionID, session.ContainerID)
}

// cleanupSession cleans up a session
func (dpi *DockerPTYIntegration) cleanupSession(session *DockerPTYSession) {
	if session.cancel != nil {
		session.cancel()
	}
	if session.PTYConn.Conn != nil {
		session.PTYConn.Close()
	}
	session.Status = PTYTerminated
	dpi.monitor.RemoveSession(session.SessionID)
}

// convertContainerInfo converts Docker container info to internal format
func (dpi *DockerPTYIntegration) convertContainerInfo(info *types.ContainerJSON) *DockerContainer {
	return &DockerContainer{
		ID:          info.ID,
		Name:        info.Name,
		Image:       info.Config.Image,
		Status:      info.State.Status,
		Networks:    info.NetworkSettings.Networks,
		Mounts:      info.Mounts,
		Environment: info.Config.Env,
		WorkingDir:  info.Config.WorkingDir,
		User:        info.Config.User,
	}
}

// buildEnvironment builds environment variables
func (dpi *DockerPTYIntegration) buildEnvironment(customEnv map[string]string) []string {
	defaultEnv := map[string]string{
		"TERM":   "xterm-256color",
		"LANG":   "en_US.UTF-8",
		"LC_ALL": "en_US.UTF-8",
		"PS1":    "\\u@\\h:\\w\\$ ",
		"HOME":   "/root",
		"PATH":   "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
	}

	// 사용자 정의 환경변수 병합
	for key, value := range customEnv {
		defaultEnv[key] = value
	}

	var envList []string
	for key, value := range defaultEnv {
		envList = append(envList, fmt.Sprintf("%s=%s", key, value))
	}

	return envList
}

// generateSessionID generates a unique session ID
func generateSessionID() string {
	return "pty-" + uuid.New().String()[:8]
}

// monitorSession monitors a PTY session
func (dpi *DockerPTYIntegration) monitorSession(ctx context.Context, session *DockerPTYSession) {
	ticker := time.NewTicker(dpi.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := dpi.checkContainerHealth(session); err != nil {
				log.Errorf("Container health check failed for session %s: %v", session.SessionID, err)

				if dpi.shouldReconnect(session, err) {
					go dpi.attemptReconnection(session)
				}
			}
		case <-ctx.Done():
			return
		case <-dpi.stopCh:
			return
		}
	}
}

// checkContainerHealth checks container health
func (dpi *DockerPTYIntegration) checkContainerHealth(session *DockerPTYSession) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	containerInfo, err := dpi.client.ContainerInspect(ctx, session.ContainerID)
	if err != nil {
		return fmt.Errorf("failed to inspect container: %w", err)
	}

	if !containerInfo.State.Running {
		return fmt.Errorf("container is not running: %s", containerInfo.State.Status)
	}

	// Exec 상태 확인
	execInfo, err := dpi.client.ContainerExecInspect(ctx, session.ExecID)
	if err != nil {
		return fmt.Errorf("failed to inspect exec: %w", err)
	}

	if execInfo.Running == false {
		return fmt.Errorf("exec process is not running")
	}

	return nil
}

// shouldReconnect determines if reconnection should be attempted
func (dpi *DockerPTYIntegration) shouldReconnect(session *DockerPTYSession, err error) bool {
	// 컨테이너가 재시작되었거나 exec이 종료된 경우 재연결 시도
	return session.Status == PTYActive && err != nil
}

// attemptReconnection attempts to reconnect a PTY session
func (dpi *DockerPTYIntegration) attemptReconnection(session *DockerPTYSession) {
	session.mutex.Lock()
	session.Status = PTYReconnecting
	session.mutex.Unlock()

	maxRetries := 3
	backoffDelay := time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		log.Infof("Attempting PTY reconnection for session %s (attempt %d/%d)",
			session.SessionID, attempt, maxRetries)

		ctx, cancel := context.WithTimeout(context.Background(), dpi.config.ReconnectTimeout)
		newSession, err := dpi.ConnectContainer(ctx, session.ContainerID, session.Config)
		cancel()

		if err == nil {
			// 기존 세션 정리
			dpi.cleanupSession(session)

			// 새 세션으로 교체
			newSession.SessionID = session.SessionID
			dpi.mutex.Lock()
			dpi.sessions[session.SessionID] = newSession
			dpi.mutex.Unlock()

			log.Infof("PTY reconnection successful for session %s", session.SessionID)
			return
		}

		log.Errorf("PTY reconnection attempt %d failed: %v", attempt, err)

		if attempt < maxRetries {
			time.Sleep(backoffDelay)
			backoffDelay *= 2
		}
	}

	// 재연결 실패
	session.mutex.Lock()
	session.Status = PTYError
	session.mutex.Unlock()

	dpi.DisconnectContainer(session.SessionID)
	log.Errorf("PTY reconnection failed for session %s after %d attempts",
		session.SessionID, maxRetries)
}