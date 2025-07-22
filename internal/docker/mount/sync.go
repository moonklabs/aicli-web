package mount

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Syncer 파일 동기화 및 모니터링을 담당합니다.
type Syncer struct {
	watchInterval time.Duration
	watchers      map[string]*FileWatcher
	mu            sync.RWMutex
}

// NewSyncer 새로운 동기화 매니저를 생성합니다.
func NewSyncer() *Syncer {
	return &Syncer{
		watchInterval: 5 * time.Second,
		watchers:      make(map[string]*FileWatcher),
	}
}

// SyncStatus 동기화 상태 정보입니다.
type SyncStatus struct {
	LastSync      time.Time `json:"last_sync"`
	FilesChanged  int       `json:"files_changed"`
	SyncDuration  string    `json:"sync_duration"`
	Errors        []string  `json:"errors,omitempty"`
	IsActive      bool      `json:"is_active"`
}

// FileWatcher 파일 변경 사항을 감시합니다.
type FileWatcher struct {
	sourcePath   string
	lastModTime  time.Time
	callback     func([]string)
	excludePatterns []string
	cancel       context.CancelFunc
	errors       []string
	mu           sync.RWMutex
}

// WatchChanges 파일 변경 사항을 실시간으로 감시합니다.
func (s *Syncer) WatchChanges(ctx context.Context, sourcePath string, excludePatterns []string, callback func([]string)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// 기존 watcher가 있으면 정리
	if existing, exists := s.watchers[sourcePath]; exists {
		existing.Stop()
	}
	
	// 새로운 watcher 생성
	watcher := &FileWatcher{
		sourcePath:      sourcePath,
		lastModTime:     time.Now(),
		callback:        callback,
		excludePatterns: excludePatterns,
		errors:          make([]string, 0),
	}
	
	// 컨텍스트 생성
	watchCtx, cancel := context.WithCancel(ctx)
	watcher.cancel = cancel
	
	s.watchers[sourcePath] = watcher
	
	// 백그라운드에서 감시 시작
	go s.watchLoop(watchCtx, watcher)
	
	return nil
}

// watchLoop 파일 변경 감시 루프를 실행합니다.
func (s *Syncer) watchLoop(ctx context.Context, watcher *FileWatcher) {
	ticker := time.NewTicker(s.watchInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.checkForChanges(watcher); err != nil {
				watcher.addError(fmt.Sprintf("watch error: %v", err))
			}
		}
	}
}

// checkForChanges 파일 변경 사항을 확인합니다.
func (s *Syncer) checkForChanges(watcher *FileWatcher) error {
	changes, newModTime, err := s.scanForChanges(watcher.sourcePath, watcher.lastModTime, watcher.excludePatterns)
	if err != nil {
		return err
	}
	
	if len(changes) > 0 && watcher.callback != nil {
		watcher.callback(changes)
	}
	
	watcher.mu.Lock()
	watcher.lastModTime = newModTime
	watcher.mu.Unlock()
	
	return nil
}

// scanForChanges 디렉토리를 스캔하여 변경된 파일을 찾습니다.
func (s *Syncer) scanForChanges(rootPath string, since time.Time, excludePatterns []string) ([]string, time.Time, error) {
	var changes []string
	var latestModTime time.Time
	
	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // 개별 파일 에러는 무시
		}
		
		// 제외 패턴 검사
		if s.shouldExclude(path, rootPath, excludePatterns) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		
		// 변경 시간 확인
		modTime := info.ModTime()
		if modTime.After(since) {
			relPath, err := filepath.Rel(rootPath, path)
			if err == nil {
				changes = append(changes, relPath)
			}
		}
		
		if modTime.After(latestModTime) {
			latestModTime = modTime
		}
		
		return nil
	})
	
	return changes, latestModTime, err
}

// shouldExclude 파일이 제외 패턴에 해당하는지 확인합니다.
func (s *Syncer) shouldExclude(path, rootPath string, excludePatterns []string) bool {
	relPath, err := filepath.Rel(rootPath, path)
	if err != nil {
		return true
	}
	
	// 제외 패턴 확인
	for _, pattern := range excludePatterns {
		// 정확한 매칭
		if matched, _ := filepath.Match(pattern, relPath); matched {
			return true
		}
		
		// 디렉토리 패턴 처리 (예: "node_modules/*")
		if strings.HasSuffix(pattern, "/*") {
			dirPattern := strings.TrimSuffix(pattern, "/*")
			if matched, _ := filepath.Match(dirPattern, filepath.Dir(relPath)); matched {
				return true
			}
			if strings.Contains(relPath, dirPattern+string(filepath.Separator)) {
				return true
			}
		}
		
		// 확장자 패턴 처리 (예: "*.tmp")
		if strings.HasPrefix(pattern, "*.") {
			if strings.HasSuffix(relPath, pattern[1:]) {
				return true
			}
		}
		
		// 부분 문자열 매칭
		if strings.Contains(relPath, pattern) {
			return true
		}
	}
	
	return false
}

// StopWatch 파일 감시를 중지합니다.
func (s *Syncer) StopWatch(sourcePath string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if watcher, exists := s.watchers[sourcePath]; exists {
		watcher.Stop()
		delete(s.watchers, sourcePath)
	}
}

// Stop 파일 watcher를 중지합니다.
func (fw *FileWatcher) Stop() {
	if fw.cancel != nil {
		fw.cancel()
	}
}

// addError watcher에 에러를 추가합니다.
func (fw *FileWatcher) addError(errMsg string) {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	
	fw.errors = append(fw.errors, fmt.Sprintf("%s: %s", time.Now().Format(time.RFC3339), errMsg))
	
	// 에러 로그 크기 제한 (최대 10개)
	if len(fw.errors) > 10 {
		fw.errors = fw.errors[1:]
	}
}

// CheckMountStatus 마운트 상태를 확인합니다.
func (s *Syncer) CheckMountStatus(ctx context.Context, containerID string, mountPath string) (*SyncStatus, error) {
	s.mu.RLock()
	watcher, exists := s.watchers[mountPath]
	s.mu.RUnlock()
	
	status := &SyncStatus{
		LastSync:     time.Now(),
		FilesChanged: 0,
		SyncDuration: "0ms",
		IsActive:     exists,
	}
	
	if exists {
		watcher.mu.RLock()
		status.LastSync = watcher.lastModTime
		status.Errors = make([]string, len(watcher.errors))
		copy(status.Errors, watcher.errors)
		watcher.mu.RUnlock()
	}
	
	return status, nil
}

// GetFileStats 파일 통계 정보를 조회합니다.
func (s *Syncer) GetFileStats(ctx context.Context, sourcePath string, excludePatterns []string) (*FileStats, error) {
	stats := &FileStats{
		Path:      sourcePath,
		ScannedAt: time.Now(),
	}
	
	err := filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // 개별 파일 에러는 무시
		}
		
		// 제외 패턴 검사
		if s.shouldExclude(path, sourcePath, excludePatterns) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		
		if info.IsDir() {
			stats.DirectoryCount++
		} else {
			stats.FileCount++
			stats.TotalSize += info.Size()
			
			// 가장 최근 수정 파일 추적
			if info.ModTime().After(stats.LastModified) {
				stats.LastModified = info.ModTime()
				stats.LastModifiedFile = path
			}
		}
		
		return nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("scan directory: %w", err)
	}
	
	return stats, nil
}

// SyncDirectory 디렉토리 동기화를 수행합니다. (향후 구현)
func (s *Syncer) SyncDirectory(ctx context.Context, sourcePath, targetPath string, options *SyncOptions) (*SyncResult, error) {
	start := time.Now()
	
	result := &SyncResult{
		SourcePath: sourcePath,
		TargetPath: targetPath,
		StartTime:  start,
		Status:     "completed",
	}
	
	// 실제 동기화 로직은 Docker 마운트에 의해 처리됨
	// 여기서는 상태 정보만 반환
	
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	
	return result, nil
}

// StopAll 모든 감시를 중지합니다.
func (s *Syncer) StopAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	for path, watcher := range s.watchers {
		watcher.Stop()
		delete(s.watchers, path)
	}
}

// GetActiveWatchers 활성화된 watcher 목록을 반환합니다.
func (s *Syncer) GetActiveWatchers() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	paths := make([]string, 0, len(s.watchers))
	for path := range s.watchers {
		paths = append(paths, path)
	}
	
	return paths
}

// FileStats 파일 통계 정보입니다.
type FileStats struct {
	Path             string    `json:"path"`
	FileCount        int       `json:"file_count"`
	DirectoryCount   int       `json:"directory_count"`
	TotalSize        int64     `json:"total_size"`
	LastModified     time.Time `json:"last_modified"`
	LastModifiedFile string    `json:"last_modified_file"`
	ScannedAt        time.Time `json:"scanned_at"`
}

// SyncOptions 동기화 옵션입니다.
type SyncOptions struct {
	ExcludePatterns []string `json:"exclude_patterns"`
	DryRun          bool     `json:"dry_run"`
	Recursive       bool     `json:"recursive"`
}

// SyncResult 동기화 결과입니다.
type SyncResult struct {
	SourcePath   string        `json:"source_path"`
	TargetPath   string        `json:"target_path"`
	StartTime    time.Time     `json:"start_time"`
	EndTime      time.Time     `json:"end_time"`
	Duration     time.Duration `json:"duration"`
	Status       string        `json:"status"`
	FilesChanged int           `json:"files_changed"`
	Errors       []string      `json:"errors,omitempty"`
}