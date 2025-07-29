package agent

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aicli/aicli-web/internal/docker"
	"github.com/google/uuid"
)

// sync와 atomic 패키지 사용을 명시적으로 보장
var _ = sync.RWMutex{}
var _ = atomic.AddInt32

// NewContainerPool은 새로운 컨테이너 풀을 생성합니다
func NewContainerPool(initialSize, maxSize int, dockerClient docker.Client) *ContainerPool {
	ctx, cancel := context.WithCancel(context.Background())
	
	pool := &ContainerPool{
		availableContainers: make(chan *PrebuiltContainer, maxSize),
		inUseContainers:     make(map[string]*PrebuiltContainer),
		maxSize:             maxSize,
		warmupSize:          initialSize / 2, // 50% warmup
		dockerClient:        dockerClient,
		imageCache:          NewImageCache(ctx),
		ctx:                 ctx,
		cancel:              cancel,
		cleanupInterval:     5 * time.Minute,
	}
	
	return pool
}

// Start는 컨테이너 풀을 시작합니다
func (cp *ContainerPool) Start() error {
	// 초기 컨테이너들 생성
	if err := cp.warmupContainers(); err != nil {
		return fmt.Errorf("failed to warmup containers: %w", err)
	}
	
	// 이미지 캐시 시작
	if err := cp.imageCache.Start(); err != nil {
		return fmt.Errorf("failed to start image cache: %w", err)
	}
	
	// 백그라운드 정리 작업 시작
	go cp.cleanupLoop()
	go cp.warmupLoop()
	go cp.monitoringLoop()
	
	return nil
}

// Stop은 컨테이너 풀을 중지합니다
func (cp *ContainerPool) Stop() error {
	cp.cancel()
	
	// 이미지 캐시 중지
	cp.imageCache.Stop()
	
	// 모든 컨테이너들 정리
	return cp.cleanupAllContainers()
}

// AcquireContainer는 풀에서 컨테이너를 가져옵니다
func (cp *ContainerPool) AcquireContainer(agentID string) (*PrebuiltContainer, error) {
	select {
	case container := <-cp.availableContainers:
		// 풀에서 사용 가능한 컨테이너 가져오기
		cp.mutex.Lock()
		cp.inUseContainers[container.ID] = container
		cp.mutex.Unlock()
		
		// 컨테이너 상태 업데이트
		container.Status = PoolContainerStatusInUse
		container.LastUsed = time.Now()
		atomic.AddInt32(&container.UseCount, 1)
		
		return container, nil
		
	default:
		// 풀이 비어있으면 새 컨테이너 생성
		if cp.canCreateContainer() {
			return cp.createContainer(agentID)
		}
		
		// 풀이 가득참 - 대기 또는 에러 반환
		return nil, fmt.Errorf("container pool exhausted")
	}
}

// ReleaseContainer는 컨테이너를 풀에 반환합니다
func (cp *ContainerPool) ReleaseContainer(containerID string) error {
	cp.mutex.Lock()
	container, exists := cp.inUseContainers[containerID]
	if !exists {
		cp.mutex.Unlock()
		return fmt.Errorf("container %s not found in use", containerID)
	}
	
	delete(cp.inUseContainers, containerID)
	cp.mutex.Unlock()
	
	// 컨테이너 정리 및 상태 업데이트
	if err := cp.cleanupContainer(container); err != nil {
		// 정리 실패시 컨테이너 삭제
		return cp.destroyContainer(container)
	}
	
	// 컨테이너 상태 업데이트
	container.Status = PoolContainerStatusReady
	container.LastUsed = time.Now()
	
	// 풀에 반환
	select {
	case cp.availableContainers <- container:
		return nil
	default:
		// 풀이 가득찬 경우 컨테이너 삭제
		return cp.destroyContainer(container)
	}
}

// WarmupContainers는 컨테이너들을 미리 생성합니다
func (cp *ContainerPool) WarmupContainers() error {
	return cp.warmupContainers()
}

// Optimize는 풀을 최적화합니다
func (cp *ContainerPool) Optimize() error {
	cp.mutex.Lock()
	defer cp.mutex.Unlock()
	
	// 사용되지 않는 오래된 컨테이너들 정리
	now := time.Now()
	maxIdleTime := 10 * time.Minute
	
	containersToRemove := make([]*PrebuiltContainer, 0)
	
	// 사용 가능한 컨테이너들 중 오래된 것들 찾기
	tempContainers := make([]*PrebuiltContainer, 0, len(cp.availableContainers))
	
	// 채널에서 모든 컨테이너 꺼내기
	for {
		select {
		case container := <-cp.availableContainers:
			if now.Sub(container.LastUsed) > maxIdleTime {
				containersToRemove = append(containersToRemove, container)
			} else {
				tempContainers = append(tempContainers, container)
			}
		default:
			goto cleanup
		}
	}
	
cleanup:
	// 유지할 컨테이너들을 다시 채널에 넣기
	for _, container := range tempContainers {
		select {
		case cp.availableContainers <- container:
		default:
			// 채널이 가득찬 경우 초과 컨테이너 삭제
			containersToRemove = append(containersToRemove, container)
		}
	}
	
	// 오래된 컨테이너들 삭제
	for _, container := range containersToRemove {
		cp.destroyContainer(container)
	}
	
	// 풀 크기가 부족하면 새 컨테이너 생성
	currentSize := int(cp.currentSize.Load())
	if currentSize < cp.warmupSize {
		needed := cp.warmupSize - currentSize
		for i := 0; i < needed; i++ {
			if container, err := cp.createContainer("warmup"); err == nil {
				select {
				case cp.availableContainers <- container:
				default:
					cp.destroyContainer(container)
					break
				}
			}
		}
	}
	
	return nil
}

// GetPoolStats는 풀 통계를 반환합니다
func (cp *ContainerPool) GetPoolStats() ContainerPoolStats {
	cp.mutex.RLock()
	defer cp.mutex.RUnlock()
	
	available := len(cp.availableContainers)
	inUse := len(cp.inUseContainers)
	
	return ContainerPoolStats{
		TotalContainers:     available + inUse,
		AvailableContainers: available,
		InUseContainers:     inUse,
		MaxCapacity:         cp.maxSize,
		Utilization:         float64(inUse) / float64(cp.maxSize),
		LastOptimized:       time.Now(), // TODO: 실제 최적화 시간 추적
	}
}

// ContainerPoolStats는 컨테이너 풀 통계입니다
type ContainerPoolStats struct {
	TotalContainers     int       `json:"total_containers"`
	AvailableContainers int       `json:"available_containers"`
	InUseContainers     int       `json:"in_use_containers"`
	MaxCapacity         int       `json:"max_capacity"`
	Utilization         float64   `json:"utilization"`
	LastOptimized       time.Time `json:"last_optimized"`
}

// 내부 메서드들

func (cp *ContainerPool) warmupContainers() error {
	cp.creationMutex.Lock()
	defer cp.creationMutex.Unlock()
	
	for i := 0; i < cp.warmupSize; i++ {
		container, err := cp.createContainer(fmt.Sprintf("warmup-%d", i))
		if err != nil {
			return fmt.Errorf("failed to create warmup container %d: %w", i, err)
		}
		
		// 풀에 추가
		select {
		case cp.availableContainers <- container:
		default:
			// 풀이 가득찬 경우 컨테이너 삭제
			cp.destroyContainer(container)
			break
		}
	}
	
	return nil
}

func (cp *ContainerPool) createContainer(agentID string) (*PrebuiltContainer, error) {
	containerID := uuid.New().String()
	
	// 기본 이미지에서 컨테이너 생성
	imageID := "aicli-agent:latest" // TODO: 설정 가능하게 만들기
	
	// 이미지 캐시에서 확인
	if !cp.imageCache.HasImage(imageID) {
		if err := cp.pullAndCacheImage(imageID); err != nil {
			return nil, fmt.Errorf("failed to pull image: %w", err)
		}
	}
	
	// Docker 컨테이너 생성
	containerConfig := map[string]interface{}{
		"Image":  imageID,
		"Labels": map[string]string{
			"aicli.pool":     "agent-pool",
			"aicli.agent":    agentID,
			"aicli.created":  time.Now().Format(time.RFC3339),
		},
		"Env": []string{
			"AICLI_AGENT_ID=" + agentID,
			"AICLI_POOL_MODE=true",
		},
	}
	
	// 실제 Docker 컨테이너 생성 (Docker 클라이언트 사용)
	dockerContainer, err := cp.createDockerContainer(containerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create docker container: %w", err)
	}
	
	container := &PrebuiltContainer{
		ID:              containerID,
		ImageID:         imageID,
		Config:          containerConfig,
		CreatedAt:       time.Now(),
		LastUsed:        time.Now(),
		UseCount:        0,
		Status:          PoolContainerStatusReady,
		DockerContainer: dockerContainer,
	}
	
	cp.currentSize.Add(1)
	
	return container, nil
}

func (cp *ContainerPool) createDockerContainer(config map[string]interface{}) (interface{}, error) {
	// Docker 클라이언트를 사용한 실제 컨테이너 생성
	// 실제 구현에서는 docker.Client 인터페이스 사용
	// 여기서는 모의 구현
	return &struct {
		ID   string
		Name string
	}{
		ID:   uuid.New().String(),
		Name: fmt.Sprintf("aicli-agent-%s", uuid.New().String()[:8]),
	}, nil
}

func (cp *ContainerPool) cleanupContainer(container *PrebuiltContainer) error {
	// 컨테이너 내부 정리
	// 1. 임시 파일 삭제
	// 2. 프로세스 종료
	// 3. 메모리 정리
	// 4. 환경 변수 리셋
	
	container.Status = PoolContainerStatusRecycling
	
	// 실제 정리 작업 수행
	if err := cp.performContainerCleanup(container); err != nil {
		return fmt.Errorf("cleanup failed: %w", err)
	}
	
	// 리소스 사용량 리셋
	container.ResourceUsage = ResourceUsage{
		LastUpdated: time.Now(),
	}
	
	return nil
}

func (cp *ContainerPool) performContainerCleanup(container *PrebuiltContainer) error {
	// 실제 Docker 컨테이너 정리 작업
	// 여기서는 모의 구현
	time.Sleep(100 * time.Millisecond) // 정리 시간 시뮬레이션
	return nil
}

func (cp *ContainerPool) destroyContainer(container *PrebuiltContainer) error {
	// Docker 컨테이너 삭제
	if err := cp.removeDockerContainer(container); err != nil {
		return fmt.Errorf("failed to remove docker container: %w", err)
	}
	
	cp.currentSize.Add(-1)
	return nil
}

func (cp *ContainerPool) removeDockerContainer(container *PrebuiltContainer) error {
	// 실제 Docker 컨테이너 삭제
	// 여기서는 모의 구현
	return nil
}

func (cp *ContainerPool) pullAndCacheImage(imageID string) error {
	// 이미지 풀 및 캐시
	return cp.imageCache.PullAndCache(imageID)
}

func (cp *ContainerPool) canCreateContainer() bool {
	current := int(cp.currentSize.Load())
	return current < cp.maxSize
}

func (cp *ContainerPool) cleanupLoop() {
	ticker := time.NewTicker(cp.cleanupInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-cp.ctx.Done():
			return
		case <-ticker.C:
			cp.Optimize()
		}
	}
}

func (cp *ContainerPool) warmupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-cp.ctx.Done():
			return
		case <-ticker.C:
			cp.maintainPoolSize()
		}
	}
}

func (cp *ContainerPool) maintainPoolSize() {
	available := len(cp.availableContainers)
	if available < cp.warmupSize/2 {
		// 풀 크기가 부족하면 컨테이너 추가 생성
		needed := cp.warmupSize/2 - available
		for i := 0; i < needed && cp.canCreateContainer(); i++ {
			if container, err := cp.createContainer("maintenance"); err == nil {
				select {
				case cp.availableContainers <- container:
				default:
					cp.destroyContainer(container)
					break
				}
			}
		}
	}
}

func (cp *ContainerPool) monitoringLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-cp.ctx.Done():
			return
		case <-ticker.C:
			cp.updateResourceUsage()
		}
	}
}

func (cp *ContainerPool) updateResourceUsage() {
	cp.mutex.RLock()
	defer cp.mutex.RUnlock()
	
	// 사용 중인 컨테이너들의 리소스 사용량 업데이트
	for _, container := range cp.inUseContainers {
		cp.updateContainerResourceUsage(container)
	}
}

func (cp *ContainerPool) updateContainerResourceUsage(container *PrebuiltContainer) {
	// Docker 컨테이너의 실제 리소스 사용량 조회
	// 여기서는 모의 데이터
	container.ResourceUsage = ResourceUsage{
		CPUUsage:    0.1,  // 10% CPU
		MemoryUsage: 50 * 1024 * 1024, // 50MB
		DiskUsage:   10 * 1024 * 1024, // 10MB
		NetworkRx:   1024,
		NetworkTx:   1024,
		LastUpdated: time.Now(),
	}
}

func (cp *ContainerPool) cleanupAllContainers() error {
	cp.mutex.Lock()
	defer cp.mutex.Unlock()
	
	// 사용 가능한 컨테이너들 정리
	for {
		select {
		case container := <-cp.availableContainers:
			cp.destroyContainer(container)
		default:
			goto inUseCleanup
		}
	}
	
inUseCleanup:
	// 사용 중인 컨테이너들 정리
	for _, container := range cp.inUseContainers {
		cp.destroyContainer(container)
	}
	
	cp.inUseContainers = make(map[string]*PrebuiltContainer)
	
	return nil
}

// NewImageCache는 새로운 이미지 캐시를 생성합니다
func NewImageCache(ctx context.Context) *ImageCache {
	cache := &ImageCache{
		cachedImages:    make(map[string]*CachedImage),
		maxCacheSize:    10 * 1024 * 1024 * 1024, // 10GB
		ctx:             ctx,
		cleanupInterval: 30 * time.Minute,
	}
	
	return cache
}

// Start는 이미지 캐시를 시작합니다
func (ic *ImageCache) Start() error {
	go ic.cleanupLoop()
	return nil
}

// Stop은 이미지 캐시를 중지합니다
func (ic *ImageCache) Stop() error {
	ic.cancel()
	return nil
}

// HasImage는 이미지가 캐시에 있는지 확인합니다
func (ic *ImageCache) HasImage(imageID string) bool {
	ic.mutex.RLock()
	defer ic.mutex.RUnlock()
	
	_, exists := ic.cachedImages[imageID]
	return exists
}

// PullAndCache는 이미지를 풀하고 캐시합니다
func (ic *ImageCache) PullAndCache(imageID string) error {
	ic.mutex.Lock()
	defer ic.mutex.Unlock()
	
	// 이미 캐시되어 있으면 사용 시간만 업데이트
	if cached, exists := ic.cachedImages[imageID]; exists {
		cached.LastUsed = time.Now()
		atomic.AddInt32(&cached.UseCount, 1)
		return nil
	}
	
	// 이미지 풀 (실제 구현에서는 Docker 클라이언트 사용)
	imageSize, err := ic.pullImage(imageID)
	if err != nil {
		return fmt.Errorf("failed to pull image: %w", err)
	}
	
	// 캐시 공간 확인 및 정리
	if err := ic.ensureCacheSpace(imageSize); err != nil {
		return fmt.Errorf("failed to ensure cache space: %w", err)
	}
	
	// 캐시에 추가
	cached := &CachedImage{
		ID:        imageID,
		Tag:       imageID, // 간단한 구현
		Size:      imageSize,
		CreatedAt: time.Now(),
		LastUsed:  time.Now(),
		UseCount:  1,
	}
	
	ic.cachedImages[imageID] = cached
	ic.currentCacheSize.Add(imageSize)
	
	return nil
}

func (ic *ImageCache) pullImage(imageID string) (int64, error) {
	// 실제 Docker 이미지 풀 구현
	// 여기서는 모의 구현
	time.Sleep(2 * time.Second) // 풀 시간 시뮬레이션
	return 500 * 1024 * 1024, nil // 500MB
}

func (ic *ImageCache) ensureCacheSpace(requiredSize int64) error {
	currentSize := ic.currentCacheSize.Load()
	if currentSize+requiredSize <= ic.maxCacheSize {
		return nil
	}
	
	// LRU 방식으로 오래된 이미지들 제거
	return ic.evictOldImages(requiredSize)
}

func (ic *ImageCache) evictOldImages(requiredSize int64) error {
	// 사용 시간 기준으로 정렬하여 오래된 이미지들 제거
	// 실제 구현에서는 더 정교한 LRU 알고리즘 사용
	
	var freedSize int64
	toRemove := make([]string, 0)
	
	for id, cached := range ic.cachedImages {
		if time.Since(cached.LastUsed) > 1*time.Hour {
			toRemove = append(toRemove, id)
			freedSize += cached.Size
		}
		
		if freedSize >= requiredSize {
			break
		}
	}
	
	// 선택된 이미지들 제거
	for _, id := range toRemove {
		if cached, exists := ic.cachedImages[id]; exists {
			delete(ic.cachedImages, id)
			ic.currentCacheSize.Add(-cached.Size)
		}
	}
	
	return nil
}

func (ic *ImageCache) cleanupLoop() {
	ticker := time.NewTicker(ic.cleanupInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ic.ctx.Done():
			return
		case <-ticker.C:
			ic.performCleanup()
		}
	}
}

func (ic *ImageCache) performCleanup() {
	ic.mutex.Lock()
	defer ic.mutex.Unlock()
	
	// 오래된 이미지들 정리 (24시간 이상 미사용)
	maxAge := 24 * time.Hour
	now := time.Now()
	
	toRemove := make([]string, 0)
	for id, cached := range ic.cachedImages {
		if now.Sub(cached.LastUsed) > maxAge {
			toRemove = append(toRemove, id)
		}
	}
	
	// 제거 실행
	for _, id := range toRemove {
		if cached, exists := ic.cachedImages[id]; exists {
			delete(ic.cachedImages, id)
			ic.currentCacheSize.Add(-cached.Size)
		}
	}
}