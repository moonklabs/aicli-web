---
task_id: T07_S03
sprint_sequence_id: S03_M02
status: open
complexity: Low
last_updated: 2025-07-21T16:00:00Z
---

# Task: Data Validation and Constraints

## Description
비즈니스 규칙을 적용하는 데이터 검증 시스템을 구현합니다. 데이터베이스 레벨과 애플리케이션 레벨의 검증을 조합하여 데이터 무결성을 보장합니다.

## Goal / Objectives
- 모델별 검증 규칙 정의 및 구현
- 커스텀 검증자 시스템 구축
- 데이터베이스 제약조건과 동기화
- 검증 에러 메시지 표준화
- 국제화(i18n) 지원 준비

## Acceptance Criteria
- [ ] 모든 모델에 검증 규칙 적용
- [ ] 커스텀 검증 함수 등록 가능
- [ ] 검증 실패 시 상세한 에러 정보 제공
- [ ] 데이터베이스 제약조건과 일치
- [ ] 검증 규칙 문서화 완료

## Subtasks
- [ ] 검증 프레임워크 선택 및 통합
- [ ] 모델별 검증 규칙 정의
- [ ] 커스텀 검증자 구현
- [ ] 검증 에러 타입 및 포맷 정의
- [ ] 검증 미들웨어 구현
- [ ] 검증 규칙 문서 작성

## 기술 가이드

### 주요 인터페이스 및 통합 지점
- `internal/models/*.go` - 각 모델에 검증 태그 추가
- 검증 라이브러리: `github.com/go-playground/validator/v10`
- Gin 프레임워크의 기본 검증과 통합

### 검증 규칙 정의
```go
// internal/models/workspace.go 업데이트
type Workspace struct {
    ID          string `validate:"required,uuid"`
    Name        string `validate:"required,min=1,max=100,no_special_chars"`
    ProjectPath string `validate:"required,dir_exists"`
    OwnerID     string `validate:"required,uuid"`
    Status      WorkspaceStatus `validate:"required,workspace_status"`
    // ...
}

// 커스텀 검증자
func init() {
    validate.RegisterValidation("no_special_chars", validateNoSpecialChars)
    validate.RegisterValidation("dir_exists", validateDirExists)
    validate.RegisterValidation("workspace_status", validateWorkspaceStatus)
}
```

### 검증 에러 구조
```go
// internal/validation/errors.go
type ValidationError struct {
    Field   string      `json:"field"`
    Value   interface{} `json:"value,omitempty"`
    Tag     string      `json:"tag"`
    Message string      `json:"message"`
}

type ValidationErrors struct {
    Errors []ValidationError `json:"errors"`
}

func (e ValidationErrors) Error() string {
    // 에러 메시지 포맷팅
}
```

### 비즈니스 규칙 검증
```go
// internal/validation/rules.go
type BusinessValidator interface {
    ValidateCreate(ctx context.Context, model interface{}) error
    ValidateUpdate(ctx context.Context, model interface{}) error
    ValidateDelete(ctx context.Context, id string) error
}

// 워크스페이스 비즈니스 검증
type WorkspaceValidator struct {
    storage storage.WorkspaceStorage
}

func (v *WorkspaceValidator) ValidateCreate(ctx context.Context, ws *models.Workspace) error {
    // 중복 이름 체크
    exists, err := v.storage.ExistsByName(ctx, ws.OwnerID, ws.Name)
    if exists {
        return ErrDuplicateWorkspaceName
    }
    
    // 프로젝트 경로 접근 권한 체크
    // 최대 워크스페이스 수 제한 체크
    // 등...
}
```

## 구현 노트

### 검증 계층
1. **구조체 태그 검증**: 기본 형식 검증
2. **비즈니스 규칙 검증**: 도메인 로직 검증
3. **데이터베이스 제약조건**: 최종 안전장치

### 검증 미들웨어
```go
// internal/middleware/validation.go
func ValidationMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 바인딩 에러를 ValidationErrors로 변환
        c.Next()
        
        if len(c.Errors) > 0 {
            err := c.Errors[0]
            if ve, ok := err.Err.(validator.ValidationErrors); ok {
                c.JSON(400, formatValidationErrors(ve))
                c.Abort()
            }
        }
    }
}
```

### 에러 메시지 국제화 준비
```go
// 향후 i18n 지원을 위한 메시지 키 사용
const (
    MsgFieldRequired     = "validation.field.required"
    MsgFieldTooLong      = "validation.field.too_long"
    MsgDuplicateName     = "validation.duplicate.name"
    MsgInvalidUUID       = "validation.invalid.uuid"
)
```

## Output Log
*(작업 진행 시 업데이트)*