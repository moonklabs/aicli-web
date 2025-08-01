package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

// sync와 atomic 패키지 사용을 명시적으로 보장
var _ = sync.RWMutex{}
var _ = atomic.AddInt32

// NewWorktreePool은 새로운 워크트리 풀을 생성합니다
func NewWorktreePool(initialSize int) *WorktreePool {
	ctx, cancel := context.WithCancel(context.Background())

	pool := &WorktreePool{
		availableWorktrees: make(chan *PrebuiltWorktree, initialSize*2),
		inUseWorktrees:     make(map[string]*PrebuiltWorktree),
		maxSize:            initialSize * 2,
		ctx:                ctx,
		cancel:             cancel,
		cleanupInterval:    10 * time.Minute,
	}

	return pool
}

// Start는 워크트리 풀을 시작합니다
func (wp *WorktreePool) Start() error {
	// 백그라운드 작업 시작
	go wp.cleanupLoop()
	go wp.maintenanceLoop()

	return nil
}

// Stop은 워크트리 풀을 중지합니다
func (wp *WorktreePool) Stop() error {
	wp.cancel()

	// 모든 워크트리 정리
	return wp.cleanupAllWorktrees()
}

// AcquireWorktree는 풀에서 워크트리를 가져옵니다
func (wp *WorktreePool) AcquireWorktree(agentID string) (*PrebuiltWorktree, error) {
	// 프로젝트 ID 기반으로 적절한 워크트리 찾기
	projectID := wp.extractProjectID(agentID)

	// 먼저 같은 프로젝트의 사용 가능한 워크트리 찾기
	if worktree := wp.findAvailableWorktreeForProject(projectID); worktree != nil {
		return wp.assignWorktreeToAgent(worktree, agentID)
	}

	// 사용 가능한 워크트리가 없으면 새로 생성
	return wp.createWorktreeForAgent(agentID, projectID)
}

// ReleaseWorktree는 워크트리를 풀에 반환합니다
func (wp *WorktreePool) ReleaseWorktree(worktreeID string) error {
	wp.mutex.Lock()
	worktree, exists := wp.inUseWorktrees[worktreeID]
	if !exists {
		wp.mutex.Unlock()
		return fmt.Errorf("worktree %s not found in use", worktreeID)
	}

	delete(wp.inUseWorktrees, worktreeID)
	wp.mutex.Unlock()

	// 워크트리 정리
	if err := wp.cleanupWorktree(worktree); err != nil {
		// 정리 실패시 워크트리 삭제
		return wp.destroyWorktree(worktree)
	}

	// 상태 업데이트
	worktree.Status = WorktreeStatusReady
	worktree.LastUsed = time.Now()

	// 풀에 반환
	select {
	case wp.availableWorktrees <- worktree:
		return nil
	default:
		// 풀이 가득찬 경우 워크트리 삭제
		return wp.destroyWorktree(worktree)
	}
}

// Optimize는 워크트리 풀을 최적화합니다
func (wp *WorktreePool) Optimize() error {
	wp.mutex.Lock()
	defer wp.mutex.Unlock()

	// 오래된 워크트리들 정리
	now := time.Now()
	maxIdleTime := 30 * time.Minute

	worktreesToRemove := make([]*PrebuiltWorktree, 0)
	tempWorktrees := make([]*PrebuiltWorktree, 0, len(wp.availableWorktrees))

	// 채널에서 모든 워크트리 꺼내기
	for {
		select {
		case worktree := <-wp.availableWorktrees:
			if now.Sub(worktree.LastUsed) > maxIdleTime {
				worktreesToRemove = append(worktreesToRemove, worktree)
			} else {
				tempWorktrees = append(tempWorktrees, worktree)
			}
		default:
			goto cleanup
		}
	}

cleanup:
	// 유지할 워크트리들을 다시 채널에 넣기
	for _, worktree := range tempWorktrees {
		select {
		case wp.availableWorktrees <- worktree:
		default:
			// 채널이 가득찬 경우 초과 워크트리 삭제
			worktreesToRemove = append(worktreesToRemove, worktree)
		}
	}

	// 오래된 워크트리들 삭제
	for _, worktree := range worktreesToRemove {
		wp.destroyWorktree(worktree)
	}

	// 디스크 공간 정리
	return wp.cleanupDiskSpace()
}

// GetPoolStats는 풀 통계를 반환합니다
func (wp *WorktreePool) GetPoolStats() WorktreePoolStats {
	wp.mutex.RLock()
	defer wp.mutex.RUnlock()

	available := len(wp.availableWorktrees)
	inUse := len(wp.inUseWorktrees)

	return WorktreePoolStats{
		TotalWorktrees:     available + inUse,
		AvailableWorktrees: available,
		InUseWorktrees:     inUse,
		MaxCapacity:        wp.maxSize,
		Utilization:        float64(inUse) / float64(wp.maxSize),
		DiskUsage:          wp.calculateDiskUsage(),
		LastOptimized:      time.Now(),
	}
}

// WorktreePoolStats는 워크트리 풀 통계입니다
type WorktreePoolStats struct {
	TotalWorktrees     int       `json:"total_worktrees"`
	AvailableWorktrees int       `json:"available_worktrees"`
	InUseWorktrees     int       `json:"in_use_worktrees"`
	MaxCapacity        int       `json:"max_capacity"`
	Utilization        float64   `json:"utilization"`
	DiskUsage          int64     `json:"disk_usage"`
	LastOptimized      time.Time `json:"last_optimized"`
}

// SyncWorktree는 워크트리를 최신 상태로 동기화합니다
func (wp *WorktreePool) SyncWorktree(worktreeID string) error {
	wp.mutex.RLock()
	worktree, exists := wp.inUseWorktrees[worktreeID]
	if !exists {
		wp.mutex.RUnlock()
		return fmt.Errorf("worktree %s not found", worktreeID)
	}
	wp.mutex.RUnlock()

	worktree.Status = WorktreeStatusSyncing

	// Git pull 또는 fetch 수행
	if err := wp.performGitSync(worktree); err != nil {
		worktree.Status = WorktreeStatusError
		return fmt.Errorf("failed to sync worktree: %w", err)
	}

	worktree.Status = WorktreeStatusInUse
	worktree.LastUsed = time.Now()

	return nil
}

// 내부 메서드들

func (wp *WorktreePool) extractProjectID(agentID string) string {
	// 에이전트 ID에서 프로젝트 ID 추출
	// 실제 구현에서는 에이전트 정보에서 가져와야 함
	return "default-project" // TODO: 실제 프로젝트 ID 추출
}

func (wp *WorktreePool) findAvailableWorktreeForProject(projectID string) *PrebuiltWorktree {
	// 비블로킹으로 사용 가능한 워크트리 찾기
	tempWorktrees := make([]*PrebuiltWorktree, 0)
	var foundWorktree *PrebuiltWorktree

	// 채널에서 워크트리들을 꺼내면서 프로젝트 매칭 찾기
	for len(tempWorktrees) < cap(wp.availableWorktrees) {
		select {
		case worktree := <-wp.availableWorktrees:
			if worktree.ProjectID == projectID && foundWorktree == nil {
				foundWorktree = worktree
			} else {
				tempWorktrees = append(tempWorktrees, worktree)
			}
		default:
			break
		}
	}

	// 사용하지 않을 워크트리들을 다시 채널에 넣기
	for _, worktree := range tempWorktrees {
		select {
		case wp.availableWorktrees <- worktree:
		default:
			// 채널이 가득찬 경우 워크트리 삭제
			wp.destroyWorktree(worktree)
		}
	}

	return foundWorktree
}

func (wp *WorktreePool) assignWorktreeToAgent(worktree *PrebuiltWorktree, agentID string) (*PrebuiltWorktree, error) {
	wp.mutex.Lock()
	defer wp.mutex.Unlock()

	worktree.Status = WorktreeStatusInUse
	worktree.LastUsed = time.Now()
	atomic.AddInt32(&worktree.UseCount, 1)

	wp.inUseWorktrees[worktree.ID] = worktree

	return worktree, nil
}

func (wp *WorktreePool) createWorktreeForAgent(agentID, projectID string) (*PrebuiltWorktree, error) {
	if int(wp.currentSize.Load()) >= wp.maxSize {
		return nil, fmt.Errorf("worktree pool exhausted")
	}

	worktreeID := uuid.New().String()
	worktreePath := wp.generateWorktreePath(worktreeID, projectID)

	// 디렉토리 생성
	if err := os.MkdirAll(worktreePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create worktree directory: %w", err)
	}

	// Git 저장소 초기화 또는 클론
	if err := wp.initializeGitWorktree(worktreePath, projectID); err != nil {
		os.RemoveAll(worktreePath)
		return nil, fmt.Errorf("failed to initialize git worktree: %w", err)
	}

	worktree := &PrebuiltWorktree{
		ID:        worktreeID,
		ProjectID: projectID,
		Branch:    "main", // TODO: 프로젝트별 기본 브랜치 설정
		Path:      worktreePath,
		CreatedAt: time.Now(),
		LastUsed:  time.Now(),
		UseCount:  1,
		Status:    WorktreeStatusInUse,
	}

	wp.mutex.Lock()
	wp.inUseWorktrees[worktreeID] = worktree
	wp.mutex.Unlock()

	wp.currentSize.Add(1)

	return worktree, nil
}

func (wp *WorktreePool) generateWorktreePath(worktreeID, projectID string) string {
	baseDir := "/tmp/aicli-worktrees" // TODO: 설정 가능하게 만들기
	return filepath.Join(baseDir, projectID, worktreeID)
}

func (wp *WorktreePool) initializeGitWorktree(path, projectID string) error {
	// Git 저장소 초기화 또는 클론
	// 실제 구현에서는 git 라이브러리 사용

	// 1. Git 저장소 클론 또는 worktree 생성
	repoURL := wp.getRepositoryURL(projectID)
	if repoURL != "" {
		return wp.cloneRepository(repoURL, path)
	}

	// 2. 빈 저장소 초기화
	return wp.initEmptyRepository(path)
}

func (wp *WorktreePool) getRepositoryURL(projectID string) string {
	// 프로젝트 ID에서 저장소 URL 조회
	// 실제 구현에서는 데이터베이스에서 조회
	return "" // 빈 문자열은 빈 저장소 초기화
}

func (wp *WorktreePool) cloneRepository(repoURL, path string) error {
	// Git 클론 수행
	// 실제 구현에서는 go-git 또는 git 명령어 사용

	// 모의 구현: .git 디렉토리 생성
	gitDir := filepath.Join(path, ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		return err
	}

	// 기본 파일들 생성
	configFile := filepath.Join(gitDir, "config")
	configContent := fmt.Sprintf(`[core]
	repositoryformatversion = 0
	filemode = true
	bare = false
	logallrefupdates = true
[remote "origin"]
	url = %s
	fetch = +refs/heads/*:refs/remotes/origin/*
[branch "main"]
	remote = origin
	merge = refs/heads/main
`, repoURL)

	return os.WriteFile(configFile, []byte(configContent), 0644)
}

func (wp *WorktreePool) initEmptyRepository(path string) error {
	// 빈 Git 저장소 초기화
	gitDir := filepath.Join(path, ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		return err
	}

	// 기본 설정 파일 생성
	configFile := filepath.Join(gitDir, "config")
	configContent := `[core]
	repositoryformatversion = 0
	filemode = true
	bare = false
	logallrefupdates = true
`

	return os.WriteFile(configFile, []byte(configContent), 0644)
}

func (wp *WorktreePool) cleanupWorktree(worktree *PrebuiltWorktree) error {
	// 워크트리 정리
	// 1. 수정된 파일 정리
	// 2. 브랜치 리셋
	// 3. 임시 파일 삭제

	worktree.Status = WorktreeStatusSyncing

	if err := wp.resetWorktreeState(worktree); err != nil {
		return fmt.Errorf("failed to reset worktree state: %w", err)
	}

	return nil
}

func (wp *WorktreePool) resetWorktreeState(worktree *PrebuiltWorktree) error {
	// Git 상태 리셋
	// 실제 구현에서는 git reset --hard HEAD 등 수행

	// 모의 구현: 임시 파일들 정리
	tempPattern := filepath.Join(worktree.Path, "*.tmp")
	if matches, err := filepath.Glob(tempPattern); err == nil {
		for _, match := range matches {
			os.Remove(match)
		}
	}

	return nil
}

func (wp *WorktreePool) destroyWorktree(worktree *PrebuiltWorktree) error {
	// 워크트리 완전 삭제
	if err := os.RemoveAll(worktree.Path); err != nil {
		return fmt.Errorf("failed to remove worktree directory: %w", err)
	}

	wp.currentSize.Add(-1)
	return nil
}

func (wp *WorktreePool) performGitSync(worktree *PrebuiltWorktree) error {
	// Git 동기화 수행
	// 실제 구현에서는 git fetch, git pull 등 수행

	// 모의 구현: 간단한 대기
	time.Sleep(500 * time.Millisecond)
	return nil
}

func (wp *WorktreePool) calculateDiskUsage() int64 {
	var totalSize int64

	wp.mutex.RLock()
	defer wp.mutex.RUnlock()

	// 사용 중인 워크트리들의 디스크 사용량 계산
	for _, worktree := range wp.inUseWorktrees {
		if size, err := wp.getDirectorySize(worktree.Path); err == nil {
			totalSize += size
		}
	}

	// 사용 가능한 워크트리들의 디스크 사용량도 계산 필요
	// 하지만 채널에서 꺼내지 않고 계산하는 것은 복잡하므로 근사치 사용
	availableCount := len(wp.availableWorktrees)
	if availableCount > 0 {
		avgSize := totalSize / int64(len(wp.inUseWorktrees)+1)
		totalSize += avgSize * int64(availableCount)
	}

	return totalSize
}

func (wp *WorktreePool) getDirectorySize(path string) (int64, error) {
	var size int64

	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})

	return size, err
}

func (wp *WorktreePool) cleanupDiskSpace() error {
	// 디스크 공간 정리
	// 1. 임시 파일 삭제
	// 2. Git GC 실행
	// 3. 로그 파일 정리

	baseDir := "/tmp/aicli-worktrees"

	// 임시 파일 패턴들
	tempPatterns := []string{
		"*.tmp",
		"*.temp",
		"*.log",
		".DS_Store",
	}

	return filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // 에러 무시하고 계속
		}

		if info.IsDir() {
			return nil
		}

		for _, pattern := range tempPatterns {
			if matched, _ := filepath.Match(pattern, info.Name()); matched {
				os.Remove(path)
				break
			}
		}

		return nil
	})
}

func (wp *WorktreePool) cleanupLoop() {
	ticker := time.NewTicker(wp.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-wp.ctx.Done():
			return
		case <-ticker.C:
			wp.Optimize()
		}
	}
}

func (wp *WorktreePool) maintenanceLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-wp.ctx.Done():
			return
		case <-ticker.C:
			wp.performMaintenance()
		}
	}
}

func (wp *WorktreePool) performMaintenance() {
	// 정기 유지보수 작업
	// 1. 디스크 사용량 체크
	// 2. Git 저장소 상태 체크
	// 3. 손상된 워크트리 정리

	totalDiskUsage := wp.calculateDiskUsage()
	maxDiskUsage := int64(10 * 1024 * 1024 * 1024) // 10GB

	if totalDiskUsage > maxDiskUsage {
		// 디스크 사용량이 초과되면 오래된 워크트리들 정리
		wp.cleanupOldWorktrees()
	}

	// Git 저장소 무결성 체크
	wp.checkWorktreeIntegrity()
}

func (wp *WorktreePool) cleanupOldWorktrees() {
	wp.mutex.Lock()
	defer wp.mutex.Unlock()

	// 사용 중이지 않은 오래된 워크트리들 찾아서 삭제
	cutoffTime := time.Now().Add(-2 * time.Hour)

	worktreesToRemove := make([]*PrebuiltWorktree, 0)
	tempWorktrees := make([]*PrebuiltWorktree, 0)

	// 사용 가능한 워크트리들 체크
	for {
		select {
		case worktree := <-wp.availableWorktrees:
			if worktree.LastUsed.Before(cutoffTime) {
				worktreesToRemove = append(worktreesToRemove, worktree)
			} else {
				tempWorktrees = append(tempWorktrees, worktree)
			}
		default:
			goto cleanup
		}
	}

cleanup:
	// 유지할 워크트리들 다시 넣기
	for _, worktree := range tempWorktrees {
		select {
		case wp.availableWorktrees <- worktree:
		default:
			worktreesToRemove = append(worktreesToRemove, worktree)
		}
	}

	// 선택된 워크트리들 삭제
	for _, worktree := range worktreesToRemove {
		wp.destroyWorktree(worktree)
	}
}

func (wp *WorktreePool) checkWorktreeIntegrity() {
	wp.mutex.RLock()
	defer wp.mutex.RUnlock()

	// 사용 중인 워크트리들의 무결성 체크
	for _, worktree := range wp.inUseWorktrees {
		if !wp.isWorktreeValid(worktree) {
			worktree.Status = WorktreeStatusError
		}
	}
}

func (wp *WorktreePool) isWorktreeValid(worktree *PrebuiltWorktree) bool {
	// 워크트리 유효성 체크
	// 1. 디렉토리 존재 여부
	// 2. .git 디렉토리 존재 여부
	// 3. Git 저장소 상태

	if _, err := os.Stat(worktree.Path); err != nil {
		return false
	}

	gitDir := filepath.Join(worktree.Path, ".git")
	if _, err := os.Stat(gitDir); err != nil {
		return false
	}

	return true
}

func (wp *WorktreePool) cleanupAllWorktrees() error {
	wp.mutex.Lock()
	defer wp.mutex.Unlock()

	// 사용 가능한 워크트리들 정리
	for {
		select {
		case worktree := <-wp.availableWorktrees:
			wp.destroyWorktree(worktree)
		default:
			goto inUseCleanup
		}
	}

inUseCleanup:
	// 사용 중인 워크트리들 정리
	for _, worktree := range wp.inUseWorktrees {
		wp.destroyWorktree(worktree)
	}

	wp.inUseWorktrees = make(map[string]*PrebuiltWorktree)

	return nil
}
