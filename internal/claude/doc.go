// Package claude는 Claude CLI와의 통합을 담당합니다.
//
// 이 패키지는 Claude CLI 프로세스 관리, 상태 추적, 입출력 처리 등의
// 핵심 기능을 제공합니다.
//
// 주요 컴포넌트:
//
// ProcessManager - Claude CLI 프로세스의 생명주기를 관리합니다:
//   - 프로세스 시작/중지/강제종료
//   - 상태 추적 및 헬스체크
//   - 시그널 처리 및 우아한 종료
//   - 입출력 리다이렉션
//
// ProcessManagerV2 - 향상된 프로세스 관리자:
//   - 상태 머신 기반 상태 관리
//   - 프로세스 메트릭 수집
//   - 하트비트 모니터링
//   - 컨텍스트 기반 타임아웃
//
// StateMachine - 프로세스 상태 전환 관리:
//   - 유효한 상태 전환 검증
//   - 상태 변경 이벤트 알림
//   - 동시성 안전 보장
//
// 사용 예제:
//
//	logger := logrus.New()
//	pm := claude.NewProcessManagerV2(logger)
//	
//	config := &claude.ProcessConfig{
//	    Command:    "claude",
//	    Args:       []string{"chat", "--no-stream"},
//	    WorkingDir: "/workspace",
//	}
//	
//	err := pm.Start(context.Background(), config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	
//	// 프로세스 상태 확인
//	if pm.IsRunning() {
//	    fmt.Printf("Claude CLI 실행 중 (PID: %d)\n", pm.GetPID())
//	}
//	
//	// 정상 종료
//	err = pm.Stop(30 * time.Second)
package claude