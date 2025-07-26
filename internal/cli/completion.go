package cli

import (
	"os"

	"github.com/spf13/cobra"
)

// completionCmd represents the completion command
var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate completion script",
	Long: `To load completions:

Bash:
  $ source <(aicli completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ aicli completion bash > /etc/bash_completion.d/aicli
  # macOS:
  $ aicli completion bash > /usr/local/etc/bash_completion.d/aicli

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:

  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ aicli completion zsh > "${fpath[1]}/_aicli"

  # You will need to start a new shell for this setup to take effect.

Fish:
  $ aicli completion fish | source

  # To load completions for each session, execute once:
  $ aicli completion fish > ~/.config/fish/completions/aicli.fish

PowerShell:
  PS> aicli completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> aicli completion powershell > aicli.ps1
  # and source this file from your PowerShell profile.
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.ExactValidArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "bash":
			cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
		}
	},
}

// addCompletionCmd adds the completion command to the root command
func addCompletionCmd() {
	rootCmd.AddCommand(completionCmd)
}