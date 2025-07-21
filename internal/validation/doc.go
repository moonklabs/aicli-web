// Package validation은 AICLI 웹 애플리케이션의 종합적인 데이터 검증 시스템을 제공합니다.
//
// 이 패키지는 다음과 같은 검증 계층을 구현합니다:
//
// 1. 구조체 태그 검증 (Struct Tag Validation)
//    - go-playground/validator/v10 기반
//    - 기본 형식 검증 (required, min, max, email, uuid 등)
//    - 커스텀 검증 함수 (workspace_status, project_status, claude_api_key 등)
//
// 2. 비즈니스 규칙 검증 (Business Rules Validation)
//    - 도메인별 특화 검증 로직
//    - 데이터베이스 제약조건과 연동
//    - 리소스 제한 및 권한 확인
//
// 3. 경로 안전성 검증 (Path Safety Validation)
//    - 디렉토리 존재 여부 및 권한 확인
//    - 위험한 경로 패턴 차단
//    - 워크스페이스 경계 검증
//
// 4. 명령어 보안 검증 (Command Security Validation)
//    - 위험한 명령어 패턴 감지
//    - 시스템 보호 명령어 차단
//
// 5. 국제화 지원 (Internationalization Support)
//    - 다국어 에러 메시지 (한국어, 영어)
//    - Accept-Language 헤더 기반 자동 언어 감지
//    - 필드명 번역
//
// 사용 예제:
//
// 기본 구조체 검증:
//
//   type CreateWorkspaceRequest struct {
//       Name        string `validate:"required,min=1,max=100,no_special_chars"`
//       ProjectPath string `validate:"required,safepath"`
//       ClaudeKey   string `validate:"omitempty,claude_api_key"`
//   }
//
//   func CreateWorkspace(req CreateWorkspaceRequest) error {
//       if err := validation.Validate(req); err != nil {
//           return err
//       }
//       // 비즈니스 로직 계속...
//   }
//
// Gin 미들웨어와 함께 사용:
//
//   r := gin.Default()
//   r.Use(validation.ValidationMiddleware())
//
//   r.POST("/workspaces", func(c *gin.Context) {
//       var req CreateWorkspaceRequest
//       if !validation.ValidateRequestBody(c, &req) {
//           return
//       }
//       if !validation.ValidateBusinessRules(c, "create", &req) {
//           return
//       }
//       // 비즈니스 로직 실행...
//   })
//
// 비즈니스 검증자 설정:
//
//   validation.SetupBusinessValidators(
//       workspaceStorage,
//       projectStorage,
//       sessionStorage,
//       taskStorage,
//   )
//
// 언어 설정:
//
//   validation.SetLanguage(validation.LanguageEnglish)
//   validation.T(validation.MsgFieldRequired, "name") // "The name field is required"
//
// 검증 에러 처리:
//
//   err := validation.Validate(model)
//   if err != nil {
//       if validationErrors, ok := err.(validation.ValidationErrors); ok {
//           // 구조체 검증 에러 처리
//           for _, fieldErr := range validationErrors.Errors {
//               fmt.Printf("Field: %s, Message: %s\n", fieldErr.Field, fieldErr.Message)
//           }
//       } else if businessErr, ok := err.(validation.BusinessValidationError); ok {
//           // 비즈니스 규칙 에러 처리
//           fmt.Printf("Code: %s, Message: %s\n", businessErr.Code, businessErr.Message)
//       }
//   }
//
// 경로 검증:
//
//   err := validation.ValidatePathWithOptions("/path/to/project", validation.PathValidationOptions{
//       MustExist: true,
//       MustBeDir: true,
//       Writable:  true,
//       MaxDepth:  10,
//   })
//
// 이 패키지는 다음과 같은 아키텍처 원칙을 따릅니다:
//
// - 계층적 검증: 기본 → 비즈니스 → 보안
// - 확장성: 새로운 검증 규칙 쉽게 추가 가능
// - 국제화: 다국어 지원으로 글로벌 서비스 준비
// - 성능: 효율적인 검증으로 응답 시간 최소화
// - 보안: 다중 보안 검사로 시스템 보호
//
// 패키지 초기화는 자동으로 이루어지지만, 명시적으로 초기화하려면:
//
//   validation.InitializeValidation()
//
package validation