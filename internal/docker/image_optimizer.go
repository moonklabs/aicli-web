package docker

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
)

// ImageOptimizer는 Docker 이미지 최적화 및 캐싱을 담당합니다
type ImageOptimizer struct {
	// Docker 클라이언트
	client *client.Client

	// 설정
	config OptimizerConfig

	// 캐시 관리
	cache        *ImageCache
	layerCache   *LayerCache
	buildCache   *BuildCache

	// 최적화 전략
	strategies   []OptimizationStrategy
	
	// 통계
	stats        *OptimizerStats
	statsMu      sync.RWMutex

	// 생명주기
	running      atomic.Bool
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
}

// OptimizerConfig는 최적화 설정입니다
type OptimizerConfig struct {
	// 캐시 설정
	EnableLayerCache     bool          `json:"enable_layer_cache"`
	EnableBuildCache     bool          `json:"enable_build_cache"`
	EnableImageCache     bool          `json:"enable_image_cache"`
	CacheDir            string        `json:"cache_dir"`
	MaxCacheSize        int64         `json:"max_cache_size"`
	CacheRetention      time.Duration `json:"cache_retention"`

	// 최적화 설정
	EnableMultiStage    bool    `json:"enable_multi_stage"`
	EnableCompression   bool    `json:"enable_compression"`
	CompressionLevel    int     `json:"compression_level"`
	EnableMinification  bool    `json:"enable_minification"`
	OptimizationLevel   int     `json:"optimization_level"`

	// 빌드 설정
	MaxConcurrentBuilds int           `json:"max_concurrent_builds"`
	BuildTimeout        time.Duration `json:"build_timeout"`
	EnableParallelBuild bool          `json:"enable_parallel_build"`
	BuildMemoryLimit    int64         `json:"build_memory_limit"`

	// 네트워킹
	RegistryMirrors     []string      `json:"registry_mirrors"`
	PullTimeout         time.Duration `json:"pull_timeout"`
	EnablePullOptimization bool       `json:"enable_pull_optimization"`

	// 정리 설정
	AutoCleanup         bool          `json:"auto_cleanup"`
	CleanupInterval     time.Duration `json:"cleanup_interval"`
	MaxImageAge         time.Duration `json:"max_image_age"`
}

// ImageCache는 이미지 캐시를 관리합니다
type ImageCache struct {
	// 캐시 저장소
	entries      map[string]*CacheEntry
	entriesMu    sync.RWMutex

	// 캐시 통계
	stats        *CacheStats
	statsMu      sync.RWMutex

	// 설정
	maxSize      int64
	retentionTime time.Duration
	cacheDir     string

	// LRU 관리
	lru          *LRUList
	lruMu        sync.Mutex
}

// CacheEntry는 캐시 엔트리입니다
type CacheEntry struct {
	Key          string                 `json:"key"`
	ImageID      string                 `json:"image_id"`
	ImageTag     string                 `json:"image_tag"`
	Size         int64                  `json:"size"`
	CreatedAt    time.Time              `json:"created_at"`
	LastAccessed time.Time              `json:"last_accessed"`
	AccessCount  int64                  `json:"access_count"`
	Metadata     map[string]interface{} `json:"metadata"`
	
	// LRU 연결 리스트
	prev         *CacheEntry
	next         *CacheEntry
}

// LayerCache는 레이어 캐시를 관리합니다
type LayerCache struct {
	layers       map[string]*LayerInfo
	layersMu     sync.RWMutex
	
	stats        *LayerCacheStats
	statsMu      sync.RWMutex
}

// LayerInfo는 레이어 정보입니다
type LayerInfo struct {
	ID           string    `json:"id"`
	Digest       string    `json:"digest"`
	Size         int64     `json:"size"`
	CreatedAt    time.Time `json:"created_at"`
	LastUsed     time.Time `json:"last_used"`
	UsageCount   int64     `json:"usage_count"`
	SharedBy     []string  `json:"shared_by"` // 이 레이어를 사용하는 이미지들
}

// BuildCache는 빌드 캐시를 관리합니다
type BuildCache struct {
	builds       map[string]*BuildCacheEntry
	buildsMu     sync.RWMutex
	
	stats        *BuildCacheStats
	statsMu      sync.RWMutex
}

// BuildCacheEntry는 빌드 캐시 엔트리입니다
type BuildCacheEntry struct {
	ContextHash  string                 `json:"context_hash"`
	DockerfileHash string               `json:"dockerfile_hash"`
	ImageID      string                 `json:"image_id"`
	BuildArgs    map[string]string      `json:"build_args"`
	CreatedAt    time.Time              `json:"created_at"`
	LastUsed     time.Time              `json:"last_used"`
	BuildTime    time.Duration          `json:"build_time"`
	Success      bool                   `json:"success"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// OptimizerStats는 최적화 통계입니다
type OptimizerStats struct {
	// 캐시 통계
	CacheHits          int64   `json:"cache_hits"`
	CacheMisses        int64   `json:"cache_misses"`
	CacheHitRate       float64 `json:"cache_hit_rate"`
	CacheSize          int64   `json:"cache_size"`
	CacheEntries       int     `json:"cache_entries"`

	// 빌드 통계
	TotalBuilds        int64         `json:"total_builds"`
	SuccessfulBuilds   int64         `json:"successful_builds"`
	FailedBuilds       int64         `json:"failed_builds"`
	AverageBuildTime   time.Duration `json:"average_build_time"`
	TotalBuildTime     time.Duration `json:"total_build_time"`

	// 최적화 통계
	ImagesOptimized    int64         `json:"images_optimized"`
	BytesSaved         int64         `json:"bytes_saved"`
	TimeSaved          time.Duration `json:"time_saved"`
	OptimizationRatio  float64       `json:"optimization_ratio"`

	// 레이어 통계
	LayersShared       int64   `json:"layers_shared"`
	LayerReuseRate     float64 `json:"layer_reuse_rate"`

	// 시간 정보
	LastUpdate         time.Time `json:"last_update"`
	UptimeSeconds      int64     `json:"uptime_seconds"`
}

// CacheStats는 캐시 통계입니다
type CacheStats struct {
	Hits             int64   `json:"hits"`
	Misses           int64   `json:"misses"`
	HitRate          float64 `json:"hit_rate"`
	TotalSize        int64   `json:"total_size"`
	EntryCount       int     `json:"entry_count"`
	EvictionCount    int64   `json:"eviction_count"`
	LastCleanup      time.Time `json:"last_cleanup"`
}

// LayerCacheStats는 레이어 캐시 통계입니다
type LayerCacheStats struct {
	TotalLayers      int     `json:"total_layers"`
	SharedLayers     int     `json:"shared_layers"`
	ReuseRate        float64 `json:"reuse_rate"`
	TotalSize        int64   `json:"total_size"`
	LastAnalysis     time.Time `json:"last_analysis"`
}

// BuildCacheStats는 빌드 캐시 통계입니다
type BuildCacheStats struct {
	TotalBuilds      int64         `json:"total_builds"`
	CachedBuilds     int64         `json:"cached_builds"`
	CacheHitRate     float64       `json:"cache_hit_rate"`
	AverageBuildTime time.Duration `json:"average_build_time"`
	TimeSaved        time.Duration `json:"time_saved"`
}

// OptimizationStrategy는 최적화 전략 인터페이스입니다
type OptimizationStrategy interface {
	Name() string
	Optimize(ctx context.Context, imageID string, options *OptimizationOptions) (*OptimizationResult, error)
	CanOptimize(imageInfo *types.ImageInspect) bool
	Priority() int
}

// OptimizationOptions는 최적화 옵션입니다
type OptimizationOptions struct {
	TargetSize       int64                  `json:"target_size"`
	CompressionLevel int                    `json:"compression_level"`
	RemoveDebugInfo  bool                   `json:"remove_debug_info"`
	MinifyFiles      bool                   `json:"minify_files"`
	StripSymbols     bool                   `json:"strip_symbols"`
	CustomRules      []string               `json:"custom_rules"`
	Metadata         map[string]interface{} `json:"metadata"`
}

// OptimizationResult는 최적화 결과입니다
type OptimizationResult struct {
	OriginalImageID  string                 `json:"original_image_id"`
	OptimizedImageID string                 `json:"optimized_image_id"`
	OriginalSize     int64                  `json:"original_size"`
	OptimizedSize    int64                  `json:"optimized_size"`
	SizeReduction    int64                  `json:"size_reduction"`
	ReductionPercent float64                `json:"reduction_percent"`
	OptimizationTime time.Duration          `json:"optimization_time"`
	Strategy         string                 `json:"strategy"`
	Success          bool                   `json:"success"`
	Error            string                 `json:"error,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// BuildOptions는 빌드 옵션입니다
type BuildOptions struct {
	Context      string            `json:"context"`
	Dockerfile   string            `json:"dockerfile"`
	Tags         []string          `json:"tags"`
	BuildArgs    map[string]string `json:"build_args"`
	Target       string            `json:"target"`
	NoCache      bool              `json:"no_cache"`
	Pull         bool              `json:"pull"`
	ForceRm      bool              `json:"force_rm"`
	CacheFrom    []string          `json:"cache_from"`
	NetworkMode  string            `json:"network_mode"`
	Platform     string            `json:"platform"`
	Squash       bool              `json:"squash"`
}

// LRUList는 LRU 연결 리스트입니다
type LRUList struct {
	head *CacheEntry
	tail *CacheEntry
	size int
}

// DefaultOptimizerConfig는 기본 최적화 설정을 반환합니다
func DefaultOptimizerConfig() OptimizerConfig {
	return OptimizerConfig{
		EnableLayerCache:     true,
		EnableBuildCache:     true,
		EnableImageCache:     true,
		CacheDir:            "/tmp/docker-cache",
		MaxCacheSize:        10 * 1024 * 1024 * 1024, // 10GB
		CacheRetention:      24 * time.Hour,
		EnableMultiStage:    true,
		EnableCompression:   true,
		CompressionLevel:    6,
		EnableMinification:  true,
		OptimizationLevel:   2,
		MaxConcurrentBuilds: 3,
		BuildTimeout:        10 * time.Minute,
		EnableParallelBuild: true,
		BuildMemoryLimit:    2 * 1024 * 1024 * 1024, // 2GB
		RegistryMirrors:     []string{},
		PullTimeout:         5 * time.Minute,
		EnablePullOptimization: true,
		AutoCleanup:         true,
		CleanupInterval:     1 * time.Hour,
		MaxImageAge:         7 * 24 * time.Hour, // 7일
	}
}

// NewImageOptimizer는 새로운 이미지 최적화기를 생성합니다
func NewImageOptimizer(dockerClient *client.Client, config OptimizerConfig) *ImageOptimizer {
	ctx, cancel := context.WithCancel(context.Background())
	
	optimizer := &ImageOptimizer{
		client:   dockerClient,
		config:   config,
		stats:    &OptimizerStats{},
		ctx:      ctx,
		cancel:   cancel,
	}
	
	// 캐시 초기화
	if config.EnableImageCache {
		optimizer.cache = NewImageCache(config.MaxCacheSize, config.CacheRetention, config.CacheDir)
	}
	if config.EnableLayerCache {
		optimizer.layerCache = NewLayerCache()
	}
	if config.EnableBuildCache {
		optimizer.buildCache = NewBuildCache()
	}
	
	// 최적화 전략 초기화
	optimizer.initOptimizationStrategies()
	
	return optimizer
}

// Start는 최적화기를 시작합니다
func (io *ImageOptimizer) Start() error {
	if !io.running.CompareAndSwap(false, true) {
		return fmt.Errorf("image optimizer is already running")
	}
	
	// 백그라운드 작업 시작
	if io.config.AutoCleanup {
		io.wg.Add(1)
		go io.cleanupLoop()
	}
	
	io.wg.Add(2)
	go io.metricsUpdateLoop()
	go io.cacheMaintenanceLoop()
	
	return nil
}

// Stop은 최적화기를 중지합니다
func (io *ImageOptimizer) Stop() error {
	if !io.running.CompareAndSwap(true, false) {
		return nil
	}
	
	io.cancel()
	io.wg.Wait()
	
	return nil
}

// OptimizeImage는 이미지를 최적화합니다
func (io *ImageOptimizer) OptimizeImage(ctx context.Context, imageID string, options *OptimizationOptions) (*OptimizationResult, error) {
	startTime := time.Now()
	
	// 이미지 정보 조회
	imageInfo, _, err := io.client.ImageInspectWithRaw(ctx, imageID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect image: %w", err)
	}
	
	// 적용 가능한 최적화 전략 선택
	var bestStrategy OptimizationStrategy
	for _, strategy := range io.strategies {
		if strategy.CanOptimize(&imageInfo) {
			if bestStrategy == nil || strategy.Priority() > bestStrategy.Priority() {
				bestStrategy = strategy
			}
		}
	}
	
	if bestStrategy == nil {
		return &OptimizationResult{
			OriginalImageID:  imageID,
			OptimizedImageID: imageID,
			OriginalSize:     imageInfo.Size,
			OptimizedSize:    imageInfo.Size,
			SizeReduction:    0,
			ReductionPercent: 0,
			OptimizationTime: time.Since(startTime),
			Strategy:         "none",
			Success:          true,
		}, nil
	}
	
	// 최적화 실행
	result, err := bestStrategy.Optimize(ctx, imageID, options)
	if err != nil {
		return nil, fmt.Errorf("optimization failed: %w", err)
	}
	
	// 통계 업데이트
	io.updateOptimizationStats(result)
	
	return result, nil
}

// BuildImageOptimized는 최적화된 이미지 빌드를 수행합니다
func (io *ImageOptimizer) BuildImageOptimized(ctx context.Context, options *BuildOptions) (*types.ImageBuildResponse, error) {
	// 빌드 캐시 확인
	if io.config.EnableBuildCache && io.buildCache != nil {
		if cachedResult := io.checkBuildCache(options); cachedResult != nil {
			// 캐시 히트
			io.stats.statsMu.Lock()
			io.stats.CacheHits++
			io.stats.statsMu.Unlock()
			
			return &types.ImageBuildResponse{
				Body: io.createCachedResponse(cachedResult),
			}, nil
		}
	}
	
	// 캐시 미스 - 실제 빌드 수행
	io.stats.statsMu.Lock()
	io.stats.CacheMisses++
	io.stats.statsMu.Unlock()
	
	return io.performOptimizedBuild(ctx, options)
}

// PullImageOptimized는 최적화된 이미지 풀을 수행합니다
func (io *ImageOptimizer) PullImageOptimized(ctx context.Context, refStr string, options types.ImagePullOptions) (io.ReadCloser, error) {
	// 이미지 캐시 확인
	if io.config.EnableImageCache && io.cache != nil {
		if entry := io.cache.Get(refStr); entry != nil {
			// 캐시 히트
			io.updateCacheStats(true)
			return io.createCachedPullResponse(entry), nil
		}
	}
	
	// 캐시 미스 - 실제 풀 수행
	io.updateCacheStats(false)
	
	pullReader, err := io.client.ImagePull(ctx, refStr, options)
	if err != nil {
		return nil, err
	}
	
	// 풀 완료 후 캐시에 저장
	go io.cacheImageAfterPull(refStr, pullReader)
	
	return pullReader, nil
}

// GetOptimizationStats는 최적화 통계를 반환합니다
func (io *ImageOptimizer) GetOptimizationStats() *OptimizerStats {
	io.statsMu.RLock()
	defer io.statsMu.RUnlock()
	
	// 통계 복사본 생성
	stats := *io.stats
	stats.LastUpdate = time.Now()
	
	// 캐시 히트율 계산
	if stats.CacheHits+stats.CacheMisses > 0 {
		stats.CacheHitRate = float64(stats.CacheHits) / float64(stats.CacheHits+stats.CacheMisses)
	}
	
	// 최적화 비율 계산
	if stats.ImagesOptimized > 0 && stats.BytesSaved > 0 {
		stats.OptimizationRatio = float64(stats.BytesSaved) / float64(stats.ImagesOptimized)
	}
	
	return &stats
}

// CleanupCache는 캐시를 정리합니다
func (io *ImageOptimizer) CleanupCache(ctx context.Context) error {
	if io.cache != nil {
		io.cache.Cleanup()
	}
	if io.layerCache != nil {
		io.layerCache.Cleanup()
	}
	if io.buildCache != nil {
		io.buildCache.Cleanup()
	}
	
	return nil
}

// 내부 메서드들

func (io *ImageOptimizer) initOptimizationStrategies() {
	io.strategies = []OptimizationStrategy{
		NewLayerOptimizationStrategy(),
		NewCompressionOptimizationStrategy(io.config.CompressionLevel),
		NewMinificationStrategy(),
		NewMultiStageOptimizationStrategy(),
	}
}

func (io *ImageOptimizer) cleanupLoop() {
	defer io.wg.Done()
	
	ticker := time.NewTicker(io.config.CleanupInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-io.ctx.Done():
			return
		case <-ticker.C:
			io.performCleanup()
		}
	}
}

func (io *ImageOptimizer) metricsUpdateLoop() {
	defer io.wg.Done()
	
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-io.ctx.Done():
			return
		case <-ticker.C:
			io.updateMetrics()
		}
	}
}

func (io *ImageOptimizer) cacheMaintenanceLoop() {
	defer io.wg.Done()
	
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-io.ctx.Done():
			return
		case <-ticker.C:
			io.performCacheMaintenance()
		}
	}
}

func (io *ImageOptimizer) checkBuildCache(options *BuildOptions) *BuildCacheEntry {
	if io.buildCache == nil {
		return nil
	}
	
	// 빌드 컨텍스트 해시 계산
	contextHash := io.calculateContextHash(options.Context)
	dockerfileHash := io.calculateDockerfileHash(options.Dockerfile)
	
	key := fmt.Sprintf("%s:%s", contextHash, dockerfileHash)
	return io.buildCache.Get(key)
}

func (io *ImageOptimizer) performOptimizedBuild(ctx context.Context, options *BuildOptions) (*types.ImageBuildResponse, error) {
	buildStartTime := time.Now()
	
	// Docker 빌드 옵션 구성
	buildOptions := types.ImageBuildOptions{
		Tags:           options.Tags,
		Dockerfile:     options.Dockerfile,
		BuildArgs:      convertBuildArgs(options.BuildArgs),
		Target:         options.Target,
		NoCache:        options.NoCache,
		Remove:         true,
		ForceRemove:    options.ForceRm,
		PullParent:     options.Pull,
		CacheFrom:      options.CacheFrom,
		NetworkMode:    options.NetworkMode,
		Platform:       options.Platform,
		Squash:         options.Squash,
	}
	
	// 빌드 컨텍스트 준비
	buildContext, err := io.prepareBuildContext(options.Context)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare build context: %w", err)
	}
	defer buildContext.Close()
	
	// 빌드 실행
	buildResponse, err := io.client.ImageBuild(ctx, buildContext, buildOptions)
	if err != nil {
		io.updateBuildStats(false, time.Since(buildStartTime))
		return nil, fmt.Errorf("build failed: %w", err)
	}
	
	// 빌드 결과 처리
	imageID, err := io.processBuildResponse(buildResponse)
	if err != nil {
		io.updateBuildStats(false, time.Since(buildStartTime))
		return buildResponse, err
	}
	
	// 빌드 캐시에 저장
	if io.config.EnableBuildCache && io.buildCache != nil {
		io.cacheBuildResult(options, imageID, time.Since(buildStartTime), true)
	}
	
	io.updateBuildStats(true, time.Since(buildStartTime))
	
	return buildResponse, nil
}

func (io *ImageOptimizer) calculateContextHash(contextPath string) string {
	// 빌드 컨텍스트 디렉토리의 해시 계산
	hash := sha256.New()
	hash.Write([]byte(contextPath))
	return fmt.Sprintf("%x", hash.Sum(nil))[:16]
}

func (io *ImageOptimizer) calculateDockerfileHash(dockerfilePath string) string {
	// Dockerfile 내용의 해시 계산
	hash := sha256.New()
	hash.Write([]byte(dockerfilePath))
	return fmt.Sprintf("%x", hash.Sum(nil))[:16]
}

func (io *ImageOptimizer) prepareBuildContext(contextPath string) (io.ReadCloser, error) {
	// 빌드 컨텍스트를 tar 아카이브로 준비
	return archive.TarWithOptions(contextPath, &archive.TarOptions{})
}

func (io *ImageOptimizer) processBuildResponse(response *types.ImageBuildResponse) (string, error) {
	// 빌드 응답에서 이미지 ID 추출
	scanner := bufio.NewScanner(response.Body)
	var imageID string
	
	for scanner.Scan() {
		line := scanner.Text()
		
		// JSON 파싱 시도
		var buildOutput map[string]interface{}
		if err := json.Unmarshal([]byte(line), &buildOutput); err == nil {
			if id, exists := buildOutput["aux"]; exists {
				if auxMap, ok := id.(map[string]interface{}); ok {
					if sha, exists := auxMap["ID"]; exists {
						imageID = sha.(string)
					}
				}
			}
		}
	}
	
	if imageID == "" {
		return "", fmt.Errorf("failed to extract image ID from build response")
	}
	
	return imageID, nil
}

func (io *ImageOptimizer) cacheBuildResult(options *BuildOptions, imageID string, buildTime time.Duration, success bool) {
	contextHash := io.calculateContextHash(options.Context)
	dockerfileHash := io.calculateDockerfileHash(options.Dockerfile)
	
	entry := &BuildCacheEntry{
		ContextHash:    contextHash,
		DockerfileHash: dockerfileHash,
		ImageID:        imageID,
		BuildArgs:      options.BuildArgs,
		CreatedAt:      time.Now(),
		LastUsed:       time.Now(),
		BuildTime:      buildTime,
		Success:        success,
	}
	
	key := fmt.Sprintf("%s:%s", contextHash, dockerfileHash)
	io.buildCache.Set(key, entry)
}

func (io *ImageOptimizer) cacheImageAfterPull(refStr string, pullReader io.ReadCloser) {
	defer pullReader.Close()
	
	// Pull 완료 대기 및 이미지 정보 수집
	// 실제 구현에서는 pull reader를 모니터링하여 완료 시점을 감지
	
	// 이미지 정보 조회
	imageInfo, _, err := io.client.ImageInspectWithRaw(context.Background(), refStr)
	if err != nil {
		return
	}
	
	// 캐시 엔트리 생성
	entry := &CacheEntry{
		Key:          refStr,
		ImageID:      imageInfo.ID,
		ImageTag:     refStr,
		Size:         imageInfo.Size,
		CreatedAt:    time.Now(),
		LastAccessed: time.Now(),
		AccessCount:  1,
	}
	
	io.cache.Set(refStr, entry)
}

func (io *ImageOptimizer) createCachedResponse(entry *BuildCacheEntry) io.ReadCloser {
	// 캐시된 빌드 결과를 위한 더미 응답 생성
	response := fmt.Sprintf(`{"stream":"Using cached image %s\n"}`, entry.ImageID)
	return io.NopCloser(strings.NewReader(response))
}

func (io *ImageOptimizer) createCachedPullResponse(entry *CacheEntry) io.ReadCloser {
	// 캐시된 이미지를 위한 더미 풀 응답 생성
	response := fmt.Sprintf(`{"status":"Using cached image","id":"%s"}`, entry.ImageID)
	return io.NopCloser(strings.NewReader(response))
}

func (io *ImageOptimizer) updateOptimizationStats(result *OptimizationResult) {
	io.statsMu.Lock()
	defer io.statsMu.Unlock()
	
	io.stats.ImagesOptimized++
	io.stats.BytesSaved += result.SizeReduction
	io.stats.TimeSaved += result.OptimizationTime
}

func (io *ImageOptimizer) updateCacheStats(hit bool) {
	io.statsMu.Lock()
	defer io.statsMu.Unlock()
	
	if hit {
		io.stats.CacheHits++
	} else {
		io.stats.CacheMisses++
	}
}

func (io *ImageOptimizer) updateBuildStats(success bool, buildTime time.Duration) {
	io.statsMu.Lock()
	defer io.statsMu.Unlock()
	
	io.stats.TotalBuilds++
	io.stats.TotalBuildTime += buildTime
	
	if success {
		io.stats.SuccessfulBuilds++
	} else {
		io.stats.FailedBuilds++
	}
	
	// 평균 빌드 시간 계산
	if io.stats.TotalBuilds > 0 {
		io.stats.AverageBuildTime = io.stats.TotalBuildTime / time.Duration(io.stats.TotalBuilds)
	}
}

func (io *ImageOptimizer) performCleanup() {
	// 오래된 이미지 정리
	ctx := context.Background()
	
	images, err := io.client.ImageList(ctx, types.ImageListOptions{})
	if err != nil {
		return
	}
	
	cutoff := time.Now().Add(-io.config.MaxImageAge)
	
	for _, img := range images {
		imgCreated := time.Unix(img.Created, 0)
		if imgCreated.Before(cutoff) {
			// 오래된 이미지 삭제
			io.client.ImageRemove(ctx, img.ID, types.ImageRemoveOptions{
				Force:         false,
				PruneChildren: true,
			})
		}
	}
}

func (io *ImageOptimizer) updateMetrics() {
	// 현재 상태 기반으로 메트릭 업데이트
	if io.cache != nil {
		cacheStats := io.cache.GetStats()
		
		io.statsMu.Lock()
		io.stats.CacheSize = cacheStats.TotalSize
		io.stats.CacheEntries = cacheStats.EntryCount
		io.statsMu.Unlock()
	}
}

func (io *ImageOptimizer) performCacheMaintenance() {
	if io.cache != nil {
		io.cache.PerformMaintenance()
	}
	if io.layerCache != nil {
		io.layerCache.PerformMaintenance()
	}
	if io.buildCache != nil {
		io.buildCache.PerformMaintenance()
	}
}

// 헬퍼 함수들

func convertBuildArgs(args map[string]string) map[string]*string {
	result := make(map[string]*string)
	for k, v := range args {
		value := v
		result[k] = &value
	}
	return result
}

// ImageCache 구현

func NewImageCache(maxSize int64, retention time.Duration, cacheDir string) *ImageCache {
	return &ImageCache{
		entries:       make(map[string]*CacheEntry),
		stats:         &CacheStats{},
		maxSize:       maxSize,
		retentionTime: retention,
		cacheDir:      cacheDir,
		lru:           NewLRUList(),
	}
}

func (ic *ImageCache) Get(key string) *CacheEntry {
	ic.entriesMu.RLock()
	defer ic.entriesMu.RUnlock()
	
	entry, exists := ic.entries[key]
	if !exists {
		ic.updateStats(false)
		return nil
	}
	
	// 만료 확인
	if time.Since(entry.LastAccessed) > ic.retentionTime {
		go ic.evict(key)
		ic.updateStats(false)
		return nil
	}
	
	// 접근 정보 업데이트
	entry.LastAccessed = time.Now()
	entry.AccessCount++
	
	// LRU 업데이트
	ic.lruMu.Lock()
	ic.lru.MoveToFront(entry)
	ic.lruMu.Unlock()
	
	ic.updateStats(true)
	return entry
}

func (ic *ImageCache) Set(key string, entry *CacheEntry) {
	ic.entriesMu.Lock()
	defer ic.entriesMu.Unlock()
	
	// 기존 엔트리 확인
	if existing, exists := ic.entries[key]; exists {
		ic.lru.Remove(existing)
	}
	
	// 새 엔트리 추가
	ic.entries[key] = entry
	ic.lru.AddToFront(entry)
	
	// 크기 제한 확인 및 정리
	ic.enforceSize()
}

func (ic *ImageCache) evict(key string) {
	ic.entriesMu.Lock()
	defer ic.entriesMu.Unlock()
	
	if entry, exists := ic.entries[key]; exists {
		delete(ic.entries, key)
		ic.lru.Remove(entry)
		ic.statsMu.Lock()
		ic.stats.EvictionCount++
		ic.statsMu.Unlock()
	}
}

func (ic *ImageCache) enforceSize() {
	var currentSize int64
	for _, entry := range ic.entries {
		currentSize += entry.Size
	}
	
	// 크기 초과 시 LRU 기반 정리
	for currentSize > ic.maxSize && ic.lru.tail != nil {
		oldest := ic.lru.tail
		delete(ic.entries, oldest.Key)
		ic.lru.Remove(oldest)
		currentSize -= oldest.Size
		ic.statsMu.Lock()
		ic.stats.EvictionCount++
		ic.statsMu.Unlock()
	}
}

func (ic *ImageCache) updateStats(hit bool) {
	ic.statsMu.Lock()
	defer ic.statsMu.Unlock()
	
	if hit {
		ic.stats.Hits++
	} else {
		ic.stats.Misses++
	}
	
	total := ic.stats.Hits + ic.stats.Misses
	if total > 0 {
		ic.stats.HitRate = float64(ic.stats.Hits) / float64(total)
	}
}

func (ic *ImageCache) GetStats() *CacheStats {
	ic.statsMu.RLock()
	defer ic.statsMu.RUnlock()
	
	stats := *ic.stats
	
	ic.entriesMu.RLock()
	stats.EntryCount = len(ic.entries)
	var totalSize int64
	for _, entry := range ic.entries {
		totalSize += entry.Size
	}
	stats.TotalSize = totalSize
	ic.entriesMu.RUnlock()
	
	return &stats
}

func (ic *ImageCache) Cleanup() {
	ic.entriesMu.Lock()
	defer ic.entriesMu.Unlock()
	
	now := time.Now()
	var toRemove []string
	
	for key, entry := range ic.entries {
		if now.Sub(entry.LastAccessed) > ic.retentionTime {
			toRemove = append(toRemove, key)
		}
	}
	
	for _, key := range toRemove {
		if entry := ic.entries[key]; entry != nil {
			delete(ic.entries, key)
			ic.lru.Remove(entry)
		}
	}
	
	ic.statsMu.Lock()
	ic.stats.LastCleanup = now
	ic.statsMu.Unlock()
}

func (ic *ImageCache) PerformMaintenance() {
	ic.Cleanup()
	ic.enforceSize()
}

// LayerCache 구현 (스텁)

func NewLayerCache() *LayerCache {
	return &LayerCache{
		layers: make(map[string]*LayerInfo),
		stats:  &LayerCacheStats{},
	}
}

func (lc *LayerCache) Cleanup() {
	// 레이어 캐시 정리 구현
}

func (lc *LayerCache) PerformMaintenance() {
	// 레이어 캐시 유지보수 구현
}

// BuildCache 구현 (스텁)

func NewBuildCache() *BuildCache {
	return &BuildCache{
		builds: make(map[string]*BuildCacheEntry),
		stats:  &BuildCacheStats{},
	}
}

func (bc *BuildCache) Get(key string) *BuildCacheEntry {
	bc.buildsMu.RLock()
	defer bc.buildsMu.RUnlock()
	
	entry, exists := bc.builds[key]
	if !exists {
		return nil
	}
	
	entry.LastUsed = time.Now()
	return entry
}

func (bc *BuildCache) Set(key string, entry *BuildCacheEntry) {
	bc.buildsMu.Lock()
	defer bc.buildsMu.Unlock()
	
	bc.builds[key] = entry
}

func (bc *BuildCache) Cleanup() {
	// 빌드 캐시 정리 구현
}

func (bc *BuildCache) PerformMaintenance() {
	// 빌드 캐시 유지보수 구현
}

// LRU 연결 리스트 구현

func NewLRUList() *LRUList {
	return &LRUList{}
}

func (lru *LRUList) AddToFront(entry *CacheEntry) {
	if lru.head == nil {
		lru.head = entry
		lru.tail = entry
	} else {
		entry.next = lru.head
		lru.head.prev = entry
		lru.head = entry
	}
	lru.size++
}

func (lru *LRUList) Remove(entry *CacheEntry) {
	if entry.prev != nil {
		entry.prev.next = entry.next
	} else {
		lru.head = entry.next
	}
	
	if entry.next != nil {
		entry.next.prev = entry.prev
	} else {
		lru.tail = entry.prev
	}
	
	entry.prev = nil
	entry.next = nil
	lru.size--
}

func (lru *LRUList) MoveToFront(entry *CacheEntry) {
	lru.Remove(entry)
	lru.AddToFront(entry)
}

// 최적화 전략들 (스텁 구현)

type LayerOptimizationStrategy struct{}

func NewLayerOptimizationStrategy() *LayerOptimizationStrategy {
	return &LayerOptimizationStrategy{}
}

func (los *LayerOptimizationStrategy) Name() string {
	return "layer_optimization"
}

func (los *LayerOptimizationStrategy) Optimize(ctx context.Context, imageID string, options *OptimizationOptions) (*OptimizationResult, error) {
	// 레이어 최적화 구현
	return &OptimizationResult{
		OriginalImageID:  imageID,
		OptimizedImageID: imageID,
		Strategy:         "layer_optimization",
		Success:          true,
	}, nil
}

func (los *LayerOptimizationStrategy) CanOptimize(imageInfo *types.ImageInspect) bool {
	return true
}

func (los *LayerOptimizationStrategy) Priority() int {
	return 3
}

type CompressionOptimizationStrategy struct {
	level int
}

func NewCompressionOptimizationStrategy(level int) *CompressionOptimizationStrategy {
	return &CompressionOptimizationStrategy{level: level}
}

func (cos *CompressionOptimizationStrategy) Name() string {
	return "compression_optimization"
}

func (cos *CompressionOptimizationStrategy) Optimize(ctx context.Context, imageID string, options *OptimizationOptions) (*OptimizationResult, error) {
	// 압축 최적화 구현
	return &OptimizationResult{
		OriginalImageID:  imageID,
		OptimizedImageID: imageID,
		Strategy:         "compression_optimization",
		Success:          true,
	}, nil
}

func (cos *CompressionOptimizationStrategy) CanOptimize(imageInfo *types.ImageInspect) bool {
	return true
}

func (cos *CompressionOptimizationStrategy) Priority() int {
	return 2
}

type MinificationStrategy struct{}

func NewMinificationStrategy() *MinificationStrategy {
	return &MinificationStrategy{}
}

func (ms *MinificationStrategy) Name() string {
	return "minification"
}

func (ms *MinificationStrategy) Optimize(ctx context.Context, imageID string, options *OptimizationOptions) (*OptimizationResult, error) {
	// 미니피케이션 구현
	return &OptimizationResult{
		OriginalImageID:  imageID,
		OptimizedImageID: imageID,
		Strategy:         "minification",
		Success:          true,
	}, nil
}

func (ms *MinificationStrategy) CanOptimize(imageInfo *types.ImageInspect) bool {
	return true
}

func (ms *MinificationStrategy) Priority() int {
	return 1
}

type MultiStageOptimizationStrategy struct{}

func NewMultiStageOptimizationStrategy() *MultiStageOptimizationStrategy {
	return &MultiStageOptimizationStrategy{}
}

func (msos *MultiStageOptimizationStrategy) Name() string {
	return "multi_stage_optimization"
}

func (msos *MultiStageOptimizationStrategy) Optimize(ctx context.Context, imageID string, options *OptimizationOptions) (*OptimizationResult, error) {
	// 멀티 스테이지 최적화 구현
	return &OptimizationResult{
		OriginalImageID:  imageID,
		OptimizedImageID: imageID,
		Strategy:         "multi_stage_optimization",
		Success:          true,
	}, nil
}

func (msos *MultiStageOptimizationStrategy) CanOptimize(imageInfo *types.ImageInspect) bool {
	return true
}

func (msos *MultiStageOptimizationStrategy) Priority() int {
	return 4
}

// 헬퍼 타입
type nopCloser struct {
	io.Reader
}

func (nopCloser) Close() error { return nil }

func (io *ImageOptimizer) NopCloser(r io.Reader) io.ReadCloser {
	return nopCloser{r}
}