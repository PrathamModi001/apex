package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const dedupTTL = 7 * 24 * time.Hour

// RedisDeduplicator uses Redis SetNX for SHA-256 deduplication.
type RedisDeduplicator struct {
	rdb *redis.Client
}

// NewDeduplicator creates a RedisDeduplicator backed by the provided Redis client.
func NewDeduplicator(rdb *redis.Client) *RedisDeduplicator {
	return &RedisDeduplicator{rdb: rdb}
}

// CheckAndMark returns (true, nil) if the SHA-256 was already seen.
// Returns (false, nil) and marks the hash when it is new.
func (d *RedisDeduplicator) CheckAndMark(ctx context.Context, sha256 string) (bool, error) {
	key := fmt.Sprintf("dedup:%s", sha256)
	// SetNX: returns true if the key was newly set (i.e. it did NOT exist before)
	wasSet, err := d.rdb.SetNX(ctx, key, "1", dedupTTL).Result()
	if err != nil {
		return false, fmt.Errorf("redis SetNX: %w", err)
	}
	isDuplicate := !wasSet
	return isDuplicate, nil
}
