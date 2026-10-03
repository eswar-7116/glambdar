package ratelimit

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestLocalLimiter_Unlimited(t *testing.T) {
	l := NewLocalLimiter()
	allowed, err := l.Allow(context.Background(), "test-func", 0)
	if err != nil {
		t.Fatal(err)
	}
	if !allowed {
		t.Error("expected unlimited to allow")
	}
}

func TestLocalLimiter_Enforced(t *testing.T) {
	l := NewLocalLimiter()
	funcName := "limited-func"
	limit := 5

	// Only 1 request allowed
	allowed, err := l.Allow(context.Background(), funcName, limit)
	if err != nil {
		t.Fatal(err)
	}
	if !allowed {
		t.Error("first request should be allowed")
	}

	// Second request should be rate-limited
	allowed, err = l.Allow(context.Background(), funcName, limit)
	if err != nil {
		t.Fatal(err)
	}
	if allowed {
		t.Error("second immediate request should be rate-limited with burst=1")
	}
}

func TestLocalLimiter_NegativeLimit(t *testing.T) {
	l := NewLocalLimiter()
	for i := range 10 {
		allowed, err := l.Allow(context.Background(), "neg-func", -5)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !allowed {
			t.Errorf("negative limit should be treated as unlimited, request %d was denied", i)
		}
	}
}

func TestLocalLimiter_MultipleFunctions(t *testing.T) {
	l := NewLocalLimiter()
	ctx := context.Background()

	// Exhaust burst
	l.Allow(ctx, "func-a", 5)
	allowed, _ := l.Allow(ctx, "func-a", 5)
	if allowed {
		t.Error("func-a should be rate-limited after burst exhausted")
	}

	// func-b should still be allowed
	allowed, _ = l.Allow(ctx, "func-b", 5)
	if !allowed {
		t.Error("func-b should be allowed independently of func-a")
	}
}

func TestLocalLimiter_UpdateLimit(t *testing.T) {
	l := NewLocalLimiter()
	ctx := context.Background()
	funcName := "updatable"

	// Set limit=5
	l.Allow(ctx, funcName, 5)

	// Increase limit
	l.UpdateLimit(funcName, 100)

	// Wait for tokens to refill
	time.Sleep(100 * time.Millisecond)

	allowedCount := 0
	for range 15 {
		allowed, _ := l.Allow(ctx, funcName, 100)
		if allowed {
			allowedCount++
		}
	}

	if allowedCount < 5 {
		t.Errorf("expected several requests allowed after raising limit, got %d", allowedCount)
	}
}

func TestLocalLimiter_UpdateLimit_ToUnlimited(t *testing.T) {
	l := NewLocalLimiter()
	ctx := context.Background()
	funcName := "make-unlimited"

	// Set limit=1
	l.Allow(ctx, funcName, 1)

	// Update to unlimited
	l.UpdateLimit(funcName, 0)

	for i := range 100 {
		allowed, err := l.Allow(ctx, funcName, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !allowed {
			t.Errorf("expected unlimited after UpdateLimit(0), request %d denied", i)
		}
	}
}

func TestLocalLimiter_UpdateLimit_NonexistentFunc(t *testing.T) {
	l := NewLocalLimiter()

	l.UpdateLimit("never-seen", 100)

	allowed, err := l.Allow(context.Background(), "never-seen", 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Error("first request should be allowed")
	}
}

func TestLocalLimiter_ConcurrentAccess(t *testing.T) {
	l := NewLocalLimiter()
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := range 100 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			funcName := "concurrent-func"
			if n%2 == 0 {
				funcName = "concurrent-func-2"
			}
			l.Allow(ctx, funcName, 1000)
		}(i)
	}
	wg.Wait()
}

func TestLocalLimiter_HighLimit(t *testing.T) {
	l := NewLocalLimiter()
	ctx := context.Background()

	allowedCount := 0
	for range 1000 {
		allowed, err := l.Allow(ctx, "high-limit", 10000)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if allowed {
			allowedCount++
		}
	}

	if allowedCount != 1000 {
		t.Errorf("expected all 1000 requests allowed with high limit, got %d", allowedCount)
	}
}
