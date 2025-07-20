package config

import (
	"os"
	"path/filepath"
)

// 기본값 상수 정의 (사양 준수)
const (
	DefaultClaudeModelSimple       = "claude-3-opus"
	DefaultClaudeTemperatureSimple = 0.7
	DefaultClaudeTimeoutSimple     = 30
	DefaultOutputFormatSimple      = "table"
	DefaultColorModeSimple         = "auto"
	DefaultOutputWidthSimple       = 120
	DefaultLogLevelSimple          = "info"
	DefaultMaxProjectsSimple       = 10
)

// DefaultConfigSimple는 사양에 맞는 기본 설정을 반환합니다
func DefaultConfigSimple() *Config {
	homeDir, _ := os.UserHomeDir()
	
	return &Config{
		Claude: struct {
			APIKey      string  `yaml:"api_key" mapstructure:"api_key" validate:"required"`
			Model       string  `yaml:"model" mapstructure:"model" validate:"required"`
			Temperature float64 `yaml:"temperature" mapstructure:"temperature" validate:"min=0,max=1"`
			Timeout     int     `yaml:"timeout" mapstructure:"timeout" validate:"min=1"`
		}{
			Model:       DefaultClaudeModelSimple,
			Temperature: DefaultClaudeTemperatureSimple,
			Timeout:     DefaultClaudeTimeoutSimple,
		},
		Workspace: struct {
			DefaultPath string `yaml:"default_path" mapstructure:"default_path" validate:"dir"`
			AutoSync    bool   `yaml:"auto_sync" mapstructure:"auto_sync"`
			MaxProjects int    `yaml:"max_projects" mapstructure:"max_projects" validate:"min=1,max=100"`
		}{
			DefaultPath: filepath.Join(homeDir, ".aicli", "workspaces"),
			AutoSync:    true,
			MaxProjects: DefaultMaxProjectsSimple,
		},
		Output: struct {
			Format    string `yaml:"format" mapstructure:"format" validate:"oneof=table json yaml"`
			ColorMode string `yaml:"color_mode" mapstructure:"color_mode" validate:"oneof=auto always never"`
			Width     int    `yaml:"width" mapstructure:"width" validate:"min=40"`
		}{
			Format:    DefaultOutputFormatSimple,
			ColorMode: DefaultColorModeSimple,
			Width:     DefaultOutputWidthSimple,
		},
		Logging: struct {
			Level    string `yaml:"level" mapstructure:"level" validate:"oneof=debug info warn error"`
			FilePath string `yaml:"file_path" mapstructure:"file_path"`
		}{
			Level:    DefaultLogLevelSimple,
			FilePath: filepath.Join(homeDir, ".aicli", "logs", "aicli.log"),
		},
	}
}