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
각 프로젝트별 격리된 Docker 컨테이너에서 Claude CLI를 실행하고 관리합니다.`,
		Version: version.Version,
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
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.aicli.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().StringVarP(&output, "output", "o", "table", "output format (table|json|yaml)")

	// 플래그를 viper와 바인딩
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("output", rootCmd.PersistentFlags().Lookup("output"))

	// 하위 명령 추가
	rootCmd.AddCommand(commands.NewWorkspaceCmd())
	rootCmd.AddCommand(commands.NewTaskCmd())
	rootCmd.AddCommand(commands.NewLogsCmd())
	rootCmd.AddCommand(commands.NewConfigCmd())
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