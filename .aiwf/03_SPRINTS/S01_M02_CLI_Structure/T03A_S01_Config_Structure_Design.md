---
task_id: T03A_S01
sprint_sequence_id: S01_M02
status: open
complexity: Medium
last_updated: 2025-07-21T06:25:00Z
github_issue: # Optional: GitHub issue number
---

# Task: 설정 구조체 및 스키마 설계

## Description
AICode Manager의 설정 관리 시스템의 기본 구조를 설계합니다. 설정 구조체, 기본값, 환경 변수 매핑을 정의하여 확장 가능하고 유지보수가 용이한 설정 체계를 구축합니다.

## Goal / Objectives
- 설정 구조체 및 데이터 모델 정의
- 기본값 상수 및 검증 규칙 설정
- 환경 변수 매핑 체계 구축
- 설정 스키마 문서화

## Acceptance Criteria
- [ ] Config 구조체 완전 정의 완료
- [ ] 모든 설정 항목의 기본값 정의
- [ ] 환경 변수 이름 매핑 규칙 수립
- [ ] 설정 검증 타입 및 규칙 정의
- [ ] 설정 스키마 문서 작성

## Subtasks
- [ ] 핵심 설정 구조체 설계
- [ ] 기본값 상수 정의
- [ ] 환경 변수 매핑 체계 설계
- [ ] 설정 검증 규칙 정의
- [ ] 설정 스키마 문서 작성

## 기술 가이드

### 설정 구조체 설계
```go
type Config struct {
    Claude struct {
        APIKey      string  `yaml:"api_key" mapstructure:"api_key" validate:"required"`
        Model       string  `yaml:"model" mapstructure:"model" validate:"required"`
        Temperature float64 `yaml:"temperature" mapstructure:"temperature" validate:"min=0,max=1"`
        Timeout     int     `yaml:"timeout" mapstructure:"timeout" validate:"min=1"`
    } `yaml:"claude" mapstructure:"claude"`
    
    Workspace struct {
        DefaultPath string `yaml:"default_path" mapstructure:"default_path" validate:"dir"`
        AutoSync    bool   `yaml:"auto_sync" mapstructure:"auto_sync"`
        MaxProjects int    `yaml:"max_projects" mapstructure:"max_projects" validate:"min=1,max=100"`
    } `yaml:"workspace" mapstructure:"workspace"`
    
    Output struct {
        Format    string `yaml:"format" mapstructure:"format" validate:"oneof=table json yaml"`
        ColorMode string `yaml:"color_mode" mapstructure:"color_mode" validate:"oneof=auto always never"`
        Width     int    `yaml:"width" mapstructure:"width" validate:"min=40"`
    } `yaml:"output" mapstructure:"output"`
    
    Logging struct {
        Level    string `yaml:"level" mapstructure:"level" validate:"oneof=debug info warn error"`
        FilePath string `yaml:"file_path" mapstructure:"file_path"`
    } `yaml:"logging" mapstructure:"logging"`
}
```

### 기본값 정의
```go
const (
    DefaultClaudeModel       = "claude-3-opus"
    DefaultClaudeTemperature = 0.7
    DefaultClaudeTimeout     = 30
    DefaultOutputFormat      = "table"
    DefaultColorMode         = "auto"
    DefaultOutputWidth       = 120
    DefaultLogLevel          = "info"
    DefaultMaxProjects       = 10
)

func DefaultConfig() *Config {
    homeDir, _ := os.UserHomeDir()
    
    return &Config{
        Claude: struct {
            APIKey      string  `yaml:"api_key" mapstructure:"api_key" validate:"required"`
            Model       string  `yaml:"model" mapstructure:"model" validate:"required"`
            Temperature float64 `yaml:"temperature" mapstructure:"temperature" validate:"min=0,max=1"`
            Timeout     int     `yaml:"timeout" mapstructure:"timeout" validate:"min=1"`
        }{
            Model:       DefaultClaudeModel,
            Temperature: DefaultClaudeTemperature,
            Timeout:     DefaultClaudeTimeout,
        },
        // ... 기타 기본값
    }
}
```

### 환경 변수 매핑
- `AICLI_CLAUDE_API_KEY` → `claude.api_key`
- `AICLI_CLAUDE_MODEL` → `claude.model`
- `AICLI_CLAUDE_TEMPERATURE` → `claude.temperature`
- `AICLI_WORKSPACE_DEFAULT_PATH` → `workspace.default_path`
- `AICLI_OUTPUT_FORMAT` → `output.format`
- `AICLI_OUTPUT_COLOR_MODE` → `output.color_mode`
- `AICLI_LOG_LEVEL` → `logging.level`

### 검증 규칙
- **API Key**: 필수 값, 최소 길이 20자
- **Model**: 허용된 모델 목록 검증
- **Temperature**: 0.0-1.0 범위
- **경로**: 존재하는 디렉토리 확인
- **열거형**: 정의된 값 목록 내에서만 허용

## Output Log
*(This section is populated as work progresses on the task)*