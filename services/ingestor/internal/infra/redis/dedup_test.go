package redis_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"

	redisinfra "apex/ingestor/internal/infra/redis"
)

func newTestDedup(t *testing.T) (*redisinfra.RedisDeduplicator, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	rdb := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	d := redisinfra.NewDeduplicator(rdb)
	return d, mr
}

// 1. New key: CheckAndMark → returns (false, nil), key exists in Redis
func TestDedup_NewKey(t *testing.T) {
	d, mr := newTestDedup(t)
	defer mr.Close()

	isDup, err := d.CheckAndMark(context.Background(), "abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if isDup {
		t.Error("expected isDuplicate=false for new key")
	}
	if !mr.Exists("dedup:abc123") {
		t.Error("key should exist in Redis after CheckAndMark")
	}
}

// 2. Duplicate key: call twice → second call returns (true, nil)
func TestDedup_DuplicateKey(t *testing.T) {
	d, mr := newTestDedup(t)
	defer mr.Close()

	if _, err := d.CheckAndMark(context.Background(), "dup_hash"); err != nil {
		t.Fatalf("first call error: %v", err)
	}
	isDup, err := d.CheckAndMark(context.Background(), "dup_hash")
	if err != nil {
		t.Fatalf("second call error: %v", err)
	}
	if !isDup {
		t.Error("expected isDuplicate=true on second call with same hash")
	}
}

// 3. TTL set: key has TTL approximately 7 days
func TestDedup_TTLSet(t *testing.T) {
	d, mr := newTestDedup(t)
	defer mr.Close()

	_, _ = d.CheckAndMark(context.Background(), "ttl_hash")

	ttl := mr.TTL("dedup:ttl_hash")
	sevenDays := 7 * 24 * time.Hour

	// Allow a small tolerance (1 minute) since time may pass between set and check
	tolerance := time.Minute
	if ttl < sevenDays-tolerance || ttl > sevenDays+tolerance {
		t.Errorf("expected TTL ~7 days, got %v", ttl)
	}
}

// 4. Expired key: set TTL to 1ms, fast-forward, check → treated as new
func TestDedup_ExpiredKey(t *testing.T) {
	d, mr := newTestDedup(t)
	defer mr.Close()

	// First call: mark key
	isDup, err := d.CheckAndMark(context.Background(), "expire_hash")
	if err != nil {
		t.Fatalf("first call error: %v", err)
	}
	if isDup {
		t.Fatal("first call should not be duplicate")
	}

	// Fast-forward miniredis clock so the key expires
	mr.FastForward(8 * 24 * time.Hour)

	// After expiry the key should be treated as new
	isDup, err = d.CheckAndMark(context.Background(), "expire_hash")
	if err != nil {
		t.Fatalf("post-expiry call error: %v", err)
	}
	if isDup {
		t.Error("expected expired key to be treated as new")
	}
}
