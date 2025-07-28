package cli

import (
	"github.com/spf13/cobra"
)

// createCompletionCmdWithSubcommands creates completion command with subcommands
func createCompletionCmdWithSubcommands() *cobra.Command {
	// 기존 함수를 사용하여 기본 completion 명령어 생성
	cmd := newCompletionCmd()

	// 각 셸별 서브커맨드 추가
	cmd.AddCommand(&cobra.Command{
		Use:   "bash",
		Short: "Generate bash completion script",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Root().GenBashCompletion(cmd.OutOrStdout())
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "zsh",
		Short: "Generate zsh completion script",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Root().GenZshCompletion(cmd.OutOrStdout())
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "fish",
		Short: "Generate fish completion script",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Root().GenFishCompletion(cmd.OutOrStdout(), true)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "powershell",
		Short: "Generate powershell completion script",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Root().GenPowerShellCompletionWithDesc(cmd.OutOrStdout())
		},
	})

	return cmd
}

// completionCmd represents the completion command
var completionCmd = createCompletionCmdWithSubcommands()

// addCompletionCmd adds the completion command to the root command
func addCompletionCmd() {
	rootCmd.AddCommand(completionCmd)
}
