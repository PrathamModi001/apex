package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	keyPrefix = "idem:"
	ttl       = 48 * time.Hour
)

// IdempotencyChecker uses Redis SetNX to prevent duplicate invoice processing.
type IdempotencyChecker struct {
	rdb *redis.Client
}

// NewIdempotencyChecker creates an IdempotencyChecker backed by the given Redis client.
func NewIdempotencyChecker(rdb *redis.Client) *IdempotencyChecker {
	return &IdempotencyChecker{rdb: rdb}
}

// NewRedisClient parses a Redis URL and returns a configured client.
func NewRedisClient(redisURL string) (*redis.Client, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}
	return redis.NewClient(opts), nil
}

// CheckAndMark returns (alreadySeen, error).
// If the key has not been seen before, it is atomically marked with a 48-hour TTL.
func (c *IdempotencyChecker) CheckAndMark(ctx context.Context, key string) (bool, error) {
	redisKey := keyPrefix + key
	wasSet, err := c.rdb.SetNX(ctx, redisKey, "1", ttl).Result()
	if err != nil {
		return false, fmt.Errorf("redis setnx %q: %w", redisKey, err)
	}
	// wasSet=true means the key was NEW (we set it); alreadySeen = !wasSet
	alreadySeen := !wasSet
	return alreadySeen, nil
}
