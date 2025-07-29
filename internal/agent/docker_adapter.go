package agent

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"aicli-web/internal/docker"
	"aicli-web/internal/models"
)

// dockerAdapter Docker 컨테이너 관리를 위한 어댑터 구현체
type dockerAdapter struct {
	containerManager docker.ContainerManagement
	client          docker.ClientInterface
	statsCollector  docker.StatsCollection
	agentIntegration *DockerAgentIntegration // 에이전트 전용 Docker 통합
}

// NewDockerAdapter 새로운 Docker 어댑터를 생성합니다.
func NewDockerAdapter(containerManager docker.ContainerManagement, client docker.ClientInterface, statsCollector docker.StatsCollection) DockerAdapter {
	return &dockerAdapter{
		containerManager: containerManager,
		client:          client,
		statsCollector:  statsCollector,
		agentIntegration: nil, // 나중에 주입
	}
}

// SetAgentIntegration 에이전트 Docker 통합을 설정합니다.
func (d *dockerAdapter) SetAgentIntegration(integration *DockerAgentIntegration) {
	d.agentIntegration = integration
}

// CreateContainer 새로운 컨테이너를 생성합니다.
func (d *dockerAdapter) CreateContainer(ctx context.Context, config ContainerConfig) (*ContainerInfo, error) {
	// Agent 컨테이너인 경우 전용 통합 사용
	if config.Labels != nil && config.Labels["type"] == "agent" {
		// Agent 모델 생성 (임시)
		agent := &models.Agent{
			ID:        config.Labels["agent_id"],
			ProjectID: config.Labels["project_id"],
			Name:      config.Labels["agent_name"],
			Type:      models.AgentType(config.Labels["agent_type"]),
			Config:    make(map[string]interface{}),
		}
		
		// 환경 변수에서 설정 추출
		for k, v := range config.Environment {
			if k == "CLAUDE_API_KEY" {
				agent.Config["claude_api_key"] = v
			} else if k == "GIT_USER_NAME" {
				agent.Config["git_user_name"] = v
			} else if k == "GIT_USER_EMAIL" {
				agent.Config["git_user_email"] = v
			}
		}
		
		// 에이전트 통합이 설정되지 않은 경우 기본 로직으로 폴백
		if d.agentIntegration == nil {
			return nil, fmt.Errorf("에이전트 Docker 통합이 설정되지 않음")
		}
		
		// 네트워크 확인
		if err := d.agentIntegration.EnsureAgentNetwork(ctx); err != nil {
			return nil, fmt.Errorf("에이전트 네트워크 생성 실패: %w", err)
		}
		
		// 에이전트 컨테이너 생성
		containerID, err := d.agentIntegration.CreateAgentContainer(ctx, agent, config.WorkingDir)
		if err != nil {
			return nil, fmt.Errorf("에이전트 컨테이너 생성 실패: %w", err)
		}
		
		return &ContainerInfo{
			ID:      containerID,
			Name:    fmt.Sprintf("aicli-agent-%s", agent.ID),
			Status:  "created",
			Created: time.Now(),
		}, nil
	}
	
	// 기존 로직 유지
	createReq, err := d.convertToCreateRequest(config)
	if err != nil {
		return nil, fmt.Errorf("컨테이너 설정 변환 실패: %w", err)
	}

	// 컨테이너 생성
	wsContainer, err := d.containerManager.CreateWorkspaceContainer(ctx, createReq)
	if err != nil {
		return nil, fmt.Errorf("컨테이너 생성 실패: %w", err)
	}

	// 컨테이너 정보 변환
	return d.convertToContainerInfo(wsContainer), nil
}

// StartContainer 컨테이너를 시작합니다.
func (d *dockerAdapter) StartContainer(ctx context.Context, containerID string) error {
	// 에이전트 컨테이너인지 확인
	containerInfo, err := d.client.ContainerInspect(ctx, containerID)
	if err == nil && containerInfo.Config != nil && containerInfo.Config.Labels != nil {
		if containerInfo.Config.Labels[LabelAgentID] != "" {
			// 에이전트 전용 시작 로직 사용
			if d.agentIntegration != nil {
				return d.agentIntegration.StartAgentContainer(ctx, containerID)
			}
		}
	}
	
	// 기존 로직
	err = d.containerManager.StartContainer(ctx, containerID)
	if err != nil {
		return fmt.Errorf("컨테이너 시작 실패: %w", err)
	}
	return nil
}

// StopContainer 컨테이너를 중지합니다.
func (d *dockerAdapter) StopContainer(ctx context.Context, containerID string) error {
	// 에이전트 컨테이너인지 확인
	containerInfo, err := d.client.ContainerInspect(ctx, containerID)
	if err == nil && containerInfo.Config != nil && containerInfo.Config.Labels != nil {
		if containerInfo.Config.Labels[LabelAgentID] != "" {
			// 에이전트 전용 중지 로직 사용
			if d.agentIntegration != nil {
				return d.agentIntegration.StopAgentContainer(ctx, containerID)
			}
		}
	}
	
	// 기존 로직
	timeout := 30 * time.Second
	err = d.containerManager.StopContainer(ctx, containerID, timeout)
	if err != nil {
		return fmt.Errorf("컨테이너 중지 실패: %w", err)
	}
	return nil
}

// RemoveContainer 컨테이너를 제거합니다.
func (d *dockerAdapter) RemoveContainer(ctx context.Context, containerID string) error {
	// 에이전트 컨테이너인지 확인
	containerInfo, err := d.client.ContainerInspect(ctx, containerID)
	if err == nil && containerInfo.Config != nil && containerInfo.Config.Labels != nil {
		if agentID := containerInfo.Config.Labels[LabelAgentID]; agentID != "" {
			// 에이전트 전용 제거 로직 사용
			if d.agentIntegration != nil {
				if err := d.agentIntegration.RemoveAgentContainer(ctx, containerID); err != nil {
					return err
				}
				// 볼륨도 정리
				return d.agentIntegration.CleanupAgentVolumes(ctx, agentID)
			}
		}
	}
	
	// 기존 로직
	err = d.containerManager.RemoveContainer(ctx, containerID, true) // force=true
	if err != nil {
		return fmt.Errorf("컨테이너 제거 실패: %w", err)
	}
	return nil
}

// GetContainerStatus 컨테이너 상태를 조회합니다.
func (d *dockerAdapter) GetContainerStatus(ctx context.Context, containerID string) (ContainerStatus, error) {
	wsContainer, err := d.containerManager.InspectContainer(ctx, containerID)
	if err != nil {
		return ContainerStatus{}, fmt.Errorf("컨테이너 조회 실패: %w", err)
	}

	status := ContainerStatus{
		ID:     wsContainer.ID,
		Status: string(wsContainer.State),
	}

	if wsContainer.Started != nil {
		status.StartedAt = *wsContainer.Started
	}
	if wsContainer.Finished != nil {
		status.FinishedAt = *wsContainer.Finished
	}
	if wsContainer.ExitCode != nil {
		status.ExitCode = *wsContainer.ExitCode
	}

	return status, nil
}

// GetContainerHealth 컨테이너 헬스 상태를 확인합니다.
func (d *dockerAdapter) GetContainerHealth(ctx context.Context, containerID string) (HealthStatus, error) {
	// Docker inspect를 통해 헬스체크 정보 조회
	containerJSON, err := d.client.ContainerInspect(ctx, containerID)
	if err != nil {
		return HealthStatus{}, fmt.Errorf("컨테이너 정보 조회 실패: %w", err)
	}

	health := HealthStatus{
		Status:    "unknown",
		LastCheck: time.Now(),
	}

	// 컨테이너가 실행 중인지 확인
	if containerJSON.State != nil {
		if containerJSON.State.Running {
			health.Status = "healthy"
		} else if containerJSON.State.Dead || containerJSON.State.ExitCode != 0 {
			health.Status = "unhealthy"
		}

		// 헬스체크 정보가 있다면 추가
		if containerJSON.State.Health != nil {
			health.Status = strings.ToLower(containerJSON.State.Health.Status)
			
			if len(containerJSON.State.Health.Log) > 0 {
				lastCheck := containerJSON.State.Health.Log[len(containerJSON.State.Health.Log)-1]
				health.LastCheck = lastCheck.Start
				
				// 최근 체크들을 변환
				for _, log := range containerJSON.State.Health.Log {
					check := HealthCheck{
						Name:      "docker-healthcheck",
						Status:    fmt.Sprintf("%d", log.ExitCode),
						Message:   log.Output,
						CheckedAt: log.Start,
						Duration:  log.End.Sub(log.Start),
					}
					if log.ExitCode == 0 {
						check.Status = "passing"
					} else {
						check.Status = "failing"
					}
					health.Checks = append(health.Checks, check)
				}
			}
		}
	}

	return health, nil
}

// GetContainerMetrics 컨테이너 메트릭을 조회합니다.
func (d *dockerAdapter) GetContainerMetrics(ctx context.Context, containerID string) (ContainerMetrics, error) {
	// StatsCollection 인터페이스를 통해 메트릭 수집
	containerStats, err := d.statsCollector.Collect(ctx, containerID)
	if err != nil {
		return ContainerMetrics{}, fmt.Errorf("컨테이너 통계 조회 실패: %w", err)
	}

	// 메트릭 변환
	metrics := ContainerMetrics{
		ContainerID: containerID,
		Timestamp:   time.Now(),
	}

	if containerStats != nil {
		// 기본적인 메트릭만 설정 (실제 변환은 containerStats 구조에 따라 달라짐)
		metrics.CPU.UsagePercent = 0.0 // 실제 구현 시 containerStats에서 가져와야 함
		metrics.Memory.UsageBytes = 0  // 실제 구현 시 containerStats에서 가져와야 함
	}

	return metrics, nil
}

// GetContainerLogs 컨테이너 로그를 조회합니다.
func (d *dockerAdapter) GetContainerLogs(ctx context.Context, containerID string, opts LogOptions) (LogStream, error) {
	// 간단한 로그 스트림 구현 (향후 개선 필요)
	// 현재는 기본적인 구현만 제공
	return &simpleLogStream{
		containerID: containerID,
		opts:        opts,
	}, nil
}

// ExecuteCommand 컨테이너에서 명령을 실행합니다.
func (d *dockerAdapter) ExecuteCommand(ctx context.Context, containerID string, cmd []string) (ExecResult, error) {
	// Exec 설정
	execConfig := docker.ExecConfig{
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
		AttachStdin:  false,
		Tty:          false,
	}

	// Exec 생성
	execResp, err := d.client.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return ExecResult{}, fmt.Errorf("exec 생성 실패: %w", err)
	}

	// Exec 시작
	startConfig := docker.ExecStartConfig{
		Detach: false,
		Tty:    false,
	}

	hijackedResp, err := d.client.ContainerExecStart(ctx, execResp.ID, startConfig)
	if err != nil {
		return ExecResult{}, fmt.Errorf("exec 시작 실패: %w", err)
	}
	defer func() {
		if hijackedResp.Conn != nil {
			hijackedResp.Close()
		}
	}()

	// 출력 읽기 (안전하게 처리)
	var stdout, stderr string
	if hijackedResp.Reader != nil {
		var err error
		stdout, stderr, err = d.readExecOutput(hijackedResp.Reader)
		if err != nil {
			return ExecResult{}, fmt.Errorf("exec 출력 읽기 실패: %w", err)
		}
	}

	// Exec 상태 확인
	execInspect, err := d.client.ContainerExecInspect(ctx, execResp.ID)
	if err != nil {
		return ExecResult{}, fmt.Errorf("exec 상태 조회 실패: %w", err)
	}

	return ExecResult{
		ExitCode: execInspect.ExitCode,
		Stdout:   stdout,
		Stderr:   stderr,
	}, nil
}

// convertToCreateRequest 컨테이너 설정을 Docker 생성 요청으로 변환
func (d *dockerAdapter) convertToCreateRequest(config ContainerConfig) (*docker.CreateContainerRequest, error) {
	// 메모리 제한 파싱
	var memoryLimit int64
	if config.MemoryLimit != "" {
		if limit, err := parseResourceLimit(config.MemoryLimit); err == nil {
			memoryLimit = limit
		}
	}

	// CPU 제한 파싱
	var cpuLimit float64
	if config.CPULimit != "" {
		if limit, err := strconv.ParseFloat(config.CPULimit, 64); err == nil {
			cpuLimit = limit
		}
	}

	// 포트 매핑 변환 (간단히 구현)
	ports := make(map[string]string)
	for k, v := range config.Labels {
		if strings.HasPrefix(k, "port.") {
			ports[strings.TrimPrefix(k, "port.")] = v
		}
	}

	createReq := &docker.CreateContainerRequest{
		WorkspaceID: "agent", // 기본값
		Name:        fmt.Sprintf("agent-%d", time.Now().Unix()),
		ProjectPath: "/workspace", // 기본값
		Image:       config.Image,
		Environment: config.Environment,
		WorkingDir:  config.WorkingDir,
		MemoryLimit: memoryLimit,
		CPULimit:    cpuLimit,
		Ports:       ports,
	}

	return createReq, nil
}

// convertToContainerInfo 워크스페이스 컨테이너를 컨테이너 정보로 변환
func (d *dockerAdapter) convertToContainerInfo(wsContainer *docker.WorkspaceContainer) *ContainerInfo {
	info := &ContainerInfo{
		ID:      wsContainer.ID,
		Name:    wsContainer.Name,
		Status:  string(wsContainer.State),
		Created: wsContainer.Created,
	}

	if wsContainer.Ports != nil {
		info.Ports = wsContainer.Ports
	}

	return info
}

// envMapToSlice 환경변수 맵을 슬라이스로 변환
func envMapToSlice(envMap map[string]string) []string {
	var envSlice []string
	for key, value := range envMap {
		envSlice = append(envSlice, fmt.Sprintf("%s=%s", key, value))
	}
	return envSlice
}

// parseResourceLimit 리소스 제한 문자열 파싱
func parseResourceLimit(limit string) (int64, error) {
	if limit == "" {
		return 0, nil
	}

	// 간단한 파싱 (예: "512m" -> 512 * 1024 * 1024)
	if strings.HasSuffix(limit, "m") || strings.HasSuffix(limit, "M") {
		num, err := strconv.ParseInt(limit[:len(limit)-1], 10, 64)
		if err != nil {
			return 0, err
		}
		return num * 1024 * 1024, nil
	}

	if strings.HasSuffix(limit, "g") || strings.HasSuffix(limit, "G") {
		num, err := strconv.ParseInt(limit[:len(limit)-1], 10, 64)
		if err != nil {
			return 0, err
		}
		return num * 1024 * 1024 * 1024, nil
	}

	// 숫자만 있는 경우 바이트로 간주
	return strconv.ParseInt(limit, 10, 64)
}

// readExecOutput Exec 출력을 읽어서 stdout, stderr로 분리
func (d *dockerAdapter) readExecOutput(reader io.Reader) (stdout, stderr string, err error) {
	// Docker의 exec 출력은 multiplexed stream이므로 적절히 파싱해야 함
	// 간단한 구현으로 전체를 stdout으로 처리
	output, err := io.ReadAll(reader)
	if err != nil {
		return "", "", err
	}
	
	return string(output), "", nil
}

// simpleLogStream 간단한 로그 스트림 구현체
type simpleLogStream struct {
	containerID string
	opts        LogOptions
	closed      bool
}

func (ls *simpleLogStream) Read() ([]byte, error) {
	if ls.closed {
		return nil, io.EOF
	}
	
	// 간단한 로그 메시지 반환 (실제 구현에서는 Docker API 호출 필요)
	logMessage := fmt.Sprintf("[%s] Container %s log entry\n", time.Now().Format(time.RFC3339), ls.containerID)
	ls.closed = true // 한 번만 읽고 종료
	return []byte(logMessage), nil
}

func (ls *simpleLogStream) Close() error {
	ls.closed = true
	return nil
}