package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aicli/aicli-web/internal/git"
	"github.com/aicli/aicli-web/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// GitIntegrationTestSuite Git 통합 테스트 스위트
// Git worktree 관리, 브랜치 작업, 동시성 등을 테스트합니다.
type GitIntegrationTestSuite struct {
	suite.Suite
	ctx            context.Context
	cancel         context.CancelFunc
	gitManager     *git.Manager
	testRepoDir    string
	testWorktrees  []string
	cleanupFunctions []func()
}

// SetupSuite 테스트 스위트 초기화
func (suite *GitIntegrationTestSuite) SetupSuite() {
	suite.ctx, suite.cancel = context.WithTimeout(context.Background(), 10*time.Minute)
	
	// Git 매니저 초기화
	suite.gitManager = git.NewManager()
	
	// 테스트용 Git 저장소 생성
	suite.testRepoDir = testutil.TempDir(suite.T(), "git-integration-test")
	suite.setupTestRepository()
	
	suite.T().Logf("Git 통합 테스트 환경 초기화 완료 - 저장소: %s", suite.testRepoDir)
}

// TearDownSuite 테스트 스위트 정리
func (suite *GitIntegrationTestSuite) TearDownSuite() {
	// 모든 worktree 정리
	suite.cleanupTestWorktrees()
	
	// 정리 함수들 실행
	for i := len(suite.cleanupFunctions) - 1; i >= 0; i-- {
		suite.cleanupFunctions[i]()
	}
	
	if suite.cancel != nil {
		suite.cancel()
	}
}

// SetupTest 각 테스트 초기화
func (suite *GitIntegrationTestSuite) SetupTest() {
	suite.testWorktrees = make([]string, 0)
}

// TearDownTest 각 테스트 정리
func (suite *GitIntegrationTestSuite) TearDownTest() {
	suite.cleanupTestWorktrees()
}

// TestGitWorktreeLifecycle Git worktree 생명주기 테스트
func (suite *GitIntegrationTestSuite) TestGitWorktreeLifecycle() {
	suite.T().Log("🔄 Git worktree 생명주기 테스트 시작")

	branchName := fmt.Sprintf("test-branch-%d", time.Now().Unix())
	worktreePath := filepath.Join(suite.testRepoDir, "worktrees", branchName)

	// 1. 새 브랜치 생성
	suite.T().Logf("   🌿 새 브랜치 생성: %s", branchName)
	err := suite.gitManager.CreateBranch(suite.ctx, suite.testRepoDir, branchName, "main")
	require.NoError(suite.T(), err)

	// 2. Worktree 생성
	suite.T().Logf("   📁 Worktree 생성: %s", worktreePath)
	err = suite.gitManager.CreateWorktree(suite.ctx, suite.testRepoDir, worktreePath, branchName)
	require.NoError(suite.T(), err)
	suite.testWorktrees = append(suite.testWorktrees, worktreePath)

	// Worktree 디렉토리가 생성되었는지 확인
	assert.DirExists(suite.T(), worktreePath)
	
	// 3. Worktree에서 파일 작업
	suite.T().Log("   📝 Worktree에서 파일 작업 테스트")
	testFilePath := filepath.Join(worktreePath, "test-file.txt")
	testContent := fmt.Sprintf("Test content created at %s", time.Now().Format(time.RFC3339))
	
	err = os.WriteFile(testFilePath, []byte(testContent), 0644)
	require.NoError(suite.T(), err)
	
	// 4. Git 상태 확인
	suite.T().Log("   📊 Git 상태 확인")
	status, err := suite.gitManager.GetWorktreeStatus(suite.ctx, worktreePath)
	require.NoError(suite.T(), err)
	assert.NotNil(suite.T(), status)
	assert.True(suite.T(), status.HasChanges, "파일 변경이 감지되어야 함")
	
	// 5. 파일 커밋
	suite.T().Log("   💾 파일 커밋")
	err = suite.gitManager.AddFiles(suite.ctx, worktreePath, []string{"test-file.txt"})
	require.NoError(suite.T(), err)
	
	commitMsg := fmt.Sprintf("Add test file in worktree %s", branchName)
	err = suite.gitManager.Commit(suite.ctx, worktreePath, commitMsg)
	require.NoError(suite.T(), err)
	
	// 커밋 후 상태 확인
	statusAfterCommit, err := suite.gitManager.GetWorktreeStatus(suite.ctx, worktreePath)
	require.NoError(suite.T(), err)
	assert.False(suite.T(), statusAfterCommit.HasChanges, "커밋 후에는 변경사항이 없어야 함")
	
	// 6. Worktree 정보 조회
	suite.T().Log("   📖 Worktree 정보 조회")
	worktreeInfo, err := suite.gitManager.GetWorktreeInfo(suite.ctx, worktreePath)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), branchName, worktreeInfo.Branch)
	assert.Equal(suite.T(), worktreePath, worktreeInfo.Path)
	
	// 7. Worktree 삭제
	suite.T().Log("   🗑️  Worktree 삭제")
	err = suite.gitManager.RemoveWorktree(suite.ctx, suite.testRepoDir, worktreePath)
	require.NoError(suite.T(), err)
	
	// Worktree 디렉토리가 삭제되었는지 확인
	assert.NoDirExists(suite.T(), worktreePath)
	
	// 테스트 worktree 목록에서 제거
	suite.testWorktrees = suite.testWorktrees[:0]

	suite.T().Log("🎉 Git worktree 생명주기 테스트 성공")
}

// TestConcurrentWorktreeOperations 동시 worktree 작업 테스트
func (suite *GitIntegrationTestSuite) TestConcurrentWorktreeOperations() {
	suite.T().Log("🔄 동시 worktree 작업 테스트 시작")

	worktreeCount := 3
	worktreePaths := make([]string, worktreeCount)
	branchNames := make([]string, worktreeCount)

	// 동시에 여러 worktree 생성
	suite.T().Logf("   📁 %d개 worktree 동시 생성", worktreeCount)
	for i := 0; i < worktreeCount; i++ {
		branchName := fmt.Sprintf("concurrent-branch-%d-%d", i, time.Now().Unix())
		worktreePath := filepath.Join(suite.testRepoDir, "worktrees", branchName)
		
		branchNames[i] = branchName
		worktreePaths[i] = worktreePath

		// 브랜치 생성
		err := suite.gitManager.CreateBranch(suite.ctx, suite.testRepoDir, branchName, "main")
		require.NoError(suite.T(), err)

		// Worktree 생성
		err = suite.gitManager.CreateWorktree(suite.ctx, suite.testRepoDir, worktreePath, branchName)
		require.NoError(suite.T(), err)
		suite.testWorktrees = append(suite.testWorktrees, worktreePath)
	}

	// 모든 worktree에서 동시에 파일 작업
	suite.T().Log("   📝 모든 worktree에서 동시 파일 작업")
	for i, worktreePath := range worktreePaths {
		go func(index int, path string) {
			testFilePath := filepath.Join(path, fmt.Sprintf("concurrent-test-%d.txt", index))
			testContent := fmt.Sprintf("Concurrent test content %d at %s", index, time.Now().Format(time.RFC3339))
			
			err := os.WriteFile(testFilePath, []byte(testContent), 0644)
			if err != nil {
				suite.T().Errorf("파일 작성 실패 (worktree %d): %v", index, err)
				return
			}

			// 파일 커밋
			err = suite.gitManager.AddFiles(suite.ctx, path, []string{fmt.Sprintf("concurrent-test-%d.txt", index)})
			if err != nil {
				suite.T().Errorf("파일 추가 실패 (worktree %d): %v", index, err)
				return
			}

			commitMsg := fmt.Sprintf("Add concurrent test file %d", index)
			err = suite.gitManager.Commit(suite.ctx, path, commitMsg)
			if err != nil {
				suite.T().Errorf("커밋 실패 (worktree %d): %v", index, err)
				return
			}
		}(i, worktreePath)
	}

	// 작업 완료 대기
	time.Sleep(5 * time.Second)

	// 모든 worktree 상태 확인
	suite.T().Log("   📊 모든 worktree 상태 확인")
	for i, worktreePath := range worktreePaths {
		status, err := suite.gitManager.GetWorktreeStatus(suite.ctx, worktreePath)
		require.NoError(suite.T(), err, "Worktree %d 상태 조회 실패", i)
		
		suite.T().Logf("   📋 Worktree %d 상태: 변경사항=%v, 브랜치=%s", 
			i, status.HasChanges, status.CurrentBranch)
	}

	suite.T().Log("🎉 동시 worktree 작업 테스트 성공")
}

// TestWorktreeBranchSwitching Worktree 브랜치 전환 테스트
func (suite *GitIntegrationTestSuite) TestWorktreeBranchSwitching() {
	suite.T().Log("🔄 Worktree 브랜치 전환 테스트 시작")

	// 테스트용 브랜치들 생성
	branch1 := fmt.Sprintf("switch-test-1-%d", time.Now().Unix())
	branch2 := fmt.Sprintf("switch-test-2-%d", time.Now().Unix())
	worktreePath := filepath.Join(suite.testRepoDir, "worktrees", "switch-test")

	suite.T().Logf("   🌿 테스트 브랜치 생성: %s, %s", branch1, branch2)
	err := suite.gitManager.CreateBranch(suite.ctx, suite.testRepoDir, branch1, "main")
	require.NoError(suite.T(), err)
	
	err = suite.gitManager.CreateBranch(suite.ctx, suite.testRepoDir, branch2, "main")
	require.NoError(suite.T(), err)

	// 첫 번째 브랜치로 worktree 생성
	suite.T().Logf("   📁 Worktree 생성 (브랜치: %s)", branch1)
	err = suite.gitManager.CreateWorktree(suite.ctx, suite.testRepoDir, worktreePath, branch1)
	require.NoError(suite.T(), err)
	suite.testWorktrees = append(suite.testWorktrees, worktreePath)

	// 첫 번째 브랜치에서 파일 생성
	suite.T().Log("   📝 첫 번째 브랜치에서 파일 생성")
	file1Path := filepath.Join(worktreePath, "branch1-file.txt")
	err = os.WriteFile(file1Path, []byte("Content from branch 1"), 0644)
	require.NoError(suite.T(), err)

	err = suite.gitManager.AddFiles(suite.ctx, worktreePath, []string{"branch1-file.txt"})
	require.NoError(suite.T(), err)
	
	err = suite.gitManager.Commit(suite.ctx, worktreePath, "Add file from branch 1")
	require.NoError(suite.T(), err)

	// 두 번째 브랜치로 전환
	suite.T().Logf("   🔄 브랜치 전환: %s -> %s", branch1, branch2)
	err = suite.gitManager.SwitchBranch(suite.ctx, worktreePath, branch2)
	require.NoError(suite.T(), err)

	// 브랜치 전환 확인
	info, err := suite.gitManager.GetWorktreeInfo(suite.ctx, worktreePath)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), branch2, info.Branch)

	// 첫 번째 브랜치의 파일이 없는지 확인
	assert.NoFileExists(suite.T(), file1Path, "브랜치 전환 후 이전 브랜치의 파일이 없어야 함")

	// 두 번째 브랜치에서 다른 파일 생성
	suite.T().Log("   📝 두 번째 브랜치에서 파일 생성")
	file2Path := filepath.Join(worktreePath, "branch2-file.txt")
	err = os.WriteFile(file2Path, []byte("Content from branch 2"), 0644)
	require.NoError(suite.T(), err)

	err = suite.gitManager.AddFiles(suite.ctx, worktreePath, []string{"branch2-file.txt"})
	require.NoError(suite.T(), err)
	
	err = suite.gitManager.Commit(suite.ctx, worktreePath, "Add file from branch 2")
	require.NoError(suite.T(), err)

	// 다시 첫 번째 브랜치로 전환
	suite.T().Logf("   🔄 브랜치 재전환: %s -> %s", branch2, branch1)
	err = suite.gitManager.SwitchBranch(suite.ctx, worktreePath, branch1)
	require.NoError(suite.T(), err)

	// 브랜치별 파일 존재 확인
	assert.FileExists(suite.T(), file1Path, "첫 번째 브랜치로 돌아온 후 해당 파일이 있어야 함")
	assert.NoFileExists(suite.T(), file2Path, "첫 번째 브랜치에서는 두 번째 브랜치의 파일이 없어야 함")

	suite.T().Log("🎉 Worktree 브랜치 전환 테스트 성공")
}

// TestWorktreeConflictResolution Worktree 충돌 해결 테스트
func (suite *GitIntegrationTestSuite) TestWorktreeConflictResolution() {
	suite.T().Log("🔄 Worktree 충돌 해결 테스트 시작")

	baseBranch := "main"
	conflictBranch := fmt.Sprintf("conflict-test-%d", time.Now().Unix())
	worktreePath := filepath.Join(suite.testRepoDir, "worktrees", "conflict-test")

	// 충돌 테스트용 브랜치 생성
	suite.T().Logf("   🌿 충돌 테스트 브랜치 생성: %s", conflictBranch)
	err := suite.gitManager.CreateBranch(suite.ctx, suite.testRepoDir, conflictBranch, baseBranch)
	require.NoError(suite.T(), err)

	// Worktree 생성
	suite.T().Log("   📁 Worktree 생성")
	err = suite.gitManager.CreateWorktree(suite.ctx, suite.testRepoDir, worktreePath, conflictBranch)
	require.NoError(suite.T(), err)
	suite.testWorktrees = append(suite.testWorktrees, worktreePath)

	// 메인 저장소에서 파일 수정 (충돌 상황 준비)
	suite.T().Log("   📝 메인 저장소에서 파일 수정")
	mainFilePath := filepath.Join(suite.testRepoDir, "README.md")
	err = os.WriteFile(mainFilePath, []byte("README content from main branch\nUpdated in main"), 0644)
	require.NoError(suite.T(), err)

	err = suite.gitManager.AddFiles(suite.ctx, suite.testRepoDir, []string{"README.md"})
	require.NoError(suite.T(), err)
	
	err = suite.gitManager.Commit(suite.ctx, suite.testRepoDir, "Update README in main")
	require.NoError(suite.T(), err)

	// Worktree에서 같은 파일 다르게 수정
	suite.T().Log("   📝 Worktree에서 같은 파일 다르게 수정")
	worktreeFilePath := filepath.Join(worktreePath, "README.md")
	err = os.WriteFile(worktreeFilePath, []byte("README content from conflict branch\nUpdated in feature branch"), 0644)
	require.NoError(suite.T(), err)

	err = suite.gitManager.AddFiles(suite.ctx, worktreePath, []string{"README.md"})
	require.NoError(suite.T(), err)
	
	err = suite.gitManager.Commit(suite.ctx, worktreePath, "Update README in feature branch")
	require.NoError(suite.T(), err)

	// 머지 시도 (충돌 발생 예상)
	suite.T().Log("   🔀 머지 시도 (충돌 예상)")
	err = suite.gitManager.MergeBranch(suite.ctx, worktreePath, baseBranch)
	
	if err != nil {
		suite.T().Logf("   ⚠️  예상된 머지 충돌 발생: %v", err)
		
		// 충돌 상태 확인
		status, err := suite.gitManager.GetWorktreeStatus(suite.ctx, worktreePath)
		require.NoError(suite.T(), err)
		
		suite.T().Logf("   📊 충돌 상태: 변경사항=%v, 충돌파일=%d개", 
			status.HasChanges, len(status.ConflictedFiles))
		
		if len(status.ConflictedFiles) > 0 {
			suite.T().Log("   🔧 충돌 해결 시뮬레이션")
			
			// 충돌 파일 수동 해결 (간단한 케이스)
			resolvedContent := "README content resolved\nMerged both changes"
			err = os.WriteFile(worktreeFilePath, []byte(resolvedContent), 0644)
			require.NoError(suite.T(), err)
			
			// 해결된 파일 스테이징
			err = suite.gitManager.AddFiles(suite.ctx, worktreePath, []string{"README.md"})
			require.NoError(suite.T(), err)
			
			// 머지 커밋 완료
			err = suite.gitManager.Commit(suite.ctx, worktreePath, "Resolve merge conflict")
			require.NoError(suite.T(), err)
			
			suite.T().Log("   ✅ 충돌 해결 완료")
		}
	} else {
		suite.T().Log("   ✅ 머지 성공 (충돌 없음)")
	}

	suite.T().Log("🎉 Worktree 충돌 해결 테스트 완료")
}

// TestWorktreePerformance Worktree 성능 테스트
func (suite *GitIntegrationTestSuite) TestWorktreePerformance() {
	suite.T().Log("🔄 Worktree 성능 테스트 시작")

	operationCount := 10
	times := make([]time.Duration, operationCount)

	suite.T().Logf("   ⏱️  %d개 worktree 생성 성능 측정", operationCount)
	
	for i := 0; i < operationCount; i++ {
		branchName := fmt.Sprintf("perf-test-%d", i)
		worktreePath := filepath.Join(suite.testRepoDir, "worktrees", branchName)
		
		startTime := time.Now()
		
		// 브랜치 생성
		err := suite.gitManager.CreateBranch(suite.ctx, suite.testRepoDir, branchName, "main")
		require.NoError(suite.T(), err)
		
		// Worktree 생성
		err = suite.gitManager.CreateWorktree(suite.ctx, suite.testRepoDir, worktreePath, branchName)
		require.NoError(suite.T(), err)
		suite.testWorktrees = append(suite.testWorktrees, worktreePath)
		
		times[i] = time.Since(startTime)
	}

	// 성능 통계 계산
	var totalTime time.Duration
	var minTime, maxTime time.Duration = times[0], times[0]
	
	for _, t := range times {
		totalTime += t
		if t < minTime {
			minTime = t
		}
		if t > maxTime {
			maxTime = t
		}
	}
	
	avgTime := totalTime / time.Duration(operationCount)
	
	suite.T().Logf("   📊 성능 결과:")
	suite.T().Logf("      평균: %v", avgTime)
	suite.T().Logf("      최소: %v", minTime)
	suite.T().Logf("      최대: %v", maxTime)
	suite.T().Logf("      총 시간: %v", totalTime)
	
	// 성능 검증 (평균 5초 이내)
	assert.Less(suite.T(), avgTime, 5*time.Second, "Worktree 생성 평균 시간이 5초를 초과하면 안됨")

	suite.T().Log("🎉 Worktree 성능 테스트 완료")
}

// 헬퍼 메서드들

// setupTestRepository 테스트용 Git 저장소 설정
func (suite *GitIntegrationTestSuite) setupTestRepository() {
	// Git 저장소 초기화
	err := suite.gitManager.InitRepository(suite.ctx, suite.testRepoDir)
	require.NoError(suite.T(), err)
	
	// 초기 파일 생성
	readmePath := filepath.Join(suite.testRepoDir, "README.md")
	readmeContent := `# Test Repository
This is a test repository for Git integration testing.

## Purpose
- Test Git worktree operations
- Test branch management
- Test concurrent operations
`
	err = os.WriteFile(readmePath, []byte(readmeContent), 0644)
	require.NoError(suite.T(), err)
	
	// Git 사용자 설정
	err = suite.gitManager.SetUserConfig(suite.ctx, suite.testRepoDir, "Test User", "test@example.com")
	require.NoError(suite.T(), err)
	
	// 초기 커밋
	err = suite.gitManager.AddFiles(suite.ctx, suite.testRepoDir, []string{"README.md"})
	require.NoError(suite.T(), err)
	
	err = suite.gitManager.Commit(suite.ctx, suite.testRepoDir, "Initial commit")
	require.NoError(suite.T(), err)
	
	suite.T().Log("   ✅ 테스트 Git 저장소 초기화 완료")
}

// cleanupTestWorktrees 테스트 worktree들 정리
func (suite *GitIntegrationTestSuite) cleanupTestWorktrees() {
	for _, worktreePath := range suite.testWorktrees {
		err := suite.gitManager.RemoveWorktree(suite.ctx, suite.testRepoDir, worktreePath)
		if err != nil {
			suite.T().Logf("Worktree 정리 실패: %s - %v", worktreePath, err)
		}
	}
	suite.testWorktrees = suite.testWorktrees[:0]
}

// addCleanupFunction 정리 함수 추가
func (suite *GitIntegrationTestSuite) addCleanupFunction(fn func()) {
	suite.cleanupFunctions = append(suite.cleanupFunctions, fn)
}

// TestGitIntegrationSuite Git 통합 테스트 실행
func TestGitIntegrationSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("통합 테스트는 short 모드에서 제외됩니다")
	}

	suite.Run(t, new(GitIntegrationTestSuite))
}