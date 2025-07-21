package errors

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// DiagnosticInfo는 시스템 진단 정보를 담는 구조체입니다.
type DiagnosticInfo struct {
	System      SystemInfo            `json:"system"`
	Environment EnvironmentInfo       `json:"environment"`
	Config      ConfigInfo            `json:"config"`
	Process     ProcessInfo           `json:"process"`
	Timestamp   time.Time             `json:"timestamp"`
	Custom      map[string]interface{} `json:"custom,omitempty"`
}

// SystemInfo는 시스템 정보를 담습니다.
type SystemInfo struct {
	OS           string `json:"os"`
	Architecture string `json:"architecture"`
	GoVersion    string `json:"go_version"`
	NumCPU       int    `json:"num_cpu"`
	WorkingDir   string `json:"working_dir"`
	HomeDir      string `json:"home_dir"`
	TempDir      string `json:"temp_dir"`
}

// EnvironmentInfo는 환경 변수 정보를 담습니다.
type EnvironmentInfo struct {
	PATH             string            `json:"path"`
	CLIConfigDir     string            `json:"cli_config_dir"`
	ClaudeAPIKey     bool              `json:"claude_api_key_set"` // 값은 노출하지 않고 설정 여부만
	CustomEnvVars    map[string]string `json:"custom_env_vars,omitempty"`
	TerminalSupport  TerminalInfo      `json:"terminal"`
}

// TerminalInfo는 터미널 지원 정보를 담습니다.
type TerminalInfo struct {
	ColorSupport bool   `json:"color_support"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	Term         string `json:"term"`
}

// ConfigInfo는 설정 상태 정보를 담습니다.
type ConfigInfo struct {
	ConfigPath   string              `json:"config_path"`
	ConfigExists bool                `json:"config_exists"`
	ConfigValid  bool                `json:"config_valid"`
	ConfigError  string              `json:"config_error,omitempty"`
	Settings     map[string]interface{} `json:"settings,omitempty"`
}

// ProcessInfo는 현재 프로세스 정보를 담습니다.
type ProcessInfo struct {
	PID        int      `json:"pid"`
	PPID       int      `json:"ppid"`
	Args       []string `json:"args"`
	Executable string   `json:"executable"`
	Version    string   `json:"version"`
}

// DiagnosticCollector는 진단 정보를 수집하는 인터페이스입니다.
type DiagnosticCollector interface {
	Collect() *DiagnosticInfo
	CollectSystemInfo() SystemInfo
	CollectEnvironmentInfo() EnvironmentInfo
	CollectConfigInfo() ConfigInfo
	CollectProcessInfo() ProcessInfo
}

// DefaultDiagnosticCollector는 기본 진단 정보 수집기입니다.
type DefaultDiagnosticCollector struct {
	configPath string
	version    string
}

// NewDiagnosticCollector는 새로운 진단 정보 수집기를 생성합니다.
func NewDiagnosticCollector(configPath, version string) DiagnosticCollector {
	return &DefaultDiagnosticCollector{
		configPath: configPath,
		version:    version,
	}
}

// Collect는 모든 진단 정보를 수집합니다.
func (d *DefaultDiagnosticCollector) Collect() *DiagnosticInfo {
	return &DiagnosticInfo{
		System:      d.CollectSystemInfo(),
		Environment: d.CollectEnvironmentInfo(),
		Config:      d.CollectConfigInfo(),
		Process:     d.CollectProcessInfo(),
		Timestamp:   time.Now(),
		Custom:      make(map[string]interface{}),
	}
}

// CollectSystemInfo는 시스템 정보를 수집합니다.
func (d *DefaultDiagnosticCollector) CollectSystemInfo() SystemInfo {
	workingDir, _ := os.Getwd()
	homeDir, _ := os.UserHomeDir()
	tempDir := os.TempDir()
	
	return SystemInfo{
		OS:           runtime.GOOS,
		Architecture: runtime.GOARCH,
		GoVersion:    runtime.Version(),
		NumCPU:       runtime.NumCPU(),
		WorkingDir:   workingDir,
		HomeDir:      homeDir,
		TempDir:      tempDir,
	}
}

// CollectEnvironmentInfo는 환경 정보를 수집합니다.
func (d *DefaultDiagnosticCollector) CollectEnvironmentInfo() EnvironmentInfo {
	// Claude API 키 설정 여부 확인 (값은 노출하지 않음)
	claudeAPIKeySet := os.Getenv("CLAUDE_API_KEY") != "" || 
					 os.Getenv("ANTHROPIC_API_KEY") != ""
	
	// CLI 설정 디렉토리
	cliConfigDir := os.Getenv("AICLI_CONFIG_DIR")
	if cliConfigDir == "" {
		homeDir, _ := os.UserHomeDir()
		cliConfigDir = filepath.Join(homeDir, ".aicli")
	}
	
	// 터미널 정보
	terminal := d.collectTerminalInfo()
	
	// 사용자 정의 환경 변수 (AICLI_ 프리픽스)
	customEnvVars := make(map[string]string)
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "AICLI_") {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) == 2 {
				// 민감한 정보는 마스킹
				value := parts[1]
				if strings.Contains(strings.ToUpper(parts[0]), "KEY") ||
				   strings.Contains(strings.ToUpper(parts[0]), "TOKEN") ||
				   strings.Contains(strings.ToUpper(parts[0]), "SECRET") {
					if len(value) > 8 {
						value = value[:4] + "****" + value[len(value)-4:]
					} else {
						value = "****"
					}
				}
				customEnvVars[parts[0]] = value
			}
		}
	}
	
	return EnvironmentInfo{
		PATH:            os.Getenv("PATH"),
		CLIConfigDir:    cliConfigDir,
		ClaudeAPIKey:    claudeAPIKeySet,
		CustomEnvVars:   customEnvVars,
		TerminalSupport: terminal,
	}
}

// collectTerminalInfo는 터미널 정보를 수집합니다.
func (d *DefaultDiagnosticCollector) collectTerminalInfo() TerminalInfo {
	// 색상 지원 확인
	colorSupport := os.Getenv("NO_COLOR") == "" && 
				   (os.Getenv("TERM") != "" && os.Getenv("TERM") != "dumb")
	
	// 터미널 크기는 운영체제별로 다르게 처리해야 하므로 기본값 사용
	width := 80
	height := 24
	
	return TerminalInfo{
		ColorSupport: colorSupport,
		Width:        width,
		Height:       height,
		Term:         os.Getenv("TERM"),
	}
}

// CollectConfigInfo는 설정 정보를 수집합니다.
func (d *DefaultDiagnosticCollector) CollectConfigInfo() ConfigInfo {
	configPath := d.configPath
	if configPath == "" {
		homeDir, _ := os.UserHomeDir()
		configPath = filepath.Join(homeDir, ".aicli", "config.yaml")
	}
	
	// 설정 파일 존재 여부 확인
	configExists := false
	if _, err := os.Stat(configPath); err == nil {
		configExists = true
	}
	
	// 설정 파일 유효성 확인 (간단한 검사)
	configValid := true
	configError := ""
	settings := make(map[string]interface{})
	
	if configExists {
		// 실제 설정 로드는 config 패키지에 의존하므로 여기서는 기본적인 검사만
		if file, err := os.Open(configPath); err != nil {
			configValid = false
			configError = fmt.Sprintf("설정 파일을 열 수 없습니다: %v", err)
		} else {
			file.Close()
			// 여기서 YAML 파싱을 할 수 있지만, 의존성을 최소화하기 위해 기본 검사만 수행
		}
	}
	
	return ConfigInfo{
		ConfigPath:   configPath,
		ConfigExists: configExists,
		ConfigValid:  configValid,
		ConfigError:  configError,
		Settings:     settings,
	}
}

// CollectProcessInfo는 프로세스 정보를 수집합니다.
func (d *DefaultDiagnosticCollector) CollectProcessInfo() ProcessInfo {
	executable, _ := os.Executable()
	
	// PPID는 직접 가져오기 어려우므로 0으로 설정
	ppid := 0
	
	return ProcessInfo{
		PID:        os.Getpid(),
		PPID:       ppid,
		Args:       os.Args,
		Executable: executable,
		Version:    d.version,
	}
}

// EnrichErrorWithDiagnostics는 에러에 진단 정보를 추가합니다.
func EnrichErrorWithDiagnostics(err *CLIError, collector DiagnosticCollector) *CLIError {
	if err == nil || collector == nil {
		return err
	}
	
	diagnostics := collector.Collect()
	
	// 시스템 정보 추가
	err.AddDebug("system_os", diagnostics.System.OS)
	err.AddDebug("system_arch", diagnostics.System.Architecture)
	err.AddDebug("go_version", diagnostics.System.GoVersion)
	err.AddDebug("working_dir", diagnostics.System.WorkingDir)
	
	// 환경 정보 추가
	err.AddDebug("config_dir", diagnostics.Environment.CLIConfigDir)
	err.AddDebug("claude_api_key_set", diagnostics.Environment.ClaudeAPIKey)
	err.AddDebug("color_support", diagnostics.Environment.TerminalSupport.ColorSupport)
	
	// 설정 정보 추가
	err.AddDebug("config_path", diagnostics.Config.ConfigPath)
	err.AddDebug("config_exists", diagnostics.Config.ConfigExists)
	err.AddDebug("config_valid", diagnostics.Config.ConfigValid)
	if diagnostics.Config.ConfigError != "" {
		err.AddDebug("config_error", diagnostics.Config.ConfigError)
	}
	
	// 프로세스 정보 추가
	err.AddDebug("process_pid", diagnostics.Process.PID)
	err.AddDebug("process_version", diagnostics.Process.Version)
	err.AddDebug("timestamp", diagnostics.Timestamp.Format(time.RFC3339))
	
	return err
}

// GenerateDiagnosticReport는 진단 보고서를 생성합니다.
func GenerateDiagnosticReport(collector DiagnosticCollector) string {
	diagnostics := collector.Collect()
	
	var buf strings.Builder
	
	buf.WriteString("=== AICode Manager 진단 보고서 ===\n\n")
	buf.WriteString(fmt.Sprintf("생성 시각: %s\n\n", diagnostics.Timestamp.Format("2006-01-02 15:04:05")))
	
	// 시스템 정보
	buf.WriteString("## 시스템 정보\n")
	buf.WriteString(fmt.Sprintf("운영체제: %s\n", diagnostics.System.OS))
	buf.WriteString(fmt.Sprintf("아키텍처: %s\n", diagnostics.System.Architecture))
	buf.WriteString(fmt.Sprintf("Go 버전: %s\n", diagnostics.System.GoVersion))
	buf.WriteString(fmt.Sprintf("CPU 코어: %d\n", diagnostics.System.NumCPU))
	buf.WriteString(fmt.Sprintf("작업 디렉토리: %s\n", diagnostics.System.WorkingDir))
	buf.WriteString(fmt.Sprintf("홈 디렉토리: %s\n", diagnostics.System.HomeDir))
	buf.WriteString("\n")
	
	// 환경 정보
	buf.WriteString("## 환경 정보\n")
	buf.WriteString(fmt.Sprintf("설정 디렉토리: %s\n", diagnostics.Environment.CLIConfigDir))
	buf.WriteString(fmt.Sprintf("Claude API 키 설정: %t\n", diagnostics.Environment.ClaudeAPIKey))
	buf.WriteString(fmt.Sprintf("터미널 색상 지원: %t\n", diagnostics.Environment.TerminalSupport.ColorSupport))
	buf.WriteString(fmt.Sprintf("터미널: %s\n", diagnostics.Environment.TerminalSupport.Term))
	
	if len(diagnostics.Environment.CustomEnvVars) > 0 {
		buf.WriteString("사용자 환경 변수:\n")
		for key, value := range diagnostics.Environment.CustomEnvVars {
			buf.WriteString(fmt.Sprintf("  %s: %s\n", key, value))
		}
	}
	buf.WriteString("\n")
	
	// 설정 정보
	buf.WriteString("## 설정 정보\n")
	buf.WriteString(fmt.Sprintf("설정 파일 경로: %s\n", diagnostics.Config.ConfigPath))
	buf.WriteString(fmt.Sprintf("설정 파일 존재: %t\n", diagnostics.Config.ConfigExists))
	buf.WriteString(fmt.Sprintf("설정 파일 유효: %t\n", diagnostics.Config.ConfigValid))
	if diagnostics.Config.ConfigError != "" {
		buf.WriteString(fmt.Sprintf("설정 오류: %s\n", diagnostics.Config.ConfigError))
	}
	buf.WriteString("\n")
	
	// 프로세스 정보
	buf.WriteString("## 프로세스 정보\n")
	buf.WriteString(fmt.Sprintf("프로세스 ID: %d\n", diagnostics.Process.PID))
	buf.WriteString(fmt.Sprintf("실행 파일: %s\n", diagnostics.Process.Executable))
	buf.WriteString(fmt.Sprintf("버전: %s\n", diagnostics.Process.Version))
	buf.WriteString(fmt.Sprintf("명령행 인수: %v\n", diagnostics.Process.Args))
	
	return buf.String()
}