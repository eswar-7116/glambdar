package pool

import (
	"context"
	"sync"
	"testing"
)

func TestPoolManager_GetOrCreate(t *testing.T) {
	pm := &PoolManager{}

	p1, _ := pm.GetOrCreate("func-1", 1)
	if p1 == nil {
		t.Fatalf("Expected pool for func-1, got nil")
	}

	// Should return the same pool
	p2, _ := pm.GetOrCreate("func-1", 1)
	if p1 != p2 {
		t.Errorf("Expected same pool instance for same funcName, got different")
	}

	// Should return a different pool
	p3, _ := pm.GetOrCreate("func-2", 1)
	if p1 == p3 {
		t.Errorf("Expected different pool instance, got same")
	}

	// Test Delete
	pm.DeletePool(context.Background(), nil, "func-1")

	// Should create a new pool
	p4, _ := pm.GetOrCreate("func-1", 1)
	if p1 == p4 {
		t.Errorf("Expected new pool instance after delete, got same")
	}
}

func TestPoolManager_GetPoolStatuses_Empty(t *testing.T) {
	pm := &PoolManager{}
	statuses := pm.GetPoolStatuses()
	if len(statuses) != 0 {
		t.Errorf("Expected empty statuses map, got len %d", len(statuses))
	}
}

func TestPoolManager_GetPoolStatuses_WithPools(t *testing.T) {
	pm := &PoolManager{}
	pm.GetOrCreate("func-a", 5)
	pm.GetOrCreate("func-b", 10)

	statuses := pm.GetPoolStatuses()
	if len(statuses) != 2 {
		t.Fatalf("Expected 2 statuses, got %d", len(statuses))
	}

	if st, ok := statuses["func-a"]; !ok || st.MaxConcurrency != 5 {
		t.Errorf("Expected func-a with MaxConcurrency 5, got %+v", st)
	}
	if st, ok := statuses["func-b"]; !ok || st.MaxConcurrency != 10 {
		t.Errorf("Expected func-b with MaxConcurrency 10, got %+v", st)
	}
}

func TestPoolManager_GetOrCreate_Concurrent(t *testing.T) {
	pm := &PoolManager{}
	funcName := "concurrent-func"

	var wg sync.WaitGroup
	pools := make([]*ContainerPool, 100)

	for i := range 100 {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			p, _ := pm.GetOrCreate(funcName, 10)
			pools[idx] = p
		}(i)
	}
	wg.Wait()

	// All pools should be the same
	first := pools[0]
	for i := 1; i < 100; i++ {
		if pools[i] != first {
			t.Errorf("Expected same pool instance for concurrent GetOrCreate, got different at index %d", i)
		}
	}
}

func TestPoolManager_DeletePool_Nonexistent(t *testing.T) {
	pm := &PoolManager{}

	// Should not panic
	pm.DeletePool(context.Background(), nil, "nonexistent")
}
