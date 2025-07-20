package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// NewConfigCmd는 config 관련 명령어를 생성합니다.
func NewConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "설정 관리",
		Long:  `AICLI의 설정을 조회하고 변경합니다.`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	// 하위 명령 추가
	cmd.AddCommand(newConfigGetCmd())
	cmd.AddCommand(newConfigSetCmd())
	cmd.AddCommand(newConfigListCmd())

	return cmd
}

// newConfigGetCmd는 설정 값을 조회하는 명령어입니다.
func newConfigGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get [key]",
		Short: "설정 값 조회",
		Long:  `지정된 설정 키의 값을 조회합니다.`,
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

			// Viper에서 설정 값 조회
			value := viper.Get(key)
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
		Long:  `지정된 설정 키의 값을 변경합니다.`,
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

			// Viper에 설정 값 저장
			viper.Set(key, typedValue)

			// 설정 파일에 저장
			if global {
				if err := viper.WriteConfig(); err != nil {
					// 설정 파일이 없으면 생성
					if err := viper.SafeWriteConfig(); err != nil {
						return fmt.Errorf("설정 파일 저장 실패: %v", err)
					}
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
		Long:  `현재 적용된 모든 설정을 나열합니다.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 모든 설정 키 가져오기
			settings := viper.AllSettings()

			if len(settings) == 0 {
				fmt.Println("설정이 없습니다.")
				return nil
			}

			// 설정 출력
			fmt.Println("Current configuration:")
			fmt.Println("----------------------")
			printSettings(settings, "")

			// 설정 파일 경로 표시
			if configFile := viper.ConfigFileUsed(); configFile != "" {
				fmt.Printf("\nConfig file: %s\n", configFile)
			}

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