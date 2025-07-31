package docker

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
)

// PTYConfig PTY 세션 설정
type PTYConfig struct {
	Shell       string            `json:"shell"`        // /bin/bash, /bin/zsh 등
	WorkingDir  string            `json:"working_dir"`  // 초기 작업 디렉토리
	Environment map[string]string `json:"environment"`  // 환경 변수
	Size        PTYSize           `json:"size"`         // 터미널 크기
	User        string            `json:"user"`         // 실행 사용자
	Tty         bool              `json:"tty"`          // TTY 할당 여부
}

// PTYSize 터미널 크기
type PTYSize struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}


// ExecResult 명령 실행 결과
type ExecResult struct {
	ExitCode int    `json:"exit_code"`
	Output   string `json:"output,omitempty"`
	Error    string `json:"error,omitempty"`
	Duration time.Duration `json:"duration"`
}

// DockerPTYIntegration Docker PTY 통합 인터페이스
type DockerPTYIntegration interface {
	CreateContainerPTY(ctx context.Context, containerID string, config PTYConfig) (PTYSession, error)
	GetContainerPTYSessions(containerID string) []PTYSession
	AttachToPTY(ctx context.Context, sessionID string) (PTYSession, error)
	ExecuteCommand(ctx context.Context, containerID string, cmd []string) (*ExecResult, error)
	MonitorContainer(ctx context.Context, containerID string) (<-chan ContainerEvent, error)
	DetectShell(ctx context.Context, containerID string) (string, error)
	ValidateContainer(ctx context.Context, containerID string) error
}

// ContainerPTYManager 컨테이너 PTY 관리자
type ContainerPTYManager struct {
	dockerClient      ClientInterface
	ptyManager        PTYSessionManagement
	containerSessions map[string][]PTYSession
	execResults       map[string]*ExecResult
	monitors          map[string]<-chan ContainerEvent
	mu                sync.RWMutex
	ctx               context.Context
	cancel            context.CancelFunc
}

// NewContainerPTYManager 새로운 컨테이너 PTY 관리자 생성
func NewContainerPTYManager(dockerClient ClientInterface, ptyManager PTYSessionManagement) *ContainerPTYManager {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &ContainerPTYManager{
		dockerClient:      dockerClient,
		ptyManager:        ptyManager,
		containerSessions: make(map[string][]PTYSession),
		execResults:       make(map[string]*ExecResult),
		monitors:          make(map[string]<-chan ContainerEvent),
		ctx:               ctx,
		cancel:            cancel,
	}
}

// CreateContainerPTY 컨테이너에 새로운 PTY 세션 생성
func (cpm *ContainerPTYManager) CreateContainerPTY(ctx context.Context, containerID string, config PTYConfig) (PTYSession, error) {
	// 컨테이너 유효성 검증
	if err := cpm.ValidateContainer(ctx, containerID); err != nil {
		return nil, fmt.Errorf("container validation failed: %w", err)
	}

	// 기본값 설정
	if config.Shell == "" {
		detectedShell, err := cpm.DetectShell(ctx, containerID)
		if err != nil {
			// 기본 쉘로 폴백
			config.Shell = "/bin/bash"
		} else {
			config.Shell = detectedShell
		}
	}

	if config.WorkingDir == "" {
		config.WorkingDir = "/workspace"
	}

	if config.User == "" {
		config.User = "root"
	}

	if config.Size.Width == 0 {
		config.Size.Width = 80
	}

	if config.Size.Height == 0 {
		config.Size.Height = 24
	}

	// 향상된 PTY 세션 생성
	session := NewEnhancedPTYSession(containerID, cpm.dockerClient, config)
	
	// 세션 시작
	if err := session.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start PTY session: %w", err)
	}

	// 세션 등록
	cpm.mu.Lock()
	cpm.containerSessions[containerID] = append(cpm.containerSessions[containerID], session)
	cpm.mu.Unlock()

	return session, nil
}

// GetContainerPTYSessions 컨테이너의 모든 PTY 세션 조회
func (cpm *ContainerPTYManager) GetContainerPTYSessions(containerID string) []PTYSession {
	cpm.mu.RLock()
	defer cpm.mu.RUnlock()

	sessions := cpm.containerSessions[containerID]
	var activeSessions []PTYSession

	for _, session := range sessions {
		if session.IsAlive() {
			activeSessions = append(activeSessions, session)
		}
	}

	return activeSessions
}

// AttachToPTY 기존 PTY 세션에 연결
func (cpm *ContainerPTYManager) AttachToPTY(ctx context.Context, sessionID string) (PTYSession, error) {
	return cpm.ptyManager.GetSession(sessionID)
}

// ExecuteCommand 컨테이너에서 명령 실행
func (cpm *ContainerPTYManager) ExecuteCommand(ctx context.Context, containerID string, cmd []string) (*ExecResult, error) {
	startTime := time.Now()

	// Exec 설정 생성
	execConfig := ExecConfig{
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
		AttachStdin:  false,
		Tty:          false,
		User:         "root",
		WorkingDir:   "/workspace",
	}

	// Exec 인스턴스 생성
	createResp, err := cpm.dockerClient.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create exec instance: %w", err)
	}

	// Exec 시작
	startConfig := ExecStartConfig{
		Detach: false,
		Tty:    false,
	}

	hijacked, err := cpm.dockerClient.ContainerExecStart(ctx, createResp.ID, startConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to start exec instance: %w", err)
	}
	defer hijacked.Close()

	// 출력 읽기
	output, err := io.ReadAll(hijacked.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read exec output: %w", err)
	}

	// Exec 결과 조회
	inspect, err := cpm.dockerClient.ContainerExecInspect(ctx, createResp.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect exec instance: %w", err)
	}

	result := &ExecResult{
		ExitCode: inspect.ExitCode,
		Output:   string(output),
		Duration: time.Since(startTime),
	}

	if inspect.ExitCode != 0 {
		result.Error = fmt.Sprintf("command exited with code %d", inspect.ExitCode)
	}

	// 결과 저장
	cpm.mu.Lock()
	cpm.execResults[createResp.ID] = result
	cpm.mu.Unlock()

	return result, nil
}

// MonitorContainer 컨테이너 이벤트 모니터링
func (cpm *ContainerPTYManager) MonitorContainer(ctx context.Context, containerID string) (<-chan ContainerEvent, error) {
	cpm.mu.Lock()
	defer cpm.mu.Unlock()

	// 이미 모니터링 중인지 확인
	if eventChan, exists := cpm.monitors[containerID]; exists {
		return eventChan, nil
	}

	// 새 이벤트 채널 생성
	eventChan := make(chan ContainerEvent, 100)

	// 백그라운드에서 Docker 이벤트 스트림 모니터링
	go func() {
		defer close(eventChan)

		// Docker 이벤트 필터 설정
		eventFilters := filters.NewArgs()
		eventFilters.Add("container", containerID)
		eventFilters.Add("type", "container")
		
		eventFilter := types.EventsOptions{
			Filters: eventFilters,
		}

		eventStream, errStream := cpm.dockerClient.Events(ctx, eventFilter)

		for {
			select {
			case <-ctx.Done():
				return
			case err := <-errStream:
				if err != nil {
					fmt.Printf("Error monitoring container %s: %v\n", containerID, err)
					return
				}
			case event, ok := <-eventStream:
				if !ok {
					return
				}

				containerEvent := ContainerEvent{
					ContainerID: event.Actor.ID,
					WorkspaceID: event.Actor.Attributes["workspace.id"], // 워크스페이스 ID 추출
					Type:        ContainerEventType(event.Action),
					Status:      ContainerStateCreated, // 기본값, 필요시 매핑 로직 추가
					Message:     fmt.Sprintf("Container event: %s", event.Action),
					Timestamp:   time.Unix(event.Time, 0),
					Attributes:  event.Actor.Attributes,
				}

				select {
				case eventChan <- containerEvent:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	cpm.monitors[containerID] = eventChan
	return eventChan, nil
}

// DetectShell 컨테이너에서 사용 가능한 쉘 감지
func (cpm *ContainerPTYManager) DetectShell(ctx context.Context, containerID string) (string, error) {
	// 일반적인 쉘들을 우선순위대로 확인
	shells := []string{"/bin/bash", "/bin/zsh", "/bin/sh", "/bin/ash"}

	for _, shell := range shells {
		result, err := cpm.ExecuteCommand(ctx, containerID, []string{"test", "-x", shell})
		if err == nil && result.ExitCode == 0 {
			return shell, nil
		}
	}

	return "", fmt.Errorf("no compatible shell found in container")
}

// ValidateContainer 컨테이너 유효성 검증
func (cpm *ContainerPTYManager) ValidateContainer(ctx context.Context, containerID string) error {
	// 컨테이너 검사
	containerInfo, err := cpm.dockerClient.ContainerInspect(ctx, containerID)
	if err != nil {
		return fmt.Errorf("container not found: %w", err)
	}

	// 컨테이너 상태 확인
	if !containerInfo.State.Running {
		return fmt.Errorf("container %s is not running (state: %s)", containerID, containerInfo.State.Status)
	}

	// 컨테이너 헬스체크 (선택적)
	if containerInfo.State.Health != nil && containerInfo.State.Health.Status == "unhealthy" {
		return fmt.Errorf("container %s is unhealthy", containerID)
	}

	return nil
}

// CleanupInactiveSessions 비활성 세션 정리
func (cpm *ContainerPTYManager) CleanupInactiveSessions() {
	cpm.mu.Lock()
	defer cpm.mu.Unlock()

	for containerID, sessions := range cpm.containerSessions {
		var activeSessions []PTYSession
		
		for _, session := range sessions {
			if session.IsAlive() {
				activeSessions = append(activeSessions, session)
			} else {
				// 비활성 세션 정리
				session.Stop()
			}
		}
		
		cpm.containerSessions[containerID] = activeSessions
		
		// 빈 세션 리스트 정리
		if len(activeSessions) == 0 {
			delete(cpm.containerSessions, containerID)
		}
	}
}

// Shutdown 관리자 종료
func (cpm *ContainerPTYManager) Shutdown() error {
	cpm.mu.Lock()
	defer cpm.mu.Unlock()

	// 컨텍스트 취소
	cpm.cancel()

	// 모든 세션 종료
	for containerID, sessions := range cpm.containerSessions {
		for _, session := range sessions {
			if err := session.Stop(); err != nil {
				fmt.Printf("Warning: failed to stop PTY session for container %s: %v\n", containerID, err)
			}
		}
	}

	// 맵 정리
	cpm.containerSessions = make(map[string][]PTYSession)
	cpm.execResults = make(map[string]*ExecResult)
	cpm.monitors = make(map[string]<-chan ContainerEvent)

	return nil
}

// GetStats 관리자 통계 정보
func (cpm *ContainerPTYManager) GetStats() *ContainerPTYStats {
	cpm.mu.RLock()
	defer cpm.mu.RUnlock()

	stats := &ContainerPTYStats{
		TotalContainers:  len(cpm.containerSessions),
		TotalSessions:    0,
		ActiveSessions:   0,
		ContainerCounts:  make(map[string]int),
		MonitoredContainers: len(cpm.monitors),
	}

	for containerID, sessions := range cpm.containerSessions {
		stats.TotalSessions += len(sessions)
		
		activeSessions := 0
		for _, session := range sessions {
			if session.IsAlive() {
				activeSessions++
			}
		}
		
		stats.ActiveSessions += activeSessions
		stats.ContainerCounts[containerID] = activeSessions
	}

	return stats
}

// ContainerPTYStats 컨테이너 PTY 관리자 통계
type ContainerPTYStats struct {
	TotalContainers     int            `json:"total_containers"`
	TotalSessions       int            `json:"total_sessions"`
	ActiveSessions      int            `json:"active_sessions"`
	ContainerCounts     map[string]int `json:"container_counts"`
	MonitoredContainers int            `json:"monitored_containers"`
}