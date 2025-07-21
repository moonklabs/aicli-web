package validation

import (
	"sync"

	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/storage"
)

// Package level variables
var (
	once                sync.Once
	validationService   *ValidationService
	validationManager   *ValidationManager
	messageTranslator   MessageTranslator
)

// InitializeValidation 검증 시스템 초기화
func InitializeValidation() {
	once.Do(func() {
		// 메시지 번역기 초기화
		messageTranslator = NewDefaultMessageTranslator()
		GlobalTranslator = messageTranslator
		
		// 검증 관리자 초기화
		validationManager = GetGinValidator()
		
		// 검증 서비스 초기화
		validationService = NewValidationService()
	})
}

// SetupBusinessValidators 비즈니스 검증자들 설정
func SetupBusinessValidators(
	workspaceStorage storage.WorkspaceStorage,
	projectStorage storage.ProjectStorage,
	sessionStorage storage.SessionStorage,
	taskStorage storage.TaskStorage,
) {
	if validationService == nil {
		InitializeValidation()
	}

	// 워크스페이스 비즈니스 검증자 등록
	workspaceValidator := NewWorkspaceBusinessValidator(workspaceStorage)
	validationService.RegisterBusinessValidator(&models.Workspace{}, workspaceValidator)
	validationService.RegisterBusinessValidator(&models.CreateWorkspaceRequest{}, workspaceValidator)
	validationService.RegisterBusinessValidator(&models.UpdateWorkspaceRequest{}, workspaceValidator)

	// 프로젝트 비즈니스 검증자 등록
	projectValidator := NewProjectBusinessValidator(projectStorage, workspaceStorage)
	validationService.RegisterBusinessValidator(&models.Project{}, projectValidator)

	// 세션 비즈니스 검증자 등록
	sessionValidator := NewSessionBusinessValidator(sessionStorage, projectStorage)
	validationService.RegisterBusinessValidator(&models.SessionCreateRequest{}, sessionValidator)

	// 태스크 비즈니스 검증자 등록
	taskValidator := NewTaskBusinessValidator(taskStorage, sessionStorage)
	validationService.RegisterBusinessValidator(&models.TaskCreateRequest{}, taskValidator)
	validationService.RegisterBusinessValidator(&models.TaskUpdateRequest{}, taskValidator)
}

// GetValidationService 검증 서비스 반환
func GetValidationService() *ValidationService {
	if validationService == nil {
		InitializeValidation()
	}
	return validationService
}

// GetValidationManager 검증 관리자 반환
func GetValidationManager() *ValidationManager {
	if validationManager == nil {
		InitializeValidation()
	}
	return validationManager
}

// GetMessageTranslator 메시지 번역기 반환
func GetMessageTranslator() MessageTranslator {
	if messageTranslator == nil {
		InitializeValidation()
	}
	return messageTranslator
}

// ValidationConfig 검증 설정
type ValidationConfig struct {
	// 리소스 제한 설정
	MaxWorkspacesPerUser      int
	MaxProjectsPerWorkspace   int
	MaxSessionsPerProject     int
	MaxTasksPerSession        int
	
	// 경로 검증 설정
	MaxPathDepth             int
	AllowedPathPrefixes      []string
	ForbiddenPathPrefixes    []string
	
	// API 키 설정
	ClaudeAPIKeyMinLength    int
	ClaudeAPIKeyMaxLength    int
	
	// 명령어 보안 설정
	MaxCommandLength         int
	ForbiddenCommandPatterns []string
	
	// 국제화 설정
	DefaultLanguage          Language
	SupportedLanguages       []Language
}

// DefaultValidationConfig 기본 검증 설정
func DefaultValidationConfig() *ValidationConfig {
	return &ValidationConfig{
		MaxWorkspacesPerUser:    20,
		MaxProjectsPerWorkspace: 50,
		MaxSessionsPerProject:   3,
		MaxTasksPerSession:     5,
		
		MaxPathDepth: 10,
		AllowedPathPrefixes: []string{
			"/home",
			"/tmp",
			"/var/tmp",
			"/opt",
		},
		ForbiddenPathPrefixes: []string{
			"/",
			"/bin",
			"/boot",
			"/dev",
			"/etc",
			"/lib",
			"/proc",
			"/root",
			"/sys",
			"/usr",
			"/var",
		},
		
		ClaudeAPIKeyMinLength: 50,
		ClaudeAPIKeyMaxLength: 200,
		
		MaxCommandLength: 10000,
		ForbiddenCommandPatterns: []string{
			"rm -rf /",
			"dd if=",
			"mkfs",
			"format",
			"fdisk",
			"> /dev/",
			"shutdown",
			"reboot",
			"init 0",
			"init 6",
			"halt",
			"poweroff",
		},
		
		DefaultLanguage: LanguageKorean,
		SupportedLanguages: []Language{
			LanguageKorean,
			LanguageEnglish,
		},
	}
}

// ApplyValidationConfig 검증 설정 적용
func ApplyValidationConfig(config *ValidationConfig) {
	if messageTranslator == nil {
		InitializeValidation()
	}
	
	// 기본 언어 설정
	messageTranslator.SetLanguage(config.DefaultLanguage)
}

// ValidationStats 검증 통계
type ValidationStats struct {
	TotalValidations      int64
	SuccessfulValidations int64
	FailedValidations     int64
	BusinessRuleViolations int64
	MostCommonErrors      map[string]int64
}

// GetValidationStats 검증 통계 반환 (향후 구현)
func GetValidationStats() *ValidationStats {
	return &ValidationStats{
		// TODO: 실제 통계 수집 구현
		TotalValidations: 0,
		SuccessfulValidations: 0,
		FailedValidations: 0,
		BusinessRuleViolations: 0,
		MostCommonErrors: make(map[string]int64),
	}
}

// ResetValidationStats 검증 통계 초기화 (향후 구현)
func ResetValidationStats() {
	// TODO: 통계 초기화 구현
}

// IsValidationInitialized 검증 시스템 초기화 여부 확인
func IsValidationInitialized() bool {
	return validationService != nil && validationManager != nil && messageTranslator != nil
}

// SetLanguage 전역 언어 설정
func SetLanguage(lang Language) {
	if messageTranslator != nil {
		messageTranslator.SetLanguage(lang)
	}
}

// GetCurrentLanguage 현재 언어 반환
func GetCurrentLanguage() Language {
	if messageTranslator != nil {
		return messageTranslator.GetLanguage()
	}
	return LanguageKorean
}