package cli

import (
	"os"
	"fmt"

	"github.com/spf13/cobra"
)

// newCompletionCmd는 테스트에서 사용하는 completion 명령어를 생성합니다.
func newCompletionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "셸 자동 완성 스크립트 생성",
		Long: `지정된 셸에 대한 자동완성 스크립트를 생성합니다.

지원하는 쉘:
  - Bash
  - Zsh
  - Fish
  - PowerShell

사용 예시:
  $ aicli completion bash > /etc/bash_completion.d/aicli
  $ source /etc/bash_completion.d/aicli

자세한 사용법은 각 서브커맨드를 참조하세요.`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.ExactValidArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				return cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				return cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
			default:
				return fmt.Errorf("지원하지 않는 셸: %s", args[0])
			}
		},
	}
	
	return cmd
}

