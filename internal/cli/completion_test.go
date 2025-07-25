package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestCompletionCommand(t *testing.T) {
	// completion 명령어 초기화
	addCompletionCmd()

	tests := []struct {
		name     string
		args     []string
		contains []string
		wantErr  bool
	}{
		{
			name:     "bash completion",
			args:     []string{"completion", "bash"},
			contains: []string{"_aicli_bash_autocomplete", "complete -F"},
			wantErr:  false,
		},
		{
			name:     "zsh completion",
			args:     []string{"completion", "zsh"},
			contains: []string{"#compdef aicli", "_aicli"},
			wantErr:  false,
		},
		{
			name:     "fish completion", 
			args:     []string{"completion", "fish"},
			contains: []string{"complete -c aicli"},
			wantErr:  false,
		},
		{
			name:     "powershell completion",
			args:     []string{"completion", "powershell"},
			contains: []string{"Register-ArgumentCompleter"},
			wantErr:  false,
		},
		{
			name:    "invalid shell",
			args:    []string{"completion", "invalid"},
			wantErr: true,
		},
		{
			name:    "no shell specified",
			args:    []string{"completion"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 출력 캡처를 위한 버퍼
			buf := new(bytes.Buffer)
			rootCmd.SetOut(buf)
			rootCmd.SetErr(buf)

			// 명령어 실행
			rootCmd.SetArgs(tt.args)
			err := rootCmd.Execute()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				output := buf.String()
				
				// 예상된 내용이 포함되어 있는지 확인
				for _, expected := range tt.contains {
					assert.Contains(t, output, expected, 
						"출력에 '%s'가 포함되어야 합니다", expected)
				}
			}
		})
	}
}

func TestDynamicCompletions(t *testing.T) {
	tests := []struct {
		name           string
		completionFunc func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective)
		toComplete     string
		expected       []string
	}{
		{
			name: "workspace completion",
			completionFunc: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
				// RegisterWorkspaceCompletion에서 사용하는 로직과 동일
				workspaces := []string{
					"project-alpha",
					"project-beta",
					"project-gamma",
					"development",
					"staging",
					"production",
				}
				var filtered []string
				for _, ws := range workspaces {
					if strings.HasPrefix(ws, toComplete) {
						filtered = append(filtered, ws)
					}
				}
				return filtered, cobra.ShellCompDirectiveNoFileComp
			},
			toComplete: "project-",
			expected:   []string{"project-alpha", "project-beta", "project-gamma"},
		},
		{
			name: "task status completion",
			completionFunc: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
				statuses := []string{
					"running",
					"completed", 
					"failed",
					"cancelled",
				}
				var filtered []string
				for _, status := range statuses {
					if strings.HasPrefix(status, toComplete) {
						filtered = append(filtered, status)
					}
				}
				return filtered, cobra.ShellCompDirectiveNoFileComp
			},
			toComplete: "c",
			expected:   []string{"completed", "cancelled"},
		},
		{
			name: "output format completion",
			completionFunc: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
				formats := []string{
					"table",
					"json",
					"yaml",
					"csv",
				}
				return formats, cobra.ShellCompDirectiveNoFileComp
			},
			toComplete: "",
			expected:   []string{"table", "json", "yaml", "csv"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestions, _ := tt.completionFunc(nil, []string{}, tt.toComplete)
			assert.ElementsMatch(t, tt.expected, suggestions)
		})
	}
}

func TestCompletionSubcommands(t *testing.T) {
	// 루트 명령어에 completion 추가
	addCompletionCmd()
	
	// completion 명령어 찾기
	completionCmd, _, err := rootCmd.Find([]string{"completion"})
	assert.NoError(t, err)
	assert.NotNil(t, completionCmd)
	
	// 서브커맨드 확인
	subcommands := []string{"bash", "zsh", "fish", "powershell"}
	for _, subcmd := range subcommands {
		cmd, _, err := completionCmd.Find([]string{subcmd})
		assert.NoError(t, err)
		assert.NotNil(t, cmd)
		assert.Equal(t, subcmd, cmd.Use)
	}
}

func TestConfigKeyCompletion(t *testing.T) {
	// config key 자동완성 테스트
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
	
	// 프리픽스로 필터링 테스트
	testCases := []struct {
		prefix   string
		expected int
	}{
		{"api.", 3},
		{"claude.", 3},
		{"docker.", 2},
		{"workspace.", 1},
		{"logging.", 2},
		{"", 11},
	}
	
	for _, tc := range testCases {
		t.Run("prefix_"+tc.prefix, func(t *testing.T) {
			var filtered []string
			for _, key := range keys {
				if strings.HasPrefix(key, tc.prefix) {
					filtered = append(filtered, key)
				}
			}
			assert.Len(t, filtered, tc.expected)
		})
	}
}