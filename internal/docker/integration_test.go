// +build integration

package docker

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 통합 테스트는 실제 Docker daemon이 필요합니다.
// go test -tags=integration 로 실행하세요.

func TestIntegration_CompleteWorkflow(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker daemon not available")
	}

	ctx := context.Background()
	
	// 테스트용 설정
	config := DefaultConfig()
	config.NetworkName = "integration-test-network"

	// 1. 매니저 생성
	manager, err := NewManager(config)
	require.NoError(t, err)
	defer func() {
		// 클린업
		if client := manager.Client(); client != nil && client.networkID != "" {
			client.cli.NetworkRemove(ctx, client.networkID)
		}
		manager.Shutdown()
	}()

	// 2. 헬스체크
	healthy, err := manager.GetFactory().IsHealthy(ctx)
	require.NoError(t, err)
	assert.True(t, healthy)

	// 3. 시스템 상태 조회
	systemStatus, err := manager.GetSystemStatus(ctx)
	require.NoError(t, err)
	assert.NotNil(t, systemStatus)
	assert.True(t, systemStatus.Healthy)

	// 4. 네트워크 관리 테스트
	networkManager := manager.Network()
	
	// 네트워크 목록 조회
	networks, err := networkManager.ListNetworks(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(networks), 1) // 최소한 기본 네트워크는 있어야 함

	// 기본 네트워크 정보 확인
	found := false
	for _, net := range networks {
		if net.Name == config.NetworkName {
			found = true
			assert.Equal(t, "bridge", net.Driver)
			assert.True(t, net.Attachable)
			assert.False(t, net.Internal)
			break
		}
	}
	assert.True(t, found, "Default network should be found")

	// 5. 통계 수집 테스트
	statsCollector := manager.Stats()
	
	// 시스템 통계 조회
	systemStats, err := statsCollector.GetSystemStats(ctx)
	require.NoError(t, err)
	assert.NotNil(t, systemStats)
	assert.Greater(t, systemStats.CPUCount, 0)
	assert.NotEmpty(t, systemStats.DockerVersion)

	// 집계 통계 조회
	aggregatedStats, err := statsCollector.GetAggregatedStats(ctx)
	require.NoError(t, err)
	assert.NotNil(t, aggregatedStats)

	// 6. 헬스체크 테스트
	healthChecker := manager.Health()
	
	// Daemon 체크
	err = healthChecker.CheckDaemon(ctx)
	assert.NoError(t, err)

	// 시스템 정보 조회
	info, err := healthChecker.GetSystemInfo(ctx)
	require.NoError(t, err)
	assert.NotNil(t, info)

	// 버전 정보 조회
	version, err := healthChecker.GetVersion(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, version.Version)

	// 7. 정리 작업
	err = manager.Cleanup(ctx)
	assert.NoError(t, err)
}

func TestIntegration_NetworkLifecycle(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker daemon not available")
	}

	ctx := context.Background()
	manager, err := NewManagerWithDefaults()
	require.NoError(t, err)
	defer manager.Shutdown()

	networkManager := manager.Network()

	// 1. 새 네트워크 생성
	createReq := CreateNetworkRequest{
		Name:       "test-lifecycle-network",
		Driver:     "bridge",
		Internal:   false,
		Attachable: true,
		Labels: map[string]string{
			"test": "integration",
		},
	}

	network, err := networkManager.CreateNetwork(ctx, createReq)
	require.NoError(t, err)
	require.NotNil(t, network)
	defer networkManager.DeleteNetwork(ctx, network.ID)

	assert.Equal(t, createReq.Name, network.Name)
	assert.Equal(t, createReq.Driver, network.Driver)
	assert.Equal(t, createReq.Internal, network.Internal)
	assert.Equal(t, createReq.Attachable, network.Attachable)

	// 2. 네트워크 조회
	retrievedNetwork, err := networkManager.GetNetwork(ctx, network.ID)
	require.NoError(t, err)
	assert.Equal(t, network.ID, retrievedNetwork.ID)
	assert.Equal(t, network.Name, retrievedNetwork.Name)

	// 3. 네트워크 목록에서 확인
	networks, err := networkManager.ListNetworks(ctx)
	require.NoError(t, err)
	
	found := false
	for _, net := range networks {
		if net.ID == network.ID {
			found = true
			break
		}
	}
	assert.True(t, found, "Created network should be in the list")

	// 4. 네트워크에 연결된 컨테이너 조회 (빈 목록이어야 함)
	containers, err := networkManager.GetNetworkContainers(ctx, network.ID)
	require.NoError(t, err)
	assert.Empty(t, containers)
}

func TestIntegration_ErrorHandling(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker daemon not available")
	}

	ctx := context.Background()
	manager, err := NewManagerWithDefaults()
	require.NoError(t, err)
	defer manager.Shutdown()

	networkManager := manager.Network()

	// 1. 존재하지 않는 네트워크 조회
	_, err = networkManager.GetNetwork(ctx, "non-existent-network-id")
	assert.Error(t, err)

	// 2. 존재하지 않는 네트워크 삭제
	err = networkManager.DeleteNetwork(ctx, "non-existent-network-id")
	assert.Error(t, err)

	// 3. 잘못된 네트워크 이름으로 생성 시도
	invalidReq := CreateNetworkRequest{
		Name:   "", // 빈 이름
		Driver: "bridge",
	}
	err = ValidateNetworkConfig(invalidReq)
	assert.Error(t, err)

	// 4. 잘못된 서브넷으로 네트워크 생성
	invalidSubnetReq := CreateNetworkRequest{
		Name:   "invalid-subnet-network",
		Driver: "bridge",
		Subnet: "invalid-subnet",
	}
	err = ValidateNetworkConfig(invalidSubnetReq)
	assert.Error(t, err)
}

func TestIntegration_HealthMonitoring(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker daemon not available")
	}

	ctx := context.Background()
	manager, err := NewManagerWithDefaults()
	require.NoError(t, err)
	defer manager.Shutdown()

	healthChecker := manager.Health()

	// 실시간 모니터링 테스트
	callbackCount := 0
	done := make(chan bool, 1)

	callback := func(err error) {
		callbackCount++
		if err != nil {
			t.Logf("Health check error: %v", err)
		}
		
		// 2번 호출되면 완료
		if callbackCount >= 2 {
			select {
			case done <- true:
			default:
			}
		}
	}

	// 짧은 간격으로 모니터링 시작
	testHealthChecker := NewHealthChecker(manager.Client(), 2*time.Second)
	testHealthChecker.StartMonitoring(ctx, callback)

	// 최대 10초 대기
	select {
	case <-done:
		assert.GreaterOrEqual(t, callbackCount, 2)
	case <-time.After(10 * time.Second):
		t.Log("Health monitoring test timed out")
		assert.GreaterOrEqual(t, callbackCount, 1, "At least one health check should have been performed")
	}
}

func TestIntegration_StatsCollection(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker daemon not available")
	}

	ctx := context.Background()
	manager, err := NewManagerWithDefaults()
	require.NoError(t, err)
	defer manager.Shutdown()

	statsCollector := manager.Stats()

	// 시스템 통계 수집
	systemStats, err := statsCollector.GetSystemStats(ctx)
	require.NoError(t, err)
	assert.NotNil(t, systemStats)
	
	// 기본적인 시스템 정보 확인
	assert.GreaterOrEqual(t, systemStats.ContainersTotal, 0)
	assert.GreaterOrEqual(t, systemStats.ImagesTotal, 0)
	assert.Greater(t, systemStats.CPUCount, 0)
	assert.Greater(t, systemStats.MemoryTotal, int64(0))
	assert.NotEmpty(t, systemStats.DockerVersion)

	// 집계 통계 수집
	aggregatedStats, err := statsCollector.GetAggregatedStats(ctx)
	require.NoError(t, err)
	assert.NotNil(t, aggregatedStats)
	
	// 관리되는 컨테이너가 없을 수 있으므로 0 이상인지만 확인
	assert.GreaterOrEqual(t, aggregatedStats.TotalContainers, 0)
	assert.GreaterOrEqual(t, aggregatedStats.RunningContainers, 0)
}

func TestIntegration_FactoryReinitialization(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker daemon not available")
	}

	ctx := context.Background()
	config := DefaultConfig()
	config.NetworkName = "reinit-test-network"

	factory, err := NewFactory(config)
	require.NoError(t, err)
	defer func() {
		// 클린업
		if client := factory.GetClient(); client != nil && client.networkID != "" {
			client.cli.NetworkRemove(ctx, client.networkID)
		}
		factory.Close()
	}()

	// 초기 상태 확인
	healthy, err := factory.IsHealthy(ctx)
	require.NoError(t, err)
	assert.True(t, healthy)

	originalNetworkID := factory.GetClient().GetNetworkID()
	assert.NotEmpty(t, originalNetworkID)

	// 재초기화
	err = factory.Reinitialize(ctx)
	require.NoError(t, err)

	// 재초기화 후 상태 확인
	healthy, err = factory.IsHealthy(ctx)
	require.NoError(t, err)
	assert.True(t, healthy)

	// 네트워크 ID는 동일해야 함 (기존 네트워크를 재사용)
	newNetworkID := factory.GetClient().GetNetworkID()
	assert.Equal(t, originalNetworkID, newNetworkID)
}

// 성능 벤치마크 테스트
func BenchmarkIntegration_SystemStatus(b *testing.B) {
	if !isDockerAvailable() {
		b.Skip("Docker daemon not available")
	}

	ctx := context.Background()
	manager, err := NewManagerWithDefaults()
	require.NoError(b, err)
	defer manager.Shutdown()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := manager.GetSystemStatus(ctx)
		require.NoError(b, err)
	}
}

func BenchmarkIntegration_HealthCheck(b *testing.B) {
	if !isDockerAvailable() {
		b.Skip("Docker daemon not available")
	}

	ctx := context.Background()
	manager, err := NewManagerWithDefaults()
	require.NoError(b, err)
	defer manager.Shutdown()

	healthChecker := manager.Health()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := healthChecker.CheckDaemon(ctx)
		require.NoError(b, err)
	}
}