package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewTaskCmd는 task 관련 명령어를 생성합니다.
func NewTaskCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "task",
		Aliases: []string{"t"},
		Short:   "Claude 태스크 관리",
		Long:    `워크스페이스에서 실행 중인 Claude 태스크를 관리합니다.`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	// 하위 명령 추가
	cmd.AddCommand(newTaskCreateCmd())
	cmd.AddCommand(newTaskListCmd())
	cmd.AddCommand(newTaskStatusCmd())
	cmd.AddCommand(newTaskCancelCmd())

	return cmd
}

// newTaskCreateCmd는 새 태스크를 생성하는 명령어입니다.
func newTaskCreateCmd() *cobra.Command {
	var (
		workspace   string
		command     string
		interactive bool
		detach      bool
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "새 태스크 생성",
		Long:  `지정된 워크스페이스에서 새로운 Claude 태스크를 생성하고 실행합니다.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 필수 플래그 검증
			if workspace == "" {
				return fmt.Errorf("워크스페이스는 필수입니다 (--workspace)")
			}
			if command == "" && !interactive {
				return fmt.Errorf("명령어가 필요하거나 대화형 모드를 사용하세요 (--command 또는 --interactive)")
			}

			// TODO: API 클라이언트를 통해 실제 태스크 생성
			fmt.Printf("Creating task in workspace '%s'...\n", workspace)
			if interactive {
				fmt.Println("Starting interactive Claude session...")
			} else {
				fmt.Printf("Executing command: %s\n", command)
			}
			
			if detach {
				fmt.Println("Task started in background. Task ID: task-12345")
			} else {
				fmt.Println("Task completed successfully!")
			}
			
			return nil
		},
	}

	// 플래그 정의
	cmd.Flags().StringVarP(&workspace, "workspace", "w", "", "워크스페이스 이름 (필수)")
	cmd.Flags().StringVarP(&command, "command", "c", "", "실행할 명령어")
	cmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "대화형 모드로 실행")
	cmd.Flags().BoolVarP(&detach, "detach", "d", false, "백그라운드에서 실행")

	// 필수 플래그 표시
	cmd.MarkFlagRequired("workspace")

	return cmd
}

// newTaskListCmd는 태스크 목록을 조회하는 명령어입니다.
func newTaskListCmd() *cobra.Command {
	var (
		workspace string
		all       bool
		status    string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "태스크 목록 조회",
		Long:  `실행 중이거나 완료된 태스크 목록을 조회합니다.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: API 클라이언트를 통해 실제 태스크 목록 조회
			fmt.Println("Tasks:")
			if workspace != "" {
				fmt.Printf("Filtering by workspace: %s\n", workspace)
			}
			if status != "" {
				fmt.Printf("Filtering by status: %s\n", status)
			}
			fmt.Println("- No tasks found")
			return nil
		},
	}

	// 플래그 정의
	cmd.Flags().StringVarP(&workspace, "workspace", "w", "", "특정 워크스페이스의 태스크만 조회")
	cmd.Flags().BoolVarP(&all, "all", "a", false, "모든 상태의 태스크 조회")
	cmd.Flags().StringVarP(&status, "status", "s", "", "상태별 필터링 (running|completed|failed)")

	return cmd
}

// newTaskStatusCmd는 태스크 상태를 조회하는 명령어입니다.
func newTaskStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status [task-id]",
		Short: "태스크 상태 조회",
		Long:  `지정된 태스크의 상세 상태를 조회합니다.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]

			// TODO: API 클라이언트를 통해 실제 태스크 상태 조회
			fmt.Printf("Task ID: %s\n", taskID)
			fmt.Println("Status: Not Found")
			return nil
		},
	}
}

// newTaskCancelCmd는 실행 중인 태스크를 취소하는 명령어입니다.
func newTaskCancelCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "cancel [task-id]",
		Short: "태스크 취소",
		Long:  `실행 중인 태스크를 취소합니다.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]

			// TODO: API 클라이언트를 통해 실제 태스크 취소
			fmt.Printf("Cancelling task %s...\n", taskID)
			if force {
				fmt.Println("Force cancelling task...")
			}
			fmt.Println("Task cancelled successfully!")
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "강제 종료")

	return cmd
}