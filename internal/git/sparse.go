package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	gogit "github.com/go-git/go-git/v5"
	"github.com/pkg/errors"
)

// configureSparseCheckout sparse-checkout 설정
func configureSparseCheckout(repo *gogit.Repository, worktreePath string, paths []string) error {
	if len(paths) == 0 {
		return nil // sparse checkout 사용 안 함
	}

	// .git/info/sparse-checkout 파일 생성
	gitDir := filepath.Join(worktreePath, ".git")
	infoDir := filepath.Join(gitDir, "info")
	if err := os.MkdirAll(infoDir, 0755); err != nil {
		return errors.Wrap(err, "info 디렉토리 생성 실패")
	}

	sparseCheckoutFile := filepath.Join(infoDir, "sparse-checkout")
	content := strings.Join(paths, "\n") + "\n"
	if err := os.WriteFile(sparseCheckoutFile, []byte(content), 0644); err != nil {
		return errors.Wrap(err, "sparse-checkout 파일 생성 실패")
	}

	// core.sparseCheckout 설정 활성화
	cfg, err := repo.Config()
	if err != nil {
		return errors.Wrap(err, "설정 가져오기 실패")
	}

	cfg.Raw.SetOption("core", "", "sparseCheckout", "true")

	if err := repo.SetConfig(cfg); err != nil {
		return errors.Wrap(err, "설정 저장 실패")
	}

	return nil
}

// validateSparseCheckoutPaths sparse checkout 경로 검증
func validateSparseCheckoutPaths(paths []string) error {
	for _, path := range paths {
		// 절대 경로 금지
		if filepath.IsAbs(path) {
			return fmt.Errorf("절대 경로는 사용할 수 없습니다: %s", path)
		}

		// 상위 디렉토리 참조 금지
		if strings.Contains(path, "..") {
			return fmt.Errorf("상위 디렉토리 참조는 사용할 수 없습니다: %s", path)
		}

		// 빈 경로 금지
		if strings.TrimSpace(path) == "" {
			return fmt.Errorf("빈 경로는 사용할 수 없습니다")
		}
	}
	return nil
}

// SparseCheckoutManager sparse checkout 관리자
type SparseCheckoutManager struct {
	basePath string
}

// NewSparseCheckoutManager 새로운 sparse checkout 관리자 생성
func NewSparseCheckoutManager(basePath string) *SparseCheckoutManager {
	return &SparseCheckoutManager{
		basePath: basePath,
	}
}

// EnableSparseCheckout 기존 저장소에 sparse checkout 활성화
func (m *SparseCheckoutManager) EnableSparseCheckout(repoPath string, paths []string) error {
	// 경로 검증
	if err := validateSparseCheckoutPaths(paths); err != nil {
		return &Error{
			Code:    ErrCodeInvalidRef,
			Message: fmt.Sprintf("잘못된 sparse checkout 경로: %v", err),
		}
	}

	// 저장소 열기
	repo, err := gogit.PlainOpen(repoPath)
	if err != nil {
		return &Error{
			Code:    ErrCodeNotFound,
			Message: fmt.Sprintf("저장소를 열 수 없습니다: %v", err),
		}
	}

	// sparse checkout 설정
	if err := configureSparseCheckout(repo, repoPath, paths); err != nil {
		return &Error{
			Code:    ErrCodeUnknown,
			Message: fmt.Sprintf("sparse checkout 설정 실패: %v", err),
		}
	}

	// worktree 가져오기
	wt, err := repo.Worktree()
	if err != nil {
		return &Error{
			Code:    ErrCodeUnknown,
			Message: fmt.Sprintf("worktree 가져오기 실패: %v", err),
		}
	}

	// 재체크아웃하여 sparse checkout 적용
	if err := wt.Reset(&gogit.ResetOptions{Mode: gogit.HardReset}); err != nil {
		return &Error{
			Code:    ErrCodeUnknown,
			Message: fmt.Sprintf("sparse checkout 적용 실패: %v", err),
		}
	}

	return nil
}

// DisableSparseCheckout sparse checkout 비활성화
func (m *SparseCheckoutManager) DisableSparseCheckout(repoPath string) error {
	// 저장소 열기
	repo, err := gogit.PlainOpen(repoPath)
	if err != nil {
		return &Error{
			Code:    ErrCodeNotFound,
			Message: fmt.Sprintf("저장소를 열 수 없습니다: %v", err),
		}
	}

	// core.sparseCheckout 설정 비활성화
	cfg, err := repo.Config()
	if err != nil {
		return &Error{
			Code:    ErrCodeUnknown,
			Message: fmt.Sprintf("설정 가져오기 실패: %v", err),
		}
	}

	cfg.Raw.SetOption("core", "", "sparseCheckout", "false")

	if err := repo.SetConfig(cfg); err != nil {
		return &Error{
			Code:    ErrCodeUnknown,
			Message: fmt.Sprintf("설정 저장 실패: %v", err),
		}
	}

	// sparse-checkout 파일 삭제
	sparseCheckoutFile := filepath.Join(repoPath, ".git", "info", "sparse-checkout")
	os.Remove(sparseCheckoutFile)

	// 전체 파일 체크아웃
	wt, err := repo.Worktree()
	if err != nil {
		return &Error{
			Code:    ErrCodeUnknown,
			Message: fmt.Sprintf("worktree 가져오기 실패: %v", err),
		}
	}

	if err := wt.Reset(&gogit.ResetOptions{Mode: gogit.HardReset}); err != nil {
		return &Error{
			Code:    ErrCodeUnknown,
			Message: fmt.Sprintf("전체 체크아웃 실패: %v", err),
		}
	}

	return nil
}

// GetSparseCheckoutPaths 현재 sparse checkout 경로 조회
func (m *SparseCheckoutManager) GetSparseCheckoutPaths(repoPath string) ([]string, error) {
	sparseCheckoutFile := filepath.Join(repoPath, ".git", "info", "sparse-checkout")

	// 파일이 없으면 sparse checkout 사용 안 함
	if _, err := os.Stat(sparseCheckoutFile); os.IsNotExist(err) {
		return nil, nil
	}

	content, err := os.ReadFile(sparseCheckoutFile)
	if err != nil {
		return nil, &Error{
			Code:    ErrCodePermission,
			Message: fmt.Sprintf("sparse-checkout 파일 읽기 실패: %v", err),
		}
	}

	lines := strings.Split(string(content), "\n")
	var paths []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			paths = append(paths, line)
		}
	}

	return paths, nil
}

// UpdateSparseCheckoutPaths sparse checkout 경로 업데이트
func (m *SparseCheckoutManager) UpdateSparseCheckoutPaths(repoPath string, paths []string) error {
	// 경로 검증
	if err := validateSparseCheckoutPaths(paths); err != nil {
		return &Error{
			Code:    ErrCodeInvalidRef,
			Message: fmt.Sprintf("잘못된 sparse checkout 경로: %v", err),
		}
	}

	sparseCheckoutFile := filepath.Join(repoPath, ".git", "info", "sparse-checkout")
	content := strings.Join(paths, "\n") + "\n"

	if err := os.WriteFile(sparseCheckoutFile, []byte(content), 0644); err != nil {
		return &Error{
			Code:    ErrCodePermission,
			Message: fmt.Sprintf("sparse-checkout 파일 업데이트 실패: %v", err),
		}
	}

	// 변경사항 적용을 위해 재체크아웃
	repo, err := gogit.PlainOpen(repoPath)
	if err != nil {
		return &Error{
			Code:    ErrCodeNotFound,
			Message: fmt.Sprintf("저장소를 열 수 없습니다: %v", err),
		}
	}

	wt, err := repo.Worktree()
	if err != nil {
		return &Error{
			Code:    ErrCodeUnknown,
			Message: fmt.Sprintf("worktree 가져오기 실패: %v", err),
		}
	}

	if err := wt.Reset(&gogit.ResetOptions{Mode: gogit.HardReset}); err != nil {
		return &Error{
			Code:    ErrCodeUnknown,
			Message: fmt.Sprintf("sparse checkout 적용 실패: %v", err),
		}
	}

	return nil
}
