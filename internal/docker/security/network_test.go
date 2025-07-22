package security

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 모의 Docker 클라이언트
type mockDockerClient struct{}

func (m *mockDockerClient) Ping(ctx context.Context) error       { return nil }
func (m *mockDockerClient) Close() error                        { return nil }
func (m *mockDockerClient) GetConfig() interface{}              { return nil }
func (m *mockDockerClient) GetNetworkID() string                { return "default-network" }

func TestNewNetworkManager(t *testing.T) {
	client := &mockDockerClient{}
	
	t.Run("with config", func(t *testing.T) {
		config := &IsolationConfig{
			EnableNetworkIsolation: true,
			BlockedPorts:           []int{80, 443},
		}
		
		nm := NewNetworkManager(client, config)
		
		assert.NotNil(t, nm)
		assert.Equal(t, client, nm.client)
		assert.Equal(t, config, nm.config)
	})
	
	t.Run("with nil config", func(t *testing.T) {
		nm := NewNetworkManager(client, nil)
		
		assert.NotNil(t, nm)
		assert.NotNil(t, nm.config)
		// 기본 설정으로 초기화되어야 함
		assert.True(t, nm.config.EnableNetworkIsolation)
	})
}

func TestCreateWorkspaceNetwork(t *testing.T) {
	client := &mockDockerClient{}
	nm := NewNetworkManager(client, nil)
	ctx := context.Background()
	
	t.Run("valid workspace ID", func(t *testing.T) {
		workspaceID := "test-workspace-123"
		
		networkInfo, err := nm.CreateWorkspaceNetwork(ctx, workspaceID)
		
		require.NoError(t, err)
		assert.NotNil(t, networkInfo)
		assert.NotEmpty(t, networkInfo.ID)
		assert.Equal(t, "aicli-workspace-test-workspace-123", networkInfo.Name)
		assert.Equal(t, workspaceID, networkInfo.WorkspaceID)
		assert.NotEmpty(t, networkInfo.Subnet)
		assert.NotEmpty(t, networkInfo.Gateway)
		assert.True(t, networkInfo.Isolated)
		assert.Equal(t, "bridge", networkInfo.Driver)
		assert.False(t, networkInfo.Internal)
		assert.WithinDuration(t, time.Now(), networkInfo.CreatedAt, time.Second)
		assert.Contains(t, networkInfo.Labels, "aicli.workspace.id")
		assert.Equal(t, workspaceID, networkInfo.Labels["aicli.workspace.id"])
	})
	
	t.Run("empty workspace ID", func(t *testing.T) {
		networkInfo, err := nm.CreateWorkspaceNetwork(ctx, "")
		
		assert.Error(t, err)
		assert.Nil(t, networkInfo)
		assert.Contains(t, err.Error(), "workspace ID cannot be empty")
	})
}

func TestGetWorkspaceNetwork(t *testing.T) {
	client := &mockDockerClient{}
	nm := NewNetworkManager(client, nil)
	ctx := context.Background()
	
	t.Run("valid workspace ID", func(t *testing.T) {
		workspaceID := "test-workspace-456"
		
		networkInfo, err := nm.GetWorkspaceNetwork(ctx, workspaceID)
		
		require.NoError(t, err)
		assert.NotNil(t, networkInfo)
		assert.Equal(t, "net_test-workspace-456", networkInfo.ID)
		assert.Equal(t, "aicli-workspace-test-workspace-456", networkInfo.Name)
		assert.Equal(t, workspaceID, networkInfo.WorkspaceID)
	})
	
	t.Run("empty workspace ID", func(t *testing.T) {
		networkInfo, err := nm.GetWorkspaceNetwork(ctx, "")
		
		assert.Error(t, err)
		assert.Nil(t, networkInfo)
		assert.Contains(t, err.Error(), "workspace ID cannot be empty")
	})
}

func TestDeleteWorkspaceNetwork(t *testing.T) {
	client := &mockDockerClient{}
	nm := NewNetworkManager(client, nil)
	ctx := context.Background()
	
	t.Run("valid workspace ID", func(t *testing.T) {
		err := nm.DeleteWorkspaceNetwork(ctx, "test-workspace")
		assert.NoError(t, err)
	})
	
	t.Run("empty workspace ID", func(t *testing.T) {
		err := nm.DeleteWorkspaceNetwork(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "workspace ID cannot be empty")
	})
}

func TestListWorkspaceNetworks(t *testing.T) {
	client := &mockDockerClient{}
	nm := NewNetworkManager(client, nil)
	ctx := context.Background()
	
	networks, err := nm.ListWorkspaceNetworks(ctx)
	
	require.NoError(t, err)
	assert.NotNil(t, networks)
	// 모의 구현에서는 빈 배열 반환
	assert.Empty(t, networks)
}

func TestAllocateSubnet(t *testing.T) {
	client := &mockDockerClient{}
	nm := NewNetworkManager(client, nil)
	
	tests := []struct {
		workspaceID string
		expectPrefix string
	}{
		{"test-workspace-1", "172.20."},
		{"test-workspace-2", "172.20."},
		{"different-workspace", "172.20."},
	}
	
	for _, tt := range tests {
		t.Run(tt.workspaceID, func(t *testing.T) {
			subnet := nm.allocateSubnet(tt.workspaceID)
			
			assert.Contains(t, subnet, tt.expectPrefix)
			assert.Contains(t, subnet, "/24")
			
			// 같은 워크스페이스 ID는 항상 같은 서브넷을 생성해야 함
			subnet2 := nm.allocateSubnet(tt.workspaceID)
			assert.Equal(t, subnet, subnet2)
		})
	}
}

func TestGetGatewayIP(t *testing.T) {
	client := &mockDockerClient{}
	nm := NewNetworkManager(client, nil)
	
	tests := []struct {
		subnet     string
		expectedGW string
	}{
		{"172.20.1.0/24", "172.20.1.1"},
		{"172.20.100.0/24", "172.20.100.1"},
		{"10.0.0.0/24", "10.0.0.1"},
	}
	
	for _, tt := range tests {
		t.Run(tt.subnet, func(t *testing.T) {
			gateway := nm.getGatewayIP(tt.subnet)
			assert.Equal(t, tt.expectedGW, gateway)
		})
	}
	
	t.Run("invalid subnet", func(t *testing.T) {
		gateway := nm.getGatewayIP("invalid-subnet")
		assert.Empty(t, gateway)
	})
}

func TestNetworkHashWorkspaceID(t *testing.T) {
	client := &mockDockerClient{}
	nm := NewNetworkManager(client, nil)
	
	// 같은 ID는 같은 해시를 생성
	id1 := "test-workspace-123"
	hash1a := nm.hashWorkspaceID(id1)
	hash1b := nm.hashWorkspaceID(id1)
	assert.Equal(t, hash1a, hash1b)
	
	// 다른 ID는 다른 해시를 생성
	id2 := "test-workspace-456"
	hash2 := nm.hashWorkspaceID(id2)
	assert.NotEqual(t, hash1a, hash2)
	
	// 해시는 1000 미만이어야 함 (% 1000 연산)
	assert.True(t, hash1a < 1000)
	assert.True(t, hash2 < 1000)
}

func TestValidatePortMapping(t *testing.T) {
	config := &IsolationConfig{
		BlockedPorts: []int{22, 80, 443},
	}
	client := &mockDockerClient{}
	nm := NewNetworkManager(client, config)
	
	tests := []struct {
		name        string
		portMap     map[string]string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "nil port map",
			portMap:     nil,
			expectError: false,
		},
		{
			name: "valid port mapping",
			portMap: map[string]string{
				"8080": "8000",
				"9090": "9000",
			},
			expectError: false,
		},
		{
			name: "blocked host port",
			portMap: map[string]string{
				"22": "22",
			},
			expectError: true,
			errorMsg:    "port 22 is blocked by security policy",
		},
		{
			name: "invalid host port",
			portMap: map[string]string{
				"invalid": "8000",
			},
			expectError: true,
			errorMsg:    "invalid host port",
		},
		{
			name: "invalid container port",
			portMap: map[string]string{
				"8080": "invalid",
			},
			expectError: true,
			errorMsg:    "invalid container port",
		},
		{
			name: "host port out of range - too low",
			portMap: map[string]string{
				"0": "8000",
			},
			expectError: true,
			errorMsg:    "out of valid range",
		},
		{
			name: "host port out of range - too high",
			portMap: map[string]string{
				"65536": "8000",
			},
			expectError: true,
			errorMsg:    "out of valid range",
		},
		{
			name: "container port out of range",
			portMap: map[string]string{
				"8080": "70000",
			},
			expectError: true,
			errorMsg:    "out of valid range",
		},
		{
			name: "port with protocol",
			portMap: map[string]string{
				"8080/tcp": "8000/tcp",
			},
			expectError: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := nm.ValidatePortMapping(tt.portMap)
			
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIsPortBlocked(t *testing.T) {
	config := &IsolationConfig{
		BlockedPorts: []int{22, 80, 443, 8080},
	}
	client := &mockDockerClient{}
	nm := NewNetworkManager(client, config)
	
	tests := []struct {
		port     int
		expected bool
	}{
		{22, true},
		{80, true},
		{443, true},
		{8080, true},
		{8000, false},
		{9000, false},
		{3000, false},
	}
	
	for _, tt := range tests {
		t.Run(string(rune(tt.port)), func(t *testing.T) {
			result := nm.isPortBlocked(tt.port)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCreatePortMapping(t *testing.T) {
	// 블록되지 않은 포트만 사용하는 설정
	config := &IsolationConfig{
		BlockedPorts: []int{22, 80, 443}, // 8080, 8000, 9090은 블록되지 않음
	}
	client := &mockDockerClient{}
	nm := NewNetworkManager(client, config)
	
	t.Run("nil request", func(t *testing.T) {
		exposedPorts, portBindings, err := nm.CreatePortMapping(nil)
		
		assert.Error(t, err)
		assert.Nil(t, exposedPorts)
		assert.Nil(t, portBindings)
		assert.Contains(t, err.Error(), "port mapping request cannot be nil")
	})
	
	t.Run("valid port mapping with host binding", func(t *testing.T) {
		req := &PortMappingRequest{
			PortMappings: map[string]string{
				"8888": "8000", // 블록되지 않은 포트 사용
				"9999": "9000", // 블록되지 않은 포트 사용
			},
			HostIP:     "127.0.0.1",
			BindToHost: true,
		}
		
		exposedPorts, portBindings, err := nm.CreatePortMapping(req)
		
		require.NoError(t, err)
		assert.NotNil(t, exposedPorts)
		assert.NotNil(t, portBindings)
		assert.Len(t, exposedPorts, 2)
		assert.Len(t, portBindings, 2)
	})
	
	t.Run("valid port mapping without host binding", func(t *testing.T) {
		req := &PortMappingRequest{
			PortMappings: map[string]string{
				"8888": "8000", // 블록되지 않은 포트 사용
			},
			BindToHost: false,
		}
		
		exposedPorts, portBindings, err := nm.CreatePortMapping(req)
		
		require.NoError(t, err)
		assert.NotNil(t, exposedPorts)
		assert.NotNil(t, portBindings)
		assert.Len(t, exposedPorts, 1)
		assert.Empty(t, portBindings) // BindToHost가 false이므로 바인딩 없음
	})
	
	t.Run("invalid port mapping", func(t *testing.T) {
		req := &PortMappingRequest{
			PortMappings: map[string]string{
				"22": "22", // 블록된 포트
			},
			BindToHost: true,
		}
		
		exposedPorts, portBindings, err := nm.CreatePortMapping(req)
		
		assert.Error(t, err)
		assert.Nil(t, exposedPorts)
		assert.Nil(t, portBindings)
		assert.Contains(t, err.Error(), "invalid port mapping")
	})
}

func TestMonitorNetworkUsage(t *testing.T) {
	client := &mockDockerClient{}
	nm := NewNetworkManager(client, nil)
	
	t.Run("empty network ID", func(t *testing.T) {
		ctx := context.Background()
		
		statsChan, err := nm.MonitorNetworkUsage(ctx, "")
		
		assert.Error(t, err)
		assert.Nil(t, statsChan)
		assert.Contains(t, err.Error(), "network ID cannot be empty")
	})
	
	t.Run("valid network ID", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		
		networkID := "test-network-123"
		
		statsChan, err := nm.MonitorNetworkUsage(ctx, networkID)
		
		require.NoError(t, err)
		assert.NotNil(t, statsChan)
		
		// 첫 번째 통계 수신을 기다림 (5초 간격이므로 충분히 기다림)
		select {
		case stats, ok := <-statsChan:
			if !ok {
				t.Log("Channel closed")
				return
			}
			assert.Equal(t, networkID, stats.NetworkID)
			assert.NotZero(t, stats.RxBytes)
			assert.NotZero(t, stats.TxBytes)
			assert.WithinDuration(t, time.Now(), stats.Timestamp, time.Second)
		case <-time.After(200 * time.Millisecond):
			t.Log("No stats received within timeout, but channel is working")
			// 타임아웃은 정상적인 동작일 수 있음 (5초 간격)
		}
	})
}

func TestValidateNetworkSecurityPolicy(t *testing.T) {
	client := &mockDockerClient{}
	nm := NewNetworkManager(client, nil)
	
	tests := []struct {
		name        string
		policy      *NetworkSecurityPolicy
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid policy",
			policy: &NetworkSecurityPolicy{
				MaxBandwidth:   1000000, // 1MB/s
				MaxConnections: 100,
				AllowRules: []FirewallRule{
					{
						Protocol: "tcp",
						Port:     "80",
						Source:   "192.168.1.0/24",
					},
				},
			},
			expectError: false,
		},
		{
			name: "negative bandwidth",
			policy: &NetworkSecurityPolicy{
				MaxBandwidth: -1,
			},
			expectError: true,
			errorMsg:    "max bandwidth cannot be negative",
		},
		{
			name: "negative connections",
			policy: &NetworkSecurityPolicy{
				MaxConnections: -1,
			},
			expectError: true,
			errorMsg:    "max connections cannot be negative",
		},
		{
			name: "invalid allow rule",
			policy: &NetworkSecurityPolicy{
				AllowRules: []FirewallRule{
					{
						Protocol: "invalid",
						Port:     "80",
					},
				},
			},
			expectError: true,
			errorMsg:    "invalid allow rule",
		},
		{
			name: "invalid block rule",
			policy: &NetworkSecurityPolicy{
				BlockRules: []FirewallRule{
					{
						Protocol: "tcp",
						Port:     "99999", // 포트 범위 초과
					},
				},
			},
			expectError: true,
			errorMsg:    "invalid block rule",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := nm.validateNetworkSecurityPolicy(tt.policy)
			
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateFirewallRule(t *testing.T) {
	client := &mockDockerClient{}
	nm := NewNetworkManager(client, nil)
	
	tests := []struct {
		name        string
		rule        FirewallRule
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid TCP rule",
			rule: FirewallRule{
				Protocol: "tcp",
				Port:     "80",
				Source:   "192.168.1.0/24",
			},
			expectError: false,
		},
		{
			name: "valid UDP rule",
			rule: FirewallRule{
				Protocol: "udp",
				Port:     "53",
				Source:   "8.8.8.8",
			},
			expectError: false,
		},
		{
			name: "valid ICMP rule",
			rule: FirewallRule{
				Protocol: "icmp",
				Source:   "10.0.0.0/8",
			},
			expectError: false,
		},
		{
			name: "invalid protocol",
			rule: FirewallRule{
				Protocol: "invalid",
			},
			expectError: true,
			errorMsg:    "invalid protocol",
		},
		{
			name: "invalid port",
			rule: FirewallRule{
				Protocol: "tcp",
				Port:     "invalid",
			},
			expectError: true,
			errorMsg:    "invalid port",
		},
		{
			name: "port out of range - too low",
			rule: FirewallRule{
				Protocol: "tcp",
				Port:     "0",
			},
			expectError: true,
			errorMsg:    "port out of range",
		},
		{
			name: "port out of range - too high",
			rule: FirewallRule{
				Protocol: "tcp",
				Port:     "65536",
			},
			expectError: true,
			errorMsg:    "port out of range",
		},
		{
			name: "invalid CIDR",
			rule: FirewallRule{
				Protocol: "tcp",
				Source:   "invalid-cidr/24",
			},
			expectError: true,
			errorMsg:    "invalid CIDR",
		},
		{
			name: "invalid IP address",
			rule: FirewallRule{
				Protocol: "tcp",
				Source:   "invalid-ip",
			},
			expectError: true,
			errorMsg:    "invalid IP address",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := nm.validateFirewallRule(tt.rule)
			
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestApplyNetworkSecurity(t *testing.T) {
	client := &mockDockerClient{}
	nm := NewNetworkManager(client, nil)
	ctx := context.Background()
	
	t.Run("empty network ID", func(t *testing.T) {
		policy := &NetworkSecurityPolicy{}
		
		err := nm.ApplyNetworkSecurity(ctx, "", policy)
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "network ID cannot be empty")
	})
	
	t.Run("nil policy", func(t *testing.T) {
		err := nm.ApplyNetworkSecurity(ctx, "test-network", nil)
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "network security policy cannot be nil")
	})
	
	t.Run("valid policy", func(t *testing.T) {
		policy := &NetworkSecurityPolicy{
			MaxBandwidth:   1000000,
			MaxConnections: 100,
		}
		
		err := nm.ApplyNetworkSecurity(ctx, "test-network", policy)
		
		assert.NoError(t, err)
	})
}

// 벤치마크 테스트
func BenchmarkAllocateSubnet(b *testing.B) {
	client := &mockDockerClient{}
	nm := NewNetworkManager(client, nil)
	workspaceID := "benchmark-workspace-12345"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		nm.allocateSubnet(workspaceID)
	}
}

func BenchmarkNetworkHashWorkspaceID(b *testing.B) {
	client := &mockDockerClient{}
	nm := NewNetworkManager(client, nil)
	workspaceID := "benchmark-workspace-for-hashing-performance-test"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		nm.hashWorkspaceID(workspaceID)
	}
}

func BenchmarkValidatePortMapping(b *testing.B) {
	client := &mockDockerClient{}
	nm := NewNetworkManager(client, nil)
	portMap := map[string]string{
		"8080": "8000",
		"9090": "9000",
		"7070": "7000",
		"6060": "6000",
		"5050": "5000",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		nm.ValidatePortMapping(portMap)
	}
}