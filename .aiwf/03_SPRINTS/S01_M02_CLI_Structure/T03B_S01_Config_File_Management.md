---
task_id: T03B_S01
sprint_sequence_id: S01_M02
status: open
complexity: Medium
last_updated: 2025-07-21T06:26:00Z
github_issue: # Optional: GitHub issue number
---

# Task: 설정 파일 관리 시스템 구현

## Description
AICode Manager의 설정 파일 읽기/쓰기 및 파일 시스템 관리 기능을 구현합니다. 안전한 파일 권한 관리와 디렉토리 자동 생성을 통해 안정적인 설정 파일 관리를 제공합니다.

## Goal / Objectives
- YAML 설정 파일 읽기/쓰기 구현
- 설정 디렉토리 자동 생성 및 권한 관리
- 파일 잠금 및 동시성 안전성 확보
- 설정 파일 백업 및 복구 기능

## Acceptance Criteria
- [ ] `~/.aicli/config.yaml` 파일 자동 생성
- [ ] YAML 파일 읽기/쓰기 기능 구현
- [ ] 디렉토리 권한 설정 (0700) 적용
- [ ] 파일 권한 설정 (0600) 적용
- [ ] 동시 액세스 안전성 확보
- [ ] 설정 파일 백업 메커니즘 구현

## Subtasks
- [ ] 설정 디렉토리 관리 구현
- [ ] YAML 파일 읽기 기능
- [ ] YAML 파일 쓰기 기능
- [ ] 파일 권한 및 보안 설정
- [ ] 파일 잠금 및 동시성 처리
- [ ] 백업 및 복구 기능
- [ ] 파일 시스템 에러 처리

## 기술 가이드

### 설정 디렉토리 관리
```go
const (
    ConfigDirName  = ".aicli"
    ConfigFileName = "config.yaml"
    BackupSuffix   = ".backup"
)

type FileManager struct {
    configDir  string
    configPath string
    mutex      sync.RWMutex
}

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

func (fm *FileManager) EnsureConfigDir() error {
    if err := os.MkdirAll(fm.configDir, 0700); err != nil {
        return fmt.Errorf("failed to create config directory: %w", err)
    }
    return nil
}
```

### YAML 파일 읽기/쓰기
```go
func (fm *FileManager) ReadConfig() (*Config, error) {
    fm.mutex.RLock()
    defer fm.mutex.RUnlock()
    
    if !fm.configExists() {
        return DefaultConfig(), nil
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

func (fm *FileManager) WriteConfig(config *Config) error {
    fm.mutex.Lock()
    defer fm.mutex.Unlock()
    
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
    
    // 원자적 이동
    if err := os.Rename(tempPath, fm.configPath); err != nil {
        os.Remove(tempPath) // 정리
        return fmt.Errorf("failed to move temp file: %w", err)
    }
    
    return nil
}
```

### 파일 권한 및 보안
```go
func (fm *FileManager) secureFile(path string) error {
    // 파일 권한을 0600으로 설정 (소유자만 읽기/쓰기)
    if err := os.Chmod(path, 0600); err != nil {
        return fmt.Errorf("failed to set file permissions: %w", err)
    }
    return nil
}

func (fm *FileManager) validatePermissions() error {
    info, err := os.Stat(fm.configPath)
    if err != nil {
        return err
    }
    
    mode := info.Mode()
    if mode != 0600 {
        return fmt.Errorf("config file has insecure permissions: %o", mode)
    }
    
    return nil
}
```

### 백업 및 복구 메커니즘
```go
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

func (fm *FileManager) restoreBackup() error {
    backupPath := fm.configPath + BackupSuffix
    if _, err := os.Stat(backupPath); os.IsNotExist(err) {
        return fmt.Errorf("backup file not found")
    }
    
    return os.Rename(backupPath, fm.configPath)
}
```

### 에러 처리 전략
- **읽기 실패**: 기본 설정으로 폴백
- **쓰기 실패**: 백업에서 복구
- **권한 에러**: 사용자에게 권한 수정 안내
- **파싱 에러**: 구체적인 오류 위치 제공

### 동시성 안전성
- `sync.RWMutex`를 활용한 읽기/쓰기 잠금
- 원자적 파일 업데이트 (임시 파일 + 이동)
- 프로세스 간 파일 잠금 (필요시 `flock` 활용)

## Output Log
*(This section is populated as work progresses on the task)*