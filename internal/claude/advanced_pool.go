package claude

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// AdvancedSessionPool은 고급 세션 풀 관리자입니다
type AdvancedSessionPool struct {
	basePool      *SessionPool
	scaler        *AutoScaler
	monitor       *PoolMonitor
	loadBalancer  *LoadBalancer
	healthChecker *HealthChecker
	metrics       *PoolMetrics
	
	// 설정
	config        AdvancedPoolConfig
	
	// 동적 상태
	currentLoad   atomic.Value // float64
	lastScaleTime time.Time
	
	// 동시성 제어
	mu sync.RWMutex
	
	// 생명주기 관리
	ctx    context.Context
	cancel context.CancelFunc
}

// AdvancedPoolConfig는 고급 풀 설정입니다
type AdvancedPoolConfig struct {
	// 기본 풀 설정
	BaseConfig SessionPoolConfig
	
	// 동적 스케일링 설정
	AutoScaling AutoScalingConfig
	
	// 부하 분산 설정
	LoadBalancing LoadBalancingConfig
	
	// 모니터링 설정
	Monitoring MonitoringConfig
	
	// 헬스 체크 설정
	HealthCheck HealthCheckConfig
}

// AutoScalingConfig는 자동 스케일링 설정입니다
type AutoScalingConfig struct {
	Enabled            bool          `json:"enabled"`
	MinSessions        int           `json:"min_sessions"`
	MaxSessions        int           `json:"max_sessions"`
	TargetUtilization  float64       `json:"target_utilization"`
	ScaleUpThreshold   float64       `json:"scale_up_threshold"`
	ScaleDownThreshold float64       `json:"scale_down_threshold"`
	ScaleUpCooldown    time.Duration `json:"scale_up_cooldown"`
	ScaleDownCooldown  time.Duration `json:"scale_down_cooldown"`
	ScaleFactor        float64       `json:"scale_factor"`
}

// LoadBalancingConfig는 부하 분산 설정입니다
type LoadBalancingConfig struct {
	Strategy         LoadBalancingStrategy `json:"strategy"`
	SessionAffinity  bool                  `json:"session_affinity"`
	WeightedRouting  bool                  `json:"weighted_routing"`
	HealthAware      bool                  `json:"health_aware"`
	StickyDuration   time.Duration         `json:"sticky_duration"`
}

// MonitoringConfig는 모니터링 설정입니다
type MonitoringConfig struct {
	MetricsInterval     time.Duration `json:"metrics_interval"`
	EnableCPUTracking   bool          `json:"enable_cpu_tracking"`
	EnableMemoryTracking bool         `json:"enable_memory_tracking"`
	AlertThresholds     AlertThresholds `json:"alert_thresholds"`
}

// HealthCheckConfig는 헬스 체크 설정입니다
type HealthCheckConfig struct {
	Interval        time.Duration `json:"interval"`
	Timeout         time.Duration `json:"timeout"`
	FailureThreshold int          `json:"failure_threshold"`
	SuccessThreshold int          `json:"success_threshold"`
}

// LoadBalancingStrategy는 부하 분산 전략입니다
type LoadBalancingStrategy int

const (
	RoundRobin LoadBalancingStrategy = iota
	LeastConnections
	WeightedRoundRobin
	ResourceBased
	ResponseTimeBased
)

// AlertThresholds는 알람 임계값입니다
type AlertThresholds struct {
	HighCPUUsage    float64 `json:"high_cpu_usage"`
	HighMemoryUsage int64   `json:"high_memory_usage"`
	HighErrorRate   float64 `json:"high_error_rate"`
	LowAvailability float64 `json:"low_availability"`
}

// PoolStatistics는 풀 통계 정보입니다
type PoolStatistics struct {
	Size             int           `json:"size"`
	ActiveSessions   int           `json:"active_sessions"`
	IdleSessions     int           `json:"idle_sessions"`
	MemoryUsage      int64         `json:"memory_usage"`
	CPUUsage         float64       `json:"cpu_usage"`
	ThroughputRPS    float64       `json:"throughput_rps"`
	AverageLatency   time.Duration `json:"average_latency"`
	ErrorRate        float64       `json:"error_rate"`
	Utilization      float64       `json:"utilization"`
	LastScaleAction  string        `json:"last_scale_action"`
	LastScaleTime    time.Time     `json:"last_scale_time"`
}

// SessionMetrics는 개별 세션 메트릭입니다
type SessionMetrics struct {
	SessionID     string        `json:"session_id"`
	StartTime     time.Time     `json:"start_time"`
	LastUsed      time.Time     `json:"last_used"`
	RequestCount  int64         `json:"request_count"`
	MemoryUsage   int64         `json:"memory_usage"`
	CPUUsage      float64       `json:"cpu_usage"`
	Status        SessionStatus `json:"status"`
	ResponseTime  time.Duration `json:"response_time"`
	ErrorCount    int64         `json:"error_count"`
	Weight        float64       `json:"weight"`
	AffinityKey   string        `json:"affinity_key"`
}

// DefaultAdvancedPoolConfig는 기본 고급 풀 설정을 반환합니다
func DefaultAdvancedPoolConfig() AdvancedPoolConfig {
	return AdvancedPoolConfig{
		BaseConfig: DefaultSessionPoolConfig(),
		AutoScaling: AutoScalingConfig{
			Enabled:            true,
			MinSessions:        5,
			MaxSessions:        100,
			TargetUtilization:  0.7,
			ScaleUpThreshold:   0.8,
			ScaleDownThreshold: 0.3,
			ScaleUpCooldown:    2 * time.Minute,
			ScaleDownCooldown:  5 * time.Minute,
			ScaleFactor:        1.5,
		},
		LoadBalancing: LoadBalancingConfig{
			Strategy:         WeightedRoundRobin,
			SessionAffinity:  true,
			WeightedRouting:  true,
			HealthAware:      true,
			StickyDuration:   30 * time.Minute,
		},
		Monitoring: MonitoringConfig{
			MetricsInterval:      30 * time.Second,
			EnableCPUTracking:    true,
			EnableMemoryTracking: true,
			AlertThresholds: AlertThresholds{
				HighCPUUsage:    0.8,
				HighMemoryUsage: 1024 * 1024 * 1024, // 1GB
				HighErrorRate:   0.05,                // 5%
				LowAvailability: 0.95,                // 95%
			},
		},
		HealthCheck: HealthCheckConfig{
			Interval:         30 * time.Second,
			Timeout:          5 * time.Second,
			FailureThreshold: 3,
			SuccessThreshold: 2,
		},
	}
}

// NewAdvancedSessionPool은 새로운 고급 세션 풀을 생성합니다
func NewAdvancedSessionPool(manager SessionManager, config AdvancedPoolConfig) *AdvancedSessionPool {
	ctx, cancel := context.WithCancel(context.Background())
	
	// 기본 풀 생성
	basePool := NewSessionPool(manager, config.BaseConfig)
	
	pool := &AdvancedSessionPool{
		basePool:      basePool,
		config:        config,
		ctx:           ctx,
		cancel:        cancel,
		lastScaleTime: time.Now(),
	}
	
	// 컴포넌트 초기화
	pool.scaler = NewAutoScaler(pool, config.AutoScaling)
	pool.monitor = NewPoolMonitor(pool, config.Monitoring)
	pool.loadBalancer = NewLoadBalancer(pool, config.LoadBalancing)
	pool.healthChecker = NewHealthChecker(pool, config.HealthCheck)
	pool.metrics = NewPoolMetrics()
	
	// 초기 부하값 설정
	pool.currentLoad.Store(0.0)
	
	// 백그라운드 작업 시작
	pool.startBackgroundTasks()
	
	return pool
}

// AcquireSession은 풀에서 세션을 가져옵니다 (고급 기능 포함)
func (p *AdvancedSessionPool) AcquireSession(ctx context.Context, config SessionConfig) (*PooledSession, error) {
	// 1. 부하 기반 검증
	if err := p.validateLoad(); err != nil {
		return nil, fmt.Errorf("load validation failed: %w", err)
	}
	
	// 2. 세션 어피니티 확인
	if affinitySession := p.findAffinitySession(config); affinitySession != nil {
		p.updateMetrics(affinitySession, "affinity_hit")
		return affinitySession, nil
	}
	
	// 3. 부하 분산을 통한 세션 선택
	session, err := p.loadBalancer.SelectSession(ctx, config)
	if err == nil && session != nil {
		p.updateMetrics(session, "load_balanced")
		return session, nil
	}
	
	// 4. 기본 풀에서 세션 획득 시도
	session, err = p.basePool.AcquireSession(ctx, config)
	if err != nil {
		// 5. 자동 스케일링 시도
		if p.config.AutoScaling.Enabled {
			if scaleErr := p.scaler.ScaleUp(); scaleErr == nil {
				// 스케일업 후 재시도
				session, err = p.basePool.AcquireSession(ctx, config)
			}
		}
	}
	
	if err != nil {
		p.updateMetrics(nil, "acquisition_failed")
		return nil, fmt.Errorf("failed to acquire session: %w", err)
	}
	
	// 6. 세션 메트릭 초기화
	p.initSessionMetrics(session)
	p.updateMetrics(session, "acquired")
	
	return session, nil
}

// ReleaseSession은 세션을 풀에 반환합니다
func (p *AdvancedSessionPool) ReleaseSession(sessionID string) error {
	// 기본 풀에 반환
	err := p.basePool.ReleaseSession(sessionID)
	
	// 메트릭 업데이트
	if err == nil {
		p.updateMetrics(nil, "released")
		
		// 스케일다운 검토
		if p.config.AutoScaling.Enabled {
			go p.scaler.ConsiderScaleDown()
		}
	} else {
		p.updateMetrics(nil, "release_failed")
	}
	
	return err
}

// Scale은 풀 크기를 동적으로 조정합니다
func (p *AdvancedSessionPool) Scale(targetSize int) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	currentSize := p.basePool.GetPoolStats().TotalSessions
	
	if targetSize > currentSize {
		// 스케일 업
		return p.scaleUp(targetSize - currentSize)
	} else if targetSize < currentSize {
		// 스케일 다운
		return p.scaleDown(currentSize - targetSize)
	}
	
	return nil // 변경 불필요
}

// AutoScale은 자동 스케일링을 활성화/비활성화합니다
func (p *AdvancedSessionPool) AutoScale(enable bool) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	p.config.AutoScaling.Enabled = enable
	
	if enable {
		p.scaler.Start()
	} else {
		p.scaler.Stop()
	}
	
	return nil
}

// GetPoolStats는 풀 통계를 반환합니다
func (p *AdvancedSessionPool) GetPoolStats() PoolStatistics {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	baseStats := p.basePool.GetPoolStats()
	
	var memUsage runtime.MemStats
	runtime.ReadMemStats(&memUsage)
	
	return PoolStatistics{
		Size:            baseStats.TotalSessions,
		ActiveSessions:  baseStats.ActiveSessions,
		IdleSessions:    baseStats.IdleSessions,
		MemoryUsage:     int64(memUsage.Alloc),
		CPUUsage:        p.monitor.GetCurrentCPUUsage(),
		ThroughputRPS:   p.metrics.GetThroughputRPS(),
		AverageLatency:  p.metrics.GetAverageLatency(),
		ErrorRate:       p.metrics.GetErrorRate(),
		Utilization:     p.calculateUtilization(),
		LastScaleAction: p.scaler.GetLastAction(),
		LastScaleTime:   p.lastScaleTime,
	}
}

// GetSessionMetrics는 세션별 메트릭을 반환합니다
func (p *AdvancedSessionPool) GetSessionMetrics() []SessionMetrics {
	return p.monitor.GetSessionMetrics()
}

// Shutdown은 고급 풀을 종료합니다
func (p *AdvancedSessionPool) Shutdown() error {
	p.cancel()
	
	// 백그라운드 작업 중지
	if p.scaler != nil {
		p.scaler.Stop()
	}
	if p.monitor != nil {
		p.monitor.Stop()
	}
	if p.healthChecker != nil {
		p.healthChecker.Stop()
	}
	
	// 기본 풀 종료
	return p.basePool.Shutdown()
}

// 내부 메서드들

func (p *AdvancedSessionPool) validateLoad() error {
	load := p.currentLoad.Load().(float64)
	if load > 0.9 { // 90% 이상 부하시 거부
		return fmt.Errorf("system overloaded: %.2f", load)
	}
	return nil
}

func (p *AdvancedSessionPool) findAffinitySession(config SessionConfig) *PooledSession {
	if !p.config.LoadBalancing.SessionAffinity {
		return nil
	}
	
	return p.loadBalancer.FindAffinitySession(config)
}

func (p *AdvancedSessionPool) initSessionMetrics(session *PooledSession) {
	metrics := SessionMetrics{
		SessionID:    session.ID,
		StartTime:    time.Now(),
		LastUsed:     time.Now(),
		RequestCount: 0,
		Status:       SessionStatusActive,
		Weight:       1.0,
	}
	
	p.monitor.SetSessionMetrics(session.ID, metrics)
}

func (p *AdvancedSessionPool) updateMetrics(session *PooledSession, action string) {
	p.metrics.RecordAction(action, session)
	
	if session != nil {
		p.monitor.UpdateSessionMetrics(session.ID, action)
	}
}

func (p *AdvancedSessionPool) scaleUp(count int) error {
	// 최대값 체크
	currentSize := p.basePool.GetPoolStats().TotalSessions
	if currentSize+count > p.config.AutoScaling.MaxSessions {
		count = p.config.AutoScaling.MaxSessions - currentSize
	}
	
	// 실제 스케일링은 AutoScaler에서 처리
	return p.scaler.ScaleUpBy(count)
}

func (p *AdvancedSessionPool) scaleDown(count int) error {
	// 최소값 체크
	currentSize := p.basePool.GetPoolStats().TotalSessions
	if currentSize-count < p.config.AutoScaling.MinSessions {
		count = currentSize - p.config.AutoScaling.MinSessions
	}
	
	// 실제 스케일링은 AutoScaler에서 처리
	return p.scaler.ScaleDownBy(count)
}

func (p *AdvancedSessionPool) calculateUtilization() float64 {
	stats := p.basePool.GetPoolStats()
	if stats.TotalSessions == 0 {
		return 0.0
	}
	
	return float64(stats.ActiveSessions) / float64(stats.TotalSessions)
}

func (p *AdvancedSessionPool) startBackgroundTasks() {
	// 모니터링 시작
	go p.monitor.Start()
	
	// 헬스 체커 시작
	go p.healthChecker.Start()
	
	// 자동 스케일러 시작 (활성화된 경우)
	if p.config.AutoScaling.Enabled {
		go p.scaler.Start()
	}
	
	// 메트릭 수집 시작
	go p.metrics.Start()
}

// SessionStatus는 세션 상태입니다
type SessionStatus int

const (
	SessionStatusActive SessionStatus = iota
	SessionStatusIdle
	SessionStatusBusy
	SessionStatusError
	SessionStatusClosed
)