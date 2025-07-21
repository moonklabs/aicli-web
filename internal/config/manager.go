package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-playground/validator/v10"
)

// Manager 는 설정 관리의 상위 레벨 인터페이스를 제공합니다
type Manager struct {
	fileManager *FileManager
	validator   *validator.Validate
	config      *Config
}

// NewManager 는 새로운 설정 관리자를 생성합니다
func NewManager() (*Manager, error) {
	fm, err := NewFileManager()
	if err != nil {
		return nil, err
	}

	// 설정 디렉토리 확인
	if err := fm.EnsureConfigDir(); err != nil {
		return nil, err
	}

	// 검증기 초기화
	v := validator.New()
	
	// 커스텀 검증 규칙 등록
	v.RegisterValidation("dir", validateDirectory)

	return &Manager{
		fileManager: fm,
		validator:   v,
	}, nil
}

// Load 는 설정을 로드합니다
func (m *Manager) Load() error {
	// 환경 변수 적용 전 파일에서 읽기
	config, err := m.fileManager.ReadConfig()
	if err != nil {
		return fmt.Errorf("설정 파일 로드 실패: %w", err)
	}

	// 환경 변수 적용
	m.applyEnvironmentVariables(config)

	// 검증
	if err := m.Validate(config); err != nil {
		return fmt.Errorf("설정 검증 실패: %w", err)
	}

	m.config = config
	return nil
}

// Save 는 현재 설정을 파일에 저장합니다
func (m *Manager) Save() error {
	if m.config == nil {
		return fmt.Errorf("저장할 설정이 없습니다")
	}

	// 검증
	if err := m.Validate(m.config); err != nil {
		return fmt.Errorf("설정 검증 실패: %w", err)
	}

	// 파일에 쓰기
	if err := m.fileManager.WriteConfig(m.config); err != nil {
		return fmt.Errorf("설정 파일 저장 실패: %w", err)
	}

	return nil
}

// Get 는 현재 설정을 반환합니다
func (m *Manager) Get() *Config {
	if m.config == nil {
		m.config = DefaultConfig()
	}
	return m.config
}

// Set 은 새로운 설정을 적용합니다
func (m *Manager) Set(config *Config) error {
	// 검증
	if err := m.Validate(config); err != nil {
		return err
	}

	m.config = config
	return nil
}

// Validate 는 설정의 유효성을 검사합니다
func (m *Manager) Validate(config *Config) error {
	if config == nil {
		return fmt.Errorf("설정이 nil입니다")
	}

	// 구조체 태그 기반 검증
	if err := m.validator.Struct(config); err != nil {
		return fmt.Errorf("설정 검증 실패: %w", err)
	}

	// 추가 비즈니스 로직 검증
	if config.Claude.APIKey == "" {
		return fmt.Errorf("Claude API 키가 필요합니다")
	}

	// 모델 검증
	validModels := []string{"claude-3-opus", "claude-3-sonnet", "claude-3-haiku"}
	modelValid := false
	for _, valid := range validModels {
		if config.Claude.Model == valid {
			modelValid = true
			break
		}
	}
	if !modelValid {
		return fmt.Errorf("지원하지 않는 Claude 모델: %s", config.Claude.Model)
	}

	return nil
}

// Reset 은 설정을 기본값으로 초기화합니다
func (m *Manager) Reset() error {
	m.config = DefaultConfig()
	return m.Save()
}

// Backup 은 현재 설정의 백업을 생성합니다
func (m *Manager) Backup() error {
	// 먼저 현재 설정을 저장
	if err := m.Save(); err != nil {
		return err
	}
	
	// 백업은 Write 과정에서 자동으로 생성됨
	return nil
}

// Restore 는 백업에서 설정을 복구합니다
func (m *Manager) Restore() error {
	if err := m.fileManager.RestoreBackup(); err != nil {
		return err
	}
	
	// 복구된 설정을 다시 로드
	return m.Load()
}

// GetConfigPath 는 설정 파일 경로를 반환합니다
func (m *Manager) GetConfigPath() string {
	return m.fileManager.GetConfigPath()
}

// applyEnvironmentVariables 는 환경 변수를 설정에 적용합니다
func (m *Manager) applyEnvironmentVariables(config *Config) {
	// Claude 설정
	if apiKey := os.Getenv("AICLI_CLAUDE_API_KEY"); apiKey != "" {
		config.Claude.APIKey = apiKey
	}
	if model := os.Getenv("AICLI_CLAUDE_MODEL"); model != "" {
		config.Claude.Model = model
	}
	if temp := os.Getenv("AICLI_CLAUDE_TEMPERATURE"); temp != "" {
		if t, err := parseFloat(temp); err == nil && t >= 0 && t <= 1 {
			config.Claude.Temperature = t
		}
	}
	if timeout := os.Getenv("AICLI_CLAUDE_TIMEOUT"); timeout != "" {
		if t, err := parseInt(timeout); err == nil && t > 0 {
			config.Claude.Timeout = t
		}
	}

	// Workspace 설정
	if path := os.Getenv("AICLI_WORKSPACE_DEFAULT_PATH"); path != "" {
		config.Workspace.DefaultPath = path
	}
	if autoSync := os.Getenv("AICLI_WORKSPACE_AUTO_SYNC"); autoSync != "" {
		config.Workspace.AutoSync = strings.ToLower(autoSync) == "true"
	}
	if maxProjects := os.Getenv("AICLI_WORKSPACE_MAX_PROJECTS"); maxProjects != "" {
		if m, err := parseInt(maxProjects); err == nil && m > 0 && m <= 100 {
			config.Workspace.MaxProjects = m
		}
	}

	// Output 설정
	if format := os.Getenv("AICLI_OUTPUT_FORMAT"); format != "" {
		if isValidFormat(format) {
			config.Output.Format = format
		}
	}
	if colorMode := os.Getenv("AICLI_OUTPUT_COLOR_MODE"); colorMode != "" {
		if isValidColorMode(colorMode) {
			config.Output.ColorMode = colorMode
		}
	}
	if width := os.Getenv("AICLI_OUTPUT_WIDTH"); width != "" {
		if w, err := parseInt(width); err == nil && w >= 40 {
			config.Output.Width = w
		}
	}

	// Logging 설정
	if level := os.Getenv("AICLI_LOG_LEVEL"); level != "" {
		if isValidLogLevel(level) {
			config.Logging.Level = level
		}
	}
	if filePath := os.Getenv("AICLI_LOG_FILE_PATH"); filePath != "" {
		config.Logging.FilePath = filePath
	}

	// Storage 설정
	if storageType := os.Getenv("AICLI_STORAGE_TYPE"); storageType != "" {
		if isValidStorageType(storageType) {
			config.Storage.Type = storageType
		}
	}
	if dataSource := os.Getenv("AICLI_STORAGE_DATA_SOURCE"); dataSource != "" {
		config.Storage.DataSource = dataSource
	}
	if maxConns := os.Getenv("AICLI_STORAGE_MAX_CONNS"); maxConns != "" {
		if m, err := parseInt(maxConns); err == nil && m > 0 && m <= 100 {
			config.Storage.MaxConns = m
		}
	}
}

// Helper 함수들

func parseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}

func parseInt(s string) (int, error) {
	var i int
	_, err := fmt.Sscanf(s, "%d", &i)
	return i, err
}

func isValidFormat(format string) bool {
	validFormats := []string{"table", "json", "yaml"}
	for _, valid := range validFormats {
		if format == valid {
			return true
		}
	}
	return false
}

func isValidColorMode(mode string) bool {
	validModes := []string{"auto", "always", "never"}
	for _, valid := range validModes {
		if mode == valid {
			return true
		}
	}
	return false
}

func isValidLogLevel(level string) bool {
	validLevels := []string{"debug", "info", "warn", "error"}
	for _, valid := range validLevels {
		if level == valid {
			return true
		}
	}
	return false
}

// validateDirectory 는 디렉토리 경로가 유효한지 검증합니다
func validateDirectory(fl validator.FieldLevel) bool {
	path := fl.Field().String()
	if path == "" {
		return true // 빈 값은 허용
	}
	
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	
	return info.IsDir()
}

// isValidStorageType 는 스토리지 타입이 유효한지 검증합니다
func isValidStorageType(storageType string) bool {
	validTypes := []string{"memory", "sqlite", "boltdb"}
	for _, valid := range validTypes {
		if storageType == valid {
			return true
		}
	}
	return false
}

// AllSettings는 모든 설정을 맵 형태로 반환합니다
func (m *Manager) AllSettings() map[string]interface{} {
	if m.config == nil {
		m.config = GetDefaultConfig()
	}
	
	// 구조체를 맵으로 변환
	return map[string]interface{}{
		"claude": map[string]interface{}{
			"api_key":     m.config.Claude.APIKey,
			"model":       m.config.Claude.Model,
			"temperature": m.config.Claude.Temperature,
			"timeout":     m.config.Claude.Timeout,
		},
		"workspace": map[string]interface{}{
			"default_path": m.config.Workspace.DefaultPath,
			"max_projects": m.config.Workspace.MaxProjects,
			"auto_sync":    m.config.Workspace.AutoSync,
		},
		"output": map[string]interface{}{
			"format":     m.config.Output.Format,
			"color_mode": m.config.Output.ColorMode,
			"width":      m.config.Output.Width,
		},
		"logging": map[string]interface{}{
			"level":     m.config.Logging.Level,
			"file_path": m.config.Logging.FilePath,
		},
	}
}

// IsSet은 특정 키가 설정되어 있는지 확인합니다
func (m *Manager) IsSet(key string) bool {
	// 환경 변수 확인
	envKey := "AICLI_" + strings.ToUpper(strings.ReplaceAll(key, ".", "_"))
	if os.Getenv(envKey) != "" {
		return true
	}
	
	// 설정 파일에서 확인
	return m.IsFromConfigFile(key)
}

// GetEnv는 해당 키의 환경 변수 값을 반환합니다
func (m *Manager) GetEnv(key string) string {
	envKey := "AICLI_" + strings.ToUpper(strings.ReplaceAll(key, ".", "_"))
	return os.Getenv(envKey)
}

// IsFromConfigFile은 해당 키가 설정 파일에서 왔는지 확인합니다
func (m *Manager) IsFromConfigFile(key string) bool {
	// 설정이 로드되지 않은 경우
	if m.config == nil {
		return false
	}
	
	// 기본값과 비교하여 변경되었는지 확인
	defaultConfig := GetDefaultConfig()
	
	switch key {
	case "claude.api_key":
		return m.config.Claude.APIKey != defaultConfig.Claude.APIKey
	case "claude.model":
		return m.config.Claude.Model != defaultConfig.Claude.Model
	case "claude.temperature":
		return m.config.Claude.Temperature != defaultConfig.Claude.Temperature
	case "claude.timeout":
		return m.config.Claude.Timeout != defaultConfig.Claude.Timeout
	case "workspace.default_path":
		return m.config.Workspace.DefaultPath != defaultConfig.Workspace.DefaultPath
	case "workspace.max_projects":
		return m.config.Workspace.MaxProjects != defaultConfig.Workspace.MaxProjects
	case "workspace.auto_sync":
		return m.config.Workspace.AutoSync != defaultConfig.Workspace.AutoSync
	case "output.format":
		return m.config.Output.Format != defaultConfig.Output.Format
	case "output.color_mode":
		return m.config.Output.ColorMode != defaultConfig.Output.ColorMode
	case "output.width":
		return m.config.Output.Width != defaultConfig.Output.Width
	case "logging.level":
		return m.config.Logging.Level != defaultConfig.Logging.Level
	case "logging.file_path":
		return m.config.Logging.FilePath != defaultConfig.Logging.FilePath
	default:
		return false
	}
}

// ConvertValue는 문자열 값을 적절한 타입으로 변환합니다
func (m *Manager) ConvertValue(key string, value string) (interface{}, error) {
	switch key {
	case "claude.api_key", "claude.model", "workspace.default_path", 
	     "output.format", "output.color_mode", "logging.level", "logging.file_path":
		return value, nil
	case "claude.temperature":
		return parseFloat(value)
	case "claude.timeout", "workspace.max_projects", "output.width":
		return parseInt(value)
	case "workspace.auto_sync":
		return strings.ToLower(value) == "true", nil
	default:
		return value, nil
	}
}

// Reset은 설정을 기본값으로 초기화합니다
func (m *Manager) Reset() error {
	m.config = GetDefaultConfig()
	return m.Save()
}