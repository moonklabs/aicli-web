package commands

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/aicli/aicli-web/internal/cli/errors"
	"github.com/aicli/aicli-web/internal/cli/output"
)

// NewWorkspaceCmd는 workspace 관련 명령어를 생성합니다.
func NewWorkspaceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "workspace",
		Aliases: []string{"ws"},
		Short:   "워크스페이스 관리",
		Long:    `Claude CLI 실행을 위한 격리된 워크스페이스를 관리합니다.

워크스페이스는 각 프로젝트마다 독립적으로 생성되며, Docker 컨테이너 내에서
안전하게 격리된 환경을 제공합니다. 각 워크스페이스는 자체 파일 시스템,
환경 변수, Claude API 설정을 가집니다.

워크스페이스를 통해:
  • 여러 프로젝트를 동시에 관리
  • 프로젝트별 독립적인 Claude 설정 사용
  • 안전한 코드 실행 환경 보장
  • 프로젝트 간 격리 및 보안 유지`,
		Example: `  # 새 워크스페이스 생성
  aicli workspace create --name myproject --path ~/projects/myapp
  
  # 워크스페이스 목록 조회
  aicli workspace list
  
  # 워크스페이스 정보 확인
  aicli workspace info myproject
  
  # 워크스페이스 삭제
  aicli workspace delete myproject`,
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
		Long:  `현재 생성된 모든 워크스페이스의 목록을 조회합니다.

각 워크스페이스의 이름, 상태, 생성 시간, 프로젝트 경로 등의
정보를 표시합니다. 출력 형식은 전역 --output 플래그로 변경할 수 있습니다.`,
		Example: `  # 기본 테이블 형식으로 목록 조회
  aicli workspace list
  
  # JSON 형식으로 출력
  aicli workspace list --output json
  
  # 짧은 별칭 사용
  aicli ws list`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: API 클라이언트를 통해 실제 워크스페이스 목록 조회
			// 임시 데이터 (실제로는 API에서 가져옴)
			workspaces := []map[string]interface{}{
				{
					"name":       "project-alpha",
					"status":     "active",
					"created_at": time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
					"path":       "/home/user/projects/alpha",
				},
				{
					"name":       "project-beta", 
					"status":     "inactive",
					"created_at": time.Now().Add(-48 * time.Hour).Format(time.RFC3339),
					"path":       "/home/user/projects/beta",
				},
			}
			
			// 빈 목록 처리
			if len(workspaces) == 0 {
				fmt.Println("No workspaces found")
				return nil
			}
			
			// 출력 포맷터 생성
			formatter := output.DefaultFormatterManager()
			formatter.SetHeaders([]string{"name", "status", "created_at", "path"})
			
			return formatter.Print(workspaces)
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
		Long:  `새로운 Claude CLI 워크스페이스를 생성합니다.

워크스페이스는 지정된 프로젝트 디렉토리를 Docker 볼륨으로 마운트하여
격리된 환경에서 Claude CLI를 실행할 수 있게 합니다. Claude API 키는
환경 변수나 --claude-key 플래그로 지정할 수 있습니다.`,
		Example: `  # 기본 워크스페이스 생성
  aicli workspace create --name myproject --path ~/projects/myapp
  
  # Claude API 키와 함께 생성
  aicli workspace create --name myproject --path ~/projects/myapp --claude-key sk-ant-...
  
  # 환경 변수에서 API 키 사용
  export CLAUDE_API_KEY=sk-ant-...
  aicli workspace create --name myproject --path ~/projects/myapp`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 필수 플래그 검증
			if name == "" {
				return errors.RequiredFlagError("name", "워크스페이스를 식별하는 고유한 이름")
			}
			if projectPath == "" {
				return errors.RequiredFlagError("path", "Claude가 작업할 프로젝트 디렉토리 경로")
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
		Long:  `지정된 워크스페이스를 삭제합니다.

워크스페이스와 관련된 모든 리소스(컨테이너, 볼륨, 로그 등)가 삭제됩니다.
실행 중인 태스크가 있는 경우 먼저 중지됩니다. --force 플래그를 사용하면
확인 프롬프트 없이 즉시 삭제합니다.

주의: 이 작업은 되돌릴 수 없습니다.`,
		Example: `  # 확인 후 삭제
  aicli workspace delete myproject
  
  # 강제 삭제 (확인 없음)
  aicli workspace delete myproject --force
  
  # 짧은 별칭 사용
  aicli ws delete myproject -f`,
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
		Long:  `지정된 워크스페이스의 상세 정보를 조회합니다.

표시되는 정보:
  • 워크스페이스 이름 및 ID
  • 프로젝트 경로
  • 생성 시간 및 최종 사용 시간
  • 상태 (활성/비활성)
  • 실행 중인 태스크 수
  • 사용 중인 리소스 (CPU, 메모리)
  • Claude API 설정 상태`,
		Example: `  # 워크스페이스 정보 조회
  aicli workspace info myproject
  
  # JSON 형식으로 출력
  aicli workspace info myproject --output json
  
  # 짧은 별칭 사용
  aicli ws info myproject`,
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
			// 임시 데이터 (실제로는 API에서 가져옴)
			info := map[string]interface{}{
				"name":          workspaceName,
				"id":            "ws-12345",
				"status":        "active",
				"created_at":    time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
				"last_used":     time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
				"project_path":  "/home/user/projects/" + workspaceName,
				"running_tasks": 2,
				"cpu_usage":     "15%",
				"memory_usage":  "512MB / 2GB",
				"claude_api":    "configured",
			}
			
			// 출력 포맷터 생성
			formatter := output.DefaultFormatterManager()
			
			return formatter.Print(info)
		},
	}
}