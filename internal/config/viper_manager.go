package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/go-playground/validator/v10"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// ConfigManager는 Viper를 사용한 설정 관리자입니다
type ConfigManager struct {
	viper       *viper.Viper
	fileManager *FileManager
	validator   *validator.Validate
	mutex       sync.RWMutex
	watchers    []ConfigWatcher
}

// ConfigWatcher는 설정 변경을 감지하는 인터페이스입니다
type ConfigWatcher interface {
	OnConfigChange(key string, oldValue, newValue interface{})
}

// NewConfigManager는 새로운 Viper 기반 설정 관리자를 생성합니다
func NewConfigManager() (*ConfigManager, error) {
	v := viper.New()
	fm, err := NewFileManager()
	if err != nil {
		return nil, err
	}

	// 검증기 초기화
	validate := validator.New()
	validate.RegisterValidation("dir", validateDirectory)

	cm := &ConfigManager{
		viper:       v,
		fileManager: fm,
		validator:   validate,
		watchers:    make([]ConfigWatcher, 0),
	}

	if err := cm.initialize(); err != nil {
		return nil, err
	}

	return cm, nil
}

// initialize는 ConfigManager를 초기화합니다
func (cm *ConfigManager) initialize() error {
	// 설정 디렉토리 확인
	if err := cm.fileManager.EnsureConfigDir(); err != nil {
		return err
	}

	// 설정 파일 경로 설정
	cm.viper.SetConfigName("config")
	cm.viper.SetConfigType("yaml")
	cm.viper.AddConfigPath(filepath.Dir(cm.fileManager.GetConfigPath()))

	// 환경 변수 설정
	cm.viper.SetEnvPrefix("AICLI")
	cm.viper.AutomaticEnv()
	cm.viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// 기본값 설정
	cm.setDefaults()

	// 설정 파일 읽기 (없으면 기본값 사용)
	if err := cm.viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("failed to read config file: %w", err)
		}
		// 설정 파일이 없으면 기본값으로 생성
		if err := cm.saveToFile(); err != nil {
			return fmt.Errorf("failed to create default config file: %w", err)
		}
	}

	return nil
}

// setDefaults는 기본값을 설정합니다
func (cm *ConfigManager) setDefaults() {
	homeDir, _ := os.UserHomeDir()

	// Claude 기본값
	cm.viper.SetDefault("claude.model", DefaultClaudeModelSimple)
	cm.viper.SetDefault("claude.temperature", DefaultClaudeTemperatureSimple)
	cm.viper.SetDefault("claude.timeout", DefaultClaudeTimeoutSimple)

	// Workspace 기본값
	cm.viper.SetDefault("workspace.default_path", filepath.Join(homeDir, ".aicli", "workspaces"))
	cm.viper.SetDefault("workspace.auto_sync", true)
	cm.viper.SetDefault("workspace.max_projects", DefaultMaxProjectsSimple)

	// Output 기본값
	cm.viper.SetDefault("output.format", DefaultOutputFormatSimple)
	cm.viper.SetDefault("output.color_mode", DefaultColorModeSimple)
	cm.viper.SetDefault("output.width", DefaultOutputWidthSimple)

	// Logging 기본값
	cm.viper.SetDefault("logging.level", DefaultLogLevelSimple)
	cm.viper.SetDefault("logging.file_path", filepath.Join(homeDir, ".aicli", "logs", "aicli.log"))
}

// Get은 설정 값을 가져옵니다 (우선순위: 플래그 > 환경변수 > 파일 > 기본값)
func (cm *ConfigManager) Get(key string) interface{} {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	return cm.viper.Get(key)
}

// GetString은 문자열 설정 값을 가져옵니다
func (cm *ConfigManager) GetString(key string) string {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	return cm.viper.GetString(key)
}

// GetInt는 정수 설정 값을 가져옵니다
func (cm *ConfigManager) GetInt(key string) int {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	return cm.viper.GetInt(key)
}

// GetFloat64는 실수 설정 값을 가져옵니다
func (cm *ConfigManager) GetFloat64(key string) float64 {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	return cm.viper.GetFloat64(key)
}

// GetBool은 불린 설정 값을 가져옵니다
func (cm *ConfigManager) GetBool(key string) bool {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	return cm.viper.GetBool(key)
}

// Set은 설정 값을 설정합니다
func (cm *ConfigManager) Set(key string, value interface{}) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// 현재 값과 소스 확인
	oldValue := cm.viper.Get(key)
	source := cm.GetValueSource(key)

	// 검증
	if err := cm.validateValue(key, value); err != nil {
		return fmt.Errorf("validation failed for %s: %w", key, err)
	}

	// 값 설정
	cm.viper.Set(key, value)

	// 파일에 저장 (파일 기반 설정인 경우)
	if source == "file" || source == "default" {
		if err := cm.saveToFile(); err != nil {
			return err
		}
	}

	// 환경 변수나 플래그로 설정된 경우 경고
	if source == "env" || source == "flag" {
		return fmt.Errorf("cannot override %s value set by %s", key, source)
	}

	// 변경 알림
	cm.notifyWatchers(key, oldValue, value)

	return nil
}

// GetValueSource는 설정 값의 소스를 확인합니다
func (cm *ConfigManager) GetValueSource(key string) string {
	// Viper는 소스 추적을 직접 지원하지 않으므로 우회 방법 사용
	envKey := strings.ToUpper(strings.ReplaceAll("AICLI_"+key, ".", "_"))
	if os.Getenv(envKey) != "" {
		return "env"
	}

	if cm.viper.IsSet(key) && !cm.isDefaultValue(key) {
		return "file"
	}

	return "default"
}

// isDefaultValue는 현재 값이 기본값인지 확인합니다
func (cm *ConfigManager) isDefaultValue(key string) bool {
	// 임시로 기본값만 있는 새 Viper 인스턴스 생성
	tempViper := viper.New()
	cm.setDefaultsForViper(tempViper)
	
	return cm.viper.Get(key) == tempViper.Get(key)
}

// setDefaultsForViper는 특정 Viper 인스턴스에 기본값을 설정합니다
func (cm *ConfigManager) setDefaultsForViper(v *viper.Viper) {
	homeDir, _ := os.UserHomeDir()

	// Claude 기본값
	v.SetDefault("claude.model", DefaultClaudeModelSimple)
	v.SetDefault("claude.temperature", DefaultClaudeTemperatureSimple)
	v.SetDefault("claude.timeout", DefaultClaudeTimeoutSimple)

	// Workspace 기본값
	v.SetDefault("workspace.default_path", filepath.Join(homeDir, ".aicli", "workspaces"))
	v.SetDefault("workspace.auto_sync", true)
	v.SetDefault("workspace.max_projects", DefaultMaxProjectsSimple)

	// Output 기본값
	v.SetDefault("output.format", DefaultOutputFormatSimple)
	v.SetDefault("output.color_mode", DefaultColorModeSimple)
	v.SetDefault("output.width", DefaultOutputWidthSimple)

	// Logging 기본값
	v.SetDefault("logging.level", DefaultLogLevelSimple)
	v.SetDefault("logging.file_path", filepath.Join(homeDir, ".aicli", "logs", "aicli.log"))
}

// validateValue는 설정 값을 검증합니다
func (cm *ConfigManager) validateValue(key string, value interface{}) error {
	// 키별 검증 규칙
	switch key {
	case "claude.api_key":
		if str, ok := value.(string); !ok || len(str) < 20 {
			return fmt.Errorf("API key must be at least 20 characters")
		}
	case "claude.model":
		validModels := []string{"claude-3-opus", "claude-3-sonnet", "claude-3-haiku"}
		if str, ok := value.(string); ok {
			for _, valid := range validModels {
				if str == valid {
					return nil
				}
			}
		}
		return fmt.Errorf("invalid model, must be one of: %v", validModels)
	case "claude.temperature":
		if f, ok := value.(float64); !ok || f < 0 || f > 1 {
			return fmt.Errorf("temperature must be between 0 and 1")
		}
	case "claude.timeout":
		if i, ok := value.(int); !ok || i < 1 {
			return fmt.Errorf("timeout must be at least 1 second")
		}
	case "workspace.max_projects":
		if i, ok := value.(int); !ok || i < 1 || i > 100 {
			return fmt.Errorf("max_projects must be between 1 and 100")
		}
	case "output.format":
		validFormats := []string{"table", "json", "yaml"}
		if str, ok := value.(string); ok {
			for _, valid := range validFormats {
				if str == valid {
					return nil
				}
			}
		}
		return fmt.Errorf("invalid format, must be one of: %v", validFormats)
	case "output.color_mode":
		validModes := []string{"auto", "always", "never"}
		if str, ok := value.(string); ok {
			for _, valid := range validModes {
				if str == valid {
					return nil
				}
			}
		}
		return fmt.Errorf("invalid color mode, must be one of: %v", validModes)
	case "output.width":
		if i, ok := value.(int); !ok || i < 40 {
			return fmt.Errorf("width must be at least 40")
		}
	case "logging.level":
		validLevels := []string{"debug", "info", "warn", "error"}
		if str, ok := value.(string); ok {
			for _, valid := range validLevels {
				if str == valid {
					return nil
				}
			}
		}
		return fmt.Errorf("invalid log level, must be one of: %v", validLevels)
	}

	return nil
}

// saveToFile은 현재 설정을 파일에 저장합니다
func (cm *ConfigManager) saveToFile() error {
	// Config 구조체로 변환
	config := &Config{}
	if err := cm.viper.Unmarshal(config); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// 파일에 저장
	return cm.fileManager.WriteConfig(config)
}

// Watch는 설정 파일 변경을 감지합니다
func (cm *ConfigManager) Watch() error {
	cm.viper.WatchConfig()
	cm.viper.OnConfigChange(func(e fsnotify.Event) {
		cm.handleConfigChange(e)
	})
	return nil
}

// handleConfigChange는 설정 파일 변경을 처리합니다
func (cm *ConfigManager) handleConfigChange(e fsnotify.Event) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// 변경 전 값들을 저장
	oldValues := make(map[string]interface{})
	for _, key := range cm.viper.AllKeys() {
		oldValues[key] = cm.viper.Get(key)
	}

	// 새 설정 읽기
	if err := cm.viper.ReadInConfig(); err != nil {
		return
	}

	// 변경된 키 찾기 및 알림
	for key, oldValue := range oldValues {
		newValue := cm.viper.Get(key)
		if oldValue != newValue {
			cm.notifyWatchers(key, oldValue, newValue)
		}
	}
}

// RegisterWatcher는 설정 변경 감시자를 등록합니다
func (cm *ConfigManager) RegisterWatcher(watcher ConfigWatcher) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cm.watchers = append(cm.watchers, watcher)
}

// notifyWatchers는 모든 감시자에게 변경을 알립니다
func (cm *ConfigManager) notifyWatchers(key string, oldValue, newValue interface{}) {
	for _, watcher := range cm.watchers {
		go func(w ConfigWatcher) {
			w.OnConfigChange(key, oldValue, newValue)
		}(watcher)
	}
}

// GetConfig는 전체 설정을 Config 구조체로 반환합니다
func (cm *ConfigManager) GetConfig() (*Config, error) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	config := &Config{}
	if err := cm.viper.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return config, nil
}

// Reset은 설정을 기본값으로 초기화합니다
func (cm *ConfigManager) Reset() error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// 모든 설정 초기화
	cm.viper = viper.New()
	cm.setDefaults()

	// 파일에 저장
	return cm.saveToFile()
}

// ConvertValue는 문자열 값을 적절한 타입으로 변환합니다
func (cm *ConfigManager) ConvertValue(key string, value string) (interface{}, error) {
	// 키별 타입 변환
	switch key {
	case "claude.temperature":
		var f float64
		if _, err := fmt.Sscanf(value, "%f", &f); err != nil {
			return nil, err
		}
		return f, nil
	case "claude.timeout", "workspace.max_projects", "output.width":
		var i int
		if _, err := fmt.Sscanf(value, "%d", &i); err != nil {
			return nil, err
		}
		return i, nil
	case "workspace.auto_sync":
		return strings.ToLower(value) == "true", nil
	default:
		return value, nil
	}
}

// BindFlags는 명령줄 플래그를 설정에 바인딩합니다
func (cm *ConfigManager) BindFlag(key string, flag interface{}) error {
	return cm.viper.BindPFlag(key, flag.(*pflag.Flag))
}

// AllSettings는 모든 설정을 맵으로 반환합니다
func (cm *ConfigManager) AllSettings() map[string]interface{} {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	return cm.viper.AllSettings()
}

// IsSet은 특정 키가 설정되었는지 확인합니다
func (cm *ConfigManager) IsSet(key string) bool {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	return cm.viper.IsSet(key)
}