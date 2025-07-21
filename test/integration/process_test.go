//go:build integration
// +build integration

package integration

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"aicli-web/internal/claude"
	"aicli-web/test/helpers"
)

// TestProcessManagerIntegration은 프로세스 관리자의 통합 테스트를 수행합니다
func TestProcessManagerIntegration(t *testing.T) {
	env := helpers.NewTestEnvironment(t)
	helper := helpers.NewProcessTestHelper(env)
	
	t.Run("프로세스 생성 및 종료", func(t *testing.T) {
		ctx := context.Background()
		pm := claude.NewProcessManager(env.GetTestLogger())
		config := helper.CreateTestProcessConfig()
		
		// 프로세스 시작
		err := pm.Start(ctx, config)
		require.NoError(t, err)
		
		// 상태 확인
		assert.Equal(t, claude.StatusRunning, pm.GetStatus())
		assert.Greater(t, pm.GetPID(), 0)
		assert.True(t, pm.IsRunning())
		
		// 짧은 시간 대기
		time.Sleep(2 * time.Second)
		
		// 프로세스 종료
		err = pm.Stop(10 * time.Second)
		require.NoError(t, err)
		
		// 최종 상태 확인
		assert.Equal(t, claude.StatusStopped, pm.GetStatus())
		assert.False(t, pm.IsRunning())
	})
	
	t.Run("동시 다중 프로세스", func(t *testing.T) {
		const numProcesses = 5
		var wg sync.WaitGroup
		results := make(chan error, numProcesses)
		
		ctx := context.Background()
		
		// 여러 프로세스를 동시에 시작
		for i := 0; i < numProcesses; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				
				pm := claude.NewProcessManager(env.GetTestLogger())
				config := helper.CreateTestProcessConfig()
				
				// 프로세스 시작
				err := pm.Start(ctx, config)
				if err != nil {
					results <- fmt.Errorf("프로세스 %d 시작 실패: %w", id, err)
					return
				}
				
				// 상태 확인
				if !pm.IsRunning() {
					results <- fmt.Errorf("프로세스 %d가 실행되지 않음", id)
					return
				}
				
				// 잠깐 실행
				time.Sleep(1 * time.Second)
				
				// 프로세스 종료
				err = pm.Stop(5 * time.Second)
				if err != nil {
					results <- fmt.Errorf("프로세스 %d 종료 실패: %w", id, err)
					return
				}
				
				results <- nil
			}(i)
		}
		
		// 모든 고루틴 완료 대기
		wg.Wait()
		close(results)
		
		// 결과 확인
		errorCount := 0
		for err := range results {
			if err != nil {
				t.Error(err)
				errorCount++
			}
		}
		
		assert.Equal(t, 0, errorCount, "모든 프로세스가 성공적으로 실행되어야 함")
	})
	
	t.Run("프로세스 상태 전이", func(t *testing.T) {
		ctx := context.Background()
		pm := claude.NewProcessManager(env.GetTestLogger())
		config := helper.CreateTestProcessConfig()
		
		// 초기 상태 확인
		assert.Equal(t, claude.StatusIdle, pm.GetStatus())
		
		// 프로세스 시작
		err := pm.Start(ctx, config)
		require.NoError(t, err)
		
		// 실행 상태로 전이 대기
		waitCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		
		err = helper.WaitForProcessState(waitCtx, pm, claude.StatusRunning)
		require.NoError(t, err, "프로세스가 실행 상태로 전이되지 않음")
		
		// 프로세스 종료
		err = pm.Stop(5 * time.Second)
		require.NoError(t, err)
		
		// 종료 상태로 전이 확인
		assert.Equal(t, claude.StatusStopped, pm.GetStatus())
	})
	
	t.Run("프로세스 재시작", func(t *testing.T) {
		ctx := context.Background()
		pm := claude.NewProcessManager(env.GetTestLogger())
		config := helper.CreateTestProcessConfig()
		
		// 첫 번째 시작
		err := pm.Start(ctx, config)
		require.NoError(t, err)
		firstPID := pm.GetPID()
		
		// 종료
		err = pm.Stop(5 * time.Second)
		require.NoError(t, err)
		
		// 재시작
		err = pm.Start(ctx, config)
		require.NoError(t, err)
		secondPID := pm.GetPID()
		
		// PID가 다른지 확인 (새 프로세스)
		assert.NotEqual(t, firstPID, secondPID, "재시작 시 새로운 프로세스여야 함")
		
		// 정리
		err = pm.Stop(5 * time.Second)
		require.NoError(t, err)
	})
}

// TestProcessHealthCheck은 프로세스 헬스체크 테스트를 수행합니다
func TestProcessHealthCheck(t *testing.T) {
	env := helpers.NewTestEnvironment(t)
	helper := helpers.NewProcessTestHelper(env)
	
	t.Run("정상 프로세스 헬스체크", func(t *testing.T) {
		ctx := context.Background()
		pm := claude.NewProcessManager(env.GetTestLogger())
		config := helper.CreateTestProcessConfig()
		
		// 프로세스 시작
		err := pm.Start(ctx, config)
		require.NoError(t, err)
		defer pm.Stop(5 * time.Second)
		
		// 여러 번 헬스체크 수행
		for i := 0; i < 3; i++ {
			err = pm.HealthCheck()
			assert.NoError(t, err, "실행 중인 프로세스는 헬스체크 통과해야 함")
			time.Sleep(500 * time.Millisecond)
		}
	})
	
	t.Run("종료된 프로세스 헬스체크", func(t *testing.T) {
		ctx := context.Background()
		pm := claude.NewProcessManager(env.GetTestLogger())
		config := helper.CreateTestProcessConfig()
		
		// 프로세스 시작 후 종료
		err := pm.Start(ctx, config)
		require.NoError(t, err)
		
		err = pm.Stop(5 * time.Second)
		require.NoError(t, err)
		
		// 종료 후 헬스체크는 실패해야 함
		err = pm.HealthCheck()
		assert.Error(t, err, "종료된 프로세스는 헬스체크 실패해야 함")
	})
}

// TestProcessErrorHandling은 프로세스 에러 처리 테스트를 수행합니다
func TestProcessErrorHandling(t *testing.T) {
	env := helpers.NewTestEnvironment(t)
	
	t.Run("존재하지 않는 명령어", func(t *testing.T) {
		ctx := context.Background()
		pm := claude.NewProcessManager(env.GetTestLogger())
		
		config := &claude.ProcessConfig{
			Command: "nonexistent_command_12345",
			Args:    []string{"--help"},
		}
		
		// 시작 실패 확인
		err := pm.Start(ctx, config)
		assert.Error(t, err, "존재하지 않는 명령어는 시작 실패해야 함")
		
		// 에러 타입 확인
		if processErr, ok := err.(*claude.ProcessError); ok {
			assert.Equal(t, claude.ErrTypeStartFailed, processErr.Type)
		}
		
		assert.Equal(t, claude.StatusError, pm.GetStatus())
	})
	
	t.Run("프로세스 강제 종료", func(t *testing.T) {
		ctx := context.Background()
		pm := claude.NewProcessManager(env.GetTestLogger())
		
		config := &claude.ProcessConfig{
			Command: "sleep",
			Args:    []string{"10"},
		}
		
		// 프로세스 시작
		err := pm.Start(ctx, config)
		require.NoError(t, err)
		
		// 짧은 타임아웃으로 강제 종료
		err = pm.Stop(100 * time.Millisecond)
		
		// 강제 종료되었는지 확인
		assert.Equal(t, claude.StatusStopped, pm.GetStatus())
		assert.False(t, pm.IsRunning())
	})
}

// TestProcessResourceManagement는 프로세스 리소스 관리 테스트를 수행합니다
func TestProcessResourceManagement(t *testing.T) {
	env := helpers.NewTestEnvironment(t)
	helper := helpers.NewProcessTestHelper(env)
	
	t.Run("다중 프로세스 리소스 격리", func(t *testing.T) {
		const numProcesses = 3
		processes := make([]claude.ProcessManager, numProcesses)
		ctx := context.Background()
		
		// 여러 프로세스 생성
		for i := 0; i < numProcesses; i++ {
			processes[i] = claude.NewProcessManager(env.GetTestLogger())
			config := helper.CreateTestProcessConfig()
			
			err := processes[i].Start(ctx, config)
			require.NoError(t, err, "프로세스 %d 시작 실패", i)
		}
		
		// 모든 프로세스가 독립적으로 실행되는지 확인
		pids := make(map[int]bool)
		for i, pm := range processes {
			assert.True(t, pm.IsRunning(), "프로세스 %d가 실행되지 않음", i)
			
			pid := pm.GetPID()
			assert.Greater(t, pid, 0, "유효하지 않은 PID: 프로세스 %d", i)
			
			// PID 중복 확인
			assert.False(t, pids[pid], "중복된 PID: %d", pid)
			pids[pid] = true
		}
		
		// 모든 프로세스 정리
		for i, pm := range processes {
			err := pm.Stop(5 * time.Second)
			assert.NoError(t, err, "프로세스 %d 종료 실패", i)
		}
	})
	
	t.Run("프로세스 메모리 사용량 모니터링", func(t *testing.T) {
		if testing.Short() {
			t.Skip("짧은 테스트 모드에서 메모리 테스트 생략")
		}
		
		ctx := context.Background()
		pm := claude.NewProcessManager(env.GetTestLogger())
		config := helper.CreateTestProcessConfig()
		
		// 프로세스 시작
		err := pm.Start(ctx, config)
		require.NoError(t, err)
		defer pm.Stop(5 * time.Second)
		
		// 메모리 사용량 모니터링 (5초간)
		for i := 0; i < 5; i++ {
			memUsage := pm.GetMemoryUsage()
			t.Logf("메모리 사용량 %d초: %d bytes", i+1, memUsage)
			
			// 메모리 사용량이 합리적인 범위인지 확인 (1GB 미만)
			assert.Less(t, memUsage, int64(1024*1024*1024), "메모리 사용량이 너무 큼")
			
			time.Sleep(1 * time.Second)
		}
	})
}

// TestProcessConcurrency는 프로세스 동시성 테스트를 수행합니다
func TestProcessConcurrency(t *testing.T) {
	env := helpers.NewTestEnvironment(t)
	helper := helpers.NewProcessTestHelper(env)
	
	t.Run("동시 시작/종료", func(t *testing.T) {
		const concurrency = 10
		var wg sync.WaitGroup
		errors := make(chan error, concurrency*2) // start + stop
		
		ctx := context.Background()
		
		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				
				pm := claude.NewProcessManager(env.GetTestLogger())
				config := helper.CreateTestProcessConfig()
				
				// 시작
				err := pm.Start(ctx, config)
				if err != nil {
					errors <- fmt.Errorf("프로세스 %d 시작 실패: %w", id, err)
					return
				}
				
				// 잠깐 대기
				time.Sleep(100 * time.Millisecond)
				
				// 종료
				err = pm.Stop(5 * time.Second)
				if err != nil {
					errors <- fmt.Errorf("프로세스 %d 종료 실패: %w", id, err)
				}
			}(i)
		}
		
		wg.Wait()
		close(errors)
		
		// 에러 확인
		errorCount := 0
		for err := range errors {
			if err != nil {
				t.Error(err)
				errorCount++
			}
		}
		
		assert.Equal(t, 0, errorCount, "동시 시작/종료에서 에러 발생")
	})
}