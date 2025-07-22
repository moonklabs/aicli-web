package docker

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
)

// NetworkManager Docker 네트워크 관리를 담당합니다.
type NetworkManager struct {
	client *Client
}

// NewNetworkManager 새로운 네트워크 매니저를 생성합니다.
func NewNetworkManager(client *Client) *NetworkManager {
	return &NetworkManager{
		client: client,
	}
}

// NetworkInfo 네트워크 정보 구조체
type DockerNetworkInfo struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Driver     string            `json:"driver"`
	Scope      string            `json:"scope"`
	Internal   bool              `json:"internal"`
	Attachable bool              `json:"attachable"`
	Subnet     string            `json:"subnet"`
	Gateway    string            `json:"gateway"`
	Created    time.Time         `json:"created"`
	Labels     map[string]string `json:"labels"`
}

// CreateNetworkRequest 네트워크 생성 요청 구조체
type CreateNetworkRequest struct {
	Name       string            `json:"name"`
	Driver     string            `json:"driver"`
	Internal   bool              `json:"internal"`
	Attachable bool              `json:"attachable"`
	Subnet     string            `json:"subnet,omitempty"`
	Gateway    string            `json:"gateway,omitempty"`
	Labels     map[string]string `json:"labels,omitempty"`
}

// CreateNetwork 새로운 네트워크를 생성합니다.
func (nm *NetworkManager) CreateNetwork(ctx context.Context, req CreateNetworkRequest) (*DockerNetworkInfo, error) {
	// 기본값 설정
	if req.Driver == "" {
		req.Driver = "bridge"
	}

	// 네트워크 생성 설정
	createOptions := types.NetworkCreate{
		Driver:     req.Driver,
		Internal:   req.Internal,
		Attachable: req.Attachable,
		Labels: nm.client.MergeLabels(
			nm.client.WorkspaceLabels("", ""),
			req.Labels,
		),
	}

	// 서브넷 및 게이트웨이 설정
	if req.Subnet != "" || req.Gateway != "" {
		ipamConfig := network.IPAMConfig{}
		
		if req.Subnet != "" {
			ipamConfig.Subnet = req.Subnet
		}
		
		if req.Gateway != "" {
			ipamConfig.Gateway = req.Gateway
		}

		createOptions.IPAM = &network.IPAM{
			Config: []network.IPAMConfig{ipamConfig},
		}
	}

	// 네트워크 생성
	resp, err := nm.client.cli.NetworkCreate(ctx, req.Name, createOptions)
	if err != nil {
		return nil, fmt.Errorf("create network: %w", err)
	}

	// 생성된 네트워크 정보 조회
	return nm.GetNetwork(ctx, resp.ID)
}

// GetNetwork 네트워크 정보를 조회합니다.
func (nm *NetworkManager) GetNetwork(ctx context.Context, networkID string) (*DockerNetworkInfo, error) {
	inspect, err := nm.client.cli.NetworkInspect(ctx, networkID, types.NetworkInspectOptions{})
	if err != nil {
		return nil, fmt.Errorf("inspect network: %w", err)
	}

	info := &DockerNetworkInfo{
		ID:         inspect.ID,
		Name:       inspect.Name,
		Driver:     inspect.Driver,
		Scope:      inspect.Scope,
		Internal:   inspect.Internal,
		Attachable: inspect.Attachable,
		Labels:     inspect.Labels,
	}

	// 생성 시간 파싱
	if created, err := time.Parse(time.RFC3339, inspect.Created.Format(time.RFC3339)); err == nil {
		info.Created = created
	}

	// IPAM 정보 추출
	if inspect.IPAM != nil && len(inspect.IPAM.Config) > 0 {
		config := inspect.IPAM.Config[0]
		info.Subnet = config.Subnet
		info.Gateway = config.Gateway
	}

	return info, nil
}

// ListNetworks aicli에서 관리하는 네트워크 목록을 조회합니다.
func (nm *NetworkManager) ListNetworks(ctx context.Context) ([]DockerNetworkInfo, error) {
	// aicli 관리 네트워크 필터
	filterArgs := filters.NewArgs()
	filterArgs.Add("label", fmt.Sprintf("%s.managed=true", nm.client.labelPrefix))

	networks, err := nm.client.cli.NetworkList(ctx, types.NetworkListOptions{
		Filters: filterArgs,
	})
	if err != nil {
		return nil, fmt.Errorf("list networks: %w", err)
	}

	results := make([]DockerNetworkInfo, len(networks))
	for i, net := range networks {
		info, err := nm.GetNetwork(ctx, net.ID)
		if err != nil {
			// 에러가 발생하면 기본 정보만 포함
			results[i] = DockerNetworkInfo{
				ID:      net.ID,
				Name:    net.Name,
				Driver:  net.Driver,
				Scope:   net.Scope,
				Labels:  net.Labels,
				Created: net.Created,
			}
		} else {
			results[i] = *info
		}
	}

	return results, nil
}

// DeleteNetwork 네트워크를 삭제합니다.
func (nm *NetworkManager) DeleteNetwork(ctx context.Context, networkID string) error {
	return nm.client.cli.NetworkRemove(ctx, networkID)
}

// ConnectContainer 컨테이너를 네트워크에 연결합니다.
func (nm *NetworkManager) ConnectContainer(ctx context.Context, networkID, containerID string, config *network.EndpointSettings) error {
	endpointConfig := &network.EndpointSettings{}
	if config != nil {
		*endpointConfig = *config
	}

	return nm.client.cli.NetworkConnect(ctx, networkID, containerID, endpointConfig)
}

// DisconnectContainer 컨테이너를 네트워크에서 분리합니다.
func (nm *NetworkManager) DisconnectContainer(ctx context.Context, networkID, containerID string, force bool) error {
	return nm.client.cli.NetworkDisconnect(ctx, networkID, containerID, force)
}

// GetNetworkContainers 네트워크에 연결된 컨테이너 목록을 조회합니다.
func (nm *NetworkManager) GetNetworkContainers(ctx context.Context, networkID string) ([]string, error) {
	inspect, err := nm.client.cli.NetworkInspect(ctx, networkID, types.NetworkInspectOptions{})
	if err != nil {
		return nil, fmt.Errorf("inspect network: %w", err)
	}

	containers := make([]string, 0, len(inspect.Containers))
	for containerID := range inspect.Containers {
		containers = append(containers, containerID)
	}

	return containers, nil
}

// EnsureNetwork 네트워크가 존재하지 않으면 생성합니다.
func (nm *NetworkManager) EnsureNetwork(ctx context.Context, name string) (string, error) {
	// 기존 네트워크 확인
	networks, err := nm.client.cli.NetworkList(ctx, types.NetworkListOptions{})
	if err != nil {
		return "", fmt.Errorf("list networks: %w", err)
	}

	for _, net := range networks {
		if net.Name == name {
			return net.ID, nil
		}
	}

	// 네트워크 생성
	req := CreateNetworkRequest{
		Name:       name,
		Driver:     "bridge",
		Attachable: true,
		Internal:   false,
	}

	info, err := nm.CreateNetwork(ctx, req)
	if err != nil {
		return "", err
	}

	return info.ID, nil
}

// CleanupNetworks 사용하지 않는 네트워크를 정리합니다.
func (nm *NetworkManager) CleanupNetworks(ctx context.Context) error {
	// aicli 관리 네트워크 조회
	networks, err := nm.ListNetworks(ctx)
	if err != nil {
		return fmt.Errorf("list networks: %w", err)
	}

	for _, net := range networks {
		// 네트워크에 연결된 컨테이너 확인
		containers, err := nm.GetNetworkContainers(ctx, net.ID)
		if err != nil {
			continue // 에러 무시하고 다음으로
		}

		// 연결된 컨테이너가 없으면 삭제
		if len(containers) == 0 {
			if err := nm.DeleteNetwork(ctx, net.ID); err != nil {
				// 삭제 에러는 로그만 남기고 계속 진행
				continue
			}
		}
	}

	return nil
}

// ValidateNetworkConfig 네트워크 설정의 유효성을 검사합니다.
func ValidateNetworkConfig(req CreateNetworkRequest) error {
	if req.Name == "" {
		return fmt.Errorf("network name cannot be empty")
	}

	// 네트워크 이름 검증
	if err := ValidateContainerName(req.Name); err != nil {
		return fmt.Errorf("invalid network name: %w", err)
	}

	// 서브넷 검증
	if req.Subnet != "" {
		if _, _, err := net.ParseCIDR(req.Subnet); err != nil {
			return fmt.Errorf("invalid subnet format: %s", req.Subnet)
		}
	}

	// 게이트웨이 검증
	if req.Gateway != "" {
		if ip := net.ParseIP(req.Gateway); ip == nil {
			return fmt.Errorf("invalid gateway format: %s", req.Gateway)
		}

		// 서브넷이 지정된 경우 게이트웨이가 해당 범위에 있는지 확인
		if req.Subnet != "" {
			_, ipnet, err := net.ParseCIDR(req.Subnet)
			if err == nil {
				gatewayIP := net.ParseIP(req.Gateway)
				if !ipnet.Contains(gatewayIP) {
					return fmt.Errorf("gateway %s is not in subnet %s", req.Gateway, req.Subnet)
				}
			}
		}
	}

	return nil
}

// GetDefaultNetworkConfig 기본 네트워크 설정을 반환합니다.
func (nm *NetworkManager) GetDefaultNetworkConfig(name string) CreateNetworkRequest {
	return CreateNetworkRequest{
		Name:       name,
		Driver:     "bridge",
		Internal:   false,
		Attachable: true,
		Labels: map[string]string{
			nm.client.labelKey("managed"): "true",
			nm.client.labelKey("created"): time.Now().Format(time.RFC3339),
		},
	}
}