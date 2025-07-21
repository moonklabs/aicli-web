package commands

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"aicli-web/internal/cli/output"
)

// NewTaskCmd는 task 관련 명령어를 생성합니다.
func NewTaskCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "task",
		Aliases: []string{"t"},
		Short:   "Claude 태스크 관리",
		Long:    `워크스페이스에서 실행 중인 Claude 태스크를 관리합니다.

태스크는 Claude CLI에 전달되는 작업 단위로, 코드 생성, 버그 수정,
리팩토링 등의 작업을 수행합니다. 각 태스크는 고유한 ID를 가지며,
실시간으로 진행 상황을 모니터링할 수 있습니다.

태스크 유형:
  • 명령형: 특정 명령을 실행하고 종료
  • 대화형: Claude와 상호작용하며 작업 수행
  • 백그라운드: 장기 실행 작업을 백그라운드에서 처리`,
		Example: `  # 명령형 태스크 실행
  aicli task create --workspace myproject --command "add user authentication"
  
  # 대화형 모드로 실행
  aicli task create --workspace myproject --interactive
  
  # 백그라운드에서 실행
  aicli task create --workspace myproject --command "refactor database layer" --detach
  
  # 태스크 목록 조회
  aicli task list --workspace myproject`,
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
		Long:  `지정된 워크스페이스에서 새로운 Claude 태스크를 생성하고 실행합니다.

태스크는 명령형 또는 대화형으로 실행할 수 있습니다:
  • 명령형: --command 플래그로 실행할 명령어 지정
  • 대화형: --interactive 플래그로 Claude와 직접 대화

--detach 플래그를 사용하면 태스크를 백그라운드에서 실행하고 즉시
태스크 ID를 반환합니다. 이후 'aicli logs' 명령으로 진행 상황을 확인할 수 있습니다.`,
		Example: `  # 기본 명령형 태스크
  aicli task create --workspace myproject --command "implement login feature"
  
  # 대화형 모드
  aicli task create --workspace myproject --interactive
  
  # 백그라운드 실행
  aicli task create -w myproject -c "analyze codebase" --detach
  
  # Git 커밋 메시지 생성 예시
  aicli task create -w myproject -c "write commit message for recent changes"`,
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

	// 워크스페이스 자동완성
	cmd.RegisterFlagCompletionFunc("workspace", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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
	})

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
		Long:  `실행 중이거나 완료된 태스크 목록을 조회합니다.

기본적으로 실행 중인 태스크만 표시합니다. --all 플래그를 사용하면
완료된 태스크도 함께 표시합니다. --status 플래그로 특정 상태의
태스크만 필터링할 수 있습니다.

표시되는 정보:
  • 태스크 ID
  • 워크스페이스
  • 상태 (running, completed, failed, cancelled)
  • 시작 시간
  • 실행 시간 또는 종료 시간`,
		Example: `  # 실행 중인 태스크만 조회
  aicli task list
  
  # 특정 워크스페이스의 태스크 조회
  aicli task list --workspace myproject
  
  # 모든 태스크 조회
  aicli task list --all
  
  # 완료된 태스크만 조회
  aicli task list --status completed
  
  # JSON 형식으로 출력
  aicli task list --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: API 클라이언트를 통해 실제 태스크 목록 조회
			// 임시 데이터 (실제로는 API에서 가져옴)
			tasks := []map[string]interface{}{
				{
					"id":        "task-12345",
					"workspace": "project-alpha",
					"status":    "running",
					"command":   "implement login feature",
					"started_at": time.Now().Add(-10 * time.Minute).Format(time.RFC3339),
					"duration":   "10m",
				},
				{
					"id":         "task-12346",
					"workspace":  "project-beta",
					"status":     "completed",
					"command":    "fix database migration",
					"started_at":  time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
					"duration":    "45m",
				},
			}
			
			// 필터링 적용
			filteredTasks := []map[string]interface{}{}
			for _, task := range tasks {
				// 워크스페이스 필터
				if workspace != "" && task["workspace"] != workspace {
					continue
				}
				// 상태 필터
				if status != "" && task["status"] != status {
					continue
				}
				// all 플래그가 없으면 실행 중인 것만
				if !all && task["status"] != "running" {
					continue
				}
				filteredTasks = append(filteredTasks, task)
			}
			
			// 빈 목록 처리
			if len(filteredTasks) == 0 {
				fmt.Println("No tasks found")
				return nil
			}
			
			// 출력 포맷터 생성
			formatter := output.DefaultFormatterManager()
			formatter.SetHeaders([]string{"id", "workspace", "status", "command", "started_at", "duration"})
			
			return formatter.Print(filteredTasks)
		},
	}

	// 플래그 정의
	cmd.Flags().StringVarP(&workspace, "workspace", "w", "", "특정 워크스페이스의 태스크만 조회")
	cmd.Flags().BoolVarP(&all, "all", "a", false, "모든 상태의 태스크 조회")
	cmd.Flags().StringVarP(&status, "status", "s", "", "상태별 필터링 (running|completed|failed)")

	// 워크스페이스 자동완성
	cmd.RegisterFlagCompletionFunc("workspace", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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
	})

	// 상태 자동완성
	cmd.RegisterFlagCompletionFunc("status", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		statuses := []string{
			"running",
			"completed",
			"failed",
			"cancelled",
		}
		return statuses, cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}

// newTaskStatusCmd는 태스크 상태를 조회하는 명령어입니다.
func newTaskStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status [task-id]",
		Short: "태스크 상태 조회",
		Long:  `지정된 태스크의 상세 상태를 조회합니다.

표시되는 정보:
  • 태스크 ID 및 이름
  • 워크스페이스
  • 현재 상태
  • 시작 시간 및 실행 시간
  • 사용 중인 리소스 (CPU, 메모리)
  • 최근 로그 몇 줄
  • 진행률 (가능한 경우)`,
		Example: `  # 태스크 상태 확인
  aicli task status task-001
  
  # 짧은 별칭 사용
  aicli t status task-001
  
  # JSON 형식으로 출력
  aicli task status task-001 --output json`,
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			// TODO: 실제 태스크 ID 목록을 가져오는 로직 구현
			tasks := []string{
				"task-001",
				"task-002",
				"task-003",
				"bug-fix-101",
				"feature-201",
			}
			return tasks, cobra.ShellCompDirectiveNoFileComp
		},
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
		Long:  `실행 중인 태스크를 취소합니다.

태스크는 정상적으로 종료되며, 현재까지의 작업 결과는 보존됩니다.
--force 플래그를 사용하면 강제로 종료하며, 진행 중인 작업이
불완전할 수 있습니다.

취소된 태스크의 로그는 보존되며 'aicli logs' 명령으로 확인할 수 있습니다.`,
		Example: `  # 정상 취소
  aicli task cancel task-001
  
  # 강제 종료
  aicli task cancel task-001 --force
  
  # 짧은 별칭 사용
  aicli t cancel task-001 -f`,
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			// TODO: 실행 중인 태스크 ID만 반환하도록 개선
			tasks := []string{
				"task-001",
				"task-002",
				"task-003",
			}
			return tasks, cobra.ShellCompDirectiveNoFileComp
		},
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