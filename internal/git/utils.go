package git

import (
	"os"
	"path/filepath"
)

// calculateDirSize 디렉토리 크기 계산
func calculateDirSize(path string) (int64, error) {
	var size int64

	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			// 권한 오류 등은 무시
			return nil
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})

	return size, err
}

// calculateWorktreeSize worktree 크기 계산
func calculateWorktreeSize(worktreePath string) (int64, error) {
	// .git 디렉토리 제외하고 계산
	var size int64

	err := filepath.Walk(worktreePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// .git 디렉토리는 제외
		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		}

		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})

	return size, err
}

// countActiveWorktrees 활성 worktree 수 계산
func (m *AdvancedWorktreeManager) countActiveWorktrees() int {
	// TODO: 실제 파일 시스템에서 worktree 수 계산
	// 현재는 캐시된 worktree 수를 반환
	return len(m.cache)
}

// calculateTotalDiskUsage 전체 worktree 디스크 사용량 계산
func (m *AdvancedWorktreeManager) calculateTotalDiskUsage() int64 {
	// TODO: 실제 디스크 사용량 계산 구현
	// 현재는 0을 반환
	return 0
}

// cleanupOldWorktrees 오래된 worktree 정리
func (m *AdvancedWorktreeManager) cleanupOldWorktrees() int {
	deletedCount := 0
	// TODO: 구현 필요
	// 1. 모든 worktree 스캔
	// 2. 마지막 접근 시간 확인
	// 3. maxAge보다 오래된 것들 삭제
	// 4. 삭제된 수 반환
	return deletedCount
}

// updateActiveWorktreeCount 활성 worktree 수 업데이트
func (m *AdvancedWorktreeManager) updateActiveWorktreeCount() {
	count := m.countActiveWorktrees()
	m.metrics.mu.Lock()
	m.metrics.ActiveWorktrees = count
	m.metrics.mu.Unlock()
	
	if m.promMetrics != nil {
		m.promMetrics.SetActiveWorktrees(count)
	}
}

// updateDiskUsage 디스크 사용량 업데이트
func (m *AdvancedWorktreeManager) updateDiskUsage() {
	usage := m.calculateTotalDiskUsage()
	m.metrics.mu.Lock()
	m.metrics.DiskUsageBytes = usage
	m.metrics.mu.Unlock()
	
	if m.promMetrics != nil {
		m.promMetrics.SetDiskUsage(usage)
	}
}