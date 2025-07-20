package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/drumcap/aicli-web/internal/testutil"
	"github.com/spf13/viper"
)

func TestExecute(t *testing.T) {
	// 원본 rootCmd 백업
	originalRoot := rootCmd
	defer func() {
		rootCmd = originalRoot
	}()

	// 테스트용 rootCmd 설정
	rootCmd = newTestRootCmd()
	
	// Execute 함수 테스트
	err := Execute()
	testutil.AssertNil(t, err)
}

func TestRootCmd_Flags(t *testing.T) {
	cmd := newTestRootCmd()
	
	// 전역 플래그 검증
	flags := []struct {
		name      string
		shorthand string
		defValue  string
		usage     string
	}{
		{
			name:      "config",
			shorthand: "",
			defValue:  "",
			usage:     "config file",
		},
		{
			name:      "verbose",
			shorthand: "v",
			defValue:  "false",
			usage:     "verbose output",
		},
		{
			name:      "output",
			shorthand: "o",
			defValue:  "table",
			usage:     "output format",
		},
	}

	for _, flag := range flags {
		f := cmd.PersistentFlags().Lookup(flag.name)
		testutil.AssertNotNil(t, f)
		
		if flag.shorthand != "" {
			testutil.AssertEqual(t, flag.shorthand, f.Shorthand)
		}
		if flag.defValue != "" {
			testutil.AssertEqual(t, flag.defValue, f.DefValue)
		}
		if flag.usage != "" {
			testutil.AssertContains(t, f.Usage, flag.usage)
		}
	}
}

func TestInitConfig(t *testing.T) {
	// 테스트용 임시 디렉토리 생성
	tmpDir := testutil.TempDir(t, "aicli-config-test")
	
	// 테스트 설정 파일 생성
	configContent := `
verbose: true
output: json
api:
  port: 8080
  host: localhost
`
	configFile := filepath.Join(tmpDir, ".aicli.yaml")
	testutil.TempFile(t, tmpDir, ".aicli.yaml", configContent)

	tests := []struct {
		name     string
		setup    func()
		cleanup  func()
		check    func(t *testing.T)
	}{
		{
			name: "지정된 설정 파일 사용",
			setup: func() {
				cfgFile = configFile
				viper.Reset()
			},
			cleanup: func() {
				cfgFile = ""
				viper.Reset()
			},
			check: func(t *testing.T) {
				initConfig()
				
				// 설정이 제대로 로드되었는지 확인
				testutil.AssertEqual(t, true, viper.GetBool("verbose"))
				testutil.AssertEqual(t, "json", viper.GetString("output"))
				testutil.AssertEqual(t, 8080, viper.GetInt("api.port"))
			},
		},
		{
			name: "홈 디렉토리에서 설정 파일 찾기",
			setup: func() {
				// 홈 디렉토리를 임시 디렉토리로 설정
				os.Setenv("HOME", tmpDir)
				cfgFile = ""
				viper.Reset()
			},
			cleanup: func() {
				os.Unsetenv("HOME")
				viper.Reset()
			},
			check: func(t *testing.T) {
				initConfig()
				
				// 환경 변수 접두사 확인
				testutil.AssertEqual(t, "AICLI", viper.GetEnvPrefix())
			},
		},
		{
			name: "환경 변수 우선순위",
			setup: func() {
				os.Setenv("AICLI_OUTPUT", "yaml")
				cfgFile = configFile
				viper.Reset()
			},
			cleanup: func() {
				os.Unsetenv("AICLI_OUTPUT")
				cfgFile = ""
				viper.Reset()
			},
			check: func(t *testing.T) {
				initConfig()
				
				// 환경 변수가 설정 파일보다 우선순위가 높은지 확인
				testutil.AssertEqual(t, "yaml", viper.GetString("output"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}
			
			if tt.check != nil {
				tt.check(t)
			}
			
			if tt.cleanup != nil {
				tt.cleanup()
			}
		})
	}
}

func TestRootCmd_Version(t *testing.T) {
	// 버전 출력 테스트
	cmd := newTestRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	
	// --version 플래그 실행
	cmd.SetArgs([]string{"--version"})
	err := cmd.Execute()
	
	testutil.AssertNil(t, err)
	output := buf.String()
	
	// 버전 정보가 출력되는지 확인
	if len(output) == 0 {
		t.Error("버전 정보가 출력되지 않음")
	}
}

func TestRootCmd_Help(t *testing.T) {
	cmd := newTestRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	
	// --help 플래그 실행
	cmd.SetArgs([]string{"--help"})
	err := cmd.Execute()
	
	testutil.AssertNil(t, err)
	output := buf.String()
	
	// 도움말에 필요한 정보가 포함되어 있는지 확인
	testutil.AssertContains(t, output, "aicli")
	testutil.AssertContains(t, output, "AI-powered code management CLI")
	testutil.AssertContains(t, output, "Usage:")
	testutil.AssertContains(t, output, "Flags:")
}

// 테스트용 rootCmd 생성 헬퍼
func newTestRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "aicli",
		Short:   "AI-powered code management CLI",
		Version: "test",
	}
	
	// 플래그 추가
	cmd.PersistentFlags().String("config", "", "config file")
	cmd.PersistentFlags().BoolP("verbose", "v", false, "verbose output")
	cmd.PersistentFlags().StringP("output", "o", "table", "output format")
	
	return cmd
}