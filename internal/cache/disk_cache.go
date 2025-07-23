package cache

import (
	"crypto/md5"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

// DiskCache는 디스크 기반 L2 캐시입니다
type DiskCache struct {
	// 설정
	config DiskCacheConfig
	
	// 메타데이터 인덱스
	index      map[string]*DiskCacheEntry
	indexMutex sync.RWMutex
	
	// 통계
	stats DiskCacheStats
	statsMutex sync.RWMutex
	
	// 파일 시스템 관리
	baseDir   string
	fileStats map[string]os.FileInfo
	fileMutex sync.RWMutex
	
	// 생명주기
	running atomic.Bool
}

// DiskCacheConfig는 디스크 캐시 설정입니다
type DiskCacheConfig struct {
	Directory      string        `json:"directory"`       // 캐시 디렉토리
	MaxSize        int64         `json:"max_size"`        // 최대 디스크 크기 (바이트)
	MaxEntries     int           `json:"max_entries"`     // 최대 파일 수
	DefaultTTL     time.Duration `json:"default_ttl"`     // 기본 TTL
	EvictionPolicy EvictionPolicy `json:"eviction_policy"` // 축출 정책
	SyncInterval   time.Duration `json:"sync_interval"`   // 동기화 간격
	Compression    bool          `json:"compression"`     // 압축 사용 여부
	Encryption     bool          `json:"encryption"`      // 암호화 사용 여부
}

// DiskCacheEntry는 디스크 캐시 엔트리 메타데이터입니다
type DiskCacheEntry struct {
	Key         string        `json:"key"`
	FileName    string        `json:"file_name"`
	TTL         time.Duration `json:"ttl"`
	CreatedAt   time.Time     `json:"created_at"`
	AccessedAt  time.Time     `json:"accessed_at"`
	ModifiedAt  time.Time     `json:"modified_at"`
	AccessCount int64         `json:"access_count"`
	Size        int64         `json:"size"`
	Checksum    string        `json:"checksum"`
	Compressed  bool          `json:"compressed"`
	Encrypted   bool          `json:"encrypted"`
}

// DiskCacheStats는 디스크 캐시 통계입니다
type DiskCacheStats struct {
	Hits          int64         `json:"hits"`
	Misses        int64         `json:"misses"`
	Sets          int64         `json:"sets"`
	Deletes       int64         `json:"deletes"`
	Evictions     int64         `json:"evictions"`
	Expired       int64         `json:"expired"`
	Size          int64         `json:"size"`
	Entries       int           `json:"entries"`
	HitRate       float64       `json:"hit_rate"`
	AvgLatency    time.Duration `json:"avg_latency"`
	LastAccess    time.Time     `json:"last_access"`
	DiskErrors    int64         `json:"disk_errors"`
	Corrupted     int64         `json:"corrupted"`
}

// CacheValue는 캐시된 값의 래퍼입니다
type CacheValue struct {
	Data      interface{} `json:"data"`
	Metadata  map[string]interface{} `json:"metadata"`
	CreatedAt time.Time   `json:"created_at"`
	TTL       time.Duration `json:"ttl"`
}

// NewDiskCache는 새로운 디스크 캐시를 생성합니다
func NewDiskCache(config DiskCacheConfig) (*DiskCache, error) {
	if config.Directory == "" {
		return nil, fmt.Errorf("cache directory is required")
	}
	
	if config.MaxSize <= 0 {
		return nil, fmt.Errorf("max size must be positive")
	}
	
	// 디렉토리 생성
	if err := os.MkdirAll(config.Directory, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}
	
	cache := &DiskCache{
		config:    config,
		index:     make(map[string]*DiskCacheEntry),
		baseDir:   config.Directory,
		fileStats: make(map[string]os.FileInfo),
		stats: DiskCacheStats{
			LastAccess: time.Now(),
		},
	}
	
	// 기존 캐시 파일들 로드
	if err := cache.loadExistingEntries(); err != nil {
		return nil, fmt.Errorf("failed to load existing entries: %w", err)
	}
	
	return cache, nil
}

// Start는 디스크 캐시를 시작합니다
func (dc *DiskCache) Start() error {
	if !dc.running.CompareAndSwap(false, true) {
		return fmt.Errorf("disk cache is already running")
	}
	
	// 인덱스 파일 저장
	if err := dc.saveIndex(); err != nil {
		return fmt.Errorf("failed to save index: %w", err)
	}
	
	return nil
}

// Stop은 디스크 캐시를 중지합니다
func (dc *DiskCache) Stop() error {
	if !dc.running.CompareAndSwap(true, false) {
		return nil
	}
	
	// 인덱스 저장
	return dc.saveIndex()
}

// Get은 키에 해당하는 값을 조회합니다
func (dc *DiskCache) Get(key string) (interface{}, bool) {
	start := time.Now()
	defer func() {
		dc.updateLatency(time.Since(start))
		dc.statsMutex.Lock()
		dc.stats.LastAccess = time.Now()
		dc.statsMutex.Unlock()
	}()
	
	dc.indexMutex.RLock()
	entry, exists := dc.index[key]
	dc.indexMutex.RUnlock()
	
	if !exists {
		atomic.AddInt64(&dc.stats.Misses, 1)
		return nil, false
	}
	
	// TTL 확인
	if dc.isExpired(entry) {
		dc.Delete(key) // 만료된 엔트리 삭제
		atomic.AddInt64(&dc.stats.Misses, 1)
		atomic.AddInt64(&dc.stats.Expired, 1)
		return nil, false
	}
	
	// 파일에서 데이터 읽기
	value, err := dc.readFromFile(entry)
	if err != nil {
		atomic.AddInt64(&dc.stats.DiskErrors, 1)
		atomic.AddInt64(&dc.stats.Misses, 1)
		return nil, false
	}
	
	// 접근 정보 업데이트
	dc.indexMutex.Lock()
	entry.AccessedAt = time.Now()
	atomic.AddInt64(&entry.AccessCount, 1)
	dc.indexMutex.Unlock()
	
	atomic.AddInt64(&dc.stats.Hits, 1)
	return value, true
}

// Set은 키-값 쌍을 설정합니다
func (dc *DiskCache) Set(key string, value interface{}, ttl time.Duration) error {
	if !dc.running.Load() {
		return fmt.Errorf("disk cache is not running")
	}
	
	if ttl == 0 {
		ttl = dc.config.DefaultTTL
	}
	
	// 파일명 생성
	fileName := dc.generateFileName(key)
	filePath := filepath.Join(dc.baseDir, fileName)
	
	// 데이터를 파일에 저장
	size, checksum, err := dc.writeToFile(filePath, value)
	if err != nil {
		atomic.AddInt64(&dc.stats.DiskErrors, 1)
		return fmt.Errorf("failed to write to file: %w", err)
	}
	
	now := time.Now()
	
	// 인덱스 엔트리 생성/업데이트
	dc.indexMutex.Lock()
	
	// 기존 엔트리가 있으면 크기 차이 계산
	var sizeDiff int64 = size
	if existingEntry, exists := dc.index[key]; exists {
		sizeDiff = size - existingEntry.Size
		// 기존 파일 삭제
		os.Remove(filepath.Join(dc.baseDir, existingEntry.FileName))
	}
	
	// 용량 확인 및 축출
	if dc.stats.Size+sizeDiff > dc.config.MaxSize || len(dc.index) >= dc.config.MaxEntries {
		if err := dc.evictEntries(sizeDiff); err != nil {
			dc.indexMutex.Unlock()
			os.Remove(filePath)
			return fmt.Errorf("failed to evict entries: %w", err)
		}
	}
	
	entry := &DiskCacheEntry{
		Key:         key,
		FileName:    fileName,
		TTL:         ttl,
		CreatedAt:   now,
		AccessedAt:  now,
		ModifiedAt:  now,
		AccessCount: 1,
		Size:        size,
		Checksum:    checksum,
		Compressed:  dc.config.Compression,
		Encrypted:   dc.config.Encryption,
	}
	
	dc.index[key] = entry
	dc.indexMutex.Unlock()
	
	// 통계 업데이트
	atomic.AddInt64(&dc.stats.Sets, 1)
	atomic.AddInt64(&dc.stats.Size, sizeDiff)
	
	dc.statsMutex.Lock()
	dc.stats.Entries = len(dc.index)
	dc.statsMutex.Unlock()
	
	return nil
}

// Delete는 키를 삭제합니다
func (dc *DiskCache) Delete(key string) error {
	dc.indexMutex.Lock()
	entry, exists := dc.index[key]
	if exists {
		delete(dc.index, key)
	}
	dc.indexMutex.Unlock()
	
	if !exists {
		return fmt.Errorf("key not found: %s", key)
	}
	
	// 파일 삭제
	filePath := filepath.Join(dc.baseDir, entry.FileName)
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		atomic.AddInt64(&dc.stats.DiskErrors, 1)
		return fmt.Errorf("failed to remove file: %w", err)
	}
	
	// 통계 업데이트
	atomic.AddInt64(&dc.stats.Deletes, 1)
	atomic.AddInt64(&dc.stats.Size, -entry.Size)
	
	dc.statsMutex.Lock()
	dc.stats.Entries = len(dc.index)
	dc.statsMutex.Unlock()
	
	return nil
}

// Clear는 모든 엔트리를 삭제합니다
func (dc *DiskCache) Clear() error {
	dc.indexMutex.Lock()
	defer dc.indexMutex.Unlock()
	
	// 모든 파일 삭제
	for _, entry := range dc.index {
		filePath := filepath.Join(dc.baseDir, entry.FileName)
		os.Remove(filePath)
	}
	
	// 인덱스 초기화
	dc.index = make(map[string]*DiskCacheEntry)
	
	// 통계 리셋
	dc.statsMutex.Lock()
	dc.stats.Size = 0
	dc.stats.Entries = 0
	dc.statsMutex.Unlock()
	
	return nil
}

// Size는 캐시 크기를 반환합니다
func (dc *DiskCache) Size() int64 {
	return atomic.LoadInt64(&dc.stats.Size)
}

// Keys는 모든 키를 반환합니다
func (dc *DiskCache) Keys() []string {
	dc.indexMutex.RLock()
	defer dc.indexMutex.RUnlock()
	
	keys := make([]string, 0, len(dc.index))
	for key, entry := range dc.index {
		if !dc.isExpired(entry) {
			keys = append(keys, key)
		}
	}
	
	return keys
}

// Stats는 캐시 통계를 반환합니다
func (dc *DiskCache) Stats() CacheStats {
	dc.statsMutex.RLock()
	defer dc.statsMutex.RUnlock()
	
	stats := CacheStats{
		Hits:      atomic.LoadInt64(&dc.stats.Hits),
		Misses:    atomic.LoadInt64(&dc.stats.Misses),
		Sets:      atomic.LoadInt64(&dc.stats.Sets),
		Deletes:   atomic.LoadInt64(&dc.stats.Deletes),
		Evictions: atomic.LoadInt64(&dc.stats.Evictions),
		Size:      atomic.LoadInt64(&dc.stats.Size),
		Entries:   dc.stats.Entries,
		AvgLatency: dc.stats.AvgLatency,
		LastAccess: dc.stats.LastAccess,
	}
	
	// 히트율 계산
	totalRequests := stats.Hits + stats.Misses
	if totalRequests > 0 {
		stats.HitRate = float64(stats.Hits) / float64(totalRequests)
	}
	
	return stats
}

// SetEvictionPolicy는 축출 정책을 설정합니다
func (dc *DiskCache) SetEvictionPolicy(policy EvictionPolicy) {
	dc.config.EvictionPolicy = policy
}

// SetMaxSize는 최대 크기를 설정합니다
func (dc *DiskCache) SetMaxSize(size int64) error {
	if size <= 0 {
		return fmt.Errorf("max size must be positive")
	}
	
	dc.config.MaxSize = size
	
	// 현재 크기가 새로운 최대값보다 크면 축출
	if dc.Size() > size {
		dc.indexMutex.Lock()
		dc.evictEntries(dc.Size() - size)
		dc.indexMutex.Unlock()
	}
	
	return nil
}

// CleanupExpired는 만료된 엔트리들을 정리합니다
func (dc *DiskCache) CleanupExpired() {
	dc.indexMutex.Lock()
	defer dc.indexMutex.Unlock()
	
	var expiredKeys []string
	var expiredSize int64
	
	for key, entry := range dc.index {
		if dc.isExpired(entry) {
			expiredKeys = append(expiredKeys, key)
			expiredSize += entry.Size
		}
	}
	
	// 만료된 엔트리 삭제
	for _, key := range expiredKeys {
		entry := dc.index[key]
		delete(dc.index, key)
		
		// 파일 삭제
		filePath := filepath.Join(dc.baseDir, entry.FileName)
		os.Remove(filePath)
	}
	
	// 통계 업데이트
	atomic.AddInt64(&dc.stats.Expired, int64(len(expiredKeys)))
	atomic.AddInt64(&dc.stats.Size, -expiredSize)
	
	dc.statsMutex.Lock()
	dc.stats.Entries = len(dc.index)
	dc.statsMutex.Unlock()
}

// GetEvictionCandidates는 축출 후보들을 반환합니다
func (dc *DiskCache) GetEvictionCandidates(policy EvictionPolicy, count int) []string {
	dc.indexMutex.RLock()
	defer dc.indexMutex.RUnlock()
	
	switch policy {
	case EvictionLRU:
		return dc.getLRUCandidates(count)
	case EvictionLFU:
		return dc.getLFUCandidates(count)
	case EvictionFIFO:
		return dc.getFIFOCandidates(count)
	case EvictionTTL:
		return dc.getTTLCandidates(count)
	default:
		return dc.getLRUCandidates(count)
	}
}

// 내부 메서드들

func (dc *DiskCache) isExpired(entry *DiskCacheEntry) bool {
	if entry.TTL == 0 {
		return false
	}
	return time.Since(entry.CreatedAt) > entry.TTL
}

func (dc *DiskCache) generateFileName(key string) string {
	hash := md5.Sum([]byte(key))
	return fmt.Sprintf("%x.cache", hash)
}

func (dc *DiskCache) readFromFile(entry *DiskCacheEntry) (interface{}, error) {
	filePath := filepath.Join(dc.baseDir, entry.FileName)
	
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()
	
	var cacheValue CacheValue
	decoder := gob.NewDecoder(file)
	if err := decoder.Decode(&cacheValue); err != nil {
		atomic.AddInt64(&dc.stats.Corrupted, 1)
		return nil, fmt.Errorf("failed to decode cache value: %w", err)
	}
	
	// 체크섬 검증 (간단화)
	// 실제 구현에서는 파일 내용의 체크섬을 계산하여 검증
	
	return cacheValue.Data, nil
}

func (dc *DiskCache) writeToFile(filePath string, value interface{}) (int64, string, error) {
	cacheValue := CacheValue{
		Data:      value,
		Metadata:  make(map[string]interface{}),
		CreatedAt: time.Now(),
		TTL:       dc.config.DefaultTTL,
	}
	
	file, err := os.Create(filePath)
	if err != nil {
		return 0, "", fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()
	
	encoder := gob.NewEncoder(file)
	if err := encoder.Encode(cacheValue); err != nil {
		return 0, "", fmt.Errorf("failed to encode cache value: %w", err)
	}
	
	// 파일 정보 가져오기
	fileInfo, err := file.Stat()
	if err != nil {
		return 0, "", fmt.Errorf("failed to get file info: %w", err)
	}
	
	// 체크섬 계산 (간단화)
	checksum := fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%v", value))))
	
	return fileInfo.Size(), checksum, nil
}

func (dc *DiskCache) updateLatency(latency time.Duration) {
	dc.statsMutex.Lock()
	defer dc.statsMutex.Unlock()
	
	dc.stats.AvgLatency = (dc.stats.AvgLatency + latency) / 2
}

func (dc *DiskCache) evictEntries(requiredSpace int64) error {
	var evictedSize int64
	evictionCount := 0
	maxEvictions := len(dc.index) / 2 // 최대 절반까지 축출
	
	candidates := dc.GetEvictionCandidates(dc.config.EvictionPolicy, maxEvictions)
	
	for _, key := range candidates {
		if evictedSize >= requiredSpace {
			break
		}
		
		if entry, exists := dc.index[key]; exists {
			// 파일 삭제
			filePath := filepath.Join(dc.baseDir, entry.FileName)
			os.Remove(filePath)
			
			evictedSize += entry.Size
			delete(dc.index, key)
			evictionCount++
		}
	}
	
	// 통계 업데이트
	atomic.AddInt64(&dc.stats.Evictions, int64(evictionCount))
	atomic.AddInt64(&dc.stats.Size, -evictedSize)
	
	if evictedSize < requiredSpace {
		return fmt.Errorf("insufficient space evicted: required %d, evicted %d", requiredSpace, evictedSize)
	}
	
	return nil
}

func (dc *DiskCache) loadExistingEntries() error {
	indexPath := filepath.Join(dc.baseDir, "index.json")
	
	// 인덱스 파일이 존재하지 않으면 빈 인덱스로 시작
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		return nil
	}
	
	// 인덱스 파일 읽기
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return fmt.Errorf("failed to read index file: %w", err)
	}
	
	var index map[string]*DiskCacheEntry
	if err := json.Unmarshal(data, &index); err != nil {
		return fmt.Errorf("failed to unmarshal index: %w", err)
	}
	
	// 실제 파일 존재 여부 확인
	var totalSize int64
	validEntries := make(map[string]*DiskCacheEntry)
	
	for key, entry := range index {
		filePath := filepath.Join(dc.baseDir, entry.FileName)
		if _, err := os.Stat(filePath); err == nil {
			validEntries[key] = entry
			totalSize += entry.Size
		}
	}
	
	dc.index = validEntries
	atomic.StoreInt64(&dc.stats.Size, totalSize)
	
	dc.statsMutex.Lock()
	dc.stats.Entries = len(dc.index)
	dc.statsMutex.Unlock()
	
	return nil
}

func (dc *DiskCache) saveIndex() error {
	indexPath := filepath.Join(dc.baseDir, "index.json")
	
	dc.indexMutex.RLock()
	data, err := json.MarshalIndent(dc.index, "", "  ")
	dc.indexMutex.RUnlock()
	
	if err != nil {
		return fmt.Errorf("failed to marshal index: %w", err)
	}
	
	if err := os.WriteFile(indexPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write index file: %w", err)
	}
	
	return nil
}

func (dc *DiskCache) getLRUCandidates(count int) []string {
	type keyTime struct {
		key  string
		time time.Time
	}
	
	var candidates []keyTime
	for key, entry := range dc.index {
		candidates = append(candidates, keyTime{key, entry.AccessedAt})
	}
	
	// 접근 시간으로 정렬 (오래된 순)
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

func (dc *DiskCache) getLFUCandidates(count int) []string {
	type keyCount struct {
		key   string
		count int64
	}
	
	var candidates []keyCount
	for key, entry := range dc.index {
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

func (dc *DiskCache) getFIFOCandidates(count int) []string {
	type keyTime struct {
		key  string
		time time.Time
	}
	
	var candidates []keyTime
	for key, entry := range dc.index {
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

func (dc *DiskCache) getTTLCandidates(count int) []string {
	type keyTTL struct {
		key     string
		expires time.Time
	}
	
	var candidates []keyTTL
	for key, entry := range dc.index {
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