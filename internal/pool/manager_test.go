package pool

import (
	"context"
	"testing"
	"golang.org/x/time/rate"
)

func TestPoolManager_GetOrCreate(t *testing.T) {
	pm := &PoolManager{}

	p1 := pm.GetOrCreate("func-1", 0, 1)
	if p1 == nil {
		t.Fatalf("Expected pool for func-1, got nil")
	}

	// Should return the same pool instance
	p2 := pm.GetOrCreate("func-1", 0, 1)
	if p1 != p2 {
		t.Errorf("Expected same pool instance for same funcName, got different")
	}

	// Should return a different pool instance for a different func
	p3 := pm.GetOrCreate("func-2", 0, 1)
	if p1 == p3 {
		t.Errorf("Expected different pool instance, got same")
	}

	// Test Delete
	pm.DeletePool(context.Background(), nil, "func-1")

	// Should create a new one now since the old was deleted
	p4 := pm.GetOrCreate("func-1", 0, 1)
	if p1 == p4 {
		t.Errorf("Expected new pool instance after delete, got same")
	}
}

func TestPoolManager_UpdateLimiter(t *testing.T) {
	pm := &PoolManager{}
	funcName := "update-limit-func"

	// Initial create with limit 10
	p := pm.GetOrCreate(funcName, 10, 1)
	if p.Limiter.Limit() != 10 {
		t.Errorf("expected limit 10, got %v", p.Limiter.Limit())
	}

	// Update to limit 20
	pm.UpdateLimiter(funcName, 20)
	if p.Limiter.Limit() != 20 {
		t.Errorf("expected limit 20, got %v", p.Limiter.Limit())
	}
	if p.Limiter.Burst() != 2 {
		t.Errorf("expected burst 20, got %d", p.Limiter.Burst())
	}
}

func TestParseRateLimit(t *testing.T) {
	tests := []struct {
		input     int
		wantLimit rate.Limit
		wantBurst int
	}{
		{0, rate.Inf, 1000000000},
		{-5, rate.Inf, 1000000000},
		{5, 5, 1},
		{20, 20, 2},
		{100, 100, 10},
	}

	for _, tt := range tests {
		limit, burst := parseRateLimit(tt.input)
		if limit != tt.wantLimit {
			t.Errorf("parseRateLimit(%d) limit = %v, want %v", tt.input, limit, tt.wantLimit)
		}
		if burst != tt.wantBurst {
			t.Errorf("parseRateLimit(%d) burst = %d, want %d", tt.input, burst, tt.wantBurst)
		}
	}
}
