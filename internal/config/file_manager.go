package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

const (
	ConfigDirName  = ".aicli"
	ConfigFileName = "config.yaml"
	BackupSuffix   = ".backup"
)

// FileManager 는 설정 파일의 읽기/쓰기 및 디렉토리 관리를 담당합니다
type FileManager struct {
	configDir  string
	configPath string
	mutex      sync.RWMutex
}

// NewFileManager 는 새로운 FileManager 인스턴스를 생성합니다
func NewFileManager() (*FileManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ConfigDirName)
	configPath := filepath.Join(configDir, ConfigFileName)

	return &FileManager{
		configDir:  configDir,
		configPath: configPath,
	}, nil
}

// EnsureConfigDir 는 설정 디렉토리가 존재하는지 확인하고, 없으면 생성합니다
func (fm *FileManager) EnsureConfigDir() error {
	if err := os.MkdirAll(fm.configDir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	return nil
}

// configExists 는 설정 파일이 존재하는지 확인합니다
func (fm *FileManager) configExists() bool {
	_, err := os.Stat(fm.configPath)
	return !os.IsNotExist(err)
}

// ReadConfig 는 설정 파일을 읽어 Config 구조체로 반환합니다
func (fm *FileManager) ReadConfig() (*Config, error) {
	fm.mutex.RLock()
	defer fm.mutex.RUnlock()

	if !fm.configExists() {
		// 설정 파일이 없으면 기본 설정 반환
		return GetDefaultConfig(), nil
	}

	data, err := os.ReadFile(fm.configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// WriteConfig 는 Config 구조체를 설정 파일로 저장합니다
func (fm *FileManager) WriteConfig(config *Config) error {
	fm.mutex.Lock()
	defer fm.mutex.Unlock()

	// 디렉토리 확인
	if err := fm.EnsureConfigDir(); err != nil {
		return err
	}

	// 백업 생성
	if err := fm.createBackup(); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// 임시 파일에 먼저 쓰기
	tempPath := fm.configPath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// 파일 권한 설정
	if err := fm.secureFile(tempPath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to set file permissions: %w", err)
	}

	// 원자적 이동
	if err := os.Rename(tempPath, fm.configPath); err != nil {
		os.Remove(tempPath) // 정리
		return fmt.Errorf("failed to move temp file: %w", err)
	}

	return nil
}

// secureFile 은 파일 권한을 0600으로 설정합니다
func (fm *FileManager) secureFile(path string) error {
	// 파일 권한을 0600으로 설정 (소유자만 읽기/쓰기)
	if err := os.Chmod(path, 0600); err != nil {
		return fmt.Errorf("failed to set file permissions: %w", err)
	}
	return nil
}

// validatePermissions 는 설정 파일의 권한이 안전한지 확인합니다
func (fm *FileManager) ValidatePermissions() error {
	info, err := os.Stat(fm.configPath)
	if err != nil {
		return err
	}

	mode := info.Mode().Perm()
	if mode != 0600 {
		return fmt.Errorf("config file has insecure permissions: %o", mode)
	}

	return nil
}

// createBackup 은 현재 설정 파일의 백업을 생성합니다
func (fm *FileManager) createBackup() error {
	if !fm.configExists() {
		return nil // 백업할 파일이 없음
	}

	backupPath := fm.configPath + BackupSuffix

	src, err := os.Open(fm.configPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(backupPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return err
	}

	return fm.secureFile(backupPath)
}

// RestoreBackup 은 백업 파일에서 설정을 복구합니다
func (fm *FileManager) RestoreBackup() error {
	backupPath := fm.configPath + BackupSuffix
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup file not found")
	}

	return os.Rename(backupPath, fm.configPath)
}

// GetConfigPath 는 설정 파일의 전체 경로를 반환합니다
func (fm *FileManager) GetConfigPath() string {
	return fm.configPath
}

// RemoveConfig 는 설정 파일을 삭제합니다
func (fm *FileManager) RemoveConfig() error {
	fm.mutex.Lock()
	defer fm.mutex.Unlock()
	
	if !fm.configExists() {
		return nil
	}
	
	if err := os.Remove(fm.configPath); err != nil {
		return fmt.Errorf("failed to remove config file: %w", err)
	}
	
	return nil
}

