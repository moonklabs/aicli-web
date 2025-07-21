package claude

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
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

func TestNewHealthChecker(t *testing.T) {
	t.Run("with logger", func(t *testing.T) {
		logger := logrus.New()
		hc := NewHealthChecker(logger)
		assert.NotNil(t, hc)
		
		status := hc.GetHealthStatus()
		assert.False(t, status.Healthy)
		assert.Contains(t, status.Message, "헬스체크가 아직 수행되지 않았습니다")
	})

	t.Run("without logger", func(t *testing.T) {
		hc := NewHealthChecker(nil)
		assert.NotNil(t, hc)
	})
}

func TestHealthChecker_CheckHealth(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	t.Run("healthy process", func(t *testing.T) {
		hc := NewHealthChecker(logger)
		mockPM := new(MockProcessManager)
		
		mockPM.On("IsRunning").Return(true)
		mockPM.On("HealthCheck").Return(nil)
		
		err := hc.CheckHealth(context.Background(), mockPM)
		assert.NoError(t, err)
		
		status := hc.GetHealthStatus()
		assert.True(t, status.Healthy)
		assert.Contains(t, status.Message, "프로세스가 정상적으로 작동 중입니다")
		assert.NotNil(t, status.Metrics)
		
		mockPM.AssertExpectations(t)
	})

	t.Run("process not running", func(t *testing.T) {
		hc := NewHealthChecker(logger)
		mockPM := new(MockProcessManager)
		
		mockPM.On("IsRunning").Return(false)
		
		err := hc.CheckHealth(context.Background(), mockPM)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "프로세스가 실행 중이 아닙니다")
		
		status := hc.GetHealthStatus()
		assert.False(t, status.Healthy)
		assert.Contains(t, status.Message, "프로세스가 실행 중이 아닙니다")
		
		mockPM.AssertExpectations(t)
	})

	t.Run("health check failed", func(t *testing.T) {
		hc := NewHealthChecker(logger)
		mockPM := new(MockProcessManager)
		
		mockPM.On("IsRunning").Return(true)
		mockPM.On("HealthCheck").Return(assert.AnError)
		
		err := hc.CheckHealth(context.Background(), mockPM)
		assert.Error(t, err)
		
		status := hc.GetHealthStatus()
		assert.False(t, status.Healthy)
		assert.Contains(t, status.Message, "헬스체크 실패")
		
		mockPM.AssertExpectations(t)
	})
}

func TestHealthChecker_RegisterHandler(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	
	hc := NewHealthChecker(logger)
	mockPM := new(MockProcessManager)
	
	handlerCalled := false
	var handlerStatus HealthStatus
	
	// 핸들러 등록
	hc.RegisterHealthHandler(func(status HealthStatus) {
		handlerCalled = true
		handlerStatus = status
	})
	
	// 헬스체크 수행
	mockPM.On("IsRunning").Return(true)
	mockPM.On("HealthCheck").Return(nil)
	
	err := hc.CheckHealth(context.Background(), mockPM)
	assert.NoError(t, err)
	
	// 핸들러가 호출되었는지 확인
	assert.True(t, handlerCalled)
	assert.True(t, handlerStatus.Healthy)
	
	mockPM.AssertExpectations(t)
}

func TestHealthChecker_StartStop(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	
	hc := NewHealthChecker(logger)
	mockPM := new(MockProcessManager)
	
	// 헬스체크가 여러 번 호출될 것을 예상
	mockPM.On("IsRunning").Return(true)
	mockPM.On("HealthCheck").Return(nil)
	
	ctx, cancel := context.WithCancel(context.Background())
	
	// 헬스체크 시작
	go hc.Start(ctx, mockPM, 50*time.Millisecond)
	
	// 잠시 실행 후 중지
	time.Sleep(150 * time.Millisecond)
	cancel()
	
	// Stop 메서드도 호출
	hc.Stop()
	
	// 약간의 시간을 두고 정리 대기
	time.Sleep(50 * time.Millisecond)
}

func TestHealthChecker_MultipleHandlers(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	
	hc := NewHealthChecker(logger)
	mockPM := new(MockProcessManager)
	
	handler1Called := false
	handler2Called := false
	
	// 여러 핸들러 등록
	hc.RegisterHealthHandler(func(status HealthStatus) {
		handler1Called = true
	})
	
	hc.RegisterHealthHandler(func(status HealthStatus) {
		handler2Called = true
	})
	
	// 헬스체크 수행
	mockPM.On("IsRunning").Return(true)
	mockPM.On("HealthCheck").Return(nil)
	
	err := hc.CheckHealth(context.Background(), mockPM)
	assert.NoError(t, err)
	
	// 모든 핸들러가 호출되었는지 확인
	assert.True(t, handler1Called)
	assert.True(t, handler2Called)
	
	mockPM.AssertExpectations(t)
}

func TestHealthStatus_Transitions(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	
	hc := NewHealthChecker(logger)
	mockPM := new(MockProcessManager)
	
	var statusHistory []bool
	
	// 상태 변경 추적 핸들러
	hc.RegisterHealthHandler(func(status HealthStatus) {
		statusHistory = append(statusHistory, status.Healthy)
	})
	
	// 초기 상태: 건강함
	mockPM.On("IsRunning").Return(true).Once()
	mockPM.On("HealthCheck").Return(nil).Once()
	_ = hc.CheckHealth(context.Background(), mockPM)
	
	// 상태 변경: 비정상
	mockPM.On("IsRunning").Return(true).Once()
	mockPM.On("HealthCheck").Return(assert.AnError).Once()
	_ = hc.CheckHealth(context.Background(), mockPM)
	
	// 상태 변경: 다시 정상
	mockPM.On("IsRunning").Return(true).Once()
	mockPM.On("HealthCheck").Return(nil).Once()
	_ = hc.CheckHealth(context.Background(), mockPM)
	
	// 상태 이력 확인
	assert.Equal(t, []bool{true, false, true}, statusHistory)
	
	mockPM.AssertExpectations(t)
}