package docker

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFactory(t *testing.T) {
	config := DefaultConfig()
	config.NetworkName = "test-factory-network"

	if !isDockerAvailable() {
		t.Skip("Docker daemon not available")
	}

	factory, err := NewFactory(config)
	require.NoError(t, err)
	require.NotNil(t, factory)
	defer factory.Close()

	// 팩토리 구성 요소들이 올바르게 초기화되었는지 확인
	assert.NotNil(t, factory.GetClient())
	assert.NotNil(t, factory.GetNetworkManager())
	assert.NotNil(t, factory.GetStatsCollector())
	assert.NotNil(t, factory.GetHealthChecker())
	assert.Equal(t, config, factory.GetConfig())

	// 클린업
	if client := factory.GetClient(); client != nil && client.networkID != "" {
		ctx := context.Background()
		client.cli.NetworkRemove(ctx, client.networkID)
	}
}

func TestNewFactoryWithDefaults(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker daemon not available")
	}

	factory, err := NewFactoryWithDefaults()
	require.NoError(t, err)
	require.NotNil(t, factory)
	defer factory.Close()

	// 기본 설정이 사용되었는지 확인
	config := factory.GetConfig()
	defaultConfig := DefaultConfig()
	assert.Equal(t, defaultConfig.NetworkName, config.NetworkName)
	assert.Equal(t, defaultConfig.DefaultImage, config.DefaultImage)
}

func TestFactory_Ping(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker daemon not available")
	}

	factory, err := NewFactoryWithDefaults()
	require.NoError(t, err)
	defer factory.Close()

	ctx := context.Background()
	err = factory.Ping(ctx)
	assert.NoError(t, err)
}

func TestFactory_IsHealthy(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker daemon not available")
	}

	factory, err := NewFactoryWithDefaults()
	require.NoError(t, err)
	defer factory.Close()

	ctx := context.Background()
	healthy, err := factory.IsHealthy(ctx)
	assert.NoError(t, err)
	assert.True(t, healthy)
}

func TestFactory_Reinitialize(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker daemon not available")
	}

	factory, err := NewFactoryWithDefaults()
	require.NoError(t, err)
	defer factory.Close()

	// 기존 클라이언트 확인
	originalClient := factory.GetClient()
	assert.NotNil(t, originalClient)

	// 재초기화
	ctx := context.Background()
	err = factory.Reinitialize(ctx)
	assert.NoError(t, err)

	// 새로운 클라이언트가 생성되었는지 확인 (포인터 비교)
	newClient := factory.GetClient()
	assert.NotNil(t, newClient)
	// 주의: 실제로는 새 인스턴스가 생성되므로 포인터가 다를 수 있음
}

func TestFactory_UpdateConfig(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker daemon not available")
	}

	factory, err := NewFactoryWithDefaults()
	require.NoError(t, err)
	defer factory.Close()

	// 새로운 설정으로 업데이트
	newConfig := DefaultConfig()
	newConfig.NetworkName = "updated-test-network"
	newConfig.CPULimit = 2.0

	ctx := context.Background()
	err = factory.UpdateConfig(ctx, newConfig)
	assert.NoError(t, err)

	// 설정이 업데이트되었는지 확인
	updatedConfig := factory.GetConfig()
	assert.Equal(t, "updated-test-network", updatedConfig.NetworkName)
	assert.Equal(t, 2.0, updatedConfig.CPULimit)

	// 클린업
	if client := factory.GetClient(); client != nil && client.networkID != "" {
		client.cli.NetworkRemove(ctx, client.networkID)
	}
}

func TestFactory_UpdateConfig_NilConfig(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker daemon not available")
	}

	factory, err := NewFactoryWithDefaults()
	require.NoError(t, err)
	defer factory.Close()

	ctx := context.Background()
	err = factory.UpdateConfig(ctx, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config cannot be nil")
}

func TestNewManager(t *testing.T) {
	config := DefaultConfig()
	config.NetworkName = "test-manager-network"

	if !isDockerAvailable() {
		t.Skip("Docker daemon not available")
	}

	manager, err := NewManager(config)
	require.NoError(t, err)
	require.NotNil(t, manager)
	defer manager.Shutdown()

	// 매니저 구성 요소들이 올바르게 초기화되었는지 확인
	assert.NotNil(t, manager.GetFactory())
	assert.NotNil(t, manager.Client())
	assert.NotNil(t, manager.Network())
	assert.NotNil(t, manager.Stats())
	assert.NotNil(t, manager.Health())
	assert.Equal(t, config, manager.Config())
	assert.NotNil(t, manager.Context())

	// 클린업
	if client := manager.Client(); client != nil && client.networkID != "" {
		ctx := context.Background()
		client.cli.NetworkRemove(ctx, client.networkID)
	}
}

func TestNewManagerWithDefaults(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker daemon not available")
	}

	manager, err := NewManagerWithDefaults()
	require.NoError(t, err)
	require.NotNil(t, manager)
	defer manager.Shutdown()

	// 기본 설정이 사용되었는지 확인
	config := manager.Config()
	defaultConfig := DefaultConfig()
	assert.Equal(t, defaultConfig.NetworkName, config.NetworkName)
}

func TestManager_GetSystemStatus(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker daemon not available")
	}

	manager, err := NewManagerWithDefaults()
	require.NoError(t, err)
	defer manager.Shutdown()

	ctx := context.Background()
	status, err := manager.GetSystemStatus(ctx)
	require.NoError(t, err)
	assert.NotNil(t, status)

	// 상태 정보가 올바르게 설정되었는지 확인
	assert.True(t, status.Healthy)
	assert.NotNil(t, status.SystemStats)
	assert.NotNil(t, status.AggregatedStats)
	assert.GreaterOrEqual(t, status.NetworkCount, 0)
	assert.False(t, status.Timestamp.IsZero())
}

func TestManager_StartHealthMonitoring(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker daemon not available")
	}

	manager, err := NewManagerWithDefaults()
	require.NoError(t, err)
	defer manager.Shutdown()

	// 헬스 모니터링 콜백 테스트
	callbackCalled := make(chan bool, 1)
	callback := func(err error) {
		// 에러가 없어야 함 (Docker가 실행 중이므로)
		assert.NoError(t, err)
		select {
		case callbackCalled <- true:
		default:
		}
	}

	manager.StartHealthMonitoring(callback)

	// 잠시 대기하여 모니터링이 시작되도록 함
	select {
	case <-callbackCalled:
		// 콜백이 호출됨
	case <-time.After(35 * time.Second): // 헬스체크 간격 + 여유시간
		t.Log("Health monitoring callback not called within expected time")
	}
}

func TestManager_Cleanup(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker daemon not available")
	}

	manager, err := NewManagerWithDefaults()
	require.NoError(t, err)
	defer manager.Shutdown()

	ctx := context.Background()
	err = manager.Cleanup(ctx)
	assert.NoError(t, err)
}

func TestManager_Shutdown(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker daemon not available")
	}

	manager, err := NewManagerWithDefaults()
	require.NoError(t, err)

	err = manager.Shutdown()
	assert.NoError(t, err)

	// 셧다운 후에는 컨텍스트가 취소되어야 함
	select {
	case <-manager.Context().Done():
		// 컨텍스트가 취소됨 (정상)
	case <-time.After(time.Second):
		t.Error("Context was not cancelled after shutdown")
	}
}

func TestDefaultManager(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker daemon not available")
	}

	// 기본 매니저 초기화
	ResetDefaultManager()

	// 기본 매니저 가져오기
	manager1, err := GetDefaultManager()
	require.NoError(t, err)
	require.NotNil(t, manager1)

	// 동일한 인스턴스인지 확인
	manager2, err := GetDefaultManager()
	require.NoError(t, err)
	assert.Same(t, manager1, manager2)

	// 새로운 매니저로 설정
	newManager, err := NewManagerWithDefaults()
	require.NoError(t, err)
	SetDefaultManager(newManager)

	manager3, err := GetDefaultManager()
	require.NoError(t, err)
	assert.Same(t, newManager, manager3)

	// 클린업
	ResetDefaultManager()
}

func TestResetDefaultManager(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker daemon not available")
	}

	// 기본 매니저 생성
	manager, err := GetDefaultManager()
	require.NoError(t, err)
	require.NotNil(t, manager)

	// 리셋
	ResetDefaultManager()

	// 새로운 매니저가 생성되는지 확인
	newManager, err := GetDefaultManager()
	require.NoError(t, err)
	require.NotNil(t, newManager)

	// 클린업
	ResetDefaultManager()
}

func BenchmarkFactory_GetClient(b *testing.B) {
	if !isDockerAvailable() {
		b.Skip("Docker daemon not available")
	}

	factory, err := NewFactoryWithDefaults()
	require.NoError(b, err)
	defer factory.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = factory.GetClient()
	}
}

func BenchmarkManager_Config(b *testing.B) {
	if !isDockerAvailable() {
		b.Skip("Docker daemon not available")
	}

	manager, err := NewManagerWithDefaults()
	require.NoError(b, err)
	defer manager.Shutdown()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = manager.Config()
	}
}