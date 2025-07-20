---
task_id: T03C_S01
sprint_sequence_id: S01_M02
status: completed
complexity: High
last_updated: 2025-07-21 08:52
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
- [x] Viper 통합 및 초기화 완료
- [x] 설정 우선순위 체계 구현
- [x] 동적 설정 검증 시스템 구현
- [x] `aicli config get/set/list` 명령어 구현
- [x] 설정 변경 감지 및 실시간 반영
- [x] 설정 충돌 해결 메커니즘 구현

## Subtasks
- [x] Viper 설정 및 초기화
- [x] 다중 소스 통합 (파일, 환경변수, 플래그)
- [x] 우선순위 체계 구현
- [x] 설정 검증 시스템 구현
- [x] `config` CLI 명령어 구현
- [x] 동적 설정 감지 및 반영
- [x] 충돌 해결 및 에러 처리

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
[2025-07-21 08:38]: Viper 기반 ConfigManager 구현 시작
[2025-07-21 08:39]: internal/config/viper_manager.go 생성 - Viper 통합 설정 관리자 구현 완료
[2025-07-21 08:40]: 우선순위 체계 구현 - 플래그 > 환경변수 > 파일 > 기본값 순서 적용
[2025-07-21 08:41]: 동적 설정 검증 시스템 구현 - 키별 검증 규칙 및 타입 체크
[2025-07-21 08:42]: ConfigWatcher 인터페이스 및 설정 변경 감지 시스템 구현
[2025-07-21 08:43]: internal/cli/commands/config_viper.go 생성 - CLI config 명령어 구현
[2025-07-21 08:44]: config get/set/list/validate/reset 하위 명령어 모두 구현 완료
[2025-07-21 08:45]: internal/config/singleton.go 생성 - 싱글톤 패턴으로 전역 설정 접근 제공
[2025-07-21 08:46]: internal/config/watcher_example.go 생성 - 설정 감시자 예제 구현 (LoggingWatcher, ClaudeWatcher, ReloadWatcher)
[2025-07-21 08:47]: internal/config/viper_manager_test.go 생성 - 단위 테스트 작성 완료
[2025-07-21 08:48]: 기존 config.go를 config_old.go로 백업하고 Viper 기반 구현으로 교체
[2025-07-21 08:49]: examples/config_usage.go 생성 - 설정 시스템 사용 예제 작성
[2025-07-21 08:50]: FileManager에 GetConfigPath 메서드 추가 - ConfigManager와의 통합 완료
[2025-07-21 08:51]: 모든 하위 태스크 완료 - Viper 통합 설정 관리 시스템 구현 완료

[2025-07-21 08:52]: 코드 리뷰 - 실패
결과: **실패** - 사양과 구현 사이에 여러 불일치 사항 발견
**범위:** T03C_S01_Config_Integration 태스크의 코드 변경사항
**발견사항:** 
1. CLI 명령어에서 config.GetManager() 대신 config.NewConfigManager() 사용 (심각도: 7/10) - 사양에서는 싱글톤 패턴을 위한 GetManager() 사용을 명시했으나, 구현에서는 매번 새 인스턴스를 생성하는 NewConfigManager() 사용
2. handleConfigChange 구현의 차이 (심각도: 5/10) - 사양의 의사 코드보다 훨씬 상세한 구현이지만, 기술 가이드의 간단한 형태와 다름
3. getValueSource가 private 메서드 (심각도: 3/10) - 구현에서 사용하려 했으나 접근 불가하여 실제로는 제거됨
4. validator 초기화 시 커스텀 검증 규칙 추가 (심각도: 2/10) - 사양에 없는 dir 검증 규칙 등록
**요약:** 핵심 기능은 모두 구현되었으나, CLI 명령어에서 싱글톤 패턴을 사용하지 않는 중요한 불일치가 있음. 이는 설정 일관성과 성능에 영향을 줄 수 있음
**권장사항:** CLI 명령어의 config.NewConfigManager()를 config.GetManager()로 변경하여 싱글톤 패턴을 활용하도록 수정

[2025-07-21 08:53]: 코드 리뷰 문제 수정 작업 시작
[2025-07-21 08:53]: internal/cli/commands/config.go 수정 - config.NewConfigManager()를 config.GetManager()로 변경 (5곳)
[2025-07-21 08:54]: internal/config/viper_manager.go 수정 - getValueSource를 GetValueSource로 변경하여 public 메서드로 전환
[2025-07-21 08:54]: internal/cli/commands/config.go 수정 - 설정 값 출력 시 소스 정보도 함께 표시하도록 개선

[2025-07-21 08:55]: 코드 리뷰 재실행 - 통과
결과: **통과** - 주요 불일치 사항이 수정됨
**범위:** T03C_S01_Config_Integration 태스크의 수정된 코드
**발견사항:** 
- CLI 명령어에서 config.GetManager() 사용 확인 (수정 완료)
- GetValueSource 메서드가 public으로 변경되어 설정 소스 표시 가능 (수정 완료)
- handleConfigChange 구현이 사양보다 상세하지만 기능적으로 올바름 (허용 가능)
- validator 초기화 시 dir 검증 규칙 추가는 실용적인 개선사항 (허용 가능)
**요약:** 핵심 불일치 사항이 모두 수정되었으며, Viper 통합 설정 관리 시스템이 사양에 따라 올바르게 구현됨
**권장사항:** 없음 - 구현이 사양을 준수하며 추가된 개선사항도 실용적임