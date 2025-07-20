package commands

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

// NewLogsCmd는 logs 관련 명령어를 생성합니다.
func NewLogsCmd() *cobra.Command {
	var (
		workspace  string
		taskID     string
		follow     bool
		since      string
		tail       int
		timestamps bool
	)

	cmd := &cobra.Command{
		Use:   "logs",
		Short: "로그 조회",
		Long:  `워크스페이스나 태스크의 로그를 조회합니다.

워크스페이스의 모든 로그나 특정 태스크의 로그를 조회할 수 있습니다.
--follow 플래그를 사용하면 실시간으로 로그를 스트리밍합니다.

로그 필터링 옵션:
  • --since: 특정 시간 이후의 로그만 조회 (예: 10m, 1h, 2d)
  • --tail: 마지막 N개 라인만 조회
  • --timestamps: 각 로그 라인에 타임스탬프 표시

로그는 표준 출력으로 스트리밍되므로 다른 도구와 파이프로 연결할 수 있습니다.`,
		Example: `  # 워크스페이스의 모든 로그 조회
  aicli logs --workspace myproject
  
  # 특정 태스크 로그 조회
  aicli logs --task task-001
  
  # 실시간 로그 스트리밍
  aicli logs -w myproject --follow
  
  # 최근 10분간의 로그만 조회
  aicli logs -w myproject --since 10m
  
  # 마지막 100줄만 조회
  aicli logs -t task-001 --tail 100
  
  # 타임스탬프와 함께 로그 출력
  aicli logs -w myproject --timestamps
  
  # grep과 함께 사용
  aicli logs -w myproject | grep ERROR`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 워크스페이스나 태스크 ID 중 하나는 필수
			if workspace == "" && taskID == "" {
				return fmt.Errorf("워크스페이스(--workspace) 또는 태스크 ID(--task) 중 하나는 필수입니다")
			}

			// TODO: API 클라이언트를 통해 실제 로그 조회
			if taskID != "" {
				fmt.Printf("Fetching logs for task: %s\n", taskID)
			} else {
				fmt.Printf("Fetching logs for workspace: %s\n", workspace)
			}

			if since != "" {
				fmt.Printf("Showing logs since: %s\n", since)
			}

			if tail > 0 {
				fmt.Printf("Showing last %d lines\n", tail)
			}

			// 샘플 로그 출력
			fmt.Println("---")
			if timestamps {
				fmt.Printf("[%s] Task started\n", time.Now().Format(time.RFC3339))
				fmt.Printf("[%s] Initializing Claude CLI...\n", time.Now().Format(time.RFC3339))
			} else {
				fmt.Println("Task started")
				fmt.Println("Initializing Claude CLI...")
			}

			if follow {
				fmt.Println("Following log output... (press Ctrl+C to stop)")
				// TODO: 실시간 로그 스트리밍 구현
			}

			return nil
		},
	}

	// 플래그 정의
	cmd.Flags().StringVarP(&workspace, "workspace", "w", "", "워크스페이스 이름")
	cmd.Flags().StringVarP(&taskID, "task", "t", "", "태스크 ID")
	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "실시간 로그 스트리밍")
	cmd.Flags().StringVarP(&since, "since", "s", "", "특정 시간 이후의 로그만 조회 (예: 10m, 1h)")
	cmd.Flags().IntVar(&tail, "tail", 0, "마지막 N개 라인만 조회")
	cmd.Flags().BoolVar(&timestamps, "timestamps", false, "타임스탬프 표시")

	return cmd
}