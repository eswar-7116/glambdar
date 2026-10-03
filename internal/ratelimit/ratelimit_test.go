package ratelimit

import (
	"context"
	"testing"
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
