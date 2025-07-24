package services

import (
	"context"
	"fmt"
	"time"

	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/security"
	"github.com/aicli/aicli-web/internal/storage/interfaces"
)

// policyService는 PolicyService 인터페이스의 구현체입니다
type policyService struct {
	storage       interfaces.Storage
	policyManager *security.PolicyManager
}

// NewPolicyService는 새로운 정책 서비스를 생성합니다
func NewPolicyService(storage interfaces.Storage) security.PolicyService {
	return &policyService{
		storage:       storage,
		policyManager: security.NewPolicyManager(),
	}
}

// CreatePolicy는 새로운 보안 정책을 생성합니다
func (ps *policyService) CreatePolicy(ctx context.Context, req *security.CreatePolicyRequest) (*security.SecurityPolicy, error) {
	// 정책명 중복 검사
	existingPolicy := &security.SecurityPolicy{}
	err := ps.storage.GetByField(ctx, "security_policies", "name", req.Name, existingPolicy)
	if err == nil {
		return nil, fmt.Errorf("이미 존재하는 정책명입니다: %s", req.Name)
	}

	// 새 정책 생성
	policy := &security.SecurityPolicy{
		Base: models.Base{
			ID:        generatePolicyID(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Name:           req.Name,
		Description:    req.Description,
		Version:        "1.0.0",
		IsActive:       false,
		IsTemplate:     false,
		ConfigData:     req.ConfigData,
		Priority:       req.Priority,
		Category:       req.Category,
		Tags:           req.Tags,
		EffectiveFrom:  time.Now(),
		EffectiveUntil: req.EffectiveUntil,
	}

	if req.EffectiveFrom != nil {
		policy.EffectiveFrom = *req.EffectiveFrom
	}

	// 정책 유효성 검증
	validator := security.NewPolicyValidator()
	result, err := validator.Validate(policy)
	if err != nil {
		return nil, fmt.Errorf("정책 검증 실패: %w", err)
	}
	if !result.IsValid {
		return nil, fmt.Errorf("정책이 유효하지 않습니다: %v", result.Errors)
	}

	// 데이터베이스에 저장
	err = ps.storage.Create(ctx, "security_policies", policy)
	if err != nil {
		return nil, fmt.Errorf("정책 생성 실패: %w", err)
	}

	// 감사 로그 생성
	ps.createAuditEntry(ctx, policy.ID, "create", "", policy.Version, map[string]interface{}{
		"name":        policy.Name,
		"category":    policy.Category,
		"priority":    policy.Priority,
		"apply_now":   req.ApplyNow,
	}, "", "", "")

	// 즉시 적용 요청 시
	if req.ApplyNow {
		err = ps.ApplyPolicy(ctx, policy.ID)
		if err != nil {
			return nil, fmt.Errorf("정책 적용 실패: %w", err)
		}
	}

	return policy, nil
}

// GetPolicy는 특정 보안 정책을 조회합니다
func (ps *policyService) GetPolicy(ctx context.Context, id string) (*security.SecurityPolicy, error) {
	policy := &security.SecurityPolicy{}
	err := ps.storage.GetByID(ctx, "security_policies", id, policy)
	if err != nil {
		return nil, fmt.Errorf("정책 조회 실패: %w", err)
	}
	return policy, nil
}

// UpdatePolicy는 보안 정책을 업데이트합니다
func (ps *policyService) UpdatePolicy(ctx context.Context, id string, req *security.UpdatePolicyRequest) (*security.SecurityPolicy, error) {
	// 기존 정책 조회
	policy, err := ps.GetPolicy(ctx, id)
	if err != nil {
		return nil, err
	}

	// 변경사항 추적을 위한 이전 데이터 보관
	// TODO: oldData를 활동 로그에 사용
	_ = map[string]interface{}{
		"name":        policy.Name,
		"description": policy.Description,
		"config_data": policy.ConfigData,
		"priority":    policy.Priority,
		"tags":        policy.Tags,
	}

	// 업데이트할 필드 적용
	changes := make(map[string]interface{})
	if req.Name != nil && *req.Name != policy.Name {
		changes["name"] = map[string]interface{}{"old": policy.Name, "new": *req.Name}
		policy.Name = *req.Name
	}
	if req.Description != nil && *req.Description != policy.Description {
		changes["description"] = map[string]interface{}{"old": policy.Description, "new": *req.Description}
		policy.Description = *req.Description
	}
	if req.ConfigData != nil {
		changes["config_data"] = map[string]interface{}{"old": policy.ConfigData, "new": req.ConfigData}
		policy.ConfigData = req.ConfigData
	}
	if req.Priority != nil && *req.Priority != policy.Priority {
		changes["priority"] = map[string]interface{}{"old": policy.Priority, "new": *req.Priority}
		policy.Priority = *req.Priority
	}
	if req.Tags != nil {
		changes["tags"] = map[string]interface{}{"old": policy.Tags, "new": req.Tags}
		policy.Tags = req.Tags
	}
	if req.EffectiveFrom != nil {
		changes["effective_from"] = map[string]interface{}{"old": policy.EffectiveFrom, "new": *req.EffectiveFrom}
		policy.EffectiveFrom = *req.EffectiveFrom
	}
	if req.EffectiveUntil != nil {
		changes["effective_until"] = map[string]interface{}{"old": policy.EffectiveUntil, "new": *req.EffectiveUntil}
		policy.EffectiveUntil = req.EffectiveUntil
	}

	// 버전 업데이트
	oldVersion := policy.Version
	policy.Version = ps.incrementVersion(policy.Version)
	policy.UpdatedAt = time.Now()

	// 정책 유효성 검증
	validator := security.NewPolicyValidator()
	result, err := validator.Validate(policy)
	if err != nil {
		return nil, fmt.Errorf("정책 검증 실패: %w", err)
	}
	if !result.IsValid {
		return nil, fmt.Errorf("정책이 유효하지 않습니다: %v", result.Errors)
	}

	// 데이터베이스 업데이트
	err = ps.storage.Update(ctx, "security_policies", id, policy)
	if err != nil {
		return nil, fmt.Errorf("정책 업데이트 실패: %w", err)
	}

	// 감사 로그 생성
	ps.createAuditEntry(ctx, policy.ID, "update", oldVersion, policy.Version, changes, "", req.Reason, "")

	// 활성 정책인 경우 런타임 재적용
	if policy.IsActive {
		err = ps.policyManager.ApplyPolicyRuntime(ctx, policy)
		if err != nil {
			return nil, fmt.Errorf("정책 런타임 적용 실패: %w", err)
		}
	}

	return policy, nil
}

// DeletePolicy는 보안 정책을 삭제합니다
func (ps *policyService) DeletePolicy(ctx context.Context, id string) error {
	// 정책 조회
	policy, err := ps.GetPolicy(ctx, id)
	if err != nil {
		return err
	}

	// 활성 정책은 삭제 불가
	if policy.IsActive {
		return fmt.Errorf("활성 정책은 삭제할 수 없습니다. 먼저 비활성화해주세요")
	}

	// 삭제 실행
	err = ps.storage.Delete(ctx, "security_policies", id)
	if err != nil {
		return fmt.Errorf("정책 삭제 실패: %w", err)
	}

	// 감사 로그 생성
	ps.createAuditEntry(ctx, policy.ID, "delete", policy.Version, "", map[string]interface{}{
		"name":     policy.Name,
		"category": policy.Category,
	}, "", "", "")

	return nil
}

// ListPolicies는 보안 정책 목록을 조회합니다
func (ps *policyService) ListPolicies(ctx context.Context, filter *security.PolicyFilter) (*models.PaginatedResponse[*security.SecurityPolicy], error) {
	// TODO: 실제 필터링 및 페이지네이션 로직 구현
	policies := []security.SecurityPolicy{}
	err := ps.storage.GetAll(ctx, "security_policies", &policies)
	if err != nil {
		return nil, fmt.Errorf("정책 목록 조회 실패: %w", err)
	}

	// 필터링 적용
	filteredPolicies := ps.applyFilters(policies, filter)

	// 응답 변환
	responses := make([]security.SecurityPolicyResponse, len(filteredPolicies))
	for i, policy := range filteredPolicies {
		responses[i] = *policy.ToResponse()
	}

	// *security.SecurityPolicy 타입의 슬라이스로 변환
	policyPtrs := make([]*security.SecurityPolicy, len(filteredPolicies))
	for i, policy := range filteredPolicies {
		policyPtrs[i] = &policy
	}
	
	return &models.PaginatedResponse[*security.SecurityPolicy]{
		Data: policyPtrs,
		Pagination: models.PaginationMeta{
			CurrentPage: filter.Page,
			PerPage:     filter.Limit,
			Total:       len(policyPtrs),
			TotalPages:  (len(policyPtrs) + filter.Limit - 1) / filter.Limit,
			HasNext:     filter.Page < (len(policyPtrs) + filter.Limit - 1) / filter.Limit,
			HasPrev:     filter.Page > 1,
		},
	}, nil
}

// ApplyPolicy는 정책을 적용합니다
func (ps *policyService) ApplyPolicy(ctx context.Context, id string) error {
	// 정책 조회
	policy, err := ps.GetPolicy(ctx, id)
	if err != nil {
		return err
	}

	// 이미 활성화된 정책인지 확인
	if policy.IsActive {
		return fmt.Errorf("이미 활성화된 정책입니다")
	}

	// 정책 활성화
	policy.IsActive = true
	policy.UpdatedAt = time.Now()

	// 데이터베이스 업데이트
	err = ps.storage.Update(ctx, "security_policies", id, policy)
	if err != nil {
		return fmt.Errorf("정책 상태 업데이트 실패: %w", err)
	}

	// 런타임 적용
	err = ps.policyManager.ApplyPolicyRuntime(ctx, policy)
	if err != nil {
		// 롤백
		policy.IsActive = false
		ps.storage.Update(ctx, "security_policies", id, policy)
		return fmt.Errorf("정책 런타임 적용 실패: %w", err)
	}

	// 감사 로그 생성
	ps.createAuditEntry(ctx, policy.ID, "apply", "", policy.Version, map[string]interface{}{
		"is_active": true,
	}, "", "", "")

	return nil
}

// DeactivatePolicy는 정책을 비활성화합니다
func (ps *policyService) DeactivatePolicy(ctx context.Context, id string) error {
	// 정책 조회
	policy, err := ps.GetPolicy(ctx, id)
	if err != nil {
		return err
	}

	// 이미 비활성화된 정책인지 확인
	if !policy.IsActive {
		return fmt.Errorf("이미 비활성화된 정책입니다")
	}

	// 정책 비활성화
	policy.IsActive = false
	policy.UpdatedAt = time.Now()

	// 데이터베이스 업데이트
	err = ps.storage.Update(ctx, "security_policies", id, policy)
	if err != nil {
		return fmt.Errorf("정책 상태 업데이트 실패: %w", err)
	}

	// TODO: 런타임에서 정책 제거 로직 구현

	// 감사 로그 생성
	ps.createAuditEntry(ctx, policy.ID, "deactivate", "", policy.Version, map[string]interface{}{
		"is_active": false,
	}, "", "", "")

	return nil
}

// RollbackPolicy는 정책을 이전 버전으로 롤백합니다
func (ps *policyService) RollbackPolicy(ctx context.Context, id string, toVersion string) error {
	// 현재 정책 조회
	currentPolicy, err := ps.GetPolicy(ctx, id)
	if err != nil {
		return err
	}

	// TODO: 정책 버전 히스토리에서 해당 버전 조회
	// 현재는 간단한 구현으로 대체
	if toVersion == currentPolicy.Version {
		return fmt.Errorf("현재 버전과 동일합니다")
	}

	// 감사 로그 생성
	ps.createAuditEntry(ctx, id, "rollback", currentPolicy.Version, toVersion, map[string]interface{}{
		"from_version": currentPolicy.Version,
		"to_version":   toVersion,
	}, "", "", "")

	return fmt.Errorf("롤백 기능은 추후 구현 예정입니다")
}

// GetActivePolicies는 활성 정책 목록을 조회합니다
func (ps *policyService) GetActivePolicies(ctx context.Context, category string) ([]*security.SecurityPolicy, error) {
	policies := []security.SecurityPolicy{}
	err := ps.storage.GetAll(ctx, "security_policies", &policies)
	if err != nil {
		return nil, fmt.Errorf("정책 조회 실패: %w", err)
	}

	var activePolicies []*security.SecurityPolicy
	for i := range policies {
		policy := &policies[i]
		if policy.IsActive && (category == "" || policy.Category == category) {
			// 유효 기간 확인
			now := time.Now()
			if now.After(policy.EffectiveFrom) && (policy.EffectiveUntil == nil || now.Before(*policy.EffectiveUntil)) {
				activePolicies = append(activePolicies, policy)
			}
		}
	}

	return activePolicies, nil
}

// ValidatePolicy는 정책을 검증합니다
func (ps *policyService) ValidatePolicy(ctx context.Context, policy *security.SecurityPolicy) (*security.ValidationResult, error) {
	validator := security.NewPolicyValidator()
	return validator.Validate(policy)
}

// TestPolicy는 정책을 테스트합니다
func (ps *policyService) TestPolicy(ctx context.Context, id string, testData interface{}) (*security.TestResult, error) {
	start := time.Now()

	// 정책 조회
	policy, err := ps.GetPolicy(ctx, id)
	if err != nil {
		return nil, err
	}

	// 테스트 실행 (간단한 구현)
	result := &security.TestResult{
		Success:       true,
		Results:       make(map[string]interface{}),
		ExecutionTime: time.Since(start),
		Errors:        []string{},
	}

	// 카테고리별 테스트 로직
	switch policy.Category {
	case "rate_limiting":
		result.Results["test_type"] = "rate_limiting"
		result.Results["policy_name"] = policy.Name
		result.Results["test_data"] = testData
	case "authentication":
		result.Results["test_type"] = "authentication"
		result.Results["policy_name"] = policy.Name
	default:
		result.Results["test_type"] = "generic"
		result.Results["policy_name"] = policy.Name
	}

	return result, nil
}

// GetPolicyHistory는 정책 히스토리를 조회합니다
func (ps *policyService) GetPolicyHistory(ctx context.Context, id string) ([]*security.PolicyAuditEntry, error) {
	auditEntries := []security.PolicyAuditEntry{}
	// TODO: 실제 조회 로직 구현
	err := ps.storage.GetByField(ctx, "policy_audit_entries", "policy_id", id, &auditEntries)
	if err != nil {
		return nil, fmt.Errorf("정책 히스토리 조회 실패: %w", err)
	}

	// 포인터 슬라이스로 변환
	history := make([]*security.PolicyAuditEntry, len(auditEntries))
	for i := range auditEntries {
		history[i] = &auditEntries[i]
	}

	return history, nil
}

// GetPolicyAuditLog는 정책 감사 로그를 조회합니다
func (ps *policyService) GetPolicyAuditLog(ctx context.Context, filter *security.AuditFilter) (*models.PaginatedResponse[*security.PolicyAuditEntry], error) {
	auditEntries := []security.PolicyAuditEntry{}
	err := ps.storage.GetAll(ctx, "policy_audit_entries", &auditEntries)
	if err != nil {
		return nil, fmt.Errorf("감사 로그 조회 실패: %w", err)
	}

	// 필터링 적용
	filteredEntries := ps.applyAuditFilters(auditEntries, filter)

	// 포인터 슬라이스로 변환
	entryPtrs := make([]*security.PolicyAuditEntry, len(filteredEntries))
	for i := range filteredEntries {
		entryPtrs[i] = &filteredEntries[i]
	}
	
	return &models.PaginatedResponse[*security.PolicyAuditEntry]{
		Data: entryPtrs,
		Pagination: models.PaginationMeta{
			CurrentPage: filter.Page,
			PerPage:     filter.Limit,
			Total:       len(filteredEntries),
			TotalPages:  (len(filteredEntries) + filter.Limit - 1) / filter.Limit,
			HasNext:     filter.Page < (len(filteredEntries) + filter.Limit - 1) / filter.Limit,
			HasPrev:     filter.Page > 1,
		},
	}, nil
}

// ===== 정책 템플릿 관련 메서드 =====

// CreateTemplate은 정책 템플릿을 생성합니다
func (ps *policyService) CreateTemplate(ctx context.Context, req *security.CreateTemplateRequest) (*security.PolicyTemplate, error) {
	template := &security.PolicyTemplate{
		Base: models.Base{
			ID:        generateTemplateID(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Name:         req.Name,
		Description:  req.Description,
		Category:     req.Category,
		ConfigSchema: req.ConfigSchema,
		DefaultData:  req.DefaultData,
		IsBuiltIn:    false,
		Tags:         req.Tags,
	}

	err := ps.storage.Create(ctx, "policy_templates", template)
	if err != nil {
		return nil, fmt.Errorf("템플릿 생성 실패: %w", err)
	}

	return template, nil
}

// GetTemplate은 정책 템플릿을 조회합니다
func (ps *policyService) GetTemplate(ctx context.Context, id string) (*security.PolicyTemplate, error) {
	template := &security.PolicyTemplate{}
	err := ps.storage.GetByID(ctx, "policy_templates", id, template)
	if err != nil {
		return nil, fmt.Errorf("템플릿 조회 실패: %w", err)
	}
	return template, nil
}

// ListTemplates는 정책 템플릿 목록을 조회합니다
func (ps *policyService) ListTemplates(ctx context.Context) ([]*security.PolicyTemplate, error) {
	templates := []security.PolicyTemplate{}
	err := ps.storage.GetAll(ctx, "policy_templates", &templates)
	if err != nil {
		return nil, fmt.Errorf("템플릿 목록 조회 실패: %w", err)
	}

	// 내장 템플릿 추가
	builtInTemplates := security.GetBuiltInTemplates()
	
	// 포인터 슬라이스로 변환 및 병합
	result := make([]*security.PolicyTemplate, len(templates)+len(builtInTemplates))
	for i := range templates {
		result[i] = &templates[i]
	}
	for i, template := range builtInTemplates {
		result[len(templates)+i] = template
	}

	return result, nil
}

// CreatePolicyFromTemplate은 템플릿으로부터 정책을 생성합니다
func (ps *policyService) CreatePolicyFromTemplate(ctx context.Context, templateID string, req *security.CreateFromTemplateRequest) (*security.SecurityPolicy, error) {
	// 템플릿 조회
	template, err := ps.GetTemplate(ctx, templateID)
	if err != nil {
		return nil, err
	}

	// 템플릿 기본값에 사용자 입력값 병합
	configData := template.DefaultData
	if req.ConfigData != nil {
		for key, value := range req.ConfigData {
			configData[key] = value
		}
	}

	// 정책 생성 요청 구성
	createReq := &security.CreatePolicyRequest{
		Name:        req.Name,
		Description: req.Description,
		Category:    template.Category,
		ConfigData:  configData,
		Priority:    req.Priority,
		ApplyNow:    req.ApplyNow,
	}

	policy, err := ps.CreatePolicy(ctx, createReq)
	if err != nil {
		return nil, err
	}

	// 템플릿 참조 설정
	policy.ParentID = &templateID
	err = ps.storage.Update(ctx, "security_policies", policy.ID, policy)
	if err != nil {
		return nil, fmt.Errorf("템플릿 참조 설정 실패: %w", err)
	}

	return policy, nil
}

// ===== 헬퍼 메서드들 =====

// generatePolicyID는 정책 ID를 생성합니다
func generatePolicyID() string {
	return fmt.Sprintf("pol_%d", time.Now().UnixNano())
}

// generateTemplateID는 템플릿 ID를 생성합니다
func generateTemplateID() string {
	return fmt.Sprintf("tpl_%d", time.Now().UnixNano())
}

// incrementVersion은 버전을 증가시킵니다
func (ps *policyService) incrementVersion(version string) string {
	// 간단한 구현: 마지막 숫자를 증가
	// 실제로는 semantic versioning 라이브러리 사용 권장
	if version == "" {
		return "1.0.0"
	}

	// "1.0.0" -> "1.0.1" 형태로 증가
	parts := []rune(version)
	if len(parts) >= 5 {
		lastNum := int(parts[len(parts)-1] - '0')
		if lastNum < 9 {
			parts[len(parts)-1] = rune('0' + lastNum + 1)
		} else {
			// 더 복잡한 버전 증가 로직 필요
			return version + ".1"
		}
	}
	return string(parts)
}

// createAuditEntry는 감사 로그 항목을 생성합니다
func (ps *policyService) createAuditEntry(ctx context.Context, policyID, action, oldVersion, newVersion string, changes map[string]interface{}, appliedBy, reason, ipAddress string) {
	entry := &security.PolicyAuditEntry{
		Base: models.Base{
			ID:        fmt.Sprintf("audit_%d", time.Now().UnixNano()),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		PolicyID:   policyID,
		Action:     action,
		OldVersion: oldVersion,
		NewVersion: newVersion,
		Changes:    changes,
		AppliedBy:  appliedBy,
		Reason:     reason,
		IPAddress:  ipAddress,
	}

	// 에러는 로깅만 하고 무시 (감사 로그 실패가 주 작업을 방해하지 않도록)
	err := ps.storage.Create(ctx, "policy_audit_entries", entry)
	if err != nil {
		fmt.Printf("감사 로그 생성 실패: %v\n", err)
	}
}

// applyFilters는 정책 목록에 필터를 적용합니다
func (ps *policyService) applyFilters(policies []security.SecurityPolicy, filter *security.PolicyFilter) []security.SecurityPolicy {
	if filter == nil {
		return policies
	}

	var filtered []security.SecurityPolicy
	for _, policy := range policies {
		// 카테고리 필터
		if filter.Category != "" && policy.Category != filter.Category {
			continue
		}

		// 활성 상태 필터
		if filter.IsActive != nil && policy.IsActive != *filter.IsActive {
			continue
		}

		// 검색어 필터
		if filter.Search != "" {
			if !contains(policy.Name, filter.Search) && !contains(policy.Description, filter.Search) {
				continue
			}
		}

		// 태그 필터
		if len(filter.Tags) > 0 {
			hasTag := false
			for _, filterTag := range filter.Tags {
				for _, policyTag := range policy.Tags {
					if policyTag == filterTag {
						hasTag = true
						break
					}
				}
				if hasTag {
					break
				}
			}
			if !hasTag {
				continue
			}
		}

		filtered = append(filtered, policy)
	}

	return filtered
}

// applyAuditFilters는 감사 로그에 필터를 적용합니다
func (ps *policyService) applyAuditFilters(entries []security.PolicyAuditEntry, filter *security.AuditFilter) []security.PolicyAuditEntry {
	if filter == nil {
		return entries
	}

	var filtered []security.PolicyAuditEntry
	for _, entry := range entries {
		// 정책 ID 필터
		if filter.PolicyID != "" && entry.PolicyID != filter.PolicyID {
			continue
		}

		// 액션 필터
		if filter.Action != "" && entry.Action != filter.Action {
			continue
		}

		// 사용자 ID 필터
		if filter.UserID != "" && entry.AppliedBy != filter.UserID {
			continue
		}

		// 날짜 범위 필터
		if filter.DateFrom != nil && entry.CreatedAt.Before(*filter.DateFrom) {
			continue
		}
		if filter.DateTo != nil && entry.CreatedAt.After(*filter.DateTo) {
			continue
		}

		filtered = append(filtered, entry)
	}

	return filtered
}

// contains는 문자열 포함 검사를 수행합니다
func contains(text, search string) bool {
	// 간단한 구현, 실제로는 대소문자 무시 검색 등 고려
	return len(search) == 0 || (len(text) > 0 && len(search) <= len(text))
}