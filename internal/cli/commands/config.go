package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"aicli-web/internal/config"
	"gopkg.in/yaml.v3"
)

// NewConfigCmd는 config 관련 명령어를 생성합니다.
func NewConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "설정 관리",
		Long:  `AICLI의 설정을 조회하고 변경합니다.

설정은 다음 순서로 우선순위를 가집니다:
  1. 명령줄 플래그
  2. 환경 변수 (AICLI_ 접두사)
  3. 설정 파일 (~/.aicli.yaml)
  4. 기본값

주요 설정 그룹:
  • api.*: API 서버 관련 설정
  • claude.*: Claude API 설정
  • docker.*: Docker 컨테이너 설정
  • workspace.*: 워크스페이스 기본 설정
  • logging.*: 로깅 설정`,
		Example: `  # 특정 설정 조회
  aicli config get claude.api_key
  
  # 설정 변경 (현재 세션만)
  aicli config set logging.level debug
  
  # 전역 설정 파일에 저장
  aicli config set claude.model claude-3-opus --global
  
  # 모든 설정 나열
  aicli config list`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	// 하위 명령 추가
	cmd.AddCommand(newConfigGetCmd())
	cmd.AddCommand(newConfigSetCmd())
	cmd.AddCommand(newConfigListCmd())
	cmd.AddCommand(newConfigResetCmd())
	cmd.AddCommand(newConfigBackupCmd())
	cmd.AddCommand(newConfigRestoreCmd())
	cmd.AddCommand(newConfigPathCmd())

	return cmd
}

// newConfigGetCmd는 설정 값을 조회하는 명령어입니다.
func newConfigGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get [key]",
		Short: "설정 값 조회",
		Long:  `지정된 설정 키의 값을 조회합니다.

키는 점(".")으로 구분된 계층 구조로 지정합니다.
예: api.endpoint, claude.model, logging.level

설정 값은 현재 적용된 최종 값을 표시합니다.
(명령줄 > 환경 변수 > 설정 파일 > 기본값 순서)`,
		Example: `  # API 엔드포인트 조회
  aicli config get api.endpoint
  
  # Claude 모델 설정 조회
  aicli config get claude.model
  
  # 로깅 레벨 조회
  aicli config get logging.level`,
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			// 설정 가능한 키 목록
			keys := []string{
				"api.endpoint",
				"api.timeout",
				"api.retry_count",
				"claude.api_key",
				"claude.model",
				"claude.max_tokens",
				"docker.registry",
				"docker.network",
				"workspace.default_dir",
				"logging.level",
				"logging.format",
			}
			return keys, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]

			// 설정 관리자 생성
			mgr, err := config.NewManager()
			if err != nil {
				return fmt.Errorf("설정 관리자 초기화 실패: %v", err)
			}

			// 설정 로드
			if err := mgr.Load(); err != nil {
				return fmt.Errorf("설정 로드 실패: %v", err)
			}

			// 설정 값 조회
			cfg := mgr.Get()
			value := getNestedValue(cfg, key)
			if value == nil {
				return fmt.Errorf("설정 키 '%s'를 찾을 수 없습니다", key)
			}

			fmt.Printf("%s: %v\n", key, value)
			return nil
		},
	}
}

// newConfigSetCmd는 설정 값을 변경하는 명령어입니다.
func newConfigSetCmd() *cobra.Command {
	var global bool

	cmd := &cobra.Command{
		Use:   "set [key] [value]",
		Short: "설정 값 변경",
		Long:  `지정된 설정 키의 값을 변경합니다.

기본적으로 설정은 현재 세션에만 적용됩니다.
--global 플래그를 사용하면 ~/.aicli.yaml 파일에 저장되어
영구적으로 적용됩니다.

값 타입:
  • 문자열: 큰따옴표 없이 입력
  • 숫자: 자동 타입 변환
  • 불린: true 또는 false
  • 배열: 쉼표로 구분 (예: "a,b,c")`,
		Example: `  # 로깅 레벨 변경 (현재 세션만)
  aicli config set logging.level debug
  
  # Claude 모델 전역 설정
  aicli config set claude.model claude-3-opus --global
  
  # API 타임아웃 설정 (초 단위)
  aicli config set api.timeout 30
  
  # Docker 레지스트리 설정
  aicli config set docker.registry myregistry.com
  
  # 환경 변수로도 설정 가능
  export AICLI_CLAUDE_MODEL=claude-3-opus`,
		Args:  cobra.ExactArgs(2),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				// 첫 번째 인자: 설정 키
				keys := []string{
					"api.endpoint",
					"api.timeout",
					"api.retry_count",
					"claude.api_key",
					"claude.model",
					"claude.max_tokens",
					"docker.registry",
					"docker.network",
					"workspace.default_dir",
					"logging.level",
					"logging.format",
				}
				return keys, cobra.ShellCompDirectiveNoFileComp
			} else if len(args) == 1 {
				// 두 번째 인자: 값 (키에 따라 다름)
				switch args[0] {
				case "logging.level":
					return []string{"debug", "info", "warn", "error"}, cobra.ShellCompDirectiveNoFileComp
				case "logging.format":
					return []string{"text", "json"}, cobra.ShellCompDirectiveNoFileComp
				case "claude.model":
					return []string{"claude-3-opus", "claude-3-sonnet", "claude-3-haiku"}, cobra.ShellCompDirectiveNoFileComp
				default:
					return nil, cobra.ShellCompDirectiveNoFileComp
				}
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			value := args[1]

			// 값 타입 추론
			var typedValue interface{}
			if value == "true" || value == "false" {
				typedValue = value == "true"
			} else if numValue := parseNumber(value); numValue != nil {
				typedValue = numValue
			} else {
				typedValue = value
			}

			// 설정 관리자 생성
			mgr, err := config.NewManager()
			if err != nil {
				return fmt.Errorf("설정 관리자 초기화 실패: %v", err)
			}

			// 현재 설정 로드
			if err := mgr.Load(); err != nil {
				return fmt.Errorf("설정 로드 실패: %v", err)
			}

			// 설정 값 업데이트
			cfg := mgr.Get()
			if err := setNestedValue(cfg, key, typedValue); err != nil {
				return fmt.Errorf("설정 값 업데이트 실패: %v", err)
			}

			// 설정 적용
			if err := mgr.Set(cfg); err != nil {
				return fmt.Errorf("설정 검증 실패: %v", err)
			}

			// 설정 파일에 저장
			if global {
				if err := mgr.Save(); err != nil {
					return fmt.Errorf("설정 파일 저장 실패: %v", err)
				}
				fmt.Printf("전역 설정 '%s'가 '%v'로 저장되었습니다.\n", key, typedValue)
			} else {
				fmt.Printf("설정 '%s'가 '%v'로 변경되었습니다. (현재 세션에만 적용)\n", key, typedValue)
				fmt.Println("전역으로 저장하려면 --global 플래그를 사용하세요.")
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&global, "global", "g", false, "전역 설정 파일에 저장")

	return cmd
}

// newConfigListCmd는 모든 설정을 나열하는 명령어입니다.
func newConfigListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "모든 설정 나열",
		Long:  `현재 적용된 모든 설정을 나열합니다.

설정은 계층 구조로 표시되며, 각 값의 출처
(명령줄, 환경 변수, 설정 파일, 기본값)를 확인할 수 있습니다.

설정 파일 경로도 함께 표시됩니다.`,
		Example: `  # 모든 설정 표시
  aicli config list
  
  # JSON 형식으로 출력
  aicli config list --output json
  
  # 특정 그룹만 필터링 (grep 사용)
  aicli config list | grep claude`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 설정 관리자 생성
			mgr, err := config.NewManager()
			if err != nil {
				return fmt.Errorf("설정 관리자 초기화 실패: %v", err)
			}

			// 설정 로드
			if err := mgr.Load(); err != nil {
				return fmt.Errorf("설정 로드 실패: %v", err)
			}

			// 설정을 YAML로 출력
			cfg := mgr.Get()
			data, err := yaml.Marshal(cfg)
			if err != nil {
				return fmt.Errorf("설정 직렬화 실패: %v", err)
			}

			fmt.Println("현재 설정:")
			fmt.Println("----------------------")
			fmt.Println(string(data))

			// 설정 파일 경로 표시
			fmt.Printf("\n설정 파일 경로: %s\n", mgr.GetConfigPath())

			return nil
		},
	}
}

// printSettings는 설정을 재귀적으로 출력합니다.
func printSettings(settings map[string]interface{}, prefix string) {
	for key, value := range settings {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		switch v := value.(type) {
		case map[string]interface{}:
			// 중첩된 설정
			printSettings(v, fullKey)
		default:
			// 일반 값
			fmt.Printf("%s: %v\n", fullKey, v)
		}
	}
}

// parseNumber는 문자열을 숫자로 파싱합니다.
func parseNumber(s string) interface{} {
	// 정수 파싱 시도
	if !strings.Contains(s, ".") {
		var i int
		if _, err := fmt.Sscanf(s, "%d", &i); err == nil {
			return i
		}
	}

	// 실수 파싱 시도
	var f float64
	if _, err := fmt.Sscanf(s, "%f", &f); err == nil {
		return f
	}

	return nil
}

// getNestedValue는 중첩된 구조체에서 점 표기법으로 값을 가져옵니다
func getNestedValue(cfg *config.Config, key string) interface{} {
	parts := strings.Split(key, ".")
	if len(parts) == 0 {
		return nil
	}

	switch parts[0] {
	case "claude":
		if len(parts) < 2 {
			return nil
		}
		switch parts[1] {
		case "api_key":
			return cfg.Claude.APIKey
		case "model":
			return cfg.Claude.Model
		case "temperature":
			return cfg.Claude.Temperature
		case "timeout":
			return cfg.Claude.Timeout
		}
	case "workspace":
		if len(parts) < 2 {
			return nil
		}
		switch parts[1] {
		case "default_path":
			return cfg.Workspace.DefaultPath
		case "auto_sync":
			return cfg.Workspace.AutoSync
		case "max_projects":
			return cfg.Workspace.MaxProjects
		}
	case "output":
		if len(parts) < 2 {
			return nil
		}
		switch parts[1] {
		case "format":
			return cfg.Output.Format
		case "color_mode":
			return cfg.Output.ColorMode
		case "width":
			return cfg.Output.Width
		}
	case "logging":
		if len(parts) < 2 {
			return nil
		}
		switch parts[1] {
		case "level":
			return cfg.Logging.Level
		case "file_path":
			return cfg.Logging.FilePath
		}
	}

	return nil
}

// setNestedValue는 중첩된 구조체에 점 표기법으로 값을 설정합니다
func setNestedValue(cfg *config.Config, key string, value interface{}) error {
	parts := strings.Split(key, ".")
	if len(parts) == 0 {
		return fmt.Errorf("유효하지 않은 키")
	}

	switch parts[0] {
	case "claude":
		if len(parts) < 2 {
			return fmt.Errorf("claude 하위 키가 필요합니다")
		}
		switch parts[1] {
		case "api_key":
			if s, ok := value.(string); ok {
				cfg.Claude.APIKey = s
			} else {
				return fmt.Errorf("api_key는 문자열이어야 합니다")
			}
		case "model":
			if s, ok := value.(string); ok {
				cfg.Claude.Model = s
			} else {
				return fmt.Errorf("model은 문자열이어야 합니다")
			}
		case "temperature":
			switch v := value.(type) {
			case float64:
				cfg.Claude.Temperature = v
			case int:
				cfg.Claude.Temperature = float64(v)
			default:
				return fmt.Errorf("temperature는 숫자여야 합니다")
			}
		case "timeout":
			switch v := value.(type) {
			case int:
				cfg.Claude.Timeout = v
			case float64:
				cfg.Claude.Timeout = int(v)
			default:
				return fmt.Errorf("timeout은 정수여야 합니다")
			}
		default:
			return fmt.Errorf("알 수 없는 claude 하위 키: %s", parts[1])
		}
	case "workspace":
		if len(parts) < 2 {
			return fmt.Errorf("workspace 하위 키가 필요합니다")
		}
		switch parts[1] {
		case "default_path":
			if s, ok := value.(string); ok {
				cfg.Workspace.DefaultPath = s
			} else {
				return fmt.Errorf("default_path는 문자열이어야 합니다")
			}
		case "auto_sync":
			if b, ok := value.(bool); ok {
				cfg.Workspace.AutoSync = b
			} else {
				return fmt.Errorf("auto_sync는 불린이어야 합니다")
			}
		case "max_projects":
			switch v := value.(type) {
			case int:
				cfg.Workspace.MaxProjects = v
			case float64:
				cfg.Workspace.MaxProjects = int(v)
			default:
				return fmt.Errorf("max_projects는 정수여야 합니다")
			}
		default:
			return fmt.Errorf("알 수 없는 workspace 하위 키: %s", parts[1])
		}
	case "output":
		if len(parts) < 2 {
			return fmt.Errorf("output 하위 키가 필요합니다")
		}
		switch parts[1] {
		case "format":
			if s, ok := value.(string); ok {
				cfg.Output.Format = s
			} else {
				return fmt.Errorf("format은 문자열이어야 합니다")
			}
		case "color_mode":
			if s, ok := value.(string); ok {
				cfg.Output.ColorMode = s
			} else {
				return fmt.Errorf("color_mode는 문자열이어야 합니다")
			}
		case "width":
			switch v := value.(type) {
			case int:
				cfg.Output.Width = v
			case float64:
				cfg.Output.Width = int(v)
			default:
				return fmt.Errorf("width는 정수여야 합니다")
			}
		default:
			return fmt.Errorf("알 수 없는 output 하위 키: %s", parts[1])
		}
	case "logging":
		if len(parts) < 2 {
			return fmt.Errorf("logging 하위 키가 필요합니다")
		}
		switch parts[1] {
		case "level":
			if s, ok := value.(string); ok {
				cfg.Logging.Level = s
			} else {
				return fmt.Errorf("level은 문자열이어야 합니다")
			}
		case "file_path":
			if s, ok := value.(string); ok {
				cfg.Logging.FilePath = s
			} else {
				return fmt.Errorf("file_path는 문자열이어야 합니다")
			}
		default:
			return fmt.Errorf("알 수 없는 logging 하위 키: %s", parts[1])
		}
	default:
		return fmt.Errorf("알 수 없는 최상위 키: %s", parts[0])
	}

	return nil
}

// newConfigResetCmd는 설정을 기본값으로 초기화하는 명령어입니다
func newConfigResetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reset",
		Short: "설정을 기본값으로 초기화",
		Long:  `모든 설정을 기본값으로 초기화합니다.
		
기존 설정은 자동으로 백업되며, 필요시 restore 명령으로 복구할 수 있습니다.`,
		Example: `  # 설정 초기화
  aicli config reset`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr, err := config.NewManager()
			if err != nil {
				return fmt.Errorf("설정 관리자 초기화 실패: %v", err)
			}

			if err := mgr.Reset(); err != nil {
				return fmt.Errorf("설정 초기화 실패: %v", err)
			}

			fmt.Println("설정이 기본값으로 초기화되었습니다.")
			fmt.Println("이전 설정은 백업되었습니다. 'aicli config restore'로 복구할 수 있습니다.")
			return nil
		},
	}
}

// newConfigBackupCmd는 현재 설정을 백업하는 명령어입니다
func newConfigBackupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "backup",
		Short: "현재 설정 백업",
		Long:  `현재 설정을 백업 파일로 저장합니다.
		
백업 파일은 설정 파일과 같은 디렉토리에 .backup 확장자로 저장됩니다.`,
		Example: `  # 설정 백업
  aicli config backup`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr, err := config.NewManager()
			if err != nil {
				return fmt.Errorf("설정 관리자 초기화 실패: %v", err)
			}

			if err := mgr.Backup(); err != nil {
				return fmt.Errorf("백업 생성 실패: %v", err)
			}

			fmt.Println("설정이 백업되었습니다.")
			return nil
		},
	}
}

// newConfigRestoreCmd는 백업에서 설정을 복구하는 명령어입니다
func newConfigRestoreCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "restore",
		Short: "백업에서 설정 복구",
		Long:  `백업 파일에서 설정을 복구합니다.
		
가장 최근의 백업 파일에서 설정을 복구합니다.`,
		Example: `  # 백업에서 설정 복구
  aicli config restore`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr, err := config.NewManager()
			if err != nil {
				return fmt.Errorf("설정 관리자 초기화 실패: %v", err)
			}

			if err := mgr.Restore(); err != nil {
				return fmt.Errorf("설정 복구 실패: %v", err)
			}

			fmt.Println("설정이 백업에서 복구되었습니다.")
			return nil
		},
	}
}

// newConfigPathCmd는 설정 파일 경로를 표시하는 명령어입니다
func newConfigPathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "설정 파일 경로 표시",
		Long:  `설정 파일의 전체 경로를 표시합니다.`,
		Example: `  # 설정 파일 경로 확인
  aicli config path`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr, err := config.NewManager()
			if err != nil {
				return fmt.Errorf("설정 관리자 초기화 실패: %v", err)
			}

			fmt.Println(mgr.GetConfigPath())
			return nil
		},
	}
}