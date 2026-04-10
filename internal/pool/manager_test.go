package pool

import (
	"testing"
)

func TestPoolManager_GetOrCreate(t *testing.T) {
	pm := &PoolManager{}

	p1 := pm.GetOrCreate("func-1")
	if p1 == nil {
		t.Fatalf("Expected pool for func-1, got nil")
	}

	// Should return the same pool instance
	p2 := pm.GetOrCreate("func-1")
	if p1 != p2 {
		t.Errorf("Expected same pool instance for same funcName, got different")
	}

	// Should return a different pool instance for a different func
	p3 := pm.GetOrCreate("func-2")
	if p1 == p3 {
		t.Errorf("Expected different pool instance, got same")
	}

	// Test Delete
	pm.Delete("func-1")

	// Should create a new one now since the old was deleted
	p4 := pm.GetOrCreate("func-1")
	if p1 == p4 {
		t.Errorf("Expected new pool instance after delete, got same")
	}
}
