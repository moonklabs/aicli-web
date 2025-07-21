// Package errors는 AICode Manager CLI의 통합된 에러 처리 시스템을 제공합니다.
//
// 이 패키지는 다음과 같은 기능을 제공합니다:
//
// 1. 에러 분류 및 타입 정의
// 2. 사용자 친화적인 에러 메시지 포맷팅
// 3. 진단 정보 수집 및 보고
// 4. 구조화된 로깅 시스템
// 5. 에러 복구 및 재시도 메커니즘
//
// # 기본 사용법
//
// 에러 생성:
//
//	err := errors.NewValidationError("필수 필드가 누락되었습니다", 
//		"--name 플래그를 추가하세요")
//
// 에러 포맷팅:
//
//	formatter := errors.NewHumanErrorFormatter(true, true)
//	formatted := formatter.Format(err)
//	fmt.Print(formatted)
//
// 진단 정보 포함:
//
//	collector := errors.NewDiagnosticCollector("/path/to/config", "1.0.0")
//	enrichedErr := errors.EnrichErrorWithDiagnostics(err, collector)
//
// 로깅:
//
//	logger, _ := errors.NewFileErrorLogger("/var/log/aicli.log", errors.LogLevelInfo)
//	logger.LogError(err)
//
// 재시도와 복구:
//
//	policy := errors.DefaultRetryPolicy()
//	manager := errors.NewRecoveryManager()
//	
//	err := errors.RetryWithRecovery(ctx, policy, manager, func(ctx context.Context, attempt int) error {
//		// 재시도할 작업
//		return doSomething()
//	})
//
// # 에러 타입
//
// 이 패키지는 다음과 같은 에러 타입을 정의합니다:
//
//   - ValidationError: 입력 검증 오류 (종료 코드: 1)
//   - ConfigError: 설정 관련 오류 (종료 코드: 2)
//   - NetworkError: 네트워크 연결 오류 (종료 코드: 3)
//   - FileSystemError: 파일 시스템 오류 (종료 코드: 4)
//   - PermissionError: 권한 오류 (종료 코드: 5)
//   - AuthenticationError: 인증 오류 (종료 코드: 6)
//   - ProcessError: 프로세스 실행 오류 (종료 코드: 7)
//   - NotFoundError: 리소스 미발견 오류 (종료 코드: 8)
//   - ConflictError: 충돌 상황 오류 (종료 코드: 9)
//   - InternalError: 내부 시스템 오류 (종료 코드: 127)
//
// # 포맷터
//
// 다양한 출력 형식을 지원합니다:
//
//   - HumanErrorFormatter: 사용자 친화적인 텍스트 형식 (색상, 아이콘 지원)
//   - JSONErrorFormatter: JSON 형식
//   - PlainErrorFormatter: 단순 텍스트 형식
//
// # 진단 정보
//
// 에러 발생 시 다음과 같은 진단 정보를 자동으로 수집합니다:
//
//   - 시스템 정보 (OS, 아키텍처, Go 버전 등)
//   - 환경 정보 (환경 변수, 터미널 설정 등)
//   - 설정 정보 (설정 파일 경로, 유효성 등)
//   - 프로세스 정보 (PID, 실행 파일 경로 등)
//
// # 로깅
//
// 계층적 로그 레벨을 지원합니다:
//
//   - SILENT: 로그 출력 안함
//   - ERROR: 에러만 출력
//   - WARN: 경고 이상 출력
//   - INFO: 정보 이상 출력
//   - DEBUG: 모든 로그 출력
//
// # 복구 전략
//
// 자동 에러 복구 전략을 제공합니다:
//
//   - ConfigRecoveryStrategy: 설정 파일 복구
//   - NetworkRecoveryStrategy: 네트워크 연결 복구
//   - ProcessRecoveryStrategy: 프로세스 재시작
//
// # 종료 코드
//
// 표준화된 종료 코드 체계를 제공하여 스크립트나 CI/CD에서
// 에러 타입을 구분할 수 있습니다.
package errors