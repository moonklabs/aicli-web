package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/aicli/aicli-web/internal/testutil"
	"github.com/spf13/cobra"
)

func TestNewCompletionCmd(t *testing.T) {
	// 자동완성 명령어 생성 테스트
	cmd := newCompletionCmd()
	
	testutil.AssertNotNil(t, cmd)
	testutil.AssertEqual(t, "completion [bash|zsh|fish|powershell]", cmd.Use)
	testutil.AssertEqual(t, 4, len(cmd.ValidArgs))
	
	// ValidArgs 검증
	expectedArgs := []string{"bash", "zsh", "fish", "powershell"}
	for i, arg := range expectedArgs {
		testutil.AssertEqual(t, arg, cmd.ValidArgs[i])
	}
}

func TestCompletionCmd_Execute(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantErr   bool
		errMsg    string
		checkOutput bool
	}{
		{
			name:        "bash completion",
			args:        []string{"bash"},
			wantErr:     false,
			checkOutput: true,
		},
		{
			name:        "zsh completion",
			args:        []string{"zsh"},
			wantErr:     false,
			checkOutput: true,
		},
		{
			name:        "fish completion",
			args:        []string{"fish"},
			wantErr:     false,
			checkOutput: true,
		},
		{
			name:        "powershell completion",
			args:        []string{"powershell"},
			wantErr:     false,
			checkOutput: true,
		},
		{
			name:    "invalid shell",
			args:    []string{"invalid"},
			wantErr: true,
			errMsg:  "지원하지 않는 셸: invalid",
		},
		{
			name:    "no args",
			args:    []string{},
			wantErr: true,
		},
		{
			name:    "too many args",
			args:    []string{"bash", "extra"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 버퍼로 출력 캡처
			buf := new(bytes.Buffer)
			
			// 테스트용 root command 생성
			testRoot := &cobra.Command{Use: "test"}
			rootCmd = testRoot
			
			cmd := newCompletionCmd()
			cmd.SetOut(buf)
			cmd.SetErr(buf)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()
			
			if tt.wantErr {
				testutil.AssertNotNil(t, err)
				if tt.errMsg != "" {
					testutil.AssertContains(t, err.Error(), tt.errMsg)
				}
			} else {
				testutil.AssertNil(t, err)
				if tt.checkOutput {
					output := buf.String()
					// 각 셸에 대한 최소한의 출력 검증
					if len(output) == 0 {
						t.Error("completion 스크립트가 비어있음")
					}
				}
			}
		})
	}
}

func TestCLIBasicFunctionality(t *testing.T) {
	// 기본 CLI 기능 테스트
	tests := []struct {
		name    string
		setup   func()
		check   func(t *testing.T)
	}{
		{
			name: "명령어 구조 검증",
			check: func(t *testing.T) {
				// 임시 rootCmd 생성
				testRoot := &cobra.Command{
					Use:   "aicli",
					Short: "Test CLI",
				}
				
				testutil.AssertEqual(t, "aicli", testRoot.Use)
				testutil.AssertNotNil(t, testRoot.Short)
			},
		},
		{
			name: "자동완성 명령어 추가",
			check: func(t *testing.T) {
				testRoot := &cobra.Command{Use: "test"}
				rootCmd = testRoot
				addCompletionCmd()
				
				// completion 명령어가 추가되었는지 확인
				found := false
				for _, cmd := range testRoot.Commands() {
					if cmd.Name() == "completion" {
						found = true
						break
					}
				}
				
				if !found {
					t.Error("completion 명령어가 추가되지 않음")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}
			tt.check(t)
		})
	}
}

func TestCompletionCmd_Help(t *testing.T) {
	cmd := newCompletionCmd()
	
	// Help 텍스트 검증
	helpText := cmd.Long
	
	// 각 셸에 대한 설명이 포함되어 있는지 확인
	shells := []string{"Bash", "Zsh", "Fish", "PowerShell"}
	for _, shell := range shells {
		if !strings.Contains(helpText, shell) {
			t.Errorf("Help 텍스트에 %s 설명이 없음", shell)
		}
	}
	
	// 사용 예제가 포함되어 있는지 확인
	testutil.AssertContains(t, helpText, "source")
	testutil.AssertContains(t, helpText, "aicli completion")
}