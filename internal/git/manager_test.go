package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWorktreeManager 테스트를 위한 임시 디렉토리 설정
func setupTestManager(t *testing.T) (*worktreeManager, string) {
	tempDir := t.TempDir()
	manager := NewWorktreeManager(tempDir).(*worktreeManager)
	return manager, tempDir
}

// TestNewWorktreeManager WorktreeManager 생성 테스트
func TestNewWorktreeManager(t *testing.T) {
	tests := []struct {
		name     string
		basePath string
		wantPath string
	}{
		{
			name:     "기본 경로 사용",
			basePath: "",
			wantPath: "/var/lib/aicli/git",
		},
		{
			name:     "사용자 정의 경로",
			basePath: "/tmp/test-git",
			wantPath: "/tmp/test-git",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewWorktreeManager(tt.basePath).(*worktreeManager)
			assert.Equal(t, tt.wantPath, manager.basePath)
			assert.NotNil(t, manager.repos)
		})
	}
}

// TestClone 저장소 복제 테스트
func TestClone(t *testing.T) {
	manager, tempDir := setupTestManager(t)
	ctx := context.Background()

	// 테스트용 Git 저장소 URL (공개 저장소 사용)
	testRepoURL := "https://github.com/git-fixtures/basic.git"

	tests := []struct {
		name    string
		url     string
		path    string
		opts    CloneOptions
		wantErr bool
		errCode string
	}{
		{
			name: "기본 복제",
			url:  testRepoURL,
			path: "",
			opts: CloneOptions{
				Timeout: 30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "지정된 경로로 복제",
			url:  testRepoURL,
			path: filepath.Join(tempDir, "custom-repo"),
			opts: CloneOptions{
				Timeout: 30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "얕은 복제",
			url:  testRepoURL,
			path: "",
			opts: CloneOptions{
				Depth:   1,
				Timeout: 30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "잘못된 URL",
			url:  "https://invalid-url-that-does-not-exist.com/repo.git",
			path: "",
			opts: CloneOptions{
				Timeout: 5 * time.Second,
			},
			wantErr: true,
			errCode: ErrCodeNetwork,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, err := manager.Clone(ctx, tt.url, tt.path, tt.opts)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errCode != "" {
					gitErr, ok := err.(*Error)
					require.True(t, ok)
					assert.Equal(t, tt.errCode, gitErr.Code)
				}
			} else {
				require.NoError(t, err)
				assert.NotNil(t, repo)
				assert.NotEmpty(t, repo.ID)
				assert.NotEmpty(t, repo.Path)
				assert.Equal(t, tt.url, repo.URL)

				// 복제된 저장소 확인
				_, err := os.Stat(repo.Path)
				assert.NoError(t, err)
			}
		})
	}
}

// TestCreateWorktree Worktree 생성 테스트
func TestCreateWorktree(t *testing.T) {
	manager, _ := setupTestManager(t)
	ctx := context.Background()

	// 먼저 테스트용 저장소 복제
	testRepoURL := "https://github.com/git-fixtures/basic.git"
	repo, err := manager.Clone(ctx, testRepoURL, "", CloneOptions{Timeout: 30 * time.Second})
	require.NoError(t, err)

	tests := []struct {
		name    string
		repo    *Repository
		wtName  string
		opts    WorktreeOptions
		wantErr bool
	}{
		{
			name:    "기본 Worktree 생성",
			repo:    repo,
			wtName:  "feature-1",
			opts:    WorktreeOptions{},
			wantErr: false,
		},
		{
			name:   "새 브랜치로 Worktree 생성",
			repo:   repo,
			wtName: "feature-2",
			opts: WorktreeOptions{
				NewBranch: "feature/new-branch",
			},
			wantErr: false,
		},
		{
			name:   "지정된 경로에 Worktree 생성",
			repo:   repo,
			wtName: "feature-3",
			opts: WorktreeOptions{
				Path: filepath.Join(manager.basePath, "custom-worktree"),
			},
			wantErr: false,
		},
		{
			name:   "잠금 설정된 Worktree 생성",
			repo:   repo,
			wtName: "feature-4",
			opts: WorktreeOptions{
				Lock:       true,
				LockReason: "작업 진행 중",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			worktree, err := manager.CreateWorktree(ctx, tt.repo, tt.wtName, tt.opts)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, worktree)
				assert.NotEmpty(t, worktree.ID)
				assert.Equal(t, tt.wtName, worktree.Name)
				assert.NotEmpty(t, worktree.Path)
				assert.Equal(t, tt.repo, worktree.Repository)

				if tt.opts.Lock {
					assert.True(t, worktree.IsLocked)
					assert.Equal(t, tt.opts.LockReason, worktree.LockReason)
				}
			}
		})
	}
}

// TestListWorktrees Worktree 목록 조회 테스트
func TestListWorktrees(t *testing.T) {
	manager, _ := setupTestManager(t)
	ctx := context.Background()

	// 테스트용 저장소 복제
	testRepoURL := "https://github.com/git-fixtures/basic.git"
	repo, err := manager.Clone(ctx, testRepoURL, "", CloneOptions{Timeout: 30 * time.Second})
	require.NoError(t, err)

	// 여러 Worktree 생성
	worktreeNames := []string{"wt-1", "wt-2", "wt-3"}
	for _, name := range worktreeNames {
		_, err := manager.CreateWorktree(ctx, repo, name, WorktreeOptions{})
		require.NoError(t, err)
	}

	// Worktree 목록 조회
	worktrees, err := manager.ListWorktrees(ctx, repo)
	require.NoError(t, err)
	assert.Len(t, worktrees, len(worktreeNames))

	// 각 Worktree 확인
	foundNames := make(map[string]bool)
	for _, wt := range worktrees {
		foundNames[wt.Name] = true
		assert.NotEmpty(t, wt.Path)
		assert.Equal(t, repo, wt.Repository)
	}

	// 모든 이름이 있는지 확인
	for _, name := range worktreeNames {
		assert.True(t, foundNames[name], "Worktree %s를 찾을 수 없음", name)
	}
}

// TestRemoveWorktree Worktree 삭제 테스트
func TestRemoveWorktree(t *testing.T) {
	manager, _ := setupTestManager(t)
	ctx := context.Background()

	// 테스트용 저장소 복제
	testRepoURL := "https://github.com/git-fixtures/basic.git"
	repo, err := manager.Clone(ctx, testRepoURL, "", CloneOptions{Timeout: 30 * time.Second})
	require.NoError(t, err)

	// Worktree 생성
	worktreeName := "test-worktree"
	wt, err := manager.CreateWorktree(ctx, repo, worktreeName, WorktreeOptions{})
	require.NoError(t, err)

	// Worktree 경로 확인
	_, err = os.Stat(wt.Path)
	require.NoError(t, err)

	// Worktree 삭제
	err = manager.RemoveWorktree(ctx, repo, worktreeName)
	require.NoError(t, err)

	// Worktree 경로가 삭제되었는지 확인
	_, err = os.Stat(wt.Path)
	assert.True(t, os.IsNotExist(err))

	// 존재하지 않는 Worktree 삭제 시도
	err = manager.RemoveWorktree(ctx, repo, "non-existent")
	require.Error(t, err)
	gitErr, ok := err.(*Error)
	require.True(t, ok)
	assert.Equal(t, ErrCodeNotFound, gitErr.Code)
}

// TestCreateBranch 브랜치 생성 테스트
func TestCreateBranch(t *testing.T) {
	manager, _ := setupTestManager(t)
	ctx := context.Background()

	// 테스트용 저장소 복제
	testRepoURL := "https://github.com/git-fixtures/basic.git"
	repo, err := manager.Clone(ctx, testRepoURL, "", CloneOptions{Timeout: 30 * time.Second})
	require.NoError(t, err)

	// Worktree 생성
	worktree, err := manager.CreateWorktree(ctx, repo, "test-wt", WorktreeOptions{})
	require.NoError(t, err)

	tests := []struct {
		name       string
		branchName string
		baseBranch string
		wantErr    bool
	}{
		{
			name:       "현재 브랜치 기준으로 새 브랜치 생성",
			branchName: "feature/test-1",
			baseBranch: "",
			wantErr:    false,
		},
		{
			name:       "특정 브랜치 기준으로 새 브랜치 생성",
			branchName: "feature/test-2",
			baseBranch: "master",
			wantErr:    false,
		},
		{
			name:       "중복된 브랜치 이름",
			branchName: "feature/test-1",
			baseBranch: "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.CreateBranch(ctx, worktree, tt.branchName, tt.baseBranch)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.branchName, worktree.Branch)
			}
		})
	}
}

// TestListBranches 브랜치 목록 조회 테스트
func TestListBranches(t *testing.T) {
	manager, _ := setupTestManager(t)
	ctx := context.Background()

	// 테스트용 저장소 복제
	testRepoURL := "https://github.com/git-fixtures/basic.git"
	repo, err := manager.Clone(ctx, testRepoURL, "", CloneOptions{Timeout: 30 * time.Second})
	require.NoError(t, err)

	// 브랜치 목록 조회
	branches, err := manager.ListBranches(ctx, repo)
	require.NoError(t, err)
	assert.NotEmpty(t, branches)

	// 기본 브랜치 확인
	var hasCurrentBranch bool
	for _, branch := range branches {
		assert.NotEmpty(t, branch.Name)
		if branch.IsCurrent {
			hasCurrentBranch = true
		}
		if !branch.IsRemote {
			assert.NotEmpty(t, branch.LastCommit.Hash)
			assert.NotEmpty(t, branch.LastCommit.Author)
		}
	}
	assert.True(t, hasCurrentBranch, "현재 브랜치를 찾을 수 없음")
}

// TestGetStatus 상태 확인 테스트
func TestGetStatus(t *testing.T) {
	manager, _ := setupTestManager(t)
	ctx := context.Background()

	// 테스트용 저장소 복제
	testRepoURL := "https://github.com/git-fixtures/basic.git"
	repo, err := manager.Clone(ctx, testRepoURL, "", CloneOptions{Timeout: 30 * time.Second})
	require.NoError(t, err)

	// Worktree 생성
	worktree, err := manager.CreateWorktree(ctx, repo, "test-status", WorktreeOptions{})
	require.NoError(t, err)

	// 초기 상태 확인 (깨끗한 상태)
	status, err := manager.GetStatus(ctx, worktree)
	require.NoError(t, err)
	assert.True(t, status.IsClean)
	assert.Empty(t, status.Modified)
	assert.Empty(t, status.Added)
	assert.Empty(t, status.Deleted)
	assert.Empty(t, status.Untracked)

	// 새 파일 생성
	testFile := filepath.Join(worktree.Path, "test-file.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	// 변경된 상태 확인
	status, err = manager.GetStatus(ctx, worktree)
	require.NoError(t, err)
	assert.False(t, status.IsClean)
	assert.Contains(t, status.Untracked, "test-file.txt")
}

// TestCleanup 정리 작업 테스트
func TestCleanup(t *testing.T) {
	manager, _ := setupTestManager(t)
	ctx := context.Background()

	// 테스트용 저장소 복제
	testRepoURL := "https://github.com/git-fixtures/basic.git"
	repo, err := manager.Clone(ctx, testRepoURL, "", CloneOptions{Timeout: 30 * time.Second})
	require.NoError(t, err)

	// Worktree 생성
	_, err = manager.CreateWorktree(ctx, repo, "test-cleanup", WorktreeOptions{})
	require.NoError(t, err)

	// Worktree 디렉토리 확인
	worktreesPath := filepath.Join(manager.basePath, "worktrees", repo.ID)
	_, err = os.Stat(worktreesPath)
	require.NoError(t, err)

	// 캐시 확인
	manager.mu.RLock()
	_, exists := manager.repos[repo.ID]
	manager.mu.RUnlock()
	assert.True(t, exists)

	// 정리 작업 수행
	err = manager.Cleanup(ctx, repo)
	require.NoError(t, err)

	// Worktree 디렉토리가 삭제되었는지 확인
	_, err = os.Stat(worktreesPath)
	assert.True(t, os.IsNotExist(err))

	// 캐시에서 제거되었는지 확인
	manager.mu.RLock()
	_, exists = manager.repos[repo.ID]
	manager.mu.RUnlock()
	assert.False(t, exists)
}

// TestGetAuth 인증 설정 테스트
func TestGetAuth(t *testing.T) {
	manager, _ := setupTestManager(t)

	tests := []struct {
		name    string
		opts    CloneOptions
		wantNil bool
	}{
		{
			name: "Token 인증",
			opts: CloneOptions{
				Token: "test-token",
			},
			wantNil: false,
		},
		{
			name: "사용자명/비밀번호 인증",
			opts: CloneOptions{
				Username: "user",
				Password: "pass",
			},
			wantNil: false,
		},
		{
			name:    "인증 없음",
			opts:    CloneOptions{},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth, err := manager.getAuth(tt.opts)
			require.NoError(t, err)

			if tt.wantNil {
				assert.Nil(t, auth)
			} else {
				assert.NotNil(t, auth)
			}
		})
	}
}
