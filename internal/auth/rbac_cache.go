package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	
	"github.com/go-redis/redis/v8"
	"github.com/aicli/aicli-web/internal/models"
)

// RedisPermissionCache Redis 기반 권한 캐시 구현
type RedisPermissionCache struct {
	client *redis.Client
	prefix string
}

// NewRedisPermissionCache Redis 권한 캐시 생성자
func NewRedisPermissionCache(client *redis.Client, prefix string) *RedisPermissionCache {
	if prefix == "" {
		prefix = "rbac"
	}
	return &RedisPermissionCache{
		client: client,
		prefix: prefix,
	}
}

// GetUserPermissionMatrix 사용자 권한 매트릭스 조회
func (rpc *RedisPermissionCache) GetUserPermissionMatrix(userID string) (*models.UserPermissionMatrix, error) {
	ctx := context.Background()
	key := rpc.userMatrixKey(userID)
	
	data, err := rpc.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // 캐시 미스
		}
		return nil, fmt.Errorf("Redis 조회 실패: %w", err)
	}
	
	var matrix models.UserPermissionMatrix
	if err := json.Unmarshal([]byte(data), &matrix); err != nil {
		return nil, fmt.Errorf("권한 매트릭스 역직렬화 실패: %w", err)
	}
	
	return &matrix, nil
}

// SetUserPermissionMatrix 사용자 권한 매트릭스 저장
func (rpc *RedisPermissionCache) SetUserPermissionMatrix(userID string, matrix *models.UserPermissionMatrix, ttl time.Duration) error {
	ctx := context.Background()
	key := rpc.userMatrixKey(userID)
	
	data, err := json.Marshal(matrix)
	if err != nil {
		return fmt.Errorf("권한 매트릭스 직렬화 실패: %w", err)
	}
	
	if err := rpc.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("Redis 저장 실패: %w", err)
	}
	
	// 사용자별 캐시 키 목록에 추가 (무효화를 위해)
	userKeysKey := rpc.userKeysKey(userID)
	if err := rpc.client.SAdd(ctx, userKeysKey, key).Err(); err != nil {
		// 키 목록 저장 실패는 로깅만 하고 계속 진행
		fmt.Printf("사용자 키 목록 저장 실패: %v\n", err)
	}
	rpc.client.Expire(ctx, userKeysKey, ttl*2) // 키 목록은 더 오래 보관
	
	return nil
}

// InvalidateUser 사용자 권한 캐시 무효화
func (rpc *RedisPermissionCache) InvalidateUser(userID string) error {
	ctx := context.Background()
	userKeysKey := rpc.userKeysKey(userID)
	
	// 사용자의 모든 캐시 키 조회
	keys, err := rpc.client.SMembers(ctx, userKeysKey).Result()
	if err != nil && err != redis.Nil {
		return fmt.Errorf("사용자 키 목록 조회 실패: %w", err)
	}
	
	// 모든 캐시 키 삭제
	if len(keys) > 0 {
		if err := rpc.client.Del(ctx, keys...).Err(); err != nil {
			return fmt.Errorf("사용자 캐시 키 삭제 실패: %w", err)
		}
	}
	
	// 키 목록도 삭제
	rpc.client.Del(ctx, userKeysKey)
	
	return nil
}

// InvalidateRole 역할 권한 캐시 무효화
func (rpc *RedisPermissionCache) InvalidateRole(roleID string) error {
	ctx := context.Background()
	roleUsersKey := rpc.roleUsersKey(roleID)
	
	// 해당 역할을 가진 모든 사용자 조회
	users, err := rpc.client.SMembers(ctx, roleUsersKey).Result()
	if err != nil && err != redis.Nil {
		return fmt.Errorf("역할 사용자 목록 조회 실패: %w", err)
	}
	
	// 각 사용자의 캐시 무효화
	for _, userID := range users {
		if err := rpc.InvalidateUser(userID); err != nil {
			fmt.Printf("사용자 %s 캐시 무효화 실패: %v\n", userID, err)
		}
	}
	
	// 역할 사용자 목록 삭제
	rpc.client.Del(ctx, roleUsersKey)
	
	return nil
}

// InvalidateGroup 그룹 권한 캐시 무효화
func (rpc *RedisPermissionCache) InvalidateGroup(groupID string) error {
	ctx := context.Background()
	groupUsersKey := rpc.groupUsersKey(groupID)
	
	// 해당 그룹에 속한 모든 사용자 조회
	users, err := rpc.client.SMembers(ctx, groupUsersKey).Result()
	if err != nil && err != redis.Nil {
		return fmt.Errorf("그룹 사용자 목록 조회 실패: %w", err)
	}
	
	// 각 사용자의 캐시 무효화
	for _, userID := range users {
		if err := rpc.InvalidateUser(userID); err != nil {
			fmt.Printf("사용자 %s 캐시 무효화 실패: %v\n", userID, err)
		}
	}
	
	// 그룹 사용자 목록 삭제
	rpc.client.Del(ctx, groupUsersKey)
	
	return nil
}

// TrackUserRole 사용자-역할 관계 추적 (캐시 무효화를 위해)
func (rpc *RedisPermissionCache) TrackUserRole(userID, roleID string) error {
	ctx := context.Background()
	roleUsersKey := rpc.roleUsersKey(roleID)
	
	if err := rpc.client.SAdd(ctx, roleUsersKey, userID).Err(); err != nil {
		return fmt.Errorf("사용자-역할 추적 실패: %w", err)
	}
	
	// 키 만료 시간 설정 (24시간)
	rpc.client.Expire(ctx, roleUsersKey, 24*time.Hour)
	
	return nil
}

// TrackUserGroup 사용자-그룹 관계 추적 (캐시 무효화를 위해)
func (rpc *RedisPermissionCache) TrackUserGroup(userID, groupID string) error {
	ctx := context.Background()
	groupUsersKey := rpc.groupUsersKey(groupID)
	
	if err := rpc.client.SAdd(ctx, groupUsersKey, userID).Err(); err != nil {
		return fmt.Errorf("사용자-그룹 추적 실패: %w", err)
	}
	
	// 키 만료 시간 설정 (24시간)
	rpc.client.Expire(ctx, groupUsersKey, 24*time.Hour)
	
	return nil
}

// GetCacheStats 캐시 통계 조회
func (rpc *RedisPermissionCache) GetCacheStats() (map[string]interface{}, error) {
	ctx := context.Background()
	
	// 패턴에 맞는 모든 키 조회
	pattern := rpc.prefix + ":*"
	keys, err := rpc.client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, fmt.Errorf("캐시 키 조회 실패: %w", err)
	}
	
	stats := map[string]interface{}{
		"total_keys": len(keys),
		"patterns": map[string]int{
			"user_matrix": 0,
			"user_keys":   0,
			"role_users":  0,
			"group_users": 0,
		},
	}
	
	// 키 패턴별 분류
	for _, key := range keys {
		if contains(key, ":user:") && contains(key, ":matrix") {
			stats["patterns"].(map[string]int)["user_matrix"]++
		} else if contains(key, ":user:") && contains(key, ":keys") {
			stats["patterns"].(map[string]int)["user_keys"]++
		} else if contains(key, ":role:") {
			stats["patterns"].(map[string]int)["role_users"]++
		} else if contains(key, ":group:") {
			stats["patterns"].(map[string]int)["group_users"]++
		}
	}
	
	return stats, nil
}

// ClearAll 모든 RBAC 캐시 삭제 (개발/테스트용)
func (rpc *RedisPermissionCache) ClearAll() error {
	ctx := context.Background()
	pattern := rpc.prefix + ":*"
	
	keys, err := rpc.client.Keys(ctx, pattern).Result()
	if err != nil {
		return fmt.Errorf("캐시 키 조회 실패: %w", err)
	}
	
	if len(keys) > 0 {
		if err := rpc.client.Del(ctx, keys...).Err(); err != nil {
			return fmt.Errorf("캐시 삭제 실패: %w", err)
		}
	}
	
	return nil
}

// 키 생성 헬퍼 메서드들

func (rpc *RedisPermissionCache) userMatrixKey(userID string) string {
	return fmt.Sprintf("%s:user:%s:matrix", rpc.prefix, userID)
}

func (rpc *RedisPermissionCache) userKeysKey(userID string) string {
	return fmt.Sprintf("%s:user:%s:keys", rpc.prefix, userID)
}

func (rpc *RedisPermissionCache) roleUsersKey(roleID string) string {
	return fmt.Sprintf("%s:role:%s:users", rpc.prefix, roleID)
}

func (rpc *RedisPermissionCache) groupUsersKey(groupID string) string {
	return fmt.Sprintf("%s:group:%s:users", rpc.prefix, groupID)
}

// contains 문자열 포함 여부 확인 헬퍼
func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (s == substr || 
		    s[:len(substr)] == substr ||
		    s[len(s)-len(substr):] == substr ||
		    findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// InMemoryPermissionCache 인메모리 권한 캐시 구현 (Redis 없는 환경용)
type InMemoryPermissionCache struct {
	userMatrices map[string]*cachedMatrix
	userKeys     map[string][]string
	roleUsers    map[string][]string
	groupUsers   map[string][]string
}

type cachedMatrix struct {
	matrix    *models.UserPermissionMatrix
	expiresAt time.Time
}

// NewInMemoryPermissionCache 인메모리 캐시 생성자
func NewInMemoryPermissionCache() *InMemoryPermissionCache {
	return &InMemoryPermissionCache{
		userMatrices: make(map[string]*cachedMatrix),
		userKeys:     make(map[string][]string),
		roleUsers:    make(map[string][]string),
		groupUsers:   make(map[string][]string),
	}
}

// GetUserPermissionMatrix 사용자 권한 매트릭스 조회
func (ipc *InMemoryPermissionCache) GetUserPermissionMatrix(userID string) (*models.UserPermissionMatrix, error) {
	cached, exists := ipc.userMatrices[userID]
	if !exists {
		return nil, nil // 캐시 미스
	}
	
	// 만료 확인
	if time.Now().After(cached.expiresAt) {
		delete(ipc.userMatrices, userID)
		return nil, nil // 만료된 캐시
	}
	
	return cached.matrix, nil
}

// SetUserPermissionMatrix 사용자 권한 매트릭스 저장
func (ipc *InMemoryPermissionCache) SetUserPermissionMatrix(userID string, matrix *models.UserPermissionMatrix, ttl time.Duration) error {
	ipc.userMatrices[userID] = &cachedMatrix{
		matrix:    matrix,
		expiresAt: time.Now().Add(ttl),
	}
	return nil
}

// InvalidateUser 사용자 권한 캐시 무효화
func (ipc *InMemoryPermissionCache) InvalidateUser(userID string) error {
	delete(ipc.userMatrices, userID)
	delete(ipc.userKeys, userID)
	return nil
}

// InvalidateRole 역할 권한 캐시 무효화
func (ipc *InMemoryPermissionCache) InvalidateRole(roleID string) error {
	users, exists := ipc.roleUsers[roleID]
	if !exists {
		return nil
	}
	
	for _, userID := range users {
		ipc.InvalidateUser(userID)
	}
	
	delete(ipc.roleUsers, roleID)
	return nil
}

// InvalidateGroup 그룹 권한 캐시 무효화
func (ipc *InMemoryPermissionCache) InvalidateGroup(groupID string) error {
	users, exists := ipc.groupUsers[groupID]
	if !exists {
		return nil
	}
	
	for _, userID := range users {
		ipc.InvalidateUser(userID)
	}
	
	delete(ipc.groupUsers, groupID)
	return nil
}