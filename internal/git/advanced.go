package git

import (
	"container/list"
	"context"
	"fmt"
	"sync"
	"time"
)

// AdvancedWorktreeManager 고급 기능을 포함한 WorktreeManager
type AdvancedWorktreeManager struct {
	// 기본 매니저
	base WorktreeManager

	// 동시성 제어
	concurrencyLimit int
	semaphore        chan struct{}
	mu               sync.RWMutex

	// LRU 캐시
	cache       map[string]*cacheEntry
	cacheList   *list.List
	cacheSize   int
	maxCacheSize int

	// GC 설정
	gcInterval      time.Duration
	maxAge          time.Duration
	cleanupRunning  bool
	cleanupStop     chan struct{}

	// 메트릭
	metrics *WorktreeMetrics
	promMetrics *PrometheusMetrics
}

// cacheEntry LRU 캐시 엔트리
type cacheEntry struct {
	worktree  *Worktree
	element   *list.Element
	lastUsed  time.Time
}

// WorktreeMetrics 성능 메트릭
type WorktreeMetrics struct {
	mu sync.RWMutex

	// 카운터
	TotalCreated    int64
	TotalDeleted    int64
	CacheHits       int64
	CacheMisses     int64
	CloneOperations int64
	
	// 타이밍
	CloneDurations   []time.Duration
	CreateDurations  []time.Duration
	
	// 리소스
	ActiveWorktrees  int
	CachedWorktrees  int
	DiskUsageBytes   int64
}

// AdvancedOptions 고급 워크트리 매니저 옵션
type AdvancedOptions struct {
	// 동시성 제한 (기본값: 5)
	ConcurrencyLimit int
	
	// LRU 캐시 크기 (기본값: 100)
	MaxCacheSize int
	
	// GC 간격 (기본값: 1시간)
	GCInterval time.Duration
	
	// 최대 worktree 나이 (기본값: 30일)
	MaxAge time.Duration
	
	// Prometheus 메트릭 비활성화 (테스트용)
	DisablePrometheus bool
}

// NewAdvancedWorktreeManager 고급 워크트리 매니저 생성
func NewAdvancedWorktreeManager(base WorktreeManager, opts AdvancedOptions) *AdvancedWorktreeManager {
	// 기본값 설정
	if opts.ConcurrencyLimit <= 0 {
		opts.ConcurrencyLimit = 5
	}
	if opts.MaxCacheSize <= 0 {
		opts.MaxCacheSize = 100
	}
	if opts.GCInterval <= 0 {
		opts.GCInterval = 1 * time.Hour
	}
	if opts.MaxAge <= 0 {
		opts.MaxAge = 30 * 24 * time.Hour
	}

	manager := &AdvancedWorktreeManager{
		base:             base,
		concurrencyLimit: opts.ConcurrencyLimit,
		semaphore:        make(chan struct{}, opts.ConcurrencyLimit),
		cache:            make(map[string]*cacheEntry),
		cacheList:        list.New(),
		maxCacheSize:     opts.MaxCacheSize,
		gcInterval:       opts.GCInterval,
		maxAge:           opts.MaxAge,
		cleanupStop:      make(chan struct{}),
		metrics:          &WorktreeMetrics{},
	}
	
	// Prometheus 메트릭 초기화 (비활성화 옵션 확인)
	if !opts.DisablePrometheus {
		manager.promMetrics = NewPrometheusMetrics()
	}

	// GC 시작
	go manager.startGarbageCollector()

	return manager
}

// Clone 저장소 복제 (동시성 제어 포함)
func (m *AdvancedWorktreeManager) Clone(ctx context.Context, url, path string, opts CloneOptions) (*Repository, error) {
	// 세마포어로 동시 실행 제한
	select {
	case m.semaphore <- struct{}{}:
		defer func() { <-m.semaphore }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	start := time.Now()
	defer func() {
		m.recordCloneDuration(time.Since(start))
		m.incrementCloneOperations()
	}()

	// Shallow clone 지원
	if opts.Depth == 0 && m.shouldUseShallowClone(url) {
		opts.Depth = 1
	}

	return m.base.Clone(ctx, url, path, opts)
}

// CreateWorktree Worktree 생성
func (m *AdvancedWorktreeManager) CreateWorktree(ctx context.Context, repo *Repository, name string, opts WorktreeOptions) (*Worktree, error) {
	// 세마포어로 동시 실행 제한
	select {
	case m.semaphore <- struct{}{}:
		defer func() { <-m.semaphore }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	start := time.Now()
	defer func() {
		m.recordCreateDuration(time.Since(start))
	}()

	// 기본 매니저로 생성
	worktree, err := m.base.CreateWorktree(ctx, repo, name, opts)
	if err != nil {
		return nil, err
	}

	// 캐시에 추가 (조회용)
	cacheKey := fmt.Sprintf("%s-%s", repo.ID, name)
	m.addToCache(cacheKey, worktree)
	m.incrementTotalCreated()

	return worktree, nil
}

// GetWorktree 캐시에서 Worktree 조회
func (m *AdvancedWorktreeManager) GetWorktree(repo *Repository, name string) *Worktree {
	cacheKey := fmt.Sprintf("%s-%s", repo.ID, name)
	return m.getFromCache(cacheKey)
}

// RemoveWorktree Worktree 삭제 (캐시에서도 제거)
func (m *AdvancedWorktreeManager) RemoveWorktree(ctx context.Context, repo *Repository, name string) error {
	// 캐시에서 제거
	cacheKey := fmt.Sprintf("%s-%s", repo.ID, name)
	m.removeFromCache(cacheKey)

	// 기본 매니저로 삭제
	err := m.base.RemoveWorktree(ctx, repo, name)
	if err == nil {
		m.incrementTotalDeleted()
	}

	return err
}

// ListWorktrees Worktree 목록 조회
func (m *AdvancedWorktreeManager) ListWorktrees(ctx context.Context, repo *Repository) ([]*Worktree, error) {
	return m.base.ListWorktrees(ctx, repo)
}

// CreateBranch 브랜치 생성 및 체크아웃
func (m *AdvancedWorktreeManager) CreateBranch(ctx context.Context, worktree *Worktree, branchName string, baseBranch string) error {
	return m.base.CreateBranch(ctx, worktree, branchName, baseBranch)
}

// ListBranches 브랜치 목록 조회
func (m *AdvancedWorktreeManager) ListBranches(ctx context.Context, repo *Repository) ([]Branch, error) {
	return m.base.ListBranches(ctx, repo)
}

// GetStatus 상태 확인
func (m *AdvancedWorktreeManager) GetStatus(ctx context.Context, worktree *Worktree) (*Status, error) {
	return m.base.GetStatus(ctx, worktree)
}

// Cleanup 정리 작업
func (m *AdvancedWorktreeManager) Cleanup(ctx context.Context, repo *Repository) error {
	// 캐시에서 해당 저장소의 모든 엔트리 제거
	m.mu.Lock()
	for key := range m.cache {
		if len(key) > len(repo.ID) && key[:len(repo.ID)] == repo.ID {
			delete(m.cache, key)
		}
	}
	m.mu.Unlock()

	return m.base.Cleanup(ctx, repo)
}

// Stop 고급 매니저 종료
func (m *AdvancedWorktreeManager) Stop() {
	close(m.cleanupStop)
}

// GetMetrics 메트릭 조회
func (m *AdvancedWorktreeManager) GetMetrics() WorktreeMetrics {
	m.metrics.mu.RLock()
	defer m.metrics.mu.RUnlock()
	
	// 복사본 반환
	metrics := *m.metrics
	metrics.ActiveWorktrees = m.getActiveWorktreeCount()
	metrics.CachedWorktrees = len(m.cache)
	
	return metrics
}

// LRU 캐시 관리 메서드들

func (m *AdvancedWorktreeManager) getFromCache(key string) *Worktree {
	m.mu.Lock()
	defer m.mu.Unlock()

	if entry, exists := m.cache[key]; exists {
		// LRU 리스트에서 맨 앞으로 이동
		m.cacheList.MoveToFront(entry.element)
		entry.lastUsed = time.Now()
		m.recordCacheHit()
		return entry.worktree
	}

	return nil
}

func (m *AdvancedWorktreeManager) addToCache(key string, worktree *Worktree) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 이미 캐시에 있으면 업데이트
	if entry, exists := m.cache[key]; exists {
		entry.worktree = worktree
		entry.lastUsed = time.Now()
		m.cacheList.MoveToFront(entry.element)
		return
	}

	// 캐시 크기 제한 확인
	if len(m.cache) >= m.maxCacheSize {
		// 가장 오래된 엔트리 제거
		oldest := m.cacheList.Back()
		if oldest != nil {
			oldKey := oldest.Value.(string)
			delete(m.cache, oldKey)
			m.cacheList.Remove(oldest)
		}
	}

	// 새 엔트리 추가
	element := m.cacheList.PushFront(key)
	m.cache[key] = &cacheEntry{
		worktree: worktree,
		element:  element,
		lastUsed: time.Now(),
	}
}

func (m *AdvancedWorktreeManager) removeFromCache(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if entry, exists := m.cache[key]; exists {
		m.cacheList.Remove(entry.element)
		delete(m.cache, key)
	}
}

// 가비지 컬렉터

func (m *AdvancedWorktreeManager) startGarbageCollector() {
	ticker := time.NewTicker(m.gcInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.runGarbageCollection()
		case <-m.cleanupStop:
			return
		}
	}
}

func (m *AdvancedWorktreeManager) runGarbageCollection() {
	m.mu.Lock()
	if m.cleanupRunning {
		m.mu.Unlock()
		return
	}
	m.cleanupRunning = true
	m.mu.Unlock()

	defer func() {
		m.mu.Lock()
		m.cleanupRunning = false
		m.mu.Unlock()
	}()

	start := time.Now()
	deletedCount := 0

	// 캐시 정리
	m.cleanupCache()

	// TODO: 오래된 worktree 정리 구현
	// deletedCount += m.cleanupOldWorktrees()

	// GC 메트릭 기록
	if m.promMetrics != nil {
		m.promMetrics.RecordGCRun(time.Since(start), deletedCount)
	}
}

func (m *AdvancedWorktreeManager) cleanupCache() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	var toRemove []string

	for key, entry := range m.cache {
		if now.Sub(entry.lastUsed) > m.maxAge {
			toRemove = append(toRemove, key)
		}
	}

	for _, key := range toRemove {
		if entry, exists := m.cache[key]; exists {
			if entry.element != nil {
				m.cacheList.Remove(entry.element)
			}
			delete(m.cache, key)
		}
	}
}

// 유틸리티 메서드들

func (m *AdvancedWorktreeManager) shouldUseShallowClone(url string) bool {
	// TODO: URL이나 저장소 크기를 기반으로 shallow clone 여부 결정
	return true
}

func (m *AdvancedWorktreeManager) getActiveWorktreeCount() int {
	return m.countActiveWorktrees()
}

// 메트릭 기록 메서드들

func (m *AdvancedWorktreeManager) recordCloneDuration(duration time.Duration) {
	m.metrics.mu.Lock()
	defer m.metrics.mu.Unlock()
	m.metrics.CloneDurations = append(m.metrics.CloneDurations, duration)
	
	// Prometheus 메트릭 기록
	if m.promMetrics != nil {
		m.promMetrics.RecordCloneOperation(duration)
	}
}

func (m *AdvancedWorktreeManager) recordCreateDuration(duration time.Duration) {
	m.metrics.mu.Lock()
	defer m.metrics.mu.Unlock()
	m.metrics.CreateDurations = append(m.metrics.CreateDurations, duration)
	
	// Prometheus 메트릭 기록
	if m.promMetrics != nil {
		m.promMetrics.RecordWorktreeCreate(duration)
	}
}

func (m *AdvancedWorktreeManager) incrementCloneOperations() {
	m.metrics.mu.Lock()
	defer m.metrics.mu.Unlock()
	m.metrics.CloneOperations++
}

func (m *AdvancedWorktreeManager) incrementTotalCreated() {
	m.metrics.mu.Lock()
	defer m.metrics.mu.Unlock()
	m.metrics.TotalCreated++
	
	// Prometheus 메트릭 기록
	if m.promMetrics != nil {
		m.promMetrics.RecordWorktreeCreated()
	}
}

func (m *AdvancedWorktreeManager) incrementTotalDeleted() {
	m.metrics.mu.Lock()
	defer m.metrics.mu.Unlock()
	m.metrics.TotalDeleted++
	
	// Prometheus 메트릭 기록
	if m.promMetrics != nil {
		m.promMetrics.RecordWorktreeDeleted()
	}
}

func (m *AdvancedWorktreeManager) recordCacheHit() {
	m.metrics.mu.Lock()
	defer m.metrics.mu.Unlock()
	m.metrics.CacheHits++
	
	// Prometheus 메트릭 기록
	if m.promMetrics != nil {
		m.promMetrics.RecordCacheHit()
	}
}

func (m *AdvancedWorktreeManager) recordCacheMiss() {
	m.metrics.mu.Lock()
	defer m.metrics.mu.Unlock()
	m.metrics.CacheMisses++
	
	// Prometheus 메트릭 기록
	if m.promMetrics != nil {
		m.promMetrics.RecordCacheMiss()
	}
}