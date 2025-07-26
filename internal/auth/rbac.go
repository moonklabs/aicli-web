package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
	
	"github.com/aicli/aicli-web/internal/models"
)

// RBACManager RBAC 시스템 관리자
type RBACManager struct {
	storage     RBACStorage
	cache       PermissionCache
	evaluator   PermissionEvaluator
}

// RBACStorage RBAC 저장소 인터페이스
type RBACStorage interface {
	// Role 관련
	GetRoleByID(ctx context.Context, roleID string) (*models.Role, error)
	GetRolesByUserID(ctx context.Context, userID string) ([]models.Role, error)
	GetRolesByGroupID(ctx context.Context, groupID string) ([]models.Role, error)
	GetRoleHierarchy(ctx context.Context, roleID string) ([]models.Role, error)
	
	// Permission 관련
	GetPermissionsByRoleID(ctx context.Context, roleID string) ([]models.Permission, error)
	GetAllPermissions(ctx context.Context) ([]models.Permission, error)
	
	// UserRole 관련
	GetUserRoles(ctx context.Context, userID string, resourceID *string) ([]models.UserRole, error)
	GetUserGroups(ctx context.Context, userID string) ([]models.UserGroup, error)
	GetGroupRoles(ctx context.Context, groupID string, resourceID *string) ([]models.GroupRole, error)
	
	// Resource 관련
	GetResourceByID(ctx context.Context, resourceID string) (*models.Resource, error)
	GetResourceHierarchy(ctx context.Context, resourceID string) ([]models.Resource, error)
}

// PermissionCache 권한 캐시 인터페이스
type PermissionCache interface {
	GetUserPermissionMatrix(userID string) (*models.UserPermissionMatrix, error)
	SetUserPermissionMatrix(userID string, matrix *models.UserPermissionMatrix, ttl time.Duration) error
	InvalidateUser(userID string) error
	InvalidateRole(roleID string) error
	InvalidateGroup(groupID string) error
}

// PermissionEvaluator 권한 평가자
type PermissionEvaluator struct {
	conditionEvaluator ConditionEvaluator
}

// ConditionEvaluator 조건 평가자 인터페이스
type ConditionEvaluator interface {
	EvaluateConditions(conditions string, context map[string]interface{}) (bool, error)
}

// NewRBACManager RBAC 매니저 생성자
func NewRBACManager(storage RBACStorage, cache PermissionCache) *RBACManager {
	return &RBACManager{
		storage:   storage,
		cache:     cache,
		evaluator: PermissionEvaluator{
			conditionEvaluator: &JSONConditionEvaluator{},
		},
	}
}

// CheckPermission 사용자 권한 확인
func (rm *RBACManager) CheckPermission(ctx context.Context, req *models.CheckPermissionRequest) (*models.CheckPermissionResponse, error) {
	// 1. 캐시된 권한 매트릭스 조회
	matrix, err := rm.cache.GetUserPermissionMatrix(req.UserID)
	if err != nil || matrix == nil {
		// 캐시 미스 - 권한 매트릭스 재계산
		matrix, err = rm.ComputeUserPermissionMatrix(ctx, req.UserID)
		if err != nil {
			return nil, fmt.Errorf("권한 매트릭스 계산 실패: %w", err)
		}
		
		// 캐시에 저장 (30분 TTL)
		if cacheErr := rm.cache.SetUserPermissionMatrix(req.UserID, matrix, 30*time.Minute); cacheErr != nil {
			// 캐시 저장 실패는 로깅만 하고 계속 진행
			fmt.Printf("권한 매트릭스 캐시 저장 실패: %v\n", cacheErr)
		}
	}
	
	// 2. 권한 키 생성
	permKey := rm.buildPermissionKey(req.ResourceType, req.ResourceID, req.Action)
	
	// 3. 권한 결정 조회 (특정 리소스 ID 먼저 확인)
	decision, exists := matrix.FinalPermissions[permKey]
	if !exists {
		// 와일드카드 권한 확인
		wildcardKey := rm.buildPermissionKey(req.ResourceType, "*", req.Action)
		decision, exists = matrix.FinalPermissions[wildcardKey]
		if !exists {
			// 기본적으로 거부
			decision = models.PermissionDecision{
				ResourceType: req.ResourceType,
				ResourceID:   req.ResourceID,
				Action:       req.Action,
				Effect:       models.PermissionDeny,
				Source:       "default",
				Reason:       "권한이 명시적으로 부여되지 않음",
			}
		} else {
			// 와일드카드 권한을 사용할 때 리소스 ID 업데이트
			decision.ResourceID = req.ResourceID
		}
	}
	
	// 4. 조건 평가 (필요한 경우)
	if decision.Conditions != "" && len(req.Attributes) > 0 {
		conditionMet, err := rm.evaluator.conditionEvaluator.EvaluateConditions(
			decision.Conditions, 
			convertAttributesToMap(req.Attributes),
		)
		if err != nil {
			return nil, fmt.Errorf("조건 평가 실패: %w", err)
		}
		
		if !conditionMet {
			decision.Effect = models.PermissionDeny
			decision.Reason = "조건이 충족되지 않음"
		}
	}
	
	// 5. 평가 과정 기록
	evaluation := rm.buildEvaluationTrace(matrix, permKey, decision)
	
	return &models.CheckPermissionResponse{
		Allowed:    decision.Effect == models.PermissionAllow,
		Decision:   decision,
		Evaluation: evaluation,
	}, nil
}

// ComputeUserPermissionMatrix 사용자 권한 매트릭스 계산
func (rm *RBACManager) ComputeUserPermissionMatrix(ctx context.Context, userID string) (*models.UserPermissionMatrix, error) {
	matrix := &models.UserPermissionMatrix{
		UserID:           userID,
		DirectRoles:      make([]string, 0),
		InheritedRoles:   make([]string, 0),
		GroupRoles:       make([]string, 0),
		FinalPermissions: make(map[string]models.PermissionDecision),
		ComputedAt:       time.Now(),
	}
	
	// 1. 직접 할당된 역할 조회
	directRoles, err := rm.storage.GetRolesByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("사용자 역할 조회 실패: %w", err)
	}
	
	for _, role := range directRoles {
		matrix.DirectRoles = append(matrix.DirectRoles, role.ID)
	}
	
	// 2. 그룹을 통한 역할 조회
	groups, err := rm.storage.GetUserGroups(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("사용자 그룹 조회 실패: %w", err)
	}
	
	groupRoleMap := make(map[string]bool)
	for _, group := range groups {
		groupRoles, err := rm.storage.GetRolesByGroupID(ctx, group.ID)
		if err != nil {
			continue // 에러 발생한 그룹은 건너뛰기
		}
		
		for _, role := range groupRoles {
			if !groupRoleMap[role.ID] {
				matrix.GroupRoles = append(matrix.GroupRoles, role.ID)
				groupRoleMap[role.ID] = true
			}
		}
	}
	
	// 3. 모든 역할에 대해 권한 상속 계산
	allRoles := append(directRoles, []models.Role{}...)
	for roleID := range groupRoleMap {
		role, err := rm.storage.GetRoleByID(ctx, roleID)
		if err != nil {
			continue
		}
		allRoles = append(allRoles, *role)
	}
	
	// 4. 역할별 권한 계산 및 병합
	for _, role := range allRoles {
		err := rm.processRolePermissions(ctx, &role, matrix, "role")
		if err != nil {
			fmt.Printf("역할 %s 권한 처리 실패: %v\n", role.ID, err)
			continue
		}
		
		// 상속된 역할 처리
		inheritedRoles, err := rm.storage.GetRoleHierarchy(ctx, role.ID)
		if err != nil {
			continue
		}
		
		for _, inheritedRole := range inheritedRoles {
			matrix.InheritedRoles = append(matrix.InheritedRoles, inheritedRole.ID)
			err := rm.processRolePermissions(ctx, &inheritedRole, matrix, "inherited")
			if err != nil {
				fmt.Printf("상속 역할 %s 권한 처리 실패: %v\n", inheritedRole.ID, err)
			}
		}
	}
	
	// 5. 중복 제거 및 정렬
	matrix.DirectRoles = rm.removeDuplicates(matrix.DirectRoles)
	matrix.InheritedRoles = rm.removeDuplicates(matrix.InheritedRoles)
	matrix.GroupRoles = rm.removeDuplicates(matrix.GroupRoles)
	
	return matrix, nil
}

// processRolePermissions 역할의 권한을 처리하여 매트릭스에 반영
func (rm *RBACManager) processRolePermissions(ctx context.Context, role *models.Role, matrix *models.UserPermissionMatrix, source string) error {
	permissions, err := rm.storage.GetPermissionsByRoleID(ctx, role.ID)
	if err != nil {
		return fmt.Errorf("역할 권한 조회 실패: %w", err)
	}
	
	for _, perm := range permissions {
		if !perm.IsActive {
			continue
		}
		
		// 권한 키 생성 (일반적인 형태)
		permKey := rm.buildPermissionKey(perm.ResourceType, "*", perm.Action)
		
		// 기존 권한 확인
		existingDecision, exists := matrix.FinalPermissions[permKey]
		
		decision := models.PermissionDecision{
			ResourceType: perm.ResourceType,
			ResourceID:   "*",
			Action:       perm.Action,
			Effect:       perm.Effect,
			Source:       fmt.Sprintf("%s:%s", source, role.Name),
			Reason:       fmt.Sprintf("역할 '%s'을 통해 부여됨", role.Name),
			Conditions:   perm.Conditions,
		}
		
		// 권한 충돌 해결
		if exists {
			// 거부 권한이 허용 권한보다 우선
			if existingDecision.Effect == models.PermissionDeny && decision.Effect == models.PermissionAllow {
				continue // 기존 거부 권한 유지
			} else if existingDecision.Effect == models.PermissionAllow && decision.Effect == models.PermissionDeny {
				// 새로운 거부 권한이 우선
				decision.Reason = fmt.Sprintf("거부 권한이 허용 권한을 오버라이드함 (출처: %s)", decision.Source)
			}
		}
		
		matrix.FinalPermissions[permKey] = decision
	}
	
	return nil
}

// buildPermissionKey 권한 키 생성
func (rm *RBACManager) buildPermissionKey(resourceType models.ResourceType, resourceID string, action models.ActionType) string {
	return fmt.Sprintf("%s:%s:%s", resourceType, resourceID, action)
}

// buildEvaluationTrace 평가 과정 추적 정보 생성
func (rm *RBACManager) buildEvaluationTrace(matrix *models.UserPermissionMatrix, permKey string, decision models.PermissionDecision) []string {
	trace := make([]string, 0)
	
	trace = append(trace, fmt.Sprintf("사용자 ID: %s", matrix.UserID))
	trace = append(trace, fmt.Sprintf("직접 역할: [%s]", strings.Join(matrix.DirectRoles, ", ")))
	trace = append(trace, fmt.Sprintf("상속 역할: [%s]", strings.Join(matrix.InheritedRoles, ", ")))
	trace = append(trace, fmt.Sprintf("그룹 역할: [%s]", strings.Join(matrix.GroupRoles, ", ")))
	trace = append(trace, fmt.Sprintf("권한 키: %s", permKey))
	trace = append(trace, fmt.Sprintf("최종 결정: %s", decision.Effect))
	trace = append(trace, fmt.Sprintf("결정 근거: %s", decision.Reason))
	
	if decision.Conditions != "" {
		trace = append(trace, fmt.Sprintf("적용 조건: %s", decision.Conditions))
	}
	
	return trace
}

// removeDuplicates 중복 제거 및 정렬
func (rm *RBACManager) removeDuplicates(slice []string) []string {
	keys := make(map[string]bool)
	result := make([]string, 0)
	
	for _, item := range slice {
		if !keys[item] {
			keys[item] = true
			result = append(result, item)
		}
	}
	
	sort.Strings(result)
	return result
}

// InvalidateUserPermissions 사용자 권한 캐시 무효화
func (rm *RBACManager) InvalidateUserPermissions(userID string) error {
	return rm.cache.InvalidateUser(userID)
}

// InvalidateRolePermissions 역할 권한 캐시 무효화 (해당 역할을 가진 모든 사용자)
func (rm *RBACManager) InvalidateRolePermissions(roleID string) error {
	return rm.cache.InvalidateRole(roleID)
}

// InvalidateGroupPermissions 그룹 권한 캐시 무효화
func (rm *RBACManager) InvalidateGroupPermissions(groupID string) error {
	return rm.cache.InvalidateGroup(groupID)
}

// JSONConditionEvaluator JSON 기반 조건 평가자 구현
type JSONConditionEvaluator struct{}

// EvaluateConditions JSON 조건 평가
func (jce *JSONConditionEvaluator) EvaluateConditions(conditions string, context map[string]interface{}) (bool, error) {
	if conditions == "" {
		return true, nil // 조건이 없으면 항상 통과
	}
	
	var conditionMap map[string]interface{}
	if err := json.Unmarshal([]byte(conditions), &conditionMap); err != nil {
		return false, fmt.Errorf("조건 JSON 파싱 실패: %w", err)
	}
	
	// 간단한 조건 평가 구현
	return jce.evaluateConditionMap(conditionMap, context), nil
}

// evaluateConditionMap 조건 맵 평가
func (jce *JSONConditionEvaluator) evaluateConditionMap(conditions map[string]interface{}, context map[string]interface{}) bool {
	for key, expectedValue := range conditions {
		contextValue, exists := context[key]
		if !exists {
			return false // 컨텍스트에 필요한 값이 없음
		}
		
		// 값 비교 (간단한 구현)
		if fmt.Sprintf("%v", contextValue) != fmt.Sprintf("%v", expectedValue) {
			return false
		}
	}
	
	return true
}

// convertAttributesToMap 속성을 맵으로 변환
func convertAttributesToMap(attributes map[string]string) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range attributes {
		result[k] = v
	}
	return result
}

// PermissionAggregator 권한 집계자
type PermissionAggregator struct {
	rbacManager *RBACManager
}

// NewPermissionAggregator 권한 집계자 생성
func NewPermissionAggregator(rbacManager *RBACManager) *PermissionAggregator {
	return &PermissionAggregator{
		rbacManager: rbacManager,
	}
}

// AggregateUserPermissions 사용자의 모든 권한 집계
func (pa *PermissionAggregator) AggregateUserPermissions(ctx context.Context, userID string) (map[string][]models.PermissionDecision, error) {
	matrix, err := pa.rbacManager.ComputeUserPermissionMatrix(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("권한 매트릭스 계산 실패: %w", err)
	}
	
	// 리소스 타입별 권한 그룹핑
	groupedPermissions := make(map[string][]models.PermissionDecision)
	
	for _, decision := range matrix.FinalPermissions {
		resourceType := string(decision.ResourceType)
		if _, exists := groupedPermissions[resourceType]; !exists {
			groupedPermissions[resourceType] = make([]models.PermissionDecision, 0)
		}
		groupedPermissions[resourceType] = append(groupedPermissions[resourceType], decision)
	}
	
	// 각 그룹 내에서 정렬
	for resourceType := range groupedPermissions {
		sort.Slice(groupedPermissions[resourceType], func(i, j int) bool {
			perms := groupedPermissions[resourceType]
			return string(perms[i].Action) < string(perms[j].Action)
		})
	}
	
	return groupedPermissions, nil
}

// GetEffectiveRoles 사용자의 효과적인 역할 목록 조회
func (pa *PermissionAggregator) GetEffectiveRoles(ctx context.Context, userID string) ([]models.Role, error) {
	matrix, err := pa.rbacManager.ComputeUserPermissionMatrix(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("권한 매트릭스 계산 실패: %w", err)
	}
	
	// 모든 역할 ID 수집
	allRoleIDs := append(matrix.DirectRoles, matrix.InheritedRoles...)
	allRoleIDs = append(allRoleIDs, matrix.GroupRoles...)
	allRoleIDs = pa.rbacManager.removeDuplicates(allRoleIDs)
	
	// 역할 정보 조회
	roles := make([]models.Role, 0, len(allRoleIDs))
	for _, roleID := range allRoleIDs {
		role, err := pa.rbacManager.storage.GetRoleByID(ctx, roleID)
		if err != nil {
			continue // 에러 발생한 역할은 건너뛰기
		}
		roles = append(roles, *role)
	}
	
	return roles, nil
}