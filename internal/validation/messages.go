package validation

import (
	"fmt"
	"strings"
)

// MessageKey 메시지 키 타입
type MessageKey string

// 검증 메시지 키 상수
const (
	// 기본 검증 메시지
	MsgFieldRequired     MessageKey = "validation.field.required"
	MsgFieldTooShort     MessageKey = "validation.field.too_short"
	MsgFieldTooLong      MessageKey = "validation.field.too_long"
	MsgFieldInvalidEmail MessageKey = "validation.field.invalid_email"
	MsgFieldInvalidUUID  MessageKey = "validation.field.invalid_uuid"
	MsgFieldInvalidURL   MessageKey = "validation.field.invalid_url"
	MsgFieldInvalidNumber MessageKey = "validation.field.invalid_number"
	MsgFieldInvalidDate   MessageKey = "validation.field.invalid_date"
	
	// 경로 검증 메시지
	MsgPathNotExists     MessageKey = "validation.path.not_exists"
	MsgPathNotDirectory  MessageKey = "validation.path.not_directory"
	MsgPathNotFile       MessageKey = "validation.path.not_file"
	MsgPathNotAccessible MessageKey = "validation.path.not_accessible"
	MsgPathNotWritable   MessageKey = "validation.path.not_writable"
	MsgPathNotReadable   MessageKey = "validation.path.not_readable"
	MsgPathDangerous     MessageKey = "validation.path.dangerous"
	
	// 상태 검증 메시지
	MsgInvalidWorkspaceStatus MessageKey = "validation.status.invalid_workspace"
	MsgInvalidProjectStatus   MessageKey = "validation.status.invalid_project"
	MsgInvalidSessionStatus   MessageKey = "validation.status.invalid_session"
	MsgInvalidTaskStatus      MessageKey = "validation.status.invalid_task"
	
	// 비즈니스 규칙 메시지
	MsgDuplicateName        MessageKey = "validation.business.duplicate_name"
	MsgResourceNotFound     MessageKey = "validation.business.resource_not_found"
	MsgResourceLimit        MessageKey = "validation.business.resource_limit"
	MsgPermissionDenied     MessageKey = "validation.business.permission_denied"
	MsgDependencyExists     MessageKey = "validation.business.dependency_exists"
	MsgInvalidConfiguration MessageKey = "validation.business.invalid_configuration"
	
	// API 키 검증 메시지
	MsgInvalidClaudeAPIKey MessageKey = "validation.api_key.invalid_claude"
	MsgAPIKeyTooShort     MessageKey = "validation.api_key.too_short"
	MsgAPIKeyTooLong      MessageKey = "validation.api_key.too_long"
	
	// 명령어 검증 메시지
	MsgDangerousCommand MessageKey = "validation.command.dangerous"
	MsgCommandTooLong   MessageKey = "validation.command.too_long"
)

// Language 지원 언어
type Language string

const (
	LanguageKorean  Language = "ko"
	LanguageEnglish Language = "en"
)

// MessageTranslator 메시지 번역기 인터페이스
type MessageTranslator interface {
	Translate(key MessageKey, lang Language, params ...interface{}) string
	SetLanguage(lang Language)
	GetLanguage() Language
}

// DefaultMessageTranslator 기본 메시지 번역기
type DefaultMessageTranslator struct {
	currentLang Language
	messages    map[Language]map[MessageKey]string
}

// NewDefaultMessageTranslator 새로운 기본 메시지 번역기 생성
func NewDefaultMessageTranslator() *DefaultMessageTranslator {
	translator := &DefaultMessageTranslator{
		currentLang: LanguageKorean,
		messages:    make(map[Language]map[MessageKey]string),
	}
	
	translator.loadMessages()
	return translator
}

// loadMessages 메시지 로드
func (t *DefaultMessageTranslator) loadMessages() {
	// 한국어 메시지
	t.messages[LanguageKorean] = map[MessageKey]string{
		// 기본 검증 메시지
		MsgFieldRequired:     "%s 필드는 필수입니다",
		MsgFieldTooShort:    "%s 필드는 최소 %s자 이상이어야 합니다",
		MsgFieldTooLong:     "%s 필드는 최대 %s자 이하여야 합니다",
		MsgFieldInvalidEmail: "%s 필드는 유효한 이메일 주소여야 합니다",
		MsgFieldInvalidUUID:  "%s 필드는 유효한 UUID여야 합니다",
		MsgFieldInvalidURL:   "%s 필드는 유효한 URL이어야 합니다",
		MsgFieldInvalidNumber: "%s 필드는 유효한 숫자여야 합니다",
		MsgFieldInvalidDate:   "%s 필드는 유효한 날짜여야 합니다",
		
		// 경로 검증 메시지
		MsgPathNotExists:     "%s 경로가 존재하지 않습니다",
		MsgPathNotDirectory:  "%s 경로는 디렉토리여야 합니다",
		MsgPathNotFile:       "%s 경로는 파일이어야 합니다",
		MsgPathNotAccessible: "%s 경로에 접근할 수 없습니다",
		MsgPathNotWritable:   "%s 경로에 쓰기 권한이 없습니다",
		MsgPathNotReadable:   "%s 경로에 읽기 권한이 없습니다",
		MsgPathDangerous:     "%s 경로에 위험한 문자가 포함되어 있습니다",
		
		// 상태 검증 메시지
		MsgInvalidWorkspaceStatus: "%s 필드는 유효한 워크스페이스 상태여야 합니다 (active, inactive, archived)",
		MsgInvalidProjectStatus:   "%s 필드는 유효한 프로젝트 상태여야 합니다 (active, inactive, archived)",
		MsgInvalidSessionStatus:   "%s 필드는 유효한 세션 상태여야 합니다 (pending, active, idle, ending, ended, error)",
		MsgInvalidTaskStatus:      "%s 필드는 유효한 태스크 상태여야 합니다 (pending, running, completed, failed, cancelled)",
		
		// 비즈니스 규칙 메시지
		MsgDuplicateName:        "%s 이름이 이미 존재합니다",
		MsgResourceNotFound:     "%s을(를) 찾을 수 없습니다",
		MsgResourceLimit:        "%s 최대 개수를 초과했습니다",
		MsgPermissionDenied:     "%s에 대한 권한이 없습니다",
		MsgDependencyExists:     "%s에 종속된 항목이 존재합니다",
		MsgInvalidConfiguration: "%s 설정이 올바르지 않습니다",
		
		// API 키 검증 메시지
		MsgInvalidClaudeAPIKey: "Claude API 키 형식이 올바르지 않습니다 (sk-ant-api03-으로 시작해야 함)",
		MsgAPIKeyTooShort:     "API 키가 너무 짧습니다 (최소 %s자)",
		MsgAPIKeyTooLong:      "API 키가 너무 깁니다 (최대 %s자)",
		
		// 명령어 검증 메시지
		MsgDangerousCommand: "위험한 명령어가 감지되었습니다: %s",
		MsgCommandTooLong:   "명령어가 너무 깁니다 (최대 %s자)",
	}
	
	// 영어 메시지
	t.messages[LanguageEnglish] = map[MessageKey]string{
		// 기본 검증 메시지
		MsgFieldRequired:     "The %s field is required",
		MsgFieldTooShort:    "The %s field must be at least %s characters",
		MsgFieldTooLong:     "The %s field must not exceed %s characters",
		MsgFieldInvalidEmail: "The %s field must be a valid email address",
		MsgFieldInvalidUUID:  "The %s field must be a valid UUID",
		MsgFieldInvalidURL:   "The %s field must be a valid URL",
		MsgFieldInvalidNumber: "The %s field must be a valid number",
		MsgFieldInvalidDate:   "The %s field must be a valid date",
		
		// 경로 검증 메시지
		MsgPathNotExists:     "The path %s does not exist",
		MsgPathNotDirectory:  "The path %s must be a directory",
		MsgPathNotFile:       "The path %s must be a file",
		MsgPathNotAccessible: "Cannot access the path %s",
		MsgPathNotWritable:   "No write permission for path %s",
		MsgPathNotReadable:   "No read permission for path %s",
		MsgPathDangerous:     "The path %s contains dangerous characters",
		
		// 상태 검증 메시지
		MsgInvalidWorkspaceStatus: "The %s field must be a valid workspace status (active, inactive, archived)",
		MsgInvalidProjectStatus:   "The %s field must be a valid project status (active, inactive, archived)",
		MsgInvalidSessionStatus:   "The %s field must be a valid session status (pending, active, idle, ending, ended, error)",
		MsgInvalidTaskStatus:      "The %s field must be a valid task status (pending, running, completed, failed, cancelled)",
		
		// 비즈니스 규칙 메시지
		MsgDuplicateName:        "The %s name already exists",
		MsgResourceNotFound:     "Cannot find %s",
		MsgResourceLimit:        "Maximum number of %s exceeded",
		MsgPermissionDenied:     "No permission for %s",
		MsgDependencyExists:     "Dependencies exist for %s",
		MsgInvalidConfiguration: "Invalid %s configuration",
		
		// API 키 검증 메시지
		MsgInvalidClaudeAPIKey: "Invalid Claude API key format (must start with sk-ant-api03-)",
		MsgAPIKeyTooShort:     "API key too short (minimum %s characters)",
		MsgAPIKeyTooLong:      "API key too long (maximum %s characters)",
		
		// 명령어 검증 메시지
		MsgDangerousCommand: "Dangerous command detected: %s",
		MsgCommandTooLong:   "Command too long (maximum %s characters)",
	}
}

// Translate 메시지 번역
func (t *DefaultMessageTranslator) Translate(key MessageKey, lang Language, params ...interface{}) string {
	// 언어별 메시지 맵 가져오기
	langMessages, exists := t.messages[lang]
	if !exists {
		// 지원하지 않는 언어인 경우 한국어로 fallback
		langMessages = t.messages[LanguageKorean]
	}
	
	// 메시지 템플릿 가져오기
	template, exists := langMessages[key]
	if !exists {
		// 메시지가 없는 경우 기본 메시지 반환
		return fmt.Sprintf("Validation failed for key: %s", string(key))
	}
	
	// 파라미터가 있으면 포맷팅
	if len(params) > 0 {
		return fmt.Sprintf(template, params...)
	}
	
	return template
}

// SetLanguage 언어 설정
func (t *DefaultMessageTranslator) SetLanguage(lang Language) {
	t.currentLang = lang
}

// GetLanguage 현재 언어 반환
func (t *DefaultMessageTranslator) GetLanguage() Language {
	return t.currentLang
}

// Global translator instance
var GlobalTranslator MessageTranslator

// InitializeTranslator 글로벌 번역기 초기화
func InitializeTranslator() {
	GlobalTranslator = NewDefaultMessageTranslator()
}

// T 번역 헬퍼 함수
func T(key MessageKey, params ...interface{}) string {
	if GlobalTranslator == nil {
		InitializeTranslator()
	}
	return GlobalTranslator.Translate(key, GlobalTranslator.GetLanguage(), params...)
}

// TL 언어 지정 번역 헬퍼 함수
func TL(key MessageKey, lang Language, params ...interface{}) string {
	if GlobalTranslator == nil {
		InitializeTranslator()
	}
	return GlobalTranslator.Translate(key, lang, params...)
}

// GetFieldDisplayName 필드 표시명 가져오기
func GetFieldDisplayName(fieldName string, lang Language) string {
	displayNames := map[Language]map[string]string{
		LanguageKorean: {
			"name":          "이름",
			"project_path":  "프로젝트 경로",
			"path":          "경로",
			"claude_key":    "Claude API 키",
			"status":        "상태",
			"owner_id":      "소유자 ID",
			"workspace_id":  "워크스페이스 ID",
			"project_id":    "프로젝트 ID",
			"session_id":    "세션 ID",
			"command":       "명령어",
			"description":   "설명",
			"git_url":       "Git URL",
			"git_branch":    "Git 브랜치",
			"language":      "언어",
			"email":         "이메일",
			"password":      "비밀번호",
			"id":            "ID",
		},
		LanguageEnglish: {
			"name":          "name",
			"project_path":  "project path",
			"path":          "path",
			"claude_key":    "Claude API key",
			"status":        "status",
			"owner_id":      "owner ID",
			"workspace_id":  "workspace ID",
			"project_id":    "project ID",
			"session_id":    "session ID",
			"command":       "command",
			"description":   "description",
			"git_url":       "Git URL",
			"git_branch":    "Git branch",
			"language":      "language",
			"email":         "email",
			"password":      "password",
			"id":            "ID",
		},
	}
	
	if langMap, exists := displayNames[lang]; exists {
		if displayName, exists := langMap[fieldName]; exists {
			return displayName
		}
	}
	
	// fallback: 원본 필드명 반환
	return fieldName
}

// UpdateTranslatedFieldError 번역된 필드 에러 업데이트
func UpdateTranslatedFieldError(fe ValidationError, lang Language) ValidationError {
	// 필드명 번역
	fe.Field = GetFieldDisplayName(fe.Field, lang)
	
	// 메시지 번역
	switch fe.Tag {
	case "required":
		fe.Message = TL(MsgFieldRequired, lang, fe.Field)
	case "min":
		fe.Message = TL(MsgFieldTooShort, lang, fe.Field, fe.Param)
	case "max":
		fe.Message = TL(MsgFieldTooLong, lang, fe.Field, fe.Param)
	case "email":
		fe.Message = TL(MsgFieldInvalidEmail, lang, fe.Field)
	case "uuid":
		fe.Message = TL(MsgFieldInvalidUUID, lang, fe.Field)
	case "url":
		fe.Message = TL(MsgFieldInvalidURL, lang, fe.Field)
	case "dir":
		fe.Message = TL(MsgPathNotDirectory, lang, fe.Field)
	case "safepath":
		fe.Message = TL(MsgPathDangerous, lang, fe.Field)
	case "workspace_status":
		fe.Message = TL(MsgInvalidWorkspaceStatus, lang, fe.Field)
	case "project_status":
		fe.Message = TL(MsgInvalidProjectStatus, lang, fe.Field)
	case "session_status":
		fe.Message = TL(MsgInvalidSessionStatus, lang, fe.Field)
	case "task_status":
		fe.Message = TL(MsgInvalidTaskStatus, lang, fe.Field)
	case "claude_api_key":
		fe.Message = TL(MsgInvalidClaudeAPIKey, lang)
	default:
		// 기본 메시지 유지
	}
	
	return fe
}

// TranslateValidationErrors 검증 에러들을 특정 언어로 번역
func TranslateValidationErrors(errors ValidationErrors, lang Language) ValidationErrors {
	translatedErrors := ValidationErrors{
		Model:  errors.Model,
		Errors: make([]ValidationError, len(errors.Errors)),
	}
	
	for i, err := range errors.Errors {
		translatedErrors.Errors[i] = UpdateTranslatedFieldError(err, lang)
	}
	
	return translatedErrors
}

// GetLanguageFromContext 컨텍스트에서 언어 정보 추출
func GetLanguageFromContext(acceptLanguage string) Language {
	// Accept-Language 헤더 파싱
	if acceptLanguage == "" {
		return LanguageKorean
	}
	
	// 간단한 언어 감지 (더 정교한 구현 가능)
	acceptLanguage = strings.ToLower(acceptLanguage)
	if strings.Contains(acceptLanguage, "en") {
		return LanguageEnglish
	}
	
	return LanguageKorean
}

// ErrorCodeTranslation 에러 코드별 번역 메시지
var ErrorCodeTranslation = map[Language]map[string]string{
	LanguageKorean: {
		ErrCodeDuplicateName:        "중복된 이름입니다",
		ErrCodeInvalidStatus:        "유효하지 않은 상태입니다",
		ErrCodeResourceNotFound:     "리소스를 찾을 수 없습니다",
		ErrCodePermissionDenied:     "권한이 없습니다",
		ErrCodeResourceLimit:        "리소스 제한을 초과했습니다",
		ErrCodeDependencyExists:     "종속성이 존재합니다",
		ErrCodeInvalidConfiguration: "유효하지 않은 설정입니다",
		ErrCodePathNotAccessible:    "경로에 접근할 수 없습니다",
	},
	LanguageEnglish: {
		ErrCodeDuplicateName:        "Duplicate name",
		ErrCodeInvalidStatus:        "Invalid status",
		ErrCodeResourceNotFound:     "Resource not found",
		ErrCodePermissionDenied:     "Permission denied",
		ErrCodeResourceLimit:        "Resource limit exceeded",
		ErrCodeDependencyExists:     "Dependencies exist",
		ErrCodeInvalidConfiguration: "Invalid configuration",
		ErrCodePathNotAccessible:    "Path not accessible",
	},
}

// TranslateBusinessError 비즈니스 에러 번역
func TranslateBusinessError(err BusinessValidationError, lang Language) BusinessValidationError {
	if translations, exists := ErrorCodeTranslation[lang]; exists {
		if translation, exists := translations[err.Code]; exists {
			err.Message = translation
		}
	}
	
	// 필드명도 번역
	if err.Field != "" {
		err.Field = GetFieldDisplayName(err.Field, lang)
	}
	
	return err
}