package config

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigManager_Basic(t *testing.T) {
	// 임시 설정 디렉토리 생성
	tempDir, err := ioutil.TempDir("", "aicli-config-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// HOME 환경 변수 임시 변경
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// ConfigManager 생성
	cm, err := NewConfigManager()
	require.NoError(t, err)

	// 기본값 테스트
	assert.Equal(t, "claude-3-opus", cm.GetString("claude.model"))
	assert.Equal(t, 0.7, cm.GetFloat64("claude.temperature"))
	assert.Equal(t, 30, cm.GetInt("claude.timeout"))
	assert.Equal(t, "table", cm.GetString("output.format"))
	assert.Equal(t, "info", cm.GetString("logging.level"))
}

func TestConfigManager_Set(t *testing.T) {
	// 임시 설정 디렉토리 생성
	tempDir, err := ioutil.TempDir("", "aicli-config-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// HOME 환경 변수 임시 변경
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// ConfigManager 생성
	cm, err := NewConfigManager()
	require.NoError(t, err)

	// 값 설정
	err = cm.Set("claude.model", "claude-3-sonnet")
	require.NoError(t, err)
	assert.Equal(t, "claude-3-sonnet", cm.GetString("claude.model"))

	err = cm.Set("claude.temperature", 0.9)
	require.NoError(t, err)
	assert.Equal(t, 0.9, cm.GetFloat64("claude.temperature"))

	err = cm.Set("workspace.max_projects", 20)
	require.NoError(t, err)
	assert.Equal(t, 20, cm.GetInt("workspace.max_projects"))

	err = cm.Set("workspace.auto_sync", false)
	require.NoError(t, err)
	assert.Equal(t, false, cm.GetBool("workspace.auto_sync"))
}

func TestConfigManager_Validation(t *testing.T) {
	// 임시 설정 디렉토리 생성
	tempDir, err := ioutil.TempDir("", "aicli-config-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// HOME 환경 변수 임시 변경
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// ConfigManager 생성
	cm, err := NewConfigManager()
	require.NoError(t, err)

	// 유효하지 않은 모델
	err = cm.Set("claude.model", "invalid-model")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid model")

	// 범위를 벗어난 temperature
	err = cm.Set("claude.temperature", 1.5)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "temperature must be between 0 and 1")

	// 음수 timeout
	err = cm.Set("claude.timeout", -1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timeout must be at least 1")

	// 범위를 벗어난 max_projects
	err = cm.Set("workspace.max_projects", 150)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max_projects must be between 1 and 100")

	// 유효하지 않은 output format
	err = cm.Set("output.format", "invalid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid format")

	// 유효하지 않은 log level
	err = cm.Set("logging.level", "trace")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid log level")
}

func TestConfigManager_EnvironmentVariables(t *testing.T) {
	// 임시 설정 디렉토리 생성
	tempDir, err := ioutil.TempDir("", "aicli-config-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// HOME 환경 변수 임시 변경
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// 환경 변수 설정
	os.Setenv("AICLI_CLAUDE_MODEL", "claude-3-haiku")
	os.Setenv("AICLI_CLAUDE_TEMPERATURE", "0.5")
	os.Setenv("AICLI_OUTPUT_FORMAT", "json")
	os.Setenv("AICLI_LOGGING_LEVEL", "debug")
	defer func() {
		os.Unsetenv("AICLI_CLAUDE_MODEL")
		os.Unsetenv("AICLI_CLAUDE_TEMPERATURE")
		os.Unsetenv("AICLI_OUTPUT_FORMAT")
		os.Unsetenv("AICLI_LOGGING_LEVEL")
	}()

	// ConfigManager 생성
	cm, err := NewConfigManager()
	require.NoError(t, err)

	// 환경 변수가 적용되었는지 확인
	assert.Equal(t, "claude-3-haiku", cm.GetString("claude.model"))
	assert.Equal(t, 0.5, cm.GetFloat64("claude.temperature"))
	assert.Equal(t, "json", cm.GetString("output.format"))
	assert.Equal(t, "debug", cm.GetString("logging.level"))
}

func TestConfigManager_ConvertValue(t *testing.T) {
	// 임시 설정 디렉토리 생성
	tempDir, err := ioutil.TempDir("", "aicli-config-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// HOME 환경 변수 임시 변경
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// ConfigManager 생성
	cm, err := NewConfigManager()
	require.NoError(t, err)

	// float 변환
	val, err := cm.ConvertValue("claude.temperature", "0.8")
	require.NoError(t, err)
	assert.Equal(t, 0.8, val)

	// int 변환
	val, err = cm.ConvertValue("claude.timeout", "60")
	require.NoError(t, err)
	assert.Equal(t, 60, val)

	// bool 변환
	val, err = cm.ConvertValue("workspace.auto_sync", "true")
	require.NoError(t, err)
	assert.Equal(t, true, val)

	val, err = cm.ConvertValue("workspace.auto_sync", "false")
	require.NoError(t, err)
	assert.Equal(t, false, val)

	// string (기본)
	val, err = cm.ConvertValue("output.format", "yaml")
	require.NoError(t, err)
	assert.Equal(t, "yaml", val)
}

func TestConfigManager_Reset(t *testing.T) {
	// 임시 설정 디렉토리 생성
	tempDir, err := ioutil.TempDir("", "aicli-config-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// HOME 환경 변수 임시 변경
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// ConfigManager 생성
	cm, err := NewConfigManager()
	require.NoError(t, err)

	// 값 변경
	err = cm.Set("claude.model", "claude-3-sonnet")
	require.NoError(t, err)
	err = cm.Set("logging.level", "debug")
	require.NoError(t, err)

	// 리셋
	err = cm.Reset()
	require.NoError(t, err)

	// 기본값으로 복원되었는지 확인
	assert.Equal(t, "claude-3-opus", cm.GetString("claude.model"))
	assert.Equal(t, "info", cm.GetString("logging.level"))
}

func TestConfigManager_Watcher(t *testing.T) {
	// 임시 설정 디렉토리 생성
	tempDir, err := ioutil.TempDir("", "aicli-config-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// HOME 환경 변수 임시 변경
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// ConfigManager 생성
	cm, err := NewConfigManager()
	require.NoError(t, err)

	// 테스트 감시자
	changed := make(chan bool, 1)
	var lastKey string
	var lastOldValue, lastNewValue interface{}

	testWatcher := &testConfigWatcher{
		onChange: func(key string, oldValue, newValue interface{}) {
			lastKey = key
			lastOldValue = oldValue
			lastNewValue = newValue
			changed <- true
		},
	}

	cm.RegisterWatcher(testWatcher)

	// 값 변경
	err = cm.Set("claude.model", "claude-3-sonnet")
	require.NoError(t, err)

	// 변경 알림 대기
	select {
	case <-changed:
		assert.Equal(t, "claude.model", lastKey)
		assert.Equal(t, "claude-3-opus", lastOldValue)
		assert.Equal(t, "claude-3-sonnet", lastNewValue)
	case <-time.After(time.Second):
		t.Fatal("Watcher notification timeout")
	}
}

// 테스트용 감시자
type testConfigWatcher struct {
	onChange func(string, interface{}, interface{})
}

func (w *testConfigWatcher) OnConfigChange(key string, oldValue, newValue interface{}) {
	if w.onChange != nil {
		w.onChange(key, oldValue, newValue)
	}
}

func TestConfigManager_FileCreation(t *testing.T) {
	// 임시 설정 디렉토리 생성
	tempDir, err := ioutil.TempDir("", "aicli-config-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// HOME 환경 변수 임시 변경
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// ConfigManager 생성
	cm, err := NewConfigManager()
	require.NoError(t, err)

	// 설정 파일이 생성되었는지 확인
	configPath := filepath.Join(tempDir, ".aicli", "config.yaml")
	assert.FileExists(t, configPath)

	// 파일 권한 확인
	info, err := os.Stat(configPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}