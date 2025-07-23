package security

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aicli/aicli-web/internal/models"
)

// SecurityPolicy 보안 정책 모델
type SecurityPolicy struct {
	models.Base
	Name            string                 `json:"name" db:"name"`
	Description     string                 `json:"description" db:"description"`
	Version         string                 `json:"version" db:"version"`
	IsActive        bool                   `json:"is_active" db:"is_active"`
	IsTemplate      bool                   `json:"is_template" db:"is_template"`
	ParentID        *string                `json:"parent_id" db:"parent_id"` // 템플릿으로부터 생성된 경우
	ConfigData      map[string]interface{} `json:"config_data" db:"config_data"` // JSON으로 저장
	Priority        int                    `json:"priority" db:"priority"`       // 우선순위 (높은 값이 우선)
	Category        string                 `json:"category" db:"category"`       // authentication, authorization, rate_limiting 등
	Tags            []string               `json:"tags" db:"tags"`               // JSON 배열로 저장
	EffectiveFrom   time.Time              `json:"effective_from" db:"effective_from"`
	EffectiveUntil  *time.Time             `json:"effective_until" db:"effective_until"`
	AppliedBy       string                 `json:"applied_by" db:"applied_by"`      // 적용한 관리자 ID
	ValidationRules []ValidationRule       `json:"validation_rules,omitempty"`     // DB에는 저장하지 않음
	AuditLog        []PolicyAuditEntry     `json:"audit_log,omitempty"`            // 별도 테이블
}

// ValidationRule 정책 유효성 검증 규칙
type ValidationRule struct {
	Field    string      `json:"field"`
	Rule     string      `json:"rule"`     // required, range, enum 등
	Value    interface{} `json:"value"`    // 검증 값
	Message  string      `json:"message"`  // 오류 메시지
	Severity string      `json:"severity"` // error, warning, info
}

// PolicyAuditEntry 정책 변경 감사 로그
type PolicyAuditEntry struct {
	models.Base
	PolicyID    string                 `json:"policy_id" db:"policy_id"`
	Action      string                 `json:"action" db:"action"`           // create, update, delete, apply, rollback
	OldVersion  string                 `json:"old_version" db:"old_version"`
	NewVersion  string                 `json:"new_version" db:"new_version"`
	Changes     map[string]interface{} `json:"changes" db:"changes"`         // JSON으로 저장
	AppliedBy   string                 `json:"applied_by" db:"applied_by"`
	Reason      string                 `json:"reason" db:"reason"`
	IPAddress   string                 `json:"ip_address" db:"ip_address"`
	UserAgent   string                 `json:"user_agent" db:"user_agent"`
}

// PolicyTemplate 정책 템플릿
type PolicyTemplate struct {
	models.Base
	Name         string                 `json:"name" db:"name"`
	Description  string                 `json:"description" db:"description"`
	Category     string                 `json:"category" db:"category"`
	ConfigSchema map[string]interface{} `json:"config_schema" db:"config_schema"` // JSON Schema
	DefaultData  map[string]interface{} `json:"default_data" db:"default_data"`   // 기본값
	IsBuiltIn    bool                   `json:"is_built_in" db:"is_built_in"`     // 시스템 내장 여부
	Tags         []string               `json:"tags" db:"tags"`
}

// PolicyManager 정책 관리자
type PolicyManager struct {
	activePolicies map[string]*SecurityPolicy
	templates      map[string]*PolicyTemplate
	mu             sync.RWMutex
	changeNotifiers []PolicyChangeNotifier
	validator      *PolicyValidator
}

// PolicyChangeNotifier 정책 변경 알림 인터페이스
type PolicyChangeNotifier interface {
	OnPolicyChanged(ctx context.Context, policy *SecurityPolicy, action string) error
}

// PolicyValidator 정책 검증기
type PolicyValidator struct {
	rules map[string][]ValidationRule
}

// PolicyService 정책 서비스 인터페이스
type PolicyService interface {
	// 정책 CRUD
	CreatePolicy(ctx context.Context, req *CreatePolicyRequest) (*SecurityPolicy, error)
	GetPolicy(ctx context.Context, id string) (*SecurityPolicy, error)
	UpdatePolicy(ctx context.Context, id string, req *UpdatePolicyRequest) (*SecurityPolicy, error)
	DeletePolicy(ctx context.Context, id string) error
	ListPolicies(ctx context.Context, filter *PolicyFilter) (*models.PaginatedResponse[*SecurityPolicy], error)
	
	// 정책 적용 및 관리
	ApplyPolicy(ctx context.Context, id string) error
	DeactivatePolicy(ctx context.Context, id string) error
	RollbackPolicy(ctx context.Context, id string, toVersion string) error
	GetActivePolicies(ctx context.Context, category string) ([]*SecurityPolicy, error)
	
	// 정책 템플릿
	CreateTemplate(ctx context.Context, req *CreateTemplateRequest) (*PolicyTemplate, error)
	GetTemplate(ctx context.Context, id string) (*PolicyTemplate, error)
	ListTemplates(ctx context.Context) ([]*PolicyTemplate, error)
	CreatePolicyFromTemplate(ctx context.Context, templateID string, req *CreateFromTemplateRequest) (*SecurityPolicy, error)
	
	// 정책 검증
	ValidatePolicy(ctx context.Context, policy *SecurityPolicy) (*ValidationResult, error)
	TestPolicy(ctx context.Context, id string, testData interface{}) (*TestResult, error)
	
	// 감사 및 히스토리
	GetPolicyHistory(ctx context.Context, id string) ([]*PolicyAuditEntry, error)
	GetPolicyAuditLog(ctx context.Context, filter *AuditFilter) (*models.PaginatedResponse[*PolicyAuditEntry], error)
}

// 요청/응답 구조체들

type CreatePolicyRequest struct {
	Name           string                 `json:"name" validate:"required,min=3,max=100"`
	Description    string                 `json:"description" validate:"max=500"`
	Category       string                 `json:"category" validate:"required,oneof=authentication authorization rate_limiting security_headers cors csrf session"`
	ConfigData     map[string]interface{} `json:"config_data" validate:"required"`
	Priority       int                    `json:"priority" validate:"min=1,max=100"`
	Tags           []string               `json:"tags,omitempty"`
	EffectiveFrom  *time.Time             `json:"effective_from,omitempty"`
	EffectiveUntil *time.Time             `json:"effective_until,omitempty"`
	ApplyNow       bool                   `json:"apply_now,omitempty"`
}

type UpdatePolicyRequest struct {
	Name           *string                `json:"name,omitempty" validate:"omitempty,min=3,max=100"`
	Description    *string                `json:"description,omitempty" validate:"omitempty,max=500"`
	ConfigData     map[string]interface{} `json:"config_data,omitempty"`
	Priority       *int                   `json:"priority,omitempty" validate:"omitempty,min=1,max=100"`
	Tags           []string               `json:"tags,omitempty"`
	EffectiveFrom  *time.Time             `json:"effective_from,omitempty"`
	EffectiveUntil *time.Time             `json:"effective_until,omitempty"`
	Reason         string                 `json:"reason,omitempty"`
}

type CreateTemplateRequest struct {
	Name         string                 `json:"name" validate:"required,min=3,max=100"`
	Description  string                 `json:"description" validate:"max=500"`
	Category     string                 `json:"category" validate:"required"`
	ConfigSchema map[string]interface{} `json:"config_schema" validate:"required"`
	DefaultData  map[string]interface{} `json:"default_data,omitempty"`
	Tags         []string               `json:"tags,omitempty"`
}

type CreateFromTemplateRequest struct {
	Name        string                 `json:"name" validate:"required,min=3,max=100"`
	Description string                 `json:"description,omitempty"`
	ConfigData  map[string]interface{} `json:"config_data,omitempty"`
	Priority    int                    `json:"priority" validate:"min=1,max=100"`
	ApplyNow    bool                   `json:"apply_now,omitempty"`
}

type PolicyFilter struct {
	Category  string     `json:"category,omitempty"`
	IsActive  *bool      `json:"is_active,omitempty"`
	Tags      []string   `json:"tags,omitempty"`
	Search    string     `json:"search,omitempty"`
	CreatedBy string     `json:"created_by,omitempty"`
	DateFrom  *time.Time `json:"date_from,omitempty"`
	DateTo    *time.Time `json:"date_to,omitempty"`
	models.PaginationRequest
}

type AuditFilter struct {
	PolicyID string     `json:"policy_id,omitempty"`
	Action   string     `json:"action,omitempty"`
	UserID   string     `json:"user_id,omitempty"`
	DateFrom *time.Time `json:"date_from,omitempty"`
	DateTo   *time.Time `json:"date_to,omitempty"`
	models.PaginationRequest
}

type ValidationResult struct {
	IsValid bool              `json:"is_valid"`
	Errors  []ValidationError `json:"errors,omitempty"`
	Warnings []ValidationError `json:"warnings,omitempty"`
	Score   int               `json:"score"` // 0-100
}

type ValidationError struct {
	Field    string `json:"field"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
	Code     string `json:"code"`
}

type TestResult struct {
	Success       bool                   `json:"success"`
	Results       map[string]interface{} `json:"results"`
	ExecutionTime time.Duration          `json:"execution_time"`
	Errors        []string               `json:"errors,omitempty"`
}

// NewPolicyManager 새로운 정책 관리자 생성
func NewPolicyManager() *PolicyManager {
	return &PolicyManager{
		activePolicies:  make(map[string]*SecurityPolicy),
		templates:       make(map[string]*PolicyTemplate),
		changeNotifiers: make([]PolicyChangeNotifier, 0),
		validator:       NewPolicyValidator(),
	}
}

// RegisterNotifier 정책 변경 알림자 등록
func (pm *PolicyManager) RegisterNotifier(notifier PolicyChangeNotifier) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.changeNotifiers = append(pm.changeNotifiers, notifier)
}

// ApplyPolicyRuntime 런타임에 정책 적용
func (pm *PolicyManager) ApplyPolicyRuntime(ctx context.Context, policy *SecurityPolicy) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	// 정책 유효성 검증
	if result, err := pm.validator.Validate(policy); err != nil || !result.IsValid {
		if err != nil {
			return fmt.Errorf("정책 검증 실패: %w", err)
		}
		return fmt.Errorf("정책이 유효하지 않습니다: %v", result.Errors)
	}
	
	// 기존 같은 카테고리 정책들과 우선순위 체크
	for _, existingPolicy := range pm.activePolicies {
		if existingPolicy.Category == policy.Category && 
		   existingPolicy.Priority >= policy.Priority && 
		   existingPolicy.ID != policy.ID {
			return fmt.Errorf("더 높은 우선순위의 정책이 이미 활성화되어 있습니다: %s", existingPolicy.Name)
		}
	}
	
	// 정책 활성화
	pm.activePolicies[policy.ID] = policy
	
	// 변경 알림 발송
	for _, notifier := range pm.changeNotifiers {
		if err := notifier.OnPolicyChanged(ctx, policy, "apply"); err != nil {
			// 로그만 남기고 계속 진행
			fmt.Printf("정책 변경 알림 실패: %v\n", err)
		}
	}
	
	return nil
}

// GetActivePolicy 활성 정책 조회
func (pm *PolicyManager) GetActivePolicy(category string, key string) (*SecurityPolicy, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	var bestPolicy *SecurityPolicy
	var bestPriority int = -1
	
	for _, policy := range pm.activePolicies {
		if policy.Category == category && policy.IsActive {
			// 현재 시간이 유효 기간 내인지 확인
			now := time.Now()
			if now.Before(policy.EffectiveFrom) {
				continue
			}
			if policy.EffectiveUntil != nil && now.After(*policy.EffectiveUntil) {
				continue
			}
			
			// 더 높은 우선순위인지 확인
			if policy.Priority > bestPriority {
				bestPolicy = policy
				bestPriority = policy.Priority
			}
		}
	}
	
	return bestPolicy, bestPolicy != nil
}

// NewPolicyValidator 새로운 정책 검증기 생성
func NewPolicyValidator() *PolicyValidator {
	return &PolicyValidator{
		rules: make(map[string][]ValidationRule),
	}
}

// Validate 정책 유효성 검증
func (pv *PolicyValidator) Validate(policy *SecurityPolicy) (*ValidationResult, error) {
	result := &ValidationResult{
		IsValid:  true,
		Errors:   []ValidationError{},
		Warnings: []ValidationError{},
		Score:    100,
	}
	
	// 필수 필드 검증
	if policy.Name == "" {
		result.IsValid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:    "name",
			Message:  "정책 이름은 필수입니다",
			Severity: "error",
			Code:     "REQUIRED_FIELD",
		})
	}
	
	if policy.Category == "" {
		result.IsValid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:    "category",
			Message:  "정책 카테고리는 필수입니다",
			Severity: "error",
			Code:     "REQUIRED_FIELD",
		})
	}
	
	// 카테고리별 특화 검증
	if err := pv.validateByCategory(policy, result); err != nil {
		return nil, err
	}
	
	// 우선순위 검증
	if policy.Priority < 1 || policy.Priority > 100 {
		result.IsValid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:    "priority",
			Message:  "우선순위는 1-100 범위여야 합니다",
			Severity: "error",
			Code:     "INVALID_RANGE",
		})
	}
	
	// 유효 기간 검증
	if policy.EffectiveUntil != nil && policy.EffectiveFrom.After(*policy.EffectiveUntil) {
		result.IsValid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:    "effective_until",
			Message:  "종료일은 시작일 이후여야 합니다",
			Severity: "error",
			Code:     "INVALID_DATE_RANGE",
		})
	}
	
	// 점수 계산
	if len(result.Errors) > 0 {
		result.Score = max(0, result.Score-len(result.Errors)*20)
	}
	if len(result.Warnings) > 0 {
		result.Score = max(0, result.Score-len(result.Warnings)*5)
	}
	
	return result, nil
}

// validateByCategory 카테고리별 특화 검증
func (pv *PolicyValidator) validateByCategory(policy *SecurityPolicy, result *ValidationResult) error {
	switch policy.Category {
	case "rate_limiting":
		return pv.validateRateLimitingPolicy(policy, result)
	case "authentication":
		return pv.validateAuthenticationPolicy(policy, result)
	case "authorization":
		return pv.validateAuthorizationPolicy(policy, result)
	case "security_headers":
		return pv.validateSecurityHeadersPolicy(policy, result)
	default:
		result.Warnings = append(result.Warnings, ValidationError{
			Field:    "category",
			Message:  "알려지지 않은 카테고리입니다",
			Severity: "warning",
			Code:     "UNKNOWN_CATEGORY",
		})
	}
	return nil
}

// validateRateLimitingPolicy Rate Limiting 정책 검증
func (pv *PolicyValidator) validateRateLimitingPolicy(policy *SecurityPolicy, result *ValidationResult) error {
	config := policy.ConfigData
	
	// requests_per_second 검증
	if rps, exists := config["requests_per_second"]; exists {
		if rpsFloat, ok := rps.(float64); ok {
			if rpsFloat <= 0 || rpsFloat > 10000 {
				result.Errors = append(result.Errors, ValidationError{
					Field:    "config_data.requests_per_second",
					Message:  "초당 요청 수는 1-10000 범위여야 합니다",
					Severity: "error",
					Code:     "INVALID_RANGE",
				})
				result.IsValid = false
			}
		} else {
			result.Errors = append(result.Errors, ValidationError{
				Field:    "config_data.requests_per_second",
				Message:  "초당 요청 수는 숫자여야 합니다",
				Severity: "error",
				Code:     "INVALID_TYPE",
			})
			result.IsValid = false
		}
	}
	
	return nil
}

// validateAuthenticationPolicy 인증 정책 검증
func (pv *PolicyValidator) validateAuthenticationPolicy(policy *SecurityPolicy, result *ValidationResult) error {
	config := policy.ConfigData
	
	// session_timeout 검증
	if timeout, exists := config["session_timeout"]; exists {
		if timeoutFloat, ok := timeout.(float64); ok {
			if timeoutFloat < 300 || timeoutFloat > 86400 { // 5분 ~ 24시간
				result.Warnings = append(result.Warnings, ValidationError{
					Field:    "config_data.session_timeout",
					Message:  "세션 타임아웃이 권장 범위(5분-24시간)를 벗어납니다",
					Severity: "warning",
					Code:     "OUT_OF_RECOMMENDED_RANGE",
				})
			}
		}
	}
	
	return nil
}

// validateAuthorizationPolicy 권한 정책 검증
func (pv *PolicyValidator) validateAuthorizationPolicy(policy *SecurityPolicy, result *ValidationResult) error {
	// RBAC 관련 설정 검증
	return nil
}

// validateSecurityHeadersPolicy 보안 헤더 정책 검증
func (pv *PolicyValidator) validateSecurityHeadersPolicy(policy *SecurityPolicy, result *ValidationResult) error {
	config := policy.ConfigData
	
	// CSP 검증
	if csp, exists := config["content_security_policy"]; exists {
		if cspStr, ok := csp.(string); ok {
			if len(cspStr) > 4096 {
				result.Warnings = append(result.Warnings, ValidationError{
					Field:    "config_data.content_security_policy",
					Message:  "CSP 헤더가 너무 깁니다 (4KB 초과)",
					Severity: "warning",
					Code:     "TOO_LONG",
				})
			}
		}
	}
	
	return nil
}

// GetBuiltInTemplates 내장 정책 템플릿 조회
func GetBuiltInTemplates() []*PolicyTemplate {
	return []*PolicyTemplate{
		{
			Base:        models.Base{ID: "tpl_rate_limit_basic"},
			Name:        "기본 Rate Limiting",
			Description: "일반적인 API Rate Limiting 정책",
			Category:    "rate_limiting",
			ConfigSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"requests_per_second": map[string]interface{}{
						"type":    "number",
						"minimum": 1,
						"maximum": 10000,
						"default": 100,
					},
					"burst_size": map[string]interface{}{
						"type":    "number",
						"minimum": 1,
						"maximum": 1000,
						"default": 10,
					},
					"block_duration": map[string]interface{}{
						"type":    "number",
						"minimum": 60,
						"maximum": 3600,
						"default": 300,
					},
				},
				"required": []string{"requests_per_second"},
			},
			DefaultData: map[string]interface{}{
				"requests_per_second": 100,
				"burst_size":          10,
				"block_duration":      300,
			},
			IsBuiltIn: true,
			Tags:      []string{"api", "protection", "basic"},
		},
		{
			Base:        models.Base{ID: "tpl_security_headers"},
			Name:        "보안 헤더",
			Description: "표준 보안 헤더 설정",
			Category:    "security_headers",
			ConfigSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"hsts_max_age": map[string]interface{}{
						"type":    "number",
						"default": 31536000,
					},
					"content_security_policy": map[string]interface{}{
						"type":    "string",
						"default": "default-src 'self'",
					},
					"x_frame_options": map[string]interface{}{
						"type":    "string",
						"enum":    []string{"DENY", "SAMEORIGIN"},
						"default": "DENY",
					},
				},
			},
			DefaultData: map[string]interface{}{
				"hsts_max_age":            31536000,
				"content_security_policy": "default-src 'self'",
				"x_frame_options":         "DENY",
				"x_content_type_options":  "nosniff",
				"referrer_policy":         "strict-origin-when-cross-origin",
			},
			IsBuiltIn: true,
			Tags:      []string{"headers", "security", "standard"},
		},
	}
}

// ToResponse SecurityPolicy를 응답용으로 변환
func (sp *SecurityPolicy) ToResponse() *SecurityPolicyResponse {
	return &SecurityPolicyResponse{
		ID:             sp.ID,
		Name:           sp.Name,
		Description:    sp.Description,
		Version:        sp.Version,
		IsActive:       sp.IsActive,
		IsTemplate:     sp.IsTemplate,
		ConfigData:     sp.ConfigData,
		Priority:       sp.Priority,
		Category:       sp.Category,
		Tags:           sp.Tags,
		EffectiveFrom:  sp.EffectiveFrom,
		EffectiveUntil: sp.EffectiveUntil,
		AppliedBy:      sp.AppliedBy,
		CreatedAt:      sp.CreatedAt,
		UpdatedAt:      sp.UpdatedAt,
	}
}

// SecurityPolicyResponse 보안 정책 응답 구조체
type SecurityPolicyResponse struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	Description    string                 `json:"description"`
	Version        string                 `json:"version"`
	IsActive       bool                   `json:"is_active"`
	IsTemplate     bool                   `json:"is_template"`
	ConfigData     map[string]interface{} `json:"config_data"`
	Priority       int                    `json:"priority"`
	Category       string                 `json:"category"`
	Tags           []string               `json:"tags"`
	EffectiveFrom  time.Time              `json:"effective_from"`
	EffectiveUntil *time.Time             `json:"effective_until"`
	AppliedBy      string                 `json:"applied_by"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

// PolicyStats 정책 통계
type PolicyStats struct {
	TotalPolicies     int                    `json:"total_policies"`
	ActivePolicies    int                    `json:"active_policies"`
	PoliciesByCategory map[string]int        `json:"policies_by_category"`
	RecentChanges     []PolicyAuditEntry    `json:"recent_changes"`
	EffectivenesScore int                    `json:"effectiveness_score"`
}

// max 헬퍼 함수
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}