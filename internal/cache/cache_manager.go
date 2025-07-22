package cache

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// CacheManager는 캐시 관리 인터페이스입니다
type CacheManager interface {
	// 계층화된 캐시
	GetL1Cache() Cache  // 메모리 캐시
	GetL2Cache() Cache  // 디스크 캐시
	
	// 캐시 전략
	SetEvictionPolicy(policy EvictionPolicy) error
	SetCacheSize(level int, size int64) error
	
	// 성능 최적화
	PrewarmCache(keys []string) error
	GetCacheStats() CacheStatistics
	
	// 생명주기
	Start() error
	Stop() error
}

// Cache는 캐시 인터페이스입니다
type Cache interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{}, ttl time.Duration) error
	Delete(key string) error
	Clear() error
	Size() int64
	Keys() []string
	Stats() CacheStats
}

// MultiLevelCacheManager는 다층 캐시 관리자입니다
type MultiLevelCacheManager struct {
	// 캐시 레벨들
	l1Cache *MemoryCache
	l2Cache *DiskCache
	
	// 설정
	config CacheConfig
	
	// 통계
	stats CacheStatistics
	statsMutex sync.RWMutex
	
	// 정책 관리
	evictionPolicy EvictionPolicy
	policyMutex    sync.RWMutex
	
	// 백그라운드 작업
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	
	// 상태
	running atomic.Bool
}

// CacheConfig는 캐시 설정입니다
type CacheConfig struct {
	// L1 캐시 (메모리)
	L1MaxSize       int64         `json:"l1_max_size"`
	L1TTL           time.Duration `json:"l1_ttl"`
	L1MaxEntries    int           `json:"l1_max_entries"`
	
	// L2 캐시 (디스크)
	L2MaxSize       int64         `json:"l2_max_size"`
	L2TTL           time.Duration `json:"l2_ttl"`
	L2MaxEntries    int           `json:"l2_max_entries"`
	L2Directory     string        `json:"l2_directory"`
	
	// 정책
	EvictionPolicy  EvictionPolicy `json:"eviction_policy"`
	
	// 성능
	PrewarmEnabled  bool          `json:"prewarm_enabled"`
	StatsInterval   time.Duration `json:"stats_interval"`
	CleanupInterval time.Duration `json:"cleanup_interval"`
	
	// 동기화
	L1ToL2Promotion bool          `json:"l1_to_l2_promotion"`
	L2ToL1Demotion  bool          `json:"l2_to_l1_demotion"`
	SyncInterval    time.Duration `json:"sync_interval"`
}

// EvictionPolicy는 축출 정책입니다
type EvictionPolicy int

const (
	EvictionLRU EvictionPolicy = iota  // Least Recently Used
	EvictionLFU                        // Least Frequently Used
	EvictionFIFO                       // First In First Out
	EvictionTTL                        // Time To Live
	EvictionAdaptive                   // 적응형
)

// CacheStatistics는 전체 캐시 통계입니다
type CacheStatistics struct {
	// 전체 통계
	TotalRequests   int64 `json:"total_requests"`
	TotalHits       int64 `json:"total_hits"`
	TotalMisses     int64 `json:"total_misses"`
	HitRate         float64 `json:"hit_rate"`
	
	// 레벨별 통계
	L1Stats CacheStats `json:"l1_stats"`
	L2Stats CacheStats `json:"l2_stats"`
	
	// 성능 메트릭
	AvgGetLatency   time.Duration `json:"avg_get_latency"`
	AvgSetLatency   time.Duration `json:"avg_set_latency"`
	MemoryUsage     int64         `json:"memory_usage"`
	DiskUsage       int64         `json:"disk_usage"`
	
	// 정책 효율성
	EvictionCount   int64   `json:"eviction_count"`
	PromotionCount  int64   `json:"promotion_count"`
	DemotionCount   int64   `json:"demotion_count"`
	PolicyEfficiency float64 `json:"policy_efficiency"`
	
	// 시간 정보
	LastUpdated     time.Time `json:"last_updated"`
	Uptime          time.Duration `json:"uptime"`
}

// CacheStats는 개별 캐시 통계입니다
type CacheStats struct {
	Hits            int64         `json:"hits"`
	Misses          int64         `json:"misses"`
	Sets            int64         `json:"sets"`
	Deletes         int64         `json:"deletes"`
	Evictions       int64         `json:"evictions"`
	Size            int64         `json:"size"`
	Entries         int           `json:"entries"`
	AvgLatency      time.Duration `json:"avg_latency"`
	HitRate         float64       `json:"hit_rate"`
	LastAccess      time.Time     `json:"last_access"`
}

// CacheEntry는 캐시 엔트리입니다
type CacheEntry struct {
	Key        string      `json:"key"`
	Value      interface{} `json:"value"`
	TTL        time.Duration `json:"ttl"`
	CreatedAt  time.Time   `json:"created_at"`
	AccessedAt time.Time   `json:"accessed_at"`
	AccessCount int64      `json:"access_count"`
	Size       int64       `json:"size"`
}

// DefaultCacheConfig는 기본 캐시 설정을 반환합니다
func DefaultCacheConfig() CacheConfig {
	return CacheConfig{
		L1MaxSize:       100 * 1024 * 1024, // 100MB
		L1TTL:           30 * time.Minute,
		L1MaxEntries:    10000,
		L2MaxSize:       1024 * 1024 * 1024, // 1GB
		L2TTL:           24 * time.Hour,
		L2MaxEntries:    100000,
		L2Directory:     "/tmp/aicli-cache",
		EvictionPolicy:  EvictionLRU,
		PrewarmEnabled:  true,
		StatsInterval:   10 * time.Second,
		CleanupInterval: 5 * time.Minute,
		L1ToL2Promotion: true,
		L2ToL1Demotion:  true,
		SyncInterval:    time.Minute,
	}
}

// NewMultiLevelCacheManager는 새로운 다층 캐시 관리자를 생성합니다
func NewMultiLevelCacheManager(config CacheConfig) (*MultiLevelCacheManager, error) {
	ctx, cancel := context.WithCancel(context.Background())
	
	// L1 캐시 (메모리) 생성
	l1Config := MemoryCacheConfig{
		MaxSize:     config.L1MaxSize,
		MaxEntries:  config.L1MaxEntries,
		DefaultTTL:  config.L1TTL,
		EvictionPolicy: config.EvictionPolicy,
	}
	l1Cache, err := NewMemoryCache(l1Config)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create L1 cache: %w", err)
	}
	
	// L2 캐시 (디스크) 생성
	l2Config := DiskCacheConfig{
		Directory:   config.L2Directory,
		MaxSize:     config.L2MaxSize,
		MaxEntries:  config.L2MaxEntries,
		DefaultTTL:  config.L2TTL,
		EvictionPolicy: config.EvictionPolicy,
	}
	l2Cache, err := NewDiskCache(l2Config)
	if err != nil {
		cancel()
		l1Cache.Stop()
		return nil, fmt.Errorf("failed to create L2 cache: %w", err)
	}
	
	manager := &MultiLevelCacheManager{
		l1Cache:        l1Cache,
		l2Cache:        l2Cache,
		config:         config,
		evictionPolicy: config.EvictionPolicy,
		ctx:            ctx,
		cancel:         cancel,
		stats: CacheStatistics{
			LastUpdated: time.Now(),
		},
	}
	
	return manager, nil
}

// Start는 캐시 관리자를 시작합니다
func (cm *MultiLevelCacheManager) Start() error {
	if !cm.running.CompareAndSwap(false, true) {
		return fmt.Errorf("cache manager is already running")
	}
	
	// L1, L2 캐시 시작
	if err := cm.l1Cache.Start(); err != nil {
		return fmt.Errorf("failed to start L1 cache: %w", err)
	}
	
	if err := cm.l2Cache.Start(); err != nil {
		cm.l1Cache.Stop()
		return fmt.Errorf("failed to start L2 cache: %w", err)
	}
	
	// 백그라운드 작업들 시작
	cm.wg.Add(3)
	go cm.statisticsCollector()
	go cm.cleanupWorker()
	go cm.synchronizer()
	
	return nil
}

// Stop은 캐시 관리자를 중지합니다
func (cm *MultiLevelCacheManager) Stop() error {
	if !cm.running.CompareAndSwap(true, false) {
		return nil // 이미 중지됨
	}
	
	// 컨텍스트 취소
	cm.cancel()
	
	// 백그라운드 작업 완료 대기
	cm.wg.Wait()
	
	// 캐시들 중지
	if err := cm.l1Cache.Stop(); err != nil {
		return fmt.Errorf("failed to stop L1 cache: %w", err)
	}
	
	if err := cm.l2Cache.Stop(); err != nil {
		return fmt.Errorf("failed to stop L2 cache: %w", err)
	}
	
	return nil
}

// GetL1Cache는 L1 캐시를 반환합니다
func (cm *MultiLevelCacheManager) GetL1Cache() Cache {
	return cm.l1Cache
}

// GetL2Cache는 L2 캐시를 반환합니다
func (cm *MultiLevelCacheManager) GetL2Cache() Cache {
	return cm.l2Cache
}

// SetEvictionPolicy는 축출 정책을 설정합니다
func (cm *MultiLevelCacheManager) SetEvictionPolicy(policy EvictionPolicy) error {
	cm.policyMutex.Lock()
	cm.evictionPolicy = policy
	cm.policyMutex.Unlock()
	
	// 각 캐시에 정책 적용
	cm.l1Cache.SetEvictionPolicy(policy)
	cm.l2Cache.SetEvictionPolicy(policy)
	
	return nil
}

// SetCacheSize는 캐시 크기를 설정합니다
func (cm *MultiLevelCacheManager) SetCacheSize(level int, size int64) error {
	switch level {
	case 1:
		return cm.l1Cache.SetMaxSize(size)
	case 2:
		return cm.l2Cache.SetMaxSize(size)
	default:
		return fmt.Errorf("invalid cache level: %d", level)
	}
}

// PrewarmCache는 캐시를 미리 워밍합니다
func (cm *MultiLevelCacheManager) PrewarmCache(keys []string) error {
	if !cm.config.PrewarmEnabled {
		return nil
	}
	
	// 병렬로 키들을 로드
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 10) // 동시 실행 제한
	
	for _, key := range keys {
		wg.Add(1)
		go func(k string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			
			// L2에서 L1으로 프로모션 시도
			if value, exists := cm.l2Cache.Get(k); exists {
				cm.l1Cache.Set(k, value, cm.config.L1TTL)
			}
		}(key)
	}
	
	wg.Wait()
	return nil
}

// GetCacheStats는 캐시 통계를 반환합니다
func (cm *MultiLevelCacheManager) GetCacheStats() CacheStatistics {
	cm.statsMutex.RLock()
	defer cm.statsMutex.RUnlock()
	
	stats := cm.stats
	stats.L1Stats = cm.l1Cache.Stats()
	stats.L2Stats = cm.l2Cache.Stats()
	stats.LastUpdated = time.Now()
	
	// 전체 히트율 계산
	totalRequests := stats.L1Stats.Hits + stats.L1Stats.Misses + stats.L2Stats.Hits + stats.L2Stats.Misses
	totalHits := stats.L1Stats.Hits + stats.L2Stats.Hits
	
	if totalRequests > 0 {
		stats.HitRate = float64(totalHits) / float64(totalRequests)
	}
	
	stats.TotalRequests = totalRequests
	stats.TotalHits = totalHits
	stats.TotalMisses = totalRequests - totalHits
	
	return stats
}

// Get은 다층 캐시에서 값을 조회합니다
func (cm *MultiLevelCacheManager) Get(key string) (interface{}, bool) {
	start := time.Now()
	defer func() {
		cm.updateLatency("get", time.Since(start))
	}()
	
	// L1 캐시에서 먼저 시도
	if value, exists := cm.l1Cache.Get(key); exists {
		atomic.AddInt64(&cm.stats.TotalRequests, 1)
		atomic.AddInt64(&cm.stats.TotalHits, 1)
		return value, true
	}
	
	// L2 캐시에서 시도
	if value, exists := cm.l2Cache.Get(key); exists {
		atomic.AddInt64(&cm.stats.TotalRequests, 1)
		atomic.AddInt64(&cm.stats.TotalHits, 1)
		
		// L1으로 프로모션
		if cm.config.L1ToL2Promotion {
			cm.l1Cache.Set(key, value, cm.config.L1TTL)
			atomic.AddInt64(&cm.stats.PromotionCount, 1)
		}
		
		return value, true
	}
	
	atomic.AddInt64(&cm.stats.TotalRequests, 1)
	atomic.AddInt64(&cm.stats.TotalMisses, 1)
	return nil, false
}

// Set은 다층 캐시에 값을 설정합니다
func (cm *MultiLevelCacheManager) Set(key string, value interface{}, ttl time.Duration) error {
	start := time.Now()
	defer func() {
		cm.updateLatency("set", time.Since(start))
	}()
	
	// L1에 설정
	if err := cm.l1Cache.Set(key, value, ttl); err != nil {
		return fmt.Errorf("failed to set in L1 cache: %w", err)
	}
	
	// L2에도 설정 (백그라운드에서)
	go func() {
		cm.l2Cache.Set(key, value, cm.config.L2TTL)
	}()
	
	return nil
}

// Delete는 모든 캐시 레벨에서 키를 삭제합니다
func (cm *MultiLevelCacheManager) Delete(key string) error {
	var errs []error
	
	if err := cm.l1Cache.Delete(key); err != nil {
		errs = append(errs, fmt.Errorf("L1 delete failed: %w", err))
	}
	
	if err := cm.l2Cache.Delete(key); err != nil {
		errs = append(errs, fmt.Errorf("L2 delete failed: %w", err))
	}
	
	if len(errs) > 0 {
		return fmt.Errorf("cache delete errors: %v", errs)
	}
	
	return nil
}

// Clear는 모든 캐시를 정리합니다
func (cm *MultiLevelCacheManager) Clear() error {
	var errs []error
	
	if err := cm.l1Cache.Clear(); err != nil {
		errs = append(errs, fmt.Errorf("L1 clear failed: %w", err))
	}
	
	if err := cm.l2Cache.Clear(); err != nil {
		errs = append(errs, fmt.Errorf("L2 clear failed: %w", err))
	}
	
	if len(errs) > 0 {
		return fmt.Errorf("cache clear errors: %v", errs)
	}
	
	return nil
}

// 내부 메서드들

func (cm *MultiLevelCacheManager) updateLatency(operation string, latency time.Duration) {
	cm.statsMutex.Lock()
	defer cm.statsMutex.Unlock()
	
	switch operation {
	case "get":
		cm.stats.AvgGetLatency = (cm.stats.AvgGetLatency + latency) / 2
	case "set":
		cm.stats.AvgSetLatency = (cm.stats.AvgSetLatency + latency) / 2
	}
}

func (cm *MultiLevelCacheManager) statisticsCollector() {
	defer cm.wg.Done()
	
	ticker := time.NewTicker(cm.config.StatsInterval)
	defer ticker.Stop()
	
	startTime := time.Now()
	
	for {
		select {
		case <-cm.ctx.Done():
			return
		case <-ticker.C:
			cm.statsMutex.Lock()
			cm.stats.Uptime = time.Since(startTime)
			cm.stats.MemoryUsage = cm.l1Cache.Size()
			cm.stats.DiskUsage = cm.l2Cache.Size()
			cm.stats.LastUpdated = time.Now()
			cm.statsMutex.Unlock()
		}
	}
}

func (cm *MultiLevelCacheManager) cleanupWorker() {
	defer cm.wg.Done()
	
	ticker := time.NewTicker(cm.config.CleanupInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-cm.ctx.Done():
			return
		case <-ticker.C:
			cm.performCleanup()
		}
	}
}

func (cm *MultiLevelCacheManager) performCleanup() {
	// 만료된 엔트리 정리
	cm.l1Cache.CleanupExpired()
	cm.l2Cache.CleanupExpired()
	
	// 정책에 따른 축출 수행
	cm.enforceEvictionPolicy()
}

func (cm *MultiLevelCacheManager) enforceEvictionPolicy() {
	// L1 캐시 크기 확인 및 축출
	if cm.l1Cache.Size() > cm.config.L1MaxSize {
		keysToEvict := cm.l1Cache.GetEvictionCandidates(cm.evictionPolicy, 10)
		for _, key := range keysToEvict {
			if value, exists := cm.l1Cache.Get(key); exists {
				// L2로 데모션
				if cm.config.L2ToL1Demotion {
					cm.l2Cache.Set(key, value, cm.config.L2TTL)
					atomic.AddInt64(&cm.stats.DemotionCount, 1)
				}
				cm.l1Cache.Delete(key)
				atomic.AddInt64(&cm.stats.EvictionCount, 1)
			}
		}
	}
	
	// L2 캐시 크기 확인 및 축출
	if cm.l2Cache.Size() > cm.config.L2MaxSize {
		keysToEvict := cm.l2Cache.GetEvictionCandidates(cm.evictionPolicy, 20)
		for _, key := range keysToEvict {
			cm.l2Cache.Delete(key)
			atomic.AddInt64(&cm.stats.EvictionCount, 1)
		}
	}
}

func (cm *MultiLevelCacheManager) synchronizer() {
	defer cm.wg.Done()
	
	ticker := time.NewTicker(cm.config.SyncInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-cm.ctx.Done():
			return
		case <-ticker.C:
			cm.performSync()
		}
	}
}

func (cm *MultiLevelCacheManager) performSync() {
	// L1과 L2 간 동기화 로직
	// 실제 구현에서는 더 정교한 동기화 필요
	
	// 정책 효율성 계산
	cm.calculatePolicyEfficiency()
}

func (cm *MultiLevelCacheManager) calculatePolicyEfficiency() {
	stats := cm.GetCacheStats()
	
	// 간단한 효율성 점수 계산
	efficiency := stats.HitRate * 0.7 + 
		(1.0 - float64(stats.EvictionCount)/float64(stats.TotalRequests)) * 0.3
	
	cm.statsMutex.Lock()
	cm.stats.PolicyEfficiency = efficiency
	cm.statsMutex.Unlock()
}

// String returns the eviction policy as a string
func (ep EvictionPolicy) String() string {
	switch ep {
	case EvictionLRU:
		return "LRU"
	case EvictionLFU:
		return "LFU"
	case EvictionFIFO:
		return "FIFO"
	case EvictionTTL:
		return "TTL"
	case EvictionAdaptive:
		return "Adaptive"
	default:
		return "Unknown"
	}
}