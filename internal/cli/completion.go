package cli

import (
	"os"

	"github.com/spf13/cobra"
)

// createCompletionCmd는 자동완성 명령어를 생성합니다
func createCompletionCmd() *cobra.Command {
	completionCmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "쉘 자동완성 스크립트 생성",
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
				return cmd.Usage()
			}
		},
	}

	// Bash 자동완성 서브커맨드
	bashCmd := &cobra.Command{
		Use:   "bash",
		Short: "Bash 자동완성 스크립트 생성",
		Long: `Bash 쉘을 위한 자동완성 스크립트를 생성합니다.

설치 방법:

1. 자동완성 스크립트를 생성하고 저장:
   $ aicli completion bash > /etc/bash_completion.d/aicli

   또는 사용자별 설치:
   $ aicli completion bash > ~/.aicli-completion.bash

2. 현재 쉘 세션에 적용:
   $ source ~/.aicli-completion.bash

3. 영구적으로 적용하려면 ~/.bashrc에 추가:
   $ echo "source ~/.aicli-completion.bash" >> ~/.bashrc`,
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Root().GenBashCompletion(os.Stdout)
		},
	}

	// Zsh 자동완성 서브커맨드
	zshCmd := &cobra.Command{
		Use:   "zsh",
		Short: "Zsh 자동완성 스크립트 생성",
		Long: `Zsh 쉘을 위한 자동완성 스크립트를 생성합니다.

설치 방법:

1. 자동완성 스크립트를 생성하고 저장:
   $ aicli completion zsh > "${fpath[1]}/_aicli"

   또는 사용자별 설치:
   $ aicli completion zsh > ~/.aicli-completion.zsh

2. ~/.zshrc에 추가:
   $ echo "source ~/.aicli-completion.zsh" >> ~/.zshrc

3. 새로운 쉘 세션을 시작하거나 설정을 다시 로드:
   $ source ~/.zshrc

주의: fpath가 설정되어 있는지 확인하세요.
   $ echo $fpath`,
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Root().GenZshCompletion(os.Stdout)
		},
	}

	// Fish 자동완성 서브커맨드
	fishCmd := &cobra.Command{
		Use:   "fish",
		Short: "Fish 자동완성 스크립트 생성",
		Long: `Fish 쉘을 위한 자동완성 스크립트를 생성합니다.

설치 방법:

1. 자동완성 스크립트를 생성하고 저장:
   $ aicli completion fish > ~/.config/fish/completions/aicli.fish

2. 새로운 쉘 세션을 시작하면 자동으로 적용됩니다.`,
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Root().GenFishCompletion(os.Stdout, true)
		},
	}

	// PowerShell 자동완성 서브커맨드
	powershellCmd := &cobra.Command{
		Use:   "powershell",
		Short: "PowerShell 자동완성 스크립트 생성",
		Long: `PowerShell을 위한 자동완성 스크립트를 생성합니다.

설치 방법:

1. 자동완성 스크립트를 생성하고 저장:
   PS> aicli completion powershell > aicli.ps1

2. PowerShell 프로필에 추가:
   PS> echo ". ./aicli.ps1" >> $PROFILE

3. 새로운 PowerShell 세션을 시작하면 자동으로 적용됩니다.`,
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
		},
	}

	// 서브커맨드 추가
	completionCmd.AddCommand(bashCmd, zshCmd, fishCmd, powershellCmd)
	
	return completionCmd
}

// addCompletionCmd는 자동완성 명령어를 추가합니다
func addCompletionCmd() {
	// root 커맨드에 completion 추가
	rootCmd.AddCommand(createCompletionCmd())
}

// 동적 자동완성을 위한 함수들

// RegisterWorkspaceCompletion은 workspace 관련 명령어에 동적 자동완성을 등록합니다
func RegisterWorkspaceCompletion(cmd *cobra.Command) {
	// workspace 이름 자동완성
	cmd.RegisterFlagCompletionFunc("name", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		// TODO: 실제 워크스페이스 목록을 가져오는 로직 구현
		// 현재는 예시 데이터 반환
		workspaces := []string{
			"project-alpha",
			"project-beta", 
			"project-gamma",
			"development",
			"staging",
			"production",
		}
		return workspaces, cobra.ShellCompDirectiveNoFileComp
	})
}

// RegisterTaskCompletion은 task 관련 명령어에 동적 자동완성을 등록합니다
func RegisterTaskCompletion(cmd *cobra.Command) {
	// task ID 자동완성
	cmd.RegisterFlagCompletionFunc("task-id", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		// TODO: 실제 태스크 목록을 가져오는 로직 구현
		// 현재는 예시 데이터 반환
		tasks := []string{
			"task-001",
			"task-002",
			"task-003",
			"bug-fix-101",
			"feature-201",
		}
		return tasks, cobra.ShellCompDirectiveNoFileComp
	})

	// task 상태 자동완성
	cmd.RegisterFlagCompletionFunc("status", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		statuses := []string{
			"pending",
			"running",
			"completed",
			"failed",
			"cancelled",
		}
		return statuses, cobra.ShellCompDirectiveNoFileComp
	})
}

// RegisterOutputFormatCompletion은 출력 형식 플래그에 대한 자동완성을 등록합니다
func RegisterOutputFormatCompletion(cmd *cobra.Command) {
	cmd.RegisterFlagCompletionFunc("output", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		formats := []string{
			"table",
			"json",
			"yaml",
			"csv",
		}
		return formats, cobra.ShellCompDirectiveNoFileComp
	})
}

// RegisterConfigKeyCompletion은 config 명령어의 키 자동완성을 등록합니다  
func RegisterConfigKeyCompletion(cmd *cobra.Command) {
	cmd.RegisterFlagCompletionFunc("key", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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
	})
}