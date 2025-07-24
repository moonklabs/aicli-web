package cache

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Cache 캐시 인터페이스
type Cache interface {
	// Get 캐시에서 값 조회
	Get(ctx context.Context, key string, dest interface{}) error
	
	// Set 캐시에 값 저장
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	
	// Delete 캐시에서 값 삭제
	Delete(ctx context.Context, key string) error
	
	// Clear 모든 캐시 삭제
	Clear(ctx context.Context) error
	
	// Exists 키 존재 여부 확인
	Exists(ctx context.Context, key string) (bool, error)
	
	// Stats 캐시 통계 반환
	Stats() CacheStats
	
	// Close 캐시 종료
	Close() error
}

// CacheStats 캐시 통계
type CacheStats struct {
	Hits        int64   `json:"hits"`
	Misses      int64   `json:"misses"`
	HitRatio    float64 `json:"hit_ratio"`
	ItemCount   int64   `json:"item_count"`
	TotalSize   int64   `json:"total_size"`
	Evictions   int64   `json:"evictions"`
}

// CacheEntry 캐시 엔트리
type CacheEntry struct {
	Key       string      `json:"key"`
	Value     interface{} `json:"value"`
	Data      []byte      `json:"data"`
	CreatedAt time.Time   `json:"created_at"`
	ExpiresAt *time.Time  `json:"expires_at,omitempty"`
	AccessCount int64     `json:"access_count"`
	LastAccess  time.Time `json:"last_access"`
	Size        int64     `json:"size"`
}

// IsExpired 만료 여부 확인
func (ce *CacheEntry) IsExpired() bool {
	if ce.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*ce.ExpiresAt)
}

// MemoryCache 인메모리 캐시 구현
type MemoryCache struct {
	items       map[string]*CacheEntry
	mu          sync.RWMutex
	logger      *zap.Logger
	
	// 통계
	stats CacheStats
	
	// 설정
	maxSize     int64         // 최대 크기 (바이트)
	maxItems    int64         // 최대 항목 수
	defaultTTL  time.Duration // 기본 TTL
	cleanupInterval time.Duration // 정리 간격
	
	// 정리 고루틴 제어
	cleanupTicker *time.Ticker
	stopCleanup   chan struct{}
	cleanupOnce   sync.Once
}

// MemoryCacheConfig 인메모리 캐시 설정
type MemoryCacheConfig struct {
	MaxSize         int64         // 최대 크기 (바이트)
	MaxItems        int64         // 최대 항목 수
	DefaultTTL      time.Duration // 기본 TTL
	CleanupInterval time.Duration // 정리 간격
	Logger          *zap.Logger   // 로거
}

// DefaultMemoryCacheConfig 기본 인메모리 캐시 설정
func DefaultMemoryCacheConfig() MemoryCacheConfig {
	return MemoryCacheConfig{
		MaxSize:         64 * 1024 * 1024, // 64MB
		MaxItems:        10000,             // 10,000개 항목
		DefaultTTL:      5 * time.Minute,   // 5분
		CleanupInterval: time.Minute,       // 1분마다 정리
		Logger:          zap.NewNop(),
	}
}

// NewMemoryCache 새 인메모리 캐시 생성
func NewMemoryCache(config MemoryCacheConfig) *MemoryCache {
	if config.Logger == nil {
		config.Logger = zap.NewNop()
	}
	
	cache := &MemoryCache{
		items:           make(map[string]*CacheEntry),
		logger:          config.Logger,
		maxSize:         config.MaxSize,
		maxItems:        config.MaxItems,
		defaultTTL:      config.DefaultTTL,
		cleanupInterval: config.CleanupInterval,
		stopCleanup:     make(chan struct{}),
	}
	
	// 정리 고루틴 시작
	cache.startCleanup()
	
	return cache
}

// startCleanup 정리 고루틴 시작
func (mc *MemoryCache) startCleanup() {
	mc.cleanupTicker = time.NewTicker(mc.cleanupInterval)
	
	go func() {
		for {
			select {
			case <-mc.cleanupTicker.C:
				mc.cleanup()
			case <-mc.stopCleanup:
				mc.cleanupTicker.Stop()
				return
			}
		}
	}()
}

// cleanup 만료된 항목 정리
func (mc *MemoryCache) cleanup() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	// now := time.Now() // 사용하지 않음
	var expiredKeys []string
	var freedSize int64
	
	for key, entry := range mc.items {
		if entry.IsExpired() {
			expiredKeys = append(expiredKeys, key)
			freedSize += entry.Size
		}
	}
	
	for _, key := range expiredKeys {
		delete(mc.items, key)
		mc.stats.Evictions++
	}
	
	mc.stats.ItemCount = int64(len(mc.items))
	mc.stats.TotalSize -= freedSize
	
	if len(expiredKeys) > 0 {
		mc.logger.Debug("만료된 캐시 항목 정리 완료",
			zap.Int("expired_count", len(expiredKeys)),
			zap.Int64("freed_size", freedSize),
			zap.Int64("remaining_items", mc.stats.ItemCount),
		)
	}
}

// evictLRU LRU 방식으로 항목 제거
func (mc *MemoryCache) evictLRU(needed int64) {
	if len(mc.items) == 0 {
		return
	}
	
	// 접근 시간 기준으로 가장 오래된 항목들 찾기
	type keyTime struct {
		key  string
		time time.Time
	}
	
	var candidates []keyTime
	for key, entry := range mc.items {
		candidates = append(candidates, keyTime{key, entry.LastAccess})
	}
	
	// 접근 시간 순으로 정렬
	for i := 0; i < len(candidates); i++ {
		for j := i + 1; j < len(candidates); j++ {
			if candidates[i].time.After(candidates[j].time) {
				candidates[i], candidates[j] = candidates[j], candidates[i]
			}
		}
	}
	
	// 필요한 만큼 제거
	var freedSize int64
	var evictedCount int
	
	for _, candidate := range candidates {
		if freedSize >= needed && mc.stats.ItemCount < mc.maxItems {
			break
		}
		
		if entry, exists := mc.items[candidate.key]; exists {
			delete(mc.items, candidate.key)
			freedSize += entry.Size
			evictedCount++
			mc.stats.Evictions++
		}
	}
	
	mc.stats.ItemCount = int64(len(mc.items))
	mc.stats.TotalSize -= freedSize
	
	if evictedCount > 0 {
		mc.logger.Debug("LRU 방식으로 캐시 항목 제거",
			zap.Int("evicted_count", evictedCount),
			zap.Int64("freed_size", freedSize),
		)
	}
}

// Get 캐시에서 값 조회
func (mc *MemoryCache) Get(ctx context.Context, key string, dest interface{}) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	entry, exists := mc.items[key]
	if !exists {
		mc.stats.Misses++
		return ErrCacheMiss
	}
	
	// 만료 확인
	if entry.IsExpired() {
		delete(mc.items, key)
		mc.stats.Misses++
		mc.stats.Evictions++
		mc.stats.ItemCount--
		mc.stats.TotalSize -= entry.Size
		return ErrCacheMiss
	}
	
	// 통계 업데이트
	mc.stats.Hits++
	entry.AccessCount++
	entry.LastAccess = time.Now()
	
	// 역직렬화
	if err := json.Unmarshal(entry.Data, dest); err != nil {
		mc.logger.Error("캐시 값 역직렬화 실패",
			zap.String("key", key),
			zap.Error(err),
		)
		return fmt.Errorf("캐시 값 역직렬화 실패: %w", err)
	}
	
	return nil
}

// Set 캐시에 값 저장
func (mc *MemoryCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if ttl <= 0 {
		ttl = mc.defaultTTL
	}
	
	// 직렬화
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("캐시 값 직렬화 실패: %w", err)
	}
	
	size := int64(len(key)) + int64(len(data)) + 200 // 메타데이터 추가 크기
	
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	// 기존 항목이 있으면 크기에서 제외
	if existing, exists := mc.items[key]; exists {
		mc.stats.TotalSize -= existing.Size
		mc.stats.ItemCount--
	}
	
	// 공간 확보 필요 시 LRU 제거
	if mc.stats.TotalSize+size > mc.maxSize || mc.stats.ItemCount >= mc.maxItems {
		mc.evictLRU(size)
	}
	
	// 새 엔트리 생성
	entry := &CacheEntry{
		Key:         key,
		Value:       value,
		Data:        data,
		CreatedAt:   time.Now(),
		AccessCount: 0,
		LastAccess:  time.Now(),
		Size:        size,
	}
	
	if ttl > 0 {
		expiresAt := time.Now().Add(ttl)
		entry.ExpiresAt = &expiresAt
	}
	
	// 저장
	mc.items[key] = entry
	mc.stats.ItemCount = int64(len(mc.items))
	mc.stats.TotalSize += size
	
	mc.logger.Debug("캐시에 값 저장",
		zap.String("key", key),
		zap.Int64("size", size),
		zap.Duration("ttl", ttl),
	)
	
	return nil
}

// Delete 캐시에서 값 삭제
func (mc *MemoryCache) Delete(ctx context.Context, key string) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	if entry, exists := mc.items[key]; exists {
		delete(mc.items, key)
		mc.stats.ItemCount--
		mc.stats.TotalSize -= entry.Size
		
		mc.logger.Debug("캐시에서 값 삭제",
			zap.String("key", key),
			zap.Int64("size", entry.Size),
		)
	}
	
	return nil
}

// Clear 모든 캐시 삭제
func (mc *MemoryCache) Clear(ctx context.Context) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	itemCount := len(mc.items)
	totalSize := mc.stats.TotalSize
	
	mc.items = make(map[string]*CacheEntry)
	mc.stats.ItemCount = 0
	mc.stats.TotalSize = 0
	
	mc.logger.Info("모든 캐시 삭제",
		zap.Int("deleted_items", itemCount),
		zap.Int64("freed_size", totalSize),
	)
	
	return nil
}

// Exists 키 존재 여부 확인
func (mc *MemoryCache) Exists(ctx context.Context, key string) (bool, error) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	entry, exists := mc.items[key]
	if !exists {
		return false, nil
	}
	
	// 만료 확인
	if entry.IsExpired() {
		return false, nil
	}
	
	return true, nil
}

// Stats 캐시 통계 반환
func (mc *MemoryCache) Stats() CacheStats {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	stats := mc.stats
	total := stats.Hits + stats.Misses
	if total > 0 {
		stats.HitRatio = float64(stats.Hits) / float64(total)
	}
	
	return stats
}

// Close 캐시 종료
func (mc *MemoryCache) Close() error {
	mc.cleanupOnce.Do(func() {
		close(mc.stopCleanup)
	})
	
	mc.logger.Info("인메모리 캐시 종료")
	return nil
}

// 에러 정의
var (
	ErrCacheMiss = fmt.Errorf("cache miss")
	ErrCacheExpired = fmt.Errorf("cache expired")
)

// CacheKey 캐시 키 생성 유틸리티
type CacheKey struct {
	prefix string
}

// NewCacheKey 새 캐시 키 생성기
func NewCacheKey(prefix string) *CacheKey {
	return &CacheKey{prefix: prefix}
}

// Generate 캐시 키 생성
func (ck *CacheKey) Generate(components ...interface{}) string {
	key := ck.prefix
	
	for _, component := range components {
		switch v := component.(type) {
		case string:
			key += ":" + v
		case int, int32, int64:
			key += ":" + fmt.Sprintf("%d", v)
		case float32, float64:
			key += ":" + fmt.Sprintf("%.2f", v)
		default:
			// 복잡한 객체는 JSON 해시로 변환
			data, _ := json.Marshal(v)
			hash := fmt.Sprintf("%x", md5.Sum(data))
			key += ":" + hash
		}
	}
	
	return key
}

// WorkspaceKey 워크스페이스 캐시 키 생성
func (ck *CacheKey) WorkspaceKey(id string) string {
	return ck.Generate("workspace", id)
}

// WorkspaceListKey 워크스페이스 목록 캐시 키 생성
func (ck *CacheKey) WorkspaceListKey(ownerID string, page, limit int) string {
	return ck.Generate("workspace_list", ownerID, page, limit)
}

// ProjectKey 프로젝트 캐시 키 생성
func (ck *CacheKey) ProjectKey(id string) string {
	return ck.Generate("project", id)
}

// ProjectListKey 프로젝트 목록 캐시 키 생성
func (ck *CacheKey) ProjectListKey(workspaceID string, page, limit int) string {
	return ck.Generate("project_list", workspaceID, page, limit)
}

// SessionKey 세션 캐시 키 생성
func (ck *CacheKey) SessionKey(id string) string {
	return ck.Generate("session", id)
}

// TaskKey 태스크 캐시 키 생성
func (ck *CacheKey) TaskKey(id string) string {
	return ck.Generate("task", id)
}

// QueryKey 쿼리 결과 캐시 키 생성
func (ck *CacheKey) QueryKey(query string, params ...interface{}) string {
	components := []interface{}{query}
	components = append(components, params...)
	return ck.Generate(components...)
}