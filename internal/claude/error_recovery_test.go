package claude

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)


func TestErrorClassifier_ClassifyError(t *testing.T) {
	tests := []struct {
		name           string
		error          error
		expectedType   RecoveryErrorType
		expectedAction RecoveryAction
	}{
		{
			name:           "connection refused error",
			error:          errors.New("connection refused"),
			expectedType:   RecoveryErrorTypeTransient,
			expectedAction: ActionRetry,
		},
		{
			name:           "timeout error",
			error:          errors.New("request timeout"),
			expectedType:   RecoveryErrorTypeTransient,
			expectedAction: ActionRetry,
		},
		{
			name:           "permission denied error",
			error:          errors.New("permission denied"),
			expectedType:   RecoveryErrorTypePermanent,
			expectedAction: ActionFail,
		},
		{
			name:           "process exited error",
			error:          errors.New("process exited with code 1"),
			expectedType:   RecoveryErrorTypeProcess,
			expectedAction: ActionRestart,
		},
		{
			name:           "out of memory error",
			error:          errors.New("out of memory"),
			expectedType:   RecoveryErrorTypeResource,
			expectedAction: ActionCircuitBreak,
		},
		{
			name:           "unknown error",
			error:          errors.New("some unknown error"),
			expectedType:   RecoveryErrorTypeUnknown,
			expectedAction: ActionIgnore,
		},
	}

	classifier := NewRecoveryErrorClassifier()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errorType, action := classifier.ClassifyError(tt.error)
			assert.Equal(t, tt.expectedType, errorType)
			assert.Equal(t, tt.expectedAction, action)
		})
	}
}

func TestExponentialBackoff(t *testing.T) {
	policy := &RecoveryPolicy{
		RestartInterval:   1 * time.Second,
		BackoffMultiplier: 2.0,
		MaxBackoff:        10 * time.Second,
	}

	backoff := NewExponentialBackoff(policy)

	// 첫 번째 시도
	duration1 := backoff.NextBackoff()
	assert.Equal(t, 1*time.Second, duration1)
	assert.Equal(t, 1, backoff.GetAttempts())

	// 두 번째 시도
	duration2 := backoff.NextBackoff()
	assert.Equal(t, 2*time.Second, duration2)
	assert.Equal(t, 2, backoff.GetAttempts())

	// 세 번째 시도
	duration3 := backoff.NextBackoff()
	assert.Equal(t, 4*time.Second, duration3)
	assert.Equal(t, 3, backoff.GetAttempts())

	// 최대값 확인을 위해 여러 번 시도
	for i := 0; i < 5; i++ {
		backoff.NextBackoff()
	}
	
	duration := backoff.NextBackoff()
	assert.Equal(t, 10*time.Second, duration) // 최대값에 도달

	// 리셋 테스트
	backoff.Reset()
	assert.Equal(t, 0, backoff.GetAttempts())
	
	durationAfterReset := backoff.NextBackoff()
	assert.Equal(t, 1*time.Second, durationAfterReset)
}

func TestCircuitBreaker(t *testing.T) {
	config := &CircuitBreakerConfig{
		FailureThreshold:         3,
		RecoveryTimeout:          1 * time.Second,
		SuccessThreshold:         2,
		RequestVolumeThreshold:   5,
		ErrorPercentageThreshold: 50.0,
	}

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // 테스트 중 로그 출력 최소화

	cb := NewCircuitBreaker(config, logger)

	// 초기 상태는 Closed
	assert.True(t, cb.IsClosed())
	assert.True(t, cb.Allow())

	// 실패 기록 (임계값까지)
	for i := 0; i < 3; i++ {
		cb.RecordError()
	}

	// 아직 요청 볼륨이 부족하므로 여전히 Closed
	assert.True(t, cb.IsClosed())

	// 더 많은 요청과 실패를 기록하여 임계값 초과
	for i := 0; i < 5; i++ {
		cb.Allow()
		cb.RecordError()
	}

	// 이제 Open 상태여야 함
	assert.True(t, cb.IsOpen())
	assert.False(t, cb.Allow())

	// 복구 타임아웃 대기
	time.Sleep(1100 * time.Millisecond)

	// Half-Open 상태로 전환되어 요청 허용
	assert.True(t, cb.Allow())
	assert.True(t, cb.IsHalfOpen())

	// 성공 기록으로 Closed 상태로 복귀
	for i := 0; i < 2; i++ {
		cb.RecordSuccess()
	}

	assert.True(t, cb.IsClosed())
}

func TestErrorRecoveryManager_HandleError(t *testing.T) {
	mockPM := new(MockProcessManager)
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	policy := &RecoveryPolicy{
		MaxRestarts:       3,
		RestartInterval:   100 * time.Millisecond,
		BackoffMultiplier: 2.0,
		MaxBackoff:        1 * time.Second,
		CircuitBreakerConfig: &CircuitBreakerConfig{
			FailureThreshold:         5,
			RecoveryTimeout:          1 * time.Second,
			SuccessThreshold:         3,
			RequestVolumeThreshold:   10,
			ErrorPercentageThreshold: 50.0,
		},
		Enabled: true,
	}

	erm := NewErrorRecoveryManager(policy, mockPM, logger)
	defer erm.Stop()

	tests := []struct {
		name           string
		error          error
		expectedAction RecoveryAction
	}{
		{
			name:           "transient error should retry",
			error:          errors.New("connection refused"),
			expectedAction: ActionRetry,
		},
		{
			name:           "permanent error should fail",
			error:          errors.New("permission denied"),
			expectedAction: ActionFail,
		},
		{
			name:           "process error should restart",
			error:          errors.New("process exited"),
			expectedAction: ActionRestart,
		},
		{
			name:           "resource error should circuit break",
			error:          errors.New("out of memory"),
			expectedAction: ActionCircuitBreak,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := erm.HandleError(tt.error)
			assert.Equal(t, tt.expectedAction, action)
		})
	}
}

func TestErrorRecoveryManager_Restart(t *testing.T) {
	mockPM := new(MockProcessManager)
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	policy := DefaultRecoveryPolicy()
	policy.RestartInterval = 10 * time.Millisecond // 테스트를 위해 짧게 설정

	erm := NewErrorRecoveryManager(policy, mockPM, logger)
	defer erm.Stop()

	ctx := context.Background()

	// 프로세스가 실행 중이라고 가정
	mockPM.On("IsRunning").Return(true)
	mockPM.On("Stop", mock.AnythingOfType("time.Duration")).Return(nil)
	mockPM.On("Start", mock.AnythingOfType("*context.emptyCtx"), mock.AnythingOfType("*claude.ProcessConfig")).Return(nil)

	// 재시작 테스트
	err := erm.Restart(ctx)
	assert.NoError(t, err)

	// 모킹 검증
	mockPM.AssertExpectations(t)

	// 통계 확인
	stats := erm.GetRecoveryStats()
	assert.Equal(t, int64(1), stats.RestartCount)
}

func TestErrorRecoveryManager_RestartLimits(t *testing.T) {
	mockPM := new(MockProcessManager)
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	policy := DefaultRecoveryPolicy()
	policy.MaxRestarts = 2
	policy.RestartInterval = 10 * time.Millisecond

	erm := NewErrorRecoveryManager(policy, mockPM, logger)
	defer erm.Stop()

	ctx := context.Background()

	// 프로세스 모킹
	mockPM.On("IsRunning").Return(true)
	mockPM.On("Stop", mock.AnythingOfType("time.Duration")).Return(nil)
	mockPM.On("Start", mock.AnythingOfType("*context.emptyCtx"), mock.AnythingOfType("*claude.ProcessConfig")).Return(nil)

	// 최대 재시작 횟수까지 재시작
	for i := 0; i < 2; i++ {
		err := erm.Restart(ctx)
		assert.NoError(t, err)
	}

	// 더 이상 재시작이 허용되지 않아야 함
	processErr := errors.New("process exited")
	action := erm.HandleError(processErr)
	assert.Equal(t, ActionFail, action)
}

func TestSlidingWindow(t *testing.T) {
	window := newSlidingWindow(1 * time.Second)

	// 성공 기록
	window.RecordSuccess()
	window.RecordSuccess()
	
	// 실패 기록
	window.RecordFailure()

	assert.Equal(t, int64(3), window.TotalRequests())
	assert.Equal(t, int64(1), window.TotalFailures())
	assert.InDelta(t, 33.33, window.ErrorPercentage(), 0.01)

	// 리셋 테스트
	window.Reset()
	assert.Equal(t, int64(0), window.TotalRequests())
	assert.Equal(t, int64(0), window.TotalFailures())
	assert.Equal(t, 0.0, window.ErrorPercentage())
}

func TestRecoveryStats(t *testing.T) {
	stats := NewRecoveryStats()

	// 에러 증가
	stats.IncrementError(RecoveryErrorTypeTransient)
	stats.IncrementError(RecoveryErrorTypeProcess)
	stats.IncrementError(RecoveryErrorTypeTransient)

	assert.Equal(t, int64(3), stats.TotalErrors)
	assert.Equal(t, int64(2), stats.ErrorsByType[RecoveryErrorTypeTransient])
	assert.Equal(t, int64(1), stats.ErrorsByType[RecoveryErrorTypeProcess])

	// 액션 증가
	stats.IncrementAction(ActionRetry)
	stats.IncrementAction(ActionRestart)

	assert.Equal(t, int64(1), stats.ActionsByType[ActionRetry])
	assert.Equal(t, int64(1), stats.ActionsByType[ActionRestart])

	// 재시작 증가
	beforeRestart := time.Now()
	stats.IncrementRestart()
	
	assert.Equal(t, int64(1), stats.RestartCount)
	assert.True(t, stats.LastRestart.After(beforeRestart))

	// 성공적인 실행 증가
	stats.IncrementSuccessfulRun()
	assert.Equal(t, int64(1), stats.SuccessfulRuns)
}

func TestErrorRecoveryManager_EnableDisable(t *testing.T) {
	mockPM := new(MockProcessManager)
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	policy := DefaultRecoveryPolicy()
	policy.Enabled = false // 비활성화로 시작

	erm := NewErrorRecoveryManager(policy, mockPM, logger)
	defer erm.Stop()

	// 비활성화 상태에서는 모든 에러를 무시
	assert.False(t, erm.IsEnabled())
	action := erm.HandleError(errors.New("some error"))
	assert.Equal(t, ActionIgnore, action)

	// 활성화
	ctx := context.Background()
	err := erm.Start(ctx)
	assert.NoError(t, err)
	assert.True(t, erm.IsEnabled())

	// 이제 에러 처리가 동작해야 함
	action = erm.HandleError(errors.New("connection refused"))
	assert.Equal(t, ActionRetry, action)

	// 비활성화
	err = erm.Stop()
	assert.NoError(t, err)
	assert.False(t, erm.IsEnabled())
}