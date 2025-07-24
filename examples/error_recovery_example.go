//go:build example
// +build example

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/aicli/aicli-web/internal/claude"
)

func main() {
	// 로거 설정
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	// 프로세스 관리자 생성
	processManager := claude.NewProcessManager(logger)

	// 복구 정책 설정
	policy := &claude.RecoveryPolicy{
		MaxRestarts:       5,
		RestartInterval:   30 * time.Second,
		BackoffMultiplier: 2.0,
		MaxBackoff:        5 * time.Minute,
		CircuitBreakerConfig: &claude.CircuitBreakerConfig{
			FailureThreshold:         3,
			RecoveryTimeout:          1 * time.Minute,
			SuccessThreshold:         2,
			RequestVolumeThreshold:   10,
			ErrorPercentageThreshold: 60.0,
		},
		Enabled: true,
	}

	// 에러 복구 관리자 생성
	// errorRecovery := claude.NewErrorRecoveryManager(policy, processManager, logger)
	_ = policy
	_ = processManager

	// 이벤트 핸들러 설정
	errorRecovery.SetEventHandlers(
		// 복구 시작 핸들러
		func(err error) {
			log.Printf("복구 프로세스가 시작되었습니다. 에러: %v", err)
		},
		// 복구 완료 핸들러
		func(err error, success bool) {
			if success {
				log.Printf("복구가 성공적으로 완료되었습니다")
			} else {
				log.Printf("복구가 실패했습니다. 에러: %v", err)
			}
		},
		// 재시작 핸들러
		func(restartCount int64) {
			log.Printf("프로세스가 재시작되었습니다 (총 %d번째)", restartCount)
		},
	)

	ctx := context.Background()

	// 에러 복구 시스템 시작
	if err := errorRecovery.Start(ctx); err != nil {
		log.Fatalf("에러 복구 시스템 시작 실패: %v", err)
	}

	// 프로세스 시작
	config := &claude.ProcessConfig{
		Command:    "claude",
		Args:       []string{"--workspace", "./workspace"},
		WorkingDir: "./workspace",
		Environment: map[string]string{
			"CLAUDE_API_KEY": "your-api-key",
		},
		Timeout: 60 * time.Second,
	}

	if err := processManager.Start(ctx, config); err != nil {
		log.Fatalf("프로세스 시작 실패: %v", err)
	}

	// 다양한 에러 시나리오 시뮬레이션
	simulateErrors(errorRecovery)

	// 통계 출력
	printStats(errorRecovery)

	// 정리
	if err := errorRecovery.Stop(); err != nil {
		log.Printf("에러 복구 시스템 중지 중 오류: %v", err)
	}
}

// simulateErrors 다양한 에러 시나리오를 시뮬레이션합니다
func simulateErrors(errorRecovery claude.ErrorRecovery) {
	errors := []error{
		fmt.Errorf("connection refused"),      // 재시도
		fmt.Errorf("request timeout"),        // 재시도
		fmt.Errorf("process exited"),         // 재시작
		fmt.Errorf("permission denied"),      // 실패
		fmt.Errorf("out of memory"),          // 회로 차단
		fmt.Errorf("temporary failure"),      // 재시도
		fmt.Errorf("service unavailable"),    // 재시도
		fmt.Errorf("rate limit exceeded"),    // 재시도
	}

	fmt.Println("\n=== 에러 시나리오 시뮬레이션 ===")

	for i, err := range errors {
		fmt.Printf("\n%d. 에러 처리: %v\n", i+1, err)
		
		action := errorRecovery.HandleError(err)
		fmt.Printf("   복구 액션: %s\n", action)

		// 재시작 액션인 경우 실제 재시작 수행
		if action == claude.ActionRestart {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			if err := errorRecovery.Restart(ctx); err != nil {
				fmt.Printf("   재시작 실패: %v\n", err)
			} else {
				fmt.Printf("   재시작 성공\n")
			}
			cancel()
		}

		// 잠시 대기
		time.Sleep(100 * time.Millisecond)
	}
}

// printStats 통계를 출력합니다
func printStats(errorRecovery claude.ErrorRecovery) {
	stats := errorRecovery.GetRecoveryStats()

	fmt.Println("\n=== 에러 복구 통계 ===")
	fmt.Printf("총 에러 수: %d\n", stats.TotalErrors)
	fmt.Printf("재시작 횟수: %d\n", stats.RestartCount)
	fmt.Printf("성공적인 실행: %d\n", stats.SuccessfulRuns)
	fmt.Printf("평균 가동 시간: %v\n", stats.AverageUptime)

	if !stats.LastRestart.IsZero() {
		fmt.Printf("마지막 재시작: %v\n", stats.LastRestart.Format(time.RFC3339))
	}

	if stats.LastError != nil {
		fmt.Printf("마지막 에러: %v\n", stats.LastError)
	}

	fmt.Println("\n에러 타입별 통계:")
	for errorType, count := range stats.ErrorsByType {
		fmt.Printf("  %s: %d\n", errorType, count)
	}

	fmt.Println("\n액션별 통계:")
	for action, count := range stats.ActionsByType {
		fmt.Printf("  %s: %d\n", action, count)
	}
}

// 고급 사용 예시: 커스텀 에러 분류 규칙 추가
func advancedUsage() {
	logger := logrus.New()
	processManager := claude.NewProcessManager(logger)
	policy := claude.DefaultRecoveryPolicy()
	// errorRecovery := claude.NewErrorRecoveryManager(policy, processManager, logger)
	_ = policy
	_ = processManager

	// 커스텀 분류 규칙 추가 (실제 구현에서는 이 기능을 제공해야 함)
	// classifier := errorRecovery.GetClassifier() // 이런 메서드가 있다고 가정
	// classifier.AddRule(claude.ErrorTypeAPI, claude.ClassificationRule{
	// 	ErrorPattern: "api quota exceeded",
	// 	Action:       claude.ActionRetry,
	// 	Retryable:    true,
	// 	BackoffType:  claude.BackoffExponential,
	// })

	fmt.Println("고급 사용 예시가 설정되었습니다")
}

// 모니터링 통합 예시
func monitoringIntegration(errorRecovery claude.ErrorRecovery) {
	// 별도 고루틴에서 주기적으로 통계 수집
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				stats := errorRecovery.GetRecoveryStats()
				
				// 외부 모니터링 시스템으로 메트릭 전송
				sendMetricsToMonitoring(stats)
				
				// 알람 조건 확인
				checkAlertConditions(stats)
			}
		}
	}()
}

// sendMetricsToMonitoring 외부 모니터링 시스템으로 메트릭을 전송합니다
func sendMetricsToMonitoring(stats *claude.RecoveryStats) {
	// 예: Prometheus, DataDog, CloudWatch 등으로 메트릭 전송
	fmt.Printf("메트릭 전송: 에러=%d, 재시작=%d\n", stats.TotalErrors, stats.RestartCount)
}

// checkAlertConditions 알람 조건을 확인합니다
func checkAlertConditions(stats *claude.RecoveryStats) {
	// 재시작 횟수가 임계값을 초과하면 알람
	if stats.RestartCount > 10 {
		fmt.Println("⚠️ 알람: 재시작 횟수가 임계값을 초과했습니다!")
	}

	// 에러율이 높으면 알람
	if stats.TotalErrors > 0 && stats.SuccessfulRuns > 0 {
		errorRate := float64(stats.TotalErrors) / float64(stats.SuccessfulRuns+stats.TotalErrors) * 100
		if errorRate > 50 {
			fmt.Printf("⚠️ 알람: 에러율이 %.2f%%로 높습니다!\n", errorRate)
		}
	}
}