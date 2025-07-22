package security

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
)

// DockerClient 인터페이스 (import cycle 방지용)
type DockerClient interface {
	Ping(context.Context) error
	Close() error
}

// NetworkManager 네트워크 격리 관리자
type NetworkManager struct {
	client DockerClient
	config *IsolationConfig
}

// NewNetworkManager 새로운 네트워크 관리자 생성
func NewNetworkManager(client DockerClient, config *IsolationConfig) *NetworkManager {
	if config == nil {
		config = DefaultIsolationConfig()
	}
	return &NetworkManager{
		client: client,
		config: config,
	}
}

// CreateWorkspaceNetwork 워크스페이스 전용 네트워크 생성
func (nm *NetworkManager) CreateWorkspaceNetwork(ctx context.Context, workspaceID string) (*NetworkInfo, error) {
	if workspaceID == "" {
		return nil, fmt.Errorf("workspace ID cannot be empty")
	}

	networkName := fmt.Sprintf("aicli-workspace-%s", workspaceID)
	
	// CIDR 대역 할당 (172.20.x.0/24 대역 사용)
	subnet := nm.allocateSubnet(workspaceID)
	gateway := nm.getGatewayIP(subnet)
	
	networkConfig := types.NetworkCreate{
		Driver:     "bridge",
		Internal:   false, // 외부 인터넷 접근 허용
		Attachable: false, // 다른 컨테이너 연결 금지
		IPAM: &network.IPAM{
			Driver: "default",
			Config: []network.IPAMConfig{
				{
					Subnet:  subnet,
					Gateway: gateway,
				},
			},
		},
		Options: map[string]string{
			"com.docker.network.bridge.name":           networkName,
			"com.docker.network.driver.mtu":           "1500",
			"com.docker.network.bridge.enable_icc":    "false", // 컨테이너 간 통신 차단
			"com.docker.network.bridge.enable_ip_masquerade": "true",
		},
		Labels: map[string]string{
			"aicli.workspace.id": workspaceID,
			"aicli.managed":      "true",
			"aicli.isolation":    "workspace",
			"aicli.created_at":   time.Now().Format(time.RFC3339),
		},
	}
	
	// Docker 클라이언트를 통해 네트워크 생성
	// 실제 구현에서는 client.NetworkCreate를 호출
	// 현재는 모의 구현
	networkID := fmt.Sprintf("net_%s_%d", workspaceID, time.Now().Unix())
	
	return &NetworkInfo{
		ID:          networkID,
		Name:        networkName,
		WorkspaceID: workspaceID,
		Subnet:      subnet,
		Gateway:     gateway,
		Isolated:    true,
		CreatedAt:   time.Now(),
		Driver:      "bridge",
		Internal:    false,
		Labels:      networkConfig.Labels,
	}, nil
}

// GetWorkspaceNetwork 워크스페이스 네트워크 조회
func (nm *NetworkManager) GetWorkspaceNetwork(ctx context.Context, workspaceID string) (*NetworkInfo, error) {
	if workspaceID == "" {
		return nil, fmt.Errorf("workspace ID cannot be empty")
	}

	networkName := fmt.Sprintf("aicli-workspace-%s", workspaceID)
	
	// 실제 구현에서는 Docker API를 통해 네트워크 조회
	// 현재는 모의 구현
	return &NetworkInfo{
		ID:          fmt.Sprintf("net_%s", workspaceID),
		Name:        networkName,
		WorkspaceID: workspaceID,
		Subnet:      nm.allocateSubnet(workspaceID),
		Gateway:     nm.getGatewayIP(nm.allocateSubnet(workspaceID)),
		Isolated:    true,
		Driver:      "bridge",
		Internal:    false,
		CreatedAt:   time.Now().Add(-1 * time.Hour), // 가정: 1시간 전 생성
	}, nil
}

// DeleteWorkspaceNetwork 워크스페이스 네트워크 삭제
func (nm *NetworkManager) DeleteWorkspaceNetwork(ctx context.Context, workspaceID string) error {
	if workspaceID == "" {
		return fmt.Errorf("workspace ID cannot be empty")
	}

	// 실제 구현에서는 Docker API를 통해 네트워크 삭제
	// 현재는 모의 구현
	return nil
}

// ListWorkspaceNetworks 모든 워크스페이스 네트워크 목록 조회
func (nm *NetworkManager) ListWorkspaceNetworks(ctx context.Context) ([]*NetworkInfo, error) {
	// 실제 구현에서는 Docker API를 통해 aicli 레이블이 있는 네트워크들 조회
	// 현재는 모의 구현
	return []*NetworkInfo{}, nil
}

// allocateSubnet 워크스페이스 ID에 대해 고유한 서브넷 할당
func (nm *NetworkManager) allocateSubnet(workspaceID string) string {
	// 워크스페이스 ID에서 해시를 생성하여 고유한 서브넷 할당
	// 예: 172.20.1.0/24, 172.20.2.0/24, ...
	hash := nm.hashWorkspaceID(workspaceID)
	octet := (hash % 254) + 1 // 1-254 범위
	return fmt.Sprintf("172.20.%d.0/24", octet)
}

// getGatewayIP 서브넷에서 게이트웨이 IP 추출
func (nm *NetworkManager) getGatewayIP(subnet string) string {
	_, network, err := net.ParseCIDR(subnet)
	if err != nil {
		return "" // 에러 처리
	}
	
	// 게이트웨이는 네트워크의 첫 번째 IP (.1)
	ip := network.IP
	ip[len(ip)-1] = 1
	return ip.String()
}

// hashWorkspaceID 워크스페이스 ID 해싱
func (nm *NetworkManager) hashWorkspaceID(workspaceID string) uint32 {
	hash := uint32(0)
	for _, char := range workspaceID {
		hash = (hash*31 + uint32(char)) % 1000
	}
	return hash
}

// ValidatePortMapping 포트 매핑 제한 검사
func (nm *NetworkManager) ValidatePortMapping(portMap map[string]string) error {
	if portMap == nil {
		return nil
	}

	for hostPort, containerPort := range portMap {
		// 호스트 포트 검증
		hostPortNum, err := strconv.Atoi(strings.Split(hostPort, "/")[0])
		if err != nil {
			return fmt.Errorf("invalid host port %s: %w", hostPort, err)
		}
		
		// 차단된 포트 검사
		if nm.isPortBlocked(hostPortNum) {
			return fmt.Errorf("port %s is blocked by security policy", hostPort)
		}
		
		// 포트 범위 검증 (1-65535)
		if hostPortNum < 1 || hostPortNum > 65535 {
			return fmt.Errorf("host port %d is out of valid range (1-65535)", hostPortNum)
		}
		
		// 컨테이너 포트 검증
		containerPortNum, err := strconv.Atoi(strings.Split(containerPort, "/")[0])
		if err != nil {
			return fmt.Errorf("invalid container port %s: %w", containerPort, err)
		}
		
		if containerPortNum < 1 || containerPortNum > 65535 {
			return fmt.Errorf("container port %d is out of valid range (1-65535)", containerPortNum)
		}
	}
	
	return nil
}

// isPortBlocked 포트가 차단되었는지 확인
func (nm *NetworkManager) isPortBlocked(port int) bool {
	for _, blockedPort := range nm.config.BlockedPorts {
		if port == blockedPort {
			return true
		}
	}
	return false
}

// CreatePortMapping 안전한 포트 매핑 생성
func (nm *NetworkManager) CreatePortMapping(req *PortMappingRequest) (nat.PortSet, map[nat.Port][]nat.PortBinding, error) {
	if req == nil {
		return nil, nil, fmt.Errorf("port mapping request cannot be nil")
	}

	// 포트 매핑 검증
	if err := nm.ValidatePortMapping(req.PortMappings); err != nil {
		return nil, nil, fmt.Errorf("invalid port mapping: %w", err)
	}

	exposedPorts := make(nat.PortSet)
	portBindings := make(map[nat.Port][]nat.PortBinding)

	for hostPortStr, containerPortStr := range req.PortMappings {
		// 컨테이너 포트 파싱
		containerPort, err := nat.NewPort("tcp", containerPortStr)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid container port %s: %w", containerPortStr, err)
		}

		// 포트 노출 설정
		exposedPorts[containerPort] = struct{}{}

		// 호스트 포트 바인딩 설정
		if req.BindToHost {
			portBindings[containerPort] = []nat.PortBinding{
				{
					HostIP:   req.HostIP,
					HostPort: hostPortStr,
				},
			}
		}
	}

	return exposedPorts, portBindings, nil
}

// MonitorNetworkUsage 네트워크 사용량 모니터링
func (nm *NetworkManager) MonitorNetworkUsage(ctx context.Context, networkID string) (<-chan *NetworkStats, error) {
	if networkID == "" {
		return nil, fmt.Errorf("network ID cannot be empty")
	}

	statsChan := make(chan *NetworkStats, 10)
	
	go func() {
		defer close(statsChan)
		
		// 첫 번째 통계를 즉시 전송
		stats := &NetworkStats{
			NetworkID:       networkID,
			RxBytes:         int64(time.Now().Unix() * 1024), // 모의 데이터
			TxBytes:         int64(time.Now().Unix() * 512),  // 모의 데이터
			RxPackets:       int64(time.Now().Unix()),
			TxPackets:       int64(time.Now().Unix() / 2),
			ConnectionCount: 5, // 모의 연결 수
			Timestamp:       time.Now(),
		}
		
		select {
		case statsChan <- stats:
		case <-ctx.Done():
			return
		}
		
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// 실제 구현에서는 Docker API를 통해 네트워크 통계 수집
				// 현재는 모의 데이터 생성
				stats := &NetworkStats{
					NetworkID:       networkID,
					RxBytes:         int64(time.Now().Unix() * 1024), // 모의 데이터
					TxBytes:         int64(time.Now().Unix() * 512),  // 모의 데이터
					RxPackets:       int64(time.Now().Unix()),
					TxPackets:       int64(time.Now().Unix() / 2),
					ConnectionCount: 5, // 모의 연결 수
					Timestamp:       time.Now(),
				}
				
				select {
				case statsChan <- stats:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	
	return statsChan, nil
}

// ApplyNetworkSecurity 네트워크 보안 정책 적용
func (nm *NetworkManager) ApplyNetworkSecurity(ctx context.Context, networkID string, policy *NetworkSecurityPolicy) error {
	if networkID == "" {
		return fmt.Errorf("network ID cannot be empty")
	}
	if policy == nil {
		return fmt.Errorf("network security policy cannot be nil")
	}

	// 실제 구현에서는 iptables 규칙 또는 Docker 네트워크 설정을 통해 보안 정책 적용
	// 현재는 정책 검증만 수행
	return nm.validateNetworkSecurityPolicy(policy)
}

// validateNetworkSecurityPolicy 네트워크 보안 정책 검증
func (nm *NetworkManager) validateNetworkSecurityPolicy(policy *NetworkSecurityPolicy) error {
	// 트래픽 제한 검증
	if policy.MaxBandwidth < 0 {
		return fmt.Errorf("max bandwidth cannot be negative")
	}
	
	if policy.MaxConnections < 0 {
		return fmt.Errorf("max connections cannot be negative")
	}
	
	// 허용/차단 규칙 검증
	for _, rule := range policy.AllowRules {
		if err := nm.validateFirewallRule(rule); err != nil {
			return fmt.Errorf("invalid allow rule: %w", err)
		}
	}
	
	for _, rule := range policy.BlockRules {
		if err := nm.validateFirewallRule(rule); err != nil {
			return fmt.Errorf("invalid block rule: %w", err)
		}
	}
	
	return nil
}

// validateFirewallRule 방화벽 규칙 검증
func (nm *NetworkManager) validateFirewallRule(rule FirewallRule) error {
	// 프로토콜 검증
	if rule.Protocol != "tcp" && rule.Protocol != "udp" && rule.Protocol != "icmp" {
		return fmt.Errorf("invalid protocol: %s", rule.Protocol)
	}
	
	// 포트 범위 검증
	if rule.Port != "" {
		port, err := strconv.Atoi(rule.Port)
		if err != nil {
			return fmt.Errorf("invalid port: %s", rule.Port)
		}
		if port < 1 || port > 65535 {
			return fmt.Errorf("port out of range: %d", port)
		}
	}
	
	// IP 주소/CIDR 검증
	if rule.Source != "" {
		if strings.Contains(rule.Source, "/") {
			_, _, err := net.ParseCIDR(rule.Source)
			if err != nil {
				return fmt.Errorf("invalid CIDR: %s", rule.Source)
			}
		} else {
			if net.ParseIP(rule.Source) == nil {
				return fmt.Errorf("invalid IP address: %s", rule.Source)
			}
		}
	}
	
	return nil
}

// NetworkInfo 네트워크 정보
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

// NetworkStats 네트워크 통계
type NetworkStats struct {
	NetworkID       string    `json:"network_id"`
	RxBytes         int64     `json:"rx_bytes"`
	TxBytes         int64     `json:"tx_bytes"`
	RxPackets       int64     `json:"rx_packets"`
	TxPackets       int64     `json:"tx_packets"`
	ConnectionCount int       `json:"connection_count"`
	Timestamp       time.Time `json:"timestamp"`
}

// PortMappingRequest 포트 매핑 요청
type PortMappingRequest struct {
	PortMappings map[string]string `json:"port_mappings"` // "hostPort": "containerPort"
	HostIP       string            `json:"host_ip"`       // 바인드할 호스트 IP (기본값: "0.0.0.0")
	BindToHost   bool              `json:"bind_to_host"`  // 호스트에 바인드 여부
}

// NetworkSecurityPolicy 네트워크 보안 정책
type NetworkSecurityPolicy struct {
	NetworkID      string         `json:"network_id"`
	MaxBandwidth   int64          `json:"max_bandwidth"`   // 최대 대역폭 (bytes/sec)
	MaxConnections int            `json:"max_connections"` // 최대 연결 수
	AllowRules     []FirewallRule `json:"allow_rules"`     // 허용 규칙
	BlockRules     []FirewallRule `json:"block_rules"`     // 차단 규칙
	EnableDPI      bool           `json:"enable_dpi"`      // Deep Packet Inspection 활성화
	LogTraffic     bool           `json:"log_traffic"`     // 트래픽 로깅 활성화
}

// FirewallRule 방화벽 규칙
type FirewallRule struct {
	Protocol    string `json:"protocol"`    // tcp, udp, icmp
	Source      string `json:"source"`      // IP 주소 또는 CIDR
	Destination string `json:"destination"` // IP 주소 또는 CIDR
	Port        string `json:"port"`        // 포트 번호
	Action      string `json:"action"`      // allow, deny
	Priority    int    `json:"priority"`    // 규칙 우선순위
}