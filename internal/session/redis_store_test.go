package session

import (
	"context"
	"testing"
	"time"

	"github.com/aicli/aicli-web/internal/models"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRedisClient는 테스트용 Redis 클라이언트를 모킹합니다.
type mockRedisClient struct {
	data map[string]string
	sets map[string]map[string]struct{}
	ttls map[string]time.Duration
}

func newMockRedisClient() *mockRedisClient {
	return &mockRedisClient{
		data: make(map[string]string),
		sets: make(map[string]map[string]struct{}),
		ttls: make(map[string]time.Duration),
	}
}

func (m *mockRedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	m.data[key] = value.(string)
	if expiration > 0 {
		m.ttls[key] = expiration
	}
	return redis.NewStatusCmd(ctx)
}

func (m *mockRedisClient) Get(ctx context.Context, key string) *redis.StringCmd {
	cmd := redis.NewStringCmd(ctx)
	if value, exists := m.data[key]; exists {
		cmd.SetVal(value)
	} else {
		cmd.SetErr(redis.Nil)
	}
	return cmd
}

func (m *mockRedisClient) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	cmd := redis.NewIntCmd(ctx)
	count := 0
	for _, key := range keys {
		if _, exists := m.data[key]; exists {
			delete(m.data, key)
			delete(m.ttls, key)
			count++
		}
	}
	cmd.SetVal(int64(count))
	return cmd
}

func (m *mockRedisClient) SAdd(ctx context.Context, key string, members ...interface{}) *redis.IntCmd {
	cmd := redis.NewIntCmd(ctx)
	if m.sets[key] == nil {
		m.sets[key] = make(map[string]struct{})
	}
	count := 0
	for _, member := range members {
		memberStr := member.(string)
		if _, exists := m.sets[key][memberStr]; !exists {
			m.sets[key][memberStr] = struct{}{}
			count++
		}
	}
	cmd.SetVal(int64(count))
	return cmd
}

func (m *mockRedisClient) SRem(ctx context.Context, key string, members ...interface{}) *redis.IntCmd {
	cmd := redis.NewIntCmd(ctx)
	count := 0
	if set, exists := m.sets[key]; exists {
		for _, member := range members {
			memberStr := member.(string)
			if _, exists := set[memberStr]; exists {
				delete(set, memberStr)
				count++
			}
		}
	}
	cmd.SetVal(int64(count))
	return cmd
}

func (m *mockRedisClient) SMembers(ctx context.Context, key string) *redis.StringSliceCmd {
	cmd := redis.NewStringSliceCmd(ctx)
	var members []string
	if set, exists := m.sets[key]; exists {
		for member := range set {
			members = append(members, member)
		}
	}
	cmd.SetVal(members)
	return cmd
}

func (m *mockRedisClient) Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd {
	cmd := redis.NewBoolCmd(ctx)
	m.ttls[key] = expiration
	cmd.SetVal(true)
	return cmd
}

func (m *mockRedisClient) TTL(ctx context.Context, key string) *redis.DurationCmd {
	cmd := redis.NewDurationCmd(ctx, time.Second)
	if ttl, exists := m.ttls[key]; exists {
		cmd.SetVal(ttl)
	} else {
		cmd.SetVal(-1 * time.Second)
	}
	return cmd
}

func (m *mockRedisClient) Exists(ctx context.Context, keys ...string) *redis.IntCmd {
	cmd := redis.NewIntCmd(ctx)
	count := 0
	for _, key := range keys {
		if _, exists := m.data[key]; exists {
			count++
		}
	}
	cmd.SetVal(int64(count))
	return cmd
}

func (m *mockRedisClient) Scan(ctx context.Context, cursor uint64, match string, count int64) *redis.ScanCmd {
	cmd := redis.NewScanCmd(ctx, nil)
	var keys []string
	for key := range m.data {
		keys = append(keys, key)
	}
	cmd.SetVal(keys, 0)
	return cmd
}


func TestNewRedisStore(t *testing.T) {
	client := newMockRedisClient()
	store := NewRedisStore(client, "test", time.Hour)
	
	assert.NotNil(t, store)
	assert.Equal(t, client, store.client)
	assert.Equal(t, "test", store.keyPrefix)
	assert.Equal(t, time.Hour, store.defaultTTL)
}

func TestRedisStore_sessionKey(t *testing.T) {
	store := NewRedisStore(nil, "aicli", time.Hour)
	
	key := store.sessionKey("session123")
	assert.Equal(t, "aicli:session:session123", key)
}

func TestRedisStore_userSessionsKey(t *testing.T) {
	store := NewRedisStore(nil, "aicli", time.Hour)
	
	key := store.userSessionsKey("user456")
	assert.Equal(t, "aicli:user_sessions:user456", key)
}

func TestRedisStore_deviceSessionsKey(t *testing.T) {
	store := NewRedisStore(nil, "aicli", time.Hour)
	
	key := store.deviceSessionsKey("device789")
	assert.Equal(t, "aicli:device_sessions:device789", key)
}

func TestRedisStore_Create(t *testing.T) {
	client := newMockRedisClient()
	store := NewRedisStore(client, "test", time.Hour)
	ctx := context.Background()
	
	session := &models.AuthSession{
		ID:       "session123",
		UserID:   "user456",
		IsActive: true,
		DeviceInfo: &models.DeviceFingerprint{
			Fingerprint: "device789",
		},
		CreatedAt:  time.Now(),
		LastAccess: time.Now(),
		ExpiresAt:  time.Now().Add(time.Hour),
	}
	
	err := store.Create(ctx, session)
	assert.NoError(t, err)
	
	// 세션이 저장되었는지 확인
	sessionKey := store.sessionKey("session123")
	assert.Contains(t, client.data, sessionKey)
	
	// 사용자 세션 목록에 추가되었는지 확인
	userKey := store.userSessionsKey("user456")
	assert.Contains(t, client.sets, userKey)
	assert.Contains(t, client.sets[userKey], "session123")
	
	// 디바이스 세션에 추가되었는지 확인
	deviceKey := store.deviceSessionsKey("device789")
	assert.Contains(t, client.sets, deviceKey)
	assert.Contains(t, client.sets[deviceKey], "session123")
}

func TestRedisStore_Get(t *testing.T) {
	client := newMockRedisClient()
	store := NewRedisStore(client, "test", time.Hour)
	ctx := context.Background()
	
	// 존재하지 않는 세션 조회
	session, err := store.Get(ctx, "nonexistent")
	assert.Error(t, err)
	assert.Equal(t, ErrSessionNotFound, err)
	assert.Nil(t, session)
	
	// 세션 생성
	originalSession := &models.AuthSession{
		ID:       "session123",
		UserID:   "user456",
		IsActive: true,
		CreatedAt:  time.Now(),
		LastAccess: time.Now(),
		ExpiresAt:  time.Now().Add(time.Hour),
	}
	
	err = store.Create(ctx, originalSession)
	require.NoError(t, err)
	
	// 세션 조회
	retrievedSession, err := store.Get(ctx, "session123")
	assert.NoError(t, err)
	assert.NotNil(t, retrievedSession)
	assert.Equal(t, originalSession.ID, retrievedSession.ID)
	assert.Equal(t, originalSession.UserID, retrievedSession.UserID)
	assert.Equal(t, originalSession.IsActive, retrievedSession.IsActive)
}

func TestRedisStore_Update(t *testing.T) {
	client := newMockRedisClient()
	store := NewRedisStore(client, "test", time.Hour)
	ctx := context.Background()
	
	// 존재하지 않는 세션 업데이트 시도
	session := &models.AuthSession{
		ID:       "nonexistent",
		UserID:   "user456",
		IsActive: false,
	}
	
	err := store.Update(ctx, session)
	assert.Error(t, err)
	assert.Equal(t, ErrSessionNotFound, err)
	
	// 세션 생성
	originalSession := &models.AuthSession{
		ID:       "session123",
		UserID:   "user456",
		IsActive: true,
		CreatedAt:  time.Now(),
		LastAccess: time.Now(),
		ExpiresAt:  time.Now().Add(time.Hour),
	}
	
	err = store.Create(ctx, originalSession)
	require.NoError(t, err)
	
	// 세션 업데이트
	originalSession.IsActive = false
	originalSession.LastAccess = time.Now().Add(time.Minute)
	
	err = store.Update(ctx, originalSession)
	assert.NoError(t, err)
	
	// 업데이트 확인
	updatedSession, err := store.Get(ctx, "session123")
	assert.NoError(t, err)
	assert.False(t, updatedSession.IsActive)
}

func TestRedisStore_Delete(t *testing.T) {
	client := newMockRedisClient()
	store := NewRedisStore(client, "test", time.Hour)
	ctx := context.Background()
	
	// 존재하지 않는 세션 삭제
	err := store.Delete(ctx, "nonexistent")
	assert.NoError(t, err)
	
	// 세션 생성
	session := &models.AuthSession{
		ID:       "session123",
		UserID:   "user456",
		IsActive: true,
		DeviceInfo: &models.DeviceFingerprint{
			Fingerprint: "device789",
		},
	}
	
	err = store.Create(ctx, session)
	require.NoError(t, err)
	
	// 세션 삭제
	err = store.Delete(ctx, "session123")
	assert.NoError(t, err)
	
	// 삭제 확인
	_, err = store.Get(ctx, "session123")
	assert.Error(t, err)
	assert.Equal(t, ErrSessionNotFound, err)
	
	// 사용자 세션 목록에서도 제거되었는지 확인
	userKey := store.userSessionsKey("user456")
	assert.NotContains(t, client.sets[userKey], "session123")
	
	// 디바이스 세션에서도 제거되었는지 확인
	deviceKey := store.deviceSessionsKey("device789")
	assert.NotContains(t, client.sets[deviceKey], "session123")
}

func TestRedisStore_GetUserSessions(t *testing.T) {
	client := newMockRedisClient()
	store := NewRedisStore(client, "test", time.Hour)
	ctx := context.Background()
	
	// 존재하지 않는 사용자
	sessions, err := store.GetUserSessions(ctx, "nonexistent")
	assert.NoError(t, err)
	assert.Empty(t, sessions)
	
	// 사용자 세션들 생성
	session1 := &models.AuthSession{
		ID:       "session1",
		UserID:   "user456",
		IsActive: true,
	}
	
	session2 := &models.AuthSession{
		ID:       "session2",
		UserID:   "user456",
		IsActive: false, // 비활성 세션
	}
	
	session3 := &models.AuthSession{
		ID:       "session3",
		UserID:   "user456",
		IsActive: true,
	}
	
	err = store.Create(ctx, session1)
	require.NoError(t, err)
	err = store.Create(ctx, session2)
	require.NoError(t, err)
	err = store.Create(ctx, session3)
	require.NoError(t, err)
	
	// 사용자 세션 조회 (활성 세션만)
	sessions, err = store.GetUserSessions(ctx, "user456")
	assert.NoError(t, err)
	assert.Len(t, sessions, 2) // 활성 세션만 반환
	
	sessionIDs := make([]string, len(sessions))
	for i, s := range sessions {
		sessionIDs[i] = s.ID
	}
	assert.Contains(t, sessionIDs, "session1")
	assert.Contains(t, sessionIDs, "session3")
	assert.NotContains(t, sessionIDs, "session2") // 비활성 세션은 제외
}

func TestRedisStore_CountUserActiveSessions(t *testing.T) {
	client := newMockRedisClient()
	store := NewRedisStore(client, "test", time.Hour)
	ctx := context.Background()
	
	// 초기 상태
	count, err := store.CountUserActiveSessions(ctx, "user456")
	assert.NoError(t, err)
	assert.Equal(t, 0, count)
	
	// 세션들 생성
	sessions := []*models.AuthSession{
		{ID: "s1", UserID: "user456", IsActive: true},
		{ID: "s2", UserID: "user456", IsActive: true},
		{ID: "s3", UserID: "user456", IsActive: false},
	}
	
	for _, session := range sessions {
		err = store.Create(ctx, session)
		require.NoError(t, err)
	}
	
	// 활성 세션 수 확인
	count, err = store.CountUserActiveSessions(ctx, "user456")
	assert.NoError(t, err)
	assert.Equal(t, 2, count) // 활성 세션 2개만
}

func TestRedisStore_ExtendSession(t *testing.T) {
	client := newMockRedisClient()
	store := NewRedisStore(client, "test", time.Hour)
	ctx := context.Background()
	
	// 존재하지 않는 세션 연장 시도
	err := store.ExtendSession(ctx, "nonexistent", time.Hour*2)
	assert.Error(t, err)
	assert.Equal(t, ErrSessionNotFound, err)
	
	// 세션 생성
	session := &models.AuthSession{
		ID:       "session123",
		UserID:   "user456",
		IsActive: true,
	}
	
	err = store.Create(ctx, session)
	require.NoError(t, err)
	
	// 세션 연장
	err = store.ExtendSession(ctx, "session123", time.Hour*2)
	assert.NoError(t, err)
	
	// TTL 확인
	sessionKey := store.sessionKey("session123")
	ttl := client.ttls[sessionKey]
	assert.Equal(t, time.Hour*2, ttl)
}

// Benchmark 테스트
func BenchmarkRedisStore_Create(b *testing.B) {
	client := newMockRedisClient()
	store := NewRedisStore(client, "test", time.Hour)
	ctx := context.Background()
	
	session := &models.AuthSession{
		ID:       "benchmark",
		UserID:   "user",
		IsActive: true,
		DeviceInfo: &models.DeviceFingerprint{
			Fingerprint: "device",
		},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		session.ID = string(rune(i))
		store.Create(ctx, session)
	}
}

func BenchmarkRedisStore_Get(b *testing.B) {
	client := newMockRedisClient()
	store := NewRedisStore(client, "test", time.Hour)
	ctx := context.Background()
	
	session := &models.AuthSession{
		ID:       "benchmark",
		UserID:   "user",
		IsActive: true,
	}
	
	store.Create(ctx, session)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.Get(ctx, "benchmark")
	}
}

func BenchmarkRedisStore_GetUserSessions(b *testing.B) {
	client := newMockRedisClient()
	store := NewRedisStore(client, "test", time.Hour)
	ctx := context.Background()
	
	// 여러 세션 생성
	for i := 0; i < 10; i++ {
		session := &models.AuthSession{
			ID:       string(rune(i)),
			UserID:   "user",
			IsActive: true,
		}
		store.Create(ctx, session)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.GetUserSessions(ctx, "user")
	}
}