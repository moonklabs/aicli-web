package docker

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Factory Docker 클라이언트와 관련 매니저들을 생성하고 관리합니다.
type Factory struct {
	config            *Config
	client            *Client
	networkManager    *NetworkManager
	statsCollector    *StatsCollector
	healthChecker     *HealthChecker
	containerManager  *ContainerManager
	lifecycleManager  *LifecycleManager
	mountManager      *MountManager
	metricsCollector  *MetricsCollector
	mu                sync.RWMutex
}

// NewFactory 새로운 Docker 팩토리를 생성합니다.
func NewFactory(config *Config) (*Factory, error) {
	client, err := NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("create docker client: %w", err)
	}

	statsCollector := NewStatsCollector(client)
	
	factory := &Factory{
		config:           config,
		client:           client,
		networkManager:   NewNetworkManager(client),
		statsCollector:   statsCollector,
		healthChecker:    NewHealthChecker(client, 30*time.Second),
		containerManager: NewContainerManager(client),
		lifecycleManager: NewLifecycleManager(client),
		mountManager:     NewMountManager(),
		metricsCollector: NewMetricsCollector(statsCollector, nil), // Manager는 나중에 설정
	}

	return factory, nil
}

// NewFactoryWithDefaults 기본 설정으로 팩토리를 생성합니다.
func NewFactoryWithDefaults() (*Factory, error) {
	return NewFactory(DefaultConfig())
}

// GetClient Docker 클라이언트를 반환합니다.
func (f *Factory) GetClient() *Client {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.client
}

// GetNetworkManager 네트워크 매니저를 반환합니다.
func (f *Factory) GetNetworkManager() *NetworkManager {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.networkManager
}

// GetStatsCollector 통계 수집기를 반환합니다.
func (f *Factory) GetStatsCollector() *StatsCollector {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.statsCollector
}

// GetHealthChecker 헬스체커를 반환합니다.
func (f *Factory) GetHealthChecker() *HealthChecker {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.healthChecker
}

// GetContainerManager 컨테이너 매니저를 반환합니다.
func (f *Factory) GetContainerManager() *ContainerManager {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.containerManager
}

// GetLifecycleManager 생명주기 매니저를 반환합니다.
func (f *Factory) GetLifecycleManager() *LifecycleManager {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.lifecycleManager
}

// GetMountManager 마운트 매니저를 반환합니다.
func (f *Factory) GetMountManager() *MountManager {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.mountManager
}

// GetMetricsCollector 메트릭 수집기를 반환합니다.
func (f *Factory) GetMetricsCollector() *MetricsCollector {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.metricsCollector
}

// GetConfig 설정을 반환합니다.
func (f *Factory) GetConfig() *Config {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.config
}

// Ping Docker daemon 연결을 확인합니다.
func (f *Factory) Ping(ctx context.Context) error {
	f.mu.RLock()
	client := f.client
	f.mu.RUnlock()
	
	return client.Ping(ctx)
}

// IsHealthy Factory와 모든 구성 요소가 정상 상태인지 확인합니다.
func (f *Factory) IsHealthy(ctx context.Context) (bool, error) {
	// Docker daemon 연결 확인
	if err := f.Ping(ctx); err != nil {
		return false, fmt.Errorf("docker daemon not available: %w", err)
	}

	// 네트워크 확인
	if f.client.networkID == "" {
		return false, fmt.Errorf("docker network not initialized")
	}

	// 네트워크 존재 여부 확인
	_, err := f.networkManager.GetNetwork(ctx, f.client.networkID)
	if err != nil {
		return false, fmt.Errorf("docker network not accessible: %w", err)
	}

	return true, nil
}

// Reinitialize 팩토리를 다시 초기화합니다.
func (f *Factory) Reinitialize(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// 기존 클라이언트 정리
	if f.client != nil {
		f.client.Close()
	}

	// 새로운 클라이언트 생성
	client, err := NewClient(f.config)
	if err != nil {
		return fmt.Errorf("reinitialize docker client: %w", err)
	}

	// 매니저들 재생성
	f.client = client
	f.networkManager = NewNetworkManager(client)
	f.statsCollector = NewStatsCollector(client)
	f.healthChecker = NewHealthChecker(client, 30*time.Second)
	f.containerManager = NewContainerManager(client)
	
	// 기존 생명주기 매니저 정리
	if f.lifecycleManager != nil {
		f.lifecycleManager.Close()
	}
	f.lifecycleManager = NewLifecycleManager(client)

	return nil
}

// UpdateConfig 설정을 업데이트하고 클라이언트를 재초기화합니다.
func (f *Factory) UpdateConfig(ctx context.Context, config *Config) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	f.mu.Lock()
	f.config = config
	f.mu.Unlock()

	return f.Reinitialize(ctx)
}

// Close 팩토리와 모든 리소스를 정리합니다.
func (f *Factory) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// 생명주기 매니저 정리
	if f.lifecycleManager != nil {
		f.lifecycleManager.Close()
	}

	// 클라이언트 정리
	if f.client != nil {
		return f.client.Close()
	}

	return nil
}

// Manager Docker 리소스를 통합 관리하는 매니저
type Manager struct {
	factory  *Factory
	ctx      context.Context
	cancel   context.CancelFunc
	mu       sync.RWMutex
}

// NewManager 새로운 Docker 매니저를 생성합니다.
func NewManager(config *Config) (*Manager, error) {
	factory, err := NewFactory(config)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	manager := &Manager{
		factory: factory,
		ctx:     ctx,
		cancel:  cancel,
	}

	return manager, nil
}

// NewManagerWithDefaults 기본 설정으로 매니저를 생성합니다.
func NewManagerWithDefaults() (*Manager, error) {
	return NewManager(DefaultConfig())
}

// GetFactory 팩토리를 반환합니다.
func (m *Manager) GetFactory() *Factory {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.factory
}

// Client Docker 클라이언트를 반환합니다.
func (m *Manager) Client() *Client {
	return m.factory.GetClient()
}

// Network 네트워크 매니저를 반환합니다.
func (m *Manager) Network() *NetworkManager {
	return m.factory.GetNetworkManager()
}

// Stats 통계 수집기를 반환합니다.
func (m *Manager) Stats() *StatsCollector {
	return m.factory.GetStatsCollector()
}

// Health 헬스체커를 반환합니다.
func (m *Manager) Health() *HealthChecker {
	return m.factory.GetHealthChecker()
}

// Container 컨테이너 매니저를 반환합니다.
func (m *Manager) Container() *ContainerManager {
	return m.factory.GetContainerManager()
}

// Lifecycle 생명주기 매니저를 반환합니다.
func (m *Manager) Lifecycle() *LifecycleManager {
	return m.factory.GetLifecycleManager()
}

// Mount 마운트 매니저를 반환합니다.
func (m *Manager) Mount() *MountManager {
	return m.factory.GetMountManager()
}

// Metrics 메트릭 수집기를 반환합니다.
func (m *Manager) Metrics() *MetricsCollector {
	return m.factory.GetMetricsCollector()
}

// Config 설정을 반환합니다.
func (m *Manager) Config() *Config {
	return m.factory.GetConfig()
}

// Context 매니저의 컨텍스트를 반환합니다.
func (m *Manager) Context() context.Context {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.ctx
}

// StartHealthMonitoring 헬스 모니터링을 시작합니다.
func (m *Manager) StartHealthMonitoring(callback func(error)) {
	m.factory.GetHealthChecker().StartMonitoring(m.ctx, callback)
}

// GetSystemStatus 시스템 전체 상태를 조회합니다.
func (m *Manager) GetSystemStatus(ctx context.Context) (*SystemStatus, error) {
	healthy, err := m.factory.IsHealthy(ctx)
	if err != nil {
		return nil, err
	}

	systemStats, err := m.factory.GetStatsCollector().GetSystemStats(ctx)
	if err != nil {
		return nil, err
	}

	aggregatedStats, err := m.factory.GetStatsCollector().GetAggregatedStats(ctx)
	if err != nil {
		return nil, err
	}

	networks, err := m.factory.GetNetworkManager().ListNetworks(ctx)
	if err != nil {
		return nil, err
	}

	return &SystemStatus{
		Healthy:         healthy,
		SystemStats:     systemStats,
		AggregatedStats: aggregatedStats,
		NetworkCount:    len(networks),
		Timestamp:       time.Now(),
	}, nil
}

// SystemStatus 시스템 전체 상태 정보
type SystemStatus struct {
	Healthy         bool             `json:"healthy"`
	SystemStats     *SystemStats     `json:"system_stats"`
	AggregatedStats *AggregatedStats `json:"aggregated_stats"`
	NetworkCount    int              `json:"network_count"`
	Timestamp       time.Time        `json:"timestamp"`
}

// Cleanup 사용하지 않는 리소스를 정리합니다.
func (m *Manager) Cleanup(ctx context.Context) error {
	// 네트워크 정리
	if err := m.factory.GetNetworkManager().CleanupNetworks(ctx); err != nil {
		return fmt.Errorf("cleanup networks: %w", err)
	}

	// 통계 캐시 정리
	m.factory.GetStatsCollector().ClearAllCache()

	return nil
}

// CleanupWorkspace 특정 워크스페이스의 모든 컨테이너를 정리합니다.
func (m *Manager) CleanupWorkspace(ctx context.Context, workspaceID string, force bool) error {
	return m.factory.GetContainerManager().CleanupWorkspace(ctx, workspaceID, force)
}

// Shutdown 매니저를 종료합니다.
func (m *Manager) Shutdown() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 컨텍스트 취소
	if m.cancel != nil {
		m.cancel()
	}

	// 팩토리 정리
	return m.factory.Close()
}

// 전역 매니저 인스턴스 관리
var (
	defaultManager *Manager
	defaultOnce    sync.Once
	defaultMu      sync.RWMutex
)

// GetDefaultManager 기본 매니저 인스턴스를 반환합니다.
func GetDefaultManager() (*Manager, error) {
	var err error
	defaultOnce.Do(func() {
		defaultManager, err = NewManagerWithDefaults()
	})
	return defaultManager, err
}

// SetDefaultManager 기본 매니저를 설정합니다.
func SetDefaultManager(manager *Manager) {
	defaultMu.Lock()
	defer defaultMu.Unlock()
	defaultManager = manager
}

// ResetDefaultManager 기본 매니저를 초기화합니다.
func ResetDefaultManager() {
	defaultMu.Lock()
	defer defaultMu.Unlock()
	if defaultManager != nil {
		defaultManager.Shutdown()
		defaultManager = nil
	}
	defaultOnce = sync.Once{}
}