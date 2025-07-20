package config

import (
	"log"
)

// LoggingWatcher는 설정 변경을 로그에 기록하는 예제 감시자입니다
type LoggingWatcher struct {
	logger *log.Logger
}

// NewLoggingWatcher는 새로운 로깅 감시자를 생성합니다
func NewLoggingWatcher(logger *log.Logger) *LoggingWatcher {
	return &LoggingWatcher{
		logger: logger,
	}
}

// OnConfigChange는 설정이 변경될 때 호출됩니다
func (w *LoggingWatcher) OnConfigChange(key string, oldValue, newValue interface{}) {
	w.logger.Printf("Configuration changed: %s = %v -> %v", key, oldValue, newValue)
}

// ClaudeWatcher는 Claude 설정 변경을 처리하는 예제 감시자입니다
type ClaudeWatcher struct {
	onModelChange func(newModel string)
}

// NewClaudeWatcher는 새로운 Claude 감시자를 생성합니다
func NewClaudeWatcher(onModelChange func(string)) *ClaudeWatcher {
	return &ClaudeWatcher{
		onModelChange: onModelChange,
	}
}

// OnConfigChange는 설정이 변경될 때 호출됩니다
func (w *ClaudeWatcher) OnConfigChange(key string, oldValue, newValue interface{}) {
	if key == "claude.model" && w.onModelChange != nil {
		if newModel, ok := newValue.(string); ok {
			w.onModelChange(newModel)
		}
	}
}

// ReloadWatcher는 특정 설정 변경시 재시작이 필요한 컴포넌트를 처리하는 감시자입니다
type ReloadWatcher struct {
	reloadKeys map[string]func()
}

// NewReloadWatcher는 새로운 재로드 감시자를 생성합니다
func NewReloadWatcher() *ReloadWatcher {
	return &ReloadWatcher{
		reloadKeys: make(map[string]func()),
	}
}

// RegisterReloadKey는 특정 키 변경시 실행할 함수를 등록합니다
func (w *ReloadWatcher) RegisterReloadKey(key string, reloadFunc func()) {
	w.reloadKeys[key] = reloadFunc
}

// OnConfigChange는 설정이 변경될 때 호출됩니다
func (w *ReloadWatcher) OnConfigChange(key string, oldValue, newValue interface{}) {
	if reloadFunc, exists := w.reloadKeys[key]; exists {
		reloadFunc()
	}
}