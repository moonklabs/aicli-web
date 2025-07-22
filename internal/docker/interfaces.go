package docker

import (
	"context"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
	
	"github.com/aicli/aicli-web/internal/models"
	mountpkg "github.com/aicli/aicli-web/internal/docker/mount"
	securitypkg "github.com/aicli/aicli-web/internal/docker/security"
)

// DockerClient Docker 클라이언트 인터페이스
type DockerClient interface {
	// 기본 연결
	Ping(ctx context.Context) error
	Close() error
	
	// 설정 관리
	GetConfig() *Config
	GetNetworkID() string
}

// NetworkManagement 네트워크 관리 인터페이스
type NetworkManagement interface {
	CreateNetwork(ctx context.Context, req CreateNetworkRequest) (*NetworkInfo, error)
	GetNetwork(ctx context.Context, networkID string) (*NetworkInfo, error)
	ListNetworks(ctx context.Context) ([]NetworkInfo, error)
	DeleteNetwork(ctx context.Context, networkID string) error
	ConnectContainer(ctx context.Context, networkID, containerID string, config *network.EndpointSettings) error
	DisconnectContainer(ctx context.Context, networkID, containerID string, force bool) error
	EnsureNetwork(ctx context.Context, name string) (string, error)
	CleanupNetworks(ctx context.Context) error
}

// HealthMonitoring 헬스체크 및 모니터링 인터페이스
type HealthMonitoring interface {
	CheckDaemon(ctx context.Context) error
	CheckContainer(ctx context.Context, containerID string) (bool, error)
	GetSystemInfo(ctx context.Context) (*types.Info, error)
	GetVersion(ctx context.Context) (types.Version, error)
	WaitHealthy(ctx context.Context, containerID string, timeout time.Duration) error
	StartMonitoring(ctx context.Context, callback func(error))
}

// StatsCollection 통계 수집 인터페이스
type StatsCollection interface {
	Collect(ctx context.Context, containerID string) (*ContainerStats, error)
	CollectAll(ctx context.Context) (map[string]*ContainerStats, error)
	GetSystemStats(ctx context.Context) (*SystemStats, error)
	GetAggregatedStats(ctx context.Context) (*AggregatedStats, error)
	Monitor(ctx context.Context, containerID string, interval time.Duration) (<-chan *ContainerStats, error)
	MonitorAll(ctx context.Context, interval time.Duration) (<-chan map[string]*ContainerStats, error)
}

// DockerFactory Docker 구성 요소 팩토리 인터페이스
type DockerFactory interface {
	GetClient() DockerClient
	GetNetworkManager() NetworkManagement
	GetStatsCollector() StatsCollection
	GetHealthChecker() HealthMonitoring
	GetMountManager() MountManagement
	IsHealthy(ctx context.Context) (bool, error)
	Ping(ctx context.Context) error
	Close() error
}

// DockerManager 통합 Docker 관리 인터페이스
type DockerManager interface {
	GetFactory() DockerFactory
	Client() DockerClient
	Network() NetworkManagement
	Stats() StatsCollection
	Health() HealthMonitoring
	Mount() MountManagement
	Config() *Config
	Context() context.Context
	GetSystemStatus(ctx context.Context) (*SystemStatus, error)
	Cleanup(ctx context.Context) error
	Shutdown() error
}

// WorkspaceContainerInterface 워크스페이스 컨테이너 인터페이스
type WorkspaceContainerInterface interface {
	GetID() string
	GetName() string
	GetWorkspaceID() string
	GetState() string
	GetCreatedAt() time.Time
}

// ContainerManagement 컨테이너 관리 인터페이스
type ContainerManagement interface {
	CreateWorkspaceContainer(ctx context.Context, req *CreateContainerRequest) (*WorkspaceContainer, error)
	InspectContainer(ctx context.Context, containerID string) (*WorkspaceContainer, error)
	ListWorkspaceContainers(ctx context.Context, workspaceID string) ([]*WorkspaceContainer, error)
	ListContainers(ctx context.Context, labels map[string]string) ([]*WorkspaceContainer, error)
	StartContainer(ctx context.Context, containerID string) error
	StopContainer(ctx context.Context, containerID string, timeout time.Duration) error
	RestartContainer(ctx context.Context, containerID string, timeout time.Duration) error
	RemoveContainer(ctx context.Context, containerID string, force bool) error
	CleanupWorkspace(ctx context.Context, workspaceID string, force bool) error
}

// ContainerManagerInterface 통합 테스트용 인터페이스
type ContainerManagerInterface interface {
	ContainerManagement
	LifecycleManagement
}

// StatusTrackerInterface 상태 추적 인터페이스
type StatusTrackerInterface interface {
	StateTracking
	ResourceMonitoring
	MetricsCollection
}

// NetworkManagerInterface 네트워크 관리 인터페이스
type NetworkManagerInterface interface {
	NetworkManagement
	SecurityNetworkManagement
}

// MountManagerInterface 마운트 관리 인터페이스
type MountManagerInterface interface {
	MountManagement
}

// ClientInterface Docker 클라이언트 인터페이스 확장
type ClientInterface interface {
	DockerClient
	NetworkManagement
	HealthMonitoring
	StatsCollection
	
	// Docker API 메서드 추가
	ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error)
	ContainerKill(ctx context.Context, containerID string, signal string) error
	ContainerRemove(ctx context.Context, containerID string, options types.ContainerRemoveOptions) error
	ContainerExecCreate(ctx context.Context, containerID string, config ExecConfig) (types.IDResponse, error)
	ContainerExecStart(ctx context.Context, execID string, config ExecStartConfig) (types.HijackedResponse, error)
	ContainerExecInspect(ctx context.Context, execID string) (types.ContainerExecInspect, error)
}

// LifecycleManagement 컨테이너 생명주기 이벤트 관리 인터페이스
type LifecycleManagement interface {
	Subscribe(workspaceID string, handler ContainerEventHandler)
	Unsubscribe(workspaceID string)
	GetContainerHistory(ctx context.Context, containerID string, since time.Time) ([]ContainerEvent, error)
	WaitForContainerState(ctx context.Context, containerID string, targetState ContainerState, timeout time.Duration) error
	Close()
}

// MountManagement 마운트 관리 인터페이스
type MountManagement interface {
	CreateWorkspaceMount(workspace *models.Workspace) (*mountpkg.MountConfig, error)
	CreateCustomMount(req *mountpkg.CreateMountRequest) (*mountpkg.MountConfig, error)
	ValidateMountConfig(config *mountpkg.MountConfig) error
	ToDockerMount(config *mountpkg.MountConfig) (mount.Mount, error)
	GetMountStatus(ctx context.Context, config *mountpkg.MountConfig) (*mountpkg.MountStatus, error)
	RefreshMountConfig(config *mountpkg.MountConfig) error
	StartFileWatcher(ctx context.Context, sourcePath string, excludePatterns []string, callback func([]string)) error
	StopFileWatcher(sourcePath string)
	GetFileStats(ctx context.Context, sourcePath string, excludePatterns []string) (*mountpkg.FileStats, error)
	GetActiveWatchers() []string
	StopAllWatchers()
}

// StateTracking 상태 추적 인터페이스
type StateTracking interface {
	Start() error
	Stop() error
	GetWorkspaceState(workspaceID string) (interface{}, bool)
	GetAllWorkspaceStates() map[string]interface{}
	ForceSync(workspaceID string) error
	OnStateChange(callback interface{})
	GetStats() interface{}
}

// ResourceMonitoring 리소스 모니터링 인터페이스
type ResourceMonitoring interface {
	Start() error
	Stop() error
	StartMonitoring(ctx context.Context, workspaceID string) (<-chan interface{}, error)
	StopMonitoring(workspaceID string)
	GetResourceSummary(ctx context.Context) (interface{}, error)
	GetActiveMonitors() []string
	GetMonitorStats() interface{}
}

// MetricsCollection 메트릭 수집 인터페이스
type MetricsCollection interface {
	Start() error
	Stop() error
	GetContainerMetrics(containerID string) (interface{}, bool)
	GetWorkspaceMetrics(workspaceID string) []interface{}
	GetAllMetrics() map[string]interface{}
	GetStats() interface{}
}

// IsolationManagement 격리 관리 인터페이스
type IsolationManagement interface {
	CreateWorkspaceIsolation(workspace *models.Workspace) (*securitypkg.WorkspaceIsolation, error)
	ValidateIsolation(isolation *securitypkg.WorkspaceIsolation) error
	ApplyToContainer(isolation *securitypkg.WorkspaceIsolation, config *container.Config, hostConfig *container.HostConfig) error
	UpdateConfig(config *securitypkg.IsolationConfig) error
	GetConfig() *securitypkg.IsolationConfig
}

// SecurityNetworkManagement 보안 네트워크 관리 인터페이스
type SecurityNetworkManagement interface {
	CreateWorkspaceNetwork(ctx context.Context, workspaceID string) (*NetworkInfo, error)
	GetWorkspaceNetwork(ctx context.Context, workspaceID string) (*NetworkInfo, error)
	DeleteWorkspaceNetwork(ctx context.Context, workspaceID string) error
	ListWorkspaceNetworks(ctx context.Context) ([]*NetworkInfo, error)
	ValidatePortMapping(portMap map[string]string) error
	CreatePortMapping(req *PortMappingRequest) (nat.PortSet, map[nat.Port][]nat.PortBinding, error)
	MonitorNetworkUsage(ctx context.Context, networkID string) (<-chan *NetworkStats, error)
	ApplyNetworkSecurity(ctx context.Context, networkID string, policy *NetworkSecurityPolicy) error
}

// SecurityResourceManagement 보안 리소스 관리 인터페이스
type SecurityResourceManagement interface {
	CreateResourceLimits() *securitypkg.ResourceLimits
	CreateCustomResourceLimits(req *securitypkg.ResourceLimitRequest) *securitypkg.ResourceLimits
	ToDockerResources(limits *securitypkg.ResourceLimits) container.Resources
	ValidateResourceLimits(limits *securitypkg.ResourceLimits) error
	ValidateResourceUsage(metrics *securitypkg.WorkspaceMetrics) []securitypkg.ResourceViolation
	CalculateOptimalLimits(workloadType securitypkg.WorkloadType, historyMetrics []securitypkg.WorkspaceMetrics) *securitypkg.ResourceLimits
	GetResourceLimitPreset(preset securitypkg.ResourcePreset) *securitypkg.ResourceLimits
}

// SecurityMonitoring 보안 모니터링 인터페이스
type SecurityMonitoring interface {
	StartMonitoring() <-chan securitypkg.SecurityAlert
	StopMonitoring()
	Subscribe(workspaceID string, handler securitypkg.AlertHandler)
	Unsubscribe(workspaceID string)
	ReportViolation(workspaceID string, violation securitypkg.ResourceViolation)
	ReportSecurityBreach(workspaceID string, breach securitypkg.SecurityBreach)
	GetSecurityDashboard() *securitypkg.SecurityDashboard
	GetWorkspaceViolations(workspaceID string) []securitypkg.ResourceViolation
	ClearViolations(workspaceID string)
}

// ImageManager 이미지 관리 인터페이스 (향후 구현 예정)
type ImageManager interface {
	PullImage(ctx context.Context, image string) error
	BuildImage(ctx context.Context, req BuildImageRequest) error
	ListImages(ctx context.Context) ([]ImageInfo, error)
	RemoveImage(ctx context.Context, imageID string, force bool) error
	InspectImage(ctx context.Context, imageID string) (*ImageInfo, error)
}


// BuildImageRequest 이미지 빌드 요청 (향후 구현)
type BuildImageRequest struct {
	ContextPath string            `json:"context_path"`
	Dockerfile  string            `json:"dockerfile"`
	Tag         string            `json:"tag"`
	BuildArgs   map[string]string `json:"build_args,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
}

// ImageInfo 이미지 정보 (향후 구현)
type ImageInfo struct {
	ID          string            `json:"id"`
	RepoTags    []string          `json:"repo_tags"`
	Created     time.Time         `json:"created"`
	Size        int64             `json:"size"`
	VirtualSize int64             `json:"virtual_size"`
	Labels      map[string]string `json:"labels"`
}

// LogEntry 로그 엔트리 (향후 구현)
type LogEntry struct {
	Timestamp   time.Time `json:"timestamp"`
	Stream      string    `json:"stream"` // stdout/stderr
	Message     string    `json:"message"`
	ContainerID string    `json:"container_id"`
}

// Exec 관련 타입들
type ExecConfig struct {
	Cmd          []string `json:"cmd"`
	AttachStdout bool     `json:"attach_stdout"`
	AttachStderr bool     `json:"attach_stderr"`
	AttachStdin  bool     `json:"attach_stdin"`
	Tty          bool     `json:"tty"`
	User         string   `json:"user,omitempty"`
	WorkingDir   string   `json:"working_dir,omitempty"`
}

type ExecStartConfig struct {
	Detach bool `json:"detach"`
	Tty    bool `json:"tty"`
}

type ExecResult struct {
	ExitCode int    `json:"exit_code"`
	Output   string `json:"output,omitempty"`
	Error    string `json:"error,omitempty"`
}

// Security 관련 타입 정의 (import cycle 방지용)
type NetworkInfo struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	WorkspaceID string            `json:"workspace_id"`
	Subnet      string            `json:"subnet"`
	Gateway     string            `json:"gateway"`
	Driver      string            `json:"driver"`
	Isolated    bool              `json:"isolated"`
	Internal    bool              `json:"internal"`
	Labels      map[string]string `json:"labels"`
	CreatedAt   time.Time         `json:"created_at"`
}

type NetworkStats struct {
	NetworkID       string    `json:"network_id"`
	RxBytes         int64     `json:"rx_bytes"`
	TxBytes         int64     `json:"tx_bytes"`
	RxPackets       int64     `json:"rx_packets"`
	TxPackets       int64     `json:"tx_packets"`
	ConnectionCount int       `json:"connection_count"`
	Timestamp       time.Time `json:"timestamp"`
}

type PortMappingRequest struct {
	PortMappings map[string]string `json:"port_mappings"` // "hostPort": "containerPort"
	HostIP       string            `json:"host_ip"`       // 바인드할 호스트 IP
	BindToHost   bool              `json:"bind_to_host"`  // 호스트에 바인드 여부
}

type NetworkSecurityPolicy struct {
	NetworkID      string         `json:"network_id"`
	MaxBandwidth   int64          `json:"max_bandwidth"`   // 최대 대역폭 (bytes/sec)
	MaxConnections int            `json:"max_connections"` // 최대 연결 수
	AllowRules     []FirewallRule `json:"allow_rules"`     // 허용 규칙
	BlockRules     []FirewallRule `json:"block_rules"`     // 차단 규칙
	EnableDPI      bool           `json:"enable_dpi"`      // Deep Packet Inspection 활성화
	LogTraffic     bool           `json:"log_traffic"`     // 트래픽 로깅 활성화
}

type FirewallRule struct {
	Protocol    string `json:"protocol"`    // tcp, udp, icmp
	Source      string `json:"source"`      // IP 주소 또는 CIDR
	Destination string `json:"destination"` // IP 주소 또는 CIDR
	Port        string `json:"port"`        // 포트 번호
	Action      string `json:"action"`      // allow, deny
	Priority    int    `json:"priority"`    // 규칙 우선순위
}

// 인터페이스 구현 검증을 위한 컴파일 타임 체크 (일시적으로 주석 처리)
/*
var (
	_ DockerClient         = (*Client)(nil)
	_ NetworkManagement    = (*NetworkManager)(nil)
	_ HealthMonitoring     = (*HealthChecker)(nil)
	_ StatsCollection      = (*StatsCollector)(nil)
	_ ContainerManagement  = (*ContainerManager)(nil)
	_ LifecycleManagement  = (*LifecycleManager)(nil)
	_ MountManagement      = (*MountManager)(nil)
	// DockerFactory와 DockerManager는 포인터 메서드를 사용하므로 별도 체크
)
*/