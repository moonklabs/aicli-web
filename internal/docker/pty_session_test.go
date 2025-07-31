package docker

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockPTYClient PTY 테스트를 위한 Mock 클라이언트
type MockPTYClient struct {
	mock.Mock
}

func (m *MockPTYClient) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockPTYClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockPTYClient) GetConfig() *Config {
	args := m.Called()
	return args.Get(0).(*Config)
}

func (m *MockPTYClient) GetNetworkID() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockPTYClient) ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error) {
	args := m.Called(ctx, containerID)
	return args.Get(0).(types.ContainerJSON), args.Error(1)
}

func (m *MockPTYClient) ContainerKill(ctx context.Context, containerID string, signal string) error {
	args := m.Called(ctx, containerID, signal)
	return args.Error(0)
}

func (m *MockPTYClient) ContainerRemove(ctx context.Context, containerID string, options types.ContainerRemoveOptions) error {
	args := m.Called(ctx, containerID, options)
	return args.Error(0)
}

func (m *MockPTYClient) ContainerExecCreate(ctx context.Context, containerID string, config ExecConfig) (types.IDResponse, error) {
	args := m.Called(ctx, containerID, config)
	return args.Get(0).(types.IDResponse), args.Error(1)
}

func (m *MockPTYClient) ContainerExecStart(ctx context.Context, execID string, config ExecStartConfig) (types.HijackedResponse, error) {
	args := m.Called(ctx, execID, config)
	return args.Get(0).(types.HijackedResponse), args.Error(1)
}

func (m *MockPTYClient) ContainerExecInspect(ctx context.Context, execID string) (types.ContainerExecInspect, error) {
	args := m.Called(ctx, execID)
	return args.Get(0).(types.ContainerExecInspect), args.Error(1)
}

// MockHijackedResponse Hijacked 연결을 위한 Mock
type MockHijackedResponse struct {
	Conn   io.ReadWriteCloser
	Reader *bufio.Reader
}

// MockReadWriteCloser 테스트용 ReadWriteCloser
type MockReadWriteCloser struct {
	readData  []byte
	writeData []byte
	closed    bool
}

func NewMockReadWriteCloser(readData string) *MockReadWriteCloser {
	return &MockReadWriteCloser{
		readData: []byte(readData),
	}
}

func (m *MockReadWriteCloser) Read(p []byte) (n int, err error) {
	if m.closed {
		return 0, io.EOF
	}
	if len(m.readData) == 0 {
		return 0, io.EOF
	}
	n = copy(p, m.readData)
	m.readData = m.readData[n:]
	return n, nil
}

func (m *MockReadWriteCloser) Write(p []byte) (n int, err error) {
	if m.closed {
		return 0, errors.New("connection closed")
	}
	m.writeData = append(m.writeData, p...)
	return len(p), nil
}

func (m *MockReadWriteCloser) Close() error {
	m.closed = true
	return nil
}

func (m *MockReadWriteCloser) GetWrittenData() string {
	return string(m.writeData)
}

// TestNewPTYSession PTY 세션 생성 테스트
func TestNewPTYSession(t *testing.T) {
	mockClient := &MockPTYClient{}
	containerID := "test-container-123"

	session := NewPTYSession(containerID, mockClient)

	assert.NotNil(t, session)
	assert.NotEmpty(t, session.ID())
	assert.Equal(t, containerID, session.ContainerID())
	assert.False(t, session.IsAlive())
	assert.WithinDuration(t, time.Now(), session.GetCreatedAt(), time.Second)
}

// TestPTYSessionStart PTY 세션 시작 테스트
func TestPTYSessionStart(t *testing.T) {
	mockClient := &MockPTYClient{}
	containerID := "test-container-123"
	execID := "exec-id-456"

	// Mock 설정
	mockConn := NewMockReadWriteCloser("test output")
	hijackedResp := types.HijackedResponse{
		Conn:   mockConn,
		Reader: bufio.NewReader(strings.NewReader("test output")),
	}

	mockClient.On("ContainerExecCreate", mock.Anything, containerID, mock.AnythingOfType("ExecConfig")).
		Return(types.IDResponse{ID: execID}, nil)
	mockClient.On("ContainerExecStart", mock.Anything, execID, mock.AnythingOfType("ExecStartConfig")).
		Return(hijackedResp, nil)

	session := NewPTYSession(containerID, mockClient)
	ctx := context.Background()

	err := session.Start(ctx)

	assert.NoError(t, err)
	assert.True(t, session.IsAlive())
	assert.WithinDuration(t, time.Now(), session.GetLastActivity(), time.Second)

	// 정리
	session.Stop()
	mockClient.AssertExpectations(t)
}

// TestPTYSessionWriteRead PTY 세션 읽기/쓰기 테스트
func TestPTYSessionWriteRead(t *testing.T) {
	mockClient := &MockPTYClient{}
	containerID := "test-container-123"
	execID := "exec-id-456"

	// Mock 설정
	mockConn := NewMockReadWriteCloser("hello world")
	hijackedResp := types.HijackedResponse{
		Conn:   mockConn,
		Reader: bufio.NewReader(strings.NewReader("hello world")),
	}

	mockClient.On("ContainerExecCreate", mock.Anything, containerID, mock.AnythingOfType("ExecConfig")).
		Return(types.IDResponse{ID: execID}, nil)
	mockClient.On("ContainerExecStart", mock.Anything, execID, mock.AnythingOfType("ExecStartConfig")).
		Return(hijackedResp, nil)

	session := NewPTYSession(containerID, mockClient)
	ctx := context.Background()

	// 세션 시작
	err := session.Start(ctx)
	assert.NoError(t, err)

	// 데이터 쓰기
	testData := []byte("echo test\n")
	n, err := session.Write(testData)
	assert.NoError(t, err)
	assert.Equal(t, len(testData), n)

	// 데이터 읽기
	readBuffer := make([]byte, 1024)
	n, err = session.Read(readBuffer)
	assert.NoError(t, err)
	assert.Greater(t, n, 0)

	// 정리
	session.Stop()
	mockClient.AssertExpectations(t)
}

// TestPTYSessionManagerCreate PTY 세션 관리자 생성 테스트
func TestPTYSessionManagerCreate(t *testing.T) {
	mockClient := &MockPTYClient{}
	maxSessions := 10

	manager := NewPTYSessionManager(mockClient, maxSessions)

	assert.NotNil(t, manager)
	assert.Equal(t, 0, manager.GetSessionCount())

	// 정리
	manager.Shutdown()
}

// TestPTYSessionManagerCreateSession 세션 생성 테스트
func TestPTYSessionManagerCreateSession(t *testing.T) {
	mockClient := &MockPTYClient{}
	containerID := "test-container-123"
	execID := "exec-id-456"
	maxSessions := 10

	// Mock 설정
	mockConn := NewMockReadWriteCloser("test output")
	hijackedResp := types.HijackedResponse{
		Conn:   mockConn,
		Reader: bufio.NewReader(strings.NewReader("test output")),
	}

	mockClient.On("ContainerInspect", mock.Anything, containerID).
		Return(types.ContainerJSON{}, nil)
	mockClient.On("ContainerExecCreate", mock.Anything, containerID, mock.AnythingOfType("ExecConfig")).
		Return(types.IDResponse{ID: execID}, nil)
	mockClient.On("ContainerExecStart", mock.Anything, execID, mock.AnythingOfType("ExecStartConfig")).
		Return(hijackedResp, nil)

	manager := NewPTYSessionManager(mockClient, maxSessions)
	ctx := context.Background()

	// 세션 생성
	session, err := manager.CreateSession(ctx, containerID)

	assert.NoError(t, err)
	assert.NotNil(t, session)
	assert.Equal(t, containerID, session.ContainerID())
	assert.True(t, session.IsAlive())
	assert.Equal(t, 1, manager.GetSessionCount())

	// 정리
	manager.Shutdown()
	mockClient.AssertExpectations(t)
}

// TestPTYSessionManagerMaxSessions 최대 세션 수 제한 테스트
func TestPTYSessionManagerMaxSessions(t *testing.T) {
	mockClient := &MockPTYClient{}
	maxSessions := 2
	containerID := "test-container-123"

	manager := NewPTYSessionManager(mockClient, maxSessions)
	ctx := context.Background()

	// Mock 설정 (최대 세션 수만큼)
	for i := 0; i < maxSessions; i++ {
		execID := fmt.Sprintf("exec-id-%d", i)
		mockConn := NewMockReadWriteCloser("test output")
		hijackedResp := types.HijackedResponse{
			Conn:   mockConn,
			Reader: bufio.NewReader(strings.NewReader("test output")),
		}

		mockClient.On("ContainerInspect", mock.Anything, containerID).
			Return(types.ContainerJSON{}, nil).Once()
		mockClient.On("ContainerExecCreate", mock.Anything, containerID, mock.AnythingOfType("ExecConfig")).
			Return(types.IDResponse{ID: execID}, nil).Once()
		mockClient.On("ContainerExecStart", mock.Anything, execID, mock.AnythingOfType("ExecStartConfig")).
			Return(hijackedResp, nil).Once()
	}

	// 최대 세션 수만큼 생성
	var sessions []PTYSession
	for i := 0; i < maxSessions; i++ {
		session, err := manager.CreateSession(ctx, containerID)
		assert.NoError(t, err)
		sessions = append(sessions, session)
	}

	assert.Equal(t, maxSessions, manager.GetSessionCount())

	// 최대 세션 수 초과 시도
	_, err := manager.CreateSession(ctx, containerID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "maximum number of sessions")

	// 정리
	manager.Shutdown()
	mockClient.AssertExpectations(t)
}

// TestPTYSessionManagerGetSession 세션 조회 테스트
func TestPTYSessionManagerGetSession(t *testing.T) {
	mockClient := &MockPTYClient{}
	containerID := "test-container-123"
	execID := "exec-id-456"
	maxSessions := 10

	// Mock 설정
	mockConn := NewMockReadWriteCloser("test output")
	hijackedResp := types.HijackedResponse{
		Conn:   mockConn,
		Reader: bufio.NewReader(strings.NewReader("test output")),
	}

	mockClient.On("ContainerInspect", mock.Anything, containerID).
		Return(types.ContainerJSON{}, nil)
	mockClient.On("ContainerExecCreate", mock.Anything, containerID, mock.AnythingOfType("ExecConfig")).
		Return(types.IDResponse{ID: execID}, nil)
	mockClient.On("ContainerExecStart", mock.Anything, execID, mock.AnythingOfType("ExecStartConfig")).
		Return(hijackedResp, nil)

	manager := NewPTYSessionManager(mockClient, maxSessions)
	ctx := context.Background()

	// 세션 생성
	originalSession, err := manager.CreateSession(ctx, containerID)
	assert.NoError(t, err)

	// 세션 조회
	retrievedSession, err := manager.GetSession(originalSession.ID())
	assert.NoError(t, err)
	assert.Equal(t, originalSession.ID(), retrievedSession.ID())
	assert.Equal(t, containerID, retrievedSession.ContainerID())

	// 존재하지 않는 세션 조회
	_, err = manager.GetSession("non-existent-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// 정리
	manager.Shutdown()
	mockClient.AssertExpectations(t)
}

// TestPTYSessionManagerGetStats 통계 조회 테스트
func TestPTYSessionManagerGetStats(t *testing.T) {
	mockClient := &MockPTYClient{}
	containerID := "test-container-123"
	maxSessions := 10

	manager := NewPTYSessionManager(mockClient, maxSessions)

	// 초기 통계
	stats := manager.GetStats()
	assert.NotNil(t, stats)
	assert.Equal(t, 0, stats.TotalSessions)
	assert.Equal(t, 0, stats.ActiveSessions)
	assert.Equal(t, maxSessions, stats.MaxSessions)

	// 정리
	manager.Shutdown()
}

// TestPTYSessionManagerRemoveSession 세션 제거 테스트
func TestPTYSessionManagerRemoveSession(t *testing.T) {
	mockClient := &MockPTYClient{}
	containerID := "test-container-123"
	execID := "exec-id-456"
	maxSessions := 10

	// Mock 설정
	mockConn := NewMockReadWriteCloser("test output")
	hijackedResp := types.HijackedResponse{
		Conn:   mockConn,
		Reader: bufio.NewReader(strings.NewReader("test output")),
	}

	mockClient.On("ContainerInspect", mock.Anything, containerID).
		Return(types.ContainerJSON{}, nil)
	mockClient.On("ContainerExecCreate", mock.Anything, containerID, mock.AnythingOfType("ExecConfig")).
		Return(types.IDResponse{ID: execID}, nil)
	mockClient.On("ContainerExecStart", mock.Anything, execID, mock.AnythingOfType("ExecStartConfig")).
		Return(hijackedResp, nil)

	manager := NewPTYSessionManager(mockClient, maxSessions)
	ctx := context.Background()

	// 세션 생성
	session, err := manager.CreateSession(ctx, containerID)
	assert.NoError(t, err)
	assert.Equal(t, 1, manager.GetSessionCount())

	// 세션 제거
	err = manager.RemoveSession(session.ID())
	assert.NoError(t, err)
	assert.Equal(t, 0, manager.GetSessionCount())

	// 존재하지 않는 세션 제거
	err = manager.RemoveSession("non-existent-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// 정리
	manager.Shutdown()
	mockClient.AssertExpectations(t)
}

// 벤치마크 테스트들

// BenchmarkPTYSessionCreate 세션 생성 성능 테스트
func BenchmarkPTYSessionCreate(b *testing.B) {
	mockClient := &MockPTYClient{}
	containerID := "test-container-123"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		session := NewPTYSession(containerID, mockClient)
		_ = session
	}
}

// BenchmarkPTYSessionManagerOperations 세션 관리자 작업 성능 테스트
func BenchmarkPTYSessionManagerOperations(b *testing.B) {
	mockClient := &MockPTYClient{}
	maxSessions := 1000

	manager := NewPTYSessionManager(mockClient, maxSessions)
	defer manager.Shutdown()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stats := manager.GetStats()
		_ = stats
	}
}