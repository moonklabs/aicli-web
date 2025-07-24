package session

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/aicli/aicli-web/internal/models"
)

// RedisStore는 Redis를 백엔드로 하는 세션 저장소입니다.
type RedisStore struct {
	client     RedisClient
	keyPrefix  string
	defaultTTL time.Duration
}

// NewRedisStore는 새로운 Redis 세션 저장소를 생성합니다.
func NewRedisStore(client RedisClient, keyPrefix string, defaultTTL time.Duration) *RedisStore {
	return &RedisStore{
		client:     client,
		keyPrefix:  keyPrefix,
		defaultTTL: defaultTTL,
	}
}

// NewRedisStoreFromUniversal은 redis.UniversalClient로부터 Redis 세션 저장소를 생성합니다.
func NewRedisStoreFromUniversal(client redis.UniversalClient, keyPrefix string, defaultTTL time.Duration) *RedisStore {
	return NewRedisStore(NewRedisClientAdapter(client), keyPrefix, defaultTTL)
}

// sessionKey는 세션 ID로부터 Redis 키를 생성합니다.
func (s *RedisStore) sessionKey(sessionID string) string {
	return fmt.Sprintf("%s:session:%s", s.keyPrefix, sessionID)
}

// userSessionsKey는 사용자 ID로부터 사용자 세션 목록 키를 생성합니다.
func (s *RedisStore) userSessionsKey(userID string) string {
	return fmt.Sprintf("%s:user_sessions:%s", s.keyPrefix, userID)
}

// deviceSessionsKey는 디바이스 핑거프린트로부터 디바이스 세션 키를 생성합니다.
func (s *RedisStore) deviceSessionsKey(fingerprint string) string {
	return fmt.Sprintf("%s:device_sessions:%s", s.keyPrefix, fingerprint)
}

// Create는 새로운 세션을 생성합니다.
func (s *RedisStore) Create(ctx context.Context, session *models.AuthSession) error {
	sessionData, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("세션 직렬화 실패: %w", err)
	}

	key := s.sessionKey(session.ID)
	
	// 세션 데이터 저장
	err = s.client.Set(ctx, key, sessionData, s.defaultTTL).Err()
	if err != nil {
		return fmt.Errorf("세션 저장 실패: %w", err)
	}

	// 사용자 세션 목록에 추가
	userKey := s.userSessionsKey(session.UserID)
	err = s.client.SAdd(ctx, userKey, session.ID).Err()
	if err != nil {
		return fmt.Errorf("사용자 세션 목록 업데이트 실패: %w", err)
	}
	
	// 사용자 세션 목록 TTL 설정
	s.client.Expire(ctx, userKey, s.defaultTTL*2)

	// 디바이스 세션 추적
	if session.DeviceInfo != nil && session.DeviceInfo.Fingerprint != "" {
		deviceKey := s.deviceSessionsKey(session.DeviceInfo.Fingerprint)
		err = s.client.SAdd(ctx, deviceKey, session.ID).Err()
		if err != nil {
			return fmt.Errorf("디바이스 세션 추적 실패: %w", err)
		}
		s.client.Expire(ctx, deviceKey, s.defaultTTL*2)
	}

	return nil
}

// Get은 세션 ID로 세션을 조회합니다.
func (s *RedisStore) Get(ctx context.Context, sessionID string) (*models.AuthSession, error) {
	key := s.sessionKey(sessionID)
	
	data, err := s.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("세션 조회 실패: %w", err)
	}

	var session models.AuthSession
	err = json.Unmarshal([]byte(data), &session)
	if err != nil {
		return nil, fmt.Errorf("세션 역직렬화 실패: %w", err)
	}

	return &session, nil
}

// Update는 기존 세션을 업데이트합니다.
func (s *RedisStore) Update(ctx context.Context, session *models.AuthSession) error {
	// 기존 세션이 존재하는지 확인
	_, err := s.Get(ctx, session.ID)
	if err != nil {
		return err
	}

	sessionData, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("세션 직렬화 실패: %w", err)
	}

	key := s.sessionKey(session.ID)
	
	// TTL 유지하면서 업데이트
	ttl := s.client.TTL(ctx, key).Val()
	if ttl <= 0 {
		ttl = s.defaultTTL
	}
	
	err = s.client.Set(ctx, key, sessionData, ttl).Err()
	if err != nil {
		return fmt.Errorf("세션 업데이트 실패: %w", err)
	}

	return nil
}

// Delete는 세션을 삭제합니다.
func (s *RedisStore) Delete(ctx context.Context, sessionID string) error {
	session, err := s.Get(ctx, sessionID)
	if err != nil && err != ErrSessionNotFound {
		return err
	}
	
	if session != nil {
		// 사용자 세션 목록에서 제거
		userKey := s.userSessionsKey(session.UserID)
		s.client.SRem(ctx, userKey, sessionID)

		// 디바이스 세션에서 제거
		if session.DeviceInfo != nil && session.DeviceInfo.Fingerprint != "" {
			deviceKey := s.deviceSessionsKey(session.DeviceInfo.Fingerprint)
			s.client.SRem(ctx, deviceKey, sessionID)
		}
	}

	// 세션 삭제
	key := s.sessionKey(sessionID)
	err = s.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("세션 삭제 실패: %w", err)
	}

	return nil
}

// GetUserSessions는 사용자의 모든 활성 세션을 조회합니다.
func (s *RedisStore) GetUserSessions(ctx context.Context, userID string) ([]*models.AuthSession, error) {
	userKey := s.userSessionsKey(userID)
	
	sessionIDs, err := s.client.SMembers(ctx, userKey).Result()
	if err != nil {
		return nil, fmt.Errorf("사용자 세션 ID 조회 실패: %w", err)
	}

	var sessions []*models.AuthSession
	for _, sessionID := range sessionIDs {
		session, err := s.Get(ctx, sessionID)
		if err != nil {
			// 만료된 세션은 목록에서 제거
			if err == ErrSessionNotFound {
				s.client.SRem(ctx, userKey, sessionID)
				continue
			}
			return nil, fmt.Errorf("세션 조회 실패 (%s): %w", sessionID, err)
		}
		
		// 비활성 세션은 제외
		if session.IsActive {
			sessions = append(sessions, session)
		}
	}

	return sessions, nil
}

// GetDeviceSessions는 특정 디바이스의 모든 세션을 조회합니다.
func (s *RedisStore) GetDeviceSessions(ctx context.Context, fingerprint string) ([]*models.AuthSession, error) {
	deviceKey := s.deviceSessionsKey(fingerprint)
	
	sessionIDs, err := s.client.SMembers(ctx, deviceKey).Result()
	if err != nil {
		return nil, fmt.Errorf("디바이스 세션 ID 조회 실패: %w", err)
	}

	var sessions []*models.AuthSession
	for _, sessionID := range sessionIDs {
		session, err := s.Get(ctx, sessionID)
		if err != nil {
			if err == ErrSessionNotFound {
				s.client.SRem(ctx, deviceKey, sessionID)
				continue
			}
			return nil, fmt.Errorf("세션 조회 실패 (%s): %w", sessionID, err)
		}
		
		if session.IsActive {
			sessions = append(sessions, session)
		}
	}

	return sessions, nil
}

// CleanupExpiredSessions는 만료된 세션들을 정리합니다.
func (s *RedisStore) CleanupExpiredSessions(ctx context.Context) error {
	// 스캔 패턴으로 모든 세션 키 조회
	pattern := fmt.Sprintf("%s:session:*", s.keyPrefix)
	
	var cursor uint64
	var keys []string
	
	for {
		var scanKeys []string
		var err error
		
		scanKeys, cursor, err = s.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return fmt.Errorf("세션 키 스캔 실패: %w", err)
		}
		
		keys = append(keys, scanKeys...)
		
		if cursor == 0 {
			break
		}
	}

	var expiredCount int
	for _, key := range keys {
		ttl := s.client.TTL(ctx, key).Val()
		if ttl < 0 {
			// TTL이 음수면 만료됨
			s.client.Del(ctx, key)
			expiredCount++
		}
	}

	return nil
}

// ExtendSession은 세션의 만료 시간을 연장합니다.
func (s *RedisStore) ExtendSession(ctx context.Context, sessionID string, duration time.Duration) error {
	key := s.sessionKey(sessionID)
	
	exists, err := s.client.Exists(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("세션 존재 확인 실패: %w", err)
	}
	
	if exists == 0 {
		return ErrSessionNotFound
	}

	err = s.client.Expire(ctx, key, duration).Err()
	if err != nil {
		return fmt.Errorf("세션 TTL 연장 실패: %w", err)
	}

	return nil
}

// CountUserActiveSessions는 사용자의 활성 세션 수를 반환합니다.
func (s *RedisStore) CountUserActiveSessions(ctx context.Context, userID string) (int, error) {
	sessions, err := s.GetUserSessions(ctx, userID)
	if err != nil {
		return 0, err
	}
	
	return len(sessions), nil
}