// +build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/aicli/aicli-web/internal/claude"
)

// 이 파일은 프로세스 관리자 사용 예제입니다.
// go run process_example.go 로 실행할 수 있습니다.

func main() {
	// 로거 설정
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	// 프로세스 관리자 생성
	pm := claude.NewProcessManager(logger)

	// 예제 1: 간단한 명령어 실행
	fmt.Println("=== 예제 1: 간단한 명령어 실행 ===")
	runSimpleCommand(pm)

	// 예제 2: 장시간 실행되는 프로세스 관리
	fmt.Println("\n=== 예제 2: 장시간 실행 프로세스 ===")
	runLongRunningProcess(pm)

	// 예제 3는 태스크 요구사항에 없는 기능이므로 제거됨

	// 예제 4: 에러 처리
	fmt.Println("\n=== 예제 4: 에러 처리 ===")
	runWithErrorHandling(pm)
}

func runSimpleCommand(pm claude.ProcessManager) {
	ctx := context.Background()

	config := &claude.ProcessConfig{
		Command: "echo",
		Args:    []string{"Hello, AICode Manager!"},
	}

	if err := pm.Start(ctx, config); err != nil {
		log.Printf("프로세스 시작 실패: %v", err)
		return
	}

	fmt.Printf("프로세스 PID: %d\n", pm.GetPID())
	fmt.Printf("프로세스 상태: %s\n", pm.GetStatus())

	// 프로세스 완료 대기
	if err := pm.Wait(); err != nil {
		log.Printf("프로세스 대기 중 오류: %v", err)
	}

	fmt.Printf("최종 상태: %s\n", pm.GetStatus())
}

func runLongRunningProcess(pm claude.ProcessManager) {
	ctx := context.Background()

	// Claude CLI 시뮬레이션 (실제로는 claude 명령어 사용)
	config := &claude.ProcessConfig{
		Command: "sh",
		Args:    []string{"-c", "for i in {1..5}; do echo \"작업 중... $i\"; sleep 1; done"},
	}

	if err := pm.Start(ctx, config); err != nil {
		log.Printf("프로세스 시작 실패: %v", err)
		return
	}

	fmt.Printf("프로세스 시작됨 (PID: %d)\n", pm.GetPID())

	// 헬스체크
	go func() {
		for pm.IsRunning() {
			time.Sleep(2 * time.Second)
			if err := pm.HealthCheck(); err != nil {
				log.Printf("헬스체크 실패: %v", err)
			} else {
				fmt.Println("헬스체크: 정상")
			}
		}
	}()

	// 3초 후 정상 종료
	time.Sleep(3 * time.Second)
	fmt.Println("프로세스 중지 시도...")

	if err := pm.Stop(5 * time.Second); err != nil {
		log.Printf("프로세스 중지 실패: %v", err)
		// 강제 종료 시도
		if err := pm.Kill(); err != nil {
			log.Printf("프로세스 강제 종료 실패: %v", err)
		}
	}

	fmt.Printf("최종 상태: %s\n", pm.GetStatus())
}

// runWithIORedirection는 태스크 요구사항에 없는 기능이므로 제거됨

func runWithErrorHandling(pm claude.ProcessManager) {
	ctx := context.Background()

	// 존재하지 않는 명령어 실행 시도
	config := &claude.ProcessConfig{
		Command: "nonexistent_command",
		Args:    []string{"--help"},
	}

	if err := pm.Start(ctx, config); err != nil {
		fmt.Printf("예상된 오류 발생: %v\n", err)
		
		// ProcessError 타입 확인
		if perr, ok := err.(*claude.ProcessError); ok {
			fmt.Printf("에러 타입: %s\n", perr.Type)
			fmt.Printf("프로세스 상태: %s\n", perr.Status)
		}
	}

	// 정상 종료 테스트
	fmt.Println("\n정상 종료 테스트:")
	pm2 := claude.NewProcessManager(logrus.StandardLogger())

	config2 := &claude.ProcessConfig{
		Command: "sleep",
		Args:    []string{"2"},
	}

	if err := pm2.Start(ctx, config2); err != nil {
		log.Printf("프로세스 시작 실패: %v", err)
		return
	}

	// 프로세스 완료 대기
	if err := pm2.Wait(); err != nil {
		fmt.Printf("프로세스 실행 중 오류: %v\n", err)
	} else {
		fmt.Printf("프로세스가 정상적으로 완료됨\n")
	}
}