package commands

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/aicli/aicli-web/pkg/version"
)

// NewVersionCmd는 version 명령어를 생성합니다.
func NewVersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "버전 정보 표시",
		Long: `AICLI의 버전 정보와 빌드 세부사항을 표시합니다.

표시되는 정보:
  • AICLI 버전
  • Git 커밋 해시
  • 빌드 시간
  • Go 버전
  • 운영체제 및 아키텍처`,
		Example: `  # 버전 정보 표시
  aicli version
  
  # 짧은 버전만 표시
  aicli version --short
  
  # JSON 형식으로 출력
  aicli version --output json`,
		Run: func(cmd *cobra.Command, args []string) {
			short, _ := cmd.Flags().GetBool("short")
			
			if short {
				fmt.Println(version.Version)
				return
			}
			
			// 상세 버전 정보
			fmt.Printf("AICLI - AI-powered Code Management CLI\n")
			fmt.Printf("Version:      %s\n", version.Version)
			fmt.Printf("Git Commit:   %s\n", version.GitCommit)
			fmt.Printf("Built:        %s\n", version.BuildDate)
			fmt.Printf("Go Version:   %s\n", runtime.Version())
			fmt.Printf("OS/Arch:      %s/%s\n", runtime.GOOS, runtime.GOARCH)
			
			// 추가 정보
			fmt.Printf("\nClaude CLI Integration: Enabled\n")
			fmt.Printf("Docker Support:         Enabled\n")
			fmt.Printf("API Version:            v1\n")
			
			// 라이센스 정보
			fmt.Printf("\nLicense: MIT\n")
			fmt.Printf("Documentation: https://github.com/aicli/aicli-web\n")
			fmt.Printf("Report Issues: https://github.com/aicli/aicli-web/issues\n")
		},
	}
	
	// 플래그 정의
	cmd.Flags().BoolP("short", "s", false, "짧은 버전 정보만 표시")
	
	return cmd
}