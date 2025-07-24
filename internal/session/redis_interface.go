package session

import (
	"context"
	"time"
	
	"github.com/go-redis/redis/v8"
)

// RedisClient는 RedisStore가 필요로 하는 최소한의 Redis 인터페이스입니다.
type RedisClient interface {
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	Get(ctx context.Context, key string) *redis.StringCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	TTL(ctx context.Context, key string) *redis.DurationCmd
	Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd
	Exists(ctx context.Context, keys ...string) *redis.IntCmd
	SAdd(ctx context.Context, key string, members ...interface{}) *redis.IntCmd
	SRem(ctx context.Context, key string, members ...interface{}) *redis.IntCmd
	SMembers(ctx context.Context, key string) *redis.StringSliceCmd
	Scan(ctx context.Context, cursor uint64, match string, count int64) *redis.ScanCmd
}

// redisClientAdapter는 redis.UniversalClient를 RedisClient 인터페이스로 변환합니다.
type redisClientAdapter struct {
	redis.UniversalClient
}

// NewRedisClientAdapter는 새로운 Redis 클라이언트 어댑터를 생성합니다.
func NewRedisClientAdapter(client redis.UniversalClient) RedisClient {
	return &redisClientAdapter{client}
}