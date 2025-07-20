---
task_id: T03C_S01
sprint_sequence_id: S01_M02
status: open
complexity: High
last_updated: 2025-07-21T06:27:00Z
github_issue: # Optional: GitHub issue number
---

# Task: 설정 통합 및 우선순위 시스템 구현

## Description
Viper 라이브러리를 활용하여 다중 설정 소스(파일, 환경 변수, 플래그)를 통합하고 우선순위 체계를 구현합니다. 동적 설정 로드 및 CLI 명령어 통합을 통해 완전한 설정 관리 시스템을 완성합니다.

## Goal / Objectives
- Viper 통합 및 다중 소스 설정 관리
- 설정 우선순위 체계 구현 (플래그 > 환경변수 > 파일 > 기본값)
- 동적 설정 로드 및 검증 시스템
- CLI 명령어 통합 (`aicli config` 명령어군)

## Acceptance Criteria
- [ ] Viper 통합 및 초기화 완료
- [ ] 설정 우선순위 체계 구현
- [ ] 동적 설정 검증 시스템 구현
- [ ] `aicli config get/set/list` 명령어 구현
- [ ] 설정 변경 감지 및 실시간 반영
- [ ] 설정 충돌 해결 메커니즘 구현

## Subtasks
- [ ] Viper 설정 및 초기화
- [ ] 다중 소스 통합 (파일, 환경변수, 플래그)
- [ ] 우선순위 체계 구현
- [ ] 설정 검증 시스템 구현
- [ ] `config` CLI 명령어 구현
- [ ] 동적 설정 감지 및 반영
- [ ] 충돌 해결 및 에러 처리

## 기술 가이드

### Viper 통합 및 초기화
```go
type ConfigManager struct {
    viper      *viper.Viper
    fileManager *FileManager
    validator   *validator.Validate
    mutex       sync.RWMutex
    watchers    []ConfigWatcher
}

func NewConfigManager() (*ConfigManager, error) {
    v := viper.New()
    fm, err := NewFileManager()
    if err != nil {
        return nil, err
    }
    
    cm := &ConfigManager{
        viper:       v,
        fileManager: fm,
        validator:   validator.New(),
    }
    
    if err := cm.initialize(); err != nil {
        return nil, err
    }
    
    return cm, nil
}

func (cm *ConfigManager) initialize() error {
    // 설정 파일 경로 설정
    cm.viper.SetConfigName("config")
    cm.viper.SetConfigType("yaml")
    cm.viper.AddConfigPath(cm.fileManager.configDir)
    
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
    }
    
    return nil
}
```

### 설정 우선순위 체계
```go
// 우선순위: 명령줄 플래그 > 환경 변수 > 설정 파일 > 기본값
func (cm *ConfigManager) Get(key string) interface{} {
    cm.mutex.RLock()
    defer cm.mutex.RUnlock()
    
    return cm.viper.Get(key)
}

func (cm *ConfigManager) Set(key string, value interface{}) error {
    cm.mutex.Lock()
    defer cm.mutex.Unlock()
    
    // 현재 값과 소스 확인
    oldValue := cm.viper.Get(key)
    source := cm.getValueSource(key)
    
    // 검증
    if err := cm.validateValue(key, value); err != nil {
        return fmt.Errorf("validation failed for %s: %w", key, err)
    }
    
    // 값 설정
    cm.viper.Set(key, value)
    
    // 파일에 저장 (파일 기반 설정인 경우)
    if source == "file" || source == "default" {
        return cm.saveToFile()
    }
    
    // 환경 변수나 플래그로 설정된 경우 경고
    if source == "env" || source == "flag" {
        return fmt.Errorf("cannot override %s value set by %s", key, source)
    }
    
    return nil
}

func (cm *ConfigManager) getValueSource(key string) string {
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
```

### 동적 설정 검증
```go
type ConfigValidator struct {
    validate *validator.Validate
    rules    map[string]ValidationRule
}

type ValidationRule struct {
    Required    bool
    Type        string
    Min         interface{}
    Max         interface{}
    Options     []string
    CustomFunc  func(interface{}) error
}

func (cm *ConfigManager) validateValue(key string, value interface{}) error {
    rule, exists := cm.validator.rules[key]
    if !exists {
        return nil // 규칙이 없으면 통과
    }
    
    // 타입 검증
    if err := cm.validateType(value, rule.Type); err != nil {
        return err
    }
    
    // 범위 검증
    if err := cm.validateRange(value, rule.Min, rule.Max); err != nil {
        return err
    }
    
    // 옵션 검증
    if len(rule.Options) > 0 {
        if err := cm.validateOptions(value, rule.Options); err != nil {
            return err
        }
    }
    
    // 커스텀 검증
    if rule.CustomFunc != nil {
        return rule.CustomFunc(value)
    }
    
    return nil
}
```

### CLI 명령어 통합
```go
func NewConfigCommand() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "config",
        Short: "Manage aicli configuration",
        Long:  "Get, set, and list configuration options for aicli",
    }
    
    cmd.AddCommand(
        newConfigGetCommand(),
        newConfigSetCommand(),
        newConfigListCommand(),
        newConfigValidateCommand(),
        newConfigResetCommand(),
    )
    
    return cmd
}

func newConfigGetCommand() *cobra.Command {
    return &cobra.Command{
        Use:   "get <key>",
        Short: "Get configuration value",
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            cm, err := config.GetManager()
            if err != nil {
                return err
            }
            
            key := args[0]
            value := cm.Get(key)
            
            if value == nil {
                return fmt.Errorf("configuration key '%s' not found", key)
            }
            
            fmt.Printf("%s = %v\n", key, value)
            return nil
        },
    }
}

func newConfigSetCommand() *cobra.Command {
    return &cobra.Command{
        Use:   "set <key> <value>",
        Short: "Set configuration value",
        Args:  cobra.ExactArgs(2),
        RunE: func(cmd *cobra.Command, args []string) error {
            cm, err := config.GetManager()
            if err != nil {
                return err
            }
            
            key := args[0]
            value := args[1]
            
            // 타입 변환
            typedValue, err := cm.ConvertValue(key, value)
            if err != nil {
                return fmt.Errorf("invalid value for %s: %w", key, err)
            }
            
            if err := cm.Set(key, typedValue); err != nil {
                return err
            }
            
            fmt.Printf("Successfully set %s = %v\n", key, typedValue)
            return nil
        },
    }
}
```

### 설정 변경 감지
```go
type ConfigWatcher interface {
    OnConfigChange(key string, oldValue, newValue interface{})
}

func (cm *ConfigManager) Watch() error {
    cm.viper.WatchConfig()
    cm.viper.OnConfigChange(func(e fsnotify.Event) {
        cm.handleConfigChange(e)
    })
    return nil
}

func (cm *ConfigManager) handleConfigChange(e fsnotify.Event) {
    // 설정 변경 처리
    for _, watcher := range cm.watchers {
        // 변경된 키-값 쌍을 감지하고 알림
        go func(w ConfigWatcher) {
            // 변경 감지 로직
            w.OnConfigChange("key", "oldValue", "newValue")
        }(watcher)
    }
}
```

### 충돌 해결 전략
- **우선순위 기반**: 높은 우선순위 소스가 낮은 우선순위 오버라이드
- **타입 안전성**: 설정 값의 타입 일관성 보장
- **검증 실패 시**: 이전 유효한 값으로 롤백
- **파일 손상 시**: 백업에서 복구 후 기본값 적용

## Output Log
*(This section is populated as work progresses on the task)*