package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RateLimiter implements app.RateLimiter using a Redis counter per key.
type RateLimiter struct {
	client redis.UniversalClient
}

// NewRateLimiter creates a RateLimiter backed by the given Redis client.
func NewRateLimiter(client redis.UniversalClient) *RateLimiter {
	return &RateLimiter{client: client}
}

// Allow checks whether the given key is within the rate limit.
// Uses INCR + EXPIRE (counter resets after window).
// Returns (true, nil) if the request is allowed; (false, nil) if rate limited.
func (rl *RateLimiter) Allow(ctx context.Context, key string, rate int, window time.Duration) (bool, error) {
	redisKey := fmt.Sprintf("rl:%s", key)

	pipe := rl.client.Pipeline()
	incrCmd := pipe.Incr(ctx, redisKey)
	pipe.Expire(ctx, redisKey, window)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, fmt.Errorf("redis pipeline: %w", err)
	}

	count := incrCmd.Val()
	return count <= int64(rate), nil
}
