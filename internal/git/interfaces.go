package git

import (
	"context"
	"time"
)

// WorktreeManager Git worktree 관리 인터페이스
type WorktreeManager interface {
	// 저장소 복제
	Clone(ctx context.Context, url, path string, opts CloneOptions) (*Repository, error)
	
	// Worktree 생성
	CreateWorktree(ctx context.Context, repo *Repository, name string, opts WorktreeOptions) (*Worktree, error)
	
	// Worktree 삭제
	RemoveWorktree(ctx context.Context, repo *Repository, name string) error
	
	// Worktree 목록 조회
	ListWorktrees(ctx context.Context, repo *Repository) ([]*Worktree, error)
	
	// 브랜치 생성 및 체크아웃
	CreateBranch(ctx context.Context, worktree *Worktree, branchName string, baseBranch string) error
	
	// 브랜치 목록 조회
	ListBranches(ctx context.Context, repo *Repository) ([]Branch, error)
	
	// 상태 확인
	GetStatus(ctx context.Context, worktree *Worktree) (*Status, error)
	
	// 정리 작업
	Cleanup(ctx context.Context, repo *Repository) error
}

// Repository Git 저장소 정보
type Repository struct {
	ID           string    // 저장소 고유 ID
	Path         string    // 저장소 경로
	URL          string    // 원격 저장소 URL
	DefaultBranch string   // 기본 브랜치
	CreatedAt    time.Time // 생성 시간
}

// Worktree Git worktree 정보
type Worktree struct {
	ID         string    // Worktree 고유 ID
	Name       string    // Worktree 이름
	Path       string    // Worktree 경로
	Branch     string    // 현재 브랜치
	Repository *Repository // 저장소 참조
	CreatedAt  time.Time // 생성 시간
	IsLocked   bool      // 잠금 상태
	LockReason string    // 잠금 이유
}

// Branch Git 브랜치 정보
type Branch struct {
	Name      string    // 브랜치 이름
	IsRemote  bool      // 원격 브랜치 여부
	IsCurrent bool      // 현재 브랜치 여부
	LastCommit CommitInfo // 마지막 커밋 정보
}

// CommitInfo 커밋 정보
type CommitInfo struct {
	Hash      string    // 커밋 해시
	Author    string    // 작성자
	Email     string    // 이메일
	Message   string    // 커밋 메시지
	Timestamp time.Time // 커밋 시간
}

// Status Git 상태 정보
type Status struct {
	IsClean   bool     // 깨끗한 상태 여부
	Modified  []string // 수정된 파일
	Added     []string // 추가된 파일
	Deleted   []string // 삭제된 파일
	Untracked []string // 추적되지 않는 파일
	Branch    string   // 현재 브랜치
	Ahead     int      // 원격보다 앞선 커밋 수
	Behind    int      // 원격보다 뒤처진 커밋 수
}

// CloneOptions 저장소 복제 옵션
type CloneOptions struct {
	// 인증 정보
	Username string
	Password string
	Token    string
	
	// SSH 키 경로
	SSHKeyPath string
	
	// 복제 옵션
	Depth      int    // 얕은 복제 깊이 (0이면 전체)
	Branch     string // 특정 브랜치만 복제
	SingleBranch bool // 단일 브랜치만 복제
	
	// 프록시 설정
	ProxyURL string
	
	// 타임아웃
	Timeout time.Duration
}

// WorktreeOptions Worktree 생성 옵션
type WorktreeOptions struct {
	// 브랜치 설정
	Branch    string // 체크아웃할 브랜치
	NewBranch string // 새로 생성할 브랜치 이름
	
	// 경로 설정
	Path string // Worktree 경로 (비어있으면 자동 생성)
	
	// 잠금 설정
	Lock       bool   // 생성 시 잠금
	LockReason string // 잠금 이유
	
	// 정리 옵션
	Force bool // 강제 생성 (기존 파일 덮어쓰기)
	
	// 고급 옵션
	SparseCheckoutPaths []string // sparse checkout 경로 (비어있으면 전체 체크아웃)
}

// Error Git 작업 중 발생하는 에러
type Error struct {
	Code    string // 에러 코드
	Message string // 에러 메시지
	Details map[string]interface{} // 추가 정보
}

func (e *Error) Error() string {
	return e.Message
}

// 에러 코드 상수
const (
	ErrCodeNotFound      = "NOT_FOUND"
	ErrCodeAlreadyExists = "ALREADY_EXISTS"
	ErrCodePermission    = "PERMISSION_DENIED"
	ErrCodeInvalidRef    = "INVALID_REF"
	ErrCodeNetwork       = "NETWORK_ERROR"
	ErrCodeAuth          = "AUTH_FAILED"
	ErrCodeLocked        = "WORKTREE_LOCKED"
	ErrCodeDirty         = "WORKTREE_DIRTY"
	ErrCodeUnknown       = "UNKNOWN_ERROR"
)