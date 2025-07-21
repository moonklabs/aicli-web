package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/drumcap/aicli-web/internal/cli/commands"
	"github.com/drumcap/aicli-web/pkg/version"
)

var (
	// 전역 플래그
	cfgFile string
	verbose bool
	output  string
	
	// rootCmd는 CLI의 기본 명령어를 나타냅니다
	rootCmd = &cobra.Command{
		Use:   "aicli",
		Short: "AI-powered code management CLI",
		Long: `AICode Manager(aicli)는 Claude CLI를 웹 플랫폼으로 관리하는 시스템입니다.
각 프로젝트별 격리된 Docker 컨테이너에서 Claude CLI를 실행하고 관리합니다.

주요 기능:
  • 프로젝트별 격리된 워크스페이스 관리
  • Claude CLI 태스크 실행 및 모니터링
  • 실시간 로그 스트리밍
  • Git 워크플로우 통합
  • 멀티 프로젝트 병렬 작업 지원

시작하기:
  먼저 워크스페이스를 생성한 다음, 해당 워크스페이스에서 Claude 태스크를 실행합니다.

  $ aicli workspace create --name myproject --path /path/to/project
  $ aicli task create --workspace myproject --command "implement feature X"

더 자세한 정보는 'aicli help [command]'를 사용하세요.`,
		Version: version.Version,
		Example: `  # 워크스페이스 생성
  aicli workspace create --name myproject --path ~/projects/myapp

  # 태스크 실행
  aicli task create --workspace myproject --command "add login feature"

  # 로그 확인
  aicli logs --workspace myproject --follow

  # 설정 변경
  aicli config set claude.model claude-3-opus`,
	}
)

// Execute는 모든 하위 명령을 rootCmd에 추가하고 설정 플래그를 적절히 설정합니다.
// 이는 main.main()에서만 한 번 호출됩니다.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// 전역 플래그 정의
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "설정 파일 경로 (기본값: $HOME/.aicli.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "상세 출력 모드 활성화")
	rootCmd.PersistentFlags().StringVarP(&output, "output", "o", "table", "출력 형식 (table|json|yaml)")

	// 플래그를 viper와 바인딩
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("output", rootCmd.PersistentFlags().Lookup("output"))

	// 출력 형식 자동완성
	rootCmd.RegisterFlagCompletionFunc("output", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		formats := []string{
			"table",
			"json",
			"yaml",
		}
		return formats, cobra.ShellCompDirectiveNoFileComp
	})

	// 하위 명령 추가
	rootCmd.AddCommand(commands.NewWorkspaceCmd())
	rootCmd.AddCommand(commands.NewTaskCmd())
	rootCmd.AddCommand(commands.NewLogsCmd())
	rootCmd.AddCommand(commands.NewConfigCmd())
	rootCmd.AddCommand(commands.NewVersionCmd())
	rootCmd.AddCommand(commands.NewDBCmd())
	
	// 자동 완성 명령어 추가
	addCompletionCmd()
}

// initConfig는 설정 파일을 읽습니다.
func initConfig() {
	if cfgFile != "" {
		// 명령줄에서 지정한 설정 파일 사용
		viper.SetConfigFile(cfgFile)
	} else {
		// 홈 디렉토리에서 .aicli.yaml 파일 찾기
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// 홈 디렉토리에서 설정 파일 검색
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".aicli")
	}

	// 환경 변수 자동 읽기
	viper.AutomaticEnv()
	viper.SetEnvPrefix("AICLI")

	// 설정 파일이 있으면 읽기
	if err := viper.ReadInConfig(); err == nil {
		if verbose {
			fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
		}
	}
}