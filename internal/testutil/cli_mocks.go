package testutil

import (
	"context"
	"io"
	"os/exec"
	"sync"
	"time"

	"github.com/stretchr/testify/mock"
)

// MockClaudeWrapper Claude CLI 래퍼의 모의 객체
type MockClaudeWrapper struct {
	mock.Mock
	mutex sync.RWMutex
}

// MockClaudeResponse Claude CLI 응답 구조체
type MockClaudeResponse struct {
	Content   string            `json:"content"`
	Metadata  map[string]string `json:"metadata"`
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
}

// Start Claude CLI 프로세스 시작
func (m *MockClaudeWrapper) Start(ctx context.Context, workspaceDir string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	args := m.Called(ctx, workspaceDir)
	return args.Error(0)
}

// Stop Claude CLI 프로세스 중지
func (m *MockClaudeWrapper) Stop(ctx context.Context) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	args := m.Called(ctx)
	return args.Error(0)
}

// Execute Claude CLI 명령 실행
func (m *MockClaudeWrapper) Execute(ctx context.Context, command string) (*MockClaudeResponse, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	args := m.Called(ctx, command)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*MockClaudeResponse), args.Error(1)
}

// IsRunning 프로세스 실행 상태 확인
func (m *MockClaudeWrapper) IsRunning() bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	args := m.Called()
	return args.Bool(0)
}

// GetStatus 프로세스 상태 반환
func (m *MockClaudeWrapper) GetStatus() string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	args := m.Called()
	return args.String(0)
}

// MockProcessManager 프로세스 관리자의 모의 객체
type MockProcessManager struct {
	mock.Mock
	mutex sync.RWMutex
}

// ProcessConfig 프로세스 설정
type ProcessConfig struct {
	Command     string
	Args        []string
	WorkingDir  string
	Environment map[string]string
	Timeout     time.Duration
}

// ProcessStatus 프로세스 상태
type ProcessStatus int

const (
	StatusStopped ProcessStatus = iota
	StatusStarting
	StatusRunning
	StatusStopping
	StatusError
	StatusUnknown
)

// Start 프로세스 시작
func (m *MockProcessManager) Start(ctx context.Context, config *ProcessConfig) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	args := m.Called(ctx, config)
	return args.Error(0)
}

// Stop 프로세스 중지
func (m *MockProcessManager) Stop(timeout time.Duration) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	args := m.Called(timeout)
	return args.Error(0)
}

// Kill 프로세스 강제 종료
func (m *MockProcessManager) Kill() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	args := m.Called()
	return args.Error(0)
}

// IsRunning 프로세스 실행 여부 확인
func (m *MockProcessManager) IsRunning() bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	args := m.Called()
	return args.Bool(0)
}

// GetStatus 프로세스 상태 반환
func (m *MockProcessManager) GetStatus() ProcessStatus {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	args := m.Called()
	return args.Get(0).(ProcessStatus)
}

// GetPID 프로세스 ID 반환
func (m *MockProcessManager) GetPID() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	args := m.Called()
	return args.Int(0)
}

// Wait 프로세스 종료 대기
func (m *MockProcessManager) Wait() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	args := m.Called()
	return args.Error(0)
}

// HealthCheck 프로세스 상태 확인
func (m *MockProcessManager) HealthCheck() error {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	args := m.Called()
	return args.Error(0)
}

// MockFileSystem 파일 시스템의 향상된 모의 객체
type MockFileSystem struct {
	mock.Mock
	files map[string][]byte
	dirs  map[string]bool
	mutex sync.RWMutex
}

// NewMockFileSystem 새로운 모의 파일 시스템 생성
func NewMockFileSystem() *MockFileSystem {
	return &MockFileSystem{
		files: make(map[string][]byte),
		dirs:  make(map[string]bool),
	}
}

// ReadFile 파일 읽기
func (mfs *MockFileSystem) ReadFile(filename string) ([]byte, error) {
	mfs.mutex.RLock()
	defer mfs.mutex.RUnlock()
	
	// Mock 호출 기록
	args := mfs.Called(filename)
	
	// 실제 파일 데이터 반환 (있는 경우)
	if data, exists := mfs.files[filename]; exists {
		return data, nil
	}
	
	// Mock에서 설정된 반환값 사용
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

// WriteFile 파일 쓰기
func (mfs *MockFileSystem) WriteFile(filename string, data []byte, perm int) error {
	mfs.mutex.Lock()
	defer mfs.mutex.Unlock()
	
	args := mfs.Called(filename, data, perm)
	
	// 실제 파일 데이터 저장
	mfs.files[filename] = make([]byte, len(data))
	copy(mfs.files[filename], data)
	
	return args.Error(0)
}

// Exists 파일/디렉토리 존재 여부 확인
func (mfs *MockFileSystem) Exists(path string) bool {
	mfs.mutex.RLock()
	defer mfs.mutex.RUnlock()
	
	args := mfs.Called(path)
	
	// 실제 데이터 확인
	if _, exists := mfs.files[path]; exists {
		return true
	}
	if _, exists := mfs.dirs[path]; exists {
		return true
	}
	
	return args.Bool(0)
}

// MkdirAll 디렉토리 생성
func (mfs *MockFileSystem) MkdirAll(path string, perm int) error {
	mfs.mutex.Lock()
	defer mfs.mutex.Unlock()
	
	args := mfs.Called(path, perm)
	
	// 실제 디렉토리 생성 시뮬레이션
	mfs.dirs[path] = true
	
	return args.Error(0)
}

// Remove 파일/디렉토리 삭제
func (mfs *MockFileSystem) Remove(path string) error {
	mfs.mutex.Lock()
	defer mfs.mutex.Unlock()
	
	args := mfs.Called(path)
	
	// 실제 데이터 삭제
	delete(mfs.files, path)
	delete(mfs.dirs, path)
	
	return args.Error(0)
}

// AddFile 테스트용 파일 추가
func (mfs *MockFileSystem) AddFile(filename string, content []byte) {
	mfs.mutex.Lock()
	defer mfs.mutex.Unlock()
	
	mfs.files[filename] = make([]byte, len(content))
	copy(mfs.files[filename], content)
}

// AddDir 테스트용 디렉토리 추가
func (mfs *MockFileSystem) AddDir(dirname string) {
	mfs.mutex.Lock()
	defer mfs.mutex.Unlock()
	
	mfs.dirs[dirname] = true
}

// MockCommand 명령어 실행의 모의 객체
type MockCommand struct {
	mock.Mock
	processes map[string]*MockProcess
	mutex     sync.RWMutex
}

// MockProcess 프로세스의 모의 객체
type MockProcess struct {
	cmd      *exec.Cmd
	stdout   io.ReadCloser
	stderr   io.ReadCloser
	stdin    io.WriteCloser
	exitCode int
	running  bool
}

// NewMockCommand 새로운 모의 명령어 객체 생성
func NewMockCommand() *MockCommand {
	return &MockCommand{
		processes: make(map[string]*MockProcess),
	}
}

// Start 명령어 시작
func (mc *MockCommand) Start(name string, args ...string) (*MockProcess, error) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	
	mockArgs := mc.Called(name, args)
	
	if mockArgs.Get(0) == nil {
		return nil, mockArgs.Error(1)
	}
	
	process := mockArgs.Get(0).(*MockProcess)
	process.running = true
	mc.processes[name] = process
	
	return process, mockArgs.Error(1)
}

// Stop 명령어 중지
func (mc *MockCommand) Stop(name string) error {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	
	args := mc.Called(name)
	
	if process, exists := mc.processes[name]; exists {
		process.running = false
	}
	
	return args.Error(0)
}

// IsRunning 명령어 실행 상태 확인
func (mc *MockCommand) IsRunning(name string) bool {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()
	
	args := mc.Called(name)
	
	if process, exists := mc.processes[name]; exists {
		return process.running
	}
	
	return args.Bool(0)
}

// Output 명령어 출력 반환
func (mp *MockProcess) Output() ([]byte, error) {
	if mp.stdout != nil {
		return io.ReadAll(mp.stdout)
	}
	return []byte{}, nil
}

// Wait 프로세스 종료 대기
func (mp *MockProcess) Wait() error {
	mp.running = false
	return nil
}

// ExitCode 종료 코드 반환
func (mp *MockProcess) ExitCode() int {
	return mp.exitCode
}

// MockWorkspace 워크스페이스의 모의 객체
type MockWorkspace struct {
	mock.Mock
	workspaces map[string]*WorkspaceInfo
	mutex      sync.RWMutex
}

// WorkspaceInfo 워크스페이스 정보
type WorkspaceInfo struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Path        string            `json:"path"`
	Status      string            `json:"status"`
	CreatedAt   time.Time         `json:"created_at"`
	LastAccess  time.Time         `json:"last_access"`
	Metadata    map[string]string `json:"metadata"`
}

// NewMockWorkspace 새로운 모의 워크스페이스 생성
func NewMockWorkspace() *MockWorkspace {
	return &MockWorkspace{
		workspaces: make(map[string]*WorkspaceInfo),
	}
}

// Create 워크스페이스 생성
func (mw *MockWorkspace) Create(ctx context.Context, name, path string) (*WorkspaceInfo, error) {
	mw.mutex.Lock()
	defer mw.mutex.Unlock()
	
	args := mw.Called(ctx, name, path)
	
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	
	workspace := args.Get(0).(*WorkspaceInfo)
	mw.workspaces[workspace.ID] = workspace
	
	return workspace, args.Error(1)
}

// Delete 워크스페이스 삭제
func (mw *MockWorkspace) Delete(ctx context.Context, id string) error {
	mw.mutex.Lock()
	defer mw.mutex.Unlock()
	
	args := mw.Called(ctx, id)
	
	delete(mw.workspaces, id)
	
	return args.Error(0)
}

// List 워크스페이스 목록 조회
func (mw *MockWorkspace) List(ctx context.Context) ([]*WorkspaceInfo, error) {
	mw.mutex.RLock()
	defer mw.mutex.RUnlock()
	
	args := mw.Called(ctx)
	
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	
	return args.Get(0).([]*WorkspaceInfo), args.Error(1)
}

// Get 워크스페이스 정보 조회
func (mw *MockWorkspace) Get(ctx context.Context, id string) (*WorkspaceInfo, error) {
	mw.mutex.RLock()
	defer mw.mutex.RUnlock()
	
	args := mw.Called(ctx, id)
	
	if workspace, exists := mw.workspaces[id]; exists {
		return workspace, nil
	}
	
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	
	return args.Get(0).(*WorkspaceInfo), args.Error(1)
}

// AddWorkspace 테스트용 워크스페이스 추가
func (mw *MockWorkspace) AddWorkspace(workspace *WorkspaceInfo) {
	mw.mutex.Lock()
	defer mw.mutex.Unlock()
	
	mw.workspaces[workspace.ID] = workspace
}