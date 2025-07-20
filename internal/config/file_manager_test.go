package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileManager(t *testing.T) {
	// 테스트용 임시 디렉토리 생성
	tempDir, err := os.MkdirTemp("", "aicli-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// HOME 환경 변수를 임시 디렉토리로 설정
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", oldHome)

	t.Run("NewFileManager", func(t *testing.T) {
		fm, err := NewFileManager()
		assert.NoError(t, err)
		assert.NotNil(t, fm)
		
		expectedDir := filepath.Join(tempDir, ConfigDirName)
		assert.Equal(t, expectedDir, fm.GetConfigDir())
		
		expectedPath := filepath.Join(expectedDir, ConfigFileName)
		assert.Equal(t, expectedPath, fm.GetConfigPath())
	})

	t.Run("EnsureConfigDir", func(t *testing.T) {
		fm, err := NewFileManager()
		require.NoError(t, err)

		// 디렉토리가 없는 상태에서 생성
		err = fm.EnsureConfigDir()
		assert.NoError(t, err)

		// 디렉토리가 생성되었는지 확인
		info, err := os.Stat(fm.GetConfigDir())
		assert.NoError(t, err)
		assert.True(t, info.IsDir())
		
		// 권한 확인 (0700)
		assert.Equal(t, os.FileMode(0700), info.Mode().Perm())

		// 이미 존재하는 상태에서도 에러가 없어야 함
		err = fm.EnsureConfigDir()
		assert.NoError(t, err)
	})

	t.Run("ConfigExists", func(t *testing.T) {
		fm, err := NewFileManager()
		require.NoError(t, err)

		// 초기에는 설정 파일이 없어야 함
		assert.False(t, fm.ConfigExists())

		// 설정 파일 생성
		err = fm.EnsureConfigDir()
		require.NoError(t, err)
		
		err = os.WriteFile(fm.GetConfigPath(), []byte("test"), 0600)
		require.NoError(t, err)

		// 이제 존재해야 함
		assert.True(t, fm.ConfigExists())
	})

	t.Run("ReadConfig - 파일이 없을 때", func(t *testing.T) {
		fm, err := NewFileManager()
		require.NoError(t, err)

		// 파일이 없으면 기본 설정 반환
		config, err := fm.ReadConfig()
		assert.NoError(t, err)
		assert.NotNil(t, config)
		
		// 기본값 확인
		assert.Equal(t, DefaultClaudeModel, config.Claude.Model)
		assert.Equal(t, DefaultClaudeTemperature, config.Claude.Temperature)
	})

	t.Run("WriteConfig and ReadConfig", func(t *testing.T) {
		fm, err := NewFileManager()
		require.NoError(t, err)

		// 설정 생성
		config := &Config{}
		config.Claude.APIKey = "test-api-key"
		config.Claude.Model = "claude-3-opus"
		config.Claude.Temperature = 0.5
		config.Claude.Timeout = 60

		// 설정 저장
		err = fm.WriteConfig(config)
		assert.NoError(t, err)

		// 설정 읽기
		readConfig, err := fm.ReadConfig()
		assert.NoError(t, err)
		assert.Equal(t, config.Claude.APIKey, readConfig.Claude.APIKey)
		assert.Equal(t, config.Claude.Model, readConfig.Claude.Model)
		assert.Equal(t, config.Claude.Temperature, readConfig.Claude.Temperature)
		assert.Equal(t, config.Claude.Timeout, readConfig.Claude.Timeout)

		// 파일 권한 확인
		err = fm.ValidatePermissions()
		assert.NoError(t, err)
	})

	t.Run("Backup and Restore", func(t *testing.T) {
		fm, err := NewFileManager()
		require.NoError(t, err)

		// 원본 설정
		config1 := &Config{}
		config1.Claude.APIKey = "original-key"
		config1.Claude.Model = "claude-3-opus"
		
		err = fm.WriteConfig(config1)
		require.NoError(t, err)

		// 백업 파일 확인
		backupPath := fm.GetConfigPath() + BackupSuffix
		_, err = os.Stat(backupPath)
		assert.NoError(t, err, "백업 파일이 생성되어야 함")

		// 새로운 설정으로 덮어쓰기
		config2 := &Config{}
		config2.Claude.APIKey = "new-key"
		config2.Claude.Model = "claude-3-sonnet"
		
		err = fm.WriteConfig(config2)
		require.NoError(t, err)

		// 현재 설정 확인
		current, err := fm.ReadConfig()
		require.NoError(t, err)
		assert.Equal(t, "new-key", current.Claude.APIKey)

		// 백업에서 복구
		err = fm.RestoreBackup()
		assert.NoError(t, err)

		// 복구된 설정 확인
		restored, err := fm.ReadConfig()
		require.NoError(t, err)
		assert.Equal(t, "original-key", restored.Claude.APIKey)
	})

	t.Run("ValidatePermissions", func(t *testing.T) {
		fm, err := NewFileManager()
		require.NoError(t, err)

		// 설정 파일 생성
		config := DefaultConfig()
		err = fm.WriteConfig(config)
		require.NoError(t, err)

		// 권한이 올바른지 확인
		err = fm.ValidatePermissions()
		assert.NoError(t, err)

		// 권한 변경
		err = os.Chmod(fm.GetConfigPath(), 0644)
		require.NoError(t, err)

		// 권한 검증 실패해야 함
		err = fm.ValidatePermissions()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "안전하지 않습니다")
	})

	t.Run("동시성 안전성", func(t *testing.T) {
		fm, err := NewFileManager()
		require.NoError(t, err)

		// 초기 설정
		config := &Config{}
		config.Claude.APIKey = "initial"
		err = fm.WriteConfig(config)
		require.NoError(t, err)

		// 동시에 여러 고루틴에서 읽기/쓰기
		done := make(chan bool)
		errors := make(chan error, 10)

		// 읽기 고루틴들
		for i := 0; i < 5; i++ {
			go func() {
				defer func() { done <- true }()
				for j := 0; j < 10; j++ {
					_, err := fm.ReadConfig()
					if err != nil {
						errors <- err
					}
				}
			}()
		}

		// 쓰기 고루틴들
		for i := 0; i < 2; i++ {
			go func(id int) {
				defer func() { done <- true }()
				for j := 0; j < 5; j++ {
					config := &Config{}
					config.Claude.APIKey = fmt.Sprintf("key-%d-%d", id, j)
					if err := fm.WriteConfig(config); err != nil {
						errors <- err
					}
				}
			}(i)
		}

		// 모든 고루틴 완료 대기
		for i := 0; i < 7; i++ {
			<-done
		}

		// 에러 확인
		close(errors)
		for err := range errors {
			t.Errorf("동시성 에러: %v", err)
		}
	})

	t.Run("RemoveConfig", func(t *testing.T) {
		fm, err := NewFileManager()
		require.NoError(t, err)

		// 설정 파일 생성
		config := DefaultConfig()
		err = fm.WriteConfig(config)
		require.NoError(t, err)
		assert.True(t, fm.ConfigExists())

		// 설정 파일 삭제
		err = fm.RemoveConfig()
		assert.NoError(t, err)
		assert.False(t, fm.ConfigExists())

		// 이미 없는 파일 삭제 시도
		err = fm.RemoveConfig()
		assert.NoError(t, err) // 에러가 없어야 함
	})
}