package redis_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"

	redisinfra "apex/event-worker/internal/infra/redis"
)

func newTestChecker(t *testing.T) (*redisinfra.IdempotencyChecker, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	rdb := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	return redisinfra.NewIdempotencyChecker(rdb), mr
}

func TestIdempotencyChecker_NewKey_NotSeen(t *testing.T) {
	checker, mr := newTestChecker(t)
	defer mr.Close()

	seen, err := checker.CheckAndMark(context.Background(), "sha256abc:inv-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if seen {
		t.Error("expected not seen for new key, got seen=true")
	}
}

func TestIdempotencyChecker_SameKey_Seen(t *testing.T) {
	checker, mr := newTestChecker(t)
	defer mr.Close()

	key := "sha256abc:inv-002"
	// First call — mark it
	_, err := checker.CheckAndMark(context.Background(), key)
	if err != nil {
		t.Fatalf("first call error: %v", err)
	}

	// Second call — should be seen
	seen, err := checker.CheckAndMark(context.Background(), key)
	if err != nil {
		t.Fatalf("second call error: %v", err)
	}
	if !seen {
		t.Error("expected seen=true for duplicate key, got false")
	}
}

func TestIdempotencyChecker_TTL_ApproximatelyFortyEightHours(t *testing.T) {
	checker, mr := newTestChecker(t)
	defer mr.Close()

	key := "sha256abc:inv-003"
	_, err := checker.CheckAndMark(context.Background(), key)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// miniredis exposes TTL via the underlying store
	redisKey := "idem:" + key
	ttlDur := mr.TTL(redisKey)

	want := 48 * time.Hour
	// Allow ±2 seconds tolerance
	if ttlDur < want-2*time.Second || ttlDur > want+2*time.Second {
		t.Errorf("TTL = %v, want ~%v", ttlDur, want)
	}
}
