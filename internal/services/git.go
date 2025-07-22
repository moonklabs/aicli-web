package services

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/aicli/aicli-web/internal/models"
)

// GitService Git 관련 작업을 처리하는 서비스
type GitService struct{}

// NewGitService 새 Git 서비스 생성
func NewGitService() *GitService {
	return &GitService{}
}

// GetGitInfo 프로젝트의 Git 정보를 조회
func (s *GitService) GetGitInfo(projectPath string) (*models.GitInfo, error) {
	// Git 리포지토리인지 확인
	if !s.isGitRepository(projectPath) {
		return nil, nil
	}

	gitInfo := &models.GitInfo{}

	// 원격 URL 가져오기
	remoteURL, err := s.getRemoteURL(projectPath)
	if err == nil {
		gitInfo.RemoteURL = remoteURL
	}

	// 현재 브랜치 가져오기
	branch, err := s.getCurrentBranch(projectPath)
	if err == nil {
		gitInfo.CurrentBranch = branch
	}

	// 상태 확인
	status, err := s.getStatus(projectPath)
	if err == nil {
		gitInfo.Status = status
		gitInfo.IsClean = !status.HasChanges
	}

	// 마지막 커밋 정보
	commit, err := s.getLastCommit(projectPath)
	if err == nil {
		gitInfo.LastCommit = commit
	}

	return gitInfo, nil
}

// isGitRepository Git 리포지토리인지 확인
func (s *GitService) isGitRepository(path string) bool {
	gitDir := filepath.Join(path, ".git")
	info, err := os.Stat(gitDir)
	return err == nil && info.IsDir()
}

// getRemoteURL 원격 리포지토리 URL 가져오기
func (s *GitService) getRemoteURL(path string) (string, error) {
	cmd := exec.Command("git", "config", "--get", "remote.origin.url")
	cmd.Dir = path
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// getCurrentBranch 현재 브랜치 가져오기
func (s *GitService) getCurrentBranch(path string) (string, error) {
	cmd := exec.Command("git", "branch", "--show-current")
	cmd.Dir = path
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// getStatus Git 상태 가져오기
func (s *GitService) getStatus(path string) (*models.GitStatus, error) {
	status := &models.GitStatus{
		Modified:  []string{},
		Added:     []string{},
		Deleted:   []string{},
		Untracked: []string{},
	}

	// git status --porcelain 명령 실행
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = path
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		// 상태 코드와 파일명 분리
		if len(line) < 3 {
			continue
		}

		statusCode := line[:2]
		fileName := strings.TrimSpace(line[3:])

		switch {
		case strings.Contains(statusCode, "M"):
			status.Modified = append(status.Modified, fileName)
		case strings.Contains(statusCode, "A"):
			status.Added = append(status.Added, fileName)
		case strings.Contains(statusCode, "D"):
			status.Deleted = append(status.Deleted, fileName)
		case strings.Contains(statusCode, "?"):
			status.Untracked = append(status.Untracked, fileName)
		}
	}

	status.HasChanges = len(status.Modified) > 0 || len(status.Added) > 0 ||
		len(status.Deleted) > 0 || len(status.Untracked) > 0

	return status, nil
}

// getLastCommit 마지막 커밋 정보 가져오기
func (s *GitService) getLastCommit(path string) (*models.CommitInfo, error) {
	// 커밋 해시
	hashCmd := exec.Command("git", "rev-parse", "HEAD")
	hashCmd.Dir = path
	hashOutput, err := hashCmd.Output()
	if err != nil {
		return nil, err
	}
	hash := strings.TrimSpace(string(hashOutput))

	// 커밋 정보
	logCmd := exec.Command("git", "log", "-1", "--format=%an|%s|%ct")
	logCmd.Dir = path
	logOutput, err := logCmd.Output()
	if err != nil {
		return nil, err
	}

	parts := strings.Split(strings.TrimSpace(string(logOutput)), "|")
	if len(parts) != 3 {
		return nil, fmt.Errorf("unexpected git log format")
	}

	// Unix timestamp를 time.Time으로 변환
	var timestamp int64
	fmt.Sscanf(parts[2], "%d", &timestamp)

	return &models.CommitInfo{
		Hash:      hash,
		Author:    parts[0],
		Message:   parts[1],
		Timestamp: time.Unix(timestamp, 0),
	}, nil
}

// InitRepository 새 Git 리포지토리 초기화
func (s *GitService) InitRepository(path string) error {
	cmd := exec.Command("git", "init")
	cmd.Dir = path
	return cmd.Run()
}

// CloneRepository 리포지토리 복제
func (s *GitService) CloneRepository(url, path string) error {
	cmd := exec.Command("git", "clone", url, path)
	return cmd.Run()
}