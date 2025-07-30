package integration

import (
	"fmt"
	"os"
	"testing"
	"time"
)

// TestMain 통합 테스트 메인 함수
// 테스트 환경 설정, 실행, 정리 등을 관리합니다.
func TestMain(m *testing.M) {
	fmt.Println("🧪 AICode Manager 통합 테스트 스위트 시작")
	fmt.Println("=" + fmt.Sprintf("%s", time.Now().Format("2006-01-02 15:04:05")) + "=")
	fmt.Println()

	// 환경 변수 설정
	setupTestEnvironment()

	// 테스트 실행
	fmt.Println("📋 테스트 실행 순서:")
	fmt.Println("   1. Agent Integration Test Suite - 에이전트 생명주기 E2E 테스트")
	fmt.Println("   2. Git Integration Test Suite - Git worktree 통합 테스트")
	fmt.Println("   3. API Integration Test Suite - API 엔드포인트 통합 테스트")
	fmt.Println("   4. Performance Test Suite - 성능 및 부하 테스트")
	fmt.Println()

	// 시작 시간 기록
	startTime := time.Now()

	// 테스트 실행
	code := m.Run()

	// 종료 시간 및 결과 출력
	duration := time.Since(startTime)
	fmt.Println()
	fmt.Println("=" + fmt.Sprintf("테스트 완료 - 소요시간: %v", duration) + "=")
	
	if code == 0 {
		fmt.Println("✅ 모든 통합 테스트가 성공적으로 완료되었습니다!")
	} else {
		fmt.Println("❌ 일부 통합 테스트가 실패했습니다.")
	}

	// 정리 작업
	cleanupTestEnvironment()

	os.Exit(code)
}

// setupTestEnvironment 테스트 환경 설정
func setupTestEnvironment() {
	fmt.Println("🔧 테스트 환경 설정 중...")

	// 환경 변수 설정
	testEnvVars := map[string]string{
		"GO_ENV":           "test",
		"LOG_LEVEL":        "error", // 테스트 중 로그 최소화
		"DB_PATH":          ":memory:", // 메모리 DB 사용
		"DISABLE_AUTH":     "true",     // 인증 비활성화
		"TEST_MODE":        "true",
		"DOCKER_HOST":      "", // 기본 Docker 호스트 사용
	}

	for key, value := range testEnvVars {
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}

	// 테스트 디렉토리 확인
	if _, err := os.Stat("test/integration"); os.IsNotExist(err) {
		fmt.Println("   ⚠️  integration 테스트 디렉토리가 없습니다.")
	}

	fmt.Println("   ✅ 테스트 환경 설정 완료")
}

// cleanupTestEnvironment 테스트 환경 정리
func cleanupTestEnvironment() {
	fmt.Println("🧹 테스트 환경 정리 중...")
	
	// 임시 파일들 정리는 각 테스트에서 수행
	// 여기서는 전역 정리만 수행
	
	fmt.Println("   ✅ 테스트 환경 정리 완료")
}