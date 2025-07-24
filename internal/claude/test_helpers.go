package claude

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"
)

// MockProcessManager 테스트용 프로세스 매니저 모의 객체
type MockProcessManager struct {
	mock.Mock
}

func (m *MockProcessManager) Start(ctx context.Context, config *ProcessConfig) error {
	args := m.Called(ctx, config)
	return args.Error(0)
}

func (m *MockProcessManager) Stop(timeout time.Duration) error {
	args := m.Called(timeout)
	return args.Error(0)
}

func (m *MockProcessManager) Kill() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockProcessManager) IsRunning() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockProcessManager) GetStatus() ProcessStatus {
	args := m.Called()
	return args.Get(0).(ProcessStatus)
}

func (m *MockProcessManager) GetPID() int {
	args := m.Called()
	return args.Int(0)
}

func (m *MockProcessManager) Wait() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockProcessManager) HealthCheck() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockProcessManager) RestartProcess(identifier string) error {
	args := m.Called(identifier)
	return args.Error(0)
}