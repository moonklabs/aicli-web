package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Execute는 CLI를 실행합니다.
func Execute() error {
	// 자동 완성 명령어 추가
	rootCmd.AddCommand(newCompletionCmd())
	
	return rootCmd.Execute()
}

// newCompletionCmd는 셸 자동 완성 스크립트를 생성하는 명령어입니다.
func newCompletionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "셸 자동 완성 스크립트 생성",
		Long: `지정된 셸에 대한 자동 완성 스크립트를 생성합니다.

### Bash:

  $ source <(aicli completion bash)

  # 영구적으로 적용하려면:
  $ aicli completion bash > /etc/bash_completion.d/aicli

### Zsh:

  $ source <(aicli completion zsh)

  # 영구적으로 적용하려면:
  $ aicli completion zsh > "${fpath[1]}/_aicli"

### Fish:

  $ aicli completion fish | source

  # 영구적으로 적용하려면:
  $ aicli completion fish > ~/.config/fish/completions/aicli.fish

### PowerShell:

  PS> aicli completion powershell | Out-String | Invoke-Expression

  # 영구적으로 적용하려면, 프로필에 추가:
  PS> aicli completion powershell >> $PROFILE
`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.ExactValidArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return rootCmd.GenBashCompletion(os.Stdout)
			case "zsh":
				return rootCmd.GenZshCompletion(os.Stdout)
			case "fish":
				return rootCmd.GenFishCompletion(os.Stdout, true)
			case "powershell":
				return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
			default:
				return fmt.Errorf("지원하지 않는 셸: %s", args[0])
			}
		},
	}

	return cmd
}