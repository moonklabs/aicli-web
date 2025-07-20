---
task_id: T07_S01
sprint_sequence_id: S01_M02
status: open
complexity: Medium
last_updated: 2025-07-21T06:21:00Z
github_issue: # Optional: GitHub issue number
---

# Task: CLI 테스트 프레임워크 구축

## Description
AICode Manager CLI의 포괄적인 테스트 프레임워크를 구축합니다. 단위 테스트, 통합 테스트, 명령어 실행 테스트를 통해 CLI 동작의 정확성과 안정성을 보장합니다.

## Goal / Objectives
- CLI 명령어 실행 테스트 자동화
- 모의 객체 및 테스트 유틸리티 구현
- 통합 테스트 시나리오 작성
- 테스트 커버리지 향상 (목표: 80% 이상)

## Acceptance Criteria
- [ ] CLI 명령어 실행 테스트 프레임워크 구현
- [ ] 모의 Claude CLI 및 파일 시스템 구현
- [ ] 단위 테스트 및 통합 테스트 작성
- [ ] 테스트 유틸리티 및 헬퍼 함수 구현
- [ ] CI/CD 파이프라인과 통합된 테스트 실행

## Subtasks
- [ ] CLI 테스트 프레임워크 설계
- [ ] 모의 객체 (Mock) 구현
- [ ] 테스트 유틸리티 함수 작성
- [ ] 단위 테스트 케이스 구현
- [ ] 통합 테스트 시나리오 작성
- [ ] 테스트 커버리지 측정 및 개선

## 기술 가이드

### 주요 인터페이스 및 통합 지점
- **기존 패키지**: `internal/testutil/` 패키지 확장
- **테스트 라이브러리**: 
  - `github.com/stretchr/testify` (assertion, mock)
  - `github.com/spf13/cobra/cobra/cmd` (CLI 테스트)
- **CI 통합**: GitHub Actions 워크플로우

### CLI 테스트 프레임워크 설계
```go
type CLITestRunner struct {
    cmd        *cobra.Command
    stdin      io.Reader
    stdout     *bytes.Buffer
    stderr     *bytes.Buffer
    env        map[string]string
    workingDir string
}

func NewCLITestRunner() *CLITestRunner {
    return &CLITestRunner{
        stdout: &bytes.Buffer{},
        stderr: &bytes.Buffer{},
        env:    make(map[string]string),
    }
}

func (r *CLITestRunner) RunCommand(args ...string) error {
    r.cmd.SetArgs(args)
    r.cmd.SetOut(r.stdout)
    r.cmd.SetErr(r.stderr)
    r.cmd.SetIn(r.stdin)
    
    return r.cmd.Execute()
}

func (r *CLITestRunner) GetOutput() string {
    return r.stdout.String()
}

func (r *CLITestRunner) GetError() string {
    return r.stderr.String()
}
```

### 모의 객체 구현
```go
type MockClaudeWrapper struct {
    mock.Mock
}

func (m *MockClaudeWrapper) Start(ctx context.Context, workspaceDir string) error {
    args := m.Called(ctx, workspaceDir)
    return args.Error(0)
}

func (m *MockClaudeWrapper) Execute(ctx context.Context, command string) (*Response, error) {
    args := m.Called(ctx, command)
    return args.Get(0).(*Response), args.Error(1)
}

// 파일 시스템 모의 객체
type MockFileSystem struct {
    files map[string][]byte
    dirs  map[string]bool
}

func (mfs *MockFileSystem) ReadFile(filename string) ([]byte, error) {
    if data, exists := mfs.files[filename]; exists {
        return data, nil
    }
    return nil, os.ErrNotExist
}
```

### 구현 노트

#### 단계별 구현 접근법
1. **테스트 인프라 구축**
   - CLI 테스트 러너 구현
   - 모의 객체 인터페이스 정의
   - 테스트 유틸리티 함수 작성

2. **단위 테스트 작성**
   ```go
   func TestConfigGet(t *testing.T) {
       runner := NewCLITestRunner()
       runner.SetEnv("AICLI_CLAUDE_API_KEY", "test-key")
       
       err := runner.RunCommand("config", "get", "claude.api_key")
       
       assert.NoError(t, err)
       assert.Contains(t, runner.GetOutput(), "test-key")
   }
   ```

3. **통합 테스트 시나리오**
   - 전체 워크플로우 테스트
   - 실제 파일 시스템 상호작용
   - 다중 명령어 시퀀스 테스트

4. **테스트 데이터 관리**
   - 테스트 픽스처 생성
   - 임시 디렉토리 관리
   - 테스트 간 격리 보장

### 테스트 시나리오 분류

#### 1. 명령어 실행 테스트
- 기본 명령어 실행 검증
- 플래그 및 인자 파싱 테스트
- 에러 상황 처리 검증

#### 2. 설정 관리 테스트
- 설정 파일 읽기/쓰기 테스트
- 환경 변수 오버라이드 검증
- 기본값 처리 테스트

#### 3. 출력 포맷팅 테스트
- 다양한 출력 형식 검증
- 색상 지원 테스트
- 터미널 크기 대응 테스트

#### 4. 에러 처리 테스트
- 다양한 에러 시나리오 시뮬레이션
- 에러 메시지 형식 검증
- 종료 코드 확인

### 테스트 유틸리티 함수
```go
// 임시 워크스페이스 생성
func CreateTempWorkspace(t *testing.T) string {
    dir, err := ioutil.TempDir("", "aicli-test-*")
    require.NoError(t, err)
    
    t.Cleanup(func() {
        os.RemoveAll(dir)
    })
    
    return dir
}

// 설정 파일 생성
func CreateTestConfig(t *testing.T, config map[string]interface{}) string {
    configFile := filepath.Join(CreateTempWorkspace(t), "config.yaml")
    
    data, err := yaml.Marshal(config)
    require.NoError(t, err)
    
    err = ioutil.WriteFile(configFile, data, 0644)
    require.NoError(t, err)
    
    return configFile
}

// CLI 출력 검증
func AssertOutputContains(t *testing.T, runner *CLITestRunner, expected string) {
    output := runner.GetOutput()
    assert.Contains(t, output, expected, "Expected output to contain: %s", expected)
}
```

### 기존 테스트 패턴 기반 테스트 접근법
- 기존 `internal/testutil/` 패키지 활용
- Table-driven 테스트 패턴 적용
- 서브테스트를 통한 테스트 조직화
- 병렬 테스트 실행 최적화

### 성능 테스트
```go
func BenchmarkCLIStartup(b *testing.B) {
    for i := 0; i < b.N; i++ {
        runner := NewCLITestRunner()
        err := runner.RunCommand("version")
        require.NoError(b, err)
    }
}
```

### CI/CD 통합
- 테스트 커버리지 리포트 생성
- 실패한 테스트 상세 로그
- 성능 회귀 감지
- 플랫폼별 테스트 실행

### 테스트 실행 예시
```bash
# 모든 테스트 실행
go test ./...

# 커버리지 포함 실행
go test -coverprofile=coverage.out ./...

# 특정 패키지 테스트
go test ./internal/cli/...

# 통합 테스트만 실행
go test -tags=integration ./...
```

## Output Log
*(This section is populated as work progresses on the task)*