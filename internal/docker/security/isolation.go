package security

import (
	"fmt"
	"hash/fnv"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/aicli/aicli-web/internal/models"
)

// IsolationManager 격리 설정 관리자
type IsolationManager struct {
	config *IsolationConfig
}

// IsolationConfig 격리 설정
type IsolationConfig struct {
	// 네트워크 격리
	EnableNetworkIsolation bool     `yaml:"enable_network_isolation" json:"enable_network_isolation"`
	AllowedNetworks       []string `yaml:"allowed_networks" json:"allowed_networks"`
	BlockedPorts          []int    `yaml:"blocked_ports" json:"blocked_ports"`
	
	// 리소스 제한
	DefaultCPULimit    float64 `yaml:"default_cpu_limit" json:"default_cpu_limit"`
	DefaultMemoryLimit int64   `yaml:"default_memory_limit" json:"default_memory_limit"`
	DefaultDiskLimit   int64   `yaml:"default_disk_limit" json:"default_disk_limit"`
	
	// 보안 설정
	EnableSeccomp    bool `yaml:"enable_seccomp" json:"enable_seccomp"`
	EnableAppArmor   bool `yaml:"enable_apparmor" json:"enable_apparmor"`
	DisablePrivileged bool `yaml:"disable_privileged" json:"disable_privileged"`
	
	// 파일 시스템 보안
	ReadOnlyRootFS  bool `yaml:"read_only_root_fs" json:"read_only_root_fs"`
	NoNewPrivileges bool `yaml:"no_new_privileges" json:"no_new_privileges"`
	
	// 로깅 및 모니터링
	EnableAuditLog     bool `yaml:"enable_audit_log" json:"enable_audit_log"`
	MonitorSystemCalls bool `yaml:"monitor_system_calls" json:"monitor_system_calls"`
}

// NewIsolationManager 새로운 격리 관리자 생성
func NewIsolationManager() *IsolationManager {
	return &IsolationManager{
		config: DefaultIsolationConfig(),
	}
}

// NewIsolationManagerWithConfig 설정과 함께 격리 관리자 생성
func NewIsolationManagerWithConfig(config *IsolationConfig) *IsolationManager {
	return &IsolationManager{
		config: config,
	}
}

// DefaultIsolationConfig 기본 격리 설정 생성
func DefaultIsolationConfig() *IsolationConfig {
	return &IsolationConfig{
		// 네트워크 기본값
		EnableNetworkIsolation: true,
		AllowedNetworks:        []string{"aicli-network"},
		BlockedPorts:           []int{22, 80, 443, 3000, 8000, 8080},
		
		// 리소스 기본 제한
		DefaultCPULimit:    1.0,                   // 1 CPU
		DefaultMemoryLimit: 512 * 1024 * 1024,    // 512MB
		DefaultDiskLimit:   1024 * 1024 * 1024,   // 1GB
		
		// 보안 기본 설정
		EnableSeccomp:     true,
		EnableAppArmor:    true,
		DisablePrivileged: true,
		ReadOnlyRootFS:    false, // 워크스페이스는 쓰기 가능
		NoNewPrivileges:   true,
		
		// 모니터링 기본 설정
		EnableAuditLog:     true,
		MonitorSystemCalls: false, // 성능 상 기본 비활성화
	}
}

// CreateWorkspaceIsolation 워크스페이스용 격리 설정 생성
func (im *IsolationManager) CreateWorkspaceIsolation(workspace *models.Workspace) (*WorkspaceIsolation, error) {
	if workspace == nil {
		return nil, fmt.Errorf("workspace cannot be nil")
	}

	isolation := &WorkspaceIsolation{
		WorkspaceID:      workspace.ID,
		NetworkMode:      "custom",
		NetworkName:      fmt.Sprintf("aicli-workspace-%s", workspace.ID),
		IsolationLevel:   IsolationLevelStandard,
		ResourceLimits:   im.createResourceLimits(),
		SecurityOptions:  im.createSecurityOptions(),
		MonitoringConfig: im.createMonitoringConfig(),
		CreatedAt:        time.Now(),
	}
	
	return isolation, nil
}

// createResourceLimits 리소스 제한 설정 생성
func (im *IsolationManager) createResourceLimits() *ResourceLimits {
	return &ResourceLimits{
		CPUShares:      1024, // 기본 가중치
		CPUQuota:       int64(im.config.DefaultCPULimit * 100000),
		CPUPeriod:      100000,
		Memory:         im.config.DefaultMemoryLimit,
		MemorySwap:     im.config.DefaultMemoryLimit, // Swap 비활성화
		PidsLimit:      100,  // 최대 100개 프로세스
		IOMaxBandwidth: "100m", // 100MB/s
		IOMaxIOps:      1000,   // 1000 IOPS
	}
}

// createSecurityOptions 보안 옵션 생성
func (im *IsolationManager) createSecurityOptions() *SecurityOptions {
	opts := &SecurityOptions{
		NoNewPrivileges: im.config.NoNewPrivileges,
		ReadOnlyRootFS:  im.config.ReadOnlyRootFS,
		Capabilities: &CapabilityConfig{
			Drop: []string{
				"ALL", // 모든 capability 제거 후 필요한 것만 추가
			},
			Add: []string{
				"CHOWN",        // 파일 소유권 변경
				"DAC_OVERRIDE", // 파일 권한 우회 (필요한 경우)
				"SETGID",       // 그룹 ID 설정
				"SETUID",       // 사용자 ID 설정
			},
		},
	}

	// Seccomp 프로파일 설정
	if im.config.EnableSeccomp {
		opts.SeccompProfile = "default"
	}

	// AppArmor 프로파일 설정
	if im.config.EnableAppArmor {
		opts.AppArmorProfile = "docker-default"
	}

	return opts
}

// createMonitoringConfig 모니터링 설정 생성
func (im *IsolationManager) createMonitoringConfig() *MonitoringConfig {
	return &MonitoringConfig{
		EnableResourceMonitoring: true,
		EnableNetworkMonitoring:  true,
		EnableFileSystemAudit:    im.config.EnableAuditLog,
		LogLevel:                 "info",
		AlertThresholds: &AlertThresholds{
			CPUThreshold:     85.0,  // 85% CPU 사용률
			MemoryThreshold:  90.0,  // 90% 메모리 사용률
			NetworkThreshold: 100*1024*1024, // 100MB/s 네트워크 I/O
			DiskThreshold:    50*1024*1024,  // 50MB/s 디스크 I/O
		},
	}
}

// ApplyToContainer 컨테이너 구성에 격리 설정 적용
func (im *IsolationManager) ApplyToContainer(isolation *WorkspaceIsolation, config *container.Config, hostConfig *container.HostConfig) error {
	if isolation == nil {
		return fmt.Errorf("isolation config cannot be nil")
	}
	if config == nil {
		return fmt.Errorf("container config cannot be nil")
	}
	if hostConfig == nil {
		return fmt.Errorf("host config cannot be nil")
	}

	// 리소스 제한 적용
	if err := im.applyResourceLimits(isolation.ResourceLimits, hostConfig); err != nil {
		return fmt.Errorf("failed to apply resource limits: %w", err)
	}

	// 보안 옵션 적용
	if err := im.applySecurityOptions(isolation.SecurityOptions, config, hostConfig); err != nil {
		return fmt.Errorf("failed to apply security options: %w", err)
	}

	// 네트워크 모드 설정
	if isolation.NetworkMode == "custom" {
		hostConfig.NetworkMode = container.NetworkMode(isolation.NetworkName)
	}

	return nil
}

// applyResourceLimits 리소스 제한 적용
func (im *IsolationManager) applyResourceLimits(limits *ResourceLimits, hostConfig *container.HostConfig) error {
	if limits == nil {
		return fmt.Errorf("resource limits cannot be nil")
	}

	// CPU 제한
	hostConfig.Resources.CPUShares = limits.CPUShares
	hostConfig.Resources.CPUQuota = limits.CPUQuota
	hostConfig.Resources.CPUPeriod = limits.CPUPeriod

	// 메모리 제한
	hostConfig.Resources.Memory = limits.Memory
	hostConfig.Resources.MemorySwap = limits.MemorySwap

	// 프로세스 수 제한
	if limits.PidsLimit > 0 {
		hostConfig.Resources.PidsLimit = &limits.PidsLimit
	}

	// I/O 가중치
	hostConfig.Resources.BlkioWeight = 500

	return nil
}

// applySecurityOptions 보안 옵션 적용
func (im *IsolationManager) applySecurityOptions(opts *SecurityOptions, config *container.Config, hostConfig *container.HostConfig) error {
	if opts == nil {
		return fmt.Errorf("security options cannot be nil")
	}

	// 새 권한 금지
	if opts.NoNewPrivileges {
		hostConfig.SecurityOpt = append(hostConfig.SecurityOpt, "no-new-privileges:true")
	}

	// 루트 파일시스템 읽기 전용
	if opts.ReadOnlyRootFS {
		hostConfig.ReadonlyRootfs = true
	}

	// Capability 설정
	if opts.Capabilities != nil {
		if len(opts.Capabilities.Drop) > 0 {
			hostConfig.CapDrop = opts.Capabilities.Drop
		}
		if len(opts.Capabilities.Add) > 0 {
			hostConfig.CapAdd = opts.Capabilities.Add
		}
	}

	// Seccomp 프로파일
	if opts.SeccompProfile != "" {
		hostConfig.SecurityOpt = append(hostConfig.SecurityOpt, fmt.Sprintf("seccomp=%s", opts.SeccompProfile))
	}

	// AppArmor 프로파일
	if opts.AppArmorProfile != "" {
		hostConfig.SecurityOpt = append(hostConfig.SecurityOpt, fmt.Sprintf("apparmor=%s", opts.AppArmorProfile))
	}

	// 권한 모드 비활성화
	if im.config.DisablePrivileged {
		hostConfig.Privileged = false
	}

	return nil
}

// GetConfig 설정 반환
func (im *IsolationManager) GetConfig() *IsolationConfig {
	return im.config
}

// UpdateConfig 설정 업데이트
func (im *IsolationManager) UpdateConfig(config *IsolationConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}
	im.config = config
	return nil
}

// ValidateIsolation 격리 설정 검증
func (im *IsolationManager) ValidateIsolation(isolation *WorkspaceIsolation) error {
	if isolation == nil {
		return fmt.Errorf("isolation config cannot be nil")
	}

	// 워크스페이스 ID 검증
	if isolation.WorkspaceID == "" {
		return fmt.Errorf("workspace ID cannot be empty")
	}

	// 네트워크 모드 검증
	if isolation.NetworkMode == "" {
		return fmt.Errorf("network mode cannot be empty")
	}

	// 리소스 제한 검증
	if isolation.ResourceLimits == nil {
		return fmt.Errorf("resource limits cannot be nil")
	}

	if isolation.ResourceLimits.Memory <= 0 {
		return fmt.Errorf("memory limit must be positive")
	}

	// 보안 옵션 검증
	if isolation.SecurityOptions == nil {
		return fmt.Errorf("security options cannot be nil")
	}

	return nil
}

// hashWorkspaceID 워크스페이스 ID 해싱
func (im *IsolationManager) hashWorkspaceID(workspaceID string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(workspaceID))
	return h.Sum32()
}

// WorkspaceIsolation 워크스페이스 격리 설정
type WorkspaceIsolation struct {
	WorkspaceID      string            `json:"workspace_id"`
	NetworkMode      string            `json:"network_mode"`
	NetworkName      string            `json:"network_name"`
	IsolationLevel   IsolationLevel    `json:"isolation_level"`
	ResourceLimits   *ResourceLimits   `json:"resource_limits"`
	SecurityOptions  *SecurityOptions  `json:"security_options"`
	MonitoringConfig *MonitoringConfig `json:"monitoring_config"`
	CreatedAt        time.Time         `json:"created_at"`
}

// IsolationLevel 격리 수준
type IsolationLevel string

const (
	IsolationLevelBasic    IsolationLevel = "basic"    // 기본 격리
	IsolationLevelStandard IsolationLevel = "standard" // 표준 격리
	IsolationLevelStrict   IsolationLevel = "strict"   // 엄격한 격리
)

// ResourceLimits 리소스 제한
type ResourceLimits struct {
	CPUShares      int64  `json:"cpu_shares"`        // CPU 가중치
	CPUQuota       int64  `json:"cpu_quota"`         // CPU 할당량
	CPUPeriod      int64  `json:"cpu_period"`        // CPU 주기
	Memory         int64  `json:"memory"`            // 메모리 제한
	MemorySwap     int64  `json:"memory_swap"`       // Swap 제한
	PidsLimit      int64  `json:"pids_limit"`        // 프로세스 수 제한
	IOMaxBandwidth string `json:"io_max_bandwidth"`  // I/O 대역폭 제한
	IOMaxIOps      int64  `json:"io_max_iops"`       // I/O 속도 제한
}

// SecurityOptions 보안 옵션
type SecurityOptions struct {
	SeccompProfile  string            `json:"seccomp_profile"`   // Seccomp 프로파일
	AppArmorProfile string            `json:"apparmor_profile"`  // AppArmor 프로파일
	SELinuxLabels   []string          `json:"selinux_labels"`    // SELinux 레이블
	Capabilities    *CapabilityConfig `json:"capabilities"`      // Capability 설정
	NoNewPrivileges bool              `json:"no_new_privileges"` // 새 권한 금지
	ReadOnlyRootFS  bool              `json:"read_only_root_fs"` // 루트 FS 읽기전용
}

// CapabilityConfig Capability 설정
type CapabilityConfig struct {
	Drop []string `json:"drop"` // 제거할 capability
	Add  []string `json:"add"`  // 추가할 capability
}

// MonitoringConfig 모니터링 설정
type MonitoringConfig struct {
	EnableResourceMonitoring bool             `json:"enable_resource_monitoring"`
	EnableNetworkMonitoring  bool             `json:"enable_network_monitoring"`
	EnableFileSystemAudit    bool             `json:"enable_filesystem_audit"`
	LogLevel                 string           `json:"log_level"`
	AlertThresholds          *AlertThresholds `json:"alert_thresholds"`
}

// AlertThresholds 알림 임계값
type AlertThresholds struct {
	CPUThreshold     float64 `json:"cpu_threshold"`      // CPU 사용률 경고 임계값
	MemoryThreshold  float64 `json:"memory_threshold"`   // 메모리 사용률 경고 임계값
	NetworkThreshold int64   `json:"network_threshold"`  // 네트워크 I/O 경고 임계값
	DiskThreshold    int64   `json:"disk_threshold"`     // 디스크 I/O 경고 임계값
}