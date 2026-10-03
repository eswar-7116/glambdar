package ratelimit

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func requireTestRedis(t *testing.T) string {
	t.Helper()
	if os.Getenv("RUN_INTEGRATION_TESTS") != "1" {
		t.Skip("RUN_INTEGRATION_TESTS != 1; skipping integration tests")
	}
	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("TEST_REDIS_ADDR not set; skipping Redis test (set TEST_REDIS_ADDR=localhost:6379 to run)")
	}

	client := redis.NewClient(&redis.Options{Addr: addr})
	if err := client.Ping(context.Background()).Err(); err != nil {
		t.Skipf("Cannot connect to Redis at %s: %v", addr, err)
	}

	// Clean up keys
	keys, _ := client.Keys(context.Background(), keyPrefix+"*").Result()
	if len(keys) > 0 {
		client.Del(context.Background(), keys...)
	}

	return addr
}

func TestRedisLimiter_Unlimited(t *testing.T) {
	addr := requireTestRedis(t)
	client := redis.NewClient(&redis.Options{Addr: addr})
	defer client.Close()

	l := NewRedisLimiter(client)

	allowed, err := l.Allow(context.Background(), "unlimited-func", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Error("expected unlimited limit to allow request")
	}
}

func TestRedisLimiter_AllowsUpToCapacity(t *testing.T) {
	addr := requireTestRedis(t)
	client := redis.NewClient(&redis.Options{Addr: addr})
	defer client.Close()

	l := NewRedisLimiter(client)
	ctx := context.Background()
	funcName := "capacity-func"

	// Limit = 2 (capacity 2)
	for i := range 2 {
		allowed, err := l.Allow(ctx, funcName, 2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !allowed {
			t.Errorf("expected request %d to be allowed", i)
		}
	}

	// Should be denied
	allowed, _ := l.Allow(ctx, funcName, 2)
	if allowed {
		t.Error("expected 3rd request to be denied (capacity exceeded)")
	}
}

func TestRedisLimiter_TokenRefill(t *testing.T) {
	addr := requireTestRedis(t)
	client := redis.NewClient(&redis.Options{Addr: addr})
	defer client.Close()

	l := NewRedisLimiter(client)
	ctx := context.Background()
	funcName := "refill-func"

	// Set capacity=1
	allowed, _ := l.Allow(ctx, funcName, 1)
	if !allowed {
		t.Fatal("expected first request to be allowed")
	}

	// Should be denied
	allowed, _ = l.Allow(ctx, funcName, 1)
	if allowed {
		t.Fatal("expected immediate second request to be denied")
	}

	// Wait for refill
	time.Sleep(1100 * time.Millisecond)

	// Should be allowed
	allowed, _ = l.Allow(ctx, funcName, 1)
	if !allowed {
		t.Error("expected request to be allowed after token refill")
	}
}

func TestRedisLimiter_MultipleFunctions(t *testing.T) {
	addr := requireTestRedis(t)
	client := redis.NewClient(&redis.Options{Addr: addr})
	defer client.Close()

	l := NewRedisLimiter(client)
	ctx := context.Background()

	// Exhaust func-a
	l.Allow(ctx, "func-a", 1)
	allowed, _ := l.Allow(ctx, "func-a", 1)
	if allowed {
		t.Error("expected func-a to be denied")
	}

	// func-b should be allowed
	allowed, _ = l.Allow(ctx, "func-b", 1)
	if !allowed {
		t.Error("expected func-b to be allowed independently")
	}
}

func TestRedisLimiter_FailOpen(t *testing.T) {
	// Connect to dead port
	client := redis.NewClient(&redis.Options{
		Addr:        "127.0.0.1:9999",
		DialTimeout: 100 * time.Millisecond,
		ReadTimeout: 100 * time.Millisecond,
	})
	defer client.Close()

	l := NewRedisLimiter(client)

	// Should return true, nil when Redis is down
	allowed, err := l.Allow(context.Background(), "fail-open-func", 10)

	if err != nil {
		t.Errorf("expected no error (fail open), got: %v", err)
	}
	if !allowed {
		t.Error("expected request to be allowed when Redis fails")
	}
}
