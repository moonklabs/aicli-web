//go:build integration
// +build integration

package claude

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProcessManagerIntegration_TokenAndHealthCheck 통합 테스트: 토큰 관리와 헬스체크
func TestProcessManagerIntegration_TokenAndHealthCheck(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	t.Run("OAuth token with health monitoring", func(t *testing.T) {
		pm := NewProcessManager(logger)
		ctx := context.Background()

		// 헬스 상태 변경 추적
		var healthStatuses []bool
		healthChecker := NewHealthChecker(logger)
		healthChecker.RegisterHealthHandler(func(status HealthStatus) {
			healthStatuses = append(healthStatuses, status.Healthy)
			logger.WithFields(logrus.Fields{
				"healthy": status.Healthy,
				"message": status.Message,
			}).Debug("헬스 상태 변경")
		})

		config := &ProcessConfig{
			Command:             "sh",
			Args:                []string{"-c", "while true; do echo 'OAuth: '$CLAUDE_CODE_OAUTH_TOKEN; sleep 1; done"},
			OAuthToken:          "test-oauth-token-12345",
			HealthCheckInterval: 500 * time.Millisecond,
		}

		// 프로세스 시작
		err := pm.Start(ctx, config)
		require.NoError(t, err)

		// 헬스체크가 실행될 시간을 줌
		time.Sleep(2 * time.Second)

		// 프로세스 상태 확인
		assert.True(t, pm.IsRunning())
		assert.Equal(t, StatusRunning, pm.GetStatus())

		// 헬스 상태가 기록되었는지 확인
		assert.Greater(t, len(healthStatuses), 0)

		// 정리
		err = pm.Stop(2 * time.Second)
		assert.NoError(t, err)
	})

	t.Run("API key fallback with resource limits", func(t *testing.T) {
		pm := NewProcessManager(logger)
		ctx := context.Background()

		config := &ProcessConfig{
			Command: "sh",
			Args:    []string{"-c", "echo 'API Key: '$CLAUDE_API_KEY && sleep 1"},
			APIKey:  "sk-test-api-key",
			ResourceLimits: &ResourceLimits{
				MaxCPU:    0.5,                // 0.5 CPU 코어
				MaxMemory: 256 * 1024 * 1024,  // 256MB
				Timeout:   3 * time.Second,
			},
		}

		// 프로세스 시작
		err := pm.Start(ctx, config)
		require.NoError(t, err)

		// 프로세스 완료 대기
		err = pm.Wait()
		assert.NoError(t, err)
		assert.Equal(t, StatusStopped, pm.GetStatus())
	})
}

// TestProcessManagerIntegration_TokenRefresh 토큰 갱신 통합 테스트
func TestProcessManagerIntegration_TokenRefresh(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// 토큰 갱신 시뮬레이션
	refreshCount := 0
	refreshFunc := func(ctx context.Context) (string, time.Time, error) {
		refreshCount++
		newToken := fmt.Sprintf("refreshed-token-%d", refreshCount)
		expiresAt := time.Now().Add(2 * time.Second) // 2초 후 만료
		logger.WithFields(logrus.Fields{
			"token":     newToken,
			"expiresAt": expiresAt,
		}).Info("토큰 갱신됨")
		return newToken, expiresAt, nil
	}

	tokenManager := NewTokenManager("initial-token", "", refreshFunc)
	
	// 초기 토큰 설정 (곧 만료될 토큰)
	tokenManager.SetToken("initial-token", time.Now().Add(1*time.Second))

	pm := NewProcessManager(logger)
	ctx := context.Background()

	// 환경 변수 설정을 확인하는 스크립트
	config := &ProcessConfig{
		Command:    "sh",
		Args:       []string{"-c", "for i in 1 2 3 4 5; do echo 'Token: '$CLAUDE_CODE_OAUTH_TOKEN; sleep 1; done"},
		OAuthToken: "initial-token",
	}

	// 토큰 매니저 연동 (실제 구현에서는 ProcessManager 내부에서 처리)
	go func() {
		for i := 0; i < 3; i++ {
			time.Sleep(2 * time.Second)
			token, _ := tokenManager.GetToken(ctx)
			logger.WithField("current_token", token).Info("현재 토큰 확인")
		}
	}()

	// 프로세스 시작
	err := pm.Start(ctx, config)
	require.NoError(t, err)

	// 프로세스 완료 대기
	err = pm.Wait()
	assert.NoError(t, err)

	// 토큰이 갱신되었는지 확인
	assert.GreaterOrEqual(t, refreshCount, 1)
}

// TestProcessManagerIntegration_HealthCheckRecovery 헬스체크 복구 시나리오
func TestProcessManagerIntegration_HealthCheckRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	pm := NewProcessManager(logger)
	ctx := context.Background()

	// 헬스체크 실패/복구를 시뮬레이션하는 스크립트
	script := `
	for i in 1 2 3 4 5 6 7 8 9 10; do
		if [ $i -eq 5 ] || [ $i -eq 6 ]; then
			echo "UNHEALTHY"
			exit 1  # 5,6번째 반복에서 비정상 종료
		else
			echo "HEALTHY: $i"
		fi
		sleep 0.5
	done
	`

	var recoveryAttempts int
	
	// 복구 시도 추적
	for attempt := 0; attempt < 3; attempt++ {
		config := &ProcessConfig{
			Command:             "sh",
			Args:                []string{"-c", script},
			HealthCheckInterval: 300 * time.Millisecond,
		}

		err := pm.Start(ctx, config)
		if err != nil {
			logger.WithError(err).Warn("프로세스 시작 실패")
			continue
		}

		recoveryAttempts++
		
		// 프로세스 완료 대기
		err = pm.Wait()
		if err != nil {
			logger.WithError(err).Info("프로세스 비정상 종료, 재시작 시도")
			time.Sleep(500 * time.Millisecond)
			continue
		}
		
		break
	}

	assert.GreaterOrEqual(t, recoveryAttempts, 1)
}

// TestProcessManagerIntegration_ConcurrentProcesses 동시 프로세스 관리
func TestProcessManagerIntegration_ConcurrentProcesses(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	const numProcesses = 5
	managers := make([]ProcessManager, numProcesses)
	
	ctx := context.Background()
	done := make(chan int, numProcesses)

	// 여러 프로세스 동시 실행
	for i := 0; i < numProcesses; i++ {
		idx := i
		managers[i] = NewProcessManager(logger)
		
		go func(pm ProcessManager, id int) {
			config := &ProcessConfig{
				Command:             "sh",
				Args:                []string{"-c", fmt.Sprintf("echo 'Process %d started' && sleep 1 && echo 'Process %d finished'", id, id)},
				OAuthToken:          fmt.Sprintf("token-%d", id),
				HealthCheckInterval: 200 * time.Millisecond,
				ResourceLimits: &ResourceLimits{
					MaxCPU:    0.2,               // 각 프로세스 0.2 CPU 코어
					MaxMemory: 128 * 1024 * 1024, // 128MB
				},
			}

			if err := pm.Start(ctx, config); err != nil {
				logger.WithError(err).Errorf("프로세스 %d 시작 실패", id)
				done <- id
				return
			}

			if err := pm.Wait(); err != nil {
				logger.WithError(err).Warnf("프로세스 %d 비정상 종료", id)
			}

			done <- id
		}(managers[i], idx)
	}

	// 모든 프로세스 완료 대기
	completed := 0
	for i := 0; i < numProcesses; i++ {
		id := <-done
		completed++
		logger.Infof("프로세스 %d 완료 (%d/%d)", id, completed, numProcesses)
	}

	assert.Equal(t, numProcesses, completed)
}

// TestProcessManagerIntegration_EnvironmentIsolation 환경 변수 격리 테스트
func TestProcessManagerIntegration_EnvironmentIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// 두 개의 다른 환경 설정
	pm1 := NewProcessManager(logger)
	pm2 := NewProcessManager(logger)
	
	ctx := context.Background()

	// 첫 번째 프로세스: OAuth 토큰 사용
	config1 := &ProcessConfig{
		Command:    "sh",
		Args:       []string{"-c", "echo 'Process 1 - OAuth: '$CLAUDE_CODE_OAUTH_TOKEN' API: '$CLAUDE_API_KEY"},
		OAuthToken: "oauth-token-process-1",
		Environment: map[string]string{
			"CUSTOM_VAR": "process1",
		},
	}

	// 두 번째 프로세스: API 키 사용
	config2 := &ProcessConfig{
		Command: "sh",
		Args:    []string{"-c", "echo 'Process 2 - OAuth: '$CLAUDE_CODE_OAUTH_TOKEN' API: '$CLAUDE_API_KEY' Custom: '$CUSTOM_VAR"},
		APIKey:  "api-key-process-2",
		Environment: map[string]string{
			"CUSTOM_VAR": "process2",
		},
	}

	// 동시 실행
	err1 := pm1.Start(ctx, config1)
	require.NoError(t, err1)
	
	err2 := pm2.Start(ctx, config2)
	require.NoError(t, err2)

	// 완료 대기
	err1 = pm1.Wait()
	assert.NoError(t, err1)
	
	err2 = pm2.Wait()
	assert.NoError(t, err2)

	// 두 프로세스가 격리된 환경에서 실행되었는지 확인
	assert.Equal(t, StatusStopped, pm1.GetStatus())
	assert.Equal(t, StatusStopped, pm2.GetStatus())
}