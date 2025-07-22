package cache

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// MemoryCache는 메모리 기반 L1 캐시입니다
type MemoryCache struct {
	// 데이터 저장소
	data      map[string]*MemoryCacheEntry
	dataMutex sync.RWMutex
	
	// LRU 순서 추적
	accessOrder []string
	accessMutex sync.Mutex
	
	// 설정
	config MemoryCacheConfig
	
	// 통계
	stats MemoryCacheStats
	statsMutex sync.RWMutex
	
	// 생명주기
	running atomic.Bool
}

// MemoryCacheConfig는 메모리 캐시 설정입니다
type MemoryCacheConfig struct {
	MaxSize        int64         `json:"max_size"`        // 최대 메모리 크기 (바이트)
	MaxEntries     int           `json:"max_entries"`     // 최대 엔트리 수
	DefaultTTL     time.Duration `json:"default_ttl"`     // 기본 TTL
	EvictionPolicy EvictionPolicy `json:"eviction_policy"` // 축출 정책
	CleanupInterval time.Duration `json:"cleanup_interval"` // 정리 간격
}

// MemoryCacheEntry는 메모리 캐시 엔트리입니다
type MemoryCacheEntry struct {
	Key         string      `json:"key"`
	Value       interface{} `json:"value"`
	TTL         time.Duration `json:"ttl"`
	CreatedAt   time.Time   `json:"created_at"`
	AccessedAt  time.Time   `json:"accessed_at"`
	AccessCount int64       `json:"access_count"`
	Size        int64       `json:"size"`
	Expired     bool        `json:"expired"`
}

// MemoryCacheStats는 메모리 캐시 통계입니다
type MemoryCacheStats struct {
	Hits        int64         `json:"hits"`
	Misses      int64         `json:"misses"`
	Sets        int64         `json:"sets"`
	Deletes     int64         `json:"deletes"`
	Evictions   int64         `json:"evictions"`
	Expired     int64         `json:"expired"`
	Size        int64         `json:"size"`
	Entries     int           `json:"entries"`
	HitRate     float64       `json:"hit_rate"`
	AvgLatency  time.Duration `json:"avg_latency"`
	LastAccess  time.Time     `json:"last_access"`
}

// NewMemoryCache는 새로운 메모리 캐시를 생성합니다
func NewMemoryCache(config MemoryCacheConfig) (*MemoryCache, error) {
	if config.MaxSize <= 0 {
		return nil, fmt.Errorf("max size must be positive")
	}
	
	if config.MaxEntries <= 0 {
		return nil, fmt.Errorf("max entries must be positive")
	}
	
	cache := &MemoryCache{
		data:        make(map[string]*MemoryCacheEntry),
		accessOrder: make([]string, 0),
		config:      config,
		stats: MemoryCacheStats{
			LastAccess: time.Now(),
		},
	}
	
	return cache, nil
}

// Start는 메모리 캐시를 시작합니다
func (mc *MemoryCache) Start() error {
	if !mc.running.CompareAndSwap(false, true) {
		return fmt.Errorf("memory cache is already running")
	}
	
	// 정리 작업 시작 (실제 구현에서는 별도 고루틴)
	return nil
}

// Stop은 메모리 캐시를 중지합니다
func (mc *MemoryCache) Stop() error {
	if !mc.running.CompareAndSwap(true, false) {
		return nil
	}
	
	// 데이터 정리
	mc.dataMutex.Lock()
	mc.data = make(map[string]*MemoryCacheEntry)
	mc.dataMutex.Unlock()
	
	mc.accessMutex.Lock()
	mc.accessOrder = mc.accessOrder[:0]
	mc.accessMutex.Unlock()
	
	return nil
}

// Get은 키에 해당하는 값을 조회합니다
func (mc *MemoryCache) Get(key string) (interface{}, bool) {
	start := time.Now()
	defer func() {
		mc.updateLatency(time.Since(start))
		mc.statsMutex.Lock()
		mc.stats.LastAccess = time.Now()
		mc.statsMutex.Unlock()
	}()
	
	mc.dataMutex.RLock()
	entry, exists := mc.data[key]
	mc.dataMutex.RUnlock()
	
	if !exists {
		atomic.AddInt64(&mc.stats.Misses, 1)
		return nil, false
	}
	
	// TTL 확인
	if mc.isExpired(entry) {
		mc.dataMutex.Lock()
		delete(mc.data, key)
		mc.dataMutex.Unlock()
		
		mc.removeFromAccessOrder(key)
		atomic.AddInt64(&mc.stats.Misses, 1)
		atomic.AddInt64(&mc.stats.Expired, 1)
		return nil, false
	}
	
	// 접근 정보 업데이트
	entry.AccessedAt = time.Now()
	atomic.AddInt64(&entry.AccessCount, 1)
	
	// LRU 순서 업데이트
	mc.updateAccessOrder(key)
	
	atomic.AddInt64(&mc.stats.Hits, 1)
	return entry.Value, true
}

// Set은 키-값 쌍을 설정합니다
func (mc *MemoryCache) Set(key string, value interface{}, ttl time.Duration) error {
	if !mc.running.Load() {
		return fmt.Errorf("memory cache is not running")
	}
	
	if ttl == 0 {
		ttl = mc.config.DefaultTTL
	}
	
	size := mc.estimateSize(value)
	now := time.Now()
	
	entry := &MemoryCacheEntry{
		Key:         key,
		Value:       value,
		TTL:         ttl,
		CreatedAt:   now,
		AccessedAt:  now,
		AccessCount: 1,
		Size:        size,
		Expired:     false,
	}
	
	mc.dataMutex.Lock()
	
	// 기존 엔트리가 있으면 크기 차이 계산
	var sizeDiff int64 = size
	if existingEntry, exists := mc.data[key]; exists {
		sizeDiff = size - existingEntry.Size
	}
	
	// 용량 확인 및 축출
	if mc.stats.Size+sizeDiff > mc.config.MaxSize || len(mc.data) >= mc.config.MaxEntries {
		if err := mc.evictEntries(sizeDiff); err != nil {
			mc.dataMutex.Unlock()
			return fmt.Errorf("failed to evict entries: %w", err)
		}
	}
	
	mc.data[key] = entry
	mc.dataMutex.Unlock()
	
	// 통계 업데이트
	atomic.AddInt64(&mc.stats.Sets, 1)
	atomic.AddInt64(&mc.stats.Size, sizeDiff)
	
	mc.statsMutex.Lock()
	mc.stats.Entries = len(mc.data)
	mc.statsMutex.Unlock()
	
	// 접근 순서 업데이트
	mc.updateAccessOrder(key)
	
	return nil
}

// Delete는 키를 삭제합니다
func (mc *MemoryCache) Delete(key string) error {
	mc.dataMutex.Lock()
	entry, exists := mc.data[key]
	if exists {
		delete(mc.data, key)
	}
	mc.dataMutex.Unlock()
	
	if !exists {
		return fmt.Errorf("key not found: %s", key)
	}
	
	// 접근 순서에서 제거
	mc.removeFromAccessOrder(key)
	
	// 통계 업데이트
	atomic.AddInt64(&mc.stats.Deletes, 1)
	atomic.AddInt64(&mc.stats.Size, -entry.Size)
	
	mc.statsMutex.Lock()
	mc.stats.Entries = len(mc.data)
	mc.statsMutex.Unlock()
	
	return nil
}

// Clear는 모든 엔트리를 삭제합니다
func (mc *MemoryCache) Clear() error {
	mc.dataMutex.Lock()
	mc.data = make(map[string]*MemoryCacheEntry)
	mc.dataMutex.Unlock()
	
	mc.accessMutex.Lock()
	mc.accessOrder = mc.accessOrder[:0]
	mc.accessMutex.Unlock()
	
	// 통계 리셋
	mc.statsMutex.Lock()
	mc.stats.Size = 0
	mc.stats.Entries = 0
	mc.statsMutex.Unlock()
	
	return nil
}

// Size는 캐시 크기를 반환합니다
func (mc *MemoryCache) Size() int64 {
	return atomic.LoadInt64(&mc.stats.Size)
}

// Keys는 모든 키를 반환합니다
func (mc *MemoryCache) Keys() []string {
	mc.dataMutex.RLock()
	defer mc.dataMutex.RUnlock()
	
	keys := make([]string, 0, len(mc.data))
	for key := range mc.data {
		if !mc.isExpired(mc.data[key]) {
			keys = append(keys, key)
		}
	}
	
	return keys
}

// Stats는 캐시 통계를 반환합니다
func (mc *MemoryCache) Stats() CacheStats {
	mc.statsMutex.RLock()
	defer mc.statsMutex.RUnlock()
	
	stats := CacheStats{
		Hits:      atomic.LoadInt64(&mc.stats.Hits),
		Misses:    atomic.LoadInt64(&mc.stats.Misses),
		Sets:      atomic.LoadInt64(&mc.stats.Sets),
		Deletes:   atomic.LoadInt64(&mc.stats.Deletes),
		Evictions: atomic.LoadInt64(&mc.stats.Evictions),
		Size:      atomic.LoadInt64(&mc.stats.Size),
		Entries:   mc.stats.Entries,
		AvgLatency: mc.stats.AvgLatency,
		LastAccess: mc.stats.LastAccess,
	}
	
	// 히트율 계산
	totalRequests := stats.Hits + stats.Misses
	if totalRequests > 0 {
		stats.HitRate = float64(stats.Hits) / float64(totalRequests)
	}
	
	return stats
}

// SetEvictionPolicy는 축출 정책을 설정합니다
func (mc *MemoryCache) SetEvictionPolicy(policy EvictionPolicy) {
	mc.config.EvictionPolicy = policy
}

// SetMaxSize는 최대 크기를 설정합니다
func (mc *MemoryCache) SetMaxSize(size int64) error {
	if size <= 0 {
		return fmt.Errorf("max size must be positive")
	}
	
	mc.config.MaxSize = size
	
	// 현재 크기가 새로운 최대값보다 크면 축출
	if mc.Size() > size {
		mc.dataMutex.Lock()
		mc.evictEntries(mc.Size() - size)
		mc.dataMutex.Unlock()
	}
	
	return nil
}

// CleanupExpired는 만료된 엔트리들을 정리합니다
func (mc *MemoryCache) CleanupExpired() {
	mc.dataMutex.Lock()
	defer mc.dataMutex.Unlock()
	
	var expiredKeys []string
	var expiredSize int64
	
	for key, entry := range mc.data {
		if mc.isExpired(entry) {
			expiredKeys = append(expiredKeys, key)
			expiredSize += entry.Size
		}
	}
	
	// 만료된 엔트리 삭제
	for _, key := range expiredKeys {
		delete(mc.data, key)
		mc.removeFromAccessOrder(key)
	}
	
	// 통계 업데이트
	atomic.AddInt64(&mc.stats.Expired, int64(len(expiredKeys)))
	atomic.AddInt64(&mc.stats.Size, -expiredSize)
	
	mc.statsMutex.Lock()
	mc.stats.Entries = len(mc.data)
	mc.statsMutex.Unlock()
}

// GetEvictionCandidates는 축출 후보들을 반환합니다
func (mc *MemoryCache) GetEvictionCandidates(policy EvictionPolicy, count int) []string {
	mc.dataMutex.RLock()
	defer mc.dataMutex.RUnlock()
	
	switch policy {
	case EvictionLRU:
		return mc.getLRUCandidates(count)
	case EvictionLFU:
		return mc.getLFUCandidates(count)
	case EvictionFIFO:
		return mc.getFIFOCandidates(count)
	case EvictionTTL:
		return mc.getTTLCandidates(count)
	default:
		return mc.getLRUCandidates(count)
	}
}

// 내부 메서드들

func (mc *MemoryCache) isExpired(entry *MemoryCacheEntry) bool {
	if entry.TTL == 0 {
		return false
	}
	return time.Since(entry.CreatedAt) > entry.TTL
}

func (mc *MemoryCache) estimateSize(value interface{}) int64 {
	// 간단한 크기 추정 (실제로는 더 정확한 계산 필요)
	switch v := value.(type) {
	case string:
		return int64(len(v))
	case []byte:
		return int64(len(v))
	default:
		return 100 // 기본값
	}
}

func (mc *MemoryCache) updateLatency(latency time.Duration) {
	mc.statsMutex.Lock()
	defer mc.statsMutex.Unlock()
	
	mc.stats.AvgLatency = (mc.stats.AvgLatency + latency) / 2
}

func (mc *MemoryCache) updateAccessOrder(key string) {
	mc.accessMutex.Lock()
	defer mc.accessMutex.Unlock()
	
	// 기존 위치에서 제거
	for i, k := range mc.accessOrder {
		if k == key {
			mc.accessOrder = append(mc.accessOrder[:i], mc.accessOrder[i+1:]...)
			break
		}
	}
	
	// 맨 앞에 추가 (가장 최근 접근)
	mc.accessOrder = append([]string{key}, mc.accessOrder...)
}

func (mc *MemoryCache) removeFromAccessOrder(key string) {
	mc.accessMutex.Lock()
	defer mc.accessMutex.Unlock()
	
	for i, k := range mc.accessOrder {
		if k == key {
			mc.accessOrder = append(mc.accessOrder[:i], mc.accessOrder[i+1:]...)
			break
		}
	}
}

func (mc *MemoryCache) evictEntries(requiredSpace int64) error {
	var evictedSize int64
	evictionCount := 0
	maxEvictions := len(mc.data) / 2 // 최대 절반까지 축출
	
	candidates := mc.GetEvictionCandidates(mc.config.EvictionPolicy, maxEvictions)
	
	for _, key := range candidates {
		if evictedSize >= requiredSpace {
			break
		}
		
		if entry, exists := mc.data[key]; exists {
			evictedSize += entry.Size
			delete(mc.data, key)
			mc.removeFromAccessOrder(key)
			evictionCount++
		}
	}
	
	// 통계 업데이트
	atomic.AddInt64(&mc.stats.Evictions, int64(evictionCount))
	atomic.AddInt64(&mc.stats.Size, -evictedSize)
	
	if evictedSize < requiredSpace {
		return fmt.Errorf("insufficient space evicted: required %d, evicted %d", requiredSpace, evictedSize)
	}
	
	return nil
}

func (mc *MemoryCache) getLRUCandidates(count int) []string {
	mc.accessMutex.Lock()
	defer mc.accessMutex.Unlock()
	
	if len(mc.accessOrder) <= count {
		return append([]string(nil), mc.accessOrder...)
	}
	
	// 가장 오래된 것들부터 (뒤에서부터)
	start := len(mc.accessOrder) - count
	return append([]string(nil), mc.accessOrder[start:]...)
}

func (mc *MemoryCache) getLFUCandidates(count int) []string {
	type keyCount struct {
		key   string
		count int64
	}
	
	var candidates []keyCount
	for key, entry := range mc.data {
		candidates = append(candidates, keyCount{key, entry.AccessCount})
	}
	
	// 접근 횟수로 정렬 (낮은 순)
	for i := 0; i < len(candidates); i++ {
		for j := i + 1; j < len(candidates); j++ {
			if candidates[i].count > candidates[j].count {
				candidates[i], candidates[j] = candidates[j], candidates[i]
			}
		}
	}
	
	result := make([]string, 0, count)
	for i := 0; i < count && i < len(candidates); i++ {
		result = append(result, candidates[i].key)
	}
	
	return result
}

func (mc *MemoryCache) getFIFOCandidates(count int) []string {
	type keyTime struct {
		key  string
		time time.Time
	}
	
	var candidates []keyTime
	for key, entry := range mc.data {
		candidates = append(candidates, keyTime{key, entry.CreatedAt})
	}
	
	// 생성 시간으로 정렬 (오래된 순)
	for i := 0; i < len(candidates); i++ {
		for j := i + 1; j < len(candidates); j++ {
			if candidates[i].time.After(candidates[j].time) {
				candidates[i], candidates[j] = candidates[j], candidates[i]
			}
		}
	}
	
	result := make([]string, 0, count)
	for i := 0; i < count && i < len(candidates); i++ {
		result = append(result, candidates[i].key)
	}
	
	return result
}

func (mc *MemoryCache) getTTLCandidates(count int) []string {
	type keyTTL struct {
		key     string
		expires time.Time
	}
	
	var candidates []keyTTL
	for key, entry := range mc.data {
		if entry.TTL > 0 {
			expires := entry.CreatedAt.Add(entry.TTL)
			candidates = append(candidates, keyTTL{key, expires})
		}
	}
	
	// 만료 시간으로 정렬 (빨리 만료되는 순)
	for i := 0; i < len(candidates); i++ {
		for j := i + 1; j < len(candidates); j++ {
			if candidates[i].expires.After(candidates[j].expires) {
				candidates[i], candidates[j] = candidates[j], candidates[i]
			}
		}
	}
	
	result := make([]string, 0, count)
	for i := 0; i < count && i < len(candidates); i++ {
		result = append(result, candidates[i].key)
	}
	
	return result
}