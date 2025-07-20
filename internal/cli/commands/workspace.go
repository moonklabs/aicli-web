package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewWorkspaceCmd는 workspace 관련 명령어를 생성합니다.
func NewWorkspaceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "workspace",
		Aliases: []string{"ws"},
		Short:   "워크스페이스 관리",
		Long:    `Claude CLI 실행을 위한 격리된 워크스페이스를 관리합니다.`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	// 하위 명령 추가
	cmd.AddCommand(newWorkspaceListCmd())
	cmd.AddCommand(newWorkspaceCreateCmd())
	cmd.AddCommand(newWorkspaceDeleteCmd())
	cmd.AddCommand(newWorkspaceInfoCmd())

	return cmd
}

// newWorkspaceListCmd는 워크스페이스 목록을 조회하는 명령어입니다.
func newWorkspaceListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "워크스페이스 목록 조회",
		Long:  `현재 생성된 모든 워크스페이스의 목록을 조회합니다.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: API 클라이언트를 통해 실제 워크스페이스 목록 조회
			fmt.Println("Available workspaces:")
			fmt.Println("- No workspaces found")
			return nil
		},
	}
}

// newWorkspaceCreateCmd는 새 워크스페이스를 생성하는 명령어입니다.
func newWorkspaceCreateCmd() *cobra.Command {
	var (
		name        string
		projectPath string
		claudeKey   string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "새 워크스페이스 생성",
		Long:  `새로운 Claude CLI 워크스페이스를 생성합니다.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 필수 플래그 검증
			if name == "" {
				return fmt.Errorf("워크스페이스 이름은 필수입니다 (--name)")
			}
			if projectPath == "" {
				return fmt.Errorf("프로젝트 경로는 필수입니다 (--path)")
			}

			// TODO: API 클라이언트를 통해 실제 워크스페이스 생성
			fmt.Printf("Creating workspace '%s' for project at '%s'...\n", name, projectPath)
			fmt.Println("Workspace created successfully!")
			return nil
		},
	}

	// 플래그 정의
	cmd.Flags().StringVarP(&name, "name", "n", "", "워크스페이스 이름 (필수)")
	cmd.Flags().StringVarP(&projectPath, "path", "p", "", "프로젝트 경로 (필수)")
	cmd.Flags().StringVarP(&claudeKey, "claude-key", "k", "", "Claude API 키 (선택)")

	// 필수 플래그 표시
	cmd.MarkFlagRequired("name")
	cmd.MarkFlagRequired("path")

	return cmd
}

// newWorkspaceDeleteCmd는 워크스페이스를 삭제하는 명령어입니다.
func newWorkspaceDeleteCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete [workspace-name]",
		Short: "워크스페이스 삭제",
		Long:  `지정된 워크스페이스를 삭제합니다.`,
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			// TODO: 실제 워크스페이스 목록을 가져오는 로직 구현
			workspaces := []string{
				"project-alpha",
				"project-beta",
				"project-gamma",
				"development",
				"staging",
				"production",
			}
			return workspaces, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			workspaceName := args[0]

			// 확인 프롬프트
			if !force {
				fmt.Printf("정말 워크스페이스 '%s'를 삭제하시겠습니까? (y/N): ", workspaceName)
				var response string
				fmt.Scanln(&response)
				if response != "y" && response != "Y" {
					fmt.Println("삭제가 취소되었습니다.")
					return nil
				}
			}

			// TODO: API 클라이언트를 통해 실제 워크스페이스 삭제
			fmt.Printf("Deleting workspace '%s'...\n", workspaceName)
			fmt.Println("Workspace deleted successfully!")
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "확인 없이 강제 삭제")

	return cmd
}

// newWorkspaceInfoCmd는 워크스페이스 정보를 조회하는 명령어입니다.
func newWorkspaceInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info [workspace-name]",
		Short: "워크스페이스 정보 조회",
		Long:  `지정된 워크스페이스의 상세 정보를 조회합니다.`,
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			// TODO: 실제 워크스페이스 목록을 가져오는 로직 구현
			workspaces := []string{
				"project-alpha",
				"project-beta",
				"project-gamma",
				"development",
				"staging",
				"production",
			}
			return workspaces, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			workspaceName := args[0]

			// TODO: API 클라이언트를 통해 실제 워크스페이스 정보 조회
			fmt.Printf("Workspace: %s\n", workspaceName)
			fmt.Println("Status: Not Found")
			return nil
		},
	}
}