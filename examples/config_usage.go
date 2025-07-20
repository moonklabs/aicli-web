package main

import (
	"fmt"
	"log"
	"os"

	"aicli-web/internal/config"
)

func main() {
	// 1. 싱글톤 ConfigManager 사용
	fmt.Println("=== Using Singleton ConfigManager ===")
	
	// 전역 설정 값 가져오기
	model := config.GetString("claude.model")
	fmt.Printf("Current Claude model: %s\n", model)
	
	temperature := config.GetFloat64("claude.temperature")
	fmt.Printf("Current temperature: %.2f\n", temperature)
	
	// 설정 값 변경
	err := config.Set("logging.level", "debug")
	if err != nil {
		log.Printf("Error setting logging level: %v", err)
	} else {
		fmt.Println("Logging level set to debug")
	}
	
	// 2. ConfigManager 인스턴스 직접 생성
	fmt.Println("\n=== Using ConfigManager Instance ===")
	
	cm, err := config.NewConfigManager()
	if err != nil {
		log.Fatalf("Failed to create ConfigManager: %v", err)
	}
	
	// 모든 설정 출력
	allSettings := cm.AllSettings()
	fmt.Println("All settings:")
	for key, value := range allSettings {
		fmt.Printf("  %s: %v\n", key, value)
	}
	
	// 3. 환경 변수 우선순위 테스트
	fmt.Println("\n=== Environment Variable Priority ===")
	
	// 환경 변수 설정
	os.Setenv("AICLI_OUTPUT_FORMAT", "json")
	
	// 새 ConfigManager 생성 (환경 변수 반영)
	cm2, err := config.NewConfigManager()
	if err != nil {
		log.Fatalf("Failed to create ConfigManager: %v", err)
	}
	
	format := cm2.GetString("output.format")
	fmt.Printf("Output format (from env): %s\n", format)
	
	// 파일로 설정하려고 시도 (환경 변수가 우선하므로 실패해야 함)
	err = cm2.Set("output.format", "table")
	if err != nil {
		fmt.Printf("Expected error: %v\n", err)
	}
	
	// 4. 설정 감시자 예제
	fmt.Println("\n=== Configuration Watcher ===")
	
	// 로깅 감시자 등록
	logger := log.New(os.Stdout, "[CONFIG] ", log.LstdFlags)
	loggingWatcher := config.NewLoggingWatcher(logger)
	cm.RegisterWatcher(loggingWatcher)
	
	// Claude 모델 변경 감시자
	claudeWatcher := config.NewClaudeWatcher(func(newModel string) {
		fmt.Printf("Claude model changed to: %s\n", newModel)
	})
	cm.RegisterWatcher(claudeWatcher)
	
	// 설정 변경 (감시자가 알림을 받음)
	err = cm.Set("claude.model", "claude-3-sonnet")
	if err != nil {
		log.Printf("Error setting claude model: %v", err)
	}
	
	// 5. 설정 검증
	fmt.Println("\n=== Configuration Validation ===")
	
	// 유효하지 않은 값 설정 시도
	err = cm.Set("claude.temperature", 2.0) // 범위 초과
	if err != nil {
		fmt.Printf("Validation error (expected): %v\n", err)
	}
	
	err = cm.Set("output.format", "xml") // 지원하지 않는 형식
	if err != nil {
		fmt.Printf("Validation error (expected): %v\n", err)
	}
	
	// 6. 타입 변환
	fmt.Println("\n=== Type Conversion ===")
	
	// 문자열을 적절한 타입으로 변환
	val, err := cm.ConvertValue("claude.timeout", "60")
	if err == nil {
		fmt.Printf("Converted timeout: %v (type: %T)\n", val, val)
	}
	
	val, err = cm.ConvertValue("workspace.auto_sync", "false")
	if err == nil {
		fmt.Printf("Converted auto_sync: %v (type: %T)\n", val, val)
	}
	
	// 7. 전체 설정을 구조체로 가져오기
	fmt.Println("\n=== Get Full Configuration ===")
	
	cfg, err := cm.GetConfig()
	if err != nil {
		log.Fatalf("Failed to get config: %v", err)
	}
	
	fmt.Printf("Claude Settings:\n")
	fmt.Printf("  Model: %s\n", cfg.Claude.Model)
	fmt.Printf("  Temperature: %.2f\n", cfg.Claude.Temperature)
	fmt.Printf("  Timeout: %d seconds\n", cfg.Claude.Timeout)
	
	fmt.Printf("Workspace Settings:\n")
	fmt.Printf("  Default Path: %s\n", cfg.Workspace.DefaultPath)
	fmt.Printf("  Auto Sync: %v\n", cfg.Workspace.AutoSync)
	fmt.Printf("  Max Projects: %d\n", cfg.Workspace.MaxProjects)
	
	// 정리
	os.Unsetenv("AICLI_OUTPUT_FORMAT")
}