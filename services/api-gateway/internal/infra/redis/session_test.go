package redis_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"

	redisinfra "apex/api-gateway/internal/infra/redis"
)

func newTestLimiter(t *testing.T) (*redisinfra.RateLimiter, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	return redisinfra.NewRateLimiter(client), mr
}

func TestAllow_UnderLimit(t *testing.T) {
	rl, mr := newTestLimiter(t)
	defer mr.Close()

	allowed, err := rl.Allow(context.Background(), "user1:/invoices", 100, time.Minute)
	if err != nil {
		t.Fatalf("Allow: %v", err)
	}
	if !allowed {
		t.Error("expected allowed=true for first request under limit")
	}
}

func TestAllow_AtLimit(t *testing.T) {
	rl, mr := newTestLimiter(t)
	defer mr.Close()

	key := "user2:/invoices"
	// Exhaust the limit (rate = 3).
	for i := 0; i < 3; i++ {
		allowed, err := rl.Allow(context.Background(), key, 3, time.Minute)
		if err != nil {
			t.Fatalf("Allow call %d: %v", i+1, err)
		}
		if !allowed {
			t.Fatalf("call %d should have been allowed", i+1)
		}
	}

	// 4th call should be rejected.
	allowed, err := rl.Allow(context.Background(), key, 3, time.Minute)
	if err != nil {
		t.Fatalf("Allow (over limit): %v", err)
	}
	if allowed {
		t.Error("expected allowed=false after exhausting limit")
	}
}

func TestAllow_DifferentUserSamePath(t *testing.T) {
	rl, mr := newTestLimiter(t)
	defer mr.Close()

	// Exhaust user1.
	for i := 0; i < 2; i++ {
		rl.Allow(context.Background(), "user1:/vendors", 2, time.Minute) //nolint:errcheck
	}
	blocked, _ := rl.Allow(context.Background(), "user1:/vendors", 2, time.Minute)
	if blocked {
		t.Error("user1 should be blocked after 2 requests")
	}

	// user2 on the same path should still be allowed.
	allowed, err := rl.Allow(context.Background(), "user2:/vendors", 2, time.Minute)
	if err != nil {
		t.Fatalf("Allow user2: %v", err)
	}
	if !allowed {
		t.Error("user2 should be allowed independently of user1")
	}
}

func TestAllow_SameUserDifferentPath(t *testing.T) {
	rl, mr := newTestLimiter(t)
	defer mr.Close()

	// Exhaust user1 on /invoices.
	for i := 0; i < 2; i++ {
		rl.Allow(context.Background(), "user1:/invoices", 2, time.Minute) //nolint:errcheck
	}
	blocked, _ := rl.Allow(context.Background(), "user1:/invoices", 2, time.Minute)
	if blocked {
		t.Error("user1:/invoices should be blocked")
	}

	// Same user on /vendors should be independent.
	allowed, err := rl.Allow(context.Background(), "user1:/vendors", 2, time.Minute)
	if err != nil {
		t.Fatalf("Allow user1:/vendors: %v", err)
	}
	if !allowed {
		t.Error("user1:/vendors should be allowed independently")
	}
}

func TestAllow_WindowResets(t *testing.T) {
	rl, mr := newTestLimiter(t)
	defer mr.Close()

	key := "user3:/policies"

	// Exhaust.
	for i := 0; i < 2; i++ {
		rl.Allow(context.Background(), key, 2, time.Second) //nolint:errcheck
	}
	blocked, _ := rl.Allow(context.Background(), key, 2, time.Second)
	if blocked {
		t.Fatal("should be blocked now")
	}

	// Fast-forward miniredis clock past the window.
	mr.FastForward(2 * time.Second)

	// Should be allowed again.
	allowed, err := rl.Allow(context.Background(), key, 2, time.Second)
	if err != nil {
		t.Fatalf("Allow after reset: %v", err)
	}
	if !allowed {
		t.Error("expected allowed=true after window reset")
	}
}
