package helpers

import (
	"context"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/aicli/aicli-web/internal/claude"
	"github.com/aicli/aicli-web/internal/server"
)

// TestEnvironment는 통합 테스트를 위한 환경을 설정합니다
type TestEnvironment struct {
	TempDir      string
	MockClaude   *MockClaudeServer
	RealClaude   bool // 실제 Claude CLI 사용 여부
	TestData     *TestDataProvider
	APIServer    *httptest.Server
	Logger       *logrus.Logger
	mutex        sync.Mutex
	cleanup      []func()
}

// NewTestEnvironment는 새로운 테스트 환경을 생성합니다
func NewTestEnvironment(t *testing.T) *TestEnvironment {
	env := &TestEnvironment{
		TempDir:    t.TempDir(),
		RealClaude: os.Getenv("TEST_REAL_CLAUDE") == "true",
		TestData:   NewTestDataProvider(t),
		Logger:     createTestLogger(),
		cleanup:    make([]func(), 0),
	}
	
	// Mock Claude 서버 설정
	if !env.RealClaude {
		env.MockClaude = NewMockClaudeServer(t)
		env.AddCleanup(env.MockClaude.Stop)
	}
	
	// API 서버 시작
	apiServer := server.New()
	env.APIServer = httptest.NewServer(apiServer.Router())
	env.AddCleanup(env.APIServer.Close)
	
	// 정리 함수 등록
	t.Cleanup(env.Cleanup)
	
	return env
}

// AddCleanup은 정리 함수를 추가합니다
func (e *TestEnvironment) AddCleanup(cleanup func()) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.cleanup = append(e.cleanup, cleanup)
}

// Cleanup은 모든 정리 함수를 실행합니다
func (e *TestEnvironment) Cleanup() {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	
	for i := len(e.cleanup) - 1; i >= 0; i-- {
		e.cleanup[i]()
	}
}

// CreateWorkspace는 테스트용 워크스페이스를 생성합니다
func (e *TestEnvironment) CreateWorkspace(name string) string {
	workspaceDir := filepath.Join(e.TempDir, name)
	err := os.MkdirAll(workspaceDir, 0755)
	if err != nil {
		panic(fmt.Sprintf("워크스페이스 생성 실패: %v", err))
	}
	return workspaceDir
}

// GetTestLogger는 테스트용 로거를 반환합니다
func (e *TestEnvironment) GetTestLogger() *logrus.Logger {
	return e.Logger
}

// MockClaudeServer는 Claude CLI를 모킹하는 서버입니다
type MockClaudeServer struct {
	server      *httptest.Server
	responses   map[string][]byte
	interactions []MockInteraction
	mutex       sync.Mutex
	t          *testing.T
}

type MockInteraction struct {
	Timestamp time.Time
	Request   string
	Response  string
}

// NewMockClaudeServer는 새로운 모킹 서버를 생성합니다
func NewMockClaudeServer(t *testing.T) *MockClaudeServer {
	mock := &MockClaudeServer{
		responses:   make(map[string][]byte),
		interactions: make([]MockInteraction, 0),
		t:          t,
	}
	
	// HTTP 서버 시작은 실제 Claude CLI와 상호작용하지 않으므로
	// 프로세스 시뮬레이션을 위한 스크립트 생성 로직으로 대체
	
	return mock
}

// SetResponse는 특정 패턴에 대한 응답을 설정합니다
func (m *MockClaudeServer) SetResponse(pattern string, response []byte) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.responses[pattern] = response
}

// SimulateClaudeProcess는 Claude CLI 프로세스를 시뮬레이션하는 스크립트를 생성합니다
func (m *MockClaudeServer) SimulateClaudeProcess() string {
	script := `#!/bin/bash
echo "Claude CLI Simulation Started"
echo '{"event": "session_started", "timestamp": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'"}'

# 사용자 입력 대기 시뮬레이션
while read -r input; do
	if [[ "$input" == "exit" ]]; then
		break
	fi
	
	echo '{"type": "text", "content": "Processing: '$input'"}'
	sleep 0.5
	echo '{"type": "text", "content": "Response to: '$input'"}'
done

echo '{"event": "session_completed", "timestamp": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'"}'
`
	
	tmpfile, err := os.CreateTemp("", "claude_mock_*.sh")
	require.NoError(m.t, err)
	
	_, err = tmpfile.WriteString(script)
	require.NoError(m.t, err)
	
	err = tmpfile.Close()
	require.NoError(m.t, err)
	
	// 실행 권한 부여
	err = os.Chmod(tmpfile.Name(), 0755)
	require.NoError(m.t, err)
	
	return tmpfile.Name()
}

// Stop은 모킹 서버를 중지합니다
func (m *MockClaudeServer) Stop() {
	// 실제 HTTP 서버가 없으므로 정리 작업만 수행
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.interactions = nil
	m.responses = nil
}

// GetInteractions는 모킹된 상호작용 기록을 반환합니다
func (m *MockClaudeServer) GetInteractions() []MockInteraction {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return append([]MockInteraction(nil), m.interactions...)
}

// TestDataProvider는 테스트 데이터를 제공합니다
type TestDataProvider struct {
	t       *testing.T
	dataDir string
}

// NewTestDataProvider는 새로운 테스트 데이터 프로바이더를 생성합니다
func NewTestDataProvider(t *testing.T) *TestDataProvider {
	return &TestDataProvider{
		t:       t,
		dataDir: filepath.Join("testdata"),
	}
}

// LoadStreamData는 스트림 테스트 데이터를 로드합니다
func (p *TestDataProvider) LoadStreamData(filename string) []byte {
	// 테스트 데이터 생성 (실제 파일이 없는 경우)
	if filename == "complex_response.jsonl" {
		return p.generateComplexStreamData()
	}
	
	// 실제 파일에서 로드하는 로직 (옵션)
	filePath := filepath.Join(p.dataDir, filename)
	data, err := os.ReadFile(filePath)
	if err != nil {
		// 파일이 없으면 기본 데이터 생성
		p.t.Logf("테스트 데이터 파일을 찾을 수 없음: %s, 기본 데이터 사용", filePath)
		return p.generateDefaultStreamData()
	}
	
	return data
}

// generateComplexStreamData는 복잡한 스트림 데이터를 생성합니다
func (p *TestDataProvider) generateComplexStreamData() []byte {
	data := `{"type":"text","content":"안녕하세요! 도움이 필요한 작업이 있나요?"}
{"type":"tool_use","tool_name":"Write","input":{"file_path":"/tmp/test.go","content":"package main\n\nfunc main() {\n\tprintln(\"Hello World\")\n}"}}
{"type":"text","content":"Go 파일을 생성했습니다."}
{"type":"tool_use","tool_name":"Bash","input":{"command":"go run /tmp/test.go"}}
{"type":"text","content":"프로그램 실행 결과: Hello World"}
{"type":"completion","final":true}
`
	return []byte(data)
}

// generateDefaultStreamData는 기본 스트림 데이터를 생성합니다
func (p *TestDataProvider) generateDefaultStreamData() []byte {
	data := `{"type":"text","content":"기본 응답 메시지"}
{"type":"completion","final":true}
`
	return []byte(data)
}

// GenerateLargeStreamData는 대용량 스트림 데이터를 생성합니다
func (p *TestDataProvider) GenerateLargeStreamData(sizeBytes int) []byte {
	message := `{"type":"text","content":"테스트 메시지입니다."}`
	messageLen := len(message) + 1 // +1 for newline
	numMessages := sizeBytes / messageLen
	
	result := make([]byte, 0, sizeBytes)
	for i := 0; i < numMessages; i++ {
		result = append(result, []byte(fmt.Sprintf(`{"type":"text","content":"메시지 %d"}`, i))...)
		result = append(result, '\n')
	}
	
	return result
}

// createTestLogger는 테스트용 로거를 생성합니다
func createTestLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
		DisableColors: true, // CI 환경에서 색상 비활성화
	})
	
	// 테스트 시에는 출력을 줄입니다
	if os.Getenv("VERBOSE_TESTS") != "true" {
		logger.SetLevel(logrus.WarnLevel)
	}
	
	return logger
}

// WaitForCondition은 조건이 충족될 때까지 대기합니다
func WaitForCondition(ctx context.Context, condition func() bool, checkInterval time.Duration) error {
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if condition() {
				return nil
			}
		}
	}
}

// ProcessTestHelper는 프로세스 테스트를 위한 헬퍼 함수들을 제공합니다
type ProcessTestHelper struct {
	env *TestEnvironment
}

// NewProcessTestHelper는 새로운 프로세스 테스트 헬퍼를 생성합니다
func NewProcessTestHelper(env *TestEnvironment) *ProcessTestHelper {
	return &ProcessTestHelper{env: env}
}

// WaitForProcessState는 프로세스 상태가 변경될 때까지 대기합니다
func (h *ProcessTestHelper) WaitForProcessState(ctx context.Context, pm claude.ProcessManager, expectedState claude.ProcessState) error {
	return WaitForCondition(ctx, func() bool {
		return pm.GetStatus() == expectedState
	}, 100*time.Millisecond)
}

// CreateTestProcessConfig는 테스트용 프로세스 설정을 생성합니다
func (h *ProcessTestHelper) CreateTestProcessConfig() *claude.ProcessConfig {
	if h.env.RealClaude {
		// 실제 Claude CLI 사용
		return &claude.ProcessConfig{
			Command: "claude", // 실제 Claude CLI 바이너리
			Args:    []string{"--interactive"},
		}
	}
	
	// Mock Claude 사용
	scriptPath := h.env.MockClaude.SimulateClaudeProcess()
	h.env.AddCleanup(func() { os.Remove(scriptPath) })
	
	return &claude.ProcessConfig{
		Command: "bash",
		Args:    []string{scriptPath},
	}
}