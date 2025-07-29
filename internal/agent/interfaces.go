package agent

import (
	"context"
	"time"

	"github.com/aicli/aicli-web/internal/git"
	"github.com/aicli/aicli-web/internal/models"
)

// AgentService 에이전트 관리 서비스 인터페이스
type AgentService interface {
	// 에이전트 생명주기 관리
	CreateAgent(ctx context.Context, req CreateAgentRequest) (*models.Agent, error)
	GetAgent(ctx context.Context, id string) (*models.Agent, error)
	GetAgentByProjectID(ctx context.Context, projectID string) ([]*models.Agent, error)
	UpdateAgent(ctx context.Context, id string, req UpdateAgentRequest) (*models.Agent, error)
	DeleteAgent(ctx context.Context, id string) error

	// 에이전트 상태 관리
	StartAgent(ctx context.Context, id string) error
	StopAgent(ctx context.Context, id string) error
	RestartAgent(ctx context.Context, id string) error
	GetAgentStatus(ctx context.Context, id string) (AgentStatusInfo, error)

	// 배치 작업
	StartMultipleAgents(ctx context.Context, ids []string) ([]AgentOperationResult, error)
	StopMultipleAgents(ctx context.Context, ids []string) ([]AgentOperationResult, error)

	// 모니터링 및 관리
	ListActiveAgents(ctx context.Context) ([]*models.Agent, error)
	GetHealthStatus(ctx context.Context, id string) (HealthStatus, error)
	GetAgentMetrics(ctx context.Context, id string) (AgentMetrics, error)

	// 정리 작업
	CleanupStaleAgents(ctx context.Context) (int, error)
	PerformMaintenance(ctx context.Context) error
}

// DockerAdapter Docker 통합 어댑터 인터페이스
type DockerAdapter interface {
	// 컨테이너 생명주기
	CreateContainer(ctx context.Context, config ContainerConfig) (*ContainerInfo, error)
	StartContainer(ctx context.Context, containerID string) error
	StopContainer(ctx context.Context, containerID string) error
	RemoveContainer(ctx context.Context, containerID string) error

	// 컨테이너 상태 관리
	GetContainerStatus(ctx context.Context, containerID string) (ContainerStatus, error)
	GetContainerHealth(ctx context.Context, containerID string) (HealthStatus, error)
	GetContainerMetrics(ctx context.Context, containerID string) (ContainerMetrics, error)

	// 로그 및 디버깅
	GetContainerLogs(ctx context.Context, containerID string, opts LogOptions) (LogStream, error)
	ExecuteCommand(ctx context.Context, containerID string, cmd []string) (ExecResult, error)
}

// MonitoringService 에이전트 모니터링 서비스 인터페이스
type MonitoringService interface {
	// 헬스체크
	CheckAgentHealth(ctx context.Context, agent *models.Agent) (HealthStatus, error)
	StartHealthMonitoring(ctx context.Context, agent *models.Agent) error
	StopHealthMonitoring(ctx context.Context, agentID string) error

	// 메트릭 수집
	CollectAgentMetrics(ctx context.Context, agent *models.Agent) (AgentMetrics, error)
	GetMetricsHistory(ctx context.Context, agentID string, period time.Duration) ([]AgentMetrics, error)

	// 이벤트 처리
	PublishAgentEvent(ctx context.Context, event AgentEvent) error
	SubscribeToAgentEvents(ctx context.Context, agentID string) (<-chan AgentEvent, error)
}

// EventPublisher 이벤트 발행 인터페이스
type EventPublisher interface {
	PublishAgentCreated(ctx context.Context, agent *models.Agent) error
	PublishAgentStarted(ctx context.Context, agent *models.Agent) error
	PublishAgentStopped(ctx context.Context, agent *models.Agent) error
	PublishAgentError(ctx context.Context, agent *models.Agent, err error) error
	PublishAgentDeleted(ctx context.Context, agentID string) error
}

// EventBus 이벤트 버스 인터페이스
type EventBus interface {
	// 기본 이벤트 발행/구독
	Publish(ctx context.Context, event AgentEvent) error
	Subscribe(ctx context.Context, agentID string) (<-chan AgentEvent, error)
	Unsubscribe(ctx context.Context, agentID string) error
	
	// 전역 구독 (모든 이벤트)
	SubscribeGlobal(ctx context.Context) (<-chan AgentEvent, error)
	UnsubscribeGlobal(ctx context.Context, eventChan <-chan AgentEvent) error
	
	// 이벤트 히스토리
	GetEventHistory(agentID string, since time.Time) ([]AgentEvent, error)
}

// MetricsCollector 메트릭 수집 인터페이스
type MetricsCollector interface {
	CollectAgentMetrics(ctx context.Context, agent *models.Agent) (AgentMetrics, error)
	StoreMetrics(ctx context.Context, metrics AgentMetrics) error
	GetMetricsHistory(ctx context.Context, agentID string, period time.Duration) ([]AgentMetrics, error)
}

// 요청/응답 모델

// CreateAgentRequest 에이전트 생성 요청
type CreateAgentRequest struct {
	ProjectID    string              `json:"project_id" validate:"required,uuid"`
	Name         string              `json:"name" validate:"required,min=1,max=100"`
	Type         models.AgentType    `json:"type" validate:"required"`
	Config       models.AgentConfig  `json:"config" validate:"-"`
	Description  string              `json:"description,omitempty" validate:"omitempty,max=500"`
	WorktreeOpts git.WorktreeOptions `json:"worktree_opts,omitempty" validate:"-"`
}

// UpdateAgentRequest 에이전트 업데이트 요청
type UpdateAgentRequest struct {
	Name        *string             `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
	Config      *models.AgentConfig `json:"config,omitempty" validate:"-"`
	Description *string             `json:"description,omitempty" validate:"omitempty,max=500"`
}

// AgentOperationResult 에이전트 작업 결과
type AgentOperationResult struct {
	AgentID string `json:"agent_id"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// AgentStatusInfo 에이전트 상태 정보
type AgentStatusInfo struct {
	ID           string               `json:"id"`
	Status       models.AgentStatus   `json:"status"`
	ContainerID  string               `json:"container_id,omitempty"`
	WorktreeID   string               `json:"worktree_id,omitempty"`
	LastActivity time.Time            `json:"last_activity"`
	ErrorMessage string               `json:"error_message,omitempty"`
	Health       HealthStatus         `json:"health"`
	Container    *ContainerStatus     `json:"container,omitempty"`
}

// ContainerConfig 컨테이너 설정
type ContainerConfig struct {
	Image        string            `json:"image"`
	Environment  map[string]string `json:"environment,omitempty"`
	WorkingDir   string            `json:"working_dir,omitempty"`
	Mounts       []MountConfig     `json:"mounts,omitempty"`
	MemoryLimit  string            `json:"memory_limit,omitempty"`
	CPULimit     string            `json:"cpu_limit,omitempty"`
	NetworkMode  string            `json:"network_mode,omitempty"`
	Labels       map[string]string `json:"labels,omitempty"`
}

// MountConfig 마운트 설정
type MountConfig struct {
	Source      string `json:"source"`
	Target      string `json:"target"`
	Type        string `json:"type"` // bind, volume, tmpfs
	ReadOnly    bool   `json:"read_only,omitempty"`
	BindOptions *struct {
		Propagation string `json:"propagation,omitempty"`
	} `json:"bind_options,omitempty"`
}

// ContainerInfo 컨테이너 정보
type ContainerInfo struct {
	ID      string            `json:"id"`
	Name    string            `json:"name"`
	Image   string            `json:"image"`
	Status  string            `json:"status"`
	Ports   map[string]string `json:"ports,omitempty"`
	Labels  map[string]string `json:"labels,omitempty"`
	Created time.Time         `json:"created"`
}

// ContainerStatus 컨테이너 상태
type ContainerStatus struct {
	ID         string    `json:"id"`
	Status     string    `json:"status"`
	Health     string    `json:"health,omitempty"`
	ExitCode   int       `json:"exit_code,omitempty"`
	StartedAt  time.Time `json:"started_at,omitempty"`
	FinishedAt time.Time `json:"finished_at,omitempty"`
}

// HealthStatus 헬스 상태
type HealthStatus struct {
	Status      string            `json:"status"` // healthy, unhealthy, starting, unknown
	LastCheck   time.Time         `json:"last_check"`
	Checks      []HealthCheck     `json:"checks,omitempty"`
	Metrics     map[string]string `json:"metrics,omitempty"`
}

// HealthCheck 개별 헬스체크
type HealthCheck struct {
	Name        string    `json:"name"`
	Status      string    `json:"status"`
	Message     string    `json:"message,omitempty"`
	CheckedAt   time.Time `json:"checked_at"`
	Duration    time.Duration `json:"duration"`
}

// AgentMetrics 에이전트 메트릭
type AgentMetrics struct {
	AgentID     string                 `json:"agent_id"`
	Timestamp   time.Time              `json:"timestamp"`
	CPU         CPUMetrics             `json:"cpu"`
	Memory      MemoryMetrics          `json:"memory"`
	Network     NetworkMetrics         `json:"network"`
	Disk        DiskMetrics            `json:"disk"`
	Custom      map[string]interface{} `json:"custom,omitempty"`
}

// CPUMetrics CPU 메트릭
type CPUMetrics struct {
	UsagePercent float64 `json:"usage_percent"`
	SystemTime   float64 `json:"system_time"`
	UserTime     float64 `json:"user_time"`
}

// MemoryMetrics 메모리 메트릭
type MemoryMetrics struct {
	UsageBytes  int64   `json:"usage_bytes"`
	LimitBytes  int64   `json:"limit_bytes"`
	UsagePercent float64 `json:"usage_percent"`
}

// NetworkMetrics 네트워크 메트릭
type NetworkMetrics struct {
	RxBytes   int64 `json:"rx_bytes"`
	TxBytes   int64 `json:"tx_bytes"`
	RxPackets int64 `json:"rx_packets"`
	TxPackets int64 `json:"tx_packets"`
}

// DiskMetrics 디스크 메트릭
type DiskMetrics struct {
	ReadBytes  int64 `json:"read_bytes"`
	WriteBytes int64 `json:"write_bytes"`
	ReadOps    int64 `json:"read_ops"`
	WriteOps   int64 `json:"write_ops"`
}

// ContainerMetrics 컨테이너 메트릭
type ContainerMetrics struct {
	ContainerID string         `json:"container_id"`
	Timestamp   time.Time      `json:"timestamp"`
	CPU         CPUMetrics     `json:"cpu"`
	Memory      MemoryMetrics  `json:"memory"`
	Network     NetworkMetrics `json:"network"`
	BlockIO     DiskMetrics    `json:"block_io"`
}

// LogOptions 로그 조회 옵션
type LogOptions struct {
	Follow     bool      `json:"follow"`
	Timestamps bool      `json:"timestamps"`
	Since      time.Time `json:"since,omitempty"`
	Until      time.Time `json:"until,omitempty"`
	Tail       int       `json:"tail,omitempty"`
}

// LogStream 로그 스트림
type LogStream interface {
	Read() ([]byte, error)
	Close() error
}

// ExecResult 명령 실행 결과
type ExecResult struct {
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
}

// AgentEvent 에이전트 이벤트
type AgentEvent struct {
	Type      AgentEventType `json:"type"`
	AgentID   string         `json:"agent_id"`
	Timestamp time.Time      `json:"timestamp"`
	Data      interface{}    `json:"data,omitempty"`
	Message   string         `json:"message,omitempty"`
}

// AgentEventType 에이전트 이벤트 타입
type AgentEventType string

const (
	AgentEventCreated           AgentEventType = "agent.created"
	AgentEventStarting          AgentEventType = "agent.starting"
	AgentEventStarted           AgentEventType = "agent.started"
	AgentEventStopping          AgentEventType = "agent.stopping"
	AgentEventStopped           AgentEventType = "agent.stopped"
	AgentEventError             AgentEventType = "agent.error"
	AgentEventHealthCheckFailed AgentEventType = "agent.health_check_failed"
	AgentEventDeleted           AgentEventType = "agent.deleted"
)

// 에러 타입

// ServiceError 서비스 레벨 에러
type ServiceError struct {
	Code    string
	Message string
	Cause   error
}

func (e *ServiceError) Error() string {
	return e.Message
}

func (e *ServiceError) Unwrap() error {
	return e.Cause
}

// 에러 코드 상수
const (
	ErrCodeAgentNotFound       = "AGENT_NOT_FOUND"
	ErrCodeAgentAlreadyExists  = "AGENT_ALREADY_EXISTS"
	ErrCodeInvalidState        = "INVALID_STATE"
	ErrCodeContainerError      = "CONTAINER_ERROR"
	ErrCodeWorktreeError       = "WORKTREE_ERROR"
	ErrCodeResourceLimit       = "RESOURCE_LIMIT"
	ErrCodePermissionDenied    = "PERMISSION_DENIED"
	ErrCodeInternalError       = "INTERNAL_ERROR"
)