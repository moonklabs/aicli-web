package git

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

// worktreeManager go-git 기반 WorktreeManager 구현체
// go-git v5는 multiple worktree를 직접 지원하지 않으므로
// 별도의 저장소 복제로 worktree 기능을 구현합니다.
type worktreeManager struct {
	// 저장소 캐시
	repos map[string]*gogit.Repository
	mu    sync.RWMutex

	// 기본 경로
	basePath string

	// 정리 간격
	cleanupInterval time.Duration
}

// NewWorktreeManager 새로운 WorktreeManager 생성
func NewWorktreeManager(basePath string) WorktreeManager {
	if basePath == "" {
		basePath = "/var/lib/aicli/git"
	}

	manager := &worktreeManager{
		repos:           make(map[string]*gogit.Repository),
		basePath:        basePath,
		cleanupInterval: 30 * time.Minute,
	}

	// 기본 경로 생성
	os.MkdirAll(basePath, 0755)

	return manager
}

// Clone 저장소 복제
func (m *worktreeManager) Clone(ctx context.Context, url, path string, opts CloneOptions) (*Repository, error) {
	// 경로 검증
	if path == "" {
		path = filepath.Join(m.basePath, "repos", uuid.New().String())
	}

	// 인증 설정
	auth, err := m.getAuth(opts)
	if err != nil {
		return nil, &Error{
			Code:    ErrCodeAuth,
			Message: fmt.Sprintf("인증 설정 실패: %v", err),
		}
	}

	// 복제 옵션 설정
	cloneOpts := &gogit.CloneOptions{
		URL:          url,
		Auth:         auth,
		Progress:     os.Stdout,
		SingleBranch: opts.SingleBranch,
	}

	if opts.Branch != "" {
		cloneOpts.ReferenceName = plumbing.NewBranchReferenceName(opts.Branch)
	}

	if opts.Depth > 0 {
		cloneOpts.Depth = opts.Depth
	}

	// 타임아웃 처리
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	// 저장소 복제
	gogitRepo, err := gogit.PlainCloneContext(ctx, path, false, cloneOpts)
	if err != nil {
		return nil, &Error{
			Code:    ErrCodeNetwork,
			Message: fmt.Sprintf("저장소 복제 실패: %v", err),
			Details: map[string]interface{}{
				"url":  url,
				"path": path,
			},
		}
	}

	// 저장소 정보 생성
	repo := &Repository{
		ID:           uuid.New().String(),
		Path:         path,
		URL:          url,
		DefaultBranch: opts.Branch,
		CreatedAt:    time.Now(),
	}

	// 기본 브랜치 확인
	if repo.DefaultBranch == "" {
		head, err := gogitRepo.Head()
		if err == nil {
			repo.DefaultBranch = head.Name().Short()
		}
	}

	// 캐시에 저장
	m.mu.Lock()
	m.repos[repo.ID] = gogitRepo
	m.mu.Unlock()

	return repo, nil
}

// CreateWorktree Worktree 생성 (별도 저장소 복제로 구현)
func (m *worktreeManager) CreateWorktree(ctx context.Context, repo *Repository, name string, opts WorktreeOptions) (*Worktree, error) {
	// go-git v5는 multiple worktree를 직접 지원하지 않으므로
	// 별도의 저장소 복제로 worktree 기능을 구현합니다.

	// Worktree 경로 설정
	worktreePath := opts.Path
	if worktreePath == "" {
		worktreePath = filepath.Join(m.basePath, "worktrees", repo.ID, name)
	}

	// 디렉토리 생성
	if err := os.MkdirAll(filepath.Dir(worktreePath), 0755); err != nil {
		return nil, &Error{
			Code:    ErrCodePermission,
			Message: fmt.Sprintf("디렉토리 생성 실패: %v", err),
		}
	}

	// 원본 저장소에서 복제
	cloneOpts := &gogit.CloneOptions{
		URL: repo.Path, // 로컬 경로에서 복제
	}

	// 저장소 복제
	wtRepo, err := gogit.PlainClone(worktreePath, false, cloneOpts)
	if err != nil {
		return nil, &Error{
			Code:    ErrCodeUnknown,
			Message: fmt.Sprintf("Worktree 생성을 위한 복제 실패: %v", err),
		}
	}

	// Worktree 가져오기
	wt, err := wtRepo.Worktree()
	if err != nil {
		return nil, &Error{
			Code:    ErrCodeUnknown,
			Message: fmt.Sprintf("Worktree 가져오기 실패: %v", err),
		}
	}

	// 브랜치 처리
	if opts.NewBranch != "" {
		// 새 브랜치 생성 및 체크아웃
		branchRef := plumbing.NewBranchReferenceName(opts.NewBranch)
		
		// HEAD 커밋 가져오기
		head, err := wtRepo.Head()
		if err != nil {
			return nil, &Error{
				Code:    ErrCodeInvalidRef,
				Message: "HEAD 참조를 가져올 수 없습니다",
			}
		}

		// 브랜치가 이미 존재하는지 확인
		if _, err := wtRepo.Reference(branchRef, false); err == nil {
			return nil, &Error{
				Code:    ErrCodeAlreadyExists,
				Message: fmt.Sprintf("브랜치가 이미 존재합니다: %s", opts.NewBranch),
			}
		}
		
		// 새 브랜치 생성
		ref := plumbing.NewHashReference(branchRef, head.Hash())
		err = wtRepo.Storer.SetReference(ref)
		if err != nil {
			return nil, &Error{
				Code:    ErrCodeUnknown,
				Message: fmt.Sprintf("브랜치 생성 실패: %v", err),
			}
		}

		// 체크아웃
		err = wt.Checkout(&gogit.CheckoutOptions{
			Branch: branchRef,
		})
		if err != nil {
			return nil, &Error{
				Code:    ErrCodeUnknown,
				Message: fmt.Sprintf("브랜치 체크아웃 실패: %v", err),
			}
		}
	} else if opts.Branch != "" {
		// 기존 브랜치로 체크아웃
		branchRef := plumbing.NewBranchReferenceName(opts.Branch)
		err = wt.Checkout(&gogit.CheckoutOptions{
			Branch: branchRef,
		})
		if err != nil {
			return nil, &Error{
				Code:    ErrCodeInvalidRef,
				Message: fmt.Sprintf("브랜치 체크아웃 실패: %v", err),
			}
		}
	}

	// 현재 브랜치 확인
	head, err := wtRepo.Head()
	currentBranch := ""
	if err == nil {
		currentBranch = head.Name().Short()
	}

	// Sparse checkout 설정
	if len(opts.SparseCheckoutPaths) > 0 {
		sparseManager := NewSparseCheckoutManager(m.basePath)
		if err := sparseManager.EnableSparseCheckout(worktreePath, opts.SparseCheckoutPaths); err != nil {
			// 실패 시 worktree 삭제
			os.RemoveAll(worktreePath)
			return nil, err
		}
	}

	// Worktree 정보 생성
	worktree := &Worktree{
		ID:         uuid.New().String(),
		Name:       name,
		Path:       worktreePath,
		Branch:     currentBranch,
		Repository: repo,
		CreatedAt:  time.Now(),
		IsLocked:   opts.Lock,
		LockReason: opts.LockReason,
	}

	return worktree, nil
}

// RemoveWorktree Worktree 삭제
func (m *worktreeManager) RemoveWorktree(ctx context.Context, repo *Repository, name string) error {
	// Worktree 경로 계산
	worktreePath := filepath.Join(m.basePath, "worktrees", repo.ID, name)
	
	// 디렉토리 존재 확인
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		return &Error{
			Code:    ErrCodeNotFound,
			Message: fmt.Sprintf("Worktree를 찾을 수 없습니다: %s", name),
		}
	}

	// 디렉토리 삭제
	if err := os.RemoveAll(worktreePath); err != nil {
		return &Error{
			Code:    ErrCodePermission,
			Message: fmt.Sprintf("Worktree 삭제 실패: %v", err),
		}
	}

	return nil
}

// ListWorktrees Worktree 목록 조회
func (m *worktreeManager) ListWorktrees(ctx context.Context, repo *Repository) ([]*Worktree, error) {
	// Worktree 디렉토리 경로
	worktreesPath := filepath.Join(m.basePath, "worktrees", repo.ID)
	
	// 디렉토리 존재 확인
	entries, err := os.ReadDir(worktreesPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []*Worktree{}, nil
		}
		return nil, &Error{
			Code:    ErrCodePermission,
			Message: fmt.Sprintf("Worktree 디렉토리 읽기 실패: %v", err),
		}
	}

	worktrees := make([]*Worktree, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		wtPath := filepath.Join(worktreesPath, entry.Name())
		
		// 저장소 열기 시도
		wtRepo, err := gogit.PlainOpen(wtPath)
		if err != nil {
			continue // 유효하지 않은 worktree는 건너뛰기
		}

		// 현재 브랜치 확인
		head, err := wtRepo.Head()
		currentBranch := ""
		if err == nil {
			currentBranch = head.Name().Short()
		}

		worktree := &Worktree{
			Name:       entry.Name(),
			Path:       wtPath,
			Branch:     currentBranch,
			Repository: repo,
		}
		worktrees = append(worktrees, worktree)
	}

	return worktrees, nil
}

// CreateBranch 브랜치 생성 및 체크아웃
func (m *worktreeManager) CreateBranch(ctx context.Context, worktree *Worktree, branchName string, baseBranch string) error {
	// Worktree의 저장소 열기
	gogitRepo, err := gogit.PlainOpen(worktree.Path)
	if err != nil {
		return &Error{
			Code:    ErrCodeNotFound,
			Message: fmt.Sprintf("Worktree 열기 실패: %v", err),
		}
	}

	// Worktree 가져오기
	wt, err := gogitRepo.Worktree()
	if err != nil {
		return &Error{
			Code:    ErrCodeUnknown,
			Message: fmt.Sprintf("Worktree 가져오기 실패: %v", err),
		}
	}

	// 기준 브랜치 참조 가져오기
	var baseRef plumbing.ReferenceName
	if baseBranch != "" {
		baseRef = plumbing.NewBranchReferenceName(baseBranch)
	} else {
		// 현재 브랜치 사용
		head, err := gogitRepo.Head()
		if err != nil {
			return &Error{
				Code:    ErrCodeInvalidRef,
				Message: "HEAD 참조를 가져올 수 없습니다",
			}
		}
		baseRef = head.Name()
	}

	// 기준 브랜치의 커밋 해시 가져오기
	baseCommit, err := gogitRepo.ResolveRevision(plumbing.Revision(baseRef))
	if err != nil {
		return &Error{
			Code:    ErrCodeInvalidRef,
			Message: fmt.Sprintf("기준 브랜치를 찾을 수 없습니다: %s", baseBranch),
		}
	}

	// 새 브랜치 생성
	newBranchRef := plumbing.NewBranchReferenceName(branchName)
	
	// 브랜치가 이미 존재하는지 확인
	if _, err := gogitRepo.Reference(newBranchRef, false); err == nil {
		return &Error{
			Code:    ErrCodeAlreadyExists,
			Message: fmt.Sprintf("브랜치가 이미 존재합니다: %s", branchName),
		}
	}
	
	ref := plumbing.NewHashReference(newBranchRef, *baseCommit)
	
	err = gogitRepo.Storer.SetReference(ref)
	if err != nil {
		return &Error{
			Code:    ErrCodeUnknown,
			Message: fmt.Sprintf("브랜치 생성 실패: %v", err),
		}
	}

	// 새 브랜치로 체크아웃
	err = wt.Checkout(&gogit.CheckoutOptions{
		Branch: newBranchRef,
		Create: false,
	})
	if err != nil {
		return &Error{
			Code:    ErrCodeUnknown,
			Message: fmt.Sprintf("브랜치 체크아웃 실패: %v", err),
		}
	}

	// Worktree 정보 업데이트
	worktree.Branch = branchName

	return nil
}

// ListBranches 브랜치 목록 조회
func (m *worktreeManager) ListBranches(ctx context.Context, repo *Repository) ([]Branch, error) {
	// 저장소 가져오기
	gogitRepo, err := m.getRepository(repo)
	if err != nil {
		return nil, err
	}

	// 현재 브랜치 확인
	head, err := gogitRepo.Head()
	if err != nil {
		return nil, &Error{
			Code:    ErrCodeUnknown,
			Message: fmt.Sprintf("HEAD 참조 가져오기 실패: %v", err),
		}
	}
	currentBranch := head.Name().Short()

	branches := []Branch{}

	// 로컬 브랜치 조회
	branchRefs, err := gogitRepo.Branches()
	if err != nil {
		return nil, &Error{
			Code:    ErrCodeUnknown,
			Message: fmt.Sprintf("브랜치 목록 조회 실패: %v", err),
		}
	}

	err = branchRefs.ForEach(func(ref *plumbing.Reference) error {
		// 커밋 정보 가져오기
		commit, err := gogitRepo.CommitObject(ref.Hash())
		if err != nil {
			return nil // 계속 진행
		}

		branch := Branch{
			Name:      ref.Name().Short(),
			IsRemote:  false,
			IsCurrent: ref.Name().Short() == currentBranch,
			LastCommit: CommitInfo{
				Hash:      commit.Hash.String(),
				Author:    commit.Author.Name,
				Email:     commit.Author.Email,
				Message:   commit.Message,
				Timestamp: commit.Author.When,
			},
		}
		branches = append(branches, branch)
		return nil
	})

	if err != nil {
		return nil, &Error{
			Code:    ErrCodeUnknown,
			Message: fmt.Sprintf("브랜치 정보 처리 실패: %v", err),
		}
	}

	// 원격 브랜치 조회
	remotes, err := gogitRepo.Remotes()
	if err == nil && len(remotes) > 0 {
		// 원격 브랜치 참조 가져오기
		remote := remotes[0]
		refs, err := remote.List(&gogit.ListOptions{})
		if err == nil {
			for _, ref := range refs {
				if ref.Name().IsBranch() {
					branches = append(branches, Branch{
						Name:     ref.Name().Short(),
						IsRemote: true,
					})
				}
			}
		}
	}

	return branches, nil
}

// GetStatus 상태 확인
func (m *worktreeManager) GetStatus(ctx context.Context, worktree *Worktree) (*Status, error) {
	// Worktree의 저장소 열기
	gogitRepo, err := gogit.PlainOpen(worktree.Path)
	if err != nil {
		return nil, &Error{
			Code:    ErrCodeNotFound,
			Message: fmt.Sprintf("Worktree 열기 실패: %v", err),
		}
	}

	// Worktree 가져오기
	wt, err := gogitRepo.Worktree()
	if err != nil {
		return nil, &Error{
			Code:    ErrCodeUnknown,
			Message: fmt.Sprintf("Worktree 가져오기 실패: %v", err),
		}
	}

	// 상태 가져오기
	gogitStatus, err := wt.Status()
	if err != nil {
		return nil, &Error{
			Code:    ErrCodeUnknown,
			Message: fmt.Sprintf("상태 조회 실패: %v", err),
		}
	}

	// 상태 정보 변환
	status := &Status{
		IsClean:   gogitStatus.IsClean(),
		Modified:  []string{},
		Added:     []string{},
		Deleted:   []string{},
		Untracked: []string{},
		Branch:    worktree.Branch,
	}

	// 파일별 상태 확인
	for file, fileStatus := range gogitStatus {
		switch fileStatus.Worktree {
		case gogit.Modified:
			status.Modified = append(status.Modified, file)
		case gogit.Added:
			status.Added = append(status.Added, file)
		case gogit.Deleted:
			status.Deleted = append(status.Deleted, file)
		case gogit.Untracked:
			status.Untracked = append(status.Untracked, file)
		}
	}

	return status, nil
}

// Cleanup 정리 작업
func (m *worktreeManager) Cleanup(ctx context.Context, repo *Repository) error {
	// 캐시에서 제거
	m.mu.Lock()
	delete(m.repos, repo.ID)
	m.mu.Unlock()

	// Worktree 디렉토리 정리
	worktreesPath := filepath.Join(m.basePath, "worktrees", repo.ID)
	if err := os.RemoveAll(worktreesPath); err != nil {
		return &Error{
			Code:    ErrCodePermission,
			Message: fmt.Sprintf("Worktree 디렉토리 삭제 실패: %v", err),
		}
	}

	return nil
}

// getRepository 캐시된 저장소 가져오기
func (m *worktreeManager) getRepository(repo *Repository) (*gogit.Repository, error) {
	m.mu.RLock()
	gogitRepo, exists := m.repos[repo.ID]
	m.mu.RUnlock()

	if !exists {
		// 캐시에 없으면 열기 시도
		var err error
		gogitRepo, err = gogit.PlainOpen(repo.Path)
		if err != nil {
			return nil, &Error{
				Code:    ErrCodeNotFound,
				Message: fmt.Sprintf("저장소를 열 수 없습니다: %v", err),
			}
		}

		// 캐시에 저장
		m.mu.Lock()
		m.repos[repo.ID] = gogitRepo
		m.mu.Unlock()
	}

	return gogitRepo, nil
}

// getAuth 인증 정보 설정
func (m *worktreeManager) getAuth(opts CloneOptions) (transport.AuthMethod, error) {
	// SSH 키 사용
	if opts.SSHKeyPath != "" {
		auth, err := ssh.NewPublicKeysFromFile("git", opts.SSHKeyPath, "")
		if err != nil {
			return nil, errors.Wrap(err, "SSH 키 로드 실패")
		}
		return auth, nil
	}

	// Token 사용 (GitHub 등)
	if opts.Token != "" {
		return &http.BasicAuth{
			Username: "token",
			Password: opts.Token,
		}, nil
	}

	// 사용자명/비밀번호 사용
	if opts.Username != "" && opts.Password != "" {
		return &http.BasicAuth{
			Username: opts.Username,
			Password: opts.Password,
		}, nil
	}

	// 인증 없음
	return nil, nil
}