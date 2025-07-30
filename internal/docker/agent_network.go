package docker

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/docker/docker/api/types/network"
)

// AgentNetworkManager 에이전트별 격리된 네트워크 관리자
type AgentNetworkManager struct {
	client        *Client
	networkMgr    *NetworkManager
	mu            sync.RWMutex
	agentNetworks map[string]*AgentNetworkInfo
}

// AgentNetworkInfo 에이전트 네트워크 정보
type AgentNetworkInfo struct {
	AgentID     string    `json:"agent_id"`
	NetworkID   string    `json:"network_id"`
	NetworkName string    `json:"network_name"`
	Subnet      string    `json:"subnet"`
	Gateway     string    `json:"gateway"`
	IPAddress   string    `json:"ip_address"`
	CreatedAt   time.Time `json:"created_at"`
}

// NewAgentNetworkManager 새로운 에이전트 네트워크 매니저 생성
func NewAgentNetworkManager(client *Client) *AgentNetworkManager {
	return &AgentNetworkManager{
		client:        client,
		networkMgr:    NewNetworkManager(client),
		agentNetworks: make(map[string]*AgentNetworkInfo),
	}
}

// CreateAgentNetwork 에이전트별 격리된 네트워크 생성
func (anm *AgentNetworkManager) CreateAgentNetwork(ctx context.Context, agentID string) (*AgentNetworkInfo, error) {
	anm.mu.Lock()
	defer anm.mu.Unlock()

	// 이미 존재하는 네트워크 확인
	if existing, exists := anm.agentNetworks[agentID]; exists {
		return existing, nil
	}

	// 고유 서브넷 할당
	subnet, gateway, err := anm.allocateSubnet()
	if err != nil {
		return nil, fmt.Errorf("allocate subnet: %w", err)
	}

	networkName := fmt.Sprintf("agent-%s", agentID[:8])

	// 네트워크 생성 요청
	req := CreateNetworkRequest{
		Name:       networkName,
		Driver:     "bridge",
		Internal:   false, // 외부 통신 허용 (Claude API 접근용)
		Attachable: true,
		Subnet:     subnet,
		Gateway:    gateway,
		Labels: map[string]string{
			anm.client.labelKey("agent.id"):      agentID,
			anm.client.labelKey("agent.network"): "true",
			anm.client.labelKey("managed"):       "true",
			anm.client.labelKey("created"):       time.Now().Format(time.RFC3339),
		},
	}

	// 네트워크 생성
	networkInfo, err := anm.networkMgr.CreateNetwork(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("create network: %w", err)
	}

	// 에이전트 네트워크 정보 저장
	agentNetInfo := &AgentNetworkInfo{
		AgentID:     agentID,
		NetworkID:   networkInfo.ID,
		NetworkName: networkInfo.Name,
		Subnet:      subnet,
		Gateway:     gateway,
		CreatedAt:   time.Now(),
	}

	anm.agentNetworks[agentID] = agentNetInfo

	return agentNetInfo, nil
}

// GetAgentNetwork 에이전트 네트워크 정보 조회
func (anm *AgentNetworkManager) GetAgentNetwork(agentID string) (*AgentNetworkInfo, bool) {
	anm.mu.RLock()
	defer anm.mu.RUnlock()

	info, exists := anm.agentNetworks[agentID]
	return info, exists
}

// DeleteAgentNetwork 에이전트 네트워크 삭제
func (anm *AgentNetworkManager) DeleteAgentNetwork(ctx context.Context, agentID string) error {
	anm.mu.Lock()
	defer anm.mu.Unlock()

	info, exists := anm.agentNetworks[agentID]
	if !exists {
		return nil // 이미 삭제됨
	}

	// 네트워크에서 모든 컨테이너 분리 및 네트워크 삭제
	if err := anm.networkMgr.DeleteNetwork(ctx, info.NetworkID); err != nil {
		return fmt.Errorf("delete network: %w", err)
	}

	delete(anm.agentNetworks, agentID)
	return nil
}

// ConnectAgentContainer 에이전트 컨테이너를 격리 네트워크에 연결
func (anm *AgentNetworkManager) ConnectAgentContainer(ctx context.Context, agentID, containerID string) error {
	info, exists := anm.GetAgentNetwork(agentID)
	if !exists {
		return fmt.Errorf("agent network not found: %s", agentID)
	}

	// IP 주소 할당
	ipAddress, err := anm.allocateIPAddress(info.Subnet)
	if err != nil {
		return fmt.Errorf("allocate IP address: %w", err)
	}

	// 엔드포인트 설정
	endpointConfig := &network.EndpointSettings{
		IPAMConfig: &network.EndpointIPAMConfig{
			IPv4Address: ipAddress,
		},
		Aliases: []string{
			fmt.Sprintf("agent-%s", agentID[:8]),
		},
	}

	// 컨테이너를 네트워크에 연결
	if err := anm.networkMgr.ConnectContainer(ctx, info.NetworkID, containerID, endpointConfig); err != nil {
		return fmt.Errorf("connect container to network: %w", err)
	}

	// IP 주소 정보 업데이트
	anm.mu.Lock()
	info.IPAddress = ipAddress
	anm.mu.Unlock()

	return nil
}

// DisconnectAgentContainer 에이전트 컨테이너를 네트워크에서 분리
func (anm *AgentNetworkManager) DisconnectAgentContainer(ctx context.Context, agentID, containerID string) error {
	info, exists := anm.GetAgentNetwork(agentID)
	if !exists {
		return nil // 네트워크가 없으면 이미 분리된 상태
	}

	return anm.networkMgr.DisconnectContainer(ctx, info.NetworkID, containerID, false)
}

// ListAgentNetworks 모든 에이전트 네트워크 목록 조회
func (anm *AgentNetworkManager) ListAgentNetworks() map[string]*AgentNetworkInfo {
	anm.mu.RLock()
	defer anm.mu.RUnlock()

	result := make(map[string]*AgentNetworkInfo)
	for k, v := range anm.agentNetworks {
		result[k] = v
	}
	return result
}

// CleanupOrphanedNetworks 고아 네트워크 정리
func (anm *AgentNetworkManager) CleanupOrphanedNetworks(ctx context.Context) error {
	// Docker에서 에이전트 네트워크 목록 조회
	networks, err := anm.networkMgr.ListNetworks(ctx)
	if err != nil {
		return fmt.Errorf("list networks: %w", err)
	}

	for _, net := range networks {
		// 에이전트 네트워크인지 확인
		if agentNetwork, exists := net.Labels[anm.client.labelKey("agent.network")]; !exists || agentNetwork != "true" {
			continue
		}

		agentID := net.Labels[anm.client.labelKey("agent.id")]
		if agentID == "" {
			continue
		}

		// 내부 캐시에 없는 네트워크는 고아 네트워크
		if _, exists := anm.GetAgentNetwork(agentID); !exists {
			// 네트워크에 연결된 컨테이너 확인
			containers, err := anm.networkMgr.GetNetworkContainers(ctx, net.ID)
			if err != nil {
				continue
			}

			// 연결된 컨테이너가 없으면 삭제
			if len(containers) == 0 {
				if err := anm.networkMgr.DeleteNetwork(ctx, net.ID); err != nil {
					// 삭제 실패는 로그만 남기고 계속
					continue
				}
			}
		}
	}

	return nil
}

// allocateSubnet 새로운 서브넷 할당
func (anm *AgentNetworkManager) allocateSubnet() (subnet, gateway string, err error) {
	// 172.20.0.0/16 ~ 172.31.0.0/16 범위에서 /24 서브넷 할당
	_ = net.IPv4(172, 20, 0, 0) // baseIP 사용 예정
	
	// 현재 사용 중인 서브넷 확인
	usedSubnets := make(map[string]bool)
	for _, info := range anm.agentNetworks {
		usedSubnets[info.Subnet] = true
	}

	// 사용 가능한 서브넷 찾기
	for i := 0; i < 12; i++ { // 172.20.0.0 ~ 172.31.0.0
		for j := 0; j < 256; j++ { // .0.0 ~ .255.0
			testSubnet := fmt.Sprintf("172.%d.%d.0/24", 20+i, j)
			testGateway := fmt.Sprintf("172.%d.%d.1", 20+i, j)

			if !usedSubnets[testSubnet] {
				return testSubnet, testGateway, nil
			}
		}
	}

	return "", "", fmt.Errorf("no available subnet in range 172.20.0.0/16 - 172.31.0.0/16")
}

// allocateIPAddress 서브넷에서 사용 가능한 IP 주소 할당
func (anm *AgentNetworkManager) allocateIPAddress(subnet string) (string, error) {
	ip, ipnet, err := net.ParseCIDR(subnet)
	if err != nil {
		return "", fmt.Errorf("parse subnet: %w", err)
	}

	// 게이트웨이는 .1이므로 .2부터 시작
	ip = ip.To4()
	ip[3] = 2

	// .2 ~ .254 범위에서 사용 가능한 IP 찾기
	for ip[3] <= 254 {
		if ipnet.Contains(ip) {
			testIP := ip.String()
			
			// 다른 에이전트에서 사용 중인지 확인
			used := false
			for _, info := range anm.agentNetworks {
				if info.IPAddress == testIP {
					used = true
					break
				}
			}

			if !used {
				return testIP, nil
			}
		}
		ip[3]++
	}

	return "", fmt.Errorf("no available IP address in subnet %s", subnet)
}

// ValidateNetworkIsolation 네트워크 격리 상태 검증
func (anm *AgentNetworkManager) ValidateNetworkIsolation(ctx context.Context, agentID1, agentID2 string) (bool, error) {
	info1, exists1 := anm.GetAgentNetwork(agentID1)
	info2, exists2 := anm.GetAgentNetwork(agentID2)

	if !exists1 || !exists2 {
		return false, fmt.Errorf("one or both agent networks not found")
	}

	// 서로 다른 네트워크에 있는지 확인
	return info1.NetworkID != info2.NetworkID, nil
}

// GetNetworkStats 네트워크 통계 조회
func (anm *AgentNetworkManager) GetNetworkStats(ctx context.Context, agentID string) (map[string]interface{}, error) {
	info, exists := anm.GetAgentNetwork(agentID)
	if !exists {
		return nil, fmt.Errorf("agent network not found: %s", agentID)
	}

	// 네트워크에 연결된 컨테이너 수
	containers, err := anm.networkMgr.GetNetworkContainers(ctx, info.NetworkID)
	if err != nil {
		return nil, fmt.Errorf("get network containers: %w", err)
	}

	stats := map[string]interface{}{
		"agent_id":     agentID,
		"network_id":   info.NetworkID,
		"network_name": info.NetworkName,
		"subnet":       info.Subnet,
		"gateway":      info.Gateway,
		"ip_address":   info.IPAddress,
		"containers":   len(containers),
		"created_at":   info.CreatedAt,
		"uptime":       time.Since(info.CreatedAt).String(),
	}

	return stats, nil
}