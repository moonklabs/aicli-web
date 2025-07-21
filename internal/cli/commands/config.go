package commands

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"aicli-web/internal/config"
	"aicli-web/internal/cli/output"
	"gopkg.in/yaml.v3"
)

// NewConfigCommand는 Viper 기반 config 관련 명령어를 생성합니다
func NewConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage aicli configuration",
		Long:  `Get, set, and list configuration options for aicli.
		
Configuration priority order:
  1. Command-line flags
  2. Environment variables (AICLI_ prefix)
  3. Configuration file (~/.aicli/config.yaml)
  4. Default values

Key configuration groups:
  • claude.*: Claude API settings
  • workspace.*: Workspace defaults
  • output.*: Output formatting
  • logging.*: Logging settings`,
		Example: `  # Get specific configuration
  aicli config get claude.model
  
  # Set configuration value
  aicli config set logging.level debug
  
  # List all configurations
  aicli config list
  
  # Validate configuration
  aicli config validate`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	// 하위 명령 추가
	cmd.AddCommand(
		newConfigGetCommand(),
		newConfigSetCommand(),
		newConfigListCommand(),
		newConfigValidateCommand(),
		newConfigResetCommand(),
	)

	return cmd
}

// newConfigGetCommand는 설정 값을 조회하는 명령어입니다
func newConfigGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: "Get configuration value",
		Long:  `Get the value of a specific configuration key.
		
The value shown is the final resolved value considering all sources
in priority order (flags > env > file > default).`,
		Example: `  # Get Claude model configuration
  aicli config get claude.model
  
  # Get workspace default path
  aicli config get workspace.default_path`,
		Args: cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			// 자동완성을 위한 설정 키 목록
			keys := []string{
				"claude.api_key",
				"claude.model", 
				"claude.temperature",
				"claude.timeout",
				"workspace.default_path",
				"workspace.auto_sync",
				"workspace.max_projects",
				"output.format",
				"output.color_mode",
				"output.width",
				"logging.level",
				"logging.file_path",
			}
			return keys, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cm, err := config.GetManager()
			if err != nil {
				return err
			}

			key := args[0]
			value := cm.Get(key)

			if value == nil {
				return fmt.Errorf("configuration key '%s' not found", key)
			}

			// 값과 소스 출력
			source := cm.GetValueSource(key)
			fmt.Printf("%s = %v (from %s)\n", key, value, source)
			
			return nil
		},
	}
}

// newConfigSetCommand는 설정 값을 변경하는 명령어입니다
func newConfigSetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set configuration value",
		Long:  `Set a configuration value.
		
The value will be saved to the configuration file and persist across sessions.
Values set via environment variables or flags cannot be overridden.`,
		Example: `  # Set logging level
  aicli config set logging.level debug
  
  # Set Claude model
  aicli config set claude.model claude-3-opus
  
  # Set output width
  aicli config set output.width 150`,
		Args: cobra.ExactArgs(2),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				// 첫 번째 인자: 설정 키
				keys := []string{
					"claude.api_key",
					"claude.model",
					"claude.temperature", 
					"claude.timeout",
					"workspace.default_path",
					"workspace.auto_sync",
					"workspace.max_projects",
					"output.format",
					"output.color_mode",
					"output.width",
					"logging.level",
					"logging.file_path",
				}
				return keys, cobra.ShellCompDirectiveNoFileComp
			} else if len(args) == 1 {
				// 두 번째 인자: 값 (키에 따라 다름)
				switch args[0] {
				case "logging.level":
					return []string{"debug", "info", "warn", "error"}, cobra.ShellCompDirectiveNoFileComp
				case "output.format":
					return []string{"table", "json", "yaml"}, cobra.ShellCompDirectiveNoFileComp
				case "output.color_mode":
					return []string{"auto", "always", "never"}, cobra.ShellCompDirectiveNoFileComp
				case "claude.model":
					return []string{"claude-3-opus", "claude-3-sonnet", "claude-3-haiku"}, cobra.ShellCompDirectiveNoFileComp
				case "workspace.auto_sync":
					return []string{"true", "false"}, cobra.ShellCompDirectiveNoFileComp
				}
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cm, err := config.GetManager()
			if err != nil {
				return err
			}

			key := args[0]
			value := args[1]

			// 타입 변환
			typedValue, err := cm.ConvertValue(key, value)
			if err != nil {
				return fmt.Errorf("invalid value for %s: %w", key, err)
			}

			if err := cm.Set(key, typedValue); err != nil {
				return err
			}

			fmt.Printf("Successfully set %s = %v\n", key, typedValue)
			return nil
		},
	}
}

// newConfigListCommand는 모든 설정을 나열하는 명령어입니다
func newConfigListCommand() *cobra.Command {
	var showSource bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all configuration settings",
		Long:  `List all current configuration settings.
		
Shows the hierarchical structure of all settings with their current values.
Use --source to see where each value comes from.`,
		Example: `  # List all settings
  aicli config list
  
  # List with value sources
  aicli config list --source
  
  # Filter specific group (using grep)
  aicli config list | grep claude`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cm, err := config.GetManager()
			if err != nil {
				return err
			}

			// 전체 설정 가져오기
			allSettings := cm.AllSettings()

			// 출력 포맷터 생성
			formatter := output.DefaultFormatterManager()
			
			if showSource {
				// 소스 정보와 함께 테이블 형식으로 출력
				configs := flattenSettingsWithSource(cm, allSettings, "")
				formatter.SetHeaders([]string{"key", "value", "source"})
				return formatter.Print(configs)
			} else {
				// 일반 출력 (테이블/JSON/YAML)
				return formatter.Print(allSettings)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&showSource, "source", false, "Show the source of each value")

	return cmd
}

// printSettingsWithSource는 설정을 소스 정보와 함께 출력합니다
func printSettingsWithSource(cm *config.ConfigManager, settings map[string]interface{}, prefix string) {
	// 키를 정렬하여 일관된 출력 보장
	keys := make([]string, 0, len(settings))
	for k := range settings {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := settings[key]
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		switch v := value.(type) {
		case map[string]interface{}:
			// 중첩된 설정
			printSettingsWithSource(cm, v, fullKey)
		default:
			// 일반 값
			source := cm.GetValueSource(fullKey)
			fmt.Printf("%s = %v (from %s)\n", fullKey, v, source)
		}
	}
}

// newConfigValidateCommand는 설정을 검증하는 명령어입니다
func newConfigValidateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate current configuration",
		Long:  `Validate the current configuration for correctness.
		
Checks that all required fields are present and all values are valid.`,
		Example: `  # Validate configuration
  aicli config validate`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cm, err := config.GetManager()
			if err != nil {
				return err
			}

			// 전체 설정을 Config 구조체로 가져오기
			cfg, err := cm.GetConfig()
			if err != nil {
				return fmt.Errorf("failed to get configuration: %w", err)
			}

			// 필수 필드 검증
			var errors []string

			// Claude API 키 검증
			if cfg.Claude.APIKey == "" {
				errors = append(errors, "claude.api_key is required")
			}

			// 모델 검증
			validModels := []string{"claude-3-opus", "claude-3-sonnet", "claude-3-haiku"}
			modelValid := false
			for _, valid := range validModels {
				if cfg.Claude.Model == valid {
					modelValid = true
					break
				}
			}
			if !modelValid {
				errors = append(errors, fmt.Sprintf("claude.model must be one of: %v", validModels))
			}

			// Temperature 범위 검증
			if cfg.Claude.Temperature < 0 || cfg.Claude.Temperature > 1 {
				errors = append(errors, "claude.temperature must be between 0 and 1")
			}

			// Timeout 검증
			if cfg.Claude.Timeout < 1 {
				errors = append(errors, "claude.timeout must be at least 1 second")
			}

			// MaxProjects 검증
			if cfg.Workspace.MaxProjects < 1 || cfg.Workspace.MaxProjects > 100 {
				errors = append(errors, "workspace.max_projects must be between 1 and 100")
			}

			// Output format 검증
			validFormats := []string{"table", "json", "yaml"}
			formatValid := false
			for _, valid := range validFormats {
				if cfg.Output.Format == valid {
					formatValid = true
					break
				}
			}
			if !formatValid {
				errors = append(errors, fmt.Sprintf("output.format must be one of: %v", validFormats))
			}

			// Color mode 검증
			validColorModes := []string{"auto", "always", "never"}
			colorModeValid := false
			for _, valid := range validColorModes {
				if cfg.Output.ColorMode == valid {
					colorModeValid = true
					break
				}
			}
			if !colorModeValid {
				errors = append(errors, fmt.Sprintf("output.color_mode must be one of: %v", validColorModes))
			}

			// Width 검증
			if cfg.Output.Width < 40 {
				errors = append(errors, "output.width must be at least 40")
			}

			// Log level 검증
			validLogLevels := []string{"debug", "info", "warn", "error"}
			logLevelValid := false
			for _, valid := range validLogLevels {
				if cfg.Logging.Level == valid {
					logLevelValid = true
					break
				}
			}
			if !logLevelValid {
				errors = append(errors, fmt.Sprintf("logging.level must be one of: %v", validLogLevels))
			}

			// 결과 출력
			if len(errors) > 0 {
				fmt.Println("Configuration validation failed:")
				for _, err := range errors {
					fmt.Printf("  - %s\n", err)
				}
				return fmt.Errorf("validation failed with %d errors", len(errors))
			}

			fmt.Println("Configuration is valid ✓")
			return nil
		},
	}
}

// newConfigResetCommand는 설정을 기본값으로 초기화하는 명령어입니다
func newConfigResetCommand() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "reset",
		Short: "Reset configuration to defaults",
		Long:  `Reset all configuration to default values.
		
This will overwrite your current configuration file with defaults.
Use --force to skip confirmation.`,
		Example: `  # Reset configuration (with confirmation)
  aicli config reset
  
  # Reset without confirmation
  aicli config reset --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !force {
				fmt.Print("This will reset all configuration to defaults. Continue? [y/N]: ")
				var response string
				fmt.Scanln(&response)
				if !strings.HasPrefix(strings.ToLower(response), "y") {
					fmt.Println("Reset cancelled.")
					return nil
				}
			}

			cm, err := config.GetManager()
			if err != nil {
				return err
			}

			if err := cm.Reset(); err != nil {
				return fmt.Errorf("failed to reset configuration: %w", err)
			}

			fmt.Println("Configuration reset to defaults.")
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Skip confirmation")

	return cmd
}

// flattenSettingsWithSource는 중첩된 설정을 평면화하고 소스 정보를 추가합니다
func flattenSettingsWithSource(cm *config.Manager, settings map[string]interface{}, prefix string) []map[string]interface{} {
	var result []map[string]interface{}
	
	// 키를 정렬하여 일관된 출력 순서 보장
	keys := make([]string, 0, len(settings))
	for k := range settings {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	
	for _, key := range keys {
		value := settings[key]
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}
		
		switch v := value.(type) {
		case map[string]interface{}:
			// 중첩된 맵인 경우 재귀적으로 처리
			result = append(result, flattenSettingsWithSource(cm, v, fullKey)...)
		default:
			// 값과 소스 정보 추가
			source := "default"
			if cm.IsSet(fullKey) {
				// 실제 소스 확인 (환경변수, 설정파일 등)
				if cm.GetEnv(fullKey) != "" {
					source = "env"
				} else if cm.IsFromConfigFile(fullKey) {
					source = "config"
				}
			}
			
			result = append(result, map[string]interface{}{
				"key":    fullKey,
				"value":  fmt.Sprintf("%v", value),
				"source": source,
			})
		}
	}
	
	return result
}